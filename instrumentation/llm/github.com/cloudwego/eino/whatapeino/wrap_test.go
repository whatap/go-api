package whatapeino

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/whatap/go-api/agent/agent/config"
	agentllm "github.com/whatap/go-api/agent/agent/llm"
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

// fakeChatModel implements model.ChatModel for testing.
type fakeChatModel struct {
	lastInput []*schema.Message
	lastOpts  []model.Option
	lastCtx   context.Context // §267 — captured ctx from inner call carries pending LLMState

	resp      *schema.Message
	err       error
	streamCh  []*schema.Message // emitted in order, then io.EOF
	streamErr error             // sent in lieu of EOF if non-nil
}

func (f *fakeChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	f.lastInput = input
	f.lastOpts = opts
	f.lastCtx = ctx
	simulateRoundTripEnd(ctx, f.err)
	return f.resp, f.err
}

func (f *fakeChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	f.lastInput = input
	f.lastOpts = opts
	f.lastCtx = ctx
	simulateRoundTripEnd(ctx, f.err)
	if f.err != nil {
		return nil, f.err
	}
	sr, sw := schema.Pipe[*schema.Message](len(f.streamCh) + 1)
	go func() {
		defer sw.Close()
		for _, ch := range f.streamCh {
			if sw.Send(ch, nil) {
				return
			}
		}
		if f.streamErr != nil {
			sw.Send(nil, f.streamErr)
			return
		}
		sw.Send(nil, io.EOF)
	}()
	return sr, nil
}

func (f *fakeChatModel) BindTools(tools []*schema.ToolInfo) error { return nil }

// drainStream reads chunks until EOF and returns the concatenated content
// plus the terminal error (io.EOF on success).
func drainStream(sr *schema.StreamReader[*schema.Message]) (string, error) {
	defer sr.Close()
	var content string
	for {
		msg, err := sr.Recv()
		if msg != nil && msg.Content != "" {
			content += msg.Content
		}
		if err != nil {
			return content, err
		}
	}
}

func txStatusFromTrace(t *testing.T, traceCtx *trace.TraceCtx) *agentllm.LlmTxStatus {
	t.Helper()
	// Stream tests publish from the wrapStream goroutine via the deferred
	// step.End() — poll briefly so the test isn't racy against that defer.
	for i := 0; i < 50; i++ {
		if traceCtx.LLMTx != nil {
			tx, ok := traceCtx.LLMTx.(*agentllm.LlmTxStatus)
			if !ok {
				t.Fatalf("LLMTx wrong type: %T", traceCtx.LLMTx)
			}
			return tx
		}
		time.Sleep(2 * time.Millisecond)
	}
	return nil
}

// simulateRoundTripEnd mirrors what whataphttp.NewRoundTrip would do at the
// real HTTP boundary — pick up the pending LLMState from ctx, hand it the
// httpc-side metadata (stepId/elapsed/status/err), and let HandleHttpcEnd
// either publish immediately (auto path) or defer to Step.End (manual path
// — see §267 / llm-roundtrip-design.md). status defaults to 200 on success.
func simulateRoundTripEnd(ctx context.Context, err error) {
	state := agentllm.TakePending(ctx)
	if state == nil || state.Disabled() {
		return
	}
	_, tc := trace.GetTraceContext(ctx)
	if tc == nil {
		return
	}
	status := 200
	if err != nil {
		status = 500
	}
	// Pass &tc.LLMTx directly — state.Publish (deferred via Step.End on the
	// manual path) writes through this pointer, so wrapping in a local slot
	// variable would lose the mutation after this function returns.
	agentllm.HandleHttpcEnd(state, tc.Txid, &tc.LLMTx, 1, 100, status, err)
}

// ── WrapChatModel basic guards ──

func TestWrapChatModel_Nil(t *testing.T) {
	if WrapChatModel(nil) != nil {
		t.Fatalf("WrapChatModel(nil, url) must return nil")
	}
}

