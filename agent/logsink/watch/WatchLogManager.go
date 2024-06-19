package watch

import (
	"math"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lestrrat-go/strftime"

	"github.com/whatap/go-api/agent/agent/config"
	langconf "github.com/whatap/go-api/agent/lang/conf"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/golib/util/hmap"
	"github.com/whatap/golib/util/stringutil"
)

type DateFormatFile struct {
	fileName     string
	prevFileName string
	curFileName  string
}

func NewDateFormatFile(fileName, curFileName string) *DateFormatFile {
	p := new(DateFormatFile)
	p.fileName = fileName
	p.prevFileName = curFileName
	p.curFileName = curFileName
	return p
}

type WatchLogManager struct {
	watchEnabled bool
	conf         *config.Config

	table *hmap.StringKeyLinkedMap

	dateFormatFiles *hmap.StringKeyLinkedMap
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
	watchLogManager.dateFormatFiles = hmap.NewStringKeyLinkedMap()

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
		// shutdown
		if config.GetConfig().Shutdown {
			logutil.Infoln("WA211-14", "Shutdown WatchLogManager")
			break
		}

		if this.watchEnabled == false {
			time.Sleep(5 * time.Second)
			continue
		}
		time.Sleep(time.Duration(int(this.conf.WatchLogCheckInterval)) * time.Millisecond)
		this.process()
	}
}

func (this *WatchLogManager) process() {
	// check dateformat files
	this.processDateFormatFiles()
	now := time.Now().UnixMilli()
	en := this.table.Values()
	if this.conf.DebugLogSinkEnabled {
		logutil.Infoln("WA-LOGS-201", " process ids size ", this.table.Size())
	}

	for en.HasMoreElements() {
		var dog *WatchLog
		if el := en.NextElement(); el != nil {
			dog = el.(*WatchLog)
			if this.conf.DebugLogSinkEnabled {
				logutil.Infoln("WA-LOGS-202", "process execute dog ", dog.Id)
			}
			func() {
				defer func() {
					if r := recover(); r != nil {
						dog.Stop()
					}
				}()
				if dog.IsActive() {
					dog.Process()
				}
				// expire dog
				if dog.ExpirationTime != 0 && dog.ExpirationTime < now {
					dog.Stop()
					this.table.Remove(dog.Id)
					if this.conf.DebugLogSinkEnabled {
						logutil.Infoln("WA-LOGS-203", "expire dog id=", dog.Id, ", ", dog.ExpirationTime)
					}
				}
			}()
		}
	}
}

func (this *WatchLogManager) processDateFormatFiles() {
	conf := config.GetConfig()
	// check dateformat files
	en := this.dateFormatFiles.Values()
	for en.HasMoreElements() {
		var dff *DateFormatFile
		if el := en.NextElement(); el != nil {
			dff = el.(*DateFormatFile)
			if str, err := strftime.Format(dff.fileName, time.Now()); err == nil {
				// add new file
				if str != dff.curFileName {
					dff.prevFileName = dff.curFileName
					dff.curFileName = str

					// new and activate
					dog := this.Add(dff.curFileName, dff.curFileName, filepath.Base(dff.fileName), []string{}, conf.LogSinkInterval)
					dog.ActivateFirst()

					if this.conf.DebugLogSinkEnabled {
						logutil.Infoln("WA-LOGS-204", "add new dateformatfile to WatchLog ", dff.fileName, ", id=", dff.curFileName)
					}
					// set stop sign after interval .
					if tmp := this.table.Get(dff.prevFileName); tmp != nil {
						if dog, ok := tmp.(*WatchLog); ok {
							dog.ExpirationTime = time.Now().UnixMilli() + conf.LogSinkStopInterval
							if this.conf.DebugLogSinkEnabled {
								logutil.Infoln("WA-LOGS-205", "set expiration time ", dff.fileName, ",id=", dff.prevFileName, ", interval=", conf.LogSinkStopInterval)
							}
						}
					}
				}
			}
		}
	}
}

func (this *WatchLogManager) Add(id string, file string, category string, words []string, checkInterval int32) *WatchLog {
	// java intern, 이미 있는 건 그대로 사용.
	var dog *WatchLog
	if tmp := this.table.Get(id); tmp != nil {
		if dog_tmp, ok := tmp.(*WatchLog); ok {
			dog = dog_tmp
			if this.conf.DebugLogSinkEnabled {
				logutil.Infoln("WA-LOGS-206", "exists ", id)
			}
		}
	}
	if dog == nil {
		dog = NewWatchLog(id)
		this.table.Put(id, dog)
		if this.conf.DebugLogSinkEnabled {
			logutil.Infoln("WA-LOGS-207", "add ", id)
		}
	}

	dog.Config(id, file)
	dog.Words = words
	dog.CheckInterval = int(math.Max(float64(checkInterval), float64(1000)))
	if category != "" {
		dog.Category = category
	}
	return dog
}

func (this *WatchLogManager) resetDogList(reset bool) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA-LOGS-208", "resetDogList Recover", r, ",stack=", string(debug.Stack()))
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
			category := ""
			checkInterval := this.conf.LogSinkInterval

			if this.conf.DebugLogSinkEnabled {
				logutil.Println("WA-LOGS-209", "resetDogList logsink ", "id=", id, ",file=", file, ",enabled=", enabled)
			}
			if file != "" {
				if str, err := strftime.Format(file, time.Now()); err == nil {
					// dateformat file
					if file != str {
						this.dateFormatFiles.Put(file, NewDateFormatFile(file, str))
						if this.conf.DebugLogSinkEnabled {
							logutil.Infoln("WA-LOGS-210", "resetDogList add dateformatFile ", file, ", ", str)
						}
						// dateformat 이름을 카테고리로 지정
						category = filepath.Base(file)
						id = str
						file = str
						// 변경된 파일명을 다시 ids에 넣어줌. 밑에서 삭제 되지 않도록
						sl[i] = id
					}
				}
				dog := this.Add(id, file, category, words, checkInterval)

				if reset {
					dog.Reset()
				}

				if enabled {
					if this.conf.DebugLogSinkEnabled {
						logutil.Println("WA-LOGS-211", "Activate ", "id=", id, ",file=", file, ",enabled=", enabled)
					}
					dog.Activate()
				} else {
					if this.conf.DebugLogSinkEnabled {
						logutil.Println("WA-LOGS-212", "Stop ", "id=", id, ",file=", file, ",enabled=", enabled)
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
				logutil.Println("WA-LOGS-213", "resetDogList ", "id=", id, ",file=", file, ",enabled=", enabled)
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
						logutil.Println("WA-LOGS-214", "Activate ", "id=", id, ",file=", file, ",enabled=", enabled)
					}
					dog.Activate()
				} else {
					if this.conf.DebugLogSinkEnabled {
						logutil.Println("WA-LOGS-215", "Stop ", "id=", id, ",file=", file, ",enabled=", enabled)
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
			if this.conf.DebugLogSinkEnabled {
				logutil.Println("WA-LOGS-216", "clear. ", "remove ", id)
			}
		}
	}
}
