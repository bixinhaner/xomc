# Dashboard 全网预聚合直读发布与回滚

## 发布前门禁

在生产规模隔离副本先应用 TSDB 迁移，再运行：

```bash
psql "$TSDB_DSN" -v ON_ERROR_STOP=1 \
  -f omcgo/scripts/verify-dashboard-network-rollups.sql
```

必须同时满足：

- LTE、NR、GSM 的 hourly、daily、weekly 均有结果；
- 首页默认指标最近已完成的 24 个小时、30 个自然日、12 个自然周缺失清单为空；
- 不完整窗口符合已确认的业务状态；
- 执行计划只访问 `pm_aggregation_results`，使用 Dashboard 局部索引，无临时文件；
- 暖态数据库 P95 小于 200ms，接口 P95 小于 1s。

缺少指标时，只修正现有三制式内置全网任务的指标选择或映射，等待上卷任务产出后重新核验。禁止在 Dashboard 内扫描原始 PM 明细补算。

## 发布顺序

1. 在隔离生产规模副本应用迁移并通过覆盖与性能门禁。
2. 重启 postgres-tsdb，使 `pg_stat_statements` preload 和 `track_io_timing` 生效。
3. 启动或更新 OTel Collector、Prometheus、Alertmanager、Grafana。
4. 部署一个 app 实例，观察 15 分钟。
5. 条件满足后恢复全部 app 实例，继续观察 60 分钟。

观察指标：

- `dashboard_kpi_query_duration_seconds` P95；
- timeout、rejected、fresh/stale/miss、coalesced；
- `pm_network_rollup_lag_seconds`；
- TSDB temp bytes/files rate、超过 5 秒查询；
- postgres-tsdb CPU、内存和 block I/O。

## 停止条件

出现任一情况立即停止扩大发布：

- 首页指标口径冲突或覆盖缺失；
- 执行计划访问原始表或兼容视图；
- 单次查询产生临时文件；
- API P95 大于等于 1s，或数据库 P95 大于等于 200ms；
- postgres-tsdb CPU 达到 70% 并持续 10 分钟；
- timeout/rejected 持续增长，或 stale 超过 15 分钟。

## 回滚

回滚 app 到上一版本，但保留新增局部索引、`pg_stat_statements` 扩展和监控配置。旧版本会恢复重查询路径，因此回滚期间应临时限制首页访问，并持续保留 TSDB 资源告警。

不要删除全网预聚合结果、停用现有内置上卷任务或删除共享扩展。数据库参数如需降低日志量，可调整 `TSDB_LOG_MIN_DURATION_STATEMENT`；紧急关闭临时文件日志可设置 `TSDB_LOG_TEMP_FILES=-1`，指标采集仍应保持。
