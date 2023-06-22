package api

import (
	"runtime/debug"

	agentconfig "github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/counter/meter"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/agent/stat"
	agenttrace "github.com/whatap/go-api/agent/agent/trace"
	"github.com/whatap/go-api/agent/util/logutil"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/step"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/urlutil"
)

func StartHttpc(ctx *agenttrace.TraceContext, startTime int64, url string) *step.HttpcStepX {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA-API11310", " Recover ", r, "/n", string(debug.Stack()))
		}
	}()
	st := step.NewHttpcStepX()

	HttpcURL := urlutil.NewURL(url)
	nUrl := agenttrace.GetInstanceURLPatternDetector().Normalize(HttpcURL.Path)
	st.Url = hash.HashStr(nUrl)
	st.Host = hash.HashStr(HttpcURL.Host)
	st.Port = int32(HttpcURL.Port)

	// Active status
	if ctx != nil {
		st.StartTime = int32(startTime - ctx.StartTime)
		ctx.ActiveHttpcHash = st.Url
	}
	data.SendHashText(pack.TEXT_HTTPC_URL, st.Url, nUrl)
	data.SendHashText(pack.TEXT_HTTPC_HOST, st.Host, HttpcURL.Host)
	return st
}

func EndHttpc(ctx *agenttrace.TraceContext, st *step.HttpcStepX, elapsed int32, status int32, reason string, cpu, mem, mcallee int64, err error) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA-API11320", " Recover ", r, "/n", string(debug.Stack()))
		}
	}()
	if st == nil {
		return
	}
	conf := agentconfig.GetConfig()
	st.Elapsed = elapsed
	st.Callee = mcallee
	thr := ErrorToThr(err)

	if ctx == nil {
		// 통계만 추가
		if thr != nil {
			stat.GetInstanceStatHttpc().AddHttpcTime(0, st.Url, st.Host, st.Port, st.Elapsed, true)
			//thr.ErrorStack = stackToArray(p.Stack)
			stat.GetInstanceStatError().AddError(thr, thr.ErrorMessage, 0, false, nil, 0, 0)
		} else {
			stat.GetInstanceStatHttpc().AddHttpcTime(0, st.Url, st.Host, st.Port, st.Elapsed, false)
		}
		return
	}

	if conf.ProfileHttpcResourceEnabbled {
		st.StartCpu = int32(cpu)
		st.StartMem = int64(mem)
	}

	if thr != nil {
		ProfileErrorStep(thr, ctx)
		st.Error = ctx.Error
		st.Stack = thr.ErrorStack
	}

	ctx.HttpcCount++
	ctx.HttpcTime += st.Elapsed
	// Active status
	ctx.ActiveHttpcHash = 0

	// DEBUG METER
	//meter.AddHTTPC(st.Host, st.Elapsed, st.Error != 0)
	meter.GetInstanceMeterHTTPC().Add(st.Host, st.Elapsed, st.Error != 0)
	stat.GetInstanceStatHttpc().AddHttpcTime(ctx.ServiceHash, st.Url, st.Host, st.Port, st.Elapsed, st.Error != 0)

	if st.Error == 0 {
		ctx.Profile.Add(st)
	} else {
		ctx.Profile.AddHeavy(st)
	}
}

func ProfileHttpc(ctx *agenttrace.TraceContext, startTime int64, url string, elapsed int32, status int32, reason string, cpu, mem, mcallee int64, err error) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA-API11330", " Recover ", r, "/n", string(debug.Stack()))
		}
	}()
	st := StartHttpc(ctx, startTime, url)
	EndHttpc(ctx, st, elapsed, status, reason, cpu, mem, mcallee, err)
}

func ProfileHttpc1(ctx *agenttrace.TraceContext, startTime int64, url string, elapsed int32, status int32, reason string, cpu, mem, mcallee int64, err error) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA-API11340", " Recover ", r) //, string(debug.Stack()))
		}
	}()
	conf := agentconfig.GetConfig()
	st := step.NewHttpcStepX()

	HttpcURL := urlutil.NewURL(url)
	nUrl := agenttrace.GetInstanceURLPatternDetector().Normalize(HttpcURL.Path)
	st.Url = hash.HashStr(nUrl)
	st.Host = hash.HashStr(HttpcURL.Host)
	st.Port = int32(HttpcURL.Port)
	st.Elapsed = elapsed
	st.Callee = mcallee

	data.SendHashText(pack.TEXT_HTTPC_URL, st.Url, nUrl)
	data.SendHashText(pack.TEXT_HTTPC_HOST, st.Host, HttpcURL.Host)

	thr := ErrorToThr(err)
	// ctx interface 변환 전에 먼저 nil 체크, 기존 panic 보완
	if ctx == nil {
		// 통계만 추가
		if thr != nil {
			stat.GetInstanceStatHttpc().AddHttpcTime(0, st.Url, st.Host, st.Port, st.Elapsed, true)
			//thr.ErrorStack = stackToArray(p.Stack)
			stat.GetInstanceStatError().AddError(thr, thr.ErrorMessage, 0, false, nil, 0, 0)
		} else {
			stat.GetInstanceStatHttpc().AddHttpcTime(0, st.Url, st.Host, st.Port, st.Elapsed, false)
		}
		return
	}

	st.StartTime = int32(startTime - ctx.StartTime)

	if conf.ProfileHttpcResourceEnabbled {
		st.StartCpu = int32(cpu)
		st.StartMem = int64(mem)
	}

	if thr != nil {
		ProfileErrorStep(thr, ctx)
		st.Error = ctx.Error
		st.Stack = thr.ErrorStack
	}

	ctx.HttpcCount++
	ctx.HttpcTime += st.Elapsed

	// DEBUG METER
	//meter.AddHTTPC(st.Host, st.Elapsed, st.Error != 0)
	meter.GetInstanceMeterHTTPC().Add(st.Host, st.Elapsed, st.Error != 0)
	stat.GetInstanceStatHttpc().AddHttpcTime(ctx.ServiceHash, st.Url, st.Host, st.Port, st.Elapsed, st.Error != 0)

	if st.Error == 0 {
		ctx.Profile.Add(st)
	} else {
		ctx.Profile.AddHeavy(st)
	}

}
