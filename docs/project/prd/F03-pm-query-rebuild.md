# PRD — 指标查询页从头重做（PM Query Rebuild）

**PRD ID**：F03-pm-query-rebuild
**功能域**：F03 PM（核心）+ frontend
**作者**：Claude（基于用户 TODO.MD 阶段 1 拍板）
**创建日期**：2026-05-26
**最后更新**：2026-05-26
**状态**：Approved
**关联 Sprint**：sprint-13
**关联 Risk**：— （无新增 P0/P1）
**Backlog**：T-0174
**配对发布**：T-0173（菜单命名重构）— 打包发

---

## 1. 业务背景（Why）

当前 `/performance/query` 是 mock 壳：

- `omcmb/webcode/src/pages/performance/KPIQuery/index.tsx` 顶部 1000+ 行硬编码 `TEMPLATE_DEVICE_DATA`（ENB00001~15 假数据，时间全 `2026-04-01 10:00`）
- "查询"按钮无效；设备搜索框不联想不过滤
- 模板列表 `PUBLIC_TEMPLATES` / `PRIVATE_TEMPLATES` 是字面量
- 表格列结构错位：只有维度列（serialNumber / hostName / enodeId / cellId），无"指标值列"，看不到 PM 数据

后端 `GET /api/v1/pm/metrics/aggregated` 已存在（`omcgo/internal/pm/handler.go:99,224`），返回 long format（行=metric_value），缺前端 long→wide 透视表组件。

设备规模 10 万→100 万 + 指标 ~1031 条白名单，原 textarea 逗号粘贴 SN 不可用。

**业务目标**：把 `/performance/query` 从 mock 壳替换为可用的指标查询页 — 接真后端 / 列表选 Modal / PM 透视表 / 模板存储；与 T-0173 菜单重构一起释放给运维用。

---

## 2. 用户故事（Who / What）

> As a **网管运维**，I want **在指标查询页选定 N 台设备 + N 个指标 + 时间窗 + 粒度，看到行=时间 / 列=指标的透视表**，So that **直观对比某段时间各指标走势，单元格能 copy 进 Excel**。

> As a **运维工程师**，I want **设备/指标用 Modal 列表挑选（含模糊搜索 + 已选面板 + 服务端分页）**，So that **10 万设备规模下不靠记忆 SN 也能完成查询**。

> As a **复用既有习惯的老用户**，I want **顶部"批量粘贴"入口接受逗号/换行分隔的 SN 列表**，So that **从旧表格复制粘贴的工作流不被破坏**。

> As a **运维主管**，I want **常用查询配置存为公共模板**，So that **下属直接选模板查询，不用重复配 N 个字段**。

> As a **普通运维**，I want **存私有模板**，So that **不污染团队的公共模板列表**。

---

## 3. 验收标准（Given/When/Then）

```
AC-1 查询真后端
Given: 用户在指标查询页选定 device_sn=BLQ-A / metric_paths=PHY.NbrCqi6,PHY.NbrCqi7 / granularity=15min / 时间窗近 1 小时
When:  点击"查询"
Then:  前端调 GET /api/v1/pm/metrics/aggregated?granularity=15min&device_sn=BLQ-A&metric_paths=PHY.NbrCqi6,PHY.NbrCqi7&start_time=...&end_time=...
       表格显示 long→wide 转换结果：行=4 个 15 分钟桶，列=2 个 metric path，单元格=数值
       缺采桶单元格显示 "-"
```

```
AC-2 多设备/多 LDN 列分组
Given: 用户选 2 设备 + 2 指标 + 同 LDN
When:  查询返回 4 列指标值
Then:  列标题形如 `PHY.NbrCqi6 [BLQ-A / LDN-1]` / `PHY.NbrCqi6 [BLQ-B / LDN-1]` / ...
       hover 列标题显示完整 metric_path 全文
```

```
AC-3 设备选择 Modal
Given: 用户点击设备字段右侧"列表选"按钮
When:  Modal 打开
Then:  - 顶部搜索框支持模糊匹配 SN / OUI / 站点名（debounce 300ms 调 GET /api/v1/devices?search=...&page=1&size=20）
       - 表格列：SN / 站点名 / 制式 / 运营商 / 在线状态
       - 多选 checkbox，已选项面板可见可批量删
       - 顶部"批量粘贴"按钮 → 弹小 Modal 接受逗号/换行分隔 SN，校验存在性后加入已选列表
       - 服务端分页 size=20，可翻页
       - 确认按钮把已选 SN 写回查询表单
```

```
AC-4 指标选择 Modal
Given: 用户点击指标字段右侧"列表选"按钮
When:  Modal 打开
Then:  - 顶部模糊搜索（高亮匹配）
       - 中部按指标组折叠列表（L.Thrp.* / HO.* / PHY.* 等，来自 GET /api/v1/pm/indicators?search=...&page=...&size=...）
       - 多选 checkbox + 已选项面板可批量删
       - 底部服务端分页
       - 确认按钮把已选 metric_paths 写回查询表单
```

