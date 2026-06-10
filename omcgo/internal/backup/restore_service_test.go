package backup

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	devicemodel "github.com/omcgo/omcgo/internal/core/model"
	devtask "github.com/omcgo/omcgo/internal/task"
)

// jsonUnmarshal 是 encoding/json.Unmarshal 的别名，让本测试文件读起来更短。
func jsonUnmarshal(data []byte, v interface{}) error { return json.Unmarshal(data, v) }

// --- mocks ---

type mockRestoreRepo struct {
	created []*RestoreTask
	listErr error
}

func (m *mockRestoreRepo) Create(_ context.Context, t *RestoreTask) error {
	t.ID = uuid.New()
	m.created = append(m.created, t)
	return nil
}
func (m *mockRestoreRepo) GetByID(_ context.Context, id uuid.UUID) (*RestoreTask, error) {
	for _, t := range m.created {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, commonerrors.ErrNotFound
}
func (m *mockRestoreRepo) List(_ context.Context, _ RestoreFilter) (*model.ListResponse[RestoreTask], error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	items := make([]RestoreTask, 0, len(m.created))
	for _, t := range m.created {
		items = append(items, *t)
	}
	return model.NewListResponse(items, int64(len(items)), 1, 20), nil
}
func (m *mockRestoreRepo) UpdateErrorMessage(_ context.Context, id uuid.UUID, msg string) error {
	for _, t := range m.created {
		if t.ID == id {
			t.ErrorMessage = &msg
			return nil
		}
	}
	return commonerrors.ErrNotFound
}
func (m *mockRestoreRepo) FindByIDPrefix(_ context.Context, _ string, _ int) ([]*RestoreTask, error) {
	return nil, nil
}
func (m *mockRestoreRepo) MarkComplete(_ context.Context, _ uuid.UUID, _ RestoreStatus, _ int16, _ time.Time, _ string) error {
	return nil
}

// fakeDeviceLookup satisfies the DeviceLookup interface restored to the
// service. Narrow interface = small mock.
type fakeDeviceLookup struct {
	knownSNs map[string]bool
}

func (f *fakeDeviceLookup) GetBySerialNumber(_ context.Context, sn string) (*devicemodel.Device, error) {
	if f.knownSNs[sn] {
		return &devicemodel.Device{SerialNumber: sn}, nil
	}
	return nil, nil
}

type fakeEnqueuer struct {
	requests []*devtask.CreateTaskRequest
	err      error
}

func (f *fakeEnqueuer) CreateTask(_ context.Context, req *devtask.CreateTaskRequest) (*devtask.Task, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.requests = append(f.requests, req)
	return &devtask.Task{ID: uuid.New().String(), DeviceSN: req.DeviceSN, Method: req.Method}, nil
}

type fakeStater struct {
	exists bool
}

func (f *fakeStater) StatObject(_ context.Context, _ string, _ string, _ minio.StatObjectOptions) (minio.ObjectInfo, error) {
	if !f.exists {
		return minio.ObjectInfo{}, minio.ErrorResponse{Code: "NoSuchKey"}
	}
	return minio.ObjectInfo{Size: 1024}, nil
}

// fakeObjReader satisfies RestoreObjectReader: returns fixed bytes (or an error)
// so tests can assert the Download MD5 computed from the streamed content.
type fakeObjReader struct {
	content []byte
	err     error
}

func (f *fakeObjReader) GetObjectStream(_ context.Context, _, _ string) (io.ReadCloser, error) {
	if f.err != nil {
		return nil, f.err
	}
	return io.NopCloser(bytes.NewReader(f.content)), nil
}

// --- tests ---

func newSvc(t *testing.T, knownSNs []string, statExists bool) (*RestoreService, *mockRestoreRepo, *fakeEnqueuer) {
	t.Helper()
	known := map[string]bool{}
	for _, sn := range knownSNs {
		known[sn] = true
	}
	repo := &mockRestoreRepo{}
	enq := &fakeEnqueuer{}
	devRepo := &fakeDeviceLookup{knownSNs: known}
	stater := &fakeStater{exists: statExists}
	svc := NewRestoreService(repo, devRepo, enq, stater, NewRestoreMetrics(nil), zap.NewNop())
	return svc, repo, enq
}

func TestCreate_validRequest_fanOut(t *testing.T) {
	svc, repo, enq := newSvc(t, []string{"SN001", "SN002"}, true)
	rt, err := svc.Create(context.Background(), &CreateRestoreRequest{
		Bucket:          CanonicalRestoreBucket,
		ObjectPath:      "backup/2026/04/29/cfg.xml.gz",
		TargetDeviceSNs: []string{"SN001", "SN002"},
	}, "alice")
	require.NoError(t, err)
	require.NotNil(t, rt)
	assert.Equal(t, RestorePending, rt.Status)
	assert.Equal(t, []string{"SN001", "SN002"}, rt.TargetDeviceSNs)
	require.NotNil(t, rt.CreatedBy)
	assert.Equal(t, "alice", *rt.CreatedBy)
	require.Len(t, repo.created, 1)
	require.Len(t, enq.requests, 2, "two device tasks should be enqueued")
	for _, req := range enq.requests {
		assert.Equal(t, "Download", req.Method)
		// FileType = "10 <OUI> Configuration File"；fake device 没填 OUI →
		// 退化到 fallback OUI 48BF74（Baicells）。
		assert.Contains(t, string(req.Params), `"file_type":"10 48BF74 Configuration File"`)
		assert.Contains(t, string(req.Params), `"url":"config_backup/backup/2026/04/29/cfg.xml.gz"`)
	}
}

// TestCreate_computesMD5FromStream 验证配置文件恢复在下发时读取源文件流现算 MD5
// 并写入 Download params（Download 报文必填）。同一份文件发给多设备 → 同一 MD5。
func TestCreate_computesMD5FromStream(t *testing.T) {
	svc, _, enq := newSvc(t, []string{"SN001", "SN002"}, true)
	content := []byte("restore-config-stream-bytes-甲乙丙")
	sum := md5.Sum(content)
	wantMD5 := hex.EncodeToString(sum[:])
	svc.SetObjectReader(&fakeObjReader{content: content})

	_, err := svc.Create(context.Background(), &CreateRestoreRequest{
		Bucket:          CanonicalRestoreBucket,
		ObjectPath:      "backup/2026/04/29/cfg.xml.gz",
		TargetDeviceSNs: []string{"SN001", "SN002"},
	}, "alice")
	require.NoError(t, err)
	require.Len(t, enq.requests, 2)
	for _, req := range enq.requests {
		assert.Contains(t, string(req.Params), `"md5":"`+wantMD5+`"`,
			"配置恢复 Download 必须携带下发时读流现算的 MD5")
	}
}

// TestCreate_md5ReadError_failsBeforeRowCreated 验证 MD5 计算失败时在建
// restore_task 行之前返回错误，不留孤儿行、不下发缺 MD5 的 Download。
func TestCreate_md5ReadError_failsBeforeRowCreated(t *testing.T) {
	svc, repo, enq := newSvc(t, []string{"SN001"}, true)
	svc.SetObjectReader(&fakeObjReader{err: errors.New("minio unreachable")})

	_, err := svc.Create(context.Background(), &CreateRestoreRequest{
		Bucket:          CanonicalRestoreBucket,
		ObjectPath:      "backup/2026/04/29/cfg.xml.gz",
		TargetDeviceSNs: []string{"SN001"},
	}, "alice")
	require.Error(t, err)
	assert.Empty(t, repo.created, "md5 失败应在建 restore_task 行前返回，不留孤儿行")
	assert.Empty(t, enq.requests, "md5 失败不应下发任何 Download")
}

func TestCreate_unknownDevice_skippedNoted(t *testing.T) {
	svc, repo, enq := newSvc(t, []string{"SN001"}, true)
	_, err := svc.Create(context.Background(), &CreateRestoreRequest{
		Bucket:          CanonicalRestoreBucket,
		ObjectPath:      "backup/cfg.xml.gz",
		TargetDeviceSNs: []string{"SN001", "SN_GHOST"},
	}, "")
	require.NoError(t, err)
	require.Len(t, enq.requests, 1, "ghost device must NOT enqueue")
	require.Len(t, repo.created, 1)
	require.NotNil(t, repo.created[0].ErrorMessage)
	assert.Contains(t, *repo.created[0].ErrorMessage, "SN_GHOST")
}

func TestCreate_invalidPath_traversal(t *testing.T) {
	svc, _, _ := newSvc(t, []string{"SN001"}, true)
	cases := []CreateRestoreRequest{
		{Bucket: CanonicalRestoreBucket, ObjectPath: "../etc/passwd", TargetDeviceSNs: []string{"SN001"}},
		{Bucket: "../firmware", ObjectPath: "cfg.xml", TargetDeviceSNs: []string{"SN001"}},
		{Bucket: "/abs", ObjectPath: "cfg.xml", TargetDeviceSNs: []string{"SN001"}},
	}
	for i, tc := range cases {
		_, err := svc.Create(context.Background(), &tc, "")
		require.Error(t, err, "case %d should reject", i)
		assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput), "case %d wrong error class", i)
	}
}

