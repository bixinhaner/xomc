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

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// fixedClock returns a stable nowFunc for deterministic tests.
func fixedClock(t time.Time) func() {
	prev := nowFunc
	nowFunc = func() time.Time { return t }
	return func() { nowFunc = prev }
}

func newSubscription(maxDevices int, expiresIn time.Duration, grace int) *License {
	exp := nowFunc().Add(expiresIn)
	return &License{
		ID:              uuid.New(),
		LicenseType:     TypeSubscription,
		Status:          StatusActive,
		MaxDevices:      maxDevices,
		LicenseCode:     "TEST-SUB",
		ExpiryDate:      &exp,
		GracePeriodDays: grace,
	}
}

func newPerpetual(maxDevices int) *License {
	return &License{
		ID:          uuid.New(),
		LicenseType: TypePerpetual,
		Status:      StatusActive,
		MaxDevices:  maxDevices,
		LicenseCode: "TEST-PERP",
	}
}

func TestEnforcer_EnforceCapacity_Allowed(t *testing.T) {
	repo := &mockLicenseRepo{
		getActiveFn: func(ctx context.Context) (*License, error) {
			return newPerpetual(100), nil
		},
		countDevFn: func(ctx context.Context) (int, error) { return 50, nil },
	}
	e := NewEnforcer(repo, zap.NewNop(), nil)

	require.NoError(t, e.EnforceCapacity(context.Background(), 1))
}

func TestEnforcer_EnforceCapacity_AtBoundary(t *testing.T) {
	// used 99 + additional 1 == max 100 → allowed
	repo := &mockLicenseRepo{
		getActiveFn: func(ctx context.Context) (*License, error) {
			return newPerpetual(100), nil
		},
		countDevFn: func(ctx context.Context) (int, error) { return 99, nil },
	}
	e := NewEnforcer(repo, zap.NewNop(), nil)

	require.NoError(t, e.EnforceCapacity(context.Background(), 1))
}

func TestEnforcer_EnforceCapacity_Exceeded(t *testing.T) {
	repo := &mockLicenseRepo{
		getActiveFn: func(ctx context.Context) (*License, error) {
			return newPerpetual(100), nil
		},
		countDevFn: func(ctx context.Context) (int, error) { return 100, nil },
	}
	e := NewEnforcer(repo, zap.NewNop(), nil)

	err := e.EnforceCapacity(context.Background(), 1)
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrLicenseCapacityExceeded))
}

func TestEnforcer_EnforceCapacity_NoActiveLicense_DefaultAllow(t *testing.T) {
	repo := &mockLicenseRepo{
		getActiveFn: func(ctx context.Context) (*License, error) { return nil, nil },
	}
	e := NewEnforcer(repo, zap.NewNop(), nil)

	require.NoError(t, e.EnforceCapacity(context.Background(), 99999))
}

func TestEnforcer_EnforceCapacity_NegativeAdditional_RejectsAsInvalidInput(t *testing.T) {
	repo := &mockLicenseRepo{}
	e := NewEnforcer(repo, zap.NewNop(), nil)

	err := e.EnforceCapacity(context.Background(), -1)
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
}

func TestEnforcer_EnforceExpiry_Perpetual_AlwaysAllowed(t *testing.T) {
	repo := &mockLicenseRepo{
		getActiveFn: func(ctx context.Context) (*License, error) { return newPerpetual(100), nil },
	}
	e := NewEnforcer(repo, zap.NewNop(), nil)

	require.NoError(t, e.EnforceExpiry(context.Background(), "device.create"))
}

func TestEnforcer_EnforceExpiry_FutureSubscription_Allowed(t *testing.T) {
	defer fixedClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))()
	lic := newSubscription(100, 30*24*time.Hour, 0)
	repo := &mockLicenseRepo{
		getActiveFn: func(ctx context.Context) (*License, error) { return lic, nil },
	}
	e := NewEnforcer(repo, zap.NewNop(), nil)

	require.NoError(t, e.EnforceExpiry(context.Background(), "device.create"))
}

func TestEnforcer_EnforceExpiry_PastExpiry_Denied(t *testing.T) {
	defer fixedClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))()
	lic := newSubscription(100, -24*time.Hour, 0) // expired yesterday
	repo := &mockLicenseRepo{
		getActiveFn: func(ctx context.Context) (*License, error) { return lic, nil },
	}
	e := NewEnforcer(repo, zap.NewNop(), nil)

	err := e.EnforceExpiry(context.Background(), "device.create")
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrLicenseExpired))
}

func TestEnforcer_EnforceExpiry_WithinGracePeriod_Allowed(t *testing.T) {
	defer fixedClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))()
	// expired 5 days ago, grace 7 days → still allowed
	lic := newSubscription(100, -5*24*time.Hour, 7)
	repo := &mockLicenseRepo{
		getActiveFn: func(ctx context.Context) (*License, error) { return lic, nil },
	}
	e := NewEnforcer(repo, zap.NewNop(), nil)

	require.NoError(t, e.EnforceExpiry(context.Background(), "device.create"))
}

func TestEnforcer_EnforceExpiry_StatusExpired_DeniedEvenIfDateFuture(t *testing.T) {
	defer fixedClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))()
	lic := newSubscription(100, 30*24*time.Hour, 0)
	lic.Status = StatusExpired // explicitly expired by cron
	repo := &mockLicenseRepo{
		getActiveFn: func(ctx context.Context) (*License, error) { return lic, nil },
	}
	e := NewEnforcer(repo, zap.NewNop(), nil)

	err := e.EnforceExpiry(context.Background(), "device.create")
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrLicenseExpired))
}

