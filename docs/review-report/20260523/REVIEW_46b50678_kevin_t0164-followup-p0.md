# Review Report — T-0164 收尾 P0 批次（4 项）+ PM 上传自动配置

- **Branch**: draft/pm-kpi-impl
- **Scope**: pm, worker, asyncjob, dashboard, retention, migration, frontend
- **Backlog**: T-0164 收尾 P0 — G2-Gap-2 / G5-Gap-1 / G6-Gap-3+13 / G8-Gap-1（详 docs/project/plan-T-0164-followup-gaps.md）
- **附带**：今早 PM 上传自动配置（device.online OnlineSubscriber）
- **Date**: 2026-05-23
- **Author**: shangyingbin (kevin)

## Conclusion

**PASS_WITH_WARNINGS** — 可合入。0 CRITICAL，3 WARNING，6 INFO。

Go 全包 build + 全 PM + asyncjob + worker test 通过；webcode typecheck 通过；2 migration（000164 + 000165）三轮 up/down/up 幂等通过。

## Files Changed

**新文件（14）**：
- `docs/project/plan-T-0164-followup-gaps.md` — 落档全部 30+ 项缺口
- `docs/review-report/20260523/REVIEW_pending_kevin_pm-auto-setup.md` + 本文件
- `omcgo/migrations/000164_async_jobs_cron_state.sql` — G8-Gap-1
- `omcgo/migrations/000165_pm_user_prefs_by_technology.sql` — G6-Gap-3
- `omcgo/internal/core/asyncjob/cron_state.go` + `_test.go` — G8 cron 持久化 + 启动补跑算法
- `omcgo/internal/pm/online_subscriber.go` + `_test.go` — device.online → SPV 入队
- `omcgo/internal/pm/aggregator/group_runner.go` + `hourly_group.go` + `daily_group.go` + `weekly_group.go` + `monthly_group.go` + `group_runner_test.go` — G5 设备组级 4 cron runner
- `omcgo/internal/pm/retention/cleanup_runner.go` — G2 普通表 cleanup
- `omcgo/cmd/worker/retention.go` — retention cleanup wiring + cron

**修改（12）**：
- `omcgo/internal/core/appconfig/config.go` — PMConfig
- `omcgo/cmd/worker/etc/config.{dev,test,prod,local}.yaml` — pm 段
- `omcgo/cmd/worker/main.go` — OnlineSubscriber 装配
- `omcgo/cmd/worker/aggregator.go` — cron state + 8 runner（4 device + 4 group）+ retention cron 接入
- `omcgo/internal/pm/dashboard/{model,repository,service,handler}.go` + 2 test — UserPreferences 加 technology 维度
- `omcmb/frontend-core/src/{types/pmDashboard,services/api/pmDashboardApi,hooks/api/usePmDashboard,mock/data/pmDashboard}.ts` — 前端按制式分键

## 实施缺口对照

| 缺口 ID | 实施 | 状态 |
|---------|------|------|
| G8-Gap-1 (P0) | async_jobs_cron_state 表 + 启动补跑 | ✓ |
| G5-Gap-1 (P0) | 设备组独立 4 cron runner（hourly/daily/weekly/monthly_group） | ✓ |
| G2-Gap-2 (P0) | 普通表 6 张（daily/weekly/monthly × device + device_group）每日 03:00 cleanup | ✓ |
| G6-Gap-3 (P0) | pm_user_dashboard_preferences 按 (user_id, technology) 主键分键 | ✓ |
| G6-Gap-13 (P0) | KPI 卡片配置 layout 按制式分键持久化 | ✓ |
| **G1 收尾**（PM auto-setup） | device.online → SPV 3 参数自动下发 | ✓ |

## Findings

- 无 CRITICAL
- **W1**（G8 cron_state）：补跑上限 10000 buckets 防极端积压打爆 DB；超过会 log warn 提示运维介入。生产 hourly cron 一年 8760 个，正常停机不会触上限。
- **W2**（G5 设备组 cron 错峰 10 分钟）：基于"device 级聚合完成后再跑 group 级"的时间假设。若 hourly device cron 跑超 10 分钟会读到不全数据。监控告警等 P1 Prometheus 加完后能看到。
- **W3**（G2 cleanup 分批 5000×30 = 150000 行上限）：单 cron run 最多清 15 万行；积压更多剩余下次清。极端场景留 sys_configs `pm.cleanup.batch_*` 可配。
- 全部 SQL 参数化无注入；事务边界正确；defer rows.Close() 全部到位。
- 前端 typecheck 通过；usePmUserPreferences hook query key 含 technology 防缓存抖动。
- D6 corner case（firmware.changed 不重发 PM 上传配置）—— 用户接受 trade-off。

## Tests

- `Test_CatchupMissedBuckets_*`（8 case）：hourly/daily/weekly/monthly + no-backlog + future + non-advancing + 10k 上限
- `Test_HourlyGroupRunner_Wiring` / `Test_DailyGroupRunner_Wiring` / `Test_WeeklyGroupRunner_Wiring` / `Test_MonthlyGroupRunner_Wiring`
- `Test_GroupRunner_Run_CallsAggregateDeviceGroup`
- `Test_GroupRunner_Run_RejectsMissingPayload`
- `Test_OnlineSubscriber_*`（7 case，含 env 替换 / 失败容错 / 边界）
- `Test_Service_UserPreferences_TechnologyIsolated` + `Test_Repository_UserPreferences_TechnologyIsolated`
- 全 PM 包 + 全 worker 包 + asyncjob 包 go test 过

## Migration Self-check（000164 + 000165）

- 版本号 161 → 162 → 163 → 164 → 165 严格连续
- StatementBegin/End 包裹
- INSERT 与 DDL 列名匹配（本批无 INSERT）
- Down 段删 Up 段全部对象
- 000164: async_jobs_cron_state 简单表，trigger 复用既有 update_updated_at_column
- 000165: 改 PK 含数据迁移（default lte 占位后 drop default），生产无数据是 nop
- 三轮 up/down/up 实测幂等

## DoD（本批）

- [x] go build ./... 全过
- [x] go test ./internal/pm/... ./cmd/worker/... ./internal/core/asyncjob/... 全过
- [x] webcode typecheck 全过
- [x] migration 三轮幂等
- [x] backlog plan-T-0164-followup-gaps.md 落档
- [x] review report 与代码同 commit

## 待办（不在本批）

P1 → P3 共 27 项 + 跨域 2 项剩余，详 plan-T-0164-followup-gaps.md §1。
