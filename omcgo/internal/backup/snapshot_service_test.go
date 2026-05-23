package backup

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
)

// ─────────────────────────────────────────────────────────────────────────
// fakes
// ─────────────────────────────────────────────────────────────────────────

type fakeSnapshotRepo struct {
	rows      map[string]*ConfigSnapshot
	upsertErr error
	upserts   []ConfigSnapshot
	mu        sync.Mutex
}

func newFakeSnapshotRepo() *fakeSnapshotRepo {
	return &fakeSnapshotRepo{rows: map[string]*ConfigSnapshot{}}
}

func (r *fakeSnapshotRepo) Upsert(_ context.Context, s *ConfigSnapshot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.upsertErr != nil {
		return r.upsertErr
	}
	cp := *s
	r.rows[s.SerialNumber] = &cp
	r.upserts = append(r.upserts, cp)
	return nil
}

func (r *fakeSnapshotRepo) GetBySerialNumber(_ context.Context, sn string) (*ConfigSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if v, ok := r.rows[sn]; ok {
		cp := *v
		return &cp, nil
	}
	return nil, nil
}

func (r *fakeSnapshotRepo) BatchGetBySerialNumbers(_ context.Context, sns []string) (map[string]*ConfigSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[string]*ConfigSnapshot{}
	for _, sn := range sns {
		if v, ok := r.rows[sn]; ok {
			cp := *v
			out[sn] = &cp
		}
	}
	return out, nil
}

func (r *fakeSnapshotRepo) List(_ context.Context, _ SnapshotFilter) ([]ConfigSnapshot, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := make([]ConfigSnapshot, 0, len(r.rows))
	for _, v := range r.rows {
		items = append(items, *v)
	}
	return items, int64(len(items)), nil
}

func (r *fakeSnapshotRepo) Delete(_ context.Context, sn string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.rows, sn)
	return nil
}

func (r *fakeSnapshotRepo) BatchDelete(_ context.Context, sns []string) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	deleted := make([]string, 0, len(sns))
	for _, sn := range sns {
		if _, ok := r.rows[sn]; ok {
			delete(r.rows, sn)
			deleted = append(deleted, sn)
		}
	}
	return deleted, nil
}

// ----- mover -----

type copyCall struct {
	dst minio.CopyDestOptions
	src minio.CopySrcOptions
}

type putCall struct {
	bucket string
	object string
	body   []byte
	size   int64
}

type snapshotRemoveCall struct {
	bucket string
	object string
}

type fakeMover struct {
	copyErr    error
	putErr     error
	removeErr  error
	copyCalls  []copyCall
	putCalls   []putCall
	rmCalls    []snapshotRemoveCall
	mu         sync.Mutex
}

func (m *fakeMover) CopyObject(_ context.Context, dst minio.CopyDestOptions, src minio.CopySrcOptions) (minio.UploadInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.copyCalls = append(m.copyCalls, copyCall{dst, src})
	if m.copyErr != nil {
		return minio.UploadInfo{}, m.copyErr
	}
	return minio.UploadInfo{Bucket: dst.Bucket, Key: dst.Object}, nil
}

func (m *fakeMover) PutObject(_ context.Context, bucket, obj string, r io.Reader, size int64, _ minio.PutObjectOptions) (minio.UploadInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	body, _ := io.ReadAll(r)
	m.putCalls = append(m.putCalls, putCall{bucket, obj, body, size})
	if m.putErr != nil {
		return minio.UploadInfo{}, m.putErr
	}
	return minio.UploadInfo{Bucket: bucket, Key: obj}, nil
}

func (m *fakeMover) RemoveObject(_ context.Context, bucket, obj string, _ minio.RemoveObjectOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rmCalls = append(m.rmCalls, snapshotRemoveCall{bucket, obj})
	return m.removeErr
}

// ----- file lookup -----

type fakeFileLookup struct {
	bySN map[string][]BackupRestoreFile
	err  error
}

func (f *fakeFileLookup) ListBySerial(_ context.Context, sn string) ([]BackupRestoreFile, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.bySN[sn], nil
}

