package config

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"

	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/magiconair/properties"

	langconf "github.com/whatap/go-api/agent/lang/conf"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/golib/lang"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/hmap"
	"github.com/whatap/golib/util/stringutil"
)

const (
	// Infra
	TCPCheck      = "tcp.check."
	LogFileWatch  = "log.file."
	EventLogWatch = "log.event."

	REFERER_FORMAT_ALL         int32 = 0
	REFERER_FORMAT_DOMAIN      int32 = 1
	REFERER_FORMAT_DOMAIN_PATH int32 = 2
	REFERER_FORMAT_PATH        int32 = 3

	DEFAULT_IGNORE_HEADER = "host,accept,user-agent,referer, accept-language, connection"

	APP_NAME_MAX_SIZE = 50

	DEFAULT_APP_NAME = "GO"
	DEFAULT_APP_TYPE = lang.APP_TYPE_GO
)

type Config struct {
	PCODE      int64
	OID        int64
	OKIND      int32
	OKIND_NAME string
	ONODE      int32
	ONODE_NAME string

	License    string
	AccessKey  string
	WhatapHost []string // TODO: whatap.server.host to whatap_server_host
	WhatapPort int32

	ObjectName string
	AppType    int16 // private
	// Was container name  ex) Apache, Django 등
	// 추후 ONAME, OID 생성에 사용.
	AppName string
	// 멀티 프로세스 방식에서 Was의 cputime, heap 등을 계산하기 위해서
	// Process List를 기준으로 추출하기 위한 프로세스 이름
	// e) Apache -> httpd ,
	AppProcessName string

	Shutdown           bool
	Enabled            bool
	Debug              bool
	TransactionEnabled bool

	CounterEnabled             bool
	CounterEnabledTranx_       bool
	CounterEnabledAct_         bool
	CounterEnabledUser_        bool
	CounterEnabledHttpc_       bool
	CounterEnabledSql_         bool
	CounterEnabledAgentInfo_   bool
	CounterEnabledHeap_        bool
	CounterEnabledProc_        bool
	CounterEnabledPackVer_     bool
	CounterEnabledSysPerfKube_ bool
	CounterEnabledSysPerf_     bool
	CounterEnabledActiveStats_ bool
	CounterEnabledDBPool_      bool
	CounterLogEnabled          bool

	CounterVersion byte
	CounterTimeout int32

	ApacheServerStatusUrl string
	NginxServerStatusUrl  string
	FpmServerStatusUrl    string

	StatDomainEnabled  bool
	StatDomainMaxCount int32

	StatMtraceEnabled  bool
	StatMtraceMaxCount int32

	StatLoginEnabled  bool
	StatLoginMaxCount int32

	StatRefererEnabled  bool
	StatRefererMaxCount int32
	// 0: full url, 1:domain(protocol+host), 2:uri, 3: domain + uri
	StatRefererFormat int32

	StatTxMaxCount        int32
	StatSqlMaxCount       int32
	StatHttpcMaxCount     int32
	StatErrorMaxCount     int32
	StatUseragentMaxCount int32

	StatEnabled         bool
	StatIpEnabled       bool
	RealtimeUserEnabled bool

	ActiveStackEnabled bool // TODO: activeStack 데이터 수집

	CypherLevel  int32 // TODO: AES-256 동작하지 않음.
	EncryptLevel int32 // TODO: 사용되지 않는 듯

	CountInterval int32 // private

	TcpSoTimeout         int32 // TODO
	TcpSoSendTimeout     int32 // TODO
	TcpConnectionTimeout int32 // TODO

	NetSendMaxBytes    int32
	NetSendBufferSize  int32
	NetWriteBufferSize int32
	NetSendQueue1Size  int32
	NetSendQueue2Size  int32

	NetUdpHost      string
	NetUdpPort      int32
	NetUdpReadBytes int32

	NetWriteLockEnabled bool

	UdpFlushStart             bool
	UdpFlushEnd               bool
	UdpFlushError             bool
	UdpProfileBaseTimeEnabled bool
	UdpProfileBaseTime        int32
	UdpTraceIgnoreTimeEnabled bool
	UdpTraceIgnoreTime        int32

	QueueLogEnabled           bool
	QueueYieldEnabled         bool
	QueueTcpEnabled           bool
	QueueTcpSenderThreadCount int32

	// Udp read 데이터를 channel로 전달, false 일경우 Queue 사용
	QueueUdpEnabled            bool
	QueueUdpSize               int32
	QueueUdpOverflowedSize     int32
	QueueUdpReadThreadCount    int32
	QueueUdpProcessThreadCount int32

	QueueProfileEnabled            bool
	QueueProfileSize               int32
	QueueProfileProcessThreadCount int32

	QueueTextEnabled            bool
	QueueTextSize               int32
	QueueTextProcessThreadCount int32

	QueueControlEnabled            bool
	QueueControlSize               int32
	QueueControlProcessThreadCount int32

	TxMaxCount          int32
	TxDefaultCapacity   int32
	TxDefaultLoadFactor float32

	// Throttle ~ TODO: java참고

	//profile
	ProfileHttpHeaderEnabled    bool   // TODO: 수집 유무(HTTP-HEADERS)
	ProfileHttpHeaderUrlPrefix  string // TODO: 수집 유무
	ProfileHttpHeaderIgnoreKeys *hmap.StringSet

	ProfileHttpParameterEnabled   bool   // TODO: 수집 유무(HTTP-PARAMETERS)
	ProfileHttpParameterUrlPrefix string // TODO: 수집 유무
	ProfileConnectionOpenEnabled  bool   // TODO: 수집 유무(DBCStep)
	ProfileDbcClose               bool   // TODO: 수집 유무(DB CLOSE_CONNECTION)

	ProfileStepNormalCount int32
	ProfileStepHeavyCount  int32
	ProfileStepMaxCount    int32
	ProfileStepHeavyTime   int32
	ProfileBasetime        int32

	ProfileSqlParamEnabled   bool // TODO: toParamBytes 유틸 필요
	ProfileSqlCommentEnabled bool

	ProfileSqlResourceEnabled    bool
	ProfileMethodResourceEnabled bool // TODO: MethodStep에서 사용 됨
	ProfileHttpcResourceEnabled  bool

	ProfilePositionSqlHash    int32  // TODO
	ProfilePositionHttpcHash  int32  // TODO
	ProfilePositionMethodHash int32  // TODO
	ProfilePositionSql        string // TODO
	ProfilePositionHttpc      string // TODO
	ProfilePositionMethod     string // TODO
	ProfilePositionDepth      int32  // TODO

	// XLog
	ProfileErrorSqlFetchMax int32 // TODO: sql-fech
	ProfileErrorSqlTimeMax  int32 // TODO: step error처리

	// Host
	// Http ServiceName을 기존 URI에서 HOST를 포함한 형식으로 출력   /HOST/URI , Default false
	ProfileHttpHostEnabled bool
	//trace
	// 1: ip, 2: Cookie SESSION ID(java JSESSION),
	// 3: 1) _user_header_ticket_enabled 일 경우 Http Header 에서 user_header_ticket 값을 설정.
	// 	  2) Cookie 값이 trace_user_cookie_limit 보다 크면 Ip 로 설정
	//    3) Cookie 에서 WHATAP 이름의 값을  사용
	//    4) 1),2),3) 모두 없으면 Cookie 에 값 설정 key: WHATAP value: KeyGen.Next()(랜덤)
	TraceUserEnabled             bool // TODO:  UseridUtil으로 기본값인 whatap cookie설정 해야함.
	TraceUserUsingIp             bool
	TraceUserHeaderTicket        string // TODO: UseridUtil
	TraceUserHeaderTicketEnabled bool   // private
	TraceUserSetCookie           bool
	TraceUserCookieLimit         int32
	TraceUserCookieKeys          []string
	TraceUserUsingType           int32 // private

	TraceHttpClientIpHeaderKeyEnabled bool
	TraceHttpClientIpHeaderKey        string // TODO: getRemoteAddr

	TraceAutoTransactionEnabled          bool // TODO: non http
	TraceAutoTransactionBackstackEnabled bool // TODO: non http
	TraceBackgroundSocketEnabled         bool // TODO: Socket 시작될 때 사용 됨

	TraceTransactionNameHeaderKey string // TODO: 수집 유무(service_name header값으로)
	TraceTransactionNameKey       string // TODO: 수집 유무(service_name parameter추가)

	TraceErrorCallstackDepth  int32 // TODO: stack
	TraceActiveCallstackDepth int32 // TODO: stack

	TraceActiveTransactionSlowTime     int64
	TraceActiveTransactionVerySlowTime int64
	TraceActiveTransactionLostTime     int64 // private

	TraceDbcLeakEnabled          bool // TODO: 수집 유무(db connection leak)
	TraceDbcLeakFullstackEnabled bool // TODO: 수집 유무(db connection leak full stack)
	DebugDbcStackEnabled         bool // TODO: 수집 유무(db connection stack)

	TraceGtxRate      int32  // TODO: gtx
	TraceGtxCallerKey string // private

	WebStaticContentExtensions string // TODO: ctx.isStaticContents

	TraceNormalizeEnabled     bool   // TODO: ServiceURLPatternDetector (for ctx.service_name)
	TraceNormalizeUrls        string // TODO: ServiceURLPatternDetector (for ctx.service_name)
	TraceAutoNormalizeEnabled bool   // TODO: ServiceURLPatternDetector (for ctx.service_name), Annotation으로 등록된 url pattern을 자동으로 검색, 저장, php 사용 안함.

	TraceHttpcNormalizeEnabled bool   // TODO: URLPatternDetector (for ctx.httpc_url)
	TraceHttpcNormalizeUrls    string // TODO: URLPatternDetector (for ctx.httpc_url)

	TraceSqlNormalizeEnabled bool

	TraceUserAgentEnabled bool
	TraceRefererEnabled   bool

	// Log
	LogRotationEnabled bool
	LogKeepDays        int
	// Log 중복 체크 기간 옵션  (내부 사용, 단위 초), Default 10초
	LogInterval int

	// Hook ~ TODO: java참고
	HookSignature int32

	ActiveStackSecond int32

	CounterProcfdEnabled  bool // TODO: file descript count
	CounterNetstatEnabled bool // TODO: setstat status count

	RealtimeUserThinktimeMax int64
	TimeSyncIntervalMs       int64
	DetectDeadlockEnabled    bool // TODO: ThreadStat util필요

	TextReset int32

	AutoOnameEnabled bool   // TODO:
	AutoOnamePrefix  string // TODO:
	AutoOnameReset   int32  // TODO:

	QueryStringEnabled bool
	QueryStringUrls    []string
	QueryStringKeys    []string

	ErrorSnapEnabled bool

	MtraceEnabled             bool
	MtraceAutoInjectEnabled   bool
	MtraceRate                int32
	MtraceCalleeTxidEnabled   bool
	TraceMtraceCallerKey      string
	TraceMtraceCalleeKey      string
	TraceMtraceInfoKey        string
	TraceMtracePoidKey        string
	TraceMtraceSpecKey        string
	TraceMtraceSpecKey1       string
	TraceMtraceTraceparentKey string

	MtraceSendUrlLength int32
	MtraceSpec          string
	MtraceSpecHash      int32

	// Self Meter
	MeterSelfEnabled bool
	//Interval
	MeterSelfInterval  int32
	MeterSelfBufferMax int32
	MeterSelfBufferMin int32

	TxCallerMeterEnabled      bool
	TxCallerMeterPKindEnabled bool
	SqlDbcMeterEnabled        bool
	HttpcHostMeterEnabled     bool
	ActxMeterEnabled          bool

	TpsAvg30Enabled bool

	TagCounterEnabled bool
	TagCountInterval  int32

	TelegrafEnabled      bool
	TelegrafPrefix       string
	TelegrafMaxSize      int32
	TelegrafTcpPort      int32
	TelegrafTcpSoTimeout int32

	BizExceptions           *hmap.IntSet
	EnableBizExceptions_    bool
	IgnoreExceptions        *hmap.IntSet
	EnableIgnoreExceptions_ bool

	TraceIgnoreUrlSet      *hmap.IntSet
	IsTraceIgnoreUrlPrefix bool
	TraceIgnoreUrlPrefix   string

	// event HitMap
	HitMapVerEventEnabled        bool
	HitMapVerEventErrorOnly      bool
	HitMapVerEventDuration       int32
	HitMapVerEventWarningPercent int32
	HitMapVerEventFatalPercent   int32
	HitMapVerEventInterval       int32

	HitMapHorizEventEnabled   bool
	HitMapHorizEventErrorOnly bool
	HitMapHorizEventBasetime  int32
	HitMapHorizEventDuration  int32
	HitMapHorizEventInterval  int32

	//pprof
	PProfEnabled     bool
	PProfCpuEnabled  bool
	PProfHttpEnabled bool
	PProfHttpAddress string
	PProfInterval    int32

	// SM
	WmiEnabled       bool
	NativeAPIEnabled bool
	ProcessFallback  bool

	// ActiveStat
	ActiveStatEnabled    bool
	ActiveStatLogEnabled bool
	// second
	ActiveStatResetInterval int32
	// second
	ActiveStatResetIntervalByActiveTxCount int32
	// second
	ActiveStatResetRateByActiveTxCount int32

	// Shm
	ShmEnabled            bool
	ShmKey                int64
	ShmSendMetricsEnabled bool
	ShmTxCounterEnabled   bool

	// PHP
	ExtErrorEnabled     bool
	ExtExceptionEnabled bool

	ProfileMethodEnabled      bool
	ProfileMethodTime         int32
	ProfileMethodStackEnabled bool

	ProfileInternalMethodEnabled      bool
	ProfileInternalMethodParamEnabled bool
	ProfileInternalMethodTime         int32
	ProfileCompileFileEnabled         bool
	ProfileCompileFileBasetime        int32
	ProfileSessionEnabled             bool
	MasterAgentHost                   string
	MasterAgentPort                   uint16
	PodName                           string
	WhatapMicroEnabled                bool
	EnvOKind                          string
	CorrectionFactorCpu               float32
	CorrectionFactorPCpu              float32

	// Infra
	LogCoolTime                time.Duration
	PerfCounterEnabled         bool
	PerformanceCounterInterval int32
	ServerProcessFDCheck       bool

	// windows perf
	PerfCounterJsonPath string

	ActiveStatsResetDuration time.Duration

	// Apdex
	ApdexTime   int32
	ApdexTime4T int32

	// Unix Domain Socket
	UnixSocketEnabled bool
	UnixSocket        string

	// error info
	ProfileCurlReturnEnabled    bool
	ProfileCurlErrorInfoEnabled bool
	ProfileCurlErrorIgnoreEmpty bool

	ProfileMysqlReturnEnabled    bool
	ProfileMysqlErrorInfoEnabled bool
	ProfileMysqlErrorIgnoreEmpty bool

	ProfileMysqliReturnEnabled    bool
	ProfileMysqliErrorInfoEnabled bool
	ProfileMysqliErrorIgnoreEmpty bool

	ProfilePDOReturnEnabled    bool
	ProfilePDOErrorInfoEnabled bool
	ProfilePDOErrorIgnoreEmpty bool

	ProfilePgsqlReturnEnabled    bool
	ProfilePgsqlErrorInfoEnabled bool
	ProfilePgsqlErrorIgnoreEmpty bool

	ProfileOci8ReturnEnabled    bool
	ProfileOci8ErrorInfoEnabled bool
	ProfileOci8ErrorIgnoreEmpty bool

	ProfileMssqlReturnEnabled    bool
	ProfileMssqlErrorInfoEnabled bool
	ProfileMssqlErrorIgnoreEmpty bool

	ProfileSqlsrvReturnEnabled    bool
	ProfileSqlsrvErrorInfoEnabled bool
	ProfileSqlsrvErrorIgnoreEmpty bool

	ProfileRedisReturnEnabled    bool
	ProfileRedisErrorInfoEnabled bool
	ProfileRedisErrorIgnoreEmpty bool

	ProfileCubridReturnEnabled    bool
	ProfileCubridErrorInfoEnabled bool
	ProfileCubridErrorIgnoreEmpty bool

	ProfileOdbcReturnEnabled    bool
	ProfileOdbcErrorInfoEnabled bool
	ProfileOdbcErrorIgnoreEmpty bool

	ActiveStackCount int32

	DebugTcpSendEnabled         bool
	DebugTcpSendTimeSyncEnabled bool
	DebugTcpSendPacks           *hmap.StringSet
	DebugTcpReadEnabled         bool
	DebugCounterEnabled         bool
	DebugControlEnabled         bool
	DebugUdpEnabled             bool
	DebugShmEnabled             bool
	DebugTxCounterEnabled       bool

	TraceDaemonEnabled bool
	TraceDaemonUrls    *hmap.StringSet

	TraceCLIEnabled bool

	DebugGCPercent     int32
	DebugGCPercentLast int32

	NvidiasmiEnabled bool
	PidLockEnabled   bool

	IgnoreHttpMethod []string

	//	// LogSink
	//	WatchLogEnabled       bool
	//	WatchLogCheckInterval int32
	//
	//	WatchLogReadCount  int32
	//	WatchLogBufferSize int32
	//	WatchLogLineSize   int32
	//	WatchLogSendCount  int32
	//
	//	//	public static boolean logsink_stdout_enabled = logsink_enabled;
	//	//	public static boolean logsink_stderr_enabled = logsink_enabled;
	//	//	public static boolean logsink_logback_enabled = logsink_enabled;
	//	//	public static boolean logsink_tomcat_enabled = logsink_enabled;
	//	//	public static boolean logsink_custom_enabled= logsink_enabled;
	//
	//	LogSinkEnabled bool
	//
	//	LogSinkQueueSize      int32
	//	DebugLogSinkEnabled   bool
	//	LogSinkLineSize       int32
	//	DebugLogSinkLineLimit int32
	//	LogSinkZipEnabled     bool
	//
	//	//	public static int max_buffer_size=1024 * 64;
	//	//	public static int max_wait_time = 2000;
	//	LogSinkZipMinSize      int32
	//	DebugLogSinkZipEnabled bool
	//	LogSinkZipLibpath      string

	ConfLogSink
	ConfFailover

	ConfDebugTest

	// Golang
	ConfGo
	ConfGoGrpc

	// profile, missing profile
	ConfProfile

	// Fowarder
	ConfFowarder

	ConfTrace
}

