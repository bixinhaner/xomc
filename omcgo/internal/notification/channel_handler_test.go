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

func TestChannelHandler_UpdateRequiresRevisionAndVerifyIsUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	item := &ChannelConfig{ID: uuid.New(), Channel: "email", Name: "default", Revision: 1}
	handler := NewChannelHandler(NewChannelService(&channelRepositoryStub{item: item}, nil))
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))
	payload, err := json.Marshal(ChannelConfigUpdate{Name: "default", Parameters: json.RawMessage(`{}`)})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/notification-channels/"+item.ID.String(), bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusPreconditionRequired, response.Code)

	request = httptest.NewRequest(http.MethodPost, "/api/v1/notification-channels/"+item.ID.String()+"/verify", nil)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
}
