package whatapanthropic

import (
	"encoding/json"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/whatap/go-api/llm"
)

const (
	providerHost = "api.anthropic.com"

	opChat = "chat"
)

// chatConfig builds an llm.Config — URL is captured by the wrapped
// RoundTripper inside the SDK (§267).
func chatConfig(params anthropic.MessageNewParams) llm.Config {
	return llm.Config{
		Provider:      providerHost,
		Model:         string(params.Model),
		OperationType: opChat,
	}
}

// fillChatInputs walks MessageNewParams.System + Messages and records each
// input message on the step. Anthropic 의 System 은 별도 필드 (배열).
// Messages 의 각 ContentBlockParamUnion 은 text / tool_result / image 등
// 여러 variant — 단순화 위해 text 만 concatenate, 나머지는 `[<TYPE>]`
// placeholder.
func fillChatInputs(step *llm.Step, params anthropic.MessageNewParams) {
	// System 메시지 (별도 배열)
	for _, sb := range params.System {
		if t := sb.Text; t != "" {
			step.AddSystemMessage(t)
		}
	}

	// 일반 messages (user / assistant turn)
	for _, m := range params.Messages {
		text := messageParamText(m)
		if text == "" {
			continue
		}
		switch m.Role {
		case anthropic.MessageParamRoleUser:
			step.AddInputMessage(text)
		case anthropic.MessageParamRoleAssistant:
			step.AddInputMessage(text)
		default:
			step.AddInputMessage(text)
		}
	}
}

// messageParamText concatenates the textual content of a MessageParam.
// Tool_result 블록은 ToolResult 의 content (text) 누적, image / document 등은
// `[IMAGE]` / `[<TYPE>]` placeholder.
func messageParamText(m anthropic.MessageParam) string {
	parts := make([]string, 0, len(m.Content))
	for _, blk := range m.Content {
		if t := blockParamText(blk); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, "\n")
}

// blockParamText extracts text from a single ContentBlockParamUnion. The
// union has nullable variant pointers (OfText / OfImage / OfToolUse /
// OfToolResult / etc.) — we inspect each for text-bearing content.
func blockParamText(blk anthropic.ContentBlockParamUnion) string {
	if t := blk.OfText; t != nil {
		return t.Text
	}
	if t := blk.OfToolResult; t != nil {
		var sb strings.Builder
		for _, c := range t.Content {
			if c.OfText != nil {
				sb.WriteString(c.OfText.Text)
			}
		}
		if sb.Len() > 0 {
			return sb.String()
		}
		return "[TOOL_RESULT]"
	}
	if blk.OfImage != nil {
		return "[IMAGE]"
	}
	if blk.OfDocument != nil {
		return "[DOCUMENT]"
	}
	if blk.OfToolUse != nil {
		return "[TOOL_USE]"
	}
	if blk.OfThinking != nil {
		return ""
	}
	return ""
}

// fillChatOutput maps the Message response content blocks into the step.
// Anthropic 의 Content 는 ContentBlockUnion 배열 — text / thinking /
// tool_use 등 여러 variant. Type 필드로 분기.
func fillChatOutput(step *llm.Step, resp *anthropic.Message) {
	if resp == nil {
		return
	}
	var (
		contentBuilder strings.Builder
		reasonBuilder  strings.Builder
		toolBlocks     []toolUseBlock
	)
	for _, blk := range resp.Content {
		switch blk.Type {
		case "text":
			if blk.Text != "" {
				if contentBuilder.Len() > 0 {
					contentBuilder.WriteByte('\n')
				}
				contentBuilder.WriteString(blk.Text)
			}
		case "thinking":
			if blk.Thinking != "" {
				if reasonBuilder.Len() > 0 {
					reasonBuilder.WriteByte('\n')
				}
				reasonBuilder.WriteString(blk.Thinking)
			}
		case "tool_use":
			toolBlocks = append(toolBlocks, toolUseBlock{
				ID:        blk.ID,
				Name:      blk.Name,
				Arguments: string(blk.Input),
			})
		default:
			// 알 수 없는 / placeholder block — `[<TYPE>]` 로 표시
			if contentBuilder.Len() > 0 {
				contentBuilder.WriteByte('\n')
			}
			contentBuilder.WriteString("[")
			contentBuilder.WriteString(strings.ToUpper(blk.Type))
			contentBuilder.WriteString("]")
		}
	}
	if contentBuilder.Len() > 0 {
		step.AddOutputMessage(contentBuilder.String())
	}
	if reasonBuilder.Len() > 0 {
		step.AddReasoning(reasonBuilder.String())
	}
	if len(toolBlocks) > 0 {
		if payload := marshalToolBlocks(toolBlocks); payload != "" {
			step.AddTool(payload, "", "")
		}
	}
	if r := string(resp.StopReason); r != "" {
		step.SetFinishReason(r)
	}
	step.SetTokens(usageToTokens(resp.Usage))
}

// toolUseBlock — internal collector for response tool_use blocks (used by
// fillChatOutput + stream finalize).
type toolUseBlock struct {
	ID        string
	Name      string
	Arguments string // raw JSON
}

// marshalToolBlocks renders tool_use blocks as a JSON array compatible
// with the AddTool payload format (python-apm tool_calls_text equivalent).
func marshalToolBlocks(blocks []toolUseBlock) string {
	if len(blocks) == 0 {
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
	out := make([]entry, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, entry{
			ID:   b.ID,
			Type: "function",
			Function: fn{
				Name:      b.Name,
				Arguments: b.Arguments,
			},
		})
	}
	b, err := json.Marshal(out)
	if err != nil {
		return ""
	}
	return string(b)
}

// usageToTokens maps Anthropic Usage (4 cache-aware fields + standard
// input/output) into llm.Tokens. CacheCreationInputTokens / CacheReadInput
// Tokens are emitted separately so the WhaTap UI shows the cache split.
func usageToTokens(u anthropic.Usage) llm.Tokens {
	return llm.Tokens{
		Input:                  u.InputTokens,
		Output:                 u.OutputTokens,
		Total:                  u.InputTokens + u.OutputTokens,
		CacheCreationInput:     u.CacheCreationInputTokens,
		CacheReadInput:         u.CacheReadInputTokens,
	}
}
