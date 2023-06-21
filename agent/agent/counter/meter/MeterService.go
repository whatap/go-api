package meter

import (
	"github.com/whatap/go-api/agent/agent/alert"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/golib/lang"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/service"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/hmap"
)

type MeterService struct {
	Bucket      *HitBucket
	StatByOID   *hmap.IntKeyLinkedMap
	StatByPKIND *hmap.LinkedMap
	StatByPOID  *hmap.LinkedMap
	Unknown     *BucketSimple
	Arrival     int
}

var meterService *MeterService = newMeterService()

func newMeterService() *MeterService {
	p := new(MeterService)
	p.Bucket = NewHitBucket()
	p.StatByOID = hmap.NewIntKeyLinkedMap(309, 1.0).SetMax(307)
	p.StatByPKIND = hmap.NewLinkedMap(309, 1.0).SetMax(307)
	p.StatByPOID = hmap.NewLinkedMapDefault()
	p.Unknown = NewBucketSimple()
	p.Arrival = 0

	return p
}
func GetInstanceMeterService() *MeterService {
	if meterService == nil {
		return newMeterService()
	} else {
		return meterService
	}
}

func (this *MeterService) GetBucket() *HitBucket {
	return this.Bucket
}
func (this *MeterService) GetAndResetBucket() *HitBucket {
	b := this.Bucket
	this.Bucket = NewHitBucket()
	return b
}

// func (this *MeterService) Add(serviceHash int32, elapsed int32, err bool, mCallerPcode int64, mCallerOkind int32, mCallerOid int32) {
// func (this *MeterService) Add(serviceHash int32, elapsed int32, err bool, errLevel byte, mCallerPcode int64, mCallerOkind int32, mCallerOid int32) {
func (this *MeterService) Add(tx *service.TxRecord, mCallerPcode int64, mCallerOkind int32, mCallerOid int32) {
	conf := config.GetConfig()

	if tx.Elapsed < 0 {
		tx.Elapsed = 0
	}
	this.Bucket.Count++
	this.Bucket.Timesum += int64(tx.Elapsed)
	this.Bucket.DbcSum += int64(tx.DbcTime)
	this.Bucket.SqlSum += int64(tx.SqlTime)
	this.Bucket.HttpcSum += int64(tx.HttpcTime)

	// BixException 통계 제외 처리를 위해, Warning 일 때만 err 처리
	err := (tx.ErrorLevel >= pack.WARNING)
	if err {
		this.Bucket.Error++
		this.Bucket.CountTotal++
	} else {
		this.Bucket.CountTotal++
		if tx.Elapsed <= conf.ApdexTime {
			this.Bucket.CountSatisfied++
			tx.Apdex = 2
		} else if tx.Elapsed <= conf.ApdexTime4T {
			this.Bucket.CountTolerated++
			tx.Apdex = 1
		}
	}
	this.Bucket.TimeSqrSum += int64(tx.Elapsed) * int64(tx.Elapsed)
	// DEBUG min, max
	if this.Bucket.TimeMax < tx.Elapsed {
		this.Bucket.TimeMax = tx.Elapsed
	}
	if this.Bucket.TimeMin == 0 || this.Bucket.TimeMin > tx.Elapsed {
		this.Bucket.TimeMin = tx.Elapsed
	}
	// logutil.Infoln(">>>>>", "count=", this.Bucket.Count, ",uri=", tx.Service, ",elapsed=", tx.Elapsed)
	// logutil.Infoln(">>>>", ",", mCallerPcode, ",", mCallerOid, ",", mCallerOkind)

	this.Bucket.Hitmap.Add(int(tx.Elapsed), err)

	if conf.TxCallerMeterEnabled {
		var c *BucketSimple = nil
		if mCallerOid != 0 {
			if mCallerPcode == conf.PCODE {
				if this.StatByOID.ContainsKey(mCallerOid) {
					o := this.StatByOID.Get(mCallerOid)
					if o != nil {
						c = o.(*BucketSimple)
					} else {
						c = NewBucketSimple()
						this.StatByOID.Put(mCallerOid, c)
					}
				} else {
					c = NewBucketSimple()
					this.StatByOID.Put(mCallerOid, c)
				}
				c.Count++
				c.Timesum += int64(tx.Elapsed)
				c.DbcSum += int64(tx.DbcTime)
				c.SqlSum += int64(tx.SqlTime)
				c.HttpcSum += int64(tx.HttpcTime)
				if err {
					c.Error++
				}
			} else {
				key := lang.NewPOID(mCallerPcode, mCallerOid)
				o := this.StatByPOID.Get(key)
				if o != nil {
					c = o.(*BucketSimple)
				} else {
					c = NewBucketSimple()
					this.StatByPOID.Put(key, c)
				}
				c.Count++
				c.Timesum += int64(tx.Elapsed)
				if err {
					c.Error++
				}
			}

			if conf.TxCallerMeterPKindEnabled {
				// conf.tx_caller_meter_pkind_enabled {
				key := lang.NewPKIND(mCallerPcode, mCallerOkind)
				o := this.StatByPKIND.Get(key)
				if o != nil {
					c = o.(*BucketSimple)
				} else {
					c = NewBucketSimple()
					this.StatByPKIND.Put(key, c)
				}
				c.Count++
				c.Timesum += int64(tx.Elapsed)
				if err {
					c.Error++
				}
			}
		} else {
			c = this.Unknown
			c.Count++
			c.Timesum += int64(tx.Elapsed)
			if err {
				c.Error++
			}
		}
	}
}

