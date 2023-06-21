package stat

import (
	//"log"
	"container/list"
	"math"
	"sync"

	// import cycle error
	//	"github.com/whatap/go-api/agent/agent/trace"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/bitutil"
	"github.com/whatap/golib/util/hmap"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/data"
	langconf "github.com/whatap/go-api/agent/lang/conf"
	"github.com/whatap/go-api/agent/util/logutil"
)

const (
	STAT_SQL_TABLE_MAX_SIZE = 5000
)

type StatSql struct {
	table *hmap.LongKeyLinkedMap
	timer *TimingSender
}

var sqlLock = sync.Mutex{}
var statSql *StatSql

// Singleton func GetInstanceStatSql() *StatSql {
func GetInstanceStatSql() *StatSql {
	sqlLock.Lock()
	defer sqlLock.Unlock()
	if statSql != nil {
		return statSql
	}
	statSql = new(StatSql)
	//statSql.table = hmap.NewLongKeyLinkedMap().SetMax(STAT_SQL_TABLE_MAX_SIZE) //NewLongKeyLinkedMap(STAT_SQL_TABLE_MAX_SIZE+1,1)
	statSql.table = hmap.NewLongKeyLinkedMapDefault().SetMax(int(config.GetConfig().StatSqlMaxCount))
	statSql.timer = GetInstanceTimingSender()

	// conf 변경시 max 재설정
	langconf.AddConfObserver("StatSql", statSql)

	return statSql
}

// Implements lang/conf/ConfObserver/Runnable
func (this *StatSql) Run() {
	conf := config.GetConfig()
	this.table.SetMax(int(conf.StatSqlMaxCount))
}

// func (this *StatSql) getSql(dbc, sql int32) *pack.SqlRec {
func (this *StatSql) getSql(dbc, sql int32) *pack.SqlRec {
	//fmt.Println("StatSql:getSql key=", bitutil.Composite64(dbc, sql), ",hDbc=", dbc, ",wSql=", sql)
	return this.intern(bitutil.Composite64(dbc, sql))
}

// Java.LongKeyLinkedMap.intern 생성, Create overide가 불가능
// func (this *StatSql) intern(key int64) *pack.SqlRec {
func (this *StatSql) intern(key int64) *pack.SqlRec {

	if this.table.ContainsKey(key) {
		return this.table.Get(key).(*pack.SqlRec)
	}

	// Create override
	if this.table.IsFull() {
		return nil
	}

	hDbc := bitutil.GetHigh64(key)
	wSql := bitutil.GetLow64(key)

	//fmt.Println("StatSql:intern key=", key, ",hDbc=", hDbc, ",wSql=", wSql)

	rec := pack.NewSqlRec()
	rec.SetDbcSql(hDbc, wSql)
	rec.TimeMin = math.MaxInt32

	this.table.Put(key, rec)

	return rec
}

func (this *StatSql) AddFetch(dbc, sql, fetch int32, fetchTime int64) {
	if sql == 0 {
		return
	}
	r := this.getSql(dbc, sql)
	if r != nil {
		r.FetchCount += int64(fetch)
		r.FetchTime += fetchTime
	}
}

func (this *StatSql) AddUpdate(dbc, sql, updated int32) {
	if sql == 0 {
		return
	}
	r2 := this.getSql(dbc, sql)
	if r2 != nil {
		r2.UpdateCount += int64(updated)
	}
}

func (this *StatSql) AddSqlActive(dbc, sql int32) {
	//	if sql == 0 {
	//		return
	//	}
	//
	//	r := this.getSql(dbc, sql)
	//	if r != nil {
	//		r.CountActived++
	//	}
}

//func (this *StatSql) AddSqlTime(ctx *TraceContext, svc, dbc int32, sqlType byte, sql, time int32, isErr bool) {
// import cycle error를 피하기 위해 변환 TraceContext -> ServiceRec
//func (this *StatSql) AddSqlTime(urlRec *pack.ServiceRec, urlHash, dbc int32, sqlType byte, sql, time int32, isErr bool, updated int32) {

// SqlStep_3
// func (this *StatSql) AddSqlTime(urlHash, dbc int32, sqlType byte, sql, time int32, isErr bool) {
func (this *StatSql) AddSqlTime(urlHash, dbc int32, sql, time int32, isErr bool) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA531", "StatSql.addSqlTime Error", r)
		}
	}()

	if sql != 0 {
		sqlRec := this.getSql(dbc, sql)
		if sqlRec != nil {
			sqlRec.CountTotal++
			sqlRec.TimeSum += int64(time)
			sqlRec.TimeStd += int64(time) * int64(time)
			sqlRec.TimeMin = int32(math.Min(float64(sqlRec.TimeMin), float64(time)))
			sqlRec.TimeMax = int32(math.Max(float64(sqlRec.TimeMax), float64(time)))
			//sqlRec.SqlCrud = sqlType
			if isErr {
				sqlRec.CountError++
			}
			//sqlRec.UpdateCount += int64(updated)

			sqlRec.Service = urlHash

			//fmt.Println("StatSql.addSqlTime tc.add", sqlRec)
		}
	}
}

func (this *StatSql) Send(now int64) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA532", "StatSql.Send Error", r)
		}
	}()
	//fmt.Println("StatSql.Send size=", this.table.Size())

	if this.table.Size() == 0 {
		return
	}

	out := list.New()
	en := this.table.Values()
	for en.HasMoreElements() {
		out.PushBack(en.NextElement())
	}
	this.table.Clear()

	//p := pack.NewStatSqlPack().SetRecords(this.table.Size(), this.table.Values())
	p := pack.NewStatSqlPack().SetRecordsList(out)
	p.Time = now
	data.Send(p)
}

func (this *StatSql) Clear() {
	this.table.Clear()
}
