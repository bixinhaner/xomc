# TR-069 参数模型深度分析与参数操作设计方案

> **日期**: 2026-03-24
> **状态**: 已实现 (2026-03-24)
> **影响范围**: F01(ACS) / F02(Config/DataModel) / F06(Device) / 前端

---

## 1. 参数模型的本质 — 我的理解

### 1.1 参数模型是"能力描述"，不是"实际数据"

**这是最核心的认知**：参数模型（DataModel）描述的是设备**支持什么参数**，而不是设备**实际有什么参数**。

类比：
- **参数模型** ≈ "数据库 Schema"（表结构定义）
- **设备实际参数** ≈ "数据库中的数据"（具体的行和值）

参数模型说"这个基站支持 `Device.Services.FAPService.{12}.` 下的参数"，但实际上这台基站可能只创建了 1 个或 4 个 FAPService 实例。模型只是声明了**最大能力（maxInstances）**。

### 1.2 多实例对象（Multi-Instance Object）

TR-069 中的多实例对象是参数树中的"表格"节点。它们用 `{i}` 或 `{N}` 占位符表示模板路径。

**示例**：
```
参数模型中的路径:
  Device.Services.FAPService.{12}.CellConfig.LTE.RAN.Common.CellIdentity

这里 {12} 表示:
  - 这是一个多实例节点
  - 12 可能表示最大实例数（或是占位符编号）
  - 实际设备上可能有:
    Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity
    Device.Services.FAPService.2.CellConfig.LTE.RAN.Common.CellIdentity
    Device.Services.FAPService.3.CellConfig.LTE.RAN.Common.CellIdentity
    （取决于设备实际创建了多少个 FAPService 实例）
```

**多实例对象的关键属性**：

| 属性 | 含义 |
|------|------|
| `maxInstances` | 最大实例数。0 表示无限制 |
| `minInstances`（隐含） | 最小实例数。TR-069 规范中，某些对象有不可删除的最小实例 |
| `access` | READ_WRITE 表示可以 AddObject/DeleteObject；READ_ONLY 表示实例数固定 |
| `isList` | 是否是列表/表格对象 |

**多实例对象在我们系统中的表现**：

```
参数模型（XML 上传）:
  <object name="Device.Services.FAPService.{12}." access="READ_WRITE" maxInstances="0" isList="false" />

  <param name="Device.Services.FAPService.{12}.CellConfig.LTE.RAN.Common.CellIdentity"
         access="READ_WRITE" type="U_INT" min="0" max="268435455" />

种子数据（JSON）:
  "Device.Services.FAPService.1.": {
    "type": "object",
    "multi_instance": true,
    "parameters": {
      "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity": { ... }
    }
  }
```

### 1.3 参数数量：模型 vs 实际

以一个典型的 LTE 小基站为例：

```
参数模型定义的参数路径: 1954 个（模板路径）
  其中包含多实例对象节点，每个节点下可能有几十到上百个参数

设备实际的参数数量: 远超 1954 个
  假设:
  - FAPService 有 4 个实例
  - 每个 FAPService 下的 NeighborList 有 32 个实例
  - PLMNList 有 6 个实例
  - ...

  仅 FAPService 下的参数就会展开为:
    4 × (单实例参数数 + 32 × 邻区参数数 + 6 × PLMN参数数 + ...)

  总数可能达到 5000-20000+ 个实际参数路径
```

### 1.4 获取实际参数的方式

TR-069 提供了两种机制获取设备实际参数：

**方式 A: GetParameterNames (GPN)**
```
请求: GetParameterNames("Device.Services.FAPService.", NextLevel=true)
响应:
  Device.Services.FAPService.1.    (writable=true)
  Device.Services.FAPService.2.    (writable=true)

→ 这告诉我们设备实际有 2 个 FAPService 实例
```

**方式 B: GetParameterValues (GPV) 用部分路径**
```
请求: GetParameterValues("Device.Services.FAPService.")
响应:
  Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity = 123
  Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID = 45
  Device.Services.FAPService.2.CellConfig.LTE.RAN.Common.CellIdentity = 456
  Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.PhyCellID = 67
  ... (所有该路径下的参数及其值)
```

**关键区别**：
- GPN 返回**路径列表**（不含值），可用 NextLevel 逐级展开
- GPV 用**部分路径**（以 `.` 结尾的路径前缀）返回**该前缀下所有参数的路径和值**

### 1.5 当前代码的问题

当前 `extractPathsFromParameterTree` 直接取模型中的参数路径，然后传给 GPV：

```go
// engine.go:320-335
func extractPathsFromParameterTree(tree json.RawMessage) ([]string, error) {
    var params []struct {
        Path string `json:"path"`
    }
    json.Unmarshal(tree, &params)
    for _, p := range params {
        paths = append(paths, p.Path)
    }
    return paths, nil
}
```

**问题**：
1. **模板路径无法直接用于 GPV** — `Device.Services.FAPService.{12}.CellConfig.LTE.RAN.Common.CellIdentity` 不是有效的设备参数路径，设备不认识 `{12}`
2. **即使把 `{12}` 换成 `1`，也只获取了一个实例** — 设备可能有多个实例
3. **没有利用"部分路径 GPV"能力** — TR-069 允许用 `Device.Services.FAPService.` 一次获取所有 FAPService 下的参数

---

## 2. 当前系统状态分析

### 2.1 已有的基础设施

| 组件 | 状态 | 说明 |
|------|------|------|
| 参数模型存储 | ✅ 完善 | `data_model_definitions` 表，parameter_tree + object_tree |
| 三级解析 | ✅ 完善 | product → oui → carrier_default 回退 |
| 三级缓存 | ✅ 完善 | L1 内存 + L2 Redis + L3 PostgreSQL |
| XML 解析器 | ✅ 完善 | 解析 CPE 上传的参数模型 XML |
| 命令队列 | ✅ 完善 | Redis Sorted Set，支持优先级 |
| RPC 分发器 | ✅ 完善 | 12 种 RPC 方法全部实现 |
| SOAP 编解码 | ✅ 完善 | 模板渲染 + 流式解析 |
| 设备参数存储 | ✅ 基础 | `device_parameters` 表，BatchUpsert |
| 参数同步 | ⚠️ 有问题 | 直接取模型路径，无法处理多实例 |
| 参数遍历/迭代 | ❌ 缺失 | 没有基于模型的智能参数路径展开 |
| 参数读取 API | ❌ 缺失 | 前端无法查看设备参数 |
| 参数修改 API | ❌ 缺失 | 前端无法修改设备参数 |
| 对象增删 API | ❌ 缺失 | 前端无法添加/删除多实例对象 |

