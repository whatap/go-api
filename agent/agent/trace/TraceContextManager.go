package trace

import (
	"math"

	//"runtime"
	"bytes"
	"strconv"
	"strings"
	"sync"

	//"time"

	//	"log"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/counter/meter"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/agent/secure"
	"github.com/whatap/go-api/agent/agent/stat"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/golib/io"
	"github.com/whatap/golib/lang"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/pack/udp"
	"github.com/whatap/golib/lang/ref"
	"github.com/whatap/golib/lang/service"
	"github.com/whatap/golib/lang/step"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/hexa32"
	"github.com/whatap/golib/util/hmap"
	"github.com/whatap/golib/util/iputil"
	"github.com/whatap/golib/util/stringutil"
	"github.com/whatap/golib/util/urlutil"
)

var conf *config.Config = config.GetConfig()

// var ctxTable *hmap.LongKeyLinkedMap = hmap.NewLongKeyLinkedMap().SetMax(5000)
var ctxTable *hmap.LongKeyLinkedMap = hmap.NewLongKeyLinkedMap(int(conf.TxDefaultCapacity), conf.TxDefaultLoadFactor).SetMax(int(conf.TxMaxCount))
var ctxLock sync.Mutex

const (
	//	PACKET_BLANK = 0
	//	,PACKET_REQUEST = 1
	//	,PACKET_DB_CONN = 2
	//	,PACKET_DB_FETCH = 3
	//	,PACKET_DB_SQL = 4			// DB SQL start ~ end
	//	,PACKET_DB_SQL_START = 5
	//	,PACKET_DB_SQL_END = 6
	//	,PACKET_HTTPC = 7
	//	,PACKET_HTTPC_START = 8
	//	,PACKET_HTTPC_END = 9
	//	,PACKET_ERROR = 10
	//	,PACKET_REQUEST_END = 255
	//	,PACKET_PARAM = 30

	TX_BLANK     uint8 = 0
	TX_START     uint8 = 1
	TX_DB_CONN   uint8 = 2
	TX_DB_FETCH  uint8 = 3
	TX_SQL       uint8 = 4
	TX_SQL_START uint8 = 5
	TX_SQL_END   uint8 = 6

	TX_HTTPC       uint8 = 7
	TX_HTTPC_START uint8 = 8
	TX_HTTPC_END   uint8 = 9

	TX_ERROR  uint8 = 10
	TX_MSG    uint8 = 11
	TX_METHOD uint8 = 12

	// secure msg
	TX_SECURE_MSG uint8 = 13

	// sql & param
	TX_SQL_PARAM       uint8 = 14
	TX_SQL_PARAM_NAMED uint8 = 14

	TX_PARAM     uint8 = 30
	ACTIVE_STACK uint8 = 40
	ACTIVE_STATS uint8 = 41
	DBCONN_POOL  uint8 = 42

	// golang config
	CONFIG_INFO uint8 = 230

	// relay pack
	RELAY_PACK uint8 = 244

	TX_START_END uint8 = 254
	TX_END       uint8 = 255
)

func TxPut(up udp.UdpPack) {
	ctxLock.Lock()
	defer ctxLock.Unlock()
	if up == nil {
		return
	}
	//logutil.Infoln("WA569-02", "TraceContextMain TxPut Type=", up.GetPackType())
	switch up.GetPackType() {
	case udp.TX_START:
		p := up.(*udp.UdpTxStartPack)
		p.Process()
		startTx(p)
	case TX_START_END:
		p := up.(*udp.UdpTxStartEndPack)
		p.Process()
		startEndTx(p)
	case TX_END:
		p := up.(*udp.UdpTxEndPack)
		p.Process()
		endTx(p)
	case TX_SQL:
		p := up.(*udp.UdpTxSqlPack)
		p.Process()
		profileSql(p)
	case TX_SQL_PARAM:
		p := up.(*udp.UdpTxSqlParamPack)
		p.Process()
		profileSqlParam(p)
	case TX_HTTPC:
		p := up.(*udp.UdpTxHttpcPack)
		p.Process()
		profileHttpc(p)
	case TX_ERROR:
		p := up.(*udp.UdpTxErrorPack)
		p.Process()
		profileError(p)
	case TX_MSG:
		p := up.(*udp.UdpTxMessagePack)
		p.Process()
		profileMsg(p)
	case TX_SECURE_MSG:
		p := up.(*udp.UdpTxSecureMessagePack)
		p.Process()
		profileSecureMsg(p)
	case TX_METHOD:
		p := up.(*udp.UdpTxMethodPack)
		p.Process()
		profileMethod(p)
	case TX_DB_CONN:
		p := up.(*udp.UdpTxDbcPack)
		p.Process()
		profileDBC(p)
	}
}

