package notification

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTemplateManagementHandler_UpdateRequiresRevision(t *testing.T) {
	gin.SetMode(gin.TestMode)
	item := &ManagedTemplate{ID: uuid.New(), Revision: 1}
	handler := NewTemplateManagementHandler(NewTemplateManagementService(&templateManagementRepositoryStub{item: item}))
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))
	payload, err := json.Marshal(ManagedTemplateInput{Name: "alarm", Channel: "email", Subject: "subject", TextBody: "body"})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/notification-templates/"+item.ID.String(), bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusPreconditionRequired, response.Code)
}
