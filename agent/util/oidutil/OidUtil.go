package oidutil

import (
	"os"
	"strconv"
	"strings"

	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/hexa32"
	"github.com/whatap/golib/util/paramtext"
)

var oidParam = make(map[string]string)

func SetIp(ip string) {
	myIps := strings.Split(ip, ".")
	for i := 0; i < len(myIps); i++ {
		oidParam["ip"+strconv.Itoa(i)] = myIps[i]
	}
}

func SetPort(port string) {
	oidParam["port"] = port
}

func SetHostName(hostName string) {
	oidParam["hostname"] = hostName
}

func SetType(appName string) {
	oidParam["type"] = appName
}

func SetProcess(processName string) {
	oidParam["process"] = processName
}

func SetDocker(docker string) {

	if docker != "" {
		oidParam["docker"] = hexa32.ToString32(int64(hash.HashStr(strings.TrimSpace(docker))))
	} else {
		oidParam["docker"] = hexa32.ToString32(int64(hash.HashStr("none")))
	}
}

func SetIps(ips string) {
	if ips != "" {
		oidParam["ips"] = hexa32.ToString32(int64(hash.HashStr(strings.TrimSpace(ips))))
	} else {
		oidParam["ips"] = hexa32.ToString32(int64(hash.HashStr("noneIP")))
	}
}

func SetOidParam(k, v string) {
	oidParam[k] = v
}
func SetOidParamHexa32(k, v string) {
	if v != "" {
		oidParam[k] = hexa32.ToString32(int64(hash.HashStr(strings.TrimSpace(v))))
	}
}

func MakeOname(pattern string) string {
	paramText := paramtext.NewParamTextBrace(pattern, "{", "}")
	keyList := paramText.GetKeys()

	oname := pattern
	for _, key := range keyList {
		var value string
		if strings.HasPrefix("env.", key) && len(key) > 4 {
			value = os.Getenv(key[4:])
		} else {
			value = oidParam[key]
		}
		if value != "" {
			key = "{" + key + "}"
		} else {
			value = key
		}
		oname = strings.Replace(oname, key, value, -1)
	}
	return oname
}
