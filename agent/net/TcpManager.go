package net

import (
	"github.com/whatap/go-api/agent/agent/config"
	langconf "github.com/whatap/go-api/agent/lang/conf"
)

type TcpManager struct {
}

var tcpManager *TcpManager = nil

func GetInstanceTcpManager() *TcpManager {
	if tcpManager == nil {
		tcpManager = newTcpManager()
		// observer 등록
		langconf.AddConfObserver("TcpManager", tcpManager)
	}
	return tcpManager
}

func newTcpManager() *TcpManager {
	p := new(TcpManager)
	return p
}

// config 로드후 실행, implemenets ConfObjserver.Runnable
func (this *TcpManager) Run() {
	conf := config.GetConfig()
	if conf.QueueTcpEnabled {
		if TcpQueue != nil {
			TcpQueue.SetCapacity(int(conf.NetSendQueue1Size), int(conf.NetSendQueue2Size))
		}
	}
}

func StartNet() {
	InitSender()
	InitReceiver()
	tcp := GetTcpSession()
	tcp.WaitForConnection()
	GetInstanceTcpManager()
}
