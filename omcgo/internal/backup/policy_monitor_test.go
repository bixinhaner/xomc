package backup

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// monTaskRepo is a focused mock satisfying TaskRepository for cleanup tests.
type monTaskRepo struct {
	cleanupFn func(ctx context.Context, cutoff time.Time, keepLastN int) (int64, error)
	cleanupCalls int
	gotCutoff    time.Time
	gotKeep      int
}

func (m *monTaskRepo) Create(_ context.Context, _ *BackupTask) error              { return nil }
func (m *monTaskRepo) GetByID(_ context.Context, _ uuid.UUID) (*BackupTask, error) { return nil, nil }
func (m *monTaskRepo) Update(_ context.Context, _ *BackupTask) error              { return nil }
func (m *monTaskRepo) Delete(_ context.Context, _ uuid.UUID) error                { return nil }
func (m *monTaskRepo) List(_ context.Context, _ TaskFilter) (*model.ListResponse[BackupTask], error) {
	return nil, nil
}
func (m *monTaskRepo) CleanupOldRows(ctx context.Context, cutoff time.Time, keepLastN int) (int64, error) {
	m.cleanupCalls++
	m.gotCutoff = cutoff
	m.gotKeep = keepLastN
	if m.cleanupFn != nil {
		return m.cleanupFn(ctx, cutoff, keepLastN)
	}
	return 0, nil
}
func (m *monTaskRepo) UpdateFilePath(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (m *monTaskRepo) FindByIDPrefix(_ context.Context, _ string, _ int) ([]*BackupTask, error) {
	return nil, nil
}

// monPolicyRepo lets tests dictate what BackupPolicy the monitor sees.
type monPolicyRepo struct {
	current *BackupPolicy
	getErr  error
}

func (m *monPolicyRepo) Get(_ context.Context) (*BackupPolicy, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.current == nil {
		// Return canonical sentinel so PolicyService.Get falls through to DefaultPolicy().
		return nil, commonerrors.ErrNotFound
	}
	cp := *m.current
	return &cp, nil
}

func (m *monPolicyRepo) Upsert(_ context.Context, _ *BackupPolicy) error { return nil }

// ---------------------------------------------------------------------------
// V3-V7 cleanup behaviour
// ---------------------------------------------------------------------------

func TestPolicyMonitor_RunCleanupOnce_DeletesOldRows(t *testing.T) {
	// V4: retentionDays=30; cleanup deletes 3 rows
	taskRepo := &monTaskRepo{
		cleanupFn: func(_ context.Context, _ time.Time, _ int) (int64, error) {
			return 3, nil
		},
	}
	policySvc := NewPolicyService(&monPolicyRepo{current: &BackupPolicy{
		AutoCleanup:           true,
		RetentionDays:         30,
		KeepLastN:             5,
		CleanupTime:           "03:00",
		CompressionLevel:      6,
		CompressionFormat:     "gzip",
		StorageBackend:        "local",
		LocalPath:             "/var/backup/omc",
		MaxStorageGB:          500,
		EncryptionAlgorithm:   "AES-256-GCM",
		MaxBackupCount:        100,
		MinBackupCount:        3,
		AlertThresholdPercent: 80,
	}}, zap.NewNop())
	mon := NewPolicyMonitor(policySvc, taskRepo, NewPolicyMetrics(nil), zap.NewNop())

	deleted, err := mon.RunCleanupOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(3), deleted)
	assert.Equal(t, 1, taskRepo.cleanupCalls)
	assert.Equal(t, 5, taskRepo.gotKeep)
	// cutoff should be ~30 days ago (allow ±5s for test runtime)
	expected := time.Now().Add(-30 * 24 * time.Hour)
	assert.WithinDuration(t, expected, taskRepo.gotCutoff, 5*time.Second)
}

