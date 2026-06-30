# MML 模块功能测试问题修复计划（草案）

> **依据**：[mml-functional-test-issues-20260626.md](./mml-functional-test-issues-20260626.md)（20 条 BUG，P0×5 / P1×4 / P2×5 / P3×6）
> **制定日期**：2026-06-26
> **状态**：📋 计划草案，**等待用户审核** —— 用户给出指令后再开始按批次实施
> **作者**：AI 自动化测试 + 修复（GitHub Copilot）

---

## 一、修复策略与原则

| 维度 | 策略 |
|------|------|
| **批次划分** | 6 个批次（PR 单位），每批次内 BUG 相互关联或文件重叠，便于一次提交 |
| **优先级顺序** | P0 → P1 → P2 → P3；批次内严格按 BUG 编号执行 |
| **风险控制** | 每个批次先写测试再改实现；批次间相互不阻塞，可独立合并 |
| **回滚边界** | 每个批次单独 commit / PR；migration 类（BUG-02）单独成 PR |
| **测试覆盖** | 后端改动必带 `_test.go`；前端必带 vitest + 手工浏览器复测 |
| **配置改动** | BUG-05（心跳阈值）走配置变更 + 默认值修改，不破坏既有部署 |

---

## 二、批次总览

| 批次 | 范围 | 包含 BUG | 影响面 | 风险 |
|------|------|----------|--------|------|
| **批次 1** | 后端关键：脚本任务派发链路 | BUG-01, BUG-06, BUG-02 | 后端 `internal/mml/` + 1 个 migration | 中（涉及 fanout 核心 + 新表） |
| **批次 2** | 前端关键：设备弹窗 + 脚本对话框 | BUG-03, BUG-04, BUG-13 | 前端 Console / Script 页 | 低 |
| **批次 3** | 配置：心跳超时阈值 | BUG-05 | 后端 `internal/device/status_reconciler.go` + 配置 | 中（误判风险） |
| **批次 4** | 数据展示一致性 | BUG-07, BUG-08, BUG-09 | 前端任务记录详情 + 字典数据 | 低 |
| **批次 5** | 体验改进 | BUG-10, BUG-11, BUG-12, BUG-14 | 前端文案/样式/Tab/SSE | 中（BUG-14 需排查 SSE） |
| **批次 6** | P3 优化建议 | BUG-15~BUG-20 | 多处零散 | 低（可作为长期 backlog 拆细执行） |

---

## 三、批次详细计划

### 批次 1：脚本任务派发链路修复（P0 后端）

**目标**：恢复脚本任务"创建 → device_tasks 派发 → ACS 下发"的完整链路，消除审计日志的 SQL 错误。

#### BUG-01 + BUG-06 合并修复（同一文件、同一 PR）

| 项 | 内容 |
|----|------|
| **改动文件** | [omcgo/internal/mml/service.go](../../omcgo/internal/mml/service.go) 行 1100-1132 (`resolveRPCMethods`) + 行 688-702 (`ExecuteCommand` script_id 分支) |
| **辅助改动** | [omcmb/frontend-core/src/services/api/mmlApi.ts](../../omcmb/frontend-core/src/services/api/mmlApi.ts) 行 775-805 (`createTask`) |
| **修复方案** | 1) `resolveRPCMethods` 增加分号 trim：`code := strings.TrimRight(strings.TrimSpace(rawCode), ";"); code = strings.TrimSpace(code)`<br>2) 前端 `createTask` 把脚本执行入口改为只传 `script_id`，**不再本地拆 commands**；让后端 `splitScriptLines` 统一解析<br>3) 后端确认 `task.ScriptID` 在 script_id 模式下被正确回填到 mml_tasks |
| **新增测试** | `service_test.go` 新增：<br>- `TestResolveRPCMethods_TrimsTrailingSemicolon`<br>- `TestExecuteCommand_ScriptID_PopulatesTaskScriptID` |
| **手工验证** | 重新执行测试设备 + `LST DEVICE_INFO;` 脚本 → 验证 `device_tasks` 表新增记录 + `mml_tasks.script_id` 非空 |
| **风险** | 后端 trim 是纯防御性改动，零风险；前端改动会影响"自定义命令组合脚本"场景，需保留 `commands` 入参的兼容路径 |
| **依赖** | 无 |

#### BUG-02 独立 migration

