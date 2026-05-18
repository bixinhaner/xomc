package license

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/global"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// mockSystemLicenseRepo — 进程内 mock，覆盖 SystemLicenseRepository 全部 4 个方法。
//
// 行为：
//   - current 字段模拟 system_license 唯一行；nil = ErrSystemLicenseNotFound
//   - history 切片模拟 system_license_history
//   - existingIDs 模拟 license_id 唯一约束（current.LicenseID + history.LicenseID 集合）
type mockSystemLicenseRepo struct {
	current   *SystemLicense
	history   []SystemLicenseHistory
	replaceFn func(ctx context.Context, lic *SystemLicense) (*SystemLicenseHistory, error)
	existsFn  func(ctx context.Context, id string) (bool, error)
}

func (m *mockSystemLicenseRepo) GetCurrent(_ context.Context) (*SystemLicense, error) {
	if m.current == nil {
		return nil, ErrSystemLicenseNotFound
	}
	return m.current, nil
}

func (m *mockSystemLicenseRepo) Replace(ctx context.Context, lic *SystemLicense) (*SystemLicenseHistory, error) {
	if m.replaceFn != nil {
		return m.replaceFn(ctx, lic)
	}
	var hist *SystemLicenseHistory
	if m.current != nil {
		hist = &SystemLicenseHistory{
			ID:        uuid.New(),
			LicenseID: m.current.LicenseID,
		}
		m.history = append(m.history, *hist)
	}
	if lic.ID == uuid.Nil {
		lic.ID = uuid.New()
	}
	m.current = lic
	return hist, nil
}

func (m *mockSystemLicenseRepo) ListHistory(_ context.Context, _ SystemLicenseHistoryFilter) (*model.ListResponse[SystemLicenseHistory], error) {
	return &model.ListResponse[SystemLicenseHistory]{
		Items: m.history,
		Total: int64(len(m.history)),
	}, nil
}

func (m *mockSystemLicenseRepo) ExistsByLicenseID(ctx context.Context, id string) (bool, error) {
	if m.existsFn != nil {
		return m.existsFn(ctx, id)
	}
	if m.current != nil && m.current.LicenseID == id {
		return true, nil
	}
	for _, h := range m.history {
		if h.LicenseID == id {
			return true, nil
		}
	}
	return false, nil
}

// validLicenseJSON 是一个最小可用的 license JSON 文件，覆盖必填字段。
//
// 不带 signature/signature_key_id —— service 不强制验签（除非 verifier strict=true）。
func validLicenseJSON(t *testing.T, licenseID string) string {
	t.Helper()
	doc := map[string]any{
		"license_id":   licenseID,
		"license_type": string(SystemLicenseTypeCommercial),
		"issuer":       "Test OEM",
		"licensee":     "Test Customer",
		"issued_at":    "2026-05-18T00:00:00Z",
		"expiry_date":  "2049-01-01T00:00:00Z",
		"devices_support": map[string]int{
			"eNB": 10000,
			"gNB": 10000,
		},
		"feature_list": map[string]any{
			"Dashboard": "All",
		},
	}
	b, err := json.Marshal(doc)
	require.NoError(t, err)
	return string(b)
}

func newTestSystemLicenseService(repo SystemLicenseRepository) *SystemLicenseService {
	return NewSystemLicenseService(repo, zap.NewNop())
}

func TestSystemLicenseService_GetCurrent(t *testing.T) {
	tests := []struct {
		name      string
		seed      *SystemLicense
		wantNil   bool
		wantCode  int  // 0 = no business error expected
		wantNotFd bool // expect commonerrors.ErrNotFound chain
	}{
		{
			name:      "empty table returns 12113 NotConfigured",
			seed:      nil,
			wantNil:   true,
			wantCode:  global.ErrCodeSystemLicenseNotConfigured,
			wantNotFd: true,
		},
		{
			name: "current exists returns it",
			seed: &SystemLicense{
				ID:          uuid.New(),
				LicenseID:   "NO2026-05-001",
				LicenseType: SystemLicenseTypeCommercial,
				IsCurrent:   true,
			},
			wantNil:  false,
			wantCode: 0,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockSystemLicenseRepo{current: tc.seed}
			svc := newTestSystemLicenseService(repo)
			got, err := svc.GetCurrent(context.Background())
			if tc.wantNil {
				assert.Nil(t, got)
				require.Error(t, err)
				var be *commonerrors.BusinessError
				require.True(t, errors.As(err, &be), "expected BusinessError, got %T", err)
				assert.Equal(t, tc.wantCode, be.Code)
				if tc.wantNotFd {
					assert.True(t, errors.Is(err, commonerrors.ErrNotFound), "expected ErrNotFound chain")
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tc.seed.LicenseID, got.LicenseID)
		})
	}
}

