# KPI 指标库页面重构设计 — 对齐 T-0178 自定义 XML 模式

**作者**:Claude(对话整理)
**日期**:2026-05-29
**功能域**:F03 PM(指标库)
**对标方案**:T-0178(参数模型 / `product/param-model`)
**状态**:设计草案,等待评审

---

## 1. 背景与目标

### 1.1 现状

`product/kpi-library` 当前实现:

- **前端** [omcmb/webcode/src/pages/product/kpi-library/](omcmb/webcode/src/pages/product/kpi-library/)
  - 顶部 toolbar:一个"XML 导入/重载"按钮(合并)+ "刷新缓存"按钮
  - Tabs 横排 4 个:ENB (LTE) | GSM | GNB (5G NR) | **单位定义**
  - 前三个 Tab 共用 `IndicatorTab.tsx`(列表 + 搜索 + 启用/禁用)
  - 单位定义 Tab `IndicatorUnitsTab.tsx`(单位增删改)
- **后端** [omcgo/internal/pm/indicator/](omcgo/internal/pm/indicator/)
  - Loader 一次性扫 `data/indicator-library/{enb/*.xml, GSM.xml, GNB.xml}`,全量 UPSERT
  - 没有"自定义 XML"概念,XML 默认全是内置
  - 没有上传端点,没有按文件删除
  - 表:`perf_indicators_{enb,gsm,gnb}`、`indicator_group_{enb,gsm,gnb}`、`indicator_unit`(共享)、`rela_platform_indicator_formula_{enb,gsm,gnb}`、`enabled_pm_indicators_{enb,gsm,gnb}`
  - XML 根元素 `<indicatorModel platform="..." deviceType="GSM|..." indicatorCount="N">`

### 1.2 目标(用户需求 5 条)

1. 整体 UI 布局参考 `product/param-model`(drill-down + 顶部统一 toolbar)
2. ENB (LTE) / GSM / GNB (5G NR) 作为列表的**三行数据**(不再用 Tabs)
3. "单位定义" 移到顶部操作按钮 → 点击从**右侧抽屉**打开,内部仍含 CRUD
4. "XML 导入/重载" 参照 param-model **拆成两个按钮**:导入 XML(加法 UPSERT)+ 重载 XML(destructive 全量重载)
5. **新增"上传 XML"**:参照 T-0178,自定义 XML 在升级时**不被重置**(host bind mount 持久化)

### 1.3 非目标(本次不做)

- 不重写指标的启用/禁用 / 公式 / 平台映射模型(这些是另一条链)
- 不动 perf_indicators_{enb,gsm,gnb} 三表分表结构(单表合并是另一个独立 T)
- 不引入"指标行级"删除 — 删除粒度统一在 **XML 文件**层(对齐 param-model 的"按模型删")

---

## 2. UI 布局设计

### 2.1 顶层页面 — 三行列表

布局完全对标 [omcmb/webcode/src/pages/product/param-model/index.tsx](omcmb/webcode/src/pages/product/param-model/index.tsx) 的列表态:

