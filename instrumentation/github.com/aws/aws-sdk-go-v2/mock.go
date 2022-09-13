package awsv2

import (
	"context"
	"time"

	"github.com/aws/smithy-go/middleware"
)

type InitializeInput = middleware.InitializeInput
type InitializeHandler = middleware.InitializeHandler
type InitializeOutput = middleware.InitializeOutput
type DeserializeInput = middleware.DeserializeInput
type DeserializeOutput = middleware.DeserializeOutput
type Metadata = middleware.Metadata

var MockSleepTime time.Duration = time.Duration(0)

type MockHandler struct {
}

func (handler MockHandler) HandleInitialize(ctx context.Context,
	in InitializeInput) (out InitializeOutput, metadata Metadata, err error) {
	return middleware.InitializeOutput{
		Result: ctx,
	}, middleware.Metadata{}, nil
}

func (handler MockHandler) HandleDeserialize(ctx context.Context,
	in DeserializeInput) (out DeserializeOutput, metadata Metadata, err error) {
	time.Sleep(MockSleepTime)
	return middleware.DeserializeOutput{
		Result: ctx,
	}, middleware.Metadata{}, nil
}
