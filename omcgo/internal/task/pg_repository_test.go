package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/storage"
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

func requireDeviceTaskLocationsTable(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(),
		"SELECT to_regclass('public.device_task_locations') IS NOT NULL",
	).Scan(&exists); err != nil || !exists {
		t.Skipf("device_task_locations table not present: err=%v exists=%v", err, exists)
	}
}

func explainText(t *testing.T, pool *pgxpool.Pool, query string, args ...any) string {
	t.Helper()
	rows, err := pool.Query(context.Background(), query, args...)
	require.NoError(t, err)
	defer rows.Close()

	var plan strings.Builder
	for rows.Next() {
		var line string
		require.NoError(t, rows.Scan(&line))
		plan.WriteString(line)
		plan.WriteByte('\n')
	}
	require.NoError(t, rows.Err())
	return plan.String()
}

func deviceTaskPartitionsInPlan(plan string) map[string]struct{} {
	partitions := make(map[string]struct{})
	for i := 0; i < 16; i++ {
		name := fmt.Sprintf("device_tasks_p%02d", i)
		if strings.Contains(plan, name) {
			partitions[name] = struct{}{}
		}
	}
	return partitions
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

func TestPgRepo_TaskIdentityWhereCarriesDevicePartitionKey(t *testing.T) {
	query, args, err := storage.Psql.Select("id").
		From("device_tasks").
		Where(taskIdentityWhere("task-1", "SN-1")).
		ToSql()
	require.NoError(t, err)

	require.Contains(t, query, "id =")
	require.Contains(t, query, "device_sn =")
	require.Len(t, args, 2)
	require.Contains(t, args, "task-1")
	require.Contains(t, args, "SN-1")
}

func TestPgRepo_TaskIdentityWhereRejectsMissingDevicePartitionKey(t *testing.T) {
	query, args, err := storage.Psql.Select("id").
		From("device_tasks").
		Where(taskIdentityWhere("task-1", "")).
		ToSql()
	require.NoError(t, err)

	require.Contains(t, query, "id =")
	require.Contains(t, query, "false")
	require.NotContains(t, strings.ToLower(query), "device_sn")
	require.Equal(t, []any{"task-1"}, args)
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

func TestPgRepo_Integration_AcquireSyncGPVDeviceLockSerializesByDevice(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()
	deviceSN := testDeviceSNPrefix + "sync-lock"

	release, err := repo.AcquireSyncGPVDeviceLock(ctx, deviceSN)
	require.NoError(t, err)

	competingTx, err := pool.Begin(ctx)
	require.NoError(t, err)
	var acquired bool
	require.NoError(t, competingTx.QueryRow(ctx,
		"SELECT pg_try_advisory_xact_lock(hashtextextended($1, 0))", deviceSN,
	).Scan(&acquired))
	assert.False(t, acquired, "the same device lock must be exclusive across connections")
	require.NoError(t, competingTx.Rollback(ctx))

	release()

	afterReleaseTx, err := pool.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, afterReleaseTx.QueryRow(ctx,
		"SELECT pg_try_advisory_xact_lock(hashtextextended($1, 0))", deviceSN,
	).Scan(&acquired))
	assert.True(t, acquired, "the device lock must be available after release")
	require.NoError(t, afterReleaseTx.Rollback(ctx))
}

func TestPgRepo_Integration_AcquireCommandKeyLockSerializesByCommand(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()
	commandKey := "geofence:device:8:deactivate"
	lockKey := "device-task-command-key:" + commandKey

	release, err := repo.AcquireCommandKeyLock(ctx, commandKey)
	require.NoError(t, err)

	competingTx, err := pool.Begin(ctx)
	require.NoError(t, err)
	var acquired bool
	require.NoError(t, competingTx.QueryRow(ctx,
		"SELECT pg_try_advisory_xact_lock(hashtextextended($1, 0))", lockKey,
	).Scan(&acquired))
	assert.False(t, acquired, "the same command key lock must be exclusive")
	require.NoError(t, competingTx.Rollback(ctx))

	release()

	afterReleaseTx, err := pool.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, afterReleaseTx.QueryRow(ctx,
		"SELECT pg_try_advisory_xact_lock(hashtextextended($1, 0))", lockKey,
	).Scan(&acquired))
	assert.True(t, acquired, "the command key lock must be available after release")
	require.NoError(t, afterReleaseTx.Rollback(ctx))
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

func TestPgRepo_Integration_LocationContractRoutesTaskIdentityToSinglePartition(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()

	tk := freshTaskForPG("plan", "plan")
	require.NoError(t, repo.Create(ctx, tk))

	locatedSN, ok, err := repo.LocateDeviceSNByID(ctx, tk.ID)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, tk.DeviceSN, locatedSN)

	targetedReadPlan := explainText(t, pool,
		"EXPLAIN (COSTS OFF) SELECT id FROM device_tasks WHERE id=$1 AND device_sn=$2",
		tk.ID, tk.DeviceSN,
	)
	require.Len(t, deviceTaskPartitionsInPlan(targetedReadPlan), 1, targetedReadPlan)

	targetedUpdatePlan := explainText(t, pool,
		"EXPLAIN (COSTS OFF) UPDATE device_tasks SET cwmp_id=$3 WHERE id=$1 AND device_sn=$2 AND status='pending'",
		tk.ID, tk.DeviceSN, "cwmp-plan-check",
	)
	require.Len(t, deviceTaskPartitionsInPlan(targetedUpdatePlan), 1, targetedUpdatePlan)

	idOnlyPlan := explainText(t, pool,
		"EXPLAIN (COSTS OFF) SELECT id FROM device_tasks WHERE id=$1",
		tk.ID,
	)
	require.Greater(t, len(deviceTaskPartitionsInPlan(idOnlyPlan)), 1, idOnlyPlan)
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

func TestPgRepo_Integration_LatestSyncGPVSummaryIgnoresStaleUnfinishedTasks(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()

	deviceSN := testDeviceSNPrefix + "sync-gpv-summary"
	sourceID := generateUUID()
	base := time.Date(2026, 7, 13, 10, 2, 2, 0, time.UTC)

	makeTask := func(id, commandKey string, status TaskStatus, createdAt time.Time, completedAt *time.Time) *Task {
		return &Task{
			ID:                    generateUUID(),
			DeviceSN:              deviceSN,
			Method:                "GetParameterValues",
			Params:                json.RawMessage(`{"paths":["Device.DeviceInfo."]}`),
			Priority:              1,
			CommandKey:            commandKey,
			Status:                status,
			MaxRetries:            3,
			CreatedAt:             createdAt,
			CompletedAt:           completedAt,
			Source:                TaskSourceAPI,
			SourceID:              sourceID,
			CommandIndex:          0,
			DeviceIndex:           0,
			PathTranslationSource: id,
		}
	}

	staleCreated := base.Add(-72 * time.Hour)
	staleSent := makeTask("stale", "sync-gpv-"+deviceSN+"-0-r", TaskStatusSent, staleCreated, nil)
	require.NoError(t, repo.Create(ctx, staleSent))

	oldCompletedAt := staleCreated.Add(2 * time.Hour)
	require.NoError(t, repo.Create(ctx, makeTask("old-completed-0", "sync-gpv-"+deviceSN+"-0", TaskStatusCompleted, staleCreated, &oldCompletedAt)))
	require.NoError(t, repo.Create(ctx, makeTask("old-completed-1", "sync-gpv-"+deviceSN+"-1", TaskStatusCompleted, staleCreated.Add(1*time.Second), &oldCompletedAt)))

	firstCompletedAt := base.Add(3 * time.Second)
	lastCompletedAt := base.Add(13*time.Second + 473*time.Millisecond)
	require.NoError(t, repo.Create(ctx, makeTask("completed-0", "sync-gpv-"+deviceSN+"-0", TaskStatusCompleted, base, &firstCompletedAt)))
	require.NoError(t, repo.Create(ctx, makeTask("completed-1", "sync-gpv-"+deviceSN+"-1", TaskStatusCompleted, base.Add(1*time.Second), &lastCompletedAt)))

	summary, err := repo.LatestSyncGPVSummaryByDevice(ctx, deviceSN)
	require.NoError(t, err)
	require.NotNil(t, summary)

	assert.Equal(t, sourceID, summary.SourceID)
	assert.Equal(t, 2, summary.TaskCount)
	assert.Equal(t, 2, summary.SuccessfulCommands)
	assert.Equal(t, 0, summary.FailedCommands)
	assert.Equal(t, 1, summary.RequestedPathCount)
	assert.Equal(t, 1, summary.SuccessfulPathCount)
	assert.Equal(t, 0, summary.FailedPathCount)
	assert.Empty(t, summary.FailedPaths)
	assert.True(t, summary.FirstCreatedAt.Equal(base))
	require.NotNil(t, summary.LastCompletedAt)
	assert.True(t, summary.LastCompletedAt.Equal(lastCompletedAt))
	assert.InEpsilon(t, 13.473, summary.WallClockSeconds, 0.001)
}

func TestPgRepo_Integration_LatestSyncGPVSummaryIncludesDurableParamSync(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()

	deviceSN := testDeviceSNPrefix + "durable-param-sync-summary"
	runID := generateUUID()
	completedAt := time.Date(2026, 7, 15, 10, 0, 4, 0, time.UTC)
	tk := &Task{
		ID: generateUUID(), DeviceSN: deviceSN, Method: "GetParameterValues",
		Params:   json.RawMessage(`{"names":["Device.DeviceInfo.SerialNumber"]}`),
		Priority: 10, CommandKey: "param-sync-" + runID + "-0", Status: TaskStatusFailed,
		CreatedAt: completedAt.Add(-4 * time.Second), CompletedAt: &completedAt,
		ErrorCode: 9005, ErrorMessage: "Invalid parameter name", Source: TaskSourceParamSync,
		SourceID: runID, CommandIndex: 0,
	}
	require.NoError(t, repo.Create(ctx, tk))

	summary, err := repo.LatestSyncGPVSummaryByDevice(ctx, deviceSN)
	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.Equal(t, runID, summary.SourceID)
	assert.Equal(t, 1, summary.RequestedPathCount)
	assert.Equal(t, 0, summary.SuccessfulPathCount)
	assert.Equal(t, 1, summary.FailedPathCount)
	require.Len(t, summary.FailedPaths, 1)
	assert.Equal(t, 9005, summary.FailedPaths[0].FaultCode)
}

func TestPgRepo_Integration_LatestSyncGPVSummaryPrefersNewestRunOverLateOldCompletion(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()

	deviceSN := testDeviceSNPrefix + "sync-gpv-late-old"
	oldSourceID := generateUUID()
	newSourceID := generateUUID()
	base := time.Date(2026, 7, 13, 11, 0, 0, 0, time.UTC)

	makeTask := func(sourceID, commandKey string, createdAt time.Time, completedAt time.Time) *Task {
		return &Task{
			ID:           generateUUID(),
			DeviceSN:     deviceSN,
			Method:       "GetParameterValues",
			Params:       json.RawMessage(`{"paths":["Device.DeviceInfo."]}`),
			Priority:     1,
			CommandKey:   commandKey,
			Status:       TaskStatusCompleted,
			MaxRetries:   3,
			CreatedAt:    createdAt,
			CompletedAt:  &completedAt,
			Source:       TaskSourceAPI,
			SourceID:     sourceID,
			CommandIndex: 0,
			DeviceIndex:  0,
		}
	}

	require.NoError(t, repo.Create(ctx, makeTask(
		oldSourceID,
		"sync-gpv-"+deviceSN+"-0",
		base,
		base.Add(40*time.Minute),
	)))
	require.NoError(t, repo.Create(ctx, makeTask(
		newSourceID,
		"sync-gpv-"+deviceSN+"-0",
		base.Add(10*time.Minute),
		base.Add(10*time.Minute+3*time.Second),
	)))

	summary, err := repo.LatestSyncGPVSummaryByDevice(ctx, deviceSN)
	require.NoError(t, err)
	require.NotNil(t, summary)

	assert.Equal(t, newSourceID, summary.SourceID)
	assert.True(t, summary.FirstCreatedAt.Equal(base.Add(10*time.Minute)))
	require.NotNil(t, summary.LastCompletedAt)
	assert.True(t, summary.LastCompletedAt.Equal(base.Add(10*time.Minute+3*time.Second)))
	assert.InEpsilon(t, 3, summary.WallClockSeconds, 0.001)
}

func TestPgRepo_Integration_LatestSyncGPVSummaryCountsLatestRunRetries(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()

	deviceSN := testDeviceSNPrefix + "sync-gpv-retry-run"
	sourceID := generateUUID()
	base := time.Date(2026, 7, 13, 14, 45, 38, 0, time.UTC)

	makeTask := func(id, commandKey string, createdAt time.Time, completedAt time.Time) *Task {
		return &Task{
			ID:                    generateUUID(),
			DeviceSN:              deviceSN,
			Method:                "GetParameterValues",
			Params:                json.RawMessage(`{"paths":["Device.DeviceInfo."]}`),
			Priority:              1,
			CommandKey:            commandKey,
			Status:                TaskStatusCompleted,
			MaxRetries:            3,
			CreatedAt:             createdAt,
			CompletedAt:           &completedAt,
			Source:                TaskSourceAPI,
			SourceID:              sourceID,
			CommandIndex:          0,
			DeviceIndex:           0,
			PathTranslationSource: id,
		}
	}

	oldBase := base.Add(-4 * time.Hour)
	require.NoError(t, repo.Create(ctx, makeTask("old-0", "sync-gpv-"+deviceSN+"-0", oldBase, oldBase.Add(5*time.Second))))
	require.NoError(t, repo.Create(ctx, makeTask("old-1", "sync-gpv-"+deviceSN+"-1", oldBase.Add(time.Second), oldBase.Add(4*time.Minute))))

	require.NoError(t, repo.Create(ctx, makeTask("new-0", "sync-gpv-"+deviceSN+"-0", base, base.Add(3*time.Second))))
	require.NoError(t, repo.Create(ctx, makeTask("new-0-r", "sync-gpv-"+deviceSN+"-0-r", base.Add(3*time.Second), base.Add(6*time.Second))))
	require.NoError(t, repo.Create(ctx, makeTask("new-1", "sync-gpv-"+deviceSN+"-1", base.Add(time.Second), base.Add(139*time.Second))))

	summary, err := repo.LatestSyncGPVSummaryByDevice(ctx, deviceSN)
	require.NoError(t, err)
	require.NotNil(t, summary)

	assert.Equal(t, sourceID, summary.SourceID)
	assert.Equal(t, 3, summary.TaskCount)
	assert.Equal(t, 3, summary.SuccessfulCommands)
	assert.Equal(t, 0, summary.FailedCommands)
	assert.Equal(t, 1, summary.RequestedPathCount)
	assert.Equal(t, 1, summary.SuccessfulPathCount)
	assert.Equal(t, 0, summary.FailedPathCount)
	assert.Empty(t, summary.FailedPaths)
	assert.True(t, summary.FirstCreatedAt.Equal(base))
	require.NotNil(t, summary.LastCompletedAt)
	assert.True(t, summary.LastCompletedAt.Equal(base.Add(139*time.Second)))
	assert.InEpsilon(t, 139, summary.WallClockSeconds, 0.001)
}

func TestPgRepo_Integration_LatestSyncGPVSummaryReportsRecovered9005Path(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()

	deviceSN := testDeviceSNPrefix + "sync-gpv-recovered-fault"
	sourceID := generateUUID()
	base := time.Date(2026, 7, 13, 16, 0, 0, 0, time.UTC)

	makeTask := func(commandKey string, names string, createdAt time.Time, completedAt time.Time, result json.RawMessage) *Task {
		return &Task{
			ID:           generateUUID(),
			DeviceSN:     deviceSN,
			Method:       "GetParameterValues",
			Params:       json.RawMessage(`{"names":` + names + `}`),
			Priority:     1,
			CommandKey:   commandKey,
			Status:       TaskStatusCompleted,
			MaxRetries:   3,
			CreatedAt:    createdAt,
			CompletedAt:  &completedAt,
			Result:       result,
			Source:       TaskSourceSystem,
			SourceID:     sourceID,
			CommandIndex: 0,
			DeviceIndex:  0,
		}
	}

	require.NoError(t, repo.Create(ctx, makeTask(
		"sync-gpv-"+deviceSN+"-0",
		`["DeviceGSM.NriNullDel","DeviceGSM.Mcc"]`,
		base,
		base.Add(2*time.Second),
		json.RawMessage(`{"recovered":true,"bad_path":"DeviceGSM.NriNullDel","fault_code":9005,"remaining_cnt":1}`),
	)))
	require.NoError(t, repo.Create(ctx, makeTask(
		"sync-gpv-"+deviceSN+"-0-r",
		`["DeviceGSM.Mcc"]`,
		base.Add(2*time.Second),
		base.Add(5*time.Second),
		nil,
	)))

	summary, err := repo.LatestSyncGPVSummaryByDevice(ctx, deviceSN)
	require.NoError(t, err)
	require.NotNil(t, summary)

	assert.Equal(t, sourceID, summary.SourceID)
	assert.Equal(t, 2, summary.TaskCount)
	assert.Equal(t, 2, summary.SuccessfulCommands, "recovered GPV 仍是完成的 command，不阻断其它 path")
	assert.Equal(t, 0, summary.FailedCommands)
	assert.Equal(t, 2, summary.RequestedPathCount)
	assert.Equal(t, 1, summary.SuccessfulPathCount)
	assert.Equal(t, 1, summary.FailedPathCount)
	require.Len(t, summary.FailedPaths, 1)
	assert.Equal(t, "DeviceGSM.NriNullDel", summary.FailedPaths[0].Path)
	assert.Equal(t, 9005, summary.FailedPaths[0].FaultCode)
	assert.Equal(t, "sync-gpv-"+deviceSN+"-0", summary.FailedPaths[0].CommandKey)
	assert.Equal(t, string(TaskStatusCompleted), summary.FailedPaths[0].Status)
}

func TestPgRepo_Integration_LatestSyncGPVSummaryCountsEveryPathInFailedBatch(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()

	deviceSN := testDeviceSNPrefix + "sync-gpv-failed-batch"
	sourceID := generateUUID()
	base := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	completedAt := base.Add(4 * time.Second)
	task := &Task{
		ID:           generateUUID(),
		DeviceSN:     deviceSN,
		Method:       "GetParameterValues",
		Params:       json.RawMessage(`{"names":["Device.A","Device.B","Device.C"]}`),
		Priority:     1,
		CommandKey:   "sync-gpv-" + deviceSN + "-0",
		Status:       TaskStatusFailed,
		MaxRetries:   3,
		CreatedAt:    base,
		CompletedAt:  &completedAt,
		ErrorCode:    9002,
		ErrorMessage: "internal error",
		Source:       TaskSourceSystem,
		SourceID:     sourceID,
	}
	require.NoError(t, repo.Create(ctx, task))

	summary, err := repo.LatestSyncGPVSummaryByDevice(ctx, deviceSN)
	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.Equal(t, 3, summary.RequestedPathCount)
	assert.Equal(t, 0, summary.SuccessfulPathCount)
	assert.Equal(t, 3, summary.FailedPathCount)
	require.Len(t, summary.FailedPaths, 3)
	assert.ElementsMatch(t, []string{"Device.A", "Device.B", "Device.C"}, []string{
		summary.FailedPaths[0].Path,
		summary.FailedPaths[1].Path,
		summary.FailedPaths[2].Path,
	})
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

func TestPgRepo_MarkSentIfPendingRejectsExpiredTask(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()

	tk := freshTaskForPG("expired-send-fence", "expired-send-fence")
	expiredAt := time.Now().Add(-time.Second)
	tk.ExpiresAt = &expiredAt
	require.NoError(t, repo.Create(ctx, tk))

	acquired, err := repo.MarkSentIfPending(ctx, tk.ID, "cwmp-expired-send", time.Now())

	require.NoError(t, err)
	assert.False(t, acquired, "绝对截止时间已过的 pending 任务不得再下发")
	got, err := repo.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, TaskStatusPending, got.Status)
	assert.Nil(t, got.SentAt)
}

func TestPgRepo_ListActiveTasksAfterBuildsStableKeysetQuery(t *testing.T) {
	olderThan := time.Date(2026, 7, 31, 2, 0, 0, 0, time.UTC)
	after := &ActiveTaskCursor{
		CreatedAt: time.Date(2026, 7, 31, 1, 0, 0, 0, time.UTC),
		ID:        "00000000-0000-4000-8000-000000000123",
	}

	query, args, err := buildListActiveTasksSQL(olderThan, after, 100)
	require.NoError(t, err)

	assert.Contains(t, query, "status IN")
	assert.Contains(t, query, "created_at <")
	assert.Contains(t, query, "(created_at, id) >")
	assert.Contains(t, query, "ORDER BY created_at ASC, id ASC")
	assert.Contains(t, query, "LIMIT 100")
	require.Contains(t, args, olderThan)
	require.Contains(t, args, after.CreatedAt)
	require.Contains(t, args, after.ID)
}

func TestPgRepo_ListExpiredCandidatesProtectsFreshInFlightTask(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()
	now := time.Now()
	expiredAt := now.Add(-time.Minute)

	pending := freshTaskForPG("expired-pending", "expired-pending")
	pending.ExpiresAt = &expiredAt
	require.NoError(t, repo.Create(ctx, pending))

	freshSent := freshTaskForPG("fresh-inflight", "fresh-inflight")
	freshSent.Status = TaskStatusSent
	freshSent.ExpiresAt = &expiredAt
	freshSentAt := now.Add(-sentTaskExpiryGrace / 2)
	freshSent.SentAt = &freshSentAt
	require.NoError(t, repo.Create(ctx, freshSent))

	staleSent := freshTaskForPG("stale-inflight", "stale-inflight")
	staleSent.Status = TaskStatusSent
	staleSent.ExpiresAt = &expiredAt
	staleSentAt := now.Add(-sentTaskExpiryGrace - time.Second)
	staleSent.SentAt = &staleSentAt
	require.NoError(t, repo.Create(ctx, staleSent))

	candidates, err := repo.ListExpiredCandidates(ctx, now, 1000)
	require.NoError(t, err)
	candidateIDs := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidateIDs[candidate.ID] = struct{}{}
	}

	assert.Contains(t, candidateIDs, pending.ID, "尚未下发且已到期的任务应立即清理")
	assert.NotContains(t, candidateIDs, freshSent.ID, "刚下发的RPC应获得有界响应保护")
	assert.Contains(t, candidateIDs, staleSent.ID, "超过在途保护窗口的任务仍必须终结")
}

