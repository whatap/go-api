package main

import (
	"context"
	"fmt"

	_ "github.com/whatap/go-api/agent/agent/active"
	_ "github.com/whatap/go-api/agent/agent/alert"
	_ "github.com/whatap/go-api/agent/agent/boot"
	_ "github.com/whatap/go-api/agent/agent/config"
	_ "github.com/whatap/go-api/agent/agent/control"
	_ "github.com/whatap/go-api/agent/agent/counter"
	_ "github.com/whatap/go-api/agent/agent/counter/meter"
	_ "github.com/whatap/go-api/agent/agent/countertag"
	_ "github.com/whatap/go-api/agent/agent/data"
	_ "github.com/whatap/go-api/agent/agent/kube"
	_ "github.com/whatap/go-api/agent/agent/pprof"
	_ "github.com/whatap/go-api/agent/agent/secure"
	_ "github.com/whatap/go-api/agent/agent/stat"
	_ "github.com/whatap/go-api/agent/agent/topology"
	_ "github.com/whatap/go-api/agent/agent/trace"
	_ "github.com/whatap/go-api/agent/agent/trace/api"
	_ "github.com/whatap/go-api/agent/alert"
	_ "github.com/whatap/go-api/agent/lang/conf"
	_ "github.com/whatap/go-api/agent/logsink/watch"
	_ "github.com/whatap/go-api/agent/logsink/zip"
	_ "github.com/whatap/go-api/agent/net"
	_ "github.com/whatap/go-api/agent/util/crypto"
	_ "github.com/whatap/go-api/agent/util/logutil"
	_ "github.com/whatap/go-api/agent/util/oidutil"
	_ "github.com/whatap/go-api/agent/util/sqlutil"
	_ "github.com/whatap/go-api/agent/util/sys"

	_ "github.com/whatap/go-api/httpc"
	_ "github.com/whatap/go-api/instrumentation/database/sql/whatapsql"
	_ "github.com/whatap/go-api/instrumentation/github.com/Shopify/sarama/whatapsarama"
	_ "github.com/whatap/go-api/instrumentation/github.com/gin-gonic/gin/whatapgin"
	_ "github.com/whatap/go-api/instrumentation/github.com/go-chi/chi/whatapchi"
	_ "github.com/whatap/go-api/instrumentation/github.com/go-gorm/gorm/whatapgorm"
	_ "github.com/whatap/go-api/instrumentation/github.com/gofiber/fiber/v2/whatapfiber"
	_ "github.com/whatap/go-api/instrumentation/github.com/gomodule/redigo/whatapredigo"
	_ "github.com/whatap/go-api/instrumentation/github.com/gorilla/mux/whatapmux"
	_ "github.com/whatap/go-api/instrumentation/github.com/jinzhu/gorm/whatapgorm"
	_ "github.com/whatap/go-api/instrumentation/github.com/labstack/echo/v4/whatapecho"
	_ "github.com/whatap/go-api/instrumentation/github.com/labstack/echo/whatapecho"
	_ "github.com/whatap/go-api/instrumentation/github.com/valyala/fasthttp/whatapfasthttp"
	_ "github.com/whatap/go-api/instrumentation/google.golang.org/grpc/whatapgrpc"
	_ "github.com/whatap/go-api/instrumentation/k8s.io/client-go/kubernetes/whatapkubernetes"
	_ "github.com/whatap/go-api/instrumentation/net/http/whataphttp"
	_ "github.com/whatap/go-api/method"
	_ "github.com/whatap/go-api/sql"
	"github.com/whatap/go-api/trace"
)

func main() {
	fmt.Println("Whatap Golang api")

	ctx, _ := trace.Start(context.Background(), "Test")
	trace.UpdateMtraceWithContext(ctx, make(map[string][]string))

}