// ----- device lookup (snapshot-test only; restore_service_test.go has a
// different fakeDeviceLookup with knownSNs map) -----

type fakeSnapshotDeviceLookup struct {
	bySN map[string]*model.Device
}

func (f *fakeSnapshotDeviceLookup) GetBySerialNumber(_ context.Context, sn string) (*model.Device, error) {
	return f.bySN[sn], nil
}

// ─────────────────────────────────────────────────────────────────────────
// constructors
// ─────────────────────────────────────────────────────────────────────────

func newTestSnapshotService(t *testing.T,
	repo SnapshotRepository, mover SnapshotMover,
	fileLookup SnapshotBackupFileLookup, deviceLookup SnapshotDeviceLookup,
) *SnapshotService {
	t.Helper()
	return NewSnapshotService(repo, mover, fileLookup, deviceLookup, "config-snapshots", zap.NewNop())
}

// ─────────────────────────────────────────────────────────────────────────
// PromoteFromBackup tests
// ─────────────────────────────────────────────────────────────────────────

func TestPromoteFromBackup_Success(t *testing.T) {
	taskID := uuid.New()
	md5val := "abcdef0123"
	srcRow := BackupRestoreFile{
		SerialNumber: "SN001",
		FileName:     "backup-12345678-SN001.xml",
		ObjectPath:   "config_backup/backup/2026/05/22/backup-12345678-SN001.xml",
		MD5:          &md5val,
		FileSize:     2048,
		TaskID:       ptrStr(taskID.String()),
	}
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	fl := &fakeFileLookup{bySN: map[string][]BackupRestoreFile{"SN001": {srcRow}}}
	dl := &fakeSnapshotDeviceLookup{bySN: map[string]*model.Device{
		"SN001": {DeviceName: "eNB-001", ProductClass: "FAP/BSC"},
	}}
	svc := newTestSnapshotService(t, repo, mover, fl, dl)

	err := svc.PromoteFromBackup(context.Background(), taskID, "SN001")
	require.NoError(t, err)

	require.Len(t, mover.copyCalls, 1)
	cp := mover.copyCalls[0]
	assert.Equal(t, "config_backup", cp.src.Bucket)
	assert.Equal(t, "backup/2026/05/22/backup-12345678-SN001.xml", cp.src.Object)
	assert.Equal(t, "config-snapshots", cp.dst.Bucket)
	assert.Equal(t, "SN001_CFG.xml", cp.dst.Object)

	row := repo.rows["SN001"]
	require.NotNil(t, row)
	assert.Equal(t, "SN001_CFG.xml", row.FileName)
	assert.Equal(t, "xml", row.FileExt)
	assert.Equal(t, "config-snapshots", row.ObjectBucket)
	assert.Equal(t, "SN001_CFG.xml", row.ObjectPath)
	assert.Equal(t, SnapshotSourceBackup, row.Source)
	require.NotNil(t, row.SourceTaskID)
	assert.Equal(t, taskID, *row.SourceTaskID)
	require.NotNil(t, row.EnbName)
	assert.Equal(t, "eNB-001", *row.EnbName)
	require.NotNil(t, row.ProductType)
	assert.Equal(t, "FAP/BSC", *row.ProductType)
	assert.EqualValues(t, 2048, row.FileSize)
	require.NotNil(t, row.MD5)
	assert.Equal(t, md5val, *row.MD5)
}

func TestPromoteFromBackup_NVFile(t *testing.T) {
	taskID := uuid.New()
	srcRow := BackupRestoreFile{
		SerialNumber: "SN_NV1",
		FileName:     "mib-home-fap.nv",
		ObjectPath:   "config_backup/backup/mib-home-fap.nv",
		FileSize:     512,
		TaskID:       ptrStr(taskID.String()),
	}
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	fl := &fakeFileLookup{bySN: map[string][]BackupRestoreFile{"SN_NV1": {srcRow}}}
	svc := newTestSnapshotService(t, repo, mover, fl, nil)

	require.NoError(t, svc.PromoteFromBackup(context.Background(), taskID, "SN_NV1"))
	row := repo.rows["SN_NV1"]
	require.NotNil(t, row)
	assert.Equal(t, "SN_NV1_CFG.nv", row.FileName)
	assert.Equal(t, "nv", row.FileExt)
}

