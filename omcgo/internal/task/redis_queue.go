package task

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/redis/go-redis/v9"
)

const (
	// #168：24h→4h，与命令实际生命周期匹配，抑制 Redis 工作集增长（2.36G 撑爆事故的增长真因
	// 之一就是 24h 任务态累积）。被淘汰/过期的任务态以 PG 双写为权威源，可重建。
	taskDetailTTL          = 4 * time.Hour // 活跃任务详情 TTL
	cwmpMappingTTL         = 4 * time.Hour // CWMP ID → Task ID 映射 TTL
	defaultTerminalTaskTTL = 15 * time.Minute
	minimumTerminalTaskTTL = 10 * time.Minute
	queuePeekLimit         = 32
	queuePopScanLimit      = 256
	queueScoreFactor       = 1e13
)

// RedisTaskQueue 实现 TaskQueue 接口
type RedisTaskQueue struct {
	client            redis.UniversalClient
	terminalTTL       time.Duration
	cleanupTransition func(context.Context, *Task, string) error
}

var taskTransitionCASScript = redis.NewScript(`
if redis.call("EXISTS", KEYS[1]) == 0 and ARGV[5] ~= "1" then
	return -1
end
if redis.call("HEXISTS", KEYS[1], "transition_token") == 1 then
	return 0
end
local current = redis.call("HGET", KEYS[1], "status")
if current == "completed" or current == "failed" or
   current == "expired" or current == "cancelled" then
	if ARGV[6] ~= "1" then
		return 0
	end
end
if current and current ~= ARGV[1] then
	return 0
end
redis.call("HSET", KEYS[1],
	"data", ARGV[2],
	"status", ARGV[3],
	"transition_from", ARGV[1],
	"transition_old_cwmp", ARGV[4],
	"transition_token", ARGV[7],
	"transition_ttl_ms", ARGV[8],
	"pg_sync_pending", "1",
	"cleanup_pending", "1",
	"event_pending", ARGV[9],
	"transition_terminal", ARGV[10])
redis.call("PERSIST", KEYS[1])
return 1
`)

var finalizeTaskTransitionScript = redis.NewScript(`
if redis.call("EXISTS", KEYS[1]) == 0 then
	return -1
end
if redis.call("HGET", KEYS[1], "transition_token") ~= ARGV[1] then
	return -2
end
if redis.call("HGET", KEYS[1], "pg_sync_pending") ~= "0" or
   redis.call("HGET", KEYS[1], "cleanup_pending") ~= "0" or
   redis.call("HGET", KEYS[1], "event_pending") ~= "0" then
	return 0
end
local ttl = redis.call("HGET", KEYS[1], "transition_ttl_ms")
if redis.call("HGET", KEYS[1], "transition_terminal") == "1" then
	redis.call("HDEL", KEYS[1], "data")
end
redis.call("HDEL", KEYS[1],
	"transition_from", "transition_old_cwmp",
	"transition_token", "transition_ttl_ms",
	"pg_sync_pending", "cleanup_pending", "event_pending",
	"transition_terminal")
redis.call("PEXPIRE", KEYS[1], ttl)
return 1
`)

var acknowledgeTaskTransitionFieldScript = redis.NewScript(`
if redis.call("EXISTS", KEYS[1]) == 0 then return -1 end
if redis.call("HGET", KEYS[1], "transition_token") ~= ARGV[1] then return -2 end
redis.call("HSET", KEYS[1], ARGV[2], "0")
return 1
`)

var resolveTaskTransitionScript = redis.NewScript(`
if redis.call("EXISTS", KEYS[1]) == 0 then
	return -1
end
if redis.call("HGET", KEYS[1], "transition_token") ~= ARGV[1] then return -2 end
redis.call("HSET", KEYS[1],
	"data", ARGV[2],
	"status", ARGV[3],
	"pg_sync_pending", "0")
redis.call("PERSIST", KEYS[1])
return 1
`)

var rollbackSentTransitionScript = redis.NewScript(`
if redis.call("EXISTS", KEYS[1]) == 0 then return -1 end
if redis.call("HGET", KEYS[1], "transition_token") ~= ARGV[1] then return -2 end
if redis.call("HGET", KEYS[1], "status") ~= "sent" then return -3 end
redis.call("HSET", KEYS[1],
	"data", ARGV[2],
	"status", "pending",
	"transition_from", "sent",
	"transition_old_cwmp", ARGV[3],
	"transition_ttl_ms", ARGV[4],
	"pg_sync_pending", "0",
	"cleanup_pending", "1",
	"event_pending", "0")
redis.call("PERSIST", KEYS[1])
return 1
`)

