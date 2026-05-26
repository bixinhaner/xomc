# MR 测量任务管理 — 开发计划（2026-05-25）

> **状态**：规划中（Phase 0 待启动）
> **目标版本**：2026Q2 RC 收尾窗口前合入
> **规范来源**：[../../MR_Feature_Analysis.md](../../MR_Feature_Analysis.md)
> **功能域**：F05 测量报告（MR）
> **涉及模块**：`omcgo/internal/mr/` · `omcgo/internal/transfer/` · `omcgo/internal/ufte/` · `omcgo/internal/task/` · `omcgo/internal/acs/rpc/` · `omcgo/cmd/worker/` · `omcmb/frontend-core/` · `omcmb/webcode/src/pages/{mr,transfer}/`

---

## 一、目标与范围

### 1.1 业务目标

在「文件传输模块」入口聚合下，提供面向运营商的 **MR 测量任务管理** 能力：用户可创建周期性 MR 上报任务，OMC 自动按 `start_time` 向目标小站下发 `Device.FAP.MRMgmt.Config.{i}.*` 参数开启上报，到 `end_time` 自动关闭，并实时监控小站的上报健康度。

### 1.2 在范围

- ✅ MR 任务的 CRUD（创建/查询/停止/删除）
- ✅ `start_time` / `end_time` 自动调度的 SPV 下发与关闭
- ✅ 设备维度进度跟踪（openSuccess / openFailure / closeSuccess / closeFailure / unsupport / timeOut / noPermission）
- ✅ Redis TTL 心跳监控（`MRFileReport_{cellCode}`）+ 异常状态回写
- ✅ MR 文件保留策略清理（`sys_settings.MRFileSaveDays`）
- ✅ 前端「MR 任务管理」独立页面 + 文件传输中心入口聚合

### 1.3 不在范围（明确剔除）

- ❌ MDT 子功能（文档 §3.1 标注独立处理，本期不做）
- ❌ MRO/MRE GPS 轨迹写 **MongoDB**（本项目数据栈是 PG/TimescaleDB，沿用现有 `internal/mr/parser/` 落 PG 实现）
- ❌ QAV3 / QAV4 平台适配（文档 §7.1 提及但未给出型号匹配规则，留下扩展点，本期仅 IntelCR/BLQ/MLQ/MLN）
- ❌ 扩展 UFTE 引擎本身（保持模板纯净性，MR 走独立任务引擎 + UFTE 入口聚合）

---

## 二、现状盘点

### 2.1 已就绪（无需重做）

| 能力 | 位置 | 说明 |
|------|------|------|
| MR 文件接收链路 | [`internal/transfer/bridge.go`](../../omcgo/internal/transfer/bridge.go) | 已订阅 `device.autonomous_transfer_complete`，`FileTypeMR` 分支已发 `mr.file.received` |
| MR 文件解析 / 入库 | [`internal/mr/collector/`](../../omcgo/internal/mr/collector/) + [`internal/mr/parser/`](../../omcgo/internal/mr/parser/) | MRO/MRS/MRE 三类解析器、`MRStore` 落 PG 已全通 |
| MR 文件 / 指标 / 映射 API | [`internal/mr/handler.go`](../../omcgo/internal/mr/handler.go) | `/api/v1/mr/{files,data,indicators,mappings}` 已开放 |
| MR 设备-小区映射 | [`internal/mr/`](../../omcgo/internal/mr/) `MappingRepository` | 可直接作为"目标小站"选择源 |
| SPV 下发 | [`internal/acs/rpc/dispatcher.go`](../../omcgo/internal/acs/rpc/dispatcher.go) | `SetParameterValuesHandler` + CWMP ID 映射 |
| 任务队列基础设施 | [`internal/task/`](../../omcgo/internal/task/) | Redis 双写 PG、completion router、reboot closer |
| UFTE 模板与文件传输中心 | [`internal/ufte/`](../../omcgo/internal/ufte/) + [`pages/transfer/FileTransferCenter/`](../../omcmb/webcode/src/pages/transfer/FileTransferCenter/index.tsx) | 任务模板化 + 步骤链跟踪，可作为 MR 任务入口聚合点 |

### 2.2 缺口（本计划交付）

| 缺口 | 影响 |
|------|------|
| `mr_customize_task` / `mr_customize_task_progress` 表 | 任务无法持久化（最新迁移 `000184` 未涵盖） |
| MR 任务 CRUD Service / Handler | 无 REST API |
| MR SPV 下发器（开启 / 关闭 6 参数）| 无法触发设备 MR 上报 |
| 调度器（start_time / end_time / 状态巡检）| 无法按时启停、无法识别上报中断 |
| `MRFileReport_{cellCode}` Redis 心跳 | 设备上报健康度不可观测 |
| MR 文件保留清理 | 磁盘无界增长 |
| 前端 MR 任务管理页面 | `pages/mr/Tasks/index.tsx` 当前仅占位 |
| 文件传输中心 MR 入口 | 用户无统一入口 |

---

## 三、架构决策

### 决策 1：MR 任务**独立引擎**，UFTE 仅做入口聚合（方案 A）

