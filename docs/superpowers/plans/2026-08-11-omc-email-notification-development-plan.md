# OMC 邮件通知开发方案

> 日期：2026-08-11
> 分支：`codex/email-sms-notification-implementation`
> 主线基线：`ed4acab50b7f20111ae8fc88d9444eae6ce0c0f9`
> 状态：代码修复与本地验证完成；共享 SMTP、告警邮件、KPI 报表订阅主链路和 189 老 OMC 只读流程复核已完成，等待真实阿里企业邮箱 SMTP 联调

## 1. 结论

本轮只建设一套共享的阿里企业邮箱 SMTP 投递能力，并接入两个业务场景：告警邮件和 KPI 定时报表邮件。告警继续使用当前告警规则作为触发入口；KPI 报表继续使用当前指标查询模板和现有异步 CSV 导出链路。历史 OMC 的 Quartz、JavaMail、FTP、短信 Kafka 和整套通知模板调度不迁移。

当前实施已经补齐告警规则的真实配置闭环：前端可选择“邮件通知”、录入最多 50 个收件人并回显；API 会规范化、去重和校验地址；非邮件动作会清空历史收件人；规则命中后通过异步任务投递。完整邮箱地址不写入运行日志。

开发范围由以下需求组成：

| 需求 | 本轮处理 | 方案结论 |
|---|---|---|
| #72785 监控程序邮件发送适配 | 是 | 目标改为阿里企业邮箱；Office365 已停用，不保留 OAuth/Office365 兼容分支 |
| #62360 CPE 邮件监控列表中的 `module_name`、`online_count`、`offline_count` | 不迁移 | 原文确认这是已退役老 `OMCMonitor` 的 CPE 定时导出性能优化；新系统没有该外部监控邮件入口，且用户明确排除 #98018 自动汇总，不新增替代邮件/API/定时器 |
| #42492 KPI 统计邮件发送改造 | 是 | 在指标查询模板上增加定时报表邮件订阅，复用现有 `pm_kpi_export` CSV 生成链路 |
| #31315、#31316 告警邮件增加处理建议 | 是，合并实现 | 打通 XML 告警字典的 `cnSuggestion/enSuggestion` 到数据库、API、告警快照和邮件正文 |
| #98018 Zed Mobile 自动邮件发送 | **否** | 用户明确排除，不分析、不实现、不验收，也不借此新增 2G/4G/5G 在线离线汇总 |

## 2. 依据与现状

### 2.1 189 老 OMC 的只读核对结果

2026-08-12 在 `172.21.172.189:8081` 的 13.0.5 测试环境中完成只读走查，未保存配置、未触发测试邮件。实际流程确认如下：

- 告警页的信封标识可直接看出哪些模板已开启邮件；右上角通知入口进入 `Notification Setting`；
- `Notification Setting` 包含邮件总开关、分号分隔的默认收件人，以及模板级启停、发送间隔、模板收件人和“是否通知默认收件人”；
- 模板邮件编辑支持 `Real Time / 10 / 30 / 60 Minute` 的 `Interval` 和 `Tolerance Duration`；
- 告警模板主编辑先选择 `OMC / ENB / WCG / UPS / CPE / GNB` 告警源，再选择全部或自定义设备/设备组，随后从告警库按告警标识、事件类型、级别和告警源选择告警项，最后配置邮件开关、周期和收件人；
- KPI View 模板的更多菜单中存在 `Regular Report`；配置包含启停、发送时间、周期 `Day / Hour / 15Min`，以及 `Email / FTP` 通道；现场样例为 `Day + FTP`，本次仅查看后取消；
- 这些页面证明老系统存在“告警模板邮件”和“KPI 查询模板定时报表”两种业务入口，但不代表其全部配置项都要迁移。

本轮只借用业务入口和用户心智：告警邮件跟随告警规则，报表邮件跟随 KPI 查询模板。旧 OMC 的默认收件人叠加、Quartz 任务、FTP 和短信链路不搬迁。

### 2.1.1 老新业务流程映射

| 老 OMC 操作 | 新项目对应入口 | 本轮处理 |
|---|---|---|
| 打开告警通知设置 | 告警规则列表 | 复用现有真实规则入口，不恢复已删除的模拟通知设置页 |
| 选择告警源和告警项 | 规则中的来源、级别、告警标识等条件 | 按当前规则模型保留，不复制老告警库页面 |
| 选择设备/设备组 | 规则中的设备/设备组选择 | 保留老系统的业务含义 |
| 填写模板收件人 | 规则动作选择“邮件通知”后填写收件人 | 保留，最多 50 个并校验去重 |
| 启用模板 | 启用/禁用规则 | 保留 |
| 邮件总开关、默认收件人叠加 | 无 | 不迁移；部署级 `notification.smtp.enabled` 只控制基础设施是否可用 |
| Interval/Tolerance 周期汇总 | 无 | 不迁移；当前需求采用单个告警生命周期事件异步通知 |
| 邮件发送记录 | `notification_history` | 保留并增加业务幂等、尝试次数和错误状态 |
| KPI Regular Report 的 Email | KPI Query 的“定时报表” | 保留，复用现有 CSV 导出 |
| KPI FTP | 无 | 不迁移 |

189 环境复核已按只读方式完成：所有编辑页均通过 `Cancel` 退出，未点击 `OK`，未改变开关或收件人，未触发测试邮件。走查新增发现的全局开关、默认收件人合并、Interval/Tolerance、FTP 等字段没有本轮需求单依据，按老系统历史逻辑记录，不补回新项目。

### 2.2 老项目文档中可复用的业务事实

`/Users/hezg/Documents/work/OMC相关文档/old-omc-docs` 记录的老链路为：

```text
alarm_view_template.email_enable
  -> Quartz AlarmEmailJob
  -> AlarmEmailNoticeService
  -> SendMailTool/SMTP
  -> alarm_email_record
```

KPI 定时报表则由订阅配置触发，生成报表文件后再走 Email 或 FTP。可复用的是“业务配置与发送记录分离”和“报表生成后再投递”两点；Java/Quartz/Mongo/MySQL 表结构不复用。

旧文档还描述了短信、设备接入、地理围栏、联系人组、模板版本、静默期、限流和统一事件平台。这些是长期设想，没有本轮需求单依据，全部列为非目标。

### 2.3 当前主线代码基线

当前已经具备：

- `internal/notification/email_sender.go`：SMTP + 可选 STARTTLS，但不支持 465 隐式 TLS、附件；
- `internal/notification/mailer.go`：模板/原始邮件发送及 `notification_history` 记录；
- `internal/alarm/email_dispatcher.go`：另一套 SMTP，实现隐式 TLS 和 STARTTLS；
- `cmd/app/provider/alarm.go`：告警 SMTP 仍读 `OMC_SMTP_*` 环境变量；
- `internal/core/appconfig`：消息中心 SMTP 使用 `notification.smtp.*`；
- `internal/alarm/filter_engine.go`：规则动作 `notify_email` 直接拼纯文本邮件；
- `internal/pm/querytemplate`：指标查询模板及所有权/可见性；
- `internal/pm/export`：`source_type=kpi_query` 的异步 CSV 导出、对象存储和导出任务记录；
- `internal/core/asyncjob`：持久化异步任务、抢占、心跳、僵尸恢复和最多三次重试；
- `internal/core/systimezone`：系统时区唯一来源；PM 导出已使用该时区。

