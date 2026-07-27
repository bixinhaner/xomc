# OMC 有状态服务可配置存储设计

## 背景

目标机当前 PostgreSQL、TimescaleDB、Redis、NATS、MinIO 的 Docker 命名卷都落在
Docker data-root 所在的同一块旋转盘。实测该盘长期接近饱和，读取等待约 154 ms，
压测末段 iowait 约 61%，PM 消费速度显著低于输入速度。

后续运维会挂载 SSD/NVMe 并人工决定数据目录。交付包需要支持在 `deploy/.env`
中配置每个有状态服务的宿主机路径，新安装时由 `plan-resources.sh` 推荐最大可用
文件系统，升级时必须保留人工配置。

## 方案选择

1. 单一 `OMC_DATA_ROOT`：实现最简单，但五个服务仍共享同一设备，不能按压力拆盘。
2. 五个独立路径：兼容单盘默认值，也允许按物理设备拆分，采用此方案。
3. 安装脚本自动迁移旧卷：停机、属主、WAL/AOF/JetStream 一致性和回滚风险高，
   不适合作为隐式升级动作，不采用。

## 配置接口

`deploy/.env` 新增以下非密钥键：

- `POSTGRES_DATA_PATH`
- `TSDB_DATA_PATH`
- `REDIS_DATA_PATH`
- `NATS_DATA_PATH`
- `MINIO_DATA_PATH`

release compose 在变量非空时使用 bind mount；空值继续回退到原 Docker 命名卷，
避免普通升级隐式旁路存量数据。非空路径必须是绝对路径，安装和服务启动前创建目录并
校验可写性。允许逐组件配置和迁移；升级合并白名单包含这五个键，人工配置不会被新包覆盖。

开发 compose 同样读取五个变量，但保留命名卷作为未配置时的默认值，避免改变开发者
现有数据位置。release 的新包模板则由资源规划脚本填入实际绝对路径。

## 自动推荐

`deployments/release/bundle/deploy/plan-resources.sh` 在 Linux 上枚举本地、可写、非伪
文件系统挂载点，排除 `tmpfs`、`overlay`、`squashfs`、`proc`、`sysfs`、`cgroup`、
`devtmpfs` 等不适合作为持久数据盘的类型，以可用字节数最大者作为推荐盘。

默认目录为：

```text
<最大可用挂载点>/omc-data/postgres
<最大可用挂载点>/omc-data/timescaledb
<最大可用挂载点>/omc-data/redis
<最大可用挂载点>/omc-data/nats
<最大可用挂载点>/omc-data/minio
```

脚本只为 `.env` 中缺失或空值的键写入推荐值，不覆盖已有人工设置。`--dry-run` 只展示，
不修改文件。输出明确提示部署人员安装前检查并按物理盘拆分路径，同时声明脚本不会迁移
已有数据。测试通过探测覆盖变量注入虚拟挂载表，保证选择逻辑可重复验证。

## 迁移与拆盘建议

路径配置是启动接口，不等同于数据迁移。已有环境必须停栈后逐项复制数据、核对文件数量
和容量、修改 `.env`，再启动并检查健康状态；旧目录保留到验收完成，便于回滚。

推荐按独立物理设备隔离：

- NVMe A：TimescaleDB（PM COPY、KPI 聚合的主要随机写负载）。
- NVMe B：PostgreSQL 主库（事务和元数据延迟敏感）。
- SSD/NVMe C：MinIO（PM 对象顺序写和读取）。
- Redis 与 NATS 放在剩余低延迟设备；若设备不足，优先与主库/TimescaleDB 分开。

只有一块 SSD/NVMe 时，五个服务可先全部迁到该盘，仍能消除旋转盘寻道瓶颈，但不能获得
设备级隔离。自动规划只选择容量最大的挂载点作为安全的新装默认值，最终拆盘由运维修改
五个独立变量完成。

## 预期收益与验收

收益取决于实际 SSD/NVMe 和阵列配置，不承诺固定倍数。相对当前约 154 ms 的旋转盘读取
等待，SSD/NVMe 通常可把存储等待降到低毫秒级；独立设备还能降低队列深度和 iowait，
减少 PostgreSQL checkpoint、TimescaleDB 写入、MinIO 上传、Redis AOF 和 NATS
JetStream 互相争抢。目标是 PM 消费速率更接近或超过输入速率，积压不再线性增长。

本次目标机没有额外 SSD/NVMe，因此复测分两层：

1. 自动选择、路径校验、升级继承、compose 渲染和脚本测试。
2. 使用当前旋转盘上的兼容路径重新部署并重复 KPI 压测，验证功能无回退；结果明确标注
   为“硬件未迁移基线”，不把脚本变更误报为存储性能收益。
