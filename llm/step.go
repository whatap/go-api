package llm

import (
	agentllm "github.com/whatap/go-api/agent/agent/llm"
)

// Step is the manual user-facing handle returned by Start. It embeds the
// shared LLMState so user code can call AddInputMessage / SetTokens / ...
// directly. The httpc step lifecycle is owned by the wrapped RoundTripper —
// Step.End() is a no-op that exists for symmetry with `defer` patterns.
//
// Concurrency: a single Step is owned by a single goroutine, mirroring the
// `with` block semantics in python-apm. Mutating accumulators concurrently
// is not safe.
type Step struct {
	*agentllm.LLMState
}

// End publishes the accumulated LLM step — LogSinkPack dispatch, MeterLLM
// updates, and tx_status accumulation. The wrapped RoundTripper's httpc.End
// has already populated stepId / latency / status / err on the state, so
// adapter code that runs *after* the SDK call (filling tokens, finish
// reason, etc.) is captured here before publish.
//
// Idempotent — calling End multiple times or in parallel publishes once
// (state.endOnce). Auto-attached states (URL match without llm.Start) skip
// the manualPublish flag and are published from httpc.End directly; a
// stray End call on such a state is a no-op via endOnce.
func (s *Step) End() {
	if s == nil || s.LLMState == nil {
		return
	}
	s.LLMState.Publish()
}
