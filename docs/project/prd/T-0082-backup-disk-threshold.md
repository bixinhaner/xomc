# PRD: backup 磁盘阈值告警 — poll bucket size + alarm.raised（T-0082）

> **关联**: Backlog T-0082 / Sprint-08 / Domain=F06/backup+ops / Type=feat / Prio=P2
> **作者**: Claude（代 Owner=运维+Go）
> **创建**: 2026-04-29
> **状态**: 草案 → 实施
> **关键决策**: scope **聚焦于"轮询 + 边沿触发告警"**；schema 字段已就绪（MaxStorageGB + AlertThresholdPercent），无需 migration；severity 暂保持 `major`（T-0084 联合 disk-threshold + severity policy 设计已待）；alarm.cleared 配套实施防告警风暴

---

## 1. 业务背景

R-102 备份模块主链路（T-0071/0072/0073/0074/0075/0076/0077/0078/0079/0080）已闭环。**剩余唯一已知 hole**：磁盘容量保护。

T-0073 cleanup cron 解决"按时间/数量回收"，T-0076 解决"物理删除"，但都是**事后回收**——它们假定回收速度跟得上写入速度。如果运营商一次大批量备份（10K 设备同时一键 backup）+ 关闭 auto_cleanup（监管窗口期保留全量），磁盘可能在两次 cleanup 之间填满。届时：
- MinIO 新 PutObject 会失败 / 抛 server-side error
- T-0073 cron 只跑 `@daily` —— 不会即时回收
- **运维全无感知** —— alarm 引擎仅监控 backup_task 失败，不监控存储水位

T-0076 PRD §2 显式把"磁盘阈值告警"切给 T-0082。BackupPolicy 已有现成字段：
- `MaxStorageGB`（默认 500，存储容量上限，UI 已暴露）
- `AlertThresholdPercent`（50-95，默认 80，阈值百分比，UI 已暴露）

字段保存了 7 sprint 一直未被消费。本任务点燃它们。

**不做会发生什么**：磁盘填满 → 后续 backup 全部失败 → 告警风暴（每个失败的 task 单独 raise），而 root cause（磁盘满）反而无单独告警 → 运维误以为是个别设备问题，反复重试，磁盘越压越满。

---

## 2. ULTRATHINK 决策

### 2.1 容量基线 — % of 什么？

`AlertThresholdPercent` 字面是"百分比"，但**百分比相对于谁**有 3 选：

| 选项 | 公式 | 优劣 |
|------|------|------|
| A. 相对 BackupPolicy.MaxStorageGB | `usage% = used / (MaxStorageGB × 1024³)` | 字段已就绪；运维可控；与 UI policy 表单一致 |
| B. 相对 MinIO bucket quota（mc admin policy quota） | 需调 admin API + bucket 必须设过 quota | minio-go SDK 不直接暴露；许多部署不设 bucket quota |
| C. 相对 OS disk df | 与 BackupPolicy 解耦；但 MinIO 数据卷未必是独立 disk | 跨进程文件系统访问，部署复杂 |

**采纳 A**。`MaxStorageGB` 已是运维契约（policy 设了多大就多大），`AlertThresholdPercent × MaxStorageGB` 是契约上限的百分比。运维想监控真磁盘 `df` 由独立的 ops Runbook（T-0061 连接池监控）解决，与备份策略解耦。

公式：
```
threshold_bytes = MaxStorageGB × 1024³ × AlertThresholdPercent / 100
```

### 2.2 用 ListObjects 还是 admin API？

minio-go 直接暴露的 API：
- `client.ListObjects(ctx, bucket, opts)` — 流式列举；每个 ObjectInfo 含 Size
- `client.StatObject(...)` — 单对象大小

minio-go 不直接暴露：
- bucket-level used-bytes（admin API 在 `madmin-go` 子包，引入新依赖）

**采纳 ListObjects + Σ size**。利弊：
- ✅ 不引入新依赖（madmin-go 是另一个 module）
- ✅ 与 T-0083 orphan reaper 同源（reaper 也要 list-prefix）
- ⚠️ 大 bucket 慢 —— 100K 对象 ≈ 数百 MB API traffic + 数十秒；接受 hourly 轮询频率
- ⚠️ 不算 incomplete-upload；MinIO 默认会清，可忽略

