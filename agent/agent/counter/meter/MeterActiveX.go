package meter

import (
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/golib/lang"
	"github.com/whatap/golib/lang/ref"
	"github.com/whatap/golib/util/hmap"
)

type MeterActiveX struct {
	StatByOid      *hmap.IntIntLinkedMap
	StatByGroup    *hmap.LinkedMap
	StatByPOid     *hmap.LinkedMap
	StatSql        *hmap.IntIntLinkedMap
	StatSliceSql   *hmap.IntKeyLinkedMap
	StatHttpc      *hmap.IntIntLinkedMap
	StatSliceHttpc *hmap.IntKeyLinkedMap

	Unknown int
	conf    *config.Config
}

var meterActiveX *MeterActiveX //= newMeterActiveX()

func newMeterActiveX() *MeterActiveX {
	p := new(MeterActiveX)
	p.StatByOid = hmap.NewIntIntLinkedMap()
	p.StatByGroup = hmap.NewLinkedMapDefault()
	p.StatByPOid = hmap.NewLinkedMapDefault()
	p.StatSql = hmap.NewIntIntLinkedMap()
	p.StatHttpc = hmap.NewIntIntLinkedMap()
	p.Unknown = 0
	p.conf = config.GetConfig()

	return p
}
func GetInstanceMeterActiveX() *MeterActiveX {
	if meterActiveX != nil {
		return meterActiveX
	} else {
		return newMeterActiveX()
	}
}

func (this *MeterActiveX) Clear() {
	this.ReInit()
}

func (this *MeterActiveX) AddTx(callerPcode int64, callerOkind, callerOid int32) {
	if callerOid != 0 {
		if callerPcode == this.conf.PCODE {
			this.StatByOid.Add(callerOid, 1)
		} else {
			key := lang.NewPOID(callerPcode, callerOid)
			var v *ref.INT
			if this.StatByPOid.ContainsKey(key) {
				v = this.StatByPOid.Get(key).(*ref.INT)
			} else {
				v = ref.NewINT()
				this.StatByPOid.Put(key, v)
			}
			v.Value++
		}

		key := lang.NewPKIND(callerPcode, callerOkind)
		var v *ref.INT
		if this.StatByGroup.ContainsKey(key) {
			v = this.StatByGroup.Get(key).(*ref.INT)
		} else {
			v = ref.NewINT()
			this.StatByGroup.Put(key, v)
		}
		v.Value++
	} else {
		this.Unknown++
	}
}

func (this *MeterActiveX) ReInit() {
	this.StatByOid = this.reset(this.StatByOid)
	this.StatByGroup = hmap.NewLinkedMapDefault()
	this.StatByPOid = hmap.NewLinkedMapDefault()
	this.StatSql = this.reset(this.StatSql)
	this.StatSliceSql = this.resetIntKey(this.StatSliceSql)
	this.StatHttpc = this.reset(this.StatHttpc)
	this.StatSliceHttpc = this.resetIntKey(this.StatSliceHttpc)
}

func (this *MeterActiveX) reset(m *hmap.IntIntLinkedMap) *hmap.IntIntLinkedMap {
	if m == nil {
		m = hmap.NewIntIntLinkedMap()
	} else {
		m.Clear()
	}
	return m
}

func (this *MeterActiveX) resetIntKey(m *hmap.IntKeyLinkedMap) *hmap.IntKeyLinkedMap {
	if m == nil {
		m = hmap.NewIntKeyLinkedMapDefault()
	} else {
		m.Clear()
	}
	return m
}

func (this *MeterActiveX) AddSql(dbc int32) {
	this.StatSql.Add(dbc, 1)
}
func (this *MeterActiveX) AddSqlSlice(dbc int32, elapsed int) {
	if v := this.StatSliceSql.Get(dbc); v != nil {
		//this.StatSliceSql.Add(dbc, 1)
		if vArr, ok := v.([]int16); ok {
			vArr[idx(elapsed)]++
			return
		}
	}
	v := make([]int16, 3, 3)
	v[idx(elapsed)]++
	this.StatSliceSql.Put(dbc, v)
}
func (this *MeterActiveX) AddHttpc(host int32) {
	this.StatHttpc.Add(host, 1)
}

func (this *MeterActiveX) AddSqlHttpc(host int32, elapsed int) {
	if v := this.StatSliceHttpc.Get(host); v != nil {
		//this.StatSliceHttpc.Add(dbc, 1)
		if vArr, ok := v.([]int16); ok {
			vArr[idx(elapsed)]++
			return
		}
	}
	v := make([]int16, 3, 3)
	v[idx(elapsed)]++
	this.StatSliceHttpc.Put(host, v)
}

func idx(elapsed int) int {
	switch elapsed / 1000 {
	case 0:
	case 1:
	case 2:
		return 0
	case 3:
	case 4:
	case 5:
	case 6:
	case 7:
		return 1
	}
	return 2
}
