package backup

import (
	"context"
	"errors"
	"testing"

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
		assert.Contains(t, string(req.Params), `"file_type":"3"`)
		assert.Contains(t, string(req.Params), `"url":"config_backup/backup/2026/04/29/cfg.xml.gz"`)
	}
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
