//github.com/whatap/go-api/config
package config

import (
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/whatap/go-api/common/util/hash"
	"github.com/whatap/go-api/common/util/stringutil"
)

const ()

type Config struct {
	m map[string]string

	PCODE      int64
	OID        int64
	OKIND      int32
	OKIND_NAME string
	ONODE      int32
	ONODE_NAME string

	AppType int16 // private
	Enabled bool

	NetUdpHost string
	NetUdpPort int32

	TransactionEnabled bool
	//profile
	ProfileHttpHeaderEnabled   bool   // 수집 유무(HTTP-HEADERS)
	ProfileHttpHeaderUrlPrefix string // 수집 url prefix

	ProfileHttpParameterEnabled   bool   // 수집 유무(HTTP-PARAMETERS)
	ProfileHttpParameterUrlPrefix string // 수집 url prefix

	TraceUserEnabled             bool // TODO:  UseridUtil으로 기본값인 whatap cookie설정 해야함.
	TraceUserUsingIp             bool
	TraceUserHeaderTicket        string // TODO: UseridUtil
	TraceUserHeaderTicketEnabled bool   // private
	TraceUserSetCookie           bool
	TraceUserCookieLimit         int32
	TraceUserCookieKey           string // 세션 쿠키 키 이름 별로도 지정.
	TraceUserUsingType           int32  // private

	TraceHttpClientIpHeaderKeyEnabled bool
	TraceHttpClientIpHeaderKey        string // TODO: getRemoteAddr

	MtraceEnabled        bool
	MtraceRate           int32
	TraceMtraceCallerKey string
	TraceMtraceCalleeKey string
	TraceMtraceInfoKey   string
	TraceMtracePoidKey   string
	TraceMtraceSpecKey   string
	TraceMtraceSpecKey1  string

	MtraceSendUrlLength int32
	MtraceSpec          string
	MtraceSpecHash      int32

	Debug bool
}

var conf *Config = nil
var mutex = sync.Mutex{}
var AppType int16 = 3

func GetConfig() *Config {
	mutex.Lock()
	defer mutex.Unlock()
	if conf != nil {
		return conf
	}
	conf = new(Config)
	conf.ApplyDefault()
	return conf
}