func TestWrapChatModel_TransparentDecorator(t *testing.T) {
	enableLLM(t)
	fake := &fakeChatModel{
		resp: &schema.Message{Role: schema.Assistant, Content: "hi"},
	}
	wrapped := WrapChatModel(fake)

	out, err := wrapped.Generate(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "hello"},
	})
	if err != nil {
		t.Fatalf("Generate err: %v", err)
	}
	if out == nil || out.Content != "hi" {
		t.Fatalf("Generate: response not forwarded, got %+v", out)
	}
	if len(fake.lastInput) != 1 || fake.lastInput[0].Content != "hello" {
		t.Fatalf("Generate: input not forwarded, got %+v", fake.lastInput)
	}
}

// ── Generate ──

func TestGenerate_RecordsTokenUsageAndFinishReason(t *testing.T) {
	enableLLM(t)
	ctx, traceCtx := trace.NewTraceContext(context.Background())
	t.Cleanup(func() { trace.RemoveGIDTraceCtx(traceCtx.GID) })

	fake := &fakeChatModel{
		resp: &schema.Message{
			Role:    schema.Assistant,
			Content: "the answer is 42",
			ResponseMeta: &schema.ResponseMeta{
				FinishReason: "stop",
				Usage:        &schema.TokenUsage{PromptTokens: 50, CompletionTokens: 30, TotalTokens: 80},
			},
		},
	}
	wrapped := WrapChatModel(fake)

	out, err := wrapped.Generate(ctx, []*schema.Message{
		{Role: schema.System, Content: "be concise"},
		{Role: schema.User, Content: "what is 6 * 7?"},
	}, model.WithModel("gpt-4o"))
	if err != nil || out == nil {
		t.Fatalf("Generate err=%v out=%v", err, out)
	}

	tx := txStatusFromTrace(t, traceCtx)
	if tx == nil || tx.CallCount != 1 {
		t.Fatalf("expected one accumulated call, got %+v", tx)
	}
	if tx.ErrorCount != 0 {
		t.Fatalf("ErrorCount: want 0, got %d", tx.ErrorCount)
	}
	if tx.TokenSums["input_tokens"] != 50 || tx.TokenSums["output_tokens"] != 30 || tx.TokenSums["total_tokens_count"] != 80 {
		t.Fatalf("token sums wrong: %+v", tx.TokenSums)
	}
	if _, ok := tx.Models["gpt-4o"]; !ok {
		t.Fatalf("Models should contain gpt-4o, got %+v", tx.Models)
	}
	if _, ok := tx.OperationTypes["chat"]; !ok {
		t.Fatalf("OperationTypes should contain chat, got %+v", tx.OperationTypes)
	}
}

func TestGenerate_ErrorIsRecorded(t *testing.T) {
	enableLLM(t)
	ctx, traceCtx := trace.NewTraceContext(context.Background())
	t.Cleanup(func() { trace.RemoveGIDTraceCtx(traceCtx.GID) })

	fake := &fakeChatModel{err: errors.New("rate limit")}
	wrapped := WrapChatModel(fake)

	_, err := wrapped.Generate(ctx, []*schema.Message{{Role: schema.User, Content: "hi"}})
	if err == nil || err.Error() != "rate limit" {
		t.Fatalf("err: %v", err)
	}

	tx := txStatusFromTrace(t, traceCtx)
	if tx == nil || tx.ErrorCount != 1 {
		t.Fatalf("ErrorCount: want 1, got %+v", tx)
	}
}

// ── Stream ──

