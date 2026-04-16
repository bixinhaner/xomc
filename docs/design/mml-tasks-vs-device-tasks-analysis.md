# MML Tasks 与 Device Tasks 整合可行性分析

> 分析是否可以将 `mml_tasks` 表的功能整合到现有的 `device_tasks` 表中

---

## 一、表结构对比

### 1.1 device_tasks 表（现有）

```sql
CREATE TABLE device_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_sn       VARCHAR(64) NOT NULL,           -- 单个设备 SN
    method          VARCHAR(64) NOT NULL,           -- RPC 方法名
    params          JSONB,                          -- 方法参数
    priority        INTEGER DEFAULT 10,             -- 优先级
    command_key     VARCHAR(128),                   -- TR069 CommandKey
    cwmp_id         VARCHAR(256),                   -- SOAP Header ID
    
    status          VARCHAR(16) NOT NULL DEFAULT 'pending',
    retry_count     INTEGER DEFAULT 0,
    max_retries     INTEGER DEFAULT 3,
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at         TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ,
    
    result          JSONB,                          -- 单个设备执行结果
    error_code      INTEGER,
    error_message   TEXT,
    
    source          VARCHAR(32) DEFAULT 'api',      -- 任务来源
    creator_id      VARCHAR(64),                    -- 创建者 ID
    description     TEXT                            -- 任务描述
);
```

**核心特征：**
- ✅ 面向**单个设备**的任务
- ✅ TR-069 RPC 方法调用
- ✅ 实时异步任务队列
- ✅ 支持重试、优先级、超时
- ✅ 与 Redis 队列配合使用
- ❌ 不支持批量设备
- ❌ 不支持定时/周期执行
- ❌ 不支持任务编排（多命令序列）

---

### 1.2 mml_tasks 表（计划）

```sql
CREATE TABLE mml_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_name       VARCHAR(200),                   -- 任务名称
    script_id       UUID REFERENCES mml_scripts(id),-- 关联脚本（可选）
    device_sns      JSONB NOT NULL,                 -- 多个设备 SN 列表
    commands        JSONB NOT NULL DEFAULT '[]',    -- 多个命令序列
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    results         JSONB DEFAULT '[]',             -- 所有设备执行结果
    creator         VARCHAR(100),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 扩展字段
execute_type      VARCHAR(20) DEFAULT 'active',     -- 执行类型：active/scheduled/periodic
scheduled_at      TIMESTAMPTZ,                      -- 定时执行时间
period_start      TIMESTAMPTZ,                      -- 周期开始时间
period_end        TIMESTAMPTZ,                      -- 周期结束时间
period_time       TIME,                             -- 周期执行时间点
offline_retry     BOOLEAN DEFAULT false,            -- 离线重试
offline_wait      INT DEFAULT 60,                   -- 离线等待时间（秒）
failed_retry      BOOLEAN DEFAULT false,            -- 失败重试
retry_count       INT DEFAULT 3,                    -- 重试次数
retry_interval    INT DEFAULT 5,                    -- 重试间隔（分钟）
started_at        TIMESTAMPTZ,                      -- 开始执行时间
finished_at       TIMESTAMPTZ,                      -- 完成时间
total_devices     INT DEFAULT 0,                    -- 设备总数
success_count     INT DEFAULT 0,                    -- 成功设备数
failed_count      INT DEFAULT 0;                    -- 失败设备数
```

**核心特征：**
- ✅ 面向**批量设备**的任务
- ✅ 支持**多命令序列**（脚本）
- ✅ 支持**定时/周期执行**
- ✅ 支持**离线重试**策略
- ✅ 任务级别的状态跟踪（汇总统计）
- ✅ 与 MML 脚本关联
- ❌ 不直接管理单个 RPC 调用细节

---

## 二、功能差异分析

