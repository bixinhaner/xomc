# OMC 告警邮件与 Zed Mobile 状态汇总邮件设计（评审版）

> **2026-08-06 范围校正（当前有效）**
>
> 结合 RR #31002 及其子需求 #31315/#31316、老 OMC 现场页面和当前代码
> 复核，本阶段不再建设面向管理员的通用“规则 +
> 可编辑正文模板”平台。下文已经完成的告警生命周期、Outbox、逐收件人投递、重试、
> 熔断和审计能力继续保留；与本节冲突的模板管理、短信和跨业务扩展内容仅作为历史设计
> 记录，不进入本阶段交付。

> **评审规则**：本文件以下标记为“当前交付”的内容是本阶段开发和验收的唯一范围；
> 标记为“未来保留”的内容只用于解释内部扩展点，不得作为本阶段接口、页面、迁移、
> 测试或上线承诺。旧 PRD、旧 backlog 和旧需求单摘要必须以本文件的需求追踪矩阵为准。

## 当前交付边界

本阶段只交付告警邮件流程：

1. **告警邮件通知**
   - 业务入口位于告警管理，不在通用通知中心维护正文。
   - 配置项为告警筛选范围、设备/设备组、严重级别、启用状态、发送间隔、
     `Tolerance Duration`、收件人和是否包含默认收件人。
   - 标题固定为中文“告警通知”或英文“Alarm Notification”；不区分活动告警和清除告警，
     两类邮件都使用同一正文结构，并同时发送产生时间和清除时间字段。
   - 邮件在现有告警内容基础上增加告警描述和处理建议；处理建议来自告警定义中的
     `cnSuggestion/enSuggestion`。需求单没有要求把设备 SN、告警标识和状态拼入主题。

2. **Zed Mobile 2G/4G/5G 状态汇总邮件**
  - 本阶段排除，不作为当前需求验收范围。

共享能力只包括 SMTP 渠道配置、TLS、MIME 组装、逐收件人结果、错误分类、重试、熔断、
渠道健康、收件人保护和审计。

### 当前明确不做

- KPI 定时报表（#42492）和 CPE 列表邮件优化（#62360）不并入本阶段告警邮件范围；
  两者作为独立 PM/设备报表需求单对照验收，不得用 Zed 汇总需求替代；
- Zed Mobile #98018 及赞比亚相关需求不在本阶段；
- Office365 #72785 及 OAuth/Office365 专用兼容；
- SMS、Kafka 短信、直连短信和短信回执；
- 值班排班、升级链、quiet hours、跨业务通用通知规则；
- 供管理员编辑主题、HTML、正文和变量的外发模板中心；
- 扩展通用 `internal/report` 来承载 KPI 定时报表；
- 老 #62360 的 CPE 导出业务本身；只吸收其“批量取数、禁止逐设备查询”的经验。

### 页面职责

- `告警管理 → 告警通知设置`：告警筛选范围和邮件策略；
- `通知管理 → 状态汇总邮件`：本阶段不启用；如恢复 #98018，另立需求后再确认页面职责；
- `通知管理`：只保留消息、发送记录、失败/重试和渠道状态，不定义告警正文或统计口径。

告警邮件列表、详情和修改均按当前用户设备组与制式权限过滤；修改同时校验修改前、修改后
两个作用域。全局默认收件人只允许内置超管维护。一次告警邮件表单保存必须在同一数据库
事务内完成不可变版本、发布指针与启停指针更新。状态汇总邮件的收件人和启停权限必须遵循
同等级别的审计和权限规则。

## 需求追踪与冻结决策

需求单原文已通过登录后的 Redmine 页面核对。以下矩阵以原单标题、描述、自测结果和状态为
准；KPI、SMS 和通用模板内容不是这两个需求单的直接验收项。

| 来源 | 当前采用的需求结论 | 本阶段处理 | 开发前必须确认 |
| --- | --- | --- | --- |
| RR #31002 | 当前告警邮件缺少处理建议；邮件内容增加该字段及内容 | 纳入告警邮件 | 当前新架构告警字段与老邮件内容的映射 |
| 需求单 #31315 | 不区分活动/清除告警；邮件均发送产生时间和清除时间；标题为“告警通知/Alarm Notification” | 纳入固定后端正文 | 清除时间为空时的占位和告警描述字段 |
| 需求单 #31316 | 重复确认 #31315 的正文、标题和时间字段要求，已关闭且自测通过 | 作为 #31002 的同源验收证据 | 不另建第二套告警邮件逻辑 |
| 需求单 #98018 | Zed Mobile 自动汇总；2G/4G/5G 分开统计总数/在线/激活，排除 CPE，使用当地时间 | 本阶段排除 | 后续如重启该需求再单独确认 |
| #42492 / #62360 | KPI 定时报表相关摘要 | 不作为本次两个需求单验收项，另行评审 | 是否保留为独立 PM 需求 |
| Office365 #72785 | Office365/OAuth 专用兼容 | 本阶段排除 | 需求状态改为延期/替代，不得要求本阶段实现 |
| T-0007 / T-0014 | 旧邮件实现与短信凭据/短信通道任务 | 仅作历史追溯；当前不以旧 T-0007 证明 canonical 已生产验收，不实现 T-0014 | backlog 标注历史/延期，避免旧 AC 重新生效 |

### 当前冻结决策

1. **告警邮件主题**固定为中文“告警通知”或英文“Alarm Notification”，不拼接设备 SN、
  告警标识或状态；主题和正文均由后端内置渲染器生成。
2. **告警正文**不区分活动告警和清除告警，统一包含告警描述/具体问题、处理建议、产生时间
  和清除时间。清除时间为空时显示明确空值占位，不伪造清除时间；处理建议来自告警定义
  的 `cnSuggestion/enSuggestion`。
3. **告警范围策略**保留设备/设备组、严重级别、告警标识、制式和老系统间隔字段；
  `Tolerance Duration` 的精确语义仍需业务确认，未确认配置保持停用，不自动发送。
4. **状态汇总邮件**必须分别输出 2G、4G、5G 的总数、在线数、激活数，明确不统计 CPE；
  统计快照和邮件时间使用配置的本地业务时区。
5. **状态汇总自动发送**必须有持久化调度、逐收件人投递、重试、熔断和审计；收件人、周期、
  设备范围和启停权限沿用现有产品配置或在开发前由业务确认，不能借用 KPI 查询模板语义。
6. **启用门禁**：告警邮件和状态汇总邮件在 Worker、调度、逐收件人审计、权限校验和渠道
  健康链路未就绪时，API 必须拒绝 `enabled=true`；SMTP disabled 或必要依赖未注册时不得
  进入可工作启用态。
7. **邮件 accepted 语义**只表示 SMTP 服务端受理，不代表最终送达或阅读。当前阶段不实现
  SMS，因此不存在 handoff、delivered、回执超时等本阶段状态。

## 背景

