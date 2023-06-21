GO=go

all: mod_tidy mod_download build test

mod_download:
	$(GO) mod download -x

mod_tidy:
	$(GO) mod tidy

test:   #compile warning 제거후 사용 가능
	$(GO) test ./... -cover

build:
	$(GO) build ./whatap.go

local:
	echo "replace github.com/whatap/golib v0.0.1 => ../golib" >> go.mod
	
clean :
	$(GO) clean -modcache
	$(GO) clean -testcache
	$(GO) clean -cache
	$(GO) clean
	rm -rf go.sum

version:
	$(GO) version
	$(GO) env
