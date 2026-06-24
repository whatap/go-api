// github.com/whatap/go-api/httpc
package httpc

import (
	"context"
	"fmt"
	"log"
	"net/http"

	agentconfig "github.com/whatap/go-api/agent/agent/config"
	agentllm "github.com/whatap/go-api/agent/agent/llm"
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

// StartLLM is identical to Start but additionally forces an LLMState attach
// when neither a pending state (llm.Start) nor a URL match was found. Used by
// LLM SDK adapter transports (whataphttp.NewLLMRoundTrip) so every HTTP call
// through an adapter-owned transport is marked as an LLM call regardless of
// URL (mock servers, self-hosted endpoints, etc). Honours whatap.conf::
// llm_enabled — when off, falls through to plain Start behaviour.
//
// §254 Step 5.
func StartLLM(ctx context.Context, url string) (*HttpcCtx, error) {
	httpcCtx, err := Start(ctx, url)
	if err != nil || httpcCtx == nil {
		return httpcCtx, err
	}
	if httpcCtx.Extra == nil {
		if state := agentllm.AttachForced(url); state != nil {
			httpcCtx.Extra = state
		}
	}
	return httpcCtx, nil
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

	// §267 — pending LLMState (registered by llm.Start before the SDK call)
	// wins over URL auto-match. The wrapped RoundTripper inside the SDK is
	// the first httpc.Start in the chain, so it picks up the pending state
	// and updates state.URL with the real request URL. Caller-supplied
	// Provider / OperationType in Config take precedence over URL defaults.
	if httpcCtx.Extra == nil {
		if state := agentllm.TakePending(ctx); state != nil {
			state.SetURL(url)
			httpcCtx.Extra = state
		}
	}

	// §251 — auto-attach LLMState if URL matches a known LLM provider and
	// no pending state was registered. Bind paths overwrite Extra directly
	// before calling httpc.End.
	if httpcCtx.Extra == nil {
		if state := agentllm.MaybeAttachAuto(url); state != nil {
			httpcCtx.Extra = state
		}
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
		var httpcStepId int64
		if httpcCtx.step != nil {
			if st, ok := httpcCtx.step.(*step.HttpcStepX); ok {
				// §261 — LLM API 호출이면 HttpcStepX.Driver = "LLM API" 로 set 해서
				// UI 가 일반 HTTPC 가 아닌 LLM API 호출로 인식하도록 함.
				if _, isLLM := httpcCtx.Extra.(*agentllm.LLMState); isLLM {
					st.Driver = "LLM API"
				}
				agentapi.EndHttpc(wCtx, st, elapsed, int32(status), reason, 0, 0, httpcCtx.StepId, err)
				httpcStepId = st.StepId
			}
		}
		// §251 — sync LLM publish if state was attached (auto or manual).
		if state, ok := httpcCtx.Extra.(*agentllm.LLMState); ok && state != nil {
			var slot *interface{}
			if httpcCtx.ctx != nil {
				slot = &httpcCtx.ctx.LLMTx
				// §261 — mark transaction as LLM so UdpTxEndPack.IsLlm is set at trace.End.
				httpcCtx.ctx.IsLlm = 1
			}
			agentllm.HandleHttpcEnd(state, httpcCtx.Txid, slot, httpcStepId, int64(elapsed), status, err)
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
