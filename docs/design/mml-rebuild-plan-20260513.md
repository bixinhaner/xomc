# MML 命令树重建方案（基于 TR-069 标准 path）

**Date**: 2026-05-13
**Author**: Claude
**Status**: ✅ APPROVED v3.1 — implementation-ready（2026-05-13，17/17 决议全签）
**Source**: `omcgo/data/param-mappings/standard-model.xml`（1988 params + 13 objects）

> v2 更新（2026-05-13 首轮 review）：10 个决议已确认，详 §11.1
> v3 更新（2026-05-13 二轮 review §6）：4 个新决议已确认，详 §11.2
>
> **v3 核心变化**：撤回 v2 的 mml_commands.rpc_steps JSONB 设计；
> **mml_param_groups 升级为"批量 RPC 执行单元"**（选一个 group 即跑该 group 下所有 commands 各自的 RPC），
> 这条把 RPC 编排责任从单条命令内嵌拨回到 group/script 层 — 与 fanout 现有 commands 数组语义天然吻合。
>
> **v3 浮出 7 个新待审项**（详 §11.3），主要涉及 fanout 改造范围 / group 批量执行 API / 脚本 parser 容错策略。
>
> **工作量重估**：4.6d (v2) → ~7d (v3 含 fanout 改造)。建议拆 2 sprint。

> 本文目标：废弃当前 `mml_*` 表里旧的、与产品型号绑死、靠手工 seed 的命令数据，
> 改为以 TR-069 标准模型 XML 为唯一权威源，由启动期 Loader 幂等生成命令树。
> 本文不实施，仅做**方案 + 现状 schema 审查 + 优化建议**，供 review 后再开干。

---

## 1. 目标

| 目标 | 描述 |
|------|------|
| **G1 单一权威源** | 命令树数据 100% 来自 `standard-model.xml` 解析，不再依赖 24 种硬编码 product 版本 |
| **G2 启动期幂等加载** | 对齐 T-0098 dictloader 模式；XML 改 → 重启即生效；二次跑 0 副作用 |
| **G3 schema 瘦身** | 借机消除现存表结构里的冗余 / 散布 / 概念混杂 |
| **G4 用户数据不破坏** | `mml_custom_command` / `mml_scripts` / `mml_tasks` / `mml_audit_log` 完全保留 |
| **G5 FE 命令树命中** | 重建后 FE `mml/Console` 命令树 11 中文 category 渲染正常 |

---

## 2. 现状 schema 全景（11 张 mml_ 表）

### 2.1 表清单 + 角色

| 层 | 表 | 角色 | 字段数 | DDL 来源 |
|---|---|------|------|---------|
| **dictionary 层** | `mml_param_versions` | 软件版本目录 | 12 | mig 022 |
| | `mml_param_groups` | 命令分组层级（ltree）| 25 | mig 022 |
| | `mml_params` | 叶子参数定义 | 31 | mig 022 |
| | `mml_group_param_rel` | group↔param N:M | 7 | mig 022 |
| **command 层** | `mml_commands` | 命令目录（FE 树） | 9 + 5 ALTER | mig 007 + seed/004 |
| | `mml_command_params_rel` | command↔param N:M | 3 | mig 026 |
| | `mml_custom_command` | 用户自定义命令 | ~12 | mig 014 (renamed) |
| **runtime 层** | `mml_scripts` | 脚本模板 + 执行态混 | ~14 | mig 007 + 026 |
| | `mml_tasks` | 任务执行实例 | ~22 | mig 007 + seed/004 |
| | `mml_audit_log` | MML 操作审计 | ? | mig 014 |
| **历史遗留** | `mml_sub_commands` | （已 DROP）| — | mig 025→026 |
| | `mml_command_subcommand_rel` | （已 DROP）| — | mig 025→026 |

### 2.2 数据流（重建前）

```
24 硬编码 product version (mig 022 INSERT body)
    │
    ▼
mml_param_versions   ──FK──>   mml_param_groups   ──N:M──   mml_params
                                     │                          ▲
                                     │                          │
                                     └────FK 上下层──────────────┘
                                               │
                                               └─ mml_group_param_rel (N:M 旁支)

seed/006 手工 INSERT 25 命令
    │
    ▼
mml_commands  ──N:M──>  mml_params
              ──category──>  sys_dictionaries(mml_command_category)
```

---

## 3. Schema 审查 — 优化空间（按问题分级）

> 下文标 ⚠️ = **建议本次重建一并修**；标 💡 = 长期债务，可后续单独处理

### 3.1 ⚠️ A. `mml_param_groups`：parent_id + ltree path 双轨冗余

```sql
parent_id  UUID REFERENCES mml_param_groups(id)
path       LTREE                        -- 这俩等价
level      INT                          -- 也是 path 派生量
```

**问题**：维护三处数据一致性靠应用层；插入新 group 易漏一处导致 query 错位。

**建议**：保留 `path LTREE` 一项（自带 GIST 索引 + 自然递归）；删 `parent_id` 与 `level`。
LTREE 的 `nlevel(path)` 即 level；`subpath` / `subltree` 替代 parent 查询。

### 3.2 ⚠️ B. 操作 flag 散布两层（params + groups）

`mml_params` 与 `mml_param_groups` **同时** 持有：
- `is_listable`
- `is_modifiable`
- `is_addable`
- `is_removable`

**问题**：操作权限是**命令**的属性，不是**参数**的属性。1 个参数可能既出现在 LST 命令也出现在 MOD 命令，把这 4 个 flag 钉在参数上语义混乱。
**建议**：
- `mml_params` 仅留 **`is_writable`**（这是 TR-069 spec 定义的参数自身属性）
- 4 个操作 flag 全部移到 `mml_commands.operation_type` 表达（每条命令 1 个 operation_type，自然分离）
- `mml_param_groups` 同样去掉这 4 个 flag

### 3.3 ⚠️ C. i18n 字段内联 8 处

| 表 | _zh / _en 字段 |
|----|---------------|
| `mml_param_groups` | group_name_zh, group_name_en, confirm_message_zh, confirm_message_en |
| `mml_params` | param_name_zh, param_name_en, explanation_zh, explanation_en, title_zh, title_en, confirm_message_zh, confirm_message_en |

**问题**：固化双语；要加日语/俄语全表 ALTER；多语言文本占用主表行 size。
**建议**：抽 `mml_i18n` 表 `(target_table, target_id, locale, key, value)`；或主表用 `JSONB name_i18n {"zh":"...", "en":"..."}`。本次重建建议**先用 JSONB**（轻量过渡）。

### 3.4 ⚠️ D. product / platform / version 三处独立配置（多源真相）

| 字段 | 表 | 类型 |
|------|---|------|
| `product_models` | `mml_param_versions` | VARCHAR(200)[] |
| `platform_support` | `mml_param_groups` | VARCHAR(10)[] |
| `mobile_support` / `broadband_support` | `mml_param_groups` | BOOLEAN |
| `platform_support` / `mobile_support` / `broadband_support` | `mml_params` | 同上 |
| `product_types` | `mml_commands` | JSONB |
| ~~`product_types`~~ | ~~`mml_custom_command`~~ | T-0090-b 已删 |

