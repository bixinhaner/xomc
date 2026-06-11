# kpiperf — KPI(PM) 文件上传压测工具（4G/5G/GSM）

压测 OMC **KPI/PM 模块两段能力**：

1. **文件存储**：设备直传 PM 文件 → ACS 流式落 MinIO（`pm-files` 桶）。
2. **KPI 解析入库**：worker 解析 3GPP 32.435 XML → 批量写 `pm_metrics`（counter）→ 按平台反算 KPI（`metric_type='kpi'`）。

支持 **4G/5G/GSM 三制式同时压**、**总并发在三者间拆分**。默认用**指标库驱动合成法**：按各平台
（BLQ/BaiBNQ/BSC）内置指标库 (`data/indicator-library/`) 的全部源 counter（`isCounter=1` 的 `reportKey`）
生成文件 —— **覆盖全部内置指标**，且单文件无重名 counter（避开 `pm_metrics` 自然键冲突 SQLSTATE 21000）。

## 链路（直传，不走 AutonomousTransferComplete）

```
kpiperf ──PUT /smallcell/FileUploadService?fileType=4&sn=&filename=──▶ ACS:7557
   ACS 落 MinIO(pm-files) + 发 pm.file.received（瘦 payload，仅 device_sn）
   worker: resolveDevice(按 SN 查 devices) → 解析 → 写 pm_metrics(counter)
           → 路由 product_class→平台 → 反算 KPI(metric_type=kpi)
```

## ⚠️ 关键前置：设备必须「带 productClass、在上传前」注入

worker 的 KPIRouter 按 `device_sn` 缓存路由结果（**含“未匹配”的负缓存**）。若设备首次被处理时
`product_class` 为空 / 不匹配任何 `products.xml` pattern：

- **算不出 KPI**（路由不到 indicator platform）；
- counter 白名单为空 → fail-open → **放过重名 counter 撞自然键（SQLSTATE 21000）→ 整文件失败进 DLQ**。

本工具 `run`/`seed` 默认在上传前注入设备并带上每制式正确的 `product_class`：

| RAT | tech | product_class | 命中 pattern | 平台 | indicator 库 |
|-----|------|---------------|--------------|------|--------------|
| lte | lte  | `FAP/BAIBLQ/SC` | `^FAP/BAIBLQ/SC$` | BLQ    | `enb/BLQ.xml`（1031 源 counter / 63 派生 KPI）|
| nr  | nr   | `FAP/BSCNR`     | `FAP/\w*BSC\w+`   | BaiBNQ | `GNB.xml`（217 / 67）|
| gsm | gsm  | `FAP/PGSM`      | `^FAP/PGSM$`      | BSC    | `GSM.xml`（45 / 26）|

> 复用已被「无 productClass」处理过的 SN 会命中 worker 负缓存而永远算不出 KPI；本工具每制式用
> 独立 SN 段（`KPILT-LTE-* / KPILT-NR-* / KPILT-GSM-*`）并默认先注入，规避该坑。换批请先 cleanup。

## 本机端口（boss 栈占了标准端口，OMC 做了偏移）

| 服务 | 本机端口 | 传入参数 |
|------|---------|---------|
| ACS 上传 | 7557 | `-url http://localhost:7557` |
| PostgreSQL | **15432** | `-db postgres://omcgo:omcgo123@localhost:15432/omcgo?sslmode=disable` |
| NATS | **14222** | `-nats nats://localhost:14222` |
| worker metrics | 9092 | `-worker-metrics http://localhost:9092/metrics` |
| 前端(性能管理) | **18081** | 浏览器 `http://localhost:18081` |

> 工具内置默认是「标准端口」（5432/4222），本机务必按上表覆盖 `-db`/`-nats`。

## 构建

```bash
cd omcgo
go build -o bin/kpiperf ./test/pmperf
ulimit -n 65535            # 高并发必须
```

## 快速上手（在 omcgo/ 下；下面已带本机端口覆盖）

```bash
DB='postgres://omcgo:omcgo123@localhost:15432/omcgo?sslmode=disable'
NATS='nats://localhost:14222'

# ① 4G/5G/GSM 一起压，100 并发拆 34/33/33，3000 设备（每制式 1000），覆盖全部内置指标
bin/kpiperf -rats lte,nr,gsm -concurrency 100 -devices 3000 -db "$DB" -nats "$NATS"

# ② 只注入 1 万设备（不上传）
bin/kpiperf -mode seed -rats lte,nr,gsm -devices 9999 -db "$DB"

# ③ 拉满到 1 万并发（只 4G/5G）
ulimit -n 65535
bin/kpiperf -rats lte,nr -concurrency 10000 -devices 10000 -db "$DB" -nats "$NATS"

# ④ 单制式、多时间窗增量（5000 设备 × 8 窗 = 4 万文件）
bin/kpiperf -rats nr -concurrency 1000 -devices 5000 -buckets 8 -db "$DB" -nats "$NATS"

# ⑤ 只压文件存储（不连库、不验证入库；不注入设备时入库会失败，仅看存储能力）
bin/kpiperf -rats lte,nr,gsm -concurrency 1000 -devices 3000 -no-db

# ⑥ 用真机模板代替合成（每个 RAT 一个模板；4G 真机含重名 counter，需 productClass 命中才不撞自然键）
bin/kpiperf -rats lte,nr -templates 'test/pmperf/A20260611.0815...05000.xml,test/pmperf/A20260611.0900...04259.xml' -db "$DB" -nats "$NATS"

# ⑦ 跑完顺手清理
bin/kpiperf -rats lte,nr,gsm -concurrency 100 -devices 3000 -db "$DB" -nats "$NATS" -cleanup-after
```

