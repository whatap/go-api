package whatapsql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"sync"

	"log"
	// "runtime/debug"
	//	whataptracesql "github.com/whatap/go-api/sql"
)

var (
	regMux        sync.Mutex
	whatapDrivers = make(map[string]driver.Driver)
)

// func Open(ctx context.Context, driverName, dataSourceName string) (*WrapSqlDB, error) {
// 	sqlCtx, _ := whataptracesql.StartOpen(ctx, dataSourceName)
// 	db, err := sql.Open(driverName, dataSourceName)
// 	whataptracesql.End(sqlCtx, err)

// 	return &WrapSqlDB{db, ctx, dataSourceName}, err
// }

func Open(driverName, dataSourceName string) (*sql.DB, error) {
	return OpenWithRegister(context.Background(), driverName, dataSourceName)
}
func OpenContext(ctx context.Context, driverName, dataSourceName string) (*sql.DB, error) {
	return OpenWithRegister(ctx, driverName, dataSourceName)
}

func OpenWithRegister(ctx context.Context, driverName, dataSourceName string) (*sql.DB, error) {
	regName := "whatap-" + driverName
	var whatapDriver driver.Driver
	regMux.Lock()
	if wDri, ok := whatapDrivers[regName]; !ok {
		log.Println("New Driver ", regName)
		db, err := sql.Open(driverName, dataSourceName)
		if err != nil {
			return nil, err
		}
		dri := db.Driver()
		if err = db.Close(); err != nil {
			return nil, err
		}
		whatapDrivers[regName] = dri
		whatapDriver = driver.Driver(WhatapDriver{dri, ctx})
	} else {
		log.Println("Already driver ", regName)
		whatapDriver = driver.Driver(WhatapDriver{wDri, ctx})
	}
	whatapDriver.(WhatapDriver).SetContext(ctx)
	regMux.Unlock()

	if driverCtx, ok := whatapDriver.(driver.DriverContext); ok {
		log.Println("Open DriverContext ")
		connector, err := driverCtx.OpenConnector(dataSourceName)
		if err != nil {
			return nil, err
		}
		connector.(WrapConnector).SetContext(ctx)
		return sql.OpenDB(connector), nil
	}

	return sql.OpenDB(driver.Connector(WrapConnector{dsnConnector{dsn: dataSourceName, driver: whatapDriver}, ctx, dataSourceName})), nil

	// regMux.Lock()
	// defer regMux.Unlock()
	// // Register whatap driver. prefix "whatap-"
	// //regName := "whatap-" + driverName
	// db, err := sql.Open(driverName, dataSourceName)
	// if err != nil {
	// 	return nil, err
	// }
	// dri := db.Driver()
	// if err = db.Close(); err != nil {
	// 	return nil, err
	// }
	// //whatapDrivers[regName] = dri
	// whatapDriver = driver.Driver(WhatapDriver{dri, ctx})
	// func() {
	// 	defer func() {
	// 		if r := recover(); r != nil {
	// 			// panic("sql: Register called twice for driver " + name)
	// 			log.Println("Recover sql.Register ", r)
	// 		}
	// 	}()
	// 	sql.Register(regName, whatapDriver)
	// }()

	// return sql.Open(regName, dataSourceName)
}

func OpenDB(ctx context.Context, dataSourceName string, connector driver.Connector) (*sql.DB, error) {
	connector = WrapConnector{connector, ctx, dataSourceName}
	return sql.OpenDB(connector), nil
}

// type WrapSqlDB struct {
// 	*sql.DB
// 	ctx            context.Context
// 	dataSourceName string
// }

// func (db *WrapSqlDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
// 	return db.DB.Query(query, args)
// }

// func (db *WrapSqlDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
// 	sqlCtx, _ := whataptracesql.StartWithParam(ctx, db.dataSourceName, query, args...)
// 	rows, err := db.DB.QueryContext(ctx, query, args...)
// 	whataptracesql.End(sqlCtx, err)
// 	return rows, err
// }
// func (db *WrapSqlDB) Prepare(query string) (*WrapSqlStmt, error) {
// 	stmt, err := db.DB.Prepare(query)
// 	if err == nil {
// 		wStmt := &WrapSqlStmt{stmt, db.ctx, db.dataSourceName, query}
// 		return wStmt, nil
// 	} else {
// 		return nil, err
// 	}
// }
// func (db *WrapSqlDB) PrepareContext(ctx context.Context, query string) (*WrapSqlStmt, error) {
// 	stmt, err := db.DB.PrepareContext(ctx, query)
// 	if err == nil {
// 		wStmt := &WrapSqlStmt{stmt, ctx, db.dataSourceName, query}
// 		return wStmt, nil
// 	} else {
// 		return nil, err
// 	}
// }

// type WrapSqlStmt struct {
// 	*sql.Stmt
// 	ctx            context.Context
// 	dataSourceName string
// 	preparedSql    string
// }

// func (s *WrapSqlStmt) Query(args ...interface{}) (*sql.Rows, error) {
// 	sqlCtx, _ := whataptracesql.StartWithParam(s.ctx, s.dataSourceName, s.preparedSql, args...)
// 	rows, err := s.Stmt.Query(args...)
// 	whataptracesql.End(sqlCtx, err)
// 	return rows, err
// }

// func (s *WrapSqlStmt) QueryContext(ctx context.Context, args ...interface{}) (*sql.Rows, error) {
// 	sqlCtx, _ := whataptracesql.StartWithParam(ctx, s.dataSourceName, s.preparedSql, args...)
// 	rows, err := s.Stmt.QueryContext(ctx, args...)
// 	whataptracesql.End(sqlCtx, err)
// 	return rows, err
// }

// func (s *WrapSqlStmt) QueryRow(args ...interface{}) *sql.Row {
// 	sqlCtx, _ := whataptracesql.StartWithParam(s.ctx, s.dataSourceName, s.preparedSql, args...)
// 	rows := s.Stmt.QueryRow(args...)
// 	whataptracesql.End(sqlCtx, nil)
// 	return rows
// }

// func (s *WrapSqlStmt) QueryRowContext(ctx context.Context, args ...interface{}) *sql.Row {
// 	sqlCtx, _ := whataptracesql.StartWithParam(ctx, s.dataSourceName, s.preparedSql, args...)
// 	rows := s.Stmt.QueryRowContext(ctx, args...)
// 	whataptracesql.End(sqlCtx, nil)
// 	return rows
// }

// func (s *WrapSqlStmt) Exec(args ...interface{}) (driver.Result, error) {
// 	sqlCtx, _ := whataptracesql.StartWithParam(s.ctx, s.dataSourceName, s.preparedSql, args...)
// 	res, err := s.Stmt.Exec(args...)
// 	whataptracesql.End(sqlCtx, err)
// 	return res, err
// }
// func (s *WrapSqlStmt) ExecContext(ctx context.Context, args ...interface{}) (driver.Result, error) {
// 	sqlCtx, _ := whataptracesql.StartWithParam(ctx, s.dataSourceName, s.preparedSql, args...)
// 	res, err := s.Stmt.ExecContext(ctx, args...)
// 	whataptracesql.End(sqlCtx, err)
// 	return res, err
// }