当前 OMC 已具备告警、站内通知、邮件模板、邮件发送历史、Alertmanager webhook、
NATS JetStream 和多类事务 Outbox 基础，但邮件短信通知尚未形成运营商级闭环：

- 告警过滤器中的 `notify_email` 在告警持久化前同步发送邮件，失败只记日志。
- 告警模块和通知模块各维护一套 SMTP 配置与发送路径。
- `notification_history` 一条记录包含多个收件人，不能准确表达单个收件人的结果和重试。
- 模板渲染缺少严格变量校验，发送失败没有统一重试、死信和渠道熔断。
- SMS 只有数据模型，没有可工作的统一发送器和送达回执状态机。
- 告警事件直接发布 NATS，发布失败可能造成通知事件丢失。
- 前端通知设置仍有 Mock 页面，系统邮件和短信配置尚未形成完整业务入口。

本设计先吃透老 OMC 的实现和现场行为，再结合当前 Go、PostgreSQL、NATS、React
架构建设统一通知中心。第一阶段只覆盖告警产生、严重级别升级、确认停止提醒和告警
清除；后续可复用相同能力接入设备离线、任务失败、磁盘告警等系统事件。

## 设计依据

### 老 OMC 可确认实现

老项目 Java 和部分文档被 Esafenet 加密，无法直接证明全部后端方法体；本设计只采用
可读 JSP、现有说明文档和 2026-07-31、2026-08-04 老 OMC 13.0.5 现场页面共同证明的行为。

老系统的告警邮件链路为：

```text
告警视图模板
  → AlarmEmail Quartz 周期任务
  → AlarmEmailNoticeService
  → 查询模板、设备范围、告警和收件人
  → SendMailTool
  → alarm_email_record
```

现场确认：

- “告警模板”是告警范围与邮件策略的混合配置，不是单纯的消息正文模板。
- 模板同时配置网元类型、设备或设备组、告警定义、严重级别和邮件开关。
- `Interval` 支持实时、10、30、60 分钟。
- `Tolerance Duration` 支持实时、10、30、60 分钟。字段名、页面位置和调度方式表明它
  高概率用于告警持续时间门槛，但由于老后端方法体不可读，该语义仍需通过现场运行日志
  或业务方确认；迁移时不能把推断当成已验证事实。
- 收件人支持分号分隔的固定邮箱和系统默认收件人。
- 模板列表用邮件图标表示邮件已启用。
- 活动告警支持确认、取消确认、清除、标记已读，重复告警在同一 occurrence 上累加
  `Alarm Count` 并刷新更新时间；新实现必须保留这些生命周期语义。
- 现场存在“基站经纬度变化”类告警，证明电子围栏/地理安全结果可以进入网元告警域，
  但备份失败、通知渠道故障等没有网元身份的系统故障不能冒充基站告警。
- 发送历史字段包括邮件主题、收件人集合、发送时间、结果和失败原因。
- 发送结果只有成功、失败、部分成功的批次汇总，不能审计单个收件人。
- SMTP 配置是系统级单例，包含启用、发件邮箱、密码、Host、Port 和连接测试。
- 现场存在 SMTP 鉴权错误后持续重复失败的大量历史，缺少有效退避、熔断和渠道告警。
- 当前版本没有独立 SMS、短消息或 Kafka 配置页面。

老文档中的告警短信并非 OMC 直连短信供应商，而是：

```text
告警
  → Redis 队列 alarmInvokeKafkaQueue
  → 消费线程
  → Kafka
  → 外部短信或北向平台
```

因此 Kafka 确认只能证明 OMC 完成外部交接，不能证明短信到达手机。

### 行业语境

设计参考 3GPP TS 32.111-2 Alarm IRP 和 ITU-T X.733 的告警生命周期、严重级别、
可能原因、特定问题和关联语义：

- 3GPP TS 32.111-2：
  <https://portal.3gpp.org/desktopmodules/Specifications/SpecificationDetails.aspx?specificationId=1856>
- ITU-T X.733：<https://www.itu.int/rec/T-REC-X.733>

系统继续使用 Critical、Major、Minor、Warning、Cleared 语义，并把告警产生、升级、
确认、取消确认和清除视为同一告警 occurrence 的生命周期变化。

## 目标（2026-08-06 当前有效）

- 告警邮件的核心操作流程与老 OMC 一致：选择告警和网元范围、设置时间策略和收件人，
  后台生成固定格式邮件；界面和可靠性实现允许结合新架构优化。
- 已提交的告警事件不会因进程退出、NATS 中断或重复消费而丢失或重复发送。
- 告警处理与通知处理解耦，通知失败不阻塞告警入库、确认、清除和查询。
- 逐收件人记录 SMTP accepted、失败、重试和人工操作，不把 accepted 描述成最终送达。
- 支持固定邮箱和系统默认收件人，并按设备/设备组数据权限限制配置范围。
- Zed Mobile 汇总邮件复用设备状态和制式数据源，按一次快照生成固定正文；不得伪造 KPI
  查询模板或告警 occurrence。
- 保留版本、幂等、Outbox、熔断、审计等运营商级底座，但不把内部模型直接暴露为
  通用通知平台。

## 非目标

- 第一阶段不建设独立通知微服务。
- 第一阶段不承诺邮件最终阅读或最终投递；SMTP 成功只表示服务器受理。
- 本阶段不实现或展示短信渠道；历史短信设计不构成交付承诺。
- 第一阶段不建设设备组负责人或自动值班排班系统。
- 不把 `alarm_filters` 或内部通知规则模型暴露成跨业务通用编排器。
- 不让通知模块修改告警状态或成为告警事实源。
- 不新增 `000002+` 数据库迁移。
- 不在通用通知代码中散落运营商特例。
- 本阶段不把 KPI 查询模板定时报表并入 #98018；KPI 定时报表另行确认需求和统计口径。

## 选定架构

采用“统一异步通知中心 + 告警领域 Outbox + 当前 PostgreSQL/NATS 基础”的方案。

```text
TR-069/CWMP、系统任务或人工操作
  → AlarmEngine
  → 主 PostgreSQL 同一事务：
       新增/更新/删除活动告警
       写 alarm_event_outbox（完整生命周期快照）
  → Alarm Event Relay（生命周期事件唯一发布者）
  → NATS JetStream
       domain.alarm.lifecycle.raised / domain.alarm.lifecycle.updated
       domain.alarm.lifecycle.acknowledged / domain.alarm.lifecycle.unacknowledged
       domain.alarm.lifecycle.cleared
       ├─ Alarm History Projector → TimescaleDB alarms_history
       ├─ 北向接口
       └─ Notification Orchestrator
  → Notification Orchestrator：
       事件幂等入库
       occurrence 顺序投影
       规则匹配
       收件人与数据权限解析
       持续时间、抑制、聚合、恢复配对
       模板渲染
       创建逐收件人投递
  → Email Worker
  → SMTP accepted
  → 投递状态、尝试记录、渠道健康和指标

设备状态汇总独立链路：

```text
持久化汇总调度
  → 读取设备制式、在线状态、激活状态
  → 排除 CPE，按 2G/4G/5G 生成本地时区快照
  → 固定正文渲染
  → 创建逐收件人投递
  → Email Worker → SMTP accepted
