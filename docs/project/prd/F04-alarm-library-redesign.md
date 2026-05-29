# PRD — 告警库页面 drill-down 主从导航重设计(T-0179)

| 项 | 值 |
|---|---|
| Backlog | T-0179 |
| Type / Prio | feat / P2 |
| Sprint / Owner | sprint-13 / Claude+user |
| Est | M (~3.8d,详 §10) |
| Deps | T-0098 ✅(alarm_definitions schema)+ T-0178 ✅(参考实施模式) |
| 设计文档 | `docs/design/alarm-library-redesign-20260529.md` v1.2 |

---

## 1. 业务背景

当前 `/product/alarm-library` 页是单层扁平表(每行 = 1 个 alarm definition),三大问题:

1. **浏览困难**:单 ne_type 下数十/数百告警定义平铺,用户难按"网元类型"维度查看
2. **筛选冗余**:"网元类型"筛选与表中"网元类型"列重复,UX 多此一举
3. **风格不统一**:操作图标 `EyeOutlined`(查看/编辑双职责)与 T-0178 param-model 页 `EditOutlined` + `DeleteOutlined` 风格脱节

**业务目标**:对齐 param-model 二层 drill-down 模式,改成"网元类型聚合表 → 二级告警定义列表",统一操作图标与 toolbar 单行布局,降低用户认知负担。

## 2. 用户故事

### 2.1 网管运维 — 按网元类型浏览告警库
> 作为运维,**我希望进入告警库就能一眼看到当前所有网元类型(9 个 BLQ/BTS/MLN...)和每个类型下的告警总数 + 严重级别分布**,而不是先看几百行平铺定义再自己分组。

### 2.2 网管运维 — 进二级页面操作具体定义
> 作为运维,**我希望点击"BLQ"行 → 跳到二级页面只显示 BLQ 下的告警定义**,在该上下文里搜索/筛选/新增/编辑/删除;**新增定义时 ne_type 自动填 BLQ 且不可改**,避免误填到其他网元类型。

### 2.3 网管运维 — URL 持久化 + 返回按钮
> 作为运维,**我希望详情态时 URL 含 `?ne_type=BLQ`**,F5 刷新仍在详情态(不被踢回列表态);同时**二级页面顶部有"← 返回"按钮**显式回到列表。

### 2.4 DBA — 看每个 ne_type 的 XML 来源
> 作为 DBA,**我希望列表表里能看到每个网元类型的告警定义来自哪个 XML 文件**(BLQ → BLQ.xml),便于在 `data/alarm-definitions/` 目录下定位源文件。

### 2.5 产品经理 — UI 风格统一
> 作为 PM,**我希望"产品"菜单下所有子页(param-model / alarm-library / kpi-library 等)的操作图标、toolbar 布局、drill-down 模式完全一致**,降低培训成本。

## 3. 验收标准(Given-When-Then)

### GWT-1(列表态显示 ne_type 聚合)
- **Given** alarm_definitions 表有 9 个 ne_type 共 ~880 条告警定义
- **When** 进入 `/product/alarm-library` 默认列表态
- **Then** ① 表格 9 行(每个 ne_type 一行)② 每行有 6 列:网元类型 / 加载源 / 告警总数 / 严重级别分布 / 已识别-未识别 / 操作 ③ 网元类型用 `Tag color="blue"` ④ 严重级别分布 4 色圆点(红/橙/金/蓝)+ 数字

### GWT-2(加载源列正确显示)
- **Given** Loader 重载后 alarm_definitions.loaded_from 列已写入文件名
- **When** GET `/api/v1/alarm-definitions/ne-types`
- **Then** 响应每行含 `loaded_from` 字段(如 `BLQ.xml`),前端列展示并 ellipsis 长名

### GWT-3(点击 ne_type 进二级页面 + URL 同步)
- **Given** 列表态显示 BLQ 行
- **When** 点击 "BLQ" 链接或操作列 📋 详情按钮
- **Then** ① 切换到详情态 ② URL 变为 `/product/alarm-library?ne_type=BLQ` ③ 显示 BLQ 下所有告警定义 ④ toolbar 顶部显示"← 返回 BLQ 的告警定义"

### GWT-4(F5 刷新仍在详情态)
- **Given** 当前在 `/product/alarm-library?ne_type=BLQ`
- **When** F5 刷新页面
- **Then** ① 仍处于详情态 ② 仍显 BLQ 下的告警定义(不被踢回列表)

### GWT-5(二级页面"返回"按钮回列表)
- **Given** 当前在详情态
- **When** 点击顶部"← 返回"按钮
- **Then** ① URL 变为 `/product/alarm-library`(去掉 query)② 显示 9 行 ne_type 聚合表

