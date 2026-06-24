package counter

import (
	"github.com/whatap/go-api/agent/agent/counter/meter"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/value"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/hmap"
)

const millisPerFifteenMin int64 = 15 * dateutil.MILLIS_PER_MINUTE

// TaskLLM — 단일 LLM Task. 5초 process 안에서 6 카테고리 발행 + 15분 meter 분기.
// 옛 7 Task (Active/ApiStatus/Error/Feature/Perf/TokenUsage + 15분 meter) 통합.
type TaskLLM struct {
	// 15분 meter 누적 (Count + Error 둘 다 — D6 결정)
	currentCount     int64
	currentError     int64
	currentHourStart int64
	next15minTime    int64
}

func NewTaskLLM() *TaskLLM {
	return &TaskLLM{}
}

func (this *TaskLLM) process(p *pack.CounterPack1) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA140201", " TaskLLM Recover ", r)
		}
	}()

	m := meter.GetInstanceMeterLLM()
	b := m.GetBucketReset()
	active := m.ActiveSnapshot()

	// 5초 — 6 카테고리 발행
	this.sendActiveStats(p, active)
	this.sendApiStatusStats(p, b)
	this.sendErrorStats(p, b)
	this.sendFeatureStats(p, b)
	this.sendPerfStats(p, b)
	this.sendTokenUsageStats(p, b)

	// 15분 — meter (apm-go-agent TaskLLM 동등 + Count/Error 둘 다 누적)
	now := p.Time
	this.currentCount += b.MeterCount
	this.currentError += b.MeterError
	if this.currentHourStart == 0 {
		this.currentHourStart = now / dateutil.MILLIS_PER_HOUR * dateutil.MILLIS_PER_HOUR
	}
	if this.next15minTime == 0 {
		this.next15minTime = now/millisPerFifteenMin*millisPerFifteenMin + millisPerFifteenMin
	}
	if now >= this.next15minTime {
		this.sendMeter(p, this.currentCount, this.currentError, this.next15minTime)
		this.next15minTime = now/millisPerFifteenMin*millisPerFifteenMin + millisPerFifteenMin
	}
	hourStart := now / dateutil.MILLIS_PER_HOUR * dateutil.MILLIS_PER_HOUR
	if hourStart > this.currentHourStart {
		this.currentCount = 0
		this.currentError = 0
		this.currentHourStart = hourStart
	}
}

// ── 5 공통 태그 helper ──

func putBaseAndCommonTags(out *pack.TagCountPack, p *pack.CounterPack1, category string, key meter.LLMKey) {
	out.Pcode = p.Pcode
	out.Oid = p.Oid
	out.Okind = p.Okind
	out.Onode = p.Onode
	out.Time = p.Time
	out.Category = category
	out.Tags.PutLong("pid", llmPid)
	out.PutTag("provider", key.Provider)
	out.PutTag("model", key.Model)
	out.PutTag("operation_type", key.OperationType)
	out.PutTag("url", key.URL)
}

// ── Active (5초, llm_active_stat) ──

func (this *TaskLLM) sendActiveStats(p *pack.CounterPack1, active []*meter.LLMActiveEntry) {
	for _, e := range active {
		if e.Count == 0 {
			continue
		}
		out := pack.NewTagCountPack()
		putBaseAndCommonTags(out, p, "llm_active_stat", e.LLMKey)
		out.Put("count", e.Count)
		data.SendTagCount(out, true)
	}
}

// ── ApiStatus (5초, llm_api_status) ──

func (this *TaskLLM) sendApiStatusStats(p *pack.CounterPack1, b *meter.LLMBucket) {
	en := b.ApiStatus.Entries()
	for en.HasMoreElements() {
		ent := en.NextElement().(*hmap.LinkedEntry)
		e := ent.GetValue().(*meter.LLMApiStatusEntry)
		if e.FourXX == 0 && e.FiveXX == 0 {
			continue
		}
		out := pack.NewTagCountPack()
		putBaseAndCommonTags(out, p, "llm_api_status", e.LLMKey)
		out.Put("4xx_total_count", e.FourXX)
		out.Put("5xx_total_count", e.FiveXX)
		data.SendTagCount(out, true)
	}
}

// ── Error (5초, llm_error_stat) ──

func (this *TaskLLM) sendErrorStats(p *pack.CounterPack1, b *meter.LLMBucket) {
	en := b.Error.Entries()
	for en.HasMoreElements() {
		ent := en.NextElement().(*hmap.LinkedEntry)
		e := ent.GetValue().(*meter.LLMErrorEntry)
		total := e.ApiError + e.ProgramError
		if total == 0 && e.LastApiError == 0 {
			continue
		}
		out := pack.NewTagCountPack()
		putBaseAndCommonTags(out, p, "llm_error_stat", e.LLMKey)
		out.Put("api_error_count", e.ApiError)
		out.Put("program_error_count", e.ProgramError)
		out.Put("error_count", total)
		out.Put("last_api_error_count", e.LastApiError)
		data.SendTagCount(out, true)
	}
}

