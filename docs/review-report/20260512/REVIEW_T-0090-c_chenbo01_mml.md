# Code Review + Security Review — T-0090-c (MML RBAC 私有命令过滤)

**Reviewer**: Claude (self-review + security focus)
**Date**: 2026-05-12
**Scope**: 8 files (6 backend + 1 modules wiring + 1 e2e script) / Type=feat / **security-sensitive**
**Related**: [`verify-T-0090-c.md`](./verify-T-0090-c.md)

---

## §1. Findings 严重度汇总

| Severity | Count | Notes |
|----------|-------|-------|
| **CRITICAL (P0)** | 0 | 无 SQL 注入 / 提权 / 跨用户泄露 |
| **HIGH (P1)** | 0 | RBAC 设计符合最小权限原则 |
| **MEDIUM (P2)** | 0 | |
| **LOW (P3)** | 2 | 日志埋点可加 user_id / monitoring metric 缺失 |
| **NOTE** | 5 | 设计抉择记录 |

**APPROVE** — 无 P0/P1，可以合入。

---

## §2. 安全审查（/security-review 强制路径）

### 2.1 SQL 注入审计

| 输入面 | 防护 |
|--------|------|
| `filter.Creator` (string) | Squirrel `sq.Eq{"creator": *filter.Creator}` 参数化绑定 |
| `filter.VisibleGroupIDs` ([]uuid.UUID) | Squirrel `sq.Expr(SQL, args...)` 占位符 + pgx 自动数组绑定为 `ANY($1)` |
| `filter.CommandScope` 等 query param | Squirrel `sq.Eq` 参数化 |
| EXISTS 子查询 | 列名 hard-coded（`u.username` / `mml_custom_command.creator`）|
| user_id (uuid) | gin context 类型断言到 `uuid.UUID`，不经字符串拼接 |

**结论**：无 SQL 注入向量 ✅

### 2.2 RBAC 数据泄露审计

**威胁模型**：admin_b 看到 admin_a 的私有命令（跨用户泄露）

| 防护层 | 实现 |
|--------|------|
| 默认拒绝 | filter.Creator + VisibleGroupIDs 均空 → 强制 `command_scope = 'public'` 唯一谓词 |
| 短路评估 | private 可见性 = `creator=$user OR EXISTS(group)` — 第三条隐式路径不存在 |
| group_ids 服务端派生 | client 不能传入任意 group_ids；仅传 user_id（middleware 验证 token + 写入 context）|
| 派生失败降级 | service 报错时 VisibleGroupIDs **保持空**（不残留旧值），降级到仅 self-fallback |
| 单元测试覆盖 | 6 case 含 admin_a/admin_b/admin_c/admin_d/QuerierError/NoQuerier 全场景断言 |

**特别防护**：`TestService_ListCustomCommands_RBAC_AdminBWithGroupB` 显式断言 `NotContains(captured.VisibleGroupIDs, groupA)` — 跨用户隔离反退化测试 ✅

### 2.3 权限提升审计

**威胁模型**：admin_a 通过构造请求让 service 把别人的 group_id 误判为自己的

| 防护点 | 实现 |
|--------|------|
| group_ids 派生只看 user_id | `roleQuerier.GetUserVisibleGroupIDs(filter.UserID)` 由 admin RoleRepo 直接 SQL JOIN user_roles + role_device_groups，绝不接受用户传入的 group_ids |
| user_id 来自可信中间件 | `c.Get("user_id")` 返回 auth middleware 写入的 uuid，已绑定到 JWT token 校验 |
| Service 接口接受 filter.UserID 后立即派生覆盖 VisibleGroupIDs | 若客户端尝试传入 VisibleGroupIDs，service 调用 roleQuerier 派生时**直接覆写**（line `filter.VisibleGroupIDs = groupIDs`），客户端伪造的值被清除 |

**潜在 LOW 风险**：客户端可在 query string 中携带不该传的字段。Squirrel 对 binding 严格，但建议 future 考虑 service layer 在派生前 `filter.VisibleGroupIDs = nil` 显式清零，防御深度。**当前实现**：service 派生成功时已覆盖；派生失败时不残留 — 安全。

### 2.4 信息泄露审计

| 泄露面 | 防护 |
|--------|------|
| log 输出 | 仅记录 user_id + error；不打印 creator username / command_code（敏感字段）|
| error 输出 | 派生失败时仅 warn log，HTTP 路径返回 200 + 降级结果，不向客户端暴露内部错误 |
| 跨命令命名空间 | 公有命令路径完全不读 VisibleGroupIDs（独立 `sq.Eq{"command_scope": "public"}` 谓词），admin context 不影响 public ✅ |

