# MML 脚本与任务系统共用整合方案

> 基于父子关系方案（方案 B），分析 MML 脚本与设备任务系统的共用点和整合策略

---

## 一、共用点分析

### 1.1 现有功能矩阵

| 功能模块 | MML 脚本任务 | 设备任务 | 共用可能性 |
|---------|-------------|---------|-----------|
| **脚本模板** | ✅ 需要 | ❌ 不需要 | 🔴 独立 |
| **脚本解析** | ✅ 需要 | ❌ 不需要 | 🔴 独立 |
| **任务编排** | ✅ 多命令序列 | ❌ 单命令 | 🟡 部分共用 |
| **任务拆分** | ✅ 多设备×多命令 | ❌ 单设备单命令 | 🟢 可抽象 |
| **任务调度** | ✅ 定时/周期 | ❌ 即时 | 🟢 可抽象 |
| **任务执行** | ✅ RPC 调用 | ✅ RPC 调用 | 🟢 完全共用 |
| **重试机制** | ✅ 离线/失败重试 | ✅ 基础重试 | 🟢 可增强 |
| **进度跟踪** | ✅ 汇总统计 | ✅ 单任务状态 | 🟢 可抽象 |
| **结果存储** | ✅ 批量结果 | ✅ 单个结果 | 🟡 部分共用 |
| **任务控制** | ✅ 启动/暂停/终止 | ❌ 不支持 | 🟡 可扩展 |

---

### 1.2 可共用的核心能力

#### ✅ 1.2.1 任务拆分引擎（高度共用）

**现状：**
- MML 任务需要拆分为 N（设备）× M（命令）个子任务
- 自动开站也需要拆分为多个 RPC 任务
- 设备规则应用也需要批量创建任务

**共用方案：**

```go
package task

// TaskSplitter 任务拆分器（通用）
type TaskSplitter interface {
    // SplitTask 将父任务拆分为子任务
    SplitTask(ctx context.Context, parentTaskID uuid.UUID, strategy SplitStrategy) error
}

// SplitStrategy 拆分策略
type SplitStrategy struct {
    Devices    []string           // 目标设备列表
    Commands   []CommandTemplate  // 命令模板列表
    TaskType   string             // 任务类型：mml/provisioning/rule
    Priority   int                // 优先级
    Schedule   *ScheduleConfig    // 调度配置（可选）
}

// CommandTemplate 命令模板
type CommandTemplate struct {
    Method     string                 // RPC 方法
    Params     map[string]interface{} // 参数模板
    Order      int                    // 执行顺序（用于命令序列）
    DependsOn  []int                  // 依赖的前置命令索引
}
```

**使用示例：**

```go
// MML 任务拆分
splitter.SplitTask(ctx, mmlTaskID, SplitStrategy{
    Devices:  []string{"SN001", "SN002"},
    Commands: []CommandTemplate{
        {Method: "GetParameterValues", Params: {"names": ["Device.DeviceInfo."]}, Order: 1},
        {Method: "GetParameterValues", Params: {"names": ["Device.Alarm."]}, Order: 2},
    },
    TaskType: "mml",
})

// 自动开站任务拆分
splitter.SplitTask(ctx, provTaskID, SplitStrategy{
    Devices:  []string{"SN003"},
    Commands: []CommandTemplate{
        {Method: "GetParameterValues", Params: {...}, Order: 1},
        {Method: "SetParameterValues", Params: {...}, Order: 2, DependsOn: []int{1}},
        {Method: "Reboot", Params: {...}, Order: 3, DependsOn: []int{2}},
    },
    TaskType: "provisioning",
})
```

---

#### ✅ 1.2.2 任务调度器（高度共用）

**现状：**
- MML 任务支持：立即执行、定时执行、周期执行、挂起
- 其他批量任务也可能需要定时/周期执行

**共用方案：**

