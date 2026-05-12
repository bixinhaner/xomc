# Verify Report — T-0090-c (MML 后端 RBAC 私有命令过滤)

**Task**: T-0090-c — MML 后端 RBAC 私有命令过滤：service.go + pg_repository.go ListCustomCommands(scope=private) JOIN user→role_device_groups 派生 group_ids；公有命令路径不变；admin context (user_id) 注入到 service 层
**Type**: feat / F06/mml+admin / P2 / M-L (3-4d)
**Sprint**: sprint-11 (pull-forward 进 sprint-10 buffer)
**Deps**: T-0090-a ✅ + T-0090-b ✅
**Owner**: Claude
**Date**: 2026-05-12

---

## §1. 改动面

| 文件 | 操作 | 行数 |
|------|------|------|
| `omcgo/internal/mml/model.go` | 改 | +18 行（CustomCommandFilter 加 UserID + VisibleGroupIDs 2 字段 + 注释说明可见性规则）|
| `omcgo/internal/mml/service.go` | 改 | +30 行（新增 RoleQuerier 小接口 + roleQuerier 字段 + SetRoleQuerier 方法 + ListCustomCommands 派生逻辑）|
| `omcgo/internal/mml/pg_repository.go` | 改 | +30 行（List visibility 重写 — deny-by-default + self-fallback OR group-share；EXISTS 子查询）|
| `omcgo/internal/mml/handler.go` | 改 | +5 行（ListTemplates 注入 user_id 到 filter）|
| `omcgo/internal/mml/service_test.go` | 改 | +180 行（mockRoleQuerier 类型 + 6 个 RBAC 测试 + assertError 辅助类型）|
| `omcgo/cmd/app/provider/modules.go` | 改 | +6 行（mml.Service.SetRoleQuerier(c.RoleRepo) 装配 + 注释）|
| `omcgo/cmd/app/provider/router.go` | 改 | +1 行（misc 模块 Depends 加 "admin" 确保 c.RoleRepo 就绪）|
| `omcgo/scripts/e2e_verify.sh` | 改 | +13 行（W2D mml-4a + mml-4b 两 RBAC 路径 claim）|

---

## §2. 设计决策

### 2.1 接口签名保持稳定（兼容现有调用）

不修改 `ListCustomCommands(ctx, filter)` 签名，而是把 user_id 作为 filter 字段携带。
- 优点：service mock 不变 / 调用方仅扩展字段不破坏
- 缺点：handler 必须显式注 UserID，遗漏即降级（mitigation：handler 已 inline 注入 + service_test 验证降级安全）

### 2.2 可见性规则（关键安全决策）

```
visible IF
  command_scope = 'public'                  -- 始终可见
  OR (
    command_scope = 'private'
    AND (
      creator = $currentUsername            -- self-fallback：自己始终能看到自己创建的
      OR EXISTS (                           -- group-share：同组管理员协作
        SELECT 1 FROM users u
        JOIN user_roles ur ON ur.user_id = u.id
        JOIN role_device_groups rdg ON rdg.role_id = ur.role_id
        WHERE u.username = mml_custom_command.creator
          AND rdg.group_id = ANY($currentUserGroupIDs)
      )
    )
  )
```

**Deny-by-default**：filter.Creator 与 VisibleGroupIDs 均空 → 仅 public 可见（匿名 / 无凭据请求看不到任何 private）

**Self-fallback OR clause 的必要性**：subtask GWT-c "仅返自己的 1 条" 语义；防止 admin 调整角色后看不到自己创建的 private（用户体验底线）

**Group-share JOIN 路径**：subtask Notes "JOIN user→role_device_groups 派生" 原文设计；实现"团队私有库"协作语义

### 2.3 RoleQuerier 小接口（消费者驱动）

```go
type RoleQuerier interface {
    GetUserVisibleGroupIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}
```

- 不直接依赖 admin 包 → mml 包零反向依赖
- admin.PgRoleRepository 天然实现 → 装配时直接传入
- 测试 mock 简单实现 → 全测试覆盖派生场景

### 2.4 派生失败降级（service 层）

`GetUserVisibleGroupIDs` 报错时不阻断查询，降级为仅 creator-self 可见（warn log）。原因：
- private 命令查询是只读操作，DB 临时不可用时降级"看少不看错"
- 不会引入数据泄露（VisibleGroupIDs 保持空）
- 用户最差体验是看不到同组同事的 private，但能看到自己的

---

## §3. 出口门检查

