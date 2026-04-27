# OMC 数据库备份与恢复演练 Runbook

> **任务编号**：T-0042（Wave 1 W1.8 数据库定时备份 + 恢复演练）
> **首次落地**：2026-04-27
> **维护方**：运维与可观测性专家 Owner（暂列：xieguiya）
> **关联文件**：
> - 备份脚本：`omcgo/scripts/db_backup.sh`
> - 演练脚本：`omcgo/scripts/db_restore_drill.sh`
> - cron 配置：`deployments/cron/db-backup.cron`

---

## 1. 备份策略

| 维度 | 当前策略 | 说明 |
|------|---------|------|
| 频率 | 每日 03:00 全量 | cron 配置在 `deployments/cron/db-backup.cron` |
| 格式 | `pg_dump --format=custom --compress=6` | 单文件，pg_restore 友好，能挑表恢复 |
| 保留期 | 30 天滚动 | 由 `--retention-days` 控制；通过 `find -mtime` 清理 |
| 存储位置 | 仓库根 `backups/<env>/`（`.gitignore` 已忽略） | 生产应额外推送到 MinIO `config-backup` 桶或对象存储 |
| 命名 | `omc-YYYYMMDD-HHMMSS.dump` + 同名 `.meta`（含 size / sha256 / 时长） | 便于检索与校验 |
| 加密 | **TBD（生产前必须启用）** | 当前 dump 明文。计划：`pg_dump | age -r <key>` 或先 dump 再 `gpg --encrypt` |
| 凭据 | `PGPASSWORD` env 或 `~/.pgpass` | 严禁在脚本/cron 行内写明文 |
| 失败重试 | 3 次指数退避（2 → 4 → 8s） | 失败仍报错退出，触发 cron MAILTO |

### 调用示例

```bash
# 本地 dev
PGPASSWORD=omcgo123 bash omcgo/scripts/db_backup.sh --env dev

# 生产（凭据走 .pgpass）
bash omcgo/scripts/db_backup.sh --env prod \
     --db-host pg-primary.prod.internal \
     --db-port 5432 \
     --retention-days 60
```

成功输出（stdout，cron 友好的 JSON）：

```json
{
  "status": "ok",
  "env": "dev",
  "file": ".../backups/dev/omc-20260427-194749.dump",
  "size_bytes": 3480340,
  "size_human": "3.3M",
  "duration_seconds": 1,
  "sha256": "ec47719d56c0233610e715529d5decaaa1ee194ca90f9050150e3b5982d74484",
  "retention_days": 30,
  "old_files_removed": 0
}
```

---

## 2. RTO 目标 vs 实测

### 2.1 目标

| 规模 | RTO 目标 | 依据 |
|------|---------|------|
| 10 万基站起步阶段 | **≤ 30 分钟** | 项目规模假设（CLAUDE.md §1）；P0 短板里要求"恢复演练 RTO ≤ 30min" |
| 100 万基站扩展阶段 | ≤ 60 分钟 | 数据量约 10x，TimescaleDB chunk 处理是主要瓶颈，需结合 PITR / 增量备份重新评估 |

### 2.2 实测（2026-04-27 19:47-19:48）

- **环境**：本机 Homebrew Postgres 16.13 / 单机 / Apple Silicon
- **数据量**：138 张表，44 MB（其中 30,030 条 device 行、15 条 alarm_libraries、1 个 user）
- **备份耗时**：1 秒，dump 文件 3.3 MB
- **恢复耗时（RTO）**：1 秒（CREATE DATABASE + pg_restore）
- **演练总 wall-clock**：1.482 秒（含 baseline 查询 + 临时库创建 + restore + 校验 + 销毁）
- **校验通过项**：tables_exist / devices_nonempty / users_nonempty / alarm_libraries_nonempty / devices_consistency（diff=0）

> ⚠️ **重要限制**：本次实测数据量极小（44 MB / 30K 行），RTO=1s **不代表生产**。
> 生产环境 10 万基站规模的真实 RTO，需要在 staging 环境用接近生产数据量的快照重测一次。
> TimescaleDB hypertable 的 chunk 解压在大数据量下会显著拉长恢复时间，本机演练未覆盖这一路径（本地未启用 TimescaleDB 扩展）。

---

## 3. 灾难恢复 step-by-step

> 以下流程假设：(1) 备份文件可读；(2) 目标 PG 实例可连接；(3) 应用进程已停止（避免双写）。

### 3.1 完整恢复流程（ALL HANDS）

