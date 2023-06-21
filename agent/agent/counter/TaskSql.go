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

type TaskSql struct {
	stime int64
}

func NewTaskSql() *TaskSql {
	p := new(TaskSql)
	return p
}

func (this *TaskSql) process(p *pack.CounterPack1) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA371", "process Recover", r)
		}
	}()
	conf := config.GetConfig()
	if !conf.CounterEnabledSql_ {
		if conf.CounterLogEnabled {
			logutil.Println("WA371-01", "Disable counter, sql")
		}
		return
	} else {
		if conf.CounterLogEnabled {
			logutil.Println("WA371-02", "Start counter, sql")
		}
	}

	// DEBUG Meter
	//mSql := meter.GetSQLBucketReset()
	mSql := meter.GetInstanceMeterSQL().GetBucketReset()

	p.SqlTime = mSql.Time
	p.SqlCount = mSql.Count
	p.SqlError = mSql.Error

	p.SqlFetchCount = mSql.FetchCount
	p.SqlFetchTime = mSql.FetchTime

	now := dateutil.SystemNow()
	dtime := now - this.stime
	if dtime == 0 {
		return
	}
	this.stime = now

	if config.GetConfig().SqlDbcMeterEnabled {
		this.sqlMeter(p)
	}

	if config.GetConfig().ActxMeterEnabled {
		this.sqlActxMeter(p)
	}
}

func (this *TaskSql) sqlMeter(p *pack.CounterPack1) {
	meterSQL := meter.GetInstanceMeterSQL()
	//logutil.Infoln("TaskSql", "stat size=", meterSQL.Stat.Size())
	if meterSQL.Stat.Size() > 0 {
		p.SqlMeter = hmap.NewIntKeyLinkedMapDefault()
		en := meterSQL.Stat.Keys()
		for i := 0; i < 100 && en.HasMoreElements(); i++ {
			key := en.NextInt()
			b := meterSQL.Stat.Get(key).(*meter.SQLBucket)
			m := pack.NewSqlMeter()
			m.Time = b.Time
			m.Count = b.Count
			m.Error = b.Error
			m.FetchCount = b.FetchCount
			m.FetchTime = b.FetchTime
			p.SqlMeter.Put(key, m)
		}
		meterSQL.ResetStat()
	}
}

func (this *TaskSql) sqlActxMeter(p *pack.CounterPack1) {
	meterActiveX := meter.GetInstanceMeterActiveX()
	if meterActiveX.StatSql == nil {
		return
	}
	//logutil.Infoln("TaskSql", "stat size=",  meterActiveX.StatSql.Size())
	if meterActiveX.StatSql.Size() > 0 {
		if p.SqlMeter == nil {
			p.SqlMeter = hmap.NewIntKeyLinkedMapDefault()
		}
		en := meterActiveX.StatSql.Keys()
		for i := 0; i < 300 && en.HasMoreElements(); i++ {
			key := en.NextInt()
			var m *pack.SqlMeter
			o := p.SqlMeter.Get(key)
			if o == nil {
				m = pack.NewSqlMeter()
				p.SqlMeter.Put(key, pack.NewSqlMeter())
			} else {
				m = o.(*pack.SqlMeter)
			}
			m.Actx = meterActiveX.StatSql.Get(key)
		}
	}
}
