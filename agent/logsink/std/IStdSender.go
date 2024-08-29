package std

import (
	"github.com/whatap/go-api/agent/logsink"
)

type ISdSender interface {
	Add(lineLog *logsink.LineLog)
}
