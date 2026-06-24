package whatapopenai

import (
	openai "github.com/sashabaranov/go-openai"
)

// Client wraps *openai.Client. Methods we instrument are overridden on this
// type; everything else (image / audio / moderation / file / fine-tune / ...)
// forwards transparently through the embedded *openai.Client.
//
// Construct via WrapClient. The wrapper stores the same *openai.Client that
// the caller created — passing the same client to multiple WrapClient calls
// is a usage error (would double-track every call).
type Client struct {
	*openai.Client
}

// WrapClient returns a Client that emits WhaTap LLM monitoring data for the
// instrumented methods (CreateChatCompletion / CreateChatCompletionStream /
// CreateCompletion / CreateCompletionStream / CreateEmbeddings).
//
// Pass nil to get nil back (for use in conditional construction).
func WrapClient(inner *openai.Client) *Client {
	if inner == nil {
		return nil
	}
	return &Client{Client: inner}
}
