# OMC KPI/时序库物理分离（起两个 PG）落地方案【最终】

> 状态：最终方案（基于真实 `file:line` 证据 + 评审复核）  日期：2026-06-12  作者：OMC 后端架构
>
> **结论先行：能起两个 PG，但「改 DSN 就行」是假象——它属中大型重构，且在分库前必须先顺手修掉一个既有的 fresh-DB 迁移 latent bug。** 关键利好是基础设施层已是双池架构、全库外键零跨界、无连续聚合；关键代价是 7 处单 SQL 跨库 JOIN、PM 入库单事务原子性、若干超表 repo 错池读写、70 个迁移文件需逐一分诊。推荐**方案 A（两个物理 TimescaleDB 实例）**，分阶段走、以方案 D 作可选过渡。

---

## 1. 背景与诉求

用户诉求：把 KPI/时序存储的数据库与其他业务数据分开，避免数据污染、保护重要数据，问能否起两个 PG。

本质是把当前「一个 TimescaleDB 实例托管时序超表 + 主库业务表」拆成「**主库**（devices/RBAC/config 等重要数据）+ **时序库**（PM/告警历史/MR/trace 超表）」两个物理实例，让时序写洪峰与主库事务物理隔离。该诉求与项目现实高度契合：PM 入库压测实测 PG 写已成瓶颈（单 chunk 65M 行、PG 打满多核于索引维护，见 `docs/qa-report/pm-ingest-stress-report-20260612.md`），且需满足 10万→100万基站扩展。

---

## 2. 现状（已核实）

### 2.1 基础设施已是双池架构（关键利好）

- `internal/core/components/infra.go:39-40` Infra 持 `PgPool`+`TsPool` 两字段；`:103-114` `ConnectPostgres`、`:118-129` `ConnectTimescale` 两条独立 Connect*，各自注册 health（`:110-112`/`:125-127`）与 graceful-shutdown（优先级 4）。
- app 与 worker 都连两池；**ACS 只连 PgPool**（`cmd/acs/bootstrap.go` 无 `ConnectTimescale`，acs config 也无 `tsdb` 段——已核 `cmd/acs/etc/config.dev.yaml` 中 `grep tsdb` 为 0）。
- 池层 `internal/core/components/postgres.go:16-17` 按 per-config DSN 建池，分库是纯配置改动，**池层零代码改**。
- `/readyz` 与优雅关机对两池都已注册，分库即用，无需补生命周期管线。

> **口径校正**：主库「只连 PgPool / 分库纯配置改动」仅就**池层**而言成立；**不等于主库可无脑降级为纯 pg16**。主库降级的前提见 §2.3 与 §7-P2。

### 2.2 物理超表 7 张（时序边界）

`migrations/seed/000001_init_seed.sql:50-62` `_timescaledb_catalog.hypertable`（id 5/6/8/9/11/13/15）：
`alarms_history`、`mr_records`、`trace_messages`、`pm_metrics`、`pm_metrics_hourly`、`pm_group_metrics_hourly`、`pm_adhoc_aggregation_results`。
另有 6 张普通 rollup 表（`pm_metrics_daily/weekly/monthly` + group 系列）属时序域，由 aggregator `INSERT...SELECT` 维护，须随 `pm_metrics` 同进时序库。

此外，`alarm_efficiency_metrics`**物化视图**（建在 `alarms_history` 上，定义在 seed 序列 `000009_alarm_efficiency_metrics.sql` 与 `000042_alarm_efficiency_mttr_non_negative.sql`，**不在 schema baseline**）及其在 `alarms_history` 上的索引，也属时序域、须随 `alarms_history` 同迁。

### 2.3 三大结构性事实

- **FK 零阻断**：全库 84 个外键无一以 7 张超表为源或为目标（`migrations/000001_init_schema.sql` `ADD CONSTRAINT` 风格，已 grep 复核）——拆分最大利好。
  - 澄清（防概念混淆）：这 7 张超表只有**复合 PRIMARY KEY**（如 `pm_metrics_pkey(id, time)`），**没有 FOREIGN KEY**。PK 复合键是表内约束、不跨库，与「84 FK 无跨界」是两件事，勿混读为「PK 也是跨库约束」。
- **无 CAGG**：`_timescaledb_catalog.continuous_agg` 目录空；rollup 全是 Go 驱动 `INSERT...SELECT`；`pm_metrics_hourly` 等是普通超表不是连续聚合；唯一物化视图 `alarm_efficiency_metrics` 仅基于 `alarms_history`。
- **超表注册靠 catalog dump（非 `create_hypertable`）**：`migrations/000001_init_schema.sql:31` `CREATE EXTENSION timescaledb`，但 7 张超表在 schema baseline 里是普通 `CREATE TABLE`；超表注册全靠 seed 的 `_timescaledb_catalog` pg_restore（`migrations/seed/000001_init_seed.sql:23` pre_restore / `:5867` post_restore）。**这一点不是「脆弱设计」而是一个已存在的 latent bug——见 §6 阻断 B0。**

