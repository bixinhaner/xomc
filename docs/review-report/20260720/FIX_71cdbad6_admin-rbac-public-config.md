# 管理接口 RBAC 与系统配置敏感信息 P0 修复报告

## 1. 执行信息

- 基线提交：`71cdbad69835afb9f200845a976057aaa9f1f130`
- 修复分支：`codex/fix-system-config-security-p0`
- 开发目录：`/Users/hezg/Documents/work/xomc`（当前 VSCode 工作区直接切换修复分支）
- 修复范围：管理接口端点级 RBAC、公开配置白名单、默认密码与 Agent Token 脱敏和写入保护、前端 write-only 交互
- 未执行：提交、推送、数据库 schema 迁移、Secret Manager 接入

## 2. P0 结论

| 问题 | 修复前 | 修复后 | 状态 |
|---|---|---|---|
| P0-01 管理接口只认证、不执行端点权限 | 普通已登录用户可绕过 `role_api_permissions`；权限组件异常时存在放行路径 | 使用 Gin 标准路由模板和 HTTP 方法调用现有 Casbin；deny=403，checker/路由/校验异常=500；仅 builtIn 超管旁路 | 已关闭 |
| P0-02 通用配置可扩大匿名公开范围并回显敏感值 | 匿名输出信任数据库 `is_public`；管理配置 API 直接序列化默认密码和 Agent Token | 匿名输出只认代码白名单；所有配置 HTTP 响应统一安全 DTO；敏感值只返回 `is_secret/is_configured` | 已关闭（HTTP 层） |

## 3. 修改清单

| 编号 | 修改 | 作用 | 主要文件 |
|---|---|---|---|
| M-01 | 恢复 `RequireAPIPermission` 端点级校验 | 用 `(FullPath, HTTP Method)` 执行 Casbin，全部异常 fail-closed | `omcgo/internal/admin/middleware.go` |
| M-02 | 删除旧 fail-open 权限链 | 删除 `RequireApiPermission`、`ApiPermissionChecker`、`RoleApiEndpoint` 和无调用的角色端点查询 | `middleware.go`、`repository.go`、`pg_role_repository.go` |
| M-03 | 修正路由装配注释 | 明确受保护路由组执行端点级 RBAC，不再描述为“仅认证” | `omcgo/cmd/app/provider/router.go` |
| M-04 | 建立公开配置精确白名单 | 匿名接口不再信任数据库历史或客户端写入的 `is_public` | `omcgo/internal/admin/sys_config_policy.go`、`sys_config.go` |
| M-05 | public 标记改为服务器派生 | Create、Update、BatchUpsert 均按 `(category,key)` 重新计算；批量更新会纠正触达行的历史脏标记 | `omcgo/internal/admin/sys_config.go` |
| M-06 | 拒绝客户端控制 `is_public` | Create/Update 显式携带 `is_public` 返回 HTTP 400 | `sys_config.go`、`sys_config_handler.go` |
| M-07 | 建立敏感配置注册表 | 登记 `security.defaultPasswd`、`agent.agent_studio_service_token` | `sys_config_policy.go` |
| M-08 | 统一安全响应 DTO | List、Get、Create、Update、ListPublic 均输出 `SysConfigResponse`；敏感 value 固定为空 | `sys_config_policy.go`、`sys_config_handler.go` |
| M-09 | 增加敏感写入守卫 | direct Create/Update/Delete 禁止 secret；通用 batch 禁止 Agent Token、禁止空默认密码；Agent 专用服务内部写入保持兼容 | `sys_config.go`、`sys_config_handler.go` |
| M-10 | 前端接入 secret 元数据 | `is_secret/is_configured` 映射为 `isSecret/isConfigured` | `frontend-core/src/types/system.ts`、`adminApi.ts` |
| M-11 | 默认密码改为 write-only | 删除前端硬编码初始密码；不灌回敏感值；密码输入无可见性切换；按配置状态显示占位提示 | `SystemConfig/index.tsx`、`SecuritySettings.tsx`、i18n |
| M-12 | 保存时跳过空敏感值 | 已配置或全新库尚无记录时，默认密码留空保存其他安全项都不生成覆盖项；输入新密码时正常提交 | `sysConfigSerialize.ts` |
| M-13 | 保持用户管理默认密码功能 | “使用系统默认密码”依据刷新后的 `isConfigured` 启用；密码仍由后端读取，前端不显示原文，避免弹窗使用旧缓存状态 | `useSecuritySettings.ts`、`UserManagement/index.tsx` |
| M-14 | 补齐自动化测试与报告 | 覆盖 RBAC allow/deny/error/bypass、公开/敏感分类、脱敏、写守卫、HTTP 契约、前端映射和 serializer | 新增/修改测试文件及本报告 |

