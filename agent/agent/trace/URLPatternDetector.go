package trace

import (
	//"log"
	//"runtime/debug"
	_ "fmt"
	"strings"
	"sync"

	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/hmap"
	"github.com/whatap/golib/util/pathutil"
	"github.com/whatap/golib/util/stringutil"
	"github.com/whatap/go-api/agent/agent/config"
	langconf "github.com/whatap/go-api/agent/lang/conf"
	"github.com/whatap/go-api/agent/util/logutil"
)

var urlPatternDetector *URLPatternDetector = nil
var urlPatternDetectorMutex = sync.Mutex{}

type URLPatternDetector struct {
	oldHash      int32
	patterns     *hmap.IntKeyLinkedMap
	restUrlTable *pathutil.PathTree
	noNeedSet    *hmap.IntLinkedSet
	parsedSet    *hmap.IntKeyLinkedMap
	conf         *config.Config

	lock sync.Mutex
}

// TODO Observer 구현 ?, config reload 에 맞춰서 reload 필요(import cycle 에러 발생 조심 , interface 구현)
//this.addUrlList(conf.Trace_httpc_normalize_urls)

//	static {
//		try {
//			addUrlList(conf.trace_httpc_normalize_urls);
//			ConfObserver.add("URLPatternDetector", new Runnable() {
//				public void run() {
//					addUrlList(conf.trace_httpc_normalize_urls);
//				}
//			});
//		} catch (Throwable t) {
//		}
//	}
//
// Java Static 을 Singlton으로 정리
func GetInstanceURLPatternDetector() *URLPatternDetector {
	urlPatternDetectorMutex.Lock()
	defer urlPatternDetectorMutex.Unlock()
	if urlPatternDetector != nil {
		//fmt.Println("URLPatternDetector return")
		return urlPatternDetector
	}

	urlPatternDetector = new(URLPatternDetector)
	//fmt.Println("URLPatternDetector New return")
	urlPatternDetector.patterns = hmap.NewIntKeyLinkedMapDefault().SetMax(1007)
	urlPatternDetector.restUrlTable = pathutil.NewPathTree()
	urlPatternDetector.noNeedSet = hmap.NewIntLinkedSet().SetMax(5000)
	urlPatternDetector.parsedSet = hmap.NewIntKeyLinkedMap(1007, float32(1)).SetMax(1000)
	urlPatternDetector.conf = config.GetConfig()

	urlPatternDetector.lock = sync.Mutex{}

	// 초기화
	urlPatternDetector.addUrlList(urlPatternDetector.conf.TraceHttpcNormalizeUrls)

	// config observer 등록
	langconf.AddConfObserver("URLPatternDetector", urlPatternDetector)

	return urlPatternDetector
}

// lang.conf.Runnable Interface , 초기화
func (this *URLPatternDetector) Run() {
	//logutil.Println("Run")
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA10601", " Recover", r) //, string(debug.Stack()))
		}
	}()
	this.addUrlList(this.conf.TraceHttpcNormalizeUrls)
}

// func (this *URLPatternDetector) addUrlList(urls string) {
// urls  url concat comma(,)
func (this *URLPatternDetector) addUrlList(urls string) {
	if urls == "" {
		urls = ""
	}
	newHash := hash.HashStr(urls)
	if this.oldHash == newHash {
		return
	}
	this.oldHash = newHash

	//urlArr := stringutil.Tokenizer(urls, ",")
	urlArr := stringutil.Split(urls, ",")

	if urlArr == nil || len(urlArr) == 0 {
		if this.restUrlTable.Size() > 0 {
			this.restUrlTable = pathutil.NewPathTree()
		}
	} else {
		this.Build(urlArr)
	}
}

func (this *URLPatternDetector) Add(hash int32, path string) {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.patterns.ContainsKey(hash) {
		return
	}
	this.patterns.Put(hash, path)

	nodes := stringutil.Split(path, "/")

	for k := 0; k < len(nodes); k++ {
		if strings.HasPrefix(nodes[k], "{") {
			nodes[k] = "*"
		}
	}
	if len(nodes) == 0 {
		return
	}
	if len(nodes) == 1 && "*" == nodes[0] {
		return
	}

	this.restUrlTable.InsertArray(nodes, path)
}