| 功能维度 | device_tasks | mml_tasks | 差异程度 |
|---------|--------------|-----------|---------|
| **任务粒度** | 单设备单命令 | 多设备多命令 | 🔴 重大差异 |
| **批量支持** | ❌ 不支持 | ✅ 支持 | 🔴 重大差异 |
| **命令序列** | ❌ 单命令 | ✅ 多命令编排 | 🔴 重大差异 |
| **定时执行** | ❌ 不支持 | ✅ 支持 | 🟡 中等差异 |
| **周期执行** | ❌ 不支持 | ✅ 支持 | 🟡 中等差异 |
| **离线重试** | ❌ 不支持 | ✅ 支持 | 🟡 中等差异 |
| **任务编排** | ❌ 无 | ✅ 脚本关联 | 🟡 中等差异 |
| **状态管理** | 单任务状态 | 汇总统计状态 | 🟡 中等差异 |
| **RPC 细节** | ✅ cwmp_id、command_key | ❌ 不关心 | 🟢 可忽略 |
| **优先级** | ✅ 支持 | ❌ 不需要 | 🟢 可忽略 |
| **超时控制** | ✅ expires_at | ❌ 不需要 | 🟢 可忽略 |

---

## 三、整合方案分析

### 方案 A：完全整合（不推荐 ❌）

**思路：** 删除 `mml_tasks` 表，所有功能合并到 `device_tasks` 表

**问题：**

1. **数据模型不匹配**
   - `device_tasks` 一条记录 = 一个设备的一个命令
   - `mml_tasks` 一条记录 = 多个设备的多个命令
   - 如果强行整合，需要将一个 `mml_task` 拆分成 N×M 条 `device_task` 记录

2. **批量操作困难**
   ```sql
   -- 查询一个 MML 任务的进度（需要聚合）
   SELECT 
       COUNT(*) as total,
       COUNT(*) FILTER (WHERE status = 'completed') as success,
       COUNT(*) FILTER (WHERE status = 'failed') as failed
   FROM device_tasks
   WHERE description = 'mml_task:xxx'  -- 需要额外字段关联
   ```

3. **定时/周期执行无法表达**
   - `device_tasks` 是即时任务，创建后立即进入队列
   - `mml_tasks` 的定时/周期执行需要调度器支持

4. **脚本关联丢失**
   - `mml_tasks` 可以关联 `mml_scripts` 表
   - `device_tasks` 没有这个概念

5. **前端展示复杂**
   - 前端需要显示"任务列表"（宏观视角）
   - 如果只有 `device_tasks`，需要大量聚合查询

**结论：❌ 不可行，会严重破坏现有架构**

---

### 方案 B：父子关系（推荐 ✅）

**思路：** `mml_tasks` 作为父任务，`device_tasks` 作为子任务

**数据模型：**

```sql
-- mml_tasks 作为高层任务编排
CREATE TABLE mml_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_name       VARCHAR(200),
    script_id       UUID REFERENCES mml_scripts(id),
    device_sns      JSONB NOT NULL,
    commands        JSONB NOT NULL DEFAULT '[]',
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    
    -- 执行策略
    execute_type    VARCHAR(20) DEFAULT 'active',
    scheduled_at    TIMESTAMPTZ,
    period_start    TIMESTAMPTZ,
    period_end      TIMESTAMPTZ,
    period_time     TIME,
    offline_retry   BOOLEAN DEFAULT false,
    offline_wait    INT DEFAULT 60,
    failed_retry    BOOLEAN DEFAULT false,
    retry_count     INT DEFAULT 3,
    retry_interval  INT DEFAULT 5,
    
    -- 汇总统计
    total_devices   INT DEFAULT 0,
    success_count   INT DEFAULT 0,
    failed_count    INT DEFAULT 0,
    
    -- 时间
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    creator         VARCHAR(100),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- device_tasks 扩展字段（增加 parent_task_id）
ALTER TABLE device_tasks 
ADD COLUMN parent_task_id UUID REFERENCES mml_tasks(id) ON DELETE SET NULL,
ADD COLUMN task_type VARCHAR(32) DEFAULT 'normal';  -- normal/mml/provisioning/etc.

-- 索引
CREATE INDEX idx_device_tasks_parent ON device_tasks(parent_task_id);
CREATE INDEX idx_device_tasks_type ON device_tasks(task_type);
```

