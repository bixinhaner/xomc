# MML 控制台 — CMCC TD-LTE v2.3 命令树重构调整方案

> 状态：**待审核** · 关联需求：`omcgo/规范/移动/南向数据模型/cmcc-tdlte-southbound-data-model-v2.3.md` § 原始需求 R-1 ~ R-10
>
> 本方案描述对 **MML 控制台**（前端 `omcmb/webcode/src/pages/mml/Console/` + 后端 `omcgo/internal/mml/`）的全部调整。审核通过后再拆 Sprint。

---

## 1. 背景与目标

| 维度 | 目标态 |
|------|--------|
| 命令树权威源 | **CMCC TD-LTE v2.3 规范文档**（18 章节 / 73 命令块 / 625 路径）作为 standard 范围的唯一权威源 |
| 一级分组 | **object 维度**（如"设备信息" / "设备版本升级"），按 SA-SR 顺序排列但**不渲染章节名**；Customized 强制末位 |
| 命令叶子 | **per-operation 展开**：`<OP> <Object>`（如 `LST 设备信息` / `MOD 设备信息`），每条 `mml_commands` 行固定一个 `operation_type` |
| Path 集 | 按 op_type 独立：LST = 全集（含 RO+RW）/ MOD = 仅 RW / ADD-RMV = 父级 object 路径 |
| 设备选择 | 产品类型必选 + 单一性约束（混 `product_class` 后端 400 拦截） |
| API 入参 | 结构化路径数组（替代 MML 文本 round-trip），服务端按 `product_class → productId + swVersion` 翻译 standardPath → privatePath |
| Customized 模板 | 公私模板：Public 全员可见且命令完全平级；Private 始终含"用户名目录"层，普通用户只看自己 / admin 看所有人 |
| UI 文案 | 主 Tab "**操作面板** / Control Panel"；副 Tab "**参数路径指定** / ParameterPath Command" |

**改造性质**：保留现有控制台壳（DeviceTree / CommandTree / RightPanel / TerminalPanel / StepBar），扩展数据模型 + 调整交互。约 75% 基础设施已就绪（mml_tasks 表 / device_tasks 关联 / Translator / ProductRegistry / result_aggregator / SSE 通道），本方案是**接入 + 微调**而非"从零搭建"。

---

## 2. 现状速览

| 维度 | 位置 | 说明 |
|------|------|------|
| 命令树拉取 | `frontend-core/src/hooks/api/useGroupTree.ts` → `GET /mml/group-tree` | 一次性返回全树 + 每命令的 sub_fields |
| Customized 入口 | `Console/components/CommandTree.tsx:146-158` | 末尾固定分组 + `+` 按钮 → `AddTemplateModal`；commit `2849f65d` 上线 |
| 命令模型 | `internal/mml/model.go` MMLCommand | `OperationType varchar` 单值、`TargetPaths []string`、`LogicalNameI18n map`、`Source(standard|admin)` |
| 子字段模型 | `internal/mml/sub_field_model.go` MMLCommandSubField | 关联 `param_id` → `standard_params`、`DefaultSelected`、`IsRequired`、`SortOrder` |
| 分组存储 | `internal/mml/group_tree_repository.go` + `mml_param_groups`（LTREE） | 老 OMC catalog 导入；source=standard 不可删 |
| MML 任务表 | `internal/mml/model.go:147` MMLTask + `migrations/000120` | 含 15 列统计字段（status / total_devices / success_count / failed_count / result 等） |
| 任务关联 | `internal/task/model.go:60` device_tasks.{source='mml',source_id,command_index,device_index} | 一个 mml_task = M commands × N devices = M×N device_tasks |
| 路径展开 | `internal/mml/tr069_payload.go` validatePath / expandInstancePaths | 当前拒绝含 `{i}` 占位符未替换的路径 |
| 执行入口 | `internal/mml/console_executor.go:68` ExecuteStatements | 调 BuildStatementCommands + CreateAndFanoutTask |
| 翻译层 | `internal/config/parammodel/translator.go:91` Translator.ToPrivate / ToStandard | O(1) map + passthrough fallback 已实现 |
| 产品路由 | `internal/product/registry.go:158` ProductRegistry.MatchProductClass | 正则路由返回 Product (含 product_id) |
| 结果回流 | `internal/mml/result_aggregator.go` + `internal/task/completion_router.go` | source 路由 + IncrementStats 原子更新 |
| 实时回显 | `result_aggregator.publishDeviceFrame` SSE + `TerminalPanel.tsx` lines prop | SSE 通道 `mml_device_frame` executor-based 路由 |
| 标准命令现状 | seed/000111 + 000116 + 000126 | 老 OMC catalog 导入，与 v2.3 规范未对齐 |

---

## 3. 差距分析（按 R-1 ~ R-10 对照）

| 需求 | 现状 | 差距 | 影响域 |
|------|------|------|--------|
| **R-1** 命令树两层：level-1 = object 组，**取消 SA-SR 章节层级** | 现 mml_param_groups 为 2 层（业务大类 → 命令） | 章节信息退化为 catalog 元数据（chapter_code 仅用于排序）；level-1 = object（按 `####` 块归一化路径） | 后端 schema + 前端 + Loader |
| **R-2** 每个 `####` 块按权限矩阵展开为 1-4 行 mml_commands（per-operation）；叶子命令名 = `<OP> <Object>` | mml_commands.operation_type 已单值，但 standard 内容未按 v2.3 拆分 | Loader 按 LST/MOD/ADD/RMV 条件展开；前端树叶子直接是操作命令 | 后端 Loader + 前端 |
| **R-3** 每条命令固定 op_type；右侧操作面板按命令固定 op 单一渲染 | operation_type 已是单值 | 继续沿用现有 schema；前端**不实现 OperationType segmented 切换条** | 前端 |
| **R-4.1** LST `{i}` 留空 = partial path | `expandInstancePaths` 拒绝未替换 `{i}` | LST 命令模式允许保留 partial path（截断到 `.` 结尾） | 后端 tr069_payload |
| **R-4.2/4.3** MOD/ADD 的 target_paths 仅含 RW；ADD 含父级 object 路径 | 当前 target_paths 不按 op 拆分 | Loader 写入时按 op_type 过滤；前端按 op_type 渲染 | 后端 Loader + 前端 |
| **R-4.4** RMV 的 target_paths = 父级对象路径；实例号由 InstancePicker 多选提供 | InstancePicker 单选 | 扩展为多选 + 多任务批量下发 | 前端 + 后端 |
| **R-5.0** Customized 固定在 CommandTree **末尾** | 现已是末尾固定 | 沿用 | — |
| **R-5.1** Public Template 全员可见，**命令完全平级，禁止嵌套** | 现已全员可见 | 后端校验 visibility='public'；前端渲染仅 1 级叶子 | 后端 + 前端 |
| **R-5.2** Private Template **始终含用户名目录层**；普通用户只看自己的目录，admin 看所有人 | mml_commands 未必有 owner_user_id 字段 | 新增 `visibility ENUM('public','private')` + `owner_user_id UUID`；BuildTree 按 owner 聚合返回；普通用户与 admin 树结构完全一致 | 后端 schema + service + 前端 |
| **R-5.3** admin 角色见所有用户目录（含 admin 自己） | 现状未做用户分层 | BuildTree 在 admin 角色下不做 owner 过滤，按 owner_user_id 分组返回 | 后端 service + 前端 |
| **R-8.1/8.2** 产品类型默认第一项、必选、禁 allowClear | `useDeviceSelection.ts:9` 默认 `''`；`DeviceTree.tsx:128` 启用 `allowClear` | DeviceTree + Hook 首屏取字典首项；Select 去 allowClear；字典空 → 禁用 + 提示 | 前端 |
| **R-8.3** 切换产品类型清空已选设备 | `handleFilterChange` 仅重置分页 | 切换时 `setSelectedDevices([])` + toast | 前端 |
| **R-8.4** 后端拒绝混类型批量执行 | execute-statements 不校验 product_class 一致性 | 新增 `ErrDevicesMixedProductClass`；service 入口按 SN 批量查 `devices.product_class` 分组校验 | 后端 |
| **R-8.5** 命令-产品类型兼容性提示（P2 可选） | 无 | 拉取选中产品类型 ParamModel 路径集与命令路径集比对，命令节点带警告标 | 后端 + 前端 |
| **R-9.1** 移除 Control Panel 的 MML 文本预览 | `RightPanel.tsx:50` 含 `MmlEditor` | 删除 MmlEditor 实例化；执行按钮移至底部 `ConsoleActionBar`；TerminalPanel 内容切换为"执行后回显" | 前端 |
| **R-9.2** API 入参改为结构化 path 数组 | `console_handler PostExecuteStatements` 消费 MML 文本 statements | 入参 schema 重构为 `{paths[], values{}, instance_selectors, instance_indices}` | 后端 contract + 前端 |
| **R-9.3** 服务端按 `product_class → productID + software_version` 翻译 standardPath → privatePath | fanout 前无翻译 | `MMLTaskCreator.CreateAndFanoutTask` 调链路 `ProductRegistry.MatchProductClass → product.id → Translator.ToPrivate`；passthrough 透传 | 后端 |
| **R-9.4** 多设备独立翻译（softwareVersion 差异） | — | 每个 device_task 翻译一次；不做批量复用 | 后端 |
| **R-9.5** Customized 走同一结构化通道 | Customized 已绑定 sub_fields | 默认走结构化 API；裸 MML 用法见 D15 | 后端 + 前端 |
| **R-10** UI 文案：Tab 中英双语 | 现仅英文 `Control Panel` / `Param Path Expert` | 中文 "操作面板" / "参数路径指定"；英文 `Control Panel` / `ParameterPath Command`；可选物理 rename 组件文件 | 前端 i18n + 可选 rename |

---

## 4. 目标态架构

### 4.1 数据流（导入侧 · dictloader 模式）

```
cmcc-tdlte-southbound-data-model-v2.3.md            ← 人类编辑（PR diff 友好）
        │
        │  ① 离线（Python / dev-only，不进生产镜像）
        ▼
omcgo/datamodels/mml-catalog/cmcc-tdlte-v23.json    ← 结构化数据文件（commit + ship）
        │
        │  ② Go binary 启动 / 热重载（生产路径）
        ▼
internal/mml/catalogloader/  (dictloader.Loader)
   · 单事务 UPSERT、source='standard'、carrier='cmcc'、tech='lte'
   · 差集软删（仅 standard 范围）；admin / Customized 不动
   · sub_field default_selected 由 override 表保护
        │
        ▼
mml_param_groups (72 行 standard, object 维度)
mml_commands     (per-op 行，~150-200 行 standard，target_paths 按 op 独立)
mml_command_sub_fields (group 维度共享，按 op_type 过滤展示)
```

> 生产部署不依赖 Python；脚本仅在开发阶段把 MD 转成 JSON，产物随 PR 提交。详见 §5。

### 4.2 数据流（运行时）

```
前端 CommandTree → 用户选中 per-op 命令（op_type 由命令本身确定）
        │
        ▼
RightPanel 按选中命令的 operationType 单一渲染：
  LST → SubFieldChecklist（全 path，含 RO）
  MOD → SubFieldInputList（仅 RW，带值）
  ADD → SubFieldInputList（RW 初始值，对象路径=parent）
  RMV → InstancePicker（多选）
        │
        ▼   POST /mml/console/execute-statements（结构化 payload）
        ▼
console_service.ExecuteStatements
  → 校验 R-8.4 device_sns 同 product_class
  → 校验 R-3 statement.operation_type == command.operation_type
  → BuildStatementCommands 编译 statements
  → 创建 mml_task (source='mml')
  → fanout per-device：
      · 解析 devices.product_class → ProductRegistry → product.id
      · GetTranslator(product.id, dev.software_version)
      · ToPrivate(standardPath) 翻译，passthrough 兜底
      · 写 device_task.params（privatePath + source 标签）
  → enqueue → Redis Sorted Set + PG device_tasks
        │
        ▼
ACS 消费 device_task → dispatcher 渲染 SOAP → 下发 CPE
        │
        ▼   CPE 响应
        ▼
ACS handler 发布 command.{rpc}.response 事件（含 task_id）
        │
        ▼
completion_router (按 source 路由) → mml.ResultAggregator.OnTaskCompleted
        │
        ├─ IncrementStats(mml_task_id, successDelta, failedDelta)
        └─ publishDeviceFrame → SSE mml_device_frame
                                       │
                                       ▼
                       前端 TerminalPanel 接收，按 source 上色回显
                       standardPath ↔ privatePath + RPC 响应摘要
```

---

## 5. 数据导入方案（dictloader 模式 · 生产零脚本依赖）

> **设计原则**：生产部署产物 = **Go binary + 结构化数据文件（JSON）+ DB**。Python 等脚本语言**不进生产镜像 / 不进运行时**。与现有 `param_models / products / indicators / alarms` 4 个 Loader 设计对齐（整合方案 §5 `internal/core/dictloader/`），**MML catalog 是该模式的第 5 个 Loader**。

