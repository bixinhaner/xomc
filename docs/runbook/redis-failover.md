# Runbook: Redis 集群故障 + TR-069 会话恢复演练

> **章程**: W3 收尾 / Backlog T-0026（Release Gate §3.4 5 类 Runbook 之一）
> **目的**: 验证 Redis 主节点 / 集群分区故障时，OMC 三进程（app / acs / worker）go-redis 客户端自动重连，TR-069 会话状态、命令队列、L2 缓存正确恢复，无设备掉线 / 无命令丢失
> **配套**:
> - `internal/core/components/redis.go`（go-redis/v9 客户端初始化与监控 — T-0061）
> - `internal/acs/session.go`（acs:session:{device_serial} 会话状态）
> - `internal/acs/cmdqueue/`（acs:cmdq:{device_serial} 命令队列 Sorted Set）
> - `internal/config/datamodel/cache.go`（datamodel L2 Redis 缓存）
> - `docs/消息队列全流程流转说明书.md`（命令队列流转细节）
> **首次执行日**: 待 user 在 staging 环境执行后回填 §9 实测日志

---

## 1. 演练目标（RTO / 业务影响）

| 维度 | 期望值 | 说明 |
|------|--------|------|
| Redis 重连时间（RTO） | < 15 秒 | go-redis MaxRetries=3 + Sentinel/Cluster 自动 failover |
| TR-069 会话恢复率 | 100% | session 是 Hash + TTL 5min，failover 期间 CPE 重发 Inform 即恢复 |
| 命令队列丢失率 | 0% | 队列在 Redis 持久化（RDB + AOF）；failover 时未 ack 的命令应重投 |
| L2 缓存命中率回升 | < 60 秒回到 90%+ | datamodel 缓存冷启动期间走 PostgreSQL（L3）|
| Connection Request 去重失效率 | < 1% | acs:connreq:pending:{device} 短 TTL（30s），故障期间允许少量重复 |
| 限流器（ratelimit）失效影响 | 故障期间允许放过 | 故障恢复后立即重建 |

## 2. 前置条件

### 2.1 环境

- staging 环境部署 Redis 7（推荐 Cluster 模式 3 主 3 从，或 Sentinel 1 主 2 从 + 3 哨兵）
  - Cluster 节点：`redis-1.staging:6379` / `redis-2.staging:6379` / `redis-3.staging:6379`
  - 持久化：RDB（每 5 分钟）+ AOF（appendfsync everysec）
- staging 跑 omcgo-app（`:8081`）+ omcgo-acs（`:7547`）+ omcgo-worker（`:9092`）
- `cmd/{app,acs,worker}/etc/config.staging.yaml` 配置：
  ```yaml
  redis:
    mode: cluster   # 或 sentinel
    addrs:
      - redis-1.staging:6379
      - redis-2.staging:6379
      - redis-3.staging:6379
    max_retries: 3
    min_idle_conns: 10
    pool_size: 50
    read_timeout: 3s
    write_timeout: 3s
  ```

### 2.2 监控

- Prometheus + Grafana 关键指标：
  - `redis_pool_hits_total / redis_pool_misses_total`
  - `redis_pool_active_conns / redis_pool_idle_conns`
  - `redis_command_duration_seconds`（直方图）
  - `redis_command_errors_total`
  - `acs_session_active`（活跃 TR-069 会话数）
  - `acs_cmdq_depth`（命令队列深度）
  - `datamodel_cache_hit_rate`
- AlertManager 5 条 redis 告警规则已加载（T-0061）

### 2.3 工具

- `redis-cli` + `redis-cli --cluster`
- `iptables` / `tc qdisc`（网络隔离 / 延迟模拟）
- CPE 模拟器：`omcgo/scripts/cpe_simulator.py`（启 100 个虚拟 CPE）

### 2.4 业务流量基线

启动 100 个虚拟 CPE 持续 Inform：
```bash
ssh staging "python3 /opt/omcgo/scripts/cpe_simulator.py \
  --acs http://staging:7547 \
  --count 100 --inform-interval 30 --duration 600 \
  --serial-prefix DRILL_REDIS_" &
```