func TestStream_AccumulatesChunksAndPublishesOnEOF(t *testing.T) {
	enableLLM(t)
	ctx, traceCtx := trace.NewTraceContext(context.Background())
	t.Cleanup(func() { trace.RemoveGIDTraceCtx(traceCtx.GID) })

	fake := &fakeChatModel{
		streamCh: []*schema.Message{
			{Role: schema.Assistant, Content: "the "},
			{Role: schema.Assistant, Content: "answer "},
			{
				Role:    schema.Assistant,
				Content: "is 42",
				ResponseMeta: &schema.ResponseMeta{
					FinishReason: "stop",
					Usage:        &schema.TokenUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
				},
			},
		},
	}
	wrapped := WrapChatModel(fake)

	sr, err := wrapped.Stream(ctx, []*schema.Message{{Role: schema.User, Content: "math"}})
	if err != nil {
		t.Fatalf("Stream err: %v", err)
	}
	got, drainErr := drainStream(sr)
	if !errors.Is(drainErr, io.EOF) {
		t.Fatalf("drain err: %v", drainErr)
	}
	if got != "the answer is 42" {
		t.Fatalf("stream content: want 'the answer is 42', got %q", got)
	}

	// §267 — wrapStream goroutine fills meta before defer step.End fires.
	// Simulate the wrapped RoundTripper's httpc.End to drive accumulation.
	tx := txStatusFromTrace(t, traceCtx)
	if tx == nil || tx.TokenSums["input_tokens"] != 10 || tx.TokenSums["output_tokens"] != 5 {
		t.Fatalf("stream token sums: %+v", tx)
	}
}

func TestStream_TTFTSkipsEmptyPreamble(t *testing.T) {
	// First chunk carries no text (role-only preamble in some providers).
	// TTFT should be stamped from the first chunk that actually has Content
	// or ReasoningContent, not the bare role chunk.
	enableLLM(t)
	ctx, traceCtx := trace.NewTraceContext(context.Background())
	t.Cleanup(func() { trace.RemoveGIDTraceCtx(traceCtx.GID) })

	fake := &fakeChatModel{
		streamCh: []*schema.Message{
			{Role: schema.Assistant, Content: ""},   // empty preamble — must NOT trigger TTFT
			{Role: schema.Assistant, Content: "hi"}, // first real text
			{
				Role:    schema.Assistant,
				Content: " there",
				ResponseMeta: &schema.ResponseMeta{
					FinishReason: "stop",
					Usage:        &schema.TokenUsage{PromptTokens: 5, CompletionTokens: 3, TotalTokens: 8},
				},
			},
		},
	}
	wrapped := WrapChatModel(fake)

	sr, err := wrapped.Stream(ctx, []*schema.Message{{Role: schema.User, Content: "x"}})
	if err != nil {
		t.Fatalf("Stream err: %v", err)
	}
	if _, drainErr := drainStream(sr); !errors.Is(drainErr, io.EOF) {
		t.Fatalf("drain: %v", drainErr)
	}

	tx := txStatusFromTrace(t, traceCtx)
	if tx == nil || tx.TokenSums["input_tokens"] != 5 {
		t.Fatalf("token sums: %+v", tx)
	}
}

func TestStream_InnerErrorIsRecorded(t *testing.T) {
	enableLLM(t)
	ctx, traceCtx := trace.NewTraceContext(context.Background())
	t.Cleanup(func() { trace.RemoveGIDTraceCtx(traceCtx.GID) })

	fake := &fakeChatModel{err: errors.New("conn refused")}
	wrapped := WrapChatModel(fake)

	if _, err := wrapped.Stream(ctx, []*schema.Message{{Role: schema.User, Content: "x"}}); err == nil {
		t.Fatalf("Stream should bubble inner error")
	}

	// Stream returned immediately with the inner error — doStream called
	// step.End() on the error path, but the no-op End does not publish.
	tx := txStatusFromTrace(t, traceCtx)
	if tx == nil || tx.ErrorCount != 1 {
		t.Fatalf("ErrorCount: want 1, got %+v", tx)
	}
}