var conf *Config = nil
var mutex = sync.Mutex{}
var prop *properties.Properties = nil
var AppType int16 = 8

// whatap.server.host 같은 경우 dot(.) 문자 때문에 환경변수로 인식이 안되는 경우 지정 이름 사용.
var envKeys = map[string]string{
	"accesskey":             "WHATAP_ACCESSKEY",
	"license":               "WHATAP_LICENSE",
	"whatap.server.host":    "WHATAP_SERVER_HOST",
	"whatap.server.port":    "WHATAP_SERVER_PORT",
	"net_ipc_port":          "WHATAP_NET_IPC_PORT",
	"net_udp_port":          "WHATAP_NET_UDP_PORT",
	"otel_grpc_server_port": "WHATAP_OTEL_GRPC_SERVER_PORT",
}

func GetConfig() *Config {
	mutex.Lock()
	defer mutex.Unlock()
	if conf != nil {
		return conf
	}
	conf = new(Config)

	// GetConfig 이 main() 보다 먼저 호출되는 경우 -t 옵션으로 APP_TYPE을 설정.
	argsAppType := ""
	for i, it := range os.Args {
		it = strings.ToLower(it)
		if strings.HasPrefix(it, "-t") {
			if it == "-t" {
				argsAppType = os.Args[i+1]
			} else if pos := strings.Index(it, "="); pos > -1 {
				argsAppType = it[pos+1:]
			}
			break
		}
	}

	if argsAppType != "" {
		argsV, err := strconv.Atoi(argsAppType)
		if err != nil {
			// 환경 변수에서 다시 읽기
			aType := os.Getenv("WHATAP_APP_TYPE")
			if aType != "" {
				v, err := strconv.Atoi(aType)
				if err != nil {
					conf.AppType = lang.APP_TYPE_GO
				} else {
					conf.AppType = int16(v)
				}
			} else {
				// default 설정.
				conf.AppType = DEFAULT_APP_TYPE
			}
		} else {
			conf.AppType = int16(argsV)
		}
	} else {
		// 환경 변수에서 다시 읽기
		aType := os.Getenv("WHATAP_APP_TYPE")
		if aType != "" {
			v, err := strconv.Atoi(aType)
			if err != nil {
				conf.AppType = lang.APP_TYPE_GO
			} else {
				conf.AppType = int16(v)
			}
		} else {
			// default 설정.
			conf.AppType = DEFAULT_APP_TYPE
		}
	}
	// BSM AppType
	if conf.AppType == lang.APP_TYPE_BSM_PHP || conf.AppType == lang.APP_TYPE_BSM_PYTHON || conf.AppType == lang.APP_TYPE_BSM_DOTNET {
		logutil.SetLogID("opsnowbsm")
	}

	//init
	prop = properties.NewProperties()
	apply()

	reload()
	go run()

	return conf
}

