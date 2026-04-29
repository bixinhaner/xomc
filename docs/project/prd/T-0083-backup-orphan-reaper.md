# PRD: backup multi-device orphan 文件 reaper（T-0083）

> **关联**: Backlog T-0083 / Sprint-08 / Domain=F06/backup / Type=feat / Prio=P3
> **作者**: Claude（代 Owner=Go）
> **创建**: 2026-04-29
> **状态**: 草案 → 实施
> **关键决策**: list-prefix scan 而非新表 backup_files；@weekly cron；30 天 age 安全网；max-K 上限防 unbounded API；mirror RunStorageCheckOnce / RunCleanupOnce 既有 PolicyMonitor 模式

---

## 1. 业务背景

T-0079 first-write-wins：每个 backup_task 仅写入第 1 个完成上传的文件 path 到 `backup_tasks.file_path`，其他 N-1 个 multi-device 文件**无 path 索引**。
T-0076 cleanup Phase 2 物理删除按 `file_path` 删 → 多设备 backup_task 仅删 1 个文件，剩 N-1 个 **永久 orphan 在 MinIO**。

T-0076 PRD §2.2 explicitly punted 到 T-0083：
- 完整修复需 (a) 新表 `backup_files` 一行一文件 OR (b) MinIO list-prefix scan with taskID8 模糊匹配
- 当前 multi-device backup 是非主流用法，orphan 累积速度慢 — Phase 2 接受暂留

T-0083 闭环此长尾：定期 list 全 bucket，cross-check live taskID8 set，reap 不在 live set 且年龄 ≥ 30 天的对象。

---

## 2. ULTRATHINK 决策

### 2.1 list-prefix vs 新表 backup_files

| 选项 | 优 | 劣 |
|------|---|---|
| A. list-prefix scan + taskID8 cross-check | 0 schema 变更；与 T-0089 storage check 同思路 | O(n) bucket scan；大 bucket 慢 |
| B. 新表 backup_files 一行一文件 | precise；O(rows) 即可索引到全部 file_paths | schema migration + executor 双写 + N+ test 改动；scope L+ |

**采纳 A**。理由：
- T-0083 P3 / M 任务，scope 控制
- multi-device backup 是非主流，orphan 累积慢 → list bucket @weekly 即可消化
- T-0089 storage check @hourly 已用 list-prefix，证明该模式 OK

option B 是明确的未来任务（T-0083 PRD 也可标 followup 候选）。

### 2.2 @weekly cron 频率

orphan 累积速率：仅 multi-device backup 创造 orphan，每 task N-1 个文件。运维侧 multi-device 占比 < 20%，即每周 orphan 增量约 (multi-task 数 × 平均 N-1) 文件。

频率选项：
- @daily：与 cleanup 同频，太密
- **@weekly**：housekeeping 性质，每周一次足够
- @monthly：太稀，长期堆积过 retention 阈值

采纳 @weekly。PolicyMonitor.Start 加第 3 个 cron schedule。

### 2.3 30 天 age 安全网

risk：reap race — list bucket 时新 backup 完成上传，新文件被误判 orphan。

解决：reap 仅作用于 LastModified < (now - 30 天) 的对象。理由：
- T-0073 cleanup 默认 retention_days=30 — DB 行 30 天后被删
- 30 天后 backup_task DB 已不存在 → live set 不含其 taskID8 → 文件确实是 orphan
- 30 天前的对象意味着上传完成早 30 天，无 race

**与 cleanup 自然一致**：T-0073 cleanup 以 retention_days 为基准 → DB 行 retention_days+ 后被 DELETE → 文件 retention_days+ 后变 orphan candidate。

为避免 retention_days != 30 的部署被卡，age 安全网读 `policy.RetentionDays + 1`（默认 31 天），超出该值的 orphan 才 reap。

### 2.4 max-K 上限

list MinIO 不分页（minio-go 流式 channel），但 reap 调用是 N 次 RemoveObject API call。N 失控（如 bucket 一次性挤满 100K orphan）会产生大量网络流量 + MinIO 压力。

**采纳 max 1000 reap per run**。超出限额 log info "more orphans pending; will reap next tick"。每周 1000 = 每年 52000 上限，足够覆盖运营期 orphan rate。

