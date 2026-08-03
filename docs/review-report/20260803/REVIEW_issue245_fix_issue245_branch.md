# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-08-03 |
| 范围 | fullstack-dashboard-migration |
| Issue | #245 |
| 分支 | fix/issue-245-dashboard-nr-rrc-mean |

## 变更概要

补齐 NR `C010070004`（`RRC.ConnMean`，RRC 连接平均数）在 Dashboard 的指标注册、数据库布局、NR 全网任务和前端 fallback 配置；同时让 Dashboard published 与当前周期快照查询接受 KPI/counter 两类结果。

## 审查结论

`PASS_WITH_WARNINGS`

### CRITICAL

无。

### WARNING

- `seed/000001_init_seed.sql` 是 consolidated baseline。现有数据库的 goose 版本为 1 时，迁移服务会成功但显示 `no migrations to run`，不会就地更新已有 `pm_tasks`/`dashboard_kpi_layouts`。本次已在隔离临时数据库中实际从 schema/seed baseline 载入并验证；部署既有环境仍须按 baseline runbook 重建或另行设计升级迁移。

## 业务完整性

- 指标字典：`C010070004` 已存在且定义为 counter/avg。
- 后端指标注册：已加入 NR traffic。
- Dashboard published 查询：保留 network、technology、published revision、时间窗口过滤，metric type 接受 KPI/counter。
- Dashboard partial 查询：保留 network、粒度、请求指标、时间窗口过滤，metric type 接受 KPI/counter。
- 数据库布局：NR traffic 布局加入该指标。
- PM 任务：NR 全网任务加入该指标；前端模板/非全网任务沿用既有任务职责，不扩大无关维度。
- 前端 fallback/mock：同步加入指标和 `number` 单位。

## 质量与安全

- SQL 仍使用 Squirrel 参数化构建。
- 未改变认证、权限、network 维度或 published revision 过滤。
- 未引入 any、ORM、字符串拼接 SQL或敏感信息日志。
- 新增 counter partial 回归测试，保留原有 network/非请求指标过滤断言。

## 验证证据

- `go test ./internal/dashboard -count=1`：通过。
- `go build ./...`：通过。
- `npm run typecheck`：通过。
- 前端布局/API 测试：28 项通过。
- `npm run build --workspace webcode`：通过，只有既有 Vite warning。
- `bash omcgo/scripts/check-migrations.sh`：通过，包含既有三条 baseline 流同号提示。
- 隔离数据库 `issue245_verify`：`000001_init_schema.sql` 实际成功，`000001_init_seed.sql` 实际成功；验证 NR layout 和全网任务均包含 `C010070004`；临时库已删除。

## 待 MR/部署验证

推送功能分支并创建 MR 后，在 251 部署最新镜像和 baseline 数据后，用浏览器确认首页 NR 请求包含 `C010070004`，并与性能模板结果对比 hourly/daily/weekly 点位。
