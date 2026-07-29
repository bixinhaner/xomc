# ACS 30,000 全局会话上限设计

## 目标

将 Docker 生产部署使用的 ACS `session.max_concurrent` 从 10,000 持久调整为
30,000，缓解整点 Inform 风暴触发的准入拒绝，同时不放宽 PM 背压、磁盘、I/O、限流等
既有保护。

## 范围

- 修改 `omcgo/cmd/acs/etc/config.prod.yaml` 的生产默认值和相邻容量说明。
- 用发布部署契约锁定生产值为 30,000。
- 不修改 dev、test、Kubernetes 和压力测试配置。
- 重新构建发布包并滚动更新测试服务器 ACS。

## 验收

北京时间 14:00–14:05 连续观察：

- ACS 活跃会话峰值、`admission denied`/HTTP 503；
- Nginx QPS、宿主机和容器 CPU/内存、磁盘延迟；
- PM/NATS 队列、聚合发布积压、Redis 内存与 AOF；
- PostgreSQL/TimescaleDB 连接、长查询、事务率和 firing 告警。

通过条件为配置实际加载 30,000、准入拒绝为 0、队列和发布积压可恢复、无资源饱和或新增
业务错误。若仍出现 503，必须按来源区分准入上限、设备限流、PM 背压和其他 HTTP 错误。

