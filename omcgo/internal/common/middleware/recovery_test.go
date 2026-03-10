package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestRecovery_PanicReturns500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Recovery(zap.NewNop()))
	r.GET("/test", func(c *gin.Context) {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, float64(500), body["code"])
	assert.Equal(t, "internal server error", body["message"])
}

func TestRecovery_NormalRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Recovery(zap.NewNop()))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, "ok", body["status"])
}

func TestRecovery_PanicWithError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Recovery(zap.NewNop()))
	r.GET("/test", func(c *gin.Context) {
		panic(errors.New("fail"))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, float64(500), body["code"])
	assert.Equal(t, "internal server error", body["message"])
}
