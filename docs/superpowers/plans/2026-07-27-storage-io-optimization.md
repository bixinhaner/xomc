# Storage I/O Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 降低 MinIO scanner 与 Redis AOF 的持续磁盘写放大，修复 PM/MR gzip 状态漂移，并让磁盘告警只报告真实排队饱和。

**Architecture:** MinIO 保持 scanner/ILM 功能但显式降速；PM/MR collector 把 gzip 魔数探测结果作为数据库权威状态，跳过冗余归档；Redis 保持 AOF 耐久性但降低 rewrite 频率；TimescaleDB 保留当前流式架构仍依赖的 7 天压缩策略。监控同时展示活动、队列、延迟、iowait 和 scanner。

**Tech Stack:** Go 1.24、PostgreSQL 16、TimescaleDB 2.25、Redis 7、MinIO、Prometheus、Grafana、Docker Compose、Bash

## Global Constraints

- 所有改动基于 `origin/main` 的最新提交。
- 不缩短 `PM_LATE_DATA_WINDOW=168h`，不改变 PM 数据业务保留期。
- Redis 保持 `appendonly yes`、`appendfsync everysec`、`noeviction`。
- 已 gzip 判定必须使用 gzip 魔数，不得只信任文件后缀。
- 不开发历史对象迁移或 MinIO 小对象打包。
- 完成全部开发后统一运行全量验证，验证通过后才提交 MR。

---

### Task 1: 磁盘 I/O Dashboard 与组合告警

**Files:**
- Modify: `deployments/monitoring/grafana/dashboards/nginx-host-overview.json`
- Modify: `deployments/monitoring/alerts/host-container-alerts.yml`
- Modify: `deployments/release/bundle/deploy/storage-compose_test.sh`

**Interfaces:**
- Consumes: node-exporter 的 `node_disk_*`、`node_cpu_seconds_total` 与 MinIO scanner 指标。
- Produces: 主机资源 Dashboard 的 I/O 活动、队列、延迟、吞吐、iowait/scanner 面板；`HostDiskIOSaturated` 组合告警。

- [ ] **Step 1: 在 storage compose 测试中增加 Dashboard 查询和组合告警断言**
- [ ] **Step 2: 运行测试，确认因面板/告警尚不存在而失败**
- [ ] **Step 3: 增加四组 I/O 面板和三条件组合告警**
- [ ] **Step 4: 用 JSON parser、Prometheus promtool 和 storage test 验证通过**

### Task 2: MinIO scanner 与 Redis AOF 部署参数

**Files:**
- Modify: `deployments/docker/docker-compose.yml`
- Modify: `deployments/docker/docker-compose.test.yml`
- Modify: `deployments/release/bundle/deploy/docker-compose.infra.yml`
- Modify: `deployments/release/bundle/deploy/storage-compose_test.sh`

**Interfaces:**
- Produces: 所有部署入口统一的 `MINIO_SCANNER_SPEED=slow`、Redis 1GiB/500% rewrite 阈值。

- [ ] **Step 1: 增加开发/release/test Compose 参数一致性失败断言**
- [ ] **Step 2: 运行测试确认失败**
- [ ] **Step 3: 修改三个 Compose 文件并同步注释**
- [ ] **Step 4: 运行 Compose config 和 storage test**

### Task 3: PM 上传去重窗口配置化

**Files:**
- Modify: `omcgo/internal/core/appconfig/config.go`
- Create: `omcgo/internal/core/appconfig/upload_test.go`
- Modify: `omcgo/cmd/acs/main.go`
- Modify: `omcgo/cmd/acs/etc/config.dev.yaml`
- Modify: `omcgo/cmd/acs/etc/config.prod.yaml`
- Modify: `omcgo/cmd/acs/etc/config.test.yaml`

**Interfaces:**
- Produces: `UploadConfig.EffectivePMDedupTTL()` 返回显式配置或 4 小时安全默认值。
- Consumes: `event.NewDeduper(redis, cfg.Upload.EffectivePMDedupTTL(), logger)`。

