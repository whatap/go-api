package sys

import (
	"os"

	"github.com/shirou/gopsutil/process"
	//"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/util/logutil"
)

func GetProcessInfo(processName string) (map[string]float64, map[int32]float64) {
	proc_info := make(map[string]float64)
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA862", "GetProcessInfo Recover", r)
		}
	}()
	myPid := os.Getpid()
	cpuTime := make(map[int32]float64)
	// cputime of proc, psutil
	if proc, err := process.NewProcess(int32(myPid)); err == nil {
		if timeInfo, err1 := proc.Times(); err1 == nil {
			// timeInfo.Total() is seconds
			cTime := timeInfo.Total() * 1000
			cpuTime[int32(myPid)] = cTime
		}
		if cpuP, err1 := proc.CPUPercent(); err1 == nil {
			proc_info["cpu_proc"] = cpuP
		}
	} else {
		logutil.Println("WA862-01", "GetProcessInfo error", err)
	}
	return proc_info, cpuTime
}