```sql
-- 通用任务调度配置表（可选，也可以直接存在父任务表中）
CREATE TABLE task_schedules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_task_id  UUID NOT NULL,              -- 关联父任务
    execute_type    VARCHAR(20) NOT NULL DEFAULT 'active', -- active/scheduled/periodic/suspended
    scheduled_at    TIMESTAMPTZ,                -- 定时执行时间
    period_start    TIMESTAMPTZ,                -- 周期开始时间
    period_end      TIMESTAMPTZ,                -- 周期结束时间
    period_time     TIME,                       -- 周期执行时间点
    status          VARCHAR(20) DEFAULT 'active', -- active/paused/completed
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_task_schedules_parent ON task_schedules(parent_task_id);
CREATE INDEX idx_task_schedules_type ON task_schedules(execute_type);
CREATE INDEX idx_task_schedules_scheduled ON task_schedules(scheduled_at) 
    WHERE scheduled_at IS NOT NULL AND status = 'active';
```

```go
package scheduler

// TaskScheduler 通用任务调度器
type TaskScheduler struct {
    pool *pgxpool.Pool
}

// CheckAndExecute 检查并执行待调度的任务
func (s *TaskScheduler) CheckAndExecute(ctx context.Context) error {
    // 1. 查询到达定时时间的任务
    scheduledTasks, err := s.getScheduledTasks(ctx)
    if err != nil {
        return err
    }
    
    for _, task := range scheduledTasks {
        // 拆分并执行
        if err := s.splitAndExecute(ctx, task.ID); err != nil {
            log.Printf("Failed to execute scheduled task %s: %v", task.ID, err)
        }
    }
    
    // 2. 查询周期任务需要触发的实例
    periodicTasks, err := s.getPeriodicTasks(ctx)
    if err != nil {
        return err
    }
    
    for _, task := range periodicTasks {
        // 创建新的任务实例
        instanceID, err := s.createPeriodicInstance(ctx, task.ID)
        if err != nil {
            log.Printf("Failed to create periodic instance: %v", err)
            continue
        }
        
        // 拆分并执行
        if err := s.splitAndExecute(ctx, instanceID); err != nil {
            log.Printf("Failed to execute periodic task %s: %v", instanceID, err)
        }
    }
    
    return nil
}
```

**Worker 定时任务：**

```go
// 每分钟检查一次
func StartSchedulerWorker(ctx context.Context, scheduler *TaskScheduler) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := scheduler.CheckAndExecute(ctx); err != nil {
                log.Printf("Scheduler error: %v", err)
            }
        }
    }
}
```

---

#### ✅ 1.2.3 进度监控器（高度共用）

**现状：**
- MML 任务需要汇总统计：total_devices、success_count、failed_count
- 其他批量任务也需要类似的汇总

**共用方案：**

```go
package task

// ProgressMonitor 通用进度监控器
type ProgressMonitor struct {
    pool *pgxpool.Pool
}

// UpdateProgress 更新父任务进度
func (m *ProgressMonitor) UpdateProgress(ctx context.Context, parentTaskID uuid.UUID) error {
    // 查询子任务统计
    var stats struct {
        Total       int
        Completed   int
        Failed      int
        Pending     int
        Running     int
    }
    
    err := m.pool.QueryRow(ctx, `
        SELECT 
            COUNT(*) as total,
            COUNT(*) FILTER (WHERE status = 'completed') as completed,
            COUNT(*) FILTER (WHERE status = 'failed') as failed,
            COUNT(*) FILTER (WHERE status = 'pending') as pending,
            COUNT(*) FILTER (WHERE status = 'sent') as running
        FROM device_tasks
        WHERE parent_task_id = $1
    `, parentTaskID).Scan(&stats.Total, &stats.Completed, &stats.Failed, &stats.Pending, &stats.Running)
    if err != nil {
        return fmt.Errorf("query stats: %w", err)
    }
    
    // 判断父任务状态
    parentStatus := m.determineParentStatus(stats)
    
    // 更新父任务（支持不同类型的父任务表）
    // 使用多态更新或分别处理
    return m.updateParentTask(ctx, parentTaskID, stats, parentStatus)
}

func (m *ProgressMonitor) determineParentStatus(stats struct{...}) string {
    if stats.Completed + stats.Failed == stats.Total && stats.Total > 0 {
        if stats.Failed == 0 {
            return "completed"
        } else if stats.Completed == 0 {
            return "failed"
        } else {
            return "partial"
        }
    }
    return "running"
}
```

