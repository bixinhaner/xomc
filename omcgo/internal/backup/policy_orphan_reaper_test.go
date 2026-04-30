package backup

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// fakeTaskIDLister is the minimal TaskIDLister stub: pre-populated map of
// 8-char hex prefixes; optional err for V7 fail-open behaviour.
type fakeTaskIDLister struct {
	live map[string]struct{}
	err  error
}

func (f *fakeTaskIDLister) ListAllTaskIDPrefixes(_ context.Context) (map[string]struct{}, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.live == nil {
		return map[string]struct{}{}, nil
	}
	cp := make(map[string]struct{}, len(f.live))
	for k := range f.live {
		cp[k] = struct{}{}
	}
	return cp, nil
}

// reaperTestPolicy returns a fully-populated policy where RetentionDays
// drives the reaper's age cutoff (cutoff = now - (RetentionDays+1)d).
func reaperTestPolicy(retentionDays int) *BackupPolicy {
	return &BackupPolicy{
		AutoCleanup: true, RetentionDays: retentionDays, KeepLastN: 5,
		CleanupTime: "03:00", CompressionLevel: 6, CompressionFormat: "gzip",
		StorageBackend: "local", LocalPath: "/var/backup/omc", MaxStorageGB: 500,
		EncryptionAlgorithm: "AES-256-GCM", MaxBackupCount: 100, MinBackupCount: 3,
		AlertThresholdPercent: 80,
	}
}

// makeReaper builds a PolicyMonitor with all three reaper deps wired and
// the supplied bucket contents / live-set / mocks. Returns the monitor
// plus the remover so tests can inspect its calls slice.
func makeReaper(t *testing.T, retention int, objects []minio.ObjectInfo, live map[string]struct{}) (*PolicyMonitor, *fakeMinIORemover) {
	t.Helper()
	policySvc := NewPolicyService(&monPolicyRepo{current: reaperTestPolicy(retention)}, zap.NewNop())
	mon := NewPolicyMonitor(policySvc, &monTaskRepo{}, NewPolicyMetrics(nil), zap.NewNop())
	mon.SetBucketLister(&fakeBucketLister{objects: objects})
	remover := &fakeMinIORemover{}
	mon.SetMinIO(remover)
	mon.SetTaskIDLister(&fakeTaskIDLister{live: live})
	return mon, remover
}

// helper: an object key for retention bucket layout.
func obj(key string, ageDays int) minio.ObjectInfo {
	return minio.ObjectInfo{Key: key, LastModified: time.Now().Add(-time.Duration(ageDays) * 24 * time.Hour)}
}

// ---------------------------------------------------------------------------
// V1 — orphan reap success path
// ---------------------------------------------------------------------------

func TestOrphanReaper_V1_SuccessPath(t *testing.T) {
	mon, remover := makeReaper(t, 30,
		[]minio.ObjectInfo{
			obj("backup/2026/03/29/backup-aabbccdd-SN999.xml.gz", 35),
		},
		map[string]struct{}{},
	)

	reaped, err := mon.RunOrphanReaperOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, reaped)
	require.Len(t, remover.calls, 1)
	assert.Equal(t, "config_backup", remover.calls[0].bucket)
	assert.Equal(t, "backup/2026/03/29/backup-aabbccdd-SN999.xml.gz", remover.calls[0].object)
}

// ---------------------------------------------------------------------------
// V2 — live task NOT reaped
// ---------------------------------------------------------------------------

func TestOrphanReaper_V2_LiveTaskSkipped(t *testing.T) {
	mon, remover := makeReaper(t, 30,
		[]minio.ObjectInfo{
			obj("backup/2026/03/29/backup-11223344-SN001.xml.gz", 60),
		},
		map[string]struct{}{"11223344": {}},
	)

	reaped, err := mon.RunOrphanReaperOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, reaped)
	assert.Empty(t, remover.calls, "live-set member must not be deleted")
}

// ---------------------------------------------------------------------------
// V3 — younger than retention+1 days NOT reaped
// ---------------------------------------------------------------------------

func TestOrphanReaper_V3_AgeRecentSkipped(t *testing.T) {
	mon, remover := makeReaper(t, 30,
		[]minio.ObjectInfo{
			obj("backup/2026/04/29/backup-deadbeef-SN777.xml.gz", 5),
		},
		map[string]struct{}{},
	)

	reaped, err := mon.RunOrphanReaperOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, reaped)
	assert.Empty(t, remover.calls, "object newer than RetentionDays+1 must not be deleted")
}

