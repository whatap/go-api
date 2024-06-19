package counter

import (
	//"log"
	"time"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/counter/meter"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/agent/kube"
	"github.com/whatap/go-api/agent/agent/secure"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/golib/lang"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/dateutil"
)

func StartCounterManager() {
	secu := secure.GetSecurityMaster()
	conf := config.GetConfig()

	//	// tasks.add(new TopService());// 액티브 서비스 보다 먼저 처리되어야함
	//		tasks.add(new Service());
	//		tasks.add(new ActiveTranCount());// 액티브를 가장 먼저 처리함
	//		tasks.add(new RealtimeUser());
	//		tasks.add(new GCStat());
	//		tasks.add(new HeapMem());
	//		tasks.add(new HttpC());
	//		tasks.add(new ProcCpu());
	//		tasks.add(new Sql());
	//		tasks.add(new SystemPerf());
	//		tasks.add(new ThreadStat());
	//		tasks.add(new SelfMonitor());
	//		tasks.add(new ExtentionHelper());
	//		tasks.add(new AgentInfo());
	//		tasks.add(new DataSourceCount());

	// Service
	tranx := NewTaskTransaction()
	// Active
	act := NewTaskActiveTranCount()
	// RealtimeUser
	user := NewTaskRealtimeUser()
	// HttpC
	httpc := NewTaskHttpc()
	// Sql
	sql := NewTaskSql()

	agentInfo := NewTaskAgentInfo()

	heap := NewTaskHeapMem()
	proc := NewTaskProc()

	packVersion := NewTaskPackVersion()

	tasks := []Task{
		tranx,
		act,
		user,
		httpc,
		sql,
		agentInfo,
		proc,
		heap,
	}
	// counter pack version 설정, 버전이 0일 경우 메모리 /1024 처리
	tasks = append(tasks, packVersion)

	if conf.WhatapMicroEnabled {
		tasks = append(tasks, NewTaskSystemPerfKube())
	} else {
		// TODO 추후 siger.CPU, siger.DISK 관련 처리 후 열기
		tasks = append(tasks, NewTaskSystemPerf())
	}

	if conf.AppType == lang.APP_TYPE_GO {
		//tasks = append(tasks, NewTaskActiveStatsForPython())
	}

	var INTERVAL int32 = conf.CountInterval
	if INTERVAL < 5000 {
		INTERVAL = 5000
	}

	if conf.WhatapMicroEnabled {
		kube.StartClient()
	}

	go func() {
		lastSysTime := dateutil.SystemNow()
		next := (dateutil.Now() / int64(INTERVAL) * int64(INTERVAL)) + int64(INTERVAL)
		for {
			// shutdown
			if config.GetConfig().Shutdown {
				logutil.Infoln("WA211-03", "Shutdown CounterManager and clear meter")
				meter.GetInstanceConnPool().Clear()
				meter.GetInstanceMeterActiveX().Clear()
				meter.GetInstanceMeterHTTPC().Clear()
				meter.GetInstanceMeterSelf().Clear()
				meter.GetInstanceMeterService().Clear()
				meter.GetInstanceMeterSQL().Clear()
				meter.GetInstanceMeterUsers().Clear()
				break
			}

			if conf.DebugCounterEnabled {
				logutil.Infoln("[DEBUG]", "CounterManger go pcode=", secu.PCODE, ",oid=", secu.OID)
			}

			sleepx(next, int64(INTERVAL))
			now := dateutil.Now() / int64(INTERVAL) * int64(INTERVAL)
			next = now + int64(INTERVAL)

			if secu.PCODE == 0 || secu.OID == 0 {
				continue
			}
			p := pack.NewCounterPack1()
			p.Pcode = secu.PCODE
			p.Oid = secu.OID
			p.Okind = conf.OKIND
			p.Onode = conf.ONODE
			p.Time = now
			p.ApType = conf.AppType
			//p.Duration = 5
			p.CollectIntervalMs = int(dateutil.SystemNow() - lastSysTime)
			p.Duration = int32((p.CollectIntervalMs + 499) / 1000)
			lastSysTime = dateutil.SystemNow()

			if p.Duration <= 0 {
				if conf.DebugCounterEnabled {
					logutil.Infoln("[DEBUG]", "duration continue ", p.Duration)
				}
				continue
			}
			if conf.CounterEnabled {
				if conf.CounterTimeout > 0 {
					endChan := make(chan bool, 1)
					go ExecuteTasksTimeout(p, tasks, endChan)
					select {
					case <-endChan:
					case <-time.After(time.Duration(conf.CounterTimeout) * time.Millisecond):
						logutil.Println("WA300", "Counter timeout conf=", int(conf.CounterTimeout), ", t=", p.Time)
					}
					close(endChan)
				} else {
					ExecuteTasks(p, tasks)
				}
			}

			if conf.DebugCounterEnabled {
				endTime := dateutil.SystemNow()
				logutil.Infoln("[DEBUG]", "Send CounterManager t=", p.Time, ", d=", p.Duration, ", e=", (endTime - lastSysTime))
			}
			data.Send(p)

			secure.GetParamSecurity().Reload()
		}
	}()
}

