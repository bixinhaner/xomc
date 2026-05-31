# PRD: F06 数据字典 —— 字典项来源（自动同步）+ 工具栏精简

**PRD ID**：F06-data-dictionary-source
**功能域**：F06 OMC-R 核心 / 系统管理 / 数据字典
**对应页面**：`/system/data-dictionary`
**作者**：Claude（代 Owner = 产品经理 + 架构专家）
**创建**：2026-05-31
**状态**：v1 APPROVED（用户 2026-05-31 一句 "ok" 确认全部 8 个决策点采用推荐默认）
**Backlog**：T-0182

---

## 1. 背景

`/system/data-dictionary` 是字典管理页面，左侧字典列表 + 右侧字典项（支持 3 级嵌套）。当前所有字典项都靠管理员手工录入。

**痛点**：
- 部分字典本质是某张业务表的派生视图（如「设备型号」字典 ≈ `devices.product_class` 的去重列表、「区域」字典 ≈ `topology_nodes.name`），手工维护与真实数据脱节，存在长期漂移。
- 工具栏自带「实时刷新 / 列设置 / 密度」三个开关对管理员无价值（字典数据量小、字段固定、不需要实时），反而造成视觉噪声。

**本次目标**：
1. 把"工具栏三件套"在字典页面隐藏；
2. 让字典支持「绑定数据源」—— 选表 + 选字段，由系统自动同步字典项；
3. 提供"手动刷新"入口 + 每日一次的自动定时刷新。

---

## 2. 用户故事

| 角色 | 故事 |
|------|------|
| 系统管理员 | 我把「设备型号」字典绑定到 `devices` 表的 `product_class` 字段，之后无论新设备入网增加了什么型号，字典项都会在第二天凌晨自动出现，不需要我手工去同步。 |
| 系统管理员 | 我刚批量导入了一批新设备，不想等到明天 — 在字典列表里点字典名旁边的"刷新"按钮，立即触发该字典的同步。 |
| 系统管理员 | 我看到「设备型号」是数据源绑定的字典，因此页面上不显示"添加字典项"按钮、子项不允许新增 — 系统强制告诉我"这是托管字典，不能手工改"。 |
| 普通运维 | 我打开字典页面，工具栏干净清爽，没有不相关的"实时刷新 / 列设置 / 密度"按钮。 |

---

## 3. 详细需求

### 3.1 工具栏三件套移除（变更 1）

**范围**：仅 `/system/data-dictionary` 页面右侧"字典项"表格。左侧"字典列表"是手写 `<List>`，无 DataTable 工具栏，不动。

**移除**：
- 「开启实时刷新」按钮（SyncOutlined）
- 「列设置」按钮（ColumnVisibility）
- 「密度」切换（DensityToggle）

**保留**：
- 「刷新」按钮（ReloadOutlined）—— 仍需要，刷新数据本身
- 「导出」按钮（如已启用）

**实现方案**：
- `omcmb/webcode/src/components/DataTable/Toolbar.tsx` 新增 3 个独立 props：`hideRealtime?: boolean` / `hideColumnSettings?: boolean` / `hideDensity?: boolean`；
- `DataTable` 的 props 同步透传；
- 字典页面调用 `<DataTable>` 时全部传 `true`；
- 已有的 `hideToolbar` 粗粒度开关不动，避免影响其它页面。

---

### 3.2 字典数据源（变更 2 — 核心）

#### 3.2.1 数据模型

`sys_dictionaries` 表新增列：

```sql
-- 数据源绑定。null = 手工维护字典；非 null = 托管字典
source_table          VARCHAR(64),    -- 源表"业务名"（白名单见 §3.2.2）
source_label_field    VARCHAR(64),    -- 字段名 → 作为字典项的 label
source_value_field    VARCHAR(64),    -- 字段名 → 作为字典项的 value（可与 label 同字段）
source_filter         JSONB,          -- 预留：可选过滤条件（v1 不读不写）
last_refresh_at       TIMESTAMPTZ,    -- 最近一次成功同步时间
last_refresh_status   VARCHAR(16),    -- ok / failed / running
last_refresh_error    TEXT,           -- 失败时的错误摘要（≤ 500 char）
last_refresh_count    INT             -- 上次同步后字典项总数
```

`sys_dictionary_details` 表新增列：

```sql
origin                VARCHAR(16) NOT NULL DEFAULT 'manual'  -- manual | auto
```

字段语义：
- `origin = manual` —— 用户手工创建的项；
- `origin = auto` —— 来自数据源自动同步的项；
- 同步任务只删 / 增 / 改 `auto` 行，**不会动 `manual` 行**（方案 A — 兼容历史数据 + 允许临时补录）。

