package active

import (
	"math"
	//"log"
	"strings"

	"github.com/whatap/golib/lang"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/step"
	"github.com/whatap/golib/lang/value"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/keygen"
	"github.com/whatap/golib/util/stringutil"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/agent/trace"
)

// Java whatap.agent.counter.meter -> whatap.agent.active 로 변경
// trace 와 import cycle 오류

type ActiveStackDump struct {
}

var activeStackDump *ActiveStackDump

func SendActiveStack(txId int64, s string) {

	ctx := trace.GetContext(txId)
	if ctx == nil {
		return
	}
	//sent := 0
	//sent++

	currentTime := dateutil.Now()

	actStack := pack.NewActiveStackPack()
	actStack.Time = currentTime
	actStack.Seq = keygen.Next()
	actStack.ProfileSeq = ctx.ProfileSeq
	actStack.Service = ctx.ServiceHash
	actStack.Elapsed = int32(currentTime - ctx.StartTime)

	// 액티브 스택이 덤프된 상태에서만 프로파일 스텝에 추가한다.
	// 시간을 정교하게 맞춰야한다. 5초간격으로 딱떨어지는것이 필요함
	ctx.ProfileActive++
	st := step.NewActiveStackStep()
	st.Seq = actStack.Seq
	st.HasCallstack = true
	st.StartTime = actStack.Elapsed
	ctx.Profile.AddTail(st)

	//stack
	conf := config.GetConfig()
	se := stringutil.Tokenizer(s, "\n")

	max := math.Min(float64(len(se)), float64(conf.TraceActiveCallstackDepth))
	actStack.CallStack = make([]int32, int32(max))
	if conf.AppType == lang.APP_TYPE_PHP || conf.AppType == lang.APP_TYPE_BSM_PHP {
		// PHP 순방향
		for i := 0; i < int(max); i++ {
			actStack.CallStack[i] = hash.HashStr(se[i])
			actStack.CallStackHash ^= actStack.CallStack[i]
			data.SendHashText(pack.TEXT_STACK_ELEMENTS, actStack.CallStack[i], se[i])
		}
	} else {
		for i := 0; i < int(max); i++ {
			actStack.CallStack[i] = hash.HashStr(se[int(max)-i-1])
			actStack.CallStackHash ^= actStack.CallStack[i]
			data.SendHashText(pack.TEXT_STACK_ELEMENTS, actStack.CallStack[i], se[int(max)-i-1])
		}
	}
	data.Send(actStack)
}
func GetActiveTxList() *value.MapValue {
	out := value.NewMapValue()
	time := out.NewList("time")
	txHash := out.NewList("tx_hash")
	txName := out.NewList("tx_name")
	profile := out.NewList("profile")
	ip := out.NewList("ip")
	userid := out.NewList("userid")
	wClientId := out.NewList("wclientId")
	elapsed := out.NewList("elapsed")
	cputime := out.NewList("cputime")
	malloc := out.NewList("malloc")
	sqlCount := out.NewList("sqlCount")
	sqlTime := out.NewList("sqlTime")
	httpcCount := out.NewList("httpcCount")
	httpcTime := out.NewList("httpcTime")

	threadId := out.NewList("threadId")
	//threadStat := out.NewList("threadStat")

	actDbc := out.NewList("act_dbc")
	actSql := out.NewList("act_sql")
	actHttpc := out.NewList("act_httpc")

	currentTime := dateutil.SystemNow()
	en := trace.GetContextEnumeration()
	for en.HasMoreElements() {
		ctx := en.NextElement().(*trace.TraceContext)
		if ctx == nil {
			continue
		}

		time.AddLong(currentTime)
		txHash.AddLong(int64(ctx.ServiceHash))
		txName.AddString(ctx.ServiceName)
		profile.AddLong(ctx.ProfileSeq)
		ip.AddLong(int64(ctx.RemoteIp))
		userid.AddLong(ctx.WClientId)
		wClientId.AddLong(ctx.WClientId)
		elapsed.AddLong(currentTime - int64(ctx.StartTime))
		if ctx.EndCpu-ctx.StartCpu < 0 {
			cputime.AddLong(int64(0))
		} else {
			cputime.AddLong(int64(ctx.EndCpu - ctx.StartCpu))
		}
		malloc.AddLong(int64(ctx.EndMalloc - ctx.StartMalloc))
		sqlCount.AddLong(int64(ctx.SqlCount))
		sqlTime.AddLong(int64(ctx.SqlTime))
		httpcCount.AddLong(int64(ctx.HttpcTime))
		httpcTime.AddLong(int64(ctx.HttpcTime))

		threadId.AddLong(int64(ctx.ThreadId))
		//threadStat.Add(ThreadStateEnum.getState(ctx.thread))

		actDbc.AddLong(int64(ctx.ActiveDbc))
		actSql.AddLong(int64(ctx.ActiveSqlhash))
		actHttpc.AddLong(int64(ctx.ActiveHttpcHash))

	}
	return out
}

