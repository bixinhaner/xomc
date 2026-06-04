package export

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
)

// ── stub repo / uploader / source ────────────────────────────────────────────

type stubTaskRepo struct {
	task *Task

	getErr     error
	runErr     error
	succErr    error
	failErr    error
	markedRun  int
	succeeded  *succArgs
	failedWith string
	failedN    int
}

type succArgs struct {
	bucket, filePath string
	fileSize, rowN   int64
}

func (s *stubTaskRepo) Get(_ context.Context, _ uuid.UUID) (*Task, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.task, nil
}
func (s *stubTaskRepo) MarkRunning(_ context.Context, _ uuid.UUID) error {
	s.markedRun++
	return s.runErr
}
func (s *stubTaskRepo) MarkSucceeded(_ context.Context, _ uuid.UUID, bucket, fp string, fs, rn int64) error {
	s.succeeded = &succArgs{bucket, fp, fs, rn}
	return s.succErr
}
func (s *stubTaskRepo) MarkFailed(_ context.Context, _ uuid.UUID, msg string) error {
	s.failedN++
	s.failedWith = msg
	return s.failErr
}

type stubUploader struct {
	uploadErr error
	gotBody   []byte
}

func (u *stubUploader) PutObject(_ context.Context, _, _ string, reader io.Reader, _ int64, _ minio.PutObjectOptions) (minio.UploadInfo, error) {
	b, _ := io.ReadAll(reader)
	u.gotBody = b
	if u.uploadErr != nil {
		return minio.UploadInfo{}, u.uploadErr
	}
	return minio.UploadInfo{Size: int64(len(b))}, nil
}

// sliceSource 把固定批次的 ExportRow 当成 RowSource（测试用）。
type sliceSource struct {
	batches [][]ExportRow
	i       int
	err     error
}

func (s *sliceSource) Next(_ context.Context) ([]ExportRow, bool, error) {
	if s.err != nil {
		return nil, false, s.err
	}
	if s.i >= len(s.batches) {
		return nil, true, nil
	}
	b := s.batches[s.i]
	s.i++
	done := s.i >= len(s.batches)
	return b, done, nil
}

func newTestTask(src SourceType) *Task {
	return &Task{ID: uuid.New(), SourceType: src, Params: []byte(`{}`)}
}

// ── 预 running 守门：payload 坏 / 缺 task_id 直接返 error，不动任务 ─────────────

func TestRunner_JobType(t *testing.T) {
	r := NewRunner(RunnerDeps{Repo: &stubTaskRepo{}})
	assert.Equal(t, JobType, r.JobType())
}

func TestRunner_Run_MissingTaskID(t *testing.T) {
	r := NewRunner(RunnerDeps{Repo: &stubTaskRepo{}})
	_, err := r.Run(context.Background(), &asyncjob.Job{Payload: json.RawMessage(`{}`)})
	require.Error(t, err)
}

func TestRunner_Run_InvalidPayload(t *testing.T) {
	r := NewRunner(RunnerDeps{Repo: &stubTaskRepo{}})
	_, err := r.Run(context.Background(), &asyncjob.Job{Payload: json.RawMessage(`not-json`)})
	require.Error(t, err)
}

func TestRunner_Run_MarkRunningError_Propagates(t *testing.T) {
	repo := &stubTaskRepo{task: newTestTask(SourceDashboard), runErr: errors.New("not pending")}
	r := NewRunner(RunnerDeps{Repo: repo})
	payload, _ := BuildJobPayload(uuid.New())
	_, err := r.Run(context.Background(), &asyncjob.Job{Payload: payload})
	require.Error(t, err)
}

// ── 成功路径：running → 取数 → 上传 → succeeded 回填 ────────────────────────

