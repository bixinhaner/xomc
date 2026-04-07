# 设备规则设计方案

> 版本: v1.0
> 日期: 2026-04-07
> 作者: Claude

---

## 1. 概述

### 1.1 功能定位

设备规则（Device Rules）是 OMC 系统中用于自动化设备分组管理的核心功能。它定义了一组匹配条件，当满足条件的设备被发现时，自动将其归属到指定的设备分组。

### 1.2 与设备分组的关系

```
┌─────────────────────────────────────────────────────────────────┐
│                        设备规则 (Device Rule)                     │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │  匹配条件: 设备名称包含 "BJ-" 且 包含 "-5G-"                  │ │
│  │  优先级: 1                                                    │ │
│  │  启用状态: 是                                                 │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                              │                                    │
│                              │ 绑定                               │
│                              ▼                                    │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │               目标设备分组 (Device Group)                     │ │
│  │  名称: 北京5G区域                                             │ │
│  │  层级: L2 (二级分组)                                          │ │
│  │  分组自身规则: (被覆盖，不生效)                               │ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

**核心关系**:
- 设备规则可以绑定到一个设备分组
- 存在绑定关系时，**设备分组的自身匹配规则不生效**，由设备规则接管
- 删除设备规则后，设备分组恢复使用自身的匹配规则

### 1.3 参考标准

| 标准 | 说明 |
|------|------|
| TR-069 Amendment 6 | CWMP 协议，设备管理框架 |
| TR-098 | InternetGatewayDevice 数据模型 |
| TR-181 Issue 2 | Device 数据模型 |
| 3GPP TS 32.422 | PM/Measurement Report 规范 |
| 中国移动 OMC 规范 | 设备分组与归属规则管理 |

---

## 2. 功能需求

### 2.1 设备规则 CRUD

| 操作 | 说明 | 约束 |
|------|------|------|
| 创建 | 新建设备规则，定义匹配条件和目标分组 | 目标分组必须是 L2 分组 |
| 编辑 | 修改匹配条件、目标分组、启用状态 | 可以修改目标分组 |
| 删除 | 删除设备规则 | 删除后解除与分组的绑定关系 |
| 启用/禁用 | 切换规则的启用状态 | 禁用后无法"应用" |

### 2.2 规则优先级

设备规则支持优先级排序，按优先级从高到低依次匹配：

```
优先级 1: 设备名称包含 "BJ-" → 北京区域
优先级 2: 设备名称包含 "SH-" → 上海区域
优先级 3: TAC 在 100-200 范围 → 测试分组
...
```

**匹配逻辑**:
1. 设备首先匹配优先级最高的规则
2. 如果匹配成功，设备归属到规则指定的目标分组
3. 如果不匹配，继续尝试下一个优先级的规则
4. 如果所有规则都不匹配，设备保持原分组或归属到"未分组设备"

### 2.3 "应用"操作

"应用"是对设备分组下的设备列表进行重新计算的过程：

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  应用规则    │ →  │  匹配计算    │ →  │  设备迁移    │
│  (触发)      │    │  (异步任务)  │    │  (结果)      │
└──────────────┘    └──────────────┘    └──────────────┘
```

**流程**:
1. 用户点击"应用"按钮
2. 后端创建异步任务，扫描所有设备
3. 按优先级顺序匹配设备规则
4. 匹配成功的设备迁移到目标分组
5. 任务完成后通知用户

**约束**:
- 未启用的规则无法"应用"
- 应用操作是异步的，用户可以继续其他操作
- 支持查看应用进度和结果

### 2.4 规则与分组绑定机制

```
状态 1: 分组无规则绑定
┌─────────────────────┐
│  北京区域 (L2 分组)  │
│  自身规则:          │
│  - 包含 "BJ-"       │ ← 使用分组自身的规则
└─────────────────────┘

状态 2: 分组被规则绑定
┌─────────────────────┐     ┌─────────────────────┐
│  北京5G区域规则      │ ──→ │  北京5G区域 (L2 分组) │
│  规则条件:          │     │  自身规则:          │
│  - 包含 "BJ-"       │     │  - (被覆盖)         │ ← 使用规则的匹配条件
│  - 包含 "-5G-"      │     │                     │
└─────────────────────┘     └─────────────────────┘

状态 3: 规则删除后
┌─────────────────────┐
│  北京5G区域 (L2 分组)  │
│  自身规则:          │
│  - (恢复使用)       │ ← 恢复使用分组自身的规则
└─────────────────────┘
```

