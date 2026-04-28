# OMC 灾难恢复（DR）Runbook

> **任务编号**：T-0067（Wave 3 W3.H.3 — 异地备份扩展 + DR Runbook framework）
> **基础任务**：T-0042（Wave 1 W1.8 — 本地 PG 备份 + 恢复演练，见 `docs/runbook/db-backup-restore.md`）
> **首次落地**：2026-04-28
> **维护方**：运维与可观测性专家 Owner（暂列：xieguiya）
> **关联文件**：
> - 备份脚本：`omcgo/scripts/db_backup.sh`（已扩展异地 S3/MinIO 上传）
> - 演练脚本：`omcgo/scripts/db_restore_drill.sh`
> - 本地备份策略：`docs/runbook/db-backup-restore.md`
> - cron 配置：`deployments/cron/db-backup.cron`

---

## 1. 目标（RTO / RPO）

| 指标 | 目标 | 含义 |
|------|------|------|
| **RTO（Recovery Time Objective）** | **≤ 1 小时** | 灾难发生 → 业务恢复可用，端到端时长 |
| **RPO（Recovery Point Objective）** | **≤ 15 分钟** | 灾难发生时丢失数据时间窗（最近 15 分钟内的备份必须可用） |

### 1.1 RTO 拆解（1 小时预算分配）

| 阶段 | 预算 | 说明 |
|------|------|------|
| 故障识别 + 决策 | ≤ 5 min | 监控告警 + 值班决断 |
| 异地备份拉取 + 校验 | ≤ 10 min | `mc cp` / `aws s3 cp` + sha256 |
| 新 PG 实例启动 | ≤ 5 min | docker compose up / k8s deployment apply |
| pg_restore 还原 | ≤ 25 min | 取决于数据量与并发度（`--jobs=4`） |
| 配置切换 + omcgo 重启 | ≤ 10 min | config 改 host + start-all |
| 校验 + 冒烟 | ≤ 5 min | E2E 6 项校验（参考 §4.6） |

### 1.2 RPO ≤ 15 分钟的实现路径

W1.8 base 仅做每日全量备份（RPO = 24h），不达标。本任务（W3.H.3）落地 framework，待两条增量路径补齐后才能真正满足 RPO ≤ 15min：

| 路径 | 状态 | 备注 |
|------|------|------|
| **每日全量** + **WAL 持续归档**（`pgbackrest` / `wal-g`） | TBD（投产前必补） | 工业标准方案，PITR 友好；下一个 wave 补任务 |
| **每 15 min 增量 dump**（短期方案） | 可选 | 通过 cron `*/15 * * * *` 触发本脚本，dump 文件大但配合异地上传可临时满足 |
| **逻辑复制 / Streaming Replica（异地 standby）** | 长期方案 | 需要异地 PG 实例，运维成本高，但 RPO 可达秒级 |

> **当前 framework 提供的能力**：脚本 + Runbook + 演练流程已就绪；RPO ≤ 15min **数字** 待 staging 真演练 + 增量备份方案落地后回填本文档 §6。

---

## 2. 前置条件

### 2.1 网络与凭据

| 项目 | 要求 |
|------|------|
| 异地 S3/MinIO endpoint | 与生产网络互通；建议私网专线（VPN / IPSec），不走公网 |
| 凭据（access/secret key） | 放 KMS / Vault / k8s secret，**严禁明文写在 cron 行内** |
| bucket 命名 | `omc-backups-offsite-<env>`，启用版本化（versioning），开启 lifecycle 90 天后转冷存储 |
| 加密 | bucket 启用 SSE-S3 / SSE-KMS；客户端可叠加 `age` / `gpg` 加密（密钥另存） |
| 网络带宽 | 估算：每日全量 ≤ 10 GB → 至少 50 Mbps 上行；高峰小时不应阻塞业务 |

### 2.2 工具

恢复机（备机房）必须预装：

- `pg_restore` / `psql`（PostgreSQL 16 client，与生产同 major 版本）
- `mc`（MinIO client）**或** `aws` cli v2
- `sha256sum`（macOS 是 `shasum -a 256`）
- 脚本仓库：`omcgo/scripts/db_backup.sh`、`omcgo/scripts/db_restore_drill.sh`、`docs/runbook/disaster-recovery.md`

