# cmcc-tdlte-southbound-data-model-v2.3 技术实施方案

> Spec 源：`omcgo/规范/移动/南向数据模型/cmcc-tdlte-southbound-data-model-v2.3.md`
> 生成日期：2026-05-21
> 适用范围：MML Console（F06）+ 参数模型字典（F02）+ ACS RPC（F01）三域协同

> **2026-05-22 更新**：v1/v2 双轨已收敛为单轨 v2。`MML_V2_SCHEMA` env、`WithV2Mode` / `WithV2Schema` Options、`v1` catalog parser/upsert、`wrapByChapter` / `attachFamily` / `InferFamily`、v1 catalog 文件 `cmcc-tdlte-v2.3.json`（旧 object 维度）已全部删除。下文 v1/v2 双轨小节仅保留为历史记录。

---

## 0. 执行摘要（TL;DR）

| 条款 | 主题 | 状态 | 关键载体 |
|---|---|---|---|
| R-0 | 术语对齐（TR-069 / TR-181 / standardPath / privatePath） | ✅ 文档性 | spec 注释 |
| R-1 | 命令树两层结构（chapter → 命令叶子） | ✅ feature flag OFF 默认 | catalog Loader v2 + group_tree_repo v2Mode |
| R-2 | 操作维度展开 + 全中文命名 + tree_node_refs 关联 | ✅ | Python parser + catalog Loader v2 + migration 000149 |
| R-2.4 | 71 中文命令名权威表 | ✅ | `COMMAND_ZH_NAME` dict + unique-check panic |
| R-2.5 | 标准参数树关联 + link-health 失败清单 | ✅ | `mml_commands.tree_node_refs` JSONB + admin link-health API |
| R-3 | path 集 × RPC 映射 | ✅ | parser `make_leaf` |
| R-3.1 | 跨章节合并 | ✅ | parser `_merge_same_group_codes`（v2.3 触发 1 条）|
| R-3.2 | 17 条非可创建对象清单 | ✅ | parser `NON_CREATABLE` set |
| R-4.1 | LST 默认勾选 + 实例索引 | ✅ | SubFieldChecklist + InstanceArityInput |
| R-4.1.1 | `{i}` 取值范围 onBlur 校验 | ✅ **2026-05-21 重做完成** | `group_tree_repository.go` InstanceRangeMeta + `instanceRangeValidation.ts` + InstanceArityInput Tooltip |
| R-4.2 | MOD 仅 RW 路径 + 填新值 | ✅ | SubFieldInputList |
| ~~R-4.2.1~~ | ~~类型约束提示 + onBlur 校验~~ | 🚫 **降级 P2 可选 / deferred**（2026-05-21 spec 已降级，见 spec §R-4.2.1）| spec doc 注解 |
| R-4.3 | ADD 复合流程（AddObject + SPV 同会话） | ✅ **2026-05-21 实施完成** | `console_executor.go buildStatementCommandEntries` 拆 2 行 + `sequencer.go substituteNewInstance` 钩子 |
| R-4.4 | RMV 实例选择器 | ✅ | InstancePicker |
| R-5 | Customized 公私模板权限模型 | ✅ | visibility + owner_user_id + 用户名目录层 |
| R-6 | 命令树更新与权威源（admin overlay 保护） | ⚪ 大部分到位 | dictloader 启动期 UPSERT；admin overlay 表已建 |
| R-7 | 非目标（CTCC/CUCC/Translator/Customized 交互） | 🚫 | — |
| R-8.1-8.4 | 设备产品类型强约束 + 后端兜底 | ✅ | useDeviceSelection + console_validate.go |
| **R-8.5** | 命令兼容性警告（命令叶子 ⚠️ 图标） | ✅ **本期完成** | CompatibilityService + useCommandCompatibility |
| R-9.1-9.5 | 命令执行 API 重构（直传 path + 服务端翻译） | ✅ | execute-statements-structured + ParamModel Translator |
| R-10 | UI 文案与组件命名约定 | ✅ | Tab i18n + ParameterPathCommand 物理 rename |

