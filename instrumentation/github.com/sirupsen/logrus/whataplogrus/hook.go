// Package whataplogrus provides logrus integration for WhaTap APM.
//
// This package uses logrus Hook to capture logs and send them to WhaTap logsink.
// Import this package with blank identifier to enable automatic hook registration:
//
//	import _ "github.com/whatap/go-api/instrumentation/github.com/sirupsen/logrus/whataplogrus"
//
// The hook is registered automatically via init() and uses sync.Once to ensure
// it's only registered once even if imported from multiple files.
package whataplogrus

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/whatap/go-api/agent/agent/config"
	agentlogsink "github.com/whatap/go-api/agent/logsink"
	"github.com/whatap/go-api/agent/logsink/std"
	"github.com/whatap/go-api/trace"
	"github.com/whatap/go-api/trace/gid"
)

var hookOnce sync.Once

func init() {
	hookOnce.Do(func() {
		// Always register the hook - check DISABLE in Fire() instead
		// because at init() time, trace.Init() hasn't been called yet
		logrus.AddHook(&WhatapHook{
			sender: std.GetTraceLogSenderInstance(),
		})
	})
}

// WhatapHook is a logrus hook that sends log entries to WhaTap logsink.
type WhatapHook struct {
	sender std.ISdSender
}

// Levels returns all log levels to capture all logs.
func (h *WhatapHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire is called when a log entry is made.
func (h *WhatapHook) Fire(entry *logrus.Entry) error {
	// Check if WhaTap agent is disabled
	if trace.DISABLE() {
		return nil
	}

	// Get config dynamically (not at init time, because trace.Init() may not have been called yet)
	conf := config.GetConfig()

	// Check if logsink is enabled
	if conf == nil || !conf.LogSinkEnabled {
		return nil
	}

	// Get goroutine ID and transaction context
	var goroutineID int64
	var ctx *trace.TraceCtx
	if conf.LogSinkTraceEnabled {
		goroutineID = gid.GetGID()
		ctx = trace.GetGIDTraceCtx(goroutineID)
	}

	// Create log entry
	lineLog := agentlogsink.NewLineLog()
	lineLog.Category = "AppLog"

	// Format log message
	message := entry.Message
	if len(entry.Data) > 0 {
		message = entry.Message + " " + formatFields(entry.Data)
	}

	// Set level as prefix
	levelPrefix := "[" + entry.Level.String() + "] "
	agentlogsink.CheckLogContent(lineLog, levelPrefix+message)

	// Add transaction context fields
	if ctx != nil {
		if conf.LogSinkTraceTxidEnabled && ctx.Txid != 0 {
			lineLog.Fields.PutLong("@txid", ctx.Txid)
		}
		if conf.LogSinkTraceMtidEnabled && ctx.MTid != 0 {
			lineLog.Fields.PutLong("@mtid", ctx.MTid)
		}
		lineLog.Fields.PutLong("@gid", goroutineID)
	}

	// Add log level
	lineLog.Fields.PutString("@level", entry.Level.String())

	// Send to logsink queue
	if h.sender != nil {
		h.sender.Add(lineLog)
	}

	return nil
}

// WrapLogger adds WhatapHook to a logrus.Logger instance created by logrus.New().
// The init() function only hooks the global StandardLogger.
// Use this function to instrument logrus.New() instances.
//
// Example:
//
//	log := whataplogrus.WrapLogger(logrus.New())
func WrapLogger(l *logrus.Logger) *logrus.Logger {
	if l != nil {
		l.AddHook(&WhatapHook{
			sender: std.GetTraceLogSenderInstance(),
		})
	}
	return l
}

// formatFields formats logrus fields as string
func formatFields(fields logrus.Fields) string {
	if len(fields) == 0 {
		return ""
	}

	result := "{"
	first := true
	for k, v := range fields {
		if !first {
			result += ", "
		}
		result += k + "=" + formatValue(v)
		first = false
	}
	result += "}"
	return result
}

// formatValue formats a single value as string
func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case error:
		return val.Error()
	default:
		return fmt.Sprintf("%v", val)
	}
}
