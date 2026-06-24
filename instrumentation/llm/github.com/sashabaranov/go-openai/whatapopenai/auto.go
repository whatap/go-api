package whatapopenai

import (
	"context"
	"net/http"

	openai "github.com/sashabaranov/go-openai"
	whataphttp "github.com/whatap/go-api/instrumentation/net/http/whataphttp"
)

// §254 auto-inject helpers — used by the generated Transform template so the
// user-facing variable type stays *openai.Client (no variable rebinding).
//
// Each helper wraps the inner client just for the single call. The wrapper
// is a thin struct literal (one allocation) and forwards to the inner
// client through the embedded *openai.Client, so unwrapped methods (image /
// audio / files / fine-tune / ...) are not affected.

// NewClient mirrors openai.NewClient but injects a wrapped RoundTripper
// into the underlying HTTPClient so httpc.Start captures the real call URL
// (§267 RoundTrip single-entry-point design).
//
// Auto-inject rule rewrites `openai.NewClient(key)` to
// `whatapopenai.NewClient(key)`. The returned *openai.Client has the same
// type as the original, so user variable declarations stay unchanged.
func NewClient(authToken string) *openai.Client {
	cfg := openai.DefaultConfig(authToken)
	cfg.HTTPClient = wrappedHTTPClient()
	return openai.NewClientWithConfig(cfg)
}

// NewClientFromConfig mirrors openai.NewClientWithConfig but injects a
// wrapped RoundTripper into cfg.HTTPClient. Behaviour by cfg.HTTPClient
// shape:
//   - nil → fresh wrapped http.Client
//   - *http.Client with non-wrapped Transport → Transport replaced
//   - *http.Client whose Transport is already a *WrapRoundTrip → unchanged
//   - non-*http.Client HTTPDoer (custom user implementation) → unchanged
//     (caller takes responsibility — wrapping a custom HTTPDoer is unsafe)
//
// Auto-inject rule rewrites `openai.NewClientWithConfig(cfg)` to
// `whatapopenai.NewClientFromConfig(cfg)`.
func NewClientFromConfig(cfg openai.ClientConfig) *openai.Client {
	cfg.HTTPClient = ensureWrappedDoer(cfg.HTTPClient)
	return openai.NewClientWithConfig(cfg)
}

// wrappedHTTPClient builds a fresh *http.Client whose Transport runs through
// whataphttp.NewRoundTrip on top of http.DefaultTransport.
func wrappedHTTPClient() *http.Client {
	return &http.Client{Transport: whataphttp.NewLLMRoundTrip(context.Background(), http.DefaultTransport)}
}

// ensureWrappedDoer ensures the supplied HTTPDoer (sashabaranov's
// ClientConfig.HTTPClient interface) is a *http.Client whose Transport is
// wrapped by whataphttp.NewRoundTrip. A custom HTTPDoer that is not a
// *http.Client is left untouched — wrapping it would require an additional
// interface adapter and risks breaking user semantics.
func ensureWrappedDoer(doer openai.HTTPDoer) openai.HTTPDoer {
	if doer == nil {
		return wrappedHTTPClient()
	}
	hc, ok := doer.(*http.Client)
	if !ok {
		// custom HTTPDoer — user's responsibility to wrap if needed.
		return doer
	}
	if _, alreadyWrapped := hc.Transport.(*whataphttp.WrapRoundTrip); alreadyWrapped {
		return hc
	}
	hc.Transport = whataphttp.NewLLMRoundTrip(context.Background(), hc.Transport)
	return hc
}

// WrapAndCreateChatCompletion mirrors (*whatapopenai.Client).CreateChatCompletion.
// Auto-inject rule rewrites `c.CreateChatCompletion(ctx, req)` to
// `whatapopenai.WrapAndCreateChatCompletion(ctx, c, req)`.
func WrapAndCreateChatCompletion(
	ctx context.Context,
	c *openai.Client,
	req openai.ChatCompletionRequest,
) (openai.ChatCompletionResponse, error) {
	if c == nil {
		return openai.ChatCompletionResponse{}, nil
	}
	return (&Client{Client: c}).CreateChatCompletion(ctx, req)
}

// WrapAndCreateChatCompletionStream mirrors (*whatapopenai.Client).CreateChatCompletionStream.
func WrapAndCreateChatCompletionStream(
	ctx context.Context,
	c *openai.Client,
	req openai.ChatCompletionRequest,
) (*ChatCompletionStream, error) {
	if c == nil {
		return nil, nil
	}
	return (&Client{Client: c}).CreateChatCompletionStream(ctx, req)
}

// WrapAndCreateCompletion mirrors (*whatapopenai.Client).CreateCompletion.
func WrapAndCreateCompletion(
	ctx context.Context,
	c *openai.Client,
	req openai.CompletionRequest,
) (openai.CompletionResponse, error) {
	if c == nil {
		return openai.CompletionResponse{}, nil
	}
	return (&Client{Client: c}).CreateCompletion(ctx, req)
}

// WrapAndCreateEmbeddings mirrors (*whatapopenai.Client).CreateEmbeddings.
func WrapAndCreateEmbeddings(
	ctx context.Context,
	c *openai.Client,
	req openai.EmbeddingRequestConverter,
) (openai.EmbeddingResponse, error) {
	if c == nil {
		return openai.EmbeddingResponse{}, nil
	}
	return (&Client{Client: c}).CreateEmbeddings(ctx, req)
}
