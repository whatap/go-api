package whatapmongo

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"go.mongodb.org/mongo-driver/event"
	whatapsql "github.com/whatap/go-api/sql"
	"github.com/whatap/go-api/trace"
)

// spanKey is used to correlate Started/Succeeded/Failed events.
type spanKey struct {
	ConnectionID string
	RequestID    int64
}

// tracingMonitor implements CommandMonitor callbacks for WhaTap tracing.
type tracingMonitor struct {
	host  string
	spans sync.Map // map[spanKey]*whatapsql.SqlCtx
}

// NewMonitor creates a new CommandMonitor for MongoDB tracing.
func NewMonitor(host string) *event.CommandMonitor {
	tm := &tracingMonitor{host: host}
	return &event.CommandMonitor{
		Started:   tm.Started,
		Succeeded: tm.Succeeded,
		Failed:    tm.Failed,
	}
}

// Started is called when a MongoDB command starts.
func (tm *tracingMonitor) Started(ctx context.Context, evt *event.CommandStartedEvent) {
	if trace.DISABLE() {
		return
	}

	// Format command for display
	cmdStr := formatCommand(evt.CommandName, evt.DatabaseName)
	connection := formatConnectionString(tm.host, evt.DatabaseName)

	sqlCtx, _ := whatapsql.Start(ctx, connection, cmdStr)

	// Store span for later correlation
	key := spanKey{
		ConnectionID: evt.ConnectionID,
		RequestID:    evt.RequestID,
	}
	tm.spans.Store(key, sqlCtx)
}

// Succeeded is called when a MongoDB command succeeds.
func (tm *tracingMonitor) Succeeded(ctx context.Context, evt *event.CommandSucceededEvent) {
	if trace.DISABLE() {
		return
	}

	key := spanKey{
		ConnectionID: evt.ConnectionID,
		RequestID:    evt.RequestID,
	}

	if value, ok := tm.spans.LoadAndDelete(key); ok {
		if sqlCtx, ok := value.(*whatapsql.SqlCtx); ok {
			whatapsql.End(sqlCtx, nil)
		}
	}
}

// Failed is called when a MongoDB command fails.
func (tm *tracingMonitor) Failed(ctx context.Context, evt *event.CommandFailedEvent) {
	if trace.DISABLE() {
		return
	}

	key := spanKey{
		ConnectionID: evt.ConnectionID,
		RequestID:    evt.RequestID,
	}

	if value, ok := tm.spans.LoadAndDelete(key); ok {
		if sqlCtx, ok := value.(*whatapsql.SqlCtx); ok {
			whatapsql.End(sqlCtx, fmt.Errorf("%s", evt.Failure))
		}
	}
}

// formatCommand formats a MongoDB command for tracing.
func formatCommand(cmdName, dbName string) string {
	// Format: COMMAND_NAME (database)
	return fmt.Sprintf("%s (%s)", strings.ToUpper(cmdName), dbName)
}
