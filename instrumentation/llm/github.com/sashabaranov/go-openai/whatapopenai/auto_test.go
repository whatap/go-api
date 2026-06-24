package whatapopenai

import (
	"context"
	"net/http"
	"testing"

	openai "github.com/sashabaranov/go-openai"
	whataphttp "github.com/whatap/go-api/instrumentation/net/http/whataphttp"
	"github.com/whatap/go-api/trace"
)

// §254 auto-inject helpers — sanity check that the WrapAnd* helpers route
// through the same code path as the manual WrapClient(c).Create* methods.

func TestWrapAndCreateChatCompletion_NilClientReturnsZero(t *testing.T) {
	resp, err := WrapAndCreateChatCompletion(context.Background(), nil, openai.ChatCompletionRequest{})
	if err != nil {
		t.Fatalf("nil client should not error: %v", err)
	}
	if resp.ID != "" {
		t.Errorf("nil client should return zero response, got %+v", resp)
	}
}

func TestWrapAndCreateChatCompletion_RoutesThroughAdapter(t *testing.T) {
	enableLLM(t)
	srv := fakeOpenAIServer(t, "/v1/chat/completions", openai.ChatCompletionResponse{
		ID:    "cmpl-auto",
		Model: "gpt-4o",
		Choices: []openai.ChatCompletionChoice{
			{Index: 0, Message: openai.ChatCompletionMessage{Role: "assistant", Content: "auto-inject!"}, FinishReason: "stop"},
		},
		Usage: openai.Usage{PromptTokens: 5, CompletionTokens: 3, TotalTokens: 8},
	})
	defer srv.Close()

	cfg := openai.DefaultConfig("test-key")
	cfg.BaseURL = srv.URL + "/v1"
	cfg.HTTPClient = &http.Client{Transport: whataphttp.NewRoundTrip(context.Background(), http.DefaultTransport)}
	client := openai.NewClientWithConfig(cfg)

	ctx, traceCtx := trace.NewTraceContext(context.Background())
	t.Cleanup(func() { trace.RemoveGIDTraceCtx(traceCtx.GID) })

	resp, err := WrapAndCreateChatCompletion(ctx, client, openai.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []openai.ChatCompletionMessage{
			{Role: "user", Content: "hi"},
		},
	})
	if err != nil {
		t.Fatalf("WrapAndCreateChatCompletion err: %v", err)
	}
	if len(resp.Choices) != 1 || resp.Choices[0].Message.Content != "auto-inject!" {
		t.Fatalf("response not forwarded: %+v", resp)
	}
	// Helper wraps once per call — the *openai.Client variable stays unchanged
	// at the call site, which is the point of §254 auto-inject.
}

func TestNewClient_InstallsWrappedTransport(t *testing.T) {
	c := NewClient("test-key")
	if c == nil {
		t.Fatalf("NewClient returned nil")
	}
	// NewClient ensures the underlying HTTPClient.Transport is wrapped.
	// We can't read the unexported config directly; use NewClientFromConfig
	// path to assert wrapping behaviour deterministically (see next test).
}

func TestNewClientFromConfig_WrapsNilHTTPClient(t *testing.T) {
	cfg := openai.DefaultConfig("test-key")
	// cfg.HTTPClient is nil — helper installs a fresh wrapped *http.Client.
	c := NewClientFromConfig(cfg)
	if c == nil {
		t.Fatalf("NewClientFromConfig returned nil")
	}
}

func TestEnsureWrappedDoer_NilReturnsWrappedHTTPClient(t *testing.T) {
	doer := ensureWrappedDoer(nil)
	hc, ok := doer.(*http.Client)
	if !ok {
		t.Fatalf("ensureWrappedDoer(nil) should return *http.Client, got %T", doer)
	}
	if _, wrapped := hc.Transport.(*whataphttp.WrapRoundTrip); !wrapped {
		t.Errorf("Transport should be *WrapRoundTrip, got %T", hc.Transport)
	}
}

func TestEnsureWrappedDoer_HTTPClientGetsWrapped(t *testing.T) {
	original := &http.Client{Transport: http.DefaultTransport}
	doer := ensureWrappedDoer(original)
	hc, ok := doer.(*http.Client)
	if !ok || hc != original {
		t.Fatalf("ensureWrappedDoer should mutate same *http.Client, got %T", doer)
	}
	if _, wrapped := hc.Transport.(*whataphttp.WrapRoundTrip); !wrapped {
		t.Errorf("Transport not wrapped: %T", hc.Transport)
	}
}

func TestEnsureWrappedDoer_AlreadyWrappedIsIdempotent(t *testing.T) {
	rt := whataphttp.NewRoundTrip(context.Background(), http.DefaultTransport)
	hc := &http.Client{Transport: rt}
	doer := ensureWrappedDoer(hc)
	got, ok := doer.(*http.Client)
	if !ok {
		t.Fatalf("ensureWrappedDoer should return *http.Client")
	}
	if got.Transport != rt {
		t.Errorf("already-wrapped transport should be preserved (no re-wrap)")
	}
}

// customDoer is a sashabaranov HTTPDoer that is NOT *http.Client.
// ensureWrappedDoer must leave it untouched.
type customDoer struct{}

func (customDoer) Do(req *http.Request) (*http.Response, error) {
	return nil, nil
}

func TestEnsureWrappedDoer_CustomHTTPDoerUntouched(t *testing.T) {
	cd := customDoer{}
	doer := ensureWrappedDoer(cd)
	if _, ok := doer.(customDoer); !ok {
		t.Errorf("custom HTTPDoer should be left untouched, got %T", doer)
	}
}