func GetCurrentStackDetail(txId int64, threadStack string) *value.MapValue {
	out := value.NewMapValue()
	ctx := trace.GetContext(txId)

	currentTime := dateutil.SystemNow()
	if ctx != nil && ctx.ProfileSeq == txId {
		out.PutLong("time", currentTime)
		out.PutLong("tx_hash", int64(ctx.ServiceHash))
		out.PutString("tx_name", ctx.ServiceName)
		out.PutLong("profile", ctx.ProfileSeq)
		out.PutLong("ip", int64(ctx.RemoteIp))
		out.PutLong("userid", ctx.WClientId)
		out.PutLong("wclientId", ctx.WClientId)
		out.PutLong("elapsed", currentTime-int64(ctx.StartTime))
		out.PutLong("cputime", int64(ctx.EndCpu-ctx.StartCpu))
		out.PutLong("malloc", int64(ctx.EndMalloc-ctx.StartMalloc))
		out.PutLong("sqlCount", int64(ctx.SqlCount))
		out.PutLong("sqlTime", int64(ctx.SqlTime))
		out.PutLong("fetchCount", int64(ctx.FetchCount))
		out.PutLong("fetchTime", ctx.FetchTime)
		out.PutLong("httpcCount", int64(ctx.HttpcCount))
		out.PutLong("httpcTime", int64(ctx.HttpcTime))

		out.PutLong("threadId", int64(ctx.ThreadId))
		out.PutLong("threadStat", int64(0))

		if ctx.HttpHost != "" {
			out.PutString("httpHost", ctx.HttpHost)
		}
		if ctx.HttpContentType != "" {
			out.PutString("httpContentType", ctx.HttpContentType)
		}
		if ctx.HttpcUrl != "" {
			out.PutString("httpURL", ctx.HttpcUrl)
		}

		if ctx.ActiveSqlhash != 0 {
			out.PutLong("act_dbc", int64(ctx.ActiveDbc))
			out.PutLong("act_sql", int64(ctx.ActiveSqlhash))

			//if(sql.p1!=null) {
			//	out.put("act_sql_p1", new BlobValue(sql.p1));
			//}
			//if(sql.p2!=null) {
			//	out.put("act_sql_p2", new BlobValue(sql.p2));
			//}
			//int tm = ctx.getElapsedTime() - sql.start_time;
			//out.put("act_elapsed", tm < 0 ? 0 : tm);
		}

		if ctx.ActiveSqlhash != 0 {
			out.PutLong("act_dbc", int64(ctx.ActiveDbc))
			out.PutLong("act_sql", int64(ctx.ActiveSqlhash))

			//if(sql.p1!=null) {
			//	out.put("act_sql_p1", new BlobValue(sql.p1));
			//}
			//if(sql.p2!=null) {
			//	out.put("act_sql_p2", new BlobValue(sql.p2));
			//}
			//int tm = ctx.getElapsedTime() - sql.start_time;
			//out.put("act_elapsed", tm < 0 ? 0 : tm);
		}

		if ctx.ActiveHttpcHash != 0 {
			out.PutLong("act_httpc", int64(ctx.ActiveHttpcHash))

			//out.put("act_httpc_host", ctx.httpc_host);
			//out.put("act_httpc_port", ctx.httpc_port);
			//int tm = ctx.getElapsedTime() - ctx.httpc_stime;
			//out.put("act_elapsed", tm <0?0:tm);
		}
		out.PutString("method", ctx.HttpMethod)

		if ctx.Mtid != 0 {
			out.PutLong("mtid", ctx.Mtid)
			out.PutLong("mdepth", int64(ctx.Mdepth))
			out.PutLong("mcaller", ctx.McallerTxid)
			out.PutLong("mcaller_pcode", ctx.McallerPcode)
		}

		se := strings.Split(threadStack, "\n")
		if se != nil && len(se) > 1 {
			stack := value.NewListValue(nil)
			for _, s := range se {
				stack.AddString(s)
			}
			out.Put("callstack", stack)
		}
	}
	return out
}
