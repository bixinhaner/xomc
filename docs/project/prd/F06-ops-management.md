# PRD: F06 运维管理（Ops Management）

**PRD ID**：F06-ops-management
**功能域**：F06 OMC-R 核心 / 运维管理
**作者**：Claude（代 Owner = 电信业务专家 + 架构专家）
**创建日期**：2026-05-09
**最后更新**：2026-05-09
**状态**：Draft（调研稿，未排期）
**关联 PRD**：F06-license-management、F06-mml（待立项）、F06-software-upgrade（已落地）、F06-backup（已落地）
**关联 Backlog**：—（拟立 T-0101 ~ T-0107 拆分实施）
**关联 Risk**：—

> **本 PRD 的范围与"运维"边界**：本文档梳理一线网络运维工程师在 OMC 中**日常面向单台或批量设备**的"动手操作"能力，含命令、模板、任务、诊断、文件收集五大主线 + 巡检 / 维护窗口 / 审计 / 知识库等配套能力。与"配置管理（F02 模板/基线）"、"故障管理（F04 告警）"、"性能管理（F03 PM/KPI）"是**消费方关系而非替代关系** — 运维管理只编排，不重造。

---

## 0. 元信息与适用范围

### 0.1 边界一句话

> 运维管理 = "在 OMC 前台对设备发起任何**不属于** 配置 / 告警 / 性能 / 升级 / 备份 主线的、需要交互或可被记录的操作或诊断"。
> 是 "OMC 操作中台"，不是新工具，是已有 RPC / 文件传输 / 任务队列 / MML 能力的**业务编排层**。

### 0.2 适用对象

| 角色 | 主要诉求 |
|------|---------|
| **NOC L1 (网络运维一线)** | 用模板/命令快速处置告警、定位故障；不需要懂底层 RPC |
| **NOC L2 (二线)** | 自定义诊断脚本、批量回滚、收集底包帮 L3/研发分析 |
| **现场工程师 (FE)** | 远程协助现场重启 / 取日志 / 测速 |
| **OEM 售后** | 远程登录客户 OMC 收集底包、远程派任务 |
| **系统管理员 (SysAdmin)** | 维护模板库、审批高风险任务、控制权限 |
| **审计员** | 复盘所有运维操作（谁在何时对哪台设备做了什么）|

### 0.3 不在范围（明确剔除）

| 项 | 归属 |
|----|------|
| TR-069 SOAP/XML 协议栈、会话状态机 | F01 ACS |
| 配置模板编辑 / 基线管理 / 配置同步 | F02 config/template、config/baseline |
| 告警接收 / 去重 / 升级策略 | F04 alarm |
| PM 文件采集与 KPI 计算 | F03 pm |
| MR 文件采集与解析 | F05 mr |
| 固件批量升级 (软件管理) | F06 software（已实施）|
| 配置定时备份 (备份策略) | F06 backup（已实施）|
| 北向 OSS 接口 | F08 northbound |
| 设备自动开站 | F09 provision |

> 上述每一项都是独立功能域；本 PRD **仅在它们都执行完之后**做"按设备发起即时操作"和"打包多步操作"的事。

---

## 1. 业务背景（Why）

### 1.1 当前现状

**已有能力（盘点）**：
- 后端 `internal/ops/` 模块：`ops_templates` / `ops_tasks` / `ops_command_records` 三表 + CRUD service + 任务状态机 (`pending → running → success/failed/cancelled/paused`)
- 前端 `omcmb/webcode/src/pages/ops/` 五个 stub 页面：CommandManagement / Downloads / NetworkDiagnosis / TaskManagement / Templates
- 邻接已实现：MML 控制台 + 脚本任务 + 任务记录（`internal/mml/`）；统一任务队列 `internal/task/`；文件传输桥 `internal/transfer/`；配置模板 `internal/config/template/`；配置备份 `internal/backup/`

**缺口（gap）**：
- 五个前端页面均为 mock，未与后端打通
- 后端 service 仅有数据 CRUD，**无执行引擎**（创建 task 后没有调度器把模板 steps 真正发到设备）
- 无任何"立即命令"路径（前端发 → 后端转 ACS → 设备 → 回执 → 前端实时回显）
- 网络诊断完全没有：TR-069 Diagnostics 对象（IPPing / TraceRoute / Download / UploadDiagnostics 等）未接入
- 运维下载（设备 → OMC 的 Upload）只有 PM/MR/firmware 三类标准触发；**没有"按需取日志 / 诊断包"的入口**
- 无操作风险分级 / 审批流 / 维护窗口 / 紧急响应通道
- 与 MML 模块功能重叠：MML 是"按运营商规范文本协议向设备执行命令"，运维命令是"按 RPC 即时操作设备"，二者**应明确边界并互通**而非各自为政

### 1.2 不做的代价

- L1 运维必须依赖 OEM 售后远程登 SSH 才能取设备日志 / 抓包 / 测网速 → OEM 售后被无效问题淹没
- 告警事件复盘只能凭日志推测，无操作审计 → 等保 2.0 三级"重要操作可追溯"硬合规阻塞
- 批量处置（如"把华北 500 台基站全部 reboot"）只能靠脚本工程师写 MML → 高风险、易出错、无回滚
- 客户验收时被问"你们运维能不能远程做 ping 测试"答不上来 → 商机损失

### 1.3 为什么现在做

- F02 配置 / F04 告警 / F06 设备 / F06 软件升级 / F06 备份均已基本可用，运维管理是把它们**串起来给一线人员用**的最后一公里
- Wave 3 GA 准备：等保合规要求"重要操作可追溯"= 全量运维审计；客户运维 SOP（标准作业程序）= 模板化
- 现有 ops 模块已有数据模型骨架，避免推翻重做

---

## 2. 用户故事