当前主要缺口：

1. 两套 SMTP 实现和两套配置源会产生环境不一致；
2. 共享发送器不支持阿里企业邮箱常用的隐式 TLS 模式，也不支持 KPI CSV 附件；
3. `notification_history` 有 `retry_count` 字段，但 `Mailer` 仍是同步单次发送；
4. 告警 XML 已有 `cnSuggestion/enSuggestion`，解析模型、数据库和 API 均未接入；
5. KPI 查询模板没有报表订阅、调度游标和每窗口运行记录；
6. 前端告警规则抽屉没有开放后端已经支持的 `notify_email` 动作；
7. 前端存在与真实能力不一致的模拟短信/通知设置页，不能把它当作本轮实现基础。

## 3. 总体设计

```text
部署配置 notification.smtp（阿里企业邮箱）
                |
                v
        共享 SMTP Transport
      TLS / AUTH / MIME / 超时
          |               |
          v               v
  告警邮件任务        KPI 报表邮件任务
  告警规则触发        查询模板订阅触发
  字典处理建议        复用 PM CSV 导出
          |               |
          +-------+-------+
                  v
       notification_history / 业务运行记录
```

设计原则：

- SMTP 是基础设施，不感知 Office365、阿里或运营商业务；阿里企业邮箱只体现在部署配置；
- 告警和 KPI 各自拥有业务配置及幂等键，不建立通用通知规则平台；
- 邮件发送必须异步，不得阻塞告警入库或 KPI 页面请求；
- 报表邮件复用现有 PM 查询与导出，不复制 KPI 公式、聚合或 CSV 生成；
- 时间窗口和邮件展示时间统一使用系统时区；数据库继续存 `timestamptz`；
- 所有新增主库结构折回 `omcgo/migrations/000001_init_schema.sql`，不新增 `000002+`。

## 4. 工作包 A：统一阿里企业邮箱 SMTP

### 4.1 配置模型

扩展 `appconfig.SMTPConfig`：

```yaml
notification:
  smtp:
    enabled: true
    host: "<由邮件管理员提供>"
    port: 465
    username: "<secret>"
    password: "<secret>"
    from: "<企业邮箱地址>"
    tls_mode: "implicit" # implicit / starttls / none
    timeout: 10s
    max_attachment_bytes: 20971520
```

约束：

- 不在代码、迁移、示例数据或日志中写真实主机、账号和密码；
- `tls_mode=implicit` 先建立 TLS 再创建 SMTP client；`starttls` 必须校验服务端能力；
- TLS 必须校验证书和 ServerName，不提供跳过校验开关；
- 启用时启动期校验 host、port、from、TLS 模式，账号是否必填由目标 SMTP 认证要求决定；
- 删除告警模块的 `OMC_SMTP_*` 独立读取，app 与 worker 统一使用 `notification.smtp.*`；
- 不出现 `office365`、OAuth、Graph API 或 provider-specific 分支。

### 4.2 共享传输与 MIME

以 `internal/notification` 为唯一 SMTP 实现：

- 支持纯文本 UTF-8、中文主题、多个收件人；
- 增加受控附件类型，首期只允许 KPI 生成的 CSV；
- 文件从对象存储流式读取并写入 MIME，禁止整份大文件无上限读入内存；
- 校验收件人地址，日志只记录脱敏地址和 SMTP 阶段，不记录认证信息和正文；
- 单次 SMTP 失败返回带上下文错误，由异步任务框架决定重试。

删除 `internal/alarm/email_dispatcher.go` 的重复实现，或将其缩成对共享 transport 的适配器；最终只能保留一份握手、认证和 MIME 代码。

### 4.3 发送记录

沿用 `notification_history`，但本轮只使用 `channel=email`：

- 每个收件地址生成一条记录，保存业务类型、业务运行 ID、单一收件人、主题、状态和错误；
- 多收件人使用逐地址 SMTP envelope 和派生幂等键；单个地址失败不阻断其他地址，重试只触达失败地址；
- 补充幂等键或业务运行关联，避免 worker 重启重复发送；
- `retry_count` 随真实尝试次数更新；
- SMTP 返回成功仅表示服务器接受邮件，不宣称收件人已经阅读或最终送达。

## 5. 工作包 B：告警邮件处理建议（#31315/#31316）

### 5.1 字典数据链

把告警定义 XML 中已经存在的字段完整接入：

```text
cnSuggestion / enSuggestion
  -> xmlAlarm
  -> alarm_definitions.cn_suggestion / en_suggestion
  -> AlarmDefinition / Registry / REST DTO
  -> 告警邮件快照
```

涉及修改：

- `internal/alarm/definition/model.go`：XML 与领域模型增加建议字段；
- `internal/alarm/definition/loader.go`：INSERT/UPSERT 增加建议字段；
- `internal/alarm/definition/pg_repository.go`、`repository.go`、`handler.go`：查询、编辑和 API 返回字段；
- `migrations/000001_init_schema.sql`：在 `alarm_definitions` 折入 `cn_suggestion`、`en_suggestion`；
- `AlarmDefinitionDrawer.tsx` 与 i18n：支持查看、编辑中英文处理建议。

处理建议与 `description`、`probable_cause` 语义不同，不能互相复用。语言规则与现有告警字典一致：中文界面优先中文、英文界面优先英文，主语言为空时回退另一语言。

### 5.2 触发与邮件内容

继续使用当前告警过滤规则作为唯一触发入口：

- 前端 `AlarmRuleDrawer` 增加“邮件通知”动作；
- 选择邮件动作时收件人必填，并做去重和格式校验；
- 后端规则匹配后创建异步 `alarm_email_delivery` 任务，不在告警接收事务中调用 SMTP；
- 任务 payload 保存告警 ID、规则 ID、语言和内容快照所需标识；
- 幂等键为 `alarm lifecycle event + rule + recipient set`，同一新增/清除事件不会重复发信。

首期邮件为后端固定结构，不开放 HTML 编辑器：

```text
告警状态：新增 / 清除
告警名称与标识
严重等级
告警源、设备名称、设备 SN
发生/清除时间（系统时区）
可能原因
处理建议
```

建议为空时显示 `-`，不把可能原因复制为建议。新增和清除邮件必须能区分，且关联同一告警生命周期。

### 5.3 不迁移的老告警逻辑

- 不重建 Quartz 和按模板动态创建 Job；
- 不增加邮件总开关 + 默认收件人 + 模板收件人叠加规则；
- 不实现 Interval/Tolerance 周期汇总，除非 Redmine 原文明确要求；
- 不开放任意模板变量、HTML 或附件；
- 不新增静默期、升级链路、联系人组或短信联动。

## 6. 工作包 C：KPI 定时报表邮件（#42492）

### 6.1 用户入口

在现有 KPI Query 的查询模板详情中增加“定时报表”配置，而不是新建报表中心。配置至少包含：

- 启用/停用；
- 周期：`15min / hourly / daily`，与 189 的 `15Min / Hour / Day` 对齐；
- 发送时间；
- 收件人邮箱；
- 最近运行状态、最近成功时间和最近错误。

