# Issue #145 仪表板无可比数据展示 N/A 实施计划

> **供代理执行者：** 必须使用 `superpowers:test-driven-development` 按任务逐项实施；先定义 API 的“可比较”语义，再修改页面，不得仅根据百分比数值猜测是否有基线。

**目标：** 仪表板 KPI 在缺少上一周期有效数据时显示 `N/A`，不显示误导性的 0%、100%、趋势箭头或“较昨日/上周”文案。

**架构：** 后端在 KPI delta 中显式返回是否可比较；前端按照该状态渲染百分比或 N/A。在线率属于当前快照指标，保持现有展示，不混入周期比较逻辑。

**技术栈：** Go、React、TypeScript、Vitest、i18n。

---

## 一、问题拆分与原因结论

### 测试服务器数据链路证据

在 `http://172.17.9.239:8081/dashboard` 使用当前测试数据核验：

- 设备总数为 11，卡片显示 `0.0% / 较上周`；
- 活跃告警为 1，卡片显示上涨 `100.0% / 较昨日`；
- Active UE 为 0，对比区域为空；
- 同一页面多个 KPI 图表明确显示“暂无聚合数据”。

这证明卡片展示的 0%/100% 并非由有效历史聚合数据支撑，而是后端缺失基线时的兜底值经前端无条件展示形成。

### 1. 无历史基线被编码成 0% 或 100%

**结论：已确认。**

- `omcgo/internal/dashboard/service.go` 的 `calculateKPIDeltas` 在历史查询失败或设备历史值小于等于 0 时，构造 previous=0、change=0、trend=stable。
- `computeKPIDelta` 在 previous=0、current>0 时返回上涨 100%。
- 因此“没有数据”“查询失败”“上一周期确实为 0”被压缩成相同数值，截图中的 0.0% 和 100.0%不能证明存在有效比较。

**根因：** API 数据结构只有百分比，没有“基线是否存在/是否可比较”的状态。

### 2. Active UE 没有形成完整比较数据

**结论：已确认。**

- 前端读取 `kpiDeltas['UE_ACTIVE']`。
- 当前 `calculateKPIDeltas` 只构造 `total_devices` 和 `active_alarms`，没有稳定生成 UE_ACTIVE 的周期比较。
- `KPICard` 在无 delta 时留下空白区域，正好对应截图中的空白。

**根因：** 前后端对 UE 卡片的比较契约不完整。

补充数据链路结论：规范化 PM 表的 `metric_path` 存指标编号，Active UE 对应 `KGNB0568`，`UE_ACTIVE` 只是展示名/上报名。比较查询必须使用 `KGNB0568`，API 再映射为稳定别名 `UE_ACTIVE`。当前期与上周同期必须同时走全网 hourly 聚合和相同平均算法；NaN/Inf 点位必须过滤，否则会污染百分比并导致 JSON 编码失败。

### 3. 前端无法渲染“不适用”状态

**结论：已确认。**

- `KPIDelta.changePercent` 是必有数字，没有 `hasComparison`。
- 仪表板给总设备和活动告警默认传入比较标签。
- `KPICard` 只会渲染趋势/百分比或不渲染，没有 N/A 分支。

**根因：** UI 组件和国际化资源都缺少明确的不可比较状态。

### 4. 活跃告警历史基线漏数和重复计数风险

**结论：已确认并修复。**

- 未清除告警仍在主库 `alarms_active`，清除后才归档到时序库 `alarms_history`；只查 history 会漏掉长期未清除告警。
- 归档和主库删除跨库执行，故障重试期间同一 `alarm_id` 可能同时存在两表，history 也可能重复。
- 修复后先读取并去重比较时刻仍 active 的 ID，再由 history `COUNT(DISTINCT alarm_id)` 且排除 active ID，最后求和，避免漏数和双计数。

### 5. 比较窗口发生 UTC 偏移

**结论：已确认并修复。**

旧实现的 `Time.Truncate(24h)` 按绝对时间截断，在 Asia/Shanghai 会落到 08:00，并非自然日零点；“昨日/上周”也不是同一时刻。修复后设备取上周同一时刻、告警取昨日同一时刻，UE 取今天截至当前时刻与上周同日等长窗口，全部保留本地时区。

## 二、修复方案

### API 契约

为后端和前端的 `KPIDelta` 增加：

```text
has_comparison / hasComparison: boolean
```

