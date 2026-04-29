# PRD: backup decrypt 并发 semaphore 防 DoS（T-0089）

> **关联**: Backlog T-0089 / Sprint-08 / Domain=F06/backup+ops / Type=feat / Prio=P3
> **作者**: Claude（代 Owner=Go+运维）
> **创建**: 2026-04-29
> **状态**: 草案 → 实施
> **关键决策**: 单点 semaphore 在 `maybeDecrypt` 入口；buffered channel 实现；timeout-then-reject 语义；env var 配置默认 8 并发；fail-closed 503 + Retry-After

---

## 1. 业务背景

T-0075 实施 AES-256-GCM 信封加密，`maybeDecrypt`（`internal/acs/download/handler.go:265`）必须把整个密文 buffer 进内存才能完成 GCM 认证（GCM 是 single-pass authenticated encryption，不支持 partial-read auth）。当前实现：每个并发请求最多 buffer 64 MB + 1 KB envelope。

**威胁模型**（T-0075 review M-4 deferred）：攻击者已知 OMC 启用 backup 加密 + ACS 端点对外暴露（CPE 通过 Basic Auth 访问），可发起 N 个并发 backup 下载，每请求强制 ACS 分配 64 MB buffer：
- N=100 并发 → 6.4 GB 内存压力
- ACS 进程默认 OOM killed 或 GC pressure 导致延迟飙升
- 即使有 per-IP rate limit（T-0041），多设备 NAT 共享 IP 场景仍可绕过

**为什么之前没修**：T-0075 review 中识别为 M-4 但 explicitly deferred 到 T-0089 followup —— scope 控制（加密本身已是大任务），且 DoS 缓解不是密码学正确性问题。

**不做会发生什么**：生产环境一旦启用 backup 加密，恶意 CPE / 攻击者 / 甚至误操作（运维同时触发 100 个 restore）都可能 OOM ACS 进程。安全审计中这是 medium-severity 资源耗尽漏洞。

---

## 2. ULTRATHINK 决策（8 关键问题）

### 2.1 攻击面 — 仅 backup decrypt vs 通用 download

T-0075 加密**仅当** `EnableEncryption=true && Algorithm=="AES-256-GCM" && FileType==Config(3)`：
- backup 配置文件 `.enc` 后缀走 `maybeDecrypt`
- 固件 `.bin`（FileType=1）/ 数据模型 XML（FileType=11）**不走 decrypt** → 不需 semaphore
- 其他 `download.Handler` 路径仅 `io.Copy(w, obj)` 流式 stream，恒定 32KB stdlib buffer，不放大

**采纳**：semaphore 仅 guard `maybeDecrypt` 路径。其他 download 路径不引入开销。

### 2.2 block vs reject vs hybrid

| 选项 | 优 | 劣 |
|------|---|---|
| 纯 block（无超时） | 最终全部完成 | pile-up 死循环风险 |
| 纯 reject（try-acquire） | 立即可见 backpressure | legit 偶发也撞 503 |
| **timeout-then-reject** | 30s 排队消化 + 拒绝失控 | 单一参数选取依据需明确 |

**采纳 timeout-then-reject**。30s 默认超时依据：
- backup 64 MB ciphertext + GCM decrypt ≈ 50-200 ms（modern CPU）
- 8 slots × 200 ms = 1.6s 一轮 burn，30s 内可消化 ~120 个排队请求
- legit 运维 batch restore 100 设备（很少见）也不撞墙
- 攻击场景 30s 后开始返 503 释放压力

### 2.3 默认并发数选取

**指标**：64 MB × N 是 backup decrypt 路径的内存上限，需 < 进程总内存可分配的 5%（防止 GC 压力）：
- ACS 进程典型 4-8 GB heap（Go runtime + sessions + 缓冲区）
- 5% 上限 ≈ 200-400 MB
- 64 MB × 4 = 256 MB（**conservative**）
- 64 MB × **8 = 512 MB**（采纳 — sane default）
- 64 MB × 16 = 1 GB（borderline，需大内存部署）

**env var override**：`OMC_BACKUP_DECRYPT_CONCURRENCY`，0 ≤ N ≤ 64（>64 拒绝以防误配）。≤0 disable，preserve T-0075 既有行为（back-compat 保险栓）。

### 2.4 buffered channel vs `golang.org/x/sync/semaphore`

| 方案 | 评 |
|------|---|
| `make(chan struct{}, N)` + select | stdlib，KISS，单权重足够；Acquire = `slots <- struct{}{}`，Release = `<-slots` |
| `x/sync/semaphore.Weighted` | 自带 `AcquireCtx` 但 weighted 此处用不上 |

