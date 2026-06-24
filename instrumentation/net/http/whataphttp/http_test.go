package whataphttp

import (
	"strings"
	"testing"

	"github.com/whatap/go-api/trace"
)

func TestMain(m *testing.M) {
	// Initialize trace to ensure trace.DISABLE() returns false
	// This simulates real-world usage where whatap agent is running
	trace.Init(nil)
	defer trace.Shutdown()
	m.Run()
}

// TestHttpGetNilContext verifies that HttpGet handles nil context gracefully.
// This is a regression test for §83 - go-api-inst may pass nil when no handler context is available.
func TestHttpGetNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("HttpGet(nil, ...) panicked: %v", r)
		}
	}()

	// Should not panic or return "nil Context" error even with nil context
	// Connection error is expected (invalid URL), but no "nil Context" error
	_, err := HttpGet(nil, "http://localhost:1/test")
	if err != nil && strings.Contains(err.Error(), "nil Context") {
		t.Errorf("HttpGet(nil, ...) returned nil Context error: %v", err)
	}
}

// TestHttpPostNilContext verifies that HttpPost handles nil context gracefully.
func TestHttpPostNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("HttpPost(nil, ...) panicked: %v", r)
		}
	}()

	_, err := HttpPost(nil, "http://localhost:1/test", "application/json", strings.NewReader("{}"))
	if err != nil && strings.Contains(err.Error(), "nil Context") {
		t.Errorf("HttpPost(nil, ...) returned nil Context error: %v", err)
	}
}

// TestHttpPostFormNilContext verifies that HttpPostForm handles nil context gracefully.
func TestHttpPostFormNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("HttpPostForm(nil, ...) panicked: %v", r)
		}
	}()

	_, err := HttpPostForm(nil, "http://localhost:1/test", nil)
	if err != nil && strings.Contains(err.Error(), "nil Context") {
		t.Errorf("HttpPostForm(nil, ...) returned nil Context error: %v", err)
	}
}

// TestDefaultClientGetNilContext verifies that DefaultClientGet handles nil context gracefully.
func TestDefaultClientGetNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("DefaultClientGet(nil, ...) panicked: %v", r)
		}
	}()

	_, err := DefaultClientGet(nil, "http://localhost:1/test")
	if err != nil && strings.Contains(err.Error(), "nil Context") {
		t.Errorf("DefaultClientGet(nil, ...) returned nil Context error: %v", err)
	}
}

// TestDefaultClientPostNilContext verifies that DefaultClientPost handles nil context gracefully.
func TestDefaultClientPostNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("DefaultClientPost(nil, ...) panicked: %v", r)
		}
	}()

	_, err := DefaultClientPost(nil, "http://localhost:1/test", "application/json", strings.NewReader("{}"))
	if err != nil && strings.Contains(err.Error(), "nil Context") {
		t.Errorf("DefaultClientPost(nil, ...) returned nil Context error: %v", err)
	}
}

// TestDefaultClientPostFormNilContext verifies that DefaultClientPostForm handles nil context gracefully.
func TestDefaultClientPostFormNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("DefaultClientPostForm(nil, ...) panicked: %v", r)
		}
	}()

	_, err := DefaultClientPostForm(nil, "http://localhost:1/test", nil)
	if err != nil && strings.Contains(err.Error(), "nil Context") {
		t.Errorf("DefaultClientPostForm(nil, ...) returned nil Context error: %v", err)
	}
}

// §254 Step 5 — NewLLMRoundTrip wrapper carries llmMark flag.
//
// Verifies the wrapper struct has llmMark=true when constructed via
// NewLLMRoundTrip and llmMark=false via NewRoundTrip. The runtime behaviour
// (Driver="LLM API" on the HTTPC step) is exercised in the Docker
// integration test (testapps/basic/sashabaranov-app /chat /chat-stream
// /embed) since it requires a full agent + trace context.
func TestNewLLMRoundTrip_SetsLLMMark(t *testing.T) {
	wrap := NewLLMRoundTrip(nil, nil)
	w, ok := wrap.(*WrapRoundTrip)
	if !ok {
		t.Fatalf("NewLLMRoundTrip must return *WrapRoundTrip, got %T", wrap)
	}
	if !w.llmMark {
		t.Fatalf("NewLLMRoundTrip must set llmMark=true")
	}
}

func TestNewRoundTrip_DefaultsToNonLLM(t *testing.T) {
	wrap := NewRoundTrip(nil, nil)
	w, ok := wrap.(*WrapRoundTrip)
	if !ok {
		t.Fatalf("NewRoundTrip must return *WrapRoundTrip, got %T", wrap)
	}
	if w.llmMark {
		t.Fatalf("NewRoundTrip must NOT set llmMark (general HTTP path)")
	}
}
