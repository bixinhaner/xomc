package license

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// captureSink records alerts for assertions.
type captureSink struct {
	alerts []Alert
	clears []string
	err    error
}

func (c *captureSink) Send(_ context.Context, a Alert) error {
	c.alerts = append(c.alerts, a)
	return c.err
}

func (c *captureSink) Clear(_ context.Context, identifier string) error {
	c.clears = append(c.clears, identifier)
	return c.err
}

func newMonitorForTest(repo SystemLicenseRepository, dev DeviceCounter, sink AlertSink) (*Monitor, *captureSink, *EnforcementMetrics) {
	cs, ok := sink.(*captureSink)
	if !ok {
		cs = &captureSink{}
		sink = cs
	}
	metrics := NewEnforcementMetrics(prometheus.NewRegistry())
	m := NewMonitor(repo, dev, sink, metrics, zap.NewNop())
	return m, cs, metrics
}

func TestMonitor_CheckExpiringSoon_NoLicense_NoAlert(t *testing.T) {
	repo := &mockSystemLicenseRepo{current: nil}
	dev := &fakeDeviceCounter{}
	m, sink, _ := newMonitorForTest(repo, dev, nil)

	require.NoError(t, m.CheckExpiringSoon(context.Background()))
	assert.Empty(t, sink.alerts)
}

func TestMonitor_CheckExpiringSoon_NullExpiry_NoAlert(t *testing.T) {
	defer fixedClock(time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC))()
	lic := systemLicense("PERP", DevicesSupport{"eNB": 10}, 0) // expiry_date=nil
	lic.ExpiryDate = nil
	repo := &mockSystemLicenseRepo{current: lic}
	dev := &fakeDeviceCounter{}
	m, sink, _ := newMonitorForTest(repo, dev, nil)

	require.NoError(t, m.CheckExpiringSoon(context.Background()))
	assert.Empty(t, sink.alerts)
}

func TestMonitor_CheckExpiringSoon_FarFuture_NoAlert(t *testing.T) {
	defer fixedClock(time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC))()
	lic := systemLicense("FUT", DevicesSupport{"eNB": 10}, 365*24*time.Hour) // 1y away
	repo := &mockSystemLicenseRepo{current: lic}
	dev := &fakeDeviceCounter{}
	m, sink, _ := newMonitorForTest(repo, dev, nil)

	require.NoError(t, m.CheckExpiringSoon(context.Background()))
	assert.Empty(t, sink.alerts, "365d remaining is outside 30d window")
}

func TestMonitor_CheckExpiringSoon_WindowAlerts(t *testing.T) {
	defer fixedClock(time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC))()
	cases := []struct {
		name         string
		daysUntilExp int
		wantSuffix   string
		wantSeverity AlertSeverity
	}{
		{"25 days → 30d warning", 25, "30d", AlertSeverityWarning},
		{"5 days → 7d major", 5, "7d", AlertSeverityMajor},
		{"1 day → 1d critical", 1, "1d", AlertSeverityCritical},
		{"already expired (negative days) → 1d critical", -1, "1d", AlertSeverityCritical},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// 加 12 小时确保 floor((24*N - 0)/24) == N
			lic := systemLicense("L", DevicesSupport{"eNB": 10}, time.Duration(tc.daysUntilExp)*24*time.Hour+12*time.Hour)
			repo := &mockSystemLicenseRepo{current: lic}
			dev := &fakeDeviceCounter{}
			m, sink, _ := newMonitorForTest(repo, dev, nil)
			require.NoError(t, m.CheckExpiringSoon(context.Background()))
			require.Len(t, sink.alerts, 1)
			assert.Equal(t, "license_expiring_"+tc.wantSuffix, sink.alerts[0].Identifier)
			assert.Equal(t, tc.wantSeverity, sink.alerts[0].Severity)
		})
	}
}

func TestMonitor_CheckCapacity_NoLicense(t *testing.T) {
	repo := &mockSystemLicenseRepo{current: nil}
	dev := &fakeDeviceCounter{count: 100}
	m, sink, _ := newMonitorForTest(repo, dev, nil)

	require.NoError(t, m.CheckCapacity(context.Background()))
	assert.Empty(t, sink.alerts)
}