| # | 角色 | 故事 |
|---|------|------|
| US-01 | NOC L1 | 我希望在告警详情页直接点"运维模板"按钮，选"GPS 失锁恢复"，对该设备一键执行三步处置脚本，**不必记 MML 命令** |
| US-02 | NOC L1 | 我希望对某台设备点"立即 reboot"，**1 秒内**得到确认并看到设备在 60 秒后 inform 上线的实时回执 |
| US-03 | NOC L2 | 我希望对华北 300 台基站 **批量执行** GPS 同步状态查询，结果按通过 / 失败分组导出 |
| US-04 | NOC L2 | 我希望对一台离线设备发起 Ping / TraceRoute 诊断（从最近的活着的同站基站发起），定位是核心网还是接入网问题 |
| US-05 | NOC L2 | 我希望按一个按钮就能从故障设备**收集**：当前配置 / 最近 1h 日志 / GPS 状态 / 邻区表 / 一份 PCAP，打成 zip 下载到我本地 |
| US-06 | FE 现场 | 我希望现场重启基站前先在 OMC 里 **抑制告警** 30 分钟（维护窗口），避免值班室收到大量虚假告警 |
| US-07 | OEM 售后 | 我希望对单台设备开 SSH-like 命令通道，逐条调 GPV/SPV 像 telnet 一样调试，**所有命令自动审计** |
| US-08 | SysAdmin | 我希望对"批量重启 > 50 台"或"factoryReset"这种高风险任务强制走**二人审批**，未审批的任务停在 pending |
| US-09 | SysAdmin | 我希望管理一个**运维模板库**，给 L1 工程师沉淀标准 SOP；每个模板有 owner、版本、变更记录 |
| US-10 | 审计员 | 我希望按"时间窗 / 操作者 / 设备 / 命令类型 / 结果"任意组合查询运维历史，导出 CSV 给季度审计 |
| US-11 | 任意 | 我希望系统每周自动跑一次"健康巡检"（对全网或某分组按预设诊断模板），结果汇总成报告邮件给我 |
| US-12 | NOC L1 | 我希望知道哪些常见告警的处置 SOP 已有现成模板，告警详情页直接列出推荐模板 |

---

## 3. 现状盘点（详细）

### 3.1 后端 internal/ops/ 现状

| 表 / 类型 | 状态 | 备注 |
|----------|------|------|
| `ops_templates` | DDL + CRUD ✓ | 字段：name / description / category / target_device_types(JSONB) / steps(JSONB) / estimated_duration / tags / use_count |
| `ops_tasks` | DDL + 状态机 ✓ | 字段：template_id / device_sns / status / current_step / progress / success_count / fail_count |
| `ops_command_records` | DDL + Create + List ✓ | 单命令执行记录 |
| 执行引擎 | ❌ | 没有调度器把 `steps` 发到设备，CreateTask 写完 DB 就结束 |
| 命令通道 | ❌ | 没有"前端发 → 即时回执"路径 |
| 诊断接入 | ❌ | TR-069 Diagnostics 对象（IPPing 等）未接入 |
| 运维下载（即时取包）| ❌ | 只能借助 transfer 模块手工触发 Upload，无 UX |

### 3.2 邻接模块（应消费而非重造）

| 模块 | 提供能力 | 运维管理如何用 |
|------|---------|--------------|
| `internal/mml/` | MML 控制台 + 脚本任务 + 任务记录 | 运维命令的一种实现通道（与 RPC 通道并列） |
| `internal/task/` | 统一任务队列（Redis + PG 双写）+ CWMP ID 映射 + completion router | 运维任务的底层执行队列，**不**自建队列 |
| `internal/transfer/` | 文件传输桥（Download / Upload 中介） | 运维下载（收日志 / 配置 / 诊断包）的下载层 |
| `internal/config/template/` | 配置模板 | 运维模板的"配置下发"步骤复用 |
| `internal/backup/` | 配置备份 | 模板可在执行前自动备份当前配置（rollback 兜底）|
| `internal/notification/` | 通知中心 | 任务完成 / 失败通知给操作者；高风险审批通知 |
| `internal/events/` | EventBus | 维护窗口期间抑制告警 |

### 3.3 前端 ops/* 页面盘点

| 页面 | 路径 | 现状 |
|------|------|------|
| CommandManagement | `pages/ops/CommandManagement` | mock 列表，无命令输入 |
| TaskManagement | `pages/ops/TaskManagement` | mock 列表，无创建入口 |
| Templates | `pages/ops/Templates` | mock 列表，无模板编辑器 |
| NetworkDiagnosis | `pages/ops/NetworkDiagnosis` | mock 单页 |
| Downloads | `pages/ops/Downloads` | mock 列表 |

---

## 4. 核心需求模块

### 4.1 运维模板（OpsTemplate）

**定位**：把一组多步运维动作沉淀为可复用的标准作业程序（SOP），由 SysAdmin 维护、L1 一键应用。

#### 4.1.1 模板组成

```
OpsTemplate
├── 元数据：name / description / category / tags / owner / version / change_log
├── 适用范围：target_device_types[] / target_carriers[] / min_sw_version
├── 风险等级：safe / cautious / dangerous  (决定审批策略)
├── 估时：estimated_duration_seconds
├── steps[]：
│   ├── step_id / name
│   ├── type：rpc_get_param | rpc_set_param | rpc_reboot | rpc_factory_reset |
│   │        rpc_download | rpc_upload | rpc_diagnostic |
│   │        mml_command | shell_command(reserved) |
│   │        wait / sleep / loop / branch
│   ├── params(JSONB)：根据 type 的参数定义
│   ├── timeout_seconds
│   ├── on_failure：abort | continue | retry(N) | rollback
│   ├── pre_check / post_check（可选断言）
└── rollback_steps[]（可选，整模板回滚动作）
```

#### 4.1.2 关键能力

- **目录化分类**：故障处置（gps_lost / 心跳异常 / 小区不可服务）/ 例行运维（重启 / 时间同步）/ 紧急响应（factory_reset / 隔离）/ 调试（取日志 / 抓包 / Ping）
- **变量化**：模板 steps 中支持 `${device_sn}` / `${user_input.xxx}` 等占位符，执行前由调用者填值
- **断言（pre_check）**：执行前查设备是否在线 / 是否有冲突任务 / license 是否够，不满足直接拒绝
- **版本管理**：模板可被改动，但既有 `ops_tasks` 仍引用旧版本快照（task 执行时把模板 snapshot 进 task 记录）
- **使用统计**：`use_count` / `last_used_at` / `success_rate`（最近 30 天）
- **导入导出**：JSON 格式互通，方便跨环境同步 / OEM 默认库下发

