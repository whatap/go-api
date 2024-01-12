package trace

import (
	//"fmt"
	//"time"
	//"log"
	//"runtime"
	"sync"

	//"runtime/debug"
	"time"

	//	"github.com/whatap/go-api/agent/agent/counter/meter"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/agent/secure"
	"github.com/whatap/go-api/agent/agent/stat"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/service"
	"github.com/whatap/golib/util/bitutil"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/queue"
	"github.com/whatap/golib/util/stringutil"
)

// SQL 일반화 ParsedSQL (Java TraceSQL) 의 해시를 reset
// Java는 기존에 DataText 에서 모두 reset하지만 import circle 오류로 새로 만듬.
// TODO hash 관련 Text reset 을 통합할 필요 있음.
var lastDate int64
var textReset int32

// data.DataProfileAgent 를  옮겨옴 import cycle 오류
var LastReject int64

var sendTransactionQue chan *TraceContext
var traceMainLock sync.Mutex
var profileQueue *queue.RequestQueue

func StartProfileSender() {
	conf := config.GetConfig()
	// DEBUG Queue
	if conf.QueueProfileEnabled == false {
		if sendTransactionQue == nil {
			sendTransactionQue = make(chan *TraceContext, int(conf.QueueProfileSize))
		}
		if conf.QueueLogEnabled {
			logutil.Println("WA550-00", "Profile channel size=", cap(sendTransactionQue), ",conf.size=", conf.QueueProfileSize)
		}
	} else {
		if profileQueue == nil {
			profileQueue = queue.NewRequestQueue(int(conf.QueueProfileSize))
			profileQueue.Overflowed = func(o interface{}) {
				if conf.QueueLogEnabled {
					logutil.Println("WA550-01", "Profile Queue overflowed")
				}
			}
		}
		if conf.QueueLogEnabled {
			logutil.Println("WA550-02", "Profile Queue=", profileQueue.GetCapacity())
		}
	}
	if conf.QueueLogEnabled {
		logutil.Println("WA550-03", "Profile Queue thread count=", conf.QueueProfileProcessThreadCount)
	}
	for i := 0; i < int(conf.QueueProfileProcessThreadCount); i++ {
		go func() {
			for {
				process()
			}
		}()
	}

	// ParsedSql reset
	//logutil.Println("ParsedSql reset ")
	// TODO 나중에 따로 통합
	lastDate = getDate()
	textReset = config.GetConfig().TextReset
	go func() {
		for {
			// DEBUG goroutine 로그
			//logutil.Println("SQL reset ")
			resetTraceSQL()
			time.Sleep(1000 * time.Millisecond)
		}
	}()
}

