package secure

import (
	"math"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/magiconair/properties"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/golib/lang/ref"
	"github.com/whatap/golib/util/compare"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/keygen"
	"github.com/whatap/golib/util/stringutil"
)

type ParamSecurity struct {
	lasttime int64
	key      []byte
	KeyHash  int32
}

func NewParamSecurity() *ParamSecurity {
	p := new(ParamSecurity)
	p.lasttime = -1
	p.key = []byte("WHATAP")
	p.KeyHash = hash.Hash(p.key)

	return p
}

var paramSecurity *ParamSecurity
var paramSecurityMutex = sync.Mutex{}

func GetParamSecurity() *ParamSecurity {
	if paramSecurity != nil {
		return paramSecurity
	}
	paramSecurityMutex.Lock()
	defer paramSecurityMutex.Unlock()

	if paramSecurity != nil {
		return paramSecurity
	}
	paramSecurity = NewParamSecurity()

	return paramSecurity
}

func (this *ParamSecurity) Reload() {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA1000", " Recover ", r, ", stack \n", string(debug.Stack()))
		}
	}()
	home := config.GetWhatapHome()
	this.loadConf(home)
}

func (this *ParamSecurity) loadConf(home string) {
	filename := filepath.Join(home, "security.conf")
	stat, err := os.Stat(filename)
	if err != nil && os.IsNotExist(err) {
		if err := this.loadParamkey(home); err == nil {
			this.save(filename, string(this.key))
		} else {
			this.save(filename, "WHATAP")
		}
	}
	stat, err = os.Stat(filename)
	// 수정사항이 없으면 종료
	if stat.ModTime().Unix() == this.lasttime {
		return
	}
	this.lasttime = stat.ModTime().Unix()
	if err := this.load(filename); err != nil {
		logutil.Println("WA1001", "load security.conf error ", err)
	}
}

func (this *ParamSecurity) load(filename string) error {
	prop, err := properties.LoadFile(filename, properties.UTF8)
	if err != nil {
		return err
	}
	if pkey, ok := prop.Get("paramkey"); ok {
		this.key = []byte(strings.TrimSpace(pkey))
		this.KeyHash = hash.Hash(this.key)
	} else {
		this.key = []byte("WHATAP")
		this.KeyHash = hash.Hash(this.key)
	}
	return nil
}

func (this *ParamSecurity) loadParamkey(home string) error {
	f, err := os.Open(filepath.Join(home, "paramkey.txt"))
	if err != nil {
		return err
	}
	defer f.Close()

	b := make([]byte, 512)
	n, err := f.Read(b)
	if err != nil {
		logutil.Println("WA1003", " Read File Error", err)
		return err
	} else {
		if n > 3 {
			this.key = []byte(strings.TrimSpace(string(b[0:n])))
			this.KeyHash = hash.Hash(this.key)
		} else {
			this.key = []byte("WHATAP")
			this.KeyHash = hash.Hash(this.key)
		}
	}
	return nil
}

func (this *ParamSecurity) save(filename string, password string) error {
	_, err := os.Stat(filename)
	// 파일이 없으면 새로 생성.
	if err != nil && os.IsNotExist(err) {
		f, err1 := os.Create(filename)
		if err1 != nil {
			logutil.Println("WA1004", " Create File Error", err1)
			return err1
		}
		defer f.Close()

		prop := properties.NewProperties()
		prop.Set("paramkey", password)
		if _, err2 := prop.Write(f, properties.UTF8); err2 != nil {
			logutil.Println("WA1005", " Write File security.conf Error", err2)
			return err2
		}
	}
	return nil
}

func (this *ParamSecurity) getKey() []byte {
	s := "ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	sb := stringutil.NewStringBuffer()
	var x int
	for i := 0; i < 6; i++ {
		// Java 와 값이 똑 같은지 확인 필요
		//x = int(math.Abs(float64(keygen.Next())) % len(s))
		x = int(math.Abs(float64(keygen.Next()))) % len(s)
		sb.Append(string(s[x]))
	}
	return []byte(sb.ToString())
}

func (this *ParamSecurity) IsSafe(skey string) bool {
	if skey == "" || len(skey) == 0 {
		return false
	}
	// TODO
	return (compare.CompareToBytes(this.key, []byte(skey)) == 0)
}

func (this *ParamSecurity) Encrypt(b []byte, crc *ref.BYTE) []byte {
	j := 0
	//logutil.Println("Before Encrypt", b)

	for i := 0; i < len(b); i++ {
		crc.Value ^= b[i]
		b[i] ^= this.key[j]

		j = (j + 1) % len(this.key)
	}

	//logutil.Println("Encrypt", b)

	return b
}
func (this *ParamSecurity) Decrypt(b []byte, crc *ref.BYTE, dkey []byte) []byte {
	j := 0
	for i := 0; i < len(b); i++ {
		b[i] ^= dkey[j]
		crc.Value ^= b[i]

		j = (j + 1) % len(dkey)
	}
	return b
}