**问题**：5+ 处冗余字段，含义重叠（platform_support 是 4G/5G？是 LTE/NR？是公网/专网？文档不清）。
**建议**：
- 全部去除内联字段
- 命令/参数与 product 的关联通过 `products` 表（T-0098 已建）+ N:M `mml_command_product_rel` 表达
- 本次重建：**先全删，不引入新 rel 表**（standard-model.xml 不区分 product），future 真有 product-specific 需求再加

### 3.5 ⚠️ E. `mml_commands` 5 个 JSONB/array 字段语义重叠

```sql
-- 原始 (mig 007)
param_template       JSONB
product_types        JSONB DEFAULT '[]'

-- seed/004 ALTER 加的
operation_type       TEXT DEFAULT 'LST'
param_paths          JSONB DEFAULT '[]'
supported_operations JSONB DEFAULT '["LST"]'
```

**问题**：
- `param_template` vs `param_paths`：前者占位符模板，后者路径列表 — 重叠 80%
- `operation_type` vs `supported_operations`：前者 enum 单值，后者列表 — 同样命令两个字段写两遍
- `product_types`：T-0090-b 反向操作

**建议**：精简到 3 列：
```sql
operation_type TEXT NOT NULL CHECK (operation_type IN ('LST','MOD','ADD','RMV'))
target_paths   JSONB NOT NULL    -- 一个命令影响的全部 path 列表（含占位符 {i}）
target_object  TEXT              -- 仅 ADD/RMV 命令有值，object 容器 path
```
其它字段全删（param_template / param_paths / supported_operations / product_types）。

### 3.6 ⚠️ F. `mml_param_versions` 用业务 code 做主键

```sql
version_code  VARCHAR(50) PRIMARY KEY        -- 'QB1.0' 这种业务字符串
```

下游 `mml_params.param_version` / `mml_param_groups.param_version` 都 FK 到这里。

**问题**：业务 code 改名是常见诉求（'QB1.0' → 'Qcells-B-1.0'）；改一次全表 cascade，**或者**不允许改名 → 业务被技术约束反过来卡住。
**建议**：标准 UUID 主键 + `version_code` UNIQUE 索引；FK 改 UUID。

### 3.7 ⚠️ G. `mml_scripts` 混存模板 + 单次执行态（**Q5 确认后**）

```sql
-- 模板属性（保留）
script_name, description, content, device_type, creator, tags
status  -- 保留：但语义改为「模板启用状态」 active/disabled

-- mig 026 加的单次执行态（删除）
start_time, end_time, type, progress, result
```

**Q5 决议**：
- `mml_scripts.status` **保留**，但语义收紧为「模板启用状态」（取值 `active` / `disabled`），不是「上次执行状态」
- 单次执行字段 `start_time/end_time/type/progress/result` **删除**
- **周期性脚本**：每次 cron 触发 → 创建一行 `mml_tasks(script_id=...)`；查脚本执行历史 = `SELECT * FROM mml_tasks WHERE script_id=$1 ORDER BY started_at DESC`
- 已有数据迁移：老 scripts 行的 `start_time/end_time/result` 一次性 INSERT 进 `mml_tasks` 当历史 run，然后删字段

### 3.8 ⚠️ H. `mml_audit_log` 与全局 `ops_audit_log` 双轨（**Q6 确认硬删**）

`ops_audit_log` 是 T-0109 建立的统一审计基础设施（按 op_type/target_type 分类，全模块共用）。`mml_audit_log` 是 MML 模块自己造的轮子。

**Q6 决议**：直接硬删：
- migration 直接 `DROP TABLE mml_audit_log`
- MML 服务层把 audit 调用迁到 `ops_audit_log`（在 `Log()` 入口传 `target_type="mml_template"` / `"mml_script"` 等）
- 历史 audit 行**不迁移**（用户接受丢弃；如确认需要保留可在 migration 添加 INSERT...SELECT 一次性搬运）

### 3.9 ⚠️ I. `mml_params` 用 (param_version, tr069_path) 联合唯一

```sql
CONSTRAINT uq_param_version_path UNIQUE (param_version, tr069_path)
```

24 个 version × 1988 path 理论 47712 行；99% 同 path 在多 version 重复。

**问题**：写放大 + 同 path 元信息（type/access/range）多副本不一致风险。
**建议**：
- 把 `tr069_path` 提到独立 `mml_path_dict` 维度表（path 唯一，type/access/range 一份）
- `mml_params` 仅做 path↔version 关联（轻量行：path_id + version_id + 必要 override 字段）
- **本次重建简化版**：单 STANDARD version 路径直接 `(tr069_path) UNIQUE`，去 version 联合约束

### 3.10 ⚠️ J. `mml_command_params_rel` 删除（**Q3 确认**）

Q3 决议：删 `mml_command_params_rel`。替代：`mml_commands.target_paths JSONB` 直接存路径列表。
- 优势：减一张 N:M 表 + 减 JOIN
- 取舍：失去"某 param 被哪些 command 引用"的反向 SQL JOIN 能力 — 用 JSONB GIN 索引补：
  ```sql
  CREATE INDEX idx_mml_commands_target_paths_gin
      ON mml_commands USING GIN (target_paths);
  -- 反查: SELECT id FROM mml_commands WHERE target_paths @> '["Device.DeviceInfo.Foo"]'::jsonb;
  ```
- `mml_group_param_rel` **保留**（与 §3.10 旧版一致），由 Loader 按 path 前缀自动写。group↔param 反查仍有 JOIN 友好接口。

### 3.11 💡 K. `mml_param_groups` 字段 feature creep

```sql
cell_number INT DEFAULT 1                 -- ?
cell_index_location INT DEFAULT 0         -- ?
require_second_confirm BOOLEAN            -- UI 关注点
confirm_message_zh/en TEXT                -- UI 关注点
display_order INT                         -- UI 关注点
is_active BOOLEAN                         -- 软删？还是禁用？
deleted_at TIMESTAMPTZ                    -- 软删
```

**问题**：UI 关注点（confirm 文案 / display_order）耦合到 schema；`cell_number` 历史遗留语义不明。
**建议**：
- 删 `cell_number` / `cell_index_location`
- `require_second_confirm` + `confirm_message_*` 移到 `mml_commands`（确认弹窗本是命令级行为）
- 保留 `display_order` / `is_active` / `deleted_at` 作为通用治理字段

### 3.12 💡 L. `mml_params.tr069_path_parts GENERATED COLUMN` 设计良好

```sql
tr069_path_parts TEXT[] GENERATED ALWAYS AS (string_to_array(tr069_path, '.')) STORED
+ GIN(tr069_path_parts)
```

支持高效路径段子查询（"找所有 *.CellConfig.LTE.* 路径"）。**保留**。

---

## 4. 建议的优化 schema（重建后）

