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
	"github.com/whatap/go-api/common/util/stringutil"
	"github.com/whatap/go-api/config"
	"github.com/whatap/go-api/trace"
)

const (
	PACKET_DB_MAX_SIZE           = 4 * 1024  // max size of sql
	PACKET_SQL_MAX_SIZE          = 32 * 1024 // max size of sql
	PACKET_HTTPC_MAX_SIZE        = 32 * 1024 // max size of sql
	PACKET_MESSAGE_MAX_SIZE      = 32 * 1024 // max size of message
	PACKET_METHOD_STACK_MAX_SIZE = 32 * 1024 // max size of message

	COMPILE_FILE_MAX_SIZE = 2 * 1024 // max size of filename

	HTTP_HOST_MAX_SIZE   = 2 * 1024 // max size of host
	HTTP_URI_MAX_SIZE    = 2 * 1024 // max size of uri
	HTTP_METHOD_MAX_SIZE = 256      // max size of method
	HTTP_IP_MAX_SIZE     = 256      // max size of ip(request_addr)
	HTTP_UA_MAX_SIZE     = 2 * 1024 // max size of user agent
	HTTP_REF_MAX_SIZE    = 2 * 1024 // max size of referer
	HTTP_USERID_MAX_SIZE = 2 * 1024 // max size of userid

	HTTP_PARAM_MAX_COUNT      = 20
	HTTP_PARAM_KEY_MAX_SIZE   = 255 // = 을 빼고 255 byte
	HTTP_PARAM_VALUE_MAX_SIZE = 256

	HTTP_HEADER_MAX_COUNT      = 20
	HTTP_HEADER_KEY_MAX_SIZE   = 255 // = 을 빼고 255 byte
	HTTP_HEADER_VALUE_MAX_SIZE = 256

	SQL_PARAM_MAX_COUNT      = 20
	SQL_PARAM_VALUE_MAX_SIZE = 256

	STEP_ERROR_MESSAGE_MAX_SIZE = 4 * 1024
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
			p.Url = stringutil.Truncate(url, PACKET_HTTPC_MAX_SIZE)
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
			p.ErrorMessage = stringutil.Truncate(err.Error(), STEP_ERROR_MESSAGE_MAX_SIZE)
			p.ErrorType = stringutil.Truncate(fmt.Sprintf("%d:%s", status, reason), STEP_ERROR_MESSAGE_MAX_SIZE)
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
			p.Url = stringutil.Truncate(url, PACKET_HTTPC_MAX_SIZE)
			udpClient.Send(p)
		}
	}

	return fmt.Errorf("Not found Txid ")
}
