package backup

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

// fakePrefixRepo focuses on FindByIDPrefix + UpdateFilePath; everything else
// is no-op. Each test pre-loads `byPrefix` so behaviour is deterministic.
type fakePrefixRepo struct {
	byPrefix    map[string][]*BackupTask
	findErr     error
	updateErr   error
	updateCalls []updateCall
}

type updateCall struct {
	id       uuid.UUID
	filePath string
}

func (m *fakePrefixRepo) Create(_ context.Context, _ *BackupTask) error { return nil }
func (m *fakePrefixRepo) GetByID(_ context.Context, id uuid.UUID) (*BackupTask, error) {
	for _, batch := range m.byPrefix {
		for _, t := range batch {
			if t.ID == id {
				return t, nil
			}
		}
	}
	return nil, commonerrors.ErrNotFound
}
func (m *fakePrefixRepo) Update(_ context.Context, _ *BackupTask) error { return nil }
func (m *fakePrefixRepo) Delete(_ context.Context, _ uuid.UUID) error   { return nil }
func (m *fakePrefixRepo) List(_ context.Context, _ TaskFilter) (*model.ListResponse[BackupTask], error) {
	return nil, nil
}
func (m *fakePrefixRepo) CleanupOldRows(_ context.Context, _ time.Time, _ int) ([]string, int64, error) {
	return nil, 0, nil
}
func (m *fakePrefixRepo) UpdateFilePath(_ context.Context, id uuid.UUID, filePath string) error {
	m.updateCalls = append(m.updateCalls, updateCall{id, filePath})
	return m.updateErr
}
func (m *fakePrefixRepo) FindByIDPrefix(_ context.Context, prefix string, _ int) ([]*BackupTask, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.byPrefix[prefix], nil
}
func (m *fakePrefixRepo) MarkComplete(_ context.Context, _ uuid.UUID, _ TaskStatus, _ int16, _ time.Time, _ string) error {
	return nil
}

func newRecorder(t *testing.T, repo *fakePrefixRepo) *FilePathRecorder {
	t.Helper()
	return NewFilePathRecorder(repo, NewRestoreMetrics(nil), zap.NewNop())
}

type fakeMetadataRepo struct {
	rows []BackupRestoreFile
	next int64
}

func (m *fakeMetadataRepo) Upsert(_ context.Context, f *BackupRestoreFile) error {
	if f == nil {
		return errors.New("nil metadata")
	}
	taskID := ""
	if f.TaskID != nil {
		taskID = *f.TaskID
	}
	for i := range m.rows {
		rowTaskID := ""
		if m.rows[i].TaskID != nil {
			rowTaskID = *m.rows[i].TaskID
		}
		if m.rows[i].SerialNumber == f.SerialNumber && rowTaskID == taskID && m.rows[i].FileName == f.FileName {
			m.rows[i].ObjectPath = f.ObjectPath
			m.rows[i].MD5 = f.MD5
			m.rows[i].FileSize = f.FileSize
			m.rows[i].OperatorCode = f.OperatorCode
			m.rows[i].TaskID = f.TaskID
			m.rows[i].UpdateTime = time.Now()
			m.rows[i].IsDeleted = false
			m.rows[i].DeletedAt = nil
			*f = m.rows[i]
			return nil
		}
	}
	m.next++
	f.ID = m.next
	f.UpdateTime = time.Now()
	f.CreatedAt = f.UpdateTime
	f.IsDeleted = false
	m.rows = append(m.rows, *f)
	return nil
}

func (m *fakeMetadataRepo) ListBySerial(_ context.Context, sn string) ([]BackupRestoreFile, error) {
	var out []BackupRestoreFile
	for _, row := range m.rows {
		if row.SerialNumber == sn {
			out = append(out, row)
		}
	}
	return out, nil
}

func (m *fakeMetadataRepo) ListByTaskID(_ context.Context, taskID string) ([]BackupRestoreFile, error) {
	var out []BackupRestoreFile
	for _, row := range m.rows {
		if row.TaskID != nil && *row.TaskID == taskID {
			out = append(out, row)
		}
	}
	return out, nil
}

func (m *fakeMetadataRepo) DeleteByTaskID(_ context.Context, taskID string) error {
	kept := m.rows[:0]
	for _, row := range m.rows {
		if row.TaskID != nil && *row.TaskID == taskID {
			continue
		}
		kept = append(kept, row)
	}
	m.rows = kept
	return nil
}

