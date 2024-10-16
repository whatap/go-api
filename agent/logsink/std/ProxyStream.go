package std

import (
	"fmt"
	"io"
	"os"

	// "time"
	"bufio"
	"strings"

	"github.com/whatap/go-api/agent/logsink"
	"github.com/whatap/go-api/agent/util/logutil"
)

type ProxyStream struct {
	enabled       bool
	lineBuffer    *logsink.LineBuffer
	origin        *os.File
	category      string
	sender        ISdSender
	fileReadPipe  *os.File
	fileWritePipe *os.File
}

func NewProxyStream(category string, origin *os.File, sender ISdSender) *ProxyStream {
	p := new(ProxyStream)
	p.lineBuffer = logsink.NewLineBuffer()
	p.category = category
	p.origin = origin
	p.sender = sender
	if r, w, err := os.Pipe(); err == nil {
		p.fileReadPipe = r
		p.fileWritePipe = w
		go p.IOCopy()
	}
	return p
}

func (this *ProxyStream) SetEnabled(b bool) {
	this.enabled = b
}

func (this *ProxyStream) LogFlush() *logsink.LineLog {
	content := this.lineBuffer.Flush()
	if content != "" {
		lineLog := logsink.NewLineLog()
		lineLog.Category = this.category
		logsink.CheckLogContent(lineLog, content)
		return lineLog
	}
	return nil
}

func (this *ProxyStream) AddTxTag(lineLog *logsink.LineLog) {
	if lineLog == nil {
		return
	}

	// java
	// TraceContext ctx = TraceContextManager.getLocalContext();
	// if (ctx != null) {
	// 	if (ConfLogSink.logsink_trace_txid_enabled && ctx.txid != 0) {
	// 		lineLog.fields.put("@txid", ctx.txid);
	// 	}
	// 	if (ConfLogSink.logsink_trace_mtid_enabled && ctx.mtid != 0) {
	// 		lineLog.fields.put("@mtid", ctx.mtid);
	// 	}

	// 	if (ConfLogSink.logsink_trace_enabled) {
	// 		if (ConfLogSink.logsink_trace_login_enabled && ctx.login != null) {
	// 			lineLog.fields.put("@login", ctx.login);
	// 		}
	// 		if (ConfLogSink.logsink_trace_httphost_enabled && ctx.http_host != null) {
	// 			lineLog.tags.put("httphost", ctx.http_host);
	// 		}
	// 	}
	// }
}

func (this *ProxyStream) IOCopy() {
	logutil.Println(fmt.Sprintf("WA%s-001", this.category), this.category, ", Start io.Copy ")
	n, err := io.Copy(this, this.fileReadPipe)
	logutil.Println(fmt.Sprintf("WA%s-002", this.category), this.category, ", Close io.Copy ", n, ", ", err)
}

func (this *ProxyStream) Write(data []byte) (n int, err error) {
	if this.origin != nil {
		fmt.Fprint(this.origin, string(data))
	}
	if this.enabled {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			lines := this.lineBuffer.AppendLine(scanner.Text())
			for _, content := range lines {
				if content != "" && len(content) > 1 {
					lineLog := logsink.NewLineLog()
					lineLog.Category = this.category
					logsink.CheckLogContent(lineLog, content)

					// if lineLog.Content != "" {
					// 	if this.conf.LogSinkTraceEnabled {
					// 		this.AddTxTag(lineLog)
					// 	}
					// }

					this.sender.Add(lineLog)
				}
			}
		}
	}
	return len(data), nil
}

func (this *ProxyStream) GetWriter() *os.File {
	return this.fileWritePipe
}

func (this *ProxyStream) closePipe() {
	if this.fileWritePipe != nil {
		this.fileWritePipe.Close()
	}
	if this.fileReadPipe != nil {
		this.fileReadPipe.Close()
	}
}

func (this *ProxyStream) Shutdown() {
	this.closePipe()
}
