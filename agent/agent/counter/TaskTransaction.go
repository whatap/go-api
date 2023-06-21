package counter

import (
	//"log"
	"runtime/debug"

	"github.com/whatap/golib/lang"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/ref"
	"github.com/whatap/golib/util/hmap"
	"github.com/whatap/golib/util/mathutil"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/counter/meter"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/agent/trace"
	"github.com/whatap/go-api/agent/util/logutil"
)

type TaskTransaction struct {
	avg30 *hmap.LongKeyLinkedMap
}

func NewTaskTransaction() *TaskTransaction {
	p := new(TaskTransaction)
	p.avg30 = hmap.NewLongKeyLinkedMapDefault().SetMax(6)

	return p
}

type Item struct {
	tps   float64
	rtime float64
}

func (this *TaskTransaction) process(p *pack.CounterPack1) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA391", "process Recover", r, string(debug.Stack()))
		}
	}()
	conf := config.GetConfig()
	if !conf.CounterEnabledTranx_ {
		if conf.CounterLogEnabled {
			logutil.Println("WA391-01", "Disable counter, tran x")
		}
		return
	} else {
		if conf.CounterLogEnabled {
			logutil.Println("WA391-02", "Start counter, tran x")
		}
	}
	meterService := meter.GetInstanceMeterService()

	//bk := meterService.GetBucket()
	bk := meterService.GetAndResetBucket()

	p.ServiceCount = bk.Count
	if bk.Count > 0 {
		p.ServiceTime = bk.Timesum
		p.ServiceError = bk.Error

		p.ApdexSatisfied = bk.CountSatisfied
		p.ApdexTolerated = bk.CountTolerated
		p.ApdexTotal = bk.CountTotal

		p.TxDbcTime = float32(bk.DbcSum) / float32(bk.Count)
		p.TxSqlTime = float32(bk.SqlSum) / float32(bk.Count)
		p.TxHttpcTime = float32(bk.HttpcSum) / float32(bk.Count)
	}

	// DEBUG TOPOL
	if conf.TxCallerMeterEnabled {
		this.txCallerMeter(p)
	}
	if conf.ActxMeterEnabled {
		this.txCallerActxMeter(p)
	}
	var arrival int
	if p.CollectIntervalMs > 0 {
		arrival = meterService.Arrival
		meterService.Arrival = 0
		p.ArrivalRate = float32(arrival) * 1000 / float32(p.CollectIntervalMs)

		if conf.TpsAvg30Enabled == false {
			if bk.Count <= 0 {
				p.Tps = 0
				p.RespTime = 0
			} else {
				p.Tps = float32(float64(bk.Count) * 1000 / float64(p.CollectIntervalMs))
				p.RespTime = int32(float64(bk.Timesum) / float64(bk.Count))
			}
		} else {
			//logutil.Infoln(">>>>", "ArrivalRate=", p.ArrivalRate, ",duration=", p.Duration, ", arrival=", arrival)
			if bk.Count <= 0 {
				this.avg30.Put(p.Time, Item{tps: 0, rtime: 0})
			} else {
				t := float64(bk.Count) * 1000 / float64(p.CollectIntervalMs)
				r := float64(bk.Timesum) / float64(bk.Count)
				this.avg30.Put(p.Time, Item{tps: t, rtime: r})
			}
			this.calc(p, p.Time-30000)
		}
	}
	if conf.DebugTxCounterEnabled {
		logutil.Printf("WA391-03", "No Counter service_cnt=%d, service_resp=%d, apdex=%d, %d, %d, arrival=%d, %f, tps=%f, resp=%d", p.ServiceCount, p.ServiceTime, p.ApdexSatisfied, p.ApdexTolerated, p.ApdexTotal, arrival, p.ArrivalRate, p.Tps, p.RespTime)
	}

	// 이벤트 알림
	if conf.HitMapHorizEventEnabled {
		ep := bk.ProcessHorizontalEvent()
		if ep != nil {
			data.SendEvent(ep)
		}
	}

	// 이벤트 알림
	if conf.HitMapVerEventEnabled {
		ep := bk.ProcessVerticalEvent()
		if ep != nil {
			data.SendEvent(ep)
		}
	}

	hp := bk.Hitmap

	// BSM AppType - BSM은 hitmap 전송 안함
	if conf.AppType == lang.APP_TYPE_PHP || conf.AppType == lang.APP_TYPE_PYTHON ||
		conf.AppType == lang.APP_TYPE_DOTNET || conf.AppType == lang.APP_TYPE_GO {
		//HITMAP 전송함..
		hp.Time = p.Time
		hp.Pcode = p.Pcode
		hp.Oid = p.Oid
		hp.Okind = conf.OKIND
		hp.Onode = conf.ONODE

		data.Send(hp)
	}

	stdDev := mathutil.GetStandardDeviation(int(bk.Count), float64(bk.Timesum), float64(bk.TimeSqrSum))
	p.Resp90 = int32(mathutil.GetPct90(float64(p.RespTime), stdDev))
	p.Resp95 = int32(mathutil.GetPct95(float64(p.RespTime), stdDev))
	p.TimeSqrSum = bk.TimeSqrSum

	//logutil.Infoln(">>>>", "mean=", p.RespTime, ",count=", bk.Count, ",timesum=", bk.Timesum, ",timesqrsum=", bk.TimeSqrSum, ",min=", bk.TimeMin, ",max=", bk.TimeMax, ",stdDev=", stdDev, ",resp90=", p.Resp90, ",resp95=", p.Resp95)

	bk.Reset()

}
func (this *TaskTransaction) calc(p *pack.CounterPack1, tm int64) {
	var t, r float64 = 0, 0
	var c int = 0
	en := this.avg30.Entries()
	for en.HasMoreElements() {
		e := en.NextElement().(*hmap.LongKeyLinkedEntry)
		if e.GetKey() >= tm {
			c++
			v := e.GetValue().(Item)
			t += v.tps
			r += v.rtime
		}
	}
	if c > 0 {
		p.Tps = float32(t / float64(c))
		p.RespTime = int32(r / float64(c))
	}
}