func startTx(p *udp.UdpTxStartPack) {
	//logutil.Infoln(">>>>", "StartTx ", p.Txid)
	if conf.TraceDaemonEnabled && conf.TraceDaemonUrls.Contains(p.Uri) {
		//logutil.Println("WA560-00", "Daemon ", p.Uri)
		return
	}

	if !conf.TraceCLIEnabled && p.Host == "CLI" {
		//logutil.Println("WA560-01", "Ignore CLI ", p.Uri)
		return
	}

	// DEBUG arrival_rate
	meter.GetInstanceMeterService().Arrival++

	// TraceMain
	//ctx := NewTraceContext()
	ctx := PoolTraceContext()
	ctx.IsStaticContents = p.IsStatic

	urlHash := hash.HashStr(p.ServiceURL.Path)
	ctx.ServiceURL = p.ServiceURL
	// Nomalize boot.ServiceURLPatternDetector
	if conf.TraceNormalizeEnabled {
		ctx.ServiceName = GetInstanceServiceURLPatternDetector().Normalize(p.ServiceURL.Path)
	} else {
		ctx.ServiceName = p.ServiceURL.Path
	}
	ctx.ServiceHash = hash.HashStr(ctx.ServiceName)
	normalizeServiceName := ctx.ServiceName

	// host, query string 추가 전 서비스 해시도 등록, mtrace_spec1에서 caller url 이 해시로 변경되면 추가.
	data.SendHashText(pack.TEXT_SERVICE, ctx.ServiceHash, ctx.ServiceName)

	if urlHash != ctx.ServiceHash {
		addMessage(ctx, 0, 0, "OriginURL", "", p.ServiceURL.Path, 0, false)
	}

	// 특정 url 에 대해서 http method를 판단해서 수집 제외 여부를 판단합니다.
	if stringutil.InArray(p.HttpMethod, conf.IgnoreHttpMethod) {
		//logutil.Println("WA560-02", "Ignore Http method  ", p.HttpMethod, ",", normalizeServiceName, ",", p.ServiceURL.Path)
		return
	}

	// Http ServiceName을 기존 URI에서 HOST를 포함한 형식으로 출력   /HOST/URI , Default false
	// 위에서 결정된 ServiceURL 에 /HOST 추가
	if conf.ProfileHttpHostEnabled {
		if p.ServiceURL.Host != "" {
			if strings.HasPrefix(ctx.ServiceName, "/") {
				ctx.ServiceName = "/" + p.ServiceURL.HostPort() + ctx.ServiceName
			} else {
				ctx.ServiceName = "/" + p.ServiceURL.HostPort() + "/" + ctx.ServiceName
			}
			ctx.ServiceHash = hash.HashStr(ctx.ServiceName)
		}
	}

	// normalize 이후에 Uri 변경 후 적용 UDPManager 에서 옮겨옴
	if conf.QueryStringEnabled {
		qs := MatchQueryString(normalizeServiceName, p.ServiceURL, conf.QueryStringUrls, conf.QueryStringKeys)
		if qs != "" {
			ctx.ServiceName = ctx.ServiceName + "?" + qs
			ctx.ServiceHash = hash.HashStr(ctx.ServiceName)
		}
	}

	if !ctx.IsStaticContents {
		ctx.IsStaticContents = config.IsIgnoreTrace(ctx.ServiceHash, ctx.ServiceName)
	}

	// 도메인
	ctx.HttpHost = p.ServiceURL.HostPort()
	ctx.HttpHostHash = hash.HashStr(p.ServiceURL.HostPort())
	ctx.HttpMethod = strings.ToUpper(p.HttpMethod)
	data.SendHashText(pack.TEXT_HTTP_DOMAIN, ctx.HttpHostHash, ctx.HttpHost)

	ctx.StartTime = p.Time
	// 문법 : X-Forwarded-For: <client>, <proxy1>, <proxy2>
	if strings.Index(p.Ipaddr, ",") > -1 {
		ipArray := strings.Split(p.Ipaddr, ",")
		if len(ipArray) > 1 {
			p.Ipaddr = ipArray[0]
		}
	}
	ctx.RemoteIp = io.ToInt(iputil.ToBytes(p.Ipaddr), 0) // getRemoteAddr
	ctx.StartCpu = p.Cpu
	ctx.StartMalloc = p.Mem

	if conf.TraceUserAgentEnabled {
		ctx.UserAgentString = p.UAgent
		ctx.UserAgent = hash.HashStr(p.UAgent)
		data.SendHashText(pack.TEXT_USER_AGENT, ctx.UserAgent, ctx.UserAgentString)

	}
	if conf.TraceRefererEnabled {
		ctx.Referer = hash.HashStr(p.RefererURL.String())
		ctx.RefererURL = urlutil.NewURL(p.RefererURL.String())
		data.SendHashText(pack.TEXT_REFERER, ctx.Referer, p.Ref)
	}

	ctx.Txid = p.Txid
	ctx.Pid = p.Pid
	ctx.ThreadId = p.ThreadId
	ctx.ProfileSeq = p.Txid

	// WClientId 설정
	// PHP Extention 에서 보내오는 WClientId 를 설정, 만약 WClientId 값이 없으면 RemoteIp 로 설정
	if conf.TraceUserEnabled {
		if p.WClientId == "" {
			ctx.WClientId = int64(ctx.RemoteIp)
		} else {
			ctx.WClientId = int64(hash.HashStr(p.WClientId))
		}
	}

	data.SendHashText(pack.TEXT_SERVICE, ctx.ServiceHash, ctx.ServiceName)

	meter.AddMeterUsers(ctx.WClientId)
	ctxTable.Put(p.Txid, ctx)
}

func endTx(p *udp.UdpTxEndPack) {
	// ctx interface 변환 전에 먼저 nil 체크, 기존 panic 보완
	var ctx *TraceContext
	ctxIf := ctxTable.Remove(p.Txid)
	if ctxIf == nil {
		if !conf.TraceCLIEnabled && p.Host == "CLI" {
			//logutil.Println("WA560-02", "Ignore CLI ", p.Uri)
		} else {
			// lost connection 된 request end 가 5분이 넘어서 오는 경우 있음. segfault 는 아니지만 end만 가지고는 처리 안하게 변경
			logutil.Println("WA560", "endTx not found Txid=", p.Txid, ",content=", p.Host+p.Uri, ",elapsed=", p.Elapsed)
		}
		return
	} else {
		ctx = ctxIf.(*TraceContext)
		if ctx == nil {
			return
		}
		ctx.Elapsed = int32(p.Time - ctx.StartTime)
	}

	if ctx.IsStaticContents {
		return
	}

	ctx.EndCpu = p.Cpu
	ctx.EndMalloc = p.Mem

	// Transaction 시작 시간을 에이전트 시간으로 변경.
	ctx.EndTime = dateutil.Now()
	ctx.StartTime = ctx.EndTime - int64(ctx.Elapsed)

	// set mtid
	ctx.Mtid = p.Mtid
	ctx.Mdepth = p.Mdepth
	ctx.McallerTxid = p.McallerTxid
	ctx.McallerPcode = p.McallerPcode
	ctx.McallerSpec = p.McallerSpec
	ctx.McallerUrl = p.McallerUrl
	ctx.McallerUrlHash = p.McallerUrlHash

	//logutil.Infoln(">>>>", "ctx.Status = ", p.Status)
	ctx.Status = p.Status

	poid := strings.Split(p.McallerPoidKey, ",")
	if len(poid) > 0 {
		ctx.McallerPcode = hexa32.ToLong32(strings.TrimSpace(poid[0]))
	}
	if len(poid) > 1 {
		ctx.McallerOkind = int32(hexa32.ToLong32(strings.TrimSpace(poid[1])))
	}
	if len(poid) > 2 {
		ctx.McallerOid = int32(hexa32.ToLong32(strings.TrimSpace(poid[2])))
	}

	// TraceMain 에서 옮겨옴. - Hitmap, tps, 등의 통계를 더 빠르게 처리.
	//meter.GetInstanceMeterService().Add(ctx.ServiceHash, ctx.Elapsed, ctx.Error != 0, ctx.ErrorLevel, ctx.McallerPcode, ctx.McallerOkind, ctx.McallerOid)
	tx := service.NewTxRecord()
	tx.Txid = ctx.ProfileSeq
	// TraceContextManager에서 구한 현재시간보다 더 느려질 가능성 있음. elpased 오차 발생
	// tx.EndTime = dateutil.Now()
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

	tx.Fields = ctx.ExtraFields()

	// ctx를 보내고 싶지만, import cycle 오류 발생.
	meter.GetInstanceMeterService().Add(tx, ctx.McallerPcode, ctx.McallerOkind, ctx.McallerOid)

	// DEBUG Queue
	if conf.QueueProfileEnabled == false {
		sendTransactionQue <- ctx
	} else {
		if profileQueue != nil {
			profileQueue.PutForce(ctx)
		}
	}

	// DEBUG
	//log.Println("EndTx Txid=", p.Txid, "size=", ctxTable.Size(), ",len=", len(sendTransactionQue) )
	//logutil.Infoln("endTx host=", ctx.Host, ", uri=", ctx.Uri)
	//	if ctx.Mtid != 0 {
	//		logutil.Infoln("endTx Mtid=", ctx.Mtid, ", Mdepth=", ctx.Mdepth, ", McallerTexid=", ctx.McallerTxid)
	//		logutil.Infoln("endTx McallerPcode=", ctx.McallerPcode, ", McallerSpec=", ctx.McallerSpec, ", McallerUrl=", ctx.McallerUrl)
	//	}
}