### GWT-6(新增定义预填 ne_type 且锁定)
- **Given** 当前在 BLQ 详情态
- **When** 点击 [+ 新增定义] 按钮
- **Then** ① AlarmDefinitionDrawer 打开 ② Form.Item "网元类型" 字段默认值 = "BLQ" ③ 该字段 `disabled` 不可改 ④ 保存后新定义自动归到 BLQ

### GWT-7(详情态工具齐全)
- **Given** 当前在 BLQ 详情态
- **When** 检查 toolbar
- **Then** ① 显示 [← 返回] [BLQ 的告警定义] [搜索 identifier/名称] [严重级别▼] [识别状态▼] [⚠ 未识别频次] [+ 新增定义] [⬇ 重载 XML] [↻ 刷新缓存] ② **不显示**"网元类型"筛选(冗余)

### GWT-8(编辑 EditOutlined)
- **Given** 详情态告警定义行
- **When** 点击操作列 ✏️ EditOutlined 按钮
- **Then** ① Drawer 打开,加载该定义数据 ② 用户可改字段并保存 ③ 保存后 List query invalidate

### GWT-9(删除 DeleteOutlined)
- **Given** 详情态告警定义行
- **When** 点击操作列 🗑 DeleteOutlined → Popconfirm "确认删除告警定义「12001」?"
- **Then** ① 确认后 DELETE `/alarm-definitions/12001` ② 表格行消失 ③ ne_type 聚合数字 -1

### GWT-10(移除"网元类型"筛选)
- **Given** 列表态 / 详情态 toolbar
- **When** 检查所有筛选项
- **Then** ① 列表态搜索只针对 ne_type 名称 ② 详情态搜索只针对 identifier/中文名/英文名 ③ 全页面**找不到**"网元类型"下拉筛选

### GWT-11(重载 XML 后 loaded_from 列正确填充)
- **Given** 重载 XML 之前 loaded_from 列存在但部分 NULL
- **When** POST `/alarm-definitions/import-directory` 触发重载
- **Then** ① 所有行 loaded_from 字段非 NULL ② ne-types API 响应每行 loaded_from 字段为对应 XML 文件名

## 4. 运营商差异矩阵

**无运营商差异**。三家运营商(cmcc/ctcc/cucc)共用同一套 alarm_definitions 表,本 UI 改动对所有运营商一视同仁。

## 5. 非目标

- **不做**:alarm_definitions 表大规模 schema 改动(只加 1 列 `loaded_from`)
- **不做**:ne_type 维度的 RBAC(沿用 super_admin 全局)
- **不做**:ne_type 树形分组(平铺 9 行即可)
- **不做**:AlarmDefinitionDrawer 内部 Form 重写(只在外部传 `defaultNeType` + `lockNeType` 入参)
- **不做**:未识别频次按 ne_type 维度的二次聚合(沿用全局,详情态调用时传 ne_type 过滤)
- **不做**:历史 loaded_from NULL 行回填(等下次重载 XML 时自动填,无需手工迁移数据)
- **不做**:批量上传 alarm XML(沿用现有 import-directory 端点)
- **不做**:webcode-v2/webcode-v3 UI 重做(typecheck 通过即可,后续皮肤复刻独立任务)

## 6. 依赖

- T-0098 ✅ alarm_definitions / alarm_severity_levels schema
- T-0178 ✅ 参考实施模式(drill-down 主从导航 + 操作图标统一)
- 现有 `internal/alarm/definition/handler.go`(7 个 REST 端点)
- 现有 `omcmb/webcode/src/pages/product/alarm-library/AlarmDefinitionDrawer.tsx`(只加 2 入参,不重写)
- 现有 `omcmb/webcode/src/pages/product/alarm-library/UnknownStatsModal.tsx`(详情态调用时传 ne_type 过滤)
- 现有 `react-router-dom` v6 `useSearchParams`(URL 同步)

## 7. 度量

| 指标 | 目标值 |
|------|--------|
| 列表态首屏渲染时间(GET /ne-types + 表绘制)| < 200ms |
| ne_type → 详情态切换延迟 | < 100ms(URL push + state 切换,无网络等待:数据已 prefetch 或同步触发) |
| ne-types 聚合 API p99 latency | < 50ms(9 行聚合,单 SQL `GROUP BY`) |
| 列表态列数 | 6 列(对齐 param-model 列表 8 列简化版) |
| 详情态操作图标 | 2 个(✏️ EditOutlined + 🗑 DeleteOutlined),Drawer 内"取消"按钮显式退出 |
| E2E 用例覆盖 | ≥ 11 个 GWT(GWT-1..11 各 ≥1 check_status 断言) |
| 三皮肤 typecheck | webcode + webcode-v2 + webcode-v3 全过 |