### 2.4 池路由与物理超表不符（定时炸弹）

今天两 DSN 同实例所以无感，分库瞬间断裂：

- **alarm**：`internal/alarm/pg_store.go:25/29-30` 持 `tsPool` 但**从不使用**（已核 `:35` 起所有读写全走 `s.pool`，`tsPool` 是死参），`alarms_history` 全部读写走 PgPool。
- **trace**：`trace.NewPgRepository(w.PgPool)`（`cmd/worker/main.go:419`）写 `trace_messages` 超表走 PgPool。
- **pm copy**：`cmd/worker/main.go:203` copy ingestor 走 PgPool 写 `pm_metrics` 超表。
- **adhoc**：写 PgPool / 读 TsPool 分裂。
- **kpi/counter**：`internal/pm/kpi/pg_repository.go:267`、`internal/pm/counter/pg_repository.go:334` 在 TsPool 上反查主库 `devices`（错池）；`kpi_definitions`（配置字典）也被 TsPool 误写。
- **范本**：`internal/mr/pg_store.go` 正确——`mr_records` 走 `tsPool`（`:205`）、`mr_files` 走 `pool`，可作改造参照。

---

## 3. 可行性结论

**能起两个 PG，但不是「改 DSN 就行」，属中大型重构。**

| 维度 | 结论 |
|------|------|
| 利好 | 双池架构已就位、FK 零跨界、无 CAGG |
| 硬阻断 | ① fresh-DB 迁移 `000045` 处崩溃 / `pm_metrics` chunk 实为 1d（既有 bug，须先修）；② 7 处单 SQL 跨库 JOIN/子查询；③ PM 入库单事务跨库原子性 + 幂等锚点；④ 5-6 个超表 repo 错池读写；⑤ 70 个迁移文件需分诊 + alarm matview 单列；⑥ 8 个配置文件 + ACS 新增 `tsdb` 段 |
| 契合现实 | 核心痛点是 PG 写已是瓶颈，物理分实例正是对症解，且满足 10万→100万扩展 |

---

## 4. 方案对比

| 维度 | A 两物理实例 | B 同实例两 database | C 同库不同 schema | D 单库仅加固 |
|------|-------------|---------------------|-------------------|--------------|
| 防污染/保护数据 | 最强（物理隔离）| 中（逻辑+权限）| 中弱（权限）| 弱（IO+权限）|
| 解决 PG 写瓶颈 | 是 | 否（同进程争 WAL/IO/CPU）| 否 | 部分（tablespace）|
| 跨库 JOIN 改造 | 必须（7 处）| 必须（PG 跨 db 不支持 JOIN）| 不需要（跨 schema 合法）| 不需要 |
| copy 拆事务 | 必须 | 必须 | 不需要 | 不需要 |
| 迁移拆分 | 必须（70 文件分诊）| 必须（同 A）| 轻（SET SCHEMA）| 不需要 |
| 扩展性 10万→100万 | 最佳（独立扩容）| 差（绑死同进程）| 最差 | 差 |
| 运维成本 | 高（双实例/备份/监控）| 中 | 低 | 低 |
| 工期 | 大（2-4 周）| 大（性价比最差）| 小（2-3 天）| 小（3-5 天）|
| 与「起两个 PG」诉求 | 契合 | 契合（弱）| 方向相反 | 不契合（过渡）|

**判断要点**：B/C 拿不到物理隔离/写瓶颈收益，B 却付与 A 几乎相同的改造代价（PG 跨 database 同样不支持跨库 JOIN）；C 与「起两个 PG」诉求方向相反。D 仅 IO（tablespace）+ 权限 + 备份维度部分隔离，是过渡态非终态。

---

## 5. 推荐方案

**方案 A（两个物理 TimescaleDB 实例），分阶段走，可选用 D 做过渡。**

1. 已是双池架构，A 是顺势把池真正分到两实例，基础设施成本最低。
2. 只有 A 能让时序写洪峰不抢主库 WAL/IO/CPU——B/C 拿不到这个收益却付同样改造代价，D 只能部分缓解。
3. 10万→100万扩展要求时序库能独立扩容，唯 A 满足；未来时序库甚至可换专用 TSDB。

**落地节奏**：