**工作流程：**

```
1. 用户创建 MML 任务
   POST /api/v1/mml/tasks
   {
     "task_name": "批量查询设备信息",
     "device_sns": ["SN001", "SN002", "SN003"],
     "commands": [
       {"command_code": "LST BASIC_INFO", "parameters": {}},
       {"command_code": "LST ALARM", "parameters": {}}
     ]
   }
   
2. 系统创建 mml_tasks 记录
   INSERT INTO mml_tasks (id, task_name, device_sns, commands, status)
   VALUES ('task-001', '批量查询设备信息', ['SN001','SN002','SN003'], [...], 'pending')
   
3. 调度器拆分任务，为每个设备创建 device_tasks
   FOR each device_sn IN ['SN001', 'SN002', 'SN003']:
     FOR each command IN commands:
       INSERT INTO device_tasks (
         device_sn, 
         method,           -- 从 commands 映射到 RPC 方法
         params,           -- 从 commands 映射到参数
         parent_task_id,   -- 'task-001'
         task_type,        -- 'mml'
         source            -- 'mml_scheduler'
       )
       
4. ACS Worker 正常消费 device_tasks（无需修改）
   - Pop task from Redis queue
   - Execute TR-069 RPC
   - Update device_tasks status
   
5. 监控器汇总进度
   SELECT 
       COUNT(*) as total,
       COUNT(*) FILTER (WHERE status = 'completed') as success,
       COUNT(*) FILTER (WHERE status = 'failed') as failed
   FROM device_tasks
   WHERE parent_task_id = 'task-001'
   
   -- 更新 mml_tasks
   UPDATE mml_tasks 
   SET success_count = ..., failed_count = ..., 
       status = CASE WHEN ... THEN 'completed' ELSE 'running' END
   WHERE id = 'task-001'
```

**优势：**

1. ✅ **保持现有架构不变**
   - `device_tasks` 的核心逻辑完全复用
   - ACS Worker 无需修改
   - Redis 队列机制不变

2. ✅ **清晰的任务层次**
   - `mml_tasks`：用户视角的"任务"（宏观）
   - `device_tasks`：系统视角的"子任务"（微观）

3. ✅ **灵活的扩展性**
   - 未来可以支持其他类型的批量任务（如批量配置、批量重启）
   - 只需增加 `task_type` 即可区分

4. ✅ **前端友好**
   - 任务列表页面：查询 `mml_tasks`
   - 任务详情页面：查询 `device_tasks WHERE parent_task_id = ?`

5. ✅ **脚本关联**
   - `mml_tasks` 可以关联 `mml_scripts`
   - 脚本内容可以复用于生成 `device_tasks`

**劣势：**

1. ⚠️ 需要额外的数据同步逻辑
   - 需要监控器定期汇总 `device_tasks` 状态到 `mml_tasks`
   - 增加了一定的系统复杂度

2. ⚠️ 需要扩展 `device_tasks` 表
   - 增加 `parent_task_id` 和 `task_type` 字段
   - 影响范围较小（仅增加两个可选字段）

---

### 方案 C：独立并存（备选 ⚠️）

**思路：** `mml_tasks` 和 `device_tasks` 完全独立，通过应用层协调

**数据模型：**

```sql
-- 两个表完全独立，无外键关联
-- mml_tasks 存储任务编排信息
-- device_tasks 存储 RPC 执行信息
-- 通过 mml_tasks.results JSONB 字段存储关联的 device_task IDs
```

**工作流程：**

```
1. 创建 mml_tasks
2. 应用层拆分任务，创建 device_tasks
3. 在 mml_tasks.results 中记录 device_task IDs
   {
     "device_tasks": [
       {"device_sn": "SN001", "task_ids": ["uuid1", "uuid2"]},
       {"device_sn": "SN002", "task_ids": ["uuid3", "uuid4"]}
     ]
   }
4. ACS Worker 执行 device_tasks
5. 应用层查询 device_tasks 结果，更新 mml_tasks.results
```

**优势：**

1. ✅ 完全解耦，互不影响
2. ✅ `device_tasks` 表结构无需修改

