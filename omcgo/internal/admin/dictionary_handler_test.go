package admin

import (
	"context"
	"encoding/json"
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
	return commonerrors.NewBusinessError(7002, "dictionary not found", nil)
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
	return nil, commonerrors.NewBusinessError(7002, "dictionary not found", nil)
}

type mockDictDetailRepository struct {
	details []DictionaryDetail
}

func newMockDictDetailRepository() *mockDictDetailRepository {
	return &mockDictDetailRepository{
		details: make([]DictionaryDetail, 0),
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
	return nil
}

func (r *mockDictDetailRepository) GetByID(ctx context.Context, id int64) (*DictionaryDetail, error) {
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
// Helper Functions
// =============================================================

func requireJSONUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

