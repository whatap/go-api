package whatapopenai

import (
	"context"

	openai "github.com/sashabaranov/go-openai"
	"github.com/whatap/go-api/llm"
)

// CreateCompletion wraps the legacy /v1/completions endpoint.
func (w *Client) CreateCompletion(
	ctx context.Context,
	req openai.CompletionRequest,
) (openai.CompletionResponse, error) {
	ctx, step := llm.Start(ctx, completionConfig(req))
	defer step.End()

	if prompt := completionPromptText(req.Prompt); prompt != "" {
		step.AddInputMessage(prompt)
	}
	if req.Temperature != 0 {
		step.SetTemperature(float64(req.Temperature))
	}

	resp, err := w.Client.CreateCompletion(ctx, req)
	if err != nil {
		step.SetError(err, llm.ErrorTypeAPI)
		return resp, err
	}

	if len(resp.Choices) > 0 {
		c := resp.Choices[0]
		if c.Text != "" {
			step.AddOutputMessage(c.Text)
		}
		if c.FinishReason != "" {
			step.SetFinishReason(c.FinishReason)
		}
	}
	if resp.Usage != nil {
		step.SetTokens(usageToTokens(*resp.Usage))
	}
	return resp, nil
}
