package llm

import (
	"context"
)

// pendingKey is the context.Value key for a pending *LLMState. Unexported so
// only this package's RegisterPending / TakePending can read/write it.
type pendingKey struct{}

// RegisterPending attaches state as the pending LLMState on ctx and returns a
// child context. The pending state is consumed by httpc.Start (the first call
// that finds it attaches it to the HttpcCtx and updates state.URL with the
// real request URL).
//
// If the parent ctx already carries a pending LLMState, RegisterPending is a
// no-op and returns the parent unchanged — first registration wins (idempotent
// against double-wrap: e.g. manual llm.Start + auto-inject rule on the same
// call site).
//
// Pass a nil or disabled state to skip registration (returns parent ctx).
func RegisterPending(ctx context.Context, state *LLMState) context.Context {
	if ctx == nil || state == nil || state.disabled {
		return ctx
	}
	if existing, ok := ctx.Value(pendingKey{}).(*LLMState); ok && existing != nil {
		return ctx
	}
	return context.WithValue(ctx, pendingKey{}, state)
}

// TakePending returns the pending LLMState on ctx, if any. Returns nil if no
// pending state is registered. The state remains attached to the context — a
// later TakePending in the same call chain returns the same instance (no
// "consume on first read"). httpc.Start should call this once and copy the
// pointer to HttpcCtx.Extra.
func TakePending(ctx context.Context) *LLMState {
	if ctx == nil {
		return nil
	}
	state, _ := ctx.Value(pendingKey{}).(*LLMState)
	return state
}

// SetURL updates the pack URL after the real HTTP request URL is known. Called
// from httpc.Start when a pending state is attached. Provider/operation_type
// derived from the URL via InferProviderURL fill in only when still empty so
// caller-supplied Config values win.
func (s *LLMState) SetURL(url string) {
	if s == nil || s.disabled || url == "" {
		return
	}
	host, path := InferProviderURL(url)
	if s.pack.Provider == "" {
		s.pack.Provider = host
	}
	if s.pack.URL == "" {
		s.pack.URL = path
	} else if s.pack.URL == url {
		s.pack.URL = path
	}
	// OperationType defaults to "unknown" in NewLlmStepStatus, so also treat
	// that as unset for the purposes of URL-derived override.
	if s.pack.OperationType == "" || s.pack.OperationType == "unknown" {
		if m, ok := MatchURL(url); ok && m.OperationType != "" {
			s.pack.OperationType = m.OperationType
		}
	}
}
