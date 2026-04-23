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
	taskDetailTTL  = 24 * time.Hour // 任务详情 TTL
	cwmpMappingTTL = 24 * time.Hour // CWMP 映射 TTL
)

// RedisTaskQueue 实现 TaskQueue 接口
type RedisTaskQueue struct {
	client redis.UniversalClient
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

// Push 推送任务到队列
// 使用 Sorted Set，score = priority * 1e13 + timestamp 纳秒
func (q *RedisTaskQueue) Push(ctx context.Context, task *Task) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}

	// 计算 score：优先级越小越优先，同优先级按时间先进先出
	// score = priority * 1e13 + timestamp纳秒（保证同优先级先入先出）
	score := float64(task.Priority)*1e13 + float64(task.CreatedAt.UnixNano())

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

	// 获取 score 最小的元素（最高优先级）
	results, err := q.client.ZRangeWithScores(ctx, queueKey, 0, 0).Result()
	if err != nil {
		return nil, fmt.Errorf("zrange: %w", err)
	}

	if len(results) == 0 {
		return nil, nil // 队列为空
	}

	taskID := results[0].Member.(string)

	// 原子移除并获取任务详情
	pipe := q.client.Pipeline()

	// 从队列移除
	pipe.ZRem(ctx, queueKey, taskID)

	// 获取任务详情
	taskDataCmd := pipe.HGet(ctx, q.taskKey(taskID), "data")

	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("pop task: %w", err)
	}

	taskData, err := taskDataCmd.Result()
	if err != nil {
		// 任务详情不存在，可能是数据不一致
		return nil, fmt.Errorf("get task data: %w", err)
	}

	var task Task
	if err := json.Unmarshal([]byte(taskData), &task); err != nil {
		return nil, fmt.Errorf("unmarshal task: %w", err)
	}

	return &task, nil
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
		score := float64(task.Priority)*1e13 + float64(task.CreatedAt.UnixNano())
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

	var staleTasks []*Task
	staleThreshold := time.Now().Add(-duration)

	for _, taskID := range taskIDs {
		task, err := q.GetByID(ctx, taskID)
		if err != nil {
			continue
		}

		// 检查是否是 sent 状态且超过阈值时间
		if task != nil && task.Status == TaskStatusSent && task.SentAt != nil {
			if task.SentAt.Before(staleThreshold) {
				staleTasks = append(staleTasks, task)
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
		return fmt.Errorf("task not found: %s", taskID)
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
		return fmt.Errorf("task not found: %s", taskID)
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
	task, err := q.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	task.MarkFailed(errorCode, errorMsg)

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