### 2.3 环境变量（备机房 .env / vault）

```bash
# 异地 S3（备份源）
export OFFSITE_S3_ENDPOINT="https://minio.dr.example.com"
export OFFSITE_S3_BUCKET="omc-backups-offsite-prod"
export OFFSITE_S3_REGION="cn-east-1"
export OFFSITE_S3_ACCESS_KEY="***"
export OFFSITE_S3_SECRET_KEY="***"
export OFFSITE_S3_PREFIX="backups/prod/"

# 新 PG 实例（备机房 / 灾备机房）
export PGHOST="pg-dr.example.com"
export PGPORT="5432"
export PGUSER="omcgo"
export PGPASSWORD="***"   # 或 ~/.pgpass
export PGDATABASE="omcgo"
```

---

## 3. 故障场景与应对策略

| 场景 | 触发条件 | 决策路径 | RTO 影响 |
|------|---------|---------|---------|
| **A. 单点故障（主 PG 进程崩溃）** | systemd 重启失败 / OOM / 磁盘临时不可写 | 优先在原机房 fix（重启服务、扩盘）；不动备份 | 通常 ≤ 15 min |
| **B. 整机房故障 / 主 PG 数据卷损坏** | 机房断电、机柜失火、主存储阵列损坏 | 启动本 Runbook 完整流程（§4），切到备机房新 PG 实例 | 目标 ≤ 1h |
| **C. 数据损坏 / 误操作（误 DROP / TRUNCATE）** | 业务报"数据消失" / DBA 操作失误 | §5 单表/PITR 恢复，**不**切机房 | 取决于损坏面：< 30 min |
| **D. 异地 S3 不可达** | DR 演练或定期巡检发现异地拉不下来 | 升级 P1，立刻排查（不切机房，但停止主机房写盘风险） | 不影响 RTO 但影响 RPO |
| **E. 全量备份损坏** | sha256 校验不通过 / pg_restore 失败 | 回退到上一份全量；同时启动取证 | 增加 ≤ 25 min（额外一次 pg_restore） |

**决策矩阵简化口诀**：
- 进程级问题 → 不切机房
- 数据级问题（不丢机器）→ 不切机房，PITR / 单表恢复
- 机器级 / 机房级问题 → 切机房（走 §4）

---

## 4. 完整恢复步骤（场景 B：整机房故障）

> 假设：主机房不可用 / 主 PG 数据卷损坏不可修复，需在备机房用最近异地备份重建。

### 4.1 拉取最新异地备份

```bash
# 假设 mc 别名已配置；首次执行：
mc alias set omcoffsite "${OFFSITE_S3_ENDPOINT}" \
    "${OFFSITE_S3_ACCESS_KEY}" "${OFFSITE_S3_SECRET_KEY}" --api S3v4

# 列出最新 5 个备份并选最新
mc ls --reverse "omcoffsite/${OFFSITE_S3_BUCKET}/${OFFSITE_S3_PREFIX:-backups/prod/}" | head -10

# 拉取最新 dump + meta（按文件名时间戳排序）
LATEST=$(mc ls --reverse "omcoffsite/${OFFSITE_S3_BUCKET}/${OFFSITE_S3_PREFIX:-backups/prod/}" \
    | grep '\.dump$' | head -1 | awk '{print $NF}')
echo "selected: ${LATEST}"

mkdir -p ./dr-restore
mc cp "omcoffsite/${OFFSITE_S3_BUCKET}/${OFFSITE_S3_PREFIX:-backups/prod/}${LATEST}" ./dr-restore/
mc cp "omcoffsite/${OFFSITE_S3_BUCKET}/${OFFSITE_S3_PREFIX:-backups/prod/}${LATEST%.dump}.meta" ./dr-restore/
```

**aws cli 等价命令**：

```bash
aws --endpoint-url "${OFFSITE_S3_ENDPOINT}" --region "${OFFSITE_S3_REGION}" \
    s3 ls "s3://${OFFSITE_S3_BUCKET}/${OFFSITE_S3_PREFIX:-backups/prod/}" | sort -r | head -10
aws --endpoint-url "${OFFSITE_S3_ENDPOINT}" --region "${OFFSITE_S3_REGION}" \
    s3 cp "s3://${OFFSITE_S3_BUCKET}/${OFFSITE_S3_PREFIX:-backups/prod/}${LATEST}" ./dr-restore/
```

