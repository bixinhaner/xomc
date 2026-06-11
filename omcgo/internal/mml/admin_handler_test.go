package mml

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ============================================================
// admin_handler_test.go — issue #125-mml 问题 1 的 handler 层错误映射回归
//
// 覆盖：POST /mml/admin/groups 引用不存在 param_version 时,FK violation(23503)
//       经 service 翻成 ErrGroupParamVersionNotFound,handler 映射 422,
//       且响应体不外泄裸 SQL 约束名 / SQLSTATE。
// ============================================================

// stubCommandReader 满足 NewAdminHandler 的 commandReader（GetByID）契约。
// CreateGroup 路径不会触达它,仅为构造 handler 提供占位。
type stubCommandReader struct{}

func (stubCommandReader) GetByID(_ context.Context, _ uuid.UUID) (*MMLCommand, error) {
	return nil, ErrCommandNotFound
}

func setupAdminRouter(h *AdminHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func TestAdminHandler_CreateGroup_FKViolation_Maps422NoLeak(t *testing.T) {
	svc, gRepo, _, _, _ := newAdminTestService()
	// repo 抛 PostgreSQL 外键违反(23503)——param_version 不存在。
	gRepo.createErr = &pgconn.PgError{
		Code:           "23503",
		Message:        `insert or update on table "mml_command_groups" violates foreign key constraint "mml_param_groups_param_version_fkey"`,
		ConstraintName: "mml_param_groups_param_version_fkey",
	}

	h := NewAdminHandler(svc, stubCommandReader{}, zap.NewNop())
	router := setupAdminRouter(h)

	body, _ := json.Marshal(map[string]any{
		"group_code":    "SMK_BADGRP",
		"param_version": "smoke-no-such-version",
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/mml/admin/groups", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, r)

	// 精确 422（非 500）。
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code,
		"FK violation 应映射 422，实际 %d body=%s", w.Code, w.Body.String())

	// 响应体不得外泄裸 SQL 约束名 / SQLSTATE。
	respBody := w.Body.String()
	assert.NotContains(t, respBody, "foreign key constraint")
	assert.NotContains(t, respBody, "23503")
	assert.NotContains(t, respBody, "mml_param_groups_param_version_fkey")
	assert.True(t, strings.Contains(respBody, "param_version"),
		"响应应如实告知 param_version 不存在，实际 %s", respBody)
}

func TestAdminHandler_CreateGroup_Happy_201(t *testing.T) {
	svc, _, _, _, _ := newAdminTestService()
	h := NewAdminHandler(svc, stubCommandReader{}, zap.NewNop())
	router := setupAdminRouter(h)

	body, _ := json.Marshal(map[string]any{
		"group_code":    "SMK_GRP",
		"param_version": "STANDARD",
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/mml/admin/groups", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, r)

	require.Equal(t, http.StatusCreated, w.Code, "body=%s", w.Body.String())
}