func (this *URLPatternDetector) Build(urlList []string) {
	tmp := pathutil.NewPathTree()
	for _, path := range urlList {
		nodes := stringutil.Split(path, "/")
		for k := 0; k < len(nodes); k++ {
			if strings.HasPrefix(nodes[k], "{") {
				nodes[k] = "*"
			}
		}
		if len(nodes) == 1 && "*" == nodes[0] {
			continue
		}
		tmp.InsertArray(nodes, path)
	}
	en := this.patterns.Values()

	for en.HasMoreElements() {
		path := en.NextElement().(string)
		nodes := stringutil.Split(path, "/")
		for k := 0; k < len(nodes); k++ {
			if strings.HasPrefix(nodes[k], "{") {
				nodes[k] = "*"
			}
		}
		if len(nodes) == 1 && "*" == nodes[0] {
			continue
		}
		tmp.InsertArray(nodes, path)
	}
	this.restUrlTable = tmp
	this.noNeedSet.Clear()
	this.parsedSet.Clear()
}

func (this *URLPatternDetector) Normalize(url string) string {
	if this.conf.TraceHttpcNormalizeEnabled == false || url == "" {
		return url
	}

	if url == "" {
		return url
	}

	if this.noNeedSet.Contains(int32(stringutil.HashCode(url))) {
		//fmt.Println("URLPatternDetector.Normalize noNeedSet Contains")
		return url
	}
	var newUrl interface{}

	newUrl = this.parsedSet.Get(int32(stringutil.HashCode(url)))
	if newUrl != nil {
		//fmt.Println("URLPatternDetector.Normalize parsedSet.Get")
		return newUrl.(string)
	}

	newUrl = this.restUrlTable.Find(url)
	if newUrl == nil {
		//fmt.Println("URLPatternDetector.Normalize restUrlTable Not find, NoNeed ")
		this.noNeedSet.Put(int32(stringutil.HashCode(url)))
		return url
	}

	this.parsedSet.Put(int32(stringutil.HashCode(url)), newUrl.(string))
	//fmt.Println("URLPatternDetector.Normalize parsedSet Put")

	return newUrl.(string)
}

func MainURLPatternDetector() {
	pp := GetInstanceURLPatternDetector()
	u := "/a/{x}/b"
	pp.Add(hash.HashStr(u), u)
	u1 := "/b/a/{z}"
	pp.Add(hash.HashStr(u1), u1)
	pp.Add(hash.HashStr(u), u)

	//		add(HashUtil.hash(u), u);
	//		add(HashUtil.hash(u), u);
	//		add(HashUtil.hash(u), u);
	e := pp.restUrlTable.Values()
	for e.HasMoreElements() {
		//fmt.Println("URLPatternDetector.Main=" , e.NextElement(pp.restUrlTable.Top))
	}

	//fmt.Println("Tokenizer=", stringutil.Tokenizer(u, "/"))
	//fmt.Println("Split=", stringutil.Split(u, "/"))

	//fmt.Println("Tokenizer=", stringutil.Tokenizer("/a/2345/b", "/"))
	//fmt.Println("Split=", stringutil.Split("/a/2345/b", "/"))

	//fmt.Println("/a/2345/b => ",pp.Normalize("/a/2345/b"))
	//fmt.Println("/a/b/23425 => ",pp.Normalize("/a/b/23425"))
	//fmt.Println("/b/a/23425 => ",pp.Normalize("/b/a/23425"))

	//aa := GetInstanceURLPatternDetector()
	//fmt.Println("/a/2345/b => ",aa.Normalize("/a/2345/b"))
	//fmt.Println("/a/b/23425 => ",aa.Normalize("/a/b/23425"))
	//fmt.Println("/b/a/23425 => ",aa.Normalize("/b/a/23425"))

}
