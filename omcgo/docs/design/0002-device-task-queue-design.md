# 方案文档：设备任务队列系统

> 版本: 1.0
> 日期: 2026-03-21
> 状态: 待确认

---

## 1. 需求分析

### 1.1 背景

根据 `tr069_cpe_acs_exchange.md` 文档，ACS 与 CPE 的 RPC 交互需要使用唯一 ID 来关联请求和响应：

```
ID:intrnl.unset.id.GetParameterValues1772248515705.45470049
```

格式分析：
- `ID:` - 固定前缀
- `intrnl.unset.id.` - 内部标识
- `GetParameterValues` - RPC 方法名
- `1772248515705` - 时间戳（秒级）
- `.45470049` - 随机数

### 1.2 当前问题

现有 `cmdqueue` 包仅提供简单的队列功能：

```go
type Command struct {
    ID         string          // UUID
    Method     string          // RPC 方法
    Params     json.RawMessage // 参数
    Priority   int             // 优先级
    CreatedAt  time.Time
    ExpiresAt  *time.Time
    CommandKey string          // TR069 CommandKey
}
```

**缺失功能**：
1. 任务状态跟踪（pending → sent → completed/failed）
2. 任务执行历史记录
3. 持久化存储（仅 Redis，无数据库备份）
4. CPE 重连后的任务恢复机制
5. 任务结果存储

### 1.3 功能需求

| 功能 | 说明 |
|------|------|
| 任务创建 | 通过 API 创建任务，指定设备 SN、方法、参数 |
| 任务队列 | 按设备 SN 维护独立队列 |
| 状态管理 | 跟踪任务生命周期状态 |
| 持久化 | Redis（运行时） + PostgreSQL（历史记录） |
| 任务恢复 | CPE 重连后重新下发未完成任务 |
| 历史查询 | 通过 API 查询设备历史任务 |
| 结果存储 | 存储 CPE 返回的执行结果 |

---

## 2. 系统设计

### 2.1 架构概览

```
┌─────────────────────────────────────────────────────────────────────┐
│                           App Service (API)                          │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  POST /api/v1/devices/{sn}/tasks                             │    │
│  │  GET  /api/v1/devices/{sn}/tasks                             │    │
│  │  GET  /api/v1/devices/{sn}/tasks/{task_id}                   │    │
│  │  GET  /api/v1/devices/{sn}/tasks/pending                     │    │
│  └─────────────────────────────────────────────────────────────┘    │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        Task Management Service                        │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  - CreateTask(deviceSN, method, params)                       │   │
│  │  - GetPendingTasks(deviceSN)                                  │   │
│  │  - MarkTaskSent(taskID, cwmpID)                               │   │
│  │  - MarkTaskCompleted(taskID, result)                          │   │
│  │  - MarkTaskFailed(taskID, error)                              │   │
│  │  - GetTaskHistory(deviceSN, filters)                          │   │
│  └──────────────────────────────────────────────────────────────┘   │
└───────────┬───────────────────────────────────────┬─────────────────┘
            │                                       │
            ▼                                       ▼
┌───────────────────────┐               ┌─────────────────────────────┐
│   Redis (运行时队列)    │               │   PostgreSQL (持久化存储)    │
│                       │               │                             │
│  acs:taskq:{sn}       │               │  device_tasks 表            │
│  acs:task:{task_id}   │               │  device_task_results 表     │
└───────────────────────┘               └─────────────────────────────┘
            ▲
            │
            │ Pop Task
            │
┌─────────────────────────────────────────────────────────────────────┐
│                           ACS Engine                                 │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  handleEmpty():                                               │   │
│  │    1. Pop pending task from Redis queue                       │   │
│  │    2. Generate CWMP ID (ID:intrnl.unset.id.{Method}{ts}.{rand})│   │
│  │    3. Mark task as sent, store cwmp_id                        │   │
│  │    4. Build SOAP request with cwmp_id                         │   │
│  │    5. Send to CPE                                             │   │
│  └──────────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  handleRPCResponse():                                         │   │
│  │    1. Extract cwmp_id from response                           │   │
│  │    2. Find task by cwmp_id                                    │   │
│  │    3. Mark task completed/failed                              │   │
│  │    4. Store result                                            │   │
│  └──────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.2 数据模型

#### 2.2.1 任务状态

```go
type TaskStatus string

