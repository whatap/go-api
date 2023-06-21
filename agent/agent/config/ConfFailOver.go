package config

import ()

type ConfFailover struct {
	NetFailoverRetrySendDataEnabled    bool
	NetFailoverRetryCounterPackEnabled bool
	DebugTcpFailoverEnabled            bool
}

func (this *ConfFailover) Apply(conf *Config) {

	this.NetFailoverRetrySendDataEnabled = getBoolean("net_failover_retry_send_data_enabled", false)
	this.DebugTcpFailoverEnabled = getBoolean("debug_tcp_failover_enabled", false)

}