### 4.1 表清单（11 → 7，**Q3/Q6/Q7 决议反映**）

| 表 | 行 | 变更 |
|----|---|------|
| ✅ `mml_param_versions` | 1 | **硬删 24 老 versions（Q7）**；改 UUID 主键 + version_code UNIQUE；仅留 STANDARD 行 |
| ✅ `mml_param_groups` | ~80-150 | 删 parent_id/level/cell_*/4 操作 flag/confirm_*/i18n 单列；group_name → JSONB |
| ✅ `mml_params` | 1988 | 删 4 操作 flag/platform_*/mobile_*/broadband_*/title_*/explanation_*；name/explanation → JSONB |
| ✅ `mml_group_param_rel` | N:M 自动 | 不动 schema，由 Loader 自动按 path 前缀写 |
| ✅ `mml_commands` | ~140 | 删 param_template/param_paths/product_types/supported_operations；只留 operation_type + target_paths + target_object + **rpc_steps JSONB**（Q2 多 RPC） |
| ❌ `mml_command_params_rel` | — | **删表（Q3）**：target_paths JSONB + GIN 索引代替 |
| ✅ `mml_custom_command` | 用户产 | 不动（T-0090-b 已清理） |
| ✅ `mml_scripts` | 用户产 | 删 start_time/end_time/type/progress/result；**保留 status='active'/'disabled'（Q5）** |
| ✅ `mml_tasks` | 用户产 | 不动（已有 status/started_at/finished_at/result）|
| ❌ `mml_audit_log` | — | **删表（Q6）**：MML 操作改走 `ops_audit_log` |

### 4.2 新的核心字段

#### `mml_param_versions`
```sql
id            UUID PRIMARY KEY DEFAULT gen_random_uuid()
version_code  VARCHAR(50) NOT NULL UNIQUE
version_name  VARCHAR(200) NOT NULL
description   TEXT
source        VARCHAR(20) NOT NULL DEFAULT 'standard'  -- 'standard' | 'product'
is_active     BOOLEAN DEFAULT true
is_deprecated BOOLEAN DEFAULT false
created_at    TIMESTAMPTZ DEFAULT NOW()
updated_at    TIMESTAMPTZ DEFAULT NOW()
```

#### `mml_param_groups`
```sql
id            UUID PRIMARY KEY
group_code    VARCHAR(100) NOT NULL                     -- 'DEVICEINFO_ANTENNAINFO'
name_i18n     JSONB NOT NULL DEFAULT '{}'               -- {"zh":"天线信息","en":"Antenna Info"}
path          LTREE NOT NULL                            -- 唯一层级表达
version_id    UUID NOT NULL REFERENCES mml_param_versions(id)
display_order INT DEFAULT 0
is_active     BOOLEAN DEFAULT true
deleted_at    TIMESTAMPTZ
created_at    TIMESTAMPTZ DEFAULT NOW()
updated_at    TIMESTAMPTZ DEFAULT NOW()
UNIQUE (version_id, group_code)
INDEX GIST(path)
```

#### `mml_params`
```sql
id               UUID PRIMARY KEY
param_code       VARCHAR(200) NOT NULL                   -- 末段 'Azimuth'
tr069_path       VARCHAR(1000) NOT NULL                  -- 'Device.DeviceInfo.AntennaInfo.Azimuth'
tr069_path_parts TEXT[] GENERATED ALWAYS AS (...) STORED -- 保留
value_type       VARCHAR(50) NOT NULL                    -- 'STRING'/'INT'/'U_INT'/'BOOLEAN'/'DATE_TIME'
value_constraint JSONB                                   -- {min, max, maxLen, enum?}
default_value    TEXT
is_writable      BOOLEAN NOT NULL DEFAULT false          -- TR-069 access=READ_WRITE
is_leaf          BOOLEAN NOT NULL DEFAULT true
version_id       UUID NOT NULL REFERENCES mml_param_versions(id)
name_i18n        JSONB DEFAULT '{}'                      -- 可选 {"zh":"方位角","en":"Azimuth"}
explanation_i18n JSONB DEFAULT '{}'                      -- 可选
display_order    INT DEFAULT 0
is_active        BOOLEAN DEFAULT true
deleted_at       TIMESTAMPTZ
created_at       TIMESTAMPTZ DEFAULT NOW()
updated_at       TIMESTAMPTZ DEFAULT NOW()
UNIQUE (version_id, tr069_path)
INDEX GIN(tr069_path_parts)
```

#### `mml_commands`（**Q2 多 RPC + Q1 7 类 category + Q8 含完整前缀**）
```sql
id                  UUID PRIMARY KEY
command_code        VARCHAR(100) NOT NULL UNIQUE
                    -- 'LST_DEVICE_DEVICEINFO_ANTENNAINFO' 含完整前缀（Q8）
command_name_i18n   JSONB NOT NULL                -- {"zh":"列出天线信息","en":"List Antenna Info"}
category            VARCHAR(10) NOT NULL          -- '1'..'7' 既有 7 类 code（Q1）
operation_type      VARCHAR(10) NOT NULL          -- 'LST'/'MOD'/'ADD'/'RMV'
group_id            UUID REFERENCES mml_param_groups(id)
-- Q2 多 RPC 支持
rpc_steps           JSONB NOT NULL                -- [{"method":"GetParameterNames","params":{...}},
                                                  --  {"method":"GetParameterValues","params_from_step":0}]
rpc_method          VARCHAR(50) NOT NULL          -- = rpc_steps[0].method（兼容老 client，冗余但便利）
-- 目标参数 / 对象
target_paths        JSONB NOT NULL DEFAULT '[]'   -- ["Device.DeviceInfo.AntennaInfo.Azimuth", ...]
target_object       VARCHAR(500)                  -- ADD/RMV 时的 object_path（如 Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.）
-- 二次确认（从 group 上移）
require_confirm     BOOLEAN DEFAULT false
confirm_msg_i18n    JSONB DEFAULT '{}'
help_doc            TEXT DEFAULT ''
created_at          TIMESTAMPTZ DEFAULT NOW()
updated_at          TIMESTAMPTZ DEFAULT NOW()
CHECK (operation_type IN ('LST','MOD','ADD','RMV'))
CHECK (category IN ('1','2','3','4','5','6','7'))

-- Q3 反查支持：删 mml_command_params_rel 后用 JSONB GIN 索引
CREATE INDEX idx_mml_commands_target_paths_gin
    ON mml_commands USING GIN (target_paths);
CREATE INDEX idx_mml_commands_category
    ON mml_commands(category);
CREATE INDEX idx_mml_commands_group_id
    ON mml_commands(group_id);
```

### 4.4 i18n 实现方式 — 两种方案对比（**Q4 解释**）

> Q4：你问"抽单表是什么意思？没明白" — 这里展开。

i18n 字段（如 `param_name_zh` / `param_name_en` / `explanation_zh` / `explanation_en` 等）目前是**双列内联**。改造有两种路线：