### 5.1 解析脚本（Python · 仅离线开发工具）

- **职责**：MD spec → 结构化数据文件的转换。开发者本地运行，产物随 PR 提交。
- **不进生产**：
  - 不进 Dockerfile / 镜像
  - 不在生产 CI/CD 流水线作为强依赖（可选 lint job 校验"MD 修改了但 JSON 未同步"）
  - 不被任何 Go 代码 import 或 shell out
- **位置**：`omcgo/migrations/scripts/parse_cmcc_tdlte_v23.py`（与 `import_old_mml_catalog.py` 同级；明确不在 production 路径，Dockerfile 不 COPY `scripts/` 目录）
- **失败容忍**：脚本丢失不影响生产；结构化文件可由人手编辑兜底。

**输入**：`omcgo/规范/移动/南向数据模型/cmcc-tdlte-southbound-data-model-v2.3.md`
**输出**：`omcgo/datamodels/mml-catalog/cmcc-tdlte-v23.json`

**解析逻辑**：
1. 解析 H2（`## SA - DeviceInfo` 等）→ 18 个 chapter（`chapterCode='SA'`、章节描述用于排序元数据）
2. 解析 `#### 命令: <path>` → object 一级分组（72 个，剔除 SQ 段伪命令 `.*`）
3. 解析表格 → sub_fields；TR-181 路径为权威，TR-098 作 alias
4. 按权限矩阵展开为 1-4 行 commands（per-op_type）：
   - `LST` 恒生成
   - 有 📝 行 → 加入 `MOD`
   - 模板含 `{i}` 且有 📝 → 加入 `ADD` + `RMV`
5. 多层 `{iα}/{iβ}/{iγ}` 折叠为 `{i}`，记录 `instanceArity` + `instanceLevels`（如 `["MU","Slot","EU","RU","RFChannel"]`）
6. 显示名生成：`<对象的中文名> <TR-181 路径模板>`，中文名取自表格"参数名 = 路径末段对象名"对应的"中文名"
7. 输出含 `generatedAt` + `sourceDocSha256`，便于 CI 校验 MD 与 JSON 同步

### 5.2 结构化数据文件（生产唯一数据源）

**路径**：`omcgo/datamodels/mml-catalog/cmcc-tdlte-v23.json`
**格式**：JSON（D8 决策）
**生命周期**：随代码提交、随 Go binary 部署、在启动期 / 热重载时被 Loader 消费

**Schema**：

```jsonc
{
  "specVersion": "cmcc-tdlte-v2.3",
  "carrier": "cmcc",
  "tech": "lte",
  "generatedAt": "2026-05-19T12:00:00Z",
  "sourceDocSha256": "<sha256 of source MD file>",
  // groups = object-level groups（一级树节点），与 spec 章节 SA-SR 解耦
  "groups": [
    {
      // group_code = TR-181 路径模板规范化
      "groupCode": "Device.DeviceInfo.*",
      "nameI18n": { "zh-CN": "设备信息", "en-US": "Device Info" },
      "chapterCode": "SA",     // ← spec 章节元数据，仅用于排序，不渲染
      "displayOrder": 1,        // SA…SR 自然顺序
      "instanceArity": 0,
      "instanceLevels": [],
      // commands = per-operation 行（二级树叶子）
      "commands": [
        {
          "operationType": "LST",
          "targetPaths": [
            "Device.DeviceInfo.UserLabel",
            "Device.DeviceInfo.Manufacturer"
            // ... 全 17 readable paths
          ]
        },
        {
          "operationType": "MOD",
          "targetPaths": [
            "Device.DeviceInfo.UserLabel",
            "Device.DeviceInfo.DnPrefix"
          ]
        }
      ],
      // subFields 在 group 维度保存一次（共享于所有 op 命令），渲染时按 op 过滤
      "subFields": [
        {
          "mmlCode": "UserLabel",
          "standardPath": "Device.DeviceInfo.UserLabel",
          "labelI18n": { "zh-CN": "用户友好名", "en-US": "User Label" },
          "accessType": "RW",
          "valueType": "string",
          "constraint": "string(256)",
          "defaultSelected": true,
          "sortOrder": 1
        }
        // ...
      ]
    }
  ]
}
```

**关键字段**：
- `sourceDocSha256` — 当前 spec MD 的 SHA256；CI 校验 hash 与 MD 文件一致
- `specVersion` — 多版本并存的 namespace（未来 v2.4 可与 v2.3 共存）
- `chapterCode` — SA…SR 元数据，**仅用于排序与管理后台**，不渲染到用户树
- `groupCode` — object 级稳定标识 = 归一化 TR-181 路径模板
- `commands[].operationType` — 每条命令固定 op_type；`commands` 数组长度 1-4 由权限矩阵决定
- `commands[].targetPaths` — 该 op 的执行路径集（LST=全集 / MOD=仅 RW / ADD-RMV=父级 object 路径）
- `subFields` — group 维度保存一次（共享），渲染时按 op 过滤
- `instanceLevels` — 多层 `{i}` 命令的层级语义名，供前端 InstanceArityInput 渲染

### 5.3 唯一约束键

- groups: `(carrier='cmcc', tech='lte', source='standard', group_code)`，group_code = 归一化 TR-181 路径模板
- commands: `(source='standard', group_id, operation_type)`，每 group 内 op_type 唯一
- sub_fields: `(group_id, mml_code)`，同 group 的 LST/MOD/ADD/RMV 共享 sub_fields，按 op 过滤展示

### 5.4 Loader 行为契约

- 启动期：读取 `omcgo/datamodels/mml-catalog/*.json` 白名单 → 单事务 UPSERT；任何文件解析失败 → app 启动失败（fail-fast）
- 热重载：`POST /api/v1/mml/catalog/reload` → 同一进程内重跑 UPSERT；与 `parammodel:cache/refresh` 同一权限模型
- 删除策略：仅 `source='standard'` 范围内做差集软删（标 `deprecated_at`），不动 `source='admin'` / Customized
- sub_field 用户配置保护：通过 `mml_command_sub_field_overrides` 表保存用户调过的 `default_selected` / `label_i18n`（per-user 维度，详见 §8.3），Loader 只写基础列，运行期读取走 `JOIN + COALESCE`

---

## 6. 后端调整

### 6.1 Schema 变更（**先决项 P1**）

| 表 | 变更 | 必要性 |
|----|------|-------|
| `mml_param_groups` | 新增 `carrier varchar(8)`、`tech varchar(8)`、`group_code varchar(128)`（对象路径模板）、`chapter_code varchar(8)`、`instance_arity smallint NOT NULL DEFAULT 0`、`instance_levels text[]`、`deprecated_at timestamptz`；唯一索引 `(carrier, tech, source, group_code) WHERE source='standard'` | R-1 object 一级分组、章节信息退化为 metadata |
| `mml_commands` | 沿用 `operation_type varchar(8) NOT NULL`（单值）；新增 `target_paths text[] NOT NULL`（按 op_type 独立）、`deprecated_at`；唯一索引 `(source, group_id, operation_type) WHERE source='standard'`<br>**Customized 扩展**：新增 `visibility varchar(8) DEFAULT 'private' CHECK (visibility IN ('public','private'))`、`owner_user_id UUID`；对 `source='admin'` 行强制 NOT NULL `owner_user_id` | R-2/R-3 per-op 行；R-5 公私模板可见性 |
| `mml_command_sub_fields` | FK 从 `command_id` 改为 `group_id`（或新增 `group_id` + 保留 `command_id` 双 FK 兼容期）；补 `access_type varchar(4)` 冗余列（RO/RW），加速前端按 op 过滤 | R-2 sub_fields 跨 LST/MOD/ADD/RMV 共享 |
| `mml_command_sub_field_overrides` | 新表：`(sub_field_id, owner_user_id) UNIQUE`，列 `default_selected_override boolean`、`label_i18n_override jsonb`；保护用户调过的默认勾选不被 Loader 覆盖 | sub_field 用户配置保护 |

**Customized 兼容**：现网已有 admin 行 → DDL 加列时 default='private'，按 `creator_user_id`（若已存在）或 NULL 回填 `owner_user_id`；NULL 行视为"遗留公共"，本期不强制清理，前端展示在 Public Template 下。

### 6.2 路径处理调整

文件：`internal/mml/tr069_payload.go`

| 函数 | 调整 |
|------|------|
| `validatePath` | LST 命令模式放行 partial path（含未替换 `{i}` 的路径在 LST 内截断到最后一个 `.` 后保留）；MOD/ADD/RMV 仍强制实例化 |
| `expandInstancePaths` | 新增"实例号填写区"入参（`instanceSelectors map[level]string`），按层级替换 `{iα}/{iβ}` 折叠的 `{i}`；支持范围 `1,2,5` 拆为多 path |

### 6.3 Renderer / Parser

文件：`internal/mml/mml_renderer.go` / `mml_parser.go`

- LST partial path 渲染：路径以 `.` 结尾时不附加叶子参数列表（CWMP partial path 写法）
- RMV 多实例：单 statement 渲染为多条 `RMV CMD: Device.X.{i=3}` / `{i=5}` 等，提交时拆为多个 DeleteObject RPC

### 6.4 ConsoleService 执行链

文件：`internal/mml/console_service.go` / `console_executor.go`

**校验顺序**（在 ExecuteStatements 入口集中做）：
```
1. R-8.4  DeviceSns → devices.product_class 一致性校验    → ErrDevicesMixedProductClass / ErrNoValidDevices
2. R-3    每个 statement.operation_type == command 自身 op_type → ErrOperationMismatch
3. R-9.2  统一结构校验：
          - LST: paths ⊆ command.target_paths
          - MOD: paths ⊆ command.target_paths 的 RW 子集；values 的 key = paths
          - ADD: paths ⊆ command.target_paths 的 RW 子集（视为初始值字段）
          - RMV: instance_indices ≥ 1 且 command.instance_arity > 0
          - 含 {i} 的命令在 MOD/ADD 必须填齐 instance_selectors
                                                          → ErrInvalidStatementPayload
4. R-9.3  product_class → product_id 解析（ProductRegistry）→ ErrProductClassUnresolved（孤儿设备）
5. R-9.3  逐设备调用 Translator → 至少一条 path 翻译成功（含 passthrough）
                                                          → ErrTranslationFailedAll（passthrough 兜底，理论不触发）
```

**错误码新增**（`omcgo/global/errors.go`）：

| Code | HTTP | 含义 |
|------|------|------|
| `ErrDevicesMixedProductClass` | 400 | execute-statements 入参设备包含多种 `product_class` |
| `ErrNoValidDevices` | 400 | execute-statements 入参设备为空或全部失效 |
| `ErrOperationMismatch` | 400 | statement.operation_type 与 command 自身 op_type 不一致 |
| `ErrInvalidStatementPayload` | 400 | 结构化入参缺字段、paths 与 command 路径集不匹配、values key 不在 paths 内等 |
| `ErrTranslationFailedAll` | 422 | 全部 path 翻译失败（passthrough 关闭时） |
| `ErrProductClassUnresolved` | 422 | `devices.product_class` 在 ProductRegistry 找不到匹配 product（孤儿设备） |

**ADD 流程**：在同一 device_task 内串行下发 `AddObject(parentPath)` → `SetParameterValues(initial values)`（依赖 task 子任务能力；若不支持复合 RPC，先按两次 task 入队）。

**RMV 多实例**：拆为多个 task 或多个 instance 的批量 RPC（取决于 ACS `DeleteObject` 实现的批量能力，本期先按 N 个独立 task 处理）。

### 6.5 GroupTree 拉取

文件：`internal/mml/group_tree_repository.go`

- BuildTree SQL 返回 `group → [commands by op_type] → sub_fields (group-shared)` 三层结构
- **Customized 可见性过滤**（R-5.2/5.3）：BuildTree 接收 `current_user_id` + `is_admin` 参数：
  - 非 admin：`WHERE source='standard' OR (source='admin' AND visibility='public') OR (source='admin' AND visibility='private' AND owner_user_id = current_user_id)`
  - admin：`WHERE source='standard' OR source='admin'`（无 owner 过滤）
- **Private Template 始终按 owner_user_id 分组返回**（即便普通用户只能看到自己一个分组）：
  - 后端按 `owner_user_id` 聚合查询，返回 `[{ ownerUserId, ownerUsername, commands: [...] }, ...]`
  - 普通用户：列表长度恒为 1（仅当前用户）
  - admin：列表长度 = N（所有有 private 命令的用户数）
  - 前端按相同结构渲染，无需 admin / 非 admin 特殊判断
- admin 视角下私有命令需要返回 `owner_username`（user 表 join），用于树上显示用户名目录
- **排序逻辑**：返回的 group 列表中 Customized 永远在末位（service 层强制 append；不依赖 chapter_code / display_order）
- 前端 query key 加 `version='cmcc-v2.3'` 与 `current_user_id` 维度

### 6.6 结构化执行入参与服务端翻译（R-9）

