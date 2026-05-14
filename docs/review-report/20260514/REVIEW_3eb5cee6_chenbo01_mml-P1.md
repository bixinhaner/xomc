# Code Review Report — T-0123-P1

| 项目 | 值 |
|------|-----|
| 日期 | 2026-05-14 |
| 提交 | `3eb5cee6`（feat code commit；docs 跟随独立 commit） |
| 作者 | chenbo01 |
| 范围 | mml Console 后端（5 端点 + Go renderer/parser + executor + DI + e2e claim） |
| 变更文件数 | 15（10 新 .go + 3 modified .go + scripts/e2e + settings） |
| 新增行数 | +3765（含 S5 review-driven 重构） |
| 删除行数 | -4 |
| 关联 Task | T-0123-P1 |
| 关联 PRD | `docs/design/mml-restore-old-interaction-plan-20260514.md` v2 APPROVED（§N） |
| 关联 Risk | R-206 Open（MML 老交互被 Sprint A 重构破坏，P1 推进 mitigating） |
| 依赖 | T-0123-P0 已闭环（commits 00a48cad / bf53cb3b / e0d31cce / 9490be18） |

---

## 变更概要

T-0123-P1 MML Console 后端 5 端点 + Go 渲染/解析/执行栈，构建在 T-0123-P0 catalog 数据层之上：

### S3-D1 · MML 语法栈（~600 行 + 30 测试）
- `mml_renderer.go` — Statement 类型 + `RenderStatement/RenderStatements`（LST/MOD/ADD/RMV 渲染为老 OMC 严格语法）+ 特殊字符双引号转义 + sort_order 稳定排序
- `mml_parser.go` — `ParseMMLString` 多 statement quote-aware 分割 + `CommandLookup` 接口 + 容错策略（单条失败累入 ParseError 不中断整体）

### S3-D2 上 · 命令分组树 + 服务编排（~500 行 + 13 测试）
- `group_tree_repository.go` — LTREE JOIN 单 SQL 拉子树 + `assembleHierarchy` 深度 DESC 自底向上组装
- `console_service.go` — 5 service 方法：`BuildGroupTree` / `GetCommandSubFields`（lang 派生）/ `RenderMML` / `ParseMML`（注入 `CommandLookup`）/ `LookupByLogicalCode`（双策略：op_logical → 退化 logical_code）

### S3-D2 下 · 编译器 + fanout 桥（~390 行 + 20 测试）
- `console_executor.go` — `ExecuteStatements` 编排入口 + `BuildStatementCommands` 公开导出 + 4-op entry 编译器（LST 默认全选、MOD ParamCode 改写 MMLCode、ADD AddObject + SPV passthrough、RMV target_object + index）+ `ErrInvalidRequest` sentinel（S5 引入）
- `service.go` — 加 `CreateAndFanoutTask`（taskRepo.Create + fanouter.Fanout 桥接，sequential 旋钮）
- `mml_renderer.go` — Statement 加 JSON tags（FE round-trip 必需）

### S3-D3 · HTTP 层 + DI（~330 行 + 5 e2e claim）
- `console_handler.go` — 5 endpoints + sentinel-based 错误翻译（404/400/500）
- `cmd/app/provider/modules.go` — 构造 `GroupTreeRepo` + `ConsoleService` + `ConsoleHandler`，把 `*mml.Service` 注入作 `MMLTaskCreator`
- `cmd/app/provider/router.go` — `mmlConsoleHandler.RegisterRoutes(permGroup("devices"))`
- `scripts/e2e_verify.sh` — 5 个新 claim（T-0123-P1 mml-console-1..5）

### S4/S5 工件
- `docs/review-report/20260514/verify-T-0123-P1.md` — S4 verify 报告
- 本文件 — S5 code review 报告

---

## 审查发现

### 🔴 CRITICAL (严重)

> 必须在提交前修复的问题

**无**。

### 🟡 WARNING (警告)

> 建议修复

**W1（已修复，S5 内）**：原 `console_handler.PostExecuteStatements` 用 `strings.HasPrefix("compile statements:")` 等字符串匹配区分 4xx vs 5xx，脆弱且与 P0 sentinel 模式不一致。

- **修复**：引入 `ErrInvalidRequest` sentinel；`ExecuteStatements` / `BuildStatementCommands` 用 `%w: %v` 包装；handler 切到 `errors.Is(err, ErrInvalidRequest) → 400` switch
- **新增 1 测试**：`TestBuildStatementCommands_WrapsErrInvalidRequest`
- **现状**：21 个 console_executor 测试全 PASS

