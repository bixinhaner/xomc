# PM 数据完整性与聚合修订验收

## 固定通过条件

1. 以 LTE 设备身份处理 `gsm_payload_with_lte_identity.xml`：`pm_file_quarantines=1`，`pm_metrics=0`，`pm_aggregation_outbox=0`。
2. 处理 `lte_blq_valid.xml`：`C000010070`、`C000010080` 均命中，且不产生 whitelist miss。
3. 启用 `K900010006` 或 `K900010076`：所有递归 Counter 依赖在同一事务中自动启用；单独禁用仍被引用的 Counter 必须失败。
4. 20,000 文件批次在 12 分钟内排空：小时窗口不得因 timeout 提前发布；outbox 未消费时水位屏障持续阻塞关闭。
5. 已发布小时补入第 4 槽：创建可重试 rebuild，`revision` 增加，小时结果覆盖更新，并级联重建天、周/月结果；父窗口尚未关闭时必须回放后恢复 `open`，不得提前发布。
6. 同一窗口多个迟到文件只合并成一个 rebuild；运行期间又到文件时，完成后必须自动回到 `pending` 再跑一轮。`running` 任务租约超时后可被其他 worker 接管，旧 owner 不得续租或覆盖新 owner。
7. 当前日/周：Dashboard 返回 active task version 的 `partial=true` 快照；`received_slots/expected_slots` 使用自然周期口径，同时返回 `version_expected_slots`。
8. 旧任务版本只覆盖自然周期的一部分：`version_slice_complete` 与 `period_complete` 分开记录；活动版本按 `version_effective_from` 选择，乱序晚开的旧版本窗口不得覆盖新版本。
9. 当前周期 Redis/数据库读取失败或超时时，接口返回 `progress_state=unavailable`，Dashboard 明确提示，不得静默显示旧版本。
10. 永久失败 rebuild 按指数退避并让出队首，不能每秒热循环或饿死后续任务。
11. 聚合原始 outbox 至少保留 45 天；常规 raw metadata 清理不得提前删除重算回放源。
12. Dashboard 结果只按 5 分钟定时刷新；窗口聚焦、网络重连和业务事件不触发即时刷新。

## 数据库检查

```sql
SELECT declared_technology, detected_technology, reason, count(*)
FROM pm_file_quarantines
GROUP BY 1,2,3;

SELECT event_window_end, count(*) FILTER (WHERE consumed_at IS NULL) AS pending
FROM pm_aggregation_outbox
GROUP BY 1 ORDER BY 1 DESC;

SELECT task_id, task_version_id, granularity, window_start, status, revision,
       received_slots, expected_slots, data_complete, rebuild_requested_at
FROM pm_aggregation_windows
ORDER BY window_start DESC;

SELECT task_id, task_version_id, granularity, window_start, revision,
       received_slots, expected_slots, version_expected_slots,
       natural_expected_slots, version_slice_complete, period_complete
FROM pm_aggregation_results
ORDER BY window_start DESC;

SELECT status, granularity, attempts, last_error, requested_at,
       next_attempt_at, started_at, lease_expires_at, lease_owner, completed_at
FROM pm_aggregation_rebuilds
ORDER BY requested_at DESC;
```

## Prometheus 检查

```promql
sum by (declared_technology, detected_technology)
  (rate(omc_pm_technology_mismatch_files_total[15m]))

sum by (carrier, technology)
  (rate(omc_pm_whitelist_miss_values_total[15m]))

sum by (carrier, technology)
  (rate(omc_pm_known_disabled_values_total[15m]))

sum by (granularity)
  (rate(omc_pm_aggregation_rebuilds_total[15m]))

rate(omc_pm_aggregation_rebuild_errors_total[15m])
```