未来优化路径（不在本任务）：可加 madmin-go 依赖切换到 `AccountInfo`（O(1) 查 bucket usage）—— 但这是配置/部署改动，单独任务。

### 2.3 轮询频率

- `@daily`（与 cleanup 同频）—— 太稀；磁盘 6h 内涨 30% 是可能的
- `@hourly` —— 默认值；list 100K 对象 ≈ 30s 内
- `@every 15m` —— 过频；list 大 bucket 时 list 自身占用网络
- 自定义 cron in policy.CleanupTime —— 字段语义已绑定 cleanup，复用容易混

**采纳 `@hourly` 硬编码**。运维有需要可改后端常量重启；configurable 字段拆 followup（如有需求）。

### 2.4 告警去抖（anti-flap）—— 边沿触发 + alarm.cleared

**纯阈值检查的问题**：bucket=85%，threshold=80% → 每次 poll 都重发 alarm.raised → 1 小时一条告警 → 24 条/天 → 运维收件箱炸了。

两种去抖方案：
- **A. 边沿触发**：进程内维持上次状态 `lastAboveThreshold bool`；above→below 不再发 raised；below→above 才发 raised；above→below 时**配套发 alarm.cleared**
- **B. Cooldown**：维持 `lastAlarmTime`，间隔 < 6h 不重发

**采纳 A**。理由：
- alarm.cleared 是 F04 既有 EventBus subject，下游 alarm 引擎能正确处理"告警关闭"
- 不需要持久化（进程重启重新评估即可；hourly tick 一小时内就重新进入正确状态）
- 与 PolicyMonitor 已有的 in-memory state 一致（无需 DB schema）

**进程重启的语义边界**：重启后第一次 tick 若发现已在 above 状态 → 重新发 raised；这是"重启幂等的代价"，下游 alarm 引擎应去重（identifier 一致 → upsert 而非 append）。可接受。

### 2.5 Severity — 与 T-0084 的耦合点

T-0076 PRD §2.4 punt 了 severity policy-driven 化（拆 T-0084），核心理由：
> AlertSeverity column + schema migration 与 T-0082 disk-threshold 联合设计

T-0082 现在处理 disk-threshold，正是联合设计窗口。但**本任务不增加 schema column**：
- AlertSeverity 进 schema 是个独立设计动作（影响 PolicyHandler / PolicyService.validatePolicy / PolicyServiceTest / FE form）
- 把它和"启用 disk 监控"耦合到一个 PR，scope 立即膨胀
- T-0082 的核心价值是"让 disk 监控这件事先跑起来"，severity 调优是质量微调

**采纳**：T-0082 disk-threshold alarm 暂用 `severity="major"`（与 backup_task_failed 一致，便于运维一刀切配 filter）。AlertSeverity column 留 T-0084 单独闭环；届时 T-0084 同时关闭 backup_task_failed 和 backup_storage_threshold_exceeded 两处 TODO。

更新两处 TODO 注释指向 T-0084（policy_alarm_publisher.go 已有，新增 publisher 同步）。

### 2.6 Alarm identifier 与 payload schema

镜像 `FailureAlarmPayload` 模式新建 `StorageThresholdAlarmPayload`：

```go
type StorageThresholdAlarmPayload struct {
    Source            string `json:"source"`             // "backup"
    Severity          string `json:"severity"`           // "major"
    Identifier        string `json:"identifier"`         // "backup_storage_threshold_exceeded"
    Summary           string `json:"summary"`
    BucketName        string `json:"bucket_name"`        // "config_backup"
    UsedBytes         int64  `json:"used_bytes"`
    CapacityBytes     int64  `json:"capacity_bytes"`     // MaxStorageGB × 1024³
    UsagePercent      int    `json:"usage_percent"`      // 0..100+
    ThresholdPercent  int    `json:"threshold_percent"`  // policy.AlertThresholdPercent
    AlertEmail        string `json:"alert_email,omitempty"`
}
```

