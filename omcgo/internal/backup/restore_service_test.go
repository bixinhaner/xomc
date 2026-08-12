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
	"github.com/minio/minio-go/v7/pkg/s3utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/acs/transfercfg"
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
func (m *mockRestoreRepo) MarkDownloaded(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (m *mockRestoreRepo) MarkVerified(_ context.Context, _ uuid.UUID, _ RestoreStatus, _ int16, _ string, _ RestoreVerificationMethod, _ time.Time, _ string) error {
	return nil
}

// fakeDeviceLookup satisfies the DeviceLookup interface restored to the
// service. Narrow interface = small mock.
type fakeDeviceLookup struct {
	knownSNs map[string]bool
	devices  map[string]*devicemodel.Device
}

func (f *fakeDeviceLookup) GetBySerialNumber(_ context.Context, sn string) (*devicemodel.Device, error) {
	if f.devices != nil {
		if dev := f.devices[sn]; dev != nil {
			return dev, nil
		}
	}
	if f.knownSNs[sn] {
		return &devicemodel.Device{ID: uuid.New(), SerialNumber: sn}, nil
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
	// statErr（可选）覆盖 !exists 时返回的默认 NoSuchKey 错误，用于模拟其它
	// MinIO 错误码（如 InvalidBucketName）。仅在 exists==false 时生效。
	statErr error
	buckets []string
}

func (f *fakeStater) StatObject(_ context.Context, bucket string, _ string, _ minio.StatObjectOptions) (minio.ObjectInfo, error) {
	f.buckets = append(f.buckets, bucket)
	if !f.exists {
		if f.statErr != nil {
			return minio.ObjectInfo{}, f.statErr
		}
		return minio.ObjectInfo{}, minio.ErrorResponse{Code: "NoSuchKey"}
	}
	return minio.ObjectInfo{Size: 1024}, nil
}

// fakeObjReader satisfies RestoreObjectReader: returns fixed bytes (or an error)
// so tests can assert the Download MD5 computed from the streamed content.
//
// readErr（可选）模拟 minio-go 的真实行为：GetObject 不立即报错，缺失对象的错误
// 在首次 Read 时才暴露 —— 此时 GetObjectStream 成功返回 reader，但 io.Copy 失败。
type fakeObjReader struct {
	content []byte
	err     error
	readErr error
	buckets []string
}

func (f *fakeObjReader) GetObjectStream(_ context.Context, bucket, _ string) (io.ReadCloser, error) {
	f.buckets = append(f.buckets, bucket)
	if f.err != nil {
		return nil, f.err
	}
	if f.readErr != nil {
		return io.NopCloser(errReader{err: f.readErr}), nil
	}
	return io.NopCloser(bytes.NewReader(f.content)), nil
}

// errReader 总在 Read 时返回固定错误，模拟 minio-go 对象不存在时 Read 才暴露的错误。
type errReader struct{ err error }

func (r errReader) Read(_ []byte) (int, error) { return 0, r.err }

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
	setRestoreTransferPolicy(t, svc, transfercfg.ProtocolPolicyForceHTTP, transfercfg.HTTPSCapabilityNotRead)
	return svc, repo, enq
}

type restoreCapabilityReader struct {
	status transfercfg.HTTPSCapabilityStatus
	calls  int
}

func (r *restoreCapabilityReader) ReadHTTPSCapability(context.Context, uuid.UUID) transfercfg.HTTPSCapabilityStatus {
	r.calls++
	return r.status
}

func setRestoreTransferPolicy(t *testing.T, svc *RestoreService, policyValue string, capability transfercfg.HTTPSCapabilityStatus) *restoreCapabilityReader {
	t.Helper()
	policy := transfercfg.NewPolicy(transfercfg.Snapshot{
		ProtocolPolicy: policyValue,
		Download: transfercfg.DownloadSettings{
			BaseURL:      "http://download.example.com:8080/proxy",
			HTTPSBaseURL: "https://download.example.com:9443/secure-proxy",
			Path:         "/smallcell/FileDownloadService",
		},
	}, nil)
	reader := &restoreCapabilityReader{status: capability}
	svc.SetTransferProvider(policy)
	svc.SetDownloadAddressResolver(transfercfg.NewAddressResolver(policy, reader))
	return reader
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
		assert.Contains(t, string(req.Params), `"url":"http://download.example.com:8080/proxy/smallcell/FileDownloadService/config-backup/backup/2026/04/29/cfg.xml.gz"`)
	}
}

