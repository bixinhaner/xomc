package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	commonerrors "github.com/omcgo/omcgo/internal/common/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func init() {
	gin.SetMode(gin.TestMode)
}

// ginContext creates a fresh *gin.Context backed by an httptest.ResponseRecorder.
func ginContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c, w
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestOK(t *testing.T) {
	c, w := ginContext()

	body := map[string]string{"status": "ok"}
	OK(c, body)

	assert.Equal(t, http.StatusOK, w.Code)

	var got map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &got)
	require.NoError(t, err)
	assert.Equal(t, "ok", got["status"])
}

func TestCreated(t *testing.T) {
	c, w := ginContext()

	body := map[string]int{"id": 42}
	Created(c, body)

	assert.Equal(t, http.StatusCreated, w.Code)

	var got map[string]int
	err := json.Unmarshal(w.Body.Bytes(), &got)
	require.NoError(t, err)
	assert.Equal(t, 42, got["id"])
}

func TestNoContent(t *testing.T) {
	c, w := ginContext()

	NoContent(c)

	assert.Equal(t, http.StatusNoContent, c.Writer.Status())
	assert.Empty(t, w.Body.Bytes())
}

func TestMessage(t *testing.T) {
	c, w := ginContext()

	Message(c, "operation succeeded")

	assert.Equal(t, http.StatusOK, w.Code)

	var got map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &got)
	require.NoError(t, err)
	assert.Equal(t, "operation succeeded", got["message"])
}

func TestError(t *testing.T) {
	c, w := ginContext()

	Error(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var got commonerrors.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &got)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, got.Code)
	assert.Contains(t, got.Details, "invalid input")
}

func TestErrorFromErr_NotFound(t *testing.T) {
	c, w := ginContext()

	ErrorFromErr(c, commonerrors.ErrNotFound)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var got commonerrors.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &got)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, got.Code)
}

func TestErrorFromErr_AlreadyExists(t *testing.T) {
	c, w := ginContext()

	ErrorFromErr(c, commonerrors.ErrAlreadyExists)

	assert.Equal(t, http.StatusConflict, w.Code)

	var got commonerrors.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &got)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, got.Code)
}