**Identifier 冲突检测**：grep 既有 `backup_*` identifier — 仅有 `backup_task_failed`。新 identifier `backup_storage_threshold_exceeded` 无冲突。

**alarm.cleared 走同 identifier**：F04 既有"identifier 关闭" 即 raise+clear 配对消除告警。alarm.cleared payload schema 镜像 raise，但 severity / threshold 字段保留以便 alarm 引擎记账。

### 2.7 失败模式 — 必须 fail-open

list bucket 本身失败的可能：
- MinIO 不可达（网络瞬断）
- IAM 权限缺失
- bucket 不存在（部署期 MinIO 重建未恢复 bucket）

**采纳 fail-open**：list 失败 → record metric `check_total{result="failure"}` + log warn → 跳本轮 → 下轮重试。绝不因 list 失败发"假告警"，更不因 list 失败 panic 整个 PolicyMonitor。

### 2.8 测试边界 —— 不真连 MinIO

单测全用 mock BucketLister：

```go
type fakeBucketLister struct {
    objects []minio.ObjectInfo
    err     error
}

func (f *fakeBucketLister) ListObjects(ctx context.Context, bucket string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
    ch := make(chan minio.ObjectInfo, len(f.objects))
    if f.err != nil {
        ch <- minio.ObjectInfo{Err: f.err}
        close(ch)
        return ch
    }
    for _, o := range f.objects { ch <- o }
    close(ch)
    return ch
}
```

集成 / staging 真连留 T-0083 验证（reaper 也用 ListObjects）。

---

## 3. 用户故事

| 角色 | 故事 |
|------|------|
| 网管运维 | 我配 `MaxStorageGB=500 AlertThresholdPercent=80`，当 backup bucket 用到 400GB 时收到 alarm.raised 邮件（前提运维已配 backup-failure-email-route filter，T-0076 §9.4） |
| 网管运维 | 阈值跌回 80% 以下时，收到 alarm.cleared 自动关闭告警 —— 不需要人工 ack |
| 网管运维 | 阈值持续超过期间，**只发 1 次** alarm.raised；不会每小时一封邮件刷屏 |
| 后端开发 | 进程重启不会丢"上次告过警"语义到极端程度；最差是重启后第一小时再发一次 raised（alarm 引擎应去重） |
| Ops 监控 | Grafana panel 看到 `omc_backup_storage_used_bytes / omc_backup_storage_capacity_bytes` 时间序列；超阈值有视觉信号 |

---

## 4. 验收标准（GWT）

### V1 — 阈值未达，无告警
- **Given** policy `MaxStorageGB=10 AlertThresholdPercent=80`，bucket 含对象总 size = 5 GB（50%）
- **When** PolicyMonitor.RunStorageCheckOnce 跑一次
- **Then** 不发 alarm.raised；metric `backup_storage_check_total{result="success"}` +1；`backup_storage_used_bytes` gauge = 5×1024³；`backup_storage_threshold_alarm_total{kind="raised"}` 不变

### V2 — 阈值首次超过，发 alarm.raised
- **Given** policy `MaxStorageGB=10 AlertThresholdPercent=80`，monitor 内部状态 `lastAboveThreshold=false`，bucket 含对象总 size = 9 GB（90%）
- **When** RunStorageCheckOnce 跑一次
- **Then** 发 1 条 alarm.raised event（identifier=`backup_storage_threshold_exceeded`，severity=`major`，summary 含具体百分比）；`lastAboveThreshold` 变 true；`backup_storage_threshold_alarm_total{kind="raised"}` +1

### V3 — 持续超阈值，不重发
- **Given** V2 之后下一轮 tick，bucket 仍 9.5 GB（95%），`lastAboveThreshold=true`
- **When** RunStorageCheckOnce 跑一次
- **Then** **不**发新 alarm event；`backup_storage_threshold_alarm_total{kind="skipped"}` +1（标记"已知 above"）；gauge 更新

