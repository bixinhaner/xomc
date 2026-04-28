# T-0061 连接池配额监控 — Verify 报告

> **章程映射**：W3.F.3 — pgxpool / Redis / NATS 三类连接池暴露 Prometheus metrics 并接入 AlertManager 阈值告警
> **执行人**：sub-agent (worktree-agent-a7fee570)
> **执行日期**：2026-04-28

---

## 1. 改动文件清单

| 路径 | 类型 | 说明 |
|------|------|------|
| `omcgo/internal/core/components/postgres/pool_metrics.go` | 新增 | pgxpool 连接池指标采集（gauge × 3 + counter × 1），后台 5s 周期刷新；带可注入 sampler 便于单测 |
| `omcgo/internal/core/components/postgres/pool_metrics_test.go` | 新增 | 6 个表驱动用例，覆盖指标注册、单调递增、Stop 幂等、后台采样、空指针 panic |
| `omcgo/internal/core/components/redis/pool_metrics.go` | 新增 | go-redis pool 指标（in_use/idle/max gauge），通过 `PoolStats()` 抽象的 `poolStatsProvider` 适配；PoolSize 未知时回退到 TotalConns |
| `omcgo/internal/core/components/redis/pool_metrics_test.go` | 新增 | 6 个表驱动用例，含 stub 注入、PoolStatsProvider 路径、PoolSize 回退、nil 防护 |
| `omcgo/internal/core/components/nats/conn_metrics.go` | 新增 | NATS 连接指标：status gauge + reconnect_total / msgs_in_total / msgs_out_total counter；通过 `SetReconnectHandler` 自动累加重连，msgs 用 delta 写入避免回退 |
| `omcgo/internal/core/components/nats/conn_metrics_test.go` | 新增 | 8 个表驱动用例，含连接状态切换、msgs 单调性、reconnect 回调、Stop 幂等、后台采样、nil 防护 |
| `deployments/monitoring/alerts/connection-pool-alerts.yml` | 新增 | 5 条告警规则：PgxPoolHighUtilization (>80% 5m, warning) / PgxPoolNearExhaustion (>95% 2m, critical) / RedisPoolHighUtilization (>80% 5m, warning) / NatsDisconnected (==0 1m, critical) / NatsReconnectStorm (5m 内 >5 次, warning) |

**严禁清单合规**：
- 未改 `cmd/app/provider/*` 或 `cmd/app/main.go`
- 未改 `omcgo/internal/core/components/postgres/slow_query*`（T-0060 范围）
- 未改 `omcgo/migrations/`
- 未改 `deployments/k8s/*` / `scripts/db_backup.sh` / `docs/project/release-gate.md`
- 未改 `omcmb/` / `.github/`
- 未 commit / push / pull
- 未新增 `go.mod` 依赖（仅引用已存在的 `prometheus/client_golang`、`prometheus/client_model`、`go-redis/v9`、`nats.go`）

---

## 2. 章程 grep 验证

### grep 1 — 5 类指标名全命中

```
$ grep -rn "pgxpool_in_use\|pgxpool_max\|redis_pool_in_use\|nats_conn_status\|nats_reconnect_total" internal/
```

命中位置（仅列代码注册点，省略测试断言）：

| 指标 | 注册点 |
|------|--------|
| `pgxpool_in_use` | `internal/core/components/postgres/pool_metrics.go:99` |
| `pgxpool_max` | `internal/core/components/postgres/pool_metrics.go:107` |
| `redis_pool_in_use` | `internal/core/components/redis/pool_metrics.go:109` |
| `nats_conn_status` | `internal/core/components/nats/conn_metrics.go:103` (主路径) + `:163` (sampler 重载) |
| `nats_reconnect_total` | `internal/core/components/nats/conn_metrics.go:107` (主路径) + `:167` (sampler 重载) |

加上未列入 grep pattern 的 `pgxpool_idle`、`pgxpool_acquire_total`、`redis_pool_idle`、`redis_pool_max`、`nats_msgs_in_total`、`nats_msgs_out_total`，本任务总共暴露 **11 个 Prometheus 指标**（章程要求 ≥ 5）。

### grep 2 — 告警规则文件存在

```
$ ls deployments/monitoring/alerts/connection-pool-*.yml
deployments/monitoring/alerts/connection-pool-alerts.yml
```

### grep 3 — AlertManager 规则数

```
$ grep -c "^      - alert:" deployments/monitoring/alerts/connection-pool-alerts.yml
5
```

5 条告警规则（章程要求 ≥ 3）。覆盖度：

