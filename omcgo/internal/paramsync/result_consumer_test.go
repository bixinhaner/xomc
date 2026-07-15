package paramsync

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/event"
)

type recordingResultProcessor struct {
	err   error
	calls int
}

func (p *recordingResultProcessor) Process(context.Context, event.ParamSyncTaskResultPayload) (ResultProcessOutcome, error) {
	p.calls++
	return ResultProcessOutcome{}, p.err
}

func TestResultConsumer_ACKOnlyAfterProcessorCommit(t *testing.T) {
	processor := &recordingResultProcessor{err: errors.New("postgres unavailable")}
	consumer := NewResultConsumer(nil, processor)
	payload := event.ParamSyncTaskResultPayload{EventID: uuid.NewString(), RunID: uuid.New(), RequestID: uuid.New(), TaskID: uuid.NewString()}
	evt, err := event.NewEvent(event.SubjectParamSyncTaskResult, payload)
	require.NoError(t, err)

	err = consumer.Handle(context.Background(), evt)
	assert.ErrorContains(t, err, "postgres unavailable")
	assert.Equal(t, 1, processor.calls)

	processor.err = nil
	assert.NoError(t, consumer.Handle(context.Background(), evt))
	assert.Equal(t, 2, processor.calls)
}
