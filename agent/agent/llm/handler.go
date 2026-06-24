package llm

import (
	"net/url"
	"strconv"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/counter/meter"
)

// hostFromURL — best-effort host extraction for AttachForced fallback.
// Returns empty string on parse failure or scheme-less URLs.
func hostFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u == nil {
		return ""
	}
	return u.Host
}

// MaybeAttachAuto — invoked from httpc.Start. Returns a fresh LLMState if the
// URL matches a known LLM provider AND LLMMode is on; otherwise nil.
//
// This is the "automatic" entry point — the user wrote no LLM-specific code
// but happens to call a known LLM API through an instrumented HTTP client.
// The state is attached to the HttpcCtx and meta (model/tokens/...) is
// expected to be filled in by an adapter (§252) or auto-inject rule (§254).
func MaybeAttachAuto(url string) *LLMState {
	if !config.GetConfig().LLMMode || url == "" {
		return nil
	}
	m, ok := MatchURL(url)
	if !ok {
		return nil
	}
	cfg := Config{
		Provider:      m.Provider,
		OperationType: m.OperationType,
		URL:           m.URL,
	}
	return NewLLMState(cfg)
}

// AttachForced — like MaybeAttachAuto but bypasses URL host matching. Used by
// LLM SDK adapters whose wrapped transport guarantees every call is an LLM
// API call regardless of URL (e.g. mock servers, self-hosted LLM endpoints).
// Returns nil if LLMMode is off or url is empty.
//
// §254 Step 5 — invoked from httpc.StartLLM when an adapter-owned RoundTrip
// wraps the call. URL-based Provider extraction is best-effort; the URL
// match table is consulted first so known providers still get their canonical
// label, then falls back to the host portion.
func AttachForced(url string) *LLMState {
	if !config.GetConfig().LLMMode || url == "" {
		return nil
	}
	cfg := Config{URL: url}
	if m, ok := MatchURL(url); ok {
		cfg.Provider = m.Provider
		cfg.OperationType = m.OperationType
		cfg.URL = m.URL
	} else if h := hostFromURL(url); h != "" {
		cfg.Provider = h
	}
	return NewLLMState(cfg)
}

// HandleHttpcEnd — invoked from httpc.End when an LLMState is attached.
// Prepares HTTPC-derived metadata (latency / status / err / stepId) on the
// state, then either publishes immediately (auto path) or defers publish to
// Step.End (manual / adapter path — §267).
//
// Parameters:
//   state         — non-nil, attached on httpc.HttpcCtx.Extra
//   txid          — TraceCtx.Txid (0 outside a trace)
//   txLLMTxSlot   — pointer to TraceCtx.LLMTx (nil outside a trace).
//                   Stored on the state so a deferred Publish (from
//                   Step.End) accumulates into the same slot.
//   httpcStepID   — HttpcStepX.StepId (the step published to trace profile)
//   httpcElapsed  — milliseconds — used as latency
//   statusCode    — HTTP status (0 if unknown)
//   err           — terminal error (may be nil)
func HandleHttpcEnd(
	state *LLMState,
	txid int64,
	txLLMTxSlot *interface{},
	httpcStepID int64,
	httpcElapsed int64,
	statusCode int,
	err error,
) {
	if state == nil || state.disabled {
		return
	}
	if !config.GetConfig().LLMMode {
		return
	}

	// finalize timings
	if httpcElapsed >= 0 {
		v := float64(httpcElapsed)
		state.pack.Latency = &v
	}
	if state.firstTokenMs != 0 {
		v := float64(state.firstTokenMs - state.startMs)
		state.pack.Ttft = &v
	}

	// status overrides — explicit SetStatusCode wins, else use httpc status
	if state.statusCode == 0 {
		state.statusCode = statusCode
	}

	// terminal error precedence — explicit SetError wins
	if err != nil && state.terminalErr == nil {
		state.SetError(err, ErrorTypeAPI)
	}

	// step_id reuse — HttpcStepX.StepId is the trace step id
	if httpcStepID != 0 {
		state.pack.StepID = strconv.FormatInt(httpcStepID, 10)
	}
	if txid != 0 {
		state.pack.Txid = strconv.FormatInt(txid, 10)
	}

	// §267 — remember the tx slot so Step.End can accumulate later.
	state.txSlot = txLLMTxSlot

	// §267 — manual path defers Publish; auto path (URL match, no user
	// llm.Start) publishes immediately.
	if !state.manualPublish {
		state.Publish()
	}
}

