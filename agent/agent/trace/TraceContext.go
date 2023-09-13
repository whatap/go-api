package trace

import (
	"fmt"
	"sync"

	"github.com/whatap/golib/lang/step"
	"github.com/whatap/golib/lang/value"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/hexa32"
	"github.com/whatap/golib/util/urlutil"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/stat"
)

type TraceContext struct {
	Txid int64

	// bool
	IsStaticContents bool

	// int64
	ProfileSeq int64

	// thread, thread_id skip
	//public Thread thread;
	//public long thread_id;

	Pid int32
	// int64
	ThreadId int64

	ThreadStack string

	// int64
	CallerPcode int64
	// int32
	CallerOid int32
	// int64
	CallerSeq int64

	// *ProfileCollector
	// Profile *ProfileCollector
	Profile IProfileCollector

	// int64
	StartCpu int64
	// int64
	StartMalloc int64

	// int64 기존 java -> Service.cpuTime, malloc 계산을 위해 추가
	EndCpu int64
	// int64 기존 java -> Service.cpuTime, malloc 계산을 위해 추가
	EndMalloc int64

	// int64
	StartTime int64
	EndTime   int64

	// 기존 java -> getElapsedTime() 현재시간에서 Start_time의 차이
	// int32
	Elapsed int32

	// int32
	Status int32

	// int32
	ServiceHash int32
	// string
	ServiceName string
	// URLutil
	ServiceURL *urlutil.URL

	// int32
	RemoteIp int32

	// bool
	ErrorStep bool

	// int64
	Error int64

	// BixException 통계 제외 처리를 위해
	ErrorLevel byte

	// 임재환 추가 Java Thread thr 대시 stat.ErrorThrowable 구조체 사용
	Thr *stat.ErrorThrowable

	// string
	HttpMethod string
	// string
	HttpQuery string
	// string
	HttpContentType string

	// http host
	HttpHost     string
	HttpHostHash int32

	// int32
	SqlCount int32
	// int32
	SqlTime int32

	DbcTime int32

	FetchCount int32
	FetchTime  int64

	// 2017.5.23 임재환 추가
	// int32
	SqlInsert int32
	// int32
	SqlUpdate int32
	// int32
	SqlDelete int32
	// int32
	SqlSelect int32
	// int32
	SqlOthers int32

	// string
	PstmtSql string
	// int32
	PstmtHash int32

	// 2017.5.23 임재환 추가
	// int32
	ExecutedSqlhash int32
	// int32
	ActiveSqlhash int32
	// int32
	ActiveDbc int32
	// byte
	ActiveCrud byte

	// int32
	HttpcCount int32
	// int32
	HttpcTime int32

	// 2017.5.23 임재환 추가
	// string
	HttpcUrl string

	// 2017.5.23 임재환 추가
	// int32
	ActiveHttpcHash int32
	// string
	HttpcHost string
	// int32
	HttpcPort int32

	WClientId       int64
	UserAgent       int32
	UserAgentString string
	Referer         int32
	RefererURL      *urlutil.URL
	Login           string

	// 2017.5.23 임재환 추가
	//Login						string
	UserTransaction int32
	// bool
	DebugSqlCall bool
	// step.SqlStepX
	LastSqlStep *step.SqlStepX
	// intew
	ProfileActive int32

	JdbcUpdated      int32
	JdbcUpdateRecord int32
	JdbcIdenity      int32
	JdbcCommit       int32
	//public IntKeyLinkedMap<ResultStat> resultsql = new IntKeyLinkedMap<ResultStat>().setMax(11);

	// int32
	RsCount int32
	// int64
	RsTime int64
	// 2017.5.23 임재환 추가
	// bool
	DBOpening bool

	//	public Object working_rs;
	//	public ResultStat working_rstat;
	//
	//	public StringKeyLinkedMap<Object> attr;

	Mtid    int64
	Mdepth  int32
	Mcallee int64

	McallerTxid    int64
	McallerPcode   int64
	McallerSpec    string
	McallerUrl     string
	McallerUrlHash int32

	McallerOid     int32
	McallerOkind   int32
	McallerPoidKey string

	//Fields []*service.FIELD
	Fields *value.MapValue

	//PoolNewInstance string
}

var ctxPool = sync.Pool{
	New: func() interface{} {
		return NewTraceContext()
	},
}

func NewTraceContext() *TraceContext {
	p := new(TraceContext)
	p.Profile = NewProfileCollector(conf.InternalTraceCollectingMode, p)
	//p.PoolNewInstance = "new"
	return p
}
func PoolTraceContext() *TraceContext {
	p := ctxPool.Get().(*TraceContext)
	//logutil.Infoln(">>>>", "Get Ctx is new=", p.PoolNewInstance)
	return p
}

