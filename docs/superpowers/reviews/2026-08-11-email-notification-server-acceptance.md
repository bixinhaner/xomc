# OMC 告警邮件服务器部署后验收方案

> **状态**：待部署后执行
>
> **验收对象**：告警邮件通知流程，关联 RR #31002、需求单 #31315、#31316
>
> **本轮结论**：本文件只生成验收方案，不代表服务器、真实 SMTP 或真实小基站已经验收通过。
>
> **执行原则**：由 OMC 管理员和现场/NOC 代表双人执行；先隔离、再小范围、后扩大。任何阻断项未关闭前，不得全网启用。

## 1. 验收依据

1. Redmine 需求单 #31316：告警邮件不区分活动告警和清除告警，邮件同时发送产生时间和清除时间，标题改为“告警通知/Alarm Notification”；自测结果为新增告警描述和可能的处理建议。
2. [OMC 告警邮件范围收敛评审](2026-08-06-email-notification-scope-convergence.md)：冻结当前业务边界、权限、可靠性门禁和排除项。
3. [通知中心运行手册](../../operations/notification-center-runbook.md)：部署开关、Shadow、canonical、故障注入和回退顺序。
4. 老 OMC 13.0.5 页面与代码：
   - `omcmb/original-omc/OMCWebServer/src/main/webapp/WEB-INF/content/cell/fault/notification.jsp`
   - `omcmb/original-omc/OMCWebServer/src/main/webapp/WEB-INF/content/cell/fault/alarmNoticeSetting.jsp`
   - `omcmb/original-omc/OMCWebServer/src/main/webapp/WEB-INF/content/cell/fault/alarm_temp_oper.jsp`

## 2. 验收范围

### 2.1 本次必须验收

- 告警邮件配置入口、配置保存、列表、修改、停用/归档和权限控制。
- 告警范围：告警标识、严重级别、制式、设备或设备组。
- 邮件策略：启用状态、发送间隔、`Tolerance Duration`、固定收件人、系统默认收件人。
- 告警 raised 和 cleared 两种生命周期邮件，且使用同一告警 occurrence。
- 固定邮件标题、告警描述/具体问题、处理建议、产生时间和清除时间。
- 告警主流程与邮件异步解耦：邮件失败不能阻塞告警入库、确认、清除和查询。
- Outbox、NATS JetStream、Notification Inbox、Worker、SMTP accepted、逐收件人 attempt、重试、熔断和审计。
- 真实测试 SMTP 的 TLS、AUTH、出口 ACL、白名单收件人和浏览器操作证据。
- 服务重启、多实例领取、NATS/SMTP 故障恢复以及紧急停止和回退。

### 2.2 本次明确不验收

- Zed Mobile 2G/4G/5G 状态汇总邮件及 #98018。
- KPI 定时报表、CPE 列表邮件及 #42492/#62360。
- SMS、Kafka 短信、直连短信、短信回执、Office365/OAuth。
- 值班排班、升级链、quiet hours、重复提醒和跨业务通用通知编排。
- 管理员编辑邮件主题、HTML、正文或变量的通用模板中心。
- 邮件最终送达、阅读回执。`accepted` 只表示 SMTP 服务端受理。

## 3. 老流程与当前流程对照

| 对照项 | 老 OMC 业务流程 | 当前项目验收要求 |
| --- | --- | --- |
| 配置入口 | 告警通知页面/告警模板 | `告警管理 → 邮件通知设置` |
| 配置对象 | 告警范围与邮件策略混合配置 | 专用告警邮件设置；正文由后端固定渲染 |
| 范围 | 网元类型、设备/设备组、告警定义、严重级别 | 告警标识、严重级别、制式、设备/设备组；不得扩大授权范围 |
| 时间策略 | `Interval`、`Tolerance Duration`，现场值为实时/10/30/60 分钟 | 保留实时/10/30/60 分钟；精确的 Tolerance 语义须在执行前由业务确认 |
| 收件人 | 分号分隔固定邮箱，可包含系统默认邮箱 | 固定邮箱 + 可选默认收件人；地址加密存储，页面/API 只展示脱敏结果 |
| 发送链路 | `AlarmEmail Quartz → AlarmEmailNoticeService → SendMailTool → alarm_email_record` | 告警事务/Outbox → NATS → Notification Inbox/编排 → Email Worker → SMTP |
| 活动/清除 | 以老版本现场行为和需求单为准 | raised 与 cleared 均发送；同一 occurrence 的 recovery 投递关联 initial 投递 |
| 邮件标题 | 需求单要求改为“告警通知/Alarm Notification” | 只能为 `告警通知` 或 `Alarm Notification`，不得拼接 SN、告警标识或状态 |
| 邮件正文 | 原有描述并增加可能的处理建议 | 固定包含告警名称、标识、级别、网元类型、设备 SN、状态、产生/清除时间、可能原因、具体问题、处理建议 |
| 历史记录 | 批次级成功/失败/部分成功 | 逐收件人 delivery/attempt；地址脱敏；accepted 不等于最终送达 |

