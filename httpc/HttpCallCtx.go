//github.com/whatap/go-api/httpc
package httpc

import (
	"sync"

	"github.com/whatap/go-api/trace"
	"github.com/whatap/golib/lang/step"
)

var ctxPool = sync.Pool{
	New: func() interface{} {
		return NewHttpcCtx()
	},
}

type HttpcCtx struct {
	ctx  *trace.TraceCtx
	step step.Step

	Txid        int64
	ServiceName string
	StartTime   int64
	Url         string
	Cpu         int64
	Mem         int64

	StepId int64

	TraceMtraceTraceparentValue string
	TraceMtraceCallerValue      string
	TraceMtracePoidValue        string
	TraceMtraceSpecValue        string
	TraceMtraceMcallee          int64
}

func NewHttpcCtx() *HttpcCtx {
	p := new(HttpcCtx)
	return p
}

func PoolHttpcContext() *HttpcCtx {
	p := ctxPool.Get().(*HttpcCtx)
	return p
}

func CloseHttpcContext(ctx *HttpcCtx) {
	if ctx != nil {
		ctx.Clear()
		ctxPool.Put(ctx)
	}
}

func (this *HttpcCtx) Clear() {
	this.ctx = nil
	this.step = nil

	this.Txid = 0
	this.ServiceName = ""
	this.StartTime = 0
	this.Url = ""
	this.Cpu = 0
	this.Mem = 0

	this.StepId = 0

	this.TraceMtraceTraceparentValue = ""
	this.TraceMtraceCallerValue = ""
	this.TraceMtracePoidValue = ""
	this.TraceMtraceSpecValue = ""
	this.TraceMtraceMcallee = 0
}
