package whatapmongo

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

//interface for dependency injection
//TODO: API breaking change 대비
type MongoCollection interface {
	Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (*mongo.Cursor, error)
	BulkWrite(ctx context.Context, models []mongo.WriteModel, opts ...*options.BulkWriteOptions) (*mongo.BulkWriteResult, error)
	Clone(opts ...*options.CollectionOptions) (*mongo.Collection, error)
	CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error)
	Database() *mongo.Database
	DeleteMany(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error)
	DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error)
	Distinct(ctx context.Context, fieldName string, filter interface{}, opts ...*options.DistinctOptions) ([]interface{}, error)
	Drop(ctx context.Context) error
	EstimatedDocumentCount(ctx context.Context, opts ...*options.EstimatedDocumentCountOptions) (int64, error)
	Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (cur *mongo.Cursor, err error)
	FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult
	FindOneAndDelete(ctx context.Context, filter interface{}, opts ...*options.FindOneAndDeleteOptions) *mongo.SingleResult
	FindOneAndReplace(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.FindOneAndReplaceOptions) *mongo.SingleResult
	FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) *mongo.SingleResult
	Indexes() mongo.IndexView
	InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error)
	InsertMany(ctx context.Context, documents []interface{}, opts ...*options.InsertManyOptions) (*mongo.InsertManyResult, error)
	Name() string
	ReplaceOne(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.ReplaceOptions) (*mongo.UpdateResult, error)
	UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error)
	UpdateByID(ctx context.Context, id interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error)
	UpdateMany(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error)
	Watch(ctx context.Context, pipeline interface{}, opts ...*options.ChangeStreamOptions) (*mongo.ChangeStream, error)
}

type WrappedCollection struct {
	MongoCollection
	WrappedDB *WrappedDatabase
}

func (coll *WrappedCollection) Database() *WrappedDatabase {
	return coll.WrappedDB
}

func (coll *WrappedCollection) Clone(opts ...*options.CollectionOptions) (*WrappedCollection, error) {
	clone, err := coll.MongoCollection.Clone(opts...)
	if err != nil {
		return nil, err
	}
	return &WrappedCollection{
		MongoCollection: clone,
		WrappedDB: &WrappedDatabase{
			MongoDatabase:     clone.Database(),
			URI:               coll.WrappedDB.URI,
			CollectionFactory: defaultCollectionFactory,
			client:            coll.WrappedDB.client,
		},
	}, nil
}

func (coll *WrappedCollection) Aggregate(ctx context.Context,
	pipeline interface{}, opts ...*options.AggregateOptions) (*mongo.Cursor, error) {

	cmd := Aggregate{
		Pipeline: pipeline,
		Options:  opts,
	}
	tracer := NewTracer(cmd, coll)
	return tracer.RunAndTraceCursorCommand(ctx)
}

func (coll *WrappedCollection) BulkWrite(ctx context.Context, models []mongo.WriteModel,
	opts ...*options.BulkWriteOptions) (*mongo.BulkWriteResult, error) {
	cmd := BulkWrite{
		Models:  models,
		Options: opts,
	}
	tracer := NewTracer(cmd, coll)
	return tracer.RunAndTraceBulkWrite(ctx)
}

func (coll *WrappedCollection) CountDocuments(ctx context.Context, filter interface{},
	opts ...*options.CountOptions) (int64, error) {
	cmd := CountDocuments{
		Filter:  filter,
		Options: opts,
	}
	tracer := NewTracer(cmd, coll)
	return tracer.RunAndTraceCountDocuments(ctx)
}

func (coll *WrappedCollection) Distinct(ctx context.Context, fieldName string, filter interface{},
	opts ...*options.DistinctOptions) ([]interface{}, error) {

	cmd := Distinct{
		Filter:    filter,
		FieldName: fieldName,
		Options:   opts,
	}

	tracer := NewTracer(cmd, coll)
	return tracer.RunAndTraceDistinct(ctx)
}