文件：
- `internal/mml/console_handler.go` `PostExecuteStatements`
- `internal/mml/console_service.go` `ExecuteStatements`
- `internal/config/parammodel/translator.go`（**只读消费方**，不动）
- `internal/mml/tr069_payload.go`

#### 6.6.1 入参类型

```go
type ExecuteStatementsRequest struct {
    DeviceSns  []string              `json:"device_sns" binding:"required,min=1"`
    Statements []StructuredStatement `json:"statements" binding:"required,min=1"`
}

type StructuredStatement struct {
    CommandID         string            `json:"command_id" binding:"required,uuid"`
    CommandCode       string            `json:"command_code"`
    OperationType     string            `json:"operation_type" binding:"required,oneof=LST MOD ADD RMV"`
    Paths             []string          `json:"paths"`              // standardPath 子集
    Values            map[string]string `json:"values,omitempty"`   // MOD/ADD 时填，key 必须 ⊆ Paths
    InstanceSelectors map[string]string `json:"instance_selectors,omitempty"`  // {iα→"1", iβ→"2"}
    InstanceIndices   []int             `json:"instance_indices,omitempty"`    // RMV 多选
}
```

#### 6.6.2 Translator 调用契约

解析链路：`devices.product_class → ProductRegistry → product.id → Translator.ToPrivate(productID, swVersion, standardPath)`

```go
// 伪代码：fanout 前的翻译
for _, sn := range req.DeviceSns {
    dev := deviceRepo.GetBySN(sn)             // 含 ProductClass、SoftwareVersion
    product, ok := productRegistry.MatchProductClass(dev.ProductClass)
    if !ok {
        return errors.New(ErrProductClassUnresolved, "product_class %s not matched", dev.ProductClass)
    }
    translator := translatorFactory.Get(product.ID, dev.SoftwareVersion)  // 缓存 by (productID, swVersion)
    for _, stmt := range req.Statements {
        translated := make([]TranslatedPath, 0, len(stmt.Paths))
        for _, sp := range applyInstanceSelectors(stmt.Paths, stmt.InstanceSelectors) {
            result := translator.ToPrivate(sp)
            if result.Found {
                translated = append(translated, TranslatedPath{
                    Standard: sp, Private: result.Translated, Source: result.Source,
                })
            } else {
                // passthrough — 不命中也不阻塞
                translated = append(translated, TranslatedPath{
                    Standard: sp, Private: sp, Source: "passthrough",
                })
            }
        }
        // 用 translated[].Private 构造 task params；同时把 standard / source 记入 task_log
        taskCreator.Enqueue(dev, stmt.OperationType, translated, stmt.Values)
    }
}
```

`applyInstanceSelectors` 把 `Device.X.{i}.Y.{i}.Z` + `{iα:1, iβ:3}` 折叠为 `Device.X.1.Y.3.Z`（LST 留空 → partial path）。

> **devices 表关键列**：
> - `product_class` (string) — TR-069 标准 `Device.DeviceInfo.ProductClass`，CPE Inform 时上报，R-8.4 校验主键
> - `software_version` (string) — `Device.DeviceInfo.SoftwareVersion`，R-9.3 discovered 映射 swVersion 入参
> - 若已缓存 `product_id` 外键，可省略 `ProductRegistry.MatchProductClass` 调用（性能优化，P2 可选）

#### 6.6.3 values 字段的 key 同步翻译

MOD/ADD 的 `values: map[standardPath]string` 需要在翻译完成后，把 key 同步替换为 privatePath（或 passthrough 的 standardPath），写入 task.params。

#### 6.6.4 TerminalPanel 回显内容

执行 API 响应体新增 `executed_paths_per_device`：

```json
{
  "task_id": "...",
  "executed_paths_per_device": {
    "SN001": [
      {"standard": "Device.X.UserLabel", "private": "Device.X.UserLabel", "source": "passthrough"},
      {"standard": "Device.X.SoftwareVersion", "private": "Device.X.X_VENDOR_SwVer", "source": "discovered"}
    ]
  }
}
```

前端 TerminalPanel 渲染：每设备一段，显示 `<source 标签> standardPath → privatePath`；同源（passthrough）的合并为单行。

### 6.7 Catalog Loader（mml-catalog · 生产唯一数据入口）

新增 Go 包：`omcgo/internal/mml/catalogloader/`（与现有 `internal/config/parammodel/` Loader 同级）。

#### 6.7.1 包结构

```
internal/mml/catalogloader/
├── loader.go          # 实现 dictloader.Loader 接口（Load / Validate / Reload）
├── parser.go          # JSON → 内存 Catalog 结构体
├── upsert.go          # 单事务 UPSERT 到 mml_param_groups / mml_commands / sub_fields
├── differ.go          # 计算 standard 范围内"本次未覆盖"的命令 → 标 deprecated
├── model.go           # Go 结构体（与 §5.2 JSON schema 对齐）
├── validator.go       # 校验：(group_id, operation_type) 唯一、target_paths 与 op_type 一致、{i} 数量一致
└── loader_test.go
```

#### 6.7.2 接口契约（实现 dictloader.Loader）

```go
type Loader struct {
    db        *pgxpool.Pool
    dataDir   string      // 默认 "omcgo/datamodels/mml-catalog"
    fileGlob  string      // 默认 "*.json"
    logger    *zap.Logger
}

// 启动期调用（fail-fast）：解析所有白名单 JSON 文件 → 单事务 UPSERT
func (l *Loader) Load(ctx context.Context) error

// 校验数据完整性，不写库；可被 admin API 提前调用做 dry-run
func (l *Loader) Validate(ctx context.Context) (*ValidationReport, error)

// 热重载入口（POST /api/v1/mml/catalog/reload）
func (l *Loader) Reload(ctx context.Context) (*ReloadReport, error)

// 当前已加载的 spec 元信息（用于 /api/v1/mml/catalog/info）
func (l *Loader) Info() []*CatalogInfo
```

#### 6.7.3 UPSERT 事务（伪代码）

```go
func (l *Loader) upsertAll(ctx context.Context, tx pgx.Tx, catalog *Catalog) error {
    incomingGroupCodes := []string{}
    for _, g := range catalog.Groups {
        // 1. UPSERT group：(carrier, tech, source='standard', group_code) → name_i18n / display_order
        gid := upsertGroup(tx, catalog.Carrier, catalog.Tech, g)
        incomingGroupCodes = append(incomingGroupCodes, g.GroupCode)

        incomingCmdOps := []string{}
        for _, c := range g.Commands {
            // 2. UPSERT command：(source='standard', group_id, operation_type) → 全字段
            upsertCommand(tx, gid, c)
            incomingCmdOps = append(incomingCmdOps, c.OperationType)
        }
        // 3. 差集软删 commands（仅 source='standard' 同 group_id 下未在 incomingCmdOps 的 op）
        softDeleteOrphanCommands(tx, gid, incomingCmdOps)

        incomingFields := []string{}
        for _, sf := range g.SubFields {
            // 4. UPSERT sub_field：(group_id, mml_code) → 全字段（不动 default_selected，由 override 表保护）
            upsertSubField(tx, gid, sf)
            incomingFields = append(incomingFields, sf.MMLCode)
        }
        // 5. 差集软删 sub_field（仅 source='standard' 同 group_id 下未在 incomingFields 的行）
        softDeleteOrphanSubFields(tx, gid, incomingFields)
    }
    // 6. 差集软删 groups（仅 source='standard' 且 carrier/tech 匹配的）
    softDeleteOrphanGroups(tx, catalog.Carrier, catalog.Tech, incomingGroupCodes)
    return nil
}
```

软删除使用 `deprecated_at TIMESTAMPTZ` 列；列表查询默认过滤 `deprecated_at IS NULL`。

#### 6.7.4 ModuleGraph 接线

在 `cmd/app/router/deps.go` 的启动顺序中加入：

```
dictload ─┬─ parammodel
          ├─ product
          ├─ indicator
          ├─ alarm-definition
          └─ mml-catalog   ← 新增（无依赖，可与其他 4 个 Loader 并行）
```

#### 6.7.5 admin REST API

```
GET    /api/v1/mml/catalog/info                # 已加载 spec 列表 + sourceDocSha256 + generatedAt
POST   /api/v1/mml/catalog/reload              # 重新读 JSON 文件 → UPSERT（super_admin only）
POST   /api/v1/mml/catalog/validate            # dry-run 校验，不写库
GET    /api/v1/mml/catalog/diff?file=xxx.json  # 比对当前 DB 与待加载 JSON 的差异
```

> 不开放手动空建（命令树只能通过结构化文件加载）。Customized 命令仍走旧 admin 接口（`POST /api/v1/mml/commands` 等，source='admin'）。

#### 6.7.6 失败处理与可观测性

- 启动期 Loader 失败 → app 启动失败（与 parammodel Loader 一致）；deployment 应在健康检查 fail 时回滚到上一版本
- 热重载失败 → 事务回滚，DB 保持原状；API 返回 422 + 错误详情；监控 alert
- 指标：
  - `mml_catalog_load_total{status=success|fail}` counter
  - `mml_catalog_groups_total / commands_total / subfields_total` gauge
  - `mml_catalog_deprecated_total` gauge（标 deprecated 的行数，>0 时关注是否预期）
- 日志：每次 Load / Reload 记录 spec_version + sha256 + 行数对比（before/after）

---

## 7. 前端调整

### 7.1 CommandTree（按 object 一级分组 + per-op 叶子 · R-1/R-2）

文件：`omcmb/webcode/src/pages/mml/Console/components/CommandTree.tsx`

| 变更点 | 说明 |
|--------|------|
| 一级树节点 | object 组（即 spec 中每个 `####` 块），如"设备信息"、"设备版本升级"、"当前告警实例"；**不渲染 SA-SR 章节名** |
| 一级分组排序 | 按 `chapter_code + display_order` 升序保留 SA…SR locality；**Customized 强制末位** |
| 一级分组显示名 | 取 `nameI18n[currentLang]`（如"设备信息"）；副标题或 tooltip 显示路径模板 |
| **二级叶子节点** = `<OP> <Object>` | 直接对应 mml_commands 行；operation_type 决定显示前缀（LST/MOD/ADD/RMV） |
| 叶子点击 | 直接选中这条 (group, op_type) 命令；右侧 RightPanel 按命令的 op_type 单一渲染 |
| **Customized 子树** | 末位一级分组下：<br>· `Public Template` (二级容器) → 三级叶子（完全平级，禁止任何嵌套）<br>· `Private Template` (二级容器) → 三级用户名容器 → 四级叶子（命令完全平级）<br>　· 非 admin：仅显示自己一个用户名目录<br>　· admin：显示所有用户名目录（含 admin 自己） |
| 数据缓存 key | `['mml','console','group-tree', version, currentUserId, isAdmin]`；切换用户 / 角色需 invalidate |

### 7.2 操作面板（Control Panel · R-3/R-9.1/R-10）

文件：`omcmb/webcode/src/pages/mml/Console/components/RightPanel.tsx`

| 变更点 | 说明 |
|--------|------|
| 不再有"操作切换条" | 命令的 op_type 在树上选中即确定；切换命令 = 切换执行上下文 |
| 渲染逻辑 | 根据 `selectedCommand.operationType` 单一渲染：`LST → SubFieldChecklist` / `MOD → SubFieldInputList(RW)` / `ADD → SubFieldInputList(RW initial)` / `RMV → InstancePicker(多选)` |
| 切换树叶子时**完全丢弃**上一条命令的勾选与填值 | 切换命令 = 切换执行上下文；不再保留交集（用户已切到不同命令）；有未保存输入时弹 confirm |
| SubField 过滤 | 按 `sub_field.accessType` 过滤：MOD 模式自动隐藏 RO；LST 全显；同 group 的 sub_fields 在 DB 中共享一份 |
| `InstanceArityInput` 新组件 | 命令 `instanceArity > 0` 时显示；LST 允许留空 (partial path)，MOD/ADD/RMV 必填 |
| 删除 `MmlEditor` 组件实例化 | `RightPanel.tsx:50` 处的 `<MmlEditor />` 整体移除；不再有命令文本预览 |
| **执行按钮**迁移到底部 `ConsoleActionBar` | `[ 校验 ] [ 全选 ] [ 全不选 ] [ 执行 <OP> ]`，按钮名显式带 op_type；数据从 statement state 取（paths / values / instance_selectors / instance_indices） |
| **TerminalPanel 内容**切换 | 由"用户预览的 MML 文本"切换为"执行后服务端回显的 standardPath → privatePath + RPC 响应摘要"；执行前显示空 / 历史 |
| **Tab 重命名** (R-10) | 主 Tab："Control Panel" → 中文"操作面板"/英文"Control Panel"；副 Tab："Param Path Expert" → 中文"参数路径指定"/英文"ParameterPath Command"；i18n key 详见 §7.5 |

### 7.3 SubField 组件细化

