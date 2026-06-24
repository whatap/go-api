// Package whatapeino — WhaTap LLM monitoring adapter for cloudwego/eino.
//
// Wraps a `model.ChatModel` so every Generate / Stream call publishes a
// WhaTap LLM step (token counts, finish reason, messages, streaming TTFT).
// The wrapped model is a transparent decorator — behavior identical to the
// inner model.
//
// Usage:
//
//	import (
//	    einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
//	    "github.com/whatap/go-api/instrumentation/llm/github.com/cloudwego/eino/whatapeino"
//	)
//
//	chatModel, _ := einoopenai.NewChatModel(ctx, &einoopenai.ChatModelConfig{...})
//	chatModel = whatapeino.WrapChatModel(chatModel)
//
// See dev-docs/issues/252.md.
package whatapeino

import (
	"encoding/json"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/whatap/go-api/llm"
)

const operationTypeChat = "chat"

// extractModel reads the call-time Model option (model.WithModel(...)). Returns
// "" when caller did not override the constructor default.
func extractModel(opts []model.Option) string {
	common := model.GetCommonOptions(&model.Options{}, opts...)
	if common == nil || common.Model == nil {
		return ""
	}
	return *common.Model
}

// extractTemperature reads the call-time Temperature option as a float64,
// or 0 when not set.
func extractTemperature(opts []model.Option) (float64, bool) {
	common := model.GetCommonOptions(&model.Options{}, opts...)
	if common == nil || common.Temperature == nil {
		return 0, false
	}
	return float64(*common.Temperature), true
}

// fillInputs forwards each input message to the appropriate step setter so
// the system / user / assistant / tool turns are captured for the LogSinkPack
// chain.
func fillInputs(step *llm.Step, msgs []*schema.Message) {
	for _, m := range msgs {
		if m == nil {
			continue
		}
		text := messageText(m)
		switch m.Role {
		case schema.System:
			step.AddSystemMessage(text)
		case schema.User:
			step.AddInputMessage(text)
		case schema.Assistant:
			// Prior assistant turns are part of the chat history — record
			// them on the input track so the prompt is reconstructible.
			step.AddInputMessage(text)
		case schema.Tool:
			step.AddToolResult(text)
		}
	}
}

// fillOutput maps the final assistant message returned by Generate (or
// reconstructed by the stream wrapper) into the step.
func fillOutput(step *llm.Step, m *schema.Message) {
	if m == nil {
		return
	}
	if m.Content != "" {
		step.AddOutputMessage(m.Content)
	}
	if m.ReasoningContent != "" {
		step.AddReasoning(m.ReasoningContent)
	}
	if m.ResponseMeta != nil {
		if m.ResponseMeta.FinishReason != "" {
			step.SetFinishReason(m.ResponseMeta.FinishReason)
		}
		if m.ResponseMeta.Usage != nil {
			step.SetTokens(usageToTokens(*m.ResponseMeta.Usage))
		}
	}
	if len(m.ToolCalls) > 0 {
		if payload := marshalToolCalls(m.ToolCalls); payload != "" {
			step.AddTool(payload, "", "")
		}
	}
}

// messageText prefers Content, then falls back to a concatenation of
// MultiContent parts. Non-text parts (image_url / audio_url / video_url /
// file_url) are recorded as `[IMAGE]` / `[AUDIO]` / `[VIDEO]` / `[FILE]`
// placeholders so multimodal calls remain identifiable (python-apm
// content_parser.py 동등). Reasoning parts are skipped here — they flow
// into the dedicated reasoning track via the response message.
func messageText(m *schema.Message) string {
	if m.Content != "" {
		return m.Content
	}
	var out string
	for _, part := range m.MultiContent {
		var chunk string
		switch part.Type {
		case schema.ChatMessagePartTypeText:
			chunk = part.Text
		case schema.ChatMessagePartTypeImageURL:
			chunk = "[IMAGE]"
		case schema.ChatMessagePartTypeAudioURL:
			chunk = "[AUDIO]"
		case schema.ChatMessagePartTypeVideoURL:
			chunk = "[VIDEO]"
		case schema.ChatMessagePartTypeFileURL:
			chunk = "[FILE]"
		case schema.ChatMessagePartTypeReasoning:
			// Skip — reasoning content flows through a separate channel.
			continue
		default:
			if part.Text != "" {
				chunk = part.Text
			} else if part.Type != "" {
				chunk = "[" + strings.ToUpper(string(part.Type)) + "]"
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

func usageToTokens(u schema.TokenUsage) llm.Tokens {
	return llm.Tokens{
		Input:  int64(u.PromptTokens),
		Output: int64(u.CompletionTokens),
		Total:  int64(u.TotalTokens),
	}
}

// marshalToolCalls renders ToolCalls as a JSON array compatible with the
// AddTool payload format (see python-apm tool_calls_text).
func marshalToolCalls(calls []schema.ToolCall) string {
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
			Type: c.Type,
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