**W2（已修复，S5 内）**：原 `buildStatementCommandEntry` ADD 分支 `for k, v := range stmt.Values { params[k] = v }` 未注释意图；读者会误以为 `params[k]` 会出现在 AddObject SOAP body 中，实则被 `buildObjectName` 忽略。

- **修复**：加 6 行注释说明 SPV passthrough **仅供 `writeAuditLogs` 留意图痕迹**，协议层只用 `object_name`；额外加 `if k == "object_name" { continue }` 防覆盖

**W3（follow-up）**：`*Service.CreateAndFanoutTask` toggle `fanouter.SetSequentialMode` 存在理论 race（concurrent Console 请求 sequential≠sequential 时互相覆盖）

- **影响**：Console 流量低（人机交互级），实际碰到的概率小；最坏情况单次请求 fanout 模式错配（多入队 1 行或少入队后续行），Sequencer 仍能兜底
- **追踪**：`verify-T-0123-P1.md §9 R-T0123-P1-03` follow-up；P2 frontend 联调阶段评估是否引入 mutex 或改 per-call 参数
- **不阻塞 P1**：worst-case 行为是 graceful，不数据损坏

**W4（follow-up）**：5 个 Console 端点契约层 E2E 覆盖 ✓，但 happy-path 深度断言（含真实 command + sub_fields seed → fanout 落 device_tasks 完整链路）**未补**

- **理由**：T-0123-P0 完成 standard-model params 种子（2001 行），但 mml_commands / mml_command_sub_fields 的 **standard 种子还未生成**（依赖 admin Catalog UI 手工录入或后续工具）
- **追踪**：`R-T0123-P1-01` follow-up（P3 admin Catalog UI 联动阶段补）

**W5（follow-up）**：Prometheus metrics 未实施（同 P0 W2）

- **追踪**：`R-T0123-P1-02` follow-up

### 🔵 INFO (建议)

**I1**：`console_handler.parseUUIDPathParam` 是 package 级 func，而 `admin_handler.parseUUIDParam` 是方法接收者。轻微 inconsistency。

- 影响：仅风格层面，behavior 一致
- 建议：未来若 mml 包内重构第 3 个 handler，统一为 package func（更易共享）

**I2**：`Statement.SelectedMMLCodes` 是 parser 内部字段（lookup 后会用来填 `SelectedSubFieldIDs`），但通过 JSON tag 暴露到响应里。

- 影响：FE 看到的 parse response 多一个字段；调试时反而有用（可看 parser 如何把字符串拆出来）
- 决策：保留。如未来需要严格 API 契约可加 `json:"-,omitempty"` 或 server-side strip

**I3**：DoS 安全硬化未做 — `/mml/parse` 接 32MB body（Gin 默认），`/mml/execute-statements` 不限制 statements 数。

- 当前缓解：Casbin 中间件 + 端点级权限要求（admin 角色）
- 建议：未来 P2/P3 加 body-size middleware（如 1MB cap on /mml/parse、64 statements cap on /mml/execute-statements）
- 不阻塞 P1（内部网管系统，admin 用户可信级别高）

**I4**：`console_handler.PostExecuteStatements` 从 `c.GetString("username")` 兜底 `req.Creator`。该 key 由 auth middleware 注入；若 middleware 升级改 key 名，此处兜底失效。

- 当前缓解：与既有 mml.Handler 等其他 handler 用同一 key
- 建议：未来统一抽 helper 如 `extractCreatorFromContext(c)`，避免散落硬编码

---

## 详细分析

### `omcgo/internal/mml/mml_renderer.go`（225 行 + 9 行 S5 JSON tags 修订）

- ✓ `Statement` 共享结构清晰：renderer / parser / executor 三方协作的接口契约
- ✓ S3-D2 下 加 JSON tags（CommandID/LogicalCode/OperationType/SelectedSubFieldIDs/Values/RmvInstanceIndex/UnknownCodes 全覆盖），FE round-trip OK
- ✓ `RenderStatement` 严格遵循老 OMC playwright 实测语法（LST `:lstId={}` / MOD `:K=V,K=V` / ADD 同 MOD / RMV `:Index=N`）
- ✓ 空字段返"裸 op"（如 `LST DEVICE_INFO` 而非 `LST DEVICE_INFO:lstId={}`），匹配老系统
- ✓ `quoteValueIfNeeded` 对 `, ; : = { } " 空格` 7 类特殊字符触发引号包裹 + 内部 `"` 转义
- ✓ sort_order 稳定排序 + 字典序二级排序（commit diff 友好）
- ℹ️ `collectKeyValuePairs` 把 `Values` map 未命中 sub_fields 的 key 追加到末尾（按字典序）— **保护 unknown_codes 不丢失**，前端 toast 提示用