## 一键清理（清设备 + KPI 数据 + NATS PM 流）

测试设备与 KPI 数据全部带 SN 前缀（默认 `KPILT-`）：

```bash
DB_DSN='postgres://omcgo:omcgo123@localhost:15432/omcgo?sslmode=disable' \
NATS_URL='nats://localhost:14222' \
bash test/pmperf/cleanup.sh
# 等价：bin/kpiperf -mode cleanup -sn-prefix KPILT -db "$DB" -nats "$NATS"
```

清理删除 `pm_metrics`（含 KPI）、`pm_files`、`devices`、`dead_letters`（按前缀），并清空 NATS `PM` 流。
单独清 NATS：`bin/kpiperf -mode purge -nats "$NATS"`。手工兜底 SQL：

```sql
DELETE FROM pm_metrics WHERE device_sn LIKE 'KPILT-%';
DELETE FROM pm_files   WHERE device_sn LIKE 'KPILT-%';
DELETE FROM devices    WHERE serial_number LIKE 'KPILT-%';
```

> 若 worker 已积压大量失败重投（如曾用无 productClass 的设备压测），`cleanup`/`purge` 清空 PM 流后，
> 建议 `docker restart omc-worker-1` 丢弃 worker 进程内的重试队列，给干净起点。

## 看图（性能管理页面）

入库后浏览器开 `http://localhost:18081` → 性能管理 → **设备视图**，选一个 `KPILT-LTE/NR/GSM-*` 设备 +
指标，时间范围「最近 7 天」即可看到曲线（KPI 行 `metric_type='kpi'`，`metric_path` 为 K 开头的指标 id）。
任务看板（默认页）需要内置/自定义 adhoc 任务命中对应平台指标。

## 报告解读

- **阶段一 文件存储**：每制式 + 合计的上传成功率、墙钟、吞吐（文件/s、MB/s）、p50/p99 时延、状态码。
- **阶段二 解析入库**：等待入库收敛后给出解析文件数、新增 `pm_metrics` 行（counter + KPI 分列）、
  入库吞吐（文件/s、行/s）、每制式新增 counter/KPI 行，以及 worker Prometheus 增量
  （成功/失败/迟到文件、丢弃 counter、单文件处理均值、上报延迟均值）。

> 「上报延迟均值」很大是因为压测把时间窗设在最近的已完成 15min 窗（end_time 在当前时刻之前若干分钟），
> 不是真实问题。入库文件数 < 上传成功数时查 worker 日志/指标（设备未注入 / 迟到压缩 chunk / DLQ）。

## 参数

| 参数 | 默认 | 说明 |
|------|------|------|
| `-mode` | `run` | `run`/`seed`/`cleanup`(清库+清 NATS)/`purge`(仅清 NATS PM 流) |
| `-rats` | `lte,nr,gsm` | 压测制式，逗号分隔；并发/设备数在其间拆分 |
| `-url` | `http://localhost:7557` | ACS 上传端点 |
| `-concurrency` | `100` | 总并发（在飞上限），按 `-rats` 拆分 |
| `-devices` | `9999` | 设备总数，按 `-rats` 拆分 |
| `-files` | `0` | 总文件数；0=各制式 devices×buckets |
| `-buckets` | `1` | 15min 时间窗个数（扩大唯一行空间） |
| `-templates` | 空 | 每 RAT 一个真机模板；留空=指标库合成（覆盖全部内置指标） |
| `-sn-prefix` | `KPILT` | 测试设备 SN 前缀（清理按此 LIKE） |
| `-oui`/`-carrier` | `48BF74`/`cmcc` | 设备 OUI / 运营商（分区键）|
| `-db` | `…@localhost:5432/omcgo` | PostgreSQL DSN（本机用 15432）|
| `-nats` | `nats://localhost:4222` | NATS URL（本机用 14222）|
| `-no-db` | `false` | 跳过注入与入库验证，仅压存储 |
| `-seed` | `true` | run 前注入设备（带 productClass）|
| `-cleanup-after` | `false` | run 结束后一键清理 |
| `-worker-metrics` | `http://localhost:9092/metrics` | worker Prometheus |
| `-drain` | `120s` | 上传后等待入库收敛最长时长 |
| `-timeout` | `30s` | 单次上传 HTTP 超时 |
| `-json` | `false` | JSON 输出 |
| `-insecure` | `false` | https 跳过 TLS 校验 |

## 实测基线（本机，2026-06-11）

100 并发拆 34/33/33，3 制式各 1000 设备 / 1000 文件：
- 存储：3000 文件 100% 成功，墙钟 **4.9s**，**606 文件/s、22 MB/s**，p99 ~400ms。
- 入库：全部解析，新增 **144 万行**（counter 129 万 + KPI 15.5 万），0 失败/0 丢弃，约 **50s 收敛**
  （~60 文件/s、~2.9 万行/s），单文件处理均 12.6ms。
