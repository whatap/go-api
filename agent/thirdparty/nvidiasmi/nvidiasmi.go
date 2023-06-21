//go:build linux || windows || darwin
// +build linux windows darwin

package nvidiasmi

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/agent/secure"
)

const (
	DOCLINELIMIT = 50000
)

var (
	nvidiaexe              = "nvidia-smi"
	conf                   = config.GetConfig()
	NvidiaEnabled     bool = false
	NvidiaInitialized bool = false
	proc              *os.Process
)

func InitNvidiaWatch() {
	if !NvidiaInitialized {

		_, err := exec.LookPath(nvidiaexe)
		NvidiaEnabled = err == nil
		if NvidiaEnabled {
			go pollNvidiaPerf(nvidiaexe)
		}

		NvidiaEnabled = true
	}
}

func pollNvidiaPerf(exe string) {
	for {
		func() {
			if conf.NvidiasmiEnabled {
				_pollNvidiaPerf(exe)
			} else {
				time.Sleep(10 * time.Second)
			}

		}()
	}
}

func _pollNvidiaPerf(exe string) {
	cmd := exec.Command(exe, "-x", "-q", "-a", "-l", "5")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	err = cmd.Start()
	if err != nil {
		return
	}
	proc = cmd.Process
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		cmd.Wait()
	}()

	var b bytes.Buffer

	var linecount int
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		b.WriteString(line)
		if "</nvidia_smi_log>" == line {
			parseDoc(b.Bytes())
			b.Reset()
		}
		linecount += 1
		if linecount > DOCLINELIMIT {
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
			return
		}
	}
}