const (
    TaskStatusPending   TaskStatus = "pending"    // 等待下发
    TaskStatusSent      TaskStatus = "sent"       // 已发送给 CPE
    TaskStatusCompleted TaskStatus = "completed"  // CPE 执行成功
    TaskStatusFailed    TaskStatus = "failed"     // CPE 执行失败
    TaskStatusExpired   TaskStatus = "expired"    // 任务超时
    TaskStatusCancelled TaskStatus = "cancelled"  // 任务取消
)
```

#### 2.2.2 任务结构

```go
// Task 表示一个设备任务
type Task struct {
    // 基本信息
    ID           string          `json:"id"`            // UUID
    DeviceSN     string          `json:"device_sn"`     // 设备序列号
    Method       string          `json:"method"`        // RPC 方法名
    Params       json.RawMessage `json:"params"`        // 方法参数
    Priority     int             `json:"priority"`      // 优先级 (0=最高)

    // TR069 相关
    CommandKey   string          `json:"command_key"`   // TR069 CommandKey
    CWMPID       string          `json:"cwmp_id"`       // SOAP Header ID (如: ID:intrnl.unset.id.GetParameterValues1772248515705.45470049)

    // 状态管理
    Status       TaskStatus      `json:"status"`        // 当前状态
    RetryCount   int             `json:"retry_count"`   // 重试次数
    MaxRetries   int             `json:"max_retries"`   // 最大重试次数

    // 时间戳
    CreatedAt    time.Time       `json:"created_at"`    // 创建时间
    SentAt       *time.Time      `json:"sent_at"`       // 发送时间
    CompletedAt  *time.Time      `json:"completed_at"`  // 完成时间
    ExpiresAt    *time.Time      `json:"expires_at"`    // 过期时间

    // 结果
    Result       json.RawMessage `json:"result,omitempty"`  // CPE 返回结果
    ErrorCode    int             `json:"error_code,omitempty"`
    ErrorMessage string          `json:"error_message,omitempty"`

    // 元数据
    Source       string          `json:"source"`        // 任务来源 (api/scheduler/system)
    CreatorID    string          `json:"creator_id"`    // 创建者 ID
    Description  string          `json:"description"`   // 任务描述
}
```

#### 2.2.3 数据库 Schema

```sql
-- 设备任务表
CREATE TABLE device_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_sn       VARCHAR(64) NOT NULL,
    method          VARCHAR(64) NOT NULL,
    params          JSONB,
    priority        INTEGER DEFAULT 10,
    command_key     VARCHAR(128),
    cwmp_id         VARCHAR(256),

    status          VARCHAR(16) NOT NULL DEFAULT 'pending',
    retry_count     INTEGER DEFAULT 0,
    max_retries     INTEGER DEFAULT 3,

    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    sent_at         TIMESTAMP WITH TIME ZONE,
    completed_at    TIMESTAMP WITH TIME ZONE,
    expires_at      TIMESTAMP WITH TIME ZONE,

    result          JSONB,
    error_code      INTEGER,
    error_message   TEXT,

    source          VARCHAR(32) DEFAULT 'api',
    creator_id      VARCHAR(64),
    description     TEXT,

    CONSTRAINT fk_device FOREIGN KEY (device_sn) REFERENCES devices(serial_number) ON DELETE CASCADE
);

-- 索引
CREATE INDEX idx_device_tasks_device_sn ON device_tasks(device_sn);
CREATE INDEX idx_device_tasks_status ON device_tasks(status);
CREATE INDEX idx_device_tasks_created_at ON device_tasks(created_at);
CREATE INDEX idx_device_tasks_cwmp_id ON device_tasks(cwmp_id);
CREATE INDEX idx_device_tasks_pending ON device_tasks(device_sn, status, priority, created_at)
    WHERE status = 'pending';

