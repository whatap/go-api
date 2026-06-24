package llm

import (
	"sync"

	"github.com/whatap/golib/util/dateutil"
)

// Config — supplied at llm.Start / llm.Bind. Provider may be empty; auto-fill
// via InferProviderURL when URL is set.
type Config struct {
	Provider      string
	Model         string
	OperationType string
	URL           string
}

// Tokens — supplied via State.SetTokens. Only non-zero fields propagate.
type Tokens struct {
	Input              int64
	Output             int64
	Total              int64
	Cached             int64
	Reasoning          int64
	AudioInput         int64
	AudioOutput        int64
	AcceptedPrediction int64
	RejectedPrediction int64
	CacheCreationInput int64
	CacheReadInput     int64
	EmbeddingCount     int64
	Dimensions         int64
	Similarity         float64
}

// Cost — per-call USD cost. Pricing tables (model→cost) are out of scope —
// caller supplies values directly.
type Cost struct {
	Total         float64
	Input         float64
	Output        float64
	Cached        float64
	CacheCreation float64
}

// Error type constants for State.SetError.
const (
	ErrorTypeAPI     = "api_error"
	ErrorTypeProgram = "program_error"
)

// LLMState holds per-call data accumulated through user-facing methods. It is
// stored on httpc.HttpcCtx.Extra and finalized in HandleHttpcEnd.
//
// Single-goroutine mutation expected (matches python-apm's `with` block
// semantics); concurrent End calls are serialized via endOnce.
type LLMState struct {
	disabled bool

	pack *LlmStepStatus

	startMs       int64
	firstTokenMs  int64
	streamRequest bool

	statusCode int
	terminalErr error

	// §267 — manualPublish=true → Publish at user-side step.End() so
	// adapters can add response metadata (tokens / finish_reason / ...)
	// after the SDK call returns. false → RoundTrip's httpc.End publishes
	// immediately (auto-attached URL match has no user-side End).
	manualPublish bool
	// §267 — captured at httpc.End so a later Publish (from step.End)
	// can accumulate into the same trace's tx_status slot.
	txSlot *interface{}

	systemTexts     []string
	promptText      string
	completionText  string
	reasoningText   string
	toolCallsText   string
	toolResultsText string

	endOnce sync.Once
}

var disabledState = &LLMState{disabled: true, pack: NewLlmStepStatus()}

// DisabledState returns a sentinel LLMState whose mutators are no-ops. Used
// when LLMMode is off so user code does not need nil checks.
func DisabledState() *LLMState { return disabledState }

// NewLLMState creates a state seeded with cfg fields.
func NewLLMState(cfg Config) *LLMState {
	s := &LLMState{
		pack:    NewLlmStepStatus(),
		startMs: dateutil.SystemNow(),
	}
	s.pack.Provider = cfg.Provider
	s.pack.Model = cfg.Model
	s.pack.URL = cfg.URL
	if cfg.OperationType != "" {
		s.pack.OperationType = cfg.OperationType
	}
	if s.pack.Provider == "" && cfg.URL != "" {
		host, path := InferProviderURL(cfg.URL)
		s.pack.Provider = host
		if s.pack.URL == cfg.URL {
			s.pack.URL = path
		}
	}
	s.pack.StartTime = s.startMs
	return s
}

// ── User-facing accumulators ──

func (s *LLMState) AddSystemMessage(text string) {
	if s.disabled || text == "" {
		return
	}
	s.systemTexts = append(s.systemTexts, text)
}

func (s *LLMState) AddInputMessage(text string) {
	if s.disabled || text == "" {
		return
	}
	if s.promptText == "" {
		s.promptText = text
	} else {
		s.promptText += "\n" + text
	}
}

func (s *LLMState) AddOutputMessage(text string) {
	if s.disabled || text == "" {
		return
	}
	if s.completionText == "" {
		s.completionText = text
	} else {
		s.completionText += "\n" + text
	}
}

func (s *LLMState) AddReasoning(text string) {
	if s.disabled || text == "" {
		return
	}
	if s.reasoningText == "" {
		s.reasoningText = text
	} else {
		s.reasoningText += "\n" + text
	}
}