### 2.2 ObjectInfo 数据分析

从 XML 上传的 `object_tree` 包含对象定义：

```json
[
  {"name": "Device.", "access": "READ_WRITE", "max_instances": 0, "is_list": false},
  {"name": "Device.Services.FAPService.{12}.", "access": "READ_WRITE", "max_instances": 0, "is_list": false},
  {"name": "Device.Services.FAPService.{12}.CellConfig.LTE.EPC.PLMNList.{6}.", "access": "READ_WRITE", "max_instances": 6, "is_list": true},
  {"name": "Device.Services.FAPService.{12}.CellConfig.LTE.RAN.NeighborList.LTECell.{256}.", "access": "READ_WRITE", "max_instances": 256, "is_list": true}
]
```

**从 object_tree 可以提取的信息**：
- 哪些路径是多实例对象
- 每个多实例对象的最大实例数
- 对象是否可写（是否允许 AddObject/DeleteObject）
- 对象的嵌套关系（多实例内嵌多实例）

---

## 3. 设计方案

### 3.1 整体思路：两阶段参数获取

```
阶段 1: 发现实际实例（GPN）
  对参数模型中的每个多实例对象节点，发送 GPN(NextLevel=true) 获取实际实例列表

  例: GPN("Device.Services.FAPService.", NextLevel=true)
  → [Device.Services.FAPService.1., Device.Services.FAPService.2.]

阶段 2: 批量获取参数值（GPV + 部分路径）
  使用实际实例路径前缀批量获取参数值

  例: GPV(["Device.Services.FAPService.1.", "Device.Services.FAPService.2."])
  → 所有参数的路径和值
```

### 3.2 参数树迭代器设计

```go
// ParameterTreeIterator 基于参数模型生成设备的实际参数获取计划。
//
// 核心思想：参数模型是"Schema"，设备是"数据"。
// 模型中的多实例占位符（{i}）需要通过 GPN 发现实际实例，
// 然后用实际实例路径通过 GPV 获取参数值。
type ParameterTreeIterator struct {
    model         *datamodel.DataModel
    objects       []datamodel.ObjectInfo    // 从 object_tree 解析
    parameters    []datamodel.Parameter     // 从 parameter_tree 解析
    multiInstObjs []MultiInstanceObject     // 分析后的多实例对象列表
}

// MultiInstanceObject 描述一个多实例对象节点。
type MultiInstanceObject struct {
    TemplatePath  string   // 模型中的模板路径，如 "Device.Services.FAPService.{12}."
    BasePath      string   // 去掉占位符的基础路径，如 "Device.Services.FAPService."
    MaxInstances  int      // 最大实例数（0=无限）
    MinInstances  int      // 最小实例数（不可删除）
    Writable      bool     // 是否允许 AddObject/DeleteObject
    ParentPath    string   // 父多实例对象路径（支持嵌套多实例）
    ChildParams   []string // 该对象下的直接子参数路径模板
}

// DiscoveryPlan 描述获取设备全部参数的执行计划。
type DiscoveryPlan struct {
    // Phase1: GPN 请求列表（发现多实例的实际实例）
    GPNRequests []GPNRequest
    // Phase2: GPV 请求列表（获取参数值，在 GPN 完成后动态生成）
    // Phase2 的路径来源：
    //   - 非多实例的顶层路径（直接用模型路径或部分路径前缀）
    //   - 多实例的实际实例路径（从 GPN 结果展开）
}

type GPNRequest struct {
    Path      string // 对象路径前缀，如 "Device.Services.FAPService."
    NextLevel bool   // true: 只获取直接子节点
}
```

### 3.3 参数获取流程

```
                    ┌────────────────────────────────┐
                    │       分析参数模型              │
                    │  提取多实例对象列表              │
                    └───────────┬────────────────────┘
                                │
                    ┌───────────▼────────────────────┐
                    │  Phase 1: 发送 GPN 请求         │
                    │  对每个多实例对象发 GPN          │
                    │  发现实际实例编号               │
                    └───────────┬────────────────────┘
                                │
                    ┌───────────▼────────────────────┐
                    │  构建实际参数路径               │
                    │  模板路径 × 实际实例 → 展开     │
                    └───────────┬────────────────────┘
                                │
                    ┌───────────▼────────────────────┐
                    │  Phase 2: 批量 GPV              │
                    │  使用部分路径前缀获取参数值      │
                    │  分批入队（batchSize=50）        │
                    └───────────┬────────────────────┘
                                │
                    ┌───────────▼────────────────────┐
                    │  存储参数值到 device_parameters  │
                    └────────────────────────────────┘
```

**优化：使用部分路径减少 RPC 次数**

不需要发送 1954+ 个单独的 GPV 请求。利用 TR-069 的"部分路径"特性：

```
# 方案 A：逐参数请求（差，1954+ 次 RPC）
GPV(["Device.DeviceInfo.Manufacturer"])
GPV(["Device.DeviceInfo.SerialNumber"])
...

# 方案 B：部分路径请求（好，~10-20 次 RPC）
GPV(["Device.DeviceInfo."])           → 返回该节点下所有参数
GPV(["Device.ManagementServer."])     → 返回该节点下所有参数
GPV(["Device.Services.FAPService."])  → 返回所有实例的所有参数
```

方案 B 利用参数模型的 object_tree，识别出顶层对象节点，用这些节点作为 GPV 的部分路径前缀。每个前缀一次 GPV 就能获取下面所有参数。

**更优化的方案 C：直接用根路径**

```
GPV(["Device."])  → 返回设备所有参数
```

但这可能导致 SOAP 响应过大（几 MB），某些 CPE 可能超时。建议用"二级对象路径"（方案 B）作为默认策略，配置允许调整层级深度。

### 3.4 参数读取 API 设计

为前端提供按路径查询设备参数的接口。

**数据来源选择**：
- **缓存读取**：从 `device_parameters` 表读（已同步的参数值）— 快，但可能不是最新
- **实时读取**：通过 ACS 发送 GPV 到设备 — 准确，但需要设备在线且耗时

两种方式都需要支持。

#### API 端点设计