## 4. 执行前置条件

以下条件全部满足后才开始 P0；缺一项则记录为“阻断”，保持邮件渠道和规则 disabled。

| 编号 | 前置条件 | 现场填写 |
| --- | --- | --- |
| PRE-01 | 部署版本、Git revision、数据库、App/Worker/NATS/SMTP 主机已记录 |  |
| PRE-02 | 已取得仅用于验收的阿里企业邮箱 SMTP 配置：465、隐式 TLS、AUTH、出口 ACL |  |
| PRE-03 | 测试收件人已加入白名单，收件地址不写入日志、截图或文档 |  |
| PRE-04 | 已准备至少一个可控小基站，能产生 raised、ack/unack、clear/recovery |  |
| PRE-05 | 已准备一个含中英文告警名称、可能原因和处理建议的告警定义 |  |
| PRE-06 | 已准备一条无建议或缺失语言字段的定义，用于确认回退/空值显示规则 |  |
| PRE-07 | 已准备超管账号、授权设备组操作员账号、无权限设备组账号、第二管理员账号 |  |
| PRE-08 | App 与 Worker 的 `alarm.lifecycle_mode`、起始 sequence、Stream 参数、SMTP 参数和收件人密钥一致 |  |
| PRE-09 | 已记录验收前 Outbox、三个 durable、Notification Inbox、schedule、delivery、channel health 基线 |  |
| PRE-10 | 已确认本轮 Tolerance Duration 的业务语义、期望等待时间和允许误差 |  |
| PRE-11 | 已确认故障注入窗口，不在生产业务高峰或非验收网元上执行 |  |

**敏感信息规则**：密码、授权码、收件人密文、密钥和完整邮箱地址不得进入本文件、工单、截图、日志或命令历史。

## 5. 测试数据与记录表

| 数据编号 | 内容 | 实际值/脱敏标识 |
| --- | --- | --- |
| DATA-01 | 验收网元 |  |
| DATA-02 | 设备所属设备组 |  |
| DATA-03 | 告警标识/名称 |  |
| DATA-04 | 告警级别/制式 |  |
| DATA-05 | 中文处理建议 |  |
| DATA-06 | 英文处理建议 |  |
| DATA-07 | 测试收件人数 |  |
| DATA-08 | 配置规则 ID/版本/revision |  |
| DATA-09 | raised occurrence/event ID |  |
| DATA-10 | cleared occurrence/event ID |  |
| DATA-11 | initial/recovery delivery ID |  |

所有时间统一记录：服务器时间、系统业务时区、邮件服务端受理时间。截图须能看出页面、状态和时间，但不得暴露完整邮箱或 Secret。

## 6. 分阶段验收步骤

状态只允许填写：`未执行`、`通过`、`失败`、`阻断`、`不适用`。每项必须填写证据，不得只写“已验证”。

### P0：部署、开关与基线

| 编号 | 操作 | 预期结果 | 证据/状态 |
| --- | --- | --- | --- |
| P0-01 | 核对 App/Worker 配置和启动日志 | 两者 lifecycle mode、起始 sequence、`DOMAIN_ALARM` 参数、SMTP enabled 和收件人密钥一致；无 Secret 明文日志 |  |
| P0-02 | 核对主库 Outbox、Inbox、schedule、delivery、attempt 和 channel health | 验收前积压、最老时间、熔断状态已记录；不存在未解释的旧数据 |  |
| P0-03 | 核对三个 durable | `alarm-history-projector-v1`、`northbound-alarm-lifecycle-v1`、`notification-lifecycle-v1` 均存在并追平 |  |
| P0-04 | 保持 SMTP 和规则 disabled，产生一条受控告警 | 告警可写入、确认、清除；邮件不外发；disabled 不影响告警主流程 |  |
| P0-05 | 在渠道页执行 verify | 只完成 SMTP 握手；成功记录 verify 时间，不产生测试邮件 |  |
| P0-06 | 尝试在 SMTP disabled 时启用规则 | API/UI 拒绝启用并返回明确错误，规则 revision、发布指针和启停指针不发生半成功变化 |  |

