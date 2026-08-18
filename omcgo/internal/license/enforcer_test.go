package license

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// fixedClock returns a stable nowFunc for deterministic tests + cleanup.
func fixedClock(t time.Time) func() {
	prev := nowFunc
	nowFunc = func() time.Time { return t }
	return func() { nowFunc = prev }
}

// fakeDeviceCounter — DeviceCounter 简化 mock。
type fakeDeviceCounter struct {
	count       int
	countByType map[string]int
	err         error
}

func (f *fakeDeviceCounter) CountDevices(_ context.Context) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.count, nil
}

func (f *fakeDeviceCounter) CountDevicesByType(_ context.Context) (map[string]int, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.countByType, nil
}

// systemLicense 构造一个 active system_license fixture。
func systemLicense(licID string, devicesSupport DevicesSupport, expiryIn time.Duration) *SystemLicense {
	lic := &SystemLicense{
		ID:             uuid.New(),
		LicenseID:      licID,
		LicenseType:    SystemLicenseTypeCommercial,
		IsCurrent:      true,
		DevicesSupport: devicesSupport,
		IssuedAt:       nowFunc(),
	}
	if expiryIn != 0 {
		t := nowFunc().Add(expiryIn)
		lic.ExpiryDate = &t
	}
	return lic
}

func TestEnforcer_EnforceCapacity(t *testing.T) {
	tests := []struct {
		name       string
		licCurrent *SystemLicense
		usedByType map[string]int
		deviceType string
		additional int
		wantErr    error
	}{
		{
			name:       "no license configured → fail-closed reject",
			licCurrent: nil,
			deviceType: "eNB",
			additional: 1,
			wantErr:    commonerrors.ErrLicenseUnavailable,
		},
		{
			name:       "empty deviceType → unknown NE type rejected",
			licCurrent: systemLicense("L1", DevicesSupport{"eNB": 100}, 0),
			deviceType: "",
			additional: 1,
			wantErr:    commonerrors.ErrLicenseCapacityExceeded,
		},
		{
			name:       "well under per-type capacity allowed",
			licCurrent: systemLicense("L1", DevicesSupport{"eNB": 100, "gNB": 100}, 0),
			usedByType: map[string]int{"ENB": 50},
			deviceType: "eNB",
			additional: 1,
			wantErr:    nil,
		},
		{
			name:       "exactly at per-type boundary allowed",
			licCurrent: systemLicense("L2", DevicesSupport{"eNB": 50, "gNB": 50}, 0),
			usedByType: map[string]int{"ENB": 49},
			deviceType: "eNB",
			additional: 1,
			wantErr:    nil,
		},
		{
			name:       "over per-type capacity rejected (single type exceeds)",
			licCurrent: systemLicense("L3", DevicesSupport{"eNB": 10, "gNB": 10}, 0),
			usedByType: map[string]int{"ENB": 10, "GNB": 0},
			deviceType: "eNB",
			additional: 1,
			wantErr:    commonerrors.ErrLicenseCapacityExceeded,
		},
		{
			// per-type 核心用例（issue #316）：eNB 已满但 gNB=0，gNB 仍可接入，互不影响。
			name:       "other type full does not block this type",
			licCurrent: systemLicense("L4", DevicesSupport{"eNB": 10, "gNB": 10}, 0),
			usedByType: map[string]int{"ENB": 10, "GNB": 0},
			deviceType: "gNB",
			additional: 1,
			wantErr:    nil,
		},
		{
			// license 未授权该网元类型（map 无该 key）→ 严格拒绝。
			name:       "device type not authorized by license rejected",
			licCurrent: systemLicense("L5", DevicesSupport{"eNB": 100}, 0),
			usedByType: map[string]int{"ENB": 0},
			deviceType: "gNB",
			additional: 1,
			wantErr:    commonerrors.ErrLicenseCapacityExceeded,
		},
		{
			// 大小写不敏感：设备 ne_type=ENB 匹配 license key=eNB。
			name:       "case-insensitive type matching allowed",
			licCurrent: systemLicense("L6", DevicesSupport{"eNB": 5}, 0),
			usedByType: map[string]int{"ENB": 2},
			deviceType: "ENB",
			additional: 1,
			wantErr:    nil,
		},
		{
			// issue #318：GSM 与 eNB 共用容量——license 只授权 eNB 也允许 GSM 接入。
			name:       "gsm authorized via shared eNB capacity",
			licCurrent: systemLicense("L7", DevicesSupport{"eNB": 10}, 0),
			usedByType: map[string]int{"ENB": 3, "GSM": 4},
			deviceType: "GSM",
			additional: 1,
			wantErr:    nil,
		},
		{
			// issue #318：用量按 eNB+GSM 合计控制（6+4+1 > 10 拒绝）。
			name:       "gsm rejected when shared eNB pool full",
			licCurrent: systemLicense("L8", DevicesSupport{"eNB": 10}, 0),
			usedByType: map[string]int{"ENB": 6, "GSM": 4},
			deviceType: "GSM",
			additional: 1,
			wantErr:    commonerrors.ErrLicenseCapacityExceeded,
		},
		{
			// issue #318：GSM 占用把 eNB 也挤满（5+5+1 > 10 拒绝）。
			name:       "enb rejected when shared pool full due to gsm usage",
			licCurrent: systemLicense("L9", DevicesSupport{"eNB": 10}, 0),
			usedByType: map[string]int{"ENB": 5, "GSM": 5},
			deviceType: "eNB",
			additional: 1,
			wantErr:    commonerrors.ErrLicenseCapacityExceeded,
		},
		{
			// issue #318：共用容量意味着没有独立的 GSM 配额——license 未授权 eNB
			// 时 GSM 同样拒绝。
			name:       "gsm rejected when eNB not authorized",
			licCurrent: systemLicense("L10", DevicesSupport{"gNB": 10}, 0),
			usedByType: map[string]int{"GNB": 0},
			deviceType: "GSM",
			additional: 1,
			wantErr:    commonerrors.ErrLicenseCapacityExceeded,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockSystemLicenseRepo{current: tc.licCurrent}
			dev := &fakeDeviceCounter{countByType: tc.usedByType}
			e := NewEnforcer(repo, dev, zap.NewNop(), nil)
			err := e.EnforceCapacity(context.Background(), tc.deviceType, tc.additional)
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.True(t, errors.Is(err, tc.wantErr), "expected wrap of %v, got %v", tc.wantErr, err)
		})
	}
}

