# 邮件短信通知中心里程碑 A 本地验收

## 结论

里程碑 A 的可靠告警生命周期主链路、Shadow/canonical 隔离、NATS 故障恢复、容量分档和 `shadow → legacy → shadow` 回退均已通过本地验证。该结论不代表生产 canonical 或真实外发放行。

## 验收范围

- 告警事务、Outbox、标准生命周期事件和独立 durable 投影。
- NATS 停机期间主事务继续提交，恢复后 Outbox 重试且事件 ID 不重复。
- 20/s、100/s、200/s 本地分档负载，100 台隔离虚拟 gNB，生命周期事件最终一致。
- Shadow、legacy、canonical 的受控切换和回退。

## 结果

- 9600 条生命周期事件全部 published，event ID 唯一。
- occurrence 版本范围严格为 1–2，容量测试活动告警归零。
- history/northbound durable 最终零积压、零重投。
- 最高本地档实际约 188/s，Outbox P95/P99 约 99/113ms；结果仅用于发现本机瓶颈。
- `MaxAge=7d` 和 `MaxBytes=10GiB` 不能直接宣称 7 天容量承诺：按本地平均事件大小估算，10GiB 在高事件率下不足 7 天。
- 回退后恢复 Shadow，主库健康，未修改正常开发栈和旧 OMC。

## 生产门禁

仍需现场平均/峰值、最长消费者中断、磁盘预算、至少两倍峰值压测、三 durable 对比和生产级回退演练。门禁通过前保持 `legacy/shadow`，禁止 canonical、真实邮件和 SMS Worker。

详细的原始容量数字、端口、注入方式和故障日志已合并保留在本文件对应验收记录中；不再拆分为独立的 milestone-a-local-validation 与 capacity-rollback 文件。
