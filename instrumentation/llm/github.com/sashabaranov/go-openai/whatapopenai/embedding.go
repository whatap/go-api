package whatapopenai

import (
	"context"

	openai "github.com/sashabaranov/go-openai"
	"github.com/whatap/go-api/llm"
)

// CreateEmbeddings wraps the /v1/embeddings endpoint. The single-string and
// string-array request variants are both routed through here by the SDK, so
// one wrapper handles both.
func (w *Client) CreateEmbeddings(
	ctx context.Context,
	req openai.EmbeddingRequestConverter,
) (openai.EmbeddingResponse, error) {
	conv := req.Convert()
	ctx, step := llm.Start(ctx, embeddingConfig(string(conv.Model)))
	defer step.End()

	resp, err := w.Client.CreateEmbeddings(ctx, req)
	if err != nil {
		step.SetError(err, llm.ErrorTypeAPI)
		return resp, err
	}

	tokens := usageToTokens(resp.Usage)
	tokens.EmbeddingCount = int64(len(resp.Data))
	if len(resp.Data) > 0 {
		tokens.Dimensions = int64(len(resp.Data[0].Embedding))
	}
	step.SetTokens(tokens)
	return resp, nil
}
