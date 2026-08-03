# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-08-03 11:37 |
| 提交基线 | dc04aa57e |
| 作者 | hezhenguo1046 |
| 范围 | fullstack-dashboard-migration |
| 变更文件数 | 10 |
| 新增行数 | +29 |
| 删除行数 | -14 |

## 变更概要

本次修复 #245：将 NR 指标 `C010070004`（`RRC.ConnMean`，RRC 连接平均数）加入 Dashboard 的数据库布局、Go/前端 fallback 布局、NR 指标注册和四类内置 NR 聚合任务。Dashboard 已发布结果和当前周期快照查询同时接受 KPI 与 counter 类型，并新增 counter partial 快照回归覆盖。

## 审查发现

### CRITICAL

无。

### WARNING

1. `omcgo/migrations/seed/000001_init_seed.sql` 是 consolidated baseline，`pm_tasks` 和 `dashboard_kpi_layouts` 的 `INSERT ... ON CONFLICT DO NOTHING` 不会更新已经存在的数据库记录。本修改对全新安装或允许清库重建的环境有效；既有环境必须按 migration baseline runbook 备份并重建，不能仅重新执行 goose seed 期待在线更新。

### INFO

1. 251 当前环境的真实页面曾显示旧 NR 布局和旧任务指标集合；合入后需要重新部署/重建 seed 后，浏览器重新确认首页请求包含 `C010070004`，并对比性能模板返回点。
2. 前端生产构建仍有既有 Vite chunk size 和 Rolldown/esbuild deprecation 警告，本次未引入新的构建错误。

## 详细分析

### `omcgo/internal/dashboard/network_rollup_repository.go`

使用 Squirrel 将 published network rollup 的 `metric_type` 从仅 KPI 扩展为 KPI/counter 参数集合，保留 network 维度、technology task、粒度、时间窗口和 published revision 过滤。没有字符串拼接 SQL，也没有放宽权限边界。

### `omcgo/internal/dashboard/service.go`

当前周期快照只接受 network 维度、请求粒度、KPI 或 counter 类型，并继续按请求 metric path、时间范围过滤。未改变其他维度或 15 分钟数据回退逻辑。

### `omcgo/internal/dashboard/kpi_series_service_test.go`

新增 `C010070004` counter partial 点断言，验证请求指标会被保留，且原有 network 过滤和非请求指标过滤仍有效。

### `omcgo/internal/dashboard/kpi_layout_default.go`

Go fallback NR traffic panel 加入 `C010070004`，与数据库布局和前端 fallback 保持一致。

### `omcgo/internal/dashboard/kpi_panel_registry.go`

将 `C010070004` 注册为 NR/GNB traffic 指标，使动态 KPI 定义接口与首页布局使用同一编号。

### `omcgo/migrations/seed/000001_init_seed.sql`

在既有 `000001` seed baseline 内更新 NR 数据库布局和四类 NR 内置任务指标数组。SQL 仍使用单条多行 `INSERT ... VALUES ... ON CONFLICT DO NOTHING`，未新增 `000002+` 文件，符合当前未封版迁移规则。

### `omcmb/webcode/src/pages/dashboard/layoutMapping.ts`

前端 NR fallback layout 加入该指标，真实布局接口为空或非法时不会遗漏。

### `omcmb/frontend-core/src/mock/services/dashboardService.ts`

Mock layout 和指标元数据同步加入该指标，单位为 `number`，名称为“RRC连接平均数”。

## 业务完整性检查

业务链路完整，无遗漏：字典定义 → 内置任务指标白名单 → published/partial Dashboard 查询 → Go/前端布局 → 指标注册与 Mock → 回归测试。

## 业务影响范围检查

变更范围可控，未发现接口签名、数据库 schema、事件契约、认证中间件或权限边界变化。新增 counter 查询仍限定 network 维度和已发布 revision；seed 对既有库的适用范围见 WARNING。

## 前后端一致性检查

前后端指标编号、NR panel、名称和单位保持一致。Dashboard API 路径和响应结构未改变，仅扩大允许读取的已请求 metric type 范围。

## 代码质量回退检查

未发现代码质量回退：未删除测试、未移除错误处理、未引入 any、未绕过 Squirrel、未放宽认证或权限校验。

## 配套更新提醒

- **文档**：无需更新架构文档；部署时应遵循现有 `migrations/README.md` 和 baseline runbook。
- **单元测试**：已有 Dashboard SQL、snapshot、layout/API 测试覆盖本次逻辑。
- **端到端测试**：建议部署后补充真实浏览器证据：NR 首页请求包含 `C010070004`，并验证 hourly/daily/weekly 返回点；当前未修改 E2E 脚本。

## 安全检查

未发现安全问题。查询仍使用参数化 SQL，未记录 token、密码或敏感数据。

## 性能检查

未发现明显性能问题。metric type 仍是固定两值集合，查询仍使用现有 published/network/technology/time filters；指标数量仅增加一个 NR counter。

## 测试覆盖

已通过：

- `go test ./internal/dashboard -count=1`
- `go build ./...`
- `npm run typecheck`
- 前端布局/API 定向测试：28 项通过
- `npm run build --workspace webcode`
- `bash omcgo/scripts/check-migrations.sh`

## 总结

| 级别 | 数量 |
|------|------|
| CRITICAL | 0 |
| WARNING | 1 |
| INFO | 2 |

**审查结论**: `PASS_WITH_WARNINGS`
