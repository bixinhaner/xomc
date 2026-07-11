package export

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	appcontext "github.com/omcgo/omcgo/internal/core/context"
)

func TestHandler_Create_CapturesRequestLocaleInParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	taskID := uuid.New()
	repo := &stubRepo{
		createID: taskID,
		getTask:  &Task{ID: taskID, SourceType: SourceAdhoc, Status: StatusPending},
	}
	h := NewHandler(NewService(repo, &stubEnqueuer{insertID: uuid.New()}), nil, zap.NewNop())
	router := gin.New()
	h.RegisterRoutes(router.Group(""))

	body := []byte(`{"source_type":"adhoc","params":{"task_id":"11111111-1111-1111-1111-111111111111"}}`)
	req := httptest.NewRequest(http.MethodPost, "/pm/exports", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(appcontext.WithLocale(context.Background(), appcontext.LocaleEN))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, repo.created)
	assert.JSONEq(t,
		`{"task_id":"11111111-1111-1111-1111-111111111111","locale":"en-US"}`,
		string(repo.created.Params),
	)
}