func TestCreate_disallowedBucket(t *testing.T) {
	svc, _, _ := newSvc(t, []string{"SN001"}, true)
	_, err := svc.Create(context.Background(), &CreateRestoreRequest{
		Bucket:          "firmware",
		ObjectPath:      "img/v2.bin",
		TargetDeviceSNs: []string{"SN001"},
	}, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
	assert.Contains(t, err.Error(), CanonicalRestoreBucket)
}

func TestCreate_objectNotFound(t *testing.T) {
	svc, _, _ := newSvc(t, []string{"SN001"}, false /*stat returns NoSuchKey*/)
	_, err := svc.Create(context.Background(), &CreateRestoreRequest{
		Bucket:          CanonicalRestoreBucket,
		ObjectPath:      "backup/missing.xml.gz",
		TargetDeviceSNs: []string{"SN001"},
	}, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrNotFound))
}

func TestCreate_nilRequest(t *testing.T) {
	svc, _, _ := newSvc(t, []string{"SN001"}, true)
	_, err := svc.Create(context.Background(), nil, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
}

// fakeBackupTaskFinder is a tiny stub for CreateByTaskID tests (T-0079).
type fakeBackupTaskFinder struct {
	task *BackupTask
	err  error
}

func (f *fakeBackupTaskFinder) GetByID(_ context.Context, _ uuid.UUID) (*BackupTask, error) {
	return f.task, f.err
}

func TestCreateByTaskID_resolvesAndDispatches(t *testing.T) {
	svc, _, enq := newSvc(t, []string{"SN999"}, true)
	taskID := uuid.New()
	fp := "config_backup/backup/2026/04/29/backup-abcdef12-SN001.xml.gz"
	bt := &BackupTask{ID: taskID, TargetIDs: []string{"SN001"}, FilePath: &fp}
	svc.SetBackupTaskFinder(&fakeBackupTaskFinder{task: bt})

	res, err := svc.CreateByTaskID(context.Background(), &CreateByTaskIDRequest{
		BackupTaskID:    taskID,
		TargetDeviceSNs: []string{"SN999"},
	}, "alice")
	require.NoError(t, err)
	require.NotNil(t, res.Task)
	require.Nil(t, res.Warning, "single-device source must have no warning")
	require.Len(t, enq.requests, 1)
	assert.Contains(t, string(enq.requests[0].Params), `"url":"config_backup/backup/2026/04/29/backup-abcdef12-SN001.xml.gz"`)
}

func TestCreateByTaskID_multiDeviceWarning(t *testing.T) {
	svc, _, _ := newSvc(t, []string{"SN999"}, true)
	taskID := uuid.New()
	fp := "config_backup/backup/x.xml"
	bt := &BackupTask{ID: taskID, TargetIDs: []string{"SN001", "SN002", "SN003"}, FilePath: &fp}
	svc.SetBackupTaskFinder(&fakeBackupTaskFinder{task: bt})

	res, err := svc.CreateByTaskID(context.Background(), &CreateByTaskIDRequest{
		BackupTaskID:    taskID,
		TargetDeviceSNs: []string{"SN999"},
	}, "")
	require.NoError(t, err)
	require.NotNil(t, res.Warning)
	assert.Contains(t, *res.Warning, "multi-device")
	assert.Contains(t, *res.Warning, "first-write-wins")
}

func TestCreateByTaskID_filePathNullRejected(t *testing.T) {
	svc, _, _ := newSvc(t, []string{"SN999"}, true)
	taskID := uuid.New()
	bt := &BackupTask{ID: taskID, TargetIDs: []string{"SN001"}, FilePath: nil}
	svc.SetBackupTaskFinder(&fakeBackupTaskFinder{task: bt})

	_, err := svc.CreateByTaskID(context.Background(), &CreateByTaskIDRequest{
		BackupTaskID:    taskID,
		TargetDeviceSNs: []string{"SN999"},
	}, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrNotFound), "null file_path → ErrNotFound 404")
}

