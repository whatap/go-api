package meter

import (
	"sync"

	"github.com/whatap/golib/util/hmap"
)

const llmKeyMax = 1024

// LLMKey — LLM 메트릭 4 공통 키 (provider/model/operation_type/url).
// hmap.LinkedKey 인터페이스 구현 (Hash + Equals) — value receiver 로 직접 키 사용.
type LLMKey struct {
	Provider      string
	Model         string
	OperationType string
	URL           string
}

func (k LLMKey) String() string {
	return k.Provider + "|" + k.Model + "|" + k.OperationType + "|" + k.URL
}

// Hash — Java hashCode 동등 패턴 (31*h + c). 외부 의존 X.
func (k LLMKey) Hash() uint {
	var h uint
	for _, s := range [4]string{k.Provider, k.Model, k.OperationType, k.URL} {
		for i := 0; i < len(s); i++ {
			h = h*31 + uint(s[i])
		}
		h = h*31 + 0x7C // '|' separator
	}
	return h
}

func (k LLMKey) Equals(o hmap.LinkedKey) bool {
	if other, ok := o.(LLMKey); ok {
		return k == other
	}
	return false
}

// 6 stat Entry — 각 LLMKey embed.

type LLMApiStatusEntry struct {
	LLMKey
	FourXX, FiveXX int64
}

type LLMErrorEntry struct {
	LLMKey
	ApiError, ProgramError, LastApiError int64
}

type LLMFeatureEntry struct {
	LLMKey
	CallCount, ErrorCount int64
	Features              map[string]int64
}

type LLMPerfEntry struct {
	LLMKey
	CallCount, ErrorCount, TtftCount, TpotCount int64
	LatencySum, TtftSum, TpotSum                float64
}

type LLMTokenEntry struct {
	LLMKey
	CallCount, ErrorCount, StreamCount, TotalTokens int64
	TokenCounts map[string]int64
	TokenCosts  map[string]float64
	TotalCost   float64
}

type LLMActiveEntry struct {
	LLMKey
	Count int64
}

// LLMBucket — 5 reset-able stat 의 LinkedMap snapshot + 15분 meter (Count + Error)
type LLMBucket struct {
	ApiStatus  *hmap.LinkedMap
	Error      *hmap.LinkedMap
	Feature    *hmap.LinkedMap
	Perf       *hmap.LinkedMap
	Token      *hmap.LinkedMap
	MeterCount int64
	MeterError int64
}

// MeterLLM — 단일 LLM Meter (7+7 Meter/Task 통합)
type MeterLLM struct {
	apiStatusStat *hmap.LinkedMap
	errorStat     *hmap.LinkedMap
	featureStat   *hmap.LinkedMap
	perfStat      *hmap.LinkedMap
	tokenStat     *hmap.LinkedMap

	activeStat *hmap.LinkedMap // reset 안 함 (현재값)

	meterCount int64
	meterError int64

	lock sync.Mutex
}

var meterLLM *MeterLLM

func newMeterLLM() *MeterLLM {
	p := new(MeterLLM)
	p.apiStatusStat = hmap.NewLinkedMapDefault().SetMax(llmKeyMax)
	p.errorStat = hmap.NewLinkedMapDefault().SetMax(llmKeyMax)
	p.featureStat = hmap.NewLinkedMapDefault().SetMax(llmKeyMax)
	p.perfStat = hmap.NewLinkedMapDefault().SetMax(llmKeyMax)
	p.tokenStat = hmap.NewLinkedMapDefault().SetMax(llmKeyMax)
	p.activeStat = hmap.NewLinkedMapDefault().SetMax(llmKeyMax)
	return p
}

func GetInstanceMeterLLM() *MeterLLM {
	if meterLLM != nil {
		return meterLLM
	}
	meterLLM = newMeterLLM()
	return meterLLM
}

// ── Active (reset X) ──

func (this *MeterLLM) OnStart(provider, model, opType, url string) {
	key := LLMKey{provider, model, opType, url}
	this.lock.Lock()
	defer this.lock.Unlock()
	if v := this.activeStat.Get(key); v != nil {
		v.(*LLMActiveEntry).Count++
		return
	}
	this.activeStat.Put(key, &LLMActiveEntry{LLMKey: key, Count: 1})
}

