package api

import (
	"fmt"
	"runtime/debug"
	"strings"

	agentconfig "github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/counter/meter"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/agent/secure"
	"github.com/whatap/go-api/agent/agent/stat"
	agenttrace "github.com/whatap/go-api/agent/agent/trace"
	"github.com/whatap/go-api/agent/util/logutil"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/ref"
	"github.com/whatap/golib/lang/service"
	"github.com/whatap/golib/lang/step"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/hexa32"
	"github.com/whatap/golib/util/stringutil"
	"github.com/whatap/golib/util/urlutil"
)

var (
	ERROR_MSG_TITLE      = "ERROR"
	ERROR_MSG_TITLE_HASH = hash.HashStr("ERROR")
)

func StartTx(ctx *agenttrace.TraceContext) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA-API11010", " Recover ", r, "/n", string(debug.Stack()))
		}
	}()

	if ctx == nil {
		return
	}
	conf := agentconfig.GetConfig()

	meter.GetInstanceMeterService().Arrival++
	if ctx.ServiceURL == nil {
		ctx.ServiceURL = urlutil.NewURL("Unknown")
	}
	urlHash := hash.HashStr(ctx.ServiceURL.Path)
	if conf.TraceNormalizeEnabled {
		ctx.ServiceName = agenttrace.GetInstanceServiceURLPatternDetector().Normalize(ctx.ServiceURL.Path)
	} else {
		ctx.ServiceName = ctx.ServiceURL.Path
	}
	ctx.ServiceHash = hash.HashStr(ctx.ServiceName)
	normalizeServiceName := ctx.ServiceName

	data.SendHashText(pack.TEXT_SERVICE, ctx.ServiceHash, ctx.ServiceName)

	if urlHash != ctx.ServiceHash {
		agenttrace.AddMessage(ctx, 0, 0, "OriginURL", "", ctx.ServiceURL.Path, 0, false)
	}

	if stringutil.InArray(normalizeServiceName, conf.IgnoreHttpMethodUrls) || stringutil.InArray(ctx.ServiceURL.Path, conf.IgnoreHttpMethodUrls) {
		if stringutil.InArray(ctx.HttpMethod, conf.IgnoreHttpMethod) {
			return
		}
	}

	if conf.ProfileHttpHostEnabled {
		if ctx.ServiceURL.Host != "" {
			if strings.HasPrefix(ctx.ServiceName, "/") {
				ctx.ServiceName = "/" + ctx.ServiceURL.HostPort() + ctx.ServiceName
			} else {
				ctx.ServiceName = "/" + ctx.ServiceURL.HostPort() + "/" + ctx.ServiceName
			}
			ctx.ServiceHash = hash.HashStr(ctx.ServiceName)
		}
	}

	if conf.QueryStringEnabled {
		qs := agenttrace.MatchQueryString(normalizeServiceName, ctx.ServiceURL, conf.QueryStringUrls, conf.QueryStringKeys)
		if qs != "" {
			ctx.ServiceName = ctx.ServiceName + "?" + qs
			ctx.ServiceHash = hash.HashStr(ctx.ServiceName)
		}
	}

	if !ctx.IsStaticContents {
		ctx.IsStaticContents = agentconfig.IsIgnoreTrace(ctx.ServiceHash, ctx.ServiceName)
	}

	// 도메인
	ctx.HttpHost = ctx.ServiceURL.HostPort()
	ctx.HttpHostHash = hash.HashStr(ctx.ServiceURL.HostPort())
	ctx.HttpMethod = strings.ToUpper(ctx.HttpMethod)
	data.SendHashText(pack.TEXT_HTTP_DOMAIN, ctx.HttpHostHash, ctx.HttpHost)

	if conf.TraceUserAgentEnabled {
		ctx.UserAgent = hash.HashStr(ctx.UserAgentString)
		data.SendHashText(pack.TEXT_USER_AGENT, ctx.UserAgent, ctx.UserAgentString)

	}
	if conf.TraceRefererEnabled {
		ctx.Referer = hash.HashStr(ctx.RefererURL.String())
		data.SendHashText(pack.TEXT_REFERER, ctx.Referer, ctx.RefererURL.String())
	}

	ctx.ProfileSeq = ctx.Txid

	data.SendHashText(pack.TEXT_SERVICE, ctx.ServiceHash, ctx.ServiceName)

	meter.AddMeterUsers(ctx.WClientId)
	agenttrace.PutContext(ctx.Txid, ctx)
}

