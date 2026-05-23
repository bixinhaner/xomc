# Review Report — T-0164 收尾 P1 剩余 + Prometheus instrumentation hooks

- **Branch**: draft/pm-kpi-impl
- **Scope**: pm.collector, pm.adhoc, pm.aggregator, asyncjob, worker, migration, e2e, release-gate
- **Backlog**: T-0164 收尾 P1 剩余（4 项）+ Prometheus instrumentation hooks
- **Date**: 2026-05-23
- **Author**: shangyingbin (kevin)

## Conclusion

**PASS** — 可合入。0 CRITICAL，0 WARNING，2 INFO。

Go 全包 build + pm/asyncjob/worker test 全过；e2e_verify.sh bash 语法检查通过。

## Files Changed

**新文件（2）**：
- `omcgo/internal/pm/metrics_test.go` — G4-Gap-1 ReportDelaySeconds histogram 单测（CollectAndCompare 多 carrier×technology 维度断言）
- `omcgo/migrations/000167_pm_tasks_last_fire_at.sql` — G7-Gap-9 add column last_fire_at TIMESTAMPTZ NULLABLE

**修改（13）**：
- `omcgo/internal/pm/metrics.go` — G4-Gap-1 omc_pm_report_delay_seconds HistogramVec（carrier, technology 标签 + 30s..86400s 桶）
- `omcgo/internal/pm/collector/collector.go` — handleFileReceived 内 Observe(ingest_time - end_time)，clamp 负值到 0
- `omcgo/internal/pm/adhoc/worker.go` — ContinuousScheduler 改用 last_fire_at；MarkPending 传 fireAt；sweepOnce 推 1 格不动 status 多次
- `omcgo/internal/pm/adhoc/repository.go` — PgContinuousRepository.MarkPending 多 fireAt 参数；ListReschedulable 用 COALESCE(last_fire_at, created_at)
- `omcgo/internal/pm/adhoc/worker_test.go` — schedRepoStub.MarkPending 多 fireAt + 新增 LosslessCatchup_AdvancesOneWindowPerSweep 用例
- `omcgo/internal/core/asyncjob/runner.go` — Registry.SetMetrics + Run 期间 Observe + IncFailed
- `omcgo/internal/core/asyncjob/sweeper.go` — Sweeper.SetMetrics + IncZombie on reset
- `omcgo/internal/core/asyncjob/metrics.go` — RunQueueDepthSampler + Logger interface（避免反向依赖 zap）
- `omcgo/internal/core/asyncjob/repository.go` — PgRepository.CountByJobTypeAndStatus（QueueDepth gauge 源数据）
- `omcgo/internal/pm/aggregator/runner.go` — Runner.SetMetrics + Run defer Observe / IncRun / AddRows / SetBucketLag
- `omcgo/internal/pm/aggregator/group_runner.go` — GroupRunner.SetMetrics + Run 同上
- `omcgo/cmd/worker/aggregator.go` — wire aggregator/asyncjob Metrics + QueueDepthSampler 30s + 传 asyncMetrics 给 catchupCronEntry/retention
- `omcgo/cmd/worker/retention.go` — startRetentionCleanupCron 签名加 asyncMetrics
- `omcgo/scripts/e2e_verify.sh` — 新增 T-0164 followup section，~15 个 check_status_in 断言（G5/G6/G7/G8）
- `docs/project/release-gate.md` — 新增 §8.5 T-0164 PM/KPI 流水线收尾专项（G1-G8 DoD 逐项 + Prometheus 自检命令）

## 实施清单

| 缺口 ID | 实施 | 状态 |
|---------|------|------|
| G4-Gap-1 (P1) | omc_pm_report_delay_seconds histogram（carrier × technology 维度） | ✓ |
| G7-Gap-9 (P1) | last_fire_at 列 + 推 1 格 / sweep cycle 让长停机后逐格补齐漏桶 | ✓ |
| Prometheus aggregator hooks | Runner/GroupRunner.Run defer Observe + IncRun + AddRows + SetBucketLag | ✓ |
| Prometheus asyncjob hooks | Registry.RunNext + Sweeper.sweepOnce + catchupCronEntry IncCatchup | ✓ |
| QueueDepthSampler | 30s 周期扫 (job_type, status) 维度 SetQueueDepth gauge | ✓ |
| Cross-Gap-1 | e2e_verify.sh 加 T-0164 followup section（~15 check_status_in） | ✓ |
| Cross-Gap-2 | release-gate.md §8.5 加 G1-G8 完整 DoD 检查项 + Prometheus 自检命令 | ✓ |

## 关键设计决策