func parseDoc(bts []byte) {
	res := new(NvidiaSmi)
	err := xml.Unmarshal(bts, res)
	if err != nil {
		return
	}
	secu := secure.GetSecurityMaster()

	for _, gpu := range res.GPUS {
		p := pack.NewTagCountPack()

		p.Pcode = secu.PCODE
		p.Oid = secu.OID
		p.Time = dateutil.Now()
		p.Category = "server_nvidiasmi"

		p.Put("Timestamp", res.Timestamp)

		p.Tags.PutString("DriverVersion", res.DriverVersion)
		p.Tags.PutString("AttachedGpus", res.AttachedGpus)

		p.Put("ID", gpu.ID)
		p.Put("MemClockClocksGpu", gpu.MemClockClocksGpu)
		p.Put("L1Cache", gpu.L1Cache)
		p.Put("ProductName", gpu.ProductName)
		p.Put("FreeFbMemoryUsageGpu", ParseBytes(gpu.FreeFbMemoryUsageGpu))
		p.Put("PowerState", gpu.PowerState)
		p.Put("Free", ParseBytes(gpu.Free))
		p.Put("RetiredCountDoubleBitRetirementRetiredPagesGpu", gpu.RetiredCountDoubleBitRetirementRetiredPagesGpu)
		p.Put("ClocksThrottleReasonUnknown", gpu.ClocksThrottleReasonUnknown)
		p.Put("ClocksThrottleReasonApplicationsClocksSetting", gpu.ClocksThrottleReasonApplicationsClocksSetting)
		for i, proc := range gpu.Processes {

			p.Put(fmt.Sprint("GpuInstanceId", i), proc.GpuInstanceId)
			p.Put(fmt.Sprint("ComputeInstanceId", i), proc.ComputeInstanceId)
			p.Put(fmt.Sprint("Pid", i), proc.Pid)
			p.Put(fmt.Sprint("ProcessType", i), proc.ProcessType)
			p.Put(fmt.Sprint("ProcessName", i), proc.ProcessName)
			p.Put(fmt.Sprint("UsedMemory", i), proc.UsedMemory)
		}

		p.Put("MemClockApplicationsClocksGpu", gpu.MemClockApplicationsClocksGpu)
		p.Put("L2CacheSingleBitAggregateEccErrorsGpu", gpu.L2CacheSingleBitAggregateEccErrorsGpu)
		p.Put("CurrentLinkGen", gpu.CurrentLinkGen)
		p.Put("TotalSingleBitVolatileEccErrorsGpu", gpu.TotalSingleBitVolatileEccErrorsGpu)
		p.Put("TextureMemoryDoubleBitVolatileEccErrorsGpu", gpu.TextureMemoryDoubleBitVolatileEccErrorsGpu)
		p.Put("L1CacheSingleBitAggregateEccErrorsGpu", gpu.L1CacheSingleBitAggregateEccErrorsGpu)
		p.Put("PendingGom", gpu.PendingGom)
		p.Put("AutoBoostDefault", gpu.AutoBoostDefault)
		p.Put("GraphicsClockApplicationsClocksGpu", gpu.GraphicsClockApplicationsClocksGpu)
		p.Put("PciBusID", gpu.PciBusID)
		p.Put("PowerManagement", ParseBytes(gpu.PowerManagement))
		p.Put("DeviceMemoryDoubleBitAggregateEccErrorsGpu", gpu.DeviceMemoryDoubleBitAggregateEccErrorsGpu)
		p.Put("BoardID", gpu.BoardID)
		p.Put("DeviceMemoryDoubleBitVolatileEccErrorsGpu", gpu.DeviceMemoryDoubleBitVolatileEccErrorsGpu)
		for i, gclock := range gpu.SupportedGraphicsClock {
			p.Put(fmt.Sprint("SupportedGraphicsClock", i), gclock)
		}

		p.Put("PersistenceMode", gpu.PersistenceMode)
		p.Put("MemClock", gpu.MemClock)
		p.Put("GraphicsClockClocksGpu", gpu.GraphicsClockClocksGpu)
		p.Put("Used", ParseBytes(gpu.Used))
		p.Put("ImgVersion", gpu.ImgVersion)
		p.Put("UsedFbMemoryUsageGpu", ParseBytes(gpu.UsedFbMemoryUsageGpu))
		p.Put("TotalDoubleBitAggregateEccErrorsGpu", gpu.TotalDoubleBitAggregateEccErrorsGpu)
		p.Put("MinorNumber", gpu.MinorNumber)
		p.Put("ProductBrand", gpu.ProductBrand)
		p.Put("GraphicsClockDefaultApplicationsClocksGpu", gpu.GraphicsClockDefaultApplicationsClocksGpu)
		p.Put("TotalFbMemoryUsageGpu", ParseBytes(gpu.TotalFbMemoryUsageGpu))
		p.Put("RegisterFileDoubleBitVolatileEccErrorsGpu", gpu.RegisterFileDoubleBitVolatileEccErrorsGpu)
		p.Put("MinPowerLimit", parsePower(gpu.MinPowerLimit))
		p.Put("TxUtil", parsePct(gpu.TxUtil))
		p.Put("TextureMemory", gpu.TextureMemory)
		p.Put("RegisterFileDoubleBitAggregateEccErrorsGpu", gpu.RegisterFileDoubleBitAggregateEccErrorsGpu)
		p.Put("PerformanceState", gpu.PerformanceState)
		p.Put("CurrentDm", gpu.CurrentDm)
		p.Put("PciDeviceID", gpu.PciDeviceID)
		p.Put("AccountedProcesses", gpu.AccountedProcesses)
		p.Put("PendingRetirement", gpu.PendingRetirement)
		p.Put("TotalDoubleBitVolatileEccErrorsGpu", gpu.TotalDoubleBitVolatileEccErrorsGpu)
		p.Put("UUID", gpu.UUID)
		p.Put("PowerLimit", parsePower(gpu.PowerLimit))
		p.Put("ClocksThrottleReasonHwSlowdown", gpu.ClocksThrottleReasonHwSlowdown)
		p.Put("BridgeChipFw", gpu.BridgeChipFw)
		p.Put("ReplayCounter", gpu.ReplayCounter)
		p.Put("L2CacheDoubleBitAggregateEccErrorsGpu", gpu.L2CacheDoubleBitAggregateEccErrorsGpu)
		p.Put("ComputeMode", gpu.ComputeMode)
		p.Put("FanSpeed", gpu.FanSpeed)
		p.Put("Total", ParseBytes(gpu.Total))
		p.Put("SmClock", gpu.SmClock)
		p.Put("RxUtil", parsePct(gpu.RxUtil))
		p.Put("GraphicsClock", gpu.GraphicsClock)
		p.Put("PwrObject", gpu.PwrObject)
		p.Put("PciBus", gpu.PciBus)
		p.Put("DecoderUtil", parsePct(gpu.DecoderUtil))
		p.Put("PciSubSystemID", gpu.PciSubSystemID)
		p.Put("MaxLinkGen", gpu.MaxLinkGen)
		p.Put("BridgeChipType", gpu.BridgeChipType)
		p.Put("SmClockClocksGpu", gpu.SmClockClocksGpu)
		p.Put("CurrentEcc", gpu.CurrentEcc)
		p.Put("PowerDraw", parsePower(gpu.PowerDraw))
		p.Put("CurrentLinkWidth", gpu.CurrentLinkWidth)
		p.Put("AutoBoost", gpu.AutoBoost)
		p.Put("GpuUtil", parsePct(gpu.GpuUtil))
		p.Put("PciDevice", gpu.PciDevice)
		p.Put("RegisterFile", gpu.RegisterFile)
		p.Put("L2Cache", gpu.L2Cache)
		p.Put("L1CacheDoubleBitAggregateEccErrorsGpu", gpu.L1CacheDoubleBitAggregateEccErrorsGpu)
		p.Put("RetiredCount", parseInt(gpu.RetiredCount))
		p.Put("PendingDm", gpu.PendingDm)
		p.Put("AccountingModeBufferSize", gpu.AccountingModeBufferSize)
		p.Put("GpuTempSlowThreshold", parseTemp(gpu.GpuTempSlowThreshold))
		p.Put("OemObject", gpu.OemObject)
		p.Put("TextureMemorySingleBitAggregateEccErrorsGpu", gpu.TextureMemorySingleBitAggregateEccErrorsGpu)
		p.Put("RegisterFileSingleBitAggregateEccErrorsGpu", gpu.RegisterFileSingleBitAggregateEccErrorsGpu)
		p.Put("MaxLinkWidth", gpu.MaxLinkWidth)
		p.Put("TextureMemoryDoubleBitAggregateEccErrorsGpu", gpu.TextureMemoryDoubleBitAggregateEccErrorsGpu)
		p.Put("ClocksThrottleReasonGpuIdle", gpu.ClocksThrottleReasonGpuIdle)
		p.Put("MultigpuBoard", gpu.MultigpuBoard)
		p.Put("GpuTempMaxThreshold", parseTemp(gpu.GpuTempMaxThreshold))
		p.Put("MaxPowerLimit", parsePower(gpu.MaxPowerLimit))
		p.Put("L2CacheDoubleBitVolatileEccErrorsGpu", gpu.L2CacheDoubleBitVolatileEccErrorsGpu)
		p.Put("PciDomain", gpu.PciDomain)
		p.Put("MemClockDefaultApplicationsClocksGpu", gpu.MemClockDefaultApplicationsClocksGpu)
		p.Put("VbiosVersion", gpu.VbiosVersion)
		p.Put("RetiredPageAddresses", gpu.RetiredPageAddresses)
		p.Put("GpuTemp", parseTemp(gpu.GpuTemp))
		p.Put("AccountingMode", gpu.AccountingMode)
		p.Put("L1CacheDoubleBitVolatileEccErrorsGpu", gpu.L1CacheDoubleBitVolatileEccErrorsGpu)
		p.Put("DeviceMemorySingleBitAggregateEccErrorsGpu", gpu.DeviceMemorySingleBitAggregateEccErrorsGpu)
		p.Put("DisplayActive", gpu.DisplayActive)
		p.Put("DefaultPowerLimit", parsePower(gpu.DefaultPowerLimit))
		p.Put("EncoderUtil", parsePct(gpu.EncoderUtil))
		p.Put("Serial", gpu.Serial)
		p.Put("EnforcedPowerLimit", parsePower(gpu.EnforcedPowerLimit))
		p.Put("RetiredPageAddressesDoubleBitRetirementRetiredPagesGpu", gpu.RetiredPageAddressesDoubleBitRetirementRetiredPagesGpu)
		p.Put("EccObject", gpu.EccObject)
		for i, v := range gpu.Value {
			p.Put(fmt.Sprint("Value", i), v)
		}

		p.Put("DisplayMode", gpu.DisplayMode)
		p.Put("DeviceMemory", gpu.DeviceMemory)
		p.Put("PendingEcc", gpu.PendingEcc)
		p.Put("ClocksThrottleReasonSwPowerCap", parsePower(gpu.ClocksThrottleReasonSwPowerCap))
		p.Put("TotalSingleBitAggregateEccErrorsGpu", ParseBytes(gpu.TotalSingleBitAggregateEccErrorsGpu))
		p.Put("CurrentGom", gpu.CurrentGom)
		p.Put("MemoryUtil", parsePct(gpu.MemoryUtil))

		data.SendHide(p)
	}
}

