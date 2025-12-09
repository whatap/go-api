package meter

import (
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/golib/util/hmap"
)

type StatusBucket struct {
	Count int32
	Time  int64
}

func NewStatusBucket() *StatusBucket {
	p := new(StatusBucket)

	return p
}

type MeterStatus struct {
	Stat      *hmap.IntKeyLinkedMap
	Bucket200 *StatusBucket
	Bucket400 *StatusBucket
	Bucket500 *StatusBucket
}

var meterStatus *MeterStatus = newMeterStatus()

func newMeterStatus() *MeterStatus {
	p := new(MeterStatus)
	p.Stat = hmap.NewIntKeyLinkedMap(117, 1.0).SetMax(107)
	p.Bucket200 = NewStatusBucket()
	p.Bucket400 = NewStatusBucket()
	p.Bucket500 = NewStatusBucket()

	return p
}

func GetInstanceMeterStatus() *MeterStatus {
	if meterStatus != nil {
		return meterStatus
	} else {
		return newMeterStatus()
	}
}

func (this *MeterStatus) Clear() {
	this.Bucket200 = NewStatusBucket()
	this.Bucket400 = NewStatusBucket()
	this.Bucket500 = NewStatusBucket()
	this.ResetStat()
}

func (this *MeterStatus) GetBucketReset() (*StatusBucket, *StatusBucket, *StatusBucket) {
	b200 := this.Bucket200
	this.Bucket200 = NewStatusBucket()
	b400 := this.Bucket400
	this.Bucket400 = NewStatusBucket()
	b500 := this.Bucket500
	this.Bucket500 = NewStatusBucket()

	return b200, b400, b500
}

func (this *MeterStatus) Add(status int32, elapsed int32) {
	g := status / 100
	switch g {
	case 2:
		this.Bucket200.Count++
		this.Bucket200.Time += int64(elapsed)
	case 4:
		this.Bucket400.Count++
		this.Bucket400.Time += int64(elapsed)
	case 5:
		this.Bucket500.Count++
		this.Bucket500.Time += int64(elapsed)
	}

	conf := config.GetConfig()
	if conf.TxStatusMeterEnabled && status != 0 {
		var b *StatusBucket

		if this.Stat.ContainsKey(status) {
			if tmp := this.Stat.Get(status); tmp != nil {
				if val, ok := tmp.(*StatusBucket); ok {
					b = val
				} else {
					this.Stat.Remove(status)
					b = NewStatusBucket()
					this.Stat.Put(status, b)
				}
			} else {
				logutil.Println("WA451", "Invalid key value in tx status meter ", status)
				this.Stat.Remove(status)
				b = NewStatusBucket()
				this.Stat.Put(status, b)
			}
		} else {
			b = NewStatusBucket()
			this.Stat.Put(status, b)
		}

		b.Count++
		b.Time += int64(elapsed)
	}
}

func (this *MeterStatus) ResetStat() {
	this.Stat = hmap.NewIntKeyLinkedMap(117, 1.0).SetMax(107)
}