---

## 3. 数据模型设计

### 3.1 设备规则表 (device_rules)

```sql
-- 设备规则表
CREATE TABLE device_rules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) NOT NULL,                -- 规则名称
    priority        INTEGER NOT NULL DEFAULT 0,           -- 优先级（数字越小优先级越高）
    target_group_id UUID REFERENCES device_groups(id),    -- 目标设备分组
    enabled         BOOLEAN NOT NULL DEFAULT false,       -- 启用状态
    -- 匹配规则
    matching_mode   VARCHAR(16),                          -- 匹配模式: deviceName, lac, tac
    name_rule_list  JSONB,                                -- 设备名称匹配规则
    lac_list        INTEGER[],                            -- LAC 列表
    tac_list        INTEGER[],                            -- TAC 列表
    -- 审计字段
    description     TEXT,                                 -- 描述
    created_by      VARCHAR(64),
    updated_by      VARCHAR(64),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 唯一约束：同一优先级只能有一个规则
CREATE UNIQUE INDEX idx_dr_priority_unique ON device_rules (priority) WHERE enabled = true;

-- 索引
CREATE INDEX idx_dr_target_group ON device_rules (target_group_id);
CREATE INDEX idx_dr_enabled ON device_rules (enabled);

-- 注释
COMMENT ON TABLE device_rules IS '设备归属规则';
COMMENT ON COLUMN device_rules.priority IS '优先级，数字越小优先级越高';
COMMENT ON COLUMN device_rules.target_group_id IS '目标设备分组，设备匹配后归属到此分组';
COMMENT ON COLUMN device_rules.matching_mode IS '匹配模式: deviceName(设备名称), lac(位置区码), tac(跟踪区码)';
COMMENT ON COLUMN device_rules.name_rule_list IS '设备名称匹配规则，JSON数组格式';
COMMENT ON COLUMN device_rules.enabled IS '启用状态，未启用的规则无法应用';
```

### 3.2 设备名称规则 JSON 结构

```json
[
  {
    "condition": "contain",    // contain, notContain, startWith, endWith
    "value": "BJ-",            // 匹配值
    "andOr": "and"             // and, or（第一条不需要此字段）
  },
  {
    "condition": "contain",
    "value": "-5G-",
    "andOr": "and"
  }
]
```

**匹配逻辑**:
- 规则按 OR 分组（andOr = "or" 开始新组）
- 同一组内的条件必须全部满足（AND 逻辑）
- 任一 OR 组满足即匹配成功

**示例**:
```
规则: 包含 "BJ-" AND 包含 "-5G-" OR 包含 "SH-"
解析: (包含 "BJ-" AND 包含 "-5G-") OR (包含 "SH-")
```

### 3.3 设备分组表扩展

在现有 `device_groups` 表中添加字段，标记是否被规则绑定：

```sql
-- 设备分组表新增字段
ALTER TABLE device_groups ADD COLUMN bound_rule_id UUID REFERENCES device_rules(id);

COMMENT ON COLUMN device_groups.bound_rule_id IS '绑定的设备规则ID，存在时使用规则的匹配条件';
```

### 3.4 异步任务表 (device_rule_tasks)

```sql
-- 设备规则应用任务表
CREATE TABLE device_rule_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id         UUID NOT NULL REFERENCES device_rules(id),
    status          VARCHAR(16) NOT NULL DEFAULT 'pending', -- pending, running, completed, failed
    total_devices   INTEGER DEFAULT 0,                      -- 总设备数
    matched_count   INTEGER DEFAULT 0,                      -- 匹配成功数
    failed_count    INTEGER DEFAULT 0,                      -- 失败数
    started_at      TIMESTAMP WITH TIME ZONE,
    completed_at    TIMESTAMP WITH TIME ZONE,
    error_message   TEXT,
    created_by      VARCHAR(64),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX idx_drt_rule ON device_rule_tasks (rule_id);
CREATE INDEX idx_drt_status ON device_rule_tasks (status);

COMMENT ON TABLE device_rule_tasks IS '设备规则应用任务';
```