**事件驱动更新（优化方案）：**

```go
// 在 device_tasks 状态更新时触发进度更新
func (r *PgTaskRepository) UpdateTaskStatus(ctx context.Context, taskID string, status TaskStatus) error {
    // ... 更新 device_tasks 状态
    
    // 如果有父任务，异步更新进度
    if task.ParentTaskID != nil {
        go func() {
            monitor.UpdateProgress(context.Background(), *task.ParentTaskID)
        }()
    }
    
    return nil
}
```

---

#### ✅ 1.2.4 重试策略引擎（中度共用）

**现状：**
- MML 任务支持：离线重试、失败重试、重试次数、重试间隔
- device_tasks 支持：基础重试（retry_count、max_retries）

**共用方案：**

```sql
-- 在 device_tasks 表中增强重试字段
ALTER TABLE device_tasks 
ADD COLUMN IF NOT EXISTS retry_strategy JSONB DEFAULT '{}',
ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ;

-- retry_strategy JSONB 结构
{
    "offline_retry": true,          // 是否支持离线重试
    "offline_wait": 60,             // 离线等待时间（秒）
    "failed_retry": true,           // 是否支持失败重试
    "retry_count": 3,               // 重试次数
    "retry_interval": 300,          // 重试间隔（秒）
    "retry_reason": "offline"       // 重试原因：offline/failed/timeout
}
```

```go
package task

// RetryEngine 重试引擎
type RetryEngine struct {
    pool *pgxpool.Pool
}

// ProcessRetry 处理重试逻辑
func (e *RetryEngine) ProcessRetry(ctx context.Context) error {
    // 查询需要重试的任务
    rows, err := e.pool.Query(ctx, `
        SELECT id, device_sn, method, params, retry_strategy, retry_count
        FROM device_tasks
        WHERE status IN ('failed', 'expired')
          AND next_retry_at IS NOT NULL
          AND next_retry_at <= NOW()
          AND retry_count < (retry_strategy->>'retry_count')::int
    `)
    if err != nil {
        return err
    }
    defer rows.Close()
    
    for rows.Next() {
        var task struct {
            ID            string
            DeviceSN      string
            Method        string
            Params        json.RawMessage
            RetryStrategy json.RawMessage
            RetryCount    int
        }
        rows.Scan(&task.ID, &task.DeviceSN, &task.Method, &task.Params, 
                  &task.RetryStrategy, &task.RetryCount)
        
        // 检查重试条件
        strategy := parseRetryStrategy(task.RetryStrategy)
        if !e.shouldRetry(ctx, &task, strategy) {
            continue
        }
        
        // 重置任务状态
        _, err := e.pool.Exec(ctx, `
            UPDATE device_tasks
            SET status = 'pending',
                retry_count = retry_count + 1,
                next_retry_at = NOW() + (retry_strategy->>'retry_interval')::int * INTERVAL '1 second',
                error_code = NULL,
                error_message = NULL,
                updated_at = NOW()
            WHERE id = $1
        `, task.ID)
        if err != nil {
            log.Printf("Failed to retry task %s: %v", task.ID, err)
        }
    }
    
    return nil
}

func (e *RetryEngine) shouldRetry(ctx context.Context, task *Task, strategy RetryStrategy) bool {
    // 离线重试：检查设备是否上线
    if strategy.OfflineRetry && task.ErrorReason == "offline" {
        isOnline, _ := e.checkDeviceOnline(ctx, task.DeviceSN)
        if !isOnline {
            return false
        }
    }
    
    // 失败重试：检查重试次数
    if strategy.FailedRetry && task.RetryCount < strategy.RetryCount {
        return true
    }
    
    return false
}
```

---

#### ⚠️ 1.2.5 脚本模板引擎（低度共用）

