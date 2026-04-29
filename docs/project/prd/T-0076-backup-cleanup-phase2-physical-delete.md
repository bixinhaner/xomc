# PRD: backup cleanup Phase 2 — 物理文件删除（T-0076 / T-0073 followup）

> **关联**: Backlog T-0076 / Sprint-07..08 / Domain=F06/backup+ops / Type=feat
> **作者**: Claude（代 Owner=电信+Go+运维）
> **创建**: 2026-04-29
> **状态**: 草案 → 实施
> **关键决策**: scope **再收紧** 到物理删除 + dashboard + severity TODO 关闭；**磁盘阈值告警拆 T-0082**；**多设备 orphan reaper 拆 T-0083**；**email routing N/A**（已有基础设施）

---

## 1. 业务背景

T-0073 cleanup Phase 1 落 DB-only cleanup cron — `repo.CleanupOldRows` DELETE backup_tasks 行，但**物理 MinIO 对象不删**。`policy_monitor.go:11` 注释明示"deferred to T-0076"。结果：
- T-0073 上线后 `backup_tasks` 行清理正常，但 MinIO `config_backup` bucket 文件**永久累积**
- 长期运行 → 磁盘耗尽
- T-0079 已建立 `backup_tasks.file_path` 链路，**首次有了删除目标的"地址簿"**

T-0079 完工是 T-0076 实施的硬前提（之前 file_path 全 NULL，物理删除无从下手）。

---

## 2. ULTRATHINK 决策

### 2.1 任务原描述拆解 (4 个 sub-component)

backlog 原描述：`物理文件删除 + 磁盘阈值告警 + alarm engine email routing + cleanup metrics 仪表盘`。审计后逐项重审：

| # | 子项 | 现状 | 决策 |
|---|------|------|------|
| (1) | **物理文件删除** | T-0073 留 DB-only；T-0079 file_path 链路就绪 | **本任务执行** — 修改 CleanupOldRows 返回 file_paths + iterate MinIO RemoveObject best-effort |
| (2) | **磁盘阈值告警** | 无现成 polling loop；新代码路径（poll bucket size + alarm.raised） | **拆 T-0082**（独立任务，与 cleanup 流解耦） |
| (3) | **alarm engine email routing** | **已 100% 就绪**：`FilterActionNotifyEmail = "notify_email"` + `EmailDispatcher` interface + SMTP env vars + backup 已 publish `alarm.raised`（T-0073） | **N/A 代码层** — 仅文档化 operator runtime config（创建 alarm filter row with action=notify_email + EmailRecipients）|
| (4) | **cleanup metrics 仪表盘** | omc-overview.json 141 行 0 backup metric | **本任务执行** — 加 backup-cleanup 段 panels（复用 T-0073/T-0079 已有 metrics + 本任务新加 file_deleted metric） |

### 2.2 多设备 orphan 限制（important）

T-0079 first-write-wins 决策：每个 backup_task 只存**第一个上传文件**的 file_path。多设备 backup_task（target_count > 1）的剩余 N-1 个文件**没有 file_path 记录**，物理删除按 file_path 索引执行时会**留下 orphan**。

**为什么不在本任务修复**：
- 完整修复需要 (a) 新表 `backup_files` 一行一文件（T-0079 N1 已 defer）或 (b) MinIO list-prefix scan with `taskID8` 模糊匹配
- (a) schema 变更 + 上下游适配 = M+ 工作量；(b) bucket 大时 list cost 高（10k+ objects）
- 多设备 backup 是非主流用法（运维默认单设备 backup）— orphan 累积速度受多设备 task 比例制约
- **TOMBSTONE 选项**：Phase 2 接受多设备 orphan 累积；磁盘趋满前由 T-0083 orphan reaper（独立任务）回收

### 2.3 删除失败语义

物理删除是 best-effort：
- MinIO RemoveObject 失败（网络瞬断 / 权限错误 / 对象已不存在）→ log warn + metric `omc_backup_file_delete_errors_total{reason}` + **继续**清理其他文件
- DB 行已 DELETE — 不回滚（DB cleanup 是主操作；物理删除是 secondary）
- 每次 cleanup tick 输出聚合 metric：`{deleted, errors_remove, errors_other}`