**全局完成度**：spec 主体 **~85%** 已实施。剩余真实 gap 集中在 **R-4.1.1 / R-4.2.1**（前端 onBlur 校验，曾完成后被 reset）+ **R-4.3 ADD 复合流程**（后端任务编排，技术债）。

---

## 1. 架构总览

### 1.1 数据流（命令树渲染 / 执行）

```
spec MD（cmcc-tdlte-southbound-data-model-v2.3.md  ← 权威源）
   │
   │   离线 dev-only 工具：python3 parse_cmcc_tdlte_v23_v2.py
   ▼
cmcc-tdlte-v2.3.v2.json（624 params + 190 op leaves + 18 chapters）
   │
   │   启动期 Go catalog Loader v2（feature flag MML_V2_SCHEMA）
   ▼
PostgreSQL:
   - mml_param_groups （18 chapter 行，group_code=chapter:SA..SR）
   - mml_commands     （190 op 叶子，tree_node_refs JSONB 引用 standardPath）
   - mml_command_sub_fields （JOIN standard_params 拉元数据）
   ─────────────────────────────────────────────
   - standard_params  （path 单点真相，含 product/param-model 视图）
   - param_mappings   （standardPath ↔ privatePath default 映射）
   - discovered_param_mappings  （per-device-per-version 映射）
   - products         （product 装配件 → ParamModelID）
   - product_class_patterns     （productClass 正则 → product_id）

   │
   │   API
   ▼
Frontend MML Console:
   - DeviceTree    （产品类型筛选 → product_class）
   - CommandTree   （两层：chapter → 命令叶子，含 R-8.5 ⚠️ 警告）
   - RightPanel    （Control Panel：LST/MOD/ADD/RMV 单 op 视图）
   - TerminalPanel （RPC 摘要回显）

   │
   │   POST /mml/console/execute-statements-structured（R-9.2 结构化 path 入参）
   ▼
ConsoleService → fanout：每设备独立翻译 standardPath → privatePath
                       （R-9.3 ParamModel.Translator，discovered/default/passthrough）
   │
   ▼
device_tasks → ACS scheduler → CWMP RPC（GetParameterValues / SetParameterValues / AddObject / DeleteObject）
```

### 1.2 三个部署单元的角色

| 单元 | 端点 / 入口 | R-X 责任 |
|---|---|---|
| `omcgo-app` | `:8081` REST | R-1/R-2/R-3 命令树派生 + R-4 Control Panel + R-8 设备约束 + R-9 执行 API |
| `omcgo-acs` | `:7547` CWMP | R-3 RPC 实际下发（GetParameterValues 等） |
| `omcgo-worker` | 后台 | 长任务异步（暂不直接涉及 spec） |

### 1.3 模块边界

| 模块 | 路径 | spec 责任 |
|---|---|---|
| `internal/mml/catalogloader` | catalog v1/v2 双 Loader | R-1 / R-2 / R-2.5 / R-3.1 / R-3.2 |
| `internal/mml/` | ConsoleService / Handler / Repo | R-2.5 link-health / R-4 / R-5 / R-8.5 / R-9 |
| `internal/config/parammodel` | ParamRegistry + Translator | R-2.5 standard_params 数据源 + R-9.3 翻译 |
| `internal/product` | ProductRegistry | R-8 productClass 路由 / R-8.5 命令兼容性查询 |
| `omcmb/frontend-core/services/api/mmlApi.ts` | Console API client | R-9.2 结构化 API 调用 |
| `omcmb/webcode/src/pages/mml/Console/` | 三栏交互 | R-4 / R-5 / R-8 / R-10 UI 行为 |
| `migrations/scripts/parse_cmcc_tdlte_v23_v2.py` | spec MD → JSON 转换 | dev-only，R-6 离线工具 |

---

## 2. 逐条 R 实施映射