```
AC-5 模板存储 CRUD
Given: 用户填好查询条件
When:  点击"存为模板"，填名称 / 选 公共/私有
Then:  - 后端 POST /api/v1/pm/query-templates，body={name, visibility:public|private, payload:{device_sns, metric_paths, granularity, time_range_preset, ...}}
       - 公共模板需 super_admin 权限；普通用户只能存私有
       - 列表中可见
       - 选某模板自动填回查询表单
       - 列表也支持 update / delete（仅自己创建 || super_admin）
```

```
AC-6 列表侧栏对接
Given: 页面左侧"KPI 查询模板"侧栏
When:  页面加载
Then:  - 调 GET /api/v1/pm/query-templates?visibility=public + visibility=private（或合并 list 后前端拆桶）
       - 公共模板按名称排序；私有模板按 created_at 倒序
       - 点击某模板：① 填回查询条件 ② 自动触发查询
       - 模板支持收藏星标 / 重命名 / 删除（按权限灰禁）
```

---

## 4. 运营商差异（CMCC / CTCC / CUCC）

无差异。pm_metrics 字典三家共享；查询参数无 carrier 路由。

---

## 5. 非目标（Out of Scope）

- ❌ **不实现导出 Excel/CSV**（侧栏已有 ExportDrawer 占位，留 P4 follow-up）
- ❌ **不做图表视图切换**（当前 KPIQuery 有 LineChart 引用，留作选 P3 阶段后续）
- ❌ **不动 KPI 平台公式 / 跨设备聚合算子**（T-0166 范畴）
- ❌ **不做"对比模式"**（属仪表盘 panel 配置范畴，TODO.MD 阶段 2）
- ❌ **不引入 SSE / WebSocket 实时推**（查询是按需 pull，非实时）
- ❌ **不改后端 `GET /pm/metrics/aggregated` handler 签名**（复用已有）
- ❌ **不做 webcode-v2 / webcode-v3 候选皮肤的同步实现**（业务层 frontend-core 同步即可）

---

## 6. 依赖

- ✅ T-0164 全 done（后端 `GET /pm/metrics/aggregated` 真接生产）
- ✅ 设备 list 端点 `GET /api/v1/devices?search=&page=&size=` 已存在
- ✅ 指标 list 端点 `GET /api/v1/indicators?device_type=ENB|GSM|GNB&keyword=&group_id=&page=&page_size=` 已存在（T-0098 P3-03 `omcgo/internal/pm/indicator/rest_handler.go:52-64`）— **本任务直接复用，无需补端点**
- ✅ super_admin 角色 + PrivateRoute requireSuperAdmin 已存在（T-0098-P4-02）

---

## 7. 度量

- **后端**：5 个 CRUD 端点 + handler/service/repository 测试 + e2e_verify 加 5 个 claim（E/R=5/5 达标）
- **前端**：webcode typecheck 无新 error；新增 util / Modal 单测覆盖 ≥ 8 case；mock 数据从 1000+ 行删干净
- **用户回归**：手测 AC-1..AC-6 全过；时间窗精确到秒；缺采单元格显示 "-" 不显示 "0"

---

## 8. 设计备忘（S2 出口）

### 8.1 后端 — pm_query_templates 表 + 5 CRUD

**Migration 000191_pm_query_templates.sql**（DDL，紧接现有 000190）：

```sql
CREATE TABLE pm_query_templates (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(128) NOT NULL,
    visibility    VARCHAR(16) NOT NULL CHECK (visibility IN ('public','private')),
    creator_id    UUID NOT NULL,                  -- 逻辑关联 admin_users.id (跨域分区表不建外键)
    description   TEXT,
    payload       JSONB NOT NULL,                 -- 查询条件结构 { device_sns:[], metric_paths:[], granularity, time_range_preset, ... }
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uniq_pm_query_templates_creator_name ON pm_query_templates(creator_id, name);
CREATE INDEX idx_pm_query_templates_visibility_created ON pm_query_templates(visibility, created_at DESC);
```

**5 endpoints**（挂 `/api/v1/pm/query-templates`）：

| Method | Path | 行为 | 权限 |
|--------|------|------|------|
| GET    | `/pm/query-templates` | 列表（query: visibility, search, page, size）；返回 `{items, total}` | 普通用户：返回 visibility=public ∪ creator_id=self private；super_admin：全部 |
| GET    | `/pm/query-templates/:id` | 详情 | 同上读权限 |
| POST   | `/pm/query-templates` | 创建；body 含 name/visibility/payload/description | 普通用户只能 visibility=private；super_admin 可 public |
| PATCH  | `/pm/query-templates/:id` | 更新；name/payload/description | 仅 creator_id=self 或 super_admin |
| DELETE | `/pm/query-templates/:id` | 删除 | 仅 creator_id=self 或 super_admin |

**目录布局**：`omcgo/internal/pm/querytemplate/` 子包独立 — `model.go` / `pg_repository.go` / `service.go` / `handler.go`，不污染既有 PM 模块；DI 在 `cmd/app/router/router.go` 接线。

### 8.2 前端 — PM 透视表组件

