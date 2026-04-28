# Runbook: PostgreSQL 主从切换 + pgxpool 重连演练

> **章程**: W3 收尾 / Backlog T-0026（Release Gate §3.4 5 类 Runbook 之一）
> **目的**: 验证 PostgreSQL 主库故障 / 主从切换时 OMC 三进程（app / acs / worker）pgxpool 自动重连，写入恢复 < 60 秒，无脏读 / 无丢数据
> **配套**:
> - `internal/core/components/postgres.go`（pgxpool 初始化与健康检查）
> - `internal/core/storage/audit.go`（pgx 慢查询审计 — T-0060）
> - `omcgo/scripts/db_backup.sh`（备份 + 异地上传 — T-0067）
> - `docs/runbook/db-backup-restore.md`（备份恢复流程 — T-0042）
> - `docs/runbook/disaster-recovery.md`（整机房 DR — T-0024）
> **首次执行日**: 待 user 在 staging 环境执行后回填 §9 实测日志

---

## 1. 演练目标（RTO / RPO）

| 维度 | 期望值 | 说明 |
|------|--------|------|
| 写入恢复时间（RTO） | < 60 秒 | 主从切换 + pgxpool 重连 + 应用层重试合计 |
| 数据丢失（RPO） | < 5 秒 | 同步流复制（synchronous_commit=on）下，主库 crash 时尚未 ack 的事务 |
| pgxpool 重连成功率 | 100% | 三进程（app / acs / worker）所有连接全部重建 |
| 切换期间业务降级 | 写入返回 503，读走 replica（如启用读写分离） | 不能返回脏数据 |
| 切换后慢查询比例 | 与切换前持平（±10%） | 验证 T-0060 慢查询审计仍生效 |
| 关键表数据一致性 | 100% | devices / alarms / tasks / audit_logs / data_model_definitions |

## 2. 前置条件

### 2.1 环境

- staging 环境部署 PostgreSQL 16 主从架构（推荐 Patroni + etcd 或 repmgr 管理切换）
  - 主库：`pg-master.staging:5432`
  - 备库：`pg-replica.staging:5432`
  - 复制方式：流复制（streaming replication），`synchronous_commit=on`
- staging 环境跑 omcgo-app（`:8081`）+ omcgo-acs（`:7547`）+ omcgo-worker（指标 `:9092`），三进程均连同一主库
- `cmd/{app,acs,worker}/etc/config.staging.yaml` 配置：
  ```yaml
  postgres:
    host: pg-master.staging   # 通过 VIP / DNS 指向当前主库
    port: 5432
    max_conns: 50
    min_conns: 10
    health_check_period: 30s
  ```
- TimescaleDB 扩展已加载（PM/KPI 时序表是超表）

### 2.2 监控

- Prometheus + Grafana 已起，关注指标：
  - `pg_pool_acquire_total / pg_pool_acquire_failed_total`（pgxpool 监控 — T-0061）
  - `pg_pool_active_conns / pg_pool_idle_conns`
  - `pg_slow_query_total{module=...}`（慢查询审计 — T-0060）
  - `pg_replication_lag_seconds`（推荐通过 postgres_exporter 暴露）
- AlertManager 5 条 pgxpool 告警规则已加载（T-0061）

### 2.3 工具

- `psql` 客户端
- `pg_isready` 健康检查
- 切换工具：`patronictl switchover` 或 `repmgr standby switchover`
- 业务流量发生器：curl 循环 / `omcgo/bin/loadtest`

### 2.4 备份保险

切换前必做：触发一次手工备份（避免演练失败时无热数据可恢复）：
```bash
ssh staging "bash /opt/omcgo/scripts/db_backup.sh --tag pre-failover-drill"
# 期望: backups/omcgo_pre-failover-drill_YYYYMMDDHHMM.sql.gz 落地 + 异地副本上传
```

## 3. 故障注入步骤

### 3.1 基线采样（演练前 5 分钟）

```bash
# 触发标记数据，记录基线时刻
psql -h pg-master.staging -U omcgo -c \
  "INSERT INTO audit_logs(action, target_id, payload) VALUES ('drill_baseline_pg', 'T-0026', '{\"phase\":\"pre\"}'::jsonb);"

# 记录基线指标
curl -s http://staging:9091/metrics | grep -E "pg_pool_acquire_total|pg_slow_query_total|pg_replication_lag_seconds"
date -u +"%Y-%m-%dT%H:%M:%SZ baseline"
```

期望: `pg_pool_acquire_failed_total` 接近 0，`pg_replication_lag_seconds < 1`。

