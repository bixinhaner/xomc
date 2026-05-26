# PRD: F05 MR 测量任务管理

**PRD ID**：F05-mr-task-management
**功能域**：F05 测量报告（MR）
**作者**：Claude AI（PM 角色）+ 待人工审批
**创建日期**：2026-05-25
**最后更新**：2026-05-25
**状态**：Draft（待审批）
**关联 Milestone**：`docs/project/milestone/2026Q2-to-RC.md`
**关联 Sprint**：Sprint 待分配（预计 W1-W5）
**关联 Risk**：见本 PRD §6 风险章节，待登记至 `docs/project/risk-register.md`
**关联开发计划**：[`mr-task-management-development-plan-20260525.md`](../mr-task-management-development-plan-20260525.md)
**规范来源**：[`MR_Feature_Analysis.md`](../../../MR_Feature_Analysis.md)（项目根目录）
**设计基线**：[`omcmb/design/04-modules/11-mr-management/05-mr-task-config.md`](../../../omcmb/design/04-modules/11-mr-management/05-mr-task-config.md)

---

## 1. 业务背景

当前 F05 MR 模块已具备**文件接收链路**（`internal/transfer/bridge.go` → MinIO → `internal/mr/collector` → MRO/MRS/MRE 解析器 → PG）以及**指标 / 映射 / 文件查询 API**（`/api/v1/mr/{files,data,indicators,mappings}`），**但缺少触发设备主动上报 MR 文件的"任务编排能力"**：

- 设备默认**不**上报 MR 文件，必须由 OMC 通过 TR-069 `SetParameterValues` 写入 `Device.FAP.MRMgmt.Config.{i}.*` 6 个参数（启用 + URL + 周期）后，设备才会按 `UploadPeriod` 周期 HTTP POST XML 上报
- 运营商无线优化场景下，规划人员需要在**指定时间窗口**（如夜间话务低谷或异常路段排查期间）针对**指定一批小站**开启 MR 上报，结束后自动关闭以减少设备负担与 OMC 存储压力
- 当前缺失能力意味着即使下游解析链路完备，**用户也无法触发任何 MR 数据采集**，整条 MR 业务链路实际**对用户不可用**

这是 F05 从"骨架完成"到"业务可用"必须关闭的最后一公里短板。它**不是新功能堆砌**，而是把已建好的接收/解析能力**真正接通运营场景**。

---

## 2. 用户故事

> **无线优化工程师**：As a 无线优化工程师，I want 选择一批小站并设置 `start_time` / `end_time` 创建 MR 测量任务，OMC 在到时自动开启 / 关闭设备 MR 上报，So that 我能在话务低谷采集 MR 数据用于网络优化分析，不用人工逐站操作。

> **运维值班人员**：As a 运维值班人员，I want 在任务列表里实时看到每个目标小站的上报健康状态（正常 / 异常 / 平台不支持 / 无权限），So that 我能快速识别哪些站没正常上报需要排查。

> **网管管理员**：As a 网管管理员，I want 在任意时刻手动停止一个执行中的 MR 任务，So that 当采集对线网造成异常影响时能立刻干预。

> **运营商规划人员**：As a 运营商规划人员，I want 通过文件传输中心的统一入口看到所有"会让设备主动产生文件"的任务（升级 / 日志 / 备份 / **MR**），So that 我对网管侧触发的设备主动行为有统一的全局视图。

> **存储 / 运维（保留策略）**：As a OMC 运维，I want MR 文件按 `MRFileSaveDays` 配置自动清理，So that MinIO / PG 存储不会无界增长。

---

## 3. 验收标准

### AC-1：任务创建与参数校验

```
Given: 用户已登录且具备 CODE_MR_TASK_CREATE 权限
When:  POST /api/v1/mr/tasks 提交合法参数（task_name / mr_type=MRS,MRE,MRO /
       statis_period=5120 / report_period=15 / start_time=未来 5 分钟 /
       end_time=start_time+1h / target_cells=[10 个小站])
Then:  - 返回 201 + task_id
       - mr_customize_task 写入一条 task_status='waitting'
       - mr_customize_task_progress 为每个 cell 写入一条 progress_status='pending'
       - 审计日志记录创建者、时间、目标范围

Given: 缺失 MRS/MRE/MRO 任一强制项
When:  POST /api/v1/mr/tasks
Then:  返回 400 + 字段级错误信息

Given: end_time <= start_time
When:  POST /api/v1/mr/tasks
Then:  返回 400「结束时间必须晚于开始时间」
```

### AC-2：到达 start_time 自动下发开启 SPV

