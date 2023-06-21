package method

import (
	"sync"

	"github.com/whatap/go-api/trace"
	"github.com/whatap/golib/lang/step"
)

var ctxPool = sync.Pool{
	New: func() interface{} {
		return NewMethodCtx()
	},
}

type MethodCtx struct {
	ctx  *trace.TraceCtx
	step step.Step

	Txid        int64
	ServiceName string
	StartTime   int64
	Method      string
	Stack       string
	Cpu         int64
	Mem         int64
}

func NewMethodCtx() *MethodCtx {
	p := new(MethodCtx)
	return p
}

func PoolMethodContext() *MethodCtx {
	p := ctxPool.Get().(*MethodCtx)
	return p
}

func CloseMethodContext(ctx *MethodCtx) {
	if ctx != nil {
		ctx.Clear()
		ctxPool.Put(ctx)
	}
}

func (this *MethodCtx) Clear() {
	this.ctx = nil
	this.step = nil

	this.Txid = 0
	this.ServiceName = ""
	this.StartTime = 0
	this.Method = ""
	this.Stack = ""
	this.Cpu = 0
	this.Mem = 0
}
