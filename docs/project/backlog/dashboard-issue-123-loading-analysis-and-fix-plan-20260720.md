# Issue #123：Dashboard 顶部 KPI 卡长时间空白——最小修复设计

> 日期：2026-07-20
> 代码基线：`main@fbe7a208d229`
> Issue：<http://192.168.10.16/netmanager/xomc/-/issues/123>
> 范围：首页顶部“总设备数量、在线设备、活跃告警、活跃 UE”四张 KPI 卡

---

## 1. 结论

本 Issue 按最小改动处理，只修改前端：

1. Dashboard 首页由 `useDashboardData()` 改用已有的 `useDashboardSummary()`。
2. 移除首页对 `getDashboardData()` 全量 `Promise.all` 的依赖，首页只请求一次 `/dashboard/summary`。
3. 在浏览器保存四张卡上一次成功显示的值。
4. 新请求未完成或失败时继续显示上一次成功值；成功后再更新。
5. 活跃 UE 在 Summary 成功但没有 `UE_ACTIVE` 时显示 `0`。
6. 保留 Dashboard SSE 刷新能力。

不修改：

- KPI 指标编号。
- KPI 指标库 XML。
- KPI 采集、入库、聚合和查询 SQL。
- `/dashboard/summary` 后端实现。
- 其他 Dashboard 图表和页面。
- 全局 React Query 配置。

---

## 2. 已确认的问题

Dashboard 首页当前调用：

```tsx
const { data: dashboardData, isLoading } = useDashboardData();
```

`useDashboardData()` 调用 `getDashboardData()`，后者通过 `Promise.all` 等待：

- Summary
- 告警趋势
- 设备状态
- Top 告警设备
- 区域统计
- 告警类型
- Widgets

其中 `getTopAlarmDevices()` 又调用一次 `/dashboard/summary`，所以 Network 中会出现两个 Summary。

四张 KPI 卡只消费：

```ts
dashboardData.summary
```

没有消费 `chartData` 和 `widgets`。因此首页为四张卡等待整页聚合没有必要。

相关代码：

- [`omcmb/webcode/src/pages/dashboard/index.tsx`](../../../omcmb/webcode/src/pages/dashboard/index.tsx)
- [`omcmb/frontend-core/src/hooks/api/useDashboard.ts`](../../../omcmb/frontend-core/src/hooks/api/useDashboard.ts)
- [`omcmb/frontend-core/src/services/api/dashboardApi.ts`](../../../omcmb/frontend-core/src/services/api/dashboardApi.ts)

---

## 3. 修复方案

### 3.1 改为单独查询 Summary

Dashboard 页面改为：

```tsx
useDashboardRealtime();
const { data: dashboardSummary, isPending } =
  useDashboardSummary(dashboardApiScope, currentUserId);
```

四张卡和趋势数据直接读取 `dashboardSummary`。

这样首页只产生一次 `/dashboard/summary`，不再等待其他图表接口，也不会通过 `getTopAlarmDevices()` 产生第二次 Summary。

`getDashboardData()` 暂不删除，因为设备资源统计页仍在使用；本 Issue 只解除 Dashboard 首页对它的依赖。

### 3.2 保存上一次成功值

新增一个小型前端工具，只保存四张卡：

```ts
interface DashboardCardSnapshot {
  totalDevices: number;
  onlineDevices: number;
  activeAlarms: number;
  activeUE: number;
  updatedAt: number;
}
```

存储位置：

```text
localStorage
```

存储键按“实际 API 服务 + 当前用户 ID”区分，避免本地开发切换测试服务器或同一浏览器切换账号时串数据：

```text
xomc:dashboard-card-snapshot:v1:<api-scope>:<user-scope>
```

其中：

- 存在 `VITE_API_BASE_URL` 时按实际 base URL 解析。
- 生产环境使用相对 base URL 时，以 `window.location.origin` 解析。
- 本地开发存在 `VITE_API_PROXY_TARGET` 时使用该目标地址。

只在 `/dashboard/summary` 成功返回后更新快照。请求 pending、超时或失败时不得把默认值写入快照。

### 3.3 展示优先级

每次渲染按以下优先级取值：

```text
本次成功响应 > 上一次成功快照 > 首次加载默认值
```

状态规则：

| 当前 Summary | 快照 | 卡片展示 |
|---|---|---|
| pending | 无 | 骨架 |
| pending | 有 | 上一次成功值 |
| success | 任意 | 本次新值，并更新快照 |
| error | 有 | 上一次成功值 |
| error | 无 | `0`；活跃 UE 显示 `0` |

卡片的 loading 条件改为：

```ts
isPending && !snapshot
```

