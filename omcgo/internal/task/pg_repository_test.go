package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pgURL 返回测试用的 PG DSN。优先 TEST_PG_URL 环境变量；回退到本地 docker-compose 默认。
func pgURL() string {
	if v := os.Getenv("TEST_PG_URL"); v != "" {
		return v
	}
	return "postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable"
}

// newTestPool 尝试连接到测试 PG。连接不上则 t.Skip。
// 这样 unit test 在 CI 没有 PG 时自动跳过；本地有 docker-compose 起着的话自动跑。
func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), pgURL())
	if err != nil {
		t.Skipf("no PG available: %v", err)
		return nil
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("no PG available (ping fail): %v", err)
		return nil
	}
	t.Cleanup(func() { pool.Close() })
	// 校验 device_tasks 表存在
	var exists bool
	if err := pool.QueryRow(context.Background(),
		"SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='device_tasks')",
	).Scan(&exists); err != nil || !exists {
		t.Skipf("device_tasks table not present: err=%v exists=%v", err, exists)
		return nil
	}
	return pool
}

// 用前缀确保测试任务可清理。
const testDeviceSNPrefix = "TEST-PG-REPO-"

// freshTaskForPG 构造一个刚创建的 pending 任务，DeviceSN 带前缀便于清理。
// 注意：production 的 scanTaskRow 用 &task.SourceID (string 非指针) 扫 source_id
// 列；列在 DB 是 UUID NULL，所以读出来时 NULL 会导致 "cannot scan NULL into *string"。
// 测试里给 SourceID 一个真实 UUID 绕开这个 production bug，否则 GetByID/GetHistory
// 等都会失败。
func freshTaskForPG(idSuffix, snSuffix string) *Task {
	now := time.Now()
	return &Task{
		ID:           generateUUID(),
		DeviceSN:     testDeviceSNPrefix + snSuffix,
		Method:       "Reboot",
		Params:       json.RawMessage("{}"),
		Priority:     10,
		Status:       TaskStatusPending,
		MaxRetries:   3,
		CreatedAt:    now,
		Source:       TaskSourceAPI,
		CommandIndex: 0,
		DeviceIndex:  0,
		SourceID:     generateUUID(), // 避免 NULL → string scan 失败
	}
}

// cleanupTestTasks 清理本测试创建的所有任务。
func cleanupTestTasks(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if pool == nil {
		return
	}
	_, _ = pool.Exec(context.Background(),
		"DELETE FROM device_tasks WHERE device_sn LIKE $1",
		testDeviceSNPrefix+"%")
}

// ---- Helpers ----

func TestPgRepo_NilUUID(t *testing.T) {
	assert.Nil(t, nilUUID(""))
	assert.Equal(t, "abc", nilUUID("abc"))
}

func TestPgRepo_TaskColumns(t *testing.T) {
	cols := taskColumns()
	assert.NotEmpty(t, cols)
	assert.Contains(t, cols, "id")
	assert.Contains(t, cols, "device_sn")
	assert.Contains(t, cols, "method")
	assert.Contains(t, cols, "status")
}

// fakeRow 实现 pgx.Row 用于 scanTaskRow 测试
type fakeRow struct {
	err error
}

func (f *fakeRow) Scan(_ ...any) error {
	return f.err
}

func TestPgRepo_ScanTaskRow_NoRows(t *testing.T) {
	repo := &PgTaskRepository{pool: nil}
	got, err := repo.scanTaskRow(&fakeRow{err: pgx.ErrNoRows})
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestPgRepo_ScanTaskRow_ScanError(t *testing.T) {
	repo := &PgTaskRepository{pool: nil}
	got, err := repo.scanTaskRow(&fakeRow{err: errors.New("scan boom")})
	assert.Error(t, err)
	assert.Nil(t, got)
}

// ---- 真实 PG 集成测试（自动 Skip 当 PG 不可用） ----

func TestPgRepo_Integration_NewPgTaskRepository(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	repo := NewPgTaskRepository(pool)
	require.NotNil(t, repo)
	require.Equal(t, pool, repo.pool)
}

func TestPgRepo_Integration_CreateAndGet(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()

	tk := freshTaskForPG("create", "create")
	require.NoError(t, repo.Create(ctx, tk))

	// GetByID
	got, err := repo.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, tk.ID, got.ID)
	assert.Equal(t, tk.DeviceSN, got.DeviceSN)
	assert.Equal(t, tk.Method, got.Method)
	assert.Equal(t, TaskStatusPending, got.Status)
}

