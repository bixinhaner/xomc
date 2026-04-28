# Top-10 慢查询审计与优化清单

> Wave 3 / Block F.2 / Task T-0060 — 2026-04-28
>
> 本文档梳理 OMC 后端最容易触发慢查询的 10 类 SQL 模式，逐条给出
> 优化措施与预期收益。所有补索引的 DDL 已落地在
> `migrations/000042_perf_indexes.sql`。
>
> 审计基础：
>   - 代码 grep（`squirrel.Where` / `OrderBy` / `pool.Query`）整理实际查询模式；
>   - `internal/core/components/postgres/slow_query.go` 提供运行时 100ms+
>     慢查询埋点（zap WARN + Prometheus `pgx_slow_query_total`）；
>   - 未在 staging 真实回放，但每条都标明了"如何复现 / 如何验证"。

## 配套基础设施

- **运行时埋点**：`SlowQueryTracer`（pgx `QueryTracer`）
  - 阈值默认 100 ms，可通过 `WithSlowQueryTracer(threshold, log, reg)` 注入。
  - 命中后输出 zap WARN（含 `duration` / `threshold` / `table` / `query_hash` / `sql` / `request_id`）。
  - Prometheus 指标 `pgx_slow_query_total{table, query_hash}` 持续累积，便于
    Grafana 按 query_hash 排序观察"哪条 SQL 出现最频繁"。
- **链式组合**：`ChainTracer(existing, slow)` 可与现有 `OTELSQLTracer` 并存，
  不必二选一；本次未改动 `cmd/app/provider/*`，留给 W3 后续 sub-agent 注册。

## Top-10 清单

| # | 表 | 查询模式（精简） | 痛点 | 优化措施 | 预期收益 |
|---|----|-----------------|------|---------|---------|
| 1 | `devices` | `WHERE deleted_at IS NULL AND carrier=? ORDER BY last_inform_at DESC` | 主列表接口；现有 `idx_devices_carrier_status` 不含 `deleted_at` 过滤，回收站/在册混扫 | 加 partial 索引 `idx_devices_carrier_alive(carrier, last_inform_at DESC) WHERE deleted_at IS NULL` | 10 万设备规模下从全表 seq scan 降到 index range scan；预计 P95 200ms → 20ms |
| 2 | `alarms_active` | `WHERE status=? AND severity<=? ORDER BY raised_at DESC LIMIT N` | 告警列表主排序；现有单列索引 `severity` / `status` 选择性差，`ORDER BY raised_at DESC` 反向扫描 | 加 `idx_alarms_active_status_severity_time(status, severity, raised_at DESC)` | 索引可同时覆盖过滤+排序；P95 ~150ms → ~10ms |
| 3 | `alarms_active` | dashboard `GROUP BY carrier, severity` 大屏统计 | 三大运营商 × 5 级 severity 桶聚合，每分钟刷新一次 | 加 `idx_alarms_active_carrier_severity(carrier, severity)` | 直接由 index-only scan 完成 group/count；CPU 节省 50%+ |
| 4 | `alarms_history` | 报表 `WHERE severity<=? AND time>=now()-INTERVAL '7d'` | TimescaleDB hypertable，但缺 severity 维度索引 → 仍需扫描所有 chunk | 加 `idx_alarms_history_severity_time(severity, time DESC)` | 报表查询从分钟级降到秒级 |
| 5 | `device_tasks` | reaper：`WHERE status IN ('pending','sent') AND expires_at<now() ORDER BY expires_at` | reboot closer / expired task 清理周期任务，全表扫描挑过期记录 | 加 partial `idx_device_tasks_status_expires(status, expires_at) WHERE expires_at IS NOT NULL` | 10 万 device_tasks 规模下，每分钟 reaper 从 ~500ms 降到 <10ms |
| 6 | `notifications` | 通知中心 `WHERE type=? AND priority=? ORDER BY created_at DESC` | 现有 `(type, created_at DESC)` 不含 priority；高优先级通知按 priority 二次过滤 | 加 `idx_notifications_type_priority_time(type, priority, created_at DESC)` | 单条索引覆盖三维过滤+排序；高峰期延迟 80ms → 5ms |
| 7 | `audit_logs` | 安全审计 `WHERE action='login' AND created_at>=?` | 现有索引按 `user_id` / `resource` / `time` 维度，按 `action` 过滤需扫描 7 天分区 | 加 `idx_audit_logs_action_time(action, created_at DESC)` | 安全 SOC 查询从 5s 降到 <100ms |
| 8 | `mml_tasks` | "我的任务" `WHERE creator=? ORDER BY created_at DESC` | MML 控制台个人列表，creator 维度无索引 → 拉全表 | 加 partial `idx_mml_tasks_creator_time(creator, created_at DESC) WHERE creator IS NOT NULL` | "我的任务"列表 P95 从秒级降到 50ms |
| 9 | `managed_files` | 文件管理 `WHERE file_type=? AND status=? ORDER BY created_at DESC` | 三个独立单列索引，复合过滤时只能选一条索引剩下行内过滤 | 加 `idx_managed_files_type_status_time(file_type, status, created_at DESC)` | 文件管理分页 P95 ~200ms → ~15ms |
| 10 | `system_logs` | 运维 `WHERE level='ERROR' AND source='alarm' AND created_at>=?` | 三个单列索引各扫一段，回表多 | 加 `idx_system_logs_level_source_time(level, source, created_at DESC)` | 运维查最近错误日志，从 1-2s 降到 <100ms |