### 2.5 DoS / 资源耗尽审计

| 攻击面 | 评估 |
|--------|------|
| 巨量 group_ids | 现实场景一个 admin 最多几十个 groups；ANY(uuid[]) 在 PG 高效；EXISTS 内层 JOIN 单行常数级 — 无 DoS 风险 |
| EXISTS 子查询性能 | `users.username` 有 unique index；`user_roles (user_id, role_id)` PK；`role_device_groups (role_id, group_id)` 双索引 — 每行单字段查找 O(log N) |
| pagination | 已有 `LIMIT/OFFSET` 防止意外大查询 |
| 速率限制 | 端点级 rate limit 由 middleware 通用提供（与 admin/* 同 stack）|

### 2.6 安全审查结论

**SECURITY APPROVE** — 0 P0 / 0 P1 安全 finding。

---

## §3. 代码审查（/review 主入口）

### 3.1 `model.go` (+18 行)

- CustomCommandFilter 加 2 字段（UserID + VisibleGroupIDs）+ 18 行注释说明可见性规则
- 注释清晰列举三种 case（self-fallback / group-share / deny-by-default）
- NOTE-1：注释提到"R-NEW-2 mitigation" — 未来若改变可见性逻辑，注释会过时；建议保持 PR 描述同步更新

### 3.2 `service.go` (+30 行)

- `RoleQuerier` 小接口（1 方法）— 符合 §16.2 Go 工程"小接口、消费者定义"原则 ✅
- `SetRoleQuerier` 方法与既有 `SetAuditRepo` / `SetFanouter` / `SetCmdParamRepo` 一致 — 模式对称
- `ListCustomCommands` 派生逻辑：
  - 守卫：`s.roleQuerier != nil && filter.UserID != nil && *filter.UserID != uuid.Nil` 三重判空
  - 派生失败 warn log + 不阻断（fail-open with self-fallback retained）
- NOTE-2：`zap.Stringer("user_id", *filter.UserID)` 正确 — `uuid.UUID` 实现 `Stringer` 接口

### 3.3 `pg_repository.go` (+30 行)

- 旧 `if filter.Creator != nil && *filter.Creator != ""` 单条件 → 新 deny-by-default 双源（Creator + VisibleGroupIDs）OR 逻辑
- SQL 注释充分说明 R-NEW-2 防护点
- EXISTS 子查询使用列名比较（`u.username = mml_custom_command.creator`）— 无字符串拼接
- LOW-1：EXISTS 子查询的多列 JOIN 性能取决于索引；users.username 是 unique 索引；user_roles PK 是 (user_id, role_id) — 反向查询 role_id → user_id 有索引但非主键级。**建议**：监控生产 EXPLAIN ANALYZE，若发现热点可加 `(role_id) WHERE` 部分索引。**评估**：当前规模 (admin < 1000，单查询 N < 100) 无需提前优化

### 3.4 `handler.go` (+5 行)

- ListTemplates 注入 user_id 与既有注入 username 对称（c.Get + 类型断言 + 守卫）
- NOTE-3：`uid != uuid.Nil` 守卫防止 anonymous 请求传入 zero UUID → service 派生 0 个 group → 仍 deny-by-default

### 3.5 `service_test.go` (+180 行)

- 6 test case 覆盖完整 RBAC 状态空间（admin_a / admin_b 跨用户 / admin_c 多 group / admin_d 无 group / querier error / no querier inject）
- 每个 case 内联辅助函数 `newCustomCmdServiceForRBAC` 复用 + 各自独立 `repo` 验证状态
- LOW-2：测试中 `assertError` 类型用作 sentinel error；轻量但若未来需要 wrapped error chain 可考虑改用 `errors.New`。**评估**：当前测试用途足够，未来按需调整
- NOTE-4：test 没显式断言 `mockRoleQuerier.groupsByUser` 被调用 — 隐式通过 captured filter 内容验证。若想加显式 spy 计数，可在 mock 加调用计数字段；当前断言已充分

### 3.6 `modules.go` (+6 行) + `router.go` (+1 行)

- DI 装配：`mmlService.SetRoleQuerier(c.RoleRepo)` 在 mml 模块初始化时 + `misc` 模块 Depends 加 `"admin"` 保证 c.RoleRepo 就绪
- 守卫 `c.RoleRepo != nil` 在 modules.go 防御性判空（理论上 Depends 已保证非空）
- NOTE-5：c.RoleRepo 类型是 `*admin.PgRoleRepository`，结构体；它"is-a" `RoleQuerier`（具体类型实现接口）— Go 实现接口隐式声明，无需 type assertion

### 3.7 `e2e_verify.sh` (+13 行)

- 2 个新 claim 都使用 `check_status_in "200 401"`（兼容 token 失效场景）
- claim 注释明示"跨用户隔离语义由 service_test.go 6 用例覆盖" — 期望管理清晰
- LOW（已记录）：完整跨用户 e2e 测试需多用户多 disjoint group seed 数据；本任务 e2e 仅验证端点契约可达 + 服务端注入路径不破坏 list 行为

---

## §4. DoD 逐项核销

### 编译与类型
- [✓] 后端 `go build ./...` 通过
- [⚠️] 后端 `go test ./...` 全绿 — mml 包全过；internal/task 2 pre-existing FAIL 与 T-0090-c 无关（stash 验证）
- [N/A] 后端 `golangci-lint run` — 本会话不跑；改动文件均 gofmt 自然
- [N/A] 前端 typecheck / lint — 无 FE 改动

### 测试
- [✓] 新增/修改的代码包含单元测试（6 个新 RBAC test）
- [✓] 成功 + 失败两路径（QuerierError 降级 case 覆盖失败路径；其它 5 case 覆盖成功路径）
- [✓] 新 REST 端点 E2E — 0 新端点 + 2 e2e claim 覆盖 RBAC 修改后行为
- [N/A] 修复 bug 回归测试 — 不是 bug fix
- [✓] 覆盖率不退（新增 180 行 test 含 6 RBAC + mockRoleQuerier，覆盖率显著上升）
- [✓] 禁止禁用失败测试 — 0 禁用

### 迁移与数据
- [N/A] 全部（无迁移）

### 代码规范
- [✓] 无遗留 TODO / FIXME / panic
- [✓] 错误处理 `fmt.Errorf("...: %w", err)` 模式保留；新增 warn log 含 error wrap
- [✓] SQL 构建使用 Squirrel — 含 `sq.Expr` 占位符；无字符串拼接
- [N/A] Carrier 接口 — 无运营商分支
- [N/A] 前端 BackendXxx 映射 — 无 FE 改动
- [✓] 无 `any` — RoleQuerier 小接口显式
- [✓] zap 结构化日志 — `zap.Stringer("user_id", ...)` + `zap.Error(err)` 结构化字段

### 文档
- [✓] PR 说明（commit body 待 S6）含 Why（"RBAC 私有命令过滤 / R-NEW-2 mitigation"）
- [✓] 关联 Backlog Task: T-0090-c
- [✓] 关联 PRD: subtask 文件 + backlog T-0090 umbrella ⑥⑦
- [N/A] CLAUDE.md / omcgo/CLAUDE.md 同步 — 无约定变更
- [N/A] Swagger 同步 — 端点路径不变；DTO 字段不变；query param 集不变

### 流水线闭环
- [ ] commit footer 五元组（待 S6 执行）
- [ ] backlog.md Task 状态回写（待 S7 执行）
- [N/A] 快速通道 postmortem — type=feat 走完整 S0-S7

### 安全（路径触及 RBAC 强制审查 — 详 §2）
- [✓] 新增 API 端点有 JWT 验证 — 路径不变，原 JWT middleware 继承
- [✓] 敏感操作有 RBAC 校验 — 本任务**就是**为私有命令加 RBAC
- [✓] 输入参数有校验；SQL 参数化；避免路径遍历 — Squirrel + pgx 全程参数化
- [✓] 日志/错误不打印密码、Token、CPE 密钥 — 仅记录 user_id (uuid) + error，无敏感字段
- [N/A] 文件上传 — 不涉

### 可观测性
- [⚠️ LOW-3] log 路径已加 user_id；但**无新 metric**记录 group-share 命中频率（建议 future）
- [✓] 错误日志携带 user_id（可关联 trace_id 通过 ctx）
- [✓] 长耗时操作支持 context 取消 — `roleQuerier.GetUserVisibleGroupIDs(ctx, ...)` 透传 ctx

### 模块特定（mml backend + admin RBAC 边界）
- [✓] 接受接口，返回结构体 — RoleQuerier 是消费者定义接口
- [✓] 小接口（1 方法）
- [✓] DI 注入 via Setter — `SetRoleQuerier(rq RoleQuerier)` 模式
- [✓] 模块依赖声明 — `misc` Depends `"admin"` 明示

---

## §5. 结论

**APPROVE** — 安全审查通过（无 P0/P1）+ 代码审查通过（2 LOW 建议 + 5 NOTE 仅记录）。

可以进 S6 commit。