func run() {
	for {
		// DEBUG goroutine log
		//logutil.Println("Config.run()")

		time.Sleep(3000 * time.Millisecond)
		reload()
	}
}

var last_file_time int64 = -1
var last_check int64 = 0

func reload() {
	// 종료 되지 않도록  Recover
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA211 Recover", r) //, string(debug.Stack()))
		}
	}()

	now := dateutil.Now()
	if now < last_check+3000 {
		return
	}
	last_check = now
	path := GetConfFile()

	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		if last_file_time == -1 {
			logutil.Println("WA212", "fail to load license file")
			if f, err := os.Create(path); err != nil {
				logutil.Println("WA212-01", "create file error ", err)
				return
			} else {
				logutil.Println("WA212-02", "create file path ", f)
			}
			return
		} else if last_file_time == 0 {
			logutil.Println("WA212-01", "fail to load license file")
			return
		}
		last_file_time = 0
		prop = properties.NewProperties()
		apply()

		logutil.Println("WA213", " Reload Config: ", GetConfFile())
		return
	}
	new_time := stat.ModTime().Unix()
	if last_file_time == new_time {
		return
	}
	last_file_time = new_time
	//prop = properties.MustLoadFile(path, properties.UTF8)
	prop, err = properties.LoadFile(path, properties.UTF8)
	apply()

	// Observer run
	langconf.RunConfObserver()

	logutil.Println("WA214", "Config: ", GetConfFile())

}
func GetConfFile() string {
	home := GetWhatapHome()
	// config 파일이 WHATAP_HOME 과 다른 경로에 있을 경우 설정.
	confHome := os.Getenv("WHATAP_CONFIG_HOME")
	if confHome != "" {
		home = confHome
	}

	confName := os.Getenv("WHATAP_CONFIG")
	if confName == "" {
		confName = "whatap.conf"
	}

	return filepath.Join(home, confName)
}

func GetWhatapHome() string {
	home := os.Getenv("WHATAP_HOME")
	if home == "" {
		home = "."
	}

	if conf.AppType == lang.APP_TYPE_DOTNET || conf.AppType == lang.APP_TYPE_BSM_DOTNET {
		dotnet_home := os.Getenv("WHATAP_DOTNET_HOME")
		if dotnet_home != "" {
			home = dotnet_home
		}
	}
	return home
}