#### 3.2.2 可绑定的"表"白名单

后端硬编码一份 `internal/admin/data_dictionary_sources.yaml`（或 Go 常量），形如：

```yaml
- table: sys_users
  display_name: "用户"
  fields:
    - { column: id,        display: "用户ID",   type: string }
    - { column: username,  display: "用户名",   type: string }
    - { column: nickname,  display: "昵称",     type: string }
- table: devices
  display_name: "设备"
  fields:
    - { column: serial_number,  display: "序列号",     type: string }
    - { column: product_class,  display: "设备型号",   type: string }
    - { column: vendor_id,      display: "厂商 OUI",   type: string }
- table: topology_nodes
  display_name: "拓扑节点"
  fields:
    - { column: name,       display: "节点名",    type: string }
    - { column: node_type,  display: "节点类型",  type: string }
# 其它白名单（v1 暂定 8-12 张表，启动 S2 设计阶段定稿）
```

**v1 白名单候选**：由 Claude 基于 `internal/admin / internal/device / internal/topology / internal/alarm / internal/product` 的 PG 表盘点，给出 ≤ 12 张候选 → S2 设计阶段定稿。

#### 3.2.3 新增 / 编辑字典 Modal 字段调整

在现有 `name / type / status / desc` 之外，新增折叠区域「数据源（可选）」：

```
┌──────────────────────────────────────┐
│ 字典名称*  [_______________]         │
│ 字典类型*  [_______________]         │
│ 状态       [ ●开启  ○关闭 ]          │
│ 描述       [_______________]         │
│                                      │
│ ▼ 数据源（可选）                     │
│   来源表    [选择...    ▼]           │  ← 白名单下拉
│   Label 字段 [选择...    ▼]          │  ← 依赖来源表
│   Value 字段 [选择...    ▼]          │  ← 依赖来源表，允许与 Label 相同
│   [测试 → 预览前 10 条]              │  ← dry-run 端点
└──────────────────────────────────────┘
```

**校验**：
- 三个字段「同时为空」或「同时填写」，禁止部分填写；
- 编辑字典时，把 `source_*` 从有值改为空 → 后端清空 `auto` 项，保留 `manual` 项；
- 编辑字典时，切换 `source_table` → 弹确认框「将清除所有 auto 字典项，是否继续？」
- Label 与 Value 允许选同一字段（最常见场景：`product_class` 同时作 label/value）。

#### 3.2.4 手动添加 / 编辑字典项的禁用规则

当 `selectedDict.source_table != null` 时：
- 隐藏右侧表头的「+ 添加详情」按钮；
- 行内「+ 添加子项」按钮隐藏；
- `auto` 行的「编辑」「删除」按钮禁用 + tooltip "托管字典项不允许修改"；
- `manual` 行的「编辑」「删除」按钮保留，允许用户继续清理历史数据；
- 列表多一列「来源」Tag：`auto` 显示「自动同步」、`manual` 显示「手工」。

#### 3.2.5 层级（parent_id）支持

**托管字典不支持层级**。数据源是平铺的表字段，无法表达父子关系；如果某字典需要层级就别绑数据源。同步函数写入 `auto` 行时强制 `parent_id = NULL` / `level = 0`。手工字典 (`source_table IS NULL`) 仍维持现有 3 级嵌套能力。

---

### 3.3 手动刷新按钮（变更 2 续）

**位置**：左侧字典列表，每一行字典名右侧操作图标区，**"编辑"按钮之前**：

```
设备型号  [🔄 刷新] [✏️ 编辑] [🗑️ 删除]
```

**显示规则**：
- 仅当 `source_table != null` 时显示；手工字典不显示该按钮；
- 点击后调用 `POST /admin/sysDictionary/refreshSource?id={id}` 同步；
- 同步期间按钮变 loading 状态，禁用其它操作；
- 同步结束 toast：「✓ 已同步：新增 X / 删除 Y / 总计 Z 项」或「✗ 同步失败：{error}」；
- `last_refresh_at` 在字典 Card 副标题以"上次同步：5 分钟前"形式显示（用 dayjs.fromNow）。

---

### 3.4 每日定时刷新（变更 3）

**调度**：
- Cron 表达式：`0 0 2 * * *`（每日 02:00:00 北京时间，业务低峰）；
- 注册位置：`cmd/worker/main.go` 已有 `robfig/cron/v3` 调度器，加一个新任务；
- 任务名：`dictionary_source_refresh_daily`。

