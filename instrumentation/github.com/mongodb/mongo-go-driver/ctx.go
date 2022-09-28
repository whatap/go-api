package whatapmongo

import (
	"context"

	"github.com/whatap/go-api/sql"
	"github.com/whatap/go-api/trace"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
)

type whatapContext struct {
	traceCtx            context.Context
	sqlCtx              *sql.SqlCtx
	wasStartedInMonitor bool
}

func traceCtxExist(ctx context.Context) bool {
	return ctx != nil &&
		ctx.Value("whatap") != nil
}

func getTraceCtx(ctx context.Context, uri string) (ctxRaw context.Context, traceStarted bool) {
	if traceCtxExist(ctx) {
		return ctx, false
	}
	ctx, err := trace.Start(ctx, uri)
	if err != nil {
		return ctx, false
	}
	return ctx, true
}

func getBsonString(raw bson.Raw) string {
	if len(raw) > sql.SQL_PARAM_VALUE_MAX_SIZE {
		ret := raw[:sql.SQL_PARAM_VALUE_MAX_SIZE].String()
		raw = nil
		return ret
	}
	return raw.String()
}

func getStartCtx(ctx context.Context, evt *event.CommandStartedEvent,
	uri string) (whatapContext, error) {

	traceCtx, started := getTraceCtx(ctx, uri)
	sqlCtx, err := sql.StartWithParam(traceCtx, uri,
		evt.CommandName,
		"database: "+evt.DatabaseName,
		"command: "+getBsonString(evt.Command),
	)
	if err != nil {
		return whatapContext{}, err
	}

	return whatapContext{
		traceCtx:            traceCtx,
		sqlCtx:              sqlCtx,
		wasStartedInMonitor: started,
	}, nil
}