不提供 FTP 复选框。公共模板只允许 super admin 配置；私有模板只允许创建者配置，读取和修改沿用模板现有权限边界。

### 6.2 数据模型

在 `000001_init_schema.sql` 中折入两张专用表：

`pm_query_report_subscriptions`：

- `id`、`query_template_id`、`owner_id`；
- `enabled`、`period`、`send_time`/发送小时；
- `recipients`；
- `next_run_at`、`last_run_at`；
- `created_at`、`updated_at`。

`pm_query_report_runs`：

- `id`、`subscription_id`、`window_start`、`window_end`；
- `status`、`export_task_id`、`notification_history_id`；
- `attempt`、`error_message`、时间戳；
- 唯一约束 `(subscription_id, window_start, window_end)`，作为停机补跑和重复调度的最终幂等保障。

不把调度配置塞入 `pm_query_templates.payload`：订阅有独立生命周期、权限、游标和运行记录，应使用明确表结构。

### 6.3 调度与窗口

- 调度器运行在 worker，按系统时区扫描到期订阅；
- 只生成已经闭合的自然窗口，不发送当前未闭合的 15 分钟、小时或天窗口；
- 调度器在一个数据库事务中插入 run 和 async job，避免“有运行记录但没任务”或相反；
- worker 停机恢复后按 `next_run_at` 和唯一窗口键补跑；
- 修改系统时区后，未执行的下一次调度重新计算，历史窗口保持原绝对时间不变；
- 模板被删除或订阅被停用时，不再生成新任务；已经运行中的任务保留审计结果。

### 6.4 复用现有导出

把 `internal/pm/export/runner.go` 中的生成逻辑抽成共享 `Generator`：

- 现有 REST 创建的 `pm_kpi_export` runner 继续调用它；
- 新的 `pm_query_report_email` runner 也调用它；
- 输入仍为 `source_type=kpi_query` 和查询模板快照；
- 输出仍写 `pm_kpi_export_tasks` 和对象存储；
- 邮件任务从对象存储读取已生成 CSV 作为附件。

禁止在邮件模块重新查询 `pm_metrics` 或重新实现 KPI 公式。`pct`、avg/sum/max/min、周/月聚合和空值处理完全沿用现有 PM 查询/导出口径。

### 6.5 一致性与失败处理

- 邮件任务本身使用 `async_jobs` 的重试、心跳和僵尸恢复；
- 导出失败：run 记为 `export_failed`，不发送空附件；
- SMTP 失败：保留已生成文件，run 记为 `delivery_failed`，重试只重发邮件，不重复计算报表；
- 重试前检查 `notification_history`/run 是否已经发送成功，避免 SMTP 成功后进程崩溃造成重复；
- 附件超过配置上限时明确失败并记录原因，不静默截断；
- 每封邮件只包含该订阅、该自然窗口的一个 CSV 文件。

## 7. 工作包 D：#62360 兼容项

#62360 原文已确认：老 `OMCMonitor` 在生成 CPE 定时邮件列表时，过去按每台 CPE 循环查询 MongoDB 的 `module_name`、`online_count`、`offline_count`；MongoDB 高负载时会导致邮件无列表，旧修复是一次批量查询后放入内存供导出使用。

处理边界：

- Redmine 原文若确认当前仍需该列表，则数据只能来自当前设备事实表：制式/模块取规范化设备类型，在线离线取 `is_online` 的互斥统计；
- 后端一次聚合返回 `total = online + offline`，邮件层不得再次按字符串或旧 CPE 字段猜测状态；
- `module_name` 使用当前产品/制式字典的稳定值，显示名在渲染层做本地化；
- 为每个目标模块补 `online + offline = total`、未知模块、空列表和混合状态测试；
- 不新增 #98018 的 2G/4G/5G 汇总、不新增独立监控 JAR、不新增定时发送入口。

该逻辑属于已经退役的外部监控程序，且 xomc 没有对应 CPE 定时邮件产品入口，本项标记“不迁移”，不在 xomc 中制造替代逻辑。

## 8. API 与前端改动

建议新增：

```text
GET    /api/v1/pm/query-templates/:id/report-subscription
PUT    /api/v1/pm/query-templates/:id/report-subscription
DELETE /api/v1/pm/query-templates/:id/report-subscription
GET    /api/v1/pm/query-templates/:id/report-subscription/runs
```

告警沿用现有规则 CRUD，只扩展 `notify_email` 的前端表单和后端校验。告警字典沿用现有定义 API，只增加 suggestion 字段。

所有新增页面、表单、toast、Modal、空态和错误信息必须走 i18n，并用真实浏览器验证请求参数、权限和回显；不能只靠组件测试判断。

## 9. 需要删除、停用或明确保留的内容

### 9.1 删除/收敛

- 删除告警模块的重复 SMTP 实现和 `OMC_SMTP_*` 独立配置源；
- 删除或隐藏可访问页面中的假短信配置、假通知规则数据；
- 不恢复本工作区清理前那套 Outbox/NATS/统一通知平台草案；
- 旧 F04 的 SMS/Webhook/DLQ/`000024` 方案已由范围说明替代。

### 9.2 保留但不改

- `docs/design/notification-center-design-20260519.md`：站内任务消息中心；
- Alertmanager -> 邮件的系统监控入口：改用共享 SMTP，但业务含义不变；
- 现有 Webhook 告警动作和死信逻辑；
- `notification_templates`/历史页面：不扩展短信能力，也不作为 KPI 报表调度入口；
- 现有 PM 导出 API、对象存储、文件管理和下载权限；
- 老项目文档目录：只读参考，不在本任务中修改。

### 9.3 明确不做

- Redmine #98018；
- Office365、OAuth、Microsoft Graph；
- 短信、Kafka 短信桥、短信回执；
- FTP 报表推送；
- 设备接入、位置移动、电子围栏通知；
- 通用事件总线、通知规则 DSL、联系人/值班组；
- 静默期、升级、重复提醒、已读回执；
- 可编辑 HTML 模板和模板发布版本；
- 新的 KPI 计算、聚合或报表引擎。

## 10. 详细开发步骤

### 10.1 依赖顺序与交付切片

开发按下列顺序推进，后置任务不得绕过前置契约自行造临时接口：

```text
S0 需求契约锁定
  |
  v
S1 共享 SMTP 传输与配置
  |---------------------+
  v                     v
S2 告警建议数据链       S3 KPI 报表订阅数据/API
  |                     |
  v                     v
S4 告警异步邮件         S5 KPI 调度/导出/邮件
  |                     |
  +----------+----------+
             v
S6 #62360 判定、旧逻辑清理、联调发布
```

建议拆成 7 个可独立评审的提交或 PR。数据库基线、后端 DTO 和前端类型必须在同一交付切片中保持一致，不允许主线出现半套字段。

### 10.2 S0：锁定需求与测试样例

#### S0-1：逐单确认范围

输入：Redmine #72785、#62360、#42492、#31315、#31316 的描述、附件和评论。

执行：

1. 把每张需求单拆成“触发条件、输入、输出、异常、验收样例”；
2. 明确 #42492 的发送时间是否多选，以及 `Day/Hour/15Min` 表示报表粒度还是调度频率；
3. 明确 #31315/#31316 对新增、清除邮件的要求和邮件语言来源；
4. 明确 #62360 对应的程序是否仍部署，数据源和样例邮件是什么；
5. 把冲突项记录到本文第 13 节，不通过猜测扩充功能。

