# 邮件短信通知中心里程碑 B 本地验收

## 结论

里程碑 B 的通知 Inbox、持久化 schedule、编排、迁移 barrier、邮件 Worker、SMTP 重试和渠道熔断已在隔离 PostgreSQL、回环 SMTP 与 `.invalid` 地址上通过本地验证。没有连接真实邮件服务、真实收件人或旧 OMC 写接口。

## Task 8：Inbox 与持久化调度

- `event_id` 幂等，旧版本只审计，版本缺口等待补洞。
- ack/unack/clear 推进 schedule generation 并取消旧代；unack 只恢复未来 repeat，不重建 initial。
- `FOR UPDATE SKIP LOCKED` + 租约保证多实例安全领取，过期租约可恢复，完成/取消校验 worker 所有权。
- 只领取已注册的 `initial_gate`、`repeat`、`digest_flush`、`quiet_hours_release` handler。

## Task 10：通知编排

- 标准生命周期事件经有序 Inbox 投影后，按不可变规则版本生成 schedule、摘要 bucket、suppressed 审计或 delivery。
- minimum duration、repeat、升级、确认、清除、维护窗口和 recovery 配对均有明确状态语义。
- 外呼前校验 sending lease、occurrence 版本、generation、渠道启用和熔断状态。
- 收件人使用 AES-256-GCM 与 HMAC 指纹保护，历史不返回明文地址。

## Task 11：邮件可靠投递

- SMTP DATA 成功记录 `completed + accepted`，不伪造最终 delivered。
- 4xx 临时失败有界退避；永久失败进入 dead letter；结果未知禁止盲目重发。
- 每个收件人使用独立 SMTP envelope；认证/TLS/连续临时失败触发渠道熔断。
- App、Worker 和 legacy `notify_email` 统一复用 `notification.smtp` transport。
- SMS、Kafka、quiet-hours 的未冻结或未兑现能力没有提前扩展。

## Task 12：迁移兼容

- 迁移只提供旧 `notify_email` 的只读 preview、重叠检查、Shadow 命中门禁和 barrier cutover。
- 不自动生成候选规则，不猜测设备组或 Tolerance Duration，不迁移 webhook，不伪造逐收件人历史。
- barrier 保持旧首条命中语义但不发送邮件；PG 条件更新防止并发漂移。

## 验证

相关 notification、alarm、app provider、worker、event 和 NATS 测试，以及隔离数据库集成测试均通过。全仓仍可能受既有 `paramsync` schema 与 Redis 测试环境影响，不作为本里程碑失败依据。

## 生产门禁

仍需真实 SMTP TLS/AUTH/ACL、可控小基站 raised/clear/recovery 样本、Shadow 命中和收件人对比、现场容量数据及真实浏览器验收。门禁通过前保持 SMTP 和规则 disabled。

本文件合并原 Task 8、Task 10、Task 11、Task 12 和 milestone-b review，避免同一验收证据按任务重复维护。
