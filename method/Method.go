package method

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/whatap/go-api/common/lang/pack/udp"
	whatapnet "github.com/whatap/go-api/common/net"
	"github.com/whatap/go-api/common/util/dateutil"
	"github.com/whatap/go-api/trace"
)

func Start(ctx context.Context, name string) (*MethodCtx, error) {
	if v := ctx.Value("whatap"); v != nil {
		wCtx := v.(*trace.TraceCtx)
		methodCtx := NewMethodCtx()
		p := udp.NewUdpTxMethodPack()
		p.Txid = wCtx.Txid
		p.Time = dateutil.SystemNow()
		p.Method = name
		methodCtx.ctx = wCtx
		methodCtx.step = p
		return methodCtx, nil
	}

	return nil, fmt.Errorf("Not found Txid ")
}
func End(methodCtx *MethodCtx, err error) error {
	udpClient := whatapnet.GetUdpClient()
	if methodCtx != nil {
		p := methodCtx.step
		p.Elapsed = int32(dateutil.SystemNow() - p.Time)
		// if err != nil {
		// 	p.ErrorMessage = err.Error()
		// 	p.ErrorType = fmt.Sprintf("%d:%s", status, reason)
		// }
		p.Stack = string(debug.Stack())
		udpClient.Send(p)
		return nil
	}

	return fmt.Errorf("HttpcCtx is nil")
}
func Trace(ctx context.Context, name string, elapsed int, err error) error {
	udpClient := whatapnet.GetUdpClient()
	if v := ctx.Value("whatap"); v != nil {
		wCtx := v.(*trace.TraceCtx)
		p := udp.NewUdpTxMethodPack()
		p.Txid = wCtx.Txid
		p.Time = dateutil.SystemNow()
		p.Elapsed = int32(elapsed)
		p.Method = name
		udpClient.Send(p)
	}

	return fmt.Errorf("Not found Txid ")
}