```
```

主 PostgreSQL 中的告警变更和 `alarm_event_outbox` 是告警生命周期的原子事实；
TimescaleDB `alarms_history` 是可重放的幂等历史投影，不参与主库事务。PostgreSQL
通知表是通知事实与审计源，NATS JetStream 是可靠分发通道。告警模块只产生稳定领域
事件，不直接调用邮件或短信适配器。

当前清除流程先写 TimescaleDB、再删除主库活动告警，无法与 Outbox 组成跨库事务。
改造后，清除在主库单一事务中删除活动告警并写入包含完整清除快照的 Outbox；历史投影
失败时由 JetStream 重放恢复。不得使用分布式事务，也不得继续把两个独立数据库操作
描述成原子操作。

当前 `ALARM` JetStream 捕获 `alarm.>` 且使用 `WorkQueuePolicy`，只适合单一任务消费，
不能承载历史投影、北向和通知中心的独立 fan-out。新增 `DOMAIN_ALARM` 流捕获
`domain.alarm.>`，使用 `LimitsPolicy`、S2 压缩、默认 7 天 MaxAge 和可配置的硬字节上限；
每个消费者使用独立 durable。采用不同顶级前缀是为了避免与现有 `alarm.>` 流 Subject
重叠，也避免生产环境重建现有 ALARM 流。

`MaxAge=7d` 只是保留时间上界；达到 `MaxBytes` 时 JetStream 会更早淘汰消息。初始代码
默认值按 10 GiB 提供安全硬上限，但生产值必须依据现场峰值、平均载荷、消费者最长中断
窗口和磁盘预算计算后显式配置。上线门禁必须同时检查 stream bytes、最老消息、各 durable
lag 和提前淘汰风险，不能用固定字节数宣称一定保留 7 天。

Relay 默认关闭。上线时先创建 DOMAIN_ALARM 流，记录切换起始 stream sequence，再以该
sequence 创建历史投影、北向和通知 Shadow durable，最后启用 Relay。新增消费者不得依赖
“从当前尾部开始”的隐式默认；需要回放超过 7 天的事件时，从保留 30 天的 Outbox 按
`event_id` 或 sequence 范围受控重放。

不选择独立通知微服务，是因为当前系统仍是同一部署单元，拆服务会提前引入跨服务事务、
鉴权、运维和版本兼容成本。内部模块边界和事件契约保持独立，未来出现明确容量或团队
边界时可以平滑拆分。

## 组件边界

### AlarmEngine

- 继续负责告警产生、变更、确认、清除和持久化。
- 当前 `Alarm.ID` 直接作为稳定 `occurrence_id`，同一生命周期不再生成第二套标识。
- 在主 PostgreSQL 告警事务内写 `alarm_event_outbox`。
- 为同一 occurrence 的每次生命周期变化递增持久化 `alarm_version`。版本必须在持有活动行
  锁或通过期望版本校验的主库事务内分配，禁止 AlarmEngine 在事务外预计算版本。
- 清除 Outbox 保存归档所需的完整快照，使 TimescaleDB 投影不依赖已删除活动行。
- 新到达事件若被 `auto_clear` 直接清除，仍须在同一主库事务内形成 `raised v1`、
  `cleared v2` 两条有序事实，不能让 occurrence 从 cleared 开始。
- 单条、批量、自动确认、自动清除、同步对账和 Expedited Event 等所有活动告警写路径
  必须经过相同生命周期事务；REST Handler 不得再绕过 AlarmEngine 直接批量改表。
- 不解析通知规则，不读取收件人，不发送消息。
- 切换由 `alarm.lifecycle_mode=legacy|shadow|canonical` 控制，默认 `legacy`；只有 Relay、
  历史投影、北向和通知 Shadow 全部门禁通过后才能进入 `canonical`。

### Alarm Event Relay

- 使用 `FOR UPDATE SKIP LOCKED` 批量领取待发布事件。
- 发布稳定 `event_id` 和版本化事件载荷。
- NATS 发布失败进入退避，不能删除 Outbox 记录。
- 发布成功后记录时间；已发布记录默认保留 30 天，管理员可按 `event_id` 审计并重放，
  清理周期可配置。
- 是持久化后 `domain.alarm.lifecycle.*` 标准 Subject 的唯一发布者。
- 新 Subject 与现有载荷不统一的 `alarm.*` 隔离，支持 Shadow 并行验证；标准消费者
  切换完成后移除 AlarmEngine 中现有直接 `eventBus.Publish`。

### Alarm History Projector

- 持久订阅标准告警生命周期事件。
- 在 `domain.alarm.lifecycle.cleared` 时按 `alarm_id` 幂等写入 TimescaleDB
  `alarms_history`。
- 重复事件不得产生重复历史；暂时失败依赖 JetStream 重投。
- 投影延迟和失败需要独立指标与告警，不能阻塞主库告警清除。

### Notification Orchestrator

- 以 `event_id` 幂等接收事件。
- 按 `occurrence_id + alarm_version` 顺序推进 occurrence 投影。
- 匹配规则和不可变规则版本。
- 当前解析固定邮箱和默认告警邮件收件人；用户、角色和通用联系组为未来扩展。
- 执行设备数据范围求交集。
- 处理最小持续时间、摘要、恢复配对和告警生命周期幂等；重复提醒、quiet hours 和升级链
  不在本阶段管理面开放。
- 严格渲染模板并创建逐收件人、逐渠道的投递记录。
- 不直接执行外部网络调用。

### Zed Mobile Summary Scheduler

- 使用持久化调度和租约领取，不能使用 Web 进程内存 Cron。
- 按设备当前制式分为 `gsm/lte/nr` 三组，对应邮件中的 2G/4G/5G。
- 统计总数、在线数和激活数；CPE 在查询条件和聚合结果中均排除。
- 在快照中保存统计时间、IANA 时区和三制式计数，正文显示本地时间。
- 同一调度窗口、收件人和配置版本幂等，只创建一组逐收件人投递；失败进入统一重试和审计。
- 调度周期、设备范围、收件人和启停由业务配置冻结后才能开放，不能复用 KPI 查询模板。

### Channel Workers

- 当前只实现邮件窄适配器：领取投递任务、记录每次尝试、分类错误、安排重试并更新渠道健康。
- SMS Worker 和运营商特有字段映射属于未来独立需求，不在本阶段装配或运行。

### Receipt Processor

- 当前邮件没有供应商最终送达回执，因此本阶段不装配 Receipt Processor。
- 未来短信回执必须验证签名和时间窗口，并通过 `provider_message_id` 与内部投递 ID
  幂等更新结果；回调重复或乱序时不得用旧失败覆盖已送达状态。

## 告警事件契约

第一阶段消费：