**P0 通过门槛**：配置一致、基线可追溯、邮件 disabled 门禁有效、告警主流程正常。否则停止后续外发验收。

### P1：配置页面与权限

| 编号 | 操作 | 预期结果 | 证据/状态 |
| --- | --- | --- | --- |
| P1-01 | 使用超管打开“告警管理 → 邮件通知设置” | 页面正常加载；可查看列表、创建规则和管理系统默认收件人 |  |
| P1-02 | 创建一条 disabled 规则 | 可配置告警标识、级别、2G/4G/5G、设备/设备组、实时/10/30/60 分钟、Tolerance、固定收件人和默认收件人 |  |
| P1-03 | 保存后刷新并重新打开详情 | 所有字段持久化；规则版本/revision 可追踪；没有把正文编辑入口暴露给管理员 |  |
| P1-04 | 选中一个大于单页容量的设备结果集，并搜索/回读已选设备 | 服务端搜索可定位设备；已选设备不会静默截断到前 200 台 |  |
| P1-05 | 超管修改默认收件人并刷新 | 默认收件人保存成功；页面仅展示脱敏/受保护信息 |  |
| P1-06 | 普通操作员访问默认收件人入口/API | 入口不展示且 API 不被调用或返回无权；不能通过前端参数绕过 |  |
| P1-07 | 使用授权设备组操作员查看和修改授权范围内规则 | 列表、详情、修改、归档均成功；只能看到授权设备组与制式范围 |  |
| P1-08 | 使用无权限账号查看、修改、归档同一规则 | 返回无权或空结果；不能通过修改请求把旧范围替换为越权新范围 |  |
| P1-09 | 两个管理员用同一 revision 并发修改 | 先提交成功，后提交返回 `412/If-Match` 冲突；后提交者的数据不覆盖先提交者 |  |
| P1-10 | 归档 disabled 规则，再刷新列表 | 规则不再出现在有效列表；版本和审计记录保留，不物理删除投递历史 |  |

### P2：邮件正文与需求单 #31316

先启用唯一测试邮件渠道，再启用一条严格限定 DATA-01 的规则。每次只触发一条受控告警。

| 编号 | 操作 | 预期结果 | 证据/状态 |
| --- | --- | --- | --- |
| P2-01 | 触发 DATA-01 的 raised 告警，等待 accepted | 收到一封 initial 邮件；SMTP 服务端 accepted；delivery 为 completed/accepted，attempt 为第 1 次 accepted |  |
| P2-02 | 核对邮件标题 | 标题严格为 `告警通知` 或 `Alarm Notification`；不包含设备 SN、告警标识、状态或严重级别拼接内容 |  |
| P2-03 | 核对中文正文 | 包含告警名称、告警标识、告警级别、网元类型、设备 SN、告警状态、产生时间、清除时间、可能原因、具体问题、处理建议 |  |
| P2-04 | 核对 raised 邮件清除时间 | 尚未清除时显示明确空值占位，不伪造清除时间；产生时间与告警 occurrence 一致 |  |
| P2-05 | 核对中文处理建议 | 正文中的处理建议等于告警定义 `cnSuggestion`；不得使用用户可编辑模板内容替换后台建议 |  |
| P2-06 | 使用英文界面/英文模板语言触发同类告警 | 标题为 `Alarm Notification`；字段为英文；建议优先使用 `enSuggestion`，为空时按已确认的回退规则处理 |  |
| P2-07 | 清除 DATA-01 的同一 occurrence | 活动告警清除；收到 recovery 邮件；邮件仍包含产生时间和清除时间，清除时间为真实清除时间 |  |
| P2-08 | 对照 raised/recovery 邮件 | 两封邮件正文结构一致；recovery delivery 通过 `origin_delivery_id` 关联 initial；没有为同一生命周期生成无依据的重复邮件 |  |
| P2-09 | 使用缺失处理建议的定义触发告警 | 按部署前确认的空值/回退规则表现；不得把另一告警的建议串入本邮件 |  |
| P2-10 | 核对收件人 | 固定收件人和勾选的默认收件人均收到；未勾选默认收件人的规则不应发送给默认地址 |  |

**P2 通过门槛**：P2-02 至 P2-07 全部通过，才可判定 #31316 的核心业务需求通过。任何正文字段、活动/清除语义或处理建议错误均为阻断缺陷。

