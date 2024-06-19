package secure

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/golib/lang/ref"
)

func TestParamSecurity(t *testing.T) {
	home := "./"
	os.Setenv("WHATAP_HOME", home)
	config.GetConfig()

	var err error

	data := []byte("XIEKFLE")

	err = ioutil.WriteFile(filepath.Join(home, "paramkey.txt"), data, 0644)
	assert.Nil(t, err)
	assert.FileExists(t, filepath.Join(home, "paramkey.txt"))

	paramSecurity := NewParamSecurity()
	assert.Equal(t, "WHATAP", string(paramSecurity.key))

	paramSecurity.Reload()
	assert.Equal(t, "XIEKFLE", string(paramSecurity.key))
	assert.FileExists(t, filepath.Join(home, "security.conf"))

	// remove file
	err = os.Remove(filepath.Join(home, "paramkey.txt"))
	assert.Nil(t, err)

	err = os.Remove(filepath.Join(home, "security.conf"))
	assert.Nil(t, err)

	paramSecurity = NewParamSecurity()
	assert.Equal(t, "WHATAP", string(paramSecurity.key))

	paramSecurity.Reload()
	assert.FileExists(t, filepath.Join(home, "security.conf"))
	assert.Equal(t, "WHATAP", string(paramSecurity.key))

	// remove file
	err = os.Remove(filepath.Join(home, "security.conf"))
	assert.Nil(t, err)

}

func TestParamSecurityEncode(t *testing.T) {
	home := "./"
	os.Setenv("WHATAP_HOME", home)
	config.GetConfig()

	paramSecurity := GetParamSecurity()
	assert.Equal(t, "WHATAP", string(paramSecurity.key))

	s := "A112fda12fafa34"
	c1 := ref.NewBYTE()
	b := paramSecurity.Encrypt([]byte(s), c1)

	c2 := ref.NewBYTE()

	assert.Equal(t, s, string(paramSecurity.Decrypt(b, c2, []byte("WHATAP"))))
	assert.Equal(t, c1.Value, c2.Value)
}
