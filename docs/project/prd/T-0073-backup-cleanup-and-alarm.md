# PRD: backup cleanup cron + 失败告警发布（T-0073 / R-102 enforcement followup phase 1）

> **关联**: Backlog T-0073 / Sprint-06..07 / Domain=F06/backup / Type=feat
> **作者**: Claude（代 Owner=电信+Go）
> **创建**: 2026-04-29
> **状态**: 草案 → 实施
> **关键决策**: **Phase 1 持久化层增强**（alarm-on-failure 事件 + DB-only cleanup cron）；**Phase 2 = T-0076 followup**（file deletion + storage threshold alarm 等需要 file-storage 路径审计）

---

## 1. 业务背景

T-0071 BackupPolicy 持久化 MVP 已交付（commit `d05831c3`）：19 字段 × 7 类全部存到 DB。但 cleanup / 压缩 / 加密 / 告警 4 类的 **executor enforcement** 拆 follow-up（T-0073/0074/0075）。

T-0073 负责 **cleanup + 告警** 两类的 enforcement。审计后发现真实复杂度跨多个边界：

| 维度 | 现状 |
|------|------|
| Cleanup cron 框架 | 缺；`internal/license/monitor.go` 提供清晰的 cron+AlertSink 模板可借鉴 |
| Backup 文件存储路径 | **未审计**：executor 仅 enqueue TR-069 Upload，CPE 上传到何处由 transfer 模块决定（MinIO 还是别的）— 删除文件需要先审计该路径 |
| 失败告警发布通道 | 已就绪：`event.SubjectAlarmRaised` + alarm engine 订阅 |
| 失败告警 → 邮件路由 | 已就绪（T-0007 EmailDispatcher）但需要 alarm engine 的 filter rule 把 backup-failure 路由到 email |
| 磁盘使用监控 | 缺；需 syscall.Statfs 或 prometheus node_exporter |

---

## 2. ULTRATHINK 决策：Phase 化

将 T-0073 拆为 2 phase；**本 PRD 仅交付 Phase 1**。

### Phase 1（本任务）— 持久化层增强 + 失败事件发布

| 项 | 实现 | 理由 |
|----|------|------|
| Backup 任务失败 → 发布 alarm.raised 事件 | 在 executor failure path（`task.Status = TaskFailed`）读取最新 BackupPolicy；当 `alert_on_failure=true` 发布事件，payload 含 task_id + targetSNs + alertEmail | 直接利用既有 alarm engine + email channel，不引入新通道 |
| Cleanup cron 跑 DB 清理 | 新建 `policy_monitor.go`（仿 license/monitor.go），每日 1 次（hour=0/cron `@daily`）；读 BackupPolicy；DELETE FROM backup_tasks WHERE completed_at < NOW() - retentionDays AND id NOT IN (latest keep_last_N per target) | DB 行清理是**安全的**，不需文件存储路径审计；为 phase 2 提供清理"骨架"|
| 简单 Prometheus 计数器 | `backup_cleanup_runs_total` / `backup_cleanup_rows_deleted_total` / `backup_failure_alarm_published_total` | 运维可见；无需 Grafana 仪表 |

### Phase 2（**T-0076 followup**）— 文件删除 + 存储阈值告警

| 项 | 阻塞原因 |
|----|---------|
| 删除 backup_tasks.file_path 指向的真实文件 | 需先审计 transfer 模块文件存储路径（MinIO 还是 local）+ 跨进程一致性（OMC 节点删了，CPE 还有副本？设计层面） |
| 磁盘使用率超 alertThresholdPercent → alarm | 需引入 syscall.Statfs + node_exporter 集成 |
| Cleanup metrics 仪表盘 | 仪表盘工作，独立任务 |

---

## 3. 用户故事

| 角色 | 故事 |
|------|------|
| 网管运维 | 我希望 backup 任务失败时收到告警邮件（policy.alertEmail），而不是只看后台日志 |
| DBA | 我希望 backup_tasks 表不会无限增长 — 老任务行按保留策略清理 |
| QA | 我希望失败告警 + DB cleanup 都有真测试覆盖 |
| 用户 | 我希望"自动清理"开关真的生效（哪怕 phase 1 只清 DB 行不删文件 — 仍是真生效）|

---

## 4. 验收标准（GWT）

### V1 — Backup 失败发布 alarm.raised
- **Given** BackupPolicy.alertOnFailure = true
- **When** backup task `task.Status = TaskFailed`（executor.go 异常路径）
- **Then** `event.SubjectAlarmRaised` 发布；payload 含 `source="backup"` / `task_id` / `target_count` / `alert_email`；指标 `backup_failure_alarm_published_total` +1