**执行逻辑**：
```
1. SELECT * FROM sys_dictionaries WHERE source_table IS NOT NULL AND status = true;
2. 顺序遍历（不并发，避免 N+1 重负载）；
3. 对每张字典：调用同 §3.3 的同步函数（手动刷新与定时刷新走同一份核心代码）；
4. 单张失败 → 记日志 + 写 last_refresh_error，不中断后续；
5. 全部结束 → 一条 INFO 日志 `daily dict refresh: total=X, ok=Y, failed=Z`；
6. 失败时通过 EventBus 发出 `dictionary.refresh.failed` 事件（预留给告警/通知）。
```

**同步函数核心算法**（手动 + 定时共用）：

```
target_values = SELECT DISTINCT {label_field}, {value_field}
                FROM {source_table}
                WHERE deleted_at IS NULL          -- 兼容软删；无该列则跳过 WHERE
                LIMIT 5000;                       -- 上限保护
existing_auto = SELECT id, label, value FROM sys_dictionary_details
                WHERE sys_dictionary_id = ? AND origin = 'auto';

diff:
  to_insert = target_values - existing_auto
  to_update = labels 变化的（value 相同 / label 不同）
  to_delete = existing_auto - target_values

事务内：
  INSERT 新增（status=true, origin='auto', sort=0, parent_id=NULL）
  UPDATE label 变化的
  DELETE 被删除的

last_refresh_at     = now()
last_refresh_status = 'ok' / 'failed'
last_refresh_count  = COUNT(*) WHERE sys_dictionary_id = ?
```

**单字典超时保护**：5 秒（context.WithTimeout），超时跳过、记 failed，定时刷新不被单字典拖垮。

---

### 3.5 API 设计

| 方法 | 路径 | 用途 |
|------|------|------|
| `GET` | `/admin/sysDictionary/sources` | 列出白名单表 + 字段（前端 Modal 下拉的数据） |
| `GET` | `/admin/sysDictionary/sources/preview?table=&label=&value=&limit=10` | "测试"按钮 — dry-run 前 10 条 |
| `POST` | `/admin/sysDictionary/refreshSource?id={id}` | 手动触发单字典同步 |
| `POST` | `/admin/sysDictionary/createSysDictionary` | **复用** —— payload 新增 `source_table / source_label_field / source_value_field` |
| `PUT` | `/admin/sysDictionary/updateSysDictionary` | **复用** —— 同上 |
| `GET` | `/admin/sysDictionary/getSysDictionaryList` | **复用** —— 响应新增 `source_*` 与 `last_refresh_*` 字段 |
| `POST` | `/admin/sysDictionaryDetail/createSysDictionaryDetail` | **复用** —— 后端校验：source 字典只允许 `origin='manual'` 项；`auto` 不能由前端写入 |

---

### 3.6 i18n 新增 key

`dictionary.source.title` / `dictionary.source.table` / `dictionary.source.labelField` / `dictionary.source.valueField` / `dictionary.source.preview` / `dictionary.source.previewTitle` / `dictionary.source.refreshBtn` / `dictionary.source.refreshSuccess` / `dictionary.source.refreshFailed` / `dictionary.source.refreshing` / `dictionary.source.lastRefreshAt` / `dictionary.source.managedTip` / `dictionary.source.switchTableConfirm` / `dictionary.source.clearTip` / `dictionary.origin.auto` / `dictionary.origin.manual`

---

## 4. 数据库迁移

| Migration 编号 | 内容 |
|---|---|
| 000NNN_dict_source.sql | `ALTER TABLE sys_dictionaries ADD COLUMN source_table / source_label_field / source_value_field / source_filter / last_refresh_at / last_refresh_status / last_refresh_error / last_refresh_count;` `ALTER TABLE sys_dictionary_details ADD COLUMN origin VARCHAR(16) NOT NULL DEFAULT 'manual';` 索引：`(source_table) WHERE source_table IS NOT NULL`、`(sys_dictionary_id, origin)`；S2 启动时取实际下一个 migration 编号 |

**无数据回填需求**（DDL only）。

---

## 5. 非目标（v1 不做）

- ❌ `source_filter` JSONB（WHERE 条件构造器）—— 字段已建，但 v1 不读不写，留给 v2；
- ❌ 跨表 JOIN 作为数据源；
- ❌ 数据源支持自定义 SQL；
- ❌ 同步频率可配置（v1 固定每日 02:00）；
- ❌ 同步失败告警邮件 / SMS（v1 只记日志 + 事件，告警通道接入留给 F04）；
- ❌ 历史 `manual` 项与新 `auto` 项 label 冲突时的合并策略 —— v1 把 `auto` 当独立集合，不去重 `manual`；
- ❌ 托管字典支持层级（parent_id）。

