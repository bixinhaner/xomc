# OMC 邮件短信统一通知中心设计

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
可读 JSP、现有说明文档和 2026-07-31 老 OMC 13.0.5 现场页面共同证明的行为。

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

## 目标

- 已提交的告警事件不会因进程退出、NATS 中断或重复消费而丢失或重复发送。
- 告警处理与通知处理解耦，通知失败不阻塞告警入库、确认、清除和查询。
- 邮件、Kafka 短信和直连短信使用统一规则、模板、收件人、重试和审计模型。
- 准确区分“服务商受理”“最终送达”“外部交接”和“发送失败”。
- 支持固定联系人、用户、角色和系统默认 NOC 联系组，并服从设备数据权限。
- 支持实时、延迟门槛、周期汇总、Critical 重复提醒、恢复通知和告警风暴抑制。
- 每次规则匹配、投递、重试、回执和人工操作均可审计。
- 保留老业务能力，但不照搬其混合模板、批次历史和无限重复失败。
- 为后续非告警事件提供通用通知基础，不在第一阶段扩大业务范围。

## 非目标

- 第一阶段不建设独立通知微服务。
- 第一阶段不承诺邮件最终阅读或最终投递；SMTP 成功只表示服务器受理。
- Kafka 发布成功不显示为短信送达。
- 第一阶段不建设设备组负责人或自动值班排班系统。
- 不把 `alarm_filters` 继续扩展成完整通知规则引擎。
- 不让通知模块修改告警状态或成为告警事实源。
- 不新增 `000002+` 数据库迁移。
- 不在通用通知代码中散落运营商特例。

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
  → Email Worker /（接口冻结后）SMS Direct 或 SMS Kafka Worker
  → 服务商回执或 Kafka Ack
  → 投递状态、尝试记录、渠道健康和指标
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
`domain.alarm.>`，使用 `LimitsPolicy`、S2 压缩、默认 7 天 MaxAge 和 1 GiB MaxBytes；
每个消费者使用独立 durable。采用不同顶级前缀是为了避免与现有 `alarm.>` 流 Subject
重叠，也避免生产环境重建现有 ALARM 流。

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
- 为同一 occurrence 的每次生命周期变化递增持久化 `alarm_version`。
- 清除 Outbox 保存归档所需的完整快照，使 TimescaleDB 投影不依赖已删除活动行。
- 单条、批量、自动确认、自动清除、同步对账和 Expedited Event 等所有活动告警写路径
  必须经过相同生命周期事务；REST Handler 不得再绕过 AlarmEngine 直接批量改表。
- 不解析通知规则，不读取收件人，不发送消息。

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
- 解析用户、角色、固定联系人和默认联系组。
- 执行设备数据范围求交集。
- 处理最小持续时间、重复提醒、抖动、聚合、恢复配对和安静时段。
- 严格渲染模板并创建逐收件人、逐渠道的投递记录。
- 不直接执行外部网络调用。

### Channel Workers

- 邮件、直连短信、Kafka 短信分别实现窄适配器。
- 领取投递任务、记录每次尝试、分类错误、安排重试或等待回执。
- 运营商特有的短信供应商或字段映射通过 `internal/core/carrier/` 适配，不在通用
  Worker 中判断运营商。

### Receipt Processor

- 验证短信供应商回调签名和时间窗口。
- 通过 `provider_message_id` 和内部投递 ID 幂等更新结果。
- 回调重复或乱序时不允许已送达状态被旧失败回执覆盖。

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
Subject，其载荷不统一，通知中心不得消费。备份任务、磁盘阈值等系统模块应调用
AlarmEngine，或发布到独立、版本化的告警命令 Subject，由适配器完成校验和持久化。
命令事件不是告警事实，通知中心和北向接口不得订阅。自定义载荷中的 `alert_email` 等
收件人提示不进入生命周期契约，收件人统一由通知规则解析。

事件载荷至少包括：

```text
schema_version
event_id
event_type
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
event_type_name
probable_cause
specific_problem
alarm_source
status
raised_at
acknowledged_at
cleared_at
alarm_count
additional_information
```

载荷是事件发生时的不可变业务快照，通知中心不得在重放时用当前告警内容悄悄改写历史。
敏感或超大 AdditionalInformation 必须按模板允许列表使用，不能原样泄露到短信。

`alarm_version` 从 1 开始，每次产生、属性变更、确认、取消确认和清除严格递增。通知
Inbox 先以 `event_id` 去重，再按 occurrence 应用版本：

- `version <= last_applied_version`：视为重复或旧事件，记录后忽略。
- `version == last_applied_version + 1`：应用并推进状态。
- `version > last_applied_version + 1`：进入等待，不跨版本执行通知策略；缺失事件到达后
  继续处理，超过等待上限则产生运维告警。

