package whatapopenaigo

import (
	"context"
	"strings"
	"sync"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/ssestream"
	"github.com/whatap/go-api/llm"
)

// WrapAndNewChatCompletionStreaming wraps the streaming /v1/chat/completions
// call. The returned *ChatCompletionStream embeds the SDK's
// *ssestream.Stream and overrides Next / Close to accumulate token + content
// metadata into the step. The step is closed when the inner stream finishes,
// errors, or the consumer calls Close — whichever happens first.
//
// Auto-inject rule rewrites `client.Chat.Completions.NewStreaming(ctx, params)` to
// `whatapopenaigo.WrapAndNewChatCompletionStreaming(ctx, client.Chat.Completions, params)`.
func WrapAndNewChatCompletionStreaming(
	ctx context.Context,
	s openai.ChatCompletionService,
	params openai.ChatCompletionNewParams,
	opts ...option.RequestOption,
) *ChatCompletionStream {
	ctx, step := llm.Start(ctx, chatConfig(params))
	fillChatInputs(step, params)
	if params.Temperature.Valid() {
		step.SetTemperature(params.Temperature.Value)
	}
	step.MarkStream()

	inner := s.NewStreaming(ctx, params, opts...)
	return &ChatCompletionStream{Stream: inner, step: step}
}

// ChatCompletionStream wraps *ssestream.Stream[openai.ChatCompletionChunk]
// so each Next() accumulates delta content + the final usage chunk into the
// WhaTap step. Once finalized (Err / Close / inner Next returns false) the
// step is published exactly once.
type ChatCompletionStream struct {
	*ssestream.Stream[openai.ChatCompletionChunk]
	step *llm.Step

	once           sync.Once
	finalized      bool
	contentBuilder strings.Builder
	firstTokenSeen bool
	finishReason   string
	usage          openai.CompletionUsage
	model          string
}

// Next proxies to the inner stream and accumulates the current event into
// the step on the way. Returns the inner Next() result so callers iterate
// `for s.Next() { ... }` exactly as with the upstream stream.
func (s *ChatCompletionStream) Next() bool {
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
func (s *ChatCompletionStream) Err() error {
	err := s.Stream.Err()
	if err != nil {
		s.step.SetError(err, llm.ErrorTypeAPI)
		s.finalize()
	}
	return err
}

// Close finalizes the step (idempotent) and closes the inner stream.
func (s *ChatCompletionStream) Close() error {
	s.finalize()
	return s.Stream.Close()
}

// absorb extracts text delta + finish reason + usage from a chunk.
// The OpenAI streaming format emits incremental deltas in each chunk's
// Choices[0].Delta.Content. The final chunk (sent only when StreamOptions
// .IncludeUsage=true) carries the aggregate Usage.
func (s *ChatCompletionStream) absorb(ev openai.ChatCompletionChunk) {
	if ev.Model != "" {
		s.model = ev.Model
	}
	if len(ev.Choices) > 0 {
		ch := ev.Choices[0]
		if ch.Delta.Content != "" {
			if !s.firstTokenSeen {
				s.step.RecordFirstToken()
				s.firstTokenSeen = true
			}
			s.contentBuilder.WriteString(ch.Delta.Content)
		}
		if ch.FinishReason != "" {
			s.finishReason = ch.FinishReason
		}
	}
	// Usage 는 IncludeUsage=true 일 때 마지막 chunk 에만 포함. 값이 있으면 누적 결과로 덮어씀.
	if ev.Usage.TotalTokens != 0 || ev.Usage.PromptTokens != 0 || ev.Usage.CompletionTokens != 0 {
		s.usage = ev.Usage
	}
}

// finalize flushes accumulated buffers into the step exactly once.
func (s *ChatCompletionStream) finalize() {
	s.once.Do(func() {
		s.finalized = true
		if s.contentBuilder.Len() > 0 {
			s.step.AddOutputMessage(s.contentBuilder.String())
		}
		if s.finishReason != "" {
			s.step.SetFinishReason(s.finishReason)
		}
		s.step.SetTokens(usageToTokens(s.usage))
		s.step.End()
	})
}
