package net

import (
	"fmt"
	"net"
	"sync"

	//"syscall"
	"bufio"
	"runtime/debug"
	"strings"
	"time"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/counter/meter"
	"github.com/whatap/go-api/agent/agent/secure"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/golib/io"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/dateutil"

	"github.com/whatap/golib/util/iputil"
	"github.com/whatap/golib/util/queue"
	"github.com/whatap/golib/util/stringutil"
)

const (
	READ_MAX = 8 * 1024 * 1024
)

type TcpSession struct {
	client            net.Conn
	wr                *bufio.Writer
	in                *io.DataInputX
	dest              int
	LastConnectedTime int64

	RetryQueue *queue.RequestQueue
}

type TcpReturn struct {
	Code        byte
	Data        []byte
	TransferKey int32
}

var sessionLock = sync.Mutex{}
var sendLock = sync.Mutex{}
var session *TcpSession

func GetTcpSession() *TcpSession {
	sessionLock.Lock()
	defer sessionLock.Unlock()
	if session != nil {
		return session
	}
	session = new(TcpSession)
	session.RetryQueue = queue.NewRequestQueue(256)
	go func() {
		for {
			// shutdown
			if config.GetConfig().Shutdown {
				logutil.Infoln("WA211-17", "Shutdown net.TcpSession.open")
				if session.wr != nil {
					session.wr.Reset(nil)
					session.wr = nil
				}
				return
			}

			for session.open() == false {
				// shutdown
				if config.GetConfig().Shutdown {
					logutil.Infoln("WA211-17-01", "Shutdown net.TcpSession.open")
					if session.wr != nil {
						session.wr.Reset(nil)
						session.wr = nil
					}
					return
				}
				time.Sleep(3000 * time.Millisecond)
			}

			//logutil.Println("isOpen")
			for session.isOpen() {
				// shutdown
				if config.GetConfig().Shutdown {
					logutil.Infoln("WA211-17-02", "Shutdown net.TcpSession.open")
					if session.wr != nil {
						session.wr.Reset(nil)
						session.wr = nil
					}
					return
				}
				time.Sleep(5000 * time.Millisecond)
			}
		}

	}()

	return session
}