### P3：策略与告警主流程解耦

| 编号 | 操作 | 预期结果 | 证据/状态 |
| --- | --- | --- | --- |
| P3-01 | 使用实时 interval 触发一次告警 | 在约定 SLA 内产生一次 initial 投递；不等待错误的周期窗口 |  |
| P3-02 | 使用 10/30/60 分钟 interval 触发可控样本 | 按验收前确认的聚合/等待语义发送；记录首事件、发送时间、窗口和邮件数量 |  |
| P3-03 | 按确认的 Tolerance Duration 触发短于门槛和超过门槛的样本 | 短于门槛的样本按确认规则被抑制/不发送；超过门槛的样本按确认规则发送；若语义未确认，本项不得判通过 |  |
| P3-04 | 在邮件渠道故障时产生、确认并清除告警 | 告警主库事实、活动告警、确认和清除均成功；通知失败只进入 retry/dead-letter/unknown，不反向阻塞告警操作 |  |
| P3-05 | 同一事件重复投递/重复消费 | `event_id`、occurrence/version 和 delivery 幂等；不得产生重复 accepted 邮件 |  |
| P3-06 | 停止 Worker 后产生告警，再恢复 Worker | 投递持久化保留；Worker 恢复后按 lease 领取并发送；不因进程重启丢失或重复发送 |  |
| P3-07 | 启动两个 Worker 并发处理同一批投递 | `FOR UPDATE SKIP LOCKED`/租约生效；一个 delivery 同时只有一个有效发送 attempt |  |

### P4：SMTP、重试、熔断与审计

故障注入只能在隔离环境或事先批准的预生产窗口执行；真实生产 SMTP 不做破坏性注入。

| 编号 | 操作 | 预期结果 | 证据/状态 |
| --- | --- | --- | --- |
| P4-01 | SMTP 返回 451/临时连接失败 | attempt 记录失败分类和状态；进入有界退避；告警主流程仍正常 |  |
| P4-02 | 连续临时失败达到阈值 | channel circuit 打开；新的投递被抑制或延后；不会无限快速重试 |  |
| P4-03 | 修复 SMTP 后执行 verify | verify 成功记录 `last_verified_at` 并关闭 circuit；恢复投递不重复 accepted |  |
| P4-04 | SMTP 接收 DATA 后断开，不返回明确结果 | delivery/attempt 记录 `unknown`；系统不自动盲目重发；需人工与 SMTP 侧核对 |  |
| P4-05 | 产生永久失败/收件人拒收 | 按系统规则进入 dead-letter；历史显示失败原因；人工重试要求填写原因并记录操作者 |  |
| P4-06 | 查看通知历史和 attempt | 每个收件人独立记录；显示 masked address、状态、尝试次数、失败分类和时间；不显示完整地址、密文或 Secret |  |
| P4-07 | 检查操作审计 | 创建、修改、启停、归档、默认收件人变更、人工重试、权限拒绝均有操作者、时间、结果和对象记录 |  |

### P5：NATS/历史投影恢复与容量

| 编号 | 操作 | 预期结果 | 证据/状态 |
| --- | --- | --- | --- |
| P5-01 | 在隔离环境停止 NATS，产生 raised/clear | 告警主事务提交；Outbox 进入 pending/failed/retry；不丢失 occurrence |  |
| P5-02 | 恢复 NATS | Outbox 发布成功；三个 durable 最终追平；event ID 唯一，无人工复制 payload |  |
| P5-03 | 在隔离环境停止 TimescaleDB，产生 clear | 主库告警事实和 clear 不被历史投影阻塞；不手工补造历史 |  |
| P5-04 | 恢复 TimescaleDB | history projector 自动重试并追平；无重复历史；记录实际重试次数和恢复时间 |  |
| P5-05 | 使用现场平均/峰值事件率至少 2 倍执行压测 | 记录输入速率、处理 P95/P99、Outbox oldest age、三个 durable lag、数据库锁等待、Stream bytes、最大积压和磁盘余量；结果满足现场容量门禁 |  |

**P5 说明**：本地 200 条/秒基线只能作为实现参考，不能替代现场数据、最长消费者中断和磁盘预算下的容量结论。

### P6：浏览器完整回归与回退

