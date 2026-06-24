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

## Issue B（占位 / TBD）

后续仪表板 KPI 相关问题在此追加。每条 Issue 沿用 Issue A 的结构（标题 / Issue 提交稿 / 当前评估结论 / 代码证据 / 根因 / 修改方案 / 验收标准 / 风险与回滚 / 工作量预估）。

候选项（待用户确认是否纳入）：
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
  - 建议分支：`fix/dashboard-kpi-layout-migrate-to-k-codes`
  - 涉及：`omcgo`（default layout + migration + seed） + `omcmb`（兼容层 + 测试） — 跨栈
- PR-B 起：按 Issue 单独切

---

## 7. 提交前检查清单

- [ ] 每个 Issue 都能映射到明确 PR 范围
- [ ] 每个 PR 都有回归用例或手工验证步骤
- [ ] Dashboard 关键行为（首页 KPI 下拉、切语言、切制式、趋势取数）有对照截图
- [ ] 数据库变更随 migration 走，不留手跑 SQL
- [ ] 变更遵循现有分层，i18n 入口收敛在 `resolveMetricMeta`