func TestEnforcer_EnforceCapacity_NegativeAdditional(t *testing.T) {
	repo := &mockSystemLicenseRepo{}
	dev := &fakeDeviceCounter{}
	e := NewEnforcer(repo, dev, zap.NewNop(), nil)
	err := e.EnforceCapacity(context.Background(), "eNB", -1)
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
}

func TestEnforcer_RaisesExhaustedAlertOnCapacityDenied(t *testing.T) {
	lic := systemLicense("L", DevicesSupport{"eNB": 1}, 0)
	repo := &mockSystemLicenseRepo{current: lic}
	dev := &fakeDeviceCounter{countByType: map[string]int{"ENB": 1}}
	e := NewEnforcer(repo, dev, zap.NewNop(), nil)
	sink := &captureSink{}
	e.SetAlertSink(sink)

	err := e.EnforceCapacity(context.Background(), "eNB", 1)
	require.Error(t, err)
	require.Len(t, sink.alerts, 1)
	assert.Equal(t, CapacityExhaustedIdentifier, sink.alerts[0].Identifier)
	assert.Equal(t, AlertSeverityWarning, sink.alerts[0].Severity)

	// 1h 去重窗口：再次容量满拒绝不重复 Send。
	sink.alerts = nil
	_ = e.EnforceCapacity(context.Background(), "eNB", 1)
	assert.Empty(t, sink.alerts, "dedup window should suppress repeated exhausted alert")
}