## 8. 风险

| ID | 风险 | 等级 | 缓解 |
|----|------|------|------|
| R-NEW-T0179-1 | webcode-v2/v3 可能有自己版本 alarm-library 页,跟主皮肤同步改动有遗漏 | P3 | grep 三皮肤的 `alarm-library/index.tsx` + 加 typecheck CI 卡 |
| R-NEW-T0179-2 | AlarmDefinitionDrawer 改为 EditOutlined(去 Eye)后,用户误改字段后直接关 Drawer 会保存,与"查看"语义脱节 | P3 | Drawer 内保留"取消"按钮显式退出;Save 必须显式点击 |
| R-NEW-T0179-3 | ne_type 聚合 API 严重级别映射写错(red/orange/gold/blue 配 severity_code) | P2 | 单测覆盖完整映射表(8 个 code: 1/2/3/4 + 31001..31004) |
| R-NEW-T0179-4 | migration 加 loaded_from 列对生产 alarm_definitions 大表执行 ALTER TABLE 可能短暂阻塞 | P3 | ADD COLUMN with default NULL 不重写表数据,PG 12+ 是 O(1) 元数据改动,实际不阻塞 |
| R-NEW-T0179-5 | `useSearchParams` 与现有 router state 冲突 | P3 | 单测覆盖 URL 同步 + 浏览器 F5/前进/后退三种场景 |
| R-NEW-T0179-6 | 严重级别分布的 4 色圆点对色盲用户不友好 | P3 | Tooltip 显文字"致命/严重/一般/警告";色盲 ARIA 可单独 follow-up |

## 9. 实施细节(S2 设计备忘)

### 9.1 数据迁移

`migrations/000216_alarm_definitions_add_loaded_from.sql`:
```sql
-- +goose Up
ALTER TABLE alarm_definitions
  ADD COLUMN IF NOT EXISTS loaded_from VARCHAR(256);

COMMENT ON COLUMN alarm_definitions.loaded_from IS
  'T-0179: XML 源文件 basename(如 BLQ.xml);Loader.batchUpsertAlarms 重载后填,历史数据 NULL 直到下次重载';

-- +goose Down
ALTER TABLE alarm_definitions DROP COLUMN IF EXISTS loaded_from;
```

注:版本号 216 = 当前最大 215(T-0178 已用)+ 1。

### 9.2 Loader 改写

`omcgo/internal/alarm/definition/loader.go::batchUpsertAlarms` 入参加 `loadedFrom string`:
```go
func batchUpsertAlarms(
    ctx context.Context, tx pgx.Tx,
    neType, loadedFrom string,  // ← 新加
    alarms []xmlAlarm, ...
) (int, error) {
    // INSERT INTO alarm_definitions (..., loaded_from) VALUES (..., $N)
    // ON CONFLICT DO UPDATE SET ..., loaded_from = EXCLUDED.loaded_from
}
```

调用方 `loadAlarmFile` 内 `filepath.Base(path)` 传入。

### 9.3 新增 API

`GET /api/v1/alarm-definitions/ne-types?keyword=`,响应:
```json
{
  "ret": 1, "msg": "ok",
  "data": {
    "items": [
      {
        "ne_type": "BLQ",
        "loaded_from": "BLQ.xml",
        "total": 142,
        "severity_breakdown": {"critical": 46, "major": 38, "minor": 35, "warning": 23},
        "known_count": 140,
        "unknown_count": 2
      }
    ],
    "total": 9
  }
}
```

SQL(单条聚合,无 N+1):
```sql
SELECT
  ne_type,
  MIN(loaded_from) AS loaded_from,
  COUNT(*) AS total,
  COUNT(*) FILTER (WHERE severity_code IN (1, 31001)) AS critical,
  COUNT(*) FILTER (WHERE severity_code IN (2, 31002)) AS major,
  COUNT(*) FILTER (WHERE severity_code IN (3, 31003)) AS minor,
  COUNT(*) FILTER (WHERE severity_code IN (4, 31004)) AS warning,
  COUNT(*) FILTER (WHERE is_unknown = false) AS known_count,
  COUNT(*) FILTER (WHERE is_unknown = true) AS unknown_count
FROM alarm_definitions
WHERE ($1 = '' OR ne_type ILIKE '%' || $1 || '%')
GROUP BY ne_type
ORDER BY ne_type;
```

### 9.4 前端文件结构