func TestPgRepo_Integration_UpdateDoesNotDowngradeTerminalTask(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()

	tk := freshTaskForPG("term", "term")
	require.NoError(t, repo.Create(ctx, tk))

	completedAt := time.Now()
	completed := *tk
	completed.Status = TaskStatusCompleted
	completed.CompletedAt = &completedAt
	completed.Result = json.RawMessage(`{"ok":true}`)
	require.NoError(t, repo.Update(ctx, &completed))

	sentAt := completedAt.Add(-10 * time.Millisecond)
	staleSent := *tk
	staleSent.Status = TaskStatusSent
	staleSent.CWMPID = "late-cwmp"
	staleSent.SentAt = &sentAt
	staleSent.CompletedAt = nil
	staleSent.Result = nil
	require.NoError(t, repo.Update(ctx, &staleSent))

	got, err := repo.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, TaskStatusCompleted, got.Status)
	assert.NotNil(t, got.CompletedAt)
	assert.JSONEq(t, `{"ok":true}`, string(got.Result))
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

	locatedSN, ok, err := repo.LocateDeviceSNByID(ctx, tk.ID)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Empty(t, locatedSN)

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

func TestPgRepo_Integration_LatestCompletedByDeviceCommandKey(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()

	older := freshTaskForPG("completed-old", "completed-key")
	older.CommandKey = "pm_upload_setup_on_online"
	older.Status = TaskStatusCompleted
	olderCompletedAt := time.Now().Add(-time.Minute)
	older.CompletedAt = &olderCompletedAt
	require.NoError(t, repo.Create(ctx, older))

	newer := freshTaskForPG("completed-new", "completed-key")
	newer.CommandKey = older.CommandKey
	newer.Status = TaskStatusCompleted
	newer.Params = json.RawMessage(`{"version":"new"}`)
	newerCompletedAt := time.Now()
	newer.CompletedAt = &newerCompletedAt
	require.NoError(t, repo.Create(ctx, newer))

	got, err := repo.LatestCompletedByDeviceCommandKey(
		ctx, newer.DeviceSN, newer.CommandKey,
	)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, newer.ID, got.ID)
	require.JSONEq(t, string(newer.Params), string(got.Params))
}