func TestEnforcer_EnforceExpiry(t *testing.T) {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	defer fixedClock(now)()

	tests := []struct {
		name       string
		licCurrent *SystemLicense
		wantErr    error
	}{
		{
			name:       "no license → fail-closed reject",
			licCurrent: nil,
			wantErr:    commonerrors.ErrLicenseUnavailable,
		},
		{
			name:       "NULL expiry (perpetual) allowed",
			licCurrent: &SystemLicense{ID: uuid.New(), LicenseID: "PERP", LicenseType: SystemLicenseTypeCommercial},
			wantErr:    nil,
		},
		{
			name:       "future expiry allowed",
			licCurrent: systemLicense("FUT", DevicesSupport{}, 24*time.Hour),
			wantErr:    nil,
		},
		{
			name:       "past expiry rejected",
			licCurrent: systemLicense("OLD", DevicesSupport{}, -24*time.Hour),
			wantErr:    commonerrors.ErrLicenseExpired,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockSystemLicenseRepo{current: tc.licCurrent}
			dev := &fakeDeviceCounter{}
			e := NewEnforcer(repo, dev, zap.NewNop(), nil)
			err := e.EnforceExpiry(context.Background(), "device.create")
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.True(t, errors.Is(err, tc.wantErr))
		})
	}
}

func TestEnforcer_Quota(t *testing.T) {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	defer fixedClock(now)()

	t.Run("no license", func(t *testing.T) {
		repo := &mockSystemLicenseRepo{}
		dev := &fakeDeviceCounter{count: 0}
		e := NewEnforcer(repo, dev, zap.NewNop(), nil)
		q, err := e.Quota(context.Background())
		require.NoError(t, err)
		assert.False(t, q.HasActiveLicense)
	})

	t.Run("with license + per-type map", func(t *testing.T) {
		lic := systemLicense("L1", DevicesSupport{"eNB": 100, "gNB": 200}, 30*24*time.Hour)
		repo := &mockSystemLicenseRepo{current: lic}
		dev := &fakeDeviceCounter{count: 50, countByType: map[string]int{"ENB": 20, "GNB": 30}}
		e := NewEnforcer(repo, dev, zap.NewNop(), nil)
		q, err := e.Quota(context.Background())
		require.NoError(t, err)
		assert.True(t, q.HasActiveLicense)
		assert.Equal(t, 300, q.MaxDevices, "total = 100 + 200")
		assert.Equal(t, 50, q.UsedDevices)
		assert.InDelta(t, 50.0/300.0, q.UsageRatio, 1e-9)
		assert.Equal(t, 30, q.DaysRemaining)
		assert.Len(t, q.PerType, 2)
		assert.Equal(t, 100, q.PerType["eNB"].Max)
		assert.Equal(t, 20, q.PerType["eNB"].Used)
		assert.Equal(t, 200, q.PerType["gNB"].Max)
		assert.Equal(t, 30, q.PerType["gNB"].Used)
	})

	t.Run("perpetual (NULL expiry) days_remaining=-1", func(t *testing.T) {
		lic := &SystemLicense{
			ID:             uuid.New(),
			LicenseID:      "PERP",
			LicenseType:    SystemLicenseTypeCommercial,
			DevicesSupport: DevicesSupport{"eNB": 10},
		}
		repo := &mockSystemLicenseRepo{current: lic}
		dev := &fakeDeviceCounter{count: 5}
		e := NewEnforcer(repo, dev, zap.NewNop(), nil)
		q, err := e.Quota(context.Background())
		require.NoError(t, err)
		assert.Equal(t, -1, q.DaysRemaining)
	})
}

func TestEnforcer_Caching(t *testing.T) {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	defer fixedClock(now)()

	lic := systemLicense("L1", DevicesSupport{"eNB": 100}, 24*time.Hour)
	repo := &countingRepo{mockSystemLicenseRepo: mockSystemLicenseRepo{current: lic}}
	dev := &fakeDeviceCounter{count: 1}
	e := NewEnforcer(repo, dev, zap.NewNop(), nil)
	e.SetCacheTTL(1 * time.Hour)
	_ = lic

	// 第一次 → repo 调一次
	_, err := e.ActiveLicense(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, repo.getCurrentCalls, "first call should hit repo")

	// 立即第二次 → 命中缓存
	_, _ = e.ActiveLicense(context.Background())
	assert.Equal(t, 1, repo.getCurrentCalls, "second call should hit cache")

	// Invalidate → 强制刷新
	e.Invalidate()
	_, _ = e.ActiveLicense(context.Background())
	assert.Equal(t, 2, repo.getCurrentCalls, "after Invalidate, should hit repo")
}