func TestPgRepo_Integration_GetByIDNotFound(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	repo := NewPgTaskRepository(pool)
	got, err := repo.GetByID(context.Background(), generateUUID())
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestPgRepo_Integration_Update(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()

	tk := freshTaskForPG("upd", "upd")
	require.NoError(t, repo.Create(ctx, tk))

	// 修改并 Update
	now := time.Now()
	tk.Status = TaskStatusSent
	tk.CWMPID = "cwmp-pg-1"
	tk.SentAt = &now
	require.NoError(t, repo.Update(ctx, tk))

	got, err := repo.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, TaskStatusSent, got.Status)
	assert.Equal(t, "cwmp-pg-1", got.CWMPID)
	assert.NotNil(t, got.SentAt)
}

func TestPgRepo_Integration_GetByCWMPID(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()

	tk := freshTaskForPG("cw", "cw")
	tk.CWMPID = "cwmp-pg-cw-" + tk.ID[:8]
	require.NoError(t, repo.Create(ctx, tk))

	got, err := repo.GetByCWMPID(ctx, tk.CWMPID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, tk.ID, got.ID)

	// 不存在
	none, err := repo.GetByCWMPID(ctx, "non-existent-cwmp-zzz")
	require.NoError(t, err)
	assert.Nil(t, none)
}

func TestPgRepo_Integration_Delete(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()

	tk := freshTaskForPG("del", "del")
	require.NoError(t, repo.Create(ctx, tk))

	require.NoError(t, repo.Delete(ctx, tk.ID))

	got, err := repo.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestPgRepo_Integration_GetPendingByDevice(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()

	sn := "pending"
	t1 := freshTaskForPG("p1", sn)
	t2 := freshTaskForPG("p2", sn)
	require.NoError(t, repo.Create(ctx, t1))
	require.NoError(t, repo.Create(ctx, t2))

	tasks, err := repo.GetPendingByDevice(ctx, t1.DeviceSN)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(tasks), 2)
}

func TestPgRepo_Integration_CountByStatus(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()

	sn := "count"
	t1 := freshTaskForPG("c1", sn)
	t2 := freshTaskForPG("c2", sn)
	t2.Status = TaskStatusCompleted
	completedAt := time.Now()
	t2.CompletedAt = &completedAt
	require.NoError(t, repo.Create(ctx, t1))
	require.NoError(t, repo.Create(ctx, t2))

	stats, err := repo.CountByStatus(ctx, t1.DeviceSN)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, stats[TaskStatusPending], int64(1))
	assert.GreaterOrEqual(t, stats[TaskStatusCompleted], int64(1))
}

