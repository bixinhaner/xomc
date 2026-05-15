# PRD — TR069 报文跟踪（Message Trace）

**PRD ID**：F01-tr069-message-trace
**功能域**：F01 南向接口（主）+ F06 OMC-R 运维（前端入口）
**作者**：Claude（基于用户老 OMC 移植需求）
**创建日期**：2026-05-15
**最后更新**：2026-05-15
**状态**：Approved
**关联 Milestone**：`docs/project/milestone/2026Q2-to-RC.md`
**关联 Sprint**：sprint-13
**关联 Risk**：—（无新增 P0/P1 风险）
**设计文档**：`docs/design/TR069报文跟踪-设计.md`（详细架构、数据流、接口设计、决策记录均见此文档）
**Backlog**：T-0137

---

## 1. 业务背景（Why）

**一句话**：运维在现场排障时需要看到某台设备与 ACS 之间的 TR069 报文原文，定位"为什么这个参数下发失败"、"为什么 Inform 不来"这类协议层问题。

老 OMC 已有此能力，运维团队的使用习惯依赖它。新 OMC 重写后该能力缺失，导致：

- 现场遇到 CWMP 会话异常时只能靠 ACS 服务端日志推断，无法看到完整 SOAP 信封 / 参数列表 / Fault 详情，问题反查效率低
- 与设备厂商联调时无法提供"原始报文"作为对接证据，问题归因经常陷入"扯皮"
- 协议级 Bug（如某厂商私有参数路径解析异常）无现场抓证手段，开发只能凭日志反推协议交互

**为什么现在做**：F02 参数同步链 / F09 自动开站 / F06 设备管理已经稳定，开始进入"现场可运维"阶段，没有报文跟踪工具就没法让一线运维独立处理 TR069 协议问题，全部得回流给开发组。

---

## 2. 用户故事（Who / What）

> As a **网管运维工程师**，I want **选定一台问题设备开启 N 分钟抓包**，So that **我能看到这段时间该设备与 ACS 之间所有 SOAP 报文，独立完成协议层问题定位**。

> As a **现场工程师**，I want **下载抓包结果为 XML 文件**，So that **我能把报文证据发给设备厂商或开发组，加速联调**。

> As a **运维主管**，I want **在网管界面看到当前是否有抓包任务正在进行**，So that **我知道现场谁在排哪台设备的故障，避免重复操作**。

---

## 3. 验收标准（Given/When/Then）

```
AC-1 启动抓包
Given: 设备 SN "BLQ-001" 当前无活跃跟踪任务
When:  运维在 Web 界面选定该设备并点击"开始抓取"，输入采集时长 5 分钟
Then:  - 后端创建状态为 running 的跟踪任务，expires_at = now + 5min
       - 所有在线 ACS 实例 1 秒内将该 SN 加入抓包白名单
       - 其他在线且具 trace:read 权限的用户通过 WebSocket 收到任务开始通知
```

```
AC-2 报文落库
Given: 任务运行中
When:  CPE 与 ACS 完成一次 CWMP 会话（含 Inform + RPC 交互 + Empty Request）
Then:  - 该会话每条 SOAP 请求与响应均落库（direction = in/out 各一条）
       - 报文 ≤ 32KB 时存在 PG inline 列；> 32KB 时元数据存 PG，原文 GZIP 存 MinIO trace-bulk bucket
       - 报文原样落库，不做任何字段擦除或替换
```

```
AC-3 在线查看
Given: 任务下已落库 N 条报文
When:  运维在 Web 界面点击该任务的"查看报文"
Then:  - 前端分页展示报文列表（时间倒序，含 RPC method / 方向 / 时间戳）
       - 点击某条报文，前端将 XML 美化展示
       - 查询接口在 P95 < 500ms 内返回 50 条/页
```

```
AC-4 下载 XML
Given: 任务下已落库 N 条报文
When:  运维点击"下载 XML"，选择任务 ID 和时区
Then:  - 后端生成文件名形如 "BLQ-001_2026-05-15T10-00-00+08-00.xml"
       - 文件内每条报文按时间排序，含时间戳分隔头
       - 单文件 < 5MB 时直接流式返回；超过则异步生成到 MinIO，前端轮询并通过预签名 URL 下载
```

```
AC-5 超时自动停止
Given: 任务 expires_at 已过，且用户未手动停止
When:  worker 巡检触发
Then:  - 任务状态从 running → stopped
       - ACS 白名单 1 秒内移除该 SN
       - 推送 stopped 事件给在线用户
```

```
AC-6 多 ACS 实例聚合
Given: 设备 "BLQ-001" 的会话在 N 分钟窗口内由两个不同 ACS 实例服务过（如 LB 切换）
When:  运维查询该任务下报文
Then:  - 返回结果包含两个实例采集的所有报文
       - 报文按 captured_at 时间戳完整排序，运维感知不到底层多实例
```