func (m *fakeMetadataRepo) CountActiveLogFiles(context.Context) (int64, error) {
	var n int64
	for _, row := range m.rows {
		if !row.IsDeleted && isQuotaManagedLogObjectPath(row.ObjectPath) {
			n++
		}
	}
	return n, nil
}

func (m *fakeMetadataRepo) CountActiveLogFilesBySerial(_ context.Context, sn string) (int64, error) {
	var n int64
	for _, row := range m.rows {
		if !row.IsDeleted && row.SerialNumber == sn && isQuotaManagedLogObjectPath(row.ObjectPath) {
			n++
		}
	}
	return n, nil
}

func (m *fakeMetadataRepo) ListOldestActiveLogFiles(_ context.Context, limit int) ([]BackupRestoreFile, error) {
	return m.oldest(func(row BackupRestoreFile) bool {
		return !row.IsDeleted && isQuotaManagedLogObjectPath(row.ObjectPath)
	}, limit), nil
}

func (m *fakeMetadataRepo) ListOldestActiveLogFilesBySerial(_ context.Context, sn string, limit int) ([]BackupRestoreFile, error) {
	return m.oldest(func(row BackupRestoreFile) bool {
		return !row.IsDeleted && row.SerialNumber == sn && isQuotaManagedLogObjectPath(row.ObjectPath)
	}, limit), nil
}

func (m *fakeMetadataRepo) MarkFileDeleted(_ context.Context, id int64) error {
	now := time.Now()
	for i := range m.rows {
		if m.rows[i].ID == id {
			m.rows[i].IsDeleted = true
			m.rows[i].DeletedAt = &now
			return nil
		}
	}
	return nil
}

func (m *fakeMetadataRepo) oldest(match func(BackupRestoreFile) bool, limit int) []BackupRestoreFile {
	if limit <= 0 {
		return nil
	}
	out := make([]BackupRestoreFile, 0, limit)
	for _, row := range m.rows {
		if !match(row) {
			continue
		}
		out = append(out, row)
		if len(out) == limit {
			break
		}
	}
	return out
}

type fakeLogQuotaPolicy struct {
	max          int
	maxPerDevice int
}

func (p fakeLogQuotaPolicy) MaxFileCount(context.Context) int {
	return p.max
}

func (p fakeLogQuotaPolicy) MaxFileCountPerDevice(context.Context) int {
	return p.maxPerDevice
}

type fakeObjectRemover struct {
	removed []string
}

func (r *fakeObjectRemover) RemoveObject(_ context.Context, bucketName, objectName string, _ minio.RemoveObjectOptions) error {
	r.removed = append(r.removed, bucketName+"/"+objectName)
	return nil
}

// makeEvent constructs an Event with a BackupFileReceivedPayload-shaped map.
func makeEvent(t *testing.T, p map[string]interface{}) event.Event {
	t.Helper()
	evt, err := event.NewEvent(event.SubjectBackupFileReceived, p)
	require.NoError(t, err)
	return evt
}

func TestHandleFileReceived_recorded(t *testing.T) {
	id := uuid.New()
	task := &BackupTask{ID: id, TargetIDs: []string{"SN001"}}
	repo := &fakePrefixRepo{byPrefix: map[string][]*BackupTask{"abcdef12": {task}}}
	rec := newRecorder(t, repo)

	err := rec.handleFileReceived(context.Background(), makeEvent(t, map[string]interface{}{
		"bucket":                "config_backup",
		"object_path":           "backup/2026/04/29/backup-abcdef12-SN001.xml.gz",
		"filename":              "backup-abcdef12-SN001.xml.gz",
		"backup_task_id_prefix": "abcdef12",
		"device_sn":             "SN001",
		"file_size":             int64(2048),
	}))
	require.NoError(t, err)
	require.Len(t, repo.updateCalls, 1)
	assert.Equal(t, id, repo.updateCalls[0].id)
	assert.Equal(t, "config_backup/backup/2026/04/29/backup-abcdef12-SN001.xml.gz", repo.updateCalls[0].filePath)
}

