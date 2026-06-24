package llm

import (
	"errors"
	"math"
	"testing"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/counter/meter"
	"github.com/whatap/golib/util/dateutil"
)

// withLLMMode flips LLMMode for the duration of a test.
func withLLMMode(t *testing.T, on bool) {
	t.Helper()
	cfg := config.GetConfig()
	prev := cfg.LLMMode
	cfg.LLMMode = on
	t.Cleanup(func() { cfg.LLMMode = prev })
}

func TestMaybeAttachAuto_Disabled(t *testing.T) {
	withLLMMode(t, false)
	if state := MaybeAttachAuto("https://api.openai.com/v1/chat/completions"); state != nil {
		t.Fatalf("disabled mode must not auto-attach")
	}
}

func TestMaybeAttachAuto_KnownHost(t *testing.T) {
	withLLMMode(t, true)
	state := MaybeAttachAuto("https://api.openai.com/v1/chat/completions")
	if state == nil {
		t.Fatalf("known LLM host should auto-attach")
	}
	if state.pack.Provider != "api.openai.com" {
		t.Fatalf("Provider: %q", state.pack.Provider)
	}
	if state.pack.OperationType != "chat" {
		t.Fatalf("OperationType: %q", state.pack.OperationType)
	}
}

func TestMaybeAttachAuto_UnknownHost(t *testing.T) {
	withLLMMode(t, true)
	if state := MaybeAttachAuto("https://example.com/api"); state != nil {
		t.Fatalf("unknown host must not attach")
	}
}

func TestHandleHttpcEnd_FillsLatencyAndStepID(t *testing.T) {
	withLLMMode(t, true)
	state := NewLLMState(Config{Provider: "openai", Model: "gpt-4o", OperationType: "chat"})
	state.AddInputMessage("hello")
	state.SetTokens(Tokens{Input: 10, Output: 20})

	var slot interface{}
	HandleHttpcEnd(state, 12345, &slot, 67890, 1234, 200, nil)

	if state.pack.Latency == nil || *state.pack.Latency != 1234 {
		t.Fatalf("Latency: %+v", state.pack.Latency)
	}
	if state.pack.StepID != "67890" {
		t.Fatalf("StepID: %q", state.pack.StepID)
	}
	if state.pack.Txid != "12345" {
		t.Fatalf("Txid: %q", state.pack.Txid)
	}
	if !state.pack.Success {
		t.Fatalf("Success should be true")
	}
	if slot == nil {
		t.Fatalf("LLMTx slot should be lazy-allocated")
	}
	tx, ok := slot.(*LlmTxStatus)
	if !ok || tx.CallCount != 1 {
		t.Fatalf("tx.CallCount: %+v", tx)
	}
}

func TestHandleHttpcEnd_PromotesHttpcErr(t *testing.T) {
	withLLMMode(t, true)
	state := NewLLMState(Config{Provider: "openai", Model: "gpt-4o", OperationType: "chat"})
	httpErr := errors.New("connection reset")

	var slot interface{}
	HandleHttpcEnd(state, 1, &slot, 2, 100, 500, httpErr)

	if state.pack.Success {
		t.Fatalf("Success should be false on httpc err")
	}
	if state.pack.ErrorType != ErrorTypeAPI {
		t.Fatalf("ErrorType: want api_error, got %q", state.pack.ErrorType)
	}
	if state.pack.Error != "connection reset" {
		t.Fatalf("Error msg: %q", state.pack.Error)
	}
}

func TestHandleHttpcEnd_PreservesExplicitSetError(t *testing.T) {
	withLLMMode(t, true)
	state := NewLLMState(Config{Provider: "openai", Model: "gpt-4o", OperationType: "chat"})
	state.SetError(errors.New("rate limit"), ErrorTypeAPI)

	var slot interface{}
	HandleHttpcEnd(state, 1, &slot, 2, 100, 200, errors.New("ignored"))

	if state.pack.Error != "rate limit" {
		t.Fatalf("explicit SetError must win: got %q", state.pack.Error)
	}
}

func TestHandleHttpcEnd_NilState(t *testing.T) {
	// must not panic
	HandleHttpcEnd(nil, 1, nil, 2, 100, 200, nil)
}

func TestHandleHttpcEnd_AccumulatesAcrossCalls(t *testing.T) {
	withLLMMode(t, true)
	var slot interface{}

	for i := 0; i < 3; i++ {
		s := NewLLMState(Config{Provider: "openai", Model: "gpt-4o", OperationType: "chat"})
		s.SetTokens(Tokens{Input: 10, Output: 20})
		HandleHttpcEnd(s, 1, &slot, int64(i+1), 100, 200, nil)
	}

	tx := slot.(*LlmTxStatus)
	if tx.CallCount != 3 {
		t.Fatalf("CallCount: want 3, got %d", tx.CallCount)
	}
	if tx.FirstStepID != "1" || tx.LastStepID != "3" {
		t.Fatalf("step ID range: first=%q last=%q", tx.FirstStepID, tx.LastStepID)
	}
}