func TestPgRepo_Integration_PurgeOldTasks(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()

	// PurgeOldTasks 仅 purge 终态任务且 completed_at < before。
	// Use an old cutoff so this integration test does not delete fresh
	// device_tasks from local end-to-end runs that share the dev database.
	tk := freshTaskForPG("purge", "purge")
	tk.Status = TaskStatusCompleted
	tk.CreatedAt = time.Now().AddDate(-11, 0, 0)
	completedAt := time.Now().AddDate(-11, 0, 0)
	tk.CompletedAt = &completedAt
	require.NoError(t, repo.Create(ctx, tk))

	count, err := repo.PurgeOldTasks(ctx, time.Now().AddDate(-10, 0, 0).Format(time.RFC3339))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(1))

	got, err := repo.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	assert.Nil(t, got, "task should have been purged")
}

func TestPgRepo_Integration_LocateDeviceSNByIDRepairsMissingLocation(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	requireDeviceTaskLocationsTable(t, pool)
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()

	tk := freshTaskForPG("missing-location", "missing-location")
	require.NoError(t, repo.Create(ctx, tk))
	_, err := pool.Exec(ctx, "DELETE FROM device_task_locations WHERE task_id=$1", tk.ID)
	require.NoError(t, err)

	deviceSN, ok, err := repo.LocateDeviceSNByID(ctx, tk.ID)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, tk.DeviceSN, deviceSN)

	var repairedSN string
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT device_sn FROM device_task_locations WHERE task_id=$1",
		tk.ID,
	).Scan(&repairedSN))
	require.Equal(t, tk.DeviceSN, repairedSN)
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