func publishMeters(s *LLMState) {
	m := meter.GetInstanceMeterLLM()
	prov, model, opType, url := s.pack.Provider, s.pack.Model, s.pack.OperationType, s.pack.URL
	isError := !s.pack.Success

	m.OnStart(prov, model, opType, url)

	if s.pack.Latency != nil {
		m.AddPerf(prov, model, opType, url, *s.pack.Latency,
			s.pack.Ttft, s.pack.OutputTokens, s.streamRequest, isError)
	}
	if s.pack.Features != "" {
		m.AddFeature(prov, model, opType, url, splitFeatures(s.pack.Features), isError)
	}
	if s.statusCode != 0 {
		m.AddApiStatus(prov, model, opType, url, s.statusCode)
	}
	switch s.pack.ErrorType {
	case ErrorTypeAPI:
		m.AddApiError(prov, model, opType, url)
	case ErrorTypeProgram:
		m.AddProgramError(prov, model, opType, url)
	}

	tokens := nonZeroTokenCounts(s.pack)
	costs := nonZeroCosts(s.pack)
	if len(tokens) > 0 || len(costs) > 0 {
		m.AddTokenUsage(prov, model, opType, url, tokens, costs, isError, s.streamRequest)
	}

	m.IncrementMeter()
	if isError {
		m.IncrementMeterError()
	}
	m.OnEnd(prov, model, opType, url)
}

// DispatchTraceTxStatus — invoked from trace.End once per transaction. Drains
// the accumulated tx_status and publishes one llm_tx_status pack.
//
// Safe to call with nil/empty slot — zero work in that case.
func DispatchTraceTxStatus(txid int64, slot interface{}) {
	if slot == nil {
		return
	}
	tx, ok := slot.(*LlmTxStatus)
	if !ok || tx == nil || tx.CallCount == 0 {
		return
	}
	if !config.GetConfig().LLMMode {
		return
	}
	if tx.Txid == "" && txid != 0 {
		tx.Txid = strconv.FormatInt(txid, 10)
	}
	DispatchTxStatus(tx)
}

// ── helpers — kept here so state.go stays focused on user-facing setters ──

func nonZeroTokenCounts(p *LlmStepStatus) map[string]int64 {
	m := map[string]int64{}
	add := func(name string, v *int64) {
		if v != nil && *v != 0 {
			m[name] = *v
		}
	}
	add("input_tokens", p.InputTokens)
	add("output_tokens", p.OutputTokens)
	add("total_tokens_count", p.TotalTokensCount)
	add("cached_tokens", p.CachedTokens)
	add("reasoning_tokens", p.ReasoningTokens)
	add("audio_input_tokens", p.AudioInputTokens)
	add("audio_output_tokens", p.AudioOutputTokens)
	add("accepted_prediction_tokens", p.AcceptedPredictionTokens)
	add("rejected_prediction_tokens", p.RejectedPredictionTokens)
	add("cache_creation_input_tokens", p.CacheCreationInputTokens)
	add("cache_read_input_tokens", p.CacheReadInputTokens)
	add("embedding_count", p.EmbeddingCount)
	return m
}

func nonZeroCosts(p *LlmStepStatus) map[string]float64 {
	m := map[string]float64{}
	add := func(name string, v *float64) {
		if v != nil && *v != 0 {
			m[name] = *v
		}
	}
	add("cost", p.Cost)
	add("input_cost", p.InputCost)
	add("output_cost", p.OutputCost)
	add("cached_cost", p.CachedCost)
	add("cache_creation_cost", p.CacheCreationCost)
	return m
}

func splitFeatures(joined string) []string {
	if joined == "" {
		return nil
	}
	out := []string{}
	start := 0
	for i := 0; i < len(joined); i++ {
		if joined[i] == ',' {
			if i > start {
				out = append(out, joined[start:i])
			}
			start = i + 1
		}
	}
	if start < len(joined) {
		out = append(out, joined[start:])
	}
	return out
}