func startEndTx(p *udp.UdpTxStartEndPack) {
	//logutil.Infoln(">>>>", "Start End Tx ", p.Txid)
	if conf.TraceDaemonEnabled && conf.TraceDaemonUrls.Contains(p.Uri) {
		//logutil.Println("WA560-00", "Daemon ", p.Uri)
		return
	}

	if !conf.TraceCLIEnabled && p.Host == "CLI" {
		//logutil.Println("WA560-01", "Ignore CLI ", p.Uri)
		return
	}

	// DEBUG arrival_rate
	meter.GetInstanceMeterService().Arrival++

	// TraceMain
	//ctx := NewTraceContext()
	ctx := PoolTraceContext()
	//logutil.Infoln(">>>>", "StartEnd PoolTraceContext txid=", p.Txid)
	ctx.IsStaticContents = p.IsStatic
	// ignore static contents
	if ctx.IsStaticContents {
		return
	}

	urlHash := hash.HashStr(p.ServiceURL.Path)
	// Nomalize boot.ServiceURLPatternDetector
	if conf.TraceNormalizeEnabled {
		ctx.ServiceName = GetInstanceServiceURLPatternDetector().Normalize(p.ServiceURL.Path)
	} else {
		ctx.ServiceName = p.ServiceURL.Path
	}
	ctx.ServiceHash = hash.HashStr(ctx.ServiceName)
	normalizeServiceName := ctx.ServiceName

	// host, query string 추가 전 서비스 해시도 등록, mtrace_spec1에서 caller url 이 해시로 변경되면 추가.
	data.SendHashText(pack.TEXT_SERVICE, ctx.ServiceHash, ctx.ServiceName)

	if urlHash != ctx.ServiceHash {
		addMessage(ctx, 0, 0, "OriginURL", "", p.ServiceURL.Path, 0, false)
	}
	// 특정 url 에 대해서 http method를 판단해서 수집 제외 여부를 판단합니다.
	if config.InArray(p.HttpMethod, conf.IgnoreHttpMethod) {
		//logutil.Println("WA560-03", "Ignore Http method  ", p.HttpMethod, ",", normalizeServiceName, ",", p.ServiceURL.Path)
		return
	}
	// Http ServiceName을 기존 URI에서 HOST를 포함한 형식으로 출력   /HOST/URI , Default false
	// 위에서 결정된 ServiceURL 에 /HOST 추가
	if conf.ProfileHttpHostEnabled {
		if p.ServiceURL.Host != "" {
			if strings.HasPrefix(p.ServiceURL.Path, "/") {
				ctx.ServiceName = "/" + p.ServiceURL.Host + ctx.ServiceName
			} else {
				ctx.ServiceName = "/" + p.ServiceURL.Host + "/" + ctx.ServiceName
			}
			ctx.ServiceHash = hash.HashStr(ctx.ServiceName)
		}
	}

	// normalize 이후에 Uri 변경 후 적용 UDPManager 에서 옮겨옴
	if conf.QueryStringEnabled {
		qs := MatchQueryString(normalizeServiceName, p.ServiceURL, conf.QueryStringUrls, conf.QueryStringKeys)
		if qs != "" {
			ctx.ServiceName = ctx.ServiceName + "?" + qs
			ctx.ServiceHash = hash.HashStr(ctx.ServiceName)
		}
	}

	if !ctx.IsStaticContents {
		ctx.IsStaticContents = config.IsIgnoreTrace(ctx.ServiceHash, ctx.ServiceName)
	}

	// 도메인
	ctx.HttpHost = p.ServiceURL.Host
	ctx.HttpHostHash = hash.HashStr(p.ServiceURL.Host)
	//logutil.Infoln("TraceContextMain DomainHash=", ctx.HttpHostHash, "ctx.DomainName=", ctx.HttpHost, "TraceContextMain ServiceHash=", ctx.ServiceHash, "ctx.ServiceName=", ctx.ServiceName)
	ctx.HttpMethod = strings.ToUpper(p.HttpMethod)
	data.SendHashText(pack.TEXT_HTTP_DOMAIN, ctx.HttpHostHash, ctx.HttpHost)

	ctx.StartTime = p.Time
	// 문법 : X-Forwarded-For: <client>, <proxy1>, <proxy2>
	if strings.Index(p.Ipaddr, ",") > -1 {
		ipArray := strings.Split(p.Ipaddr, ",")
		if len(ipArray) > 1 {
			p.Ipaddr = ipArray[0]
		}
	}
	ctx.RemoteIp = io.ToInt(iputil.ToBytes(p.Ipaddr), 0) // getRemoteAddr
	ctx.StartCpu = p.Cpu
	ctx.StartMalloc = p.Mem

	if conf.TraceUserAgentEnabled {
		ctx.UserAgentString = p.UAgent
		ctx.UserAgent = hash.HashStr(p.UAgent)
		data.SendHashText(pack.TEXT_USER_AGENT, ctx.UserAgent, ctx.UserAgentString)

	}
	if conf.TraceRefererEnabled {
		ctx.Referer = hash.HashStr(p.RefererURL.String())
		ctx.RefererURL = urlutil.NewURL(p.RefererURL.String())
		data.SendHashText(pack.TEXT_REFERER, ctx.Referer, p.Ref)
	}

	ctx.Txid = p.Txid
	ctx.Pid = p.Pid
	ctx.ThreadId = p.ThreadId
	ctx.ProfileSeq = p.Txid

	// WClientId 설정
	// PHP Extention 에서 보내오는 WClientId 를 설정, 만약 WClientId 값이 없으면 RemoteIp 로 설정
	if conf.TraceUserEnabled {
		if p.WClientId == "" {
			ctx.WClientId = int64(ctx.RemoteIp)
		} else {
			ctx.WClientId = int64(hash.HashStr(p.WClientId))
		}
	}

	data.SendHashText(pack.TEXT_SERVICE, ctx.ServiceHash, ctx.ServiceName)

	meter.AddMeterUsers(ctx.WClientId)

	// Tx End
	ctx.Elapsed = int32(p.Elapsed)

	ctx.EndCpu = p.Cpu
	ctx.EndMalloc = p.Mem

	// Transaction 시작 시간을 에이전트 시간으로 변경.
	ctx.EndTime = dateutil.Now()
	ctx.StartTime = ctx.EndTime - int64(ctx.Elapsed)

	// set mtid
	ctx.Mtid = p.Mtid
	ctx.Mdepth = p.Mdepth
	ctx.McallerTxid = p.McallerTxid
	ctx.McallerPcode = p.McallerPcode
	ctx.McallerSpec = p.McallerSpec
	ctx.McallerUrl = p.McallerUrl
	ctx.McallerUrlHash = p.McallerUrlHash

	poid := strings.Split(p.McallerPoidKey, ",")
	if len(poid) > 0 {
		ctx.McallerPcode = hexa32.ToLong32(strings.TrimSpace(poid[0]))
	}
	if len(poid) > 1 {
		ctx.McallerOkind = int32(hexa32.ToLong32(strings.TrimSpace(poid[1])))
	}
	if len(poid) > 2 {
		ctx.McallerOid = int32(hexa32.ToLong32(strings.TrimSpace(poid[2])))
	}

	//logutil.Infoln("EndTX POID KEY", "key=", p.McallerPoidKey)
	//logutil.Infoln("EndTX POID KEY", "McallerPcode=", ctx.McallerPcode, ",McallerOkind=", ctx.McallerOkind, ", McallerOid=", ctx.McallerOid)

	// TraceMain 에서 옮겨옴. - Hitmap, tps, 등의 통계를 더 빠르게 처리.
	//meter.GetInstanceMeterService().Add(ctx.ServiceHash, ctx.Elapsed, ctx.Error != 0, ctx.ErrorLevel, ctx.McallerPcode, ctx.McallerOkind, ctx.McallerOid)
	tx := service.NewTxRecord()
	tx.Txid = ctx.ProfileSeq
	// TraceContextManager에서 구한 현재시간보다 더 느려질 가능성 있음. elpased 오차 발생
	// tx.EndTime = dateutil.Now()
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
		tx.CpuTime = int32(ctx.EndCpu)
	}

	if ctx.EndMalloc < 0 {
		tx.Malloc = -1
	} else {
		tx.Malloc = ctx.EndMalloc
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

	tx.Fields = ctx.ExtraFields()

	// ctx를 보내고 싶지만, import cycle 오류 발생.
	meter.GetInstanceMeterService().Add(tx, ctx.McallerPcode, ctx.McallerOkind, ctx.McallerOid)

	// DEBUG Queue
	if conf.QueueProfileEnabled == false {
		sendTransactionQue <- ctx
	} else {
		if profileQueue != nil {
			profileQueue.PutForce(ctx)
		}
	}
}

