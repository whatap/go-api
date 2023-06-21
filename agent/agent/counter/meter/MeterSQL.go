package meter

import (
	"github.com/whatap/golib/util/hmap"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/util/logutil"
)

type SQLBucket struct {
	Count int32
	Time  int64
	Error int32

	FetchCount int64
	FetchTime  int64
}

func NewSQLBucket() *SQLBucket {
	p := new(SQLBucket)

	return p
}

type MeterSQL struct {
	Stat   *hmap.IntKeyLinkedMap
	Bucket *SQLBucket
}

var meterSQL *MeterSQL = newMeterSQL()

func newMeterSQL() *MeterSQL {
	p := new(MeterSQL)
	p.Stat = hmap.NewIntKeyLinkedMap(117, 1.0).SetMax(107)
	p.Bucket = NewSQLBucket()

	return p
}

func GetInstanceMeterSQL() *MeterSQL {
	if meterSQL != nil {
		return meterSQL
	} else {
		return newMeterSQL()
	}
}

func (this *MeterSQL) GetBucketReset() *SQLBucket {
	b := this.Bucket
	this.Bucket = NewSQLBucket()

	return b
}

func (this *MeterSQL) Add(dbc int32, elapsed int32, err bool) {
	this.Bucket.Count++
	this.Bucket.Time += int64(elapsed)
	if err {
		this.Bucket.Error++
	}

	conf := config.GetConfig()
	// DBC별로
	if conf.SqlDbcMeterEnabled && dbc != 0 {
		var b *SQLBucket

		if this.Stat.ContainsKey(dbc) {
			// 2019.12.13 GS VGS Magento 에서 nil 에러 . key는 있지만 nil 인경우 . 삭제 후 다시 만들어 줌
			// 추후 확인 필요
			if bk := this.Stat.Get(dbc); bk != nil {
				b = bk.(*SQLBucket)
			} else {
				logutil.Println("WA431", "Invalid key value in dbc meter ", dbc)
				this.Stat.Remove(dbc)
				b = NewSQLBucket()
				this.Stat.Put(dbc, b)
			}
		} else {
			b = NewSQLBucket()
			this.Stat.Put(dbc, b)
		}

		b.Count++
		b.Time += int64(elapsed)
		if err {
			b.Error++
		}
	}
}

func (this *MeterSQL) AddFetch(dbc int32, count int32, time int64) {
	this.Bucket.FetchCount += int64(count)
	this.Bucket.FetchTime += time
}

func (this *MeterSQL) ResetStat() {
	this.Stat = hmap.NewIntKeyLinkedMap(117, 1.0).SetMax(107)
}

//var sqlBucket *SQLBucket = NewSQLBucket()
//
//var SQLStat *hmap.IntKeyLinkedMap = hmap.NewIntKeyLinkedMap(117, 1.0).SetMax(107)
//
//
//func GetSQLBucketReset() *SQLBucket {
//	b := sqlBucket
//	sqlBucket = NewSQLBucket()
//
//	return b
//}
//
//func AddSQL(dbc int32,  elapsed int32, err bool) {
//	sqlBucket.Count++
//	sqlBucket.Time += int64(elapsed)
//	if (err) {
//		sqlBucket.Error++
//	}
//
//	conf := config.GetConfig()
//	// DBC별로
//	if conf.SqlDbcMeterEnabled && dbc!=0 {
//		var b *SQLBucket
//
//		if SQLStat.ContainsKey(dbc) {
//			b = SQLStat.Get(dbc).(*SQLBucket)
//		} else {
//			b = NewSQLBucket()
//			SQLStat.Put(dbc, b)
//		}
//
//		b.Count++
//		b.Time += int64(elapsed)
//		if err {
//			b.Error++
//		}
//	}
//}
//
//func AddFetch(dbc int32 ,count int32, time int64) {
//	sqlBucket.FetchCount += int64(count)
//	sqlBucket.FetchTime += time
//}
//
//func ResetSQLStat() {
//	SQLStat = hmap.NewIntKeyLinkedMap(117, 1.0).SetMax(107)
//}
