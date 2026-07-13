package ufte

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	coremodel "github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/software"
)

type ufteTestTZProvider struct{ loc *time.Location }

func (p ufteTestTZProvider) Location(context.Context) *time.Location { return p.loc }

func withUFTETestTimezone(t *testing.T, loc *time.Location) {
	t.Helper()
	response.SetTimezoneProvider(ufteTestTZProvider{loc: loc})
	t.Cleanup(func() { response.SetTimezoneProvider(nil) })
}

func assertTimePtrEqual(t *testing.T, want time.Time, got *time.Time, msgAndArgs ...any) {
	t.Helper()
	require.NotNil(t, got, msgAndArgs...)
	assert.True(t, got.Equal(want), msgAndArgs...)
}

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
		assertTimePtrEqual(t, startedAt, got.StartedAt, "startedAt 应使用 task.StartedAt 而非 createdAt")
		assertTimePtrEqual(t, endedAt, got.EndedAt, "endedAt 应使用 task.EndedAt（已结束态才有值）")
		assertTimePtrEqual(t, scheduledAt, got.ScheduledAt)
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
		assertTimePtrEqual(t, startedAt, got.StartedAt)
		assert.Nil(t, got.EndedAt, "未结束任务 endedAt 应为空（JSON omitempty 不输出）")
		assert.Nil(t, got.ScheduledAt, "非预约模式 scheduledAt 应为空")
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
		assert.Nil(t, got.StartedAt)
		assert.Nil(t, got.EndedAt)
	})
}

func TestMapTask_ResponseTimezoneUsesSystemTimezone(t *testing.T) {
	utc, err := time.LoadLocation("UTC")
	require.NoError(t, err)
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	withUFTETestTimezone(t, utc)

	svc := newServiceForMap(t)
	catalog := mustCatalog(t, "RUNTIME_LOG_COLLECT")
	task := &software.UpgradeTask{
		ID:           uuid.New(),
		TaskName:     "log-utc-display",
		TaskType:     software.TaskTypeLogCollect,
		ProductClass: "4G eNB",
		Status:       software.TaskPending,
		CreatedAt:    coremodel.Time(time.Date(2026, 6, 23, 15, 0, 0, 0, shanghai)),
	}
	mapped, err := svc.mapTask(context.Background(), catalog, task)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/ufte/tasks", nil)
	response.OK(c, mapped)

	var env struct {
		Data struct {
			CreatedAt string `json:"createdAt"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.Equal(t, "2026-06-23T07:00:00Z", env.Data.CreatedAt)
}

// TestModelTimePtrToTimePtr 单独覆盖时间 helper 的两条路径，便于失败时定位。
func TestModelTimePtrToTimePtr(t *testing.T) {
	t.Run("nil_returns_empty", func(t *testing.T) {
		assert.Nil(t, modelTimePtrToTimePtr(nil))
	})
	t.Run("valid_returns_utc_time_pointer", func(t *testing.T) {
		shanghai := time.FixedZone("CST", 8*3600)
		want := time.Date(2026, 6, 23, 8, 15, 30, 0, time.UTC)
		value := coremodel.Time(time.Date(2026, 6, 23, 16, 15, 30, 0, shanghai))
		got := modelTimePtrToTimePtr(&value)
		assertTimePtrEqual(t, want, got)
	})
}
