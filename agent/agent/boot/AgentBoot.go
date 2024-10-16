package boot

import (
	//"log"
	"runtime"

	"runtime/debug"

	"os"
	"strconv"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/control"
	"github.com/whatap/go-api/agent/agent/counter"
	"github.com/whatap/go-api/agent/agent/countertag"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/agent/pprof"
	"github.com/whatap/go-api/agent/agent/secure"
	"github.com/whatap/go-api/agent/agent/trace"
	logsinkwatch "github.com/whatap/go-api/agent/logsink/watch"
	"github.com/whatap/go-api/agent/net"

	// "github.com/whatap/go-api/agent/thirdparty"
	"github.com/whatap/go-api/agent/util/logutil"
	whatapsys "github.com/whatap/go-api/agent/util/sys"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/openstack"
	//	"fmt"
)

var (
	AGENT_VERSION string = "0.4.3"
	BUILDNO       string = "20241016"
)

func Boot() {
	// 종료 되지 않도록 Boot 에서 Recover
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA001", " Recover", r, "stack=", string(debug.Stack()))
		}
	}()
	conf := config.GetConfig()
	logutil.GetLogger()
	if conf.Shutdown {
		logutil.Infoln("WA001-01", " Shutdown. config.shutdown is true, don't start agent")
		return
	}

	trace.StartProfileSender()
	net.StartNet()

	//extension.StartUdp()

	sendAgentInfo()
	// 서버에서 패킷 수신 및 처리
	control.InitControlHandler()
	counter.StartCounterManager()

	// Tag Counter
	countertag.StartTagCounterManager()

	//trace.StartSimula()
	pprof.GetSlefPProf()

	// thirdparty.StartAll()

	logsinkwatch.GetInstance()
}

func sendAgentInfo() {
	secu := secure.GetSecurityMaster()
	p := pack.NewParamPack()
	p.Pcode = secure.GetSecurityMaster().PCODE
	p.Oid = secure.GetSecurityMaster().OID
	conf := config.GetConfig()
	p.Okind = conf.OKIND
	p.Onode = conf.ONODE
	p.Time = dateutil.Now()
	p.Id = net.AGENT_BOOT_ENV

	p.PutString("whatap.version", os.Getenv("WHATAP_VERSION"))
	p.PutString("whatap.agent_version", AGENT_VERSION)
	p.PutString("whatap.agent_buildno", BUILDNO)
	p.PutString("whatap.home", config.GetWhatapHome())

	os.Setenv("whatap.starttime", strconv.Itoa(int(dateutil.Now())))
	p.PutString("whatap.starttime", os.Getenv("whatap.starttime"))

	p.PutString("whatap.oname", secu.ONAME)
	p.PutString("whatap.name", os.Getenv("whatap.name"))
	p.PutString("whatap.ip", os.Getenv("whatap.ip"))
	p.PutString("whatap.port", os.Getenv("whatap.port"))
	p.PutString("whatap.hostname", os.Getenv("whatap.hostname"))
	p.PutString("whatap.type", os.Getenv("whatap.type"))
	p.PutString("whatap.process", os.Getenv("whatap.process"))

	p.PutString("whatap.pid", strconv.Itoa(whatapsys.GetPid()))

	p.PutString("os.arch", runtime.GOARCH)
	p.PutString("os.name", runtime.GOOS)
	p.PutString("os.cpucore", strconv.Itoa(whatapsys.GetCPUNum()))
	cpuType, _ := whatapsys.GetCPUType()
	p.PutString("os.cpuvendor", cpuType)
	memorySize, _ := whatapsys.GetMemorySize()
	p.PutString("os.memory", strconv.FormatInt(memorySize, 10))
	logutil.Infoln("WA001", " Agent boot info\n", p.ToString())

	if openstack.IsKIC() {
		os.Setenv("CLOUD_PLATFORM", "kic")
		p.PutString("CLOUD_PLATFORM", "kic")
	}

	data.SendBoot(p)

	counter.CounterStaticAgentBootInfo = p
}
