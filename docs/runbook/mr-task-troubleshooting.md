# Runbook：MR 测量任务（F05）故障排查

> **守护人**：F05 模块 Owner + 运维  
> **相关代码**：`omcgo/internal/mr/task/`  
> **PRD**：[`docs/project/prd/F05-mr-task-management.md`](../project/prd/F05-mr-task-management.md)  
> **开发计划**：[`docs/project/mr-task-management-development-plan-20260525.md`](../project/mr-task-management-development-plan-20260525.md)

---

## 1. 快速诊断决策树

```
症状                          →   先查
─────────────────────────────────────────────────
任务创建报 400                 →   入参校验：mr_type/period/end>start/cell 去重
任务创建报 500                 →   PG: mr_customize_task 表是否存在 + operator_code 一致性
任务 waitting 卡住不进 on      →   §3 调度器
任务 on 后 cell 全 unsupport   →   §4 平台匹配
任务 on 后 cell 全 openFailure →   §5 SPV 下发链路
cell health=abnormal 但实际在上报 →   §6 Redis 心跳误报
MR 文件落 MinIO 但 mr_files 表无记录 →   §7 解析链路
@daily cleaner 未删超期文件   →   §8 cleaner
Prometheus 看不到 omc_mr_*   →   §9 指标
```

---

## 2. 关键观测点

| 指标 | 解读 |
|---|---|
| `omc_mr_task_dispatched_total{op,result}` | 下发量与成败分布；`result="enqueue_failed"` 持续上涨 → task 队列异常 |
| `omc_mr_task_active_count` | 当前 'on' 任务数；瞬时跳变到 0 → 调度器误删 / DB 异常 |
| `omc_mr_file_uploaded_total{cell_code}` | 单 cell 上报频率；停止上涨且任务还 on → 设备 / 网络问题 |
| `omc_mr_heartbeat_missed_total` | 心跳缺失累计；正常应远低于上报次数 |
| `omc_mr_heartbeat_abnormal_total` | 跨阈值切到 abnormal 累计；建议接告警规则 |
| `omc_mr_files_cleaned_total{kind}` | 每天 cleaner 删除量；连续 7 天为 0 → cleaner 未启动 |

**Trace 视图**：搜 `mr.task.scheduler.tick` 看完整调度链路（含子 span `mr.task.open` / `mr.task.close`）。

---

## 3. 任务 waitting 卡住不进 on

**确认调度器在跑**：
```bash
# app 进程日志（scheduler 跑在 app）
grep "mr scheduler" /var/log/omcgo/app.log | tail -10
# 期望：started 一次 + 每 30s 不会有 error
```

**确认 Redis 分布式锁状态**：
```bash
redis-cli GET mr:scheduler:lock:open
# 若长期被占 → 检查抢到锁的实例日志，可能 panic
# 临时解锁：redis-cli DEL mr:scheduler:lock:open
```

**确认任务真的到期**：
```bash
psql -c "SELECT task_id, task_name, task_status, start_time, NOW() FROM mr_customize_task WHERE task_status='waitting';"
```

**确认时钟同步**：start_time 是 UTC，server 时钟漂移 > 30 秒会让调度器漏抓。

---

## 4. cell 全 unsupport — 平台匹配失败

代码：`internal/mr/task/platform.go`  
规则：`IsSupported(device.product_class)`，仅匹配 IntelCR / BLQ / MLQ / MLN 4 个平台。

**调查**：
```bash
psql -c "SELECT serial_number, product_class FROM devices WHERE serial_number IN (...);"
```

不在支持列表 → 不是 bug，预期行为。若是新平台（如 QAV3/QAV4）需要在 `platform.go` 加规则。

---

## 5. cell 全 openFailure — SPV 下发链路

**fault_code='enqueue_failed'**：task 队列出问题（Redis 不可达 / Repository PG 写失败）。检查 task 模块。

**fault_code 是数字（如 9001）**：设备应答了 SOAP Fault。可能原因：
1. **路径设备不识别** — 检查 `data/param-mappings/{平台}.xml` 是否含 `Device.FAP.MRMgmt.Config.{i}.*` 6 条 mapping
2. **Translator 翻译失败 fallback 到 standardPath，但厂商私有 path 不同** — 检查 app 日志 `mr-dispatcher` 看 `translator=true/false`，若 false 说明字典未命中（按 MR_Feature_Analysis.md §4，4 个平台预期 path 就是 standardPath，应该工作）

**fault_code='timeout'**：设备无响应。检查设备在线状态（最近一次 Inform 时间）。

---

## 6. Redis 心跳误报（cell 标 abnormal 但实际在上报）

**先看心跳 key 是否真的过期**：
```bash
redis-cli TTL MRFileReport_CELL001
# > 0 → 还在；-2 → 已过期
```

