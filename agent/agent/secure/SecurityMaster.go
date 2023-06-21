package secure

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/whatap/go-api/agent/agent/config"
	langconf "github.com/whatap/go-api/agent/lang/conf"

	// "github.com/whatap/go-api/agent/lang/license"
	"github.com/whatap/go-api/agent/util/crypto"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/go-api/agent/util/oidutil"
	"github.com/whatap/golib/io"
	"github.com/whatap/golib/util/cmdutil"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/iputil"
)

type SecurityMaster struct {
	PCODE       int64
	OID         int32
	ONAME       string
	IP          int32
	SECURE_KEY  []byte
	Cypher      *crypto.Cypher
	lastOidSent int64
	PUBLIC_IP   int32
	//lastLicense   string
	lastAccessKey string
	lastOid       int64
}

type SecuritySession struct {
	TRANSFER_KEY int32
	SECURE_KEY   []byte
	HIDE_KEY     int32
	Cypher       *crypto.Cypher
}

var master *SecurityMaster = nil
var session *SecuritySession = nil
var mutex = sync.Mutex{}

func NewSecurityMaster() *SecurityMaster {
	p := new(SecurityMaster)
	p.update()
	langconf.AddConfObserver("SecurityMaster", p)
	return p
}

func GetSecurityMaster() *SecurityMaster {
	if master != nil {
		return master
	}
	mutex.Lock()
	defer mutex.Unlock()

	if master != nil {
		return master
	}
	master = NewSecurityMaster()

	return master
}
func GetSecuritySession() *SecuritySession {
	if session != nil {
		return session
	}
	mutex.Lock()
	defer mutex.Unlock()
	if session != nil {
		return session
	}
	session = &SecuritySession{}
	return session
}
func UpdateNetCypherKey(data []byte) {
	conf := config.GetConfig()
	if conf.CypherLevel > 0 {
		data = GetSecurityMaster().Cypher.Decrypt(data)
	}
	in := io.NewDataInputX(data)
	session.TRANSFER_KEY = in.ReadInt()
	session.SECURE_KEY = in.ReadBlob()
	session.HIDE_KEY = in.ReadInt()
	session.Cypher = crypto.NewCypher(session.SECURE_KEY, session.HIDE_KEY)
	master.PUBLIC_IP = in.ReadInt()
}

func (this *SecurityMaster) Run() {
	this.update()
}

func (this *SecurityMaster) update() {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA10801", " Recover", r)
		}
	}()
	conf := config.GetConfig()
	if this.Cypher == nil || conf.AccessKey != this.lastAccessKey {
		this.lastAccessKey = conf.AccessKey
		this.resetLicense(conf.AccessKey)
	}
}

func (this *SecurityMaster) DecideAgentOnameOid(myIp string) {
	this.AutoAgentNameOrPattern(myIp)

	ip := io.ToInt(iputil.ToBytes(myIp), 0)

	oidutil.SetIp(os.Getenv("whatap.ip"))
	oidutil.SetPort(os.Getenv("whatap.port"))
	oidutil.SetHostName(os.Getenv("whatap.hostname"))
	oidutil.SetType(os.Getenv("whatap.type"))
	oidutil.SetProcess(os.Getenv("whatap.process"))
	//docker full id
	oidutil.SetDocker(os.Getenv("whatap.docker"))
	oidutil.SetIps(os.Getenv("whatap.ips"))
	oidutil.SetOidParam("cmd", os.Getenv("whatap.cmd"))
	oidutil.SetOidParam("cmd_args", os.Getenv("whatap.cmd_args"))
	oidutil.SetOidParamHexa32("cmd_full", os.Getenv("whatap.cmd_full"))
	oname := oidutil.MakeOname(os.Getenv("whatap.name"))

	this.IP = ip
	this.ONAME = oname
	this.OID = hash.HashStr(oname)
	config.GetConfig().OID = int64(this.OID)

	os.Setenv("whatap.oid", strconv.Itoa(int(this.OID)))
	os.Setenv("whatap.oname", this.ONAME)
	if this.lastOid != int64(this.OID) {
		this.lastOid = int64(this.OID)

	}
	props := map[string]string{}
	props["OID"] = fmt.Sprint(this.OID)
	config.SetValues(&props)

	//fmt.Println("PCODE=", this.PCODE, "OID=", this.OID, "ONAME=", this.ONAME)
	//logutil.Println("WA10802"," PCODE=", this.PCODE, "OID=", this.OID, "ONAME=", this.ONAME)
}

func (this *SecurityMaster) AutoAgentNameOrPattern(myIp string) {
	conf := config.GetConfig()
	os.Setenv("whatap.ip", myIp)
	os.Setenv("whatap.port", "")
	hostName, _ := os.Hostname()
	os.Setenv("whatap.hostname", hostName)
	os.Setenv("whatap.type", conf.AppName)
	os.Setenv("whatap.process", conf.AppProcessName)
	//docker full id
	os.Setenv("whatap.docker", cmdutil.GetDockerFullId())
	os.Setenv("whatap.ips", iputil.GetIPsToString())
	os.Setenv("whatap.cmd", filepath.Base(os.Args[0]))
	os.Setenv("whatap.cmd_full", strings.Join(os.Args, " "))
	if len(os.Args) > 1 {
		os.Setenv("whatap.cmd_args", strings.Join(os.Args[1:], " "))
	}
	os.Setenv("whatap.name", conf.ObjectName)
}

func (this *SecurityMaster) resetLicense(lic string) {
	conf := config.GetConfig()
	// 	pcode, security_key := license.Parse(lic)
	pcode := int64(0)
	security_key := make([]byte, 0)
	this.PCODE = pcode
	conf.PCODE = pcode
	this.SECURE_KEY = security_key
	this.Cypher = crypto.NewCypher(this.SECURE_KEY, 0)

	// session.Cypher = crypto.NewCypher(this.SECURE_KEY, 0)
	// session.TRANSFER_KEY = 0
}

//
func (this *SecurityMaster) UpdateLicense(pcode int64, security_key []byte) {
	conf := config.GetConfig()
	this.PCODE = pcode
	conf.PCODE = pcode
	this.SECURE_KEY = security_key
	this.Cypher = crypto.NewCypher(this.SECURE_KEY, 0)
}

func (this *SecurityMaster) WaitForInit() {
	for this.Cypher == nil {
		time.Sleep(1000 * time.Millisecond)
	}
}

func (this *SecurityMaster) WaitForInitFor(timeoutSec float64) {
	started := time.Now()
	for this.Cypher == nil && time.Now().Sub(started).Seconds() < timeoutSec {
		time.Sleep(1000 * time.Millisecond)
	}
}
