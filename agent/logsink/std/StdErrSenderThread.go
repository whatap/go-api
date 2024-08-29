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

type StdErrSenderThread struct {
	ctx         context.Context
	cancel      context.CancelFunc
	conf        *config.Config
	queue       *queue.RequestQueue
	logWaitTime int
	stderr      *ProxyStream
	origin      *os.File
}

var instanceStdErr *StdErrSenderThread
var lockStdErr sync.Mutex

func GetInstanceStdErr() *StdErrSenderThread {
	lockStdErr.Lock()
	defer lockStdErr.Unlock()
	if instanceStdErr != nil {
		return instanceStdErr
	}
	p := new(StdErrSenderThread)
	p.ctx, p.cancel = context.WithCancel(context.Background())

	p.conf = config.GetConfig()
	p.queue = queue.NewRequestQueue(int(p.conf.LogSinkQueueSize))
	p.origin = os.Stderr
	p.reset(p.conf.LogSinkStdErrEnabled)
	langconf.AddConfObserver("StdErrSenterThread", p)

	instanceStdErr = p

	return instanceStdErr
}

// interface of lang conf observer
func (this *StdErrSenderThread) Run() {
	this.reset(this.conf.LogSinkStdErrEnabled)
	this.queue.SetCapacity(int(this.conf.LogSinkQueueSize))

	if this.conf.Shutdown {
		logutil.Infoln("WALOG001-01", "Shutdown StdErrSenderThread")
		this.reset(false)
	}
}

func (this *StdErrSenderThread) run() {
	// this.reset(this.conf.LogSinkStdErrEnabled)
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
func (this *StdErrSenderThread) Flush() {
	if this.stderr != nil {
		log := this.stderr.LogFlush()
		if log != nil {
			logsink.Send(log)
		}
	}
}

func (this *StdErrSenderThread) reset(enabled bool) {
	if enabled {
		this.logWaitTime = 500
		if this.stderr == nil {
			// new context
			this.ctx, this.cancel = context.WithCancel(context.Background())
			go this.run()

			this.stderr = NewProxyStream(this.conf.LogSinkCategoryStdErr, os.Stderr, this)
			os.Stderr = this.stderr.GetWriter()
		}
		this.stderr.SetEnabled(enabled)
	} else {
		this.logWaitTime = 30000
		if this.stderr != nil {
			os.Stderr = this.origin

			this.stderr.SetEnabled(enabled)
			this.stderr.Shutdown()
		}
		// flush for remained data
		this.Flush()
		this.stderr = nil
		this.cancel()
		this.queue.Clear()
	}
}

func (this *StdErrSenderThread) Add(lineLog *logsink.LineLog) {
	this.queue.Put(lineLog)
}