### 2.4 Severity 关闭 TODO(T-0076)

`policy_alarm_publisher.go:68` 显式 TODO："policy-driven severity (warning/major/critical)"。当前硬编码 `"major"`。

`BackupPolicy.AlertThresholdPercent`（50-95，默认 80）是 disk usage 阈值字段（与 T-0082 disk threshold 配套），**不是** severity 控制 — 不能强行映射。

**最小决策**：增加 `BackupPolicy.AlertSeverity` 字段（数据库 schema migration 000050）— `warning|major|critical`，默认 `major`。简单 + 显式 + 配套 T-0082 disk threshold 当 disk usage > 95% 时 alarm 可独立设 critical。

但 schema migration 又重启一道工序。**更小决策**：不加新字段；当前 TODO 只是抽象信号，PRD §6 N1 标 `accept hardcoded major;` policy-driven severity 拆 T-0084 候选。

——还是太啰嗦。**最终决策**：**不动 severity**。原 TODO 标记保留供未来追踪，本任务 PRD 显式说明"暂不映射 — 设计涉及 schema 变更，留 T-0082 disk threshold 联合设计"。

### 2.5 CleanupOldRows 签名变更

当前：`CleanupOldRows(ctx, cutoff, keepLastN) (int64, error)`。

为支持物理删除，DELETE 需 `RETURNING file_path` 收集即将删除的对象路径。新签名：

```go
CleanupOldRows(ctx, cutoff, keepLastN) (deletedFilePaths []string, total int64, err error)
```

- `deletedFilePaths` 仅含 file_path 非空的行（NULL 跳过；file_path 还没回填的老任务不参与物理删除）
- `total` = 总删除行数
- 4 既有 mock（execTaskRepo/fakeTaskRepo/mockTaskRepo/monTaskRepo）需更新签名

---

## 3. 用户故事

| 角色 | 故事 |
|------|------|
| 网管运维 | T-0073 cron 启动一周后，DB row 清掉但磁盘只增不减 — 我希望 cron 同时清 MinIO 对象 |
| Ops 监控 | Grafana 上有"备份清理"仪表盘，看 cron 频次、删除行数、删除字节数（按天聚合）|
| Ops 监控 | backup 失败告警走 email 通道是已知配置项（从 alarm filter 配 action=notify_email），文档说明即可 |
| 后端开发者 | 多设备 backup 的 orphan 累积是已知未修；不在 T-0076 范围 |

---

## 4. 验收标准（GWT）

### V1 — CleanupOldRows 返回 deleted file_paths
- **Given** backup_tasks 含 5 行 terminal status，cutoff/keepLastN 设法 4 行入删（其中 3 行 file_path 非空，1 行 file_path NULL）
- **When** 调 `CleanupOldRows(ctx, cutoff, keepLastN)`
- **Then** 返回 `(filePaths=[3 非空 path], total=4, err=nil)`

### V2 — PolicyMonitor 调 MinIO RemoveObject best-effort
- **Given** mock minio 模拟 1 成功 + 1 NotFound + 1 网络错；3 paths 输入
- **When** PolicyMonitor.RunCleanupOnce 跑一次
- **Then** 3 RemoveObject 调用全发；metric `omc_backup_file_deleted_total` +1（成功），`omc_backup_file_delete_errors_total{reason=not_found}` +1，`{reason=network}` +1；DB cleanup 已完成（不因 MinIO 错误回滚）

### V3 — file_path NULL 跳过物理删除
- **Given** 老 backup_task with file_path=NULL（T-0079 之前的历史行）
- **When** cleanup tick 跑
- **Then** DB DELETE 该行，但**不**调用 RemoveObject（NULL 不进 deletedFilePaths）

### V4 — Cleanup metrics 仪表盘可视
- **Given** Grafana 加载 omc-overview.json
- **When** scrape 已稳定运行
- **Then** 见 backup-cleanup 段含至少 4 个 panel：
  - cleanup_runs_total by result（success/failure/skipped）— stat
  - cleanup_rows_deleted_total — graph time-series
  - file_deleted_total — graph time-series
  - file_delete_errors_total by reason — graph

