package task

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// 回归测试（issue #116）：CompletedTotal 是 {source, status} 双标签 CounterVec，
// service.go 终态路径（completed/failed/expired/cancelled）曾只传 1 个 label 值，
// prometheus 直接 panic "inconsistent label cardinality"——ExpiredSweeper 每 10s
// panic 整个 worker，DELETE /devices/tasks/:task_id 实际取消成功但响应 500。

// completionCounterValue 读取双标签 CounterVec 的某个 (source, status) 计数。
func completionCounterValue(t *testing.T, vec *prometheus.CounterVec, source, status string) float64 {
	t.Helper()
	m := &dto.Metric{}
	c, err := vec.GetMetricWithLabelValues(source, status)
	require.NoError(t, err)
	require.NoError(t, c.Write(m))
	return m.GetCounter().GetValue()
}

// TestService_RecordCompletion_TwoLabelCardinality 覆盖 4 条终态路径共用的
// recordCompletion：任意 source × status 组合都不得 panic，且 (source, status)
// 双标签计数正确。
func TestService_RecordCompletion_TwoLabelCardinality(t *testing.T) {
	tests := []struct {
		name       string
		task       *Task
		status     TaskStatus
		wantSource string
		wantStatus string
	}{
		{
			name:       "MarkTaskCompleted 路径：api 任务计 completed",
			task:       &Task{ID: "t1", Source: TaskSourceAPI},
			status:     TaskStatusCompleted,
			wantSource: "api",
			wantStatus: "completed",
		},
		{
			name:       "MarkTaskFailed 路径：mml 任务计 failed",
			task:       &Task{ID: "t2", Source: TaskSourceMML},
			status:     TaskStatusFailed,
			wantSource: "mml",
			wantStatus: "failed",
		},
		{
			name:       "ExpireTask 路径（worker ExpiredSweeper）：scheduler 任务计 expired",
			task:       &Task{ID: "t3", Source: TaskSourceScheduler},
			status:     TaskStatusExpired,
			wantSource: "scheduler",
			wantStatus: "expired",
		},
		{
			name:       "CancelTask 路径：取消计 status=cancelled 而非 expired",
			task:       &Task{ID: "t4", Source: TaskSourceAPI},
			status:     TaskStatusCancelled,
			wantSource: "api",
			wantStatus: "cancelled",
		},
		{
			name:       "task 为 nil 时 source 空串兜底，不 panic",
			task:       nil,
			status:     TaskStatusCompleted,
			wantSource: "",
			wantStatus: "completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewTaskService(nil, nil, zap.NewNop())
			reg := prometheus.NewRegistry()
			m := NewTaskMetrics(reg)
			svc.SetMetrics(m)

			assert.NotPanics(t, func() { svc.recordCompletion(tt.task, tt.status) })

			assert.Equal(t, float64(1),
				completionCounterValue(t, m.CompletedTotal, tt.wantSource, tt.wantStatus))
		})
	}
}

// TestService_RecordCompletion_CancelNotCountedAsExpired 单独锁死 #116 的第二个
// 语义问题：CancelTask 曾把取消计入 status=expired，混淆超时与人工取消。
func TestService_RecordCompletion_CancelNotCountedAsExpired(t *testing.T) {
	svc := NewTaskService(nil, nil, zap.NewNop())
	reg := prometheus.NewRegistry()
	m := NewTaskMetrics(reg)
	svc.SetMetrics(m)

	svc.recordCompletion(&Task{ID: "t1", Source: TaskSourceAPI}, TaskStatusCancelled)

	assert.Equal(t, float64(1), completionCounterValue(t, m.CompletedTotal, "api", "cancelled"))
	assert.Equal(t, float64(0), completionCounterValue(t, m.CompletedTotal, "api", "expired"))
}

func TestService_RecordCompletion_NilMetricsSafe(t *testing.T) {
	// metrics 未注入（单测 / 未调 SetMetrics）时安全跳过，不 panic。
	svc := NewTaskService(nil, nil, zap.NewNop())
	assert.NotPanics(t, func() {
		svc.recordCompletion(&Task{ID: "t1", Source: TaskSourceAPI}, TaskStatusCompleted)
	})
}

// TestService_PG_TerminalPaths_MetricsNoPanic 走真实公开路径（miniredis + PG，
// PG 不可达自动 skip）：MarkTaskCompleted / MarkTaskFailed / CancelTask / ExpireTask
// 注入 metrics 后均不得 panic，且按 (source, status) 正确计数。
func TestService_PG_TerminalPaths_MetricsNoPanic(t *testing.T) {
	svc, _, _, repo := newServiceWithPG(t)
	if svc == nil {
		return
	}
	defer cleanupTestTasks(t, repo.pool)
	ctx := context.Background()

	reg := prometheus.NewRegistry()
	m := NewTaskMetrics(reg)
	svc.SetMetrics(m)

	mustCreate := func(suffix string) *Task {
		t.Helper()
		tk, err := svc.CreateTask(ctx, makeReq(testDeviceSNPrefix+suffix, "GetParameterValues"))
		require.NoError(t, err)
		require.NotNil(t, tk)
		return tk
	}

	// MarkTaskCompleted → (api, completed)
	tkDone := mustCreate("m116done")
	assert.NotPanics(t, func() {
		require.NoError(t, svc.MarkTaskCompleted(ctx, tkDone.ID, json.RawMessage(`{}`)))
	})
	assert.Equal(t, float64(1), completionCounterValue(t, m.CompletedTotal, "api", "completed"))

	// MarkTaskFailed → (api, failed)
	tkFail := mustCreate("m116fail")
	assert.NotPanics(t, func() {
		require.NoError(t, svc.MarkTaskFailed(ctx, tkFail.ID, 9001, "boom"))
	})
	assert.Equal(t, float64(1), completionCounterValue(t, m.CompletedTotal, "api", "failed"))

	// CancelTask → (api, cancelled)，且不得误计 expired
	tkCancel := mustCreate("m116cancel")
	assert.NotPanics(t, func() {
		require.NoError(t, svc.CancelTask(ctx, tkCancel.ID))
	})
	assert.Equal(t, float64(1), completionCounterValue(t, m.CompletedTotal, "api", "cancelled"))
	assert.Equal(t, float64(0), completionCounterValue(t, m.CompletedTotal, "api", "expired"))

	// ExpireTask（worker ExpiredSweeper 路径）→ (api, expired)
	tkExp := freshTaskForPG("m116exp", "m116exp")
	require.NoError(t, repo.Create(ctx, tkExp))
	assert.NotPanics(t, func() {
		require.NoError(t, svc.ExpireTask(ctx, tkExp))
	})
	assert.Equal(t, float64(1), completionCounterValue(t, m.CompletedTotal, "api", "expired"))
}
