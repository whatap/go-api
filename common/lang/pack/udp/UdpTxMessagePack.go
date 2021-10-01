package udp

import (
	"fmt"

	"github.com/whatap/go-api/common/io"
)

type UdpTxMessagePack struct {
	AbstractPack
	Hash  string
	Value string
	Desc  string
}

func NewUdpTxMessagePack() *UdpTxMessagePack {
	p := new(UdpTxMessagePack)
	p.Ver = UDP_PACK_VERSION
	p.AbstractPack.Flush = false
	return p
}

func NewUdpTxMessagePackVer(ver int32) *UdpTxMessagePack {
	p := new(UdpTxMessagePack)
	p.Ver = ver
	p.AbstractPack.Flush = false
	return p
}

func (this *UdpTxMessagePack) GetPackType() uint8 {
	return TX_MSG
}

func (this *UdpTxMessagePack) ToString() string {
	return fmt.Sprint(this.AbstractPack.ToString(), ",hash=", this.Hash, ",value=", this.Value, ",desc=", this.Desc)
}

func (this *UdpTxMessagePack) Clear() {
	this.AbstractPack.Clear()
	this.AbstractPack.Flush = false

	this.Hash = ""
	this.Value = ""
	this.Desc = ""
}

func (this *UdpTxMessagePack) Write(dout *io.DataOutputX) {
	this.AbstractPack.Write(dout)
	dout.WriteTextShortLength(this.Hash)
	dout.WriteTextShortLength(this.Value)
	dout.WriteTextShortLength(this.Desc)
}

func (this *UdpTxMessagePack) Read(din *io.DataInputX) {
	this.AbstractPack.Read(din)

	this.Hash = din.ReadTextShortLength()
	this.Value = din.ReadTextShortLength()
	this.Desc = din.ReadTextShortLength()
}
func (this *UdpTxMessagePack) Process() {
}