---

## 6. 依赖项

| 依赖 | 状态 |
|------|------|
| `cmd/worker` cron scheduler | ✅ 已有，加 1 个 job |
| `internal/admin` 模块 handler/service/repo | ✅ 已有 |
| `frontend-core/services/api/adminApi.ts` 字典 API | ✅ 已有，扩展 payload |
| Carrier 适配 | ❌ 无运营商差异 |
| 告警通知 | ❌ 不依赖 |

---

## 7. 风险

| 风险 | 缓解 |
|------|------|
| 白名单表的字段值数量爆炸（如 device.serial_number 数万行）→ 字典页面无法展开 | UI 给 source 字典加分页（默认 50 条/页）+ 同步函数硬上限 `LIMIT 5000` + 超限日志告警 |
| Daily job 在大数据量下耗时长 | 顺序执行 + 每张字典 5 秒超时；超时跳过、记 failed |
| 用户绑错表，导致 `auto` 项瞬间灌大量噪声 | 切换 `source_table` 时弹确认；提供"测试 → 预览前 10 条" dry-run |
| 白名单需要持续维护 | 文档化注册流程；新表想成为字典源走 PR 流程 + S2 设计稿评审 |
| 跨皮肤影响 | webcode-v2/v3 暂未实现该页面，frontend-core 类型扩展是兼容性的（新字段全可选） |

---

## 8. 验收标准（Given/When/Then）

**AC-1（工具栏精简）**
- Given 我在 `/system/data-dictionary` 右侧字典项表格
- Then 工具栏不显示「实时刷新」「列设置」「密度」三个按钮
- And 「刷新」按钮仍然存在并可用

**AC-2（新增带数据源字典）**
- Given 我打开「新增字典」对话框
- When 我填入名称/类型 + 选择来源表 `devices` + label 字段 `product_class` + value 字段 `product_class`
- And 点击保存
- Then 字典创建成功
- And 后端立即触发一次同步
- And 字典项列表里出现 `devices.product_class` 的 distinct 值，且每行 origin = `auto`

**AC-3（手动刷新）**
- Given 字典「设备型号」已绑定 `devices.product_class`
- When 我在左侧列表点击「🔄 刷新」
- Then 字典项与表内 distinct 值对齐
- And toast 提示「✓ 已同步：新增 N / 删除 M / 总计 Z 项」
- And 字典副标题更新为「上次同步：刚刚」

**AC-4（手工添加禁用）**
- Given 字典「设备型号」是 source 字典
- Then 右侧"+ 添加详情"按钮隐藏
- And 行内"+ 添加子项"按钮隐藏
- And `auto` 行的编辑/删除按钮禁用并 tooltip 提示

**AC-5（手工字典不变）**
- Given 字典「业务区域」未绑定数据源
- Then 所有现有功能（增/删/改字典项，含 3 级嵌套）保持原样

**AC-6（每日刷新）**
- Given 系统时间到达每日 02:00
- When worker 进程在运行
- Then 所有 `source_table IS NOT NULL AND status = true` 的字典全部被同步
- And 失败的字典不影响其它字典
- And 日志输出 `daily dict refresh: total=X, ok=Y, failed=Z`

**AC-7（绑定 → 解绑）**
- Given 字典已绑定数据源
- When 我编辑字典并把数据源置空保存
- Then 该字典所有 `auto` 项被删除
- And `manual` 项保留
- And 「+ 添加详情」按钮重新出现

**AC-8（dry-run 预览）**
- Given 我在「新增字典」对话框选择了来源表 + label + value 字段
- When 点击「测试 → 预览前 10 条」
- Then 弹窗显示前 10 条 `(label, value)` 元组
- And 显示总行数（如「共 247 条匹配，仅展示前 10 条」）

---

## 9. 度量

- 上线后第 7 天，`source_table IS NOT NULL` 的字典数 / 全部字典数 ≥ 30%
- 每日 02:00 任务平均耗时 < 10 秒（10 张字典内）
- 一周内同步失败率 < 1%
- 用户手动刷新点击数 / 字典数：上线后第 1 周可观察"用户对自动同步的信任度"

---

## 10. 跨皮肤影响

- `frontend-core/services/api/adminApi.ts` 扩展 `Dictionary` 类型 + payload 字段 → 新增字段全为可选，对 webcode-v2/v3 类型兼容；
- 不新增 `frontend-core/hooks/api` 文件（继续直接在页面调 API + React Query in-page）；
- webcode-v2/v3 暂未实现该页面，本次不动它们；
- `DataTable.Toolbar` 新增 3 个可选 props，对所有调用方零影响（默认值保持现行为）。

