package main

import (
	"context"
	"fmt"

	_ "github.com/whatap/go-api/config"
	_ "github.com/whatap/go-api/counter"
	_ "github.com/whatap/go-api/counter/task"
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
