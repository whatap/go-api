package countertag

import (
	//"log"
	"fmt"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/counter/meter"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/golib/lang/pack"
	// "github.com/whatap/golib/util/dateutil"
)

type TagTaskTxStatus struct {
	nextTime int64
}

func NewTagTaskTxStatus() *TagTaskTxStatus {
	p := new(TagTaskTxStatus)
	return p
}

func (this *TagTaskTxStatus) process(p *pack.TagCountPack) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA140301", " Task Telegraf Process Recover ", r)
		}
	}()

	conf := config.GetConfig()
	if conf.TxStatusMeterEnabled {
		// if this.nextTime == 0 || p.Time >= this.nextTime {

		p.Category = "tx_status"
		meterStatus := meter.GetInstanceMeterStatus()

		b200, b400, b500 := meterStatus.GetBucketReset()

		p.Put("200_total_count", b200.Count)
		p.Put("200_total_time", b200.Time)

		p.Put("400_total_count", b400.Count)
		p.Put("400_total_time", b400.Time)

		p.Put("500_total_count", b500.Count)
		p.Put("500_total_time", b500.Time)

		stat := meterStatus.Stat
		meterStatus.ResetStat()

		en := stat.Keys()
		for en.HasMoreElements() {
			k := en.NextInt()
			if tmp := stat.Get(k); tmp != nil {
				if v, ok := tmp.(*meter.StatusBucket); ok {
					p.Put(fmt.Sprintf("%d_count", k), v.Count)
					p.Put(fmt.Sprintf("%d_time", k), v.Time)
				}
			}
		}

		data.Send(p)

		// now := dateutil.Now() / dateutil.MILLIS_PER_FIVE_MINUTE * dateutil.MILLIS_PER_FIVE_MINUTE
		// this.nextTime = now + dateutil.MILLIS_PER_FIVE_MINUTE
		// }
	}
}