func CloseTraceContext(ctx *TraceContext) {
	if ctx != nil {
		//logutil.Infoln(">>>>", "Put Ctx txid=", ctx.Txid)
		ctx.Clear()
		//ctx.PoolNewInstance = "used"
		ctxPool.Put(ctx)
	}
}
func (this *TraceContext) Clear() {
	this.Txid = 0

	// bool
	this.IsStaticContents = false

	// int64
	this.ProfileSeq = 0

	// thread, thread_id skip
	//public Thread thread;
	//public long thread_id;

	this.Pid = 0
	// int64
	this.ThreadId = 0

	this.ThreadStack = ""

	// int64
	this.CallerPcode = 0
	// int32
	this.CallerOid = 0
	// int64
	this.CallerSeq = 0

	// *ProfileCollector
	this.Profile = NewProfileCollector(conf.InternalTraceCollectingMode, this)

	// int64
	this.StartCpu = 0
	// int64
	this.StartMalloc = 0

	// int64 기존 java -> Service.cpuTime, malloc 계산을 위해 추가
	this.EndCpu = 0
	// int64 기존 java -> Service.cpuTime, malloc 계산을 위해 추가
	this.EndMalloc = 0

	// int64
	this.StartTime = 0
	this.EndTime = 0

	// 기존 java -> getElapsedTime() 현재시간에서 Start_time의 차이
	// int32
	this.Elapsed = 0

	// int32
	this.Status = 0

	// int32
	this.ServiceHash = 0
	// string
	this.ServiceName = ""
	// URLutil
	this.ServiceURL = nil

	// int32
	this.RemoteIp = 0

	// bool
	this.ErrorStep = false

	// int64
	this.Error = 0

	// BixException 통계 제외 처리를 위해
	this.ErrorLevel = 0

	// 임재환 추가 Java Thread thr 대시 stat.ErrorThrowable 구조체 사용
	this.Thr = nil

	// string
	this.HttpMethod = ""
	// string
	this.HttpQuery = ""
	// string
	this.HttpContentType = ""

	// http host
	this.HttpHost = ""
	this.HttpHostHash = 0

	// int32
	this.SqlCount = 0
	// int32
	this.SqlTime = 0

	this.DbcTime = 0

	this.FetchCount = 0
	this.FetchTime = 0

	// 2017.5.23 임재환 추가
	// int32
	this.SqlInsert = 0
	// int32
	this.SqlUpdate = 0
	// int32
	this.SqlDelete = 0
	// int32
	this.SqlSelect = 0
	// int32
	this.SqlOthers = 0

	// string
	this.PstmtSql = ""
	// int32
	this.PstmtHash = 0

	// 2017.5.23 임재환 추가
	// int32
	this.ExecutedSqlhash = 0
	// int32
	this.ActiveSqlhash = 0
	// int32
	this.ActiveDbc = 0
	// byte
	this.ActiveCrud = 0

	// int32
	this.HttpcCount = 0
	// int32
	this.HttpcTime = 0

	// 2017.5.23 임재환 추가
	// string
	this.HttpcUrl = ""

	// 2017.5.23 임재환 추가
	// int32
	this.ActiveHttpcHash = 0
	// string
	this.HttpcHost = ""
	// int32
	this.HttpcPort = 0

	this.WClientId = 0
	this.UserAgent = 0
	this.UserAgentString = ""
	this.Referer = 0
	this.RefererURL = nil
	this.Login = ""

	// 2017.5.23 임재환 추가
	//Login						string
	this.UserTransaction = 0
	// bool
	this.DebugSqlCall = false
	// step.SqlStepX
	this.LastSqlStep = nil
	// intew
	this.ProfileActive = 0

	this.JdbcUpdated = 0
	this.JdbcUpdateRecord = 0
	this.JdbcIdenity = 0
	this.JdbcCommit = 0
	//public IntKeyLinkedMap<ResultStat> resultsql = new IntKeyLinkedMap<ResultStat>().setMax(11);

	// int32
	this.RsCount = 0
	// int64
	this.RsTime = 0
	// 2017.5.23 임재환 추가
	// bool
	this.DBOpening = false

	//	public Object working_rs;
	//	public ResultStat working_rstat;
	//
	//	public StringKeyLinkedMap<Object> attr;

	this.Mtid = 0
	this.Mdepth = 0
	this.Mcallee = 0

	this.McallerTxid = 0
	this.McallerPcode = 0
	this.McallerSpec = ""
	this.McallerUrl = ""
	this.McallerUrlHash = 0

	this.McallerOid = 0
	this.McallerOkind = 0

	this.Fields = nil
}

// func (this *TraceContext) Add(id int, value string) {
// 	if this.Fields == nil {
// 		this.Fields = make([]*service.FIELD, 0)
// 	}
// 	f := service.NewFIELD()
// 	f.Id = byte(id)
// 	f.Value = value
// 	this.Fields = append(this.Fields, f)
// }
// func (this *TraceContext) GetFields() []*service.FIELD {
// 	if this.Fields == nil {
// 		return nil
// 	}
// 	out := make([]*service.FIELD, 0)
// 	out = append(out, this.Fields...)
// 	return out
// }

func (this *TraceContext) SetExtraField(key string, val value.Value) {
	if this.Fields == nil {
		this.Fields = value.NewMapValue()
	}
	this.Fields.Put(key, val)
}
func (this *TraceContext) GetExtraField(key string) value.Value {
	if this.Fields == nil {
		this.Fields = value.NewMapValue()
	}
	return this.Fields.Get(key)
}
func (this *TraceContext) ExtraFields() *value.MapValue {
	if this.Fields == nil {
		this.Fields = value.NewMapValue()
	}
	return this.Fields
}

func (this *TraceContext) GetElapsedTime() int {
	return int(dateutil.SystemNow() - this.StartTime)
}

var transferPoid string

func TransferPOID() string {
	if transferPoid != "" {
		return transferPoid
	}
	UpdatePOID()
	return transferPoid
}

func UpdatePOID() {
	conf := config.GetConfig()
	transferPoid = fmt.Sprintf("%s,%s,%s", hexa32.ToString32(conf.PCODE), hexa32.ToString32(int64(conf.OKIND)), hexa32.ToString32(conf.OID))
}
