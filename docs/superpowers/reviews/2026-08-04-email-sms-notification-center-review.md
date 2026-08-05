# OMC 邮件短信通知中心复评记录

## 结论

方案按修正后条件通过，可以进入里程碑 A 的小步开发。保留“主库告警事务 + Outbox +
独立 JetStream + 多 durable + 通知编排 + 渠道 Worker”主链路；邮件先试点，SMS 只冻结
Adapter 边界，不在外部协议未确认时引入 Kafka 或供应商 SDK。

本次复评不是重新设计业务，而是把老 OMC 的实际语义、当前代码写路径和最新权限模型
逐一对齐。复评前的计划有六项会造成可靠性或领域边界问题，必须在编码前修正。

## 证据对齐

| 证据 | 必须保留或改进的含义 |
| --- | --- |
| 老 OMC 告警模板混合设备范围、告警条件、周期/容忍时间、邮件开关和收件人 | 新系统拆成通知规则、正文模板、收件人和调度策略，但提供可解释迁移预览 |
| 老 OMC 有确认、取消确认、清除、Alarm Count 和更新时间 | canonical 事件必须覆盖五类生命周期变化，重复告警更新 occurrence 而不是反复创建首次通知 |
| 老 SMS 实际是 Redis → Kafka → 外部平台 | Kafka Ack 只能标记 handoff，不能展示为短信送达 |
| 老 SMTP 连续鉴权失败仍重复发送 | 新 Worker 必须区分临时/永久错误、退避、熔断并形成逐收件人审计 |
| 当前 `AlarmEngine` 在持久化前可同步发邮件，多个写路径直接改 store | 通知必须移出告警事务；所有产生、更新、确认、取消确认和清除统一经过生命周期事务 |
| 当前活动告警在 PostgreSQL、历史在 TimescaleDB | 清除事实与主库 Outbox 原子提交，历史改成可重放投影，不能宣称跨库原子 |
| 最新权限是角色授予的设备组 + 制式 | 规则管理、收件人解析和历史查询必须使用完整 visibility grants；carrier 不是租户权限 |
| 备份/磁盘/渠道故障没有受管网元 `device_id` | 保持独立 system incident/health 域，不能伪造设备塞进网元告警表 |

## 强制修正项

1. canonical payload 使用显式稳定快照，不直接序列化内部 `model.Alarm`；生命周期类型和
   设备告警事件类型分别命名，变更字段使用受控枚举。
2. `alarm_version` 在主库事务持锁后分配，Outbox 对
   `(aggregate_id, aggregate_version)` 加唯一约束；严重级别更新携带旧级别。
3. `occurred_at` 来自实际事务变更时间；`auto_clear` 新告警产生 raised v1 与 cleared v2，
   不允许生命周期从清除事件开始。
4. 数据权限调用 `GetUserVisibleDeviceGrants` 并同时判断设备组和制式；carrier 只参与规则
   匹配。
5. 备份失败、磁盘阈值和通知渠道故障暂不迁入 AlarmEngine。未来如需统一展示，应先建立
   有独立身份、状态和权限模型的 system alarm/incident 域。
6. DOMAIN_ALARM 的时间保留和字节保留分别治理。硬上限可配置，并以峰值容量、durable lag、
   最老消息和提前淘汰告警作为 canonical 上线门禁。

## 修正后的实施顺序

1. 先定义纯契约与独立 Stream，保持 Relay 关闭，不改变生产告警行为。
2. 折回基线 schema，建立版本和 Outbox 唯一性；在事务存储内分配版本并构造事件。
3. 用 `legacy|shadow|canonical` 单一模式贯穿 Relay、历史投影和 legacy 发布；默认 legacy。
4. Shadow 验证五类生命周期、重复计数、乱序/重复、NATS 中断补发和容量。
5. 标准历史投影及北向 durable 通过后再切 canonical；系统事件不随本次切换迁移。
6. 里程碑 A 稳定后才建设通知 Inbox、规则、模板、调度和逐收件人 Email Worker。
7. 完成真实浏览器、故障注入、容量和回退演练后才能分范围替换 legacy `notify_email`。

## 开发门禁

- 每个任务先 RED 测试，再做最小 GREEN，不并行修改无关模块。
- Task 1 仅增加领域契约和 Stream 定义，不启用 Relay、不改变现网事件发布。
- 任何涉及清除路径的代码在 projector 未就绪前必须保持 legacy 行为。
- 未取得 Kafka Topic/Schema/Ack/鉴权或直连供应商回执契约前，不实现 SMS 发送器。
- 所有页面和真实请求参数必须在 V1 前端用浏览器验证；老 OMC 始终只读。

## Task 9 实施复核（2026-08-05）

Task 9 已按“管理面闭环、发送面不提前扩展”的边界完成：

- 规则按 draft / published / enabled 三个指针管理，不可变版本同时快照匹配条件、策略、
  收件人和渠道；发布不自动启用，新模板草稿或发布也不自动改变已启用规则。
- 匹配保持字段内 OR、字段间 AND，并按 priority、具体度和稳定 UUID 决胜；同一
  channel + fingerprint 只保留最高优先级候选。
- 非超管规则写入必须明确设备 ID，并使用同一 visibility grant 同时满足设备组和制式；
  broad rule 仅超管可写。`carrier` 只参与匹配，没有被误用为用户或租户权限。
- 用户、角色和联系组在事件时解析；禁用用户、空地址和无设备权限均输出排除原因。固定
  联系人仅允许已保护的 ciphertext / key version / fingerprint，所有查询响应只显示
  `address_configured`，不回显密文或地址。
- 模板新写路径使用不可变版本和严格 `missingkey=error`；变量来自白名单，HTML 使用
  `html/template`，SMS 预览返回预计分段数且不静默截断。旧模板 HTTP 写路由在生产装配中
  已关闭，只保留兼容读；旧 Alertmanager webhook 内部链路暂不改写。
- 渠道参数拒绝嵌套 password、token、secret、private key 等敏感键，响应只显示
  `secret_configured`。数据库唯一索引保证至多一个启用邮件渠道、至多一个默认联系组。
- 所有新管理路由继续经过现有 JWT、endpoint RBAC、AuditLogger 和 OperLogger。两类日志
  都不读取请求体，秘密值不会进入审计。真实连接验证尚无适配器时返回 503，而非伪造成功。

没有在本任务引入 Kafka、短信 SDK、后台发送 worker、模板 AB、可视化规则编辑器或新的
权限/审计子系统。第一阶段仍只允许 Email；固定联系人明文录入必须等正式 key provider，
SMS 必须等外部平台契约。规则 preview 当前负责命中解释，模板 preview 负责严格渲染；事件
时的最终权限求交、收件人数和投递编排由 Task 10 复用本任务的 resolver 完成。

验证证据：

- `go test ./internal/notification -run 'TestPg(Rule|ContactGroup|TemplateManagement|ChannelConfig)Repository_Integration' -count=1 -v`：PASS。
- `go run ./cmd/migrate ... --path migrations up` 与独立 seed 版本表：全新库 schema + seed PASS。
- `go test ./... -count=1`：PASS。