#### 方案 A：JSONB 内联（轻量，推荐）

把双列 `_zh` / `_en` 合并成一个 JSONB 列：

```sql
-- 改造前
param_name_zh VARCHAR(500)
param_name_en VARCHAR(500)
explanation_zh TEXT
explanation_en TEXT

-- 改造后
name_i18n        JSONB DEFAULT '{}'   -- {"zh":"方位角","en":"Azimuth"}
explanation_i18n JSONB DEFAULT '{}'   -- {"zh":"...","en":"..."}
```

**优点**：
- 加新语言（日/俄/西班牙语）**无需 ALTER TABLE**，只改 JSONB 内容
- 行级访问：`SELECT name_i18n->>'zh' AS name FROM mml_params`
- 落地简单：Loader 写 / FE 读都简单

**缺点**：
- 不能用 SQL 做全文检索某语言（需要 GIN 索引特定 key 才能高效）
- 翻译版本管理弱（无独立 updated_at per locale）

#### 方案 B：抽独立 `mml_i18n` 单表（重量，扩展性强）

```sql
CREATE TABLE mml_i18n (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_table  VARCHAR(64) NOT NULL,    -- 'mml_params' / 'mml_commands' / 'mml_param_groups'
    target_id     UUID NOT NULL,           -- 对应表的 id
    locale        VARCHAR(10) NOT NULL,    -- 'zh' / 'en' / 'ja'
    key           VARCHAR(64) NOT NULL,    -- 'name' / 'explanation' / 'confirm_msg'
    value         TEXT NOT NULL,
    updated_at    TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (target_table, target_id, locale, key)
);

-- 主表的双语列全部删掉。
-- 查询：JOIN mml_i18n WHERE target_table='mml_params' AND target_id=$1 AND locale='zh'
```

**优点**：
- 集中翻译运营（一张表看所有未翻译条目）
- 每条翻译独立版本号 / updated_at / updated_by — 适合多人翻译协作
- 增减语言 0 schema 改动

**缺点**：
- 每次主表查都要 LEFT JOIN（n*m 扩展）
- 数据落地复杂（Loader 写主表 + 每个 i18n 字段写多行）
- 行数膨胀：1988 params × 2 字段（name/explanation） × 2 语言 = ~8000 行 i18n 表

#### 决议建议：**方案 A（JSONB 内联）**

理由：
- 当前只有 zh + en 两种语言，方案 A 已够
- Loader 写数据简单（一行 JSONB 一次写完）
- 真要做多人翻译协作（Crowdin / Lokalise 风格）再升级到方案 B
- 数据库行 size 增加可忽略（JSONB 比独立列略大 ~20%）

下文 §4.2 schema 已用 JSONB（如 `name_i18n` / `explanation_i18n`）— **如同意方案 A 即按当前 schema 落地**；改方案 B 需重写 §4.2 字段定义。

---

### 4.3 字段数变化总览（v2 反映决议）

| 表 | 旧字段 | 新字段 | Δ | 备注 |
|----|-------|-------|---|------|
| `mml_param_versions` | 12 | 9 | -3 | 老 24 行硬删（Q7） |
| `mml_param_groups` | 25 | 10 | **-15** | 最大瘦身（删 parent_id/level/4 flag/cell_*/confirm_* + JSONB i18n）|
| `mml_params` | 31 | 15 | **-16** | 4 flag/platform_*/i18n 全清 |
| `mml_commands` | 14（含 ALTER）| 13 | -1 | 删 4 字段（param_template/param_paths/product_types/supported_operations）+ 加 1 字段（rpc_steps Q2）|
| `mml_command_params_rel` | 3 | — | **删表** | Q3 决议 |
| `mml_scripts` | 14 | 9 | -5 | 删 4 执行态字段（start/end/type/progress/result）+ 保留 status 1 字段（Q5）|
| `mml_audit_log` | — | — | **删表** | Q6 决议 |

总字段数：**99 → 56**（-43%，瘦身 43%）+ 删 2 表

---

## 5. Loader 实现概要

### 5.1 文件布局

```
omcgo/internal/config/parammodel/mmlstandardloader/
├── loader.go         # 入口 Load(ctx)；事务包裹 6 步 UPSERT
├── parser.go         # XML decode → []ParamSpec / []ObjectSpec
├── grouper.go        # path-prefix trie + ≤50 阈值递归切片
├── command_gen.go    # group → LST/MOD/ADD/RMV 命令生成规则
├── category_map.go   # L2 分支 → 中文 category 11 项硬编码映射
└── loader_test.go    # 解析 / 分组 / 命令生成 单测
```

### 5.2 启动期挂载位置

`internal/dictload` 已有 ModuleGraph 编排框架（T-0098）：

```
ModuleGraph.Register("mml-standard", []string{"pg", "redis"}, mmlStandardLoader.Load)
```

启动时机：PG ready 之后；与 4 个既有 dictloader 并行（不依赖它们）。

### 5.3 Load 流程伪代码

```go
func (l *Loader) Load(ctx) error {
    paths, objects := parseXML(xmlPath)             // 1988 param + 13 object

    return tx(ctx, func(tx) error {
        version := upsertVersion(tx, "STANDARD")    // 1
        params  := upsertParams(tx, paths, version) // 2: 1988 rows
        groups  := groupTreeWithThreshold(50)       // 3: 路径树 → ~100 group
        upsertGroups(tx, groups, version)           // 4
        cmds    := genCommands(groups, params)      // 5: LST/MOD/ADD/RMV
        upsertCommands(tx, cmds)
        wireRelations(tx, params, groups, cmds)     // 6: 两张 N:M rel 表
        deprecateLegacyVersions(tx)                 // 7: 24 老 version is_deprecated=true
        return nil
    })
}
```

### 5.4 幂等性保证

- 全部 UPSERT 用 `ON CONFLICT (UNIQUE_KEY) DO UPDATE`
- 二次执行：相同 paths → 相同 group_code → 相同 command_code → 0 副作用
- 删除孤儿：本次先**不实现**孤儿清理（人为 path 删除时旧行残留可接受；下次完整 reload 用 admin CLI）

### 5.5 失败行为 + 可观测性（**Q9 确认**）

**Q9 决议**：Loader 失败保持 Warn 不阻塞（对齐既有 4 个 dictloader），但**强化日志结构化** + **Prometheus metric** 让运维一行命令定位问题。

#### 结构化日志（每一步打）

```go
logger.Info("mml standard loader: phase start",
    zap.String("phase", "parse_xml"),
    zap.String("xml_path", xmlPath))

logger.Error("mml standard loader: phase failed",
    zap.String("phase", "upsert_params"),
    zap.String("xml_path", xmlPath),
    zap.Int("attempted_rows", 1988),
    zap.Int("committed_rows", 1200),     // 失败前已提交多少
    zap.String("first_failed_path", "Device.X.bad"),
    zap.Error(err))

logger.Info("mml standard loader: completed",
    zap.Duration("elapsed", elapsed),
    zap.Int("params_upserted", 1988),
    zap.Int("groups_upserted", 132),
    zap.Int("commands_generated", 142))
```

