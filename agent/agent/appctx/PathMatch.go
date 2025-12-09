package appctx

import (
	"strings"
	"sync"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/stringutil"
)

type PathMatch struct {
	urlMap   map[string]string
	confHash uint32
	mutex    sync.RWMutex
}

func NewPathMatch() *PathMatch {
	return &PathMatch{
		urlMap: make(map[string]string),
	}
}

func (p *PathMatch) Name() string {
	return "match"
}

func (p *PathMatch) Parse(hashValue uint32, url string) string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if result, exists := p.urlMap[url]; exists {
		return result
	}
	return ""
}

func (p *PathMatch) Update() {
	conf := config.GetConfig()
	val := stringutil.TrimEmpty(conf.AppContextPathSet)

	newHash := hash.HashStr(val)

	if p.confHash != uint32(newHash) {
		p.mutex.Lock()
		defer p.mutex.Unlock()

		p.confHash = uint32(newHash)
		newMap := make(map[string]string)

		if val != "" {
			items := stringutil.Tokenizer(val, ",")
			for _, item := range items {
				item = stringutil.TrimEmpty(item)
				if item == "" {
					continue
				}

				parts := p.divKeyValueTrim(item, "@")
				switch len(parts) {
				case 2:
					// name@url 형식
					name := stringutil.TrimEmpty(parts[0])
					url := stringutil.TrimEmpty(parts[1])
					if name != "" && url != "" {
						newMap[url] = name
					}
				case 1:
					// url만 있는 경우 자기 자신이 name
					url := stringutil.TrimEmpty(parts[0])
					if url != "" {
						newMap[url] = url
					}
				}
			}
		}

		p.urlMap = newMap
	}
}

// @ 구분자로 키와 값을 분리하고 양쪽 공백을 제거
func (p *PathMatch) divKeyValueTrim(s, sep string) []string {
	if s == "" || sep == "" {
		return []string{s}
	}

	pos := strings.Index(s, sep)
	if pos == -1 {
		return []string{stringutil.TrimEmpty(s)}
	}

	key := stringutil.TrimEmpty(s[:pos])
	value := stringutil.TrimEmpty(s[pos+len(sep):])

	return []string{key, value}
}
