package meter

import (
	//"log"
	"sync"
	//	"time"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/hmap"

	//	"github.com/whatap/golib/util/keygen"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/golib/lang/value"
)

type MeterSelf struct {
	packetLastTime int64
	packetSum      int64
	packetMax      int64

	cpuLastTime int64
	cpuCnt      int32
	cpuSum      float32
	cpuMax      float32

	memLastTime int64
	memCnt      int32
	memSum      int64
	memMax      int64

	packetMap *hmap.LongLongLinkedMap
	cpuMap    *hmap.LongFloatLinkedMap
	memMap    *hmap.LongLongLinkedMap

	packetMaxMap *hmap.LongLongLinkedMap
	cpuMaxMap    *hmap.LongFloatLinkedMap
	memMaxMap    *hmap.LongLongLinkedMap

	lock sync.Mutex
}

func newMeterSelf() *MeterSelf {
	conf := config.GetConfig()
	p := new(MeterSelf)
	p.packetMap = hmap.NewLongLongLinkedMapDefault().SetMax(int(conf.MeterSelfBufferMax))
	p.packetMaxMap = hmap.NewLongLongLinkedMapDefault().SetMax(int(conf.MeterSelfBufferMax))

	p.cpuMap = hmap.NewLongFloatLinkedMap().SetMax(int(conf.MeterSelfBufferMax))
	p.cpuMaxMap = hmap.NewLongFloatLinkedMap().SetMax(int(conf.MeterSelfBufferMax))

	p.memMap = hmap.NewLongLongLinkedMapDefault().SetMax(int(conf.MeterSelfBufferMax))
	p.memMaxMap = hmap.NewLongLongLinkedMapDefault().SetMax(int(conf.MeterSelfBufferMax))

	return p
}

var meterSelfLock = sync.Mutex{}
var meterSelf *MeterSelf

// Singleton
func GetInstanceMeterSelf() *MeterSelf {
	meterSelfLock.Lock()
	defer meterSelfLock.Unlock()
	if meterSelf != nil {
		return meterSelf
	}
	meterSelf = newMeterSelf()

	return meterSelf
}

func (this *MeterSelf) Clear() {
	this.packetMap.Clear()
	this.cpuMap.Clear()
	this.memMap.Clear()

	this.packetMaxMap.Clear()
	this.cpuMaxMap.Clear()
	this.memMaxMap.Clear()
}

// Call with packet length value when send Packet
func (this *MeterSelf) AddMeterSelfPacket(packet int64) {
	this.lock.Lock()
	defer func() {
		this.lock.Unlock()
		if x := recover(); x != nil {
			logutil.Println("WA421", "Recover AddMeterSelfPacket ", x)
		}
	}()

	now := dateutil.Now() / int64(config.GetConfig().MeterSelfInterval) * int64(config.GetConfig().MeterSelfInterval)
	if this.packetLastTime == 0 {
		this.packetLastTime = now
	}

	// INTERVAL 지나면 추가, 그 전 까지는 함산, 기본 단위는 kbyte
	if this.packetLastTime != now {
		this.packetMap.Put(now, (this.packetSum / 1024))
		this.packetMaxMap.Put(now, (this.packetMax / 1024))
		//logutil.Infoln("Meter Self Packet"," time=", this.packetLastTime,"packet=", packet, " sum=", (this.packetSum/1024), ",max=" , (this.packetMax/1024) )
		this.packetSum = packet
		this.packetMax = packet
		this.packetLastTime = now
	} else {
		if this.packetMax < packet {
			this.packetMax = packet
		}
		this.packetSum += packet
		//logutil.Infoln("Meter Self Packet ADD"," time=", this.packetLastTime, " packet=", packet, ",sum=", this.packetSum, ",max=", this.packetMax)
	}

}

func (this *MeterSelf) AddMeterSelfCpu(cpuUsage float32) {
	this.lock.Lock()
	defer func() {
		this.lock.Unlock()
		if x := recover(); x != nil {
			logutil.Println("WA422", "Recover AddMeterSelfCpu ", x)
		}
	}()
	now := dateutil.Now() / int64(config.GetConfig().MeterSelfInterval) * int64(config.GetConfig().MeterSelfInterval)
	if this.cpuLastTime == 0 {
		this.cpuLastTime = now
	}
	// INTERVAL 지나면 추가, 그 전 까지는 함산
	if this.cpuLastTime != now {
		// 평균
		this.cpuMap.Put(now, (this.cpuSum / float32(this.cpuCnt)))
		this.cpuMaxMap.Put(now, this.cpuMax)
		//logutil.Infoln("Meter Self Cpu"," time=",this.cpuLastTime, ",cpuUsage=", cpuUsage, " sum=", this.cpuSum, " count=", this.cpuCnt, " Avg=", (this.cpuSum/float32(this.cpuCnt)))
		this.cpuSum = cpuUsage
		this.cpuMax = cpuUsage
		this.cpuCnt = 1
		this.cpuLastTime = now
	} else {
		if this.cpuMax < cpuUsage {
			this.cpuMax = cpuUsage
		}
		this.cpuSum += cpuUsage
		this.cpuCnt++
		//logutil.Infoln("Meter Self Cpu"," time=",this.cpuLastTime, ",cpuUsage=", cpuUsage, " sum=", this.cpuSum, " count=", this.cpuCnt, " Avg=", (this.cpuSum/float32(this.cpuCnt)), ", Max=", this.cpuMax);

	}
}