**RTO 计时点 1**：记录拉取开始时间 `T0`。

### 4.2 验证 sha256 完整性

```bash
DUMP="./dr-restore/${LATEST}"
META="./dr-restore/${LATEST%.dump}.meta"

EXPECTED=$(grep '^sha256=' "${META}" | cut -d= -f2)
ACTUAL=$(sha256sum "${DUMP}" | awk '{print $1}')
# macOS: shasum -a 256 "${DUMP}" | awk '{print $1}'

if [[ "${EXPECTED}" != "${ACTUAL}" ]]; then
    echo "[FATAL] sha256 不匹配 — 备份损坏，回退上一份"
    exit 1
fi
echo "[OK] sha256 校验通过: ${ACTUAL}"
```

**失败处理**：sha256 不匹配 → 拉前一份（`mc ls` 列表中的次新），重做 §4.1-4.2。

### 4.3 启动新 PG 实例（备机房）

**docker compose 路径**（开发 / 中型环境）：

```bash
# 在备机房的 omcgo 目录
cd /opt/omcgo
docker compose -f deployments/docker/docker-compose.yml up -d postgres

# 等就绪
until docker compose exec postgres pg_isready -U omcgo; do sleep 1; done
```

**k8s 路径**（生产）：

```bash
kubectl apply -n omc-dr -f deployments/k8s/postgres-statefulset.yaml
kubectl rollout status -n omc-dr statefulset/postgres --timeout=300s
kubectl exec -n omc-dr postgres-0 -- pg_isready -U omcgo
```

### 4.4 pg_restore 到新实例

```bash
# 创建空库
psql --host="${PGHOST}" --port="${PGPORT}" --username=postgres --dbname=postgres <<SQL
DROP DATABASE IF EXISTS omcgo;
CREATE DATABASE omcgo OWNER omcgo ENCODING 'UTF8';
SQL

# 全量恢复（--jobs=4 加速）
pg_restore \
    --host="${PGHOST}" --port="${PGPORT}" \
    --username="${PGUSER}" --dbname=omcgo \
    --no-owner --no-privileges --exit-on-error \
    --jobs=4 --verbose \
    "${DUMP}"
```

**失败处理**：`pg_restore` 失败 → 回退到上一份全量；如连续 2 份都失败 → 启动取证 + P0 升级。

**RTO 计时点 2**：记录 pg_restore 完成时间 `T1`。

### 4.5 切换 omcgo-app config 指向新 PG

修改 `omcgo/cmd/{app,acs,worker}/etc/config.prod.yaml`：

```yaml
postgres:
  host: pg-dr.example.com   # ← 改为备机房 PG
  port: 5432
  user: omcgo
  password: "${PGPASSWORD}"
  database: omcgo
```

**也可走环境变量覆盖**（不改文件，立刻生效）：

```bash
export OMCGO_POSTGRES_HOST="pg-dr.example.com"
export OMCGO_POSTGRES_PASSWORD="***"
```

重启三件套：

```bash
# docker compose
docker compose -f deployments/docker/docker-compose.yml up -d omcgo-app omcgo-acs omcgo-worker

# k8s
kubectl rollout restart -n omc deployment omcgo-app omcgo-acs omcgo-worker
kubectl rollout status   -n omc deployment omcgo-app --timeout=120s
```

### 4.6 验证 migration 版本 + 6 项校验

参考 W1.8 演练脚本 `db_restore_drill.sh` 的 step 3 校验逻辑：

```bash
PSQL=(psql --host="${PGHOST}" --port="${PGPORT}" --username="${PGUSER}" --dbname=omcgo --no-psqlrc --tuples-only --no-align)

# 校验 1：迁移版本（goose）
"${PSQL[@]}" -c "SELECT max(version_id) FROM goose_db_version;"
# 期望：与 ls migrations/ | sort | tail -1 截取的编号一致（当前 000037）

# 校验 2：表数量 > 0
"${PSQL[@]}" -c "SELECT count(*) FROM information_schema.tables WHERE table_schema='public';"

# 校验 3：devices 行数 > 0
"${PSQL[@]}" -c "SELECT count(*) FROM devices;"

# 校验 4：users 行数 > 0
"${PSQL[@]}" -c "SELECT count(*) FROM users;"

# 校验 5：alarm_libraries 行数 > 0
"${PSQL[@]}" -c "SELECT count(*) FROM alarm_libraries;"

# 校验 6：TimescaleDB 扩展存活（如生产启用）
"${PSQL[@]}" -c "SELECT extname, extversion FROM pg_extension WHERE extname='timescaledb';"
```

