package license

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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

func configureRepositoryLegacyLicense(t *testing.T, svc *SystemLicenseService) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "omc.lic"))
	require.NoError(t, err)
	keyStore, err := os.ReadFile(filepath.Join("..", "..", "..", "license-run-time", "keystore", "omcPublicKey.store"))
	require.NoError(t, err)
	svc.SetLegacyTrueLicenseConfig(keyStore, "bcb9omc6", "omcPublicKey", false)
	return base64.StdEncoding.EncodeToString(raw)
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

func TestSystemLicenseService_LegacyConfigTrimsLineEndings(t *testing.T) {
	svc := newTestSystemLicenseService(&mockSystemLicenseRepo{})
	svc.SetLegacyTrueLicenseConfig([]byte{1}, "bcb9omc6\r\n", "omcPublicKey", true)
	assert.Equal(t, "bcb9omc6", svc.legacyStorePassword)
}

func TestSystemLicenseService_CheckFeature(t *testing.T) {
	repo := &mockSystemLicenseRepo{current: &SystemLicense{
		LicenseID:   "NO2026-05-001",
		LicenseType: SystemLicenseTypeCommercial,
		FeatureList: FeatureList(`{"eNB":{"Monitor":["Settings"]}}`),
		IsCurrent:   true,
	}}
	svc := newTestSystemLicenseService(repo)

	authorized, err := svc.CheckFeature(context.Background(), "eNB.Monitor.Settings")
	require.NoError(t, err)
	assert.True(t, authorized)

	authorized, err = svc.CheckFeature(context.Background(), "eNB.Monitor.Active")
	require.NoError(t, err)
	assert.False(t, authorized)

	_, err = svc.CheckFeature(context.Background(), "eNB..Settings")
	require.Error(t, err)
}

func TestSystemLicenseService_GetCurrentEnrichesPersistedLegacyFeatures(t *testing.T) {
	repo := &mockSystemLicenseRepo{current: &SystemLicense{
		LicenseID:   "NO2022-03-14002",
		LicenseType: SystemLicenseTypeCommercial,
		FeatureList: FeatureList(`{"features":[{"name_zh":"旧名称"}],"legacy_feature_ids":["80"],"legacy_feature_codes":["CODE_TOOL_DHCP"]}`),
		IsCurrent:   true,
	}}
	svc := newTestSystemLicenseService(repo)
	mapping, err := LoadLegacyFeatureMapping(filepath.Join("..", "..", "data", "license-feature-mapping.json"))
	require.NoError(t, err)
	svc.SetLegacyFeatureMapping(mapping)

	license, err := svc.GetCurrent(context.Background())
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(license.FeatureList, &payload))
	features, ok := payload["features"].([]any)
	require.True(t, ok)
	require.Len(t, features, 1)
	feature := features[0].(map[string]any)
	assert.Equal(t, "DHCP", feature["name_zh"])
	assert.Equal(t, "高级 / DHCP", feature["path"])
}

func TestSystemLicenseService_Update(t *testing.T) {
	t.Run("legacy lic first install", func(t *testing.T) {
		repo := &mockSystemLicenseRepo{}
		svc := newTestSystemLicenseService(repo)
		raw := configureRepositoryLegacyLicense(t, svc)

		result, err := svc.Update(context.Background(), UpdateRequest{
			RawContent:         raw,
			RawContentEncoding: "base64",
		})
		require.NoError(t, err)
		assert.Equal(t, "NO2022-03-14002", result.Current.LicenseID)
		assert.Equal(t, SignatureVerified, result.Current.SignatureStatus)
		assert.Nil(t, result.Replaced)
		assert.NotEmpty(t, result.Current.RawContent)
	})

	t.Run("json license is rejected", func(t *testing.T) {
		svc := newTestSystemLicenseService(&mockSystemLicenseRepo{})
		result, err := svc.Update(context.Background(), UpdateRequest{
			RawContent: `{"license_id":"not-a-legacy-license"}`,
		})
		assert.Nil(t, result)
		var businessErr *commonerrors.BusinessError
		require.ErrorAs(t, err, &businessErr)
		assert.Equal(t, global.ErrCodeSystemLicenseInvalidFormat, businessErr.Code)
	})

	t.Run("legacy feature payload includes normalized display objects", func(t *testing.T) {
		repo := &mockSystemLicenseRepo{}
		svc := newTestSystemLicenseService(repo)
		mapping, mappingErr := LoadLegacyFeatureMapping(filepath.Join("..", "..", "data", "license-feature-mapping.json"))
		require.NoError(t, mappingErr)
		svc.SetLegacyFeatureMapping(mapping)
		result, err := svc.Update(context.Background(), UpdateRequest{
			RawContent:         configureRepositoryLegacyLicense(t, svc),
			RawContentEncoding: "base64",
		})
		require.NoError(t, err)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(result.Current.FeatureList, &payload))
		features, ok := payload["features"].([]any)
		require.True(t, ok)
		require.NotEmpty(t, features)
	})
}

func TestSystemLicenseService_Update_StoresUploaderAndUploadedAt(t *testing.T) {
	repo := &mockSystemLicenseRepo{}
	svc := newTestSystemLicenseService(repo)
	uploader := uuid.New()
	fixedNow := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	orig := nowFunc
	defer func() { nowFunc = orig }()
	nowFunc = func() time.Time { return fixedNow }

	raw := configureRepositoryLegacyLicense(t, svc)
	result, err := svc.Update(context.Background(), UpdateRequest{
		RawContent:         raw,
		RawContentEncoding: "base64",
		UploadedByUserID:   &uploader,
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
