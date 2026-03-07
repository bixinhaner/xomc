package infra

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestGracefulShutdown_Order(t *testing.T) {
	logger := zap.NewNop()
	gs := NewGracefulShutdown(5*time.Second, logger)

	var order []string

	gs.Register("db", 4, func(ctx context.Context) error {
		order = append(order, "db")
		return nil
	})
	gs.Register("http", 1, func(ctx context.Context) error {
		order = append(order, "http")
		return nil
	})
	gs.Register("redis", 3, func(ctx context.Context) error {
		order = append(order, "redis")
		return nil
	})
	gs.Register("nats", 2, func(ctx context.Context) error {
		order = append(order, "nats")
		return nil
	})

	err := gs.Shutdown(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, []string{"http", "nats", "redis", "db"}, order)
}

func TestGracefulShutdown_ErrorCollection(t *testing.T) {
	logger := zap.NewNop()
	gs := NewGracefulShutdown(5*time.Second, logger)

	gs.Register("ok", 1, func(ctx context.Context) error { return nil })
	gs.Register("fail", 2, func(ctx context.Context) error {
		return assert.AnError
	})

	err := gs.Shutdown(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fail")
}