- `domain.alarm.lifecycle.raised`
- `domain.alarm.lifecycle.updated`
- `domain.alarm.lifecycle.acknowledged`
- `domain.alarm.lifecycle.unacknowledged`
- `domain.alarm.lifecycle.cleared`

`domain.alarm.lifecycle.updated` 的 `change_mask` 至少区分严重级别变化和普通字段更新。
严重级别跨越规则阈值时触发升级通知，普通描述更新不重复发送首次通知。确认使用现有独立
`domain.alarm.lifecycle.acknowledged` Subject，用于停止后续提醒，不再伪装为 updated。
取消确认使用 `domain.alarm.lifecycle.unacknowledged`，用于在告警仍活跃且重复策略未到
上限时恢复未来提醒，不重新发送首次通知。

以上新 Subject 只允许发布 AlarmEngine 已持久化的标准生命周期事件。现有
`alarm.raised`、`alarm.updated`、`alarm.acknowledged`、`alarm.cleared` 视为 legacy
Subject，其载荷不统一，通知中心不得消费。备份失败、存储阈值、通知渠道故障等缺少
`device_id` 的系统事件保持在独立 system incident/health 域；在系统告警域建模完成前，
不得通过伪造设备身份塞入要求 `alarms_active.device_id NOT NULL` 的 AlarmEngine。命令事件
不是告警事实，通知中心和北向接口不得订阅。自定义载荷中的 `alert_email` 等收件人提示
不进入生命周期契约，收件人统一由通知规则解析。

事件载荷至少包括：

```text
schema_version
event_id
lifecycle_type
occurred_at
occurrence_id
alarm_id
alarm_version
device_id
device_sn
carrier
technology
ne_type
severity
previous_severity
alarm_identifier
alarm_name
alarm_event_type
probable_cause
specific_problem
alarm_source
status
raised_at
acknowledged_at
cleared_at
alarm_count
extensions
```

载荷使用显式、稳定的 `AlarmLifecycleSnapshot` 字段，禁止直接嵌套内部 `model.Alarm`；内部
模型字段增加、数据库类型变化或 JSON tag 调整不得静默改变领域契约。`change_mask` 使用
受控枚举，`previous_severity` 在严重级别变化时由事务锁定后的旧快照给出，`occurred_at`
使用数据库事务中的实际变更时间，而不是 builder 任意取当前时间。

载荷是事件发生时的不可变业务快照，通知中心不得在重放时用当前告警内容悄悄改写历史。
`extensions` 只允许从 `managed_object_instance`、`additional_information`、
`additional_text`、`notification_type` 等审核后的键构造，并设置单值和总长度上限；模板层
仍须再次按变量白名单选择和截断，不能把整个 AdditionalInformation 原样暴露给收件人。

`alarm_version` 从 1 开始，每次产生、属性变更、确认、取消确认和清除严格递增。通知
Inbox 先以 `event_id` 去重，再按 occurrence 应用版本：

- `version <= last_applied_version`：视为重复或旧事件，记录后忽略。
- `version == last_applied_version + 1`：应用并推进状态。
- `version > last_applied_version + 1`：进入等待，不跨版本执行通知策略；缺失事件到达后
  继续处理，超过等待上限则产生 system incident/health signal、指标和站内通知。

Worker 在外部调用前必须重新检查 occurrence 当前状态、版本和 schedule generation。
已确认或已清除后领取到的旧首次通知和重复提醒必须取消；已经进入外部调用的请求则按
真实服务商结果结束，不能伪装成取消。

## 告警邮件策略

本阶段不向管理员暴露通用通知规则。告警邮件设置是告警域的专用业务适配器，内部仍可
使用不可变规则版本、Occurrence 投影和投递幂等能力，但不能把这些内部模型当成跨业务
规则平台。

### 当前匹配条件

- 告警标识。
- 严重级别。
- 设备或设备组。
- 制式。

同一字段内多个值为 OR，不同字段之间为 AND；空字段表示不限。事件类型固定由后端根据
生命周期处理，当前只交付产生、严重级别升级和清除邮件。确认/取消确认只用于停止或恢复
内部后续调度，不单独生成可编辑通知规则。

### 当前发送策略

- `Tolerance Duration` 的候选实现是最小持续时间门槛；业务未确认前，迁移配置不得自动启用。
- `Interval` 只允许实时、10、30、60 分钟；它究竟表示摘要窗口还是老系统调度间隔，必须
  在需求评审中冻结。实现不得用字段名自行推断业务语义。
- 同一 occurrence 的重复上报不重复创建首次邮件。
- 清除邮件只与已经成功受理的产生/升级邮件配对；被抑制或最终失败的首次邮件不生成孤立
  恢复邮件。
- 当前不交付 quiet hours、值班排班、升级链、重复提醒配置和跨业务维护窗口配置。
- 所有事件、调度和投递时间以 UTC 持久化；KPI 之外的显示时区使用部署级业务时区。

### 冲突与幂等

- 同一事件、用途、序号、渠道和收件人只创建一条投递。
- 规则版本、模板版本和收件人地址在投递创建时快照，后续配置修改不改写历史。
- 规则命中、权限排除、抑制、聚合和恢复配对均写入可审计结果。

## 收件人与权限

当前产品入口只支持固定邮箱和系统保留的默认告警邮件收件人。指定用户、指定角色及通用
联系组只作为内部模型的未来扩展，不在本阶段页面或 API 中配置。

默认告警邮件收件人只包含固定邮箱，用于承接老系统“Notify the Default recipients”业务；
不能通过默认收件人绕过设备范围和制式权限。

用户和角色在事件发生时解析并快照：

- 只包含启用状态的用户。
- 渠道地址不能为空且必须通过格式校验。
- 告警设备必须同时满足用户角色授予的设备组和制式权限。实现统一调用
  `PermissionService.GetUserVisibleDeviceGrants`，不能只判断设备组。
- `carrier` 只作为规则匹配和告警快照属性，不作为用户身份、租户或数据权限来源。
- 老 OMC 的 `operator_code` 不能直接等价为当前 `carrier`。邮件试点前必须由业务确认当前
  项目的运营商隔离是否已完全由设备组和制式 grant 承载；如仍需要独立租户边界，应在
  统一 PermissionService 中建模和强制执行，禁止在通知规则里用 `carrier` 条件代替授权。
- 同一地址只保留一次。
- 后续用户联系方式变化不改写既有历史。

创建或编辑规则的操作人只能选择自己有权管理的设备范围。投递历史查询继续按告警设备
数据范围过滤；邮箱和手机号默认脱敏，完整地址需要独立权限。

## 固定邮件正文与渠道语义

### 固定后端模板

本阶段不提供管理员编辑主题、HTML、正文或变量的页面和 API。邮件模板由后端维护，按
产生/升级/清除场景和中英文版本固定渲染；投递记录保留实际内置模板版本，保证历史可解释。

允许字段只来自版本化白名单：告警名称、告警标识、严重级别、网元类型、设备 SN、告警状态、
产生时间、清除时间、可能原因、具体问题和处理建议。不得把内部模型、任意数据库字段、
`additional_information` 原文或秘密传给模板。

