package outbox

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingExecutor struct {
	query string
	args  []any
	err   error
	calls int
}

func (e *recordingExecutor) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	e.calls++
	e.query = query
	e.args = args
	if e.err != nil {
		return pgconn.CommandTag{}, e.err
	}
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func TestRepositoryInsertPreservesDurableIdentity(t *testing.T) {
	executor := &recordingExecutor{}
	repo := &Repository{}
	record := Record{
		AggregateType: "device",
		AggregateID:   "device-1",
		Subject:       "device.location.observed",
		Payload:       []byte(`{"observation_version":18}`),
		DedupeKey:     "device.location.observed:device-1:18",
	}

	err := repo.insert(context.Background(), executor, record)

	require.NoError(t, err)
	require.Equal(t, 1, executor.calls)
	assert.Contains(t, executor.query, "INSERT INTO event_outbox")
	assert.Contains(t, executor.query, "aggregate_type")
	assert.Contains(t, executor.query, "dedupe_key")
	assert.Contains(t, executor.query, "ON CONFLICT (dedupe_key) DO NOTHING")
	assert.Contains(t, executor.args, record.Subject)
	assert.Contains(t, executor.args, record.DedupeKey)
}

func TestRepositoryInsertRejectsMissingIdentityBeforeSQL(t *testing.T) {
	tests := []Record{
		{AggregateType: "device", AggregateID: "device-1", DedupeKey: "key", Payload: []byte(`{}`)},
		{AggregateType: "device", AggregateID: "device-1", Subject: "subject", Payload: []byte(`{}`)},
	}
	for _, record := range tests {
		executor := &recordingExecutor{}
		err := (&Repository{}).insert(context.Background(), executor, record)
		require.Error(t, err)
		assert.Zero(t, executor.calls)
	}
}

func TestRepositoryInsertWrapsExecutionFailure(t *testing.T) {
	executor := &recordingExecutor{err: errors.New("database unavailable")}
	record := Record{
		AggregateType: "device",
		AggregateID:   "device-1",
		Subject:       "device.location.observed",
		Payload:       []byte(`{}`),
		DedupeKey:     "key",
	}

	err := (&Repository{}).insert(context.Background(), executor, record)

	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "insert event outbox"))
	assert.ErrorIs(t, err, executor.err)
}
