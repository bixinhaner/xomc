# Issue #43 product 路由字段写后热更新 KPI 路由审查

## 范围

- 审查范围：`fix/43-product-route-fields-hot-refresh` 相对 `fix/42-kpi-route-binding-hot-refresh` 的 WIP diff
- 关联 Issue：#43「[缺陷][P1] product 路由字段写后热更新 KPI 路由」
- Stacked 基线：`71328a2c`

## Standards

初审发现 1 个硬违规：

1. product 侧 KPI route invalidation 失败日志未携带 `request_id` / `trace_id`。
   - 处理：`logRouteInvalidationFailure` 改为接收 `context.Context`，从 context 补 `request_id` 字段；统一 invalidator 自身的 ERROR 日志沿用既有 request_id 逻辑。

复查结果：未发现 CRITICAL 或硬性标准违规。未触及 `internal/admin/` 或 `middleware/auth*`，安全专项审查 N/A。

## Spec

初审发现 2 个需求风险：

1. product XML reload 的 KPI route bump 早于 ProductRegistry refresh，worker 可能在新 KPI route version 下重建到旧 product 字段。
   - 处理：移除 loader 内直接 bump。产品中心 `ImportDirectory` 和通用 `admin/dictload/reload?name=product` 现在都先记录路由字段快照，reload 成功后刷新 ProductRegistry，再按实际字段变化推进 KPI route version。
2. 回归测试直接改 fake product，不足以证明 ProductRegistry 重新匹配链路。
   - 处理：router 回归改为使用真实 `product.Registry` + fake repository；测试流程为旧 route 入 L1 → 修改 product repo 路由字段 → `ProductRegistry.Refresh` → 统一 invalidator bump → 同一 worker 再 lookup。

复查结果：Issue #43 AC 已覆盖；未发现 scope creep。

## 验证

- `cd omcgo && go build ./...` 通过
- `cd omcgo && go test ./...` 通过
- `cd omcgo && go test -race ./internal/product ./internal/pm/kpi/router ./cmd/app/provider` 通过
- `git diff --check` 通过
- 新端点：N/A
- 前端改动：N/A
- 迁移：N/A

## DoD 摘要

- 后端 build/test/race 已过。
- 新增/修改代码包含回归测试：product 路由字段比较、invalidator 回调、同 worker + ProductRegistry refresh 链路覆盖。
- 失败路径：invalidator 失败不回滚已提交写入；统一 invalidator 继续负责 ERROR、指标和告警。
- 可观测性：复用 `omc_kpi_router_invalidations_total`，新增低基数 trigger `product_write` / `product_reload`。
- 安全敏感路径：N/A。
