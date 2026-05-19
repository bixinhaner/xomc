package task

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_CreateTask_DefaultExpiresIn_TableDriven 覆盖 T-0157 C1 默认超时兜底的三种分支:
//
//	1) 调用方未传 ExpiresIn (== 0) + service 配置了默认值 → 用默认值
//	2) 调用方显式传 ExpiresIn > 0 → 用调用方值（业务覆盖优先于配置默认）
//	3) 调用方未传 ExpiresIn + service 未配置默认值（== 0）→ ExpiresAt 仍为 nil（永不超时，等价历史行为）
//
// 时间断言用 [lo, hi] 区间避免 wall clock 抖动。
func Test_CreateTask_DefaultExpiresIn_TableDriven(t *testing.T) {
	type tc struct {
		name              string
		serviceDefault    int   // SetDefaultExpiresIn 注入值
		reqExpiresIn      int   // CreateTaskRequest.ExpiresIn
		wantExpiresAtNil  bool  // 期望 task.ExpiresAt == nil
		wantExpiresInSecs int   // 期望 ExpiresAt - CreatedAt（秒，仅在非 nil 时校验）
	}
	cases := []tc{
		{name: "未传_有默认_用默认", serviceDefault: 120, reqExpiresIn: 0, wantExpiresAtNil: false, wantExpiresInSecs: 120},
		{name: "显式传值_有默认_用调用方值", serviceDefault: 120, reqExpiresIn: 600, wantExpiresAtNil: false, wantExpiresInSecs: 600},
		{name: "未传_无默认_仍为永不超时", serviceDefault: 0, reqExpiresIn: 0, wantExpiresAtNil: true, wantExpiresInSecs: 0},
		{name: "显式传值_无默认_用调用方值", serviceDefault: 0, reqExpiresIn: 60, wantExpiresAtNil: false, wantExpiresInSecs: 60},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ts := newTestableService()
			ts.svc.SetDefaultExpiresIn(c.serviceDefault)
			ts.repo.createFn = func(ctx context.Context, task *Task) error { return nil }
			ts.queue.pushFn = func(ctx context.Context, task *Task) error { return nil }

			req := &CreateTaskRequest{
				DeviceSN:  "SN_EXP",
				Method:    "GetParameterValues",
				ExpiresIn: c.reqExpiresIn,
			}
			task, err := ts.CreateTask(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, task)

			if c.wantExpiresAtNil {
				assert.Nil(t, task.ExpiresAt, "ExpiresAt 应为 nil（永不超时）")
				return
			}

			require.NotNil(t, task.ExpiresAt, "ExpiresAt 不应为 nil")
			elapsed := task.ExpiresAt.Sub(task.CreatedAt)
			wantLo := time.Duration(c.wantExpiresInSecs)*time.Second - 100*time.Millisecond
			wantHi := time.Duration(c.wantExpiresInSecs)*time.Second + 100*time.Millisecond
			assert.GreaterOrEqual(t, elapsed, wantLo, "ExpiresAt - CreatedAt 偏小")
			assert.LessOrEqual(t, elapsed, wantHi, "ExpiresAt - CreatedAt 偏大")
		})
	}
}

// Test_SetDefaultExpiresIn_NegativeNormalized 验证负值被规范化为 0（不兜底，等价历史行为）。
// 避免配置文件被误填 -1 之类时把所有 task 都立即标过期。
func Test_SetDefaultExpiresIn_NegativeNormalized(t *testing.T) {
	ts := newTestableService()
	ts.svc.SetDefaultExpiresIn(-1)
	assert.Equal(t, 0, ts.svc.defaultExpiresIn, "负值应被规范化为 0")
}
