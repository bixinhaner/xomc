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

func newSubscriptionWithThresholds(maxDevices int, expiresIn time.Duration, thresholds string) *License {
	lic := newSubscription(maxDevices, expiresIn, 0)
	lic.CapacityAlertThresholds = json.RawMessage(thresholds)
	return lic
}

func TestMonitor_CheckExpiry_FlipsPastExpiry(t *testing.T) {
	defer fixedClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))()
	lic := newSubscription(100, -10*24*time.Hour, 0) // expired 10 days ago
	repo := &mockLicenseRepo{
		listActiveFn: func(ctx context.Context) ([]*License, error) { return []*License{lic}, nil },
		markExpFn: func(ctx context.Context, id uuid.UUID) error {
			assert.Equal(t, lic.ID, id)
			return nil
		},
	}
	m := NewMonitor(repo, &captureSink{}, nil, zap.NewNop())

	require.NoError(t, m.CheckExpiry(context.Background()))
}

func TestMonitor_CheckExpiry_RespectsGrace(t *testing.T) {
	defer fixedClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))()
	// expired 5 days ago, grace 7 days → still active, should NOT flip.
	lic := newSubscription(100, -5*24*time.Hour, 7)
	called := false
	repo := &mockLicenseRepo{
		listActiveFn: func(ctx context.Context) ([]*License, error) { return []*License{lic}, nil },
		markExpFn:    func(ctx context.Context, id uuid.UUID) error { called = true; return nil },
	}
	m := NewMonitor(repo, &captureSink{}, nil, zap.NewNop())

	require.NoError(t, m.CheckExpiry(context.Background()))
	assert.False(t, called, "license within grace must not be marked expired")
}

func TestMonitor_CheckExpiry_SkipsPerpetual(t *testing.T) {
	lic := newPerpetual(100)
	called := false
	repo := &mockLicenseRepo{
		listActiveFn: func(ctx context.Context) ([]*License, error) { return []*License{lic}, nil },
		markExpFn:    func(ctx context.Context, id uuid.UUID) error { called = true; return nil },
	}
	m := NewMonitor(repo, &captureSink{}, nil, zap.NewNop())

	require.NoError(t, m.CheckExpiry(context.Background()))
	assert.False(t, called)
}

func TestMonitor_CheckExpiringSoon_30DayWarning(t *testing.T) {
	defer fixedClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))()
	lic := newSubscription(100, 25*24*time.Hour, 0) // 25 days remaining → fires 30d window
	sink := &captureSink{}
	metrics := NewEnforcementMetrics(nil)
	repo := &mockLicenseRepo{
		listActiveFn: func(ctx context.Context) ([]*License, error) { return []*License{lic}, nil },
	}
	m := NewMonitor(repo, sink, metrics, zap.NewNop())

	require.NoError(t, m.CheckExpiringSoon(context.Background()))
	require.Len(t, sink.alerts, 1)
	assert.Equal(t, AlertSeverityWarning, sink.alerts[0].Severity)
	assert.Contains(t, sink.alerts[0].Identifier, "30d")
}

func TestMonitor_CheckExpiringSoon_7DayMajor(t *testing.T) {
	defer fixedClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))()
	lic := newSubscription(100, 5*24*time.Hour, 0) // 5 days remaining → fires 7d window
	sink := &captureSink{}
	metrics := NewEnforcementMetrics(nil)
	repo := &mockLicenseRepo{
		listActiveFn: func(ctx context.Context) ([]*License, error) { return []*License{lic}, nil },
	}
	m := NewMonitor(repo, sink, metrics, zap.NewNop())

	require.NoError(t, m.CheckExpiringSoon(context.Background()))
	require.Len(t, sink.alerts, 1)
	assert.Equal(t, AlertSeverityMajor, sink.alerts[0].Severity)
	assert.Contains(t, sink.alerts[0].Identifier, "7d")
}

