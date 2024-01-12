package counter

import (
	//"log"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/agent/kube"
	"github.com/whatap/go-api/agent/agent/secure"
	"github.com/whatap/go-api/agent/net"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/dateutil"
)

var CounterStaticAgentBootInfo *pack.ParamPack

type TaskAgentInfo struct {
	first          bool
	nextTime       int64
	lastNameSent   int64
	firstConnected bool
}

func NewTaskAgentInfo() *TaskAgentInfo {
	p := new(TaskAgentInfo)
	p.nextTime = dateutil.SystemNow() + dateutil.MILLIS_PER_FIVE_MINUTE
	p.firstConnected = true
	return p
}

func (this *TaskAgentInfo) process(p *pack.CounterPack1) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA321", "process Recover", r)
		}
	}()
	conf := config.GetConfig()
	if !conf.CounterEnabledAgentInfo_ {
		if conf.CounterLogEnabled {
			logutil.Println("WA321-01", "Disable counter, agent info")
		}
		return
	} else {
		if conf.CounterLogEnabled {
			logutil.Println("WA321-02", "Start counter, agent info")
		}
	}

	now := dateutil.SystemNow()
	// now := getDate()
	if now >= this.nextTime {
		// 12시간 발송 -> 1시간 정시 근처 발송.
		this.nextTime = now/dateutil.MILLIS_PER_HOUR*dateutil.MILLIS_PER_HOUR + dateutil.MILLIS_PER_HOUR

		CounterStaticAgentBootInfo.Id = net.AGENT_BOOT_ENV
		CounterStaticAgentBootInfo.Time = now
		data.Send(CounterStaticAgentBootInfo)
		logutil.Infoln("WA321-03", "CounterStaticAgentBootInfo", CounterStaticAgentBootInfo.ToString())

		// Java
		//		ComponentsVersions.send();
	}
	// Java
	// 세션이 연결된지 5초가 지났다는 것은 방화벽이 열려 있다는 것을 의미한다.
	// 방화벽이 막혀있는 동안 수집되었던 METHOD 이름 정보를 서버로 전송한다.
	//	if net.GetTcpSession().LastConnectedTime > 0 && now > net.GetTcpSession().LastConnectedTime+5000 {
	//		if first_connected {
	//			first_connected = false
	//			// 최초 방화벽이 열렸을때 모든 텍스트 전송 기록을 삭제함으로
	//			// 모든 택스트가 다시 전송되도록 한다.
	//			data.ResetHash()
	//
	//			// Java
	//			//			DataTextAgent.sendMethodAfterBoot();
	//		}
	//	}

	// Java
	kube.GetContainerInfo(func(containerKey int32, containerId string) {
		p.ContainerKey = containerKey
		data.SendText(pack.CONTAINER, containerId)
	})

	//	checkAutoScaleIn()

	this.sendName(now)

}

func (this *TaskAgentInfo) sendName(now int64) {
	if now-this.lastNameSent < dateutil.MILLIS_PER_FIVE_MINUTE {
		return
	}
	this.lastNameSent = now

	conf := config.GetConfig()
	secu := secure.GetSecurityMaster()
	if len(secu.ONAME) > 0 {
		data.AddHashText(pack.TEXT_ONAME, secu.OID, secu.ONAME)
	}

	if conf.OKIND != 0 {
		data.AddHashText(pack.TEXT_OKIND, conf.OKIND, conf.OKIND_NAME)
	}

	if conf.ONODE != 0 {
		data.AddHashText(pack.TEXT_ONAME, conf.ONODE, conf.ONODE_NAME)
	}

	// Java
	//		if (KubeUtil.container_key != 0) {
	//			data.AddHashText(pack.CONTAINER_ID, KubeUtil.container_key, KubeUtil.container_id);
	//		}

	if len(conf.MtraceSpec) > 0 {
		data.AddHashText(pack.TEXT_MTRACE_SPEC, conf.MtraceSpecHash, conf.MtraceSpec)
	}

	//logutil.Infoln("WA321-04", "AgentInfo sendName")
}

func getDate() int64 {
	return dateutil.Now() / dateutil.MILLIS_PER_HOUR
}
