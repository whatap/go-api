package whatapmongo

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

//unit 테스트용 mock 객체들

const (
	StubSessions = 17
)

type MockClient struct {
}

func (mock MockClient) Connect(ctx context.Context) error {
	return nil
}

func (mock MockClient) Disconnect(ctx context.Context) error {
	return nil
}

func (mock MockClient) Ping(ctx context.Context, rp *readpref.ReadPref) error {
	return nil
}

func (mock MockClient) Database(name string, opts ...*options.DatabaseOptions) *mongo.Database {
	return &mongo.Database{}
}

func (mock MockClient) ListDatabases(ctx context.Context, filter interface{},
	opts ...*options.ListDatabasesOptions) (mongo.ListDatabasesResult, error) {
	return mongo.ListDatabasesResult{}, nil
}

func (mock MockClient) ListDatabaseNames(ctx context.Context, filter interface{}, opts ...*options.ListDatabasesOptions) ([]string, error) {
	return []string{}, nil
}

func (mock MockClient) NumberSessionsInProgress() int {
	return StubSessions
}

func (mock MockClient) UseSession(ctx context.Context, fn func(mongo.SessionContext) error) error {
	return nil
}

func (mock MockClient) UseSessionWithOptions(ctx context.Context,
	opts *options.SessionOptions, fn func(mongo.SessionContext) error) error {
	return nil
}

func (mock MockClient) Watch(ctx context.Context, pipeline interface{},
	opts ...*options.ChangeStreamOptions) (*mongo.ChangeStream, error) {
	return nil, nil
}

type MockOptions struct {
}

func (mock MockOptions) GetURI() string {
	return "mongodb://localhost:21017"
}

const (
	StubCount    = int64(5)
	StubEstimate = int64(7)
	StubInserted = int64(11)
	StubDeleted  = int64(1)
	StubMatched  = int64(150)
	StubModified = int64(228)
	StubUpserted = int64(227)
	StubName     = "stub"
)

type StubCollection struct {
}

func (coll StubCollection) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (*mongo.Cursor, error) {
	return nil, nil
}
func (coll StubCollection) BulkWrite(ctx context.Context, models []mongo.WriteModel, opts ...*options.BulkWriteOptions) (*mongo.BulkWriteResult, error) {
	return &mongo.BulkWriteResult{
		InsertedCount: StubInserted,
		MatchedCount:  StubMatched,
		ModifiedCount: StubModified,
		DeletedCount:  StubDeleted,
		UpsertedCount: StubUpserted,
	}, nil
}
func (coll StubCollection) Clone(opts ...*options.CollectionOptions) (*mongo.Collection, error) {
	return nil, nil
}
func (coll StubCollection) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	return StubCount, nil
}
func (coll StubCollection) Database() *mongo.Database {
	return nil
}
func (coll StubCollection) DeleteMany(ctx context.Context, filter interface{},
	opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	return &mongo.DeleteResult{
		DeletedCount: StubDeleted,
	}, nil
}
func (coll StubCollection) DeleteOne(ctx context.Context, filter interface{},
	opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	return &mongo.DeleteResult{
		DeletedCount: StubDeleted,
	}, nil

}
func (coll StubCollection) Distinct(ctx context.Context, fieldName string, filter interface{}, opts ...*options.DistinctOptions) ([]interface{}, error) {
	return nil, nil
}
func (coll StubCollection) Drop(ctx context.Context) error {
	return nil
}

func (coll StubCollection) EstimatedDocumentCount(ctx context.Context, opts ...*options.EstimatedDocumentCountOptions) (int64, error) {
	return StubEstimate, nil
}
func (coll StubCollection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (cur *mongo.Cursor, err error) {
	return nil, nil
}
func (coll StubCollection) FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult {
	return nil
}
func (coll StubCollection) FindOneAndDelete(ctx context.Context, filter interface{}, opts ...*options.FindOneAndDeleteOptions) *mongo.SingleResult {
	return nil
}
func (coll StubCollection) FindOneAndReplace(ctx context.Context, filter interface{}, replace interface{}, opts ...*options.FindOneAndReplaceOptions) *mongo.SingleResult {
	return nil
}
func (coll StubCollection) FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) *mongo.SingleResult {
	return nil
}
func (coll StubCollection) Indexes() mongo.IndexView {
	return mongo.IndexView{}
}
func (coll StubCollection) InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	return nil, nil
}
func (coll StubCollection) InsertMany(ctx context.Context, documents []interface{}, opts ...*options.InsertManyOptions) (*mongo.InsertManyResult, error) {
	return nil, nil
}
func (coll StubCollection) Name() string {
	return StubName
}
func (coll StubCollection) ReplaceOne(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.ReplaceOptions) (*mongo.UpdateResult, error) {
	return &mongo.UpdateResult{
		MatchedCount:  StubMatched,
		ModifiedCount: StubModified,
		UpsertedCount: StubUpserted,
	}, nil
}
func (coll StubCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	return &mongo.UpdateResult{
		MatchedCount:  StubMatched,
		ModifiedCount: StubModified,
		UpsertedCount: StubUpserted,
	}, nil
}
func (coll StubCollection) UpdateByID(ctx context.Context, id interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	return &mongo.UpdateResult{
		MatchedCount:  StubMatched,
		ModifiedCount: StubModified,
		UpsertedCount: StubUpserted,
	}, nil
}
func (coll StubCollection) UpdateMany(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	return &mongo.UpdateResult{
		MatchedCount:  StubMatched,
		ModifiedCount: StubModified,
		UpsertedCount: StubUpserted,
	}, nil
}
func (coll StubCollection) Watch(ctx context.Context, pipeline interface{}, opts ...*options.ChangeStreamOptions) (*mongo.ChangeStream, error) {
	return nil, nil
}