- **第一阶段（不分库也该做、零风险）**：P0 池路由统一 + **B0 修 fresh-DB 迁移 bug**。改完后两 DSN 仍同实例、行为不变，但消除了「超表错池读写」定时炸弹，并修复了既有的 `000045` 崩溃。
- **第二阶段（真分库）**：P1 拆跨库 JOIN → P2 切分迁移 baseline → P3 拆 copy 事务 → P4 起第二实例并接线 → P5 数据搬迁灰度切换。
- **可选过渡**：工期紧时先上方案 D（角色 + tablespace + 备份加固）争取时间，但明确它是中转不是终点。

---

## 6. 硬阻断清单与处理

> 跨库 JOIN 总表（分库后 PG 不支持跨实例 JOIN，必须先拆）：

| # | 位置 | 超表 | 主库表 | 现池 |
|---|------|------|--------|------|
| 1 | `internal/pm/aggregator/query.go:557-559` | pm_metrics/rollup | devices | TsPool |
| 1b| `internal/pm/aggregator/query.go:801-806` | pm_metrics/rollup | devices, device_params | TsPool |
| 2 | `internal/pm/aggregator/device_group.go:74-77` | pm_metrics→group rollup | devices, device_group_members | TsPool |
| 3 | `internal/pm/adhoc/handler.go:520-522` | pm_adhoc_aggregation_results | products, device_groups | TsPool |
| 4 | `internal/pm/device_query_service.go:129` / `internal/pm/export/query.go:142-147` | pm_metrics/rollup | devices（IN 子查询）| TsPool |
| 5 | `internal/alarm/pg_store.go:218-220`（及 :66/:407/:553）| alarms_history | devices, alarm_definitions | PgPool（错）|

**阻断 B0（新增·必须置顶·既有 latent bug）—— fresh-DB 迁移在 `000045` 处崩溃 / `pm_metrics` chunk 实为 1d**

- 证据：`migrations/000045_pm_metrics_chunk_interval_4h.sql:17` 是 `SELECT set_chunk_time_interval('pm_metrics', INTERVAL '4 hours')`，该函数要求 `pm_metrics` **已是超表**；但 compose `depends_on` 决定 `migrate-schema` 整条跑在 `migrate-seed` 之前，而超表注册全靠 seed 的 catalog dump——因此 `000045` 运行时 `pm_metrics` 还是普通表，`set_chunk_time_interval` 报 `ERROR: "pm_metrics" is not a hypertable`。
- 更要命：seed `_timescaledb_catalog.dimension` 把 `pm_metrics`（dimension id 7 / hypertable id 9）的 `interval_length` 写成 `86400000000`（= 1 天，已核 `migrations/seed/000001_init_seed.sql:120`）。**所以即便绕过崩溃，fresh DB 上 `pm_metrics` 实际是 1d chunk，不是 4h；4h 只在「seed 先于 000045 历史性跑过」的增量升级路径才生效。**
- 影响：本方案 P2 验收要求「两空库各自 `goose up` 通过」，按现状 tsdb baseline 的 fresh `goose up` 会在 `000045` 直接失败；且若不修，新时序库的 `pm_metrics` 退回 1d chunk，触发 PM 入库已知瓶颈。
- **处理**：把超表创建从「seed catalog dump 还原」改成**显式** `create_hypertable('pm_metrics', by_range('time', INTERVAL '4 hours'))`（或 `create_hypertable(..., chunk_time_interval => INTERVAL '4 hours')`）+ `add_compression_policy` + `add_retention_policy`，对 7 张超表统一处理；废弃 catalog dump 还原段与独立的 `000045`（其逻辑并入建表）。**这同时摆脱了 catalog dump 的 hypertable id 5-16 硬编码冲突风险。**
- **CI 门**：新增 `fresh down -v && up` 必须全绿（现有 `scripts/check-migrations.sh` 只校验文件名/号段、不跑真迁移，故此 bug 未被覆盖）。
- 说明：这是分库前就已存在、且分库 P2 必须顺手修掉的硬阻断，建议在第一阶段随 P0 一并落地。

**阻断 B1（跨库 JOIN #1 / PM 聚合）**：`query.go:557-559`（product 维度 `FROM %s m JOIN devices d`）与 `:801-806`（band 维度多 JOIN `cell_band`/`device_params`），`a.db=TsPool`。处理：先在主库取 `(oui,sn)→product_id/band` 映射，聚合 SQL 去 JOIN、用 `WHERE IN` 下推、Go 内回填。

**阻断 B2（跨库 JOIN #2 / 最棘手 / 设备组 rollup）**：`device_group.go:74-77` 单条 `INSERT INTO pm_group_metrics_* SELECT FROM pm_metrics JOIN devices JOIN device_group_members`（hourly/daily/weekly/monthly 同构），跨实例无法单 SQL 完成。处理：在时序库冗余轻量 `device_dim(oui, sn, device_id, product_id, group_ids[], technology, band)`，由业务库定时/事件同步，rollup 改 JOIN 本库 `device_dim`。