完成标准：五张单都有一行确定的验收口径；#98018、Office365、短信仍保持排除。

#### S0-2：准备联调夹具

准备但不提交真实密钥：

- 一个阿里企业邮箱测试发件账号和白名单收件地址；
- 一条带中英文处理建议的告警 XML；
- 一条没有处理建议的告警 XML；
- 一条可触发新增和清除的测试告警；
- 一个包含 2 台设备、2 个指标的 KPI Query 私有模板；
- 15 分钟、小时、日三个闭合窗口的固定预期 CSV；
- 如 #62360 仍适用，准备能覆盖 online/offline/unknown module 的列表样例。

完成标准：开发者不连接生产邮箱即可跑完自动化测试；真实 SMTP 只用于最后联调。

### 10.3 S1：共享 SMTP 基础设施

#### S1-1：统一配置契约

修改：

- `omcgo/internal/core/appconfig/config.go`
- `omcgo/internal/core/appconfig/validate.go`
- `omcgo/internal/core/appconfig/validate_test.go`
- `omcgo/cmd/app/etc/config.{dev,test,prod}.yaml`
- `omcgo/cmd/worker/etc/config.{dev,test,prod}.yaml`

步骤：

1. 将 `SMTPConfig.StartTLS bool` 收敛为枚举 `TLSMode`：`implicit/starttls/none`；
2. 增加 `MaxAttachmentBytes`，零值使用安全默认值；
3. `enabled=true` 时校验 host、port、from、TLS mode、timeout 和附件上限；
4. 兼容旧 `starttls` 配置只允许一个发布过渡期，并在启动日志给出废弃提示；若当前版本尚未对外发布，可直接移除旧字段；
5. 示例配置只写占位符，不出现阿里真实账号、密码或固定服务地址；
6. 检查环境变量展开和 secret redaction，确保 password 不会被配置 dump 打印。

完成标准：app 与 worker 解析同一份 `notification.smtp` 契约；非法 TLS 模式在启动期失败。

#### S1-2：定义邮件领域对象和传输接口

修改/新增：

- `omcgo/internal/notification/email_sender.go`
- `omcgo/internal/notification/email_message.go`（新增）
- `omcgo/internal/notification/email_sender_test.go`

建议接口：

```go
type EmailMessage struct {
    To          []string
    Subject     string
    TextBody    string
    Attachments []Attachment
}

type Attachment struct {
    Filename    string
    ContentType string
    Size        int64
    Open        func(context.Context) (io.ReadCloser, error)
}

type EmailTransport interface {
    Send(context.Context, EmailMessage) error
}
```

步骤：

1. 在构造消息前规范化、去重并校验收件人；
2. 主题按 RFC 2047 编码，文件名同时提供安全 ASCII fallback 和 UTF-8 参数；
3. 无附件用 `text/plain`，有附件用 `multipart/mixed`；
4. 附件通过 `Open` 流式读取，累计字节超过 `MaxAttachmentBytes` 立即失败；
5. 统一 CRLF、MIME boundary、Base64 分行和 header 注入防护；
6. 禁止 caller 自定义任意邮件头，避免伪造 From/Bcc；
7. `implicit` 使用 `tls.Dialer`，`starttls` 先 EHLO 再升级，二者均校验 ServerName；
8. 每个 SMTP 阶段返回包含上下文但不含密码/正文的错误。

完成标准：同一实现覆盖纯文本告警邮件和 CSV 附件邮件，不再需要业务模块自己拼 RFC 5322 报文。

#### S1-3：扩展发送记录与幂等

修改：

- `omcgo/migrations/000001_init_schema.sql`
- `omcgo/internal/notification/history_model.go`
- `omcgo/internal/notification/history_repository.go`
- `omcgo/internal/notification/pg_history_repository.go`
- `omcgo/internal/notification/history_service.go`
- 对应测试文件

步骤：

1. 在 `notification_history` 折入 `business_type`、`business_id`、`dedup_key`、`attempted_at`；
2. 为非空 `dedup_key` 建唯一索引；
3. 增加“创建待发送记录、开始尝试、成功、失败并递增 retry_count”的原子更新；
4. 已是 `sent` 的幂等键再次执行时直接返回“已发送”，不得再次调用 SMTP；
5. 对 `recipients` 只存规范化地址；API 列表根据现有权限返回，日志使用脱敏值。

完成标准：能够区分业务任务、SMTP 尝试次数和最终结果；重复消费不会重复发信。

#### S1-4：替换两套装配

修改：

- `omcgo/cmd/app/provider/modules.go`
- `omcgo/cmd/app/provider/alarm.go`
- `omcgo/internal/alarm/email_dispatcher.go`
- `omcgo/internal/alarm/filter_engine.go`
- `omcgo/internal/notification/alert_webhook_handler.go`

步骤：

1. 在 app container 只构造一个共享 `EmailTransport`/`Mailer`；
2. Alertmanager 邮件改用新接口，行为和收件人配置不变；
3. 告警过滤引擎先通过适配器接入共享接口，待 S4 异步任务完成后移除同步调用；
4. 删除 `loadEmailConfigFromEnv()` 与 `OMC_SMTP_*` 读取；
5. `internal/alarm/email_dispatcher.go` 删除，或仅在过渡提交中保留无 SMTP 代码的适配器；
6. 启动日志只打印 enabled、host、port、TLS mode，不打印 username/password。

完成标准：仓库中只有 `internal/notification` 实现 SMTP 握手和 MIME；app/worker 配置源一致。

#### S1-5：SMTP 自动化验证

在 `email_sender_test.go` 的 fake server 基础上补：

- implicit TLS 与 STARTTLS 两套握手；
- 服务端不支持 STARTTLS；
- 证书 ServerName 不匹配；
- AUTH 成功/失败；
- 中文主题和正文；
- 多收件人去重；
- header injection；
- CSV 附件内容、文件名、Base64 分行；
- 附件超限和读取中断；
- context cancel、连接超时和 DATA 阶段失败。

完成标准：所有失败都能分类定位，测试日志中没有密码和完整邮件正文。

### 10.4 S2：告警处理建议数据链

#### S2-1：基线 schema 与领域模型

修改：

- `omcgo/migrations/000001_init_schema.sql`
- `omcgo/internal/alarm/definition/model.go`
- `omcgo/internal/alarm/definition/repository.go`

步骤：

1. `alarm_definitions` 增加可空 `cn_suggestion text`、`en_suggestion text`；
2. `AlarmDefinition`、`CreateInput`、`UpdateInput` 增加对应字段；
3. `xmlAlarm` 映射 `cnSuggestion`、`enSuggestion`；
4. 不修改 `description` 与 probable cause 的语义；
5. 不新增 `000002` 迁移。

完成标准：空建议保持 NULL/空字符串兼容，已有 XML 无需改格式。

#### S2-2：Loader、Repository 与 Registry

修改：

- `omcgo/internal/alarm/definition/loader.go`
- `omcgo/internal/alarm/definition/loader_test.go`
- `omcgo/internal/alarm/definition/pg_repository.go`
- `omcgo/internal/alarm/definition/registry.go`
- `omcgo/internal/alarm/definition/registry_test.go`