---

## 11. 实施分阶段

| 阶段 | 范围 | 估时 |
|------|------|------|
| P1 — Schema + 后端骨架 | DB migration + sources 白名单 + 三个新端点 + 同步函数 | 1d |
| P2 — Cron 集成 + handler/service 单测 | worker 注册 cron + sync engine 5+ table-driven 测试 | 0.5d |
| P3 — 前端 Modal + 右侧禁用规则 + 手动刷新按钮 | adminApi 扩展 + Modal 三字段 + dry-run 预览 + 列表禁用规则 + 手动刷新 | 1.5d |
| P4 — 工具栏精简 + i18n + E2E | DataTable 3 props + 字典页 truthy 传入 + i18n × 2 lang + e2e_verify.sh 加 3 用例 | 0.5d |
| P5 — 真机/手动验证 + DoD | 8 项 AC 逐项跑过 + PRD §11 自检 | 0.5d |

**总估时**：4d（Est=M），单人。

---

## 12. PRD 决策锁定

| # | 决策点 | 选项 | 状态 |
|---|--------|------|------|
| 1 | 工具栏三件套仅改右侧字典项表格 | 是 | ✅ 锁定 |
| 2 | 绑定数据源后允许保留 `manual` 项（方案 A） | A | ✅ 锁定 |
| 3 | 白名单 + 由 Claude 整理候选表清单 | 是 | ✅ 锁定（S2 阶段定稿） |
| 4 | Label 与 Value 允许选同一字段 | 允许 | ✅ 锁定 |
| 5 | 加 dry-run 预览前 10 条按钮 | 加 | ✅ 锁定 |
| 6 | Cron 时间 02:00 BJT | 02:00 | ✅ 锁定 |
| 7 | 托管字典不支持层级 | 不支持 | ✅ 锁定 |
| 8 | v1 不含 `source_filter` WHERE 条件 | 不含 | ✅ 锁定 |

---

## 13. 附录 A — S2 设计备忘（2026-05-31 落稿）

> 阶段：dev-pipeline S2 design。激活专家：架构（§16.1）+ Go 工程（§16.2）+ 数据与存储（§16.5）+ 前端（§16.6）+ 安全合规（§16.8）。
> **待定点**：① 白名单候选最终收敛（拟定 12 张，请在 S3 启动前用户确认到 6-10 张）；② cron 失败通知通道（v1 倾向 EventBus 先走起，监控接入由 F04 后续承接）。

### A.1 白名单候选（12 张候选表，请用户收敛）

基于 `omcgo/migrations/000001_init_schema.sql` 真实表盘点。前缀 ★ 是强烈推荐保留（业务字典最常见来源）。

```yaml
# omcgo/internal/admin/data_dictionary_sources.yaml
sources:
  ★devices:
    display_name: "设备"
    fields:
      - { column: serial_number,  display: "序列号",     type: string }
      - { column: product_class,  display: "设备型号",   type: string }
      - { column: vendor_id,      display: "厂商 OUI",   type: string }
      - { column: software_version, display: "软件版本", type: string }
  ★products:
    display_name: "产品"
    fields:
      - { column: code,           display: "产品 code",  type: string }
      - { column: name,           display: "产品名",     type: string }
      - { column: vendor,         display: "厂商",       type: string }
  ★param_models:
    display_name: "参数模型"
    fields:
      - { column: name,           display: "模型名",     type: string }
  ★topo_nodes:
    display_name: "拓扑节点"
    fields:
      - { column: name,           display: "节点名",     type: string }
      - { column: node_type,      display: "节点类型",   type: string }
  ★alarm_definitions:
    display_name: "告警定义"
    fields:
      - { column: identifier,     display: "告警标识",   type: string }
      - { column: ne_type,        display: "网元类型",   type: string }
      - { column: severity,       display: "严重级",     type: string }
  ★device_groups:
    display_name: "设备组"
    fields:
      - { column: name,           display: "组名",       type: string }
      - { column: group_type,     display: "组类型",     type: string }
  ★sites:
    display_name: "站点"
    fields:
      - { column: name,           display: "站点名",     type: string }
      - { column: region,         display: "区域",       type: string }
  product_class_patterns:
    display_name: "产品 productClass pattern"
    fields:
      - { column: pattern,        display: "正则",       type: string }
  firmware_versions:
    display_name: "固件版本"
    fields:
      - { column: version,        display: "版本号",     type: string }
      - { column: vendor,         display: "厂商",       type: string }
  kpi_definitions:
    display_name: "KPI 定义"
    fields:
      - { column: code,           display: "KPI code",   type: string }
      - { column: name,           display: "KPI 名",     type: string }
  sys_users:
    display_name: "用户"
    fields:
      - { column: username,       display: "用户名",     type: string }
      - { column: nickname,       display: "昵称",       type: string }
  roles:
    display_name: "角色"
    fields:
      - { column: code,           display: "角色 code",  type: string }
      - { column: name,           display: "角色名",     type: string }
```

