package config

import (
	"github.com/whatap/go-api/agent/util/logutil"
)

type ConfDebugTest struct {
	DebugCloseTcp     int32
	DebugCloseTcpFunc func()
}

func (this *ConfDebugTest) Apply(conf *Config) {
	closeTcp := GetInt("debug_close_tcp", 0)
	if this.DebugCloseTcp != closeTcp {
		logutil.Infoln(">>>>", "Close tcp ", closeTcp)
		this.DebugCloseTcp = closeTcp
		this.DebugCloseTcpFunc()
	}
}
