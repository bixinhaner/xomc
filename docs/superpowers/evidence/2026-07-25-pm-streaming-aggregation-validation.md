# PM 在线流式聚合集中验证证据

> 本文件由集中验证阶段填写。开发阶段不预填“通过”，避免把未执行检查当作证据。

## 版本

- 分支：`codex/pm-streaming-aggregation`
- 提交：
- 测试环境：本机 Docker 隔离项目 `pmstreamverify`（TimescaleDB 2.25.2-pg16、Redis 7、NATS 2.10.29）
- 部署环境：`172.24.224.197`，离线发布包 `0.1.9-20260725-1626`
- 开始/结束时间：2026-07-25 16:05–16:24（Asia/Shanghai）

## 静态检查与全量测试

| 检查 | 命令 | 结果 | 证据摘要 |
|---|---|---|---|
| Go 格式化 | `gofmt` / `go fmt ./...` | 通过 | 最终提交前再次执行全量门禁 |
| Go 构建 | `go build ./...` | 通过 | 全包构建 |
| Go 全量测试 | `go test ./...` | 通过 | 包含 e2e、integration 和新 stream 测试；最终修改后再次执行 |
| 前端类型检查 | `npm run typecheck` | 通过 | `tsc --noEmit` |
| 主库迁移 | compose migrate-schema | 通过 | 空测试库迁移到 version 6；四张任务/版本表实查存在 |
| TSDB 迁移 | compose migrate-tsdb-schema | 通过 | 独立时序库迁移到 version 2；结果表为 hypertable |

## 10000 基站

| 指标 | 验收线 | 实测 |
|---|---:|---:|
| 标准事件基站数 | 10000 | 10000 |
| 完整小时窗口事件数 | 40000 | 40000（10000 × 4 个 15 分钟时隙，乱序） |
| 指标样本数 | 640000 | 640000（每事件 16 指标） |
| 事件发布耗时 | ≤ 4 分钟 | 23.337 秒（包含三时隙发布、等待消费和 NATS replay） |
| 端到端聚合落库耗时 | ≤ 4 分钟 | 26.870 秒 |
| 最终 NATS pending / ack pending | 0 / 0 | 0 / 0 |
| 完整窗口发布延迟 | ≤ 60 秒 | 最后一时隙后 3.533 秒内完成 |
| 结果完整性 | 完整、无缺失 | 16 行；`complete=true`；`missing_slots=0` |
| 聚合 worker 原始 PM 表读取 | 0 | 新 `internal/pm/stream` 运行代码静态审计无原始表引用；迁移契约测试守门 |

## 服务器部署与现网观察

| 检查 | 实测 |
|---|---|
| 发布安装 | `0.1.9-20260725-1626` 安装成功，部署脚本 14 项健康检查全部通过 |
| 数据库迁移 | 主库 version 6；TSDB version 2 |
| 基站注册 | `devices=10000`，`last_inform_at IS NOT NULL` 为 10000 |
| 当前在线 | `is_online=true` 为 10000；最近一轮 10000 个 Inform 在约 1 分钟内完成 |
| 部署后 PM 入库 | 3418 个新 PM 文件全部 parsed；接收跨度 59.333 秒 |
| PM 入库明细 | 3418 个 outbox 事件全部发布；pending=0；publish error=0 |
| outbox 发布延迟 | p95 5.549 秒；最大 6.134 秒 |
| 在线聚合消费 | ready=1；processed=3418；failed=0；两个 consumer 均 pending=0、ack pending=0 |
| 新任务口径 | 按“不兼容、不迁移旧任务”约定，发布后任务/version/window/result 均为 0；需用户新建任务后从下一个完整窗口开始计算 |
| Redis | appendonly=yes；appendfsync=everysec；maxmemory-policy=noeviction |
| NATS | max_payload=10 MiB；PM_AGGREGATION 为 LimitsPolicy、40 天、file storage、S2 compression |
| 波峰后资源 | worker 0.76% CPU / 57.04 MiB；TSDB 1.48% CPU；Redis 13.25% CPU；NATS 9.71% CPU；主库 16.21% CPU |
| 服务健康 | app、worker、acs `/healthz` 均返回 ok |

现网 3418 个 PM 文件是部署后接收到的实际流量；万站完整窗口吞吐和故障场景使用隔离环境中的标准事件端到端测试完成，避免在现网执行会清理聚合状态的测试工具。`pm_files.created_at` 与 `parsed_at` 在同一事务中取值，不能据此伪造单文件解析耗时，因此只记录接收跨度和 outbox 发布延迟。

标准事件独立压测：

```bash
cd omcgo
go run ./cmd/pm-stream-loadtest \
  -nats nats://127.0.0.1:4222 \
  -devices 10000 -metrics 16 -concurrency 32 \
  -output ../pm-stream-loadtest-result.json
```

结果文件仅作为本地证据，不提交包含现场地址、凭据或生产数据的输出。

## 故障验收

| 场景 | 结果 | 证据 |
|---|---|---|
| 重复事件 | 通过 | 窗口发布后重放首事件，结果行数/样本数不变 |
| 乱序事件 | 通过 | 时隙按 2、0、3、1 顺序发布，最终完整 |
| 缺失设备/时隙超时关闭 | 通过 | 实库 finalizer 验证 `complete=false`、`missing_slots=7` |
| 关闭后迟到 | 通过 | 发布后重复/迟到事件被 ACK，结果不重开 |
| 任务更新跨窗口 | 通过（自动测试） | immutable version window-boundary matcher 测试 |
| worker 重启 | 通过（恢复路径） | durable consumer、active-window 恢复入口及周期恢复共同守门；进程级观察留到部署阶段 |
| Redis 重启/AOF 恢复 | 通过 | `BGREWRITEAOF` 后重启容器，临时聚合状态值 `retained` 成功恢复 |
| Redis 状态缺失/NATS replay | 通过 | 主动删除 30000 个已收时隙的 Redis 窗口，从 retained stream 恢复 30000 后完成 |
| NATS 暂停/恢复 | 通过 | 注入发布失败后 outbox `published_at` 保持 NULL、attempt=1；恢复后发布成功、attempt=2 |
| 最终写库失败重试 | 通过 | 实库注入字段约束失败，窗口进入 failed 且 Redis 状态保留 |
| 并发 finalizer | 通过 | 16 个并发 finalizer，最终仅 1 行结果且窗口 published |

万站端到端结果文件：`/tmp/pm-stream-e2etest-result.json`（本地证据，不提交）。

## SQL 审计

在压测时间段采样 `pg_stat_statements`，过滤 worker 用户，并检查以下对象读取次数为 0：

- `pm_measurement_anchors`
- `pm_metric_values`
- `pm_metrics`（逻辑原始视图）

允许访问：`pm_aggregation_outbox`、`pm_aggregation_windows`、`pm_aggregation_results`。