| 项 | 内容 |
|----|------|
| **改动文件** | 新增 `omcgo/migrations/0001NN_create_mml_audit_log.sql`（编号现查 `ls omcgo/migrations/`） |
| **参考字段** | [pg_repository.go:1805](../../omcgo/internal/mml/pg_repository.go#L1805) INSERT 列：`(task_id, command_code, operation_type, device_sn, parameters, param_paths, result_status, result_message, creator, duration_ms)` + 索引 `(task_id), (device_sn, created_at DESC)` |
| **修复方案** | 创建表 + 适当索引；保留 `audit_repo == nil` 短路兜底，让历史部署兼容 |
| **风险** | migration 写完必须先在 dev compose 跑通；生产部署需走 goose up |
| **依赖** | 独立，可与 BUG-01 并行 |

**批次 1 验收**：
- [ ] `go test ./internal/mml/...` 全绿
- [ ] Console 执行 + 脚本任务执行后 `omc-app-1` 日志 0 个 SQL ERROR
- [ ] `SELECT count(*) FROM device_tasks WHERE source_id IN (<新建脚本任务>)` ≥ 1
- [ ] `SELECT script_id FROM mml_tasks WHERE id = '<新建脚本任务>'` 非 NULL

---

### 批次 2：前端关键交互修复（P0 前端）

#### BUG-03 DeviceSelectModal 搜索按钮强制 refetch

| 项 | 内容 |
|----|------|
| **改动文件** | [omcmb/webcode/src/pages/mml/Console/components/DeviceSelectModal.tsx](../../omcmb/webcode/src/pages/mml/Console/components/DeviceSelectModal.tsx) |
| **修复方案** | 把 `useQuery` 改为 `enabled: false` + 命令式 `refetch()`；搜索按钮 onClick 时直接调 `refetch()`，绕过 React Query staleTime |
| **新增测试** | `DeviceSelectModal.test.tsx` 用 vitest mock fetch，验证「连续点搜索两次能发两次请求」 |
| **三皮肤铁律** | 同步检查 `omcmb/webcode-v2/` 与 `omcmb/webcode-v3/` 是否有相同组件；若有同步改 |
| **风险** | 切 enabled:false 后默认不再首屏自动 fetch，需在 useEffect onMount 显式 refetch 一次保持原行为 |

#### BUG-04 + BUG-13 合并修复

| 项 | 内容 |
|----|------|
| **改动文件** | `omcmb/webcode/src/pages/mml/Script/` 下的新增脚本对话框组件（具体路径修复时定位） |
| **修复方案** | 1) 保存 mutation `onSuccess` 中 `setOpen(false)` + `form.resetFields()` + `refetch`<br>2) Esc 关闭：若 mutation pending 中，禁用 Esc / 显示二次确认；若已发请求成功（onSuccess 已触发），同上自动关闭<br>3) 取消按钮的 onClick 不发请求，仅 close |
| **新增测试** | vitest 验证「点保存 → 成功后 dialog 自动 close + 表单清空」 |
| **风险** | 低；关注是否有用户主动按 Esc 又预期"放弃"的语义需求 |

**批次 2 验收**：
- [ ] 浏览器复测：搜索按钮可重复发请求；新增脚本 dialog 保存后自动关；按 Esc 不会偷偷创建脚本

---

### 批次 3：心跳超时阈值合理化（P0 配置）

#### BUG-05 心跳阈值

| 项 | 内容 |
|----|------|
| **改动文件** | [omcgo/internal/device/status_reconciler.go](../../omcgo/internal/device/status_reconciler.go) + 配置文件 `omcgo/cmd/app/etc/config.dev.yaml` 等 |
| **修复方案** | 1) 短期：ENB 阈值默认从 100s 调整为 600s（与 CPE 一致），CPE 保持 600s<br>2) 中期：阈值改为可配置 `device.status.{enb,cpe}_offline_threshold_seconds`，并在每次基站 Inform 时记录其声明的 `inform_interval`，动态算阈值 = `max(inform_interval×3, 默认值)`<br>3) 文档：在 `docs/runbook/` 写说明 |
| **新增测试** | `status_reconciler_test.go` 覆盖：单元 + 配置覆盖优先级 |
| **风险** | 阈值调大会延迟离线检测；建议先验证 CMCC 规范典型值再定 |
| **依赖** | 无 |
| **决策点** | ⚠️ **请用户确认：先用「短期硬编码 600s」还是直接做「中期动态阈值」**？后者工作量约为前者 3 倍 |

