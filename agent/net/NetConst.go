package net

import ()

const (
	NET_SECURE_HIDE       = 0x01
	NET_SECURE_CYPHER     = 0x02
	NET_ONE_WAY_NO_CYPHER = 0x04
	NET_RESERVED2         = 0x08
	NET_RESERVED3         = 0x10
	NET_RESERVED4         = 0x20
	NET_RESERVED5         = 0x40
	NET_RESERVED6         = 0x80

	NET_KEY_EXTENSION = 0xfd
	NET_TIME_SYNC     = 0xfe
	NET_KEY_RESET     = 0xff
)

const (
	NETSRC_AGENT_JAVA_EMBED   = 1
	NETSRC_AGENT_JAVA_WATCHER = 2
	NETSRC_SERVER_YARD        = 3
	NETSRC_SERVER_PROXY       = 4

	NETSRC_ZABBIX_PROXY = 37
	NETSRC_KUBE         = 5
	NETSRC_ONEWAY       = 10
)

const (
	TCP_NONE = 0x00
	TCP_OK   = 0x01
)

func GetSecureMask(code byte) byte {
	if code < 0 {
		return 0
	}
	return (byte)((code & NET_SECURE_HIDE) | (code & NET_SECURE_CYPHER))
}
