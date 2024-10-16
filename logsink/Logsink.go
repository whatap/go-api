package logsink

import (
	"io"

	"github.com/whatap/go-api/agent/logsink/std"
	"github.com/whatap/go-api/trace"
)

func HookStdout() {
	if trace.DISABLE() {
		return
	}
	std.GetInstanceStdOut()
}

func GetWriterHookStdout() io.Writer {
	if trace.DISABLE() {
		return nil
	}
	o := std.GetInstanceStdOut()
	return o.GetWriter()
}

func HookStderr() {
	if trace.DISABLE() {
		return
	}
	std.GetInstanceStdErr()
}

func GetWriterHookStderr() io.Writer {
	if trace.DISABLE() {
		return nil
	}
	o := std.GetInstanceStdErr()
	return o.GetWriter()
}