### V4 — 跌回阈值以下，发 alarm.cleared
- **Given** V3 之后某轮 tick，bucket 跌至 6 GB（60%），`lastAboveThreshold=true`
- **When** RunStorageCheckOnce 跑一次
- **Then** 发 1 条 alarm.cleared event（同 identifier）；`lastAboveThreshold` 变 false；`backup_storage_threshold_alarm_total{kind="cleared"}` +1

### V5 — list bucket 失败，fail-open
- **Given** mock BucketLister 返回 ObjectInfo{Err: ...}
- **When** RunStorageCheckOnce
- **Then** 不 panic；返回 err；metric `backup_storage_check_total{result="failure"}` +1；`lastAboveThreshold` 不变；下一轮可继续尝试

### V6 — policy.AlertOnFailure=false 跳过整个检查
- **Given** policy `AlertOnFailure=false`（运维显式关掉所有 backup 告警）
- **When** RunStorageCheckOnce
- **Then** 不 list bucket（节省 IO）；不发 alarm；metric `backup_storage_check_total{result="skipped"}` +1

### V7 — policy.MaxStorageGB=0 跳过（防误配）
- **Given** policy `MaxStorageGB=0`
- **When** RunStorageCheckOnce
- **Then** 不 list bucket（除 0 异常防御）；metric `backup_storage_check_total{result="skipped"}` +1；log warn "MaxStorageGB=0; storage threshold disabled"

### V8 — 进程重启后从冷启动重新评估
- **Given** 上次进程退出时 `lastAboveThreshold=true`，bucket 仍 above；新进程启动 `lastAboveThreshold=false`（默认值）
- **When** 第一次 RunStorageCheckOnce
- **Then** 视为 below→above 跃迁，再发一次 alarm.raised（接受重启重发；alarm 引擎以 identifier 去重）

### V9 — Cron 集成 — @hourly 自动跑
- **Given** PolicyMonitor.Start 已调
- **When** cron 触发到 @hourly 时点
- **Then** `RunStorageCheckOnce` 被调一次；并发安全（cron 不重叠跑同一 entry）

### V10 — Backward compat
- **Given** 既有 8 个 PolicyMonitor 测试 case（V1-V8 of T-0073/0076）
- **When** 测试套跑
- **Then** 所有既有断言通过；新加的 BucketLister 字段 nil-safe（不传 lister 等同关 disk monitor 但保留 cleanup 行为）

---

## 5. 运营商差异矩阵

无差异。三家运营商对 OMC 内部存储水位无感知；磁盘 cap 由 deployment SRE 决定，不区分 carrier。本任务**所有逻辑一致**。

---

## 6. 非目标

| # | 非目标 | 原因 / 后续承接 |
|---|--------|----------------|
| N1 | bucket quota 通过 madmin-go 查询（O(1) usage） | minio-go SDK 子包；引入需评估部署影响。ListObjects 当前足够；optimization 拆未来任务 |
| N2 | OS-level df 监控 | 与备份策略解耦；T-0061 连接池监控负责 OS 资源面 |
| N3 | AlertSeverity policy-driven（warning/major/critical） | 拆 T-0084；本任务用硬编码 `major`，与 backup_task_failed 一致 |
| N4 | 多 bucket 监控（pm/mr/firmware/...） | 当前 backup 唯一关心 `config_backup`；多 bucket 时再看 |
| N5 | 阈值跨进程 debounce（多 omcgo-app 实例时多次发 raised） | 当前部署单实例；多实例时由 alarm 引擎 identifier 去重承担 |
| N6 | 自定义 cron 频率（替代硬编码 @hourly） | 默认值合理；configurable 拆 followup |
| N7 | 历史 usage 趋势图 | Grafana time-series 已自然得到（gauge over time）；无需额外建表 |
| N8 | 强制清理（自动触发 cleanup 当 above critical） | 设计上 cleanup 是定时回收，强制触发会破坏运维可预测性；运维收到 alarm 后人工决策 |

---

## 7. 依赖

