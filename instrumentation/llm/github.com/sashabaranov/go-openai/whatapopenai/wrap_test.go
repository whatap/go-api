package whatapopenai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	openai "github.com/sashabaranov/go-openai"
	"github.com/whatap/go-api/agent/agent/config"
	agentllm "github.com/whatap/go-api/agent/agent/llm"
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

// ── WrapClient guards ──

func TestWrapClient_Nil(t *testing.T) {
	if WrapClient(nil) != nil {
		t.Fatalf("WrapClient(nil) must return nil")
	}
}

func TestWrapClient_EmbedsInner(t *testing.T) {
	inner := openai.NewClient("test-key")
	w := WrapClient(inner)
	if w == nil || w.Client != inner {
		t.Fatalf("WrapClient must embed the inner client")
	}
}

// ── extract helpers ──

func TestUsageToTokens_AllFields(t *testing.T) {
	u := openai.Usage{
		PromptTokens:     100,
		CompletionTokens: 200,
		TotalTokens:      300,
		PromptTokensDetails: &openai.PromptTokensDetails{
			AudioTokens:  10,
			CachedTokens: 50,
		},
		CompletionTokensDetails: &openai.CompletionTokensDetails{
			AudioTokens:              5,
			ReasoningTokens:          15,
			AcceptedPredictionTokens: 3,
			RejectedPredictionTokens: 1,
		},
	}
	tk := usageToTokens(u)
	if tk.Input != 100 || tk.Output != 200 || tk.Total != 300 {
		t.Fatalf("base fields wrong: %+v", tk)
	}
	if tk.AudioInput != 10 || tk.Cached != 50 {
		t.Fatalf("PromptTokensDetails: %+v", tk)
	}
	if tk.AudioOutput != 5 || tk.Reasoning != 15 || tk.AcceptedPrediction != 3 || tk.RejectedPrediction != 1 {
		t.Fatalf("CompletionTokensDetails: %+v", tk)
	}
}

func TestUsageToTokens_NilDetails(t *testing.T) {
	u := openai.Usage{PromptTokens: 5, CompletionTokens: 10, TotalTokens: 15}
	tk := usageToTokens(u)
	if tk.Cached != 0 || tk.Reasoning != 0 {
		t.Fatalf("nil details must produce zero subtokens: %+v", tk)
	}
}

func TestChatMessageText_PrefersContent(t *testing.T) {
	m := openai.ChatCompletionMessage{Content: "primary"}
	if got := chatMessageText(m); got != "primary" {
		t.Fatalf("got %q", got)
	}
}

func TestChatMessageText_FallsBackToMultiContent(t *testing.T) {
	m := openai.ChatCompletionMessage{
		MultiContent: []openai.ChatMessagePart{
			{Type: openai.ChatMessagePartTypeText, Text: "part-1"},
			{Type: openai.ChatMessagePartTypeImageURL}, // → [IMAGE] placeholder
			{Type: openai.ChatMessagePartTypeText, Text: "part-2"},
		},
	}
	if got := chatMessageText(m); got != "part-1\n[IMAGE]\npart-2" {
		t.Fatalf("got %q", got)
	}
}

func TestChatMessageText_UnknownPartTypePlaceholder(t *testing.T) {
	// 미래의 새로운 ChatMessagePartType (예: input_audio) 가 들어와도 placeholder 로 표시
	m := openai.ChatCompletionMessage{
		MultiContent: []openai.ChatMessagePart{
			{Type: openai.ChatMessagePartTypeText, Text: "hello"},
			{Type: openai.ChatMessagePartType("input_audio")},
		},
	}
	if got := chatMessageText(m); got != "hello\n[INPUT_AUDIO]" {
		t.Fatalf("got %q", got)
	}
}

func TestMarshalToolCalls(t *testing.T) {
	calls := []openai.ToolCall{
		{
			ID:   "call_1",
			Type: openai.ToolTypeFunction,
			Function: openai.FunctionCall{
				Name:      "get_weather",
				Arguments: `{"city":"Seoul"}`,
			},
		},
	}
	got := marshalToolCalls(calls)
	if !strings.Contains(got, "get_weather") || !strings.Contains(got, "Seoul") {
		t.Fatalf("marshalled JSON missing fields: %q", got)
	}
}

func TestCompletionPromptText_StringTypes(t *testing.T) {
	if got := completionPromptText("hello"); got != "hello" {
		t.Fatalf("string: %q", got)
	}
	if got := completionPromptText([]string{"a", "b"}); got != "a\nb" {
		t.Fatalf("[]string: %q", got)
	}
	if got := completionPromptText([]int{1, 2}); got != "" {
		t.Fatalf("token IDs must yield empty: %q", got)
	}
}

// ── httptest integration: full request → wrap → response → step verification ──

