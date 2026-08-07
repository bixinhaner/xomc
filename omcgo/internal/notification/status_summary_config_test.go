package notification

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/admin"
)

type statusSummaryConfigRepositoryStub struct {
	record *StatusSummaryRuntimeConfig
	input  StatusSummaryConfigInput
	seen   []ProtectedStatusSummaryRecipient
}

func (s *statusSummaryConfigRepositoryStub) Get(context.Context, uuid.UUID) (*StatusSummaryRuntimeConfig, error) {
	return s.record, nil
}

func (s *statusSummaryConfigRepositoryStub) Update(_ context.Context, id uuid.UUID, revision int64, input StatusSummaryConfigInput, recipients []ProtectedStatusSummaryRecipient, _ string) (*StatusSummaryRuntimeConfig, error) {
	s.input = input
	s.seen = recipients
	s.record = &StatusSummaryRuntimeConfig{ID: id, Enabled: input.Enabled, SendTime: input.SendTime, TimeZone: input.TimeZone, Recipients: recipients, Revision: revision + 1}
	return s.record, nil
}

type statusSummaryConfigProtectorStub struct{}

func (statusSummaryConfigProtectorStub) Protect(_, address string) ([]byte, int, []byte, error) {
	return []byte(address), 1, []byte(address), nil
}

func (statusSummaryConfigProtectorStub) Unprotect(_ string, ciphertext []byte, _ int) (string, error) {
	return string(ciphertext), nil
}

func TestStatusSummaryConfigService_RejectsEnableWhenSMTPIsDisabled(t *testing.T) {
	service := NewStatusSummaryConfigService(&statusSummaryConfigRepositoryStub{}, statusSummaryConfigProtectorStub{}, false)
	_, err := service.Update(context.Background(), 1, StatusSummaryConfigInput{
		Enabled: true, SendTime: "00:00", TimeZone: "Asia/Shanghai", Recipients: []string{"noc@example.com"},
	}, "admin")
	require.ErrorIs(t, err, ErrStatusSummaryNotReady)
}

func TestStatusSummaryConfigService_NormalizesAndProtectsRecipients(t *testing.T) {
	repository := &statusSummaryConfigRepositoryStub{}
	service := NewStatusSummaryConfigService(repository, statusSummaryConfigProtectorStub{}, true)
	config, err := service.Update(context.Background(), 1, StatusSummaryConfigInput{
		Enabled: true, SendTime: "08:30", TimeZone: "Asia/Shanghai", Recipients: []string{"NOC@example.com;ops@example.com", "noc@example.com"},
	}, "admin")

	require.NoError(t, err)
	require.Equal(t, []string{"noc@example.com", "ops@example.com"}, config.Recipients)
	require.Len(t, repository.seen, 2)
	require.Equal(t, int64(2), config.Revision)
}

func TestStatusSummaryConfigHandler_RejectsNonSuperAdmin(t *testing.T) {
	router := statusSummaryConfigTestRouter(NewStatusSummaryConfigService(nil, nil, false), false)
	request := httptest.NewRequest(http.MethodGet, "/notification/status-summary-settings", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusForbidden, response.Code)
}

func TestStatusSummaryConfigHandler_UpdateRequiresIfMatch(t *testing.T) {
	repository := &statusSummaryConfigRepositoryStub{record: &StatusSummaryRuntimeConfig{ID: StatusSummaryConfigID(), Revision: 1}}
	service := NewStatusSummaryConfigService(repository, statusSummaryConfigProtectorStub{}, true)
	router := statusSummaryConfigTestRouter(service, true)
	request := httptest.NewRequest(http.MethodPatch, "/notification/status-summary-settings", bytes.NewBufferString(`{"enabled":false,"send_time":"00:00","time_zone":"Asia/Shanghai"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusPreconditionRequired, response.Code)
}

func TestStatusSummaryConfigHandler_UpdateRejectsEnableWhenNotReady(t *testing.T) {
	repository := &statusSummaryConfigRepositoryStub{record: &StatusSummaryRuntimeConfig{ID: StatusSummaryConfigID(), Revision: 1}}
	service := NewStatusSummaryConfigService(repository, statusSummaryConfigProtectorStub{}, false)
	router := statusSummaryConfigTestRouter(service, true)
	request := httptest.NewRequest(http.MethodPatch, "/notification/status-summary-settings", bytes.NewBufferString(`{"enabled":true,"send_time":"00:00","time_zone":"Asia/Shanghai","recipients":["noc@example.com"]}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", `"1"`)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusConflict, response.Code)
}

func TestStatusSummaryConfigHandler_UpdateReturnsNewETag(t *testing.T) {
	repository := &statusSummaryConfigRepositoryStub{record: &StatusSummaryRuntimeConfig{ID: StatusSummaryConfigID(), Revision: 1}}
	service := NewStatusSummaryConfigService(repository, statusSummaryConfigProtectorStub{}, true)
	router := statusSummaryConfigTestRouter(service, true)
	request := httptest.NewRequest(http.MethodPatch, "/notification/status-summary-settings", bytes.NewBufferString(`{"enabled":true,"send_time":"00:00","time_zone":"Asia/Shanghai","recipients":["noc@example.com"]}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", `"1"`)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, `"2"`, response.Header().Get("ETag"))
}

func statusSummaryConfigTestRouter(service *StatusSummaryConfigService, superAdmin bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(admin.CtxKeyIsSuperAdmin, superAdmin)
		c.Set(admin.CtxKeyUsername, "test-admin")
		c.Next()
	})
	NewStatusSummaryConfigHandler(service).RegisterRoutes(router.Group(""))
	return router
}
