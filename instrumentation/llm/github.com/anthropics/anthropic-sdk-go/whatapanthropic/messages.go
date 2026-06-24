package whatapanthropic

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/whatap/go-api/llm"
)

// WrapAndNewMessage wraps the synchronous /v1/messages call. A WhaTap LLM
// step is opened before the call and closed after the response (or error)
// is received. The wrap target is the MessageService value (typically
// `client.Messages`) — Anthropic's SDK exposes the messages API as a
// service field on the Client, so wrapping at the service level matches
// the SDK shape directly (§253 §2.5).
//
// Auto-inject rule rewrites `client.Messages.New(ctx, params)` to
// `whatapanthropic.WrapAndNewMessage(ctx, client.Messages, params)`.
func WrapAndNewMessage(
	ctx context.Context,
	m anthropic.MessageService,
	params anthropic.MessageNewParams,
	opts ...option.RequestOption,
) (*anthropic.Message, error) {
	ctx, step := llm.Start(ctx, chatConfig(params))
	defer step.End()

	fillChatInputs(step, params)
	if params.Temperature.Valid() {
		step.SetTemperature(params.Temperature.Value)
	}

	resp, err := m.New(ctx, params, opts...)
	if err != nil {
		step.SetError(err, llm.ErrorTypeAPI)
		return resp, err
	}
	fillChatOutput(step, resp)
	return resp, nil
}
