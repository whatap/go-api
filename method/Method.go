package method

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/whatap/go-api/common/lang/pack/udp"
	whatapnet "github.com/whatap/go-api/common/net"
	"github.com/whatap/go-api/common/util/dateutil"
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

func Start(ctx context.Context, name string) (*MethodCtx, error) {
	conf := config.GetConfig()
	if !conf.ProfileMethodEnabled {
		return NewMethodCtx(), nil
	}
	if _, traceCtx := trace.GetTraceContext(ctx); traceCtx != nil {
		methodCtx := NewMethodCtx()
		methodCtx.ctx = traceCtx
		if pack := udp.CreatePack(udp.TX_METHOD, udp.UDP_PACK_VERSION); pack != nil {
			p := pack.(*udp.UdpTxMethodPack)
			p.Txid = traceCtx.Txid
			p.Time = dateutil.SystemNow()
			p.Method = stringutil.Truncate(name, HTTP_URI_MAX_SIZE)
			methodCtx.step = p
		}
		return methodCtx, nil
	}

	return nil, fmt.Errorf("Not found Txid ")
}
func End(methodCtx *MethodCtx, err error) error {
	conf := config.GetConfig()
	if !conf.ProfileMethodEnabled {
		return nil
	}
	udpClient := whatapnet.GetUdpClient()
	if methodCtx != nil && methodCtx.step != nil {
		p := methodCtx.step
		p.Elapsed = int32(dateutil.SystemNow() - p.Time)
		// if err != nil {
		// 	p.ErrorMessage = err.Error()
		// 	p.ErrorType = fmt.Sprintf("%d:%s", status, reason)
		// }
		if conf.ProfileMethodStackEnabled {
			p.Stack = stringutil.Truncate(string(debug.Stack()), PACKET_METHOD_STACK_MAX_SIZE)
		}
		udpClient.Send(p)
		return nil
	}

	return fmt.Errorf("HttpcCtx is nil")
}
func Trace(ctx context.Context, name string, elapsed int, err error) error {
	conf := config.GetConfig()
	if !conf.ProfileMethodEnabled {
		return nil
	}
	udpClient := whatapnet.GetUdpClient()
	if _, traceCtx := trace.GetTraceContext(ctx); traceCtx != nil {
		if pack := udp.CreatePack(udp.TX_METHOD, udp.UDP_PACK_VERSION); pack != nil {
			p := pack.(*udp.UdpTxMethodPack)
			p.Txid = traceCtx.Txid
			p.Time = dateutil.SystemNow()
			p.Elapsed = int32(elapsed)
			p.Method = stringutil.Truncate(name, HTTP_URI_MAX_SIZE)
			if conf.ProfileMethodStackEnabled {
				p.Stack = stringutil.Truncate(string(debug.Stack()), PACKET_METHOD_STACK_MAX_SIZE)
			}
			udpClient.Send(p)
		}
	}

	return fmt.Errorf("Not found Txid ")
}