### 2.5 filename pattern parse

T-0079 写入文件名：`backup-{taskID8}-{deviceSN}.xml(.compressed)?(.enc)?` 含两个可选 suffix。

reap parse pattern：
```regex
^backup-([0-9a-f]{8})-(.+?)\.xml(\.[a-z0-9]+)?(\.[a-z0-9]+)?$
```

(group 1) = taskID8，cross-check live set。

**不匹配 pattern 的对象不动**（保险默认）— 如运维手工放置的文件 / 其他用途的对象。

### 2.6 Live taskID8 set 构建

reaper 跑前一次性 SELECT id FROM backup_tasks → 转成 set of taskID8 prefixes（ID 前 8 hex chars）。

set 大小：百万级 backup_tasks → 8 字节 string × 1M ≈ 8MB heap，OK。

**注意**：list 期间新增的 backup_task 不在 live set。但因 30 天 age 安全网，新 task 的文件不会被 reap（< 30 天）。

### 2.7 metrics

2 新 collectors 进 backup.PolicyMetrics：
- `omc_backup_orphan_reaped_total` (counter) — 累计成功 reap 数
- `omc_backup_orphan_skipped_total{reason}` (counter) — reason ∈ {pattern_mismatch, live_task, age_recent, api_error}

可选：`omc_backup_orphan_scan_duration_seconds` (histogram) — 单次 scan 时长分布。

### 2.8 与 T-0076 cleanup 的协作

T-0076 RunCleanupOnce 处理 file_path 已记录的删除；剩 NULL/missing path 的对象由 T-0083 reap。两个 cron 互不影响：
- @daily RunCleanupOnce 删 DB rows + 已知 file_paths 对应 MinIO
- @weekly RunOrphanReaperOnce 兜底剩余 multi-device orphan

**无 race**：cleanup 与 reaper 的删除目标不重叠（cleanup 目标在 file_paths 集，reaper 目标在 live-set 之外的 30+ 天对象）。

---

## 3. 用户故事

| 角色 | 故事 |
|------|------|
| Ops 监控 | Grafana 看 `omc_backup_orphan_reaped_total` 增长曲线 — 长期趋势平稳，说明 multi-device orphan 在持续清 |
| 运维 | 每周一固定看 reaper run log — 失败时（api_error）能从 metric 反查根因 |
| 后端开发 | T-0079 first-write-wins 决策 + T-0083 reaper = 完整 multi-device 闭环；不需引入 backup_files 表 |
| QA | reaper 不会误删 < 30 天对象（即使 multi-device backup 真的还在跑）|

---

## 4. 验收标准（GWT）

### V1 — orphan reap 成功路径
- **Given** bucket 含 1 文件 `backup/2026/03/backup-aabbccdd-SN999.xml.gz`，taskID8="aabbccdd" 不在 live set，文件 LastModified < now - 31d
- **When** RunOrphanReaperOnce 跑
- **Then** RemoveObject 调；`orphan_reaped_total` +1

### V2 — live task 不 reap
- **Given** 文件 taskID8="11223344"，live set 含 "11223344"
- **When** RunOrphanReaperOnce
- **Then** RemoveObject **不**调；`orphan_skipped_total{reason="live_task"}` +1

### V3 — < 30 天对象不 reap
- **Given** 文件 LastModified = now - 5d，taskID8 不在 live set
- **When** RunOrphanReaperOnce
- **Then** RemoveObject 不调；`orphan_skipped_total{reason="age_recent"}` +1

### V4 — pattern 不匹配不 reap
- **Given** 文件 `unrelated/manual-upload.xml` 不匹配 pattern
- **When** RunOrphanReaperOnce
- **Then** RemoveObject 不调；`orphan_skipped_total{reason="pattern_mismatch"}` +1

### V5 — max-K 上限
- **Given** bucket 含 1500 orphan candidates
- **When** RunOrphanReaperOnce 一次
- **Then** RemoveObject 调 ≤ 1000 次；log info "max reap per run reached; remainder pending next tick"