**位置**：`omcmb/frontend-core/src/utils/pmPivotTransform.ts` (long→wide util) + `omcmb/webcode/src/pages/performance/KPIQuery/components/PivotTable.tsx`

**Util 签名**：
```ts
interface PivotInput {
  rows: Array<{ time: string; metric_path: string; device_sn: string; object_ldn: string; metric_value: number | null }>;
  multiDevice: boolean;   // 是否需要列标题加 [SN/LDN] 后缀
  multiLdn: boolean;
}
interface PivotOutput {
  columns: Array<{ key: string; title: string; tooltipFull: string }>;
  rows: Array<{ time: string; cells: Record<string, number | null> }>;
}
function pivotLongToWide(input: PivotInput): PivotOutput
```

**单测**：覆盖 ① 单设备单指标；② 多设备多指标列标题；③ 多 LDN 列分组；④ 缺采桶 cell=null → 显示 "-"；⑤ 桶按时间升序。

### 8.3 前端 — 设备/指标 Modal pickers

**位置**：`omcmb/webcode/src/pages/performance/KPIQuery/components/DevicePickerModal.tsx` + `MetricPickerModal.tsx`

**DevicePickerModal**：
- antd Modal width=900
- 顶部 `Input.Search` debounce 300ms → 调 useDevices({ search, page, pageSize })
- 中部 antd Table，列：SN / hostName / technology / carrier / online status；rowSelection={ multi }
- 顶部 actions：`批量粘贴` Button → 内嵌小 Modal 接受 textarea (逗号/换行分隔) → 后端 `POST /devices/_validate-sns` body `{sns:[...]}` 校验存在性（不存在的告警）→ 加入已选
- 底部已选项面板（折叠 antd Tag list，可批量删 / 全清）
- 确认按钮关闭并 onSelect(selectedSns)

**MetricPickerModal**：
- antd Modal width=720
- 顶部 `Input.Search`
- 中部按 indicator group 折叠（antd Collapse），每组内列指标 checkbox
- 服务端 `GET /api/v1/indicators?device_type=&keyword=&group_id=&page=&page_size=`（T-0098 P3-03 已存在）；device_type 必填 — 前端按选定设备的 technology 推断（多设备混选时 fallback ENB，或拆 indicator 选择按设备 type 分桶，待 S3 决策）
- 底部已选项面板
- 确认 onSelect(selectedPaths)

### 8.4 前端 — 指标查询页重做

**位置**：`omcmb/webcode/src/pages/performance/KPIQuery/index.tsx` 删除全部 1000+ 行 mock，重写为：

```tsx
// 大致骨架：
<TreeListPageLayout>
  <TemplateSidebar />            {/* 左侧栏：调真 useQueryTemplates */}
  <QueryForm>                    {/* 右侧主区 */}
    <DeviceField onListSelect={openDevicePicker} onBatchPaste={...} />
    <MetricField onListSelect={openMetricPicker} />
    <GranularitySelector />      {/* 15min/hourly/daily/weekly/monthly */}
    <TimeRangeField />           {/* 预设近 1h/24h/7d + 自定义 */}
    <ActionRow>查询 / 存为模板 / 导出占位</ActionRow>
  </QueryForm>
  <PivotTable {...pivotedResult} />
</QueryForm>
```

**业务层**（`frontend-core`）：
- `services/api/pmQueryApi.ts` — 5 个 CRUD 方法 + listAggregated 复用 pmDashboardApi 类似模式
- `hooks/api/usePmQuery.ts` — useQueryTemplates / useCreateTemplate / useDeleteTemplate / useAggregatedMetrics
- `types/pmQuery.ts` — wire 类型 + mapper

### 8.5 待定点（< 3）

1. **指标选择 device_type 推断**：S2 验证 `/api/v1/indicators` 必填 `device_type`（ENB/GSM/GNB）；前端按选定设备的 technology 推断；多设备混选 ENB+GNB 时是否拆 picker 按 device_type 分桶 — S3 决定（倾向 fallback ENB + 提示）
2. **批量粘贴校验端点**：是否直接走 `GET /devices?search=<sn>&size=1` 多次调用，还是新建 `POST /devices/_validate-sns` 批量端点？S3 决定（倾向后者，单次 round trip）
3. **TimeRangePreset 与后端 RFC3339 转换**：前端选"近 1 小时"自动算成 RFC3339 区间，不存预设原值

---

## 9. 风险

| ID | 描述 | 影响 | 缓解 |
|----|------|------|------|
| R-T0174-1 | 后端 `/pm/indicators` 可能不存在，S3 需现补 | S3 延期 ~0.5d | S2 末 grep 确认；若需补，纳入 P2 子任务 |
| R-T0174-2 | KPIQuery 删 1000+ 行 mock 时可能波及外部引用（导出 TemplateItem 等被其它文件 import） | typecheck 红 | grep `from '@/pages/performance/KPIQuery'` 确认；外部引用先抽 type 再删 mock |
| R-T0174-3 | pm_query_templates 表 unique(creator_id, name) 与 super_admin 创建 public 模板撞名情况 | 用户体验小卡 | 创建时前端 check duplicate；后端 23505 错误码友好提示 |