步骤：

1. Loader 的批量 INSERT/UPSERT 同时写建议字段；
2. `ListAll`、分页查询、单条查询的 SELECT/Scan 顺序一致增加字段；
3. Create/Update 写路径增加建议字段；
4. Registry 刷新后缓存完整 definition；
5. 测试 XML 中包含换行、XML entity、单语为空和双语为空；
6. 验证重新加载会更新 XML 建议，但不破坏用户可编辑 `description`。

完成标准：从 XML 加载、后台编辑、Registry 查询三条路径读到同一值。

#### S2-3：REST 与前端类型

修改：

- `omcgo/internal/alarm/definition/handler.go`
- `omcgo/internal/alarm/definition/service.go`
- `omcmb/frontend-core/src/types/alarmDefinition.ts`
- `omcmb/frontend-core/src/services/api/alarmDefinitionApi.ts`
- `omcmb/frontend-core/src/mock/data/alarmDefinition.ts`
- `omcmb/frontend-core/src/mock/services/alarmDefinitionService.ts`

步骤：

1. create/update DTO 和 `defView` 增加 `cn_suggestion/en_suggestion`；
2. 前端 domain type、backend wire type、mapper 和 payload 同步增加字段；
3. handler 测试覆盖 create/get/update/list 四条路径；
4. 保持 snake_case wire 与 camelCase 前端域模型边界。

完成标准：字段不会出现“后端有值、前端 mapper 丢失”的情况。

#### S2-4：告警字典页面

修改：

- `omcmb/webcode/src/pages/product/alarm-library/AlarmDefinitionDrawer.tsx`
- 对应测试与中英文 i18n 资源

步骤：

1. 在“可能原因”之后增加中英文“处理建议”多行输入；
2. view/add/edit 三种模式均正确回显；
3. 保留换行，不允许把建议自动复制到可能原因；
4. 错误、必填提示和保存 toast 全部使用 i18n；
5. 真实浏览器验证 GET/PUT payload 和刷新后回显。

完成标准：编辑 XML 导入项或手工项后，Registry 刷新与页面值一致。

### 10.5 S3：KPI 报表订阅数据与 API

#### S3-1：锁定 REST 契约

建议 wire DTO：

```json
{
  "enabled": true,
  "period": "hourly",
  "send_times": ["08:00", "20:00"],
  "recipients": ["noc@example.com"]
}
```

响应额外包含 `id`、`query_template_id`、`next_run_at`、`last_run_at`、`last_status`、`last_error`。`period` 只允许 `15min/hourly/daily`；`send_times` 的多选规则在 S0 锁定后固化校验。

完成标准：前后端先以契约测试固定字段名、枚举、空订阅的 404/空响应语义，再写页面。

#### S3-2：数据库结构

修改 `omcgo/migrations/000001_init_schema.sql`：

1. 创建 `pm_query_report_subscriptions`；
2. 创建 `pm_query_report_runs`；
3. `query_template_id` 使用级联删除或显式 service 删除，开发前二选一并写测试；建议订阅级联删除、run 保留 template/run 快照；
4. recipients 使用 `text[]`，发送时间使用能表达多选的 `time[]`；
5. 对 period、status、attempt、时间数组合法性增加 CHECK；
6. 唯一约束 `(query_template_id)` 保证一个模板最多一份订阅；
7. 唯一约束 `(subscription_id, window_start, window_end)` 保证自然窗口幂等；
8. 为 due scan 建 `(enabled, next_run_at)` 部分索引，为历史列表建 `(subscription_id, created_at desc)` 索引。

完成标准：全新数据库基线可一次建成，约束能拒绝非法 period、空收件人和重复窗口。

#### S3-3：Repository 与 Service

新增目录建议：`omcgo/internal/pm/reportsubscription/`，包含：

- `model.go`
- `repository.go`、`repository_test.go`
- `service.go`、`service_test.go`
- `schedule.go`、`schedule_test.go`

步骤：

1. SQL 全部使用 Squirrel + pgx；
2. Repository 提供 GetByTemplate、Upsert、Disable/Delete、ListRuns、ClaimDue、CreateRunAndJob；
3. `CreateRunAndJob` 在一个 pgx 事务内插入 run 和 `async_jobs`；
4. Service 复用 querytemplate 的 `canRead/canWrite` 语义，或提取共享授权函数，禁止复制出不同权限规则；
5. 规范化、去重 recipients，拒绝空地址和超过上限的地址数；
6. 更新订阅时重新计算 `next_run_at`，停用时置空；
7. run 保存模板查询 payload、locale 和窗口快照，避免模板后改影响已触发报表。

完成标准：并发两个 scheduler 只能创建一个窗口 run；私有/公共模板权限与模板 CRUD 完全一致。

#### S3-4：Handler 与路由装配

修改/新增：

- `omcgo/internal/pm/reportsubscription/handler.go`
- `omcgo/cmd/app/provider/pm.go`
- `omcgo/cmd/app/provider/router.go`（若现有 PM handler 不是自注册）

实现：

```text
GET    /api/v1/pm/query-templates/:id/report-subscription
PUT    /api/v1/pm/query-templates/:id/report-subscription
DELETE /api/v1/pm/query-templates/:id/report-subscription
GET    /api/v1/pm/query-templates/:id/report-subscription/runs
```

步骤：

1. GET 必须先验证模板读权限；
2. PUT/DELETE 必须验证模板写权限；
3. 非 owner 私有模板返回 403，未知或不可见模板不泄露额外信息；
4. runs 分页并限制最大 page size；
5. 错误包装上下文，响应不返回 SMTP 内部堆栈或收件人密钥。

完成标准：handler 测试覆盖 400/401/403/404/409/200 全部主要分支。

#### S3-5：前端 API、Hook 与表单

修改：

- `omcmb/frontend-core/src/types/pmQuery.ts`
- `omcmb/frontend-core/src/services/api/pmQueryApi.ts`
- `omcmb/frontend-core/src/hooks/api/usePmQuery.ts`
- `omcmb/webcode/src/pages/performance/KPIQuery/QueryTemplateDetailModal.tsx`
- 新增 `ReportSubscriptionForm.tsx` 及测试
- 中英文 i18n 资源

步骤：

1. 独立定义 subscription/run 类型，不塞入 QueryTemplate payload；
2. React Query key 采用 `['pm-query-templates','report-subscription',templateId]` 和 `report-runs`；
3. Modal 增加“定时报表”区域和编辑入口；
4. 只有可写用户显示保存/停用按钮，只读用户仍可看运行状态；
5. 邮箱输入支持分号/换行粘贴，在提交前转成去重数组；
6. 成功后只失效当前模板的 subscription/run query，不清空全部 KPI 查询缓存；
7. 浏览器检查真实 PUT body 和重新打开后的回显。

完成标准：公共/私有模板的按钮和后端权限一致，刷新页面不丢配置。

### 10.6 S4：告警异步邮件

#### S4-1：邮件内容快照

新增建议：

- `omcgo/internal/alarm/email_message.go`
- `omcgo/internal/alarm/email_message_test.go`

步骤：

