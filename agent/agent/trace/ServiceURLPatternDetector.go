package trace

import (
	//"log"
	//"runtime/debug"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/hmap"
	"github.com/whatap/golib/util/pathutil"
	"github.com/whatap/golib/util/stringutil"
	"github.com/whatap/go-api/agent/agent/config"
	langconf "github.com/whatap/go-api/agent/lang/conf"
	"github.com/whatap/go-api/agent/util/logutil"
)

// agnet.boot -> agent.trace로 이동 import cycle error

var serviceUrlPatternDetector *ServiceURLPatternDetector = nil
var serviceUrlPatternDetectorMutex = sync.Mutex{}

type ServiceURLPatternDetector struct {
	conf                      *config.Config
	last_conf_value           string
	traceAutoNormalizeEnabled bool
	restUrlTable              *pathutil.PathTree
	noNeedSet                 *hmap.IntLinkedSet
	parsedSet                 *hmap.IntKeyLinkedMap
	pathMap                   *SearchPathMap
	last_build                int64

	lock sync.Mutex
}

func GetInstanceServiceURLPatternDetector() *ServiceURLPatternDetector {
	serviceUrlPatternDetectorMutex.Lock()
	defer serviceUrlPatternDetectorMutex.Unlock()
	if serviceUrlPatternDetector != nil {
		//fmt.Println("URLPatternDetector return")
		return serviceUrlPatternDetector
	} else {
		p := new(ServiceURLPatternDetector)

		p.conf = config.GetConfig()
		p.last_conf_value = stringutil.TrimEmpty(p.conf.TraceNormalizeUrls)
		p.traceAutoNormalizeEnabled = p.conf.TraceAutoNormalizeEnabled
		p.restUrlTable = pathutil.NewPathTree()
		p.noNeedSet = hmap.NewIntLinkedSet().SetMax(10000)
		p.parsedSet = hmap.NewIntKeyLinkedMapDefault().SetMax(2000)
		p.pathMap = NewSearchPathMap()
		p.last_build = 0
		p.lock = sync.Mutex{}

		//부팅하자마자 등록된 옵션을 가지고 파서를 초기화한다.
		p.last_conf_value = stringutil.TrimEmpty(p.conf.TraceNormalizeUrls)
		p.traceAutoNormalizeEnabled = p.conf.TraceAutoNormalizeEnabled
		p.Build(false)

		// config observer 등록
		langconf.AddConfObserver("ServiceURLPatternDetector", p)
		return p
	}
}

// lang.conf.Runnable Interface
func (this *ServiceURLPatternDetector) Run() {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA10501", " Recover", r) //, string(debug.Stack()))
		}
	}()
	value := stringutil.TrimEmpty(this.conf.TraceNormalizeUrls)
	if this.last_conf_value != value || this.traceAutoNormalizeEnabled != this.conf.TraceAutoNormalizeEnabled {
		this.last_conf_value = stringutil.TrimEmpty(this.conf.TraceNormalizeUrls)
		this.traceAutoNormalizeEnabled = this.conf.TraceAutoNormalizeEnabled
		this.Build(false)
	}
}

// TODO 기존 java 에서 Observer, Thread로 config 정보를 갱신하는 부분
func (this *ServiceURLPatternDetector) Start(stime int64) {

	//
	//		ConfObserver.add("ServiceURLPatternDetector", new Runnable() {
	//			public void run() {
	//				String value = StringUtil.trimEmpty(conf.trace_normalize_urls);
	//				if (last_conf_value.equals(value) == false || trace_auto_normalize_enabled != conf.trace_auto_normalize_enabled) {
	//					last_conf_value = StringUtil.trimEmpty(conf.trace_normalize_urls);
	//					trace_auto_normalize_enabled = conf.trace_auto_normalize_enabled;
	//					build(false);
	//				}
	//			}
	//		});
	//		//부팅하자마자 등록된 옵션을 가지고 파서를 초기화한다.
	//		last_conf_value = StringUtil.trimEmpty(conf.trace_normalize_urls);
	//		trace_auto_normalize_enabled = conf.trace_auto_normalize_enabled;
	//		build(false);
	//
	//		Thread t = new Thread("ServiceURLPatternDetector") {
	//			@Override
	//			public void run() {
	//				try {
	//					waitForServerInit(stime);
	//					// 일정 시간이 흐른뒤 로딩된 클래스를 뒤져서 다시 파서를 초기화한다.
	//					// 이 과정은 부팅후 한번만 한다.
	//					if(conf.trace_auto_normalize_enabled){
	//						search();
	//					}
	//					last_conf_value = StringUtil.trimEmpty(conf.trace_normalize_urls);
	//					trace_auto_normalize_enabled = conf.trace_auto_normalize_enabled;
	//					build(false);
	//				} catch (Throwable t) {
	//					t.printStackTrace();
	//				}
	//			}
	//		};
	//		t.setDaemon(true);
	//		t.start();
}

