//github.com/whatap/go-api/trace
package trace

import (
	"sync"

	"github.com/whatap/go-api/common/util/dateutil"
)

type TraceCtx struct {
	Txid      int64
	Name      string
	StartTime int64

	// Pack
	Host             string
	Uri              string
	Ipaddr           string
	UAgent           string
	Ref              string
	WClientId        string
	HttpMethod       string
	IsStaticContents string

	MTid        int64
	MDepth      int32
	MCallerTxid int64

	MCallee     int64
	MCallerSpec string
	MCallerUrl  string

	MCallerPoidKey string

	TraceMtraceCallerValue string
	TraceMtracePoidValue   string
	TraceMtraceSpecValue   string

	ActiveSQL    bool
	ActiveHTTPC  bool
	ActiveDBC    bool
	ActiveSocket bool
}

func (this *TraceCtx) GetElapsedTime() int {
	return int(this.StartTime - dateutil.SystemNow())
}

var ctxPool = sync.Pool{
	New: func() interface{} {
		return NewTraceCtx()
	},
}

func NewTraceCtx() *TraceCtx {
	p := new(TraceCtx)
	return p
}
func PoolTraceContext() *TraceCtx {
	p := ctxPool.Get().(*TraceCtx)
	return p
}

func CloseTraceContext(ctx *TraceCtx) {
	if ctx != nil {
		ctx.Clear()
		ctxPool.Put(ctx)
	}
}
func (this *TraceCtx) Clear() {
	this.Txid = 0
	this.Name = ""
	this.StartTime = 0

	// Pack
	this.Host = ""
	this.Uri = ""
	this.Ipaddr = ""
	this.UAgent = ""
	this.Ref = ""
	this.WClientId = ""
	this.HttpMethod = ""
	this.IsStaticContents = ""

	this.MTid = 0
	this.MDepth = 0
	this.MCallerTxid = 0

	this.MCallee = 0
	this.MCallerSpec = ""
	this.MCallerUrl = ""

	this.MCallerPoidKey = ""

	this.TraceMtraceCallerValue = ""
	this.TraceMtracePoidValue = ""
	this.TraceMtraceSpecValue = ""

	this.ActiveSQL = false
	this.ActiveHTTPC = false
	this.ActiveDBC = false
	this.ActiveSocket = false
}