-- 分区（可选，按月分区）
-- CREATE TABLE device_tasks_2026_03 PARTITION OF device_tasks
--     FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
```

### 2.3 CWMP ID 生成规则

```go
// GenerateCWMPID 生成符合规范的 CWMP ID
// 格式: ID:intrnl.unset.id.{Method}{timestamp}.{random}
// 示例: ID:intrnl.unset.id.GetParameterValues1772248515705.45470049
func GenerateCWMPID(method string) string {
    timestamp := time.Now().Unix()
    random := rand.Intn(100000000) // 8位随机数
    return fmt.Sprintf("ID:intrnl.unset.id.%s%d.%08d", method, timestamp, random)
}

// 示例输出:
// ID:intrnl.unset.id.GetParameterValues1772248515705.45470049
// ID:intrnl.unset.id.SetParameterValues1772248515706.12345678
// ID:intrnl.unset.id.Reboot1772248515707.87654321
```

### 2.4 Redis 数据结构

```
# 设备任务队列 (Sorted Set, score = priority * 1e12 + timestamp)
acs:taskq:{device_sn} → ZSET

# 任务详情 (Hash, TTL = 24h)
acs:task:{task_id} → HASH {
    id: "uuid"
    device_sn: "1202000752246FD0899"
    method: "GetParameterValues"
    params: '{"names":["Device.DeviceInfo.SoftwareVersion"]}'
    status: "pending"
    cwmp_id: ""
    created_at: "2026-03-21T03:00:00Z"
    ...
}

# CWMP ID → Task ID 映射 (String, TTL = 24h)
acs:cwmp2task:{cwmp_id_hash} → "task_uuid"
```

### 2.5 任务生命周期

```
┌─────────────────────────────────────────────────────────────────────┐
│                        任务生命周期状态机                              │
└─────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │          │  CreateTask()
    │ PENDING  │──────────────────────────────────────┐
    │          │                                      │
    └────┬─────┘                                      │
         │                                            │
         │ Pop & Send to CPE                          │
         │ (generate CWMP ID)                         │
         ▼                                            │
    ┌──────────┐                                      │
    │          │                                      │
    │   SENT   │                                      │
    │          │                                      │
    └────┬─────┘                                      │
         │                                            │
    ┌────┴────┬───────────────┐                       │
    │         │               │                       │
    ▼         ▼               ▼                       │
┌────────┐ ┌────────┐   ┌──────────┐                 │
│COMPLETE│ │ FAILED │   │ EXPIRED  │                 │
└────────┘ └────────┘   └──────────┘                 │
    │         │               │                       │
    │         │               │                       │
    │         │  Retry < Max │                       │
    │         └──────────────┼───────────────────────┤
    │                        │                       │
    │                        ▼                       │
    │                   ┌──────────┐                 │
    │                   │ PENDING  │ (重新入队)       │
    │                   └──────────┘                 │
    │                                                │
    │  持久化到 PostgreSQL                            │
    └────────────────────────────────────────────────┘
