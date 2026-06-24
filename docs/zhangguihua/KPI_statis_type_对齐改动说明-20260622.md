# KPI 指标管理 — `statis_type` 对齐老 OMC 系统改动说明

> **日期**：2026-06-22
> **背景**：根据 [KPI_Mgmt_功能说明.md](./KPI_Mgmt_功能说明.md) §2/§4/§5 对老 OMC 系统的指标统计类型（`statis_type`）业务做差异修复。
> **改动范围**：后端 Go（pm/indicator + pm/kpi + pm/metrics）+ 前端 frontend-core + 三皮肤（webcode / webcode-v2 / webcode-v3）。
> **关联工作**：紧随 Issue #525 (KPI 详情态新建指标查不到) 合并之后；本批改动**未独立开 issue**，属于差异分析后立即修复的"零碎缺口"。

---

## 1. 改动概览

| 序号 | 缺口 | 修复点 | 文件数 |
|------|------|--------|--------|
| **G1** | `min` 聚合算子缺失：DB CHECK 允许 `min`，但 Go calculator switch 无 case，min 类指标聚合时报 `"unknown statis_type"` | 加 `StatisMin` 常量 + calculator min 分支 + 单测 | 4 |
| **G2** | 前端字段名为"数据类型"（实为 TR-069 counter `data_type`），与老 OMC `statis_type` 业务语义不匹配 | 把 UI 字段改为"统计类型"，下拉 `sum/avg/max/min/pct` 五选一，三皮肤同时落地 | 7 |
| **G3** | 后端 API 对 `statis_type` 无枚举校验，可入任意字符串 | `CreateIndicatorRequest` + `UpdateIndicatorRequest` + 旧 `addOrModifyIndicatorRequest` 字段加 `binding:"omitempty,oneof=sum avg max min pct"` + REST 集成单测 | 3 |

总计：**14 个文件**，新增 1 个测试文件，0 个 schema 改动。

---

## 2. 后端改动详情

### 2.1 G1 — `min` 聚合算子链路补齐

#### 2.1.1 [omcgo/internal/pm/metrics/model.go](../../omcgo/internal/pm/metrics/model.go)

```go
const (
    StatisSum StatisType = "sum"
    StatisAvg StatisType = "avg"
    StatisMax StatisType = "max"
    StatisMin StatisType = "min"   // ★ 新增
    StatisPct StatisType = "pct"
)
```

并把注释从"sum/avg/max/pct 四路"改为"sum/avg/max/min/pct 五路"，添加"与老 OMC `perf_indicators.statis_type` 五个枚举一一对应"说明。

#### 2.1.2 [omcgo/internal/pm/kpi/calculator.go](../../omcgo/internal/pm/kpi/calculator.go)

`AggregateByStatisType()` switch 加 `case metrics.StatisMin` 分支：

```go
case metrics.StatisMin:
    m := values[0]
    for _, v := range values[1:] {
        if v < m {
            m = v
        }
    }
    return m, nil
```

#### 2.1.3 [omcgo/internal/pm/kpi/calculator_test.go](../../omcgo/internal/pm/kpi/calculator_test.go)

- 新增 `Test_AggregateByStatisType_Min`：输入 `[3,7,1,9,4]` → 期望 `1.0`
- `Test_AggregateByStatisType_SingleValue` 循环数组追加 `metrics.StatisMin`，确保单值场景所有类型一致返回

#### 2.1.4 [omcgo/internal/pm/metrics/pg_repository_test.go](../../omcgo/internal/pm/metrics/pg_repository_test.go)

`Test_Constants_Stable` 新增 `assert.Equal(t, "min", string(StatisMin))`，防止常量字符串值被误改。

### 2.2 G3 — `statis_type` 字段枚举校验

#### 2.2.1 [omcgo/internal/pm/indicator/model.go](../../omcgo/internal/pm/indicator/model.go)

**`CreateIndicatorRequest`**（POST `/api/v1/indicators`）：
```go
StatisType string `json:"statis_type" binding:"omitempty,oneof=sum avg max min pct"`
```

**`UpdateIndicatorRequest`**（PUT `/api/v1/indicators/:id`）：
```go
StatisType *string `json:"statis_type" binding:"omitempty,oneof=sum avg max min pct"`
```
> 注：指针字段 `*string`，`omitempty` 对 nil 直接放行；非 nil 时才校验枚举。

#### 2.2.2 [omcgo/internal/pm/indicator/handler.go](../../omcgo/internal/pm/indicator/handler.go)

兼容旧入口 `/pm/indicatormg` 的 `addOrModifyIndicatorRequest.StatisType` 同样加 binding。

#### 2.2.3 [omcgo/internal/pm/indicator/rest_handler_statistype_test.go](../../omcgo/internal/pm/indicator/rest_handler_statistype_test.go)（新增文件）

两个测试函数共 **14 条用例**：

| 测试 | 合法（6） | 非法（4-5） |
|------|----------|------------|
| `TestRESTCreateIndicator_StatisType_Enum` | sum / avg / max / min / pct / 字段省略 | median / SUM（大写）/ xxx |
| `TestRESTUpdateIndicator_StatisType_Enum` | pct / min / 字段省略 | median / 求和（中文）|

**测试策略**：用空 `IndicatorManagementService{}`（依赖未注入）+ `gin.Recovery()`，合法 bind 通过后 service 因 nil 触发 panic 被 Recovery 捕获返 500；非法 bind 直接 400。用 HTTP code（400 vs 非 400）+ 响应体含 `"StatisType"` + `"oneof"` 关键字判定。

---

## 3. 前端改动详情（G2 — 三皮肤）

### 3.1 frontend-core 共享业务层

#### 3.1.1 [omcmb/frontend-core/src/types/indicatorLibrary.ts](../../omcmb/frontend-core/src/types/indicatorLibrary.ts)