Worker 在外部调用前必须重新检查 occurrence 当前状态、版本和 schedule generation。
已确认或已清除后领取到的旧首次通知和重复提醒必须取消；已经进入外部调用的请求则按
真实服务商结果结束，不能伪装成取消。

## 通知规则

### 匹配条件

规则可匹配：

- 生命周期事件：产生、清除、严重级别升级。
- 严重级别。
- 告警标识、告警名称、事件类型、告警来源、可能原因。
- 运营商、制式、网元类型。
- 设备、设备组。
- 生效时间、星期和安静时段。

同一字段内多个值为 OR，不同字段之间为 AND。空字段表示不限。

### 冲突与去重

- 规则有显式优先级。
- 多条规则可以合并补充收件人或渠道。
- 同一事件、发送用途、序号、渠道和收件人只创建一条投递。
- 多条规则对同一投递给出不同模板或策略时，最高优先级规则获胜。
- 同优先级时使用更具体的规则，再以稳定 ID 决胜，结果必须可解释和可审计。
- 预览接口必须返回命中规则、胜出原因、收件人数和被权限排除数量。

### 发送策略

规则分别配置：

- `minimum_active_duration`：告警持续达到该时间后才允许首次通知，对应老系统
  Tolerance Duration 的候选语义；未完成现场确认前，迁移规则不得自动启用。
- `delivery_mode`：实时或摘要。
- `aggregation_window`：摘要窗口。
- `repeat_interval`、`max_repeat_count`、`max_repeat_duration`：Critical 提醒。
- `stop_repeat_on_ack`：确认后停止提醒，第一阶段默认开启。
- `send_recovery`：是否发送恢复通知。
- `quiet_hours`：安静时段及 Critical 是否绕过。
- 规则、渠道和收件人速率限制。

默认建议：

- Critical：邮件和短信实时；未确认时按可配置周期提醒。
- Major：邮件实时；短信由显式规则开启。
- Minor、Warning：默认摘要，避免告警风暴。
- Cleared：与已成功受理的首次或升级通知配对。

## 收件人与权限

支持四类目标：

- 固定联系人。
- 指定用户。
- 指定角色。
- 系统保留的默认 NOC 联系组。

默认联系组可以包含固定联系人、用户和角色，用于承接老系统“Notify the Default
recipients”业务。

用户和角色在事件发生时解析并快照：

- 只包含启用状态的用户。
- 渠道地址不能为空且必须通过格式校验。
- 告警设备必须位于用户可见设备组范围。
- 同一地址只保留一次。
- 后续用户联系方式变化不改写既有历史。

创建或编辑规则的操作人只能选择自己有权管理的设备范围。投递历史查询继续按告警设备
数据范围过滤；邮箱和手机号默认脱敏，完整地址需要独立权限。

## 模板

告警规则与消息正文模板分离。模板按事件、渠道、语言和版本管理：

- 邮件包含主题、纯文本正文，可选受控 HTML 正文。
- SMS 只有短文本正文。
- Raised、Escalated、Cleared 使用独立模板或显式回退关系。
- 保存模板时校验允许变量、语法和渠道长度。
- 渲染使用严格缺失变量错误，禁止 `missingkey=zero` 静默产生错误消息。
- 每次发布产生不可变版本；投递记录关联实际模板版本。
- 收件人语言为空时依次回退系统语言和英文。
- SMS 超长时显示预计分段数并阻止静默截断。

允许变量来自版本化白名单，例如告警名称、级别、SN、网元类型、产生时间、可能原因、
特定问题和 OMC 链接。不得允许任意数据库字段或未脱敏秘密进入模板。

## 渠道语义

### Email SMTP

- SMTP 成功响应后记录 `accepted`。
- 第一阶段不声称最终送达或阅读。
- SMTP 鉴权失败、证书错误属于配置错误，打开渠道熔断。
- 单个收件人单独发送或使用安全信封，不能在 To/Cc 中泄露其他收件人。

### SMS Kafka Handoff

- Kafka Broker Ack 后记录 `handoff_only`。
- 该状态在页面显示为“已交接外部短信平台”，不能显示“短信已送达”。
- 若未来外部平台提供回执，可在不修改已有语义的前提下补充 delivered/failed。

### SMS Direct

- 服务商 API 受理后记录 `accepted` 并进入 `awaiting_receipt`。
- 成功回执记录 `delivered`。
- 失败回执按错误类别重试或终止。
- 回执超时记录 `unknown`，先查询状态，不盲目重发。