func TestMonitor_CheckCapacity_Thresholds(t *testing.T) {
	defer fixedClock(time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC))()
	cases := []struct {
		name        string
		used        int
		max         int // 总 capacity = devices_support[eNB]
		wantAlert   bool
		wantPercent int
		wantSev     AlertSeverity
	}{
		{"60% → no alert", 60, 100, false, 0, ""},
		{"80% → warning", 80, 100, true, 80, AlertSeverityWarning},
		{"90% → major", 90, 100, true, 90, AlertSeverityMajor},
		{"95% → critical", 95, 100, true, 95, AlertSeverityCritical},
		{"99% → critical", 99, 100, true, 95, AlertSeverityCritical},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lic := systemLicense("L", DevicesSupport{"eNB": tc.max}, 365*24*time.Hour)
			repo := &mockSystemLicenseRepo{current: lic}
			dev := &fakeDeviceCounter{count: tc.used}
			m, sink, _ := newMonitorForTest(repo, dev, nil)
			require.NoError(t, m.CheckCapacity(context.Background()))
			if !tc.wantAlert {
				assert.Empty(t, sink.alerts)
				return
			}
			require.Len(t, sink.alerts, 1)
			assert.Equal(t, tc.wantSev, sink.alerts[0].Severity)
			assert.Contains(t, sink.alerts[0].Identifier, "_capacity_")
		})
	}
}

func TestMonitor_CheckCapacity_Dedup6h(t *testing.T) {
	defer fixedClock(time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC))()
	lic := systemLicense("L", DevicesSupport{"eNB": 100}, 365*24*time.Hour)
	repo := &mockSystemLicenseRepo{current: lic}
	dev := &fakeDeviceCounter{count: 85}
	m, sink, _ := newMonitorForTest(repo, dev, nil)

	// 第一次 80% → 出告警
	require.NoError(t, m.CheckCapacity(context.Background()))
	assert.Len(t, sink.alerts, 1)

	// 立刻第二次 → 6h dedup 抑制
	require.NoError(t, m.CheckCapacity(context.Background()))
	assert.Len(t, sink.alerts, 1, "second call inside 6h should dedup")
}

func TestMonitor_CheckCapacity_CrossThresholdShortcircuitsDedup(t *testing.T) {
	defer fixedClock(time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC))()
	lic := systemLicense("L", DevicesSupport{"eNB": 100}, 365*24*time.Hour)
	repo := &mockSystemLicenseRepo{current: lic}
	dev := &fakeDeviceCounter{count: 85}
	m, sink, _ := newMonitorForTest(repo, dev, nil)

	require.NoError(t, m.CheckCapacity(context.Background()))
	require.Len(t, sink.alerts, 1)
	assert.Equal(t, AlertSeverityWarning, sink.alerts[0].Severity)

	// 跨档位（80 → 90） — 不应被 dedup
	dev.count = 92
	require.NoError(t, m.CheckCapacity(context.Background()))
	assert.Len(t, sink.alerts, 2)
	assert.Equal(t, AlertSeverityMajor, sink.alerts[1].Severity)
}

func TestMonitor_CheckExpiringSoon_ClearsOnRecovery(t *testing.T) {
	defer fixedClock(time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC))()
	// Tick 1: 25 天后过期 → 30d warning
	repo := &mockSystemLicenseRepo{current: systemLicense("L", DevicesSupport{"eNB": 10}, 25*24*time.Hour+12*time.Hour)}
	dev := &fakeDeviceCounter{}
	m, sink, _ := newMonitorForTest(repo, dev, nil)

	require.NoError(t, m.CheckExpiringSoon(context.Background()))
	require.Len(t, sink.alerts, 1)
	assert.Equal(t, "license_expiring_30d", sink.alerts[0].Identifier)
	assert.Empty(t, sink.clears)

	// Tick 2: 换成 365 天后过期（恢复）→ 清除所有过期告警
	repo.current = systemLicense("L", DevicesSupport{"eNB": 10}, 365*24*time.Hour)
	require.NoError(t, m.CheckExpiringSoon(context.Background()))
	assert.Len(t, sink.alerts, 1, "no new alert on recovery")
	assert.ElementsMatch(t,
		[]string{"license_expiring_1d", "license_expiring_7d", "license_expiring_30d"},
		sink.clears)

	// Tick 3: 仍然健康 → 不重复清除（expiryAlertActive 已 reset）
	require.NoError(t, m.CheckExpiringSoon(context.Background()))
	assert.Len(t, sink.clears, 3, "no duplicate clears when already healthy")
}

