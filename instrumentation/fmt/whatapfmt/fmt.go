// Package whatapfmt provides fmt-compatible functions that capture goroutine ID
// and transaction context for log collection. Use this package instead of fmt
// for transaction-linked stdout logging.
//
// Uses LineBuffer.Append() to buffer partial lines (without newline) and flush
// when a complete line is received. This matches Java agent behavior.
//
// Example:
//
//	// Before
//	fmt.Println("Hello")
//
//	// After (AST transformed)
//	whatapfmt.Println("Hello")
package whatapfmt

import (
	"fmt"

	"github.com/whatap/go-api/agent/agent/config"
	agentlogsink "github.com/whatap/go-api/agent/logsink"
	"github.com/whatap/go-api/agent/logsink/std"
	"github.com/whatap/go-api/trace"
	"github.com/whatap/go-api/trace/gid"
)

const category = "AppStdOut"

// lineBuffer buffers partial lines until a newline is received.
var lineBuffer = agentlogsink.NewLineBuffer()

// Print formats using the default formats for its operands and writes to standard output.
// It returns the number of bytes written and any write error encountered.
// Buffers partial lines until a newline is received.
//
// §235: early return before fmt.Sprint to skip the extra formatting pass when
// logsink is disabled. Original fmt.Print path is always preserved.
func Print(a ...any) (n int, err error) {
	if conf := logsinkActiveConf(); conf != nil {
		appendToLogsink(conf, fmt.Sprint(a...))
	}
	return fmt.Print(a...)
}

// Printf formats according to a format specifier and writes to standard output.
// It returns the number of bytes written and any write error encountered.
// Buffers partial lines until a newline is received.
func Printf(format string, a ...any) (n int, err error) {
	if conf := logsinkActiveConf(); conf != nil {
		appendToLogsink(conf, fmt.Sprintf(format, a...))
	}
	return fmt.Printf(format, a...)
}

// Println formats using the default formats for its operands and writes to standard output.
// Spaces are always added between operands and a newline is appended.
// It returns the number of bytes written and any write error encountered.
func Println(a ...any) (n int, err error) {
	if conf := logsinkActiveConf(); conf != nil {
		appendToLogsink(conf, fmt.Sprintln(a...))
	}
	return fmt.Println(a...)
}

// logsinkActiveConf returns the config when logsink+fmt capture is active,
// and nil otherwise. Callers must perform the cheap nil check before invoking
// fmt.Sprint*/Sprintf/Sprintln to avoid the double-format cost when logsink
// is disabled (§235).
func logsinkActiveConf() *config.Config {
	conf := config.GetConfig()
	if conf == nil || !conf.LogSinkEnabled || !conf.LogSinkFmtEnabled {
		return nil
	}
	return conf
}

// appendToLogsink buffers content and sends complete lines to logsink.
// Callers must have already verified logsink is active via logsinkActiveConf.
func appendToLogsink(conf *config.Config, content string) {
	// LineBuffer.Append()로 버퍼링 (줄바꿈 있으면 flush)
	lines := lineBuffer.Append(content)

	// flush된 라인들 전송
	for _, line := range lines {
		if line == "" {
			continue
		}
		sendLine(conf, line)
	}
}

// sendLine sends a single line to logsink with transaction context.
func sendLine(conf *config.Config, content string) {
	// 1. goroutine/트랜잭션 연계
	var goroutineID int64
	var ctx *trace.TraceCtx
	if conf.LogSinkTraceEnabled {
		goroutineID = gid.GetGID()
		ctx = trace.GetGIDTraceCtx(goroutineID)
	}

	// 2. LineLog 생성
	lineLog := agentlogsink.NewLineLog()
	lineLog.Category = category
	agentlogsink.CheckLogContent(lineLog, content)

	// 3. 트랜잭션 연계 필드 추가
	if ctx != nil {
		if conf.LogSinkTraceTxidEnabled && ctx.Txid != 0 {
			lineLog.Fields.PutLong("@txid", ctx.Txid)
		}
		if conf.LogSinkTraceMtidEnabled && ctx.MTid != 0 {
			lineLog.Fields.PutLong("@mtid", ctx.MTid)
		}
		lineLog.Fields.PutLong("@gid", goroutineID)
	}

	// 4. 비동기 큐에 추가
	sender := std.GetTraceLogSenderInstance()
	sender.Add(lineLog)
}
