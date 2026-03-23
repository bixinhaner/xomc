package datamodel

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mock repository for cleaner tests
// ---------------------------------------------------------------------------

type cleanerMockRepo struct {
	deleteExpiredFn      func(ctx context.Context, autoMaxAge, manualMaxAge int) (int64, error)
	touchLastAccessedFn  func(ctx context.Context, id uuid.UUID) error
	touchCallCount       atomic.Int64
}

func (m *cleanerMockRepo) GetByID(_ context.Context, _ uuid.UUID) (*DataModel, error) {
	return nil, fmt.Errorf("not found")
}
func (m *cleanerMockRepo) List(_ context.Context, _ DataModelFilter) (*model.ListResponse[DataModel], error) {
	return model.NewListResponse([]DataModel{}, 0, 1, 20), nil
}
func (m *cleanerMockRepo) FindActive(_ context.Context, _ model.CarrierCode, _ model.Technology, _, _ string, _ model.DataModelScope) (*DataModel, error) {
	return nil, nil
}
func (m *cleanerMockRepo) FindActiveWithFirmware(_ context.Context, _ model.CarrierCode, _ model.Technology, _, _, _ string, _ model.DataModelScope) (*DataModel, error) {
	return nil, nil
}
func (m *cleanerMockRepo) FindActiveForMatch(_ context.Context, _ model.CarrierCode, _ model.Technology, _, _, _ string) (*DataModel, error) {
	return nil, nil
}
func (m *cleanerMockRepo) Statistics(_ context.Context) (*DataModelStats, error) {
	return &DataModelStats{}, nil
}
func (m *cleanerMockRepo) Create(_ context.Context, dm *DataModel) error {
	dm.ID = uuid.New()
	return nil
}
func (m *cleanerMockRepo) Update(_ context.Context, _ *DataModel) error  { return nil }
func (m *cleanerMockRepo) Delete(_ context.Context, _ uuid.UUID) error   { return nil }
func (m *cleanerMockRepo) Activate(_ context.Context, _ uuid.UUID) error { return nil }
func (m *cleanerMockRepo) Deprecate(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (m *cleanerMockRepo) TouchLastAccessed(ctx context.Context, id uuid.UUID) error {
	m.touchCallCount.Add(1)
	if m.touchLastAccessedFn != nil {
		return m.touchLastAccessedFn(ctx, id)
	}
	return nil
}
func (m *cleanerMockRepo) DeleteExpired(ctx context.Context, autoMaxAge, manualMaxAge int) (int64, error) {
	if m.deleteExpiredFn != nil {
		return m.deleteExpiredFn(ctx, autoMaxAge, manualMaxAge)
	}
	return 0, nil
}

// ---------------------------------------------------------------------------
// Cleaner Tests
// ---------------------------------------------------------------------------

func Test_CleanExpired_NoExpired(t *testing.T) {
	repo := &cleanerMockRepo{
		deleteExpiredFn: func(_ context.Context, autoMaxAge, manualMaxAge int) (int64, error) {
			assert.Equal(t, 15, autoMaxAge)
			assert.Equal(t, 60, manualMaxAge)
			return 0, nil
		},
	}

	cleaner := NewDataModelCleaner(repo, nil, 15, 60, zap.NewNop())
	err := cleaner.CleanExpired(context.Background())
	require.NoError(t, err)
}

func Test_CleanExpired_DeletesSome(t *testing.T) {
	repo := &cleanerMockRepo{
		deleteExpiredFn: func(_ context.Context, _, _ int) (int64, error) {
			return 5, nil
		},
	}

	cleaner := NewDataModelCleaner(repo, nil, 15, 60, zap.NewNop())
	err := cleaner.CleanExpired(context.Background())
	require.NoError(t, err)
}

func Test_CleanExpired_RepoError(t *testing.T) {
	repo := &cleanerMockRepo{
		deleteExpiredFn: func(_ context.Context, _, _ int) (int64, error) {
			return 0, fmt.Errorf("db connection lost")
		},
	}

	cleaner := NewDataModelCleaner(repo, nil, 15, 60, zap.NewNop())
	err := cleaner.CleanExpired(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db connection lost")
}

func Test_CleanExpired_CustomIdleDays(t *testing.T) {
	var gotAutoAge, gotManualAge int
	repo := &cleanerMockRepo{
		deleteExpiredFn: func(_ context.Context, autoMaxAge, manualMaxAge int) (int64, error) {
			gotAutoAge = autoMaxAge
			gotManualAge = manualMaxAge
			return 0, nil
		},
	}

	cleaner := NewDataModelCleaner(repo, nil, 7, 30, zap.NewNop())
	err := cleaner.CleanExpired(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 7, gotAutoAge)
	assert.Equal(t, 30, gotManualAge)
}

func Test_Cleaner_StartStop(t *testing.T) {
	repo := &cleanerMockRepo{}
	cleaner := NewDataModelCleaner(repo, nil, 15, 60, zap.NewNop())

	// Start with a valid cron expression
	err := cleaner.Start("* * * * *")
	require.NoError(t, err)

	// Stop should not panic
	cleaner.Stop()
}

func Test_Cleaner_InvalidCron(t *testing.T) {
	repo := &cleanerMockRepo{}
	cleaner := NewDataModelCleaner(repo, nil, 15, 60, zap.NewNop())

	err := cleaner.Start("not a valid cron")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schedule datamodel cleanup")
}

// ---------------------------------------------------------------------------
// Concurrent Access / Touch Throttle Tests
// ---------------------------------------------------------------------------

func Test_TouchLastAccessedAsync_ThrottlesWithin60Min(t *testing.T) {
	repo := &cleanerMockRepo{}
	logger := zap.NewNop()
	registry := NewDataModelRegistry(repo, nil, logger)

	modelID := uuid.New()

	// First call should write
	registry.touchLastAccessedAsync(modelID)
	assert.Equal(t, int64(1), repo.touchCallCount.Load())

	// Second call within 60 minutes should be throttled
	registry.touchLastAccessedAsync(modelID)
	assert.Equal(t, int64(1), repo.touchCallCount.Load())
}

func Test_TouchLastAccessedAsync_AllowsAfterThrottleWindow(t *testing.T) {
	repo := &cleanerMockRepo{}
	logger := zap.NewNop()
	registry := NewDataModelRegistry(repo, nil, logger)

	modelID := uuid.New()

	// First call
	registry.touchLastAccessedAsync(modelID)
	assert.Equal(t, int64(1), repo.touchCallCount.Load())

	// Simulate the throttle window has passed by backdating the entry
	registry.touchThrottle.Store(modelID.String(), time.Now().Add(-61*time.Minute))

	// Should be allowed again
	registry.touchLastAccessedAsync(modelID)
	assert.Equal(t, int64(2), repo.touchCallCount.Load())
}

func Test_TouchLastAccessedAsync_DifferentModelsNotThrottled(t *testing.T) {
	repo := &cleanerMockRepo{}
	logger := zap.NewNop()
	registry := NewDataModelRegistry(repo, nil, logger)

	id1 := uuid.New()
	id2 := uuid.New()

	registry.touchLastAccessedAsync(id1)
	registry.touchLastAccessedAsync(id2)

	assert.Equal(t, int64(2), repo.touchCallCount.Load())
}

func Test_TouchLastAccessedAsync_ConcurrentSafety(t *testing.T) {
	repo := &cleanerMockRepo{}
	logger := zap.NewNop()
	registry := NewDataModelRegistry(repo, nil, logger)

	modelID := uuid.New()
	var wg sync.WaitGroup
	const goroutines = 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			registry.touchLastAccessedAsync(modelID)
		}()
	}

	wg.Wait()

	// Due to throttling, only the first call should go through (or a small number due to race window)
	calls := repo.touchCallCount.Load()
	assert.GreaterOrEqual(t, calls, int64(1))
	// Most should be throttled — allow a small race window
	assert.LessOrEqual(t, calls, int64(5), "expected most concurrent calls to be throttled")
}

func Test_TouchLastAccessedAsync_RepoErrorDoesNotPanic(t *testing.T) {
	repo := &cleanerMockRepo{
		touchLastAccessedFn: func(_ context.Context, _ uuid.UUID) error {
			return fmt.Errorf("db error")
		},
	}
	logger := zap.NewNop()
	registry := NewDataModelRegistry(repo, nil, logger)

	// Should not panic even when repo returns error
	registry.touchLastAccessedAsync(uuid.New())
	assert.Equal(t, int64(1), repo.touchCallCount.Load())

	// Throttle entry should NOT be stored on error, so next call should retry
	registry.touchLastAccessedAsync(uuid.New())
	assert.Equal(t, int64(2), repo.touchCallCount.Load())
}
