# 参数模型操作 — 详细开发实施计划

> **日期**: 2026-03-24
> **设计文档**: `docs/design/parameter-model-analysis-and-rpc-design.md`
> **状态**: 已完成 (2026-03-24)

---

## 0. 现有基础设施盘点

在开始之前，明确哪些已经有了、哪些需要新建、哪些需要改造。

### 已有（可直接复用）

| 组件 | 文件 | 能力 |
|------|------|------|
| 设备参数表 | `migrations/000002` | `device_parameters(device_id, parameter_path, parameter_value, parameter_type, writable, last_updated_at)` |
| 参数仓储接口 | `device/param_repository.go` | `BatchUpsert`, `GetByDevice`, `GetByPath`, `DeleteByDevice` |
| 参数仓储实现 | `device/pg_param_repository.go` | pgx Batch UPSERT, Squirrel 查询 |
| 参数 Handler | `device/param_handler.go` | 6 个路由：tree/search/set/sync/discover/sync-status |
| 树形构建 | `device/param_handler.go:buildTree()` | 扁平参数列表 → ParameterTreeNode 层级树 |
| 设备服务 | `device/service.go` | `SetParameters()`（校验可写性 + 入队 SPV）|
| 命令队列 | `acs/cmdqueue/queue.go` | Redis Sorted Set，Push/Pop/Peek/Len/Clear |
| RPC 分发器 | `acs/rpc/dispatcher.go` | GPV/SPV/GPN/AddObject/DeleteObject 全部已实现 |
| 参数模型 | `config/datamodel/` | model/repository/registry/importer/xml_parser/cache 完备 |
| 参数同步 | `provision/sync.go` | SyncService.StartSync + HandleSyncResult + batchPaths |
| 模型解析 | `core/model/parameter.go` | DeviceParameter + ParameterDefinition 结构体 |

### 需要新建

| 组件 | 目标文件 | 职责 |
|------|---------|------|
| 参数树迭代器 | `config/datamodel/iterator.go` | 分析模型，提取多实例对象，生成 GPN/GPV 计划 |
| 参数校验器 | `config/datamodel/validator.go` | 基于模型校验参数值（类型+约束+可写性） |
| 路径匹配器 | `config/datamodel/path_matcher.go` | 模板路径 `{N}` ↔ 实际路径 `数字` 双向匹配 |
| 参数 Schema API | `device/param_handler.go` 新增路由 | 融合模型元数据+当前值，供前端构建表单 |
| 对象增删 API | `device/param_handler.go` 新增路由 | AddObject/DeleteObject + 最小实例数校验 |
| 最小实例规则表 | `config/datamodel/min_instances.go` | TR-196 规范的 minEntries 规则 |

### 需要改造

| 组件 | 文件 | 改造内容 |
|------|------|---------|
| Constraints | `config/datamodel/model.go` | 增加 `MinLength` 字段 |
| ObjectInfo | `config/datamodel/model.go` | 增加 `MinInstances` 字段 |
| buildConstraints | `config/datamodel/xml_parser.go` | STRING 类型解析 `min` 为 MinLength |
| 参数仓储接口 | `device/param_repository.go` | 增加 `GetByPathPrefix`, `CountByPathPrefix` |
| 参数仓储实现 | `device/pg_param_repository.go` | 实现新方法 |
| ParameterTreeHandler | `device/param_handler.go` | 注入 DataModelRegistry，增强 tree/set 端点 |
| SyncService.StartSync | `provision/sync.go` | 用迭代器替换 extractPathsFromParameterTree |
| engine.go | `provision/engine.go` | handleAutoSync 改用两阶段同步 |

---

## 1. Phase 1 — 模型补全与基础工具

**目标**: 修复模型缺陷，构建路径匹配和校验的基础工具。
**无外部依赖，可独立完成。**

### Step 1.1 — Constraints 增加 MinLength

**文件**: `omcgo/internal/config/datamodel/model.go`

```go
// 改造 Constraints 结构体
type Constraints struct {
    MinValue   *int64   `json:"min_value,omitempty"`
    MaxValue   *int64   `json:"max_value,omitempty"`
    EnumValues []string `json:"enum_values,omitempty"`
    Pattern    string   `json:"pattern,omitempty"`
    MaxLength  int      `json:"max_length,omitempty"`
    MinLength  int      `json:"min_length,omitempty"`   // 新增
}
```

**文件**: `omcgo/internal/config/datamodel/xml_parser.go`

```go
// 改造 buildConstraints — STRING 类型解析 min
func buildConstraints(xmlType, minStr, maxStr string) *Constraints {
    // ...
    if isString {
        if minStr != "" {
            if minLen, err := strconv.Atoi(minStr); err == nil && minLen > 0 {
                c.MinLength = minLen  // 新增
            }
        }
        // maxStr → MaxLength（不变）
    }
    // ...
}
```

**验证**: 修改 `xml_parser_test.go` 的 `TestBuildConstraints`，增加 MinLength 断言。

**工作量**: 小（~30 分钟）

---

### Step 1.2 — ObjectInfo 增加 MinInstances

**文件**: `omcgo/internal/config/datamodel/model.go`

```go
type ObjectInfo struct {
    Name         string `json:"name"`
    Access       string `json:"access"`
    MaxInstances int    `json:"max_instances"`
    MinInstances int    `json:"min_instances"`  // 新增
    IsList       bool   `json:"is_list"`
}
```

> 注意：MinInstances 不来自 XML 解析（XML 没有此属性），而是在后续 Step 1.4 由规则表填充。
> XML 解析时 MinInstances 默认为 0，表示"未设定，由规则表决定"。

**工作量**: 极小（~10 分钟）

---

### Step 1.3 — 路径匹配器 path_matcher.go

**新文件**: `omcgo/internal/config/datamodel/path_matcher.go`

这是后续所有功能（迭代器、校验器、Schema API）的基础工具。

