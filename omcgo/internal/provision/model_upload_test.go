package provision

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// stubDiscoveryRepo 实现 ParameterDiscoveryLogRepository，仅 UpdateStatus 可注入行为，
// 其余方法不参与本测试。
type stubDiscoveryRepo struct {
	updateStatusErr error
	updateCalls     int
}

func (s *stubDiscoveryRepo) Create(_ context.Context, _ *ParameterDiscoveryLog) error { return nil }
func (s *stubDiscoveryRepo) Update(_ context.Context, _ *ParameterDiscoveryLog) error { return nil }
func (s *stubDiscoveryRepo) GetByID(_ context.Context, _ uuid.UUID) (*ParameterDiscoveryLog, error) {
	return nil, nil
}
func (s *stubDiscoveryRepo) GetByDeviceID(_ context.Context, _ uuid.UUID) (*ParameterDiscoveryLog, error) {
	return nil, nil
}
func (s *stubDiscoveryRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ DiscoveryStatus, _ string) error {
	s.updateCalls++
	return s.updateStatusErr
}

// newTestModelUploadService 构造一个仅含 discoveryRepo + logger + metrics 的最小服务，
// 足以测试 updateDiscoveryStatus 的错误处理（HIGH-27）。
func newTestModelUploadService(repo ParameterDiscoveryLogRepository, logger *zap.Logger, m *Metrics) *ModelUploadService {
	return &ModelUploadService{discoveryRepo: repo, logger: logger, metrics: m}
}

type stubModelUploadParamSync struct {
	calls []modelUploadParamSyncCall
	err   error
}

type modelUploadParamSyncCall struct {
	sourceID string
	logID    uuid.UUID
	status   string
}

func (s *stubModelUploadParamSync) SubmitModelUploadParamSync(_ context.Context, _ *model.Device, sourceID string, modelUploadID uuid.UUID, status string) (bool, int, error) {
	s.calls = append(s.calls, modelUploadParamSyncCall{sourceID: sourceID, logID: modelUploadID, status: status})
	if s.err != nil {
		return true, 0, s.err
	}
	return true, 7, nil
}

// TestUpdateDiscoveryStatus_Success 验证写库成功时不记 warn、不打点。
func TestUpdateDiscoveryStatus_Success(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	repo := &stubDiscoveryRepo{updateStatusErr: nil}
	metrics := NewMetrics(nil)
	svc := newTestModelUploadService(repo, zap.New(core), metrics)

	svc.updateDiscoveryStatus(context.Background(), uuid.New(), DiscoveryCompleted, "ok", "skipped_fileupload", "SN-1")

	if repo.updateCalls != 1 {
		t.Fatalf("UpdateStatus 应被调用 1 次，实际 %d", repo.updateCalls)
	}
	if got := logs.FilterMessage("failed to update discovery log status").Len(); got != 0 {
		t.Fatalf("成功路径不应记 warn，实际 %d 条", got)
	}
	if got := testutil.ToFloat64(metrics.DiscoveryStatusUpdateErrorsTotal.WithLabelValues("skipped_fileupload")); got != 0 {
		t.Fatalf("成功路径 metric 应为 0，实际 %v", got)
	}
}

// TestUpdateDiscoveryStatus_Failure 验证写库失败时记 warn + 打点（按 operation 分桶），
// 且不向上抛错（housekeeping 语义：失败可观测但不阻断主流程，HIGH-27）。
func TestUpdateDiscoveryStatus_Failure(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	repo := &stubDiscoveryRepo{updateStatusErr: errors.New("connection refused")}
	metrics := NewMetrics(nil)
	svc := newTestModelUploadService(repo, zap.New(core), metrics)

	svc.updateDiscoveryStatus(context.Background(), uuid.New(), DiscoveryFailed, "boom", "minio_get_failed", "SN-2")

	if repo.updateCalls != 1 {
		t.Fatalf("UpdateStatus 应被调用 1 次，实际 %d", repo.updateCalls)
	}
	warnLogs := logs.FilterMessage("failed to update discovery log status")
	if warnLogs.Len() != 1 {
		t.Fatalf("失败路径应记 1 条 warn，实际 %d 条", warnLogs.Len())
	}
	// 校验日志带上了 operation 上下文，便于运维定位。
	fields := warnLogs.All()[0].ContextMap()
	if fields["operation"] != "minio_get_failed" {
		t.Fatalf("warn 日志应含 operation=minio_get_failed，实际 %v", fields["operation"])
	}
	if got := testutil.ToFloat64(metrics.DiscoveryStatusUpdateErrorsTotal.WithLabelValues("minio_get_failed")); got != 1 {
		t.Fatalf("失败路径 metric{minio_get_failed} 应为 1，实际 %v", got)
	}
	// 不同 operation 桶应彼此独立。
	if got := testutil.ToFloat64(metrics.DiscoveryStatusUpdateErrorsTotal.WithLabelValues("parse_xml_failed")); got != 0 {
		t.Fatalf("无关 operation 桶应为 0，实际 %v", got)
	}
}