```
omcmb/webcode/src/pages/product/alarm-library/
├── index.tsx                  ← 改:管 selectedNeType + URL 同步 + toolbar 单行
├── NeTypesTab.tsx             ← 新增(列表态聚合表)
├── DefinitionsTab.tsx         ← 新增(详情态,由现有 index.tsx 精简而来)
├── SeverityDot.tsx            ← 新增小组件(着色圆点 + 数字 + Tooltip)
├── AlarmDefinitionDrawer.tsx  ← 加 2 入参 defaultNeType + lockNeType
└── UnknownStatsModal.tsx      ← 加 1 入参 neType 作过滤
```

### 9.5 frontend-core 类型 / API / Hook

```typescript
// types/alarmDefinition.ts
export interface AlarmDefinitionNeType {
  neType: string;
  loadedFrom?: string;
  total: number;
  severityBreakdown: { critical: number; major: number; minor: number; warning: number };
  knownCount: number;
  unknownCount: number;
}

// services/api/alarmDefinitionApi.ts
listNeTypes(keyword?: string): Promise<{ items: AlarmDefinitionNeType[]; total: number }>

// hooks/api/useAlarmDefinitions.ts
useAlarmDefinitionNeTypes(keyword?: string)
```

### 9.6 URL 同步实现

`index.tsx` 使用 `react-router-dom` v6 `useSearchParams`:
```typescript
const [searchParams, setSearchParams] = useSearchParams();
const selectedNeType = searchParams.get('ne_type') || undefined;

const setSelected = (ne?: string) => {
  if (ne) setSearchParams({ ne_type: ne }, { replace: false });
  else setSearchParams({}, { replace: false });
};
```

### 9.7 操作图标规范(对齐 param-model)

| 动作 | 图标 | antd 名 |
|------|------|---------|
| 编辑 | ✏️ | `EditOutlined` |
| 删除 | 🗑 | `DeleteOutlined danger` |
| 新增 | + | `PlusOutlined primary` |
| 重载 | ⬇ | `CloudDownloadOutlined danger` |
| 缓存 | ↻ | `ReloadOutlined` |
| 详情 | 📋 | `FileSearchOutlined` |
| 频次 | ⚠ | `WarningOutlined` |
| 返回 | ← | `ArrowLeftOutlined` |

### 9.8 Drawer 入参扩展

```typescript
interface AlarmDefinitionDrawerProps {
  open: boolean;
  definition: AlarmDefinition | null;
  defaultNeType?: string;  // ← 新增,新建时预填
  lockNeType?: boolean;    // ← 新增,锁定网元类型字段不可改
  onClose: () => void;
}
```

详情态调用:`<Drawer defaultNeType={selectedNeType} lockNeType />`。

## 10. 实施清单(工期分解)

| Phase | 内容 | 工期 | Owner |
|-------|------|------|-------|
| P1 后端 | migration 216 + Loader 改写 + ne-types handler/service/repo + 单测 + E2E +1 assert | 1.1d | Claude |
| P2 前端 frontend-core | types + listNeTypes API + useAlarmDefinitionNeTypes hook + mock 数据 | 0.3d | Claude |
| P3 前端 webcode | NeTypesTab + DefinitionsTab + SeverityDot + index.tsx + URL 同步 | 1.7d | Claude |
| P4 三皮肤 typecheck + Drawer/Modal 扩展 | webcode-v2/v3 grep + Drawer 加入参 + UnknownModal 加 ne_type | 0.2d | Claude |
| P5 单测 | NeTypesTab / DefinitionsTab / URL 同步 / SeverityDot | 0.3d | Claude |
| P6 文档 | omcgo/CLAUDE.md §5.4 加 alarm-library 段;backlog/sprint 日志 | 0.2d | Claude |

**合计 3.8d**;sprint-13 内单 PR 提交(或按 P1 / P2-3 / P4-6 拆 3 PR review)。

## 11. 实测产物(S5 实施完毕后填)

```
(待 S5 完成填写,字段示例:
- 后端新增文件数 / 改动行数
- 前端 typecheck / vitest 通过情况
- E2E 11 用例全部 PASS
- 浏览器演练:列表态 → 点 BLQ → URL 同步 → F5 仍详情态 → 返回 → 新增定义预填 BLQ → 编辑 → 删除
- ne-types API 响应时间 < 50ms
- 三皮肤 typecheck 0 错
)
```

## 12. 风险登记同步

提案合入后,`docs/project/risk-register.md` 加 6 条 `R-NEW-T0179-*`,Owner=Claude+user。

---

**版本历史**:2026-05-29 v1.0 起草(S0 → S2 设计备忘合并;6 轮决策已对齐:① 4 色圆点 ② 详情态未识别频次 ③ URL 同步 + 返回按钮 ④ 加 loaded_from 列 ⑤ 新增预填且锁定 ⑥ 去 EyeOutlined)。设计文档:`docs/design/alarm-library-redesign-20260529.md` v1.2。
