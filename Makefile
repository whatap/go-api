
build:
	go mod download -x
	go build ./...

clean :
	go clean 