// ---------------------------------------------------------------------------
// V4 — non-canonical pattern NOT reaped
// ---------------------------------------------------------------------------

func TestOrphanReaper_V4_PatternMismatchSkipped(t *testing.T) {
	mon, remover := makeReaper(t, 30,
		[]minio.ObjectInfo{
			obj("unrelated/manual-upload.xml", 60),
			obj("backup/garbage.txt", 60),
			obj("backup/2026/03/29/backup-XYZ-SN.xml", 60), // hex-only required
		},
		map[string]struct{}{},
	)

	reaped, err := mon.RunOrphanReaperOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, reaped)
	assert.Empty(t, remover.calls, "operator-uploaded / non-canonical objects must be left alone")
}

// ---------------------------------------------------------------------------
// V5 — max-K cap stops at maxReapPerRun
// ---------------------------------------------------------------------------

func TestOrphanReaper_V5_MaxKCap(t *testing.T) {
	objs := make([]minio.ObjectInfo, 1500)
	for i := range objs {
		objs[i] = obj(fmt.Sprintf("backup/2026/03/29/backup-%08x-SN%04d.xml.gz", i, i), 60)
	}
	mon, remover := makeReaper(t, 30, objs, map[string]struct{}{})

	reaped, err := mon.RunOrphanReaperOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, maxReapPerRun, reaped, "reap count must equal the cap when candidates exceed it")
	assert.Len(t, remover.calls, maxReapPerRun)
}

// ---------------------------------------------------------------------------
// V6 — RemoveObject API error does NOT abort scan
// ---------------------------------------------------------------------------

func TestOrphanReaper_V6_RemoveObjectErrorContinues(t *testing.T) {
	objs := []minio.ObjectInfo{
		obj("backup/2026/03/29/backup-aaaaaaaa-SN1.xml.gz", 60),
		obj("backup/2026/03/29/backup-bbbbbbbb-SN2.xml.gz", 60),
		obj("backup/2026/03/29/backup-cccccccc-SN3.xml.gz", 60),
	}
	policySvc := NewPolicyService(&monPolicyRepo{current: reaperTestPolicy(30)}, zap.NewNop())
	mon := NewPolicyMonitor(policySvc, &monTaskRepo{}, NewPolicyMetrics(nil), zap.NewNop())
	mon.SetBucketLister(&fakeBucketLister{objects: objs})
	remover := &fakeMinIORemover{
		errsByIx: []error{nil, errors.New("dial tcp: connection refused"), nil},
	}
	mon.SetMinIO(remover)
	mon.SetTaskIDLister(&fakeTaskIDLister{live: map[string]struct{}{}})

	reaped, err := mon.RunOrphanReaperOnce(context.Background())
	require.NoError(t, err, "API errors are tallied as skipped, not propagated")
	assert.Equal(t, 2, reaped, "2 successes; middle one tallied as api_error")
	assert.Len(t, remover.calls, 3, "all 3 candidates attempted despite mid-stream error")
}

// ---------------------------------------------------------------------------
// V7 — ListObjects fail returns err; no panic; no false reap
// ---------------------------------------------------------------------------

func TestOrphanReaper_V7_ListBucketErrorFailOpen(t *testing.T) {
	policySvc := NewPolicyService(&monPolicyRepo{current: reaperTestPolicy(30)}, zap.NewNop())
	mon := NewPolicyMonitor(policySvc, &monTaskRepo{}, NewPolicyMetrics(nil), zap.NewNop())
	mon.SetBucketLister(&fakeBucketLister{err: errors.New("minio unavailable")})
	remover := &fakeMinIORemover{}
	mon.SetMinIO(remover)
	mon.SetTaskIDLister(&fakeTaskIDLister{live: map[string]struct{}{}})

	reaped, err := mon.RunOrphanReaperOnce(context.Background())
	require.Error(t, err)
	assert.Equal(t, 0, reaped)
	assert.Empty(t, remover.calls, "list error must not trigger any RemoveObject")
}

// ---------------------------------------------------------------------------
// V8 — nil-safe: any of {bucketLister, minio, taskIDLister} unwired ⇒ no-op
// ---------------------------------------------------------------------------