type taskTransitionError struct {
	token string
	err   error
}

func (e *taskTransitionError) Error() string { return e.err.Error() }
func (e *taskTransitionError) Unwrap() error { return e.err }

// PurgeBySourceResult summarizes a source-scoped Redis queue purge.
// Matched counts tasks whose source matched; Deleted is only incremented after
// all apply operations for a task succeed. Skipped includes non-matching and
// stale queue entries. Errors records malformed task data or failed deletes.
type PurgeBySourceResult struct {
	Matched int64 `json:"matched"`
	Deleted int64 `json:"deleted"`
	Skipped int64 `json:"skipped"`
	Errors  int64 `json:"errors"`
}

// NewRedisTaskQueue 创建 Redis 任务队列
func NewRedisTaskQueue(client redis.UniversalClient) *RedisTaskQueue {
	return NewRedisTaskQueueWithTerminalTTL(client, defaultTerminalTaskTTL)
}

// NewRedisTaskQueueWithTerminalTTL 创建带可配置终态保留时间的任务队列。
// 终态 Hash 至少保留 10 分钟，覆盖 ACS 5 分钟会话超时并留出迟到余量，
// 同时显著大于跨进程 Redis→PG 对账窗口；
// 未配置时保留 15 分钟。活跃任务仍使用 4 小时 TTL。
func NewRedisTaskQueueWithTerminalTTL(client redis.UniversalClient, terminalTTL time.Duration) *RedisTaskQueue {
	if terminalTTL <= 0 {
		terminalTTL = defaultTerminalTaskTTL
	}
	if terminalTTL < minimumTerminalTaskTTL {
		terminalTTL = minimumTerminalTaskTTL
	}
	return &RedisTaskQueue{client: client, terminalTTL: terminalTTL}
}

// queueKey 返回设备队列的 Redis key
func (q *RedisTaskQueue) queueKey(deviceSN string) string {
	return redisx.Keys.ACSTaskQueue(deviceSN)
}

// taskKey 返回任务详情的 Redis key
func (q *RedisTaskQueue) taskKey(taskID string) string {
	return redisx.Keys.ACSTaskDetail(taskID)
}

// cwmpKey 返回 CWMP ID 映射的 Redis key
func (q *RedisTaskQueue) cwmpKey(cwmpID string) string {
	return redisx.Keys.ACSCWMP2Task(CWMPIDHash(cwmpID))
}

func (q *RedisTaskQueue) pendingTransitionKey() string {
	return redisx.Keys.ACSTaskTransitionPending()
}

func queueScore(task *Task) float64 {
	return float64(task.Priority)*queueScoreFactor + float64(task.QueueTime().UnixNano())
}

// Push 推送任务到队列
// 使用 Sorted Set，score = priority * 1e13 + 可出队时间纳秒
func (q *RedisTaskQueue) Push(ctx context.Context, task *Task) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}

	// 计算 score：优先级越小越优先，同优先级按可出队时间先进先出。
	score := queueScore(task)

	// 序列化任务
	taskData, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}

	// 使用 pipeline 原子操作
	pipe := q.client.Pipeline()

	// 1. 添加到 Sorted Set
	pipe.ZAdd(ctx, q.queueKey(task.DeviceSN), redis.Z{
		Score:  score,
		Member: task.ID,
	})

	// 2. 存储任务详情
	pipe.HSet(ctx, q.taskKey(task.ID), "data", taskData, "status", string(task.Status))
	pipe.Expire(ctx, q.taskKey(task.ID), taskDetailTTL)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("push task: %w", err)
	}

	return nil
}

