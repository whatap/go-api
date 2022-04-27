package task

import (
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/whatap/go-api/common/lang/pack"
	whatapnet "github.com/whatap/go-api/common/net"
	"github.com/whatap/go-api/common/util/stringutil"
)

type TaskGoRuntime struct {
	lastSampleTime time.Time
	lastPauseNs    uint64
	lastNumGc      uint32
}

func (this *TaskGoRuntime) Process(now int64) {
	udpClient := whatapnet.GetUdpClient()

	p := pack.NewTagCountPack()
	p.Time = now

	p.Category = "go_runtime"
	p.PutTag("pid", strconv.Itoa(os.Getpid()))
	p.PutTag("cmd", os.Args[0])
	sb := stringutil.NewStringBuffer()
	for i, v := range os.Args {
		if i > 0 {
			sb.Append(" ")
		}
		sb.Append(v)
	}
	p.PutTag("cmdFull", sb.ToString())

	p.Put("NumCpu", runtime.NumCPU())
	p.Put("NumCgoCall", runtime.NumCgoCall())
	p.Put("NumGoroutine", runtime.NumGoroutine())

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	p.Put("Alloc", m.Alloc)
	p.Put("TotalAlloc", m.TotalAlloc)
	p.Put("Sys", m.Sys)
	p.Put("Lookups", m.Lookups)
	p.Put("Mallocs", m.Mallocs)
	p.Put("Frees", m.Frees)
	p.Put("HeapAlloc", m.HeapAlloc)
	p.Put("HeapSys", m.HeapSys)
	p.Put("HeapIdel", m.HeapIdle)
	p.Put("HeapInuse", m.HeapInuse)
	p.Put("HeapReleased", m.HeapReleased)
	p.Put("HeapObjects", m.HeapObjects)
	p.Put("StackInuse", m.StackInuse)
	p.Put("StackSys", m.StackSys)
	p.Put("MSpanInuse", m.MSpanInuse)
	p.Put("MSpanSys", m.MSpanSys)
	p.Put("MCacheInuse", m.MCacheInuse)
	p.Put("MCacheSys", m.MCacheSys)
	p.Put("BuckHashSys", m.BuckHashSys)
	p.Put("GCSys", m.GCSys)
	p.Put("OtherSys", m.OtherSys)
	p.Put("NextGC", m.NextGC)
	p.Put("LastGC", nanoToMilli(m.LastGC))
	p.Put("PauseTotalNs", float64(m.PauseTotalNs)/float64(1000000))
	p.Put("NumGC", m.NumGC)
	p.Put("NumForcedGC", m.NumForcedGC)

	var gcPerSecond float64
	diffTime := time.Now().Sub(this.lastSampleTime).Seconds()

	if this.lastNumGc > 0 {
		lastSample := m.NumGC - this.lastNumGc
		diff := float64(lastSample)
		gcPerSecond = diff / diffTime
	}

	p.Put("GcPerSecond", gcPerSecond)

	var gcPausePerSecond float64
	if this.lastPauseNs > 0 {
		lastSample := m.PauseTotalNs - this.lastPauseNs
		gcPausePerSecond = float64(lastSample) / float64(time.Millisecond) / float64(diffTime)
	}
	p.Put("GcPausePerSecond", gcPausePerSecond)

	udpClient.SendRelay(p, false)

	this.lastSampleTime = time.Now()
	this.lastNumGc = m.NumGC
	this.lastPauseNs = m.PauseTotalNs
}

func bToKb(b uint64) uint64 {
	return b / 1024
}
func nanoToMilli(b uint64) uint64 {
	return b / 1000000
}