**现状：**
- MML 脚本有独立的 `mml_scripts` 表
- 其他任务类型可能不需要脚本模板

**共用方案：**

**方案 A：保持独立（推荐）**
- `mml_scripts` 表专门用于 MML 脚本管理
- 其他任务类型如有需要，创建各自的模板表

**方案 B：通用模板表（可选）**

```sql
-- 通用任务模板表（如果需要）
CREATE TABLE task_templates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_name   VARCHAR(200) NOT NULL,
    template_type   VARCHAR(50) NOT NULL,     -- mml/provisioning/config/etc.
    content         TEXT NOT NULL,            -- 模板内容
    parameters      JSONB DEFAULT '{}',       -- 参数定义
    device_type     VARCHAR(50),              -- 适用设备类型
    tags            JSONB DEFAULT '[]',
    creator         VARCHAR(100),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_task_templates_type ON task_templates(template_type);
CREATE INDEX idx_task_templates_tags_gin ON task_templates USING GIN (tags);
```

**推荐：方案 A（保持独立）**
- MML 脚本有复杂的解析逻辑（命令码、参数映射）
- 其他任务类型的模板格式差异大
- 独立表更清晰，避免过度抽象

---

## 二、整合后的数据模型

### 2.1 完整 Schema

```sql
-- ==========================================
-- 1. MML 脚本表（独立）
-- ==========================================
CREATE TABLE mml_scripts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    script_name VARCHAR(200) NOT NULL,
    description TEXT,
    content     TEXT NOT NULL,                -- 脚本内容（每行一条 MML 命令）
    device_type VARCHAR(50),                  -- 适用设备类型
    creator     VARCHAR(100),
    tags        JSONB DEFAULT '[]',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trigger_mml_scripts_updated_at
    BEFORE UPDATE ON mml_scripts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX idx_mml_scripts_tags_gin ON mml_scripts USING GIN (tags);

-- ==========================================
-- 2. MML 任务表（父任务）
-- ==========================================
CREATE TABLE mml_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_name       VARCHAR(200),
    script_id       UUID REFERENCES mml_scripts(id) ON DELETE SET NULL,
    device_sns      JSONB NOT NULL,                 -- 目标设备 SN 列表
    commands        JSONB NOT NULL DEFAULT '[]',    -- 命令序列
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    
    -- 执行策略（共用字段）
    execute_type    VARCHAR(20) DEFAULT 'active',   -- active/scheduled/periodic/suspended
    scheduled_at    TIMESTAMPTZ,                    -- 定时执行时间
    period_start    TIMESTAMPTZ,                    -- 周期开始时间
    period_end      TIMESTAMPTZ,                    -- 周期结束时间
    period_time     TIME,                           -- 周期执行时间点
    
    -- 重试策略（共用字段）
    offline_retry   BOOLEAN DEFAULT false,
    offline_wait    INT DEFAULT 60,                 -- 秒
    failed_retry    BOOLEAN DEFAULT false,
    retry_count     INT DEFAULT 3,
    retry_interval  INT DEFAULT 5,                  -- 分钟
    
    -- 汇总统计（共用字段）
    total_devices   INT DEFAULT 0,
    success_count   INT DEFAULT 0,
    failed_count    INT DEFAULT 0,
    
    -- 时间
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    
    -- 审计
    creator         VARCHAR(100),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mml_tasks_status ON mml_tasks(status);
CREATE INDEX idx_mml_tasks_created ON mml_tasks(created_at DESC);
CREATE INDEX idx_mml_tasks_execute_type ON mml_tasks(execute_type);
CREATE INDEX idx_mml_tasks_scheduled ON mml_tasks(scheduled_at) WHERE scheduled_at IS NOT NULL;

CREATE TRIGGER trigger_mml_tasks_updated_at
    BEFORE UPDATE ON mml_tasks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ==========================================
-- 3. 设备任务表（子任务 - 扩展）
-- ==========================================
ALTER TABLE device_tasks 
ADD COLUMN IF NOT EXISTS parent_task_id UUID,         -- 关联父任务（多态）
ADD COLUMN IF NOT EXISTS parent_task_type VARCHAR(50),-- 父任务类型：mml/provisioning/rule
ADD COLUMN IF NOT EXISTS task_type VARCHAR(32) DEFAULT 'normal', -- 任务类型
ADD COLUMN IF NOT EXISTS retry_strategy JSONB DEFAULT '{}', -- 重试策略
ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ,   -- 下次重试时间
ADD COLUMN IF NOT EXISTS command_order INT,           -- 命令执行顺序
ADD COLUMN IF NOT EXISTS depends_on INT[];            -- 依赖的前置命令

CREATE INDEX idx_device_tasks_parent ON device_tasks(parent_task_id, parent_task_type);
CREATE INDEX idx_device_tasks_type ON device_tasks(task_type);
CREATE INDEX idx_device_tasks_retry ON device_tasks(next_retry_at) WHERE next_retry_at IS NOT NULL;

-- ==========================================
-- 4. 自动开站任务表（父任务 - 示例）
-- ==========================================
CREATE TABLE provisioning_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id       UUID NOT NULL,
    device_serial   VARCHAR(64) NOT NULL,
    template_id     UUID,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    
    -- 共用字段
    execute_type    VARCHAR(20) DEFAULT 'active',
    scheduled_at    TIMESTAMPTZ,
    
    -- 开站特定字段
    current_step    INT DEFAULT 0,
    total_steps     INT DEFAULT 0,
    error_message   TEXT,
    
    -- 汇总统计
    total_commands  INT DEFAULT 0,
    success_count   INT DEFAULT 0,
    failed_count    INT DEFAULT 0,
    
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_provisioning_tasks_status ON provisioning_tasks(status);

-- ==========================================
-- 5. 设备规则任务表（父任务 - 示例）
-- ==========================================
CREATE TABLE device_rule_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id         UUID NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    
    -- 共用字段
    execute_type    VARCHAR(20) DEFAULT 'active',
    scheduled_at    TIMESTAMPTZ,
    
    -- 规则特定字段
    total_devices   INT DEFAULT 0,
    matched_count   INT DEFAULT 0,
    failed_count    INT DEFAULT 0,
    
    -- 汇总统计
    success_count   INT DEFAULT 0,
    
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_device_rule_tasks_status ON device_rule_tasks(status);
```

