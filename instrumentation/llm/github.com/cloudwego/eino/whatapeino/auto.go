package whatapeino

import (
	"context"
	"net/http"
	"sync"

	einoclaude "github.com/cloudwego/eino-ext/components/model/claude"
	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	whataphttp "github.com/whatap/go-api/instrumentation/net/http/whataphttp"
)

// §282 작업4 — model name capture. eino stores the model name in the
// constructor config (einoopenai.ChatModelConfig.Model / einoclaude.Config.Model),
// NOT in the per-call options, and the response (schema.ResponseMeta) carries
// no model field. So a call-site wrapper that only reads call options cannot
// learn the model. The §254 constructor helpers know cfg.Model, so they record
// it here keyed by the returned concrete model instance; configFor looks it up
// (call-time model.WithModel still takes precedence when supplied).
//
// Keys are long-lived chat-model singletons (typically built once at startup),
// so the unbounded sync.Map is not a practical leak. Manual WrapChatModel users
// who build the model via the upstream constructor are simply absent from the
// map — model stays whatever WithModel provides (no regression).
var modelNameByInstance sync.Map // map[model.BaseChatModel]string

func rememberModelName(m model.BaseChatModel, name string) {
	if m == nil || name == "" {
		return
	}
	modelNameByInstance.Store(m, name)
}

func lookupModelName(m model.BaseChatModel) string {
	if m == nil {
		return ""
	}
	if v, ok := modelNameByInstance.Load(m); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// §254 auto-inject helpers — install a wrapped RoundTripper on the eino-ext
// ChatModelConfig.HTTPClient so the real OpenAI / Claude endpoint URL is
// captured by httpc.Start (§267 RoundTrip single-entry-point design).
//
// Each helper takes the same arguments as the wrapped einoopenai /
// einoclaude.NewChatModel, copies the config struct, ensures HTTPClient
// holds a wrapped Transport, then forwards to the upstream constructor.
// Returned concrete type matches the upstream — user variable declarations
// stay valid.

// NewOpenAIChatModel mirrors einoopenai.NewChatModel with auto-injected
// RoundTrip wrap on cfg.HTTPClient. Auto-inject rule rewrites
// `einoopenai.NewChatModel(ctx, cfg)` to
// `whatapeino.NewOpenAIChatModel(ctx, cfg)`.
func NewOpenAIChatModel(ctx context.Context, cfg *einoopenai.ChatModelConfig) (*einoopenai.ChatModel, error) {
	if cfg == nil {
		return einoopenai.NewChatModel(ctx, cfg)
	}
	cfgCopy := *cfg
	cfgCopy.HTTPClient = ensureWrappedHTTPClient(cfgCopy.HTTPClient)
	m, err := einoopenai.NewChatModel(ctx, &cfgCopy)
	if err == nil {
		rememberModelName(m, cfgCopy.Model)
	}
	return m, err
}

// NewClaudeChatModel mirrors einoclaude.NewChatModel with auto-injected
// RoundTrip wrap on cfg.HTTPClient. Auto-inject rule rewrites
// `einoclaude.NewChatModel(ctx, cfg)` to
// `whatapeino.NewClaudeChatModel(ctx, cfg)`.
func NewClaudeChatModel(ctx context.Context, cfg *einoclaude.Config) (*einoclaude.ChatModel, error) {
	if cfg == nil {
		return einoclaude.NewChatModel(ctx, cfg)
	}
	cfgCopy := *cfg
	cfgCopy.HTTPClient = ensureWrappedHTTPClient(cfgCopy.HTTPClient)
	m, err := einoclaude.NewChatModel(ctx, &cfgCopy)
	if err == nil {
		rememberModelName(m, cfgCopy.Model)
	}
	return m, err
}

// ensureWrappedHTTPClient returns an *http.Client whose Transport is wrapped
// by whataphttp.NewRoundTrip. Behaviour:
//   - hc == nil → fresh wrapped *http.Client
//   - hc.Transport is *WrapRoundTrip → return original unchanged (idempotent)
//   - otherwise → return a NEW *http.Client sharing hc's other fields
//     (Timeout / CheckRedirect / Jar) but with a wrapped Transport. This
//     avoids mutating the caller's original *http.Client (eino-ext config
//     may be reused across multiple ChatModel instances).
func ensureWrappedHTTPClient(hc *http.Client) *http.Client {
	if hc == nil {
		return &http.Client{Transport: whataphttp.NewLLMRoundTrip(context.Background(), http.DefaultTransport)}
	}
	if _, alreadyWrapped := hc.Transport.(*whataphttp.WrapRoundTrip); alreadyWrapped {
		return hc
	}
	clone := *hc
	clone.Transport = whataphttp.NewLLMRoundTrip(context.Background(), hc.Transport)
	return &clone
}
