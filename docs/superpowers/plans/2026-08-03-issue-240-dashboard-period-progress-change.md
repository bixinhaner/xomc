# Issue #240：首页 KPI 当前日周进度修改方案

## 1. 现状与结论

Issue #240 要求首页日/周视图在自然周期尚未结束时展示进行中结果、版本有效区间、版本槽位覆盖和自然周期覆盖，同时保持小时视图只读取已发布小时结果，并且只按 5 分钟刷新。

当前 `main` 已包含提交 `b14a8ceb3 fix(dashboard): 补齐当前日周聚合进度`。现有实现已经具备：

- 后端 `GET /api/v1/dashboard/kpi-time-series` 的显式 `include_partial=true` 契约；
- `ProgressService` 读取 Redis 聚合状态并合成当前周只读预览；
- `period_progress`、版本有效区间、版本槽位覆盖和自然周期覆盖字段；
- 查询超时、并发、singleflight、fresh/stale cache 保护；
- 首页 hourly 走已发布接口，daily/weekly 才读取进行中快照；
- 前端 5 分钟轮询，并关闭 focus/reconnect 即时刷新；
- 后端、API 映射和进度标签测试。

因此本次不应重复实现功能，先完成运行环境恢复和真实接口验收；只有验收失败时才按下述边界补丁。

## 2. 修改边界

### 2.1 后端

检查以下契约，不改变正式发布链路：

- `omcgo/internal/dashboard/handler.go`：只有 daily/weekly 且显式 `include_partial=true` 返回 snapshot；hourly 和未 opt-in 保持旧 map。
- `omcgo/internal/dashboard/service.go`：published 曲线与 network partial 点按同一指标、粒度、时间窗口合并；进度不可用时保留 published 曲线并返回 `progress_state=unavailable`。
- `omcgo/internal/pm/stream/progress_service.go`：当前日按小时槽位计算，当前周合并已发布日与开放日，禁止写入正式聚合结果或提前发布。

### 2.2 前端

- `omcmb/frontend-core/src/services/api/dashboardApi.ts`：daily/weekly 请求带 `include_partial=true`，映射 `period_progress` 和 `progress_state`。
- `omcmb/frontend-core/src/hooks/api/useDashboard.ts`：daily/weekly 使用 snapshot，hourly 使用旧 published API；轮询固定 5 分钟。
- `omcmb/webcode/src/components/dashboard/LayoutKPIPanel.tsx`：展示覆盖率、版本区间和不可用状态；hourly 不显示进度标签。
- partial 点必须与 published 点有可辨识的图表语义，不能因映射为普通点而误导用户。

## 3. 执行顺序

1. Docker Desktop 恢复后，用最新 `main` 重建 app、worker、acs、web；运行 `migrate-schema`、`migrate-seed` 和 `migrate-tsdb-schema`，不删除现有数据卷。
2. 使用真实账号访问首页，分别验证 LTE、NR、GSM 的 hourly/daily/weekly 请求参数和响应。
3. 验证当前开放日有 partial 时，曲线保留 published 点并追加当前点；周视图不重复计算当前日。
4. 验证进度 Redis/TSDB/版本读取失败时，曲线仍显示且状态为 unavailable。
5. 仅在失败测试或真实请求证明缺陷后修改代码，并先补失败测试。

## 4. 验证命令

```bash
cd omcgo
go test ./internal/dashboard ./internal/pm/stream -run 'Test.*(KPITimeSeries|BuildCurrentWeeklyPreview|ProgressQueryResult|NaturalExpectedSlots)' -count=1
go build ./...

cd ../omcmb
npm run typecheck
npx vitest run frontend-core/src/services/api/__tests__/dashboardApi.test.ts webcode/src/components/dashboard/__tests__/LayoutKPIPanel.currentProgress.test.tsx
```

## 5. 验收标准

- 日/周视图显示当前开放周期的进行中值和覆盖率；
- 同时显示版本有效区间、版本槽位覆盖和自然周期覆盖；
- 小时视图不读取 15 分钟原始点、不显示 partial；
- 仅每 5 分钟刷新；
- 查询超时、并发、缓存和慢查询保护仍生效；
- 全量后端和前端验证通过，并保留真实浏览器请求证据。