**批次 3 验收**：
- [ ] 实测：模拟设备 5 分钟不汇报，阈值内仍在线、阈值后转离线
- [ ] 监控面板告警噪音下降

---

### 批次 4：数据展示一致性（P1）

#### BUG-07 + BUG-08 合并修复（任务记录详情）

| 项 | 内容 |
|----|------|
| **改动文件** | `omcmb/webcode/src/pages/mml/TaskRecords/` 下的详情 Modal（具体路径修复时定位） + 可能需要 `frontend-core/src/services/api/mmlApi.ts` 增加联表查询 |
| **修复方案** | 1) 详情 Modal 复用 Console 的列定义（抽到 `frontend-core` 公共模块）<br>2) 详情头部用 `command_name`（从 `mml_commands` 字典联表取，或前端做 code→name 映射 hook） |
| **新增测试** | vitest 渲染断言：详情列头中文显示名、标题包含「查询 设备基本信息」 |

#### BUG-09 字典数据修正

| 项 | 内容 |
|----|------|
| **改动文件** | `omcgo/data/param-mappings/*.json`（具体文件查 `grep -r "UserLabel\|用户友好名" omcgo/data/`） |
| **修复方案** | 1) 全局批量替换「用户友好名 → 用户标签」<br>2) 同步检查其他 leaf 的中文译名是否符合 CMCC / TR-181 规范（建议运营商规范专家二次确认） |
| **三皮肤铁律** | 字典是后端数据，对前端三皮肤一视同仁 |
| **新增测试** | 添加 dictloader 单元测试，校验关键 leaf 中文名 |
| **风险** | 译名是业务决策；建议同步走 ADR 记录决策依据 |
| **依赖** | 完成后需重启 app 让 dictloader 重新加载（或 hot reload 支持） |

**批次 4 验收**：
- [ ] 浏览器：Console 与任务记录详情列头一致、命令名展示正确
- [ ] 抽样所有 PATH 字典中文译名 → 与 CMCC TR-069 规范文档对照

---

### 批次 5：体验改进（P2）

#### BUG-10 操作类型 + 命令名叠词

| 项 | 内容 |
|----|------|
| **改动文件** | Console 执行详情 Modal 渲染逻辑 |
| **修复方案** | 渲染前判断 `command_name` 是否以「查询/修改/新增/删除」开头；若是则不再追加操作类型 tag 文案，或反之 |

#### BUG-11 Tab 状态与 URL 不同步

| 项 | 内容 |
|----|------|
| **改动文件** | tab 持久化逻辑（具体路径用 grep 找 `tabsSlice` 或 `useTabs` hook） |
| **修复方案** | 在路由变化时通过 `useEffect(location.pathname)` 派发 tab activate；若该 path 未在 tab list 则新增 tab |
| **三皮肤铁律** | webcode-v2/v3 同步检查 |

#### BUG-12 暗色主题 `<code>` 不可读

| 项 | 内容 |
|----|------|
| **改动文件** | 全局 CSS 或 ConfigProvider theme token |
| **修复方案** | `<code>` 的前景 / 背景用 AntD 5 暗色 token（`var(--colorTextSecondary)` + `var(--colorFillTertiary)`） |

#### BUG-14 命令历史漏帧（SSE）

| 项 | 内容 |
|----|------|
| **改动文件** | 后端 SSE 推送 + 前端事件聚合 hook |
| **修复方案** | 1) 后端：`run_complete` 事件之前先 flush；持久化命令记录到 DB（设计文档已要求）<br>2) 前端：用 `Map<run_id, record>` 而非 `Array`，去重靠 run_id |
| **风险** | SSE 通道改动需关注与 reliability 模块的交互（断线重连） |
| **依赖** | 与 BUG-1（设计文档历史 BUG）原因同源，需先查 [mml-console-redesign-20260603.md](../design/mml-console-redesign-20260603.md) §4.3 确认 |

**批次 5 验收**：
- [ ] 浏览器：详情 Modal 标题不再叠词；切换 MML 子菜单 tab 高亮跟随；暗色主题 `<code>` 文字清晰；连续执行 5 次命令 → 命令记录显示 5 条

