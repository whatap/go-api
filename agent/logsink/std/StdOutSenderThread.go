package std

import (
	"context"
	"io"
	"os"
	"sync"

	"github.com/whatap/go-api/agent/agent/config"
	langconf "github.com/whatap/go-api/agent/lang/conf"
	"github.com/whatap/go-api/agent/logsink"
	"github.com/whatap/go-api/agent/util/logutil"

	"github.com/whatap/golib/util/queue"
)

type StdOutSenderThread struct {
	ctx         context.Context
	cancel      context.CancelFunc
	conf        *config.Config
	queue       *queue.RequestQueue
	logWaitTime int
	stdout      *ProxyStream
}

var instanceStdOut *StdOutSenderThread
var lockStdOut sync.Mutex

func GetInstanceStdOut() *StdOutSenderThread {
	lockStdOut.Lock()
	defer lockStdOut.Unlock()
	if instanceStdOut != nil {
		return instanceStdOut
	}
	p := new(StdOutSenderThread)
	p.ctx, p.cancel = context.WithCancel(context.Background())

	p.conf = config.GetConfig()
	p.queue = queue.NewRequestQueue(int(p.conf.LogSinkQueueSize))

	langconf.AddConfObserver("StdOutSenterThread", p)
	p.reset(p.conf.LogSinkStdOutEnabled)

	go p.run()

	instanceStdOut = p

	return instanceStdOut
}

// interface of lang conf observer
func (this *StdOutSenderThread) Run() {
	this.reset(this.conf.LogSinkStdOutEnabled)
	this.queue.SetCapacity(int(this.conf.LogSinkQueueSize))

	if this.conf.Shutdown {
		logutil.Infoln("WALOG002-01", "Shutdown StdOutSenderThread")
		this.reset(false)
		if this.stdout != nil {
			this.stdout.Shutdown()
			this.stdout = nil
			this.queue.Clear()
		}
		this.cancel()
	}
}

func (this *StdOutSenderThread) run() {
	this.reset(this.conf.LogSinkStdOutEnabled)
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

func (this *StdOutSenderThread) Flush() {
	if this.stdout != nil {
		log := this.stdout.LogFlush()
		if log != nil {
			logsink.Send(log)
		}
	}
}

func (this *StdOutSenderThread) reset(enabled bool) {
	if enabled {
		this.logWaitTime = 500
		if this.stdout == nil {
			this.stdout = NewProxyStream(this.conf.LogSinkCategoryStdOut, os.Stdout, this)
			os.Stdout = this.stdout.GetWriter()
		}
		this.stdout.SetEnabled(enabled)
	} else {
		this.logWaitTime = 30000
		if this.stdout != nil {
			this.stdout.SetEnabled(enabled)
		}
		// flush for remained data
		this.Flush()
	}
}

func (this *StdOutSenderThread) Add(lineLog *logsink.LineLog) {
	this.queue.Put(lineLog)
}

func (this *StdOutSenderThread) GetWriter() io.Writer {
	if this.stdout != nil {
		return this.stdout.GetWriter()
	}
	return nil
}