| 文件 | 调整 |
|------|------|
| `SubFieldChecklist.tsx` | 接收 `mode: 'LST'`；展示 RO + RW 全部 |
| `SubFieldInputList.tsx` | 接收 `mode: 'MOD'\|'ADD'`；自动过滤 RO；ADD 模式按"可选填初始值"处理，未填值字段不下发 |
| `InstancePicker.tsx` | 升级为多选 + range 输入；提交时返回 `instanceIndices: number[]` |

### 7.4 类型与 Hook

文件：`omcmb/frontend-core/src/types/mmlConsole.ts` / `hooks/api/useGroupTree.ts`

- `BackendGroup`: `{ groupCode, nameI18n, chapterCode, displayOrder, instanceArity, instanceLevels, subFields[] }` — object 维度
- `BackendCommand`（per-op 行）: `{ id, groupId, operationType, targetPaths: string[] }`
- `BackendCustomCommand`（Customized）: `{ ..., visibility: 'public'|'private', ownerUserId, ownerUsername }`
- `useGroupTree(currentUserId, isAdmin)` 透传后端 BuildTree，按用户角色返回过滤后的树
- `useExecuteStatements` 前端校验 `statement.operationType === selectedCommand.operationType`（恒真，软校验防异步竞态）

> **多皮肤影响**：上述类型与 hook 全部位于 `frontend-core/`，会影响 `webcode-v2/` `webcode-v3/`；需在 PR 中跑 3 个皮肤的 `npm run typecheck`。

### 7.5 API 客户端与状态映射（R-9.2 / R-10）

文件：
- `omcmb/frontend-core/src/services/api/mmlApi.ts`
- `omcmb/frontend-core/src/hooks/api/useExecuteStatements.ts`
- `omcmb/frontend-core/src/types/mmlConsole.ts`

| 变更点 | 说明 |
|--------|------|
| 类型 `Statement` | 新增字段：`paths: string[]`、`values: Record<string,string>`、`instanceSelectors: Record<string,string>`、`instanceIndices: number[]`；保留 `operationType` 单值 |
| `useExecuteStatements` | 提交前从 statement state 生成 R-9.2 payload；HTTP 客户端自动 camelCase → snake_case |
| 响应消化 | 渲染 `executed_paths_per_device` 到 TerminalPanel；按 `source` 上色（discovered=绿 / default=蓝 / passthrough=灰） |

**i18n key 新增**（zh-CN / en-US 同步）：
- `mml.tabs.controlPanel` → "操作面板" / "Control Panel"
- `mml.tabs.parameterPathCommand` → "参数路径指定" / "ParameterPath Command"
- `mml.deviceTree.productClassRequired` → "请先选择产品类型" / "Select a product class"
- `mml.deviceTree.productClassSwitched` → "产品类型已切换，已清空选中设备" / "Product class changed, selected devices cleared"
- `mml.deviceTree.productClassDictEmpty` → "未配置产品类型，请联系管理员" / "No product class configured"
- `mml.execute.mixedProductClass` → "选中设备包含多种产品类型，不允许一起执行" / "Selected devices have multiple product classes"

### 7.6 DeviceTree（产品类型强约束 · R-8）

文件：
- `omcmb/webcode/src/pages/mml/Console/components/DeviceTree.tsx`
- `omcmb/webcode/src/pages/mml/Console/hooks/useDeviceSelection.ts`

> **命名说明**：前端 `ConsoleDevice.productType` 字段 / `useDictionary('product_type')` 字典 code 实质是 `devices.product_class` 的视图层别名（历史遗留）。本期沿用前端字段名，但 i18n 文案 / 提示 / 调试日志一律使用"产品类型"或 `product_class`。后端 / 数据库 / API 统一使用 `product_class`。是否同步重命名前端字段为 `productClass` 由 D17 决策。

| 变更点 | 文件:行 | 调整 |
|--------|--------|------|
| 默认值 | `useDeviceSelection.ts:9` | `useState<string>('')` → 初值由 `useDictionary('product_type')` 第一项 value 决定；hook 内 `useEffect` 等待字典加载后 `setProductTypeFilter(firstValue)` 并触发首次 `fetchDevices` |
| 字典为空保护 | 新增 | 字典 loading / 空时：DeviceTree 显示骨架屏 + "未配置产品类型，请联系管理员"；不允许选设备、不允许执行 |
| Select 必选 | `DeviceTree.tsx:128` | 删除 `allowClear`；保留 `placeholder` 仅作占位；onChange 入参类型签名收紧不允许 `undefined`/`null` |
| 切换清空 | `useDeviceSelection.ts handleFilterChange` | 切换产品类型时除重置分页外，追加 `setSelectedDevices([])`；通过 `onProductClassChanged` 回调通知上层弹 toast |
| Toast | 新增 | `message.warning(t('mml.deviceTree.productClassSwitched'))` |
| 提交侧软校验 | `Console/index.tsx` 执行入口 | 提交前断言 `selectedDevices.every(d => d.productType === productTypeFilter)`；理论恒真，多一层保险防异步竞态 |
| 错误码消化 | `useExecuteStatements` | 接收 `ErrDevicesMixedProductClass` 400 时，弹 `message.error` + 列出混入的 `product_class` 值 |

### 7.7 UI 改动说明（线框图与状态对照）

> 线框为示意，最终视觉以设计稿为准；保留现有 3 栏 Row gutter=12 / span 6:8:10 的整体布局。

#### 7.7.1 整体页面

```
┌─ /mml/console ──────────────────────────────────────────────────────────────────────────┐
│  StepBar:  ① 选设备 ──▶ ② 选命令 ──▶ ③ 配置参数 ──▶ ④ 执行                                │
├──────────────┬───────────────────────────┬──────────────────────────────────────────────┤
│ DeviceTree   │ CommandTree               │ RightPanel                                   │
│              │                           │                                              │
│ 产品类型必选 │ ▾ 设备信息                │  ┌─ TerminalPanel ───────────────────────┐│
│ [Nova430E▾] │   ▸ LST 设备信息          │  │ 执行后回显：                            ││
│   ☑ SN001    │   ▸ MOD 设备信息          │  │  SN001  Device.DeviceInfo.UserLabel     ││
│   ☐ SN002    │ ▾ 设备版本升级            │  │    → X_VENDOR.UserName (discovered)     ││
│   ...        │   ▸ LST 设备版本升级      │  │    Result: "myDevice"                   ││
│              │ ▾ 当前告警实例            │  └─────────────────────────────────────────┘│
│ [批量 SN]    │   ▸ LST 当前告警实例      │                                              │
│              │   (全 📖 → 仅 LST)        │  ┌─ Tabs ─────────────────────────────────┐ │
│              │ ▾ 支持告警实例            │  │ 操作面板  │  参数路径指定               │ │
│              │   ▸ LST 支持告警实例      │  │ (Ctrl Pnl)│ (ParameterPath Command)     │ │
│              │   ▸ MOD 支持告警实例      │  ├────────────────────────────────────────┤ │
│              │   ▸ ADD 支持告警实例      │  │ 当前命令: LST 设备信息                  │ │
│              │   ▸ RMV 支持告警实例      │  │                                          │ │
│              │ ▾ ... (~72 object 组,    │  │ ┌─ InstanceArityInput (如适用) ─────────│ │
│              │     按 SA-SR locality 排  │  │ │ {i}: [ 1 ]                            │ │
│              │     但无章节级标题)       │  │ └──────────────────────────────────────│ │
│              │                            │  │                                          │ │
│              │ ▾ Customized (末位固定)   │  │ ┌─ ActiveSubView (按命令 op_type 单一渲染) │
│              │   ▾ Public Template       │  │ │ LST → SubFieldChecklist (RO+RW)       │ │
│              │     ▸ <cmd-shared-1>      │  │ │ MOD → SubFieldInputList (仅 RW)       │ │
│              │     ▸ <cmd-shared-2>      │  │ │ ADD → SubFieldInputList (RW 初值)     │ │
│              │     ▸ <cmd-shared-3>      │  │ │ RMV → InstancePicker (多选)            │ │
│              │     (命令完全平级)         │  │ └──────────────────────────────────────│ │
│              │   ▾ Private Template       │  │                                          │ │
│              │     ▾ alice (= 当前用户)   │  │ ┌─ ConsoleActionBar (底部) ─────────────│ │
│              │       ▸ <my-cmd-1>         │  │ │ [校验] [全选] [全不选] [执行 LST]    │ │
│              │       ▸ <my-cmd-2>         │  │ └──────────────────────────────────────│ │
│              │     (非 admin：仅自己 1 个 │  └────────────────────────────────────────┘ │
│              │      用户目录)             │                                              │
└──────────────┴───────────────────────────┴──────────────────────────────────────────────┘
```

**admin 视角下 CommandTree 的 Customized 子树**（与普通用户唯一差异处）：

```
▾ Customized (末位固定)
  ▾ Public Template
    ▸ <cmd-shared-1>
    ▸ <cmd-shared-2>
    ▸ <cmd-shared-3>
    (admin 与普通用户看到的 Public Template 完全相同)
  ▾ Private Template
    ▾ alice                  ← 用户名目录
      ▸ <cmd-1>
      ▸ <cmd-2>
    ▾ bob
      ▸ <cmd-1>
    ▾ charlie
      ▸ <cmd-1>
    ▾ Admin                  ← admin 用户自己的私有命令也作为一个目录出现
      ▸ <cmd-1>
      ▸ <cmd-2>
      ▸ <cmd-3>
    (admin 看到所有人的用户名目录)
```

#### 7.7.2 CommandTree — 节点对照（普通用户 / admin 视角对比）

```
节点示例（普通用户 alice 视角）             | 数据来源                       | 视觉提示
─────────────────────────────────────────┼───────────────────────────────┼──────────────────────
▾ 设备信息                                 | group (group_code=Device.DeviceInfo.*) | 一级 object 组（树最前）
  ▸ LST 设备信息                          | mml_commands op=LST            | 叶子，点击直接选中
  ▸ MOD 设备信息                          | mml_commands op=MOD            | 叶子
  (无 ADD/RMV：无 {i})
▾ 设备版本升级                             | group (Device.DeviceInfo.SwUpgrade.*) | 一级 object 组
  ▸ LST 设备版本升级                      | mml_commands op=LST            | 只有 LST（全 📖 RO）
▾ 当前告警实例                             | group (Device.FaultMgmt.CurrentAlarm.{i}.*) | 含 {i}
  ▸ LST 当前告警实例                      | op=LST                         |
  (无 MOD：全 RO；无 ADD/RMV：read-only event)
▾ 支持告警实例                             | group (SupportedAlarm.{i}.*)   | 含 {i} + 有 RW
  ▸ LST 支持告警实例                      | op=LST                         |
  ▸ MOD 支持告警实例                      | op=MOD                         |
  ▸ ADD 支持告警实例                      | op=ADD                         |
  ▸ RMV 支持告警实例                      | op=RMV                         |
... ~72 object 组，按 SA-SR locality 排列但不显示章节名

▾ Customized                              | 固定容器                       | 末位固定（树最后）
  ▾ Public Template                       | source=admin, visibility=public| 二级容器
    ▸ <cmd-shared-1>                      | mml_commands row               | 三级叶子（完全平级，可执行）
    ▸ <cmd-shared-2>                      | mml_commands row               | 禁止任何嵌套
    ▸ <cmd-shared-3>                      | mml_commands row               |
  ▾ Private Template                      | source=admin, visibility=private| 二级容器
    ▾ alice                               | owner_user_id=alice (=self)    | 三级用户容器（普通用户仅显示自己）
      ▸ <my-cmd-1>                        | mml_commands row (owner=alice) | 四级叶子（命令平级）
      ▸ <my-cmd-2>                        | mml_commands row (owner=alice) |

节点示例（admin 用户视角 · 仅 Private Template 子树有差异）：
  ▾ Private Template
    ▾ alice / bob / charlie / Admin       | 多个用户目录                   | admin 看到所有用户目录
      ▸ <commands per user>
```

> **树深度**：
> - 标准 object 组：2 级（object 组 → 操作叶子）
> - Customized 子树：4 级（Customized → Public/Private Template → 用户名 → 命令叶子）
> - 普通用户与 admin 树形结构完全一致，仅 Private Template 下可见的用户名目录数量不同

#### 7.7.3 操作面板 — LST 模式