**安全约束（§16.8 审查通过线）**：
- 表名 / 字段名硬编码白名单，**不接受**任何前端传入的 raw 表名 / 字段名 → 杜绝 SQL 注入；
- 字典同步的 SQL 通过 `text/template` 把 `{table}` `{label}` `{value}` 插值，但插值前必须在白名单 `Resolve(name) (cleanName string, ok bool)` 函数内对照过；
- 敏感表（`sys_login_logs`、`users.password_hash`、`api_keys`、`audit_logs`、`system_license`）**绝不**进白名单 — 即使后续业务想要也走 PR 审批。

### A.2 后端接口签名

#### A.2.1 包结构（新增 1 个子包）

```
internal/admin/
├── dictionary_handler.go     (已有，扩展 4 个 handler 方法)
├── dictionary_service.go     (已有，扩展 4 个 service 方法)
├── pg_dictionary_repository.go (已有，扩展 5 个 repo 方法)
├── dictionary_model.go        (已有，扩展 model 字段)
└── dictsource/                ← 新增子包
    ├── sources.go             (白名单加载与 Resolve)
    ├── sources.yaml           (白名单数据)
    ├── engine.go              (同步引擎核心算法)
    ├── engine_test.go         (5+ table-driven 测试)
    └── cron.go                (daily refresh entry，由 cmd/worker 调用)
```

#### A.2.2 关键签名

```go
// dictsource/sources.go
type FieldSpec struct {
    Column      string `yaml:"column"`
    DisplayName string `yaml:"display"`
    Type        string `yaml:"type"`
}
type TableSpec struct {
    Table       string      `yaml:"table"`
    DisplayName string      `yaml:"display_name"`
    Fields      []FieldSpec `yaml:"fields"`
}
type Registry struct { tables map[string]TableSpec }

func Load(yamlPath string) (*Registry, error)
func (r *Registry) ListTables() []TableSpec
func (r *Registry) Resolve(table string) (TableSpec, bool)
func (r *Registry) ResolveField(table, column string) (FieldSpec, bool)

// dictsource/engine.go
type SyncResult struct {
    Inserted  int
    Updated   int
    Deleted   int
    TotalAuto int
    Err       error
}
type SyncEngine struct {
    pool     *pgxpool.Pool
    registry *Registry
    log      *zap.Logger
}
func NewSyncEngine(pool *pgxpool.Pool, registry *Registry, log *zap.Logger) *SyncEngine
func (e *SyncEngine) SyncOne(ctx context.Context, dictID int64) SyncResult
func (e *SyncEngine) SyncAll(ctx context.Context) (totalOK, totalFailed int)
func (e *SyncEngine) Preview(ctx context.Context, table, label, value string, limit int) ([]PreviewRow, int, error)

// dictsource/cron.go
func RegisterDailyCron(c *cron.Cron, engine *SyncEngine, log *zap.Logger) (cron.EntryID, error)
// 内部：spec="0 0 2 * * *"; func 体调 SyncAll；ctx WithTimeout 5min（10 张内）；总耗时打日志
```

#### A.2.3 Handler 扩展（4 个新方法 + 3 个改 payload）

```go
// dictionary_handler.go 新增
func (h *DictionaryHandler) ListSources(c *gin.Context)              // GET /admin/sysDictionary/sources
func (h *DictionaryHandler) PreviewSource(c *gin.Context)            // GET /admin/sysDictionary/sources/preview
func (h *DictionaryHandler) RefreshSource(c *gin.Context)            // POST /admin/sysDictionary/refreshSource

// 现有 Create/Update/CreateDetail 改 payload 校验：
//   - CreateDictionary: 校验 source_* 三字段同时为空或同时填写；同时填时 Resolve 校验通过；保存后立即触发一次 SyncOne
//   - UpdateDictionary: 同上 + 老→新 source_table 切换时清空旧 auto 项
//   - CreateDictionaryDetail: source_table != null 时拒绝创建 origin='auto' 项；manual 项允许
```

### A.3 数据库迁移草案

`omcgo/migrations/000NNN_dict_source.sql`（NNN 待 S3 启动前 `bash omcgo/scripts/check-migrations.sh` 取下一个连续编号）：