### 2.5 数据快照

```bash
# 切换前快照
redis-cli -h redis-1.staging --cluster info > /tmp/redis-pre-drill.txt
redis-cli -h redis-1.staging DBSIZE
redis-cli -h redis-1.staging --scan --pattern "acs:session:*" | wc -l
redis-cli -h redis-1.staging --scan --pattern "acs:cmdq:*" | wc -l
redis-cli -h redis-1.staging --scan --pattern "datamodel:*" | wc -l
```

## 3. 故障注入步骤

### 3.1 基线采样（演练前 5 分钟）

```bash
# 写一个标记 key 校验时刻
redis-cli -h redis-1.staging SET drill:baseline:redis "$(date -u +%s)" EX 3600

# 采集基线指标
curl -s http://staging:9091/metrics | grep -E "redis_pool|acs_session|acs_cmdq|datamodel_cache_hit"
date -u +"%Y-%m-%dT%H:%M:%SZ baseline"
```

期望: `acs_session_active >= 100`、`datamodel_cache_hit_rate > 0.9`。

### 3.2 故障注入 — 三种方式（任选其一）

**方式 A: kill 主节点（模拟单节点 crash + cluster failover）**
```bash
# 找出 cluster 中的一个 master
ssh redis-1.staging "redis-cli CLUSTER NODES | grep master"
# 强杀
ssh redis-1.staging "pkill -9 redis-server"
# Cluster 应在 cluster_node_timeout（默认 15s）后将 slave 提升为 master
sleep 20
ssh redis-2.staging "redis-cli CLUSTER NODES" | grep master
```

**方式 B: 网络分区（模拟 split-brain）**
```bash
# 隔离 master 节点的 6379 + 16379（cluster bus）端口
ssh redis-1.staging "sudo iptables -A INPUT -p tcp --dport 6379 -j DROP && \
                     sudo iptables -A INPUT -p tcp --dport 16379 -j DROP"
sleep 30   # 维持 30s 分区
# 还原
ssh redis-1.staging "sudo iptables -D INPUT -p tcp --dport 6379 -j DROP && \
                     sudo iptables -D INPUT -p tcp --dport 16379 -j DROP"
```

**方式 C: Sentinel 主动切换（推荐，最干净）**
```bash
ssh staging "redis-cli -h sentinel-1.staging -p 26379 SENTINEL FAILOVER mymaster"
# Sentinel 协议执行 failover，老 master 降级为 replica
sleep 10
ssh staging "redis-cli -h sentinel-1.staging -p 26379 SENTINEL get-master-addr-by-name mymaster"
```

**方式 D: docker compose 重启（local staging 简化场景）**
```bash
docker compose -f deployments/docker/docker-compose.yml stop redis
sleep 30
docker compose -f deployments/docker/docker-compose.yml start redis
```

### 3.3 故障期间持续业务流量

CPE 模拟器在 §2.4 已启动，持续 Inform 60 秒覆盖整个故障窗口。

同时主动触发命令队列写入：
```bash
# 每 2 秒派发 1 个 GetParameterValues 任务到设备
for i in $(seq 1 30); do
  curl -s -X POST -H "Authorization: Bearer $TOKEN" \
    -d "{\"device_serial\":\"DRILL_REDIS_$((i % 100))\",\"params\":[\"InternetGatewayDevice.DeviceInfo.X_VENDOR_Test\"]}" \
    http://staging:8081/api/v1/devices/get-parameter-values \
    -o /dev/null -w "%{http_code} "
  sleep 2
done | tee redis-failover-tasks.log
```

期望:
- 故障窗口前 0-5 秒：202 Accepted
- 故障窗口 5-15 秒：503 / 5xx（go-redis 重试中）
- 故障窗口 15-60 秒：202 恢复

## 4. 故障期间观察

### 4.1 应用层日志

```bash
tail -f /var/log/omcgo-acs.log | grep -iE "redis|cmdq|session"
# 期望关键日志:
# - "redis: connection lost: read tcp ... connection reset by peer"
# - "redis: retry attempt 1/3"
# - "redis: connection re-established"
# - "session restored from redis: device=..."
```