func TestMonitor_CheckCapacity_ClearsOnRecovery(t *testing.T) {
	defer fixedClock(time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC))()
	lic := systemLicense("L", DevicesSupport{"eNB": 100}, 365*24*time.Hour)
	repo := &mockSystemLicenseRepo{current: lic}
	dev := &fakeDeviceCounter{count: 85}
	m, sink, _ := newMonitorForTest(repo, dev, nil)

	// Tick 1: 85% → 80pct warning
	require.NoError(t, m.CheckCapacity(context.Background()))
	require.Len(t, sink.alerts, 1)
	assert.Empty(t, sink.clears)

	// Tick 2: 降到 60%（恢复）→ 清除所有容量告警
	dev.count = 60
	require.NoError(t, m.CheckCapacity(context.Background()))
	assert.Len(t, sink.alerts, 1, "no new alert on recovery")
	assert.ElementsMatch(t,
		[]string{"license_capacity_80pct", "license_capacity_90pct", "license_capacity_95pct"},
		sink.clears)

	// Tick 3: 仍然健康 → 不重复清除
	require.NoError(t, m.CheckCapacity(context.Background()))
	assert.Len(t, sink.clears, 3, "no duplicate clears when already healthy")
}

func TestMonitor_CheckCapacity_ClearsWhenLicenseRemoved(t *testing.T) {
	defer fixedClock(time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC))()
	lic := systemLicense("L", DevicesSupport{"eNB": 100}, 365*24*time.Hour)
	repo := &mockSystemLicenseRepo{current: lic}
	dev := &fakeDeviceCounter{count: 90}
	m, sink, _ := newMonitorForTest(repo, dev, nil)

	// Tick 1: 90% → alert
	require.NoError(t, m.CheckCapacity(context.Background()))
	require.Len(t, sink.alerts, 1)

	// Tick 2: license 删除 → 清除所有容量告警
	repo.current = nil
	require.NoError(t, m.CheckCapacity(context.Background()))
	assert.ElementsMatch(t,
		[]string{"license_capacity_80pct", "license_capacity_90pct", "license_capacity_95pct"},
		sink.clears)
}

func TestMonitor_CheckCapacity_RaisesAndClearsExhaustedAlert(t *testing.T) {
	defer fixedClock(time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC))()
	lic := systemLicense("L", DevicesSupport{"eNB": 2}, 365*24*time.Hour)
	repo := &mockSystemLicenseRepo{current: lic}
	dev := &fakeDeviceCounter{count: 2, countByType: map[string]int{"ENB": 2}}
	m, sink, _ := newMonitorForTest(repo, dev, nil)

	// Tick 1: eNB 满（2/2）→ raise exhausted（warning）
	require.NoError(t, m.CheckCapacity(context.Background()))
	found := false
	for _, a := range sink.alerts {
		if a.Identifier == CapacityExhaustedIdentifier {
			found = true
			assert.Equal(t, AlertSeverityWarning, a.Severity)
		}
	}
	require.True(t, found, "should raise capacity_exhausted alert when type full")

	// Tick 2: 恢复（删设备，ENB=1<2）→ clear exhausted
	dev.countByType = map[string]int{"ENB": 1}
	sink.clears = nil
	require.NoError(t, m.CheckCapacity(context.Background()))
	assert.Contains(t, sink.clears, CapacityExhaustedIdentifier)
}