func process() {
	traceMainLock.Lock()
	defer func() {
		traceMainLock.Unlock()
		if r := recover(); r != nil {
			logutil.Println("WA551", " Recover ", r) //, string(debug.Stack()))
			_, ok := r.(error)
			if !ok {
				logutil.Println("WA551", "pkg: ", r) //, string(debug.Stack()))
			}
		}
	}()

	var ctx *TraceContext

	// DEBUG Queue
	if conf.QueueProfileEnabled == false {
		if conf.QueueLogEnabled {
			logutil.Println("WA551-00", "Profile channel len=", len(sendTransactionQue))
		}
		if len(sendTransactionQue) == cap(sendTransactionQue) {
			logutil.Println("W551-01", "Profile Channle Full", len(sendTransactionQue))
		}
		ctx = <-sendTransactionQue
	} else {
		if conf.QueueLogEnabled {
			logutil.Println("WA551-02", "Profile Queue len=", profileQueue.Size())
		}
		if profileQueue.Size() == profileQueue.GetCapacity() {
			logutil.Println("W551-03", "Profile Queue Full", profileQueue.Size())
		}
		v := profileQueue.Get()
		if v == nil {
			return
		}
		ctx = v.(*TraceContext)
	}

	if ctx.IsStaticContents {
		//logutil.Infoln("Ignore", "IsStaticContents Resurn")
		return
	}

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
	tx.McallerStepId = ctx.McallerStepId

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

	// 2021.06.28 StatTx 관련 apdex 추가
	err := (tx.ErrorLevel >= pack.WARNING)
	if !err {
		if tx.Elapsed <= conf.ApdexTime {
			tx.Apdex = 2
		} else if tx.Elapsed <= conf.ApdexTime4T {
			tx.Apdex = 1
		}
	}
	//logutil.Infoln(">>>>", "elapsed=", tx.Elapsed, ",err=", err, ",apdex=", tx.Apdex)

	/// Profile Send ///
	profile := pack.NewProfilePack()
	profile.Transaction = tx
	profile.Time = tx.EndTime

	//meter.AddTransaction(tx.Service, tx.Elapsed, tx.Error != 0)
	//meter.AddTransaction(tx.Service, tx.Elapsed, tx.Error != 0, ctx.McallerPcode, ctx.McallerOkind, ctx.McallerOid)

	// TraceContextManager endTx 로 이동
	//meter.GetInstanceMeterService().Add(tx.Service, tx.Elapsed, tx.Error != 0, tx.ErrorLevel, ctx.McallerPcode, ctx.McallerOkind, ctx.McallerOid)

	//profile.SetProfile(ctx.Profile.GetSteps())
	SendProfile(ctx, profile, false)

	//ctx Close. sync.Pool
	CloseTraceContext(ctx)
}