### 4.2 Prometheus 指标

```bash
while true; do
  echo "=== $(date -u +%H:%M:%S) ==="
  curl -s http://staging:9091/metrics | grep -E "redis_command_errors_total|redis_pool_active|acs_session_active|acs_cmdq_depth"
  sleep 5
done
```

期望趋势:
- `redis_command_errors_total` 故障期间快速增长，恢复后增速归零
- `redis_pool_active_conns` 故障期间 → 0，恢复后回升
- `acs_session_active` 短暂下跌 5-10%（CPE 重 Inform 后回填），1 分钟内回到基线
- `acs_cmdq_depth` 故障期间升高（命令积压），恢复后由 worker 消费回落

### 4.3 Cluster / Sentinel 状态

```bash
# Cluster 模式
ssh redis-2.staging "redis-cli CLUSTER NODES"
ssh redis-2.staging "redis-cli CLUSTER INFO"  # 期望 cluster_state:ok

# Sentinel 模式
redis-cli -h sentinel-1.staging -p 26379 SENTINEL master mymaster | grep -E "name|ip|num-slaves|num-other-sentinels"
```

### 4.4 AlertManager 告警

```bash
curl -s http://staging:9093/api/v2/alerts | \
  jq '.[] | select(.labels.alertname | contains("Redis")) | {alertname: .labels.alertname, state: .status.state}'
# 期望: RedisCommandErrorRateHigh / RedisPoolDepleted firing → 恢复后 resolved
```

### 4.5 数据持久化检查

```bash
# RDB 最近落盘时间
ssh redis-1.staging "redis-cli INFO persistence | grep -E 'rdb_last_save|aof_last_write'"
# 期望: rdb_last_save_time 在故障前 < 5min；故障期间 AOF 持续写
```

## 5. 故障恢复

### 5.1 自动恢复路径（期望）

1. **Cluster failover（5-15s）**: 备节点检测主节点失联 → cluster_node_timeout 后投票 promote
2. **客户端重连（15-25s）**:
   - go-redis 检测到 MOVED redirect → 更新 slot 映射 → 重连新 master
   - 老连接关闭，连接池重建
3. **业务恢复（25-30s）**:
   - TR-069 会话：CPE 在下个 Inform 周期重连，session 重建
   - 命令队列：worker 重新消费 acs:cmdq:{device}
   - L2 缓存：失效后从 PostgreSQL 重填，命中率渐回 90%+

### 5.2 手动介入

```bash
# 1. 验证 cluster 健康
ssh staging "redis-cli --cluster check redis-2.staging:6379"

# 2. 强制刷新客户端 slot 映射（如客户端缓存过期 slot 信息）
ssh staging "systemctl restart omcgo-app omcgo-acs omcgo-worker"

# 3. 验证关键 key 仍存在
redis-cli -h redis-2.staging --scan --pattern "acs:session:*" | wc -l
redis-cli -h redis-2.staging --scan --pattern "acs:cmdq:*" | wc -l
redis-cli -h redis-2.staging GET drill:baseline:redis  # 应等于 §3.1 写入的时间戳
```

### 5.3 老节点重新加入集群

```bash
# 启动老 master（已自动降级为 replica）
ssh redis-1.staging "systemctl start redis-server"
sleep 5
# 验证它作为 replica 加入
ssh redis-1.staging "redis-cli CLUSTER NODES" | grep myself
```

## 6. 验证

### 6.1 会话状态完整性

```bash
# 6.1.1 活跃会话数恢复
curl -s -H "Authorization: Bearer $TOKEN" \
  http://staging:8081/api/v1/acs/sessions/active | jq 'length'
# 期望: >= 95（允许 5% CPE 在故障窗口尚未重 Inform）

# 6.1.2 抽样会话内容完整
DEVICE=DRILL_REDIS_1
redis-cli -h redis-2.staging HGETALL "acs:session:$DEVICE"
# 期望: 含 cwmp_id / state / last_inform_at 等字段
```

### 6.2 命令队列无丢失

