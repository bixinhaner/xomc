package aggregator

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
)

// stubEnqueuer 记录设备级 Runner chain 出的设备组任务（#479 改动三）。
type stubEnqueuer struct {
	calls   []asyncjob.InsertRequest
	failErr error // 非 nil 时模拟入队失败
}

func (s *stubEnqueuer) Insert(_ context.Context, req asyncjob.InsertRequest) (uuid.UUID, error) {
	if s.failErr != nil {
		return uuid.Nil, s.failErr
	}
	s.calls = append(s.calls, req)
	return uuid.New(), nil
}

// GroupJobTypeFor 映射红绿：4 个设备级各映到对应组级；非设备级返回空。
func Test_GroupJobTypeFor_Mapping(t *testing.T) {
	assert.Equal(t, JobTypeHourlyGroup, GroupJobTypeFor(JobTypeHourly))
	assert.Equal(t, JobTypeDailyGroup, GroupJobTypeFor(JobTypeDaily))
	assert.Equal(t, JobTypeWeeklyGroup, GroupJobTypeFor(JobTypeWeekly))
	assert.Equal(t, JobTypeMonthlyGroup, GroupJobTypeFor(JobTypeMonthly))

	// 失败路径：设备组自身 / 未知 / 空串 都不应再链出后继（避免自我递归 chain）。
	assert.Equal(t, "", GroupJobTypeFor(JobTypeHourlyGroup))
	assert.Equal(t, "", GroupJobTypeFor("pm_kpi_export"))
	assert.Equal(t, "", GroupJobTypeFor(""))
}

// 成功路径：设备级该桶聚合成功后，确定性 chain 出对应粒度的设备组任务，
// 且 chain 的 [Start,End) 与设备级桶严格一致（不再靠固定错峰猜时间）。
func Test_Runner_Run_ChainsGroupJob_OnSuccess(t *testing.T) {
	start := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)
	payload, err := BuildPayload(start, end)
	require.NoError(t, err)

	db := &stubDB{execTag: pgconn.NewCommandTag("INSERT 0 5")}
	r := NewHourlyRunner(New(db, nil, nil))
	enq := &stubEnqueuer{}
	r.SetGroupChain(GroupJobTypeFor(r.JobType()), enq)

	job := &asyncjob.Job{ID: uuid.New(), JobType: r.JobType(), Payload: payload}
	result, err := r.Run(context.Background(), job)
	require.NoError(t, err)

	// 绿：恰好 chain 一个组任务，job_type=组级，payload 同窗口。
	require.Len(t, enq.calls, 1, "设备级成功后应 chain 出 1 个设备组任务")
	assert.Equal(t, JobTypeHourlyGroup, enq.calls[0].JobType)

	var chained runPayload
	require.NoError(t, json.Unmarshal(enq.calls[0].Payload, &chained))
	assert.True(t, chained.Start.Equal(start), "chain 的 Start 应等于设备级桶 Start")
	assert.True(t, chained.End.Equal(end), "chain 的 End 应等于设备级桶 End")

	// 结果里带上 chain 的可观测字段。
	var resultMap map[string]any
	require.NoError(t, json.Unmarshal(result, &resultMap))
	assert.Equal(t, JobTypeHourlyGroup, resultMap["chained_group_job_type"])
	assert.NotEmpty(t, resultMap["chained_group_job_id"])
}

// 未配置 chain（enqueuer/jobType 任一为空）：不 chain、不报错——退化为纯设备级。
func Test_Runner_Run_NoChain_WhenUnconfigured(t *testing.T) {
	start := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)
	payload, _ := BuildPayload(start, end)

	db := &stubDB{execTag: pgconn.NewCommandTag("INSERT 0 5")}
	r := NewHourlyRunner(New(db, nil, nil)) // 未调 SetGroupChain
	job := &asyncjob.Job{ID: uuid.New(), JobType: r.JobType(), Payload: payload}

	result, err := r.Run(context.Background(), job)
	require.NoError(t, err)

	var resultMap map[string]any
	require.NoError(t, json.Unmarshal(result, &resultMap))
	_, hasChainType := resultMap["chained_group_job_type"]
	assert.False(t, hasChainType, "未配置 chain 时结果不应含 chain 字段")
}

// 失败路径①：设备级聚合本身失败（反向窗口）→ 早返回、绝不 chain 组任务
// （避免在设备级未提交时就放出组任务读半成品）。
func Test_Runner_Run_DoesNotChain_WhenDeviceAggFails(t *testing.T) {
	start := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC) // end < start
	payload, _ := BuildPayload(start, end)

	r := NewHourlyRunner(New(&stubDB{}, nil, nil))
	enq := &stubEnqueuer{}
	r.SetGroupChain(GroupJobTypeFor(r.JobType()), enq)

	job := &asyncjob.Job{ID: uuid.New(), JobType: r.JobType(), Payload: payload}
	_, err := r.Run(context.Background(), job)
	require.Error(t, err)
	assert.Empty(t, enq.calls, "设备级聚合失败时不应 chain 任何组任务")
}

// 失败路径②：chain 入队抖动 → 让设备级任务整体失败（交由 asyncjob 重试本桶并重新 chain，
// 保证组任务不被一次入队抖动静默丢失）。
func Test_Runner_Run_FailsJob_WhenChainEnqueueErrors(t *testing.T) {
	start := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)
	payload, _ := BuildPayload(start, end)

	db := &stubDB{execTag: pgconn.NewCommandTag("INSERT 0 5")}
	r := NewHourlyRunner(New(db, nil, nil))
	enq := &stubEnqueuer{failErr: errors.New("boom: pg down")}
	r.SetGroupChain(GroupJobTypeFor(r.JobType()), enq)

	job := &asyncjob.Job{ID: uuid.New(), JobType: r.JobType(), Payload: payload}
	_, err := r.Run(context.Background(), job)
	require.Error(t, err, "chain 入队失败应让设备级任务失败以触发重试")
	assert.Contains(t, err.Error(), "enqueue group chain")
}