func apply() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("apply recover ", r, ", \n", string(debug.Stack()))
		}
	}()
	logutil.Println("APP_TYPE", conf.AppType)
	conf.License = getValue("license")
	conf.AccessKey = getValue("accesskey")
	if strings.TrimSpace(conf.AccessKey) == "" && !(strings.TrimSpace(conf.License) == "") {
		conf.AccessKey = conf.License
	}
	// BSM
	if conf.AppType == lang.APP_TYPE_BSM_PHP || conf.AppType == lang.APP_TYPE_BSM_PYTHON {
		conf.WhatapHost = getStringArray("opsnowbsm.server.host", "/:,") // TODO: whatap.server.host to whatap_server_host
		conf.WhatapPort = getInt("opsnowbsm.server.port", 6600)
	} else {
		conf.WhatapHost = getStringArray("whatap.server.host", "/:,") // TODO: whatap.server.host to whatap_server_host
		conf.WhatapPort = getInt("whatap.server.port", 6600)
	}
	if conf.AppType == lang.APP_TYPE_PHP {
		conf.ObjectName = getValueDef("object_name", "{type}-{ip2}-{ip3}-{process}-{docker}-{ips}")
	} else if conf.AppType == lang.APP_TYPE_BSM_PHP {
		conf.ObjectName = getValueDef("object_name", "BSM-{type}-{ip2}-{ip3}-{process}-{docker}")
	} else if conf.AppType == lang.APP_TYPE_GO {
		conf.ObjectName = getValueDef("object_name", "{type}-{ip2}-{ip3}-{cmd}-{cmd_full}")
	} else {
		conf.ObjectName = getValueDef("object_name", "{type}-{ip2}-{ip3}-{process}")
	}

	//main에서 command-line-flag 처리
	//conf.AppType = int16(getInt("app_type", 3))
	conf.AppName = getValueDef("app_name", DEFAULT_APP_NAME)
	conf.AppName = stringutil.TruncateRune(conf.AppName, APP_NAME_MAX_SIZE)

	conf.AppProcessName = getValue("app_process_name")

	conf.Shutdown = getBoolean("shutdown", false)

	conf.Enabled = getBoolean("enabled", true)
	conf.TransactionEnabled = conf.Enabled && getBoolean("transaction_enabled", true)
	conf.CounterEnabled = conf.Enabled && getBoolean("counter_enabled", true)
	conf.CounterVersion = byte(getInt("counter_version", 2))
	conf.CounterTimeout = getInt("counter_timeout", 0)
	// 0은 작동 안함.
	if conf.CounterTimeout <= 0 {
		conf.CounterTimeout = 0
	} else {
		// 최소 1초
		if conf.CounterTimeout < 1000 {
			conf.CounterTimeout = 1000
		}
		conf.CounterTimeout = int32(math.Min(float64(conf.CounterTimeout), 4000))
	}

	conf.CounterLogEnabled = getBoolean("_counter_log_enabled", false)
	conf.CounterEnabledTranx_ = conf.Enabled && getBoolean("_counter_enabled_tranx", true)
	conf.CounterEnabledAct_ = conf.Enabled && getBoolean("_counter_enabled_act", true)
	conf.CounterEnabledUser_ = conf.Enabled && getBoolean("_counter_enabled_user", true)
	conf.CounterEnabledHttpc_ = conf.Enabled && getBoolean("_counter_enabled_httpc", true)
	conf.CounterEnabledSql_ = conf.Enabled && getBoolean("_counter_enabled_sql", true)
	conf.CounterEnabledAgentInfo_ = conf.Enabled && getBoolean("_counter_enabled_agentinfo", true)
	conf.CounterEnabledHeap_ = conf.Enabled && getBoolean("_counter_enabled_heap", true)
	conf.CounterEnabledProc_ = conf.Enabled && getBoolean("_counter_enabled_proc", true)
	conf.CounterEnabledPackVer_ = conf.Enabled && getBoolean("_counter_enabled_pack_ver", true)
	conf.CounterEnabledSysPerfKube_ = conf.Enabled && getBoolean("_counter_enabled_sys_perf_kube", true)
	conf.CounterEnabledSysPerf_ = conf.Enabled && getBoolean("_counter_enabled_sys_perf", true)
	conf.CounterEnabledActiveStats_ = conf.Enabled && getBoolean("_counter_enabled_act_stat", true)
	conf.CounterEnabledDBPool_ = conf.Enabled && getBoolean("_counter_enabled_db_pool", true)

	conf.QueueLogEnabled = getBoolean("queue_log_enabled", false)
	conf.QueueYieldEnabled = getBoolean("queue_yield_enabled", false)

	// Tcp 전송에 DoubleQueue 사용 여부 , 기본 channel
	conf.QueueTcpEnabled = getBoolean("queue_tcp_enabled", true)
	//conf.NetSendBuffer, conf.NetSendQueue1Size, conf.NetSenedQueue2Size
	conf.QueueTcpSenderThreadCount = getInt("queue_tcp_sender_thread_count", 2)
	// conf.QueueTcpSenderSleepTime = getInt("queue_tcp_sender_sleep_time", 0)
	// conf.QueueTcpSenderSleepCount = getLong("queue_tcp_sender_sleep_count", 0)

	// Udp channel , queue 사용, 기본 channel
	conf.QueueUdpEnabled = getBoolean("queue_udp_enabled", false)
	// Udp channel , queue 버퍼 크기
	conf.QueueUdpSize = getInt("queue_udp_size", 2048)
	// Udp queue 사용 overflowed 되면 tx end만 처리하는 overflowed queue의 크기
	conf.QueueUdpOverflowedSize = getInt("queue_udp_overflowed_size", 4096)
	// Udp read를 빠르게 하기 위한 스레드 개수 설정 기본 3
	conf.QueueUdpReadThreadCount = getInt("queue_udp_read_thread_count", 3)
	// Udp process (패킷 구분 처리 TxItem 생성 처리)를 빠르게 하기 위한 스레드 개수 설정 기본 3
	conf.QueueUdpProcessThreadCount = getInt("queue_udp_process_thread_count", 1)
	// conf.QueueUdpProcessSleepTime = getInt("queue_udp_process_sleep_time", 0)
	// conf.QueueUdpProcessSleepCount = getLong("queue_udp_process_sleep_count", 0)

	// TraceMain 에서 SendProfile 처리할 channel, queue 사용, 기본 channel
	conf.QueueProfileEnabled = getBoolean("queue_profile_enabled", true)
	// profile channel , queue 버퍼 크기
	conf.QueueProfileSize = getInt("queue_profile_size", 8192)
	// profile 처리 (ctx를 확인하고 tcp send 처리하는 과정)을 빠르게 하기 위한 스레드 개수 설정
	conf.QueueProfileProcessThreadCount = getInt("queue_profile_process_thread_count", 1)
	// conf.QueueProfileProcessSleepTime = getInt("queue_profile_process_sleep_time", 0)
	// conf.QueueProfileProcessSleepCount = getLong("queue_profile_process_sleep_count", 0)

	// TextPacke channel, queue 사용, 기본 channel
	conf.QueueTextEnabled = getBoolean("queue_text_enabled", true)
	conf.QueueTextSize = getInt("queue_text_size", 4096)
	conf.QueueTextProcessThreadCount = getInt("queue_text_process_thread_count", 1)
	// conf.QueueTextProcessSleepTime = getInt("queue_text_process_sleep_time", 1)
	// conf.QueueTextProcessSleepCount = getLong("queue_text_process_sleep_count", 0)

	// TODO 현재는 사용 안함 ControlHandler (에이전트로 요청하는 데이터 처리) channel, queue 사용, 기본 channel
	conf.QueueControlEnabled = getBoolean("queue_control_enabled", false)
	conf.QueueControlSize = getInt("queue_control_size", 100)
	conf.QueueControlProcessThreadCount = getInt("queue_control_process_thread_count", 1)
	// conf.QueueControlProcessSleepTime = getInt("queue_control_process_sleep_time", 0)
	// conf.QueueControlProcessSleepCount = getLong("queue_control_process_sleep_count", 0)

	conf.StatDomainEnabled = conf.Enabled && getBoolean("stat_domain_enabled", true)
	conf.StatDomainMaxCount = getInt("stat_domain_max_count", 7000)

	conf.StatMtraceEnabled = getBoolean("stat_mtrace_enabled", false)
	conf.StatMtraceMaxCount = getInt("stat_mtrace_max_count", 7000)

	conf.StatLoginEnabled = getBoolean("stat_login_enabled", false)
	conf.StatLoginMaxCount = getInt("stat_login_max_count", 7000)

	conf.StatRefererEnabled = getBoolean("stat_referer_enabled", false)
	conf.StatRefererMaxCount = getInt("stat_referer_max_count", 7000)
	conf.StatRefererFormat = getInt("stat_referer_format", 0)

	conf.StatTxMaxCount = getInt("stat_tx_max_count", 5000)
	conf.StatSqlMaxCount = getInt("stat_sql_max_count", 5000)
	conf.StatHttpcMaxCount = getInt("stat_httpc_max_count", 5000)
	conf.StatErrorMaxCount = getInt("stat_error_max_count", 1000)
	conf.StatUseragentMaxCount = getInt("stat_useragent_max_count", 500)

	conf.StatEnabled = conf.Enabled && getBoolean("stat_enabled", true)
	conf.StatIpEnabled = conf.Enabled && getBoolean("stat_ip_enabled", true)
	conf.RealtimeUserEnabled = conf.Enabled && getBoolean("realtime_user_enabled", true)

	if conf.AppType == lang.APP_TYPE_PHP || conf.AppType == lang.APP_TYPE_BSM_PHP {
		conf.ActiveStackEnabled = conf.Enabled && getBoolean("active_stack_enabled", false)
	} else if conf.AppType == lang.APP_TYPE_GO || conf.AppType == lang.APP_TYPE_BSM_GO {
		conf.ActiveStackEnabled = conf.Enabled && getBoolean("active_stack_enabled", false)
	} else {
		conf.ActiveStackEnabled = conf.Enabled && getBoolean("active_stack_enabled", true)
	}
	conf.CypherLevel = getInt("cypher_level", 128)
	conf.EncryptLevel = getInt("encrypt_level", 2)

	conf.PodName = os.Getenv("POD_NAME")
	if len(conf.PodName) < 1 {
		conf.PodName = os.Getenv("PODNAME")
	}
	conf.EnvOKind = os.Getenv("OKIND")
	conf.WhatapMicroEnabled = len(conf.PodName) > 0

	// Recover로 구문 예외 처리
	func() {
		defer func() {
			if r := recover(); r != nil {
				logutil.Println("WA217", " Recover ", r)
			}
		}()
		conf.OKIND_NAME = getValueDef("whatap.okind", cutOut(conf.PodName, "-"))
		if conf.OKIND_NAME == "" {
			conf.OKIND = 0
		} else {
			conf.OKIND = hash.HashStr(conf.OKIND_NAME)
		}
		conf.ONODE_NAME = getValueDef("whatap.onode", os.Getenv("NODE_NAME"))
		if len(conf.ONODE_NAME) < 1 {
			conf.ONODE_NAME = os.Getenv("NODE_IP")
		}
		if conf.ONODE_NAME == "" {
			conf.ONODE = 0
		} else {
			conf.ONODE = hash.HashStr(conf.ONODE_NAME)
		}
		logutil.Infoln("Config", "okind_name=", conf.OKIND_NAME, ", onode_name=", conf.ONODE_NAME)
	}()

	conf.CountInterval = getInt("_counter_interval", 5000)

	conf.TcpSoTimeout = getInt("tcp_so_timeout", 30000)
	conf.TcpSoSendTimeout = getInt("tcp_so_send_timeout", 20000)
	conf.TcpConnectionTimeout = getInt("tcp_connection_timeout", 5000)

	conf.NetSendMaxBytes = getInt("net_send_max_bytes", 5*1024*1024)
	conf.NetSendBufferSize = getInt("net_send_buffer_size", 1024)
	conf.NetWriteBufferSize = getInt("net_write_buffer_size", 8*1024*1024)
	conf.NetSendQueue1Size = getInt("net_send_queue1_size", 512)
	conf.NetSendQueue2Size = getInt("net_send_queue2_size", 1024)

	conf.NetWriteLockEnabled = getBoolean("net_write_lock_enabled", true)

	conf.NetUdpHost = getValueDef("net_udp_host", "127.0.0.1")
	conf.NetUdpPort = getInt("net_udp_port", 6600)
	conf.NetUdpReadBytes = getInt("net_udp_read_bytes", 2*1024*1024)
	conf.UdpFlushStart = getBoolean("net_udp_flush_start", true)
	conf.UdpFlushEnd = getBoolean("net_udp_flush_end", true)
	conf.UdpFlushError = getBoolean("net_udp_flush_error", false)
	conf.UdpProfileBaseTimeEnabled = getBoolean("net_udp_profile_basetime_enabled", false)
	conf.UdpProfileBaseTime = getInt("net_udp_profile_basetime", 200)
	conf.UdpTraceIgnoreTimeEnabled = getBoolean("net_udp_trace_ignoretime_enabled", false)
	conf.UdpTraceIgnoreTime = getInt("net_udp_trace_ignoretime", 50)

	conf.TxMaxCount = getInt("tx_max_count", 5000)
	conf.TxDefaultCapacity = getInt("tx_default_capacity", 101)
	strLoadFactor := getValueDef("tx_load_factor", "0.75")
	if s, err := strconv.ParseFloat(strLoadFactor, 32); err == nil {
		conf.TxDefaultLoadFactor = float32(s)
	}

	//logutil.Infoln("Config", "tx max_count=", conf.TxMaxCount, ",tx_cap=", conf.TxDefaultCapacity, ",tx_lf=", conf.TxDefaultLoadFactor, ",tx_ch_count=", conf.TxChannelCount)

	// Throttle ~ TODO: java참고

	conf.ProfileHttpHeaderEnabled = getBoolean("profile_http_header_enabled", false)
	conf.ProfileHttpHeaderUrlPrefix = getValueDef("profile_http_header_url_prefix", "/")

	// convert hyphen to underbar
	conf.ProfileHttpHeaderIgnoreKeys = toStringSet("profile_http_header_ignore_keys", strings.ReplaceAll(DEFAULT_IGNORE_HEADER, "-", "_"))

	conf.ProfileHttpParameterEnabled = getBoolean("profile_http_parameter_enabled", false)
	conf.ProfileHttpParameterUrlPrefix = getValueDef("profile_http_parameter_url_prefix", "/")
	conf.ProfileConnectionOpenEnabled = getBoolean("profile_connection_open_enabled", true)
	conf.ProfileDbcClose = getBoolean("profile_dbc_close", false)

	conf.ProfileStepNormalCount = getInt("profile_step_normal_count", 800)
	conf.ProfileStepHeavyCount = getInt("profile_step_heavy_count", 1000)
	conf.ProfileStepMaxCount = getInt("profile_step_max_count", 1024)
	conf.ProfileStepHeavyTime = getInt("profile_step_heavy_time", 100)

	if conf.ProfileStepHeavyCount > conf.ProfileStepMaxCount {
		conf.ProfileStepHeavyCount = conf.ProfileStepMaxCount
	}
	if conf.ProfileStepNormalCount > conf.ProfileStepHeavyCount {
		conf.ProfileStepNormalCount = conf.ProfileStepHeavyCount
	}

	conf.ProfileBasetime = getInt("profile_basetime", 500)

	conf.ProfileSqlParamEnabled = getBoolean("profile_sql_param_enabled", false)
	conf.ProfileSqlResourceEnabled = getBoolean("profile_sql_resource_enabled", false)
	conf.ProfileSqlCommentEnabled = getBoolean("profile_sql_comment_enabled", false)

	conf.ProfileMethodResourceEnabled = getBoolean("profile_method_resource_enabled", false)
	conf.ProfileHttpcResourceEnabled = getBoolean("profile_httpc_resource_enabled", false)

	// ProfilePositionxxxx ~ TODO: java참고
	conf.ProfilePositionSqlHash = getInt("_profile_position_sql_hash", 0)
	conf.ProfilePositionHttpcHash = getInt("_profile_position_httpc_hash", 0)
	conf.ProfilePositionMethodHash = getInt("_profile_position_method_hash", 0)
	conf.ProfilePositionSql = getValue("profile_position_sql")
	conf.ProfilePositionHttpc = getValue("profile_position_httpc")
	conf.ProfilePositionMethod = getValue("profile_position_method")
	if len(conf.ProfilePositionSql) > 0 {
		conf.ProfilePositionSqlHash = int32(hash.HashStr(conf.ProfilePositionSql))
	}

	if len(conf.ProfilePositionHttpc) > 0 {
		conf.ProfilePositionHttpcHash = int32(hash.HashStr(conf.ProfilePositionHttpc))
	}

	if len(conf.ProfilePositionMethod) > 0 {
		conf.ProfilePositionMethodHash = int32(hash.HashStr(conf.ProfilePositionMethod))
	}
	conf.ProfilePositionDepth = getInt("profile_position_depth", 50)

	conf.ProfileErrorSqlFetchMax = getInt("profile_error_sql_fetch_max", 10000)
	conf.ProfileErrorSqlTimeMax = getInt("profile_error_sql_time_max", 30000)

	conf.ProfileHttpHostEnabled = getBoolean("profile_http_host_enabled", false)

	conf.TraceUserEnabled = getBoolean("trace_user_enabled", true)
	conf.TraceUserUsingIp = getBoolean("trace_user_using_ip", false)
	conf.TraceUserHeaderTicket = getValue("trace_user_header_ticket")
	conf.TraceUserHeaderTicketEnabled = stringutil.IsNotEmpty(conf.TraceUserHeaderTicket)
	conf.TraceUserSetCookie = getBoolean("trace_user_set_cookie", false)
	conf.TraceUserCookieLimit = getInt("trace_user_cookie_limit", 2048)
	conf.TraceUserCookieKeys = getStringArray("trace_user_cookie_keys", ",")

	conf.TraceUserUsingType = 2 // default
	if !conf.TraceUserEnabled {
		conf.TraceUserUsingType = 0
	} else if conf.TraceUserUsingIp {
		conf.TraceUserUsingType = 1 // IP
	} else {
		conf.TraceUserUsingType = 2 // COOKIE
	}

	conf.TraceHttpClientIpHeaderKeyEnabled = getBoolean("trace_http_client_ip_header_key_enabled", true)
	conf.TraceHttpClientIpHeaderKey = getValueDef("trace_http_client_ip_header_key", "X-Forwarded-For")

	conf.TraceAutoTransactionEnabled = getBoolean("trace_auto_transaction_enabled", false)
	conf.TraceAutoTransactionBackstackEnabled = getBoolean("trace_auto_transaction_backstack_enabled", true)
	conf.TraceBackgroundSocketEnabled = getBoolean("trace_background_socket_enabled", false)

	conf.TraceTransactionNameHeaderKey = getValue("trace_transaction_name_header_key")
	conf.TraceTransactionNameKey = getValue("trace_transaction_name_key")

	conf.TraceErrorCallstackDepth = getInt("trace_error_callstack_depth", 50)
	conf.TraceActiveCallstackDepth = getInt("trace_active_callstack_depth", 50)

	conf.TraceActiveTransactionSlowTime = getLong("trace_active_transaction_slow_time", 3000)
	conf.TraceActiveTransactionVerySlowTime = getLong("trace_active_transaction_very_slow_time", 8000)
	conf.TraceActiveTransactionLostTime = getLong("trace_active_transaction_lost_time", 5*60000) // 5분

	conf.TraceDbcLeakEnabled = getBoolean("trace_dbc_leak_enabled", false)
	conf.TraceDbcLeakFullstackEnabled = getBoolean("trace_dbc_leak_fullstack_enabled", false)
	conf.DebugDbcStackEnabled = getBoolean("debug_dbc_stack_enabled", false)

	conf.TraceGtxRate = getInt("trace_gtx_rate", 0)
	conf.TraceGtxCallerKey = getValue("_trace_gtx_caller_key")

	conf.WebStaticContentExtensions = getValueDef("web_static_content_extensions", "js, htm, html, gif, png, jpg, css, swf, ico")

	conf.TraceNormalizeEnabled = getBoolean("trace_normalize_enabled", true)
	conf.TraceNormalizeUrls = getValue("trace_normalize_urls")
	conf.TraceAutoNormalizeEnabled = getBoolean("trace_auto_normalize_enabled", true)

	conf.TraceHttpcNormalizeEnabled = getBoolean("trace_httpc_normalize_enabled", true)
	conf.TraceHttpcNormalizeUrls = getValue("trace_httpc_normalize_urls")

	conf.TraceSqlNormalizeEnabled = getBoolean("trace_sql_normalize_enabled", true)
	conf.TraceUserAgentEnabled = getBoolean("trace_useragent_enabled", false)
	conf.TraceRefererEnabled = getBoolean("trace_referer_enabled", false)

	conf.LogRotationEnabled = getBoolean("log_rotation_enabled", true)
	// logutil conf 사용 변수 값 설정.
	logutil.SetLogRotationEnabled(conf.LogRotationEnabled)

	conf.LogKeepDays = int(getInt("log_keep_days", 7))
	// logutil conf 사용 변수 값 설정.
	logutil.SetLogKeepDays(conf.LogKeepDays)

	// Log 중복 체크 기간 옵션  (내부 사용, 단위 초), Default 10초
	conf.LogInterval = int(getInt("_log_interval", 10))
	// logutil conf 사용 변수 값 설정.
	logutil.SetLogInterval(conf.LogInterval)

	// Hook ~ TODO: java참고
	conf.HookSignature = getInt("hook_signature", 1)

	conf.ActiveStackSecond = getInt("active_stack_second", 10)

	conf.CounterProcfdEnabled = getBoolean("counter_procfd_enabled", false)
	conf.CounterNetstatEnabled = getBoolean("counter_netstat_enabled", false)

	conf.RealtimeUserThinktimeMax = int64(getInt("realtime_user_thinktime_max", 300000))
	conf.TimeSyncIntervalMs = getLong("time_sync_interval_ms", 30000)
	conf.DetectDeadlockEnabled = getBoolean("detect_deadlock_enabled", false)

	conf.TextReset = getInt("text_reset", 0)

	conf.AutoOnameEnabled = getBoolean("auto_oname_enabled", false)
	conf.AutoOnamePrefix = getValueDef("auto_oname_prefix", "agent")
	conf.AutoOnameReset = getInt("auto_oname_reset", 0)

	conf.QueryStringEnabled = getBoolean("query_string_enabled", false)
	conf.QueryStringUrls = getStringArray("query_string_urls", ",")
	conf.QueryStringKeys = getStringArray("query_string_keys", ",")

	conf.ErrorSnapEnabled = getBoolean("error_snap_enabled", true)

	conf.MtraceEnabled = getBoolean("mtrace_enabled", false)
	conf.MtraceAutoInjectEnabled = getBoolean("mtrace_auto_inject_enabled", true)
	conf.MtraceRate = getInt("mtrace_rate", 10)
	if conf.MtraceRate > 100 {
		conf.MtraceRate = 100
	} else if conf.MtraceRate < 0 {
		conf.MtraceRate = 0
	}
	conf.TraceMtraceCallerKey = getValueDef("mtrace_caller_key", "x-wtap-mst")
	conf.TraceMtraceCalleeKey = getValueDef("mtrace_callee_key", "x-wtap-tx")
	conf.TraceMtraceInfoKey = getValueDef("mtrace_info_key", "x-wtap-inf")
	conf.TraceMtracePoidKey = getValueDef("mtrace_poid_key", "x-wtap-po")
	conf.TraceMtraceSpecKey = getValueDef("mtrace_spec_key", "x-wtap-sp")
	conf.TraceMtraceSpecKey1 = getValueDef("mtrace_spec_key1", "x-wtap-sp1")
	conf.TraceMtraceTraceparentKey = getValueDef("mtrace_traceparent_key", "traceparent")
	conf.MtraceSendUrlLength = getInt("mtrace_send_url_length", 80)
	conf.MtraceSpec = getValueDef("mtrace_spec", "")
	if conf.MtraceSpec == "" {
		conf.MtraceSpecHash = 0
	} else {
		conf.MtraceSpec = strings.ReplaceAll(conf.MtraceSpec, ",", "_")
		conf.MtraceSpecHash = hash.HashStr(conf.MtraceSpec)
	}
	conf.MtraceCalleeTxidEnabled = getBoolean("mtrace_callee_txid_enabled", false)

	//2018.8.1 추가 에이전트 미터링 간격 (최소 5초 - counter 간격이 최소 간격임) - 기본 5분으로 변경, max 값 추가(cpu, mem)
	conf.MeterSelfEnabled = getBoolean("meter_self_enabled", true)
	conf.MeterSelfInterval = getInt("meter_self_interval", 300000)
	conf.MeterSelfBufferMin = getInt("meter_self_buffer_min", 100)
	conf.MeterSelfBufferMax = getInt("meter_self_buffer_max", 300)
	if conf.MeterSelfBufferMax < conf.MeterSelfBufferMin {
		conf.MeterSelfBufferMax = conf.MeterSelfBufferMin
	}

	conf.TxCallerMeterEnabled = getBoolean("tx_caller_meter_enabled", true)
	conf.TxCallerMeterPKindEnabled = getBoolean("tx_caller_meter_pkind_enabled", true)
	conf.SqlDbcMeterEnabled = getBoolean("sql_dbc_meter_enabled", true)
	conf.HttpcHostMeterEnabled = getBoolean("httpc_host_meter_enabled", true)
	conf.ActxMeterEnabled = getBoolean("actx_meter_enabled", true)

	// Calculate the average of tps with the last 30 seconds of data.
	conf.TpsAvg30Enabled = getBoolean("service_metrics_spike_enabled", false)

	conf.TagCounterEnabled = conf.Enabled && getBoolean("tag_counter_enabled", true)
	conf.TagCountInterval = getInt("tag_counter_interval", 10000)

	conf.TelegrafEnabled = getBoolean("telegraf_enabled", false)
	conf.TelegrafPrefix = getValueDef("telegraf_prefix", "Telegraf.")
	conf.TelegrafMaxSize = getInt("telegraf_max_size", 500)
	conf.TelegrafTcpPort = getInt("telegraf_tcp_port", 6600)
	conf.TelegrafTcpSoTimeout = getInt("telegraf_tcp_so_timeout", 60000)

	conf.BizExceptions = GetStringHashCodeSet("biz_exceptions", "", ",")
	conf.EnableBizExceptions_ = conf.BizExceptions != nil && conf.BizExceptions.Size() > 0
	conf.IgnoreExceptions = GetStringHashCodeSet("ignore_exceptions", "", ",")
	conf.EnableIgnoreExceptions_ = conf.IgnoreExceptions != nil && conf.IgnoreExceptions.Size() > 0

	conf.TraceIgnoreUrlSet = toHashSet("trace_ignore_url_set", "")
	conf.TraceIgnoreUrlPrefix = getValueDef("trace_ignore_url_prefix", "")
	if strings.TrimSpace(conf.TraceIgnoreUrlPrefix) == "" {
		conf.IsTraceIgnoreUrlPrefix = false
	} else {
		conf.IsTraceIgnoreUrlPrefix = true
	}

	conf.HitMapVerEventEnabled = getBoolean("hitmap_ver_event_enabled", false)
	conf.HitMapVerEventErrorOnly = getBoolean("hitmap_ver_event_error_only", false)
	conf.HitMapVerEventDuration = getInt("hitmap_ver_event_duration", 30000)
	conf.HitMapVerEventWarningPercent = getInt("hitmap_ver_event_warn_percent", 70)
	if conf.HitMapVerEventWarningPercent > 100 {
		conf.HitMapVerEventWarningPercent = 100
	}
	if conf.HitMapVerEventWarningPercent < 0 {
		conf.HitMapVerEventWarningPercent = 0
	}
	conf.HitMapVerEventFatalPercent = getInt("hitmap_ver_event_fatal_percent", 90)
	if conf.HitMapVerEventFatalPercent > 100 {
		conf.HitMapVerEventFatalPercent = 100
	}
	if conf.HitMapVerEventFatalPercent < 0 {
		conf.HitMapVerEventFatalPercent = 0
	}
	conf.HitMapVerEventInterval = getInt("hitmap_ver_event_interval", 300000)

	conf.HitMapHorizEventEnabled = getBoolean("hitmap_horiz_event_enabled", false)
	conf.HitMapHorizEventErrorOnly = getBoolean("hitmap_horiz_event_error_only", false)
	conf.HitMapHorizEventBasetime = getInt("hitmap_hoirz_event_basetime", 10000)
	conf.HitMapHorizEventDuration = getInt("hitmap_horiz_event_duration", 30000)
	conf.HitMapHorizEventInterval = getInt("hitmap_horiz_event_interval", 300000)

	// PHP Extension
	conf.ExtErrorEnabled = getBoolean("ext.error_enabled", true)
	conf.ExtExceptionEnabled = getBoolean("ext.exception_enabled", true)

	conf.ProfileMethodEnabled = getBoolean("profile_method_enabled", true)
	conf.ProfileMethodStackEnabled = getBoolean("profile_method_stack_enabled", false)

	conf.ProfileMethodTime = getInt("profile_method_time", 1000)
	conf.ProfileInternalMethodEnabled = getBoolean("profile_internal_method_enabled", false)
	conf.ProfileInternalMethodParamEnabled = getBoolean("profile_internal_method_param_enabled", false)
	conf.ProfileInternalMethodTime = getInt("profile_internal_method_time", 1000)
	conf.ProfileCompileFileEnabled = getBoolean("profile_compile_file_enabled", false)
	conf.ProfileCompileFileBasetime = getInt("profile_compile_file_basetime", 200)

	conf.ProfileSessionEnabled = getBoolean("profile_session_enabled", true)

	// BSM
	if conf.AppType == lang.APP_TYPE_BSM_PHP || conf.AppType == lang.APP_TYPE_BSM_PYTHON || conf.AppType == lang.APP_TYPE_BSM_DOTNET {
		// Realtime Diable
		conf.RealtimeUserEnabled = false
		// Topology Disable
		conf.TxCallerMeterEnabled = false
		conf.SqlDbcMeterEnabled = false
		conf.HttpcHostMeterEnabled = false
		conf.ActxMeterEnabled = false
	}

	// PProf
	conf.PProfEnabled = getBoolean("pprof_enabled", false)
	conf.PProfCpuEnabled = getBoolean("pprof_cpu_enabled", false)
	conf.PProfHttpEnabled = getBoolean("pprof_http_enabled", false)
	conf.PProfHttpAddress = getValueDef("pprof_http_address", "localhost:6600")
	conf.PProfInterval = getInt("pprof_interval", 30000)

	// SM
	conf.WmiEnabled = getBoolean("wmi.enabled", true)
	conf.NativeAPIEnabled = getBoolean("native_api.enabled", false)
	conf.ProcessFallback = getBoolean("processfallback", false)

	// ActiveStat
	conf.ActiveStatEnabled = getBoolean("active_stat_enabled", true)
	conf.ActiveStatLogEnabled = getBoolean("active_stat_log_enabled", false)
	conf.ActiveStatsResetDuration = time.Millisecond * time.Duration(getInt("active_stat_reset_duration", 100))
	// second
	conf.ActiveStatResetInterval = getInt("active_stat_reset_interval", 3600)
	// second
	conf.ActiveStatResetIntervalByActiveTxCount = getInt("active_stat_reset_interval_atc", 1800)
	// percent
	conf.ActiveStatResetRateByActiveTxCount = getInt("active_stat_reset_rate_atc", 150)

	// Shm
	conf.ShmEnabled = getBoolean("shm_enabled", true)
	conf.ShmKey = getLong("shm_key", 6600)
	conf.ShmSendMetricsEnabled = getBoolean("shm_send_metrics_enabled", false)
	conf.ShmTxCounterEnabled = getBoolean("shm_tx_counter_enabled", false)

	conf.MasterAgentHost = GetValueDef("master_agent_host", "whatap-master-agent.whatap-monitoring.svc.cluster.local")
	conf.MasterAgentPort = uint16(getLong("master_agent_port", 6600))

	conf.CorrectionFactorCpu = getFloat("correction_factor_cpu", float32(1))
	conf.CorrectionFactorPCpu = getFloat("correction_factor_pcpu", float32(1))

	// Infra
	conf.LogCoolTime = time.Second * time.Duration(getInt("log.cooltime", 180))
	conf.PerfCounterEnabled = getBoolean("perfcounter.enabled", false)
	conf.PerformanceCounterInterval = getInt("perfcounter.interval", 10)
	conf.PerfCounterJsonPath = getValueDef("perfcounter_jason_path", filepath.Join(GetWhatapHome(), "perfcounter.json"))

	conf.ServerProcessFDCheck = getBoolean("process.fdcheck", true)

	// Apdex
	conf.ApdexTime = getInt("apdex_time", 1200)
	conf.ApdexTime4T = 4 * conf.ApdexTime

	// Unix Domain Socket
	conf.UnixSocketEnabled = getBoolean("unix_socket_enabled", false)
	conf.UnixSocket = getValueDef("unix_socket", "whatap.sock")

	// error info
	conf.ProfileCurlReturnEnabled = getBoolean("profile_curl_return_enabled", true)
	conf.ProfileCurlErrorInfoEnabled = getBoolean("profile_curl_error_info_enabled", true)
	conf.ProfileCurlErrorIgnoreEmpty = getBoolean("profile_curl_error_ignore_empty", true)

	conf.ProfileMysqlReturnEnabled = getBoolean("profile_mysql_return_enabled", true)
	conf.ProfileMysqlErrorInfoEnabled = getBoolean("profile_mysql_error_info_enabled", true)
	conf.ProfileMysqlErrorIgnoreEmpty = getBoolean("profile_mysql_error_ignore_empty", true)

	conf.ProfileMysqliReturnEnabled = getBoolean("profile_mysqli_return_enabled", true)
	conf.ProfileMysqliErrorInfoEnabled = getBoolean("profile_mysqli_error_info_enabled", true)
	conf.ProfileMysqliErrorIgnoreEmpty = getBoolean("profile_mysqli_error_ignore_empty", true)

	conf.ProfilePDOReturnEnabled = getBoolean("profile_pdo_return_enabled", true)
	conf.ProfilePDOErrorInfoEnabled = getBoolean("profile_pdo_error_info_enabled", true)
	conf.ProfilePDOErrorIgnoreEmpty = getBoolean("profile_pdo_error_ignore_empty", true)

	conf.ProfilePgsqlReturnEnabled = getBoolean("profile_pgsql_return_enabled", true)
	conf.ProfilePgsqlErrorInfoEnabled = getBoolean("profile_pgsql_error_info_enabled", true)
	conf.ProfilePgsqlErrorIgnoreEmpty = getBoolean("profile_pgsql_error_ignore_empty", true)

	conf.ProfileOci8ReturnEnabled = getBoolean("profile_oci8_return_enabled", true)
	conf.ProfileOci8ErrorInfoEnabled = getBoolean("profile_oci8_error_info_enabled", true)
	conf.ProfileOci8ErrorIgnoreEmpty = getBoolean("profile_oci8_error_ignore_empty", true)

	conf.ProfileMssqlReturnEnabled = getBoolean("profile_mssql_return_enabled", true)
	conf.ProfileMssqlErrorInfoEnabled = getBoolean("profile_mssql_error_info_enabled", true)
	conf.ProfileMssqlErrorIgnoreEmpty = getBoolean("profile_mssql_error_ignore_empty", true)

	conf.ProfileSqlsrvReturnEnabled = getBoolean("profile_sqlsrv_return_enabled", true)
	conf.ProfileSqlsrvErrorInfoEnabled = getBoolean("profile_sqlsrv_error_info_enabled", true)
	conf.ProfileSqlsrvErrorIgnoreEmpty = getBoolean("profile_sqlsrv_error_ignore_empty", true)

	conf.ProfileRedisReturnEnabled = getBoolean("profile_redis_return_enabled", true)
	conf.ProfileRedisErrorInfoEnabled = getBoolean("profile_redis_error_info_enabled", true)
	conf.ProfileRedisErrorIgnoreEmpty = getBoolean("profile_redis_error_ignore_empty", false)

	conf.ProfileCubridReturnEnabled = getBoolean("profile_cubrid_return_enabled", true)
	conf.ProfileCubridErrorInfoEnabled = getBoolean("profile_cubrid_error_info_enabled", true)
	conf.ProfileCubridErrorIgnoreEmpty = getBoolean("profile_cubrid_error_ignore_empty", true)

	conf.ProfileOdbcReturnEnabled = getBoolean("profile_odbc_return_enabled", true)
	conf.ProfileOdbcErrorInfoEnabled = getBoolean("profile_odbc_error_info_enabled", true)
	conf.ProfileOdbcErrorIgnoreEmpty = getBoolean("profile_odbc_error_ignore_empty", true)

	conf.ActiveStackCount = getInt("active_stack_count", 50)

	// Debug option
	// Debug
	conf.Debug = getBoolean("debug", false)
	conf.DebugTcpSendEnabled = getBoolean("debug_tcpsend_enabled", false)
	conf.DebugTcpSendTimeSyncEnabled = getBoolean("debug_tcpsend_timesync_enabled", false)
	conf.DebugTcpSendPacks = toStringSet("debug_tcpsend_packs", "CounterPack1")
	conf.DebugTcpReadEnabled = getBoolean("debug_tcpread_enabled", false)
	conf.DebugCounterEnabled = getBoolean("debug_counter_enabled", false)

	conf.DebugControlEnabled = getBoolean("debug_control_enabled", false)
	conf.DebugUdpEnabled = getBoolean("debug_udp_enabled", false)
	conf.DebugShmEnabled = getBoolean("debug_shm_enabled", false)
	conf.DebugTxCounterEnabled = getBoolean("debug_tx_counter_enabled", false)

	conf.TraceDaemonEnabled = getBoolean("trace_daemon_enabled", false)
	conf.TraceDaemonUrls = toStringSet("trace_daemon_urls", "")

	conf.TraceCLIEnabled = getBoolean("trace_cli_enabled", true)

	// conf.DebugGCPercent = getInt("debug_gc_percent", 100)
	// if conf.DebugGCPercent > 1000 {
	// 	conf.DebugGCPercent = 1000
	// }
	// if conf.DebugGCPercentLast != conf.DebugGCPercent {
	// 	debug.SetGCPercent(int(conf.DebugGCPercent))
	// 	conf.DebugGCPercentLast = conf.DebugGCPercent
	// 	logutil.Println("WA211-01", "Change GC Percent ", int(conf.DebugGCPercent))
	// }

	conf.NvidiasmiEnabled = getBoolean("nvidiasmi_enabled", false)
	conf.PidLockEnabled = getBoolean("pidlock_enabled", false)

	//
	conf.IgnoreHttpMethod = getStringArrayDef("ignore_http_method", ",", "PATCH, OPTIONS, HEAD, TRACE")

	// LogSink
	conf.ConfLogSink.Apply(conf)

	// Failover
	conf.ConfFailover.Apply(conf)

	// Debug
	conf.ConfDebugTest.Apply(conf)

	// Golang
	conf.ConfGo.Apply(conf)
	conf.ConfGoGrpc.Apply(conf)

	conf.ConfProfile.Apply(conf)

	// Fowarder
	conf.ConfFowarder.Apply(conf)

	conf.ConfTrace.Apply(conf)
}
func GetValue(key string) string { return getValue(key) }
func getValue(key string) string {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("getvalue recover ", r, ", \n", string(debug.Stack()))
		}
	}()
	envVal := os.Getenv(key)
	if envVal == "" {
		// 동일한 이름의 env 값이 없으면, 지정된 env key 이름으로 값을 가져옴.
		if v, ok := envKeys[key]; ok {
			envVal = os.Getenv(v)
		}
	}
	//php prefix whatap.
	if conf.AppType == lang.APP_TYPE_PHP {
		if !strings.HasPrefix(key, "whatap.") {
			key = "whatap." + key
		}
	} else if conf.AppType == lang.APP_TYPE_BSM_PHP {
		if !strings.HasPrefix(key, "opsnowbsm.") {
			key = "opsnowbsm." + key
		}
	}

	value, ok := prop.Get(key)
	if ok == false {
		return strings.TrimSpace(envVal)
	}

	return strings.TrimSpace(value)
}
func GetValueDef(key, def string) string { return getValueDef(key, def) }
func getValueDef(key string, def string) string {
	v := getValue(key)

	if v == "" {
		return def
	}

	return v
}
func GetBoolean(key string, def bool) bool {
	return getBoolean(key, def)
}
func getBoolean(key string, def bool) bool {
	v := getValue(key)
	if v == "" {
		return def
	}
	value, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return value
}
func GetInt(key string, def int) int32 {
	return getInt(key, def)
}
func getInt(key string, def int) int32 {
	v := getValue(key)
	if v == "" {
		return int32(def)
	}
	value, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		return int32(def)
	}
	return int32(value)
}