#### 4.1.3 内置模板（OEM 出厂建议）

| 类别 | 模板 | 步骤 |
|------|------|------|
| 故障 | GPS 失锁恢复 | 检查 GPS 卫星数 → 重启 GPS 模块 → 等待 60s → 再次检查 |
| 故障 | 小区不可服务（NR） | 取告警 → 取 RF 参数 → 软重启 SDR → 验证 |
| 例行 | 时间同步对齐 | 取当前时间 → 强制同步 → 验证 |
| 例行 | 计划重启 | 备份配置 → 重启 → 等待 inform → 校验状态 |
| 紧急 | 设备隔离 | 关闭小区 → 通知 EMS 摘流量 → 告警抑制 |
| 调试 | 现场协助包 | 取配置 + 取最近 1h 日志 + 取 GPS + 取邻区 + IPPing 测试 → 打 zip |

---

### 4.2 运维命令（即时命令 / Commands）

**定位**：单步、即时、有回执的"动手"操作。区别于"模板任务"是即时性 + 单步 + 命令视角。

#### 4.2.1 命令类型矩阵

| 类型 | 通道 | 典型动作 | 风险等级 |
|------|------|---------|---------|
| **RPC 即时** | TR-069 RPC | reboot / factory_reset / get_param / set_param / get_rpc_methods | L1 ~ L3 |
| **MML 命令** | MML 控制台（沿用 `internal/mml/`）| 运营商专有命令（如华为风格 LST / DSP / SET）| L1 ~ L3 |
| **诊断命令** | TR-069 Diagnostics | IPPingDiagnostics / TraceRouteDiagnostics（见 §4.4）| L1 |
| **文件命令** | Download / Upload | 下发配置 / 取配置 / 取日志（见 §4.5）| L1 ~ L2 |

#### 4.2.2 风险等级与对应行为

| 等级 | 定义 | 行为 |
|------|------|------|
| **L1 safe** | 只读 / 查询 / 单设备无副作用 | 直接执行，命中操作日志 |
| **L2 cautious** | 单设备写操作 / 重启 / 配置变更 | 二次确认（前端 Modal） + 操作日志 |
| **L3 dangerous** | factory_reset / 跨多设备批量写 / 关键参数 | 审批流（4 眼原则，需另一名 admin 同意） + 全量审计 |

#### 4.2.3 关键能力

- **即时回执**：前端发起 → 后端入队 → 即返回 task_id；前端通过 SSE / WebSocket 订阅完成回执（不是轮询）
- **实时输出**：MML / Diagnostic 命令支持流式输出（设备返多帧，前端逐帧显示）
- **批量发起**：选中设备列表（最多 N 台，N 受 license 限制）→ 同命令并发执行 → 表格化结果
- **超时与重试**：单命令默认 60s 超时；批量场景设备级独立超时，不互相阻塞
- **审计**：所有命令进 `ops_command_records` + `audit_logs`；含原始命令文本 / 输出 / 操作者 / 设备 / 时间戳 / 结果 / IP / UA

#### 4.2.4 UX 形态

- **快速操作菜单**（设备列表行内）：点设备 → "运维操作" 下拉 → reboot / 取日志 / Ping / ...
- **批量操作**（设备列表多选）：选 → "批量运维" → 选命令 → 预览影响面 → 确认 / 审批 → 执行
- **MML 控制台**：原 `internal/mml/console` 入口保留，深度调试用

---

### 4.3 运维任务（OpsTask / 异步任务编排）

**定位**：跨多步骤、跨多设备、长耗时的批量编排执行。是"运维命令"和"运维模板"的执行容器。

#### 4.3.1 任务生命周期

```
[CreateTask]
     ↓
   pending ──[已审批 / 无需审批]──→ running ──→ success
                                       │            │
                                       ├──失败──→ failed
                                       │            │
                                       ├──暂停──→ paused ──恢复──→ running
                                       └──取消──→ cancelled
```

- **审批门**：dangerous 等级任务在 `pending` 状态等审批；审批通过才 `running`
- **暂停 / 恢复**：长任务（500 台批量）可手工暂停，已完成设备不重做、未开始的设备等恢复
- **取消**：cancel 后已发出的 RPC 不能撤回，但后续设备不再下发

#### 4.3.2 任务调度引擎（关键新增）

后端需要一个 **TaskExecutor**，负责：

1. **并发控制**：单任务内对多设备并发数限制（如 max=20），避免 ACS 雪崩
2. **节流**：与 `internal/task/` 统一任务队列对接，nat-limit 控制每秒新增 RPC 数
3. **步骤路由**：根据 step.type 分发到 `acs/rpc/` 各 RPC handler
4. **结果聚合**：每个 (device, step) 的成功/失败 → 实时聚合到 `ops_tasks` 进度字段
5. **失败策略**：abort（整任务停） / continue（跳过失败设备继续）/ retry / rollback
6. **回滚执行**：触发 rollback_steps 走相反方向；rollback 失败需告警人工介入
7. **断点续传**：进程重启 / 节点切换后从 PG 状态恢复任务进度

#### 4.3.3 任务可见性

- **运行视图**：实时进度条 / 设备级状态表 / 当前 step / ETA / 已完成 vs 失败 vs 排队
- **历史视图**：可搜可过滤可分页；点入查详情
- **任务详情**：模板快照（任务创建时的步骤定义）+ 每设备执行流水（按 step → RPC 调用 → 返回）

#### 4.3.4 与 internal/task 关系

| 维度 | `internal/task/` 统一任务 | `internal/ops/` 运维任务 |
|------|--------------------------|------------------------|
| 抽象层 | 单条 RPC / 命令的"派发-执行-回执"原语 | 多步骤 + 多设备的业务编排 |
| 持久 | `device_tasks` 表 | `ops_tasks` + `ops_task_steps` 表 |
| 谁调谁 | 上层 | 下层：OpsTask 拆分后逐条调 task 服务派发 |