## 4. 策略清单

### 4.1 匿名公开白名单

1. `system.system_name`
2. `system.system_version`
3. `system.show_menu_icon`
4. `basic.mrOMCName`
5. `security.isBrowserAutoRecordPass`
6. `ui_custom.ui_login_background`
7. `ui_custom.ui_menu_logo_up`
8. `ui_custom.ui_menu_logo_down`

除此之外，即使数据库行为 `is_public=true`，匿名接口也不会返回。

### 4.2 敏感配置注册表

1. `security.defaultPasswd`
2. `agent.agent_studio_service_token`

HTTP 响应契约：

- `value: ""`
- `is_secret: true`
- `is_configured: true|false`

## 5. 测试证据

| 验证项 | 命令 | 结果 |
|---|---|---|
| RBAC RED | `go test ./internal/admin -run '^TestRequireAPIPermission' -count=1` | 修复前 deny/error/nil checker 错误返回 200，符合预期 RED |
| RBAC GREEN | 同上 | PASS |
| Public/Secret 策略 | `go test ./internal/admin -run 'Test(IsPublicSysConfig\|IsSecretSysConfig\|ToSysConfigResponse\|SysConfigService_ListPublic)' -count=1` | PASS |
| 写入守卫 | `go test ./internal/admin -run 'TestValidateGeneric\|TestSysConfigService_(Create\|Update)' -count=1` | PASS |
| HTTP 脱敏契约 | `go test ./internal/admin -run '^TestSysConfigHandler_' -count=1` | PASS；测试确认响应体不含原始默认密码或 Agent Token |
| 后端跨模块 | `go test ./internal/admin ./internal/agentconfig ./cmd/app/provider -count=1` | PASS |
| 后端全量构建 | `go build ./...` | PASS |
| 后端全量测试 | `go test ./...` | PASS |
| 前端映射/序列化 | `npx vitest run ...adminApi.test.ts ...sysConfigSerialize.test.ts` | 2 files、12 tests PASS |
| 前端类型检查 | `npm run typecheck` | PASS |
| 前端 scoped ESLint | `npx eslint ... --quiet` | PASS |
| 浏览器安全回归 | `npx playwright test e2e/security-policy.spec.ts --workers=1` | 5/5 PASS |
| 前端全量 Vitest 中途对比 | `npx vitest run --reporter=json` | 当次 1298 tests：1291 PASS、7 个历史失败；失败项均在修改前基线列表内，无本次新增失败 |
| 差异格式 | `git diff --check` | PASS |

独立 code review 首轮发现 3 个 Important：敏感配置仍可通用 DELETE、全新库空默认密码导致整批保存 400、重置密码弹窗使用刷新前状态。三项均已修复并补测试；复审结论为无剩余 Critical/Important。

基线说明：修改前前端全量 Vitest 已存在 10 个与本次系统配置安全修复无关的失败（7 个测试文件）；本次定向测试、类型检查、lint 和浏览器安全回归均通过，未把这些历史失败计为本次引入。

## 提交阻断项关闭记录