// Pop 弹出最高优先级任务
func (q *RedisTaskQueue) Pop(ctx context.Context, deviceSN string) (*Task, error) {
	queueKey := q.queueKey(deviceSN)

	scanned := 0
	for scanned < queuePopScanLimit {
		// 任务详情和设备队列使用不同 Redis key；在 Cluster 模式下不能放进同一个
		// Lua 脚本访问。详情可以跨 slot 批量读取，最终通过 ZREM 的返回值原子抢占：
		// 只有真正删除队列成员的调用者可以返回该任务。
		results, err := q.client.ZRangeWithScores(ctx, queueKey, 0, queuePeekLimit-1).Result()
		if err != nil {
			return nil, fmt.Errorf("zrange: %w", err)
		}
		if len(results) == 0 {
			return nil, nil
		}

		taskIDs := make([]string, 0, len(results))
		for _, result := range results {
			taskID, ok := result.Member.(string)
			if ok {
				taskIDs = append(taskIDs, taskID)
			}
		}
		if len(taskIDs) == 0 {
			return nil, nil
		}

		pipe := q.client.Pipeline()
		cmds := make([]*redis.StringCmd, len(taskIDs))
		for i, taskID := range taskIDs {
			cmds[i] = pipe.HGet(ctx, q.taskKey(taskID), "data")
		}
		if _, err = pipe.Exec(ctx); err != nil && err != redis.Nil {
			return nil, fmt.Errorf("pop task details: %w", err)
		}

		removedOrContended := false
		now := time.Now()
		for i, cmd := range cmds {
			taskData, err := cmd.Result()
			if err == redis.Nil {
				if _, remErr := q.client.ZRem(ctx, queueKey, taskIDs[i]).Result(); remErr != nil {
					return nil, fmt.Errorf("remove stale queue member: %w", remErr)
				}
				removedOrContended = true
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("get task data: %w", err)
			}

			var task Task
			if err := json.Unmarshal([]byte(taskData), &task); err != nil {
				if _, remErr := q.client.ZRem(ctx, queueKey, taskIDs[i]).Result(); remErr != nil {
					return nil, fmt.Errorf("remove malformed queue member: %w", remErr)
				}
				removedOrContended = true
				continue
			}
			if !task.IsReadyForAttempt(now) {
				continue
			}

			removed, err := q.client.ZRem(ctx, queueKey, taskIDs[i]).Result()
			if err != nil {
				return nil, fmt.Errorf("claim popped task: %w", err)
			}
			if removed == 1 {
				return &task, nil
			}
			// 另一个实例已抢到该任务；重新读取队首，不能返回同一份详情。
			removedOrContended = true
		}

		scanned += len(taskIDs)
		if !removedOrContended {
			return nil, nil
		}
	}

	return nil, nil
}

// Peek 查看队首任务（不移除）
func (q *RedisTaskQueue) Peek(ctx context.Context, deviceSN string) (*Task, error) {
	queueKey := q.queueKey(deviceSN)

	// 获取 score 最小的元素
	results, err := q.client.ZRangeWithScores(ctx, queueKey, 0, 0).Result()
	if err != nil {
		return nil, fmt.Errorf("zrange: %w", err)
	}

	if len(results) == 0 {
		return nil, nil
	}

	taskID := results[0].Member.(string)
	return q.GetByID(ctx, taskID)
}

// Len 获取队列长度
func (q *RedisTaskQueue) Len(ctx context.Context, deviceSN string) (int64, error) {
	return q.client.ZCard(ctx, q.queueKey(deviceSN)).Result()
}

// GetByID 根据 ID 获取任务详情
func (q *RedisTaskQueue) GetByID(ctx context.Context, taskID string) (*Task, error) {
	values, err := q.client.HMGet(ctx, q.taskKey(taskID), "data", "status").Result()
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if len(values) < 1 || values[0] == nil {
		if len(values) >= 2 && values[1] != nil {
			status, ok := values[1].(string)
			if ok && isTerminal(TaskStatus(status)) {
				return &Task{ID: taskID, Status: TaskStatus(status)}, nil
			}
		}
		return nil, nil // 任务不存在或非终态详情不完整
	}
	taskData, ok := values[0].(string)
	if !ok {
		return nil, fmt.Errorf("get task: unexpected data type %T", values[0])
	}

	var task Task
	if err := json.Unmarshal([]byte(taskData), &task); err != nil {
		return nil, fmt.Errorf("unmarshal task: %w", err)
	}
	if len(values) < 2 || values[1] == nil {
		// 滚动升级兼容：旧版本 Hash 只有 data 字段。先以 HSETNX 补回已序列化
		// 状态，终态 Lua fence 才能识别旧终态并拒绝迟到响应覆盖它。
		if err := q.client.HSetNX(ctx, q.taskKey(taskID), "status", string(task.Status)).Err(); err != nil {
			return nil, fmt.Errorf("backfill task status: %w", err)
		}
	}

	return &task, nil
}

// GetByCWMPID 根据 CWMP ID 获取任务
func (q *RedisTaskQueue) GetByCWMPID(ctx context.Context, cwmpID string) (*Task, error) {
	taskID, err := q.client.Get(ctx, q.cwmpKey(cwmpID)).Result()
	if err == redis.Nil {
		return nil, nil // 映射不存在
	}
	if err != nil {
		return nil, fmt.Errorf("get cwmp mapping: %w", err)
	}

	return q.GetByID(ctx, taskID)
}