func TestDispatchTraceTxStatus_NilOrEmpty(t *testing.T) {
	withLLMMode(t, true)
	// must not panic / no-op
	DispatchTraceTxStatus(0, nil)

	tx := NewLlmTxStatus()
	DispatchTraceTxStatus(1, tx) // CallCount==0 → no-op
}

func TestDispatchTraceTxStatus_FillsTxidWhenEmpty(t *testing.T) {
	withLLMMode(t, true)
	tx := NewLlmTxStatus()
	tx.CallCount = 1 // pretend at least one call accumulated
	DispatchTraceTxStatus(99, tx)
	if tx.Txid != "99" {
		t.Fatalf("Txid should be filled from txid arg, got %q", tx.Txid)
	}
}

func TestSplitFeatures(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"tool_use", []string{"tool_use"}},
		{"tool_use,reasoning", []string{"tool_use", "reasoning"}},
		{",,a,,b,,", []string{"a", "b"}},
	}
	for _, c := range cases {
		got := splitFeatures(c.in)
		if len(got) != len(c.want) {
			t.Fatalf("splitFeatures(%q): want %v got %v", c.in, c.want, got)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("splitFeatures(%q)[%d]: want %q got %q", c.in, i, c.want[i], got[i])
			}
		}
	}
}

// §262 — publishMeters 가 AddPerf 로 TPOT 자동 계산을 발행하는지 검증.
// streaming + OutputTokens > 1 + Ttft 셋업 후 HandleHttpcEnd → MeterLLM 누적 확인.
func TestPublishMeters_AutoComputesTpot(t *testing.T) {
	withLLMMode(t, true)
	m := meter.GetInstanceMeterLLM()
	m.Clear()
	t.Cleanup(func() { m.Clear() })

	state := NewLLMState(Config{Provider: "openai", Model: "gpt-4o", OperationType: "chat", URL: "https://api.openai.com/v1/chat/completions"})
	state.MarkStream()
	state.SetTokens(Tokens{Output: 10})
	// firstTokenMs 강제 셋업 — Ttft = 40ms 가 되도록
	state.firstTokenMs = state.startMs + 40

	var slot interface{}
	HandleHttpcEnd(state, 1, &slot, 2, 100, 200, nil) // latency=100ms

	b := m.GetBucketReset()
	key := meter.LLMKey{Provider: "openai", Model: "gpt-4o", OperationType: "chat", URL: "https://api.openai.com/v1/chat/completions"}
	v := b.Perf.Get(key)
	if v == nil {
		t.Fatalf("Perf entry not found for key %s", key.String())
	}
	e := v.(*meter.LLMPerfEntry)
	if e.CallCount != 1 {
		t.Fatalf("CallCount: want 1, got %d", e.CallCount)
	}
	if e.LatencySum != 100 {
		t.Fatalf("LatencySum: want 100, got %f", e.LatencySum)
	}
	if e.TtftCount != 1 || e.TtftSum != 40 {
		t.Fatalf("Ttft: want (1, 40), got (%d, %f)", e.TtftCount, e.TtftSum)
	}
	wantTpot := (100.0 - 40.0) / 9.0
	if e.TpotCount != 1 || math.Abs(e.TpotSum-wantTpot) > 1e-9 {
		t.Fatalf("Tpot: want (1, %f), got (%d, %f)", wantTpot, e.TpotCount, e.TpotSum)
	}
}

// §267 — stream 케이스: HandleHttpcEnd 가 stream body 시작 *전* 호출되어
// firstTokenMs=0 이고, 이후 RecordFirstToken + Publish (Step.End 가 호출).
// Publish 의 fallback Ttft 계산이 정확해서 ttft/tpot 메트릭이 누적되는지 검증.
func TestPublishMeters_StreamAfterHttpcEnd(t *testing.T) {
	withLLMMode(t, true)
	m := meter.GetInstanceMeterLLM()
	m.Clear()
	t.Cleanup(func() { m.Clear() })

	state := NewLLMState(Config{Provider: "openai", Model: "gpt-4o", OperationType: "chat", URL: "https://api.openai.com/v1/chat/completions"})
	state.MarkManualPublish() // §267 — Step.End 가 publish 책임
	state.MarkStream()
	state.SetTokens(Tokens{Output: 10})

	// startMs 를 100ms 과거로 set — Publish 가 latency = now-startMs 로
	// 재계산할 때 RoundTrip transport latency 보다 큰 stream-completion
	// latency 가 나오도록 함 (real 환경의 SSE 대기 시간 시뮬레이션).
	state.startMs = dateutil.SystemNow() - 100

	var slot interface{}
	// HandleHttpcEnd — RoundTrip 의 transport.RoundTrip return 시점 latency
	// (SSE header 받은 시점). stream body 가 아직이므로 ttft 보다 작을 수
	// 있어 stream 케이스는 Publish 가 latency 재계산. 여기서 latency=33ms
	// 로 set 해두면 Publish 의 재계산이 덮어쓰는 흐름을 검증할 수 있다.
	HandleHttpcEnd(state, 1, &slot, 2, 33, 200, nil) // 짧은 transport latency

	// stream body 첫 chunk — RecordFirstToken 가 startMs + 40 으로 set.
	state.firstTokenMs = state.startMs + 40

	// Step.End → state.Publish() — Ttft fallback + latency 재계산 + publishMeters.
	state.Publish()

	b := m.GetBucketReset()
	key := meter.LLMKey{Provider: "openai", Model: "gpt-4o", OperationType: "chat", URL: "https://api.openai.com/v1/chat/completions"}
	v := b.Perf.Get(key)
	if v == nil {
		t.Fatalf("Perf entry not found for key %s", key.String())
	}
	e := v.(*meter.LLMPerfEntry)
	if e.CallCount != 1 {
		t.Fatalf("CallCount: want 1, got %d", e.CallCount)
	}
	if e.TtftCount != 1 || e.TtftSum != 40 {
		t.Fatalf("Ttft (stream-after-httpc-end): want (1, 40), got (%d, %f)", e.TtftCount, e.TtftSum)
	}
	// Publish 의 latency 재계산이 startMs (-100ms) 기반이므로 latency >= 100.
	// tpot = (latency - 40) / 9. tpot > 0 (양수) 가 보장됨.
	if e.TpotCount != 1 || e.TpotSum <= 0 {
		t.Fatalf("Tpot: want (1, >0), got (%d, %f)", e.TpotCount, e.TpotSum)
	}
}

