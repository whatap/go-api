package method

import (
	"github.com/whatap/go-api/trace"
	"github.com/whatap/golib/lang/pack/udp"
)

type MethodCtx struct {
	ctx  *trace.TraceCtx
	step *udp.UdpTxMethodPack
}

func NewMethodCtx() *MethodCtx {
	p := new(MethodCtx)
	return p
}