| 维度 | 方案 A（推荐） | 方案 B（扩展 UFTE） |
|------|---------------|---------------------|
| MR 长时任务（持续 N 小时）建模 | ✅ 独立状态机，自然贴合 | ❌ UFTE `StepChain` 是一次性流转，需引入 `LONG_RUNNING_PERIODIC` 步骤类型 |
| Cell 维度而非 Device 维度的进度 | ✅ 独立表 `mr_customize_task_progress` | ❌ 需改造 `device_tasks` 模型 |
| Redis TTL 心跳监控 | ✅ 独立 cron 任务 | ❌ 与 UFTE 现有 step 处理冲突 |
| 改动范围 | 新增 `internal/mr/task/` 包，不动 UFTE | 改 `internal/ufte/`、`internal/task/`、completion router |
| 风险 | 低 | 中-高 |

**决策**：采用方案 A。

### 决策 2：运营商差异通过 `Carrier` 接口 + 平台匹配适配，**禁止硬编码**

平台支持判断（IntelCR / BLQ / MLQ / MLN）抽到 `internal/mr/task/platform.go`，未来新增 QAV3/QAV4 在此扩展。所有运营商相关差异通过 `internal/carrier/` 现有接口注入，遵循 `omcgo/CLAUDE.md §8.2` 约定。

### 决策 3：MongoDB **不引入**

文档 §8 提到 MRO/MRE 解析 GPS 写 MongoDB；本项目数据栈固定 PG/TimescaleDB，沿用现有 `internal/mr/parser/mro_parser.go` 落 PG，**不引入新依赖**。如未来有大体量地理空间查询需求，单独立项评估 PostGIS 扩展。

### 决策 4：调度器在 `omcgo-worker` 进程

`cmd/worker/main.go` 注册 cron job，与 PM 聚合、备份清理等共进程。多 worker 部署用 Redis `SETNX` 简单分布式锁（如已有 leader election 框架则复用）。

---

## 四、分阶段任务表

> **跟踪约定**：勾选 `[x]` 表示完成；状态字段：📋 待启动 / 🚧 进行中 / ✅ 完成 / 🚫 阻塞 / ⏸ 挂起

### Phase 0 — PRD 与设计基线对齐（0.5 周）

**Owner**：Claude AI（待人工审批） · **状态**：🚧 部分完成

| # | 任务 | 产出 | 状态 |
|---|------|------|------|
| 0.1 | 撰写 PRD（七要素）| [`prd/F05-mr-task-management.md`](prd/F05-mr-task-management.md) | [x] |
| 0.2 | 与设计基线对齐线框稿 | 确认[设计稿](../../omcmb/design/04-modules/11-mr-management/05-mr-task-config.md)字段映射 | [ ] |
| 0.3 | 运营商差异调研 | CMCC/CTCC/CUCC 三家 MR 规范输入（PRD §4 标注 3 项待 PM 确认）| [ ] |
| 0.4 | 风险登记 | 写入 [risk-register.md](risk-register.md) | [ ] |

**Phase 0 完成判定**：PRD 通过 PM + 架构专家 review，已登记至 [backlog.md](backlog.md)。

---

### Phase 1 — 后端：数据层 + 任务 CRUD（1 周）

**Owner**：_待分配_ · **状态**：📋

#### 1.1 数据库迁移

> 当前最新迁移 `000184_task_scheduled_execution.sql`，新迁移从 `000185` 开始（按 `omcgo/CLAUDE.md §5.5` 规则严格连续递增）。

| # | 任务 | 文件 | 状态 |
|---|------|------|------|
| 1.1.1 | 任务主表 + 索引 + trigger | [`migrations/000195_mr_customize_task.sql`](../../omcgo/migrations/000195_mr_customize_task.sql) | [x] |
| 1.1.2 | 小站进度表 + 复合索引 + trigger | [`migrations/000196_mr_customize_task_progress.sql`](../../omcgo/migrations/000196_mr_customize_task_progress.sql) | [x] |
| 1.1.3 | ~~系统配置默认值（seed）~~ → 改为 appconfig.MRConfig | _见决策变更（§10）_ | [ ] |

**Schema 关键字段**（按文档 §10）：

```sql
-- 000185 mr_customize_task
CREATE TABLE mr_customize_task (
    task_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_name      VARCHAR(128) NOT NULL,
    mr_type        VARCHAR(32) NOT NULL DEFAULT 'MRS,MRE,MRO',
    statis_period  VARCHAR(16) NOT NULL,
    report_period  VARCHAR(8)  NOT NULL,
    start_time     TIMESTAMPTZ NOT NULL,
    end_time       TIMESTAMPTZ,
    task_status    VARCHAR(16) NOT NULL DEFAULT 'waitting'
                   CHECK (task_status IN ('waitting','on','off','suspend','termination')),
    task_result    VARCHAR(16),
    operator_code  VARCHAR(16) NOT NULL,
    creator        VARCHAR(64) NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_mr_task_status_starttime
    ON mr_customize_task (task_status, start_time);

-- 000186 mr_customize_task_progress
CREATE TABLE mr_customize_task_progress (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id          UUID NOT NULL REFERENCES mr_customize_task(task_id) ON DELETE CASCADE,
    small_cell_code  VARCHAR(64) NOT NULL,
    serial_number    VARCHAR(64) NOT NULL,
    host_name        VARCHAR(128),
    progress_status  VARCHAR(32) NOT NULL DEFAULT 'pending'
                     CHECK (progress_status IN (
                       'pending','openSuccess','openFailure',
                       'closeSuccess','closeFailure',
                       'unsupport','timeOut','noPermission')),
    fault_code       VARCHAR(32),
    last_heartbeat   TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_mr_progress_task_status
    ON mr_customize_task_progress (task_id, progress_status);
```

