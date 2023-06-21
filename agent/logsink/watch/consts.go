package watch

import "regexp"

const (
	SEND_THRESHOLD   = 0
	INDENTATION_CHAR = byte('\t')
	NEWLINE          = "\n"
)

var (
	LogSendThreshold  int32 = 500
	TxIdTag                 = "@txid"
	AppLogCategory          = "AppLog"
	AppLogPattern, _        = regexp.Compile(`-- (\{.*\}) --`)
	DebugAppLogParser       = false
)