日志关键字段（运维 grep 用）：
- `phase`: `parse_xml` / `upsert_version` / `upsert_params` / `build_groups` / `gen_commands` / `wire_rels` / `deprecate_legacy`
- `xml_path`: 出问题时立刻知道哪个 XML
- `first_failed_path`: 1988 行里哪一行炸了
- `attempted_rows` / `committed_rows`: 失败到哪一步

#### Prometheus metric（新增 4 个 gauge / counter）

```go
omc_mml_loader_status{phase}              // gauge: 0=fail, 1=ok
omc_mml_loader_last_run_at_seconds        // gauge: unix timestamp
omc_mml_loader_duration_seconds           // histogram
omc_mml_loader_total{result="success|fail",phase}  // counter
omc_mml_loader_rows_upserted{table}       // gauge: 最近一次成功的行数
```

Grafana panel: 一眼看到上次跑成不成、跑了多久、各表行数。
AlertManager rule: `omc_mml_loader_status == 0 for 5m` → page 运维。

#### Self-healing 重试（轻量）

启动期失败时，**不立即重试**（避免雪崩）。提供一个内部端点供运维主动重试：

```
POST /admin/mml/reload?force=true   X-API-Key 鉴权 + sys_admin only
```

后端调 `loader.Load(ctx)` 一次，返回 200 + 结果摘要 JSON，便于 CI/CD 健康检查。

---

## 6. 命令生成规则（**v3 — 多 RPC 由 group/script 承载**）

### 6.1 三层 RPC 编排模型（v3 关键澄清）

用户审核反馈：**mml_param_groups 不是纯逻辑分组，而是"批量 RPC 执行单元"** — 一次执行一个 group 即触发该 group 下多个 commands 各自的 RPC。

这条 review 把 RPC 编排责任从 `mml_commands` 内嵌（v2 提议的 rpc_steps）拨回到 **mml_param_groups / mml_scripts**：

```
Layer 1: mml_commands          ── 单 RPC 原子单元（rpc_method 单一字段）
Layer 2: mml_param_groups      ── 命令逻辑集合 + 批量执行入口（一次跑 N commands）
Layer 3: mml_scripts           ── 用户编辑的命令序列（多行 → 多 commands）
```

**所有上层"多 RPC"诉求都通过下层的"多 commands"展开**，dispatcher 不需要新加 step interpreter。

| 用户面诉求 | 实现机制 | 涉及表 |
|-----------|---------|-------|
| 单条命令直发 | `mml_commands(1 行)` → `mml_tasks.commands[1]` → 1 RPC | mml_commands |
| 一次执行某类命令（基础信息）| `mml_param_groups(1 行)` → 展开为该 group 下 N commands → `mml_tasks.commands[N]` → N RPCs | mml_param_groups → mml_commands |
| 用户编辑脚本 | `mml_scripts.content` 多行 → split → N commands → N RPCs | mml_scripts → mml_commands |
| 含 {i} 动态实例 LST | 单 command + fanout 检测 `{i}` 透明先 GPN 后 GPV | mml_commands + fanout 改造 |

### 6.2 撤回 `rpc_steps`，新增 group 批量执行能力

**结论 1**：`mml_commands` 保持 v1 的单 `rpc_method` 字段，**撤回** rpc_steps JSONB 设计。

**结论 2**：`mml_param_groups` 升级为 "可执行批" — 新增执行 API：

```
POST /api/v1/mml/groups/:group_id/execute
body: {
  device_sns: ["SN1", "SN2"],
  parameters: {...},          // 可选 — 覆盖 group 内所有命令的同名参数
  execute_type: "immediate"   // 沿用现有 ExecuteType
}
```

后端展开为 N 个 commands → mml_tasks → fanout → N device_tasks。

**等价于**：用户行为上看是"一键跑设备信息相关所有 RPC"；底层走相同的 `mml_tasks.commands` 数组通路，零 dispatcher 改动。

### 6.3 4 类生成规则（v3 — 单 RPC + fanout 含 {i} 透明展开）

| 操作 | 触发条件 | rpc_method | target_paths | target_object | 注 |
|------|---------|-----------|--------------|---------------|---|
| **LST** | 总是 | `GetParameterValues` | group 全部 path | NULL | 若 path 含 `{i}`，**fanout 层自动**先 `GetParameterNames` 列实例 → 再 GPV 取值 |
| **MOD** | ≥1 path `is_writable=true` | `SetParameterValues` | 仅 RW path | NULL | 单 RPC |
| **ADD** | group path 末段 `{i}.` | `AddObject` | NULL | group path 去 `{i}` | 单 RPC |
| **RMV** | 同 ADD | `DeleteObject` | NULL | 同上 | 单 RPC |

**fanout 改造点**（Q-NEW-2 决议 — option B）：

```go
// internal/mml/fanout.go: buildDeviceTaskRequests 加预处理
for cmdIdx, cmd := range mmlTask.Commands {
    rpcMethod := cmd["rpc_method"].(string)
    paths := cmd["param_paths"].([]string)

    if rpcMethod == "GetParameterValues" && containsInstanceIndex(paths) {
        // 含 {i} 路径：先插入一个 GetParameterNames 子 device_task
        reqs = append(reqs, &task.CreateTaskRequest{
            Method: "GetParameterNames",
            Params: gpnParams(paths),  // 把 {i} 段截到 "Device.X." 形式
            ...
            DependsOn: nil,  // 第一步无依赖
        })
        // 然后插入 GPV，DependsOn 指向 GPN 完成
        reqs = append(reqs, &task.CreateTaskRequest{
            Method: "GetParameterValues",
            Params: gpvParams(paths),
            DependsOn: &lastTaskID,
        })
    } else {
        // 普通命令：1 个 RPC
        reqs = append(reqs, ...)
    }
}
```

> **依赖**：internal/task 是否已支持 `DependsOn`？需要快速调研 — 不支持的话需加这个机制（看 §6.5）。

### 6.4 脚本语法扩展含参数（**Q-NEW-3 决议**）

现状 `splitScriptLines` 把每行当作纯 `command_code`；执行时用户需另填 `parameters` 字段。新语法扩展：

```
# 注释（保留）
LST_DEVICE_DEVICEINFO_ANTENNAINFO                              # 无参 LST
MOD_DEVICE_DEVICEINFO_ANTENNAINFO Azimuth=180 Downtilt=5      # 单条带 K=V 参数
MOD_NEIGHBORLIST_LTECELL[1] PCI=512                          # 带实例号 {i}=1
ADD_NEIGHBORLIST_LTECELL                                      # ADD 无 K=V
```

Parser 输出：

```go
type ScriptLine struct {
    CommandCode    string             // 'MOD_DEVICE_DEVICEINFO_ANTENNAINFO'
    InstanceIndex  *int               // {i} 实例号（如 [1] 写法），可空
    Parameters     map[string]string  // {"Azimuth": "180", "Downtilt": "5"}
    SourceLine     int                // 原始行号，错误定位用
}
```