```go
package datamodel

// 核心功能：
// 1. 判断路径是否包含多实例占位符 {N}
// 2. 模板路径 → 正则表达式（用于匹配实际路径）
// 3. 模板路径 → 基础路径（去掉 {N}，如 "Device.Services.FAPService."）
// 4. 实际路径 → 模板路径（反向匹配，如 "...FAPService.1." → "...FAPService.{12}."）
// 5. 提取路径中的实例编号

// ContainsPlaceholder 检查路径是否包含 {N} 占位符
func ContainsPlaceholder(path string) bool

// TemplateToBasePath 将 "Device.Services.FAPService.{12}." → "Device.Services.FAPService."
func TemplateToBasePath(templatePath string) string

// TemplateToRegex 将模板路径转为正则，用于匹配实际路径
// "...FAPService.{12}.CellConfig..." → "...FAPService\.\d+\.CellConfig..."
func TemplateToRegex(templatePath string) *regexp.Regexp

// ExtractInstanceNumbers 从实际路径提取所有实例编号
// "Device.Services.FAPService.2.CellConfig.LTE.EPC.PLMNList.3.PLMNID"
// → []{segment: "FAPService", instance: 2}, {segment: "PLMNList", instance: 3}}
func ExtractInstanceNumbers(actualPath string) []InstanceRef

// ReplaceInstance 将模板路径中的第 N 个 {X} 替换为实际实例编号
// ("...FAPService.{12}.CellConfig.LTE.EPC.PLMNList.{6}.", 0, 1)
// → "...FAPService.1.CellConfig.LTE.EPC.PLMNList.{6}."
func ReplaceInstance(templatePath string, placeholderIndex int, instanceNum int) string

type InstanceRef struct {
    Segment  string // 对象名，如 "FAPService"
    Instance int    // 实例编号，如 2
}
```

**新文件**: `omcgo/internal/config/datamodel/path_matcher_test.go`

覆盖场景：
- 单层占位符：`FAPService.{12}.`
- 嵌套占位符：`FAPService.{12}.PLMNList.{6}.`
- 无占位符路径
- 边界：`{0}` 表示无限制

**工作量**: 中（~2 小时）

---

### Step 1.4 — 最小实例数规则表 min_instances.go

**新文件**: `omcgo/internal/config/datamodel/min_instances.go`

```go
package datamodel

// GetMinInstances 根据对象路径和 TR-196/TR-181 规范返回最小实例数。
// 如果 ObjectInfo.MinInstances > 0（已显式设置），直接返回。
// 否则从内置规则表匹配。
func GetMinInstances(obj ObjectInfo) int

// 内置规则表（基于 TR-196 Issue 2, TR-181 Issue 2）
// 使用通配符模式匹配，按优先级排序（精确匹配优先）
var minInstancesRules = []minInstancesRule{
    // FAPService 至少 1 个（基站核心服务）
    {pattern: "Device.Services.FAPService.", min: 1},

    // PLMN 至少 1 个（网络接入必须）
    {pattern: "*.PLMNList.", min: 1},

    // 邻区可以全部删除
    {pattern: "*.NeighborList.LTECell.", min: 0},
    {pattern: "*.NeighborList.InterRATCell.*.", min: 0},

    // NR 切片至少 1 个
    {pattern: "*.SNSSAI.", min: 1},

    // SCTP 关联可以全部删除
    {pattern: "*.Transport.SCTP.Assoc.", min: 0},

    // Tunnel/IPSec 至少 1 个（网络传输必须）
    {pattern: "*.Tunnel.", min: 1},
}

// ApplyMinInstances 批量为 ObjectInfo 列表填充 MinInstances。
// 在模型加载后调用一次。
func ApplyMinInstances(objects []ObjectInfo) []ObjectInfo
```

**工作量**: 小（~1 小时）

---

### Step 1.5 — 参数校验器 validator.go

**新文件**: `omcgo/internal/config/datamodel/validator.go`

```go
package datamodel

// ParameterValidator 基于参数模型对参数值进行校验。
// 在 DataModel 加载后构建一次，可复用于多次校验。
type ParameterValidator struct {
    params       []Parameter         // 模型参数列表
    objects      []ObjectInfo        // 模型对象列表
    paramIndex   map[string]int      // 精确路径 → params 索引（非多实例参数）
    paramRegexes []paramRegexEntry   // 多实例参数的正则匹配器（预编译）
    objectIndex  map[string]int      // 对象路径 → objects 索引
    objectRegexes []objectRegexEntry // 多实例对象的正则匹配器
}

type paramRegexEntry struct {
    regex *regexp.Regexp
    index int // params 中的索引
}

// NewParameterValidator 从 DataModel 构建校验器。
func NewParameterValidator(dm *DataModel) (*ParameterValidator, error)

// ValidationError 校验错误。
type ValidationError struct {
    Path    string `json:"path"`
    Rule    string `json:"rule"`     // exists/writable/type/constraint
    Message string `json:"message"`
}

// ValidateValue 校验单个参数的值。
// 检查顺序：路径存在 → 可写性 → 类型匹配 → 约束满足
func (v *ParameterValidator) ValidateValue(path, value string) *ValidationError

// ValidateValues 批量校验，返回所有错误。
func (v *ParameterValidator) ValidateValues(params []struct{Path, Value string}) []ValidationError

// LookupParam 查找参数定义（支持多实例路径匹配）。
// 返回 nil 表示不存在。
func (v *ParameterValidator) LookupParam(actualPath string) *Parameter

// LookupObject 查找对象定义。
func (v *ParameterValidator) LookupObject(actualPath string) *ObjectInfo

// ValidateAddObject 校验是否允许添加实例。
func (v *ParameterValidator) ValidateAddObject(objectPath string, currentCount int) *ValidationError

// ValidateDeleteObject 校验是否允许删除实例。
func (v *ParameterValidator) ValidateDeleteObject(objectPath string, currentCount int) *ValidationError

// --- 内部函数 ---

// validateType 按 Type 校验值的格式
func validateType(paramType, value string) error
// validateConstraints 校验约束（MinValue/MaxValue/EnumValues/Pattern/MaxLength/MinLength）
func validateConstraints(paramType string, c *Constraints, value string) error
```

**新文件**: `omcgo/internal/config/datamodel/validator_test.go`

覆盖场景（table-driven tests）：
- 各种 Type 的合法/非法值
- MinValue/MaxValue 边界
- EnumValues 命中/未命中
- Pattern 匹配/不匹配
- MaxLength/MinLength 超限
- 只读参数拒绝修改
- 多实例路径匹配
- AddObject 超出 MaxInstances
- DeleteObject 低于 MinInstances

**依赖**: Step 1.1, 1.2, 1.3, 1.4

**工作量**: 大（~4 小时）

---

## 2. Phase 2 — 参数树迭代器与同步改造

**目标**: 正确获取设备全部参数（解决多实例问题）。

### Step 2.1 — 参数树迭代器 iterator.go

**新文件**: `omcgo/internal/config/datamodel/iterator.go`

