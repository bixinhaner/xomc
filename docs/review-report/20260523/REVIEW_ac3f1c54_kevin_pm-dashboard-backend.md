# Review Report — T-0164-P6 / G6 后端：PM 性能查看 Dashboard / Panel / 用户偏好

- **Branch**: draft/pm-kpi-impl
- **Scope**: pm, migration, app/provider
- **Backlog**: T-0164-P6（仅后端，前端留下一轮）
- **Date**: 2026-05-23
- **Author**: shangyingbin (kevin)
- **Reviewer**: Claude (AI self-review)

---

## Conclusion

**PASS_WITH_WARNINGS** — 后端部分可合入。

- 0 CRITICAL
- 2 WARNING（设计 trade-off）
- 4 INFO

13 unit + 7 integration tests pass。全包构建 `go build ./...` 通过。

---

## Files Changed

| Path | LOC | Type |
|------|-----|------|
| `migrations/000163_create_pm_dashboards_and_panels.sql` | +95 | new |
| `internal/pm/dashboard/model.go` | +140 | new |
| `internal/pm/dashboard/repository.go` | +400 | new |
| `internal/pm/dashboard/repository_test.go` | +200 | new |
| `internal/pm/dashboard/service.go` | +175 | new |
| `internal/pm/dashboard/service_test.go` | +260 | new |
| `internal/pm/dashboard/share.go` | +22 | new |
| `internal/pm/dashboard/handler.go` | +430 | new |
| `internal/pm/dashboard/handler_test.go` | +145 | new |
| `cmd/app/provider/pm.go` | +9 | mod |
| `cmd/app/provider/router.go` | +5 | mod |
| `docs/project/backlog/subtasks/T-0164-pm-kpi-pipeline.md` | +1/-1 | mod |

---

## Findings

### CRITICAL — 0

无。

### WARNING — 2

#### W1 — shared_with 用 UUID[] 数组而非关联表 → 单 dashboard 分享数量上限

**File**: `migrations/000163_create_pm_dashboards_and_panels.sql`

`pm_dashboards.shared_with UUID[]` 单数组存被分享用户列表。优点是单查询完成 List by Owner Or Shared（GIN 索引支持 ANY 操作），写入也是单 SQL（array_append）。缺点是单 dashboard 分享数 > 数千时，UPDATE 整数组成本变高（PG 数组替换全列）。

**Mitigation**：业务场景预期 < 100 用户/dashboard，远低于 GIN/UPDATE 成本临界点。

**Action**：本次接受；若未来出现"全公司广播"场景再迁移到关联表 `pm_dashboard_shares (dashboard_id, user_id)`。

#### W2 — adhoc_task_id 软引用 pm_tasks，无 FK 兜底孤儿

**File**: `migrations/000163_create_pm_dashboards_and_panels.sql::pm_panels`

`adhoc_task_id UUID` 没有 FK 约束（注释说明：避免跨域 FK）。如果用户删除 adhoc task，对应 panel 仍引用 ghost id。

**Mitigation**：
- panel 加载时按 adhoc_task_id 查 results — 找不到就显示"任务已删除"
- 前端展示侧加 fallback
- 后续可加 ON DELETE 触发器清空 panel.adhoc_task_id（不级联删 panel，因为还有 fork 关系）

**Action**：本次不修；G6 前端实施时确认 UI fallback。

### INFO — 4

- **I1**: 所有 SQL 走 squirrel 参数化或 pgx $N 绑定，无字符串拼接 user input。
- **I2**: Fork 单事务保证 dashboard + panels 同步复制，失败回滚（防孤儿 panels）。
- **I3**: ACL 用 share.go 独立 CanRead/CanWrite 函数，便于跨 service/handler 复用 + 单测。
- **I4**: handler 用 `c.Get("user_id")` 拿登录用户（admin.AuthMiddleware 注入），404/403 区分到位（ErrNotFound vs ErrPermissionDenied）。

---

## Tests

| 测试 | 覆盖 | 结果 |
|------|------|------|
| `Test_Repository_DashboardCreateGetList` | Create + Get + List by owner | PASS |
| `Test_Repository_UpdateDashboardPartial` | 部分字段更新 | PASS |
| `Test_Repository_ShareAndUnshare` | shared_with 数组操作 + shared 用户能 List 看到 | PASS |
| `Test_Repository_ForkCopiesPanels` | Fork 事务级联复制 panels + parent_dashboard_id | PASS |
| `Test_Repository_PanelCRUDAndCascade` | Panel CRUD + dashboard 删除级联 | PASS |
| `Test_Repository_UserPreferencesUpsert` | upsert 覆盖 | PASS |
| `Test_Service_Get_*` (3) | owner 可读 / 他人 403 / shared 可读 | PASS |
| `Test_Service_Update_OnlyOwner` | shared 用户不可改 | PASS |
| `Test_Service_Fork_RequiresReadAccess` | 无读权限不可 fork | PASS |
| `Test_Service_Share_FiltersOwnerSelf` | share 时 owner 被过滤 | PASS |
| `Test_Service_GetUserPreferences_ReturnsEmptyOnMiss` | 缺省返空偏好 | PASS |
| `Test_Service_CreatePanel_NonOwnerDenied` | shared 用户不可创建 panel | PASS |
| `Test_Handler_CreateRequiresUserID` | 未登录 401 | PASS |
| `Test_Handler_CreateAndGet` | Create + Get round-trip | PASS |
| `Test_Handler_GetForbidden` | 他人访问 403 | PASS |
| `Test_Handler_UpdateOnlyOwner` | shared 用户改 403 | PASS |
| `Test_Handler_GetPreferencesReturnsEmpty` | 偏好缺省 | PASS |

---

## Migration Self-check

按 `omcgo/CLAUDE.md §5.5.10` 清单：

- [x] 版本号 = 000163（前次 000162 + 1，连续）
- [x] 包含 Up/Down 两段
- [x] StatementBegin/End 包裹
- [x] INSERT 与 DDL 列名匹配（本迁移无 INSERT）
- [x] 无分区表间外键
- [x] CHECK 约束兼容 NULL（如 compare_mode）
- [x] Down 删 Up 所有对象
- [x] 三轮 up/down/up 幂等已实测通过

---

## DoD

- [x] go build ./... 通过
- [x] go test ./internal/pm/... 全过
- [x] Integration test pm/dashboard 全过
- [x] migration up/down/up 三轮幂等
- [x] 13 REST endpoints 注册到 /api/v1/pm/dashboards + /pm/user-preferences
- [x] backlog T-0164-P6 状态 planned → partial_backend_done
- [x] review report 与代码同 commit
- [ ] 前端 13 task（留下一轮）

---

## Out of scope（前端 13 task 由下一轮做）

- frontend-core: types/dashboard.ts + dashboardApi + useDashboard hooks + dashboardStore + i18n + mock
- webcode: pages/performance/DashboardList + Editor + PanelGrid + PanelConfigDrawer + PanelRenderer + ComparePanel + ForkDialog + ShareDialog + AdhocAggregation
- 删除旧三 tab + 路由整改 + playwright 自测
