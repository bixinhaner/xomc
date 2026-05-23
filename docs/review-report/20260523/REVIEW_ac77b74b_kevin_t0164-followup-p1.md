# Review Report — T-0164 收尾 P1 批次（9 项 + GPV object 隔离）

- **Branch**: draft/pm-kpi-impl
- **Scope**: pm, worker, asyncjob, dashboard, adhoc, provision, frontend, migration, sys_configs
- **Backlog**: T-0164 收尾 P1（部分）+ provision GPV object 隔离（subagent 独立产出）
- **Date**: 2026-05-23
- **Author**: shangyingbin (kevin)

## Conclusion

**PASS_WITH_WARNINGS** — 可合入。0 CRITICAL，3 WARNING（partial 实现 + v2 留底），4 INFO。

Go 全包 build + provision + pm + worker + asyncjob test 全过。

## Files Changed

**新文件（5）**：
- `omcgo/internal/core/asyncjob/metrics.go` — G8-Gap-4 5 个 Prometheus 指标
- `omcgo/internal/pm/aggregator/metrics.go` — G5-Gap-3 4 个 Prometheus 指标
- `omcgo/internal/provision/object_param_classifier.go` + `_test.go` — GPV object 隔离（14 unit test）
- `omcgo/migrations/seed/000166_seed_pm_async_jobs_sysconfigs.sql` — G7-Gap-5 + G8-Gap-3 sys_configs seed

**修改（12）**：
- `omcgo/internal/pm/adhoc/handler.go` — G7-Gap-7 creator 过滤 + admin all + isAdmin helper
- `omcgo/internal/pm/adhoc/repository.go` — G7-Gap-6 continuous cancel 写 window_end
- `omcgo/internal/pm/dashboard/service.go` — G6-Gap-8 share/unshare audit log
- `omcgo/internal/pm/handler.go` — G5-Gap-2 POST /pm/aggregation/recompute
- `omcgo/cmd/app/provider/pm.go` + `router.go` — pmAsyncJobRepo wiring
- `omcgo/cmd/worker/aggregator.go` — G8-Gap-3 sweeper threshold 从 sys_configs 读
- `omcgo/internal/provision/sync.go` — GPV scalar/object 分批
- `omcmb/webcode/src/pages/system/SystemConfig/index.tsx` — G2-Gap-1 PmRetention tab 挂载
- `omcmb/frontend-core/src/i18n/{zh-CN,en-US}/index.ts` — i18n key 'system.config.pmRetention'

## 实施清单

| 缺口 ID | 实施 | 状态 |
|---------|------|------|
| G7-Gap-7 (P1) | List handler 加 creator 默认过滤 + ?all=true（仅 admin） | ✓ |
| G7-Gap-6 (P1) | Cancel SQL 用 CASE 给 continuous 写 window_end=NOW() | ✓ |
| G6-Gap-8 (P1) | Service.Share/Unshare 注入 audit.Log（含失败也记） | ✓ |
| G5-Gap-2 (P1) | POST /pm/aggregation/recompute 直 INSERT async_jobs | ✓ |
| G2-Gap-1 (P1) | PmRetentionSection 挂载 + 2 lang i18n | ✓ |
| G7-Gap-5 (P1) | sys_configs 加 'pm.adhoc.retention_days' key（admin UI 已可改） | ✓ partial（改小 confirm 留 P2 frontend） |
| G5-Gap-3 (P1) | aggregator.Metrics 4 指标注册（Runs/Duration/RowsWritten/BucketLag） | ✓ 注册；hook 注入 v2 |
| G8-Gap-4 (P1) | asyncjob.Metrics 5 指标注册 + sys_configs seed | ✓ 注册；hook 注入 v2 |
| G8-Gap-3 (P1) | Sweeper interval/threshold 从 sys_configs 读（启动生效） | ✓ partial（HeartbeatInterval 常量 + 热重载留 v2） |
| **GPV object** | provision.enqueueGPVPrefixes scalar/object 分批 | ✓ subagent 独立产出 |
| G4-Gap-1 (P1) | PM 上报延迟 histogram | 🚧 未做（需碰 PM collector parse 层） |
| G7-Gap-9 (P1) | cron lossless 补跑漏桶 | 🚧 未做（需加 last_runs_at 列大改） |
| Cross-Gap-1 (P1) | e2e_verify.sh 加 G5/G6/G7/G8 断言 | 🚧 未做 |
| Cross-Gap-2 (P1) | release-gate.md 更新 | 🚧 未做 |

## Findings

- 无 CRITICAL
- **W1**（Prometheus instrumentation hook 缺失）：Runs / Duration / RowsWritten / Failed / Zombie / Catchup 等指标已注册但 Aggregator.Run / Registry.RunNext / Sweeper.sweepOnce 内部还没调 Inc / Observe — 当前所有 metric 值为 0。需要 v2 在 framework 层注入 hook。
- **W2**（G8-Gap-3 partial）：sweeper_interval / zombie_threshold 已可调（启动生效）；HeartbeatInterval 仍是包常量 `asyncjob.HeartbeatInterval=30s`，要让 heartbeat 可调需 Registry.runHeartbeat 接受参数 + sys_configs 读 + 热重载 — 留 v2。
- **W3**（GPV object 隔离）：subagent 实施按"末尾 `.`"分类，与 TR-069 GPV 规范一致；与 entry_type='object' 同源。极端 CPE 不规范行为下叶子参数也可能 9005，但本批不扩大覆盖范围。
- 所有 SQL 参数化无注入；audit.Log 失败也记（Success=false + ErrorMessage）合规追溯。
- recompute endpoint 用 http.StatusAccepted (202) 表"已接受待异步执行"，符合 REST 语义。
- creator 过滤行为：默认按当前用户 → admin 显式 ?all=true → 显式 ?creator=xxx 三层优先级。

## Tests

- `Test_isObjectPath / Test_classifyPrefixes / Test_buildGPVBatches`（14 case） — subagent 产出
- 现有 pm/adhoc, dashboard, retention, aggregator, asyncjob 测试全部仍过

## Migration Self-check（seed/000166）

- 版本号 165 → 166 连续
- StatementBegin/End 包裹
- INSERT 列名匹配（is_public，非 is_sensitive — 首次错列名已修）
- ON CONFLICT DO NOTHING 幂等
- Down 段清理 4 个 key

## DoD（本批）

- [x] go build ./... 全过
- [x] go test 全过
- [x] webcode typecheck 全过
- [x] 9 项 P1 主体（其中 2 项 partial 标 v2）
- [x] subagent GPV object 隔离合并
- [ ] 4 项 P1 剩余项（G4-Gap-1 / G7-Gap-9 / Cross-Gap-1+2）留下一会话

## 待办

- **P1 剩余 4 项**：G4-Gap-1（PM 上报延迟）/ G7-Gap-9（lossless 补跑）/ Cross-Gap-1（e2e）/ Cross-Gap-2（release-gate）
- **P2 13 项**：UI 完整度（左右栏 / 全局筛选 / 12 内置仪表盘 / TopN / 粒度多选 tab / 对比双模式真数据 / 显示增强 / KPI 卡片按制式分键 UI）
- **P3 7 项**：导出 + G6-G7 集成
- **Prometheus instrumentation hooks**: 已注册指标在 Run / Registry / Sweeper 内调用 Inc/Observe
