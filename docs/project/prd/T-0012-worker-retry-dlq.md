# PRD: Worker 进程重试 / 死信队列（T-0012 / R-106）

> **关联**: Backlog T-0012 / Risk R-106 / Sprint-02..04 / Domain=infra
> **作者**: Claude（代 Owner=架构+运维）
> **创建**: 2026-04-28
> **状态**: 草案 → 实施（B 方案：主会话 PRD + sub-agent 实施）

---

## 1. 业务背景

worker 进程承担 OMC 的离线数据管线：PM/MR 文件采集与解析 / KPI 计算 / 告警同步处理 / 备份执行 / 报表生成 / 任务收敛等 12 类 EventBus 订阅。

当前 `cmd/worker/main.go:77-191` 注册 12 个 subscriber，每个失败处理 = `logger.Warn(...)` 然后**继续**。R-106 风险登记册描述：

> "EventBus 订阅失败无重试，长运行稳定性未验证；PM/MR 文件处理失败会直接丢弃"

风险表现：
- ❌ 短暂故障（DB 抖动 / MinIO 临时不可达）→ 事件直接丢失
- ❌ 文件处理 panic → goroutine 崩溃但事件已 Ack
- ❌ 无 DLQ 持久化，无法事后排查 / 重放
- ❌ 无运维 admin 入口看死信、清理、重放

T-0010（NATSEventBus）已提供事件层 maxDeliveries=5 + Ack/Nak/Term，但**业务处理失败的可观测性 + 持久化 + 重放**仍缺。本 PRD 补这一层。

---

## 2. 用户故事

| 角色 | 故事 |
|------|------|
| 运维（oncall） | 我希望 worker 处理失败的事件**不丢**，能看到死信队列了解失败原因，必要时手动重放 |
| 架构师 | 我希望 retry/DLQ 是**通用框架**，新 subscriber 接入仅几行代码，不重复造轮子 |
| 开发者 | 我希望失败重试**自动**且**可观测**（Prom 指标 + 日志），不污染业务代码 |
| QA / 运营商交付 | 我希望长跑稳定性可量化（DLQ 大小、retry 成功率），便于压测与上线评估 |

---

## 3. 验收标准（Given-When-Then）

### V1 — 短暂错误自动重试成功
- **Given** PM Collector 处理 `pm.file.received` 事件，第 1 次 DB 写失败（短暂错误），第 2 次成功
- **When** Runner.Wrap 包裹的 handler 被调用
- **Then** 事件最终成功处理，无 DLQ 入条，`worker_retry_attempts_total{module="pm",result="success"}` +1

### V2 — 多次失败后进 DLQ
- **Given** PM Collector 处理事件 3 次都失败（默认 MaxAttempts=3）
- **When** retry 耗尽
- **Then** 一条 dead_letters 记录写入（含 source_module / source_subject / payload / error / retry_count=3）
- **And** `worker_dlq_entries_total{module="pm",subject="pm.file.received"}` +1
- **And** `worker_retry_attempts_total{module="pm",result="failed"}` +1（最后一次失败）

### V3 — Admin 列表查询
- **Given** dead_letters 表有 5 条 PM 模块死信
- **When** `GET /api/v1/admin/dead-letters?module=pm&page=1&page_size=10`
- **Then** 返回 200 + 5 条记录（按 created_at DESC 排序）+ pagination 信息

### V4 — Admin 重放成功
- **Given** dead_letters 表有 1 条 PM 死信，PM Collector bug 已修
- **When** `POST /api/v1/admin/dead-letters/<id>/replay`
- **Then** 该 payload 重新 publish 到原 subject (`pm.file.received`)；记录 worker_dlq_replays_total{result="published"}+1
- **And** 死信记录**保留**（不删，便于审计；运维 DELETE 才删）

### V5 — Admin 删除
- **Given** 一条 dead_letter 已确认无需重放
- **When** `DELETE /api/v1/admin/dead-letters/<id>`
- **Then** 返回 200，记录从表中删除

### V6 — context 取消时不再 retry
- **Given** worker 接收 SIGTERM，ctx 取消
- **When** Runner.Wrap 在 retry 间隔中
- **Then** 立即返回 ctx.Err()，不再 retry，事件按 EventBus 协议处理（NATS Nak / channel 丢弃）

### V7 — RBAC 拦截非 ops 用户
- **Given** 普通用户（非 ops 角色）持有 token
- **When** 调用任何 `/api/v1/admin/dead-letters/*` 端点
- **Then** 返回 403 Forbidden（RBAC middleware 拦截）

---

## 4. 运营商差异矩阵

| 维度 | CMCC | CTCC | CUCC |
|------|------|------|------|
| Retry 策略 | 一致 | 一致 | 一致 |
| DLQ 容量 | 一致（按 module 分区可选）| 一致 | 一致 |
| 死信处理流程 | OEM 运维统一 | 同 | 同 |
| 实际差异 | **无** | **无** | **无** |

