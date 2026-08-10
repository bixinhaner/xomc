package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestInitGeofenceModuleWiresDefinitionAndSettingsRepositories(t *testing.T) {
	container := &Container{
		Logger: zap.NewNop(),
	}

	err := initGeofenceModule(container)

	require.NoError(t, err)
	assert.NotNil(t, container.GeofenceService)
	assert.NotNil(t, container.GeofenceHandler)
}

type geofencePermissionRoleRepository struct{}

func (geofencePermissionRoleRepository) GetGroupIDs(
	context.Context,
	uuid.UUID,
) ([]uuid.UUID, error) {
	return nil, nil
}

func (geofencePermissionRoleRepository) SetGroupIDs(
	context.Context,
	uuid.UUID,
	[]uuid.UUID,
) error {
	return nil
}

func (geofencePermissionRoleRepository) GetUserVisibleGroupIDs(
	context.Context,
	uuid.UUID,
) ([]uuid.UUID, error) {
	return []uuid.UUID{}, nil
}

func (geofencePermissionRoleRepository) GetUserVisibleDeviceGrants(
	context.Context,
	uuid.UUID,
) ([]model.DeviceVisibilityGrant, error) {
	return []model.DeviceVisibilityGrant{}, nil
}

func (geofencePermissionRoleRepository) GetDeviceGroupData(
	context.Context,
	uuid.UUID,
) (*admin.RoleDeviceGroupData, error) {
	return &admin.RoleDeviceGroupData{}, nil
}

func (geofencePermissionRoleRepository) SetDeviceGroupData(
	context.Context,
	uuid.UUID,
	admin.RoleDeviceGroupData,
) error {
	return nil
}

func (geofencePermissionRoleRepository) ListRolesByGroupIDs(
	context.Context,
	[]uuid.UUID,
) ([]admin.Role, error) {
	return nil, nil
}

type geofencePermissionGroupExpander struct{}

func (geofencePermissionGroupExpander) ListChildIDs(
	context.Context,
	uuid.UUID,
) ([]uuid.UUID, error) {
	return nil, nil
}

func TestInitGeofenceModuleInjectsPermissionResolver(t *testing.T) {
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { require.NoError(t, redisClient.Close()) })

	container := &Container{
		Logger: zap.NewNop(),
		PermService: admin.NewPermissionService(
			geofencePermissionRoleRepository{},
			geofencePermissionGroupExpander{},
			redisClient,
			zap.NewNop(),
		),
	}
	require.NoError(t, initGeofenceModule(container))

	gin.SetMode(gin.TestMode)
	router := gin.New()
	actorID := uuid.New()
	router.Use(func(c *gin.Context) {
		c.Set(admin.CtxKeyUserID, actorID)
		c.Set(admin.CtxKeyIsSuperAdmin, false)
		c.Next()
	})
	container.GeofenceHandler.RegisterRoutes(router.Group("/api/v1"))
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/geofences/"+uuid.NewString()+"/binding-preview",
		bytes.NewBufferString(`{`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"ret":0`)
}

func TestInitGeofenceModuleLeavesPermissionResolverDisabledWhenServiceNil(
	t *testing.T,
) {
	container := &Container{
		Logger:      zap.NewNop(),
		PermService: nil,
	}
	require.NoError(t, initGeofenceModule(container))

	gin.SetMode(gin.TestMode)
	router := gin.New()
	actorID := uuid.New()
	router.Use(func(c *gin.Context) {
		c.Set(admin.CtxKeyUserID, actorID)
		c.Set(admin.CtxKeyIsSuperAdmin, false)
		c.Next()
	})
	container.GeofenceHandler.RegisterRoutes(router.Group("/api/v1"))
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/geofences/"+uuid.NewString()+"/binding-preview",
		bytes.NewBufferString(`{}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	if !assert.NotPanics(t, func() {
		router.ServeHTTP(recorder, request)
	}) {
		return
	}

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	var envelope struct {
		Ret  int    `json:"ret"`
		Msg  string `json:"msg"`
		Data any    `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Zero(t, envelope.Ret)
	require.Contains(t, envelope.Msg, "invalid input")
	require.NotContains(t, envelope.Msg, "internal error")
	require.Nil(t, envelope.Data)
}