```ts
export interface IndicatorInfo {
  // ...
  // 统计类型（对齐老 OMC perf_indicators.statis_type）。取值枚举见 STATIS_TYPE_VALUES；
  // KPI 新建/编辑表单用下拉渲染。后端 BackendIndicator.statis_type 下发，mapIndicator 透传。
  statisType?: string;
  // ...
}

// 统计类型枚举（与后端 perf_indicators.statis_type / pm_metrics.statis_type CHECK 约束对齐）。
// UI 下拉选择项取该数组；各皮肤渲染时可用 i18n key `product.kpi.indicator.statisType.<value>` 本地化。
export const STATIS_TYPE_VALUES = ['sum', 'avg', 'max', 'min', 'pct'] as const;
export type StatisTypeValue = (typeof STATIS_TYPE_VALUES)[number];
```

`CreateIndicatorInput.statisType?: string` 之前已有，本次只补全 `IndicatorInfo` 上的读取字段。

#### 3.1.2 [omcmb/frontend-core/src/services/api/indicatorLibraryApi.ts](../../omcmb/frontend-core/src/services/api/indicatorLibraryApi.ts)

- `BackendIndicator` 加 `statis_type?: string`
- `mapIndicator()` 加 `statisType: b.statis_type` 透传

#### 3.1.3 i18n（[zh-CN](../../omcmb/frontend-core/src/i18n/zh-CN/index.ts) + [en-US](../../omcmb/frontend-core/src/i18n/en-US/index.ts)）

新增 7 个 key：
| key | zh-CN | en-US |
|-----|-------|-------|
| `product.kpi.indicator.statisTypeLabel` | 统计类型 | Statistic Type |
| `product.kpi.indicator.statisTypeRequired` | 请选择统计类型 | Please select statistic type |
| `product.kpi.indicator.statisType.sum` | 求和 (sum) | Sum |
| `product.kpi.indicator.statisType.avg` | 平均 (avg) | Average |
| `product.kpi.indicator.statisType.max` | 最大值 (max) | Max |
| `product.kpi.indicator.statisType.min` | 最小值 (min) | Min |
| `product.kpi.indicator.statisType.pct` | 百分比 (pct) | Percentage |

### 3.2 三皮肤 UI 替换

| 皮肤 | 文件 | UI 组件 |
|------|------|---------|
| **v1** Antd5 | [omcmb/webcode/src/pages/product/kpi-library/IndicatorFormModal.tsx](../../omcmb/webcode/src/pages/product/kpi-library/IndicatorFormModal.tsx) | `<Select>` + `STATIS_TYPE_VALUES.map(...)` 选项 |
| **v2** shadcn | [omcmb/webcode-v2/src/pages/product/KpiIndicatorsDetail.tsx](../../omcmb/webcode-v2/src/pages/product/KpiIndicatorsDetail.tsx) | shadcn `<Select>` / `<SelectContent>` / `<SelectItem>` |
| **v3** STARFORGE HUD | [omcmb/webcode-v3/src/pages/kpi-library/IndicatorFormModal.tsx](../../omcmb/webcode-v3/src/pages/kpi-library/IndicatorFormModal.tsx) | 原生 `<select class="neon-input">` + `<option>` |

三处统一：
- 旧字段 `dataType` / `dataType state` 全部改为 `statisType`
- 表单初始预填用 `indicator.statisType`（不再用 `indicator.counterType`）
- 提交 payload 改为 `statisType: statisType || undefined`
- 头部注释更新到"统计类型"业务语义

> ⚠️ 注：后端 `data_type`（TR-069counter 类型 int/real/float）仍由 XML 字典维护，**未从后端模型移除**，只是从表单 UI 中移除—— 因为用户不需要在前端编辑该底层 TR-069 类型。

---

## 4. 验证步骤（开发自验）

### 4.1 后端

```bash
cd /Users/a1/Desktop/vscode/goomc/omcgo

# 编译
go build ./...

# 单元测试（含新加的 14 条枚举用例）
go test -count=1 ./internal/pm/indicator/... ./internal/pm/kpi/... ./internal/pm/metrics/...

# 只跑 statis_type 枚举测试
go test -count=1 -v -run "StatisType_Enum" ./internal/pm/indicator/...
```

**预期**：4 个包全 `ok`；枚举测试 14 条全 PASS。

### 4.2 前端

```bash
cd /Users/a1/Desktop/vscode/goomc/omcmb

# 三皮肤 typecheck + skin-parity 守卫
npm run typecheck
```

**预期**：
```
✓ skin-parity 通过：v2/v3 路由与可见菜单均与 v1 一致。  （129 路由 / 37 菜单）
webcode@0.0.0 typecheck → 0 errors
webcode-v2@0.0.0 typecheck → 0 errors
webcode-v3@0.0.0 typecheck → 0 errors
```

---

## 5. 回归测试 Checklist（QA / 集成）

### 5.1 后端 API（用 curl / Postman 或 e2e 脚本）

| # | 请求 | 期望响应 |
|---|------|----------|
| 1 | `POST /api/v1/indicators?deviceType=enb`<br>body：`{"en_name":"e","cn_name":"c","group_id":"g","statis_type":"sum"}` | 200/201（业务路径走通）|
| 2 | 同上但 `"statis_type":"avg"` / `"max"` / `"min"` / `"pct"` | 同上 |
| 3 | 同上但不带 `statis_type` 字段 | 200/201（`omitempty` 放行） |
| 4 | 同上但 `"statis_type":"median"` | **400** + 错误信息含 `StatisType` + `oneof` |
| 5 | 同上但 `"statis_type":"SUM"`（大写）| **400**（枚举区分大小写）|
| 6 | `PUT /api/v1/indicators/:id?deviceType=enb`<br>body：`{"statis_type":"min"}` | 200（仅枚举校验放行；业务逻辑视 id 存在与否）|
| 7 | 同上但 `"statis_type":"xxx"` | **400** |

### 5.2 前端三皮肤（手工或 Playwright）

每个皮肤分别覆盖：

