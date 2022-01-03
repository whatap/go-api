package main

import (
	"context"
	"fmt"

	_ "github.com/whatap/go-api/common/io"
	_ "github.com/whatap/go-api/common/lang/pack"
	_ "github.com/whatap/go-api/common/lang/pack/udp"
	_ "github.com/whatap/go-api/common/lang/value"
	_ "github.com/whatap/go-api/common/net"
	_ "github.com/whatap/go-api/common/util/bitutil"
	_ "github.com/whatap/go-api/common/util/compare"
	_ "github.com/whatap/go-api/common/util/dateutil"
	_ "github.com/whatap/go-api/common/util/hash"
	_ "github.com/whatap/go-api/common/util/hexa32"
	_ "github.com/whatap/go-api/common/util/hmap"
	_ "github.com/whatap/go-api/common/util/iputil"
	_ "github.com/whatap/go-api/common/util/keygen"
	_ "github.com/whatap/go-api/common/util/list"
	_ "github.com/whatap/go-api/common/util/paramtext"
	_ "github.com/whatap/go-api/common/util/stringutil"
	_ "github.com/whatap/go-api/common/util/urlutil"
	_ "github.com/whatap/go-api/config"
	_ "github.com/whatap/go-api/httpc"
	_ "github.com/whatap/go-api/method"
	_ "github.com/whatap/go-api/sql"
	"github.com/whatap/go-api/trace"
)

func main() {
	fmt.Println("Whatap Golang api")

	ctx, _ := trace.Start(context.Background(), "Test")
	trace.UpdateMtraceWithContext(ctx, make(map[string][]string))

}