// §262 — non-streaming 케이스에서 TPOT 발행이 skip 되는지 검증.
func TestPublishMeters_NonStreamingSkipsTpot(t *testing.T) {
	withLLMMode(t, true)
	m := meter.GetInstanceMeterLLM()
	m.Clear()
	t.Cleanup(func() { m.Clear() })

	state := NewLLMState(Config{Provider: "openai", Model: "gpt-4o", OperationType: "chat", URL: "https://api.openai.com/v1/chat/completions"})
	// MarkStream() 호출 안 함 — non-streaming
	state.SetTokens(Tokens{Output: 10})
	state.firstTokenMs = state.startMs + 40

	var slot interface{}
	HandleHttpcEnd(state, 1, &slot, 2, 100, 200, nil)

	b := m.GetBucketReset()
	key := meter.LLMKey{Provider: "openai", Model: "gpt-4o", OperationType: "chat", URL: "https://api.openai.com/v1/chat/completions"}
	v := b.Perf.Get(key)
	if v == nil {
		t.Fatalf("Perf entry not found")
	}
	e := v.(*meter.LLMPerfEntry)
	if e.TpotCount != 0 || e.TpotSum != 0 {
		t.Fatalf("non-streaming: TPOT should be skipped, got (%d, %f)", e.TpotCount, e.TpotSum)
	}
}

func TestNonZeroTokenCounts_SkipsNilAndZero(t *testing.T) {
	p := NewLlmStepStatus()
	v100 := int64(100)
	v0 := int64(0)
	p.InputTokens = &v100
	p.OutputTokens = &v0 // zero — must skip
	p.CachedTokens = nil // nil — must skip

	m := nonZeroTokenCounts(p)
	if m["input_tokens"] != 100 {
		t.Fatalf("input_tokens missing: %d", m["input_tokens"])
	}
	if _, ok := m["output_tokens"]; ok {
		t.Fatalf("zero output_tokens must be skipped")
	}
	if _, ok := m["cached_tokens"]; ok {
		t.Fatalf("nil cached_tokens must be skipped")
	}
}

// §254 Step 5 — AttachForced tests.

func TestAttachForced_Disabled(t *testing.T) {
	withLLMMode(t, false)
	if state := AttachForced("https://api.openai.com/v1/chat/completions"); state != nil {
		t.Fatalf("disabled mode must not force-attach")
	}
}

func TestAttachForced_EmptyURL(t *testing.T) {
	withLLMMode(t, true)
	if state := AttachForced(""); state != nil {
		t.Fatalf("empty url must not force-attach")
	}
}

func TestAttachForced_KnownHostUsesProviderTable(t *testing.T) {
	withLLMMode(t, true)
	state := AttachForced("https://api.openai.com/v1/chat/completions")
	if state == nil {
		t.Fatalf("known host should force-attach")
	}
	if state.pack.Provider != "api.openai.com" {
		t.Fatalf("Provider: %q (want canonical from URL match table)", state.pack.Provider)
	}
	if state.pack.OperationType != "chat" {
		t.Fatalf("OperationType: %q (want chat from URL match table)", state.pack.OperationType)
	}
}

func TestAttachForced_UnknownHostUsesURLHost(t *testing.T) {
	withLLMMode(t, true)
	state := AttachForced("http://localhost:8161/v1/chat/completions")
	if state == nil {
		t.Fatalf("unknown host with llm_enabled should still force-attach (§254 Step 5)")
	}
	if state.pack.Provider != "localhost:8161" {
		t.Fatalf("Provider: %q (want host fallback)", state.pack.Provider)
	}
	if state.pack.URL != "http://localhost:8161/v1/chat/completions" {
		t.Fatalf("URL: %q (want raw URL preserved)", state.pack.URL)
	}
}
