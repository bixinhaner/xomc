# Issue #190 Alarm Active Trend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让告警统计页趋势图使用每日未清除告警库存口径，且今天的最新点与当前告警统计一致。

**Architecture:** 保留现有 raised-event 服务方法，在同一路由增加 `metric=active` 分支。新增服务方法构造本地自然日快照，以批量 SQL 分别查询主库 active 与时序库 history，再按日期和级别合并。前端只为告警统计页显式传 active。

**Tech Stack:** Go 1.24、Gin、pgx/PostgreSQL、React、TypeScript、TanStack Query、Vitest。

## Global Constraints

- 仅修改 issue #190 所需的趋势接口、Hook 与告警统计页调用。
- 不改变默认 raised 口径，避免扩大已有接口影响面。
- 后端错误必须包装上下文；无数据库变更。
- 严格测试先行，每个测试先确认失败再写实现。

---

## Task 1: 固化 active 参数契约

- [x] 在 `omcgo/internal/dashboard/handler_test.go` 增加非法 `metric` 返回 400 的测试。
- [x] 运行 `cd omcgo && go test ./internal/dashboard -run TestDashHandler_AlarmTrend_InvalidMetric -count=1`，确认失败。
- [x] 在 `omcgo/internal/dashboard/handler.go` 校验 `metric=raised|active`，缺省 raised，并将 active 路由到新增服务方法。
- [x] 重跑测试确认通过。

## Task 2: 实现库存快照时间与合并逻辑

- [x] 在 `omcgo/internal/dashboard/service_test.go` 增加固定 Asia/Shanghai 时间下 7 日快照测试：前 6 日为日末、最后一天为 now。
- [x] 增加 active/history 分桶结果合并测试，覆盖级别字典码、缺失日期补零和升序。
- [x] 运行对应 Go 测试，确认因函数缺失失败。
- [x] 在 `omcgo/internal/dashboard/service.go` 新增快照构造、批量 SQL、结果扫描合并与 `GetActiveAlarmTrend`。
- [x] 重跑 `go test ./internal/dashboard`。
- [x] 根据独立审查补充跨库 ID 排除和设备组可见性测试并修复。

## Task 3: 告警统计页显式请求 active

- [x] 在 `omcmb/frontend-core/src/services/api/__tests__/dashboardApi.test.ts` 增加 `getAlarmTrend(7, 'active')` 请求参数契约测试。
- [x] 运行 `cd omcmb && npx vitest run frontend-core/src/services/api/__tests__/dashboardApi.test.ts`，确认失败。
- [x] 在 `omcmb/frontend-core/src/services/api/dashboardApi.ts` 为 `getAlarmTrend` 增加 `metric` 参数。
- [x] 在 `omcmb/frontend-core/src/hooks/api/useDashboard.ts` 将 metric 纳入 query key 并传给 API。
- [x] 在 `omcmb/webcode/src/pages/alarm/AlarmStatistics/index.tsx` 调用 `useAlarmTrend(days, 'active')`。
- [x] 重跑 Vitest。
- [x] 增加服务端日期桶为权威来源的前端测试，避免浏览器时区跨日丢点。

## Task 4: 格式化、全量验证与提交

- [x] 运行 `gofmt -w` 格式化改动 Go 文件。
- [x] 运行 `cd omcgo && go build ./... && go test ./...`（build 通过；全量测试仅既有 `internal/paramsync` 集成用例失败，定向重跑同样失败）。
- [x] 运行 `cd omcmb && npm run typecheck`。
- [x] 运行 Dashboard API Vitest。
- [x] 检查 `git diff --check` 与 diff，确认无范围外修改。
- [x] 请求代码审查，处理有效问题并重新验证。
- [x] 以 Conventional Commit 提交 issue #190。