var dbc int32 = hash.HashStr("php")

// SQL End
func profileSql(p *udp.UdpTxSqlPack) {
	conf := config.GetConfig()
	st := step.NewSqlStepX()
	st.Dbc = hash.HashStr(p.Dbc)
	st.Elapsed = p.Elapsed

	// DEBUG 일단 제외
	// TODO Configure 설정 추가
	//if (st.Elapsed > conf.Profile_error_sql_time_max) {
	//	errHash := StatError.getInstance().addError(SLOW_SQL.o, SLOW_SQL.o.getMessage(), ctx.service_hash, ctx.profile, TextTypes.SQL, step.hash)
	//	if (ctx.Error == 0) {
	//		ctx.Error = hash
	//		}
	//	st.Error = hash
	//	}
	//var crud byte
	psql := EscapeLiteral(p.Sql)
	if psql == nil {
		st.Hash = hash.HashStr(p.Sql)
		psql = NewParsedSql('*', st.Hash, "")
		// SqlStep_3
		//crud = ' '
	} else {
		st.Hash = psql.Sql
		// SqlStep_3
		//crud = psql.Type
	}

	st.Xtype = step.SQL_XTYPE_METHOD_QUERY
	if psql != nil {
		switch psql.Type {
		case 'S':
			st.Xtype = step.SQL_XTYPE_METHOD_QUERY
		case 'U':
			st.Xtype = step.SQL_XTYPE_METHOD_UPDATE
		default:
			st.Xtype = step.SQL_XTYPE_METHOD_EXECUTE
		}
	}

	ctx := GetContext(p.Txid)
	if ctx == nil {
		// 통계만 추가
		if p.ErrorType != "" {
			stat.GetInstanceStatSql().AddSqlTime(0, st.Dbc, st.Hash, st.Elapsed, true)

			thr := stat.NewErrorThrowable()
			thr.ErrorClassName = p.ErrorType
			thr.ErrorMessage = p.ErrorMessage
			thr.ErrorStack = stackToArray(p.Stack)

			stat.GetInstanceStatError().AddError(thr, thr.ErrorMessage, 0, false, nil, 0, 0)
		} else {
			stat.GetInstanceStatSql().AddSqlTime(0, st.Dbc, st.Hash, st.Elapsed, false)
		}
		return
	}

	st.StartTime = int32(p.Time - ctx.StartTime)

	// SQL Param Encrypt 추가.
	if conf.ProfileSqlParamEnabled && psql != nil {
		// SqlStep_3
		//st.SetTrue(1)
		crc := ref.NewBYTE()
		//logutil.Println("Before Encrypt ", psql.Param)
		st.P1 = toParamBytes(psql.Param, crc)
		//logutil.Println("Encrypt ", string(st.P1), ", Crc=", crc.Value , st.P2 )
		st.Pcrc = crc.Value
	}

	if conf.ProfileSqlResourceEnabled {
		// SqlStep_3
		//		st.SetTrue(2)
		//		st.Cpu = int32(p.Cpu)
		//		st.Mem = int32(p.Mem)
		st.StartCpu = int32(p.Cpu - ctx.StartCpu)
		st.StartMem = int64(p.Mem - ctx.StartMalloc)
	}

	if ctx.ErrorStep {
		st.Error = ctx.Error
		ctx.ErrorStep = false
	}

	if p.ErrorType != "" {
		thr := stat.NewErrorThrowable()
		thr.ErrorClassName = p.ErrorType
		thr.ErrorMessage = p.ErrorMessage
		thr.ErrorStack = stackToArray(p.Stack)

		profileErrorStep(thr, ctx)
		st.Error = ctx.Error
		st.Stack = stackToArray(p.Stack)
	}

	// TODO
	//	st.Xtype = (byte) (st.Xtype | xtype);
	ctx.ExecutedSqlhash = ctx.ActiveSqlhash
	ctx.ActiveSqlhash = 0
	ctx.ActiveDbc = 0

	ctx.SqlCount++
	ctx.SqlTime += st.Elapsed

	//ctx.Active_sqlhash = st.Hash;
	//ctx.Active_dbc = st.Dbc;

	// TODO
	//ctx.Active_crud =	psql.Type
	// TODO
	//	if conf.Profile_sql_param_enabled && psql != null {
	//		switch (psql.Type) {
	//			case 'S': fallthrough
	//			case 'D': fallthrough
	//			case 'U':
	//				BYTE crc = new BYTE();
	//				st.setTrue(1);
	//				st.p1 = toParamBytes(psql.param, crc);
	//				st.pcrc = crc.value;
	//
	//		}
	//	}
	// st.Crud;

	meter.GetInstanceMeterSQL().Add(st.Dbc, st.Elapsed, (st.Error != 0))
	// import cycle 내부 addSqlTime으로 변경, stat.GetInstanceStatSql().AddSqlTime  -> addSqlTime
	stat.GetInstanceStatSql().AddSqlTime(ctx.ServiceHash, st.Dbc, st.Hash, st.Elapsed, st.Error != 0)

	if st.Error == 0 {
		ctx.Profile.Add(st)
	} else {
		ctx.Profile.AddHeavy(st)
	}

	data.SendHashText(pack.TEXT_DB_URL, st.Dbc, p.Dbc)

	//log.Printf("TraceContextManager:ProfileSql: END hsah=%d, DBC %d, %s", st.Hash, st.Dbc, p.Dbc)
}

