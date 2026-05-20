package backup

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