1. 定义 `AlarmEmailSnapshot`，包含 alarm lifecycle ID、状态、规则 ID、收件人、设备、时间、可能原因、处理建议和 locale；
2. 建快照时从 Registry 读取 suggestion，按 S0 锁定的 locale 规则回退；
3. subject/body 使用纯函数渲染，固定字段顺序；
4. 新增、清除使用不同状态文案，但共享同一 alarm lifecycle 标识；
5. 邮件正文不实时反查已变化的告警定义，保证重试内容不漂移。

完成标准：给定快照能生成稳定 golden 文本；建议为空显示 `-`。

#### S4-2：入队与幂等

修改/新增：

- `omcgo/internal/alarm/filter_engine.go`
- `omcgo/internal/alarm/filter_model.go`
- `omcgo/internal/alarm/email_job.go`
- 对应 repository/service 测试

步骤：

1. 给 FilterEngine 注入最小 `EmailJobEnqueuer`，不直接依赖具体 asyncjob repository；
2. 规则命中后序列化快照并插入 `job_type=alarm_email_delivery`；
3. dedup key 使用 `alarm lifecycle event + rule ID + normalized recipient hash`；
4. 告警保存成功后才允许入队；入队失败记录错误和指标，但不回滚告警；
5. 清除事件是否发送严格按 S0 需求，不默认扩展；
6. 移除 `buildEmailSubject/buildEmailBody` 同步发送路径。

完成标准：SMTP 不可达时告警仍正常进入 active/history；重复告警事件只产生一条投递记录。

#### S4-3：Worker Runner

新增/修改：

- `omcgo/internal/alarm/email_runner.go`
- `omcgo/internal/alarm/email_runner_test.go`
- `omcgo/cmd/worker/main.go` 或独立 `email.go`
- worker SMTP 装配测试

步骤：

1. Runner 解析快照并查 `notification_history` 幂等状态；
2. 创建/更新尝试记录后调用共享 `EmailTransport`；
3. 成功 MarkSent，失败返回 error 交给 asyncjob 最多三次重试；
4. 每次重试前检查 sent，避免“SMTP 已接受但进程在 MarkSent 前崩溃”导致明显重复；该极端窗口仍需在风险章节说明 SMTP 本身没有全局幂等协议；
5. 注册独立 job type worker 和队列深度指标；
6. worker 未配置 SMTP 时任务明确失败，不静默丢弃。

完成标准：重启 worker 能恢复 pending/running zombie 任务；发送成功后重复执行 runner 不再发信。

#### S4-4：告警规则前端

修改：

- `omcmb/webcode/src/pages/alarm/AlarmRules/AlarmRuleDrawer.tsx`
- `omcmb/webcode/src/pages/alarm/AlarmRules/AlarmRuleDrawer.test.tsx`
- `omcmb/webcode/src/pages/alarm/AlarmRules/index.tsx`
- `omcmb/frontend-core` 的 alarm rule API/type（按实际 mapper 文件）
- i18n 资源

步骤：

1. 动作列表加入 `notify_email`；
2. 仅该动作显示收件人输入；
3. add/edit/view 都能回显 `email_recipients`；
4. 切换到其它动作时清除或不提交邮件字段，避免旧值污染；
5. 提交前格式校验、去重并限制数量；
6. 真实浏览器创建一条禁用规则，检查请求后立即删除测试数据。

完成标准：前端可配置后端已有的邮件动作，没有硬编码中文。

### 10.7 S5：KPI 调度、导出与邮件

#### S5-1：自然窗口计算

在 `reportsubscription/schedule.go` 实现纯函数：

```go
NextRun(subscription, now, location) time.Time
ClosedWindow(period, scheduledAt, location) (start, end time.Time)
```

步骤：

1. 15min 对齐 00/15/30/45，hourly 对齐整点，daily 对齐业务时区本地零点；
2. 永远取最后一个已闭合窗口；
3. 覆盖 Asia/Shanghai、UTC、夏令时跳时/重复时区；
4. 系统时区变化只重算 next run，不改历史 run 的绝对窗口；
5. S0 若确认 send_times 多选，NextRun 取严格晚于 now 的最近 slot。

完成标准：边界表驱动测试全部通过，不使用容器本地时区或 `time.Local`。

#### S5-2：Due Scheduler

新增/修改：

- `omcgo/internal/pm/reportsubscription/scheduler.go`
- `omcgo/internal/pm/reportsubscription/scheduler_test.go`
- `omcgo/cmd/worker/pm_streaming.go` 或独立装配文件

步骤：

1. 复用 system timezone provider/tz manager；
2. 周期扫描 `enabled=true AND next_run_at<=now()`，使用 `FOR UPDATE SKIP LOCKED`；
3. 同一事务创建 run、async job，并推进 subscription.next_run_at；
4. 唯一窗口冲突按幂等成功处理；
5. 启动时扫描过期 `next_run_at`，直接折叠到最近一个已闭合自然窗口并一次推进游标到未来，避免长停机逐窗补发形成邮件风暴；
6. 记录 due 数、created run 数、dedup 数和失败数。

完成标准：两个 scheduler 并发运行不会重复建 run；停机跨过一个窗口后可补一封，不无限追发历史邮件。

#### S5-3：抽取完整导出编排器

当前 `internal/pm/export/generator.go` 只提供 `streamCSVToObject`，`Runner.generate` 仍包含选源、对象路径、存储保护和状态编排。实施时：

1. 新增可复用 `export.Generator` 结构，封装 build source、storage admission、object path 和流式上传；
2. 将 `Runner.generate` 迁入 `Generator.Generate`；
3. 现有 `pm_kpi_export` Runner 通过新 Generator 保持行为不变；
4. 为 `source_type=kpi_query` 增加从订阅 run 快照构造 export params 的专用函数；
5. 保留现有 `streamCSVToObject`、CSV writer、指标名解析和系统时区输出；
6. 先跑现有 export 全套测试，确保普通手工导出没有回归。

修改：

- `omcgo/internal/pm/export/generator.go`
- `omcgo/internal/pm/export/runner.go`
- `omcgo/internal/pm/export/runner_test.go`
- `omcgo/internal/pm/export/generator_test.go`

完成标准：手工导出和定时报表调用同一个 Generator，代码中没有第二套 PM 查询 SQL。

#### S5-4：报表邮件 Runner

新增：

- `omcgo/internal/pm/reportsubscription/runner.go`
- `omcgo/internal/pm/reportsubscription/runner_test.go`

步骤：

1. job payload 只带 `report_run_id`；
2. Runner 加载 run 的查询模板快照和窗口，创建 `pm_kpi_export_tasks` 记录；
3. 使用共享 Generator 生成 CSV 并回填 export task；
4. 保存 `export_task_id` 后，再从 MinIO `GetObject` 打开流式附件；
5. 以 `report-run:<id>` 作为邮件 dedup key 调共享 Mailer；
6. 导出失败记 `export_failed`，不调用 SMTP；
7. SMTP 失败记 `delivery_failed` 并返回 error；重试时发现 export 已成功则直接复用对象，不重算；
8. 成功写 run=`sent`、history ID、finished_at；
9. 模板名称和窗口生成安全附件名，禁止路径分隔符和控制字符。

