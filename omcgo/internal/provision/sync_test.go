package provision

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/task"
)

// TestEnqueueGPVBatches_UsesSyncGPVExpiresIn 防止回退到全局 default_expires_in_seconds=120。
//
// 背景：BSC 等慢设备一次 Path B sync 会产生 200+ object 前缀 task。沿用 120s 时
// ACS 在 inform session 内串行 push 来不及消化，后半批被 ExpiredSweeper 标 expired，
// 导致前端临区/TRX 表项缺失（仅 168/254 实例落库）。
func TestEnqueueGPVBatches_UsesSyncGPVExpiresIn(t *testing.T) {
	var captured []*task.CreateTaskRequest
	mock := &mockCommandQueue{
		CreateFn: func(_ context.Context, req *task.CreateTaskRequest) (*task.Task, error) {
			captured = append(captured, req)
			return task.NewTask(req), nil
		},
	}
	svc := &SyncService{
		taskSvc:   mock,
		batchSize: 50,
		logger:    zap.NewNop(),
	}

	// 混合：1 个标量批 + 2 个 object 前缀（每个单批）共 3 批。
	paths := []string{"Dev.System.Mode", "Dev.WiFi.Radio.", "Dev.WiFi.SSID."}
	_, err := svc.EnqueueGPVBatches(context.Background(), "SN-TEST", paths, "src-1")
	require.NoError(t, err)

	require.Len(t, captured, 3, "应入队 3 批 task")
	for i, req := range captured {
		assert.Equalf(t, syncGPVTaskExpiresIn, req.ExpiresIn,
			"batch %d ExpiresIn 期望使用 syncGPVTaskExpiresIn=%d 而非默认 0", i, syncGPVTaskExpiresIn)
	}
	assert.Equal(t, 1800, syncGPVTaskExpiresIn, "syncGPVTaskExpiresIn 常量值不应被悄悄改小")
}