// Update 更新任务
func (q *RedisTaskQueue) Update(ctx context.Context, task *Task) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}

	if isTerminal(task.Status) {
		current, getErr := q.GetByID(ctx, task.ID)
		if getErr != nil {
			return getErr
		}
		if current == nil {
			current = task
		}
		token, changed, transitionErr := q.prepareTransition(ctx, current, task, true)
		if transitionErr != nil {
			return transitionErr
		}
		if !changed {
			return fmt.Errorf("task %s: %w", task.ID, ErrTaskNotPending)
		}
		return q.acknowledgeTransition(ctx, task.ID, token)
	}

	current, err := q.GetByID(ctx, task.ID)
	if err != nil {
		return err
	}
	if current != nil && current.Status != task.Status {
		token, changed, transitionErr := q.prepareTransition(ctx, current, task, false)
		if transitionErr != nil {
			return transitionErr
		}
		if !changed {
			return fmt.Errorf("task %s: %w", task.ID, ErrTaskNotPending)
		}
		return q.acknowledgeTransition(ctx, task.ID, token)
	}

	taskData, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}

	// Keep the detail hash and executable queue membership consistent. A task is
	// executable iff it is pending; every other state must remove any stale
	// sorted-set member left by recovery, cancellation, or a concurrent terminal
	// transition.
	pipe := q.client.Pipeline()
	pipe.HSet(ctx, q.taskKey(task.ID), "data", taskData, "status", string(task.Status))
	pipe.Expire(ctx, q.taskKey(task.ID), taskDetailTTL)
	if task.Status == TaskStatusPending {
		score := queueScore(task)
		pipe.ZAdd(ctx, q.queueKey(task.DeviceSN), redis.Z{
			Score:  score,
			Member: task.ID,
		})
	} else {
		pipe.ZRem(ctx, q.queueKey(task.DeviceSN), task.ID)
	}
	if _, err = pipe.Exec(ctx); err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	return nil
}

// Delete 删除任务
func (q *RedisTaskQueue) Delete(ctx context.Context, taskID string) error {
	// 先获取任务以确定 device_sn
	task, err := q.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return nil // 已不存在
	}

	pipe := q.client.Pipeline()

	// 从队列移除
	pipe.ZRem(ctx, q.queueKey(task.DeviceSN), taskID)

	// 删除任务详情
	pipe.Del(ctx, q.taskKey(taskID))

	// 删除 CWMP 映射（如果有）
	if task.CWMPID != "" {
		pipe.Del(ctx, q.cwmpKey(task.CWMPID))
	}

	_, err = pipe.Exec(ctx)
	return err
}

// PurgeBySource removes only queued tasks whose serialized source matches
// source. Queue keys are traversed with SCAN; task details are read through a
// pipeline. dryRun reports matches without mutating any Redis key. This method
// deliberately does not use KEYS or FLUSHDB so it cannot block Redis or affect
// unrelated namespaces.
func (q *RedisTaskQueue) PurgeBySource(ctx context.Context, source TaskSource, dryRun bool) (PurgeBySourceResult, error) {
	if source == "" {
		return PurgeBySourceResult{}, fmt.Errorf("source is required")
	}

	var result PurgeBySourceResult
	err := redisx.Scan(ctx, q.client, redisx.Keys.ACSTaskQueuePattern(), func(ctx context.Context, queueKey string) error {
		ids, err := q.client.ZRange(ctx, queueKey, 0, -1).Result()
		if err != nil {
			return fmt.Errorf("scan queue %q: %w", queueKey, err)
		}
		if len(ids) == 0 {
			return nil
		}

		pipe := q.client.Pipeline()
		cmds := make([]*redis.StringCmd, len(ids))
		for i, id := range ids {
			cmds[i] = pipe.HGet(ctx, q.taskKey(id), "data")
		}
		_, execErr := pipe.Exec(ctx)
		if execErr != nil && execErr != redis.Nil {
			// Individual command errors are accounted for below where possible;
			// continue processing the other queue entries rather than broadening
			// the purge scope.
		}

		for i, cmd := range cmds {
			data, getErr := cmd.Result()
			if getErr != nil {
				if getErr == redis.Nil {
					result.Skipped++
				} else {
					result.Errors++
				}
				continue
			}

			var task Task
			if unmarshalErr := json.Unmarshal([]byte(data), &task); unmarshalErr != nil {
				result.Errors++
				continue
			}
			if task.Source != source {
				result.Skipped++
				continue
			}
			result.Matched++
			if dryRun {
				continue
			}

			deletePipe := q.client.Pipeline()
			zrem := deletePipe.ZRem(ctx, queueKey, ids[i])
			delDetail := deletePipe.Del(ctx, q.taskKey(ids[i]))
			var delCWMP *redis.IntCmd
			if task.CWMPID != "" {
				delCWMP = deletePipe.Del(ctx, q.cwmpKey(task.CWMPID))
			}
			_, deleteErr := deletePipe.Exec(ctx)
			if deleteErr != nil || zrem.Err() != nil || delDetail.Err() != nil || (delCWMP != nil && delCWMP.Err() != nil) {
				result.Errors++
				continue
			}
			result.Deleted++
		}
		return nil
	})
	if err != nil {
		return result, err
	}
	return result, nil
}

