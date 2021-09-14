package method

import (
	"github.com/whatap/go-api/common/lang/pack/udp"
	"github.com/whatap/go-api/trace"
)

type MethodCtx struct {
	ctx  *trace.TraceCtx
	step *udp.UdpTxMethodPack
}

func NewMethodCtx() *MethodCtx {
	p := new(MethodCtx)
	return p
}