// func (this *DataProfileAgent) SendProfile(ctx *trace.TraceContext, profile *pack.ProfilePack, rejected bool) {
// data.DataProfileAgent 를  옮겨옴 import cycle 오류
func SendProfile(ctx *TraceContext, profile *pack.ProfilePack, rejected bool) {
	conf := config.GetConfig()

	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA552", " Recover ", r) //, string(debug.Stack()))
		}
	}()

	stat.GetInstanceStatRemoteIp().IncRemoteIp(ctx.RemoteIp)
	stat.GetInstanceStatUserAgent().IncUserAgent(ctx.UserAgent)

	// DEBUG TxRecord
	//transaction := profile.Transaction.(*service.WasService)
	transaction := profile.Transaction

	// DEBUG Login
	if ctx.Login != "" {
		transaction.Login = hash.HashStr(ctx.Login)
		data.SendHashText(pack.TEXT_LOGIN, transaction.Login, ctx.Login)
		if conf.StatLoginEnabled {
			tc := stat.GetInstanceStatTranxLogin().GetService(transaction.Login, ctx.ServiceHash)
			if tc != nil {
				tc.Count++
				if ctx.Error != 0 {
					tc.Error++
				}
				tc.Time += int64(transaction.Elapsed)
			}
		}
	}

	// stat domain
	if conf.StatDomainEnabled && ctx.HttpHostHash != 0 {
		tc := stat.GetInstanceStatTranxDomain().GetService(ctx.HttpHostHash, ctx.ServiceHash)
		if tc != nil {
			tc.Count++
			if ctx.Error != 0 {
				tc.Error++
			}
			tc.Time += int64(transaction.Elapsed)
		}
	}

	// stat referer
	if conf.StatRefererEnabled && ctx.Referer != 0 {
		var refererHash int32
		var tc *pack.TimeCount
		switch conf.StatRefererFormat {
		case config.REFERER_FORMAT_DOMAIN:
			refererHash = hash.HashStr(ctx.RefererURL.Domain())
			data.SendHashText(pack.TEXT_REFERER, refererHash, ctx.RefererURL.Domain())
			tc = stat.GetInstanceStatTranxReferer().GetService(refererHash, ctx.ServiceHash)
			break
		case config.REFERER_FORMAT_DOMAIN_PATH:
			refererHash = hash.HashStr(ctx.RefererURL.DomainPath())
			data.SendHashText(pack.TEXT_REFERER, refererHash, ctx.RefererURL.DomainPath())
			tc = stat.GetInstanceStatTranxReferer().GetService(refererHash, ctx.ServiceHash)
			break
		case config.REFERER_FORMAT_PATH:
			refererHash = hash.HashStr(ctx.RefererURL.Path)
			data.SendHashText(pack.TEXT_REFERER, refererHash, ctx.RefererURL.Path)
			tc = stat.GetInstanceStatTranxReferer().GetService(refererHash, ctx.ServiceHash)
			break
		default:
			tc = stat.GetInstanceStatTranxReferer().GetService(ctx.Referer, ctx.ServiceHash)
		}

		if tc != nil {
			tc.Count++
			if ctx.Error != 0 {
				tc.Error++
			}
			tc.Time += int64(transaction.Elapsed)
		}
	}

	// stat mtrace
	if conf.StatMtraceEnabled && ctx.McallerPcode != 0 {
		//logutil.Infof("MTrace stat enabled - %d %d %s %s", ctx.McallerTxid, ctx.McallerPcode, ctx.McallerSpec, ctx.McallerUrl)

		key := stat.NewMtraceKEY()

		key.CallerPcode = ctx.McallerPcode
		key.CallerOkind = ctx.McallerOkind
		transaction.McallerPcode = key.CallerPcode

		if ctx.McallerSpec != "" {
			key.CallerSpec = hash.HashStr(ctx.McallerSpec)
			transaction.McallerSpec = key.CallerSpec
			data.SendHashText(pack.TEXT_MTRACE_SPEC, key.CallerSpec, ctx.McallerSpec)
		}
		if ctx.McallerUrl != "" {
			key.CallerUrl = ctx.McallerUrlHash
			transaction.McallerUrl = key.CallerUrl
		}
		key.Url = ctx.ServiceHash
		tc := stat.GetInstanceStatTranxMtCaller().GetService(key)
		if tc != nil {
			tc.Count++
			if ctx.Error != 0 {
				tc.Error++
			}
			tc.Time += int64(transaction.Elapsed)
		}
	}

	// ServiceRec -> TransactionRec
	service_rec := stat.GetInstanceStatTranx().GetService(transaction.Service)

	if service_rec != nil {
		service_rec.Count++
		// JAVA NOT
		//		if transaction.errorLevel >= EventLevel.WARNING {
		//			stat.error++
		//		}

		if transaction.Error != 0 {
			service_rec.Error++
		}

		if transaction.Elapsed < 0 {
			transaction.Elapsed = 0
		}

		// 5분 이상 수행된 TX의 경우에는 값이 해석하기 어려울 수 있음
		//service_rec.Actived += ctx.ProfileActive
		service_rec.TimeSum += int64(transaction.Elapsed)
		if transaction.Elapsed > service_rec.TimeMax {
			service_rec.TimeMax = transaction.Elapsed
		}

		// 2021.06.29 StatTx apdex, timemin, timestd(표준편차) 추가
		switch transaction.Apdex {
		case 2:
			service_rec.ApdexSatisfied += 1
		case 1:
			service_rec.ApdexTolerated += 1
		}
		if service_rec.TimeMin == 0 || transaction.Elapsed < service_rec.TimeMin {
			service_rec.TimeMin = transaction.Elapsed
		}
		service_rec.TimeStd += (int64(transaction.Elapsed)) * (int64(transaction.Elapsed))

		service_rec.SqlCount += transaction.SqlCount
		//service_rec.SqlSelect += ctx.SqlSelect
		//service_rec.SqlInsert += ctx.SqlInsert
		//service_rec.SqlDelete += ctx.SqlDelete
		//service_rec.SqlUpdate += ctx.SqlUpdate
		//service_rec.SqlOthers += ctx.SqlOthers
		service_rec.SqlTime += int64(transaction.SqlTime)
		service_rec.SqlFetch += ctx.RsCount
		service_rec.SqlFetchTime += ctx.RsTime
		//service_rec.SqlCommitCount += ctx.JdbcCommit
		//service_rec.SqlUpdateRecord += ctx.JdbcUpdateRecord
		service_rec.HttpcCount += transaction.HttpcCount
		service_rec.HttpcTime += int64(transaction.HttpcTime)
		service_rec.MallocSum += transaction.Malloc
		service_rec.CpuSum += int64(transaction.CpuTime)

		// TODO: ctx.Status
		//		switch ctx.Status / 100 {
		//		case 2:
		//			service_rec.Status200++
		//		case 3:
		//			service_rec.Status300++
		//		case 4:
		//			service_rec.Status400++
		//		case 5:
		//			service_rec.Status500++
		//		}

		if rejected {
			now := time.Now().Unix() * 1000
			if now < LastReject+1000 {
				return
			}
			LastReject = now

		} else if service_rec.Profiled == true && // 이전(5분구간 내)에 프로파일이 수집된점이 있음
			ctx.ProfileActive == 0 && // 액티브 스택을 추적한적이 없음
			transaction.Elapsed < conf.ProfileBasetime && //
			transaction.Error == 0 &&
			// JAVA NOT
			//transaction.errorLevel < EventLevel.WARNING &&
			ctx.Mtid == 0 {

			//logutil.Printf("WA553"," Profile Not Send a=%d, e=%d, pb=%d, e=%d",ctx.ProfileActive, transaction.Elapsed, conf.ProfileBasetime, transaction.Error)
			return
		}

		service_rec.Profiled = true
	}

	// JAVA NOT
	//	if conf.blocking_detect_enabled && transaction.elapsed >= conf.blocking_detect_time {
	//		BlockingDetect.getInstance().add(ctx.service_hash, ctx.service_name, ctx.start_time, transaction.elapsed)
	//	}
	//
	//	if conf.profile_enabled==false {
	//		return
	//	}

	steps := ctx.Profile.GetSteps()

	// add splitcount 202.07.20
	transaction.StepSplitCount = ctx.Profile.GetSplitCount()
	transaction.Active = ctx.ProfileActive > 0
	profile.SetProfile(steps)
	// profile 우선순위 낮게 처리
	//data.Send(profile)
	// data.SendProfile(profile)

	if conf.TraceZipEnabled {
		GetInstanceZipProfileThread().Add(profile)
	} else {
		data.SendProfile(profile)
	}
}

