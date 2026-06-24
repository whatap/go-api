# Go API Release Notes

## Go agent v0.6.1

June 24, 2026

- [Change] Version bump to stay in lockstep with go-api-inst v0.6.1 (the auto-instrumentation tool pins `go-api` to its own version). No functional changes to the SDK.

---

## Go agent v0.6.0

June 24, 2026

> **LLM monitoring** is published as a separate release track. See [`llm-agent/ReleaseNote.md`](./llm-agent/ReleaseNote.md) — opt-in via `llm_enabled=true` in `whatap.conf`, shipped on the same TCP 6600 channel. The notes below cover only the LLM changes that surface in the main SDK.

- [Fixed] gRPC server interceptor auto-injection no longer conflicts with an existing interceptor (prevents a runtime panic).

    **Problem**
    - When auto-instrumentation injected the WhaTap interceptor via the `UnaryInterceptor` / `StreamInterceptor` option on `grpc.NewServer`, and the application already set the same option, gRPC panicked at startup ("interceptor was already set and may not be reset"). Surfaced by seaweedfs.

    **Resolution**
    - The grpc instrumentation now uses `ChainUnaryInterceptor` / `ChainStreamInterceptor`, which are additive and coexist with any interceptor the application already configured. This restores the safety guarantee (never break the user's build or runtime). Verified end-to-end on seaweedfs with zero regression on previously passing gRPC apps.

- [Feature] Eino (eino-ext) compose-path auto-extraction via call-site wrapping.

    **Problem**
    - For eino-ext usage, replacing only the constructor left provider/model/token metadata empty (the `WrapChatModel` adapter was never applied on the compose path).

    **Resolution**
    - Compose methods are now wrapped at the call site (`AppendChatModel(WrapToolCallingChatModel(cm))`) and direct calls are transformed (`WrapGenerate` / `WrapStream`). Because the wrapped value keeps its original type, no variable tracking or type change is needed. Verified on a real provider (model `gpt-4o`, tokens and `chat` op captured correctly on both the direct-call and compose paths).

- [New] LLM provider auto-extraction adapters — Eino + sashabaranov/go-openai.

    Two adapters added under `go-api/instrumentation/llm/` so applications using these Go LLM SDKs get a `Driver="LLM API"` external call, token usage, TTFT/TPOT timing, and per-call LogSinkPack rows without writing custom hook code.

    - `whatapeino.WrapChatModel` / `WrapToolCallingChatModel` (cloudwego/eino) — supports Chat / Stream / Completion / Embedding.
    - `whatapopenai.WrapClient` (sashabaranov/go-openai) — supports Chat / Stream / Completion / Embedding. Stream mid-error flush + multimodal placeholder.
    - Verified end-to-end on Argus (real OpenAI / Anthropic calls): `tpot_sum > 0`, `is-llm` marking, `Driver="LLM API"`, single step per call (no duplication).

- [New] LLM provider auto-extraction adapter — Anthropic SDK.

    `whatapanthropic` adapter under `go-api/instrumentation/llm/github.com/anthropics/anthropic-sdk-go/whatapanthropic/`.

    - Covers `Messages.New` + `Messages.NewStreaming` + tool calling.
    - 31 testapps Phase 2 integration regression PASS (2026-05-19). Argus real-traffic verification pending in the user environment.

- [New] LLM provider auto-extraction adapter — openai/openai-go (official SDK).

    `whatapopenaigo` adapter under `go-api/instrumentation/llm/github.com/openai/openai-go/whatapopenaigo/`.

    - Matches the 3-stage selector `client.Chat.Completions.New` produced by the official SDK's chained-method API.
    - Auto-inject rules grew from 103 → 106. Phase 2 PASS (2026-05-19). Argus real-traffic verification pending.

- [Change] LLM instrumentation moved to a nested Go module — `go-api/instrumentation/llm/` (Breaking, opt-in only).

    `anthropic-sdk-go` and `eino-ext/components/model/claude` require Go 1.23, which conflicts with the main `go-api` module's `go 1.18` baseline. To keep the main module on Go 1.18, the LLM adapters were extracted into a nested module that ships under its own `instrumentation/llm/v0.6.0` tag.

    **What this means in practice**

    - Auto-instrumented users (build wrapper) — no action. `whatap-go-inst go build` adds the nested module require automatically when it detects an LLM SDK in `go.mod`.
    - Manual integrators — when you import an LLM adapter package directly, fetch the nested module separately:

      ```bash
      go get github.com/whatap/go-api/instrumentation/llm@v0.6.0
      ```

    - The nested module follows the main module's version on every release (lockstep).

    Verified by full Phase 2 regression (29 testapps, 58 passed / 0 failed / 1 skipped, 5947s; LLM 2 + general 27 with zero regression).

- [Change] `llm.Start` / `llm.Bind` redesigned as a single RoundTrip entry point (Breaking for manual users).

    Previously `llm.Start` internally invoked `httpc.Start`, which caused duplicate spans when the user (or an adapter) also wrapped the HTTP transport. The redesign separates responsibilities cleanly.

    - `llm.Start` no longer calls `httpc.Start`. The URL argument was removed; `Step.End()` now owns the LLM publish path (option C).
    - Idempotency guards added so calling `llm.Start` again on an already-bound context is a no-op.
    - Streaming TTFT / TPOT fallback corrected so `tpot_sum / tpot_count` accumulate even when intermediate chunks lack explicit timing.
    - Adapters (`whatapeino`, `whatapopenai`) had their URL argument removed to match.

    Verified: 16 unit tests + a 10-minute Docker traffic test (3380 calls, fail=0, exactly one step per call, average TPOT 5.88 ms ≈ expected 5.55 ms).

- [Change] `ctx` argument is documented as optional (GID fallback).

    `httpc.Start(ctx, url)`, `sql.StartWithParam(ctx, ...)`, `method.Start(ctx, name)`, `whataphttp.NewRoundTrip(ctx, t)` and similar APIs all run `trace.GetTraceContext(ctx)` internally. When `ctx` does not carry a WhaTap trace context, the call falls back to a goroutine-ID-based lookup. This is not a behavior change — these calls have always tolerated `ctx == nil`. The release simply documents it so users in synchronous single-goroutine code can drop the ctx argument when convenient.

    Note: this guarantee does **not** extend across goroutine boundaries. A new goroutine has a new GID, so if you want a child goroutine to be part of the parent transaction, pass `ctx` explicitly. The fallback only covers the same-goroutine case.

    See `docs/whatap-guide/api-guide.md` (Korean) and `go-api/README.md` (English) for the documented contract.

- [Fixed] Remove double-formatting overhead in whatapfmt hot path.

    **Problem**
    - `whatapfmt.Print/Printf/Println` always called `fmt.Sprint*` before checking `LogSinkEnabled`, causing double formatting + `any` allocations + config load even when `logsink_enabled=false`
    - Under log-heavy workloads (e.g. ntfy), this added measurable overhead even when the runtime switch was off

    **Resolution**
    - Introduced `logsinkActiveConf()` helper that performs the config check **before** any `fmt.Sprint*` call
    - When logsink is inactive, only the original `fmt.Print*` runs; the extra cost reduces to a single config check
    - `appendToLogsink` now accepts the already-loaded config, avoiding redundant `config.GetConfig()` calls
    - Result: the overhead on log-heavy apps (ntfy, loki, msa-app) is improved back to near-baseline

    **Design principle confirmed**
    - "Always inject, toggle at runtime" remains valid, but the runtime branch must happen at the very entry of the hot path — before any formatting or allocation.

- [Fixed] Prevent duplicate transaction creation in StartWithRequest.

    **Problem**
    - When both framework middleware and net/http handler call `Start()`/`StartWithRequest()` in the same goroutine, duplicate transactions are created

    **Resolution**
    - Returns existing active transaction context instead of creating a new one

- [Fixed] Add nil pointer dereference guards in trace functions.

    **Problem**
    - `End()`, `SetHeader()`, `SetParameter()`, `Error()` etc. panic when agent is not initialized

    **Resolution**
    - Added nil guards for safe handling in uninitialized state

- [Fixed] Fix import alias collisions with user packages.

    **Problem**
    - Instrumented code using `trace`, `sql`, `logsink` as import names conflicts with user packages (e.g., Jaeger's `trace` package)

    **Resolution**
    - Changed import aliases: `trace` → `whataptrace`, `sql` → `whatapdb`, `logsink` → `whataplogsink`

- [Feature] Add Kafka Producer external call tracking for sarama.

    Kafka Producer messages are now recorded as httpc (external call) in addition to the existing trace step.

    ```
    URL format: kafka://<broker>/<topic>
    ```

    Enables Kafka calls to appear in WhaTap external call statistics dashboard.

- [Change] Build mode is now fast (toolexec) only — the legacy wrap path was removed.

    `whatap-go-inst go build` uses fast (toolexec-based) instrumentation. This is the **only** build mode in v0.6.0; the former wrap path and its `--wrap` flag were removed (see the Breaking section). Fast mode keeps build overhead low.

    ```bash
    whatap-go-inst go build ./...
    ```

- [Feature] Add vendor project build support.

    Projects using `vendor/` directory are now automatically detected and supported in both wrap and fast build modes.

- [Feature] Add external module instrumentation in fast mode.

    Fast mode now supports `--external-module` flag for instrumenting modules in GOMODCACHE.

    ```bash
    whatap-go-inst --external-module github.com/company/internal-lib go build ./...
    ```

- [Fixed] Fix dependency resolution using `go mod edit` instead of `go get`.

    **Problem**
    - `go get github.com/whatap/go-api` upgrades transitive dependencies to incompatible versions, causing build failures (e.g., apache-answer requiring Go 1.25)

    **Resolution**
    - Changed to `go mod edit -require` which only adds the require line without touching transitive dependencies

---

## Go agent v0.5.4

February 25, 2026

- [Feature] Add `whataphttp.WrapHandler()` function.

    Wraps `http.Handler` interface to automatically trace HTTP transactions.

    ```go
    // Use with http.Handle()
    http.Handle("/api", whataphttp.WrapHandler(&MyHandler{}))

    // Use in http.Server{Handler: ...} struct literal
    s := &http.Server{Handler: whataphttp.WrapHandler(mux)}
    ```

    In addition to existing `whataphttp.Func()` (for HandleFunc), now supports `http.Handler` interface-based handlers.

- [Feature] Add `whatapfasthttp.WrapHandler()` function.

    Wraps FastHTTP handler function to automatically trace HTTP transactions.

    ```go
    // Use in fasthttp.Server{Handler: ...}
    s := &fasthttp.Server{Handler: whatapfasthttp.WrapHandler(myHandler)}
    ```

- [Feature] Add `whataplogrus.WrapLogger()` function.

    Registers WhaTap Hook on custom logger instances created with `logrus.New()`.

    ```go
    // Instrument custom logger instance
    logger := whataplogrus.WrapLogger(logrus.New())
    ```

    In addition to existing blank import (global logger only), now supports individual logger instances.

- [Feature] Add 7 framework Wrap functions.

    Wraps framework instances in various code patterns such as struct field initialization.

    **Added functions**
    - `whatapgin.WrapEngine(*gin.Engine)`
    - `whatapecho.WrapEcho(*echo.Echo)` (v3 and v4 respectively)
    - `whatapfiber.WrapApp(*fiber.App)`
    - `whatapchi.WrapRouter(chi.Router)`
    - `whatapmux.WrapRouter(*mux.Router)`
    - `whatapsarama.WrapConsumer(sarama.Consumer)`

    ```go
    // Use in struct field initialization
    svc := &Service{
        App: whatapfiber.WrapApp(fiber.New()),
    }
    ```

- [Fixed] Improve large SQL normalization performance.

    **Problem**
    - O(n²) performance issue when normalizing SQL queries larger than 1MB, blocking the main goroutine

    **Resolution**
    - Changed `param.String()` to `param.Len()` for O(n) linear processing
    - Added defensive guard to skip normalization for SQL exceeding 32KB

- [Fixed] Fix whatapmux MiddlewareFunc type compatibility.

    **Problem**
    - Build error due to missing `mux.MiddlewareFunc` type in gorilla/mux forks

    **Resolution**
    - Changed return type to `func(http.Handler) http.Handler` for fork compatibility

- [Fixed] Fix logsink GetTraceLogWriter nil return issue.

    **Problem**
    - `log.New(logsink.GetTraceLogWriter(w), prefix, flag)` returns nil when agent is not initialized

    **Resolution**
    - Returns original writer as-is when nil for safe handling

- [Change] Change logrus instrumentation pattern to Hook-based approach.

    **Before**: `logrus.SetOutput(logsink.GetTraceLogWriter(os.Stderr))`
    **After**: `import _ "whataplogrus"` (blank import → Hook auto-registered in init())

    **Benefits**
    - WhaTap configuration persists even when the app calls `logrus.SetOutput()`
    - No conflicts with output settings like `logrus.SetFormatter()`
    - Injection possible even without logrus import in main.go (init file auto-generated)

- [Change] Upgrade relay agent (whatap-agent) Go build version to 1.24.13 for security vulnerability fixes.

    Removed OTLP-related dependencies (golang.org/x/net, grpc, protobuf) and upgraded Go build version to 1.24.13 to resolve security vulnerabilities.

    - Resolved 80 package vulnerabilities (OTLP dependency removal)
    - Resolved 28 stdlib vulnerabilities (Go 1.24.13 upgrade)

---

## Go agent v0.4.6

November 21, 2025

- [Feature] Add Log Stdout output feature.

    Added ability to output logs to both file and stdout simultaneously.

    **Configuration**

    ```ini
    # Output logs to stdout as well (default: false)
    log_stdout_enabled=true
    ```

    **Behavior**
    - Configurable via environment variable or configuration file
    - Supports stdout log output even when file creation fails
    - Uses `io.MultiWriter` for simultaneous file and stdout output

- [Feature] Add Debug log functions.

    Added package-level Debug log functions.

    **Added functions**
    - `logutil.Debug()`
    - `logutil.Debugln()`
    - `logutil.Debugf()`

- [Feature] Add {pid} support for OID pattern.

    Use `{pid}` pattern in `app_name` configuration to include process ID in oname.

    **Configuration example**

    ```ini
    app_name=myapp-{pid}
    ```

    **Result example**
    ```
    oname: myapp-12345
    ```

- [Fixed] Fix OID pattern env. prefix handling bug.

    **Problem**
    - `strings.HasPrefix("env.", key)` used with incorrect argument order, preventing environment variable pattern from working

    **Resolution**
    - Fixed to `strings.HasPrefix(key, "env.")` for correct behavior

- [Feature] Add HTTP Status statistics feature.

    Collects HTTP status code transaction statistics as 5-second and 5-minute metrics.

    **5-second statistics (MeterStatus)**
    - count/time aggregation per HTTP status code
    - Bucket aggregation by 200/400/500 status ranges
    - Detailed aggregation per individual status code

    **5-minute statistics (StatTranxStatus)**
    - count/time/error aggregation by URL + Status combination key
    - Sent to server every 5 minutes

    **Configuration**

    ```ini
    # Enable 5-second statistics (default: true)
    tx_status_meter_enabled=true

    # Enable 5-minute statistics (default: true)
    stat_txstatus_enabled=true

    # Maximum entries for 5-minute statistics (default: 10000)
    stat_txstatus_max_count=10000
    ```

- [Change] Change to send as LogSinkPack when packet size is exceeded.

    **Changes**
    - When packet size exceeds `NetSendMaxBytes`, uses LogSinkPack instead of EventPack to send under `#WhatapSys` category.
    - `agent/net/Sender.go`: Removed duplicate EventPack + LogSinkPack sending, sends only LogSinkPack
    - `agent/net/TcpSession.go`: Changed EventPack to LogSinkPack

    **Effect**
    - Handles packet size exceeded events in the same way as apm-go-agent.
    - Eliminates duplicate event sending, reducing unnecessary network traffic.

---

## Go agent v0.4.5 and earlier

- See previous release notes
