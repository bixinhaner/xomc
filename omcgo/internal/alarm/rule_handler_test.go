package alarm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	commonerrors "github.com/omcgo/omcgo/internal/common/errors"
	"github.com/omcgo/omcgo/internal/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockAlarmRuleRepository implements AlarmRuleRepository for testing.
type mockAlarmRuleRepository struct {
	rules map[uuid.UUID]*AlarmRule
}

func newMockAlarmRuleRepository() *mockAlarmRuleRepository {
	return &mockAlarmRuleRepository{rules: make(map[uuid.UUID]*AlarmRule)}
}

func (m *mockAlarmRuleRepository) Create(_ context.Context, rule *AlarmRule) error {
	now := time.Now()
	rule.ID = uuid.New()
	rule.CreatedAt = now
	rule.UpdatedAt = now
	m.rules[rule.ID] = rule
	return nil
}

func (m *mockAlarmRuleRepository) GetByID(_ context.Context, id uuid.UUID) (*AlarmRule, error) {
	r, ok := m.rules[id]
	if !ok {
		return nil, commonerrors.ErrNotFound
	}
	return r, nil
}

func (m *mockAlarmRuleRepository) Update(_ context.Context, rule *AlarmRule) error {
	if _, ok := m.rules[rule.ID]; !ok {
		return commonerrors.ErrNotFound
	}
	rule.UpdatedAt = time.Now()
	m.rules[rule.ID] = rule
	return nil
}

func (m *mockAlarmRuleRepository) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := m.rules[id]; !ok {
		return commonerrors.ErrNotFound
	}
	delete(m.rules, id)
	return nil
}

func (m *mockAlarmRuleRepository) List(_ context.Context, _ AlarmRuleFilter) (*model.ListResponse[AlarmRule], error) {
	var items []AlarmRule
	for _, r := range m.rules {
		items = append(items, *r)
	}
	return model.NewListResponse(items, int64(len(items)), 1, 20), nil
}

// setupRuleHandlerTest creates a RuleHandler wired to a mock repository and
// returns the handler, mock repo, and a gin router with routes registered.
func setupRuleHandlerTest() (*RuleHandler, *mockAlarmRuleRepository, *gin.Engine) {
	gin.SetMode(gin.TestMode)

	repo := newMockAlarmRuleRepository()
	logger := zap.NewNop()
	handler := NewRuleHandler(repo, logger)

	router := gin.New()
	handler.RegisterRoutes(router.Group(""))
	return handler, repo, router
}

