package whatapmongo

import (
	"context"
	"errors"
	"sync"

	"github.com/whatap/go-api/sql"
	"github.com/whatap/go-api/trace"
	"go.mongodb.org/mongo-driver/event"
)

const (
	maxBytes      = 1000
	maxStateCount = 100000
)

type monitor struct {
	state map[int64]whatapContext
	/*
		state가 무한히 커지지 않게 하기 위해 크기를 제한하는 변수
		크기는 len() 대신 원소 추가 회수를 직접 세는 방식으로 측정
		golang의 map은 원소를 삭제하더라도 할당된 메모리가 줄어들지 않기 때문
		참고: https://github.com/golang/go/issues/20135
	*/
	stateCount int64
	mutex      sync.Mutex
	uri        string
}

func NewMonitor(uri string) *event.CommandMonitor {
	m := &monitor{
		state: map[int64]whatapContext{},
		mutex: sync.Mutex{},
		uri:   uri,
	}
	return &event.CommandMonitor{
		Started:   m.Started,
		Succeeded: m.Succeeded,
		Failed:    m.Failed,
	}
}

func (m *monitor) Started(ctx context.Context, evt *event.CommandStartedEvent) {
	startCtx, err := getStartCtx(ctx, evt, m.uri)
	if err != nil {
		return
	}

	m.Store(evt.RequestID, startCtx)
}

func (m *monitor) Succeeded(ctx context.Context, evt *event.CommandSucceededEvent) {
	m.finished(&evt.CommandFinishedEvent, nil)
}

func (m *monitor) Failed(ctx context.Context, evt *event.CommandFailedEvent) {
	m.finished(&evt.CommandFinishedEvent, errors.New(evt.Failure))
}

func (m *monitor) finished(evt *event.CommandFinishedEvent, err error) {
	endCtx, exist := m.Pop(evt.RequestID)
	if !exist {
		return
	}

	sql.End(endCtx.sqlCtx, err)

	if endCtx.wasStartedInMonitor {
		trace.End(endCtx.traceCtx, err)
	}
}

func (m *monitor) Store(requestID int64, ctx whatapContext) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.stateCount > maxStateCount {
		return
	}

	m.state[requestID] = ctx
	m.stateCount += 1
}

func (m *monitor) Pop(requestID int64) (ctx whatapContext, exist bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	ret, exist := m.state[requestID]
	if exist {
		delete(m.state, requestID)
		m.stateCount -= 1
	}
	//기존 map의 참조를 끊어서 GC 유도
	if len(m.state) == 0 {
		m.state = map[int64]whatapContext{}
		m.stateCount = 0
	}
	return ret, exist
}