严格渲染规则如下：

- 缺失字段显示明确空值占位，不使用静默的 `missingkey=zero` 误导收件人。
- `Raised` 邮件的清除时间为空；`Cleared` 邮件必须带清除时间。
- 中文使用 `cnName/cnProbableCause/cnSuggestion`，英文使用对应英文定义；缺失建议时
  显示空值占位并记录模板数据缺失指标。
- 每个收件人独立 SMTP envelope，不在 To/Cc 中泄露其他收件人。

### Email SMTP

- SMTP 成功响应后记录 `accepted`，不声称最终送达或阅读。
- SMTP 鉴权失败、证书错误和明确配置错误打开渠道熔断，并产生可审计的渠道健康事件。
- 连接验证只验证 SMTP 握手，不发送测试邮件；生产测试发送不属于本阶段能力。

### 未来保留：SMS

SMS Kafka、直连 SMS、回执、`handoff_only`、`delivered` 和 `unknown` 只保留为未来扩展点，
本阶段不建 Worker、不开放配置、不注册路由、不展示页面、不写短信验收项。未来重新立项时
必须单独冻结供应商、合规模板、Broker/HTTP 协议、鉴权、回执、幂等键、SLA 和测试环境。

## 数据模型

所有新增结构折回 `omcgo/migrations/000001_init_schema.sql`，不创建 `000002+`。

本节数据模型是内部执行和审计模型，不等于产品管理面。当前只创建和使用邮件、告警邮件
设置、KPI 报表运行、渠道健康、投递和尝试相关数据；SMS 字段、回执状态、quiet-hours
调度类型和通用模板版本属于未来保留结构，不能在本阶段被 API 或 Worker 激活。

### `alarm_event_outbox`

主 PostgreSQL 告警事务内写入的领域事件：

- `id/event_id`
- `aggregate_type`
- `aggregate_id/occurrence_id`
- `aggregate_version`
- `event_type`
- `payload`
- `status`
- `attempt_count`
- `next_attempt_at`
- `locked_at`
- `published_at`
- `last_error`
- `created_at`

`event_id` 唯一，`(aggregate_id, aggregate_version)` 也必须唯一，错误信息必须脱敏。

该表属于告警领域，北向接口、历史投影和通知中心消费同一标准事件。它不是通知专属
队列。`domain.alarm.lifecycle.*` 不得存在第二个直接发布者；Shadow 期间可以与不同
命名空间的 legacy `alarm.*` 并存，但消费者不得同时把两者当成同一事实处理。

### `notification_events`

通知中心事件 Inbox 与不可变快照：

- `event_id` 唯一。
- `event_type`、`occurrence_id`、`occurred_at`。
- 版本化标准载荷。
- 首次接收和处理完成时间。

### `notification_occurrences`

保存通知中心对告警生命周期的顺序投影：

- `occurrence_id` 主键，同时等于当前 `alarm_id`。
- `last_applied_version`。
- 当前 `schedule_generation`。
- 当前状态、严重级别和关键生命周期时间。
- 最近事件 ID 和更新时间。
- 版本缺口状态及首次发现时间。

该表只用于通知编排和发送前栅栏，不成为告警事实源。

### `notification_rules` 与 `notification_rule_versions`

`notification_rules` 保存稳定身份、名称、revision、当前草稿版本、当前已发布版本、
当前已启用版本、优先级和归档状态。
`notification_rule_versions` 保存不可变的匹配条件和策略快照，并记录创建人、创建时间和
变更原因。

### `notification_rule_recipients`

保存规则版本的目标：

- `target_type=fixed_contact|user|role|contact_group`
- `target_id`
- 固定联系人的加密地址引用
- 渠道限制

### `notification_rule_channels`

保存规则版本的渠道策略：

- 渠道类型和渠道配置 ID。
- Raised、Escalated、Cleared 模板版本。
- 最小持续时间、摘要、重复、恢复、安静时段和限流策略。

### `notification_contact_groups` 与成员表

管理默认 NOC 联系组和其他静态联系组。成员仍使用固定联系人、用户或角色引用，联系组
本身不绕过数据权限。

### `notification_templates` 与 `notification_template_versions`

稳定模板身份和不可变版本。投递必须关联具体版本，历史不受后续编辑影响。

### `notification_channel_configs`

保存渠道类型、名称、启用状态、非敏感参数和秘密引用。SMTP 密码、短信密钥、证书私钥
不得明文进入业务表或查询响应。

第一阶段只允许一个默认邮件渠道处于启用状态，避免无业务需求的多 SMTP 复杂度；数据
模型保留未来多渠道能力。

### `notification_channel_health`

保存运行状态：

- `circuit_state=closed|open|half_open`
- 连续成功和失败次数。
- 最近成功、失败和验证时间。
- 最近错误类别和脱敏摘要。
- 熔断开始、下次探测时间。

### `notification_deliveries`

每个收件人、每个渠道一条：

- `event_id`
- `occurrence_id`
- `rule_version_id`
- `template_version_id`
- `channel_config_id`
- `dispatch_kind=initial|escalation|repeat|recovery|digest`
- `sequence_no`
- `recipient_type`
- 加密地址快照
- HMAC 地址指纹
- 流程状态和投递结果
- `available_at`、`next_attempt_at`
- `locked_by`、`locked_at`、`lease_expires_at`
- 创建时的 occurrence 版本和 schedule generation
- `provider_message_id`
- 原产生投递 ID，用于恢复配对
- 各阶段时间
- 抑制或失败原因码

唯一约束：

```text
event_id + dispatch_kind + sequence_no + channel + recipient_fingerprint
```

领取使用 `FOR UPDATE SKIP LOCKED` 和有期限租约。Worker 崩溃后只有租约到期的任务可被
重新领取；外部请求必须携带稳定幂等键。重试只修改 `next_attempt_at`，不创建新的业务
投递。

### `notification_delivery_attempts`

每次外部调用一条：

- 尝试序号。
- 开始和结束时间。
- 结果、错误类别、HTTP/SMTP/Kafka 安全状态摘要。
- 服务商请求 ID。
- 下次重试时间。

不得保存密码、Token、完整服务商敏感响应或未脱敏消息载荷。

### `notification_aggregation_buckets`

按规则、渠道、收件人、设备范围、严重级别和时间窗口聚合事件，并记录窗口状态、
包含事件数和最终投递。

### `notification_schedules`

保存可重启、可取消的生命周期定时任务：

- `occurrence_id`、`rule_version_id`、渠道和收件人指纹。
- 当前 `schedule_kind` 只使用 `initial_gate|digest_flush`；`repeat|quiet_hours_release` 为未来
  保留值，本阶段不由 API 创建或由 Worker 领取。
- `sequence_no`、`due_at`。
- `generation`、`state=pending|claimed|completed|cancelled`。
- `locked_by`、`locked_at`、`lease_expires_at`、`cancelled_at`。
- 创建事件版本和完成后产生的 delivery ID。