因此刷新页面时，只要存在上一次成功快照，卡片不会重新变成空白骨架。
使用 `isPending` 也能覆盖首次查询因离线而暂停、尚未取得任何数据的状态。

### 3.4 活跃 UE

本 Issue 不调整 KPI 数据链。

继续读取现有字段：

```ts
dashboardSummary?.kpiSummary?.['UE_ACTIVE']
```

显示规则：

```ts
const activeUE =
  typeof ueRaw === 'number'
    ? Math.floor(ueRaw)
    : snapshot?.activeUE ?? 0;
```

含义：

- 后端返回数值，包括真实 `0`：显示该数值。
- 新请求尚未结束：优先显示上一次成功值。
- Summary 成功但没有 `UE_ACTIVE`：显示并保存 `0`。
- 没有快照且请求失败：显示 `0`。

KPI 编号、指标定义或全网聚合口径如需调整，应作为独立 Issue 处理，不能混入 #123。

### 3.5 更新时间

页面现有“最后更新时间”应表示最近一次成功 Summary 的时间：

- 有快照时，初始值使用 `snapshot.updatedAt`。
- Summary 成功后，使用 React Query 的成功更新时间并写入快照。
- 手动刷新失败时，不更新“最后更新时间”。

---

## 4. 文件范围

新增：

- `omcmb/frontend-core/src/hooks/api/useDashboardSummary.test.tsx`
- `omcmb/frontend-core/src/utils/dashboardCardSnapshot.ts`
- `omcmb/frontend-core/src/utils/dashboardCardSnapshot.test.ts`
- `omcmb/frontend-core/src/utils/dashboardRefresh.ts`
- `omcmb/webcode/src/pages/dashboard/index.test.tsx`

修改：

- `omcmb/frontend-core/src/hooks/api/useDashboard.ts`
- `omcmb/webcode/src/pages/dashboard/index.tsx`

不修改后端文件。

---

## 5. 测试

### 5.1 快照工具测试

覆盖：

1. Summary 成功时提取四张卡数值。
2. `UE_ACTIVE` 缺失时保存 `0`。
3. pending 且有快照时使用快照并关闭卡片 loading。
4. pending 且没有快照时卡片保持 loading。
5. 新 Summary 覆盖旧快照。
6. localStorage 中 JSON 损坏时安全回退为空。
7. API scope 不同时使用不同存储键。
8. 同一 API 下不同用户使用不同存储键。
9. API base、开发代理和页面 origin 按真实请求地址归一化。

### 5.2 页面验收

1. 刷新 Dashboard，Network 只出现一次 `/dashboard/summary`。
2. 其他 Dashboard 接口变慢时，四张卡不受影响。
3. Summary 变慢但存在快照时，四张卡立即显示旧值。
4. Summary 成功后，四张卡更新为新值。
5. Summary 没有 `UE_ACTIVE` 时，活跃 UE 显示 `0`。
6. SSE `dashboard_update` 后 Summary 仍会重新获取。

### 5.3 工程验证

```bash
cd omcmb
npm run test --workspace webcode -- dashboardCardSnapshot
npm run typecheck
npx eslint --quiet frontend-core/src/hooks/api/useDashboard.ts \
  frontend-core/src/hooks/api/useDashboardSummary.test.tsx \
  webcode/src/pages/dashboard/index.tsx \
  webcode/src/pages/dashboard/index.test.tsx \
  frontend-core/src/utils/dashboardCardSnapshot.ts \
  frontend-core/src/utils/dashboardCardSnapshot.test.ts \
  frontend-core/src/utils/dashboardRefresh.ts
```

---

## 6. 最终审查：Summary 用户作用域隔离

最终独立审查确认：卡片快照已经按“API 地址 + 用户 ID”隔离，但
`useDashboardSummary()` 仍使用公共 query key `['dashboard', 'summary']`。
常规退出会清理 QueryClient；如果页面不卸载且登录用户直接从 A 切换为 B，
30 秒 `staleTime` 内仍可能读取 A 的 Summary，并将其写入 B 的快照。

按最小范围修复：

1. 先增加真实 QueryClient 回归测试，复现 A 成功后直接切换 B 时仍复用 A 缓存。
2. `useDashboardSummary(apiScope, userScope)` 的 query key 改为
   `['dashboard', 'summary', apiScope, userScope]`。
3. `userScope` 不存在时禁用 Summary 查询。
4. Dashboard 页面把已经归一化的 API scope 和当前用户 ID 传给 Summary Hook。
5. 增加页面 A → B 状态测试，确认 B 不展示 A 的数值，也不会生成 B 的错误快照。
6. 保持 SSE 使用 `['dashboard']` 前缀失效查询；不修改 KPI、后端或 SQL。