func TestHandleFileReceived_skippedAlreadySet(t *testing.T) {
	id := uuid.New()
	existing := "config_backup/backup/2026/04/28/old.xml"
	task := &BackupTask{ID: id, FilePath: &existing, TargetIDs: []string{"SN001", "SN002"}}
	repo := &fakePrefixRepo{byPrefix: map[string][]*BackupTask{"deadbeef": {task}}}
	rec := newRecorder(t, repo)

	err := rec.handleFileReceived(context.Background(), makeEvent(t, map[string]interface{}{
		"bucket":                "config_backup",
		"object_path":           "backup/2026/04/29/backup-deadbeef-SN002.xml",
		"filename":              "backup-deadbeef-SN002.xml",
		"backup_task_id_prefix": "deadbeef",
		"device_sn":             "SN002",
		"file_size":             int64(1024),
	}))
	require.NoError(t, err)
	assert.Len(t, repo.updateCalls, 0, "first-write-wins; second device upload must NOT overwrite")
}

func TestHandleFileReceived_emptyPrefixSkipped(t *testing.T) {
	repo := &fakePrefixRepo{byPrefix: map[string][]*BackupTask{}}
	rec := newRecorder(t, repo)
	err := rec.handleFileReceived(context.Background(), makeEvent(t, map[string]interface{}{
		"bucket":                "config_backup",
		"object_path":           "backup/2026/04/29/operator-uploaded.xml",
		"filename":              "operator-uploaded.xml",
		"backup_task_id_prefix": "",
		"device_sn":             "",
		"file_size":             int64(512),
	}))
	require.NoError(t, err, "unmatched filename pattern is no-op, not error")
	assert.Len(t, repo.updateCalls, 0)
}

func TestHandleFileReceived_noMatchSkipped(t *testing.T) {
	repo := &fakePrefixRepo{byPrefix: map[string][]*BackupTask{}}
	rec := newRecorder(t, repo)
	err := rec.handleFileReceived(context.Background(), makeEvent(t, map[string]interface{}{
		"bucket":                "config_backup",
		"object_path":           "backup/2026/04/29/backup-cafefe12-SN777.xml",
		"filename":              "backup-cafefe12-SN777.xml",
		"backup_task_id_prefix": "cafefe12",
		"device_sn":             "SN777",
		"file_size":             int64(2048),
	}))
	require.NoError(t, err, "no matching backup_task is no-op (task may have been cleaned up)")
	assert.Len(t, repo.updateCalls, 0)
}

func TestHandleFileReceived_dbErrorPropagates(t *testing.T) {
	repo := &fakePrefixRepo{findErr: errors.New("postgres outage")}
	rec := newRecorder(t, repo)
	err := rec.handleFileReceived(context.Background(), makeEvent(t, map[string]interface{}{
		"bucket":                "config_backup",
		"object_path":           "backup/x.xml",
		"filename":              "backup-abcdef12-SN001.xml",
		"backup_task_id_prefix": "abcdef12",
		"device_sn":             "SN001",
		"file_size":             int64(1),
	}))
	require.Error(t, err, "db errors must propagate so NATS can redeliver")
}

func TestHandleFileReceived_EnforcesLogFileQuotaOnBackupRestoreMetadata(t *testing.T) {
	repo := &fakePrefixRepo{byPrefix: map[string][]*BackupTask{}}
	meta := &fakeMetadataRepo{}
	remover := &fakeObjectRemover{}
	rec := newRecorder(t, repo)
	rec.SetFileRepository(meta)
	rec.SetLogFileQuota(meta, fakeLogQuotaPolicy{max: 10, maxPerDevice: 1}, remover)

	firstTaskID := uuid.New().String()
	secondTaskID := uuid.New().String()
	firstURL := url.URL{Path: "fault/2026/07/17/fault-SN-QUOTA-old.tar.gz"}
	secondURL := url.URL{Path: "fault/2026/07/17/fault-SN-QUOTA-new.tar.gz"}
	require.NoError(t, rec.handleFileReceived(context.Background(), makeEvent(t, map[string]interface{}{
		"bucket":                "logs",
		"object_path":           strings.TrimPrefix(firstURL.Path, "/"),
		"filename":              "fault-SN-QUOTA-old.tar.gz",
		"backup_task_id_prefix": firstTaskID[:8],
		"task_id":               firstTaskID,
		"device_sn":             "SN-QUOTA",
		"file_size":             int64(10),
	})))
	require.NoError(t, rec.handleFileReceived(context.Background(), makeEvent(t, map[string]interface{}{
		"bucket":                "logs",
		"object_path":           strings.TrimPrefix(secondURL.Path, "/"),
		"filename":              "fault-SN-QUOTA-new.tar.gz",
		"backup_task_id_prefix": secondTaskID[:8],
		"task_id":               secondTaskID,
		"device_sn":             "SN-QUOTA",
		"file_size":             int64(20),
	})))

	require.Len(t, meta.rows, 2, "限额清理不能删除任务文件元数据本身")
	assert.True(t, meta.rows[0].IsDeleted, "同设备超限后应软删最旧日志文件")
	assert.NotNil(t, meta.rows[0].DeletedAt)
	assert.False(t, meta.rows[1].IsDeleted)
	assert.Equal(t, []string{"logs/fault/2026/07/17/fault-SN-QUOTA-old.tar.gz"}, remover.removed)
}