**劣势：**

1. ❌ 数据关联弱，依赖应用层维护
2. ❌ 查询复杂，需要多次 JOIN 或聚合
3. ❌ 数据一致性难以保证
4. ❌ 不推荐用于生产环境

---

## 四、推荐方案详细设计

### 4.1 数据库 Schema

```sql
-- ==========================================
-- 1. mml_tasks 表（新增）
-- ==========================================
CREATE TABLE mml_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_name       VARCHAR(200),
    script_id       UUID REFERENCES mml_scripts(id) ON DELETE SET NULL,
    device_sns      JSONB NOT NULL,                 -- 目标设备 SN 列表
    commands        JSONB NOT NULL DEFAULT '[]',    -- 命令序列
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    
    -- 执行策略
    execute_type    VARCHAR(20) DEFAULT 'active',   -- active/scheduled/periodic
    scheduled_at    TIMESTAMPTZ,                    -- 定时执行时间
    period_start    TIMESTAMPTZ,                    -- 周期开始时间
    period_end      TIMESTAMPTZ,                    -- 周期结束时间
    period_time     TIME,                           -- 周期执行时间点
    offline_retry   BOOLEAN DEFAULT false,          -- 离线重试
    offline_wait    INT DEFAULT 60,                 -- 离线等待时间（秒）
    failed_retry    BOOLEAN DEFAULT false,          -- 失败重试
    retry_count     INT DEFAULT 3,                  -- 重试次数
    retry_interval  INT DEFAULT 5,                  -- 重试间隔（分钟）
    
    -- 汇总统计
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
CREATE INDEX IF NOT EXISTS idx_mml_tasks_commands_gin ON mml_tasks USING GIN (commands jsonb_path_ops);

-- ==========================================
-- 2. device_tasks 表扩展（修改现有表）
-- ==========================================
ALTER TABLE device_tasks 
ADD COLUMN IF NOT EXISTS parent_task_id UUID REFERENCES mml_tasks(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS task_type VARCHAR(32) DEFAULT 'normal';

CREATE INDEX IF NOT EXISTS idx_device_tasks_parent ON device_tasks(parent_task_id);
CREATE INDEX IF NOT EXISTS idx_device_tasks_type ON device_tasks(task_type);

-- ==========================================
-- 3. 触发器：自动更新 updated_at
-- ==========================================
CREATE TRIGGER trigger_mml_tasks_updated_at
    BEFORE UPDATE ON mml_tasks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### 4.2 任务拆分逻辑

```go
package mml

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/google/uuid"
    "github.com/jackc/pgx/v5/pgxpool"
)

// MMLTaskScheduler MML 任务调度器
type MMLTaskScheduler struct {
    pool *pgxpool.Pool
}

// SplitTask 将 MML 任务拆分为 device_tasks
func (s *MMLTaskScheduler) SplitTask(ctx context.Context, mmlTaskID uuid.UUID) error {
    // 1. 查询 mml_task
    var task struct {
        ID        uuid.UUID
        DeviceSNs json.RawMessage
        Commands  json.RawMessage
    }
    
    err := s.pool.QueryRow(ctx, `
        SELECT id, device_sns, commands 
        FROM mml_tasks 
        WHERE id = $1
    `, mmlTaskID).Scan(&task.ID, &task.DeviceSNs, &task.Commands)
    if err != nil {
        return fmt.Errorf("query mml_task: %w", err)
    }
    
    // 2. 解析设备列表和命令
    var deviceSNs []string
    if err := json.Unmarshal(task.DeviceSNs, &deviceSNs); err != nil {
        return fmt.Errorf("parse device_sns: %w", err)
    }
    
    var commands []MMLCommand
    if err := json.Unmarshal(task.Commands, &commands); err != nil {
        return fmt.Errorf("parse commands: %w", err)
    }
    
    // 3. 为每个设备创建 device_tasks
    tasks := []DeviceTask{}
    for _, sn := range deviceSNs {
        for _, cmd := range commands {
            // 将 MML 命令映射为 TR-069 RPC 方法
            rpcMethod, params := MapMMLToRPC(cmd)
            
            taskID := uuid.New()
            tasks = append(tasks, DeviceTask{
                ID:           taskID.String(),
                DeviceSN:     sn,
                Method:       rpcMethod,
                Params:       params,
                Priority:     10,
                Status:       "pending",
                RetryCount:   0,
                MaxRetries:   3,
                CreatedAt:    time.Now(),
                Source:       "mml_scheduler",
                CreatorID:    "system",
                Description:  fmt.Sprintf("MML Task: %s", mmlTaskID),
                ParentTaskID: &mmlTaskID,
                TaskType:     "mml",
            })
        }
    }
    
    // 4. 批量插入 device_tasks
    if err := s.batchInsertTasks(ctx, tasks); err != nil {
        return fmt.Errorf("batch insert tasks: %w", err)
    }
    
    // 5. 更新 mml_task 状态
    _, err = s.pool.Exec(ctx, `
        UPDATE mml_tasks 
        SET status = 'running', 
            started_at = NOW(),
            total_devices = $2,
            updated_at = NOW()
        WHERE id = $1
    `, mmlTaskID, len(deviceSNs))
    if err != nil {
        return fmt.Errorf("update mml_task status: %w", err)
    }
    
    return nil
}