// AddTool records a tool/function call. Either pass the full JSON-array
// payload (preferred, mirrors python tool_calls_text) or pass an empty
// payload with name+argumentsJSON to build a minimal entry.
func (s *LLMState) AddTool(payload, name, argumentsJSON string) {
	if s.disabled {
		return
	}
	if s.toolCallsText != "" {
		s.toolCallsText += "\n"
	}
	if payload != "" {
		s.toolCallsText += payload
		return
	}
	s.toolCallsText += `[{"function":{"name":"` + name + `","arguments":"` + argumentsJSON + `"}}]`
}

func (s *LLMState) AddToolResult(text string) {
	if s.disabled || text == "" {
		return
	}
	if s.toolResultsText == "" {
		s.toolResultsText = text
	} else {
		s.toolResultsText += "\n" + text
	}
}

// ── Setters ──

func (s *LLMState) SetTokens(t Tokens) {
	if s.disabled {
		return
	}
	s.pack.SetTokens(tokensToMap(t))
	if t.Similarity != 0 {
		v := t.Similarity
		s.pack.Similarity = &v
	}
}

func (s *LLMState) SetCost(c Cost) {
	if s.disabled {
		return
	}
	if c.Total != 0 {
		v := c.Total
		s.pack.Cost = &v
	}
	if c.Input != 0 {
		v := c.Input
		s.pack.InputCost = &v
	}
	if c.Output != 0 {
		v := c.Output
		s.pack.OutputCost = &v
	}
	if c.Cached != 0 {
		v := c.Cached
		s.pack.CachedCost = &v
	}
	if c.CacheCreation != 0 {
		v := c.CacheCreation
		s.pack.CacheCreationCost = &v
	}
}

func (s *LLMState) SetTemperature(t float64) {
	if s.disabled {
		return
	}
	v := t
	s.pack.Temperature = &v
}

func (s *LLMState) SetFinishReason(reason string) {
	if s.disabled {
		return
	}
	s.pack.FinishReason = reason
}

func (s *LLMState) SetFeatures(features string) {
	if s.disabled {
		return
	}
	s.pack.Features = features
}

func (s *LLMState) SetStatusCode(code int) {
	if s.disabled {
		return
	}
	s.statusCode = code
}

// SetError records an error and its classification. Subsequent calls
// overwrite — matches python-apm semantics.
func (s *LLMState) SetError(err error, errorType string) {
	if s.disabled || err == nil {
		return
	}
	s.pack.SetError(err.Error(), errorType)
	s.terminalErr = err
}

func (s *LLMState) MarkStream() {
	if s.disabled {
		return
	}
	s.streamRequest = true
	s.pack.Stream = true
}

func (s *LLMState) RecordFirstToken() {
	if s.disabled || s.firstTokenMs != 0 {
		return
	}
	s.firstTokenMs = dateutil.SystemNow()
}

// Disabled reports whether the state is the sentinel returned when LLMMode
// is off. Public so the wrapper package can decide whether to call End.
func (s *LLMState) Disabled() bool { return s.disabled }

// StatusCode returns the recorded status code (0 if not set).
func (s *LLMState) StatusCode() int { return s.statusCode }

// TerminalErr returns the error captured by SetError (nil if not set).
func (s *LLMState) TerminalErr() error { return s.terminalErr }

// IsStream reports whether MarkStream was called.
func (s *LLMState) IsStream() bool { return s.streamRequest }

// MarkManualPublish — set by llm.Start / llm.Bind so HandleHttpcEnd defers
// the LogSinkPack / Meter / tx_status dispatch to Publish (invoked by
// Step.End after the adapter / user code has populated response metadata).
// Auto-attached states (URL match, no user code) leave this false and the
// httpc.End path publishes immediately.
func (s *LLMState) MarkManualPublish() {
	if s == nil || s.disabled {
		return
	}
	s.manualPublish = true
}

// IsManualPublish reports whether the state defers publish to Step.End.
func (s *LLMState) IsManualPublish() bool {
	if s == nil {
		return false
	}
	return s.manualPublish
}