```go
package datamodel

// ParameterTreeIterator 分析参数模型，生成获取设备实际参数的执行计划。
type ParameterTreeIterator struct {
    objects    []ObjectInfo
    parameters []Parameter
    multiInst  []MultiInstanceObject  // 分析后的多实例对象（按层级排序）
}

// MultiInstanceObject 描述一个多实例对象节点。
type MultiInstanceObject struct {
    TemplatePath string  // "Device.Services.FAPService.{12}."
    BasePath     string  // "Device.Services.FAPService."
    MaxInstances int
    MinInstances int
    Writable     bool    // access == READ_WRITE
    Depth        int     // 嵌套深度（0=顶层，1=二级...）
    ParentIdx    int     // 父多实例对象在 multiInst 中的索引，-1=无父
}

// NewParameterTreeIterator 从 DataModel 构建迭代器。
func NewParameterTreeIterator(dm *DataModel) (*ParameterTreeIterator, error)

// SyncPlan 描述获取设备全部参数的两阶段执行计划。
type SyncPlan struct {
    // Phase1 GPN 请求：发现多实例对象的实际实例
    // 按层级排序：先顶层，再嵌套层（因为嵌套层的路径依赖上层的实际实例编号）
    Phase1GPNs []GPNRequest

    // Phase2 GPV 前缀：获取参数值
    // 在 Phase1 完成后，调用 BuildGPVPrefixes() 动态生成
    StaticPrefixes []string  // 非多实例的对象前缀（如 "Device.DeviceInfo."）
}

type GPNRequest struct {
    TemplatePath string // 模板路径（可能含 {N}，需要上层实例展开后替换）
    BasePath     string // GPN 实际请求路径
    Depth        int    // 嵌套深度
    NextLevel    bool   // 是否只获取直接子节点（始终为 true）
}

// BuildSyncPlan 生成同步计划。
func (it *ParameterTreeIterator) BuildSyncPlan() *SyncPlan

// InstanceMap 保存 GPN 发现的实际实例映射。
// key: 对象基础路径（如 "Device.Services.FAPService."）
// value: 实际实例编号列表（如 [1, 2, 3]）
type InstanceMap map[string][]int

// ExpandGPNsForDepth 根据上层 GPN 结果，展开指定深度的嵌套 GPN 请求。
// depth=0 的 GPN 请求可以直接发送（BasePath 不含占位符）。
// depth=1 的 GPN 请求需要用 depth=0 的实例结果替换占位符后才能发送。
// 依此类推。
func (it *ParameterTreeIterator) ExpandGPNsForDepth(plan *SyncPlan, depth int, instances InstanceMap) []GPNRequest

// BuildGPVPrefixes 在所有 GPN 完成后，生成 GPV 部分路径前缀列表。
// 合并静态前缀 + 多实例的实际实例前缀。
func (it *ParameterTreeIterator) BuildGPVPrefixes(plan *SyncPlan, instances InstanceMap) []string

// GetMultiInstanceObjects 返回分析后的多实例对象列表。
func (it *ParameterTreeIterator) GetMultiInstanceObjects() []MultiInstanceObject
```

**新文件**: `omcgo/internal/config/datamodel/iterator_test.go`

测试用例：
1. 无多实例对象的模型 → Phase1 为空，StaticPrefixes 包含所有对象前缀
2. 单层多实例（FAPService.{12}.）→ Phase1 有 1 个 GPN
3. 嵌套多实例（FAPService.{12}.PLMNList.{6}.）→ Phase1 有 2 层 GPN
4. 三层嵌套 → 正确处理
5. BuildGPVPrefixes 正确合并静态+动态前缀
6. ExpandGPNsForDepth 正确展开

**依赖**: Step 1.3 (path_matcher)

**工作量**: 大（~5 小时）

---

### Step 2.2 — 改造 SyncService 使用迭代器

**文件**: `omcgo/internal/provision/sync.go`

当前问题：`StartSync` 接收扁平路径列表并直接发 GPV。路径来自 `extractPathsFromParameterTree`，包含 `{N}` 占位符。

**改造方案**：

```go
// 1. 新增方法：两阶段同步入口
func (s *SyncService) StartTwoPhaseSync(ctx context.Context, dev *model.Device, dm *datamodel.DataModel) error {
    // 构建迭代器
    iterator, err := datamodel.NewParameterTreeIterator(dm)

    // 生成同步计划
    plan := iterator.BuildSyncPlan()

    // 如果无多实例对象，直接用静态前缀发 GPV（简化路径）
    if len(plan.Phase1GPNs) == 0 {
        return s.enqueueGPVBatch(ctx, dev, plan.StaticPrefixes)
    }

    // 有多实例：先入队 Phase1 GPN 请求（depth=0 的）
    depth0GPNs := filterByDepth(plan.Phase1GPNs, 0)
    for _, gpn := range depth0GPNs {
        s.enqueueGPN(ctx, dev, gpn)
    }

    // 保存 plan 到 Redis（供 GPN 响应处理时继续）
    s.saveSyncPlan(ctx, dev.SerialNumber, plan, iterator)

    return nil
}

// 2. 新增方法：处理 GPN 响应，推进同步流程
func (s *SyncService) HandleGPNResult(ctx context.Context, dev *model.Device,
    gpnPath string, instances []string) error {

    // 从 Redis 加载同步计划和实例映射
    plan, iterator, instanceMap := s.loadSyncPlan(ctx, dev.SerialNumber)

    // 记录发现的实例
    instanceNums := parseInstanceNumbers(instances)
    instanceMap[gpnPath] = instanceNums

    // 检查是否有更深层的 GPN 需要展开
    nextGPNs := iterator.ExpandGPNsForDepth(plan, currentDepth+1, instanceMap)
    if len(nextGPNs) > 0 {
        // 入队下一层 GPN
        for _, gpn := range nextGPNs {
            s.enqueueGPN(ctx, dev, gpn)
        }
        s.saveSyncPlan(ctx, dev.SerialNumber, plan, iterator)  // 更新状态
        return nil
    }

    // 所有 GPN 完成，生成 GPV 前缀并入队
    gpvPrefixes := iterator.BuildGPVPrefixes(plan, instanceMap)
    s.enqueueGPVBatch(ctx, dev, gpvPrefixes)

    // 清理 Redis 中的同步计划
    s.clearSyncPlan(ctx, dev.SerialNumber)

    return nil
}

// 3. 保留原 StartSync 方法（向后兼容，供手动同步使用）
// 但内部逻辑改为用部分路径前缀而非逐参数路径
```

**Redis 临时状态键**: `provision:sync_plan:{device_sn}` (TTL: 10 分钟)

**文件**: `omcgo/internal/provision/engine.go`

