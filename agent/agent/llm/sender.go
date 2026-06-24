package llm

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/whatap/go-api/agent/agent/config"
	langconf "github.com/whatap/go-api/agent/lang/conf"
	"github.com/whatap/go-api/agent/logsink"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/golib/lang/value"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/queue"
)

// LlmSenderThread — LogSinkPack 7 종 송출 백그라운드 thread.
// agent/logsink/std/TraceLogSenderThread 패턴 그대로 (큐 + langconf observer + ctx + cancel).
type LlmSenderThread struct {
	ctx         context.Context
	cancel      context.CancelFunc
	conf        *config.Config
	queue       *queue.RequestQueue
	logWaitTime int
}

var (
	llmSender    *LlmSenderThread
	llmSenderMtx sync.Mutex
)

// GetInstanceLlmSender — singleton (TraceLogSenderThread 패턴).
func GetInstanceLlmSender() *LlmSenderThread {
	llmSenderMtx.Lock()
	defer llmSenderMtx.Unlock()
	if llmSender != nil {
		return llmSender
	}

	p := new(LlmSenderThread)
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.conf = config.GetConfig()
	p.queue = queue.NewRequestQueue(int(p.conf.LogSinkQueueSize))
	p.logWaitTime = 500

	langconf.AddConfObserver("LlmSenderThread", p)
	go p.run()

	llmSender = p
	return llmSender
}

// Run — langconf.Runnable 인터페이스. shutdown 처리 + 큐 capacity 갱신.
func (this *LlmSenderThread) Run() {
	if this.conf.Shutdown {
		logutil.Infoln("WALOG004-01", "Shutdown LlmSenderThread")
		this.queue.Clear()
		this.cancel()
		return
	}
	this.queue.SetCapacity(int(this.conf.LogSinkQueueSize))
}

func (this *LlmSenderThread) run() {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WALOG004-r", "[LLM] sender recover ", r)
		}
	}()
	for {
		select {
		case <-this.ctx.Done():
			return
		default:
			tmp := this.queue.GetTimeout(this.logWaitTime)
			if tmp == nil {
				continue
			}
			if step, ok := tmp.(*LlmStepStatus); ok {
				processStep(step)
			}
		}
	}
}

// ── 모듈 진입점 ──

// DispatchStep — Phase 4 의 step.End() 시점에 호출. python dispatch_llm_pack 동등.
func DispatchStep(step *LlmStepStatus) {
	if step == nil {
		return
	}
	if !config.GetConfig().LLMMode {
		return
	}
	GetInstanceLlmSender().queue.Put(step)
}

// DispatchTxStatus — Phase 4 의 트랜잭션 종료 hook 에서 호출. python send_tx_status 동등.
func DispatchTxStatus(tx *LlmTxStatus) {
	if tx == nil || tx.CallCount == 0 {
		return
	}
	if !config.GetConfig().LLMMode {
		return
	}
	sendLineLog(buildTxStatusLineLog(tx))
}

// processStep — python _process_pack 동등 (llm_log_sink_task.py:112-138).
// step_status + 메시지 4종 + tool/tool_result 자동 발행.
func processStep(step *LlmStepStatus) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WALOG004-p", "[LLM] processStep recover ", r)
		}
	}()

	// 1. llm_step_status (메인 패킷, content 없음)
	sendLineLog(buildStepStatusLineLog(step))

	// 2. system_message (N 개 텍스트 각각 발행)
	for _, text := range step.SystemTexts {
		sendChunked(&step.LlmLogSinkPack, LogTypeSystemMessage, text, nil)
	}

	// 3. input_message — 비어 있어도 발행 (python 동작)
	sendChunked(&step.LlmLogSinkPack, LogTypeInputMessage, step.PromptText, nil)

	// 4. output_message — reasoning + completion 결합
	sendChunked(&step.LlmLogSinkPack, LogTypeOutputMessage,
		combineOutput(step.ReasoningText, step.CompletionText), nil)

	// 5. tool — tool_calls_text 가 비어있지 않을 때만
	if step.ToolCallsText != "" {
		extra := value.NewMapValue()
		if names, args := parseToolCalls(step.ToolCallsText); names != "" || args != "" {
			if names != "" {
				extra.PutString("function", names)
			}
			if args != "" {
				extra.PutString("arguments", args)
			}
		}
		sendChunked(&step.LlmLogSinkPack, LogTypeTool, step.ToolCallsText, extra)
	}

	// 6. tool_result — tool_results_text 가 비어있지 않을 때만
	if step.ToolResultsText != "" {
		sendChunked(&step.LlmLogSinkPack, LogTypeToolResult, step.ToolResultsText, nil)
	}
}

// ── 사용자 facing dispatch (Phase 4 가 호출 가능 — 직접 메시지 발행 시) ──

func DispatchSystemMessage(c *LlmLogSinkPack, content string) {
	sendChunked(c, LogTypeSystemMessage, content, nil)
}