```
┌─────────────────────────────────────────────────────────────────────┐
│  KPI 指标库 (顶部 Card)                                              │
│ ┌─────────────────────────────────────────────────────────────────┐ │
│ │ [搜索 制式/描述 ─────]    [单位定义] [上传 XML] [导入 XML] [重载 XML] [刷新缓存] │ │
│ └─────────────────────────────────────────────────────────────────┘ │
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │ 制式      │ 指标数 │ 分组数 │ XML 文件      │ 平台     │ 操作 │   │
│  ├──────────────────────────────────────────────────────────────┤   │
│  │ ENB (LTE) │ 1163   │ 12     │ 5 内置 / 0 自定义 │ ALL,BLQ... │ [详情] │
│  │ GSM       │ 73     │ 1      │ 1 内置 / 0 自定义 │ BSC          │ [详情] │
│  │ GNB (5G NR)│ 211   │ 6      │ 1 内置 / 0 自定义 │ ALL          │ [详情] │
│  └──────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

**主表列**:

| 列 | 含义 | 来源(后端字段) |
|---|---|---|
| 制式 | LTE / GSM / NR(带图标) | hardcoded 3 行 |
| 指标数 | 该制式下 perf_indicators_* 行数 | COUNT(*) |
| 分组数 | indicator_group_* 行数 | COUNT(*) |
| XML 文件 | "M 内置 / N 自定义" | 按 loaded_from 前缀分组 |
| 平台 | distinct platform_name 列表 | indicators.distinct(platform) |
| 操作 | [详情] 按钮(点行任何位置也可) | — |

**主表无单条删除按钮**(粒度不到制式)。`制式`列点击 = 进入 drill-down 详情。

### 2.2 详情(drill-down)— 指标列表

点制式行的"详情"或制式名 → 主区域切换到该制式的指标列表(对标 param-model 的 MappingsTab 形态):

```
┌─────────────────────────────────────────────────────────────────────┐
│ KPI 指标库 (顶部 Card)                                               │
│ ┌─────────────────────────────────────────────────────────────────┐ │
│ │ [← 返回]  ENB (LTE) 的指标列表       [管理 XML 文件] [刷新缓存] │ │
│ └─────────────────────────────────────────────────────────────────┘ │
│                                                                     │
│ ┌─────────────────────────────────────────────────────────────────┐ │
│ │ [搜索 中英文名]  [平台 ▼ ALL]                                    │ │
│ │ ID │ 中文名 │ 英文名 │ 分组 │ 类型 │ 单位 │ 来源 │ 启用 │ 操作   │ │
│ │ ─────────────────────────────────────────────────────────────── │ │
│ │ C00…│ 切换…│ HO.… │ 切换组│ 整数 │ time │ 内置 │ [√] │ [编辑]   │ │
│ │ ...                                                              │ │
│ └─────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

新增一列:

- **来源 Tag**:`内置` / `自定义` / `未知`(灰)— 取自该指标所属 XML 的 `loaded_from` 列

无单条删除(粒度到文件,见下)。

### 2.3 "管理 XML 文件"— 详情子工具栏按钮

点详情顶部的"管理 XML 文件" → 弹 Modal 列出**该制式下所有 XML 文件**,每行带:

| 列 | 含义 |
|---|---|
| 文件名 | `enb/ALL.xml` / `enb/BLQ.xml` / ... |
| 来源 Tag | 内置 / 自定义 |
| 包含指标数 | COUNT(*) WHERE loaded_from = ? |
| 操作 | [删除](仅自定义可点,内置灰 + Tooltip) |

删除一个自定义 XML = 物理 `.deleted.<ts>` 备份 + DB 对应指标级联清理 + 详情列表刷新。

### 2.4 "单位定义"— 右侧抽屉

顶部"单位定义"按钮 → Drawer(`width=720`, `placement="right"`)打开,内容 = 现 [IndicatorUnitsTab.tsx](omcmb/webcode/src/pages/product/kpi-library/IndicatorUnitsTab.tsx) 完整搬迁:

```
┌────────────────────────────────────┐
│ × 单位定义                          │
├────────────────────────────────────┤
│ [搜索 单位 ID/名称]   [+ 新增单位]  │
│ ────────────────────────────────── │
│ ID    │ 英文名 │ 中文名 │ 操作      │
│ time  │ Time   │ 时间   │ [编辑][删除] │
│ ...                                 │
└────────────────────────────────────┘
```

单位是**共享**资源(跨三个制式),所以不放在 drill-down 里,而是页面级抽屉。

### 2.5 "上传 XML"— 模态框

参照 param-model 的 Upload 流程,**关键差异**:多了一个"目标制式"选择(因为 indicator XML 按制式分目录)。

```
┌────────────────────────────────────────┐
│ 上传自定义 XML                          │
├────────────────────────────────────────┤
│ 目标制式 *  [ ENB (LTE)        ▼ ]      │
│ XML 文件 *  [选择文件...]  (.xml,≤1MB) │
│                                         │
│ 文件名规则:^[A-Za-z0-9_-]{1,64}\.xml$  │
│ XML 根元素必须为 <indicatorModel>       │
│ 同名文件已存在时会询问是否覆盖           │
│                                         │
│             [取消]  [上传]              │
└────────────────────────────────────────┘
```

上传成功后 → toast 提示 + 自动 refetch 主表行(该制式 "M 内置 / N 自定义" 计数更新)。
409 冲突 → 二级 Modal 询问 `force=true`,确认后服务端 `.bak.<ts>` 备份再覆盖(同 T-0178)。

---