```
Given: 一个 task_status='waitting' 的任务，start_time 已到达，目标小站包含
       1 个 IntelCR 平台设备（在线）+ 1 个未在支持列表的平台设备
When:  scheduler 下一个 tick（≤30 秒内）触发
Then:  - IntelCR 设备：ACS 下发 SetParameterValues 含 6 个参数
                       (MrEnable=true / Vendor / OmcName / MrUrl / PeriodicReportInterval=5120 / UploadPeriod=900)
                       设备响应成功 → progress_status='openSuccess'
       - 不支持平台设备：跳过下发 → progress_status='unsupport'
       - 任务整体 task_status 从 'waitting' 变 'on'
       - Prometheus mr_task_dispatched_total{result="success"} +1
```

### AC-3：设备周期上报后写入 Redis 心跳

```
Given: 任务已开启，IntelCR 设备开始按 UploadPeriod=900 秒周期上报 MR 文件
When:  设备通过 HTTP POST 上传 MR XML 到 OMC，OMC 落 MinIO 成功
Then:  - Redis 写入 MRFileReport_{cellCode}，TTL=1500 秒（按文档 §7.3 表）
       - mr.file.received 事件发布
       - 下游 mr/collector 完成解析（已有链路）
       - progress.last_heartbeat 更新为当前时间
```

### AC-4：到达 end_time 自动下发关闭 SPV

```
Given: 一个 task_status='on' 的任务，end_time 已到达
When:  scheduler 下一个 tick 触发
Then:  - 对每个 openSuccess 的 cell 下发 SetParameterValues
         单参数 (MrEnable=false)
       - 设备响应成功 → progress_status='closeSuccess'
       - 设备响应失败 → progress_status='closeFailure' 并记录 fault_code
       - 任务整体 task_status 变 'off'

Given: end_time=NULL（无限制任务）
When:  scheduler tick 检查
Then:  不触发自动关闭，仅响应用户手动 stop
```

### AC-5：上报状态健康监控

```
Given: 任务已 'on'，某 cell 在最近 3 个 UploadPeriod 周期内未上报任何文件
When:  scheduler 状态巡检 tick 触发
Then:  - Redis MRFileReport_{cellCode} 已过期不存在
       - 该 cell 的 progress.progress_status 维持 'openSuccess'，但
         progress.health_status（新字段）标记为 'abnormal'
       - Prometheus mr_heartbeat_missed_total{cell_code=...} +1
       - 前端列表对应 cell 显示红色徽标
```

### AC-6：手动停止

```
Given: 一个 task_status='on' 或 'waitting' 的任务
When:  POST /api/v1/mr/tasks/:id/stop（具备 CODE_MR_TASK_STOP 权限）
Then:  - 'waitting' 任务：直接置为 'termination'，不下发任何 SPV
       - 'on' 任务：对所有 openSuccess cell 下发关闭 SPV，最终状态 'off'
       - 审计日志记录操作人、时间、原因
```

### AC-7：文件保留清理

```
Given: sys_settings.MRFileSaveDays=3，当前日期 2026-05-25
When:  daily cleaner cron @00:00 触发
Then:  - MinIO {operatorCode}/mr/2026-05-22/ 及之前目录删除
       - mr_files 表对应记录删除
       - Prometheus mr_files_cleaned_total +N
       - 不影响任务表与进度表数据
```

### AC-8：文件传输中心入口聚合

```
Given: 用户访问 /transfer/center
When:  点击「MR 测量」分类标签
Then:  - 显示最近 5 条 MR 任务概要（任务名 / 状态 / 涉及 cell 数 / 时间范围）
       - 提供「打开 MR 任务管理」按钮跳转 /mr/tasks
       - 不允许在此页直接创建 MR 任务（保持职责分离）
```

### AC-9：权限与多租户隔离

```
Given: 用户 A 属于 operator_code='cmcc'
When:  GET /api/v1/mr/tasks
Then:  仅返回 operator_code='cmcc' 的任务

Given: 用户 A 尝试访问 operator_code='ctcc' 的任务详情
When:  GET /api/v1/mr/tasks/:id
Then:  返回 404（不暴露存在性）
```

---

## 4. 运营商差异矩阵

