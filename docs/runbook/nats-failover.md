# Runbook: NATS JetStream 故障演练 + 故障恢复

> **章程**: W3.E.3 / Backlog T-0059
> **目的**: 验证 NATS 实例故障时 OMC 服务自动重连 + 关键事件不丢
> **配套**: `internal/core/event/nats_bus.go` (NATSEventBus 实现) · `docs/eventbus/topics.md` (subject 清单)
> **首次执行日**: 待 user 在 staging 环境执行后回填 §6 实测日志

---

## 1. 演练目标

| 维度 | 期望 |
|------|------|
| 重连恢复时间 (RTO) | < 30 秒（NATSEventBus 内置 reconnect_wait=2s × max_reconnect=-1 无限重试）|
| 事件丢失率 | 0%（JetStream 持久化 + Durable Consumer + Ack/Nak/Term 重投递）|
| 重连期间 Publish 行为 | 排队（client buffer）或 fail-fast（视配置）|
| 关键消费者 (alarm/device/command/task/pm) | 全部 resume 订阅，无消息漏处理 |

## 2. 前置条件

- staging 环境跑 NATS JetStream 单实例或集群（推荐 3 节点）
- `cmd/app/etc/config.staging.yaml` 配置 `nats.url: nats://<staging-nats>:4222`
- omcgo-app + omcgo-acs + omcgo-worker 三进程已起，Subscribe 5 类关键 subject
- Prometheus + Grafana 已起（W1.7），观察指标 `nats_reconnect_total / nats_subscription_active / event_publish_failed_total`

## 3. 演练步骤

### 3.1 基线采样（演练前 5 分钟）

```bash
# 触发 5 个测试事件（每类 1 个）
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -d '{"alarm_identifier":"DRILL_BASELINE","severity":3}' \
  http://staging:8081/api/v1/alarms/test-publish

# 同样: device.inform / command.set_parameters_response / task.complete / pm.file.parsed

# 记录 baseline 指标
curl -s http://staging:9091/metrics | grep -E "nats_reconnect_total|event_publish_failed_total"
```

期望: 5 事件全部 PASS，`event_publish_failed_total = 0`。

### 3.2 故障注入

**方式 A: 直接 kill NATS 进程（recover 同节点）**
```bash
# staging 上找到 nats-server PID
ssh staging "pgrep nats-server | head -1 | xargs kill -9"
sleep 5
# 应自动 systemd / docker 重启
```

**方式 B: 网络隔离（模拟跨机房断链）**
```bash
# staging 上 iptables drop NATS 端口
ssh staging "sudo iptables -A INPUT -p tcp --dport 4222 -j DROP"
# 演练完成后还原
ssh staging "sudo iptables -D INPUT -p tcp --dport 4222 -j DROP"
```

**方式 C: docker compose stop + start（推荐，最干净）**
```bash
docker compose -f deployments/docker/docker-compose.yml stop nats
sleep 30   # 模拟 30 秒不可用
docker compose -f deployments/docker/docker-compose.yml start nats
```

### 3.3 故障期间持续 Publish

```bash
# 每秒触发 1 个 alarm.raised 事件，持续 60 秒（覆盖故障窗口）
for i in $(seq 1 60); do
  curl -s -X POST -H "Authorization: Bearer $TOKEN" \
    -d "{\"alarm_identifier\":\"DRILL_DURING_$i\",\"severity\":3}" \
    http://staging:8081/api/v1/alarms/test-publish
  sleep 1
done
```

### 3.4 故障恢复后验证

```bash
# NATS 起来后等 30 秒让 client 重连
sleep 30

# 1. 检查重连指标
curl -s http://staging:9091/metrics | grep nats_reconnect_total
# 期望: nats_reconnect_total ≥ 1

# 2. 检查 publish 失败计数（重连期间允许小量 fail）
curl -s http://staging:9091/metrics | grep event_publish_failed_total
# 期望: < 5（取决于 client buffer 配置；若 fail-fast 模式可能更高）

# 3. 检查 60 秒持续 Publish 的事件落地
psql -h staging -U omc -d omc -c \
  "SELECT count(*) FROM alarm_history WHERE alarm_identifier LIKE 'DRILL_DURING_%';"
# 期望: 60 (零丢失) 或 ≥ 50（5 个以下重投递失败可接受，需 root cause)

# 4. 检查 JetStream stream/consumer 健康
nats stream ls -s nats://staging:4222
nats consumer ls OMC_EVENTS -s nats://staging:4222
# 期望: stream 存在，所有 consumer 在线 + 0 num_pending（未处理消息归零）
```

## 4. 失败处理

### 4.1 重连失败（RTO > 30s）
- 检查 NATSEventBus 配置 `nats.reconnect_wait` / `max_reconnect`（应 -1 无限重试）
- 看 zap 日志：`grep "nats reconnect" /var/log/omcgo-app.log`
- 客户端 `nats.Conn.IsConnected()` 状态可能 stuck CLOSED → 升级 nats.go 客户端版本

### 4.2 事件丢失（< 60 / 60）
- 检查 NATSEventBus 是否用了 ConsumerWithDurable（必须，否则重启丢消费位点）
- 检查 maxDeliveries 配置（current 5）— 重投递超限会 Term，event_publish_failed_total
  对应 maxDeliveries 超限的 dead-letter
- 看 NATS server log：`docker logs docker-nats-1 | grep -i "drop\|expir"`

### 4.3 多实例间 QueueSubscribe 失衡
- 用 `nats consumer info OMC_EVENTS <consumer> -s ...` 看 num_redelivered
- 若一个实例 redelivered 远高于其他 → 该实例消费慢，扩容或排查处理逻辑

## 5. 回滚（演练失败时）

如演练触发数据不一致：
- alarm: 删演练插入的 `WHERE alarm_identifier LIKE 'DRILL_%'`
- audit_logs: 同上 `WHERE action LIKE 'drill_%'`
- 重启 omcgo-app 让 NATSEventBus 重新 Subscribe

## 6. 实测日志（待 user 在 staging 演练后回填）

```
首次演练: 2026-MM-DD by <user>
故障注入方式: A / B / C
故障窗口: 30s / 60s / ...
重连指标 nats_reconnect_total: ___
事件落地率: ___ / 60
JetStream consumer redelivered: ___
RTO 实测: ___ s
判定: PASS / FAIL（详见附 verify-T-0059-staging-drill.md）
```

## 7. 关联

- `internal/core/event/nats_bus.go` — NATSEventBus 实现
- `internal/core/event/types.go` — EventBus 接口
- `internal/core/event/subjects.go` — subject 常量
- `docs/eventbus/topics.md` — 5 类主题清单
- `omcgo/test/integration/nats_failover_test.go` — 集成测试 stub（t.Skip 在无 staging 时）

---

*章程 W3.E.3 由两半组成：本 Runbook（main 落地）+ user staging 实测（外部）。
Runbook PASS 即视为 framework 部分 PASS；真实 RTO/丢失率数字由 staging 演练填回。*