func TestHandleFileReceived_FaultLogQuotaIgnoresConfigBackupMetadata(t *testing.T) {
	repo := &fakePrefixRepo{byPrefix: map[string][]*BackupTask{}}
	meta := &fakeMetadataRepo{}
	remover := &fakeObjectRemover{}
	rec := newRecorder(t, repo)
	rec.SetFileRepository(meta)
	rec.SetLogFileQuota(meta, fakeLogQuotaPolicy{max: 1, maxPerDevice: 1}, remover)

	configTaskID := uuid.New().String()
	logTaskID := uuid.New().String()
	require.NoError(t, rec.handleFileReceived(context.Background(), makeEvent(t, map[string]interface{}{
		"bucket":                "config_backup",
		"object_path":           "backup/2026/07/17/SN-CFG_CFG.xml",
		"filename":              "SN-CFG_CFG.xml",
		"backup_task_id_prefix": configTaskID[:8],
		"task_id":               configTaskID,
		"device_sn":             "SN-CFG",
		"file_size":             int64(10),
	})))
	require.NoError(t, rec.handleFileReceived(context.Background(), makeEvent(t, map[string]interface{}{
		"bucket":                "logs",
		"object_path":           "running/2026/07/17/runtime-SN-CFG.tar.gz",
		"filename":              "runtime-SN-CFG.tar.gz",
		"backup_task_id_prefix": logTaskID[:8],
		"task_id":               logTaskID,
		"device_sn":             "SN-CFG",
		"file_size":             int64(20),
	})))

	require.Len(t, meta.rows, 2)
	assert.False(t, meta.rows[0].IsDeleted, "配置备份不属于故障日志文件数配额")
	assert.False(t, meta.rows[1].IsDeleted)
	assert.Empty(t, remover.removed)
}

func TestHandleFileReceived_FaultLogQuotaIgnoresRunningLogs(t *testing.T) {
	repo := &fakePrefixRepo{byPrefix: map[string][]*BackupTask{}}
	meta := &fakeMetadataRepo{}
	remover := &fakeObjectRemover{}
	rec := newRecorder(t, repo)
	rec.SetFileRepository(meta)
	rec.SetLogFileQuota(meta, fakeLogQuotaPolicy{max: 1, maxPerDevice: 1}, remover)

	runningTaskID := uuid.New().String()
	faultTaskID := uuid.New().String()
	require.NoError(t, rec.handleFileReceived(context.Background(), makeEvent(t, map[string]interface{}{
		"bucket":                "logs",
		"object_path":           "running/2026/07/17/runtime-SN-BOUNDARY.tar.gz",
		"filename":              "runtime-SN-BOUNDARY.tar.gz",
		"backup_task_id_prefix": runningTaskID[:8],
		"task_id":               runningTaskID,
		"device_sn":             "SN-BOUNDARY",
		"file_size":             int64(10),
	})))
	require.NoError(t, rec.handleFileReceived(context.Background(), makeEvent(t, map[string]interface{}{
		"bucket":                "logs",
		"object_path":           "fault/2026/07/17/fault-SN-BOUNDARY.tar.gz",
		"filename":              "fault-SN-BOUNDARY.tar.gz",
		"backup_task_id_prefix": faultTaskID[:8],
		"task_id":               faultTaskID,
		"device_sn":             "SN-BOUNDARY",
		"file_size":             int64(20),
	})))

	require.Len(t, meta.rows, 2)
	assert.False(t, meta.rows[0].IsDeleted, "运行日志不应占用故障日志文件数配额")
	assert.False(t, meta.rows[1].IsDeleted, "只有 1 个故障日志时不应触发故障日志配额")
	assert.Empty(t, remover.removed)
}

