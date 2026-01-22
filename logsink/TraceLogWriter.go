package logsink

import (
	"io"

	"github.com/whatap/go-api/agent/agent/config"
	agentlogsink "github.com/whatap/go-api/agent/logsink"
	"github.com/whatap/go-api/agent/logsink/std"
	"github.com/whatap/go-api/trace"
	"github.com/whatap/go-api/trace/gid"
)

// TraceLogWriter is an io.Writer that captures goroutine ID and transaction context.
// Use this with log.SetOutput() for transaction-linked log collection.
//
// Uses LineBuffer.Append() to buffer partial lines (without newline) and flush
// when a complete line is received. This matches Java agent behavior.
type TraceLogWriter struct {
	origin     io.Writer
	category   string
	sender     std.ISdSender
	conf       *config.Config
	lineBuffer *agentlogsink.LineBuffer
}

// NewTraceLogWriter creates a new TraceLogWriter.
func NewTraceLogWriter(origin io.Writer, category string) *TraceLogWriter {
	return &TraceLogWriter{
		origin:     origin,
		category:   category,
		sender:     std.GetTraceLogSenderInstance(),
		conf:       config.GetConfig(),
		lineBuffer: agentlogsink.NewLineBuffer(),
	}
}

func (w *TraceLogWriter) Write(data []byte) (int, error) {
	// 1. 원본 출력 먼저
	n, err := w.origin.Write(data)

	// 2. 로그 수집 비활성화 시 리턴
	if !w.conf.LogSinkEnabled {
		return n, err
	}

	// 3. LineBuffer.Append()로 버퍼링 (줄바꿈 있으면 flush)
	lines := w.lineBuffer.Append(string(data))

	// 4. flush된 라인들 전송
	for _, content := range lines {
		if content == "" {
			continue
		}
		w.sendLine(content)
	}

	return n, err
}

// sendLine sends a single line to logsink with transaction context.
func (w *TraceLogWriter) sendLine(content string) {
	// 1. goroutine/트랜잭션 연계 (옵션)
	var goroutineID int64
	var ctx *trace.TraceCtx
	if w.conf.LogSinkTraceEnabled {
		goroutineID = gid.GetGID()
		ctx = trace.GetGIDTraceCtx(goroutineID)
	}

	// 2. 로그 엔트리 생성
	lineLog := agentlogsink.NewLineLog()
	lineLog.Category = w.category
	agentlogsink.CheckLogContent(lineLog, content)

	// 3. 트랜잭션 연계 필드 추가
	if ctx != nil {
		if w.conf.LogSinkTraceTxidEnabled && ctx.Txid != 0 {
			lineLog.Fields.PutLong("@txid", ctx.Txid)
		}
		if w.conf.LogSinkTraceMtidEnabled && ctx.MTid != 0 {
			lineLog.Fields.PutLong("@mtid", ctx.MTid)
		}
		lineLog.Fields.PutLong("@gid", goroutineID)
	}

	// 4. 비동기 큐에 추가
	w.sender.Add(lineLog)
}

// GetTraceLogWriter returns an io.Writer that wraps the given writer and captures
// goroutine ID and transaction context. Use this with log.SetOutput() for
// transaction-linked log collection. Category defaults to "AppLog".
//
// Example:
//
//	log.SetOutput(logsink.GetTraceLogWriter(os.Stderr))
func GetTraceLogWriter(w io.Writer) io.Writer {
	if trace.DISABLE() {
		return nil
	}
	return NewTraceLogWriter(w, "AppLog")
}

// GetTraceLogWriterWithCategory returns an io.Writer with a custom category.
//
// Example:
//
//	log.SetOutput(logsink.GetTraceLogWriterWithCategory(os.Stderr, "MyApp"))
func GetTraceLogWriterWithCategory(w io.Writer, category string) io.Writer {
	if trace.DISABLE() {
		return nil
	}
	return NewTraceLogWriter(w, category)
}
