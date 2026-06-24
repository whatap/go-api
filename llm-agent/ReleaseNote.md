# Go API — LLM Monitoring Release Notes (Separate Track)

> **Separate track**: LLM monitoring has its own release notes and is opt-in (`llm_enabled=true`), but follows the main agent version (lockstep). See [`../ReleaseNote.md`](../ReleaseNote.md) for the main agent release notes.

---

## Go agent — LLM v0.6.0 (MVP)

June 24, 2026

- [New] LLM monitoring (MVP).

    Whatap Go agent now publishes LLM API metrics, message logs, and per-transaction LLM markers when `llm_enabled=true` is set in `whatap.conf`. The runtime captures token counts (input/output/cached/reasoning/...), TTFT/TPOT for streaming responses, finish reason, tool calls, and the full prompt/response content (UTF-8 20KB chunked) — all routed through the existing TCP 6600 channel.

    **Manual API**

    ```go
    import "github.com/whatap/go-api/llm"

    ctx, step := llm.Start(ctx, llm.Config{
        Provider: "openai", Model: "gpt-4o", OperationType: "chat",
    })
    defer step.End()

    step.AddSystemMessage(systemPrompt)
    step.AddInputMessage(userPrompt)
    resp, err := client.CreateChatCompletion(ctx, req)
    if err != nil { step.SetError(err, llm.ErrorTypeAPI); return }
    step.SetTokens(llm.Tokens{Input: 120, Output: 200})
    step.AddOutputMessage(resp.Choices[0].Message.Content)
    ```

    The wrapped HTTP transport (created via `whataphttp.NewLLMRoundTrip`) is the single trace entry point; `llm.Start` only registers a pending state on the context, so there is no duplicate HTTPC step.

    **Provider adapters**

    Two ready-made adapters expose the same metadata extraction as python-apm:

    - `whatapopenai.WrapClient(c *openai.Client)` — sashabaranov/go-openai chat / streaming / completion / embedding
    - `whatapeino.WrapChatModel(m model.ChatModel)` — cloudwego/eino ChatModel interface, including the OpenAI and Claude eino-ext providers
    - `whatapanthropic.WrapAndNewMessage(ctx, client.Messages, params, opts...)` / `WrapAndNewMessageStreaming(...)` — anthropics/anthropic-sdk-go. The helper takes the `MessageService` value directly (the SDK exposes the messages API as a service field on `anthropic.Client`), so the 2-step selector `client.Messages.New(...)` maps to the wrap helper without any client wrapper struct or rebinding. Usage extraction covers all four Anthropic cache-aware token fields (`input_tokens` / `output_tokens` / `cache_creation_input_tokens` / `cache_read_input_tokens`). Streaming accumulates the SDK's six discriminated union events (`message_start` / `content_block_start` / `content_block_delta` / `content_block_stop` / `message_delta` / `message_stop`) so text, tool_use, and thinking deltas are all captured.
    - `whatapopenaigo.WrapAndNewChatCompletion(ctx, client.Chat.Completions, params, opts...)` / `WrapAndNewChatCompletionStreaming(...)` — openai/openai-go (the official OpenAI Go SDK). The helper takes the `ChatCompletionService` value directly. This is a **3-step selector** pattern (`client.Chat.Completions.New(...)`), auto-converted the same way as the Anthropic adapter, one level deeper. Usage extraction covers the OpenAI cache-aware token fields (`prompt_tokens` / `completion_tokens` / `total_tokens` / `prompt_tokens_details.cached_tokens` / `prompt_tokens_details.audio_tokens` / `completion_tokens_details.reasoning_tokens` / `audio_tokens` / `accepted_prediction_tokens` / `rejected_prediction_tokens`). Streaming uses the SDK's `ssestream` package to accumulate `ChatCompletionChunk` deltas; the trailing usage chunk (when `stream_options.include_usage=true`) carries the aggregate `Usage`.

    Streams record TTFT on the first non-empty delta and accumulate token usage / finish reason from the final chunk. Multimodal content (image / audio parts) is recorded as `[IMAGE]` / `[<TYPE>]` placeholders so prompt reconstruction stays identifiable.

    **HTTPC step Driver = "LLM API"**

    LLM calls show up as a dedicated row in the WhaTap UI rather than a generic external HTTP call. The transaction is also tagged with the `is-llm=1` ExtraField for filtering.

---
