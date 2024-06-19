package pprof

import (
	"net/http"
	httpPProf "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/whatap/go-api/agent/agent/config"
	langconf "github.com/whatap/go-api/agent/lang/conf"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/golib/util/dateutil"
)

type SelfPProf struct {
	pprofFile        *os.File
	isStart          bool
	isStartCpuPProf  bool
	isStartHttpPProf bool
	conf             *config.Config

	httpSrv *http.Server
}

var selfPProf *SelfPProf

func GetSlefPProf() *SelfPProf {
	if selfPProf == nil {
		selfPProf = new(SelfPProf)
		selfPProf.conf = config.GetConfig()
		selfPProf.isStart = false
		selfPProf.httpSrv = nil
		selfPProf.isStartCpuPProf = false
		selfPProf.isStartHttpPProf = false
		selfPProf.Run()
		langconf.AddConfObserver("SelfPProf", selfPProf)
	}

	return selfPProf
}

// implements langconf.Runnable
func (this *SelfPProf) Run() {
	if this.conf.PProfEnabled && !this.isStart {
		go this.run()
	}
}

func (this *SelfPProf) run() {
	if !this.isStart {
		this.isStart = true
		logutil.Println("WAPPROF001", "Start SelfPProf")

		for {
			// shutdown
			if config.GetConfig().Shutdown {
				logutil.Infoln("WA211-07", "Shutdown SelfPProf")
				this.StopCpuPProf()
				this.StopHttpPProf()
				break
			}

			// diable then stop goroutine
			if !this.conf.PProfEnabled {
				logutil.Println("WAPPROF002", "Stop SelfPProf")
				this.StopCpuPProf()
				this.StopHttpPProf()
				break
			}
			this.process()
			time.Sleep(time.Duration(this.conf.PProfInterval) * time.Millisecond)
		}
		logutil.Println("WAPPROF001-1", "Stop SelfPProf")
		this.isStart = false
	}
}

func (this *SelfPProf) process() {
	logutil.Println("WAPPROF003", "NumGoroutine=", this.GetNumGoroutine(), ", NumCgoCall=", this.GetNumCgocall())
	this.PrintMemUsage()
	if this.conf.PProfCpuEnabled {
		if !this.isStartCpuPProf {
			this.StartCpuPProf()
		}
	} else {
		if this.isStartCpuPProf {
			this.StopCpuPProf()
		}
	}
	if this.conf.PProfHttpEnabled {
		if !this.isStartHttpPProf {
			go this.StartHttpPProf()
		}
	} else {
		if this.isStartHttpPProf {
			this.StopHttpPProf()
		}
	}
}

func (this *SelfPProf) GetNumGoroutine() int {
	return runtime.NumGoroutine()
}

func (this *SelfPProf) GetNumCgocall() int64 {
	return runtime.NumCgoCall()
}

// go tool pprof file
func (this *SelfPProf) StartCpuPProf() {
	home := config.GetWhatapHome()

	if _, err := os.Stat(filepath.Join(home, "logs")); err != nil {
		if os.IsNotExist(err) {
			os.Mkdir(filepath.Join(home, "logs"), os.ModePerm)
		}
	}
	logutil.Println("WAPROF10001", "Start CpuPProf file=", "cpupprof"+dateutil.Ymdhms(dateutil.SystemNow())+".pprof")
	if f, err := os.OpenFile(filepath.Join(home, "logs", "cpupprof"+dateutil.Ymdhms(dateutil.SystemNow())+".pprof"), os.O_CREATE|os.O_WRONLY, 0666); err == nil {
		this.pprofFile = f
	} else {
		this.pprofFile = nil
		logutil.Println("WAPPROF100", "Error Start open file ", err)
	}
	if this.pprofFile != nil {
		if err := pprof.StartCPUProfile(this.pprofFile); err == nil {
			logutil.Println("WAPPROF101", "Start cpu pprof")
			this.isStartCpuPProf = true
		} else {
			logutil.Println("WAPPROF102", "Error Start cpu pprof ", err)
		}
	}
}
func (this *SelfPProf) StopCpuPProf() {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WAPPROF002", "Stop cpu pprof recover ", r)
		}
	}()
	logutil.Println("WAPPROF104", "Stop cpu pprof")
	pprof.StopCPUProfile()
	this.isStartCpuPProf = false
	this.pprofFile.Close()
	this.pprofFile = nil
}
func (this *SelfPProf) StartHttpPProf() {
	srvMux := http.NewServeMux()

	logutil.Println("WAPPROF105", "Start http pprof ", this.conf.PProfHttpAddress)
	srvMux.HandleFunc("/debug/pprof/", httpPProf.Index)
	srvMux.HandleFunc("/debug/pprof/{category}", httpPProf.Index)
	srvMux.HandleFunc("/debug/pprof/cmdline", httpPProf.Cmdline)
	srvMux.HandleFunc("/debug/pprof/profile", httpPProf.Profile)
	srvMux.HandleFunc("/debug/pprof/symbol", httpPProf.Symbol)
	srvMux.HandleFunc("/debug/pprof/trace", httpPProf.Trace)
	if this.httpSrv != nil {
		this.httpSrv.Close()
		this.httpSrv = nil
	}
	this.httpSrv = &http.Server{Addr: this.conf.PProfHttpAddress, Handler: srvMux}
	this.isStartHttpPProf = true
	err := this.httpSrv.ListenAndServe()
	if err != nil {
		this.isStartHttpPProf = false
		logutil.Println("WAPPROF105-1", "Error ", err)
	}
	logutil.Println("WAPPROF105-2", "Close http pprof")
}
func (this *SelfPProf) StopHttpPProf() {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WAPPROF107", "Stop http pprof recover ", r)
		}
	}()
	logutil.Println("WAPPROF106", "Stop http pprof")
	if this.httpSrv != nil {
		if err := this.httpSrv.Close(); err != nil {
			logutil.Println("WAPPROF106-1", "Error ", err)
			return
		}
		this.httpSrv = nil
		this.isStartHttpPProf = false
	}
}

func (this *SelfPProf) PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	logutil.Printf("WAPPROF005", "Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v\n", bToMb(m.Alloc), bToMb(m.TotalAlloc), bToMb(m.Sys), m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