---

## 4. API 设计

### 4.1 设备规则管理 API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/device-rules` | 获取规则列表（支持分页、过滤） |
| GET | `/api/v1/device-rules/:id` | 获取单个规则详情 |
| POST | `/api/v1/device-rules` | 创建规则 |
| PUT | `/api/v1/device-rules/:id` | 更新规则 |
| DELETE | `/api/v1/device-rules/:id` | 删除规则 |
| PATCH | `/api/v1/device-rules/:id/toggle` | 启用/禁用规则 |
| PUT | `/api/v1/device-rules/batch-sort` | 批量调整优先级 |

### 4.2 规则应用 API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/device-rules/:id/apply` | 应用规则（触发异步任务） |
| GET | `/api/v1/device-rules/:id/tasks` | 获取规则的应用任务列表 |
| GET | `/api/v1/device-rules/:id/tasks/:taskId` | 获取任务详情和进度 |

### 4.3 API 请求/响应示例

#### 创建规则

```json
// POST /api/v1/device-rules
// Request
{
  "name": "北京5G设备规则",
  "priority": 1,
  "target_group_id": "30000000-0004-4000-8000-000000000002",
  "enabled": true,
  "matching_mode": "deviceName",
  "name_rule_list": [
    { "condition": "contain", "value": "BJ-" },
    { "condition": "contain", "value": "-5G-", "andOr": "and" }
  ]
}

// Response
{
  "id": "40000000-0004-4000-8000-000000000001",
  "name": "北京5G设备规则",
  "priority": 1,
  "target_group_id": "30000000-0004-4000-8000-000000000002",
  "target_group_name": "北京5G区域",
  "enabled": true,
  "matching_mode": "deviceName",
  "name_rule_list": [...],
  "operators": "包含 \"BJ-\" 且 包含 \"-5G-\"",
  "created_at": "2026-04-07T12:00:00Z"
}
```

#### 应用规则

```json
// POST /api/v1/device-rules/:id/apply
// Request
{
  "source_group_ids": ["group-1", "group-2"]  // 可选，指定从哪些分组中匹配设备
}

// Response
{
  "task_id": "50000000-0005-4000-8000-000000000001",
  "status": "pending",
  "message": "应用任务已创建"
}
```

#### 查询任务进度

```json
// GET /api/v1/device-rules/:id/tasks/:taskId
// Response
{
  "id": "50000000-0005-4000-8000-000000000001",
  "rule_id": "40000000-0004-4000-8000-000000000001",
  "status": "running",
  "total_devices": 1000,
  "matched_count": 450,
  "failed_count": 5,
  "progress_percent": 45,
  "started_at": "2026-04-07T12:01:00Z",
  "estimated_completion": "2026-04-07T12:05:00Z"
}
```

---

## 5. 后端实现

### 5.1 目录结构

```
internal/topology/
├── handler.go              # HTTP 处理器（已有）
├── service.go              # 业务逻辑（已有）
├── repository.go           # 仓储接口（已有）
├── pg_repository.go        # PostgreSQL 实现（已有）
├── model.go                # 数据模型（扩展）
├── matcher.go              # 设备匹配引擎（已有）
├── rule_handler.go         # 设备规则 HTTP 处理器（新增）
├── rule_service.go         # 设备规则业务逻辑（新增）
├── rule_repository.go      # 设备规则仓储接口（新增）
├── rule_pg_repository.go   # 设备规则 PostgreSQL 实现（新增）
└── rule_task_worker.go     # 异步任务 Worker（新增）
```

### 5.2 核心数据结构