```bash
# 6.2.1 故障前发出 30 个 GetParameterValues 任务，应全部最终成功
psql -h pg-master.staging -U omcgo -c \
  "SELECT status, COUNT(*) FROM tasks
   WHERE created_at >= NOW() - INTERVAL '10 minutes'
     AND task_type = 'get_parameter_values'
   GROUP BY status;"
# 期望: completed 接近 30（允许 < 5 因故障期间 503 未派发）

# 6.2.2 acs:cmdq:* 残留检查（故障后无僵尸命令）
for k in $(redis-cli -h redis-2.staging --scan --pattern "acs:cmdq:DRILL_REDIS_*"); do
  redis-cli -h redis-2.staging ZCARD "$k"
done | sort -nr | head -5
# 期望: 队列深度小（worker 已消费完）
```

### 6.3 L2 缓存恢复

```bash
# 6.3.1 datamodel 缓存命中率
curl -s http://staging:9091/metrics | grep "datamodel_cache_hit_rate"
# 故障后 5 分钟内: < 0.7（冷启动）
# 故障后 30 分钟: > 0.9（恢复基线）

# 6.3.2 三级回退仍正常
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://staging:8081/api/v1/datamodel/resolve?carrier=cmcc&tech=lte&oui=001122&product_class=test-product" | jq '.scope'
# 期望: 返回 product / oui / carrier_default 之一，非 error
```

### 6.4 Cluster 一致性

```bash
ssh redis-2.staging "redis-cli --cluster check redis-2.staging:6379"
# 期望: [OK] All 16384 slots covered.

ssh redis-2.staging "redis-cli CLUSTER INFO | grep cluster_state"
# 期望: cluster_state:ok
```

### 6.5 业务 smoke

```bash
# CPE 模拟器全部仍在 Inform
grep -c "Inform Response 200" /var/log/cpe_simulator.log
# 期望: 持续增长，无大段空白

# 设备列表带状态正常返回
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://staging:8081/api/v1/devices?serial_prefix=DRILL_REDIS_" | jq '.total'
# 期望: 100
```

## 7. 失败处理

### 7.1 Cluster 不收敛 / split-brain

- 用 `redis-cli --cluster check` 找冲突的 slot
- 手工 `CLUSTER FAILOVER FORCE` 在期望的 master 节点上强制接管
- 极端情况：`CLUSTER RESET SOFT` + 重建 cluster（注意会丢非持久化数据）
- 检查 `cluster-require-full-coverage`（默认 yes，部分 slot 缺失时拒绝写）

### 7.2 RTO > 30 秒

- `cluster_node_timeout` 太大：调低到 10000ms（默认 15000ms）
- go-redis pool_size 太小，新连接握手排队：调大 pool_size
- DNS 解析慢：用 IP + sidecar resolver 替代

### 7.3 命令队列丢失（completed < 25）

- 检查 AOF 配置：`appendfsync everysec` 而非 `no`
- 看 `redis-cli INFO persistence | grep aof_last_write_status`，应为 ok
- worker 消费日志：是否有 "task not found in cmdq" 错误
- 致命情况：从 RDB 备份恢复（注意 RDB 5 分钟周期可能丢部分数据）

### 7.4 会话恢复率 < 90%

- 检查 session TTL 是否合理（推荐 5 min ≥ 2 × inform_interval）
- 看 ACS 日志是否有 "session not found, recreating" 大量记录
- CPE 端可能不发 Connection Request → 等待下个周期 Inform 才恢复，属正常

### 7.5 datamodel 缓存命中率长期低

- 检查 `datamodel:cache_version` key 是否被异常 INCR
- L1 内存缓存（sync.Map）应在 omcgo-app 重启后立即工作
- 必要时 `omcctl datamodel cache-warm` 主动预热

## 8. 回滚

### 8.1 演练失败回滚

