package provider

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingAPIEndpointSyncer struct {
	calls  int
	routes gin.RoutesInfo
}

func (s *recordingAPIEndpointSyncer) SyncApiEndpoints(_ context.Context, routes gin.RoutesInfo) (admin.SyncResult, error) {
	s.calls++
	s.routes = routes
	return admin.SyncResult{Total: len(routes)}, nil
}

func TestSyncApiEndpointsUsesWiredService(t *testing.T) {
	handler := admin.NewHandler(nil, zap.NewNop())
	routes := gin.RoutesInfo{{Method: "GET", Path: "/api/v1/admin/sysConfig/apply-batches/:id"}}
	handler.SetGinRoutes(routes)
	syncer := &recordingAPIEndpointSyncer{}

	err := syncApiEndpoints(
		&Container{Logger: zap.NewNop()},
		&adminHandlerDeps{adminHandler: handler, apiEndpointService: syncer},
	)

	require.NoError(t, err)
	require.Equal(t, 1, syncer.calls)
	require.Equal(t, routes, syncer.routes)
}