| 维度 | CMCC | CTCC | CUCC | 备注 |
|------|------|------|------|------|
| 支持平台 | IntelCR / BLQ / MLQ / MLN | 同 | 同 | 文档 §2 已覆盖；QAV3/QAV4 待规范明确，扩展点已预留 |
| `MRVendor` 默认值 | `Baicells` | `Baicells` | `Baicells` | 由 `sys_settings.MRVendor` 配置，单一厂商部署一致 |
| `MROMCName` 默认值 | `CMCC-OMC` | `CTCC-OMC` | `CUCC-OMC` | 按运营商命名规范 |
| URL 前缀策略 | 标准模式 | 标准模式 | 标准模式 | `sys_settings.independent.mr.enable=0` 时统一走 OMC 主地址 |
| 独立 MR 服务器 | 部分省份启用 | 待确认 | 待确认 | `independent.mr.enable=1` 时切换；运营商规范输入未到位 |
| MR 文件保留天数 | 3-7 天 | 3 天 | 3 天 | 默认 3 天，可由各运营商部署时覆盖 `MRFileSaveDays` |
| 告警上下游 | 上报集团网管 | 上报翼网管 | 上报沃云 | 通过 F08 北向接口（不在本 PRD 范围）|

**实施约束**：所有差异通过 `internal/carrier/*.go` 现有适配器注入（默认值、URL 前缀策略），**禁止** `if carrier == "cmcc"` 硬编码（按 `omcgo/CLAUDE.md §8.2`）。平台支持判断抽到 `internal/mr/task/platform.go`，未来加 QAV3/QAV4 在此扩展。

**待 PM 确认**：
- ⚠️ CTCC / CUCC 的「独立 MR 服务器」启用条件与 URL 规范
- ⚠️ 三家运营商对 MR 文件保留天数的合规要求（监管报送 / 用户数据安全）
- ⚠️ 是否需要按运营商定制 MR 文件命名格式（当前文档 §8 是统一格式 `FDD-baicells-...`）

---

## 5. 非目标

- ✂️ **不做 MDT 子功能**（原因：文档 §3.1 注明 MDT 独立处理，需要单独流程；下一里程碑评估）
- ✂️ **不引入 MongoDB 存储 GPS 轨迹**（原因：本项目数据栈固定 PG/TimescaleDB，沿用现有 `mr/parser/mro_parser.go` 落 PG；如未来有地理空间查询需求单独立项 PostGIS）
- ✂️ **不做 QAV3 / QAV4 平台适配**（原因：文档 §7.1 提及但未给出型号匹配规则，等运营商规范输入；扩展点已在 `platform.go` 预留）
- ✂️ **不扩展 UFTE 任务引擎**（原因：MR 任务的"长时持续 + cell 维度 + TTL 心跳"特征与 UFTE 一次性流转不匹配，强行套用风险高；走独立任务引擎 + UFTE 入口聚合）
- ✂️ **不做 MR 数据二次分析 / 报表**（原因：F05 报表子功能在 `pages/mr/Reports/` 单独迭代，本 PRD 只负责采集触发）
- ✂️ **不做 MR 任务的复杂调度（如 cron / 重复执行 / 任务依赖）**（原因：首版聚焦"单次时间窗口"，按需在 v2 扩展）
- ✂️ **不做按 cell 维度的中途启停**（原因：任务粒度即整体，cell 级精细控制等用户反馈后再考虑）

---

## 6. 依赖

### 阻塞项（必须先解决）

- [ ] CMCC/CTCC/CUCC 三家 MR 规范输入（影响 §4 待确认项）— Owner：PM，预计 Phase 0 结束前
- [ ] `internal/carrier/` 接口是否已暴露 `MRVendor()` / `MROMCName()` 方法（如未暴露需扩接口）— Owner：架构，Phase 1 启动前

### 被阻塞项（本功能不完成会影响什么）

- F05 MR 模块整体业务可用性（无任务触发 = 无数据采集 = 下游报表 / 优化分析无数据）
- F08 北向接口的 MR 数据上报（依赖 F05 数据采集就绪）
- 运营商**入网测试**与**联调验收**中 MR 项检查（影响 RC 发布判定）

### 外部依赖

- 设备端 `Device.FAP.MRMgmt.Config.{i}.*` 参数模型正确实现（Baicells 厂商已确认；其他厂商待 interop 测试）
- NTP 时间同步（多 OMC / 多 worker 部署时 `start_time` / `end_time` 调度依赖系统时钟）
- Redis 可用性（TTL 心跳监控的关键依赖，Redis 抖动会触发误报）

### 本期识别的风险（待登记至 `docs/project/risk-register.md`）

| 风险 | 等级 | 影响 | 缓解 |
|------|------|------|------|
| 多 worker 部署调度并发 | 高 | 重复下发 SPV | Redis SETNX 分布式锁 |
| Redis TTL 误报 | 中 | 用户信任受损 | "连续 N 次未命中"阈值（默认 2）|
| start_time 时钟漂移 | 低 | 秒级偏差 | 部署文档强制 NTP；30 秒调度容差 |

---

