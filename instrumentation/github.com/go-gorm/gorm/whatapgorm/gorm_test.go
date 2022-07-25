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
	if assert.Nil(err) != true {
		return
	}

	db = withCallback(db, beforeTest, afterFunc())

	err = db.AutoMigrate(&Product{})
	if assert.Nil(err) != true {
		return
	}

}
func TestOpen(t *testing.T) {
	assert := assert.New(t)
	db, err := Open(sqlite.Open("test.db"), &gorm.Config{})
	if assert.Nil(err) != true {
		return
	}

	err = db.AutoMigrate(&Product{})
	if assert.Nil(err) != true {
		return
	}

	var product Product
	tx := db.Create(&Product{Code: 1, Price: 2})
	if assert.Nil(tx.Error) != true {
		return
	}

	tx = db.Find(&product, "1 = 1")
	if assert.Nil(tx.Error) != true {
		return
	}
	assert.Equal(product.Code, int(1))
	assert.Equal(product.Price, int(2))

	tx = db.Unscoped().Delete(&Product{}, "1 = 1")
	assert.Nil(tx.Error)
}

func TestOpenWithError(t *testing.T) {
	assert := assert.New(t)
	_, err := Open(mysql.New(mysql.Config{Conn: nil}), &gorm.Config{})
	if assert.NotNil(err) != true {
		return
	}
	assert.Contains(err.Error(), "connection refused")
}

func TestOpenWithContext(t *testing.T) {
	assert := assert.New(t)
	db, err := OpenWithContext(sqlite.Open("test.db"), &gorm.Config{}, context.Background())
	if assert.Nil(err) != true {
		return
	}

	err = db.AutoMigrate(&Product{})
	if assert.Nil(err) != true {
		return
	}

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
	if assert.NotNil(err) != true {
		return
	}
	assert.Contains(err.Error(), "connection refused")
}

func TestWithContext(t *testing.T) {
	assert := assert.New(t)
	db, err := Open(sqlite.Open("test.db"), &gorm.Config{})
	if assert.Nil(err) != true {
		return
	}

	dbWithContext := WithContext(context.Background(), db)

	ctx := GetContext(dbWithContext)
	assert.NotNil(ctx)

	notCtx := GetContext(db)
	assert.Nil(notCtx)

}
