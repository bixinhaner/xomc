package backup

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// monTaskRepo is a focused mock satisfying TaskRepository for cleanup tests.
// T-0076 widened CleanupOldRows return values to include the file_paths of
// deleted rows; the cleanupFn closure mirrors that.
type monTaskRepo struct {
	cleanupFn    func(ctx context.Context, cutoff time.Time, keepLastN int) ([]string, int64, error)
	cleanupCalls int
	gotCutoff    time.Time
	gotKeep      int
}

func (m *monTaskRepo) Create(_ context.Context, _ *BackupTask) error               { return nil }
func (m *monTaskRepo) GetByID(_ context.Context, _ uuid.UUID) (*BackupTask, error) { return nil, nil }
func (m *monTaskRepo) Update(_ context.Context, _ *BackupTask) error               { return nil }
func (m *monTaskRepo) Delete(_ context.Context, _ uuid.UUID) error                 { return nil }
func (m *monTaskRepo) List(_ context.Context, _ TaskFilter) (*model.ListResponse[BackupTask], error) {
	return nil, nil
}
func (m *monTaskRepo) CleanupOldRows(ctx context.Context, cutoff time.Time, keepLastN int) ([]string, int64, error) {
	m.cleanupCalls++
	m.gotCutoff = cutoff
	m.gotKeep = keepLastN
	if m.cleanupFn != nil {
		return m.cleanupFn(ctx, cutoff, keepLastN)
	}
	return nil, 0, nil
}
func (m *monTaskRepo) UpdateFilePath(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (m *monTaskRepo) FindByIDPrefix(_ context.Context, _ string, _ int) ([]*BackupTask, error) {
	return nil, nil
}
func (m *monTaskRepo) MarkComplete(_ context.Context, _ uuid.UUID, _ TaskStatus, _ int16, _ time.Time, _ string) error {
	return nil
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

// fakeMinIORemover records each RemoveObject call and yields a configured
// error per index so tests can drive success / not-found / network mixes.
type fakeMinIORemover struct {
	calls    []removeCall
	errsByIx []error
}

type removeCall struct {
	bucket string
	object string
}

func (f *fakeMinIORemover) RemoveObject(_ context.Context, bucket, object string, _ minio.RemoveObjectOptions) error {
	idx := len(f.calls)
	f.calls = append(f.calls, removeCall{bucket: bucket, object: object})
	if idx < len(f.errsByIx) {
		return f.errsByIx[idx]
	}
	return nil
}

// TestPolicyMonitor_RunCleanupOnce_PhysicalDelete_AllSuccess verifies that
// when CleanupOldRows returns N file_paths, RunCleanupOnce calls
// MinIO.RemoveObject N times in order with each path split correctly into
// (bucket, object_path).
func TestPolicyMonitor_RunCleanupOnce_PhysicalDelete_AllSuccess(t *testing.T) {
	taskRepo := &monTaskRepo{
		cleanupFn: func(_ context.Context, _ time.Time, _ int) ([]string, int64, error) {
			return []string{
				"config_backup/backup/2026/04/29/a.xml.gz",
				"config_backup/backup/2026/04/28/b.xml.zst",
			}, 2, nil
		},
	}
	policySvc := NewPolicyService(&monPolicyRepo{current: &BackupPolicy{
		AutoCleanup: true, RetentionDays: 30, KeepLastN: 5,
		CleanupTime: "03:00", CompressionLevel: 6, CompressionFormat: "gzip",
		StorageBackend: "local", LocalPath: "/var/backup/omc", MaxStorageGB: 500,
		EncryptionAlgorithm: "AES-256-GCM", MaxBackupCount: 100, MinBackupCount: 3,
		AlertThresholdPercent: 80,
	}}, zap.NewNop())
	mon := NewPolicyMonitor(policySvc, taskRepo, NewPolicyMetrics(nil), zap.NewNop())
	mr := &fakeMinIORemover{}
	mon.SetMinIO(mr)

	deleted, err := mon.RunCleanupOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)
	require.Len(t, mr.calls, 2, "RemoveObject called once per non-empty file_path")
	assert.Equal(t, "config_backup", mr.calls[0].bucket)
	assert.Equal(t, "backup/2026/04/29/a.xml.gz", mr.calls[0].object)
	assert.Equal(t, "config_backup", mr.calls[1].bucket)
	assert.Equal(t, "backup/2026/04/28/b.xml.zst", mr.calls[1].object)
}

// TestPolicyMonitor_RunCleanupOnce_PhysicalDelete_PartialFailure verifies
// best-effort semantics: a NotFound on path[0] and a network error on path[1]
// don't abort cleanup; path[2]'s RemoveObject is still called and the DB
// cleanup result is preserved.
func TestPolicyMonitor_RunCleanupOnce_PhysicalDelete_PartialFailure(t *testing.T) {
	taskRepo := &monTaskRepo{
		cleanupFn: func(_ context.Context, _ time.Time, _ int) ([]string, int64, error) {
			return []string{
				"config_backup/missing.xml",
				"config_backup/network.xml",
				"config_backup/ok.xml",
			}, 5, nil
		},
	}
	policySvc := NewPolicyService(&monPolicyRepo{current: &BackupPolicy{
		AutoCleanup: true, RetentionDays: 30, KeepLastN: 5,
		CleanupTime: "03:00", CompressionLevel: 6, CompressionFormat: "gzip",
		StorageBackend: "local", LocalPath: "/var/backup/omc", MaxStorageGB: 500,
		EncryptionAlgorithm: "AES-256-GCM", MaxBackupCount: 100, MinBackupCount: 3,
		AlertThresholdPercent: 80,
	}}, zap.NewNop())
	mon := NewPolicyMonitor(policySvc, taskRepo, NewPolicyMetrics(nil), zap.NewNop())
	mr := &fakeMinIORemover{
		errsByIx: []error{
			minio.ErrorResponse{Code: "NoSuchKey"},
			errors.New("dial tcp 10.0.0.1:9000: connection refused"),
			nil,
		},
	}
	mon.SetMinIO(mr)

	deleted, err := mon.RunCleanupOnce(context.Background())
	require.NoError(t, err, "physical delete failures must not propagate")
	assert.Equal(t, int64(5), deleted, "DB cleanup count preserved")
	assert.Len(t, mr.calls, 3, "all 3 RemoveObject calls attempted despite mid-stream errors")
}

// TestPolicyMonitor_RunCleanupOnce_PhysicalDelete_MalformedPathSkipped
// covers the parse_path branch — file_path without "/" separator should
// produce a metric error and skip RemoveObject without aborting the loop.
func TestPolicyMonitor_RunCleanupOnce_PhysicalDelete_MalformedPathSkipped(t *testing.T) {
	taskRepo := &monTaskRepo{
		cleanupFn: func(_ context.Context, _ time.Time, _ int) ([]string, int64, error) {
			return []string{
				"malformed_no_separator",
				"config_backup/ok.xml",
			}, 2, nil
		},
	}
	policySvc := NewPolicyService(&monPolicyRepo{current: &BackupPolicy{
		AutoCleanup: true, RetentionDays: 30, KeepLastN: 5,
		CleanupTime: "03:00", CompressionLevel: 6, CompressionFormat: "gzip",
		StorageBackend: "local", LocalPath: "/var/backup/omc", MaxStorageGB: 500,
		EncryptionAlgorithm: "AES-256-GCM", MaxBackupCount: 100, MinBackupCount: 3,
		AlertThresholdPercent: 80,
	}}, zap.NewNop())
	mon := NewPolicyMonitor(policySvc, taskRepo, NewPolicyMetrics(nil), zap.NewNop())
	mr := &fakeMinIORemover{}
	mon.SetMinIO(mr)

	_, err := mon.RunCleanupOnce(context.Background())
	require.NoError(t, err)
	require.Len(t, mr.calls, 1, "malformed path must NOT call RemoveObject; only valid path proceeds")
	assert.Equal(t, "config_backup", mr.calls[0].bucket)
}

// TestPolicyMonitor_RunCleanupOnce_PhysicalDelete_NoMinIONoCalls verifies
// the back-compat branch: if SetMinIO is never called, cleanup falls back
// to T-0073 DB-only behaviour with no panics or errors.
func TestPolicyMonitor_RunCleanupOnce_PhysicalDelete_NoMinIONoCalls(t *testing.T) {
	taskRepo := &monTaskRepo{
		cleanupFn: func(_ context.Context, _ time.Time, _ int) ([]string, int64, error) {
			return []string{"config_backup/x.xml"}, 1, nil
		},
	}
	policySvc := NewPolicyService(&monPolicyRepo{current: &BackupPolicy{
		AutoCleanup: true, RetentionDays: 30, KeepLastN: 5,
		CleanupTime: "03:00", CompressionLevel: 6, CompressionFormat: "gzip",
		StorageBackend: "local", LocalPath: "/var/backup/omc", MaxStorageGB: 500,
		EncryptionAlgorithm: "AES-256-GCM", MaxBackupCount: 100, MinBackupCount: 3,
		AlertThresholdPercent: 80,
	}}, zap.NewNop())
	mon := NewPolicyMonitor(policySvc, taskRepo, NewPolicyMetrics(nil), zap.NewNop())
	// No mon.SetMinIO call — physical delete disabled.

	deleted, err := mon.RunCleanupOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)
}

