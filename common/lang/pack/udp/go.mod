module github.com/whatap/go-api/common/lang/pack/udp

go 1.14

require (
	github.com/whatap/go-api/common/io v0.0.0
	github.com/whatap/go-api/common/util/dateutil v0.0.0
	github.com/whatap/go-api/common/util/paramtext v0.0.0
	github.com/whatap/go-api/common/util/stringutil v0.0.0
	github.com/whatap/go-api/common/util/urlutil v0.0.0
	golang.org/x/text v0.3.6 // indirect
)

replace github.com/whatap/go-api/common/io => ../../../io

replace github.com/whatap/go-api/common/util/dateutil => ../../../util/dateutil

replace github.com/whatap/go-api/common/util/paramtext => ../../../util/paramtext

replace github.com/whatap/go-api/common/util/urlutil => ../../../util/urlutil

replace github.com/whatap/go-api/common/util/stringutil => ../../../util/stringutil
