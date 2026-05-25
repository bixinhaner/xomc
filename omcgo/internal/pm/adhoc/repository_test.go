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
	r := NewPgRepository(pool)
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
	r := NewPgRepository(pool)
	ctx := context.Background()

	req := sampleReq()
	req.CronExpr = nil
	_, err := r.Create(ctx, req)
	assert.Error(t, err)
}

func Test_Repository_ListByStatus(t *testing.T) {
	pool := openPoolOrSkip(t)
	defer pool.Close()
	r := NewPgRepository(pool)
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
	r := NewPgRepository(pool)
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
	r := NewPgRepository(pool)
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

func Test_Repository_LockNextPending(t *testing.T) {
	pool := openPoolOrSkip(t)
	defer pool.Close()
	r := NewPgRepository(pool)
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
	r := NewPgRepository(pool)
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
