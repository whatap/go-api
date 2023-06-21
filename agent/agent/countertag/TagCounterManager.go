package countertag

import (
	//"log"
	"time"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/secure"
	"github.com/whatap/golib/lang"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/dateutil"
)

func StartTagCounterManager() {
	// telegraf
	//telegraf := NewTagTaskTelegraf()

	tasks := []Task{}

	secu := secure.GetSecurityMaster()
	conf := config.GetConfig()

	if conf.AppType == lang.APP_TYPE_GO {
		tasks = append(tasks, NewTagTaskGoRuntime())
	}

	var INTERVAL int32 = conf.TagCountInterval
	if INTERVAL < 5000 {
		INTERVAL = 5000
	}

	go func() {
		for {
			sleepx(int64(INTERVAL))
			now := dateutil.Now() / int64(INTERVAL) * int64(INTERVAL)

			if secu.PCODE == 0 || secu.OID == 0 {
				continue
			}
			p := pack.NewTagCountPack()
			p.Pcode = secu.PCODE
			p.Oid = secu.OID
			p.Okind = conf.OKIND
			p.Onode = conf.ONODE
			p.Time = now

			if conf.TagCounterEnabled {
				for i := 0; i < len(tasks); i++ {
					tasks[i].process(p)
				}
			}
			//data.SendHide(p)
		}
	}()
}

func sleepx(interval int64) {
	stime := dateutil.Now() / interval * interval
	time.Sleep(3000 * time.Millisecond)
	for {
		now := ((dateutil.Now() / interval) * interval)
		if stime != ((now / interval) * interval) {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}