func TestPolicyMonitor_RunCleanupOnce_HonoursKeepLastN(t *testing.T) {
	// V5: keep_last_n forwarded verbatim
	taskRepo := &monTaskRepo{
		cleanupFn: func(_ context.Context, _ time.Time, keepLastN int) (int64, error) {
			assert.Equal(t, 3, keepLastN)
			return 4, nil
		},
	}
	policySvc := NewPolicyService(&monPolicyRepo{current: &BackupPolicy{
		AutoCleanup:           true,
		RetentionDays:         10,
		KeepLastN:             3,
		CleanupTime:           "03:00",
		CompressionLevel:      6,
		CompressionFormat:     "gzip",
		StorageBackend:        "local",
		LocalPath:             "/var/backup/omc",
		MaxStorageGB:          500,
		EncryptionAlgorithm:   "AES-256-GCM",
		MaxBackupCount:        100,
		MinBackupCount:        3,
		AlertThresholdPercent: 80,
	}}, zap.NewNop())
	mon := NewPolicyMonitor(policySvc, taskRepo, NewPolicyMetrics(nil), zap.NewNop())

	deleted, err := mon.RunCleanupOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(4), deleted)
}

func TestPolicyMonitor_RunCleanupOnce_AutoCleanupFalseSkips(t *testing.T) {
	// V6: AutoCleanup=false → 0 deletes, repo never called
	taskRepo := &monTaskRepo{
		cleanupFn: func(_ context.Context, _ time.Time, _ int) (int64, error) {
			t.Fatal("CleanupOldRows must not be called when auto_cleanup=false")
			return 0, nil
		},
	}
	policySvc := NewPolicyService(&monPolicyRepo{current: &BackupPolicy{
		AutoCleanup:           false,
		RetentionDays:         30,
		KeepLastN:             5,
		CleanupTime:           "03:00",
		CompressionLevel:      6,
		CompressionFormat:     "gzip",
		StorageBackend:        "local",
		LocalPath:             "/var/backup/omc",
		MaxStorageGB:          500,
		EncryptionAlgorithm:   "AES-256-GCM",
		MaxBackupCount:        100,
		MinBackupCount:        3,
		AlertThresholdPercent: 80,
	}}, zap.NewNop())
	mon := NewPolicyMonitor(policySvc, taskRepo, NewPolicyMetrics(nil), zap.NewNop())

	deleted, err := mon.RunCleanupOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted)
	assert.Equal(t, 0, taskRepo.cleanupCalls)
}

func TestPolicyMonitor_RunCleanupOnce_EmptyPolicyUsesDefaults(t *testing.T) {
	// V7: when backup_policies is empty, PolicyService.Get returns DefaultPolicy()
	// which has AutoCleanup=true, RetentionDays=30, KeepLastN=5.
	taskRepo := &monTaskRepo{
		cleanupFn: func(_ context.Context, _ time.Time, keepLastN int) (int64, error) {
			assert.Equal(t, 5, keepLastN, "default keep_last_n=5")
			return 0, nil
		},
	}
	// monPolicyRepo with current=nil makes Get return ErrNotFound;
	// PolicyService.Get translates that to DefaultPolicy().
	policySvc := NewPolicyService(&monPolicyRepo{current: nil}, zap.NewNop())
	mon := NewPolicyMonitor(policySvc, taskRepo, NewPolicyMetrics(nil), zap.NewNop())

	_, err := mon.RunCleanupOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, taskRepo.cleanupCalls)
}

func TestPolicyMonitor_StartStop(t *testing.T) {
	// V3: cron lifecycle smoke test (don't wait for tick — just confirm no panic)
	mon := NewPolicyMonitor(
		NewPolicyService(&monPolicyRepo{current: DefaultPolicy()}, zap.NewNop()),
		&monTaskRepo{},
		NewPolicyMetrics(nil),
		zap.NewNop(),
	)
	require.NoError(t, mon.Start(context.Background()))
	mon.Stop()
}
