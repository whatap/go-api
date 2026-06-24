package whatapopenaigo

import (
	"context"
	"net/http"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	whataphttp "github.com/whatap/go-api/instrumentation/net/http/whataphttp"
)

// §255 auto-inject constructor helper. The wrap helpers for
// Chat.Completions.New / NewStreaming live next to the wrapped behaviour
// in chat.go / stream.go — auto.go is only responsible for the constructor
// side (installing a wrapped RoundTripper on the SDK Client).
//
// The OpenAI Go SDK exposes the chat completions API as a nested service
// field (`client.Chat.Completions`), so the wrap helpers take a
// ChatCompletionService value directly — no Client wrapper struct or
// rebinding is needed (§253 §2.5 / §255).

// NewClient mirrors openai.NewClient but injects a wrapped HTTPClient
// (transport replaced with whataphttp.NewLLMRoundTrip) ahead of the
// supplied options. The wrap option is prepended; subsequent user options
// (e.g. option.WithHTTPClient(my client)) override it (OpenAI Go SDK
// applies options in order — last write wins).
//
// Auto-inject rule rewrites `openai.NewClient(opts...)` to
// `whatapopenaigo.NewClient(opts...)`. Returned openai.Client has the same
// type as the original — user variable declarations stay unchanged.
func NewClient(opts ...option.RequestOption) openai.Client {
	wrap := option.WithHTTPClient(wrappedHTTPClient())
	all := make([]option.RequestOption, 0, len(opts)+1)
	all = append(all, wrap)
	all = append(all, opts...)
	return openai.NewClient(all...)
}

// wrappedHTTPClient builds a fresh *http.Client whose Transport runs through
// whataphttp.NewLLMRoundTrip on top of http.DefaultTransport. Always marks
// the request as an LLM call (§254 Step 5(1) host-fallback).
func wrappedHTTPClient() *http.Client {
	return &http.Client{Transport: whataphttp.NewLLMRoundTrip(context.Background(), http.DefaultTransport)}
}
