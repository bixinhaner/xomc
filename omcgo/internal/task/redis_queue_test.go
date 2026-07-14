package task

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newRedisQueueWithMini 构造一个挂在 miniredis 上的 RedisTaskQueue。
func newRedisQueueWithMini(t *testing.T) (*RedisTaskQueue, *miniredis.Miniredis) {
	t.Helper()
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr()})
	return NewRedisTaskQueue(client), m
}

func newTaskForQueue(id, sn, method string) *Task {
	return &Task{
		ID:        id,
		DeviceSN:  sn,
		Method:    method,
		Params:    json.RawMessage("{}"),
		Priority:  10,
		Status:    TaskStatusPending,
		CreatedAt: time.Now(),
	}
}

// ---- TestCWMPMapping: CWMP ID ↔ Task 映射 ----

func TestCWMPMapping_SetAndGet(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	// 先 push 一个任务
	tk := newTaskForQueue("t-cwmp-1", "SN-CW", "GetParameterValues")
	require.NoError(t, q.Push(ctx, tk))

	// 设置映射
	require.NoError(t, q.SetCWMPIDMapping(ctx, "cwmp-id-1", "t-cwmp-1"))

	// 通过 CWMP ID 查询
	got, err := q.GetByCWMPID(ctx, "cwmp-id-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "t-cwmp-1", got.ID)
}

func TestCWMPMapping_DeleteMapping(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	tk := newTaskForQueue("t-cwmp-d", "SN-CW", "Reboot")
	require.NoError(t, q.Push(ctx, tk))
	require.NoError(t, q.SetCWMPIDMapping(ctx, "cwmp-id-d", "t-cwmp-d"))

	// 验证有映射
	got, err := q.GetByCWMPID(ctx, "cwmp-id-d")
	require.NoError(t, err)
	require.NotNil(t, got)

	// 删除后再查询返回 nil
	require.NoError(t, q.DeleteCWMPIDMapping(ctx, "cwmp-id-d"))
	got2, err := q.GetByCWMPID(ctx, "cwmp-id-d")
	require.NoError(t, err)
	assert.Nil(t, got2)
}

func TestCWMPMapping_NotFound(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	got, err := q.GetByCWMPID(ctx, "non-existent-cwmp")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestCWMPMapping_MarkTaskSentWritesMapping(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	tk := newTaskForQueue("t-mark-sent", "SN-MS", "Download")
	require.NoError(t, q.Push(ctx, tk))

	require.NoError(t, q.MarkTaskSent(ctx, "t-mark-sent", "cwmp-mapped-1"))

	// 通过 CWMP ID 应能反查到任务
	got, err := q.GetByCWMPID(ctx, "cwmp-mapped-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, TaskStatusSent, got.Status)
	assert.Equal(t, "cwmp-mapped-1", got.CWMPID)
}

func TestCWMPMapping_MarkCompletedDeletesMapping(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	tk := newTaskForQueue("t-cw-done", "SN-CD", "Reboot")
	require.NoError(t, q.Push(ctx, tk))
	require.NoError(t, q.MarkTaskSent(ctx, "t-cw-done", "cwmp-cd-1"))

	// 标记完成后 CWMP 映射应被删除
	require.NoError(t, q.MarkTaskCompleted(ctx, "t-cw-done", json.RawMessage(`{"ok":true}`)))

	got, err := q.GetByCWMPID(ctx, "cwmp-cd-1")
	require.NoError(t, err)
	assert.Nil(t, got, "CWMP mapping should be deleted after completion")
}

func TestCWMPMapping_MarkFailedDeletesMapping(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	tk := newTaskForQueue("t-cw-fail", "SN-CF", "Reboot")
	require.NoError(t, q.Push(ctx, tk))
	require.NoError(t, q.MarkTaskSent(ctx, "t-cw-fail", "cwmp-cf-1"))

	require.NoError(t, q.MarkTaskFailed(ctx, "t-cw-fail", 9001, "boom"))

	got, err := q.GetByCWMPID(ctx, "cwmp-cf-1")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestCWMPMapping_MarkSentTaskNotFound(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	err := q.MarkTaskSent(ctx, "non-existent", "cwmp-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
}

func TestCWMPMapping_MarkCompletedTaskNotFound(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	err := q.MarkTaskCompleted(ctx, "non-existent", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
}

func TestCWMPMapping_MarkFailedTaskNotFound(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	err := q.MarkTaskFailed(ctx, "non-existent", 1, "err")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
}

// ---- 队列基本操作 ----

func TestRedisQueue_PushNilTask(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	err := q.Push(context.Background(), nil)
	assert.Error(t, err)
}

func TestRedisTaskQueue_PurgeBySource_LeavesNonMMLTasks(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	mmlTask := newTaskForQueue("m1", "SN1", "Reboot")
	mmlTask.Source = TaskSourceMML
	apiTask := newTaskForQueue("a1", "SN1", "Reboot")
	apiTask.Source = TaskSourceAPI
	require.NoError(t, q.Push(ctx, mmlTask))
	require.NoError(t, q.Push(ctx, apiTask))

	result, err := q.PurgeBySource(ctx, TaskSourceMML, false)
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Deleted)
	remaining, err := q.GetByID(ctx, "a1")
	require.NoError(t, err)
	require.NotNil(t, remaining)
}

func TestRedisTaskQueue_PurgeBySource_DryRunDoesNotMutate(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	tk := newTaskForQueue("m-dry", "SN-DRY", "Reboot")
	tk.Source = TaskSourceMML
	tk.CWMPID = "cwmp-dry"
	require.NoError(t, q.Push(ctx, tk))
	require.NoError(t, q.SetCWMPIDMapping(ctx, tk.CWMPID, tk.ID))

	result, err := q.PurgeBySource(ctx, TaskSourceMML, true)
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Matched)
	require.Zero(t, result.Deleted)
	got, err := q.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	mapped, err := q.GetByCWMPID(ctx, tk.CWMPID)
	require.NoError(t, err)
	require.NotNil(t, mapped)
}

func TestRedisTaskQueue_PurgeBySource_DeletesCWMPMapping(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	tk := newTaskForQueue("m-apply", "SN-APPLY", "Reboot")
	tk.Source = TaskSourceMML
	tk.CWMPID = "cwmp-apply"
	require.NoError(t, q.Push(ctx, tk))
	require.NoError(t, q.SetCWMPIDMapping(ctx, tk.CWMPID, tk.ID))

	result, err := q.PurgeBySource(ctx, TaskSourceMML, false)
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Matched)
	require.Equal(t, int64(1), result.Deleted)
	got, err := q.GetByID(ctx, tk.ID)
	require.NoError(t, err)
	require.Nil(t, got)
	mapped, err := q.GetByCWMPID(ctx, tk.CWMPID)
	require.NoError(t, err)
	require.Nil(t, mapped)
}

func TestRedisQueue_PushAndPop_PriorityOrder(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	high := newTaskForQueue("t-high", "SN-P", "GetParameterValues")
	high.Priority = 1
	high.CreatedAt = time.Now().Add(time.Second)

	low := newTaskForQueue("t-low", "SN-P", "Reboot")
	low.Priority = 10
	low.CreatedAt = time.Now()

	require.NoError(t, q.Push(ctx, low))
	require.NoError(t, q.Push(ctx, high))

	// pop 应先返回高优先级（priority 数值小）
	first, err := q.Pop(ctx, "SN-P")
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.Equal(t, "t-high", first.ID)

	second, err := q.Pop(ctx, "SN-P")
	require.NoError(t, err)
	require.NotNil(t, second)
	assert.Equal(t, "t-low", second.ID)

	// 队列空
	empty, err := q.Pop(ctx, "SN-P")
	require.NoError(t, err)
	assert.Nil(t, empty)
}

func TestRedisQueue_PopSkipsFutureNextAttempt(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	nextAttempt := time.Now().Add(50 * time.Millisecond)
	delayed := newTaskForQueue("t-delayed", "SN-DELAY", "Reboot")
	delayed.NextAttemptAt = &nextAttempt

	ready := newTaskForQueue("t-ready", "SN-DELAY", "GetParameterValues")
	ready.Priority = delayed.Priority + 1

	require.NoError(t, q.Push(ctx, delayed))
	require.NoError(t, q.Push(ctx, ready))

	first, err := q.Pop(ctx, "SN-DELAY")
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.Equal(t, "t-ready", first.ID)

	second, err := q.Pop(ctx, "SN-DELAY")
	require.NoError(t, err)
	assert.Nil(t, second, "delayed task must not be popped before next_attempt_at")

	time.Sleep(70 * time.Millisecond)
	third, err := q.Pop(ctx, "SN-DELAY")
	require.NoError(t, err)
	require.NotNil(t, third)
	assert.Equal(t, "t-delayed", third.ID)
}

func TestRedisQueue_PopConcurrentSameDeviceReturnsTaskOnce(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	require.NoError(t, q.Push(ctx, newTaskForQueue("t-race", "SN-RACE", "GetParameterValues")))

	const workers = 16
	start := make(chan struct{})
	var wg sync.WaitGroup
	gotIDs := make(chan string, workers)
	errs := make(chan error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			got, err := q.Pop(ctx, "SN-RACE")
			if err != nil {
				errs <- err
				return
			}
			if got != nil {
				gotIDs <- got.ID
			}
		}()
	}

	close(start)
	wg.Wait()
	close(gotIDs)
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	var ids []string
	for id := range gotIDs {
		ids = append(ids, id)
	}
	require.Len(t, ids, 1, "concurrent Pop calls must not dispatch the same task twice")
	assert.Equal(t, "t-race", ids[0])
}

func TestRedisQueue_Peek(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	tk := newTaskForQueue("t-peek", "SN-PK", "Reboot")
	require.NoError(t, q.Push(ctx, tk))

	got, err := q.Peek(ctx, "SN-PK")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "t-peek", got.ID)

	// Peek 不移除
	length, err := q.Len(ctx, "SN-PK")
	require.NoError(t, err)
	assert.Equal(t, int64(1), length)
}

func TestRedisQueue_PeekEmptyQueue(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	got, err := q.Peek(context.Background(), "SN-EMPTY")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestRedisQueue_Len(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	require.NoError(t, q.Push(ctx, newTaskForQueue("t1", "SN-L", "Reboot")))
	require.NoError(t, q.Push(ctx, newTaskForQueue("t2", "SN-L", "Reboot")))

	length, err := q.Len(ctx, "SN-L")
	require.NoError(t, err)
	assert.Equal(t, int64(2), length)
}

func TestRedisQueue_GetByIDNotFound(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	got, err := q.GetByID(context.Background(), "ghost")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestRedisQueue_GetByIDFound(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()
	tk := newTaskForQueue("t-get", "SN-G", "Reboot")
	require.NoError(t, q.Push(ctx, tk))

	got, err := q.GetByID(ctx, "t-get")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "t-get", got.ID)
	assert.Equal(t, "Reboot", got.Method)
}

func TestRedisQueue_UpdateNilTask(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	err := q.Update(context.Background(), nil)
	assert.Error(t, err)
}

func TestRedisQueue_UpdateNonPending(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	tk := newTaskForQueue("t-upd", "SN-U", "Reboot")
	require.NoError(t, q.Push(ctx, tk))

	// 改成 sent
	tk.Status = TaskStatusSent
	tk.CWMPID = "cwmp-upd-1"
	require.NoError(t, q.Update(ctx, tk))

	got, err := q.GetByID(ctx, "t-upd")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, TaskStatusSent, got.Status)
	assert.Equal(t, "cwmp-upd-1", got.CWMPID)
}

func TestRedisQueue_UpdatePending_Requeues(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	tk := newTaskForQueue("t-rq", "SN-RQ", "Reboot")
	require.NoError(t, q.Push(ctx, tk))

	// 弹出
	popped, err := q.Pop(ctx, "SN-RQ")
	require.NoError(t, err)
	require.NotNil(t, popped)
	emptyLen, _ := q.Len(ctx, "SN-RQ")
	assert.Equal(t, int64(0), emptyLen)

	// Update with TaskStatusPending 应重新入队
	popped.Status = TaskStatusPending
	require.NoError(t, q.Update(ctx, popped))

	length, err := q.Len(ctx, "SN-RQ")
	require.NoError(t, err)
	assert.Equal(t, int64(1), length)
}

func TestRedisQueue_DeleteRemovesAllArtifacts(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	tk := newTaskForQueue("t-del", "SN-D", "Reboot")
	require.NoError(t, q.Push(ctx, tk))
	require.NoError(t, q.MarkTaskSent(ctx, "t-del", "cwmp-del-1"))

	require.NoError(t, q.Delete(ctx, "t-del"))

	// 队列、详情、CWMP 映射都没了
	got, err := q.GetByID(ctx, "t-del")
	require.NoError(t, err)
	assert.Nil(t, got)

	gotByCWMP, err := q.GetByCWMPID(ctx, "cwmp-del-1")
	require.NoError(t, err)
	assert.Nil(t, gotByCWMP)

	length, err := q.Len(ctx, "SN-D")
	require.NoError(t, err)
	assert.Equal(t, int64(0), length)
}

func TestRedisQueue_DeleteNonExistentTask(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	// 不存在的任务删除不报错
	err := q.Delete(context.Background(), "ghost")
	assert.NoError(t, err)
}

func TestRedisQueue_Exists(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	tk := newTaskForQueue("t-ex", "SN-EX", "Reboot")
	require.NoError(t, q.Push(ctx, tk))

	exists, err := q.Exists(ctx, "SN-EX", "t-ex")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = q.Exists(ctx, "SN-EX", "ghost")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRedisQueue_GetStaleSentTasks(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	now := time.Now()
	stale := time.Now().Add(-10 * time.Minute)
	fresh := time.Now().Add(-1 * time.Minute)

	staleTask := newTaskForQueue("t-stale", "SN-STA", "Reboot")
	staleTask.Status = TaskStatusSent
	staleTask.SentAt = &stale
	staleTask.CreatedAt = now

	freshTask := newTaskForQueue("t-fresh", "SN-STA", "Reboot")
	freshTask.Status = TaskStatusSent
	freshTask.SentAt = &fresh
	freshTask.CreatedAt = now

	pendingTask := newTaskForQueue("t-pending", "SN-STA", "Reboot")
	pendingTask.Status = TaskStatusPending

	require.NoError(t, q.Push(ctx, staleTask))
	require.NoError(t, q.Push(ctx, freshTask))
	require.NoError(t, q.Push(ctx, pendingTask))

	stales, err := q.GetStaleSentTasks(ctx, "SN-STA", "5m")
	require.NoError(t, err)
	require.Len(t, stales, 1)
	assert.Equal(t, "t-stale", stales[0].ID)
}

func TestRedisQueue_GetStaleSentTasks_BadDuration(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	// 错误的 duration 会 fallback 到 5m
	stale := time.Now().Add(-10 * time.Minute)
	tk := newTaskForQueue("t-st", "SN-BD", "Reboot")
	tk.Status = TaskStatusSent
	tk.SentAt = &stale
	require.NoError(t, q.Push(ctx, tk))

	stales, err := q.GetStaleSentTasks(ctx, "SN-BD", "not-a-duration")
	require.NoError(t, err)
	assert.Len(t, stales, 1)
}

func TestRedisQueue_GetStaleSentTasks_EmptyQueue(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	stales, err := q.GetStaleSentTasks(context.Background(), "SN-EMPTY", "5m")
	require.NoError(t, err)
	assert.Empty(t, stales)
}

func TestRedisQueue_Pop_RemovesMissingDetailAndContinues(t *testing.T) {
	q, m := newRedisQueueWithMini(t)
	ctx := context.Background()

	ghost := newTaskForQueue("t-ghost-pop", "SN-POP-MISS", "GetParameterValues")
	good := newTaskForQueue("t-good-pop", "SN-POP-MISS", "GetParameterValues")
	ghost.Priority = 1
	good.Priority = 2

	require.NoError(t, q.Push(ctx, ghost))
	require.NoError(t, q.Push(ctx, good))
	m.Del(q.taskKey(ghost.ID))

	got, err := q.Pop(ctx, "SN-POP-MISS")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, good.ID, got.ID)

	exists, err := q.Exists(ctx, "SN-POP-MISS", ghost.ID)
	require.NoError(t, err)
	assert.False(t, exists)
}

// TestRedisQueue_GetStaleSentTasks_MissingDetail 覆盖 #16 批量 pipeline 的部分 key 缺失分支：
// 队列里有任务 ID，但其详情 Hash 已被删（TTL 过期），pipeline 单条返回 redis.Nil，
// 应跳过该条而非整体失败，其余陈旧任务仍正常返回。
func TestRedisQueue_GetStaleSentTasks_MissingDetail(t *testing.T) {
	q, m := newRedisQueueWithMini(t)
	ctx := context.Background()

	stale := time.Now().Add(-10 * time.Minute)

	good := newTaskForQueue("t-good", "SN-MISS", "Reboot")
	good.Status = TaskStatusSent
	good.SentAt = &stale

	ghost := newTaskForQueue("t-ghost", "SN-MISS", "Reboot")
	ghost.Status = TaskStatusSent
	ghost.SentAt = &stale

	require.NoError(t, q.Push(ctx, good))
	require.NoError(t, q.Push(ctx, ghost))

	// 手动删除 ghost 的详情 Hash，模拟详情 TTL 过期但队列 member 还在。
	m.Del(q.taskKey("t-ghost"))

	stales, err := q.GetStaleSentTasks(ctx, "SN-MISS", "5m")
	require.NoError(t, err)
	require.Len(t, stales, 1)
	assert.Equal(t, "t-good", stales[0].ID)
}

// TestRedisQueue_GetStaleSentTasks_BatchMany 多任务下批量取回正确性：
// 混合 stale-sent / fresh-sent / pending，仅返回超阈值的 sent 任务。
func TestRedisQueue_GetStaleSentTasks_BatchMany(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	stale := time.Now().Add(-10 * time.Minute)
	fresh := time.Now().Add(-1 * time.Minute)

	for i := 0; i < 5; i++ {
		st := newTaskForQueue("stale-"+strconv.Itoa(i), "SN-MANY", "Reboot")
		st.Status = TaskStatusSent
		st.SentAt = &stale
		require.NoError(t, q.Push(ctx, st))

		fr := newTaskForQueue("fresh-"+strconv.Itoa(i), "SN-MANY", "Reboot")
		fr.Status = TaskStatusSent
		fr.SentAt = &fresh
		require.NoError(t, q.Push(ctx, fr))

		pd := newTaskForQueue("pend-"+strconv.Itoa(i), "SN-MANY", "Reboot")
		require.NoError(t, q.Push(ctx, pd))
	}

	stales, err := q.GetStaleSentTasks(ctx, "SN-MANY", "5m")
	require.NoError(t, err)
	require.Len(t, stales, 5)
	for _, tk := range stales {
		assert.Equal(t, TaskStatusSent, tk.Status)
		assert.True(t, tk.SentAt.Before(time.Now().Add(-5*time.Minute)))
	}
}

func TestRedisQueue_GetQueueLengths(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	ctx := context.Background()

	require.NoError(t, q.Push(ctx, newTaskForQueue("a1", "DEV-A", "Reboot")))
	require.NoError(t, q.Push(ctx, newTaskForQueue("a2", "DEV-A", "Reboot")))
	require.NoError(t, q.Push(ctx, newTaskForQueue("b1", "DEV-B", "Reboot")))

	lengths, err := q.GetQueueLengths(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), lengths["DEV-A"])
	assert.Equal(t, int64(1), lengths["DEV-B"])
}

