package stat

import (
	//"log"
	"math"
	"sync"

	"github.com/whatap/go-api/agent/agent/config"
	langconf "github.com/whatap/go-api/agent/lang/conf"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/hmap"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/util/logutil"
)

const (
	STAT_HTTPC_TABLE_MAX_SIZE = 5000
)

type StatHttpc struct {
	table *hmap.LinkedMap
	timer *TimingSender
}

var httpcLock = sync.Mutex{}
var statHttpc *StatHttpc

// Singleton func GetInstanceStatSql() *StatSql {
func GetInstanceStatHttpc() *StatHttpc {
	httpcLock.Lock()
	defer httpcLock.Unlock()
	if statHttpc != nil {
		return statHttpc
	}
	statHttpc = new(StatHttpc)
	//statHttpc.table = hmap.NewLinkedMap().SetMax(STAT_HTTPC_TABLE_MAX_SIZE) //NewLongKeyLinkedMap(STAT_SQL_TABLE_MAX_SIZE+1,1)
	statHttpc.table = hmap.NewLinkedMapDefault().SetMax(int(config.GetConfig().StatHttpcMaxCount))
	statHttpc.timer = GetInstanceTimingSender()

	// conf 변경시 max 재설정
	langconf.AddConfObserver("StatHttpc", statHttpc)

	return statHttpc
}

// Implements lang/conf/ConfObserver/Runnable
func (this *StatHttpc) Run() {
	conf := config.GetConfig()
	this.table.SetMax(int(conf.StatHttpcMaxCount))
}

func (this *StatHttpc) getHttpc(url, host, port int32) *pack.HttpcRec {
	//fmt.Println("StatSql:getSql key=", bitutil.Composite64(dbc, sql), ",hDbc=", dbc, ",wSql=", sql)
	return this.intern(NewHTTPC(url, host, port))
}

// Java.LongKeyLinkedMap.intern 생성, Create overide가 불가능
// func (this *StatSql) intern(key int64) *pack.SqlRec {
func (this *StatHttpc) intern(key *HTTPC) *pack.HttpcRec {

	if this.table.ContainsKey(key) {
		return this.table.Get(key).(*pack.HttpcRec)
	}

	// Create override
	if this.table.IsFull() {
		return nil
	}

	rec := pack.NewHttpcRec()
	rec.SetUrlHostPort(key.url, key.host, key.port)
	rec.TimeMin = math.MaxInt32

	this.table.Put(key, rec)

	return rec
}

func (this *StatHttpc) AddHttpcTime(serviceUrlHash, httpcUrlHash, httpcHost, httpcPort, time int32, isErr bool) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA521", "StatHttpc.AddHttpcTime Error", r)
		}
	}()
	//	if serviceUrlHash == 0 || httpcUrlHash == 0 {
	//		return
	//	}
	//
	//	urlRec := GetInstanceStatTranx().GetService(serviceUrlHash)
	//
	//	if urlRec != nil {
	//		urlRec.HttpcCount++
	//		urlRec.HttpcTime++
	//
	//		if urlRec.HttpcMap == nil {
	//			urlRec.HttpcMap = pack.CreateMap(11)
	//		}
	//
	//		//tc := urlRec.HttpcMap.Intern(httpcUrlHash)
	//		var tc *pack.TimeCount
	//
	//		if !urlRec.HttpcMap.ContainsKey(httpcUrlHash) {
	//			//fmt.Println("StatSql.addSqlTime tc is nil")
	//			//return this.size() >= 1000 ? null : new TimeCount();
	//			if urlRec.HttpcMap.Size() >= 1000 {
	//				tc = nil
	//			} else {
	//				tc := pack.NewTimeCountDefault()
	//				urlRec.HttpcMap.Put(httpcUrlHash, tc)
	//				//fmt.Println("StatSql.addSqlTime tc create sql=", sql, ",TimeCount=", tc)
	//			}
	//		} else {
	//			tc = urlRec.HttpcMap.Get(httpcUrlHash).(*pack.TimeCount)
	//		}
	//
	//		if tc != nil {
	//			tc.Add(time, isErr)
	//		}
	//	}

	if httpcUrlHash != 0 {
		httpcRec := this.getHttpc(httpcUrlHash, httpcHost, httpcPort)

		if httpcRec != nil {
			httpcRec.CountTotal++
			httpcRec.TimeSum += int64(time)
			httpcRec.TimeStd += (int64(time) * int64(time))
			httpcRec.TimeMax = int32(math.Max(float64(httpcRec.TimeMax), float64(time)))
			httpcRec.TimeMin = int32(math.Min(float64(httpcRec.TimeMin), float64(time)))
			if isErr {
				httpcRec.CountError++
			}
			httpcRec.Service = serviceUrlHash
		}
	}
}

func (this *StatHttpc) Send(now int64) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA522", "StatHttpc.Send Error", r)
		}
	}()
	if this.table.Size() == 0 {
		return
	}

	p := pack.NewStatHttpcPack().SetRecords(this.table.Size(), this.table.Values())
	p.Time = now
	this.table.Clear()
	data.Send(p)
}

func (this *StatHttpc) Clear() {
	this.table.Clear()
}

// hmap.LinkedKey interface
type HTTPC struct {
	url  int32
	host int32
	port int32
}

func NewHTTPC(url, host, port int32) *HTTPC {
	p := new(HTTPC)
	p.url = url
	p.host = host
	p.port = port

	return p
}

func (this *HTTPC) Hash() uint {
	prime := uint(31)
	result := uint(1)
	result = prime*result + uint(this.host)
	result = prime*result + uint(this.port)
	result = prime*result + uint(this.url)

	return result
}

func (this *HTTPC) Equals(obj hmap.LinkedKey) bool {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA521", "StatHttpc.Equals Recover ", r)
		}
	}()
	if this == obj.(*HTTPC) {
		return true
	}
	if obj == nil {
		return false
	}
	//			if (getClass() != obj.getClass())
	//				return false;
	//
	other := obj.(*HTTPC)
	return (this.host == other.host && this.port == other.port && this.url == other.url)
}