func GetIntSet(key, defaultValue, deli string) *hmap.IntSet {
	set := hmap.NewIntSet()
	vv := stringutil.Tokenizer(GetValueDef(key, defaultValue), deli)
	if vv != nil {
		for _, x := range vv {
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Continue
					}
				}()
				if xx, err := strconv.Atoi(stringutil.TrimEmpty(x)); err != nil {
					set.Put(int32(xx))
				}
			}()
		}
	}
	return set
}

func GetStringHashSet(key, defaultValue, deli string) *hmap.IntSet {
	set := hmap.NewIntSet()
	vv := stringutil.Tokenizer(GetValueDef(key, defaultValue), deli)
	if vv != nil {
		for _, x := range vv {
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Continue
					}
				}()
				xx := hash.HashStr(stringutil.TrimEmpty(x))
				set.Put(xx)
			}()
		}
	}
	return set
}

func GetStringHashCodeSet(key, defaultValue, deli string) *hmap.IntSet {
	set := hmap.NewIntSet()
	vv := stringutil.Tokenizer(GetValueDef(key, defaultValue), deli)
	if vv != nil {
		for _, x := range vv {
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Continue
					}
				}()
				xx := stringutil.HashCode(stringutil.TrimEmpty(x))
				set.Put(int32(xx))
			}()
		}
	}
	return set
}
func GetLong(key string, def int64) int64 {
	return getLong(key, def)
}
func getLong(key string, def int64) int64 {
	v := getValue(key)
	if v == "" {
		return def
	}
	value, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return value
}
func GetStringArray(key string, deli string) []string {
	return getStringArray(key, deli)
}