> 不重造任务队列，**运维任务编排器** = `internal/task/` 之上的应用层 orchestrator。

---

### 4.4 网络诊断（NetworkDiagnosis）

**定位**：定位"设备能不能通"、"路径在哪一段卡"、"上下行链路速率多少"的能力集合。

#### 4.4.1 TR-069 标准诊断对象（设备端）

直接驱动 TR-181 `Device.IP.Diagnostics.` 子树（厂商扩展可能在 FAPService.X_VENDOR.Diagnostics.）：

| 对象 | 用途 | 关键参数 |
|------|------|---------|
| **IPPingDiagnostics** | ICMP Ping 测试 | Host / NumberOfRepetitions / Timeout / DataBlockSize → AverageResponseTime / FailureCount |
| **TraceRouteDiagnostics** | 路径追踪 | Host / Timeout / DataBlockSize → RouteHops[] |
| **DownloadDiagnostics** | 测下行带宽 | DownloadURL / EthernetPriority → TotalBytesReceived / Throughput |
| **UploadDiagnostics** | 测上行带宽 | UploadURL / EthernetPriority → TotalBytesSent / Throughput |
| **UDPEchoDiagnostics** | UDP 回环 | Host / Port / NumberOfRepetitions → RTT |

执行流程：
1. OMC 通过 SPV 写 `DiagnosticsState=Requested`
2. 设备执行（耗时数秒到数十秒）
3. 设备 inform 时携带 `Diagnostics.X.DiagnosticsState=Complete` 事件码 8（DIAGNOSTICS COMPLETE）
4. OMC 通过 GPV 拉结果 → 解析 → 入库 → 推送前端

#### 4.4.2 OMC 侧诊断（上层）

不依赖设备执行，由 OMC 后端自己发起：

| 诊断 | 描述 |
|------|------|
| **可达性测试** | 从 OMC pod 对设备 IP 做 Ping / connect TCP 7547（ACS 端口）/ 连接请求测试 |
| **会话延迟** | Inform 到 InformResponse 的 RTT 历史曲线（来自现有 metric）|
| **同站连通性** | 选另一台同站基站对故障基站做 IPPing（间接定位）|
| **GPS 同步状态聚合** | 批量拉某分组 GPS 卫星数 / 锁定时长，找异常 |
| **邻区一致性** | 对比设备 A 的邻区表里 B、与 B 的邻区表里 A，找单边邻区配置 |
| **告警风暴检测** | 1 分钟内同设备 ≥ N 条告警 → 触发"告警风暴"告警 |

#### 4.4.3 诊断结果可视化

- **结果详情页**：原始 Ping/Trace 输出 + 时序图（多 hop / 多 packet）+ 失败原因总结
- **结果导出**：诊断包导出（PDF / JSON / 原始日志）
- **结果归档**：诊断记录保留 ≥ 6 个月（合规 + 复盘）；可关联到工单 / 告警 / 任务

#### 4.4.4 厂商扩展诊断

部分 OEM 设备有专有诊断（如 SAS 测试、CBSD 心跳测试、5G NSA 双连接测试），通过 `Carrier` 接口 + `DiagnosticsAdapter` 注册扩展点，避免硬编码。

---

### 4.5 运维下载（Downloads / Collections）

**定位**：把设备上的"东西"取到 OMC（或 OMC 再到运维人员的浏览器）。本质是 TR-069 Upload（设备发起到 OMC 接收）+ OMC 侧资源组织。

#### 4.5.1 可下载内容类型

| 类型 | 触发方式 | 文件位置 |
|------|---------|---------|
| **当前配置** | Upload FileType="1 Vendor Configuration File" | MinIO `device-configs/{sn}/{ts}.cfg` |
| **设备日志** | Upload FileType="2 Vendor Log File" 或厂商私有 | MinIO `device-logs/{sn}/{ts}.tar.gz` |
| **PM 文件 (按需)** | Upload FileType="3 Performance Counter File" | MinIO `pm-files/{sn}/{ts}.xml`（与 PM 模块共享）|
| **MR 文件 (按需)** | Upload FileType="4 Measurement Report" | MinIO `mr-files/{sn}/{ts}.xml`（与 MR 模块共享）|
| **诊断包 (zip)** | OMC 拼装：拉多个文件 + 当前状态打 zip | MinIO `device-bundles/{sn}/{ts}.zip` |
| **PCAP 抓包** | 厂商扩展 RPC | MinIO `device-pcaps/{sn}/{ts}.pcap` |
| **GPS 历史** | 从 PM 库导出 + 状态参数快照 | MinIO `device-gps/{sn}/{ts}.csv` |

#### 4.5.2 关键能力

- **按需采集**：UI 点"取日志"→ 后端调 transfer.Upload → 设备上传到 MinIO → OMC 回执 → 前端给下载链接
- **打包诊断包**：UI 选多个类型（配置+日志+GPS+邻区）→ OMC 并发取 → 取完 zip → 提供单一下载链接
- **下载历史**：列表展示该设备所有历史下载，按时间倒序 / 类型分组 / 大小 / 操作者
- **过期清理**：诊断包默认保留 90 天，到期自动归档到冷存储或删除（可配置）
- **断点续传**：大文件浏览器下载支持 Range 请求；后台 transfer 支持失败重传
- **签名 URL**：MinIO presigned URL（限时 1h），避免开放公网读权限

#### 4.5.3 UX 形态

- **设备详情 → 运维 → 下载**：单台设备的下载历史 + "取新"按钮
- **运维下载主页**：按设备分组 / 按时间排 / 按类型筛 / 全文搜
- **下载篮**：跨设备多选下载，浏览器并发下载或 OMC 端打 zip

---

## 5. 配套能力（推荐补充）

### 5.1 健康巡检（Health Inspection）

定时（每天 / 每周）对指定设备分组按预设诊断模板执行检查，结果汇总成报告。

| 元素 | 内容 |
|------|------|
| 检查项 | 在线率 / GPS 锁定率 / 告警数 / 心跳延迟 / 关键参数一致性 / 邻区一致性 |
| 调度 | cron 表达式（默认每周一 02:00） |
| 通知 | 完成后邮件 / SMS 给运维负责人 |
| 报告 | PDF / CSV，存 MinIO 保留 90 天 |
| 集成 | 用 `OpsTemplate` 类型 = "inspection"，复用模板能力 |

