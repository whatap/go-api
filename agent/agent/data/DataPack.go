package data

import (
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/secure"
	"github.com/whatap/go-api/agent/net"

	//"reflect"
	"github.com/whatap/golib/util/dateutil"
)

func Send(p pack.Pack) {
	SendFlush(p, IsFlushByPack(p.GetPackType()))
}

func SendFlush(p pack.Pack, flush bool) {
	if p == nil {
		return
	}
	conf := config.GetConfig()
	secu := secure.GetSecurityMaster()
	p.SetOID(secu.OID)
	p.SetPCODE(secu.PCODE)
	p.SetOKIND(conf.OKIND)
	p.SetONODE(conf.ONODE)

	//fmt.Println("SendFlush , EncryptLevel=", conf.EncryptLevel, " fluch=", flush)
	switch conf.EncryptLevel {
	case 1:
		net.Send(0, p, flush)
	case 2:
		net.Send(net.NET_SECURE_HIDE, p, flush)
	default:
		net.Send(net.NET_SECURE_CYPHER, p, flush)
	}
}
func Sent(p *pack.TextPack) {
	if p == nil {
		return
	}
	conf := config.GetConfig()
	secu := secure.GetSecurityMaster()
	p.SetOID(secu.OID)
	p.SetPCODE(secu.PCODE)
	p.SetOKIND(conf.OKIND)
	p.SetONODE(conf.ONODE)
	p.Time = dateutil.Now()

	//fmt.Println("Sent , EncryptLevel=", conf.EncryptLevel)
	switch conf.EncryptLevel {
	case 1:
		net.Send(net.NET_SECURE_HIDE, p, true)
	default:
		net.Send(net.NET_SECURE_CYPHER, p, true)
	}
}

func SendProfile(p pack.Pack) {
	if p == nil {
		return
	}
	conf := config.GetConfig()
	secu := secure.GetSecurityMaster()
	p.SetOID(secu.OID)
	p.SetPCODE(secu.PCODE)
	p.SetOKIND(conf.OKIND)
	p.SetONODE(conf.ONODE)

	//fmt.Println("Send , EncryptLevel=", conf.EncryptLevel)
	switch conf.EncryptLevel {
	case 1:
		net.SendProfile(0, p, false)
	case 2:
		net.SendProfile(net.NET_SECURE_HIDE, p, false)
	default:
		net.SendProfile(net.NET_SECURE_CYPHER, p, false)
	}
}
func SendBoot(p pack.Pack) {
	SendSecureFlush(p, true)
}
func SendEvent(p *pack.EventPack) {
	if p == nil {
		return
	}
	conf := config.GetConfig()
	secu := secure.GetSecurityMaster()
	p.SetOID(secu.OID)
	p.SetPCODE(secu.PCODE)
	p.SetOKIND(conf.OKIND)
	p.SetONODE(conf.ONODE)
	p.Time = dateutil.Now()
	p.SetUuid()
	//fmt.Println("SendEvent , EncryptLevel=", conf.EncryptLevel)
	switch conf.EncryptLevel {
	case 1:
		net.Send(net.NET_SECURE_HIDE, p, true)
	default:
		net.Send(net.NET_SECURE_CYPHER, p, true)
	}
}
func SendResponse(p pack.Pack) {
	SendSecureFlush(p, true)
}

func SendSecure(p pack.Pack) {
	SendSecureFlush(p, IsFlushByPack(p.GetPackType()))
}

func SendSecureFlush(p pack.Pack, flush bool) {
	// DEBUG Packet
	//logutil.Println("SendSecure", "SendSecure p=", reflect.TypeOf(p).String()) //, p)
	conf := config.GetConfig()
	secu := secure.GetSecurityMaster()
	p.SetOID(secu.OID)
	p.SetPCODE(secu.PCODE)
	p.SetOKIND(conf.OKIND)
	p.SetONODE(conf.ONODE)
	switch conf.EncryptLevel {
	case 1:
		net.Send(net.NET_SECURE_HIDE, p, flush)
	default:
		net.SendProfile(net.NET_SECURE_CYPHER, p, flush)
	}
}

func SendHide(p pack.Pack) {
	SendHideFlush(p, IsFlushByPack(p.GetPackType()))
}

func SendHideFlush(p pack.Pack, flush bool) {
	// DEBUG Packet
	//logutil.Println("SendHide", "SendHide p=", reflect.TypeOf(p).String()) //, p)
	conf := config.GetConfig()
	secu := secure.GetSecurityMaster()
	p.SetOID(secu.OID)
	p.SetPCODE(secu.PCODE)
	p.SetOKIND(conf.OKIND)
	p.SetONODE(conf.ONODE)

	switch conf.EncryptLevel {
	case 1, 2:
		net.Send(net.NET_SECURE_HIDE, p, flush)
	default:
		net.SendProfile(net.NET_SECURE_CYPHER, p, flush)
	}
}
func SendTagCount(p *pack.TagCountPack, flush bool) {
	conf := config.GetConfig()
	secu := secure.GetSecurityMaster()
	p.SetOID(secu.OID)
	p.SetPCODE(secu.PCODE)
	p.SetOKIND(conf.OKIND)
	p.SetONODE(conf.ONODE)

	p.PutTag("oname", secu.ONAME)
	p.PutTag("okindName", conf.OKIND_NAME)
	p.PutTag("onodeName", conf.ONODE_NAME)

	//fmt.Println("SendFlush , EncryptLevel=", conf.EncryptLevel, " fluch=", flush)
	switch conf.EncryptLevel {
	case 1:
		net.Send(0, p, flush)
	case 2:
		net.Send(net.NET_SECURE_HIDE, p, flush)
	default:
		net.Send(net.NET_SECURE_CYPHER, p, flush)
	}
}

func IsFlushByPack(t int16) bool {
	switch t {
	case pack.PACK_PARAMETER:
		return false
	case pack.PACK_COUNTER_1:
		return true
	case pack.PACK_PROFILE:
		return false
	case pack.PACK_ACTIVESTACK_1:
		return false
	case pack.PACK_TEXT:
		return true
	case pack.PACK_ERROR_SNAP_1:
		return false
	case pack.PACK_REALTIME_USER:
		return false

	case pack.PACK_STAT_SERVICE:
		return false
	case pack.PACK_STAT_GENERAL:
		return true
	case pack.PACK_STAT_SQL:
		return false
	case pack.PACK_STAT_HTTPC:
		return false
	case pack.PACK_STAT_ERROR:
		return false
	//	case PACK_STAT_METHOD:
	//	flush = false
	//	case PACK_STAT_TOP_SERVICE
	//	flush = false
	case pack.PACK_STAT_REMOTE_IP:
		return false
	case pack.PACK_STAT_USER_AGENT:
		return true
	case pack.PACK_EVENT:
		return true
	case pack.PACK_HITMAP_1:
		return false
	case pack.PACK_EXTENSION:
		return false
	case pack.TAG_COUNT:
		return true
		//		case TAG_LOG:
		//		return false
	case pack.PACK_COMPOSITE:
		return false
		//	case PACK_BSM_RECORD:
		//		return false
		//	case PACK_AP_NUT:
		//	return false
	case pack.PACK_ADDIN_COUNT:
		return false
	case pack.PACK_LOGSINK:
		return false

	}
	return false
}
