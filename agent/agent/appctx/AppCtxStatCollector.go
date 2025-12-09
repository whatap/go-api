package appctx

import (
	"strings"
	"sync"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/agent/secure"

	// "github.com/whatap/go-api/agent/agent/trace" // 이 import 제거!
	langconf "github.com/whatap/go-api/agent/lang/conf"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/service"
	"github.com/whatap/golib/lang/value"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/hmap"
	"github.com/whatap/golib/util/iputil"
	"github.com/whatap/golib/util/stringutil"
)

// Perf 성능 통계 구조체
type Perf struct {
	Act0      int32 // 0-3초 응답시간 트랜잭션
	Act3      int32 // 3-8초 응답시간 트랜잭션
	Act8      int32 // 8초 이상 응답시간 트랜잭션
	Cnt       int32 // 총 트랜잭션 수
	Err       int32 // 에러 수
	TimeSum   int64 // 총 응답시간
	Tolerated int32 // Apdex tolerated 수
	Satisfied int32 // Apdex satisfied 수
}

func (p *Perf) Actx() int32 {
	return p.Act0 + p.Act3 + p.Act8
}

// AppCtxStatCollector 통계 수집기
type AppCtxStatCollector struct {
	appCtxTable *hmap.StringKeyLinkedMap
	actTable    *hmap.StringKeyLinkedMap
	lastTime    int64

	appCtxParser      IAppCtx
	appCtxParserName  string
	appCtxParserReset int32

	mutex  sync.RWMutex
	loader *AppCtxParserLoader
}

var instance *AppCtxStatCollector
var instanceLock = sync.Mutex{}

func GetIntanceAppCtxStatCollector() *AppCtxStatCollector {
	instanceLock.Lock()
	defer instanceLock.Unlock()

	if instance != nil {
		return instance
	}

	instance = &AppCtxStatCollector{
		appCtxTable:      hmap.NewStringKeyLinkedMap(),
		actTable:         hmap.NewStringKeyLinkedMap(),
		lastTime:         dateutil.Now(),
		appCtxParser:     &PathDefault{},
		appCtxParserName: "default",
		loader:           GetAppCtxParserLoader(),
	}

	langconf.AddConfObserver("AppCtxStatCollector", instance)
	instance.update()

	return instance
}

// Run
func (c *AppCtxStatCollector) Run() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.updateNoLock()
}

func (c *AppCtxStatCollector) intern(table *hmap.StringKeyLinkedMap, key string) *Perf {
	if table.ContainsKey(key) {
		return table.Get(key).(*Perf)
	}

	perf := &Perf{}
	table.Put(key, perf)
	return perf
}

func (c *AppCtxStatCollector) Process(workTime int64) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA221-01", "AppCtxStatCollector.process Error", r)
		}
	}()

	now := dateutil.Now()
	interval := float64(now-c.lastTime) / 1000.0
	c.lastTime = now

	if interval <= 1 {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.appCtxTable.Size()+c.actTable.Size() == 0 {
		return
	}

	// 경로 리스트 수집
	paths := make([]string, 0)
	conf := config.GetConfig()

	if stringutil.TrimEmpty(conf.AppContextPathSet) != "" {
		configPaths := stringutil.Tokenizer(conf.AppContextPathSet, ",")
		for _, item := range configPaths {
			item = stringutil.TrimEmpty(item)
			if item == "" {
				continue
			}

			var contextName string
			if strings.Contains(item, "@") {
				parts := stringutil.Tokenizer(item, "@")
				if len(parts) >= 2 {
					contextName = stringutil.TrimEmpty(parts[0]) // ← name 부분이 실제 키!
				}
			} else {
				contextName = stringutil.TrimEmpty(item)
			}

			if contextName == "" {
				continue
			}

			// name 부분으로 직접 테이블 확인
			if c.appCtxTable.ContainsKey(contextName) || c.actTable.ContainsKey(contextName) {
				paths = append(paths, contextName)
			}
		}
	} else {
		en := c.appCtxTable.Keys()
		for en.HasMoreElements() {
			paths = append(paths, en.NextString())
		}
		en = c.actTable.Keys()
		for en.HasMoreElements() {
			k := en.NextString()
			if !c.appCtxTable.ContainsKey(k) {
				paths = append(paths, k)
			}
		}
	}

	if len(paths) == 0 {
		return
	}

	// TagCountPack 생성
	secu := secure.GetSecurityMaster()
	p := pack.NewTagCountPack()

	// 기본 태그 설정
	p.Pcode = secu.PCODE
	p.Oid = secu.OID
	p.Time = workTime
	p.Category = "app_context_stat"

	p.Tags.PutString("oname", secu.ONAME)
	p.Tags.PutString("ip", iputil.ToStringFrInt(secu.IP))
	p.Tags.PutString("public_ip", iputil.ToStringFrInt(secu.PUBLIC_IP))
	p.Tags.PutLong("oid", int64(secu.OID))

	parserName := c.appCtxParser.Name()
	if parserName == "" {
		parserName = "unknown"
	}
	p.Tags.PutString("parser", parserName)
	p.Tags.PutLong("!rectype", 2)

	// 필드별 리스트 생성
	idList := value.NewListValue(nil)
	pathListValue := value.NewListValue(nil)
	activeTxCountList := value.NewListValue(nil)
	activeTx0List := value.NewListValue(nil)
	activeTx3List := value.NewListValue(nil)
	activeTx8List := value.NewListValue(nil)
	tpsList := value.NewListValue(nil)
	txCountList := value.NewListValue(nil)
	txErrorList := value.NewListValue(nil)
	txTimeSumList := value.NewListValue(nil)
	txToleratedList := value.NewListValue(nil)
	txSatisfiedList := value.NewListValue(nil)

	// 각 경로별 데이터를 리스트에 추가
	for _, path := range paths {
		// ID 해시 생성
		pathHash := hash.HashStr(path)

		idList.Add(value.NewDecimalValue(int64(pathHash)))
		pathListValue.Add(value.NewTextValue(path))

		// 활성 트랜잭션 통계
		var actPerf *Perf
		if actPerfIf := c.actTable.Get(path); actPerfIf != nil {
			actPerf = actPerfIf.(*Perf)
		} else {
			actPerf = &Perf{}
		}

		activeTxCountList.Add(value.NewDecimalValue(int64(actPerf.Actx())))
		activeTx0List.Add(value.NewDecimalValue(int64(actPerf.Act0)))
		activeTx3List.Add(value.NewDecimalValue(int64(actPerf.Act3)))
		activeTx8List.Add(value.NewDecimalValue(int64(actPerf.Act8)))

		var ctxPerf *Perf
		if ctxPerfIf := c.appCtxTable.Get(path); ctxPerfIf != nil {
			ctxPerf = ctxPerfIf.(*Perf)
		} else {
			ctxPerf = &Perf{}
		}

		tps := float64(ctxPerf.Cnt) / interval
		tpsList.Add(value.NewFloatValue(float32(tps)))
		txCountList.Add(value.NewDecimalValue(int64(ctxPerf.Cnt)))
		txErrorList.Add(value.NewDecimalValue(int64(ctxPerf.Err)))
		txTimeSumList.Add(value.NewDecimalValue(ctxPerf.TimeSum))
		txToleratedList.Add(value.NewDecimalValue(int64(ctxPerf.Tolerated)))
		txSatisfiedList.Add(value.NewDecimalValue(int64(ctxPerf.Satisfied)))
	}

	p.Put("@id", idList)
	p.Put("path", pathListValue)
	p.Put("active_tx_count", activeTxCountList)
	p.Put("active_tx_0", activeTx0List)
	p.Put("active_tx_3", activeTx3List)
	p.Put("active_tx_8", activeTx8List)
	p.Put("tps", tpsList)
	p.Put("tx_count", txCountList)
	p.Put("tx_error", txErrorList)
	p.Put("tx_time_sum", txTimeSumList)
	p.Put("tx_tolerated", txToleratedList)
	p.Put("tx_satisfied", txSatisfiedList)

	// 테이블 초기화
	c.actTable.Clear()
	if c.appCtxTable.Size() > 0 {
		c.appCtxTable.Clear()
	}

	// 데이터 전송
	c.sendData(p)
}