```bash
# 0) 通告：发出 P0 故障通知，业务团队停写
#    （略 — 走值班手册）

# 1) 停止 OMC 应用三件套，避免新写
#    生产：systemctl stop omcgo-app omcgo-acs omcgo-worker
#    本地：bash run/scripts/stop-all.sh

# 2) 选定备份文件
DUMP_FILE=$(ls -t ${REPO_ROOT}/backups/prod/omc-*.dump | head -1)
echo "will restore from: ${DUMP_FILE}"
sha256sum "${DUMP_FILE}"   # 与 .meta 文件中的 sha256 比对

# 3) 备份当前损坏库（万一需要 forensic 分析）
pg_dump --host=<pg-host> --port=5432 --username=omcgo --dbname=omcgo \
        --format=custom --file=/tmp/omcgo-corrupted-$(date +%s).dump || true

# 4) DROP & RECREATE 目标库
psql --host=<pg-host> --port=5432 --username=postgres --dbname=postgres <<SQL
SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='omcgo';
DROP DATABASE IF EXISTS omcgo;
CREATE DATABASE omcgo OWNER omcgo ENCODING 'UTF8';
SQL

# 5) pg_restore 全量恢复
pg_restore --host=<pg-host> --port=5432 --username=omcgo --dbname=omcgo \
           --no-owner --no-privileges --exit-on-error \
           --jobs=4 \
           "${DUMP_FILE}"

# 6) 校验（直接复用演练脚本的 verify 路径，或手工）
psql --host=<pg-host> --port=5432 --username=omcgo --dbname=omcgo <<SQL
SELECT count(*) FROM devices;
SELECT count(*) FROM users;
SELECT count(*) FROM alarm_libraries;
SELECT extname FROM pg_extension WHERE extname='timescaledb';
SQL

# 7) 重启 OMC 应用三件套
#    生产：systemctl start omcgo-app omcgo-acs omcgo-worker
#    本地：bash run/scripts/start-all.sh

# 8) E2E 冒烟
bash omcgo/scripts/e2e_verify.sh http://localhost:8081

# 9) 通告：恢复完成，开放业务写入
```

### 3.2 单表恢复（精确恢复某张被误删的表）

```bash
pg_restore --host=<pg-host> --port=5432 --username=omcgo --dbname=omcgo \
           --table=<table_name> \
           --data-only \
           --disable-triggers \
           "${DUMP_FILE}"
```

### 3.3 一键恢复演练（不影响生产，跑临时库）

```bash
PGPASSWORD=*** bash omcgo/scripts/db_restore_drill.sh \
    ./backups/prod/omc-20260427-030000.dump
```

---

## 4. 已知限制与风险

| 风险 | 说明 | 缓解 |
|------|------|------|
| **单机 dump 在大数据量下的瓶颈** | `pg_dump` 是逻辑备份，10 GB+ 库恢复可能超过 30min RTO 目标 | 100 万规模阶段切到 PITR（base backup + WAL archive）+ 物理快照 |
| **TimescaleDB chunk 处理** | hypertable 大量 chunk 在 restore 时一次性解压，可能拖慢 RTO；本地未实测 | staging 必须用启用 TimescaleDB 的环境复测 |
| **加密 TBD** | 当前 dump 明文，含设备序列号 / 用户哈希等敏感字段 | 投产前用 `age` 或 `gpg` 加密；密钥放 KMS / Vault |
| **本地 RTO 数据集小（44 MB）** | 实测 RTO=1s 不能外推到生产 | 见 §2.2 限制说明，staging 复测 |
| **备份只在仓库目录** | 单机磁盘损坏会同时丢失原库与备份 | cron 增加 `mc cp` 推送到 MinIO `config-backup` 桶（待补任务 T-00xx） |
| **非全量备份（增量/WAL）尚未引入** | 当前只有每日全量；丢数最坏可达 24h | 投产前补 `pgbackrest` 或 `wal-g`，做到 RPO ≤ 5min |
| **演练只校验 4 张表 + 行数** | 校验深度浅 | 后续把 `e2e_verify.sh` 的关键 GET 端点接进演练脚本，确认数据可被 app 正确读出 |

---

## 5. 下次演练计划

| 维度 | 计划 |
|------|------|
| 下次自动演练 | **2026-05-03（周日）04:00**（首次按 cron 自动跑） |
| 下次人工 staging 演练 | **2026-05-15 之前**，用接近生产数据量（≥ 5 GB / 启用 TimescaleDB）的快照重新测算 RTO |
| 触发重测的场景 | 数据量 +50%；新增 hypertable；新增大型扩展；恢复脚本逻辑改动 |
| 责任人 | 运维 Owner（首次：xieguiya），结果记录回本 runbook §2.2 表格 |
| 失败处置 | 单次失败 → 当周内复盘；连续两次失败 → 升级 P1 + risk-register 单挂条目 |

---

## 6. 变更记录

| 日期 | 变更 | 责任人 |
|------|------|--------|
| 2026-04-27 | 初版上线（脚本 + cron + 实测 RTO=1s/44MB） | T-0042 worktree agent |
