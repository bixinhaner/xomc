# T-0026 验证报告 — Runbook ≥ 5 场景达成

> **任务**：W3 收尾 / Backlog T-0026 — Release Gate §3.4 要求 Runbook ≥ 5 类故障场景
> **执行人**：T-0026 sub-agent（worktree-agent-t0026）
> **日期**：2026-04-28
> **范围**：仅新增 `docs/runbook/*.md`（不动代码、不动 backlog/charter，无 commit/push）

---

## 1. 章程 Pass 标准

| 标准 | 命令 | 期望 |
|------|------|------|
| Runbook 总数 ≥ 5 | `ls docs/runbook/*.md \| wc -l` | ≥ 5 |
| 每份新 Runbook ≥ 200 行 | `wc -l docs/runbook/<new>.md` | ≥ 200 |
| 9 章结构完整 | `grep -c "^## [1-9]\." docs/runbook/<new>.md` | = 9 |
| 故障注入步骤可执行 | 含 docker / kubectl / iptables / ssh 具体命令 | 是 |
| 实测占位"待 staging 演练后回填" | grep "实测占位" 章节 | 存在 |

## 2. 实测结果

### 2.1 Runbook 总数（≥ 5 验证）

```bash
$ ls docs/runbook/*.md | wc -l
6
```

| # | 文件 | 行数 | 来源 task | 状态 |
|---|------|------|-----------|------|
| 1 | `docs/runbook/nats-failover.md` | 154 | T-0059 / W3.E.3 | 既有 |
| 2 | `docs/runbook/disaster-recovery.md` | 388 | T-0024 / T-0067 | 既有 |
| 3 | `docs/runbook/db-backup-restore.md` | 184 | T-0042 | 既有 |
| 4 | **`docs/runbook/pg-failover.md`** | **390** | **T-0026（新）** | **本次** |
| 5 | **`docs/runbook/redis-failover.md`** | **455** | **T-0026（新）** | **本次** |
| 6 | **`docs/runbook/acs-overload.md`** | **479** | **T-0026（新）** | **本次** |

**总数 6 ≥ 5 ✅**

### 2.2 新增 Runbook 行数（≥ 200 验证）

```bash
$ wc -l docs/runbook/pg-failover.md docs/runbook/redis-failover.md docs/runbook/acs-overload.md
390 docs/runbook/pg-failover.md       # ≥ 200 ✅
455 docs/runbook/redis-failover.md    # ≥ 200 ✅
479 docs/runbook/acs-overload.md      # ≥ 200 ✅
```

### 2.3 9 章结构完整性

| Runbook | §1 目标 | §2 前置 | §3 故障注入 | §4 期间观察 | §5 故障恢复 | §6 验证 | §7 失败处理 | §8 回滚 | §9 实测占位 | 计数 |
|---------|---------|---------|-------------|--------------|-------------|---------|-------------|---------|--------------|------|
| pg-failover | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | 9/9 |
| redis-failover | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | 9/9 |
| acs-overload | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | 9/9 |

```bash
$ grep -c "^## [1-9]\." docs/runbook/pg-failover.md
9
$ grep -c "^## [1-9]\." docs/runbook/redis-failover.md
9
$ grep -c "^## [1-9]\." docs/runbook/acs-overload.md
9
```

### 2.4 故障注入可执行命令（sample）

| Runbook | 命令样本 |
|---------|----------|
| pg-failover §3.2 方式 A | `patronictl -c /etc/patroni/patroni.yml switchover --master pg-master --candidate pg-replica --force` |
| pg-failover §3.2 方式 B | `ssh pg-master.staging "sudo -u postgres pg_ctl -D /var/lib/postgresql/16/main stop -m immediate"` |
| pg-failover §3.2 方式 C | `ssh pg-master.staging "sudo iptables -A INPUT -p tcp --dport 5432 -j DROP"` |
| redis-failover §3.2 方式 A | `ssh redis-1.staging "pkill -9 redis-server"` |
| redis-failover §3.2 方式 C | `redis-cli -h sentinel-1.staging -p 26379 SENTINEL FAILOVER mymaster` |
| redis-failover §3.2 方式 D | `docker compose -f deployments/docker/docker-compose.yml stop redis` |
| acs-overload §3.2 方式 A | `python3 /opt/omcgo/scripts/cpe_simulator.py --count 1000 --bootstrap-burst` |
| acs-overload §3.2 方式 D | `loadtest --schedule 'ramp:200@1m,500@1m,1000@2m,2000@2m,5000@3m' --duration 9m` |

每份 Runbook §3 提供 **3-4 种独立故障注入方式**，覆盖：进程 kill / 网络分区 / 容器重启 / 主动切换 / 流量打压等多种触发路径。

### 2.5 实测占位章节存在性

```bash
$ grep -A 1 "^## 9. 实测占位" docs/runbook/pg-failover.md
## 9. 实测占位（待 staging 演练后回填）

$ grep -A 1 "^## 9. 实测占位" docs/runbook/redis-failover.md
## 9. 实测占位（待 staging 演练后回填）

$ grep -A 1 "^## 9. 实测占位" docs/runbook/acs-overload.md
## 9. 实测占位（待 staging 演练后回填）
```

