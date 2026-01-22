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
func Print(a ...any) (n int, err error) {
	appendToLogsink(fmt.Sprint(a...))
	return fmt.Print(a...)
}

// Printf formats according to a format specifier and writes to standard output.
// It returns the number of bytes written and any write error encountered.
// Buffers partial lines until a newline is received.
func Printf(format string, a ...any) (n int, err error) {
	appendToLogsink(fmt.Sprintf(format, a...))
	return fmt.Printf(format, a...)
}

// Println formats using the default formats for its operands and writes to standard output.
// Spaces are always added between operands and a newline is appended.
// It returns the number of bytes written and any write error encountered.
func Println(a ...any) (n int, err error) {
	appendToLogsink(fmt.Sprintln(a...))
	return fmt.Println(a...)
}

// appendToLogsink buffers content and sends complete lines to logsink.
func appendToLogsink(content string) {
	conf := config.GetConfig()

	// 1. 로그 수집 비활성화 시 스킵
	if !conf.LogSinkEnabled || !conf.LogSinkFmtEnabled {
		return
	}

	// 2. LineBuffer.Append()로 버퍼링 (줄바꿈 있으면 flush)
	lines := lineBuffer.Append(content)

	// 3. flush된 라인들 전송
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