```

### 2.6 核心接口

#### 2.6.1 TaskService 接口

```go
// TaskService 任务管理服务接口
type TaskService interface {
    // CreateTask 创建新任务
    CreateTask(ctx context.Context, req *CreateTaskRequest) (*Task, error)

    // GetTask 获取任务详情
    GetTask(ctx context.Context, taskID string) (*Task, error)

    // GetPendingTasks 获取设备待处理任务
    GetPendingTasks(ctx context.Context, deviceSN string, limit int) ([]*Task, error)

    // GetTaskHistory 获取设备任务历史
    GetTaskHistory(ctx context.Context, deviceSN string, opts *TaskHistoryOptions) ([]*Task, int64, error)

    // CancelTask 取消任务
    CancelTask(ctx context.Context, taskID string) error

    // MarkTaskSent 标记任务已发送
    MarkTaskSent(ctx context.Context, taskID, cwmpID string) error

    // MarkTaskCompleted 标记任务完成
    MarkTaskCompleted(ctx context.Context, taskID string, result json.RawMessage) error

    // MarkTaskFailed 标记任务失败
    MarkTaskFailed(ctx context.Context, taskID string, errorCode int, errorMsg string) error

    // RecoverPendingTasks 恢复未完成任务（CPE 重连时调用）
    RecoverPendingTasks(ctx context.Context, deviceSN string) error
}
```

#### 2.6.2 TaskQueue 接口

```go
// TaskQueue Redis 任务队列接口
type TaskQueue interface {
    // Push 推送任务到队列
    Push(ctx context.Context, task *Task) error

    // Pop 弹出最高优先级任务
    Pop(ctx context.Context, deviceSN string) (*Task, error)

    // Peek 查看队首任务（不移除）
    Peek(ctx context.Context, deviceSN string) (*Task, error)

    // Len 获取队列长度
    Len(ctx context.Context, deviceSN string) (int64, error)

    // GetByID 根据 ID 获取任务
    GetByID(ctx context.Context, taskID string) (*Task, error)

    // GetByCWMPID 根据 CWMP ID 获取任务
    GetByCWMPID(ctx context.Context, cwmpID string) (*Task, error)

    // Update 更新任务
    Update(ctx context.Context, task *Task) error

    // Delete 删除任务
    Delete(ctx context.Context, taskID string) error
}
```

### 2.7 REST API 设计

```yaml
# 创建任务
POST /api/v1/devices/{device_sn}/tasks
Request:
  {
    "method": "GetParameterValues",
    "params": {"names": ["Device.DeviceInfo.SoftwareVersion"]},
    "priority": 10,
    "description": "查询软件版本",
    "expires_in": 3600  # 可选，秒
  }
Response:
  {
    "code": 0,
    "data": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "device_sn": "1202000752246FD0899",
      "method": "GetParameterValues",
      "status": "pending",
      "created_at": "2026-03-21T03:00:00Z"
    }
  }

# 获取待处理任务列表
GET /api/v1/devices/{device_sn}/tasks/pending
Response:
  {
    "code": 0,
    "data": {
      "tasks": [...],
      "total": 5
    }
  }

# 获取任务历史
GET /api/v1/devices/{device_sn}/tasks?status=completed&start=2026-03-01&end=2026-03-21&page=1&page_size=20
Response:
  {
    "code": 0,
    "data": {
      "tasks": [...],
      "total": 150,
      "page": 1,
      "page_size": 20
    }
  }

# 获取任务详情
GET /api/v1/devices/{device_sn}/tasks/{task_id}
Response:
  {
    "code": 0,
    "data": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "device_sn": "1202000752246FD0899",
      "method": "GetParameterValues",
      "params": {"names": ["Device.DeviceInfo.SoftwareVersion"]},
      "status": "completed",
      "cwmp_id": "ID:intrnl.unset.id.GetParameterValues1772248515705.45470049",
      "result": {"Device.DeviceInfo.SoftwareVersion": "1.5.3.2"},
      "created_at": "2026-03-21T03:00:00Z",
      "sent_at": "2026-03-21T03:00:05Z",
      "completed_at": "2026-03-21T03:00:06Z"
    }
  }

# 取消任务
DELETE /api/v1/devices/{device_sn}/tasks/{task_id}
Response:
  {
    "code": 0,
    "message": "task cancelled"
  }
