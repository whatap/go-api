package whatapgorm

import (
	"context"

	"github.com/whatap/go-api/sql"
	"gorm.io/gorm"
)

const (
	gormContextStart    = "whatapGormContext"
	gormSQLContextStart = "whatapSQLGormContext"
)

type callbackFunc func(*gorm.DB)

func before(db *gorm.DB) {
	ctx := GetContext(db)

	sqlCtx, _ := sql.Start(ctx, db.Name(), db.Statement.SQL.String())
	db.Set(gormSQLContextStart, sqlCtx)
}

func after(db *gorm.DB) {
	v, ok := db.Get(gormSQLContextStart)
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
	cb.Row().Before("beforeRow").Register("whatapBeforeRow", beforeFunc)

	cb.Create().After("afterCreate").Register("whatapAfterCreate", afterFunc)
	cb.Update().After("afterUpdate").Register("whatapAfterUpdate", afterFunc)
	cb.Delete().After("afterDelete").Register("whatapAfterDelete", afterFunc)
	cb.Query().After("afterQuery").Register("whatapAfterQuery", afterFunc)
	cb.Row().After("afterRow").Register("whatapAfterRow", afterFunc)

	return db
}

func Open(dialector gorm.Dialector, cfg *gorm.Config) (*gorm.DB, error) {
	db, err := gorm.Open(dialector, cfg)
	if err != nil {
		return db, err
	}
	return withCallback(db, before, after), nil
}

func OpenWithContext(dialector gorm.Dialector, cfg *gorm.Config, ctx context.Context) (*gorm.DB, error) {
	db, err := gorm.Open(dialector, cfg)
	if err != nil {
		return db, err
	}

	return withCallback(db.Set(gormContextStart, ctx), before, after), nil
}

func GetContext(db *gorm.DB) context.Context {
	v, ok := db.Get(gormContextStart)
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
