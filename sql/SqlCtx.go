//github.com/whatap/go-api/sql
package sql

import (
	"sync"

	"github.com/whatap/go-api/trace"
	"github.com/whatap/golib/lang/step"
)

var ctxPool = sync.Pool{
	New: func() interface{} {
		return NewSqlCtx()
	},
}

type SqlCtx struct {
	ctx  *trace.TraceCtx
	step step.Step

	Txid        int64
	ServiceName string
	StartTime   int64
	Dbc         string
	Sql         string
	Param       string
	Type        uint8
	Cpu         int64
	Mem         int64
}

func NewSqlCtx() *SqlCtx {
	p := new(SqlCtx)
	return p
}

func PoolSqlContext() *SqlCtx {
	p := ctxPool.Get().(*SqlCtx)
	return p
}

func CloseSqlContext(ctx *SqlCtx) {
	if ctx != nil {
		ctx.Clear()
		ctxPool.Put(ctx)
	}
}

func (this *SqlCtx) Clear() {
	this.ctx = nil
	this.step = nil

	this.Txid = 0
	this.ServiceName = ""
	this.StartTime = 0
	this.Dbc = ""
	this.Sql = ""
	this.Param = ""
	this.Type = 0
	this.Cpu = 0
	this.Mem = 0
}
