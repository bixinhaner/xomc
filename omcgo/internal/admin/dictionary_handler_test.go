package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// =============================================================
// Mock Repository
// =============================================================

type mockDictRepository struct {
	dicts map[string]*Dictionary
}

func newMockDictRepository() *mockDictRepository {
	return &mockDictRepository{
		dicts: make(map[string]*Dictionary),
	}
}

func (r *mockDictRepository) GetByType(ctx context.Context, dictType string) (*Dictionary, error) {
	dict, ok := r.dicts[dictType]
	if !ok {
		return nil, commonerrors.NewBusinessError(7002, "dictionary not found", nil)
	}
	return dict, nil
}

func (r *mockDictRepository) Create(ctx context.Context, dict *Dictionary) error {
	r.dicts[dict.Type] = dict
	return nil
}

func (r *mockDictRepository) Update(ctx context.Context, dict *Dictionary) error {
	r.dicts[dict.Type] = dict
	return nil
}

func (r *mockDictRepository) Delete(ctx context.Context, id int64) error {
	for typ, dict := range r.dicts {
		if dict.ID == id {
			delete(r.dicts, typ)
			return nil
		}
	}
	// 镜像 PgDictionaryRepository.Delete:不存在记录返 sentinel ErrNotFound,
	// 以便 handler 经 HTTPStatusFromError 映射 404(issue #145 D)。
	return commonerrors.ErrNotFound
}

func (r *mockDictRepository) List(ctx context.Context) ([]Dictionary, error) {
	result := make([]Dictionary, 0, len(r.dicts))
	for _, dict := range r.dicts {
		result = append(result, *dict)
	}
	return result, nil
}

func (r *mockDictRepository) GetByID(ctx context.Context, id int64) (*Dictionary, error) {
	for _, dict := range r.dicts {
		if dict.ID == id {
			return dict, nil
		}
	}
	// 镜像 PgDictionaryRepository.GetByID:不存在返 sentinel ErrNotFound。
	// UpdateDictionary 先 GetByID,经 %w 包装后仍被 HTTPStatusFromError 识别为 404。
	return nil, commonerrors.ErrNotFound
}

type mockDictDetailRepository struct {
	details []DictionaryDetail
	// notFound 中的 id 让 Delete/GetByID 返回 sentinel ErrNotFound,
	// 镜像 PgDictionaryDetailRepository 对不存在记录的行为(issue #145 D)。
	// 默认空 → 保持「永远命中」的旧行为,不影响既有用例。
	notFound map[int64]bool
}

func newMockDictDetailRepository() *mockDictDetailRepository {
	return &mockDictDetailRepository{
		details:  make([]DictionaryDetail, 0),
		notFound: make(map[int64]bool),
	}
}

func (r *mockDictDetailRepository) GetByDictID(ctx context.Context, dictID int64) ([]DictionaryDetail, error) {
	return r.details, nil
}

func (r *mockDictDetailRepository) Create(ctx context.Context, detail *DictionaryDetail) error {
	r.details = append(r.details, *detail)
	return nil
}

func (r *mockDictDetailRepository) Update(ctx context.Context, detail *DictionaryDetail) error {
	return nil
}

func (r *mockDictDetailRepository) Delete(ctx context.Context, id int64) error {
	if r.notFound[id] {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *mockDictDetailRepository) GetByID(ctx context.Context, id int64) (*DictionaryDetail, error) {
	if r.notFound[id] {
		return nil, commonerrors.ErrNotFound
	}
	return &DictionaryDetail{ID: id}, nil
}

func (r *mockDictDetailRepository) List(ctx context.Context, req DictionaryDetailListRequest) ([]DictionaryDetail, int64, error) {
	return r.details, int64(len(r.details)), nil
}

func (r *mockDictDetailRepository) ListSubtreeIDs(ctx context.Context, rootID int64) ([]int64, error) {
	return []int64{rootID}, nil
}

func (r *mockDictDetailRepository) ShiftLevelDelta(ctx context.Context, ids []int64, delta int) error {
	return nil
}

func (r *mockDictDetailRepository) DeleteByDictionaryID(ctx context.Context, dictID int64) error {
	return nil
}

// T-0182 数据源同步专用 — no-op shims,handler/service 老测试不触发这些路径。
func (r *mockDictRepository) ListSourceBound(ctx context.Context) ([]Dictionary, error) {
	return nil, nil
}
func (r *mockDictRepository) UpdateRefreshMetadata(ctx context.Context, dictID int64, status, errMsg string, count int) error {
	return nil
}
func (r *mockDictDetailRepository) UpsertAutoBatch(ctx context.Context, dictID int64, rows []AutoDetailRow) (int, int, error) {
	return 0, 0, nil
}
func (r *mockDictDetailRepository) DeleteAutoNotIn(ctx context.Context, dictID int64, keepValues []string) (int, error) {
	return 0, nil
}
func (r *mockDictDetailRepository) CountAutoActive(ctx context.Context, dictID int64) (int, error) {
	return 0, nil
}

// =============================================================
// BatchGetDicts Tests
// =============================================================

func TestBatchGetDicts(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	dictRepo := newMockDictRepository()
	detailRepo := newMockDictDetailRepository()
	service := NewDictionaryService(dictRepo, detailRepo)
	handler := NewDictionaryHandler(service)

	// Add test dictionaries
	dictRepo.dicts["is_online"] = &Dictionary{ID: 1, Type: "is_online", Name: "在线状态"}
	dictRepo.dicts["op_state"] = &Dictionary{ID: 2, Type: "op_state", Name: "运行状态"}
	dictRepo.dicts["network_type"] = &Dictionary{ID: 3, Type: "network_type", Name: "网络类型"}

	router := gin.New()
	handler.RegisterRoutes(router.Group("/admin"))

	tests := []struct {
		name           string
		codes          string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "批量查询3个字典",
			codes:          "is_online,op_state,network_type",
			expectedStatus: http.StatusOK,
			expectedCount:  3,
		},
		{
			name:           "批量查询2个字典",
			codes:          "is_online,op_state",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "空codes参数",
			codes:          "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "只有逗号",
			codes:          ",,",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "带空格的codes",
			codes:          "is_online, op_state , network_type",
			expectedStatus: http.StatusOK,
			expectedCount:  3,
		},
		{
			name:           "部分不存在的字典",
			codes:          "is_online,not_exist,op_state",
			expectedStatus: http.StatusOK,
			expectedCount:  2, // 只返回存在的字典
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// URL encode the codes parameter
			reqURL := "/admin/sysDictionary/batch?codes=" + url.QueryEscape(tt.codes)
			req := httptest.NewRequest("GET", reqURL, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				// Check response structure
				_, ok := response["data"]
				require.True(t, ok, "response should have data field, got: %v", response)

				// data is wrapped by response.OKWithMsg
				data := response["data"].(map[string]interface{})

				dicts, ok := data["dicts"].(map[string]interface{})
				require.True(t, ok, "dicts should be a map, got: %v", data)

				assert.Equal(t, tt.expectedCount, len(dicts))
			}
		})
	}
}