func TestPromoteFromBackup_NoSourceFile_ReturnsNotFound(t *testing.T) {
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	fl := &fakeFileLookup{bySN: map[string][]BackupRestoreFile{}}
	svc := newTestSnapshotService(t, repo, mover, fl, nil)

	err := svc.PromoteFromBackup(context.Background(), uuid.New(), "SN404")
	require.Error(t, err)
	assert.Empty(t, mover.copyCalls)
	assert.Empty(t, repo.upserts)
}

func TestPromoteFromBackup_CopyObjectFails_DoesNotUpsert(t *testing.T) {
	taskID := uuid.New()
	srcRow := BackupRestoreFile{
		SerialNumber: "SN001",
		FileName:     "backup-x-SN001.xml",
		ObjectPath:   "config_backup/backup/x.xml",
		TaskID:       ptrStr(taskID.String()),
	}
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{copyErr: errors.New("simulated minio failure")}
	fl := &fakeFileLookup{bySN: map[string][]BackupRestoreFile{"SN001": {srcRow}}}
	svc := newTestSnapshotService(t, repo, mover, fl, nil)

	err := svc.PromoteFromBackup(context.Background(), taskID, "SN001")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "simulated minio failure")
	assert.Empty(t, repo.upserts, "DB upsert should be skipped if CopyObject fails")
}

func TestPromoteFromBackup_FileLookupNotConfigured(t *testing.T) {
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	svc := newTestSnapshotService(t, repo, mover, nil, nil) // fileLookup=nil

	err := svc.PromoteFromBackup(context.Background(), uuid.New(), "SN001")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrPromoteNotConfigured))
}

func TestPromoteFromBackup_RejectsInvalidExtension(t *testing.T) {
	taskID := uuid.New()
	srcRow := BackupRestoreFile{
		SerialNumber: "SN001",
		FileName:     "backup.weirdext", // 既不是 xml 也不是 nv
		ObjectPath:   "config_backup/foo/backup.weirdext",
		TaskID:       ptrStr(taskID.String()),
	}
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	fl := &fakeFileLookup{bySN: map[string][]BackupRestoreFile{"SN001": {srcRow}}}
	svc := newTestSnapshotService(t, repo, mover, fl, nil)

	err := svc.PromoteFromBackup(context.Background(), taskID, "SN001")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidConfigFileExt))
	assert.Empty(t, mover.copyCalls, "must not call CopyObject when normalization fails")
}

func TestPromoteFromBackup_FallbackToLatestWhenTaskIDMismatch(t *testing.T) {
	taskID := uuid.New()
	// 两行：第一行（最新）task_id 与请求不一致，仍应被选中作为回退。
	row1 := BackupRestoreFile{
		SerialNumber: "SN001",
		FileName:     "backup-newer-SN001.xml",
		ObjectPath:   "config_backup/backup/newer.xml",
		TaskID:       ptrStr(uuid.New().String()), // 不匹配
	}
	row2 := BackupRestoreFile{
		SerialNumber: "SN001",
		FileName:     "backup-older-SN001.xml",
		ObjectPath:   "config_backup/backup/older.xml",
		TaskID:       ptrStr(uuid.New().String()),
	}
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	fl := &fakeFileLookup{bySN: map[string][]BackupRestoreFile{"SN001": {row1, row2}}}
	svc := newTestSnapshotService(t, repo, mover, fl, nil)

	require.NoError(t, svc.PromoteFromBackup(context.Background(), taskID, "SN001"))
	require.Len(t, mover.copyCalls, 1)
	assert.Equal(t, "config_backup/backup/newer.xml",
		mover.copyCalls[0].src.Bucket+"/"+mover.copyCalls[0].src.Object)
}

// ─────────────────────────────────────────────────────────────────────────
// ImportFromUpload tests
// ─────────────────────────────────────────────────────────────────────────