// seedRule inserts a rule into the mock repository and returns it.
func seedRule(repo *mockAlarmRuleRepository, opts ...func(*AlarmRule)) *AlarmRule {
	now := time.Now()
	rule := &AlarmRule{
		ID:              uuid.New(),
		Name:            "Test Rule",
		Description:     "test description",
		AlarmCode:       "ALM001",
		Severity:        2,
		ConditionType:   "threshold",
		ConditionConfig: json.RawMessage(`{"max": 100}`),
		ActionType:      "notify",
		ActionConfig:    json.RawMessage(`{"email": "admin@test.com"}`),
		Carrier:         "cmcc",
		Technology:      "lte",
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	for _, fn := range opts {
		fn(rule)
	}
	repo.rules[rule.ID] = rule
	return rule
}

// ---------- ListRules ----------

func TestRuleHandler_ListRules_OK(t *testing.T) {
	_, repo, router := setupRuleHandlerTest()

	seedRule(repo)
	seedRule(repo, func(r *AlarmRule) {
		r.Name = "Second Rule"
		r.AlarmCode = "ALM002"
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/rules", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[AlarmRule]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Items, 2)
}

func TestRuleHandler_ListRules_Empty(t *testing.T) {
	_, _, router := setupRuleHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/rules", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ListResponse[AlarmRule]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(0), resp.Total)
}

func TestRuleHandler_ListRules_WithFilters(t *testing.T) {
	_, repo, router := setupRuleHandlerTest()

	seedRule(repo, func(r *AlarmRule) { r.Carrier = "cmcc" })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet,
		"/rules?carrier=cmcc&enabled=true&condition_type=threshold", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------- GetRule ----------

func TestRuleHandler_GetRule_Found(t *testing.T) {
	_, repo, router := setupRuleHandlerTest()

	rule := seedRule(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet,
		fmt.Sprintf("/rules/%s", rule.ID.String()), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var got AlarmRule
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, rule.ID, got.ID)
	assert.Equal(t, rule.Name, got.Name)
}

func TestRuleHandler_GetRule_NotFound(t *testing.T) {
	_, _, router := setupRuleHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet,
		fmt.Sprintf("/rules/%s", uuid.New().String()), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRuleHandler_GetRule_InvalidUUID(t *testing.T) {
	_, _, router := setupRuleHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/rules/bad-uuid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------- CreateRule ----------

func TestRuleHandler_CreateRule_Success(t *testing.T) {
	_, repo, router := setupRuleHandlerTest()

	body := `{
		"name": "High CPU Rule",
		"alarm_code": "CPU_HIGH",
		"severity": 1,
		"condition_type": "threshold",
		"condition_config": {"max": 90},
		"action_type": "notify",
		"action_config": {"email": "ops@test.com"},
		"carrier": "cmcc",
		"technology": "lte"
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/rules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var created AlarmRule
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, "High CPU Rule", created.Name)
	assert.Equal(t, 1, created.Severity)
	assert.True(t, created.Enabled) // default enabled
	assert.NotEqual(t, uuid.Nil, created.ID)

	// Verify stored in repo
	assert.Len(t, repo.rules, 1)
}

func TestRuleHandler_CreateRule_DefaultSeverity(t *testing.T) {
	_, _, router := setupRuleHandlerTest()

	// Omit severity; handler defaults to 4.
	body := `{
		"name": "Low Prio Rule",
		"condition_type": "threshold",
		"condition_config": {"max": 50},
		"action_type": "log"
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/rules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var created AlarmRule
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, 4, created.Severity)
}

func TestRuleHandler_CreateRule_ExplicitEnabled(t *testing.T) {
	_, _, router := setupRuleHandlerTest()

	body := `{
		"name": "Disabled Rule",
		"condition_type": "threshold",
		"condition_config": {"min": 0},
		"action_type": "notify",
		"enabled": false
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/rules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var created AlarmRule
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.False(t, created.Enabled)
}

func TestRuleHandler_CreateRule_ValidationError_MissingName(t *testing.T) {
	_, _, router := setupRuleHandlerTest()

	// name is required
	body := `{
		"condition_type": "threshold",
		"condition_config": {"max": 10},
		"action_type": "notify"
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/rules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRuleHandler_CreateRule_ValidationError_MissingConditionType(t *testing.T) {
	_, _, router := setupRuleHandlerTest()

	body := `{
		"name": "Bad Rule",
		"condition_config": {"max": 10},
		"action_type": "notify"
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/rules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRuleHandler_CreateRule_InvalidJSON(t *testing.T) {
	_, _, router := setupRuleHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/rules", strings.NewReader(`{broken`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------- UpdateRule ----------

func TestRuleHandler_UpdateRule_OK(t *testing.T) {
	_, repo, router := setupRuleHandlerTest()

	rule := seedRule(repo)

	newName := "Updated Name"
	newSev := 1
	body := fmt.Sprintf(`{"name": %q, "severity": %d}`, newName, newSev)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut,
		fmt.Sprintf("/rules/%s", rule.ID.String()),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var updated AlarmRule
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updated))
	assert.Equal(t, newName, updated.Name)
	assert.Equal(t, newSev, updated.Severity)
	// Fields not included in the update request should remain unchanged.
	assert.Equal(t, rule.AlarmCode, updated.AlarmCode)
}

func TestRuleHandler_UpdateRule_NotFound(t *testing.T) {
	_, _, router := setupRuleHandlerTest()

	body := `{"name": "whatever"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut,
		fmt.Sprintf("/rules/%s", uuid.New().String()),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRuleHandler_UpdateRule_InvalidUUID(t *testing.T) {
	_, _, router := setupRuleHandlerTest()

	body := `{"name": "whatever"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/rules/bad-id",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRuleHandler_UpdateRule_PartialUpdate(t *testing.T) {
	_, repo, router := setupRuleHandlerTest()

	rule := seedRule(repo, func(r *AlarmRule) {
		r.Enabled = true
		r.Name = "Original"
	})

	// Only update enabled to false.
	body := `{"enabled": false}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut,
		fmt.Sprintf("/rules/%s", rule.ID.String()),
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var updated AlarmRule
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updated))
	assert.False(t, updated.Enabled)
	assert.Equal(t, "Original", updated.Name)
}

// ---------- DeleteRule ----------

func TestRuleHandler_DeleteRule_OK(t *testing.T) {
	_, repo, router := setupRuleHandlerTest()

	rule := seedRule(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete,
		fmt.Sprintf("/rules/%s", rule.ID.String()), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, repo.rules, 0)
}

func TestRuleHandler_DeleteRule_NotFound(t *testing.T) {
	_, _, router := setupRuleHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete,
		fmt.Sprintf("/rules/%s", uuid.New().String()), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRuleHandler_DeleteRule_InvalidUUID(t *testing.T) {
	_, _, router := setupRuleHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/rules/bad-uuid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
