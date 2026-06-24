package whatapanthropic

import (
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

// usageToTokens — Anthropic Usage 의 4 cache-aware 필드 + Input/Output 매핑이
// llm.Tokens 의 해당 필드로 정확히 들어가는지 확인.
//
// §253 §2.5 의 Usage 차이 (vs §252 OpenAI) — cache_creation_input_tokens
// + cache_read_input_tokens 가 top-level 별도. 회귀 방지.
func TestUsageToTokens_AllFields(t *testing.T) {
	u := anthropic.Usage{
		InputTokens:              120,
		OutputTokens:             80,
		CacheCreationInputTokens: 50,
		CacheReadInputTokens:     30,
	}
	tk := usageToTokens(u)
	if tk.Input != 120 {
		t.Fatalf("Input: %d (want 120)", tk.Input)
	}
	if tk.Output != 80 {
		t.Fatalf("Output: %d (want 80)", tk.Output)
	}
	if tk.Total != 200 {
		t.Fatalf("Total: %d (want 200 = 120+80)", tk.Total)
	}
	if tk.CacheCreationInput != 50 {
		t.Fatalf("CacheCreationInput: %d (want 50)", tk.CacheCreationInput)
	}
	if tk.CacheReadInput != 30 {
		t.Fatalf("CacheReadInput: %d (want 30)", tk.CacheReadInput)
	}
}

func TestUsageToTokens_ZeroCacheFields(t *testing.T) {
	u := anthropic.Usage{InputTokens: 10, OutputTokens: 5}
	tk := usageToTokens(u)
	if tk.CacheCreationInput != 0 || tk.CacheReadInput != 0 {
		t.Fatalf("zero cache fields expected, got %+v", tk)
	}
	if tk.Total != 15 {
		t.Fatalf("Total: %d (want 15)", tk.Total)
	}
}

// chatConfig — Provider 고정값 + Model 전달 + op_type chat.
func TestChatConfig(t *testing.T) {
	cfg := chatConfig(anthropic.MessageNewParams{
		Model: anthropic.Model("claude-sonnet-4-5"),
	})
	if cfg.Provider != "api.anthropic.com" {
		t.Fatalf("Provider: %q", cfg.Provider)
	}
	if cfg.Model != "claude-sonnet-4-5" {
		t.Fatalf("Model: %q", cfg.Model)
	}
	if cfg.OperationType != "chat" {
		t.Fatalf("OperationType: %q", cfg.OperationType)
	}
}

// marshalToolBlocks — 0개 / 1개 / N개 케이스 JSON 정합성.
func TestMarshalToolBlocks_Empty(t *testing.T) {
	if s := marshalToolBlocks(nil); s != "" {
		t.Fatalf("empty must return empty string, got %q", s)
	}
}

func TestMarshalToolBlocks_OneBlock(t *testing.T) {
	out := marshalToolBlocks([]toolUseBlock{
		{ID: "toolu_abc", Name: "get_weather", Arguments: `{"city":"Seoul"}`},
	})
	// {"id":"toolu_abc","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"Seoul\"}"}}
	want := `[{"id":"toolu_abc","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"Seoul\"}"}}]`
	if out != want {
		t.Fatalf("marshal mismatch:\n  got=%s\n want=%s", out, want)
	}
}
