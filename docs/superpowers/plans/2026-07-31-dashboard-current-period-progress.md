# Dashboard Current Period Progress Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让首页和性能仪表盘在不改变正式日→周发布链路的前提下，展示当前日、当前周的只读进行中 KPI 快照、自然周期覆盖率和明确的不可用状态。

**Architecture:** 继续以 Redis Counter 累加器作为唯一进行中状态源。`ProgressService` 将当前日累加器与本周已发布日贡献的周累加器在内存中合并，生成不落库、不发事件、不关窗的周预览；首页仅对固定全网任务的 `network` 实体读取该预览，并通过显式 `include_partial=true` 响应契约接入。原 `/dashboard/kpi-time-series` 响应在未显式 opt-in 时保持不变，正式 published 结果、迟到重算和日→周链路保持不变。

**Tech Stack:** Go、pgx、Redis、Squirrel、Gin、React、React Query、TypeScript、Vitest

## Global Constraints

- 首页只支持 `hourly`、`daily`、`weekly`，不增加 15 分钟粒度。
- 正式聚合继续使用小时→日→周/月事件链；预览只读，不写 `pm_aggregation_results`，不发布 rollup 事件。
- 当前日覆盖率按小时槽位，当前周覆盖率按 7×24 个小时槽位；同时保留 `version_expected_slots`。
- 首页只读取 LTE/NR/GSM 固定全网任务的 `network` 实体，不扫描 20,000 个设备窗口。
- Redis/TSDB/任务版本读取失败时保留 published 曲线，但返回 `progress_state=unavailable`，不得伪装为完整或零值。
- Dashboard 查询继续受现有超时、并发、singleflight 和 fresh/stale cache 保护。
- 仅每 5 分钟定时刷新；`refetchOnWindowFocus=false`、`refetchOnReconnect=false`，不增加事件即时刷新。
- 所有生产改动必须先有能证明缺陷的失败测试。

---

### Task 1: 生成精确的当前周只读预览

**Files:**
- Modify: `omcgo/internal/pm/stream/progress_service.go`
- Modify: `omcgo/internal/pm/stream/progress_service_test.go`

**Interfaces:**
- Consumes: 当前日 `WindowState`、可选的当前周 `WindowState`、活动 `TaskVersionSnapshot`
- Produces: `buildCurrentWeeklyPreview(version, dailyKey, dailyState, weeklyState, location) (WindowKey, WindowState, progressCoverage, error)`

- [ ] **Step 1: 写失败测试**

构造本周无持久 weekly 状态、当前日收到 4/24 小时的 daily 状态，断言合成周预览包含同一 Counter 的精确 sum/count/min/max，覆盖率为 `4/168`，不修改输入状态。

- [ ] **Step 2: 运行 RED**

Run:

```bash
cd omcgo
go test ./internal/pm/stream -run 'TestBuildCurrentWeeklyPreview' -count=1
```

Expected: FAIL，预览函数尚不存在。

- [ ] **Step 3: 实现最小纯函数**

使用 `accumulatorDefinitionID` 合并累加器；周边界通过 `WindowFor(..., weekly, businessLocation)` 计算。若已有 weekly 状态，先保留此前已发布日的精确累加器，再仅合入仍为 open 的当前日状态。覆盖率使用：

```go
receivedHours := weeklyState.ReceivedSlots*24 + dailyState.ReceivedSlots
naturalHours := int64(7 * 24)
versionHours := expectedVersionChildWindows(weeklyWindow, GranularityHourly, version, location)
```

- [ ] **Step 4: 接入 `ProgressService.Query`**

一次读取 active daily/weekly Redis 状态；当 daily 活动窗口存在时，用合成预览替换同自然周的直接 weekly preview。没有 current daily 时保留已有 weekly preview。读取失败统一包装 `ErrProgressUnavailable`。

- [ ] **Step 5: 运行 GREEN 和回归**

Run:

```bash
cd omcgo
go test ./internal/pm/stream -run 'TestBuildCurrentWeeklyPreview|TestProgress|TestNaturalExpectedSlots' -count=1
```

Expected: PASS。

---

### Task 2: 为首页提供向后兼容的 partial API

**Files:**
- Modify: `omcgo/internal/dashboard/service.go`
- Modify: `omcgo/internal/dashboard/handler.go`
- Modify: `omcgo/internal/dashboard/kpi_series_service_test.go`
- Modify: `omcgo/internal/dashboard/handler_test.go`
- Modify: `omcgo/cmd/app/provider/pm.go`
- Modify: `omcgo/cmd/app/provider/modules.go`

**Interfaces:**
- Produces:

```go
type KPITimeSeriesSnapshot struct {
    Series         KPITimeSeriesResponse      `json:"series"`
    PeriodProgress []pmstream.PeriodProgress `json:"period_progress"`
    ProgressState  string                    `json:"progress_state"`
}
```

- [ ] **Step 1: 写首页服务失败测试**

用 fake `NetworkProgressReader` 返回一个 `dimension=network`、`granularity=daily`、`partial=true` 的 KPI，断言 published 曲线保留且当前日 partial 点追加；非 network、非请求 KPI、错误粒度不得混入。

- [ ] **Step 2: 写 handler 契约失败测试**

断言无 `include_partial` 时仍返回旧 map；`include_partial=true` 时返回 `{series, period_progress, progress_state}`；进度读取失败时 HTTP 仍成功、published 曲线保留、状态为 `unavailable`。

- [ ] **Step 3: 运行 RED**

Run:

```bash
cd omcgo
go test ./internal/dashboard -run 'Test.*KPITimeSeries.*Partial|Test.*CurrentPeriod' -count=1
```

Expected: FAIL，snapshot 契约和 progress reader 尚不存在。

- [ ] **Step 4: 实现服务和依赖注入**

Dashboard 只按 `BuiltinNetworkTaskID(technology)` 查询固定全网任务。组合 published 与 partial 的整个闭包走现有 `KPIQueryGuard`；`ProgressService` 自身保留 2.5 秒上限。`pmHandlerDeps` 复用同一个 `ProgressService` 实例，避免重复构造。

- [ ] **Step 5: 实现向后兼容 handler**

只有 `include_partial=true` 且粒度为 daily/weekly 时返回 snapshot；hourly 或未 opt-in 继续返回旧 map。进度错误转换为 `progress_state=unavailable`，不回退原始 PM 表。

- [ ] **Step 6: 运行 GREEN**

Run:

```bash
cd omcgo
go test ./internal/dashboard ./cmd/app/provider -count=1
```

Expected: PASS。

---

### Task 3: 首页展示进行中点和覆盖率

**Files:**
- Modify: `omcmb/frontend-core/src/types/dashboard.ts`
- Modify: `omcmb/frontend-core/src/services/api/dashboardApi.ts`
- Modify: `omcmb/frontend-core/src/services/api/__tests__/dashboardApi.test.ts`
- Modify: `omcmb/frontend-core/src/hooks/api/useDashboard.ts`
- Modify: `omcmb/frontend-core/src/hooks/api/useDashboard.test.ts`
- Modify: `omcmb/webcode/src/pages/dashboard/DashboardKPIModules.tsx`
- Modify: `omcmb/webcode/src/components/dashboard/LayoutKPIPanel.tsx`
- Modify: `omcmb/webcode/src/components/dashboard/__tests__/LayoutKPIPanel.currentProgress.test.tsx`
- Modify: `omcmb/frontend-core/src/i18n/zh-CN/index.ts`
- Modify: `omcmb/frontend-core/src/i18n/en-US/index.ts`

**Interfaces:**
- Consumes: Task 2 的 `series`、`period_progress`、`progress_state`
- Produces: 当前日/周曲线 partial 点和“进行中 4/24（16.7%）”状态；失败显示“进行中状态暂不可用”

- [ ] **Step 1: 写 API 与窗口失败测试**

断言 daily 窗口包含今天，weekly 窗口包含本周；daily/weekly 请求带 `include_partial=true`，hourly 不带；映射保留 `periodProgress` 和 `progressState`。

- [ ] **Step 2: 写 UI 失败测试**

断言 `available` 显示自然周期覆盖率和版本有效区间，`unavailable` 显示明确警告，hourly 不显示 partial 标签；全部文案通过 i18n。

- [ ] **Step 3: 运行 RED**

Run:

```bash
cd omcmb
npx vitest run frontend-core/src/services/api/__tests__/dashboardApi.test.ts webcode/src/components/dashboard/__tests__/LayoutKPIPanel.currentProgress.test.tsx
```

Expected: FAIL。

- [ ] **Step 4: 实现 API、Hook 和 UI**

新增 `getKPITimeSeriesWithProgress`，仅 `useDashboardKPIWindowSeries` 使用。保持现有 5 分钟轮询参数不变，不注册 focus/reconnect/业务事件刷新。`LayoutKPIPanel` 在标题区显示紧凑状态标签及详情 Tooltip。

- [ ] **Step 5: 运行 GREEN 和类型检查**

Run:

```bash
cd omcmb
npx vitest run frontend-core/src/services/api/__tests__/dashboardApi.test.ts webcode/src/components/dashboard/__tests__/LayoutKPIPanel.currentProgress.test.tsx
npm run typecheck
```

Expected: PASS。

---

### Task 4: 全量验证、MR、部署和线上验收

**Files:**
- Update: `docs/qa-report/pm-integrity-acceptance.md`（仅补充实际验证证据时修改）

- [ ] **Step 1: 后端全量门禁**

```bash
cd omcgo
go build ./...
go test ./... -count=1
go vet ./...
```

- [ ] **Step 2: 前端全量门禁**

```bash
cd omcmb
npm run typecheck
npm test -- --run
```

- [ ] **Step 3: 提交并创建 MR**

提交信息：

```text
fix(dashboard): 补齐当前日周聚合进度
```

推送 `codex/dashboard-current-period-progress`，创建非 Draft MR，确认流水线通过后合入。

- [ ] **Step 4: 部署合入后的最新 main**

构建新版本并通过 compose 部署；不得清空本轮线上数据，以证明真实历史周状态与当前日状态可以无损合成。

- [ ] **Step 5: 业务与性能验收**

验证：

```text
首页 hourly 只读 published；
首页 daily/weekly 含当前 partial 点、覆盖率、活动版本；
性能仪表盘 daily/weekly 同口径；
K900010006/K900010076 当前槽位完整；
PM/NATS/任务队列最终排空且无持续增长；
无新增 5xx/503、无 admission reject、无慢 SQL/锁等待；
CPU、内存、磁盘 I/O 在批次后回落；
小时结果按 12 分钟水位关闭，日/周预览不触发提前发布；
刷新仅每 5 分钟发生。
```

- [ ] **Step 6: 若发现新异常，回到 Task 1**

每个异常重新执行根因取证、失败测试、最小修复、全量门禁、MR、部署和线上复验，直到固定验收条件全部满足。
