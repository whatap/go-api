package whatapanthropic

import (
	"context"
	"net/http"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	whataphttp "github.com/whatap/go-api/instrumentation/net/http/whataphttp"
)

// §253 auto-inject constructor helper. The wrap helpers for Messages.New /
// NewStreaming live next to the wrapped behaviour in messages.go /
// stream.go — auto.go is only responsible for the constructor side
// (installing a wrapped RoundTripper on the SDK Client).
//
// The Anthropic SDK exposes the messages API as a service field on
// anthropic.Client (`client.Messages`), so the wrap helpers take a
// MessageService value directly — no Client wrapper struct or rebinding
// is needed (§253 §2.5).

// NewClient mirrors anthropic.NewClient but injects a wrapped HTTPClient
// (transport replaced with whataphttp.NewLLMRoundTrip) ahead of the
// supplied options. The wrap option is prepended; subsequent user options
// (e.g. option.WithHTTPClient(my client)) override it (Anthropic SDK
// applies options in order — last write wins).
//
// Auto-inject rule rewrites `anthropic.NewClient(opts...)` to
// `whatapanthropic.NewClient(opts...)`. Returned anthropic.Client has the
// same type as the original — user variable declarations stay unchanged.
func NewClient(opts ...option.RequestOption) anthropic.Client {
	wrap := option.WithHTTPClient(wrappedHTTPClient())
	all := make([]option.RequestOption, 0, len(opts)+1)
	all = append(all, wrap)
	all = append(all, opts...)
	return anthropic.NewClient(all...)
}

// wrappedHTTPClient builds a fresh *http.Client whose Transport runs through
// whataphttp.NewLLMRoundTrip on top of http.DefaultTransport. Always marks
// the request as an LLM call (§254 Step 5(1) host-fallback).
func wrappedHTTPClient() *http.Client {
	return &http.Client{Transport: whataphttp.NewLLMRoundTrip(context.Background(), http.DefaultTransport)}
}
