package whatapeino

import (
	"net/http"
	"testing"

	whataphttp "github.com/whatap/go-api/instrumentation/net/http/whataphttp"
)

func TestEnsureWrappedHTTPClient_NilGivesFreshWrapped(t *testing.T) {
	hc := ensureWrappedHTTPClient(nil)
	if hc == nil {
		t.Fatalf("ensureWrappedHTTPClient(nil) returned nil")
	}
	if _, ok := hc.Transport.(*whataphttp.WrapRoundTrip); !ok {
		t.Errorf("Transport should be *WrapRoundTrip, got %T", hc.Transport)
	}
}

func TestEnsureWrappedHTTPClient_AlreadyWrappedIsIdempotent(t *testing.T) {
	rt := whataphttp.NewRoundTrip(nil, http.DefaultTransport)
	orig := &http.Client{Transport: rt}
	got := ensureWrappedHTTPClient(orig)
	if got != orig {
		t.Errorf("already-wrapped client should be returned unchanged (no clone)")
	}
}

func TestEnsureWrappedHTTPClient_PreservesOriginalClientFields(t *testing.T) {
	// Caller's *http.Client may be reused across multiple ChatModels;
	// the helper must not mutate it.
	orig := &http.Client{
		Transport: http.DefaultTransport,
		Timeout:   30 * 1_000_000_000, // 30s
	}
	got := ensureWrappedHTTPClient(orig)
	if got == orig {
		t.Errorf("non-wrapped original should be cloned (got identity match)")
	}
	if _, ok := got.Transport.(*whataphttp.WrapRoundTrip); !ok {
		t.Errorf("clone Transport not wrapped: %T", got.Transport)
	}
	if got.Timeout != orig.Timeout {
		t.Errorf("clone Timeout: want %v, got %v", orig.Timeout, got.Timeout)
	}
	// Original must remain untouched.
	if _, wrapped := orig.Transport.(*whataphttp.WrapRoundTrip); wrapped {
		t.Errorf("original Transport must NOT be mutated, but it is wrapped")
	}
}
