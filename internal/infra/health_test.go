package infra

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthChecker_AllHealthy(t *testing.T) {
	hc := NewHealthChecker()
	hc.Register("db", func(ctx context.Context) error { return nil })
	hc.Register("redis", func(ctx context.Context) error { return nil })

	results := hc.CheckAll(context.Background())
	require.Len(t, results, 2)

	for _, r := range results {
		assert.Equal(t, "healthy", r.Status)
		assert.Empty(t, r.Error)
	}

	assert.True(t, hc.IsHealthy(context.Background()))
	assert.Equal(t, 200, hc.StatusCode(context.Background()))
}

func TestHealthChecker_OneUnhealthy(t *testing.T) {
	hc := NewHealthChecker()
	hc.Register("db", func(ctx context.Context) error { return nil })
	hc.Register("redis", func(ctx context.Context) error { return errors.New("connection refused") })

	results := hc.CheckAll(context.Background())
	require.Len(t, results, 2)

	assert.Equal(t, "healthy", results[0].Status)
	assert.Equal(t, "unhealthy", results[1].Status)
	assert.Equal(t, "connection refused", results[1].Error)

	assert.False(t, hc.IsHealthy(context.Background()))
	assert.Equal(t, 503, hc.StatusCode(context.Background()))
}

func TestHealthChecker_Empty(t *testing.T) {
	hc := NewHealthChecker()
	assert.True(t, hc.IsHealthy(context.Background()))
	assert.Equal(t, 200, hc.StatusCode(context.Background()))
}