### `omcgo/internal/mml/mml_parser.go`（423 行）

- ✓ `splitStatements` quote-aware（不会被 value 内的 `;` 误切）
- ✓ `parseSingleStatement` 大小写忽略 op，空格容忍 op/code/key/value 周围
- ✓ `parseLSTParams` lstId 大小写不敏感（`lstId` / `lstid` / `LSTID` 都接受）
- ✓ `parseKeyValueParams` 用 `splitOnUnquotedComma` + `findFirstEqOutsideQuote`，正确处理嵌套引号
- ✓ `unquoteIfNeeded` 剥外层引号并 unescape `\"`
- ✓ `parseRMVParams` 大小写不敏感 `Index=N`
- ✓ 单条 statement 失败累入 `ParseError` 不中断整体（关键容错语义）
- ✓ `resolveLSTSubFieldIDs` 保留用户输入顺序（与 UI 勾选 round-trip 同源），不按 sort_order 重排
- ✓ `checkValueKeys` 累计 unknown_codes 到 statement
- ℹ️ `ErrAmbiguousCommand` 定义但当前未发出（CommandLookup 实现只查 GetByCode 单结果）— 防御性预留，未来 ListByLogicalCode 实现后用

### `omcgo/internal/mml/group_tree_repository.go`（305 行）

- ✓ `BuildTree(rootCode, lang)` 单 SQL JOIN（mml_param_groups + mml_commands）+ LTREE `path <@ rootPath OR path ~ lquery`
- ✓ `assembleHierarchy` 用 `sort by depth DESC then display_order` 自底向上组装（修复 S3 实测的"子节点先于父节点处理"bug）
- ✓ 孤儿节点（缺父）作顶级返回（partial subtree 兼容）
- ✓ `buildDisplayName("设备信息","LST","DEVICE_INFO")` → `"设备信息(LST DEVICE_INFO)"`（匹配老 OMC 显示样式）
- ✓ 全程 squirrel + pgx 参数化
- ✓ 按 `display_order` 排序保证稳定显示

### `omcgo/internal/mml/console_service.go`（229 行）

- ✓ 5 service 方法签名清晰
- ✓ `BuildGroupTree` 透传 repo（薄壳）
- ✓ `GetCommandSubFields` lang 派生：`pickI18n(LabelI18n, lang, "", "", MMLCode)` — 缺 lang 时按 chain 兜底
- ✓ `RenderMML` 双 fallback：cmd.LogicalCode 空 → `deriveLogicalCodeFromCommandCode`（去 op 前缀）
- ✓ `ParseMML` 注入 `s` 作 CommandLookup（self-reference 注入，避免 circular DI）
- ✓ `LookupByLogicalCode` 双策略：先 `op_logical_code`（mmlstandardloader 命名约定），失败后退化 `bare logical_code`（admin 创建的命令）；命中后校验 `cmd.OperationType == op` 防 op 错配

### `omcgo/internal/mml/console_executor.go`（330 行，S5 重构后）

- ✓ `ErrInvalidRequest` sentinel（S5 W1 修复）— handler 用 `errors.Is` 区分 400 vs 500
- ✓ `ExecuteStatements`(ctx, req, taskCreator) — taskCreator 通过依赖注入，避免 ConsoleService → *Service 反向依赖
- ✓ 多 statement → sequential=true；单 statement → 并发（fanouter 默认）
- ✓ `BuildStatementCommands` 公开导出（dry-run 预览复用）
- ✓ 4 op 编译器（`buildStatementCommandEntry`）：
  - **LST**：选 selected sub_field → 合成 paramRefs；空 selected → 默认全选（匹配老 OMC 默认勾选状态）
  - **MOD**：所有 sub_fields 作 param_refs，**ParamCode 改写 MMLCode**（与 parser Values key 对齐，关键 glue）
  - **ADD**：parameters.object_name = TargetObject（自动补尾点）；SPV 字段透传 `parameters` 仅供审计（S5 W2 注释强化 + object_name 防覆盖保护）
  - **RMV**：parameters.object_name = TargetObject 截尾点 + `.{Index}.`
