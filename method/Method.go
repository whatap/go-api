package method

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"

	agentconfig "github.com/whatap/go-api/agent/agent/config"
	agenttrace "github.com/whatap/go-api/agent/agent/trace"
	agentapi "github.com/whatap/go-api/agent/agent/trace/api"
	"github.com/whatap/go-api/trace"

	"github.com/whatap/golib/lang/step"
	"github.com/whatap/golib/util/dateutil"
)

func Start(ctx context.Context, name string) (*MethodCtx, error) {
	if trace.DISABLE() {
		return PoolMethodContext(), nil
	}
	conf := agentconfig.GetConfig()
	if !conf.ProfileMethodEnabled {
		return PoolMethodContext(), nil
	}
	methodCtx := PoolMethodContext()

	if _, traceCtx := trace.GetTraceContext(ctx); traceCtx != nil {
		methodCtx.StartTime = dateutil.SystemNow()
		methodCtx.Method = name

		methodCtx.ctx = traceCtx
		methodCtx.Txid = traceCtx.Txid
		methodCtx.ServiceName = traceCtx.Name
		methodCtx.step = agentapi.StartMethod(traceCtx.Ctx, methodCtx.StartTime, methodCtx.Method)
		return methodCtx, nil
	}

	return methodCtx, nil
}
func End(methodCtx *MethodCtx, err error) error {
	if trace.DISABLE() {
		return nil
	}

	conf := agentconfig.GetConfig()
	if !conf.ProfileMethodEnabled {
		return nil
	}

	if methodCtx != nil && methodCtx.step != nil {
		elapsed := int32(dateutil.SystemNow() - methodCtx.StartTime)
		wCtx := trace.GetAgentTraceContext(methodCtx.ctx)

		if conf.ProfileMethodStackEnabled {
			methodCtx.Stack = string(debug.Stack())
		}
		if conf.Debug {
			log.Println("[WA-METHOD-01001] txid: ", methodCtx.Txid, ", uri: ", methodCtx.ServiceName, "\n method: ", methodCtx.Method, "\n elapsed: ", elapsed, "ms ", "\n error:  ", err)
		}
		if st, ok := methodCtx.step.(*step.MethodStepX); ok {
			agentapi.EndMethod(wCtx, st, "", elapsed, methodCtx.Cpu, methodCtx.Mem, err)
		}

		CloseMethodContext(methodCtx)
		return nil
	}

	return fmt.Errorf("MethodCtx is nil")
}
func Trace(ctx context.Context, name string, elapsed int, err error) error {
	conf := agentconfig.GetConfig()
	if !conf.ProfileMethodEnabled {
		return nil
	}
	var txid int64
	var serviceName string
	var wCtx *agenttrace.TraceContext
	if _, traceCtx := trace.GetTraceContext(ctx); traceCtx != nil {
		wCtx = traceCtx.Ctx
		txid = traceCtx.Txid
		serviceName = traceCtx.Name
		if conf.Debug {
			log.Println("[WA-METHOD-02001] txid: ", txid, ", uri: ", serviceName, "\n method: ", name, "\n elapsed: ", elapsed, "ms ", "\n error:  ", err)
		}

		agentapi.ProfileMethod(wCtx, dateutil.SystemNow(), name, "", int32(elapsed), 0, 0, err)

		return nil
	}

	return fmt.Errorf("Not found Txid ")
}