// #758 收尾：HasIncompleteSyncGPVTasksByDevice 只应把近 24h 内未完成的 sync-gpv
// 任务算作阻断项；超过 24h 的历史卡死任务（如设备重连后遗留的 sent 任务）视为
// 失效，不再永久阻断 finalize。
func TestPgRepo_Integration_HasIncompleteSyncGPV_SkipsStaleTasks(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()
	sn := testDeviceSNPrefix + "SYNCGPV"

	// 造一个 48h 前创建、仍处 sent 的 sync-gpv 任务（历史卡死场景）。
	stale := freshTaskForPG("stale", "SYNCGPV")
	stale.DeviceSN = sn
	stale.Method = "GetParameterValues"
	stale.CommandKey = "sync-gpv-" + sn + "-0-r-r-r-r"
	stale.Status = TaskStatusSent
	stale.CreatedAt = time.Now().Add(-48 * time.Hour)
	require.NoError(t, repo.Create(ctx, stale))

	orphan := freshTaskForPG("orphan", "SYNCGPV")
	orphan.DeviceSN = sn
	orphan.Method = "GetParameterValues"
	orphan.CommandKey = "sync-gpv-" + sn + "-0-r-r-r-r-r"
	orphan.Status = TaskStatusSent
	orphan.CreatedAt = time.Now().Add(-2 * time.Hour)
	orphan.ExpiresAt = nil
	require.NoError(t, repo.Create(ctx, orphan))

	// 只有历史/孤儿卡死任务时，不应被算作未完成。
	has, err := repo.HasIncompleteSyncGPVTasksByDevice(ctx, sn)
	require.NoError(t, err)
	assert.False(t, has, "历史或无过期时间的卡死 sync-gpv 任务不应阻断 finalize")

	// 再造一个刚创建、处 sent 的 sync-gpv 任务（正常进行中）。
	fresh := freshTaskForPG("fresh", "SYNCGPV")
	fresh.DeviceSN = sn
	fresh.Method = "GetParameterValues"
	fresh.CommandKey = "sync-gpv-" + sn + "-1-r-r-r-r"
	fresh.Status = TaskStatusSent
	fresh.CreatedAt = time.Now()
	freshExpires := time.Now().Add(30 * time.Minute)
	fresh.ExpiresAt = &freshExpires
	require.NoError(t, repo.Create(ctx, fresh))

	// 存在近 24h 内的进行中任务时，应算作未完成。
	has, err = repo.HasIncompleteSyncGPVTasksByDevice(ctx, sn)
	require.NoError(t, err)
	assert.True(t, has, "近 24h 内进行中的 sync-gpv 任务应阻断 finalize")
}