func TestImportFromUpload_AllSucceed(t *testing.T) {
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	svc := newTestSnapshotService(t, repo, mover, nil, nil)

	items := []SnapshotImportItem{
		{FileName: "SN001_CFG.xml", Content: []byte("<config/>")},
		{FileName: "SN002_CFG.nv", Content: []byte{0x01, 0x02, 0x03}},
	}
	res, err := svc.ImportFromUpload(context.Background(), items, "alice")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"SN001", "SN002"}, res.Succeeded)
	assert.Empty(t, res.Failed)

	require.Len(t, mover.putCalls, 2)
	require.Len(t, repo.upserts, 2)
	for _, row := range repo.upserts {
		assert.Equal(t, SnapshotSourceManualUpload, row.Source)
		require.NotNil(t, row.UpdateBy)
		assert.Equal(t, "alice", *row.UpdateBy)
		assert.NotEmpty(t, row.MD5)
	}
}

func TestImportFromUpload_PartialFailure_InvalidName(t *testing.T) {
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	svc := newTestSnapshotService(t, repo, mover, nil, nil)

	items := []SnapshotImportItem{
		{FileName: "SN001_CFG.xml", Content: []byte("ok")},
		{FileName: "weird.txt", Content: []byte("bad")},
		{FileName: "SN003_CFG.nv", Content: []byte("nvdata")},
	}
	res, err := svc.ImportFromUpload(context.Background(), items, "")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"SN001", "SN003"}, res.Succeeded)
	require.Len(t, res.Failed, 1)
	assert.Equal(t, "weird.txt", res.Failed[0].FileName)
	assert.Equal(t, ImportErrInvalidName, res.Failed[0].ErrorCode)
	assert.Contains(t, res.Failed[0].Message, "weird.txt")

	// Failed item should not produce MinIO Put or DB Upsert.
	assert.Len(t, mover.putCalls, 2)
	assert.Len(t, repo.upserts, 2)
}

func TestImportFromUpload_EmptyBodyRejected(t *testing.T) {
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	svc := newTestSnapshotService(t, repo, mover, nil, nil)

	res, err := svc.ImportFromUpload(context.Background(),
		[]SnapshotImportItem{{FileName: "SN001_CFG.xml", Content: nil}}, "")
	require.NoError(t, err)
	require.Len(t, res.Failed, 1)
	assert.Equal(t, ImportErrEmptyBody, res.Failed[0].ErrorCode)
	assert.Empty(t, mover.putCalls)
	assert.Empty(t, repo.upserts)
}

func TestImportFromUpload_PutObjectFails(t *testing.T) {
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{putErr: errors.New("network down")}
	svc := newTestSnapshotService(t, repo, mover, nil, nil)

	res, err := svc.ImportFromUpload(context.Background(),
		[]SnapshotImportItem{{FileName: "SN001_CFG.xml", Content: []byte("ok")}}, "")
	require.NoError(t, err)
	require.Len(t, res.Failed, 1)
	assert.Equal(t, ImportErrPutObject, res.Failed[0].ErrorCode)
	assert.Equal(t, "SN001", res.Failed[0].SerialNumber)
	assert.Empty(t, repo.upserts, "DB upsert must not run when MinIO Put fails")
}

func TestImportFromUpload_UpsertFails_CompensatesRemoveObject(t *testing.T) {
	repo := newFakeSnapshotRepo()
	repo.upsertErr = errors.New("DB exploded")
	mover := &fakeMover{}
	svc := newTestSnapshotService(t, repo, mover, nil, nil)

	res, err := svc.ImportFromUpload(context.Background(),
		[]SnapshotImportItem{{FileName: "SN001_CFG.xml", Content: []byte("ok")}}, "")
	require.NoError(t, err)
	require.Len(t, res.Failed, 1)
	assert.Equal(t, ImportErrUpsert, res.Failed[0].ErrorCode)

	require.Len(t, mover.putCalls, 1)
	require.Len(t, mover.rmCalls, 1, "compensating RemoveObject must run after upsert failure")
	assert.Equal(t, mover.putCalls[0].bucket, mover.rmCalls[0].bucket)
	assert.Equal(t, mover.putCalls[0].object, mover.rmCalls[0].object)
}