func ExecuteTasksTimeout(p *pack.CounterPack1, tasks []Task, ch chan bool) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA300-01", "ExecuteTasks t=", p.Time, ", Recover ", r)
		}
	}()
	ExecuteTasks(p, tasks)
	ch <- true
}

func ExecuteTasks(p *pack.CounterPack1, tasks []Task) {
	conf := config.GetConfig()

	// BSM APP Type
	if conf.AppType == lang.APP_TYPE_BSM_PYTHON || conf.AppType == lang.APP_TYPE_BSM_PHP || conf.AppType == lang.APP_TYPE_BSM_DOTNET {
		//logutil.Infoln("BSM-001", lang.APP_TYPE_BSM)
		p.ApType = lang.APP_TYPE_BSM
	} else {
		p.ApType = conf.AppType
	}

	for i := 0; i < len(tasks); i++ {
		tasks[i].process(p)
	}
}

type CounterManager struct {
	tasks []Task
}

func NewCounterManager() *CounterManager {
	counterManager := new(CounterManager)

	// Service
	tranx := NewTaskTransaction()
	// Active
	act := NewTaskActiveTranCount()
	// RealtimeUser
	user := NewTaskRealtimeUser()
	// HttpC
	httpc := NewTaskHttpc()

	// Sql
	sql := NewTaskSql()

	oinfo := NewTaskExtraINFO()
	//counter := &TaskCounter{}

	heap := NewTaskHeapMem()
	proc := NewTaskProc()

	// TODO 추후 siger.CPU, siger.DISK 관련 처리 후 열기
	systemPerf := NewTaskSystemPerf()

	tasks := []Task{
		tranx,
		act,
		user,
		httpc,
		proc,
		heap,
		// counter,
		sql,
		systemPerf,
		oinfo}

	counterManager.tasks = tasks

	return counterManager

}
func sleepx(next, interval int64) {
	stime := dateutil.Now() / interval * interval
	sleepTime := (next - dateutil.Now()) / 1000 * 1000
	//	logutil.Infoln("[DEBUG]", "now=", dateutil.Now(), ",stime=", stime, ",next=", next, ",sleepTime=", sleepTime)
	if sleepTime < 0 {
		sleepTime = 0
	} else if sleepTime > 3000 {
		sleepTime = 3000
	}
	if sleepTime > 0 {
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	}

	for {
		now := ((dateutil.Now() / interval) * interval)
		if stime != ((now / interval) * interval) {
			//			logutil.Infoln("[DEBUG]", "return now=", ((now / interval) * interval), ",stime=", stime)
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// func (this *CounterManager) Poll() {
// 	secu := secure.GetSecurityMaster()
// 	conf := config.GetConfig()

// 	var INTERVAL int32 = conf.CountInterval
// 	if INTERVAL < 5000 {
// 		INTERVAL = 5000
// 	}
// 	now := dateutil.Now() / int64(INTERVAL) * int64(INTERVAL)

// 	if secu.PCODE == 0 || secu.OID == 0 {
// 		return
// 	}
// 	p := pack.NewCounterPack1()
// 	p.Pcode = secu.PCODE
// 	p.Oid = secu.OID
// 	p.Okind = conf.OKIND
// 	p.Onode = conf.ONODE
// 	p.Time = now
// 	p.Duration = 5
// 	// BSM APP Type
// 	if conf.AppType == lang.APP_TYPE_BSM_PYTHON || config.AppType == lang.APP_TYPE_BSM_PHP {
// 		p.ApType = lang.APP_TYPE_BSM
// 	} else {
// 		p.ApType = conf.AppType
// 	}

// 	if conf.CounterEnabled {
// 		for i := 0; i < len(this.tasks); i++ {
// 			this.tasks[i].process(p)
// 		}
// 	}
// 	data.Send(p)

// 	secure.GetParamSecurity().Reload()
// }