| 依赖 | 状态 |
|------|------|
| T-0076 cleanup Phase 2 | ✅ done — PolicyMonitor 已扩展为多职责载体；本任务挂第二个 cron |
| T-0071 BackupPolicy 持久化 | ✅ done — MaxStorageGB / AlertThresholdPercent 字段已就绪 |
| T-0007 alarm 邮件通道 | ✅ done — 运维配 `notify_email` filter 即可路由 |
| `splitBucketAndPath`（T-0079） | ✅ done — 不直接用，但 list 时 bucket name 来源同 |
| `*minio.Client` ListObjects | ✅ minio-go/v7 已 import |
| `event.SubjectAlarmRaised` | ✅ |
| `event.SubjectAlarmCleared` | ✅（pkg `internal/core/event/subjects.go` 已定义） |

无新外部依赖。

---

## 8. 度量

| metric | 类型 | 含义 |
|--------|------|------|
| `omc_backup_storage_used_bytes` | gauge | 当前 bucket 已用字节数（每次 tick 更新） |
| `omc_backup_storage_capacity_bytes` | gauge | 当前 capacity 上限（policy.MaxStorageGB × 1024³） |
| `omc_backup_storage_usage_ratio` | gauge | used/capacity（0..1+；可超 1 表示已突破容量） |
| `omc_backup_storage_check_total{result}` | counter | 检查 tick 结果；result ∈ {success, failure, skipped} |
| `omc_backup_storage_threshold_alarm_total{kind}` | counter | 告警 publish；kind ∈ {raised, cleared, skipped, error}（"error" = NewEvent JSON marshal 或 bus.Publish 失败 — 罕见但需要可视化） |

复用 `backup.PolicyMetrics` 命名空间，新增 5 collector + 5 Record 方法。

**反例监控**：
- `omc_backup_storage_check_total{result="failure"}` 持续 >0 → 部署/IAM 问题，需运维处理
- `omc_backup_storage_threshold_alarm_total{kind="raised"}` 单时间窗 >1 → 告警风暴漏防（去抖逻辑失效）—— 本任务防的就是这个

---

## 9. 设计备忘（S2）

### 9.1 接口签名

新增 `policy_storage_monitor.go`：

```go
// BucketLister is the narrow consumer-side contract PolicyMonitor needs
// to compute bucket usage. *minio.Client satisfies it.
type BucketLister interface {
    ListObjects(ctx context.Context, bucket string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo
}

// SetBucketLister wires storage monitoring (T-0082); nil disables it
// (preserves T-0073/T-0076 behaviour).
func (m *PolicyMonitor) SetBucketLister(l BucketLister) { m.bucketLister = l }

// RunStorageCheckOnce performs a single bucket size sum + threshold check.
// Returns (usedBytes, err). Exposed for direct test invocation.
func (m *PolicyMonitor) RunStorageCheckOnce(ctx context.Context) (int64, error)
```

新增 `policy_alarm_publisher.go` 顶部新函数：

```go
// PublishStorageThresholdAlarm publishes alarm.raised or alarm.cleared based
// on edge transition. Caller (PolicyMonitor) tracks lastAbove state.
func PublishStorageThresholdAlarm(
    ctx context.Context,
    bus event.EventBus,
    metrics *PolicyMetrics,
    transition StorageTransition,  // {Raised, Cleared}
    info StorageInfo,               // bucket/used/capacity/threshold/email
) error
```

### 9.2 PolicyMonitor 字段扩展

```go
type PolicyMonitor struct {
    // ... existing fields ...
    bucketLister       BucketLister  // optional T-0082 — nil disables disk monitor
    bus                event.EventBus  // ⚠ 新增 — disk alarm 需直接 publish (cleanup path 不需要)

    // T-0082 in-memory edge-trigger state. Process-restart resets to false;
    // first tick re-evaluates and may resend raised once. Acceptable cost
    // for not adding a DB column.
    storageMu          sync.Mutex
    lastAboveThreshold bool
}
```

`bus` 已在 PublishFailureAlarm 通过参数注入；本任务 PolicyMonitor 拥有它（构造器加参数 OR functional option）。**采纳构造器加参数**（一次性改 cmd/app/provider/modules.go DI）。

### 9.3 Cron schedule

`Start` 中追加：

