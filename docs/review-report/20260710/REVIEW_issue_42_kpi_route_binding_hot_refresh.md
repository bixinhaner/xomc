# Issue #42 指标与平台绑定写后热更新 KPI 路由审查

- 审查日期：2026-07-10
- 审查范围：`fix/42-kpi-route-binding-hot-refresh` 相对 `fix/41-kpi-route-invalidator-recovery-alerts` 的 WIP diff
- 关联 Issue：#42「[缺陷][P1] 指标与平台绑定写后热更新 KPI 路由」
- 结论：Standards/Spec CRITICAL=0；初审 HIGH 已在本分支修复。

## Standards

审查来源：`AGENTS.md`、`CONTEXT.md`、`docs/project/dod.md`、`docs/expert-personas.md`。

初审发现：

1. HIGH：`UpsertPlatformFormula` / `DeletePlatformFormula` 缺少成功与失败路径测试。
   - 处理：已补 `service_test.go`，覆盖 upsert 成功后 commit+bump、输入/公式校验失败不 begin、不成功 commit 不 bump、route bump 失败 best-effort、delete not found 不 commit/bump。
2. MEDIUM：Indicator service 通过回调触发 KPI route invalidation，存在跨域副作用。
   - 判定：接受。该回调是消费者侧小接口，避免 `indicator -> router` import cycle；与 #41 的统一失效器设计一致，且当前只服务 KPI route 缓存一致性。
3. MEDIUM：平台公式 `platform` 输入未标准化。
   - 处理：已在 service 层 trim `platform` / `formula`，空值返回 `ErrInvalidInput`；公式校验失败也包装 `ErrInvalidInput`，REST 返回 400。

复核结论：无剩余 Standards CRITICAL/HIGH。

## Spec

Issue #42 要求：

- 指标管理和平台绑定公式 route 变更在 DB commit 后使用统一失效器。
- 平台绑定 upsert/delete 收敛到事务化 service，避免部分状态。
- 保留 `indicator:cache_version`，同时推进独立 `kpi-route:cache_version`。
- 失败写入/commit 失败不得 bump；成功操作每次只 bump 一次。
- 纯展示字段修改不制造无意义全局 bump。
- bump 最终失败仍返回已提交业务结果，可通过人工刷新入口恢复。
- ENB/GSM/GNB 表族一致。
- 回归证明同一 worker 下一份 PM 采用新 counter/KPI 集合。
- 不存在绕过 service 直接修改平台公式的管理入口。

初审发现：

1. HIGH：缺少“管理写入后，同一 worker 下一份 PM route 使用新集合”的回归证明。
   - 处理：已补 `Test_LookupByDevice_ManagementFormulaWriteInvalidatesSameWorkerNextPMRoute`，先让 worker 缓存旧 route，再模拟管理公式写入后通过统一 invalidator 推进 `kpi-route:cache_version`，随后同一 worker lookup 必须看到新增 counter/KPI。

复核结论：无剩余 Spec CRITICAL/HIGH。

## DoD 核查

- 后端 `go build ./...`：通过
- 后端 `go test ./...`：通过
- `git diff --check`：通过
- `golangci-lint run`：N/A，本机未安装 `golangci-lint`
- 前端 typecheck / 浏览器冒烟：N/A，本次无前端改动
- 新 REST 端点 E2E：N/A，本次未新增端点
- 新迁移 up/down：N/A，本次无迁移
- 安全敏感路径：N/A，未触及 `internal/admin/` 或 `middleware/auth*`
- 绕过检查：`rg` 复核后，平台公式管理写入除 repository 实现外已收敛到 `IndicatorManagementService`

## 验证命令

```bash
cd omcgo && go test ./internal/pm/indicator ./internal/pm/kpi/router ./cmd/app/provider
cd omcgo && go build ./... && go test ./...
git diff --check
command -v golangci-lint && golangci-lint --version
```

