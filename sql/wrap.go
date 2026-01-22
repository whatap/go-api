// sql/wrap.go - 범용 Wrap 함수 (Go 1.18+ Generics)
// Hook을 지원하지 않는 라이브러리(Aerospike, MongoDB 등) 계측용
package sql

import (
	"context"
)

// WrapError - error만 반환하는 함수용 (INSERT, UPDATE, DELETE 등)
//
// 사용 예시:
//
//	if err := sql.WrapError(ctx, "aerospike://host:3000", "PUT ns/set", func() error {
//	    return client.Put(policy, key, bins)
//	}); err != nil { ... }
func WrapError(ctx context.Context, dbhost, query string, fn func() error) error {
	sqlCtx, _ := Start(ctx, dbhost, query)
	err := fn()
	End(sqlCtx, err)
	return err
}

// WrapErrorP - error만 반환 + 파라미터 추적
//
// 사용 예시:
//
//	if err := sql.WrapErrorP(ctx, "mysql://host:3306", "INSERT INTO users VALUES (?, ?)",
//	    []interface{}{id, name},
//	    func() error { return db.Exec(...) },
//	); err != nil { ... }
func WrapErrorP(ctx context.Context, dbhost, query string, params []interface{}, fn func() error) error {
	sqlCtx, _ := StartWithParamArray(ctx, dbhost, query, params)
	err := fn()
	End(sqlCtx, err)
	return err
}

// Wrap - (T, error) 반환하는 함수용 (SELECT, GET 등)
//
// 사용 예시:
//
//	record, err := sql.Wrap(ctx, "aerospike://host:3000", "GET ns/set", func() (*aero.Record, error) {
//	    return client.Get(policy, key)
//	})
func Wrap[T any](ctx context.Context, dbhost, query string, fn func() (T, error)) (T, error) {
	sqlCtx, _ := Start(ctx, dbhost, query)
	result, err := fn()
	End(sqlCtx, err)
	return result, err
}

// WrapP - (T, error) 반환 + 파라미터 추적
//
// 사용 예시:
//
//	rows, err := sql.WrapP(ctx, "mysql://host:3306", "SELECT * FROM users WHERE id = ?",
//	    []interface{}{userId},
//	    func() (*sql.Rows, error) {
//	        return db.Query("SELECT * FROM users WHERE id = ?", userId)
//	    },
//	)
func WrapP[T any](ctx context.Context, dbhost, query string, params []interface{}, fn func() (T, error)) (T, error) {
	sqlCtx, _ := StartWithParamArray(ctx, dbhost, query, params)
	result, err := fn()
	End(sqlCtx, err)
	return result, err
}

// WrapOpen - DB 연결용 (StartOpen 사용)
//
// 사용 예시:
//
//	db, err := sql.WrapOpen(ctx, "mysql://host:3306", func() (*sql.DB, error) {
//	    return sql.Open("mysql", dsn)
//	})
func WrapOpen[T any](ctx context.Context, dbhost string, fn func() (T, error)) (T, error) {
	sqlCtx, _ := StartOpen(ctx, dbhost)
	result, err := fn()
	End(sqlCtx, err)
	return result, err
}