第一阶段只上线邮件。当前项目没有 Kafka 客户端、Broker 配置或运行依赖，不能仅因老 OMC
使用过 Kafka 就在新架构中默认引入。Kafka 短信必须先确认外部平台确实要求 Kafka，并冻结
Broker 安全协议、Topic、消息 Schema、Ack 和幂等契约；直连短信必须冻结供应商协议、
鉴权、回执、幂等和测试环境。两种 SMS Adapter 都复用统一通知模型，但分别独立上线。

## 数据模型

所有新增结构折回 `omcgo/migrations/000001_init_schema.sql`，不创建 `000002+`。

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

`event_id` 唯一，错误信息必须脱敏。

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
- `schedule_kind=initial_gate|repeat|digest_flush|quiet_hours_release`。
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

通知渠道故障产生 OMC 内部 `NOTIFICATION_CHANNEL_UNAVAILABLE` 系统告警，但必须设置
`origin=notification` 递归保护：故障渠道不能通知自己的故障，只允许站内告警、其他
健康渠道或北向接口处理。

重试耗尽进入死信。具有权限的管理员在修复配置后可以单条或批量重试，操作原因写入审计。

## API 设计

### 规则与模板

- `GET/POST /api/v1/notification-rules`
- `GET/PATCH /api/v1/notification-rules/{id}`
- `POST /api/v1/notification-rules/{id}/publish`
- `POST /api/v1/notification-rules/{id}/enable`
- `POST /api/v1/notification-rules/{id}/archive`
- `POST /api/v1/notification-rules/{id}/preview`
- `GET/POST /api/v1/notification-templates`
- `GET/PATCH /api/v1/notification-templates/{id}`
- `POST /api/v1/notification-templates/{id}/publish`
- `POST /api/v1/notification-templates/{id}/preview`

预览只返回命中告警、设备范围、规则解释、收件人数和渲染预览，不发送消息。

规则和模板修改产生 draft 版本，发布后版本不可变。所有 `PATCH`、`publish`、`enable`、
`archive` 请求必须带 `If-Match` 或等价 revision 前置条件；缺少前置条件返回 428，
revision 不匹配返回 412。启用请求必须指定准确的已发布 `rule_version_id`；规则渠道绑定
准确的已发布模板版本。新模板发布后不会自动改变已启用规则，管理员需要发布并启用新的
规则版本，前端必须明确提示该语义。

### 联系组与渠道

- `GET/POST /api/v1/notification-contact-groups`
- `GET/PATCH /api/v1/notification-contact-groups/{id}`
- `GET /api/v1/notification-channels`
- `PATCH /api/v1/notification-channels/{id}`
- `POST /api/v1/notification-channels/{id}/verify`
- `GET /api/v1/notification-channels/{id}/health`

渠道查询永不返回秘密。连接验证和测试发送分开授权；测试发送必须明确指定授权测试地址并
记录审计。

### 投递与审计

- `GET /api/v1/notification-deliveries`
- `GET /api/v1/notification-deliveries/{id}`
- `GET /api/v1/notification-deliveries/{id}/attempts`
- `POST /api/v1/notification-deliveries/{id}/retry`
- `POST /api/v1/notification-deliveries/retry`

批量重试必须限制数量、要求原因并再次经过数据权限校验。

## 前端设计

V1 在“系统管理 → 通知管理”提供：

1. 通知规则。
2. 消息模板。
3. 默认联系组。
4. 渠道配置与健康。
5. 投递记录与失败重试。

规则编辑流程：

```text
基本信息
  → 告警匹配
  → 收件人
  → 邮件/SMS 渠道
  → 持续时间、周期、重复、恢复和限流
  → 匹配预览
  → 保存草稿或启用
```

列表显示优先级、范围摘要、渠道徽标、启用状态和最近健康状态。启用覆盖大量设备或
Critical SMS 的规则时显示影响范围确认。

告警详情增加只读“通知轨迹”，展示命中规则、脱敏收件人、渠道状态、抑制/聚合原因、
投递尝试和恢复配对，不在该区域编辑通知规则。

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
- 备份和其他系统模块的自定义 `alarm.*` 发布改为 AlarmEngine 调用或独立告警命令事件。

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

- 选择少量设备组和管理员测试邮箱。
- 验证实时、最小持续时间、聚合、确认停止提醒、清除和故障熔断。

### 阶段 2：邮件全面启用

- 先启用 Critical/Major。
- 稳定后启用 Minor/Warning 摘要。

### 阶段 3：Kafka 短信

- 仅在外部短信平台接口确认必须使用 Kafka 后实施；否则不引入 Kafka 运行依赖。
- 只对 Critical 和明确授权规则启用。
- 页面和 API 只显示 handoff。

