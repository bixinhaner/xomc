package backup

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
)

// statefulRestoreRepo 是一个记录状态转移的 restore repo 桩，用于断言
// downloaded / completed / failed 三态机的实际落地（mockRestoreRepo 是无副作用桩）。
type statefulRestoreRepo struct {
	rows map[uuid.UUID]*RestoreTask
	// 转移轨迹，便于断言 "先 downloaded 再 completed" 的顺序。
	transitions []RestoreStatus
}

func newStatefulRepo() *statefulRestoreRepo {
	return &statefulRestoreRepo{rows: map[uuid.UUID]*RestoreTask{}}
}

func (r *statefulRestoreRepo) put(t *RestoreTask) *RestoreTask {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	cp := *t
	r.rows[t.ID] = &cp
	return &cp
}

func (r *statefulRestoreRepo) Create(_ context.Context, t *RestoreTask) error {
	r.put(t)
	return nil
}
func (r *statefulRestoreRepo) GetByID(_ context.Context, id uuid.UUID) (*RestoreTask, error) {
	if t, ok := r.rows[id]; ok {
		cp := *t
		return &cp, nil
	}
	return nil, errors.New("not found")
}
func (r *statefulRestoreRepo) List(context.Context, RestoreFilter) (*model.ListResponse[RestoreTask], error) {
	return nil, errors.New("unused")
}
func (r *statefulRestoreRepo) UpdateErrorMessage(_ context.Context, id uuid.UUID, msg string) error {
	if t, ok := r.rows[id]; ok {
		t.ErrorMessage = &msg
	}
	return nil
}
func (r *statefulRestoreRepo) FindByIDPrefix(_ context.Context, prefix string, limit int) ([]*RestoreTask, error) {
	out := make([]*RestoreTask, 0, limit)
	for id, t := range r.rows {
		if strings.HasPrefix(strings.ReplaceAll(id.String(), "-", ""), prefix) {
			cp := *t
			out = append(out, &cp)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}
func (r *statefulRestoreRepo) MarkComplete(_ context.Context, id uuid.UUID, status RestoreStatus, result int16, completedAt time.Time, errMsg string) error {
	t, ok := r.rows[id]
	if !ok {
		return errors.New("not found")
	}
	t.Status = status
	res := result
	t.TaskResult = &res
	t.CompletedAt = &completedAt
	if status == RestoreFailed && errMsg != "" {
		t.ErrorMessage = &errMsg
	}
	r.transitions = append(r.transitions, status)
	return nil
}
func (r *statefulRestoreRepo) MarkDownloaded(_ context.Context, id uuid.UUID, downloadedAt time.Time) error {
	t, ok := r.rows[id]
	if !ok {
		return errors.New("not found")
	}
	// 只从 pending/running 推进（与 PG 实现的 WHERE 同语义）。
	if t.Status == RestorePending || t.Status == RestoreRunning {
		t.Status = RestoreDownloaded
		t.DownloadedAt = &downloadedAt
		r.transitions = append(r.transitions, RestoreDownloaded)
	}
	return nil
}
func (r *statefulRestoreRepo) MarkVerified(_ context.Context, id uuid.UUID, status RestoreStatus, result int16, verifiedHash string, method RestoreVerificationMethod, verifiedAt time.Time, errMsg string) error {
	t, ok := r.rows[id]
	if !ok {
		return errors.New("not found")
	}
	t.Status = status
	res := result
	t.TaskResult = &res
	t.VerifiedAt = &verifiedAt
	t.CompletedAt = &verifiedAt
	m := method
	t.VerificationMethod = &m
	if verifiedHash != "" {
		t.VerifiedHash = &verifiedHash
	}
	if status == RestoreFailed && errMsg != "" {
		t.ErrorMessage = &errMsg
	}
	r.transitions = append(r.transitions, status)
	return nil
}

// fakeVerifier 按设备返回预设结果/错误。
type fakeVerifier struct {
	byDevice map[string]RestoreVerifyResult
	err      error
}

func (f *fakeVerifier) Verify(_ context.Context, sn, _ string, _ RestoreHashAlgo) (RestoreVerifyResult, error) {
	if f.err != nil {
		return RestoreVerifyResult{}, f.err
	}
	if res, ok := f.byDevice[sn]; ok {
		return res, nil
	}
	return RestoreVerifyResult{Match: false, Detail: "no result for device", Method: RestoreVerifyGPVReadback}, nil
}

func md5Ptr(s string) *string { return &s }

func newDownloadedTask(repo *statefulRestoreRepo, sns []string, expected string) *RestoreTask {
	algo := RestoreHashMD5
	t := &RestoreTask{
		SourceBucket:    "config-snapshots",
		TargetDeviceSNs: sns,
		Status:          RestoreRunning,
		ExpectedHash:    md5Ptr(expected),
		HashAlgo:        &algo,
	}
	return repo.put(t)
}

// ── Task 1 + Task 2: 主动校验 + 状态机 ──

// TestOrchestrator_HashMatch_Completed: 回读指纹与期望一致 → downloaded 再 completed。
func TestOrchestrator_HashMatch_Completed(t *testing.T) {
	repo := newStatefulRepo()
	task := newDownloadedTask(repo, []string{"SN001"}, "abc123")
	verifier := &fakeVerifier{byDevice: map[string]RestoreVerifyResult{
		"SN001": {Match: true, ObservedHash: "abc123", Method: RestoreVerifyDeviceChecksum},
	}}
	orch := NewRestoreVerificationOrchestrator(repo, verifier, DefaultVerificationConfig(), nil, zap.NewNop())

	require.NoError(t, orch.HandleDownloaded(context.Background(), task, time.Now()))

	got, _ := repo.GetByID(context.Background(), task.ID)
	assert.Equal(t, RestoreCompleted, got.Status)
	require.NotNil(t, got.TaskResult)
	assert.Equal(t, int16(TaskResultSuccess), *got.TaskResult)
	require.NotNil(t, got.VerifiedHash)
	assert.Equal(t, "abc123", *got.VerifiedHash)
	require.NotNil(t, got.VerificationMethod)
	assert.Equal(t, RestoreVerifyDeviceChecksum, *got.VerificationMethod)
	// 必须先经过 downloaded 中间态。
	assert.Equal(t, []RestoreStatus{RestoreDownloaded, RestoreCompleted}, repo.transitions)
}

// TestOrchestrator_HashMismatch_Failed: 回读指纹不一致 → downloaded 再 failed + error_message。
func TestOrchestrator_HashMismatch_Failed(t *testing.T) {
	repo := newStatefulRepo()
	task := newDownloadedTask(repo, []string{"SN001"}, "expected-aaa")
	verifier := &fakeVerifier{byDevice: map[string]RestoreVerifyResult{
		"SN001": {Match: false, ObservedHash: "actual-bbb", Method: RestoreVerifyGPVReadback, Detail: "hash differs"},
	}}
	orch := NewRestoreVerificationOrchestrator(repo, verifier, DefaultVerificationConfig(), nil, zap.NewNop())

	require.NoError(t, orch.HandleDownloaded(context.Background(), task, time.Now()))

	got, _ := repo.GetByID(context.Background(), task.ID)
	assert.Equal(t, RestoreFailed, got.Status)
	require.NotNil(t, got.TaskResult)
	assert.Equal(t, int16(TaskResultFailed), *got.TaskResult)
	require.NotNil(t, got.ErrorMessage)
	assert.Contains(t, *got.ErrorMessage, "SN001")
	assert.Contains(t, *got.ErrorMessage, "hash differs")
	assert.Equal(t, []RestoreStatus{RestoreDownloaded, RestoreFailed}, repo.transitions)
}

// TestOrchestrator_MultiDevice_OneMismatch_Failed: 多设备任一不一致 → 整体 failed。
func TestOrchestrator_MultiDevice_OneMismatch_Failed(t *testing.T) {
	repo := newStatefulRepo()
	task := newDownloadedTask(repo, []string{"SN001", "SN002"}, "exp")
	verifier := &fakeVerifier{byDevice: map[string]RestoreVerifyResult{
		"SN001": {Match: true, ObservedHash: "exp", Method: RestoreVerifyGPVReadback},
		"SN002": {Match: false, ObservedHash: "other", Method: RestoreVerifyGPVReadback},
	}}
	orch := NewRestoreVerificationOrchestrator(repo, verifier, DefaultVerificationConfig(), nil, zap.NewNop())

	require.NoError(t, orch.HandleDownloaded(context.Background(), task, time.Now()))
	got, _ := repo.GetByID(context.Background(), task.ID)
	assert.Equal(t, RestoreFailed, got.Status)
}

// TestOrchestrator_NoVerifier_StaysDownloaded: 默认策略下无 verifier → 停 downloaded（不谎报 completed）。
func TestOrchestrator_NoVerifier_StaysDownloaded(t *testing.T) {
	repo := newStatefulRepo()
	task := newDownloadedTask(repo, []string{"SN001"}, "exp")
	orch := NewRestoreVerificationOrchestrator(repo, nil /*no verifier*/, DefaultVerificationConfig(), nil, zap.NewNop())

	require.NoError(t, orch.HandleDownloaded(context.Background(), task, time.Now()))
	got, _ := repo.GetByID(context.Background(), task.ID)
	assert.Equal(t, RestoreDownloaded, got.Status, "无 verifier 默认停在 downloaded")
	require.NotNil(t, got.DownloadedAt)
	assert.Equal(t, []RestoreStatus{RestoreDownloaded}, repo.transitions)
}

// TestOrchestrator_NoVerifier_FallbackComplete: 显式开启兼容策略 → 退化为旧 completed 行为。
func TestOrchestrator_NoVerifier_FallbackComplete(t *testing.T) {
	repo := newStatefulRepo()
	task := newDownloadedTask(repo, []string{"SN001"}, "exp")
	cfg := VerificationConfig{OnNoVerifier: NoVerifierFallbackComplete}
	orch := NewRestoreVerificationOrchestrator(repo, nil, cfg, nil, zap.NewNop())

	require.NoError(t, orch.HandleDownloaded(context.Background(), task, time.Now()))
	got, _ := repo.GetByID(context.Background(), task.ID)
	assert.Equal(t, RestoreCompleted, got.Status)
	require.NotNil(t, got.VerificationMethod)
	assert.Equal(t, RestoreVerifyNone, *got.VerificationMethod)
}

// TestOrchestrator_NoExpectedHash_StaysDownloaded: 有 verifier 但没记期望指纹 → 无从比对 → 停 downloaded。
func TestOrchestrator_NoExpectedHash_StaysDownloaded(t *testing.T) {
	repo := newStatefulRepo()
	task := repo.put(&RestoreTask{TargetDeviceSNs: []string{"SN001"}, Status: RestoreRunning}) // 无 ExpectedHash
	verifier := &fakeVerifier{byDevice: map[string]RestoreVerifyResult{
		"SN001": {Match: true, Method: RestoreVerifyGPVReadback},
	}}
	orch := NewRestoreVerificationOrchestrator(repo, verifier, DefaultVerificationConfig(), nil, zap.NewNop())

	require.NoError(t, orch.HandleDownloaded(context.Background(), task, time.Now()))
	got, _ := repo.GetByID(context.Background(), task.ID)
	assert.Equal(t, RestoreDownloaded, got.Status)
}

// TestOrchestrator_VerifierError_StaysDownloaded: 校验过程出错（设备不可达）默认停 downloaded 等重试。
func TestOrchestrator_VerifierError_StaysDownloaded(t *testing.T) {
	repo := newStatefulRepo()
	task := newDownloadedTask(repo, []string{"SN001"}, "exp")
	verifier := &fakeVerifier{err: errors.New("device unreachable")}
	orch := NewRestoreVerificationOrchestrator(repo, verifier, DefaultVerificationConfig(), nil, zap.NewNop())

	require.NoError(t, orch.HandleDownloaded(context.Background(), task, time.Now()))
	got, _ := repo.GetByID(context.Background(), task.ID)
	assert.Equal(t, RestoreDownloaded, got.Status)
}

// TestOrchestrator_VerifierError_FailPolicy: 严格策略下校验出错即判 failed。
func TestOrchestrator_VerifierError_FailPolicy(t *testing.T) {
	repo := newStatefulRepo()
	task := newDownloadedTask(repo, []string{"SN001"}, "exp")
	verifier := &fakeVerifier{err: errors.New("timeout")}
	cfg := VerificationConfig{OnVerifierError: VerifierErrorFail}
	orch := NewRestoreVerificationOrchestrator(repo, verifier, cfg, nil, zap.NewNop())

	require.NoError(t, orch.HandleDownloaded(context.Background(), task, time.Now()))
	got, _ := repo.GetByID(context.Background(), task.ID)
	assert.Equal(t, RestoreFailed, got.Status)
	require.NotNil(t, got.ErrorMessage)
	assert.Contains(t, *got.ErrorMessage, "timeout")
}

// ── TransferCompleteRouter wiring（成功分支走编排器 / 失败分支直接 failed）──

func TestRouter_RestoreSuccess_GoesThroughVerification(t *testing.T) {
	repo := newStatefulRepo()
	id := uuid.New()
	algo := RestoreHashMD5
	repo.put(&RestoreTask{ID: id, TargetDeviceSNs: []string{"SN001"}, Status: RestoreRunning,
		ExpectedHash: md5Ptr("exp"), HashAlgo: &algo})

	verifier := &fakeVerifier{byDevice: map[string]RestoreVerifyResult{
		"SN001": {Match: true, ObservedHash: "exp", Method: RestoreVerifyDeviceChecksum},
	}}
	orch := NewRestoreVerificationOrchestrator(repo, verifier, DefaultVerificationConfig(), nil, zap.NewNop())
	router := NewTransferCompleteRouter(nil, repo, nil, zap.NewNop())
	router.SetRestoreVerificationOrchestrator(orch)

	prefix := shortTaskID(id.String())
	// 直接调内部 markRestore（success=true）模拟成功的 TransferComplete。
	require.NoError(t, router.markRestore(context.Background(), prefix, true, time.Now(), "", "CELL_RESTORE_"+prefix))

	got, _ := repo.GetByID(context.Background(), id)
	assert.Equal(t, RestoreCompleted, got.Status)
	assert.Equal(t, []RestoreStatus{RestoreDownloaded, RestoreCompleted}, repo.transitions)
}

func TestRouter_RestoreSuccess_NoOrchestrator_LegacyComplete(t *testing.T) {
	repo := newStatefulRepo()
	id := uuid.New()
	repo.put(&RestoreTask{ID: id, TargetDeviceSNs: []string{"SN001"}, Status: RestoreRunning})
	router := NewTransferCompleteRouter(nil, repo, nil, zap.NewNop())
	// 不装编排器 → 退化旧行为：直接 completed。

	prefix := shortTaskID(id.String())
	require.NoError(t, router.markRestore(context.Background(), prefix, true, time.Now(), "", "CELL_RESTORE_"+prefix))

	got, _ := repo.GetByID(context.Background(), id)
	assert.Equal(t, RestoreCompleted, got.Status)
	assert.Equal(t, []RestoreStatus{RestoreCompleted}, repo.transitions, "无编排器不经 downloaded")
}

func TestRouter_RestoreFault_FailedDirectly(t *testing.T) {
	repo := newStatefulRepo()
	id := uuid.New()
	repo.put(&RestoreTask{ID: id, TargetDeviceSNs: []string{"SN001"}, Status: RestoreRunning})
	orch := NewRestoreVerificationOrchestrator(repo, &fakeVerifier{}, DefaultVerificationConfig(), nil, zap.NewNop())
	router := NewTransferCompleteRouter(nil, repo, nil, zap.NewNop())
	router.SetRestoreVerificationOrchestrator(orch)

	prefix := shortTaskID(id.String())
	// success=false（FaultCode!=0）→ 不走校验，直接 failed。
	require.NoError(t, router.markRestore(context.Background(), prefix, false, time.Now(), "fault: code=9001", "CELL_RESTORE_"+prefix))

	got, _ := repo.GetByID(context.Background(), id)
	assert.Equal(t, RestoreFailed, got.Status)
	assert.NotContains(t, repo.transitions, RestoreDownloaded, "失败分支不经 downloaded")
}