func (this *MeterLLM) OnEnd(provider, model, opType, url string) {
	key := LLMKey{provider, model, opType, url}
	this.lock.Lock()
	defer this.lock.Unlock()
	v := this.activeStat.Get(key)
	if v == nil {
		return
	}
	e := v.(*LLMActiveEntry)
	e.Count--
	if e.Count <= 0 {
		this.activeStat.Remove(key)
	}
}

// ── ApiStatus (5초, reset O) ──

func (this *MeterLLM) AddApiStatus(provider, model, opType, url string, statusCode int) {
	if statusCode < 400 || statusCode >= 600 {
		return
	}
	key := LLMKey{provider, model, opType, url}
	this.lock.Lock()
	defer this.lock.Unlock()
	var e *LLMApiStatusEntry
	if v := this.apiStatusStat.Get(key); v != nil {
		e = v.(*LLMApiStatusEntry)
	} else {
		e = &LLMApiStatusEntry{LLMKey: key}
		this.apiStatusStat.Put(key, e)
	}
	if statusCode < 500 {
		e.FourXX++
	} else {
		e.FiveXX++
	}
}

// ── Error (5초, reset O) ──

func (this *MeterLLM) AddApiError(provider, model, opType, url string) {
	this.errorEntry(provider, model, opType, url).ApiError++
}

func (this *MeterLLM) AddProgramError(provider, model, opType, url string) {
	this.errorEntry(provider, model, opType, url).ProgramError++
}

func (this *MeterLLM) SetLastApiError(provider, model, opType, url string, count int64) {
	this.errorEntry(provider, model, opType, url).LastApiError = count
}

func (this *MeterLLM) errorEntry(provider, model, opType, url string) *LLMErrorEntry {
	key := LLMKey{provider, model, opType, url}
	this.lock.Lock()
	defer this.lock.Unlock()
	if v := this.errorStat.Get(key); v != nil {
		return v.(*LLMErrorEntry)
	}
	e := &LLMErrorEntry{LLMKey: key}
	this.errorStat.Put(key, e)
	return e
}

// ── Feature (5초, reset O, !rectype=2) ──

func (this *MeterLLM) AddFeature(provider, model, opType, url string, featureNames []string, isError bool) {
	key := LLMKey{provider, model, opType, url}
	this.lock.Lock()
	defer this.lock.Unlock()
	var e *LLMFeatureEntry
	if v := this.featureStat.Get(key); v != nil {
		e = v.(*LLMFeatureEntry)
	} else {
		e = &LLMFeatureEntry{LLMKey: key}
		this.featureStat.Put(key, e)
	}
	e.CallCount++
	if isError {
		e.ErrorCount++
	}
	if len(featureNames) == 0 {
		return
	}
	if e.Features == nil {
		e.Features = map[string]int64{}
	}
	for _, name := range featureNames {
		e.Features[name]++
	}
}

// ── Perf (5초, reset O — sum/count fallback, sketch 옵션 C) ──

// AddPerf — 단일 perf entry 누적 (MeterSQL.Add / MeterHTTPC.Add 와 동일한 통합 add 패턴).
// latencyMs 는 항상 누적. ttftMs 가 있으면 TtftSum/TtftCount 누적 + 스트리밍 응답이면
// TPOT 자동 계산 (`(latencyMs - ttftMs) / (outputTokens - 1)`) 후 TpotSum/TpotCount 누적.
// non-streaming 또는 outputTokens ≤ 1 이면 TPOT 무의미하므로 skip.
func (this *MeterLLM) AddPerf(provider, model, opType, url string,
	latencyMs float64, ttftMs *float64, outputTokens *int64,
	streaming, isError bool) {
	e := this.perfEntry(provider, model, opType, url)
	e.CallCount++
	if isError {
		e.ErrorCount++
	}
	e.LatencySum += latencyMs
	if ttftMs == nil {
		return
	}
	e.TtftSum += *ttftMs
	e.TtftCount++
	if streaming && outputTokens != nil && *outputTokens > 1 {
		tpot := (latencyMs - *ttftMs) / float64(*outputTokens-1)
		if tpot > 0 {
			e.TpotSum += tpot
			e.TpotCount++
		}
	}
}