```
# 1. 从缓存读取设备参数（快速，离线可用）
GET /api/v1/devices/:id/parameters
  Query:
    path_prefix: string  — 路径前缀过滤，如 "Device.Services.FAPService.1."
    category: string     — 按分类过滤（radio/management/alarm...）
    writable: bool       — 只显示可写参数
    page/page_size       — 分页
  Response:
    {
      "items": [
        {
          "path": "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity",
          "value": "123",
          "type": "unsignedInt",
          "writable": true,
          "last_updated_at": "2026-03-24T10:00:00Z"
        }
      ],
      "total": 5000,
      "tree": { ... }    // 可选：树形结构视图
    }

# 2. 实时从设备读取参数（需要设备在线）
POST /api/v1/devices/:id/parameters/read
  Body:
    {
      "paths": ["Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity"],
      // 或者用部分路径
      "path_prefix": "Device.Services.FAPService.1.CellConfig.LTE.RAN."
    }
  Response:
    {
      "task_id": "uuid",        // 异步任务 ID
      "status": "pending"       // pending → completed/failed
    }

# 3. 查询实时读取结果
GET /api/v1/devices/:id/parameters/read/:task_id
  Response:
    {
      "status": "completed",
      "parameters": [ ... ]
    }

# 4. 获取设备参数树结构（结合模型 + 实际参数）
GET /api/v1/devices/:id/parameter-tree
  Query:
    path_prefix: string — 路径前缀
    depth: int          — 展开深度（默认 2）
  Response:
    {
      "root": "Device.",
      "children": [
        {
          "path": "Device.DeviceInfo.",
          "type": "object",
          "children_count": 8,
          "expanded": false
        },
        {
          "path": "Device.Services.FAPService.",
          "type": "object",
          "multi_instance": true,
          "instances": [1, 2],      // 实际存在的实例编号
          "max_instances": 12,
          "min_instances": 1,
          "children_count": 45,
          "expanded": false
        }
      ]
    }
```

### 3.5 参数修改 API 设计

```
# 修改参数值（通过 ACS 下发 SetParameterValues）
POST /api/v1/devices/:id/parameters/set
  Body:
    {
      "parameters": [
        {
          "path": "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity",
          "value": "456",
          "type": "unsignedInt"       // 可选，系统可从模型推断
        }
      ],
      "commit": true                 // true: 立即生效；false: 仅暂存（SetParameterValues 的 commit 语义）
    }
  Response:
    {
      "task_id": "uuid",
      "status": "pending"
    }

# 查询修改结果
GET /api/v1/devices/:id/parameters/set/:task_id
  Response:
    {
      "status": "completed",         // completed / failed
      "results": [
        {
          "path": "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity",
          "success": true,
          "previous_value": "123",
          "new_value": "456"
        }
      ]
    }
```

**参数校验流程**：
```
前端提交修改请求
    │
    ▼
后端校验:
  1. 参数路径是否存在于模型中?
  2. 参数是否 writable?
  3. 值是否满足 Constraints (min/max/enum/pattern/maxLength)?
  4. 类型是否匹配?
    │
    ▼ 校验通过
构建 SetParameterValues SOAP 请求
    │
    ▼
入队到设备命令队列 (priority=5, 高于同步任务)
    │
    ▼
等待 CPE 响应 → 更新 device_parameters 表
```

### 3.6 对象增删 API 设计

```
# 添加对象实例
POST /api/v1/devices/:id/objects/add
  Body:
    {
      "object_path": "Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList."
    }
  Response:
    {
      "task_id": "uuid",
      "status": "pending"
    }

  # 成功后结果
  {
    "status": "completed",
    "instance_number": 3,                              // CPE 分配的实例编号
    "instance_path": "Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.3."
  }

# 删除对象实例
DELETE /api/v1/devices/:id/objects
  Body:
    {
      "object_path": "Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.3."
    }
  Response:
    {
      "task_id": "uuid",
      "status": "pending"
    }
```

**最小实例数保护**：

```go
// 删除前校验逻辑
func (s *ParameterService) ValidateDeleteObject(ctx context.Context,
    deviceID uuid.UUID, objectPath string) error {

    // 1. 从参数模型找到该对象的定义
    obj := s.findObjectDefinition(objectPath)
    if obj == nil {
        return errors.New("object path not found in model")
    }

    // 2. 检查 access 权限
    if obj.Access == "READ_ONLY" {
        return errors.New("object is read-only, cannot delete instances")
    }

    // 3. 查询当前实际实例数
    currentInstances := s.countCurrentInstances(ctx, deviceID, objectPath)

    // 4. 检查最小实例数约束
    minInstances := s.getMinInstances(obj)
    if currentInstances <= minInstances {
        return fmt.Errorf("cannot delete: current instances (%d) <= minimum required (%d)",
            currentInstances, minInstances)
    }

    return nil
}

// getMinInstances 确定对象的最小实例数
// TR-069 规范中，某些对象的 minEntries > 0，这些实例不可删除。
// 对于我们的 XML 格式，minEntries 信息可能不直接提供，
// 需要从以下来源推断：
//   1. 标准 TR-181/TR-196 规范定义（内置规则表）
//   2. 对象 access=READ_ONLY 时，所有实例不可删除
//   3. 配置文件中的自定义规则
func (s *ParameterService) getMinInstances(obj *datamodel.ObjectInfo) int {
    // 常见的不可删除对象（minEntries >= 1）
    // 根据 TR-196 Small Cell 数据模型规范：
    minEntriesRules := map[string]int{
        "Device.Services.FAPService.":                        1, // 至少 1 个 FAPService
        "*.CellConfig.LTE.EPC.PLMNList.":                     1, // 至少 1 个 PLMN
        "*.CellConfig.LTE.RAN.NeighborList.LTECell.":         0, // 邻区可全部删除
    }

    for pattern, min := range minEntriesRules {
        if matchPattern(obj.Name, pattern) {
            return min
        }
    }

    // 默认：至少保留 1 个实例
    // 保守策略，防止删除关键实例导致设备异常
    return 1
}
```

### 3.7 全量参数同步流程（改进版）

替换当前有问题的 `extractPathsFromParameterTree` + 直接 GPV 方式。