## 3. 数据模型与目录改造

### 3.1 目录布局

```
data/                                      # 镜像层 COPY,内置
  indicator-library/
    enb/                                   ← 多文件
      ALL.xml  BLQ.xml  BM.xml  ...
    GSM.xml                                ← 单文件
    GNB.xml                                ← 单文件

/opt/omc/data/                             # host bind mount,自定义,升级不丢
  indicator-library-custom/
    enb/                                   ← 与内置同结构
    gsm/
    gnb/
```

合并规则(沿用 T-0178 `mergeFileLists`):

- 内置 + 自定义按相对路径合并
- 同名文件**自定义胜出**(`custom_overrides_builtin = true`,默认)
- `loaded_from` 列写**完整前缀路径**:`indicator-library/enb/ALL.xml` 或 `indicator-library-custom/enb/MY.xml`

### 3.2 表结构变更

三张指标表加 `loaded_from` 列:

```sql
-- migrations/000NNN_indicators_loaded_from.sql
ALTER TABLE perf_indicators_enb ADD COLUMN IF NOT EXISTS loaded_from VARCHAR(256) NOT NULL DEFAULT '';
ALTER TABLE perf_indicators_gsm ADD COLUMN IF NOT EXISTS loaded_from VARCHAR(256) NOT NULL DEFAULT '';
ALTER TABLE perf_indicators_gnb ADD COLUMN IF NOT EXISTS loaded_from VARCHAR(256) NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_perf_indicators_enb_loaded_from ON perf_indicators_enb(loaded_from);
-- 同对 gsm/gnb 加索引
-- 回填:Loader 下次跑时会写,旧数据保持 '' 即 Source=unknown(不可删,安全)
```

迁移号待版本规划(参考 [omcgo/CLAUDE.md §5.5](omcgo/CLAUDE.md))。

> **取舍**:不在 `indicator_group_*` 加 loaded_from — 分组是跨文件共享的,删除文件时不必动分组(分组无指标引用时由 cleanup 单独处理或保持)。

### 3.3 Source 分类规则(沿用 T-0178)

新文件 `omcgo/internal/pm/indicator/source.go`:

```go
const (
    BuiltinDirPrefix = "indicator-library/"
    CustomDirPrefix  = "indicator-library-custom/"
)

type Source int
const (
    SourceUnknown Source = iota
    SourceBuiltin
    SourceCustom
)

func ClassifySource(loadedFrom string) Source { /* 前缀判定 */ }
func IsDeletable(loadedFrom string) bool      { /* 仅 SourceCustom = true */ }
```

---

## 4. 后端 API 设计

