package counter

import (
	//"log"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/kube"
	"github.com/whatap/go-api/agent/agent/secure"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/go-api/agent/util/sys"
)

type TaskSystemPerfKube struct {
	lastMetering float32
}

func NewTaskSystemPerfKube() *TaskSystemPerfKube {
	p := new(TaskSystemPerfKube)
	return p
}

func (this *TaskSystemPerfKube) process(p *pack.CounterPack1) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA382", "process Recover", r)
		}
	}()

	conf := config.GetConfig()
	if !conf.CounterEnabledSysPerfKube_ {
		if conf.CounterLogEnabled {
			logutil.Println("WA382-01", "Disable counter, system perf kube")
		}
		return
	} else {
		if conf.CounterLogEnabled {
			logutil.Println("WA382-02", "Start counter, system perf kube")
		}
	}

	now := dateutil.Now()
	if now < kube.NodeRecvTime+10000 {
		p.Cpu = kube.Cpu * conf.CorrectionFactorCpu
		p.Mem = float32(kube.Memory) * conf.CorrectionFactorPCpu
		if kube.Metering == 0 {
			p.Metering = float32(p.CpuCores)
		} else {
			p.Metering = kube.Metering
		}

	}
	if p.Metering != 0 {
		this.lastMetering = p.Metering
	} else {
		p.Metering = this.lastMetering
	}

	// CPU CORE
	p.CpuCores = int32(sys.GetCPUNum())
	p.HostIp = secure.GetSecurityMaster().IP
}

func (this *TaskSystemPerfKube) clear(p *pack.CounterPack1) {
	p.Cpu = 0
	p.CpuSys = 0
	p.CpuUsr = 0
	p.CpuWait = 0
	p.CpuSteal = 0
	p.CpuIrq = 0
	p.CpuProc = 0
	p.Disk = 0
	p.Mem = 0
	p.Swap = 0
	// TODO
	//p.Netstat = nil
	p.ProcFd = 0
}