完成标准：SMTP 临时失败后只看到一次导出文件、最多多次 SMTP attempt；最终邮件附件内容与 export task 下载内容一致。

#### S5-5：KPI 端到端浏览器验证

1. 创建私有 KPI Query 模板；
2. 配置禁用状态订阅并检查 GET/PUT；
3. 启用测试订阅，确认 next_run_at 使用系统时区；
4. 触发 15min/hourly/daily 固定窗口各一次；
5. 对照页面查询的真实 request params 与 run 快照；
6. 下载 export task CSV，与邮件附件做 SHA-256 对比；
7. 用另一个普通用户验证 403；
8. 删除测试订阅和模板，不保留测试收件人。

完成标准：三种窗口、权限、文件一致性和清理全部有截图或测试记录。

### 10.8 S6：#62360、旧逻辑清理和发布资料

#### S6-1：#62360 分支决策

若 Redmine/部署清单确认程序已退役：

1. 在需求映射和发布说明中写“不迁移”；
2. 不新增 module/count API、定时器或邮件；
3. 用现有设备统计能力作为替代说明，但不宣称等价验收。

若确认仍需迁移到 xomc：

1. 在 `internal/device` 增加专用只读聚合 DTO/repository 方法；
2. SQL 用规范化 module/technology 分组，并以 `is_online` 双值统计；
3. 单条 SQL 返回 total、online、offline，service 校验三者恒等；
4. 邮件只消费 DTO，不自行重算；
5. 测试空列表、未知 module、混合状态和并发状态更新；
6. 不创建 #98018 的新邮件触发器。

完成标准：只有在需求原文和当前部署都证明需要时才产生代码改动。

#### S6-2：清理模拟与过度入口

核对：

- `omcmb/webcode/src/pages/system/NotificationSettings/index.tsx`
- `omcmb/webcode/src/pages/system/SystemConfig/NotificationSettings.tsx`
- `omcmb/webcode/src/pages/notifications/TemplateForm.tsx`
- `omcmb/webcode/src/pages/notifications/TemplateList.tsx`
- `omcmb/webcode/src/pages/notifications/HistoryList.tsx`

步骤：

1. 先查路由和菜单，确认页面是否可达；
2. 可达的纯 mock 通知规则/阿里云短信密钥表单删除或从路由移除；
3. 真实的 notification template/history 页面保留邮件能力；SMS/Webhook 选项若后端没有可用发送链路则隐藏，不删除历史数据库枚举；
4. 不改站内消息中心 `MessageList`；
5. 补路由测试，确保旧 URL 不再展示假成功数据。

完成标准：用户看不到本轮未实现的短信配置或假规则，同时已有邮件历史仍可查询。

#### S6-3：运维文档与发布门禁

新增/更新 runbook：

1. 阿里企业邮箱配置项、secret 注入和 TLS 模式；
2. DNS/网络/TLS/AUTH/MAIL FROM/附件错误排查顺序；
3. 告警规则与 KPI 订阅的启停步骤；
4. `notification_history`、`async_jobs`、report runs 的查询示例；
5. 首次联调只使用测试邮箱；
6. 回退时先停订阅/规则，再关闭 SMTP，不停止告警或 PM worker；
7. 明确 Office365、短信、FTP 和 #98018 不在本版本。

发布门禁：后端测试、前端测试、迁移基线重建、真实浏览器、测试 SMTP 和安全日志检查全部通过后才能启用生产配置。

### 10.9 每个切片的完成定义

每个提交/PR 同时满足：

- 代码、schema、API、前端类型和文档无半成品契约；
- 新增共享逻辑有单元测试，用户可见流程有浏览器记录；
- SQL 使用 Squirrel + pgx，新增字段折入 `000001`；
- 错误使用 `%w` 包装上下文，日志无密码、token、完整收件人和正文；
- 用户可见文字全部走 i18n；
- 不恢复 V2/V3 皮肤、Office365、短信、FTP 或 #98018；
- `git diff --check`、相关 Go 测试和前端测试通过；失败项如实记录命令与原因。

## 11. 测试与验收

### 11.1 后端

- SMTP：隐式 TLS、STARTTLS、证书名、AUTH 失败、超时、多个收件人、中文、CSV 附件和大小限制；
- 告警字典：XML suggestion 解析、UPSERT、语言回退、编辑、未知告警；
- 告警邮件：规则匹配、收件人校验、新增/清除、处理建议、幂等、重试不阻塞告警；
- KPI：自然窗口边界、系统时区变化、停机补跑、模板权限、唯一窗口、导出失败、SMTP 失败后只重投不重算；
- PM 口径：邮件 CSV 与 KPI Query 页面同参数查询结果逐行一致。

建议验证命令在实施时按实际包补齐，至少包括：

```bash
go test ./internal/notification/... ./internal/alarm/... ./internal/pm/export/... ./internal/pm/querytemplate/... ./internal/core/asyncjob/...
```

### 11.2 前端与浏览器

- 告警规则可选择邮件并正确回显收件人；
- 告警字典可编辑/展示中英文处理建议；
- KPI 查询模板可启停订阅、配置周期/发送时间/收件人并查看运行结果；
- 非 owner 无法改私有模板订阅，非 super admin 无法改公共模板订阅；
- 所有可见文案中英文正确，无硬编码；
- Network 面板确认 period、send time、recipient、template ID 与后端契约一致；
- 实际收到的邮件主题、时间、处理建议和 CSV 与页面一致。

### 11.3 阿里企业邮箱联调

联调前由邮件管理员提供测试账号、SMTP 主机、端口、TLS 模式、发件人和收件人白名单。先在测试邮箱完成：

1. 网络和 TLS 握手；
2. 认证和发件人授权；
3. 中文告警纯文本邮件；
4. 小型 KPI CSV 附件；
5. 错误密码、不可达主机和附件超限；
6. 重试期间不重复生成告警或 KPI 文件。

不使用生产邮件组做首次测试，不在日志或截图中暴露密码。

## 12. 发布与回退

- 发布前保持 `notification.smtp.enabled=false`，部署人员填完阿里企业邮箱 secret 后再启用；
- 先启用测试收件人和一条低风险告警规则，再启用 KPI 订阅；
- 观察发送成功率、错误分类、异步队列深度和最老 pending 年龄；
- 回退只需停用告警邮件规则和 KPI 订阅，或关闭 SMTP 总配置，不影响告警入库和 KPI 查询；
- 数据结构是 `000001` 基线的一部分，未封版阶段不提供 `000002` down 迁移；测试环境回退按整套基线重建流程执行。

## 13. 外部联调前仍需确认

需求单和老 OMC 流程已核对完成。生产启用前只剩部署侧参数：

1. 阿里企业邮箱测试账号的 SMTP 主机、端口、TLS 模式和附件上限；
2. 发件人授权、测试收件人白名单和网络出站策略；
3. KPI 邮件附件最大允许值是否沿用当前默认 20 MiB。

若 Redmine 原文与本文冲突，以用户最新确认和需求单验收标准为准，但不得把 #98018、Office365 或短信重新带回范围。

## 14. 2026-08-12 实施与验证记录

已完成：