完成后重新执行相关测试、完整前端测试、TypeScript、目标 ESLint、生产构建和
`172.17.9.239:8081` Dashboard smoke，再进行独立 code review 后提交。

---

## 7. 验收标准

- Dashboard 首页不再调用 `useDashboardData()`。
- Dashboard 首页只请求一次 `/dashboard/summary`。
- 不修改任何 KPI 或后端代码。
- 有历史成功值时，硬刷新不再出现四张卡全部空白。
- 新数据成功返回后自动替换旧值。
- 活跃 UE 缺值显示 `0`。
- 前端快照单测和 TypeScript 类型检查通过。

---

## 8. 风险与边界

### 首次访问

首次访问没有历史快照时，仍需等待 `/dashboard/summary`。这是最小方案的明确边界：本 Issue 解决“记录并展示上次数据”，不治理 Summary 后端查询性能。

### 旧数据

快照用于加载和故障期间的临时展示，页面始终保留“最后更新时间”。它不会参与任何告警处置或后台计算。

### 后续性能治理

如果首次访问的 Summary 延迟仍需解决，应单独采集后端各查询阶段耗时并处理慢 SQL，不应在本 Issue 中修改 KPI 逻辑。

---

## 9. 实施与验证结果

实施分支：

```text
fix/issue-123-dashboard-card-last-value
```

实现严格限制在第 4 节列出的前端文件内，未修改 KPI 标识、KPI 查询、后端接口或数据库逻辑。

### 9.1 自动化验证

已通过：

- Dashboard API、Summary 用户作用域、卡片快照、刷新语义和页面集成测试：5 个测试文件、38 个测试全部通过。
- 新增快照工具测试：14 个测试全部通过。
- 新增页面集成测试：6 个测试全部通过。
- 新增 Summary Hook 用户切换测试：2 个用例全部通过。
- `npm run typecheck`。
- 本次修改文件的 ESLint 检查。
- 生产构建。
- `git diff --check`。

完整前端测试共 1269 项，其中 1262 项通过；剩余 7 项失败分布在 6 个未修改模块中，包括国际化、用户菜单、密码掩码、KPI 查询页面和告警规则抽屉等。本次没有为追求全绿而修改这些无关模块。全量 ESLint 另有 1 个未修改文件的既有 `no-irregular-whitespace` 错误；本次修改文件的 ESLint 全部通过。

### 9.2 `172.17.9.239:8081` 实测

测试服务器连通性和登录接口正常。本地 Vite 已切换为：

```text
localhost:3000 -> http://172.17.9.239:8081
```

真实 Dashboard 刷新结果：

- 每次刷新只出现 1 个 `/api/v1/dashboard/summary`，原来的重复 Summary 已消除。
- 两次正常实测 Summary 分别约为 `636 ms` 和 `390 ms`。
- 其他 Dashboard 请求独立返回，没有再通过页面级 `Promise.all` 控制四张卡。

故障注入验证仅对 Summary 人为增加 5 秒延迟：

- Summary 仍只发出 1 次，约 `5.4 s` 完成。
- 同期其他接口约在 `80–211 ms` 返回。
- 这证明网络请求和页面数据依赖已经解耦；卡片快照选择、成功后覆盖、失败回退和用户隔离由新增的工具及页面集成测试验证。

最终使用 Playwright Chromium 对 `172.17.9.239:8081` 执行 Dashboard 真实后端冒烟，登录、四张统计卡、两张统计图和快速入口均通过。另对当前 `localhost:3000` 的英文界面实际抓取指标库两页响应：`K900010015` 返回 `en_name: Data Volume DL`，页面也显示 `Data Volume DL`；本次没有修改 KPI 名称、指标库或下拉框逻辑。

### 9.3 额外观察

测试服务器的 `/api/v1/events/stream` 会在返回 `200` 后很快断开，前端 EventSource 因而持续重连并输出连接错误。该现象在本次修改前已存在，与四张卡被 `Promise.all` 拖住及重复 Summary 无关，未纳入 Issue #123 的最小修复范围。

---

## 10. Summary 历史字段评估

`kpi_overview`、`kpi_deltas` 和 `recent_alarms` 是不同时期的 Dashboard 需求逐步叠加到 Summary 的结果，当前契约确实过宽。但仅删除响应 JSON 字段不会降低查询耗时，必须同时停止对应后端查询。

### 10.1 `kpi_overview`

首页现在只消费其中的 `UE_ACTIVE`，其余动态 KPI 没有页面消费者。当前后端却查询 24 小时内最多 500 个不同 `metric_path` 的最新值。

结论：

