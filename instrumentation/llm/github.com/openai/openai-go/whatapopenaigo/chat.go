package whatapopenaigo

import (
	"context"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/whatap/go-api/llm"
)

// WrapAndNewChatCompletion wraps the synchronous /v1/chat/completions call.
// A WhaTap LLM step is opened before the call and closed after the response
// (or error) is received. The wrap target is the ChatCompletionService value
// (typically `client.Chat.Completions`) — the OpenAI Go SDK exposes the chat
// completions API as a nested service field on the Client, so wrapping at
// the service level matches the SDK shape directly (§253 §2.5 / §255).
//
// Auto-inject rule rewrites `client.Chat.Completions.New(ctx, params)` to
// `whatapopenaigo.WrapAndNewChatCompletion(ctx, client.Chat.Completions, params)`.
func WrapAndNewChatCompletion(
	ctx context.Context,
	s openai.ChatCompletionService,
	params openai.ChatCompletionNewParams,
	opts ...option.RequestOption,
) (*openai.ChatCompletion, error) {
	ctx, step := llm.Start(ctx, chatConfig(params))
	defer step.End()

	fillChatInputs(step, params)
	if params.Temperature.Valid() {
		step.SetTemperature(params.Temperature.Value)
	}

	resp, err := s.New(ctx, params, opts...)
	if err != nil {
		step.SetError(err, llm.ErrorTypeAPI)
		return resp, err
	}
	fillChatOutput(step, resp)
	return resp, nil
}
