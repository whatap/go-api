//github.com/whatap/go-api/httpc
package httpc

import (
	"github.com/whatap/go-api/common/lang/pack/udp"
	"github.com/whatap/go-api/trace"
)

type HttpcCtx struct {
	ctx  *trace.TraceCtx
	step *udp.UdpTxHttpcPack

	TraceMtraceCallerValue string
	TraceMtracePoidValue   string
	TraceMtraceSpecValue   string
}

func NewHttpcCtx() *HttpcCtx {
	p := new(HttpcCtx)
	return p
}