**结论**：T-0012 对运营商透明，不需要 Carrier 适配点。

---

## 5. 非目标

- ❌ 不实现自动重放（D4：手动 admin 决策）
- ❌ 不实现死信告警（已有 `worker_dlq_entries_total` metric → 由 Prometheus AlertManager 配置告警阈值，PRD 内不建告警规则）
- ❌ 不替换既有 `internal/alarm/dead_letter.go` (T-0011 webhook DLQ，语义不同：outbound webhook vs inbound event)
- ❌ 不引入 Circuit Breaker（已有 `internal/core/reliability/circuit_breaker.go`，本任务不强制叠加，subscribers 按需自行接入）
- ❌ **第一版仅 PM Collector 接入**，其他 11 个 subscriber 后续 PR 扩展（最小 viable）

---

## 6. 依赖

| 依赖 | 用途 |
|------|------|
| `internal/core/reliability/retry.go` | 既有 Retry() + RetryConfig 直接复用 |
| `internal/core/event/EventBus` | Replay 时 `Publish(subject, payload)` 重新发出 |
| `internal/admin/` 既有 RBAC middleware | Admin handler 套现有 `permGroup("ops")` |
| `pgxpool` | dead_letters 表 PG 持久化 |
| migrations/000044 | 新建表 |

无新外部依赖。

---

## 7. 度量（Prometheus 指标）

| 指标 | 类型 | 标签 | 说明 |
|------|------|------|------|
| `worker_retry_attempts_total` | counter | `module, subject, result` | result ∈ {success, failed}（最终结果，不是每次 retry）|
| `worker_dlq_entries_total` | counter | `module, subject` | 进 DLQ 计数（即 retry 耗尽事件数）|
| `worker_dlq_replays_total` | counter | `module, subject, result` | 手动重放结果，result ∈ {published, failed} |
| `worker_dlq_size` | gauge | `module` | 当前 DLQ 中条数（按 module 分组），按需懒拉取 |

---

## 8. 设计备忘（S2）

### 8.1 接口契约

#### dlq 包
```go
// internal/core/reliability/dlq/types.go
type DeadLetter struct {
    ID            uuid.UUID
    SourceModule  string    // "pm" / "mr" / "alarm" / etc
    SourceSubject string    // "pm.file.received" / etc
    Payload       []byte    // raw event payload (for replay)
    Error         string
    RetryCount    int
    CreatedAt     time.Time
    LastAttemptAt time.Time
}

type Filter struct {
    Module  *string
    Subject *string
    model.ListRequest
}

type Repository interface {
    Insert(ctx context.Context, entry *DeadLetter) error
    List(ctx context.Context, filter Filter) (*model.ListResponse[DeadLetter], error)
    Get(ctx context.Context, id uuid.UUID) (*DeadLetter, error)
    Delete(ctx context.Context, id uuid.UUID) error
    Count(ctx context.Context, sourceModule string) (int64, error)
}
```

#### runner 包
```go
// internal/core/reliability/runner/runner.go
type Runner struct {
    module    string
    retryCfg  reliability.RetryConfig
    dlq       dlq.Repository
    publisher Publisher  // small interface (Publish(subject, payload) error) — for replay
    metrics   *Metrics
    logger    *zap.Logger
}

// Publisher is the consumer-side interface for event publication during replay.
// EventBus implements this naturally.
type Publisher interface {
    Publish(ctx context.Context, subject string, payload []byte) error
}

func NewRunner(module string, retryCfg RetryConfig, dlq dlq.Repository, pub Publisher, metrics *Metrics, logger *zap.Logger) *Runner

// Wrap returns a handler with retry+DLQ instrumentation around fn.
// fn signature is the canonical EventBus handler shape.
func (r *Runner) Wrap(subject string, fn func(ctx context.Context, payload []byte) error) func(ctx context.Context, payload []byte) error
```

