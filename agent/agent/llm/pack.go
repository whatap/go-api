// Package llm — LLM 모니터링 사용자 facing 패키지 (§251 Phase 2~4).
//
// Phase 2 — LogSinkPack 7 종 (#LlmCallLog 카테고리) 데이터 모델 + 송출 인프라.
// python-apm whatap/llm/log_sink_packs/ 와 의도된 차이 (Go 관용 + go-api 의 logsink/zip 인프라 활용)는
// dev-docs/llm-agent/phase2-design.md §9 참조.
package llm

import (
	"strings"
	"sync"
)

const (
	// Category — LogSinkPack `#LlmCallLog` 카테고리 (사양 + python LlmLogSinkPack.CATEGORY).
	Category = "#LlmCallLog"

	LogTypeStepStatus    = "llm_step_status"
	LogTypeTxStatus      = "llm_tx_status"
	LogTypeSystemMessage = "system_message"
	LogTypeInputMessage  = "input_message"
	LogTypeOutputMessage = "output_message"
	LogTypeTool          = "tool"
	LogTypeToolResult    = "tool_result"
)

// TX_SUMMARY_TOKEN_FIELDS — LlmTxStatus.Accumulate 가 합산하는 토큰 필드 11종.
// python whatap/llm/log_sink_packs/llm_tx_status.py:4-11 동등.
var txSummaryTokenFields = []string{
	"input_tokens", "output_tokens", "total_tokens_count",
	"cached_tokens", "reasoning_tokens",
	"audio_input_tokens", "audio_output_tokens",
	"accepted_prediction_tokens", "rejected_prediction_tokens",
	"cache_creation_input_tokens", "cache_read_input_tokens",
	"embedding_count",
}

// InferProviderURL — "https://api.openai.com/v1/chat/completions"
//   → ("api.openai.com", "/v1/chat/completions")
//
// Used by NewLLMState to auto-fill Provider when Config.Provider is empty.
func InferProviderURL(httpcURL string) (provider, path string) {
	if httpcURL == "" {
		return "", ""
	}
	s := httpcURL
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	if i := strings.Index(s, "/"); i >= 0 {
		return s[:i], s[i:]
	}
	return s, ""
}

// LlmLogSinkPack — 모든 7 종 LogSinkPack 의 공통 base.
// 6 식별 필드 (txid/step_id/index/provider/url/operation_type) 보유.
// python whatap/llm/log_sink_packs/llm_log_sink_pack.py 동등.
type LlmLogSinkPack struct {
	LLMLogType    string
	Txid          string
	StepID        string
	Index         int64
	Provider      string
	URL           string
	OperationType string
}

// ── llm_step_status ──

// LlmStepStatus — LLM API 1:1 (python whatap/llm/log_sink_packs/llm_step_status.py 동등).
//
// 메시지 텍스트 6 필드 (SystemTexts/PromptText/CompletionText/ReasoningText/ToolCallsText/ToolResultsText)는
// processStep 자동 dispatch 흐름 위함 (sender.go).
type LlmStepStatus struct {
	LlmLogSinkPack
	Model        string
	Stream       bool
	Success      bool
	FinishReason string
	Features     string
	Temperature  *float64
	Error        string
	ErrorType    string

	// tokens (모두 nullable)
	InputTokens              *int64
	OutputTokens             *int64
	TotalTokensCount         *int64
	CachedTokens             *int64
	ReasoningTokens          *int64
	AudioInputTokens         *int64
	AudioOutputTokens        *int64
	AcceptedPredictionTokens *int64
	RejectedPredictionTokens *int64
	EmbeddingCount           *int64
	Dimensions               *int64
	Similarity               *float64
	CacheCreationInputTokens *int64
	CacheReadInputTokens     *int64

	// cost (USD, nullable)
	Cost              *float64
	InputCost         *float64
	OutputCost        *float64
	CachedCost        *float64
	CacheCreationCost *float64

	// latency (ms)
	StartTime int64 // 비공개적 의미 (사용자 facing X)
	Ttft      *float64
	Latency   *float64

	// 메시지 텍스트 — processStep 이 자동 dispatch (system_message N + input + output + tool + tool_result)
	SystemTexts     []string
	PromptText      string
	CompletionText  string
	ReasoningText   string
	ToolCallsText   string
	ToolResultsText string
}

func NewLlmStepStatus() *LlmStepStatus {
	p := &LlmStepStatus{
		LlmLogSinkPack: LlmLogSinkPack{
			LLMLogType:    LogTypeStepStatus,
			OperationType: "unknown",
		},
	}
	return p
}

// SetError — python set_error(err, error_type) 동등.
func (this *LlmStepStatus) SetError(errMsg, errType string) {
	this.Error = errMsg
	this.ErrorType = errType
}