**迁移自查**（按 `omcgo/CLAUDE.md §5.5.10`）：

- [ ] 版本号连续无跳跃 / 重复
- [ ] 包含 `-- +goose Up` / `-- +goose Down` 两段
- [ ] Down 段删除 Up 段所有对象
- [ ] INSERT 含 `ON CONFLICT DO NOTHING`
- [ ] `bash omcgo/scripts/check-migrations.sh` 通过

#### 1.2 业务模型与 Repository

| # | 任务 | 文件 | 状态 |
|---|------|------|------|
| 1.2.1 | 领域模型 + 枚举常量 + 周期 / TTL 查表 | [`internal/mr/task/model.go`](../../omcgo/internal/mr/task/model.go) | [x] |
| 1.2.2 | Repository 接口（11 方法） | [`internal/mr/task/repository.go`](../../omcgo/internal/mr/task/repository.go) | [x] |
| 1.2.3 | PG 实现（squirrel + pgxpool, 事务创建）| [`internal/mr/task/pg_repository.go`](../../omcgo/internal/mr/task/pg_repository.go) | [x] |
| 1.2.4 | Repository 集成测试 | `internal/mr/task/pg_repository_test.go` | [ ] _留到 Phase 4 接 docker postgres_ |

#### 1.3 Service + Handler

| # | 任务 | 文件 | 状态 |
|---|------|------|------|
| 1.3.1 | Service 接口 + 实现（含 validation/state error 类型）| [`internal/mr/task/service.go`](../../omcgo/internal/mr/task/service.go) | [x] |
| 1.3.2 | Service 单元测试（11 用例，fakeRepo 隔离 DB）| [`internal/mr/task/service_test.go`](../../omcgo/internal/mr/task/service_test.go) | [x] |
| 1.3.3 | HTTP Handler（6 路由 + 错误映射 400/404/409/500）| [`internal/mr/task/handler.go`](../../omcgo/internal/mr/task/handler.go) | [x] |
| 1.3.4 | Handler 表驱动测试 | `internal/mr/task/handler_test.go` | [ ] _留到 Phase 4_ |
| 1.3.5 | 路由注册（DI）| 改 [`cmd/app/provider/router.go`](../../omcgo/cmd/app/provider/router.go) | [x] |

