// Package llm — user-facing manual API for LLM monitoring (§267 redesign).
//
// Conceptual model:
//
//   - HTTPC step is the single trace entry. Whether the call is to OpenAI, an
//     internal MSA, or any other API, it travels through httpc.Start / End.
//   - llm.Start does NOT call httpc.Start. It registers a pending LLMState on
//     the returned context. The next httpc.Start in the call chain (typically
//     from a wrapped RoundTripper inside the SDK) finds the pending state,
//     attaches it to the HttpcCtx, and fills state.URL with the real request
//     URL. httpc.End publishes both the HTTP step and the LLM metrics.
//   - When llm_enabled is on AND the URL matches a known LLM host, httpc.Start
//     auto-attaches an LLMState even without llm.Start (URL pattern matching).
//   - llm.Bind is for the advanced case where the caller manages the
//     httpc.Start / End lifecycle explicitly; it attaches LLMState directly
//     onto an existing HttpcCtx and uses that hc's URL.
//   - llm_tx_status is published once at trace.End for the whole transaction.
//
// Manual usage (single LLM call inside an HTTP request handler):
//
//	ctx, step := llm.Start(ctx, llm.Config{
//	    Provider: "openai", Model: "gpt-4o", OperationType: "chat",
//	})
//	defer step.End()
//
//	step.AddSystemMessage(systemPrompt)
//	step.AddInputMessage(userPrompt)
//	resp, err := openaiClient.CreateChatCompletion(ctx, req)   // RoundTrip captures URL
//	if err != nil {
//	    step.SetError(err, llm.ErrorTypeAPI)
//	    return
//	}
//	step.SetTokens(llm.Tokens{Input: 120, Output: 200})
//	step.AddOutputMessage(resp.Choices[0].Message.Content)
//
// NOTE: the returned ctx MUST be passed to the SDK call — the pending LLMState
// travels on ctx. If the SDK's HTTP transport is not wrapped with
// whataphttp.NewRoundTrip, no httpc step is produced and metrics are skipped.
//
// Adapter / auto-inject usage (httpc lifecycle managed by adapter):
//
//	hc, _ := httpc.Start(ctx, url)
//	defer hc.End(status, "", err)
//	state := llm.Bind(hc, llm.Config{Provider: "ollama", Model: "llama3", OperationType: "chat"})
//	state.SetTokens(...)
//
// See dev-docs/llm-agent/llm-roundtrip-design.md for the design rationale.
package llm

import (
	"context"

	"github.com/whatap/go-api/agent/agent/config"
	agentllm "github.com/whatap/go-api/agent/agent/llm"
	"github.com/whatap/go-api/httpc"
)

// Config — user-facing LLM call metadata.
//
// URL is intentionally absent: the real request URL is captured by the
// wrapped RoundTripper at the time of the actual HTTP call. Caller-supplied
// Provider / OperationType take precedence over the URL-derived defaults.
type Config struct {
	Provider      string
	Model         string
	OperationType string
}

func (c Config) toAgent() agentllm.Config {
	return agentllm.Config{
		Provider:      c.Provider,
		Model:         c.Model,
		OperationType: c.OperationType,
		// URL empty — RoundTrip's httpc.Start fills it via state.SetURL.
	}
}

// Tokens — re-exported.
type Tokens = agentllm.Tokens

// Cost — re-exported.
type Cost = agentllm.Cost

// State — re-exported. Returned by Bind for adapter / auto-inject paths.
type State = agentllm.LLMState

// Error type constants.
const (
	ErrorTypeAPI     = agentllm.ErrorTypeAPI
	ErrorTypeProgram = agentllm.ErrorTypeProgram
)

// Start opens an LLM call. The returned context carries a pending LLMState
// that the next httpc.Start in this call chain attaches to its HttpcCtx —
// typically through a wrapped RoundTripper inside the SDK. Caller MUST pass
// the returned ctx to the SDK call.
//
// step.End() is a defer-friendly no-op when the RoundTripper already closed
// the HTTPC step (the common case). It exists so the call site reads like
// other Go resource patterns.
func Start(ctx context.Context, cfg Config) (context.Context, *Step) {
	if !config.GetConfig().LLMMode {
		return ctx, &Step{LLMState: agentllm.DisabledState()}
	}
	state := agentllm.NewLLMState(cfg.toAgent())
	state.MarkManualPublish()
	ctx = agentllm.RegisterPending(ctx, state)
	return ctx, &Step{LLMState: state}
}

// Bind attaches an LLMState onto a HttpcCtx that the caller already started.
// Used by adapters and auto-inject rules when they own the httpc lifecycle.
// The URL is taken from hc.Url — caller does not pass it again.
func Bind(hc *httpc.HttpcCtx, cfg Config) *State {
	if hc == nil || !config.GetConfig().LLMMode {
		return agentllm.DisabledState()
	}
	state := agentllm.NewLLMState(cfg.toAgent())
	state.MarkManualPublish()
	hc.Extra = state
	state.SetURL(hc.Url)
	return state
}
