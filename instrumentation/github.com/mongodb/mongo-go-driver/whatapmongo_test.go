package whatapmongo_test

import (
	"context"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	whatapmongo "github.com/whatap/go-api/instrumentation/github.com/mongodb/mongo-go-driver"
	"github.com/whatap/go-api/trace"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// 자동화된 테스트 아님. 디버그 출력 확인
func TestMongo(t *testing.T) {

	whatapConfig := map[string]string{
		"net_udp_port":              "6600",
		"debug":                     "true",
		"profile_sql_param_enabled": "true",
	}

	trace.Init(whatapConfig)
	defer trace.Shutdown()

	assert := assert.New(t)

	m := whatapmongo.NewMonitor("mongodb://unit-test:27017")
	assert.NotNil(m)

	t.Run("success case", func(t *testing.T) {
		start := &event.CommandStartedEvent{
			RequestID: int64(100),
		}

		m.Started(context.TODO(), start)

		success := &event.CommandSucceededEvent{
			CommandFinishedEvent: event.CommandFinishedEvent{
				RequestID: int64(100),
			},
		}

		m.Succeeded(context.TODO(), success)
	},
	)

	t.Run("failure case", func(t *testing.T) {
		start := &event.CommandStartedEvent{
			RequestID: int64(101),
		}
		m.Started(context.TODO(), start)

		failure := &event.CommandFailedEvent{
			CommandFinishedEvent: event.CommandFinishedEvent{
				RequestID: int64(101),
			},
			Failure: "test failure",
		}

		m.Failed(context.TODO(), failure)
	},
	)

	//localhost:27017로 실제 mongodb 실행해야 함
	t.Run("integration", func(t *testing.T) {
		opts := options.Client()
		opts.Monitor = whatapmongo.NewMonitor("mongodb://integration-" +
			RandStringBytes(5) +
			":27017")
		opts.ApplyURI("mongodb://localhost:27017")
		client, err := mongo.Connect(context.Background(), opts)
		if err != nil {
			panic(err)
		}
		db := client.Database("whatap")
		collection := db.Collection("go-apm")

		doc := bson.D{
			{Key: "key", Value: "value"},
			{Key: "integer", Value: 10},
			{Key: "bson.A", Value: bson.A{"bson"}},
			{Key: "bson.D", Value: bson.D{
				{Key: "a", Value: 1},
				{Key: "b", Value: 2},
				{Key: "c", Value: 3},
			},
			},
		}

		collection.InsertOne(context.TODO(), doc)
		collection.DeleteOne(context.TODO(), doc)
	},
	)
}