### 3.2 故障注入 — 三种方式（任选其一）

**方式 A: Patroni 主动切换（推荐，最干净，模拟运维主动切换）**
```bash
ssh staging "patronictl -c /etc/patroni/patroni.yml switchover --master pg-master --candidate pg-replica --force"
# patroni 会优雅 demote 老主库 → promote 新主库 → 更新 VIP
sleep 5
```

**方式 B: 直接 kill PostgreSQL 主进程（模拟主库 crash）**
```bash
# staging 上找到 postmaster PID 强杀
ssh pg-master.staging "sudo -u postgres pg_ctl -D /var/lib/postgresql/16/main stop -m immediate"
# 等待 patroni / repmgr 自动 failover（约 10-30s）
sleep 30
# 验证新主库已 promote
ssh pg-replica.staging "psql -U postgres -c 'SELECT pg_is_in_recovery();'"  # 期望 false
```

**方式 C: 网络隔离（模拟主库网络分区）**
```bash
# staging 上 iptables drop 主库 5432 端口
ssh pg-master.staging "sudo iptables -A INPUT -p tcp --dport 5432 -j DROP"
sleep 60
# 演练完成后还原
ssh pg-master.staging "sudo iptables -D INPUT -p tcp --dport 5432 -j DROP"
```

### 3.3 故障期间持续写入（覆盖故障窗口）

```bash
# 每秒触发 1 次 audit_logs 写入，持续 120 秒（覆盖 RTO + buffer）
for i in $(seq 1 120); do
  curl -s -X POST -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"action\":\"drill_during_pg_$i\",\"target_id\":\"T-0026\"}" \
    http://staging:8081/api/v1/audit/test-write \
    -o /dev/null -w "%{http_code} " &
  sleep 1
done | tee pg-failover-during.log
```

期望:
- 故障窗口前 0-5 秒：HTTP 200（pgxpool 还有活跃连接）
- 故障窗口 5-30 秒：HTTP 503（pgxpool 重连中，应用层快速失败）
- 故障窗口 30-60 秒：HTTP 200 恢复（新主库就绪 + pgxpool 重建）

## 4. 故障期间观察

### 4.1 应用层日志

```bash
# omcgo-app 应连续打印 pgxpool 重连日志
tail -f /var/log/omcgo-app.log | grep -iE "pgx|connection|conn_str"
# 期望关键日志:
# - "pgxpool: failed to acquire connection: dial tcp ... connect: connection refused"
# - "pgxpool: reconnecting to host=..."
# - "pgxpool: connection established"（恢复信号）
```

### 4.2 Prometheus 指标

```bash
# 故障窗口内每 5 秒采一次
while true; do
  curl -s http://staging:9091/metrics | \
    grep -E "^pg_pool_(acquire|active|idle)" | head -10
  date -u +"%Y-%m-%dT%H:%M:%SZ"
  echo "---"
  sleep 5
done
```

期望趋势:
- `pg_pool_acquire_failed_total` 在故障窗口快速增加，恢复后增速归零
- `pg_pool_active_conns` 故障期间 → 0，恢复后回升至基线
- `pg_slow_query_total` 切换瞬间可能 spike（新主库冷缓存），10 分钟内回落

### 4.3 AlertManager 告警

```bash
# 应触发 pgxpool 告警（T-0061 5 条规则之一）
curl -s http://staging:9093/api/v2/alerts | \
  jq '.[] | select(.labels.alertname | contains("PgPool")) | {alertname: .labels.alertname, state: .status.state}'
# 期望: PgPoolAcquireFailRateHigh / PgPoolDepleted 告警 firing → 切换完成后 resolved
```

### 4.4 复制延迟

```bash
# 在新主库（原 replica）查 WAL 接收位点
ssh pg-replica.staging "psql -U postgres -c \
  'SELECT pg_current_wal_lsn(), pg_last_wal_receive_lsn(), pg_last_wal_replay_lsn();'"
# 切换完成后，pg_is_in_recovery() = false，wal_receive_lsn = wal_replay_lsn
```

## 5. 故障恢复

### 5.1 自动恢复路径（期望）

1. **切换完成（0-30s）**: Patroni / repmgr 完成 promote，VIP 漂移到新主库
2. **pgxpool 重连（30-50s）**:
   - pgx 连接发现 EOF / connection refused → pool 标记连接失效 → 后续 acquire 触发重建
   - 重建用 config 中的 host（VIP 已指向新主库）
3. **业务恢复（50-60s）**: 新连接握手成功 → audit_logs 写入恢复 200

