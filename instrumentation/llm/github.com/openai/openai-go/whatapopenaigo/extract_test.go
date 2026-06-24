package whatapopenaigo

import (
	"testing"

	"github.com/openai/openai-go"
)

// usageToTokens — OpenAI CompletionUsage 의 prompt/completion/total 기본 3종 +
// PromptTokensDetails.CachedTokens / AudioTokens + CompletionTokensDetails
// 의 ReasoningTokens / AudioTokens / AcceptedPrediction / RejectedPrediction
// 모두 llm.Tokens 의 해당 필드로 정확히 들어가는지 확인.
//
// §255 §2.5 의 Usage 차이 (vs §252 sashabaranov) — 응답 구조는 OpenAI 호환이지만
// 공식 SDK 는 PromptTokensDetails / CompletionTokensDetails 를 raw struct 로 제공.
func TestUsageToTokens_AllFields(t *testing.T) {
	u := openai.CompletionUsage{
		PromptTokens:     120,
		CompletionTokens: 80,
		TotalTokens:      200,
	}
	u.PromptTokensDetails.CachedTokens = 30
	u.PromptTokensDetails.AudioTokens = 5
	u.CompletionTokensDetails.ReasoningTokens = 40
	u.CompletionTokensDetails.AudioTokens = 7
	u.CompletionTokensDetails.AcceptedPredictionTokens = 10
	u.CompletionTokensDetails.RejectedPredictionTokens = 3

	tk := usageToTokens(u)
	if tk.Input != 120 {
		t.Fatalf("Input: %d (want 120)", tk.Input)
	}
	if tk.Output != 80 {
		t.Fatalf("Output: %d (want 80)", tk.Output)
	}
	if tk.Total != 200 {
		t.Fatalf("Total: %d (want 200)", tk.Total)
	}
	if tk.Cached != 30 {
		t.Fatalf("Cached: %d (want 30)", tk.Cached)
	}
	if tk.AudioInput != 5 {
		t.Fatalf("AudioInput: %d (want 5)", tk.AudioInput)
	}
	if tk.AudioOutput != 7 {
		t.Fatalf("AudioOutput: %d (want 7)", tk.AudioOutput)
	}
	if tk.Reasoning != 40 {
		t.Fatalf("Reasoning: %d (want 40)", tk.Reasoning)
	}
	if tk.AcceptedPrediction != 10 {
		t.Fatalf("AcceptedPrediction: %d (want 10)", tk.AcceptedPrediction)
	}
	if tk.RejectedPrediction != 3 {
		t.Fatalf("RejectedPrediction: %d (want 3)", tk.RejectedPrediction)
	}
}

func TestUsageToTokens_ZeroDetails(t *testing.T) {
	u := openai.CompletionUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15}
	tk := usageToTokens(u)
	if tk.Cached != 0 || tk.Reasoning != 0 || tk.AudioInput != 0 {
		t.Fatalf("zero detail fields expected, got %+v", tk)
	}
	if tk.Total != 15 {
		t.Fatalf("Total: %d (want 15)", tk.Total)
	}
}

// chatConfig — Provider 고정값 + Model 전달 + op_type chat.
func TestChatConfig(t *testing.T) {
	cfg := chatConfig(openai.ChatCompletionNewParams{
		Model: "gpt-4o",
	})
	if cfg.Provider != "api.openai.com" {
		t.Fatalf("Provider: %q", cfg.Provider)
	}
	if cfg.Model != "gpt-4o" {
		t.Fatalf("Model: %q", cfg.Model)
	}
	if cfg.OperationType != "chat" {
		t.Fatalf("OperationType: %q", cfg.OperationType)
	}
}
