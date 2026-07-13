package task

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/redis/go-redis/v9"
)

const (
	// #168：24h→4h，与命令实际生命周期匹配，抑制 Redis 工作集增长（2.36G 撑爆事故的增长真因
	// 之一就是 24h 任务态累积）。被淘汰/过期的任务态以 PG 双写为权威源，可重建。
	taskDetailTTL     = 4 * time.Hour // 任务详情 TTL
	cwmpMappingTTL    = 4 * time.Hour // CWMP ID → Task ID 映射 TTL
	queuePeekLimit    = 32
	queuePopScanLimit = 256
	queueScoreFactor  = 1e13
)

// RedisTaskQueue 实现 TaskQueue 接口
type RedisTaskQueue struct {
	client redis.UniversalClient
}

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
	return &RedisTaskQueue{client: client}
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
	pipe.HSet(ctx, q.taskKey(task.ID), "data", taskData)
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
		// 获取 score 较小的一批元素，跳过尚未到 next_attempt_at 的延迟重试任务。
		results, err := q.client.ZRangeWithScores(ctx, queueKey, 0, queuePeekLimit-1).Result()
		if err != nil {
			return nil, fmt.Errorf("zrange: %w", err)
		}

		if len(results) == 0 {
			return nil, nil // 队列为空
		}

		taskIDs := make([]string, 0, len(results))
		for _, result := range results {
			taskID, ok := result.Member.(string)
			if !ok {
				continue
			}
			taskIDs = append(taskIDs, taskID)
		}
		if len(taskIDs) == 0 {
			return nil, nil
		}

		// 批量获取任务详情，选出第一个已到可出队时间的任务。
		pipe := q.client.Pipeline()
		cmds := make([]*redis.StringCmd, len(taskIDs))
		for i, taskID := range taskIDs {
			cmds[i] = pipe.HGet(ctx, q.taskKey(taskID), "data")
		}
		if _, err = pipe.Exec(ctx); err != nil && err != redis.Nil {
			return nil, fmt.Errorf("pop task: %w", err)
		}

		now := time.Now()
		removedStale := 0
		for i, cmd := range cmds {
			taskData, err := cmd.Result()
			if err != nil {
				if err == redis.Nil {
					if remErr := q.client.ZRem(ctx, queueKey, taskIDs[i]).Err(); remErr != nil {
						return nil, fmt.Errorf("remove stale queue member: %w", remErr)
					}
					removedStale++
					continue
				}
				return nil, fmt.Errorf("get task data: %w", err)
			}

			var task Task
			if err := json.Unmarshal([]byte(taskData), &task); err != nil {
				if remErr := q.client.ZRem(ctx, queueKey, taskIDs[i]).Err(); remErr != nil {
					return nil, fmt.Errorf("remove malformed queue member: %w", remErr)
				}
				removedStale++
				continue
			}
			if !task.IsReadyForAttempt(now) {
				continue
			}

			if err := q.client.ZRem(ctx, queueKey, taskIDs[i]).Err(); err != nil {
				return nil, fmt.Errorf("remove popped task: %w", err)
			}
			return &task, nil
		}

		scanned += len(taskIDs)
		if removedStale == 0 {
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
	taskData, err := q.client.HGet(ctx, q.taskKey(taskID), "data").Result()
	if err == redis.Nil {
		return nil, nil // 任务不存在
	}
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	var task Task
	if err := json.Unmarshal([]byte(taskData), &task); err != nil {
		return nil, fmt.Errorf("unmarshal task: %w", err)
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

	taskData, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}

	// 更新任务详情
	err = q.client.HSet(ctx, q.taskKey(task.ID), "data", taskData).Err()
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	// 如果任务状态变回 pending，需要重新入队
	if task.Status == TaskStatusPending {
		score := queueScore(task)
		err = q.client.ZAdd(ctx, q.queueKey(task.DeviceSN), redis.Z{
			Score:  score,
			Member: task.ID,
		}).Err()
		if err != nil {
			return fmt.Errorf("requeue task: %w", err)
		}
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
	task, err := q.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return ErrTaskNotFound
	}

	// 更新任务状态
	task.MarkSent(cwmpID)

	// 使用 pipeline 原子更新
	pipe := q.client.Pipeline()

	taskData, _ := json.Marshal(task)
	pipe.HSet(ctx, q.taskKey(taskID), "data", taskData)
	pipe.Set(ctx, q.cwmpKey(cwmpID), taskID, cwmpMappingTTL)

	_, err = pipe.Exec(ctx)
	return err
}

// MarkTaskCompleted 标记任务完成
func (q *RedisTaskQueue) MarkTaskCompleted(ctx context.Context, taskID string, result json.RawMessage) error {
	task, err := q.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return ErrTaskNotFound
	}

	task.MarkCompleted(result)

	// 更新任务详情
	taskData, _ := json.Marshal(task)
	if err := q.client.HSet(ctx, q.taskKey(taskID), "data", taskData).Err(); err != nil {
		return err
	}

	// 删除 CWMP 映射
	if task.CWMPID != "" {
		q.DeleteCWMPIDMapping(ctx, task.CWMPID)
	}

	return nil
}

// MarkTaskFailed 标记任务失败
func (q *RedisTaskQueue) MarkTaskFailed(ctx context.Context, taskID string, errorCode int, errorMsg string) error {
	return q.MarkTaskFailedWithResult(ctx, taskID, errorCode, errorMsg, nil)
}

// MarkTaskFailedWithResult 标记任务失败并附带结构化 result（如 SetParameterValuesFault 详情）。
// result 为空时等价于 MarkTaskFailed。
func (q *RedisTaskQueue) MarkTaskFailedWithResult(ctx context.Context, taskID string, errorCode int, errorMsg string, result json.RawMessage) error {
	task, err := q.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return ErrTaskNotFound
	}

	task.MarkFailedWithResult(errorCode, errorMsg, result)

	// 更新任务详情
	taskData, _ := json.Marshal(task)
	if err := q.client.HSet(ctx, q.taskKey(taskID), "data", taskData).Err(); err != nil {
		return err
	}

	// 删除 CWMP 映射
	if task.CWMPID != "" {
		q.DeleteCWMPIDMapping(ctx, task.CWMPID)
	}

	return nil
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