```
┌─ 操作面板 (Control Panel) ────────────────────────────────────────────────────────────┐
│  当前命令: LST 当前告警实例   (Device.FaultMgmt.CurrentAlarm.{i}.*)                    │
├────────────────────────────────────────────────────────────────────────────────────────┤
│  ┌─ InstanceArityInput ─────────────────────────────────────────────────────────────┐ │
│  │ 实例索引 {i}:  [______]  ◯ 留空(全部)  ◯ 单值  ◯ 范围  例: 1,2,5  或  1-3        │ │
│  │                ↑ LST 允许留空 = partial path                                       │ │
│  └────────────────────────────────────────────────────────────────────────────────────┘ │
│  ┌─ SubFieldChecklist (op=LST 全 path 含 RO+RW) ────────────────────────────────────┐ │
│  │ [☑] AlarmIdentifier        告警序列号        Device.FaultMgmt.CurrentAlarm.{i}.…│ │
│  │ [☑] AlarmRaisedTime        发生时间          📖 R   dateTime                     │ │
│  │ [☑] FaultLocation          告警源定位        📖 R   string(512)                  │ │
│  │ [☐] ManagedObjectInstance  管理对象实例      📖 R   string(512)                  │ │
│  │ [☑] EventType              告警类型          📖 R   string(64)                   │ │
│  │ ... (RO 与 RW 同时展示)                                                            │ │
│  └────────────────────────────────────────────────────────────────────────────────────┘ │
├────────────────────────────────────────────────────────────────────────────────────────┤
│  ┌─ ConsoleActionBar (底部) ────────────────────────────────────────────────────────┐ │
│  │  已选 4 path · 4 设备           [校验] [全选] [全不选]  [ 执行 LST ▸ ]            │ │
│  └────────────────────────────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────────────────────────┘

执行后 TerminalPanel 回显（节选）：
  ─── SN001 (Nova430E, sw=3.5.1) ──────────────────────────────────────────
  [discovered] Device.FaultMgmt.CurrentAlarm.1.AlarmIdentifier
            → Device.X_VENDOR.CurrentAlarm.1.AlarmId    = "A001"
  [passthrough] Device.FaultMgmt.CurrentAlarm.1.AlarmRaisedTime
            = "2026-05-19T03:21:55Z"
```

#### 7.7.4 操作面板 — MOD 模式

```
┌─ 操作面板 (Control Panel) ────────────────────────────────────────────────────────────┐
│  当前命令: MOD ManagementServer   (Device.ManagementServer.*)                          │
├────────────────────────────────────────────────────────────────────────────────────────┤
│  ┌─ InstanceArityInput (如适用) ────────────────────────────────────────────────────┐ │
│  │ 实例索引 {i}:  [  1   ]  ⚠ MOD 必填                                                │ │
│  └────────────────────────────────────────────────────────────────────────────────────┘ │
│  ┌─ SubFieldInputList (op=MOD, 仅 RW 路径，自动过滤 📖) ───────────────────────────┐ │
│  │ [☑] URL                 [http://acs.cmcc:7547/cwmp________________]  string(256) │ │
│  │ [☑] PeriodicInformEnable[ true ▾ ]                                    boolean    │ │
│  │ [☐] PeriodicInformTime  [____________________________]  dateTime  (未勾→不下发) │ │
│  │ [☑] PeriodicInformInterval [ 300___________ ] unsignedInt[1:]                    │ │
│  │ [☐] STUNEnable          ...                                                       │ │
│  │ ... (📖 RO 字段不出现)                                                              │ │
│  └────────────────────────────────────────────────────────────────────────────────────┘ │
├────────────────────────────────────────────────────────────────────────────────────────┤
│  ┌─ ConsoleActionBar (底部) ────────────────────────────────────────────────────────┐ │
│  │  已选 3 path / 已填 3 值 · 1 设备     [校验所有值] [全选] [全不选]  [ 执行 MOD ▸ ] │ │
│  └────────────────────────────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────────────────────────┘

提交 payload (R-9.2)：
{
  "device_sns": ["SN001"],
  "statements": [{
    "command_id": "abc...",
    "operation_type": "MOD",
    "paths": ["Device.ManagementServer.URL",
              "Device.ManagementServer.PeriodicInformEnable",
              "Device.ManagementServer.PeriodicInformInterval"],
    "values": {
      "Device.ManagementServer.URL": "http://acs.cmcc:7547/cwmp",
      "Device.ManagementServer.PeriodicInformEnable": "true",
      "Device.ManagementServer.PeriodicInformInterval": "300"
    },
    "instance_selectors": {},
    "instance_indices": []
  }]
}
```

#### 7.7.5 操作面板 — ADD 模式

```
┌─ 操作面板 (Control Panel) ────────────────────────────────────────────────────────────┐
│  当前命令: ADD PLMNList   (Device.Services.FAPService.{iα}.CellConfig.LTE.EPC.PLMNList.{iβ}.*) │
├────────────────────────────────────────────────────────────────────────────────────────┤
│  ┌─ InstanceArityInput (parent path 实例号必填，新实例号由设备分配) ─────────────┐ │
│  │ Parent 实例 {iα}:  [ 1 ]    ⚠ 父级路径所有 {i} 必填                                │ │
│  └────────────────────────────────────────────────────────────────────────────────────┘ │
│  ┌─ SubFieldInputList (op=ADD, RW 字段可选填初值，留空不下发) ──────────────────────┐ │
│  │ [☑] PLMNID         [ 46000__________ ]    string(6)                                │ │
│  │ [☑] CellReservedForOperatorUse [ false ▾ ]   boolean                              │ │
│  │ [☐] IsPrimary      [_______________]      boolean   (未填→AddObject 后默认值)    │ │
│  │ ...                                                                                  │ │
│  │ 提示: 先调 AddObject 创建新实例 → 再 SetParameterValues 写入已勾选字段             │ │
│  └────────────────────────────────────────────────────────────────────────────────────┘ │
├────────────────────────────────────────────────────────────────────────────────────────┤
│  ┌─ ConsoleActionBar (底部) ────────────────────────────────────────────────────────┐ │
│  │  已选 2 初值字段 · 1 设备                       [校验] [清空]   [ 执行 ADD ▸ ]   │ │
│  └────────────────────────────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────────────────────────┘
```

#### 7.7.6 操作面板 — RMV 模式（多选）

```
┌─ 操作面板 (Control Panel) ────────────────────────────────────────────────────────────┐
│  当前命令: RMV 当前告警实例   (Device.FaultMgmt.CurrentAlarm.{i}.*)                    │
├────────────────────────────────────────────────────────────────────────────────────────┤
│  ┌─ InstancePicker (多选) ─────────────────────────────────────────────────────────┐ │
│  │ 实例来源:  ◉ 自动 LST 刷新   ◯ 手动输入索引                                       │ │
│  │ [刷新实例列表]                                                                     │ │
│  │  ┌────────────────────────────────────────────────────────────────────────────┐  │ │
│  │  │ [☐] {i}=1   AlarmIdentifier=A001    SpecificProblem="LinkDown"              │  │ │
│  │  │ [☑] {i}=3   AlarmIdentifier=A003    SpecificProblem="HighTemp"              │  │ │
│  │  │ [☑] {i}=5   AlarmIdentifier=A005    SpecificProblem="LowSignal"             │  │ │
│  │  │ [☐] {i}=6   AlarmIdentifier=A006    SpecificProblem="OOS"                    │  │ │
│  │  │ [☑] {i}=7   AlarmIdentifier=A007    SpecificProblem="ConnLost"               │  │ │
│  │  └────────────────────────────────────────────────────────────────────────────┘  │ │
│  │ 已选 3 项 → 将拆为 3 个独立 DeleteObject 任务下发                                  │ │
│  └────────────────────────────────────────────────────────────────────────────────────┘ │
├────────────────────────────────────────────────────────────────────────────────────────┤
│  ┌─ ConsoleActionBar (底部) ────────────────────────────────────────────────────────┐ │
│  │  已选 3 实例 · 1 设备                                  [取消] [ 执行 RMV ▸ ]      │ │
│  └────────────────────────────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────────────────────────┘

提交 payload (R-9.2)：instance_indices: [3, 5, 7]   instance_selectors: {}
```

#### 7.7.7 InstanceArityInput — 多层 `{iα}/{iβ}` 例

```
命令: Device.DeviceInfo.MU.{iα}.Slot.{iβ}.EU.{iγ}.RU.{iδ}.RFChannel.{iε}.*  (SR 段 5 层)

┌─ InstanceArityInput (arity=5) ─────────────────────────────────────────────────────┐
│ MU       {iα}: [ 1 ]                                                                │
│ Slot     {iβ}: [ 2 ]                                                                │
│ EU       {iγ}: [ 1 ]                                                                │
│ RU       {iδ}: [ 3 ]                                                                │
│ RFChannel{iε}: [ ____ ]  ← LST 模式允许留空（partial path）                          │
│                                                                                      │
│  渲染路径: Device.DeviceInfo.MU.1.Slot.2.EU.1.RU.3.RFChannel.                       │
└──────────────────────────────────────────────────────────────────────────────────────┘
```

#### 7.7.8 DeviceTree — 产品类型强约束（R-8）

```
┌─ DeviceTree (左栏) ────────────────────────────────────────────────────────────────────┐
│  搜索: [SN / 站点名_______________ 🔍 ]                                                │
│  ┌──────────────────────────────────────────────────────────────────────────────────┐  │
│  │ 产品类型: [ Nova430E ▾ ]   ← 必选，无 [×] 清除按钮                                │  │
│  │   字典首项默认选中（如 Nova430E）；切换时弹 toast + 清空已选设备                  │  │
│  └──────────────────────────────────────────────────────────────────────────────────┘  │
│  ┌─ 全选 (当前产品类型 50 台) ──────────────────────────────────────────────────────┐  │
│  │ [☐] 全选当前页                                                                    │  │
│  └────────────────────────────────────────────────────────────────────────────────────┘ │
│  [☑] SN-001  Site-Beijing-01     ● online    Nova430E                                  │
│  [☐] SN-002  Site-Beijing-02     ● online    Nova430E                                  │
│  [☐] SN-003  Site-Shanghai-01    ○ offline   Nova430E                                  │
│  ...                                                                                    │
│  [<] 1 / 12 [>]                                                                         │
│                                                                                         │
│  [批量录入 SN]   ← 跨产品类型 SN 会被过滤掉并提示                                       │
└─────────────────────────────────────────────────────────────────────────────────────────┘

切换产品类型 (Nova430E → Nova227) 的交互序列：
  1. Select onChange("Nova227")
  2. message.warning("产品类型已切换，已清空选中设备")
  3. setSelectedDevices([])  // 已勾选的 Nova430E 设备全清
  4. setCurrentPage(1)
  5. fetchDevices(1, searchText, "Nova227")
  6. 命令树右侧 statements 不清空（用户已配置的命令保留）
```

产品类型字典空 / loading 态：

```
┌─ DeviceTree (左栏) ────────────────────────────────────────────────────────────────────┐
│  搜索: [________________________ 🔍 ]   ← 禁用                                          │
│  产品类型: [无可选项 (禁用)]                                                            │
│  ┌──────────────────────────────────────────────────────────────────────────────────┐  │
│  │  ⚠ 未配置产品类型，请联系管理员在 系统管理 → 字典维护 → product_type 添加         │  │
│  └──────────────────────────────────────────────────────────────────────────────────┘  │
│  设备列表区域：骨架屏 + 禁用                                                            │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```

跨产品类型执行被后端拒绝时的提示（理论上前端约束已拦截，此为兜底）：

```
┌─ message.error ────────────────────────────────────────────────────────────────────────┐
│  ✗ 选中设备包含多种产品类型（Nova430E、Nova227），不允许一起执行                       │
│    请先在产品类型筛选器中限定为单一类型                                                 │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```

#### 7.7.9 不变项 / 改动项汇总

**保留不动**：
- StepBar（顶部）、BatchSnModal：原样保留
- TerminalPanel 组件壳保留（位置 / 样式不动）；内容切换为执行后回显
- DeviceTree 整体结构（左栏 6/24 栏宽 + 搜索 + 列表 + 全选）保留；仅产品类型筛选器行为按 R-8 调整
- Customized `+` 入口、AddTemplateModal 弹层组件本身不动（commit `2849f65d`）；底层数据模型扩展 visibility/owner_user_id

**重命名 / 文案改造** (R-10)：
- 主 Tab 文案：`Control Panel` → 中文"操作面板"，英文保持 `Control Panel`
- 副 Tab 文案：`Param Path Expert` → 中文"参数路径指定"，英文 `ParameterPath Command`
- 组件文件物理 rename（可选）：`Console/components/ParamPathExpert.tsx` → `ParameterPathCommand.tsx`（D14）

**删除 / 移除**：
- RightPanel 顶部不实现 OperationType segmented 切换条
- `Console/components/MmlEditor.tsx` 在 Console 不再实例化；若 ScriptTask 仍引用则文件保留，否则物理删除
- CommandTree 不再渲染 SA-SR 章节级一级分组（章节名仅在管理后台 / 报表显示）

---

## 8. 数据迁移与回滚

### 8.1 迁移顺序（dictloader 模式）

1. **P1 schema 迁移**（DDL，goose 迁移文件）：
   - 新增 `mml_param_groups.{carrier,tech,group_code,chapter_code,instance_arity,instance_levels,deprecated_at}` 列与唯一索引
   - 新增 `mml_commands.{target_paths,deprecated_at,visibility,owner_user_id}` 列与唯一索引
   - 新增 `mml_command_sub_fields.{access_type,group_id,deprecated_at}` 列
   - 新增 `mml_command_sub_field_overrides` 表
   - 为旧行回填默认值（`carrier='cmcc' / tech='lte'`；Customized 行 `visibility='private'` / `owner_user_id` 从 `creator_user_id` 回填）

