package llm

import (
	"testing"

	"github.com/whatap/golib/lang/value"
)

func getDouble(t *testing.T, m *value.MapValue, key string) float64 {
	t.Helper()
	v := m.Get(key)
	if v == nil {
		t.Fatalf("key %q missing", key)
	}
	dv, ok := v.(*value.DoubleValue)
	if !ok {
		t.Fatalf("key %q not DoubleValue: %T", key, v)
	}
	return dv.Val
}

func TestParseToolCalls_StandardFormat(t *testing.T) {
	in := `[{"function":{"name":"get_weather","arguments":"{\"city\":\"Seoul\"}"}},{"function":{"name":"send_email","arguments":"{\"to\":\"test@x.com\"}"}}]`
	names, args := parseToolCalls(in)
	if names != "get_weather,send_email" {
		t.Fatalf("names: want 'get_weather,send_email', got %q", names)
	}
	if args == "" {
		t.Fatalf("args should not be empty")
	}
}

func TestParseToolCalls_StringFormat(t *testing.T) {
	// function 이 string 인 경우
	in := `[{"function":"simple_fn","arguments":"{}"}]`
	names, args := parseToolCalls(in)
	if names != "simple_fn" {
		t.Fatalf("names: want 'simple_fn', got %q", names)
	}
	if args != "{}" {
		t.Fatalf("args: want '{}', got %q", args)
	}
}

func TestParseToolCalls_Empty(t *testing.T) {
	names, args := parseToolCalls("")
	if names != "" || args != "" {
		t.Fatalf("empty input should return empty strings")
	}
}

func TestParseToolCalls_InvalidJSON(t *testing.T) {
	names, args := parseToolCalls("not json")
	if names != "" || args != "" {
		t.Fatalf("invalid JSON should return empty strings, got names=%q args=%q", names, args)
	}
}

func TestCombineOutput(t *testing.T) {
	if got := combineOutput("", ""); got != "" {
		t.Fatalf("both empty: want empty, got %q", got)
	}
	if got := combineOutput("reasoning", ""); got != "reasoning" {
		t.Fatalf("only reasoning: want 'reasoning', got %q", got)
	}
	if got := combineOutput("", "completion"); got != "completion" {
		t.Fatalf("only completion: want 'completion', got %q", got)
	}
	if got := combineOutput("reasoning", "completion"); got != "reasoning\ncompletion" {
		t.Fatalf("both: want 'reasoning\\ncompletion', got %q", got)
	}
}

func TestJoinSorted(t *testing.T) {
	in := map[string]struct{}{"openai": {}, "anthropic": {}, "google": {}}
	got := joinSorted(in)
	if got != "anthropic,google,openai" {
		t.Fatalf("joinSorted: want 'anthropic,google,openai', got %q", got)
	}
	if got := joinSorted(map[string]struct{}{}); got != "" {
		t.Fatalf("empty set: want empty, got %q", got)
	}
}

func TestRound6(t *testing.T) {
	if got := round6(1.23456789); got != 1.234568 {
		t.Fatalf("round6: want 1.234568, got %f", got)
	}
	if got := round6(0.1 + 0.2); got != 0.3 {
		t.Fatalf("round6 0.3: got %f", got)
	}
}

func TestBuildStepStatusLineLog(t *testing.T) {
	in1, out1 := int64(100), int64(200)
	cost1 := 0.005
	lat1 := 50.0
	temp1 := 0.7

	s := NewLlmStepStatus()
	s.Txid = "tx-1"
	s.StepID = "step-1"
	s.Index = 0
	s.Provider = "openai"
	s.URL = "/v1/chat/completions"
	s.OperationType = "chat"
	s.Model = "gpt-4o"
	s.Stream = false
	s.Success = true
	s.FinishReason = "stop"
	s.Features = "vision"
	s.Temperature = &temp1
	s.InputTokens = &in1
	s.OutputTokens = &out1
	s.Cost = &cost1
	s.Latency = &lat1

	alog := buildStepStatusLineLog(s)
	if alog.Category != Category {
		t.Fatalf("Category: want %s, got %s", Category, alog.Category)
	}
	if alog.Tags.GetString("llm_log_type") != LogTypeStepStatus {
		t.Fatalf("llm_log_type tag wrong")
	}
	if alog.Tags.GetString("@txid") != "tx-1" {
		t.Fatalf("@txid wrong")
	}
	if alog.Tags.GetString("model") != "gpt-4o" {
		t.Fatalf("model wrong")
	}
	if alog.Tags.GetString("stream") != "False" {
		t.Fatalf("stream wrong")
	}
	if alog.Fields.GetLong("input_tokens.n") != 100 {
		t.Fatalf("input_tokens.n wrong")
	}
	if getDouble(t, alog.Fields, "cost.n") != 0.005 {
		t.Fatalf("cost.n wrong")
	}
	if alog.Fields.GetLong("index") != 0 {
		t.Fatalf("index wrong")
	}
}

func TestBuildTxStatusLineLog(t *testing.T) {
	tx := NewLlmTxStatus()
	tx.Txid = "tx-1"
	tx.FirstStepID = "s1"
	tx.LastStepID = "s3"
	tx.CallCount = 3
	tx.ErrorCount = 1
	tx.Latency = 150.0
	tx.Cost = 0.015
	tx.Models["gpt-4o"] = struct{}{}
	tx.Providers["openai"] = struct{}{}
	tx.OperationTypes["chat"] = struct{}{}
	tx.TokenSums["input_tokens"] = 300

	alog := buildTxStatusLineLog(tx)
	if alog.Tags.GetString("llm_log_type") != LogTypeTxStatus {
		t.Fatalf("llm_log_type wrong")
	}
	if alog.Tags.GetString("@first_step_id") != "s1" {
		t.Fatalf("@first_step_id wrong")
	}
	if alog.Tags.GetString("@last_step_id") != "s3" {
		t.Fatalf("@last_step_id wrong")
	}
	if alog.Tags.GetString("model") != "gpt-4o" {
		t.Fatalf("model wrong")
	}
	if alog.Fields.GetLong("call_count.n") != 3 {
		t.Fatalf("call_count.n wrong")
	}
	if alog.Fields.GetLong("error_count.n") != 1 {
		t.Fatalf("error_count.n wrong")
	}
	if alog.Fields.GetLong("input_tokens.n") != 300 {
		t.Fatalf("input_tokens.n wrong")
	}
	if getDouble(t, alog.Fields, "latency.n") != 150.0 {
		t.Fatalf("latency.n wrong")
	}
}

func TestBuildMessageLineLog_Chunking(t *testing.T) {
	c := &LlmLogSinkPack{
		Txid:     "tx-1",
		StepID:   "step-1",
		Provider: "openai",
		URL:      "/v1/chat/completions",
	}

	// 단일 청크
	alog1 := buildMessageLineLog(c, LogTypeInputMessage, "hello", nil, 0, 1)
	if alog1.Content != "hello" {
		t.Fatalf("single chunk content wrong")
	}
	if alog1.Fields.ContainsKey("chunk_index") {
		t.Fatalf("single chunk should not have chunk_index")
	}

	// 다중 청크
	alog2 := buildMessageLineLog(c, LogTypeInputMessage, "chunk1of3", nil, 0, 3)
	if alog2.Fields.GetLong("chunk_index") != 0 {
		t.Fatalf("chunk_index wrong")
	}
	if alog2.Fields.GetLong("chunk_total") != 3 {
		t.Fatalf("chunk_total wrong")
	}
}