```
AC-7 任务保留与查询
Given: 任务已结束 ≤ 3 天
When:  运维在历史任务列表中查询
Then:  - 任务记录依然存在，可查看完整报文（不被覆盖）
       - 超过 3 天后报文表自动 drop chunk，任务记录可保留但报文不可查
```

---

## 4. 运营商差异矩阵

| 维度 | CMCC | CTCC | CUCC | 备注 |
|------|------|------|------|------|
| —    | —    | —    | —    | **无差异 — 三家一致**。SOAP/CWMP 是设备 ↔ ACS 的协议层数据，运营商对此无定制；展示与导出均按原文 |

---

## 5. 非目标（Non-Goals）

- ✂️ **不做"全网全量录制"开关**（原因：10 万级网中所有设备同时录制会造成灾难性数据量；老 OMC `tr069MsgEnbEnable` 模式弃用）
- ✂️ **不纳入 eGW 企业网关**（原因：当前主线是 CPE + 4G/5G 小基站；eGW 走独立通道，后续阶段评估）
- ✂️ **不做报文敏感字段 mask / 脱敏**（原因：报文原样保真，运维能看到 CPE 真实交互；权限由 RBAC 控制）
- ✂️ **不做报文内容语义解析 / 参数自动提取**（原因：这是 F02 配置同步的职责，不混入跟踪功能）
- ✂️ **不做单 SN 报文限流 / 单用户每日任务上限**（原因：内部运维工具，无滥用场景）

---

## 6. 依赖

### 阻塞项（必须先解决）

无。所有基础设施（PostgreSQL/TimescaleDB、MinIO、NATS JetStream、`internal/notification/` WebSocket Hub、ACS `protocolLogger` 钩子点、`internal/core/event` EventBus）均已就绪。

### 被阻塞项（本功能不完成会影响什么）

- 现场运维独立处理 TR069 协议问题的能力（当前依赖回流给开发组）
- 与设备厂商联调时的"原始报文"取证能力

### 外部依赖

无。

---

## 7. 度量（如何证明上线成功）

| 指标 | 基线 | 目标 | 度量方式 |
|------|------|------|---------|
| 一线运维独立解决 TR069 协议问题占比 | ~30%（估算） | ≥ 70% | 运维工单标签统计（上线 1 个月后） |
| 报文采集端到端延迟（CPE 发送 → PG 落库可查） | N/A | P95 < 5s | `trace_capture_latency_seconds` Histogram |
| 单条报文采集对 ACS 主路径的影响 | N/A | P99 < 1ms | ACS handler latency 直方图对比（开/关跟踪状态） |
| 报文丢失率 | N/A | 0% | `trace_messages_dropped_total / trace_messages_captured_total` |

**反例监控**（不应发生）：

- ACS handler P99 latency 不应因开启跟踪而增加超过 5%
- TimescaleDB 落库速率不应跟不上 NATS JetStream 投递（JetStream 队列堆积告警）
- 不应出现"任务 expires 后 SN 仍在白名单内"（NATS 漏消息 + 兜底也失效）

---

## 8. 实施要点（非规范性，供参考）

- 预计涉及模块（后端）：新建 `internal/trace/`（handler/service/repository/cache）+ 修改 `internal/acs/handler.go` 增加 capture hook
- 预计涉及模块（前端）：`omcmb/frontend-core/src/services/api/traceApi.ts` + `omcmb/frontend-core/src/hooks/api/useTrace.ts` + `omcmb/webcode/src/pages/ops/MessageTrace/`（或挂到设备详情页 Tab）
- 预计新增端点：`POST /api/v1/trace/tasks`、`GET /api/v1/trace/tasks/{id}/messages` 等 7 个（详见设计文档 §5）
- 预计新增迁移：1 张任务表 + 1 张报文 hypertable + 1 个 MinIO bucket（`trace-bulk`）
- 预计新增 NATS subject：`trace.task.{started,stopped,purged}`、`trace.message.captured`
- 预计工作量：**L（3-5 天）** — M1 MVP 1 个 Sprint / M2 工程化 1 个 Sprint / M3 加固 0.5 Sprint，建议拆 sub-task

**注**：完整架构、数据流、决策记录见 `docs/design/TR069报文跟踪-设计.md`。实现细节以代码评审为准。

---

## 9. 审批

| 角色 | 姓名/占位 | 日期 | 备注 |
|------|---------|------|------|
| 产品经理 | user（kevin） | 2026-05-15 | 设计决策 D1-D10 已确认 |
| 架构师 | Claude | 2026-05-15 | 与新 OMC 三进程架构对齐 |
| 领域专家 | Claude（TR-069 协议） | 2026-05-15 | 拦截点选 `protocolLogger` 附近 |
| QA/发布经理 | — | — | S6 commit 前补审 |

---

## 10. 变更记录

| 日期 | 版本 | 变更摘要 | 作者 |
|------|------|---------|------|
| 2026-05-15 | v1.0 | 初稿；基于设计文档定稿 10 项决策同步落 PRD | Claude |