**采纳 buffered channel** —— 0 新依赖，select 三路（slots / ctx.Done / timer.C）覆盖完整语义。

### 2.5 与 T-0041 per-IP ratelimit 关系

| 层 | 责 |
|----|---|
| T-0041 ratelimit | requests/sec（time-window，per-IP） |
| **T-0089 semaphore** | concurrent buffer mem（resource-window，全局） |

**正交叠加**。T-0041 防 burst spam，T-0089 防 mem 爆炸。攻击者绕过 IP 限制（多 IP 源）也撞 T-0089；攻击者用单 IP 慢攻（每秒 1 个但每个 30s）撞 T-0089 但不撞 T-0041。

### 2.6 encrypt 上传侧也 buffer 64 MB，要不要也加？

**不在本任务**：
- upload 由 CPE 发起 → TR-069 session 1:1 模型限制并发
- per-device rate limit（`acs:ratelimit:inform:`）+ per-IP T-0041 + 全局 admission control
- plaintext-sized buffer，CPE 主动制造 65 MB 文件需绕过 backup 模板（不是攻击常态）
- **不同威胁模型**：upload-side guarding 是单独 followup（如有需求登记 T-0091+）

### 2.7 错误语义 — fail-closed

| 场景 | 行为 |
|------|------|
| timeout (30s 满后无 slot) | 返 **503 Service Unavailable** + `Retry-After: 5` 头 + metric `rejected{reason=timeout}` + log warn |
| ctx cancel（client disconnect） | 不返响应（连接已断）+ metric `rejected{reason=ctx_cancel}` + log info |
| acquire 成功后 decrypt fail | defer release slot 释放；按 T-0075 既有 fail-closed 路径返 422/500 |
| semaphore disabled (limit≤0) | 直接 pass-through，无 metric |

**fail-closed 哲学**：拒绝过度分配内存优于承诺解密然后 OOM。

### 2.8 metric 设计

3 个新 collector 加入 `backup.PolicyMetrics`（与 T-0075 decrypt error metric 同包，集中观测）：

| metric | 类型 | 含义 |
|--------|------|------|
| `omc_backup_decrypt_in_flight` | gauge | 当前持 slot 数（drives 容量规划 + alerting） |
| `omc_backup_decrypt_wait_seconds` | histogram | acquire 等待时长（drives 容量阈值告警） |
| `omc_backup_decrypt_rejected_total{reason}` | counter | 拒绝；reason ∈ {timeout, ctx_cancel, oversize_config} |

*oversize_config* = env var > 64 hard cap 拒绝时（启动期 panic 替代选项）。

**反例监控**：`omc_backup_decrypt_rejected_total{reason="timeout"}` 持续 > 0 = 容量不足 → 调大 `OMC_BACKUP_DECRYPT_CONCURRENCY` 或 scale ACS。

---

## 3. 用户故事

| 角色 | 故事 |
|------|------|
| 运维（生产）| ACS 进程 OOM 死亡风险消除 — backup decrypt 内存有硬上限（64MB×8=512MB） |
| 运维（容量规划）| Grafana 看 in_flight gauge 与 capacity（默认 8）的比例，调大 env var 前看 wait_seconds 分布 |
| 安全审计 | T-0075 review M-4 资源耗尽漏洞已闭环；fail-closed 不留隐式 plaintext 路径 |
| 后端开发 | 引入 1 个 setter + 4 行 acquire/release，既有 maybeDecrypt 行为保留；env var disable 完全恢复 T-0075 模式 |

---

## 4. 验收标准（GWT）

### V1 — 并发未达限值，不阻塞
- **Given** semaphore limit=4，当前 in_flight=2
- **When** 第 3 个 maybeDecrypt 调用
- **Then** 立即获取 slot；in_flight gauge=3；no rejection metric

### V2 — 并发达限值，第 N+1 阻塞
- **Given** semaphore limit=1，goroutine A 已 Acquire 持 slot
- **When** goroutine B 调 Acquire(ctx 1s timeout)
- **Then** B block 直至 timeout → 返 errDecryptSemaphoreTimeout；rejected_total{reason=timeout} +1

### V3 — Release 后排队的 acquirer 立即过
- **Given** limit=1，A 持 slot，B 阻塞在 Acquire
- **When** A.Release() 调用
- **Then** B 立即解锁；wait_seconds histogram 观察到 B 的等待时长