func TestOrphanReaper_V8_NilSafeNoop(t *testing.T) {
	t.Run("no_bucket_lister", func(t *testing.T) {
		mon := NewPolicyMonitor(NewPolicyService(&monPolicyRepo{current: reaperTestPolicy(30)}, zap.NewNop()),
			&monTaskRepo{}, NewPolicyMetrics(nil), zap.NewNop())
		mon.SetMinIO(&fakeMinIORemover{})
		mon.SetTaskIDLister(&fakeTaskIDLister{live: map[string]struct{}{}})

		reaped, err := mon.RunOrphanReaperOnce(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 0, reaped)
	})

	t.Run("no_minio", func(t *testing.T) {
		mon := NewPolicyMonitor(NewPolicyService(&monPolicyRepo{current: reaperTestPolicy(30)}, zap.NewNop()),
			&monTaskRepo{}, NewPolicyMetrics(nil), zap.NewNop())
		mon.SetBucketLister(&fakeBucketLister{})
		mon.SetTaskIDLister(&fakeTaskIDLister{live: map[string]struct{}{}})

		reaped, err := mon.RunOrphanReaperOnce(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 0, reaped)
	})

	t.Run("no_task_id_lister", func(t *testing.T) {
		mon := NewPolicyMonitor(NewPolicyService(&monPolicyRepo{current: reaperTestPolicy(30)}, zap.NewNop()),
			&monTaskRepo{}, NewPolicyMetrics(nil), zap.NewNop())
		mon.SetBucketLister(&fakeBucketLister{})
		mon.SetMinIO(&fakeMinIORemover{})

		reaped, err := mon.RunOrphanReaperOnce(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 0, reaped)
	})
}

// ---------------------------------------------------------------------------
// V9 — cron Start integration: 3 schedules registered (cleanup + storage + reaper)
// ---------------------------------------------------------------------------

func TestOrphanReaper_V9_CronLifecycle3Entries(t *testing.T) {
	mon := NewPolicyMonitor(
		NewPolicyService(&monPolicyRepo{current: DefaultPolicy()}, zap.NewNop()),
		&monTaskRepo{},
		NewPolicyMetrics(nil),
		zap.NewNop(),
	)
	require.NoError(t, mon.Start(context.Background()))
	defer mon.Stop()

	require.NotNil(t, mon.cron)
	assert.Len(t, mon.cron.Entries(), 3, "cleanup (@daily) + storage check (@hourly) + orphan reaper (@weekly)")
}

// ---------------------------------------------------------------------------
// V11 — taskIDLister error fail-open: returns err, no false reap
// (V11 covers the listLiveSet error path PRD §9.3 explicitly does check)
// ---------------------------------------------------------------------------

func TestOrphanReaper_V11_TaskIDListerErrorPropagates(t *testing.T) {
	policySvc := NewPolicyService(&monPolicyRepo{current: reaperTestPolicy(30)}, zap.NewNop())
	mon := NewPolicyMonitor(policySvc, &monTaskRepo{}, NewPolicyMetrics(nil), zap.NewNop())
	mon.SetBucketLister(&fakeBucketLister{})
	remover := &fakeMinIORemover{}
	mon.SetMinIO(remover)
	mon.SetTaskIDLister(&fakeTaskIDLister{err: errors.New("db down")})

	reaped, err := mon.RunOrphanReaperOnce(context.Background())
	require.Error(t, err)
	assert.Equal(t, 0, reaped)
	assert.Empty(t, remover.calls, "live-set error must short-circuit before any deletion")
}

// ---------------------------------------------------------------------------
// V12 — parseBackupFilename unit coverage (regex behaviour pinned)
// ---------------------------------------------------------------------------

func TestParseBackupFilename(t *testing.T) {
	cases := []struct {
		key  string
		want string
		ok   bool
	}{
		{"backup/2026/03/29/backup-aabbccdd-SN999.xml.gz", "aabbccdd", true},
		{"backup/2026/03/29/backup-aabbccdd-SN999.xml.zst", "aabbccdd", true},
		{"backup/2026/03/29/backup-aabbccdd-SN999.xml.gz.enc", "aabbccdd", true},
		{"backup-deadbeef-SN001.xml", "deadbeef", true},
		{"manual-upload.xml", "", false},
		{"backup/2026/03/29/backup-XYZ-SN.xml", "", false},      // hex-only required
		{"backup-aabbccdd-.xml", "", false},                     // empty deviceSN segment (.+? non-greedy needs ≥1)
		{"backup/2026/03/29/backup-AABBCCDD-SN.xml", "", false}, // case-sensitive lowercase
	}
	for _, c := range cases {
		t.Run(c.key, func(t *testing.T) {
			got, ok := parseBackupFilename(c.key)
			assert.Equal(t, c.ok, ok)
			assert.Equal(t, c.want, got)
		})
	}
}
