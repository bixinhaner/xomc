# Review Report — T-0164-P8 G8 worker wiring 收尾

- **Branch**: draft/pm-kpi-impl
- **Scope**: worker
- **Backlog**: T-0164-P8
- **Date**: 2026-05-23
- **Author**: shangyingbin (kevin)
- **Reviewer**: Claude (AI self-review)

---

## Conclusion

**PASS_WITH_WARNINGS** — 可合入。

- 0 CRITICAL
- 2 WARNING（生产时区 + 进程重启 cron 跳次，均可后续优化）
- 3 INFO

`go build ./... && go test ./cmd/worker/...` 全过；依赖 T-0164-P5 commit `3a8ce4ec` 提供的 aggregator 包。

---

## Files Changed

| Path | LOC | Type |
|------|-----|------|
| `cmd/worker/aggregator.go` | +200 | new |
| `cmd/worker/main.go` | +5 | mod |
| `docs/project/backlog/subtasks/T-0164-pm-kpi-pipeline.md` | +1/-1 | mod |

---

## Findings

### CRITICAL — 0

无。

### WARNING — 2

#### W1 — cron 用本地时区（生产 docker UTC 一致；开发环境东八区 +8h 偏移）

**File**: `cmd/worker/aggregator.go`

`cron.New()` 默认按 wall-clock 本地时区。生产 docker compose 已设 TZ=UTC，cron 触发时刻与
UTC 桶对齐。开发宿主机（CST）触发时刻会落在 18:05/08:05 等"看起来奇怪"的本地时刻，但
触发回调内 `time.Now().UTC().Truncate(time.Hour)` 算出的 bucket 仍是正确的上一 UTC 整点。

**Mitigation**：触发时刻偏移不影响正确性，只影响"什么时候开跑"。

**Action**：本次不强约束；后续可显式 `cron.New(cron.WithLocation(time.UTC))` 锁定。

#### W2 — 进程重启期间错过 cron 触发时刻 → 该 bucket 不会自动补算

**File**: `cmd/worker/aggregator.go`

robfig/cron/v3 没有"开机时补跑错过的触发"语义；如果 worker 在 11:00-11:10 重启，
11:05 的 hourly 触发会被跳过，pm_metrics_hourly 缺该桶数据。

**Mitigation**：
- 长期：G7 自定义聚合任务支持"指定 bucket 范围"，可手动触发补算（已规划）。
- 短期：sysadmin 可手动 INSERT async_jobs 重跑（payload 格式即 BuildPayload 输出）。

**Action**：本次不修；T-0164-P7 / G7 自定义聚合任务覆盖此需求。

### INFO — 3

- **I1**: cron 触发器只 INSERT async_jobs，不直接调 runner，解耦正确（多 worker 跨进程靠 LockNextPending SKIP LOCKED 自然分配）。
- **I2**: 每个 JobType 单独 worker goroutine 5s tick，相比共享 goroutine + jobType 轮询，更易隔离故障 + 不会出现"快 jobType 饿死慢 jobType"。
- **I3**: Sweeper 异步运行（context.Background 子 ctx），worker shutdown 时优雅退出。

---

## DoD

- [x] go build ./... 通过
- [x] go test ./cmd/worker/... 通过
- [x] backlog T-0164-P8 状态 dev_done_pending_review（worker wiring 完成）
- [x] review report 与代码同 commit
- [ ] docker 重新部署 + 验证 cron 触发（留早上 review）

---

## Out of scope

- docker rebuild 验证 cron 真实触发 → 早上 review 时跑 deploy skill
- 集成测试新 worker 入队 → 已被 T-0164-P5 整合到 aggregator.integration_test.go
- 手动重跑接口 → T-0164-P7 / G7 覆盖