// Exists 判断某任务是否已经在该设备的任务队列 Sorted Set 内。
// 基于 ZScore：存在返回 true；不存在（redis.Nil）返回 false。用于启动恢复时去重。
func (q *RedisTaskQueue) Exists(ctx context.Context, deviceSN, taskID string) (bool, error) {
	_, err := q.client.ZScore(ctx, q.queueKey(deviceSN), taskID).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("zscore: %w", err)
	}
	return true, nil
}

// GetStaleSentTasks 获取 sent 状态超过指定时间的任务
func (q *RedisTaskQueue) GetStaleSentTasks(ctx context.Context, deviceSN string, staleDuration string) ([]*Task, error) {
	// 解析过期时间
	duration, err := time.ParseDuration(staleDuration)
	if err != nil {
		duration = 5 * time.Minute
	}

	// 获取所有任务 ID
	taskIDs, err := q.client.ZRange(ctx, q.queueKey(deviceSN), 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("zrange: %w", err)
	}
	if len(taskIDs) == 0 {
		return nil, nil
	}

	// 批量取任务详情：用单次 pipeline 把 N 个 HGET 合并为一个 RTT，
	// 替代原先逐 ID 调用 GetByID 的 N+1 往返（消除 #16 报告的 1 ZRANGE + N HGET）。
	pipe := q.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(taskIDs))
	for i, taskID := range taskIDs {
		cmds[i] = pipe.HGet(ctx, q.taskKey(taskID), "data")
	}
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		// redis.Nil 表示部分 key 缺失（任务详情已 TTL 过期），属正常情况，逐条处理；
		// 其它错误才视为失败。
		return nil, fmt.Errorf("pipeline hget: %w", err)
	}

	var staleTasks []*Task
	staleThreshold := time.Now().Add(-duration)

	for _, cmd := range cmds {
		taskData, err := cmd.Result()
		if err != nil {
			// key 缺失 / 单条读取失败：跳过（与原逐条 GetByID 容错语义一致）。
			continue
		}

		var task Task
		if err := json.Unmarshal([]byte(taskData), &task); err != nil {
			continue
		}

		// 检查是否是 sent 状态且超过阈值时间
		if task.Status == TaskStatusSent && task.SentAt != nil {
			if task.SentAt.Before(staleThreshold) {
				staleTasks = append(staleTasks, &task)
			}
		}
	}

	return staleTasks, nil
}

// SetCWMPIDMapping 设置 CWMP ID 到 Task ID 的映射
func (q *RedisTaskQueue) SetCWMPIDMapping(ctx context.Context, cwmpID, taskID string) error {
	return q.client.Set(ctx, q.cwmpKey(cwmpID), taskID, cwmpMappingTTL).Err()
}

// DeleteCWMPIDMapping 删除 CWMP ID 映射
func (q *RedisTaskQueue) DeleteCWMPIDMapping(ctx context.Context, cwmpID string) error {
	return q.client.Del(ctx, q.cwmpKey(cwmpID)).Err()
}

// MarkTaskSent 标记任务已发送（供 ACS Handler 调用）
func (q *RedisTaskQueue) MarkTaskSent(ctx context.Context, taskID, cwmpID string) error {
	current, err := q.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if current == nil {
		return ErrTaskNotFound
	}
	if current.Status != TaskStatusPending {
		return ErrTaskNotPending
	}
	task := cloneTaskSnapshot(current)
	task.MarkSent(cwmpID)
	token, changed, err := q.prepareTransition(ctx, current, task, false)
	if err != nil {
		return err
	}
	if !changed {
		return ErrTaskNotPending
	}
	if err := q.acknowledgeTransition(ctx, taskID, token); err != nil {
		return &taskTransitionError{token: token, err: err}
	}
	return nil
}

// MarkTaskCompleted 标记任务完成
func (q *RedisTaskQueue) MarkTaskCompleted(ctx context.Context, taskID string, result json.RawMessage) error {
	token, changed, err := q.markTaskCompleted(ctx, taskID, result)
	if err != nil || !changed {
		return err
	}
	return q.acknowledgeTransition(ctx, taskID, token)
}

