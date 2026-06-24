package whatapeino

import (
	"errors"
	"io"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/whatap/go-api/llm"
)

const streamBufferSize = 16

// wrapStream copies the inner StreamReader through a fresh pipe, accumulating
// each chunk's Content / ReasoningContent / ResponseMeta into the supplied
// step. The step is closed once the inner stream emits io.EOF or an error.
//
// The returned StreamReader behaves identically to the inner one — same
// chunks, same ordering, same final io.EOF — so wrapping is transparent to
// the caller's loop.
func wrapStream(step *llm.Step, inner *schema.StreamReader[*schema.Message]) *schema.StreamReader[*schema.Message] {
	out, sw := schema.Pipe[*schema.Message](streamBufferSize)

	go func() {
		defer step.End()
		defer sw.Close()
		defer inner.Close()

		var (
			firstTokenSeen bool
			contentBuilder strings.Builder
			reasonBuilder  strings.Builder
			lastMeta       *schema.ResponseMeta
			lastToolCalls  []schema.ToolCall
		)

		for {
			chunk, err := inner.Recv()

			if errors.Is(err, io.EOF) {
				flushAccumulators(step, &contentBuilder, &reasonBuilder, lastMeta, lastToolCalls)
				sw.Send(chunk, err)
				return
			}
			if err != nil {
				// 부분 응답 누적분은 발행 (SetError 와 함께 추적 가치 있음).
				flushAccumulators(step, &contentBuilder, &reasonBuilder, lastMeta, lastToolCalls)
				step.SetError(err, llm.ErrorTypeAPI)
				sw.Send(chunk, err)
				return
			}

			if chunk != nil {
				// TTFT — stamp the first chunk that actually carries text.
				// Empty pre-amble chunks (role / metadata only) are skipped.
				if !firstTokenSeen && (chunk.Content != "" || chunk.ReasoningContent != "") {
					step.RecordFirstToken()
					firstTokenSeen = true
				}
				if chunk.Content != "" {
					contentBuilder.WriteString(chunk.Content)
				}
				if chunk.ReasoningContent != "" {
					reasonBuilder.WriteString(chunk.ReasoningContent)
				}
				if chunk.ResponseMeta != nil {
					lastMeta = chunk.ResponseMeta
				}
				if len(chunk.ToolCalls) > 0 {
					lastToolCalls = chunk.ToolCalls
				}
			}

			if closed := sw.Send(chunk, nil); closed {
				// Consumer hung up — finalize what we have and stop.
				flushAccumulators(step, &contentBuilder, &reasonBuilder, lastMeta, lastToolCalls)
				return
			}
		}
	}()

	return out
}

func flushAccumulators(
	step *llm.Step,
	content, reason *strings.Builder,
	meta *schema.ResponseMeta,
	toolCalls []schema.ToolCall,
) {
	if content.Len() > 0 {
		step.AddOutputMessage(content.String())
	}
	if reason.Len() > 0 {
		step.AddReasoning(reason.String())
	}
	if meta != nil {
		if meta.FinishReason != "" {
			step.SetFinishReason(meta.FinishReason)
		}
		if meta.Usage != nil {
			step.SetTokens(usageToTokens(*meta.Usage))
		}
	}
	if len(toolCalls) > 0 {
		if payload := marshalToolCalls(toolCalls); payload != "" {
			step.AddTool(payload, "", "")
		}
	}
}
