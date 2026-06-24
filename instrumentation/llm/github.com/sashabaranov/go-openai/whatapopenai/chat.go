package whatapopenai

import (
	"context"

	openai "github.com/sashabaranov/go-openai"
	"github.com/whatap/go-api/llm"
)

// CreateChatCompletion wraps the synchronous chat completion call. A WhaTap
// LLM step is opened before the call and closed after the response (or
// error) is received.
func (w *Client) CreateChatCompletion(
	ctx context.Context,
	req openai.ChatCompletionRequest,
) (openai.ChatCompletionResponse, error) {
	ctx, step := llm.Start(ctx, chatConfig(req))
	defer step.End()

	fillChatInputs(step, req.Messages)
	if req.Temperature != 0 {
		step.SetTemperature(float64(req.Temperature))
	}

	resp, err := w.Client.CreateChatCompletion(ctx, req)
	if err != nil {
		step.SetError(err, llm.ErrorTypeAPI)
		return resp, err
	}
	fillChatOutput(step, &resp)
	return resp, nil
}
