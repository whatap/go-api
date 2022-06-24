package whatapgorm

import (
	"context"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/whatap/go-api/instrumentation/database/sql/whatapsql"
)

type Product struct {
	gorm.Model
	Code  int
	Price int
}

var beforeCheck bool = false
var afterCheck bool = false

func beforeTest(scope *gorm.Scope) {
	scope.Set("TEST", Product{Code: 1, Price: 2})
	beforeCheck = true
}
func afterTest(t *testing.T, scope *gorm.Scope) {
	assert := assert.New(t)
	v, ok := scope.Get("TEST")
	assert.True(ok)
	product := v.(Product)

	assert.Equal(product.Code, int(1))
	assert.Equal(product.Price, int(2))
	afterCheck = true
}

func TestWithCallback(t *testing.T) {
	assert := assert.New(t)
	afterFunc := func() func(*gorm.Scope) {
		return func(scope *gorm.Scope) {
			afterTest(t, scope)
		}
	}

	db, err := gorm.Open("sqlite3", "test.db")

	assert.Nil(err)

	db = withCallback(db, beforeTest, afterFunc())
	db = db.AutoMigrate(&Product{})
	assert.Nil(db.Error)

	var product Product
	tx := db.Create(&Product{Code: 1, Price: 2})
	assert.Nil(tx.Error)
	tx = db.Find(&product, "1 = 1")
	assert.Nil(tx.Error)
	tx = db.Unscoped().Delete(&Product{}, "1 = 1")
	assert.Nil(tx.Error)

	assert.True(beforeCheck)
	assert.True(afterCheck)
}

func TestOpen(t *testing.T) {
	assert := assert.New(t)
	db, err := Open("sqlite3", "test.db")
	assert.Nil(err)

	db = db.AutoMigrate(&Product{})
	assert.Nil(db.Error)

	var product Product
	tx := db.Create(&Product{Code: 1, Price: 2})
	assert.Nil(tx.Error)
	tx = db.Find(&product, "1 = 1")
	assert.Nil(tx.Error)

	assert.Equal(product.Code, int(1))
	assert.Equal(product.Price, int(2))

	tx = db.Unscoped().Delete(Product{}, "1 = 1")
	assert.Nil(tx.Error)
}

func TestOpenWithError(t *testing.T) {
	assert := assert.New(t)
	_, err := Open("mysql", nil)
	assert.NotNil(err)
	assert.Contains(err.Error(), "invalid database source")
}

func TestOpenWithContext(t *testing.T) {
	assert := assert.New(t)

	db, err := OpenWithContext(context.Background(), "sqlite3", "test.db")
	assert.Nil(err)

	db = db.AutoMigrate(&Product{})
	assert.Nil(db.Error)

	var product Product
	tx := db.Create(&Product{Code: 1, Price: 2})
	assert.Nil(tx.Error)
	tx = db.Find(&product, "1 = 1")
	assert.Nil(tx.Error)

	assert.Equal(product.Code, int(1))
	assert.Equal(product.Price, int(2))

	tx = db.Unscoped().Delete(&Product{}, "1 = 1")
	assert.Nil(tx.Error)
}

func TestOpenWitContextWithError(t *testing.T) {
	assert := assert.New(t)
	_, err := OpenWithContext(context.Background(), "sqlite3", nil)
	assert.NotNil(err)
	assert.Contains(err.Error(), "invalid database source")
}

func TestOpenWithSQLCommon(t *testing.T) {
	assert := assert.New(t)
	var conn gorm.SQLCommon
	var err error

	conn, err = whatapsql.Open("sqlite3", "test.db")

	db, err := Open("sqlite3", conn)
	assert.Nil(err)

	db = db.AutoMigrate(&Product{})
	assert.Nil(db.Error)

	var product Product
	tx := db.Create(&Product{Code: 1, Price: 2})
	assert.Nil(tx.Error)
	tx = db.Find(&product, "1 = 1")
	assert.Nil(tx.Error)

	assert.Equal(product.Code, int(1))
	assert.Equal(product.Price, int(2))

	tx = db.Unscoped().Delete(&Product{}, "1 = 1")
	assert.Nil(tx.Error)
}

func TestOpenWithContextWithSQLCommon(t *testing.T) {
	assert := assert.New(t)
	var conn gorm.SQLCommon
	var err error

	conn, err = whatapsql.Open("sqlite3", "test.db")

	db, err := OpenWithContext(context.Background(), "sqlite3", conn)
	assert.Nil(err)

	db = db.AutoMigrate(&Product{})
	assert.Nil(db.Error)

	var product Product
	tx := db.Create(&Product{Code: 1, Price: 2})
	assert.Nil(tx.Error)
	tx = db.Find(&product, "1 = 1")
	assert.Nil(tx.Error)

	assert.Equal(product.Code, int(1))
	assert.Equal(product.Price, int(2))

	tx = db.Unscoped().Delete(&Product{}, "1 = 1")
	assert.Nil(tx.Error)
}

func TestWithContext(t *testing.T) {
	assert := assert.New(t)

	db, err := Open("sqlite3", "test.db")
	assert.Nil(err)

	dbWithContext := WithContext(context.Background(), db)
	scope := dbWithContext.NewScope(nil)
	ctx := GetContext(scope)
	assert.NotNil(ctx)

	scope = db.NewScope(nil)
	ctx = GetContext(scope)
	assert.Nil(ctx)

}
