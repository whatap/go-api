package counter

import (
	//"log"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/counter/meter"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/util/logutil"
)

type TaskRealtimeUser struct {
}

func NewTaskRealtimeUser() *TaskRealtimeUser {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA362", "NewTaskRealtimeUser Recover", r)
		}
	}()

	p := new(TaskRealtimeUser)
	meter.GetRealtimeUsers()

	return p
}
func (this *TaskRealtimeUser) process(p *pack.CounterPack1) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA361", "process Recover", r)
		}
	}()
	conf := config.GetConfig()
	if config.GetConfig().RealtimeUserEnabled == false {
		if conf.CounterLogEnabled {
			logutil.Println("WA361-02", "Disable RealtimeUserEnabled")
		}
		return
	}
	if !conf.CounterEnabledUser_ {
		if conf.CounterLogEnabled {
			logutil.Println("WA361-01", "Disable counter, user")
		}
		return
	} else {
		if conf.CounterLogEnabled {
			logutil.Println("WA361-03", "Start counter, user")
		}

	}

	loglog := meter.GetRealtimeUsers()

	if loglog == nil {
		return
	}
	pk := pack.NewRealtimeUserPack()
	pk.Time = p.Time
	pk.Logbits = loglog.GetBytes()
	data.Send(pk)

	//logutil.Println("RealtimeUserPack time=", pk.Time, "logbits=", pk.Logbits)

}
