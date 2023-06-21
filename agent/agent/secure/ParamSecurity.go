package secure

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/whatap/golib/lang/ref"
	"github.com/whatap/golib/util/compare"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/keygen"
	"github.com/whatap/golib/util/stringutil"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/util/logutil"
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
	var f *os.File

	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA1000", " Recover ", r)
		}
		if f != nil {
			f.Close()
		}
	}()
	home := config.GetWhatapHome()

	stat, err := os.Stat(filepath.Join(home, "paramkey.txt"))

	// 파일이 없으면 새로 생성.
	if os.IsNotExist(err) {
		f, err := os.Create(filepath.Join(home, "paramkey.txt"))
		if err != nil {
			logutil.Println("WA1001", " Create File Error", err)
			return
		}

		b := this.getKey()

		f.Write(b)
		f.Sync()
		stat, err := f.Stat()
		if err != nil {
			this.lasttime = stat.ModTime().Unix()
			this.key = b
			this.KeyHash = hash.Hash(this.key)
		}

	} else {
		// 수정사항이 없으면 종료
		if stat.ModTime().Unix() == this.lasttime {
			return
		}

		f, err := os.Open(filepath.Join(home, "paramkey.txt"))
		if err != nil {
			logutil.Println("WA1002", " Open File Error", err)
			return
		}

		this.lasttime = stat.ModTime().Unix()
		b := make([]byte, 512)
		//b := FileUtil.readAll(f);
		n, err := f.Read(b)
		if err != nil {
			logutil.Println("WA1003", " Read File Error", err)
		} else {
			if n > 3 {
				//this.key = []byte(strings.TrimSpace(strings.ToUpper(string(b[0:n]))))
				// 2017.11.02 대문자 변경 삭제
				this.key = []byte(strings.TrimSpace(string(b[0:n])))
				this.KeyHash = hash.Hash(this.key)
			} else {
				this.key = []byte("WHATAP")
				this.KeyHash = hash.Hash(this.key)
			}
		}
	}

	if f != nil {
		f.Close()
	}
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

func ParamSecurityMain() {
	s := "A112fda12fafa34"
	c1 := ref.NewBYTE()
	b := paramSecurity.Encrypt([]byte(s), c1)

	c2 := ref.NewBYTE()

	fmt.Println(string(paramSecurity.Decrypt(b, c2, []byte("WHATAP"))))
	fmt.Println(len("WHATAP"))
	fmt.Println(c1.Value == c2.Value)
}
