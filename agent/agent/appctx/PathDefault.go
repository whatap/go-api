package appctx

import (
	"strings"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/golib/util/stringutil"
)

type PathDefault struct{}

func (p *PathDefault) Name() string {
	return "default"
}

func (p *PathDefault) Parse(hash uint32, url string) string {
	if url == "" {
		return ""
	}

	conf := config.GetConfig()

	switch conf.AppContextPathDepth {
	case 2:
		// depth=2: /path1/path2 형태에서 /path1/path2 추출
		if idx1 := strings.Index(url[1:], "/"); idx1 > 0 {
			idx1++ // 첫 번째 '/' 이후 위치
			if idx2 := strings.Index(url[idx1+1:], "/"); idx2 > 0 {
				url = url[:idx1+1+idx2]
			}
		}
		// 마지막 슬래시 제거 (있다면)
		if len(url) > 1 && url[len(url)-1] == '/' {
			url = url[:len(url)-1]
		}

		if conf.AppContextPathSet == "" {
			return url
		} else {
			result := p.matchPathSet(url, conf.AppContextPathSet)
			return result
		}
	default:
		// 두 번째 슬래시가 있는 경우
		if idx := strings.Index(url[1:], "/"); idx > 0 {
			url = url[:idx+1]
		}

		// 마지막 슬래시 제거 (있다면)
		if len(url) > 1 && url[len(url)-1] == '/' {
			url = url[:len(url)-1]
		}

		if conf.AppContextPathSet == "" {
			return url
		} else {
			result := p.matchPathSet(url, conf.AppContextPathSet)
			return result
		}
	}

	return ""
}

// AppContextPathSet에서 URL과 매칭되는 name 찾기
func (p *PathDefault) matchPathSet(url string, pathSet string) string {
	paths := stringutil.Tokenizer(pathSet, ",")
	for _, item := range paths {
		item = stringutil.TrimEmpty(item)

		if item == "" {
			continue
		}

		// @ 구분자가 있는 경우 name@url 형식으로 처리
		if strings.Contains(item, "@") {
			parts := strings.Split(item, "@")
			if len(parts) >= 2 {
				name := stringutil.TrimEmpty(parts[0])
				configUrl := stringutil.TrimEmpty(parts[1])

				if configUrl == url {
					return name
				}
			}
		} else {
			trimmedItem := stringutil.TrimEmpty(item)

			if trimmedItem == url {
				return url
			}
		}
	}

	return ""
}

func (p *PathDefault) Update() {
}