### 4.1 端点清单

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/indicators/summary` | GET | **新增** 返回三行汇总(制式 / 指标数 / 分组数 / 来源分组 / 平台数) |
| `/api/v1/indicators/import-directory?mode=import` | POST | **改造** 加 `mode` 参数;`import` = 加法 UPSERT(同今天) |
| `/api/v1/indicators/import-directory?mode=reload` | POST | **新增分支** destructive 全量重载;按 updated_at < start 删孤儿(`perf_indicators_*`),级联 `rela_platform_indicator_formula_*`、`enabled_pm_indicators_*` |
| `/api/v1/indicators/upload-xml?tech=enb\|gsm\|gnb&force=false\|true` | POST | **新增** multipart 上传到 `indicator-library-custom/<tech>/<name>.xml`,上传后自动 reload |
| `/api/v1/indicators/files?tech=enb\|gsm\|gnb` | GET | **新增** 列出该制式下所有 XML 文件(从 DB distinct loaded_from + 物理目录扫描合并去重) |
| `/api/v1/indicators/files/{path}` | DELETE | **新增** 删除自定义 XML 文件,内置返 403;物理 `.deleted.<ts>` + 级联清理 DB 行 |

GET/POST/PUT/DELETE `/indicators[/:id]` 与 `/indicator-units[/:id]` 保持现状(不动)。

### 4.2 上传验证器(沿用 T-0178 `upload.go` 四件套)

新文件 `omcgo/internal/pm/indicator/upload.go`:

| 检查 | 规则 |
|------|------|
| 文件名 | `^[A-Za-z0-9_-]{1,64}\.xml$`,禁路径分隔符 / 点开头 / 保留名(`ALL.xml` `GSM.xml` `GNB.xml`?见 §10 决策) |
| XML 合法 | `xml.NewDecoder` Strict=true 走一遍 |
| 根元素 | `<indicatorModel>`(配合 deviceType / platform 属性存在性检查) |
| 大小 | ≤ 1 MiB(`MaxUploadXMLSize = 1<<20`) |

### 4.3 文件锁(沿用 T-0178)

`acquireFileLock(basename)` — `sync.Map[string]*sync.Mutex`,Upload + Delete + 单文件 Reload 三方共享同一把锁。

> **多实例假设**:同 param-model — 单实例为前提,横扩前需补 PG advisory lock。

### 4.4 删除时的级联策略

删除一个自定义 XML 文件(`indicator-library-custom/enb/MY.xml`)的步骤:

1. 物理 `mv MY.xml MY.xml.deleted.<ts>`(备份,worker cron 30 天后清)
2. DB 事务:
   - `DELETE FROM perf_indicators_enb WHERE loaded_from = 'indicator-library-custom/enb/MY.xml'`
   - `DELETE FROM rela_platform_indicator_formula_enb WHERE indicator_id IN (...)`(被删指标的公式)
   - `DELETE FROM enabled_pm_indicators_enb WHERE indicator_id IN (...)`(启用记录)
3. 不动 `indicator_group_*`(分组可能被其他文件引用)
4. 刷缓存

> 备份失败 → 全流程**保守回滚**(同 T-0178),返 500 + 新错误码 `ErrCodeIndicatorBackupFailed=20NN`(待编号)。

### 4.5 新增错误码(`global/errors.go`)

| 错误码 | 含义 |
|--------|------|
| `ErrCodeIndicatorBuiltinNotDeletable = 20XX` | DELETE 内置 XML 返 403 |
| `ErrCodeIndicatorBackupFailed = 20XX+1` | 删除时备份失败保守回滚返 500 |
| `ErrCodeIndicatorUploadInvalidTech = 20XX+2` | upload-xml 的 `tech` 不在 enb/gsm/gnb |

---

## 5. 前端实现拆分

### 5.1 文件清单

```
omcmb/webcode/src/pages/product/kpi-library/
├── index.tsx                 ← 重写:列表态 + drill-down 切换 + 顶部 toolbar
├── SummaryTab.tsx            ← 新增:三行制式表
├── IndicatorsByTech.tsx      ← 新增:drill-down 详情(替代原 IndicatorTab)
├── XMLFilesModal.tsx         ← 新增:"管理 XML 文件" 弹窗
├── UploadXmlModal.tsx        ← 新增:上传 XML 表单
├── UnitsDrawer.tsx           ← 新增:抽屉壳(内容来自 IndicatorUnitsTab)
└── IndicatorUnitsTab.tsx     ← 保留:抽屉的内容主体(轻改名也行)
```

`IndicatorTab.tsx`(旧)逻辑搬到 `IndicatorsByTech.tsx`,加"来源"列。

### 5.2 frontend-core 新增

[omcmb/frontend-core/src/services/api/indicatorLibraryApi.ts](omcmb/frontend-core/src/services/api/indicatorLibraryApi.ts):

```ts
indicatorLibraryApi.summary()                  // GET /indicators/summary
indicatorLibraryApi.reloadDirectory()          // POST .../import-directory?mode=reload
indicatorLibraryApi.uploadXml(tech, file, force) // POST .../upload-xml
indicatorLibraryApi.listFiles(tech)            // GET .../files?tech=
indicatorLibraryApi.deleteFile(loadedFrom)     // DELETE .../files/{path}
```

[omcmb/frontend-core/src/hooks/api/useIndicatorsLibrary.ts](omcmb/frontend-core/src/hooks/api/useIndicatorsLibrary.ts) 同步加 hook(`useIndicatorSummary` / `useIndicatorReload` / `useIndicatorUploadXml` / `useIndicatorFiles` / `useIndicatorDeleteFile`),invalidate 规则照搬 useParamModels。

### 5.3 现 Tab 结构 → drill-down 状态机

```ts
const [selectedTech, setSelectedTech] = useState<'enb'|'gsm'|'gnb'|undefined>();
const [unitsOpen, setUnitsOpen] = useState(false);
const inDetail = Boolean(selectedTech);
```

- 列表态:渲染 `<SummaryTab onSelect={setSelectedTech} />`
- 详情态:渲染 `<IndicatorsByTech tech={selectedTech} />`,toolbar 左侧切换为 `[← 返回] {tech} 的指标列表`

---

## 6. 部署改动

### 6.1 docker compose

[deployments/docker/docker-compose.app.yml](deployments/docker/docker-compose.app.yml)(以及 acs / worker 视配置)中,**app / worker 共享同一份** custom 目录:

```yaml
services:
  app:
    volumes:
      - /opt/omc/data/indicator-library-custom:/etc/omcgo/data/indicator-library-custom:rw
  worker:
    volumes:
      - /opt/omc/data/indicator-library-custom:/etc/omcgo/data/indicator-library-custom:rw
