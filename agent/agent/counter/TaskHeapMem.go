package counter

import (
	//"log"
	"runtime"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/go-api/agent/util/sys"
	"github.com/whatap/golib/lang/pack"
)

type TaskHeapMem struct {
}

func NewTaskHeapMem() *TaskHeapMem {
	p := new(TaskHeapMem)
	return p
}
func (this *TaskHeapMem) process(p *pack.CounterPack1) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA331", "process Recover", r)
		}
	}()
	conf := config.GetConfig()
	if !conf.CounterEnabledHeap_ {
		if conf.CounterLogEnabled {
			logutil.Println("WA331-01", "Disable counter, heap")
		}
		return
	} else {
		if conf.CounterLogEnabled {
			logutil.Println("WA331-02", "Start counter, heap")
		}
	}

	sysMem := sys.GetSysMemInfo()

	total := sysMem.VirtualTotal //+ sysMem.SwapTotal
	//free := sysMem.VirtualFree + sysMem.SwapFree

	p.HeapTot = int64(total)

	// TODO
	// golang runtime heap use
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	// fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	// fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	// fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	// fmt.Printf("\tNumGC = %v\n", m.NumGC)

	p.HeapUse = int64(m.HeapInuse) + int64(m.StackInuse)
	p.HeapPerm = 0

	//logutil.Println("WA332", "HeapTot=", p.HeapTot, "HeapUse=", p.HeapUse, "HeapPerm=", p.HeapPerm, "heapUse=", p.HeapUse)

	//		long total = Runtime.getRuntime().totalMemory();
	//		long free = Runtime.getRuntime().freeMemory();
	//		long used = total - free;
	//		p.heap_tot = (int) (total / 1024);
	//		p.heap_use = (int) (used / 1024);
	//		try {
	//			p.heap_pending_finalization = membean.getObjectPendingFinalizationCount();
	//
	//			if (permGenBean != null) {
	//				MemoryUsage usage = permGenBean.getUsage();
	//				used = usage.getUsed();
	//				p.heap_perm = (int) (used / 1024);
	//			}
	//		} catch (Exception e) {
	//		}
	//
	//	}

}
