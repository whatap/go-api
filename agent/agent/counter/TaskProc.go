package counter

import (
	//"log"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/counter/meter"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/go-api/agent/util/sys"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/dateutil"
)

type TaskProc struct {
	oldCpu  float64
	oldTime int64

	oldPsutilCpuTime float64
	oldPsutilTime    int64

	oldCpuTime map[int32]float64
}

func NewTaskProc() *TaskProc {
	p := new(TaskProc)
	p.oldCpuTime = make(map[int32]float64, 0)
	return p
}

func (this *TaskProc) process(p *pack.CounterPack1) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA351", "TaskProc Recover", r)
		}
	}()
	conf := config.GetConfig()
	if !conf.CounterEnabledProc_ {
		if conf.CounterLogEnabled {
			logutil.Println("WA351-01", "Disable counter, proc")
		}
		return
	} else {
		if conf.CounterLogEnabled {
			logutil.Println("WA351-02", "Start counter, proc")
		}

	}

	now := dateutil.SystemNow()

	procInfo, cpuTime := sys.GetProcessInfo(conf.AppProcessName)

	if conf.MeterSelfEnabled {
		meter.GetInstanceMeterSelf().AddMeterSelfCpu(float32(procInfo["self_cpu_proc"]))
		meter.GetInstanceMeterSelf().AddMeterSelfMem(int64(procInfo["self_heap_proc"] / 1024.0))
		//meter.GetInstanceMeterSelf().PrintSelfStat();
		//logutil.Infoln("self_cpu_proc=", float32(procInfo["self_cpu_proc"]), ",self_heap_proc=", int64(procInfo["self_heap_proc"]/1024.0))
	}

	p.CpuProc = float32(procInfo["cpu_proc"])
	// move TaskHeapMem (golang)
	//p.HeapUse = int64(procInfo["heap_proc"])

	if this.oldCpuTime == nil || len(this.oldCpuTime) <= 0 {
		this.oldCpuTime = cpuTime
		this.oldTime = now
		return
	}

	for k, v := range cpuTime {
		//if val, ok := this.oldCpuTime[k]; ok {
		val, ok := this.oldCpuTime[k]
		if ok {
			procInfo["cpu_time"] += v - val
		}
	}

	dTime := now - this.oldTime

	// 초당 cputime ms
	p.Cputime = int64(procInfo["cpu_time"] / float64(dTime/1000))

	this.oldCpuTime = cpuTime
	this.oldTime = now

}
