# cAdvisor 陈旧容器序列告警根治计划

## 线上证据

- 业务升级后，安装器对 resources.env、Compose 和 Docker inspect 的 77 项检查全部通过。
- Prometheus 仍触发 `OMCResourcePlanDrift{compose_service="web"}`，计算值恰好为声明值的 2 倍。
- cAdvisor 当前只暴露一个新 web 容器；Prometheus 在约 5 分钟 staleness 窗口内仍保留旧容器序列，聚合时把新旧 limit 相加。
- `ContainerMemoryNearLimit` 同时引用了已经删除的旧 tempo 容器临终样本（98.5%），当前 tempo 实际仅约 21.7%。
- `omc_persistent_queue_pending` 由 app 和 worker 同时采集同一 PostgreSQL 快照，原告警和 Dashboard 用 `sum` 将同一积压重复计算约两倍。

## 修复

1. 建立基于 `container_last_seen` 的新鲜容器记录规则。
2. 内存饱和、CPU 饱和、TSDB 资源和资源计划比对只使用新鲜容器序列。
3. 资源 limit 缺失检测同样只认可新鲜序列，防止旧容器暂时掩盖当前容器缺失。
4. 保留 OOM 历史窗口语义；OOM 事件告警不做新鲜过滤。
5. 持久化队列按 `queue/status` 先取多进程快照最大值去重，再跨状态求和。
6. 发布门禁锁定记录规则、新鲜度过滤和队列去重接入点。

## 验收

- Prometheus 规则语法检查通过。
- 发布门禁、后端、前端验证通过。
- 部署切换后不再因旧容器序列触发资源漂移或内存假告警。
- 实际资源异常和当前容器序列缺失仍可正常检测。
