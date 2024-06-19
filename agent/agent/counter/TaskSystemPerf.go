package counter

import (
	//"log"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/secure"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/go-api/agent/util/sys"
	"github.com/whatap/golib/lang"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/hash"
)

type TaskSystemPerf struct {
}

func NewTaskSystemPerf() *TaskSystemPerf {
	p := new(TaskSystemPerf)
	return p
}

func (this *TaskSystemPerf) process(p *pack.CounterPack1) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA381", "process Recover", r)
		}
	}()

	conf := config.GetConfig()
	if !conf.CounterEnabledSysPerf_ {
		if conf.CounterLogEnabled {
			logutil.Println("WA381-01", "Disable counter, system perf")
		}
		return
	} else {
		if conf.CounterLogEnabled {
			logutil.Println("WA381-02", "Start counter, system perf")
		}
	}
	//		if (sigar == null)
	//			return;

	cpuInfo := sys.GetCPUTimes()
	cpuPer := sys.GetCPUPercent()
	// Percent 계산
	cpuTot := cpuInfo.User + cpuInfo.System + cpuInfo.Nice + cpuInfo.Iowait + cpuInfo.Irq +
		cpuInfo.Softirq + cpuInfo.Steal + cpuInfo.Guest + cpuInfo.GuestNice + cpuInfo.Stolen + cpuInfo.Idle

	//	logutil.Infof(">>>>", "User=%f, Sys=%f, Nice=%f, Iowait=%f, Irq=%f, Softirq=%f, steal=%f, guest=%f, guestnice=%f, solen=%f, Idle=%f, Cpu=%f, total=%f, Used=%f",
	//		cpuInfo.User, cpuInfo.System, cpuInfo.Nice, cpuInfo.Iowait, cpuInfo.Irq,
	//		cpuInfo.Softirq, cpuInfo.Steal, cpuInfo.Guest, cpuInfo.GuestNice, cpuInfo.Stolen, cpuInfo.Idle, cpuTot, cpuInfo.Total, (1.0 - float32(cpuInfo.Idle/cpuTot)))

	// TODO , psutil 에서 나오는 TimeStat 을 Percent로 표시해서 넣어 줘야 함
	// java whatap.agent.tracer.sigar whatap.xtra.sigar.SigarMain
	//			vc.cpu = (float) ((1.0D - cpuPerc.getIdle()) * 100);
	//			vc.sys = (float) cpuPerc.getSys() * 100;
	//			vc.usr = (float) cpuPerc.getUser() * 100;
	//			vc.wait = (float) cpuPerc.getWait() * 100;
	//			vc.steal = (float) cpuPerc.getStolen() * 100;
	//			vc.irq = (float) cpuPerc.getSoftIrq();
	// CpuUsed
	//p.Cpu = float32((1.0 - float32(cpuInfo.Idle/cpuTot)) * 100)
	p.Cpu = float32(cpuPer.Percent)
	p.CpuSys = float32(float32(cpuInfo.System/cpuTot) * 100)
	p.CpuUsr = float32(float32(cpuInfo.User/cpuTot) * 100)
	p.CpuWait = float32(float32(cpuInfo.Iowait/cpuTot) * 100)
	p.CpuSteal = float32(float32(cpuInfo.Stolen/cpuTot) * 100)
	p.CpuIrq = float32(float32(cpuInfo.Irq / cpuTot))

	//logutil.Infof(">>>>", "Cpu=%f, Sys=%f, User=%f, Wait=%f, Steal=%f, Irq=%f", p.Cpu, p.CpuSys, p.CpuUsr, p.CpuWait, p.CpuSteal, p.CpuIrq)

	//	public float cpuProc(int pid) {
	//
	//		try {
	//			if (err_pcpu)
	//				return 0;
	//			int cpuCores = sigar.getCpuList().length;
	//			if (cpuCores > 0) {
	//				ProcCpu cpu = sigar.getProcCpu(pid);
	//				return (float) (cpu.getPercent() * 100.0D / cpuCores);
	//			}
	//
	//		} catch (Throwable t) {
	//			err_pcpu=true;
	//		}
	//		return 0;
	//	}

	// TODO
	// java sigar
	//	FileSystemUsage fileSystemUsage = sigar.getFileSystemUsage(dir);
	//			if (fileSystemUsage != null) {
	//				try {
	//					fileSystemUsage.gather(sigarImpl, dir);
	//				} catch (SigarException e) {
	//
	//				}
	//				long tot = fileSystemUsage.getTotal();
	//				long used = fileSystemUsage.getUsed();
	//				return tot == 0 ? 0f : (used * 100) / tot;
	if conf.AppType == lang.APP_TYPE_PHP || conf.AppType == lang.APP_TYPE_BSM_PHP {
		// PHP 는 "/" 경로의 Disk 정보
		p.Disk = float32(sys.GetSysDiskUsedPercent("/"))

	} else {
		// Python 은 WHATAP_HOME의 Disk 정보
		home := config.GetWhatapHome()
		p.Disk = float32(sys.GetSysDiskUsedPercent(home))
	}
	memInfo := sys.GetSysMemInfo()

	//	sigar.mem()
	//	Mem m = sigar.getMem();
	//	m.gather(sigarImpl);
	//	long tot = m.getTotal();
	//	long used = m.getActualUsed();
	//	return tot == 0 ? 0f : (used * 100) / tot;
	if memInfo.VirtualTotal > 0 {
		p.Mem = float32((memInfo.VirtualUsed * 100) / memInfo.VirtualTotal)
	} else {
		p.Mem = 0
	}

	//	sigar.swap()
	//	Swap m = sigar.getSwap();
	//	m.gather(sigarImpl);
	//	long tot = m.getTotal();
	//	long used = m.getUsed();
	//	return tot == 0 ? 0f : (used * 100) / tot;
	if memInfo.SwapTotal > 0 {
		p.Swap = float32((memInfo.SwapUsed * 100) / memInfo.SwapTotal)
	} else {
		p.Swap = 0
	}

	if conf.CounterNetstatEnabled {
		p.Netstat = &pack.NETSTAT{Est: 0}
	}
	if conf.CounterProcfdEnabled {
		p.ProcFd = 0
	}

	// CPU CORE
	//p.cpu_cores = Runtime.getRuntime().availableProcessors();
	p.CpuCores = int32(sys.GetCPUNum())

	// 2023.11.17 In linux, metering information is based on product uuid instead of client.LocalAddr.
	secu := secure.GetSecurityMaster()
	if conf.MeteringUseLinuxUUIDEnabled {
		if secu.MeterIP != "" {
			p.HostIp = hash.HashStr(secu.MeterIP)
		} else {
			p.HostIp = secu.IP
		}
	} else {
		p.HostIp = secu.IP
	}

	//fmt.Println("TaskSystemPref p.Cpu=", p.Cpu, ",p.CpuSys=" , p.CpuSys, ",p.CpuUsr=" , p.CpuUsr, ",p.CpuWait=" ,
	//	p.CpuWait, ",p.CpuSteal=" , p.CpuSteal, ",p.CpuIrq=" , p.CpuIrq, ",p.CpuProc=" , p.CpuProc, ",p.Disk=" ,
	//	p.Disk, ",p.Mem=" , p.Mem, ",p.Swap=" , p.Swap, ",p.CpuCores=", p.CpuCores)

}

func (this *TaskSystemPerf) clear(p *pack.CounterPack1) {
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