```

### 6.2 deploy.sh 初始化

[deployments/release/bundle/deploy/deploy.sh](deployments/release/bundle/deploy/deploy.sh) 在"建立目录结构"那一步,新增:

```bash
sudo mkdir -p /opt/omc/data/indicator-library-custom/{enb,gsm,gnb}
sudo chown -R 1000:1000 /opt/omc/data/indicator-library-custom  # 与容器内 uid 对齐
```

> 与 T-0178 的 param-mappings-custom 用同一套约定,运维侧零学习成本。

### 6.3 worker cron 备份清理

复用 T-0178 的 `BackupCleanup` 框架,新增对 `data/indicator-library-custom/**/*.{deleted,bak,tmp}.*` 的清理(30 天 / 1 小时同 T-0178 阈值)。
新指标 `indicator_backup_cleanup_total{kind="deleted|bak|tmp", result="swept|error|skipped"}`。

---

## 7. 实施分阶段(参考 T-0178 8 commit 节奏)

| 阶段 | 内容 | 估时 | 可独立合入? |
|------|------|------|---------|
| **P1.1** 后端 source 分类器 | source.go + 单测 | 0.5d | ✓ |
| **P1.2** 后端 Loader 双目录合并 + loaded_from 写入 | loader.go + filelist.go + 迁移加 loaded_from 列 | 1d | ✓ |
| **P1.3** 后端 Delete by file + 守门 + 错误码 | handler.go 新端点 + 守门 | 1d | ✓ |
| **P1.4** 后端 Upload + 验证器 + summary 端点 | upload.go + handler.go + DTO 加 source/deletable | 1d | ✓ |
| **P1.5** 后端 import-directory 加 mode=reload | handler.go + DeleteOrphansSince(三表) | 0.5d | ✓ |
| **P2** worker BackupCleanup 扩展 | backup_cleanup.go 加 indicator-library-custom 路径 + 指标 | 0.3d | ✓ |
| **P3** 部署改造 | docker-compose volume + deploy.sh mkdir + bundle 同步 | 0.3d | ✓ |
| **P4** 前端重构 | 五个新文件 + hooks + apiService | 1.5d | 依赖 P1 全部上线 |

**合计**:约 6 个工作日(单人,不含设计评审)。

每个阶段 1 commit,最后 P5 docs commit(本文档 + CLAUDE.md §5.3.2 补充 + risk-register 条目),共 ~9 个 commit。

---

## 8. 风险与取舍

| 风险 | 缓解 |
|------|------|
| **三表分表(enb/gsm/gnb)使 handler 重复代码激增** | 抽 `tech` 参数 + 共用 helper 函数;接受 ~20% 代码重复换 schema 简单 |
| **loaded_from 回填**:旧数据 `''` → SourceUnknown → 不可删 | 安全(只读永远不会误删);需要时跑一次 reload 全量回填,等同 T-0178 P1.2 |
| **deviceType 与目录的映射**:GSM.xml(单文件)vs enb/*.xml(目录) | 上传到 `indicator-library-custom/gsm/<name>.xml`(目录化),Loader 内置侧依然认 `GSM.xml` 单文件;自定义侧统一目录化 |
| **重载孤儿删除可能误删运维手动启用的指标** | reload 是 "destructive" 语义,UI 上要给 Popconfirm + 数量预览;沿用 param-model 红色 danger 按钮 + 描述提示 |
| **指标公式 / 启用状态**:删 XML 时是否带走 | 是,见 §4.4 — 自定义指标的公式/启用是绑定关系,文件删了它们一起删;内置不删 |
| **单实例假设**(file lock) | 与 T-0178 一致风险,等多实例横扩时统一补 PG advisory lock |

---

## 9. 监控 / 告警新增项

| 指标 | 类型 | 维度 |
|------|------|------|
| `indicator_upload_total` | counter | result="ok\|invalid_filename\|invalid_xml\|too_large\|conflict\|backup_failed" |
| `indicator_delete_total` | counter | source="builtin\|custom", result="ok\|forbidden\|backup_failed" |
| `indicator_reload_orphans_deleted_total` | counter | tech="enb\|gsm\|gnb" |
| `indicator_backup_cleanup_total` | counter | kind="deleted\|bak\|tmp", result="swept\|error\|skipped" |

---

## 10. 待用户确认决策

实施前需要落定的口径:

### D1. 自定义 XML 文件**保留名**白名单

param-model 把 `standard-model.xml` / `products.xml` 等列为保留名。对 indicator,我倾向把以下列为**保留名**(自定义文件不能用):

- `ALL.xml`, `BLQ.xml`, `BLX.xml`, `BM.xml`, `ENB_DEFAULT_098.xml`(LTE 内置)
- `GSM.xml`, `GNB.xml`(单文件内置 — 注意 custom 侧是目录化的,这里只是保险)

**问题**:用户期望保留名清单是否需要更宽松?例如允许 custom 同名覆盖内置(就像 T-0178 的 `custom_overrides_builtin=true`)?

### D2. 自定义 XML 的合法根元素

内置 `<indicatorModel platform="..." deviceType="..." indicatorCount="N">`。自定义是否**强制要求**:

- `platform` 属性必填?
- `deviceType` 属性与上传时选的 `tech` 必须一致(防止用户传错制式)?

### D3. "重载 XML" 删孤儿的口径

按 updated_at < start 判定孤儿,会**误伤启用了但不在 XML 里的指标**(理论上不应存在,但运维历史可能有)。是否:

- (推荐)reload 前 dry-run 一次返孤儿清单 + 数量,用户在 Popconfirm 里确认
- (简化)直接删,沿用 T-0178 的"destructive 不可撤销"语义

### D4. 抽屉 vs Modal 给"单位定义"

- 抽屉(右侧 width=720)— **本文档采用**,空间充足适合 CRUD 列表
- 全屏 Modal — 适合编辑表单为主的场景
- 二级独立菜单 — 像 standard-params 那样把单位拉出来做菜单项;但单位是指标库的子资源,独立菜单语义弱

确认抽屉?

### D5. drill-down 详情是否要进一步"按 XML 文件分组折叠"?

- (本文档方案)详情就是平表 + 来源列,删除走"管理 XML 文件"Modal
- (备选)详情按 XML 文件分组,每组顶部直接显示文件名 + 来源 tag + 删除按钮;省一次 Modal

选 A 实现简单,选 B 操作路径短。倾向 A。

---

## 11. 后续(本次不做、记入 backlog)

- 指标公式(`rela_platform_indicator_formula_*`)的自定义 XML 上传 — 当前公式只能在 UI 单条编辑,未来可考虑统一上传
- 单位定义的 XML 化 — 当前单位是手动维护,未来如果要与 XML 同步,可加 `indicator-units.xml` 文件
- 三表合一(perf_indicators 单表带 `tech` 列)— 长期重构,跟分区 / 索引重设计一起做

---

## 12. 关联文档

- T-0178 PRD:[docs/project/prd/F02-param-model-custom-xml.md](docs/project/prd/F02-param-model-custom-xml.md)
- T-0178 后端实现说明:[omcgo/CLAUDE.md §5.3.1](omcgo/CLAUDE.md)(参数模型分层目录)
- 当前 kpi-library 实现:
  - [omcmb/webcode/src/pages/product/kpi-library/](omcmb/webcode/src/pages/product/kpi-library/)
  - [omcgo/internal/pm/indicator/](omcgo/internal/pm/indicator/)
- 参考样板:
  - [omcmb/webcode/src/pages/product/param-model/index.tsx](omcmb/webcode/src/pages/product/param-model/index.tsx)
  - [omcgo/internal/config/parammodel/upload.go](omcgo/internal/config/parammodel/upload.go)
  - [omcgo/internal/config/parammodel/source.go](omcgo/internal/config/parammodel/source.go)