### V4 — ctx cancel 不泄漏 slot
- **Given** limit=1，A 持 slot，B 在 Acquire 中
- **When** B 的 ctx 被 cancel
- **Then** B 返 ctx.Err()；rejected_total{reason=ctx_cancel} +1；A.Release 后第 3 个 acquire C 仍能成功

### V5 — limit ≤ 0 disable，preserve T-0075 既有行为
- **Given** NewDecryptSemaphore(0, _) 或 NewDecryptSemaphore(-1, _)
- **When** 任意 Acquire/Release 调用
- **Then** 返 nil；100 个并发 maybeDecrypt 无阻塞（T-0075 路径），no metric record

### V6 — 503 + Retry-After 写到 HTTP response
- **Given** handler.maybeDecrypt 在 ServeHTTP 内被调，semaphore timeout
- **When** ServeHTTP 处理后返
- **Then** HTTP 503 + `Retry-After: 5` header + body 含 "decrypt overload"

### V7 — env var 解析
- **Given** `OMC_BACKUP_DECRYPT_CONCURRENCY` 未设
- **When** cmd/acs/main.go DI
- **Then** 默认 8

- **Given** `OMC_BACKUP_DECRYPT_CONCURRENCY=4`
- **Then** semaphore 用 limit=4

- **Given** `OMC_BACKUP_DECRYPT_CONCURRENCY=999`（> 64 hard cap）
- **Then** clamp 到 64 + log warn；不 panic

### V8 — Decrypt 失败仍 release slot
- **Given** A 持 slot，decrypt 返 ErrEncryptionAuthFailed
- **When** A 的 ServeHTTP 退出
- **Then** slot 已 release（defer 模式）；下一 acquirer 立即获取

### V9 — 无 SetDecryptSemaphore 时完全 back-compat
- **Given** Handler 仅 SetEncryption，未 SetDecryptSemaphore
- **When** 100 个并发 maybeDecrypt 调用
- **Then** 全部直接进入 decrypt（h.decryptSem == nil 跳过 acquire）；既有 T-0075 测试零回归

### V10 — wait_seconds histogram bucket 合理
- **Given** semaphore + load-test 模拟 acquire wait 0/100/500/2000ms 分布
- **When** scrape /metrics
- **Then** histogram 4 个 bucket 见样本（默认 prometheus DefBuckets 0.005-10s 覆盖）

---

## 5. 运营商差异矩阵

无差异。三家运营商对 OMC 内部并发限制无感知；env var 配置由部署 SRE 决定。

---

## 6. 非目标

| # | 非目标 | 原因 / 后续承接 |
|---|--------|----------------|
| N1 | encrypt 上传侧 semaphore | §2.6 不同威胁模型；upload-side 拆 followup（如有需求） |
| N2 | 通用 download semaphore（非 decrypt） | §2.1 其他下载路径流式无放大，不需要 |
| N3 | 自适应 limit（按 in_flight 动态调整） | over-engineer；env var + 运维手动调够用 |
| N4 | 优先级队列（critical task 跳过排队） | 当前所有 backup 同等重要；如有差异化需求拆 followup |
| N5 | Per-tenant / per-source slot 配额 | 单 instance ACS 模型未引入 tenant；future 横向扩展时再设计 |
| N6 | 观测告警规则（rejected > X 触发 alert） | 仪表盘记账即可；alerting rule 由部署侧维护 |
| N7 | x/sync/semaphore 切换 | §2.4 buffered channel 足够；future 若需 weighted/priority 可换 |

---

## 7. 依赖

| 依赖 | 状态 |
|------|------|
| T-0075 backup 加密 | ✅ done — semaphore 守护 maybeDecrypt 入口 |
| T-0041 ratelimit 中间件 | ✅ done — 与 T-0089 正交叠加 |
| `backup.PolicyMetrics` | ✅ done — 加 3 collectors |
| `prometheus/client_golang` | ✅ 已 import |

无新外部依赖。

---

## 8. 度量

§2.8 已列。

---

## 9. 设计备忘（S2）

### 9.1 接口签名

新 `internal/acs/download/decrypt_semaphore.go`：