---

### 2.2 多态关联设计

**问题：** `device_tasks.parent_task_id` 需要关联多种类型的父任务表

**解决方案：**

```sql
-- 方案 1：使用外键约束（不推荐，PostgreSQL 不支持多态外键）
-- parent_task_id UUID REFERENCES ??? 

-- 方案 2：使用组合字段 + 应用层约束（推荐）
parent_task_id UUID,
parent_task_type VARCHAR(50),  -- 'mml' / 'provisioning' / 'rule'

-- 查询示例
SELECT dt.*
FROM device_tasks dt
WHERE dt.parent_task_id = 'task-uuid'
  AND dt.parent_task_type = 'mml';
```

```go
// 应用层确保数据一致性
func (s *TaskSplitter) SplitTask(ctx context.Context, parentTaskID uuid.UUID, parentType string, strategy SplitStrategy) error {
    // 验证父任务存在
    exists, err := s.checkParentTaskExists(ctx, parentTaskID, parentType)
    if err != nil || !exists {
        return fmt.Errorf("parent task %s (%s) not found", parentTaskID, parentType)
    }
    
    // 拆分任务
    tasks := s.buildChildTasks(strategy)
    for i := range tasks {
        tasks[i].ParentTaskID = &parentTaskID
        tasks[i].ParentTaskType = &parentType
    }
    
    return s.batchInsertTasks(ctx, tasks)
}
```

---

## 三、共用服务架构

### 3.1 服务分层