func (c *AppCtxStatCollector) sendData(p *pack.TagCountPack) {
	data.SendHide(p)

	conf := config.GetConfig()
	if conf.CounterLogEnabled {
		logutil.Println("Sending AppCtx TagCountPack:", p.Category, "time:", dateutil.DateTime(p.Time))
	}
}

func (c *AppCtxStatCollector) ActiveTx(biz string, divAct int32) {
	if stringutil.TrimEmpty(biz) == "" {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	perf := c.intern(c.actTable, biz)
	if perf == nil {
		return
	}

	switch divAct {
	case 1:
		perf.Act3++
	case 2:
		perf.Act8++
	default:
		perf.Act0++
	}
}

func (c *AppCtxStatCollector) EndTx(contextName string, tx *service.TxRecord) {
	if stringutil.TrimEmpty(contextName) == "" {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	perf := c.intern(c.appCtxTable, contextName)
	if perf == nil {
		return
	}

	perf.Cnt++
	perf.TimeSum += int64(tx.Elapsed)

	if tx.Error != 0 {
		perf.Err++
	} else {
		// Apdex 분류
		conf := config.GetConfig()
		apdexTime := conf.ApdexTime
		if apdexTime <= 0 {
			apdexTime = 1200 // 기본값 1.2초
		}

		if tx.Elapsed <= apdexTime {
			perf.Satisfied++
		} else if tx.Elapsed <= apdexTime*4 {
			perf.Tolerated++
		}
		// 4T 이상은 Frustrated (별도 카운팅 없음)
	}
}

func (c *AppCtxStatCollector) GetContextPath(hashValue uint32, url string) string {
	c.mutex.RLock()
	parser := c.appCtxParser
	c.mutex.RUnlock()

	result := parser.Parse(hashValue, url)

	return result
}

func (c *AppCtxStatCollector) update() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.updateNoLock()
}

func (c *AppCtxStatCollector) updateNoLock() {
	conf := config.GetConfig()
	if !conf.AppContextEnabled {
		return
	}

	newParser := stringutil.TrimEmpty(conf.AppContextParser)
	reset := conf.AppContextParserReset

	if c.appCtxParserName != newParser || c.appCtxParserReset != reset {
		c.appCtxParserReset = reset
		c.appCtxParserName = newParser

		switch newParser {
		case "default":
			c.appCtxParser = &PathDefault{}
		case "prefix":
			c.appCtxParser = NewPathPrefix()
		case "match":
			c.appCtxParser = NewPathMatch()
		default:
			c.appCtxParser = &PathDefault{}
		}
	}

	c.appCtxParser.Update()
}

func (c *AppCtxStatCollector) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.appCtxTable.Clear()
	c.actTable.Clear()
}

func GetContextPath(hashValue uint32, url string) string {
	collector := GetIntanceAppCtxStatCollector()
	return collector.GetContextPath(hashValue, url)
}
