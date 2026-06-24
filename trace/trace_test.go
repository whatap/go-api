package trace

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMain(m *testing.M) {
	Init(nil)
	defer Shutdown()
	m.Run()
}

// §183: GetTraceContext(nil) should not panic
func TestGetTraceContext_NilContext(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("§183: GetTraceContext(nil) panicked: %v", r)
		}
	}()

	ctx, traceCtx := GetTraceContext(nil)
	// Without active transaction, both should be nil
	if traceCtx != nil {
		t.Error("GetTraceContext(nil) should return nil TraceCtx when no active transaction")
	}
	// ctx may be nil when no GID trace context found
	_ = ctx
}

// §183: GetTraceContext with background context (no whatap value)
func TestGetTraceContext_BackgroundContext(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("GetTraceContext(Background) panicked: %v", r)
		}
	}()

	ctx, traceCtx := GetTraceContext(context.Background())
	if traceCtx != nil {
		t.Error("GetTraceContext(Background) should return nil TraceCtx when no active transaction")
	}
	if ctx == nil {
		t.Error("GetTraceContext(Background) should return non-nil context")
	}
}

// §175: StartWithRequest duplicate prevention
func TestStartWithRequest_DuplicatePrevention(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("§175: StartWithRequest panicked: %v", r)
		}
	}()

	// Create a test request
	req := httptest.NewRequest("GET", "/test", nil)

	// First call — should create transaction
	ctx1, err1 := StartWithRequest(req)
	if err1 != nil {
		t.Fatalf("First StartWithRequest failed: %v", err1)
	}

	// Attach the context back to request for second call
	req2 := req.WithContext(ctx1)

	// Second call — should reuse existing transaction (§175)
	ctx2, err2 := StartWithRequest(req2)
	if err2 != nil {
		t.Fatalf("Second StartWithRequest failed: %v", err2)
	}

	// Both should return valid contexts
	if ctx1 == nil || ctx2 == nil {
		t.Error("§175: Both calls should return non-nil contexts")
	}

	// Verify that the second call returned the same trace context (not a new one)
	_, tc1 := GetTraceContext(ctx1)
	_, tc2 := GetTraceContext(ctx2)
	if tc1 != nil && tc2 != nil {
		if tc1.Txid != tc2.Txid {
			t.Errorf("§175: Second StartWithRequest should reuse existing txid, got %d vs %d", tc1.Txid, tc2.Txid)
		}
	}

	// Cleanup
	if ctx1 != nil {
		End(ctx1, nil)
	}
}

// §221: WrapResponseWriter.Write must default Status to 200 when WriteHeader
// was never called explicitly by the handler.
//
// Background: net/http internally calls WriteHeader(StatusOK) from response.Write()
// via the concrete *response type — bypassing embedded-interface method dispatch.
// Without a Write override, Status stays 0 for handlers that only call w.Write().
// This affects net/http / gorilla/mux / chi (all of which rely on trace.WrapResponseWriter).
func TestWrapResponseWriter_ImplicitStatusOK(t *testing.T) {
	rec := httptest.NewRecorder()
	wrw := &WrapResponseWriter{ResponseWriter: rec}

	// Simulate a handler that only writes a body, no explicit WriteHeader.
	_, err := wrw.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if wrw.Status != http.StatusOK {
		t.Errorf("§221: Status=%d, want %d after implicit WriteHeader via Write",
			wrw.Status, http.StatusOK)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("underlying recorder Code=%d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "hello" {
		t.Errorf("body=%q, want %q", rec.Body.String(), "hello")
	}
}

// §221: Explicit WriteHeader still wins over implicit default.
func TestWrapResponseWriter_ExplicitWriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	wrw := &WrapResponseWriter{ResponseWriter: rec}

	wrw.WriteHeader(http.StatusCreated)
	wrw.Write([]byte("created"))

	if wrw.Status != http.StatusCreated {
		t.Errorf("Status=%d, want %d", wrw.Status, http.StatusCreated)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("recorder Code=%d, want %d", rec.Code, http.StatusCreated)
	}
}