2. **P1 数据回填迁移**（DML，紧接 P1 schema）：基于旧 `mml_commands` 的 `operation_type` 字段，补 `target_paths`（按 op_type 重新计算：LST=group 全 path 集；MOD=仅 RW）

3. **P2 数据加载**（不走 goose seed）：
   - 提交结构化文件 `omcgo/datamodels/mml-catalog/cmcc-tdlte-v23.json`
   - 新版 Go binary 启动时，`internal/mml/catalogloader/` 自动 UPSERT
   - 对运行中环境：`POST /api/v1/mml/catalog/reload` 触发热重载

4. **P2 后续 cleanup**：standard 范围内"本次导入未覆盖"的旧 command 由 Loader differ 自动标 `deprecated_at`（不删除），观察期之后由独立维护脚本 / API 物理删除

### 8.2 回滚

- **Schema 迁移**（P1）：每个 `.sql` 文件必带 `-- +goose Down`，回滚顺序与 Up 反向
- **数据加载**（P2）回滚有 3 种粒度：
  1. **数据文件回滚**（最快）：把 `cmcc-tdlte-v23.json` 回滚到上一版 → `POST /mml/catalog/reload` → 数据库 standard 数据回到上版状态
  2. **Loader 关闭**（紧急）：配置项 `mml.catalog.enabled = false`，启动期跳过 Loader；DB 中的 standard 数据保持上次成功 reload 的快照
  3. **回到老 catalog**（最重）：保留 `000111 / 000116 / 000126` 老 catalog 迁移作为兜底；本期不删除，回归窗口结束后再清理
- **Loader 本身的 bug**：因 Load 在事务内执行，单次 reload 失败自动回滚 DB；不会出现"加载到一半"的中间态

### 8.3 sub_field 用户配置保护

- `default_selected`、`label_i18n` 中用户自定义的 override 用单独表 `mml_command_sub_field_overrides` 保存
- 表 schema：`(sub_field_id, owner_user_id) UNIQUE`，列 `default_selected_override boolean`、`label_i18n_override jsonb`
- Owner 维度：**per-user**（每用户独立 override，与 R-5 私有模板一致）
- 运行期读取：`mml_command_sub_fields LEFT JOIN mml_command_sub_field_overrides ON (sub_field_id, owner_user_id=current_user_id)`，用 `COALESCE(override.default_selected_override, base.default_selected)` 取最终值

---

## 9. 测试策略

| 层 | 用例 | 工具 |
|---|------|------|
| 单元 - 解析脚本 (dev) | 18 章节覆盖；多层 `{iα}/{iβ}` 折叠；伪命令 `.*` 过滤；权限矩阵 → 1-4 行 per-op commands 展开；输出 JSON 与 §5.2 schema 一致 | Python pytest（脚本侧，**不进生产 CI gate**） |
| 单元 - 后端 Loader | JSON 解析、UPSERT 幂等、差集软删边界（仅 standard）、sub_field override 不被覆盖、specVersion 多版本路由 | Go `_test.go` |
| 单元 - 后端 service | `validatePath` LST partial path 放行；`expandInstancePaths` 多层实例号；ExecuteStatements 拒绝不一致 op-type；Translator 调用与 passthrough | Go `_test.go` |
| 单元 - 前端 | 切换命令丢弃上一条状态（弹 confirm）；MOD 隐藏 RO；InstanceArityInput 范围解析；DeviceTree 切换 product_class 清空已选 | Vitest + Testing Library |
| 集成 - Loader | 干净库 / 现网库快照两种基线上 `Load()` 二次执行结果一致（行数 / 列值 hash 比对）；admin 行 / Customized 行不被触碰；override 表保护 default_selected | Go integration test + PG fixture |
| 集成 - 数据完整性 | CI lint：`sourceDocSha256` 与 MD 文件实际 sha256 一致 | CI 任意语言脚本 |
| E2E | 18 章节抽样：LST 全勾 / LST 部分勾 / MOD 改 RW / ADD 新实例 / RMV 多选 | `scripts/e2e_verify.sh` 扩展 |
| E2E - RBAC | alice 创建 1 个 private 命令，bob 登录看不到；admin 登录看到 alice/bob 两个用户目录 | E2E |
| 回归 | Customized PrivateTemplate/PublicTemplate `+` 入口、私有命令保存与执行 | E2E 既有用例 |
| 多皮肤 | `webcode/`、`webcode-v2/`、`webcode-v3/` 全部 `npm run typecheck` 通过 | CI |

**DoD 阈值**：72 个 object 组至少 18 组（每章节抽样 1 组）有 E2E 用例；现有 226+ E2E 断言不退化。

---

## 10. 分阶段实施

### P1.a — Schema + Catalog Loader 骨架（1 周）

- DDL：新增列与索引（`carrier`、`tech`、`group_code`（object 路径）、`chapter_code`、`instance_arity`、`instance_levels`、`target_paths`、`visibility`、`owner_user_id`、`deprecated_at`、`access_type`、`group_id`）
- 新表 `mml_command_sub_field_overrides`
- 新建 Go 包 `internal/mml/catalogloader/`（实现 dictloader.Loader 接口骨架）
- 提交首版示例 JSON 数据文件
- 出口：迁移可上 / dry-run reload 不报错；`go build ./... && go test ./...` 通过

### P1.b — R-8 产品类型强约束（与 P1.a 并行 · 0.5 周）

- 后端 `ErrDevicesMixedProductClass` / `ErrProductClassUnresolved` 错误码 + ExecuteStatements 入口按 `devices.product_class` 分组校验
- 前端 DeviceTree 默认值/必选/切换清空交互
- i18n 文案上线
- 出口：混类型 400 拦截；DeviceTree 默认第一项 + 切换清空交互；E2E 用例通过

### P1.c — fanout Translator 注入（1 周）

- 改 `MMLTaskCreator.CreateAndFanoutTask`：per device 调 ProductRegistry + Translator
- Translator factory cache by (productId, swVersion)
- 写 `device_task.params` 用 privatePath；同时记录 `source: discovered/default/passthrough` 到 task_log
- 出口：单元测试覆盖三态翻译；现网 standard 数据被回填 `target_paths`（按现有 op_type 重新计算）

### P2.a — CommandTree 重构（R-1/R-2/R-5）（2 周）

- 解析脚本 + Catalog Loader 完整 UPSERT 73 group × per-op 命令
- 前端 CommandTree 完全重画（object 一级 + per-op 叶子 + Customized 末位 + RBAC 子树）
- BuildTree API 加 owner 维度
- 出口：18 章节导入完成；object 一级展示；Customized 末位 + RBAC 子树正确

### P2.b — Control Panel 改造（R-3/R-9.1/R-10）（1.5 周）

- 不实现 OperationType segmented 切换条
- 删 MmlEditor 实例化
- 加 ConsoleActionBar
- Tab 重命名 + i18n
- TerminalPanel 内容切换为执行回显（含 source 标签上色）
- 出口：操作切换在所有命令上可用；TerminalPanel 显示 standardPath ↔ privatePath

### P2.c — 结构化 API（R-9.2）（与 P2.b 并行 · 0.5 周）

- 后端 `ExecuteStatementsRequest` 结构化字段
- 旧 render/parse 接口标 deprecated
- 出口：标准命令全部走结构化 API；既有 Customized / 老 catalog 行为不退化

### P3 — RMV 多选 / ADD 复合 RPC / 收尾（1 周）

- RMV 多实例批量任务
- ADD 内嵌 SetParameterValues（依赖 task service 复合 RPC 能力）
- 副 Tab 组件物理 rename（D14）
- D26 ADD/RMV 启发式 overlay 第一批
- 老 catalog 行物理删除（独立 Sprint）

---

## 11. 风险与待澄清

| # | 风险 / 问题 | 影响 | 验证手段 |
|---|------------|------|---------|
| R1 | `mml_param_groups` 是否已有 `carrier`/`tech` 列？若无 → P1.a 必须先加 | 阻塞 standard 行隔离 | 看现网 schema |
| R2 | 现网是否已有用户手动调整过的 `mml_command_sub_fields.default_selected`？ | 影响 override 表回填策略 | 运维 / DBA dry-run |
| R3 | ACS `DeleteObject` 是否支持批量删除多实例？ | 影响 RMV 多选实现成本 | F01 ACS owner |
| R4 | ACS `AddObject` 与 `SetParameterValues` 是否能在同一会话内串行？ | 影响 ADD 用户体验（一次提交 vs 两次任务） | F01 ACS owner |
| R5 | 老 catalog（000111+000116）与本规范的 command 覆盖差有多大？是否会产生大量 deprecated 残留？ | 影响 P3 清理工作量 | 解析脚本 dry-run 报告 |
| R6 | `instance_arity` 在多层 `{iα..iε}` 命令（如 SR.MU/Slot/EU/RU/RFChannel 5 层）的 UI 表达 | 影响 InstanceArityInput 复杂度 | 设计稿确认 |
| R7 | SQ 段的伪命令 `#### 命令: .*` 是规范文档的注释，解析器需明确过滤；其他段是否存在类似伪命令？ | 影响导入完整性 | 解析脚本 dry-run 报告 |
| R8 | 显示名"中文别名"的推断规则在哪些命令上会失败（路径末段对象不在表格内）？ | 影响 UI 可读性 | 解析脚本 dry-run 报告 |
| R9 | 多皮肤 `webcode-v2/` `webcode-v3/` 是否会强约束 RightPanel 形态？ | 影响前端拆分粒度 | 前端 owner |
| R10 | 产品类型字典 (`useDictionary('product_type')`) 是否覆盖现网所有设备的 `devices.product_class`？是否存在 `product_class IS NULL` 或 `''` 的脏数据？是否存在字典中无对应项的 `product_class`（孤儿）？ | 字典外 product_class 设备无法选中；孤儿设备触发 `ErrProductClassUnresolved` | 运维 / DBA dry-run：`SELECT product_class, COUNT(*) FROM devices GROUP BY product_class` |
| R11 | 现网用户是否依赖"跨 `product_class` 批量 LST"的现存能力（典型场景：批量查同一只读路径如 SoftwareVersion）？ | 影响是否给 LST 开例外 | 产品 / 运营商联系人 |
| R12 | BatchSnModal 粘贴跨 `product_class` SN 时的处理策略 | 影响 BatchSnModal 交互 | D13 |
| R13 | ParamModel `Translator.ToPrivate` 当前是否已经按"按 productId 入参"落地？若仍按 paramModelId 入参，R-9.3 需要先做一道 productId → paramModelId 适配 | 阻塞 R-9.3 P2 落地 | parammodel owner |
| R14 | `dev.software_version` 字段在现有 `devices` 表是否常驻？若由 Inform 间歇刷新，旧值可能与 discovered 表 swVersion 不匹配 → 翻译时降级到 default | 影响 discovered 命中率与可观测性 | F06 device owner |
| R15 | passthrough 路径数（未命中 discovered/default）是否需要 Prometheus 指标暴露？ | 影响 ParamModel 字典完整度可观测 | 运维 owner |
| R16 | TerminalPanel 回显结构 `executed_paths_per_device` 在多设备 × 多 statement 大批量时（如 100 设备 × 5 statement × 20 path = 10K 行）的渲染性能 | 影响大批量场景用户体验 | 前端 owner |
| R17 | 旧 `POST /mml/console/render` / `POST /mml/console/parse` 接口在脚本任务侧 (`ScriptTask`) 是否会继续使用？过渡期 1 release 的废弃节奏是否合理？ | 影响 D7 决策 | mml 模块 owner |
| R18 | 前端 `productType` 与后端 `product_class` 命名鸿沟：当前 `deviceApi.getList` 返回的 `productType` 究竟是 Axios 拦截器把 `product_class` 自动 camelCase 后变成 `productClass` 然后被业务层手动 alias 成 `productType`，还是后端 API 显式输出 `product_type`？ | 影响 D17 范围 | 前端 owner 一次性看完 |
| R19 | Loader 启动期幂等性：73 命令 × 平均 N 个 sub_fields 的 UPSERT 在大单事务下耗时多少？是否影响 app cold start P99？ | 影响是否要拆分批 / 异步加载 | 后端 owner |
| R20 | Loader 的"差集软删"在多人并行操作场景下的语义：管理员 A 通过 reload 把 command X 标 deprecated，管理员 B 同时通过 admin API 把另一个 admin-source command Y 改了 → 是否会互相影响？标软删边界是否严格限定 source='standard'？ | 影响并发安全与运维流程 | 后端 owner |
| R21 | sub_fields 从 command 维度迁到 group 维度的数据迁移路径：现网若已有 `mml_command_sub_fields.command_id` 数据，如何把它们重新关联到 group？是合并去重还是仅迁第一条 LST 的 sub_fields？ | 影响 P2 数据迁移复杂度 | 后端 owner 看现网数据 |
| R22 | Customized R-5.2 私有模板对应的 `owner_user_id` 现网是否已存在（可能现有字段叫 `creator_user_id` / `created_by`）？回填策略？ | R-5.2 落地难度 | 后端 owner |
| R23 | admin 角色判断口径：用 `admin/role.go` 现有 `IsAdmin` / `IsSuperAdmin` 函数？还是单独定义"可见所有人私有模板"的细粒度 RBAC 权限码？ | 影响 §6.5 BuildTree 鉴权代码 | admin 模块 owner |
| R24 | 73 个 object 组在树上没有章节分组时是否影响导航效率？是否需要前端搜索框 + 中文别名模糊匹配？ | 影响 UX；R-1 后用户在 72 平级节点里找命令的成本 | 产品 / UX |
| R25 | spec 中"是否生成 ADD/RMV"的启发式（含 {i} + 至少一条 RW path）不够精确；某些事件表如 SupportedAlarm 可能不应允许用户 ADD/RMV（设备维护，非配置）。导入 JSON 后是否需要人工审 / overlay 来排除？ | 影响 ADD/RMV 命令的合法性，过度生成会让用户看到无效操作 | 产品 + 后端联合 review |

