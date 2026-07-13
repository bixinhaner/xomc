package mml

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestHandler_CreateScriptExecution_RejectsClientPlanFields(t *testing.T) {
	script := scriptExecutionFixture()
	created := 0
	scriptRepo := &hScriptRepo{
		GetByIDFn: func(context.Context, uuid.UUID) (*MMLScript, error) { return script, nil },
	}
	taskRepo := &hTaskRepo{
		CreateFn: func(_ context.Context, task *MMLTask) error { created++; task.ID = uuid.New(); return nil },
	}
	svc := NewService(&mockCommandRepo{}, scriptRepo, taskRepo, &mockCustomCommandRepo{}, nil, zap.NewNop())
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("username", "alice"); c.Next() })
	NewHandler(svc, zap.NewNop()).RegisterRoutes(r.Group("/api/v1"))

	body, err := json.Marshal(map[string]interface{}{
		"task_name": "巡检", "execute_type": "immediate",
		"commands": []map[string]interface{}{{"command_code": "CLIENT_OVERRIDE"}},
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mml/scripts/"+script.ID.String()+"/executions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	require.Equal(t, http.StatusBadRequest, resp.Code)
	require.Equal(t, 0, created)
}