func restoreQueuedURL(t *testing.T, req *devtask.CreateTaskRequest) string {
	t.Helper()
	var params map[string]interface{}
	require.NoError(t, jsonUnmarshal(req.Params, &params))
	got, ok := params["url"].(string)
	require.True(t, ok)
	return got
}

func TestCreate_UsesHTTPSDownloadResolverAndEncodesObjectPath(t *testing.T) {
	svc, _, enq := newSvc(t, []string{"SN HTTPS+RESTORE"}, true)
	reader := setRestoreTransferPolicy(t, svc, transfercfg.ProtocolPolicyPreferHTTPS, transfercfg.HTTPSCapabilityEnabled)
	svc.SetObjectReader(&fakeObjReader{content: []byte("restore config")})

	_, err := svc.Create(context.Background(), &CreateRestoreRequest{
		Bucket:          CanonicalRestoreBucket,
		ObjectPath:      "backup/2026/08/13/配置 A+B.xml",
		TargetDeviceSNs: []string{"SN HTTPS+RESTORE"},
	}, "alice")

	require.NoError(t, err)
	require.Len(t, enq.requests, 1)
	assert.Equal(t, 1, reader.calls)
	assert.Equal(t,
		"https://download.example.com:9443/secure-proxy/smallcell/FileDownloadService/config-backup/backup/2026/08/13/%E9%85%8D%E7%BD%AE%20A+B.xml",
		restoreQueuedURL(t, enq.requests[0]),
	)
	assert.Contains(t, string(enq.requests[0].Params), `"target_file_name":"配置 A+B.xml"`)
}

type failingTransferResolver struct {
	err error
}

func (r failingTransferResolver) Resolve(context.Context, uuid.UUID, transfercfg.TransferDirection) (transfercfg.AddressDecision, error) {
	return transfercfg.AddressDecision{}, r.err
}

func TestCreate_DownloadResolverErrorFailsClosedWithoutEnqueue(t *testing.T) {
	svc, repo, enq := newSvc(t, []string{"SN001"}, true)
	svc.SetDownloadAddressResolver(failingTransferResolver{err: errors.New("resolver down")})

	rt, err := svc.Create(context.Background(), &CreateRestoreRequest{
		Bucket:          CanonicalRestoreBucket,
		ObjectPath:      "backup/cfg.xml",
		TargetDeviceSNs: []string{"SN001"},
	}, "alice")

	require.NoError(t, err)
	require.NotNil(t, rt)
	require.Len(t, repo.created, 1)
	assert.Empty(t, enq.requests, "resolver 失败时不能退回裸 bucket/object 下发")
	require.NotNil(t, repo.created[0].ErrorMessage)
	assert.Contains(t, *repo.created[0].ErrorMessage, "SN001")
}

func TestCreate_LegacyLogicalBucketUsesValidPhysicalBucketForMinIO(t *testing.T) {
	svc, repo, _ := newSvc(t, []string{"SN001"}, true)
	reader := &fakeObjReader{content: []byte("config")}
	svc.SetObjectReader(reader)
	stater := svc.stater.(*fakeStater)

	_, err := svc.Create(context.Background(), &CreateRestoreRequest{
		Bucket:          "config_backup",
		ObjectPath:      "backup/cfg.xml",
		TargetDeviceSNs: []string{"SN001"},
	}, "alice")
	require.NoError(t, err)

	require.Equal(t, []string{"config-backup"}, stater.buckets)
	require.Equal(t, []string{"config-backup"}, reader.buckets)
	for _, bucket := range append(append([]string{}, stater.buckets...), reader.buckets...) {
		require.NoError(t, s3utils.CheckValidBucketNameStrict(bucket),
			"every bucket sent to MinIO must satisfy strict S3 naming")
	}
	require.Len(t, repo.created, 1)
	assert.Equal(t, "config-backup", repo.created[0].SourceBucket)
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

// InvalidBucketName 仍按 not-found 类错误收敛，避免 SDK 细节泄露。正常 restore
// 路径已在进入 MinIO 前把 legacy config_backup 归一为 config-backup。
func TestCreate_statInvalidBucketName(t *testing.T) {
	svc, repo, enq := newSvc(t, []string{"SN001"}, false)
	// 覆盖默认 NoSuchKey，模拟桶名非法时 minio-go 的客户端校验拒绝。
	svc.stater = &fakeStater{exists: false, statErr: minio.ErrorResponse{
		Code:    "InvalidBucketName",
		Message: "The specified bucket is not valid.",
	}}

	_, err := svc.Create(context.Background(), &CreateRestoreRequest{
		Bucket:          CanonicalRestoreBucket,
		ObjectPath:      "backup/missing.xml.gz",
		TargetDeviceSNs: []string{"SN001"},
	}, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrNotFound),
		"InvalidBucketName 必须翻译为 ErrNotFound → 404，而非 default 500")
	assert.NotContains(t, err.Error(), "bucket is not valid",
		"不外泄裸 SDK 错误消息")
	assert.Empty(t, repo.created, "NotFound 应在建 restore_task 行之前返回，不留孤儿行")
	assert.Empty(t, enq.requests, "源桶非法时不应下发任何 Download")
}