**校验**：
- `command_code` 必须在 `mml_commands` 表找到，否则 `error: line %d: unknown command %q`
- 参数 key 必须在 `mml_commands.target_paths` 引用的 mml_params 里找到（按 param_code 匹配），否则 `error: line %d: unknown parameter %q for command %s`
- 必填参数检查（如 MOD 没填值 → 报错）

### 6.5 多行严格序列执行（**Q-NEW-4 决议**）

用户拍板 — 多行命令**严格序列**：上一行 device_task 终态后才发下一行。

**现状**：`fanout.go:124-130` 一次性把所有 device_tasks 推入 task service 队列；ACS 是按设备 SN 串行处理 device_tasks（一个设备同时只跑一个），但**跨设备并行**，跨命令也是 enqueue order 无依赖。

**需补能力 — `device_tasks.depends_on UUID` 字段 + dispatcher 等待逻辑**：

```sql
ALTER TABLE device_tasks ADD COLUMN depends_on UUID;
CREATE INDEX idx_device_tasks_depends_on ON device_tasks(depends_on) WHERE depends_on IS NOT NULL;
```

dispatcher 出队前检查：`SELECT 1 FROM device_tasks WHERE id=depends_on AND status IN ('completed','failed','expired')` — 上游未终态则跳过该任务等下一轮。

**这是新增的基础设施改造**，scope 比纯 MML 重建大。**评估**：

| 选项 | 工作量 | 风险 |
|------|-------|------|
| 本期一并做 device_tasks.depends_on | +1.5d | 改 internal/task 核心；影响其它模块（ops/backup/provision）但只加列不破坏 |
| 本期不做，先记录 scope 缺口 | 0 | 多行脚本"看起来"严格序列，实际可能并行（如设备 A 跑 line2 时设备 B 还在 line1）|
| 简化版：mml fanout 内部用 channel/waitgroup 串行 enqueue | +0.3d | 仅在 MML 任务内有效；其它模块未受益 |

> **建议本期做简化版**（option 3），device_tasks.depends_on 拆 future task。

---

### ⚠️ 待用户拍板的 v3 新增决策点

下面 3 点是 §6 重写后引入的新决策，需要你确认：

### 6.3 命令命名规则（Q8 确认含完整前缀）

**Q8 决议**：`command_code` 含**完整层级前缀**（不去顶层 `Device`），与 TR-069 path 保持一致。

`command_code = <OP>_<GROUP_CODE>`
`group_code = ` 路径段去 `{i}` → UPPER_SNAKE_CASE，**保留所有层级**

**示例**：

| TR-069 path | group_code | LST command_code | MOD command_code |
|------------|-----------|------------------|------------------|
| `Device.DeviceInfo.AntennaInfo.*` | `DEVICE_DEVICEINFO_ANTENNAINFO` | `LST_DEVICE_DEVICEINFO_ANTENNAINFO` | `MOD_DEVICE_DEVICEINFO_ANTENNAINFO` |
| `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.*` | `DEVICE_SERVICES_FAPSERVICE_CELLCONFIG_LTE_RAN_NEIGHBORLIST_LTECELL` | `LST_..._LTECELL` | `MOD_..._LTECELL` + `ADD_..._LTECELL` + `RMV_..._LTECELL` |
| `Device.FaultMgmt.CurrentAlarm.*` | `DEVICE_FAULTMGMT_CURRENTALARM` | `LST_DEVICE_FAULTMGMT_CURRENTALARM` | — (全 RO，无 MOD) |
| `DeviceGSM.Bts.{i}.*` | `DEVICEGSM_BTS` | `LST_DEVICEGSM_BTS` | `MOD_DEVICEGSM_BTS` + `ADD_...` + `RMV_...` |

> 长 code（80 字符）的 cosmetic 问题：FE 显示用 `command_name_i18n.zh`（"列出邻区 LTE 小区"），code 仅做内部 key 不展示给最终用户。

---

## 7. Category 映射（**Q1 确认基于既有 7 类**）

**Q1 决议**：保留 `sys_dictionaries.mml_command_category` **既有 7 类**，不增不减；把 standard-model.xml 的 1988 path 重新映射到这 7 类。

### 7.1 既有 7 类（保持不变）

| code | name | 说明 |
|------|------|------|
| `1` | 小区管理 | 小区配置 / RAN 参数 / 物理层 |
| `2` | 邻区管理 | NeighborList / 邻区配置 / 切换 |
| `3` | 基站管理 | 设备信息 / 天线 / 板卡 / 以太网 / 高可用 / 时间 |
| `4` | 告警查询 | 当前告警 / 告警事件 |
| `5` | 性能采集 | 性能管理 / KPI / 计数器 |
| `6` | 传输管理 | Transport / IPsec / 网络配置 |
| `7` | 版本管理 | 软件版本 / 固件 / ACS 升级配置 |

### 7.2 path → category 映射规则

> 按 path 前缀**自顶向下匹配**第一条命中规则；多模匹配时**前序优先**。

| 优先级 | path 前缀模式 | category | 估算命令数 |
|------|--------------|----------|----------|
| 1 | `Device.FaultMgmt.*` | **4 告警查询** | ~4 |
| 2 | `Device.FAP.PerfMgmt.*` | **5 性能采集** | ~3 |
| 3 | `Device.Services.FAPService.{i}.FAPControl.*.SelfConfig.*.KPI.*` | **5 性能采集** | ~2 |
| 4 | `Device.KeepalivedMgmt.Counter.*` | **5 性能采集** | ~1 |
| 5 | `Device.Services.FAPService.{i}.CellConfig.*.NeighborList.*` | **2 邻区管理** | ~12 |
| 6 | `Device.Services.FAPService.{i}.CellConfig.{i}.*.NeighborList.*` | **2 邻区管理** | ~6 |
| 7 | `DeviceGSM.Bts.{i}.NeighborList.*` 或 `DeviceGSM.Bts.{i}.Si2quaterNeighborList.*` | **2 邻区管理** | ~3 |
| 8 | `Device.Services.FAPService.{i}.CellConfig.*` | **1 小区管理** | ~25 |
| 9 | `Device.Services.FAPService.{i}.FAPControl.*` | **1 小区管理**（含 RAN 调度）| ~10 |
| 10 | `Device.Services.GsmBTSCellDT.{i}.*` | **1 小区管理** | ~3 |
| 11 | `Device.Services.FAPService.{i}.Transport.*` 或 `Device.Services.FAPService.{i}.Ipsec.*` | **6 传输管理** | ~5 |
| 12 | `Device.Services.FAPService.Ipsec.*` | **6 传输管理** | ~3 |
| 13 | `Device.Ethernet.*` 或 `Device.IP.*` 或 `Device.IPsec.*` | **6 传输管理** | ~6 |
| 14 | `Device.ManagementServer.*` | **6 传输管理**（ACS 走 TR-069 也归传输）| ~3 |
| 15 | `Device.DeviceInfo.*Software*` 或 `Device.DeviceInfo.AdditionalSoftwareVersion` 或 `Device.RemoteDeviceList.{i}.Software*` 或 `Device.Software*` 或 `Device.SoftwareCtrl.*` | **7 版本管理** | ~5 |
| 16 | `Device.DeviceInfo.*` | **3 基站管理** | ~18 |
| 17 | `Device.FAP.*`（剩余非 PerfMgmt 的）| **3 基站管理** | ~3 |
| 18 | `Device.KeepalivedMgmt.*`（剩余非 Counter）| **3 基站管理** | ~2 |
| 19 | `Device.Ethernet.*` (已上面) — N/A | — | — |
| 20 | `Device.Time.*` | **3 基站管理** | ~2 |
| 21 | `Device.LAN_HostConfigManagement.*` 或 `Device.LogMgmt.*` 或 `Device.Nr.*` | **3 基站管理** | ~3 |
| 22 | `Device.Services.FAPService.{i}.Capabilities.*` 或 `Device.Services.FAPService.{i}.X_COM.*` 或 `Device.Services.FAPService.{i}.AccessMgmt.*` 或 `Device.Services.FAPService.{i}.EMBEDDED_*` | **1 小区管理**（FAP 服务能力）| ~5 |
| 23 | `Device.FAPService.MmePoolConfigParam.*` | **6 传输管理** | ~1 |
| 24 | `DeviceGSM.*`（剩余非 NeighborList）| **3 基站管理** | ~4 |
| 25 | `boardconf.*` | **3 基站管理** | ~2 |
| 26 | `InternetGatewayDevice.*` | **3 基站管理** | ~1 |
| **兜底** | 任何其它 | **3 基站管理**（fail-safe）| 0-3 |