### 5.2 手动介入（如自动恢复失败）

如果 RTO > 60 秒：

```bash
# 1. 确认新主库就绪
psql -h pg-master.staging -U omcgo -c "SELECT pg_is_in_recovery();"  # 应 false

# 2. 验证应用层 DNS / VIP 解析正确
ssh staging "getent hosts pg-master.staging"  # 应解析到新主库 IP

# 3. 强制重启 omcgo-app（pgxpool 全量重建）
ssh staging "systemctl restart omcgo-app omcgo-acs omcgo-worker"
sleep 10

# 4. 健康检查
curl -s http://staging:8081/health | jq '.checks.postgres'  # 期望 "ok"
```

## 6. 验证

### 6.1 数据完整性

```bash
# 6.1.1 标记数据写入数（120 个 drill_during_pg_*）
psql -h pg-master.staging -U omcgo -c \
  "SELECT COUNT(*) FROM audit_logs WHERE action LIKE 'drill_during_pg_%';"
# 期望: 接近 120（允许故障窗口 5-30s 期间约 25-40 个 503 失败）
# 实际丢失数 = 120 - count，应 <= 30

# 6.1.2 检查无脏数据 / 无重复主键
psql -h pg-master.staging -U omcgo -c \
  "SELECT action, COUNT(*) FROM audit_logs WHERE action LIKE 'drill_during_pg_%' GROUP BY action HAVING COUNT(*) > 1;"
# 期望: 0 行（每个 action 唯一）

# 6.1.3 关键表行数与切换前快照对比
psql -h pg-master.staging -U omcgo -c \
  "SELECT 'devices' AS t, COUNT(*) FROM devices
   UNION ALL SELECT 'alarms', COUNT(*) FROM alarms
   UNION ALL SELECT 'tasks', COUNT(*) FROM tasks
   UNION ALL SELECT 'data_model_definitions', COUNT(*) FROM data_model_definitions;"
# 期望: 与 §3.1 基线对比，差值仅来自故障期间业务流量，无负数 / 无骤减
```

### 6.2 pgxpool 健康

```bash
# 6.2.1 三进程 pool 全部健康
for svc in 8081 7557 9092; do
  curl -s http://staging:$svc/metrics 2>/dev/null | grep "^pg_pool_active_conns"
done
# 期望: 每个进程 active_conns 在 [min_conns, max_conns] 区间

# 6.2.2 acquire 成功率回到基线
curl -s http://staging:9091/metrics | grep -E "pg_pool_acquire_(total|failed_total)"
# 期望: failed_total 在切换后 5 分钟内不再增长
```

### 6.3 慢查询审计仍生效

```bash
psql -h pg-master.staging -U omcgo -c \
  "SELECT module, COUNT(*) FROM slow_query_log
   WHERE created_at >= NOW() - INTERVAL '10 minutes'
   GROUP BY module;"
# 期望: 各模块仍有慢查询记录（证明 T-0060 审计 hook 在新主库仍生效）
```

### 6.4 业务功能 smoke

```bash
# devices 列表分页
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://staging:8081/api/v1/devices?page=1&size=10" | jq '.total'
# alarms 实时
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://staging:8081/api/v1/alarms/active" | jq 'length'
# tasks 队列
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://staging:8081/api/v1/tasks?status=pending" | jq '.total'
# 期望: 三个端点全 200，返回数据合理
```

## 7. 失败处理

### 7.1 自动 failover 未触发（Patroni / repmgr 卡住）

- 排查 Patroni leader lock：`patronictl list` 看 Leader 字段是否长时间 unknown
- etcd / consul 集群健康：`etcdctl endpoint status --cluster`
- 手工 promote：`ssh pg-replica.staging "sudo -u postgres pg_ctl promote -D /var/lib/postgresql/16/main"`
- 更新 VIP / DNS 指向新主库

### 7.2 pgxpool 重连卡死（RTO > 120s）

- 检查 pgx 配置 `health_check_period`（默认 1m，太大会延迟探测失效）
- pgx v5 已有 `MaxConnLifetimeJitter`，应自动剔除老连接
- 终极方案：重启 omcgo-app 进程（pool 全量重建）

### 7.3 数据丢失数 > 30（实际 < 90 写入成功）

- 检查 `synchronous_commit` 是否为 on（`SHOW synchronous_commit;`）
- 看 PostgreSQL 主库 crash 时刻日志：`grep -i "crash\|panic" /var/log/postgresql/postgresql-*.log`
- 评估 RPO：若 > 5s 业务窗口的数据丢失，需调大 `wal_keep_size` 或启用同步备库（`synchronous_standby_names`）