// SQL End
func profileSqlParam(p *udp.UdpTxSqlParamPack) {
	conf := config.GetConfig()
	st := step.NewSqlStepX()
	st.Dbc = hash.HashStr(p.Dbc)
	st.Elapsed = p.Elapsed
	st.Hash = hash.HashStr(p.Sql)
	data.SendHashText(pack.TEXT_SQL, st.Hash, p.Sql)

	psql := EscapeLiteral(p.Sql)
	if psql == nil {
		st.Hash = hash.HashStr(p.Sql)
		psql = NewParsedSql('*', st.Hash, "")
	} else {
		st.Hash = psql.Sql
	}

	st.Xtype = step.SQL_XTYPE_METHOD_QUERY
	if psql != nil {
		switch psql.Type {
		case 'S':
			st.Xtype = step.SQL_XTYPE_METHOD_QUERY
		case 'U':
			st.Xtype = step.SQL_XTYPE_METHOD_UPDATE
		default:
			st.Xtype = step.SQL_XTYPE_METHOD_EXECUTE
		}
	}

	ctx := GetContext(p.Txid)
	if ctx == nil {
		// 통계만 추가
		if p.ErrorType != "" {
			stat.GetInstanceStatSql().AddSqlTime(0, st.Dbc, st.Hash, st.Elapsed, true)

			thr := stat.NewErrorThrowable()
			thr.ErrorClassName = p.ErrorType
			thr.ErrorMessage = p.ErrorMessage
			thr.ErrorStack = stackToArray(p.Stack)

			stat.GetInstanceStatError().AddError(thr, thr.ErrorMessage, 0, false, nil, 0, 0)
		} else {
			stat.GetInstanceStatSql().AddSqlTime(0, st.Dbc, st.Hash, st.Elapsed, false)
		}
		return
	}

	st.StartTime = int32(p.Time - ctx.StartTime)

	// SQL Param Encrypt 추가.
	if conf.ProfileSqlParamEnabled && psql != nil {
		//st.SetTrue(1)
		crc := ref.NewBYTE()
		st.P1 = toParamBytes(psql.Param, crc)
		// bind
		if p.Param != "" {
			st.P2 = toParamBytes(p.Param, crc)
			//logutil.Println("Encrypt ", string(st.P2), ", Crc=", crc.Value, p.Param)
		}
		st.Pcrc = crc.Value
	}

	if conf.ProfileSqlResourceEnabled {
		// SqlStep_3
		//		st.SetTrue(2)
		//		st.Cpu = int32(p.Cpu)
		//		st.Mem = int32(p.Mem)
		st.StartCpu = int32(p.Cpu - ctx.StartCpu)
		st.StartMem = int64(p.Mem - ctx.StartMalloc)
	}

	if ctx.ErrorStep {
		st.Error = ctx.Error
		ctx.ErrorStep = false
	}

	if p.ErrorType != "" {
		thr := stat.NewErrorThrowable()
		thr.ErrorClassName = p.ErrorType
		thr.ErrorMessage = p.ErrorMessage
		thr.ErrorStack = stackToArray(p.Stack)

		profileErrorStep(thr, ctx)
		st.Error = ctx.Error
		st.Stack = stackToArray(p.Stack)
	}

	// TODO
	//	st.Xtype = (byte) (st.Xtype | xtype);
	ctx.ExecutedSqlhash = ctx.ActiveSqlhash
	ctx.ActiveSqlhash = 0
	ctx.ActiveDbc = 0

	ctx.SqlCount++
	ctx.SqlTime += st.Elapsed

	//ctx.Active_sqlhash = st.Hash;
	//ctx.Active_dbc = st.Dbc;

	// TODO
	//ctx.Active_crud =	psql.Type
	// TODO
	//	if conf.Profile_sql_param_enabled && psql != null {
	//		switch (psql.Type) {
	//			case 'S': fallthrough
	//			case 'D': fallthrough
	//			case 'U':
	//				BYTE crc = new BYTE();
	//				st.setTrue(1);
	//				st.p1 = toParamBytes(psql.param, crc);
	//				st.pcrc = crc.value;
	//
	//		}
	//	}
	// st.Crud;

	meter.GetInstanceMeterSQL().Add(st.Dbc, st.Elapsed, (st.Error != 0))
	// import cycle 내부 addSqlTime으로 변경, stat.GetInstanceStatSql().AddSqlTime  -> addSqlTime
	stat.GetInstanceStatSql().AddSqlTime(ctx.ServiceHash, st.Dbc, st.Hash, st.Elapsed, st.Error != 0)

	if st.Error == 0 {
		ctx.Profile.Add(st)
	} else {
		ctx.Profile.AddHeavy(st)
	}

	data.SendHashText(pack.TEXT_DB_URL, st.Dbc, p.Dbc)

	//log.Printf("TraceContextManager:ProfileSqlParam: END hsah=%d, DBC %d, %s", st.Hash, st.Dbc, p.Dbc)
}