func DispatchInputMessage(c *LlmLogSinkPack, content string) {
	sendChunked(c, LogTypeInputMessage, content, nil)
}

func DispatchOutputMessage(c *LlmLogSinkPack, completion, reasoning string) {
	sendChunked(c, LogTypeOutputMessage, combineOutput(reasoning, completion), nil)
}

func DispatchToolResults(c *LlmLogSinkPack, content string) {
	sendChunked(c, LogTypeToolResult, content, nil)
}

func DispatchToolCalls(c *LlmLogSinkPack, toolCallsText string) {
	extra := value.NewMapValue()
	if names, args := parseToolCalls(toolCallsText); names != "" || args != "" {
		if names != "" {
			extra.PutString("function", names)
		}
		if args != "" {
			extra.PutString("arguments", args)
		}
	}
	sendChunked(c, LogTypeTool, toolCallsText, extra)
}

// ── LineLog 빌더 (비공개) ──

func putBaseTags(t *value.MapValue, p *LlmLogSinkPack, logType string) {
	t.PutString("llm_log_type", logType)
	t.PutString("@txid", p.Txid)
	t.PutString("@step_id", p.StepID)
	t.PutString("provider", p.Provider)
}

func buildStepStatusLineLog(s *LlmStepStatus) *logsink.LineLog {
	alog := logsink.NewLineLog()
	alog.Category = Category
	putBaseTags(alog.Tags, &s.LlmLogSinkPack, LogTypeStepStatus)
	alog.Tags.PutString("model", s.Model)
	if s.Stream {
		alog.Tags.PutString("stream", "True")
	} else {
		alog.Tags.PutString("stream", "False")
	}
	alog.Tags.PutString("finish_reason", s.FinishReason)
	alog.Tags.PutString("operation_type", s.OperationType)
	alog.Tags.PutString("features", s.Features)

	alog.Fields.PutLong("index", s.Index)
	if s.Temperature != nil {
		putDouble(alog.Fields,"temperature.n", *s.Temperature)
	}
	appendStepTokenFields(alog.Fields, s)
	if s.ErrorType == "api_error" || s.ErrorType == "program_error" {
		alog.Fields.PutString("error", s.Error)
		alog.Fields.PutString("error_type", s.ErrorType)
	}
	return alog
}

func buildTxStatusLineLog(t *LlmTxStatus) *logsink.LineLog {
	alog := logsink.NewLineLog()
	alog.Category = Category
	alog.Tags.PutString("llm_log_type", LogTypeTxStatus)
	alog.Tags.PutString("@txid", t.Txid)
	alog.Tags.PutString("@first_step_id", t.FirstStepID)
	alog.Tags.PutString("@last_step_id", t.LastStepID)
	if len(t.Providers) > 0 {
		alog.Tags.PutString("provider", joinSorted(t.Providers))
	}
	if len(t.Models) > 0 {
		alog.Tags.PutString("model", joinSorted(t.Models))
	}
	if len(t.OperationTypes) > 0 {
		alog.Tags.PutString("operation_type", joinSorted(t.OperationTypes))
	}

	alog.Fields.PutLong("call_count.n", t.CallCount)
	alog.Fields.PutLong("error_count.n", t.ErrorCount)
	for _, k := range txSummaryTokenFields {
		if v := t.TokenSums[k]; v != 0 {
			alog.Fields.PutLong(k+".n", v)
		}
	}
	if t.Latency != 0 {
		putDouble(alog.Fields,"latency.n", t.Latency)
	}
	if t.Cost != 0 {
		putDouble(alog.Fields,"cost.n", round6(t.Cost))
	}
	if t.InputCost != 0 {
		putDouble(alog.Fields,"input_cost.n", round6(t.InputCost))
	}
	if t.OutputCost != 0 {
		putDouble(alog.Fields,"output_cost.n", round6(t.OutputCost))
	}
	return alog
}

func sendChunked(c *LlmLogSinkPack, logType, content string, extraFields *value.MapValue) {
	chunks := SplitContent(content)
	if len(chunks) == 0 {
		// 빈 content — caller 가 명시 호출 (예: extra 만 있는 케이스)
		sendLineLog(buildMessageLineLog(c, logType, "", extraFields, 0, 1))
		return
	}
	total := len(chunks)
	for idx, chunk := range chunks {
		sendLineLog(buildMessageLineLog(c, logType, chunk, extraFields, idx, total))
	}
}

func buildMessageLineLog(c *LlmLogSinkPack, logType, content string, extraFields *value.MapValue, chunkIdx, chunkTotal int) *logsink.LineLog {
	alog := logsink.NewLineLog()
	alog.Category = Category
	putBaseTags(alog.Tags, c, logType)
	if c.URL != "" {
		alog.Tags.PutString("url", c.URL)
	}
	if c.OperationType != "" {
		alog.Tags.PutString("operation_type", c.OperationType)
	}
	alog.Fields.PutLong("index", c.Index)
	if chunkTotal > 1 {
		alog.Fields.PutLong("chunk_index", int64(chunkIdx))
		alog.Fields.PutLong("chunk_total", int64(chunkTotal))
	}
	if extraFields != nil {
		alog.Fields.PutAll(extraFields)
	}
	alog.Content = content
	return alog
}