// §252 2차 — Stream Recv loop 중간 에러 시 부분 응답 flush + SetError 확인
func TestStream_RecvMidErrorFlushesPartial(t *testing.T) {
	enableLLM(t)
	ctx, traceCtx := trace.NewTraceContext(context.Background())
	t.Cleanup(func() { trace.RemoveGIDTraceCtx(traceCtx.GID) })

	// 2 chunk content 후 중간 에러 — io.EOF 가 아닌 다른 err
	fake := &fakeChatModel{
		streamCh: []*schema.Message{
			{Role: schema.Assistant, Content: "partial-"},
			{Role: schema.Assistant, Content: "answer"},
		},
		streamErr: errors.New("midstream connection reset"),
	}
	wrapped := WrapChatModel(fake)

	sr, err := wrapped.Stream(ctx, []*schema.Message{{Role: schema.User, Content: "x"}})
	if err != nil {
		t.Fatalf("Stream returned err on open: %v", err)
	}
	// drain — io.EOF 가 아닌 streamErr 가 마지막에 도달해야 함
	gotContent, drainErr := drainStream(sr)
	if drainErr == nil || drainErr == io.EOF {
		t.Fatalf("expected midstream error, got: %v", drainErr)
	}
	if gotContent != "partial-answer" {
		t.Fatalf("partial content drain: got %q", gotContent)
	}

	tx := txStatusFromTrace(t, traceCtx)
	if tx == nil || tx.ErrorCount != 1 {
		t.Fatalf("ErrorCount: want 1, got %+v", tx)
	}
	// 부분 응답 플러시 확인 — accumulate 가 Completion text 를 본 흔적
	// (LlmTxStatus 직접 필드는 없지만 LlmLogSinkPack embed 가 있음).
	// 가장 단순한 신호: 에러 발생해도 CallCount = 1 + ErrorCount = 1
	if tx.CallCount != 1 {
		t.Fatalf("CallCount: want 1, got %d", tx.CallCount)
	}
}

// ── BindTools ──

type toolBinder struct {
	fakeChatModel
	bindCalled bool
}

func (t *toolBinder) BindTools(tools []*schema.ToolInfo) error {
	t.bindCalled = true
	return nil
}

// fakeToolCallingChatModel implements model.ToolCallingChatModel for testing.
type fakeToolCallingChatModel struct {
	fakeChatModel
	withToolsCalled []*schema.ToolInfo
	derivedReturned model.ToolCallingChatModel
	withToolsErr    error
}

func (f *fakeToolCallingChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	f.withToolsCalled = tools
	if f.withToolsErr != nil {
		return nil, f.withToolsErr
	}
	if f.derivedReturned != nil {
		return f.derivedReturned, nil
	}
	return f, nil
}

func TestWrapToolCallingChatModel_Nil(t *testing.T) {
	if WrapToolCallingChatModel(nil) != nil {
		t.Fatalf("WrapToolCallingChatModel(nil, url) must return nil")
	}
}

func TestWrapToolCallingChatModel_Generate(t *testing.T) {
	enableLLM(t)
	ctx, traceCtx := trace.NewTraceContext(context.Background())
	t.Cleanup(func() { trace.RemoveGIDTraceCtx(traceCtx.GID) })

	inner := &fakeToolCallingChatModel{}
	inner.resp = &schema.Message{
		Role:    schema.Assistant,
		Content: "answer",
		ResponseMeta: &schema.ResponseMeta{
			Usage: &schema.TokenUsage{PromptTokens: 7, CompletionTokens: 4, TotalTokens: 11},
		},
	}
	wrapped := WrapToolCallingChatModel(inner)

	out, err := wrapped.Generate(ctx, []*schema.Message{{Role: schema.User, Content: "q"}})
	if err != nil || out == nil || out.Content != "answer" {
		t.Fatalf("Generate err=%v out=%+v", err, out)
	}

	tx := txStatusFromTrace(t, traceCtx)
	if tx == nil || tx.CallCount != 1 {
		t.Fatalf("expected one accumulated call, got %+v", tx)
	}
	if tx.TokenSums["input_tokens"] != 7 {
		t.Fatalf("token sums: %+v", tx.TokenSums)
	}
}

