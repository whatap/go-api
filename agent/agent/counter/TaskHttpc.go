package counter

import (
	//"log"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/hmap"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/counter/meter"
	"github.com/whatap/go-api/agent/util/logutil"
)

type TaskHttpc struct {
	stime int64
}

func NewTaskHttpc() *TaskHttpc {
	p := new(TaskHttpc)
	return p
}

func (this *TaskHttpc) process(p *pack.CounterPack1) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA341", "process Recover", r)
		}
	}()
	conf := config.GetConfig()
	if !conf.CounterEnabledHttpc_ {
		if conf.CounterLogEnabled {
			logutil.Println("WA341-01", "Disable counter, httpc")
		}
		return
	} else {
		if conf.CounterLogEnabled {
			logutil.Println("WA341-02", "Start counter, httpc")
		}

	}

	// DEBUG METER
	mHttpc := meter.GetInstanceMeterHTTPC().GetBucketReset()

	p.HttpcTime = mHttpc.Time
	p.HttpcCount = mHttpc.Count
	p.HttpcError = mHttpc.Error

	now := dateutil.SystemNow()
	dtime := now - this.stime
	if dtime == 0 {
		return
	}
	this.stime = now

	if config.GetConfig().HttpcHostMeterEnabled {
		this.httpcMeter(p)
	}
	if config.GetConfig().ActxMeterEnabled {
		this.httpcActxMeter(p)
	}

}

func (this *TaskHttpc) httpcMeter(p *pack.CounterPack1) {
	meterHTTPC := meter.GetInstanceMeterHTTPC()
	//logutil.Infoln("TaskHttpc", "httpcMeter size=", meterHTTPC.Stat.Size())
	if meterHTTPC.Stat.Size() > 0 {
		p.HttpcMeter = hmap.NewIntKeyLinkedMapDefault()
		en := meterHTTPC.Stat.Keys()

		for i := 0; i < 100 && en.HasMoreElements(); i++ {
			key := en.NextInt()
			b := meterHTTPC.Stat.Get(key).(*meter.HTTPCBucket)
			m := pack.NewHttpcMeter()
			m.Time = b.Time
			m.Count = b.Count
			m.Error = b.Error
			p.HttpcMeter.Put(key, m)
		}

		meterHTTPC.ResetStat()
	}
	//TraceH
}

func (this *TaskHttpc) httpcActxMeter(p *pack.CounterPack1) {

	meterActiveX := meter.GetInstanceMeterActiveX()
	if meterActiveX.StatHttpc == nil {
		return
	}
	//logutil.Infoln("TaskHttpc", "httpcActxMeter size=", meterActiveX.StatHttpc.Size())
	if meterActiveX.StatHttpc.Size() > 0 {
		if p.HttpcMeter == nil {
			p.HttpcMeter = hmap.NewIntKeyLinkedMapDefault()
		}
		en := meterActiveX.StatHttpc.Keys()
		for i := 0; i < 300 && en.HasMoreElements(); i++ {
			key := en.NextInt()
			var m *pack.HttpcMeter
			o := p.HttpcMeter.Get(key)
			if o == nil {
				m = pack.NewHttpcMeter()
				p.HttpcMeter.Put(key, m)
			} else {
				m = o.(*pack.HttpcMeter)
			}
			m.Actx = meterActiveX.StatHttpc.Get(key)
		}
	}
}
