package urlutil

import (
	//	"log"
	_ "fmt"
	"net/url"
	"strconv"
	"strings"
)

// java URL 처럼
type URL struct {
	Url      string
	Protocol string
	Host     string
	RawPath  string
	Path     string
	RawPort  string
	Port     int
	RawQuery string
	Query    string
	File     string
}

func NewURL(url string) *URL {
	p := new(URL)
	p.Url = url

	p.process()

	return p
}

func (this *URL) process() {
	var tmp string
	var pos int
	var err error

	// Protocol
	tmp = this.Url
	pos = strings.Index(tmp, "://")
	if pos > -1 {
		this.Protocol = tmp[0:pos]
		tmp = tmp[pos+3:]
	}

	// Host, Port
	pos = strings.Index(tmp, "/")
	if pos > -1 {
		this.Host = tmp[0:pos]
		tmp = tmp[pos:]

		// Port 분리
		pos = strings.Index(this.Host, ":")
		if pos > -1 {
			this.RawPort = this.Host[pos+1:]
			this.Port, err = strconv.Atoi(this.Host[pos+1:])
			if err != nil {
				//fmt.Println("WA871", "Port ParseInt Error:", err)
			}
			this.Host = this.Host[0:pos]
		} else {
			this.Port = 80
		}

	} else {
		this.Host = tmp
		if this.Protocol == "https" {
			this.Port = 443
		} else {
			this.Port = 80
		}
	}

	// Path, File, Query
	pos = strings.Index(tmp, "?")
	if pos > -1 {
		this.File = tmp
		this.Path = tmp[0:pos]
		this.Query = tmp[pos+1:]
	} else {
		this.Path = tmp
		this.File = tmp
		this.Query = ""
	}

	// Path, Query URLDecode추가 (net/url)
	this.RawPath = this.Path
	this.Path, err = url.PathUnescape(this.RawPath)
	if err != nil {
		this.Path = this.RawPath
	}
	this.RawQuery = this.Query
	this.Query, err = url.QueryUnescape(this.RawQuery)
	if err != nil {
		this.Query = this.RawQuery
	}

}

// URL Decode 된  url 정보를 다시 조합해서 출력
func (this *URL) String() string {
	rt := ""
	if this.Protocol != "" {
		rt = rt + this.Protocol + "://"
	}
	rt = rt + this.Host

	if this.RawPort != "" {
		rt = rt + ":" + this.RawPort
	}

	rt = rt + this.Path

	if this.Query != "" {
		rt = rt + "?" + this.Query
	}
	return rt
}

func (this *URL) Domain() string {
	rt := ""
	if this.Protocol != "" {
		rt = rt + this.Protocol + "://"
	}
	rt = rt + this.Host
	return rt
}

func (this *URL) DomainPath() string {
	rt := ""
	if this.Protocol != "" {
		rt = rt + this.Protocol + "://"
	}
	rt = rt + this.Host

	if this.RawPort != "" {
		rt = rt + ":" + this.RawPort
	}

	rt = rt + this.Path
	return rt
}

func MainURL() {
	//p := NewURL("http://www.naver.com")
	////fmt.Println("url=", p.Url , "\r\n", p.Protocol, ", ", p.Host, ", ", p.Port , ", ", p.Path , ", ", p.File, ", ", p.Query)
	//
	//p = NewURL("https://www.naver.com/")
	////fmt.Println("url=", p.Url , "\r\n", p.Protocol, ", ", p.Host, ", ", p.Port , ", ", p.Path , ", ", p.File, ", ", p.Query)
	//
	//p = NewURL("https://www.naver.com/a/b/c/")
	////fmt.Println("url=", p.Url , "\r\n", p.Protocol, ", ", p.Host, ", ", p.Port , ", ", p.Path , ", ", p.File, ", ", p.Query)
	//
	//p = NewURL("http://www.naver.com/a/b/c/index.php")
	////fmt.Println("url=", p.Url , "\r\n", p.Protocol, ", ", p.Host, ", ", p.Port , ", ", p.Path , ", ", p.File, ", ", p.Query)
	//
	//p = NewURL("http://www.naver.com/a/b/c/d/index.php?aal=3&bbb=3")
	////fmt.Println("url=", p.Url , "\r\n", p.Protocol, ", ", p.Host, ", ", p.Port , ", ", p.Path , ", ", p.File, ", ", p.Query)
	//
	//p = NewURL("https://www.naver.com:80")
	////fmt.Println("url=", p.Url , "\r\n", p.Protocol, ", ", p.Host, ", ", p.Port , ", ", p.Path , ", ", p.File, ", ", p.Query)
	//
	//p = NewURL("https://www.naver.com:80/")
	////fmt.Println("url=", p.Url , "\r\n", p.Protocol, ", ", p.Host, ", ", p.Port , ", ", p.Path , ", ", p.File, ", ", p.Query)
	//
	//p = NewURL("https://www.naver.com:80/a/b/c/")
	////fmt.Println("url=", p.Url , "\r\n", p.Protocol, ", ", p.Host, ", ", p.Port , ", ", p.Path , ", ", p.File, ", ", p.Query)
	//
	//p = NewURL("http://www.naver.com:80/a/b/c/index.php")
	////fmt.Println("url=", p.Url , "\r\n", p.Protocol, ", ", p.Host, ", ", p.Port , ", ", p.Path , ", ", p.File, ", ", p.Query)
	//
	//p = NewURL("http://www.naver.com:80/a/b/c/d/index.php?aal=3&bbb=3")
	////fmt.Println("url=", p.Url , "\r\n", p.Protocol, ", ", p.Host, ", ", p.Port , ", ", p.Path , ", ", p.File, ", ", p.Query)
}
