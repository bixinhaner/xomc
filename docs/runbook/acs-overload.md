# Runbook: ACS Inform 风暴 + 限流过载演练

> **章程**: W3 收尾 / Backlog T-0026（Release Gate §3.4 5 类 Runbook 之一）
> **目的**: 验证 ACS 引擎在极端 Inform 流量（设备批量重启、网络故障恢复、整片基站断电恢复）下，per-device 限流器 + 全局准入控制器协同工作，避免雪崩 / OOM / 数据库连接池耗尽
> **配套**:
> - `internal/acs/server.go`（HTTP server + 全局准入）
> - `internal/acs/handler.go`（Inform 入口）
> - `internal/acs/ratelimit.go`（per-device + 全局限流，T-0041）
> - `internal/acs/admission.go`（AdmissionController 全局并发上限）
> - `internal/acs/session.go`（会话状态机）
> - `omcgo/scripts/cpe_simulator.py`（2200+ 行 CPE 模拟器）
> - `omcgo/scripts/loadtest`（压测二进制）
> **首次执行日**: 待 user 在 staging 环境执行后回填 §9 实测日志

---

## 1. 演练目标（容量 / 降级阈值）

| 维度 | 期望值 | 说明 |
|------|--------|------|
| 稳态 QPS（Inform） | ≥ 333 sessions/s（10 万设备 × 5 min 周期）| 章程基线 |
| 风暴峰值 QPS | 5× 稳态 = ~1,700 sessions/s | 模拟整片基站重启 |
| per-device 限流生效 | 单设备 > 10 次/min Inform 时 429 | T-0041 默认配置 |
| 全局准入生效 | 并发会话超阈值（默认 5,000）时 503 + Retry-After | T-0041 |
| ACS 进程 OOM | 不发生 | RSS 增长可控（< 4GB 单实例）|
| pgxpool 不耗尽 | 失败率 < 1% | acquire_failed_total 增长平缓 |
| Redis pool 不耗尽 | 失败率 < 1% | TR-069 会话写入不掉 |
| 风暴消退后恢复时间 | < 60 秒回到稳态延迟 P99 < 200ms | session 状态机不卡死 |
| 限流响应正确性 | 429 / 503 应带 Retry-After 头 | CPE 收到后退避重试 |

## 2. 前置条件

### 2.1 环境

- staging 环境部署 omcgo-acs 多实例（推荐 3 副本，分摊负载）
  - dev 端口 `:7557`，CWMP 标准 `:7547`
  - metrics `:9090`
- 后端依赖：
  - PostgreSQL（主库 + 1 备库）
  - Redis Cluster（3 主 3 从，session/cmdq/heartbeat 在此）
  - NATS JetStream（Inform 事件 publish 到 `device.inform.*`）
- `cmd/acs/etc/config.staging.yaml` 配置：
  ```yaml
  acs:
    listen: ":7547"
    max_concurrent_sessions: 5000     # 全局准入阈值
    session_ttl: 5m
    ratelimit:
      per_device_inform: 10           # 次/min
      per_device_burst: 5
      global_qps: 2000                # ACS 单实例硬上限
  postgres:
    max_conns: 50
  redis:
    pool_size: 50
  ```

### 2.2 监控

Prometheus 指标：
- `acs_inform_total{result=success|throttled|rejected}`
- `acs_session_active`（活跃会话）
- `acs_session_create_duration_seconds`（直方图）
- `acs_ratelimit_dropped_total{reason=per_device|global}`
- `acs_admission_rejected_total`（503 + Retry-After）
- `acs_session_state_total{state=...}`
- `pg_pool_acquire_failed_total{module=acs}`
- `redis_command_errors_total`
- `process_resident_memory_bytes{job=omcgo-acs}`

### 2.3 工具

- CPE 模拟器：`omcgo/scripts/cpe_simulator.py`（支持 `--burst-mode` 模拟集中重启）
- 压测：`omcgo/bin/loadtest --mode acs --concurrency N`
- `pidstat / vmstat`（监控 ACS 进程资源）

### 2.4 容量预估

| 设备数 | 5 分钟周期稳态 QPS | 5 倍峰值 QPS | 单 ACS 实例承载 |
|--------|--------------------|---------------|-----------------|
| 10,000 | 33 | 165 | 1 实例足够 |
| 50,000 | 167 | 835 | 1-2 实例 |
| 100,000 | 333 | 1,665 | 2-3 实例 |
| 500,000 | 1,667 | 8,333 | 8-10 实例 |