唯一键至少包含 occurrence、规则版本、渠道、收件人、kind、sequence 和 generation。
确认、清除或规则切换通过递增 generation 并取消旧任务；领取后、外呼前仍必须执行
occurrence 状态与 generation 栅栏检查。

### 既有表处理

- `notifications` 继续作为站内通知，不与外部渠道投递混用。
- 现有 `notification_history` 在代码切换后改为 legacy 只读来源或从基线移除。
- 新历史接口查询 `notification_deliveries` 和 `notification_delivery_attempts`。
- 老批次历史不得伪造成不存在的逐收件人结果。

## 状态模型

使用两个维度，避免把流程进度和业务结果混在一个状态中。

流程状态：

```text
queued
  → sending
  → retry_wait → queued
  → awaiting_receipt
  → completed
  → dead_letter

queued → suppressed
queued/retry_wait → cancelled
```

投递结果：

```text
none
accepted
delivered
failed
unknown
handoff_only
```

典型组合：

- 邮件 SMTP 成功：`completed + accepted`
- Kafka Ack：`completed + handoff_only`
- 直连短信受理：`awaiting_receipt + accepted`
- 短信成功回执：`completed + delivered`
- 回执超时：`completed + unknown`
- 永久失败或重试耗尽：`dead_letter + failed`

老系统的成功、失败、部分成功在 API 层按一组逐收件人投递动态汇总，不再作为事实状态
直接存储。

## 告警生命周期策略

### 首次产生

- 创建稳定 occurrence。
- 创建持久化 `initial_gate` schedule，满足最小持续时间后才创建首次投递。
- 在门槛到达前清除的瞬时告警记录为 suppressed，不发送产生或清除消息。

### 重复上报

- 同一 occurrence 的普通重复上报只更新告警计数，不重复创建首次投递。
- 规则明确配置周期提醒时，使用 `sequence_no` 创建受限提醒。

### 严重级别升级

- 只有跨越规则阈值或进入更高严重级别策略时发送升级通知。
- 降级默认不发送，可由未来规则显式开启。

### 确认

- 确认事件按版本推进 occurrence，递增 generation，并停止尚未发送的后续提醒。
- 已进入外部调用的投递不能伪装取消；按真实结果结束。

### 取消确认

- 取消确认事件按版本推进 occurrence 并递增 generation，不重新发送首次通知。
- 告警仍活跃、规则允许重复且未超过最大次数或最长提醒时间时，从
  `max(now, last_accepted_at + repeat_interval)` 安排下一次提醒，sequence 延续原值。

### 清除

- 清除事件按版本推进 occurrence，递增 generation，并取消尚未发送的首次门槛任务和
  重复提醒。
- 只给已经成功受理产生或升级通知的“收件人 + 渠道”组合发送恢复。
- 原产生通知被抑制或最终失败时，不发送容易误解的孤立恢复消息。
- 摘要窗口内产生后又清除的低级别告警可以在摘要中表现为“窗口内发生并恢复”，不发送
  两条独立消息。

## 告警风暴控制

采用四层控制：

1. occurrence 生命周期去重。
2. 最小持续时间、抖动窗口和恢复迟滞。
3. Minor/Warning 按规则、设备组、渠道和收件人聚合。
4. 渠道、规则和收件人限流与配额。

Critical 默认绕过普通摘要，但仍受幂等、收件人级速率限制、最大提醒次数和最长提醒
时间约束。超限事件不能静默丢弃，必须记录 suppressed 或进入摘要。

## 重试、熔断与死信

错误分类：

- 临时错误：网络超时、限流、服务商 5xx，指数退避并加入随机抖动。
- 永久错误：地址格式错误、短信模板拒绝，不重试。
- 配置错误：SMTP 鉴权、证书、Kafka 配置错误，打开渠道熔断。
- 未知结果：请求超时但可能已受理，等待回执或查询状态，不盲目重发。

渠道熔断时，新任务保留为可审计的等待或失败状态，不能每几分钟重复制造相同错误。
配置修复并验证成功后进入 half-open 探测，再恢复 closed。

通知渠道故障产生 OMC 内部 `NOTIFICATION_CHANNEL_UNAVAILABLE` system incident/health
event、指标和站内通知，不伪造成带设备身份的网元告警。若通过其他健康渠道或北向处理，
必须设置 `origin=notification` 递归保护：故障渠道不能通知自己的故障。

重试耗尽进入死信。具有权限的管理员在修复配置后可以单条或批量重试，操作原因写入审计。

## API 设计

本阶段只开放业务专用 API；通用 `notification-rules`、`notification-templates` 和联系组
管理 API 不注册到主路由。所有写接口必须带 `If-Match`，缺少返回 428，版本不匹配返回 412。

### 告警邮件设置

- `GET /api/v1/alarm-email-settings`
- `GET /api/v1/alarm-email-settings/{id}`
- `POST /api/v1/alarm-email-settings`
- `PATCH /api/v1/alarm-email-settings/{id}`
- `DELETE /api/v1/alarm-email-settings/{id}`：归档并停用，不物理删除历史
- `GET /api/v1/alarm-email-settings/default-recipients`
- `PATCH /api/v1/alarm-email-settings/default-recipients`

告警邮件设置 API 只接受告警范围、级别、设备/设备组、制式、时间策略、固定收件人和是否
包含默认收件人；不接受正文、模板、渠道或任意事件表达式。列表、详情、修改和归档按当前
用户设备组与制式权限过滤；修改同时校验旧范围和新范围。默认收件人仅内置超管可读写。

一次保存必须在同一事务内完成不可变版本、发布指针和启停指针更新。`enabled=true` 还必须
通过当前启用门禁；链路未就绪时返回明确冲突错误，不允许页面通过开关绕过。

### Zed Mobile 状态汇总邮件

本阶段提供专用配置 API，默认配置为 disabled；周期、收件人和设备范围未从老 OMC 页面
确认前，不开放普通用户范围配置。执行记录使用 `run_key + 配置时区 + 本地日期` 幂等，
快照保存 2G/4G/5G 总数、在线数、激活数、排除 CPE 数和本地时区。

当前 API 仅内置超管可用，不复用 KPI 查询模板：

- `GET/POST /api/v1/notification/status-summary-settings`
- `GET/PATCH /api/v1/notification/status-summary-settings/{id}`
- `POST /api/v1/notification/status-summary-settings/{id}/enable`
- `POST /api/v1/notification/status-summary-settings/{id}/disable`

### KPI 定时报表（暂缓）

`#42492/#62360` 的 KPI 定时报表代码和页面暂不作为本阶段需求验收项；保留现有代码但不扩展，
待独立 PM 需求确认统计窗口、权限和导出链路后再决定是否启用。

### 通知管理与审计