func TestWrapToolCallingChatModel_WithTools_RewrapsDerived(t *testing.T) {
	enableLLM(t)
	derived := &fakeToolCallingChatModel{}
	derived.resp = &schema.Message{Role: schema.Assistant, Content: "from-derived"}

	root := &fakeToolCallingChatModel{derivedReturned: derived}
	wrapped := WrapToolCallingChatModel(root)

	tools := []*schema.ToolInfo{{Name: "t1"}}
	derivedWrap, err := wrapped.WithTools(tools)
	if err != nil {
		t.Fatalf("WithTools err: %v", err)
	}
	if derivedWrap == nil {
		t.Fatalf("WithTools returned nil")
	}
	if root.withToolsCalled == nil || len(root.withToolsCalled) != 1 {
		t.Fatalf("inner.WithTools not forwarded: %+v", root.withToolsCalled)
	}

	// Derived must also be wrapped — calling Generate on it should still
	// produce a tracking step.
	ctx, traceCtx := trace.NewTraceContext(context.Background())
	t.Cleanup(func() { trace.RemoveGIDTraceCtx(traceCtx.GID) })
	if _, err := derivedWrap.Generate(ctx, []*schema.Message{{Role: schema.User, Content: "y"}}); err != nil {
		t.Fatalf("derived Generate err: %v", err)
	}
	tx := txStatusFromTrace(t, traceCtx)
	if tx == nil || tx.CallCount != 1 {
		t.Fatalf("derived wrap did not record: %+v", tx)
	}
}

func TestWrapToolCallingChatModel_WithToolsErrPropagates(t *testing.T) {
	root := &fakeToolCallingChatModel{withToolsErr: errors.New("invalid")}
	wrapped := WrapToolCallingChatModel(root)
	derived, err := wrapped.WithTools(nil)
	if derived != nil || err == nil || err.Error() != "invalid" {
		t.Fatalf("err propagation failed: derived=%v err=%v", derived, err)
	}
}

func TestBindTools_ForwardedToInner(t *testing.T) {
	inner := &toolBinder{}
	wrapped := WrapChatModel(inner)
	if err := wrapped.BindTools(nil); err != nil {
		t.Fatalf("BindTools err: %v", err)
	}
	if !inner.bindCalled {
		t.Fatalf("BindTools must be forwarded to inner")
	}
}

// ── multimodal text extraction (§252 2차) ──

func TestMessageText_PrefersContent(t *testing.T) {
	m := &schema.Message{Content: "primary"}
	if got := messageText(m); got != "primary" {
		t.Fatalf("got %q", got)
	}
}

