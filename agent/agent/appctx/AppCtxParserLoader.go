package appctx

import (
	"os"
	"plugin"
	"sync"

	"github.com/whatap/go-api/agent/util/logutil"
)

type AppCtxParserLoader struct {
	jarPath     string
	appCtxClass string
	jarFileTime int64
	ctxParser   IAppCtx
	mutex       sync.Mutex
}

var parserLoader *AppCtxParserLoader
var loaderOnce sync.Once

func GetAppCtxParserLoader() *AppCtxParserLoader {
	loaderOnce.Do(func() {
		parserLoader = &AppCtxParserLoader{
			jarFileTime: -1,
		}
	})
	return parserLoader
}

func (l *AppCtxParserLoader) Load(pluginPath, className string) IAppCtx {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if pluginPath == "" {
		l.jarPath = ""
		l.jarFileTime = -1
		l.ctxParser = nil
		return nil
	}

	// 파일 존재 확인
	fileInfo, err := os.Stat(pluginPath)
	if err != nil {
		l.jarPath = ""
		l.jarFileTime = -1
		l.ctxParser = nil
		return nil
	}

	// 캐시 확인
	if l.ctxParser != nil &&
		className == l.appCtxClass &&
		pluginPath == l.jarPath &&
		fileInfo.ModTime().Unix() == l.jarFileTime {
		return l.ctxParser
	}

	// 파일 정보 업데이트
	l.jarFileTime = fileInfo.ModTime().Unix()
	l.jarPath = pluginPath
	l.appCtxClass = className

	// Go 플러그인 로딩
	p, err := plugin.Open(pluginPath)
	if err != nil {
		logutil.Println("WA221-02", "IAppCtx load fail:", className, err.Error())
		return &PathDefault{}
	}

	// 심볼 찾기
	symbol, err := p.Lookup(className)
	if err != nil {
		logutil.Println("WA221-03", "IAppCtx load fail:", className, err.Error())
		return &PathDefault{}
	}

	if parserFunc, ok := symbol.(func() IAppCtx); ok {
		l.ctxParser = parserFunc()
		logutil.Println("WA221-04", "IAppCtx load:", className)
	} else {
		logutil.Println("WA221-05", "IAppCtx load fail: invalid symbol type", className)
		return &PathDefault{}
	}

	return l.ctxParser
}
