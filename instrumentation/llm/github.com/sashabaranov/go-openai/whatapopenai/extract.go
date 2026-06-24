// Package whatapopenai — WhaTap LLM monitoring adapter for
// github.com/sashabaranov/go-openai.
//
// Wraps the SDK's *openai.Client so every CreateChatCompletion /
// CreateChatCompletionStream / CreateCompletion / CreateEmbeddings call
// publishes a WhaTap LLM step (token counts, finish reason, messages,
// streaming TTFT). The wrapped client embeds *openai.Client so all other
// methods (image / audio / moderation / fine-tune / files / ...) forward
// transparently to the inner client without instrumentation.
//
// Usage:
//
//	import (
//	    "github.com/sashabaranov/go-openai"
//	    "github.com/whatap/go-api/instrumentation/llm/github.com/sashabaranov/go-openai/whatapopenai"
//	)
//
//	client := whatapopenai.WrapClient(openai.NewClient(apiKey))
//	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{...})
//
// See dev-docs/issues/252.md.
package whatapopenai

import (
	"encoding/json"
	"strings"

	openai "github.com/sashabaranov/go-openai"
	"github.com/whatap/go-api/llm"
)

const (
	providerHost = "api.openai.com"

	opChat       = "chat"
	opCompletion = "completions"
	opEmbedding  = "embeddings"
)

// chatConfig builds an llm.Config — URL is captured by the wrapped
// RoundTripper inside the SDK (§267).
func chatConfig(req openai.ChatCompletionRequest) llm.Config {
	return llm.Config{
		Provider:      providerHost,
		Model:         req.Model,
		OperationType: opChat,
	}
}

func completionConfig(req openai.CompletionRequest) llm.Config {
	return llm.Config{
		Provider:      providerHost,
		Model:         req.Model,
		OperationType: opCompletion,
	}
}

func embeddingConfig(model string) llm.Config {
	return llm.Config{
		Provider:      providerHost,
		Model:         model,
		OperationType: opEmbedding,
	}
}

// fillChatInputs maps each request message to the matching step setter.
// MultiContent text parts are concatenated; non-text parts (image_url etc.)
// are recorded as `[IMAGE]` / `[<TYPE>]` placeholders so multimodal calls
// remain identifiable (python-apm content_parser.py 동등).
func fillChatInputs(step *llm.Step, msgs []openai.ChatCompletionMessage) {
	for _, m := range msgs {
		text := chatMessageText(m)
		if text == "" && len(m.ToolCalls) == 0 {
			continue
		}
		switch strings.ToLower(m.Role) {
		case "system", "developer":
			step.AddSystemMessage(text)
		case "user", "assistant":
			step.AddInputMessage(text)
		case "tool", "function":
			step.AddToolResult(text)
		default:
			// unknown role — record on the input track so prompt is reconstructible
			step.AddInputMessage(text)
		}
	}
}

func chatMessageText(m openai.ChatCompletionMessage) string {
	if m.Content != "" {
		return m.Content
	}
	var out string
	for _, p := range m.MultiContent {
		var chunk string
		switch p.Type {
		case openai.ChatMessagePartTypeText:
			chunk = p.Text
		case openai.ChatMessagePartTypeImageURL:
			chunk = "[IMAGE]"
		default:
			if p.Type != "" {
				chunk = "[" + strings.ToUpper(string(p.Type)) + "]"
			}
		}
		if chunk == "" {
			continue
		}
		if out == "" {
			out = chunk
		} else {
			out += "\n" + chunk
		}
	}
	return out
}

// fillChatOutput maps the chat completion response into the step. Only the
// first choice is recorded; multi-choice responses lose 2..N. python-apm has
// the same behaviour.
func fillChatOutput(step *llm.Step, resp *openai.ChatCompletionResponse) {
	if resp == nil {
		return
	}
	if len(resp.Choices) > 0 {
		c := resp.Choices[0]
		if c.Message.Content != "" {
			step.AddOutputMessage(c.Message.Content)
		}
		if c.Message.ReasoningContent != "" {
			step.AddReasoning(c.Message.ReasoningContent)
		}
		if c.FinishReason != "" {
			step.SetFinishReason(string(c.FinishReason))
		}
		if len(c.Message.ToolCalls) > 0 {
			if payload := marshalToolCalls(c.Message.ToolCalls); payload != "" {
				step.AddTool(payload, "", "")
			}
		}
	}
	step.SetTokens(usageToTokens(resp.Usage))
}

func usageToTokens(u openai.Usage) llm.Tokens {
	t := llm.Tokens{
		Input:  int64(u.PromptTokens),
		Output: int64(u.CompletionTokens),
		Total:  int64(u.TotalTokens),
	}
	if u.PromptTokensDetails != nil {
		t.AudioInput = int64(u.PromptTokensDetails.AudioTokens)
		t.Cached = int64(u.PromptTokensDetails.CachedTokens)
	}
	if u.CompletionTokensDetails != nil {
		t.AudioOutput = int64(u.CompletionTokensDetails.AudioTokens)
		t.Reasoning = int64(u.CompletionTokensDetails.ReasoningTokens)
		t.AcceptedPrediction = int64(u.CompletionTokensDetails.AcceptedPredictionTokens)
		t.RejectedPrediction = int64(u.CompletionTokensDetails.RejectedPredictionTokens)
	}
	return t
}

// marshalToolCalls renders ToolCalls as a JSON array compatible with the
// AddTool payload format (see python-apm tool_calls_text).
func marshalToolCalls(calls []openai.ToolCall) string {
	if len(calls) == 0 {
		return ""
	}
	type fn struct {
		Name      string `json:"name,omitempty"`
		Arguments string `json:"arguments,omitempty"`
	}
	type entry struct {
		ID       string `json:"id,omitempty"`
		Type     string `json:"type,omitempty"`
		Function fn     `json:"function"`
	}
	out := make([]entry, 0, len(calls))
	for _, c := range calls {
		out = append(out, entry{
			ID:   c.ID,
			Type: string(c.Type),
			Function: fn{
				Name:      c.Function.Name,
				Arguments: c.Function.Arguments,
			},
		})
	}
	b, err := json.Marshal(out)
	if err != nil {
		return ""
	}
	return string(b)
}

// completionPromptText extracts a text prompt from CompletionRequest.Prompt
// which the SDK accepts as string / []string / []int / [][]int / nil.
func completionPromptText(prompt any) string {
	switch v := prompt.(type) {
	case string:
		return v
	case []string:
		return strings.Join(v, "\n")
	}
	return ""
}