```go
// model.go 扩展

// DeviceRule 设备规则
type DeviceRule struct {
    ID            uuid.UUID      `json:"id"`
    Name          string         `json:"name"`
    Priority      int            `json:"priority"`
    TargetGroupID *uuid.UUID     `json:"target_group_id,omitempty"`
    TargetGroupName string       `json:"target_group_name,omitempty"`
    Enabled       bool           `json:"enabled"`
    MatchingMode  MatchingMode   `json:"matching_mode,omitempty"`
    NameRuleList  []NameRule     `json:"name_rule_list,omitempty"`
    LACList       []int          `json:"lac_list,omitempty"`
    TACList       []int          `json:"tac_list,omitempty"`
    Description   string         `json:"description,omitempty"`
    Operators     string         `json:"operators"` // 生成的规则描述
    CreatedBy     string         `json:"created_by,omitempty"`
    UpdatedBy     string         `json:"updated_by,omitempty"`
    CreatedAt     time.Time      `json:"created_at"`
    UpdatedAt     time.Time      `json:"updated_at"`
}

// CreateRuleRequest 创建规则请求
type CreateRuleRequest struct {
    Name          string       `json:"name" binding:"required"`
    Priority      int          `json:"priority"`
    TargetGroupID string       `json:"target_group_id" binding:"required"`
    Enabled       bool         `json:"enabled"`
    MatchingMode  string       `json:"matching_mode" binding:"required"`
    NameRuleList  []NameRule   `json:"name_rule_list"`
    LACList       []int        `json:"lac_list"`
    TACList       []int        `json:"tac_list"`
    Description   string       `json:"description"`
}

// UpdateRuleRequest 更新规则请求
type UpdateRuleRequest struct {
    Name          *string      `json:"name"`
    Priority      *int         `json:"priority"`
    TargetGroupID *string      `json:"target_group_id"`
    Enabled       *bool        `json:"enabled"`
    MatchingMode  *string      `json:"matching_mode"`
    NameRuleList  []NameRule   `json:"name_rule_list"`
    LACList       []int        `json:"lac_list"`
    TACList       []int        `json:"tac_list"`
    Description   *string      `json:"description"`
}

// RuleTask 规则应用任务
type RuleTask struct {
    ID            uuid.UUID  `json:"id"`
    RuleID        uuid.UUID  `json:"rule_id"`
    Status        string     `json:"status"` // pending, running, completed, failed
    TotalDevices  int        `json:"total_devices"`
    MatchedCount  int        `json:"matched_count"`
    FailedCount   int        `json:"failed_count"`
    StartedAt     *time.Time `json:"started_at,omitempty"`
    CompletedAt   *time.Time `json:"completed_at,omitempty"`
    ErrorMessage  string     `json:"error_message,omitempty"`
    CreatedBy     string     `json:"created_by,omitempty"`
    CreatedAt     time.Time  `json:"created_at"`
}

// ApplyRuleRequest 应用规则请求
type ApplyRuleRequest struct {
    SourceGroupIDs []string `json:"source_group_ids"` // 可选，指定源分组
}
```

### 5.3 规则服务层