func (this *TaskTransaction) txCallerMeter(p *pack.CounterPack1) {
	meterService := meter.GetInstanceMeterService()
	//logutil.Infoln("TaskTransaction", "txCallerMeter size=", meterService.StatByOID.Size())
	if meterService.StatByOID.Size() > 0 {
		p.TxcallerOidMeter = hmap.NewIntKeyLinkedMapDefault()
		en := meterService.StatByOID.Keys()
		for i := 0; i < 300 && en.HasMoreElements(); i++ {
			key := en.NextInt()
			b := meterService.StatByOID.Get(key).(*meter.BucketSimple)
			m := pack.NewTxMeter()

			m.Time = b.Timesum
			m.Count = b.Count
			m.Error = b.Error
			p.TxcallerOidMeter.Put(key, m)
		}
	}

	//logutil.Infoln("TaskTransaction", "txCallerMeter size=", meterService.StatByPKIND.Size())
	if meterService.StatByPKIND.Size() > 0 {
		p.TxcallerGroupMeter = hmap.NewLinkedMapDefault()
		en := meterService.StatByPKIND.Keys()
		for i := 0; i < 300 && en.HasMoreElements(); i++ {
			key := en.NextElement().(*lang.PKIND)
			b := meterService.StatByPKIND.Get(key).(*meter.BucketSimple)
			m := pack.NewTxMeter()

			m.Time = b.Timesum
			m.Count = b.Count
			m.Error = b.Error
			p.TxcallerGroupMeter.Put(key, m)
		}
	}

	if meterService.StatByPOID.Size() > 0 {
		p.TxcallerPOidMeter = hmap.NewLinkedMapDefault()
		en := meterService.StatByPOID.Keys()
		for i := 0; i < 300 && en.HasMoreElements(); i++ {
			key := en.NextElement().(*lang.POID)
			b := meterService.StatByPOID.Get(key).(*meter.BucketSimple)
			m := pack.NewTxMeter()

			m.Time = b.Timesum
			m.Count = b.Count
			m.Error = b.Error
			p.TxcallerPOidMeter.Put(key, m)
		}
	}

	p.TxcallerUnknown = pack.NewTxMeter()
	p.TxcallerUnknown.Count = meterService.Unknown.Count
	p.TxcallerUnknown.Error = meterService.Unknown.Error
	p.TxcallerUnknown.Time = meterService.Unknown.Timesum

	meterService.ResetStat()
	//TraceContext.updatePOID()
	trace.UpdatePOID()
}

func (this *TaskTransaction) txCallerActxMeter(p *pack.CounterPack1) {
	meterAct := meter.GetInstanceMeterActiveX()
	if meterAct.StatByOid == nil {
		return
	}

	if meterAct.StatByOid.Size() > 0 {
		if p.TxcallerOidMeter == nil {
			p.TxcallerOidMeter = hmap.NewIntKeyLinkedMapDefault()
		}

		en := meterAct.StatByOid.Keys()
		for i := 0; i < 300 && en.HasMoreElements(); i++ {
			key := en.NextInt()
			var m *pack.TxMeter
			o := p.TxcallerOidMeter.Get(key)
			if o == nil {
				m = pack.NewTxMeter()
				p.TxcallerOidMeter.Put(key, m)
			} else {
				m = o.(*pack.TxMeter)
			}
			m.Actx = meterAct.StatByOid.Get(key)
		}
	}

	if meterAct.StatByGroup.Size() > 0 {
		if p.TxcallerGroupMeter == nil {
			p.TxcallerGroupMeter = hmap.NewLinkedMapDefault()
		}

		en := meterAct.StatByGroup.Keys()
		for i := 0; i < 300 && en.HasMoreElements(); i++ {
			key := en.NextElement().(*lang.PKIND)
			var m *pack.TxMeter
			o := p.TxcallerGroupMeter.Get(key)
			if o == nil {
				m = pack.NewTxMeter()
				p.TxcallerGroupMeter.Put(key, m)
			} else {
				m = o.(*pack.TxMeter)
			}
			num := meterAct.StatByGroup.Get(key)
			if num != nil {
				m.Actx = int32(num.(*ref.INT).Value)
			}
		}
	}

	if meterAct.StatByPOid.Size() > 0 {
		if p.TxcallerGroupMeter == nil {
			p.TxcallerPOidMeter = hmap.NewLinkedMapDefault()
		}

		en := meterAct.StatByPOid.Keys()
		for i := 0; i < 300 && en.HasMoreElements(); i++ {
			key := en.NextElement().(*lang.POID)
			var m *pack.TxMeter
			o := p.TxcallerPOidMeter.Get(key)
			if o == nil {
				m = pack.NewTxMeter()
				p.TxcallerPOidMeter.Put(key, m)
			} else {
				m = o.(*pack.TxMeter)
			}
			num := meterAct.StatByPOid.Get(key)
			if num != nil {
				m.Actx = int32(num.(*ref.INT).Value)
			}
		}
	}

	if p.TxcallerUnknown == nil {
		p.TxcallerUnknown = pack.NewTxMeter()
	}

	p.TxcallerUnknown.Actx = int32(meterAct.Unknown)
}