func GetWhatapHome() string {
	home := os.Getenv("WHATAP_HOME")
	if home == "" {
		home = "."
	}
	return home
}
func (conf *Config) ApplyDefault() {
	m := make(map[string]string)
	m["enabled"] = "true"
	m["net_udp_port"] = "6600"
	m["transaction_enabled"] = "true"
	m["profile_http_header_enabled"] = "false"
	m["profile_http_header_url_prefix"] = "/"
	m["profile_http_parameter_enabled"] = "false"
	m["profile_http_parameter_url_prefix"] = "/"

	m["trace_user_enabled"] = "true"
	m["trace_user_using_ip"] = "false"
	m["trace_user_header_ticket"] = ""
	m["trace_user_set_cookie"] = "false"
	m["trace_user_cookie_limit"] = "2048"
	m["trace_user_cookie_key"] = ""
	m["trace_http_client_ip_header_key_enabled"] = "true"
	m["trace_http_client_ip_header_key"] = "x-forwarded-for"

	m["mtrace_enabled"] = "true"
	m["mtrace_caller_key"] = "x-wtap-mst"
	m["mtrace_callee_key"] = "x-wtap-tx"
	m["mtrace_info_key"] = "x-wtap-inf"
	m["mtrace_poid_key"] = "x-wtap-po"
	m["mtrace_spec_key"] = "x-wtap-sp"
	m["mtrace_spec_key1"] = "x-wtap-sp1"
	m["mtrace_send_url_length"] = "80"
	m["mtrace_spec"] = "ver1.0"
	m["mtrace_rate"] = "100"

	m["debug"] = "false"

	conf.ApplyConfig(m)
}
func (conf *Config) ApplyConfig(m map[string]string) {
	if conf.m == nil {
		conf.m = m
	} else {
		for k, v := range m {
			conf.m[k] = v
		}
	}

	conf.PCODE = conf.getLong("pcode", 0)
	conf.OID = conf.getLong("oid", 0)
	conf.OKIND = conf.getInt("okind", 0)
	conf.OKIND_NAME = conf.getValueDef("okind_name", "")
	conf.ONODE = conf.getInt("onode", 0)
	conf.ONODE_NAME = conf.getValueDef("onode_name", "")

	conf.Enabled = conf.getBoolean("enabled", true)
	conf.NetUdpHost = conf.getValueDef("net_udp_host", "127.0.0.1")
	conf.NetUdpPort = conf.getInt("net_udp_port", 6600)

	conf.TransactionEnabled = conf.Enabled && conf.getBoolean("transaction_enabled", true)

	conf.ProfileHttpHeaderEnabled = conf.getBoolean("profile_http_header_enabled", false)
	conf.ProfileHttpHeaderUrlPrefix = conf.getValueDef("profile_http_header_url_prefix", "/")

	conf.ProfileHttpParameterEnabled = conf.getBoolean("profile_http_parameter_enabled", false)
	conf.ProfileHttpParameterUrlPrefix = conf.getValueDef("profile_http_parameter_url_prefix", "/")

	conf.TraceUserEnabled = conf.getBoolean("trace_user_enabled", true)
	conf.TraceUserUsingIp = conf.getBoolean("trace_user_using_ip", false)
	conf.TraceUserHeaderTicket = conf.getValue("trace_user_header_ticket")
	conf.TraceUserHeaderTicketEnabled = stringutil.IsNotEmpty(conf.TraceUserHeaderTicket)
	conf.TraceUserSetCookie = conf.getBoolean("trace_user_set_cookie", false)
	conf.TraceUserCookieLimit = conf.getInt("trace_user_cookie_limit", 2048)
	conf.TraceUserCookieKey = conf.getValueDef("trace_user_cookie_key", "")

	conf.TraceUserUsingType = 2 // default
	if !conf.TraceUserEnabled {
		conf.TraceUserUsingType = 0
	} else if conf.TraceUserUsingIp {
		conf.TraceUserUsingType = 1 // IP
	} else {
		conf.TraceUserUsingType = 2 // COOKIE
	}

	conf.TraceHttpClientIpHeaderKeyEnabled = conf.getBoolean("trace_http_client_ip_header_key_enabled", true)
	conf.TraceHttpClientIpHeaderKey = conf.getValue("trace_http_client_ip_header_key")
	conf.MtraceEnabled = conf.getBoolean("mtrace_enabled", false)
	conf.MtraceRate = conf.getInt("mtrace_rate", 10)
	conf.TraceMtraceCallerKey = conf.getValueDef("mtrace_caller_key", "x-wtap-mst")
	conf.TraceMtraceCalleeKey = conf.getValueDef("mtrace_callee_key", "x-wtap-tx")
	conf.TraceMtraceInfoKey = conf.getValueDef("mtrace_info_key", "x-wtap-inf")
	conf.TraceMtracePoidKey = conf.getValueDef("mtrace_poid_key", "x-wtap-po")
	conf.TraceMtraceSpecKey = conf.getValueDef("mtrace_spec_key", "x-wtap-sp")
	conf.TraceMtraceSpecKey1 = conf.getValueDef("mtrace_spec_key1", "x-wtap-sp1")
	conf.MtraceSendUrlLength = conf.getInt("mtrace_send_url_length", 80)
	conf.MtraceSpec = conf.getValueDef("mtrace_spec", "")
	if conf.MtraceSpec == "" {
		conf.MtraceSpecHash = 0
	} else {
		conf.MtraceSpec = strings.ReplaceAll(conf.MtraceSpec, ",", "_")
		conf.MtraceSpecHash = hash.HashStr(conf.MtraceSpec)
	}

	conf.Debug = conf.GetBoolean("debug", false)

}
func (conf *Config) GetValue(key string) string { return conf.getValue(key) }
func (conf *Config) getValue(key string) string {
	if v, ok := conf.m[key]; ok {
		return strings.TrimSpace(v)
	}
	return os.Getenv(key)
}
func (conf *Config) GetValueDef(key, def string) string { return conf.getValueDef(key, def) }
func (conf *Config) getValueDef(key string, def string) string {
	v := conf.getValue(key)

	if v == "" {
		return def
	}

	return v
}
func (conf *Config) GetBoolean(key string, def bool) bool {
	return conf.getBoolean(key, def)
}
func (conf *Config) getBoolean(key string, def bool) bool {
	v := conf.getValue(key)
	if v == "" {
		return def
	}
	value, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return value
}
func (conf *Config) GetInt(key string, def int) int32 {
	return conf.getInt(key, def)
}
func (conf *Config) getInt(key string, def int) int32 {
	v := conf.getValue(key)
	if v == "" {
		return int32(def)
	}
	value, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		return int32(def)
	}
	return int32(value)
}

