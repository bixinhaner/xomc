# kpiperf — KPI(PM) 文件上传压测工具

压测 OMC **KPI/PM 模块两段能力**：

1. **文件存储**：设备直传 PM 文件 → ACS 流式落 MinIO（`pm-files` 桶）。
2. **KPI 解析入库**：worker 订阅 `pm.file.received` → 下载 → 解析 3GPP 32.435 XML → 批量写 `pm_metrics`（counter）→ 反算 KPI（`metric_type='kpi'` 行）。

工具用本目录两个真机样本（4G eNodeB + 5G gNB）作模板，按 `(SN, KPI 时间窗)` 批量生成并发上传，支持 **200–10000 并发**可指定。

## 链路（直传路径，不走 AutonomousTransferComplete）

```
kpiperf ──PUT /smallcell/FileUploadService?fileType=4&sn=<SN>&filename=<name>──▶ ACS:7557
                                                          │ 流式落 MinIO(pm-files)
                                                          │ 发 pm.file.received（瘦 payload，仅 device_sn）
                                                          ▼
worker: resolveDevice(按 SN 查 devices 表) → 解析 XML → BatchInsert pm_metrics → KPI 反算
```

> **关键**：worker 的 `resolveDevice` 对**未注入的 SN 会报错并重试进 DLQ**——文件存储那一段对任意 SN 都成立，但**解析入库必须先注入测试设备**（`internal/pm/collector/collector.go` 的 `GetBySerialNumber` 命中 `devices` 表才放行）。设备身份以 URL `?sn=` 为准（`applyPayloadIdentity` 覆盖 XML 内解析出的 SN），所以 body 内 `localDn` 改不改都不影响路由。

## 前置条件

- OMC 容器栈已起（`docker compose ... up -d`）：ACS `:7557`、PostgreSQL `:5432`、worker metrics `:9092`、MinIO/NATS 正常。
- 文件描述符上限够高（高并发必须）：`ulimit -n 65535`。
- 内存：峰值约 `并发数 × 单文件大小`（样本 4G≈120KB / 5G≈260KB）。1 万并发跑 5G 模板峰值 ~2.6GB，按需下调并发或只用 4G 模板。

## 构建

```bash
cd omcgo
go build -o bin/kpiperf ./test/pmperf      # 或直接 go run ./test/pmperf <flags>
ulimit -n 65535
```

## 快速上手

所有命令在 `omcgo/` 目录下执行（模板默认路径相对该目录）。

```bash
# ① 仅注入 1 万个测试设备（cmcc/lte，SN=KPILT-0000001…）
bin/kpiperf -mode seed -devices 10000

# ② 标准跑：注入 1 万设备 + 1 万文件（每设备一份）+ 并发 500 + 入库验证
bin/kpiperf -concurrency 500 -devices 10000

# ③ 拉满并发到 1 万
ulimit -n 65535
bin/kpiperf -concurrency 10000 -devices 10000

# ④ 时长模式：持续 2 分钟，1000 并发，5000 设备 × 20 个时间窗（扩大唯一行空间，多量入库）
bin/kpiperf -duration 2m -concurrency 1000 -devices 5000 -buckets 20

# ⑤ 只压文件存储（不连库、不验证入库）
bin/kpiperf -no-db -concurrency 1000 -devices 2000

# ⑥ 只用 4G 模板（省内存），JSON 输出
bin/kpiperf -concurrency 2000 -devices 5000 -json \
  -templates 'test/pmperf/A20260611.0815+0800-0830+0800_48BF74.120299024119AA05000.xml'

# ⑦ 跑完顺手清理
bin/kpiperf -concurrency 500 -devices 10000 -cleanup-after
```

## 一键清理

测试设备与 KPI 数据全部带 SN 前缀（默认 `KPILT-`），一条命令清干净：

```bash
bash test/pmperf/cleanup.sh                 # 默认前缀 KPILT
SN_PREFIX=FOO bash test/pmperf/cleanup.sh   # 自定义前缀
# 等价于：
bin/kpiperf -mode cleanup -sn-prefix KPILT
```

清理会按 `device_sn / serial_number LIKE 'KPILT-%'` 删除 `pm_metrics`、`pm_files`、`devices` 三张表。
手工兜底 SQL（容器内 `docker compose exec -T postgres psql -U omcgo -d omcgo`）：