```go
// 改造 handleAutoSync — 使用 StartTwoPhaseSync
func (e *ProvisioningEngine) handleAutoSync(ctx context.Context, task *ProvisioningTask,
    dev *model.Device, dm *datamodel.DataModel) error {
    // ...
    // 旧: paramPaths, err := extractPathsFromParameterTree(dm.ParameterTree)
    //     s.syncService.StartSync(ctx, dev, paramPaths)
    // 新:
    if err := e.syncService.StartTwoPhaseSync(ctx, dev, dm); err != nil {
        return e.failTask(ctx, task, fmt.Errorf("start two-phase sync: %w", err))
    }
    // ...
}
```

**文件**: `omcgo/internal/provision/engine.go` — 新增 GPN 响应订阅

```go
// Subscribe 中新增:
bus.QueueSubscribe(event.SubjectCommandGetNamesResponse, "provision-gpn", func(...) {
    return e.handleGPNResponse(ctx, evt)
})

func (e *ProvisioningEngine) handleGPNResponse(ctx context.Context, evt event.Event) error {
    // 解码 GPN 响应 payload
    // 调用 syncService.HandleGPNResult()
}
```

**需要确认**: ACS 端 GPN 响应是否已发布事件。如果没有，需要在 ACS 的 session handler 中增加 GPN 响应事件发布。

**依赖**: Step 2.1

**工作量**: 大（~6 小时）

---

### Step 2.3 — ACS 端 GPN 响应事件发布（如需要）

检查 ACS session handler 中 GetParameterNamesResponse 的处理。如果尚未发布事件，需要新增。

**文件**: `omcgo/internal/acs/session/handler.go`（或类似文件）

```go
// 在 GetParameterNamesResponse 处理完成后，发布事件:
event.SubjectCommandGetNamesResponse = "command.get_names.response"

// Payload:
type gpnResponsePayload struct {
    DeviceSN   string   `json:"device_sn"`
    Path       string   `json:"path"`        // 请求的路径
    NextLevel  bool     `json:"next_level"`
    Parameters []struct {
        Name     string `json:"name"`
        Writable bool   `json:"writable"`
    } `json:"parameters"`
}
```

**文件**: `omcgo/internal/core/event/subjects.go`

```go
// 新增事件主题（如果不存在）
SubjectCommandGetNamesResponse = "command.get_names.response"
```

**工作量**: 中（~2 小时，需要仔细阅读 ACS session 代码）

---

## 3. Phase 3 — 参数仓储扩展与 Schema API

**目标**: 前端可以浏览设备参数（含模型元数据）。

### Step 3.1 — 参数仓储增加前缀查询

**文件**: `omcgo/internal/device/param_repository.go`

```go
type DeviceParameterRepository interface {
    BatchUpsert(ctx context.Context, deviceID uuid.UUID, params []model.DeviceParameter) error
    GetByDevice(ctx context.Context, deviceID uuid.UUID) ([]model.DeviceParameter, error)
    GetByPath(ctx context.Context, deviceID uuid.UUID, path string) (*model.DeviceParameter, error)
    DeleteByDevice(ctx context.Context, deviceID uuid.UUID) error

    // 新增：
    GetByPathPrefix(ctx context.Context, deviceID uuid.UUID, prefix string) ([]model.DeviceParameter, error)
    CountByPathPrefix(ctx context.Context, deviceID uuid.UUID, prefix string) (int, error)
    SearchByKeyword(ctx context.Context, deviceID uuid.UUID, keyword string, limit int) ([]model.DeviceParameter, error)
}
```

**文件**: `omcgo/internal/device/pg_param_repository.go`

```go
func (r *PgDeviceParameterRepository) GetByPathPrefix(ctx context.Context, deviceID uuid.UUID, prefix string) ([]model.DeviceParameter, error) {
    query, args, err := psql.Select(...).
        From("device_parameters").
        Where(sq.Eq{"device_id": deviceID}).
        Where(sq.Like{"parameter_path": prefix + "%"}).
        OrderBy("parameter_path ASC").
        ToSql()
    // ...
}

func (r *PgDeviceParameterRepository) CountByPathPrefix(ctx context.Context, deviceID uuid.UUID, prefix string) (int, error) {
    query, args, err := psql.Select("COUNT(*)").
        From("device_parameters").
        Where(sq.Eq{"device_id": deviceID}).
        Where(sq.Like{"parameter_path": prefix + "%"}).
        ToSql()
    // ...
}

func (r *PgDeviceParameterRepository) SearchByKeyword(ctx context.Context, deviceID uuid.UUID, keyword string, limit int) ([]model.DeviceParameter, error) {
    query, args, err := psql.Select(...).
        From("device_parameters").
        Where(sq.Eq{"device_id": deviceID}).
        Where(sq.ILike{"parameter_path": "%" + keyword + "%"}).
        OrderBy("parameter_path ASC").
        Limit(uint64(limit)).
        ToSql()
    // ...
}
```

**数据库索引**（新迁移 `000061_device_parameters_path_index.up.sql`）：

```sql
-- 加速前缀查询（LIKE 'prefix%' 可以利用 btree 索引）
-- parameter_path 已是 PK 的一部分，但单独索引加速 prefix 查询
CREATE INDEX IF NOT EXISTS idx_device_params_path_prefix
ON device_parameters (device_id, parameter_path varchar_pattern_ops);
```

**工作量**: 中（~2 小时）

---

### Step 3.2 — ParameterTreeHandler 注入 DataModelRegistry

当前 `ParameterTreeHandler` 只依赖 `DeviceService` 和 `DeviceParameterRepository`。
需要注入 `DataModelRegistry` 才能查询参数模型元数据。

**文件**: `omcgo/internal/device/param_handler.go`

```go
type ParameterTreeHandler struct {
    deviceService *DeviceService
    paramRepo     DeviceParameterRepository
    dmRegistry    *datamodel.DataModelRegistry  // 新增
    logger        *zap.Logger
}

func NewParameterTreeHandler(
    deviceService *DeviceService,
    paramRepo DeviceParameterRepository,
    dmRegistry *datamodel.DataModelRegistry,  // 新增
    logger *zap.Logger,
) *ParameterTreeHandler
```

**文件**: `omcgo/cmd/app/router/router.go` — 更新构造函数调用

```go
// 旧: paramTreeHandler := device.NewParameterTreeHandler(deviceService, paramRepo, logger)
// 新:
paramTreeHandler := device.NewParameterTreeHandler(deviceService, paramRepo, dmRegistry, logger)
```

**工作量**: 小（~30 分钟）

---

### Step 3.3 — Parameter Schema API

**文件**: `omcgo/internal/device/param_handler.go` — 新增路由和 handler

