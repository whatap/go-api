package meter

import (
	"sync"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/golib/util/hmap"
)

type HTTPCBucket struct {
	Count int32
	Time  int64
	Error int32
}

func NewHTTPCBucket() *HTTPCBucket {
	p := new(HTTPCBucket)

	return p
}

type MeterHTTPC struct {
	Bucket *HTTPCBucket
	Stat   *hmap.IntKeyLinkedMap
	lock   sync.Mutex
}

var meterHTTPC *MeterHTTPC = newMeterHTTPC()

func newMeterHTTPC() *MeterHTTPC {
	p := new(MeterHTTPC)
	p.Bucket = NewHTTPCBucket()
	p.Stat = hmap.NewIntKeyLinkedMap(117, 1.0).SetMax(107)
	return p
}
func GetInstanceMeterHTTPC() *MeterHTTPC {
	if meterHTTPC == nil {
		return newMeterHTTPC()
	} else {
		return meterHTTPC
	}
}
func (this *MeterHTTPC) Clear() {
	this.Bucket = NewHTTPCBucket()
	this.ResetStat()
}
func (this *MeterHTTPC) GetBucketReset() *HTTPCBucket {
	b := this.Bucket
	this.Bucket = NewHTTPCBucket()
	return b
}

func (this *MeterHTTPC) Add(host int32, elapsed int32, err bool) {
	this.lock.Lock()
	defer this.lock.Unlock()

	this.Bucket.Count++
	this.Bucket.Time += int64(elapsed)
	if err {
		this.Bucket.Error++
	}

	conf := config.GetConfig()

	if conf.HttpcHostMeterEnabled {
		var b *HTTPCBucket

		if this.Stat.ContainsKey(host) {
			b = this.Stat.Get(host).(*HTTPCBucket)
		} else {
			b = NewHTTPCBucket()
			this.Stat.Put(host, b)
		}

		b.Count++
		b.Time += int64(elapsed)
		if err {
			b.Error++
		}
	}
}

func (this *MeterHTTPC) ResetStat() {
	this.Stat = hmap.NewIntKeyLinkedMap(117, 1.0).SetMax(107)
}

//
//
//var httpcBucket *HTTPCBucket = NewHTTPCBucket()
//var httpcLock sync.Mutex
//
//var HttpcStat *hmap.IntKeyLinkedMap = hmap.NewIntKeyLinkedMap(117, 1.0).SetMax(107)
//
//func GetHTTPCBucketReset() *HTTPCBucket {
//	b := httpcBucket
//	httpcBucket = NewHTTPCBucket()
//	return b
//}
//
//func AddHTTPC(host int32, elapsed int32, err bool) {
//	httpcLock.Lock()
//	defer httpcLock.Unlock()
//
//	httpcBucket.Count++
//	httpcBucket.Time += int64(elapsed)
//	if err {
//		httpcBucket.Error++
//	}
//
//	conf := config.GetConfig()
//
//	if conf.HttpcHostMeterEnabled {
//		var b *HTTPCBucket
//
//		if HttpcStat.ContainsKey(host) {
//			b = HttpcStat.Get(host).(*HTTPCBucket)
//		} else {
//			b = NewHTTPCBucket()
//			HttpcStat.Put(host, b)
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
//func ResetHttpcStat() {
//	HttpcStat = hmap.NewIntKeyLinkedMap(117, 1.0).SetMax(107)
//}