### 5.2 维护窗口（Maintenance Window）

为某设备 / 分组开"维护期"，期间告警抑制 + 自动开站暂停 + 任务可执行高风险动作。

| 字段 | 说明 |
|------|------|
| name / scope (single / group / all) | 范围 |
| start_at / end_at | 时间窗 |
| suppress_alarms | 是否抑制告警（默认 yes）|
| pause_provision | 暂停该范围自动开站（默认 yes）|
| allow_dangerous | 允许高风险任务（默认 yes）|
| creator / approver | 操作员 + 审批员 |

事件机制：开始时发 `maintenance.window.started`，结束发 `*.ended`，告警 / 开站 / 运维模块订阅。

### 5.3 运维审计与变更管理

**所有运维操作必须可追溯**。统一进 `ops_audit_logs` 表（独立于 `audit_logs` 高频写）：

| 字段 | 内容 |
|------|------|
| op_type | template_run / command_run / task_create / task_cancel / diagnostic / download / approval / ... |
| target | device_sn 或 group_id 或 task_id |
| operator | user_id |
| ip / ua | 来源 |
| input | 入参（去敏感）|
| output_summary | 简短结果（命令输出最多 4KB，超长存 MinIO 链接）|
| result | success / failed / cancelled / pending_approval |
| risk_level | L1 / L2 / L3 |
| approver | 审批者（若 L3）|
| created_at | |

保留期：**12 个月**（高于通用 6 个月，运维变更影响面大）。

### 5.4 批量操作 UX

- **设备选择器**：支持按 OUI / carrier / type / group / region / 在线状态 / 软件版本 等组合筛选
- **影响面预览**：在确认前显示"将影响 X 台设备，其中 Y 台离线（跳过），Z 台在维护窗口外"
- **二次确认**：高风险显示影响范围 + 二人输入密码确认
- **批次拆分**：用户可指定"分 5 批，每批 100 台，批间间隔 10 分钟"避免冲击
- **执行后导出**：所有设备结果一表 CSV 导出，方便交付给客户

### 5.5 运维知识库 / Playbook

把"告警 → 模板"的映射沉淀：

| 字段 | 说明 |
|------|------|
| alarm_pattern | 匹配规则（alarm_code / severity）|
| recommended_templates[] | 推荐模板 |
| docs | 富文本说明（成因 / 影响 / 注意事项）|
| tags | 关键词搜索 |
| success_rate | 该模板对该告警的历史处置成功率 |

告警详情页直接显示"推荐处置：模板 X、模板 Y" → 一键应用。

### 5.6 紧急响应（Break-glass）

故障紧急情况下绕过正常审批的临时高权限通道。

| 元素 | 内容 |
|------|------|
| 触发 | SysAdmin 申请"break-glass 30 分钟"，输入工单号 + 理由 |
| 效果 | 当前用户临时获得"无需审批执行 L3 操作"权限 |
| 全程录像 | 所有操作进 `ops_audit_logs.break_glass=true` 标记，事后 SysAdmin 必须 24h 内复盘评注 |
| 自动告警 | 进入 break-glass 立即发送"紧急权限激活"告警给安全负责人 |

### 5.7 远程会话 / 控制台（可选 V2）

- WebSocket-based 终端会话，OEM 售后 / L3 工程师对单台设备开命令行
- 全程录屏（命令 + 输出按时间戳记录）
- 支持多人 view-only 旁观（培训 / 客户协助）

---

## 6. 数据模型

### 6.1 既有表（小幅扩展）

```sql
-- 仅列出新增列，原表保留
ALTER TABLE ops_templates
    ADD COLUMN risk_level VARCHAR(16) NOT NULL DEFAULT 'safe',  -- safe/cautious/dangerous
    ADD COLUMN version INT NOT NULL DEFAULT 1,
    ADD COLUMN change_log JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN owner_user_id UUID REFERENCES users(id),
    ADD COLUMN rollback_steps JSONB,
    ADD COLUMN target_carriers JSONB NOT NULL DEFAULT '[]';

ALTER TABLE ops_tasks
    ADD COLUMN template_snapshot JSONB,  -- 任务创建时的模板快照（防模板后改影响审计）
    ADD COLUMN risk_level VARCHAR(16) NOT NULL DEFAULT 'safe',
    ADD COLUMN approval_state VARCHAR(16) NOT NULL DEFAULT 'not_required',
    -- not_required / pending / approved / rejected
    ADD COLUMN approver_user_id UUID REFERENCES users(id),
    ADD COLUMN approved_at TIMESTAMPTZ,
    ADD COLUMN batch_config JSONB,  -- {batch_size, interval_seconds}
    ADD COLUMN failure_policy VARCHAR(16) NOT NULL DEFAULT 'continue';
    -- continue / abort / retry
```

### 6.2 新建表

