package whatapsqlx

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/whatap/go-api/instrumentation/database/sql/whatapsql"
)

// Open wraps sqlx.Open with WhaTap instrumentation.
// It uses whatapsql.Open internally to get a traced *sql.DB,
// then wraps it with sqlx.NewDb.
func Open(driverName, dataSourceName string) (*sqlx.DB, error) {
	db, err := whatapsql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	return sqlx.NewDb(db, driverName), nil
}

// OpenContext wraps sqlx.Open with WhaTap instrumentation and context.
func OpenContext(ctx context.Context, driverName, dataSourceName string) (*sqlx.DB, error) {
	db, err := whatapsql.OpenContext(ctx, driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	return sqlx.NewDb(db, driverName), nil
}

// Connect wraps sqlx.Connect with WhaTap instrumentation.
// It opens a connection and verifies it with a ping.
func Connect(driverName, dataSourceName string) (*sqlx.DB, error) {
	db, err := Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// ConnectContext wraps sqlx.ConnectContext with WhaTap instrumentation.
func ConnectContext(ctx context.Context, driverName, dataSourceName string) (*sqlx.DB, error) {
	db, err := OpenContext(ctx, driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	if err = db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// MustConnect wraps sqlx.MustConnect with WhaTap instrumentation.
// It panics on error.
func MustConnect(driverName, dataSourceName string) *sqlx.DB {
	db, err := Connect(driverName, dataSourceName)
	if err != nil {
		panic(err)
	}
	return db
}

// MustOpen wraps sqlx.MustOpen with WhaTap instrumentation.
// It panics on error.
func MustOpen(driverName, dataSourceName string) *sqlx.DB {
	db, err := Open(driverName, dataSourceName)
	if err != nil {
		panic(err)
	}
	return db
}
