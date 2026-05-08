package parammodel

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/product"
)

// fakeIntersectRepo 捕获最近一次 UpsertDiscoveredMappings 调用，便于断言。
type fakeIntersectRepo struct {
	upsertErr     error
	calls         int
	lastProductID uuid.UUID
	lastSwVersion string
	lastMappings  []ParamMapping
}

func (f *fakeIntersectRepo) UpsertDiscoveredMappings(_ context.Context, productID uuid.UUID, swVersion string, mappings []ParamMapping) error {
	f.calls++
	if f.upsertErr != nil {
		return f.upsertErr
	}
	f.lastProductID = productID
	f.lastSwVersion = swVersion
	f.lastMappings = mappings
	return nil
}

// fakeInvalidator 捕获 InvalidateProduct 调用次数。
type fakeInvalidator struct {
	calls int
	err   error
}

func (f *fakeInvalidator) InvalidateProduct(_ context.Context, _ uuid.UUID, _ string) error {
	f.calls++
	return f.err
}

// helper: 构造一个含 paramModelID 的 fakeProductGetter。
func newProductWith(paramModelID *uuid.UUID, override map[string]any) (*fakeProductGetter, uuid.UUID) {
	pid := uuid.New()
	pg := newFakeProductGetter()
	pg.products[pid] = &product.Product{
		ID:                  pid,
		Name:                "TestProduct",
		ParamModelID:        paramModelID,
		DeviceAttrsOverride: override,
	}
	return pg, pid
}

func mkInt64(v int64) *int64 { return &v }

// 默认 paramModel 含 5 条 mappings（4 parameter + 1 object）。
func defaultMappingsFixture() []ParamMapping {
	return []ParamMapping{
		{ID: uuid.New(), StandardPath: "Device.WiFi.SSID.{i}.Enable", PrivatePath: "Dev.WiFi.SSID.{i}.Enabled", EntryType: "parameter", Access: "readWrite", DataType: "boolean", ChangeApplies: "atOnce", IsStorable: true, IsActive: true},
		{ID: uuid.New(), StandardPath: "Device.WiFi.SSID.{i}.SSIDName", PrivatePath: "Dev.WiFi.SSID.{i}.Name", EntryType: "parameter", Access: "readWrite", DataType: "string", IsStorable: true, IsActive: true, MinValue: mkInt64(1), MaxValue: mkInt64(32)},
		{ID: uuid.New(), StandardPath: "Device.WiFi.Radio.{i}.Channel", PrivatePath: "Dev.WiFi.Radio.{i}.Ch", EntryType: "parameter", Access: "readWrite", DataType: "unsignedInt", IsStorable: true, IsActive: true, MinValue: mkInt64(1), MaxValue: mkInt64(13)},
		{ID: uuid.New(), StandardPath: "Device.WiFi.SSID.", PrivatePath: "Dev.WiFi.SSID.", EntryType: "object", Access: "readOnly", IsStorable: false, IsActive: true},
		{ID: uuid.New(), StandardPath: "Device.System.Mode", PrivatePath: "Dev.Sys.Mode", EntryType: "parameter", Access: "readOnly", DataType: "string", IsStorable: false, IsActive: true},
	}
}

// CPE 上传 4 条（其中 3 条匹配默认，1 条是默认没有的）
func cpeEntriesFixture() []CPEEntry {
	return []CPEEntry{
		{PrivatePath: "Dev.WiFi.SSID.{i}.Enabled", EntryType: "parameter", Access: "readOnly", DataType: "boolean", ChangeApplies: "byReboot", MinValue: mkInt64(0), MaxValue: mkInt64(1)},
		{PrivatePath: "Dev.WiFi.SSID.{i}.Name", EntryType: "parameter", Access: "readWrite", DataType: "string", ChangeApplies: "atOnce", MinValue: mkInt64(2), MaxValue: mkInt64(64)},
		{PrivatePath: "Dev.WiFi.Radio.{i}.Ch", EntryType: "parameter", Access: "readWrite", DataType: "unsignedInt", ChangeApplies: "byReboot", MinValue: mkInt64(36), MaxValue: mkInt64(165)},
		{PrivatePath: "Dev.Vendor.Extra", EntryType: "parameter", Access: "readWrite", DataType: "string"},
	}
}