**阻断 B3（跨库 JOIN #3 / adhoc 结果查询）**：`handler.go:520-522` `pm_adhoc_aggregation_results LEFT JOIN products/device_groups`，`h.pool=TsPool`。处理：只取 `product_id/object_ldn` 原始列，`product_name/group_name` 由 Go 端小字典缓存回填。

**阻断 B4（跨库子查询 #4）**：`device_query_service.go:129` 与 `export/query.go:142-147`，`pm_metrics WHERE (device_oui,device_sn) IN (SELECT FROM devices)`。处理：先在主库查 `devices` 得键集，作 IN 参数下推（IN 列表过大须分批）。

**阻断 B5（跨库 JOIN #5 / 告警历史）**：`alarm/pg_store.go:218-220`（ListHistory）/`:66`/`:407`/`:553`，`alarms_history`（超表）`LEFT JOIN devices/alarm_definitions` 且整条经 PgPool（`s.tsPool` 是死参）。处理：把 `alarms_history` 读写切到 `tsPool` + 归档时把 `device_name/technology` 冗余进 `alarms_history`、本地化名用 Go 字典回填，彻底去 JOIN；`dashboard/efficiency.go`、`dashboard/heatmap.go` 读 `alarms_history` 一并切 `tsPool`。

**阻断 B6（REFRESH matview 跨库 / 独立路径）**：`dashboard/efficiency.go:150` 的 `REFRESH MATERIALIZED VIEW CONCURRENTLY alarm_efficiency_metrics`，分库后该 matview 随 `alarms_history` 在时序库一侧，REFRESH 这条 **DDL 必须改打 `tsPool`**（否则刷新命中主库空 matview）。`CONCURRENTLY` 要求 matview 有唯一索引——须确认其唯一索引随 matview 一并迁到 tsdb-seed。这是独立于「读 `alarms_history`」的写 DDL 路径，单列追踪。

**阻断 B7（PM 入库单事务原子性 + 幂等锚点 / 头号热路径）**：`internal/pm/metrics/copy_ingest.go:162-184` 在单 tx 里**先**写 `pm_files` 标记（主库，`:162-165` `INSERT ... ON CONFLICT(device_sn,file_name) DO NOTHING`，`:171` `RowsAffected()==0→return false` 跳过）、**后** `CopyFrom pm_metrics`（超表，`:177`）。**这个写序是删了 `uq_pm_metrics_natural`（迁移 000042）后唯一的跨文件去重锚点**（已核 `copy_ingest.go:16-19/116-128` 注释明确说明 `pm_files` 标记是入库幂等的唯一锚点、`pm_metrics` 无唯一约束）。

- **正确处理（写序绝不可反）**：分库后拆成两实例两步，序必须保持「**先在主库写/抢占 `pm_files` 标记拿到去重判定 → 仅当标记为本次新建（`RowsAffected()==1`）才写 `pm_metrics`（时序库）**」。`pm_metrics` 无任何唯一约束，若反过来「先写 `pm_metrics` 再写标记」，崩在两步之间时 NATS 重投会把同一文件的 metric 行**再 COPY 一遍**，`pm_files` 的 `DO NOTHING` 只去重标记行、拦不住已翻倍的 `pm_metrics` 行 → 所有 KPI `SUM` **双计数**。
- 候选实现：(a) 主库 `pm_files` 标记作前置闸门（最小改动、贴合现状）；(b) 引入 outbox 表在主库；二选一并明确崩在两步间的重投语义。
- **验收必须断言行数不翻倍**（`DLQ=0` 不足以证明无双计数）：注入「标记写成功、metrics 未写完」的崩溃，重投后断言 `pm_metrics` 目标文件行数不变。

**阻断 B8（aggregator 跨库 INSERT...SELECT）**：`pm_metrics` 与 6 张 rollup 表（`pm_metrics_daily/weekly/monthly` + group 系列）必须**整体同进时序库**，否则 `INSERT INTO pm_metrics_hourly SELECT FROM pm_metrics` 跨实例断裂。`kpi_definitions`（配置字典、被 TsPool 误写）应**留主库**、写从 TsPool 切回 PgPool。

---

## 7. 分阶段落地

> 每步给出 goal / 关键步骤 / files_touched / verify。

### P0 池路由统一（不分库，零行为变更，零风险预备）+ B0 修迁移 bug

**goal**：把所有超表读写统一到 TsPool、所有主库表反查统一到 PgPool，消除「超表错池读写」定时炸弹；同时修掉 `000045` fresh-DB 崩溃。两 DSN 仍同实例，外部行为不变，可独立提交上线。