func (coll *WrappedCollection) Find(ctx context.Context,
	filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {

	cmd := Find{
		Filter:  filter,
		Options: opts,
	}

	tracer := NewTracer(cmd, coll)
	return tracer.RunAndTraceCursorCommand(ctx)
}

func (coll *WrappedCollection) FindOne(ctx context.Context,
	filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult {

	cmd := FindOne{
		Filter:  filter,
		Options: opts,
	}

	tracer := NewTracer(cmd, coll)
	return tracer.RunAndTraceSingleResultCommand(ctx)
}

func (coll *WrappedCollection) FindOneAndUpdate(ctx context.Context,
	filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) *mongo.SingleResult {

	cmd := FindOneAndUpdate{
		Filter:  filter,
		Update:  update,
		Options: opts,
	}

	tracer := NewTracer(cmd, coll)
	return tracer.RunAndTraceSingleResultCommand(ctx)
}

func (coll *WrappedCollection) FindOneAndReplace(ctx context.Context,
	filter interface{}, replace interface{}, opts ...*options.FindOneAndReplaceOptions) *mongo.SingleResult {

	cmd := FindOneAndReplace{
		Filter:  filter,
		Replace: replace,
		Options: opts,
	}

	tracer := NewTracer(cmd, coll)
	return tracer.RunAndTraceSingleResultCommand(ctx)
}

func (coll *WrappedCollection) FindOneAndDelete(ctx context.Context,
	filter interface{}, opts ...*options.FindOneAndDeleteOptions) *mongo.SingleResult {

	cmd := FindOneAndDelete{
		Filter:  filter,
		Options: opts,
	}

	tracer := NewTracer(cmd, coll)
	return tracer.RunAndTraceSingleResultCommand(ctx)
}

func (coll *WrappedCollection) InsertOne(ctx context.Context,
	document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {

	cmd := InsertOne{
		Document: document,
		Options:  opts,
	}

	tracer := NewTracer(cmd, coll)
	return tracer.RunAndTraceInsertOne(ctx)
}

func (coll *WrappedCollection) InsertMany(ctx context.Context,
	documents []interface{}, opts ...*options.InsertManyOptions) (*mongo.InsertManyResult, error) {

	cmd := InsertMany{
		Documents: documents,
		Options:   opts,
	}

	tracer := NewTracer(cmd, coll)
	return tracer.RunAndTraceInsertMany(ctx)
}

func (coll *WrappedCollection) ReplaceOne(ctx context.Context, filter, replacement interface{},
	opts ...*options.ReplaceOptions) (*mongo.UpdateResult, error) {
	cmd := ReplaceOne{
		Filter:      filter,
		Replacement: replacement,
		Options:     opts,
	}

	tracer := NewTracer(cmd, coll)
	return tracer.RunAndTraceUpdateCommand(ctx)
}

func (coll *WrappedCollection) UpdateByID(ctx context.Context, id interface{}, update interface{},
	opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	cmd := UpdateByID{
		ID:      id,
		Update:  update,
		Options: opts,
	}

	tracer := NewTracer(cmd, coll)
	return tracer.RunAndTraceUpdateCommand(ctx)
}

func (coll *WrappedCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{},
	opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	cmd := UpdateOne{
		Filter:  filter,
		Update:  update,
		Options: opts,
	}

	tracer := NewTracer(cmd, coll)
	return tracer.RunAndTraceUpdateCommand(ctx)
}

func (coll *WrappedCollection) UpdateMany(ctx context.Context, filter interface{}, update interface{},
	opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	cmd := UpdateMany{
		Filter:  filter,
		Update:  update,
		Options: opts,
	}

	tracer := NewTracer(cmd, coll)
	return tracer.RunAndTraceUpdateCommand(ctx)
}

func (coll *WrappedCollection) DeleteOne(ctx context.Context,
	filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {

	cmd := DeleteOne{
		Filter:  filter,
		Options: opts,
	}

	tracer := NewTracer(cmd, coll)
	return tracer.RunAndTraceDeleteCommand(ctx)
}

func (coll *WrappedCollection) DeleteMany(ctx context.Context,
	filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {

	cmd := DeleteMany{
		Filter:  filter,
		Options: opts,
	}

	tracer := NewTracer(cmd, coll)
	return tracer.RunAndTraceDeleteCommand(ctx)
}

func (coll *WrappedCollection) Drop(ctx context.Context) error {
	return NewTracer(Drop{}, coll).RunAndTraceDrop(ctx)
}