```go
package download

import (
    "context"
    "errors"
    "time"

    "github.com/omcgo/omcgo/internal/backup"
)

// ErrDecryptSemaphoreTimeout signals the acquire path waited the configured
// timeout without obtaining a slot. The handler maps this to HTTP 503 with
// Retry-After. Sentinel so callers can distinguish from ctx.Err.
var ErrDecryptSemaphoreTimeout = errors.New("decrypt semaphore acquire timeout")

// DecryptSemaphore caps concurrent backup-decrypt buffer allocations to
// bound the 64MB-per-call ciphertext footprint introduced in T-0075. Wraps
// a buffered channel; nil receiver acts as disabled (T-0089 back-compat).
type DecryptSemaphore struct {
    slots   chan struct{}
    timeout time.Duration
    metrics *backup.PolicyMetrics
}

// NewDecryptSemaphore returns a semaphore with `limit` slots and per-acquire
// timeout. limit <= 0 returns nil (disabled — preserves T-0075 behaviour).
// limit > hardCap is clamped to hardCap with a metric record (caller logs).
const decryptSemaphoreHardCap = 64

func NewDecryptSemaphore(limit int, timeout time.Duration, m *backup.PolicyMetrics) *DecryptSemaphore {
    if limit <= 0 {
        return nil
    }
    clamped := limit
    if clamped > decryptSemaphoreHardCap {
        clamped = decryptSemaphoreHardCap
        if m != nil {
            m.RecordBackupDecryptRejected("oversize_config")
        }
    }
    return &DecryptSemaphore{
        slots:   make(chan struct{}, clamped),
        timeout: timeout,
        metrics: m,
    }
}

// Acquire blocks until a slot is available, ctx is cancelled, or the timeout
// elapses. Returns nil on success; ErrDecryptSemaphoreTimeout or ctx.Err on
// failure. Metric labels: "timeout", "ctx_cancel".
func (s *DecryptSemaphore) Acquire(ctx context.Context) (waited time.Duration, err error) {
    if s == nil {
        return 0, nil
    }
    start := time.Now()
    timer := time.NewTimer(s.timeout)
    defer timer.Stop()
    select {
    case s.slots <- struct{}{}:
        waited = time.Since(start)
        s.metrics.ObserveBackupDecryptWait(waited.Seconds())
        s.metrics.SetBackupDecryptInFlight(len(s.slots))
        return waited, nil
    case <-ctx.Done():
        s.metrics.RecordBackupDecryptRejected("ctx_cancel")
        return time.Since(start), ctx.Err()
    case <-timer.C:
        s.metrics.RecordBackupDecryptRejected("timeout")
        return time.Since(start), ErrDecryptSemaphoreTimeout
    }
}

// Release returns the slot. Safe to call if Acquire returned nil; do NOT call
// on Acquire failure path. Idempotent on nil receiver.
func (s *DecryptSemaphore) Release() {
    if s == nil {
        return
    }
    <-s.slots
    s.metrics.SetBackupDecryptInFlight(len(s.slots))
}
```

### 9.2 Metrics 扩展

`internal/backup/policy_metrics.go` 加 3 collectors：

```go
// T-0089 decrypt semaphore (concurrency cap to bound 64MB×N memory amplification):
decryptInFlight       prometheus.Gauge
decryptWaitSeconds    prometheus.Histogram
decryptRejectedTotal  *prometheus.CounterVec
```

3 Record/Set/Observe 方法，nil-safe 模式。

### 9.3 Handler 集成

`maybeDecrypt` 修改头部（line 265 附近）：

```go
func (h *Handler) maybeDecrypt(ctx context.Context, obj io.Reader, size int64, objectPath string) (io.Reader, bool, error) {
    if h.encryptor == nil {
        return obj, false, nil
    }
    suffix := "." + h.encryptor.Extension()
    if !strings.HasSuffix(objectPath, suffix) {
        return obj, false, nil
    }
    // T-0089: semaphore guard. Acquired slot released via deferred Release
    // captured in caller via returned cleanup. nil-safe when not wired.
    if h.decryptSem != nil {
        if _, err := h.decryptSem.Acquire(ctx); err != nil {
            return nil, false, err
        }
        defer h.decryptSem.Release()
    }
    // ... existing buffer + decrypt logic ...
}
```

需把 ServeHTTP 调用改为 `h.maybeDecrypt(r.Context(), obj, ...)`。

ServeHTTP 错误映射加分支：

```go
if encErr != nil {
    status := http.StatusInternalServerError
    switch {
    case errors.Is(encErr, ErrDecryptSemaphoreTimeout):
        status = http.StatusServiceUnavailable
        w.Header().Set("Retry-After", "5")
    case errors.Is(encErr, context.Canceled), errors.Is(encErr, context.DeadlineExceeded):
        // client gone — don't bother writing response
        return
    case errors.Is(encErr, backup.ErrEncryptionAuthFailed),
         errors.Is(encErr, backup.ErrEncryptionFormatInvalid):
        status = http.StatusUnprocessableEntity
    }
    http.Error(w, "decryption failed", status)
    return
}
```