func (this *TcpSession) open() (ret bool) {
	sessionLock.Lock()
	defer func() {
		sessionLock.Unlock()
		if x := recover(); x != nil {
			logutil.Println("WA172", x, string(debug.Stack()))
			ret = false
			this.Close()
		}
	}()

	if this.isOpen() {
		return true
	}

	// if secure.GetSecurityMaster().PCODE == 0 {
	// 	this.Close()
	// 	return false
	// }

	conf := config.GetConfig()
	// DEBUG TEST
	conf.ConfDebugTest.DebugCloseTcpFunc = this.Close

	if strings.TrimSpace(conf.AccessKey) == "" {
		logutil.Println("WA173-00", "accesskey is not set. value is ", conf.AccessKey)
		return false
	}
	hosts := conf.WhatapHost
	if hosts == nil || len(hosts) == 0 {
		logutil.Println("WA173-01", "whatap.server.host is not set. value is ", conf.WhatapHost)
		this.Close()
		return false
	}
	this.dest += 1
	if this.dest >= len(hosts) {
		this.dest = 0
	}
	port := conf.WhatapPort

	var client net.Conn
	var err error
	if conf.FowarderEnabled {
		// connect to proxy (temp, test)
		logutil.Println("WA173-02", "Connect to fowarder ", fmt.Sprintf("%s:%d", conf.NetIPCHost, conf.NetIPCPort))
		client, err = net.DialTimeout("tcp", fmt.Sprintf("%s:%d", conf.NetIPCHost, conf.NetIPCPort), time.Duration(conf.TcpConnectionTimeout)*time.Millisecond)
		if err != nil {
			logutil.Println("WA173-03", "Connection error. (invalid whatap.server.host key error.)", err)
			if client != nil {
				client.Close()
			}
			this.Close()
			return false
		}

		// reset key from proxy

		client.SetDeadline(time.Now().Add(time.Duration(conf.TcpSoTimeout) * time.Millisecond))
		client.Write(this.keyResetToFowarder(fmt.Sprintf("%s:%d", hosts[this.dest], port)))
		this.in = io.NewDataInputNet(client)
		pcode, myIP, data, err := this.readKeyResetFromFowarder(this.in)
		if err != nil {
			logutil.Println("WA173-04", "Connection error. (invalid whatap.server.host key error.)", err)
			this.Close()
			return false
		}
		// logutil.Infoln(">>>>", "pcode=", pcode, ", myIP=", myIP)
		// secure.GetSecurityMaster().UpdateLicense(this.readKeyResetFromFowarder(this.in))
		secure.GetSecurityMaster().UpdateLicense(pcode, data)

		if secure.GetSecurityMaster().PCODE == 0 {
			this.Close()
			return false
		}
		secure.GetSecurityMaster().DecideAgentOnameOid(stringutil.Tokenizer(myIP, ":")[0])

	} else {
		if secure.GetSecurityMaster().PCODE == 0 {
			this.Close()
			return false
		}

		// connect to whatap
		client, err = net.DialTimeout("tcp", fmt.Sprintf("%s:%d", hosts[this.dest], port), time.Duration(conf.TcpConnectionTimeout)*time.Millisecond)
		if err != nil {
			logutil.Println("WA173-05", "Connection error. (invalid whatap.server.host key error.)", err)
			if client != nil {
				client.Close()
			}
			this.Close()
			return false
		}

		secure.GetSecurityMaster().DecideAgentOnameOid(stringutil.Tokenizer(client.LocalAddr().String(), ":")[0])
	}

	client.SetDeadline(time.Now().Add(time.Duration(conf.TcpSoTimeout) * time.Millisecond))
	client.Write(this.keyReset())
	this.in = io.NewDataInputNet(client)
	data := this.readKeyReset(this.in)
	secure.UpdateNetCypherKey(data)
	logutil.Infoln("WA173-06", "set write buffer ", conf.NetWriteBufferSize, ", bytes")
	this.wr = bufio.NewWriterSize(client, int(conf.NetWriteBufferSize))

	s := secure.GetSecurityMaster()
	logutil.Infoln("WA171", "PCODE=", s.PCODE, " OID=", s.OID, " ONAME=", s.ONAME)
	logutil.Infoln("WA174", "Net TCP: Connected to ", client.RemoteAddr().String())
	this.LastConnectedTime = dateutil.SystemNow()

	if conf.NetFailoverRetrySendDataEnabled {
		// ignore result
		this.SendFailover()
		//		if !this.SendFailover() {
		//			if client != nil {
		//				client.Close()
		//			}
		//			this.Close()
		//			return false
		//		}
	}
	this.RetryQueue.Clear()

	this.client = client

	return true
}
func (this *TcpSession) isOpen() bool {
	//logutil.Printf("Client %p", this.client)
	return this.client != nil
}
func (this *TcpSession) readKeyReset(in *io.DataInputX) []byte {
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Sprintln("invalid license key error. ", r))
		}
	}()
	_ = in.ReadByte()
	_ = in.ReadByte()
	pcode := in.ReadLong()
	oid := in.ReadInt()
	_ = in.ReadInt()
	data := in.ReadIntBytesLimit(1024)
	secu := secure.GetSecurityMaster()

	if pcode != secu.PCODE || oid != secu.OID {
		return []byte{}
	} else {
		return data
	}
}