```sql
DELETE FROM pm_metrics WHERE device_sn LIKE 'KPILT-%';
DELETE FROM pm_files   WHERE device_sn LIKE 'KPILT-%';
DELETE FROM devices    WHERE serial_number LIKE 'KPILT-%';
```

## 报告解读

- **阶段一 文件存储**：上传成功率、墙钟、吞吐（文件/s、MB/s）、时延 p50/p90/p95/p99/max、HTTP 状态码分布。
- **阶段二 解析入库**：等待入库收敛后给出新解析文件数（`pm_files.parsed` 增量）、新增 `pm_metrics` 行数、入库吞吐（文件/s、行/s），以及 worker Prometheus 增量（成功/失败/迟到文件、丢弃 counter、单文件处理均值、上报延迟均值）。

入库文件数 < 上传成功数时常见原因：设备未注入（SN 未命中）、迟到数据落进 TimescaleDB 压缩 chunk 被跳过（late_arrival）、worker 落后或进了 DLQ。

## 参数

| 参数 | 默认 | 说明 |
|------|------|------|
| `-mode` | `run` | `run`(注入+上传+验证) / `seed`(仅注入) / `cleanup`(仅清理) |
| `-url` | `http://localhost:7557` | ACS 上传端点 base（或经 nginx `:8080`） |
| `-templates` | 本目录两个样本 | 逗号分隔模板，多个则按文件序轮转（4G/5G 混压） |
| `-concurrency` | `200` | 最大在飞上传数（200–10000） |
| `-devices` | `10000` | 不同设备 SN 数（run 模式默认注入这么多） |
| `-files` | `0` | 总文件数；0 = `devices×buckets`（每设备每窗一份） |
| `-duration` | `0` | 按时长持续压测（>0 忽略 `-files`） |
| `-buckets` | `1` | 15min KPI 时间窗个数（向最近完成窗回溯铺开） |
| `-sn-prefix` | `KPILT` | 测试设备 SN 前缀（清理按此 LIKE） |
| `-oui` | `48BF74` | 设备 OUI（6 hex） |
| `-carrier` | `cmcc` | `cmcc`/`ctcc`/`cucc`/`other`（devices 分区键） |
| `-tech` | `lte` | `lte`/`nr`/`gsm` |
| `-product-class` | `空` | 设备 product_class（填了才便于 KPI 路由；留空 counter 仍入库） |
| `-filetype` | `4` | 上传 fileType（PM=4） |
| `-username`/`-password` | 空 | 上传端点 Basic Auth（默认无鉴权） |
| `-db` | `…localhost:5432/omcgo` | PostgreSQL DSN（注入/验证/清理） |
| `-no-db` | `false` | 跳过注入与入库验证，仅压文件存储 |
| `-seed` | `true` | run 模式上传前注入设备 |
| `-cleanup-after` | `false` | run 模式结束后一键清理 |
| `-worker-metrics` | `http://localhost:9092/metrics` | worker Prometheus（抓 PM 解析指标增量） |
| `-drain` | `120s` | 上传后等待入库收敛的最长时长 |
| `-timeout` | `30s` | 单次上传 HTTP 超时 |
| `-json` | `false` | JSON 输出 |
| `-rewrite-body-sn` | `true` | 改写 body 内 `localDn` 的 SN（保真，不影响路由） |
| `-insecure` | `false` | https 时跳过 TLS 校验 |

## 文件生成的改写点

每份文件相对模板只改身份与时间，保留运营商真机的全部 measType / 命名空间 / 结构：

- **SN**：URL `?sn=`（权威）+ 文件名 `{OUI}.{SN}` 段 + body `managedElement localDn`（`-rewrite-body-sn`）。
- **KPI 时间**：`fileHeader/measCollec@beginTime`、各 `measInfo/granPeriod@endTime`、`fileFooter/measCollec@endTime` 统一替换为分配到的 15min 时间窗（默认最近一个已完成窗口）。
- **文件名**：`A{date}.{start}+0800-{end}+0800_{OUI}.{SN}__{seq}.xml`，`seq` 保证全局唯一，避免同日 MinIO 路径互相覆盖。

`pm_metrics` 自然键含 `device_sn + end_time + time + object_ldn`，因此不同 SN 必产生不同行；同 `(SN,时间窗)` 重复上传则触发 UPSERT（行数不增，但仍走完整解析入库）。`-buckets` 越大、每设备可铺的唯一时间窗越多。
