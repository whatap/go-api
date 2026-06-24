package whatapanthropic

import (
	"context"
	"strings"
	"sync"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"
	"github.com/whatap/go-api/llm"
)

// WrapAndNewMessageStreaming wraps the streaming /v1/messages call. The
// returned *MessageStream embeds the SDK's *ssestream.Stream and overrides
// Next / Close to accumulate token + content metadata into the step. The
// step is closed when the inner stream finishes, errors, or the consumer
// calls Close — whichever happens first.
//
// Auto-inject rule rewrites `client.Messages.NewStreaming(ctx, params)` to
// `whatapanthropic.WrapAndNewMessageStreaming(ctx, client.Messages, params)`.
func WrapAndNewMessageStreaming(
	ctx context.Context,
	m anthropic.MessageService,
	params anthropic.MessageNewParams,
	opts ...option.RequestOption,
) *MessageStream {
	ctx, step := llm.Start(ctx, chatConfig(params))
	fillChatInputs(step, params)
	if params.Temperature.Valid() {
		step.SetTemperature(params.Temperature.Value)
	}
	step.MarkStream()

	inner := m.NewStreaming(ctx, params, opts...)
	return &MessageStream{Stream: inner, step: step}
}

// MessageStream wraps *ssestream.Stream[anthropic.MessageStreamEventUnion]
// so each Next() accumulates ContentBlockDelta / MessageDelta metadata
// into the WhaTap step. Once finalized (Err / Close / MessageStop) the
// step is published exactly once.
type MessageStream struct {
	*ssestream.Stream[anthropic.MessageStreamEventUnion]
	step *llm.Step

	once           sync.Once
	finalized      bool
	contentBuilder strings.Builder
	reasonBuilder  strings.Builder
	firstTokenSeen bool
	stopReason     string
	usage          anthropic.Usage
	activeTool     *toolUseBlock
	toolBlocks     []toolUseBlock
}

// Next proxies to the inner stream and accumulates the current event into
// the step on the way. Returns the inner Next() result so callers iterate
// `for s.Next() { ... }` exactly as with the upstream stream.
func (s *MessageStream) Next() bool {
	ok := s.Stream.Next()
	if !ok {
		s.finalize()
		return false
	}
	s.absorb(s.Stream.Current())
	return true
}

// Err passes through the inner error after finalizing (so the WhaTap step
// reflects the terminal status).
func (s *MessageStream) Err() error {
	err := s.Stream.Err()
	if err != nil {
		s.step.SetError(err, llm.ErrorTypeAPI)
		s.finalize()
	}
	return err
}

// Close finalizes the step (idempotent) and closes the inner stream.
func (s *MessageStream) Close() error {
	s.finalize()
	return s.Stream.Close()
}

// absorb dispatches each event variant to its accumulator.
// See §253 §2.5.1 (issue 253) for the per-event contract.
func (s *MessageStream) absorb(ev anthropic.MessageStreamEventUnion) {
	switch ev.Type {
	case "message_start":
		m := ev.AsMessageStart()
		// MessageStart 의 Message.Usage 는 보통 InputTokens 만 채워짐.
		// OutputTokens 는 message_delta 에서 갱신.
		s.usage.InputTokens = m.Message.Usage.InputTokens
		s.usage.CacheCreationInputTokens = m.Message.Usage.CacheCreationInputTokens
		s.usage.CacheReadInputTokens = m.Message.Usage.CacheReadInputTokens
	case "content_block_start":
		b := ev.AsContentBlockStart()
		// tool_use 블록 시작 — toolCallBuilder 초기화
		if b.ContentBlock.Type == "tool_use" {
			s.activeTool = &toolUseBlock{
				ID:   b.ContentBlock.ID,
				Name: b.ContentBlock.Name,
			}
		}
	case "content_block_delta":
		d := ev.AsContentBlockDelta()
		// TextDelta — 텍스트 누적 + 첫 토큰 마크
		if td := d.Delta.AsTextDelta(); td.Text != "" {
			if !s.firstTokenSeen {
				s.step.RecordFirstToken()
				s.firstTokenSeen = true
			}
			s.contentBuilder.WriteString(td.Text)
		}
		// InputJSONDelta — tool_use arguments 누적
		if jd := d.Delta.AsInputJSONDelta(); jd.PartialJSON != "" && s.activeTool != nil {
			s.activeTool.Arguments += jd.PartialJSON
		}
		// ThinkingDelta — reasoning 누적
		if thd := d.Delta.AsThinkingDelta(); thd.Thinking != "" {
			s.reasonBuilder.WriteString(thd.Thinking)
		}
	case "content_block_stop":
		// tool_use 블록 finalize → toolBlocks 에 push
		if s.activeTool != nil {
			s.toolBlocks = append(s.toolBlocks, *s.activeTool)
			s.activeTool = nil
		}
	case "message_delta":
		md := ev.AsMessageDelta()
		if r := string(md.Delta.StopReason); r != "" {
			s.stopReason = r
		}
		// message_delta 의 Usage.OutputTokens 가 최종 값
		if md.Usage.OutputTokens != 0 {
			s.usage.OutputTokens = md.Usage.OutputTokens
		}
	case "message_stop":
		// 다음 Next() 가 false 반환 → finalize() 트리거. 별도 처리 없음.
	}
}

// finalize flushes accumulated buffers into the step exactly once.
func (s *MessageStream) finalize() {
	s.once.Do(func() {
		s.finalized = true
		if s.contentBuilder.Len() > 0 {
			s.step.AddOutputMessage(s.contentBuilder.String())
		}
		if s.reasonBuilder.Len() > 0 {
			s.step.AddReasoning(s.reasonBuilder.String())
		}
		if s.stopReason != "" {
			s.step.SetFinishReason(s.stopReason)
		}
		if len(s.toolBlocks) > 0 {
			if payload := marshalToolBlocks(s.toolBlocks); payload != "" {
				s.step.AddTool(payload, "", "")
			}
		}
		s.step.SetTokens(usageToTokens(s.usage))
		s.step.End()
	})
}
