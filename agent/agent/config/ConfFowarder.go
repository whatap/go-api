package config

import (
	"github.com/whatap/golib/lang"
)

type ConfFowarder struct {
	FowarderEnabled      bool
	DebugFowarderEnalbed bool

	NetIPCHost string
	NetIPCPort int32
}

func (this *ConfFowarder) Apply(conf *Config) {
	this.FowarderEnabled = GetBoolean("fowarder_enabled", false)
	if conf.AppType == lang.APP_TYPE_GO {
		this.FowarderEnabled = GetBoolean("fowarder_enabled", true)
	}

	// Fowarder host, port
	this.NetIPCHost = GetValueDef("net_ipc_host", GetValueDef("net_udp_host", "127.0.0.1"))
	this.NetIPCPort = GetInt("net_ipc_port", int(GetInt("net_udp_port", 6600)))

	this.DebugFowarderEnalbed = GetBoolean("debug_fowarder_enabled", false)
}