// MMLCommand MML 命令结构
type MMLCommand struct {
    CommandCode string                 `json:"command_code"`
    Parameters  map[string]interface{} `json:"parameters"`
}

// DeviceTask 设备任务结构
type DeviceTask struct {
    ID           string
    DeviceSN     string
    Method       string
    Params       map[string]interface{}
    Priority     int
    Status       string
    RetryCount   int
    MaxRetries   int
    CreatedAt    time.Time
    Source       string
    CreatorID    string
    Description  string
    ParentTaskID *uuid.UUID
    TaskType     string
}

// MapMMLToRPC 将 MML 命令映射为 TR-069 RPC 方法
func MapMMLToRPC(cmd MMLCommand) (string, map[string]interface{}) {
    // 根据 command_code 映射到对应的 TR-069 RPC 方法
    // 示例：
    // LST BASIC_INFO -> GetParameterValues
    // SET PARAM -> SetParameterValues
    // LST ALARM -> GetParameterValues (特定参数路径)
    
    switch cmd.CommandCode {
    case "LST BASIC_INFO":
        return "GetParameterValues", map[string]interface{}{
            "names": []string{
                "Device.DeviceInfo.Manufacturer",
                "Device.DeviceInfo.ModelName",
                "Device.DeviceInfo.SoftwareVersion",
            },
        }
    case "LST ALARM":
        return "GetParameterValues", map[string]interface{}{
            "names": []string{
                "Device.DeviceInfo.AlarmStatus",
            },
        }
    default:
        return "GetParameterValues", map[string]interface{}{
            "names": []string{"Device."},
        }
    }
}

func (s *MMLTaskScheduler) batchInsertTasks(ctx context.Context, tasks []DeviceTask) error {
    // 批量插入逻辑（使用 COPY 或批量 INSERT）
    // ...
    return nil
}
```

### 4.3 进度汇总逻辑

```go
// TaskProgressMonitor 任务进度监控器
type TaskProgressMonitor struct {
    pool *pgxpool.Pool
}