E2E 冒烟（确认 app 真正能读写）：

```bash
bash omcgo/scripts/e2e_verify.sh https://omc.example.com
# 期望：Results: PASS=226 FAIL=0（数字以脚本实际为准）
```

**RTO 计时点 3**：记录业务恢复时间 `T2`，**RTO = T2 - 故障开始时间**。

### 4.7 RTO 计时模板（事后填写，归档到 §6）

| 字段 | 值 | 备注 |
|------|----|------|
| 故障开始 | YYYY-MM-DDTHH:MM:SSZ | 监控告警 / 上报时间 |
| 决策启动 DR | YYYY-MM-DDTHH:MM:SSZ | 值班 lead 拉群时间 |
| T0（拉取开始） | | |
| T1（restore 完成） | | |
| T2（业务恢复，E2E 通过） | | |
| **RTO 总时长** | T2 - 故障开始 | 目标 ≤ 1h |
| dump 文件时间戳 | YYYYMMDD-HHMMSS | 文件名内嵌 |
| **RPO（数据丢失窗口）** | 业务最后写入时间 - dump 时间戳 | 目标 ≤ 15min |
| 数据量 | XX GB | dump 解压后估算 |
| 校验结果 | PASS / FAIL | 6 项 + E2E |

---

## 5. 演练步骤（每月 1 次模拟整机房故障）

### 5.1 演练目的与节奏

| 维度 | 设定 |
|------|------|
| 频率 | **每月 1 次**，第一个周日 04:00（业务低谷） |
| 范围 | 完整模拟场景 B（整机房故障 → 切备机房 → E2E 通过） |
| 责任人 | 运维 Owner + 一位 oncall 配合（双人在场） |
| 不影响生产 | 演练在 staging 或备机房隔离环境，**绝不**指向生产 PG host |

### 5.2 演练步骤（缩减版 — 不切真生产，只走流程）

```bash
# 步骤 1：选定一份近期生产备份（生产备份桶 → staging 拉取）
mc cp "omcoffsite-prod/${BUCKET}/backups/prod/<latest>.dump" ./dr-drill/
mc cp "omcoffsite-prod/${BUCKET}/backups/prod/<latest>.meta" ./dr-drill/

# 步骤 2：sha256 校验
EXPECTED=$(grep ^sha256= ./dr-drill/<latest>.meta | cut -d= -f2)
ACTUAL=$(sha256sum ./dr-drill/<latest>.dump | awk '{print $1}')
[[ "$EXPECTED" == "$ACTUAL" ]] && echo OK || { echo FAIL; exit 1; }

# 步骤 3：复用演练脚本（创建临时库 → restore → 校验 → 销毁）
PGPASSWORD=*** bash omcgo/scripts/db_restore_drill.sh ./dr-drill/<latest>.dump

# 步骤 4：把演练脚本输出的 RTO 数字记录到本文档 §6
```

### 5.3 演练失败处置

| 失败类型 | 处置 |
|---------|------|
| 单次失败 | 当周内复盘，写 postmortem 到 `docs/postmortem/YYYYMMDD-dr-drill-<reason>.md` |
| 连续 2 次失败 | 升级 P1，risk-register 挂条目；阻塞下一个 release |
| sha256 不匹配 | 立刻倒查异地上传链路（网络 / 客户端 / bucket lifecycle）；同时排查源 dump 是否已损坏 |
| RTO 超 1h | 分析瓶颈（pg_restore --jobs 调优 / 网络 / 磁盘 IO）；写专项优化任务 |

---

## 6. 实测占位（待 staging 演练后回填）