```go
// RegisterRoutes 中新增:
devices.GET("/:id/parameters/schema", h.GetParameterSchema)

// GetParameterSchema 融合模型元数据+当前值，供前端构建编辑表单。
// GET /api/v1/devices/:id/parameters/schema?path_prefix=Device.Services.FAPService.1.
func (h *ParameterTreeHandler) GetParameterSchema(c *gin.Context) {
    // 1. 解析 device ID
    // 2. 查询设备信息（获取 carrier, tech, oui, productClass, firmwareVersion）
    // 3. 通过 dmRegistry.ResolveForDevice(dev) 获取匹配的 DataModel
    // 4. 构建 ParameterValidator（获取参数定义+约束信息）
    // 5. 查询 device_parameters 获取当前值（按 path_prefix 过滤）
    // 6. 合并：模型定义 + 当前值 → ParameterSchemaItem 列表
    // 7. 从 object_tree 提取该路径下的多实例对象信息
    // 8. 返回 {parameters: [...], objects: [...]}
}
```

**响应结构体**：

```go
// ParameterSchemaItem 合并了模型定义和设备当前值。
type ParameterSchemaItem struct {
    Path         string             `json:"path"`
    Type         string             `json:"type"`
    Writable     bool               `json:"writable"`
    Description  string             `json:"description,omitempty"`
    DefaultValue string             `json:"default_value,omitempty"`
    Notify       string             `json:"notify,omitempty"`
    ForcedInform bool               `json:"forced_inform,omitempty"`
    ChangeApplies string            `json:"change_applies,omitempty"`
    Category     string             `json:"category,omitempty"`
    IsList       bool               `json:"is_list,omitempty"`
    Constraints  *datamodel.Constraints `json:"constraints,omitempty"`

    // 设备当前状态
    CurrentValue *string            `json:"current_value"`       // null = 未同步
    LastSyncedAt *time.Time         `json:"last_synced_at,omitempty"`
}

// ObjectSchemaItem 对象节点信息（供前端显示增删按钮）。
type ObjectSchemaItem struct {
    Path             string `json:"path"`
    Access           string `json:"access"`
    MaxInstances     int    `json:"max_instances"`
    MinInstances     int    `json:"min_instances"`
    CurrentInstances []int  `json:"current_instances"` // 从 device_parameters 分析
    CanAdd           bool   `json:"can_add"`
    CanDeleteAny     bool   `json:"can_delete_any"`
    IsList           bool   `json:"is_list"`
}
```

**核心逻辑 — 合并模型与当前值**：

```go
func mergeSchemaWithValues(
    validator *datamodel.ParameterValidator,
    params []model.DeviceParameter,
    pathPrefix string,
) []ParameterSchemaItem {
    // 1. 建立当前值索引: path → DeviceParameter
    valueMap := make(map[string]model.DeviceParameter)
    for _, p := range params {
        valueMap[p.ParameterPath] = p
    }

    // 2. 遍历当前值中 pathPrefix 下的参数
    var items []ParameterSchemaItem
    for _, p := range params {
        if !strings.HasPrefix(p.ParameterPath, pathPrefix) {
            continue
        }
        item := ParameterSchemaItem{
            Path:         p.ParameterPath,
            CurrentValue: &p.ParameterValue,
            LastSyncedAt: &p.LastUpdatedAt,
        }

        // 3. 从模型查找定义并填充元数据
        if def := validator.LookupParam(p.ParameterPath); def != nil {
            item.Type = def.Type
            item.Writable = def.Writable
            item.Description = def.Description
            item.DefaultValue = def.DefaultValue
            item.Notify = def.Notify
            item.ForcedInform = def.ForcedInform
            item.ChangeApplies = def.ChangeApplies
            item.Category = def.Category
            item.IsList = def.IsList
            item.Constraints = def.Constraints
        } else {
            // 模型中没有定义（可能是模型过时），用 device_parameters 的基本信息
            item.Type = string(p.ParameterType)
            item.Writable = p.Writable
        }

        items = append(items, item)
    }
    return items
}
```

**依赖**: Step 1.5 (validator), Step 3.1 (GetByPathPrefix), Step 3.2 (注入 dmRegistry)

**工作量**: 大（~5 小时）

---

### Step 3.4 — 增强 GetParameterTree 端点

当前 `GetParameterTree` 只返回 name/fullPath/isLeaf/value/type/writable。
需要增加模型元数据，特别是多实例对象的信息。

**文件**: `omcgo/internal/device/param_handler.go`

```go
// 增强 ParameterTreeNode
type ParameterTreeNode struct {
    Name          string               `json:"name"`
    FullPath      string               `json:"full_path"`
    IsLeaf        bool                 `json:"is_leaf"`
    Value         string               `json:"value,omitempty"`
    Type          string               `json:"type,omitempty"`
    Writable      bool                 `json:"writable"`
    Children      []*ParameterTreeNode `json:"children,omitempty"`

    // 新增模型元数据
    Description   string               `json:"description,omitempty"`
    MultiInstance bool                 `json:"multi_instance,omitempty"`
    MaxInstances  int                  `json:"max_instances,omitempty"`
    MinInstances  int                  `json:"min_instances,omitempty"`
    InstanceCount int                  `json:"instance_count,omitempty"` // 当前实例数
    CanAdd        bool                 `json:"can_add,omitempty"`
    CanDelete     bool                 `json:"can_delete,omitempty"`
    ChangeApplies string               `json:"change_applies,omitempty"`
    DefaultValue  string               `json:"default_value,omitempty"`
    Constraints   *datamodel.Constraints `json:"constraints,omitempty"`
}

// 增强 GetParameterTree: 查询参数模型，注入元数据
func (h *ParameterTreeHandler) GetParameterTree(c *gin.Context) {
    // ... 现有逻辑 ...

    // 新增: 获取设备对应的参数模型
    dev, _ := h.deviceService.GetDevice(ctx, id)
    if dev != nil && h.dmRegistry != nil {
        dm, _ := h.dmRegistry.ResolveForDevice(ctx, dev)
        if dm != nil {
            validator, _ := datamodel.NewParameterValidator(dm)
            enrichTreeWithModel(tree, validator)  // 给树节点注入模型元数据
        }
    }
    // ...
}

// enrichTreeWithModel 递归遍历树节点，从模型注入元数据。
func enrichTreeWithModel(nodes []*ParameterTreeNode, v *datamodel.ParameterValidator) {
    for _, node := range nodes {
        if node.IsLeaf {
            if def := v.LookupParam(node.FullPath); def != nil {
                node.Description = def.Description
                node.ChangeApplies = def.ChangeApplies
                node.DefaultValue = def.DefaultValue
                node.Constraints = def.Constraints
            }
        } else {
            if obj := v.LookupObject(node.FullPath + "."); obj != nil {
                node.MultiInstance = obj.MaxInstances > 0 || ContainsPlaceholder(obj.Name)
                node.MaxInstances = obj.MaxInstances
                node.MinInstances = obj.MinInstances
                node.InstanceCount = len(node.Children)
                node.CanAdd = obj.Access == "READ_WRITE" &&
                    (obj.MaxInstances == 0 || node.InstanceCount < obj.MaxInstances)
                node.CanDelete = obj.Access == "READ_WRITE" &&
                    node.InstanceCount > obj.MinInstances
            }
        }
        enrichTreeWithModel(node.Children, v)
    }
}
```