func (this *TcpSession) readKeyResetFromFowarder(in *io.DataInputX) (pcode int64, ip string, data []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			// panic(fmt.Sprintln("invalid license key error. ", r))
			if pcode == 0 {
				err = fmt.Errorf("invalid license key error. pcode is not validated %d ", pcode)
			}
			if ip == "" {
				err = fmt.Errorf("invalid license key error. local ip is not validated %s ", ip)
			}
			if data == nil || len(data) == 0 {
				err = fmt.Errorf("invalid license key error. session is not validated")
			}
			return
		}
	}()
	// NET_FOWARDER
	_ = in.ReadByte() // netSrc := in.ReadByte()
	// NET_RES_FOWARDER
	netFlag := in.ReadByte() // netFlag := in.ReadByte()
	msg := in.ReadIntBytesLimit(2048)

	din := io.NewDataInputX(msg)
	pcode = din.ReadLong()
	// forward keyreset. add ip info. NET_RES_FOWARDER_1
	if netFlag == NET_RES_FOWARDER_1 {
		ip = din.ReadText()
	} else {
		// set localaddress
		ips := iputil.LocalAddresses()
		if len(ips) > 0 {
			ip = ips[0].String()
		}
	}
	data = din.ReadIntBytesLimit(1024)
	return pcode, ip, data, nil
}
func (this *TcpSession) keyReset() []byte {
	defer func() {
		err := recover()
		if err != nil {
			logutil.Println("WA175", "Recover ", err)
		}
	}()
	conf := config.GetConfig()
	secu := secure.GetSecurityMaster()
	secu.WaitForInit()
	dout := io.NewDataOutputX()

	msg := dout.WriteText("hello").WriteText(secu.ONAME).WriteInt(secu.IP).ToByteArray()
	if conf.CypherLevel > 0 {
		msg = secu.Cypher.Encrypt(msg)
	}
	dout = io.NewDataOutputX()
	dout.WriteByte(NETSRC_AGENT_JAVA_EMBED)

	var trkey int32 = 0
	if conf.CypherLevel == 128 {
		dout.WriteByte(byte(NET_KEY_RESET))
	} else {
		dout.WriteByte(byte(NET_KEY_EXTENSION))

		if conf.CypherLevel == 0 {
			trkey = 0
		} else {
			b0 := byte(1)
			b1 := byte(conf.CypherLevel / 8)
			trkey = io.ToInt([]byte{byte(b0), byte(b1), byte(0), byte(0)}, 0)
		}
	}
	dout.WriteLong(secu.PCODE)
	dout.WriteInt(secu.OID)
	dout.WriteInt(trkey)
	dout.WriteIntBytes(msg)

	//logutil.Infoln(">>>>", "license=", conf.AccessKey, ", pcode=", secu.PCODE, ", oid=", secu.OID)

	return dout.ToByteArray()
}

func (this *TcpSession) keyResetToFowarder(addr string) []byte {
	defer func() {
		err := recover()
		if err != nil {
			logutil.Println("WA175-01", "Recover ", err)
		}
	}()
	conf := config.GetConfig()

	// Fowader secu.Cypher 없이 진행
	dout := io.NewDataOutputX()
	dout.WriteText(addr)
	dout.WriteText(conf.AccessKey)

	msg := dout.ToByteArray()
	dout = io.NewDataOutputX()
	// Netsrc NETSRC_AGENT_JAVA_EMBED
	dout.WriteByte(1)
	// NetFlag NET_REQ_FOWARDER
	dout.WriteByte(NET_REQ_FOWARDER_1)
	dout.WriteIntBytes(msg)

	return dout.ToByteArray()
}

