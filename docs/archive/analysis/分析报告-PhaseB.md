# Phase B 完成度分析报告 — 中等工作量模块

## 概述

Phase B 涵盖 Config Baselines CRUD、Config Tasks & Neighbors、FTP Config、MR Indicators & Mappings 四个模块。定位为"中等工作量"阶段，主要是配置管理和测量报告基础设施的后端实现。

---

## 1. 模块清单与端点统计

| 模块 | 新增端点 | 新增表 | 新增 Go 文件 | 代码行数 |
|------|---------|-------|-------------|---------|
| Config Baselines | 5 | 1 | 5 | ~1,090 |
| Config Tasks & Neighbors | 3 | 2 | (共用 baseline 包) | ~包含在上方 |
| FTP Config | 5 | 1 | 4 | ~728 |
| MR Indicators & Mappings | 6 | 2 | 4 | ~1,062 |
| **合计** | **19** | **6** | **13** | **~2,880** |

---

## 2. 后端实现分析

### 2.1 Config Baselines CRUD

- **位置**: `omcgo/internal/config/baseline/`
- **文件**:
  - `handler.go` (288 行) — HTTP 处理
  - `model.go` (114 行) — BaselineConfig, ConfigTask, NeighborParam
  - `service.go` (140 行) — 业务逻辑
  - `repository.go` (28 行) — 接口定义
  - `pg_repository.go` (520 行) — PostgreSQL 实现
- **端点**:
  - `GET    /config/baselines` → `handler.go:88` — 分页 + device_type/status 过滤
  - `POST   /config/baselines` → `handler.go:116` — 创建基线配置
  - `GET    /config/baselines/:id` → `handler.go:143`
  - `PUT    /config/baselines/:id` → `handler.go:160`
  - `DELETE /config/baselines/:id` → `handler.go:193` — 204 No Content
- **迁移**: `000027_create_config_baselines.up.sql` — config_baselines 表

### 2.2 Config Tasks & Neighbors

- **位置**: 同 `omcgo/internal/config/baseline/` (共用包)
- **端点**:
  - `GET  /config/tasks` → `handler.go:211` — 任务列表
  - `POST /config/tasks` → `handler.go:236` — 创建配置任务 (param-sync/batch-config/baseline-apply/neighbor-update)
  - `GET  /config/neighbors` → `handler.go:267` — 邻区列表
- **迁移**: `000027` 同表 — config_tasks, config_neighbors
- **模型**: ConfigTask (5 种状态 × 4 种类型), NeighborParam (intra-freq/inter-freq/inter-rat)

### 2.3 FTP Config

- **位置**: `omcgo/internal/omcr/backup/`
- **文件**:
  - `ftp_model.go` (30 行) — FTPConfig 结构
  - `ftp_repository.go` (17 行) — 接口
  - `ftp_pg_repository.go` (213 行) — PostgreSQL 实现
  - `handler.go` (FTP 部分 ~143 行，325-468)
- **端点**:
  - `GET    /backup/ftp-configs` → `handler.go:325`
  - `POST   /backup/ftp-configs` → `handler.go:350`
  - `PUT    /backup/ftp-configs/:id` → `handler.go:387`
  - `DELETE /backup/ftp-configs/:id` → `handler.go:435`
  - `POST   /backup/ftp-configs/:id/test` → `handler.go:451` — 连接测试
- **迁移**: `000028_create_ftp_configs.up.sql` — ftp_configs 表

### 2.4 MR Indicators & Mappings

- **位置**: `omcgo/internal/mr/`
- **文件**:
  - `indicator_model.go` (50 行) — MRIndicator, MRDeviceMapping
  - `indicator_repository.go` (22 行) — 接口
  - `pg_indicator_repository.go` (352 行) — PostgreSQL 实现
  - `handler.go` (Indicator/Mapping 部分 ~171 行)
- **端点**:
  - `GET /mr/indicators` → `handler.go:193` — 分页查询
  - `GET /mr/indicators/all` → `handler.go:217` — 全量（无分页）
  - `GET /mr/mappings` → `handler.go:278` — device_sn/enabled 过滤
  - `PUT /mr/mappings/:id` → `handler.go:303` — 更新采样间隔
  - `PUT /mr/mappings/:id/toggle` → `handler.go:337` — 启用/禁用切换
  - `GET /mr/indicators/:code/stats` → `handler.go:227` — 指标统计
