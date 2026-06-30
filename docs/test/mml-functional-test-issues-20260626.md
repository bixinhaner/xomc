# MML 模块功能测试问题清单

> **测试日期**：2026-06-26
> **测试人**：AI 自动化测试（GitHub Copilot）
> **测试范围**：MML 控制台 / MML 脚本任务 / MML 任务记录 / MML 私有命令 / MML 目录管理（admin/catalog）
> **测试环境**：docker compose 栈（web `:8081`）；登录 admin/admin123
> **设计文档基线**：
> - [mml-requirements-design.md](../design/mml-requirements-design.md)（v5.0，2026-04-21）
> - [mml-console-architecture-overview-20260521.md](../design/mml-console-architecture-overview-20260521.md)
> - [mml-console-redesign-20260603.md](../design/mml-console-redesign-20260603.md)（v2 三栏改版主推方案）
> - [mml-admin-catalog-redesign-20260527.md](../design/mml-admin-catalog-redesign-20260527.md)
> - [mml-task-flow-design-20260424.md](../design/mml-task-flow-design-20260424.md)
> - [mml-script-task-integration.md](../design/mml-script-task-integration.md)
> **环境数据**：测试前 `devices` 表为空，已临时插入 5 台测试设备（`MMLTEST-BAIBLQ-001..003` + `MMLTEST-BSC-001..002`），SQL 见 `/tmp/seed_mml_test_devices.sql`。

---

## 问题严重级定义

| 级别 | 含义 |
|------|------|
| **P0-阻断** | 核心功能完全无法使用，无 workaround |
| **P1-高** | 主流程功能错误或缺失，有 workaround 但影响业务 |
| **P2-中** | 非主流程问题、UX 缺陷、文案错误 |
| **P3-低** | 优化建议、可读性问题 |

---

## 问题总览（按严重级排序）

| 编号 | 级别 | 模块 | 一句话描述 |
|------|------|------|------------|
| BUG-01 | **P0** | 后端 / 任务派发 | 脚本任务命令字符串带分号 `;` 时 `resolveRPCMethods` 匹配失败 → fanout 跳过 → `device_tasks` 空，任务实际未下发 |
| BUG-02 | **P0** | 后端 / 审计 | `mml_audit_log` 表不存在，每次执行 MML 命令都在日志里堆 SQL 错误 + 完整堆栈 |
| BUG-03 | **P0** | 前端 / 设备弹窗 | DeviceSelectModal 初次空查后再点「搜索」不会重发请求（React Query 缓存未失效） |
| BUG-04 | **P0** | 前端 / 脚本管理 | 「新增脚本」对话框按 ESC 关闭也会创建脚本（数据无意残留） |
| BUG-05 | **P0** | 后端 / 设备心跳 | 心跳超时阈值 100s 过激进，新插入或临时改在线的设备 1~2 分钟即被标为离线 |
| BUG-06 | **P1** | 后端 / 任务关联 | 脚本任务创建后 `mml_tasks.script_id` 未回填，导致 `ResultAggregator` 无法回写 `mml_scripts.last_run_status` |
| BUG-07 | **P1** | 前端 / 数据一致性 | Console 表头使用中文显示名（用户友好名/DN前缀），任务记录详情却用英文 leaf（UserLabel/DnPrefix） |
| BUG-08 | **P1** | 前端 / 任务记录 | 任务记录详情头部显示命令码 `LST DEVICE_INFO`，应展示命令名「查询 设备基本信息」 |
| BUG-09 | **P1** | 字典数据 | `Device.DeviceInfo.UserLabel` 显示名落地为「用户友好名」，TR-181 标准与设计应为「用户标签」 |
| BUG-10 | **P2** | 前端 / 文案 | 执行详情对话框出现「LST 查询 查询 设备基本信息」，「查询」重复（操作类型 tag + 命令名前缀都渲染了"查询"） |
| BUG-11 | **P2** | 前端 / Tab | 切换 MML 子菜单后，顶部 Tab 高亮仍停留在旧 tab；URL 已变但 tab 状态/active key 未同步 |
| BUG-12 | **P2** | 前端 / 暗色主题 | 任务 ID `<code>` 在暗色主题下为灰底灰字，几乎不可读 |
| BUG-13 | **P2** | 前端 / 脚本管理 | 「新增脚本」保存后对话框不自动关闭，「保存」按钮持续 loading（实际后端已 201） |
| BUG-14 | **P2** | 前端 / Console | 命令历史只显示最后一次执行（执行 2 次只剩 1 条），疑似 SSE 完成帧丢失 |
| BUG-15 | **P3** | 后端 / 数据 | 5 台测试设备无任何告警却显示 `active_alarm_count=1, alarm_severity=critical` |
| BUG-16 | **P3** | 前端 / API 约定 | 列表响应字段不统一：部分返 `items`、部分返 `list`，造成前端 `mapBackend*` 易错 |
| BUG-17 | **P3** | 前端 / 私有命令 | 表头列宽过窄，列标题被 placeholder 文字挤压截断 |
| BUG-18 | **P3** | 前端 / 脚本执行 | 「选择脚本」实际是 textarea，未提供从已存脚本下拉复用的能力（误导命名） |
| BUG-19 | **P3** | 前端 / 脚本执行 | 「设备 SN」只能手输或回车，缺「从已注册设备选择」按钮，不利于多设备批量 |
| BUG-20 | **P3** | 后端 / SQL 日志 | SQL 错误 `params` 字段把 jsonb 参数 base64（`bnVsbA==`）输出，可读性差 |

