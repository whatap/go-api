package meter

import (
	"github.com/whatap/golib/util/hmap"
)

type ConnPoolBucket struct {
	Active   *hmap.IntIntMap
	InActive *hmap.IntIntMap
}

type MeterConnPool struct {
	Bucket *ConnPoolBucket
}

var meterConnPool *MeterConnPool = newMeterConnPool()

func newMeterConnPool() *MeterConnPool {
	p := new(MeterConnPool)
	p.GetBucketReset()

	return p
}
func GetInstanceConnPool() *MeterConnPool {
	if meterConnPool != nil {
		return meterConnPool
	} else {
		return newMeterConnPool()
	}
}

func (this *MeterConnPool) Clear() {
	this.Bucket = NewConnPoolBucket()
}

func NewConnPoolBucket() *ConnPoolBucket {
	p := new(ConnPoolBucket)
	p.Active = hmap.NewIntIntMapDefault()
	p.InActive = hmap.NewIntIntMapDefault()

	return p
}

func (this *MeterConnPool) GetBucketReset() *ConnPoolBucket {
	b := this.Bucket
	this.Bucket = NewConnPoolBucket()
	return b
}

func (this *MeterConnPool) AddDBConnPool(urlHash int32, actCnt int32, inactCnt int32) {
	this.Bucket.Active.Put(urlHash, actCnt)
	this.Bucket.InActive.Put(urlHash, inactCnt)
}

func (this *ConnPoolBucket) GetActiveMap() *hmap.IntIntMap {
	return this.Active
}

func (this *ConnPoolBucket) GetIdleMap() *hmap.IntIntMap {
	return this.InActive
}
