package stat

import (
	//"log"
	//	"runtime/debug"
	_ "fmt"
	"sync"
	"time"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/golib/util/dateutil"

	// import cycle not allowed
	//"github.com/whatap/go-api/agent/agent/stat"
	"github.com/whatap/go-api/agent/util/logutil"
)

type TimingSender struct {
}

var timingSenderLock = sync.Mutex{}
var timingSender *TimingSender

// Singleton
func GetInstanceTimingSender() *TimingSender {
	timingSenderLock.Lock()
	defer timingSenderLock.Unlock()
	if timingSender != nil {
		return timingSender
	}
	timingSender = new(TimingSender)
	//fmt.Println("TimingSender Go RUn")
	go run()
	// TODO
	//	instance.setDaemon(true);
	//			instance.start();
	//			ConfThread.whatap(instance);

	return timingSender
}

func run() {
	conf := config.GetConfig()

	lastUnit := dateutil.GetFiveMinUnit(dateutil.Now())
	//fmt.Println("TimingSender Start=", lastUnit, ",now=", dateutil.Now())

	for {
		// shutdown
		if config.GetConfig().Shutdown {
			logutil.Infoln("WA211-08", "Shutdown TimingSender")
			GetInstanceStatTranx().Clear()
			GetInstanceStatTranxDomain().Clear()
			GetInstanceStatTranxMtCaller().Clear()
			GetInstanceStatTranxLogin().Clear()
			GetInstanceStatTranxReferer().Clear()

			GetInstanceStatSql().Clear()
			GetInstanceStatHttpc().Clear()
			GetInstanceStatError().Clear()
			GetInstanceStatRemoteIp().Clear()
			GetInstanceStatUserAgent().Clear()
			break
		}
		// DEBUG goroutine 로그 출력
		//logutil.Println("TimmingSender Run")

		// 10초
		time.Sleep(10 * time.Second)

		// 5분에 한번씩 실행되도록 함
		nowUnit := dateutil.GetFiveMinUnit(dateutil.Now())
		//DEBUG 1분에 한번
		//nowUnit := dateutil.GetMinUnit(dateutil.Now())

		//fmt.Println("TimingSender after 10 seconds, lastUnit=", lastUnit, ", nowUnit=", nowUnit, ",now=", dateutil.Now())

		if lastUnit == nowUnit {
			continue
		}
		lastUnit = nowUnit

		// for문 내부에서 panic 발생 해도 계속 진행할 수 있게 예외처리
		func() {
			defer func() {
				if r := recover(); r != nil {
					logutil.Println("WA541", "TimingSender run:for: ", r)
					// DEBUG
					//debug.PrintStack()
				}
			}()
			//try {
			now := dateutil.Now()
			//fmt.Println("TimingSender Send Enable=", conf.Stat_enabled, ",now=", now)

			//설정으로 통계정보 수집이 통제 될 수 있다.
			if conf.StatEnabled {
				//fmt.Println("TimingSender Run")
				GetInstanceStatTranx().Send(now)
				GetInstanceStatTranxDomain().Send(now)
				GetInstanceStatTranxMtCaller().Send(now)
				GetInstanceStatTranxLogin().Send(now)
				GetInstanceStatTranxReferer().Send(now)

				GetInstanceStatSql().Send(now)
				GetInstanceStatHttpc().Send(now)
				GetInstanceStatError().Send(now)
				GetInstanceStatRemoteIp().Send(now)
				GetInstanceStatUserAgent().Send(now)

			} else {
				//fmt.Println("TimingSender Clear")
				GetInstanceStatTranx().Clear()
				GetInstanceStatTranxDomain().Clear()
				GetInstanceStatTranxMtCaller().Clear()
				GetInstanceStatTranxLogin().Clear()
				GetInstanceStatTranxReferer().Clear()

				GetInstanceStatSql().Clear()
				GetInstanceStatHttpc().Clear()
				GetInstanceStatError().Clear()
				GetInstanceStatRemoteIp().Clear()
				GetInstanceStatUserAgent().Clear()
			}
		}()

		//일단2분쉬고..
		time.Sleep(2 * time.Minute)
	}
}

//
//
//public class TimingSender extends Thread {
//	private static TimingSender instance;
//
//	public synchronized static TimingSender getInstance() {
//		if (instance == null) {
//			instance = new TimingSender();
//			instance.setDaemon(true);
//			instance.start();
//			ConfThread.whatap(instance);
//		}
//		return instance;
//	}
//
//	@Override
//	public void run() {
//		Configure conf =Configure.getInstance();
//		long lastUnit = DateUtil.getFiveMinUnit(DateUtil.currentTime());
//		while (true) {
//			ThreadUtil.sleep(10000);
//
//			// 5분에 한번씩 실행되도록 함
//			long nowUnit = DateUtil.getFiveMinUnit(DateUtil.currentTime());
//			if (lastUnit == nowUnit)
//				continue;
//			lastUnit = nowUnit;
//			try {
//				long now = DateUtil.now();
//				//설정으로 통계정보 수집이 통제 될 수 있다.
//				if (conf.stat_enabled) {
//					StatTranx.getInstance().send(now);
//					StatSql.getInstance().send(now);
//					StatHttpc.getInstance().send(now);
//					StatError.getInstance().send(now);
//					StatRemoteIp.getInstance().send(now);
//					StatUserAgent.getInstance().send(now);
//				} else {
//					StatTranx.getInstance().clear();
//					StatSql.getInstance().clear();
//					StatHttpc.getInstance().clear();
//					StatError.getInstance().clear();
//					StatRemoteIp.getInstance().clear();
//					StatUserAgent.getInstance().clear();
//				}
//
//				//일단2분쉬고..
//				ThreadUtil.sleep(DateUtil.MILLIS_PER_MINUTE*2);
//
//			} catch (Throwable e) {
//				e.printStackTrace();
//			}
//		}
//	}
//
//}