func TestMonitor_CheckExpiringSoon_1DayCritical(t *testing.T) {
	defer fixedClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))()
	lic := newSubscription(100, 12*time.Hour, 0) // <1 day → 1d window
	sink := &captureSink{}
	metrics := NewEnforcementMetrics(nil)
	repo := &mockLicenseRepo{
		listActiveFn: func(ctx context.Context) ([]*License, error) { return []*License{lic}, nil },
	}
	m := NewMonitor(repo, sink, metrics, zap.NewNop())

	require.NoError(t, m.CheckExpiringSoon(context.Background()))
	require.Len(t, sink.alerts, 1)
	assert.Equal(t, AlertSeverityCritical, sink.alerts[0].Severity)
	assert.Contains(t, sink.alerts[0].Identifier, "1d")
}

func TestMonitor_CheckExpiringSoon_NoAlertOutsideWindows(t *testing.T) {
	defer fixedClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))()
	lic := newSubscription(100, 100*24*time.Hour, 0) // 100 days, way out
	sink := &captureSink{}
	metrics := NewEnforcementMetrics(nil)
	repo := &mockLicenseRepo{
		listActiveFn: func(ctx context.Context) ([]*License, error) { return []*License{lic}, nil },
	}
	m := NewMonitor(repo, sink, metrics, zap.NewNop())

	require.NoError(t, m.CheckExpiringSoon(context.Background()))
	assert.Empty(t, sink.alerts)
}

func TestMonitor_CheckCapacity_Below80PctNoAlert(t *testing.T) {
	defer fixedClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))()
	lic := newSubscriptionWithThresholds(100, 30*24*time.Hour, `[80, 90, 95]`)
	sink := &captureSink{}
	metrics := NewEnforcementMetrics(nil)
	repo := &mockLicenseRepo{
		getActiveFn: func(ctx context.Context) (*License, error) { return lic, nil },
		countDevFn:  func(ctx context.Context) (int, error) { return 70, nil },
	}
	m := NewMonitor(repo, sink, metrics, zap.NewNop())

	require.NoError(t, m.CheckCapacity(context.Background()))
	assert.Empty(t, sink.alerts)
}

func TestMonitor_CheckCapacity_80PctWarning(t *testing.T) {
	defer fixedClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))()
	lic := newSubscriptionWithThresholds(100, 30*24*time.Hour, `[80, 90, 95]`)
	sink := &captureSink{}
	metrics := NewEnforcementMetrics(nil)
	repo := &mockLicenseRepo{
		getActiveFn:   func(ctx context.Context) (*License, error) { return lic, nil },
		countDevFn:    func(ctx context.Context) (int, error) { return 80, nil },
		updCapAlertFn: func(ctx context.Context, id uuid.UUID, threshold int, at time.Time) error { return nil },
	}
	m := NewMonitor(repo, sink, metrics, zap.NewNop())

	require.NoError(t, m.CheckCapacity(context.Background()))
	require.Len(t, sink.alerts, 1)
	assert.Equal(t, AlertSeverityWarning, sink.alerts[0].Severity)
	assert.Contains(t, sink.alerts[0].Identifier, "80")
}

func TestMonitor_CheckCapacity_95PctCritical(t *testing.T) {
	defer fixedClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))()
	lic := newSubscriptionWithThresholds(100, 30*24*time.Hour, `[80, 90, 95]`)
	sink := &captureSink{}
	metrics := NewEnforcementMetrics(nil)
	repo := &mockLicenseRepo{
		getActiveFn:   func(ctx context.Context) (*License, error) { return lic, nil },
		countDevFn:    func(ctx context.Context) (int, error) { return 96, nil },
		updCapAlertFn: func(ctx context.Context, id uuid.UUID, threshold int, at time.Time) error { return nil },
	}
	m := NewMonitor(repo, sink, metrics, zap.NewNop())

	require.NoError(t, m.CheckCapacity(context.Background()))
	require.Len(t, sink.alerts, 1)
	assert.Equal(t, AlertSeverityCritical, sink.alerts[0].Severity)
	assert.Contains(t, sink.alerts[0].Identifier, "95")
}

