module github.com/whatap/go-api/instrumentation/github.com/fiber/whatapfiber

go 1.18

require (
	github.com/gofiber/fiber/v2 v2.36.0
	github.com/whatap/go-api v0.1.12
	github.com/whatap/go-api/instrumentation/github.com/valyala/fasthttp v0.0.0
)

require (
	github.com/andybalholm/brotli v1.0.4 // indirect
	github.com/google/uuid v1.1.2 // indirect
	github.com/klauspost/compress v1.15.6 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.39.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	github.com/whatap/golib v0.0.3 // indirect
	golang.org/x/sys v0.0.0-20220227234510-4e6760a101f9 // indirect
	golang.org/x/text v0.3.7 // indirect
)

replace (
	github.com/whatap/go-api v0.1.12 => /home/ubuntu/whatap-go/go-api
	github.com/whatap/go-api/instrumentation/github.com/valyala/fasthttp v0.0.0 => /home/ubuntu/whatap-go/go-api/instrumentation/github.com/valyala/fasthttp
	github.com/whatap/golib v0.0.1 => /home/ubuntu/whatap-go/golib
)