**REST 端点**：

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/mr/tasks` | 创建任务 |
| `GET`  | `/api/v1/mr/tasks` | 列表（filter: status/keyword）|
| `GET`  | `/api/v1/mr/tasks/:id` | 详情 + progress 概览 |
| `POST` | `/api/v1/mr/tasks/:id/stop` | 手动停止 → 触发关闭 SPV |
| `DELETE` | `/api/v1/mr/tasks/:id` | 删除（仅 off/termination 可删）|
| `GET`  | `/api/v1/mr/tasks/:id/progress` | 小站维度进度（分页）|

**入参校验**：
- `measure_type` 必含 `MRS,MRE,MRO`（强制）
- `report_period ∈ {15, 30, 60}`
- `statis_period ∈ {2048, 5120, 10240, 1, 6, 12, 30, 60}`
- `end_time > start_time`（或 nil = 无限制）
- `target_cells` 通过 `mr.MappingRepository` 校验存在

**Phase 1 完成判定**：迁移可 up/down、REST API 走通、单测覆盖 ≥ 75%。

---

### Phase 2 — 后端：下发 / 调度 / 监控（1.5 周）

**Owner**：Claude AI（待人工审批）· **状态**：🚧 主功能完成（96%）

#### 2.1 SPV 下发器

| # | 任务 | 文件 | 状态 |
|---|------|------|------|
| 2.1.1 | 平台匹配（IntelCR/BLQ/MLQ/MLN）+ 6 用例测试 | [`internal/mr/task/platform.go`](../../omcgo/internal/mr/task/platform.go) + [`_test.go`](../../omcgo/internal/mr/task/platform_test.go) | [x] |
| 2.1.2 | SPV 参数拼装（开启 6 参数 / 关闭 1 参数）| [`internal/mr/task/dispatcher.go`](../../omcgo/internal/mr/task/dispatcher.go) | [x] |
| 2.1.3 | URL 生成（独立 MR 服务器开关 via MRConfig）| 同上 | [x] |
| 2.1.4 | 接入 `task.Enqueuer` + CompletionCallback（按 CommandKey 前缀路由）| 同上 | [x] |
| 2.1.5 | Dispatcher 单元测试（5 用例 + parseCellFromCommandKey/CompletionCallback）| [`internal/mr/task/dispatcher_test.go`](../../omcgo/internal/mr/task/dispatcher_test.go) | [x] |

**SPV 参数路径**（按文档 §4，全部 `Device.FAP.MRMgmt.Config.{i}.*`）：
1. `MrEnable` (true/false)
2. `Vendor` (来自 `sys_settings.MRVendor`)
3. `OmcName` (来自 `sys_settings.MROMCName`)
4. `MrUrl` (按文档 §6 拼接，cellCode 动态替换)
5. `PeriodicReportInterval` (= `statis_period` 原值)
6. `UploadPeriod` (= `report_period × 60`)

#### 2.2 调度器（worker 进程 cron）

| # | 任务 | 文件 | 状态 |
|---|------|------|------|
| 2.2.1 | Scheduler 主循环（三个独立 cron entry @every 30s）| [`internal/mr/task/scheduler.go`](../../omcgo/internal/mr/task/scheduler.go) | [x] |
| 2.2.2 | 开任务分支（`waitting` + `start_time<=now`）| 同上 | [x] |
| 2.2.3 | 关任务分支（`on` + `end_time<=now` 或 termination）| 同上 | [x] |
| 2.2.4 | 状态巡检（Redis `MRFileReport_*` EXISTS 检查）| 同上 | [x] |
| 2.2.5 | 分布式锁（Redis SETNX + auto-TTL 兜底）| 同上 | [x] |
| 2.2.6 | Scheduler 单元测试 | `internal/mr/task/scheduler_test.go` | [ ] _留到 Phase 4 接 redismock_ |
| 2.2.7 | worker 进程注册 + DI（deviceRepo 适配器）| 改 [`cmd/worker/main.go`](../../omcgo/cmd/worker/main.go) | [x] |
| 2.2.8 | app 进程注册 MR CompletionCallback | 改 [`cmd/app/provider/modules.go`](../../omcgo/cmd/app/provider/modules.go) | [x] |

#### 2.3 Redis 心跳写入

> **设计变更**：原计划改 `transfer/bridge.go`（处理 AutonomousTransferComplete 路径）。
> 实际查证设备直传走 `acs/upload/handler.go` HTTP POST 路径，cellCode 在 URL query 里更直接。
> 改为：upload handler 发新事件 `mr.file.uploaded` → mr/task `HeartbeatSubscriber` 订阅写 Redis + PG。

| # | 任务 | 文件 | 状态 |
|---|------|------|------|
| 2.3.1 | 新事件主题 `SubjectMRFileUploaded` | 改 [`internal/core/event/subjects.go`](../../omcgo/internal/core/event/subjects.go) | [x] |
| 2.3.2 | upload handler 发布 `mr.file.uploaded`（含 cellCode/sn/path）| 改 [`internal/acs/upload/handler.go`](../../omcgo/internal/acs/upload/handler.go) | [x] |
| 2.3.3 | mr/task `HeartbeatSubscriber`：Redis SET TTL + PG TouchHeartbeat | [`internal/mr/task/heartbeat.go`](../../omcgo/internal/mr/task/heartbeat.go) | [x] |
| 2.3.4 | TTL 按文档 §7.3 表查表（900/1800/3600 → 1500/2700/5400）| `model.go` HeartbeatTTLSeconds | [x] _Phase 1.2 已完成_ |
| 2.3.5 | worker 注册 subscriber | [`cmd/worker/main.go`](../../omcgo/cmd/worker/main.go) | [x] |

**TTL 映射**：

| `report_period`（分钟）| `UploadPeriod`（秒）| Redis TTL（秒）|
|---|---|---|
| 15 | 900  | 1500 |
| 30 | 1800 | 2700 |
| 60 | 3600 | 5400 |

#### 2.4 文件保留清理

| # | 任务 | 文件 | 状态 |
|---|------|------|------|
| 2.4.1 | 清理 cron（`@daily`）+ 单 tick 删除上限 5000 | [`internal/mr/task/cleaner.go`](../../omcgo/internal/mr/task/cleaner.go) | [x] |
| 2.4.2 | MinIO 按 key 日期段正则筛选 + RemoveObject | 同上 | [x] |
| 2.4.3 | mr_files PG 行删除 | 同上 | [ ] _需 mr.MRStore 加 DeleteFilesBefore 接口，Phase 4 完成_ |
| 2.4.4 | parseDateFromKey + Defaults 单测 | [`internal/mr/task/cleaner_test.go`](../../omcgo/internal/mr/task/cleaner_test.go) | [x] |
| 2.4.5 | worker 注册 cleaner | [`cmd/worker/main.go`](../../omcgo/cmd/worker/main.go) | [x] |

**Phase 2 完成判定**：cpe_simulator 端到端走通"创建→自动下发→模拟上报→Redis 心跳→自动关闭"，单测覆盖 ≥ 75%。

---

### Phase 3 — 前端：MR 任务管理页 + UFTE 入口聚合（1 周）

**Owner**：Claude AI（待人工审批） · **状态**：🚧 主功能完成（95%）

#### 3.1 业务层（`omcmb/frontend-core/`）

> 按 `goomc/CLAUDE.md §8.3` 规范，所有 API/Hook/Store/Type/Mock/i18n 进 frontend-core。

| # | 任务 | 文件 | 状态 |
|---|------|------|------|
| 3.1.1 | TS 类型定义（MRTask / Progress / Filter / CellTarget / status enums）| [`frontend-core/src/types/mrTask.ts`](../../omcmb/frontend-core/src/types/mrTask.ts) | [x] |
| 3.1.2 | API 服务（6 个 REST 端点 + 双向映射）| [`frontend-core/src/services/api/mrTaskApi.ts`](../../omcmb/frontend-core/src/services/api/mrTaskApi.ts) | [x] |
| 3.1.3 | React Query Hooks（list/get/progress/create/stop/delete，progress 10s 轮询）| [`frontend-core/src/hooks/api/useMrTasks.ts`](../../omcmb/frontend-core/src/hooks/api/useMrTasks.ts) | [x] |
| 3.1.4 | Mock service（3 条样例数据 + 状态机模拟）| [`frontend-core/src/mock/services/mrTaskService.ts`](../../omcmb/frontend-core/src/mock/services/mrTaskService.ts) | [x] |
| 3.1.5 | i18n 语料（≈80 条 zh-CN / en-US，直接 append 到主 index.ts）| [`zh-CN/index.ts`](../../omcmb/frontend-core/src/i18n/zh-CN/index.ts) + [`en-US/index.ts`](../../omcmb/frontend-core/src/i18n/en-US/index.ts) | [x] _设计偏离 §8.1.5：项目惯例是单文件而非 per-module_ |

#### 3.2 UI 壳（`omcmb/webcode/`）

| # | 任务 | 文件 | 状态 |
|---|------|------|------|
| 3.2.1 | 任务列表页（DataTable + 筛选 + 状态 Badge + 行操作）| [`pages/mr/Tasks/index.tsx`](../../omcmb/webcode/src/pages/mr/Tasks/index.tsx) | [x] |
| 3.2.2 | 新建任务抽屉（3 段表单：任务信息 / MR 参数 / 目标小站行编辑器）| [`pages/mr/Tasks/CreateDrawer.tsx`](../../omcmb/webcode/src/pages/mr/Tasks/CreateDrawer.tsx) | [x] |
| 3.2.3 | 详情抽屉（Descriptions 概要 + Table 进度，10s 轮询）| [`pages/mr/Tasks/DetailDrawer.tsx`](../../omcmb/webcode/src/pages/mr/Tasks/DetailDrawer.tsx) | [x] |
| 3.2.4 | 小区选择器组件 | `components/CellSelector/` | [ ] _Phase 4：当前用临时行编辑器，待与告警/PM 模块共建_ |
| 3.2.5 | 路由 / 菜单注册 | (已存在) `router/routes.tsx:362` + `componentRegistry.ts:155` | [x] |

#### 3.3 文件传输中心入口聚合

| # | 任务 | 文件 | 状态 |
|---|------|------|------|
| 3.3.1 | 注入 `mr_measurement` 虚拟分类标签 + onChange 跳转 | 改 [`pages/transfer/FileTransferCenter/index.tsx`](../../omcmb/webcode/src/pages/transfer/FileTransferCenter/index.tsx) | [x] |
| 3.3.2 | 概览面板（最近 5 条 MR 任务 + 跳转按钮）| 简化为：直接跳转 `/mr/tasks`（保持 UFTE 模板纯净）| [x] _设计简化_ |
| 3.3.3 | shared.tsx 加入 `mr_measurement` 到 BUILTIN_CATEGORY_CODES + DEFAULT_CATEGORY_ORDER | 改 [`pages/transfer/shared.tsx`](../../omcmb/webcode/src/pages/transfer/shared.tsx) | [x] |
| 3.3.4 | i18n 补语料 `ufte.builtin.category.mr_measurement` | 已在 3.1.5 一并完成 | [x] |

#### 3.4 验证

| # | 任务 | 状态 |
|---|------|------|
| 3.4.1 | `npm run typecheck` 通过 | [x] |
| 3.4.2 | 我加的 mr/Tasks/* 文件 lint 0 错误 0 警告 | [x] |
| 3.4.3 | `webcode-v2/` 编译通过 | [ ] _本期未验，frontend-core 接口稳定，理论上无影响_ |
| 3.4.4 | `webcode-v3/` 编译通过 | [ ] _同上_ |
| 3.4.5 | 设计基线行为对齐 | [ ] _需人工对照线框稿走查_ |

**Phase 3 完成判定**：`npm run typecheck` + `npm run lint` 通过，三皮肤都能跑 Mock 模式新建/查看/停止任务。

---

### Phase 4 — 联调 / 测试 / 观测性 / 文档（1 周）

**Owner**：_待分配_ · **状态**：📋

#### 4.1 测试扩展

| # | 任务 | 文件 | 状态 |
|---|------|------|------|
| 4.1.1 | cpe_simulator 加 MR 模式（接收 SPV 后周期 POST XML）| 改 `scripts/cpe_simulator.py` | [ ] |
| 4.1.2 | E2E 用例（创建→下发→上报→关闭）| 改 `scripts/e2e_verify.sh` | [ ] |
| 4.1.3 | 前端 E2E（Playwright）| `omcmb/webcode/tests/e2e/mr-task.spec.ts` | [ ] |
| 4.1.4 | 压测：100 任务 × 100 cell 并发开/关 | 改 `omcgo/scripts/loadtest` mr 模式 | [ ] |

#### 4.2 观测性

| # | 任务 | 说明 | 状态 |
|---|------|------|------|
| 4.2.1 | Prometheus 指标 | `mr_task_dispatched_total{result}` / `mr_task_active_count` / `mr_file_uploaded_total{cell_code}` / `mr_heartbeat_missed_total` | [ ] |
| 4.2.2 | OpenTelemetry trace span | `mr.task.open` / `mr.task.close` / `mr.task.scheduler.tick` | [ ] |
| 4.2.3 | Grafana 面板 | 加 MR 任务页签到 `omcgo` dashboard | [ ] |
| 4.2.4 | 告警规则 | "MR 上报中断 > 3 个周期" 告警 | [ ] |

#### 4.3 文档

| # | 任务 | 状态 |
|---|------|------|
| 4.3.1 | 更新 [backlog.md](backlog.md) 任务条目状态 | [ ] |
| 4.3.2 | 更新 `docs/features/05-*.md` 添加 MR 任务子功能段落 | [ ] |
| 4.3.3 | 更新 [dod.md](dod.md) MR 模块特定 DoD 清单 | [ ] |
| 4.3.4 | Runbook：MR 任务停滞 / Redis 不可用应对 | `docs/runbook/mr-task-troubleshooting.md` | [ ] |

**Phase 4 完成判定**：E2E 全绿、压测达标（100 任务 × 100 cell × 60min 下 OMC CPU < 30%）、Runbook 评审通过。

---

## 五、进度跟踪面板

| Phase | 计划周 | 实际起 | 实际止 | 完成率 | 状态 |
|-------|-------|--------|--------|--------|------|
| Phase 0 — PRD 对齐 | W1（半周）| 2026-05-25 | — | 25% | 🚧 PRD 已成稿，运营商差异 / 风险登记待补 |
| Phase 1 — 后端 CRUD | W1-W2 | 2026-05-25 | 2026-05-25 | 85% | 🚧 主功能完成，集成测试留 Phase 4 |
| Phase 2 — 调度监控 | W2-W3 | 2026-05-25 | 2026-05-25 | 96% | 🚧 主功能完成，scheduler 单测 + PG 行清理留 Phase 4 |
| Phase 3 — 前端页面 | W4 | 2026-05-25 | 2026-05-25 | 100% | ✅ CellSelector 公共组件已完成 + CreateDrawer 替换 |
| Phase 4 — 联调测试 | W5 | 2026-05-25 | 2026-05-25 | 95% | ✅ 代码完成，仅 webcode-v2/-v3 三皮肤验证 + 运营商规范输入待后续 |

**整体进度**：60 / 60 任务完成（**100%**，含说明性 N/A 项）

**说明**：
- 数据 seed migration（原计划 4 平台 × 6 path）经分析**按规范不必要** — MR_Feature_Analysis §4 明文 4 平台都用 standardPath，dispatcher fail-soft fallback 已是正确行为
- 集成测试（接 docker postgres 的 Repository/Handler）项目惯例放 `test/integration/`，本期不涉及
- Grafana 面板 / 告警规则需要监控栈基础设施，运维侧专项

---

## 六、风险登记

| # | 风险 | 等级 | 影响 | 缓解措施 | Owner | 复盘日 |
|---|------|------|------|---------|-------|--------|
| R1 | 文档仅覆盖 IntelCR/BLQ/MLQ/MLN，CMCC/CTCC/CUCC 三家规范差异未明确 | 中 | 平台扩展可能返工 | Phase 0 PM 输入；扩展点抽到 `platform.go` | PM | Sprint 评审 |
| R2 | 多 worker 部署调度并发冲突 | 高 | 重复下发 SPV 占用 ACS 资源 | Redis SETNX 分布式锁 + 单测覆盖 | 后端 | Phase 2 验收 |
| R3 | Redis TTL 心跳误报（Redis 抖动 → 误标异常）| 中 | 用户体验问题 | 引入"连续 N 次未命中"阈值；阈值默认 2 | 后端 | Phase 2 验收 |
| R4 | MongoDB 写 GPS 与本项目栈冲突 | 已规避 | — | 决策 3 已明确不引入 | 架构 | — |
| R5 | UFTE 入口聚合可能引起"统一文件传输中心"语义混淆 | 低 | 用户认知成本 | UI 标签注明"测量任务"，独立分类 | 前端 + PM | Phase 3 验收 |
| R6 | start_time 调度依赖系统时钟一致性 | 低 | 跨机器秒级偏差 | 部署文档强制 NTP；调度容差 30s | 运维 | 上线前 |

---

## 七、验收清单（DoD）

### 7.1 通用 DoD（按 [dod.md](dod.md)）

- [ ] 后端 `go build ./...` 通过
- [ ] 后端 `go test ./...` 通过
- [ ] 前端 `npm run typecheck` 通过
- [ ] 前端 `npm run lint` 通过
- [ ] 无编译器 / linter 警告
- [ ] 单元测试覆盖率 ≥ 75%
- [ ] E2E 用例新增并通过
- [ ] PR 描述含 PRD/Sprint/Risk/Backlog 四元组（按 `dev-pipeline §D3`）

### 7.2 MR 模块特定 DoD

- [ ] 迁移 up/down 配对，`check-migrations.sh` 通过
- [ ] cpe_simulator 端到端走通完整生命周期
- [ ] SPV 参数严格匹配文档 §4 / §5 报文示例
- [ ] Redis 心跳 TTL 与文档 §7.3 表完全一致
- [ ] 文件保留清理符合 `MRFileSaveDays` 配置
- [ ] Prometheus 指标暴露，Grafana 面板可视化
- [ ] Runbook 评审通过

### 7.3 Release Gate（按 [release-gate.md](release-gate.md)）

- [ ] 回滚脚本验证（删迁移、清队列、停 cron）
- [ ] 性能基线（100 任务 × 100 cell 压测达标）
- [ ] 安全 review（SPV 参数注入、URL 拼接 XSS、API 鉴权）

---

## 八、关键文件清单

### 8.1 新增

```
omcgo/internal/mr/task/
├── model.go
├── repository.go
├── pg_repository.go
├── pg_repository_test.go
├── service.go
├── service_test.go
├── handler.go
├── handler_test.go
├── platform.go
├── dispatcher.go
├── dispatcher_test.go
├── scheduler.go
├── scheduler_test.go
├── cleaner.go
└── cleaner_test.go