```
┌─────────────────────────────────────────────────────────┐
│                    业务层（独立）                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │  MML Service │  │Provisioning  │  │ Rule Service │  │
│  │              │  │   Service    │  │              │  │
│  │ - 脚本解析   │  │ - 模板匹配   │  │ - 规则匹配   │  │
│  │ - 命令映射   │  │ - 开站流程   │  │ - 批量应用   │  │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  │
└─────────┼─────────────────┼─────────────────┼───────────┘
          │                 │                 │
┌─────────┼─────────────────┼─────────────────┼───────────┐
│         ▼       共用服务层（抽象）           ▼           │
│  ┌──────────────────────────────────────────────────┐  │
│  │          Task Splitter（任务拆分器）              │  │
│  │  - N×M 任务拆分                                  │  │
│  │  - 命令依赖分析                                  │  │
│  │  - 批量插入优化                                  │  │
│  └──────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────┐  │
│  │          Task Scheduler（任务调度器）             │  │
│  │  - 定时任务触发                                  │  │
│  │  - 周期任务实例化                                │  │
│  │  - 挂起/恢复控制                                 │  │
│  └──────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────┐  │
│  │         Progress Monitor（进度监控器）            │  │
│  │  - 子任务状态汇总                                │  │
│  │  - 父任务状态更新                                │  │
│  │  - 事件驱动更新                                  │  │
│  └──────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────┐  │
│  │           Retry Engine（重试引擎）                │  │
│  │  - 离线重试检测                                  │  │
│  │  - 失败重试调度                                  │  │
│  │  - 重试策略执行                                  │  │
│  └──────────────────────────────────────────────────┘  │
└────────────────────────┬────────────────────────────────┘
                         │
┌────────────────────────┼────────────────────────────────┐
│                        ▼    基础设施层（现有）           │
│  ┌──────────────────────────────────────────────────┐  │
│  │           device_tasks 表                        │  │
│  │           Redis 任务队列                         │  │
│  │           ACS Worker（执行引擎）                 │  │
│  └──────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────┘
```

### 3.2 代码组织

```
internal/
├── task/                           # 共用任务服务
│   ├── splitter.go                 # 任务拆分器
│   ├── scheduler.go                # 任务调度器
│   ├── monitor.go                  # 进度监控器
│   ├── retry.go                    # 重试引擎
│   ├── model.go                    # 共用数据模型
│   └── repository.go               # 共用仓储
│
├── mml/                            # MML 业务服务（独立）
│   ├── service.go                  # MML 服务
│   ├── script_parser.go            # 脚本解析器
│   ├── command_mapper.go           # 命令映射器
│   ├── handler.go                  # HTTP 处理器
│   └── model.go                    # MML 数据模型
│
├── provisioning/                   # 自动开站服务（独立）
│   ├── service.go
│   ├── template_engine.go
│   └── handler.go
│
└── device/                         # 设备管理服务
    └── rule/                       # 设备规则服务（独立）
        ├── service.go
        └── handler.go
```

---

## 四、共用场景示例

### 4.1 MML 脚本任务完整流程

```
1. 用户创建 MML 脚本任务
   POST /api/v1/mml/tasks
   {
     "task_name": "批量查询设备信息",
     "script_id": "script-uuid",
     "device_sns": ["SN001", "SN002", "SN003"],
     "execute_type": "scheduled",
     "scheduled_at": "2026-04-17T02:00:00Z",
     "offline_retry": true,
     "offline_wait": 300,
     "failed_retry": true,
     "retry_count": 3,
     "retry_interval": 5
   }

2. MML Service 处理
   - 解析脚本内容（mml_scripts.content）
   - 将 MML 命令映射为 TR-069 RPC 方法
   - 创建 mml_tasks 记录

3. Task Scheduler 调度
   - 到达定时时间后触发
   - 调用 Task Splitter 拆分任务

4. Task Splitter 拆分
   mml_task (父任务)
   ├── device_task (SN001, GetParameterValues, order=1)
   ├── device_task (SN001, GetParameterValues, order=2)
   ├── device_task (SN002, GetParameterValues, order=1)
   ├── device_task (SN002, GetParameterValues, order=2)
   ├── device_task (SN003, GetParameterValues, order=1)
   └── device_task (SN003, GetParameterValues, order=2)

5. ACS Worker 执行
   - 从 Redis 队列 pop device_task
   - 执行 TR-069 RPC 调用
   - 更新 device_task 状态
   - 触发 Progress Monitor 更新

6. Progress Monitor 汇总
   - 查询 device_tasks WHERE parent_task_id = ?
   - 统计 total/success/failed
   - 更新 mml_tasks 状态

7. Retry Engine 重试（如果需要）
   - 检测失败/超时任务
   - 检查重试条件（设备在线、重试次数）
   - 重置任务状态为 pending

8. 前端查询进度
   GET /api/v1/mml/tasks/:id
   {
     "id": "task-uuid",
     "status": "running",
     "total_devices": 3,
     "success_count": 2,
     "failed_count": 0,
     "progress": "67%"
   }
```