func TestCreateByTaskID_finderNotConfigured(t *testing.T) {
	svc, _, _ := newSvc(t, []string{"SN999"}, true)
	// Do NOT call SetBackupTaskFinder — simulate the case where DI didn't wire it.

	_, err := svc.CreateByTaskID(context.Background(), &CreateByTaskIDRequest{
		BackupTaskID:    uuid.New(),
		TargetDeviceSNs: []string{"SN999"},
	}, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
}

// ─────────────────────────────────────────────────────────────────────────
// B5: CreateBySnapshot
// ─────────────────────────────────────────────────────────────────────────

// fakeSnapshotLookup satisfies SnapshotLookup with an in-memory map.
type fakeSnapshotLookup struct {
	rows map[string]*ConfigSnapshot
	err  error
}

func (f *fakeSnapshotLookup) BatchGetBySerialNumbers(_ context.Context, sns []string) (map[string]*ConfigSnapshot, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := map[string]*ConfigSnapshot{}
	for _, sn := range sns {
		if v, ok := f.rows[sn]; ok {
			out[sn] = v
		}
	}
	return out, nil
}

func makeSnap(sn string) *ConfigSnapshot {
	return &ConfigSnapshot{
		SerialNumber: sn,
		FileName:     sn + "_CFG.xml",
		FileExt:      "xml",
		ObjectBucket: "config-snapshots",
		ObjectPath:   sn + "_CFG.xml",
		Source:       SnapshotSourceManualUpload,
	}
}

func TestCreateBySnapshot_AllPresent_FansOutPerDevice(t *testing.T) {
	svc, repo, enq := newSvc(t, []string{"SN001", "SN002"}, true)
	svc.SetSnapshotLookup(&fakeSnapshotLookup{rows: map[string]*ConfigSnapshot{
		"SN001": makeSnap("SN001"),
		"SN002": makeSnap("SN002"),
	}})

	res, err := svc.CreateBySnapshot(context.Background(),
		&CreateBySnapshotRequest{TargetDeviceSNs: []string{"SN001", "SN002"}}, "alice")
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Task)
	assert.Empty(t, res.Missing)

	// One restore_tasks row, placeholder source_object_path.
	require.Len(t, repo.created, 1)
	assert.Equal(t, "config-snapshots", repo.created[0].SourceBucket)
	assert.Equal(t, SnapshotRestoreSourcePlaceholder, repo.created[0].SourceObjectPath)
	assert.ElementsMatch(t, []string{"SN001", "SN002"}, repo.created[0].TargetDeviceSNs)

	// Two device_tasks, each with its own URL pointing at the device's snapshot.
	require.Len(t, enq.requests, 2)
	urls := []string{}
	for _, r := range enq.requests {
		var params map[string]interface{}
		require.NoError(t, jsonUnmarshal(r.Params, &params))
		urls = append(urls, params["url"].(string))
	}
	assert.ElementsMatch(t,
		[]string{"config-snapshots/SN001_CFG.xml", "config-snapshots/SN002_CFG.xml"},
		urls)
}