- alarm：`internal/alarm/pg_store.go` 中 `alarms_history` 的 INSERT/SELECT 从 `s.pool` 改 `s.tsPool`（`tsPool` 已注入但闲置）；`history_retention.go` 的 add/remove retention policy 用 `tsPool`。
- trace：`internal/trace/pg_repository.go` 改注入 TsPool（接线点 `cmd/worker/main.go:419`、`cmd/app/provider/modules.go`、`cmd/acs/main.go`）；**ACS 需先补 `ConnectTimescale`（`cmd/acs/bootstrap.go` 当前无）+ 新增 `tsdb` config 段**（acs config 当前无 tsdb——这是新增配置键，不是改 DSN；`infra.ConnectTimescale` 自带 health/GS 注册，故无额外生命周期活）。
- pm copy：`cmd/worker/main.go:203` `SetCopyIngestor` 从 `w.PgPool` 改 `w.TsPool`（仅把 `pm_metrics` 写对池；`pm_files` 主库表的跨实例拆事务留 P3，此步同实例无碍）。
- adhoc：统一写读到 TsPool（`cmd/app/provider/pm.go`、`cmd/worker/adhoc.go` 的 `NewPgRepository` 从 PgPool 改 TsPool；export adhocDB `cmd/worker/aggregator.go` 对齐 TsPool）。
- kpi/counter devices 反查：`internal/pm/kpi/pg_repository.go:267`、`internal/pm/counter/pg_repository.go:334` 的 `SELECT FROM devices` 改打 PgPool（注入第二池或上移 service 层先查键）。
- kpi_definitions：UPSERT 从 TsPool 切回 PgPool。
- **B0**：把 7 超表的注册从 catalog dump 改显式 `create_hypertable + add_compression_policy + add_retention_policy`，`pm_metrics` 直接带 `chunk_time_interval => INTERVAL '4 hours'`，废弃独立 `000045`；加 CI `fresh down -v && up` 门。

**files_touched**：`internal/alarm/pg_store.go`、`internal/alarm/history_retention.go`、`internal/trace/pg_repository.go`、`cmd/acs/bootstrap.go`、`cmd/acs/main.go`、`cmd/acs/etc/config.{dev,test,prod,local}.yaml`（新增 tsdb 段）、`cmd/app/provider/modules.go`、`cmd/worker/main.go`、`cmd/app/provider/pm.go`、`cmd/worker/adhoc.go`、`cmd/worker/aggregator.go`、`internal/pm/kpi/pg_repository.go`、`internal/pm/counter/pg_repository.go`、`migrations/000001_init_schema.sql`、`migrations/000045_pm_metrics_chunk_interval_4h.sql`、`migrations/seed/000001_init_seed.sql`、`scripts/check-migrations.sh`（或 CI 配置）。

**verify**：`go build ./... && go test ./internal/alarm/... ./internal/trace/... ./internal/pm/...`；两 DSN 仍同实例下跑 `bash scripts/e2e_verify.sh` + smoke 套件（`omcgo/scripts/smoke/`）告警/trace/PM 列表接口全绿；`grep s.tsPool` 在 alarm/trace 不再为 0；**fresh `down -v && up` 全绿且 `_timescaledb_catalog.dimension` 中 `pm_metrics` 的 `interval_length` = 4h（14400000000）**。

### P1 跨库 JOIN 拆解（应用层两步 + 时序库 device_dim 冗余）

**goal**：把 7 处单 SQL 跨库 JOIN/子查询改成「主库取键/字典 → 时序库 IN 下推/本库 device_dim JOIN → Go 内回填」，使时序查询不再引用主库表。仍单实例，但 SQL 已可跨实例执行。

- 新增时序库 `device_dim(oui,sn,device_id,product_id,group_ids[],technology,band)`，由业务库定时/事件同步（设备/分组变更触发）。
- 逐条改 B1-B5（见 §6）。
- **B6**：`dashboard/efficiency.go:150` 的 `REFRESH MATERIALIZED VIEW CONCURRENTLY alarm_efficiency_metrics` 改打 tsPool，确认 matview 保有唯一索引以支持 `CONCURRENTLY`。
- 确认 `alarm_efficiency_metrics` 仅基于 `alarms_history`，随其迁时序库。

**files_touched**：`internal/pm/aggregator/{query,device_group,daily_group,hourly_group,weekly_group,monthly_group}.go`、`internal/pm/adhoc/handler.go`、`internal/pm/device_query_service.go`、`internal/pm/export/query.go`、`internal/alarm/pg_store.go`、`internal/dashboard/{efficiency,heatmap,service}.go`。

**verify**：`go test ./internal/pm/... ./internal/alarm/... ./internal/dashboard/...`；单测断言每条改造后的 SQL 不再 `FROM/JOIN devices/products/device_groups/device_group_members/device_params`；dashboard 注入 TsPool 后 KPI/告警图表接口 e2e 绿；10万级设备集 `ListMetricObjects` 两步查询性能压测对比单 SQL 基线达标。

