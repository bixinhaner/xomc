package ufte

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	coremodel "github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/software"
)

// TestMapTask_SerializesStartedAndEndedAt 锁定 issue #571 修复：
// 任务管理列表的「开始时间 / 结束时间」依赖 mapTask 把 software.UpgradeTask
// 的 *model.Time 指针字段序列化为前端可消费的 ISO 字符串；nil 时返回空串走 omitempty。
func TestMapTask_SerializesStartedAndEndedAt(t *testing.T) {
	svc := newServiceForMap(t)
	catalog := mustCatalog(t, "ENB_IMG_UPGRADE")
	startedAt := time.Date(2026, 6, 23, 8, 0, 0, 0, time.UTC)
	endedAt := time.Date(2026, 6, 23, 8, 15, 30, 0, time.UTC)
	scheduledAt := time.Date(2026, 6, 23, 7, 30, 0, 0, time.UTC)

	t.Run("ended_task_serializes_both_times", func(t *testing.T) {
		started := coremodel.Time(startedAt)
		ended := coremodel.Time(endedAt)
		scheduled := coremodel.Time(scheduledAt)
		task := &software.UpgradeTask{
			ID:           uuid.New(),
			TaskName:     "smoke-ended",
			TaskType:     software.TaskTypeUpgrade,
			ProductClass: "4G eNB",
			Status:       software.TaskEnded,
			CreatedAt:    coremodel.Time(time.Date(2026, 6, 23, 7, 0, 0, 0, time.UTC)),
			StartedAt:    &started,
			EndedAt:      &ended,
			ScheduledAt:  &scheduled,
		}

		got, err := svc.mapTask(context.Background(), catalog, task)
		require.NoError(t, err)
		assert.Equal(t, "2026-06-23T08:00:00Z", got.StartedAt, "startedAt 应使用 task.StartedAt 而非 createdAt")
		assert.Equal(t, "2026-06-23T08:15:30Z", got.EndedAt, "endedAt 应使用 task.EndedAt（已结束态才有值）")
		assert.Equal(t, "2026-06-23T07:30:00Z", got.ScheduledAt)
		assert.NotEqual(t, got.CreatedAt, got.EndedAt, "endedAt 不应错用 createdAt（修复前的 bug 表象）")
	})

	t.Run("in_progress_task_endedAt_empty", func(t *testing.T) {
		started := coremodel.Time(startedAt)
		task := &software.UpgradeTask{
			ID:           uuid.New(),
			TaskName:     "smoke-running",
			TaskType:     software.TaskTypeUpgrade,
			ProductClass: "4G eNB",
			Status:       software.TaskInProgress,
			CreatedAt:    coremodel.Time(time.Date(2026, 6, 23, 7, 0, 0, 0, time.UTC)),
			StartedAt:    &started,
			EndedAt:      nil, // 未结束 → repo 未写入
			ScheduledAt:  nil,
		}

		got, err := svc.mapTask(context.Background(), catalog, task)
		require.NoError(t, err)
		assert.Equal(t, "2026-06-23T08:00:00Z", got.StartedAt)
		assert.Empty(t, got.EndedAt, "未结束任务 endedAt 应为空串（JSON omitempty 不输出）")
		assert.Empty(t, got.ScheduledAt, "非预约模式 scheduledAt 应为空串")
	})

	t.Run("pending_task_both_times_empty", func(t *testing.T) {
		task := &software.UpgradeTask{
			ID:           uuid.New(),
			TaskName:     "smoke-pending",
			TaskType:     software.TaskTypeUpgrade,
			ProductClass: "4G eNB",
			Status:       software.TaskPending,
			CreatedAt:    coremodel.Time(time.Date(2026, 6, 23, 7, 0, 0, 0, time.UTC)),
		}
		got, err := svc.mapTask(context.Background(), catalog, task)
		require.NoError(t, err)
		assert.Empty(t, got.StartedAt)
		assert.Empty(t, got.EndedAt)
	})
}

// TestModelTimePtrToString 单独覆盖时间序列化 helper 的两条路径，便于失败时定位。
func TestModelTimePtrToString(t *testing.T) {
	t.Run("nil_returns_empty", func(t *testing.T) {
		assert.Empty(t, modelTimePtrToString(nil))
	})
	t.Run("valid_returns_rfc3339", func(t *testing.T) {
		value := coremodel.Time(time.Date(2026, 6, 23, 8, 15, 30, 0, time.UTC))
		assert.Equal(t, "2026-06-23T08:15:30Z", modelTimePtrToString(&value))
	})
}
