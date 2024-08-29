package logsink

import (
	"fmt"
	"time"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/util/logutil"

	"github.com/whatap/golib/lang"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/value"
	"github.com/whatap/golib/util/stringutil"
)

var (
	LastLog   int64 = time.Now().UnixMilli()
	LastAlert int64 = time.Now().UnixMilli()
)

func CheckLogContent(lineLog *LineLog, orgContent string) {
	if orgContent == "" {
		return
	}

	conf := config.GetConfig()
	if orgContent != "" {
		sz := len(orgContent)
		if conf.LogSinkLimitContentEnabled && sz > int(conf.LogSinkLimitContentLength) {
			lineLog.Content = string(orgContent[0:int(conf.LogSinkLimitContentLength)])
			lineLog.Content += "...(truncated)"
			lineLog.Truncated = true
			lineLog.OrgContentLength = sz
		} else {
			lineLog.Content = orgContent
		}
	}
}

func Log(lineLog *LineLog) {
	message := CreateLogMessage(lineLog)

	LogStr(message)

	AlertLogTruncated(message)
}

func CreateLogMessage(lineLog *LineLog) string {
	sb := stringutil.NewStringBuffer()
	// message
	sb.AppendLine("LOG_TRUNCATED: too big log")
	// category
	sb.Append("category: ").AppendLine(lineLog.Category)

	// txid
	txidVal := lineLog.Fields.Get("@txid")
	if txidVal != nil {
		// if value.
		sb.Append("txid: ").AppendLine(fmt.Sprintf("%d", lineLog.Fields.GetLong("@txid")))
	}

	// log content
	sb.Append("content: ").Append(lineLog.Content[0:20]).AppendLine("...")

	// original length
	sb.Append("original content length: ").AppendLine(fmt.Sprintf("%d", lineLog.OrgContentLength))

	return sb.ToString()
}

func LogStr(message string) {
	conf := config.GetConfig()
	if conf.DebugLogSinkLimitContentEnabled == false {
		return
	}

	now := time.Now().UnixMilli()
	if now-LastLog < int64(conf.LogSinkLimitContentLogSilentTime) {
		return
	}
	LastLog = now
	logutil.Println(message)
}

func AlertLogTruncated(message string) {
	conf := config.GetConfig()
	if conf.LogsinkLimitContentAlertEnabled == false {
		return
	}

	now := time.Now().UnixMilli()
	if now-LastAlert < int64(conf.LogSinkLimitContentAlertSilentTime) {
		return
	}
	LastAlert = now

	e := pack.NewEventPack()
	e.Level = lang.EVENT_LEVEL_CRITICAL
	e.Title = "LOG_TRUNCATED"
	e.Status = lang.EVENT_STATUS_ON
	e.Message = message

	data.SendEvent(e)
}

func AddTruncatedTag(lineLog *LineLog) {
	lineLog.Tags.Put("@truncated", value.NewBoolValue(true))
}
