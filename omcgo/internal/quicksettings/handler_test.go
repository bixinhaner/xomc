package quicksettings

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRouter(reg *Registry) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api/v1")
	NewHandler(reg).RegisterRoutes(api)
	return r
}

func TestHandler_GetGroups_LTE(t *testing.T) {
	reg := NewRegistry()
	reg.Replace(TechLTE, []Group{
		{ID: "enb-cell", TitleZh: "小区参数", Params: []Param{{Name: "ECI", StandardPath: "Device.X.CellIdentity"}}},
	})

	r := setupRouter(reg)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/quicksettings/groups?tech=lte", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Groups []Group `json:"groups"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Groups, 1)
	assert.Equal(t, "enb-cell", resp.Groups[0].ID)
	assert.Equal(t, "小区参数", resp.Groups[0].TitleZh)
}

func TestHandler_GetGroups_NR_Empty(t *testing.T) {
	reg := NewRegistry()
	r := setupRouter(reg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/quicksettings/groups?tech=nr", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Groups []Group `json:"groups"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Groups, 0)
}

func TestHandler_GetGroups_InvalidTech_400(t *testing.T) {
	r := setupRouter(NewRegistry())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/quicksettings/groups?tech=cdma", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetGroups_MissingTech_400(t *testing.T) {
	r := setupRouter(NewRegistry())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/quicksettings/groups", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
