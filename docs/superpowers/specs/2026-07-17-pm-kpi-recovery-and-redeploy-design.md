# PM/KPI 压测恢复与重新部署设计

## 目标

修复 PM 文件在成功入库并压缩改名后，重复或重投事件仍读取旧 `.xml` 路径而进入
DLQ 的问题；清除已经失效的 PM 消息积压；在不删除既有 PM/KPI 入库数据和业务设备
数据的前提下重新发布 OMC，并重新测量端到端 KPI 压测性能。

## 范围

本次包含：

1. worker PM collector 的幂等短路修复和自动化回归测试。
2. 生产配置中的 `OMC_PUBLIC_HOST` 修正。
3. AIDE 对 Docker/OMC 动态数据目录的排除，以及 OMC 大日志轮转。
4. 按目标服务器 32 逻辑 CPU、32 GiB 内存重新生成并应用容器资源限额。
5. 清空 NATS `PM` stream 和 `source_module='pm'` 的 DLQ 记录。
6. 构建本地项目发布包、复制到服务器、使用既有 `install.sh` 升级并重启 OMC。
7. 对升级后的现有 CPE 压测重新采集 HTTP、ACS、worker、NATS、PostgreSQL、
   TimescaleDB、MinIO 和宿主机指标。

本次不包含：

- 不删除 `pm_files`、`pm_metrics` 或业务设备数据。
- 不清理 MinIO 中已经成功入库的 PM 原始文件。
- 不变更 KPI 白名单；当前 `whitelist_miss` 已由用户确认符合预期。
- 不进行物理磁盘迁移；服务器当前没有额外 SSD/NVMe 目标盘。
- 不使用裸进程重启脚本，部署和重启均走交付包内 Docker Compose 流程。

## 根因与修复设计

当前链路为：

```text
pm.file.received
  -> resolveDevice
  -> MinIO GetObject(事件里的旧 .xml)
  -> 解析、计算 KPI
  -> CopyIngest 插入 pm_files marker 和指标
  -> 异步压缩为 .xml.gz 并删除旧 .xml
```

当同一 `(device_sn, file_name)` 的重复事件稍后到达时，数据库已经存在成功 marker，
但 collector 仍先读取旧 `.xml`。旧对象已被归档器删除，因此事件被误判为失败并重试。

修复后：

```text
pm.file.received
  -> resolveDevice
  -> 查询 pm_files(device_sn, file_name)
  -> 已存在且 parsed=true：记录 duplicate 指标，直接返回 nil 让 NATS ACK
  -> 不存在：继续下载、解析和 CopyIngest
```

幂等查询必须使用现有唯一键 `(device_sn, file_name)`，只返回是否存在/必要的文件状态，
不得扫描指标表。查询失败视为真实基础设施错误并进入现有重试流程，不能静默绕过。

为避免扩大 `PMFileStore` 面向 UI 的接口，本次给 collector 增加一个最小的
`FileMarkerLookup` 接口，由 `PgPMFileStore` 实现 `ExistsByDeviceSNAndFileName`。
worker 启动时显式注入。测试使用轻量 fake，验证存在 marker 时 MinIO、解析器和
CopyIngest 均不会被调用。

## 运行配置设计

### OMC_PUBLIC_HOST

目标值为 `172.24.224.78`。升级前写入当前部署的 `.env`，并依赖 `install.sh`
继承非版本配置到新 release。升级后检查 worker 容器环境变量和渲染后的 PM 上传 URL。

### AIDE

保留 AIDE，但增加本机专用排除规则，至少排除：

- `/home/docker-data`
- `/opt/omc/run`
- `/opt/omc/current`
- `/opt/omc/releases`
- `/opt/omc/packages`

规则采用 AIDE 的递归否定形式，避免每天对 PostgreSQL、TimescaleDB、MinIO、
NATS、overlayfs 和运行日志做全量校验。若当前 AIDE 任务仍在运行，先记录状态，再停止
该次扫描；不禁用 AIDE 的日常任务。