func TestClassifyMinIORemoveErr(t *testing.T) {
	assert.Equal(t, "", classifyMinIORemoveErr(nil))
	assert.Equal(t, "not_found", classifyMinIORemoveErr(minio.ErrorResponse{Code: "NoSuchKey"}))
	assert.Equal(t, "not_found", classifyMinIORemoveErr(minio.ErrorResponse{Code: "NoSuchBucket"}))
	assert.Equal(t, "network", classifyMinIORemoveErr(errors.New("dial tcp: connection refused")))
	assert.Equal(t, "network", classifyMinIORemoveErr(errors.New("i/o timeout")))
	assert.Equal(t, "network", classifyMinIORemoveErr(context.DeadlineExceeded))
	assert.Equal(t, "other", classifyMinIORemoveErr(errors.New("access denied")))
}

func TestPolicyMonitor_RunCleanupOnce_DeletesOldRows(t *testing.T) {
	// V4: retentionDays=30; cleanup deletes 3 rows
	taskRepo := &monTaskRepo{
		cleanupFn: func(_ context.Context, _ time.Time, _ int) ([]string, int64, error) {
			return nil, 3, nil
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
		cleanupFn: func(_ context.Context, _ time.Time, keepLastN int) ([]string, int64, error) {
			assert.Equal(t, 3, keepLastN)
			return nil, 4, nil
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
		cleanupFn: func(_ context.Context, _ time.Time, _ int) ([]string, int64, error) {
			t.Fatal("CleanupOldRows must not be called when auto_cleanup=false")
			return nil, 0, nil
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
		cleanupFn: func(_ context.Context, _ time.Time, keepLastN int) ([]string, int64, error) {
			assert.Equal(t, 5, keepLastN, "default keep_last_n=5")
			return nil, 0, nil
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