## 7. 度量（如何证明上线成功）

| 指标 | 基线 | 目标 | 度量方式 |
|------|------|------|---------|
| **可用性**：成功创建任务并完整走完生命周期的比率 | N/A（功能新增）| ≥ 95% | `mr_task_dispatched_total{result="success"} / total` |
| **下发延迟**：start_time → 第一个 SPV 实际下发的时间 | N/A | P95 ≤ 60 秒 | scheduler 巡检 30 秒 + ACS 队列延迟 |
| **关闭可靠性**：到达 end_time 后所有 openSuccess cell 关闭成功率 | N/A | ≥ 99% | `mr_task_close_total{result="success"} / total` |
| **心跳准确率**：Redis 心跳标记 abnormal 的 cell 中，确实没有新文件上报的比率（≠ 误报）| N/A | ≥ 90% | 抽样审计 + Grafana 面板 |
| **存储占用**：MR 文件清理后 MinIO 占用相对峰值的下降比 | N/A | 至少回收超期文件 100% | MinIO 桶大小对比 |
| **API 性能**：列表查询 P95 延迟 | N/A | < 300ms | Prometheus histogram |

**反例监控**（上线后看**不应发生什么**）：

- ❌ 不应出现 `mr_task_dispatched_total{result="duplicate"}` > 0（重复下发）
- ❌ 不应出现 ACS SPV 队列因 MR 任务积压（监控 `acs_command_queue_depth` 不超过基线 20%）
- ❌ 不应出现 worker 进程 CPU 因 scheduler 巡检超过 30%（100 任务 × 100 cell 规模）
- ❌ 不应出现 `mr_file_uploaded_total` 对未配置任务的 cell 持续上升（设备遗留启用状态）

---

## 8. 实施要点（非规范性，供参考）

> 详细计划见 [`mr-task-management-development-plan-20260525.md`](../mr-task-management-development-plan-20260525.md)。

**架构定位**：MR 任务**独立引擎**（`internal/mr/task/`），UFTE 仅做入口聚合，**不**扩展 UFTE。

**新增模块**：
- 后端 `omcgo/internal/mr/task/`：model / repository / service / handler / platform / dispatcher / scheduler / cleaner
- 前端 `omcmb/frontend-core/`：types / api / hooks / mock / i18n
- 前端 `omcmb/webcode/src/pages/mr/Tasks/`：index / CreateDrawer / DetailDrawer
- 前端 `omcmb/webcode/src/components/CellSelector/`（如未存在，与告警 / PM 模块共用）

**新增端点**：
- `POST /api/v1/mr/tasks` / `GET /api/v1/mr/tasks` / `GET /api/v1/mr/tasks/:id`
- `POST /api/v1/mr/tasks/:id/stop` / `DELETE /api/v1/mr/tasks/:id`
- `GET /api/v1/mr/tasks/:id/progress`

**新增迁移**（按 `omcgo/CLAUDE.md §5.5` 规则，从 `000185` 开始连续递增）：
- `000195_mr_customize_task.sql`
- `000196_mr_customize_task_progress.sql`
- `seed/000187_mr_sys_settings.sql`

**侵入修改**（仅一处）：
- `internal/transfer/bridge.go` MR 分支：文件落 MinIO 后写 Redis `MRFileReport_{cellCode}` TTL

**进程归属**：
- API（`POST/GET /api/v1/mr/tasks/*`）→ `omcgo-app`
- 调度器 / 清理器 → `omcgo-worker`
- SPV 下发实际执行 → `omcgo-acs`（通过现有 `internal/task/` 队列）

**预计工作量**：M-L / 25-30 人日（含联调测试）
**预计 Sprint 数**：5 周 / 1 个 Sprint+

---

## 9. 审批

| 角色 | 姓名/占位 | 日期 | 备注 |
|------|---------|------|------|
| 产品经理 | _待签名_ | | 需补充 §4 待确认项 |
| 架构师 | _待签名_ | | 确认决策 1-4（独立引擎 / Carrier 接口 / 不引入 MongoDB / worker 进程）|
| 领域专家（电信业务）| _待签名_ | | 确认 SPV 参数与文档 §4 / §5 完全一致 |
| 领域专家（TR-069）| _待签名_ | | 确认 SPV 报文规范与厂商联调可行 |
| QA / 发布经理 | _待签名_ | | 确认 DoD 与 Release Gate 适配 |

---

## 10. 变更记录

| 日期 | 版本 | 变更摘要 | 作者 |
|------|------|---------|------|
| 2026-05-25 | v1.0 | 初稿（基于 `MR_Feature_Analysis.md` 规范与现状盘点）| Claude AI |