// newSvcNilStater 构造 stater 未注入的 service（复现 #125-backup 触发条件：
// stater 为 nil → 跳过 StatObject 存在性预检 → 缺失对象落到 computeSourceMD5）。
func newSvcNilStater(t *testing.T, knownSNs []string) (*RestoreService, *mockRestoreRepo, *fakeEnqueuer) {
	t.Helper()
	known := map[string]bool{}
	for _, sn := range knownSNs {
		known[sn] = true
	}
	repo := &mockRestoreRepo{}
	enq := &fakeEnqueuer{}
	devRepo := &fakeDeviceLookup{knownSNs: known}
	// stater 显式传 nil → 预检被跳过，由 computeSourceMD5 的 GetObject 兜底翻译。
	svc := NewRestoreService(repo, devRepo, enq, nil, NewRestoreMetrics(nil), zap.NewNop())
	setRestoreTransferPolicy(t, svc, transfercfg.ProtocolPolicyForceHTTP, transfercfg.HTTPSCapabilityNotRead)
	return svc, repo, enq
}

// TestCreate_staterNil_objectNotFound_onStreamOpen 复现 #125-backup：stater 未注入时
// 跳过存在性预检，缺失对象在 GetObjectStream 阶段暴露 MinIO NoSuchKey →
// computeSourceMD5 兜底翻译为 ErrNotFound（handler 映射 404，而非 default 500）。
func TestCreate_staterNil_objectNotFound_onStreamOpen(t *testing.T) {
	svc, repo, enq := newSvcNilStater(t, []string{"SN001"})
	svc.SetObjectReader(&fakeObjReader{err: minio.ErrorResponse{Code: "NoSuchKey"}})

	_, err := svc.Create(context.Background(), &CreateRestoreRequest{
		Bucket:          CanonicalRestoreBucket,
		ObjectPath:      "backup/missing.xml.gz",
		TargetDeviceSNs: []string{"SN001"},
	}, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrNotFound),
		"stater 未注入时缺失源对象必须翻译为 ErrNotFound → 404")
	assert.Empty(t, repo.created, "NotFound 应在建 restore_task 行之前返回，不留孤儿行")
	assert.Empty(t, enq.requests, "缺失源对象不应下发任何 Download")
}

// TestCreate_staterNil_objectNotFound_onRead 复现 minio-go 真实行为：GetObject
// 不立即报错，缺失对象的 NoSuchKey 在首次 Read 时才暴露 → io.Copy 失败 →
// computeSourceMD5 兜底翻译为 ErrNotFound。
func TestCreate_staterNil_objectNotFound_onRead(t *testing.T) {
	svc, repo, _ := newSvcNilStater(t, []string{"SN001"})
	svc.SetObjectReader(&fakeObjReader{readErr: minio.ErrorResponse{Code: "NoSuchKey"}})

	_, err := svc.Create(context.Background(), &CreateRestoreRequest{
		Bucket:          CanonicalRestoreBucket,
		ObjectPath:      "backup/missing.xml.gz",
		TargetDeviceSNs: []string{"SN001"},
	}, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrNotFound),
		"Read 阶段暴露的 NoSuchKey 也必须翻译为 ErrNotFound → 404")
	assert.Empty(t, repo.created)
}

// TestCreate_staterNil_noSuchBucket_translated 验证 NoSuchBucket 同样翻译为 404。
func TestCreate_staterNil_noSuchBucket_translated(t *testing.T) {
	svc, _, _ := newSvcNilStater(t, []string{"SN001"})
	svc.SetObjectReader(&fakeObjReader{err: minio.ErrorResponse{Code: "NoSuchBucket"}})

	_, err := svc.Create(context.Background(), &CreateRestoreRequest{
		Bucket:          CanonicalRestoreBucket,
		ObjectPath:      "backup/missing.xml.gz",
		TargetDeviceSNs: []string{"SN001"},
	}, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrNotFound))
}