---

## 一、MML 控制台（/mml/console）

### BUG-03【P0】DeviceSelectModal「搜索」按钮初次空查后不重发请求

**复现**：
1. 进 `/mml/console`，点「选择设备」按钮 → 弹窗首次打开，自动以默认筛选发起一次 `/devices` 请求，返回为空。
2. 修改 productClass 或 SN 关键字（甚至不改），点「搜索」按钮。
3. **预期**：浏览器 Network 看到一次 `/devices` 请求。
   **实际**：0 个新请求（已用 `window.fetch` 拦截器证实），表格依然为空。
4. 关闭并重新打开弹窗 → 自动 fetch 一次，恢复正常。

**根因猜测**：
- React Query 缓存策略：相同 query key 在 staleTime 内返回缓存；点击「搜索」时只调用 `doSearch()`（更新 state），若 state 未变 → query key 未变 → 不触发 refetch。
- 见 [DeviceSelectModal.tsx](../../omcmb/webcode/src/pages/mml/Console/components/DeviceSelectModal.tsx#L1-L160) 中 `filterParams` + 模块级 `lastProductId / lastProductClass` 缓存。

**影响**：用户体感是「点了搜索没反应」，必须重开弹窗。

**修复建议**：
- 「搜索」按钮触发 `queryClient.invalidateQueries(['devices', ...])` 强制 refetch，而非只更新 state。
- 或为搜索按钮加 `enabled: false` 模式，改为命令式 `refetch()` 调用。

---

### BUG-10【P2】执行详情对话框「LST 查询 查询 设备基本信息」重复「查询」

**复现**：
1. Console 选设备 + 选「查询 设备基本信息」命令 + 执行。
2. 点击结果表「查看详情」。
3. 详情 Modal 标题渲染为「LST 查询 查询 设备基本信息」。

**根因**：操作类型 tag（"LST"→"查询"）和命令名前缀已经包含「查询」，UI 又把两者拼一起，产生「查询 查询」叠词。

**修复建议**：渲染时若命令名已含操作类型描述前缀（查询/修改/新增），不再追加 tag 文案。

---

### BUG-12【P2】任务 ID `<code>` 在暗色主题下灰底灰字不可读

**复现**：暗色主题下 Console 命令记录 / 任务记录页面，所有任务 ID 显示为浅灰色文字 + 灰色背景的 `<code>`。

**修复建议**：`<code>` 在暗色主题下对前景色用 `var(--colorTextSecondary)` + 背景 `var(--colorFillTertiary)`，与 AntD 暗色 token 对齐。

---

### BUG-14【P2】Console 命令历史漏帧（执行 2 次只显示 1 条）

**复现**：
1. Console 对同一设备连续执行同一命令 2 次（前一次先完成）。
2. 「命令记录」面板展开，**只能看到 1 条记录**（最新或最早，行为不稳定）。

**根因猜测**：与历史 BUG-1（SSE 完成帧丢失）同源，`run_complete` 帧若与下一次 `run_start` 帧靠得太近，前一次记录被覆盖。

**关联文档**：见 [mml-console-redesign-20260603.md](../design/mml-console-redesign-20260603.md) §4.3 命令历史持久化要求。

---

## 二、MML 脚本任务（/mml/script）

### BUG-01【P0】脚本任务命令字符串带分号 `;` 时无法解析 → 任务实际未下发

**复现**：
1. 进 `/mml/script`，新增脚本「查询设备基本信息脚本-test1」，内容写：
   ```
   LST DEVICE_INFO;
   ```
2. 点「执行」→ 填设备 SN `MMLTEST-BAIBLQ-001` → 立即执行 → 确认。
3. UI 提示创建成功（HTTP 201），`mml_tasks` 表新增 1 行（status=`pending`, total_devices=1）。
4. **预期**：device_tasks 表新增 1 条 RPC（method=GetParameterValues），向 ACS 派发。
   **实际**：`device_tasks` 表完全为空，App 日志出现：
   ```
   {"level":"warn","logger":"mml-fanout","msg":"skip command without rpc_method",
    "mml_task_id":"a6d3e218-…","cmd_idx":0,"command_code":"LST DEVICE_INFO;"}
   ```
5. 查 `mml_tasks.commands` JSON：`[{"command_code": "LST DEVICE_INFO;"}]` — 只有 command_code 没有 rpc_method。
6. 查 `mml_commands` 字典：`command_code='LST DEVICE_INFO'`（**不带分号**），`rpc_method='GetParameterValues'` 完整。

**根因**：[omcgo/internal/mml/service.go:1100-1132](../../omcgo/internal/mml/service.go#L1100-L1132) `resolveRPCMethods` 用 `code` 整串和按空格/冒号截首段两个候选去查字典，但**没有 trim 尾部 `;`**。脚本入口的 `command_code` 携带分号（前端 `mmlApi.createTask` 直接把脚本每行字符串映射成 `{command_code: line}`），字典两次查找都 miss → `rpc_method` 留空 → fanout `cmd_idx=0` 命令被 skip → 该设备的 device_task 不创建。

**用户体感**：脚本任务永远「pending」或自动滚到「failed/0 成功」，但 UI 不会告知"命令解析失败"。

**修复建议（最小改动）**：在 [resolveRPCMethods](../../omcgo/internal/mml/service.go#L1100) 取 candidates 之前做：
```go
code := strings.TrimSpace(rawCode)
code = strings.TrimRight(code, ";")  // 兼容 "LST DEVICE_INFO;" 风格
code = strings.TrimSpace(code)
```
同时建议前端 `mmlApi.createTask` 在 commands 序列化前做一次 `cmd.trim().replace(/;+$/,'')`，二处兜底。

**附带 BUG-06 关联**：测试中 `mml_tasks.script_id` 列也为 NULL（应为 `50e2ba1d-...`），是因前端 createTask 把脚本内容当 `commands` 数组传，**不传 `script_id`**。详见 BUG-06。

---

### BUG-04【P0】「新增脚本」对话框按 ESC 关闭也会创建脚本

**复现**：
1. 进 `/mml/script` 点「+ 新增脚本」。
2. 填名称 `test1` / 描述 `test11` / 标签 `test111` / 内容 `LST DEVICE_INFO;`。
3. 点「保存」→ 后端 201 但对话框不关闭（见 BUG-13）。
4. 按 `Esc` 关闭对话框。
5. 回到列表，发现**多了一条 `test1` 脚本**。
6. 即使从未点过保存，只要在对话框里填了内容然后按 Esc，部分场景也会持久化（待复核）。

**根因猜测**：保存按钮的 onClick 实际已发请求，loading 期间用户按 Esc 关闭了 dialog，前端 state 不重置，下一次 react-query refetch 列表把脏数据带出来——但 BUG-13 说明保存按钮本身完成后没自动 close，意味着创建已经走完。本质上 dialog 没区分「未保存」和「已保存待关闭」。

**修复建议**：
- 保存成功后立即关闭对话框（修 BUG-13）。
- Esc 关闭时若处于 loading 状态，先弹确认或忽略。

---

### BUG-06【P1】脚本任务 `mml_tasks.script_id` 未回填

**复现**：通过脚本「执行」入口创建任务（前端走 `POST /mml/tasks`）→ DB `mml_tasks.script_id IS NULL`。

**根因**：[omcmb/frontend-core/src/services/api/mmlApi.ts:775-805](../../omcmb/frontend-core/src/services/api/mmlApi.ts#L775-L805) `createTask` 把脚本内容拆成 commands 数组传给后端，**未填 `script_id`**：
```ts
const payload = {
  task_name: data.taskName,
  script_id: data.scriptId,     // ← data.scriptId 实际 undefined
  commands: data.commands.map((cmd) => ({ command_code: cmd })),
  ...
};
```
但 service 层 [ExecuteCommand](../../omcgo/internal/mml/service.go#L688-L702) 仍支持 `req.ScriptID` 解析脚本 → 回填 `task.ScriptID`：
```go
if req.ScriptID != nil && *req.ScriptID != "" {
    sid, _ := uuid.Parse(*req.ScriptID)
    script, _ := s.scriptRepo.GetByID(ctx, sid)
    scriptID = &sid
    for _, line := range splitScriptLines(script.Content) { ... }
}
```

**影响**：
- `ResultAggregator.maybeUpdateScript` 看不到 `script_id` → 不能回写 `mml_scripts.last_run_status / last_run_at`。
- 「脚本任务执行历史」（`/mml/scripts/{scriptId}/runs`）查询会丢失这部分任务。

**修复建议**：前端「执行」Drawer 提交时把 `scriptId` 一并传后端，由后端 `splitScriptLines` 解析；前端不需要本地拆 commands。

---

### BUG-13【P2】「新增脚本」保存成功后对话框不自动关闭，按钮持续 loading

**复现**：见 BUG-04 复现步骤 3。后端返回 201，table 已 refetch 出新行，但 dialog 内的「保存」按钮仍 spinning。

**修复建议**：`onSuccess: () => { closeModal(); refetch(); }`。

---

### BUG-18【P3】「选择脚本」字段实为 textarea，命名误导

「新建MML脚本任务」Drawer 中第三块「选择脚本」打开 textarea 显示 `LST DEVICE_INFO;`，用户可能误以为是下拉「从已有脚本选一份模板」。建议改名「脚本内容」或同时提供下拉「从已有脚本载入」。

---

### BUG-19【P3】「设备 SN」无「从已注册设备选择」入口

Drawer 中设备 SN 输入只能手敲/逗号粘贴/回车确认，不像 Console 那样有「+ 选择设备」按钮调起 DeviceSelectModal。批量执行场景下用户须从设备管理页复制 SN，体验差。

---

## 三、MML 任务记录（/mml/task-records）

### BUG-07【P1】Console 列名与任务记录详情列名不一致（中文 vs 英文 leaf）

**现象**：
- Console 执行结果表：「用户友好名 / DN前缀 / 制造商 / 设备型号 ...」（中文显示名）。
- 任务记录页详情 Modal：「UserLabel / DnPrefix / Manufacturer / ModelName ...」（直接渲染 TR-181 leaf）。

**预期**：同一份数据在两处应有一致的列标题，且优先用 i18n 显示名。

**修复建议**：任务记录详情 Modal 复用 Console 的列定义 / mapBackend 逻辑。

---

### BUG-08【P1】任务记录详情头部用命令码而非命令名

**现象**：详情 Dialog 顶部展示「· LST DEVICE_INFO」（command_code），应展示「查询 设备基本信息」（command_name）。

**修复**：从 `mml_commands` 字典联表取 `command_name`，或前端用全局 dict 做 code → name 映射。

---

## 四、MML 目录管理（/mml/admin/catalog，admin 角色）

### 总体验收（与 [mml-admin-catalog-redesign-20260527.md](../design/mml-admin-catalog-redesign-20260527.md) §R1-R6 对照）

| 要求 | 验收 |
|------|------|
| R1 一级分组（取消多级嵌套） | ✅ 通过：22 个分组扁平展示，「自定义命令」位列末尾 |
| R2 分组卡片右侧 `…` 菜单（编辑/新增命令/删除） | ✅ 通过 |
| R3 命令行内 `edit/delete` 图标 | ✅ 通过 |
| R4 右栏命令元数据卡（命令编码/逻辑码/操作类型/显示名/source/需二次确认） | ✅ 通过 |
| R5 PATH 列表（MML Code / 显示名 / 标准路径） | ✅ 通过，「+ 批量添加」按钮可见 |
| R6 搜索框支持「分组 / 命令 / path」混合搜索 | 待验证 |

### BUG-09【P1】字典 description 落地为「用户友好名」，与 TR-181 标准/中文显示偏差

**现象**：在 catalog 「查询 设备基本信息」的 PATH 列表里，`Device.DeviceInfo.UserLabel` 显示名为「用户友好名」。
- TR-181 原文："UserLabel — A user-friendly label for this device"（**用户友好的标签**）。
- 国内运营商规范多译为「用户标签」或「设备别名」。

**影响**：Console / 任务记录的列头都跟随这份字典，所有界面展示都不规范。

**修复**：审查 `omcgo/data/param-mappings/*.json` 或对应 i18n 字典，把所有 leaf 的 `display_name_zh` 走一遍中文标准译名。

---

## 五、MML 私有命令（/mml/private-command）

### BUG-17【P3】表格列宽过窄，列标题被 placeholder 干扰截断

**现象**：列标题渲染为「命令编码（命令编码必须自定义，且全局唯」（被截断）+「 操作类型 描述 ...」叠在一起。

**修复**：明确每列 `width`；列标题与 placeholder 提示文案解耦。

---

## 六、跨页面公共问题

### BUG-02【P0】后端 `mml_audit_log` 表不存在，每次执行都报 SQL 错误

**现象**：所有 MML 执行（Console 直接执行 / 脚本任务 / 私有命令）路径都会触发：
```
ERROR: relation "mml_audit_log" does not exist (SQLSTATE 42P01)
```
完整堆栈见 `omc-app-1` 容器日志，每次执行后追加 ~40 行 stacktrace（污染日志，触发告警噪音）。

**根因**：[omcgo/internal/mml/pg_repository.go:1805](../../omcgo/internal/mml/pg_repository.go#L1805) `PgAuditRepository.CreateBatch` INSERT 到 `mml_audit_log` 表，但当前 schema 未创建该表（`\dt mml*` 输出无此表，只有 `mml_catalog_link_health/mml_command_groups/mml_command_sub_*/mml_commands/mml_custom_command*/mml_param_versions/mml_scripts/mml_tasks` 11 张）。

**修复建议**：
- 短期：补 migration 创建 `mml_audit_log` 表（参考代码里 INSERT 列定义 `(task_id, command_code, operation_type, device_sn, parameters, param_paths, result_status, result_message, creator, duration_ms)`）。
- 或：临时短路 `writeAuditLogs` 在 audit table 未就绪时直接 return，避免污染日志。

---

### BUG-05【P0】心跳超时阈值 100s 过激进

**现象**：[omcgo/internal/device/status_reconciler.go](../../omcgo/internal/device/status_reconciler.go) 默认阈值 ENB=100s / CPE=600s。
- 本次测试 5 台基站类（FAP）测试设备，每隔约 90s 必须 `UPDATE devices SET is_online=true, last_inform_at=NOW()` 才能避免被刷成 offline。
- 真实场景里，3GPP TS 32.583 推荐 inform_interval 默认 1800s（30 分钟），CMCC 规范也是 5-30 分钟级别 —— **100s 阈值与现实严重不匹配**。

**修复建议**：
- 阈值应从 `inform_interval × 3` 动态计算（参考 SNMP MIB 实践 = 3 个采样周期）。
- 或允许按产品/运营商配置：CMCC LTE 基站可设 5 分钟、5G NR 设 10 分钟。
- 短期：把 ENB 阈值至少调到 600s（与 CPE 一致）。

---

### BUG-11【P2】Tab 状态与 URL 不同步

**现象**：在「脚本任务」tab 高亮的情况下点击侧边栏「MML 配置」（admin/catalog）或「私有命令」（/mml/private-command），URL 已切换、main 内容已渲染新页面，但 tab bar 仍高亮「脚本任务」，没有自动新开 tab 或切换 active key。

**修复建议**：路由变化时 dispatch 一个 tab 同步 action（与现有 tab 持久化逻辑对齐）。

---

### BUG-15【P3】测试设备显示虚假告警状态

**现象**：5 台 SQL 直插的测试设备（无任何告警数据）在 Console、设备弹窗、设备管理列表均显示「⚠ critical / 告警 1」。

**根因待查**：可能是设备列表 API 在 alarm 聚合 join 时 `LEFT JOIN ON serial_number` 命中了 NULL 但 COALESCE 默认 1；或 status 列计算逻辑有 fallback 强制设 critical。

---

### BUG-16【P3】API 字段命名不统一（`items` vs `list`）

**现象**：
- `/api/v1/mml/scripts` 返回 `{items, total, page, page_size}`。
- 部分老 API（例如 `/api/v1/topology/*`）仍返 `{list, total}`。

**影响**：前端 `mapBackend*` 函数易写错；新人入手心智成本高。

**修复**：约定统一为 `items`（已是 mml/device 主流），文档 ADR 立约束。

---

### BUG-20【P3】SQL 错误日志可读性差

`omc-app-1` 日志里 SQL 错误把 `params` 字段做了 base64：
```
"params":["a6d3e218-…","LST DEVICE_INFO;","","MMLTEST-BAIBLQ-001",
          "bnVsbA==", "bnVsbA==", "pending","","admin",null]
```
`bnVsbA==` 是 `"null"` 的 base64，运维排查时还要手解码。建议把 jsonb 类型直接 raw 输出（保留原文）。

---

## 测试附录

### A. 测试设备种子 SQL（`/tmp/seed_mml_test_devices.sql`）

5 台 FAP 类型测试设备，覆盖 BAIBLQ（4G）+ MyBSC100/200（5G NR）：
```sql
INSERT INTO devices (serial_number, product_id, ..., is_online, last_inform_at) VALUES
  ('MMLTEST-BAIBLQ-001', 'b6e57263-...', ..., true, NOW()),
  ('MMLTEST-BAIBLQ-002', 'b6e57263-...', ..., true, NOW()),
  ('MMLTEST-BAIBLQ-003', 'b6e57263-...', ..., true, NOW()),
  ('MMLTEST-BSC-001',    '6c188387-...', ..., true, NOW()),
  ('MMLTEST-BSC-002',    '6c188387-...', ..., true, NOW());
```

### B. 模拟任务完成 SQL（`/tmp/mock_task_complete.sql`）

由于测试设备 IP（10.20.30.x）无真实 CPE，ACS Connection Request 必失败 → device_tasks 永远 pending。本测试人为 UPDATE 一条 completed + 一条 failed 来验证结果 UI：
- `3cec55fe-...` → completed + 17 个 path 的 GPV 响应（mock Baicells/BaiBLQ_5.0.16.1）
- `48a97214-...` → failed + 9001 `[Server] Device not connected` + 9005 fault on UserLabel

### C. 已确认正常工作的功能（无需修复）

| 功能 | 状态 |
|------|------|
| Console 设备选择弹窗（多种过滤、单选/多选） | ✅ 基本可用，除 BUG-03 |
| Console 命令选择树（分组展开/搜索） | ✅ 正常 |
| Console 参数配置面板（动态字段、必填校验） | ✅ 正常 |
| Console 执行按钮 → 触发 mml_tasks 创建 | ✅ 正常 |
| Console 结果表 17 列动态渲染 | ✅ 正常 |
| Console「查看详情」Dialog | ✅ 渲染数据正确，除 BUG-10 |
| Console 单设备结果 CSV 下载 | ✅ 触发 success toast |
| 命令记录面板折叠/展开 | ✅ 正常（除 BUG-14 漏帧） |
| 任务记录列表分页 | ✅ 正常 |
| 任务记录详情 Modal | ✅ 渲染正确，除 BUG-07/08 |
| 脚本任务列表（信息/执行/编辑/删除 按钮） | ✅ 正常 |
| 脚本「新增脚本」表单 4 字段（名称/描述/标签/内容） | ✅ 正常（除 BUG-13/04） |
| 脚本「执行」Drawer 自动生成任务名 + 4 种执行方式 + 重试策略 | ✅ 字段完整（除 BUG-18/19 UX） |
| 私有命令列表 | ✅ 表格能渲染（除 BUG-17 列宽） |
| 目录管理：分组展开、命令查看、PATH 列表渲染 | ✅ 设计 R1-R5 全部通过 |