// TestHandleFileReceived_updateAlreadySetIsSkip covers the CAS-lost branch
// (review HIGH fix): two concurrent recorders both observe null in the
// fast-path read; the loser's UpdateFilePath returns ErrFilePathAlreadySet
// (atomic CAS at DB layer) — recorder must classify as skipped_already_set,
// not error.
func TestHandleFileReceived_updateAlreadySetIsSkip(t *testing.T) {
	id := uuid.New()
	task := &BackupTask{ID: id, TargetIDs: []string{"SN001", "SN002"}}
	repo := &fakePrefixRepo{
		byPrefix:  map[string][]*BackupTask{"abcdef12": {task}},
		updateErr: ErrFilePathAlreadySet,
	}
	rec := newRecorder(t, repo)
	err := rec.handleFileReceived(context.Background(), makeEvent(t, map[string]interface{}{
		"bucket":                "config_backup",
		"object_path":           "backup/x.xml",
		"filename":              "backup-abcdef12-SN002.xml",
		"backup_task_id_prefix": "abcdef12",
		"device_sn":             "SN002",
		"file_size":             int64(1),
	}))
	require.NoError(t, err, "CAS-lost is benign first-write-wins skip, not error")
}

func TestHandleFileReceived_updateNotFoundIsSkip(t *testing.T) {
	id := uuid.New()
	task := &BackupTask{ID: id, TargetIDs: []string{"SN001"}}
	repo := &fakePrefixRepo{
		byPrefix:  map[string][]*BackupTask{"abcdef12": {task}},
		updateErr: commonerrors.ErrNotFound, // raced with cleanup
	}
	rec := newRecorder(t, repo)
	err := rec.handleFileReceived(context.Background(), makeEvent(t, map[string]interface{}{
		"bucket":                "config_backup",
		"object_path":           "backup/x.xml",
		"filename":              "backup-abcdef12-SN001.xml",
		"backup_task_id_prefix": "abcdef12",
		"device_sn":             "SN001",
		"file_size":             int64(1),
	}))
	require.NoError(t, err, "race with cleanup is benign; treat as skip")
}

// ─────────────────────────────────────────────────────────────────────────
// B3: SnapshotPromoter hook tests
// ─────────────────────────────────────────────────────────────────────────

// fakeSnapshotPromoter 记录 PromoteFromBackup 调用，可注入失败错误。
type fakeSnapshotPromoter struct {
	calls   []promoteCall
	failErr error
}

type promoteCall struct {
	taskID uuid.UUID
	sn     string
}

func (p *fakeSnapshotPromoter) PromoteFromBackup(_ context.Context, taskID uuid.UUID, sn string) error {
	p.calls = append(p.calls, promoteCall{taskID, sn})
	return p.failErr
}

func TestHandleFileReceived_PromoterCalledOnRecorded(t *testing.T) {
	id := uuid.New()
	task := &BackupTask{ID: id, TargetIDs: []string{"SN001"}}
	repo := &fakePrefixRepo{byPrefix: map[string][]*BackupTask{"abcdef12": {task}}}
	rec := newRecorder(t, repo)
	promoter := &fakeSnapshotPromoter{}
	rec.SetSnapshotPromoter(promoter)

	err := rec.handleFileReceived(context.Background(), makeEvent(t, map[string]interface{}{
		"bucket":                "config_backup",
		"object_path":           "backup/2026/05/22/backup-abcdef12-SN001.xml",
		"filename":              "backup-abcdef12-SN001.xml",
		"backup_task_id_prefix": "abcdef12",
		"task_id":               id.String(),
		"device_sn":             "SN001",
		"file_size":             int64(2048),
	}))
	require.NoError(t, err)
	require.Len(t, promoter.calls, 1, "promote must be called on the recorded path")
	assert.Equal(t, id, promoter.calls[0].taskID,
		"promote 应使用 event payload.task_id 作为 backupTaskID")
	assert.Equal(t, "SN001", promoter.calls[0].sn)
}