### 7.3 7 类粗估命令分布

| category | 估算命令数 | 占比 |
|---------|---------|------|
| 1 小区管理 | ~48 | 39% |
| 2 邻区管理 | ~21 | 17% |
| 3 基站管理 | ~38 | 31% |
| 4 告警查询 | ~4 | 3% |
| 5 性能采集 | ~6 | 5% |
| 6 传输管理 | ~18 | 14% |
| 7 版本管理 | ~5 | 4% |
| **总计** | ~140 | 100% |

> 数字是按 §6 4 类操作 + ≤50 path 阈值粗估；实际跑 Loader 后看真值。
> **如映射不合理**：调整 §7.2 规则表的前缀正则即可，schema 不变。

---

## 8. 迁移步骤

### 8.1 Migration 阶段（DDL + 数据 TRUNCATE）

```
migrations/000090_mml_schema_rebuild.sql   <!-- 原计划 000089，与 devices partial unique PR 撞号上调至 090 -->

  -- mml_param_versions 改 UUID 主键
  -- mml_param_groups 删 15 字段
  -- mml_params 删 16 字段
  -- mml_commands 删 4 字段（param_template/param_paths/product_types/supported_operations）
  -- mml_scripts 删 6 执行态字段
  -- TRUNCATE 旧数据
  -- 老 24 version is_deprecated=true（保留 id）

migrations/seed/000090_mml_category_dict.sql
  -- sys_dictionaries.mml_command_category 17 项重 seed
```

### 8.2 Loader 接入

```
internal/config/parammodel/mmlstandardloader/ 新增
cmd/app/provider/modules.go 在 dictload ModuleGraph 加一条
```

### 8.3 兼容性

| 老接口 / 数据 | 重建后 |
|--------------|-------|
| `GET /api/v1/mml/commands` | 返回新命令集（FE 树形态改变）|
| `mml_custom_command` 用户私有命令 | 保留；`commandCode` 找不到映射时 FE 自由文本展示 |
| 在飞 mml_tasks | 已 JSONB snapshot commands，不受影响 |
| 历史 mml_scripts 执行态 | 迁移到 mml_tasks（一次性数据迁移 SQL）|
| 现有 mml_audit_log 行 | 保留；future task 迁到 ops_audit_log |

---

## 9. 开放问题（需进一步审核）

| ID | 问题 | 备选答案 |
|----|------|---------|
| Q1 | 17 个 category 映射是否合理？或细化 / 合并？ | 见 §7 |
| Q2 | `mml_param_groups` 是否真的还需要？group 信息可以从 path 直接派生 | 删 / 留 |
| Q3 | `mml_command_params_rel` 还需要吗？`mml_commands.target_paths` JSONB 已含 path 列表 | 删 / 留 |
| Q4 | i18n 是用 JSONB（4.2 提案）还是抽 mml_i18n 单表？ | JSONB（轻）/ 表（重）|
| Q5 | `mml_scripts` 执行态字段历史数据怎么处理？ | 迁 mml_tasks / 直接删 |
| Q6 | `mml_audit_log` 本次直接删？还是只 deprecate？ | 删 / 留 |
| Q7 | 老 24 个 mml_param_versions 真的会被复用？还是直接 hard delete？ | 软删 / 硬删 |
| Q8 | command_code 命名是否含顶层 `DEVICE_` 前缀？ | 含 / 简化去前缀 |
| Q9 | Loader 失败策略：仅 Warn 不阻塞？还是 fatal？ | Warn（一致 dictloader）/ Fatal（强约束）|
| Q10 | 重建后第一次部署时机 — 测试环境先跑？还是直接生产？ | 测试 → 生产 / 直接生产 |

---

## 10. 工作量估算（v2 反映决议后）

| 阶段 | 工作量 | 备注 |
|------|-------|------|
| Loader 实现 + 单测 | 1.5d | parser/grouper/command_gen + rpc_steps 序列生成 + 7 类映射规则表（§7.2 26 条规则）|
| Migration 000090 schema 改 + TRUNCATE + DROP audit_log + 硬删 24 版本 | 0.7d | DDL 改动比 v1 多（多个 DROP TABLE / COLUMN）；编号由 000089 上调（撞号 fix） |
| sys_dictionaries category 现有 7 类核对（不重 seed，只确认）| 0.1d | 仅做核对 |
| `mml_scripts` 老执行态 → `mml_tasks` 数据迁移 SQL | 0.3d | INSERT...SELECT |
| MML 服务层 audit 调用迁 `ops_audit_log` | 0.4d | 替换 mml/audit_repository.go 所有调用点 |
| FE 命令树 fallback `mml_custom_command` 命名找不到时自由文本 | 0.5d | mml/Console 改动 |
| Loader 失败可观测性（4 Prometheus metric + 结构化日志）| 0.3d | Q9 要求 |
| 集成测试 + 浏览器实测 + E2E claim | 0.6d | 测试平台先跑（Q10）|
| 文档（本文 finalize + commit footer + Closing Evidence）| 0.2d | |
| **合计** | **~4.6d** | M-L task，1 sprint 内可完成；建议拆 2 commit（schema + Loader / data migration + FE） |

---

## 11. 决策追踪表（v1 用户审核结果）

