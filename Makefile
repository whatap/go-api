test:   #compile warning 제거후 사용 가능
	go test ./...

build:
	go mod download -x
	go build ./...

clean :
	go clean 