// UpdateTaskProgress 更新任务进度
func (m *TaskProgressMonitor) UpdateTaskProgress(ctx context.Context, mmlTaskID uuid.UUID) error {
    // 查询子任务统计
    var stats struct {
        Total   int
        Success int
        Failed  int
    }
    
    err := m.pool.QueryRow(ctx, `
        SELECT 
            COUNT(*) as total,
            COUNT(*) FILTER (WHERE status = 'completed') as success,
            COUNT(*) FILTER (WHERE status = 'failed') as failed
        FROM device_tasks
        WHERE parent_task_id = $1
    `, mmlTaskID).Scan(&stats.Total, &stats.Success, &stats.Failed)
    if err != nil {
        return fmt.Errorf("query stats: %w", err)
    }
    
    // 判断任务状态
    newStatus := "running"
    if stats.Success+stats.Failed == stats.Total && stats.Total > 0 {
        if stats.Failed == 0 {
            newStatus = "completed"
        } else if stats.Success == 0 {
            newStatus = "failed"
        } else {
            newStatus = "partial"
        }
    }
    
    // 更新 mml_task
    _, err = m.pool.Exec(ctx, `
        UPDATE mml_tasks 
        SET 
            success_count = $2,
            failed_count = $3,
            status = $4,
            finished_at = CASE WHEN $4 IN ('completed', 'failed', 'partial') THEN NOW() ELSE finished_at END,
            updated_at = NOW()
        WHERE id = $1
    `, mmlTaskID, stats.Success, stats.Failed, newStatus)
    if err != nil {
        return fmt.Errorf("update mml_task: %w", err)
    }
    
    return nil
}
```

### 4.4 API 设计

```go
// MMLTaskHandler MML 任务处理器
type MMLTaskHandler struct {
    scheduler *MMLTaskScheduler
    monitor   *TaskProgressMonitor
}

