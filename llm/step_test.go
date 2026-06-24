package llm

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/whatap/go-api/agent/agent/config"
	agentllm "github.com/whatap/go-api/agent/agent/llm"
	"github.com/whatap/go-api/httpc"
	whataphttp "github.com/whatap/go-api/instrumentation/net/http/whataphttp"
	"github.com/whatap/go-api/trace"
)

func TestMain(m *testing.M) {
	trace.Init(nil)
	defer trace.Shutdown()
	m.Run()
}

func enableLLM(t *testing.T) {
	t.Helper()
	cfg := config.GetConfig()
	prev := cfg.LLMMode
	cfg.LLMMode = true
	t.Cleanup(func() { cfg.LLMMode = prev })
}

func TestStart_DisabledReturnsNoOp(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LLMMode = false
	ctx, step := Start(context.Background(), Config{Model: "gpt-4o"})
	if step.LLMState == nil || !step.LLMState.Disabled() {
		t.Fatalf("disabled mode must return DisabledState")
	}
	if ctx == nil {
		t.Fatalf("Start must return non-nil ctx even when disabled")
	}
	// no-op mutators / End
	step.AddInputMessage("ignored")
	step.SetTokens(Tokens{Input: 100})
	step.End()
}

func TestStart_RegistersPendingOnCtx(t *testing.T) {
	enableLLM(t)
	ctx, step := Start(context.Background(), Config{
		Provider: "openai", Model: "gpt-4o", OperationType: "chat",
	})
	if step.LLMState.Disabled() {
		t.Fatalf("step should be enabled")
	}
	pending := agentllm.TakePending(ctx)
	if pending == nil {
		t.Fatalf("Start must register pending LLMState on returned ctx")
	}
	if pending != step.LLMState {
		t.Errorf("pending state != step.LLMState — should be the same instance")
	}
}

func TestStart_NoHttpcStart(t *testing.T) {
	// §267 核心: llm.Start must NOT call httpc.Start. Verify by checking
	// that no HttpcCtx machinery is touched — TraceCtx LLMTx remains nil
	// because no httpc.End fires.
	enableLLM(t)
	ctx, traceCtx := trace.NewTraceContext(context.Background())
	traceCtx.Txid = 7777
	t.Cleanup(func() { trace.RemoveGIDTraceCtx(traceCtx.GID) })

	_, step := Start(ctx, Config{Provider: "openai", Model: "gpt-4o", OperationType: "chat"})
	step.SetTokens(Tokens{Input: 100, Output: 200})
	step.End()

	if traceCtx.LLMTx != nil {
		t.Errorf("LLMTx should remain nil — llm.Start must not publish via httpc.End")
	}
}

func TestRoundTripIntegration_AttachesPendingAndPublishes(t *testing.T) {
	// §267 흐름 통합: llm.Start (pending) → SDK HTTP 호출 (RoundTrip wrap)
	//   → httpc.Start 가 pending 잡아 attach + state.URL 갱신
	//   → httpc.End 가 LLMTx 누적 발행.
	enableLLM(t)

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer mock.Close()

	ctx, traceCtx := trace.NewTraceContext(context.Background())
	traceCtx.Txid = 4242
	t.Cleanup(func() { trace.RemoveGIDTraceCtx(traceCtx.GID) })

	ctx, step := Start(ctx, Config{Provider: "openai", Model: "gpt-4o", OperationType: "chat"})

	client := &http.Client{Transport: whataphttp.NewRoundTrip(ctx, http.DefaultTransport)}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, mock.URL+"/v1/chat/completions", http.NoBody)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	_ = resp.Body.Close()

	step.SetTokens(Tokens{Input: 100, Output: 200})
	step.End()

	if traceCtx.LLMTx == nil {
		t.Fatalf("LLMTx should be populated by httpc.End on RoundTrip attach")
	}
	tx, ok := traceCtx.LLMTx.(*agentllm.LlmTxStatus)
	if !ok {
		t.Fatalf("LLMTx wrong type: %T", traceCtx.LLMTx)
	}
	if tx.CallCount != 1 {
		t.Errorf("CallCount: want 1 (RoundTrip published once), got %d", tx.CallCount)
	}
}

func TestBind_AttachesToExistingHttpc(t *testing.T) {
	enableLLM(t)
	hc := httpc.PoolHttpcContext()
	hc.Url = "http://localhost:11434/api/chat"
	defer httpc.CloseHttpcContext(hc)

	state := Bind(hc, Config{
		Provider:      "ollama",
		Model:         "llama3",
		OperationType: "chat",
	})
	if state.Disabled() {
		t.Fatalf("state should be enabled")
	}
	if hc.Extra == nil {
		t.Fatalf("Bind must populate hc.Extra")
	}
	if got, ok := hc.Extra.(*agentllm.LLMState); !ok || got != state {
		t.Fatalf("hc.Extra: want same *LLMState (%T), got %T", state, hc.Extra)
	}
}

func TestBind_NilHcReturnsDisabled(t *testing.T) {
	enableLLM(t)
	state := Bind(nil, Config{Model: "gpt-4o"})
	if !state.Disabled() {
		t.Fatalf("nil hc must yield disabled state")
	}
}

func TestStep_EndIsNoOpAndIdempotent(t *testing.T) {
	enableLLM(t)
	_, step := Start(context.Background(), Config{Model: "gpt-4o"})
	step.End()
	step.End() // must not panic
	step.SetError(errors.New("late"), ErrorTypeAPI)
	step.End() // still no-op
}

func TestStep_ConcurrentEndIsSafe(t *testing.T) {
	enableLLM(t)
	_, step := Start(context.Background(), Config{Model: "gpt-4o"})
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			step.End()
		}()
	}
	wg.Wait()
}

func TestStart_IdempotentPending(t *testing.T) {
	// Manual + auto-inject 둘 다 적용된 사용자 코드 — 두 번째 Start 는 첫 번째
	// pending 을 그대로 유지 (RegisterPending idempotent).
	enableLLM(t)
	ctx, first := Start(context.Background(), Config{Provider: "openai", Model: "gpt-4o"})
	ctx, second := Start(ctx, Config{Provider: "anthropic", Model: "claude"})

	pending := agentllm.TakePending(ctx)
	if pending != first.LLMState {
		t.Errorf("idempotent: first registration should win, but pending != first")
	}
	// second.LLMState 는 새로 만들어진 state — 그러나 pending 등록은 안 됨.
	if pending == second.LLMState {
		t.Errorf("idempotent: second LLMState should not have replaced pending")
	}
}