func TestSystemLicenseService_Update(t *testing.T) {
	tests := []struct {
		name             string
		raw              func(t *testing.T) string
		seedCurrent      *SystemLicense
		seedExists       func(ctx context.Context, id string) (bool, error)
		wantErrCode      int  // 0 = no BusinessError expected
		wantReplacedNil  bool // first install should have nil Replaced
		wantSigStatus    SignatureStatus
		wantLicenseIDOut string
	}{
		{
			name: "first install OK",
			raw: func(t *testing.T) string {
				return validLicenseJSON(t, "NO2026-05-001")
			},
			seedCurrent:      nil,
			wantErrCode:      0,
			wantReplacedNil:  true,
			wantSigStatus:    SignatureUnverified,
			wantLicenseIDOut: "NO2026-05-001",
		},
		{
			name: "replace existing OK",
			raw: func(t *testing.T) string {
				return validLicenseJSON(t, "NO2026-05-002")
			},
			seedCurrent: &SystemLicense{
				ID:          uuid.New(),
				LicenseID:   "NO2026-05-001",
				LicenseType: SystemLicenseTypeCommercial,
				IsCurrent:   true,
			},
			wantErrCode:      0,
			wantReplacedNil:  false,
			wantSigStatus:    SignatureUnverified,
			wantLicenseIDOut: "NO2026-05-002",
		},
		{
			name: "empty raw_content rejected as InvalidFormat",
			raw: func(t *testing.T) string {
				return "   "
			},
			wantErrCode: global.ErrCodeSystemLicenseInvalidFormat,
		},
		{
			name: "invalid JSON rejected as InvalidFormat",
			raw: func(t *testing.T) string {
				return `{"license_id":`
			},
			wantErrCode: global.ErrCodeSystemLicenseInvalidFormat,
		},
		{
			name: "missing license_id rejected as InvalidFormat",
			raw: func(t *testing.T) string {
				return `{"license_type":"Commercial","issued_at":"2026-05-18T00:00:00Z"}`
			},
			wantErrCode: global.ErrCodeSystemLicenseInvalidFormat,
		},
		{
			name: "invalid license_type rejected as InvalidFormat",
			raw: func(t *testing.T) string {
				return `{"license_id":"X","license_type":"Bogus","issued_at":"2026-05-18T00:00:00Z"}`
			},
			wantErrCode: global.ErrCodeSystemLicenseInvalidFormat,
		},
		{
			name: "missing issued_at rejected as InvalidFormat",
			raw: func(t *testing.T) string {
				return `{"license_id":"X","license_type":"Commercial"}`
			},
			wantErrCode: global.ErrCodeSystemLicenseInvalidFormat,
		},
		{
			name: "license_id already exists (pre-check) → 12110",
			raw: func(t *testing.T) string {
				return validLicenseJSON(t, "NO2022-03-14002")
			},
			seedExists: func(_ context.Context, _ string) (bool, error) {
				return true, nil
			},
			wantErrCode: global.ErrCodeSystemLicenseIDExists,
		},
		{
			name: "license_id exists raced from repo.Replace → 12110",
			raw: func(t *testing.T) string {
				return validLicenseJSON(t, "NO2026-05-XXX")
			},
			seedExists: func(_ context.Context, _ string) (bool, error) {
				return false, nil // pre-check passes
			},
			// 模拟 pre-check 后并发上传：Replace 撞 UNIQUE。
			wantErrCode: global.ErrCodeSystemLicenseIDExists,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockSystemLicenseRepo{
				current:  tc.seedCurrent,
				existsFn: tc.seedExists,
			}
			// 给"race"用例注入 Replace 错误。
			if tc.name == "license_id exists raced from repo.Replace → 12110" {
				repo.replaceFn = func(_ context.Context, _ *SystemLicense) (*SystemLicenseHistory, error) {
					return nil, ErrSystemLicenseIDExists
				}
			}

			svc := newTestSystemLicenseService(repo)
			result, err := svc.Update(context.Background(), UpdateRequest{
				RawContent: tc.raw(t),
			})

			if tc.wantErrCode != 0 {
				require.Error(t, err)
				var be *commonerrors.BusinessError
				require.True(t, errors.As(err, &be), "expected BusinessError, got %T: %v", err, err)
				assert.Equal(t, tc.wantErrCode, be.Code)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, result.Current)
			assert.Equal(t, tc.wantLicenseIDOut, result.Current.LicenseID)
			assert.Equal(t, tc.wantSigStatus, result.Current.SignatureStatus)
			if tc.wantReplacedNil {
				assert.Nil(t, result.Replaced)
			} else {
				assert.NotNil(t, result.Replaced)
			}
			// raw_content 完整存档
			assert.NotEmpty(t, result.Current.RawContent)
		})
	}
}

func TestSystemLicenseService_Update_StoresUploaderAndUploadedAt(t *testing.T) {
	repo := &mockSystemLicenseRepo{}
	svc := newTestSystemLicenseService(repo)
	uploader := uuid.New()
	fixedNow := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	orig := nowFunc
	defer func() { nowFunc = orig }()
	nowFunc = func() time.Time { return fixedNow }

	result, err := svc.Update(context.Background(), UpdateRequest{
		RawContent:       validLicenseJSON(t, "NO2026-05-099"),
		UploadedByUserID: &uploader,
	})
	require.NoError(t, err)
	require.NotNil(t, result.Current.UploadedByUserID)
	assert.Equal(t, uploader, *result.Current.UploadedByUserID)
	assert.True(t, result.Current.UploadedAt.Equal(fixedNow), "uploaded_at should be deterministic")
}

func TestSystemLicenseService_ListHistory_Passthrough(t *testing.T) {
	repo := &mockSystemLicenseRepo{
		history: []SystemLicenseHistory{
			{ID: uuid.New(), LicenseID: "NO2022-03-14002"},
			{ID: uuid.New(), LicenseID: "NO2022-09-01001"},
		},
	}
	svc := newTestSystemLicenseService(repo)
	resp, err := svc.ListHistory(context.Background(), SystemLicenseHistoryFilter{})
	require.NoError(t, err)
	assert.EqualValues(t, 2, resp.Total)
	assert.Len(t, resp.Items, 2)
}