| 编号 | 操作 | 预期结果 | 证据/状态 |
| --- | --- | --- | --- |
| P6-01 | 浏览器完成登录、进入邮件设置、创建、列表回读、修改、停用、归档 | 页面无 4xx/5xx；成功/失败提示明确；刷新后状态一致 |  |
| P6-02 | 浏览器查看通知历史、delivery 详情和 attempt | 数据与 API/数据库只读快照一致；地址脱敏；accepted 文案不写成“最终送达” |  |
| P6-03 | 浏览器验证越权和 revision 冲突 | 无权操作被拒；412 冲突被提示；表单旧值不会覆盖新值 |  |
| P6-04 | 按运行手册执行紧急停止 | 先禁用渠道，等待发送租约窗口，确认无新的 accepted，再关闭 SMTP 配置；告警接收/确认/清除继续工作 |  |
| P6-05 | 若需回退 canonical | 先停止外发，再冻结规则，App/Worker 同步回 legacy；禁止直接 SQL 删除 barrier、规则、delivery 或 attempt |  |
| P6-06 | 回退后执行受控告警 | 告警主流程和 legacy 路径可用；遗留 Outbox/Stream/审计数据保留；没有误发邮件 |  |

## 7. 只读检查与证据要求

服务器验收时复用运行手册第 3、6、8、9 节的只读查询和指标。至少保存以下证据：

- 部署版本、配置快照摘要、App/Worker 启动日志摘要。
- `DOMAIN_ALARM` 的消息数、bytes、first/last sequence、MaxAge、MaxBytes。
- 三个 durable 的 pending、ack pending、redelivered 前后值。
- Outbox、Notification Inbox、schedule、delivery、attempt 的状态统计和最老时间。
- channel health 的 circuit、连续失败、最近 verify、最近 accepted 和最近错误分类。
- 浏览器关键页面截图：配置列表、编辑表单、权限拒绝、412 冲突、投递历史、attempt 详情。
- 原始邮件 `.eml` 或脱敏正文：主题、正文、产生/清除时间、处理建议、收件人数量。
- SMTP 服务端日志中的 envelope、DATA accepted、message/time window；不得保留授权码。
- 故障注入前后快照、恢复时间、重复检查和回退结果。

证据文件命名建议：`{日期}-{阶段}-{用例编号}-{pass|fail}-{脱敏说明}`。完整地址、密码、密钥、token 和未脱敏邮件不得归档。

## 8. 通过标准与阻断项

### 8.1 通过标准

- P0、P1、P2、P3、P6 全部通过。
- P4 的真实 SMTP TLS/AUTH/ACL、逐收件人审计、重试、熔断、unknown 处置和敏感信息保护全部通过。
- P5 的 NATS/TimescaleDB 恢复通过；容量门禁由现场峰值和磁盘预算签字确认。
- 需求单 #31316 的标题、活动/清除、产生/清除时间、描述和处理建议无偏差。
- 双人复核并签字，邮件渠道和规则最终状态明确记录。

### 8.2 直接阻断上线

- 告警事实、确认或清除因邮件故障失败。
- 规则越权可见、可改、可启用，或修改新旧范围校验不完整。
- SMTP disabled 时仍能启用规则或产生外发。
- 邮件标题不符合要求，正文缺少处理建议、产生时间、清除时间或告警描述。
- raised/clear 产生错误 occurrence、重复 accepted 或 recovery 无法关联 initial。
- 完整收件人、密文、授权码或 Secret 出现在页面、API、日志或截图。
- NATS/Worker/TimescaleDB 中断恢复后丢事件、跳版本、重复历史或无法追平。
- 未取得真实 SMTP、测试收件人白名单、可控小基站样本或现场容量数据，却宣称生产验收通过。

## 9. 验收记录与最终结论

| 项目 | 结果 |
| --- | --- |
| 部署版本/Git revision |  |
| 验收环境/服务器 |  |
| 验收开始/结束时间 |  |
| 测试网元与设备组 |  |
| 测试收件人数量（仅填数量） |  |
| P0/P1/P2/P3/P4/P5/P6 |  |
| 未关闭缺陷及风险 |  |
| 最终邮件渠道状态 |  |
| 最终规则状态 |  |
| 是否允许扩大范围 |  |
| OMC 管理员签字 |  |
| 现场/NOC 代表签字 |  |

**最终结论**：只有所有阻断项关闭、核心需求通过、恢复与回退证据齐全后，才填写“通过”。否则填写“条件不通过”或“阻断”，并保持邮件渠道及规则 disabled，按[通知中心运行手册](../../operations/notification-center-runbook.md)执行处置。