```sql
-- 任务步骤执行明细（按设备 × 步骤展开）
CREATE TABLE ops_task_executions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id         UUID NOT NULL REFERENCES ops_tasks(id) ON DELETE CASCADE,
    device_sn       VARCHAR(64) NOT NULL,
    step_index      INT NOT NULL,
    step_name       VARCHAR(200) NOT NULL,
    step_type       VARCHAR(32) NOT NULL,
    status          VARCHAR(16) NOT NULL,  -- pending/running/success/failed/skipped
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    duration_ms     INT,
    request         JSONB,
    response        JSONB,
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ops_task_executions_task    ON ops_task_executions(task_id);
CREATE INDEX idx_ops_task_executions_device  ON ops_task_executions(device_sn);
CREATE INDEX idx_ops_task_executions_status  ON ops_task_executions(status);

-- 网络诊断记录
CREATE TABLE ops_diagnostics (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_sn       VARCHAR(64),  -- NULL 表示 OMC 侧诊断
    diag_type       VARCHAR(32) NOT NULL,  -- ip_ping/tracerouter/download/upload/udp_echo/...
    initiator       VARCHAR(16) NOT NULL,  -- device/omc
    request         JSONB NOT NULL,  -- 入参
    result          JSONB,           -- 结果
    status          VARCHAR(16) NOT NULL,  -- pending/running/complete/failed/timeout
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ,
    duration_ms     INT,
    operator        VARCHAR(128),
    task_id         UUID REFERENCES ops_tasks(id) ON DELETE SET NULL,
    file_path       VARCHAR(500),  -- MinIO 路径（结果包）
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ops_diagnostics_device   ON ops_diagnostics(device_sn);
CREATE INDEX idx_ops_diagnostics_type     ON ops_diagnostics(diag_type);
CREATE INDEX idx_ops_diagnostics_started  ON ops_diagnostics(started_at DESC);

-- 运维下载记录
CREATE TABLE ops_downloads (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_sn       VARCHAR(64) NOT NULL,
    content_type    VARCHAR(32) NOT NULL,  -- config/log/pm/mr/diagnostic_bundle/pcap/gps/...
    file_path       VARCHAR(500) NOT NULL, -- MinIO 路径
    file_size       BIGINT NOT NULL,
    checksum        VARCHAR(64),
    status          VARCHAR(16) NOT NULL,  -- pending/uploading/complete/failed/expired
    operator        VARCHAR(128),
    task_id         UUID REFERENCES ops_tasks(id) ON DELETE SET NULL,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ops_downloads_device      ON ops_downloads(device_sn);
CREATE INDEX idx_ops_downloads_type        ON ops_downloads(content_type);
CREATE INDEX idx_ops_downloads_created     ON ops_downloads(created_at DESC);

-- 运维审计日志（独立高频写）
CREATE TABLE ops_audit_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    op_type         VARCHAR(32) NOT NULL,
    target_type     VARCHAR(16) NOT NULL,  -- device/group/task/template
    target_id       VARCHAR(128) NOT NULL,
    operator_user_id UUID REFERENCES users(id),
    operator_name   VARCHAR(128) NOT NULL,
    risk_level      VARCHAR(16) NOT NULL,
    input           JSONB,
    output_summary  TEXT,
    result          VARCHAR(16) NOT NULL,
    approver_user_id UUID REFERENCES users(id),
    break_glass     BOOLEAN NOT NULL DEFAULT FALSE,
    client_ip       INET,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ops_audit_logs_operator   ON ops_audit_logs(operator_user_id);
CREATE INDEX idx_ops_audit_logs_target     ON ops_audit_logs(target_type, target_id);
CREATE INDEX idx_ops_audit_logs_created    ON ops_audit_logs(created_at DESC);

-- 维护窗口
CREATE TABLE ops_maintenance_windows (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(200) NOT NULL,
    scope_type          VARCHAR(16) NOT NULL,  -- device/group/all
    scope_ids           JSONB NOT NULL DEFAULT '[]',
    start_at            TIMESTAMPTZ NOT NULL,
    end_at              TIMESTAMPTZ NOT NULL,
    suppress_alarms     BOOLEAN NOT NULL DEFAULT TRUE,
    pause_provision     BOOLEAN NOT NULL DEFAULT TRUE,
    allow_dangerous     BOOLEAN NOT NULL DEFAULT TRUE,
    reason              TEXT,
    creator_user_id     UUID REFERENCES users(id),
    approver_user_id    UUID REFERENCES users(id),
    status              VARCHAR(16) NOT NULL DEFAULT 'planned',
    -- planned / approved / active / ended / cancelled
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 运维知识库（Playbook）
CREATE TABLE ops_playbooks (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alarm_pattern         JSONB NOT NULL,  -- {alarm_code: [...], severity: [...]}
    recommended_templates JSONB NOT NULL DEFAULT '[]',  -- [template_id, ...]
    title                 VARCHAR(200) NOT NULL,
    docs                  TEXT,
    tags                  JSONB NOT NULL DEFAULT '[]',
    success_rate          NUMERIC(5,2),  -- 0.00-100.00
    use_count             INT NOT NULL DEFAULT 0,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## 7. API 设计（关键端点）

### 7.1 模板管理

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/ops/templates` | GET | 列表（分页 + 过滤）|
| `/api/v1/ops/templates` | POST | 创建（仅 SysAdmin）|
| `/api/v1/ops/templates/:id` | GET | 详情 + 历史版本 |
| `/api/v1/ops/templates/:id` | PUT | 更新（自动 version + 1，写 change_log）|
| `/api/v1/ops/templates/:id` | DELETE | 软删（保留引用任务）|
| `/api/v1/ops/templates/:id/duplicate` | POST | 复制为新模板 |
| `/api/v1/ops/templates/import` | POST | JSON 导入 |
| `/api/v1/ops/templates/:id/export` | GET | JSON 导出 |

### 7.2 命令执行

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/ops/commands/rpc` | POST | 即时 RPC（body: action / device_sn / params）|
| `/api/v1/ops/commands/mml` | POST | MML 命令（代理到 `internal/mml`）|
| `/api/v1/ops/commands/:id/stream` | GET (SSE) | 命令实时输出流 |
| `/api/v1/ops/commands/records` | GET | 历史记录（filter 多维）|

### 7.3 任务编排

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/ops/tasks` | GET / POST | 列表 / 创建 |
| `/api/v1/ops/tasks/:id` | GET | 详情（含进度 + 设备级状态）|
| `/api/v1/ops/tasks/:id/cancel` | POST | 取消 |
| `/api/v1/ops/tasks/:id/pause` | POST | 暂停 |
| `/api/v1/ops/tasks/:id/resume` | POST | 恢复 |
| `/api/v1/ops/tasks/:id/approve` | POST | 审批 |
| `/api/v1/ops/tasks/:id/rollback` | POST | 执行 rollback_steps |
| `/api/v1/ops/tasks/:id/executions` | GET | 设备 × 步骤明细 |
| `/api/v1/ops/tasks/:id/events` | GET (SSE) | 实时进度推送 |

