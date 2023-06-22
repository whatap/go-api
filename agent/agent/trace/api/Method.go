package api

import (
	"runtime/debug"
	"strings"

	agentconfig "github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/data"
	agenttrace "github.com/whatap/go-api/agent/agent/trace"
	"github.com/whatap/go-api/agent/util/logutil"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/step"
	"github.com/whatap/golib/util/hash"
)

func StartMethod(ctx *agenttrace.TraceContext, startTime int64, method string) *step.MethodStepX {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA-API11210", " Recover ", r, "/n", string(debug.Stack()))
		}
	}()
	st := step.NewMethodStepX()

	if !strings.HasSuffix(method, ")") {
		method = method + "()"
	}

	st.Hash = hash.HashStr(method)
	data.SendHashText(pack.TEXT_METHOD, st.Hash, method)

	if ctx != nil {
		st.StartTime = int32(startTime - ctx.StartTime)
	}
	return st
}
func EndMethod(ctx *agenttrace.TraceContext, st *step.MethodStepX, methodStack string, elapsed int32, cpu, mem int64, err error) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA-API11220", " Recover ", r, "/n", string(debug.Stack()))
		}
	}()

	if ctx == nil || st == nil {
		return
	}
	conf := agentconfig.GetConfig()

	st.Elapsed = elapsed
	if methodStack != "" {
		st.SetTrue(2)
		st.Stack = agenttrace.StackToArray(methodStack)
	}

	if conf.ProfileMethodResourceEnabled {
		st.SetTrue(1)
		st.StartCpu = int32(cpu)
		st.StartMem = int32(mem)
	}
	ctx.Profile.Add(st)
}

func ProfileMethod(ctx *agenttrace.TraceContext, startTime int64, method, methodStack string, elapsed int32, cpu, mem int64, err error) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA-API11230", " Recover ", r, "/n", string(debug.Stack()))
		}
	}()
	st := StartMethod(ctx, startTime, method)
	EndMethod(ctx, st, methodStack, elapsed, cpu, mem, err)
}