func TestRedisQueue_GetQueueLengths_Empty(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	lengths, err := q.GetQueueLengths(context.Background())
	require.NoError(t, err)
	assert.Empty(t, lengths)
}

func TestRedisQueue_PopMissingDetails(t *testing.T) {
	q, m := newRedisQueueWithMini(t)
	ctx := context.Background()

	tk := newTaskForQueue("t-corrupt", "SN-COR", "Reboot")
	require.NoError(t, q.Push(ctx, tk))

	// 模拟数据不一致：详情 hash 被外部清空，但 sorted set 还有 ID
	m.Del("acs:task:t-corrupt")

	got, err := q.Pop(ctx, "SN-COR")
	assert.NoError(t, err)
	assert.Nil(t, got)

	exists, err := q.Exists(ctx, "SN-COR", tk.ID)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRedisQueue_GetByIDCorruptJSON(t *testing.T) {
	q, m := newRedisQueueWithMini(t)
	ctx := context.Background()

	// 直接写非 JSON 数据
	m.HSet("acs:task:bad-json", "data", "{not-json")

	got, err := q.GetByID(ctx, "bad-json")
	assert.Error(t, err)
	assert.Nil(t, got)
}

// ---- helper functions ----

func TestParseBool(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"true", true},
		{"false", false},
		{"1", true},
		{"0", false},
		{"junk", false},
		{"", false},
	}
	for _, tt := range tests {
		got := parseBool(tt.in)
		assert.Equal(t, tt.want, got, "input=%s", tt.in)
	}
}

func TestRedisQueue_KeyHelpers(t *testing.T) {
	q, _ := newRedisQueueWithMini(t)
	// 这些只是 trivial 包装，但需要被覆盖到
	assert.NotEmpty(t, q.queueKey("SN-K"))
	assert.NotEmpty(t, q.taskKey("t-K"))
	assert.NotEmpty(t, q.cwmpKey("cwmp-K"))
	assert.Contains(t, q.queueKey("SN-K"), "SN-K")
	assert.Contains(t, q.taskKey("t-K"), "t-K")
}