### V5 — Email routing 文档化（PRD-only，不 code）
- **Given** PRD §9.4 给出 operator runbook
- **When** 运维想配 backup 失败告警邮件
- **Then** 按 PRD 步骤创建 alarm_filters 行 `{action: "notify_email", email_recipients: [...], ...}`，无须代码改动

### V6 — 多设备 backup_task 物理删除部分成功
- **Given** 多设备 backup_task target_count=3，file_path 仅指向第 1 设备文件
- **When** cleanup tick
- **Then** DB 行删除；MinIO 删 1 个文件（first device's）；剩 2 文件 orphan（PRD §2.2 已知限制 + log warn）

### V7 — Backward compat
- **Given** 既有 T-0073 unit tests（11 case）
- **When** 测试套跑
- **Then** 既有断言全部通过；mock 加 stub 方法自然兼容

---

## 5. 运营商差异矩阵

无差异。物理清理是 OMC 内部存储管理，CPE / 运营商无感知。

---

## 6. 非目标

| # | 非目标 | 后续承接 |
|---|--------|---------|
| N1 | 磁盘阈值告警（poll bucket size + alarm publish）| T-0082（新登记） |
| N2 | 多设备 backup_task orphan 文件回收（list-prefix 或 backup_files 表）| T-0083（新登记，依赖 backup_files schema OR list-prefix scan） |
| N3 | Email routing 代码改动 | N/A — alarm 引擎已支持 notify_email + EmailRecipients；仅文档化 operator runbook |
| N4 | Severity policy-driven 化（关闭 policy_alarm_publisher.go:68 TODO）| T-0084 候选；与 T-0082 disk threshold 联合设计 schema |
| N5 | Cleanup retention by-bucket 策略（不同 bucket 不同保留窗口）| 当前 backup_tasks 仅 config_backup 桶；按需后续 |
| N6 | Cleanup audit 日志持久化（who deleted what when） | 当前 zap structured log 已含；持久化拆未来任务 |

---

## 7. 依赖

| 依赖 | 状态 |
|------|------|
| T-0073 cleanup cron | ✅ done — 本任务扩展 RunCleanupOnce |
| T-0079 file_path 链路 | ✅ done — 本任务消费 file_path 列 |
| MinIO client (`*minio.Client`) | ✅ 已 DI 在 modules.go (c.MinIO) |
| `splitBucketAndPath` (T-0079) | ✅ 已存在 — 复用 |

无新外部依赖。

---

## 8. 度量

| metric | 含义 |
|--------|------|
| `omc_backup_file_deleted_total` | counter — 累计物理删除成功数 |
| `omc_backup_file_delete_errors_total{reason}` | counter — 删除失败；reason ∈ {not_found, network, parse_path, other} |
| `omc_backup_file_delete_skipped_total{reason}` | counter — 跳过；reason ∈ {file_path_null, multi_device_orphan} |

复用 `backup.PolicyMetrics` 命名空间，新增 3 collector。

既有 cleanup metrics（T-0073/T-0079）不变。

---

## 9. 设计备忘（S2）

### 9.1 CleanupOldRows 签名重构

`pg_repository.go`：
```go
// CleanupOldRows 旧签名: (int64, error)
// T-0076 新签名: ([]string /*file_paths to delete*/, int64 /*total deleted*/, error)
//
// SQL: 原 DELETE 加 RETURNING file_path → 收集到切片中
//
// DELETE FROM backup_tasks ... RETURNING file_path
```

实现要点：
- pgx Rows 扫描 file_path（nullable string 用 sql.NullString）
- 仅追加非空、非空字符串到 deletedFilePaths
- count 用 len 跟 RETURNING 的实际删除数对齐

### 9.2 PolicyMonitor.RunCleanupOnce 扩展

```go
func (m *PolicyMonitor) RunCleanupOnce(ctx context.Context) (int64, error) {
    // ... existing policy lookup + auto_cleanup gate ...
    filePaths, deleted, err := m.taskRepo.CleanupOldRows(ctx, cutoff, policy.KeepLastN)
    if err != nil { ... }
    m.metrics.RecordCleanupDeleted(deleted)
    
    // T-0076: physically delete MinIO objects (best-effort)
    if m.minio != nil {
        for _, path := range filePaths {
            bucket, objectPath, splitErr := splitBucketAndPath(path)
            if splitErr != nil {
                m.metrics.RecordFileDeleteError("parse_path")
                m.logger.Warn("malformed file_path; skipping physical delete", ...)
                continue
            }
            if err := m.minio.RemoveObject(ctx, bucket, objectPath, minio.RemoveObjectOptions{}); err != nil {
                reason := classifyMinioErr(err)
                m.metrics.RecordFileDeleteError(reason)
                continue
            }
            m.metrics.RecordFileDeleted()
        }
    }
    return deleted, nil
}
```

