package active

import (
	"math"
	"sync"
	"time"

	agentconfig "github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/agent/secure"
	"github.com/whatap/go-api/agent/agent/trace"
	wnet "github.com/whatap/go-api/agent/net"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/step"
	"github.com/whatap/golib/util/compressutil"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/keygen"
	"github.com/whatap/golib/util/stringutil"
)

type ActiveStackDumpGo struct {
	conf *agentconfig.Config
}

var (
	activeStackDumpGo   *ActiveStackDumpGo
	activeStackDumpOnce sync.Once
)

// GetActiveStackDumpGo returns singleton instance and starts the goroutine
func GetActiveStackDumpGo() *ActiveStackDumpGo {
	activeStackDumpOnce.Do(func() {
		activeStackDumpGo = &ActiveStackDumpGo{
			conf: agentconfig.GetConfig(),
		}
		go activeStackDumpGo.run()
	})
	return activeStackDumpGo
}

func (a *ActiveStackDumpGo) run() {
	for {
		// Sleep first, then process (like Java version)
		interval := a.conf.ActiveStackSecond
		if interval < 2 {
			interval = 2 // Math.max(2, interval)
		}
		time.Sleep(time.Duration(interval) * time.Second)

		// Process with recover (like Java try-catch)
		func() {
			defer func() {
				recover() // ignore panic
			}()
			if a.conf.ActiveStackEnabled {
				a.process()
			}
		}()
	}
}

func (a *ActiveStackDumpGo) process() {
	maxCount := int(a.conf.ActiveStackCount)
	if maxCount <= 0 {
		maxCount = 100
	}

	// Collect all goroutine stacks into map
	stackMap := make(map[uint64]string, 256)
	CollectGoroutineStacksCallback(func(id uint64, state, stack string) {
		stackMap[id] = stack
	})

	if len(stackMap) == 0 {
		return
	}

	// Prepare pack list for zip (like Java ArrayList)
	var packs []interface{}
	if a.conf.ActiveStackZipEnabled {
		packs = make([]interface{}, 0, 40)
	}

	// Match TraceContexts with goroutine stacks and send
	sent := 0
	en := trace.GetContextEnumeration()
	for sent < maxCount && en.HasMoreElements() {
		ctx, ok := en.NextElement().(*trace.TraceContext)
		if !ok || ctx == nil {
			continue
		}

		if stack := stackMap[uint64(ctx.ThreadId)]; stack != "" {
			actStack := CreateActiveStackPack(ctx, stack)
			if actStack == nil {
				continue
			}
			sent++

			if packs != nil {
				packs = append(packs, actStack)
				if len(packs) >= 30 {
					SendActiveStackZip(packs)
					packs = make([]interface{}, 0, 40)
				}
			} else {
				// flush every 100
				SendActiveStackPack(actStack, sent%100 == 0)
			}
		}
	}

	// Send remaining packs
	if packs != nil {
		switch len(packs) {
		case 0:
			// do nothing
		case 1:
			SendActiveStackPack(packs[0], false)
		default:
			SendActiveStackZip(packs)
		}
	}
}

// CreateActiveStackPack creates an ActiveStackPack from TraceContext and stack string
func CreateActiveStackPack(ctx *trace.TraceContext, s string) *pack.ActiveStackPack {
	if ctx == nil {
		return nil
	}

	conf := agentconfig.GetConfig()
	currentTime := dateutil.Now()

	actStack := pack.NewActiveStackPack()
	actStack.Time = currentTime
	actStack.Seq = keygen.Next()
	actStack.ProfileSeq = ctx.ProfileSeq
	actStack.Service = ctx.ServiceHash
	actStack.Elapsed = int32(currentTime - ctx.StartTime)

	// 액티브 스택이 덤프된 상태에서만 프로파일 스텝에 추가한다.
	ctx.ProfileActive++
	st := step.NewActiveStackStep()
	st.Seq = actStack.Seq
	st.HasCallstack = true
	st.StartTime = actStack.Elapsed
	ctx.Profile.AddTail(st)

	// stack
	se := stringutil.Tokenizer(s, "\n")
	max := math.Min(float64(len(se)), float64(conf.TraceActiveCallstackDepth))
	actStack.CallStack = make([]int32, int32(max))

	// Java와 동일하게 순방향 (se[i] 직접 사용)
	for i := 0; i < int(max); i++ {
		actStack.CallStack[i] = hash.HashStr(se[i])
		actStack.CallStackHash ^= actStack.CallStack[i]
		data.SendHashText(pack.TEXT_STACK_ELEMENTS, actStack.CallStack[i], se[i])
	}

	return actStack
}

// SendActiveStackPack sends a single ActiveStackPack
func SendActiveStackPack(p interface{}, flush bool) {
	if actStack, ok := p.(*pack.ActiveStackPack); ok {
		data.Send(actStack)
	}
}

// SendActiveStackZip sends multiple ActiveStackPacks as a ZipPack (like Java sendActiveStackZip)
func SendActiveStackZip(packs []interface{}) {
	if len(packs) == 0 {
		return
	}

	conf := agentconfig.GetConfig()
	secuMaster := secure.GetSecurityMaster()

	// Convert to []pack.Pack
	items := make([]pack.Pack, 0, len(packs))
	for _, p := range packs {
		if actStack, ok := p.(*pack.ActiveStackPack); ok {
			items = append(items, actStack)
		}
	}

	if len(items) == 0 {
		return
	}

	// Create ZipPack (like Java)
	zp := pack.NewZipPack()
	zp.Pcode = secuMaster.PCODE
	zp.Oid = secuMaster.OID
	zp.Okind = conf.OKIND
	zp.Onode = conf.ONODE
	zp.Time = dateutil.SystemNow()
	zp.SetRecords(items)

	// Compress (status=1 means gzip)
	zp.Status = 1
	var err error
	zp.Records, err = compressutil.DoZip(zp.Records)
	if err != nil {
		return
	}

	// Send via TcpRequestMgr.addProfile equivalent
	wnet.SendProfile(0, zp, false)
}