本演练以 100,000 设备规模为基线。

## 3. 故障注入步骤

### 3.1 基线采样（演练前 5 分钟）

```bash
# 启动 100 设备稳态流量（5 分钟周期）
ssh staging "python3 /opt/omcgo/scripts/cpe_simulator.py \
  --acs http://staging:7547 \
  --count 100 --inform-interval 300 --duration 1800 \
  --serial-prefix DRILL_OVL_BASE_" &
sleep 60

# 采集基线
curl -s http://staging:9090/metrics | grep -E "acs_inform_total|acs_session_active|process_resident_memory"
date -u +"%Y-%m-%dT%H:%M:%SZ baseline"
```

期望: `acs_session_active ≈ 100`、`acs_inform_total{result="success"}` 持续增长，无 throttled / rejected。

### 3.2 故障注入 — 三种风暴模式

**方式 A: 突发风暴（瞬时 1000 设备同时上线，模拟整片基站断电恢复）**
```bash
# 1000 个 CPE 同时发送 BOOTSTRAP Inform，inform_interval=300s
ssh staging "python3 /opt/omcgo/scripts/cpe_simulator.py \
  --acs http://staging:7547 \
  --count 1000 --inform-interval 300 --duration 1800 \
  --bootstrap-burst \
  --serial-prefix DRILL_OVL_BURST_" &
# 期望: 突发 QPS 峰值 ~500-800，5 秒后稳定
```

**方式 B: 持续高 QPS（模拟 5,000 设备 30s 心跳压测，远高于 5min 标准）**
```bash
# 5000 设备 × 30s inform = 167 QPS 持续 5 分钟
ssh staging "python3 /opt/omcgo/scripts/cpe_simulator.py \
  --acs http://staging:7547 \
  --count 5000 --inform-interval 30 --duration 300 \
  --serial-prefix DRILL_OVL_HIGH_" &
# 期望: 持续 167 QPS，per-device 限流不触发（30s 周期 < 10 次/min 阈值）
```

**方式 C: 单设备 Inform 洪泛（恶意 / 故障 CPE 高频上报）**
```bash
# 单个设备每秒 Inform 一次，持续 60 秒
for i in $(seq 1 60); do
  curl -s -X POST -H "Content-Type: text/xml; charset=utf-8" \
    --data-binary @/opt/omcgo/test/fixtures/inform-bootstrap.xml \
    http://staging:7547/ \
    -o /dev/null -w "%{http_code} "
  sleep 1
done | tee acs-overload-flood.log
# 期望: 第 11 次起返回 429 + Retry-After（per-device 限流 10 次/min）
```

**方式 D: loadtest 压测器（递进加压）**
```bash
# 200 → 500 → 1K → 2K → 5K 并发递进
ssh staging "/opt/omcgo/bin/loadtest \
  --mode acs \
  --target http://staging:7547 \
  --schedule 'ramp:200@1m,500@1m,1000@2m,2000@2m,5000@3m' \
  --duration 9m" &
```

### 3.3 故障期间持续观察（建议方式 A + 方式 C 组合，最具代表性）

执行方式 A 同时，每 10 秒抓一次指标：
```bash
while true; do
  echo "=== $(date -u +%H:%M:%S) ==="
  curl -s http://staging:9090/metrics | grep -E "^acs_(inform|session|ratelimit|admission)_" | sort
  sleep 10
done | tee acs-overload-metrics.log
```

## 4. 故障期间观察

### 4.1 ACS 进程指标

```bash
# 资源使用
ssh staging "pidstat -p \$(pgrep omcgo-acs) 5 12"  # 60 秒采样
# 期望: %CPU < 80%、RSS 增长 < 1GB

# Goroutine 数（pprof）
curl -s http://staging:9090/debug/pprof/goroutine?debug=1 | head -3
# 期望: < 5000 goroutines（每会话 1-2 个）
```

### 4.2 限流 / 准入触发情况

```bash
# 风暴期间限流计数
curl -s http://staging:9090/metrics | grep -E "acs_ratelimit_dropped_total|acs_admission_rejected_total"
# 期望（方式 A 突发场景）:
# - acs_ratelimit_dropped_total{reason="global"} 在峰值时增长，恢复后归零
# - acs_admission_rejected_total 仅在并发会话 > 5000 时增长
# 期望（方式 C 单设备洪泛）:
# - acs_ratelimit_dropped_total{reason="per_device"} 显著增长
```

