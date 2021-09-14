package udp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/whatap/go-api/common/io"
	"github.com/whatap/go-api/common/util/urlutil"
)

type UdpTxStartPack struct {
	AbstractPack
	// Pack
	Host             string
	Uri              string
	Ipaddr           string
	UAgent           string
	Ref              string
	WClientId        string
	HttpMethod       string
	IsStaticContents string

	//Processing data
	ServiceURL *urlutil.URL
	RefererURL *urlutil.URL
	IsStatic   bool
}

func NewUdpTxStartPack() *UdpTxStartPack {
	p := new(UdpTxStartPack)
	p.Ver = UDP_PACK_VERSION
	p.AbstractPack.Flush = true
	return p
}
func NewUdpTxStartPackVer(ver int32) *UdpTxStartPack {
	p := new(UdpTxStartPack)
	p.Ver = ver
	p.AbstractPack.Flush = true
	return p
}

func (this *UdpTxStartPack) GetPackType() uint8 {
	return TX_START
}

func (this *UdpTxStartPack) ToString() string {
	return fmt.Sprint(this.AbstractPack.ToString(), ",host=", this.Host, ",uri=", this.Uri)
}

func (this *UdpTxStartPack) Clear() {
	this.AbstractPack.Clear()
	this.AbstractPack.Flush = true

	// Pack
	this.Host = ""
	this.Uri = ""
	this.Ipaddr = ""
	this.UAgent = ""
	this.Ref = ""
	this.WClientId = ""
	this.HttpMethod = ""
	this.IsStaticContents = ""

	//Processing data
	this.ServiceURL = nil
	this.RefererURL = nil
	this.IsStatic = false
}
func (this *UdpTxStartPack) SetStaticContents(b bool) {
	this.IsStatic = b
	if b {
		this.IsStaticContents = "1"
	} else {
		this.IsStaticContents = "0"
	}
}

func (this *UdpTxStartPack) Write(dout *io.DataOutputX) {
	this.AbstractPack.Write(dout)
	dout.WriteTextShortLength(this.Host)
	dout.WriteTextShortLength(this.Uri)
	dout.WriteTextShortLength(this.Ipaddr)
	dout.WriteTextShortLength(this.UAgent)
	dout.WriteTextShortLength(this.Ref)
	dout.WriteTextShortLength(this.WClientId)
	if this.Ver > 40000 {
		// Batch
	} else if this.Ver > 30000 {
		// Dotnet
		dout.WriteTextShortLength(this.IsStaticContents)
	} else if this.Ver > 20000 {
		// Python
		dout.WriteTextShortLength(this.IsStaticContents)
	} else {
		// PHP
		if this.Ver >= 10103 {
			dout.WriteTextShortLength(this.HttpMethod)
		}
	}
}

func (this *UdpTxStartPack) Read(din *io.DataInputX) {
	this.AbstractPack.Read(din)

	this.Host = din.ReadTextShortLength()
	this.Uri = din.ReadTextShortLength()
	this.Ipaddr = din.ReadTextShortLength()
	this.UAgent = din.ReadTextShortLength()
	this.Ref = din.ReadTextShortLength()
	this.WClientId = din.ReadTextShortLength()

	if this.Ver > 40000 {
		// Batch
	} else if this.Ver > 30000 {
		// Dotnet
		this.IsStaticContents = din.ReadTextShortLength()
	} else if this.Ver > 20000 {
		// Python
		this.IsStaticContents = din.ReadTextShortLength()
	} else {
		// PHP
		if this.Ver >= 10103 {
			this.HttpMethod = din.ReadTextShortLength()
		}
	}
}

func (this *UdpTxStartPack) Process() {
	if strings.HasPrefix(this.Uri, "/") {
		this.ServiceURL = urlutil.NewURL(this.Host + this.Uri)
	} else {
		this.ServiceURL = urlutil.NewURL(this.Host + "/" + this.Uri)
	}
	this.RefererURL = urlutil.NewURL(this.Ref)
	if this.IsStaticContents == "" {
		this.IsStatic = false
	} else if b, err := strconv.ParseBool(this.IsStaticContents); err == nil {
		this.IsStatic = b
	}
}