### V6 — RemoveObject API error 不 abort scan
- **Given** 3 candidates，第 2 个 RemoveObject 返 network error
- **When** RunOrphanReaperOnce
- **Then** 3 个 candidate 都被尝试；`orphan_skipped_total{reason="api_error"}` +1

### V7 — list bucket fail-open
- **Given** ListObjects 返 err
- **When** RunOrphanReaperOnce
- **Then** 不 panic，返 err；no orphan 被误删

### V8 — bucketLister/listTaskIDs nil-safe
- **Given** PolicyMonitor 未 SetBucketLister 或 taskRepo 不可用
- **When** RunOrphanReaperOnce
- **Then** no-op return（与 T-0082 storage check 一致策略）

### V9 — Cron 集成
- **Given** PolicyMonitor.Start
- **When** cron 启动
- **Then** Entries() 含 3 个（@daily cleanup + @hourly storage + @weekly reaper）

### V10 — backwards compat — 既有 11 PolicyMonitor case 全过
- **Given** 既有 cleanup + storage tests
- **When** go test -race ./internal/backup/...
- **Then** 全 PASS

---

## 5. 运营商差异矩阵

无差异。orphan reaper 是 OMC 内部 housekeeping。

---

## 6. 非目标

| # | 非目标 | 原因 |
|---|--------|------|
| N1 | 新表 backup_files 持久化每文件 path | 大 scope；future 任务（如有需求） |
| N2 | reap 触发实时（CPE upload 失败时立即 cleanup）| @weekly 即可；real-time 过度工程 |
| N3 | reaper 失败 alarm 通知 | metric 已有；future 加 alarm 当 api_error 持续 > N |
| N4 | configurable cron 频率（env var）| 默认 @weekly 合理；future 加 |
| N5 | encrypted (.enc) / compressed (.gz/.zst) suffix 路由分别处理 | regex 已 cover；treat as 同等 orphan |

---

## 7. 依赖

| 依赖 | 状态 |
|------|------|
| T-0076 cleanup Phase 2 | ✅ done — file_path 链路已建 |
| T-0079 file_path 链路 | ✅ done — 提供 live taskID8 set 来源 |
| T-0082 storage check / list bucket pattern | ✅ done — mirror BucketLister 接口 |
| `MinIO.ListObjects` | ✅ |

---

## 8. 度量

§2.7 已列。

---

## 9. 设计备忘（S2）

### 9.1 PolicyMonitor 扩展

新增 `RunOrphanReaperOnce(ctx) (reaped int, err error)` 方法（mirror RunCleanupOnce / RunStorageCheckOnce shape）。

### 9.2 接口要求

复用 T-0082 BucketLister。新增 narrow consumer iface for live-set 查询：

```go
// TaskIDLister 返回 backup_tasks 表所有 task ID 的 8-char hex 前缀集合。
// PolicyMonitor 在 reap 前一次性构建 live set，避免 O(N) 单查。
type TaskIDLister interface {
    ListAllTaskIDPrefixes(ctx context.Context) (map[string]struct{}, error)
}
```

实施在 `pg_repository.go`：`SELECT REPLACE(id::text, '-', '') ... LIMIT 8` 类型的 query，或 `SUBSTRING(REPLACE(id::text, '-', ''), 1, 8)` 一次性 SELECT DISTINCT。

### 9.3 Reaper 流程

