//go:build integration

package adhoc

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 6 个集成测试（依赖 docker postgres + migration 000162）：
//   1. Create + Get round-trip
//   2. Create continuous without cron 拒绝
//   3. List by Status 过滤
//   4. Cancel pending→canceled
//   5. Cancel succeeded→ErrTerminalState
//   6. LockNextPending：pending→running 抢锁
//
// 运行：OMCGO_DB_DSN=postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable \
//        go test -tags integration -count=1 ./internal/pm/adhoc/...
//
// 注：KPI/时序库物理分离后 NewPgRepository(pgPool, tsPool) 双池——pgPool 管 pm_tasks/
// pm_adhoc_task_runs，tsPool 写 pm_adhoc_aggregation_results。集成测试单 DSN，两池传同一
// pool（同实例），InsertResults round-trip 行为不变。

func openPoolOrSkip(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("OMCGO_DB_DSN")
	if dsn == "" {
		t.Skip("OMCGO_DB_DSN not set; skipping integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	return pool
}

func sampleReq() CreateRequest {
	cron := "5 * * * *"
	return CreateRequest{
		Name:          "test-adhoc-" + uuid.New().String()[:8],
		Mode:          ModeContinuous,
		CronExpr:      &cron,
		DeviceSNs:     []string{"AGGR-INT-TEST-001"},
		MetricPaths:   []string{"L.Cell.Avail.Dur"},
		Granularities: []string{"hourly"},
		WindowStart:   time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		WindowEnd:     time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
		Creator:       "test",
	}
}

func cleanup(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) {
	t.Helper()
	_, _ = pool.Exec(context.Background(),
		`DELETE FROM pm_tasks WHERE id=$1 AND task_subtype=$2`, id, TaskSubtype)
}

func Test_Repository_CreateAndGet(t *testing.T) {
	pool := openPoolOrSkip(t)
	defer pool.Close()
	r := NewPgRepository(pool, pool)
	ctx := context.Background()

	req := sampleReq()
	id, err := r.Create(ctx, req)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, id)
	defer cleanup(t, pool, id)

	got, err := r.Get(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, req.Name, got.Name)
	assert.Equal(t, ModeContinuous, got.Mode)
	assert.Equal(t, StatusPending, got.Status)
	assert.Equal(t, req.DeviceSNs, got.DeviceSNs)
	assert.Equal(t, req.MetricPaths, got.MetricPaths)
	assert.Equal(t, req.Granularities, got.Granularities)
	assert.Equal(t, 0, got.Progress)
}

func Test_Repository_CreateContinuousWithoutCronRejected(t *testing.T) {
	pool := openPoolOrSkip(t)
	defer pool.Close()
	r := NewPgRepository(pool, pool)
	ctx := context.Background()

	req := sampleReq()
	req.CronExpr = nil
	_, err := r.Create(ctx, req)
	assert.Error(t, err)
}

func Test_Repository_ListByStatus(t *testing.T) {
	pool := openPoolOrSkip(t)
	defer pool.Close()
	r := NewPgRepository(pool, pool)
	ctx := context.Background()

	req := sampleReq()
	id, err := r.Create(ctx, req)
	require.NoError(t, err)
	defer cleanup(t, pool, id)

	pending := StatusPending
	list, err := r.List(ctx, ListFilter{Status: &pending, Limit: 50})
	require.NoError(t, err)
	found := false
	for _, t := range list {
		if t.ID == id {
			found = true
			break
		}
	}
	assert.True(t, found, "newly created pending task should appear in pending list")
}

func Test_Repository_CancelPending(t *testing.T) {
	pool := openPoolOrSkip(t)
	defer pool.Close()
	r := NewPgRepository(pool, pool)
	ctx := context.Background()

	req := sampleReq()
	id, err := r.Create(ctx, req)
	require.NoError(t, err)
	defer cleanup(t, pool, id)

	require.NoError(t, r.Cancel(ctx, id))
	got, err := r.Get(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, StatusCanceled, got.Status)
}

func Test_Repository_CancelTerminalRejected(t *testing.T) {
	pool := openPoolOrSkip(t)
	defer pool.Close()
	r := NewPgRepository(pool, pool)
	ctx := context.Background()

	req := sampleReq()
	id, err := r.Create(ctx, req)
	require.NoError(t, err)
	defer cleanup(t, pool, id)

	// 先把任务直接 UpdateStatus → succeeded（绕过 worker 实际跑）
	hundred := 100
	require.NoError(t, r.UpdateStatus(ctx, id, StatusSucceeded, &hundred, ""))

	err = r.Cancel(ctx, id)
	assert.ErrorIs(t, err, ErrTerminalState)
}

func Test_Repository_Update_OneshotRequeueTerminalStatuses(t *testing.T) {
	pool := openPoolOrSkip(t)
	defer pool.Close()
	r := NewPgRepository(pool, pool)
	ctx := context.Background()

	for _, tt := range []struct {
		name           string
		beforeStatus   Status
		beforeProgress int
		wantStatus     Status
		wantProgress   int
	}{
		{name: "succeeded", beforeStatus: StatusSucceeded, beforeProgress: 100, wantStatus: StatusPending, wantProgress: 0},
		{name: "failed", beforeStatus: StatusFailed, beforeProgress: 100, wantStatus: StatusPending, wantProgress: 0},
		{name: "canceled", beforeStatus: StatusCanceled, beforeProgress: 40, wantStatus: StatusCanceled, wantProgress: 40},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := sampleReq()
			req.Mode = ModeOneshot
			req.CronExpr = nil
			req.Name = "oneshot-requeue-" + tt.name + "-" + uuid.New().String()[:8]
			id, err := r.Create(ctx, req)
			require.NoError(t, err)
			defer cleanup(t, pool, id)

			_, err = pool.Exec(ctx,
				`UPDATE pm_tasks SET status=$1, progress=$2 WHERE id=$3`,
				string(tt.beforeStatus), tt.beforeProgress, id)
			require.NoError(t, err)

			require.NoError(t, r.Update(ctx, id, UpdateRequest{
				IsBuiltin:       false,
				Mode:            ModeOneshot,
				RequeueTerminal: true,
				Name:            req.Name,
				DeviceSNs:       req.DeviceSNs,
				MetricPaths:     req.MetricPaths,
				Granularities:   req.Granularities,
				WindowStart:     req.WindowStart.Add(-time.Hour),
				WindowEnd:       req.WindowEnd.Add(-time.Hour),
				Visibility:      req.Visibility,
			}))

			got, err := r.Get(ctx, id)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, got.Status)
			assert.Equal(t, tt.wantProgress, got.Progress)
			assert.True(t, got.WindowStart.Equal(req.WindowStart.Add(-time.Hour)))
			assert.True(t, got.WindowEnd.Equal(req.WindowEnd.Add(-time.Hour)))
		})
	}
}