func TestHandleFileReceived_PromoterCalledOnCASLost(t *testing.T) {
	id := uuid.New()
	task := &BackupTask{ID: id, TargetIDs: []string{"SN001", "SN002"}}
	repo := &fakePrefixRepo{
		byPrefix:  map[string][]*BackupTask{"abcdef12": {task}},
		updateErr: ErrFilePathAlreadySet,
	}
	rec := newRecorder(t, repo)
	promoter := &fakeSnapshotPromoter{}
	rec.SetSnapshotPromoter(promoter)

	err := rec.handleFileReceived(context.Background(), makeEvent(t, map[string]interface{}{
		"bucket":                "config_backup",
		"object_path":           "backup/x-SN002.xml",
		"filename":              "backup-abcdef12-SN002.xml",
		"backup_task_id_prefix": "abcdef12",
		"device_sn":             "SN002",
		"file_size":             int64(2048),
	}))
	require.NoError(t, err)
	require.Len(t, promoter.calls, 1,
		"promote must run for the losing sibling too — its SN has its own snapshot row")
	assert.Equal(t, "SN002", promoter.calls[0].sn)
}

func TestHandleFileReceived_PromoterFailureDoesNotBreakMainFlow(t *testing.T) {
	id := uuid.New()
	task := &BackupTask{ID: id, TargetIDs: []string{"SN001"}}
	repo := &fakePrefixRepo{byPrefix: map[string][]*BackupTask{"abcdef12": {task}}}
	rec := newRecorder(t, repo)
	promoter := &fakeSnapshotPromoter{failErr: errors.New("simulated minio outage")}
	rec.SetSnapshotPromoter(promoter)

	err := rec.handleFileReceived(context.Background(), makeEvent(t, map[string]interface{}{
		"bucket":                "config_backup",
		"object_path":           "backup/x-SN001.xml",
		"filename":              "backup-abcdef12-SN001.xml",
		"backup_task_id_prefix": "abcdef12",
		"device_sn":             "SN001",
		"file_size":             int64(1024),
	}))
	require.NoError(t, err,
		"promote failure must not fail the EventBus handler — main backup chain stays green")
	require.Len(t, promoter.calls, 1)
	require.Len(t, repo.updateCalls, 1,
		"main UpdateFilePath path still ran before the promote hook")
}

func TestHandleFileReceived_PromoterNotCalledWhenUnwired(t *testing.T) {
	id := uuid.New()
	task := &BackupTask{ID: id, TargetIDs: []string{"SN001"}}
	repo := &fakePrefixRepo{byPrefix: map[string][]*BackupTask{"abcdef12": {task}}}
	rec := newRecorder(t, repo)
	// SetSnapshotPromoter not invoked — backward-compat path.

	err := rec.handleFileReceived(context.Background(), makeEvent(t, map[string]interface{}{
		"bucket":                "config_backup",
		"object_path":           "backup/x-SN001.xml",
		"filename":              "backup-abcdef12-SN001.xml",
		"backup_task_id_prefix": "abcdef12",
		"device_sn":             "SN001",
		"file_size":             int64(1024),
	}))
	require.NoError(t, err)
}

// TestHandleFileReceived_PromoterCalledForUFTEPath 验证 T-0164 核心修复：
// UFTE 链路（主任务在 software.upgrade_tasks，prefix 永远不命中 backup_tasks）
// 也必须 promote 到 config_snapshots，否则 config-snapshots bucket 永远是空的。
// 这是部署后真实观察到的 bug（"no backup_task matches prefix; metadata-only"
// 路径未触发 promote）。
func TestHandleFileReceived_PromoterCalledForUFTEPath(t *testing.T) {
	repo := &fakePrefixRepo{byPrefix: map[string][]*BackupTask{}} // backup_tasks 空
	rec := newRecorder(t, repo)
	promoter := &fakeSnapshotPromoter{}
	rec.SetSnapshotPromoter(promoter)
	ufteTaskID := uuid.New()

	err := rec.handleFileReceived(context.Background(), makeEvent(t, map[string]interface{}{
		"bucket":                "config_backup",
		"object_path":           "backup/2026/05/22/Config_xx_SN777.xml",
		"filename":              "Config_20260522.0611-0500_MMMM.SN777.xml",
		"backup_task_id_prefix": "cafefe12",
		"task_id":               ufteTaskID.String(), // UFTE main task UUID
		"device_sn":             "SN777",
		"file_size":             int64(512),
	}))
	require.NoError(t, err)
	require.Len(t, promoter.calls, 1,
		"UFTE 链路（backup_tasks 不命中）也必须触发 promote")
	assert.Equal(t, ufteTaskID, promoter.calls[0].taskID)
	assert.Equal(t, "SN777", promoter.calls[0].sn)
}
