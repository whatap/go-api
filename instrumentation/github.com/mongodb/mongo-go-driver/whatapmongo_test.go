package whatapmongo_test

import (
	"context"
	"math/rand"
	"testing"

	whatapmongo "github.com/whatap/go-api/instrumentation/github.com/mongodb/mongo-go-driver"
	"github.com/whatap/go-api/trace"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func TestClient(t *testing.T) {
	assert := assert.New(t)

	t.Run("with mock object", func(t *testing.T) {
		c := whatapmongo.WrappedClient{
			MongoClient: whatapmongo.MockClient{},
			Options:     whatapmongo.MockOptions{},
		}
		assert.Equal("mongodb://localhost:21017", c.Options.GetURI())

		ctx := context.TODO()
		err := c.Connect(ctx)
		assert.Nil(err)

		err = c.Ping(ctx, readpref.Primary())
		assert.Nil(err)

		dbs, err := c.ListDatabases(ctx, nil)
		assert.Nil(err)
		assert.NotNil(dbs)

		names, err := c.ListDatabaseNames(ctx, nil)
		assert.Nil(err)
		assert.NotNil(names)

		db := c.Database("skynet")
		assert.NotNil(db)
		assert.Equal(&c, db.Client())

		sessions := c.NumberSessionsInProgress()
		assert.Equal(whatapmongo.StubSessions, sessions)
		err = c.Disconnect(ctx)
		assert.Nil(err)

		err = c.UseSession(ctx, func(mongo.SessionContext) error {
			return nil
		})
		assert.Nil(err)

		err = c.UseSessionWithOptions(ctx, &options.SessionOptions{},
			func(mongo.SessionContext) error {
				return nil
			},
		)
		assert.Nil(err)

		_, err = c.Watch(ctx, mongo.Pipeline{})
		assert.Nil(err)
	},
	)
}

func TestCollection(t *testing.T) {
	assert := assert.New(t)
	wdb := &whatapmongo.WrappedDatabase{}
	wdb.CollectionFactory = func(db *whatapmongo.WrappedDatabase, name string,
		opts ...*options.CollectionOptions) *whatapmongo.WrappedCollection {
		return &whatapmongo.WrappedCollection{
			WrappedDB: wdb,
		}
	}

	coll := wdb.Collection("collection", nil)
	assert.NotNil(coll)

	db := coll.Database()
	assert.NotNil(db)

}

func TestCommand(t *testing.T) {
	assert := assert.New(t)
	coll := whatapmongo.StubCollection{}

	t.Run("count documents", func(t *testing.T) {
		countDocuments := whatapmongo.CountDocuments{}

		count, err := countDocuments.Run(context.TODO(), coll)
		assert.Nil(err)
		assert.Equal(whatapmongo.StubCount, count)
		assert.Equal("CountDocuments", countDocuments.Name())
	},
	)

	t.Run("delete tests", func(t *testing.T) {
		Assert := func(cmd whatapmongo.Command) {
			resultRaw, err := cmd.Run(context.TODO(), coll)
			assert.Nil(err)
			result, typeMatched := resultRaw.(*mongo.DeleteResult)
			assert.True(typeMatched)
			assert.Equal(whatapmongo.StubDeleted, result.DeletedCount)
		}

		Assert(whatapmongo.DeleteOne{})
		Assert(whatapmongo.DeleteMany{})
	},
	)

	t.Run("update tests", func(t *testing.T) {
		Assert := func(cmd whatapmongo.Command) {
			resultRaw, err := cmd.Run(context.TODO(), coll)
			assert.Nil(err)
			result, typeMatched := resultRaw.(*mongo.UpdateResult)
			assert.True(typeMatched)
			assert.Equal(whatapmongo.StubMatched, result.MatchedCount)
		}

		commands := []whatapmongo.Command{
			whatapmongo.UpdateByID{},
			whatapmongo.UpdateOne{},
			whatapmongo.UpdateMany{},
			whatapmongo.ReplaceOne{},
		}

		for _, cmd := range commands {
			Assert(cmd)
		}
	},
	)

	t.Run("Bulk write", func(t *testing.T) {
		bulkWrite := whatapmongo.BulkWrite{}

		resultRaw, err := bulkWrite.Run(context.TODO(), coll)
		assert.Nil(err)
		result, typeMatched := resultRaw.(*mongo.BulkWriteResult)
		assert.True(typeMatched)
		assert.Equal(whatapmongo.StubInserted, result.InsertedCount)
		assert.Equal(whatapmongo.StubMatched, result.MatchedCount)
		assert.Equal(whatapmongo.StubModified, result.ModifiedCount)
		assert.Equal(whatapmongo.StubDeleted, result.DeletedCount)
		assert.Equal(whatapmongo.StubUpserted, result.UpsertedCount)
	},
	)

	t.Run("Drop test", func(t *testing.T) {
		drop := whatapmongo.Drop{
			CollectionName: "unitTest",
		}
		result, err := drop.Run(context.TODO(), coll)
		assert.Nil(result)
		assert.Nil(err)
	},
	)

}

func TestUriParser(t *testing.T) {
	assert := assert.New(t)

	testTable := [][]string{
		//Stand alone
		{"tcp@mongodb0.example.com:27017", "mongodb://mongodb0.example.com:27017"},
		{"tcp@sample.host:27017", "mongodb://sample.host:27017/?maxPoolSize=20&w=majority"},
		{"tcp@wrongFormatAddress", "wrongFormatAddress"},
		{"tcp@user@sample.host:27017", "mongodb://user:pass@sample.host:27017/?maxPoolSize=20&w=majority"},
		{"tcp@user@127.0.0.1:27017", "mongodb://user:pass@127.0.0.1:27017/?maxPoolSize=20&w=majority"},
		{"tcp@myDBReader@mongodb0.example.com:27017",
			"mongodb://myDBReader:D1fficultP%40ssw0rd@mongodb0.example.com:27017/?authSource=admin"},

		//Replica set
		{"tcp@mongodb0.example.com:27017,mongodb1.example.com:27017,mongodb2.example.com:27017",
			"mongodb://mongodb0.example.com:27017,mongodb1.example.com:27017,mongodb2.example.com:27017/?replicaSet=myRepl"},
		{"tcp@myDBReader@mongodb0.example.com:27017,mongodb1.example.com:27017,mongodb2.example.com:27017",
			"mongodb://myDBReader:D1fficultP%40ssw0rd@mongodb0.example.com:27017,mongodb1.example.com:27017,mongodb2.example.com:27017/?authSource=admin&replicaSet=myRepl"},
	}

	for _, testCase := range testTable {
		expected := testCase[0]
		actual := whatapmongo.NewURI(testCase[1]).ToString()

		assert.Equal(expected, actual)
	}
}

type TestData struct {
	Name string
	Age  int
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

//테스트 환경 - mongodb: localhost:27017, go-agent: localhost:6600
//CAUTION: 아직 자동화 된 테스트가 아니므로 trace 라이브러리와 sql 라이브러리의 debug output을 확인할 것
func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	assert := assert.New(t)
	opts := options.Client().ApplyURI("mongodb://localhost:27017")

	client, err := whatapmongo.Connect(context.TODO(), opts)
	defer func() {
		err := client.Disconnect(context.TODO())
		assert.Nil(err)
	}()
	assert.Nil(err)
	assert.NotNil(client)

	collection := client.Database("testDatabase").Collection("testCollection")
	assert.NotNil(client)

	t.Run("client connecting", func(t *testing.T) {
		opts := options.Client().ApplyURI("mongodb://localhost:21017")
		c, err := whatapmongo.NewClient(opts)
		assert.Nil(err)
		assert.NotNil(c)

		ctx := context.TODO()

		c, err = whatapmongo.Connect(ctx, opts)
		assert.Nil(err)
		assert.NotNil(c)

		db := c.Database("skynet")
		assert.NotNil(db)
		assert.Equal("skynet", db.Name())
		coll := db.Collection("skynet")
		assert.NotNil(coll)

		assert.Equal(coll.Database(), db)
	},
	)

	t.Run("Clone test", func(t *testing.T) {
		clone, err := collection.Clone()
		assert.Nil(err)
		assert.NotNil(clone)
		assert.Equal(collection.Database().URI, clone.Database().URI)

	},
	)

	luke := TestData{"Luke", 20}
	whatapConfig := map[string]string{
		"net_udp_port":             "6600",
		"debug":                    "true",
		"profile_sql_aram_enabled": "true",
	}
	trace.Init(whatapConfig)
	defer trace.Shutdown()

	//uri := whatapmongo.FormatURI("mongodb://localhost:27017") + "/" + "test-" + randSeq(7)
	base := whatapmongo.NewURI("mongodb://localhost:27017")
	uri := base.Appended("test-" + randSeq(7))

	t.Run("InsertOne test", func(t *testing.T) {
		ctx, err := trace.Start(context.TODO(), uri.Appended("InsertOne").ToString())
		assert.Nil(err)
		insertResult, err := collection.InsertOne(ctx, luke)
		assert.Nil(err)
		assert.NotNil(insertResult)
		assert.NotNil(ctx.Value("whatap"))
		trace.End(ctx, err)
	},
	)

	t.Run("ReplaceOne test", func(t *testing.T) {
		ctx, err := trace.Start(context.TODO(), uri.Appended("ReplaceOne").ToString())
		assert.Nil(err)
		replaceOneResult, err := collection.ReplaceOne(ctx, luke, TestData{Name: "luke", Age: 50})
		assert.Nil(err)
		assert.NotNil(replaceOneResult)
		trace.End(ctx, err)
	},
	)

	t.Run("DeleteOne test", func(t *testing.T) {
		ctx, err := trace.Start(context.TODO(), uri.Appended("DeleteOne").ToString())
		assert.Nil(err)
		deleteResult, err := collection.DeleteOne(ctx, luke)
		assert.Nil(err)
		assert.NotNil(deleteResult)
		trace.End(ctx, err)
	},
	)

	yoda := TestData{"yoda", 100}
	han := TestData{"han", 25}

	t.Run("InsertMany test", func(t *testing.T) {
		docs := []interface{}{luke, yoda, han}
		ctx, err := trace.Start(context.TODO(), uri.Appended("InsertMany").ToString())
		assert.Nil(err)
		insertManyResult, err := collection.InsertMany(ctx, docs)
		assert.Nil(err)
		assert.NotNil(insertManyResult)
		trace.End(ctx, nil)
	},
	)

	t.Run("Distinct test", func(t *testing.T) {
		ctx, err := trace.Start(context.TODO(), uri.Appended("Distinct").ToString())
		assert.Nil(err)
		distinctResult, err := collection.Distinct(ctx, "Name", bson.M{}, nil)
		assert.Nil(err)
		assert.NotNil(distinctResult)
		trace.End(ctx, err)
	},
	)

	t.Run("Count document test", func(t *testing.T) {
		ctx, err := trace.Start(context.TODO(), uri.Appended("CountDocument").ToString())
		assert.Nil(err)
		distinctResult, err := collection.Distinct(ctx, "Name", bson.M{}, nil)
		assert.Nil(err)
		assert.NotNil(distinctResult)
		count, err := collection.CountDocuments(ctx, yoda)
		assert.Nil(err)
		assert.Equal(int64(1), count)
		//_, err = collection.EstimatedDocumentCount(ctx)
		//assert.Nil(err)
		trace.End(ctx, err)
	},
	)

	t.Run("Find operations test", func(t *testing.T) {
		ctx, err := trace.Start(context.TODO(), uri.Appended("Find").ToString())
		assert.Nil(err)
		findResult, err := collection.Find(ctx, luke)
		assert.Nil(err)
		assert.NotNil(findResult)

		findOneResult := collection.FindOne(ctx, luke)
		assert.NotNil(findOneResult)

		LUKE := TestData{"LUKE", 20}

		findOneAndReplace := collection.FindOneAndReplace(ctx, luke, LUKE)
		assert.NotNil(findOneAndReplace)

		findOneAndUpdate := collection.FindOneAndUpdate(ctx, LUKE, luke)
		assert.NotNil(findOneAndUpdate)

		findOneAndDelete := collection.FindOneAndDelete(ctx, luke)
		assert.NotNil(findOneAndDelete)

		trace.End(ctx, nil)
	},
	)

	t.Run("Update and Replace test", func(t *testing.T) {
		YODA := TestData{"YODA", 100}
		ctx, err := trace.Start(context.TODO(), uri.Appended("UpdateAndReplace").ToString())
		assert.Nil(err)
		replaceOneResult, err := collection.ReplaceOne(ctx, yoda, YODA)
		assert.Nil(err)
		assert.NotNil(replaceOneResult)

		filter := bson.D{primitive.E{Key: "_id", Value: "yoda"}}
		update := bson.D{primitive.E{Key: "$set", Value: bson.D{{Key: "Age", Value: 1}}}}
		byIDResult, err := collection.UpdateByID(ctx, filter, update)
		assert.Nil(err)
		assert.NotNil(byIDResult)

		updateOneResult, err := collection.UpdateOne(ctx, filter, update)
		assert.Nil(err)
		assert.NotNil(updateOneResult)

		updateManyResult, err := collection.UpdateMany(ctx, filter, update)
		assert.Nil(err)
		assert.NotNil(updateManyResult)

		trace.End(ctx, nil)
	},
	)

	t.Run("BulkWrite", func(t *testing.T) {
		var firstID, secondID primitive.ObjectID
		firstUpdate := bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "email", Value: "firstEmail@example.com"},
			}},
		}
		secondUpdate := bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "email", Value: "secondEmail@example.com"},
			}},
		}
		models := []mongo.WriteModel{
			mongo.NewUpdateOneModel().SetFilter(bson.D{{Key: "_id", Value: firstID}}).
				SetUpdate(firstUpdate).SetUpsert(true),
			mongo.NewUpdateOneModel().SetFilter(bson.D{{Key: "_id", Value: secondID}}).
				SetUpdate(secondUpdate).SetUpsert(true),
		}
		opts := options.BulkWrite().SetOrdered(false)

		ctx, err := trace.Start(context.TODO(), uri.Appended("BulkWrite").ToString())
		assert.Nil(err)
		result, err := collection.BulkWrite(ctx, models, opts)
		assert.Nil(err)

		assert.Equal(int64(1), result.MatchedCount)
		assert.Equal(int64(1), result.ModifiedCount)
		assert.Equal(int64(0), result.DeletedCount)
		assert.Equal(int64(1), result.UpsertedCount)
		trace.End(ctx, nil)
	},
	)

	t.Run("Aggregate", func(t *testing.T) {
		ctx, err := trace.Start(context.TODO(), uri.Appended("Aggregate").ToString())
		assert.Nil(err)
		matchStage := bson.D{{Key: "$match", Value: bson.D{{Key: "podcast", Value: "id"}}}}
		groupStage := bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$podcast"}, {Key: "total", Value: bson.D{{Key: "$sum", Value: "$duration"}}}}}}

		cursor, err := collection.Aggregate(ctx, mongo.Pipeline{matchStage, groupStage})
		assert.Nil(err)
		assert.NotNil(cursor)
		trace.End(ctx, nil)
	},
	)

	t.Run("DeleteMany", func(t *testing.T) {
		ctx, err := trace.Start(context.TODO(), uri.Appended("DeleteMany").ToString())
		assert.Nil(err)
		defer trace.End(ctx, err)

		//cleanup all documents
		deleteManyResult, err := collection.DeleteMany(context.TODO(), bson.D{})
		assert.Nil(err)
		assert.NotNil(deleteManyResult)
	},
	)

	ctx, err := trace.Start(context.TODO(), uri.Appended("Drop").ToString())
	assert.Nil(err)
	defer trace.End(ctx, err)
	err = collection.Drop(ctx)
	assert.Nil(err)

}
