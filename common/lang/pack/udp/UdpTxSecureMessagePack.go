package udp

import (
	"fmt"

	"github.com/whatap/go-api/common/io"
)

type UdpTxSecureMessagePack struct {
	AbstractPack
	Hash  string
	Value string
	Desc  string
}

func NewUdpTxSecureMessagePack() *UdpTxSecureMessagePack {
	p := new(UdpTxSecureMessagePack)
	p.Ver = UDP_PACK_VERSION
	p.AbstractPack.Flush = false
	return p
}

func NewUdpTxSecureMessagePackVer(ver int32) *UdpTxSecureMessagePack {
	p := new(UdpTxSecureMessagePack)
	p.Ver = ver
	p.AbstractPack.Flush = false
	return p
}

func (this *UdpTxSecureMessagePack) GetPackType() uint8 {
	return TX_SECURE_MSG
}

func (this *UdpTxSecureMessagePack) ToString() string {
	return fmt.Sprint(this.AbstractPack.ToString(), ",hash=", this.Hash, ",value=", this.Value, ",desc=", this.Desc)
}

func (this *UdpTxSecureMessagePack) Clear() {
	this.AbstractPack.Clear()
	this.AbstractPack.Flush = false

	this.Hash = ""
	this.Value = ""
	this.Desc = ""
}

func (this *UdpTxSecureMessagePack) Write(dout *io.DataOutputX) {
	this.AbstractPack.Write(dout)
	dout.WriteTextShortLength(this.Hash)
	dout.WriteTextShortLength(this.Value)
	dout.WriteTextShortLength(this.Desc)
	if this.Ver > 40000 {
		// Batch
	} else if this.Ver > 30000 {
		// Dotnet
	} else if this.Ver > 20000 {
		// Python
	} else {
		// PHP
	}
}

func (this *UdpTxSecureMessagePack) Read(din *io.DataInputX) {
	this.AbstractPack.Read(din)

	this.Hash = din.ReadTextShortLength()
	this.Value = din.ReadTextShortLength()
	this.Desc = din.ReadTextShortLength()
	if this.Ver > 40000 {
		// Batch
	} else if this.Ver > 30000 {
		// Dotnet
	} else if this.Ver > 20000 {
		// Python
	} else {
		// PHP
	}
}
func (this *UdpTxSecureMessagePack) Process() {
	if this.Ver > 40000 {
		// Batch
	} else if this.Ver > 30000 {
		// Dotnet
	} else if this.Ver > 20000 {
		// Python
	} else {
		// PHP
	}
}