```go
if _, err := m.cron.AddFunc("@hourly", func() {
    c, c2 := context.WithTimeout(scoped, 2*time.Minute)
    defer c2()
    if _, err := m.RunStorageCheckOnce(c); err != nil {
        m.logger.Warn("backup storage check tick failed", zap.Error(err))
    }
}); err != nil {
    cancel()
    m.cancel = nil
    m.cron = nil
    return fmt.Errorf("schedule storage check: %w", err)
}
```

2 分钟超时 cap：100K 对象 ≈ 30s；2 分钟有缓冲。

### 9.4 RunStorageCheckOnce 流程

```go
func (m *PolicyMonitor) RunStorageCheckOnce(ctx context.Context) (int64, error) {
    if m.bucketLister == nil {
        return 0, nil  // disabled
    }
    policy, err := m.policyService.Get(ctx)
    if err != nil { ... }
    if !policy.AlertOnFailure || policy.MaxStorageGB <= 0 {
        m.metrics.RecordStorageCheck("skipped")
        return 0, nil
    }
    
    // List bucket, sum sizes
    var used int64
    for obj := range m.bucketLister.ListObjects(ctx, CanonicalRestoreBucket, minio.ListObjectsOptions{Recursive: true}) {
        if obj.Err != nil {
            m.metrics.RecordStorageCheck("failure")
            return 0, fmt.Errorf("list bucket: %w", obj.Err)
        }
        used += obj.Size
    }
    capacity := int64(policy.MaxStorageGB) * 1024 * 1024 * 1024
    threshold := capacity * int64(policy.AlertThresholdPercent) / 100
    above := used > threshold
    
    m.metrics.RecordStorageCheck("success")
    m.metrics.SetStorageUsedBytes(used)
    m.metrics.SetStorageCapacityBytes(capacity)
    if capacity > 0 {
        m.metrics.SetStorageUsageRatio(float64(used) / float64(capacity))
    }
    
    // Edge-trigger
    m.storageMu.Lock()
    prev := m.lastAboveThreshold
    m.lastAboveThreshold = above
    m.storageMu.Unlock()
    
    info := StorageInfo{
        BucketName: CanonicalRestoreBucket,
        UsedBytes: used,
        CapacityBytes: capacity,
        UsagePercent: percentClamped(used, capacity),
        ThresholdPercent: policy.AlertThresholdPercent,
        AlertEmail: policy.AlertEmail,
    }
    
    switch {
    case above && !prev:
        return used, PublishStorageThresholdAlarm(ctx, m.bus, m.metrics, TransitionRaised, info)
    case !above && prev:
        return used, PublishStorageThresholdAlarm(ctx, m.bus, m.metrics, TransitionCleared, info)
    default:
        m.metrics.RecordStorageThresholdAlarm("skipped")  // already-known state
        return used, nil
    }
}
```

`percentClamped` 简单工具：clamp 到 [0, ∞)，>100 也允许（突破容量场景）。

### 9.5 Alarm payload 模式

```go
// StorageThresholdAlarmPayload mirrors FailureAlarmPayload schema for
// consistency with F04 alarm engine consumers (route by source/identifier).
type StorageThresholdAlarmPayload struct {
    Source           string `json:"source"`            // "backup"
    Severity         string `json:"severity"`          // "major" (T-0084 followup for policy-driven)
    Identifier       string `json:"identifier"`        // "backup_storage_threshold_exceeded"
    Summary          string `json:"summary"`
    BucketName       string `json:"bucket_name"`
    UsedBytes        int64  `json:"used_bytes"`
    CapacityBytes    int64  `json:"capacity_bytes"`
    UsagePercent     int    `json:"usage_percent"`
    ThresholdPercent int    `json:"threshold_percent"`
    AlertEmail       string `json:"alert_email,omitempty"`
}

type StorageTransition int
const (
    TransitionRaised StorageTransition = iota
    TransitionCleared
)
```

`alarm.cleared` 用同样 payload 但发布到 `event.SubjectAlarmCleared`。Summary 文案区分（"reached" vs "back to normal"）。

### 9.6 文件清单