```
新流程:

1. 分析参数模型
   ├─ 解析 object_tree → 识别所有多实例对象
   ├─ 解析 parameter_tree → 识别所有参数路径模板
   └─ 构建对象层级关系（处理嵌套多实例）

2. Phase 1: 发现实际实例 (GPN)
   ├─ 提取所有多实例对象的基础路径
   │   例: ["Device.Services.FAPService.",
   │        "Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.",
   │        "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell."]
   │
   ├─ 先发顶层多实例 GPN
   │   GPN("Device.Services.FAPService.", NextLevel=true) → [1, 2]
   │
   ├─ 展开嵌套多实例（用实际实例编号替换 {i}）
   │   GPN("Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.", NextLevel=true) → [1, 2, 3]
   │   GPN("Device.Services.FAPService.2.CellConfig.LTE.EPC.PLMNList.", NextLevel=true) → [1]
   │   ... (对每个上层实例 × 每个嵌套多实例对象)
   │
   └─ 结果: 完整的实例映射表
       {
         "Device.Services.FAPService.": [1, 2],
         "Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.": [1, 2, 3],
         "Device.Services.FAPService.2.CellConfig.LTE.EPC.PLMNList.": [1],
         ...
       }

3. Phase 2: 批量获取参数值 (GPV + 部分路径前缀)
   ├─ 使用二级对象路径作为 GPV 前缀（而非逐个参数路径）
   │   GPV(["Device.DeviceInfo."])
   │   GPV(["Device.ManagementServer."])
   │   GPV(["Device.Services.FAPService.1."])
   │   GPV(["Device.Services.FAPService.2."])
   │   GPV(["Device.X_CMCC."])
   │
   └─ 按 batchSize 分批入队

4. 存储结果
   └─ BatchUpsert 到 device_parameters 表
```

### 3.8 模块结构

```
omcgo/internal/config/datamodel/
├── model.go              # 现有：增加 MinInstances 等字段到 ObjectInfo
├── iterator.go           # 新增：参数树迭代器，多实例展开逻辑
├── iterator_test.go      # 新增：迭代器测试

omcgo/internal/device/
├── parameter_handler.go  # 新增：设备参数 REST API handler
├── parameter_service.go  # 新增：参数读取/修改/对象增删业务逻辑
├── parameter_model.go    # 现有/扩展：DeviceParameter 模型
├── parameter_pg_repo.go  # 现有/扩展：参数值持久层

omcgo/internal/provision/
├── sync.go               # 修改：使用新的迭代器替换 extractPathsFromParameterTree
```

### 3.9 前端交互设计要点

```
参数浏览页面:
┌──────────────────────────────────────────────────────────┐
│ 设备参数                                    [同步] [刷新] │
├──────────────────────────────────────────────────────────┤
│ ▼ Device.                                               │
│   ▼ DeviceInfo.                                         │
│     Manufacturer      = "HuaWei"        (只读)          │
│     SerialNumber      = "1202000588233"  (只读)          │
│     SoftwareVersion   = "V100R001C00"    (只读)          │
│   ▼ ManagementServer.                                   │
│     URL               = "http://..."     [编辑]          │
│     PeriodicInformInterval = 300         [编辑]          │
│   ▼ Services.FAPService.  [多实例: 2个]  [+添加]         │
│     ▼ 1.                                 [不可删除]      │
│       ▼ CellConfig.LTE.RAN.Common.                      │
│         CellIdentity  = 123              [编辑]          │
│         EARFCNDL      = 38400            [编辑]          │
│       ▼ CellConfig.LTE.EPC.PLMNList.  [3个] [+添加]     │
│         ▼ 1.                             [不可删除]      │
│           PLMNID      = "46000"          [编辑]          │
│         ▼ 2.                             [删除]          │
│           PLMNID      = "46002"          [编辑]          │
│         ▼ 3.                             [删除]          │
│           PLMNID      = "46007"          [编辑]          │
│     ▼ 2.                                 [删除]          │
│       ...                                                │
└──────────────────────────────────────────────────────────┘
```

---

## 4. 实施阶段

### Phase 1: 参数树迭代器 + 改进同步

**目标**: 正确获取设备的全部参数

1. 实现 `ParameterTreeIterator` — 分析模型，提取多实例对象
2. 实现两阶段同步（GPN 发现 + GPV 部分路径获取）
3. 修改 `SyncService` 使用新迭代器
4. 单元测试 + 集成测试

### Phase 2: 参数缓存读取 API

**目标**: 前端可以浏览设备参数

1. 实现 `GET /api/v1/devices/:id/parameters` — 缓存读取
2. 实现 `GET /api/v1/devices/:id/parameter-tree` — 树形结构
3. 扩展 `device_parameters` 表（如需要加索引）
4. 前端页面基础框架

### Phase 3: 参数实时读取 + 修改

**目标**: 前端可以读取最新值并修改参数

1. 实现 `POST /api/v1/devices/:id/parameters/read` — 实时读取
2. 实现 `POST /api/v1/devices/:id/parameters/set` — 修改参数
3. 参数校验（类型、约束、可写性）
4. 异步任务跟踪

### Phase 4: 对象增删

**目标**: 前端可以添加/删除多实例对象

1. 实现 `POST /api/v1/devices/:id/objects/add` — AddObject
2. 实现 `DELETE /api/v1/devices/:id/objects` — DeleteObject
3. 最小实例数保护
4. 实例数变更后自动刷新参数缓存

---

## 5. 关键设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 多实例发现 | GPN(NextLevel=true) | 精确发现实际实例，不依赖猜测 |
| 参数批量获取 | GPV(部分路径前缀) | 减少 RPC 次数，一次获取子树所有参数 |
| 最小实例数来源 | 内置规则表 + 配置 | XML 不提供 minEntries，从 TR-196 规范提取 |
| 参数修改交互 | 异步任务模式 | CPE 响应需要时间，不能同步等待 |
| 参数缓存 | device_parameters 表 | 设备离线时仍可查看上次同步的参数 |
| 前端展示 | 懒加载树形结构 | 参数数量大（5000+），不能一次加载 |

---

## 6. 风险与注意事项

1. **嵌套多实例的 GPN 次数** — 如果多实例嵌套 3 层（FAPService → PLMNList → SubList），GPN 次数会呈指数增长。需要限制最大深度。
2. **GPV 响应大小** — 某些 CPE 对单次 GPV 的响应大小有限制。部分路径前缀的层级需要可配置。
3. **并发安全** — 多个用户同时修改同一设备参数时，需要用 Redis 分布式锁保护。
4. **CPE 兼容性** — 不是所有 CPE 都支持部分路径 GPV。需要有降级策略（逐参数请求）。
5. **会话限制** — 每个 ACS 会话最大 RPC 数（maxRPCPerSession=15）。大量 GPN + GPV 可能需要多个会话。

