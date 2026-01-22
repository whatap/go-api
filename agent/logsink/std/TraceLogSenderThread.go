package std

import (
	"context"
	"sync"

	"github.com/whatap/go-api/agent/agent/config"
	langconf "github.com/whatap/go-api/agent/lang/conf"
	"github.com/whatap/go-api/agent/logsink"
	"github.com/whatap/go-api/agent/util/logutil"

	"github.com/whatap/golib/util/queue"
)

type TraceLogSenderThread struct {
	ctx         context.Context
	cancel      context.CancelFunc
	conf        *config.Config
	queue       *queue.RequestQueue
	logWaitTime int
}

var instanceTraceLog *TraceLogSenderThread
var lockTraceLog sync.Mutex

func GetTraceLogSenderInstance() *TraceLogSenderThread {
	lockTraceLog.Lock()
	defer lockTraceLog.Unlock()
	if instanceTraceLog != nil {
		return instanceTraceLog
	}
	p := new(TraceLogSenderThread)
	p.ctx, p.cancel = context.WithCancel(context.Background())

	p.conf = config.GetConfig()
	p.queue = queue.NewRequestQueue(int(p.conf.LogSinkQueueSize))

	langconf.AddConfObserver("TraceLogSenderThread", p)
	p.reset(p.conf.LogSinkEnabled)

	go p.run()

	instanceTraceLog = p

	return instanceTraceLog
}

// interface of lang conf observer
func (this *TraceLogSenderThread) Run() {
	this.reset(this.conf.LogSinkEnabled)
	this.queue.SetCapacity(int(this.conf.LogSinkQueueSize))

	if this.conf.Shutdown {
		logutil.Infoln("WALOG003-01", "Shutdown TraceLogSenderThread")
		this.reset(false)
		this.queue.Clear()
		this.cancel()
	}
}

func (this *TraceLogSenderThread) run() {
	for {
		select {
		case <-this.ctx.Done():
			return
		default:
			tmp := this.queue.GetTimeout(this.logWaitTime)
			if tmp != nil {
				if lineLog, ok := tmp.(*logsink.LineLog); ok {
					logsink.Send(lineLog)
				}
			}
		}
	}
}

func (this *TraceLogSenderThread) reset(enabled bool) {
	if enabled {
		this.logWaitTime = 500
	} else {
		this.logWaitTime = 30000
	}
}

// ISdSender interface implementation
func (this *TraceLogSenderThread) Add(lineLog *logsink.LineLog) {
	this.queue.Put(lineLog)
}