// =============================================================
// 错误映射回归 — issue #145 D:不存在记录的 DELETE/PUT 应返 404 而非 500
// =============================================================
//
// 修复前 dictionary_handler.go 对 service error 硬编码
// AbortWithError(c, http.StatusInternalServerError, err),即便 repo 返
// commonerrors.ErrNotFound 也落 500。修复改走 HTTPStatusFromError 映射。

func newDictTestRouter() (*gin.Engine, *mockDictRepository, *mockDictDetailRepository) {
	gin.SetMode(gin.TestMode)
	dictRepo := newMockDictRepository()
	detailRepo := newMockDictDetailRepository()
	service := NewDictionaryService(dictRepo, detailRepo)
	handler := NewDictionaryHandler(service)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/admin"))
	return router, dictRepo, detailRepo
}

func TestDictionaryHandler_NotFoundMapsTo404(t *testing.T) {
	tests := []struct {
		name   string
		method string
		// path 为完整 URL(含 query);body 非空则作为 JSON 请求体。
		path string
		body string
		// setup 可向 mock 仓库注入「不存在 id」语义;nil 表示沿用空仓库
		// (此时所有 id 天然不存在)。
		setup func(dictRepo *mockDictRepository, detailRepo *mockDictDetailRepository)
	}{
		{
			name:   "DELETE 不存在字典 → 404",
			method: http.MethodDelete,
			path:   "/admin/sysDictionary/deleteSysDictionary?id=99999",
		},
		{
			name:   "PUT 不存在字典 → 404",
			method: http.MethodPut,
			path:   "/admin/sysDictionary/updateSysDictionary",
			body:   `{"id":99999,"name":"x"}`,
		},
		{
			name:   "DELETE 不存在字典明细 → 404",
			method: http.MethodDelete,
			path:   "/admin/sysDictionaryDetail/deleteSysDictionaryDetail?id=99999",
			setup: func(_ *mockDictRepository, detailRepo *mockDictDetailRepository) {
				detailRepo.notFound[99999] = true
			},
		},
		{
			name:   "PUT 不存在字典明细 → 404",
			method: http.MethodPut,
			path:   "/admin/sysDictionaryDetail/updateSysDictionaryDetail",
			body:   `{"id":99999,"label":"x"}`,
			setup: func(_ *mockDictRepository, detailRepo *mockDictDetailRepository) {
				detailRepo.notFound[99999] = true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, dictRepo, detailRepo := newDictTestRouter()
			if tt.setup != nil {
				tt.setup(dictRepo, detailRepo)
			}

			var bodyReader io.Reader
			if tt.body != "" {
				bodyReader = bytes.NewBufferString(tt.body)
			}
			req := httptest.NewRequest(tt.method, tt.path, bodyReader)
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusNotFound, w.Code,
				"不存在记录应返回 404,body=%s", w.Body.String())
		})
	}
}

// TestDictionaryHandler_DeleteSuccess 确认正常删除仍返 200(防止把成功路径误映射)。
func TestDictionaryHandler_DeleteSuccess(t *testing.T) {
	router, dictRepo, _ := newDictTestRouter()
	dictRepo.dicts["gender"] = &Dictionary{ID: 42, Type: "gender", Name: "性别"}

	req := httptest.NewRequest(http.MethodDelete, "/admin/sysDictionary/deleteSysDictionary?id=42", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "存在记录删除应返回 200,body=%s", w.Body.String())
}

// =============================================================
// Helper Functions
// =============================================================

func requireJSONUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