### P2 迁移/种子 baseline 物理拆分（main/ + tsdb/ 两套独立 goose 序列）

**goal**：把单库 baseline 切成主库 baseline（去 timescaledb）+ 时序库 baseline（timescaledb 扩展 + 7 超表 + 6 rollup + matview + 策略），各自独立 goose 版本表。

- **前置交付物：70 个迁移文件分诊清单**——实仓有 **34 个 schema 迁移**（`migrations/*.sql`）+ **36 个 seed 迁移**（`migrations/seed/*.sql`，已核计数），逐一标注 main/tsdb 归属。注意 **seed 是 `goose_db_version` + `goose_db_version_seed` 两条独立号段、相互独立且有空洞**，不是单一 `000001`。只处理 `000001/000045` 会漏掉 `000009/000042`（alarm matview）、`000026`（pm_group_metrics_technology）及任何后续改超表的增量。
- 主库 schema：从 `000001_init_schema.sql` 摘除 `:31 CREATE EXTENSION timescaledb` 与 7 超表 + 6 rollup 的 `CREATE TABLE`。
- 时序库 schema：把 7 超表 + 6 rollup + `device_dim` 带到 tsdb baseline，采用 **B0 的显式 `create_hypertable + add_*_policy`**；`pm_metrics` 直接带 4h chunk。
- seed 拆分：main-seed 拿全部 public INSERT（RBAC/menus/字典）；**tsdb-seed 必须含 `000009_alarm_efficiency_metrics.sql` 与 `000042_alarm_efficiency_mttr_non_negative.sql`（alarm matview + 索引随 `alarms_history` 进 tsdb）**。
- compose 演进为 4 个迁移容器：`migrate-main-schema`/`migrate-main-seed`/`migrate-tsdb-schema`/`migrate-tsdb-seed`，各自 `--dsn`/`--path`/`GOOSE_TABLE`；app/acs/worker `depends_on` 全部四个 `completed_successfully`。
- Makefile 新增 `migrate-main-up` / `migrate-tsdb-up`。

**files_touched**：`migrations/000001_init_schema.sql`、`migrations/000045_*.sql`（已并入 B0）、`migrations/seed/000001_init_seed.sql`、`migrations/seed/000009_alarm_efficiency_metrics.sql`、`migrations/seed/000042_alarm_efficiency_mttr_non_negative.sql`、`migrations/tsdb/`（新建）、`migrations/README.md`、`deployments/docker/docker-compose.yml`、`Makefile`、`cmd/migrate/main.go`。

**verify**：两个空库分别 `goose up` 跑通（主库纯 pg16 镜像能独立 `down→up`，时序库重建 7 超表 + 策略）；`psql` 查 `_timescaledb_catalog.hypertable` 与 `policy_compression`/`policy_retention` 计数对得上；`bash scripts/check-migrations.sh` 无撞号；migrate-tsdb-schema 在 fresh DB 上不报「非超表」错；`pm_metrics` chunk = 4h。

> **主库降级口径**：主库降纯 pg16 的前提是**全部 7 张超表（含 `alarms_history`/`trace_messages`/`mr_records`，不只 PM 三表）都迁走、且主库无任何残留 timescaledb 依赖对象（函数默认值等）**。若不能确证，主库保留 timescaledb 扩展。

### P3 PM 入库拆事务（copy_ingest 跨实例幂等重设计）

**goal**：把 `pm_files`(主库)+`pm_metrics`(超表) 的单 tx 拆成两实例两步写，保留 at-least-once 幂等不丢不**重**（不双计数）。

- **写序固定为**「先在主库写/抢占 `pm_files` 标记（`ON CONFLICT DO NOTHING`）拿到去重判定 → 仅当 `RowsAffected()==1` 才写 `pm_metrics`（时序库）」——见 §6 阻断 B7，**写序绝不可反**。
- `copy_ingest.go:162-184` 拆两段：`pgPool` INSERT `pm_files` marker → `tsPool.CopyFrom(pm_metrics)`；两段都 durable 提交，明确两段间崩溃的重投语义（标记已建但 metrics 未写完 → 需可安全补写或显式重灌）。
- 因 `uq_pm_metrics_natural` 已删，`pm_metrics` 无跨文件去重，故标记是写 metrics 的**前置闸门**；若选 outbox 方案，幂等键仍落主库。

**files_touched**：`internal/pm/metrics/copy_ingest.go`、`internal/pm/metrics/pg_repository.go`、`cmd/worker/main.go`。

**verify**：`go test ./internal/pm/metrics/...`；**注入「标记已写、metrics 未写完」崩溃，重投后断言 `pm_metrics` 目标文件行数不翻倍**（不只 `DLQ=0`）；PM 入库压测（`omcgo/test/pmperf/`）对比拆事务前吞吐与 DLQ（应为 0）。