func Test_Repository_LockNextPending(t *testing.T) {
	pool := openPoolOrSkip(t)
	defer pool.Close()
	r := NewPgRepository(pool, pool)
	ctx := context.Background()

	req := sampleReq()
	id, err := r.Create(ctx, req)
	require.NoError(t, err)
	defer cleanup(t, pool, id)

	got, err := r.LockNextPending(ctx, "test-worker-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, StatusRunning, got.Status)
	assert.NotNil(t, got.LockOwner)
	assert.Equal(t, "test-worker-1", *got.LockOwner)

	// 第二次抢锁同 id 不会再被抢（pending→running 已切换）
	// 但其它 pending 任务可能被抢——这里我们 explicitly 验证不会再返回同 id
	got2, err2 := r.LockNextPending(ctx, "test-worker-2")
	if err2 == nil {
		assert.NotEqual(t, id, got2.ID, "same task should not be locked twice")
	}
}

// InsertResults round-trip
func Test_Repository_InsertResultsRoundTrip(t *testing.T) {
	pool := openPoolOrSkip(t)
	defer pool.Close()
	r := NewPgRepository(pool, pool)
	ctx := context.Background()

	taskID := uuid.New()
	now := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	rows := []ResultRow{
		{
			TaskID: taskID, DeviceOUI: "INTTST", DeviceSN: "AGGR-ADH-001",
			MetricPath: "L.Cell.Avail.Dur", MetricType: "counter", MetricValue: 700,
			Granularity: "hourly", Time: now.Add(time.Hour), StartTime: now, EndTime: now.Add(time.Hour),
		},
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM pm_adhoc_aggregation_results WHERE task_id=$1`, taskID)
	}()

	require.NoError(t, r.InsertResults(ctx, rows))

	var count int
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM pm_adhoc_aggregation_results WHERE task_id=$1`, taskID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

// T-0186：运行历史写入 + 倒序查询 + 分页 round-trip。
func Test_Repository_TaskRuns_OrderAndPaginate(t *testing.T) {
	pool := openPoolOrSkip(t)
	defer pool.Close()
	r := NewPgRepository(pool, pool)
	ctx := context.Background()

	taskID := uuid.New()
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM pm_adhoc_task_runs WHERE task_id=$1`, taskID)
	}()

	// run_seq 自增：连续插 3 次
	base := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		seq, err := r.NextRunSeq(ctx, taskID)
		require.NoError(t, err)
		assert.Equal(t, i+1, seq, "run_seq 应自增")
		started := base.Add(time.Duration(i) * time.Hour)
		runID, err := r.InsertRun(ctx, TaskRun{
			TaskID: taskID, RunSeq: seq, Granularity: "hourly", Dimension: "device",
			Status: StatusRunning, StartedAt: started,
		})
		require.NoError(t, err)
		// 偶数成功，最后一个失败
		if i == 2 {
			require.NoError(t, r.FinishRun(ctx, runID, StatusFailed, 0, "boom"))
		} else {
			require.NoError(t, r.FinishRun(ctx, runID, StatusSucceeded, 10, ""))
		}
	}

	// 倒序：最新（run_seq=3，started 最大）在最前
	runs, err := r.ListRuns(ctx, taskID, 50, 0)
	require.NoError(t, err)
	require.Len(t, runs, 3)
	assert.Equal(t, 3, runs[0].RunSeq, "倒序首行应是最新一次")
	assert.Equal(t, StatusFailed, runs[0].Status)
	assert.Equal(t, "boom", runs[0].Error)
	assert.Equal(t, 1, runs[2].RunSeq)

	// 分页：limit=1 offset=1 → 中间那次（run_seq=2）
	page, err := r.ListRuns(ctx, taskID, 1, 1)
	require.NoError(t, err)
	require.Len(t, page, 1)
	assert.Equal(t, 2, page[0].RunSeq)
}
