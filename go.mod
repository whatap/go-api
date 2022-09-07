module github.com/whatap/go-api

go 1.14

require (
	github.com/Shopify/sarama v1.34.1
	github.com/gin-gonic/gin v1.7.7
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gomodule/redigo v1.8.9
	github.com/gorilla/mux v1.8.0
	github.com/jinzhu/gorm v1.9.16
	github.com/labstack/echo v3.3.10+incompatible
	github.com/labstack/echo/v4 v4.7.2
	github.com/mattn/go-sqlite3 v1.14.12
	github.com/stretchr/testify v1.8.0
	github.com/whatap/golib v0.0.1
	golang.org/x/net v0.0.0-20220520000938-2e3eb7b945c2
	google.golang.org/grpc v1.42.0
	gorm.io/driver/mysql v1.3.4
	gorm.io/driver/sqlite v1.3.4
	gorm.io/gorm v1.23.6
)

replace (
	github.com/whatap/go-api/instrumentation/github.com/fiber/whatapfiber v0.0.1 => ./instrumentation/github.com/fiber
	github.com/whatap/golib v0.0.1 => ../golib
)
