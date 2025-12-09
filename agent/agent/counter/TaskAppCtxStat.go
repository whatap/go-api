package counter

import (
	"github.com/whatap/go-api/agent/agent/appctx"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/golib/lang/pack"
)

type TaskAppCtxStat struct {
	collector *appctx.AppCtxStatCollector
}

func NewTaskAppCtxStat() *TaskAppCtxStat {
	return &TaskAppCtxStat{
		collector: appctx.GetIntanceAppCtxStatCollector(),
	}
}

func (t *TaskAppCtxStat) process(p *pack.CounterPack1) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA314-01", "TaskAppCtxStat.process Error", r)
		}
	}()

	conf := config.GetConfig()

	if !conf.AppContextEnabled {
		if conf.CounterLogEnabled {
			logutil.Println("WA314-02", "Disable AppContext")
		}
		return
	} else {
		if conf.CounterLogEnabled {
			logutil.Println("WA314-02", "Start AppContext")
		}
	}

	// 직접 process 호출 (workTime 전달)
	t.collector.Process(p.Time)

	if conf.CounterLogEnabled {
		logutil.Printf("WA314-03", "AppCtxStat processed at time=%d", p.Time)
	}
}