```go
// rule_service.go

type DeviceRuleService struct {
    repo        DeviceRuleRepository
    groupRepo   DeviceGroupRepository
    deviceRepo  DeviceRepository
    taskRepo    RuleTaskRepository
    matcher     *DeviceMatcher
    taskQueue   chan uuid.UUID // 任务队列
    logger      *zap.Logger
}

// CreateRule 创建规则并绑定到分组
func (s *DeviceRuleService) CreateRule(ctx context.Context, req CreateRuleRequest, operator string) (*DeviceRule, error) {
    // 1. 验证目标分组存在且为 L2
    groupID, err := uuid.Parse(req.TargetGroupID)
    if err != nil {
        return nil, commonerrors.NewBusinessError(global.ErrCodeInvalidParameter, "invalid target_group_id", nil)
    }

    group, err := s.groupRepo.GetByID(ctx, groupID)
    if err != nil {
        return nil, commonerrors.NewBusinessError(global.ErrCodeGroupNotFound, "target group not found", nil)
    }

    if group.Level != 2 {
        return nil, commonerrors.NewBusinessError(global.ErrCodeGroupLevelInvalid, "target group must be L2", nil)
    }

    // 2. 检查优先级是否冲突
    if req.Enabled {
        exists, _ := s.repo.ExistsByPriority(ctx, req.Priority, nil)
        if exists {
            return nil, commonerrors.NewBusinessError(global.ErrCodeRulePriorityDuplicate, "priority already exists", nil)
        }
    }

    // 3. 创建规则
    rule := &DeviceRule{
        ID:           uuid.New(),
        Name:         req.Name,
        Priority:     req.Priority,
        TargetGroupID: &groupID,
        Enabled:      req.Enabled,
        MatchingMode: MatchingMode(req.MatchingMode),
        NameRuleList: req.NameRuleList,
        LACList:      req.LACList,
        TACList:      req.TACList,
        Description:  req.Description,
        Operators:    generateOperators(req),
        CreatedBy:    operator,
        UpdatedBy:    operator,
    }

    if err := s.repo.Create(ctx, rule); err != nil {
        return nil, fmt.Errorf("create rule: %w", err)
    }

    // 4. 更新分组的绑定关系
    if req.Enabled {
        if err := s.groupRepo.UpdateBoundRule(ctx, groupID, rule.ID); err != nil {
            s.logger.Warn("failed to update group bound rule", zap.Error(err))
        }
    }

    return rule, nil
}

// DeleteRule 删除规则并解除绑定
func (s *DeviceRuleService) DeleteRule(ctx context.Context, id uuid.UUID) error {
    rule, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return err
    }

    // 1. 解除与分组的绑定
    if rule.TargetGroupID != nil {
        if err := s.groupRepo.ClearBoundRule(ctx, *rule.TargetGroupID); err != nil {
            s.logger.Warn("failed to clear group bound rule", zap.Error(err))
        }
    }

    // 2. 删除规则
    return s.repo.Delete(ctx, id)
}

// ApplyRule 应用规则（异步）
func (s *DeviceRuleService) ApplyRule(ctx context.Context, ruleID uuid.UUID, req ApplyRuleRequest, operator string) (*RuleTask, error) {
    // 1. 获取规则
    rule, err := s.repo.GetByID(ctx, ruleID)
    if err != nil {
        return nil, err
    }

    // 2. 检查是否启用
    if !rule.Enabled {
        return nil, commonerrors.NewBusinessError(global.ErrCodeRuleNotEnabled, "rule is not enabled", nil)
    }

    // 3. 创建任务
    task := &RuleTask{
        ID:       uuid.New(),
        RuleID:   ruleID,
        Status:   "pending",
        CreatedBy: operator,
    }

    if err := s.taskRepo.Create(ctx, task); err != nil {
        return nil, fmt.Errorf("create task: %w", err)
    }

    // 4. 提交到任务队列
    s.taskQueue <- task.ID

    return task, nil
}

// processTask 处理应用任务（Worker 协程）
func (s *DeviceRuleService) processTask(ctx context.Context, taskID uuid.UUID) {
    task, err := s.taskRepo.GetByID(ctx, taskID)
    if err != nil {
        s.logger.Error("task not found", zap.String("task_id", taskID.String()))
        return
    }

    // 更新状态为运行中
    now := time.Now()
    task.Status = "running"
    task.StartedAt = &now
    s.taskRepo.Update(ctx, task)

    // 获取规则
    rule, err := s.repo.GetByID(ctx, task.RuleID)
    if err != nil {
        s.failTask(ctx, task, "rule not found")
        return
    }

    // 获取所有设备
    devices, err := s.deviceRepo.GetAll(ctx)
    if err != nil {
        s.failTask(ctx, task, fmt.Sprintf("get devices: %v", err))
        return
    }

    task.TotalDevices = len(devices)

    // 逐个匹配并迁移
    for _, device := range devices {
        matched, err := s.matcher.matchGroup(ctx, DeviceGroup{
            MatchingMode: rule.MatchingMode,
            NameRuleList: rule.NameRuleList,
            LACList:      rule.LACList,
            TACList:      rule.TACList,
        }, MatchRequest{
            DeviceID:   device.ID,
            DeviceName: device.Name,
            LAC:        device.LAC,
            TAC:        device.TAC,
        })

        if err != nil {
            task.FailedCount++
            s.logger.Warn("match device failed",
                zap.String("device_id", device.ID.String()),
                zap.Error(err),
            )
            continue
        }

        if matched && rule.TargetGroupID != nil {
            // 迁移设备到目标分组
            if err := s.groupRepo.AddDevice(ctx, *rule.TargetGroupID, device.ID); err != nil {
                task.FailedCount++
            } else {
                task.MatchedCount++
            }
        }
    }

    // 更新任务完成状态
    completedAt := time.Now()
    task.Status = "completed"
    task.CompletedAt = &completedAt
    s.taskRepo.Update(ctx, task)
}

func (s *DeviceRuleService) failTask(ctx context.Context, task *RuleTask, errMsg string) {
    task.Status = "failed"
    task.ErrorMessage = errMsg
    completedAt := time.Now()
    task.CompletedAt = &completedAt
    s.taskRepo.Update(ctx, task)
}
```

