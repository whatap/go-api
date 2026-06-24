package whatapeino

import (
	"context"

	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/whatap/go-api/llm"
)

// WrapChatModel returns a ChatModel that emits WhaTap LLM monitoring data
// for every Generate / Stream call. Behaviour is otherwise identical to the
// inner model — the wrapper is a transparent decorator.
//
// The real HTTP endpoint URL is captured by the wrapped RoundTripper inside
// the SDK (§267). For this to work, the SDK's HTTP transport must be wrapped
// with whataphttp.NewRoundTrip — typically via §254 auto-inject, or by
// supplying a custom http.Client in the SDK config.
//
// Pass nil inner to get nil back. Wrapping the same instance twice returns
// the existing wrapper (idempotent — §264 §4.2).
//
// Deprecated path: eino flagged model.ChatModel as deprecated in favour of
// model.ToolCallingChatModel. Use WrapToolCallingChatModel for new code.
func WrapChatModel(inner model.ChatModel) model.ChatModel {
	if inner == nil {
		return nil
	}
	if _, ok := inner.(*wrappedChatModel); ok {
		return inner // idempotent — already wrapped (§264 §4.2)
	}
	return &wrappedChatModel{inner: inner}
}

// WrapToolCallingChatModel is the WrapChatModel equivalent for the newer
// model.ToolCallingChatModel interface (BindTools → WithTools immutable).
// WithTools returns a fresh wrapped instance so monitoring follows derived
// models too. URL capture semantics are identical to WrapChatModel.
func WrapToolCallingChatModel(inner model.ToolCallingChatModel) model.ToolCallingChatModel {
	if inner == nil {
		return nil
	}
	if _, ok := inner.(*wrappedToolCallingChatModel); ok {
		return inner // idempotent — already wrapped (§264 §4.2)
	}
	return &wrappedToolCallingChatModel{inner: inner}
}

// WrapBaseChatModel wraps any model.BaseChatModel for WhaTap LLM monitoring
// without changing the caller-visible static type — it accepts and returns
// model.BaseChatModel. This is the wrapper used by §282 auto-inject at eino
// compose call sites (Chain.AppendChatModel / Graph.AddChatModelNode /
// Workflow.AddChatModelNode / ChainBranch.AddChatModel / Parallel.AddChatModel),
// whose model argument is declared as model.BaseChatModel. Because the static
// type is preserved, wrapping the argument never changes the user's variable
// types and cannot introduce a compile error.
//
// To preserve the inner model's capabilities for eino's internal runtime type
// assertions, the concrete wrapper is chosen by the inner's dynamic type:
//   - a ToolCallingChatModel stays tool-callable (WithTools),
//   - a (deprecated) ChatModel keeps BindTools,
//   - a plain BaseChatModel is wrapped as-is.
//
// Pass nil inner to get nil back. Idempotent — wrapping an already-wrapped
// instance returns it unchanged.
func WrapBaseChatModel(inner model.BaseChatModel) model.BaseChatModel {
	if inner == nil {
		return nil
	}
	switch inner.(type) {
	case *wrappedToolCallingChatModel, *wrappedChatModel, *wrappedBaseChatModel:
		return inner // idempotent — already wrapped (§264 §4.2)
	}
	switch v := inner.(type) {
	case model.ToolCallingChatModel:
		return WrapToolCallingChatModel(v)
	case model.ChatModel:
		return WrapChatModel(v)
	default:
		return &wrappedBaseChatModel{inner: inner}
	}
}

type wrappedChatModel struct {
	inner model.ChatModel
}

func (w *wrappedChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return doGenerate(w.inner, ctx, input, opts)
}

func (w *wrappedChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return doStream(w.inner, ctx, input, opts)
}

// BindTools forwards verbatim — tool registration has no observable side
// effect on the WhaTap step.
func (w *wrappedChatModel) BindTools(tools []*schema.ToolInfo) error {
	return w.inner.BindTools(tools)
}

type wrappedToolCallingChatModel struct {
	inner model.ToolCallingChatModel
}

func (w *wrappedToolCallingChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return doGenerate(w.inner, ctx, input, opts)
}

func (w *wrappedToolCallingChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return doStream(w.inner, ctx, input, opts)
}

// WithTools returns a derived ToolCallingChatModel re-wrapped so the new
// instance also emits WhaTap data.
func (w *wrappedToolCallingChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	derived, err := w.inner.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return &wrappedToolCallingChatModel{inner: derived}, nil
}

// WrapGenerate is the call-site wrapper for a direct `cm.Generate(ctx, in)`
// call (§282 작업2). Auto-inject rewrites the concrete eino-ext
// `*ChatModel.Generate` call into `whatapeino.WrapGenerate(cm, ctx, in)`, so
// the WhaTap LLM step (model/token/op via response parsing) is emitted around
// the inner Generate. The return values are identical to cm.Generate — the
// caller's variable types are unchanged.
//
// cm keeps its concrete static type at the call site; this helper only borrows
// it as a model.BaseChatModel, so the rewrite never changes user types.
func WrapGenerate(cm model.BaseChatModel, ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return doGenerate(cm, ctx, input, opts)
}

