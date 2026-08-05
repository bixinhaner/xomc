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

func TestContactGroupHandler_UpdateRequiresRevision(t *testing.T) {
	repository := &contactGroupRepositoryStub{group: &ContactGroup{ID: uuid.New(), Revision: 2}}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewContactGroupHandler(NewContactGroupService(repository)).RegisterRoutes(router.Group(""))
	request := httptest.NewRequest(http.MethodPatch, "/notification-contact-groups/"+repository.group.ID.String(), bytes.NewBufferString(`{"name":"NOC"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusPreconditionRequired, response.Code)
}