- `GET /api/v1/notification-channels`：只读渠道元数据和启用状态
- `POST /api/v1/notification-channels/{id}/verify`：只做 SMTP 握手
- `GET /api/v1/notification-channels/{id}/health`
- `GET /api/v1/notification-deliveries`
- `GET /api/v1/notification-deliveries/{id}`
- `GET /api/v1/notification-deliveries/{id}/attempts`
- `POST /api/v1/notification-deliveries/{id}/retry`
- `POST /api/v1/notification-deliveries/retry`

渠道查询永不返回秘密；当前不开放在线 SMTP 参数编辑和测试邮件发送。投递查询只返回当前
用户有权查看的设备范围和脱敏地址；人工重试必须要求原因、重新校验权限并记录审计。

## 前端设计

V1 只维护以下入口：

1. `告警管理 → 告警邮件设置`：配置告警范围、级别、设备/设备组、制式、时间策略和收件人。
2. `通知管理 → 状态汇总邮件`：确认老 OMC 配置口径后配置 Zed Mobile 周期、范围和收件人。
3. `通知管理 → 渠道健康/投递记录`：查看邮件渠道状态、逐收件人结果、失败原因和重试。

不提供通用规则、模板、联系组、SMS 或在线 SMTP Secret 编辑页面。告警邮件页面只展示
“固定后端正文”提示，不出现主题/HTML/变量编辑控件。状态汇总页面必须显示实际统计时间、
2G/4G/5G 三组计数和排除 CPE 数；未确认配置时显示不可启用状态。

所有启用控件都必须反映服务端门禁：链路未就绪时显示不可启用状态和原因，不能只依赖前端
隐藏。所有表单、toast、Modal、空态、错误信息和渠道状态文本走 i18n；页面不显示完整
邮箱、手机号、Secret 或内部加密字段。

告警详情可以增加只读通知轨迹，展示规则版本、脱敏收件人、渠道状态、抑制/聚合原因、
投递尝试和恢复配对，但不在告警详情内编辑通知配置。

现有 Mock 的 NotificationSettings 改为真实页面；模板和历史页面复用现有路由与组件
模式。所有用户可见文本走 i18n，所有表单、Modal、toast、空态和请求参数必须用真实
浏览器验证。

## 权限与审计

权限至少拆分为：

- 查看、编辑通知规则。
- 启停和归档规则。
- 管理模板。
- 管理默认联系组。
- 管理渠道配置。
- 连接验证和测试发送。
- 查看投递记录。
- 查看完整联系地址。
- 人工重试死信。

规则修改、启停、归档、渠道验证、测试发送、秘密引用变更和人工重试全部写操作日志。
审计保存操作人、时间、对象、变更前后摘要、原因和请求关联 ID，敏感字段只记录“已变更”。

## 安全

- 渠道秘密使用现有安全配置能力或外部 Secret 引用，不明文存储。
- 固定联系人和投递地址快照加密存储。
- 使用服务端 HMAC 指纹做地址去重，不能用可逆明文哈希。
- API 和日志默认脱敏邮箱、手机号和服务商标识。
- 短信回调必须验证签名、防重放并限制来源。
- HTML 邮件模板采用允许列表清洗，禁止任意脚本、远程跟踪内容和未审核 HTML。
- 投递和尝试数据按可配置保留期归档或清理；审计保留期服从运营商要求。

## 可观测性

指标至少包括：

- Outbox 待发布数量和最老事件延迟。
- 事件处理延迟和规则匹配数量。
- 各渠道 accepted、delivered、handoff、failed、unknown 数量。
- 重试、死信、抑制、聚合和限流数量。
- 短信回执等待数量和延迟。
- 渠道熔断次数、持续时间和恢复结果。

全链路关联：

```text
occurrence_id
  → event_id
  → rule_version_id
  → delivery_id
  → attempt_id
  → provider_message_id
```

结构化日志包含上述 ID、渠道、错误类别和耗时，不包含完整地址、秘密或原始敏感响应。

## 迁移

### 当前项目迁移

- 将 `alarm_filters.notify_email` 和 `email_recipients` 转换成默认禁用的独立通知规则，
  先输出命中范围和收件人预览。
- 迁移工具必须按当前优先级和首条命中语义扫描重叠规则，列出原 `notify_email` 下方可能
  被释放的 `ignore`、`auto_ack`、`auto_clear` 等规则；存在冲突时禁止自动切换。
- Shadow 阶段保留原 `notify_email` 行为。按设备范围切换新通知规则后，原位置保留不
  发送邮件的兼容 barrier，继续阻止低优先级规则意外生效。只有业务方处理完冲突并确认
  行为一致后，才能移除 barrier，让 `alarm_filters` 恢复告警处理职责。
- `notify_webhook` 不在第一阶段自动迁移；未来迁移必须复用同样的首条命中兼容流程。
- 告警模块和通知模块的 SMTP 配置收敛为一个默认邮件渠道。
- 现有模板转换成带不可变版本的模板。
- 现有通知历史标记为 legacy，只读展示原始批次语义。
- 删除告警持久化前的同步邮件调用。
- 北向接口和其他标准消费者切换到 `domain.alarm.lifecycle.*` 后，删除 AlarmEngine 对
  legacy
  `alarm.raised`、`alarm.updated`、`alarm.acknowledged`、`alarm.cleared` 的直接发布。
- 备份和其他系统模块的自定义 `alarm.*` 保持在 system incident/health 边界并列入后续
  专项迁移；本轮不得迁入要求设备身份的 AlarmEngine，canonical 消费者也不得订阅。

### 老 OMC 配置迁移

如需迁移老系统配置：

- 老告警视图模板转换为通知规则。
- 网元、设备组、告警列表、严重级别映射为匹配条件。
- Interval 映射实时或摘要周期。
- Tolerance Duration 暂按候选 `minimum_active_duration` 展示；只有现场日志或业务确认
  语义一致后才正式映射。无法确认的迁移规则保持禁用并标记不兼容项。
- 固定邮箱映射固定联系人。
- Notify Default Recipients 映射默认 NOC 联系组。
- 邮件图标状态映射规则邮件渠道启用状态。

迁移工具先输出预览和不兼容项，不直接启用规则。由于老历史缺少逐收件人事实，不迁移为
新的投递明细。

## 分阶段上线

### 阶段 0：Shadow

- 在保留 legacy `alarm.*` 行为期间写 Outbox 并发布独立的
  `domain.alarm.lifecycle.*`，
  验证主库告警事务、Outbox 和生命周期顺序。
- Alarm History Projector 先只对比 legacy 历史，不写 TimescaleDB；完成核对后，在同一
  受控开关中停止原同步归档并启用幂等投影写入，避免双写历史。
- 通知中心只消费事件、匹配规则、解析收件人和生成预览。
- 不创建真实外部发送。
- 对比告警数量、命中规则、收件人和老系统结果。
- 输出 `notify_email` 首条命中兼容差异，存在未确认差异时禁止进入试点。

### 阶段 1：邮件试点

