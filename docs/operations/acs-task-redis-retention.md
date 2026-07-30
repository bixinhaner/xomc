# ACS task Redis 保留策略

`acs:task:{taskID}` 是设备任务的运行时副本，PostgreSQL `device_tasks` 是持久化权威源。

- `pending`、`sent` 和待重试任务保留 4 小时。
- `completed`、最终 `failed`、`cancelled`、`expired` 默认保留 15 分钟。
- `task.terminal_redis_ttl` 可调整终态窗口；未配置取 15 分钟，小于 10 分钟会被钳到 10 分钟。
- 终态转换通过单键 Lua 脚本同时写入任务内容和短 TTL，第一个终态获胜；迟到重复响应不会覆盖状态或延长 TTL。
- 终态会幂等删除设备队列成员和 `acs:cwmp2task:*` 反查键。

15 分钟覆盖 ACS 默认 5 分钟会话窗口，并显著大于 worker 的 60 秒 Redis→PostgreSQL
对账宽限。若设备响应特征发生变化，先核对会话超时与对账宽限，再提高本配置。

以线上观测的每分钟 7,410 个任务、平均每个终态 Hash 2,770 字节估算，15 分钟稳态约
111,150 个 Hash、约 0.29 GiB；原 4 小时窗口理论上约 177.8 万个 Hash、约 4.59 GiB。

升级时不扫描或批量改写已有约百万个键，以免阻塞 Redis。旧版本产生的终态键最多按原
4 小时 TTL 自然回落；升级后新完成的任务立即使用短 TTL。若测试环境按部署流程清空
Redis，则不存在旧键过渡期。