### 2.1 R-0 术语对齐（v2 增补，文档性）

| R-0 子条款 | 落地 |
|---|---|
| TR-069 是协议 / TR-181 是数据模型 | spec 注释明确 |
| OMC 内部 path 统一规则（standardPath = TR-181 形式） | `internal/config/parammodel/translator.go` 严守 |
| 三种 path 执行链路关系（standardPath ↔ privatePath ↔ runtime path） | Translator + Loader 已实现 |
| spec MD 表两栏取舍（保留 TR-181 列，IGD 列仅参考） | parser 解析时已落实 |
| 跨章节兼容承诺 | — |

**无代码改动**，纯术语规范。

### 2.2 R-1 命令树结构（两层）

| 维度 | 实施 |
|---|---|
| 18 SA-SR 章节 = 一级分组 | parser `chapterMetadata` 表 + Go `chapterMetadata` 表（group_tree_repository.go）|
| `group_code = chapter:<SA-SR>` | parser 写入 `mml_param_groups` |
| 显示名 = 章节中文小标题 | parser `chapter.zh` field |
| 排序 SA→SR | `chapterDisplayOrder(code)` 返回 1..18 |
| LTREE path 只允许一段 | catalog Loader v2 启动期 `normalizeChapterLTreePath` 校验 |
| `catalog_protected=true` | upsert_v2.go 写入 |
| **feature flag 控制** | `MML_V2_SCHEMA` env；默认 OFF（兼容 v1 dev DB） |

**关键代码**：`internal/mml/catalogloader/upsert_v2.go`、`internal/mml/group_tree_repository.go` v2Mode

**状态**：✅ 已实施，feature flag OFF 默认（dev 环境跑 v1 catalog）

### 2.3 R-2 命令叶子（含 R-2.1-2.5）

#### R-2 主体（operation 维度展开）

| 派生规则 | 实施位置 |
|---|---|
| LST 恒有 / MOD 含 RW / ADD+RMV 含 {i}+RW+非可创建外 | parser `make_leaf` 函数 |
| 不存在的操作 → 树上不出现 | parser 输出 + Loader 直接 INSERT |

#### R-2.1 全中文命名

| 规则 | 实施 |
|---|---|
| 显示名 = `<OP> <command_zh_name>` | parser 用 `COMMAND_ZH_NAME` 字典查表 |
| **同名禁令**（跨章节互不相同） | parser 启动期 unique-check（71 条）|
| **未命中即 panic** | parser `raise SystemExit(1)` |
| v1 父级段前缀消歧规则作废 | v2 parser 完全不再处理 |

#### R-2.2 DB 存储 + tree_node_refs

| 字段 | 实施 |
|---|---|
| `(source='standard', group_id, group_code_object, operation_type)` 四元组唯一 | DB unique 索引 |
| 不写 `target_paths`（v2 改用 `tree_node_refs`） | upsert_v2.go INSERT 列出 `tree_node_refs` JSONB |
| `command_code = <OP>:<group_code_object>` | parser 生成 |
| `logical_name_i18n.zh-CN = command_zh_name` | parser 写入 |

**migration**：`000149_mml_catalog_v2_columns.sql` 加 `tree_node_refs JSONB` + GIN 索引

#### R-2.3 叶子拆分

| 规则 | 实施 |
|---|---|
| 父子 object 严格分离 | parser `parent_object_path` + 路径前缀比对 |
| 数字实例号归一化为 `{i}` | parser `GREEK_RE` 折叠 `{iα..iε}` → `{i}` |

#### R-2.4 71 条权威表

- parser `COMMAND_ZH_NAME` dict（lines 66-156 of `parse_cmcc_tdlte_v23_v2.py`）
- spec MD 表 line 134-206
- 双向唯一性自检

#### R-2.5 关联标准参数树