// fakeOpenAIServer accepts the path and replies with the supplied JSON body.
func fakeOpenAIServer(t *testing.T, path string, body any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			http.Error(w, "unexpected path: "+r.URL.Path, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}))
}

func newWrappedClient(serverURL string) *Client {
	cfg := openai.DefaultConfig("test-key")
	cfg.BaseURL = serverURL + "/v1"
	// §267 — wrap the HTTP transport so the SDK's outbound call drives
	// httpc.Start/End, which picks up the pending LLMState registered by
	// llm.Start inside CreateChatCompletion / etc.
	cfg.HTTPClient = &http.Client{
		Transport: whataphttp.NewRoundTrip(context.Background(), http.DefaultTransport),
	}
	return WrapClient(openai.NewClientWithConfig(cfg))
}

func TestCreateChatCompletion_RecordsTokens(t *testing.T) {
	enableLLM(t)
	ctx, traceCtx := trace.NewTraceContext(context.Background())
	t.Cleanup(func() { trace.RemoveGIDTraceCtx(traceCtx.GID) })

	srv := fakeOpenAIServer(t, "/v1/chat/completions", openai.ChatCompletionResponse{
		ID:    "cmpl-x",
		Model: "gpt-4o",
		Choices: []openai.ChatCompletionChoice{
			{
				Index:        0,
				Message:      openai.ChatCompletionMessage{Role: "assistant", Content: "the answer is 42"},
				FinishReason: openai.FinishReasonStop,
			},
		},
		Usage: openai.Usage{PromptTokens: 50, CompletionTokens: 30, TotalTokens: 80},
	})
	defer srv.Close()

	client := newWrappedClient(srv.URL)
	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    "gpt-4o",
		Messages: []openai.ChatCompletionMessage{{Role: "user", Content: "what is 6 * 7?"}},
	})
	if err != nil {
		t.Fatalf("CreateChatCompletion err: %v", err)
	}
	if resp.Choices[0].Message.Content != "the answer is 42" {
		t.Fatalf("response content: %q", resp.Choices[0].Message.Content)
	}

	tx := waitForTx(t, traceCtx, 1)
	if tx.TokenSums["input_tokens"] != 50 || tx.TokenSums["output_tokens"] != 30 || tx.TokenSums["total_tokens_count"] != 80 {
		t.Fatalf("token sums: %+v", tx.TokenSums)
	}
	if _, ok := tx.Models["gpt-4o"]; !ok {
		t.Fatalf("Models should contain gpt-4o, got %+v", tx.Models)
	}
}

func TestCreateChatCompletion_HttpError(t *testing.T) {
	enableLLM(t)
	ctx, traceCtx := trace.NewTraceContext(context.Background())
	t.Cleanup(func() { trace.RemoveGIDTraceCtx(traceCtx.GID) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newWrappedClient(srv.URL)
	_, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    "gpt-4o",
		Messages: []openai.ChatCompletionMessage{{Role: "user", Content: "x"}},
	})
	if err == nil {
		t.Fatalf("expected error from 500 response")
	}
	tx := waitForTx(t, traceCtx, 1)
	if tx.ErrorCount != 1 {
		t.Fatalf("ErrorCount: want 1, got %d", tx.ErrorCount)
	}
}

func TestCreateEmbeddings_RecordsDimensionsAndCount(t *testing.T) {
	enableLLM(t)
	ctx, traceCtx := trace.NewTraceContext(context.Background())
	t.Cleanup(func() { trace.RemoveGIDTraceCtx(traceCtx.GID) })

	srv := fakeOpenAIServer(t, "/v1/embeddings", openai.EmbeddingResponse{
		Object: "list",
		Model:  openai.SmallEmbedding3,
		Data: []openai.Embedding{
			{Embedding: []float32{0.1, 0.2, 0.3}},
			{Embedding: []float32{0.4, 0.5, 0.6}},
		},
		Usage: openai.Usage{PromptTokens: 10, TotalTokens: 10},
	})
	defer srv.Close()

	client := newWrappedClient(srv.URL)
	_, err := client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Model: openai.SmallEmbedding3,
		Input: []string{"hello", "world"},
	})
	if err != nil {
		t.Fatalf("CreateEmbeddings err: %v", err)
	}

	tx := waitForTx(t, traceCtx, 1)
	if tx.TokenSums["embedding_count"] != 2 {
		t.Fatalf("embedding_count: want 2, got %d", tx.TokenSums["embedding_count"])
	}
	// Note: dimensions is recorded on the per-step LlmStepStatus pack but is
	// intentionally not summed across the transaction (see txSummaryTokenFields
	// in agent/agent/llm/pack.go). Per-step dimension tracking is verified
	// indirectly by the Tokens struct unit test in agent/agent/llm.
}

// ── Stream ──

