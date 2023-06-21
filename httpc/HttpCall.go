//github.com/whatap/go-api/httpc
package httpc

import (
	"context"
	"fmt"
	"log"
	"net/http"

	agentconfig "github.com/whatap/go-api/agent/agent/config"
	agenttrace "github.com/whatap/go-api/agent/agent/trace"
	agentapi "github.com/whatap/go-api/agent/agent/trace/api"
	"github.com/whatap/go-api/trace"

	"github.com/whatap/golib/lang/step"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/keygen"
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

func GetMTrace(httpcCtx *HttpcCtx) http.Header {
	rt := make(http.Header)
	conf := agentconfig.GetConfig()
	if conf.MtraceEnabled && httpcCtx.TraceMtraceCallerValue != "" {
		rt.Set(conf.TraceMtraceCallerKey, httpcCtx.TraceMtraceCallerValue)
		rt.Set(conf.TraceMtracePoidKey, httpcCtx.TraceMtracePoidValue)
		rt.Set(conf.TraceMtraceSpecKey1, httpcCtx.TraceMtraceSpecValue)
	}
	// Mcallee
	if conf.MtraceCalleeTxidEnabled {
		httpcCtx.TraceMtraceMcallee = keygen.Next()
		rt.Set(conf.TraceMtraceCalleeKey, fmt.Sprintf("%d", httpcCtx.TraceMtraceMcallee))
	}

	return rt
}

func Start(ctx context.Context, url string) (*HttpcCtx, error) {
	conf := agentconfig.GetConfig()
	if !conf.Enabled {
		return PoolHttpcContext(), nil
	}
	httpcCtx := PoolHttpcContext()

	httpcCtx.StartTime = dateutil.SystemNow()
	httpcCtx.Url = url
	if _, traceCtx := trace.GetTraceContext(ctx); traceCtx != nil {
		httpcCtx.ctx = traceCtx
		httpcCtx.Txid = traceCtx.Txid
		httpcCtx.ServiceName = traceCtx.Name

		if conf.MtraceEnabled {
			// multi trace info
			httpcCtx.TraceMtraceCallerValue = traceCtx.TraceMtraceCallerValue
			httpcCtx.TraceMtracePoidValue = traceCtx.TraceMtracePoidValue
			httpcCtx.TraceMtraceSpecValue = traceCtx.TraceMtraceSpecValue
		}

		httpcCtx.step = agentapi.StartHttpc(traceCtx.Ctx, httpcCtx.StartTime, httpcCtx.Url)
	}

	return httpcCtx, nil
}
func End(httpcCtx *HttpcCtx, status int, reason string, err error) error {
	conf := agentconfig.GetConfig()
	if !conf.Enabled {
		return nil
	}

	elapsed := int32(dateutil.SystemNow() - httpcCtx.StartTime)
	if httpcCtx != nil && httpcCtx.step != nil {
		wCtx := trace.GetAgentTraceContext(httpcCtx.ctx)
		if conf.Debug {
			log.Println("[WA-HTTPC-02001] txid: ", httpcCtx.Txid, ", uri: ", httpcCtx.ServiceName, "\n http url: ", httpcCtx.Url, "\n elapsed: ", elapsed, "ms ", "\n status: ", status, "\n mcallee: ", httpcCtx.TraceMtraceMcallee, "\n error:  ", err)
		}
		if st, ok := httpcCtx.step.(*step.HttpcStepX); ok {
			agentapi.EndHttpc(wCtx, st, elapsed, int32(status), reason, 0, 0, httpcCtx.TraceMtraceMcallee, err)
		}
		CloseHttpcContext(httpcCtx)
		return nil
	}

	if conf.Debug {
		log.Println("[WA-HTTPC-02002] End: Not found Txid ", "\n status: ", status, "\n error:  ", err)
	}
	return fmt.Errorf("HttpcCtx is nil")
}
func Trace(ctx context.Context, host string, port int, url string, elapsed int, status int, reason string, err error) error {
	conf := agentconfig.GetConfig()
	if !conf.Enabled {
		return nil
	}

	var txid int64
	var serviceName string
	var wCtx *agenttrace.TraceContext
	var mcallee int64
	if _, traceCtx := trace.GetTraceContext(ctx); traceCtx != nil {
		wCtx = traceCtx.Ctx
		txid = traceCtx.Txid
		serviceName = traceCtx.Name
	}

	if conf.Debug {
		log.Println("[WA-HTTPC-02001] txid: ", txid, ", uri: ", serviceName, "\n http url: ", url, "\n elapsed: ", elapsed, "ms ", "\n status: ", status, "\n mcallee: ", mcallee, "\n error:  ", err)
	}
	agentapi.ProfileHttpc(wCtx, dateutil.SystemNow(), url, int32(elapsed), int32(status), reason, 0, 0, mcallee, err)
	return nil
}
