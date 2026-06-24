// Package whatapmongo provides WhaTap instrumentation for MongoDB Go Driver.
package whatapmongo

import (
	"context"
	"fmt"
	"strings"

	whatapsql "github.com/whatap/go-api/sql"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Connect creates a traced MongoDB client.
// It wraps mongo.Connect and adds a CommandMonitor for tracing.
func Connect(ctx context.Context, opts ...*options.ClientOptions) (*mongo.Client, error) {
	mergedOpts := options.MergeClientOptions(opts...)

	// Extract URI for tracing
	uri := extractURI(mergedOpts)

	// Add tracing monitor
	monitor := NewMonitor(uri)

	// Merge existing monitor if present
	if mergedOpts.Monitor != nil {
		existingMonitor := mergedOpts.Monitor
		monitor = mergeMonitors(existingMonitor, monitor)
	}

	mergedOpts.SetMonitor(monitor)

	// Track connection establishment (SQL_TYPE_DBC)
	connection := fmt.Sprintf("mongodb://%s", uri)
	sqlCtx, _ := whatapsql.StartOpen(ctx, connection)
	client, err := mongo.Connect(ctx, mergedOpts)
	whatapsql.End(sqlCtx, err)

	return client, err
}

// NewClient creates a traced MongoDB client without connecting.
// Wraps the deprecated mongo.NewClient — adds CommandMonitor for query tracing
// but does NOT call Connect. The caller is expected to call client.Connect() separately.
//
// Deprecated: Use Connect instead for mongo-driver v1.12+.
func NewClient(opts ...*options.ClientOptions) (*mongo.Client, error) {
	mergedOpts := options.MergeClientOptions(opts...)

	// Extract URI for tracing
	uri := extractURI(mergedOpts)

	// Add tracing monitor
	monitor := NewMonitor(uri)

	// Merge existing monitor if present
	if mergedOpts.Monitor != nil {
		monitor = mergeMonitors(mergedOpts.Monitor, monitor)
	}
	mergedOpts.SetMonitor(monitor)

	//nolint:staticcheck // deprecated but required for old-pattern compatibility (§207)
	return mongo.NewClient(mergedOpts)
}

// extractURI extracts the connection URI from client options.
func extractURI(opts *options.ClientOptions) string {
	if opts == nil {
		return "unknown"
	}

	// Try to get hosts from options
	if opts.Hosts != nil && len(opts.Hosts) > 0 {
		return opts.Hosts[0]
	}

	return "unknown"
}

// mergeMonitors combines two CommandMonitors.
func mergeMonitors(existing, tracing *event.CommandMonitor) *event.CommandMonitor {
	return &event.CommandMonitor{
		Started: func(ctx context.Context, evt *event.CommandStartedEvent) {
			if existing.Started != nil {
				existing.Started(ctx, evt)
			}
			if tracing.Started != nil {
				tracing.Started(ctx, evt)
			}
		},
		Succeeded: func(ctx context.Context, evt *event.CommandSucceededEvent) {
			if existing.Succeeded != nil {
				existing.Succeeded(ctx, evt)
			}
			if tracing.Succeeded != nil {
				tracing.Succeeded(ctx, evt)
			}
		},
		Failed: func(ctx context.Context, evt *event.CommandFailedEvent) {
			if existing.Failed != nil {
				existing.Failed(ctx, evt)
			}
			if tracing.Failed != nil {
				tracing.Failed(ctx, evt)
			}
		},
	}
}

// formatConnectionString formats the connection info for display.
func formatConnectionString(host, database string) string {
	if host == "" || host == "unknown" {
		return fmt.Sprintf("mongodb://%s", database)
	}

	// Hide password if present in host
	if idx := strings.Index(host, "@"); idx > 0 {
		// Find the password part (between : and @)
		if colonIdx := strings.Index(host, ":"); colonIdx > 0 && colonIdx < idx {
			host = host[:colonIdx] + ":#" + host[idx:]
		}
	}

	return fmt.Sprintf("mongodb://%s/%s", host, database)
}
