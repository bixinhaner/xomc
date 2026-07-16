# Issue 78 Alarm History Retention 01:08 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 只将 `alarms_history` retention 固定到每天 01:08 Asia/Shanghai，不影响其他定时任务和业务口径。

**Architecture:** 运行时 policy 重建与既有数据库迁移使用同一调度参数。Go 代码保证以后保存配置仍固定相位；TSDB 增量迁移只修改现有 `alarms_history` retention job 的调度属性，不修改 policy config。

**Tech Stack:** Go、pgx/v5、TimescaleDB、Goose、testify/pgxmock。

## Global Constraints

- 不立即删除历史告警。
- 不修改保留天数、清理字段、chunk interval 或前端查询。
- 不修改 `alarms_history` 之外的任何定时任务。
- 迁移不得修改 TimescaleDB job `config`。

---

### Task 1: 固定运行时 policy 相位

**Files:**
- Modify: `omcgo/internal/alarm/history_retention.go`
- Test: `omcgo/internal/alarm/history_retention_test.go`

**Interfaces:**
- Consumes: `TimescaleHistoryRetentionApplier.Apply(ctx context.Context, days int) error`
- Produces: 固定到 01:08 Asia/Shanghai 的 `add_retention_policy` SQL

- [x] **Step 1: 写失败测试**

新增测试，捕获 `add_retention_policy` SQL 并断言包含以下参数：

```sql
schedule_interval => INTERVAL '1 day'
initial_start => TIMESTAMPTZ '2000-01-01 01:08:00+08'
timezone => 'Asia/Shanghai'
```

- [x] **Step 2: 验证 RED**

```bash
cd omcgo
go test ./internal/alarm -run TestTimescaleHistoryRetentionApplierApplyUsesFixedDailySchedule -count=1
```

Expected: FAIL，当前 SQL 仅传入 `drop_after`。

- [x] **Step 3: 最小实现**

将建 policy SQL 提取为常量，并只增加三个调度参数：

```go
const addAlarmHistoryRetentionPolicySQL = `
SELECT add_retention_policy(
    'alarms_history',
    drop_after => $1::interval,
    schedule_interval => INTERVAL '1 day',
    initial_start => TIMESTAMPTZ '2000-01-01 01:08:00+08',
    timezone => 'Asia/Shanghai',
    if_not_exists => TRUE
)`
```

- [x] **Step 4: 验证 GREEN**

```bash
cd omcgo
go test ./internal/alarm -run 'Test.*HistoryRetention' -count=1
```

Expected: PASS。

### Task 2: 只迁移 alarms_history job

**Files:**
- Create: `omcgo/migrations/tsdb/000002_fix_alarm_history_retention_schedule.sql`
- Modify: `omcgo/migrations/README.md`

**Interfaces:**
- Consumes: `timescaledb_information.jobs` 中唯一的 `policy_retention/public/alarms_history` job
- Produces: fixed daily schedule at 01:08 Asia/Shanghai

- [x] **Step 1: 新增 Up 迁移**

先校验目标 job 恰好一条，再执行：

```sql
SELECT alter_job(
    job_id,
    schedule_interval => INTERVAL '1 day',
    fixed_schedule => TRUE,
    initial_start => TIMESTAMPTZ '2000-01-01 01:08:00+08',
    timezone => 'Asia/Shanghai'
)
FROM timescaledb_information.jobs
WHERE proc_name = 'policy_retention'
  AND hypertable_schema = 'public'
  AND hypertable_name = 'alarms_history';
```

- [x] **Step 2: 新增 Down 迁移**

对同一唯一 job 恢复：

```sql
schedule_interval => INTERVAL '1 day',
fixed_schedule => FALSE,
timezone => NULL
```

不得删除或重建 policy。

- [x] **Step 3: disposable TimescaleDB 验证**

执行 baseline → 记录 job `config` → Up → Down，验证：

- Up/Down 均成功；
- `config` 前后不变；
- Up 后 fixed schedule 对齐 01:08；
- 查询中没有其他 hypertable job 被修改。

- [x] **Step 4: 更新迁移索引并验证**

```bash
cd omcgo
go build ./...
go test ./internal/alarm -run 'Test.*HistoryRetention' -count=1
```

Expected: PASS。