func (q *RedisTaskQueue) markTaskCompleted(ctx context.Context, taskID string, result json.RawMessage) (string, bool, error) {
	task, err := q.GetByID(ctx, taskID)
	if err != nil {
		return "", false, err
	}
	if task == nil {
		return "", false, ErrTaskNotFound
	}

	current := *task
	task.MarkCompleted(result)
	return q.prepareTransition(ctx, &current, task, false)
}

// MarkTaskFailed 标记任务失败
func (q *RedisTaskQueue) MarkTaskFailed(ctx context.Context, taskID string, errorCode int, errorMsg string) error {
	return q.MarkTaskFailedWithResult(ctx, taskID, errorCode, errorMsg, nil)
}

// MarkTaskFailedWithResult 标记任务失败并附带结构化 result（如 SetParameterValuesFault 详情）。
// result 为空时等价于 MarkTaskFailed。
func (q *RedisTaskQueue) MarkTaskFailedWithResult(ctx context.Context, taskID string, errorCode int, errorMsg string, result json.RawMessage) error {
	token, changed, err := q.markTaskFailedWithResult(ctx, taskID, errorCode, errorMsg, result)
	if err != nil || !changed {
		return err
	}
	return q.acknowledgeTransition(ctx, taskID, token)
}

func (q *RedisTaskQueue) markTaskFailedWithResult(ctx context.Context, taskID string, errorCode int, errorMsg string, result json.RawMessage) (string, bool, error) {
	task, err := q.GetByID(ctx, taskID)
	if err != nil {
		return "", false, err
	}
	if task == nil {
		return "", false, ErrTaskNotFound
	}

	current := *task
	task.MarkFailedWithResult(errorCode, errorMsg, result)
	return q.prepareTransition(ctx, &current, task, false)
}

// prepareTransition atomically fences a task state transition in the detail
// hash and makes the hash persistent until PostgreSQL and cross-slot indexes
// have both acknowledged the transition.
func (q *RedisTaskQueue) prepareTransition(
	ctx context.Context,
	current, candidate *Task,
	materializeMissing bool,
) (string, bool, error) {
	if current == nil || candidate == nil || current.ID == "" || current.ID != candidate.ID {
		return "", false, fmt.Errorf("valid matching transition tasks required")
	}
	taskData, err := json.Marshal(candidate)
	if err != nil {
		return "", false, fmt.Errorf("marshal transitioned task: %w", err)
	}
	token := uuid.NewString()
	materializeArg := "0"
	if materializeMissing {
		materializeArg = "1"
	}
	allowTerminalRetry := "0"
	if candidate.Status == TaskStatusPending &&
		(current.Status == TaskStatusFailed || current.Status == TaskStatusExpired) {
		allowTerminalRetry = "1"
	}
	// The pending index is a different Redis Cluster slot from the task hash,
	// so no single Lua script can update both. Write the idempotent index first:
	// a stale member is safe and swept, while a winning persistent hash without
	// an index would have no bounded recovery path.
	if err := q.client.ZAddNX(ctx, q.pendingTransitionKey(), redis.Z{
		Score:  float64(time.Now().UnixNano()),
		Member: pendingTransitionMember(candidate.ID, token),
	}).Err(); err != nil {
		return "", false, fmt.Errorf("index pending task transition: %w", err)
	}
	eventPending := "0"
	terminalTransition := "0"
	transitionTTL := taskDetailTTL
	if isTerminal(candidate.Status) && (candidate.SourceID != "" || candidate.CreatorID != "") {
		eventPending = "1"
	}
	if isTerminal(candidate.Status) {
		transitionTTL = q.terminalTTL
		terminalTransition = "1"
	}
	result, err := taskTransitionCASScript.Run(
		ctx,
		q.client,
		[]string{q.taskKey(candidate.ID)},
		string(current.Status),
		taskData,
		string(candidate.Status),
		current.CWMPID,
		materializeArg,
		allowTerminalRetry,
		token,
		transitionTTL.Milliseconds(),
		eventPending,
		terminalTransition,
	).Int64()
	if err != nil {
		return "", false, fmt.Errorf("prepare task transition: %w", err)
	}
	if result < 0 {
		return "", false, ErrTaskNotFound
	}
	if result == 0 {
		return "", false, nil
	}
	return token, true, nil
}

func pendingTransitionMember(taskID, token string) string {
	return taskID + "|" + token
}

func parsePendingTransitionMember(member string) (string, string, bool) {
	idx := strings.LastIndexByte(member, '|')
	if idx <= 0 || idx == len(member)-1 {
		return "", "", false
	}
	return member[:idx], member[idx+1:], true
}

