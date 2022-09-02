package whatapmongo

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	QueryNameAggregate         = "Aggregate"
	QueryNameBulkWrite         = "BulkWrite"
	QueryNameInsertOne         = "InsertOne"
	QueryNameInsertMany        = "InsertMany"
	QueryNameDeleteOne         = "DeleteOne"
	QueryNameDeleteMany        = "DeleteMany"
	QueryNameCountDocuments    = "CountDocuments"
	QueryNameDistinct          = "Distinct"
	QueryNameFind              = "Find"
	QueryNameFindOne           = "FindOne"
	QueryNameFindOneAndUpdate  = "FindOneAndUpdate"
	QueryNameFindOneAndReplace = "FindOneAndReplace"
	QueryNameFindOneAndDelete  = "FindOneAndDelete"
	QueryNameReplaceOne        = "ReplaceOne"
	QueryNameUpdateByID        = "UpdateByID"
	QueryNameUpdateOne         = "UpdateOne"
	QueryNameUpdateMany        = "UpdateMany"
	QueryNameDrop              = "Drop"
)

type Command interface {
	Run(ctx context.Context, coll MongoCollection) (result interface{}, err error)
	Name() string
}

type Aggregate struct {
	Pipeline interface{}
	Options  []*options.AggregateOptions
}

func (aggregate Aggregate) Run(ctx context.Context, coll MongoCollection) (interface{}, error) {
	return coll.Aggregate(ctx,
		aggregate.Pipeline, aggregate.Options...)
}

func (aggregate Aggregate) Name() string {
	return QueryNameAggregate
}

type BulkWrite struct {
	Models  []mongo.WriteModel
	Options []*options.BulkWriteOptions
}

func (bulkWrite BulkWrite) Run(ctx context.Context, coll MongoCollection) (interface{}, error) {
	return coll.BulkWrite(ctx, bulkWrite.Models)
}

func (bulkWrite BulkWrite) Name() string {
	return QueryNameBulkWrite
}

type CountDocuments struct {
	Filter  interface{}
	Options []*options.CountOptions
}

func (count CountDocuments) Run(ctx context.Context, coll MongoCollection) (interface{}, error) {
	return coll.CountDocuments(ctx, count.Filter, count.Options...)
}

func (count CountDocuments) Name() string {
	return QueryNameCountDocuments
}

type Distinct struct {
	Filter    interface{}
	FieldName string
	Options   []*options.DistinctOptions
}

func (distinct Distinct) Run(ctx context.Context, coll MongoCollection) (interface{}, error) {
	return coll.Distinct(ctx, distinct.FieldName, distinct.Filter,
		distinct.Options...)
}

func (distinct Distinct) Name() string {
	return QueryNameDistinct
}

type Find struct {
	Filter  interface{}
	Options []*options.FindOptions
}

func (find Find) Run(ctx context.Context, coll MongoCollection) (interface{}, error) {
	return coll.Find(ctx, find.Filter,
		find.Options...)
}

func (find Find) Name() string {
	return QueryNameFind
}

type FindOne struct {
	Filter  interface{}
	Options []*options.FindOneOptions
}

func (findOne FindOne) Run(ctx context.Context, coll MongoCollection) (interface{}, error) {
	return coll.FindOne(ctx, findOne.Filter,
		findOne.Options...), nil
}

func (findOne FindOne) Name() string {
	return QueryNameFindOne
}

type FindOneAndUpdate struct {
	Filter  interface{}
	Update  interface{}
	Options []*options.FindOneAndUpdateOptions
}

func (findOneAndUpdate FindOneAndUpdate) Run(ctx context.Context, coll MongoCollection) (interface{}, error) {
	return coll.FindOneAndUpdate(ctx, findOneAndUpdate.Filter, findOneAndUpdate.Update,
		findOneAndUpdate.Options...), nil
}

func (findOneAndUpdate FindOneAndUpdate) Name() string {
	return QueryNameFindOneAndUpdate
}

type FindOneAndReplace struct {
	Filter  interface{}
	Replace interface{}
	Options []*options.FindOneAndReplaceOptions
}

func (findOneAndReplace FindOneAndReplace) Run(ctx context.Context, coll MongoCollection) (interface{}, error) {
	return coll.FindOneAndReplace(ctx, findOneAndReplace.Filter, findOneAndReplace.Replace,
		findOneAndReplace.Options...), nil
}