func EndTx(ctx *agenttrace.TraceContext) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA-API11020", " Recover ", r, "/n", string(debug.Stack()))
		}
	}()

	if ctx == nil {
		return
	}

	agenttrace.RemoveContext(ctx.Txid)
	ctx.Elapsed = int32(dateutil.SystemNow() - ctx.StartTime)

	if ctx.IsStaticContents {
		return
	}

	// Transaction 시작 시간을 에이전트 시간으로 변경.
	ctx.EndTime = dateutil.Now()
	ctx.StartTime = ctx.EndTime - int64(ctx.Elapsed)

	poid := strings.Split(ctx.McallerPoidKey, ",")
	if len(poid) > 0 {
		ctx.McallerPcode = hexa32.ToLong32(strings.TrimSpace(poid[0]))
	}
	if len(poid) > 1 {
		ctx.McallerOkind = int32(hexa32.ToLong32(strings.TrimSpace(poid[1])))
	}
	if len(poid) > 2 {
		ctx.McallerOid = int32(hexa32.ToLong32(strings.TrimSpace(poid[2])))
	}

	tx := service.NewTxRecord()
	tx.Txid = ctx.ProfileSeq
	tx.EndTime = ctx.EndTime
	tx.Elapsed = ctx.Elapsed
	tx.Service = ctx.ServiceHash

	tx.IpAddr = ctx.RemoteIp
	tx.WClientId = ctx.WClientId
	tx.UserAgent = ctx.UserAgent

	tx.McallerPcode = ctx.McallerPcode
	tx.McallerOkind = ctx.McallerOkind
	tx.McallerOid = ctx.McallerOid
	tx.Mtid = ctx.Mtid
	tx.Mdepth = ctx.Mdepth
	tx.Mcaller = ctx.McallerTxid

	tx.Cipher = secure.GetParamSecurity().KeyHash

	// Cpu, Meory 음수 처리
	if ctx.EndCpu < 0 {
		tx.CpuTime = -1
	} else {
		tx.CpuTime = int32(ctx.EndCpu - ctx.StartCpu)
	}

	if ctx.EndMalloc < 0 {
		tx.Malloc = -1
	} else {
		tx.Malloc = ctx.EndMalloc - ctx.StartMalloc
	}

	tx.SqlCount = ctx.SqlCount
	tx.SqlTime = ctx.SqlTime
	tx.SqlFetchCount = ctx.RsCount
	tx.SqlFetchTime = int32(ctx.RsTime)
	tx.DbcTime = ctx.DbcTime

	if ctx.Error != 0 {
		tx.Error = ctx.Error
	}
	// BixException 통계 제외 처리를 위해
	tx.ErrorLevel = ctx.ErrorLevel

	tx.Domain = ctx.HttpHostHash
	tx.Referer = ctx.Referer

	tx.HttpcCount = ctx.HttpcCount
	tx.HttpcTime = ctx.HttpcTime

	tx.Status = ctx.Status

	if ctx.HttpMethod != "" {
		tx.HttpMethod = service.WebMethodName[ctx.HttpMethod]
	}

	tx.Fields = ctx.GetFields()

	// ctx를 보내고 싶지만, import cycle 오류 발생.
	meter.GetInstanceMeterService().Add(tx, ctx.McallerPcode, ctx.McallerOkind, ctx.McallerOid)

	agenttrace.SendTransaction(ctx)
}

func ProfileMsg(ctx *agenttrace.TraceContext, title, message string, elapsed, value int32) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA-API11030", " Recover ", r, "/n", string(debug.Stack()))
		}
	}()
	if ctx == nil {
		return
	}
	st := step.NewMessageStep()
	now := dateutil.SystemNow()
	st.StartTime = int32(now - ctx.StartTime)
	st.Time = int32(elapsed)
	st.Hash = int32(hash.HashStr(title))
	st.Value = int32(value)
	st.Desc = message
	data.SendHashText(pack.TEXT_MESSAGE, st.Hash, title)

	ctx.Profile.Add(st)
}