- **迁移**: `000029_create_mr_indicators.up.sql` — mr_indicators + mr_device_mappings + 5 条种子数据

---

## 3. 前端对齐分析

| 后端端点 | 前端 API 文件 | 前端方法 | Hook |
|---------|-------------|---------|------|
| GET /config/baselines | `configBaselineApi.ts` | `getBaselines()` | `useBaselineConfigs()` |
| POST /config/baselines | `configBaselineApi.ts` | `createBaseline()` | `useCreateBaselineConfig()` |
| GET /config/baselines/:id | `configBaselineApi.ts` | `getBaselineById()` | `useBaselineConfigById()` |
| PUT /config/baselines/:id | `configBaselineApi.ts` | `updateBaseline()` | `useUpdateBaselineConfig()` |
| DELETE /config/baselines/:id | `configBaselineApi.ts` | `deleteBaselines()` | `useDeleteBaselineConfigs()` |
| GET /config/tasks | `configBaselineApi.ts` | `getTasks()` | `useConfigTasks()` |
| POST /config/tasks | `configBaselineApi.ts` | `createTask()` | `useCreateConfigTask()` |
| GET /config/neighbors | `configBaselineApi.ts` | `getNeighbors()` | `useNeighborParams()` |
| GET /backup/ftp-configs | `backupApi.ts` | `getFTPConfigs()` | `useFTPConfigs()` |
| POST /backup/ftp-configs | `backupApi.ts` | `createFTPConfig()` | `useCreateFTPConfig()` |
| PUT /backup/ftp-configs/:id | `backupApi.ts` | `updateFTPConfig()` | `useUpdateFTPConfig()` |
| DELETE /backup/ftp-configs/:id | `backupApi.ts` | `deleteFTPConfigs()` | `useDeleteFTPConfigs()` |
| POST /backup/ftp-configs/:id/test | `backupApi.ts` | `testFTPConnection()` | `useTestFTPConnection()` |
| GET /mr/indicators | `mrApi.ts` | `getIndicators()` | `useMRIndicators()` |
| GET /mr/indicators/all | `mrApi.ts` | `getAllIndicators()` | `useAllMRIndicators()` |
| GET /mr/mappings | `mrApi.ts` | `getMappings()` | `useMRMappings()` |
| PUT /mr/mappings/:id | `mrApi.ts` | `updateMapping()` | `useUpdateMRMapping()` |
| PUT /mr/mappings/:id/toggle | `mrApi.ts` | `toggleMapping()` | `useToggleMRMapping()` |
| GET /mr/indicators/:code/stats | `mrApi.ts` | `getIndicatorStats()` | `useMRIndicatorStats()` |

**对齐率: 19/19 = 100%**

---

## 4. E2E 测试覆盖

| 测试段 | 用例数 | 覆盖范围 |
|-------|-------|---------|
| S61: Config Baselines CRUD | 7 | list → create → get → update → verify → delete 204 → get 404 |
| S62: Config Tasks & Neighbors | 4 | tasks list/create, neighbors list, field check |
| S63: FTP Config CRUD | 6 | list → create → update → test → delete 204 → filter |
| S64: MR Indicators & Mappings | 5 | indicators paginated/all, mappings list/update/toggle |
| **合计** | **22** | |

**全部通过** (427/427 PASS)

---

## 5. 问题发现与修复

| 问题 | 模块 | 原因 | 修复 |
|------|------|------|------|
| MR toggle 返回 400 EOF | MR Mappings | E2E 脚本未发送 JSON body | 补充 `{"enabled":false}` 请求体 |

修复后全部通过。

---

## 6. 结论

Phase B 以中等代码量（~2,880 行 Go + 6 张表 + 3 个迁移文件）实现了 19 个端点。Config Baselines 模块为批量配置管理提供了完整 CRUD 能力，FTP Config 为备份恢复的远程传输奠定基础，MR Indicators 补齐了测量报告的指标元数据管理。所有端点 100% 前后端对齐，22 个 E2E 用例全部通过。