// TestUpdateDiscoveryStatus_NilMetrics 验证 metrics 为 nil 时不 panic（仍记 warn）。
func TestUpdateDiscoveryStatus_NilMetrics(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	repo := &stubDiscoveryRepo{updateStatusErr: errors.New("boom")}
	svc := newTestModelUploadService(repo, zap.New(core), nil)

	svc.updateDiscoveryStatus(context.Background(), uuid.New(), DiscoveryFailed, "boom", "intersect_failed", "SN-3")

	if got := logs.FilterMessage("failed to update discovery log status").Len(); got != 1 {
		t.Fatalf("nil metrics 下仍应记 1 条 warn，实际 %d", got)
	}
}

// TestMetrics_Counters 验证 NewMetrics 注册的两个 counter 可独立累加。
func TestMetrics_Counters(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	m.discoveryStatusUpdateErr("op_a")
	m.discoveryStatusUpdateErr("op_a")
	m.redisThrottleFailure("device_online")

	if got := testutil.ToFloat64(m.DiscoveryStatusUpdateErrorsTotal.WithLabelValues("op_a")); got != 2 {
		t.Fatalf("discovery counter{op_a} 应为 2，实际 %v", got)
	}
	if got := testutil.ToFloat64(m.RedisThrottleFailuresTotal.WithLabelValues("device_online")); got != 1 {
		t.Fatalf("redis throttle counter{device_online} 应为 1，实际 %v", got)
	}

	// nil 接收者安全。
	var nilM *Metrics
	nilM.discoveryStatusUpdateErr("x")
	nilM.redisThrottleFailure("y")
}

func TestModelUploadTerminalSync_SubmitsUploadedStatus(t *testing.T) {
	submitter := &stubModelUploadParamSync{}
	svc := newTestModelUploadService(&stubDiscoveryRepo{}, zap.NewNop(), nil)
	svc.SetParamSyncSubmitter(submitter)
	logID := uuid.New()
	dev := &model.Device{ID: uuid.New(), SerialNumber: "SN-MODEL-UPLOAD"}
	svc.rememberTerminalSync(logID, true)

	svc.submitTerminalSync(context.Background(), dev, logID, "uploaded")

	if len(submitter.calls) != 1 {
		t.Fatalf("expected one model_upload parameter sync submit, got %d", len(submitter.calls))
	}
	if submitter.calls[0].status != "uploaded" {
		t.Fatalf("expected uploaded status, got %q", submitter.calls[0].status)
	}
	if submitter.calls[0].sourceID != logID.String() || submitter.calls[0].logID != logID {
		t.Fatalf("expected log id metadata to be propagated")
	}
}

func TestModelUploadTerminalSync_CanBeDisabledForFirmwareChanged(t *testing.T) {
	submitter := &stubModelUploadParamSync{}
	svc := newTestModelUploadService(&stubDiscoveryRepo{}, zap.NewNop(), nil)
	svc.SetParamSyncSubmitter(submitter)
	logID := uuid.New()
	svc.rememberTerminalSync(logID, false)

	svc.submitTerminalSync(context.Background(), &model.Device{ID: uuid.New(), SerialNumber: "SN-FW"}, logID, "not_supported")

	if len(submitter.calls) != 0 {
		t.Fatalf("firmware-triggered model upload must not submit parameter sync, got %d calls", len(submitter.calls))
	}
}
