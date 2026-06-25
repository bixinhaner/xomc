# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-06-25 16:10 |
| 提交 | `ea707eaf` (主改动) + `3d45a4a1` (审查补丁) |
| 作者 | shangyingbin |
| 范围 | pm-adhoc |
| 变更文件数 | 3 (主) + 2 (补丁) |
| 主改动行数 | +273 / -10 |
| 审查补丁行数 | +339 / -0 |

## 变更概要

在 `pm/adhoc/handler.go` 给 6 个接口（编辑、取消、硬删、读结果、筛选选项、运行历史）补齐"自建任务归属权校验"：当前登录用户必须是任务创建者或超级管理员才能操作；内置任务保持公共可读语义。修复 OWASP A01 IDOR 安全洞：之前知道任务 id 的普通用户可越权操作他人任务。

主改动 commit `ea707eaf` 覆盖了写接口（Update/Cancel/Delete）的 403/200 单元测试；本次审查发现读接口（Results/FilterOptions/Runs）的归属权单元测试缺失，已在补丁 commit `3d45a4a1` 中补齐，并把会话期间编写的运行栈验证脚本一并入仓供后续复用。

## 审查发现

### 🔴 CRITICAL (严重)

无。

### 🟡 WARNING (警告)

无（审查补丁 `3d45a4a1` 已修复 Results/FilterOptions/Runs 单元测试覆盖缺口）。

### 🔵 INFO (建议)

1. **handler 内权限校验模板复制 5 处**：
   "取任务 → 判 IsBuiltin → 归属权检查 → 403 + permission denied: not task owner" 这套 5 行模板在 6 个 handler 里复制了 5 份。可接受（OMC 项目中类似模板复用很常见），但如果未来还要加 PM adhoc 新接口，容易漏权限校验。后续可考虑抽出 `requireTaskOwnerOrAdmin(c, task)` 辅助函数。

2. **`isAdmin()` 的 role-name 旁路语义需要长期跟踪**：
   现有 `handler.isAdmin()` 除了看 `is_super_admin`（由 `users.source='builtIn'` 派生）外，**还把角色名 == "admin" 或 "super_admin" 视为超管**。这意味着：分配了内置 admin role 的非内置用户也会被归属权校验旁路放行。当前 OMC 设计里这是预期的（admin role 就是相当于超管角色），但如果未来产品上允许"自定义角色叫 admin 但不享受超管旁路"，这条逻辑需要重审。**本次不在改动范围内**，仅作知识传承记录。

3. **私有函数 `canOperate` / `canViewResults` 无独立单元测试**：
   只通过 handler 测试间接覆盖。这两个函数逻辑很简单（短路 + 字段比较），通过 handler 测覆盖足够，无需为简单函数额外加直接测试。

## 详细分析

### `omcgo/internal/pm/adhoc/handler.go`

- 5 个写/读 handler 在 `repo.Get` 之后立即加入归属权校验，先于业务逻辑短路，符合"防御靠前"的安全实践。
- `canOperate` / `canViewResults` 两个判定函数语义清晰：
  - `canOperate(task, user, admin)`：超管短路 → 创建者匹配；用于编辑/取消/硬删。
  - `canViewResults(task, user, admin)`：内置任务短路 → 超管短路 → 创建者匹配；用于读结果/筛选选项/运行历史。两者差异（内置短路）正确体现"内置任务是公共资产"的业务语义。
- Runs handler 把原来"仅校验存在"的 `if _, err := h.repo.Get(...)` 改成 `task, err := h.repo.Get(...)` 取实体，复用变量名清晰。

### `omcgo/internal/pm/adhoc/handler_test.go`

主改动 6 个新用例覆盖 Update/Cancel/Delete 的 403/200 + 内置任务跳过路径，断言点明确。
审查补丁 `3d45a4a1` 新增 4 个：3 个 NonOwner Forbidden + 1 个内置任务权限放行用例。其中内置任务用例因 `pool=nil` 后续会在 SQL 段 panic，用 `defer recover` 屏蔽 panic、断言 `w.Code != 403` 来证明"权限层未拦"，注释明确说明了这个测试技巧的取舍。

### `omcgo/internal/pm/adhoc/handler_validation_test.go`

4 处既有用例补 `Creator: "anonymous"` 字段。原因：新加的归属权校验通过 `extractCreator(c)` 取 `username`（中间件未注入时默认 "anonymous"），既有 Update 测试用的是无中间件 `newTestRouter`，所以 ctx 取出 username 为空 → `canOperate` 比较 `task.Creator == ""` 为假；故让 stub 返回的 task `Creator: "anonymous"` 匹配上下文。改动正确，最小侵入。

### `scripts/verify_652_adhoc_permission.sh`

运行栈三段验证脚本（API + 日志，无前端段）。涵盖：
- admin 加密登录（RSA-OAEP）流程
- pg 直插临时用户 alice + 临时角色 pm-tester-652（避开 admin role 的超管旁路）
- Casbin policy reload（重启 app）让新插的 user/role 生效
- 14 个 curl 断言：6 个负向 + admin 超管放行 + 内置全员可读 + alice 操作自己任务

