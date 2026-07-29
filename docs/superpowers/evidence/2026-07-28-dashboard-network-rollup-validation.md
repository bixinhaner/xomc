# Dashboard 全网预聚合查询上线验证

## 交付信息

- 分支：`codex/dashboard-network-rollup-hardening`
- 部署版本：`0.1.6-20260729-0120`
- 部署提交：`59f7f5b4f`
- 目标环境：`172.24.224.197`
- 发布时间：2026-07-29（Asia/Shanghai）

## 自动化验证

- 后端：`go build ./...`、`go test ./...` 通过。
- 竞态：`go test -race ./internal/dashboard ./internal/pm/stream` 通过。
- 前端：`npm run typecheck` 通过。
- Dashboard 相关前端测试：4 个文件、56 个用例通过。
- 部署与监控回归：`storage-compose_test.sh` 87/87 通过，`monitoring-profile_test.sh` 通过。
- 发布包在本地和服务器分别校验 SHA-256，一致值为
  `dab3f1f5fdb0abb47e00db1de4e13768fb51cde7c122c6e65caf6235ebe3c975`。

## 线上验证

- 安装脚本完整健康检查 25/25 通过。
- app、ACS、worker、web 均运行 `0.1.6-20260729-0120`，重启次数均为 0。
- 浏览器实际页面显示最终版本，默认选择 eNB（LTE）和小时粒度。
- 首页提供小时、天、周三个现有聚合粒度；访问日志只出现
  `granularity=hourly|daily|weekly`，没有 15 分钟查询。
- 定时请求日志在 17:04、17:09、17:14 出现，符合只按 5 分钟刷新；焦点和重连即时刷新已关闭。
- LTE 小时模式下业务量、利用率、保持性、移动性图表已渲染；无结果指标只保留自身空态，
  不再因 JSON `null` 破坏整批 16 个指标。
- 最近 5 分钟 Dashboard 查询 P95 约 24 ms，请求速率约 0.007 次/秒。
- 主机磁盘实时写入约 2.7 MiB/s；Docker `BlockIO` 为生命周期累计量，不能当作实时速率。

## 线上保护措施

- Dashboard 查询总超时 3 秒，数据库 statement timeout 2.5 秒。
- 并发上限 4，排队上限 100 ms；过载映射为 503，查询超时映射为 504。
- 结果缓存 fresh 4 分 30 秒、stale 15 分钟，并使用 singleflight 合并同键查询。
- `pg_stat_statements`、`track_io_timing`、1 秒慢 SQL、临时文件日志均已启用。
- Prometheus 已实际加载 `omc.dashboard_kpi` 10 条和
  `omc.tsdb_query_resources` 3 条规则，13/13 均为 `health=ok`。
- 升级安装会定向重建 6 个配置 bind mount 的监控容器，并使用 `--no-deps`
  避免连带重启 PostgreSQL、TSDB、Redis 等基础设施。

## 数据边界

线上数据库当前只有 LTE 全网小时聚合存在已发布结果；LTE 日/周以及 NR/GSM
小时、日、周尚无结果。`pm_metrics_hourly/daily/weekly` 等兼容对象是
`pm_aggregation_results` 的视图，并非另一套可回退读取的历史物化表。
首页严格读取现有全网预聚合结果，不回退扫描原始 PM 明细。

当前还观察到两个独立于本次 Dashboard 查询链路的存量问题：LTE 内置聚合成员刷新会触发
PostgreSQL 65535 参数上限，TSDB shadow-dim 清理存在 20–30 秒慢 SQL。它们已被新增慢查询
指标和资源告警捕获，但不应通过恢复 Dashboard 原始明细扫描来规避。

## 已知基线

仓库全量前端测试在本分支改动前已有 14 个无关失败；本次相关测试、类型检查和生产构建均通过。
