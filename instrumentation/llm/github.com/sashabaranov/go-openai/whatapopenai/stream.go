package whatapopenai

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"

	openai "github.com/sashabaranov/go-openai"
	"github.com/whatap/go-api/llm"
)

// CreateChatCompletionStream wraps the streaming chat completion call. The
// returned ChatCompletionStream embeds *openai.ChatCompletionStream and
// overrides Recv / Close to accumulate token / content metadata into the
// step. The step is closed when the inner stream emits io.EOF, an error,
// or the consumer calls Close — whichever happens first.
func (w *Client) CreateChatCompletionStream(
	ctx context.Context,
	req openai.ChatCompletionRequest,
) (*ChatCompletionStream, error) {
	ctx, step := llm.Start(ctx, chatConfig(req))
	fillChatInputs(step, req.Messages)
	if req.Temperature != 0 {
		step.SetTemperature(float64(req.Temperature))
	}
	step.MarkStream()

	inner, err := w.Client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		step.SetError(err, llm.ErrorTypeAPI)
		step.End()
		return nil, err
	}
	return &ChatCompletionStream{
		ChatCompletionStream: inner,
		step:                 step,
	}, nil
}

// ChatCompletionStream wraps *openai.ChatCompletionStream so each Recv
// accumulates content / usage / finish_reason into the WhaTap step. Once
// finalized (io.EOF / error / Close) the step is published.
type ChatCompletionStream struct {
	*openai.ChatCompletionStream
	step *llm.Step

	once             sync.Once
	finalized        bool
	contentBuilder   strings.Builder
	reasonBuilder    strings.Builder
	firstTokenSeen   bool
	lastUsage        *openai.Usage
	lastFinishReason openai.FinishReason
	lastToolCalls    []openai.ToolCall
}

// Recv proxies to the inner stream, accumulating delta content along the way.
// On io.EOF or error the step is finalized exactly once.
func (s *ChatCompletionStream) Recv() (openai.ChatCompletionStreamResponse, error) {
	resp, err := s.ChatCompletionStream.Recv()
	if errors.Is(err, io.EOF) {
		s.finalize()
		return resp, err
	}
	if err != nil {
		s.step.SetError(err, llm.ErrorTypeAPI)
		s.finalize()
		return resp, err
	}

	for _, c := range resp.Choices {
		if !s.firstTokenSeen && (c.Delta.Content != "" || c.Delta.ReasoningContent != "") {
			s.step.RecordFirstToken()
			s.firstTokenSeen = true
		}
		if c.Delta.Content != "" {
			s.contentBuilder.WriteString(c.Delta.Content)
		}
		if c.Delta.ReasoningContent != "" {
			s.reasonBuilder.WriteString(c.Delta.ReasoningContent)
		}
		if c.FinishReason != "" {
			s.lastFinishReason = c.FinishReason
		}
		if len(c.Delta.ToolCalls) > 0 {
			s.lastToolCalls = c.Delta.ToolCalls
		}
	}
	if resp.Usage != nil {
		s.lastUsage = resp.Usage
	}
	return resp, nil
}

// Close finalizes the step (if not already done) and closes the underlying
// stream. Idempotent — extra calls are no-ops on the step side, and the
// underlying Close is also safe to call repeatedly per sashabaranov SDK.
func (s *ChatCompletionStream) Close() {
	s.finalize()
	s.ChatCompletionStream.Close()
}

func (s *ChatCompletionStream) finalize() {
	s.once.Do(func() {
		s.finalized = true
		if s.contentBuilder.Len() > 0 {
			s.step.AddOutputMessage(s.contentBuilder.String())
		}
		if s.reasonBuilder.Len() > 0 {
			s.step.AddReasoning(s.reasonBuilder.String())
		}
		if s.lastUsage != nil {
			s.step.SetTokens(usageToTokens(*s.lastUsage))
		}
		if s.lastFinishReason != "" {
			s.step.SetFinishReason(string(s.lastFinishReason))
		}
		if len(s.lastToolCalls) > 0 {
			if payload := marshalToolCalls(s.lastToolCalls); payload != "" {
				s.step.AddTool(payload, "", "")
			}
		}
		s.step.End()
	})
}
