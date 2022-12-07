all: mod_tidy mod_download build test

mod_download:
	go mod download -x

mod_tidy:
	go mod tidy

test:   #compile warning 제거후 사용 가능
	go test ./... -cover

build:
	go build ./...

local:
	echo "replace github.com/whatap/golib v0.0.1 => ../golib" >> go.mod
	
clean :
	go clean 