### 7.4 网络诊断

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/ops/diagnostics/ping` | POST | IPPing |
| `/api/v1/ops/diagnostics/traceroute` | POST | TraceRoute |
| `/api/v1/ops/diagnostics/throughput` | POST | Download/Upload |
| `/api/v1/ops/diagnostics/reachability` | POST | OMC 可达性测试 |
| `/api/v1/ops/diagnostics/inspection` | POST | 健康巡检触发（同步 / 异步）|
| `/api/v1/ops/diagnostics` | GET | 历史记录 |
| `/api/v1/ops/diagnostics/:id` | GET | 详情 + 结果包下载链接 |

### 7.5 运维下载

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/ops/downloads/collect` | POST | 触发采集（body: device_sn / content_types[]）|
| `/api/v1/ops/downloads` | GET | 列表 |
| `/api/v1/ops/downloads/:id` | GET | 详情 + presigned URL |
| `/api/v1/ops/downloads/:id` | DELETE | 删除（仅本人 / SysAdmin）|
| `/api/v1/ops/downloads/bundle` | POST | 打包下载（多设备 / 多类型 → 一个 zip）|

### 7.6 维护窗口

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/ops/maintenance-windows` | GET / POST | 列表 / 创建 |
| `/api/v1/ops/maintenance-windows/:id` | GET / PUT / DELETE | |
| `/api/v1/ops/maintenance-windows/:id/approve` | POST | 审批 |
| `/api/v1/ops/maintenance-windows/active` | GET | 当前生效列表 |

### 7.7 审计与知识库

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/ops/audit-logs` | GET | 审计日志查询（多维过滤）|
| `/api/v1/ops/audit-logs/export` | GET | CSV / JSON 导出 |
| `/api/v1/ops/playbooks` | GET / POST | 知识库 |
| `/api/v1/ops/playbooks/match` | GET | 给一个 alarm，返回推荐模板 |

---

## 8. 权限模型

### 8.1 三级权限点（按动作粒度）

| 权限 key | 覆盖 | 默认角色 |
|---------|------|---------|
| `ops:template:view` | 查看模板 | viewer+ |
| `ops:template:write` | 编辑 / 创建 / 删除模板 | sys_admin |
| `ops:command:safe` | 执行 L1 命令（GPV / Ping）| operator+ |
| `ops:command:cautious` | 执行 L2 命令（reboot / SPV）| operator+ |
| `ops:command:dangerous` | 执行 L3 命令（factory_reset）| sys_admin + 二人审批 |
| `ops:task:view` | 查看任务 | viewer+ |
| `ops:task:create` | 创建任务 | operator+ |
| `ops:task:approve` | 审批高风险任务 | sys_admin |
| `ops:diagnostic:run` | 发起诊断 | operator+ |
| `ops:download:trigger` | 触发文件采集 | operator+ |
| `ops:download:view` | 查看下载历史 | viewer+ |
| `ops:maintenance:plan` | 建维护窗口 | operator+ |
| `ops:maintenance:approve` | 审批维护窗口 | sys_admin |
| `ops:audit:view` | 查审计日志 | auditor + sys_admin |
| `ops:break_glass` | 紧急权限激活 | sys_admin |

### 8.2 审批流（4 眼原则）

- L3 dangerous 命令 / 任务 / 维护窗口 → task 状态 = `pending_approval`
- 系统选出"非创建者"的 sys_admin（≥ 1 人）→ 推送审批通知
- 审批者打开审批页 → 看到完整任务详情（影响面、模板内容、回滚方案）→ 同意 / 拒绝
- 同意：task → `running`；拒绝：task → `cancelled`，原因写 audit_log

---

## 9. 运营商差异

| 维度 | CMCC | CTCC | CUCC |
|------|------|------|------|
| 命令通道偏好 | MML 优先（传统脚本运维）| RPC 优先 | RPC + MML 都可 |
| 诊断对象命名 | TR-181 标准 | TR-181 + 私有扩展 | TR-181 标准 |
| 日志路径约定 | `Vendor.Log` | `Device.X_CTCC_Log` | TR-181 标准 |
| 高风险审批 | 必须 | 推荐 | 推荐 |
| 模板审计保留 | 12 个月 | 12 个月 | 6 个月 |

差异通过 `Carrier` 接口的 `OpsAdapter` 扩展点封装：
- `OpsAdapter.GetLogPath() → string`
- `OpsAdapter.MapDiagnosticType(t) → string`
- `OpsAdapter.DefaultApprovalPolicy() → ApprovalPolicy`

禁止 `if carrier == "cmcc"` 硬编码。

---

## 10. 非目标

- ❌ **不替代 MML 模块**。MML 控制台 / 脚本任务保留独立入口，运维管理可以"调用"MML 命令但不重写它
- ❌ **不接管固件升级**（用 `internal/software/`）/ 配置定时备份（用 `internal/backup/`）/ 配置模板下发（用 `internal/config/template/`）
- ❌ **不实现 SSH 直连设备**（小基站不开 SSH，TR-069 才是协议）
- ❌ **不实现"机柜门禁 / 物理巡检"** 这类非数字化运维
- ❌ **首版不实现远程控制台**（5.7 可选 V2）
- ❌ **不接入故障工单系统**（OSS 集成在 F08 北向）

---

## 11. 验收标准（Given-When-Then 关键示例）

### V1 — 单命令即时回执
- **Given** 在线设备 SN=AB123
- **When** POST `/ops/commands/rpc` body={action:"reboot", device_sn:"AB123"}
- **Then** 返回 202 + task_id；SSE 流在 5 秒内推送 `command.dispatched` 事件；60 秒内推送 `command.completed{success:true}`
- **And** `ops_command_records` 新增 1 行，`ops_audit_logs` 新增 1 行

### V2 — 模板批量任务
- **Given** 模板 T1（3 步），目标 100 台设备
- **When** 创建任务，risk_level=cautious → 直接 running
- **Then** 任务进度条 0% → 100%，最终 success_count + fail_count = 100
- **And** 每设备 × 每步在 `ops_task_executions` 新增 1 行

### V3 — 高风险任务审批
- **Given** 模板 T2（含 factory_reset），risk_level=dangerous
- **When** operator A 创建任务
- **Then** 任务状态=`pending_approval`，不执行
- **And** sys_admin B 收到通知，审批同意 → 任务 → running
- **And** sys_admin A 自己审批自己 → 返回 403（4 眼原则）