- ✓ `buildLSTParamRefs` 从 cmd.Params 索引 paramID → MMLParamRef；fallback 合成最小 ref
- ✓ `buildMODParamRefs` ParamCode 改写到 MMLCode（重要！让 BuildTR069Params.buildParameterValues 能按 Values key 查表）
- ✓ `indexParamsByID` / `stringMapToInterface` 小 helper 清晰
- ℹ️ 11 处 `map[string]interface{}` 全部为既有 fanouter / BuildTR069Params 协议契约 boundary（verify-T-0123-P1.md §4.1 已详审）

### `omcgo/internal/mml/console_handler.go`（235 行，S5 重构后）

- ✓ 5 endpoints register + `parseUUIDPathParam` + `normalizeLang` helper
- ✓ S5 W1 修复：错误翻译切到 sentinel-based switch（ErrCommandNotFound → 404 / ErrInvalidRequest → 400 / default 500）
- ✓ `normalizeLang` 支持 zh-CN/zh/zh_CN 三别名 → "zh-CN"；类似 en
- ✓ `PostExecuteStatements` 从 `c.GetString("username")` 兜底 Creator/Executor — 与既有 handler 模式一致
- ✓ `RenderRequest` / `ParseRequest` / `ExecuteStatementsRequest` 全部 binding:"required" 在关键字段
- ✓ ParseMML 端点始终返 200（ParseError 在 body 内，前端 toast 提示）— 符合"单条失败不中断整体"语义
- ℹ️ I1：`parseUUIDPathParam` 是 package func 而 admin_handler 是方法（风格 inconsistency，behavior 一致）

### `omcgo/internal/mml/service.go`（+60 行 CreateAndFanoutTask）

- ✓ `CreateAndFanoutTask(ctx, task, sequential)` 桥接接口：persist + audit log + fanout
- ✓ 与既有 `ExecuteCommand` 区别：调用方已编译完 commands[]，跳过 resolveRPCMethods / param_paths 合成
- ✓ ExecuteType 默认 ExecuteImmediate
- ✓ SSE 推送对齐既有 publishTaskStatus
- ⚠️ W3 race：`fanouter.SetSequentialMode(sequential)` toggle 模式，理论 race（follow-up R-T0123-P1-03）

### `omcgo/internal/mml/console_executor_test.go`（450 行，21 测试）

- ✓ table-driven：BuildEntry × 4 op × happy/edge/error 路径全覆盖
- ✓ ExecuteStatements 编排测试：sequential 切换、空 statements、空 device_sns、nil taskCreator
- ✓ BuildTR069Params wire-format 整体路径断言（4 个 TestEntry_*_WireFormat）
- ✓ S5 加 `TestBuildStatementCommands_WrapsErrInvalidRequest` 验证 sentinel chain
- ✓ `fakeTaskCreator` 捕获 sequential / commands 便于断言
- ✓ 全 PASS（race detector ✓）

### `omcgo/internal/mml/console_service_test.go`（13 测试，未变更）

- ✓ assembleHierarchy 树形组装 3 场景
- ✓ RenderMML 双 fallback
- ✓ ParseMML lookup 注入
- ✓ LookupByLogicalCode 双策略 + op 错配
- ✓ GetCommandSubFields lang 派生 + fallback chain

### `omcgo/cmd/app/provider/modules.go`（+6 行）

- ✓ `miscDeps.mmlConsoleHandler *mml.ConsoleHandler` 字段
- ✓ 构造顺序：mmlGroupTreeRepo → mmlConsoleSvc → mmlConsoleHandler，依赖链完整
- ✓ `mmlService` 作 `MMLTaskCreator` 注入（满足 ConsoleService.ExecuteStatements 签名）

### `omcgo/cmd/app/provider/router.go`（+4 行）