每份 Runbook §9 含 12-18 个待回填字段（RTO / RPO / 错误率 / 资源峰值 / 告警触发 / 判定 PASS|FAIL / 回滚动作），与 T-0059 / T-0067 模式一致。

## 3. 三类场景选型理由

| Runbook | 关联章程 task | 业务关键性 | 选型理由 |
|---------|--------------|-----------|---------|
| **pg-failover.md** | T-0042 备份 + T-0067 异地恢复 | 极高 | DB 是 OMC 唯一持久化主存（设备/告警/任务/审计），主从切换是商用网管必演练 |
| **redis-failover.md** | F01 ACS 会话状态依赖 | 极高 | TR-069 会话、命令队列、L2 缓存全在 Redis；故障直接影响南向通道 |
| **acs-overload.md** | T-0041 ratelimit + 章程 333 sessions/s 基线 | 高 | 整片基站重启、网络故障恢复都会触发 Inform 风暴；不演练 = 上线后必崩 |

未选 alarm-storm.md / license-expiry.md，理由：
- alarm-storm 与 F04 W2.A 重叠较多（已在 W2.A.5 78.4% 覆盖率验证去重路径）
- license-expiry 关联 T-0015 仍计划中，演练对象（License 模块）尚未完整落地

## 4. 与现有 Runbook 互补关系

```
disaster-recovery.md (T-0024)            ← 整机房 DR（最大灾难）
├── db-backup-restore.md (T-0042)        ← 数据层备份恢复（已落地，RTO 1.034s 实测）
├── pg-failover.md (T-0026 NEW)          ← DB 主从切换（亚机房级故障）
│
├── redis-failover.md (T-0026 NEW)       ← Redis 集群分区（影响 ACS 会话）
├── nats-failover.md (T-0059)            ← NATS JetStream 故障（影响事件流）
│
└── acs-overload.md (T-0026 NEW)         ← ACS 容量极限（流量层）
```

形成 **DR (整机房) → 数据层 (PG/Redis/NATS) → 流量层 (ACS) 三段防御** 的完整 Runbook 体系。

## 5. Pass 判定

| 标准 | 期望 | 实测 | 结论 |
|------|------|------|------|
| Runbook 总数 ≥ 5 | ≥ 5 | **6** | ✅ |
| 新增 ≥ 3 | ≥ 3 | **3**（pg/redis/acs）| ✅ |
| 每份 ≥ 200 行 | ≥ 200 | 390 / 455 / 479 | ✅ |
| 9 章结构完整 | = 9 | 9 / 9 / 9 | ✅ |
| 故障注入可执行 | 有具体命令 | docker/kubectl/iptables/ssh/patronictl/redis-cli 全覆盖 | ✅ |
| 实测占位章节 | 存在 + "待 staging 回填" | 全 3 份均有 | ✅ |

**T-0026 PASS ✅**

## 6. 限制说明

- **未在 staging 真实演练**：Runbook framework 落地，真实 RTO / RPO / 错误率数字待 user 在 staging 环境演练后填回 §9（与 T-0059 / T-0067 同模式）
- **告警规则名称为预期态**：Runbook 中引用的 PgPoolAcquireFailRateHigh / RedisCommandErrorRateHigh / ACSHighInformRate 三组告警，前两组已在 T-0061 落地（`deployments/monitoring/alerts.yml` 5 + 5 条），ACS 告警待 W4 容量演练后定阈值
- **Patroni / Sentinel 工具假设**：Runbook §3.2 方式 A/C 假设 staging 已部署集群协调工具；如 staging 仍是单实例，可走方式 B/D（直接 kill / docker stop）

## 7. 关联 backlog / charter 项

- **Release Gate §3.4** — Runbook ≥ 5 类故障场景：本任务直接解锁
- **W3 Wave 退出门槛** — Runbook 完整性是 W3 章程要素之一
- **后续 staging 演练任务**：建议新增 T-0070 ~ T-0072 三项 staging 演练 backlog 把 §9 实测填回

## 8. 文件清单

| 文件 | 操作 | 行数 |
|------|------|------|
| `docs/runbook/pg-failover.md` | 新建 | 390 |
| `docs/runbook/redis-failover.md` | 新建 | 455 |
| `docs/runbook/acs-overload.md` | 新建 | 479 |
| `docs/review-report/20260428/verify-T-0026.md` | 新建（本文件） | — |
| `.wave-progress.log` | 新建 | 心跳 |
| `.wave-status.txt` | 新建 | DONE |

不动文件（章程禁动）：
- `omcgo/` `omcmb/` `.github/` `deployments/` `scripts/`
- `docs/project/backlog.md` `docs/methodology/` `docs/project/release-gate.md`
- `docs/runbook/nats-failover.md` `docs/runbook/disaster-recovery.md` `docs/runbook/db-backup-restore.md`（既有）

---

*本验证报告对应 dev-pipeline --fast docs 快通模式：3 份新建 Runbook + 验证报告 + 心跳协议，
不引入新依赖、不动代码、不动章程类文档。主会话 sub-agent 收尾后由 user 进 worktree commit。*
