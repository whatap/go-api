//github.com/whatap/go-api/trace
package trace

import (
	"sync"

	// "github.com/whatap/golib/io"
	// "github.com/whatap/golib/lang/pack/udp"
	// whatapnet "github.com/whatap/golib/net"
	"github.com/whatap/golib/util/dateutil"

	agenttrace "github.com/whatap/go-api/agent/agent/trace"
)

const (
	UDP_READ_MAX                    = 64 * 1024
	UDP_PACKET_BUFFER               = 64 * 1024
	UDP_PACKET_BUFFER_CHUNKED_LIMIT = 48 * 1024
	UDP_PACKET_CHANNEL_MAX          = 2048
	UDP_PACKET_FLUSH_TIMEOUT        = 10 * 1000

	UDP_PACKET_HEADER_SIZE = 9
	// typ pos 0
	UDP_PACKET_HEADER_TYPE_POS = 0
	// ver pos 1
	UDP_PACKET_HEADER_VER_POS = 1
	// len pos 5
	UDP_PACKET_HEADER_LEN_POS = 5

	UDP_PACKET_SQL_MAX_SIZE = 32768
)

type TraceCtx struct {
	Txid int64
	GID  int64

	Name      string
	StartTime int64

	Ctx *agenttrace.TraceContext

	// Pack
	Host             string
	Uri              string
	Ipaddr           string
	UAgent           string
	Ref              string
	WClientId        string
	HttpMethod       string
	IsStaticContents string
	Status           int32

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
	TraceMtraceMcallee     int64
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
func (this *TraceCtx) GetElapsedTime() int {
	return int(this.StartTime - dateutil.SystemNow())
}

func (this *TraceCtx) Clear() {
	this.Txid = 0
	this.Name = ""
	this.Ctx = nil

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
	this.Status = 0

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
	this.TraceMtraceMcallee = 0
}