- app/worker 统一 `notification.smtp`，支持 implicit TLS、STARTTLS、纯文本 MIME 和受限 CSV 附件；
- 告警处理建议从 XML、数据库、API、前端字典到邮件正文贯通；
- 告警邮件改为持久化异步任务，接入发送历史、幂等和重试；
- 告警规则前后端开放 `notify_email`，收件人最多 50 个、规范化去重，非邮件动作清空地址；
- KPI Query 报表订阅、系统时区调度、停机补跑、运行记录、导出复用和失败重试完成；
- 删除两处不可达且与真实能力不一致的模拟 Notification Settings 页面；
- 新增本运维手册并补充老新业务流程映射。
- 告警规则生效时间贯通基线表、CRUD API、规则引擎和编辑回填，采用 `[开始, 结束)` 语义；
- 通知历史查询收紧到超级管理员，普通 `Alarm.View` 用户不再能读取完整收件人和正文；
- 邮件异步 pending 接管阈值调整为 4 分钟，短于通用任务 5 分钟 zombie 恢复阈值；
- 告警与 KPI 邮件改成逐收件人投递、审计和幂等，部分失败时保留成功结果并只重试失败地址；
- KPI 停机恢复折叠到最近一个闭合自然窗口，游标一次推进到未来，不逐窗追发历史邮件。
- 通知历史的发送尝试改为条件 SQL 原子抢占：只有 `failed` 或超过 4 分钟的 `pending` 可接管，防止两个 worker 同时重试同一幂等键时重复进入 SMTP。

已验证：

- 本次涉及的 `internal/notification`、`internal/alarm`、`internal/pm/reportsubscription` 测试全部通过；
- 后端 `go test ./...` 并发全跑时出现两个 `paramsync` 数据库时序用例和一个 `pm/stream` 短租约用例抖动，三个失败用例脱离并发负载单独 `-count=1` 重跑均通过；
- 前端 TypeScript 类型检查通过；
- 告警规则抽屉、KPI Query 和 alarm API 针对性测试通过；
- app、worker 和 V1 前端生产构建通过；
- 前端全量 Vitest 已改为单 worker 串行复跑，293 个测试文件、1973 个用例全部通过；告警规则抽屉专项 10 个用例在最后一次 UI 收口后再次通过；
- `git diff --check` 通过；
- 空库 `000001` 建库和旧基线数据库 MainReconcile 已通过，新增约束和外键已核对；
- 本地隔离环境使用真实 app/worker、PostgreSQL/TimescaleDB、Redis、NATS 和 MinIO；向 `device.inform.alarm` 发布“产生 → 重复上报 → 清除”三次事件，只形成两封邮件，活动库归零、历史告警归档，raised/cleared 共用同一 `alarm_id` 且各有独立幂等键；
- 本地隔离调度器真实创建 KPI 自然窗口 run、CSV 导出任务和 MinIO 对象，随后发送 multipart CSV 附件并把 run 与 `notification_history` 标记为 `sent`；
- 本地 SMTP 捕获器确认告警主题固定为“告警通知/Alarm Notification”，新增/清除正文均包含描述、可能原因、处理建议、发生时间和清除时间，时间遵循 `Asia/Shanghai`；
- 本地生产构建通过真实浏览器加载后进入 License 恢复页；当前共享开发库未安装 License，因此告警/KPI 受 License 保护的真实 HTTP 页面不能作为本轮 UI 验收依据。后端真实业务链路改由 app/worker、消息队列和数据库结果完成验证，不把 License 阻断误报成业务失败；
- 真实浏览器连接本地 V1 Mock，按“告警管理 → 告警规则 → 新增 → 邮件通知”完成无保存验证，确认收件人、设备/设备组、告警筛选和时间范围展示正常；同时修复静态菜单目录/子项重复 key、告警规则页静态 message 上下文和 `Space.direction` 弃用告警，刷新后控制台错误/警告为 0；
- 本轮修复后再次以真实浏览器连接本地 V1：编辑规则、填写并保存 `2026-08-12 18:00:00–19:00:00`，重新打开后开始/结束时间均正确回填；切换“邮件通知”后收件人输入和最多 50 个提示正常显示；
- 真实浏览器在 189 老 OMC 13.0.5 完成告警通知、模板条件、模板邮件和 KPI Regular Report 的只读走查；所有页面均取消退出，未保存、未发送邮件，并确认新项目入口映射与本方案一致。
- 当前实现重新构建 app、worker、web 后健康检查通过；空白 PostgreSQL 16 容器从三个 `000001` 基线完成 schema/seed/TSDB 建库，seed 再执行无重复数据，旧库 MainReconcile 也完成；
- 真实 worker 消费 `device.inform.alarm` 后完成告警邮件任务：一个收件人成功、一个模拟拒收时成功地址只投递一次，失败地址重试三次；XML 告警定义中的中文名称、可能原因和处理建议均进入邮件正文；
- KPI 调度器生成闭合的 15 分钟自然窗口，复用 CSV 导出并从 MinIO 读取 66 字节 UTF-8 附件完成 multipart 投递；附件上限降为 4 字节后导出仍成功、投递标记为 `delivery_failed` 并重试三次，状态边界正确；
- SMTP 认证配置异常时任务失败并重试三次，日志未出现测试密码、完整收件人或正文；由于没有阿里企业邮箱凭据，本项只证明失败安全和脱敏，不宣称已通过真实账号认证；
- 使用同一签名密钥生成非超级管理员 token 请求通知历史接口，返回 403 `super_admin role required`，确认收件人和正文不会暴露给普通用户；
- 本轮 E2E 创建的固定测试设备、告警定义/规则、KPI 模板/订阅/run、通知历史、异步任务、Redis key 和两个 MinIO CSV 均已按固定 ID 清理，清理后数据库和缓存计数为 0。

2026-08-13 收口复验：

- `internal/notification` 以 `-race -count=1` 通过，新增双并发失败重试回归用例，确认仅一次 SMTP 调用、仅一次 `retry_count` 增量；
- 告警、KPI 订阅、配置、app provider 和 worker 相关测试全部通过，相关包 `go vet` 通过；
- `go test -run '^$' ./...` 完成整个 Go 模块跨包编译检查，未发现历史仓储接口改名引起的断裂；
- 前端类型检查、改动文件 ESLint 和生产构建通过，全量 293 个测试文件、1973 个用例再次串行全绿；
- 本地 V1 Mock 已重新启动在 `127.0.0.1:3300`；应用内浏览器对该地址的自动重载被 URL 安全策略拒绝，因此本次抽屉重开和 KPI 页面的最终交互回归没有冒充为已通过；可人工刷新已保留的本地页面继续验证。

仍待外部条件：

- 真实阿里企业邮箱账号、白名单收件人和附件上限到位后完成 SMTP 联调；
- Redmine #72785 已核对：旧单曾要求改用 Office365 STARTTLS；按用户 2026-08-12 最新确认反向替换为阿里企业邮箱，Office365 不进入实现或部署配置；
- Redmine #42492 已核对：KPI 定时报表必须走任务/worker 异步发送并携带附件，多机部署不能重复发送；
- Redmine #31315/#31316 已核对：邮件固定标题为“告警通知/Alarm Notification”，正文包含告警描述、发生时间、清除时间和处理建议，新增/清除使用同一内容结构；
- Redmine #62360 已核对并判定不迁移；#98018 始终排除。