### 7.4 切换后出现脏读 / 复制冲突

- 立刻停止业务流量（`kubectl scale deployment omcgo-app --replicas=0`）
- 比对新旧主库的 `pg_current_wal_lsn`，找出未传输的 WAL
- 用 `pg_waldump` 提取冲突事务，人工对账
- 如必要，从 §2.4 的预演练备份恢复（参考 db-backup-restore.md）

### 7.5 演练后 replica 未自动重建

- 老主库降级为 replica 失败 → 用 `pg_basebackup` 重新拉一遍：
  ```bash
  ssh pg-master.staging "sudo -u postgres pg_basebackup \
    -h pg-replica.staging -D /var/lib/postgresql/16/main \
    -U replicator -W -X stream -P"
  ```
- 检查 `recovery.conf` / `postgresql.auto.conf` 的 `primary_conninfo` 指向新主库

## 8. 回滚

### 8.1 演练失败回滚（数据不一致 / 业务受损）

```bash
# 8.1.1 暂停业务流量
kubectl scale deployment omcgo-app --replicas=0
kubectl scale deployment omcgo-acs --replicas=0
kubectl scale deployment omcgo-worker --replicas=0

# 8.1.2 从预演练备份恢复
bash /opt/omcgo/scripts/db_restore.sh \
  --backup backups/omcgo_pre-failover-drill_*.sql.gz \
  --target pg-master.staging
# 详见 docs/runbook/db-backup-restore.md §4

# 8.1.3 验证恢复后行数
psql -h pg-master.staging -U omcgo -c "SELECT COUNT(*) FROM audit_logs;"

# 8.1.4 重启业务
kubectl scale deployment omcgo-app --replicas=2
kubectl scale deployment omcgo-acs --replicas=2
kubectl scale deployment omcgo-worker --replicas=1
```

### 8.2 演练数据清理（PASS 但不希望污染基线）

```sql
-- 删除演练插入的测试数据
DELETE FROM audit_logs WHERE action LIKE 'drill_baseline_pg' OR action LIKE 'drill_during_pg_%';
-- 重置 pgxpool 监控计数器（重启 omcgo-app 即可）
```

### 8.3 切换方向回滚（角色互换）

如果新主库稳定性不如老主库：
```bash
ssh staging "patronictl switchover --master pg-replica --candidate pg-master --force"
# 等同于反向再做一次 §3.2 方式 A
```

## 9. 实测占位（待 staging 演练后回填）

```
首次演练: 2026-MM-DD by <user>
故障注入方式: A / B / C
故障窗口: ___ 秒
RTO 实测: ___ 秒（写入 503 → 200 恢复）
RPO 实测: ___ 秒（最后一个成功 commit → 切换瞬间）
故障期间写入成功数: ___ / 120
故障期间写入失败数: ___ / 120
pgxpool 重连耗时: ___ 秒
pg_pool_acquire_failed_total 增量: ___
pg_slow_query_total 切换前/后/恢复后: ___ / ___ / ___
关键表行数差异（devices/alarms/tasks）: ___ / ___ / ___
告警触发: PgPoolAcquireFailRateHigh / PgPoolDepleted（firing 时长 ___ s）
判定: PASS / FAIL（详见附 verify-T-0026-pg-staging-drill.md）
回滚动作: 是 / 否（如是，原因: ___）
```

---

## 附：关联文档

- `internal/core/components/postgres.go` — pgxpool 初始化
- `internal/core/components/postgres_metrics.go` — pgxpool Prometheus 指标（T-0061）
- `internal/core/storage/audit.go` — 慢查询审计 hook（T-0060）
- `omcgo/migrations/000NNN_*.sql` — 11 条性能索引（T-0060）
- `omcgo/scripts/db_backup.sh` — 备份脚本 + 异地上传（T-0067）
- `docs/runbook/db-backup-restore.md` — 备份恢复 Runbook（T-0042）
- `docs/runbook/disaster-recovery.md` — DR 整体 Runbook（T-0024）
- `deployments/monitoring/alerts.yml` — pgxpool 5 条 AlertManager 规则（T-0061）

---

*本 Runbook 由两半组成：framework（main 落地，本文件）+ staging 实测（外部）。
Runbook PASS 即视为 framework 部分 PASS；真实 RTO/RPO 数字由 user staging 演练填回 §9。
预计单次演练耗时 60-90 分钟（含基线采样 5min + 注入 5min + 观察 30min + 验证 20min + 回滚 20min）。*