### 4.3 数据库 / Redis 连接池

```bash
# pgxpool
curl -s http://staging:9090/metrics | grep -E "pg_pool_(acquire_failed|active|idle)_"
# Redis pool
curl -s http://staging:9090/metrics | grep -E "redis_pool_(active|idle)_conns|redis_command_errors_total"
# 期望: pg_pool_acquire_failed_total 增长缓慢；redis_command_errors_total 接近 0
```

### 4.4 会话状态机健康

```bash
# 各状态会话数分布
curl -s http://staging:9090/metrics | grep "acs_session_state_total"
# 期望: PROCESSING / RPC_PENDING 短暂上升后回落，IDLE → COMPLETE 流通畅
# 警示: RPC_PENDING 长期堆积 → CPE 不响应 → 会话超时清理
```

### 4.5 NATS publish 健康

```bash
# device.inform.* 主题积压
ssh staging "nats consumer info OMC_EVENTS device-inform-handler -s nats://staging:4222"
# 期望: num_pending 在风暴峰值短暂上升，恢复期 < 30s 归零
```

### 4.6 AlertManager 告警

```bash
curl -s http://staging:9093/api/v2/alerts | \
  jq '.[] | select(.labels.alertname | test("ACS|Inform|Throttle"; "i")) | {alertname, state: .status.state}'
# 期望（峰值期间 firing）:
# - ACSHighInformRate（QPS > 2× 稳态）
# - ACSAdmissionRejecting（503 比例 > 1%）
# 恢复后 1 分钟内 resolved
```

## 5. 故障恢复

### 5.1 自动恢复路径（期望）

1. **限流生效（< 1s）**:
   - per-device limiter（rate.Limiter）拒绝多余 Inform，返回 429 + Retry-After
   - global admission 在并发 > 5000 时返回 503，CPE 端按 Retry-After 退避
2. **session GC（5min TTL）**: 卡住的 RPC_PENDING 会话超时自动清理
3. **风暴消退（< 60s）**:
   - CPE 端 BOOTSTRAP 完成后切到 PERIODIC（5min 周期）
   - 单设备洪泛限流后，CPE 端按 SOAP 错误返回退避
4. **指标回归基线**:
   - `acs_inform_total` 增速回到 333/s
   - `acs_session_active` 回到 100K 稳态
   - RSS / goroutine 数 GC 回收

### 5.2 手动介入（如风暴持续 / 进程卡死）

```bash
# 1. 临时调低 max_concurrent_sessions（紧急情况）
# 修改 config.staging.yaml: max_concurrent_sessions: 2000
# Reload（如支持热重载）或滚动重启 ACS 实例
kubectl rollout restart deployment/omcgo-acs

# 2. 横向扩容 ACS 副本
kubectl scale deployment omcgo-acs --replicas=6

# 3. 强制 GC 卡死会话（紧急）
redis-cli -h redis-1.staging --scan --pattern "acs:session:*" | \
  xargs -L 100 redis-cli -h redis-1.staging EXPIRE 60   # 让所有会话 60s 后过期

# 4. 隔离恶意设备（单设备洪泛场景）
ssh staging "iptables -A INPUT -s <CPE_IP> -p tcp --dport 7547 -j DROP"
```

### 5.3 ACS 重启后状态恢复

```bash
# 验证 ACS 重启后能从 Redis 恢复未完成会话
ssh staging "systemctl restart omcgo-acs"
sleep 10
curl -s http://staging:9090/metrics | grep "acs_session_active"
# 期望: 重启后从 Redis 拉回活跃会话数（不是从 0 开始）
```

## 6. 验证

### 6.1 风暴期间无雪崩

```bash
# 6.1.1 错误率（500 系列）应 < 0.1%（限流触发的 429/503 不算 5xx 错误）
curl -s http://staging:9090/metrics | grep "acs_inform_total" | \
  awk '{ if ($0 ~ /result="error"/) e+=$2; else if ($0 ~ /result="success"/) s+=$2 } \
       END { printf "error_rate=%.4f%%\n", 100*e/(s+e) }'
# 期望: error_rate < 0.1%

# 6.1.2 P99 延迟在风暴期间 < 500ms（稳态 < 200ms）
curl -s http://staging:9090/metrics | grep "acs_session_create_duration_seconds_bucket" | \
  awk '/le="0.5"/ { p99=$2 } /le="\+Inf"/ { total=$2 } END { printf "P99<500ms ratio=%.4f\n", p99/total }'
# 期望: > 0.99
```

