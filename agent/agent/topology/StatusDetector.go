package topology

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/secure"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/golib/lang"
	langtopology "github.com/whatap/golib/lang/topology"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/hmap"
	"github.com/whatap/golib/util/iputil"
	"github.com/whatap/golib/util/stringutil"
)

type StatusDetector struct {
}

func NewStatusDetector() *StatusDetector {
	p := new(StatusDetector)
	return p
}

func (this *StatusDetector) Process() *langtopology.NODE {

	var node *langtopology.NODE

	stat, err := this.netstat()

	if err != nil {
		logutil.Println("WAATO001", "Error Get NetStat, ", err)
		node = langtopology.NewNODE()
	} else {
		node = this.parseNetstat(stat)
	}

	conf := config.GetConfig()

	if conf.AppType == lang.APP_TYPE_PHP || conf.AppType == lang.APP_TYPE_BSM_PHP {
		node.Attr.PutString("type", "php")
	} else if conf.AppType == lang.APP_TYPE_DOTNET || conf.AppType == lang.APP_TYPE_BSM_DOTNET {
		node.Attr.PutString("type", "dotnet")
	} else if conf.AppType == lang.APP_TYPE_GO || conf.AppType == lang.APP_TYPE_BSM_GO {
		node.Attr.PutString("type", "golang")
	} else {
		node.Attr.PutString("type", "python")
	}
	node.Attr.PutString("was", conf.AppProcessName)
	node.Attr.PutLong("time", dateutil.Now())
	ip := secure.GetSecurityMaster().IP
	node.Attr.PutString("ip", iputil.ToStringInt(ip))
	node.Attr.PutLong("pid", int64(os.Getpid()))
	node.Attr.PutString("pnam", os.Args[0])

	return node
}

func (this *StatusDetector) netstat() (string, error) {

	var cmd *exec.Cmd
	if runtime.GOOS == "linux" {
		cmd = exec.Command("netstat", "-an", "-t")
	} else if runtime.GOOS == "windows" {
		return "", nil
	} else {
		cmd = exec.Command("netstat", "-an", "-p", "tcp")
	}

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func (this *StatusDetector) parseNetstat(netstat string) *langtopology.NODE {

	node := langtopology.NewNODE()

	localIPs := LocalIPs()

	r := bufio.NewScanner(strings.NewReader(netstat))
	for r.Scan() {
		line := r.Text()
		if strings.HasPrefix(line, "tcp") {
			this.parse(node, line, localIPs)
		}
	}

	return node
}

/**
 * <pre>
 * <LINUX>
 * tcp        0      0 10.10.3.75:39504        210.122.36.175:10051    TIME_WAIT
 * tcp        0      0 10.10.3.75:22           10.10.201.10:51196      ESTABLISHED
 * tcp        0      0 10.10.3.75:39510        210.122.36.175:10051    TIME_WAIT
 * tcp6       0      0 :::111                  :::*                    LISTEN
 * tcp6       0      0 :::6610                 :::*                    LISTEN
 * tcp6       0      0 :::37010                :::*                    LISTEN
 * tcp6       0      0 :::22                   :::*                    LISTEN
 * tcp6       0      0 :::6620                 :::*                    LISTEN
 * tcp6       0      0 :::7710                 :::*                    LISTEN
 * tcp6       0      0 :::34496                :::*                    LISTEN
 * tcp6       0      0 127.0.0.1:34548         127.0.0.1:34496         TIME_WAIT
 * tcp6       0      0 127.0.0.1:34544         127.0.0.1:34496         TIME_WAIT
 * tcp6       0      0 10.10.3.75:6610         10.10.3.157:41844       ESTABLISHED
 *
 * <MAC>
 * tcp4       0      0  192.168.219.186.57596  64.233.188.125.5222    ESTABLISHED
 * tcp4       0      0  192.168.219.186.57551  64.233.188.125.5222    ESTABLISHED
 * tcp4       0      0  192.168.25.56.57473    64.233.187.125.5222    ESTABLISHED
 * tcp4       0      0  192.168.0.4.54343      108.177.97.125.5222    ESTABLISHED
 * tcp4       0      0  127.0.0.1.17603        *.*                    LISTEN
 * tcp4       0      0  127.0.0.1.17600        *.*                    LISTEN
 * tcp4       0      0  *.17500                *.*                    LISTEN
 * tcp6       0      0  *.17500                *.*                    LISTEN
 * </pre>
 */
func (this *StatusDetector) parse(n *langtopology.NODE, line string, localIPs *hmap.StringSet) {
	tokens := stringutil.Tokenizer(line, "\t\n ")
	if !strings.HasPrefix(line, "tcp") {
		return
	}
	if tokens[5] == "LISTEN" {
		n.AddListen(localIPs, tokens[3])
	} else {
		n.AddOutter(tokens[3], tokens[4])
	}
}

func localAddresses() {
	ifaces, err := net.Interfaces()
	if err != nil {
		fmt.Print(fmt.Errorf("localAddresses: %+v\n", err.Error()))
		return
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			fmt.Print(fmt.Errorf("localAddresses: %+v\n", err.Error()))
			continue
		}
		for _, a := range addrs {
			switch v := a.(type) {
			case *net.IPAddr:
				fmt.Printf("%v : %s (%s)\n", i.Name, v, v.IP.DefaultMask())
			}

		}
	}
}

func LocalIPs() *hmap.StringSet {

	ipSet := hmap.NewStringSet()

	ifaces, err := net.Interfaces()
	if err != nil {
		logutil.PrintlnError("WAATO002", "Error Get net interfaces, ", err)
		return ipSet
	}

	for _, i := range ifaces {
		// 상태가 up 인경우만 검
		if i.Flags&net.FlagUp != 1 {
			continue
		}

		addrs, err := i.Addrs()
		if err != nil {
			logutil.PrintlnError("WAATO003", "Error Get addrs of ifaces, ", err)
			continue
		}

		for _, a := range addrs {
			switch v := a.(type) {
			case *net.IPAddr:
				//fmt.Printf("%v : %s (%s) %t %d [%v,%v]\n", i.Name, v, v.IP.DefaultMask(), v.IP.IsLoopback(), len(v.IP), v.IP.To4(), v.IP.To16)
				if !v.IP.IsLoopback() && v.IP.To4() != nil {
					ipSet.Put(v.IP.To4().String())
				}

			case *net.IPNet:
				//fmt.Printf("%v : %s [%v/%v] %t %d [%v,%v]\n", i.Name, v, v.IP, v.Mask, v.IP.IsLoopback(), len(v.IP), v.IP.To4(), v.IP.To16())
				if !v.IP.IsLoopback() && v.IP.To4() != nil {
					ipSet.Put(v.IP.To4().String())
				}
			}
		}
	}

	return ipSet
}

func StatusDetectorMain() {
	NewStatusDetector().Process()
}
