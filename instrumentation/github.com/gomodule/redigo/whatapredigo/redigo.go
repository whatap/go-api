package whatapredigo

import (
	"context"
	"fmt"

	"github.com/gomodule/redigo/redis"
	"github.com/whatap/go-api/sql"
	"github.com/whatap/go-api/trace"
)

func wrappingConn(c redis.Conn, connection string, ctx context.Context) Conn {
	if connWithTimeout, ok := c.(redis.ConnWithTimeout); ok {
		return contextConnWithTimeout{ConnWithTimeout: connWithTimeout, connection: connection, ctx: ctx}
	}
	return contextConn{Conn: c, connection: connection, ctx: ctx}
}

func Dial(network, address string, options ...redis.DialOption) (Conn, error) {
	c, err := redis.Dial(network, address, options...)
	return wrappingConn(c, fmt.Sprintf("%s@%s", network, address), nil), err
}

func DialContext(ctx context.Context, network, address string, options ...redis.DialOption) (Conn, error) {
	c, err := redis.DialContext(ctx, network, address, options...)
	return wrappingConn(c, fmt.Sprintf("%s@%s", network, address), ctx), err
}

func DialURL(rawurl string, options ...redis.DialOption) (Conn, error) {
	c, err := redis.DialURL(rawurl, options...)
	return wrappingConn(c, rawurl, nil), err
}

func DialURLContext(ctx context.Context, rawurl string, options ...redis.DialOption) (Conn, error) {
	c, err := redis.DialURL(rawurl, options...)
	return wrappingConn(c, rawurl, ctx), err
}

type Conn interface {
	redis.Conn
	WithContext(ctx context.Context) Conn
}

type contextConn struct {
	redis.Conn
	connection string
	ctx        context.Context
}

func (c contextConn) WithContext(ctx context.Context) Conn {
	c.ctx = ctx
	return c
}

func (c contextConn) Do(commandName string, args ...interface{}) (reply interface{}, err error) {
	return DoRun(c.ctx, c.Conn, c.connection, commandName, args...)
}

func (c contextConn) Send(commandName string, args ...interface{}) error {
	return SendRun(c.ctx, c.Conn, c.connection, commandName, args...)
}

type contextConnWithTimeout struct {
	redis.ConnWithTimeout
	connection string
	ctx        context.Context
}

func (c contextConnWithTimeout) WithContext(ctx context.Context) Conn {
	c.ctx = ctx
	return c
}

func (c contextConnWithTimeout) Do(commandName string, args ...interface{}) (reply interface{}, err error) {
	return DoRun(c.ctx, c.ConnWithTimeout, c.connection, commandName, args...)
}

func (c contextConnWithTimeout) Send(commandName string, args ...interface{}) error {
	return SendRun(c.ctx, c.ConnWithTimeout, c.connection, commandName, args...)
}

func getCommandString(commandName string, args ...interface{}) string {
	var cmd string
	if commandName == "" {
		cmd = fmt.Sprintf("CLOSE")
	} else {
		cmd = commandName
	}
	return cmd
}

// Redis는SQL은 아니지만 같은 DB 계열임.  통계 처리를 위해 SQL로 처리
func DoRun(ctx context.Context, conn redis.Conn, connection, commandName string, args ...interface{}) (interface{}, error) {
	cmd := getCommandString(commandName, args)
	if ctx == nil {
		ctx, _ = trace.Start(context.Background(), connection)
		defer trace.End(ctx, nil)
	}

	sqlCtx, _ := sql.StartWithParam(ctx, connection, cmd, args...)
	ret, err := conn.Do(commandName, args...)
	sql.End(sqlCtx, nil)
	return ret, err
}

func SendRun(ctx context.Context, conn redis.Conn, connection, commandName string, args ...interface{}) error {
	cmd := getCommandString(commandName, args)
	if ctx == nil {
		ctx, _ = trace.Start(context.Background(), connection)
		defer trace.End(ctx, nil)
	}

	sqlCtx, _ := sql.StartWithParam(ctx, connection, cmd, args...)
	err := conn.Send(commandName, args...)
	sql.End(sqlCtx, nil)
	return err
}
