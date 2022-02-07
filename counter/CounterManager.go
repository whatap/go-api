package counter

import (
	"log"
	"sync"
	"time"

	"github.com/whatap/go-api/common/util/dateutil"
	"github.com/whatap/go-api/config"
	"github.com/whatap/go-api/counter/task"
)

var counter *Counter
var counterLock sync.Mutex

type Counter struct {
	tasks   map[string]Task
	endChan chan bool
}

func init() {
	GetCounterManager()
}

func GetCounterManager() *Counter {
	counterLock.Lock()
	defer counterLock.Unlock()
	if counter == nil {
		counter = &Counter{}
		counter.tasks = make(map[string]Task)
		counter.endChan = make(chan bool, 1)
		// The Add function should not be used because counterLock is also used in the Add function.
		counter.tasks["goRuntime"] = &task.TaskGoRuntime{}
		go counter.process()
	}
	return counter
}
func (this *Counter) Add(name string, t Task) {
	counterLock.Lock()
	defer counterLock.Unlock()
	this.tasks[name] = t
}
func (this *Counter) process() {
	conf := config.GetConfig()
	var INTERVAL int32 = conf.GoCounterInterval
	if INTERVAL < 5000 {
		INTERVAL = 5000
	}

	lastSysTime := dateutil.SystemNow()
	next := (dateutil.Now() / int64(INTERVAL) * int64(INTERVAL)) + int64(INTERVAL)
	for {
		this.sleepx(next, int64(INTERVAL))
		now := dateutil.Now() / int64(INTERVAL) * int64(INTERVAL)
		next = now + int64(INTERVAL)

		duration := int32((int(dateutil.SystemNow()-lastSysTime) + 499) / 1000)
		lastSysTime = dateutil.SystemNow()

		if duration <= 0 {
			continue
		}
		if conf.GoCounterEnabled {
			go this.executeTasks(now)
			select {
			case <-this.endChan:
			case <-time.After(time.Duration(conf.GoCounterTimeout) * time.Millisecond):
				if conf.Debug {
					log.Println("WA300", "Counter timeout conf=", int(conf.GoCounterTimeout), ", t=", now)
				}
			}
		}
	}
}

func (this *Counter) executeTasks(now int64) {
	for _, t := range this.tasks {
		if initTask, ok := t.(TaskInitialize); ok {
			initTask.Init()
		}
		t.Process(now)
	}
	this.endChan <- true
}

func (this *Counter) sleepx(next, interval int64) {
	stime := dateutil.Now() / interval * interval
	sleepTime := (next - dateutil.Now()) / 1000 * 1000
	if sleepTime < 0 {
		sleepTime = 0
	} else if sleepTime > 3000 {
		sleepTime = 3000
	}
	if sleepTime > 0 {
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	}
	for {
		now := ((dateutil.Now() / interval) * interval)
		if stime != ((now / interval) * interval) {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}

type Task interface {
	Process(now int64)
}
type TaskInitialize interface {
	Init() error
}
