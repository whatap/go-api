package whatapgorm

import (
	"context"

	"github.com/jinzhu/gorm"
	"github.com/whatap/go-api/sql"
)

const (
	gormContextStart    = "whatapGormContext"
	gormSQLContextStart = "whatapSQLGormContext"
)

type callbackFunc func(*gorm.Scope)

func before(scope *gorm.Scope) {
	if scope == nil {
		return
	}
	ctx := GetContext(scope)

	sqlCtx, _ := sql.StartWithParamArray(ctx, scope.Dialect().GetName(), scope.SQL, scope.SQLVars)
	scope.Set(gormSQLContextStart, sqlCtx)
}

func after(scope *gorm.Scope) {
	if scope == nil {
		return
	}
	v, ok := scope.Get(gormSQLContextStart)
	if ok {
		sqlCtx := v.(*sql.SqlCtx)
		sql.End(sqlCtx, nil)
	}
}

func withCallback(db *gorm.DB, beforeFunc, afterFunc callbackFunc) *gorm.DB {
	cb := db.Callback()
	cb.Create().Before("beforeCreate").Register("whatapBeforeCreate", beforeFunc)
	cb.Update().Before("beforeUpdate").Register("whatapBeforeUpdate", beforeFunc)
	cb.Delete().Before("beforeDelete").Register("whatapBeforeDelete", beforeFunc)
	cb.Query().Before("beforeQuery").Register("whatapBeforeQuery", beforeFunc)
	cb.RowQuery().Before("beforeRowQuery").Register("whatapBeforeRowQuery", beforeFunc)

	cb.Create().After("afterCreate").Register("whatapAfterCreate", afterFunc)
	cb.Update().After("afterUpdate").Register("whatapAfterUpdate", afterFunc)
	cb.Delete().After("afterDelete").Register("whatapAfterDelete", afterFunc)
	cb.Query().After("afterQuery").Register("whatapAfterQuery", afterFunc)
	cb.RowQuery().After("afterRowQuery").Register("whatapAfterRowQuery", afterFunc)

	return db
}

func Open(dialect string, args ...interface{}) (*gorm.DB, error) {

	var db *gorm.DB
	var err error
	switch args[0].(type) {
	case gorm.SQLCommon:
		db, err = gorm.Open(dialect, args[0])
	default:
		db, err = gorm.Open(dialect, args...)
	}

	if err != nil {
		return db, err
	}
	return withCallback(db, before, after), nil
}

func OpenWithContext(ctx context.Context, dialect string, args ...interface{}) (*gorm.DB, error) {
	var db *gorm.DB
	var err error
	switch args[0].(type) {
	case gorm.SQLCommon:
		db, err = gorm.Open(dialect, args[0])
	default:
		db, err = gorm.Open(dialect, args...)
	}

	if err != nil {
		return db, err
	}

	return withCallback(db.Set(gormContextStart, ctx), before, after), nil
}

func GetContext(scope *gorm.Scope) context.Context {
	v, ok := scope.Get(gormContextStart)
	var ctx context.Context
	if ok {
		ctx, _ = v.(context.Context)
	} else {
		ctx = nil
	}
	return ctx
}

func WithContext(ctx context.Context, db *gorm.DB) *gorm.DB {
	return db.Set(gormContextStart, ctx)
}