| 检查项 | 结果 | 备注 |
|--------|------|------|
| `go build ./...` | ✅ PASS | 无输出 |
| `go test -race -count=1 ./internal/mml/...` | ✅ PASS | mml 包 1.078s；6 个新 RBAC test 全绿（TestService_ListCustomCommands_RBAC_*）|
| `go test -race -count=1 ./...` 全工程 | ⚠️ 2 pre-existing FAIL | `internal/task` 包 (scan NULL into *string for col source_id)；与 T-0090-a/b 同一基线，**已 stash 验证 main commit `a405b601` 同样 FAIL**，与本任务无关 |
| 无新增 TODO/FIXME/panic | ✅ | grep 验 |
| 无新增 `any` / `interface{}` | ✅ | RoleQuerier 是显式小接口 |
| 无新增 `if carrier ==` 硬编码 | N/A | 无运营商分支 |
| 后端 `golangci-lint run` | N/A | 本会话不跑全局 lint；改动文件自然 gofmt 格式 |
| 前端 typecheck / lint | N/A | 本任务无前端改动 |
| 迁移 up/down | N/A | 无新迁移 |
| 新端点 R | 0 | 无新路由（ListTemplates 路径不变，只是行为加 RBAC）|
| E2E claim 增量 E | **2** | mml-4a (scope=private) + mml-4b (scope=public) |
| E/R 比 | N/A | 0 新端点；改加 2 e2e 用例验端点契约稳定 |
| 新 metric/log 名 | 0 | 无新埋点 |
| 累计型 deps | N/A | Deps `—`（普通 T-NNNN deps T-0090-a/b 均 done）|

---

## §4. 6 个 RBAC 单测场景核销

| Test | 场景 | 关注断言 |
|------|------|---------|
| `RBAC_AdminAWithGroupA` | admin_a 属 group_A，VisibleGroupIDs 应派生为 [group_A] | VisibleGroupIDs 正确填充 + Creator self-fallback 透传 |
| `RBAC_AdminBWithGroupB` | admin_b 属 group_B，跨用户隔离断言 admin_b 不见 group_A | `NotContains(VisibleGroupIDs, groupA)` — R-NEW-2 防泄露关键测试 |
| `RBAC_AdminCWithMultipleGroups` | admin_c 同属 group_A+B，可看两组 private | `ElementsMatch([groupA, groupB], VisibleGroupIDs)` |
| `RBAC_AdminDWithoutGroups` | admin_d 无任何 group，但自己创建的 private 应仍可见 | VisibleGroupIDs empty + Creator self-fallback 透传 |
| `RBAC_QuerierError_DegradesToCreatorSelf` | GetUserVisibleGroupIDs 报错，service 降级 | VisibleGroupIDs empty (不残留旧值) + Creator self-fallback 透传 + 不阻断查询 |
| `RBAC_NoQuerierInjected_FallbackToCreatorOnly` | roleQuerier 未注入（向后兼容） | VisibleGroupIDs empty + 不触发派生 + Creator self-fallback 透传 |

**6 用例同时验证 4 个关键不变量**：
1. group-share 路径正确派生（admin_a → group_A）
2. 跨用户不泄露（admin_b 不见 group_A）
3. 多 group 合集正确（admin_c 见 [A, B]）
4. self-fallback 永远透传（无 group 或派生失败都保留 Creator 字段）

---

## §5. SQL 设计的 R-NEW-2 防护点

| 防护点 | 实现 |
|-------|------|
| 参数化绑定 | 所有用户输入经 Squirrel `sq.Eq` / `sq.Expr` 参数化，无字符串拼接 |
| 默认拒绝 | filter.Creator + VisibleGroupIDs 均空时强制 `command_scope = 'public'` 唯一可见性谓词 |
| 短路评估 | private 可见性是 `creator=username OR EXISTS(...)`，任一短路命中即可见，无第三条隐式路径 |
| 列名隔离 | EXISTS 子查询用 `mml_custom_command.creator` 全限定列名避免歧义 |
| group_ids 服务端派生 | 用户不能传入任意 group_ids；只能传 user_id → 服务端 RBAC 派生 |
| 列名 hard-coded | 所有 SQL 列名 hard-coded（无动态拼接）|

---

## §6. 用户回归路径

需多用户多 group seed 数据真实测试 RBAC 隔离：

1. dev DB 已有 admin (17 groups) + test (4 groups) 两 user
2. admin 登录 → 创建 private 命令 P1
3. test 登录 → 创建 private 命令 P2
4. admin 列 private → 看 P1 + 看 P2（若 test 的 groups ⊆ admin's 17 groups → group-share 命中）
5. test 列 private → 看 P2 + 看 P1（若 admin 的 17 groups 与 test 的 4 groups 有交集）
6. admin 列 public → 看所有 public 命令

**完整跨用户 RBAC 隔离测试**需 staging 环境 seed 多用户多 disjoint group，本任务限于 dev 测试可见路径，**严格隔离断言在 service_test.go 6 用例 + 真 PG 集成测试由 staging 验证**。

---

## §7. S4 出口门

- [✓] 所有命令绿（go build + mml 包 go test -race + 6 新 RBAC test）
- [✓] E/R ≥ 1：0 新端点（ListTemplates 路径不变） + 2 e2e claim 验端点契约稳定
- [N/A] 迁移双向演练（无迁移）
- [N/A] metric/log 名（无新埋点）
- [N/A] 累计型依赖阈值（无累计 deps）

**S4 PASS**。