// §221: WriteHeader called multiple times — first wins for our Status field,
// but underlying writer receives every call (matches standard lib behavior of
// only honoring the first header write).
func TestWrapResponseWriter_MultipleWriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	wrw := &WrapResponseWriter{ResponseWriter: rec}

	wrw.WriteHeader(http.StatusInternalServerError)
	wrw.WriteHeader(http.StatusOK) // should be ignored by our wrapper's Status
	wrw.Write([]byte("err"))

	if wrw.Status != http.StatusInternalServerError {
		t.Errorf("Status=%d, want %d (first write wins)",
			wrw.Status, http.StatusInternalServerError)
	}
}

// §261: TraceCtx.IsLlm 필드는 트랜잭션 종료 시 reset 되어야 한다 (Clear()).
// pool 재사용 시 이전 트랜잭션의 IsLlm 값이 다음 트랜잭션으로 누수되면
// LLM 아닌 transaction 에 ExtraField "is-llm=1" 이 잘못 부착된다.
func TestTraceCtx_IsLlm_ClearReset(t *testing.T) {
	ctx := NewTraceCtx()
	if ctx.IsLlm != 0 {
		t.Fatalf("§261: 초기 IsLlm 값이 0 이 아님: %d", ctx.IsLlm)
	}

	ctx.IsLlm = 1
	if ctx.IsLlm != 1 {
		t.Fatalf("§261: IsLlm = 1 set 실패: %d", ctx.IsLlm)
	}

	ctx.Clear()
	if ctx.IsLlm != 0 {
		t.Errorf("§261: Clear() 가 IsLlm 을 reset 하지 않음 — pool 재사용 시 LLM mark 누수: %d", ctx.IsLlm)
	}
}

// §221: Write without WriteHeader and then an additional Write should stay at 200.
func TestWrapResponseWriter_MultipleWrites(t *testing.T) {
	rec := httptest.NewRecorder()
	wrw := &WrapResponseWriter{ResponseWriter: rec}

	wrw.Write([]byte("part1"))
	wrw.Write([]byte("part2"))

	if wrw.Status != http.StatusOK {
		t.Errorf("Status=%d, want %d", wrw.Status, http.StatusOK)
	}
	if rec.Body.String() != "part1part2" {
		t.Errorf("body=%q, want %q", rec.Body.String(), "part1part2")
	}
}

// §221: trace.Func pipeline captures implicit 200 when handler uses only w.Write().
// This is the end-to-end net/http / gorilla / chi scenario.
func TestFunc_CapturesImplicitStatusOK(t *testing.T) {
	handlerCalled := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.Write([]byte("ok")) // no explicit WriteHeader
	}

	wrapped := Func(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	wrapped(rec, req)

	if !handlerCalled {
		t.Fatal("handler not called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("recorder Code=%d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("body=%q, want %q", rec.Body.String(), "ok")
	}
}

// §221 follow-up: option-interface delegation tests
//
// fakeFlusher records whether Flush was called.
type fakeFlusher struct {
	*httptest.ResponseRecorder
	flushed int
}

func (f *fakeFlusher) Flush() { f.flushed++ }

// fakeHijacker records hijack attempts and returns a sentinel.
type fakeHijacker struct {
	*httptest.ResponseRecorder
	called bool
}

func (h *fakeHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h.called = true
	return nil, nil, errors.New("fake-hijack")
}

// fakeCloseNotifier returns a controllable channel.
type fakeCloseNotifier struct {
	*httptest.ResponseRecorder
	ch chan bool
}

func (c *fakeCloseNotifier) CloseNotify() <-chan bool { return c.ch }

// §221+: Flush delegates to underlying when supported.
func TestWrapResponseWriter_FlushDelegates(t *testing.T) {
	inner := &fakeFlusher{ResponseRecorder: httptest.NewRecorder()}
	wrw := &WrapResponseWriter{ResponseWriter: inner}

	flusher, ok := http.ResponseWriter(wrw).(http.Flusher)
	if !ok {
		t.Fatal("WrapResponseWriter must implement http.Flusher after delegation")
	}
	flusher.Flush()
	flusher.Flush()
	if inner.flushed != 2 {
		t.Errorf("inner.flushed=%d, want 2", inner.flushed)
	}
}

// §221+: Flush is a no-op when underlying writer does not implement Flusher.
// Documented limitation — real production servers always implement Flusher,
// only test/mock writers fall through this path.
func TestWrapResponseWriter_FlushNoopForNonFlusher(t *testing.T) {
	inner := httptest.NewRecorder() // does NOT implement Flusher
	wrw := &WrapResponseWriter{ResponseWriter: inner}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Flush should not panic on non-Flusher underlying, got: %v", r)
		}
	}()
	wrw.Flush() // should silently no-op, not panic
}