**依赖**: Step 1.5, Step 3.2

**工作量**: 中（~3 小时）

---

## 4. Phase 4 — 参数修改增强（模型校验）

**目标**: 修改参数时用模型做完整校验。

### Step 4.1 — 增强 SetParameterValues 端点

当前 `SetParameterValues` 仅校验 device_parameters 表中的 writable 标记。
需要增加基于模型的完整校验。

**文件**: `omcgo/internal/device/param_handler.go`

```go
func (h *ParameterTreeHandler) SetParameterValues(c *gin.Context) {
    // ... 现有: 解析 ID、绑定 JSON、查询设备 ...

    // 新增: 基于模型的完整校验
    if h.dmRegistry != nil {
        dm, _ := h.dmRegistry.ResolveForDevice(c.Request.Context(), dev)
        if dm != nil {
            validator, err := datamodel.NewParameterValidator(dm)
            if err == nil {
                // 校验每个参数
                var validationErrors []datamodel.ValidationError
                for _, item := range req.Parameters {
                    if ve := validator.ValidateValue(item.Path, item.Value); ve != nil {
                        validationErrors = append(validationErrors, *ve)
                    }
                }
                if len(validationErrors) > 0 {
                    c.JSON(http.StatusBadRequest, gin.H{
                        "error":             "parameter validation failed",
                        "validation_errors": validationErrors,
                    })
                    return
                }
            }
        }
    }

    // 检查是否包含需要重启才生效的参数，在响应中提示
    var rebootRequired bool
    if validator != nil {
        for _, item := range req.Parameters {
            if def := validator.LookupParam(item.Path); def != nil {
                if def.ChangeApplies == "RebootRequired" || def.ChangeApplies == "NotifyRequired" {
                    rebootRequired = true
                    break
                }
            }
        }
    }

    // ... 现有: 入队 SPV ...

    c.JSON(http.StatusAccepted, gin.H{
        "message":          "set parameter values command queued",
        "parameters":       len(req.Parameters),
        "reboot_required":  rebootRequired,  // 新增: 前端可据此提示用户
    })
}
```

**依赖**: Step 1.5, Step 3.2

**工作量**: 中（~2 小时）

---

## 5. Phase 5 — 对象增删 API

**目标**: 前端可以添加/删除多实例对象实例。

### Step 5.1 — AddObject / DeleteObject 端点

**文件**: `omcgo/internal/device/param_handler.go` — 新增路由

```go
// RegisterRoutes 中新增:
devices.POST("/:id/objects/add", h.AddObject)
devices.POST("/:id/objects/delete", h.DeleteObject)
```

```go
// AddObjectRequest 添加对象请求。
type AddObjectRequest struct {
    ObjectPath string `json:"object_path" binding:"required"`  // 如 "Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList."
}

// AddObject 处理 POST /api/v1/devices/:id/objects/add
func (h *ParameterTreeHandler) AddObject(c *gin.Context) {
    // 1. 解析设备 ID、绑定 JSON
    // 2. 查询设备
    // 3. 获取模型并构建校验器
    // 4. 统计当前实例数:
    //    countPrefix = objectPath  // 如 "...PLMNList."
    //    currentCount = h.paramRepo.CountByPathPrefix(ctx, deviceID, countPrefix) 中去重实例编号
    //    更准确的方式: 从 device_parameters 查 countPrefix 下的直接子实例编号
    // 5. validator.ValidateAddObject(objectPath, currentCount)
    // 6. 校验通过 → 入队 AddObject RPC

    addParams, _ := json.Marshal(map[string]interface{}{
        "object_name": req.ObjectPath,
    })
    cmd := &cmdqueue.Command{
        ID:       uuid.New().String(),
        Method:   "AddObject",
        Params:   addParams,
        Priority: 3,  // 高于同步，低于重启
    }
    h.deviceService.GetCommandQueue().Push(ctx, dev.SerialNumber, cmd)

    c.JSON(http.StatusAccepted, gin.H{"message": "add object command queued"})
}

// DeleteObjectRequest 删除对象请求。
type DeleteObjectRequest struct {
    ObjectPath string `json:"object_path" binding:"required"`  // 如 "Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.3."
}

// DeleteObject 处理 POST /api/v1/devices/:id/objects/delete
func (h *ParameterTreeHandler) DeleteObject(c *gin.Context) {
    // 1. 解析设备 ID、绑定 JSON
    // 2. 查询设备
    // 3. 获取模型并构建校验器
    // 4. 从 objectPath 提取父对象路径（去掉最后的实例编号）
    //    "...PLMNList.3." → parentPath = "...PLMNList."
    // 5. 统计当前实例数（同 AddObject）
    // 6. validator.ValidateDeleteObject(parentPath, currentCount)
    // 7. 校验通过 → 入队 DeleteObject RPC

    delParams, _ := json.Marshal(map[string]interface{}{
        "object_name": req.ObjectPath,
    })
    cmd := &cmdqueue.Command{
        ID:       uuid.New().String(),
        Method:   "DeleteObject",
        Params:   delParams,
        Priority: 3,
    }
    h.deviceService.GetCommandQueue().Push(ctx, dev.SerialNumber, cmd)

    c.JSON(http.StatusAccepted, gin.H{"message": "delete object command queued"})
}
```

**辅助函数 — 统计实例数**：

```go
// countInstances 从 device_parameters 中统计某对象路径下的实际实例数。
// 例如: prefix = "Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList."
// 会找到 PLMNList.1.*, PLMNList.2.*, PLMNList.3.* → 返回 3
func (h *ParameterTreeHandler) countInstances(ctx context.Context, deviceID uuid.UUID, objectPrefix string) (int, []int) {
    params, _ := h.paramRepo.GetByPathPrefix(ctx, deviceID, objectPrefix)
    instanceSet := make(map[int]bool)
    for _, p := range params {
        suffix := strings.TrimPrefix(p.ParameterPath, objectPrefix)
        // suffix 形如 "1.PLMNID", "2.PLMNID", "3.Enable"
        parts := strings.SplitN(suffix, ".", 2)
        if num, err := strconv.Atoi(parts[0]); err == nil {
            instanceSet[num] = true
        }
    }
    instances := make([]int, 0, len(instanceSet))
    for num := range instanceSet {
        instances = append(instances, num)
    }
    sort.Ints(instances)
    return len(instances), instances
}
```

