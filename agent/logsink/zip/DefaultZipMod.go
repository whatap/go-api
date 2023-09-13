package zip

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
)

type DefaultZipMod struct {
	id byte
}

func NewDefaultZipMod() *DefaultZipMod {
	p := new(DefaultZipMod)
	return p
}
func (this *DefaultZipMod) ID() byte {
	return ZIP_MOD_DEFULAT_GZIP
}

func (this *DefaultZipMod) Compress(in []byte) (output []byte, err error) {
	if in == nil {
		err = fmt.Errorf("error input data is nil ")
		return
	}
	buf := new(bytes.Buffer)

	gz := gzip.NewWriter(buf)
	gz.Write(in)
	gz.Flush()

	// gz.Close 가 호출 되어야만 buf.Bytes 내용이 정상 출력 됨
	err = gz.Close()
	if err == nil {
		output = buf.Bytes()
	}
	return
}

func (this *DefaultZipMod) Decompress(in []byte) ([]byte, error) {
	r, err := gzip.NewReader(ioutil.NopCloser(bytes.NewBuffer(in)))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	if b, err := ioutil.ReadAll(r); err != nil {
		return nil, err
	} else {
		return b, nil
	}
}