```sql
-- +goose Up
ALTER TABLE sys_dictionaries
    ADD COLUMN source_table          VARCHAR(64),
    ADD COLUMN source_label_field    VARCHAR(64),
    ADD COLUMN source_value_field    VARCHAR(64),
    ADD COLUMN source_filter         JSONB,
    ADD COLUMN last_refresh_at       TIMESTAMPTZ,
    ADD COLUMN last_refresh_status   VARCHAR(16),
    ADD COLUMN last_refresh_error    TEXT,
    ADD COLUMN last_refresh_count    INT;

ALTER TABLE sys_dictionary_details
    ADD COLUMN origin VARCHAR(16) NOT NULL DEFAULT 'manual';

CREATE INDEX idx_sys_dict_source_table
    ON sys_dictionaries (source_table)
    WHERE source_table IS NOT NULL;

CREATE INDEX idx_sys_dict_detail_dictid_origin
    ON sys_dictionary_details (sys_dictionary_id, origin);

-- 软约束（手工字典 origin 必须 manual；source 字典写入由后端 service 守门，DB 不强制以兼容现有数据）：
-- 不加 CHECK 约束（v1 让 service 层兜底，避免历史数据破坏迁移）

-- +goose Down
DROP INDEX IF EXISTS idx_sys_dict_detail_dictid_origin;
DROP INDEX IF EXISTS idx_sys_dict_source_table;
ALTER TABLE sys_dictionary_details DROP COLUMN IF EXISTS origin;
ALTER TABLE sys_dictionaries
    DROP COLUMN IF EXISTS last_refresh_count,
    DROP COLUMN IF EXISTS last_refresh_error,
    DROP COLUMN IF EXISTS last_refresh_status,
    DROP COLUMN IF EXISTS last_refresh_at,
    DROP COLUMN IF EXISTS source_filter,
    DROP COLUMN IF EXISTS source_value_field,
    DROP COLUMN IF EXISTS source_label_field,
    DROP COLUMN IF EXISTS source_table;
```

### A.4 同步函数核心 SQL（pseudo）

```go
// dictsource/engine.go::SyncOne 关键 SQL（由 squirrel 构建 + pgx 执行；表/字段名经 Registry.Resolve 校验后才插值）
func (e *SyncEngine) syncCore(ctx context.Context, dict Dictionary) (SyncResult, error) {
    spec, ok := e.registry.Resolve(dict.SourceTable)
    if !ok { return SyncResult{}, ErrInvalidSourceTable }
    if _, ok := e.registry.ResolveField(dict.SourceTable, dict.SourceLabelField); !ok { return SyncResult{}, ErrInvalidLabelField }
    if _, ok := e.registry.ResolveField(dict.SourceTable, dict.SourceValueField); !ok { return SyncResult{}, ErrInvalidValueField }

    // 1. 拉源数据（白名单已校验，pgx 不接受表名占位符 → 直接拼字符串安全）
    sourceSQL := fmt.Sprintf(
        `SELECT DISTINCT %s::TEXT AS label, %s::TEXT AS value
         FROM %s
         WHERE %s IS NOT NULL
         LIMIT 5000`,
        dict.SourceLabelField, dict.SourceValueField, spec.Table, dict.SourceValueField,
    )
    targetRows, err := e.queryTarget(ctx, sourceSQL)

    // 2. 拉现有 auto 项
    existingAuto, err := e.queryExistingAuto(ctx, dict.ID)

    // 3. diff
    toInsert, toUpdate, toDelete := diff(targetRows, existingAuto)

    // 4. 事务内写
    tx, _ := e.pool.BeginTx(ctx, pgx.TxOptions{})
    defer tx.Rollback(ctx)
    e.batchInsert(ctx, tx, dict.ID, toInsert)   // status=true, origin='auto', sort=0, parent_id=NULL
    e.batchUpdate(ctx, tx, toUpdate)
    e.batchDelete(ctx, tx, toDelete)
    e.updateMetadata(ctx, tx, dict.ID, "ok", "", len(targetRows))
    tx.Commit(ctx)

    return SyncResult{Inserted: len(toInsert), Updated: len(toUpdate), Deleted: len(toDelete), TotalAuto: len(targetRows)}, nil
}
```

**5 秒超时**：`ctx, cancel := context.WithTimeout(parent, 5*time.Second); defer cancel()` 包在 `SyncOne` 入口；daily SyncAll 整体 5 分钟超时，单字典超时不中断整体。

### A.5 前端组件签名

#### A.5.1 `DataTable.Toolbar` 扩展