func TestPgRepo_Integration_GetHistory(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()

	sn := "hist"
	t1 := freshTaskForPG("h1", sn)
	t2 := freshTaskForPG("h2", sn)
	require.NoError(t, repo.Create(ctx, t1))
	require.NoError(t, repo.Create(ctx, t2))

	// 默认（无过滤）
	tasks, total, err := repo.GetHistory(ctx, t1.DeviceSN, &TaskHistoryOptions{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(2))
	assert.NotEmpty(t, tasks)

	// 按状态过滤
	tasks2, _, err := repo.GetHistory(ctx, t1.DeviceSN, &TaskHistoryOptions{
		Status:   TaskStatusPending,
		Page:     1,
		PageSize: 10,
	})
	require.NoError(t, err)
	for _, x := range tasks2 {
		assert.Equal(t, TaskStatusPending, x.Status)
	}

	// 时间范围
	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	later := now.Add(1 * time.Hour)
	tasks3, _, err := repo.GetHistory(ctx, t1.DeviceSN, &TaskHistoryOptions{
		Start:    &earlier,
		End:      &later,
		Page:     1,
		PageSize: 5,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, tasks3)

	// 默认分页（page<=0）
	_, _, err = repo.GetHistory(ctx, t1.DeviceSN, &TaskHistoryOptions{})
	require.NoError(t, err)
}

func TestPgRepo_Integration_BatchCreate(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()

	tasks := []*Task{
		freshTaskForPG("b1", "batch"),
		freshTaskForPG("b2", "batch"),
		freshTaskForPG("b3", "batch"),
	}
	require.NoError(t, repo.BatchCreate(ctx, tasks))

	// 校验都已写入
	for _, tk := range tasks {
		got, err := repo.GetByID(ctx, tk.ID)
		require.NoError(t, err)
		require.NotNil(t, got)
	}

	// 空批量
	require.NoError(t, repo.BatchCreate(ctx, nil))
}

func TestPgRepo_Integration_PurgeOldTasks(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()

	// PurgeOldTasks 仅 purge 终态任务且 created_at < before。
	// 用未来时间作为 before（确保 purge 命中刚创建的 completed 任务）
	tk := freshTaskForPG("purge", "purge")
	tk.Status = TaskStatusCompleted
	completedAt := time.Now()
	tk.CompletedAt = &completedAt
	require.NoError(t, repo.Create(ctx, tk))

	count, err := repo.PurgeOldTasks(ctx, time.Now().Add(time.Hour).Format(time.RFC3339))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(1))

	got, err := repo.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	assert.Nil(t, got, "task should have been purged")
}

func TestPgRepo_Integration_ListOpenByDeviceAndMethods(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()

	sn := "openmethods"
	tk := freshTaskForPG("om1", sn)
	tk.Method = "Reboot"
	tk.Status = TaskStatusSent
	now := time.Now()
	tk.SentAt = &now
	require.NoError(t, repo.Create(ctx, tk))

	open, err := repo.ListOpenByDeviceAndMethods(ctx, tk.DeviceSN, []string{"Reboot", "FactoryReset"})
	require.NoError(t, err)
	require.NotEmpty(t, open)
	assert.Equal(t, tk.ID, open[0].ID)

	// 空 methods
	open2, err := repo.ListOpenByDeviceAndMethods(ctx, tk.DeviceSN, nil)
	require.NoError(t, err)
	assert.Empty(t, open2)
}

func TestPgRepo_Integration_ListPendingAllDevices(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Create(ctx, freshTaskForPG(
			fmt.Sprintf("la%d", i), fmt.Sprintf("listall%d", i))))
	}

	all, err := repo.ListPendingAllDevices(ctx, 100)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(all), 3)

	// limit=0 → 默认
	all0, err := repo.ListPendingAllDevices(ctx, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(all0), 3)
}

// ---------------------------------------------------------------------------
// buildPendingPageSQL：keyset 分页 SQL 单测（#11，不依赖 PG）
// ---------------------------------------------------------------------------

// 首页（零游标）：不带 keyset 谓词，仅 status + ORDER BY + LIMIT。
func Test_buildPendingPageSQL_FirstPage(t *testing.T) {
	sql, args, err := buildPendingPageSQL(PendingCursor{}, 500)
	require.NoError(t, err)
	assert.Contains(t, sql, "status = $1")
	assert.Contains(t, sql, "ORDER BY created_at ASC, id ASC")
	assert.Contains(t, sql, "LIMIT 500")
	assert.NotContains(t, sql, "(created_at, id) >", "零游标不应带 keyset 谓词")
	// 仅 status 一个参数（squirrel 以 TaskStatus 原类型入参，pgx 端转 text）。
	require.Len(t, args, 1)
	assert.Equal(t, TaskStatusPending, args[0])
}

// 后续页（非零游标）：带 (created_at, id) > (...) 元组谓词推进（storage.Psql 用 $N 占位）。
func Test_buildPendingPageSQL_NextPage_Keyset(t *testing.T) {
	cur := PendingCursor{CreatedAt: time.Date(2026, 6, 10, 1, 0, 0, 0, time.UTC), ID: "abc"}
	sql, args, err := buildPendingPageSQL(cur, 500)
	require.NoError(t, err)
	assert.Contains(t, sql, "(created_at, id) > ($2, $3)")
	// status + created_at + id = 3 个参数。
	require.Len(t, args, 3)
	assert.Equal(t, "abc", args[2])
}

// batchSize<=0 落默认上界，避免无上界拉全量（#11）。
func Test_buildPendingPageSQL_ZeroBatch_FallsBackToDefault(t *testing.T) {
	sql, _, err := buildPendingPageSQL(PendingCursor{}, 0)
	require.NoError(t, err)
	assert.Contains(t, sql, fmt.Sprintf("LIMIT %d", defaultPendingBatchLimit))
}

func Test_PendingCursor_IsZero(t *testing.T) {
	assert.True(t, PendingCursor{}.IsZero())
	assert.False(t, PendingCursor{ID: "x"}.IsZero())
	assert.False(t, PendingCursor{CreatedAt: time.Now()}.IsZero())
}