// SetTokens — python set_tokens(token_dict) 동등 (hasattr 패턴 → 명시 switch).
func (this *LlmStepStatus) SetTokens(tokens map[string]int64) {
	for k, v := range tokens {
		v := v // local copy for &v
		switch k {
		case "input_tokens":
			this.InputTokens = &v
		case "output_tokens":
			this.OutputTokens = &v
		case "total_tokens_count":
			this.TotalTokensCount = &v
		case "cached_tokens":
			this.CachedTokens = &v
		case "reasoning_tokens":
			this.ReasoningTokens = &v
		case "audio_input_tokens":
			this.AudioInputTokens = &v
		case "audio_output_tokens":
			this.AudioOutputTokens = &v
		case "accepted_prediction_tokens":
			this.AcceptedPredictionTokens = &v
		case "rejected_prediction_tokens":
			this.RejectedPredictionTokens = &v
		case "cache_creation_input_tokens":
			this.CacheCreationInputTokens = &v
		case "cache_read_input_tokens":
			this.CacheReadInputTokens = &v
		case "embedding_count":
			this.EmbeddingCount = &v
		case "dimensions":
			this.Dimensions = &v
		}
	}
}

// CalculateCost — Phase 2 stub. 정식 구현은 §252 어댑터 또는 별도 issue (pricing 데이터 큰 양).
func (this *LlmStepStatus) CalculateCost() {
	// TODO §252 — pricing.go 이관 시 구현
}

// tokenByName — name 으로 토큰 필드 lookup (TxStatus.Accumulate 에서 사용).
func (this *LlmStepStatus) tokenByName(name string) *int64 {
	switch name {
	case "input_tokens":
		return this.InputTokens
	case "output_tokens":
		return this.OutputTokens
	case "total_tokens_count":
		return this.TotalTokensCount
	case "cached_tokens":
		return this.CachedTokens
	case "reasoning_tokens":
		return this.ReasoningTokens
	case "audio_input_tokens":
		return this.AudioInputTokens
	case "audio_output_tokens":
		return this.AudioOutputTokens
	case "accepted_prediction_tokens":
		return this.AcceptedPredictionTokens
	case "rejected_prediction_tokens":
		return this.RejectedPredictionTokens
	case "cache_creation_input_tokens":
		return this.CacheCreationInputTokens
	case "cache_read_input_tokens":
		return this.CacheReadInputTokens
	case "embedding_count":
		return this.EmbeddingCount
	}
	return nil
}

// ── llm_tx_status ──

// LlmTxStatus — 트랜잭션 1:1 집계 (python LlmTxStatus 동등).
type LlmTxStatus struct {
	LlmLogSinkPack
	FirstStepID    string
	LastStepID     string
	CallCount      int64
	ErrorCount     int64
	Latency        float64
	Cost           float64
	InputCost      float64
	OutputCost     float64
	Models         map[string]struct{}
	OperationTypes map[string]struct{}
	Providers      map[string]struct{}
	TokenSums      map[string]int64

	mu sync.Mutex
}

func NewLlmTxStatus() *LlmTxStatus {
	return &LlmTxStatus{
		LlmLogSinkPack: LlmLogSinkPack{
			LLMLogType:    LogTypeTxStatus,
			OperationType: "unknown",
		},
		Models:         map[string]struct{}{},
		OperationTypes: map[string]struct{}{},
		Providers:      map[string]struct{}{},
		TokenSums:      map[string]int64{},
	}
}

// Accumulate — python LlmTxStatus.accumulate(pack) 동등.
// step_status pack 의 데이터를 트랜잭션 요약에 누적.
func (this *LlmTxStatus) Accumulate(pk *LlmStepStatus) {
	this.mu.Lock()
	defer this.mu.Unlock()

	this.CallCount++
	if !pk.Success {
		this.ErrorCount++
	}
	for _, f := range txSummaryTokenFields {
		if v := pk.tokenByName(f); v != nil {
			this.TokenSums[f] += *v
		}
	}
	if pk.Latency != nil {
		this.Latency += *pk.Latency
	}
	if pk.Cost != nil {
		this.Cost += *pk.Cost
	}
	if pk.InputCost != nil {
		this.InputCost += *pk.InputCost
	}
	if pk.OutputCost != nil {
		this.OutputCost += *pk.OutputCost
	}
	if pk.StepID != "" {
		if this.FirstStepID == "" {
			this.FirstStepID = pk.StepID
		}
		this.LastStepID = pk.StepID
	}
	if pk.Provider != "" {
		this.Providers[pk.Provider] = struct{}{}
	}
	if pk.Model != "" {
		this.Models[pk.Model] = struct{}{}
	}
	if pk.OperationType != "" && pk.OperationType != "unknown" {
		this.OperationTypes[pk.OperationType] = struct{}{}
	}
}