| 告警名 | 指标 | 阈值 | for | severity |
|--------|------|------|-----|----------|
| `PgxPoolHighUtilization` | `pgxpool_in_use / pgxpool_max` | > 0.8 | 5m | warning |
| `PgxPoolNearExhaustion` | 同上 | > 0.95 | 2m | critical |
| `RedisPoolHighUtilization` | `redis_pool_in_use / redis_pool_max` | > 0.8 | 5m | warning |
| `NatsDisconnected` | `nats_conn_status` | == 0 | 1m | critical |
| `NatsReconnectStorm` | `increase(nats_reconnect_total[5m])` | > 5 | 5m | warning |

---

## 3. 自跑验证

### 3.1 编译

```
$ go build ./...
（无输出，编译通过）
```

### 3.2 单元测试（race + count=1）

```
$ go test -race -count=1 \
    ./internal/core/components/postgres/... \
    ./internal/core/components/redis/... \
    ./internal/core/components/nats/...
ok      github.com/omcgo/omcgo/internal/core/components/postgres   1.400s
ok      github.com/omcgo/omcgo/internal/core/components/redis      1.703s
ok      github.com/omcgo/omcgo/internal/core/components/nats       2.276s
```

20 个测试用例全部通过，含 `-race` 数据竞争检测。

### 3.3 YAML 校验

```
$ python3 -c "import yaml; yaml.safe_load(open('deployments/monitoring/alerts/connection-pool-alerts.yml'))"
（无报错，结构合法）
```

---

## 4. 设计决策

### 4.1 为什么抽 sampler，而不是直接持有 *pgxpool.Pool

`pgxpool.Stat` 内部是私有字段，单元测试无法构造确定性快照。本实现把"采样"抽象成 `func() poolSnapshot`：
- 主路径 `RegisterPoolMetrics(*pgxpool.Pool, ...)` 从真实 pool.Stat() 闭包出 sampler
- 测试路径 `registerPoolMetricsWithSampler(...)` 直接喂入构造好的 snapshot

好处：测试不依赖真实 PG / Redis / NATS 服务器，CI 可在任何环境跑通；同时保留对外简洁 API。

### 4.2 counter 的 delta 写入

`pgxpool.Stat().AcquireCount()` 与 `nats.Conn.Stats().InMsgs/OutMsgs` 都是连接生命周期内累计，但 pool / conn 重建后会从 0 开始。Prometheus counter 必须单调递增，因此本实现：

```go
if current > p.lastTotal {
    p.AcquireTotal.Add(float64(current - p.lastTotal))
}
```

底层计数回退（重建）时仅重置 baseline，不更新 prom counter — 保证 rate() 查询不会出现负值。已在 `TestPoolMetrics_AcquireTotalIsMonotonic` / `TestConnMetrics_MsgCountersIncreaseMonotonically` 覆盖。

### 4.3 NATS 重连回调使用 SetReconnectHandler

NATS 客户端在 `nats.go` 已经注册了 `nats.ReconnectHandler` 用于打印日志，但本实现通过 `conn.SetReconnectHandler(...)` 覆盖以累加 `nats_reconnect_total`。代价：原始日志逻辑被替换。

**未来改进**：若需要保留日志 + 累加指标，应在 `components/nats/nats.go` 的 `NewNATSClient` 里把 metrics 注册下沉，使两者共用同一个回调链。本任务范围限制在不改 `nats.go` 主链路（属于 `core/components/nats` 模块的初始化代码），故采用 SetReconnectHandler 覆盖方案；同时提供 `RegisterConnMetricsWithSampler` 给特殊场景使用。

---

## 5. 后续工作（不属于本任务）

1. 在 `cmd/app/provider/`（T-0060/T-0061 严禁）或 `cmd/app/router/` 的初始化路径里调用 `RegisterPoolMetrics` / `RegisterConnMetrics`，把指标真正接入 metrics endpoint — 留给后续 wave 主链路集成。
2. 向 Grafana dashboard 加入 pool 利用率面板（参考 `deployments/monitoring/grafana-dashboard.json`）。
3. 在 `docs/project/release-gate.md` 增补"连接池利用率告警必须 wired-up"的发布门控（属于 T-0064 / 发布经理范畴）。

---

## 6. DoD

- [x] 5 类核心指标全部命中 grep
- [x] AlertManager 规则 ≥ 3 条
- [x] `go build ./...` 通过
- [x] `go test -race` 三个 package 全绿（20 个用例）
- [x] YAML 通过 `yaml.safe_load`
- [x] 严禁清单全合规
- [x] 不新增 go.mod 依赖
