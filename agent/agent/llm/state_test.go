package llm

import (
	"errors"
	"testing"
)

func TestLLMState_AddMessages_Accumulate(t *testing.T) {
	s := NewLLMState(Config{Model: "gpt-4o"})
	s.AddSystemMessage("sys-1")
	s.AddSystemMessage("sys-2")
	s.AddInputMessage("hello")
	s.AddInputMessage("world")
	s.AddOutputMessage("hi")
	s.AddReasoning("thinking...")

	if len(s.systemTexts) != 2 {
		t.Fatalf("systemTexts: want 2 entries, got %d", len(s.systemTexts))
	}
	if s.promptText != "hello\nworld" {
		t.Fatalf("promptText: %q", s.promptText)
	}
	if s.completionText != "hi" {
		t.Fatalf("completionText: %q", s.completionText)
	}
	if s.reasoningText != "thinking..." {
		t.Fatalf("reasoningText: %q", s.reasoningText)
	}
}

func TestLLMState_SetTokens(t *testing.T) {
	s := NewLLMState(Config{Model: "gpt-4o"})
	s.SetTokens(Tokens{Input: 120, Output: 200, Cached: 50})

	if s.pack.InputTokens == nil || *s.pack.InputTokens != 120 {
		t.Fatalf("InputTokens: %+v", s.pack.InputTokens)
	}
	if s.pack.OutputTokens == nil || *s.pack.OutputTokens != 200 {
		t.Fatalf("OutputTokens: %+v", s.pack.OutputTokens)
	}
	if s.pack.CachedTokens == nil || *s.pack.CachedTokens != 50 {
		t.Fatalf("CachedTokens: %+v", s.pack.CachedTokens)
	}
}

func TestLLMState_SetCost(t *testing.T) {
	s := NewLLMState(Config{Model: "gpt-4o"})
	s.SetCost(Cost{Total: 0.005, Input: 0.001, Output: 0.004})

	if s.pack.Cost == nil || *s.pack.Cost != 0.005 {
		t.Fatalf("Cost: %+v", s.pack.Cost)
	}
	if s.pack.InputCost == nil || *s.pack.InputCost != 0.001 {
		t.Fatalf("InputCost: %+v", s.pack.InputCost)
	}
	if s.pack.OutputCost == nil || *s.pack.OutputCost != 0.004 {
		t.Fatalf("OutputCost: %+v", s.pack.OutputCost)
	}
}

func TestLLMState_SetError(t *testing.T) {
	s := NewLLMState(Config{Model: "gpt-4o"})
	s.SetError(errors.New("rate limit"), ErrorTypeAPI)

	if s.pack.Error != "rate limit" {
		t.Fatalf("Error: %q", s.pack.Error)
	}
	if s.pack.ErrorType != ErrorTypeAPI {
		t.Fatalf("ErrorType: %q", s.pack.ErrorType)
	}
	if s.terminalErr == nil {
		t.Fatalf("terminalErr should be captured")
	}
}

func TestLLMState_StreamAndRecordFirstToken(t *testing.T) {
	s := NewLLMState(Config{Model: "gpt-4o"})
	s.MarkStream()
	s.RecordFirstToken()
	first := s.firstTokenMs
	s.RecordFirstToken() // idempotent
	if s.firstTokenMs != first {
		t.Fatalf("RecordFirstToken should be idempotent")
	}
	if !s.streamRequest || !s.pack.Stream {
		t.Fatalf("Stream flag should be set")
	}
}

func TestLLMState_DisabledIsNoOp(t *testing.T) {
	s := DisabledState()
	s.AddInputMessage("ignored")
	s.SetTokens(Tokens{Input: 999})
	s.SetError(errors.New("nope"), ErrorTypeAPI)
	s.MarkStream()

	if s.promptText != "" || s.pack.InputTokens != nil || s.pack.Error != "" || s.pack.Stream {
		t.Fatalf("disabled state mutators must be no-ops")
	}
}

func TestLLMState_AddTool_Payload(t *testing.T) {
	s := NewLLMState(Config{Model: "gpt-4o"})
	s.AddTool(`[{"function":{"name":"get_weather","arguments":"{\"city\":\"Seoul\"}"}}]`, "", "")
	if s.toolCallsText == "" {
		t.Fatalf("toolCallsText should be populated")
	}
}

func TestLLMState_AddTool_NameArgsFallback(t *testing.T) {
	s := NewLLMState(Config{Model: "gpt-4o"})
	s.AddTool("", "send_email", `{"to":"a@b.com"}`)
	if s.toolCallsText == "" {
		t.Fatalf("name+args fallback should populate")
	}
}

func TestLLMState_InfersProviderFromURL(t *testing.T) {
	s := NewLLMState(Config{
		Model: "gpt-4o",
		URL:   "https://api.openai.com/v1/chat/completions",
	})
	if s.pack.Provider != "api.openai.com" {
		t.Fatalf("Provider should be inferred host, got %q", s.pack.Provider)
	}
}
