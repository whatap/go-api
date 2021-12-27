package whatapsql

import (
	"context"
	"database/sql/driver"
	"errors"

	"github.com/whatap/go-api/config"
	whatapsql "github.com/whatap/go-api/sql"
	"github.com/whatap/go-api/trace"
)

type dsnConnector struct {
	dsn    string
	driver driver.Driver
}

func (t dsnConnector) Connect(_ context.Context) (driver.Conn, error) {
	return t.driver.Open(t.dsn)
}

func (t dsnConnector) Driver() driver.Driver {
	return t.driver
}

type WhatapDriver struct {
	driver.Driver
	ctx context.Context
}

func (d WhatapDriver) Open(name string) (driver.Conn, error) {
	return d.Driver.Open(name)
}

func (d WhatapDriver) OpenConnector(name string) (c driver.Connector, err error) {
	if dCtx, ok := d.Driver.(driver.DriverContext); ok {
		c, err = dCtx.OpenConnector(name)
		if err != nil {
			return nil, err
		}
		return WrapConnector{c, d.ctx, name}, nil
	}
	return driver.Connector(WrapConnector{dsnConnector{name, d}, d.ctx, name}), nil
}

func (d WhatapDriver) OpenConnectorContext(name string, ctx context.Context) (c driver.Connector, err error) {
	if dCtx, ok := d.Driver.(driver.DriverContext); ok {
		c, err = dCtx.OpenConnector(name)
		if err != nil {
			return nil, err
		}
		return WrapConnector{c, ctx, name}, nil
	}
	return driver.Connector(WrapConnector{dsnConnector{name, d}, ctx, name}), nil
}

type WrapConnector struct {
	driver.Connector
	ctx            context.Context
	dataSourceName string
}

func (ct WrapConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conf := config.GetConfig()
	if !conf.GoSqlProfileEnabled {
		return ct.Connector.Connect(ctx)
	}

	wCtx := selectContext(ctx, ct.ctx)
	sqlCtx, _ := whatapsql.StartOpen(wCtx, ct.dataSourceName)
	c, err := ct.Connector.Connect(ctx)
	whatapsql.End(sqlCtx, err)
	if err != nil {
		return nil, err
	}
	return driver.Conn(WrapConn{c, wCtx, ct.dataSourceName}), err
}

type WrapConn struct {
	driver.Conn
	ctx            context.Context
	dataSourceName string
}

func (c WrapConn) Exec(query string, args []driver.Value) (res driver.Result, err error) {
	if exec, ok := c.Conn.(driver.Execer); ok {
		sqlCtx, _ := whatapsql.StartWithParam(c.ctx, c.dataSourceName, query, convertDriverValue(args)...)
		res, err := exec.Exec(query, args)
		whatapsql.End(sqlCtx, err)
		return res, err
	}
	return nil, driver.ErrSkip
}

