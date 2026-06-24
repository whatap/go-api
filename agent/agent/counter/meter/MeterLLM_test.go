package meter

import (
	"math"
	"testing"

	"github.com/whatap/golib/util/hmap"
)

const (
	openai    = "openai"
	gpt4o     = "gpt-4o"
	chat      = "chat"
	urlChat   = "https://api.openai.com/v1/chat/completions"
	anthropic = "anthropic"
	claude    = "claude-opus-4-6"
	messages  = "messages"
	urlAnth   = "https://api.anthropic.com/v1/messages"
)

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func keyOpenAI() LLMKey { return LLMKey{openai, gpt4o, chat, urlChat} }

func TestLLMKey_HashEquals(t *testing.T) {
	k1 := LLMKey{openai, gpt4o, chat, urlChat}
	k2 := LLMKey{openai, gpt4o, chat, urlChat}
	k3 := LLMKey{openai, gpt4o, chat, urlAnth}

	if !k1.Equals(k2) {
		t.Fatal("Equals: same fields should be true")
	}
	if k1.Equals(k3) {
		t.Fatal("Equals: different URL should be false")
	}
	if k1.Hash() != k2.Hash() {
		t.Fatal("Hash: same fields should give same hash")
	}
	if k1.Equals(LLMKey{}) {
		t.Fatal("Equals: empty LLMKey should not match")
	}
}

func TestMeterLLM_OnStartOnEnd(t *testing.T) {
	m := newMeterLLM()
	m.OnStart(openai, gpt4o, chat, urlChat)
	m.OnStart(openai, gpt4o, chat, urlChat)
	m.OnStart(anthropic, claude, messages, urlAnth)

	snap := m.ActiveSnapshot()
	if len(snap) != 2 {
		t.Fatalf("ActiveSnapshot: want 2 keys, got %d", len(snap))
	}
	for _, e := range snap {
		if e.Provider == openai && e.Count != 2 {
			t.Fatalf("openai count: want 2, got %d", e.Count)
		}
		if e.Provider == anthropic && e.Count != 1 {
			t.Fatalf("anthropic count: want 1, got %d", e.Count)
		}
	}

	m.OnEnd(openai, gpt4o, chat, urlChat)
	snap2 := m.ActiveSnapshot()
	for _, e := range snap2 {
		if e.Provider == openai && e.Count != 1 {
			t.Fatalf("openai after one end: want 1, got %d", e.Count)
		}
	}

	m.OnEnd(openai, gpt4o, chat, urlChat)
	snap3 := m.ActiveSnapshot()
	for _, e := range snap3 {
		if e.Provider == openai {
			t.Fatalf("openai should be removed after count drops to 0")
		}
	}
}

func TestMeterLLM_ActiveSnapshotIndependence(t *testing.T) {
	m := newMeterLLM()
	m.OnStart(openai, gpt4o, chat, urlChat)
	snap := m.ActiveSnapshot()
	m.OnStart(openai, gpt4o, chat, urlChat)

	for _, e := range snap {
		if e.Count != 1 {
			t.Fatalf("snapshot should be independent copy: want 1, got %d", e.Count)
		}
	}
}

func TestMeterLLM_AddApiStatus(t *testing.T) {
	m := newMeterLLM()
	m.AddApiStatus(openai, gpt4o, chat, urlChat, 400)
	m.AddApiStatus(openai, gpt4o, chat, urlChat, 429)
	m.AddApiStatus(openai, gpt4o, chat, urlChat, 500)
	m.AddApiStatus(openai, gpt4o, chat, urlChat, 502)
	m.AddApiStatus(openai, gpt4o, chat, urlChat, 200) // ignored
	m.AddApiStatus(openai, gpt4o, chat, urlChat, 600) // out of range

	b := m.GetBucketReset()
	e := getApiStatus(t, b, keyOpenAI())
	if e.FourXX != 2 {
		t.Fatalf("4xx: want 2, got %d", e.FourXX)
	}
	if e.FiveXX != 2 {
		t.Fatalf("5xx: want 2, got %d", e.FiveXX)
	}

	// 다음 5초 cycle — bucket 비어 있어야
	b2 := m.GetBucketReset()
	if b2.ApiStatus.Size() != 0 {
		t.Fatalf("ApiStatus should be empty after reset")
	}
}

func TestMeterLLM_AddError(t *testing.T) {
	m := newMeterLLM()
	m.AddApiError(openai, gpt4o, chat, urlChat)
	m.AddApiError(openai, gpt4o, chat, urlChat)
	m.AddProgramError(openai, gpt4o, chat, urlChat)
	m.SetLastApiError(openai, gpt4o, chat, urlChat, 5)

	b := m.GetBucketReset()
	e := getError(t, b, keyOpenAI())
	if e.ApiError != 2 {
		t.Fatalf("ApiError: want 2, got %d", e.ApiError)
	}
	if e.ProgramError != 1 {
		t.Fatalf("ProgramError: want 1, got %d", e.ProgramError)
	}
	if e.LastApiError != 5 {
		t.Fatalf("LastApiError: want 5, got %d", e.LastApiError)
	}
}