func (this *MeterSelf) AddMeterSelfMem(memUsage int64) {
	this.lock.Lock()
	defer func() {
		this.lock.Unlock()
		if x := recover(); x != nil {
			logutil.Println("WA423", "Recover AddMeterSelfMem ", x)
		}
	}()

	now := dateutil.Now() / int64(config.GetConfig().MeterSelfInterval) * int64(config.GetConfig().MeterSelfInterval)
	if this.memLastTime == 0 {
		this.memLastTime = now
	}

	// INTERVAL 지나면 추가, 그 전 까지는 함산
	if this.memLastTime != now {
		// 평균
		this.memMap.Put(now, (this.memSum / int64(this.memCnt)))
		this.memMaxMap.Put(now, this.memMax)
		//logutil.Infoln("Meter Self Mem"," time=", this.memLastTime, ",cpuUsage=", memUsage, " sum=", this.memSum, " count=", this.memCnt, " Avg=", (this.memSum/int64(this.memCnt)))
		this.memSum = memUsage
		this.memMax = memUsage
		this.memCnt = 1
		this.memLastTime = now
	} else {
		if this.memMax < memUsage {
			this.memMax = memUsage
		}
		this.memSum += memUsage
		this.memCnt++
		//logutil.Infoln("Meter Self Mem"," time=",this.memLastTime, ",cpuUsage=", memUsage, " sum=", this.memSum, " count=", this.memCnt, " Avg=", (this.memSum/int64(this.memCnt)), ", Max=", this.memMax);

	}
}

func (this *MeterSelf) GetMeterSelfStat() *value.MapValue {
	this.lock.Lock()
	defer func() {
		this.lock.Unlock()
		if x := recover(); x != nil {
			logutil.Println("WA424", "Recover GetMeterSelfStat ", x)
		}
	}()

	out := value.NewMapValue()

	packetTimeList := out.NewList("packetTime")
	packetList := out.NewList("packet")
	packetMaxList := out.NewList("packetMax")
	cpuTimeList := out.NewList("cpuTime")
	cpuList := out.NewList("cpu")
	cpuMaxList := out.NewList("cpuMax")
	memTimeList := out.NewList("memTime")
	memList := out.NewList("mem")
	memMaxList := out.NewList("memMax")

	packetEn := this.packetMap.Entries()
	for packetEn.HasMoreElements() {
		ctx := packetEn.NextElement().(*hmap.LongLongLinkedEntry)
		if ctx == nil {
			continue
		}
		packetTimeList.AddLong(ctx.GetKey())
		packetList.AddLong(ctx.GetValue())
	}

	packetMaxEn := this.packetMaxMap.Entries()
	for packetMaxEn.HasMoreElements() {
		ctx := packetMaxEn.NextElement().(*hmap.LongLongLinkedEntry)
		if ctx == nil {
			continue
		}
		packetMaxList.AddLong(ctx.GetValue())
	}

	cpuEn := this.cpuMap.Entries()
	for cpuEn.HasMoreElements() {
		ctx := cpuEn.NextElement().(*hmap.LongFloatLinkedEntry)
		if ctx == nil {
			continue
		}
		cpuTimeList.AddLong(ctx.GetKey())
		cpuList.Add(value.NewFloatValue(ctx.GetValue()))
	}

	cpuMaxEn := this.cpuMaxMap.Entries()
	for cpuMaxEn.HasMoreElements() {
		ctx := cpuMaxEn.NextElement().(*hmap.LongFloatLinkedEntry)
		if ctx == nil {
			continue
		}
		cpuMaxList.Add(value.NewFloatValue(ctx.GetValue()))
	}

	memEn := this.memMap.Entries()
	for memEn.HasMoreElements() {
		ctx := memEn.NextElement().(*hmap.LongLongLinkedEntry)
		if ctx == nil {
			continue
		}
		memTimeList.AddLong(ctx.GetKey())
		memList.AddLong(ctx.GetValue())
	}

	memMaxEn := this.memMaxMap.Entries()
	for memMaxEn.HasMoreElements() {
		ctx := memMaxEn.NextElement().(*hmap.LongLongLinkedEntry)
		if ctx == nil {
			continue
		}
		memMaxList.AddLong(ctx.GetValue())
	}

	return out
}

func (this *MeterSelf) PrintSelfStat() {

	logutil.Infoln("WA425", "Self Metering")
	logutil.Infoln("WA426", this.GetMeterSelfStat())
	//logutil.Infoln("NET Tot:%l Avg:%l Max:%1 Min:%1");
	//logutil.Infoln("CPU Tot:%l Avg:%l Max:%1 Min:%1");
	//logutil.Infoln("MEM Tot:%l Avg:%l Max:%1 Min:%1");
}

func MeterSelfMain() {
}