> 2026-05-13 v1 review 用户 10/10 已拍板，详见文档头 v2 更新摘要。

### 11.1 v1 review 决议

| 决策点 | 决议 | 日期 |
|--------|------|------|
| §3.1 A: 删 parent_id/level，保留 ltree | ✅ 同意 | 2026-05-13 |
| §3.2 B: 操作 flag 从 params 移到 commands | ✅ 同意 | 2026-05-13 |
| §3.3 C: i18n 用 JSONB（Q4 后） | ✅ 待 Q4 二次确认（默认 A） | 2026-05-13 |
| §3.4 D: 全删 product/platform 内联字段 | ✅ 同意 | 2026-05-13 |
| §3.5 E: mml_commands 精简到 3 JSONB | ✅ 同意（+ Q2 加 rpc_steps）| 2026-05-13 |
| §3.6 F: mml_param_versions 改 UUID 主键 | ✅ 同意 + Q7 硬删老 24 行 | 2026-05-13 |
| §3.7 G: mml_scripts 删执行态 | ✅ Q5 — 保留 status 模板态，删单次执行字段 | 2026-05-13 |
| §3.8 H: mml_audit_log 处理 | ✅ Q6 — **硬删表**，迁 ops_audit_log | 2026-05-13 |
| §3.9 I: mml_params 简化 UNIQUE 约束 | ✅ 同意（单 STANDARD version 直接 path UNIQUE）| 2026-05-13 |
| §3.10 J: mml_command_params_rel | ✅ Q3 — **删表**，target_paths JSONB + GIN 替代 | 2026-05-13 |
| §3.11 K: mml_param_groups 删 confirm/cell_* | ✅ 同意 | 2026-05-13 |
| Q1 命令 category 映射 | ✅ 保留**既有 7 类**，重新分配 path 归类 | 2026-05-13 |
| Q2 mml_param_groups 是否保留 | ✅ **保留** + mml_commands 加 rpc_steps 支持多 RPC | 2026-05-13 |
| Q3 mml_command_params_rel | ✅ 删（target_paths JSONB 替代） | 2026-05-13 |
| Q4 i18n 单表 vs JSONB | ⚠️ 二次解释中 — 默认 JSONB（轻量），等用户拍板 | 2026-05-13 |
| Q5 mml_scripts 状态字段 | ✅ 保留模板 status，删单次执行字段 | 2026-05-13 |
| Q6 mml_audit_log | ✅ 硬删 | 2026-05-13 |
| Q7 24 老 versions | ✅ 硬删 | 2026-05-13 |
| Q8 command_code 前缀 | ✅ 含完整层级（含 DEVICE_）| 2026-05-13 |
| Q9 Loader 失败策略 | ✅ Warn 不阻塞 + 结构化日志 + 4 Prometheus metric | 2026-05-13 |
| Q10 部署窗口 | ✅ 先测试平台，无生产 | 2026-05-13 |

### 11.2 v2 review 结果（2026-05-13 二轮）

| 决策点 | 决议 | 备注 |
|--------|------|------|
| Q-NEW-1 rpc_steps 撤回？ | ✅ 撤回；mml_param_groups 升级为"批量执行单元" | §6.1/6.2 重写 |
| Q-NEW-2 {i} LST 怎么生成？ | ✅ 一个 LST + fanout 透明 GPN→GPV | §6.3 fanout 改造 |
| Q-NEW-3 脚本格式 | ✅ 扩展含参数语法 | §6.4 新 parser |
| Q-NEW-4 脚本多行 | ✅ 严格序列 | §6.5 选简化版 fanout 内串行 |

### 11.3 v3 新增决策追踪（全部已签）

| ID | 问题 | 决议 | 日期 |
|----|------|------|------|
| **Q-V3-1** | `mml_param_groups` 批量执行 API 本期实现？ | ✅ **A — 本期实现 endpoint + FE 改** | 2026-05-13 |
| **Q-V3-2** | `{i}` 透明 GPN→GPV 依赖机制？ | ✅ **B — fanout 内 channel 串行**（不动 internal/task）| 2026-05-13 |
| **Q-V3-3** | 多行脚本严格序列依赖机制？ | ✅ **B — fanout 内 channel 串行**（同 Q-V3-2 统一）| 2026-05-13 |
| **Q-V3-4** | 脚本带参 parser 失败时？ | ✅ **A — 整脚本拒绝 fail-fast**（返 400 + 行号 + 原因）| 2026-05-13 |
| **Q-V3-5** | i18n 方案？ | ✅ **A — JSONB 内联 name_i18n** | 2026-05-13 |
| **Q-V3-6** | §7.2 26 条 path → 7 category 规则？ | ✅ 通过 | 2026-05-13 |
| **Q-V3-7** | `/admin/mml/reload` 端点本期做？ | ✅ 做 | 2026-05-13 |

> **共 17/17 决议全签**（v1 10 项 + v3 7 项），方案进入 implementation-ready 状态。

### 11.4 工作量重估（v3）

| 阶段 | v2 | v3 增 | v3 合计 |
|------|----|------|---------|
| Loader 实现 | 1.5d | +0.2d（参数 parser） | 1.7d |
| Migration / Schema 改 | 0.7d | 0 | 0.7d |
| MML 服务层迁 ops_audit_log | 0.4d | 0 | 0.4d |
| Loader 可观测性 | 0.3d | 0 | 0.3d |
| FE Console fallback | 0.5d | +0.3d（group 批量按钮）| 0.8d |
| **fanout 改造**（{i} 透明 + 多行串行）| — | **+1.0-1.5d** | **1.0-1.5d** |
| mml_scripts 数据迁移 | 0.3d | 0 | 0.3d |
| 集成测试 + 浏览器实测 | 0.6d | +0.2d（group 执行测试）| 0.8d |
| 文档收尾 | 0.2d | 0 | 0.2d |
| **合计** | 4.6d | +1.7-2.2d | **6.3-6.8d** |

工作量从 M（4.6d）升到 **M-L（~7d）**，超 1 sprint 容量上限。建议**拆 2 个 sprint**：
- Sprint A（~3.5d）：schema 重建 + Loader + Console 基础 fallback
- Sprint B（~3.5d）：fanout 改造（{i} 透明 + 多行串行）+ group 批量 API + 集成测试

---

## 附录：standard-model.xml 结构统计

| 维度 | 值 |
|------|---|
| 文件大小 | 2008 行 |
| header 自报 | `totalPaths=2001  totalObjects=13  totalParams=1988` |
| 顶层命名空间 | Device (1840) / DeviceGSM (139) / boardconf (7) / InternetGatewayDevice (2) |
| access 分布 | READ_WRITE 1613 / READ_ONLY 375 |
| type 分布 | STRING 1509 / U_INT 284 / INT 102 / BOOLEAN 81 / DATE_TIME 12 |
| 最大子树 | Device.Services.FAPService.{i}.CellConfig.* (779 params) |
| 含 `{i}` | 多数 Services 路径，需 ADD/RMV 命令 |