---

### 批次 6：P3 优化建议（拆细执行）

> 这批属于"非紧急但需要跟进"的清单，建议每条独立开 issue，分配到长期 backlog；可在迭代空档执行。

| BUG | 单项建议 |
|-----|---------|
| **BUG-15** 虚假告警 | 排查设备列表 API 的 alarm 聚合 SQL；写最小复现测试 |
| **BUG-16** items vs list 不统一 | 立 ADR 约定统一为 `items`；按模块逐步迁移；不阻塞当前需求 |
| **BUG-17** 私有命令列宽 | 明确各列 width；标题与 placeholder 解耦 |
| **BUG-18** 「选择脚本」命名 | 改名「脚本内容」；如有需求再加「从已有脚本载入」下拉 |
| **BUG-19** 设备 SN 选择 | 复用 DeviceSelectModal 组件 |
| **BUG-20** SQL 日志 base64 | 修改 `internal/core/components/postgres/tracer.go` 让 jsonb raw 输出 |

---

## 四、推荐执行顺序

```
日 D+1   批次 1（BUG-01/06/02）        ← 解封脚本任务实际下发链路
日 D+2   批次 2（BUG-03/04/13）        ← 解封 Console + Script 关键交互
日 D+3   批次 3（BUG-05）              ← 用户决策短期/中期方案后
日 D+4   批次 4（BUG-07/08/09）        ← 数据一致性 + 字典
日 D+5+  批次 5（BUG-10/11/12/14）     ← 体验优化
日 D+N   批次 6（P3 清单）             ← 拆细独立排期
```

> 上述节奏假设单人专职推进；并行执行可压缩 30~50%。**不在文档里给具体小时数**（OMC 项目规范）。

---

## 五、需要用户决策的点

请在批准修复前确认以下事项：

1. **修复范围**：是否全部 20 条都修，还是先做 P0+P1（9 条）？
2. **批次顺序**：是否按上述推荐顺序，还是有其他优先级（如先修 BUG-05 减少告警噪音）？
3. **BUG-05 方案**：短期硬编码 600s（快）/ 中期动态阈值（彻底）？
4. **BUG-09 字典**：是否需要在改之前先组织运营商规范专家审定中文译名？
5. **三皮肤铁律应用**：BUG-03/11/12/14 涉及前端组件，是否要求 v1/v2/v3 同步修改 + skin-parity 校验通过？（默认按 §8.3 必须）
6. **migration 流程**：BUG-02 的 migration 是否需要走「先写、CR、再合并」流程，还是按 hotfix 直接进 main？

---

## 六、附：每条 BUG 的「最小可验证修复」一句话总结

| BUG | 一句话 fix |
|-----|------------|
| 01 | `resolveRPCMethods` 加 `TrimRight(code, ";")` |
| 02 | 新建 migration 创 `mml_audit_log` 表 + 索引 |
| 03 | DeviceSelectModal 搜索按钮改命令式 `refetch()` |
| 04 | Dialog Esc 关闭 + 保存成功后状态机统一管理 |
| 05 | ENB 阈值 100s → 600s 短期；中期接 inform_interval×3 |
| 06 | 前端 createTask 传 `script_id` 不本地拆 commands |
| 07 | 任务记录详情列定义复用 Console 公共列 |
| 08 | 任务记录详情头部用 `command_name` 非 code |
| 09 | 字典批量改「用户友好名 → 用户标签」+ 同类审视 |
| 10 | 渲染前去重「查询 查询」叠词 |
| 11 | tab 监听路由变化派发 activate |
| 12 | `<code>` 用 AntD 5 暗色 token |
| 13 | 保存成功 onSuccess 立即关闭 dialog |
| 14 | SSE 用 `Map<run_id, record>` 去重 + 后端持久化 |
| 15 | 排查设备列表 alarm 聚合 SQL JOIN 逻辑 |
| 16 | ADR 约定 `items` 命名 |
| 17 | 给私有命令表格列设 width |
| 18 | 「选择脚本」改名「脚本内容」 |
| 19 | 复用 DeviceSelectModal 到脚本执行 Drawer |
| 20 | SQL tracer 改 jsonb raw 输出 |

---

**📌 待办：请回复"开始批次 N"或"全部按推荐顺序执行"或调整建议**
