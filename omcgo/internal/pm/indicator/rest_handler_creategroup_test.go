package indicator

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// stubGroupRepo 是 CreateGroup 用的最小 GroupRepository：只记录 Create 入参，
// 其余方法留空（本测试不触发）。
type stubGroupRepo struct {
	created    *IndicatorGroup
	createdDT  DeviceType
	createErr  error
	createCall bool
}

func (s *stubGroupRepo) Create(_ context.Context, dt DeviceType, group *IndicatorGroup) error {
	s.createCall = true
	s.createdDT = dt
	s.created = group
	return s.createErr
}

// ── unused stubs (interface satisfaction only) ──────────────────────────────
func (s *stubGroupRepo) List(_ context.Context, _ DeviceType) ([]*IndicatorGroup, error) {
	return nil, nil
}
func (s *stubGroupRepo) ListByPlatform(_ context.Context, _ DeviceType, _ string) ([]*IndicatorGroup, error) {
	return nil, nil
}
func (s *stubGroupRepo) GetByID(_ context.Context, _ DeviceType, _ string) (*IndicatorGroup, error) {
	return nil, nil
}
func (s *stubGroupRepo) Update(_ context.Context, _ DeviceType, _ string, _ *UpdateGroupRequest) error {
	return nil
}
func (s *stubGroupRepo) Delete(_ context.Context, _ DeviceType, _ string, _ pgx.Tx) error {
	return nil
}
func (s *stubGroupRepo) CountIndicatorsByGroup(_ context.Context, _ DeviceType) (map[string]int64, error) {
	return nil, nil
}

func newRESTHandlerWithGroupRepo(repo GroupRepository) *RESTHandler {
	svc := &IndicatorManagementService{groupRepo: repo}
	return NewRESTHandler(svc, nil, zap.NewNop())
}

func setupRESTRouter(h *RESTHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

// TestRESTCreateGroup_QueryDeviceTypeOnly 回归 issue#145 C 项：
// POST /indicator-groups?deviceType=enb 且 body 不带 device_type 时应 201，
// 而非 400（旧 binding:required 在 query 注入前触发误拒）。query 为单一来源。
func TestRESTCreateGroup_QueryDeviceTypeOnly(t *testing.T) {
	repo := &stubGroupRepo{}
	r := setupRESTRouter(newRESTHandlerWithGroupRepo(repo))

	w := httptest.NewRecorder()
	body := `{"parent_id":"default","en_name":"my_group"}`
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/indicator-groups?deviceType=enb", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, "query 注入 deviceType 应足够,无需 body 也带 device_type; body=%s", w.Body.String())
	require.True(t, repo.createCall)
	assert.Equal(t, DeviceTypeENB, repo.createdDT)
	assert.Equal(t, "my_group", repo.created.EnName)
}

// TestRESTCreateGroup_QueryAndBodyConsistent 确认 body 也带 device_type 时仍 201,
// 且 query 注入覆盖生效（query 为权威来源）。
func TestRESTCreateGroup_QueryAndBodyConsistent(t *testing.T) {
	repo := &stubGroupRepo{}
	r := setupRESTRouter(newRESTHandlerWithGroupRepo(repo))

	w := httptest.NewRecorder()
	body := `{"device_type":"GNB","parent_id":"default","en_name":"g2"}`
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/indicator-groups?deviceType=enb", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, "body=%s", w.Body.String())
	// query=enb 覆盖 body=GNB。
	assert.Equal(t, DeviceTypeENB, repo.createdDT)
}

// TestRESTCreateGroup_MissingDeviceTypeQuery 确认 query 与 body 都缺 deviceType 时 400
// （由 parseDeviceTypeQuery 在 bind 前拦截）。
func TestRESTCreateGroup_MissingDeviceTypeQuery(t *testing.T) {
	repo := &stubGroupRepo{}
	r := setupRESTRouter(newRESTHandlerWithGroupRepo(repo))

	w := httptest.NewRecorder()
	body := `{"parent_id":"default"}`
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/indicator-groups", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, repo.createCall)
}

// TestRESTCreateGroup_InvalidBodyDeviceTypeEnum 确认 body 带非法枚举值时仍被 oneof 校验拦截 400
// （omitempty 只放过空值,非空非法值照拒）。
func TestRESTCreateGroup_InvalidBodyDeviceTypeEnum(t *testing.T) {
	repo := &stubGroupRepo{}
	r := setupRESTRouter(newRESTHandlerWithGroupRepo(repo))

	w := httptest.NewRecorder()
	body := `{"device_type":"WIFI","parent_id":"default"}`
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/indicator-groups?deviceType=enb", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, repo.createCall)
}

// TestService_CreateGroup_InvalidDeviceTypeMapsBadRequest 确认 service 对空/非法 device_type
// 返回可被 HTTPStatusFromError 映射为 400 的错误（ErrInvalidInput），而非裸 error 落 500。
// 覆盖 indicatormg(handler.go) 入口靠 body 传 device_type 的兜底路径。
func TestService_CreateGroup_InvalidDeviceTypeMapsBadRequest(t *testing.T) {
	cases := []struct {
		name       string
		deviceType string
	}{
		{"empty", ""},
		{"unknown", "WIFI"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &stubGroupRepo{}
			svc := &IndicatorManagementService{groupRepo: repo}

			_, err := svc.CreateGroup(context.Background(), &CreateGroupRequest{
				DeviceType: tc.deviceType,
				ParentID:   "default",
			})
			require.Error(t, err)
			assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput),
				"应包装 ErrInvalidInput 以映射 400, got: %v", err)
			assert.Equal(t, http.StatusBadRequest, commonerrors.HTTPStatusFromError(err))
			assert.False(t, repo.createCall, "非法 device_type 不应触达 repo")
		})
	}
}

// TestService_CreateGroup_ValidDeviceType 确认合法 device_type 正常落库（成功路径）。
func TestService_CreateGroup_ValidDeviceType(t *testing.T) {
	repo := &stubGroupRepo{}
	svc := &IndicatorManagementService{groupRepo: repo}

	g, err := svc.CreateGroup(context.Background(), &CreateGroupRequest{
		DeviceType: "ENB",
		ParentID:   "default",
		EnName:     "ok_group",
	})
	require.NoError(t, err)
	require.True(t, repo.createCall)
	assert.Equal(t, DeviceTypeENB, repo.createdDT)
	assert.Equal(t, "ok_group", g.EnName)
	assert.Equal(t, "0", g.IsBuildIn)
}