```ts
// omcmb/webcode/src/components/DataTable/Toolbar.tsx
interface ToolbarProps {
  // ... 现有 props 保持不变
  hideRealtime?: boolean;        // 默认 false，true 时隐藏「实时刷新」
  hideColumnSettings?: boolean;  // 默认 false，true 时隐藏「列设置」
  hideDensity?: boolean;         // 默认 false，true 时隐藏「密度」
}
// 同步透传到 DataTable 主组件 props
```

#### A.5.2 SourcePicker 新组件（字典 Modal 内嵌）

```ts
// omcmb/webcode/src/pages/system/DataDictionary/SourcePicker.tsx
interface SourcePickerValue {
  sourceTable?: string;
  sourceLabelField?: string;
  sourceValueField?: string;
}
interface SourcePickerProps {
  value?: SourcePickerValue;
  onChange?: (v: SourcePickerValue) => void;
  disabled?: boolean;            // 编辑现有 source 字典 + 切换源表时 disabled=false 允许改
}
// 内部：
//   - useQuery(['dict-sources'], adminApi.listDictionarySources)
//   - <Form.Item> 三个 Select 联动（label/value 选项依赖 table）
//   - 「测试 → 预览前 10 条」按钮 → adminApi.previewDictionarySource → <Modal> 展示 Table
```

#### A.5.3 frontend-core API 扩展

```ts
// omcmb/frontend-core/src/services/api/adminApi.ts 新增
interface DictionarySource {
  table: string;
  displayName: string;
  fields: Array<{ column: string; display: string; type: string }>;
}
interface DictionaryPreviewRow {
  label: string;
  value: string;
}

adminApi.listDictionarySources():                                          Promise<DictionarySource[]>
adminApi.previewDictionarySource(table, label, value, limit?):             Promise<{ rows: DictionaryPreviewRow[]; total: number }>
adminApi.refreshDictionarySource(id):                                      Promise<{ inserted: number; updated: number; deleted: number; total: number }>

// 现有类型扩展（新字段全可选，对 webcode-v2/v3 兼容）
interface Dictionary {
  // ...
  sourceTable?: string;
  sourceLabelField?: string;
  sourceValueField?: string;
  lastRefreshAt?: string;
  lastRefreshStatus?: 'ok' | 'failed' | 'running';
  lastRefreshError?: string;
  lastRefreshCount?: number;
}
interface DictionaryDetail {
  // ...
  origin: 'manual' | 'auto';
}
```

### A.6 观测埋点

| 类型 | 名称 | 标签 | 说明 |
|------|------|------|------|
| Metric | `dictionary_source_refresh_total` | `dict_id` `result=ok\|failed\|timeout` | 同步任务次数计数 |
| Metric | `dictionary_source_refresh_duration_seconds` | `dict_id` | 单字典同步耗时直方图 |
| Metric | `dictionary_source_auto_items_total` | `dict_id` | 每张字典当前 auto 项总数（gauge，同步成功后更新） |
| Log key | `dict_source_sync_started` | dict_id, table, label, value | INFO，每次同步开始 |
| Log key | `dict_source_sync_completed` | dict_id, inserted, updated, deleted, total, duration_ms | INFO，每次同步成功 |
| Log key | `dict_source_sync_failed` | dict_id, error | WARN，单字典同步失败（不中断其它） |
| Log key | `dict_source_daily_summary` | total, ok, failed, duration_ms | INFO，daily cron 整体结束 |

### A.7 跨皮肤影响盘点

| 文件 / 模块 | 影响 |
|------------|------|
| `frontend-core/services/api/adminApi.ts` | 新增 3 个方法 + 类型字段扩展（全可选）→ v2/v3 类型兼容 |
| `frontend-core/types/` | `Dictionary` / `DictionaryDetail` 新增可选字段 → 类型兼容 |
| `frontend-core/mock/` | 加 mock data 含 source 字段（默认 undefined）→ Mock 模式无影响 |
| `webcode/components/DataTable/Toolbar.tsx` | 新增 3 个可选 props，默认 false → 其它页面零行为变化 |
| `webcode-v2 / v3` | 不实现 `/system/data-dictionary` 页面 → 本期不动 |

### A.8 S2 出口门核查

- [✓] 接口契约明确（A.2 / A.5）
- [✓] 迁移草案 up/down 配对（A.3）
- [✓] Carrier 差异点列出（无差异，PRD §6 已声明）
- [✓] 观测埋点名字列出（A.6）
- [✓] 待定点 < 3（仅 2 个：A.1 白名单收敛 + A.0 cron 失败通道）

**S2 PASS**。下一步：S3 implement，按 PRD §11 P1 → P5 分阶段实施。