func (q *RedisTaskQueue) defaultCleanupTransition(ctx context.Context, task *Task, oldCWMPID string) error {
	pipe := q.client.Pipeline()
	if task.Status == TaskStatusPending {
		pipe.ZAdd(ctx, q.queueKey(task.DeviceSN), redis.Z{Score: queueScore(task), Member: task.ID})
	} else {
		pipe.ZRem(ctx, q.queueKey(task.DeviceSN), task.ID)
	}
	if task.Status == TaskStatusSent && task.CWMPID != "" {
		pipe.Set(ctx, q.cwmpKey(task.CWMPID), task.ID, cwmpMappingTTL)
	}
	if oldCWMPID != "" {
		pipe.Del(ctx, q.cwmpKey(oldCWMPID))
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("clean task transition indexes: %w", err)
	}
	return nil
}

func (q *RedisTaskQueue) acknowledgeTransition(ctx context.Context, taskID, token string) error {
	key := q.taskKey(taskID)
	acked, err := acknowledgeTaskTransitionFieldScript.Run(
		ctx, q.client, []string{key}, token, "pg_sync_pending",
	).Int64()
	if err != nil {
		return fmt.Errorf("ack task pg transition: %w", err)
	}
	if acked < 0 {
		return fmt.Errorf("stale task transition token")
	}
	values, err := q.client.HMGet(ctx, key, "data", "transition_old_cwmp", "transition_token").Result()
	if err != nil {
		return fmt.Errorf("load pending task transition: %w", err)
	}
	if len(values) == 0 || values[0] == nil {
		_ = q.client.ZRem(ctx, q.pendingTransitionKey(), taskID).Err()
		return ErrTaskNotFound
	}
	var task Task
	if err := json.Unmarshal([]byte(values[0].(string)), &task); err != nil {
		return fmt.Errorf("unmarshal pending task transition: %w", err)
	}
	oldCWMPID := ""
	if len(values) > 1 && values[1] != nil {
		oldCWMPID, _ = values[1].(string)
	}
	if len(values) < 3 || values[2] != token {
		return fmt.Errorf("stale task transition token")
	}
	cleanup := q.cleanupTransition
	if cleanup == nil {
		cleanup = q.defaultCleanupTransition
	}
	if err := cleanup(ctx, &task, oldCWMPID); err != nil {
		return err
	}
	acked, err = acknowledgeTaskTransitionFieldScript.Run(
		ctx, q.client, []string{key}, token, "cleanup_pending",
	).Int64()
	if err != nil {
		return fmt.Errorf("ack task index cleanup: %w", err)
	}
	if acked < 0 {
		return fmt.Errorf("stale task transition token")
	}
	finalized, err := finalizeTaskTransitionScript.Run(
		ctx, q.client, []string{key}, token,
	).Int64()
	if err != nil {
		return fmt.Errorf("finalize task transition: %w", err)
	}
	if finalized < 0 {
		_ = q.client.ZRem(ctx, q.pendingTransitionKey(), taskID).Err()
		return ErrTaskNotFound
	}
	if finalized == 1 {
		if err := q.client.ZRem(ctx, q.pendingTransitionKey(), pendingTransitionMember(taskID, token)).Err(); err != nil {
			return fmt.Errorf("remove pending task transition: %w", err)
		}
	}
	return nil
}

type pendingTransitionRef struct {
	TaskID string
	Token  string
}

func (q *RedisTaskQueue) listPendingTransitions(ctx context.Context, limit int64) ([]pendingTransitionRef, error) {
	if limit <= 0 {
		return nil, nil
	}
	members, err := q.client.ZRange(ctx, q.pendingTransitionKey(), 0, limit-1).Result()
	if err != nil {
		return nil, err
	}
	refs := make([]pendingTransitionRef, 0, len(members))
	for _, member := range members {
		taskID, token, ok := parsePendingTransitionMember(member)
		if ok {
			refs = append(refs, pendingTransitionRef{TaskID: taskID, Token: token})
		}
	}
	return refs, nil
}

func (q *RedisTaskQueue) deferPendingTransition(ctx context.Context, taskID, token string) error {
	return q.client.ZAddXX(ctx, q.pendingTransitionKey(), redis.Z{
		Score:  float64(time.Now().UnixNano()),
		Member: pendingTransitionMember(taskID, token),
	}).Err()
}

type preparedTaskTransition struct {
	Task          *Task
	From          TaskStatus
	PGSyncPending bool
	EventPending  bool
	Token         string
}