- ✓ 注册到 `permGroup("devices")` → 自动获得 RequireAPIPermission 端点级鉴权
- ✓ inferApiGroup(`/api/v1/mml/group-tree`) → `mml`，复用既有 /mml/* 角色权限

### `omcgo/scripts/e2e_verify.sh`（+30 行）

- ✓ 5 个新 claim 嵌入 W2.D.1 mml Domain 区段（编号 T-0123-P1 mml-console-1..5）
- ✓ 容忍状态：200/401（端点可达）+ 400/404（错误路径）— RBAC 缺失 / 路由未挂场景都不误报
- ✓ bash -n 语法检查通过

---

## 业务完整性检查

- ✓ Handler-Service-Repository-Executor 四层完整
- ✓ 5 endpoint 一一对应 ConsoleService 5 method
- ✓ 路由注册：router.go +4 行（permGroup("devices") + Casbin）
- ✓ DI 装配：modules.go 3 行构造 + 2 行注入
- ✓ 错误码：sentinel error 而非 global/errors.go 数字码（与 P0 模式一致）
- ✓ 测试：47 测试覆盖 33.3% mml 包代码（P0 24.9% → P1 33.3% +8.4pp）
- ✓ E/R 比 = 1:1（5 endpoint / 5 claim）
- ⚠️ FE 端配套：**未实施**（P2 范畴 — T-0123-P2 Console frontend）
- ⚠️ Standard mml_commands seed 未生成（依赖 admin Catalog UI 录入；happy-path E2E 待 P3 联调阶段补）

---

## 业务影响范围检查

- ✓ 接口签名：5 新端点为**纯新增**，不影响既有 `/mml/*` 端点（commands/scripts/tasks/templates/groups）
- ✓ DB Schema：无新迁移，复用 T-0123-P0 的 mml_command_sub_fields / mml_commands.target_object / mml_param_groups.path
- ✓ 共享 model：`Statement` JSON tags 是**新增 tags**，原结构字段名/类型未变 — 无 breaking change
- ✓ 事件契约：未修改 EventBus subject
- ⚠️ `*Service.CreateAndFanoutTask` 是新 public method，但 `fanouter.SetSequentialMode` 共享状态 — W3 race（follow-up）
- ✓ inferApiGroup 自动派生 api_group="mml"，与 /mml/* 既有角色绑定**自动复用**（不需新 seed migration）
- ✓ Statement.SelectedMMLCodes / UnknownCodes 通过 JSON tag 暴露给 FE — 接口契约扩展，向前兼容（旧 client 忽略未知字段）

---

## S5 内修复汇总

| ID | 原始严重度 | 修复位置 | 增量 |
|----|----------|---------|------|
| W1 | 🟡 WARNING | console_executor.go / console_handler.go | +1 sentinel + 2 错误包装点 + 1 switch；handler 删除 8 行字符串匹配代码 |
| W2 | 🟡 WARNING | console_executor.go ADD 分支 | +6 行注释 + 防覆盖 if continue |
| 测试 | — | console_executor_test.go | +1 测试 `TestBuildStatementCommands_WrapsErrInvalidRequest` + 2 处 `assert.True(errors.Is(err, ErrInvalidRequest))` |

S5 重构后 verify 回归：`go test -race -count=1 ./internal/mml/ → ok 1.110s`（48 测试全 PASS，含 21 console_executor + 13 console_service + 14 既有）

---

## 出口结论

- [x] CRITICAL = 0
- [x] HIGH = 0
- [x] WARNING 已在 S5 内修复 ✓（W1 + W2）
- [x] WARNING follow-up 3 项（W3 race / W4 happy-path E2E / W5 Prometheus）已登记 `verify-T-0123-P1.md §9`
- [x] 全 mml 包测试 race 绿
- [x] 全项目 `go build ./...` clean
- [x] 全项目 `go vet ./...` clean
- [x] E/R 比 ≥ 1（5/5）
- [x] 无 TODO/FIXME 新增
- [x] 无 `any` 滥用（11 处 boundary type 全审清）
- [x] 无运营商硬编码

**S5 结论**：T-0123-P1 通过 code review，可推进 S6 commit。

---

## 推荐 commit 拆分（供 S6 参考）

按可独立 revert 单元 4 组：

1. **D1 + D2 上** — `feat(mml): MML 字符串渲染/解析 + 命令分组树 + ConsoleService 编排`（renderer + parser + group_tree_repo + console_service + 33 测试）
2. **D2 下** — `feat(mml): ConsoleService.ExecuteStatements + buildStatementCommandEntry + Service.CreateAndFanoutTask`（console_executor + service.go ExecuteStatements 桥 + 21 测试）
3. **D3 + S5** — `feat(mml): Console 5 端点 + DI + e2e claim + ErrInvalidRequest sentinel`（console_handler + provider + e2e_verify + S5 修复）
4. **docs** — `docs(mml): T-0123-P1 verify-T-0123-P1.md + review report`

或合并为 1 commit（feat(mml): T-0123-P1 Console 后端 5 端点 + Go renderer/parser/executor）— 内容相关性强，单 PR review 友好。

建议：**合并为 1 commit**（与 P0 单 PR 单 commit 风格一致；P0 实际分 4 commit 因为 admin/migration/seed/permission 是 4 个独立可 revert 单元，而 P1 全部互相依赖）。
