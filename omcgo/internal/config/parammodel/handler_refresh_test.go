package parammodel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type countingRegistryRefresher struct {
	calls int
}

func (r *countingRegistryRefresher) Refresh(context.Context) error {
	r.calls++
	return nil
}

func TestHandlerRefreshAsyncRefreshesProductRoutesWithoutParamRegistry(t *testing.T) {
	productRoutes := &countingRegistryRefresher{}
	h := &Handler{logger: zap.NewNop()}
	h.SetProductRegistryRefresher(productRoutes)

	h.refreshAsync(context.Background(), "update-model")

	require.Equal(t, 1, productRoutes.calls)
}