| 子条款 | 实施 |
|---|---|
| 引用键 = standardPath（软外键） | catalog Loader v2 `upsertCommandV2` + `preloadStandardParams` |
| 命中 → tree_node_refs 加入 | upsert_v2.go |
| 未命中 → link-health 失败清单 | `mml_catalog_link_health` 表（migration 000149） |
| Admin API `GET /api/v1/mml/catalog/link-health` | `internal/mml/catalogloader/linkhealth.go` + admin handler |
| 链接 100% 成功（dev DB 实测） | `link_failures=0` |

**关键代码**：`internal/mml/catalogloader/upsert_v2.go`、`linkhealth.go`

**状态**：✅ 完整实施

### 2.4 R-3 操作 × path × RPC 映射

#### R-3 主体

| OP | RPC | path 内容 | 实施 |
|---|---|---|---|
| LST | GetParameterValues | 全集（RO + RW） | parser `paths = all_paths` |
| MOD | SetParameterValues | 仅 RW 子集 | parser `paths = rw_paths` |
| ADD | AddObject | 父级对象路径 | parser `parent_object_path(norm)` |
| RMV | DeleteObject | 实例路径模板（runtime 拼实例号） | 同 ADD |

#### R-3.1 跨章节合并

- parser `_merge_same_group_codes` 函数
- 合并规则：`target_paths` union / `perm` OR / 命令归首次出现章节
- v2.3 实际触发 1 条：`Device.Services.FAPService.{i}.*`（SF+SH → SF）
- 合并日志写入 catalog JSON `mergeLog` 字段

#### R-3.2 17 条非可创建对象清单

- parser `NON_CREATABLE` set（lines 162-180）
- 触发即跳过 ADD/RMV 派生
- v2.3 过滤掉 17 × 2 = 34 条命令行
- 维护强约束：必须改 spec R-3.2，禁止 DB overlay

**状态**：✅ 完整实施

### 2.5 R-4 操作面板行为

#### R-4.1 LST

- `SubFieldChecklist` 渲染勾选列表
- 默认全勾选（spec 默认）
- 实例索引输入由 `InstanceArityInput` 处理（Greek 字母 iα/iβ/iγ）
- **状态**：✅ 主体完成

#### R-4.1.1 {i} 取值范围校验 **未实施（reset）**

- 后端 P4.a 曾完成 `mml_commands.instance_range_meta` JSONB 透传到前端 `GroupTreeCommand.InstanceRangeMeta`
- 前端 P4.b 曾完成 `InstanceArityInput` 消费 + `validateInstanceLayer` 纯函数 + Tooltip
- **2026-05-21 用户 `git reset --hard origin/main`** 全部丢弃
- 当前主分支无相关代码 / 数据库列已 DROP

**重做成本**：~400 LOC backend (BuildTree SQL + struct field) + frontend (validation pure fn + UI)，整套设计可参考被 reset 的 commit `999524d5` / `28ce0ac8`（仍在 reflog 中可恢复）

#### R-4.2 MOD

- `SubFieldInputList` 过滤 READ_ONLY（line 28-31）保留 RW
- 用户填新值 → `setValue(uid, mmlCode, value)` 写 store
- **状态**：✅ 主体完成

#### R-4.2.1 类型约束 onBlur 校验 **未实施（reset）**

- 曾通过 5 阶段 P4.c.1-5 实施：migration 000150 加 `standard_params.constraint_meta JSONB` + `constraint_hint TEXT`、parser 升级、seed 回填、API 透传、前端校验
- 2026-05-21 用户全量 reset
- 当前主分支：列已 DROP，sub-fields API 仍硬编码 `'{}' AS constraint_text_i18n`（admin_repository.go:255）

**重做成本**：~5 commits，~1500 LOC。设计文档已存档（不在仓库；reflog 可恢复）

#### R-4.3 ADD 复合流程 **部分实施**

- 前端 `SubFieldInputList` 在 ADD 模式渲染 RW 字段列表（同 MOD）
- **后端 `console_executor.go` 注释明确说 1MML=1RPC 不做 AddObject + SetParameterValues 复合**
- 结果：用户能填初始值但实际只 AddObject，新实例字段为空

