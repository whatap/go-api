// Package whatapopenaigo — WhaTap LLM monitoring adapter for the official
// OpenAI Go SDK (github.com/openai/openai-go).
//
// The OpenAI SDK exposes the chat completions API as a nested service field
// on openai.Client (`client.Chat.Completions.New(...)` — a 3-step selector
// chain). Because the interesting method lives on *ChatCompletionService
// (not *Client), this adapter wraps at the service level: the user passes
// `client.Chat.Completions` (a ChatCompletionService value) to the helper
// and the helper opens a WhaTap LLM step around the SDK call.
//
// Token counts, finish reason, message content, and streaming TTFT are
// extracted automatically. Streaming uses the SDK's ssestream package to
// accumulate ChatCompletionChunk deltas.
//
// Usage (manual):
//
//	import (
//	    "github.com/openai/openai-go"
//	    "github.com/openai/openai-go/option"
//	    "github.com/whatap/go-api/instrumentation/llm/github.com/openai/openai-go/whatapopenaigo"
//	)
//
//	client := openai.NewClient(option.WithAPIKey(key))
//	resp, err := whatapopenaigo.WrapAndNewChatCompletion(ctx, client.Chat.Completions, openai.ChatCompletionNewParams{
//	    Model:    "gpt-4o",
//	    Messages: []openai.ChatCompletionMessageParamUnion{
//	        openai.UserMessage("hello"),
//	    },
//	})
//
// Usage (auto-inject): the whatap-go-inst rule rewrites
// `client.Chat.Completions.New(ctx, params)` to
// `whatapopenaigo.WrapAndNewChatCompletion(ctx, client.Chat.Completions, params)`
// — same signature, no variable rebinding needed.
//
// See dev-docs/issues/255.md.
package whatapopenaigo