### 6.2 限流公平性

```bash
# 6.2.1 方式 C 单设备洪泛后，该设备前 10 次 200，第 11+ 次 429
grep -c "200" acs-overload-flood.log
grep -c "429" acs-overload-flood.log
# 期望: 200 次数 ~= 10，429 次数 ~= 50

# 6.2.2 429/503 响应带 Retry-After 头
curl -i -X POST http://staging:7547/ -d @inform.xml | grep -i "Retry-After"
# 期望: Retry-After: 30（或类似秒数）
```

### 6.3 资源占用可控

```bash
# 6.3.1 RSS 风暴前后对比
ssh staging "ps -o rss= -p \$(pgrep omcgo-acs) | awk '{ print \$1/1024 \"MB\" }'"
# 期望: < 4096MB（峰值），< 2048MB（恢复后）

# 6.3.2 goroutine 数恢复
curl -s http://staging:9090/debug/pprof/goroutine?debug=1 | head -1
# 期望: 风暴 30s 后回到 < 1000
```

### 6.4 数据完整性

```bash
# 6.4.1 风暴期间设备最终全部注册成功（方式 A 1000 设备）
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://staging:8081/api/v1/devices?serial_prefix=DRILL_OVL_BURST_" | jq '.total'
# 期望: 1000（允许 < 5 个因 admission 拒绝后未重试）

# 6.4.2 NATS 事件无丢失
ssh staging "nats consumer info OMC_EVENTS device-inform-handler -s nats://staging:4222 | grep -E 'num_ack_pending|num_redelivered|num_pending'"
# 期望: num_pending = 0，num_redelivered 接近 0
```

### 6.5 多实例负载均衡

```bash
# 三个 ACS 副本各自承担约 1/3 流量
for pod in $(kubectl get pod -l app=omcgo-acs -o name); do
  kubectl exec $pod -- curl -s localhost:9090/metrics | grep "^acs_inform_total{result=\"success\"}" | tail -1
done
# 期望: 三个副本 inform_total 差异 < 20%
```

## 7. 失败处理

### 7.1 ACS 进程 OOM 被杀

- 立即扩容 `kubectl scale deployment omcgo-acs --replicas=6`
- 调低 `max_concurrent_sessions`（默认 5000 → 紧急 3000）
- 检查内存泄漏：`go tool pprof http://staging:9090/debug/pprof/heap`
- 重点排查 session map / goroutine 泄漏

### 7.2 限流误伤合法设备

- 如果稳态流量被 throttled：调高 `per_device_inform` 阈值
- 如果 CPE 不响应 Retry-After：CPE 固件 bug，反馈厂商
- 临时白名单：`acs.ratelimit.whitelist: [serial1, serial2]`

### 7.3 pgxpool 耗尽（acquire_failed_total > 1%）

- 调大 `postgres.max_conns`（默认 50 → 100）
- 检查慢查询（T-0060 审计）：`SELECT * FROM slow_query_log WHERE module='acs' ORDER BY duration_ms DESC LIMIT 10`
- ACS 热路径减少 DB 写入频次（用 batch / 异步）

### 7.4 Redis pool 耗尽

- 调大 `redis.pool_size`（默认 50 → 100）
- 检查 long-running command（`redis-cli CLIENT LIST | grep -v idle=0`）
- session HSET 用 pipeline 减少 round-trip

### 7.5 NATS publish 阻塞

- 检查 NATS server 连接：`curl -s http://nats.staging:8222/connz`
- max_pending_per_subject 是否过小：调大 stream 配置
- 极端情况：降级为 ChannelBus（仅本进程消费）

### 7.6 CPE 端"风暴自激"（拒绝后立即重试，加剧风暴）

- 限流响应必须带正确的 Retry-After
- ACS 端为 BOOTSTRAP 事件做随机抖动延迟（jitter）
- 反馈 CPE 厂商：固件需实现指数退避

## 8. 回滚

### 8.1 演练失败回滚

