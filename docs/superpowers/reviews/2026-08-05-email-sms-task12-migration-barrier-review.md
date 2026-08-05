# 邮件短信通知 Task 12 迁移与 Barrier Review

## 结论

Task 12 的后端迁移门禁已完成，改动只覆盖当前项目 `alarm_filters.notify_email` 到独立通知规则的
安全迁移，没有增加通用迁移平台、前端页面、短信逻辑或新的迁移表。当前可以生成只读预览和
默认禁用的候选规则；只有兼容项确认、有效 Shadow 样本一致、候选规则明确 enabled 且仍与旧
规则一致时，才允许把旧规则原位置替换成不发送的 compatibility barrier。

里程碑 B 的现场业务验收仍未完成。没有可控告警样本和测试 SMTP 时，不把单元/PG 测试写成
“真实邮件链路已验收”。

## 老 OMC 与当前项目的边界

老 OMC 文档确认邮件链路是 `alarm_view_template` + Quartz + `AlarmEmailNoticeService` + SMTP，
包含周期、延迟、设备组/设备范围、活动/清除告警和逐地址发送；`Tolerance Duration` 的现场语义
仍不能只靠字段名确定。当前项目的存量入口则是告警持久化前同步执行的
`alarm_filters.notify_email`，并采用严格的优先级首条命中。

因此本次没有假装把老 OMC 的 `alarm_view_template` 直接自动导入当前模型：

- 当前 `alarm_filters` 没有 Tolerance Duration 字段，切换必须显式确认该语义；
- 新规则模型尚不能无损表示 `alarm_sources` 和 `device_group_ids`，预览会列为不支持范围并硬阻断，
  不生成可能扩大命中面的候选规则；
- `notify_webhook` 不参与迁移；
- 老 OMC 历史是批次语义，未伪造成新系统逐收件人投递事实。

这些限制符合通信行业 OMC 的变更原则：宁可保持旧链路并输出不兼容项，也不允许静默扩大网元
范围、遗漏告警或重复通知 NOC。

## 核心不变量

1. 预览按旧仓储顺序 `priority, created_at, id` 重建 FirstMatch 顺序，并保守列出所有可能重叠的
   低优先级规则。
2. 新候选规则使用旧 FirstMatch 排名生成唯一正优先级，解决旧规则允许 0/负数而新规则只允许
   正数的问题，同时保持旧邮件规则间的稳定顺序。
3. 固定邮箱在写入候选规则前使用现有 recipient protector 加密；候选规则仅有 draft，默认不
   enabled。
4. 切换要求重叠规则 ID 集合逐项相等、Tolerance 语义已确认、Shadow 有样本且命中数相等、
   enabled 版本的范围/收件人数/邮件渠道/迁移标记仍与旧规则一致。
5. PG 切换更新同时校验旧规则未变化、仍为 enabled `notify_email`，并再次校验目标通知版本仍
   enabled；条件漂移时不写 barrier。
6. barrier 返回 `Handled=true` 保持首条命中，但不发邮件、不修改告警；AlarmEngine 仍继续持久化
   告警并产生生命周期事件。
7. 普通过滤规则 API 和普通仓储 Create/Update/Delete/Toggle 均不能创建或修改 barrier，避免
   与迁移并发时误恢复旧邮件链路。

## 过度设计检查

本次有意没有实现以下内容：

- 没有新增迁移状态表、工作流引擎或自动审批系统；
- 没有为尚未确认的老 OMC 设备组和 Tolerance Duration 猜测映射；
- 没有新增 migration HTTP API 或前端，留给 Task 13 按权限和审计契约统一暴露；
- 没有顺带扩展短信、系统事件、测试发送、隐式 TLS 465 或告警规则通用重构；
- 没有自动删除旧收件人，保留受控回退所需的原规则数据；
- 没有把 `notify_webhook` 纳入第一阶段。

## 验证证据

- `go test ./internal/alarm ./internal/notification -run 'Migration|Barrier|FirstMatch|DomainSchemaIdentity' -count=1`
- `go test ./internal/alarm ./internal/notification -count=1`
- `go vet ./internal/alarm ./internal/notification`
- 使用项目正式 goose 工具在全新 `omcgo_notification_task12_fresh` 数据库加载
  `000001_init_schema.sql`，版本 1 成功。
- `ALARM_FILTER_MIGRATION_TEST_DSN=... go test ./internal/alarm -run '^TestPgAlarmFilterMigrationBarrier_Integration$' -count=1`
  验证未 enabled 时拒绝、enabled 后切换成功，以及普通写操作不能触碰 barrier。

## 后续门禁

进入 Task 13 或真实试点前，仍需准备测试 SMTP 和可控告警样本，完成 raised、minimum duration、
repeat、ack cancel、clear recovery、熔断、逐收件人历史以及 legacy/new 命中和收件人对比。未完成前
不得在生产范围执行 Cutover。