**待实施**：ACS task scheduler 支持串联 RPC（AddObject 返回新 instance number → 拼进 SPV path）。涉及 ACS session 编排，复杂度中高。

#### R-4.4 RMV

- `InstancePicker` 提供实例选择器
- POST `/ops/commands/rpc` action="get_param" 探测当前实例集合
- 多选 → fanout DeleteObject
- **状态**：✅ 已实施

### 2.6 R-5 Customized 公私模板

| 子条款 | 实施 |
|---|---|
| 5.1 Public Template 任意用户可见 + 命令平级 | DB `mml_custom_commands.command_scope='public'` + CommandTree `publicGroup` 渲染 |
| 5.2 Private Template 用户名目录层 + 角色过滤 | `owner_user_id` 字段 + `service.go` 按 owner+role 过滤 + `CommandTree.tsx` `byCreator` 分组 |
| 5.3 Customized 排末尾 + 不深嵌套 | `index.tsx` `children` 顺序末位；`buildCustomTreeData` 强制平级 |

**关键代码**：`internal/mml/pg_repository.go`（visibility 过滤）+ `CommandTree.tsx`（`buildCustomTreeData`）

**状态**：✅ 完整实施

### 2.7 R-6 命令树更新与权威源

| 要求 | 实施 |
|---|---|
| spec MD 是 SA-SR 唯一权威源 | parse_cmcc_tdlte_v23_v2.py 入口 |
| 离线 dev-only MD→JSON 工具，生产镜像不含 Python | parser 在 `migrations/scripts/` 而非 `cmd/`/`internal/` |
| 启动期 Go Loader 幂等 UPSERT | dictloader 框架（参考 §5.3 CLAUDE.md） |
| admin overlay 表保护用户调整 | `mml_command_sub_fields` overlay 字段（default_selected 等）|
| 删除只删 `source='standard'` 行 | catalog Loader 启动期清理逻辑 |

**状态**：⚪ 大部分到位（admin 热重载 API `POST /api/v1/mml/catalog/reload` 待补；当前需重启 app 进程）

### 2.8 R-7 非目标

| 非目标 | 状态 |
|---|---|
| 不涉及 CTCC / CUCC 命令树导入 | ✅ 保持 |
| 不修改 Customized 交互逻辑 | ✅ Customized 独立流程 |
| 不修改 ParamModel Translator 机制 | ✅ Translator 维持 T-0098 P1-P5 |

### 2.9 R-8 设备列表产品类型约束

| 子条款 | 实施 |
|---|---|
| 8.1 默认值 = 字典首项 | `useDeviceSelection.ts:56` |
| 8.2 必选 + 不允许 allowClear | `DeviceTree.tsx:136-147` |
| 8.3 单一性 + 切换清空设备 | `useDeviceSelection.ts handleFilterChange` |
| 8.4 后端兜底 ErrDevicesMixedProductClass | `console_validate.go:29-39` |
| **8.5 命令兼容性警告 ⚠️** | ✅ **本期完成**（CompatibilityService + useCommandCompatibility + WarningOutlined）|

**关键代码**：`internal/mml/command_compatibility.go`、`omcmb/frontend-core/src/hooks/api/useMmlConsole.ts useCommandCompatibility`

### 2.10 R-9 命令执行 API 重构

| 子条款 | 实施 |
|---|---|
| 9.1 移除 MmlEditor 文本预览 | v2.4 D37 已下线 |
| 9.2 API 直传 path 重构 `/execute-statements-structured` | ConsoleService `ExecuteStructuredStatements` + frontend `statementToStructured` |
| 9.3 服务端翻译 standardPath → privatePath | ParamModel Translator `TranslateToPrivate(ctx, productID, swVersion, standardPath)` |
| 9.4 多设备独立翻译 | fanout per device_task |
| 9.5 Customized 走结构化通道 | AddTemplateModal 已绑定 commandCode + sub_fields |

