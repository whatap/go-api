package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	whatapmongo "github.com/whatap/go-api/instrumentation/github.com/mongodb/mongo-go-driver"
	"github.com/whatap/go-api/trace"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TestData struct {
	Name string
	Age  int
}

func main() {
	dsn := "mongodb://localhost"
	udpPort := "6600"
	port := "9090"

	whatapConfig := make(map[string]string)
	whatapConfig["net_udp_port"] = fmt.Sprintf("%d", udpPort)
	whatapConfig["debug"] = "true"
	whatapConfig["profile_sql_param_enabled"] = "true"

	trace.Init(whatapConfig)
	defer trace.Shutdown()

	ctx := context.TODO()
	client, err := whatapmongo.Connect(ctx, options.Client().ApplyURI(dsn))
	if err != nil {
		log.Panic(err)
	}
	log.Info("start")

	http.HandleFunc("/InsertOne", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Trace Start")
		ctx, err := trace.StartWithRequest(r)
		defer func() {
			time.Sleep(time.Second)
			fmt.Println("Trace End")
			trace.End(ctx, nil)
		}()

		coll := client.Database("whatap").Collection("mongo")
		doc := bson.D{{"title", "Record of a Shriveled Datum"}, {"text", "No bytes, no problem. Just insert a document, in MongoDB"}}

		_, err = coll.InsertOne(ctx, doc)

		if err != nil {
			log.Error(err)
			trace.Error(ctx, err)
			return
		}
	})

	http.HandleFunc("/InsertMany", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Trace Start")
		ctx, err := trace.StartWithRequest(r)
		defer func() {
			time.Sleep(time.Second)
			fmt.Println("Trace End")
			trace.End(ctx, nil)
		}()

		coll := client.Database("whatap").Collection("mongo")
		doc := bson.D{{"title", "Record of a Shriveled Datum"}, {"text", "No bytes, no problem. Just insert a document, in MongoDB"}}

		_, err = coll.InsertMany(ctx, []interface{}{doc})

		if err != nil {
			log.Error(err)
			trace.Error(ctx, err)
			return
		}
	})

	http.HandleFunc("/DeleteOne", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Trace Start")
		ctx, err := trace.StartWithRequest(r)
		defer func() {
			time.Sleep(time.Second)
			fmt.Println("Trace End")
			trace.End(ctx, nil)
		}()

		coll := client.Database("whatap").Collection("mongo")
		doc := bson.D{{"title", "Record of a Shriveled Datum"}, {"text", "No bytes, no problem. Just insert a document, in MongoDB"}}

		_, err = coll.DeleteOne(ctx, doc)

		time.Sleep(time.Second)

		if err != nil {
			log.Error(err)
			trace.Error(ctx, err)
			return
		}
	})

	http.HandleFunc("/DeleteMany", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Trace Start")
		ctx, err := trace.StartWithRequest(r)
		defer func() {
			time.Sleep(time.Second)
			fmt.Println("Trace End")
			trace.End(ctx, nil)
		}()

		coll := client.Database("whatap").Collection("mongo")
		doc := bson.D{{"title", "Record of a Shriveled Datum"}, {"text", "No bytes, no problem. Just insert a document, in MongoDB"}}

		_, err = coll.DeleteMany(ctx, doc)

		if err != nil {
			log.Error(err)
			trace.Error(ctx, err)
			return
		}
	})

	http.HandleFunc("/Find", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Trace Start")
		luke := TestData{"luke", 20}
		ctx, err := trace.StartWithRequest(r)
		defer func() {
			time.Sleep(time.Second)
			fmt.Println("Trace End")
			trace.End(ctx, nil)
		}()

		coll := client.Database("whatap").Collection("mongo")
		_, err = coll.Find(ctx, luke)
		if err != nil {
			log.Error(err)
			trace.Error(ctx, err)
			return
		}

		_ = coll.FindOne(ctx, luke)

		LUKE := TestData{"LUKE", 20}

		_ = coll.FindOneAndReplace(ctx, luke, LUKE)
		_ = coll.FindOneAndUpdate(ctx, LUKE, luke)
		_ = coll.FindOneAndDelete(ctx, luke)

		trace.End(ctx, nil)

		if err != nil {
			log.Error(err)
			trace.Error(ctx, err)
			return
		}
	})

	http.HandleFunc("/UpdateAndReplace", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Trace Start")
		luke := TestData{"luke", 20}
		ctx, err := trace.StartWithRequest(r)
		defer func() {
			time.Sleep(time.Second)
			fmt.Println("Trace End")
			trace.End(ctx, nil)
		}()

		coll := client.Database("whatap").Collection("mongo")
		_, err = coll.ReplaceOne(ctx, luke, TestData{"LUKE", 25})
		if err != nil {
			log.Error(err)
			trace.Error(ctx, err)
			return
		}
		filter := bson.D{{"_id", "yoda"}}
		update := bson.D{{"$set", bson.D{{"Age", 1}}}}

		_, err = coll.UpdateByID(ctx, filter, update)
		if err != nil {
			log.Error(err)
			trace.Error(ctx, err)
			return
		}

		_, err = coll.UpdateOne(ctx, filter, update)
		if err != nil {
			log.Error(err)
			trace.Error(ctx, err)
			return
		}

		_, err = coll.UpdateMany(ctx, filter, update)
		if err != nil {
			log.Error(err)
			trace.Error(ctx, err)
			return
		}

		trace.End(ctx, nil)
	})

	//request 를 사용하지 않는 예제
	func() {
		ctx := context.TODO()
		trace.Start(ctx, whatapmongo.NewURI(dsn).Appended("no-request").ToString())
		defer trace.End(ctx, nil)
		coll := client.Database("whatap").Collection("mongo")
		doc := bson.D{{"trace", "no-request"}}

		_, err = coll.InsertOne(ctx, doc)

		if err != nil {
			log.Error(err)
			trace.Error(ctx, err)
			return
		}
	}()

	_ = http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}
