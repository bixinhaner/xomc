# Review Report — T-0164 收尾 P2 第 1 批（G6-Gap-5 / G6-Gap-4 / G6-Gap-9）

- **Branch**: draft/pm-kpi-impl
- **Scope**: pm.dashboard 后端 + frontend-core/pmDashboard 类型 + PanelRenderer/PanelConfigDrawer
- **Backlog**: T-0164 收尾 P2 batch 1
- **Date**: 2026-05-23
- **Author**: shangyingbin (kevin)

## Conclusion

**PASS** — 可合入。0 CRITICAL，0 WARNING，1 INFO（用 mock 数据，G6-G7 集成接真数据待 P3）。

Go 全包 build + dashboard test 全过；前端 typecheck 通过。

## Files Changed

**新文件（2）**：
- `omcgo/migrations/000168_pm_panels_topn_bignumber_builtin_flag.sql` — DDL：panel_type CHECK + is_builtin flag
- `omcgo/migrations/seed/000169_seed_pm_builtin_dashboards.sql` — 12 内置 dashboard seed

**修改（6）**：
- `omcgo/internal/pm/dashboard/model.go` — PanelType 加 topn / big_number；Dashboard 加 IsBuiltin
- `omcgo/internal/pm/dashboard/repository.go` — dashboardCols + scanDashboard + ListByOwnerOrShared 含 is_builtin / Fork 显式置 FALSE
- `omcgo/internal/pm/dashboard/service.go` — Update/Delete 拒 is_builtin（ErrBuiltinReadonly）
- `omcgo/internal/pm/dashboard/handler.go` — DTO 加 is_builtin / 错误映射 ErrBuiltinReadonly → 403
- `omcmb/frontend-core/src/types/pmDashboard.ts` — PanelType 扩 + Dashboard.isBuiltin + mapper
- `omcmb/webcode/src/pages/performance/PmDashboard/DashboardList.tsx` — "系统内置" Tag + hide 编辑/删除
- `omcmb/webcode/src/pages/performance/PmDashboard/PanelRenderer.tsx` — TopN + BigNumber + threshold markLine + pct 自动单位 + 缺采断线
- `omcmb/webcode/src/pages/performance/PmDashboard/PanelConfigDrawer.tsx` — 加 topn/big_number 选项 + 类型化 config + time_axis 切换

## 实施清单

| 缺口 ID | 实施 | 状态 |
|---------|------|------|
| G6-Gap-5 (P2) | panel_type CHECK 扩 topn + big_number；PanelRenderer 2 个新渲染器；config 类型化字段 | ✓ |
| G6-Gap-4 (P2) | is_builtin 列 + 12 内置 dashboard seed（3 制式 × 4 报表）+ Service guard + UI Tag | ✓ |
| G6-Gap-9 (P2) | pct % 自动单位 / threshold markLine / 缺采断线 / time_axis end vs ingest 切换 / KPI 红色阈值 | ✓ |

## 关键设计决策

### is_builtin readonly 设计
- 数据层防护：Service.Update/Delete 检查 IsBuiltin 拒，返 ErrBuiltinReadonly（HTTP 403）
- UI 层防护：DashboardList 隐藏编辑/删除按钮；保留"派生"让用户能基于内置创建可改副本
- Fork 显式置 FALSE：防止内置 dashboard 派生后仍 readonly

### 内置 dashboard UUID 命名
- 用结构化 sentinel UUID：`30000000-{TT}{RR}00-0000-0000-{12 hex}`
  - TT = 11/12/13/14（LTE 子类 1-4）、21/22/23/24（NR 子类）、31/32/33/34（GSM 子类）
  - 便于运维直接按 ID 前缀识别是哪类 dashboard

### PanelRenderer pct 单位推断
- 正则 `(\.|^)(Rate|SuccRate|FailRate|DropRate|Pct|Percentage|Ratio)(\.|$)` 命中即默认 `%`
- 用户显式 `config.unit` 覆盖自动推断
- 与现有 perf_indicators_*.en_name 命名规范对齐（如 L.RRC.SuccRate）

### 缺采点不画 0
- echarts 用 `'-'` 占位 + `connectNulls: false` → 自动断线，避免误读为"等于零"
- mock 数据混入一个 `'-'` 演示行为

### time_axis 切换
- 后端 G3+G4 已写入三时间字段（start_time/end_time/ingest_time）
- 前端 panel.config.time_axis 选 `end_time` (默认) 或 `ingest_time`
- 接真数据时后端 query handler 按 time_axis 字段切换 ORDER BY / X 轴

## 测试结果

```
ok      github.com/omcgo/omcgo/internal/pm/dashboard  (cached)
ok      cd omcmb/webcode && npm run typecheck (no errors)
```

## 风险与边界

- **migration 000168** 涉及 DROP CONSTRAINT + ADD CONSTRAINT —— 短暂锁表（pm_panels 是普通表，行数 <万级，瞬间完成）
- **seed 000169** 12 行 ON CONFLICT DO NOTHING，幂等
- **Down 路径**：CHECK 回退 + DROP COLUMN —— 老 dashboard 仍可读
- **frontend 多皮肤影响**：仅改 frontend-core/types/pmDashboard.ts 加可选字段；webcode-v2/v3 typecheck 不会破（is_builtin?: boolean 是可选的）

## INFO（可选）

1. **TopN/BigNumber 数据来源** 仍用 mock；G6-G7 集成 P3 改为 hook 拉真数据（同 KpiCard/LineChart 一致）。
2. **time_axis = ingest_time** 暂只改 X 轴 label；接真数据时 query 需带 `?time_axis=ingest_time` 参数让后端 select ingest_time as ts。

## 部署后验证

```bash
# 1. 验证 migration 应用
docker exec omc-docker-postgres-1 psql -U omcgo -d omcgo -c "
  SELECT column_name FROM information_schema.columns
  WHERE table_name='pm_dashboards' AND column_name='is_builtin';"

# 2. 验证 12 内置 dashboard
docker exec omc-docker-postgres-1 psql -U omcgo -d omcgo -c "
  SELECT technology, COUNT(*) FROM pm_dashboards WHERE is_builtin GROUP BY technology;"
# 期望：gsm/lte/nr 各 4 行

# 3. 浏览器 /performance/pm-dashboard 看到"系统内置" Tag + 12 行
# 4. 选一个内置点删除/编辑应被前端按钮 hide；直接 API 调用应返 403
```

## Sign-off

| 角色 | 结论 |
|------|------|
| Go 工程专家 | ✓ Pass — error 分类清晰；defer transaction rollback 兜底 |
| 数据与存储专家 | ✓ Pass — migration 000168/seed 000169 编号连续；Down 配对；UUID 8-4-4-4-12 合规 |
| 前端专家 | ✓ Pass — 改动定位正确（业务层 + UI 各司其职）；多皮肤兼容 |
| 电信业务专家 | ✓ Pass — 3 制式 × 4 报表覆盖运营商主流场景（LTE 全网概览 + 日报/周报/月报） |