func (this *TcpSession) Send(code byte, b []byte, flush bool) (ret bool) {
	defer func() {
		if x := recover(); x != nil {
			ret = false
			logutil.Println("WA176", " Send Recover ", x)
			this.Close()
		}
	}()

	conf := config.GetConfig()

	secu := secure.GetSecurityMaster()
	secuSession := secure.GetSecuritySession()
	out := io.NewDataOutputX()
	out.WriteByte(NETSRC_AGENT_JAVA_EMBED)
	out.WriteByte(code)
	out.WriteLong(secu.PCODE)
	out.WriteInt(secu.OID)
	out.WriteInt(secuSession.TRANSFER_KEY)
	out.WriteIntBytes(b)
	sendbuf := out.ToByteArray()

	if this.client == nil {
		logutil.Println("WA176-01", " this.client is nil ")
		return false
	}

	// set SetWriteDeadline Write i/o timeout 처리 , Write 전에 반복해서 Deadline 설정
	err := session.client.SetWriteDeadline(time.Now().Add(time.Duration(conf.TcpSoSendTimeout) * time.Millisecond))
	if err != nil {
		logutil.Println("WA177", " SetWriteDeadline failed:", err)
		return false
	}
	if conf.NetWriteLockEnabled {
		if _, err := this.WriteLock(sendbuf); err != nil {
			logutil.Println("WA179-0201", " Write Lock Error", err, ",stack=", string(debug.Stack()))
			this.Close()
			return false
		}
	} else {
		if _, err := this.Write(sendbuf); err != nil {
			logutil.Println("WA179-02", " Write Error", err, ",stack=", string(debug.Stack()))
			this.Close()
			return false
		}
	}
	if flush {
		if n, err := this.Flush(); err != nil {
			logutil.Println("WA179-03", " Flush Error", err, ",stack=", string(debug.Stack()))
			this.Close()
			return false
		} else {
			// DEBUG Meter Self
			if conf.MeterSelfEnabled {
				meter.GetInstanceMeterSelf().AddMeterSelfPacket(int64(n))
			}
			// clear temp when bufio flush
			if conf.DebugTcpSendEnabled || conf.DebugTcpFailoverEnabled {
				logutil.Infoln("WA174-D-03", "flush=", n, ", retry queue reset sz=", this.RetryQueue.Size())
			}
			this.RetryQueue.Clear()
		}
	}
	return true
}
func (this *TcpSession) SendFailover() bool {
	conf := config.GetConfig()
	if conf.DebugTcpFailoverEnabled {
		logutil.Infoln("WA174-D-01", "Open retry queue sz=", this.RetryQueue.Size())
	}
	if this.RetryQueue.Size() > 0 {
		sz := this.RetryQueue.Size()
		for i := 0; i < sz; i++ {
			v := this.RetryQueue.GetNoWait()
			if v == nil {
				continue
			}
			temp := v.(*TcpSend)
			secu := secure.GetSecurityMaster()
			secuSession := secure.GetSecuritySession()
			if n, b, err := this.getEncryptData(temp); err == nil {
				out := io.NewDataOutputX()
				out.WriteByte(NETSRC_AGENT_JAVA_EMBED)
				out.WriteByte(temp.flag)
				out.WriteLong(secu.PCODE)
				out.WriteInt(secu.OID)
				out.WriteInt(secuSession.TRANSFER_KEY)
				out.WriteIntBytes(b)
				sendbuf := out.ToByteArray()
				if conf.NetWriteLockEnabled {
					if _, err := this.WriteLock(sendbuf); err != nil {
						logutil.Println("WA174-0301", "Error Write Lock Retry ", "t=", pack.GetPackTypeString(temp.pack.GetPackType()), ", n=", n, ",", err)
						return false
					}
				} else {
					if _, err := this.Write(sendbuf); err != nil {
						logutil.Println("WA174-03", "Error Write Retry ", "t=", pack.GetPackTypeString(temp.pack.GetPackType()), ", n=", n, ",", err)
						return false
					}
				}
			}
		}
		if n, err := this.Flush(); err != nil {
			logutil.Println("WA174-04", "Error Flush Retry ", ",", err)
			return false
		} else {
			// DEBUG Meter Self
			if conf.MeterSelfEnabled {
				meter.GetInstanceMeterSelf().AddMeterSelfPacket(int64(n))
			}
			// clear temp when bufio flush
			if conf.DebugTcpFailoverEnabled {
				logutil.Infoln("WA174-D-04", "Open retry counter ", "flush=", n)
			}
		}
	}

	return true
}

func (this *TcpSession) Write(sendbuf []byte) (int, error) {
	nbyteleft := len(sendbuf)
	// 다 보내지지 않았을 경우 추가 전송을 위핸 변수 설정.
	pos := 0

	for 0 < nbyteleft {
		nbytethistime, err := this.wr.Write(sendbuf[pos : pos+nbyteleft])
		if err != nil {
			return pos, err
		}

		// DEBUG 로그
		if nbyteleft > nbytethistime {
			logutil.Printf("WA179", "available=%d, send=%d, remine=%d", nbyteleft, nbytethistime, (nbyteleft - nbytethistime))
		}

		nbyteleft -= nbytethistime
		pos += nbytethistime
	}
	return pos, nil
}

func (this *TcpSession) WriteLock(sendbuf []byte) (int, error) {
	sendLock.Lock()
	defer sendLock.Unlock()
	return this.Write(sendbuf)
}