> ⚠️ **当前状态**：T-0067 仅落地 framework（脚本扩展 + Runbook）。真演练 RTO/RPO 数字需由维护方在 staging 跑一次完整流程后回填本节，与 T-0008 / T-0059 同模式（framework PASS）。

### 6.1 RTO 实测记录表

| 演练日期 | 数据量 | 网络带宽 | T0→T1（restore） | T0→T2（含 E2E） | RTO 总 | 是否达标（≤1h） | 责任人 | 备注 |
|---------|--------|---------|----------------|----------------|--------|----------------|--------|------|
| TBD | TBD | TBD | TBD | TBD | TBD | TBD | TBD | 首次 staging 演练 |

### 6.2 RPO 实测记录表

| 演练日期 | 备份频率 | 异地上传滞后 | 业务最后写 - dump 时间 | RPO | 是否达标（≤15min） | 备注 |
|---------|---------|------------|---------------------|-----|------------------|------|
| TBD | TBD | TBD | TBD | TBD | TBD | 增量备份方案落地后才能 ≤ 15min |

### 6.3 已识别风险

| 风险 | 影响 | 缓解 |
|------|------|------|
| 当前只有每日全量 → RPO ≈ 24h（不达标） | RPO 远超 15min 目标 | 引入 `pgbackrest` / `wal-g`（下一个 wave 任务） |
| 异地 S3 上传失败仅 warn，不阻塞 | 可能导致连续多日异地空缺而无人发觉 | Prometheus 告警：`omc_backup_offsite_total{status="failure"}` 24h 超 1 次即告警 |
| 加密尚未落地 | dump 含设备序列号 / 用户哈希，泄露后风险高 | 投产前必须用 `age` / `gpg` 加密；密钥放 Vault |
| 演练频率仅每月 | 故障切换可能因生疏而超时 | 上线初期 2 周一次；稳定后降到每月 |
| 备机房 PG 实例非常驻（冷备） | 启动 + 恢复合计耗时 ≥ 30 min | 长期方案：异地 streaming replica（热备） |

---

## 7. 配套监控告警建议（待落地）

> 这里列出 metric 名称与告警逻辑，**不**在本任务中实现 Go 代码 / PromQL；监控团队据此补 PromQL rule。

| Metric / 信号 | 告警条件 | 严重级 | 说明 |
|--------------|---------|--------|------|
| `omc_backup_offsite_total{status="failure"}` | 24h 内 ≥ 1 次 | P1 | 异地上传失败 — 立刻人工跟进 |
| `omc_backup_offsite_total{status="success"}` | 25h 内未递增 | P1 | 异地备份链路停摆 |
| 异地 S3 bucket 文件最新时间 | now - latest > 30 min | P0 | RPO 即将不达标 |
| `omc_backup_duration_seconds` | > 600s | P2 | 全量备份时长异常 |
| pg_restore 演练（cron 触发） | 失败 ≥ 1 次 / 周 | P1 | 见 §5.3 |

实现路径：mtail / promtail 解析 `db_backup.sh` stdout JSON + LOG 行；告警规则放 `deployments/monitoring/prometheus/rules.yml`（本任务不动该目录）。

---

## 8. 与其他文档的关系

| 文档 | 关系 |
|------|------|
| `docs/runbook/db-backup-restore.md` | W1.8 base — 本地备份策略 + 单机演练（本文档的"前置条件"） |
| `docs/runbook/nats-failover.md` | W3.E.3 NATS 故障演练 — 同模式 framework，可参考其 §6 实测占位写法 |
| `docs/project/risk-register.md` | RPO ≤ 15min 当前不达标 → 应在 risk-register 挂条目，附本文档 §6.3 风险表 |
| `docs/project/release-gate.md` | 投产前必须验证：异地备份连续 7 天成功 + 至少 1 次 staging 完整演练通过 |

---

## 9. 变更记录

| 日期 | 变更 | 责任人 |
|------|------|--------|
| 2026-04-28 | 初版 — T-0067 落地 framework（异地脚本扩展 + DR Runbook 全流程） | T-0067 worktree agent (W3.H.3) |
| TBD | 首次 staging 真演练数据回填到 §6 | 待运维 Owner |
| TBD | 引入 wal-g/pgbackrest，RPO 真正 ≤ 15min | 待 Wave 4+ 任务 |