func parsePct(src string) (ret float32) {
	ret = 0
	trimmed := strings.ReplaceAll(src, "%", "")
	trimmed = strings.TrimSpace(trimmed)

	v, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return
	}
	ret = float32(v)

	return
}

func parseInt(src string) (ret int32) {
	ret = 0
	trimmed := strings.TrimSpace(src)

	v, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return
	}
	ret = int32(v)

	return
}

func parseTemp(src string) (ret float32) {
	ret = 0
	trimmed := strings.ReplaceAll(src, "C", "")
	trimmed = strings.ReplaceAll(trimmed, "F", "")
	trimmed = strings.TrimSpace(trimmed)

	v, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return
	}
	ret = float32(v)

	return
}

func parsePower(src string) (ret float32) {
	ret = 0
	trimmed := strings.ReplaceAll(src, "W", "")
	trimmed = strings.TrimSpace(trimmed)

	v, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return
	}
	ret = float32(v)

	return
}

func toBytes(unit string) int64 {
	var ret int64
	switch unit {
	case "TB":
		ret = 1000000000000
	case "tB", "TiB":
		ret = 0x10000000000
	case "GB":
		ret = 1000000000
	case "gB", "GiB":
		ret = 0x40000000
	case "MB":
		ret = 1000000
	case "mB", "MiB":
		ret = 0x100000
	case "kB":
		ret = 1000
	case "KB", "KiB":
		ret = 0x400
	default:
		ret = 1
	}
	return ret
}

func ParseBytes(src string) int64 {
	words := strings.Fields(src)
	if len(words) == 2 {
		val, _ := strconv.ParseInt(words[0], 10, 64)
		val *= toBytes(words[1])

		return val
	} else if len(words) == 2 {
		val, _ := strconv.ParseInt(words[0], 10, 64)

		return val
	}

	return 0
}

func OnShutdown() {
	if proc != nil {
		proc.Kill()
	}
}
