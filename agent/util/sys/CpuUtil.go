package sys

import (
	//"log"
	"bytes"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/cpu"

	"github.com/whatap/golib/util/cmdutil"
	"github.com/whatap/go-api/agent/util/logutil"
)

type SysCpuInfo struct {
	Total     float64
	User      float64
	System    float64
	Idle      float64
	Nice      float64
	Iowait    float64
	Irq       float64
	Softirq   float64
	Steal     float64
	Guest     float64
	GuestNice float64
	Stolen    float64

	Percent float64
}

func GetCPUNum() int {
	return runtime.NumCPU()
}

func GetCPUType() (string, error) {
	var cpuType string
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA839", " GetCPUType Recover", r)
			cpuType = ""
		}
	}()

	cpuInfoStat, err := cpu.Info()

	if err != nil {
		return "", err
	}

	for _, it := range cpuInfoStat {
		cpuType = it.ModelName
		break
	}

	return cpuType, nil
}

func GetCPUPercent() *SysCpuInfo {
	var p *SysCpuInfo
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA831", "GetCPUTimes Recover", r)
			p = nil
		}
	}()

	p = new(SysCpuInfo)

	// sum  per cpu
	percpu := false

	//func Percent(interval time.Duration, percpu bool) ([]float64, error)
	percent, _ := cpu.Percent(0, percpu)
	//if err != nil {

	for _, it := range percent {
		p.Percent += it
		//fmt.Println("GetCPUPercent range =%d", it)
	}

	//fmt.Println("GetCPUPercent=%d", p.Percent)

	return p

	//} else {
	//	logutil.Println("GetCPUPercent Error ", err)
	//}
	//return nil
}

//	type TimesStat struct {
//	   CPU       string  `json:"cpu"`
//	   User      float64 `json:"user"`
//	   System    float64 `json:"system"`
//	   Idle      float64 `json:"idle"`
//	   Nice      float64 `json:"nice"`
//	   Iowait    float64 `json:"iowait"`
//	   Irq       float64 `json:"irq"`
//	   Softirq   float64 `json:"softirq"`
//	   Steal     float64 `json:"steal"`
//	   Guest     float64 `json:"guest"`
//	   GuestNice float64 `json:"guestNice"`
//	   Stolen    float64 `json:"stolen"`
//	}
func GetCPUTimes() *SysCpuInfo {
	var p *SysCpuInfo
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA832", "GetCPUTimes Recover", r)
			p = nil
		}
	}()

	p = new(SysCpuInfo)

	percpu := false

	// sum  per cpu
	//func Times(percpu bool) ([]TimesStat, error)
	times, _ := cpu.Times(percpu)
	//if err != nil {

	for _, it := range times {
		p.Total += it.Total()
		p.User += it.User
		p.System += it.System
		p.Idle += it.Idle
		p.Nice += it.Nice
		p.Iowait += it.Iowait
		p.Irq += it.Irq
		p.Softirq += it.Softirq
		p.Steal += it.Steal
		p.Guest += it.Guest
		p.GuestNice += it.GuestNice
		//p.Stolen += it.Stolen
		//fmt.Println("GetCPUTimes range User=%d, system=%d", p.User, p.System)
	}
	//fmt.Println("GetCPUTimes", p)

	return p
	//} else {
	//	logutil.Println("GetCPUTimes Error ", err)
	//}
	//return nil

}

// 지정된 이름의 프로세스의 Cpu 사용 Percent 의 합
func GetCPUSumByProcess(processName string) float64 {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA833", "GetCPUSumByProcess Recover", r)
		}
	}()

	// httpd CPU 사용량
	//out, err := exec.Command("ps aux | grep httpd | awk '{print $3}' | awk '{total = total + $1} END {print total}'").Output()
	//out, err := exec.Command("ps", "aux | grep httpd | awk '{print $3}' | awk '{total = total + $1} END {print total}'").Output()
	c1 := exec.Command("ps", "aux")
	c2 := exec.Command("grep", processName)
	c3 := exec.Command("awk", "{print $3}")
	c4 := exec.Command("awk", "{total = total + $1} END {print total}")

	// Run the pipeline
	output, stderr, err := cmdutil.Pipeline(c1, c2, c3, c4)
	if err != nil {
		logutil.Printf("WA834", "Error : %s", err)
		return 0
	}

	// Print the stdout, if any
	if len(output) <= 0 {
		return 0
	}

	// Print the stderr, if any
	if len(stderr) > 0 {
		logutil.Printf("WA835", "(stderr)\n%s", stderr)
	}

	cpu, err := strconv.ParseFloat(strings.Replace(string(output), "\n", "", -1), 1)
	if err != nil {
		logutil.Println("WA836", "Process: ParseFloat ", err)
		return 0
	}

	return cpu
}

func GetProcessStack(pid int) string {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA837", "GetProcessStack Recover", r)
		}
	}()

	if pid <= 0 {
		return ""
	}

	var output bytes.Buffer

	cmd := exec.Command("pstack", strconv.Itoa(pid))
	cmd.Stdout = &output

	err := cmd.Run()
	if err != nil {
		logutil.Println("WA838", "GetProcessStack Error", err)
		return ""
	}

	// Print the stdout, if any
	if output.Len() <= 0 {
		return ""
	}

	//fmt.Println("GetProcessStack out=", output.String())

	return output.String()
}