func (c WrapConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (res driver.Result, err error) {
	wCtx := selectContext(ctx, c.ctx)
	if execCtx, ok := c.Conn.(driver.ExecerContext); ok {
		sqlCtx, _ := whatapsql.StartWithParam(wCtx, c.dataSourceName, query, convertDriverNamedValue(args)...)
		res, err := execCtx.ExecContext(ctx, query, args)
		whatapsql.End(sqlCtx, err)
		return res, err
	}
	return nil, driver.ErrSkip
}

func (c WrapConn) Query(query string, args []driver.Value) (rows driver.Rows, err error) {
	if queryer, ok := c.Conn.(driver.Queryer); ok {
		sqlCtx, _ := whatapsql.StartWithParam(c.ctx, c.dataSourceName, query, convertDriverValue(args)...)
		res, err := queryer.Query(query, args)
		whatapsql.End(sqlCtx, err)
		return res, err
	}
	return nil, driver.ErrSkip
}

func (c WrapConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (rows driver.Rows, err error) {
	wCtx := selectContext(ctx, c.ctx)
	if queryerCtx, ok := c.Conn.(driver.QueryerContext); ok {
		sqlCtx, _ := whatapsql.StartWithParam(wCtx, c.dataSourceName, query, convertDriverNamedValue(args)...)
		res, err := queryerCtx.QueryContext(ctx, query, args)
		whatapsql.End(sqlCtx, err)
		return res, err
	}
	return nil, driver.ErrSkip
}
func (c WrapConn) Prepare(query string) (stmt driver.Stmt, err error) {
	stmt, err = c.Conn.Prepare(query)
	if err != nil {
		return nil, err
	}
	return driver.Stmt(WrapStmt{stmt, c.ctx, c.dataSourceName, query}), err
}

func (c WrapConn) PrepareContext(ctx context.Context, query string) (stmt driver.Stmt, err error) {
	wCtx := selectContext(ctx, c.ctx)
	if prepCtx, ok := c.Conn.(driver.ConnPrepareContext); ok {
		stmt, err = prepCtx.PrepareContext(ctx, query)
	} else {
		stmt, err = c.Conn.Prepare(query)
	}
	if err != nil {
		return nil, err
	}
	return driver.Stmt(WrapStmt{stmt, wCtx, c.dataSourceName, query}), err
}

func (c WrapConn) Close() error {
	sqlCtx, _ := whatapsql.Start(c.ctx, c.dataSourceName, "Close")
	err := c.Conn.Close()
	whatapsql.End(sqlCtx, err)
	return err
}

func (c WrapConn) Begin() (tx driver.Tx, err error) {
	sqlCtx, _ := whatapsql.Start(c.ctx, c.dataSourceName, "Begin")
	tx, err = c.Conn.Begin()
	whatapsql.End(sqlCtx, err)
	if err != nil {
		return nil, err
	}
	return WrapTx{tx, c.ctx, c.dataSourceName}, nil
}

func (c WrapConn) BeginTx(ctx context.Context, opts driver.TxOptions) (tx driver.Tx, err error) {
	wCtx := selectContext(ctx, c.ctx)
	if connBeginTx, ok := c.Conn.(driver.ConnBeginTx); ok {
		sqlCtx, _ := whatapsql.Start(wCtx, c.dataSourceName, "BeginTx")
		tx, err = connBeginTx.BeginTx(ctx, opts)
		whatapsql.End(sqlCtx, err)
		if err != nil {
			return nil, err
		}
		return WrapTx{tx, c.ctx, c.dataSourceName}, nil
	}
	tx, err = c.Conn.Begin()
	if err != nil {
		return nil, err
	}
	return WrapTx{tx, c.ctx, c.dataSourceName}, nil
}

type WrapStmt struct {
	driver.Stmt
	ctx            context.Context
	dataSourceName string
	preparedSql    string
}

func (s WrapStmt) Exec(args []driver.Value) (res driver.Result, err error) {
	sqlCtx, _ := whatapsql.StartWithParam(s.ctx, s.dataSourceName, s.preparedSql, convertDriverValue(args)...)
	res, err = s.Stmt.Exec(args)
	whatapsql.End(sqlCtx, err)
	return res, err
}

func (s WrapStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (res driver.Result, err error) {
	wCtx := selectContext(ctx, s.ctx)
	if execCtx, ok := s.Stmt.(driver.StmtExecContext); ok {
		sqlCtx, _ := whatapsql.StartWithParam(wCtx, s.dataSourceName, s.preparedSql, convertDriverNamedValue(args)...)
		res, err := execCtx.ExecContext(ctx, args)
		whatapsql.End(sqlCtx, err)
		return res, err
	}
	dargs, err := namedValueToValue(args)
	if err != nil {
		return nil, err
	}
	return s.Stmt.Exec(dargs)
}

func (s WrapStmt) Query(args []driver.Value) (rows driver.Rows, err error) {
	sqlCtx, _ := whatapsql.StartWithParam(s.ctx, s.dataSourceName, s.preparedSql, convertDriverValue(args)...)
	res, err := s.Stmt.Query(args)
	whatapsql.End(sqlCtx, err)
	return res, err
}

func (s WrapStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (rows driver.Rows, err error) {
	wCtx := selectContext(ctx, s.ctx)
	if queryerCtx, ok := s.Stmt.(driver.StmtQueryContext); ok {
		sqlCtx, _ := whatapsql.StartWithParam(wCtx, s.dataSourceName, s.preparedSql, convertDriverNamedValue(args)...)
		res, err := queryerCtx.QueryContext(ctx, args)
		whatapsql.End(sqlCtx, err)
		return res, err
	}
	dargs, err := namedValueToValue(args)
	if err != nil {
		return nil, err
	}
	return s.Stmt.Query(dargs)
}

type WrapTx struct {
	driver.Tx
	ctx            context.Context
	dataSourceName string
}

func (t WrapTx) Commit() (err error) {
	sqlCtx, _ := whatapsql.Start(t.ctx, t.dataSourceName, "Commit")
	err = t.Tx.Commit()
	whatapsql.End(sqlCtx, err)
	return err
}

func (t WrapTx) Rollback() (err error) {
	sqlCtx, _ := whatapsql.Start(t.ctx, t.dataSourceName, "Commit")
	err = t.Tx.Rollback()
	whatapsql.End(sqlCtx, err)
	return err
}

func convertDriverValue(args []driver.Value) []interface{} {
	iArgs := make([]interface{}, 0)
	for _, it := range args {

		iArgs = append(iArgs, interface{}(it))
	}
	return iArgs
}

func convertDriverNamedValue(args []driver.NamedValue) []interface{} {
	iArgs := make([]interface{}, 0)
	for _, it := range args {
		iArgs = append(iArgs, it)
	}
	return iArgs
}

func namedValueToValue(named []driver.NamedValue) ([]driver.Value, error) {
	dargs := make([]driver.Value, len(named))
	for n, param := range named {
		if len(param.Name) > 0 {
			return nil, errors.New("sql: driver does not support the use of Named Parameters")
		}
		dargs[n] = param.Value
	}
	return dargs, nil
}

func selectContext(contexts ...context.Context) (ctx context.Context) {
	var first context.Context
	for i, it := range contexts {
		if i == 0 {
			first = it
		}
		if _, traceCtx := trace.GetTraceContext(it); traceCtx != nil {
			return it
		}
	}
	return first
}
