package llm

import (
	"context"
	"testing"
)

func TestRegisterPending_Basic(t *testing.T) {
	state := NewLLMState(Config{Provider: "openai", Model: "gpt-4o", OperationType: "chat"})
	ctx := RegisterPending(context.Background(), state)

	got := TakePending(ctx)
	if got != state {
		t.Fatalf("TakePending: got %p, want %p", got, state)
	}
}

func TestRegisterPending_NilCtxOrState(t *testing.T) {
	// nil ctx — graceful handling, returns nil unchanged
	//nolint:staticcheck // SA1012: intentional nil-ctx defensive check
	if got := RegisterPending(nil, NewLLMState(Config{})); got != nil {
		t.Errorf("RegisterPending(nil, state): got non-nil, want nil")
	}
	// nil state — returns parent unchanged
	parent := context.Background()
	if got := RegisterPending(parent, nil); got != parent {
		t.Errorf("RegisterPending(parent, nil): expected parent unchanged")
	}
	// disabled state — returns parent unchanged (skip registration)
	if got := RegisterPending(parent, DisabledState()); got != parent {
		t.Errorf("RegisterPending(parent, disabled): expected parent unchanged")
	}
	if TakePending(parent) != nil {
		t.Errorf("TakePending on parent: expected nil")
	}
}

func TestRegisterPending_Idempotent(t *testing.T) {
	first := NewLLMState(Config{Provider: "openai", Model: "gpt-4o"})
	second := NewLLMState(Config{Provider: "anthropic", Model: "claude"})

	ctx := RegisterPending(context.Background(), first)
	ctx2 := RegisterPending(ctx, second)

	got := TakePending(ctx2)
	if got != first {
		t.Errorf("Idempotent: expected first registration to win, got different state")
	}
	if ctx2 != ctx {
		t.Errorf("Idempotent: expected unchanged ctx when pending already present")
	}
}

func TestTakePending_NoneRegistered(t *testing.T) {
	if got := TakePending(context.Background()); got != nil {
		t.Errorf("TakePending on empty ctx: expected nil, got %p", got)
	}
	//nolint:staticcheck // SA1012: intentional nil-ctx defensive check
	if got := TakePending(nil); got != nil {
		t.Errorf("TakePending(nil): expected nil, got %p", got)
	}
}

func TestSetURL_UpdatesEmptyFields(t *testing.T) {
	state := NewLLMState(Config{Model: "gpt-4o"}) // Provider/URL empty
	state.SetURL("https://api.openai.com/v1/chat/completions")

	if state.pack.Provider != "api.openai.com" {
		t.Errorf("Provider: got %q, want %q", state.pack.Provider, "api.openai.com")
	}
	if state.pack.URL != "/v1/chat/completions" {
		t.Errorf("URL: got %q, want %q", state.pack.URL, "/v1/chat/completions")
	}
	if state.pack.OperationType != "chat" {
		t.Errorf("OperationType: got %q, want %q", state.pack.OperationType, "chat")
	}
}

func TestSetURL_PreservesUserSuppliedConfig(t *testing.T) {
	state := NewLLMState(Config{
		Provider:      "custom-proxy",
		Model:         "gpt-4o",
		OperationType: "chat-custom",
		URL:           "/v1/chat/completions", // already a path
	})
	state.SetURL("https://api.openai.com/v1/chat/completions")

	if state.pack.Provider != "custom-proxy" {
		t.Errorf("Provider: should not be overwritten, got %q", state.pack.Provider)
	}
	if state.pack.OperationType != "chat-custom" {
		t.Errorf("OperationType: should not be overwritten, got %q", state.pack.OperationType)
	}
	if state.pack.URL != "/v1/chat/completions" {
		t.Errorf("URL: should remain path, got %q", state.pack.URL)
	}
}

func TestSetURL_NilOrDisabled(t *testing.T) {
	// nil receiver — must not panic
	var nilState *LLMState
	nilState.SetURL("https://api.openai.com/v1/chat/completions")

	// disabled state — no-op
	disabled := DisabledState()
	disabled.SetURL("https://api.openai.com/v1/chat/completions")
	if disabled.pack.URL != "" {
		t.Errorf("Disabled state URL should remain empty, got %q", disabled.pack.URL)
	}
}
