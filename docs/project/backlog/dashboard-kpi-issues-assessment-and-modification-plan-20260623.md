# 仪表板 KPI 相关问题评估与修改方案（2026-06-23）

## 1. 目标与范围

本文档用于统一仪表板（Dashboard）KPI 相关问题的 Issue 口径，并给出按现有架构可落地的逐项修改方案。

风格对齐 [GIS Issue 评估与修改方案](gis-issues-assessment-and-modification-plan-20260617.md)：每个 Issue 给出现状、复现步骤、代码证据、根因、按分层的修改方案、验收标准、风险与回滚、工作量预估。

本轮先收录 Issue A（KPI 图表下拉中英文翻译缺失），后续遇到的仪表板 KPI 问题以新 Issue 形式追加到本文件，不再为每个小问题单独开新文档。

> 说明：选择把"下拉翻译缺失"作为独立 Issue 而不是直接合并到老的 #227，是因为现在的现象不是当年 #227 的"无数据"，而是 #227 的修复策略（前端 K 编号化）漏掉了"持久化 layout 与 default layout 跟随迁移"这一步，导致出现一个新的、面向 UI 的回归。

---

## 2. 架构对齐原则

- 前端：`webcode` 只做页面编排，`frontend-core` 持有数据契约与 hook，i18n 资源放在 `frontend-core/src/i18n/`。
- 后端：`handler -> service -> repository`，layout 的"默认值"与"持久化数据"是同源契约，二者格式必须同时演进。
- KPI 指标编号（K 编号 / KGNB / KGSM）是与性能管理模块对齐的唯一标识，**不再引入 symbolic 别名**（如 `LTE_PDCP_VOLUME_DL`）作为长期方案；旧别名只允许作为短期兼容回退。
- 下拉框 label 的最终决策链应有单一入口：`useMetricMetadata` + `resolveMetricMeta`，页面层不直接拼字符串。

---

## 3. Issue 逐项评估与修改

## Issue A（P1, Bug）

### 标题
首页 KPI 折线图下拉框未做中英文翻译，直接显示 `LTE_PDCP_VOLUME_DL` 等原始 key

### Issue 提交稿（可直接用于 GitHub Bug 模板）

#### 现状描述
首页仪表板的 KPI 折线图区，LTE/NR/GSM 三个制式下的多个 Panel（业务量、可用性、利用率、接入性）的指标选择下拉框，选项 label 显示的是后端 layout 里存的原始 key（例如 `LTE_PDCP_VOLUME_DL`、`LTE_CELL_AVAILABLE`、`LTE_PRB_UTIL_DL`、`WIRELESS_SETUP_SR`），而不是"下行数据流量 / DL Data Volume"这样的中英文名称。语言切换中英文都不生效。

#### 复现步骤
1. 启动后端 + 前端，进入仪表板首页。
2. 切到 LTE 制式（NR / GSM 同样可复现）。
3. 展开"业务量""可用性""利用率""接入性"四个 Panel 的指标下拉框。
4. 观察下拉里的 option label。

#### 期望行为
- 下拉选项应根据当前界面语言（zh-CN / en-US）显示对应的中英文名称。
- 切换语言时下拉内容随之切换。
- 同一指标在仪表板下拉、性能管理列表里显示的名称应一致。

#### 实际行为
- 下拉选项直接显示 raw key（如 `LTE_PDCP_VOLUME_DL`、`WIRELESS_SETUP_SR`）。
- 中英文切换不影响下拉文本。
- 在指标库 `/api/v1/indicators` 里搜不到这些 key（指标库存的是 `K900010015` 这种 K 编号）。

#### 影响面
- 所有进入仪表板首页的用户，所有制式。
- 本地 docker 环境和 staging 环境都可复现。
- 影响功能域：F02 Dashboard / KPI。

