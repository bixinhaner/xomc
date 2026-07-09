# Review: Issue #14 PM 日聚合调度竞态

## Scope

- Fixed point: `main`
- Working tree review: uncommitted diff plus new files
- Spec sources: GitLab Issue `#14`, `tmp/todo15_5g_day_analysis.md`, ADR-0007

## Standards

已检查 `CONTEXT.md`、`docs/adr/README.md`、`docs/project/dod.md`、`docs/expert-personas.md`、`docs/agents/issue-tracker.md`、`docs/agents/triage-labels.md`、`omcgo/migrations/README.md`。

发现并已修复：

- 通用 `asyncjob` 完成日志曾记录原始 `payload/result`，存在敏感信息泄露风险；已移除原始 payload/result 日志，改由 PM runner 输出安全结构化字段。
- `async_jobs` bucket schema 曾同时写入 baseline 和增量迁移；已撤回 baseline 改动，仅保留 `000010` 增量迁移。
- `MarkFailed` 自动 retry 语义缺少测试；已补 `TestRegistry_RunNext_FailureRetriesUntilMaxAttempts`。

保留判断：

- `uq_async_jobs_bucket` 未使用 `CONCURRENTLY`。本迁移已在一次性 PostgreSQL 上演练 `up -> down -> up`；当前本地部署仍处开发期，未对现网大表做在线变更。

## Spec

发现并已修复：

- hourly cron/catchup 入队失败时曾仍可能推进 `cron_state.last_bucket_end`；已改为 enqueue 失败不更新 cron state，catchup 在失败桶处停止推进。
- 聚合完成日志曾无法结构化看到 rows/chain；已在 device/group runner 统一记录 `counter_rows`、`kpi_rows`、`group_rows`、`chained_job_types`、`chained_job_ids`、bucket、duration、status、attempt 和 error。
- bucket 去重只测 SQL 字符串；已补 PostgreSQL 行为测试 `TestPgRepository_InsertBucketDedupe_WithPostgres`，覆盖 pending/running/succeeded 不重复排队，以及 failed/canceled/zombie 恢复 pending。

## Verification

- `go build ./...`
- `go test ./...`
- `go test ./internal/core/asyncjob -run TestPgRepository_InsertBucketDedupe_WithPostgres -count=1 -v` with one-shot PostgreSQL
- schema migration `up -> down -> up` with one-shot PostgreSQL
- `git diff --check`