```

---

## 3. 实现细节

### 3.1 目录结构

```
internal/
├── task/                      # 新增：任务管理模块
│   ├── model.go               # Task 结构定义
│   ├── service.go             # TaskService 实现
│   ├── repository.go          # TaskRepository 接口
│   ├── pg_repository.go       # PostgreSQL 实现
│   ├── queue.go               # TaskQueue 接口
│   ├── redis_queue.go         # Redis 实现
│   ├── cwmp_id.go             # CWMP ID 生成
│   └── handler.go             # HTTP Handler
│
├── acs/
│   ├── handler.go             # 改造：集成任务队列
│   └── ...
```

### 3.2 ACS Handler 集成

```go
// handleEmpty 处理空 POST，下发待处理任务
func (h *Handler) handleEmpty(w http.ResponseWriter, r *http.Request, log *zap.Logger) {
    // ... 获取 session ...

    // 从任务队列获取待处理任务
    task, err := h.taskQueue.Pop(r.Context(), session.DeviceSN)
    if err != nil {
        log.Error("pop task from queue", zap.Error(err))
        // 发送空响应关闭会话
        w.WriteHeader(http.StatusNoContent)
        return
    }

    if task == nil {
        // 无待处理任务，发送空响应
        log.Debug("no pending tasks")
        w.WriteHeader(http.StatusNoContent)
        h.completeSession(r.Context(), session)
        return
    }

    // 生成 CWMP ID
    cwmpID := task.GenerateCWMPID()

    // 标记任务已发送
    if err := h.taskQueue.MarkSent(r.Context(), task.ID, cwmpID); err != nil {
        log.Error("mark task sent", zap.Error(err))
    }

    // 构建 SOAP 请求
    soapReq, err := h.rpcDispatcher.BuildRequestWithCWMPID(task, cwmpID)
    if err != nil {
        log.Error("build soap request", zap.Error(err))
        h.taskQueue.MarkFailed(r.Context(), task.ID, 0, err.Error())
        w.WriteHeader(http.StatusNoContent)
        return
    }

    // 发送 SOAP 请求
    w.Header().Set("Content-Type", "text/xml; charset=utf-8")
    w.Write(soapReq)

    log.Info("sent RPC request",
        zap.String("task_id", task.ID),
        zap.String("cwmp_id", cwmpID),
        zap.String("method", task.Method))
}

// handleRPCResponse 处理 RPC 响应
func (h *Handler) handleRPCResponse(w http.ResponseWriter, r *http.Request, body []byte, log *zap.Logger) {
    // 解析响应，提取 CWMP ID
    cwmpID := parseCWMPIDFromResponse(body)

    // 根据 CWMP ID 查找任务
    task, err := h.taskQueue.GetByCWMPID(r.Context(), cwmpID)
    if err != nil {
        log.Warn("task not found for cwmp_id", zap.String("cwmp_id", cwmpID))
    }

    // 解析响应结果
    result, err := h.rpcDispatcher.ParseResponse(task.Method, body)
    if err != nil {
        h.taskQueue.MarkFailed(r.Context(), task.ID, 0, err.Error())
    } else {
        h.taskQueue.MarkCompleted(r.Context(), task.ID, result)
    }

    // 继续处理下一个任务或关闭会话
    // ...
}
```

### 3.3 任务恢复机制

```go
// OnDeviceConnect 设备连接时调用
func (s *taskService) OnDeviceConnect(ctx context.Context, deviceSN string) error {
    // 1. 检查 Redis 中是否有 sent 状态超过 5 分钟的任务
    // 2. 将这些任务重置为 pending 状态
    // 3. 重新入队

    tasks, err := s.queue.GetStaleSentTasks(ctx, deviceSN, 5*time.Minute)
    if err != nil {
        return err
    }

    for _, task := range tasks {
        if task.RetryCount >= task.MaxRetries {
            s.MarkFailed(ctx, task.ID, 0, "exceeded max retries")
            continue
        }

        task.Status = TaskStatusPending
        task.RetryCount++
        task.CWMPID = ""
        task.SentAt = nil

        if err := s.queue.Update(ctx, task); err != nil {
            log.Error("reset stale task", zap.Error(err), zap.String("task_id", task.ID))
            continue
        }

        // 重新入队
        if err := s.queue.Push(ctx, task); err != nil {
            log.Error("requeue task", zap.Error(err), zap.String("task_id", task.ID))
        }
    }

    return nil
}
```

---

## 4. 实施计划

### 4.1 阶段划分

| 阶段 | 内容 | 工作量 | 依赖 |
|------|------|--------|------|
| P0 | 数据库表创建 | 0.5 天 | - |
| P0 | Task 模型 + CWMP ID 生成 | 1 天 | - |
| P0 | Redis TaskQueue 实现 | 2 天 | P0 |
| P0 | TaskService 实现 | 2 天 | P0 |
| P0 | ACS Handler 集成 | 1 天 | P0 |
| P1 | REST API 实现 | 1 天 | P0 |
| P1 | 任务恢复机制 | 1 天 | P0 |
| P2 | 管理界面 | 2 天 | P1 |

**总计**: 约 10.5 天

### 4.2 里程碑

- **M1**: 核心功能完成，可通过 API 创建任务，ACS 可下发任务
- **M2**: 完整 API + 任务恢复机制
- **M3**: 管理界面 + 监控指标

---

## 5. 监控与告警

### 5.1 Prometheus 指标

```go
var (
    taskCreatedTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "omc_task_created_total",
            Help: "Total number of tasks created",
        },
        []string{"device_sn", "method"},
    )

    taskCompletedTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "omc_task_completed_total",
            Help: "Total number of tasks completed",
        },
        []string{"device_sn", "method", "status"},
    )

    taskQueueLength = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "omc_task_queue_length",
            Help: "Current number of pending tasks in queue",
        },
        []string{"device_sn"},
    )

    taskDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "omc_task_duration_seconds",
            Help:    "Task execution duration",
            Buckets: []float64{.1, .5, 1, 5, 10, 30, 60, 300},
        },
        []string{"method"},
    )
)
```

### 5.2 告警规则

```yaml
# 待处理任务积压
- alert: TaskQueueBacklog
  expr: omc_task_queue_length > 100
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Task queue backlog for device {{ $labels.device_sn }}"