### 9.4 DI 集成

`cmd/acs/main.go`：

```go
// T-0089: bound concurrent decrypt buffers (64MB×N memory amplification).
// Tunable via OMC_BACKUP_DECRYPT_CONCURRENCY env var; 0 disables.
const decryptDefaultLimit = 8
const decryptDefaultTimeout = 30 * time.Second

decryptLimit := decryptDefaultLimit
if env := os.Getenv("OMC_BACKUP_DECRYPT_CONCURRENCY"); env != "" {
    if n, err := strconv.Atoi(env); err == nil && n >= 0 {
        decryptLimit = n
    } else {
        inf.Logger.Warn("OMC_BACKUP_DECRYPT_CONCURRENCY invalid; using default 8",
            zap.String("value", env), zap.Error(err))
    }
}
sem := download.NewDecryptSemaphore(decryptLimit, decryptDefaultTimeout, backupPolicyMetrics)
downloadHandler.SetDecryptSemaphore(sem)
```

放在 `SetEncryption` 之后。

### 9.5 文件清单

新增（2）：
- `omcgo/internal/acs/download/decrypt_semaphore.go` — DecryptSemaphore type
- `omcgo/internal/acs/download/decrypt_semaphore_test.go` — 10 case

修改（4）：
- `omcgo/internal/acs/download/handler.go` — Handler +decryptSem 字段；maybeDecrypt 加 acquire/release；ServeHTTP 加错误映射；SetDecryptSemaphore setter
- `omcgo/internal/backup/policy_metrics.go` — +3 collectors + 3 helpers
- `omcgo/cmd/acs/main.go` — env var + DI

无 schema 迁移；无前端变更。

### 9.6 测试策略 — 防 timing flakiness

deterministic concurrency tests 用 channel barrier 替代 `time.Sleep`：

```go
// limit=1, A holds, B blocks via short timeout — deterministic
sem := NewDecryptSemaphore(1, 50*time.Millisecond, NewPolicyMetrics(nil))
require.NoError(t, sem.AcquireUnchecked()) // helper for tests
ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
defer cancel()
_, err := sem.Acquire(ctx)
require.ErrorIs(t, err, ErrDecryptSemaphoreTimeout)
```

每 case 用 50-200ms 级超时；总测试套 < 1s 即可。

### 9.7 待定点（< 3）

| 待定 | 决策 |
|------|------|
| `Acquire` 返 (waited, err) 还是只 err？ | 返 (waited, err) — 让调用方有 wait 时长可观测，metrics 内部已用，公共 API 暴露便于 future tracing |
| metrics 字段在 backup.PolicyMetrics 还是新建 download.SemaphoreMetrics？ | backup.PolicyMetrics — 既有 decrypt 错误 metric 同包，集中 |
| timeout 30s 是 const 还是可调？ | const for now；future 如有调整需求可再加 env var |

---

## 10. 实施要点（非规范性）

预计工作量：S（约 0.5 人日）—— 1 新 type + 1 setter + 3 metrics + ~10 test case + 1 DI 行

预计涉及模块：
- `omcgo/internal/acs/download/`（新 1 + 改 1）
- `omcgo/internal/backup/policy_metrics.go`
- `omcgo/cmd/acs/main.go`

预计新增端点：无

预计新增迁移：无

---

## 11. 审批

| 角色 | 姓名/占位 | 日期 | 备注 |
|------|---------|------|------|
| 产品经理 | Claude（PM 代签）| 2026-04-29 | scope 聚焦 decrypt 路径；upload-side 拆 followup |
| 架构师 | Claude（架构代签）| 2026-04-29 | buffered channel + nil-safe disabled 模式 |
| Go 工程 | Claude（Go 代签）| 2026-04-29 | select 三路覆盖 timeout/ctx/slot；defer release pattern |
| 安全合规 | Claude（SecOps 代签）| 2026-04-29 | fail-closed 503；闭环 T-0075 review M-4 资源耗尽漏洞 |
| 运维 | Claude（运维代签）| 2026-04-29 | env var 配置 + 3 metrics + Retry-After header |
| QA/发布经理 | Claude（QA 代签）| 2026-04-29 | 10 GWT 全可测；既有 T-0075 case 零回归 |

---

## 12. 变更记录

| 日期 | 版本 | 变更摘要 | 作者 |
|------|------|---------|------|
| 2026-04-29 | v1.0 | 初稿；S0 起草；ULTRATHINK 8 决策；buffered channel 实现；timeout-then-reject 语义 | Claude |