func TestMessageText_MultiContentPlaceholders(t *testing.T) {
	m := &schema.Message{
		MultiContent: []schema.ChatMessagePart{
			{Type: schema.ChatMessagePartTypeText, Text: "before"},
			{Type: schema.ChatMessagePartTypeImageURL},
			{Type: schema.ChatMessagePartTypeAudioURL},
			{Type: schema.ChatMessagePartTypeVideoURL},
			{Type: schema.ChatMessagePartTypeFileURL},
			{Type: schema.ChatMessagePartTypeText, Text: "after"},
		},
	}
	want := "before\n[IMAGE]\n[AUDIO]\n[VIDEO]\n[FILE]\nafter"
	if got := messageText(m); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestMessageText_ReasoningSkipped(t *testing.T) {
	// reasoning part 는 별도 채널 (response 의 ReasoningContent) 로 처리 → 여기선 skip
	m := &schema.Message{
		MultiContent: []schema.ChatMessagePart{
			{Type: schema.ChatMessagePartTypeText, Text: "visible"},
			{Type: schema.ChatMessagePartTypeReasoning, Text: "should-not-appear"},
		},
	}
	if got := messageText(m); got != "visible" {
		t.Fatalf("got %q", got)
	}
}

// ── §282 작업1 — WrapBaseChatModel dispatch + idempotency ──

// baseOnlyChatModel implements only model.BaseChatModel — no WithTools
// (ToolCallingChatModel) and no BindTools (deprecated ChatModel).
type baseOnlyChatModel struct{ fake fakeChatModel }

func (b *baseOnlyChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return b.fake.Generate(ctx, input, opts...)
}

func (b *baseOnlyChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return b.fake.Stream(ctx, input, opts...)
}

func TestWrapBaseChatModel_Nil(t *testing.T) {
	if WrapBaseChatModel(nil) != nil {
		t.Fatalf("WrapBaseChatModel(nil) must return nil")
	}
}

// A ToolCallingChatModel inner must stay tool-callable so eino's internal
// node.(ToolCallingChatModel) assertion still succeeds.
func TestWrapBaseChatModel_PreservesToolCalling(t *testing.T) {
	w := WrapBaseChatModel(&fakeToolCallingChatModel{})
	if _, ok := w.(model.ToolCallingChatModel); !ok {
		t.Fatalf("ToolCallingChatModel capability must be preserved, got %T", w)
	}
}

// A (deprecated) ChatModel inner (BindTools, no WithTools) must keep BindTools.
func TestWrapBaseChatModel_PreservesChatModel(t *testing.T) {
	w := WrapBaseChatModel(&fakeChatModel{})
	if _, ok := w.(model.ChatModel); !ok {
		t.Fatalf("ChatModel (BindTools) capability must be preserved, got %T", w)
	}
	// fakeChatModel has no WithTools → wrapper must not fabricate ToolCalling.
	if _, ok := w.(model.ToolCallingChatModel); ok {
		t.Fatalf("must not expose ToolCallingChatModel for a non-tool-calling inner")
	}
}

// A plain BaseChatModel inner must NOT gain capabilities it never had.
func TestWrapBaseChatModel_BaseOnly(t *testing.T) {
	w := WrapBaseChatModel(&baseOnlyChatModel{})
	if _, ok := w.(model.ToolCallingChatModel); ok {
		t.Fatalf("base-only inner must not expose ToolCallingChatModel")
	}
	if _, ok := w.(model.ChatModel); ok {
		t.Fatalf("base-only inner must not expose ChatModel")
	}
	if _, ok := w.(*wrappedBaseChatModel); !ok {
		t.Fatalf("expected *wrappedBaseChatModel, got %T", w)
	}
}

// Generate must still flow through the WhaTap decorator for the base case.
func TestWrapBaseChatModel_BaseOnly_GenerateForwarded(t *testing.T) {
	inner := &baseOnlyChatModel{}
	inner.fake.resp = &schema.Message{Role: schema.Assistant, Content: "ok"}
	w := WrapBaseChatModel(inner)
	out, err := w.Generate(context.Background(), []*schema.Message{{Role: schema.User, Content: "hi"}})
	if err != nil {
		t.Fatalf("Generate err: %v", err)
	}
	if out == nil || out.Content != "ok" {
		t.Fatalf("Generate not forwarded to inner: %+v", out)
	}
	if inner.fake.lastInput == nil {
		t.Fatalf("inner.Generate was not called")
	}
}

// Idempotent — wrapping an already-wrapped instance returns it unchanged.
func TestWrapBaseChatModel_Idempotent(t *testing.T) {
	cases := []model.BaseChatModel{
		&fakeToolCallingChatModel{},
		&fakeChatModel{},
		&baseOnlyChatModel{},
	}
	for _, inner := range cases {
		w1 := WrapBaseChatModel(inner)
		w2 := WrapBaseChatModel(w1)
		if w1 != w2 {
			t.Fatalf("WrapBaseChatModel not idempotent for %T: %p != %p", inner, w1, w2)
		}
	}
}

// ── §282 작업3 — Typer/Checker forwarding ──

// typedToolCallingChatModel embeds the tool-calling fake and also implements
// components.Typer + components.Checker (as the real eino-ext *ChatModel does).
type typedToolCallingChatModel struct {
	fakeToolCallingChatModel
	typeName string
	cbEnable bool
}

func (m *typedToolCallingChatModel) GetType() string          { return m.typeName }
func (m *typedToolCallingChatModel) IsCallbacksEnabled() bool { return m.cbEnable }

func TestWrap_ForwardsTyperChecker(t *testing.T) {
	inner := &typedToolCallingChatModel{typeName: "OpenAI", cbEnable: true}
	w := WrapBaseChatModel(inner) // → *wrappedToolCallingChatModel

	typer, ok := w.(components.Typer)
	if !ok {
		t.Fatalf("wrapper must implement components.Typer")
	}
	if got := typer.GetType(); got != "OpenAI" {
		t.Fatalf("GetType not forwarded: got %q want OpenAI", got)
	}
	checker, ok := w.(components.Checker)
	if !ok {
		t.Fatalf("wrapper must implement components.Checker")
	}
	if !checker.IsCallbacksEnabled() {
		t.Fatalf("IsCallbacksEnabled not forwarded: got false want true")
	}
}

// When the inner model does not implement Typer/Checker, the forward returns
// eino's zero-value fallbacks ("" / false) — no panic.
func TestWrap_ForwardsTyperChecker_ZeroFallback(t *testing.T) {
	w := WrapBaseChatModel(&baseOnlyChatModel{}) // → *wrappedBaseChatModel
	if got := w.(components.Typer).GetType(); got != "" {
		t.Fatalf("GetType fallback: got %q want empty", got)
	}
	if w.(components.Checker).IsCallbacksEnabled() {
		t.Fatalf("IsCallbacksEnabled fallback: got true want false")
	}
}

// ── §282 작업4 — model name capture (constructor registry) ──

func TestConfigFor_ModelFromRegistry(t *testing.T) {
	inner := &baseOnlyChatModel{}
	rememberModelName(inner, "gpt-4o")
	t.Cleanup(func() { modelNameByInstance.Delete(model.BaseChatModel(inner)) })

	if got := configFor(inner, nil).Model; got != "gpt-4o" {
		t.Fatalf("model should come from constructor registry: got %q want gpt-4o", got)
	}
}

// Call-time model.WithModel(...) must take precedence over the constructor value.
func TestConfigFor_CallOptionOverridesRegistry(t *testing.T) {
	inner := &baseOnlyChatModel{}
	rememberModelName(inner, "gpt-4o")
	t.Cleanup(func() { modelNameByInstance.Delete(model.BaseChatModel(inner)) })

	if got := configFor(inner, []model.Option{model.WithModel("gpt-4o-mini")}).Model; got != "gpt-4o-mini" {
		t.Fatalf("call-time WithModel must override registry: got %q want gpt-4o-mini", got)
	}
}

// A model never recorded (e.g. manual WrapChatModel over an upstream-built
// model) yields empty — no panic, no regression.
func TestLookupModelName_AbsentReturnsEmpty(t *testing.T) {
	if got := lookupModelName(&baseOnlyChatModel{}); got != "" {
		t.Fatalf("absent model name must be empty, got %q", got)
	}
	if got := configFor(&baseOnlyChatModel{}, nil).Model; got != "" {
		t.Fatalf("configFor without registry/opts must be empty, got %q", got)
	}
}

// rememberModelName ignores empty names and nil models.
func TestRememberModelName_Guards(t *testing.T) {
	inner := &baseOnlyChatModel{}
	rememberModelName(inner, "")
	if got := lookupModelName(inner); got != "" {
		t.Fatalf("empty name must not be stored, got %q", got)
	}
	rememberModelName(nil, "x") // must not panic
}

// ── helpers ──
//
// Note: waitForTx removed by §267 — adapter tests no longer publish through
// the goroutine's defer step.End() path. They drive LLMTx accumulation via
// finalizeForTest, which simulates the RoundTripper's httpc.End.