func TestPgRepo_Integration_CountOpenSyncGPV_SkipsFailedBatchResidue(t *testing.T) {
	pool := newTestPool(t)
	if pool == nil {
		return
	}
	defer cleanupTestTasks(t, pool)
	repo := NewPgTaskRepository(pool)
	ctx := context.Background()
	sn := testDeviceSNPrefix + "SYNCGPVCOUNT"
	sourceID := uuid.New().String()
	expiresAt := time.Now().Add(30 * time.Minute)

	sentResidue := freshTaskForPG("sent-residue", "SYNCGPVCOUNT")
	sentResidue.DeviceSN = sn
	sentResidue.Method = "GetParameterValues"
	sentResidue.CommandKey = "sync-gpv-" + sn + "-0-r"
	sentResidue.SourceID = sourceID
	sentResidue.Status = TaskStatusSent
	sentResidue.CreatedAt = time.Now()
	sentResidue.ExpiresAt = &expiresAt
	require.NoError(t, repo.Create(ctx, sentResidue))

	failedSameBatch := freshTaskForPG("failed-same-batch", "SYNCGPVCOUNT")
	failedSameBatch.DeviceSN = sn
	failedSameBatch.Method = "GetParameterValues"
	failedSameBatch.CommandKey = "sync-gpv-" + sn + "-1"
	failedSameBatch.SourceID = sourceID
	failedSameBatch.Status = TaskStatusFailed
	failedSameBatch.CreatedAt = time.Now()
	failedSameBatch.ExpiresAt = &expiresAt
	require.NoError(t, repo.Create(ctx, failedSameBatch))

	count, err := repo.CountOpenSyncGPVByDevice(ctx, sn)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count, "同一同步批次已有失败终态时,残留 sent 不应让参数树一直同步中")

	hasOpen, err := repo.HasOpenSyncGPVTasksByDevice(ctx, sn)
	require.NoError(t, err)
	assert.True(t, hasOpen, "防重入口径必须仍识别 sent 残留,避免重复创建新的 sync source")
}