// CreateTask 创建 MML 任务
func (h *MMLTaskHandler) CreateTask(c *gin.Context) {
    var req struct {
        TaskName  string        `json:"task_name"`
        ScriptID  *uuid.UUID    `json:"script_id"`
        DeviceSNs []string      `json:"device_sns" binding:"required"`
        Commands  []MMLCommand  `json:"commands" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // 创建 mml_task
    var taskID uuid.UUID
    err := h.scheduler.pool.QueryRow(c.Request.Context(), `
        INSERT INTO mml_tasks (task_name, script_id, device_sns, commands, creator)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `, req.TaskName, req.ScriptID, req.DeviceSNs, req.Commands, GetUsername(c)).Scan(&taskID)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    // 异步拆分任务
    go func() {
        if err := h.scheduler.SplitTask(context.Background(), taskID); err != nil {
            log.Printf("Failed to split task %s: %v", taskID, err)
        }
    }()
    
    c.JSON(201, gin.H{
        "data": gin.H{
            "id": taskID,
            "status": "pending",
        },
    })
}

// GetTaskProgress 获取任务进度
func (h *MMLTaskHandler) GetTaskProgress(c *gin.Context) {
    taskID := c.Param("id")
    
    // 更新进度
    go func() {
        h.monitor.UpdateTaskProgress(context.Background(), uuid.MustParse(taskID))
    }()
    
    // 查询任务信息
    var task struct {
        ID           uuid.UUID
        TaskName     string
        Status       string
        TotalDevices int
        SuccessCount int
        FailedCount  int
        StartedAt    *time.Time
        FinishedAt   *time.Time
    }
    
    err := h.scheduler.pool.QueryRow(c.Request.Context(), `
        SELECT id, task_name, status, total_devices, success_count, failed_count, started_at, finished_at
        FROM mml_tasks
        WHERE id = $1
    `, taskID).Scan(&task.ID, &task.TaskName, &task.Status, &task.TotalDevices, &task.SuccessCount, &task.FailedCount, &task.StartedAt, &task.FinishedAt)
    if err != nil {
        c.JSON(404, gin.H{"error": "任务不存在"})
        return
    }
    
    c.JSON(200, gin.H{"data": task})
}

// GetTaskDetails 获取任务详情（子任务列表）
func (h *MMLTaskHandler) GetTaskDetails(c *gin.Context) {
    taskID := c.Param("id")
    page := c.DefaultQuery("page", "1")
    pageSize := c.DefaultQuery("pageSize", "20")
    
    // 查询子任务
    rows, err := h.scheduler.pool.Query(c.Request.Context(), `
        SELECT id, device_sn, method, status, created_at, completed_at, error_message
        FROM device_tasks
        WHERE parent_task_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `, taskID, pageSize, (page-1)*pageSize)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    defer rows.Close()
    
    tasks := []gin.H{}
    for rows.Next() {
        var task struct {
            ID           string
            DeviceSN     string
            Method       string
            Status       string
            CreatedAt    time.Time
            CompletedAt  *time.Time
            ErrorMessage *string
        }
        rows.Scan(&task.ID, &task.DeviceSN, &task.Method, &task.Status, &task.CreatedAt, &task.CompletedAt, &task.ErrorMessage)
        tasks = append(tasks, gin.H{
            "id":            task.ID,
            "device_sn":     task.DeviceSN,
            "method":        task.Method,
            "status":        task.Status,
            "created_at":    task.CreatedAt,
            "completed_at":  task.CompletedAt,
            "error_message": task.ErrorMessage,
        })
    }
    
    c.JSON(200, gin.H{
        "data": gin.H{
            "items": tasks,
            "total": len(tasks),
        },
    })
}
```

---

## 五、实施计划

### Phase 1: 数据库迁移（1 天）

- [ ] 创建 `mml_tasks` 表
- [ ] 扩展 `device_tasks` 表（增加 `parent_task_id` 和 `task_type`）
- [ ] 创建索引和触发器

### Phase 2: 核心逻辑开发（3-5 天）

- [ ] 实现 `MMLTaskScheduler.SplitTask()` 方法
- [ ] 实现 MML 命令到 TR-069 RPC 的映射逻辑
- [ ] 实现 `TaskProgressMonitor.UpdateTaskProgress()` 方法
- [ ] 实现批量插入逻辑

### Phase 3: API 开发（2-3 天）

- [ ] 实现 MML 任务 CRUD API
- [ ] 实现任务进度查询 API
- [ ] 实现任务详情查询 API
- [ ] 实现任务控制 API（启动/暂停/终止）

### Phase 4: 前端集成（2-3 天）

- [ ] 实现 MML 脚本任务页面
- [ ] 实现任务列表和进度展示
- [ ] 实现任务详情页面
- [ ] 实现任务控制按钮

### Phase 5: 测试与优化（2-3 天）

- [ ] 单元测试
- [ ] 集成测试
- [ ] 性能测试（批量任务拆分性能）
- [ ] 异常场景测试

**总工期：10-15 天**

---

## 六、总结

### 推荐方案：方案 B（父子关系）✅

**核心理念：**
- `mml_tasks` 作为**业务层任务**（用户视角）
- `device_tasks` 作为**执行层任务**（系统视角）
- 通过 `parent_task_id` 建立关联

**优势：**
1. ✅ 完全复用现有 `device_tasks` 基础设施
2. ✅ ACS Worker 无需修改
3. ✅ 清晰的任务层次和职责划分
4. ✅ 支持未来扩展（其他批量任务类型）
5. ✅ 前端展示友好

**实施风险：**
- ⚠️ 低风险：仅扩展 `device_tasks` 表（增加两个可选字段）
- ⚠️ 需要额外的进度同步逻辑（可通过定时任务或事件驱动解决）

**不推荐方案 A（完全整合）的原因：**
- ❌ 破坏现有架构
- ❌ 数据模型不匹配
- ❌ 查询复杂度高
- ❌ 扩展性差

---

## 七、附录

### 7.1 现有 device_tasks 使用场景

| 场景 | 来源 | 说明 |
|------|------|------|
| API 创建任务 | 用户手动触发 | 单设备参数查询/设置 |
| 自动开站 | 开站引擎 | 开站流程中的 RPC 调用 |
| 设备规则 | 规则引擎 | 批量规则应用 |
| 定时任务 | 调度器 | 周期性参数同步 |

### 7.2 未来扩展性

通过 `task_type` 字段，可以支持更多任务类型：

```sql
-- 任务类型枚举
task_type = 'normal'         -- 普通任务（API 创建）
task_type = 'mml'            -- MML 任务
task_type = 'provisioning'   -- 自动开站任务
task_type = 'rule'           -- 设备规则任务
task_type = 'upgrade'        -- 批量升级任务
task_type = 'config'         -- 批量配置任务
```

所有类型的任务都共享 `device_tasks` 的执行机制，但通过 `mml_tasks`、`provisioning_tasks` 等父任务表实现不同的业务逻辑。