**对比 PG 心跳时间戳**：
```bash
psql -c "SELECT small_cell_code, last_heartbeat, missed_heartbeat, health_status FROM mr_customize_task_progress WHERE small_cell_code='CELL001';"
```

**如果 PG last_heartbeat 在更新但 Redis 已过期** → 说明 heartbeat subscriber 写 Redis 失败（Redis 故障 / 网络）。短期内会触发 abnormal 误报，恢复 Redis 即自愈。

**长期治理**：调高 `mr.heartbeat_miss_threshold`（默认 2 次，可改 3）。

---

## 7. MR 文件落 MinIO 但 mr_files 无记录 — 解析链路

**MR 文件入口有两条**（按设备 / OMC 部署方式）：

| 路径 | 触发 | 事件 | 消费者 |
|---|---|---|---|
| AutonomousTransferComplete | 设备主动通知 ACS 文件已上传 | `mr.file.received` | `mr.Collector` |
| 直接 HTTP POST 上传 | 设备 POST 到 `/smallcell/FileUploadService` | `mr.file.uploaded` + `mr.file.received` | `mr.task.HeartbeatSubscriber` + `mr.Collector` |

**排查**：
```bash
# 1. MinIO 是否真有文件
mc ls minio/mr-files/cmcc/$(date +%Y/%m/%d)/

# 2. NATS 事件是否到 worker
nats sub mr.file.received

# 3. worker 日志
grep "MR collector\|MR file processed" /var/log/omcgo/worker.log | tail -20
```

**常见原因**：worker 没注入 `DeviceLookup`，直传路径 payload 不带 device_id 时 collector 报错。检查 `cmd/worker/main.go` 中 `mrCollector.SetDeviceLookup(...)` 是否调用。

---

## 8. cleaner 未删超期文件

**确认 cleaner 启动**：
```bash
grep "mr cleaner started" /var/log/omcgo/app.log
```

**确认配置**：
```bash
grep -A 3 "^mr:" /etc/omcgo/app/config.yaml
# file_save_days: 3   ← 必须 > 0
```

**手动触发一次**（如有 admin 端点；当前没有 — 等下次 @daily cron）：
```bash
# 紧急情况：临时把 file_save_days 调小，重启 app 进程
```

---

## 9. Prometheus 看不到 omc_mr_* 指标

1. **app 进程 metrics 端口可达**：`curl http://app:9091/metrics | grep omc_mr`
2. **首次启动后还没产生数据**：dispatcher 在第一次下发 SPV 后才会让 `_dispatched_total` 出现。等任务触发即可。
3. **Prometheus scrape 配置**：确认 prometheus.yml 含 omcgo-app job。

---

## 10. 紧急操作清单

| 操作 | 命令 | 影响 |
|---|---|---|
| 停某任务 | `POST /api/v1/mr/tasks/:id/stop` `{"operator_code":"cmcc"}` | 下发关闭 SPV，不删数据 |
| 删某任务 | `DELETE /api/v1/mr/tasks/:id?operator_code=cmcc` | 物理删任务 + 级联删 progress；要求 off/termination 状态 |
| 临时关闭整个 MR 调度 | app 进程 yaml `mr.scheduler_interval_seconds: 0` 重启 app | scheduler 不启动；已下发的任务设备继续上报 |
| 清干净所有心跳 key | `redis-cli --scan --pattern "MRFileReport_*" \| xargs -r redis-cli DEL` | 触发短期"全部 abnormal"误报 1 个上报周期 |

---

## 11. 已知 gap / 数据问题

### 11.1 IntelCR 平台无独立 ParamModel XML

`data/param-mappings/` 下有 BLQ.xml / MLN.xml / MLQ.xml 但**无 IntelCR.xml**。当前 IntelCR 设备的 ParamRegistry mapping 走默认（standardPath = privatePath），与 MR_Feature_Analysis.md §4 的"四平台路径一致"假设吻合。

**如果某天 IntelCR 厂商引入私有 path**：在 `data/param-mappings/` 加 IntelCR.xml + 加 6 条 `Device.FAP.MRMgmt.Config.{i}.*` mapping，重启 app（dictloader 自动加载）。

### 11.2 mr_records（TimescaleDB hypertable）的清理

`cleaner.go` 只删 `mr_files` 表元数据 + MinIO 对象，**不删 mr_records hypertable**。后者由 TimescaleDB retention policy 单独管（参 `migrations/000006_alarms_mr_firmware.sql` 中 `add_retention_policy`）。

如果发现 mr_records 数据量异常 → 检查 retention policy 是否仍启用：
```sql
SELECT * FROM timescaledb_information.policies WHERE hypertable_name='mr_records';
```