func (q *RedisTaskQueue) loadPendingTransition(
	ctx context.Context,
	taskID, token string,
) (*preparedTaskTransition, error) {
	values, err := q.client.HMGet(
		ctx, q.taskKey(taskID), "data", "transition_from", "pg_sync_pending",
		"event_pending", "transition_token",
	).Result()
	if err != nil {
		return nil, fmt.Errorf("load pending transition: %w", err)
	}
	if len(values) < 5 || values[0] == nil || values[1] == nil || values[4] != token {
		return nil, nil
	}
	data, ok := values[0].(string)
	if !ok {
		return nil, fmt.Errorf("pending transition data has type %T", values[0])
	}
	var task Task
	if err := json.Unmarshal([]byte(data), &task); err != nil {
		return nil, fmt.Errorf("unmarshal pending transition: %w", err)
	}
	from, _ := values[1].(string)
	pgPending, _ := values[2].(string)
	eventPending, _ := values[3].(string)
	return &preparedTaskTransition{
		Task:          &task,
		From:          TaskStatus(from),
		PGSyncPending: pgPending != "0",
		EventPending:  eventPending != "0",
		Token:         token,
	}, nil
}

func (q *RedisTaskQueue) removePendingTransition(ctx context.Context, taskID, token string) error {
	return q.client.ZRem(ctx, q.pendingTransitionKey(), pendingTransitionMember(taskID, token)).Err()
}

func (q *RedisTaskQueue) resolvePendingTransition(
	ctx context.Context,
	durable *Task, token string,
) error {
	if durable == nil {
		return fmt.Errorf("durable task is nil")
	}
	data, err := json.Marshal(durable)
	if err != nil {
		return fmt.Errorf("marshal durable transition state: %w", err)
	}
	result, err := resolveTaskTransitionScript.Run(
		ctx, q.client, []string{q.taskKey(durable.ID)}, token, data, string(durable.Status),
	).Int64()
	if err != nil {
		return fmt.Errorf("resolve pending transition from pg: %w", err)
	}
	if result < 0 {
		return ErrTaskNotFound
	}
	return nil
}

func (q *RedisTaskQueue) acknowledgeTransitionEvent(ctx context.Context, taskID, token string) error {
	result, err := acknowledgeTaskTransitionFieldScript.Run(
		ctx, q.client, []string{q.taskKey(taskID)}, token, "event_pending",
	).Int64()
	if err != nil {
		return fmt.Errorf("ack task transition event: %w", err)
	}
	if result < 0 {
		return fmt.Errorf("stale task transition token")
	}
	finalized, err := finalizeTaskTransitionScript.Run(
		ctx, q.client, []string{q.taskKey(taskID)}, token,
	).Int64()
	if err != nil {
		return fmt.Errorf("finalize event task transition: %w", err)
	}
	if finalized == 1 {
		return q.removePendingTransition(ctx, taskID, token)
	}
	return nil
}

// rollbackSentTransition compensates a PG sent→pending rollback while retaining
// the original transition token. A stale rollback can never overwrite a newer
// transition; cleanup failure remains in the durable pending index.
func (q *RedisTaskQueue) rollbackSentTransition(
	ctx context.Context,
	pending *Task,
	oldCWMPID, token string,
) error {
	if pending == nil || pending.Status != TaskStatusPending || token == "" {
		return fmt.Errorf("pending task and transition token are required")
	}
	data, err := json.Marshal(pending)
	if err != nil {
		return fmt.Errorf("marshal sent transition rollback: %w", err)
	}
	result, err := rollbackSentTransitionScript.Run(
		ctx, q.client, []string{q.taskKey(pending.ID)},
		token, data, oldCWMPID, taskDetailTTL.Milliseconds(),
	).Int64()
	if err != nil {
		return fmt.Errorf("rollback sent task transition: %w", err)
	}
	if result < 0 {
		return fmt.Errorf("rollback sent task transition CAS conflict: %w", ErrTaskNotPending)
	}
	return q.acknowledgeTransition(ctx, pending.ID, token)
}

// GetQueueLengths 获取所有设备的队列长度（用于监控）
func (q *RedisTaskQueue) GetQueueLengths(ctx context.Context) (map[string]int64, error) {
	// 扫描所有任务队列 key
	keys, err := q.client.Keys(ctx, redisx.Keys.ACSTaskQueuePattern()).Result()
	if err != nil {
		return nil, err
	}

	lengths := make(map[string]int64)
	prefix := redisx.Keys.ACSTaskQueuePrefix()
	for _, key := range keys {
		deviceSN := key[len(prefix):]
		length, err := q.client.ZCard(ctx, key).Result()
		if err != nil {
			continue
		}
		if length > 0 {
			lengths[deviceSN] = length
		}
	}

	return lengths, nil
}

// parseBool 辅助函数
func parseBool(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}