func ProfileSecureMsg(ctx *agenttrace.TraceContext, title, message string, elapsed, value int32) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA-API11040", " Recover ", r, "/n", string(debug.Stack()))
		}
	}()
	if ctx == nil {
		return
	}

	conf := agentconfig.GetConfig()
	if conf.ProfileHttpParameterEnabled && strings.HasPrefix(ctx.ServiceName, conf.ProfileHttpParameterUrlPrefix) {
		st := step.NewSecureMsgStep()

		st.StartTime = int32(dateutil.SystemNow() - ctx.StartTime)
		st.Hash = int32(hash.HashStr(title))
		sb := stringutil.NewStringBuffer()
		sb.Append(message)
		// append get parameter from url query_string
		if title == "GET Parameter" {
			m := ctx.ServiceURL.ParseQuery()
			for k, v := range m {
				if strings.Index(message, k) > -1 {
					continue
				}
				sb.Append(k)
				if len(v) > 1 {
					sb.AppendLine("=ARRAY")
				} else {
					sb.Append("=").AppendLine(v[0])
				}
			}
		}
		crc := ref.NewBYTE()
		st.Value = agenttrace.ToParamBytes(sb.ToString(), crc)
		st.Crc = crc.Value

		data.SendHashText(pack.TEXT_MESSAGE, st.Hash, title)

		ctx.Profile.Add(st)
	}

}

func ProfileErrorStep(thr *stat.ErrorThrowable, ctx *agenttrace.TraceContext) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA-API11050", " Recover ", r, "/n", string(debug.Stack()))
		}
	}()
	conf := agentconfig.GetConfig()
	if agenttrace.IsIgnoreException(thr) {
		return
	}

	ctx.Thr = thr
	if agenttrace.IsBizException(thr) {
		ctx.Error = stat.GetInstanceStatError().AddError(ctx.Thr, ctx.Thr.ErrorMessage, ctx.ServiceHash, conf.ErrorSnapEnabled, ctx.Profile.GetStep4Error(), 0, 0)
		ctx.ErrorLevel = pack.INFO
	} else {
		ctx.Error = stat.GetInstanceStatError().AddError(ctx.Thr, ctx.Thr.ErrorMessage, ctx.ServiceHash, conf.ErrorSnapEnabled, ctx.Profile.GetStep4Error(), 0, 0)
		ctx.ErrorLevel = pack.WARNING
	}
}

func ProfileError(ctx *agenttrace.TraceContext, err error) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA-API11060", " Recover ", r, "/n", string(debug.Stack()))
		}
	}()
	thr := stat.NewErrorThrowable()
	thr.ErrorClassName = fmt.Sprintf("%T", err)
	thr.ErrorMessage = err.Error()
	// To-Do
	// thr.ErrorStack = stackToArray(p.Stack)

	if agenttrace.IsIgnoreException(thr) {
		return
	}

	if ctx == nil {
		stat.GetInstanceStatError().AddError(thr, thr.ErrorMessage, 0, false, nil, 0, 0)
		return
	}

	conf := agentconfig.GetConfig()

	st := step.NewMessageStep()

	st.StartTime = int32(dateutil.SystemNow() - ctx.StartTime)
	st.Time = 0
	//st.Hash = int32(hash.HashStr(thr.ErrorClassName))
	st.Hash = ERROR_MSG_TITLE_HASH
	st.Value = 0

	data.SendHashText(pack.TEXT_MESSAGE, st.Hash, ERROR_MSG_TITLE)

	// Error를 Message로 출력할 때 Message Desc 는 해시 처리 되지 않고 텍스트로 전달 되기 때문에 길이를 잘라서 설정.
	msg := stringutil.TrimEmpty(fmt.Sprintf("%s-%s", thr.ErrorClassName, thr.ErrorMessage))
	//msg = stringutil.Truncate(msg, 400)
	st.Desc = msg

	//ctx.Profile.Add(st)
	// Error 는 무조건 step 삽입
	ctx.Profile.AddHeavy(st)

	// 임재환 추가 java Thread thr 대신 ErrorThrowable 구조체 사용
	ctx.Thr = thr
	if agenttrace.IsBizException(thr) {
		ctx.Error = stat.GetInstanceStatError().AddError(ctx.Thr, ctx.Thr.ErrorMessage, ctx.ServiceHash, conf.ErrorSnapEnabled, ctx.Profile.GetStep4Error(), 0, 0)
		ctx.ErrorLevel = pack.INFO
	} else {
		ctx.Error = stat.GetInstanceStatError().AddError(ctx.Thr, ctx.Thr.ErrorMessage, ctx.ServiceHash, conf.ErrorSnapEnabled, ctx.Profile.GetStep4Error(), 0, 0)
		ctx.ErrorLevel = pack.WARNING
	}
}

func ErrorToThr(err error) *stat.ErrorThrowable {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA-API11070", " Recover ", r) //, string(debug.Stack()))
		}
	}()
	if err == nil {
		return nil
	}
	thr := stat.NewErrorThrowable()
	thr.ErrorClassName = fmt.Sprintf("%T", err)
	thr.ErrorMessage = err.Error()
	return thr
}
