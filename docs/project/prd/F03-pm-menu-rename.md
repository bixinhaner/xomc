# PRD — 性能管理菜单命名重构 + 顺序调整

**PRD ID**：F03-pm-menu-rename
**功能域**：F03 PM（菜单层 UX）
**作者**：Claude（基于用户 TODO.MD 阶段 1 拍板）
**创建日期**：2026-05-26
**最后更新**：2026-05-26
**状态**：Approved
**关联 Sprint**：sprint-13
**关联 Risk**：— （无新增风险，纯 UX rename）
**Backlog**：T-0173
**配对发布**：T-0174（指标查询页从头重做）— 打包发，避免菜单先改但页面仍 mock

---

## 1. 业务背景（Why）

T-0164 PM/KPI 流水线 G1-G8 落地后，性能管理菜单挂了多个新页面（PM 仪表盘 G6 / 自定义聚合 G7），但侧栏菜单文案仍沿用早期工程命名（"性能查看"/"性能查询"/"基站KPI"/"标准KPI"），存在三类问题：

1. **术语模糊**：菜单名是工程视角（"查看 vs 查询" / "基站 vs 标准"），用户视角看不出每个页面的功能边界
2. **行业惯例缺失**："性能查看"应是行业通用的"性能仪表盘"；"标准KPI"易误读为"运营商规范"而非"指标元数据库"
3. **顺序倒置**：当前消费类（查看/查询）与配置类（指标库/测量任务）混排，用户找不到"看数据"的入口

T-0164 G6 浏览器闭环后，仪表盘真实可用；但菜单不挂出，用户感知不到；TODO.MD 阶段 1 同步收集了 5 项命名 + 顺序拍板。

---

## 2. 用户故事（Who / What）

> As a **网管运维**，I want **侧栏直接看到"性能仪表盘 / 指标查询"两个一级入口**，So that **不用问也知道从哪儿看实时数据**。

> As a **运维管理员**，I want **配置类菜单（指标库 / 测量任务管理）排在消费类之后**，So that **日常浏览不被"配置维护"项干扰**。

> As a **国际化用户**，I want **菜单英文文案同步更新（Performance Dashboard / Metric Query / ...）**，So that **英文环境一致可读**。

---

## 3. 验收标准（Given/When/Then）

```
AC-1 菜单文案改名
Given: 用户登录 /dashboard 后展开侧栏"性能管理"
When:  查看子菜单文案
Then:  按顺序显示 5 项中文 — "性能仪表盘" / "指标查询" / "自定义聚合" / "指标库" / "测量任务管理"
       英文环境对应 — "Performance Dashboard" / "Metric Query" / "Custom Aggregation" / "Indicator Library" / "Measurement Task Management"
```

```
AC-2 路由保持不变
Given: 现有书签 /performance/query / /performance/kpi-standard / /performance/kpi-station / /performance/pm-dashboard / /performance/pm-adhoc
When:  用户访问任一 URL
Then:  对应页面正常加载（路由 path 不动；只改菜单文案与可见性）
```

```
AC-3 新挂菜单可见
Given: T-0164 G6 / G7 路由已存在但未挂菜单
When:  本次重构上线
Then:  /performance/pm-dashboard（性能仪表盘）+ /performance/pm-adhoc（自定义聚合）两个二级菜单出现在侧栏，用户可点击进入
```

```
AC-4 顺序符合消费→配置
Given: 5 项菜单
When:  视觉自上而下扫描
Then:  消费类（仪表盘 / 指标查询 / 自定义聚合）在前；配置类（指标库 / 测量任务管理）在后
```

```
AC-5 多皮肤候选不破
Given: webcode-v2 / webcode-v3 候选皮肤可能也消费 frontend-core/i18n
When:  zh-CN / en-US 语料 keys 调整
Then:  i18n key 命名复用现有（如 nav.performance.query → 仍叫 query 不改 key，只改 value）；新增 key 限于 dashboard / adhoc 两项；webcode-v2/v3 typecheck 不破
```

---

## 4. 运营商差异（CMCC / CTCC / CUCC）

无差异。菜单文案三家通用，不涉及 Carrier 适配。