// SQL 일반화 ParsedSQL (Java TraceSQL) 의 해시를 reset
// Java는 기존에 DataText 에서 모두 reset하지만 import circle 오류로 새로 만듬.
// TODO hash 관련 Text reset 을 통합할 필요 있음.
func resetTraceSQL() {
	// traceMainLock 을 같이 쓰면 process 가 너무 빈번히 돌아서 resetTraceSQL 은 돌지 못함. lock 삭제
	// try catch, 오류가 발생해서 쓰레드가 종료되는 것을 처리.
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA554", " Recover:", r)
		}
	}()

	conf := config.GetConfig()

	// UTC 00시 기준으로 해시 초기화
	today := getDate()
	if lastDate != today || textReset != conf.TextReset {
		lastDate = today
		textReset = conf.TextReset
		logutil.Println("WA555", " SQL Text Reset")
		// ParsedSql
		resetSqlText()
	}
}

func getDate() int64 {
	return dateutil.Now() / dateutil.MILLIS_PER_HOUR
}

func IsBizException(ex *stat.ErrorThrowable) bool {
	conf := config.GetConfig()
	return conf.EnableBizExceptions_ && conf.BizExceptions.Contains(int32(stringutil.HashCode(ex.ErrorClassName)))
}
func IsIgnoreException(ex *stat.ErrorThrowable) bool {
	conf := config.GetConfig()
	return conf.EnableIgnoreExceptions_ && conf.IgnoreExceptions.Contains(int32(stringutil.HashCode(ex.ErrorClassName)))
}

// ToDo 검증 필요
func IsIgnoreExceptionLong(err int64) bool {
	conf := config.GetConfig()
	if conf.EnableIgnoreExceptions_ {
		return conf.IgnoreExceptions.Contains(bitutil.GetHigh64(err))
	}
	return false
}
