package whatapsql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"sync"

	"github.com/whatap/go-api/config"
)

var (
	regMux        sync.Mutex
	whatapDrivers = make(map[string]driver.Driver)
)

func Open(driverName, dataSourceName string) (*sql.DB, error) {
	return OpenWithRegister(context.Background(), driverName, dataSourceName)
}
func OpenContext(ctx context.Context, driverName, dataSourceName string) (*sql.DB, error) {
	return OpenWithRegister(ctx, driverName, dataSourceName)
}

func OpenWithRegister(ctx context.Context, driverName, dataSourceName string) (*sql.DB, error) {
	conf := config.GetConfig()
	if !conf.GoSqlProfileEnabled {
		return sql.Open(driverName, dataSourceName)
	}
	regName := "whatap-" + driverName
	regMux.Lock()
	var whatapDriver driver.Driver
	wDri, ok := whatapDrivers[regName]
	if !ok {
		db, err := sql.Open(driverName, dataSourceName)
		if err != nil {
			return nil, err
		}
		dri := db.Driver()
		if err = db.Close(); err != nil {
			return nil, err
		}
		whatapDriver = driver.Driver(WhatapDriver{dri, ctx})
		whatapDrivers[regName] = whatapDriver
	} else {
		whatapDriver = wDri
	}
	regMux.Unlock()
	connector, err := whatapDriver.(WhatapDriver).OpenConnectorContext(dataSourceName, ctx)
	if err != nil {
		return nil, err
	}

	return sql.OpenDB(connector), nil
}

func OpenDB(ctx context.Context, dataSourceName string, connector driver.Connector) (*sql.DB, error) {
	connector = WrapConnector{connector, ctx, dataSourceName}
	return sql.OpenDB(connector), nil
}