### 4.2 自动开站任务完整流程

```
1. 设备上线触发开站
   - ACS 接收到 Inform 事件
   - 创建 provisioning_task（父任务）

2. Task Splitter 拆分
   provisioning_task (父任务)
   ├── device_task (GetParameterValues - 设备信息)
   ├── device_task (SetParameterValues - 基础配置)
   ├── device_task (SetParameterValues - 网络配置)
   └── device_task (Reboot - 重启生效)

3. ACS Worker 顺序执行
   - 按 command_order 排序
   - 检查 depends_on 依赖
   - 逐个执行 RPC 调用

4. Progress Monitor 汇总
   - 更新 provisioning_task.current_step
   - 更新 provisioning_task.status
```

---

## 五、API 设计增强

### 5.1 通用任务管理 API

```go
// 通用任务查询（支持所有类型的父任务）
GET /api/v1/tasks?type=mml&status=running&page=1&page_size=20
GET /api/v1/tasks?type=provisioning&status=pending
GET /api/v1/tasks?type=rule&status=completed

// 通用任务进度查询
GET /api/v1/tasks/:id/progress
{
    "task_id": "uuid",
    "task_type": "mml",
    "status": "running",
    "total": 100,
    "completed": 67,
    "failed": 3,
    "progress": "67%"
}

// 通用任务详情（子任务列表）
GET /api/v1/tasks/:id/details?page=1&page_size=20
{
    "items": [
        {
            "id": "subtask-uuid",
            "device_sn": "SN001",
            "method": "GetParameterValues",
            "status": "completed",
            "command_order": 1,
            "created_at": "2026-04-16T10:00:00Z",
            "completed_at": "2026-04-16T10:00:05Z"
        }
    ],
    "total": 100,
    "page": 1,
    "page_size": 20
}

// 通用任务控制
POST /api/v1/tasks/:id/start      // 启动挂起任务
POST /api/v1/tasks/:id/pause      // 暂停任务
POST /api/v1/tasks/:id/terminate  // 终止任务
DELETE /api/v1/tasks/:id          // 删除任务
```

### 5.2 MML 特定 API

```go
// MML 脚本管理
GET    /api/v1/mml/scripts         // 脚本列表
POST   /api/v1/mml/scripts         // 创建脚本
PUT    /api/v1/mml/scripts/:id     // 更新脚本
DELETE /api/v1/mml/scripts/:id     // 删除脚本

// MML 任务管理
GET    /api/v1/mml/tasks           // 任务列表
POST   /api/v1/mml/tasks           // 创建任务
GET    /api/v1/mml/tasks/:id       // 任务详情
POST   /api/v1/mml/tasks/:id/start      // 启动任务
POST   /api/v1/mml/tasks/:id/pause      // 暂停任务
POST   /api/v1/mml/tasks/:id/terminate  // 终止任务
DELETE /api/v1/mml/tasks/:id             // 删除任务
GET    /api/v1/mml/tasks/:id/results    // 任务结果

// MML 命令执行（单设备即时执行）
POST   /api/v1/mml/execute         // 执行 MML 命令
```

---

## 六、实施计划

### Phase 1: 基础设施扩展（3-5 天）