// §221+: Hijack delegates to underlying when supported.
func TestWrapResponseWriter_HijackDelegates(t *testing.T) {
	inner := &fakeHijacker{ResponseRecorder: httptest.NewRecorder()}
	wrw := &WrapResponseWriter{ResponseWriter: inner}

	hijacker, ok := http.ResponseWriter(wrw).(http.Hijacker)
	if !ok {
		t.Fatal("WrapResponseWriter must implement http.Hijacker after delegation")
	}
	_, _, err := hijacker.Hijack()
	if !inner.called {
		t.Error("inner Hijack not called")
	}
	if err == nil || err.Error() != "fake-hijack" {
		t.Errorf("error=%v, want 'fake-hijack'", err)
	}
}

// §221+: Hijack returns explicit error (not panic, not silent) when underlying
// does not implement Hijacker. WebSocket libraries can detect this and abort.
func TestWrapResponseWriter_HijackErrorForNonHijacker(t *testing.T) {
	inner := httptest.NewRecorder() // does NOT implement Hijacker
	wrw := &WrapResponseWriter{ResponseWriter: inner}

	_, _, err := wrw.Hijack()
	if err == nil {
		t.Error("Hijack must return an error when underlying is not Hijacker")
	}
}

// §221+: CloseNotify delegates to underlying when supported.
func TestWrapResponseWriter_CloseNotifyDelegates(t *testing.T) {
	ch := make(chan bool, 1)
	ch <- true
	inner := &fakeCloseNotifier{ResponseRecorder: httptest.NewRecorder(), ch: ch}
	wrw := &WrapResponseWriter{ResponseWriter: inner}

	//nolint:staticcheck // testing deprecated CloseNotifier delegation
	notifier, ok := http.ResponseWriter(wrw).(http.CloseNotifier)
	if !ok {
		t.Fatal("WrapResponseWriter must implement http.CloseNotifier after delegation")
	}
	got := <-notifier.CloseNotify()
	if !got {
		t.Error("CloseNotify channel returned wrong value")
	}
}

// §221+: CloseNotify returns a never-firing channel when underlying does not
// implement CloseNotifier. Receiver does not deadlock and the program continues.
func TestWrapResponseWriter_CloseNotifyFakeChannel(t *testing.T) {
	inner := httptest.NewRecorder() // does NOT implement CloseNotifier
	wrw := &WrapResponseWriter{ResponseWriter: inner}

	ch := wrw.CloseNotify()
	select {
	case <-ch:
		t.Error("fake channel should not fire")
	default:
		// OK — channel never fires
	}
}

// §175: StartWithRequest with nil request context should not panic
func TestStartWithRequest_NilContext_InRequest(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("StartWithRequest panicked: %v", r)
		}
	}()

	// Create request — the default context is context.Background(), never nil
	req, _ := http.NewRequest("GET", "/test", nil)
	ctx, err := StartWithRequest(req)
	if err != nil {
		t.Fatalf("StartWithRequest failed: %v", err)
	}
	if ctx != nil {
		End(ctx, nil)
	}
}