func getStringArray(key string, deli string) []string {
	v := getValue(key)
	if v == "" {
		return []string{}
	}
	tokens := stringutil.Tokenizer(v, deli)
	// trim Space
	trimTokens := make([]string, 0)
	for _, v := range tokens {
		trimTokens = append(trimTokens, strings.TrimSpace(v))
	}
	return trimTokens
}

func getStringArrayDef(key string, deli string, def string) []string {
	v := getValueDef(key, def)
	if v == "" {
		return []string{}
	}
	tokens := stringutil.Tokenizer(v, deli)
	// trim Space
	trimTokens := make([]string, 0)
	for _, v := range tokens {
		trimTokens = append(trimTokens, strings.TrimSpace(v))
	}
	return trimTokens
}

func getFloat(key string, def float32) float32 {
	v := getValue(key)
	if v == "" {
		return float32(def)
	}
	value, err := strconv.ParseFloat(v, 32)
	if err != nil {
		return float32(def)
	}
	return float32(value)
}

func SetValues(keyValues *map[string]string) {
	path := GetConfFile()
	//props := properties.MustLoadFile(path, properties.UTF8)
	props, err := properties.LoadFile(path, properties.UTF8)
	if err != nil {
		logutil.Errorf("WA21901", "error load properties %v", err)
		return
	}
	for key, value := range *keyValues {
		if strings.TrimSpace(key) != "" {
			//php prefix whatap.
			if conf.AppType == lang.APP_TYPE_PHP {
				if !strings.HasPrefix(key, "whatap.") && key != "extension" {
					key = "whatap." + key
				}
			} else if conf.AppType == lang.APP_TYPE_BSM_PHP {
				if !strings.HasPrefix(key, "opsnowbsm.") && key != "extension" {
					key = "opsnowbsm." + key
				}
			}
		}

		props.Set(key, value)
	}

	line := ""
	if f, err := os.OpenFile(path, os.O_RDWR, 0644); err != nil {
		logutil.Println("WA215", " Error ", err)
		return
	} else {
		defer f.Close()

		r := bufio.NewReader(f)
		new_keys := props.Keys()
		old_keys := map[string]bool{}
		for {
			data, _, err := r.ReadLine()
			if err != nil { // new key
				for _, key := range new_keys {
					if old_keys[key] {
						continue
					}
					match, _ := regexp.MatchString("^\\w", key)
					if match {
						value, _ := props.Get(key)
						if strings.TrimSpace(value) != "" {
							tmp := strings.Replace(value, "\\\\", "\\", -1)
							tmp = strings.Replace(tmp, "\\", "\\\\", -1)
							line += fmt.Sprintf("%s=%s\n", key, tmp)
						}
					}
				}
				break
			}
			if strings.Index(string(data), "=") == -1 {
				line += fmt.Sprintf("%s\n", string(data))
				//io.WriteString(f, line)
			} else {
				datas := strings.Split(string(data), "=")
				key := strings.Trim(datas[0], " ")
				value := strings.Trim(datas[1], " ")
				old_keys[key] = true

				match, _ := regexp.MatchString("^\\w", key)
				if match {
					value, _ = props.Get(key)
				}
				// value 가 없는 경우 항목 추가 안함(삭제)
				if strings.TrimSpace(value) != "" {
					tmp := strings.Replace(value, "\\\\", "\\", -1)
					tmp = strings.Replace(tmp, "\\", "\\\\", -1)

					line += fmt.Sprintf("%s=%s\n", key, tmp)
				}
				//io.WriteString(f, line)
			}
		}
	}

	if f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0644); err != nil {
		logutil.Println("WA216", " Error ", err)
		return
	} else {
		defer f.Close()
		io.WriteString(f, line)

		// flush
		f.Sync()
	}
}
func ToString() string {
	sb := stringutil.NewStringBuffer()
	if prop == nil {
		return ""
	}
	for _, key := range prop.Keys() {
		if v, ok := prop.Get(key); ok {
			sb.Append(key).Append("=").AppendLine(v)
		}
	}

	sb.Append("pcode").Append("=").AppendLine(fmt.Sprintf("%d", conf.PCODE))
	sb.Append("oid").Append("=").AppendLine(fmt.Sprintf("%d", conf.OID))
	sb.Append("okind").Append("=").AppendLine(fmt.Sprintf("%d", conf.OKIND))
	sb.Append("okind_name").Append("=").AppendLine(conf.OKIND_NAME)
	sb.Append("onode").Append("=").AppendLine(fmt.Sprintf("%d", conf.ONODE))
	sb.Append("onode_name").Append("=").AppendLine(conf.ONODE_NAME)

	return sb.ToString()
}

