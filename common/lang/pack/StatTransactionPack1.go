package pack

import (
	"container/list"
	"fmt"

	"github.com/whatap/go-api/common/io"
	"github.com/whatap/go-api/common/util/hmap"
)

type StatTransactionPack1 struct {
	AbstractPack
	// [] byte
	Records []byte
	// int
	RecordCount int

	Spec int

	Version byte
}

func NewStatTransactionPack1() *StatTransactionPack1 {
	p := new(StatTransactionPack1)
	//2021.06.28
	p.Version = 4
	return p
}

func (this *StatTransactionPack1) GetPackType() int16 {
	return PACK_STAT_SERVICE_1
}

func (this *StatTransactionPack1) ToString() string {
	//	sb.Append(",bytes=" + ArrayUtil.len(records));

	return fmt.Sprintln("StatService1 ", this.Oid, ",", this.Pcode, ",", this.Time, ",records=", this.RecordCount, ",spec=", this.Spec, ",bytes=", len(this.Records))
}

func (this *StatTransactionPack1) Write(dout *io.DataOutputX) {
	this.AbstractPack.Write(dout)
	dout.WriteBlob(this.Records)
	dout.WriteDecimal(int64(this.RecordCount))
	dout.WriteByte(0) //version
	dout.WriteDecimal(int64(this.Spec))
}

func (this *StatTransactionPack1) Read(din *io.DataInputX) {
	this.AbstractPack.Read(din)
	this.Records = din.ReadBlob()
	this.RecordCount = int(din.ReadDecimal())
	din.ReadByte()
	this.Spec = int(din.ReadDecimal())
	//return this
}

func (this *StatTransactionPack1) SetRecords(size int, items hmap.Enumeration) *StatTransactionPack1 {
	o := io.NewDataOutputX()
	o.WriteShort(int16(size))
	for i := 0; i < size; i++ {
		//fmt.Println("StatTransactionPack1:SetRecords i=", i)
		WriteTransactionRec(o, items.NextElement().(*TransactionRec), this.Version)
	}
	this.Records = o.ToByteArray()
	this.RecordCount = size
	return this
}

func (this *StatTransactionPack1) SetRecordsList(items *list.List) *StatTransactionPack1 {
	o := io.NewDataOutputX()
	size := items.Len()
	o.WriteShort(int16(size))
	for e := items.Front(); e != nil; e = e.Next() {
		WriteTransactionRec(o, e.Value.(*TransactionRec), this.Version)
	}
	this.Records = o.ToByteArray()
	this.RecordCount = size
	return this
}

func (this *StatTransactionPack1) GetRecords() *list.List {
	items := list.New()

	if this.Records == nil {
		return nil
	}
	in := io.NewDataInputX(this.Records)
	size := int(in.ReadShort()) & 0xffff
	for i := 0; i < size; i++ {
		items.PushBack(ReadTransactionRec(in))
	}
	return items
}