### 5.4 匹配引擎调整

```go
// matcher.go 调整

// MatchDevice 调整：优先使用绑定的规则
func (m *DeviceMatcher) MatchDevice(ctx context.Context, req MatchRequest) (*MatchResult, error) {
    // 获取所有启用的设备规则，按优先级排序
    rules, err := m.ruleRepo.GetEnabledByPriority(ctx)
    if err != nil {
        return nil, fmt.Errorf("get rules: %w", err)
    }

    // 按优先级匹配规则
    for _, rule := range rules {
        if rule.TargetGroupID == nil {
            continue
        }

        matched, err := m.matchRule(ctx, rule, req)
        if err != nil {
            m.logger.Warn("match rule failed",
                zap.String("rule_id", rule.ID.String()),
                zap.Error(err),
            )
            continue
        }

        if matched {
            // 获取目标分组信息
            group, err := m.repo.GetByID(ctx, *rule.TargetGroupID)
            if err != nil {
                continue
            }

            return &MatchResult{
                GroupID:   *rule.TargetGroupID,
                GroupName: group.Name,
                MatchedBy: rule.MatchingMode,
                RuleID:    &rule.ID,
                RuleName:  rule.Name,
            }, nil
        }
    }

    // 没有规则匹配，尝试分组的自身规则
    groups, err := m.repo.GetTreeWithCounts(ctx)
    if err != nil {
        return nil, fmt.Errorf("get groups: %w", err)
    }

    for _, l1Group := range groups {
        for _, l2Group := range l1Group.Children {
            // 跳过有绑定规则的分组
            if l2Group.BoundRuleID != nil {
                continue
            }

            if l2Group.MatchingMode == "" {
                continue
            }

            matched, err := m.matchGroup(ctx, l2Group, req)
            if err != nil {
                continue
            }

            if matched {
                return &MatchResult{
                    GroupID:   l2Group.ID,
                    GroupName: l2Group.Name,
                    MatchedBy: l2Group.MatchingMode,
                }, nil
            }
        }
    }

    return nil, nil
}
```

---

## 6. 前端实现

### 6.1 页面结构

```
omcmb/webcode/src/pages/device/DeviceRules/
├── index.tsx           # 规则列表页面
├── RuleDrawer.tsx      # 新增/编辑规则抽屉
├── ApplyModal.tsx      # 应用规则弹窗
├── TaskDrawer.tsx      # 任务进度抽屉
├── types.ts            # 类型定义
└── utils.ts            # 工具函数（生成 operators 描述）
```

### 6.2 状态管理

```typescript
// 规则列表状态
const [rules, setRules] = useState<DeviceRule[]>([]);
const [loading, setLoading] = useState(false);

// 编辑抽屉状态
const [editDrawerOpen, setEditDrawerOpen] = useState(false);
const [editingRule, setEditingRule] = useState<DeviceRule | null>(null);

// 应用弹窗状态
const [applyModalOpen, setApplyModalOpen] = useState(false);
const [applyingRule, setApplyingRule] = useState<DeviceRule | null>(null);

// 任务抽屉状态
const [taskDrawerOpen, setTaskDrawerOpen] = useState(false);
const [currentTask, setCurrentTask] = useState<RuleTask | null>(null);
```

