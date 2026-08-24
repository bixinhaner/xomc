package attention

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/admin"
)

type capturingReader struct {
	getCalls  int
	pageCalls int
	scope     Scope
	limits    [2]int
	section   Section
	page      int
	pageSize  int
}

func (r *capturingReader) Get(_ context.Context, scope Scope, abnormalLimit, todoLimit int) (*Response, error) {
	r.getCalls++
	r.scope = scope
	r.limits = [2]int{abnormalLimit, todoLimit}
	return &Response{Abnormalities: AttentionSection{Status: SectionOK, Items: []Item{}}, Todos: AttentionSection{Status: SectionOK, Items: []Item{}}}, nil
}

func (r *capturingReader) GetPage(_ context.Context, scope Scope, section Section, page, pageSize int) (*Page, error) {
	r.pageCalls++
	r.scope = scope
	r.section = section
	r.page = page
	r.pageSize = pageSize
	return &Page{Status: SectionOK, Items: []Item{}, Page: page, PageSize: pageSize}, nil
}

type stubGroupsResolver struct {
	groups []uuid.UUID
	err    error
}

func (r stubGroupsResolver) ResolveFromContext(*gin.Context) ([]uuid.UUID, error) {
	return r.groups, r.err
}

var attentionHandlerUserID = uuid.MustParse("892d12c0-ec1a-4fd1-8070-902d3aaf84e9")

func attentionRouter(handler *Handler, authenticated bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if authenticated {
		router.Use(func(c *gin.Context) {
			c.Set(admin.CtxKeyUserID, attentionHandlerUserID)
			c.Set(admin.CtxKeyUsername, "attention-user")
			c.Set(admin.CtxKeyIsSuperAdmin, true)
			c.Next()
		})
	}
	handler.RegisterRoutes(router.Group("/api/v1"))
	return router
}

func TestHandler_GetUsesDefaultsAndAuthenticatedScope(t *testing.T) {
	groupID := uuid.New()
	reader := &capturingReader{}
	router := attentionRouter(NewHandler(reader, stubGroupsResolver{groups: []uuid.UUID{groupID}}), true)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/attention", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, reader.getCalls)
	assert.Equal(t, [2]int{1, 1}, reader.limits)
	assert.Equal(t, attentionHandlerUserID, reader.scope.UserID)
	assert.Equal(t, "attention-user", reader.scope.Username)
	assert.True(t, reader.scope.IsSuperAdmin)
	assert.Equal(t, []uuid.UUID{groupID}, reader.scope.VisibleGroups)
}

func TestHandler_RejectsInvalidLimitBeforeCallingService(t *testing.T) {
	reader := &capturingReader{}
	router := attentionRouter(NewHandler(reader, nil), true)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/attention?abnormal_limit=6", nil))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Zero(t, reader.getCalls)
}

func TestHandler_GetTodoPagePassesPaginationContract(t *testing.T) {
	reader := &capturingReader{}
	router := attentionRouter(NewHandler(reader, nil), true)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/attention/todos?page=3&page_size=15", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, reader.pageCalls)
	assert.Equal(t, SectionTodos, reader.section)
	assert.Equal(t, 3, reader.page)
	assert.Equal(t, 15, reader.pageSize)
}

func TestHandler_FailsClosedWhenScopeCannotBeResolved(t *testing.T) {
	reader := &capturingReader{}
	router := attentionRouter(NewHandler(reader, stubGroupsResolver{err: errors.New("scope unavailable")}), true)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/attention", nil))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Zero(t, reader.getCalls)
}

func TestHandler_RequiresAuthenticatedUser(t *testing.T) {
	reader := &capturingReader{}
	router := attentionRouter(NewHandler(reader, nil), false)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/attention", nil))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Zero(t, reader.getCalls)
	require.NotEmpty(t, w.Body.String())
}