func TestMeterLLM_AddFeature(t *testing.T) {
	m := newMeterLLM()
	m.AddFeature(openai, gpt4o, chat, urlChat, []string{"vision", "tool_use"}, false)
	m.AddFeature(openai, gpt4o, chat, urlChat, []string{"vision"}, false)
	m.AddFeature(openai, gpt4o, chat, urlChat, []string{}, true) // 실패 + 빈 features

	b := m.GetBucketReset()
	e := getFeature(t, b, keyOpenAI())
	if e.CallCount != 3 {
		t.Fatalf("CallCount: want 3, got %d", e.CallCount)
	}
	if e.ErrorCount != 1 {
		t.Fatalf("ErrorCount: want 1, got %d", e.ErrorCount)
	}
	if e.Features["vision"] != 2 {
		t.Fatalf("vision: want 2, got %d", e.Features["vision"])
	}
	if e.Features["tool_use"] != 1 {
		t.Fatalf("tool_use: want 1, got %d", e.Features["tool_use"])
	}
}

func TestMeterLLM_AddPerf(t *testing.T) {
	m := newMeterLLM()
	ttft1, ttft2 := 50.0, 30.0
	out10 := int64(10)
	// streaming + ttft + outputTokens > 1 → TPOT 자동 계산
	m.AddPerf(openai, gpt4o, chat, urlChat, 120.5, &ttft1, &out10, true, false)
	// non-streaming → ttft 만 누적, TPOT skip
	m.AddPerf(openai, gpt4o, chat, urlChat, 80.0, &ttft2, &out10, false, false)
	// 실패 케이스 — ttft nil → ttft/TPOT 누적 skip, CallCount/Error/Latency 만 누적
	m.AddPerf(openai, gpt4o, chat, urlChat, 200.0, nil, nil, false, true)

	b := m.GetBucketReset()
	e := getPerf(t, b, keyOpenAI())
	if e.CallCount != 3 {
		t.Fatalf("CallCount: want 3, got %d", e.CallCount)
	}
	if e.ErrorCount != 1 {
		t.Fatalf("ErrorCount: want 1, got %d", e.ErrorCount)
	}
	if !almostEqual(e.LatencySum, 400.5) {
		t.Fatalf("LatencySum: want 400.5, got %f", e.LatencySum)
	}
	if e.TtftCount != 2 || !almostEqual(e.TtftSum, 80.0) {
		t.Fatalf("Ttft: want (2, 80.0), got (%d, %f)", e.TtftCount, e.TtftSum)
	}
	// 첫 호출만 TPOT 계산 — (120.5 - 50.0) / (10-1) = 7.8333...
	wantTpot := (120.5 - 50.0) / 9.0
	if e.TpotCount != 1 || !almostEqual(e.TpotSum, wantTpot) {
		t.Fatalf("Tpot: want (1, %f), got (%d, %f)", wantTpot, e.TpotCount, e.TpotSum)
	}
}

func TestMeterLLM_AddPerf_TpotSkipGuards(t *testing.T) {
	cases := []struct {
		name      string
		latency   float64
		ttft      *float64
		out       *int64
		streaming bool
	}{
		{"non_streaming", 100.0, ptrF(40.0), ptrI(10), false},
		{"output_tokens_nil", 100.0, ptrF(40.0), nil, true},
		{"output_tokens_one", 100.0, ptrF(40.0), ptrI(1), true},
		{"output_tokens_zero", 100.0, ptrF(40.0), ptrI(0), true},
		{"ttft_equals_latency", 100.0, ptrF(100.0), ptrI(10), true}, // tpot=0 → skip
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := newMeterLLM()
			m.AddPerf(openai, gpt4o, chat, urlChat, c.latency, c.ttft, c.out, c.streaming, false)
			b := m.GetBucketReset()
			e := getPerf(t, b, keyOpenAI())
			if e.TpotCount != 0 || e.TpotSum != 0 {
				t.Fatalf("%s: TPOT should be skipped, got (%d, %f)", c.name, e.TpotCount, e.TpotSum)
			}
		})
	}
}

func ptrF(v float64) *float64 { return &v }
func ptrI(v int64) *int64     { return &v }