func (conf *Config) GetIntSet(key, defaultValue, deli string) []int32 {
	set := make([]int32, 0)
	vv := stringutil.Tokenizer(conf.GetValueDef(key, defaultValue), deli)
	if vv != nil {
		for _, x := range vv {
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Continue
					}
				}()
				if xx, err := strconv.Atoi(strings.TrimSpace(x)); err != nil {
					set = append(set, int32(xx))
				}
			}()
		}
	}
	return set
}

func (conf *Config) GetStringHashSet(key, defaultValue, deli string) []int32 {
	set := make([]int32, 0)
	vv := stringutil.Tokenizer(conf.GetValueDef(key, defaultValue), deli)
	if vv != nil {
		for _, x := range vv {
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Continue
					}
				}()
				xx := hash.HashStr(strings.TrimSpace(x))
				set = append(set, xx)
			}()
		}
	}
	return set
}

func (conf *Config) GetStringHashCodeSet(key, defaultValue, deli string) []int32 {
	set := make([]int32, 0)
	vv := stringutil.Tokenizer(conf.GetValueDef(key, defaultValue), deli)
	if vv != nil {
		for _, x := range vv {
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Continue
					}
				}()
				xx := stringutil.HashCode(strings.TrimSpace(x))
				set = append(set, int32(xx))
			}()
		}
	}
	return set
}
func (conf *Config) GetLong(key string, def int64) int64 {
	return conf.getLong(key, def)
}
func (conf *Config) getLong(key string, def int64) int64 {
	v := conf.getValue(key)
	if v == "" {
		return def
	}
	value, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return value
}
func (conf *Config) GetStringArray(key string, deli string) []string {
	return conf.getStringArray(key, deli)
}
func (conf *Config) getStringArray(key string, deli string) []string {
	v := conf.getValue(key)
	if v == "" {
		return []string{}
	}
	tokens := stringutil.Tokenizer(v, deli)
	return tokens
}

func (conf *Config) getFloat(key string, def float32) float32 {
	v := conf.getValue(key)
	if v == "" {
		return float32(def)
	}
	value, err := strconv.ParseFloat(v, 32)
	if err != nil {
		return float32(def)
	}
	return float32(value)
}

// func SetValues(keyValues *map[string]string) {
// 	path := GetConfFile()
// 	props := properties.MustLoadFile(path, properties.UTF8)
// 	for key, value := range *keyValues {
// 		if strings.TrimSpace(key) != "" {
// 			//php prefix whatap.
// 			if conf.AppType == lang.APP_TYPE_PHP {
// 				if !strings.HasPrefix(key, "whatap.") && key != "extension" {
// 					key = "whatap." + key
// 				}
// 			} else if conf.AppType == lang.APP_TYPE_BSM_PHP {
// 				if !strings.HasPrefix(key, "opsnowbsm.") && key != "extension" {
// 					key = "opsnowbsm." + key
// 				}
// 			}
// 		}

// 		props.Set(key, value)
// 	}

// 	line := ""
// 	if f, err := os.OpenFile(path, os.O_RDWR, 0644); err != nil {
// 		logutil.Println("WA215", " Error ", err)
// 		return
// 	} else {
// 		defer f.Close()

// 		r := bufio.NewReader(f)
// 		new_keys := props.Keys()
// 		old_keys := map[string]bool{}
// 		for {
// 			data, _, err := r.ReadLine()
// 			if err != nil { // new key
// 				for _, key := range new_keys {
// 					if old_keys[key] {
// 						continue
// 					}
// 					match, _ := regexp.MatchString("^\\w", key)
// 					if match {
// 						value, _ := props.Get(key)
// 						if strings.TrimSpace(value) != "" {
// 							tmp := strings.Replace(value, "\\\\", "\\", -1)
// 							tmp = strings.Replace(tmp, "\\", "\\\\", -1)
// 							line += fmt.Sprintf("%s=%s\n", key, tmp)
// 						}
// 					}
// 				}
// 				break
// 			}
// 			if strings.Index(string(data), "=") == -1 {
// 				line += fmt.Sprintf("%s\n", string(data))
// 				//io.WriteString(f, line)
// 			} else {
// 				datas := strings.Split(string(data), "=")
// 				key := strings.Trim(datas[0], " ")
// 				value := strings.Trim(datas[1], " ")
// 				old_keys[key] = true