---

## 7. 参数模型字段完整清单与使用场景

### 7.1 Parameter 结构体 — 每个字段的含义与使用

参数模型中的 `Parameter` 包含以下字段，**每个字段都有实际用途，不能遗漏**：

| 字段 | 类型 | 来源 | 当前状态 | 用途 |
|------|------|------|---------|------|
| `Path` | string | XML `name` 属性 | ✅ 已使用 | 参数唯一标识路径 |
| `UnifiedName` | string | 手动映射 | ⚠️ 未使用 | 跨厂商统一命名（如不同厂商的 CellIdentity 路径不同但含义相同） |
| `Type` | string | XML `type` 属性 | ✅ 已存储 | **校验**: 类型检查；**前端**: 决定输入控件类型 |
| `Writable` | bool | XML `access` 属性 | ✅ 已存储 | **校验**: 禁止修改只读参数；**前端**: 只读参数隐藏编辑按钮 |
| `Description` | string | 手动/规范 | ⚠️ 部分有 | **前端**: 参数说明文字 / tooltip |
| `Constraints` | *Constraints | XML `min`/`max` | ✅ 已解析 | **校验**: 值域/长度/枚举/正则校验；**前端**: 表单校验规则 |
| `Category` | string | 路径自动推断 | ✅ 已生成 | **前端**: 按分类分组过滤 |
| `Notify` | string | XML `notify` 属性 | ✅ 已解析 | **显示**: 告知用户该参数的通知策略 |
| `ForcedInform` | bool | XML `forcedInform` | ✅ 已解析 | **显示**: 标记 CPE 必须在 Inform 中上报的参数 |
| `DefaultValue` | string | XML `defaultValue` | ✅ 已解析 | **校验/前端**: 显示默认值，重置时使用 |
| `ChangeApplies` | string | XML `changeApplies` | ✅ 已解析 | **前端**: 告知用户修改后何时生效 |
| `IsList` | bool | XML `isList` 属性 | ✅ 已解析 | **显示**: 标识列表类型参数 |

### 7.2 Constraints 结构体 — 参数校验的核心

```go
type Constraints struct {
    MinValue   *int64   `json:"min_value,omitempty"`    // 数值类型最小值
    MaxValue   *int64   `json:"max_value,omitempty"`    // 数值类型最大值
    EnumValues []string `json:"enum_values,omitempty"`  // 枚举可选值列表
    Pattern    string   `json:"pattern,omitempty"`       // 正则表达式
    MaxLength  int      `json:"max_length,omitempty"`   // 字符串最大长度
}
```

**XML min/max 的双重语义**（已在 `buildConstraints` 中正确处理）：
- **STRING 类型**: `min` = 最小长度（当前被忽略），`max` = 最大长度 → `MaxLength`
- **数值类型** (INT/U_INT/LONG等): `min`/`max` = 值域范围 → `MinValue`/`MaxValue`

**⚠️ 当前缺失**: STRING 的 `min` 属性（最小长度）被 `buildConstraints` 忽略了。需要补充 `MinLength` 字段。

**每种约束的校验规则和前端表现**：

| 约束字段 | 校验规则 | 前端交互 | 示例 |
|---------|---------|---------|------|
| `MinValue` | `value >= MinValue` | 数字输入框 min 属性 + 错误提示 | CellIdentity: min=0 |
| `MaxValue` | `value <= MaxValue` | 数字输入框 max 属性 + 错误提示 | CellIdentity: max=268435455 |
| `EnumValues` | `value ∈ EnumValues` | **下拉选择框**(Select) 替代文本输入 | DLBandwidth: ["n6","n15","n25","n50","n75","n100"] |
| `Pattern` | `regexp.Match(Pattern, value)` | 输入时实时正则校验 | PLMNID: `^[0-9]{5,6}$` |
| `MaxLength` | `len(value) <= MaxLength` | 输入框 maxLength 属性 | URL: max=256 |
| `MinLength` | `len(value) >= MinLength` | 输入框 minLength + 校验 | （当前缺失，需补充） |

### 7.3 ObjectInfo 结构体 — 对象节点的元数据

```go
type ObjectInfo struct {
    Name         string `json:"name"`           // 对象路径（含占位符），如 "Device.Services.FAPService.{12}."
    Access       string `json:"access"`         // READ_WRITE / READ_ONLY
    MaxInstances int    `json:"max_instances"`  // 最大实例数（0=无限制）
    IsList       bool   `json:"is_list"`        // 是否为列表对象
}
```

**每个字段的使用场景**：

| 字段 | 校验使用 | 前端使用 |
|------|---------|---------|
| `Name` | 定位多实例对象节点 | 构建树形结构 |
| `Access` | **AddObject/DeleteObject 前必检**: READ_ONLY 对象禁止增删实例 | 隐藏/禁用 [+添加] [删除] 按钮 |
| `MaxInstances` | **AddObject 前必检**: 当前实例数 < MaxInstances 才允许添加 | 显示 "已满 N/N" 或允许添加 |
| `IsList` | 区分列表对象（可排序）和普通对象 | 列表对象显示序号 |

### 7.4 ModelMetadata — 模型级元数据

存储在 `model_metadata` JSONB 列中：

```json
{
  "generate_time": "2026-03-24T00:55:06+02:00",  // 模型生成时间
  "model_version": "1.0",                         // 模型版本号
  "total_entries": 2096,                           // 声明的参数+对象总数
  "serial_number": "1202000588233HB0039-LTE"       // 来源设备序列号
}
```

**使用场景**：
- `generate_time` — 前端显示模型新鲜度，判断是否需要重新上传
- `model_version` — 固件升级后对比模型版本，检测参数变化
- `total_entries` — 导入校验：声明数 vs 实际解析数是否一致
- `serial_number` — 追溯模型来源设备

### 7.5 Type 字段 — 决定前端输入控件和校验逻辑