func (this *MeterLLM) perfEntry(provider, model, opType, url string) *LLMPerfEntry {
	key := LLMKey{provider, model, opType, url}
	this.lock.Lock()
	defer this.lock.Unlock()
	if v := this.perfStat.Get(key); v != nil {
		return v.(*LLMPerfEntry)
	}
	e := &LLMPerfEntry{LLMKey: key}
	this.perfStat.Put(key, e)
	return e
}

// ── TokenUsage (5초, reset O, !rectype=2) ──

func (this *MeterLLM) AddTokenUsage(provider, model, opType, url string,
	tokens map[string]int64, costs map[string]float64,
	isError, isStream bool) {

	key := LLMKey{provider, model, opType, url}
	this.lock.Lock()
	defer this.lock.Unlock()
	var e *LLMTokenEntry
	if v := this.tokenStat.Get(key); v != nil {
		e = v.(*LLMTokenEntry)
	} else {
		e = &LLMTokenEntry{LLMKey: key}
		this.tokenStat.Put(key, e)
	}
	e.CallCount++
	if isError {
		e.ErrorCount++
	}
	if isStream {
		e.StreamCount++
	}
	if len(tokens) > 0 {
		if e.TokenCounts == nil {
			e.TokenCounts = map[string]int64{}
		}
		for name, n := range tokens {
			e.TokenCounts[name] += n
			e.TotalTokens += n
		}
	}
	if len(costs) > 0 {
		if e.TokenCosts == nil {
			e.TokenCosts = map[string]float64{}
		}
		for name, c := range costs {
			e.TokenCosts[name] += c
			e.TotalCost += c
		}
	}
}

// ── 15분 meter (Count + Error 둘 다 — D6 결정) ──

func (this *MeterLLM) IncrementMeter() {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.meterCount++
}

func (this *MeterLLM) IncrementMeterError() {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.meterError++
}

// ── 회수 ──

// GetBucketReset — 5 reset-able stat + 15분 meter 회수 + 새 LinkedMap 으로 교체
func (this *MeterLLM) GetBucketReset() *LLMBucket {
	this.lock.Lock()
	defer this.lock.Unlock()
	out := &LLMBucket{
		ApiStatus:  this.apiStatusStat,
		Error:      this.errorStat,
		Feature:    this.featureStat,
		Perf:       this.perfStat,
		Token:      this.tokenStat,
		MeterCount: this.meterCount,
		MeterError: this.meterError,
	}
	this.apiStatusStat = hmap.NewLinkedMapDefault().SetMax(llmKeyMax)
	this.errorStat = hmap.NewLinkedMapDefault().SetMax(llmKeyMax)
	this.featureStat = hmap.NewLinkedMapDefault().SetMax(llmKeyMax)
	this.perfStat = hmap.NewLinkedMapDefault().SetMax(llmKeyMax)
	this.tokenStat = hmap.NewLinkedMapDefault().SetMax(llmKeyMax)
	this.meterCount = 0
	this.meterError = 0
	return out
}

// ActiveSnapshot — 현재값 카피 (reset 안 함)
func (this *MeterLLM) ActiveSnapshot() []*LLMActiveEntry {
	this.lock.Lock()
	defer this.lock.Unlock()
	out := make([]*LLMActiveEntry, 0, this.activeStat.Size())
	en := this.activeStat.Entries()
	for en.HasMoreElements() {
		ent := en.NextElement().(*hmap.LinkedEntry)
		e := ent.GetValue().(*LLMActiveEntry)
		out = append(out, &LLMActiveEntry{LLMKey: e.LLMKey, Count: e.Count})
	}
	return out
}

func (this *MeterLLM) Clear() {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.apiStatusStat = hmap.NewLinkedMapDefault().SetMax(llmKeyMax)
	this.errorStat = hmap.NewLinkedMapDefault().SetMax(llmKeyMax)
	this.featureStat = hmap.NewLinkedMapDefault().SetMax(llmKeyMax)
	this.perfStat = hmap.NewLinkedMapDefault().SetMax(llmKeyMax)
	this.tokenStat = hmap.NewLinkedMapDefault().SetMax(llmKeyMax)
	this.activeStat = hmap.NewLinkedMapDefault().SetMax(llmKeyMax)
	this.meterCount = 0
	this.meterError = 0
}