新增：
- `omcgo/internal/backup/policy_storage_monitor.go` — RunStorageCheckOnce + BucketLister + edge-trigger state
- `omcgo/internal/backup/policy_storage_monitor_test.go` — 10 case 覆盖 V1-V10
- `omcgo/internal/backup/policy_storage_alarm.go` — PublishStorageThresholdAlarm + payload + transition enum
- `omcgo/internal/backup/policy_storage_alarm_test.go` — payload + raise/clear 路径

修改：
- `omcgo/internal/backup/policy_monitor.go` — PolicyMonitor 字段扩展（bus + bucketLister + storageMu + lastAboveThreshold）；Start 追加 @hourly schedule；NewPolicyMonitor 增加 bus 参数
- `omcgo/internal/backup/policy_metrics.go` — 加 5 collector + 5 Record 方法（gauges + counters）
- `omcgo/internal/backup/policy_alarm_publisher.go` — TODO 注释 comment 同步指向 T-0084（不改逻辑）
- `omcgo/internal/backup/policy_monitor_test.go` — 既有 8 case 适配 NewPolicyMonitor 签名变化
- `omcgo/cmd/app/provider/modules.go` — DI 装 EventBus + BucketLister 进 PolicyMonitor
- `deployments/monitoring/grafana/dashboards/omc-overview.json` — 加 backup-storage panel row（4 panels）

无新增 schema migration；既有迁移不动。

### 9.7 待定点

| 待定 | 决策 |
|------|------|
| ListObjects 在大 bucket（10M+ 对象）的 IO 成本 | 接受；@hourly 频率内可消化；超大 bucket 单独评估 madmin-go 切换 |
| Cron schedule（@hourly）configurable 化 | 否；硬编码默认即可；configurable 拆 followup（如有需求） |
| alarm.cleared payload severity 字段 | 保留 "major"（与 raised 一致便于对账）；alarm 引擎 cleared 处理时通常忽略 severity |

### 9.8 Dashboard 改动

`deployments/monitoring/grafana/dashboards/omc-overview.json` 加 backup-storage row（4 panel）：
1. Stat — `omc_backup_storage_usage_ratio` 当前值（threshold 0.8 → 红/绿）
2. Time-series — `omc_backup_storage_used_bytes` & `omc_backup_storage_capacity_bytes` overlay
3. Time-series — `rate(omc_backup_storage_check_total[1h])` by result
4. Stat — `sum(omc_backup_storage_threshold_alarm_total{kind="raised"})`

---

## 10. 实施要点（非规范性）

预计涉及模块：
- `omcgo/internal/backup/`（新增 2 文件 + 修改 4 文件）
- `omcgo/cmd/app/provider/modules.go`（DI）
- `deployments/monitoring/grafana/dashboards/omc-overview.json`（新增 panel row）

预计新增端点：无（纯后台 cron）

预计新增迁移：**无**（MaxStorageGB / AlertThresholdPercent 字段已存在）

预计工作量：M（约 1 人日）—— 8-10 测试 case + 5 metrics + 1 cron entry

---

## 11. 审批

| 角色 | 姓名/占位 | 日期 | 备注 |
|------|---------|------|------|
| 产品经理 | Claude（PM 代签）| 2026-04-29 | scope 锁定；severity policy-driven 拆 T-0084 |
| 架构师 | Claude（架构代签）| 2026-04-29 | edge-trigger 进程内 state 不持久化；接受重启重发 |
| 数据专家 | Claude（数据代签）| 2026-04-29 | ListObjects 路径；@hourly 频率；不引入 madmin-go |
| 运维专家 | Claude（运维代签）| 2026-04-29 | 4 panel 仪表盘配套；fail-open 不致命 |
| QA/发布经理 | Claude（QA 代签）| 2026-04-29 | 10 GWT 全可测；既有 8 PolicyMonitor case 兼容 |

---

## 12. 变更记录

| 日期 | 版本 | 变更摘要 | 作者 |
|------|------|---------|------|
| 2026-04-29 | v1.0 | 初稿；S0 起草，scope 锁定为"轮询 + 边沿触发 + alarm.cleared 配套"，无 schema 变更 | Claude |