func TestMeterLLM_AddTokenUsage(t *testing.T) {
	m := newMeterLLM()
	tokens := map[string]int64{"input": 100, "output": 200, "cached": 50}
	costs := map[string]float64{"input": 0.001, "output": 0.004}
	m.AddTokenUsage(openai, gpt4o, chat, urlChat, tokens, costs, false, false)
	m.AddTokenUsage(openai, gpt4o, chat, urlChat,
		map[string]int64{"input": 10},
		map[string]float64{"input": 0.0001},
		true, true)

	b := m.GetBucketReset()
	e := getToken(t, b, keyOpenAI())
	if e.CallCount != 2 {
		t.Fatalf("CallCount: want 2, got %d", e.CallCount)
	}
	if e.ErrorCount != 1 || e.StreamCount != 1 {
		t.Fatalf("Error/Stream: want (1, 1), got (%d, %d)", e.ErrorCount, e.StreamCount)
	}
	if e.TokenCounts["input"] != 110 {
		t.Fatalf("input tokens: want 110, got %d", e.TokenCounts["input"])
	}
	if e.TotalTokens != 360 {
		t.Fatalf("TotalTokens: want 360, got %d", e.TotalTokens)
	}
	wantCost := 0.001 + 0.004 + 0.0001
	if !almostEqual(e.TotalCost, wantCost) {
		t.Fatalf("TotalCost: want %f, got %f", wantCost, e.TotalCost)
	}
}

func TestMeterLLM_15MinMeter(t *testing.T) {
	m := newMeterLLM()
	m.IncrementMeter()
	m.IncrementMeter()
	m.IncrementMeter()
	m.IncrementMeterError()
	m.IncrementMeterError()

	b := m.GetBucketReset()
	if b.MeterCount != 3 {
		t.Fatalf("MeterCount: want 3, got %d", b.MeterCount)
	}
	if b.MeterError != 2 {
		t.Fatalf("MeterError: want 2, got %d", b.MeterError)
	}

	// reset 확인
	b2 := m.GetBucketReset()
	if b2.MeterCount != 0 || b2.MeterError != 0 {
		t.Fatalf("Meter should reset, got count=%d error=%d", b2.MeterCount, b2.MeterError)
	}
}

func TestMeterLLM_Singleton(t *testing.T) {
	a := GetInstanceMeterLLM()
	b := GetInstanceMeterLLM()
	if a != b {
		t.Fatalf("singleton: same pointer required")
	}
}

func TestMeterLLM_Clear(t *testing.T) {
	m := newMeterLLM()
	m.OnStart(openai, gpt4o, chat, urlChat)
	m.AddApiError(openai, gpt4o, chat, urlChat)
	m.IncrementMeter()
	m.Clear()

	if len(m.ActiveSnapshot()) != 0 {
		t.Fatalf("Active should be empty after Clear")
	}
	b := m.GetBucketReset()
	if b.MeterCount != 0 || b.Error.Size() != 0 {
		t.Fatalf("Clear should reset all stats")
	}
}

func TestMeterLLM_SetMaxLRU(t *testing.T) {
	// LinkedMap.SetMax(1024) 가 정상 작동하는지 — 1025 키 넣으면 가장 오래된 것 제거
	m := newMeterLLM()
	for i := 0; i < llmKeyMax+10; i++ {
		m.AddApiError("provX", "modelX", "chat", "/url"+string(rune('a'+(i%26)))+string(rune('0'+(i/26))))
	}
	b := m.GetBucketReset()
	if b.Error.Size() > llmKeyMax {
		t.Fatalf("LinkedMap.SetMax(%d) violated: size=%d", llmKeyMax, b.Error.Size())
	}
}

// ── 헬퍼 — Entry 검색 ──

func getApiStatus(t *testing.T, b *LLMBucket, key LLMKey) *LLMApiStatusEntry {
	t.Helper()
	v := b.ApiStatus.Get(key)
	if v == nil {
		t.Fatalf("ApiStatus entry not found for %s", key.String())
	}
	return v.(*LLMApiStatusEntry)
}

func getError(t *testing.T, b *LLMBucket, key LLMKey) *LLMErrorEntry {
	t.Helper()
	v := b.Error.Get(key)
	if v == nil {
		t.Fatalf("Error entry not found for %s", key.String())
	}
	return v.(*LLMErrorEntry)
}

func getFeature(t *testing.T, b *LLMBucket, key LLMKey) *LLMFeatureEntry {
	t.Helper()
	v := b.Feature.Get(key)
	if v == nil {
		t.Fatalf("Feature entry not found for %s", key.String())
	}
	return v.(*LLMFeatureEntry)
}

func getPerf(t *testing.T, b *LLMBucket, key LLMKey) *LLMPerfEntry {
	t.Helper()
	v := b.Perf.Get(key)
	if v == nil {
		t.Fatalf("Perf entry not found for %s", key.String())
	}
	return v.(*LLMPerfEntry)
}

func getToken(t *testing.T, b *LLMBucket, key LLMKey) *LLMTokenEntry {
	t.Helper()
	v := b.Token.Get(key)
	if v == nil {
		t.Fatalf("Token entry not found for %s", key.String())
	}
	return v.(*LLMTokenEntry)
}

// 컴파일 시 hmap 인터페이스 사용 검증 (cast 가 정상)
var _ hmap.LinkedKey = LLMKey{}
