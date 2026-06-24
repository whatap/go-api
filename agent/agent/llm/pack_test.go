package llm

import (
	"testing"
)

func TestInferProviderURL(t *testing.T) {
	cases := []struct {
		in        string
		wantHost  string
		wantPath  string
	}{
		{"https://api.openai.com/v1/chat/completions", "api.openai.com", "/v1/chat/completions"},
		{"http://localhost:8080/proxy", "localhost:8080", "/proxy"},
		{"https://api.anthropic.com", "api.anthropic.com", ""},
		{"api.openai.com/v1/chat", "api.openai.com", "/v1/chat"},
		{"", "", ""},
	}
	for _, c := range cases {
		host, path := InferProviderURL(c.in)
		if host != c.wantHost || path != c.wantPath {
			t.Fatalf("InferProviderURL(%q) = (%q, %q), want (%q, %q)",
				c.in, host, path, c.wantHost, c.wantPath)
		}
	}
}

func TestLlmStepStatus_SetError(t *testing.T) {
	s := NewLlmStepStatus()
	s.SetError("connection refused", "api_error")
	if s.Error != "connection refused" || s.ErrorType != "api_error" {
		t.Fatalf("SetError failed")
	}
}

func TestLlmStepStatus_SetTokens(t *testing.T) {
	s := NewLlmStepStatus()
	s.SetTokens(map[string]int64{
		"input_tokens":               100,
		"output_tokens":              200,
		"cached_tokens":              50,
		"reasoning_tokens":           10,
		"cache_read_input_tokens":    5,
		"unknown_field":              999, // ignored
	})
	if *s.InputTokens != 100 {
		t.Fatalf("InputTokens: want 100, got %d", *s.InputTokens)
	}
	if *s.OutputTokens != 200 {
		t.Fatalf("OutputTokens: want 200, got %d", *s.OutputTokens)
	}
	if *s.CachedTokens != 50 {
		t.Fatalf("CachedTokens: want 50, got %d", *s.CachedTokens)
	}
	if *s.CacheReadInputTokens != 5 {
		t.Fatalf("CacheReadInputTokens: want 5, got %d", *s.CacheReadInputTokens)
	}
}

func TestLlmTxStatus_Accumulate(t *testing.T) {
	tx := NewLlmTxStatus()

	in1, out1, lat1, cost1 := int64(100), int64(200), 50.0, 0.005
	step1 := NewLlmStepStatus()
	step1.StepID = "s1"
	step1.Model = "gpt-4o"
	step1.Provider = "openai"
	step1.OperationType = "chat"
	step1.Success = true
	step1.InputTokens = &in1
	step1.OutputTokens = &out1
	step1.Latency = &lat1
	step1.Cost = &cost1
	tx.Accumulate(step1)

	in2, out2, lat2 := int64(50), int64(80), 30.0
	step2 := NewLlmStepStatus()
	step2.StepID = "s2"
	step2.Model = "gpt-4o-mini"
	step2.Provider = "openai"
	step2.OperationType = "chat"
	step2.Success = false // 실패
	step2.InputTokens = &in2
	step2.OutputTokens = &out2
	step2.Latency = &lat2
	tx.Accumulate(step2)

	if tx.CallCount != 2 {
		t.Fatalf("CallCount: want 2, got %d", tx.CallCount)
	}
	if tx.ErrorCount != 1 {
		t.Fatalf("ErrorCount: want 1, got %d", tx.ErrorCount)
	}
	if tx.TokenSums["input_tokens"] != 150 {
		t.Fatalf("input_tokens sum: want 150, got %d", tx.TokenSums["input_tokens"])
	}
	if tx.TokenSums["output_tokens"] != 280 {
		t.Fatalf("output_tokens sum: want 280, got %d", tx.TokenSums["output_tokens"])
	}
	if tx.Latency != 80.0 {
		t.Fatalf("Latency sum: want 80.0, got %f", tx.Latency)
	}
	if tx.Cost != 0.005 {
		t.Fatalf("Cost sum: want 0.005, got %f", tx.Cost)
	}
	if tx.FirstStepID != "s1" || tx.LastStepID != "s2" {
		t.Fatalf("StepID: first=%s last=%s", tx.FirstStepID, tx.LastStepID)
	}
	if _, ok := tx.Models["gpt-4o"]; !ok {
		t.Fatalf("gpt-4o not in Models")
	}
	if _, ok := tx.Models["gpt-4o-mini"]; !ok {
		t.Fatalf("gpt-4o-mini not in Models")
	}
	if _, ok := tx.Providers["openai"]; !ok {
		t.Fatalf("openai not in Providers")
	}
	if _, ok := tx.OperationTypes["chat"]; !ok {
		t.Fatalf("chat not in OperationTypes")
	}
}

func TestLlmTxStatus_AccumulateUnknownOpType(t *testing.T) {
	tx := NewLlmTxStatus()
	step := NewLlmStepStatus()
	step.OperationType = "unknown" // default — set 안 해도 unknown
	step.Success = true
	tx.Accumulate(step)
	if _, ok := tx.OperationTypes["unknown"]; ok {
		t.Fatalf("unknown should be excluded from OperationTypes set")
	}
}
