// github.com/whatap/go-api/httpc
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
)

func GetMTrace(httpcCtx *HttpcCtx) http.Header {
	if trace.DISABLE() {
		return make(http.Header)
	}

	rt := make(http.Header)
	conf := agentconfig.GetConfig()
	if conf.MtraceEnabled && httpcCtx.TraceMtraceCallerValue != "" {
		rt.Set(conf.TraceMtraceTraceparentKey, httpcCtx.TraceMtraceTraceparentValue)
		rt.Set(conf.TraceMtraceCallerKey, httpcCtx.TraceMtraceCallerValue)
		rt.Set(conf.TraceMtracePoidKey, httpcCtx.TraceMtracePoidValue)
		rt.Set(conf.TraceMtraceSpecKey1, httpcCtx.TraceMtraceSpecValue)
	}
	// 2023.11.07 deprcated
	// Mcallee
	// if conf.MtraceCalleeTxidEnabled {
	// 	httpcCtx.TraceMtraceMcallee = keygen.Next()
	// 	rt.Set(conf.TraceMtraceCalleeKey, fmt.Sprintf("%d", httpcCtx.TraceMtraceMcallee))
	// }

	return rt
}

func Start(ctx context.Context, url string) (*HttpcCtx, error) {
	if trace.DISABLE() {
		return PoolHttpcContext(), nil
	}

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
		st := agentapi.StartHttpc(traceCtx.Ctx, httpcCtx.StartTime, httpcCtx.Url)
		st.StepId = traceCtx.MStepId
		httpcCtx.StepId = traceCtx.MStepId
		httpcCtx.step = st
	}

	return httpcCtx, nil
}
func End(httpcCtx *HttpcCtx, status int, reason string, err error) error {
	if trace.DISABLE() {
		return nil
	}

	conf := agentconfig.GetConfig()
	if !conf.Enabled {
		return nil
	}

	elapsed := int32(dateutil.SystemNow() - httpcCtx.StartTime)
	if httpcCtx != nil {
		wCtx := trace.GetAgentTraceContext(httpcCtx.ctx)
		if conf.Debug {
			log.Println("[WA-HTTPC-02001] txid: ", httpcCtx.Txid, ", uri: ", httpcCtx.ServiceName, "\n http url: ", httpcCtx.Url, "\n elapsed: ", elapsed, "ms ", "\n status: ", status, "\n step id: ", httpcCtx.StepId, "\n error:  ", err)
		}
		if httpcCtx.step != nil {
			if st, ok := httpcCtx.step.(*step.HttpcStepX); ok {
				agentapi.EndHttpc(wCtx, st, elapsed, int32(status), reason, 0, 0, httpcCtx.StepId, err)
			}
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
	if trace.DISABLE() {
		return nil
	}

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