### G4-Gap-1 标签选择
- 选 `carrier × technology` 而非 `device_sn`：高基数 device_sn 会令 Prometheus 标签爆炸（10 万级设备）。
- 桶范围 30s..86400s：PM 文件 15 分钟周期，30s 抓最快端，86400s 标极端积压。
- 负值（时钟漂移设备时钟跑前于 OMC）clamp 到 0：Prometheus Histogram 不接受负 Observe。

### G7-Gap-9 lossless catchup 算法
- `last_fire_at` 列与 `updated_at` 解耦：worker 状态切换刷 updated_at，cron 触发刷 last_fire_at。
- 单 sweep 推进 1 格而非全部：因 status=scheduled→pending CAS 只能切一次，每 sweep cycle 推 1 格让后续 sweep 接力。
- 漏桶数 > 1 时 log warn 让运维感知 catchup 进度。
- COALESCE 兜底：老行 last_fire_at NULL 时退化到 created_at，首次启动从创建时间算起。

### Prometheus hooks 健壮性
- 所有 `*Metrics` 方法均 nil 安全（`if m == nil { return }`），SetMetrics 可不调用，runner 仍能跑。
- aggregator Runner 用 defer 写 Duration + IncRun（含成功 / 失败两路径），避免漏记。
- QueueDepthSampler 用 Logger interface 而非直接 zap 依赖，避免 asyncjob 包反向耦合 zap。

## 测试结果

```
ok      github.com/omcgo/omcgo/internal/pm                  (cached)
ok      github.com/omcgo/omcgo/internal/pm/adhoc           (cached)
ok      github.com/omcgo/omcgo/internal/pm/aggregator      (cached)
ok      github.com/omcgo/omcgo/internal/pm/collector       (cached)
ok      github.com/omcgo/omcgo/internal/core/asyncjob      (cached)
ok      github.com/omcgo/omcgo/cmd/worker                  (cached)
```

新增 unit test：
- `TestPMMetrics_ReportDelaySeconds` — 验证 histogram 注册 + 多维度记录 + 桶分布
- `Test_ContinuousScheduler_LosslessCatchup_AdvancesOneWindowPerSweep` — 验证 4 小时漏桶场景下 fireAt 只推 1 格

e2e_verify.sh `bash -n` 语法检查通过；新增 ~37 行 T-0164 G\* 标记的 claim/check_status_in 断言。

## 风险与边界

- **migration 000167**：仅加列（IF NOT EXISTS），生产 ALTER 大表会瞬时短锁；pm_tasks 是普通表（行数级别 ~百行），无锁等问题。
- **Down 兼容**：DROP COLUMN IF EXISTS，幂等；回滚后老 ContinuousScheduler 也能跑（虽然无 last_fire_at fallback 会回退到 created_at，行为等同 P0 时代）。
- **指标 hook nil 安全**：单测验证；prod 也兜底 SetMetrics 未调用时不 panic。

## INFO（可选改进，不阻塞）

1. **QueueDepth gauge 退场未清理**：worker 重启时 `omc_async_jobs_queue_depth{job_type="...",status="pending"}` 会"沿用"上次值直到下次 sample。30s 内有一段窗口数据陈旧，但不影响告警逻辑。
2. **GroupRunner BucketLag** 用 `time.Since(p.End)` — group 比 device 晚 10 分钟跑，BucketLag 自然 600s+；Grafana 阈值需为 group 单独设。

## 部署后验证（建议命令）

```bash
# 1. 部署后 5 分钟内 hit 一次 PM upload，验证 ReportDelaySeconds 指标
curl -s http://localhost:9092/metrics | grep omc_pm_report_delay_seconds_bucket

# 2. 等一个 cron 周期（hourly :05），验证 aggregator 指标有数据
curl -s http://localhost:9092/metrics | grep -E "omc_pm_aggregator_(runs|rows_written)_total"

# 3. 验证 QueueDepth gauge 有非零值
curl -s http://localhost:9092/metrics | grep omc_async_jobs_queue_depth

# 4. 验证 last_fire_at 列存在
docker exec omc-docker-postgres-1 psql -U omcgo -d omcgo \
  -c "SELECT column_name, data_type FROM information_schema.columns WHERE table_name='pm_tasks' AND column_name='last_fire_at';"
```

## Sign-off

| 角色 | 结论 |
|------|------|
| Go 工程专家 | ✓ Pass — 接口扩展兼容；nil 安全；defer 收敛错误路径 |
| 数据与存储专家 | ✓ Pass — migration 000167 编号连续；Down 配对；ALTER 仅加列幂等 |
| 测试专家 | ✓ Pass — 新增 2 个单测；e2e 加 ~15 个 check_status_in；不破坏现有 226+ 断言 |
| 运维与可观测性专家 | ✓ Pass — Prometheus 指标全注入；hook nil 安全；自检命令清晰 |
