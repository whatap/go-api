package appctx

import (
	"strings"
	"sync"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/hmap"
	"github.com/whatap/golib/util/stringutil"
)

type PathPrefix struct {
	list      []NameAndUrl
	confHash  uint32
	noNeedSet *hmap.IntLinkedSet
	parsedSet *hmap.IntKeyLinkedMap
	mutex     sync.RWMutex
}

func NewPathPrefix() *PathPrefix {
	return &PathPrefix{
		list:      make([]NameAndUrl, 0),
		noNeedSet: hmap.NewIntLinkedSet().SetMax(10000),
		parsedSet: hmap.NewIntKeyLinkedMapDefault().SetMax(10000),
	}
}

func (p *PathPrefix) Name() string {
	return "prefix"
}

func (p *PathPrefix) Parse(hashValue uint32, url string) string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	intHash := int32(hashValue)

	if p.noNeedSet.Contains(intHash) {
		return ""
	}

	if result := p.parsedSet.Get(intHash); result != nil {
		return result.(string)
	}

	for _, item := range p.list {
		if strings.HasPrefix(url, item.URL) {
			p.parsedSet.Put(intHash, item.Name)
			return item.Name
		}
	}

	p.noNeedSet.Put(intHash)
	return ""
}

func (p *PathPrefix) Update() {
	conf := config.GetConfig()
	val := stringutil.TrimEmpty(conf.AppContextPathSet)

	newHash := hash.HashStr(val)

	if p.confHash != uint32(newHash) {
		p.mutex.Lock()
		defer p.mutex.Unlock()

		p.confHash = uint32(newHash)
		p.list = p.list[:0] // clear slice

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
						p.list = append(p.list, NameAndUrl{
							Name: name,
							URL:  url,
						})
					}
				case 1:
					// url만 있는 경우 name으로도 사용
					url := stringutil.TrimEmpty(parts[0])
					if url != "" {
						p.list = append(p.list, NameAndUrl{
							Name: url,
							URL:  url,
						})
					}
				}
			}
		}

		// 캐시 초기화
		p.noNeedSet.Clear()
		p.parsedSet.Clear()
	}
}

// @ 구분자로 키와 값을 분리하고 양쪽 공백을 제거
func (p *PathPrefix) divKeyValueTrim(s, sep string) []string {
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
