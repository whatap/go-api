package whatapmongo

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

//interface for dependency injection
//TODO: API breaking change 대비
type MongoClient interface {
	Connect(ctx context.Context) error
	Database(name string, opts ...*options.DatabaseOptions) *mongo.Database
	Disconnect(ctx context.Context) error
	ListDatabases(ctx context.Context, filter interface{}, opts ...*options.ListDatabasesOptions) (mongo.ListDatabasesResult, error)
	ListDatabaseNames(ctx context.Context, filter interface{}, opts ...*options.ListDatabasesOptions) ([]string, error)
	NumberSessionsInProgress() int
	Ping(ctx context.Context, rp *readpref.ReadPref) error
	UseSession(ctx context.Context, fn func(mongo.SessionContext) error) error
	UseSessionWithOptions(ctx context.Context, opts *options.SessionOptions, fn func(mongo.SessionContext) error) error
	Watch(ctx context.Context, pipeline interface{}, opts ...*options.ChangeStreamOptions) (*mongo.ChangeStream, error)
}

type WrappedClient struct {
	MongoClient
	Options MongoClientOptions
}

func Connect(ctx context.Context, opts ...*options.ClientOptions) (*WrappedClient, error) {
	mongoClient, err := mongo.Connect(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &WrappedClient{
		MongoClient: mongoClient,
		Options:     options.MergeClientOptions(opts...),
	}, nil
}

func NewClient(opts ...*options.ClientOptions) (*WrappedClient, error) {
	mongoClient, err := mongo.NewClient(opts...)
	if err != nil {
		return nil, err
	}
	return &WrappedClient{
		MongoClient: mongoClient,
		Options:     options.MergeClientOptions(opts...),
	}, nil
}

func (client *WrappedClient) Database(name string, opts ...*options.DatabaseOptions) *WrappedDatabase {
	internalDB := client.MongoClient.Database(name, opts...)
	wdb := &WrappedDatabase{
		MongoDatabase: internalDB,
		URI:           NewURI(client.Options.GetURI()),
		client:        client,
	}
	wdb.CollectionFactory = defaultCollectionFactory

	return wdb
}
