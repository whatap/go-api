//github.com/whatap/go-api/httpc
package httpc

import (
	"context"
	"fmt"
	"net/http"

	"github.com/whatap/go-api/common/lang/pack/udp"
	whatapnet "github.com/whatap/go-api/common/net"
	"github.com/whatap/go-api/common/util/dateutil"

	// "github.com/whatap/go-api/common/util/urlutil"
	"github.com/whatap/go-api/config"
	"github.com/whatap/go-api/trace"
)

func MTrace(httpcCtx *HttpcCtx) http.Header {
	rt := make(http.Header)
	conf := config.GetConfig()
	rt.Set(conf.TraceMtraceCallerKey, httpcCtx.TraceMtraceCallerValue)
	rt.Set(conf.TraceMtracePoidKey, httpcCtx.TraceMtracePoidValue)
	rt.Set(conf.TraceMtraceSpecKey1, httpcCtx.TraceMtraceSpecValue)

	return rt
}
func Start(ctx context.Context, url string) (*HttpcCtx, error) {
	if v := ctx.Value("whatap"); v != nil {
		wCtx := v.(*trace.TraceCtx)
		httpcCtx := NewHttpcCtx()
		// multi trace info
		httpcCtx.TraceMtraceCallerValue = wCtx.TraceMtraceCallerValue
		httpcCtx.TraceMtracePoidValue = wCtx.TraceMtracePoidValue
		httpcCtx.TraceMtraceSpecValue = wCtx.TraceMtraceSpecValue

		p := udp.NewUdpTxHttpcPack()
		p.Txid = wCtx.Txid
		p.Time = dateutil.SystemNow()
		p.Url = url
		httpcCtx.ctx = wCtx
		httpcCtx.step = p
		return httpcCtx, nil
	}

	return nil, fmt.Errorf("Not found Txid ")
}
func End(httpcCtx *HttpcCtx, status int, reason string, err error) error {
	udpClient := whatapnet.GetUdpClient()
	if httpcCtx != nil {
		p := httpcCtx.step
		p.Elapsed = int32(dateutil.SystemNow() - p.Time)
		if err != nil {
			p.ErrorMessage = err.Error()
			p.ErrorType = fmt.Sprintf("%d:%s", status, reason)
		}
		udpClient.Send(p)
		return nil
	}

	return fmt.Errorf("HttpcCtx is nil")
}
func Trace(ctx context.Context, host string, port int, url string, elapsed int, status int, reason string, err error) error {
	udpClient := whatapnet.GetUdpClient()
	if v := ctx.Value("whatap"); v != nil {
		wCtx := v.(*trace.TraceCtx)
		p := udp.NewUdpTxHttpcPack()
		p.Txid = wCtx.Txid
		p.Time = dateutil.SystemNow()
		p.Elapsed = int32(elapsed)
		p.Url = url
		udpClient.Send(p)
	}

	return fmt.Errorf("Not found Txid ")
}