### V2 — Backup 失败但 alertOnFailure=false → 不发布
- **Given** alertOnFailure=false
- **When** task 失败
- **Then** 仅 logger.Warn 不发布事件；指标不递增

### V3 — Cleanup cron @daily 跑一次
- **Given** Monitor.Start(ctx) 已调用
- **When** 时间到 cron 表达式 `@daily`
- **Then** RunCleanupOnce 被触发；测试通过显式调 RunCleanupOnce 验证逻辑

### V4 — Cleanup 删除超过 retentionDays 的行
- **Given** retentionDays=30；DB 内 5 行 backup_tasks，3 行 completed_at = now-40d，2 行 completed_at = now-10d
- **When** RunCleanupOnce
- **Then** 3 行被 DELETE；2 行保留；指标 `backup_cleanup_rows_deleted_total` +3

### V5 — Cleanup 保留 keep_last_n 行（即使老）
- **Given** retentionDays=10；DB 内 7 行 backup_tasks 全 completed_at=now-30d；keep_last_n=3
- **When** RunCleanupOnce
- **Then** 4 行被 DELETE；3 行（最新 3）保留；指标 +4

### V6 — autoCleanup=false → 不清理
- **Given** policy.autoCleanup=false
- **When** RunCleanupOnce
- **Then** 0 行 DELETE；early return；logger.Debug 记跳过

### V7 — Singleton policy fetch within cleanup
- **Given** Monitor 启动后 BackupPolicy 表为空
- **When** RunCleanupOnce
- **Then** 用 DefaultPolicy()（PolicyService.Get 已默认 fallback）；不 panic；按 retentionDays=30 / keep_last_n=5 默认值跑

### V8 — Tests
- 4 monitor 单测：alertSink mock + policy mock + taskRepo mock；cover V3-V7 路径
- 2 executor failure-publishing 单测：mock policyRepo + mock eventBus；cover V1+V2

### V9 — 0 schema 变更
- 本 PRD 不动 migration（cleanup 操作既有 backup_tasks 表 + policy 表已有的字段）；migration 头停留在 000047

---

## 5. 运营商差异矩阵

| 维度 | CMCC | CTCC | CUCC |
|------|------|------|------|
| Cleanup 策略 | 一致 | 一致 | 一致 |
| 告警路由 | 一致（均走 F04） | 一致 | 一致 |
| **实际差异** | **无** | **无** | **无** |

---

## 6. 非目标（Phase 2 / 其他）

- ❌ **删除 backup 任务对应的物理文件**（MinIO/local）→ T-0076 Phase 2，需先审计 transfer 模块
- ❌ **磁盘使用率阈值告警**（policy.alertThresholdPercent 实际生效）→ T-0076 Phase 2，需 disk monitoring
- ❌ **Backup 任务失败告警的邮件路由（alarm engine filter rule "backup_failure → email"）**：本期仅发布 alarm.raised；alarm engine 的 filter rule 配置由 ops 在 dashboard 配（既有能力）
- ❌ **alarm engine 集成测试**（端到端：fail → alarm.raised → email send）→ 现有 alarm engine 已有自己的测试覆盖
- ❌ **Cleanup metrics Grafana 仪表盘** → ops 工作，独立任务

---

## 7. 设计备忘

### 7.1 文件布局

```
omcgo/internal/backup/
├── policy_monitor.go            # 新：cron + RunCleanupOnce + AlertSink interface
├── policy_monitor_test.go       # 新：4 单测 (V3-V7)
├── policy_alarm_publisher.go    # 新：PublishFailureAlarm helper（executor 调用）
├── policy_alarm_publisher_test.go  # 新：2 单测 (V1+V2)
└── executor.go                  # 修：失败路径调 PublishFailureAlarm
```

### 7.2 Monitor 接口（仿 license/monitor.go）

```go
type PolicyMonitor struct {
    policyService *PolicyService
    taskRepo      TaskRepository
    metrics       *PolicyMetrics  // 新：本期 3 counter
    logger        *zap.Logger
    cron          *cron.Cron
    cancel        context.CancelFunc
}

func NewPolicyMonitor(
    policyService *PolicyService,
    taskRepo TaskRepository,
    metrics *PolicyMetrics,
    logger *zap.Logger,
) *PolicyMonitor

func (m *PolicyMonitor) Start(ctx context.Context) error
func (m *PolicyMonitor) Stop()
func (m *PolicyMonitor) RunCleanupOnce(ctx context.Context) (int64, error) // returns rows deleted
```

### 7.3 Cleanup SQL（squirrel + pgx）

