package task

import (
	"runtime"

	"github.com/whatap/go-api/common/lang/pack"
	whatapnet "github.com/whatap/go-api/common/net"
)

type TaskGoRuntime struct {
}

func (this *TaskGoRuntime) Process(now int64) {
	udpClient := whatapnet.GetUdpClient()

	p := pack.NewTagCountPack()
	p.Time = now

	p.Category = "go_runtime"
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
	p.Put("PauseTotalNs", nanoToMilli(m.PauseTotalNs))
	p.Put("NumGC", m.NumGC)
	p.Put("NumForcedGC", m.NumForcedGC)

	udpClient.SendRelay(p, false)
}

func bToKb(b uint64) uint64 {
	return b / 1024
}
func nanoToMilli(b uint64) uint64 {
	return b / 1000000
}