- 选择少量设备组、KPI 查询模板和管理员白名单邮箱。
- 先验证告警 raised/clear/recovery、权限范围、固定正文、SMTP accepted、失败重试和熔断。
- 再验证 KPI 完整窗口、同源导出、附件大小限制、逐收件人发送和重启恢复。
- 不验收 SMS、quiet hours、重复提醒、值班排班或跨业务系统事件。

### 阶段 2：邮件全面启用

- 生产门禁通过后，按设备组分批启用告警邮件和 KPI 定时报表。
- 每批重新检查 Outbox/durable lag、投递重复、权限范围、渠道健康、死信和附件失败。
- 任一门禁失败时只停通知消费者或邮件渠道，不停止告警主流程和 KPI 查询。

### 未来保留：SMS

SMS 不属于本阶段上线阶段。未来必须另立需求单，完成供应商/短信平台协议、合规模板、
回执、幂等、SLA、凭据、测试环境和运维责任确认后，才新增 Kafka 或直连 Worker。

提供全局渠道开关、单规则开关和运营商范围开关。紧急回退只停止通知消费者或渠道，
告警主流程继续运行。

## 测试设计

### 单元测试

- 规则字段内 OR、字段间 AND。
- 固定模板主题包含设备 SN、告警标识和状态；正文字段完整、清除时间和处理建议正确。
- 中文/英文字段选择、空建议占位和特殊字符编码。
- occurrence 去重、严重级别升级、确认停止和清除恢复配对。
- Tolerance Duration 和 Interval 的最终冻结语义。
- KPI 15 分钟/小时/天窗口均为完整半开窗口，发送延迟不会读到未来数据。
- KPI 模板 revision、权限快照和收件人快照的变更语义。
- 错误分类、退避、熔断、half-open 和死信。
- 启用门禁在 legacy、shadow、SMTP disabled、Worker 未注册时拒绝启用。

### 集成测试

- 主 PostgreSQL 告警变更与 `alarm_event_outbox` 原子性。
- 清除后 TimescaleDB 历史投影失败、重试和幂等恢复。
- Outbox 锁竞争、NATS 失败、恢复和重放。
- 生命周期事件乱序、版本缺口、重复和旧版本忽略。
- 重复事件不产生重复投递。
- 服务重启后最小持续时间、摘要窗口和报表运行仍可恢复。
- 确认或清除与 Worker 领取并发时，外呼前栅栏阻止旧任务发送。
- SMTP 受理、鉴权失败和部分收件人失败。
- 规则修改后历史仍关联原版本。
- KPI 导出与查询模板同源，验证 `statis_type`、15 分钟点、维度、`object_ldn`、权限和分页边界。
- 模板权限撤销、设备组变更、报表重试和多 Worker 领取不扩大数据范围、不重复发件。
- 数据范围、完整地址权限和启用门禁。

### 场景测试

- eNB、gNB、CPE、UPS、WCG/GSM 告警邮件。
- 不同运营商和制式。
- 产生、重复、升级、确认、取消确认、清除和快速抖动。
- 大量设备同时离线或链路故障形成告警风暴。
- NATS、SMTP、对象存储、KPI 导出失败和恢复。
- 范围过大、权限变化、重复地址和错误联系地址。
- 中英文、业务时区、夏令时边界和特殊字符。

### 浏览器验收

- 告警邮件设置创建、修改、归档、旧/新范围权限校验和并发 412。
- 告警主题/正文固定提示、收件人脱敏和默认收件人超管隔离。
- KPI 查询模板定时报表真实保存、启停、发送时间、周期和下一次执行时间。
- 渠道健康与秘密不回显。
- 投递明细、尝试、脱敏和重试权限。
- raised → clear → recovery 的通知轨迹。
- 所有真实请求参数与后端契约一致。

## 验收标准

- 已提交告警事件不会因进程或 NATS 故障丢失。
- 事件重放不会产生重复邮件；SMS 不属于本阶段验收。
- 在需求单最终确认的正常负载条件下，符合发送条件的告警到 SMTP accepted 的内部处理
  延迟 P95 不超过 30 秒；外部 SMTP 供应商延迟单独统计。
- 通知模块故障不影响告警入库、查询、确认和清除。
- SMTP 受理不显示为最终送达。
- 每个收件人、渠道和尝试均可审计。
- 确认或清除后不发送已取消的迟到首次邮件；取消确认不重发首次邮件。
- 生命周期事件乱序或重复不能导致状态回退，也不能在确认或清除后产生迟到通知。
- TimescaleDB 暂时不可用不阻塞主库告警清除，恢复后历史投影完整且不重复。
- 瞬时和风暴告警按策略抑制或聚合，不能静默丢失审计事实。
- 配置错误触发熔断，不持续产生相同失败记录。
- 权限范围外的设备、投递和完整联系地址不可见。
- KPI 报表只发送已完成窗口，附件与查询模板同源，不重新计算 KPI，不绕过权限。
- 模板、权限、收件人和窗口变更不会改写已创建运行的审计事实。
- 链路未就绪时 API 拒绝启用，生产外发门禁未通过时 SMTP 和规则保持 disabled。
- 数据库修改只维护三个 `000001` 基线文件，不新增后续迁移。

## 主要风险与控制

### 老系统后端源码不可读

只迁移现场和可读文档共同确认的业务语义；未知 Java 细节不作为新系统契约。

### 需求单原文和历史语义不完整

当前工作区只有需求摘要，Redmine/GitLab 原单正文需要在开发前补齐。老 OMC 的 Interval、
Tolerance Duration、语言和主题规则不能只凭页面字段推断；无法确认的迁移配置保持停用。

### KPI 窗口或权限快照错误

调度时间、完整窗口、模板 revision 和设备数据权限必须在同一执行契约中冻结，并用延迟执行、
权限撤销、设备组变化和重试场景验证，避免报表发送未来数据或越权数据。

### 生产外发门禁未完成

本地假 SMTP、隔离数据库和 Shadow 只能证明实现可试点，不能证明真实 SMTP、出口 ACL、
容量、白名单收件人和小基站生命周期样本已通过。门禁未完成时保持 disabled。

### 告警风暴放大外部成本

Critical 实时但有上限；低级别默认聚合；所有渠道设置限流、配额和熔断。

### 动态权限导致收件人变化

事件发生时解析并快照，历史保持可解释；恢复通知使用原已受理投递，不重新扩大范围。

### 渠道故障事件递归

通知来源 system incident 带递归保护，只走站内、指标或其他健康渠道，不冒充网元告警。

## 被否决方案

### 直接订阅现有 NATS 事件并发送

实现快，但现有告警发布失败只记日志，不能保证事件不丢；不满足运营商级可靠性。

### 继续扩展 `alarm_filters.notify_email`

告警过滤器当前是单动作、首个匹配语义，适合告警处理，不适合多规则、多渠道、动态收件人、
聚合、恢复和审计。继续扩展会加深告警域与通知域耦合。

### 第一阶段拆独立微服务

会提前增加跨服务事务、鉴权、部署和版本成本；当前内部模块与事件契约已经能提供清晰边界。
