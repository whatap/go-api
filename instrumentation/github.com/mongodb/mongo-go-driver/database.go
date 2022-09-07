package whatapmongo

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

//interface for dependency injection
//TODO: API breaking change 대비
type MongoDatabase interface {
	CollectionApi
	CommandRunner
	AdminApi
	Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (*mongo.Cursor, error)
	Client() *mongo.Client
}

type CollectionApi interface {
	Collection(name string, opts ...*options.CollectionOptions) *mongo.Collection
	CreateCollection(ctx context.Context, name string,
		opts ...*options.CreateCollectionOptions) error
	ListCollectionNames(ctx context.Context, filter interface{},
		opts ...*options.ListCollectionsOptions) ([]string, error)
	ListCollectionSpecifications(ctx context.Context, filter interface{},
		opts ...*options.ListCollectionsOptions) ([]*mongo.CollectionSpecification, error)
	ListCollections(ctx context.Context, filter interface{},
		opts ...*options.ListCollectionsOptions) (*mongo.Cursor, error)
	Name() string
}

type CommandRunner interface {
	RunCommand(ctx context.Context, runCommand interface{}, opts ...*options.RunCmdOptions) *mongo.SingleResult
	RunCommandCursor(ctx context.Context, runCommand interface{}, opts ...*options.RunCmdOptions) (*mongo.Cursor, error)
}

type AdminApi interface {
	ReadConcern() *readconcern.ReadConcern
	WriteConcern() *writeconcern.WriteConcern
	CreateView(ctx context.Context, viewName, viewOn string, pipeline interface{}, opts ...*options.CreateViewOptions) error
	Drop(ctx context.Context) error
	ReadPreference() *readpref.ReadPref
	Watch(ctx context.Context, pipeline interface{}, opts ...*options.ChangeStreamOptions) (*mongo.ChangeStream, error)
}

func defaultCollectionFactory(wdb *WrappedDatabase, name string, opts ...*options.CollectionOptions) *WrappedCollection {
	return &WrappedCollection{
		MongoCollection: wdb.MongoDatabase.Collection(name, opts...),
		WrappedDB:       wdb,
	}
}

type WrappedDatabase struct {
	MongoDatabase
	URI URI
	//dependency injection
	CollectionFactory func(wdb *WrappedDatabase, name string, opts ...*options.CollectionOptions) *WrappedCollection
	client            *WrappedClient
}

func (db *WrappedDatabase) Collection(name string,
	opts ...*options.CollectionOptions) *WrappedCollection {

	return db.CollectionFactory(db, name, opts...)
}

func (db *WrappedDatabase) Client() *WrappedClient {
	return db.client
}