// fakeStreamServer emits SSE chunks for /v1/chat/completions.
func fakeStreamServer(t *testing.T, chunks []openai.ChatCompletionStreamResponse) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.Error(w, "unexpected path", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "no flusher", http.StatusInternalServerError)
			return
		}
		for _, c := range chunks {
			data, _ := json.Marshal(c)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
}

func TestCreateChatCompletionStream_AccumulatesAndPublishes(t *testing.T) {
	enableLLM(t)
	ctx, traceCtx := trace.NewTraceContext(context.Background())
	t.Cleanup(func() { trace.RemoveGIDTraceCtx(traceCtx.GID) })

	srv := fakeStreamServer(t, []openai.ChatCompletionStreamResponse{
		{Choices: []openai.ChatCompletionStreamChoice{{Delta: openai.ChatCompletionStreamChoiceDelta{Content: "the "}}}},
		{Choices: []openai.ChatCompletionStreamChoice{{Delta: openai.ChatCompletionStreamChoiceDelta{Content: "answer "}}}},
		{
			Choices: []openai.ChatCompletionStreamChoice{
				{
					Delta:        openai.ChatCompletionStreamChoiceDelta{Content: "is 42"},
					FinishReason: openai.FinishReasonStop,
				},
			},
			Usage: &openai.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		},
	})
	defer srv.Close()

	client := newWrappedClient(srv.URL)
	stream, err := client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:    "gpt-4o",
		Messages: []openai.ChatCompletionMessage{{Role: "user", Content: "x"}},
		Stream:   true,
	})
	if err != nil {
		t.Fatalf("Stream err: %v", err)
	}

	var got string
	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("Recv: %v", err)
		}
		for _, c := range resp.Choices {
			got += c.Delta.Content
		}
	}
	stream.Close()

	if got != "the answer is 42" {
		t.Fatalf("stream content: %q", got)
	}

	tx := waitForTx(t, traceCtx, 1)
	if tx.TokenSums["input_tokens"] != 10 || tx.TokenSums["output_tokens"] != 5 {
		t.Fatalf("stream token sums: %+v", tx.TokenSums)
	}
}

// §252 2차 — Stream Recv loop 중간 에러 시 SetError + 부분 응답 finalize 확인
func TestCreateChatCompletionStream_MidStreamError(t *testing.T) {
	enableLLM(t)
	ctx, traceCtx := trace.NewTraceContext(context.Background())
	t.Cleanup(func() { trace.RemoveGIDTraceCtx(traceCtx.GID) })

	// SSE 2 chunk 전송 후 connection close (마지막 [DONE] 없이 truncate)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		chunks := []openai.ChatCompletionStreamResponse{
			{Choices: []openai.ChatCompletionStreamChoice{{Delta: openai.ChatCompletionStreamChoiceDelta{Content: "partial "}}}},
			{Choices: []openai.ChatCompletionStreamChoice{{Delta: openai.ChatCompletionStreamChoiceDelta{Content: "answer"}}}},
		}
		for _, c := range chunks {
			data, _ := json.Marshal(c)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
		// truncate — malformed final event (SSE parse 에러 유도)
		_, _ = fmt.Fprint(w, "data: {not-json\n\n")
		flusher.Flush()
	}))
	defer srv.Close()

	client := newWrappedClient(srv.URL)
	stream, err := client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:    "gpt-4o",
		Messages: []openai.ChatCompletionMessage{{Role: "user", Content: "x"}},
		Stream:   true,
	})
	if err != nil {
		t.Fatalf("Stream open err: %v", err)
	}

	var got string
	var lastErr error
	for {
		resp, recvErr := stream.Recv()
		if recvErr != nil {
			lastErr = recvErr
			break
		}
		for _, c := range resp.Choices {
			got += c.Delta.Content
		}
	}
	stream.Close()

	// 마지막 에러는 io.EOF 가 아니어야 (SSE parse 실패)
	if lastErr == nil || errors.Is(lastErr, io.EOF) {
		t.Fatalf("expected non-EOF parse error, got: %v", lastErr)
	}
	if got != "partial answer" {
		t.Fatalf("partial content drain: %q", got)
	}

	tx := waitForTx(t, traceCtx, 1)
	if tx.ErrorCount != 1 {
		t.Fatalf("ErrorCount: want 1, got %d", tx.ErrorCount)
	}
}

// ── helpers ──

func waitForTx(t *testing.T, traceCtx *trace.TraceCtx, want int64) *agentllm.LlmTxStatus {
	t.Helper()
	for i := 0; i < 200; i++ {
		if traceCtx.LLMTx != nil {
			if tx, ok := traceCtx.LLMTx.(*agentllm.LlmTxStatus); ok && tx.CallCount >= want {
				return tx
			}
		}
		// short pause without time package import — busy yield
		for j := 0; j < 10000; j++ {
			_ = j
		}
	}
	t.Fatalf("LLMTx did not reach CallCount=%d in time", want)
	return nil
}