### 日志轮转

为 `/opt/omc/run/logs/**/*.log` 增加宿主机 logrotate 规则：

- 每日轮转，或单文件达到 100 MiB 时轮转。
- 保留 7 份。
- 压缩历史文件，延迟一轮压缩。
- 使用 `copytruncate`，避免要求业务进程重新打开文件。

部署后用 `logrotate -d` 验证规则，不强制轮转当前文件，避免在压测高峰制造额外 I/O。

### 资源规划

使用交付包自带 `plan-resources.sh --assume-dedicated` 按目标服务器实测资源生成
`resources.env`。保留监控栈。生成结果先检查总内存、两个 PostgreSQL 的
`shared_buffers`、worker CPU/GOMAXPROCS 和 NATS/MinIO 上限，再由 Compose 应用。

由于当前唯一数据盘是 PERC H730P 暴露的旋转逻辑卷，本次不承诺通过资源限额消除磁盘
瓶颈；资源调整目标是避免不合理 CPU 限流和内存配置，权威基线仍需标注存储条件。

## 清理设计

在停止 app/acs/worker 写入之后执行：

1. 记录 PM stream 和 PM DLQ 清理前计数。
2. 使用项目现有 NATS 管理能力 purge `PM` stream。
3. 删除主库中 `source_module='pm'` 的 `dead_letters`。
4. 不删除其他模块 DLQ。
5. 不删除 `pm_files`、`pm_metrics`、设备和 MinIO 对象。
6. 服务启动后确认新的 `pm-workers` consumer 从干净位置消费，pending 为 0 或能持续收敛。

如果清理命令或数据库删除影响范围与上述条件不一致，立即停止，不执行扩大范围的替代命令。

## 构建与部署

1. 本地完成单元测试、`go build ./...`、相关 Go 测试和发布脚本静态检查。
2. 使用 `deployments/release/build-release.sh --channel release` 构建唯一版本项目包。
3. 通过 `scp` 复制包到目标服务器 `/opt/omc/packages/`。
4. 校验 SHA-256，解压到独立临时目录。
5. 在目标服务器按上述顺序暂停写入、清理队列/DLQ、修正本机配置。
6. 运行新包 `deploy/install.sh`，由其加载业务镜像、执行迁移、切换 `current`、
   Compose 更新和健康检查。
7. 运行 `healthcheck.sh`、Compose `ps`、关键 API/metrics 检查。
8. 失败时保留新 release 和日志，使用上一版 release 配置回退；不删除数据卷。

## 验证与压测判定

代码验证：

- marker 已存在时 handler 返回成功。
- marker 已存在时不访问 MinIO、不解析、不 CopyIngest、不进入 DLQ。
- marker 不存在时原有成功和失败路径保持不变。
- marker 查询错误沿现有重试链路返回。

部署验证：

- 全部容器健康。
- worker 的 `OMC_PUBLIC_HOST` 为 `172.24.224.78`。
- PM stream 清理后没有历史百万级 pending。
- 新 PM 文件成功数持续增长，失败数和 PM DLQ 不持续增长。

重新分析使用至少 5 分钟稳定窗口，输出：

- FileUpload 文件/秒、HTTP 成功率、平均、P50、P95、P99、最大延迟。
- ACS 活跃会话和会话 P50/P95/P99。
- PM worker 成功/失败文件速率和处理 P95。
- NATS incoming、pending、ack pending、redelivery。
- 主库/时序库等待事件、容器 CPU/内存/块 I/O。
- 宿主 load、CPU iowait、磁盘利用率、队列深度和读写延迟。

端到端通过条件：

- PM HTTP 成功率至少 99.9%。
- PM worker 在稳定窗口内无持续失败增长。
- PM pending 在压测结束后可收敛，而不是持续扩大。
- 抽样文件能在 `pm_files` 和 `pm_metrics` 中找到对应成功结果。
- 报告明确区分上传能力、入库能力和单旋转盘造成的硬件上限。
