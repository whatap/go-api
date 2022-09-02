package whatapmongo

import (
	"context"
	"fmt"

	"github.com/whatap/go-api/sql"
	"github.com/whatap/go-api/trace"
	"go.mongodb.org/mongo-driver/mongo"
)

type Tracer struct {
	Command    Command
	Collection MongoCollection
	URI        URI
}

func NewTracer(cmd Command, coll *WrappedCollection) Tracer {
	return Tracer{
		Command:    cmd,
		Collection: coll.MongoCollection,
		URI:        coll.WrappedDB.URI,
	}
}

func traceCtxExist(ctx context.Context) bool {
	return ctx != nil &&
		ctx.Value("whatap") != nil
}

func getTraceCtx(ctx context.Context, uri string) (ctxRaw context.Context, traceStarted bool, err error) {
	if traceCtxExist(ctx) {
		return ctx, false, nil
	}
	ctx, err = trace.Start(ctx, uri)
	if err != nil {
		return ctx, false, err
	}
	return ctx, true, nil
}

func (tracer Tracer) RunAndTraceCommand(ctx context.Context, sqlParams ...interface{}) (interface{}, error) {

	ctx, traceStarted, err := getTraceCtx(ctx, tracer.URI.ToString())
	//사용자의 프로그램을 방해하지 않도록 실제 명령 수행 결과에 오류가 있을 때만 에러 리턴.
	//trace 과정에서의 error는 로깅만 수행
	if err != nil {
		trace.Error(ctx, err)
	}
	if traceStarted {
		defer trace.End(ctx, nil)
	}

	sqlCtx, _ := sql.StartWithParam(ctx, tracer.URI.ToString(), tracer.Command.Name(), sqlParams)

	result, err := tracer.Command.Run(ctx, tracer.Collection)
	if err != nil {
		return result, err
	}

	err = sql.End(sqlCtx, err)
	if err != nil {
		trace.Error(ctx, err)
	}

	return result, nil
}

func (tracer Tracer) RunAndTraceSingleResultCommand(ctx context.Context) *mongo.SingleResult {
	resultRaw, err := tracer.RunAndTraceCommand(ctx)
	if err != nil {
		return nil
	}
	result, typeMatched := resultRaw.(*mongo.SingleResult)
	if !typeMatched {
		return nil
	}
	return result
}

func (tracer Tracer) RunAndTraceUpdateCommand(ctx context.Context) (*mongo.UpdateResult, error) {
	resultRaw, err := tracer.RunAndTraceCommand(ctx)
	if err != nil {
		return nil, err
	}
	result, typeMatched := resultRaw.(*mongo.UpdateResult)
	if !typeMatched {
		return result, fmt.Errorf("invalid Update result type: %t", resultRaw)
	}

	return result, err
}

func (tracer Tracer) RunAndTraceDistinct(ctx context.Context) ([]interface{}, error) {

	distinct, typeMatched := tracer.Command.(Distinct)
	if !typeMatched {
		return nil, fmt.Errorf("invalid Distinct command type: %t", tracer.Command)
	}

	resultRaw, err := tracer.RunAndTraceCommand(ctx, distinct.FieldName)
	if err != nil {
		return nil, err
	}
	result, typeMatched := resultRaw.([]interface{})
	if !typeMatched {
		return result, fmt.Errorf("invalid Distinct result type: %t", resultRaw)
	}

	return result, err

}

func (tracer Tracer) RunAndTraceCountDocuments(ctx context.Context) (int64, error) {
	resultRaw, err := tracer.RunAndTraceCommand(ctx)
	if err != nil {
		return 0, err
	}
	result, typeMatched := resultRaw.(int64)
	if !typeMatched {
		return result, fmt.Errorf("invalid Count result type: %t", resultRaw)
	}

	return result, err
}

func (tracer Tracer) RunAndTraceDeleteCommand(ctx context.Context) (*mongo.DeleteResult, error) {
	resultRaw, err := tracer.RunAndTraceCommand(ctx)
	if err != nil {
		return nil, err
	}
	result, typeMatched := resultRaw.(*mongo.DeleteResult)
	if !typeMatched {
		return result, fmt.Errorf("invalid Delete result type: %t", resultRaw)
	}

	return result, nil
}

func (tracer Tracer) RunAndTraceCursorCommand(ctx context.Context) (*mongo.Cursor, error) {
	resultRaw, err := tracer.RunAndTraceCommand(ctx)
	if err != nil {
		return nil, err
	}
	result, typeMatched := resultRaw.(*mongo.Cursor)
	if !typeMatched {
		return result, fmt.Errorf("invalid Cursor result type: %t", resultRaw)
	}

	return result, nil
}

func (tracer Tracer) RunAndTraceInsertOne(ctx context.Context) (*mongo.InsertOneResult, error) {
	resultRaw, err := tracer.RunAndTraceCommand(ctx)
	if err != nil {
		return nil, err
	}
	result, typeMatched := resultRaw.(*mongo.InsertOneResult)
	if !typeMatched {
		return result, fmt.Errorf("invalid InsertOne result type: %t", resultRaw)
	}

	return result, nil
}

func (tracer Tracer) RunAndTraceInsertMany(ctx context.Context) (*mongo.InsertManyResult, error) {
	resultRaw, err := tracer.RunAndTraceCommand(ctx)
	if err != nil {
		return nil, err
	}
	result, typeMatched := resultRaw.(*mongo.InsertManyResult)
	if !typeMatched {
		return result, fmt.Errorf("invalid InsertMany result type: %t", resultRaw)
	}

	return result, nil
}

func (tracer Tracer) RunAndTraceBulkWrite(ctx context.Context) (*mongo.BulkWriteResult, error) {
	resultRaw, err := tracer.RunAndTraceCommand(ctx)
	if err != nil {
		return nil, err
	}
	result, typeMatched := resultRaw.(*mongo.BulkWriteResult)
	if !typeMatched {
		return result, fmt.Errorf("invalid Bulkwrite result type: %t", resultRaw)
	}

	return result, nil
}

func (tracer Tracer) RunAndTraceDrop(ctx context.Context) error {
	_, err := tracer.RunAndTraceCommand(ctx, tracer.Collection.Name())
	return err
}