以下是 2026-07-20 提交前复验的实际命令结果；只有此表中的 PASS 可作为本轮提交前证据。

| 阻断项 | 修改 | 测试证据 | 状态 |
|---|---|---|---|
| 内置角色权限缺失 | 新增 seed `000002`，为 `admin`/`operator` 授予全部已登记 API，`viewer` 授予全部 GET API；使用幂等冲突处理 | 静态 migration 契约 PASS；真实 PostgreSQL 16 行为测试 PASS（事务回滚） | PASS |
| policy 写入后不立即生效 | `SetRoleApiEndpoints`、`AssignRole`、`RemoveRole` 均在事务提交后同步 reload，再通知 watcher；刷新错误显式返回 | `cd omcgo && go test ./internal/admin -run 'Test.*(Policy|RoleApi|Casbin|CheckPermission|Watcher)' -count=1`，PASS（约 2.72s） | PASS |
| Casbin 并发读写 | 使用 `SyncedEnforcer`，使 Enforce 与 policy reload 可安全并发 | `cd omcgo && go test -race ./internal/admin -run TestCasbinAuthorizerConcurrentEnforceAndReload -count=1`，PASS（约 5.36s） | PASS |

### 本轮命令证据

| 验证项 | 命令 | 结果 |
|---|---|---|
| 差异格式 | `git diff --check` | PASS（退出码 0、无输出） |
| 后端静态检查 | `cd omcgo && go vet ./...` | PASS（退出码 0，约 5.47s） |
| 后端构建 | `cd omcgo && go build ./...` | PASS（退出码 0，约 13.59s） |
| 后端全量测试 | `cd omcgo && go test ./...` | PASS（退出码 0；提权运行以允许 miniredis/httptest 本地监听；关键包均 PASS） |
| 前端类型检查 | `cd omcmb && npm run typecheck` | PASS（退出码 0，约 13.33s；仅有 npm `home` 配置弃用警告） |
| 前端构建 | `cd omcmb && npm run build --workspace webcode` | PASS（退出码 0，Vite production build 约 2.16s；仅有弃用和 chunk size 警告） |

### 非本次范围的既有问题

- 全量前端 lint：本轮未执行（任务明确不运行；历史错误保留）；
- 全量 Vitest：本轮未执行（任务明确不运行；历史失败保留）；
- AuditLogger 全量 race：本轮未执行全量 `-race`，该历史问题不应与本次 Casbin 定向 race PASS 混写；
- 安全设置页 `refetchConfiguredState` 未传播 refetch error：既有问题，未在本轮改动或复验。

## 6. 仍需后续处理的风险

### P1：敏感值仍以明文存储在 `sys_configs`

本次关闭的是 HTTP 回显、匿名公开和通用写入口风险，没有改变数据库存储方式。具有数据库读取权限、备份读取权限或 SQL 注入能力的主体仍可能看到默认密码和 Agent Token。建议后续使用应用层加密（密钥不与数据库同存）或外部 Secret Manager，并制定密钥轮换和备份恢复方案。

### P1：后端仍保留首次部署默认密码兜底常量

`internal/admin/security_policy.go` 仍保留 `defaultDefaultPassword = "OMC@123456"`。本次已删除前端硬编码和原文回显，但未改变既有后端首次部署兼容策略。建议单独评审是否改为“未显式配置即不可使用系统默认密码”，并同步调整初始化、升级和测试策略。

### 上线检查：角色端点授权完整性

权限恢复为 fail-closed 后，非 builtIn 用户必须拥有对应 `(path, method)` 授权。上线前应使用 builtIn 超管检查并同步 API endpoint，确认 admin/operator/viewer 的 `role_api_permissions` 完整，避免把原有权限缺口表现为 403。

### 数据治理：历史 `is_public` 脏标记

运行时已经忽略白名单外的 `is_public=true`，安全风险已关闭；数据库历史标记仍可能存在。建议后续补一次数据审计或迁移，将白名单外标记统一归零，减少运维误判。