func (findOneAndReplace FindOneAndReplace) Name() string {
	return QueryNameFindOneAndReplace
}

type FindOneAndDelete struct {
	Filter  interface{}
	Delete  interface{}
	Options []*options.FindOneAndDeleteOptions
}

func (findOneAndDelete FindOneAndDelete) Run(ctx context.Context, coll MongoCollection) (interface{}, error) {
	return coll.FindOneAndDelete(ctx, findOneAndDelete.Filter,
		findOneAndDelete.Options...), nil
}

func (findOneAndDelete FindOneAndDelete) Name() string {
	return QueryNameFindOneAndDelete
}

type InsertOne struct {
	Document interface{}
	Options  []*options.InsertOneOptions
}

func (insertOne InsertOne) Run(ctx context.Context, coll MongoCollection) (interface{}, error) {
	return coll.InsertOne(ctx, insertOne.Document,
		insertOne.Options...)
}

func (insertOne InsertOne) Name() string {
	return QueryNameInsertOne
}

type InsertMany struct {
	Documents []interface{}
	Options   []*options.InsertManyOptions
}

func (insertMany InsertMany) Run(ctx context.Context, coll MongoCollection) (interface{}, error) {
	return coll.InsertMany(ctx, insertMany.Documents,
		insertMany.Options...)
}

func (insertMany InsertMany) Name() string {
	return QueryNameInsertMany
}

type ReplaceOne struct {
	Filter      interface{}
	Replacement interface{}
	Options     []*options.ReplaceOptions
}

func (replaceOne ReplaceOne) Run(ctx context.Context, coll MongoCollection) (interface{}, error) {
	return coll.ReplaceOne(ctx, replaceOne.Filter, replaceOne.Replacement,
		replaceOne.Options...)
}

func (replaceOne ReplaceOne) Name() string {
	return QueryNameReplaceOne
}

type UpdateByID struct {
	ID      interface{}
	Update  interface{}
	Options []*options.UpdateOptions
}

func (updateByID UpdateByID) Run(ctx context.Context, coll MongoCollection) (interface{}, error) {
	return coll.UpdateByID(ctx, updateByID.ID, updateByID.Update,
		updateByID.Options...)
}

func (updateByID UpdateByID) Name() string {
	return QueryNameUpdateByID
}

type UpdateOne struct {
	Filter  interface{}
	Update  interface{}
	Options []*options.UpdateOptions
}

func (updateOne UpdateOne) Run(ctx context.Context, coll MongoCollection) (interface{}, error) {
	return coll.UpdateOne(ctx, updateOne.Filter, updateOne.Update,
		updateOne.Options...)
}

func (updateOne UpdateOne) Name() string {
	return QueryNameUpdateOne
}

type UpdateMany struct {
	Filter  interface{}
	Update  interface{}
	Options []*options.UpdateOptions
}

func (updateMany UpdateMany) Run(ctx context.Context, coll MongoCollection) (interface{}, error) {
	return coll.UpdateMany(ctx, updateMany.Filter, updateMany.Update,
		updateMany.Options...)
}

func (updateMany UpdateMany) Name() string {
	return QueryNameUpdateMany
}

type DeleteOne struct {
	Filter  interface{}
	Options []*options.DeleteOptions
}

func (deleteOne DeleteOne) Run(ctx context.Context, coll MongoCollection) (interface{}, error) {
	return coll.DeleteOne(ctx, deleteOne.Filter,
		deleteOne.Options...)
}

func (deleteOne DeleteOne) Name() string {
	return QueryNameDeleteOne
}

type DeleteMany struct {
	Filter  interface{}
	Options []*options.DeleteOptions
}

func (deleteMany DeleteMany) Run(ctx context.Context, coll MongoCollection) (interface{}, error) {
	return coll.DeleteMany(ctx, deleteMany.Filter,
		deleteMany.Options...)
}

func (deleteMany DeleteMany) Name() string {
	return QueryNameDeleteMany
}

type Drop struct {
	CollectionName string
}

//Drop 명령은 error만 리턴함. Command interface 를 구현하기 위해 nil 리턴
func (drop Drop) Run(ctx context.Context, coll MongoCollection) (interface{}, error) {
	return nil, coll.Drop(ctx)
}

func (drop Drop) Name() string {
	return QueryNameDrop
}