func (this *TcpSession) Flush() (n int, err error) {
	n = this.wr.Buffered()
	if err = this.wr.Flush(); err != nil {
		return 0, err
	}
	return n, nil
}
func (this *TcpSession) Close() {
	if this.client != nil {
		logutil.Infoln("WA181", " Close TCP connection")
		defer func() {
			if r := recover(); r != nil {
				logutil.Println("WA181-01", " Close Recover", string(debug.Stack()))
			}
			this.client = nil
		}()

		this.client.Close()
	}
	this.client = nil
	this.LastConnectedTime = 0
}

func (this *TcpSession) WaitForConnection() {
	//logutil.Println("isOpen")
	for this.isOpen() == false {
		if config.GetConfig().Shutdown {
			logutil.Infoln("WA211-17-03", "Shutdown net.TcpSession WaitForConnection")
			if this.wr != nil {
				this.wr.Reset(nil)
				this.wr = nil
			}
			return
		}
		time.Sleep(1000 * time.Millisecond)
	}
}

var empty *TcpReturn = new(TcpReturn)

func (this *TcpSession) Read() (ret *TcpReturn) {
	// DataInputX 에서 Panic 발생 (EOF), 기타 오류 시 커넥션 종료
	// return empty 처리
	defer func() {
		if x := recover(); x != nil {
			ret = empty
			logutil.Println("WA183 Read Recover ", x, "\n", string(debug.Stack()))
			this.Close()
		}
	}()

	//logutil.Println("isOpen")
	if this.isOpen() == false {
		return empty
	}

	conf := config.GetConfig()
	// set SetReadDeadline Read i/o timeout 처리 , Read 전에 반복해서 Deadline 설정
	err := session.client.SetReadDeadline(time.Now().Add(time.Duration(conf.TcpSoTimeout) * time.Millisecond))
	// DEBUG goroutine 로그
	//logutil.Println("SetReadDeadline =", time.Now(), ",deadline=", time.Now().Add(time.Duration(conf.TcpSoTimeout)*time.Millisecond))
	if err != nil {
		logutil.Println("WA182 SetReadDeadline failed:", err)
		//return
	}

	tt := this.in.ReadByte()
	code := this.in.ReadByte()
	pcode := this.in.ReadLong()
	oid := this.in.ReadInt()
	transfer_key := this.in.ReadInt()
	if conf.DebugTcpReadEnabled {
		logutil.Infoln("WA182-02", "Tcp Receive ", tt, ", ", code, ", ", pcode, ", ", oid, ", ", transfer_key)
	}

	data := this.in.ReadIntBytesLimit(READ_MAX)
	secu := secure.GetSecurityMaster()
	if pcode != secu.PCODE || oid != secu.OID {
		return empty
	}

	return &TcpReturn{Code: code, Data: data, TransferKey: transfer_key}

}

func (this *TcpSession) getEncryptData(p *TcpSend) (n int, b []byte, err error) {
	conf := config.GetConfig()
	secuTcp := secure.GetSecuritySession()
	if conf.CypherLevel == 0 {
		b = pack.ToBytesPack(p.pack)
		n = len(b)
	} else {
		switch GetSecureMask(p.flag) {
		case NET_SECURE_HIDE:
			if secuTcp.Cypher != nil {
				b = pack.ToBytesPack(p.pack)
				b = secuTcp.Cypher.Hide(b)
				n = len(b)
			} else {
				// send default
				b = pack.ToBytesPack(p.pack)
				n = len(b)
			}
		case NET_SECURE_CYPHER:
			if secuTcp.Cypher != nil {
				b = pack.ToBytesPackECB(p.pack, int(conf.CypherLevel/8)) // 16bytes배수로
				b = secuTcp.Cypher.Encrypt(b)
				n = len(b)
			} else {
				// send default
				b = pack.ToBytesPack(p.pack)
				n = len(b)
			}
		default:
			b := pack.ToBytesPack(p.pack)
			n = len(b)
		}
	}
	if n > int(conf.NetSendMaxBytes) {
		p := pack.NewEventPack()
		p.Level = pack.FATAL
		p.Title = "NEW_OVERFLOW"
		p.Message = fmt.Sprintf("Too big data: %d", p.GetPackType())
		logutil.Println("WA185", p.Title, ",", p.Message)
		err = fmt.Errorf("%s", p.Message)
		Send(NET_SECURE_CYPHER, p, true)
		return n, b, err
	} else {
		return n, b, nil
	}
}
