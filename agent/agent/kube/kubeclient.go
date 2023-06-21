package kube

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/secure"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/golib/io"
	"github.com/whatap/golib/lang/value"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/iputil"
	"github.com/whatap/golib/util/stringutil"
)

var NodeAgentHost string
var NodeAgentPort uint16

func StartClient() {

	loadContainerId()
	conf := config.GetConfig()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logutil.Println("KubeClient", "Recover ", r)
			}
		}()
		for {
			process(conf.PodName)
			time.Sleep(3 * time.Second)
		}

	}()

}

func process(podname string) {
	// fmt.Println("process step -1 podname:", podname)
	secu := secure.GetSecurityMaster()
	p := value.NewMapValue()
	p.PutString("cmd", "regist")
	p.PutLong("pcode", secu.PCODE)
	p.PutLong("oid", int64(secu.OID))
	p.PutString("oname", secu.ONAME)
	p.PutString("ip", iputil.ToStringInt(secu.IP))
	hostname, _ := os.Hostname()
	p.PutString("hostname", hostname)
	p.Put("kube.micro", value.NewBoolValue(true))
	p.PutString("pod_name", podname)

	conf := config.GetConfig()
	if conf.OKIND != 0 {
		p.PutLong("okind", int64(conf.OKIND))
		p.PutString("okind_name", conf.OKIND_NAME)
	}
	if conf.ONODE != 0 {
		p.PutLong("onode", int64(conf.ONODE))
		p.PutString("onode_name", conf.ONODE_NAME)
	}
	if len(containerId) > 0 {
		p.PutString("container_id", containerId)
	}
	// fmt.Println("process step -2 podname:", conf.OKind, conf.ONode, containerId)
	sendTo("Master", conf.MasterAgentHost, conf.MasterAgentPort, p,
		func(m *value.MapValue) {
			// fmt.Println("process step -2.1 ret:", m.ToString())
			host := m.GetString("node.agent.ip")
			port := uint16(m.GetLong("node.agent.port"))
			if len(host) > 0 {
				NodeAgentHost = host
			}
			if port != 0 {
				NodeAgentPort = port
			}
		})
	sendTo("WorkNode", NodeAgentHost, NodeAgentPort, p,
		func(m *value.MapValue) {
			Cpu = m.GetFloat("cpu")
			CpuSys = m.GetFloat("cpu_sys")
			CpuUser = m.GetFloat("cpu_user")
			ThrottledPeriods = m.GetFloat("throttled_periods")
			ThrottledTime = m.GetFloat("throttled_time")

			Memory = m.GetLong("memory")
			Failcnt = m.GetLong("failcnt")
			Limit = m.GetLong("limit")
			MaxUsage = m.GetLong("maxUsage")

			NodeRecvTime = time.Now().UnixNano() * 1000000
			Metering = m.GetFloat("metering")
		})
}

type MapValueH1 func(m *value.MapValue)
type SessionMap map[string]net.Conn

var prefixSessionMap SessionMap = SessionMap{}

const READ_MAX int32 = int32(8 * 1024 * 1024)

func sendTo(prefix string, host string, port uint16, p *value.MapValue, h MapValueH1) (func_err error) {
	// fmt.Println("sendTo step -0.1 ", prefix, host, port)
	if len(host) < 1 {
		func_err = fmt.Errorf("kubeclient %s failed no host address", prefix)
		return
	}
	// fmt.Println("sendTo step -0.2 ", prefix, host, port)
	if _, ok := prefixSessionMap[prefix]; !ok {
		d := net.Dialer{Timeout: 3 * time.Second, KeepAlive: 30 * time.Second}
		conn, err := d.Dial("tcp", fmt.Sprint(host, ":", port))
		if err != nil {
			// fmt.Println("sendTo step -0.3 ", err)
			func_err = err
			return
		}
		// fmt.Println("sendTo step -0.4 ")
		prefixSessionMap[prefix] = conn
	}
	// fmt.Println("sendTo step -0.5 ")
	dout := io.NewDataOutputX()
	dout.WriteUShort(0xCAFE)

	doutx := io.NewDataOutputX()
	value.WriteMapValue(doutx, p)

	b := doutx.ToByteArray()
	dout.WriteIntBytes(b)
	conn := prefixSessionMap[prefix]
	conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	b = dout.ToByteArray()
	nbytesleft := len(b)
	nbytesuntilnow := 0
	// fmt.Println("sendTo step -1 ", prefix, host, port)
	for nbytesleft > 0 {
		nbytesthistime, err := conn.Write(b[nbytesuntilnow:])
		if err != nil {
			conn.Close()
			delete(prefixSessionMap, prefix)

			func_err = err
			return
		}
		nbytesleft -= nbytesthistime
		nbytesuntilnow += nbytesthistime
	}
	// fmt.Println("sendTo step -2 ", prefix, host, port)
	din := io.NewDataInputNet(conn)
	headercode := din.ReadUShort()
	// fmt.Println("sendTo step -2.1 ", headercode)

	if headercode == 0xCAFE {
		// fmt.Println("sendTo step -3 ", prefix, host, port)
		b = din.ReadIntBytesLimit(int(READ_MAX))
		if len(b) > 0 {
			// fmt.Println("sendTo step -4 ", len(b))
			dinx := io.NewDataInputX(b)
			m := value.ReadMapValue(dinx)
			// fmt.Println("sendTo step -4.0.1 ", m)
			if m != nil {
				// fmt.Println("sendTo step -4.1", m.ToString())
				h(m)
			}

		}
	}
	// fmt.Println("sendTo step -5 ")

	return
}

var (
	NodeRecvTime     int64
	Cpu              float32
	CpuSys           float32
	CpuUser          float32
	ThrottledPeriods float32
	ThrottledTime    float32
	Metering         float32

	Memory   int64
	Failcnt  int64
	Limit    int64
	MaxUsage int64

	containerKey int32
	containerId  string
)

func loadContainerId() (err error) {
	if containerKey != 0 {
		return
	}

	filepath := "/proc/self/cgroup"
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	pattern, _ := regexp.Compile(`(\.scope)+$`)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\n")
		if len(line) > 0 {
			line = stringutil.CutLastString(line, "/")
			line = stringutil.CutLastString(line, "-")
			containerId = pattern.ReplaceAllString(line, "")
			if len(containerId) > 5 { // 컨테이너 아이디는 최소 5자 이상이어야 한다.
				containerKey = hash.HashStr(containerId)
				return
			}
		}
	}

	containerKey = 0
	containerId = ""

	return
}

func GetContainerInfo(h2 func(int32, string)) {
	if containerKey != 0 {
		h2(containerKey, containerId)
	}
}