### V4 — IPPing 诊断
- **Given** 设备 AB123 在线
- **When** POST `/ops/diagnostics/ping` body={device_sn:"AB123", host:"8.8.8.8", count:5}
- **Then** 后端 SPV `IPPingDiagnostics.*`；等设备 inform 事件码 8
- **And** 30 秒内得结果（5 次 ping，平均时延等）；`ops_diagnostics` 新增 1 行

### V5 — 一键诊断包
- **Given** 设备 AB123
- **When** POST `/ops/downloads/collect` body={device_sn:"AB123", content_types:["config","log","gps","neighbor"]}
- **Then** 后端并发触发 4 个 Upload；全部完成后打 zip
- **And** 返回 presigned URL；`ops_downloads` 新增 1 行，content_type=`diagnostic_bundle`

### V6 — 维护窗口抑制告警
- **Given** 维护窗口 W1 active，scope=[AB123]
- **When** AB123 在 W1 期间触发告警
- **Then** 告警入库但 `suppressed=true`；不进当前告警活跃表；不触发通知
- **And** W1 结束后新告警按正常流程处理

### V7 — 审计追溯
- **Given** 任意运维操作过去 12 个月
- **When** GET `/ops/audit-logs?operator=X&risk_level=L3`
- **Then** 返回该用户所有 L3 操作；可导出 CSV；含输入 / 输出 / 审批者

---

## 12. Gap 清单与实施路线图

### 12.1 Gap 严重性

| # | Gap | 严重性 | 拆分任务 |
|---|-----|-------|---------|
| 1 | OpsTask 无执行引擎（CreateTask 写完 DB 就停） | **P0** | T-0101 任务调度引擎 |
| 2 | 即时命令通道（POST → SSE 回执） | **P0** | T-0102 命令通道 |
| 3 | 5 个前端 stub 页面接 API | **P0** | T-0103 前端打通 |
| 4 | TR-069 Diagnostics 接入（IPPing / TraceRoute 等） | **P1** | T-0104 诊断子系统 |
| 5 | 运维下载（按需采集 + 打包） | **P1** | T-0105 下载子系统 |
| 6 | 风险分级 + 审批流 | **P1** | T-0106 审批流 |
| 7 | 维护窗口 + 告警抑制 | **P2** | T-0107 维护窗口 |
| 8 | 健康巡检（定时诊断 + 报告） | **P2** | T-0108 巡检 |
| 9 | 运维审计 12 个月归档 | **P2** | T-0109 审计归档 |
| 10 | 知识库 / Playbook | **P3** | T-0110 知识库 |
| 11 | 紧急响应（break-glass） | **P3** | T-0111 break-glass |
| 12 | 远程控制台 V2 | **P3** | 暂搁 V2 |

### 12.2 阶段路线（建议）

| 阶段 | 范围 | 工作量 | 价值 |
|------|------|-------|------|
| **W1 - MVP 命令通道** | T-0101 + T-0102 + T-0103（即时命令端到端打通）| ~10 工作日 | 一线 L1 能用即时命令了 |
| **W2 - 任务编排** | 任务调度引擎完整（含 pause/resume/rollback）+ 前端进度页 | ~8 工作日 | 批量操作能用 |
| **W3 - 诊断 + 下载** | T-0104 + T-0105 | ~10 工作日 | 故障定位能力补齐 |
| **W4 - 治理（审批 + 维护窗口 + 审计）** | T-0106 + T-0107 + T-0109 | ~8 工作日 | 合规与高风险防护到位 |
| **W5 - 智能（巡检 + 知识库）** | T-0108 + T-0110 | ~6 工作日 | 体验 + 客户满意度 |
| **W6 - 紧急响应** | T-0111 | ~3 工作日 | 等保高级要求 |

**总工作量**：~45 工作日（约 2 个月，单人）；可并行至 1.5 个月（2 人）。

---

## 13. 待决议（pending decisions）

| # | 议题 | 选项 | 默认（待决议）|
|---|------|------|--------------|
| Q1 | MML 模块与运维命令如何融合 | A. 合并成一个"命令中心" / B. 双入口共存（建议）/ C. 隐藏 MML 入口 | **B** |
| Q2 | 任务调度引擎放哪 | A. 复用 `internal/task/`+ 上层应用 orchestrator（建议）/ B. 独立队列 | **A** |
| Q3 | 高风险 4 眼审批是否首版必须 | A. 必须 / B. 首版仅记录 +警告 | **A**（合规阻塞）|
| Q4 | 诊断结果是否进入 KPI / MR 库 | A. 独立 `ops_diagnostics` 表 / B. 进 PM 库 | **A**（边界清）|
| Q5 | 模板的 `steps[]` schema 是否引入 JSON Schema 严格校验 | A. 引入 / B. 仅文档约定（首版）/ C. 用 protobuf | **B**（V2 再引）|
| Q6 | 跨 OMC 实例（多集群）共享模板 | A. 支持 / B. 不支持（首版）| **B** |

---

## 14. 关联文档与参考

- 后端模块：`omcgo/internal/ops/`（templates + tasks + command_records 已有）
- 邻接模块：`omcgo/internal/{mml,task,transfer,config/template,backup,software,notification}/`
- 前端页面：`omcmb/webcode/src/pages/ops/{CommandManagement,Downloads,NetworkDiagnosis,TaskManagement,Templates}/`
- TR-069 标准：TR-069 Amendment 6 + TR-181 Device:2 `Device.IP.Diagnostics`
- 运营商规范（如适用）：CMCC PCM 接口规范、CTCC 网管命令规范、CUCC 设备运维接口规范
- 同类系统参考：华为 iManager U2020 维护操作、爱立信 OSS-RC Operations、思科 EPNM Templates

---

## 变更日志

| 版本 | 日期 | 作者 | 备注 |
|------|------|------|------|
| 0.1 | 2026-05-09 | Claude | 初稿（调研稿）：5 主线 + 7 配套 + 数据模型 + API + 路线图 6 阶段 |
