# Go API Release Notes

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