### 6.3 关键交互

#### 规则列表

```tsx
// 表格列
const columns = [
  { key: 'priority', title: '优先级', width: 70, sortable: true },
  { key: 'actions', title: '操作', width: 200, fixed: 'left' },
  { key: 'enabled', title: '启用', width: 80, render: Switch },
  { key: 'operators', title: '匹配规则', ellipsis: true },
  { key: 'target_group_name', title: '目标分组', width: 150 },
  { key: 'created_at', title: '创建时间', width: 160 },
];

// 操作按钮
// - 启用状态显示"应用"按钮
// - 所有规则显示"编辑"和"删除"按钮
// - 优先级列支持拖拽排序或上下移动
```

#### 应用规则

```tsx
// ApplyModal.tsx
const handleApply = async () => {
  // 1. 调用应用 API
  const response = await deviceRulesApi.applyRule(rule.id, {
    source_group_ids: selectedGroupIds,
  });

  // 2. 打开任务进度抽屉
  setCurrentTask(response);
  setTaskDrawerOpen(true);
  setApplyModalOpen(false);

  // 3. 轮询任务状态
  pollTaskStatus(response.task_id);
};
```

---

## 7. 异步任务机制

### 7.1 任务队列

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  API 请求   │ ──→ │  任务队列   │ ──→ │   Worker    │
│  (创建任务) │     │  (Channel)  │     │  (处理任务) │
└─────────────┘     └─────────────┘     └─────────────┘
```

### 7.2 Worker 启动

```go
// 在应用启动时启动 Worker
func StartRuleTaskWorkers(service *DeviceRuleService, workerCount int) {
    for i := 0; i < workerCount; i++ {
        go func(workerID int) {
            for taskID := range service.taskQueue {
                ctx := context.Background()
                service.processTask(ctx, taskID)
            }
        }(i)
    }
}
```

### 7.3 任务状态轮询

前端通过轮询 API 获取任务进度：

```typescript
const pollTaskStatus = async (taskId: string) => {
  const poll = async () => {
    const task = await deviceRulesApi.getTask(ruleId, taskId);
    setCurrentTask(task);

    if (task.status === 'running') {
      setTimeout(poll, 2000); // 每 2 秒轮询
    }
  };

  poll();
};
```

### 7.4 任务通知（可选扩展）

使用 WebSocket 或 SSE 实现实时推送：

```go
// SSE 端点
// GET /api/v1/device-rules/:id/tasks/:taskId/stream
func (h *Handler) StreamTaskProgress(c *gin.Context) {
    // 设置 SSE headers
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")

    // 推送任务进度
    for {
        task := getTaskProgress(taskID)
        c.SSEvent("progress", task)
        c.Flush()

        if task.Status == "completed" || task.Status == "failed" {
            break
        }

        time.Sleep(1 * time.Second)
    }
}
```

---

## 8. 运营商适配

### 8.1 匹配规则差异

| 运营商 | 设备名称规则 | LAC/TAC 使用 |
|--------|-------------|--------------|
| 中国移动 | 按省份前缀分组（BJ-、SH-） | LTE 使用 LAC，5G 使用 TAC |
| 中国电信 | 按城市编码分组 | 主要使用 TAC |
| 中国联通 | 按区域码分组 | LAC/TAC 混合使用 |

### 8.2 适配器扩展

```go
// carrier/adapter.go

type CarrierAdapter interface {
    // 设备规则相关
    GetDefaultMatchingMode() string
    ParseDeviceNamePattern(name string) (region, city string)
    GetLACTACSource() string // "tr069", "device_info", "config"
}

// cmcc/adapter.go
type CMCCAdapter struct {}

func (a *CMCCAdapter) GetDefaultMatchingMode() string {
    return "deviceName" // 移动默认按设备名称分组
}

// ctcc/adapter.go
type CTCCAdapter struct {}