```go
func (m *PolicyMonitor) RunOrphanReaperOnce(ctx context.Context) (int, error) {
    if m.bucketLister == nil || m.minio == nil || m.taskIDLister == nil {
        return 0, nil
    }
    // 1. Build live set
    liveSet, err := m.taskIDLister.ListAllTaskIDPrefixes(ctx)
    if err != nil { return 0, err }
    
    // 2. Get retention threshold
    policy, err := m.policyService.Get(ctx)
    if err != nil { return 0, err }
    cutoff := time.Now().Add(-time.Duration(policy.RetentionDays+1) * 24 * time.Hour)
    
    // 3. Scan + classify + reap (max 1000)
    reaped := 0
    skipped := map[string]int{}
    for obj := range m.bucketLister.ListObjects(ctx, CanonicalRestoreBucket, minio.ListObjectsOptions{Recursive: true}) {
        if obj.Err != nil { return reaped, obj.Err }
        
        if reaped >= maxReapPerRun {
            m.logger.Info("orphan reaper max per run; remainder next tick")
            break
        }
        
        taskID8, ok := parseBackupFilename(obj.Key)
        if !ok {
            skipped["pattern_mismatch"]++
            continue
        }
        if _, isLive := liveSet[taskID8]; isLive {
            skipped["live_task"]++
            continue
        }
        if obj.LastModified.After(cutoff) {
            skipped["age_recent"]++
            continue
        }
        
        bucket, objectPath, splitErr := splitBucketAndPath(obj.Key)
        if splitErr != nil { skipped["pattern_mismatch"]++; continue }
        if err := m.minio.RemoveObject(ctx, bucket, objectPath, minio.RemoveObjectOptions{}); err != nil {
            skipped["api_error"]++
            m.metrics.RecordOrphanSkipped("api_error")
            continue
        }
        reaped++
        m.metrics.RecordOrphanReaped()
    }
    
    for reason, n := range skipped {
        for i := 0; i < n; i++ {
            m.metrics.RecordOrphanSkipped(reason)
        }
    }
    
    m.logger.Info("orphan reaper completed",
        zap.Int("reaped", reaped),
        zap.Any("skipped", skipped),
    )
    return reaped, nil
}

const maxReapPerRun = 1000
```

### 9.4 parseBackupFilename helper

```go
var backupFilenameRe = regexp.MustCompile(`^.*backup-([0-9a-f]{8})-.+?\.xml(\.[a-z0-9]+)?(\.[a-z0-9]+)?$`)

func parseBackupFilename(key string) (taskID8 string, ok bool) {
    m := backupFilenameRe.FindStringSubmatch(key)
    if m == nil { return "", false }
    return m[1], true
}
```

(注意：T-0079 的 regex 在 acs/upload/handler.go 是 `^backup-([0-9a-f]{8})-...`，不带前缀；这里 key 是包含 path 的 full object key 如 `backup/2026/03/29/backup-aabbccdd-SN999.xml.gz`，所以 regex 需 anchor `.*` 前缀容许)

### 9.5 文件清单

修改：
- `omcgo/internal/backup/policy_monitor.go` — 加 RunOrphanReaperOnce + parseBackupFilename + maxReapPerRun const + Start +1 cron schedule
- `omcgo/internal/backup/policy_metrics.go` — +2 collectors + RecordOrphanReaped + RecordOrphanSkipped helpers
- `omcgo/internal/backup/repository.go` — TaskRepository interface +ListAllTaskIDPrefixes method
- `omcgo/internal/backup/pg_repository.go` — ListAllTaskIDPrefixes 实施 SQL
- `omcgo/internal/backup/policy_monitor_test.go` — +V1-V10 case
- `omcgo/cmd/app/provider/modules.go` — DI taskIDLister（taskRepo 已 inject 给 PolicyMonitor → 加 setter SetTaskIDLister + 用 backupTaskRepo 满足 narrow iface）

无新增文件（除 PRD/verify）。

### 9.6 Mock 更新

5 既有 monTaskRepo / mockTaskRepo / fakeTaskRepo / execTaskRepo 加 ListAllTaskIDPrefixes stub method（返 nil set OK）。

---

## 10. 实施要点

预计工作量：M（约 1.5 人日）— PolicyMonitor 扩展 + repository iface + 5 mock 同步 + 10 case 新测 + DI

预计涉及模块：`omcgo/internal/backup/`（5 文件）+ `cmd/app/provider/modules.go`

预计新增端点：无；预计新迁移：无

---

## 11. 审批

| 角色 | 占位 | 日期 | 备注 |
|------|------|------|------|
| 产品 / 架构 / Go / 数据 / 运维 / QA | Claude | 2026-04-29 | list-prefix 而非新表（scope 控制）；30 天 age 安全网防 race |

---

## 12. 变更记录

| 日期 | 版本 | 摘要 | 作者 |
|------|------|------|------|
| 2026-04-29 | v1.0 | 初稿；ULTRATHINK 8 决策；mirror T-0076/0082 既有 PolicyMonitor 模式 | Claude |