| Type 值 | 含义 | 前端控件 | 校验规则 |
|---------|------|---------|---------|
| `string` | 字符串 | `<Input>` 文本框 | MaxLength, MinLength, Pattern |
| `unsignedInt` | 无符号整数 | `<InputNumber min={0}>` | MinValue, MaxValue, 不能为负 |
| `int` | 有符号整数 | `<InputNumber>` | MinValue, MaxValue |
| `boolean` | 布尔值 | `<Switch>` 开关 | 只接受 "true"/"false"/"0"/"1" |
| `dateTime` | 日期时间 | `<DatePicker showTime>` | ISO 8601 格式 |
| `base64` | Base64 编码 | `<TextArea>` + 文件上传 | 合法 Base64 |
| `long` | 64位有符号整数 | `<InputNumber>` | MinValue, MaxValue |
| `unsignedLong` | 64位无符号整数 | `<InputNumber min={0}>` | MinValue, MaxValue, 不能为负 |

**特殊处理**：当有 `EnumValues` 约束时，无论什么 Type，都应使用 `<Select>` 下拉框。

### 7.6 Notify 字段 — 参数通知策略

| Notify 值 | 含义 | 前端显示 |
|-----------|------|---------|
| `ACTIVE_NOTIFICATION` | 值变化时 CPE 主动上报 | 标签: "主动通知" 🟢 |
| `PASSIVE_NOTIFICATION` | 值变化时 CPE 标记，下次 Inform 上报 | 标签: "被动通知" 🟡 |
| `NO_NOTIFICATION` | 不通知 | 不显示标签或显示 "无通知" |

**运维价值**：主动通知的参数修改后，ACS 会立即收到 VALUE CHANGE 事件。被动通知的参数需要等下次 Inform 周期。无通知的参数修改后需要主动 GPV 查询。

### 7.7 ChangeApplies 字段 — 修改生效时机

| ChangeApplies 值 | 含义 | 前端提示 |
|------------------|------|---------|
| `Immediate` | 立即生效 | "修改后立即生效" |
| `NotifyRequired` | 需要重启基站后生效 | ⚠️ "**需要重启设备后生效**" |
| `RebootRequired` | 需要硬件重启 | ⚠️ "**需要重启设备后生效**" |
| 空值 | 未指定 | 不显示提示 |

**前端必须对 `NotifyRequired` / `RebootRequired` 做突出提示**，避免运维人员修改后误以为已生效。批量修改时，如果包含需要重启的参数，应在确认对话框中特别提醒。

### 7.8 DefaultValue 字段 — 默认值

**使用场景**：

1. **前端显示** — 在编辑输入框中以 placeholder 或灰色文字显示默认值
2. **重置功能** — "恢复默认值" 按钮将参数重置为 DefaultValue
3. **新增实例** — AddObject 后，新实例的参数初始值显示为 DefaultValue
4. **异常检测** — 对比当前值与默认值，高亮偏离默认的参数

### 7.9 ForcedInform 字段

**含义**：标记为 `forcedInform=true` 的参数，CPE **必须**在每次 Inform 消息中携带其当前值。

**使用场景**：
- **前端标识** — 在参数列表中标注 "Inform 必报" 标签
- **监控** — 这些参数的值可以从 Inform 消息中直接获取，无需额外 GPV
- **诊断** — 如果 Inform 中缺少 forcedInform 参数，可能是 CPE 实现有 bug

### 7.10 Category 字段 — 参数分类

从参数路径自动推断，用于前端分类导航：

| Category | 中文名 | 参数路径特征 | 典型参数 |
|----------|--------|------------|---------|
| `management` | 管理配置 | ManagementServer | ACS URL, Inform 周期 |
| `device_info` | 设备信息 | DeviceInfo | 厂商, 序列号, 版本 |
| `radio` | 射频配置 | FAPService, CellConfig | PCI, 频点, 功率, 带宽 |
| `time` | 时间配置 | Time, NTP | NTP 服务器, 时区 |
| `alarm` | 告警管理 | FaultMgmt, Alarm | 告警阈值, 告警使能 |
| `pm` | 性能管理 | PerfMgmt | PM 周期, PM 使能 |
| `transport` | 传输配置 | Tunnel, IPSec | 安全网关, SCTP |
| `neighbor` | 邻区配置 | Neighbor | 邻区列表 |
| *(空)* | 其他/扩展 | X_CMCC 等 | 运营商私有扩展参数 |

**前端使用**：参数浏览页面左侧提供分类过滤面板，点击分类快速定位。

---

## 8. 参数校验引擎详细设计

### 8.1 校验流程（后端 + 前端双重校验）

```
┌─────────────────────────────────────────────────────────────┐
│                     前端校验（即时反馈）                      │
│                                                              │
│  1. Type 检查                                                │
│     string  → 文本框                                         │
│     int/unsignedInt → 纯数字，unsignedInt 不能为负            │
│     boolean → 开关组件，无需校验                              │
│                                                              │
│  2. Constraints 检查                                         │
│     EnumValues? → Select 下拉框，天然合规                     │
│     MinValue/MaxValue? → InputNumber min/max 属性             │
│     MaxLength? → Input maxLength 属性                         │
│     Pattern? → onBlur 时正则校验                              │
│                                                              │
│  3. Writable 检查                                            │
│     writable=false → 不渲染编辑控件                           │
│                                                              │
│  4. ChangeApplies 提示                                       │
│     RebootRequired → 红色警告 "修改后需重启设备"              │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼ 提交
┌─────────────────────────────────────────────────────────────┐
│                     后端校验（权威校验）                      │
│                                                              │
│  func ValidateParameterValue(param Parameter, value string)  │
│                                                              │
│  1. 路径存在性 — 参数路径是否在模型的 parameter_tree 中       │
│     * 注意：实际路径需要将 {N} 替换为具体实例号再匹配         │
│     * 如 FAPService.1. 匹配模型中的 FAPService.{12}.          │
│                                                              │
│  2. 可写性 — param.Writable == true                          │
│                                                              │
│  3. 类型校验 — 根据 param.Type:                              │
│     string       → 直接通过                                  │
│     int          → strconv.ParseInt(value, 10, 64)           │
│     unsignedInt  → strconv.ParseUint(value, 10, 32)          │
│     boolean      → value ∈ {"true","false","0","1"}          │
│     dateTime     → time.Parse(RFC3339, value)                │
│     base64       → base64.StdEncoding.DecodeString(value)    │
│     long         → strconv.ParseInt(value, 10, 64)           │
│     unsignedLong → strconv.ParseUint(value, 10, 64)          │
│                                                              │
│  4. 约束校验 — 如果 param.Constraints != nil:                │
│     MinValue     → parsedValue >= *MinValue                  │
│     MaxValue     → parsedValue <= *MaxValue                  │
│     EnumValues   → value ∈ EnumValues                        │
│     Pattern      → regexp.MatchString(Pattern, value)        │
│     MaxLength    → len(value) <= MaxLength                   │
│                                                              │
│  返回: nil 或 ValidationError{Field, Rule, Message}          │
└─────────────────────────────────────────────────────────────┘
```