func profileHttpc(p *udp.UdpTxHttpcPack) {
	conf := config.GetConfig()
	st := step.NewHttpcStepX()

	//st.Url = hash.HashStr(p.HttpcURL.Path)
	// Nomalize
	// Nomalize boot.ServiceURLPatternDetector
	nUrl := GetInstanceURLPatternDetector().Normalize(p.HttpcURL.Path)
	st.Url = hash.HashStr(nUrl)
	st.Host = hash.HashStr(p.HttpcURL.Host)
	st.Port = int32(p.HttpcURL.Port)
	st.Elapsed = p.Elapsed
	st.StepId = p.StepId

	// ctx interface 변환 전에 먼저 nil 체크, 기존 panic 보완
	ctx := GetContext(p.Txid)
	if ctx == nil {
		// 통계만 추가
		if p.ErrorType != "" {
			stat.GetInstanceStatHttpc().AddHttpcTime(0, st.Url, st.Host, st.Port, st.Elapsed, true)

			thr := stat.NewErrorThrowable()
			thr.ErrorClassName = p.ErrorType
			thr.ErrorMessage = p.ErrorMessage
			thr.ErrorStack = stackToArray(p.Stack)

			stat.GetInstanceStatError().AddError(thr, thr.ErrorMessage, 0, false, nil, 0, 0)
		} else {
			stat.GetInstanceStatHttpc().AddHttpcTime(0, st.Url, st.Host, st.Port, st.Elapsed, false)
		}
		return
	}

	st.StartTime = int32(p.Time - ctx.StartTime)

	if conf.ProfileHttpcResourceEnabled {
		st.StartCpu = int32(p.Cpu - ctx.StartCpu)
		st.StartMem = int64(p.Mem - ctx.StartMalloc)
	}

	if ctx.ErrorStep {
		st.Error = ctx.Error
		ctx.ErrorStep = false
	}

	if p.ErrorType != "" {
		thr := stat.NewErrorThrowable()
		thr.ErrorClassName = p.ErrorType
		thr.ErrorMessage = p.ErrorMessage
		thr.ErrorStack = stackToArray(p.Stack)
		profileErrorStep(thr, ctx)
		st.Error = ctx.Error
		st.Stack = stackToArray(p.Stack)
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
	data.SendHashText(pack.TEXT_HTTPC_URL, st.Url, nUrl)
	data.SendHashText(pack.TEXT_HTTPC_HOST, st.Host, p.HttpcURL.Host)
}

func profileErrorStep(thr *stat.ErrorThrowable, ctx *TraceContext) {
	if IsIgnoreException(thr) {
		return
	}

	// 임재환 추가 java Thread thr 대신 ErrorThrowable 구조체 사용
	ctx.Thr = thr
	if IsBizException(thr) {
		//ctx.Error = stat.GetInstanceStatError().AddErrorHashOnly(ctx.Thr, ctx.Thr.ErrorMessage)
		ctx.Error = stat.GetInstanceStatError().AddError(ctx.Thr, ctx.Thr.ErrorMessage, ctx.ServiceHash, conf.ErrorSnapEnabled, ctx.Profile.GetStep4Error(), 0, 0)
		// BixException 통계 제외 처리를 위해
		ctx.ErrorLevel = pack.INFO //-> metering 에서 에러 카운트 안되게 해야 함.
	} else {
		ctx.Error = stat.GetInstanceStatError().AddError(ctx.Thr, ctx.Thr.ErrorMessage, ctx.ServiceHash, conf.ErrorSnapEnabled, ctx.Profile.GetStep4Error(), 0, 0)
		// BixException 통계 제외 처리를 위해
		ctx.ErrorLevel = pack.WARNING
	}
}

func profileError(p *udp.UdpTxErrorPack) {

	thr := stat.NewErrorThrowable()
	thr.ErrorClassName = p.ErrorType
	thr.ErrorMessage = p.ErrorMessage
	thr.ErrorStack = stackToArray(p.Stack)

	if IsIgnoreException(thr) {
		return
	}

	// ctx interface 변환 전에 먼저 nil 체크, 기존 panic 보완
	ctx := GetContext(p.Txid)
	if ctx == nil {
		// 통계만 추가
		stat.GetInstanceStatError().AddError(thr, thr.ErrorMessage, 0, false, nil, 0, 0)
		return
	}
	//logutil.Println("profileError:", thr)
	conf := config.GetConfig()

	// Step Error 처리는 PHP 는 제외
	if conf.AppType == lang.APP_TYPE_PHP || conf.AppType == lang.APP_TYPE_BSM_PHP ||
		conf.AppType == lang.APP_TYPE_GO || conf.AppType == lang.APP_TYPE_BSM_GO {
		// DEBUG PHP 는 메세지 스텝으로 출력
		st := step.NewMessageStep()

		st.StartTime = int32(p.Time - ctx.StartTime)
		st.Time = p.Elapsed
		st.Hash = int32(hash.HashStr(thr.ErrorClassName))
		st.Value = 0

		data.SendHashText(pack.TEXT_MESSAGE, st.Hash, thr.ErrorClassName)

		// Error를 Message로 출력할 때 Message Desc 는 해시 처리 되지 않고 텍스트로 전달 되기 때문에 길이를 잘라서 설정.
		msg := stringutil.TrimEmpty(thr.ErrorMessage)
		msg = stringutil.Truncate(msg, 400)

		st.Desc = msg

		//ctx.Profile.Add(st)
		// Error 는 무조건 step 삽입
		ctx.Profile.AddHeavy(st)

	} else {
		ctx.ErrorStep = true
	}

	// 임재환 추가 java Thread thr 대신 ErrorThrowable 구조체 사용
	ctx.Thr = thr
	if IsBizException(thr) {
		//ctx.Error = stat.GetInstanceStatError().AddErrorHashOnly(ctx.Thr, ctx.Thr.ErrorMessage)
		ctx.Error = stat.GetInstanceStatError().AddError(ctx.Thr, ctx.Thr.ErrorMessage, ctx.ServiceHash, conf.ErrorSnapEnabled, ctx.Profile.GetStep4Error(), 0, 0)
		// BixException 통계 제외 처리를 위해
		ctx.ErrorLevel = pack.INFO //-> metering 에서 에러 카운트 안되게 해야 함.
	} else {
		ctx.Error = stat.GetInstanceStatError().AddError(ctx.Thr, ctx.Thr.ErrorMessage, ctx.ServiceHash, conf.ErrorSnapEnabled, ctx.Profile.GetStep4Error(), 0, 0)
		// BixException 통계 제외 처리를 위해
		ctx.ErrorLevel = pack.WARNING
	}
}

func profileMsg(p *udp.UdpTxMessagePack) {
	// ctx interface 변환 전에 먼저 nil 체크, 기존 panic 보완
	ctx := GetContext(p.Txid)
	if ctx == nil {
		return
	}

	st := step.NewMessageStep()

	st.StartTime = int32(p.Time - ctx.StartTime)
	st.Time = p.Elapsed
	st.Hash = int32(hash.HashStr(p.Hash))
	if p.Value != "" {
		if value, err := strconv.Atoi(p.Value); err == nil {
			st.Value = int32(value)
		}
	}

	data.SendHashText(pack.TEXT_MESSAGE, st.Hash, p.Hash)

	var buf bytes.Buffer
	if p.Hash == "HTTP-HEADERS" {
		arr := strings.Split(p.Desc, "\n")
		for _, it := range arr {
			k, v := stringutil.ToPair(it, "=")
			if strings.HasPrefix(k, "HTTP_") {
				k = k[5:]
			}
			k = strings.ReplaceAll(strings.ToLower(k), "-", "_")
			if !conf.ProfileHttpHeaderIgnoreKeys.HasKey(k) {
				buf.WriteString(k + "=" + v + "\n")
			} else {
				buf.WriteString(k + "=#\n")
			}
		}
		st.Desc = buf.String()
	} else {
		st.Desc = p.Desc
	}

	ctx.Profile.Add(st)
}

func profileSecureMsg(p *udp.UdpTxSecureMessagePack) {
	// ctx interface 변환 전에 먼저 nil 체크, 기존 panic 보완
	ctx := GetContext(p.Txid)
	if ctx == nil {
		return
	}

	conf := config.GetConfig()
	if conf.ProfileHttpParameterEnabled && strings.HasPrefix(ctx.ServiceName, conf.ProfileHttpParameterUrlPrefix) {
		st := step.NewSecureMsgStep()

		st.StartTime = int32(p.Time - ctx.StartTime)
		st.Hash = int32(hash.HashStr(p.Hash))
		sb := stringutil.NewStringBuffer()
		sb.Append(p.Desc)
		// append get parameter from url query_string
		if p.Hash == "GET Parameter" {
			m := ctx.ServiceURL.ParseQuery()
			for k, v := range m {
				if strings.Index(p.Desc, k) > -1 {
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
		st.Value = toParamBytes(sb.ToString(), crc)
		st.Crc = crc.Value
		//logutil.Infoln(p.Desc)

		// DataTextAgent.MESSAGE.add(hash, "HTTP-PARAMETERS")
		data.SendHashText(pack.TEXT_MESSAGE, st.Hash, p.Hash)

		ctx.Profile.Add(st)
	}

}

func profileMethod(p *udp.UdpTxMethodPack) {
	// ctx interface 변환 전에 먼저 nil 체크, 기존 panic 보완
	ctx := GetContext(p.Txid)
	if ctx == nil {
		return
	}

	conf := config.GetConfig()
	st := step.NewMethodStepX()

	if !strings.HasSuffix(p.Method, ")") {
		p.Method = p.Method + "()"
	}

	st.Hash = hash.HashStr(p.Method)
	data.SendHashText(pack.TEXT_METHOD, st.Hash, p.Method)

	st.StartTime = int32(p.Time - ctx.StartTime)
	st.Elapsed = p.Elapsed

	if conf.ProfileMethodResourceEnabled {
		st.SetTrue(1)
		st.StartCpu = int32(p.Cpu - ctx.StartCpu)
		st.StartMem = int32(p.Mem - ctx.StartMalloc)
	}

	if p.Stack != "" {
		st.SetTrue(2)
		st.Stack = stackToArray(p.Stack)
	}
	ctx.Profile.Add(st)
}

func profileDBC(p *udp.UdpTxDbcPack) {
	// ctx interface 변환 전에 먼저 nil 체크, 기존 panic 보완
	ctx := GetContext(p.Txid)
	if ctx == nil {
		return
	}

	st := step.NewDBCStep()
	st.Hash = hash.HashStr(p.Dbc)
	data.SendHashText(pack.TEXT_DB_URL, st.Hash, p.Dbc)
	// JAVA 에서 DBCStep 은 TEXT 를 METHOD 로 가져옴.   agent.trace.JdbcUrls -> DataTextAgent.addMethod(dbc_hash, url);
	data.SendHashText(pack.TEXT_METHOD, st.Hash, p.Dbc)

	st.StartTime = int32(p.Time - ctx.StartTime)
	st.Elapsed = p.Elapsed

	if p.ErrorType != "" {
		thr := stat.NewErrorThrowable()
		thr.ErrorClassName = p.ErrorType
		thr.ErrorMessage = p.ErrorMessage
		thr.ErrorStack = stackToArray(p.Stack)
		profileErrorStep(thr, ctx)

		msg := stringutil.TrimEmpty(p.ErrorMessage)
		msg = stringutil.Truncate(msg, 200)
		msgHash := hash.HashStr(msg)
		data.SendHashText(pack.TEXT_ERROR, msgHash, msg)
		st.Error = msgHash
	}

	//log.Println("DBC", st.Hash, "," , p.Dbc, "\n,"\n,step=", st)
	ctx.DbcTime += st.Elapsed
	if st.Error == 0 {
		ctx.Profile.Add(st)
	} else {
		ctx.Profile.AddHeavy(st)
	}
}

// Deprecated: Use stat.GetInstanceStatSql.AddSqlTime instead
func addSqlTime(ctx *TraceContext, serviceHash, dbc int32, crud byte, hash, elapsed int32, isErr bool) {

	//		import cyle error를 피하기 위해서 TraceContextManager로 이동 (trace 패키지)
	urlRec := stat.GetInstanceStatTranx().GetService(serviceHash)
	//fmt.Println("addSqlTime urlRec=", urlRec)

	if urlRec != nil {
		switch crud {
		case 'S':
			ctx.SqlSelect++
		case 'U':
			ctx.SqlUpdate++
		case 'D':
			ctx.SqlDelete++
		case 'I':
			ctx.SqlInsert++
		default:
			ctx.SqlOthers++
		}
	}

	// ctx는 pack.ServiceRec으로 변경
	//stat.GetInstanceStatSql().AddSqlTime(ctx, ctx.ServiceHash, st.Dbc, st.Crud, st.Hash, st.Elapsed, st.Error != 0)
	//stat.GetInstanceStatSql().AddSqlTime(urlRec, serviceHash, dbc, crud, hash, elapsed, isErr)
	stat.GetInstanceStatSql().AddSqlTime(serviceHash, dbc, hash, elapsed, isErr)
}

func GetContextEnumeration() hmap.Enumeration {
	//fmt.Println("GetContextEnumeration size=", ctxTable.Size())
	return ctxTable.Values()
}

func GetContext(key int64) *TraceContext {
	tc := ctxTable.Get(key)
	if tc == nil {
		return nil
	}
	return tc.(*TraceContext)
}
func PutContext(key int64, v interface{}) interface{} {
	return ctxTable.Put(key, v)
}

// counter.TaskActiveTranCount 에서 종료 시간이 지난 cts 를 삭제 할 때 호출
func RemoveContext(key int64) interface{} {
	//fmt.Println("ctx Remove=", key)
	return ctxTable.Remove(key)
}

func ContainsTxid(txid int64) bool {
	return ctxTable.ContainsKey(txid)
}

// counter.TaskActiveTranCount 에서 종료 시간이 지난 ctx 를 삭제 할 때 호출
// 정상 적인 종료 처리 진행.
func RemoveLostContext(key int64) {
	//fmt.Println("ctx Remove=", key)
	//ctx := GetContext(key)
	//logutil.Infoln("RemoveLostContext ", "Txid=", key)

	// Msg 처리
	tx := udp.NewUdpTxMessagePack()
	tx.Time = dateutil.SystemNow()
	tx.Txid = key
	tx.Hash = "Lost Connection"
	tx.Value = "0"
	tx.Desc = ""
	//tx.Content = ""
	tx.Process()
	TxPut(tx)

	// End 처리
	txEnd := udp.NewUdpTxEndPack()
	txEnd.Time = dateutil.SystemNow()
	txEnd.Txid = key
	txEnd.Uri = ""
	// Lost Connection 에서 CPU, Memory 음수 값이 나오는 걸 처리
	txEnd.Cpu = -1
	txEnd.Mem = -1
	txEnd.Process()

	TxPut(txEnd)
}

func ToParamBytes(p string, crc *ref.BYTE) []byte {
	return toParamBytes(p, crc)
}
func toParamBytes(p string, crc *ref.BYTE) []byte {
	if p == "" || len(p) == 0 {
		return nil
	}
	b := []byte(p)
	return secure.GetParamSecurity().Encrypt(b, crc)
}
func StackToArray(s string) []int32 {
	return stackToArray(s)
}
func stackToArray(s string) []int32 {
	se := stringutil.Tokenizer(s, "\n")
	conf := config.GetConfig()
	max := math.Min(float64(len(se)), float64(conf.TraceErrorCallstackDepth))
	stack := make([]int32, int32(max))
	if conf.AppType == lang.APP_TYPE_PHP || conf.AppType == lang.APP_TYPE_BSM_PHP {
		// PHP 순방향
		for i := 0; i < int(max); i++ {
			stack[i] = hash.HashStr(se[i])
			data.SendHashText(pack.TEXT_STACK_ELEMENTS, stack[i], se[i])
		}
	} else {
		for i := 0; i < int(max); i++ {
			stack[i] = hash.HashStr(se[int(max)-i-1])
			data.SendHashText(pack.TEXT_STACK_ELEMENTS, stack[i], se[int(max)-i-1])
		}
	}
	return stack
}

func AddMessage(ctx *TraceContext, t int32, et int32, title, v, desc string, cut int, isHeavy bool) {
	addMessage(ctx, t, et, title, v, desc, cut, isHeavy)
}

func addMessage(ctx *TraceContext, t int32, et int32, title, v, desc string, cut int, isHeavy bool) {
	// DEBUG PHP 는 메세지 스텝으로 출력
	st := step.NewMessageStep()

	st.StartTime = t
	st.Time = et
	st.Hash = int32(hash.HashStr(title))
	st.Value = 0
	if v != "" {
		if value, err := strconv.Atoi(v); err == nil {
			st.Value = int32(value)
		}
	}

	data.SendHashText(pack.TEXT_MESSAGE, st.Hash, title)

	if cut > 0 {
		// Error를 Message로 출력할 때 Message Desc 는 해시 처리 되지 않고 텍스트로 전달 되기 때문에 길이를 잘라서 설정.
		msg := stringutil.TrimEmpty(desc)
		msg = stringutil.Truncate(msg, cut)
		st.Desc = msg
	} else {
		st.Desc = desc
	}
	if isHeavy {
		ctx.Profile.AddHeavy(st)
	} else {
		ctx.Profile.Add(st)
	}
}

func MatchQueryString(serviceName string, serviceURL *urlutil.URL, matchUrls []string, matchKeys []string) string {
	params := serviceURL.ParseQuery()
	path := serviceURL.Path
	sb := stringutil.NewStringBuffer()
	if stringutil.InArray(serviceName, matchUrls) || stringutil.InArray(path, matchUrls) {
		keyCount := 0
		// key 설정이 안되있는 경우 전체 붙이기
		if len(matchKeys) == 0 {
			return serviceURL.Query
		}
		for _, key := range matchKeys {
			for k, v := range params {
				if k == key {
					if keyCount > 0 {
						sb.Append("&")
					}
					sb.Append(k)
					if len(v) > 1 {
						sb.Append("=ARRAY")
					} else {
						sb.Append("=").Append(v[0])
					}
					keyCount++
					break
				}
			}
		}
	}
	return sb.ToString()
}
func SendTransaction(ctx *TraceContext) {
	// DEBUG Queue
	if conf.QueueProfileEnabled == false {
		sendTransactionQue <- ctx
	} else {
		if profileQueue != nil {
			profileQueue.PutForce(ctx)
		}
	}
}