func (this *MeterService) ResetStat() {
	if this.StatByOID.Size() > 0 {
		// BucketSimple
		this.StatByOID = hmap.NewIntKeyLinkedMap(309, 1.0).SetMax(307)
	}
	if this.StatByPKIND.Size() > 0 {
		// BucketSimple
		this.StatByPKIND = hmap.NewLinkedMap(309, 1.0).SetMax(307)
	}

	if this.StatByPOID.Size() > 0 {
		// BucketSimple
		this.StatByPOID = hmap.NewLinkedMap(309, 1.0).SetMax(307)
	}
	if this.Unknown.Count > 0 {
		this.Unknown = NewBucketSimple()
	}
}

type HitBucket struct {
	Hitmap *pack.HitMapPack1

	Count int32

	CountSatisfied int32
	CountTolerated int32
	CountTotal     int32

	Timesum  int64
	DbcSum   int64
	SqlSum   int64
	HttpcSum int64
	Error    int32

	TimeSqrSum int64
	TimeMin    int32
	TimeMax    int32

	LastVerEvent       int64
	LastVerOverStart   int64
	LastHorizEvent     int64
	LastHorizOverStart int64
}

func NewHitBucket() *HitBucket {
	p := &HitBucket{}
	p.Hitmap = pack.NewHitMapPack1()
	return p
}

func (this *HitBucket) Reset() {
	this.Count = 0
	this.Timesum = 0
	this.Error = 0
	this.Hitmap = pack.NewHitMapPack1()
}

func (this *HitBucket) ProcessVerticalEvent() *pack.EventPack {
	conf := config.GetConfig()
	now := dateutil.SystemNow()
	// 현재시간 percent
	cnt := int32(0)
	// 2.5(index 20) 초 이상 히트맵 카운트 100 개를 100프로로 설정
	for i := pack.HITMAP_VERTICAL_INDEX; i < pack.HITMAP_LENGTH; i++ {
		if this.Hitmap.Hit[i] > 0 {
			if conf.HitMapVerEventErrorOnly {
				if this.Hitmap.Error[i] > 0 {
					cnt++
				}
			} else {
				cnt++
			}
		}
	}
	//logutil.Infoln("HitBucket", "Verticall=", cnt, ",starttype=", this.LastVerOverStart)

	if cnt >= conf.HitMapVerEventWarningPercent {
		if this.LastVerOverStart == 0 {
			this.LastVerOverStart = now
		}
		if this.LastVerOverStart > 0 && now-this.LastVerOverStart >= int64(conf.HitMapVerEventDuration) {
			this.LastVerOverStart = 0
			if cnt >= conf.HitMapVerEventFatalPercent {
				return alert.HitMapVertical(cnt, pack.FATAL)
			} else {
				return alert.HitMapVertical(cnt, pack.WARNING)
			}
		}
	} else {
		this.LastVerOverStart = 0
	}
	return nil
}

func (this *HitBucket) ProcessHorizontalEvent() *pack.EventPack {
	conf := config.GetConfig()
	now := dateutil.SystemNow()

	// 현재시간 percent
	cnt := 0
	idx := 0
	pos := this.Hitmap.HitMapIndex(int(conf.HitMapHorizEventBasetime))
	// 5(index 60) 초 이상 히트맵 카운트, 5초 이상 최소 시간이 있으면 중지
	for i := pos; i < pack.HITMAP_LENGTH; i++ {
		if this.Hitmap.Hit[i] > 0 {
			if conf.HitMapHorizEventErrorOnly {
				if this.Hitmap.Error[i] > 0 {
					cnt++
					idx = i
					break
				}
			} else {
				cnt++
				idx = i
				break
			}
		}
	}

	//logutil.Infof("HitBucket", "Horizontal %d", cnt)

	if cnt > 0 {
		if this.LastHorizOverStart == 0 {
			this.LastHorizOverStart = now
		}
		if now-this.LastHorizOverStart >= int64(conf.HitMapHorizEventDuration) {
			this.LastHorizOverStart = 0
			return alert.HitMapHorizontal(this.Hitmap.HitMapTime(idx))
		}

	} else {
		this.LastHorizOverStart = 0
	}
	return nil
}

type BucketSimple struct {
	Count    int32
	Timesum  int64
	DbcSum   int64
	SqlSum   int64
	HttpcSum int64
	Error    int32
}

func NewBucketSimple() *BucketSimple {
	p := new(BucketSimple)
	return p
}

//
//var hitBucket *HitBucket = NewHitBucket()
//
//var statByOID *hmap.IntKeyLinkedMap = hmap.NewIntKeyLinkedMap(309, 1.0).SetMax(307)
//var statByPKIND *hmap.LinkedMap = hmap.NewLinkedMap(309, 1.0).SetMax(307)
////
////	public LinkedMap<PKIND, BucketSimple> statByPKIND = new LinkedMap<PKIND, BucketSimple>(309, 1f) {
////		protected BucketSimple create(PKIND pkind) {
////			return new BucketSimple();
////		};
////	}.setMax(307);
////
//
//
//func GetBucket() *HitBucket {
//	return hitBucket
//}
////func AddTransaction(serviceHash int32, elapsed int32, err bool) {
//func AddTransaction(serviceHash int32, elapsed int32, err bool, mCallerPcode int64, mCallerOkind int32, mCallerOid int32) {
//	if elapsed < 0 {
//		elapsed = 0
//	}
//	hitBucket.Count++
//	hitBucket.Timesum += int64(elapsed)
//	hitBucket.Hitmap.Add(int(elapsed), err)
//	if err {
//		hitBucket.Error++
//	}
//
//
//}
