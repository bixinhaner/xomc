# PM 聚合快照空扫根治与生产验收

## 版本与变更

- 目标机：`172.24.224.197`
- 最终版本：`100.0.0-20260731-2246`
- 主干提交：`448a5791d304`
- 业务根治 MR：`!479`（聚合任务快照修订水位与复用）
- 部署根治 MR：`!480`（`--skip-infra` 镜像前置门禁）
- 部署全过程保留 PostgreSQL、TimescaleDB、Redis、NATS、MinIO 数据。

## 回归门禁

- `go build ./...`：通过
- `go test ./... -count=1`：通过
- `go test -race ./internal/pm/stream -count=1`：通过
- `go vet ./...`：通过
- release 回归：`PASS=248 FAIL=0`
- 新增镜像前置门禁测试：`PASS=16 FAIL=0`
- `bash -n` 与 `git diff --check`：通过

## 生产验收（2026-07-31 22:51–22:55 CST）

### 服务、ACS 与队列

- `healthcheck.sh`：90 项通过，0 项失败。
- app、worker、双 ACS、web、PostgreSQL、TSDB、Redis、NATS、MinIO 与监控栈均运行正常。
- ACS 全局活跃会话 177；准入拒绝、限流拒绝、PM 背压拒绝均为 0；部署后日志无 503。
- JetStream `pm-workers` 与 `pm-registration-wait`：pending、ack pending、redelivered 均为 0。
- 设备任务持续完成；最终部署后新增 expired 为 0。4 条 dead letter 均为 13:51 CST 历史记录，无新增。

### 快照空扫根治

- app 与 worker 启动时各执行一次 336,186 行成员快照加载，分别约 1.12 秒与 1.08 秒。
- 随后运行超过 3 个 worker 分钟刷新周期，并通过 Dashboard 连续查询 6 组日结果和 6 组周结果。
- 最终 app/worker 该慢查询计数仍各为 1，证明定时刷新与 Dashboard 查询均只读取轻量修订水位，未重复全量扫描。

### PM、KPI 与周期结果

- 最近 15 分钟 PM 文件 20,000 个，未解析 0；最近 30 分钟 measurement anchors 280,000 条，最新窗口到 22:45 CST。
- `K900010006`、`K900010076` 均有最新 21:00 小时窗口结果，结果生成于 22:23 CST。
- 14:00 小时窗口共 20,003 个实体，全部 `published/complete`；140,000/140,000 槽位完整。
- 14:00 窗口首次发布于 15:12:09 CST，符合 12 分钟关闭要求；两个关键 KPI 各 20,003 行、320,000 样本，`complete=true`、`period_complete=true`。
- Dashboard 六个面板均显示日进行中 `3/24（12.5%）`，周进行中 `3/168（1.8%）`。

### 数据库与资源

- PostgreSQL、TimescaleDB：超过 5 秒活动查询均为 0，未授予锁均为 0。
- 最终采样：app 442 MiB/1.5 GiB，worker 257 MiB/2 GiB，主库 2.39 GiB/7 GiB，TSDB 3.87 GiB/7 GiB；无 OOM/重启异常。
- 5 秒磁盘增量：读 0.47 MiB、写 4.30 MiB、设备利用率 6.8%；`vmstat` iowait 为 0。容器 Block I/O 大数是累计量，不代表当前磁盘持续高负载。
- Prometheus 唯一 firing 告警为预期常驻的 `DeadMansSwitch`。

## 结论

业务、性能、队列、KPI、数据库、小时/日/周数据均达到验收标准。原来的分钟级 336,186 行空扫已消除；部署缺失镜像也会在复制 release、切换 `current` 和改写数据之前被阻断。