| # | 路径 | 操作 | 期望 |
|---|------|------|------|
| 1 | `产品中心 → KPI 指标库 → 选 ENB/GSM/GNB → 选 platform → 新建指标` | 弹窗有"统计类型"下拉，显示 5 项：`求和 (sum)` / `平均 (avg)` / `最大值 (max)` / `最小值 (min)` / `百分比 (pct)` |
| 2 | 选其中一项 → 填必填字段 → 保存 | 后端落库 `statis_type` = 选中值；详情页刷新后正确回显 |
| 3 | 不选统计类型 → 保存 | 允许保存（字段可选）|
| 4 | 编辑已有指标（带 `statis_type`）| 下拉预填正确值 |
| 5 | 详情抽屉 / 列表显示 | 如果有显示该字段，文案为"统计类型"，不再是"数据类型" |

| 皮肤 | 路由 | 备注 |
|------|------|------|
| v1 | `http://localhost:8081/product/kpi-library` | Antd5，下拉为 antd `<Select>` |
| v2 | `http://localhost:8081/v2/product/kpi-library` | shadcn，下拉为 shadcn `<Select>` |
| v3 | `http://localhost:8081/v3/product/kpi-library` | STARFORGE HUD，下拉为霓虹风格原生 select |

### 5.3 全链路一致性（设计回归）

```
DB CHECK ─────────────► CHECK (statis_type IN ('sum','avg','max','min','pct'))
Go 常量 ───────────────► StatisSum/StatisAvg/StatisMax/StatisMin/StatisPct          ★ G1 修复
Go 聚合算子 ───────────► AggregateByStatisType switch 5 个 case                     ★ G1 修复
后端 API binding ──────► oneof=sum avg max min pct（Create + Update + addOrModify） ★ G3 修复
前端 TS 枚举 ──────────► STATIS_TYPE_VALUES = ['sum','avg','max','min','pct']      ★ G2 修复
前端 UI 控件 ──────────► Select 下拉 5 项（v1/v2/v3 三皮肤）                       ★ G2 修复
i18n ─────────────────► 7 keys（zh-CN + en-US 同步）                              ★ G2 修复
```

任一层与其他层不一致即为回归 bug，应立即拦截。

---

## 6. 风险与回滚

### 6.1 兼容性风险

