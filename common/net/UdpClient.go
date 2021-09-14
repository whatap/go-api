//github.com/whatap/go-api/common/net
package net

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/whatap/go-api/common/io"
	"github.com/whatap/go-api/common/lang/pack/udp"
	"github.com/whatap/go-api/common/util/dateutil"
	"github.com/whatap/go-api/config"
)

const (
	UDP_READ_MAX                    = 64 * 1024
	UDP_PACKET_BUFFER               = 64 * 1024
	UDP_PACKET_BUFFER_CHUNKED_LIMIT = 48 * 1024
	UDP_PACKET_CHANNEL_MAX          = 255
	UDP_PACKET_FLUSH_TIMEOUT        = 10 * 1000

	UDP_PACKET_HEADER_SIZE = 9
	// typ pos 0
	UDP_PACKET_HEADER_TYPE_POS = 0
	// ver pos 1
	UDP_PACKET_HEADER_VER_POS = 1
	// len pos 5
	UDP_PACKET_HEADER_LEN_POS = 5

	UDP_PACKET_SQL_MAX_SIZE = 32768
)

type UdpClient struct {
	host string
	port int

	udp net.Conn
	wr  *bufio.Writer

	sendCh       chan *UdpData
	lastSendTime int64

	lock sync.Mutex
}

type UdpData struct {
	Type  byte
	Ver   int32
	Data  []byte
	Flush bool
}

//
var udpClient *UdpClient

func GetUdpClient() *UdpClient {
	if udpClient != nil {
		return udpClient
	}
	udpClient = new(UdpClient)
	udpClient.host = "127.0.0.1"
	udpClient.port = 6600
	udpClient.sendCh = make(chan *UdpData, UDP_PACKET_CHANNEL_MAX)
	udpClient.open()
	go func() {
		time.Sleep(1000 * time.Millisecond)
		for {
			for udpClient.isOpen() {
				time.Sleep(5000 * time.Millisecond)
			}
			for udpClient.open() == false {
				time.Sleep(5000 * time.Millisecond)
			}
		}
	}()
	go udpClient.process()
	go udpClient.receive()
	return udpClient
}

func (this *UdpClient) open() (ret bool) {
	if this.isOpen() {
		return true
	}
	udpClient, err := net.DialTimeout("udp", fmt.Sprintf("%s:%d", this.host, this.port), time.Duration(60000)*time.Millisecond)
	if err != nil {
		fmt.Println("UDP", "Connect error. "+this.host+":", this.port)
		this.Close()
		return false
	}
	this.udp = udpClient
	this.wr = bufio.NewWriterSize(this.udp, UDP_PACKET_BUFFER)
	//logutil.Printf("UDP", "Connected %s:%d", this.host, this.port)

	return true
}
func (this *UdpClient) isOpen() bool {
	return this.udp != nil && this.wr != nil
}

func (this *UdpClient) GetLocalAddr() net.Addr {
	return this.udp.LocalAddr()
}

func (this *UdpClient) Send(p udp.UdpPack) {
	dout := udp.WritePack(io.NewDataOutputX(), p)
	this.sendQueue(p.GetPackType(), p.GetVersion(), dout.ToByteArray(), p.IsFlush())
	udp.ClosePack(p)
}

func (this *UdpClient) sendQueue(t uint8, ver int32, b []byte, flush bool) bool {
	this.lock.Lock()
	defer func() {
		// recover for closed channel
		if r := recover(); r != nil {
		}
		this.lock.Unlock()
	}()

	fmt.Println("add send data to channel type=", t, ", ver=", ver)

	if udpClient.open() {
		buff := make([]byte, len(b))
		copy(buff, b)
		this.sendCh <- &UdpData{t, ver, buff, flush}
		return true
	} else {
		return false
	}

}

func (this *UdpClient) process() {
	for {
		select {
		case sendData := <-this.sendCh:
			this.send(sendData)
		default:
			if !this.isOpen() {
				continue
			}
			time.Sleep(1 * time.Second)
			// 시간 비교하여 발송.

			if this.wr.Buffered() > 0 && dateutil.SystemNow()-this.lastSendTime > UDP_PACKET_FLUSH_TIMEOUT {
				this.lastSendTime = dateutil.Now()
				if err := this.wr.Flush(); err != nil {
					fmt.Println("UDP", "Error time Flush ", err)
					this.Close()
					continue
				}
			}
		}
	}
}
func (this *UdpClient) send(sendData *UdpData) {
	if !this.isOpen() {
		return
	}
	out := io.NewDataOutputX()
	out.WriteByte(sendData.Type)
	out.WriteInt(sendData.Ver)
	out.WriteIntBytes(sendData.Data)
	sendBytes := out.ToByteArray()
	fmt.Println("get send data from channel type=", sendData.Type, ", ver=", sendData.Ver)
	if this.wr.Buffered() > 0 && this.wr.Buffered()+len(sendBytes) > UDP_PACKET_BUFFER_CHUNKED_LIMIT {
		this.lastSendTime = dateutil.Now()
		if err := this.wr.Flush(); err != nil {
			fmt.Println("UDP", "Error Flush ", err)
			this.Close()
			return
		}
	}
	if _, err := this.wr.Write(sendBytes); err != nil {
		fmt.Println("UDP", "Error Write send=", len(sendBytes), ",err=", err)
		this.Close()
		return
	}
	// flush == true
	if this.wr.Buffered() > 0 && sendData.Flush {
		this.lastSendTime = dateutil.Now()
		if err := this.wr.Flush(); err != nil {
			fmt.Println("UDP", "Error force Flush ", err)
			this.Close()
			return
		}
	} else {
		fmt.Println("UDP", "force Flush ")
	}
}

func (this *UdpClient) receive() {
	buff := make([]byte, UDP_PACKET_BUFFER)

	for {
		if this.udp != nil {
			udpNet := this.udp.(*net.UDPConn)
			if n, addr, err := udpNet.ReadFromUDP(buff); err == nil {
				fmt.Println("Udp Receive ", n, ", clientAddr=", addr)
			} else {
				fmt.Println("Error", "Udp Receive", err)
				continue
			}
			offset := 0
			t := uint8(buff[offset])
			v := io.ToInt(buff[offset+UDP_PACKET_HEADER_VER_POS:offset+UDP_PACKET_HEADER_VER_POS+4], 0)
			l := io.ToInt(buff[offset+UDP_PACKET_HEADER_LEN_POS:offset+UDP_PACKET_HEADER_LEN_POS+4], 0)

			fmt.Println("Receive UDP t=", t, ", v=", v, ", l=", l)
			offset += UDP_PACKET_HEADER_SIZE

			tmp := buff[offset : offset+int(l)]
			offset += int(l)
			switch t {
			case udp.CONFIG_INFO:
				p := udp.ToPack(t, v, tmp)
				fmt.Println("Receive CONFIG_INFO ", p)
				if p != nil {
					config.GetConfig().ApplyConfig(p.(*udp.UdpConfigPack).MapData)
				}
			}
		}
	}
}
func (this *UdpClient) Close() {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.udp != nil {
		defer func() {
			recover()
			this.udp = nil
		}()
		// block to receive
		close(this.sendCh)
		// send all remaining data
		for sendData := range this.sendCh {
			this.send(sendData)
		}
		this.udp.Close()
		fmt.Println("UDP", "Closed ")
	}
	this.udp = nil
	this.wr = nil
}

func UdpShutdown() {
	if udpClient != nil {
		udpClient.Close()
	}
}