### P4 起第二实例 + 配置/可观测性/备份接线

**goal**：物理起 `postgres-tsdb` 实例，把 **8 处** `tsdb.dsn` 全部指过去，补齐监控与备份。

- dev compose 新增 `postgres-tsdb` 服务（pin `timescale/timescaledb:2.25.2-pg16`，独立 volume `tsdbdata` + `5433:5432` + 写吞吐调优档）；重切资源档（dev 宿主 Docker VM 资源紧、与 boss 栈共享，需评估）。
- **改 8 处 `tsdb.dsn`**（已核）：`cmd/app/etc/config.{dev,test,prod,local}.yaml` + `cmd/worker/etc/config.{dev,test,prod,local}.yaml`——**注意含 `config.local.yaml`（本机裸跑用，漏改会本地连错库）与 `worker/config.test.yaml`**。prod 用新 `${TSDB_HOST}/${POSTGRES_TSDB_*}` 变量。
- prod 离线包：`deployments/release/bundle/deploy/docker-compose.infra.yml` 加服务、`plan-resources.sh` 加第二实例内存/连接派生、`RESOURCE-PLANNING.md` 更新预算、`deploy/.env` 加凭据。
- otelcol：`deployments/monitoring/otelcol/config.yaml` 加第二个 `postgresql` receiver（`endpoint=postgres-tsdb:5432`），`metricstransform` 给 `pg_*` 补 `instance` label 防指标 collide。
- 备份：`scripts/db_backup.sh` 对两实例各跑一次；时序库体量大评估改物理备份/MinIO 重灌豁免。

**files_touched**：`deployments/docker/docker-compose.yml`、`cmd/app/etc/config.{dev,test,prod,local}.yaml`、`cmd/worker/etc/config.{dev,test,prod,local}.yaml`、`deployments/release/bundle/deploy/docker-compose.infra.yml`、`deployments/monitoring/otelcol/config.yaml`、`scripts/db_backup.sh`。

**verify**：`$COMPOSE up -d --build` 起两实例；两实例 `/readyz` 在册；otelcol 两实例 pg 指标在 Prometheus 带 `instance` label 不撞；`db_backup.sh` 对两实例各产出 dump。

### P5 数据搬迁 + 灰度切换

**goal**：把现有时序表数据从旧单实例搬到 `postgres-tsdb`，灰度验证后切流。

- 新时序库重放完整 `migrations/tsdb` 链，确认 timescaledb 扩展 + bgw scheduler 在跑、`pm_metrics` = 4h。
- 数据搬迁见 §8（含搬迁期写入承接）。
- 灰度：预发影子读验证两步查询与 rollup 正确性，对比 KPI 聚合值与单库基线一致 → 生产灰度 → 全量。
- 切流：`tsdb.dsn` 正式指向 `postgres-tsdb`，观察 PM 入库吞吐/DLQ、告警列表、dashboard 图表、adhoc 导出全链路。
- 切流后主库 `drop` 已迁走的 7 超表 + 6 rollup + matview——**安排在稳定运行 1-2 周后单独执行**。

**files_touched**：`scripts/`（数据搬迁脚本，新建）、`deployments/docker/docker-compose.yml`。

**verify**：切流后跑全量 smoke（`omcgo/scripts/smoke/` + e2e/smoke）+ PM 连续压测；对比切流前后 KPI 聚合值、告警计数、adhoc 导出结果一致；监控两实例 IO/CPU 确认写洪峰隔离生效（主库事务延迟不再受 PM 写影响）。

---

## 8. 数据迁移与切换

- **PM 数据**（`pm_metrics`/rollup）：优先从 MinIO 重灌（最干净），降低搬迁与回滚成本；可重灌故搬迁期无写入丢失风险。
- **`trace_messages`**：retention 仅 3d，搬迁期可接受丢弃/不搬。
- **不可重算且 retention 长的——必须给搬迁期写入承接机制**：
  - `alarms_history`（retention 365d）与 `mr_records`（retention 90d）**不可 MinIO 重算**，生产可达数十 GB。`pg_dump --table` custom 格式 + 跨实例 COPY 期间的**新写入必须有承接**：采用「**影子双写**（搬迁窗口内业务对新旧两库同写）或**短停写窗口** + 按 chunk COPY」，否则切流期这两类时序数据有丢失风险。
- **`device_dim`**：首次全量同步，后续定时/事件增量。
- 新时序库必须重放完整 `migrations/tsdb` 链，确认 timescaledb 扩展 + bgw scheduler 在跑、压缩/保留策略生效、`pm_metrics` = 4h，否则 chunk 退回 1d 触发已知瓶颈。

---

## 9. 回滚

分阶段可逆，每阶段独立回滚：

