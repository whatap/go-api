package net

import (
	//"log"
	"time"
	//"runtime/debug"

	"github.com/whatap/golib/io"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/secure"
	"github.com/whatap/go-api/agent/util/logutil"
)

var start bool = false
var RecvBuffer chan pack.Pack

func InitReceiver() {
	if start == false {
		lock.Lock()
		if start == false {
			start = true
			RecvBuffer = make(chan pack.Pack, 100)
			go run()
		}
		defer lock.Unlock()
	}
}

func run() {
	conf := config.GetConfig()
	secuMaster := secure.GetSecurityMaster()
	secuMaster.WaitForInit()
	secuSession := secure.GetSecuritySession()
	for {

		func() {
			// for문이 종료 되지 않도록 Recover
			defer func() {
				if r := recover(); r != nil {
					logutil.Println("WA721", " Recover", r) //, string(debug.Stack()))
				}
			}()

			session := GetTcpSession()
			//logutil.Println("isOpen() = ", session.isOpen())
			for session.isOpen() == false {
				session = GetTcpSession()
				time.Sleep(1000 * time.Millisecond)
			}
			out := session.Read()
			if out.Code == NET_TIME_SYNC {
				in := io.NewDataInputX(out.Data)
				prevAgentTime := in.ReadLong()
				serverTime := in.ReadLong()
				now := dateutil.SystemNow()
				turnaroundTime := now - prevAgentTime
				if turnaroundTime < 500 {
					// 전달 시간이 500 이하 인 경우에만 시간을 맞춤
					// 항상 서버 시간보다 약간늦게 가야한다. turnaroundTime 만큼 늦게 시계가 진행될
					// 것이다.
					// SrviceTime 설정 변경 SetServerTime 에서 서버시간과의 차이를 다시구해서 Delta로 설정
					dateutil.SetServerTime(serverTime, 1)
				}
				//continue
				return
			}
			if conf.CypherLevel > 0 {
				if out.TransferKey != secuSession.TRANSFER_KEY {
					//continue
					return
				}
				switch GetSecureMask(out.Code) {
				case NET_SECURE_HIDE:
					if secuSession.Cypher != nil {
						out.Data = secuSession.Cypher.Hide(out.Data)
					}
				case NET_SECURE_CYPHER:
					if secuSession.Cypher != nil {
						out.Data = secuSession.Cypher.Decrypt(out.Data)
					}
				default:
					out.Data = nil
				}
			}
			if out.Data != nil && len(out.Data) > 0 {
				p := pack.ToPack(out.Data)
				RecvBuffer <- p
			}
		}()

		// DEBUG func() defer 처리 후 sleep 으로 변경
		time.Sleep(1 * time.Millisecond)
	}
}