规则：

- 历史查询失败：`false`；同时记录日志，不向用户伪造 0%。
- 上一周期无样本：`false`。
- 上一周期聚合值为 0：默认 `false`，避免除零及误导性 100%；如果未来产品要表达“从 0 增长”，应另行定义文案。
- 当前和历史均有有效基线：`true`，正常计算百分比和趋势。

前端 `hasComparison=false` 或缺少 delta 时显示 `N/A`，隐藏箭头和比较周期标签。

## 三、实施任务

### 任务 1：后端增加可比较状态

**文件：**

- 修改：`omcgo/internal/dashboard/service.go`
- 修改/新增：`omcgo/internal/dashboard/*_test.go` 中对应 KPI delta 测试。

- [x] 先为历史查询错误、无样本、零基线添加失败测试，期望 `HasComparison=false`。
- [x] 为有效上涨添加测试，期望 `HasComparison=true` 且百分比正确；趋势阈值沿用既有逻辑。
- [x] 给 `KPIDelta` 增加 `HasComparison bool \`json:"has_comparison"\``。
- [x] 让 `calculateKPIDeltas` 保留查询错误和数据缺失语义，不再构造伪比较值。
- [x] 对错误路径保留上下文日志，避免 N/A 掩盖后端故障。

定向测试：

```bash
cd omcgo && go test ./internal/dashboard -run 'Test.*KPIDelta|Test.*KPIDeltas' -count=1 -v
```

### 任务 2：补齐 Active UE 比较契约

**文件：**

- 修改：`omcgo/internal/dashboard/service.go`
- 修改/新增：`omcgo/internal/dashboard/*_test.go`

- [x] 复用现有网络 KPI 时序查询能力取得 UE_ACTIVE 当前/上一周期数据，不新建重复查询链路。
- [x] 无任一周期样本或上一周期为 0 时返回 `has_comparison=false`。
- [x] 两个周期均有效时返回 UE_ACTIVE delta。
- [x] 通过公共 delta 与序列聚合测试覆盖无数据、零基线和有效基线。

### 任务 3：前端映射并渲染 N/A

**文件：**

- 修改：`omcmb/frontend-core/src/types/dashboard.ts`
- 修改：`omcmb/frontend-core/src/services/api/dashboardApi.ts`
- 修改：`omcmb/frontend-core/src/services/api/__tests__/dashboardApi.test.ts`
- 修改：`omcmb/webcode/src/components/KPICard/index.tsx`
- 修改：`omcmb/webcode/src/components/KPICard/KPICard.test.tsx`
- 修改：`omcmb/webcode/src/pages/dashboard/index.tsx`
- 修改：`omcmb/webcode/src/pages/dashboard/index.test.tsx`
- 修改：`omcmb/frontend-core/src/i18n/zh-CN/index.ts`
- 修改：`omcmb/frontend-core/src/i18n/en-US/index.ts`

- [x] API 映射测试先断言 `has_comparison` 转为 `hasComparison`。
- [x] KPICard 增加显式不可比较属性，测试 N/A、无箭头、无比较标签。
- [x] 总设备、活动告警、Active UE 都按同一规则传参。
- [x] 增加中英文国际化键，显示值统一为 `N/A`。
- [x] 保持在线率卡片现有“当前在线率”语义，不错误显示 N/A。

### 任务 4：完整验证

- [ ] `cd omcgo && go build ./... && go test ./internal/dashboard -count=1`
- [ ] `cd omcmb && npm run typecheck`
- [ ] `cd omcmb && npm test --workspace webcode -- --run src/pages/dashboard src/components/KPICard ../frontend-core/src/services/api/__tests__/dashboardApi.test.ts`
- [ ] 空历史库验收：总设备、活动告警、Active UE 显示 N/A。
- [ ] 构造有效上一周期数据验收：显示百分比、正确箭头和正确比较周期。
- [ ] 模拟历史查询失败，确认页面 N/A 且后端有错误日志。

## 四、不接受的表面修复

- 前端用 `changePercent === 0` 判断 N/A：真实持平也是 0%。
- 前端用 `previousValue === 0` 自行推断：API 仍无法区分无样本和真实零值。
- 只填补 Active UE 的空白：总设备和告警仍会显示伪百分比。
- 吞掉历史查询错误：页面可以降级为 N/A，但服务端必须保留可诊断日志。