// 				match, _ := regexp.MatchString("^\\w", key)
// 				if match {
// 					value, _ = props.Get(key)
// 				}
// 				// value 가 없는 경우 항목 추가 안함(삭제)
// 				if strings.TrimSpace(value) != "" {
// 					tmp := strings.Replace(value, "\\\\", "\\", -1)
// 					tmp = strings.Replace(tmp, "\\", "\\\\", -1)

// 					line += fmt.Sprintf("%s=%s\n", key, tmp)
// 				}
// 				//io.WriteString(f, line)
// 			}
// 		}
// 	}

// 	if f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0644); err != nil {
// 		logutil.Println("WA216", " Error ", err)
// 		return
// 	} else {
// 		defer f.Close()
// 		io.WriteString(f, line)

// 		// flush
// 		f.Sync()
// 	}
// }

// func SearchKey(keyPrefix string) *map[string]string {
// 	keyValues := map[string]string{}
// 	for _, key := range prop.Keys() {
// 		if strings.HasPrefix(key, keyPrefix) {
// 			if v, ok := prop.Get(key); ok {
// 				keyValues[key] = v
// 			}
// 		}
// 	}

// 	return &keyValues
// }

// func FilterPrefix(keyPrefix string) map[string]string {
// 	keyValues := make(map[string]string)
// 	//php prefix whatap.
// 	if conf.AppType == lang.APP_TYPE_PHP {
// 		if !strings.HasPrefix(keyPrefix, "whatap.") {
// 			keyPrefix = "whatap." + keyPrefix
// 		}
// 	} else if conf.AppType == lang.APP_TYPE_BSM_PHP {
// 		if !strings.HasPrefix(keyPrefix, "opsnowbsm.") {
// 			keyPrefix = "opsnowbsm." + keyPrefix
// 		}
// 	}
// 	pp := prop.FilterPrefix(keyPrefix)
// 	for _, key := range pp.Keys() {
// 		keyValues[key] = pp.GetString(key, "")
// 	}
// 	return keyValues
// }

// func cutOut(val, delim string) string {
// 	defer func() {
// 		if r := recover(); r != nil {
// 			logutil.Println("WA217", " Recover ", r)
// 		}
// 	}()
// 	if val == "" {
// 		return val
// 	}
// 	x := strings.LastIndex(val, delim)
// 	if x <= 0 {
// 		return ""
// 	}
// 	//return val.substring(0, x);
// 	return val[0:x]

// }

// func toHashSet(key, def string) *hmap.IntSet {
// 	set := hmap.NewIntSet()
// 	vv := strings.Split(getValueDef(key, def), ",")
// 	if vv != nil {
// 		for _, x := range vv {
// 			func() {
// 				defer func() {
// 					if r := recover(); r != nil {
// 						logutil.Infoln("WA218", " Recover ", r)
// 					}
// 				}()

// 				x = strings.TrimSpace(x)
// 				if len(x) > 0 {
// 					xx := hash.HashStr(x)
// 					set.Put(xx)
// 				}
// 			}()
// 		}
// 	}
// 	return set
// }

// func toStringSet(key, def string) *hmap.StringSet {
// 	set := hmap.NewStringSet()
// 	vv := strings.Split(getValueDef(key, def), ",")
// 	if vv != nil {
// 		for _, x := range vv {
// 			func() {
// 				defer func() {
// 					if r := recover(); r != nil {
// 						logutil.Infoln("WA219", " Recover ", r)
// 					}
// 				}()
// 				x = strings.TrimSpace(x)
// 				if len(x) > 0 {
// 					set.Put(x)
// 				}
// 			}()
// 		}
// 	}
// 	return set
// }

// func IsIgnoreTrace(hash int32, service string) bool {
// 	if conf.TraceIgnoreUrlSet.Contains(hash) {
// 		return true
// 	}
// 	if conf.IsTraceIgnoreUrlPrefix {
// 		if strings.HasPrefix(service, conf.TraceIgnoreUrlPrefix) {
// 			return true
// 		}
// 	}
// 	return false
// }

func (conf *Config) ToString() string {
	return conf.String()
}
func (conf *Config) String() string {
	sb := stringutil.NewStringBuffer()
	for k, v := range conf.m {
		sb.Append(k).Append("=").AppendLine(v)
	}
	return sb.ToString()
}