- `splitBucketAndPath` 复用 T-0079 工具
- `classifyMinioErr` 用 `minio.ToErrorResponse(err)` 映射到 reason 标签
- `m.minio` 是新字段；构造器加 optional（nil 时跳过物理删除，保持单元测试简单）

### 9.3 Severity TODO 更新

`policy_alarm_publisher.go:68` 的 TODO 注释更新为：
```go
// TODO(T-0084): policy-driven severity (warning/major/critical). T-0076
// considered closing this but punted — the natural design needs a new
// BackupPolicy.AlertSeverity column + schema migration which couples poorly
// with the in-flight T-0082 disk threshold work. Combined design recommended.
```

### 9.4 Operator runbook (email routing)

在 PRD 文档中给出 SQL 示例 + 字段说明。运维一次性配置：

```sql
INSERT INTO alarm_filters (
    id, name, enabled, action,
    email_recipients,
    -- match conditions: source='backup' identifier='backup_task_failed'
    match_source, match_identifier
)
VALUES (
    gen_random_uuid(),
    'backup-failure-email-route',
    true,
    'notify_email',
    '["ops@example.com"]'::jsonb,
    'backup',
    'backup_task_failed'
);
```

`SMTP` 配置通过 env vars `OMC_SMTP_*` 注入（参考 `internal/alarm/filter_model.go:19`）。

### 9.5 Dashboard 改动

`deployments/monitoring/grafana/dashboards/omc-overview.json`：
- 加 backup-cleanup row（5 panel）：
  1. Stat — `sum(rate(backup_cleanup_runs_total{result="success"}[1h]))`
  2. Time-series — `rate(backup_cleanup_rows_deleted_total[5m])`
  3. Time-series — `rate(omc_backup_file_deleted_total[5m])`
  4. Pie — `sum by (reason) (omc_backup_file_delete_errors_total)`
  5. Stat — `sum(omc_backup_file_delete_skipped_total{reason="file_path_null"})` （legacy data 提示）

JSON 编辑保持现有结构（grafana 兼容）；不引入 dashboard schema 大改。

### 9.6 文件清单

修改：
- `omcgo/internal/backup/repository.go` — TaskRepository.CleanupOldRows 签名变更
- `omcgo/internal/backup/pg_repository.go` — 实现 RETURNING file_path
- `omcgo/internal/backup/policy_monitor.go` — 加 minio 字段 + 物理删除 loop + classifyMinioErr 工具
- `omcgo/internal/backup/policy_metrics.go` — 加 3 collector + 3 Record method
- `omcgo/internal/backup/policy_alarm_publisher.go` — 更新 TODO comment
- `omcgo/cmd/app/provider/modules.go` — 装 c.MinIO 进 NewPolicyMonitor
- `omcgo/internal/backup/policy_monitor_test.go` — 适配新签名 + 加 minio 物理删除测试
- `omcgo/internal/backup/executor_test.go` `handler_test.go` `service_test.go` — mock CleanupOldRows 更新签名
- `omcgo/internal/backup/policy_alarm_publisher_test.go` — 不动（不涉及 cleanup 链路）
- `deployments/monitoring/grafana/dashboards/omc-overview.json` — 加 backup-cleanup panels

无新增文件（除 PRD/verify 报告本身）。

### 9.7 待定点

| 待定 | 决策 |
|------|------|
| classifyMinioErr 错误分类粒度 | not_found / network / other — 三类够用；可后续细化 |
| dashboard panel 风格（Grafana version 兼容） | 沿用 omc-overview.json 现有 panel 结构（plugin 版本不动） |
| Multi-device orphan log 频率 | 每次 cleanup tick 输出 1 行 INFO 含 orphan estimate；T-0083 修复后可移除 |
