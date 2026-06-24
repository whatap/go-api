// Package whatapanthropic — WhaTap LLM monitoring adapter for
// github.com/anthropics/anthropic-sdk-go.
//
// The Anthropic Go SDK exposes the messages API as a service field on
// anthropic.Client (`client.Messages.New(...)`). Because the interesting
// method lives on *MessageService (not *Client), this adapter wraps at the
// service level: the user passes `client.Messages` (a MessageService value)
// to the helper and the helper opens a WhaTap LLM step around the SDK call.
//
// Token counts (including cache creation/read tokens), finish reason,
// content blocks, and streaming TTFT are extracted automatically.
// Streaming uses the SDK's discriminated union event types
// (MessageStartEvent / ContentBlockStartEvent / ContentBlockDeltaEvent /
// ContentBlockStopEvent / MessageDeltaEvent / MessageStopEvent) to
// accumulate text + tool_use deltas (§253 §2.5.1).
//
// Usage (manual):
//
//	import (
//	    "github.com/anthropics/anthropic-sdk-go"
//	    "github.com/anthropics/anthropic-sdk-go/option"
//	    "github.com/whatap/go-api/instrumentation/llm/github.com/anthropics/anthropic-sdk-go/whatapanthropic"
//	)
//
//	client := anthropic.NewClient(option.WithAPIKey(key))
//	resp, err := whatapanthropic.WrapAndNewMessage(ctx, client.Messages, anthropic.MessageNewParams{
//	    Model:     anthropic.ModelClaudeSonnet4_5,
//	    MaxTokens: 1024,
//	    Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock("hello"))},
//	})
//
// Usage (auto-inject): the whatap-go-inst rule rewrites
// `client.Messages.New(ctx, params)` to
// `whatapanthropic.WrapAndNewMessage(ctx, client.Messages, params)` — same
// signature, no variable rebinding needed.
//
// See dev-docs/issues/253.md.
package whatapanthropic
