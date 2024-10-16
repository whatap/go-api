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

type StdErrSenderThread struct {
	ctx         context.Context
	cancel      context.CancelFunc
	conf        *config.Config
	queue       *queue.RequestQueue
	logWaitTime int
	stderr      *ProxyStream
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
	langconf.AddConfObserver("StdErrSenterThread", p)
	p.reset(p.conf.LogSinkStdErrEnabled)
	go p.run()

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
		if this.stderr != nil {
			this.stderr.Shutdown()
			this.stderr = nil
			this.queue.Clear()
		}
		this.cancel()
	}
}

func (this *StdErrSenderThread) run() {
	this.reset(this.conf.LogSinkStdErrEnabled)
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
			this.stderr = NewProxyStream(this.conf.LogSinkCategoryStdErr, os.Stderr, this)
			os.Stderr = this.stderr.GetWriter()
		}
		this.stderr.SetEnabled(enabled)
	} else {
		this.logWaitTime = 30000
		if this.stderr != nil {
			this.stderr.SetEnabled(enabled)
		}
		// flush for remained data
		this.Flush()
	}
}

func (this *StdErrSenderThread) Add(lineLog *logsink.LineLog) {
	this.queue.Put(lineLog)
}

func (this *StdErrSenderThread) GetWriter() io.Writer {
	if this.stderr != nil {
		return this.stderr.GetWriter()
	}
	return nil
}