func TestRunner_Run_Success(t *testing.T) {
	task := newTestTask(SourceDashboard)
	repo := &stubTaskRepo{task: task}
	up := &stubUploader{}
	r := NewRunner(RunnerDeps{Repo: repo, Uploader: up, Bucket: "reports"})
	// 注入 stub 源：2 行数据点。
	r.buildSourceFn = func(_ context.Context, _ *Task) (RowSource, *nameResolver, error) {
		return &sliceSource{batches: [][]ExportRow{{
			{Device: "ABCDEF/SN1", MetricCode: "K001", MetricName: "上行吞吐", Value: 1.5},
			{Device: "ABCDEF/SN1", MetricCode: "K002", MetricName: "下行吞吐", Value: 2.5},
		}}}, nil, nil
	}

	payload, _ := BuildJobPayload(task.ID)
	out, err := r.Run(context.Background(), &asyncjob.Job{Payload: payload})
	require.NoError(t, err)
	assert.Equal(t, 1, repo.markedRun)
	require.NotNil(t, repo.succeeded)
	assert.Equal(t, "reports", repo.succeeded.bucket)
	assert.Equal(t, int64(2), repo.succeeded.rowN) // 2 数据行（不含表头）
	assert.Greater(t, repo.succeeded.fileSize, int64(0))
	assert.Equal(t, 0, repo.failedN)

	// 上传体头三字节为 UTF-8 BOM。
	require.GreaterOrEqual(t, len(up.gotBody), 3)
	assert.Equal(t, []byte{0xEF, 0xBB, 0xBF}, up.gotBody[:3])

	var res map[string]any
	require.NoError(t, json.Unmarshal(out, &res))
	assert.Equal(t, "succeeded", res["status"])
}

// ── 失败路径：取数报错 → MarkFailed + error 落库，job 不重试（返回 nil） ───────

func TestRunner_Run_GenerateError_MarksFailed(t *testing.T) {
	task := newTestTask(SourceDashboard)
	repo := &stubTaskRepo{task: task}
	r := NewRunner(RunnerDeps{Repo: repo, Uploader: &stubUploader{}, Bucket: "reports"})
	r.buildSourceFn = func(_ context.Context, _ *Task) (RowSource, *nameResolver, error) {
		return &sliceSource{err: errors.New("db read failed")}, nil, nil
	}

	payload, _ := BuildJobPayload(task.ID)
	out, err := r.Run(context.Background(), &asyncjob.Job{Payload: payload})
	require.NoError(t, err) // 框架视为已处理，不重试
	assert.Equal(t, 1, repo.failedN)
	assert.Contains(t, repo.failedWith, "db read failed")
	assert.Nil(t, repo.succeeded)

	var res map[string]any
	require.NoError(t, json.Unmarshal(out, &res))
	assert.Equal(t, "failed", res["status"])
}

// 上传失败也走 failed。
func TestRunner_Run_UploadError_MarksFailed(t *testing.T) {
	task := newTestTask(SourceDashboard)
	repo := &stubTaskRepo{task: task}
	up := &stubUploader{uploadErr: errors.New("minio down")}
	r := NewRunner(RunnerDeps{Repo: repo, Uploader: up, Bucket: "reports"})
	r.buildSourceFn = func(_ context.Context, _ *Task) (RowSource, *nameResolver, error) {
		return &sliceSource{batches: [][]ExportRow{{{Device: "d", MetricCode: "K1"}}}}, nil, nil
	}
	payload, _ := BuildJobPayload(task.ID)
	_, err := r.Run(context.Background(), &asyncjob.Job{Payload: payload})
	require.NoError(t, err)
	assert.Equal(t, 1, repo.failedN)
}

// 取数失败 + MarkFailed 也失败 → 返回 error 让框架重试。
func TestRunner_Run_MarkFailedAlsoFails_ReturnsError(t *testing.T) {
	task := newTestTask(SourceDashboard)
	repo := &stubTaskRepo{task: task, failErr: errors.New("db down")}
	r := NewRunner(RunnerDeps{Repo: repo, Uploader: &stubUploader{}, Bucket: "reports"})
	r.buildSourceFn = func(_ context.Context, _ *Task) (RowSource, *nameResolver, error) {
		return &sliceSource{err: errors.New("read error")}, nil, nil
	}
	payload, _ := BuildJobPayload(task.ID)
	_, err := r.Run(context.Background(), &asyncjob.Job{Payload: payload})
	require.Error(t, err)
}

// 无 uploader / 无 bucket → generate 报错 → MarkFailed。
func TestRunner_Run_NoUploader_MarksFailed(t *testing.T) {
	task := newTestTask(SourceDashboard)
	repo := &stubTaskRepo{task: task}
	r := NewRunner(RunnerDeps{Repo: repo, Bucket: "reports"}) // uploader nil
	r.buildSourceFn = func(_ context.Context, _ *Task) (RowSource, *nameResolver, error) {
		return &sliceSource{}, nil, nil
	}
	payload, _ := BuildJobPayload(task.ID)
	_, err := r.Run(context.Background(), &asyncjob.Job{Payload: payload})
	require.NoError(t, err)
	assert.Equal(t, 1, repo.failedN)
}
