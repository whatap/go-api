package whatapopenaigo

import (
	"encoding/json"
	"strings"

	"github.com/openai/openai-go"
	"github.com/whatap/go-api/llm"
)

const (
	providerHost = "api.openai.com"

	opChat = "chat"
)

// chatConfig builds an llm.Config — URL is captured by the wrapped
// RoundTripper inside the SDK (§267).
func chatConfig(params openai.ChatCompletionNewParams) llm.Config {
	return llm.Config{
		Provider:      providerHost,
		Model:         string(params.Model),
		OperationType: opChat,
	}
}

// fillChatInputs walks the request Messages slice. Each message variant
// (System / User / Assistant / Tool) carries its content through a
// discriminated union — we record the textual portion via the SDK's helper
// accessors. Multimodal parts (image / audio) are recorded as `[<TYPE>]`
// placeholders so prompt reconstruction remains identifiable.
func fillChatInputs(step *llm.Step, params openai.ChatCompletionNewParams) {
	for _, m := range params.Messages {
		text, role := messageParamText(m)
		if text == "" {
			continue
		}
		switch role {
		case "system":
			step.AddSystemMessage(text)
		case "assistant", "tool":
			step.AddInputMessage(text)
		default:
			step.AddInputMessage(text)
		}
	}
}

// messageParamText extracts (text, role) from a ChatCompletionMessageParamUnion.
// The union has nullable variant pointers (OfSystem / OfUser / OfAssistant /
// OfTool / OfDeveloper / OfFunction) — we inspect each for text-bearing
// content. Multipart content (text+image arrays) collapses to text parts +
// placeholders.
func messageParamText(m openai.ChatCompletionMessageParamUnion) (string, string) {
	if sys := m.OfSystem; sys != nil {
		return contentUnionString(sys.Content.OfString.Value, sys.Content.OfArrayOfContentParts), "system"
	}
	if usr := m.OfUser; usr != nil {
		// User content can be string or array of parts
		if usr.Content.OfString.Valid() {
			return usr.Content.OfString.Value, "user"
		}
		var parts []string
		for _, p := range usr.Content.OfArrayOfContentParts {
			if t := p.OfText; t != nil && t.Text != "" {
				parts = append(parts, t.Text)
				continue
			}
			if p.OfImageURL != nil {
				parts = append(parts, "[IMAGE]")
				continue
			}
			if p.OfInputAudio != nil {
				parts = append(parts, "[AUDIO]")
				continue
			}
			if p.OfFile != nil {
				parts = append(parts, "[FILE]")
				continue
			}
		}
		return strings.Join(parts, "\n"), "user"
	}
	if asst := m.OfAssistant; asst != nil {
		if asst.Content.OfString.Valid() {
			return asst.Content.OfString.Value, "assistant"
		}
		var parts []string
		for _, p := range asst.Content.OfArrayOfContentParts {
			if t := p.OfText; t != nil && t.Text != "" {
				parts = append(parts, t.Text)
				continue
			}
			if p.OfRefusal != nil && p.OfRefusal.Refusal != "" {
				parts = append(parts, "[REFUSAL] "+p.OfRefusal.Refusal)
			}
		}
		return strings.Join(parts, "\n"), "assistant"
	}
	if tool := m.OfTool; tool != nil {
		return contentUnionString(tool.Content.OfString.Value, nil), "tool"
	}
	if dev := m.OfDeveloper; dev != nil {
		return contentUnionString(dev.Content.OfString.Value, nil), "system"
	}
	return "", ""
}

// contentUnionString — helper to flatten a (string | []TextPart) union into
// plain text. Returns empty if both are empty.
func contentUnionString(str string, parts []openai.ChatCompletionContentPartTextParam) string {
	if str != "" {
		return str
	}
	if len(parts) == 0 {
		return ""
	}
	chunks := make([]string, 0, len(parts))
	for _, p := range parts {
		if p.Text != "" {
			chunks = append(chunks, p.Text)
		}
	}
	return strings.Join(chunks, "\n")
}

// fillChatOutput maps the ChatCompletion response into the step. Choices is
// usually length 1 (n=1 by default). Tool calls are serialised as a JSON
// array under the AddTool payload format.
func fillChatOutput(step *llm.Step, resp *openai.ChatCompletion) {
	if resp == nil || len(resp.Choices) == 0 {
		return
	}
	ch := resp.Choices[0]
	if ch.Message.Content != "" {
		step.AddOutputMessage(ch.Message.Content)
	}
	if ch.Message.Refusal != "" {
		step.AddReasoning("[REFUSAL] " + ch.Message.Refusal)
	}
	if len(ch.Message.ToolCalls) > 0 {
		if payload := marshalToolCalls(ch.Message.ToolCalls); payload != "" {
			step.AddTool(payload, "", "")
		}
	}
	if ch.FinishReason != "" {
		step.SetFinishReason(ch.FinishReason)
	}
	step.SetTokens(usageToTokens(resp.Usage))
}

// marshalToolCalls — render tool_calls as a JSON array compatible with the
// AddTool payload format (python-apm tool_calls_text equivalent). The
// OpenAI SDK currently only supports `Type == constant.Function`, so we
// emit type="function" verbatim.
func marshalToolCalls(calls []openai.ChatCompletionMessageToolCall) string {
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
			Type: "function",
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

// usageToTokens maps openai.CompletionUsage into llm.Tokens. Cached prompt
// tokens (prompt_tokens_details.cached_tokens) and reasoning / audio /
// accepted/rejected prediction tokens (completion_tokens_details.*) are
// captured as the corresponding llm.Tokens cache-aware fields.
func usageToTokens(u openai.CompletionUsage) llm.Tokens {
	return llm.Tokens{
		Input:              u.PromptTokens,
		Output:             u.CompletionTokens,
		Total:              u.TotalTokens,
		Cached:             u.PromptTokensDetails.CachedTokens,
		AudioInput:         u.PromptTokensDetails.AudioTokens,
		AudioOutput:        u.CompletionTokensDetails.AudioTokens,
		Reasoning:          u.CompletionTokensDetails.ReasoningTokens,
		AcceptedPrediction: u.CompletionTokensDetails.AcceptedPredictionTokens,
		RejectedPrediction: u.CompletionTokensDetails.RejectedPredictionTokens,
	}
}