func newServiceForTest(t *testing.T, defaults []ParamMapping, paramModelID uuid.UUID, override map[string]any) (
	*IntersectService, *fakeIntersectRepo, *fakeInvalidator, uuid.UUID, *fakeProductGetter,
) {
	t.Helper()
	repo := newFakeRepo()
	repo.defaultByModel[paramModelID] = defaults

	pmCopy := paramModelID
	pg, pid := newProductWith(&pmCopy, override)
	pg.products[pid].ParamModelID = &paramModelID

	wrepo := &fakeIntersectRepo{}
	inv := &fakeInvalidator{}

	svc := NewIntersectService(repo, wrepo, pg, inv, nil, zap.NewNop())
	return svc, wrepo, inv, pid, pg
}

// ── 测试 ────────────────────────────────────────────────────────────

func TestIntersect_HappyPath_NoOverride(t *testing.T) {
	pmID := uuid.New()
	svc, wrepo, inv, pid, _ := newServiceForTest(t, defaultMappingsFixture(), pmID, nil)

	res, err := svc.IntersectCPEModel(context.Background(), IntersectInput{
		ProductID:       pid,
		SoftwareVersion: "v1.2.3",
		Entries:         cpeEntriesFixture(),
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, 5, res.DefaultCount)
	assert.Equal(t, 4, res.UploadedCount)
	assert.Equal(t, 3, res.Matched)        // 3 个 path 双方都有
	assert.Equal(t, 2, res.DefaultsMissing) // SSID. + System.Mode
	assert.Equal(t, 1, res.UploadedExtras)  // Dev.Vendor.Extra
	assert.False(t, res.OverrideDataType)

	require.Equal(t, 1, wrepo.calls)
	assert.Equal(t, pid, wrepo.lastProductID)
	assert.Equal(t, "v1.2.3", wrepo.lastSwVersion)
	require.Len(t, wrepo.lastMappings, 3)
	for _, m := range wrepo.lastMappings {
		require.NotNil(t, m.SoftwareVersion)
		assert.Equal(t, "v1.2.3", *m.SoftwareVersion)
		assert.True(t, m.IsActive)
	}
	// 无 override → Access 仍取默认
	for _, m := range wrepo.lastMappings {
		if m.PrivatePath == "Dev.WiFi.SSID.{i}.Enabled" {
			assert.Equal(t, "readWrite", m.Access, "默认 access 不应被 CPE 的 readOnly 覆盖")
		}
	}
	assert.Equal(t, 1, inv.calls)
}

func TestIntersect_OverrideAccess_UsesCPEValue(t *testing.T) {
	pmID := uuid.New()
	svc, wrepo, _, pid, _ := newServiceForTest(t, defaultMappingsFixture(), pmID, map[string]any{
		"access": true,
	})

	_, err := svc.IntersectCPEModel(context.Background(), IntersectInput{
		ProductID:       pid,
		SoftwareVersion: "v1.0.0",
		Entries:         cpeEntriesFixture(),
	})
	require.NoError(t, err)

	for _, m := range wrepo.lastMappings {
		if m.PrivatePath == "Dev.WiFi.SSID.{i}.Enabled" {
			assert.Equal(t, "readOnly", m.Access, "access=true 应取 CPE 上传的 readOnly")
		}
	}
}

func TestIntersect_OverrideMinMax_UsesCPEValues(t *testing.T) {
	pmID := uuid.New()
	svc, wrepo, _, pid, _ := newServiceForTest(t, defaultMappingsFixture(), pmID, map[string]any{
		"min_value": true,
		"max_value": true,
	})

	_, err := svc.IntersectCPEModel(context.Background(), IntersectInput{
		ProductID:       pid,
		SoftwareVersion: "v1.0.0",
		Entries:         cpeEntriesFixture(),
	})
	require.NoError(t, err)

	for _, m := range wrepo.lastMappings {
		if m.PrivatePath == "Dev.WiFi.Radio.{i}.Ch" {
			require.NotNil(t, m.MinValue)
			require.NotNil(t, m.MaxValue)
			assert.EqualValues(t, 36, *m.MinValue)
			assert.EqualValues(t, 165, *m.MaxValue)
		}
	}
}

func TestIntersect_DataTypeOverride_RejectedAlwaysFalse(t *testing.T) {
	pmID := uuid.New()
	svc, wrepo, _, pid, _ := newServiceForTest(t, defaultMappingsFixture(), pmID, map[string]any{
		"data_type": true, // 业务禁止；应被强制忽略 + WARN
	})

	res, err := svc.IntersectCPEModel(context.Background(), IntersectInput{
		ProductID:       pid,
		SoftwareVersion: "v1.0.0",
		Entries: []CPEEntry{
			{PrivatePath: "Dev.WiFi.SSID.{i}.Enabled", DataType: "string"}, // 假装设备说是 string
		},
	})
	require.NoError(t, err)
	assert.True(t, res.OverrideDataType, "应记录请求被拒绝")

	for _, m := range wrepo.lastMappings {
		if m.PrivatePath == "Dev.WiFi.SSID.{i}.Enabled" {
			assert.Equal(t, "boolean", m.DataType, "data_type 永远以默认为准")
		}
	}
}

func TestIntersect_OverrideChangeApplies_UsesCPEValue(t *testing.T) {
	pmID := uuid.New()
	svc, wrepo, _, pid, _ := newServiceForTest(t, defaultMappingsFixture(), pmID, map[string]any{
		"change_applies": true,
	})

	_, err := svc.IntersectCPEModel(context.Background(), IntersectInput{
		ProductID:       pid,
		SoftwareVersion: "v1.0.0",
		Entries:         cpeEntriesFixture(),
	})
	require.NoError(t, err)

	for _, m := range wrepo.lastMappings {
		if m.PrivatePath == "Dev.WiFi.SSID.{i}.Enabled" {
			assert.Equal(t, "byReboot", m.ChangeApplies, "change_applies=true 应取 CPE 值")
		}
	}
}

func TestIntersect_EmptyCPE_ZeroMatchedButStillUpsert(t *testing.T) {
	pmID := uuid.New()
	svc, wrepo, _, pid, _ := newServiceForTest(t, defaultMappingsFixture(), pmID, nil)

	res, err := svc.IntersectCPEModel(context.Background(), IntersectInput{
		ProductID:       pid,
		SoftwareVersion: "v1.0.0",
		Entries:         nil,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, res.Matched)
	assert.Equal(t, 5, res.DefaultsMissing)
	require.Equal(t, 1, wrepo.calls, "即使空 CPE 也要 DELETE（清理旧数据）")
	assert.Empty(t, wrepo.lastMappings)
}

func TestIntersect_EmptyDefault_NoRowsWritten(t *testing.T) {
	pmID := uuid.New()
	svc, wrepo, _, pid, _ := newServiceForTest(t, []ParamMapping{}, pmID, nil)

	res, err := svc.IntersectCPEModel(context.Background(), IntersectInput{
		ProductID:       pid,
		SoftwareVersion: "v1.0.0",
		Entries:         cpeEntriesFixture(),
	})
	require.NoError(t, err)
	assert.Equal(t, 0, res.DefaultCount)
	assert.Equal(t, 0, res.Matched)
	require.Equal(t, 1, wrepo.calls)
	assert.Empty(t, wrepo.lastMappings)
}

func TestIntersect_ProductNotFound(t *testing.T) {
	svc, _, _, _, pg := newServiceForTest(t, defaultMappingsFixture(), uuid.New(), nil)
	pg.products = map[uuid.UUID]*product.Product{} // 清空

	_, err := svc.IntersectCPEModel(context.Background(), IntersectInput{
		ProductID:       uuid.New(),
		SoftwareVersion: "v1.0.0",
		Entries:         cpeEntriesFixture(),
	})
	require.ErrorIs(t, err, ErrProductNotFound)
}

func TestIntersect_NoParamModelID(t *testing.T) {
	repo := newFakeRepo()
	pg := newFakeProductGetter()
	pid := uuid.New()
	pg.products[pid] = &product.Product{ID: pid, Name: "P", ParamModelID: nil}

	svc := NewIntersectService(repo, &fakeIntersectRepo{}, pg, &fakeInvalidator{}, nil, zap.NewNop())
	_, err := svc.IntersectCPEModel(context.Background(), IntersectInput{
		ProductID:       pid,
		SoftwareVersion: "v1.0.0",
		Entries:         cpeEntriesFixture(),
	})
	require.ErrorIs(t, err, ErrNoParamModel)
}

func TestIntersect_EmptySoftwareVersion(t *testing.T) {
	pmID := uuid.New()
	svc, _, _, pid, _ := newServiceForTest(t, defaultMappingsFixture(), pmID, nil)

	_, err := svc.IntersectCPEModel(context.Background(), IntersectInput{
		ProductID:       pid,
		SoftwareVersion: "",
		Entries:         cpeEntriesFixture(),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "software_version is empty")
}

func TestIntersect_ProductGetterUnset(t *testing.T) {
	svc := NewIntersectService(newFakeRepo(), &fakeIntersectRepo{}, nil, &fakeInvalidator{}, nil, zap.NewNop())
	_, err := svc.IntersectCPEModel(context.Background(), IntersectInput{
		ProductID:       uuid.New(),
		SoftwareVersion: "v1.0.0",
		Entries:         cpeEntriesFixture(),
	})
	require.ErrorIs(t, err, ErrProductGetterUnset)
}

func TestIntersect_UpsertError_Propagated(t *testing.T) {
	pmID := uuid.New()
	repo := newFakeRepo()
	repo.defaultByModel[pmID] = defaultMappingsFixture()

	pmCopy := pmID
	pg, pid := newProductWith(&pmCopy, nil)

	wrepo := &fakeIntersectRepo{upsertErr: errors.New("db down")}
	svc := NewIntersectService(repo, wrepo, pg, &fakeInvalidator{}, nil, zap.NewNop())

	_, err := svc.IntersectCPEModel(context.Background(), IntersectInput{
		ProductID:       pid,
		SoftwareVersion: "v1.0.0",
		Entries:         cpeEntriesFixture(),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upsert discovered")
}

func TestIntersect_InvalidatorError_NotFatal(t *testing.T) {
	pmID := uuid.New()
	svc, wrepo, inv, pid, _ := newServiceForTest(t, defaultMappingsFixture(), pmID, nil)
	inv.err = errors.New("redis down")

	res, err := svc.IntersectCPEModel(context.Background(), IntersectInput{
		ProductID:       pid,
		SoftwareVersion: "v1.0.0",
		Entries:         cpeEntriesFixture(),
	})
	require.NoError(t, err, "失效失败不致命")
	assert.Equal(t, 3, res.Matched)
	assert.Equal(t, 1, wrepo.calls)
	assert.Equal(t, 1, inv.calls)
}

func TestIntersect_NilInvalidator_OK(t *testing.T) {
	pmID := uuid.New()
	repo := newFakeRepo()
	repo.defaultByModel[pmID] = defaultMappingsFixture()

	pmCopy := pmID
	pg, pid := newProductWith(&pmCopy, nil)

	svc := NewIntersectService(repo, &fakeIntersectRepo{}, pg, nil, nil, zap.NewNop())
	res, err := svc.IntersectCPEModel(context.Background(), IntersectInput{
		ProductID:       pid,
		SoftwareVersion: "v1.0.0",
		Entries:         cpeEntriesFixture(),
	})
	require.NoError(t, err)
	assert.Equal(t, 3, res.Matched)
}

func TestIntersect_DuplicatePrivatePathInCPE_LastWins(t *testing.T) {
	pmID := uuid.New()
	svc, wrepo, _, pid, _ := newServiceForTest(t, defaultMappingsFixture(), pmID, map[string]any{"access": true})

	entries := []CPEEntry{
		{PrivatePath: "Dev.WiFi.SSID.{i}.Enabled", Access: "readOnly"},
		{PrivatePath: "Dev.WiFi.SSID.{i}.Enabled", Access: "writeOnly"}, // 后者覆盖
	}
	_, err := svc.IntersectCPEModel(context.Background(), IntersectInput{
		ProductID:       pid,
		SoftwareVersion: "v1.0.0",
		Entries:         entries,
	})
	require.NoError(t, err)

	for _, m := range wrepo.lastMappings {
		if m.PrivatePath == "Dev.WiFi.SSID.{i}.Enabled" {
			assert.Equal(t, "writeOnly", m.Access)
		}
	}
}

func TestIntersect_ListDefaultError_Propagated(t *testing.T) {
	repo := newFakeRepo()
	repo.listDefaultErr = errors.New("query failed")

	pmID := uuid.New()
	pmCopy := pmID
	pg, pid := newProductWith(&pmCopy, nil)

	svc := NewIntersectService(repo, &fakeIntersectRepo{}, pg, &fakeInvalidator{}, nil, zap.NewNop())
	_, err := svc.IntersectCPEModel(context.Background(), IntersectInput{
		ProductID:       pid,
		SoftwareVersion: "v1.0.0",
		Entries:         cpeEntriesFixture(),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list default mappings")
}

func TestNormalizeOverride_Variants(t *testing.T) {
	cases := []struct {
		name string
		in   map[string]any
		want overrideFlags
		dt   bool
	}{
		{"nil", nil, overrideFlags{}, false},
		{"all_false", map[string]any{"access": false, "min_value": false}, overrideFlags{}, false},
		{"access_true", map[string]any{"access": true}, overrideFlags{access: true}, false},
		{"all_true_with_dt", map[string]any{"access": true, "data_type": true, "change_applies": true, "min_value": true, "max_value": true},
			overrideFlags{access: true, dataType: false /* 强制 */, changeApplies: true, minValue: true, maxValue: true}, true},
		{"non_bool_value_ignored", map[string]any{"access": "yes"}, overrideFlags{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, dt := normalizeOverride(tc.in)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.dt, dt)
		})
	}
}