**状态**：✅ 完整实施（结构化通道为主，MML 文本通道作为脚本任务过渡保留）

### 2.11 R-10 UI 文案与命名

| 项 | 实施 |
|---|---|
| Tab 命名"操作面板"/"参数路径指定" | `i18n/zh-CN/index.ts` line 3965-3966 |
| `ParamPathExpert.tsx` → `ParameterPathCommand.tsx` 物理 rename | RightPanel.tsx line 14 import 已切 |
| i18n key 补 `mml.tabs.controlPanel` / `mml.tabs.parameterPathCommand` | 已加 |

**状态**：✅ 完整实施

---

## 3. 关键不变量与设计取舍

### 3.1 设计取舍（已落地）

| 取舍 | 选择 | 代价 |
|---|---|---|
| 命令树层级 | **严格两层**（章节 → 命令叶子）禁止子分组 | 跨章节同 path 需用 R-3.1 合并而非分裂 |
| path 真相源 | `standard_params` 单点真相，`mml_commands.tree_node_refs` 软引用 | catalog Loader 启动期需 link-health 校验 |
| 命名权威表 | 71 条中文名硬编码 in parser，**未命中即 panic** | spec 改 path 必须同步改权威表 |
| 操作维度 | catalog 编译期固化（每 op 独立 DB 行），非运行时 toggle | 同 object 在 DB 多行（LST + MOD + ADD + RMV） |
| MD → JSON 转换 | dev-only Python 离线工具，**生产镜像不含 Python** | spec 变更需开发者本地运行脚本 + PR JSON 产物 |
| ParamModel 默认 vs discovered | R-8.5 用 default（产品级语义） | discovered 仅运行时翻译用 |
| v1 / v2 双轨 | feature flag `MML_V2_SCHEMA` + `WithV2Mode/WithV2Schema` Options | 维护两条 Loader 代码路径直到切流量完毕 |

### 3.2 失败模式（panic 阻止入库）

- spec 改 group_code 未更中文名表 → catalog Loader panic
- 71 条中文名跨章节撞名 → panic
- LTREE path 深度 > 1 → panic
- 这是**故意的硬约束**，防止隐式 fallback 污染线上数据

---

## 4. 当前 Gap 清单（按优先级）

### P0 — spec 明文要求但完全未实施

| Gap | spec 条款 | 影响 | 估计工作量 |
|---|---|---|---|
| ~~`{i}` 取值范围 onBlur 校验~~ | ~~R-4.1.1~~ | 已重做（2026-05-21 cherry-pick `999524d5`+`28ce0ac8`）| ✅ 已闭环 |
| ~~类型约束 onBlur 校验 + 提示~~ | ~~R-4.2.1~~ | 已降级为 P2 可选（spec 同步降级），不再视为 gap | — |

### P1 — 部分实施 / 功能缺陷

| Gap | spec 条款 | 影响 | 估计 |
|---|---|---|---|
| ~~ADD 复合流程（AddObject + SPV 同会话）~~ | ~~R-4.3~~ | ✅ **已闭环**（2026-05-21 通过 ConsoleService 拆 2 行 commands + Sequencer .{NEW}. 替换实现，未改 ACS、未改 schema）| — |

### P2 — Nice-to-have / 工程债

| Gap | spec 条款 | 影响 | 估计 |
|---|---|---|---|
| Admin 热重载 API `POST /api/v1/mml/catalog/reload` | R-6 | 当前需重启 app 进程才能 reload catalog | 小（~50 LOC）|
| catalog Loader 启动期清理 `source='standard'` 失效行 | R-6 | spec 删除条款时 DB 残留旧 command 行 | 小（~30 LOC）|
| v2 feature flag 默认 ON | R-1 | 当前默认 OFF（dev DB 还在跑 v1） | 0 LOC（仅 deployment config 切换 + 验证）|
| v1 / v2 双轨下线 v1 | — | catalog Loader 维护成本 | 大（需 v2 在生产稳定后才能动手）|