| 风险 | 评估 | 缓解 |
|------|------|------|
| **存量数据**：旧 indicator 行 `statis_type` 为非枚举值（如老系统迁过来的脏数据）| 中 | binding 是**请求侧**校验，不影响读路径；存量行可读可显示，仅修改时被拦 |
| **旧前端兼容**：未更新到新 i18n 的客户端访问 | 低 | UI 字段从 input 改为 select，旧 input 提交的字符串若不在枚举内会被后端 400 拦下，行为符合预期 |
| **业务代码 hardcode**：[aggregator.go:431](../../omcgo/internal/pm/aggregator/aggregator.go#L431) 仍写死 `statis = string(metrics.StatisPct)` | 低 | 这是 KPI 行的缺省值，本次不动；若需要支持 KPI 行按用户配置写其他值，需另外 issue |

### 6.2 回滚

| 范围 | 回滚动作 |
|------|----------|
| 后端 binding | 移除 3 处 `binding:"omitempty,oneof=..."` 标签 |
| 后端 min 算子 | 删除 calculator.go 的 `case metrics.StatisMin` 分支 + model.go 的 `StatisMin` 常量 |
| 前端 | 三皮肤 dataType 字段恢复（git revert 即可）；frontend-core 的 `statisType` 字段保留不影响 |

无 schema 变更，无数据迁移，回滚简单。

---

## 6.5 后续追加 — Unit / Level 字段下拉化（2026-06-23）

### 6.5.1 背景

用户截图反馈老 OMC 系统「单位 Unit」字段是固定枚举下拉（截图显示 `%` / `bit` / `Byte` / `Byte/s` / `char` / `dBm` / `Erl` / `Gbit` / `GByte` / `Kb/PRB` / `Kbit` 等），「等级 Level」对外仅 `Device` 和 `PLMN` 两项。我方当前实现两个字段均为自由文本 `<Input>`，易输错且与老系统业务约束不一致。

### 6.5.2 改动文件清单

| 层 | 文件 | 内容 |
|----|------|------|
| **frontend-core** | [omcmb/frontend-core/src/types/indicatorLibrary.ts](../../omcmb/frontend-core/src/types/indicatorLibrary.ts) | 新增 `INDICATOR_UNIT_OPTIONS`（25 项 readonly tuple）+ `INDICATOR_LEVEL_OPTIONS`（Device/PLMN 2 项）+ `IndicatorUnitValue` / `IndicatorLevelValue` 字面量类型 |
| **i18n** | [omcmb/frontend-core/src/i18n/zh-CN/index.ts](../../omcmb/frontend-core/src/i18n/zh-CN/index.ts) / [en-US](../../omcmb/frontend-core/src/i18n/en-US/index.ts) | 新增 `unitRequired` / `levelRequired` 2 个 key（"请选择单位/级别" / "Please select unit/level"）|
| **v1 Antd** | [omcmb/webcode/src/pages/product/kpi-library/IndicatorFormModal.tsx](../../omcmb/webcode/src/pages/product/kpi-library/IndicatorFormModal.tsx) | Unit `<Input>` → `<Select allowClear showSearch>`；Level `<Input>` → `<Select allowClear>` |
| **v2 shadcn** | [omcmb/webcode-v2/src/pages/product/KpiIndicatorsDetail.tsx](../../omcmb/webcode-v2/src/pages/product/KpiIndicatorsDetail.tsx) | Unit/Level `<Input>` → shadcn `<Select>` / `<SelectContent>` / `<SelectItem>` |
| **v3 HUD** | [omcmb/webcode-v3/src/pages/kpi-library/IndicatorFormModal.tsx](../../omcmb/webcode-v3/src/pages/kpi-library/IndicatorFormModal.tsx) | Unit/Level `<input>` → 原生 `<select class="neon-input">` + `<option>` |

### 6.5.3 Unit 选项集（25 项，A→Z）

`%` · `bit` · `Byte` · `Byte/s` · `char` · `dBm` · `Erl` · `GByte` · `Gbit` · `Kb/PRB` · `KByte` · `KByte/s` · `Kbit` · `Kbps` · `MByte` · `Mbps` · `milliseconds` · `ms` · `no` · `number` · `ppm` · `s` · `seconds` · `time` · `W`

**来源**：截图所示 11 项 + `omcgo/data/indicator-library/**/*.xml` 实际 `unitId` 全集（去重，避免存量数据回显为空）。

### 6.5.4 Level 选项集（2 项）

| value | label |
|-------|-------|
| `device` | Device |
| `plmn` | PLMN |

> ⚠️ **存量数据兼容**：`omcgo/data/indicator-library/**/*.xml` 中存在 `indicatorLevel="both"` 数据。下拉**不暴露 both 选项**，但通过 antd `<Select allowClear>` 与 shadcn / 原生 select 对受控外值的容忍机制，**编辑已有 'both' 指标时仍能正常回显**该值；用户重新选择后会落入新枚举（device/plmn）。

### 6.5.5 验证

```bash
cd /Users/a1/Desktop/vscode/goomc/omcmb && npm run typecheck
# ✓ skin-parity 通过：v2/v3 路由与可见菜单均与 v1 一致。（129 路由 / 37 可见）
# ✓ webcode/v2/v3 三皮肤 typecheck 全过
```

### 6.5.6 回归 Checklist 增量

在第 5.2 章三皮肤手工/Playwright 测试基础上增加：

| # | 操作 | 期望 |
|---|------|------|
| 1 | 新建指标 → 点开 Unit 下拉 | 显示 25 个选项，按 A→Z 排序；v1 支持键盘搜索过滤 |
| 2 | 选 `%` 保存 → 详情回显 | 列表/详情正确显示 `%` |
| 3 | 新建指标（非 GNB） → 点开 Level 下拉 | 显示且仅显示 Device / PLMN 两项 |
| 4 | 编辑老数据（`indicatorLevel="both"` 的内置指标）| Level 字段正确回显 "both" 文本（受控外值），下拉本身仍只显 2 项；用户改选 Device → 保存成功，落库为 device |
| 5 | GNB 模式下编辑指标 | Level 字段隐藏（与 statis_type 改造前保持一致）|

### 6.5.7 部署

升级到本机测试环境（docker compose 重建 web 镜像即可，无后端代码变更）：

```bash
cd /Users/a1/Desktop/vscode/goomc
docker compose -f deployments/docker/docker-compose.yml -p omc up -d --build web
```

访问：
- v1：`http://localhost:8081/product/kpi-library`
- v2：`http://localhost:8081/v2/product/kpi-library`
- v3：`http://localhost:8081/v3/product/kpi-library`

---

## 6.6 后续追加 — 「计数器」字段语义重塑为「指标类型」+ 公式一站式配置（2026-06-23）

### 6.6.1 背景

原表单上有一个名为「计数器」的 Switch 开关，绑后端 `isCounter` 字段（取值 `'0'|'1'`）。但用户反馈：
1. **「计数器」这个名字太晦涩** — 不是电信背景的人很难理解（误以为是某种"启用开关"或"计算方式开关"，实际是**指标的数据来源类型**）
2. **新建指标后还要进抽屉/编辑页再配公式** — 流程不连贯

代码层面查实：`isCounter` 是电信网管行业的**数据来源分类**（参考 [KPI_Mgmt_功能说明.md](./KPI_Mgmt_功能说明.md)）：

| 取值 | 业务含义 | 数据来源 | 是否需要公式 |
|------|----------|----------|--------------|
| `'1'`（Counter）| 原始计数器 | 设备 PM 文件直接上报 | 否 |
| `'0'`（KPI）   | 派生指标   | OMC 按公式从其他指标算出 | **是**（必须有 arithmetic）|

且后端 [service.go#L208-213](../../omcgo/internal/pm/indicator/service.go#L208) 已经按公式**自动校验+覆盖** `isCounter`（公式只引用单个指标 ID 且无运算符 → 自动改回 Counter）— 双保险逻辑已存在。

### 6.6.2 改动一句话

- **「计数器」Switch → 「指标类型」Radio**（直接采集 / 公式计算）
- 选「公式计算」后**下方实时联动出现公式输入框**，新建表单一站式配好公式，避免「先建空壳→再去抽屉配公式」的割裂

### 6.6.3 变更文件清单（5 个）

| 文件 | 变更要点 |
|------|----------|
| `omcmb/frontend-core/src/types/indicatorLibrary.ts` | 新增 `INDICATOR_TYPE_OPTIONS`（`counter`/`kpi`）+ `IndicatorTypeValue` 类型，与 `isCounter='0'\|'1'` 一一映射 |
| `omcmb/frontend-core/src/i18n/{zh-CN,en-US}/index.ts` | 新增 6 个 i18n key：`typeLabel` / `typeCounter` / `typeCounterHint` / `typeKpi` / `typeKpiHint` / `arithmeticLabel` / `arithmeticPh` / `arithmeticHelp` / `arithmeticRequired` / `formulaBracketMismatch`（旧 `isCounterLabel` 保留以维持向后兼容，UI 不再使用）|
| `omcmb/webcode/src/pages/product/kpi-library/IndicatorFormModal.tsx` (v1) | Antd `Radio.Group` + `Tooltip` 悬停说明；Antd `shouldUpdate` 联动 `Form.Item` 渲染公式区 |
| `omcmb/webcode-v2/src/pages/product/KpiIndicatorsDetail.tsx` (v2) | 原生 `<input type="radio">` + Tailwind 卡片式选项（含说明副文）；条件渲染 textarea |
| `omcmb/webcode-v3/src/pages/kpi-library/IndicatorFormModal.tsx` (v3) | HUD `button[role=radio]` 双卡片网格；条件渲染 neon-input textarea |

### 6.6.4 UI 文案（zh-CN）

| 文案 key | 内容 |
|----------|------|
| `typeLabel` | 指标类型 |
| `typeCounter` | 直接采集 |
| `typeCounterHint` | 设备直接上报的原始计数器，无需配置公式 |
| `typeKpi` | 公式计算 |
| `typeKpiHint` | 由其他指标按公式实时算出（派生 KPI）|
| `arithmeticLabel` | 计算公式 |
| `arithmeticPh` | 例：`(C000060011+C000060022)/1000` |
| `arithmeticHelp` | 使用指标 ID 与运算符 `+` `-` `*` `/` `(` `)`；`Duration` 代表统计周期秒数 |
| `arithmeticRequired` | 公式计算类型必须填写公式 |

### 6.6.5 字段映射

```
[UI]                      [Payload]                  [DB]
indicatorType='counter' → isCounter:'1' arithmetic:undefined → perf_indicators_*.is_counter='1' arithmetic=NULL
indicatorType='kpi'     → isCounter:'0' arithmetic:'公式'    → perf_indicators_*.is_counter='0' arithmetic='公式'
```

后端 `addOrModifyIndicator` 已支持 `arithmetic` 字段（[handler.go](../../omcgo/internal/pm/indicator/handler.go#L122) + [service.go](../../omcgo/internal/pm/indicator/service.go#L201)），无需后端改动。

**双保险**：用户若把不含运算符的单指标 ID 当公式提交，后端会按 [service.go#L211](../../omcgo/internal/pm/indicator/service.go#L211) 自动改回 `isCounter='1'`（这是原有逻辑）。

### 6.6.6 前端校验

- `indicatorType` Radio 默认 `kpi`（与老 OMC 自定义指标几乎都是派生 KPI 的事实对齐）
- `kpi` 类型下 `arithmetic` **前端必填校验**（三皮肤一致，提示 `arithmeticRequired`）
- `counter` 类型不显示公式输入；切换到 counter 时已填写的公式**不会自动清空**（仅 submit 时丢弃 → 用户切换回 kpi 仍能看到原内容，体验更连贯）

### 6.6.7 验证

```bash
cd omcmb && npm run typecheck   # 三皮肤 ✓ + skin-parity 129 路由 / 37 菜单 ✓
# 仅前端改动，无需重建 app/acs/worker，只重建 web 容器：
cd /Users/a1/Desktop/vscode/goomc
docker compose -f deployments/docker/docker-compose.yml -p omc up -d --build --no-deps --force-recreate web
```

访问：
- v1：`http://localhost:8081/product/kpi-library`
- v2：`http://localhost:8081/v2/product/kpi-library`
- v3：`http://localhost:8081/v3/product/kpi-library`

**关键验证用例**：
1. 新建表单：「指标类型」默认选中「公式计算」，下方公式输入框可见
2. 切到「直接采集」→ 公式输入框消失；切回「公式计算」→ 公式输入框重现且保留之前输入的内容
3. 选「公式计算」且公式留空 → 点保存提示「公式计算类型必须填写公式」
4. 选「公式计算」+ 填合法公式（如 `C000060011/C000060022`）→ 保存成功；编辑指标重新打开 → 公式预填回填
5. 编辑已有指标：若原 `isCounter='1'`（XML 内置 Counter）→ Radio 应选中「直接采集」、公式输入框隐藏
6. 三皮肤行为一致（v1 Antd Radio + Tooltip / v2 卡片式 radio / v3 HUD 双卡片网格）

---

## 6.7 后续追加 — 新建/修改公式维护方式统一为 PlatformFormulasSection（2026-06-23）

### 6.7.1 背景

§6.6 让新建表单选「公式计算」时下方实时联动出现 arithmetic textarea，但用户复查后指出问题：

> 新增页面和修改页面的公式维护方式不一样？按照修改页面的公式维护功能做。

**事实层面**：

| 维度 | 新建表单（§6.6 改动后）| 修改页面（v1 IndicatorDrawer 现状） |
|---|---|---|
| UI | 单条 arithmetic textarea | 「全平台公式」CRUD 表格 |
| 数据存储 | `perf_indicators_*.arithmetic`（一个指标一条公式）| `rela_platform_indicator_formula_*`（按 platform 多条公式）|
| 操作粒度 | 仅可填/改单条 | 可按 platform 增/删/改多条 |

这两个本就是后端两套不同的存储 / 接口，前端不应让用户在不同地方看到不同的概念。

### 6.7.2 决策：统一为「全平台公式 CRUD」（PlatformFormulasSection）

- **删除**§6.6 给新建表单加的 arithmetic textarea + 联动校验
- **保留**「指标类型」Radio（语义不变）
- **新增**：新建态下保存基础信息成功后，**弹窗不关闭、转入「编辑态」**，PlatformFormulasSection 自动出现（条件 = `indicatorType==='kpi' && currentIndicator != null`），用户在同一弹窗里配多平台公式

技术原因：`useFormulas` / `useUpsertFormula` / `useDeleteFormula` 都需要 indicatorId，新建态没 indicatorId 无法直接调用；最简单做法是先建主指标拿到 indicatorId 再让公式 CRUD 区出现。

### 6.7.3 改动文件清单（6 个）

| 文件 | 变更要点 |
|------|----------|
| `omcmb/webcode/src/pages/product/kpi-library/PlatformFormulasSection.tsx` | **新增**共享组件 — 从 IndicatorDrawer 拆出来的「全平台公式」CRUD 表格（Antd Table + Modal）|
| `omcmb/webcode/src/pages/product/kpi-library/IndicatorDrawer.tsx` | 删除内嵌的 PlatformFormula CRUD（Table/Modal/Form/useFormulas/useUpsertFormula/useDeleteFormula 整段），改引用 `<PlatformFormulasSection />` |
| `omcmb/webcode/src/pages/product/kpi-library/IndicatorFormModal.tsx` (v1) | 删 arithmetic textarea；加 `currentIndicator` 本地态；create 成功后 `setCurrentIndicator(returned)` 弹窗不关；新增 `<PlatformFormulasSection />` 条件渲染；Footer 改自定义按钮（cancel/close 文案动态切换）|
| `omcmb/webcode-v2/src/pages/product/KpiIndicatorsDetail.tsx` (v2) | 删 arithmetic textarea + state；加 `currentIndicator` 本地态；create/update onSuccess 注入 `setCurrentIndicator(data)`；FormulaSection 条件改为 `indicatorType==='kpi' && currentIndicator`；底部按钮文案动态切 |
| `omcmb/webcode-v3/src/pages/kpi-library/IndicatorFormModal.tsx` (v3) | 同 v2，PlatformFormulaSection 条件移至 `indicatorType==='kpi' && currentIndicator` 区，NeonButton 文案动态切 |
| `omcmb/frontend-core/src/types/indicatorLibrary.ts` | 无变更（INDICATOR_TYPE_OPTIONS / IndicatorTypeValue 已在 §6.6 加入）|

### 6.7.4 关键行为

| 场景 | 行为 |
|------|------|
| 新建 → 选「公式计算」| 弹窗下方此时**没有**公式区，因为还没 indicatorId |
| 新建 → 填好基础信息点「保存」| 后端 createIndicator 返回 IndicatorInfo → 注入 `currentIndicator` → 弹窗**不关闭**、转入「编辑态」→ PlatformFormulasSection 自动出现 → 标题变「编辑指标」/ 取消按钮变「关闭」 |
| 新建 → 配多条平台公式 | 在 PlatformFormulasSection 里点「新建公式」→ 填 platform + formula → 保存（调 useUpsertFormula）|
| 编辑已有指标（修改页面）| 行为不变 — 弹窗一打开就是编辑态，PlatformFormulasSection 与新建保存后完全一样 |
| 选「直接采集」| 不显示公式区（counter 类型不需要公式）|
| 关闭弹窗 | 直接调 onClose（基础信息已保存 → 不丢失；公式 CRUD 实时落库 → 也不丢失）|

### 6.7.5 验证

```bash
cd omcmb && npm run typecheck   # 三皮肤 ✓ + skin-parity 129/37 ✓
cd /Users/a1/Desktop/vscode/goomc
docker compose -f deployments/docker/docker-compose.yml -p omc up -d --build --no-deps --force-recreate web
```

**关键验证用例**：
1. 新建表单：选「公式计算」时**不会**立刻看到公式输入框（与 §6.6 不同 — 现在要先保存基础信息）
2. 填好基础信息保存 → 弹窗不关 → 标题变成「编辑指标」+ 「全平台公式」CRUD 表格出现 + 「新建公式」按钮可点
3. 在 CRUD 表格里 add/edit/delete platform 公式 → 与修改页面（IndicatorDrawer 详情抽屉里）完全一致
4. 关闭弹窗 → 再次打开「编辑」该指标 → 公式列表完整保留
5. 「直接采集」类型 → 公式区始终不显示（即便已保存基础信息也不显示）
6. v1/v2/v3 三皮肤行为一致；v1 IndicatorDrawer 详情抽屉的「全平台公式」也是同一个 PlatformFormulasSection 组件

### 6.7.6 取舍说明

| 备选方案 | 为什么没选 |
|---------|-----------|
| 新建时本地暂存多条 PlatformFormula，保存基础信息后批量调 useUpsertFormula | 需要改造 PlatformFormulasSection 接受「本地态/连后端」两种模式 — 复杂度高；与现有 hook/组件的契约不符 |
| 新建保存后跳转到 IndicatorDrawer 抽屉里配公式 | 流程被切成两个弹窗，更不连贯（用户原始诉求是「直接配好公式」）|
| **当前方案**：新建保存后弹窗不关、转编辑态、PlatformFormulasSection 自动出现 | UX 上一气呵成、与编辑页面 UI 100% 一致、不需要引入新概念 |

---

## 6.8 后续追加 — 平台字段下拉化 + ALL 通用平台落地（2026-06-23）

### 6.8.1 背景

§6.7 把新建/修改页面的公式维护统一到 `PlatformFormulasSection` 后，用户在试用时连提两点：

1. **「平台」是 free text 输入** — 容易拼错（如 `BLQ` 误写 `blq`/`BIQ`）；用户拼错后 KPI 引擎会静默查不到公式，KPI 数据为空，难排查。
2. **「所有平台共用的公式」无法表达** — 老 OMC XML 字典里早就有 `enb/ALL.xml`（68 条用 `platform="ALL"` 声明的跨平台共用公式），但当前后端 `ListByPlatform` 是精确 SQL 匹配（`WHERE platform_name = $1`），**只有产品 `IndicatorPlatform` 字段也写 `'ALL'` 才能命中**，对绑了具体平台（BLQ/BLX/...）的产品完全不可见。等于 XML 字典里的 ALL 仅作"具名平台"而非"通配"。

### 6.8.2 改动一句话

- **平台输入** `<Input>` → **`<Select>` 下拉**，选项 = `[ALL（所有平台）, ...后端 `/api/v1/indicators/platforms` 返回的 distinct 平台名]`，三皮肤同步。
- **后端** `ListByPlatform` 改为按 `(具体平台, ALL)` **并查 + 应用层 first-wins 去重**：同 `indicator_id` 两处都有时**具体平台覆盖 ALL**，ALL 仅作兜底。
- **新建页面与修改页面同步生效**：复用同一组件，一处改动两处生效（同 §6.7）。

### 6.8.3 变更文件清单（7 个）

| 层 | 文件 | 内容 |
|----|------|------|
| **后端模型** | `omcgo/internal/pm/indicator/model.go` | 新增常量 `PlatformAll = "ALL"`，与 XML 字典 `enb/ALL.xml` 的 `platform="ALL"` 对齐 |
| **后端仓储** | `omcgo/internal/pm/indicator/pg_platform_formula_repository.go` | `ListByPlatform` 改造：`WHERE platform_name IN ($1, 'ALL')` + `ORDER BY (是ALL?1:0), platform_name, indicator_id` + Go 端 first-wins 去重（具体平台优先于 ALL）。传入本就是 ALL 时只查 ALL，避免冗余 |
| **i18n** | `omcmb/frontend-core/src/i18n/zh-CN/index.ts` | 新增 `product.kpi.platformAllSuffix='（所有平台）'`；改写 `platformExtra` 告知"ALL = 所有平台共用，具体平台优先" |
| **i18n** | `omcmb/frontend-core/src/i18n/en-US/index.ts` | 同上英文版（` (all platforms)` / `Pick a product platform. ALL = applies to every platform; specific takes precedence, ALL is fallback`）|
| **v1 共享组件** | `omcmb/webcode/src/pages/product/kpi-library/PlatformFormulasSection.tsx` | 加 `usePlatformList(deviceType)` hook 调用；platform 字段 antd `<Input>` → `<Select showSearch optionFilterProp="label">`，options 用 `useMemo` 把 ALL 常驻顶部 + 后端 distinct 列拼接 |
| **v2 shadcn** | `omcmb/webcode-v2/src/pages/product/KpiIndicatorsDetail.tsx` | `FormulaSection` 内同样接入 `usePlatformList`；platform shadcn `<Input>` → 原生 `<select>` + Tailwind 边框配 shadcn 视觉（v2 未引 shadcn Select 组件） |
| **v3 HUD** | `omcmb/webcode-v3/src/pages/kpi-library/IndicatorFormModal.tsx` | `PlatformFormulaSection` 内同样接入；platform 原生 `<input>` → 原生 `<select class="neon-input">` |

### 6.8.4 后端查询语义对照

```sql
-- 改造前
SELECT id, platform_name, indicator_id, formula, ...
  FROM rela_platform_indicator_formula_enb
 WHERE platform_name = 'BLQ'        -- 精确匹配，ALL.xml 数据不可见

-- 改造后
SELECT id, platform_name, indicator_id, formula, ...
  FROM rela_platform_indicator_formula_enb
 WHERE platform_name IN ('BLQ', 'ALL')
 ORDER BY (CASE WHEN platform_name='ALL' THEN 1 ELSE 0 END),  -- 具体平台先
          platform_name,
          indicator_id
-- 再用 Go 端 map[indicator_id]struct{} 做 first-wins 去重 →
-- 同 indicator_id 在 BLQ 与 ALL 都有公式时，BLQ 覆盖 ALL（ALL 仅作兜底）
```

**为什么不用 SQL 的 `DISTINCT ON (indicator_id) ORDER BY ...`**：Squirrel 不直接支持 PG 方言 `DISTINCT ON`，且 Go 端 first-wins 实现 7 行可读、单测易写、跨数据库可移植；查询规模可控（公式表按平台维度，单平台一般几十到几百行），不值得为此换 SQL 形态。

### 6.8.5 前端下拉选项装配

```ts
// 三皮肤一致逻辑（v1 用 useMemo，v2/v3 用 IIFE）
const fromApi = platformsData?.items ?? [];          // 后端 distinct 列
const ordered = ['ALL', ...fromApi.filter(p => p !== 'ALL')];  // ALL 永远在第一位
const seen = new Set<string>();
const platformOptions = ordered.filter(p => {
  if (seen.has(p)) return false;
  seen.add(p);
  return true;
});
```

**关键细节**：
1. **ALL 始终是第一项**，即便后端 distinct 列表里还没人选过 ALL（公式表里没 ALL 行），选项依然可选 — 因为这是 UI 约定，不依赖运行时数据。
2. **老数据兼容**：若公式行的 `platform_name` 不在下拉集（如运维历史手写过一个未在 XML 字典里出现的平台名），三皮肤都在 select 里加了一个额外 `<option value={platform}>{platform}</option>` 让回显值不丢；编辑现有行时该字段 `disabled`（因 platform 是主键不可改）。
3. **v1 用 antd Select** 支持 `showSearch + optionFilterProp="label"` 键盘筛选；v2/v3 用原生 `<select>` 因 v2 未引 shadcn Select 组件、v3 HUD 风格的下拉与 `.neon-input` 类直接套上即可。

### 6.8.6 覆盖规则示例（业务上最关键的语义说明）

| 公式表已有 | 产品 IndicatorPlatform | KPI 引擎查得 |
|------------|----------------------|-------------|
| `('ALL', K001, formula_X)` | 任意（BLQ/BLX/MLN/...）| `formula_X` |
| `('BLQ', K001, formula_X)` | BLQ | `formula_X` |
| `('BLQ', K001, formula_X)` | BLX | **无**（精确平台优先于无） |
| `('ALL', K001, formula_global)` + `('BLQ', K001, formula_blq)` | BLQ | **`formula_blq`**（具体覆盖 ALL）|
| `('ALL', K001, formula_global)` + `('BLQ', K001, formula_blq)` | BLX | `formula_global`（BLX 没具体的，用 ALL 兜底）|

### 6.8.7 兼容性

| 风险点 | 评估 | 说明 |
|--------|------|------|
| 存量 XML 字典 (`enb/ALL.xml` 68 条) | **零迁移** | 文件已经在用 `platform="ALL"`，新逻辑完全契合 |
| 存量公式行 `platform_name='ALL'` | **正向获益** | 改造前这些行仅对绑 `IndicatorPlatform='ALL'` 的产品可见；改造后**所有产品可见**且具体平台优先。如果以前有运维"为规避缺陷把同一公式手动复制到多个具体平台"的痕迹，新逻辑下这些复制行依然按具体平台返回，行为不变 |
| API 契约 | **不变** | `GET /indicators/platforms` 字段不变、写入 upsert 字段不变、表结构不变 |
| router/KPIRoute 装配 | **不变** | `pm/kpi/router` 调 `formulas.ListByPlatform` 拿到的列表已是去重后的最终公式集，下游 `uniqueIndicatorIDs` + `assembleRoute` 不需要改 |
| `router_test.go` fake | **不变** | `fakeFormulas.ListByPlatform` 是 stub 直接返回固定列表，不走真实 SQL 路径 |
| 数据库迁移 | **零** | 无表结构变更 |

### 6.8.8 验证

```bash
cd /Users/a1/Desktop/vscode/goomc/omcgo
go build ./... && go test ./internal/pm/indicator/... ./internal/pm/kpi/...
# ✓ ok pm/indicator + pm/kpi + pm/kpi/expr + pm/kpi/router

cd /Users/a1/Desktop/vscode/goomc/omcmb && npm run typecheck
# ✓ skin-parity 129/37 + v1/v2/v3 三皮肤 typecheck 全过

cd /Users/a1/Desktop/vscode/goomc
docker compose -p omc -f deployments/docker/docker-compose.yml up -d --no-deps --build app web
# 三皮肤冒烟 200 + /healthz 200 + bundle 含 platformAllSuffix 新文案
```

### 6.8.9 验收 Checklist

| # | 操作 | 期望 |
|---|------|------|
| 1 | 进编辑/新建指标弹窗 → 点「新增公式」 | 「平台」字段是下拉，首项 `ALL（所有平台）`，其下是该 deviceType 公式表里 distinct 出的所有平台名 |
| 2 | 选 ALL + 填公式 + 保存 | DB 落 `platform_name='ALL'`；后续任意产品的 KPIRoute 装配都能拿到这条公式 |
| 3 | 已对某 indicator 配过 ALL 公式 → 同 indicator 再加一条具体平台公式 | KPI 引擎对绑该具体平台的产品**返回具体平台的公式**（不返回 ALL）|
| 4 | 编辑现有公式 | 平台字段 `disabled`，下拉值正确预填；formula 可改 |
| 5 | 老数据有 `platform_name='customplatform'`（XML 字典外）| 平台字段在下拉里展示该值（额外 option 回显）且 disabled，行为不丢 |
| 6 | v1/v2/v3 三皮肤同等行为 | 均显示下拉、均带 ALL 顶部选项；v1 支持键盘 search |

### 6.8.10 取舍说明

| 备选方案 | 为什么没选 |
|---------|-----------|
| Free text 输入 + 旁加 typeahead 提示 | 没解决"用户拼错 → 静默命不中"的核心问题；只是把鸡肋的"我提示你了"留作免责 |
| 加独立"平台管理"页面 + 平台表 | 改造面大（新表/新 API/新页面/新 ADR）；本次的痛点（输错+无通用）不需要这么重的方案 |
| 只前端加 ALL 选项，后端不动 | 用户选 ALL 保存后 KPI 引擎仍查不到（仅匹配产品 IndicatorPlatform='ALL' 的产品）→ 选了等于没选，比 free text 更误导 |
| **当前方案**：前端下拉（ALL + 后端 distinct）+ 后端 ALL fallback + 具体平台优先去重 | 字典已用的约定 `ALL` 即"通配"，全栈打通；UX 直观；零迁移；与 XML 字典对齐 |

---

## 7. 后续遗留（未在本次范围内）

引用 [KPI_Mgmt_功能说明.md](./KPI_Mgmt_功能说明.md) 差异分析中识别但本次**未处理**的事项：


| ID | 缺口 | 优先级 | 备注 |
|----|------|--------|------|
| **G1（差异分析）** | reStatisticsKPI 后台重算链路缺失（`calculating_status` 无消费者）| 🔴 P0 | 需新建 worker 模块 |
| **H1（差异分析）** | 前端 KPI 管理主页面三皮肤全部缺失 | 🔴 P0 | 后端 API 完整，纯前端工作量 |
| **E1（差异分析）** | DeleteIndicator 缺"其他 KPI 公式引用"检查 | 🔴 P0 | 中等改动 |
| **G7（差异分析）** | aggregator.go:431 KPI 行强制写 pct | 🟡 P2 | 设计澄清后决定 |
| **C1（差异分析）** | 自定义指标数 ≤9999 上限校验缺失 | 🟡 P2 | 简单 |
| **G2（差异分析）** | eGW/WCG 网元支持 | 🟢 P3 | 看业务需求 |

> 上述 6 项需各自拆 GitHub Issue 走 `/ship` 流程，不在本批次内。

---

## 8. 提交建议

如果需要打包成 1 个 commit：

```
fix(pm,frontend-core,frontend): KPI statis_type 对齐老 OMC 业务语义（min 算子 + 枚举校验 + 三皮肤 UI）

依据 docs/zhangguihua/KPI_Mgmt_功能说明.md 差异分析，修复 3 处 statis_type 相关
缺口：

1. min 聚合算子链路（G1）：
   - internal/pm/metrics/model.go 加 StatisMin 常量
   - internal/pm/kpi/calculator.go switch 加 min 分支
   - calculator_test.go / pg_repository_test.go 补单测

2. statis_type 字段从前端"数据类型"改为"统计类型"下拉（G2，三皮肤）：
   - frontend-core types/api/i18n 加 STATIS_TYPE_VALUES + IndicatorInfo.statisType
   - webcode/v2/v3 三皮肤 IndicatorFormModal/Detail 字段替换为下拉
   - 旧字段 dataType（TR-069 counter 类型）从 UI 移除，仍由后端 XML 维护

3. 后端 statis_type 枚举校验（G3）：
   - CreateIndicatorRequest/UpdateIndicatorRequest/addOrModifyIndicatorRequest
     加 binding:"omitempty,oneof=sum avg max min pct"
   - 新增 rest_handler_statistype_test.go 14 条 REST 集成用例

验证：
- go build ./... ✓
- go test -count=1 ./internal/pm/{indicator,kpi,metrics}/... ✓
- npm run typecheck（skin-parity + v1/v2/v3）✓

详见 docs/zhangguihua/KPI_statis_type_对齐改动说明-20260622.md
```

或拆 3 commit（按 G1/G2/G3 各一）也可。

---

*本文档随代码改动同步成稿，作为 QA 回归与后续维护的事实源。如发现实际行为与本文档不符，以代码为准并更新本文档。*
