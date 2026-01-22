package whatapgoredis

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-redis/redis/v8"
	whatapsql "github.com/whatap/go-api/sql"
	"github.com/whatap/go-api/trace"
)

// contextKey is used to store SqlCtx in context
type contextKey struct{}

// sqlCtxValue wraps SqlCtx for context storage
type sqlCtxValue struct {
	sqlCtx *whatapsql.SqlCtx
}

// NewClient creates a new traced Redis client.
// It wraps redis.NewClient and adds tracing hooks.
func NewClient(opt *redis.Options) *redis.Client {
	client := redis.NewClient(opt)
	client.AddHook(&tracingHook{addr: opt.Addr})
	return client
}

// NewClusterClient creates a new traced Redis cluster client.
func NewClusterClient(opt *redis.ClusterOptions) *redis.ClusterClient {
	client := redis.NewClusterClient(opt)
	addr := strings.Join(opt.Addrs, ",")
	client.AddHook(&tracingHook{addr: addr})
	return client
}

// NewFailoverClient creates a new traced Redis failover client.
func NewFailoverClient(opt *redis.FailoverOptions) *redis.Client {
	client := redis.NewFailoverClient(opt)
	addr := fmt.Sprintf("sentinel://%s", opt.MasterName)
	client.AddHook(&tracingHook{addr: addr})
	return client
}

// NewRing creates a new traced Redis ring client.
func NewRing(opt *redis.RingOptions) *redis.Ring {
	client := redis.NewRing(opt)
	addrs := make([]string, 0, len(opt.Addrs))
	for name := range opt.Addrs {
		addrs = append(addrs, name)
	}
	addr := strings.Join(addrs, ",")
	client.AddHook(&tracingHook{addr: addr})
	return client
}

// WrapClient adds tracing hooks to an existing Redis client.
func WrapClient(client redis.UniversalClient) {
	switch c := client.(type) {
	case *redis.Client:
		c.AddHook(&tracingHook{addr: c.Options().Addr})
	case *redis.ClusterClient:
		opts := c.Options()
		addr := strings.Join(opts.Addrs, ",")
		c.AddHook(&tracingHook{addr: addr})
	case *redis.Ring:
		c.AddHook(&tracingHook{addr: "ring"})
	}
}

// tracingHook implements redis.Hook interface for WhaTap tracing.
// v8 uses BeforeProcess/AfterProcess pattern.
type tracingHook struct {
	addr string
}

// BeforeProcess implements redis.Hook.
func (h *tracingHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	if trace.DISABLE() {
		return ctx, nil
	}

	cmdStr := formatCmd(cmd)
	params := getParams(cmd)
	connection := fmt.Sprintf("redis://%s", h.addr)

	sqlCtx, _ := whatapsql.StartWithParamArray(ctx, connection, cmdStr, params)
	return context.WithValue(ctx, contextKey{}, &sqlCtxValue{sqlCtx: sqlCtx}), nil
}

// AfterProcess implements redis.Hook.
func (h *tracingHook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	if trace.DISABLE() {
		return nil
	}

	if val, ok := ctx.Value(contextKey{}).(*sqlCtxValue); ok && val.sqlCtx != nil {
		whatapsql.End(val.sqlCtx, cmd.Err())
	}
	return nil
}

// BeforeProcessPipeline implements redis.Hook.
func (h *tracingHook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
	if trace.DISABLE() {
		return ctx, nil
	}

	cmdStr := formatPipelineCmds(cmds)
	params := getPipelineParams(cmds)
	connection := fmt.Sprintf("redis://%s", h.addr)

	sqlCtx, _ := whatapsql.StartWithParamArray(ctx, connection, cmdStr, params)
	return context.WithValue(ctx, contextKey{}, &sqlCtxValue{sqlCtx: sqlCtx}), nil
}

// AfterProcessPipeline implements redis.Hook.
func (h *tracingHook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	if trace.DISABLE() {
		return nil
	}

	if val, ok := ctx.Value(contextKey{}).(*sqlCtxValue); ok && val.sqlCtx != nil {
		// Check for any errors in the pipeline
		var pipelineErr error
		for _, cmd := range cmds {
			if err := cmd.Err(); err != nil {
				pipelineErr = err
				break
			}
		}
		whatapsql.End(val.sqlCtx, pipelineErr)
	}
	return nil
}

// formatCmd formats a Redis command for tracing.
func formatCmd(cmd redis.Cmder) string {
	name := cmd.Name()
	args := cmd.Args()

	if len(args) <= 1 {
		return strings.ToUpper(name)
	}

	// Replace actual values with ? for security
	placeholders := make([]string, len(args)-1)
	for i := range placeholders {
		placeholders[i] = "?"
	}

	return fmt.Sprintf("%s (%s)", strings.ToUpper(name), strings.Join(placeholders, ", "))
}

// formatPipelineCmds formats multiple Redis commands for tracing.
func formatPipelineCmds(cmds []redis.Cmder) string {
	if len(cmds) == 0 {
		return "PIPELINE (empty)"
	}

	cmdNames := make([]string, len(cmds))
	for i, cmd := range cmds {
		cmdNames[i] = strings.ToUpper(cmd.Name())
	}

	return fmt.Sprintf("PIPELINE [%s]", strings.Join(cmdNames, ", "))
}

// getParams extracts command parameters for tracing.
func getParams(cmd redis.Cmder) []interface{} {
	if cmd == nil {
		return nil
	}
	args := cmd.Args()
	if len(args) <= 1 {
		return nil
	}
	return args[1:]
}

// getPipelineParams extracts all parameters from pipeline commands for tracing.
func getPipelineParams(cmds []redis.Cmder) []interface{} {
	if len(cmds) == 0 {
		return nil
	}
	var params []interface{}
	for _, cmd := range cmds {
		if cmd == nil {
			continue
		}
		args := cmd.Args()
		if len(args) > 1 {
			params = append(params, args[1:]...)
		}
	}
	return params
}