// Publish — finalize accumulated state and dispatch one LogSinkPack +
// publishMeters + tx_status accumulation. Idempotent (endOnce) — safe to
// call from both httpc.End (auto path) and Step.End (manual path).
//
// Callers must have populated stepId / latency / status / err via the
// shared HandleHttpcEnd preamble before invoking Publish.
func (s *LLMState) Publish() {
	if s == nil || s.disabled {
		return
	}
	s.endOnce.Do(func() {
		// §267 stream fallback — firstTokenMs is set by RecordFirstToken
		// inside the stream-reader goroutine, which fires AFTER the
		// wrapped RoundTripper's httpc.End. HandleHttpcEnd therefore sees
		// firstTokenMs==0 and skips Ttft. Recompute here so streaming
		// metrics (ttft_sum / ttft_count / tpot_*) populate correctly.
		if s.firstTokenMs != 0 && s.pack.Ttft == nil {
			v := float64(s.firstTokenMs - s.startMs)
			s.pack.Ttft = &v
		}
		// §267 stream latency fallback — transport.RoundTrip returns after
		// the SSE header is read but before the body chunks arrive, so the
		// RoundTripper's recorded latency (~response-header time) can be
		// SMALLER than the time we observe the first token. When that
		// happens, `latency - ttft` is negative and TPOT is skipped. For
		// streaming calls we recompute latency from now-startMs only if
		// the transport-recorded latency is too small to be plausible
		// (≤ ttft), so realistic-latency tests (HandleHttpcEnd with a
		// big latency value) keep their authoritative latency intact.
		if s.streamRequest && s.pack.Ttft != nil {
			recompute := s.pack.Latency == nil || *s.pack.Latency <= *s.pack.Ttft
			if recompute {
				v := float64(dateutil.SystemNow() - s.startMs)
				s.pack.Latency = &v
			}
		}
		// flush accumulated text into pack (deferred so adapters can keep
		// appending after the HTTPC step is closed).
		s.pack.SystemTexts = s.systemTexts
		s.pack.PromptText = s.promptText
		s.pack.CompletionText = s.completionText
		s.pack.ReasoningText = s.reasoningText
		s.pack.ToolCallsText = s.toolCallsText
		s.pack.ToolResultsText = s.toolResultsText
		s.pack.Success = s.pack.Error == ""

		publishMeters(s)

		if s.txSlot != nil {
			if *s.txSlot == nil {
				*s.txSlot = NewLlmTxStatus()
			}
			if tx, ok := (*s.txSlot).(*LlmTxStatus); ok {
				if tx.Txid == "" {
					tx.Txid = s.pack.Txid
				}
				tx.Accumulate(s.pack)
			}
		}

		DispatchStep(s.pack)
	})
}

// ── helpers ──

func tokensToMap(t Tokens) map[string]int64 {
	m := map[string]int64{}
	if t.Input != 0 {
		m["input_tokens"] = t.Input
	}
	if t.Output != 0 {
		m["output_tokens"] = t.Output
	}
	if t.Total != 0 {
		m["total_tokens_count"] = t.Total
	}
	if t.Cached != 0 {
		m["cached_tokens"] = t.Cached
	}
	if t.Reasoning != 0 {
		m["reasoning_tokens"] = t.Reasoning
	}
	if t.AudioInput != 0 {
		m["audio_input_tokens"] = t.AudioInput
	}
	if t.AudioOutput != 0 {
		m["audio_output_tokens"] = t.AudioOutput
	}
	if t.AcceptedPrediction != 0 {
		m["accepted_prediction_tokens"] = t.AcceptedPrediction
	}
	if t.RejectedPrediction != 0 {
		m["rejected_prediction_tokens"] = t.RejectedPrediction
	}
	if t.CacheCreationInput != 0 {
		m["cache_creation_input_tokens"] = t.CacheCreationInput
	}
	if t.CacheReadInput != 0 {
		m["cache_read_input_tokens"] = t.CacheReadInput
	}
	if t.EmbeddingCount != 0 {
		m["embedding_count"] = t.EmbeddingCount
	}
	if t.Dimensions != 0 {
		m["dimensions"] = t.Dimensions
	}
	return m
}