- **P0/P1/P3（纯代码 + 同实例）**：`revert` commit 重新部署即回，无数据风险（同实例下新旧 SQL 都能跑；P3 revert 回单 tx 版本仍原子）。**B0 修迁移的回滚**：保留旧 catalog dump 路径分支，新显式建表方案不挂载即不生效（但建议不回滚 B0，因它修的是既有 bug）。
- **P2（迁移拆分）**：回滚=继续用单库合并 baseline，新 tsdb 目录不挂载即不生效。
- **P4/P5（起第二实例 + 切流）——唯一带数据风险的回滚点**：
  1. 切流前**不删主库原超表**，保留只读窗口（建议 7-14 天）。
  2. 回滚只需把 8 处 `tsdb.dsn` 改回主库 host、`revert` P4 compose 改动重启 app/worker，数据回落主库原表；切流期新增时序数据反向 COPY 回主库，或 PM 走 MinIO 重灌补齐。
  3. `device_dim` 是冗余表，回滚后主库 JOIN 路径恢复，`device_dim` 留着不影响。
- **护栏**：切流采用预发影子验证 → 生产灰度 → 全量；任一步 KPI 聚合值/告警计数与单库基线不一致即停止切流回退、不删原表。整体回滚窗内不可逆操作仅「主库 `drop` 原超表」一步，安排在切流稳定运行 1-2 周后单独执行。

---

## 10. 验收清单

- **P0**：`grep s.tsPool` 在 alarm/trace 不再为 0；e2e + smoke 告警/trace/PM 列表全绿；**fresh `down -v && up` 全绿且 `pm_metrics` chunk = 4h（既有 bug 已修）**。
- **P1**：单测断言改造后 SQL 不再 JOIN 主库表；大设备集两步查询性能达标；matview REFRESH 走 tsPool。
- **P2**：两空库各自 `goose up` 通过；hypertable/policy 计数对得上；70 文件分诊清单完成；alarm matview（seed `000009/000042`）已归 tsdb-seed。
- **P3**：注入「标记已写、metrics 未写完」崩溃，重投后 `pm_metrics` 行数**不翻倍**；PM 压测 DLQ=0。
- **P4**：8 处 `tsdb.dsn` 全部改完（含两个 `config.local.yaml` + `worker/config.test.yaml`）；ACS 已加 `tsdb` 段 + `ConnectTimescale`；两实例 `/readyz` 在册；otelcol 指标带 `instance` label 不撞。
- **P5**：切流后 KPI 聚合值/告警计数/adhoc 导出与单库基线一致；`alarms_history`/`mr_records` 搬迁期写入承接生效、无丢失；监控确认主库事务延迟不再受 PM 写洪峰影响。

---

## 11. 待确认项（需进一步验证 / 不确定）

1. **B0 修复的实测复现**：评审在 throwaway 库实测 `set_chunk_time_interval('pm_metrics')` 在 fresh DB 报 `is not a hypertable`，且 seed dimension 写死 1d（已核 `migrations/seed/000001_init_seed.sql:120` `interval_length=86400000000`）。落地前应在干净库**一次性复现 + 验证显式 `create_hypertable` 方案**整条 fresh `goose up` 绿。
2. **显式 `create_hypertable` 重建 7 超表的等价性**：废弃 catalog dump 后，须逐表确认列/索引/压缩段/保留期与现状一致（尤其各超表的 dimension 配置、compress/retention policy 参数），避免重建丢配置。
3. **`device_dim` 同步机制**：定时 vs 事件触发、增量一致性窗口、设备/分组变更的传播延迟对 rollup 正确性的影响——需设计与压测。`group_ids[]` 数组在 rollup JOIN 中的展开性能待验证。
4. **搬迁期写入承接的具体形态**：`alarms_history`/`mr_records` 选「影子双写」还是「短停写窗口」，取决于业务可接受的停写时长与双写实现复杂度——需与运营/QA 对齐 SLA。
5. **dev 宿主资源**：Docker VM 资源紧、与 boss 栈共享端口，两实例（postgres + postgres-tsdb）的内存/端口/卷分配需重切资源档并实测（本机起栈配方见运维记录）。
6. **IN 列表分批阈值**：B4 的 `WHERE (oui,sn) IN (...)` 在 10万级设备时的分批大小与查询计划，需压测确定。
7. **生产 `tsdb.dsn` 变量化**：prod 用新 `${TSDB_HOST}/${POSTGRES_TSDB_*}` 变量、test 用第二实例 IP 的具体取值，需与部署侧确认。
8. **70 文件分诊清单本身**：本方案给出框架，但逐文件 main/tsdb 归属判定（尤其后续新增的改超表增量）需在 P2 实际产出并经 review，本文档未逐一枚举全部 70 个文件的归属。