**依赖**: Step 1.5, Step 3.1, Step 3.2

**工作量**: 中（~4 小时）

---

## 6. Phase 6 — 前端参数浏览页面

**目标**: 前端实现设备参数浏览、编辑、增删功能。

> 注: 前端工作在 `omcmb/` 子仓库中进行。

### Step 6.1 — API 服务层

**新文件**: `omcmb/webcode/src/services/api/parameterApi.ts`

```typescript
export const parameterApi = {
  // 获取参数树
  getParameterTree: (deviceId: string, format?: 'tree' | 'flat') =>
    http.get(`/devices/${deviceId}/parameters/tree`, { params: { format } }),

  // 获取参数 Schema（含模型元数据+当前值）
  getParameterSchema: (deviceId: string, pathPrefix?: string) =>
    http.get(`/devices/${deviceId}/parameters/schema`, { params: { path_prefix: pathPrefix } }),

  // 搜索参数
  searchParameters: (deviceId: string, keyword: string) =>
    http.get(`/devices/${deviceId}/parameters/search`, { params: { q: keyword } }),

  // 修改参数值
  setParameterValues: (deviceId: string, params: { path: string; value: string }[]) =>
    http.put(`/devices/${deviceId}/parameters`, { parameters: params }),

  // 触发同步
  triggerSync: (deviceId: string) =>
    http.post(`/devices/${deviceId}/parameters/sync`),

  // 获取同步状态
  getSyncStatus: (deviceId: string) =>
    http.get(`/devices/${deviceId}/parameters/sync-status`),

  // 添加对象实例
  addObject: (deviceId: string, objectPath: string) =>
    http.post(`/devices/${deviceId}/objects/add`, { object_path: objectPath }),

  // 删除对象实例
  deleteObject: (deviceId: string, objectPath: string) =>
    http.post(`/devices/${deviceId}/objects/delete`, { object_path: objectPath }),
}
```

### Step 6.2 — React Query Hook

**新文件**: `omcmb/webcode/src/hooks/api/useDeviceParameters.ts`

```typescript
// useParameterTree — 获取参数树
// useParameterSchema — 获取参数 Schema
// useSetParameterValues — mutation: 修改参数
// useTriggerSync — mutation: 触发同步
// useSyncStatus — 轮询同步状态
// useAddObject — mutation: 添加对象
// useDeleteObject — mutation: 删除对象
```

### Step 6.3 — 页面组件

**新文件**: `omcmb/webcode/src/pages/device/parameters/`

```
DeviceParametersPage.tsx      # 主页面：参数树 + 搜索 + 同步按钮
├── ParameterTree.tsx         # 树形组件（Ant Design Tree 或自定义虚拟树）
├── ParameterEditor.tsx       # 参数编辑器（根据 type/constraints 动态渲染控件）
├── ObjectActions.tsx         # 对象增删按钮（CanAdd/CanDelete 控制）
├── ParameterSearch.tsx       # 参数搜索框
└── SyncStatusBar.tsx         # 同步状态条
```

**ParameterEditor 核心逻辑**（参照设计文档 8.4 节）：
- `boolean` → Switch
- 有 `enum_values` → Select
- `int/unsignedInt/long/unsignedLong` → InputNumber (min/max)
- `dateTime` → DatePicker
- `string` → Input (maxLength, placeholder=defaultValue)
- `change_applies=RebootRequired` → 橙色警告标签

### Step 6.4 — 路由注册

**文件**: `omcmb/webcode/src/router/` — 在设备详情下新增参数 Tab

**工作量**: 大（~8-10 小时，前端 3 个步骤合计）

---

## 7. 实施排期与依赖关系

```
Phase 1: 模型补全与基础工具
├── Step 1.1 Constraints.MinLength        (独立, ~30min)
├── Step 1.2 ObjectInfo.MinInstances      (独立, ~10min)
├── Step 1.3 path_matcher.go              (独立, ~2h)
├── Step 1.4 min_instances.go             (依赖 1.2, ~1h)
└── Step 1.5 validator.go                 (依赖 1.1+1.2+1.3+1.4, ~4h)

Phase 2: 迭代器与同步改造
├── Step 2.1 iterator.go                  (依赖 1.3, ~5h)
├── Step 2.2 改造 SyncService             (依赖 2.1, ~6h)
└── Step 2.3 ACS GPN 事件发布             (独立, ~2h)

Phase 3: Schema API
├── Step 3.1 仓储扩展 + 迁移              (独立, ~2h)
├── Step 3.2 Handler 注入 dmRegistry      (独立, ~30min)
├── Step 3.3 Parameter Schema API         (依赖 1.5+3.1+3.2, ~5h)
└── Step 3.4 增强 GetParameterTree        (依赖 1.5+3.2, ~3h)

Phase 4: 参数修改增强
└── Step 4.1 增强 SetParameterValues      (依赖 1.5+3.2, ~2h)

Phase 5: 对象增删
└── Step 5.1 AddObject + DeleteObject     (依赖 1.5+3.1+3.2, ~4h)

Phase 6: 前端
├── Step 6.1 parameterApi.ts              (独立, ~1h)
├── Step 6.2 useDeviceParameters.ts       (依赖 6.1, ~1h)
├── Step 6.3 页面组件                      (依赖 6.2, ~6h)
└── Step 6.4 路由注册                      (依赖 6.3, ~1h)
```

### 依赖关系图

```
1.1─┐
1.2─┤
1.3─┼─→ 1.5 ─┬─→ 3.3 ─→ (Phase 3 完成)
1.4─┘         ├─→ 3.4
              ├─→ 4.1
              └─→ 5.1

1.3 ──→ 2.1 ──→ 2.2 ──→ (Phase 2 完成)
               2.3 ──┘

3.1 ──┬─→ 3.3
3.2 ──┤   3.4
      └─→ 5.1

6.1 → 6.2 → 6.3 → 6.4  (前端独立于后端 API 完成后开始)
```

### 可并行的工作

以下步骤可以同时进行（无依赖冲突）：

| 并行组 | 步骤 |
|--------|------|
| 第一批 | 1.1 + 1.2 + 1.3 + 2.3 + 3.1 + 3.2 |
| 第二批 | 1.4 + 1.5（1.1-1.3 完成后）+ 2.1（1.3 完成后）|
| 第三批 | 2.2 + 3.3 + 3.4 + 4.1 + 5.1（1.5 和 2.1 完成后）|
| 第四批 | 6.1-6.4（后端 API 完成后）|