### 8.2 校验器 Go 实现

```go
// ParameterValidator 基于参数模型对参数值进行校验。
type ParameterValidator struct {
    paramMap  map[string]Parameter   // path → Parameter（模板路径）
    objectMap map[string]ObjectInfo  // path → ObjectInfo
}

// NewParameterValidator 从 DataModel 构建校验器。
func NewParameterValidator(dm *DataModel) (*ParameterValidator, error) {
    // 解析 parameter_tree 和 object_tree
    // 建立路径索引（将 {N} 占位符替换为正则表达式用于匹配）
}

// ValidateValue 校验单个参数值。
func (v *ParameterValidator) ValidateValue(path, value string) *ValidationError {
    param := v.matchParam(path) // 用正则匹配 {N} 占位符
    if param == nil {
        return &ValidationError{Path: path, Rule: "exists", Message: "参数路径不存在于数据模型中"}
    }
    if !param.Writable {
        return &ValidationError{Path: path, Rule: "writable", Message: "参数为只读，不可修改"}
    }
    if err := validateType(param.Type, value); err != nil {
        return &ValidationError{Path: path, Rule: "type", Message: err.Error()}
    }
    if param.Constraints != nil {
        if err := validateConstraints(param.Type, param.Constraints, value); err != nil {
            return &ValidationError{Path: path, Rule: "constraint", Message: err.Error()}
        }
    }
    return nil
}

// ValidateAddObject 校验对象添加是否允许。
func (v *ParameterValidator) ValidateAddObject(objectPath string, currentCount int) *ValidationError {
    obj := v.matchObject(objectPath)
    if obj == nil {
        return &ValidationError{Path: objectPath, Rule: "exists", Message: "对象路径不存在于数据模型中"}
    }
    if obj.Access != "READ_WRITE" {
        return &ValidationError{Path: objectPath, Rule: "access", Message: "对象为只读，不可添加实例"}
    }
    if obj.MaxInstances > 0 && currentCount >= obj.MaxInstances {
        return &ValidationError{
            Path:    objectPath,
            Rule:    "max_instances",
            Message: fmt.Sprintf("已达最大实例数 %d，不可继续添加", obj.MaxInstances),
        }
    }
    return nil
}

// ValidateDeleteObject 校验对象删除是否允许。
func (v *ParameterValidator) ValidateDeleteObject(objectPath string, currentCount, minInstances int) *ValidationError {
    obj := v.matchObject(objectPath)
    if obj == nil {
        return &ValidationError{Path: objectPath, Rule: "exists", Message: "对象路径不存在于数据模型中"}
    }
    if obj.Access != "READ_WRITE" {
        return &ValidationError{Path: objectPath, Rule: "access", Message: "对象为只读，不可删除实例"}
    }
    if currentCount <= minInstances {
        return &ValidationError{
            Path:    objectPath,
            Rule:    "min_instances",
            Message: fmt.Sprintf("当前实例数 %d 已达最小要求 %d，不可删除", currentCount, minInstances),
        }
    }
    return nil
}
```

### 8.3 前端接收模型元数据的 API

前端需要获取参数的完整模型信息（而不仅仅是值），才能构建正确的编辑表单。

```
# 获取参数的模型定义（含校验规则）
GET /api/v1/devices/:id/parameter-schema
  Query:
    path_prefix: string — 路径前缀
  Response:
    {
      "parameters": [
        {
          "path": "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity",
          "type": "unsignedInt",
          "writable": true,
          "description": "小区标识 (Cell ID)",
          "default_value": "",
          "notify": "ACTIVE_NOTIFICATION",
          "forced_inform": false,
          "change_applies": "Immediate",
          "category": "radio",
          "constraints": {
            "min_value": 0,
            "max_value": 268435455
          },
          // 实际当前值（来自 device_parameters 缓存，可能为 null 表示未同步）
          "current_value": "123",
          "last_synced_at": "2026-03-24T10:00:00Z"
        },
        {
          "path": "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.DLBandwidth",
          "type": "string",
          "writable": true,
          "description": "下行带宽",
          "change_applies": "RebootRequired",          // ⚠️ 前端需特殊提示
          "constraints": {
            "enum_values": ["n6", "n15", "n25", "n50", "n75", "n100"]
          },
          "current_value": "n50",
          "last_synced_at": "2026-03-24T10:00:00Z"
        }
      ],
      "objects": [
        {
          "path": "Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.",
          "access": "READ_WRITE",
          "max_instances": 6,
          "current_instances": [1, 2, 3],
          "min_instances": 1,                           // 从规则表获取
          "can_add": true,                              // 3 < 6
          "can_delete_any": true                        // 3 > 1
        }
      ]
    }
```

### 8.4 前端表单生成规则

前端根据 parameter-schema 的返回，动态生成参数编辑表单：

```typescript
function renderParameterEditor(param: ParameterSchema) {
  // 1. 只读参数：纯文本展示
  if (!param.writable) {
    return <Text type="secondary">{param.current_value}</Text>
  }

  // 2. 有枚举值：下拉框（优先级最高，覆盖其他类型）
  if (param.constraints?.enum_values?.length) {
    return <Select options={param.constraints.enum_values.map(v => ({label: v, value: v}))} />
  }

  // 3. 按类型选择控件
  switch (param.type) {
    case 'boolean':
      return <Switch />

    case 'int':
    case 'unsignedInt':
    case 'long':
    case 'unsignedLong':
      return <InputNumber
        min={param.constraints?.min_value}
        max={param.constraints?.max_value}
        // unsignedInt/unsignedLong 额外约束 min >= 0
      />

    case 'dateTime':
      return <DatePicker showTime />

    case 'string':
    default:
      return <Input
        maxLength={param.constraints?.max_length}
        placeholder={param.default_value || undefined}
      />
  }

  // 4. ChangeApplies 警告
  if (param.change_applies === 'RebootRequired' || param.change_applies === 'NotifyRequired') {
    // 显示橙色警告标签
  }
}
```

---

## 9. 需要补充的代码改动

### 9.1 Constraints 结构体补充 MinLength