func TestMonitor_CheckCapacity_DedupWithinWindow(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	defer fixedClock(now)()
	lic := newSubscriptionWithThresholds(100, 30*24*time.Hour, `[80, 90, 95]`)
	// Last alert 2h ago for same threshold 80 → must dedup.
	twoHoursAgo := now.Add(-2 * time.Hour)
	lastT := 80
	lic.LastCapacityAlertAt = &twoHoursAgo
	lic.LastCapacityAlertThreshold = &lastT

	sink := &captureSink{}
	metrics := NewEnforcementMetrics(nil)
	repo := &mockLicenseRepo{
		getActiveFn: func(ctx context.Context) (*License, error) { return lic, nil },
		countDevFn:  func(ctx context.Context) (int, error) { return 82, nil },
	}
	m := NewMonitor(repo, sink, metrics, zap.NewNop())

	require.NoError(t, m.CheckCapacity(context.Background()))
	assert.Empty(t, sink.alerts, "same threshold within 6h dedup window must skip")
}

func TestMonitor_CheckCapacity_HigherThresholdBypassesDedup(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	defer fixedClock(now)()
	lic := newSubscriptionWithThresholds(100, 30*24*time.Hour, `[80, 90, 95]`)
	// Last alert 2h ago for threshold 80; usage now 92 → should fire 90.
	twoHoursAgo := now.Add(-2 * time.Hour)
	lastT := 80
	lic.LastCapacityAlertAt = &twoHoursAgo
	lic.LastCapacityAlertThreshold = &lastT

	sink := &captureSink{}
	metrics := NewEnforcementMetrics(nil)
	repo := &mockLicenseRepo{
		getActiveFn:   func(ctx context.Context) (*License, error) { return lic, nil },
		countDevFn:    func(ctx context.Context) (int, error) { return 92, nil },
		updCapAlertFn: func(ctx context.Context, id uuid.UUID, threshold int, at time.Time) error { return nil },
	}
	m := NewMonitor(repo, sink, metrics, zap.NewNop())

	require.NoError(t, m.CheckCapacity(context.Background()))
	require.Len(t, sink.alerts, 1)
	assert.Contains(t, sink.alerts[0].Identifier, "90")
}

func TestMonitor_CheckCapacity_NoActiveLicense_RecordsZero(t *testing.T) {
	repo := &mockLicenseRepo{
		getActiveFn: func(ctx context.Context) (*License, error) { return nil, nil },
	}
	metrics := NewEnforcementMetrics(nil)
	m := NewMonitor(repo, &captureSink{}, metrics, zap.NewNop())

	require.NoError(t, m.CheckCapacity(context.Background()))
}

func TestMonitor_CheckCapacity_RepoErrorPropagates(t *testing.T) {
	repo := &mockLicenseRepo{
		getActiveFn: func(ctx context.Context) (*License, error) {
			return nil, errors.New("db down")
		},
	}
	m := NewMonitor(repo, &captureSink{}, nil, zap.NewNop())

	err := m.CheckCapacity(context.Background())
	require.Error(t, err)
}

func TestNoopAlertSink_DoesNothing(t *testing.T) {
	require.NoError(t, NoopAlertSink{}.Send(context.Background(), Alert{}))
}

func TestCapacitySeverity_Helper(t *testing.T) {
	thresholds := []int{80, 90, 95}
	assert.Equal(t, AlertSeverityWarning, capacitySeverity(80, thresholds))
	assert.Equal(t, AlertSeverityMajor, capacitySeverity(90, thresholds))
	assert.Equal(t, AlertSeverityCritical, capacitySeverity(95, thresholds))
}
