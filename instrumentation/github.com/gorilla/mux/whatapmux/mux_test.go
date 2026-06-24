package whatapmux

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/whatap/go-api/trace"
)

func TestMain(m *testing.M) {
	trace.Init(nil)
	defer trace.Shutdown()
	m.Run()
}

// §124: Middleware returns func(http.Handler) http.Handler (not mux.MiddlewareFunc)
// This ensures compatibility with gorilla/mux forks like containous/mux
func TestMiddleware_ReturnType(t *testing.T) {
	middleware := Middleware()
	if middleware == nil {
		t.Fatal("§124: Middleware() should not return nil")
	}

	// Verify the middleware wraps a handler correctly
	called := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(nextHandler)
	if wrappedHandler == nil {
		t.Fatal("§124: Middleware()(handler) should not return nil")
	}

	// Test the wrapped handler serves correctly
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if !called {
		t.Error("§124: Next handler should have been called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("§124: Expected status 200, got %d", w.Code)
	}
}

// §124: WrapRouter should not panic with nil router
func TestWrapRouter_NilRouter(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("§124: WrapRouter(nil) panicked: %v", r)
		}
	}()

	result := WrapRouter(nil)
	if result != nil {
		t.Error("§124: WrapRouter(nil) should return nil")
	}
}