```bash
# 8.1.1 停止所有 CPE 模拟器
ssh staging "pkill -f cpe_simulator.py"
ssh staging "pkill -f loadtest"

# 8.1.2 清理演练设备
psql -h pg-master.staging -U omcgo -c \
  "DELETE FROM devices WHERE serial_number LIKE 'DRILL_OVL_%';"
redis-cli -h redis-1.staging --scan --pattern "acs:session:DRILL_OVL_*" | \
  xargs -r redis-cli -h redis-1.staging DEL
redis-cli -h redis-1.staging --scan --pattern "acs:cmdq:DRILL_OVL_*" | \
  xargs -r redis-cli -h redis-1.staging DEL

# 8.1.3 重置 ACS 限流计数器（重启即可，limiter 状态在内存）
kubectl rollout restart deployment/omcgo-acs

# 8.1.4 验证回到稳态
sleep 60
curl -s http://staging:9090/metrics | grep -E "acs_session_active|acs_inform_total"
# 期望: session_active 回到非演练设备数
```

### 8.2 配置回滚

如演练中临时调低 `max_concurrent_sessions`：
```bash
# 还原 config.staging.yaml 默认值
git checkout cmd/acs/etc/config.staging.yaml
kubectl rollout restart deployment/omcgo-acs
```

### 8.3 黑名单清理

如演练中加了 iptables 隔离规则：
```bash
ssh staging "iptables -F INPUT"  # 谨慎，只在专用 staging 机
# 或精确删除：
ssh staging "iptables -D INPUT -s <CPE_IP> -p tcp --dport 7547 -j DROP"
```

## 9. 实测占位（待 staging 演练后回填）

```
首次演练: 2026-MM-DD by <user>
故障注入方式: A / B / C / D（突发 / 持续高 QPS / 单设备洪泛 / 递进加压）
风暴峰值 QPS: ___ /s
稳态 QPS: ___ /s
最高并发会话数: ___
acs_inform_total 风暴期间增量: ___
acs_ratelimit_dropped_total{reason="per_device"}: ___
acs_ratelimit_dropped_total{reason="global"}: ___
acs_admission_rejected_total: ___
ACS 进程峰值 RSS: ___ MB
峰值 goroutine 数: ___
P99 延迟（风暴期间 / 稳态）: ___ ms / ___ ms
错误率（5xx）: ___%
pg_pool_acquire_failed_total 增量: ___
redis_command_errors_total 增量: ___
NATS num_pending 峰值: ___
风暴消退后恢复时间: ___ 秒
告警触发: ACSHighInformRate / ACSAdmissionRejecting（firing 时长 ___ s）
判定: PASS / FAIL（详见附 verify-T-0026-acs-staging-drill.md）
回滚动作: 是 / 否（如是，原因: ___）
后续 action items（容量调整 / 配置优化）: ___
```

---

## 附：关联文档

- `internal/acs/server.go` — HTTP server + 全局准入
- `internal/acs/handler.go` — Inform 入口
- `internal/acs/ratelimit.go` — per-device + 全局限流（T-0041）
- `internal/acs/admission.go` — AdmissionController
- `internal/acs/session.go` — 会话状态机（IDLE → INFORM_RECEIVED → PROCESSING → RPC_PENDING → RPC_RESPONSE → COMPLETE）
- `omcgo/scripts/cpe_simulator.py` — CPE 模拟器
- `omcgo/scripts/loadtest` 二进制 — 压测工具（modes: acs/kpi/mr/all）
- `docs/runbook/redis-failover.md` — 关联 Redis 故障演练
- `docs/runbook/disaster-recovery.md` — DR 整体 Runbook（T-0024）
- `deployments/monitoring/alerts.yml` — ACS 告警规则

### 关键指标速查

```
acs_inform_total{result="success|throttled|rejected|error"}
acs_session_active                                — 当前活跃会话
acs_session_create_duration_seconds               — 直方图，P50/P95/P99
acs_ratelimit_dropped_total{reason="per_device|global"}
acs_admission_rejected_total                       — 503 + Retry-After
acs_session_state_total{state="IDLE|...|COMPLETE"}
process_resident_memory_bytes{job="omcgo-acs"}
```

---

*本 Runbook 由两半组成：framework（main 落地，本文件）+ staging 实测（外部）。
Runbook PASS 即视为 framework 部分 PASS；真实 QPS / 限流命中数 / 资源峰值由 user staging 演练填回 §9。
预计单次演练耗时 60-90 分钟（含基线采样 5min + 风暴注入 10min + 观察 30min + 验证 20min + 回滚 20min）。
强烈建议演练前阅读 `docs/消息队列全流程流转说明书.md` 和 `cpe_simulator.py` 注释，理解正常流量基线再注入异常。*