- 全量 `kpi_overview` 对当前首页属于过度取数。
- 不能在 #123 中直接删除，因为活跃 UE 卡和本次快照都依赖 `UE_ACTIVE`。
- 后续应把 `active_ue` 变为卡片摘要的一等字段，或保持兼容字段但只查询 `UE_ACTIVE`。
- 这会触及 KPI 取数契约，应在独立 Issue 中通过真实库 `EXPLAIN (ANALYZE, BUFFERS)` 验证后实施。

### 10.2 `kpi_deltas`

它实际只生成：

- `total_devices`
- `active_alarms`

它没有生成 `UE_ACTIVE`，所以前端 UE 卡读取 `kpiDeltas['UE_ACTIVE']` 永远得不到后端数据。

更重要的是，Summary 先等待五组并行查询完成，然后才串行执行历史设备数和历史告警数查询。历史告警查询使用 `raised_at` / `cleared_at`，当前基线 schema 没有这两个字段的显式索引，也没有使用超表分区时间列 `time` 限定范围；随着历史告警量增加，它存在明显的拖尾风险。

结论：

- 它是最值得优先拆出的历史包袱。
- 若产品不要求卡片环比，可删除两个卡片的趋势文案并停止计算 `kpi_deltas`。
- 若仍要保留环比，应迁到独立接口和独立 React Query，不得阻塞四张主数值卡。
- 不建议只把两条历史查询移进现有 `errgroup`；这只能减少部分串行时间，Summary 仍要等待最慢查询。

### 10.3 `recent_alarms`

Dashboard 首页已经不消费该字段，但告警统计页的“高频告警设备”仍通过再次调用 Summary 读取它。因此它也是跨页面复用形成的耦合，不能直接删除；后续应提供独立的 Top 告警设备接口，再迁移消费者。

### 10.4 对 #123 的决定

本次不把上述 API 重构并入 #123：

- 239 测试服务器上 Summary 正常实测为约 `390–636 ms`。
- #123 的重复 Summary 和页面级 `Promise.all` 已由前端最小改动消除。
- 删除或拆分字段会改变卡片趋势、告警统计页或活跃 UE 的数据契约，超出本 Issue 的最小修复范围。

建议另建一个 Dashboard Summary 瘦身 Issue，按以下顺序实施：

1. 将 `kpi_deltas` 拆为非阻塞查询，或确认无需求后删除。
2. 将 `recent_alarms` 迁移到独立接口。
3. 将全量 `kpi_overview` 收敛为明确的 `active_ue` 契约，并单独验证 KPI 口径和 SQL。

---

## 11. Code Review 修正结果

本轮审查继续保持前端最小范围，不修改 KPI、后端或 SQL。已按 TDD 完成：

1. 快照键改为按“实际 API 地址 + 当前用户 ID”隔离，防止同一浏览器切换账号或后端时复用其他作用域数据。
2. API scope 同时识别 `VITE_API_BASE_URL` 和开发代理 `VITE_API_PROXY_TARGET`，统一解析为归一化绝对 API URL。
3. Summary 成功后写入轻量 localStorage 快照；页面每次渲染读取当前作用域的单条快照，因此同一页面挂载期间即使 Query 数据被清除，也会回退到最近一次成功值。
4. 卡片首次无数据状态改用 React Query `isPending`，覆盖离线 paused 场景。
5. “最后更新时间”只来源于成功 Summary 或有效快照；从未成功时显示 `--`，并始终显示该信息以说明缓存数据的新旧。
6. 手动刷新改为精确调用 Summary `refetch({ throwOnError: true })`，失败不得提示成功。
7. 新增页面状态链测试，覆盖“旧快照 A → Summary 成功 B → Query 数据清空/进入 pending → 仍显示 B”、跨用户隔离、paused 首次加载、真实更新时间和刷新错误传播。
8. Summary 与 SSE 在页面源码中各保留一个调用点；单次 Summary 网络请求已由 239 测试服务器实测确认，避免用 React 重渲染次数误判网络请求次数。

验证命令：

```bash
cd omcmb
npm run test --workspace webcode -- \
  dashboardApi dashboardCardSnapshot dashboardRefresh dashboard/index useDashboardSummary
npm run typecheck
npx eslint --quiet frontend-core/src/hooks/api/useDashboard.ts \
  frontend-core/src/hooks/api/useDashboardSummary.test.tsx \
  webcode/src/pages/dashboard/index.tsx \
  webcode/src/pages/dashboard/index.test.tsx \
  frontend-core/src/utils/dashboardCardSnapshot.ts \
  frontend-core/src/utils/dashboardCardSnapshot.test.ts \
  frontend-core/src/utils/dashboardRefresh.ts
```
