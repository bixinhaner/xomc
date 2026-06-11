package task

import (
	"context"
	stderrors "errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	coreerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// TestErrTaskNotFound_MapsTo404 锁定 #125 回归：ErrTaskNotFound 必须被识别为
// "资源不存在"，从而经 coreerrors.HTTPStatusFromError 映射为 HTTP 404，而非 500。
//
// 修复前 handler 用 err.Error()=="task not found" 精确比对，而 service 返回带
// 后缀的 fmt.Errorf("task not found: %s", id)，永不匹配 → 误判 500。改为哨兵
// 错误 + errors.Is 后，无论 ID 是什么都能正确判定。
func TestErrTaskNotFound_MapsTo404(t *testing.T) {
	// 哨兵自身可被 errors.Is 识别。
	assert.True(t, stderrors.Is(ErrTaskNotFound, ErrTaskNotFound))

	// 包装了核心 ErrNotFound，故 HTTPStatusFromError → 404。
	assert.True(t, stderrors.Is(ErrTaskNotFound, coreerrors.ErrNotFound))
	assert.Equal(t, http.StatusNotFound, coreerrors.HTTPStatusFromError(ErrTaskNotFound))

	// 错误消息不外泄底层存储/SQL 细节。
	assert.Contains(t, ErrTaskNotFound.Error(), "task not found")
	assert.NotContains(t, ErrTaskNotFound.Error(), "SELECT")
	assert.NotContains(t, ErrTaskNotFound.Error(), "redis")
}

// TestRedisQueue_MarkMethods_NotFoundSentinel 覆盖三条真实 not-found 路径
// （MarkTaskSent / MarkTaskCompleted / MarkTaskFailed[WithResult]）：任务不存在
// 时必须返回可被 errors.Is(err, ErrTaskNotFound) 判定的哨兵错误。
func TestRedisQueue_MarkMethods_NotFoundSentinel(t *testing.T) {
	tests := []struct {
		name string
		call func(q *RedisTaskQueue, ctx context.Context) error
	}{
		{
			name: "MarkTaskSent",
			call: func(q *RedisTaskQueue, ctx context.Context) error {
				return q.MarkTaskSent(ctx, "missing-id", "cwmp-x")
			},
		},
		{
			name: "MarkTaskCompleted",
			call: func(q *RedisTaskQueue, ctx context.Context) error {
				return q.MarkTaskCompleted(ctx, "missing-id", nil)
			},
		},
		{
			name: "MarkTaskFailed",
			call: func(q *RedisTaskQueue, ctx context.Context) error {
				return q.MarkTaskFailed(ctx, "missing-id", 1, "boom")
			},
		},
		{
			name: "MarkTaskFailedWithResult",
			call: func(q *RedisTaskQueue, ctx context.Context) error {
				return q.MarkTaskFailedWithResult(ctx, "missing-id", 1, "boom", nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q, _ := newRedisQueueWithMini(t)
			err := tc.call(q, context.Background())
			require.Error(t, err)
			assert.True(t, stderrors.Is(err, ErrTaskNotFound),
				"%s 对不存在任务应返回 ErrTaskNotFound，got %v", tc.name, err)
			assert.Equal(t, http.StatusNotFound, coreerrors.HTTPStatusFromError(err))
		})
	}
}

// TestService_CancelTask_NotFoundSentinel 确认 service 层 CancelTask 的 not-found
// 路径返回哨兵错误（经 testable 镜像，与生产 CancelTask 逻辑一致）。
func TestService_CancelTask_NotFoundSentinel(t *testing.T) {
	ts := newTestableService()
	ts.queue.getByIDFn = func(ctx context.Context, taskID string) (*Task, error) {
		return nil, nil
	}
	ts.repo.getByIDFn = func(ctx context.Context, id string) (*Task, error) {
		return nil, nil
	}

	err := ts.CancelTask(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.True(t, stderrors.Is(err, ErrTaskNotFound))
	assert.Equal(t, http.StatusNotFound, coreerrors.HTTPStatusFromError(err))
}