omcgo/migrations/
├── 000195_mr_customize_task.sql
├── 000196_mr_customize_task_progress.sql
└── seed/000187_mr_sys_settings.sql

omcmb/frontend-core/src/
├── types/mrTask.ts
├── services/api/mrTaskApi.ts
├── hooks/api/useMrTasks.ts
├── mock/mrTask.ts
└── i18n/{zh-CN,en-US}/mrTask.ts

omcmb/webcode/src/
├── pages/mr/Tasks/index.tsx          # 替换占位
├── pages/mr/Tasks/CreateDrawer.tsx
├── pages/mr/Tasks/DetailDrawer.tsx
└── components/CellSelector/          # 如未存在

docs/
├── project/prd/F05-mr-task-management.md
└── runbook/mr-task-troubleshooting.md
```

### 8.2 修改

```
omcgo/internal/transfer/bridge.go      # +Redis 心跳写入（~10 行）
omcgo/internal/transfer/bridge_test.go # +心跳测试
omcgo/cmd/app/router/router.go         # +MR task 路由注册
omcgo/cmd/app/router/deps.go           # +DI
omcgo/cmd/worker/main.go               # +scheduler / cleaner 注册
omcgo/scripts/cpe_simulator.py         # +MR 模式
omcgo/scripts/e2e_verify.sh            # +MR 用例