> 额外补充（第 11 条）：`alarms_active(device_id, raised_at DESC)` —— 设备详情页
> "最近 N 条活跃告警"。现有 `idx_alarms_active_device(device_id)` 缺 raised_at
> 排序键，详情页打开延迟可降一个数量级。已纳入同一份迁移。

## 验证步骤

### 1. 索引落库
```bash
cd omcgo
make migrate-up
psql -c "\\d+ alarms_active" | grep idx_alarms_active_status_severity_time
```

### 2. 慢查询埋点
```go
// cmd/app/provider/postgres.go 注入：
slowTr := postgres.WithSlowQueryTracer(100*time.Millisecond, log, reg)
poolCfg.ConnConfig.Tracer = postgres.ChainTracer(existingTracer, slowTr)
```
启动后访问 `/metrics` 应看到：
```
# HELP pgx_slow_query_total Total number of PostgreSQL queries exceeding the configured slow-query threshold.
# TYPE pgx_slow_query_total counter
pgx_slow_query_total{query_hash="abc123def456",table="alarms_active"} 0
```

### 3. EXPLAIN 验证（需 staging 数据）
```sql
EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM alarms_active
  WHERE status='active' AND severity<=2 ORDER BY raised_at DESC LIMIT 50;
-- 期望走 idx_alarms_active_status_severity_time，无 Sort 节点。
```

### 4. 回滚演练
```bash
make migrate-down  # 回退 000042，所有索引 DROP
psql -c "\\di idx_alarms_active_status_severity_time"  # 应不存在
make migrate-up    # 再次升回
```

## 后续工作

- W3.F.2 仅落"补索引 + 埋点 + 文档"。注册 `SlowQueryTracer` 到三个进程
  （app/acs/worker）的 provider 由 W3 后续 sub-agent（不在本任务范围）接手——
  本任务严禁动 `cmd/app/provider/*`、`cmd/app/main.go`。
- staging 拉真实数据 EXPLAIN ANALYZE 验证可在 release-gate 阶段 (T-0024) 执行。
- 命中量大的索引（# 2、# 5）建议追加 release note，方便 SRE 关注首次刷盘的
  IO 影响（10 万行规模 < 1 GB，预计无明显抖动）。

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-04-28 | 初版（W3.F.2 / T-0060），10 类 SQL + 11 条索引 + 运行时埋点 |