---

## 5. 非目标（Out of Scope）

- ❌ **不改路由 URL**：保留 `/performance/query` / `/performance/kpi-standard` / `/performance/kpi-station` 等，避免破坏书签 / 外部链接
- ❌ **不改页面内 UI / 功能**：纯菜单文案 + 顺序调整，页面组件不动
- ❌ **不改后端 API**：纯前端 i18n + navConfig
- ❌ **不引入新菜单层级**：保持二级菜单结构，不做嵌套折叠
- ❌ **不做权限重映射**：现有 PrivateRoute / RBAC 配置完全保留
- ❌ **不对齐 v2/v3 候选皮肤的视觉样式**：候选皮肤独立演进；本次只确保 frontend-core/i18n 改动不破其编译

---

## 6. 依赖

- ✅ T-0164 已 done（性能仪表盘 G6 + 自定义聚合 G7 路由已挂）— 否则新挂菜单点进去会 404
- 无后端依赖
- 无三方包升级

---

## 7. 度量

- **编译通过**：webcode npm run typecheck + lint 全过；webcode-v2 / webcode-v3 typecheck 不退化
- **路由可达**：5 个性能管理子菜单点击均能正确路由（手测）
- **i18n 覆盖**：zh-CN / en-US 双语 key 全到位，无 raw key 漏到 UI

---

## 8. 设计备忘（S2 出口）

### 8.1 改动清单（最小化）

| 文件 | 改动 |
|------|------|
| `omcmb/webcode/src/components/Layout/Sidebar/navConfig.ts` | performance 节点 children：① 新增 `perf-dashboard` 项（path `/performance/pm-dashboard`）顺序最上；② 调整现有 `perf-query` / `perf-kpi-bs` / `perf-kpi-std` 顺序；③ 新增 `perf-adhoc`（path `/performance/pm-adhoc`）插入中段 |
| `omcmb/frontend-core/src/i18n/zh-CN/index.ts` | ① 改 `nav.performance.query`: '性能查询' → '指标查询'；② 改 `nav.performance.kpiStation`: '测量维护' → '测量任务管理'；③ 改 `nav.performance.kpiStandard`: '指标管理' → '指标库'；④ 新增 `nav.performance.dashboard`: '性能仪表盘'；⑤ 新增 `nav.performance.adhoc`: '自定义聚合' |
| `omcmb/frontend-core/src/i18n/en-US/index.ts` | 镜像 zh-CN：'Metric Query' / 'Measurement Task Management' / 'Indicator Library' / 'Performance Dashboard' / 'Custom Aggregation' |

### 8.2 新菜单顺序

```
性能管理 (nav.performance)
├── 性能仪表盘     ← 新挂 perf-dashboard → /performance/pm-dashboard
├── 指标查询       ← 改名 perf-query (key 不动) → /performance/query
├── 自定义聚合     ← 新挂 perf-adhoc → /performance/pm-adhoc
├── 指标库         ← 改名 perf-kpi-std → /performance/kpi-standard
└── 测量任务管理   ← 改名 perf-kpi-bs → /performance/kpi-station
```

### 8.3 i18n key 命名策略

- **保留旧 key**：`query` / `kpiStation` / `kpiStandard` — 只改 value，不改 key。理由：避免 webcode-v2/v3 等候选皮肤的潜在引用断裂；改 value 是 i18n 标准 RTM 行为
- **新增 key**：`dashboard` / `adhoc` — 全新挂菜单，新加无冲突
- **复用现有英文 key**：见 8.1 表

### 8.4 待定点（< 3）

无。本任务纯文案 + 顺序变更，命名已与用户 TODO.MD 拍板对齐。

---

## 9. 风险

| ID | 描述 | 影响 | 缓解 |
|----|------|------|------|
| R-T0173-1 | webcode-v2/v3 候选皮肤可能引用了未来要改的 i18n value（如 hardcoded "性能查询" 字符串而非 i18n key）| 候选皮肤显示错乱 | grep zh-CN value 在 v2/v3 是否出现；如有则同步改；本任务范围内不调 v2/v3 UI 文件 |