### 11.1 端到端 13 项边界条件

详见 §14.3。最关键 3 项：
- **G1** Translator 注入位置（D10 已签 ✅ 选 service 层 fanout 处）
- **G4** ACS 响应事件 payload 是否含 `task_id`：决定 completion_router 能否回溯到 mml_task
- **G12** `AddObject` / `DeleteObject` RPC 在 `acs/rpc/` 的具体实现存在性

---

## 12. 决策清单

| # | 决策点 | 默认建议 |
|---|--------|---------|
| D1 | 一级分组显示名直接用 "说明"列中文，不再用业务大类名 | ✅ 采纳 |
| D2 | 多层 `{iα}/{iβ}` 折叠为 `{i}` + `instance_arity` 数字 | ✅ |
| D3 | 老 catalog（standard，无 group_code 的旧行）暂保留，仅打 deprecated | ✅ 风险最低 |
| D4 | 解析脚本归属位置：`omcgo/migrations/scripts/parse_cmcc_tdlte_v23.py`（与 `import_old_mml_catalog.py` 同级） | ✅ |
| D5 | 是否同步引入版本字段（`mml_commands.spec_version`），为后续 v2.4 留扩展点？ | ⏳ 建议加，开销极低 |
| D6 | R-8 产品类型字典空时是否硬性禁用 DeviceTree | ✅ 硬性禁用 |
| D7 | BatchSnModal 跨 `product_class` SN 处理：① 过滤+提示 ② 阻塞重选 ③ 自动切换为多数派 | ✅ ①过滤+提示（最小破坏，与"切换 product_class 清空已选"语义一致） |
| D8 | R-8.4 后端校验位置：service 入口（建议）vs handler 层 | ✅ service 入口（便于 unit test，也兼顾未来 gRPC / 北向接口复用） |
| D9 | LST 是否给 R-8.4 开例外（允许跨产品类型 LST 同一只读路径）？ | ❌ 不开例外。LST 看似无副作用但路径权限 / 存在性仍可能因 ParamModel 差异而失败；若现网有此需求另立项做"跨产品类型只读批量查询"专用接口 |
| D10 | 是否需要给 admin 一个"查看请求 payload"调试入口（在 ConsoleActionBar 旁加 `[查看请求]` 按钮，弹窗显示将提交的 JSON）？ | ⏳ 可选；P2 建议先不做，等用户反馈再补 |
| D11 | Customized 命令的执行通道：① 也走 R-9.2 结构化 API（推荐）② 保留独立 MML 文本通道 | ✅ ① 走结构化 API；前提是 AddTemplateModal 强制要求绑定 commandCode + sub_fields（已具备） |
| D12 | 旧 `POST /mml/console/render` / `parse` 接口的废弃节奏：① P2 结束即废弃 ② 保留 1 release 给 ScriptTask 平滑迁移 ③ 永久保留 | ✅ ②（1 release 后清理）；ScriptTask 是独立模块，避免连带改造 |
| D13 | R-9.3 全部 path passthrough 时是否要 warning：① 仅 task_log 记录 ② TerminalPanel 高亮 ③ message.warning 弹窗 | ✅ ②（高亮，不弹窗，避免大批量执行时打扰） |
| D14 | TerminalPanel 回显的 standardPath ↔ privatePath 是否对运维隐藏（仅 source 标签 + privatePath）？ | ❌ 同时显示双 path，便于排查 |
| D15 | 大批量 (≥ 100 设备 × ≥ 10 path) 的 R-9.3 翻译开销：① 同步翻译 ② 分批并发翻译 ③ 后端流式响应 | ⏳ P2 先同步翻译（Redis L2 hit 应足够）；若实测压力大再上 ② |
| D16 | 结构化数据文件格式：① **JSON** ② **XML** | ✅ **JSON**。理由：i18n map / 嵌套 commands + sub_fields 在 JSON 更紧凑；与 ParamModel 不需对账；Go `encoding/json` 零依赖 |
| D17 | Python 脚本 `parse_cmcc_tdlte_v23.py` 是否提交到仓库？ | ✅ **提交**到 `omcgo/migrations/scripts/parse_cmcc_tdlte_v23.py`。理由：审计可见 / 多人复用 / 可在 CI 跑等价性校验；明确路径不进 production 构建 |
| D18 | spec 升级到 v2.4 时的多版本策略：① 覆盖式 ② 并存式（`specVersion` 字段隔离） | ⏳ 建议 ②并存式，但 P3 再做。MVP 期单一 active 版本（v2.3） |
| D19 | 启动期 Loader 失败后的行为：① fail-fast（app 起不来） ② degrade（app 起来但 mml console 标 unavailable） | ✅ ①fail-fast，与 parammodel Loader 一致。理由：command catalog 是 mml console 主流程的必需数据 |
| D20 | `mml.catalog.enabled = false` 紧急关闭开关是否在 P2 就预留？ | ✅ 预留。Viper 配置项，零成本；运维紧急回滚兜底 |
| D21 | Customized Public Template 命令的**删除权限**：① 仅 admin 可删；② admin + 创建者可删；③ 任何用户可删 | ✅ ②admin + 创建者（与典型 share-by-link 模式一致；admin 兜底） |
| D22 | 切换树叶子时如有未保存输入，是否弹 confirm？ | ✅ 弹（防止误丢失）；本地草稿不持久化 |
| D23 | 副 Tab 组件文件物理 rename：`ParamPathExpert.tsx` → `ParameterPathCommand.tsx` | ⏳ 建议 P2 一并 rename（一次性改动，diff 清晰）；如担忧 git history 难追，可保留旧文件名仅改 UI 文案 |
| D24 | sub_fields 迁移到 group 维度的 schema 路径：① 新增 `group_id` + 保留 `command_id` 双 FK 兼容期；② 直接改 FK；③ 新建 `mml_group_sub_fields` 表 | ⏳ 建议 ①（兼容期 1 release，新 BuildTree 读 group_id，旧 service 仍读 command_id；P3 物理删 command_id） |
| D25 | R-1 后 73 个 object 组无章节归类，是否需要顶部加搜索框 + 中文别名模糊匹配？ | ✅ 建议加（轻量 antd Input，前端本地过滤；不需后端 API） |
| D26 | R-2 ADD/RMV 启发式生成（含 {i} + 至少一条 RW）可能过度（如 SupportedAlarm.{i}.* 📖📝 是否真允许用户添加新告警模板？）：① 全自动生成，由 admin 后台 overlay 隐藏；② JSON 中手写 `disableAdd: true` 字段；③ 默认仅 LST，要 ADD/RMV 显式声明 | ⏳ 建议 ②（保留自动+手动 overlay 兼顾）；P2 先 ① 全自动；运营反馈后再 overlay |
| D27 | Translator 注入位置：① BuildStatementCommands 时一次性翻译 ❌ ② `MMLTaskCreator.CreateAndFanoutTask` fanout per-device 时翻译 ③ ACS 取队列后翻译 | ✅ ② 在 service 层 fanout 处翻译，写 privatePath 入 device_task.params；ACS 直接消费已翻译路径。理由：每设备 swVersion 不同 → discovered 不同；ACS 不应承担字典依赖 |
| D28 | RMV 模式 InstancePicker 实例号数据来源：① 自动 LST 拉取一次 ② 仅手动输入索引 ③ 缓存 + 手动兜底 | ⏳ 建议 ①+③（自动 LST 默认 + 手动输入兜底） |
| D29 | `mml_command_sub_field_overrides` 表的 owner 维度：① 全局共享 ② per-user | ✅ ②per-user（owner_user_id 维度，与 R-5 私有模板一致）；理由：避免用户互相覆盖配置 |
| D30 | 前端 `productType` 字段（`ConsoleDevice.productType`、`useDictionary('product_type')`、`productTypeFilter` 等）是否在本期同步重命名为 `productClass` 以对齐后端 `devices.product_class`？ | ⏳ 建议 P3 单独立项（影响 frontend-core + 3 个皮肤，跨包改动风险）。本期保留历史命名，仅以注释 + 文档明确"前端 productType ≡ 后端 product_class"；P3 做集中重命名 PR |

---

## 13. 命令清单全表（Appendix A · 派生于 spec 的人类审阅视图）

> 受文档长度限制，完整 72 个 object 组、~152 行 per-op 命令、~625 条 TR-181 路径的明细单独存放：
>
> **清单文件**：[`docs/design/mml-console-cmcc-tdlte-v23-catalog-listing.md`](./mml-console-cmcc-tdlte-v23-catalog-listing.md)

清单文件包含：
- **§1 主索引**：72 个 group 的 1 行摘要表（chapter / group_code / perm / arity / LST / MOD / ADD / RMV 数量）
- **§2 路径明细**：每个 group 的 TR-181 leaf 参数清单（含权限标记与类型）
- **§3 统计校验**：与 spec metadata（73 命令 / 625 参数）逐章节对账
- **§4 文件对照**：spec MD / JSON 数据文件 / 本调整方案 / 清单 4 份文档关系

### 13.1 派生与同步契约

```
权威源（人类编辑）
  cmcc-tdlte-southbound-data-model-v2.3.md
        │
        │ 离线 Python（dev-only，不进生产）
        ▼
  cmcc-tdlte-v23.json           ← 运行时数据源（Catalog Loader 消费）
        │
        │ 离线工具同步派生
        ▼
  catalog-listing.md            ← 本附录（人类审阅视图）
```

三个文件由 `sourceDocSha256` 在 CI 兜底校验"MD 改了但 JSON / 清单未同步"。

### 13.2 审阅关注点

| 关注点 | 决策 |
|--------|------|
| 73 个 `####` 块是否都被拆分正确？是否有遗漏或重复？ | 见清单 §3 统计校验表 |
| SQ 段的伪命令 `.*` 是否正确剔除？ | ✅ 已剔除（72 = 73 - 1） |
| ADD/RMV 自动生成的 `✓⚠` 标记项（共 ~18 group） | D26 — 哪些应进入 overlay 黑名单？典型可疑项：G-17/G-25 (FAPService 载波 i=1~3 固定)、G-64~G-68 (硬件描述 MU/Slot/EU/RU) |
| 全 RO 的多实例对象（如 G-06 CurrentAlarm.{i}.* 全只读 → 只 LST）的判定逻辑是否符合预期 | 清单 G-06~G-09 / G-69~G-72 |
| 多层 `{i}` 实例（arity ≥ 2）的层级语义名是否准确（用于 InstanceArityInput UI 渲染） | 清单 arity 列 |
| 中文别名命名一致性（同一 object 在树上、TerminalPanel、操作面板的显示文案是否一致） | 清单 G-XX 标题 |

---

## 14. 端到端流程可行性评估

> **结论先行**：完整的 17 步执行流程**整体可落地**，~75% 基础设施已存在（mml_tasks 表 / device_tasks.source='mml' 关联 / Translator / ProductRegistry / result_aggregator / completion_router / TerminalPanel SSE 通道均已实现）。本方案叠加的新工作集中在 ① catalog 重构（R-1/R-2/R-5）② DeviceTree 强约束（R-8）③ 结构化 API + Translator 注入 fanout（R-9）④ UI 文案 / Tab 改名（R-10）。**13 个边界条件需要在 P1 起步前澄清**（详见 §14.3）。

### 14.1 端到端流程逐步对照（17 步流程 vs 现有代码 vs 本期工作量）

