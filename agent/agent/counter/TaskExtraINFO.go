package counter

import (
	//"log"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/agent/secure"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/dateutil"
)

type TaskExtraINFO struct {
	sent int64
}

func NewTaskExtraINFO() *TaskExtraINFO {
	p := new(TaskExtraINFO)
	return p
}

func (this *TaskExtraINFO) process(p *pack.CounterPack1) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA321", "process Recover", r)
		}
	}()
	conf := config.GetConfig()
	if !conf.CounterEnabledAgentInfo_ {
		if conf.CounterLogEnabled {
			logutil.Println("WA321-01", "Disable counter, extra info (oinfo)")
		}
		return
	} else {
		if conf.CounterLogEnabled {
			logutil.Println("WA321-02", "Start counter, extra info (oinfo)")
		}
	}

	if p.Time-this.sent < dateutil.MILLIS_PER_FIVE_MINUTE {
		return
	}
	this.sent = p.Time

	secu := secure.GetSecurityMaster()
	if len(secu.ONAME) > 0 {
		data.AddHashText(pack.TEXT_ONAME, secu.OID, secu.ONAME)
	}

	if conf.OKIND != 0 {
		data.AddHashText(pack.TEXT_OKIND, conf.OKIND, conf.OKIND_NAME)
	}

	if conf.ONODE != 0 {
		data.AddHashText(pack.ONODE_NAME, conf.ONODE, conf.ONODE_NAME)
	}

}