func TestImportFromUpload_PutObjectBodyBytesMatch(t *testing.T) {
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	svc := newTestSnapshotService(t, repo, mover, nil, nil)

	payload := bytes.Repeat([]byte{0xAB}, 1024)
	res, err := svc.ImportFromUpload(context.Background(),
		[]SnapshotImportItem{{FileName: "SN001_CFG.nv", Content: payload}}, "")
	require.NoError(t, err)
	assert.Empty(t, res.Failed)

	require.Len(t, mover.putCalls, 1)
	assert.Equal(t, payload, mover.putCalls[0].body)
	assert.EqualValues(t, len(payload), mover.putCalls[0].size)
}

// ─────────────────────────────────────────────────────────────────────────
// Delete tests
// ─────────────────────────────────────────────────────────────────────────

func TestDelete_RemovesMinIOAndDB(t *testing.T) {
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	svc := newTestSnapshotService(t, repo, mover, nil, nil)

	// 先种一条
	require.NoError(t, repo.Upsert(context.Background(), &ConfigSnapshot{
		SerialNumber: "SN001",
		FileName:     "SN001_CFG.xml",
		FileExt:      "xml",
		ObjectBucket: "config-snapshots",
		ObjectPath:   "SN001_CFG.xml",
		Source:       SnapshotSourceManualUpload,
	}))

	require.NoError(t, svc.Delete(context.Background(), "SN001"))
	assert.NotContains(t, repo.rows, "SN001")
	require.Len(t, mover.rmCalls, 1)
	assert.Equal(t, "config-snapshots", mover.rmCalls[0].bucket)
	assert.Equal(t, "SN001_CFG.xml", mover.rmCalls[0].object)
}

func TestDelete_MinIOFailureStillDeletesDB(t *testing.T) {
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{removeErr: errors.New("minio offline")}
	svc := newTestSnapshotService(t, repo, mover, nil, nil)
	require.NoError(t, repo.Upsert(context.Background(), &ConfigSnapshot{
		SerialNumber: "SN001",
		FileName:     "SN001_CFG.xml",
		FileExt:      "xml",
		ObjectBucket: "config-snapshots",
		ObjectPath:   "SN001_CFG.xml",
		Source:       SnapshotSourceManualUpload,
	}))

	require.NoError(t, svc.Delete(context.Background(), "SN001"))
	assert.NotContains(t, repo.rows, "SN001", "DB delete must proceed even when MinIO Remove fails")
}

func TestDelete_NonExistentSN_NoOp(t *testing.T) {
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	svc := newTestSnapshotService(t, repo, mover, nil, nil)

	require.NoError(t, svc.Delete(context.Background(), "ghost"))
	assert.Empty(t, mover.rmCalls, "no MinIO Remove if row doesn't exist")
}

// ─────────────────────────────────────────────────────────────────────────
// BatchGet / read proxies
// ─────────────────────────────────────────────────────────────────────────

func TestBatchGetBySerialNumbers_MissingReportedByOmission(t *testing.T) {
	repo := newFakeSnapshotRepo()
	mover := &fakeMover{}
	svc := newTestSnapshotService(t, repo, mover, nil, nil)
	require.NoError(t, repo.Upsert(context.Background(), &ConfigSnapshot{
		SerialNumber: "SN001",
		FileName:     "SN001_CFG.xml", FileExt: "xml",
		ObjectBucket: "config-snapshots", ObjectPath: "SN001_CFG.xml",
		Source: SnapshotSourceManualUpload,
	}))

	out, err := svc.BatchGetBySerialNumbers(context.Background(),
		[]string{"SN001", "SN_MISSING"})
	require.NoError(t, err)
	assert.Contains(t, out, "SN001")
	assert.NotContains(t, out, "SN_MISSING",
		"BatchGet must omit missing SNs so caller can identify reject set")
}

// ─────────────────────────────────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────────────────────────────────

func ptrStr(s string) *string { return &s }
