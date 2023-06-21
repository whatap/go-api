package watch

import (
	"math"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/whatap/golib/util/hmap"
	"github.com/whatap/golib/util/stringutil"
	"github.com/whatap/go-api/agent/agent/config"
	langconf "github.com/whatap/go-api/agent/lang/conf"
	"github.com/whatap/go-api/agent/util/logutil"
)

type WatchLogManager struct {
	watchEnabled bool
	conf         *config.Config

	table *hmap.StringKeyLinkedMap
}

var watchLogManager *WatchLogManager
var watchLogManagerMutex = sync.Mutex{}

func GetInstance() *WatchLogManager {
	watchLogManagerMutex.Lock()
	defer watchLogManagerMutex.Unlock()
	if watchLogManager != nil {
		return watchLogManager
	}
	watchLogManager = new(WatchLogManager)
	watchLogManager.watchEnabled = false
	watchLogManager.conf = config.GetConfig()
	watchLogManager.table = hmap.NewStringKeyLinkedMap()

	langconf.AddConfObserver("WatchLogManager", watchLogManager)

	go watchLogManager.run()

	return watchLogManager
}

func (this *WatchLogManager) Run() {
	this.resetDogList(this.conf.WatchLogEnabled != this.watchEnabled)
	this.watchEnabled = this.conf.WatchLogEnabled

	DebugAppLogParser = this.conf.DebugLogSinkEnabled
	if len(this.conf.TxIdTag) > 0 {
		TxIdTag = this.conf.TxIdTag
	}

	if len(this.conf.AppLogCategory) > 0 {
		AppLogCategory = this.conf.AppLogCategory
	}

	if len(this.conf.AppLogPattern) > 0 {
		AppLogPattern, _ = regexp.Compile(this.conf.AppLogPattern)
	}

	if this.conf.LogSendThreshold > 0 {
		LogSendThreshold = this.conf.LogSendThreshold
	}
}

func (this *WatchLogManager) run() {
	this.resetDogList(true)
	this.watchEnabled = this.conf.WatchLogEnabled

	for {
		if this.watchEnabled == false {
			time.Sleep(5 * time.Second)
			continue
		}
		time.Sleep(time.Duration(int(this.conf.WatchLogCheckInterval)) * time.Millisecond)
		this.process()
	}
}

func (this *WatchLogManager) process() {
	en := this.table.Values()
	for en.HasMoreElements() {
		var dog *WatchLog
		if el := en.NextElement(); el != nil {
			dog = el.(*WatchLog)
			func() {
				defer func() {
					if r := recover(); r != nil {
						dog.Stop()
					}
				}()
				if dog.IsActive() {
					dog.Process()
				}
			}()
		}
	}
}

func (this *WatchLogManager) resetDogList(reset bool) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA-LOGS-201", "resetDogList Recover", r, ",stack=", string(debug.Stack()))
		}
	}()
	ids := make([]string, 0)
	if len(this.conf.LogSinkFiles) > 0 {
		for _, it := range this.conf.LogSinkFiles {
			ids = append(ids, strings.TrimSpace(it))
		}
		sl := sort.StringSlice(ids)
		sl.Sort()
		for i := 0; i < len(sl); i++ {
			id := sl[i]
			enabled := true
			file := sl[i]
			words := make([]string, 0)
			checkInterval := this.conf.LogSinkInterval

			if this.conf.DebugLogSinkEnabled {
				logutil.Println("WA-LOGS-202", "resetDogList logsink ", "id=", id, ",file=", file, ",enabled=", enabled)
			}
			if file != "" {
				dog := NewWatchLog(id)
				this.table.Put(id, dog)

				dog.Config(id, file)
				dog.Words = words
				dog.CheckInterval = int(math.Max(float64(checkInterval), float64(1000)))
				if reset {
					dog.Reset()
				}

				if enabled {
					if this.conf.DebugLogSinkEnabled {
						logutil.Println("WA-LOGS-203", "Activate ", "id=", id, ",file=", file, ",enabled=", enabled)
					}
					dog.Activate()
				} else {
					if this.conf.DebugLogSinkEnabled {
						logutil.Println("WA-LOGS-204", "Stop ", "id=", id, ",file=", file, ",enabled=", enabled)
					}
					dog.Stop()
				}
			}
		}
	} else {
		for k, _ := range config.FilterPrefix("watchlog.") {
			if strings.HasSuffix(k, ".enabled") {
				id := stringutil.Substring(k, "watchlog.", ".enabled")
				ids = append(ids, id)
			}
		}

		//	sort.Slice(ids, func(i, j string) bool {
		//		returnm i < j
		//	})
		//
		sl := sort.StringSlice(ids)
		sl.Sort()
		for i := 0; i < len(sl); i++ {
			id := sl[i]
			enabled := config.GetBoolean("watchlog."+id+".enabled", false)
			file := config.GetValue("watchlog." + id + ".file")
			words := config.GetStringArray("watchlog."+id+".words", ",")
			checkInterval := config.GetInt("watchlog."+id+".check_interval", 1000)

			if this.conf.DebugLogSinkEnabled {
				logutil.Println("WA-LOGS-202", "resetDogList ", "id=", id, ",file=", file, ",enabled=", enabled)
			}
			// DEBUG
			//if file != "" && len(words) > 0 {
			if file != "" {
				dog := NewWatchLog(id)
				this.table.Put(id, dog)

				dog.Config(id, file)
				dog.Words = words
				dog.CheckInterval = int(math.Max(float64(checkInterval), float64(1000)))
				if reset {
					dog.Reset()
				}

				if enabled {
					if this.conf.DebugLogSinkEnabled {
						logutil.Println("WA-LOGS-203", "Activate ", "id=", id, ",file=", file, ",enabled=", enabled)
					}
					dog.Activate()
				} else {
					if this.conf.DebugLogSinkEnabled {
						logutil.Println("WA-LOGS-204", "Stop ", "id=", id, ",file=", file, ",enabled=", enabled)
					}
					dog.Stop()
				}
			}
		}
	}
	// 삭제된 id들에 대해서는 삭제한다.
	en1 := this.table.Keys()
	for en1.HasMoreElements() {
		id := en1.NextString()
		exists := false
		for _, it := range ids {
			if id == it {
				exists = true
				break
			}
		}
		if exists == false {
			this.table.Remove(id)
		}
	}
}
