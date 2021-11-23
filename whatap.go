package main

import (
	"fmt"
        "context"

        "github.com/whatap/go-api/trace"
        _ "github.com/whatap/go-api/method"
        _ "github.com/whatap/go-api/sql"
        _ "github.com/whatap/go-api/httpc"
        _ "github.com/whatap/go-api/config"

)

func main(){
	fmt.Println("Whatap Golang api")

        wCtx,_ := trace.Start(context.Background(),"Test")
        trace.UpdateMtraceWithContext(wCtx, make(map[string][]string))

}