| # | 流程步骤 | 现有代码定位 | 本期改动 | 工作量评估 |
|---|---------|------------|---------|-----------|
| 1 | 选择产品类型 | `useDeviceSelection.ts:9` `productTypeFilter`、`DeviceTree.tsx:128` Select | R-8 默认值 / 必选 / 切换清空 / 后端校验 `ErrDevicesMixedProductClass` | **S** 前端小改 + 1 个错误码 |
| 2 | 设备列表选择设备 | DeviceTree 现有 SN 多选 | R-8 联动 | **S** 已有 |
| 3 | 选择命令 | `CommandTree.tsx` 现读 `useGroupTree` 树；`mml_param_groups` / `mml_commands` 已建 | R-1/R-2 重构：取消 SA-SR 分组；per-op 命令展开；R-5 Customized RBAC | **M** schema + Loader + 前端树重构 |
| 4 | 控制面板确认 path | `SubFieldChecklist` / `SubFieldInputList` / `InstancePicker` 已建 | R-4 按命令固定 op_type 单一渲染；InstancePicker 升级多选 | **M** 前端 |
| 5 | 点击执行 | `RightPanel.tsx:50` 含 `MmlEditor` 文本框 + 执行按钮 | R-9.1 删除 MmlEditor；底部加 `ConsoleActionBar`；R-9.2 直传结构化 payload | **S** 前端 |
| 6 | 添加 MML 任务记录 | 已存在 `mml_tasks` 表（migrations/000120 补齐 15 列）；`MMLTask` 结构体 含 TaskName / Status / TotalDevices / SuccessCount / FailedCount / Result / ExecuteType | 沿用现有，仅在 ExecuteStatements 改为结构化入参 | **S** 已有 |
| 7 | 按 SN→产品类型转化 path 映射 | 已存在 `Translator.ToPrivate(standardPath)` (`internal/config/parammodel/translator.go:91`) O(1) map + 自动 passthrough；`ProductRegistry.MatchProductClass(productClass)` 返回 Product (含 product_id) (`internal/product/registry.go:158`) | **R-9.3 新增 fanout 阶段注入翻译**：当前 `console_executor.go:68 ExecuteStatements` → `BuildStatementCommands` 不做 per-device 翻译；需要在 `MMLTaskCreator.CreateAndFanoutTask` 实现内、为每个 device 调链路 `dev.product_class → ProductRegistry → product_id → Translator.ToPrivate(standardPath)` | **M** 改 service 层 fanout 接入点 |
| 8 | 转换成功后投入队列 | 已存在 `internal/task/` 统一队列（Redis Sorted Set + PG `device_tasks`）；source='mml' / source_id=mml_task_id 已建立关联 (`internal/task/model.go:60`) | 沿用 | **S** 已有 |
| 9 | ACS 消费 RPC 任务 | 已存在 ACS session 状态机 + admission controller；连接请求 (`internal/acs/connreq/`) | 沿用，但要确认 G3（device.software_version 新鲜度）+ G11（连接请求触发时机） | **S** 已有 |
| 10 | 转化成 RPC 报文 | 已存在 `internal/acs/rpc/` dispatcher + command；4 种 RPC（GPV / SPV / AddObject / DeleteObject）由 dispatcher 路由 | 沿用 | **S** 已有，需验证 AddObject + 串行 SPV 能力（G5） |
| 11 | 下发给基站 | TR-069 协议层，已落地 | — | — |
| 12 | 基站响应 | TR-069 协议层 | — | — |
| 13 | ACS 响应入队 | 已存在 `internal/acs/handler.go` 持 eventBus；响应处理后向 `command.{rpc}.response` subject 发布 | 验证 payload 是否含 `task_id`（用于 mml_task 回溯） | **S** 已有，需端到端验证 |
| 14 | APP/WORKER 消费处理 | 已存在 `internal/mml/result_aggregator.go` + `internal/task/completion_router.go`：completion_router 注册 mml source handler → ResultAggregator.OnTaskCompleted → `IncrementStats(mmlID, successDelta, failedDelta)` 原子更新 | 沿用 | **S** 已有 |
| 15 | 回调通知 MML 任务结果 | 已存在 ResultAggregator.IncrementStats / publishDeviceFrame；executor-based 路由 SSE | 沿用 | **S** 已有 |
| 16 | 更新 MML 任务状态 / 结果 | 已存在 `mml_tasks.status / success_count / failed_count / result` 字段；状态机 pending → running → completed/failed/partial | 沿用，仅需补 per-device 结果中的 `source: discovered/default/passthrough` 标签存储（R-9.3） | **S** + 加 1 个 JSONB 字段 |
| 17 | TerminalPanel 实时回显 | 已存在 SSE 通道 `mml_device_frame`（ResultAggregator.publishDeviceFrame 推送，前端订阅）；`TerminalPanel.tsx:28` 接收 lines prop | R-9.1 内容切换为"server 回显 standardPath ↔ privatePath + RPC 响应摘要"；executor 字段需含当前用户 ID 做 SSE 过滤 | **S-M** 数据格式调整 + SSE 推送过滤 |

> **图例**：S = 0.5-1.5 人日 · M = 2-5 人日 · L = 1+ 周
> 总估算（仅本期增量）：5 × M + 12 × S ≈ **3-4 周** 工作量（1 个 backend + 1 个 frontend 全职）

### 14.2 关键集成点已就绪（不需要新建）

| 组件 | 文件 | 现状 |
|------|------|------|
| MML 任务持久化 | `internal/mml/model.go:147` `MMLTask` + migration `000120_mml_tasks_restore_execution_columns.sql` | ✅ 表 + struct 齐全 |
| 设备任务关联 | `internal/task/model.go:60` `source='mml' / source_id=mml_task_id / command_index / device_index` | ✅ 关联已建 |
| Translator | `internal/config/parammodel/translator.go:91` `ToPrivate(standard)` / `ToStandard(private)` + passthrough | ✅ O(1) map + fallback 已实现 |
| ProductRegistry | `internal/product/registry.go:158` `MatchProductClass()` 正则路由返回 Product 含 product_id | ✅ 已落地，孤儿设备走 `ErrOrphan` |
| ACS RPC dispatcher | `internal/acs/rpc/command.go` + `dispatcher.go` | ✅ 4 RPC 已实现 |
| 任务完成路由 | `internal/task/completion_router.go:22-112` 按 source 路由到对应 handler | ✅ 已就绪 |
| 结果聚合 | `internal/mml/result_aggregator.go:14-75` `OnTaskCompleted → IncrementStats` | ✅ 已就绪 |
| MML 任务列表前端 | `omcmb/webcode/src/pages/mml/TaskRecord/index.tsx:72` + `useMMLTaskResults` hook | ✅ 列表 + 详情模态已就绪 |

### 14.3 边界条件 / 关键澄清点（**P1 启动前必须澄清**，共 13 项）

| # | 关键点 | 风险 | 验证手段 / 决策点 |
|---|--------|------|-----------------|
| **G1** | `BuildStatementCommands` 与 Translator 集成位置 — 当前在 `console_executor.go:80` 一次性编译所有 statements 不做 per-device 翻译；本期要求在 fanout per-device 时调 Translator → 需在 `MMLTaskCreator.CreateAndFanoutTask`（service.go 实现）拦截改造 | 改造点未到位则 path 不翻译 / device_task params 仍是 standardPath | 看 service.go MMLTaskCreator 完整实现，定义 Translator factory 注入位置（D27） |
| **G2** | Translator factory 缓存 — 当前 `Translator` 实例预加载 per (productId, swVersion)，100 设备 × 不同 swVersion 会创建 100 个 Translator → 需要 Registry 层 cache | 大批量执行内存抖动 / 启动期重复加载 | 看 `internal/config/parammodel/registry.go` 是否提供 `GetTranslator(productID, swVersion)` 缓存方法；若没有，本期需补 |
| **G3** | `device_info.product_class / software_version / product_id` 字段实存 | R-9.3 翻译链多一次 lookup；R-15 risk | 看 `internal/device/inform_handler.go` 字段写入；DBA dry-run `SELECT product_class, software_version, product_id FROM device_info LIMIT 10` |
| **G4** | ACS 响应事件的 subject 与 payload 含 task_id — 现有 handler.go 引用 eventBus.Publish 但 subject 模式 / payload 结构需端到端验证（task_id 是否随 RPC 响应一路回传） | 不含 task_id 则 completion_router 找不到归属 → mml_task 状态不更新 | 看 `internal/acs/handler.go` 响应处理段；接 `internal/acs/rpc/dispatcher.go` 发起时 task_id 注入逻辑 |
| **G5** | ADD + SetParameterValues 原子性 — ADD 操作语义上需要 AddObject 后立即对新实例 SPV；当前 acs/rpc 是否支持同会话内串行？还是要拆 2 个独立 task | 影响 ADD UX 与失败回滚（已 AddObject 但 SPV 失败 → 空实例残留） | 看 `internal/mml/sequencer.go` 是否有 ADD 复合 RPC；与 ACS owner 确认会话内串行 |
| **G6** | RMV 多实例的实例号偏移问题 — TR-069 中 `DeleteObject(Device.X.{i=3})` 后剩余实例编号是否重排？连续删 {3} {5} {7} 是否需 sort desc 顺序避免编号错位？ | 错位会导致删错实例 | 协议 review + ACS RPC owner 确认；如需 sort 由 sequencer 处理 |
| **G7** | RMV 模式 InstancePicker 实例号数据来源 — 自动 LST / 缓存 / 手动输入 | 影响 UX 与 LST 配额 | D28 已签：建议 ①+③（自动 LST + 手动兜底） |
| **G8** | Customized 命令的 path 是否走 Translator — 用户自定义命令的 paths 可能是 standardPath / 也可能是 privatePath 裸写（高级用法）；如何识别？ | 错翻译 / 漏翻译 | D11 已记；建议 `customized_command.skip_translation: bool` 字段 |
| **G9** | `mml_command_sub_field_overrides` 表的 owner 维度 — D29 已签 per-user | 用户配置覆盖语义清晰 | D29 已签 ✅ per-user |
| **G10** | TerminalPanel SSE 多租户隔离 / 鉴权 — `mml_device_frame` SSE 当前是 executor-based 路由，但跨用户隔离机制需验证 | 隐私 / 安全 | 看 SSE handler 鉴权逻辑；端到端 E2E 测试 |
| **G11** | 设备长期 offline 时的 MML 任务超时 — 任务入队但设备永不上线，何时 timeout？task TTL? | 任务表堆积 / 用户得不到反馈 | 看 `internal/task/scheduler.go` 是否有 task TTL；若无，补 timeout 策略（默认建议 5 min for LST/MOD，15 min for Download/Upload） |
| **G12** | AddObject/DeleteObject RPC 在 acs/rpc 的具体实现存在性 — 仅确认 dispatcher + command 抽象，需具体 RPC 方法文件 grep | 缺一不可，缺则 ADD/RMV 失败 | grep `AddObject` `DeleteObject` in `internal/acs/rpc/`；缺则本期补 |
| **G13** | R-9.3 全 passthrough 时是否真能成功执行 — Translator 未命中 → 透传 standardPath → ACS 下发后 CPE 可能回 9005 (Invalid parameter name) | 用户体验：执行回显报错难懂 | TerminalPanel 显示 passthrough 标记时主动提示"path 未配置映射，预期可能失败" |

### 14.4 端到端验证套件

P1.c 完成后即可跑端到端验证：

1. **冒烟流程**（手测）：选 1 个产品类型 → 选 1 个设备 → 选 `LST 设备信息` → 点执行 → TerminalPanel 见 17 path 回显 + discovered/passthrough 标签
2. **混类型拒绝**（自动 E2E）：选 2 个不同 product_class 设备 → 执行 → 400 `ErrDevicesMixedProductClass`
3. **MOD 翻译全链**：选 `MOD ManagementServer`，改 URL → device_task.params 用 privatePath → ACS 下发 → CPE 响应 → TerminalPanel 见结果
4. **ADD 流程**：选 `ADD PLMNList` → 父级实例 1 → AddObject + SPV → 新实例号回写 → result_aggregator 标 success
5. **RMV 多选**：选 `RMV 当前告警实例` → 多选 3 个 → 拆 3 个 DeleteObject task → 结果聚合
6. **Customized RBAC**：alice 创建 1 个 private 命令，bob 登录看不到；admin 登录看到 alice/bob 两个用户目录
7. **离线设备超时**（G11）：关 1 个设备 → 执行 → 超时（5 min）→ device_task=failed → mml_task.failed_count++

### 14.5 落地可行性总结

| 维度 | 评估 |
|------|------|
| 整体可落地性 | ✅ **可落地**（核心基础设施 ~75% 已就绪） |
| 总工作量 | ~5-6 周（1 后端 + 1 前端全职），与 §10 P1+P2+P3 累计一致 |
| 关键卡点 | G1（Translator 注入位置）/ G4（ACS 响应 task_id 链路）/ G12（AddObject/DeleteObject RPC 实现存在性）— 这 3 项要在 P1.c 之前 grep 验证 |
| 新增表 / 新增 schema | 仅 `mml_command_sub_field_overrides` 新表 + 7-8 个列；其余表 / 索引复用 |
| 新增 Go 包 | `internal/mml/catalogloader/`（与 4 个现有 dictloader 同级） |
| 新增前端组件 | `ConsoleActionBar`、`InstanceArityInput`（其余复用） |
| 风险等级 | 中（13 个边界条件已识别，决策清晰） |

---

> 审核通过后，按 §10 拆 P1 Sprint（P1.a/P1.b/P1.c 三轨并行），P2/P3 按依赖顺序后续滚动。
