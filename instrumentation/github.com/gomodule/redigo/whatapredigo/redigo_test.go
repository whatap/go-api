package whatapredigo

import (
	"context"
	"testing"

	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
	"github.com/whatap/go-api/trace"
)

// Demo 환경에 따라 수정 필요
const (
	network     = "tcp"
	address     = "phpdemo3:6379"
	failAddress = "127.0.0.1:6380"
	url         = "redis://phpdemo3:6379"
)

func dialTest(t *testing.T, conn Conn) {
	assert := assert.New(t)

	var res interface{}
	var err error

	res, err = conn.Do("SET", "KEY", "VALUE")
	assert.Nil(err)
	assert.Contains(res, "OK")

	byteRes, err := redis.Bytes(conn.Do("GET", "KEY"))
	assert.Nil(err)

	assert.Contains(string(byteRes), "VALUE")

	_, err = conn.Do("NOT_COMMAND")
	assert.NotNil(err)
	assert.Contains(err.Error(), "unknown command")

	sendErr := conn.Send("SET", "KEY", "VALUE")
	assert.Nil(sendErr)

	err = conn.Flush()
	assert.Nil(err)

	data, err := conn.Receive()
	assert.Contains(data, "OK")
	assert.Nil(err)

}

func TestDialContext(t *testing.T) {
	whatapConfig := make(map[string]string)
	trace.Init(whatapConfig)
	defer trace.Shutdown()

	assert := assert.New(t)

	ctx, err := trace.Start(context.Background(), "TEST")
	if assert.Nil(err) != true {
		return
	}

	conn, err := DialContext(ctx, network, address)
	if assert.Nil(err) != true {
		return
	}

	dialTest(t, conn)
	conn.Close()
}

func TestDialURLContext(t *testing.T) {
	whatapConfig := make(map[string]string)
	trace.Init(whatapConfig)
	defer trace.Shutdown()

	assert := assert.New(t)

	ctx, err := trace.Start(context.Background(), "TEST")
	if assert.Nil(err) != true {
		return
	}

	conn, err := DialURLContext(ctx, url)
	if assert.Nil(err) != true {
		return
	}

	dialTest(t, conn)
	conn.Close()

}

func TestDial(t *testing.T) {
	whatapConfig := make(map[string]string)
	trace.Init(whatapConfig)
	defer trace.Shutdown()

	assert := assert.New(t)

	ctx, err := trace.Start(context.Background(), "TEST")
	if assert.Nil(err) != true {
		return
	}

	conn, err := Dial(network, address)
	if assert.Nil(err) != true {
		return
	}

	conn.WithContext(ctx)

	dialTest(t, conn)
	conn.Close()
}

func TestDialURL(t *testing.T) {
	whatapConfig := make(map[string]string)
	trace.Init(whatapConfig)
	defer trace.Shutdown()

	assert := assert.New(t)

	ctx, err := trace.Start(context.Background(), "TEST")
	if assert.Nil(err) != true {
		return
	}

	conn, err := DialURL(url)
	if assert.Nil(err) != true {
		return
	}

	conn.WithContext(ctx)

	dialTest(t, conn)
	conn.Close()
}

func TestConnectionError(t *testing.T) {
	whatapConfig := make(map[string]string)
	trace.Init(whatapConfig)
	defer trace.Shutdown()

	assert := assert.New(t)

	ctx, _ := trace.Start(context.Background(), "TEST")
	_, err := DialContext(ctx, network, failAddress)

	assert.Contains(err.Error(), "connect: connection refused")
	assert.NotNil(err)
}