func sendLineLog(alog *logsink.LineLog) {
	if alog.Time <= 0 {
		alog.Time = dateutil.SystemNow()
	}
	logsink.Send(alog)
}

// ── helpers ──

func combineOutput(reasoning, completion string) string {
	if reasoning != "" && completion != "" {
		return reasoning + "\n" + completion
	}
	if reasoning != "" {
		return reasoning
	}
	return completion
}

// parseToolCalls — tool_calls_text JSON 에서 function/arguments 추출.
// python whatap/llm/log_sink_packs/llm_tool_calls.py::_parse 동등.
func parseToolCalls(text string) (names, args string) {
	if text == "" {
		return "", ""
	}
	var tcList []map[string]interface{}
	if err := json.Unmarshal([]byte(text), &tcList); err != nil {
		logutil.Println("LLM017", "[LLM] tool_calls parse failed: ", err)
		return "", ""
	}
	var ns, as []string
	for _, tc := range tcList {
		if fnDict, ok := tc["function"].(map[string]interface{}); ok {
			if n, _ := fnDict["name"].(string); n != "" {
				ns = append(ns, n)
			}
			if a, _ := fnDict["arguments"].(string); a != "" {
				as = append(as, a)
			}
		} else if fnStr, ok := tc["function"].(string); ok && fnStr != "" {
			ns = append(ns, fnStr)
			if a, _ := tc["arguments"].(string); a != "" {
				as = append(as, a)
			}
		}
	}
	return strings.Join(ns, ","), strings.Join(as, ",")
}

func joinSorted(set map[string]struct{}) string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ",")
}

func round6(v float64) float64 {
	return math.Round(v*1e6) / 1e6
}

// putDouble — golib v0.0.38 호환 헬퍼 (MapValue.PutDouble 미존재). NewDoubleValue + Put 패턴.
func putDouble(m *value.MapValue, key string, v float64) {
	m.Put(key, value.NewDoubleValue(v))
}

// appendStepTokenFields — python LlmStepStatus._token_fields() 동등.
// 1차에는 set 된 모든 nullable 필드 발행. PROVIDER_TOKEN_FIELDS 분기는 §252 어댑터에서.
func appendStepTokenFields(f *value.MapValue, s *LlmStepStatus) {
	if s.InputTokens != nil {
		f.PutLong("input_tokens.n", *s.InputTokens)
	}
	if s.OutputTokens != nil {
		f.PutLong("output_tokens.n", *s.OutputTokens)
	}
	if s.TotalTokensCount != nil {
		f.PutLong("total_tokens_count.n", *s.TotalTokensCount)
	}
	if s.CachedTokens != nil {
		f.PutLong("cached_tokens.n", *s.CachedTokens)
	}
	if s.ReasoningTokens != nil {
		f.PutLong("reasoning_tokens.n", *s.ReasoningTokens)
	}
	if s.AudioInputTokens != nil {
		f.PutLong("audio_input_tokens.n", *s.AudioInputTokens)
	}
	if s.AudioOutputTokens != nil {
		f.PutLong("audio_output_tokens.n", *s.AudioOutputTokens)
	}
	if s.AcceptedPredictionTokens != nil {
		f.PutLong("accepted_prediction_tokens.n", *s.AcceptedPredictionTokens)
	}
	if s.RejectedPredictionTokens != nil {
		f.PutLong("rejected_prediction_tokens.n", *s.RejectedPredictionTokens)
	}
	if s.EmbeddingCount != nil {
		f.PutLong("embedding_count.n", *s.EmbeddingCount)
	}
	if s.Dimensions != nil {
		f.PutLong("dimensions.n", *s.Dimensions)
	}
	if s.Similarity != nil {
		putDouble(f,"similarity.n", *s.Similarity)
	}
	if s.CacheCreationInputTokens != nil {
		f.PutLong("cache_creation_input_tokens.n", *s.CacheCreationInputTokens)
	}
	if s.CacheReadInputTokens != nil {
		f.PutLong("cache_read_input_tokens.n", *s.CacheReadInputTokens)
	}
	if s.Cost != nil {
		putDouble(f,"cost.n", *s.Cost)
	}
	if s.InputCost != nil {
		putDouble(f,"input_cost.n", *s.InputCost)
	}
	if s.OutputCost != nil {
		putDouble(f,"output_cost.n", *s.OutputCost)
	}
	if s.CachedCost != nil {
		putDouble(f,"cached_cost.n", *s.CachedCost)
	}
	if s.CacheCreationCost != nil {
		putDouble(f,"cache_creation_cost.n", *s.CacheCreationCost)
	}
	if s.Latency != nil {
		putDouble(f,"latency.n", *s.Latency)
	}
	if s.Ttft != nil {
		putDouble(f,"ttft.n", *s.Ttft)
	}
}