// TestCreate_staterNil_genericMinIOError_notTranslated 验证非 NotFound 的 MinIO
// 错误（如 AccessDenied）不被误翻译为 404 —— 仍走 default 500 路径（internal error）。
func TestCreate_staterNil_genericMinIOError_notTranslated(t *testing.T) {
	svc, _, _ := newSvcNilStater(t, []string{"SN001"})
	svc.SetObjectReader(&fakeObjReader{err: minio.ErrorResponse{Code: "AccessDenied"}})

	_, err := svc.Create(context.Background(), &CreateRestoreRequest{
		Bucket:          CanonicalRestoreBucket,
		ObjectPath:      "backup/forbidden.xml.gz",
		TargetDeviceSNs: []string{"SN001"},
	}, "")
	require.Error(t, err)
	assert.False(t, errors.Is(err, commonerrors.ErrNotFound),
		"非 NotFound 的 MinIO 错误不得被误翻译为 404")
}

// TestTranslateMinIONotFound_table 直接覆盖错误翻译函数：NoSuchKey/NoSuchBucket
// 及客户端命名校验拒绝 InvalidBucketName/XMinioInvalidObjectName → ErrNotFound 且
// 不外泄裸 SDK 细节；其它错误 → nil（调用方按内部错误处理）。
// InvalidBucketName 用例覆盖外部/异常调用传入非法物理桶名时的错误翻译。
func TestTranslateMinIONotFound_table(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		wantNF  bool
		wantNil bool
	}{
		{"nil error", nil, false, true},
		{"NoSuchKey", minio.ErrorResponse{Code: "NoSuchKey"}, true, false},
		{"NoSuchBucket", minio.ErrorResponse{Code: "NoSuchBucket"}, true, false},
		{"InvalidBucketName", minio.ErrorResponse{Code: "InvalidBucketName", Message: "The specified bucket is not valid."}, true, false},
		{"XMinioInvalidObjectName", minio.ErrorResponse{Code: "XMinioInvalidObjectName"}, true, false},
		{"AccessDenied", minio.ErrorResponse{Code: "AccessDenied"}, false, true},
		{"plain error", errors.New("connection refused"), false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := translateMinIONotFound("config-backup", "backup/x.xml", tc.err)
			if tc.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tc.wantNF, errors.Is(got, commonerrors.ErrNotFound))
			// 不外泄裸 SDK 错误码 / 消息：只保留 bucket/object 路径上下文。
			assert.NotContains(t, got.Error(), "NoSuchKey")
			assert.NotContains(t, got.Error(), "NoSuchBucket")
			assert.NotContains(t, got.Error(), "InvalidBucketName")
			assert.NotContains(t, got.Error(), "bucket is not valid")
			assert.Contains(t, got.Error(), "config-backup/backup/x.xml")
		})
	}
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
	assert.Contains(t, string(enq.requests[0].Params), `"url":"http://download.example.com:8080/proxy/smallcell/FileDownloadService/config-backup/backup/2026/04/29/backup-abcdef12-SN001.xml.gz"`)
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
		[]string{
			"http://download.example.com:8080/proxy/smallcell/FileDownloadService/config-snapshots/SN001_CFG.xml",
			"http://download.example.com:8080/proxy/smallcell/FileDownloadService/config-snapshots/SN002_CFG.xml",
		},
		urls)
}

func TestCreateBySnapshot_ForceHTTPDoesNotReadCapabilityAndEncodesPath(t *testing.T) {
	svc, _, enq := newSvc(t, []string{"SN001"}, true)
	reader := setRestoreTransferPolicy(t, svc, transfercfg.ProtocolPolicyForceHTTP, transfercfg.HTTPSCapabilityEnabled)
	svc.SetSnapshotLookup(&fakeSnapshotLookup{rows: map[string]*ConfigSnapshot{
		"SN001": {
			SerialNumber: "SN001",
			FileName:     "配置 A+B.xml",
			FileExt:      "xml",
			ObjectBucket: "config-snapshots",
			ObjectPath:   "SN001/配置 A+B.xml",
			Source:       SnapshotSourceManualUpload,
		},
	}})

	res, err := svc.CreateBySnapshot(context.Background(),
		&CreateBySnapshotRequest{TargetDeviceSNs: []string{"SN001"}}, "alice")

	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, enq.requests, 1)
	assert.Zero(t, reader.calls)
	assert.Equal(t,
		"http://download.example.com:8080/proxy/smallcell/FileDownloadService/config-snapshots/SN001/%E9%85%8D%E7%BD%AE%20A+B.xml",
		restoreQueuedURL(t, enq.requests[0]),
	)
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
	assert.Equal(t, "config-backup", bucket)
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