```sql
-- Step 1: 选出"待保留的最新 keep_last_n 行 ID"（每个 target 内）
WITH ranked AS (
  SELECT id, ROW_NUMBER() OVER (PARTITION BY target_type ORDER BY completed_at DESC) AS rn
  FROM backup_tasks
  WHERE status = 'completed'
)
-- Step 2: 删除超过 retentionDays AND 不在 keep_last_n 内
DELETE FROM backup_tasks
WHERE status IN ('completed','failed','cancelled')
  AND completed_at < $1
  AND id NOT IN (SELECT id FROM ranked WHERE rn <= $2)
```

参数：`$1 = NOW() - retentionDays days`，`$2 = keep_last_n`

> 注意：`backup_tasks` 是普通表（非分区），DELETE 安全。target_type 是任务级粒度（device/group），单设备多次 backup 都按 target_ids 维度跟踪 → 用 target_type 分区 row_number 是粗粒度；MVP 接受。Phase 2 可改为 per-device 粒度需要 lateral join 到 target_ids JSONB，复杂。

### 7.4 PublishFailureAlarm 接口

```go
// PublishFailureAlarm 发布 alarm.raised 事件，payload 包含 backup task 失败上下文。
// 当 BackupPolicy.AlertOnFailure=false 时 short-circuit。
func PublishFailureAlarm(
    ctx context.Context,
    policyService *PolicyService,
    eventBus event.EventBus,
    metrics *PolicyMetrics,
    task *BackupTask,
) error
```

Payload 示例：
```json
{
  "source": "backup",
  "severity": "major",
  "identifier": "backup_task_failed",
  "summary": "Backup task failed for N target(s)",
  "task_id": "uuid",
  "target_count": N,
  "error_message": "...",
  "alert_email": "ops@example.com"
}
```

### 7.5 Metrics

3 新 Prometheus counter（注册在新 `internal/backup/policy_metrics.go`）：

```
backup_cleanup_runs_total                  counter (success/failure)
backup_cleanup_rows_deleted_total          counter
backup_failure_alarm_published_total       counter
```

### 7.6 DI wiring

```go
// modules.go initBackupModule 末尾
metrics := backup.NewPolicyMetrics(c.MetricsReg)
monitor := backup.NewPolicyMonitor(policyService, backupTaskRepo, metrics, logger)
if err := monitor.Start(context.Background()); err != nil {
    logger.Warn("start backup policy monitor", zap.Error(err))
}
c.miscDeps.backupPolicyMonitor = monitor

// executor 注入 policyService + metrics（构造时）
backupExecutor := backup.NewBackupExecutor(..., policyService, metrics, logger)
```

### 7.7 Executor failure-path patch

```go
// executor.go ~line 159
if successCount == 0 && total > 0 {
    task.Status = TaskFailed
    errMsg := "no devices were successfully queued"
    task.ErrorMessage = &errMsg
    // T-0073 Phase 1: opt-in failure alarm publish
    if e.policyService != nil {
        if pubErr := backup.PublishFailureAlarm(ctx, e.policyService, e.eventBus, e.metrics, task); pubErr != nil {
            e.logger.Warn("publish backup failure alarm", zap.Error(pubErr))
        }
    }
}
```

---

## 8. DoD

- [ ] PRD 七要素全 + 运营商一致矩阵
- [ ] V1-V9 全部测试通过
- [ ] go build / vet / test 全过
- [ ] 0 schema 改动（migration 头不变）
- [ ] 3 新 Prometheus metric 注册 + grep 命中 ≥1
- [ ] 6 单元测试（V1+V2 publisher + V3-V7 monitor）
- [ ] backlog T-0073 → done + Phase 2 followup T-0076 登记
- [ ] R-102 进展更新（enforcement 1/3 部分进入；T-0074/0075 仍 triaged）

---

## 9. 风险评估

| 风险 | 缓解 |
|------|------|
| Cleanup DELETE 误删未结束任务 | WHERE 子句严格 status IN (completed/failed/cancelled)；不动 pending/running |
| keep_last_n SQL 在 backup_tasks 行数巨大时慢 | MVP 接受；future 加 index `(target_type, completed_at)` |
| alarm.raised 大量 backup-failure 触发风暴 | 频率本来就低（计划备份每天最多一次）；不去重 |
| DI 改 NewBackupExecutor 签名破坏既有调用 | 新增参数全 optional（policyService/metrics 可 nil）；nil-safe 内部检查 |
| TaskRepository 缺 cleanup 方法 | 新增 `CleanupOldRows` 接口方法 |

---

## 10. 多皮肤影响

无前端改动；frontend-core 不动；webcode 不动。

---

*PRD by /dev-pipeline pick T-0073 ULTRATHINK A 方案。Phase 化交付：MVP = alarm-on-failure + DB cleanup；file delete + 磁盘阈值留 T-0076。*
