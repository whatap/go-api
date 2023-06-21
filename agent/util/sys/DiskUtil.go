package sys

import (
	//"log"
	"github.com/shirou/gopsutil/disk"
	"github.com/whatap/go-api/agent/util/logutil"
)

func GetSysDiskUsedPercent(path string) float64 {

	stat, err := disk.Usage(path)
	if err != nil {
		logutil.Println("WA851", " Usage Error path=", path, ",Error=", err)
		return 0
	}

	return float64(stat.UsedPercent)
}