func SearchKey(keyPrefix string) *map[string]string {
	keyValues := map[string]string{}
	for _, key := range prop.Keys() {
		if strings.HasPrefix(key, keyPrefix) {
			if v, ok := prop.Get(key); ok {
				keyValues[key] = v
			}
		}
	}

	return &keyValues
}

func FilterPrefix(keyPrefix string) map[string]string {
	keyValues := make(map[string]string)
	//php prefix whatap.
	if conf.AppType == lang.APP_TYPE_PHP {
		if !strings.HasPrefix(keyPrefix, "whatap.") {
			keyPrefix = "whatap." + keyPrefix
		}
	} else if conf.AppType == lang.APP_TYPE_BSM_PHP {
		if !strings.HasPrefix(keyPrefix, "opsnowbsm.") {
			keyPrefix = "opsnowbsm." + keyPrefix
		}
	}
	pp := prop.FilterPrefix(keyPrefix)
	for _, key := range pp.Keys() {
		keyValues[key] = pp.GetString(key, "")
	}
	return keyValues
}

func cutOut(val, delim string) string {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA217", " Recover ", r)
		}
	}()
	if val == "" {
		return val
	}
	x := strings.LastIndex(val, delim)
	if x <= 0 {
		return ""
	}
	//return val.substring(0, x);
	return val[0:x]

}

