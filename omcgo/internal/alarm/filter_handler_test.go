package alarm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubAlarmFilterRuleRepository struct {
	createFn      func(context.Context, *AlarmFilterRule) error
	getByIDFn     func(context.Context, uuid.UUID) (*AlarmFilterRule, error)
	updateFn      func(context.Context, *AlarmFilterRule) error
	deleteFn      func(context.Context, uuid.UUID) error
	listFn        func(context.Context, AlarmFilterRuleFilter) (*model.ListResponse[AlarmFilterRule], error)
	listEnabledFn func(context.Context) ([]AlarmFilterRule, error)
	toggleFn      func(context.Context, uuid.UUID) error
}

func (s *stubAlarmFilterRuleRepository) Create(ctx context.Context, rule *AlarmFilterRule) error {
	if s.createFn != nil {
		return s.createFn(ctx, rule)
	}
	return nil
}

func (s *stubAlarmFilterRuleRepository) GetByID(ctx context.Context, id uuid.UUID) (*AlarmFilterRule, error) {
	if s.getByIDFn != nil {
		return s.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (s *stubAlarmFilterRuleRepository) Update(ctx context.Context, rule *AlarmFilterRule) error {
	if s.updateFn != nil {
		return s.updateFn(ctx, rule)
	}
	return nil
}

func (s *stubAlarmFilterRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, id)
	}
	return nil
}

func (s *stubAlarmFilterRuleRepository) List(ctx context.Context, filter AlarmFilterRuleFilter) (*model.ListResponse[AlarmFilterRule], error) {
	if s.listFn != nil {
		return s.listFn(ctx, filter)
	}
	return &model.ListResponse[AlarmFilterRule]{}, nil
}

func (s *stubAlarmFilterRuleRepository) Toggle(ctx context.Context, id uuid.UUID) error {
	if s.toggleFn != nil {
		return s.toggleFn(ctx, id)
	}
	return nil
}

func (s *stubAlarmFilterRuleRepository) ListEnabled(ctx context.Context) ([]AlarmFilterRule, error) {
	if s.listEnabledFn != nil {
		return s.listEnabledFn(ctx)
	}
	return nil, nil
}

func TestFilterHandlerCreate_SetsOperatorAuditFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var captured *AlarmFilterRule
	repo := &stubAlarmFilterRuleRepository{
		createFn: func(_ context.Context, rule *AlarmFilterRule) error {
			copy := *rule
			captured = &copy
			return nil
		},
	}
	handler := NewFilterHandler(repo, zap.NewNop())
	router := gin.New()
	router.POST("/rules", func(c *gin.Context) {
		c.Set(admin.CtxKeyUsername, "alice")
		handler.Create(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/rules", strings.NewReader(`{"name":"rule-a","filter_type":"device","action":"ignore"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusCreated, resp.Code)
	require.NotNil(t, captured)
	require.Equal(t, "alice", captured.CreatedBy)
	require.Equal(t, "alice", captured.UpdatedBy)
}

func TestFilterHandlerCreate_ValidatesEffectiveWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &stubAlarmFilterRuleRepository{}
	handler := NewFilterHandler(repo, zap.NewNop())
	router := gin.New()
	router.POST("/rules", handler.Create)

	req := httptest.NewRequest(http.MethodPost, "/rules", strings.NewReader(`{"name":"rule-a","filter_type":"device","action":"ignore","effective_start":"2026-08-12T09:00:00+08:00"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestFilterHandlerUpdate_ClearsEffectiveWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	id := uuid.New()
	start := time.Date(2026, 8, 12, 9, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	end := start.Add(time.Hour)
	var captured *AlarmFilterRule
	repo := &stubAlarmFilterRuleRepository{
		getByIDFn: func(context.Context, uuid.UUID) (*AlarmFilterRule, error) {
			return &AlarmFilterRule{ID: id, Name: "rule-a", FilterType: FilterTypeDevice, Action: FilterActionIgnore, EffectiveStart: &start, EffectiveEnd: &end}, nil
		},
		updateFn: func(_ context.Context, rule *AlarmFilterRule) error {
			captured = rule
			return nil
		},
	}
	handler := NewFilterHandler(repo, zap.NewNop())
	router := gin.New()
	router.PUT("/rules/:id", handler.Update)

	req := httptest.NewRequest(http.MethodPut, "/rules/"+id.String(), strings.NewReader(`{"clear_effective_window":true}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.NotNil(t, captured)
	require.Nil(t, captured.EffectiveStart)
	require.Nil(t, captured.EffectiveEnd)
}

func TestFilterHandlerUpdate_SetsUpdatedByFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	id := uuid.New()
	existing := &AlarmFilterRule{
		ID:         id,
		Name:       "rule-a",
		FilterType: FilterTypeDevice,
		Action:     FilterActionIgnore,
		Enabled:    true,
		CreatedBy:  "bob",
	}
	var captured *AlarmFilterRule
	repo := &stubAlarmFilterRuleRepository{
		getByIDFn: func(_ context.Context, gotID uuid.UUID) (*AlarmFilterRule, error) {
			require.Equal(t, id, gotID)
			copy := *existing
			return &copy, nil
		},
		updateFn: func(_ context.Context, rule *AlarmFilterRule) error {
			copy := *rule
			captured = &copy
			return nil
		},
	}
	handler := NewFilterHandler(repo, zap.NewNop())
	router := gin.New()
	router.PUT("/rules/:id", func(c *gin.Context) {
		c.Set(admin.CtxKeyUsername, "alice")
		handler.Update(c)
	})

	req := httptest.NewRequest(http.MethodPut, "/rules/"+id.String(), strings.NewReader(`{"name":"rule-b"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.NotNil(t, captured)
	require.Equal(t, "rule-b", captured.Name)
	require.Equal(t, "bob", captured.CreatedBy)
	require.Equal(t, "alice", captured.UpdatedBy)
}

func TestFilterHandlerToggle_SetsUpdatedByFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	id := uuid.New()
	existing := &AlarmFilterRule{
		ID:         id,
		Name:       "rule-a",
		FilterType: FilterTypeDevice,
		Action:     FilterActionIgnore,
		Enabled:    true,
		CreatedBy:  "bob",
	}
	var captured *AlarmFilterRule
	repo := &stubAlarmFilterRuleRepository{
		getByIDFn: func(_ context.Context, gotID uuid.UUID) (*AlarmFilterRule, error) {
			require.Equal(t, id, gotID)
			copy := *existing
			return &copy, nil
		},
		updateFn: func(_ context.Context, rule *AlarmFilterRule) error {
			copy := *rule
			captured = &copy
			return nil
		},
	}
	handler := NewFilterHandler(repo, zap.NewNop())
	router := gin.New()
	router.POST("/rules/:id/toggle", func(c *gin.Context) {
		c.Set(admin.CtxKeyUsername, "alice")
		handler.Toggle(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/rules/"+id.String()+"/toggle", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.NotNil(t, captured)
	require.False(t, captured.Enabled)
	require.Equal(t, "alice", captured.UpdatedBy)
}