### 阶段 4：直连短信

- 供应商协议、回执和测试环境确认后启用 delivered 语义。

提供全局渠道开关、单规则开关和运营商范围开关。紧急回退只停止通知消费者或渠道，
告警主流程继续运行。

## 测试设计

### 单元测试

- 规则字段内 OR、字段间 AND。
- 优先级、具体度、目标合并和稳定决胜。
- 用户、角色、默认联系组解析及设备范围求交集。
- occurrence 去重、严重级别升级、确认停止和清除配对。
- 最小持续时间、抖动、聚合、限流和安静时段。
- 模板变量白名单、严格缺失变量、多语言和 SMS 长度。
- 错误分类、退避、熔断、half-open 和死信。
- 回执签名、重复、乱序和超时。

### 集成测试

- 主 PostgreSQL 告警变更与 `alarm_event_outbox` 原子性。
- 清除后 TimescaleDB 历史投影失败、重试和幂等恢复。
- Outbox 锁竞争、NATS 失败、恢复和重放。
- 生命周期事件乱序、版本缺口、重复和旧版本忽略。
- 重复事件不产生重复投递。
- 服务重启后最小持续时间、重复提醒和安静时段任务仍可恢复。
- 确认或清除与 Worker 领取并发时，外呼前栅栏阻止旧任务发送。
- SMTP 受理、鉴权失败和部分收件人失败。
- Kafka Ack 只产生 handoff。
- 直连短信受理、回执成功、失败和未知。
- 规则修改后历史仍关联原版本。
- 数据范围和完整地址权限。

### 场景测试

- eNB、gNB、CPE、UPS、WCG/GSM 告警。
- 不同运营商和制式。
- 产生、重复、升级、确认、清除和快速抖动。
- 大量设备同时离线或链路故障形成告警风暴。
- NATS、SMTP、Kafka、短信服务商中断和恢复。
- 规则覆盖范围过大、重复规则和错误联系地址。
- 中英文、时区和特殊字符。

### 浏览器验收

- 规则创建、预览、启停、归档。
- 模板校验和预览。
- 渠道健康与秘密不回显。
- 投递明细、尝试、脱敏和重试权限。
- 告警详情通知轨迹。
- 所有真实请求参数与后端契约一致。

## 验收标准

- 已提交告警事件不会因进程或 NATS 故障丢失。
- 事件重放不会产生重复邮件或短信。
- 正常负载下实时通知内部处理延迟 P95 不超过 30 秒；外部供应商延迟单独统计。
- 通知模块故障不影响告警入库、查询、确认和清除。
- Kafka Ack 不显示为 delivered。
- SMTP 受理不显示为最终送达。
- 每个收件人、渠道和尝试均可审计。
- Critical 提醒在确认或清除后停止。
- 取消确认不重发首次通知，只能在既有规则次数和时长上限内恢复未来提醒。
- 生命周期事件乱序或重复不能导致状态回退，也不能在确认或清除后产生迟到通知。
- TimescaleDB 暂时不可用不阻塞主库告警清除，恢复后历史投影完整且不重复。
- 瞬时和风暴告警按策略抑制或聚合，不能静默丢失审计事实。
- 配置错误触发熔断，不持续产生相同失败记录。
- 权限范围外的设备、投递和完整联系地址不可见。
- 数据库修改只维护三个 `000001` 基线文件，不新增后续迁移。

## 主要风险与控制

### 老系统后端源码不可读

只迁移现场和可读文档共同确认的业务语义；未知 Java 细节不作为新系统契约。

### 外部短信最终状态不明确

Kafka 和直连短信分成不同渠道语义。没有回执时只记录 handoff 或 accepted/unknown。

### 告警风暴放大外部成本

Critical 实时但有上限；低级别默认聚合；所有渠道设置限流、配额和熔断。

### 动态权限导致收件人变化

事件发生时解析并快照，历史保持可解释；恢复通知使用原已受理投递，不重新扩大范围。

### 渠道故障告警递归

通知来源系统告警带递归保护，只走站内或其他健康渠道。

## 被否决方案

### 直接订阅现有 NATS 事件并发送

实现快，但现有告警发布失败只记日志，不能保证事件不丢；不满足运营商级可靠性。

### 继续扩展 `alarm_filters.notify_email`

告警过滤器当前是单动作、首个匹配语义，适合告警处理，不适合多规则、多渠道、动态收件人、
聚合、恢复和审计。继续扩展会加深告警域与通知域耦合。

### 第一阶段拆独立微服务

会提前增加跨服务事务、鉴权、部署和版本成本；当前内部模块与事件契约已经能提供清晰边界。
