module github.com/whatap/go-api

go 1.14

require (
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/go-sql-driver/mysql v1.6.0
	github.com/whatap/go-api/common/io v0.0.0
	github.com/whatap/go-api/common/lang/pack/udp v0.0.0
	github.com/whatap/go-api/common/net v0.0.0
	github.com/whatap/go-api/common/util/dateutil v0.0.0
	github.com/whatap/go-api/common/util/hash v0.0.0
	github.com/whatap/go-api/common/util/hexa32 v0.0.0
	github.com/whatap/go-api/common/util/keygen v0.0.0
	github.com/whatap/go-api/common/util/paramtext v0.0.0
	github.com/whatap/go-api/common/util/stringutil v0.0.0
	github.com/whatap/go-api/common/util/urlutil v0.0.0
	github.com/whatap/go-api/config v0.0.0
	github.com/whatap/go-api/httpc v0.0.0
	github.com/whatap/go-api/method v0.0.0
	github.com/whatap/go-api/sql v0.0.0
	github.com/whatap/go-api/trace v0.0.0

)

replace github.com/whatap/go-api/trace => ./trace

replace github.com/whatap/go-api/sql => ./sql

replace github.com/whatap/go-api/httpc => ./httpc

replace github.com/whatap/go-api/method => ./method

replace github.com/whatap/go-api/config => ./config

replace github.com/whatap/go-api/common/io => ./common/io

replace github.com/whatap/go-api/common/lang/pack/udp => ./common/lang/pack/udp

replace github.com/whatap/go-api/common/net => ./common/net

replace github.com/whatap/go-api/common/util/dateutil => ./common/util/dateutil

replace github.com/whatap/go-api/common/util/hash => ./common/util/hash

replace github.com/whatap/go-api/common/util/hexa32 => ./common/util/hexa32

replace github.com/whatap/go-api/common/util/keygen => ./common/util/keygen

replace github.com/whatap/go-api/common/util/paramtext => ./common/util/paramtext

replace github.com/whatap/go-api/common/util/stringutil => ./common/util/stringutil

replace github.com/whatap/go-api/common/util/urlutil => ./common/util/urlutil