### 8.2 DB schema (migration 000044)

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS dead_letters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_module VARCHAR(64) NOT NULL,
    source_subject VARCHAR(128) NOT NULL,
    payload BYTEA NOT NULL,
    error TEXT NOT NULL,
    retry_count INT NOT NULL DEFAULT 0
        CHECK (retry_count >= 0 AND retry_count <= 100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dead_letters_module_created
    ON dead_letters(source_module, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_dead_letters_subject
    ON dead_letters(source_subject);

-- +goose Down
DROP INDEX IF EXISTS idx_dead_letters_subject;
DROP INDEX IF EXISTS idx_dead_letters_module_created;
DROP TABLE IF EXISTS dead_letters;
```

### 8.3 Admin REST API（RBAC: ops）

```
GET    /api/v1/admin/dead-letters
  ?module=<source_module>     可选过滤
  ?subject=<source_subject>   可选过滤
  &page=1&page_size=20
→ 200 + ListResponse<DeadLetter>

GET    /api/v1/admin/dead-letters/:id        → 200 + DeadLetter | 404
DELETE /api/v1/admin/dead-letters/:id        → 200 (deleted)
POST   /api/v1/admin/dead-letters/:id/replay → 200 (republished, record retained)
                                              → 404 / 500
```

`/replay` 行为：
1. 读 dead_letter 详情（404 if not found）
2. 调 `publisher.Publish(ctx, dl.SourceSubject, dl.Payload)`
3. 成功 → metric `worker_dlq_replays_total{result="published"}+1`
4. 失败 → metric `result="failed"+1`，返回 500 + err
5. **不**自动删除原死信（运维确认后手动 DELETE）

### 8.4 Worker 接入示范（PM Collector）

修改最小化：
```go
// internal/pm/collector/pm_collector.go 新增
func (c *PMCollector) SetRunner(r runner.Wrapper) { c.runner = r }
type Wrapper interface {
    Wrap(subject string, fn func(ctx, payload []byte) error) func(ctx, payload []byte) error
}

// Subscribe 内部
handler := c.handlePMFileReceived
if c.runner != nil {
    handler = c.runner.Wrap("pm.file.received", c.handlePMFileReceived)
}
return bus.Subscribe(ctx, "pm.file.received", handler)
```

`cmd/worker/main.go` 多 ~10 行：
```go
dlqRepo := dlq.NewPgRepository(w.PgPool)
runnerMetrics := runner.NewMetrics(w.MetricsReg)
pmRunner := runner.NewRunner("pm", reliability.DefaultRetryConfig(), dlqRepo, w.EventBus, runnerMetrics, logger)
pmCollector.SetRunner(pmRunner)
```

### 8.5 测试策略

| 文件 | 用例数 | 覆盖 |
|------|--------|------|
| `dlq/dlq_test.go` | ≥ 6 | Insert / List / Get / Delete / Count / not-found |
| `runner/runner_test.go` | ≥ 8 | Wrap 成功 / 短暂失败重试成功 / 全失败入 DLQ / ctx 取消立即停 / nil dlq 降级 / metrics 计数 / replay 路径 |
| `admin/dead_letter_handler_test.go` | ≥ 6 | List / Get / Delete / Replay 成功 / Replay 失败 / 404 / RBAC（403 一例）|
| `pg_repository_test.go` | (skipped) | 用 mock interface 完成上层覆盖；PG 集成测试由 e2e 段覆盖 |

总覆盖 ≥ 70%（与既有规范对齐）。

### 8.6 安全考虑

- Admin 端点全套 `permGroup("ops")` — 与既有 admin 路由模式对齐
- Replay payload 不暴露在错误响应中（避免敏感数据 leak）
- error 字段长度上限：TRUNCATE 到 4096 字符（防止 stack trace 撑爆 PG 行）

---

## 9. 验收（DoD）

- [ ] PRD 七要素全 + 运营商差异矩阵填写
- [ ] migration 000044 up/down 配对
- [ ] `go build ./...` 通过
- [ ] `go vet ./...` 0 warning
- [ ] `go test -race -count=1 ./internal/core/reliability/... ./internal/admin/... ./internal/pm/...` 全绿
- [ ] 覆盖率 dlq/runner ≥ 70%
- [ ] 新端点（GET /admin/dead-letters）在 e2e_verify.sh 加 ≥ 1 claim
- [ ] 4 个 metric `grep -rn` 都返回 ≥ 1
- [ ] `risk-register.md` R-106 状态 Open → Closed
- [ ] backlog T-0012 状态 planned → done

---

## 10. 关联

- `internal/core/reliability/retry.go` — 既有 Retry() 复用
- `internal/alarm/dead_letter.go` — T-0011 模板（语义不同，本任务并存）
- `internal/core/event/types.go` — EventBus.Publish 用于 replay
- `cmd/worker/main.go:77-191` — 12 subscriber 注册位置（本任务接入 PM 一处，其他 11 后续 PR）
- `docs/project/risk-register.md#R-106` — 关闭依据
- `docs/project/backlog.md` T-0012 — 状态回写

---

## 11. 后续 PR（不在本任务范围）

- 接入剩余 11 subscriber：MR / Alarm Receiver / Alarm Sync / Reboot Monitor / Reboot Closer / Transfer Bridge / Backup Executor / Report Generator / Expedited Receiver / Alarm Sync Processor / Heartbeat
- 死信告警规则（Prometheus AlertManager 配置 `worker_dlq_entries_total > 100/h` 触发告警）
- 死信压测（24h 高负载下 DLQ 增长率 < 0.1%）
- 集成 Circuit Breaker（reliability/circuit_breaker.go）形成 retry+CB+DLQ 三层防御

---

*本 PRD 由 dev-pipeline /pick T-0012 ULTRATHINK B 方案生成（主会话 PRD + sub-agent 实施 S3 + 主会话整合 + push）。*
