package license

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// captureSink records alerts for assertions.
type captureSink struct {
	alerts []Alert
	err    error
}

func (c *captureSink) Send(_ context.Context, a Alert) error {
	c.alerts = append(c.alerts, a)
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
