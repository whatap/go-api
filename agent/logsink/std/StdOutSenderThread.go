package std

import (
	"context"
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
	origin      *os.File
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
	p.origin = os.Stdout
	p.reset(p.conf.LogSinkStdOutEnabled)

	langconf.AddConfObserver("StdOutSenterThread", p)
	instanceStdOut = p

	return instanceStdOut
}

// interface of lang conf observer
func (this *StdOutSenderThread) Run() {
	this.reset(this.conf.LogSinkStdOutEnabled)
	this.queue.SetCapacity(int(this.conf.LogSinkQueueSize))

	if this.conf.Shutdown {
		logutil.Infoln("WALOG002-01", "Shutdown StdErrSenderThread")
		this.reset(false)
	}
}

func (this *StdOutSenderThread) run() {
	// this.reset(this.conf.LogSinkStdOutEnabled)
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
			// new context
			this.ctx, this.cancel = context.WithCancel(context.Background())
			go this.run()

			this.stdout = NewProxyStream(this.conf.LogSinkCategoryStdOut, os.Stdout, this)
			os.Stdout = this.stdout.GetWriter()
		}
		this.stdout.SetEnabled(enabled)
	} else {
		this.logWaitTime = 30000
		if this.stdout != nil {
			os.Stdout = this.origin

			this.stdout.SetEnabled(enabled)
			this.stdout.Shutdown()
		}
		// flush for remained data
		this.Flush()
		this.stdout = nil
		this.cancel()
		this.queue.Clear()
	}
}

func (this *StdOutSenderThread) Add(lineLog *logsink.LineLog) {
	this.queue.Put(lineLog)
}
