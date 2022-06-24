package whatapgorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Product struct {
	gorm.Model
	Code  int
	Price int
}

func beforeTest(db *gorm.DB) {
	db.Set("TEST", Product{Code: 1, Price: 2})
}
func afterTest(t *testing.T, db *gorm.DB) {
	assert := assert.New(t)
	v, ok := db.Get("TEST")
	assert.True(ok)
	product := v.(Product)

	assert.Equal(product.Code, int(1))
	assert.Equal(product.Price, int(2))
}

func TestWithCallback(t *testing.T) {
	assert := assert.New(t)

	afterFunc := func() func(*gorm.DB) {
		return func(db *gorm.DB) {
			afterTest(t, db)
		}
	}
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	assert.Nil(err)

	db = withCallback(db, beforeTest, afterFunc())

	err = db.AutoMigrate(&Product{})
	assert.Nil(err)

}
func TestOpen(t *testing.T) {
	assert := assert.New(t)
	db, err := Open(sqlite.Open("test.db"), &gorm.Config{})
	assert.Nil(err)

	err = db.AutoMigrate(&Product{})
	assert.Nil(err)

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

func TestOpenWithError(t *testing.T) {
	assert := assert.New(t)
	_, err := Open(mysql.New(mysql.Config{Conn: nil}), &gorm.Config{})
	assert.NotNil(err)
	assert.Contains(err.Error(), "connection refused")
}

func TestOpenWithContext(t *testing.T) {
	assert := assert.New(t)
	db, err := OpenWithContext(sqlite.Open("test.db"), &gorm.Config{}, context.Background())
	assert.Nil(err)

	err = db.AutoMigrate(&Product{})
	assert.Nil(err)

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
	_, err := OpenWithContext(mysql.New(mysql.Config{Conn: nil}), &gorm.Config{}, context.Background())
	assert.NotNil(err)
	assert.Contains(err.Error(), "connection refused")
}

func TestWithContext(t *testing.T) {
	assert := assert.New(t)
	db, err := Open(sqlite.Open("test.db"), &gorm.Config{})
	assert.Nil(err)

	dbWithContext := WithContext(context.Background(), db)

	ctx := GetContext(dbWithContext)
	assert.NotNil(ctx)

	notCtx := GetContext(db)
	assert.Nil(notCtx)

}