func TestCreateBySnapshot_PartialMissing_IntegrallyRejects(t *testing.T) {
	svc, repo, enq := newSvc(t, []string{"SN001", "SN002", "SN003"}, true)
	svc.SetSnapshotLookup(&fakeSnapshotLookup{rows: map[string]*ConfigSnapshot{
		"SN001": makeSnap("SN001"),
		// SN002, SN003 missing
	}})

	res, err := svc.CreateBySnapshot(context.Background(),
		&CreateBySnapshotRequest{TargetDeviceSNs: []string{"SN001", "SN002", "SN003"}}, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrNotFound))
	require.NotNil(t, res)
	assert.ElementsMatch(t, []string{"SN002", "SN003"}, res.Missing)

	// No persistence side-effects (no restore_task row, no device_tasks).
	assert.Empty(t, repo.created, "integral rejection must not create restore_tasks")
	assert.Empty(t, enq.requests, "integral rejection must not enqueue device_tasks")
}

func TestCreateBySnapshot_NotConfigured(t *testing.T) {
	svc, _, _ := newSvc(t, []string{"SN001"}, true)
	// SetSnapshotLookup NOT called.
	_, err := svc.CreateBySnapshot(context.Background(),
		&CreateBySnapshotRequest{TargetDeviceSNs: []string{"SN001"}}, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
}

func TestCreateBySnapshot_EmptyTargets(t *testing.T) {
	svc, _, _ := newSvc(t, nil, true)
	svc.SetSnapshotLookup(&fakeSnapshotLookup{rows: map[string]*ConfigSnapshot{}})
	_, err := svc.CreateBySnapshot(context.Background(),
		&CreateBySnapshotRequest{TargetDeviceSNs: []string{}}, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
}

func TestSplitBucketAndPath(t *testing.T) {
	bucket, path, err := splitBucketAndPath("config_backup/backup/2026/04/29/x.xml.gz")
	require.NoError(t, err)
	assert.Equal(t, "config_backup", bucket)
	assert.Equal(t, "backup/2026/04/29/x.xml.gz", path)

	_, _, err = splitBucketAndPath("nopath")
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))

	_, _, err = splitBucketAndPath("/leadingslash")
	require.Error(t, err)
}

func TestValidateRestorePath_table(t *testing.T) {
	good := []string{
		"backup/2026/04/29/cfg.xml",
		"backup/cfg.xml.gz",
	}
	for _, p := range good {
		assert.NoError(t, validateRestorePath(CanonicalRestoreBucket, p), "should accept %q", p)
	}
	bad := []string{
		"",
		"../escape",
		"/absolute/path",
		"backup/../etc/passwd",
	}
	for _, p := range bad {
		assert.Error(t, validateRestorePath(CanonicalRestoreBucket, p), "should reject %q", p)
	}
}
