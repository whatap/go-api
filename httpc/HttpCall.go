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
	conf := config.GetConfig()
	if !conf.Enabled {
		return NewHttpcCtx(), nil
	}
	if v := ctx.Value("whatap"); v != nil {
		wCtx := v.(*trace.TraceCtx)
		httpcCtx := NewHttpcCtx()
		httpcCtx.ctx = wCtx

		// multi trace info
		httpcCtx.TraceMtraceCallerValue = wCtx.TraceMtraceCallerValue
		httpcCtx.TraceMtracePoidValue = wCtx.TraceMtracePoidValue
		httpcCtx.TraceMtraceSpecValue = wCtx.TraceMtraceSpecValue

		if pack := udp.CreatePack(udp.TX_HTTPC, udp.UDP_PACK_VERSION); pack != nil {
			p := pack.(*udp.UdpTxHttpcPack)
			p.Txid = wCtx.Txid
			p.Time = dateutil.SystemNow()
			p.Url = url
			httpcCtx.step = p
		}

		return httpcCtx, nil
	}

	return nil, fmt.Errorf("Not found Txid ")
}
func End(httpcCtx *HttpcCtx, status int, reason string, err error) error {
	conf := config.GetConfig()
	if !conf.Enabled {
		return nil
	}
	udpClient := whatapnet.GetUdpClient()
	if httpcCtx != nil && httpcCtx.step != nil {
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
	conf := config.GetConfig()
	if !conf.Enabled {
		return nil
	}
	udpClient := whatapnet.GetUdpClient()
	if v := ctx.Value("whatap"); v != nil {
		wCtx := v.(*trace.TraceCtx)
		if pack := udp.CreatePack(udp.TX_HTTPC, udp.UDP_PACK_VERSION); pack != nil {
			p := pack.(*udp.UdpTxHttpcPack)
			p.Txid = wCtx.Txid
			p.Time = dateutil.SystemNow()
			p.Elapsed = int32(elapsed)
			p.Url = url
			udpClient.Send(p)
		}
	}

	return fmt.Errorf("Not found Txid ")
}