func TestMonitor_CheckCapacity_ExhaustedAlertCountsGsmIntoEnbPool(t *testing.T) {
	// issue #318：GSM 与 eNB 共用容量——eNB 配额的用量按 ENB+GSM 合计评估
	// （ENB=1、GSM=1 对 eNB=2 已满），即使 ENB 单类型未满也应 raise。
	defer fixedClock(time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC))()
	lic := systemLicense("L", DevicesSupport{"eNB": 2}, 365*24*time.Hour)
	repo := &mockSystemLicenseRepo{current: lic}
	dev := &fakeDeviceCounter{count: 2, countByType: map[string]int{"ENB": 1, "GSM": 1}}
	m, sink, _ := newMonitorForTest(repo, dev, nil)

	require.NoError(t, m.CheckCapacity(context.Background()))
	found := false
	for _, a := range sink.alerts {
		if a.Identifier == CapacityExhaustedIdentifier {
			found = true
		}
	}
	require.True(t, found, "should raise exhausted alert when ENB+GSM combined usage fills the eNB pool")
}

func TestMonitor_CheckCumulativeUsage_ExceededAlerts(t *testing.T) {
	defer fixedClock(time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC))()
	lic := &SystemLicense{
		ID: uuid.New(), LicenseID: "CUM", LicenseType: SystemLicenseTypeCommercial, IsCurrent: true,
		FeatureList: FeatureList(`{"time_limit_hours":10}`),
	}
	repo := &mockSystemLicenseRepo{current: lic}
	dev := &fakeDeviceCounter{}
	m, sink, _ := newMonitorForTest(repo, dev, nil)
	m.SetUsageRepo(stubUsageRepo{lastVisited: time.Date(2026, 5, 18, 2, 0, 0, 0, time.UTC)}) // 10h ago

	require.NoError(t, m.CheckCumulativeUsage(context.Background()))
	require.Len(t, sink.alerts, 1)
	assert.Equal(t, "license_cumulative_exceeded", sink.alerts[0].Identifier)
	assert.Equal(t, AlertSeverityCritical, sink.alerts[0].Severity)
}

func TestMonitor_CheckCumulativeUsage_WithinLimitNoAlert(t *testing.T) {
	defer fixedClock(time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC))()
	lic := &SystemLicense{
		ID: uuid.New(), LicenseID: "CUM", LicenseType: SystemLicenseTypeCommercial, IsCurrent: true,
		FeatureList: FeatureList(`{"time_limit_hours":100}`),
	}
	repo := &mockSystemLicenseRepo{current: lic}
	m, sink, _ := newMonitorForTest(repo, &fakeDeviceCounter{}, nil)
	m.SetUsageRepo(stubUsageRepo{lastVisited: time.Date(2026, 5, 18, 11, 0, 0, 0, time.UTC)}) // 1h ago

	require.NoError(t, m.CheckCumulativeUsage(context.Background()))
	assert.Empty(t, sink.alerts)
}

func TestMonitor_CheckCumulativeUsage_NoTimeLimitClearsPreviousAlert(t *testing.T) {
	defer fixedClock(time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC))()
	// 先超限发告警
	lic := &SystemLicense{
		ID: uuid.New(), LicenseID: "CUM", LicenseType: SystemLicenseTypeCommercial, IsCurrent: true,
		FeatureList: FeatureList(`{"time_limit_hours":1}`),
	}
	repo := &mockSystemLicenseRepo{current: lic}
	m, sink, _ := newMonitorForTest(repo, &fakeDeviceCounter{}, nil)
	m.SetUsageRepo(stubUsageRepo{lastVisited: time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)}) // 2h > 1h limit

	require.NoError(t, m.CheckCumulativeUsage(context.Background()))
	require.Len(t, sink.alerts, 1)

	// 换成无 time_limit 的 license → 清除告警
	repo.current = &SystemLicense{
		ID: uuid.New(), LicenseID: "CUM2", LicenseType: SystemLicenseTypeCommercial, IsCurrent: true,
		FeatureList: FeatureList(`{}`),
	}
	require.NoError(t, m.CheckCumulativeUsage(context.Background()))
	assert.Contains(t, sink.clears, "license_cumulative_exceeded")
}
