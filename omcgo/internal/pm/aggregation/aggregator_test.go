package aggregation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// NOTE: The Aggregator depends directly on pgxpool.Pool for raw SQL execution.
// RefreshContinuousAggregates cannot be unit-tested without a live TimescaleDB
// instance. These tests cover the constructor and struct wiring only. Full
// integration tests for continuous aggregate refresh belong in test/integration/.

func TestNewAggregator_NilPool(t *testing.T) {
	logger := zap.NewNop()

	agg := NewAggregator(nil, logger)

	assert.NotNil(t, agg)
	assert.Nil(t, agg.pool)
	assert.NotNil(t, agg.logger)
}

func TestNewAggregator_NilLogger(t *testing.T) {
	// Ensure NewAggregator does not panic when logger is nil.
	agg := NewAggregator(nil, nil)

	assert.NotNil(t, agg)
	assert.Nil(t, agg.pool)
	assert.Nil(t, agg.logger)
}