func TestEnforcer_Cache_TTL(t *testing.T) {
	calls := 0
	repo := &mockLicenseRepo{
		getActiveFn: func(ctx context.Context) (*License, error) {
			calls++
			return newPerpetual(10), nil
		},
	}
	e := NewEnforcer(repo, zap.NewNop(), nil)
	e.SetCacheTTL(50 * time.Millisecond)

	_, err := e.ActiveLicense(context.Background())
	require.NoError(t, err)
	_, err = e.ActiveLicense(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, calls, "second call should hit cache")

	// Move clock forward beyond TTL.
	defer fixedClock(time.Now().Add(100 * time.Millisecond))()
	_, err = e.ActiveLicense(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, calls, "expired cache should refresh")
}

func TestEnforcer_Invalidate_ForcesReload(t *testing.T) {
	calls := 0
	repo := &mockLicenseRepo{
		getActiveFn: func(ctx context.Context) (*License, error) {
			calls++
			return newPerpetual(10), nil
		},
	}
	e := NewEnforcer(repo, zap.NewNop(), nil)
	_, err := e.ActiveLicense(context.Background())
	require.NoError(t, err)

	e.Invalidate()
	_, err = e.ActiveLicense(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 2, calls)
}

func TestEnforcer_Quota_NoActiveLicense(t *testing.T) {
	repo := &mockLicenseRepo{
		getActiveFn: func(ctx context.Context) (*License, error) { return nil, nil },
	}
	e := NewEnforcer(repo, zap.NewNop(), nil)

	q, err := e.Quota(context.Background())
	require.NoError(t, err)
	assert.False(t, q.HasActiveLicense)
}

func TestEnforcer_Quota_PerpetualSetsDaysRemainingNegativeOne(t *testing.T) {
	repo := &mockLicenseRepo{
		getActiveFn: func(ctx context.Context) (*License, error) { return newPerpetual(100), nil },
		countDevFn:  func(ctx context.Context) (int, error) { return 25, nil },
	}
	e := NewEnforcer(repo, zap.NewNop(), nil)

	q, err := e.Quota(context.Background())
	require.NoError(t, err)
	assert.True(t, q.HasActiveLicense)
	assert.Equal(t, 100, q.MaxDevices)
	assert.Equal(t, 25, q.UsedDevices)
	assert.InDelta(t, 0.25, q.UsageRatio, 0.001)
	assert.Equal(t, -1, q.DaysRemaining)
	assert.Equal(t, "perpetual", q.LicenseType)
}

func TestEnforcer_Quota_SubscriptionDaysRemaining(t *testing.T) {
	defer fixedClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))()
	lic := newSubscription(50, 30*24*time.Hour, 0)
	repo := &mockLicenseRepo{
		getActiveFn: func(ctx context.Context) (*License, error) { return lic, nil },
		countDevFn:  func(ctx context.Context) (int, error) { return 10, nil },
	}
	e := NewEnforcer(repo, zap.NewNop(), nil)

	q, err := e.Quota(context.Background())
	require.NoError(t, err)
	// 30 days - 1 hour rounding = 29 or 30 depending on integer division.
	assert.GreaterOrEqual(t, q.DaysRemaining, 29)
	assert.LessOrEqual(t, q.DaysRemaining, 30)
}

func TestEnforcer_RepoErrorPropagates(t *testing.T) {
	repoErr := errors.New("db unreachable")
	repo := &mockLicenseRepo{
		getActiveFn: func(ctx context.Context) (*License, error) { return nil, repoErr },
	}
	e := NewEnforcer(repo, zap.NewNop(), nil)

	err := e.EnforceCapacity(context.Background(), 1)
	require.Error(t, err)
	assert.True(t, errors.Is(err, repoErr))
}

func TestEnforcementError_Classification(t *testing.T) {
	// capacity error
	kind, ok := EnforcementError(commonerrors.ErrLicenseCapacityExceeded)
	assert.True(t, ok)
	assert.Equal(t, "capacity_exceeded", kind)

	// expiry error
	kind, ok = EnforcementError(commonerrors.ErrLicenseExpired)
	assert.True(t, ok)
	assert.Equal(t, "expired", kind)

	// non-enforcement error
	_, ok = EnforcementError(errors.New("other"))
	assert.False(t, ok)
}

func TestEnforcer_ExpiryWithoutDate_Allowed(t *testing.T) {
	// Subscription license with nil expiry date — treat as not-expired.
	lic := &License{
		ID:          uuid.New(),
		LicenseType: TypeSubscription,
		Status:      StatusActive,
		MaxDevices:  100,
		ExpiryDate:  nil,
	}
	repo := &mockLicenseRepo{
		getActiveFn: func(ctx context.Context) (*License, error) { return lic, nil },
	}
	e := NewEnforcer(repo, zap.NewNop(), nil)

	require.NoError(t, e.EnforceExpiry(context.Background(), "device.create"))
}

// Sanity-check parseThresholds (used by Monitor) — included here because
// monitor_test.go is the natural home but this guards a small contract.
func TestParseThresholds_HappyAndMalformed(t *testing.T) {
	out := parseThresholds(json.RawMessage(`[80, 90, 95]`))
	assert.Equal(t, []int{80, 90, 95}, out)

	out = parseThresholds(json.RawMessage(`malformed`))
	assert.Nil(t, out)

	out = parseThresholds(nil)
	assert.Nil(t, out)
}