func (a *CTCCAdapter) GetDefaultMatchingMode() string {
    return "tac" // 电信默认按 TAC 分组
}
```

---

## 9. 性能考量

### 9.1 大规模设备匹配

| 规模 | 策略 |
|------|------|
| < 1 万设备 | 同步匹配，直接遍历 |
| 1-10 万设备 | 异步任务，批量处理（每批 1000） |
| > 10 万设备 | 分片并行处理，使用 goroutine pool |

### 9.2 批量处理实现

```go
func (s *DeviceRuleService) processTask(ctx context.Context, taskID uuid.UUID) {
    // ...

    batchSize := 1000
    for i := 0; i < len(devices); i += batchSize {
        end := i + batchSize
        if end > len(devices) {
            end = len(devices)
        }

        batch := devices[i:end]

        var wg sync.WaitGroup
        for _, device := range batch {
            wg.Add(1)
            go func(d Device) {
                defer wg.Done()
                // 匹配并迁移
            }(device)
        }
        wg.Wait()

        // 更新进度
        task.MatchedCount = matchedCount
        s.taskRepo.Update(ctx, task)
    }
}
```

---

## 10. 安全与审计

### 10.1 权限控制

| 操作 | 权限码 | 说明 |
|------|--------|------|
| 查看规则 | `device:rule:read` | 查看规则列表和详情 |
| 创建规则 | `device:rule:create` | 创建新规则 |
| 编辑规则 | `device:rule:update` | 修改规则配置 |
| 删除规则 | `device:rule:delete` | 删除规则 |
| 应用规则 | `device:rule:apply` | 触发规则应用 |

### 10.2 审计日志

```go
// 记录关键操作
type RuleAuditLog struct {
    ID        uuid.UUID
    Action    string // create, update, delete, apply, toggle
    RuleID    uuid.UUID
    RuleName  string
    Operator  string
    Details   string // JSON 格式的变更详情
    Timestamp time.Time
}
```

---

## 11. 测试计划

### 11.1 单元测试

| 测试项 | 说明 |
|--------|------|
| 规则匹配逻辑 | 测试 AND/OR 组合条件 |
| 优先级排序 | 测试规则按优先级匹配 |
| 绑定关系 | 测试规则与分组的绑定/解绑 |

### 11.2 集成测试

| 测试项 | 说明 |
|--------|------|
| CRUD 流程 | 创建→编辑→删除规则 |
| 应用流程 | 应用→轮询→完成 |
| 分组覆盖 | 规则覆盖分组自身规则 |

### 11.3 性能测试

| 场景 | 指标 |
|------|------|
| 1 万设备匹配 | < 10 秒 |
| 10 万设备匹配 | < 2 分钟 |
| 并发应用任务 | 支持 5 个并发任务 |

---

## 12. 实施计划

### 12.1 阶段划分

| 阶段 | 内容 | 工期 |
|------|------|------|
| P1 | 数据库迁移、后端模型和仓储层 | 1 天 |
| P2 | 后端服务层和 API | 2 天 |
| P3 | 异步任务 Worker | 1 天 |
| P4 | 前端页面实现 | 2 天 |
| P5 | 测试和修复 | 1 天 |

### 12.2 依赖项

- 现有 `device_groups` 表和分组管理功能
- 现有 `DeviceMatcher` 匹配引擎
- 现有设备列表 API

---

## 13. 附录

### A. 错误码定义

```go
const (
    ErrCodeRuleNotFound        = 1201
    ErrCodeRulePriorityDuplicate = 1202
    ErrCodeRuleNotEnabled      = 1203
    ErrCodeRuleTaskNotFound    = 1204
    ErrCodeRuleTaskRunning     = 1205
)
```

### B. 数据库迁移文件

```
migrations/
├── 000075_create_device_rules.up.sql
├── 000075_create_device_rules.down.sql
├── 000076_create_device_rule_tasks.up.sql
└── 000076_create_device_rule_tasks.down.sql
```

### C. 相关文档

- [设备分组匹配规则待办](../todo-device-matching.md)
- [TR-069 协议对比分析](../analysis/tr069-protocol-comparison.md)
- [后端架构设计](../architecture/backend-design.md)