- [ ] **Step 1: 写 `PMDedupTTL` 默认 4 小时、显式值透传的失败测试**
- [ ] **Step 2: 运行测试确认缺少字段/方法而失败**
- [ ] **Step 3: 增加配置字段、默认方法并替换 ACS 24 小时常量**
- [ ] **Step 4: 更新三套 ACS YAML，运行 appconfig 与 ACS 测试**

### Task 4: PM gzip 状态原子入库并跳过冗余归档

**Files:**
- Modify: `omcgo/internal/pm/metrics/copy_ingest.go`
- Modify: `omcgo/internal/pm/metrics/sparse_ingest_integration_test.go`
- Modify: `omcgo/internal/pm/collector/collector.go`
- Modify: `omcgo/internal/pm/collector/result_normalization_test.go`

**Interfaces:**
- Produces: `metrics.FileMarker.RawCompressed bool`。
- Consumes: `compress.MaybeGunzip` 的 `applied` 返回值。

- [ ] **Step 1: 写 marker 保存 `raw_compressed=true` 的集成失败测试**
- [ ] **Step 2: 写 collector 把 gzip 探测结果传给 marker 的失败测试**
- [ ] **Step 3: 运行目标测试确认失败原因正确**
- [ ] **Step 4: 扩展 marker/INSERT，并把 `wasGzip` 传入 `ingestViaCopy`**
- [ ] **Step 5: 仅在 `!wasGzip` 时调度 rawArchiver**
- [ ] **Step 6: 运行 PM collector/metrics 测试**

### Task 5: MR gzip 标记并跳过冗余归档

**Files:**
- Modify: `omcgo/internal/mr/collector/collector.go`
- Create: `omcgo/internal/mr/collector/raw_archive_test.go`

**Interfaces:**
- Consumes: `MRStore.MarkCompressed` 与 `compress.MaybeGunzip` 的实际探测值。
- Produces: gzip MR 同步标真、明文 MR 继续异步归档。

- [ ] **Step 1: 写 gzip MR 标真且不调度归档、明文 MR 调度归档的失败测试**
- [ ] **Step 2: 运行测试确认失败**
- [ ] **Step 3: 在解析成功路径按 `wasGzip` 分支标记或调度**
- [ ] **Step 4: 运行 MR collector 测试**

### Task 6: TimescaleDB 压缩边界复核

**Files:**
- Modify: `omcgo/migrations/tsdb/000001_tsdb_schema.sql`
- Modify: `omcgo/internal/pm/aggregator/sparse_maintenance_test.go`
- Modify: `omcgo/migrations/README.md`

**Interfaces:**
- Consumes: 当前 worker 流式聚合启动路径与现有 7 天自动压缩策略。
- Produces: 明确保留现有策略，避免未启动维护器时原始 chunk 永久不压缩。

- [x] **Step 1: 复核 `RunSparseMaintenance` 是否由当前 worker 启动**
- [x] **Step 2: 确认流式聚合启动路径未运行旧维护器**
- [x] **Step 3: 保留两条 7 天自动 policy，撤销危险删除**
- [x] **Step 4: 运行 aggregator 测试和迁移静态检查**

### Task 7: 统一验证与 MR

**Files:**
- Verify only.

**Interfaces:**
- Produces: 可复核的测试、构建、配置和 MR 证据。

- [ ] **Step 1: 运行 `cd omcgo && gofmt`、`go build ./...`、`go test ./...`**
- [ ] **Step 2: 运行 `cd omcmb && npm run typecheck`**
- [ ] **Step 3: 运行 release storage/monitoring tests、JSON/YAML/Compose 配置校验**
- [ ] **Step 4: 检查 diff、无敏感信息、无无关文件**
- [ ] **Step 5: Conventional Commit 提交、推送分支并创建非 Draft MR**
