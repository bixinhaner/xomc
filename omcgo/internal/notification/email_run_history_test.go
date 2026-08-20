package notification

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/core/model"
)

type emailRunHistoryReaderStub struct {
	filter       EmailRunHistoryFilter
	business     string
	runID        uuid.UUID
	callerID     uuid.UUID
	isSuperAdmin bool
}

func (s *emailRunHistoryReaderStub) ListEmailRuns(_ context.Context, filter EmailRunHistoryFilter) (*model.ListResponse[EmailRunHistory], error) {
	s.filter = filter
	return model.NewListResponse([]EmailRunHistory{}, 0, filter.Page, filter.PageSize), nil
}

func (s *emailRunHistoryReaderStub) ListEmailRunDeliveries(_ context.Context, business string, runID, callerID uuid.UUID, isSuperAdmin bool) ([]EmailRunDeliveryHistory, error) {
	s.business = business
	s.runID = runID
	s.callerID = callerID
	s.isSuperAdmin = isSuperAdmin
	return []EmailRunDeliveryHistory{{ID: uuid.New(), Recipient: "operator@example.com", Status: "sent"}}, nil
}

func TestEmailRunHistoryHandlerListUsesAuthenticatedOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reader := &emailRunHistoryReaderStub{}
	router := emailRunHistoryTestRouter(reader, uuid.New(), false)

	request := httptest.NewRequest(http.MethodGet, "/notifications/email-runs?business_type=alarm&status=sent&page=2&page_size=10", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, EmailRunBusinessAlarm, reader.filter.BusinessType)
	require.Equal(t, "sent", reader.filter.Status)
	require.Equal(t, 2, reader.filter.Page)
	require.Equal(t, 10, reader.filter.PageSize)
	require.NotEqual(t, uuid.Nil, reader.filter.CallerID)
	require.False(t, reader.filter.IsSuperAdmin)
}

func TestEmailRunHistoryHandlerRejectsUnknownBusiness(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reader := &emailRunHistoryReaderStub{}
	router := emailRunHistoryTestRouter(reader, uuid.New(), false)

	request := httptest.NewRequest(http.MethodGet, "/notifications/email-runs?business_type=sms", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusBadRequest, response.Code)
}

func TestEmailRunHistoryHandlerDeliveryScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reader := &emailRunHistoryReaderStub{}
	callerID := uuid.New()
	runID := uuid.New()
	router := emailRunHistoryTestRouter(reader, callerID, true)

	request := httptest.NewRequest(http.MethodGet, "/notifications/email-runs/kpi/"+runID.String()+"/deliveries", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, EmailRunBusinessKPI, reader.business)
	require.Equal(t, runID, reader.runID)
	require.Equal(t, callerID, reader.callerID)
	require.True(t, reader.isSuperAdmin)
}

func emailRunHistoryTestRouter(reader EmailRunHistoryReader, callerID uuid.UUID, isSuperAdmin bool) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(admin.CtxKeyUserID, callerID)
		c.Set(admin.CtxKeyIsSuperAdmin, isSuperAdmin)
		c.Next()
	})
	NewEmailRunHistoryHandler(reader).RegisterRoutes(router.Group("/notifications"))
	return router
}
