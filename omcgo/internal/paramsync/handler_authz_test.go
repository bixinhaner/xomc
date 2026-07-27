package paramsync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

type fixedParamSyncPermission struct{}

func (fixedParamSyncPermission) GetUserVisibleDeviceGrants(context.Context, uuid.UUID, bool) ([]model.DeviceVisibilityGrant, error) {
	return []model.DeviceVisibilityGrant{}, nil
}

type denyingParamSyncAuthorizer struct{}

func (denyingParamSyncAuthorizer) AuthorizeDeviceGroupAccessByGrants(context.Context, uuid.UUID, []model.DeviceVisibilityGrant) error {
	return commonerrors.ErrForbidden
}

func paramSyncAuthzRouter(handler *Handler, isSuper bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(admin.CtxKeyUserID, uuid.New())
		c.Set(admin.CtxKeyIsSuperAdmin, isSuper)
		c.Next()
	})
	handler.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func TestHandlerRejectsRequestOutsideVisibleDeviceScope(t *testing.T) {
	repo := newMemoryRequestRepo()
	req := &SyncRequest{ID: uuid.New(), DeviceID: uuid.New(), DeviceSN: "SN-HIDDEN"}
	repo.requests[req.ID] = req
	service := NewService(repo, stubPlanner{})
	handler := NewHandler(service).WithAuthorization(fixedParamSyncPermission{}, denyingParamSyncAuthorizer{})
	r := paramSyncAuthzRouter(handler, false)

	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodGet, "/api/v1/parameter-sync/requests/"+req.ID.String(), nil)
	r.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandlerRestrictsDeadOutboxToSuperAdmin(t *testing.T) {
	service := NewService(newMemoryRequestRepo(), stubPlanner{})
	handler := NewHandler(service).WithOperations(NewOperations(nil, service))
	r := paramSyncAuthzRouter(handler, false)

	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodGet, "/api/v1/parameter-sync/outbox/dead", nil)
	r.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
