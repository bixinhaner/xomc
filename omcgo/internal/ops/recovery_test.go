package ops

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
)

func TestRecoverPendingTasks_MarksRunningAsFailed(t *testing.T) {
	taskA := OpsTask{ID: uuid.New(), Status: OpsTaskRunning, Creator: "alice"}
	taskB := OpsTask{ID: uuid.New(), Status: OpsTaskRunning, Creator: "bob"}

	var transitionedIDs []uuid.UUID
	var capturedTo OpsTaskStatus
	taskRepo := &mockTaskRepo{
		listFn: func(_ context.Context, filter TaskFilter) (*model.ListResponse[OpsTask], error) {
			require.NotNil(t, filter.Status)
			assert.Equal(t, OpsTaskRunning, *filter.Status, "RecoverPendingTasks 应只查 running")
			return model.NewListResponse([]OpsTask{taskA, taskB}, 2, 1, 1000), nil
		},
		transitionStatusFn: func(_ context.Context, id uuid.UUID, validFrom []OpsTaskStatus, to OpsTaskStatus, _, _ bool) error {
			transitionedIDs = append(transitionedIDs, id)
			capturedTo = to
			assert.Equal(t, []OpsTaskStatus{OpsTaskRunning}, validFrom)
			return nil
		},
	}
	executor := newTestExecutor(taskRepo, &stubTaskExecRepo{})

	count, err := executor.RecoverPendingTasks(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, count, "2 个 running task 应都被恢复")
	assert.Equal(t, OpsTaskFailed, capturedTo)
	assert.Len(t, transitionedIDs, 2)
}

func TestRecoverPendingTasks_NoRunningTasks(t *testing.T) {
	taskRepo := &mockTaskRepo{
		listFn: func(_ context.Context, _ TaskFilter) (*model.ListResponse[OpsTask], error) {
			return model.NewListResponse([]OpsTask{}, 0, 1, 1000), nil
		},
	}
	executor := newTestExecutor(taskRepo, &stubTaskExecRepo{})

	count, err := executor.RecoverPendingTasks(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, count, "无 running task 时应返 0")
}

func TestRecoverPendingTasks_TransitionFailureContinues(t *testing.T) {
	taskA := OpsTask{ID: uuid.New(), Status: OpsTaskRunning}
	taskB := OpsTask{ID: uuid.New(), Status: OpsTaskRunning}
	taskC := OpsTask{ID: uuid.New(), Status: OpsTaskRunning}

	var transitioned int
	taskRepo := &mockTaskRepo{
		listFn: func(_ context.Context, _ TaskFilter) (*model.ListResponse[OpsTask], error) {
			return model.NewListResponse([]OpsTask{taskA, taskB, taskC}, 3, 1, 1000), nil
		},
		transitionStatusFn: func(_ context.Context, id uuid.UUID, _ []OpsTaskStatus, _ OpsTaskStatus, _, _ bool) error {
			if id == taskB.ID {
				return errors.New("simulated transition error")
			}
			transitioned++
			return nil
		},
	}
	executor := newTestExecutor(taskRepo, &stubTaskExecRepo{})

	count, err := executor.RecoverPendingTasks(context.Background())
	require.NoError(t, err, "单 task 失败不应中断主流程")
	assert.Equal(t, 2, count, "成功转移的 2 个被计数（A+C），失败的 B 不计")
	assert.Equal(t, 2, transitioned)
}

func TestRecoverPendingTasks_ListErrorPropagates(t *testing.T) {
	taskRepo := &mockTaskRepo{
		listFn: func(_ context.Context, _ TaskFilter) (*model.ListResponse[OpsTask], error) {
			return nil, errors.New("db unreachable")
		},
	}
	executor := newTestExecutor(taskRepo, &stubTaskExecRepo{})

	count, err := executor.RecoverPendingTasks(context.Background())
	require.Error(t, err, "List 失败应传播")
	assert.Equal(t, 0, count)
	assert.Contains(t, err.Error(), "list running tasks for recovery")
}