func TestEnforcer_EmptyTable_CachedAsNil(t *testing.T) {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	defer fixedClock(now)()

	repo := &countingRepo{} // 空表
	dev := &fakeDeviceCounter{count: 0}
	e := NewEnforcer(repo, dev, zap.NewNop(), nil)
	e.SetCacheTTL(1 * time.Hour)

	lic1, err := e.ActiveLicense(context.Background())
	require.NoError(t, err)
	assert.Nil(t, lic1)

	// 第二次：缓存命中 nil，不重打 DB
	lic2, _ := e.ActiveLicense(context.Background())
	assert.Nil(t, lic2)
	assert.Equal(t, 1, repo.getCurrentCalls, "nil-cache should not re-query")
}

// countingRepo 复用 mockSystemLicenseRepo 但加 call counter，专测缓存路径。
type countingRepo struct {
	mockSystemLicenseRepo
	getCurrentCalls int
}

func (r *countingRepo) GetCurrent(ctx context.Context) (*SystemLicense, error) {
	r.getCurrentCalls++
	return r.mockSystemLicenseRepo.GetCurrent(ctx)
}

// stubUsageRepo — SystemLicenseUsageRepository 简化 mock。
type stubUsageRepo struct {
	lastVisited time.Time
	total       float64
	err         error
}

func (s stubUsageRepo) LastVisited(_ context.Context) (time.Time, error) {
	return s.lastVisited, s.err
}

func (s stubUsageRepo) Advance(_ context.Context, addHours float64) (float64, error) {
	if s.err != nil {
		return 0, s.err
	}
	return s.total + addHours, nil
}

func (s stubUsageRepo) CurrentUsage(_ context.Context) (float64, time.Time, error) {
	return s.total, s.lastVisited, s.err
}

func TestEnforceExpiry_CumulativeUsage(t *testing.T) {
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	defer fixedClock(now)()
	// perpetual（无 expiry_date）但 time_limit_hours=10；lastVisited 20h 前 → 累计超限
	repo := &mockSystemLicenseRepo{current: &SystemLicense{
		ID: uuid.New(), LicenseID: "CUM", LicenseType: SystemLicenseTypeCommercial, IsCurrent: true,
		FeatureList: FeatureList(`{"time_limit_hours":10}`),
	}}
	e := NewEnforcer(repo, &fakeDeviceCounter{}, zap.NewNop(), nil)
	e.SetUsageRepo(stubUsageRepo{lastVisited: now.Add(-20 * time.Hour)})
	err := e.EnforceExpiry(context.Background(), "device.create")
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrLicenseExpired))
}

func TestEnforceExpiry_TimeRollback(t *testing.T) {
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	defer fixedClock(now)()
	repo := &mockSystemLicenseRepo{current: &SystemLicense{
		ID: uuid.New(), LicenseID: "RB", LicenseType: SystemLicenseTypeCommercial, IsCurrent: true,
		FeatureList: FeatureList(`{"time_limit_hours":100}`),
	}}
	e := NewEnforcer(repo, &fakeDeviceCounter{}, zap.NewNop(), nil)
	e.SetUsageRepo(stubUsageRepo{lastVisited: now.Add(1 * time.Hour)}) // 上次检查时间在未来 → 回拨
	err := e.EnforceExpiry(context.Background(), "device.create")
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrLicenseExpired))
}

func TestEnforceExpiry_CumulativeWithinLimit(t *testing.T) {
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	defer fixedClock(now)()
	repo := &mockSystemLicenseRepo{current: &SystemLicense{
		ID: uuid.New(), LicenseID: "OK", LicenseType: SystemLicenseTypeCommercial, IsCurrent: true,
		FeatureList: FeatureList(`{"time_limit_hours":100}`),
	}}
	e := NewEnforcer(repo, &fakeDeviceCounter{}, zap.NewNop(), nil)
	e.SetUsageRepo(stubUsageRepo{lastVisited: now.Add(-1 * time.Hour)}) // elapsed 1h < 100h
	err := e.EnforceExpiry(context.Background(), "device.create")
	require.NoError(t, err)
}