### Bug / 追踪

| 问题 | 来源 | 当前状态 |
|---|---|---|
| `Customized 命令` 是否参与 R-8.5 兼容性警告 | R-8.5 + R-5 交叉 | 已在 hotfix commit `3faf58ee` 中决策：**不参与**（spec §R-5 customized 不在 R-8.5 范围内）|
| `mml_commands.target_paths` 与 `tree_node_refs` 双轨期 | R-2.5 | catalog Loader v2 同时写两列，DROP target_paths 待 v2 切流量完毕 |

---

## 5. 风险与回退

### 5.1 Loader 切流量（v1 → v2）

**触发**：`MML_V2_SCHEMA=true` env

**风险**：
- v2 数据全量重写 mml_commands（不同 group_code 命名）
- 用户已创建的 Customized 命令不受影响（source='admin' 隔离）
- 前端 CommandTree 切到 chapter 顶层视图（视觉差异大）

**回退**：env 切回 false，重启 app — group_tree_repository 自动按 v2Mode=false 回退 v1 视图

### 5.2 命令兼容性 API 调用风暴

**触发**：用户高频切产品类型（DeviceTree 下拉）

**风险**：每次切产品 → React Query 重新 fetch `/mml/console/command-compatibility`

**已有兜底**：staleTime 5min + queryKey 含 productClass 自动缓存

### 5.3 spec MD 升级不同步

**触发**：spec MD 改 path 但未更 71 条中文名表

**已有兜底**：catalog Loader 启动期 panic（spec 设计约束）

---

## 6. 不在范围（spec R-7 + 项目实际）

| 主题 | 备注 |
|---|---|
| CTCC / CUCC 命令树导入 | spec 明文 R-7 排除 |
| Customized 模板交互改造 | spec 明文 R-7 排除 |
| ParamModel Translator 改造 | spec 明文 R-7 排除（沿用 T-0098 P1-P5） |
| 5G NR (SA) 命令树 | spec v2.3 仅覆盖 LTE，NR 待后续 spec 版本 |
| 设备配置基线（baseline）/ 配置模板（template） | F02 子模块，与本 spec 主线无关 |
| PM / 告警 / MR 数据管线 | F03/F04/F05，与本 spec 无关 |

---

## 7. 未来扩展路径

### 7.1 spec v2.4 / v3.0 升级流程

1. spec MD 增删 `####` 命令 / 改权限矩阵 / 改 path
2. （如新加 group_code）同步更新 §R-2.4 71 条中文名表
3. 离线运行 `python3 omcgo/migrations/scripts/parse_cmcc_tdlte_v23_v2.py`
4. 提交 `cmcc-tdlte-v2.3.v2.json` 产物到 git
5. App 重启 → catalog Loader v2 幂等 UPSERT
6. （如增删 standard_path）同步运行 `gen_constraint_meta_backfill.py` + 新 backfill migration

### 7.2 增加运营商（CTCC / CUCC）

- 新建 `omcgo/规范/电信/...` / `omcgo/规范/联通/...` spec MD
- 新建对应 parser（复用 `parse_cmcc_tdlte_v23_v2.py` 框架 + 各运营商专有 chapter / NON_CREATABLE / 命名表）
- 新增 `data_source` 列区分（`mml_commands.data_source = 'cmcc' | 'ctcc' | 'cucc'`）— migration 待立
- 设备 productClass + carrier 双键路由命令树

### 7.3 NR (5G SA) 命令树扩展

- spec 新增 NR 章节（如 SS / ST / ...）
- 不影响 LTE 现有 18 章节
- 命名表 71 条扩展（如 +20 条）
- 自动通过 parser unique-check

### 7.4 v1 catalog Loader 下线

