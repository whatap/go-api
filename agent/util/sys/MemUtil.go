package sys

import (
	//"log"
	"os/exec"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/mem"

	"github.com/whatap/golib/util/cmdutil"
	"github.com/whatap/go-api/agent/util/logutil"
)

type SysMemInfo struct {
	VirtualTotal       uint64
	VirtualUsed        uint64
	VirtualFree        uint64
	VirtualUsedPercent float64
	SwapTotal          uint64
	SwapUsed           uint64
	SwapFree           uint64
	SwapUsedPercent    float64
}

func GetMemorySize() (int64, error) {
	var total uint64
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA846", " GetMemorySize Recover", r)
			total = 0
		}
	}()

	sysMem := GetSysMemInfo()

	total = sysMem.VirtualTotal

	return int64(total), nil
}

func GetSysMemInfo() *SysMemInfo {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA841", " GetSysMemInfo Recover", r)
		}
	}()

	p := new(SysMemInfo)

	v, _ := mem.VirtualMemory()
	//if err != nil {
	p.VirtualTotal = v.Total
	p.VirtualUsed = v.Used
	p.VirtualFree = v.Free
	p.VirtualUsedPercent = v.UsedPercent
	//}

	s, _ := mem.SwapMemory()
	//if err != nil {
	p.SwapTotal = s.Total
	p.SwapUsed = s.Used
	p.SwapFree = s.Free
	p.SwapUsedPercent = s.UsedPercent
	//}

	//logutil.Println("WA841", "SysMemInfo:",p)
	return p
}

func GetMemSumByProcess(processName string) int {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA842", "GetMemSumByProcess Recover", r)
		}
	}()

	// httpd CPU 사용량
	//out, err := exec.Command("ps aux | grep httpd | awk '{print $3}' | awk '{total = total + $1} END {print total}'").Output()
	//out, err := exec.Command("ps", "aux | grep httpd | awk '{print $3}' | awk '{total = total + $1} END {print total}'").Output()
	c1 := exec.Command("ps", "aux")
	c2 := exec.Command("grep", processName)
	c3 := exec.Command("awk", "{print $6}")
	c4 := exec.Command("awk", "{total = total + $1} END {print total}")

	// Run the pipeline
	output, stderr, err := cmdutil.Pipeline(c1, c2, c3, c4)
	if err != nil {
		logutil.Printf("WA843", "Error : %s", err)
		return 0
	}

	// Print the stdout, if any
	if len(output) <= 0 {
		return 0
	}

	// Print the stderr, if any
	if len(stderr) > 0 {
		logutil.Printf("WA844", "(stderr)\n%s", stderr)
	}

	mem, err := strconv.ParseFloat(strings.Replace(string(output), "\n", "", -1), 1)
	if err != nil {
		logutil.Println("WA845", "Process: ParseFloat ", err)
		return 0
	}

	return int(mem)
}

//type SwapMemoryStat struct {
//    Total       uint64  `json:"total"`
//    Used        uint64  `json:"used"`
//    Free        uint64  `json:"free"`
//    UsedPercent float64 `json:"usedPercent"`
//    Sin         uint64  `json:"sin"`
//    Sout        uint64  `json:"sout"`
//}

//type VirtualMemoryStat struct {
//    // Total amount of RAM on this system
//    Total uint64 `json:"total"`
//
//    // RAM available for programs to allocate
//    //
//    // This value is computed from the kernel specific values.
//    Available uint64 `json:"available"`
//
//    // RAM used by programs
//    //
//    // This value is computed from the kernel specific values.
//    Used uint64 `json:"used"`
//
//    // Percentage of RAM used by programs
//    //
//    // This value is computed from the kernel specific values.
//    UsedPercent float64 `json:"usedPercent"`
//
//    // This is the kernel's notion of free memory; RAM chips whose bits nobody
//    // cares about the value of right now. For a human consumable number,
//    // Available is what you really want.
//    Free uint64 `json:"free"`
//
//    // OS X / BSD specific numbers:
//    // http://www.macyourself.com/2010/02/17/what-is-free-wired-active-and-inactive-system-memory-ram/
//    Active   uint64 `json:"active"`
//    Inactive uint64 `json:"inactive"`
//    Wired    uint64 `json:"wired"`
//
//    // Linux specific numbers
//    // https://www.centos.org/docs/5/html/5.1/Deployment_Guide/s2-proc-meminfo.html
//    // https://www.kernel.org/doc/Documentation/filesystems/proc.txt
//    Buffers      uint64 `json:"buffers"`
//    Cached       uint64 `json:"cached"`
//    Writeback    uint64 `json:"writeback"`
//    Dirty        uint64 `json:"dirty"`
//    WritebackTmp uint64 `json:"writebacktmp"`
//    Shared       uint64 `json:"shared"`
//    Slab         uint64 `json:"slab"`
//    PageTables   uint64 `json:"pagetables"`
//    SwapCached   uint64 `json:"swapcached"`
//}
