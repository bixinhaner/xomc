package notification

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/admin"
)

type deliveryApplicationStub struct {
	listFilter DeliveryFilter
	retryIDs   []uuid.UUID
	reason     string
	actor      string
}

func (s *deliveryApplicationStub) List(_ context.Context, _ uuid.UUID, _ bool, filter DeliveryFilter) ([]DeliveryView, error) {
	s.listFilter = filter
	return []DeliveryView{}, nil
}

func (s *deliveryApplicationStub) Get(context.Context, uuid.UUID, bool, uuid.UUID) (DeliveryView, error) {
	return DeliveryView{}, nil
}

func (s *deliveryApplicationStub) Attempts(context.Context, uuid.UUID, bool, uuid.UUID) ([]DomainDeliveryAttempt, error) {
	return []DomainDeliveryAttempt{}, nil
}

func (s *deliveryApplicationStub) Retry(_ context.Context, _ uuid.UUID, _ bool, ids []uuid.UUID, reason, actor string) (int, error) {
	s.retryIDs, s.reason, s.actor = ids, reason, actor
	return len(ids), nil
}

func TestDeliveryHandler_ListParsesTraceFilters(t *testing.T) {
	service := &deliveryApplicationStub{}
	router := deliveryTestRouter(service)
	occurrenceID := uuid.New()
	request := httptest.NewRequest(http.MethodGet,
		"/api/v1/notification-deliveries?occurrence_id="+occurrenceID.String()+"&channel=email&flow_state=suppressed&limit=20&offset=5", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, occurrenceID, *service.listFilter.OccurrenceID)
	require.Equal(t, "email", service.listFilter.Channel)
	require.Equal(t, "suppressed", service.listFilter.FlowState)
	require.Equal(t, 20, service.listFilter.Limit)
	require.Equal(t, 5, service.listFilter.Offset)
}

func TestDeliveryHandler_BatchRetryUsesStaticRouteAndActor(t *testing.T) {
	service := &deliveryApplicationStub{}
	router := deliveryTestRouter(service)
	id := uuid.New()
	body := `{"delivery_ids":["` + id.String() + `"],"reason":"configuration verified"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/notification-deliveries/retry", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, []uuid.UUID{id}, service.retryIDs)
	require.Equal(t, "configuration verified", service.reason)
	require.Equal(t, "operator", service.actor)
}

func deliveryTestRouter(service DeliveryApplication) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(admin.CtxKeyUserID, uuid.New())
		c.Set(admin.CtxKeyIsSuperAdmin, false)
		c.Set(admin.CtxKeyUsername, "operator")
		c.Next()
	})
	NewDeliveryHandler(service).RegisterRoutes(router.Group("/api/v1"))
	return router
}