- 待 `MML_V2_SCHEMA=true` 在生产稳定 ≥ 1 release
- 删除 `internal/mml/catalogloader/upsert.go` v1 路径
- 删除 `mml_commands.target_paths` 列（migration）
- 删除 `group_tree_repository.go` v1 wrap/family 代码

---

## 8. 实施优先级建议

按 ROI 排序，建议下一步动作：

| 排序 | 动作 | 理由 |
|---|---|---|
| 1 | **生产切到 v2 schema** (`MML_V2_SCHEMA=true`) | 0 LOC 工作量；解锁所有 v2 功能（chapter 视图 / tree_node_refs / link-health）|
| 2 | **R-4.1.1 + R-4.2.1 重做** 或 **正式放弃**并更新 spec | 模糊状态最浪费决策力；曾实施过的代码已被 reset，需要明确方向 |
| 3 | **R-4.3 ADD 复合流程**（如果优先级高于上两条）| 真实功能缺陷，但涉及 ACS scheduler，风险大于 P0 |
| 4 | **Admin 热重载 API**（R-6） | 小工作量提升运维体验 |
| 5 | catalog Loader cleanup `source='standard'` 失效行 | spec regression 路径打通 |

---

## 附录 A：spec 与项目文件对照速查表

| spec 条款 | 关键文件 | 关键 DB 表 / 列 |
|---|---|---|
| R-0 | （文档性） | — |
| R-1 | `internal/mml/catalogloader/upsert_v2.go`, `group_tree_repository.go` | `mml_param_groups.group_code='chapter:*'` |
| R-2.1-2.5 | `internal/mml/catalogloader/upsert_v2.go`, `linkhealth.go`, parser | `mml_commands.tree_node_refs`, `mml_catalog_link_health` |
| R-2.4 | parser `COMMAND_ZH_NAME` (lines 66-156) | — |
| R-3 | parser `make_leaf` | `mml_commands.operation_type` |
| R-3.1 | parser `_merge_same_group_codes` | — |
| R-3.2 | parser `NON_CREATABLE` set | — |
| R-4.1 | `SubFieldChecklist.tsx`, `InstanceArityInput.tsx` | — |
| R-4.1.1 | **❌ 待重做** | — |
| R-4.2 | `SubFieldInputList.tsx` | — |
| R-4.2.1 | **❌ 待重做** | — |
| R-4.3 | `console_executor.go`（⚪ 1MML=1RPC 未做复合）| — |
| R-4.4 | `InstancePicker.tsx` | — |
| R-5 | `CommandTree.tsx buildCustomTreeData`, `pg_repository.go visibility 过滤` | `mml_custom_commands.command_scope`, `owner_user_id` |
| R-6 | `internal/mml/catalogloader/loader.go` | dictloader 框架 |
| R-8.1-8.4 | `useDeviceSelection.ts`, `DeviceTree.tsx`, `console_validate.go` | `devices.product_class`, `product_class_patterns` |
| R-8.5 | `command_compatibility.go`, `useCommandCompatibility.ts` | （只读跨表 JOIN） |
| R-9.1-9.5 | `ConsoleService.ExecuteStructuredStatements`, `Translator.TranslateToPrivate` | `param_mappings`, `discovered_param_mappings` |
| R-10 | i18n keys, `RightPanel.tsx`, `ParameterPathCommand.tsx` | — |

## 附录 B：本会话期成果（commit log）

| Hash | 类型 | spec 条款 | 内容 |
|---|---|---|---|
| `48c240d1` | docs | R-8.5 | 命令兼容性警告设计文档 |
| `992f9001` | feat backend | R-8.5 | CompatibilityService + endpoint + 单测 |
| `bb8055d7` | feat frontend | R-8.5 | UI 装饰 + hook + i18n |
| `3faf58ee` | fix frontend | R-8.5 + R-5 | hotfix renderOpLeafTitle Customized 兼容 |
| （本 commit） | docs | 全 spec | 技术实施方案 |

**本地领先 origin/main 5 个 commit**，待 push。
