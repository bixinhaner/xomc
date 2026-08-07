package notification

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRuleHandler_MutationRequiresIfMatch(t *testing.T) {
	repository := &ruleRepositoryStub{rule: &NotificationRule{ID: uuid.New(), Revision: 2}}
	router := ruleTestRouter(repository)
	request := httptest.NewRequest(http.MethodPatch, "/notification-rules/"+repository.rule.ID.String(), bytes.NewBufferString(`{"name":"rule"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusPreconditionRequired, response.Code)
}

func TestRuleHandler_StaleRevisionReturnsPreconditionFailed(t *testing.T) {
	repository := &ruleRepositoryStub{rule: &NotificationRule{ID: uuid.New(), Revision: 3}, err: ErrRevisionMismatch}
	router := ruleTestRouter(repository)
	request := httptest.NewRequest(http.MethodPost, "/notification-rules/"+repository.rule.ID.String()+"/publish", nil)
	request.Header.Set("If-Match", `"2"`)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusPreconditionFailed, response.Code)
	require.Equal(t, int64(2), repository.publishRevision)
}

func TestRuleHandler_EnableRequiresExactVersionAndReturnsNewETag(t *testing.T) {
	versionID := uuid.New()
	repository := &ruleRepositoryStub{rule: &NotificationRule{ID: uuid.New(), Revision: 5}}
	router := ruleTestRouter(repository)
	request := httptest.NewRequest(http.MethodPost, "/notification-rules/"+repository.rule.ID.String()+"/enable",
		bytes.NewBufferString(`{"rule_version_id":"`+versionID.String()+`"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", `W/"4"`)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, int64(4), repository.enableRevision)
	require.Equal(t, versionID, repository.enableVersionID)
	require.Equal(t, `"5"`, response.Header().Get("ETag"))
}

func TestRuleHandler_DisableUsesOptimisticRevisionAndReturnsNewETag(t *testing.T) {
	repository := &ruleRepositoryStub{rule: &NotificationRule{ID: uuid.New(), Revision: 7}}
	router := ruleTestRouter(repository)
	request := httptest.NewRequest(http.MethodPost, "/notification-rules/"+repository.rule.ID.String()+"/disable", nil)
	request.Header.Set("If-Match", `"6"`)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, int64(6), repository.disableRevision)
	require.Equal(t, `"7"`, response.Header().Get("ETag"))
}

func ruleTestRouter(repository RuleRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewRuleHandler(NewRuleService(repository)).RegisterRoutes(router.Group(""))
	return router
}
