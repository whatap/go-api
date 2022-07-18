package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"

	whash "github.com/whatap/golib/util/hash"
)

type ConfNet struct {
	License string

	PCODE      int64
	OID        int64
	ONAME      string
	OKIND      int32
	OKIND_NAME string
	ONODE      int32
	ONODE_NAME string

	WhatapHost []string
	WhatapPort int32
	Hosts      []string
}

func (this *ConfNet) ApplyDefault(m map[string]string) {

	m["go.sql_profile_enabled"] = "true"
	m["go.counter_enabled"] = "true"
	m["go.counter_interval"] = "5000"
	m["go.counter_timeout"] = "5000"

}
func (this *ConfNet) Apply(conf *Config) {
	if license := os.Getenv("WHATAP_LICENSE"); license != "" {
		this.License = license
	} else {
		this.License = conf.GetValue("license")
	}
	if pcode := os.Getenv("WHATAP_PCODE"); pcode != "" {
		if v, err := strconv.ParseInt(pcode, 10, 64); err == nil {
			this.PCODE = v
		} else {
			this.PCODE = conf.GetLong("pcode", 0)
		}
	} else {
		this.PCODE = conf.GetLong("pcode", 0)
	}

	this.OID = conf.GetLong("oid", 0)
	this.OKIND = conf.GetInt("okind", 0)
	this.OKIND_NAME = conf.GetValueDef("okind_name", "")
	this.ONODE = conf.GetInt("onode", 0)
	this.ONODE_NAME = conf.GetValueDef("onode_name", "")

	if host := os.Getenv("WHATAP_HOST"); host != "" {
		this.WhatapHost = conf.GetStringArray(host, "/:,")
	} else {
		this.WhatapHost = conf.GetStringArray("whatap.server.host", "/:,")

	}
	if tmp := os.Getenv("WHATAP_PORT"); tmp != "" {
		if v, err := strconv.Atoi(tmp); err != nil {
			this.WhatapPort = int32(v)
		} else {
			this.WhatapPort = conf.GetInt("whatap.server.port", 6600)
		}
	} else {
		this.WhatapPort = conf.GetInt("whatap.server.port", 6600)
	}

	this.Hosts = make([]string, 0)
	for it := range this.WhatapHost {
		addr := fmt.Sprintf("tcp://%s:%d", it, this.WhatapPort)

		u, err := url.Parse(addr)
		if err != nil {
			//this.Log.Errorf("invalid address: %s", server)
			continue
		}

		if u.Scheme != "tcp" {
			//this.Log.Errorf("only tcp is supported: %s", server)
			continue
		}
		this.Hosts = append(this.Hosts, u.Host)
	}

	// this.TcpSoTimeout = conf.GetInt("tcp_so_timeout", 120000)
	// this.TcpSoSendTimeout = conf.GetInt("tcp_so_send_timeout", 20000)
	// this.TcpConnectionTimeout = conf.GetInt("tcp_connection_timeout", 5000)

	// this.NetSendMaxBytes = conf.GetInt("net_send_max_bytes", 5*1024*1024)
	// this.NetSendBufferSize = conf.GetInt("net_send_buffer_size", 1024)
	// this.NetWriteBufferSize = conf.GetInt("net_write_buffer_size", 8*1024*1024)
	// this.NetSendQueue1Size = conf.GetInt("net_send_queue1_size", 256)
	// this.NetSendQueue2Size = conf.GetInt("net_send_queue2_size", 512)

	// this.NetUdpHost = conf.GetValue("net_udp_host")
	// this.NetUdpPort = conf.GetInt("net_udp_port", 6600)
	// this.NetUdpReadBytes = conf.GetInt("net_udp_read_bytes", 2*1024*1024)

	hn, err := os.Hostname()
	if err != nil {
		//return fmt.Errorf("failed to get hostname: %v", err)
	}
	this.ONAME = hn
	this.OID = int64(whash.HashStr(this.ONAME))
}