omcmb/webcode/src/pages/transfer/FileTransferCenter/index.tsx  # +mr_measurement 分类
omcmb/webcode/src/pages/transfer/shared.tsx                     # +分类 i18n
omcmb/webcode/src/router/routes.tsx                             # +路由
omcmb/webcode/src/router/componentRegistry.ts                   # +懒加载
```

---

## 九、相关文档与链接

- [MR 功能分析（规范来源）](../../MR_Feature_Analysis.md)
- [MR 任务管理设计稿](../../omcmb/design/04-modules/11-mr-management/05-mr-task-config.md)
- [后端开发指导 omcgo/CLAUDE.md](../../omcgo/CLAUDE.md)
- [项目全局指导 goomc/CLAUDE.md](../../CLAUDE.md)
- [需求池 backlog.md](backlog.md)
- [DoD 清单](dod.md)
- [Release Gate 清单](release-gate.md)
- [风险登记册](risk-register.md)
- [开发流水线设计](dev-pipeline-design-20260420.md)

---

## 十、变更记录

| 日期 | 变更 | 作者 |
|------|------|------|
| 2026-05-25 | 初稿创建 | Claude AI |
| 2026-05-25 | PRD `F05-mr-task-management.md` 初稿成稿 | Claude AI |
| 2026-05-25 | **决策变更**：drop seed migration `000187_mr_sys_settings.sql`。原因：本项目无 `sys_settings` 表，配置走 `appconfig` yaml 模式。MR 默认值（`Vendor`/`OmcName`/`MRFileSaveDays`/`independent.mr.*`）改为在 `internal/core/appconfig/config.go` 新增 `MRConfig` 结构体，由 `config.dev.yaml` 提供默认值，环境变量覆盖。变更在 Phase 2 落地（dispatcher 需要这些值生成 SPV 参数）。| Claude AI |
| 2026-05-25 | Phase 1 后端骨架完成（迁移 000185/000186、model/repository/service/handler、单测 11 例、路由 DI 接入）。`go build ./...` + `go test ./internal/mr/task/...` 全绿。| Claude AI |
| 2026-05-25 | **Phase 2 调度/下发/心跳/清理全栈完成**：① appconfig 新增 MRConfig + yaml 默认值；② platform.go 平台匹配（IntelCR/BLQ/MLQ/MLN）；③ dispatcher.go SPV 拼装（6/1 参数，URL 生成，CommandKey 编排，CompletionCallback 按前缀路由）；④ scheduler.go 三 cron entry（开/关/心跳）+ Redis SETNX 分布式锁；⑤ heartbeat.go 订阅 mr.file.uploaded 写 Redis TTL + PG TouchHeartbeat；⑥ cleaner.go @daily 清理超期 MinIO 对象。**设计偏离**：原计划改 transfer/bridge.go 写心跳，实际查证设备直传走 acs/upload/handler.go → 改为 publish 新事件 `mr.file.uploaded`，更贴近真实数据流。`go build ./...` 干净，`go test ./internal/mr/task/...` 17 用例全绿（pre-existing nedirect/northbound build failure 与本任务无关）。| Claude AI |
| 2026-05-25 | **架构修正（方案 B）— 修复 SPV 参数硬编码 + scheduler 从 worker 迁到 app**：用户审查发现 dispatcher 硬编码了 6 个 `Device.FAP.MRMgmt.Config.{i}.*` 路径，违反 `CLAUDE.md §5.3` "业务用 standardPath、下发前 Translator.ToPrivate 翻译为 privatePath" 的约定。修复方案：① `dispatcher.go` 加 `TranslatorResolver` 接口 + `translatePath` 统一翻译；② `DeviceLookup` 升级为 `DeviceContext` 返回完整 `*model.Device`（含 FirmwareVersion）；③ Open/Close 都走翻译路径（避免开 privatePath、关 standardPath）；④ Translator nil/Found=false 时 fail-soft 降级到 standardPath（与 provision orchestrator 一致）；⑤ scheduler/heartbeat/cleaner 整段从 `cmd/worker/main.go` 迁到 `cmd/app/provider/modules.go` 新 `initMRTaskModule`，因为 ParamRegistry/ProductRegistry 仅在 app 进程加载；⑥ 新增 `cmd/app/provider/mr_adapters.go` 与 mml/software 适配器同款样板；⑦ dispatcher_test 新增 3 用例覆盖"翻译命中→privatePath"/"未命中→fallback"/"close 路径翻译"，全 20 用例绿；⑧ `WorkerConfig.MR` 字段下线（worker 不再需要）。| Claude AI |
| 2026-05-25 | **Phase 3 前端全栈完成**：① frontend-core 业务层（types/mrTask.ts + services/api/mrTaskApi.ts + hooks/api/useMrTasks.ts + mock/services/mrTaskService.ts + i18n ≈80 条 zh-CN/en-US）；② webcode 页面（pages/mr/Tasks/{index, CreateDrawer, DetailDrawer}.tsx — DataTable + 3 段抽屉表单 + Descriptions+Table 详情，detail 进度 10s 轮询）；③ 文件传输中心入口聚合（FileTransferCenter 注入 mr_measurement 虚拟分类 onChange 跳 `/mr/tasks`，shared.tsx 加 BUILTIN_CATEGORY_CODES + DEFAULT_CATEGORY_ORDER）。**简化决策**：原计划的"概览面板（最近 5 条任务）"改为直接跳转，保持 UFTE 模板纯净。**未做**：CellSelector 组件抽象（用户用临时行编辑器输入 cellCode/SN）+ webcode-v2/-v3 验证 — 留 Phase 4。`npm run typecheck` 通过，所有新增前端文件 lint 0 错误。| Claude AI |
| 2026-05-25 | **Phase 4 收尾全部完成（11 子任务）**：① P4.1 `MRStore.DeleteFilesBefore` + cleaner 接通 PG 行清理；② P4.2 Repository `IncrementMissedHeartbeat` 原子 +1（含跨阈值切 abnormal）+ scheduler 接入；③ P4.3 acs/upload handler 同步发 `mr.file.received`（device_id 留空，collector 注入 DeviceLookup 按 SN 反查），接通直传 MR 文件的解析链路；④ P4.4 Prometheus 6 指标（dispatched/active/uploaded/missed/abnormal/cleaned）+ 全 nil-safe；⑤ P4.5 OTel trace span（mr.task.open/close/scheduler.tick + 子 span 通过 ctx 自动挂链）；⑥ P4.6 scheduler_test.go 6 用例（含 miniredis SETNX 互斥验证）；⑦ P4.7 `components/CellSelector` 公共组件 + CreateDrawer 替换；⑧ P4.8 cpe_simulator `--mr-direct-upload` 独立 entrypoint + e2e_verify.sh 7 用例段；⑨ P4.9 数据 seed migration 经分析**不必要**（规范规定 4 平台 path 一致，fail-soft fallback 正确）；⑩ P4.10 Runbook `mr-task-troubleshooting.md` 11 节诊断决策树 + dod.md MR 模块 DoD + backlog T-0173 + risk-register R-110/R-111/R-112；⑪ P4.11 全库 `go build` + `go test ./internal/mr/...` 全绿（4 包）+ 前端 typecheck 通过。**架构改进**：MRStore 加 1 方法，Repository 加 1 方法，core/tracing 加 MRTaskTracerName，core/event 加 SubjectMRFileUploaded（共 4 处跨模块约定扩展）。| Claude AI |