当前 `buildConstraints` 对 STRING 类型忽略了 `min` 值。需要补充：

```go
// model.go — Constraints 增加 MinLength
type Constraints struct {
    MinValue   *int64   `json:"min_value,omitempty"`
    MaxValue   *int64   `json:"max_value,omitempty"`
    EnumValues []string `json:"enum_values,omitempty"`
    Pattern    string   `json:"pattern,omitempty"`
    MaxLength  int      `json:"max_length,omitempty"`
    MinLength  int      `json:"min_length,omitempty"`   // 新增：字符串最小长度
}

// xml_parser.go — buildConstraints 补充 MinLength
func buildConstraints(xmlType, minStr, maxStr string) *Constraints {
    if minStr == "" && maxStr == "" {
        return nil
    }
    c := &Constraints{}
    isString := strings.ToUpper(xmlType) == "STRING"
    if isString {
        if minStr != "" {
            if minLen, err := strconv.Atoi(minStr); err == nil && minLen > 0 {
                c.MinLength = minLen  // 新增
            }
        }
        if maxStr != "" {
            if maxLen, err := strconv.Atoi(maxStr); err == nil && maxLen > 0 {
                c.MaxLength = maxLen
            }
        }
    } else {
        // 数值类型: min/max 是值域范围（不变）
    }
    // ...
}
```

### 9.2 ObjectInfo 结构体补充 MinInstances

```go
// model.go — ObjectInfo 增加 MinInstances
type ObjectInfo struct {
    Name         string `json:"name"`
    Access       string `json:"access"`
    MaxInstances int    `json:"max_instances"`
    MinInstances int    `json:"min_instances"`  // 新增：最小实例数（从规则表或配置加载）
    IsList       bool   `json:"is_list"`
}
```

### 9.3 实际路径 ↔ 模板路径的匹配

参数模型中的路径包含 `{N}` 占位符，但设备实际参数路径使用数字编号。校验时需要做匹配：

```go
// 将模板路径转为正则表达式
// "Device.Services.FAPService.{12}.CellConfig.LTE.EPC.PLMNList.{6}.PLMNID"
// → "^Device\\.Services\\.FAPService\\.\\d+\\.CellConfig\\.LTE\\.EPC\\.PLMNList\\.\\d+\\.PLMNID$"
func templatePathToRegex(templatePath string) *regexp.Regexp {
    // 转义 . 为 \.
    escaped := regexp.QuoteMeta(templatePath)
    // 将 \{N\} 替换为 \d+
    pattern := regexp.MustCompile(`\\\{\\d+\\\}`).ReplaceAllString(escaped, `\d+`)
    return regexp.MustCompile("^" + pattern + "$")
}

// 匹配实际路径到模板参数定义
func (v *ParameterValidator) matchParam(actualPath string) *Parameter {
    // 优先精确匹配（适用于非多实例路径）
    if p, ok := v.paramMap[actualPath]; ok {
        return &p
    }
    // 正则匹配（适用于多实例路径）
    for templatePath, p := range v.paramMap {
        if containsPlaceholder(templatePath) {
            regex := templatePathToRegex(templatePath)
            if regex.MatchString(actualPath) {
                return &p
            }
        }
    }
    return nil
}
```

---

## 10. 字段遗漏检查清单

逐一确认所有 XML 属性和 Go 字段是否被正确使用：

### XML `<param>` 属性检查

| XML 属性 | Go 字段 | 解析 | 存储 | 校验 | 前端显示 | 状态 |
|----------|---------|------|------|------|---------|------|
| `name` | Path | ✅ | ✅ | ✅ | ✅ | 完备 |
| `access` | Writable | ✅ | ✅ | ✅ 需要 | ✅ 需要 | **需实现校验和前端** |
| `type` | Type | ✅ | ✅ | ✅ 需要 | ✅ 需要 | **需实现校验和前端** |
| `min` | Constraints.MinValue 或 MinLength | ✅ 部分 | ✅ 部分 | ✅ 需要 | ✅ 需要 | **STRING MinLength 缺失** |
| `max` | Constraints.MaxValue 或 MaxLength | ✅ | ✅ | ✅ 需要 | ✅ 需要 | **需实现** |
| `notify` | Notify | ✅ | ✅ | — | ✅ 需要 | **需前端显示** |
| `forcedInform` | ForcedInform | ✅ | ✅ | — | ✅ 需要 | **需前端标签** |
| `defaultValue` | DefaultValue | ✅ | ✅ | ✅ 需要 | ✅ 需要 | **需前端placeholder + 重置** |
| `changeApplies` | ChangeApplies | ✅ | ✅ | — | ✅ 需要 | **需前端警告提示** |
| `isList` | IsList | ✅ | ✅ | — | ✅ 需要 | **需前端列表标识** |

### XML `<object>` 属性检查

| XML 属性 | Go 字段 | 解析 | 存储 | 校验 | 前端显示 | 状态 |
|----------|---------|------|------|------|---------|------|
| `name` | Name | ✅ | ✅ | ✅ | ✅ | 完备 |
| `access` | Access | ✅ | ✅ | ✅ 需要 | ✅ 需要 | **需实现增删权限校验** |
| `maxInstances` | MaxInstances | ✅ | ✅ | ✅ 需要 | ✅ 需要 | **需实现添加上限校验** |
| `isList` | IsList | ✅ | ✅ | — | ✅ 需要 | **需前端标识** |

### JSON 种子数据特有字段

| JSON 字段 | 说明 | 状态 |
|-----------|------|------|
| `multi_instance: true` | 标记多实例对象 | ⚠️ 仅种子数据有，XML 解析数据无此字段 |
| `description` | 中文参数描述 | ⚠️ 种子数据有，XML 解析后无 description |
| `constraints.enum_values` | 枚举值列表 | ✅ 两种格式都支持 |
| `constraints.pattern` | 正则校验 | ✅ 种子数据支持 |

### 需要统一的地方

1. **XML 解析缺少 description** — XML 格式没有 description 属性，但种子 JSON 有。需要从 TR-196/TR-181 规范数据库补充，或支持管理员手动添加。
2. **multi_instance 标记不统一** — 种子 JSON 用 `multi_instance: true`，XML 解析后通过 object_tree 的 `{N}` 占位符识别。两种数据源需要统一判断逻辑。
3. **STRING 的 MinLength 缺失** — `buildConstraints` 需要补充对 STRING 类型 `min` 属性的解析。