#### 初步定位
- 持久化 layout：表 `public.dashboard_kpi_layouts`，`tech='lte'` 行的 `layout.panels[*].metrics` 仍是 symbolic 别名。
- 后端默认 layout：[goomc/omcgo/internal/dashboard/kpi_layout_default.go](goomc/omcgo/internal/dashboard/kpi_layout_default.go) 里 `defaultLayoutJSON` 同样硬编码 symbolic 别名。
- 前端指标元数据：[goomc/omcmb/webcode/src/components/dashboard/useMetricMetadata.ts](goomc/omcmb/webcode/src/components/dashboard/useMetricMetadata.ts#L96-L120) 的 `resolveMetricMeta` 第 3 层 fallback 直接返回 raw key。
- 前端 KPI 配置：[goomc/omcmb/webcode/src/pages/dashboard/kpi-config.ts](goomc/omcmb/webcode/src/pages/dashboard/kpi-config.ts#L130-L210) 已在 commit `88866603` 全量迁到 K 编号，旧别名的 `label` 映射被一并删除。

#### 严重等级
P1（首页可见的明显 UI 退化，影响所有用户）

#### 关联
- 历史 commit：`88866603 fix(#227): 修复Dashboard KPI无数据问题 - 前端直接使用KPI代码`（2026-06-15）
- 历史 PR：#384（合并 #227 的修复）
- 当前 issue：待提（建议挂 F02 Dashboard）
- 关联文档：本文件 Issue A。
- 建议分支：`fix/dashboard-kpi-layout-migrate-to-k-codes`

### 当前评估结论
确认存在，根因明确，属于 #227 修复时遗留的"持久化数据未跟随迁移"问题。优先级 P1。

### 现场现象（已复现）
- 截图证据：LTE 制式下"业务量""可用性""利用率""接入性"四张图的下拉框 label 直接显示 `LTE_PDCP_VOLUME_DL` 等原始 key。
- 接口数据：`docker exec` 直查 `dashboard_kpi_layouts` 表，LTE 行的 `metrics` 数组确实是 symbolic 别名：

  ```text
  traffic       : ["LTE_PDCP_VOLUME_DL","LTE_PDCP_VOLUME_UL","LTE_PDCP_RATE_DL","LTE_PDCP_RATE_UL"]
  availability  : ["LTE_CELL_AVAILABLE"]
  utilization   : ["LTE_PRB_UTIL_DL","LTE_PRB_UTIL_UL"]
  accessibility : ["WIRELESS_SETUP_SR","RRC_CONN_SETUP_SR","ERAB_SETUP_SR","CSFB_SR"]
  retainability : ["ERAB_DROP_RATE"]
  mobility      : ["HO_INTRA_ENB_OUT_SR","HO_INTRA_ENB_IN_SR","HO_INTER_ENB_OUT_SR","HO_INTER_ENB_IN_SR"]
  ```

### 代码证据（现状）
- 后端 layout 服务：
  - [goomc/omcgo/internal/dashboard/handler.go](goomc/omcgo/internal/dashboard/handler.go#L275-L290) — `GET /api/v1/dashboard/kpi-layout`
  - [goomc/omcgo/internal/dashboard/kpi_layout.go](goomc/omcgo/internal/dashboard/kpi_layout.go#L123-L142) — service 层：数据库无行时 fallback 到 `defaultKPILayout(tech)`
  - [goomc/omcgo/internal/dashboard/kpi_layout_default.go](goomc/omcgo/internal/dashboard/kpi_layout_default.go#L11-L33) — `defaultLayoutJSON` 仍硬编码旧别名
  - [goomc/omcgo/migrations/seed/000001_init_seed.sql](goomc/omcgo/migrations/seed/000001_init_seed.sql#L971-L980) — 种子数据 INSERT 仍是旧别名
  - 数据库表：`public.dashboard_kpi_layouts(tech, layout jsonb, ...)`
- 前端取数与映射：
  - [goomc/omcmb/frontend-core/src/hooks/api/useDashboard.ts](goomc/omcmb/frontend-core/src/hooks/api/useDashboard.ts#L134-L145) — `useKPILayout(tech)`
  - [goomc/omcmb/frontend-core/src/services/api/dashboardApi.ts](goomc/omcmb/frontend-core/src/services/api/dashboardApi.ts#L496-L502) — `getKPILayout`
  - [goomc/omcmb/webcode/src/pages/dashboard/layoutMapping.ts](goomc/omcmb/webcode/src/pages/dashboard/layoutMapping.ts#L70-L79) — `resolveLayout(tech, remoteLayout)`：有远程优先用远程
- 前端 KPI 配置与翻译：
  - [goomc/omcmb/webcode/src/pages/dashboard/kpi-config.ts](goomc/omcmb/webcode/src/pages/dashboard/kpi-config.ts#L130-L210) — 已全量 K 编号化
  - [goomc/omcmb/webcode/src/components/dashboard/useMetricMetadata.ts](goomc/omcmb/webcode/src/components/dashboard/useMetricMetadata.ts#L96-L120) — `resolveMetricMeta` 三段 fallback
  - [goomc/omcmb/webcode/src/components/dashboard/LayoutKPIPanel.tsx](goomc/omcmb/webcode/src/components/dashboard/LayoutKPIPanel.tsx#L130-L160) — 下拉 `indicatorOptions` 生成
- i18n 资源（中英文 key 已就位）：
  - [goomc/omcmb/frontend-core/src/i18n/zh-CN/index.ts](goomc/omcmb/frontend-core/src/i18n/zh-CN/index.ts#L2684-L2708) — `dashboard.kpi.*` 完整
  - [goomc/omcmb/frontend-core/src/i18n/en-US/index.ts](goomc/omcmb/frontend-core/src/i18n/en-US/index.ts#L2673-L2695) — 对齐
- 关键历史 commit：
  - `88866603 fix(#227): 修复Dashboard KPI无数据问题 - 前端直接使用KPI代码`
    - 把 `kpi-config.ts` 里的 `key` 从 `LTE_PDCP_VOLUME_DL` 等迁到 `K900010015` 等
    - 同步删除了原 `key: 'LTE_PDCP_VOLUME_DL'` → `label: 'dashboard.kpi.totalDataVolumeDl'` 的别名→i18n 映射
    - **未触碰** `defaultLayoutJSON` 和种子数据，造成持久化层与前端契约错位

### 根因
迁移到 K 编号时，只迁了"前端配置 + 数据查询路径"，没迁"持久化 layout + 默认 layout"。导致：

1. `useKPILayout(lte)` 返回的 `panel.metrics` 是 `LTE_PDCP_VOLUME_DL` 等 symbolic 别名。
2. 前端 `resolveMetricMeta(key, meta, t)` 依次走：
   - 第 1 层：指标库 `useAllIndicators(ENB)` — 指标库存的是 `K900010015`，**miss**。
   - 第 2 层：`getKPIConfigByKey(key)` — `kpi-config.ts` 已 K 编号化，旧别名都被删了，**miss**。
   - 第 3 层：fallback `return { name: metricKey, ... }` —— **直接返回 raw key**，下拉里就显示成 `LTE_PDCP_VOLUME_DL`。
3. 这也解释了为什么 i18n 资源（`dashboard.kpi.totalDataVolumeDl` 等 key 在 zh-CN/en-US 资源里都存在）看起来"完整"但用户依然看不到中文——因为代码根本走不到调用 `t()` 的分支。

派生问题：选中这种 raw key 后，趋势接口 `useMultiKPITrendComparison(['LTE_PDCP_VOLUME_DL', ...])` 也查不到 `pm_metrics` 数据（PM 表里只有 K 编号），图区会同时出现"暂无聚合数据"。是同一根因衍生的两个症状。

### 修改方案（按分层）

总策略：以 **K 编号** 为唯一编号体系，前后端 + 数据库三层一次性对齐；保留前端兜底兼容层防止旧客户端缓存或用户本地配置仍带旧别名。

**1. 后端默认 layout（kpi_layout_default.go）**
- 把 `defaultLayoutJSON` 里 LTE / NR / GSM 三个制式的 `metrics` 数组从 symbolic 别名替换为对应 K 编号。
- 参考映射表见下文"附录 A"。

**2. 数据库种子（migrations/seed）**
- 改 [000001_init_seed.sql](goomc/omcgo/migrations/seed/000001_init_seed.sql) 的 `INSERT INTO public.dashboard_kpi_layouts`，与 `defaultLayoutJSON` 同源；如果本地已 seed 过，提供一段一次性 `UPDATE` SQL（见附录 B）。
- 长期方案：新增一条 migration（例如 `0000NN_dashboard_kpi_layouts_migrate_to_k_codes.up.sql`），用 `jsonb_set` 批量替换现有行，保证已部署环境升级后能正确显示。

**3. 前端兜底（防御）**
- 在 [kpi-config.ts](goomc/omcmb/webcode/src/pages/dashboard/kpi-config.ts) 新增一张显式的 `LEGACY_ALIAS_TO_K_CODE: Record<string, string>` 映射表（短期兼容层），不再嵌在 `indicators` 里造成混乱。
- 在 [resolveMetricMeta](goomc/omcmb/webcode/src/components/dashboard/useMetricMetadata.ts) 里第 1 层 miss 时，先查 `LEGACY_ALIAS_TO_K_CODE`：若命中则用映射后的 K 编号再查一次指标库；仍 miss 才走 fallback。
- 在 [LayoutKPIPanel.tsx](goomc/omcmb/webcode/src/components/dashboard/LayoutKPIPanel.tsx) 的 fallback 分支增加一次告警日志（`console.warn` + 日志埋点），便于及早发现仍残留旧别名的 layout。

**4. layout 写入侧 normalize（可选 P2）**
- 后端 `SaveKPILayout` 在写库前对 `metrics` 做一次 normalize，把已知旧别名转成 K 编号，避免管理员从老 UI 把旧别名再写回库。

**5. 文档**
- 在 `docs/implementation/dashboard-align-to-pm-complete-implementation-plan.md` 末尾补"持久化层迁移"小节，标注本次补救。
- 本文件 Issue A 持续记录。

### 验收标准
- LTE / NR / GSM 三个制式下，所有 Panel 的指标下拉 label 都是中文（zh-CN）或英文（en-US），不再出现 `LTE_*` / `WIRELESS_*` / `RRC_*` 之类的 raw key。
- 切换界面语言，下拉内容立刻随之切换。
- 同一指标在仪表板下拉和性能管理列表里的中英文名称一致。
- 选中下拉项后，趋势接口能返回 PM 数据（不再出现"暂无聚合数据"提示）。
- 全新环境（`docker compose down -v` + 重新 seed）和增量升级环境（已存数据 + 跑 migration）都通过。
- 单测：`resolveMetricMeta` 对 `LEGACY_ALIAS_TO_K_CODE` 命中场景有覆盖。

### 风险与回滚
- 风险 1：管理员已自定义保存过的 layout 里也可能含旧别名，单跑 SQL 替换可能与用户已存的"非标"key 冲突。
  - 缓解：migration 只针对 `defaultLayoutJSON` 已知的别名集做替换，未知 key 保留原样；前端兜底层还在。
- 风险 2：前端 `LEGACY_ALIAS_TO_K_CODE` 映射表不完整，遗漏的 key 仍走 fallback。
  - 缓解：fallback 加埋点 + 周期性巡检，按需补全。
- 回滚：单独回滚 migration（提供 `.down.sql`）+ revert 前端 `kpi-config.ts` 的兼容层，即可恢复原状。

### 工作量预估
- 后端（kpi_layout_default + migration + 种子）：0.3 人日
- 前端（兼容层 + 测试）：0.3 人日
- 联调验证：0.2 人日

### 附录 A：旧别名 → K 编号 映射

| 旧 symbolic 别名 | K 编号 | i18n key (zh / en) | 含义 |
| --- | --- | --- | --- |
| `LTE_PDCP_VOLUME_DL` | `K900010015` | `dashboard.kpi.totalDataVolumeDl` | 下行数据业务流量 / Total Data Volume DL |
| `LTE_PDCP_VOLUME_UL` | `K900010016` | `dashboard.kpi.totalDataVolumeUl` | 上行数据业务流量 / Total Data Volume UL |
| `LTE_PDCP_RATE_DL` | _暂无 K 编号_ | `dashboard.kpi.throughputDl` | 下行速率（待 KPI 模块补） |
| `LTE_PDCP_RATE_UL` | _暂无 K 编号_ | `dashboard.kpi.throughputUl` | 上行速率（待 KPI 模块补） |
| `LTE_CELL_AVAILABLE` | `K900010006` | `dashboard.kpi.cellAvailable` | 小区可用性 |
| `LTE_PRB_UTIL_DL` | `K900010014` | `dashboard.kpi.dlPrbUtilRate` | 下行 PRB 利用率 |
| `LTE_PRB_UTIL_UL` | `K900010013` | `dashboard.kpi.ulPrbUtilRate` | 上行 PRB 利用率 |
| `WIRELESS_SETUP_SR` | `K900010006` | `dashboard.kpi.wirelessSetupSr` | 无线建立成功率 |
| `RRC_CONN_SETUP_SR` | `K900010002` | `dashboard.kpi.rrcSetupSr` | RRC 建立成功率 |
| `ERAB_SETUP_SR` | `K900010005` | `dashboard.kpi.erabSetupSr` | E-RAB 建立成功率 |
| `CSFB_SR` | `K900010029` | `dashboard.kpi.csfbSr` | CSFB 成功率 |
| `ERAB_DROP_RATE` | `K900010027` | `dashboard.kpi.erabDropRate` | E-RAB 掉线率 |
| `HO_INTRA_ENB_OUT_SR` | `K900010017` | `dashboard.kpi.hoIntraEnbOutSr` | 同基站切换成功率—切出 |
| `HO_INTRA_ENB_IN_SR` | `K900010022` | `dashboard.kpi.hoIntraEnbInSr` | 同基站切换成功率—切入 |
| `HO_INTER_ENB_OUT_SR` | `K900010021` | `dashboard.kpi.hoInterEnbOutSr` | 异基站切换成功率—切出 |
| `HO_INTER_ENB_IN_SR` | `K900010026` | `dashboard.kpi.hoInterEnbInSr` | 异基站切换成功率—切入 |

> 注：`LTE_PDCP_RATE_DL` / `LTE_PDCP_RATE_UL` 当前无对应 K 编号（参见 `kpi-config.ts` 内 TODO）。处理方式两选一：
> 1. 从 layout 暂时移除（避免显示一个永远查不到数据的指标）；
> 2. 保留并在前端走兼容层显示 i18n 名，但数据列空，配 toast 提示"待 PM 模块支持"。
> 建议方案 1，等 KPI 模块补齐后再加回。

### 附录 B：一次性升级 SQL（示例，跑前请按附录 A 核对）

```sql
-- 适用于已存在 dashboard_kpi_layouts 行的增量升级环境
-- 写到独立 migration 里，不要直接 prod 手跑

UPDATE public.dashboard_kpi_layouts
SET layout = replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(
       layout::text,
       '"LTE_PDCP_VOLUME_DL"', '"K900010015"'),
       '"LTE_PDCP_VOLUME_UL"', '"K900010016"'),
       '"LTE_CELL_AVAILABLE"', '"K900010006"'),
       '"LTE_PRB_UTIL_DL"',    '"K900010014"'),
       '"LTE_PRB_UTIL_UL"',    '"K900010013"'),
       '"WIRELESS_SETUP_SR"',  '"K900010006"'),
       '"RRC_CONN_SETUP_SR"',  '"K900010002"'),
       '"ERAB_SETUP_SR"',      '"K900010005"'),
       '"CSFB_SR"',            '"K900010029"'),
       '"ERAB_DROP_RATE"',     '"K900010027"'),
       '"HO_INTRA_ENB_OUT_SR"','"K900010017"'),
       '"HO_INTRA_ENB_IN_SR"', '"K900010022"'),
       '"HO_INTER_ENB_OUT_SR"','"K900010021"'),
       '"HO_INTER_ENB_IN_SR"', '"K900010026"'),
       -- 暂无 K 编号的速率指标先用空字符串占位，前端会过滤掉
       '"LTE_PDCP_RATE_DL",',  ''),
    updated_at = now()
WHERE tech IN ('lte','nr','gsm');
```

> 上述 SQL 是临时草稿。正式 migration 应改用 `jsonb_path_query` + `jsonb_set` 精确改写 `panels[*].metrics`，避免文本替换在 `title` 等其它字段误伤。

---

## Issue B（P2, Feature）

### 标题
首页 KPI 折线图下拉框由单选改多选，支持一张图同时画多个指标的对比曲线

### Issue 提交稿（可直接用于 GitHub Feature 模板）

#### 背景与动机
当前首页 KPI 折线图（业务量 / 可用性 / 利用率 / 接入性 等 Panel）的指标下拉框是**单选**：

- 管理员在系统管理 → KPI 配置页给 panel 配置了多个指标（如 traffic panel 配了 `LTE_PDCP_VOLUME_DL` / `LTE_PDCP_VOLUME_UL` / `LTE_PDCP_RATE_DL` / `LTE_PDCP_RATE_UL` 4 条），首页用户却只能一次看一条。
- 想做"上下行对比"、"主副指标关联趋势"等典型场景，必须切两次下拉、心算对比，UX 体验差。
- 同时性能管理（PM）模块的图表已经支持多 KPI 多色对比（见 `webcode-v3/src/pages/performance/PerformanceCharts.tsx`），首页和 PM 体验割裂。

#### 期望行为
- 折线图下拉框改为**多选**（带勾选 + 计数 + "全选/清空"快捷操作），默认勾选 panel 配置的全部指标（与管理员配置语义一致）。
- 选中 N 个指标后，图表同时画 N 条主线（今日）；对比线（昨日/上周）按 Issue B-子方案的策略决定是否一起画。
- 切语言、切制式、切对比类型时多选状态稳定。
- 配套：legend 多行/滚动、按指标固定配色、单位不一致时的处理策略。

#### 不在本 Issue 范围
- 管理员配置页 KPI 选择器的搜索/分页/缓存（→ 后续 Issue）。
- 持久化"用户首页多选偏好"到后端（→ 后续 Issue，本期只做 URL/zustand 内存态）。
- 跨 panel 拖拽指标 / 自定义新增图。

#### 严重等级
P2（功能增强；无功能阻塞，但显著提升首页价值密度）。

### 当前评估结论
**强烈推荐做，且改动量小。** 通过架构盘点确认：90% 的基础设施已就位，几乎所有难活早就在数据层 / 图表层 / 管理员配置侧做完了，首页是历史遗留的"UI 单选阉割"。

| 层面 | 现状 | 是否需要改 |
| --- | --- | --- |
| 后端趋势接口 `/api/v1/dashboard/kpi-time-series` | 已支持 `kpi_names=A,B,C` 逗号分隔，批量返回 `{ "A": [...], "B": [...] }` 字典 | 🟢 0 改动 |
| 后端 `kpi_layout` 存储 | `metrics` 字段已经是 `string[]`；`SaveKPILayout` 原样透传 | 🟢 0 改动 |
| 前端 hook `useMultiKPITrendComparison(string[], compareWith, enabled)` | 已存在并已返回 `MultiTrendComparisonData = { [name]: TrendComparisonData }` | 🟢 0 改动 |
| 前端 `<LineChart>` | `series: LineSeries[]` 早就支持 N 条 series；legend `type: 'scroll'` 已开 | 🟢 0 改动 |
| 类型 `KPILayoutPanel.metrics: string[]` + `layoutMapping.collectMetrics()` | 已数组化、已去重 | 🟢 0 改动 |
| 管理员配置页 `MetricPickerModal` | 已多选 + 校验"至少 1 个" | 🟢 0 改动 |
| **首页 `LayoutKPIPanel.tsx` 下拉框 + `buildSeries()`** | 状态 `selectedMetric: string`、`buildSeries` 只吃单 key、调用 `useKPITrendComparisonV2` | 🟡 **需要改造（核心改动点）** |
| i18n | 指标名 `dashboard.kpi.*` 已有；legend 自动由指标名拼接 | 🟢 0 改动；只需要新增"全选/清空"几个文案 |

### 代码证据（现状）

- 下拉框单选状态：
  - [goomc/omcmb/webcode/src/components/dashboard/LayoutKPIPanel.tsx](goomc/omcmb/webcode/src/components/dashboard/LayoutKPIPanel.tsx#L155-L162)

    ```tsx
    const [selectedMetric, setSelectedMetric] = useState<string>(panel.metrics[0] ?? '');
    // ...
    <Select value={selectedMetric} onChange={setSelectedMetric} options={indicatorOptions} />
    ```

- 单 metric → 双 series（今日 + 昨日）：
  - [goomc/omcmb/webcode/src/components/dashboard/LayoutKPIPanel.tsx](goomc/omcmb/webcode/src/components/dashboard/LayoutKPIPanel.tsx#L118-L123)

    ```ts
    const { series } = useMemo(
      () => buildSeries(selectedMetric, trendData, xData, todayLabel, yesterdayLabel, selectedMeta.conversion),
      [selectedMetric, trendData, xData, todayLabel, yesterdayLabel, selectedMeta.conversion],
    );
    ```

- 已具备批量能力（被调用方只用了单 key 路径）：
  - [goomc/omcmb/frontend-core/src/hooks/api/useDashboard.ts](goomc/omcmb/frontend-core/src/hooks/api/useDashboard.ts#L475-L520) — `useMultiKPITrendComparison`
  - [goomc/omcmb/frontend-core/src/services/api/dashboardApi.ts](goomc/omcmb/frontend-core/src/services/api/dashboardApi.ts#L480-L496) — `getKPITimeSeries`，`kpi_names: names.join(',')`
  - [goomc/omcgo/internal/dashboard/handler.go](goomc/omcgo/internal/dashboard/handler.go#L177-L208) — `parseKPINames` 解析逗号分隔

- 图表组件早就支持多 series：
  - [goomc/omcmb/webcode/src/components/Charts/LineChart.tsx](goomc/omcmb/webcode/src/components/Charts/LineChart.tsx#L7-L47) — `series: LineSeries[]`，含 `name / data / color / dashed`

- 现成的多 KPI 对比参考实现：
  - [goomc/omcmb/webcode-v3/src/pages/performance/PerformanceCharts.tsx](goomc/omcmb/webcode-v3/src/pages/performance/PerformanceCharts.tsx#L1-L25) — `selectedKpis: string[]` + `useMultipleKPISeries` + 颜色循环

### 根因（为何是历史遗留 UI 单选）
追溯起来不是 bug 而是渐进实现：

1. 最早 panel 模型是"一图一指标"，`selectedMetric: string` 直接绑 `panel.metrics[0]`。
2. 后来管理员配置页（issue #213 S3）演进出"一图多指标"语义，并升级了类型与存储。
3. 数据 hook 重构出 `useMultiKPITrendComparison` 用于 PM 模块。
4. 首页折线图组件没跟上演进，仍保留单选 UI 与 `useKPITrendComparisonV2(selectedMetric)` 的单 key 调用路径。

### 修改方案（按分层 + 单选→多选切换点拆解）

总策略：**只动首页 `LayoutKPIPanel.tsx` 一个组件 + 重写 `buildSeries`**，其他层全部复用现有能力。先做"行为正确"，再做"UX 打磨"。

#### 0. 设计决策（先决定，再写代码）

> 这些决策影响行为，建议在 issue 评论里先对齐后再开 PR。

- **D1 默认选中策略**：默认勾选 panel 全部 metrics vs 只勾第一项？
  - **决策（已实施）**：仅勾第一项。理由：默认进入单指标 today+yesterday 对比视图（与旧版单选行为完全一致，无回归感知），用户主动叠加才进入多选；同时下拉框头部 tag 不爆炸、视觉负担小。
- **D2 多选时是否还画对比线（昨日/上周）**：
  - 候选 a：选 1 个 → 画今日+对比；选 ≥ 2 个 → **只画今日**（避免 2N 条线视觉爆炸）。
  - 候选 b：始终画今日+对比（虚线），N 大时图很乱。
  - 候选 c：单独加一个"显示对比"开关。
  - **建议 a**，最简单稳健；后续可以做 c 升级。
- **D3 单位不一致**：业务量是 KB、速率是 Mbps、可用性是 %，混选时 Y 轴怎么处理？
  - 候选 a：双 Y 轴（≤ 2 种单位时启用）；3+ 种单位时禁用混选并 toast。
  - 候选 b：始终单 Y 轴 + tooltip 显示各自单位；用户自负责。
  - 候选 c：按单位自动分组到不同子图（改动大）。
  - **建议 b**，本期实现成本最低；D3 升级单独排期。
- **D4 颜色管理**：
  - 候选 a：按 `panel.metrics` 顺序在固定调色板里循环取色（与 PM 模块一致，简单）。
  - 候选 b：每个 K 编号在 `KPI_CATALOG` 里固定一个 `color`（一致性强，但要求填表）。
  - **建议 a** 起步，把"是否固定色"留给 D4 升级。
- **D5 选择数量上限/下限**：是否强制最少/最多？
  - **决策（已实施）**：**不设硬下限**。理由：调研 Grafana / DataDog / Kibana 等同类运维监控产品的多选过滤器均允许 0 选，"清空"是用户的明确意图，强行拦截反而产生"为什么删不掉"的困惑。0 选时显示 `请至少选择一个指标` 友好占位即可。tag 渲染采用 `tagRender` 去掉逐个 ×，反选靠"下拉点 ✓"标准心智，避免选 N 个时 N 个 × 的视觉负担；保留 `allowClear` 提供一键清空入口。软上限 6 + toast 暂未实施，留待后续。
- **D6 状态持久化范围**：
  - 候选 a：纯组件内 `useState`（刷新丢）。
  - 候选 b：URL query（可分享）。
  - 候选 c：zustand 全局（跨页面保留）。
  - **建议 a** 起步；URL 持久化作单独 issue。

#### 1. 前端 — `LayoutKPIPanel.tsx`（核心改动）

- 状态从单 key 改成 key 数组：

  ```ts
  // 原
  const [selectedMetric, setSelectedMetric] = useState<string>(panel.metrics[0] ?? '');
  // 改
  const [selectedMetrics, setSelectedMetrics] = useState<string[]>(panel.metrics);
  ```

- 下拉组件改成 antd `Select mode="multiple"`（或 `mode="tags"`）：

  ```tsx
  <Select
    mode="multiple"
    maxTagCount="responsive"
    value={selectedMetrics}
    onChange={setSelectedMetrics}
    options={indicatorOptions}
    style={{ minWidth: 200 }}
    size="small"
    allowClear
    placeholder={t('dashboard.kpi.selectMetricsPlaceholder')}
  />
  ```

- 趋势取数从 `useKPITrendComparisonV2(selectedMetric)` 切到 `useMultiKPITrendComparison(selectedMetrics, compareWith, enabled)`，复用其字典返回。

- `buildSeries` 重写为多 metric 入参；按决策 D2，根据 `selectedMetrics.length` 决定是否产出对比线：

  ```ts
  function buildSeries(
    metrics: string[],
    multiData: MultiTrendComparisonData | undefined,
    xData: string[],
    todayLabel: (name: string) => string,
    yesterdayLabel: (name: string) => string,
    metaResolver: (key: string) => ResolvedMetricMeta,
    palette: string[],
  ): { series: LineSeries[] } {
    if (!multiData || metrics.length === 0) return { series: [] };
    const showCompare = metrics.length === 1; // D2
    const out: LineSeries[] = [];
    metrics.forEach((key, idx) => {
      const meta = metaResolver(key);
      const color = palette[idx % palette.length];
      const cur = multiData[key]?.currentSeries ?? [];
      out.push({ name: todayLabel(meta.name), data: cur, color });
      if (showCompare) {
        const prev = multiData[key]?.compareSeries ?? [];
        out.push({ name: yesterdayLabel(meta.name), data: prev, color, dashed: true });
      }
    });
    return { series: out };
  }
  ```

- 颜色调色板与 PM 模块对齐：
  `const PALETTE = ['#1677FF', '#52C41A', '#FA8C16', '#722ED1', '#13C2C2', '#EB2F96'];`

- 空选择 / 全部反选时图区显示"请至少选一个指标"占位（沿用现有 `Empty` 组件）。

#### 2. 前端 — i18n 新增 key（只新增，不动现存）

`frontend-core/src/i18n/{zh-CN,en-US}/index.ts` 在 `dashboard.kpi.*` 下追加：
- `selectMetricsPlaceholder`: "请选择指标" / "Select metrics"
- `noMetricSelected`: "请至少选择一个指标" / "Select at least one metric"
- `tooManyMetricsWarn`: "已选 {count} 项，建议不超过 {max} 项以保持图表可读" / "{count} selected; for chart readability we recommend at most {max}"

#### 3. 前端 — 单测（`__tests__/LayoutKPIPanel.test.tsx` 新增）

- 多选 1 个 → series 长度 = 2（today + yesterday，dashed）
- 多选 2 个 → series 长度 = 2（only today，分别用 palette[0] / palette[1]）
- 多选 0 个 → 显示占位、不发起 trend 请求
- 颜色按 idx 循环、单位回退正确（沿用 `resolveMetricMeta`）

#### 4. 后端 — 0 改动

确认 `parseKPINames` 在传入 6+ 个 metric 时正常工作（已是 `strings.Split`）。如需保险，可加一道软上限（如 max 16），但不在本 Issue 必做。

#### 5. 文档

- 用户文档：仪表板使用手册补"多指标对比"小节。
- 本文件 Issue B 持续更新决策结论。

### 验收标准
- 任一 panel 下拉默认勾选其 `panel.metrics` 全部项，图表立刻渲染 N 条曲线。
- 取消勾选某项 → 对应曲线立即从图中消失；legend 同步更新。
- 全部取消 → 图区显示"请至少选择一个指标"占位，不发起趋势请求。
- 选 1 个 → 同时画今日 + 昨日（昨日虚线）。
- 选 ≥ 2 个 → 只画今日 N 条；颜色按调色板循环、每条曲线一色。
- 切语言：legend 文案随之切换；选择状态保留。
- 切制式 (LTE → NR → GSM)：下拉重置为新 panel 的全选；图表重渲染。
- 兼容 Issue A：旧 symbolic alias 在多选下也能正确显示中文名。
- 选 6 项以上：toast 提示"建议不超过 6 项"但不阻塞。
- 单测 4/4 通过。

### 风险与回滚
- 风险 1：单位混选时单 Y 轴 + 不同量纲并存（如 % 和 Mbps），数值幅度差距大导致小值曲线被压扁不可见。
  - 缓解：tooltip 显示各自单位+原始值；在文档注明此为已知限制，D3 升级单独排期。
- 风险 2：选择数量过多时图表可读性差（≥ 8 条线）。
  - 缓解：D5 软上限 + toast；legend 已是 scroll 模式不会爆框。
- 风险 3：`useMultiKPITrendComparison` 对未在指标库的 key（如本期仍残留的 `LTE_PDCP_RATE_DL` 符号）发请求会拿到空数据。
  - 缓解：Issue A 的 catalog 兜底已覆盖；fallback 时该曲线显示空但不报错。
- 回滚：单 commit 集中在 `LayoutKPIPanel.tsx` + 新增测试，`git revert` 即可。

### 工作量预估
- 前端核心改动（LayoutKPIPanel + buildSeries + 调色板）：0.5 人日
- i18n 新增 key + 中英文翻译：0.1 人日
- 单测：0.2 人日
- 联调验证 + 截图：0.2 人日
- **合计：约 1 人日**

### 实施顺序与依赖
- **依赖 Issue A 已合入**（catalog + alias 兜底，否则多选场景下 raw key 漏出更明显）。
- 决策项 D1~D6 在 PR 提交前需在 issue 评论里收敛。
- 建议分支：`feat/dashboard-kpi-multi-select`

---

## Issue C（P3, Refactor）

### 标题
首页"网络制式" Segmented 与 KPI 配置页 Tabs 改为字典驱动，去除前端硬编码

### Issue 提交稿（可直接用于 GitHub Feature 模板）

#### 背景与动机
首页仪表板顶部的"网络制式"切换器（LTE / NR / GSM）目前是**前端硬编码**：

- 三项 label 与 value 都写死在 [dashboard/index.tsx](goomc/omcmb/webcode/src/pages/dashboard/index.tsx) 与 [dashboard/kpi-config.ts](goomc/omcmb/webcode/src/pages/dashboard/kpi-config.ts) 的 `TECH_LABELS`。
- 后端早已存在通用字典体系（`sys_dictionaries` + `sys_dictionary_details`），并且已有 `type='network_type'` 字典（id=15，含 `lte` / `nr` 两行，label 分别为 "eNB(LTE)" / "gNB(NR)"）作为"设备网络制式"的统一来源；设备列表 / 回收站等模块已通过 `useDictionaryBatch(['network_type'])` 消费。
- 当前管理员若想增删一个制式（例如临时屏蔽某项）或者改显示文案，必须改前端代码并发版；与字典管理页"运行时可改"的承诺不一致。
- 同一概念"网络制式"在系统里出现两套展示真相：设备模块按字典展示 `eNB(LTE) / gNB(NR)`，Dashboard 与 KPI 配置页按 `TECH_LABELS` 展示 `LTE / NR / GSM`。

本 Issue 让首页 Segmented 与 KPI 配置页 Tabs 改为运行时从字典读取，**直接复用字典原 label**，与设备模块的展示对齐。

#### 期望行为
- 字典管理页 `network_type` 字典里**启用 + 排序**的明细决定首页 Segmented / KPI 配置 Tabs 的**选项集合、顺序、显示文案**。
- 编辑某项的 label / 切换 `status` 启用状态 / 调 `sort` → 刷新页面即可见，无需发版。
- 首页 Segmented 与 KPI 配置 Tabs 仍只渲染前端 `TechnologyType = 'lte' | 'nr' | 'gsm'` 已知的 value，未知 value 自动过滤（前端 KPI 配置静态契约本期不动）。
- loading / error / 空字典 → 返回空数组，UI 显示空 Segmented / 空 Tabs，让运维感知字典缺失；**不再注入前端硬编码 fallback**。

#### 不在本 Issue 范围
- 取消 / 放开 TypeScript 静态契约 `TechnologyType`（涉及 PM 模块 / 后端 CHECK 约束，独立 Issue）。
- 后端 5 张表（`pm_dashboards` / `pm_tasks` / `pm_user_dashboard_preferences` / `dashboard_kpi_layouts` 等）的 `technology` 列 CHECK 约束改为软引用字典（独立 Issue）。
- 字典 seed / migration 改动（本期不动数据库；基线字典已有 `lte` / `nr`，是否补 `gsm` / 调 label 全由运维通过字典管理页决定）。
- 字典管理页本身的 UI / CRUD（已上线，无需改）。

#### 严重等级
P3（重构 / 配置外置；无功能阻塞，但消除一处硬编码、为后续多制式扩展铺路）。

### 当前评估结论
**强烈推荐做，且改动量极小。** 字典体系、消费 hook 均已就位；本期**纯前端改动，0 SQL 改动**：新建 1 个收敛 hook + 改 2 个接入点（首页 Segmented、KPI 配置 Tabs）+ 删除 `TECH_LABELS` 常量。

| 层面 | 现状 | 是否需要改 |
| --- | --- | --- |
| 后端字典接口 `GET /admin/sysDictionary/findSysDictionary?type=network_type` | 已就位 | 🟢 0 改动 |
| 字典表 `sys_dictionaries id=15 (type='network_type')` | 已存在 | 🟢 0 改动 |
| 字典明细 seed | 含 `lte` / `nr` 两行，label 为 "eNB(LTE)" / "gNB(NR)" | 🟢 0 改动（GSM 是否补由运维决定） |
| 前端 hook `useDictionary(code)` / `useDictionaryBatch(codes)` | 已就位 | 🟢 0 改动 |
| 现成消费方参考 [RecycleBin/index.tsx#L55-L56](goomc/omcmb/webcode/src/pages/device/RecycleBin/index.tsx#L55-L56) | 已上线 | 🟢 0 改动 |
| **首页 [dashboard/index.tsx](goomc/omcmb/webcode/src/pages/dashboard/index.tsx) Segmented options** | 硬编码 `TECH_LABELS.lte/.nr/.gsm` | 🔴 **改为 `useTechnologyDictionary()`** |
| **KPI 配置页 [system/KpiConfig/index.tsx](goomc/omcmb/webcode/src/pages/system/KpiConfig/index.tsx) Tabs** | 硬编码 | 🔴 同步改 |
| `dashboard/kpi-config.ts` 的 `TECH_LABELS` 常量 | 仅 hook fallback 与硬编码 Segmented 使用 | 🔴 **删除常量定义**（保留 `TechnologyType` 类型 export） |
| `dashboard/kpi-config.ts` 的 `TechnologyType` 类型 | 静态联合类型 | 🟢 不动（本期不打开契约） |
| 后端 5 张表 `CHECK (technology IN ('lte','nr','gsm'))` | 与字典脱钩 | 🟢 本期不动（决策 D3） |

### 代码证据（现状）

**1. 前端硬编码点（已改造点的旧状态，供 PR diff 参考）**

- 首页 Segmented 选项：原 [dashboard/index.tsx](goomc/omcmb/webcode/src/pages/dashboard/index.tsx) 硬编码 `[{label:TECH_LABELS.lte,value:'lte'}, ...]`。
- KPI 配置 Tabs：原 [system/KpiConfig/index.tsx](goomc/omcmb/webcode/src/pages/system/KpiConfig/index.tsx) 硬编码 `items=[{key:'lte',label:TECH_LABELS.lte}, ...]`。
- 静态 label 表 + 静态联合类型：
  - [kpi-config.ts](goomc/omcmb/webcode/src/pages/dashboard/kpi-config.ts) `TECH_LABELS = { lte: 'LTE', nr: 'NR', gsm: 'GSM' }`（本期删）
  - [kpi-config.ts](goomc/omcmb/webcode/src/pages/dashboard/kpi-config.ts) `type TechnologyType = 'lte' | 'nr' | 'gsm'`（本期保留）

**2. 现成基建（无需改造）**

- 字典 hook：
  - [useSystem.ts](goomc/omcmb/frontend-core/src/hooks/api/useSystem.ts) — `useDictionary(dictType)` + `useDictionaryBatch(codes)`
- 字典 API 端点：
  - [adminApi.ts](goomc/omcmb/frontend-core/src/services/api/adminApi.ts) — `findDictionaryByType(type)` → 返回 `{ ..., sysDictionaryDetails: [{label, value, sort, status, labelI18n, ...}] }`
- 现成消费方：
  - [device/RecycleBin/index.tsx](goomc/omcmb/webcode/src/pages/device/RecycleBin/index.tsx) — `useDictionaryBatch(['network_type'])` 调用样板

**3. 字典 baseline 现状（无需改动）**

- 字典本体 `id=15, type='network_type', name='设备网络制式'`。
- 字典明细：
  - `(45, label='eNB(LTE)', value='lte', sort=1, status=true)`
  - `(46, label='gNB(NR)',  value='nr',  sort=2, status=true)`
- 是否需要 GSM 行：由运维通过字典管理页按需新增；前端不预设、不 seed。

**4. 后端 CHECK 约束（本期不动，备忘）**

- [000001_init_schema.sql:5036](goomc/omcgo/migrations/000001_init_schema.sql#L5036) — `pm_dashboards_technology_check CHECK (technology = ANY (ARRAY['lte','nr','gsm']))`
- [000001_init_schema.sql:5171](goomc/omcgo/migrations/000001_init_schema.sql#L5171) — `chk_pm_tasks_technology`
- [000001_init_schema.sql:5194](goomc/omcgo/migrations/000001_init_schema.sql#L5194) — `pm_user_dashboard_preferences_technology_check`
- [000001_init_schema.sql:17732](goomc/omcgo/migrations/000001_init_schema.sql#L17732) — `dashboard_kpi_layouts_tech_check`

### 根因
- 历史实现把"网络制式"当作前端常量看待（仅 LTE / NR / GSM 三类），未走通用字典体系。
- 设备模块后来引入字典（commit `2d8d0dc`、设备列表性能优化），但首页 dashboard 与 KPI 配置页没跟上。
- 结果：同一概念"网络制式"在系统里有两套展示真相（字典原文案 vs 前端 `TECH_LABELS`），且 dashboard 这套不可运行时编辑。

### 设计决策（已与用户对齐）

- **D1 字典源策略**：复用现有 `network_type` 字典（id=15）；**显示文案直接复用字典 label**（基线即 "eNB(LTE)" / "gNB(NR)"），与设备模块对齐。**不引入 `extend` 短码**作为 dashboard 专属展示文案 —— "同一字典两套消费方需要两套展示文案"是伪需求，运维若要改文案直接改字典 label，两边同步更新。
- **D2（已撤销）**：~~补 GSM 字典项~~ —— 本期不 seed 任何字典行。基线 `lte` / `nr` 是否够用、是否补 GSM，由运维通过字典管理页按需决定。
- **D3 后端 CHECK 约束**：本期**不动**。短期保留 5 张表硬编码 `ARRAY['lte','nr','gsm']` 的耦合。后续若要真正"字典驱动制式扩展"再单独 Issue 处理（迁移到 reference table 或软校验）。
- **D4 加载策略**：loading / error / 字典空 → 返回空数组，UI 显示空 Segmented / 空 Tabs，让运维**感知字典缺失**。**不再注入前端硬编码 fallback** —— fallback 与"字典驱动"的精神相悖（fallback 会掩盖真实的字典缺失故障）。下游图表组件本就支持"无 technology / 无 layout"的退化渲染，不会因为空 Segmented 而崩溃。
- **D5 i18n 入口收敛**：新建 `useTechnologyDictionary()` hook 统一 label 解析逻辑：`labelI18n[locale] → label → value.toUpperCase()`（最终兜底防御 NPE）。**不再引入 `extend` 优先链**。
- **D6 排序**：按字典 `sort` 字段升序渲染 Segmented / Tabs；前端不再保留 LTE→NR→GSM 的硬编码顺序。
- **D7 范围**：除首页 Segmented 外，**KPI 配置页**（[system/KpiConfig/index.tsx](goomc/omcmb/webcode/src/pages/system/KpiConfig/index.tsx)）的 tech Tabs 也一并纳入本 Issue，避免遗留第二处硬编码。

### 修改方案（按分层）

总策略：**前端一收敛点（`useTechnologyDictionary` hook）+ 两接入点（首页 Segmented、KPI 配置 Tabs）+ 删除 `TECH_LABELS` 常量**。0 SQL 改动，0 后端改动。TypeScript 静态契约与后端 CHECK 约束本期不动，留待后续放开制式集合的 Issue 单独处理。

#### 1. 前端 — 新建收敛 hook

新建 [goomc/omcmb/webcode/src/components/dashboard/useTechnologyDictionary.ts](goomc/omcmb/webcode/src/components/dashboard/useTechnologyDictionary.ts)：

```ts
import { useMemo } from 'react';
import { useIntl } from 'react-intl';
import { useDictionary } from '@core/hooks/api/useSystem';
import type { TechnologyType } from '@/pages/dashboard/kpi-config';

const KNOWN_TECHS: ReadonlySet<TechnologyType> = new Set(['lte', 'nr', 'gsm']);

export interface TechnologyOption {
  value: TechnologyType;
  label: string;
  sort: number;
}

const pickLabel = (...candidates: Array<string | undefined | null>): string | undefined => {
  for (const c of candidates) {
    const s = (c ?? '').trim();
    if (s) return s;
  }
  return undefined;
};

export function useTechnologyDictionary(): {
  options: TechnologyOption[];
  isLoading: boolean;
} {
  const { data, isLoading } = useDictionary('network_type');
  const { locale } = useIntl();

  const options = useMemo<TechnologyOption[]>(() => {
    const details = data?.sysDictionaryDetails;
    if (!details?.length) return [];

    return details
      .filter((d) => d.status !== false)
      .filter((d) => KNOWN_TECHS.has(d.value as TechnologyType))
      .map<TechnologyOption>((d) => {
        const tech = d.value as TechnologyType;
        const label = pickLabel(d.labelI18n?.[locale], d.label) ?? tech.toUpperCase();
        return { value: tech, label, sort: d.sort ?? 0 };
      })
      .sort((a, b) => a.sort - b.sort);
  }, [data, locale]);

  return { options, isLoading };
}
```

#### 2. 前端 — 接入点 1：首页 Segmented

修改 [goomc/omcmb/webcode/src/pages/dashboard/index.tsx](goomc/omcmb/webcode/src/pages/dashboard/index.tsx)：

```tsx
import { useTechnologyDictionary } from '@/components/dashboard/useTechnologyDictionary';
// ...
const { options: techOptions } = useTechnologyDictionary();

// 越界回退：当前选中 technology 不在 techOptions 中（字典禁用了当前项）→ 切到第一项
useEffect(() => {
  if (techOptions.length && !techOptions.some((o) => o.value === technology)) {
    setTechnology(techOptions[0].value);
  }
}, [techOptions, technology]);

// ...
<Segmented value={technology} onChange={(v) => setTechnology(v as TechnologyType)} options={techOptions} />
```

#### 3. 前端 — 接入点 2：KPI 配置 Tabs

修改 [goomc/omcmb/webcode/src/pages/system/KpiConfig/index.tsx](goomc/omcmb/webcode/src/pages/system/KpiConfig/index.tsx)，同样以 `useTechnologyDictionary` 提供的 options 生成 Tabs `items`，并带同样的越界回退。

#### 4. 前端 — 删除 `TECH_LABELS` 常量

从 [kpi-config.ts](goomc/omcmb/webcode/src/pages/dashboard/kpi-config.ts) 删除 `TECH_LABELS` 定义与 export（hook 简化后已无消费者）。`TechnologyType` 类型保留 export（hook 与 KPI panel 配置仍引用）。

#### 5. 前端 — 单测（`useTechnologyDictionary.test.tsx` 新增）

7 个 case：
- loading 时返回空数组（无 fallback）
- 字典空列表 → 返回空数组
- 使用字典原 label（"eNB(LTE)" / "gNB(NR)"）
- `labelI18n[locale]` 优先于 `label`
- `labelI18n` / `label` 全为空 → 兜底到 `value.toUpperCase()`
- 未知 value（如 `'cdma'`） → 过滤
- `status=false` → 过滤；按 `sort` 升序

#### 6. 文档

- 本文件 Issue C 段持续记录决策与验收。

### 验收标准
- 字典管理页 → `network_type` 字典 → 改某项 label / 切换启用状态 / 调 sort → 刷新页面，首页 Segmented 与 KPI 配置 Tabs 立即反映变化，无需发版。
- 首页 Segmented / KPI 配置 Tabs 的显示文案与设备模块对齐（基线即 "eNB(LTE)" / "gNB(NR)"）。
- 临时禁用某项（status=false） → Segmented / Tabs 自动隐藏该项；若禁用项正是当前选中项，自动回退到第一项。
- 字典接口 timeout / 500 → 返回空数组，UI 显示空 Segmented / 空 Tabs，下游 KPI 面板按"无 layout"退化渲染；运维可见、可定位。
- 设备列表 / 回收站的"网络制式"列与筛选下拉显示不变（不受 Dashboard 改造影响 —— 二者本就消费同一字典）。
- 单测 7/7 通过；`tsc --noEmit` 0 错。

### 风险与回滚
- 风险 1：字典误删全部 `network_type` 明细 → 首页 / KPI 配置页空 Segmented / Tabs。
  - **这是预期行为**（决策 D4），让运维感知字典缺失。建议字典管理页后续加"内置字典不可删"的保护（独立 Issue）。
- 风险 2：管理员把字典 label 改成超长文案 → Segmented 单 tab 撑宽布局。
  - 缓解：`useTechnologyDictionary` 对 label 做 trim；UI 层可加 `max-width` 截断 + tooltip 显示完整文案（按需）。
- 风险 3：管理员新增 value 不在 `'lte' | 'nr' | 'gsm'` 联合类型里（如 `'wifi'`） → 被前端过滤，不展示。
  - 这是**预期行为**（本期不放开静态契约），管理员需通过独立 Issue 走"放开类型 + 后端 CHECK"流程。
- 回滚：单 commit revert 即可恢复硬编码 Segmented / Tabs；0 数据库改动 → 无数据回滚成本。

### 工作量预估
- 前端（hook + 两接入点 + 删 TECH_LABELS）：0.3 人日
- 单测：0.2 人日
- 联调验证 + 截图：0.2 人日
- **合计：约 0.7 人日**

### 实施顺序与依赖
- **独立于 Issue A / B**，可单独切 PR；建议在 A、B 之后做，避免与 Dashboard 同区域改动并发冲突。
- 决策 D1~D7 已对齐。
- 建议分支：`feat/dashboard-tech-from-dictionary`

---

## Issue C+（占位 / TBD）

后续仪表板 KPI 相关问题在此追加。每条 Issue 沿用 Issue A / B 的结构。

候选项（待用户确认是否纳入）：
- 用户级"首页多选偏好"持久化到后端（user-scoped settings）。
- 多选下的单位不一致 → 双 Y 轴 / 单位分组子图（Issue B 决策 D3 的升级版）。
- KPI 选择器（管理员配置页）放开全部指标后，搜索 / 分页 / 缓存策略。
- 趋势接口在选中"暂无 K 编号"指标时的空数据提示与降级。
- KPI Panel 在小屏 / 高密度下的图表压缩与 tooltip 互斥。
- 多制式同 K 编号场景下的"重复指标去重"语义。

---

## 4. 合并说明

- 本文档与 GIS 那份 [gis-issues-assessment-and-modification-plan-20260617.md](gis-issues-assessment-and-modification-plan-20260617.md) 是同级 backlog 文档，按功能域分文件管理。
- Dashboard 域内的多个 KPI 问题统一往本文件追加，不再逐个新建分散文档。

---

## 5. 实施顺序建议

1. Issue A（P1）：先发，影响所有用户、修复成本低、风险可控。
2. Issue B 起后续问题按 P 级和受影响范围排序，逐项 PR 切片。

---

## 6. 分支与 PR 切片建议

- PR-A（Issue A）
  - 建议分支：`fix/600-dashboard-kpi-i18n-catalog`（已合入主线为准，否则按此命名）
  - 涉及：`omcmb`（catalog + alias 兜底 + 测试） + `goomc/docs`（本文档）— 前端单栈
  - 持久化层迁移（default layout / seed / migration）拆为后续 Issue。
- PR-B（Issue B）
  - 建议分支：`feat/dashboard-kpi-multi-select`
  - 涉及：`omcmb/webcode/src/components/dashboard/LayoutKPIPanel.tsx` + `frontend-core/src/i18n/*` + 新增组件单测
  - **依赖 PR-A 合入**（catalog 兜底是多选场景下中英文显示的前置条件）
- PR-C（Issue C）
  - 建议分支：`feat/dashboard-tech-from-dictionary`
  - 涉及：`omcmb/webcode/src/components/dashboard/useTechnologyDictionary.ts`（新增）+ `omcmb/webcode/src/pages/dashboard/index.tsx`（接入）+ `omcmb/webcode/src/pages/system/KPIConfig/*`（如有 tech tab） + `omcgo/migrations/seed/000001_init_seed.sql`（GSM + extend）+ 新增 migration + 单测
  - **独立于 PR-A/B**，可并行；建议排在 A/B 后避免同区域并发冲突
- 后续 PR：按 Issue C+ 单独切

---

## 7. 提交前检查清单

- [ ] 每个 Issue 都能映射到明确 PR 范围
- [ ] 每个 PR 都有回归用例或手工验证步骤
- [ ] Dashboard 关键行为（首页 KPI 下拉、切语言、切制式、趋势取数）有对照截图
- [ ] 数据库变更随 migration 走，不留手跑 SQL
- [ ] 变更遵循现有分层，i18n 入口收敛在 `resolveMetricMeta`