```bash
# 8.1.1 停止 CPE 模拟器
ssh staging "pkill -f cpe_simulator.py"

# 8.1.2 清理演练数据
redis-cli -h redis-2.staging --scan --pattern "drill:*" | xargs redis-cli -h redis-2.staging DEL
redis-cli -h redis-2.staging --scan --pattern "acs:session:DRILL_REDIS_*" | xargs redis-cli -h redis-2.staging DEL
redis-cli -h redis-2.staging --scan --pattern "acs:cmdq:DRILL_REDIS_*" | xargs redis-cli -h redis-2.staging DEL
psql -h pg-master.staging -U omcgo -c \
  "DELETE FROM devices WHERE serial_number LIKE 'DRILL_REDIS_%';
   DELETE FROM tasks WHERE device_serial LIKE 'DRILL_REDIS_%';"

# 8.1.3 重启业务清理客户端状态
kubectl rollout restart deployment/omcgo-app
kubectl rollout restart deployment/omcgo-acs
kubectl rollout restart deployment/omcgo-worker
```

### 8.2 Cluster 严重损坏回滚

```bash
# 极端情况：从 RDB / AOF 备份恢复 Redis 数据
ssh redis-2.staging "systemctl stop redis-server"
ssh redis-2.staging "cp /var/backups/redis/dump.rdb /var/lib/redis/dump.rdb"
ssh redis-2.staging "systemctl start redis-server"
# 重建 cluster
redis-cli --cluster create redis-1.staging:6379 redis-2.staging:6379 redis-3.staging:6379 \
  --cluster-replicas 1 --cluster-yes
```

### 8.3 角色互换回滚

```bash
# 如果新 master 不稳定，主动切回老 master
redis-cli -h redis-1.staging CLUSTER FAILOVER  # 在原 master 节点上执行
```

## 9. 实测占位（待 staging 演练后回填）

```
首次演练: 2026-MM-DD by <user>
故障注入方式: A / B / C / D
故障窗口: ___ 秒
RTO 实测（go-redis 重连）: ___ 秒
TR-069 会话恢复率: ___% (___/100)
命令队列丢失数: ___ / 30
datamodel 缓存命中率（故障前/故障 5min/30min）: ___ / ___ / ___
redis_command_errors_total 增量: ___
Cluster 收敛耗时: ___ 秒
告警触发: RedisCommandErrorRateHigh / RedisPoolDepleted（firing 时长 ___ s）
判定: PASS / FAIL（详见附 verify-T-0026-redis-staging-drill.md）
回滚动作: 是 / 否（如是，原因: ___）
后续 action items: ___
```

---

## 附：关联文档

- `internal/core/components/redis.go` — go-redis/v9 客户端
- `internal/core/components/redis_metrics.go` — Redis Prometheus 指标（T-0061）
- `internal/acs/session.go` — 会话状态机
- `internal/acs/cmdqueue/redis_queue.go` — 命令队列实现
- `internal/config/datamodel/cache.go` — L2 Redis 缓存
- `omcgo/scripts/cpe_simulator.py` — CPE 模拟器
- `docs/消息队列全流程流转说明书.md` — 命令队列流转细节
- `docs/runbook/disaster-recovery.md` — DR 整体 Runbook（T-0024）
- `deployments/monitoring/alerts.yml` — Redis 5 条 AlertManager 规则（T-0061）

### Redis Key 命名速查（演练时常用）

```
acs:session:{device_serial}          — 会话状态 Hash（TTL 5 min）
acs:cmdq:{device_serial}             — 命令队列 Sorted Set
acs:heartbeat:{device_serial}        — 心跳时间戳（TTL = 2×inform_interval）
acs:connreq:pending:{device_serial}  — Connection Request 去重（TTL 30s）
datamodel:product:{carrier}:{tech}:{oui}:{product_class}
datamodel:oui:{carrier}:{tech}:{oui}
datamodel:default:{carrier}:{tech}
datamodel:resolve:{...}              — 解析结果缓存（TTL 1h）
datamodel:cache_version              — 跨实例协调
alarm:active:{device_serial}         — 活跃告警 Hash
ratelimit:inform:{device_serial}     — 限流计数器
```

---

*本 Runbook 由两半组成：framework（main 落地，本文件）+ staging 实测（外部）。
Runbook PASS 即视为 framework 部分 PASS；真实 RTO/会话恢复率/命令丢失数由 user staging 演练填回 §9。
预计单次演练耗时 60-90 分钟（含基线采样 5min + 注入 5min + 观察 30min + 验证 20min + 回滚 20min）。*