- [ ] 扩展 `device_tasks` 表（增加共用字段）
- [ ] 创建 `mml_tasks` 表
- [ ] 创建共用服务层基础结构
  - [ ] Task Splitter 接口和实现
  - [ ] Progress Monitor 接口和实现

### Phase 2: 共用服务开发（5-7 天）

- [ ] 实现 Task Scheduler（定时/周期调度）
- [ ] 实现 Retry Engine（重试策略）
- [ ] 实现事件驱动的进度更新
- [ ] 编写共用服务的单元测试

### Phase 3: MML 业务层开发（5-7 天）

- [ ] 实现 MML 脚本解析器
- [ ] 实现 MML 命令映射器
- [ ] 实现 MML Service（调用共用服务）
- [ ] 实现 MML HTTP Handler

### Phase 4: 前端集成（3-5 天）

- [ ] MML 脚本管理页面
- [ ] MML 脚本任务页面
- [ ] 任务进度实时展示
- [ ] 任务控制按钮

### Phase 5: 测试与优化（3-5 天）

- [ ] 集成测试（完整流程）
- [ ] 性能测试（批量任务拆分）
- [ ] 异常场景测试
- [ ] 压力测试

**总工期：19-29 天（约 4-6 周）**

---

## 七、优势总结

### 7.1 代码复用率

| 模块 | 复用程度 | 说明 |
|------|---------|------|
| Task Splitter | 100% | 所有批量任务共用 |
| Task Scheduler | 100% | 所有定时/周期任务共用 |
| Progress Monitor | 100% | 所有父任务进度汇总共用 |
| Retry Engine | 100% | 所有重试策略共用 |
| device_tasks | 100% | 底层执行完全共用 |
| ACS Worker | 100% | 无需修改 |

**整体复用率：~70%**

### 7.2 维护成本降低

- **Bug 修复**：共用服务修复一次，所有任务类型受益
- **功能增强**：新增功能（如优先级队列）只需在共用服务实现
- **性能优化**：批量插入、索引优化等只需做一次

### 7.3 扩展性提升

未来新增任务类型（如批量配置、批量诊断）：
1. 创建新的父任务表（如 `config_tasks`）
2. 实现业务特定的逻辑（如配置模板解析）
3. 调用共用服务（Splitter、Scheduler、Monitor）
4. **无需修改底层执行引擎**

---

## 八、风险与注意事项

### 8.1 数据一致性

**风险：** 多态关联可能导致数据不一致

**缓解措施：**
- 应用层严格校验 `parent_task_id` 和 `parent_task_type` 的匹配
- 定期运行数据一致性检查脚本
- 使用数据库触发器记录变更日志

### 8.2 性能考虑

**风险：** 大量子任务可能影响查询性能

**缓解措施：**
- 为 `parent_task_id` + `parent_task_type` 创建复合索引
- 使用分区表（按月分区）存储历史任务
- 进度汇总使用事件驱动，避免频繁全表扫描

### 8.3 复杂度控制

**风险：** 共用服务层过度抽象，增加理解成本

**缓解措施：**
- 保持共用服务接口简洁
- 编写详细的文档和示例
- 业务层保持独立，避免过度耦合

---

## 九、总结

### 推荐方案：父子关系 + 共用服务层 ✅

**核心架构：**
```
业务层（独立） → 共用服务层（抽象） → 基础设施层（现有）
```

**共用点：**
1. ✅ Task Splitter（任务拆分）
2. ✅ Task Scheduler（任务调度）
3. ✅ Progress Monitor（进度监控）
4. ✅ Retry Engine（重试策略）
5. ✅ device_tasks（底层执行）

**独立点：**
1. 🔴 MML 脚本解析（业务特定）
2. 🔴 命令映射逻辑（业务特定）
3. 🔴 父任务表结构（业务特定）

**收益：**
- 代码复用率 ~70%
- 维护成本降低 50%+
- 扩展新任务类型只需 1-2 天

**实施建议：**
- 分阶段实施，先完成基础设施扩展
- 共用服务层先行，业务层后续
- 充分测试共用服务的边界场景
