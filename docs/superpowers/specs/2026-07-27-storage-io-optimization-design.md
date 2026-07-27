# 存储 I/O 降噪与安全压缩设计

## 状态

- 日期：2026-07-27
- 状态：已确认
- 范围：磁盘 I/O 可观测性、MinIO scanner、PM/MR gzip 状态、Redis AOF
- 非范围：MinIO 小对象打包归档、历史数据迁移、缩短 PM 业务保留期、拆分 Redis 实例

## 背景与证据

10000 基站压测中，Linux 5.15、`CONFIG_HZ=250` 的 QEMU 虚拟盘长期显示约 100%
I/O 活动时间。30 秒同窗采样显示：

- `sda` 活动时间 99.2%，平均队列仅 0.069，平均延迟约 0.1ms；
- MinIO scanner 约 265 objects/s，MinIO 读取 4.17MiB/s，等于整盘读取量；
- 容器写入主要来自 TimescaleDB、Redis AOF、MinIO 与 NATS；
- NATS consumer pending、ack pending、redelivery 均为 0。

因此当前现象是高频小 I/O 使 Linux `io_ticks` 饱和，不是已经形成磁盘排队。现有单一
busy 面板会产生误报，并掩盖未来真实的延迟、队列和 PSI 饱和。

另外，ACS 已在 PM/MR 上传入口把明文 XML gzip 后写入 MinIO，但 worker 仍为所有新文件
调度 `rawArchiver`。归档器并发满时直接 `dropped_busy`，使已经 gzip 的对象仍在数据库
中显示 `raw_compressed=false`，同时产生冗余 HEAD 请求和误导指标。

## 设计

### 1. I/O 可观测性与告警

在现有主机资源 Dashboard 增加四组视图：

1. I/O 活动时钟占比，明确这不是容量或延迟饱和度；
2. 平均队列深度；
3. 读写综合平均延迟；
4. CPU iowait、读写吞吐和 MinIO scanner 速率。

新增组合告警。只有磁盘活动时间高、平均队列高、平均延迟高三个条件同时持续满足时，
才报告磁盘 I/O 饱和。保留磁盘空间和容量趋势告警。

Prometheus 继续抓取 MinIO v2 cluster 端点；Dashboard 直接使用已经暴露的
`minio_node_scanner_objects_scanned`、ILM pending/missed 指标。

### 2. MinIO scanner 降速

开发 Compose 和 release Compose 都显式设置：

```text
MINIO_SCANNER_SPEED=slow
```

不使用 `slowest`，避免生命周期、修复和容量统计延迟过大。scanner 不关闭，ILM pending
和 missed tasks 必须继续可观测。该项用于降低持续读 IOPS，不替代后续小对象归档设计。

### 3. gzip 状态成为入库事实

PM collector 使用 `compress.MaybeGunzip` 返回的实际探测结果：

- `FileMarker` 增加 `RawCompressed bool`；
- `CopyIngest` 在插入 `pm_files` 时原子写入 `raw_compressed`；
- 已 gzip 的对象不再调度 `rawArchiver`；
- 明文历史/旁路对象仍走现有 best-effort 归档器。

MR 保持“先保存文件元数据、解析失败也留痕”的现有语义。实际探测为 gzip 后同步调用
`MarkCompressed(old=path,new=path)` 更新标记，并跳过归档器；明文对象继续调度归档器。

不能仅根据 `.gz` 后缀判断，权威值必须来自 gzip 魔数探测。

### 4. Redis 与 TimescaleDB 写放大

Redis 保持 AOF everysec 和 noeviction，不降低聚合状态耐久性；release 继续保留现有
no-appendfsync-on-rewrite。仅提高自动 rewrite 阈值：

```text
auto-aof-rewrite-min-size=1gb
auto-aof-rewrite-percentage=500
```

PM 上传 Redis 去重窗口加入 `upload.pm_dedup_ttl`，默认 4 小时。Redis 负责短期入口去重，
TimescaleDB 的 `(device_sn,file_name)` 与 `(device_sn,content_sha256)` 唯一约束继续负责
最终幂等。

最新主线已经切换到流式聚合。旧的 `RunSparseMaintenance` 水位维护代码仍在仓库中，
但当前 worker 启动路径不再运行它。因此两张原始 PM 表继续保留固定 7 天
`add_compression_policy`；若本次删除，chunk 会永久不压缩并造成长期容量风险。
压缩权威重构留待后续结合流式窗口状态单独设计。

## 失败与回退

- scanner 降速后如果 ILM pending/missed 增长，回退到 `default`；
- MR 压缩状态后续标记失败不重试已经解析完成的文件；明文路径仍保留归档器；
- Redis rewrite 阈值只改变压缩频率，不改变 AOF everysec 写入与恢复语义；
- TimescaleDB 现有 7 天自动压缩策略保持不变。

## 验收

- Dashboard JSON、Prometheus 规则和 Compose 配置通过静态验证；
- gzip PM marker 原子写入 `raw_compressed=true`，且不再调度归档器；
- 明文 PM/MR 仍调度归档器；
- release 与开发 Compose 的 MinIO/Redis 参数一致；
- TSDB 基线的原始表 7 天自动压缩策略保持不变；
- `go build ./...`、`go test ./...`、前端 typecheck、release storage tests 全部通过；
- 部署后业务健康、NATS 无积压，并复测 scanner IOPS、磁盘 queue/await/iowait。