// WrapStream is the call-site wrapper for a direct `cm.Stream(ctx, in)` call
// (§282 작업2). Mirrors WrapGenerate for the streaming path.
func WrapStream(cm model.BaseChatModel, ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return doStream(cm, ctx, input, opts)
}

// wrappedBaseChatModel is the minimal decorator used when the inner model
// implements only model.BaseChatModel (neither ToolCallingChatModel nor the
// deprecated ChatModel). Returned by WrapBaseChatModel for that case.
type wrappedBaseChatModel struct {
	inner model.BaseChatModel
}

func (w *wrappedBaseChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return doGenerate(w.inner, ctx, input, opts)
}

func (w *wrappedBaseChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return doStream(w.inner, ctx, input, opts)
}

// §282 작업3 — Typer/Checker forwarding. eino reads a component's display
// type name via components.Typer.GetType and its callback policy via
// components.Checker.IsCallbacksEnabled (both optional interfaces, probed by
// runtime assertion). The decorators forward both to the inner model so the
// node's type name (e.g. "OpenAI") and callback behaviour are preserved
// through the wrapper. forwardType returns "" / forwardCallbacks returns false
// when the inner does not implement the optional interface — matching eino's
// own zero-value fallbacks.
func forwardType(inner any) string {
	if t, ok := inner.(components.Typer); ok {
		return t.GetType()
	}
	return ""
}

func forwardCallbacks(inner any) bool {
	if c, ok := inner.(components.Checker); ok {
		return c.IsCallbacksEnabled()
	}
	return false
}

func (w *wrappedChatModel) GetType() string                     { return forwardType(w.inner) }
func (w *wrappedChatModel) IsCallbacksEnabled() bool            { return forwardCallbacks(w.inner) }
func (w *wrappedToolCallingChatModel) GetType() string          { return forwardType(w.inner) }
func (w *wrappedToolCallingChatModel) IsCallbacksEnabled() bool { return forwardCallbacks(w.inner) }
func (w *wrappedBaseChatModel) GetType() string                 { return forwardType(w.inner) }
func (w *wrappedBaseChatModel) IsCallbacksEnabled() bool        { return forwardCallbacks(w.inner) }

// doGenerate wraps the synchronous Generate call shared by both ChatModel
// and ToolCallingChatModel paths. Registers a pending LLMState on ctx and
// lets the wrapped RoundTripper inside inner.Generate own the HTTPC step.
func doGenerate(inner model.BaseChatModel, ctx context.Context, input []*schema.Message, opts []model.Option) (*schema.Message, error) {
	ctx, step := llm.Start(ctx, configFor(inner, opts))
	defer step.End()

	fillInputs(step, input)
	if t, ok := extractTemperature(opts); ok {
		step.SetTemperature(t)
	}

	out, err := inner.Generate(ctx, input, opts...)
	if err != nil {
		step.SetError(err, llm.ErrorTypeAPI)
		return out, err
	}
	fillOutput(step, out)
	return out, err
}

// doStream wraps the streaming Stream call shared by both ChatModel and
// ToolCallingChatModel paths. The pending LLMState rides on the returned ctx
// — the RoundTripper inside inner.Stream attaches it to the HTTPC step.
func doStream(inner model.BaseChatModel, ctx context.Context, input []*schema.Message, opts []model.Option) (*schema.StreamReader[*schema.Message], error) {
	ctx, step := llm.Start(ctx, configFor(inner, opts))

	fillInputs(step, input)
	if t, ok := extractTemperature(opts); ok {
		step.SetTemperature(t)
	}
	step.MarkStream()

	innerStream, err := inner.Stream(ctx, input, opts...)
	if err != nil {
		step.SetError(err, llm.ErrorTypeAPI)
		step.End()
		return nil, err
	}
	return wrapStream(step, innerStream), nil
}

// configFor constructs an llm.Config from the call-time options and the inner
// model. Model precedence: call-time model.WithModel(...) override > the
// constructor-time model recorded by the §254 helpers (§282 작업4) > "".
// Provider is intentionally left empty — state.SetURL fills it from the real
// endpoint URL (InferProviderURL) when the wrapped RoundTripper attaches the
// pending state to httpc.Start, matching the §253/§255 adapters' behaviour.
func configFor(inner model.BaseChatModel, opts []model.Option) llm.Config {
	m := extractModel(opts)
	if m == "" {
		m = lookupModelName(inner)
	}
	return llm.Config{
		Model:         m,
		OperationType: operationTypeChat,
	}
}