// ── Feature (5초, llm_feature_stat, !rectype=2) ──

func (this *TaskLLM) sendFeatureStats(p *pack.CounterPack1, b *meter.LLMBucket) {
	en := b.Feature.Entries()
	for en.HasMoreElements() {
		ent := en.NextElement().(*hmap.LinkedEntry)
		e := ent.GetValue().(*meter.LLMFeatureEntry)
		if e.CallCount == 0 {
			continue
		}
		out := pack.NewTagCountPack()
		putBaseAndCommonTags(out, p, "llm_feature_stat", e.LLMKey)
		out.Tags.PutLong("!rectype", 2)
		out.Put("call_count", e.CallCount)
		out.Put("error_count", e.ErrorCount)

		features := value.NewListValue(nil)
		featuresCount := value.NewListValue(nil)
		for name, cnt := range e.Features {
			features.AddString(name)
			featuresCount.AddLong(cnt)
		}
		out.Put("features", features)
		out.Put("features_count", featuresCount)

		data.SendTagCount(out, true)
	}
}

// ── Perf (5초, llm_perf_stat — sum/count fallback) ──

func (this *TaskLLM) sendPerfStats(p *pack.CounterPack1, b *meter.LLMBucket) {
	en := b.Perf.Entries()
	for en.HasMoreElements() {
		ent := en.NextElement().(*hmap.LinkedEntry)
		e := ent.GetValue().(*meter.LLMPerfEntry)
		if e.CallCount == 0 {
			continue
		}
		out := pack.NewTagCountPack()
		putBaseAndCommonTags(out, p, "llm_perf_stat", e.LLMKey)
		out.Put("call_count", e.CallCount)
		out.Put("error_count", e.ErrorCount)
		out.Put("latency_sum", e.LatencySum)
		out.Put("ttft_sum", e.TtftSum)
		out.Put("ttft_count", e.TtftCount)
		out.Put("tpot_sum", e.TpotSum)
		out.Put("tpot_count", e.TpotCount)
		// sketch fields omitted (option C — sum/count fallback, §263 검토)
		data.SendTagCount(out, true)
	}
}

// ── TokenUsage (5초, llm_token_usage, !rectype=2) ──

func (this *TaskLLM) sendTokenUsageStats(p *pack.CounterPack1, b *meter.LLMBucket) {
	en := b.Token.Entries()
	for en.HasMoreElements() {
		ent := en.NextElement().(*hmap.LinkedEntry)
		e := ent.GetValue().(*meter.LLMTokenEntry)
		if e.CallCount == 0 {
			continue
		}
		out := pack.NewTagCountPack()
		putBaseAndCommonTags(out, p, "llm_token_usage", e.LLMKey)
		out.Tags.PutLong("!rectype", 2)
		out.Put("call_count", e.CallCount)
		out.Put("error_count", e.ErrorCount)
		out.Put("stream_count", e.StreamCount)

		tokens := value.NewListValue(nil)
		tokensCost := value.NewListValue(nil)
		tokensCount := value.NewListValue(nil)

		costMap := e.TokenCosts
		for name, cnt := range e.TokenCounts {
			tokens.AddString(name)
			tokensCount.AddLong(cnt)
			cost := 0.0
			if costMap != nil {
				cost = costMap[name]
			}
			tokensCost.Add(value.NewDoubleValue(cost))
		}
		out.Put("tokens", tokens)
		out.Put("tokens_cost", tokensCost)
		out.Put("tokens_count", tokensCount)
		out.Put("total_tokens", e.TotalTokens)
		out.Put("total_cost", e.TotalCost)

		data.SendTagCount(out, true)
	}
}

// ── 15분 meter (LLM_TRANSACTION) ──

func (this *TaskLLM) sendMeter(base *pack.CounterPack1, units, errorCount, sendTime int64) {
	p := pack.NewTagCountPack()
	p.Pcode = base.Pcode
	p.Oid = base.Oid
	p.Okind = base.Okind
	p.Onode = base.Onode
	p.Time = sendTime
	p.Category = "meter"
	p.PutTag("mtype", "LLM_TRANSACTION")
	p.PutTag("munit", "LLM_TRANSACTION")
	p.PutTag("mperiod", "900")
	p.PutTag("msize", "3600")
	p.Put("units", units)             // 사양 명시 — 전체 호출 수
	p.Put("error_count", errorCount)  // D6 — 사양 확장. 실패 호출 수
	data.SendTagCount(p, true)
}
