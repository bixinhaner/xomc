# Issue #191 Alarm Heatmap and Top Devices Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在不扩大告警统计页范围的前提下，让热力图可按级别观察，并让 Top10 真实反映当前未清除告警的级别构成。

**Architecture:** 热力图复用两个既有 API，由前端根据 severity 选择请求并构造级别色阶。Top10 新增一个受设备组权限约束的 Dashboard 查询端点，前端共享层负责 snake_case 映射，页面用既有 BarChart 渲染四级堆叠条形。

**Tech Stack:** Go 1.24、Gin、Squirrel、pgx/PostgreSQL、React、TypeScript、TanStack Query、ECharts、Vitest。

## Global Constraints

- 只修改 Alarm Heatmap 和 Top Alarm Devices。
- 不改 Summary、Dashboard 首页和共享 BarChart。
- 所有新增可见文字必须走 zh-CN/en-US i18n。
- Top10 固定 10 条，只查 `alarms_active`，不引入通用筛选参数。
- 严格测试先行。

---

## Task 1: 新增 Top10 后端端点

- [x] 新建 `omcgo/internal/dashboard/top_alarm_devices_test.go`，先测试 SQL 的 active 表、四级别名、分组、确定性排序、Limit 10 和可见组过滤。
- [x] 增加 Handler 路由与权限上下文失败测试。
- [x] 运行新增 Go 测试，确认失败。
- [x] 新建 `omcgo/internal/dashboard/top_alarm_devices.go`，实现 DTO、Squirrel 查询、扫描与错误包装。
- [x] 在 `handler.go` 注册/实现端点，并在生产 router 注入可见组 resolver。
- [x] 重跑 `go test ./internal/dashboard ./cmd/app/provider`。

## Task 2: 前端接入真实 Top10 并堆叠显示

- [x] 在 Dashboard API 契约测试中断言专用端点和完整四级映射，确认旧实现失败。
- [x] 更新共享 TopAlarmDevice 类型、API 映射和 Mock。
- [x] 在告警统计页用完整 SN、四级 stack 系列、标准颜色和完整 Tooltip 渲染；点击继续传完整 SN。
- [x] 运行 Dashboard API Vitest 和 typecheck。

## Task 3: 热力图增加级别维度

- [x] 新增纯模型测试，覆盖 all 请求总热力图、指定级别请求 by-severity 以及色板选择。
- [x] 运行测试确认失败。
- [x] 增加 severity 状态与 selector，将 days+severity 纳入 query key。
- [x] 显示数值 visualMap，并在 Tooltip 显示本地化星期、小时、级别和数量。
- [x] 补充 zh-CN/en-US i18n 文案，移除组件硬编码可见文字。
- [x] 运行相关 Vitest 和 typecheck。

## Task 4: 验证、审查与提交

- [x] 运行 `gofmt -w`。
- [x] 运行 `cd omcgo && go build ./... && go test ./...`（build 通过；全量测试仅既有 `internal/paramsync` 集成用例失败，与 #190 分支结果一致）。
- [x] 运行 `cd omcmb && npm run typecheck`。
- [x] 运行 Dashboard API 与图表模型 Vitest。
- [x] 运行 `git diff --check` 并复核范围。
- [x] 请求独立代码审查并处理有效问题。
- [x] 以 Conventional Commit 提交 issue #191。