---

## 8. 验收标准

### Phase 1 验收

- [ ] `go build ./...` 通过
- [ ] `go test ./internal/config/datamodel/...` 全部通过
- [ ] validator 测试覆盖：8 种 Type × 合法/非法值，6 种 Constraint 类型，多实例路径匹配
- [ ] path_matcher 测试覆盖：单层/嵌套/无占位符/边界

### Phase 2 验收

- [ ] 用 CPE 模拟器测试两阶段同步：
  - 无多实例模型 → 直接 GPV 静态前缀
  - 有多实例模型 → Phase1 GPN 发现实例 → Phase2 GPV 获取参数
  - 嵌套多实例 → 多轮 GPN 逐层展开
- [ ] device_parameters 中参数数量 > 模型参数数量（多实例展开后）
- [ ] 旧的 `extractPathsFromParameterTree` 不再被调用

### Phase 3 验收

- [ ] `GET /devices/:id/parameters/schema?path_prefix=Device.Services.FAPService.1.` 返回：
  - 每个参数包含 type, writable, constraints, description, default_value, notify, change_applies, current_value
  - objects 包含 can_add, can_delete, max_instances, current_instances
- [ ] `GET /devices/:id/parameters/tree` 返回的树节点包含模型元数据

### Phase 4 验收

- [ ] `PUT /devices/:id/parameters` 提交非法值（超出 min/max、不在 enum、类型错误）→ 400 + validation_errors
- [ ] 提交只读参数 → 400 + validation_errors
- [ ] 提交包含 RebootRequired 参数 → 200 + reboot_required: true

### Phase 5 验收

- [ ] AddObject 达到 MaxInstances 时 → 400 + 错误信息
- [ ] DeleteObject 降至 MinInstances 时 → 400 + 错误信息
- [ ] AddObject 对 READ_ONLY 对象 → 400 + 错误信息

### Phase 6 验收

- [ ] 参数树可展开/折叠
- [ ] 可写参数显示编辑按钮，只读参数不显示
- [ ] 枚举参数显示下拉框
- [ ] 数值参数显示 InputNumber (min/max)
- [ ] RebootRequired 参数显示警告标签
- [ ] 多实例对象显示添加/删除按钮（受 can_add/can_delete 控制）
- [ ] 搜索功能可用
- [ ] 同步按钮可触发，状态条正确显示

---

## 9. 风险缓解

| 风险 | 缓解措施 |
|------|---------|
| GPN 事件未在 ACS 端发布 | Step 2.3 先行调查，如需改动 ACS session handler 则独立提交 |
| 嵌套多实例 GPN 次数爆炸 | iterator 设置 `maxDepth=3` 上限，超过则 fallback 到 GPV 根路径 |
| CPE 不支持部分路径 GPV | 在 AppConfig 中添加 `gpv_strategy: partial_path \| individual`，默认 partial_path |
| 大量参数导致 Schema API 慢 | GetByPathPrefix 有索引；前端按需加载（path_prefix 参数限定范围）|
| MinInstances 规则表不完整 | 默认值为 1（保守策略），后续从真实设备测试中补充 |
| Redis 同步计划 TTL 过期 | 设置 10 分钟 TTL，超时任务被 TaskReaper 清理；前端显示超时状态 |

---

## 10. 文件变更清单

### 新增文件（后端 7 个）

| 文件 | 大小估算 | Phase |
|------|---------|-------|
| `internal/config/datamodel/path_matcher.go` | ~120 行 | 1 |
| `internal/config/datamodel/path_matcher_test.go` | ~150 行 | 1 |
| `internal/config/datamodel/min_instances.go` | ~60 行 | 1 |
| `internal/config/datamodel/validator.go` | ~250 行 | 1 |
| `internal/config/datamodel/validator_test.go` | ~300 行 | 1 |
| `internal/config/datamodel/iterator.go` | ~300 行 | 2 |
| `internal/config/datamodel/iterator_test.go` | ~250 行 | 2 |
| `migrations/000061_device_parameters_path_index.up.sql` | ~5 行 | 3 |
| `migrations/000061_device_parameters_path_index.down.sql` | ~2 行 | 3 |

### 修改文件（后端 8 个）

| 文件 | 改动范围 | Phase |
|------|---------|-------|
| `internal/config/datamodel/model.go` | +2 字段 | 1 |
| `internal/config/datamodel/xml_parser.go` | ~5 行 | 1 |
| `internal/config/datamodel/xml_parser_test.go` | ~10 行 | 1 |
| `internal/device/param_repository.go` | +3 方法 | 3 |
| `internal/device/pg_param_repository.go` | +~60 行 | 3 |
| `internal/device/param_handler.go` | +~300 行(schema/add/delete/enrich) | 3-5 |
| `internal/provision/sync.go` | +~100 行(两阶段同步) | 2 |
| `internal/provision/engine.go` | ~30 行改造 | 2 |
| `cmd/app/router/router.go` | ~3 行（构造函数参数） | 3 |

### 新增文件（前端 ~5 个）

| 文件 | Phase |
|------|-------|
| `src/services/api/parameterApi.ts` | 6 |
| `src/hooks/api/useDeviceParameters.ts` | 6 |
| `src/pages/device/parameters/DeviceParametersPage.tsx` | 6 |
| `src/pages/device/parameters/ParameterEditor.tsx` | 6 |
| `src/pages/device/parameters/ParameterTree.tsx` | 6 |

---

## 11. 推荐实施顺序

```
Day 1: Phase 1 全部（模型补全 + 基础工具）
       Step 1.1 → 1.2 → 1.3 → 1.4 → 1.5
       验收: go test 全部通过

Day 2: Phase 2 前半 + Phase 3 前半（并行）
       Step 2.1 (iterator) + Step 3.1 (仓储扩展) + Step 3.2 (注入 dmRegistry)
       验收: go build 通过

Day 3: Phase 2 后半 + Phase 3 后半
       Step 2.2 (改造 SyncService) + Step 2.3 (ACS GPN 事件)
       Step 3.3 (Schema API) + Step 3.4 (增强 Tree)
       验收: curl 测试 Schema API 返回正确结构

Day 4: Phase 4 + Phase 5（参数修改增强 + 对象增删）
       Step 4.1 + Step 5.1
       验收: curl 测试校验拒绝非法值，对象增删命令入队

Day 5: Phase 6（前端）
       Step 6.1 → 6.2 → 6.3 → 6.4
       验收: 前端页面可浏览参数、编辑、搜索
```

每天结束后 `go build ./...` 和 `go test ./...` 必须全部通过。