// TODO Java Annotaion 검색은 추후
func (this *ServiceURLPatternDetector) Search() {
	//	Class[] classes = JavaAgent.instrumentation.getAllLoadedClasses();
	//	for (int i = 0; i < classes.length; i++) {
	//		if (classes[i].getClassLoader() == null)
	//			continue;
	//		ComponentsVersions.search(classes[i]);
	//		pathMap.scan(classes[i]);
	//	}
	//	ComponentsVersions.send();
}

func (this *ServiceURLPatternDetector) Build(force bool) {
	this.lock.Lock()
	defer func() {
		this.lock.Unlock()
		if r := recover(); r != nil {
			logutil.Println("WA10400", " Recover", r)
		}
	}()

	// 빈번하게 호출되는 것을 막음
	now := dateutil.Now()
	if force == false && now < this.last_build+10000 {
		return
	}

	this.last_build = now
	this.last_conf_value = stringutil.TrimEmpty(this.conf.TraceNormalizeUrls)

	// 임시 저장용 콜렉션들..
	pathTreeTmp := pathutil.NewPathTree()
	noNeedTmp := hmap.NewIntLinkedSet().SetMax(10000)

	// 어노테시션에서 수집된 것을 먼저 빌드함 ..
	// TODO Annotation 수집은 일단 주석 (PHP, Phtyon 에서 가능하면 그 때 다시 변환 )
	//			if conf.TraceAutoNormalizeEnabled {
	//				StringEnumer en = pathMap.pathParamUrlSet.elements();
	//				while (en.hasMoreElements()) {
	//					addPath(pathTreeTmp, en.nextString());
	//				}
	//				noNeedTmp.putAll(pathMap.noPathParamUrlHashSet);
	//			}

	// 설정에 등록된 URL들을
	urls := stringutil.Tokenizer(this.conf.TraceNormalizeUrls, ",")
	if urls != nil {
		for _, u := range urls {
			if u != "" {
				if strings.Index(u, "{") >= 0 {
					this.addPath(pathTreeTmp, u)
				} else {
					noNeedTmp.Put(int32(stringutil.HashCode(u)))
				}
			}
		}
	}

	this.restUrlTable = pathTreeTmp
	this.noNeedSet = noNeedTmp
	this.parsedSet.Clear()
}

func (this *ServiceURLPatternDetector) addPath(pathTree *pathutil.PathTree, path string) {
	path = stringutil.TrimEmpty(path)

	if len(path) < 3 {
		return
	}

	nodes := stringutil.Split(path, "/")
	for k := 0; k < len(nodes); k++ {
		if strings.HasPrefix(nodes[k], "{") {
			nodes[k] = "*"
		}
	}

	if len(nodes) == 0 {
		return
	}

	if len(nodes) == 1 && nodes[0] == "*" {
		return
	}

	//logutil.Println("InsertArray=", nodes, ",path=", path)

	pathTree.InsertArray(nodes, path)
}

func (this *ServiceURLPatternDetector) waitForServerInit(stime int64) {
	now := dateutil.Now()
	for now < stime {
		time.Sleep(1000 * time.Millisecond)
		now = dateutil.Now()
	}
}

func (this *ServiceURLPatternDetector) Normalize(url string) string {
	if this.conf.TraceNormalizeEnabled == false || this.noNeedSet.Contains(int32(stringutil.HashCode(url))) {
		return url
	}
	newUrl := this.parsedSet.Get(int32(stringutil.HashCode(url)))

	if newUrl != nil {
		return newUrl.(string)
	}
	newUrl = this.restUrlTable.Find(url)

	if newUrl == nil {
		this.noNeedSet.Put(int32(stringutil.HashCode(url)))
		return url
	}

	this.parsedSet.Put(int32(stringutil.HashCode(url)), newUrl.(string))

	//logutil.Println("Nomalize=", url, ",newUrl=", newUrl.(string))

	return newUrl.(string)
}

func ServiceUTLPatternDetectorMain() {
	p := GetInstanceServiceURLPatternDetector()

	p.conf.TraceNormalizeEnabled = true
	p.conf.TraceNormalizeUrls = "/get/{id},/set/{value},/get2/{table}, /{cate}/page/{page}/neSrl/{index}"

	p.Build(true)
	//	fmt.Println("Normalize = ", p.Normalize("/get/111"))
	//	fmt.Println("Normalize = ", p.Normalize("/set/34234"))
	//	fmt.Println("Normalize = ", p.Normalize("/get2/34234"))

	t := pathutil.NewPathTree()

	t.Insert("/get/*", "/get/{id}")
	t.Insert("/set/*", "/set/{value}")
	t.Insert("/get2/*", "/get2/{table}")

	//	fmt.Println("Find = ", (t.Find("/get/111")))
	//	fmt.Println("Find = ", (t.Find("/set/34234")))
	//	fmt.Println("Find = ", (t.Find("/get2/34234")))

	fmt.Println(p.Normalize("/306803/page/1/neSrl/4553439"))

	t.Insert("/get2/*", "/get2/{table}")

}