# 任务失败率过高
- alert: HighTaskFailureRate
  expr: rate(omc_task_completed_total{status="failed"}[5m]) / rate(omc_task_completed_total[5m]) > 0.1
  for: 5m
  labels:
    severity: warning
```

---

## 6. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| Redis 故障导致任务丢失 | 高 | PostgreSQL 持久化 + 启动时恢复 |
| CPE 长时间不响应 | 中 | 任务超时机制 + 自动重试 |
| 高并发下队列性能 | 中 | Redis Pipeline + 分片 |
| 历史数据膨胀 | 低 | 按月分区 + 定期归档 |

---

## 7. 总结

本方案实现了一个完整的设备任务队列系统，具备以下特点：

1. **标准 CWMP ID** - 符合 TR069 规范的 ID 格式
2. **状态跟踪** - 完整的任务生命周期管理
3. **双重存储** - Redis 高性能队列 + PostgreSQL 持久化
4. **任务恢复** - CPE 重连后自动恢复未完成任务
5. **REST API** - 完整的管理 API
6. **监控告警** - Prometheus 指标 + 告警规则

---

## 附录：迁移计划

### 从现有 cmdqueue 迁移

现有 `cmdqueue.Command` 可直接映射到新的 `Task` 结构：

```go
// 迁移函数
func MigrateCommandToTask(cmd *cmdqueue.Command) *Task {
    return &Task{
        ID:          cmd.ID,
        DeviceSN:    "", // 需要从上下文获取
        Method:      cmd.Method,
        Params:      cmd.Params,
        Priority:    cmd.Priority,
        CommandKey:  cmd.CommandKey,
        Status:      TaskStatusPending,
        CreatedAt:   cmd.CreatedAt,
        ExpiresAt:   cmd.ExpiresAt,
        MaxRetries:  3,
        Source:      "migrated",
    }
}
```

迁移步骤：
1. 创建新表和 Redis 结构
2. 双写：同时写入旧 cmdqueue 和新 task queue
3. 逐步切换读取到新队列
4. 下线旧 cmdqueue