脚本幂等（每次清理重建）+ 自动清理产物。**对未来涉及 RBAC + 业务归属权双层校验的 issue 有复用价值**（OMC 里"两个会话同时 reload Casbin"和"用户身份的 admin 角色 vs source=builtIn 双重判定"这两个坑很容易踩）。

## 业务完整性检查

业务链路完整，无遗漏：
- 改动仅在 handler 层叠加守门，不需新建 service/repository 方法，不涉及 schema 变更。
- 路由注册无变化（沿用既有 6 个端点）。
- 错误响应统一走 `response.Fail(c, http.StatusForbidden, "permission denied: not task owner")`，前端可识别 msg 串。
- 单元测试 + 运行栈测试双覆盖。

## 业务影响范围检查

变更范围可控，仅本模块 `internal/pm/adhoc/`：
- 无跨模块引用变更。
- 无对外接口签名变化（Update/Cancel/Delete/Results/FilterOptions/Runs 端点路径、请求体、响应字段全部不变）。
- 仅响应**码** + 响应 msg 在"非 owner"场景从 200 改为 403——是行为修复（之前是越权的 bug），不算 breaking change，但**前端可能需要识别 403 给友好提示**（INFO 级别提醒，不阻塞合并）。

## 前后端一致性检查

本次变更仅涉及后端。方案文件 `notes/tmp/pm-adhoc-permission-control.md` 明确说明前端无需改：列表已按创建者过滤，普通用户的任务列表里不会出现他人任务，不存在"看到但操作失败"的体验缺口。

建议关注的对应前端同步需求（INFO）：
- 错误码 `403 + msg="permission denied: not task owner"` 前端可识别并给业务文案提示（避免直接抛技术 msg）；不强制。

## 代码质量回退检查

未发现代码质量回退：
- 无删除测试用例；既有 4 处测试补 Creator 字段是为新逻辑适配，非弱化。
- 无移除 error check。
- 无降级安全措施（**反过来：本次是正向安全收益**）。
- 无引入 any / interface{}。
- 无 SQL 拼接。
- 无 ORM 引入。
- 无硬编码替代配置（用户名、超管标志均从 gin ctx 取）。

## 配套更新提醒

- **文档**：方案文件 `notes/tmp/pm-adhoc-permission-control.md` 已写，是本任务设计源。`CLAUDE.md` / 模块 README 无需更新（这是行为修复，不是新功能/新概念）。
- **单元测试**：审查补丁 `3d45a4a1` 已补齐读接口测试覆盖。
- **端到端测试**：`scripts/verify_652_adhoc_permission.sh` 已入仓，覆盖运行栈三段。`omcgo/scripts/e2e_verify.sh` 主流程不强制更新（adhoc 任务的归属权属于 RBAC 数据面，不在 e2e_verify.sh 的覆盖范围）。

## 安全检查

**正向安全收益（OWASP A01 IDOR 修复）**：
- 修复前：普通用户知道任务 id 即可对他人自建任务发起编辑/取消/删除/读结果/读筛选/读运行历史。属于 Insecure Direct Object Reference（OWASP Top 10 A01 - Broken Access Control）。
- 修复后：归属权 + 超管 + 内置任务三档授权策略，覆盖所有暴露 task id 的操作端点。
- 未引入新安全风险：所有错误响应都走统一 `response.Fail`，无敏感数据泄露；ctx 取值带类型断言，零容忍式失败默认（取不到视为 false / "anonymous"）。

## 性能检查

可忽略的额外开销：
- Cancel / Delete 各新增 1 次 `repo.Get(id)` 主键查询（之前直接走 Cancel/Delete，无 Get）。单次主键查询微秒级，可忽略。
- Update / Results / FilterOptions / Runs 本来就已有 `repo.Get`，不增加查询。

## 测试覆盖

合并审查补丁后，6 个端点的归属权各路径都有单元测试或运行栈测试：

| 端点 | 单元测试 | 运行栈 |
|------|----------|--------|
| Update | NonOwner 403 + Admin 200 + 内置跳过 ✓ | ✓ |
| Cancel | NonOwner 403 + Owner 200 + 内置跳过 ✓ | ✓ |
| Delete | NonOwner 403 ✓（owner/内置由既有用例覆盖） | ✓ |
| Results | NonOwner 403 + 内置放行 ✓（补丁 `3d45a4a1`） | ✓ |
| FilterOptions | NonOwner 403 ✓（补丁 `3d45a4a1`） | ✓ |
| Runs | NonOwner 403 ✓（补丁 `3d45a4a1`） | ✓ |

`go test ./internal/pm/adhoc/...` 全过。

## 总结

**结论：可合并**。无 CRITICAL 与 WARNING，3 条 INFO 均为长期可优化项。

- 安全收益：修补一个明确的 IDOR 漏洞（OWASP A01）。
- 改动范围：聚焦单模块单文件 + 测试，零跨模块影响。
- 质量：代码风格 / SQL / 错误处理 / 类型安全均符合项目规范。
- 测试：双层覆盖（单测 + 运行栈），都已 PASS。
