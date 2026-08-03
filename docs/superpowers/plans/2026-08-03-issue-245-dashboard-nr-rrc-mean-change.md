# Issue #245：首页 5G RRC 连接平均数修改方案

## 1. 现状与根因

Issue #245 反馈：性能模板中“RRC连接平均数”有数据，但首页 5G 图表没有打点。

当前代码和字典形成了明确的配置断裂：

- `omcgo/data/indicator-library/GNB.xml` 定义 `C010070004`，名称为 `RRC.ConnMean`，中文名为“RRC连接平均数”，类型是 counter，统计方式为 avg；
- `omcmb/webcode/src/pages/dashboard/layoutMapping.ts` 的 NR 默认首页布局只包含 `KGNB0511`、`KGNB0510`、`KGNB0517`、`KGNB0516`、`KGNB0506`、`KGNB0505`，没有 `C010070004`；
- `omcgo/migrations/seed/000001_init_seed.sql` 的内置 NR 全网、设备组、产品、频段任务也只配置上述 KGNB 指标；
- 首页 Dashboard 查询使用 `metric_type=kpi` 的 network rollup，不能自动读出模板中直接查询的 counter；
- 因此该问题不是首先判断为原始 PM 文件缺失，而是首页指标白名单、聚合任务指标集合和查询类型没有覆盖该 counter。

## 2. 目标

让首页 NR 图表能够读取并展示 `C010070004` 的已发布小时/日/周结果，同时保持：

- 其他制式和既有 NR KPI 不变；
- avg counter 仍按 PM 既有 avg 口径处理，不把日/周聚合结果简单平均成错误口径；
- #240 的 daily/weekly current-period snapshot 规则适用于该指标；
- 权限、network 维度、technology=nr 和 published revision 过滤不放宽。

## 3. 实施步骤

### 第一步：先增加失败测试

在后端补充：

- Dashboard 查询允许 NR 首页请求 `C010070004`；
- network rollup SQL 仍使用 NR 内置任务，并保留 `metric_type=kpi` 的既有路径边界；
- 若该指标要进入首页，必须确认聚合器已生成对应 KPI/可查询结果，而不是仅把 counter 名字塞进前端；
- 对 `C010070004` 的 avg 口径增加一个聚合结果映射测试，覆盖 hourly 和 daily/weekly。

在前端补充：

- `collectMetrics(buildDefaultLayout('nr'))` 包含 `C010070004`；
- 指标元数据映射显示“RRC连接平均数”和 `number` 单位；
- 首页请求包含该指标，并在有响应点时生成曲线。

### 第二步：统一指标配置

按现有项目迁移铁律，修改同一份 baseline seed，不新增 `000002+` 迁移：

- `omcmb/webcode/src/pages/dashboard/layoutMapping.ts`：将 `C010070004` 放入 NR 首页布局的合适 panel；
- `omcmb/frontend-core/src/mock/services/dashboardService.ts`：同步 mock 布局、指标元数据和确定性数据；
- `omcgo/migrations/seed/000001_init_seed.sql`：将 `C010070004` 加入需要覆盖的内置 NR 任务指标集合；
- 如首页仍只接受 KPI 结果，补充后端指标类型映射，使该 avg counter 通过既有 KPI/聚合结果契约进入 network rollup，不能在 Dashboard repository 中绕过聚合器直查 `pm_metrics`。

### 第三步：校验数据链路

依次核对：

1. GNB 原始 PM 文件包含 `RRC.ConnMean`；
2. 指标白名单启用 `C010070004`；
3. 15 分钟原始 counter 已落库；
4. 内置 NR 任务生成 hourly/daily/weekly 结果；
5. published revision 存在；
6. `/dashboard/kpi-time-series?technology=nr&kpi_names=C010070004` 返回点。

若第 1 至 4 步失败，不能只修首页；应先修指标启用或聚合任务链路，并补对应测试。

## 4. 执行顺序

本问题在 #240 完成运行验收后处理，顺序固定为：

1. 先完成 #240 的 current-period 契约和刷新验收；
2. 再添加 `C010070004` 的配置、聚合任务覆盖和测试；
3. 重新载入 seed/指标配置，重启 app/worker；
4. 真实浏览器同时对比首页 NR 图表与“内置-产品-gNB”模板的请求参数、时间粒度和返回点数。

## 5. 验证命令

```bash
cd omcgo
go test ./internal/dashboard ./internal/pm -run 'Test.*(NR|NetworkRollup|KPITimeSeries|Metric)' -count=1
go build ./...

cd ../omcmb
npm run typecheck
npx vitest run webcode/src/pages/dashboard omcmb/frontend-core/src --run
```

容器环境可用后：

```bash
docker compose -f deployments/docker/docker-compose.yml up -d --build app worker acs web
docker compose -f deployments/docker/docker-compose.yml run --rm migrate-seed
docker compose -f deployments/docker/docker-compose.yml ps
```

## 6. 验收标准

- 首页 NR 的“RRC连接平均数”有与模板一致的点；
- 请求使用 `technology=nr`、正确指标编号和相同时间粒度；
- hourly 读取已发布小时结果，daily/weekly 遵循 #240 的进行中快照规则；
- avg 统计口径与 PM 既有定义一致；
- 没有通过放宽 technology、dimension 或权限过滤来“凑出数据”；
- 前后端定向测试、类型检查、构建和真实浏览器验证通过。