func toHashSet(key, def string) *hmap.IntSet {
	set := hmap.NewIntSet()
	vv := strings.Split(getValueDef(key, def), ",")
	if vv != nil {
		for _, x := range vv {
			func() {
				defer func() {
					if r := recover(); r != nil {
						logutil.Infoln("WA218", " Recover ", r)
					}
				}()

				x = strings.TrimSpace(x)
				if len(x) > 0 {
					xx := hash.HashStr(x)
					set.Put(xx)
				}
			}()
		}
	}
	return set
}

func toStringSet(key, def string) *hmap.StringSet {
	set := hmap.NewStringSet()
	vv := strings.Split(getValueDef(key, def), ",")
	if vv != nil {
		for _, x := range vv {
			func() {
				defer func() {
					if r := recover(); r != nil {
						logutil.Infoln("WA219", " Recover ", r)
					}
				}()
				x = strings.TrimSpace(x)
				if len(x) > 0 {
					set.Put(x)
				}
			}()
		}
	}
	return set
}

func IsIgnoreTrace(hash int32, service string) bool {
	if conf.TraceIgnoreUrlSet == nil {
		return false
	}

	if conf.TraceIgnoreUrlSet.Contains(hash) {
		return true
	}
	if conf.IsTraceIgnoreUrlPrefix {
		if strings.HasPrefix(service, conf.TraceIgnoreUrlPrefix) {
			return true
		}
	}
	return false
}
func InArray(str string, list []string) bool {
	for _, it := range list {
		if strings.ToUpper(strings.TrimSpace(str)) == strings.ToUpper(strings.TrimSpace(it)) {
			return true
		}
	}
	return false
}

func InArrayCaseSensitive(str string, list []string) bool {
	for _, it := range list {
		if strings.TrimSpace(str) == strings.TrimSpace(it) {
			return true
		}
	}
	return false
}
