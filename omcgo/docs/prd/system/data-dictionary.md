# 系统管理 — 字典管理（System / Data Dictionary）PRD

> 文档目的：管理系统枚举字典（如「设备状态」「告警级别」「用户性别」），为前端下拉/标签提供集中式数据源，并支持运营商自定义扩展。

| 版本 | 日期 | 作者 | 备注 |
|------|------|------|------|
| 0.1  | 2026-05-06 | Backend/Frontend Team | 基于现网 `DataDictionary/index.tsx` + `internal/admin/dictionary_*.go` 抽取 |
| 0.2  | 2026-05-07 | Frontend Team | **新增「添加子项」**（字典项树形）。详见末尾 §10 增量。**已审核**：Q4=同父唯一/跨分支可重名；其余按推荐 |

**关联功能域**：F06 OMC-R 核心 / 配置基础

**相关文件**：

| 层 | 路径 |
|----|------|
| 前端页面 | [omcmb/webcode/src/pages/system/DataDictionary/index.tsx](../../../../omcmb/webcode/src/pages/system/DataDictionary/index.tsx) |
| 前端 API | [omcmb/frontend-core/src/services/api/adminApi.ts](../../../../omcmb/frontend-core/src/services/api/adminApi.ts)（`getDictionaryList` / `createDictionary` / ...）|
| 后端 Handler | [omcgo/internal/admin/dictionary_handler.go](../../../internal/admin/dictionary_handler.go) |
| 后端 Service | [omcgo/internal/admin/dictionary_service.go](../../../internal/admin/dictionary_service.go) |
| 后端 Repo | [omcgo/internal/admin/pg_dictionary_repository.go](../../../internal/admin/pg_dictionary_repository.go) |
| 数据库 | [omcgo/migrations/000009_sys_admin.sql:72-108](../../../migrations/000009_sys_admin.sql#L72-L108) |

---

## 1. 业务背景

字典（Dictionary）是「主从结构」的枚举数据：

- **字典主表**（`sys_dictionaries`）：每条 = 一个枚举类别，如 `gender`（性别）、`device_status`（设备状态）
- **字典详情**（`sys_dictionary_details`）：每条 = 主表的一个选项，如 `(label="男", value="1")`、`(label="女", value="2")`

使用场景：
- 前端表单下拉框统一从 `GET /admin/dictionaries/{type}` 取选项，避免硬编码
- 告警级别 / 设备状态等业务枚举集中维护
- 运营商定制：同一字典在不同部署可有不同选项集

---

## 2. 实体模型

### 2.1 主表 `sys_dictionaries`（[migrations/000009_sys_admin.sql:72-88](../../../migrations/000009_sys_admin.sql#L72-L88)）

| 列 | 类型 | 约束 | 说明 |
|----|------|------|------|
| `id` | BIGSERIAL | PK | |
| `name` | VARCHAR(128) | NOT NULL | 中文展示名，如 "性别" |
| `type` | VARCHAR(64) | NOT NULL | 英文标识（程序引用），如 `gender`，活跃记录唯一 |
| `status` | BOOLEAN | NOT NULL DEFAULT TRUE | TRUE = 启用，FALSE = 禁用 |
| `description` | TEXT | NULL | |
| `created_at` / `updated_at` | TIMESTAMPTZ | NOT NULL | |
| `deleted_at` | TIMESTAMPTZ | NULL | 软删除 |

唯一索引：`UNIQUE (type) WHERE deleted_at IS NULL`（部分唯一索引，软删后允许重建）

### 2.2 详情表 `sys_dictionary_details`（[migrations/000009_sys_admin.sql:92-108](../../../migrations/000009_sys_admin.sql#L92-L108)）

| 列 | 类型 | 约束 | 说明 |
|----|------|------|------|
| `id` | BIGSERIAL | PK | |
| `label` | VARCHAR(128) | NOT NULL | 选项展示文本 |
| `value` | VARCHAR(128) | NOT NULL | 选项实际值（程序引用）|
| `extend` | TEXT | NULL | 扩展数据（如颜色、图标，JSON 或纯文本）|
| `status` | BOOLEAN | NOT NULL DEFAULT TRUE | 启用/禁用 |
| `sort` | INT | NOT NULL DEFAULT 0 | 排序 |
| `sys_dictionary_id` | BIGINT | NOT NULL, FK → `sys_dictionaries(id)` ON DELETE CASCADE | 父字典 |
| `created_at` / `updated_at` / `deleted_at` | TIMESTAMPTZ | | |

### 2.3 前端类型（推断）

```ts
interface SysDictionary {
  id: number;
  name: string;
  type: string;
  status: boolean;
  desc?: string;          // ⚠️ 后端列名 description，前端送 desc，需映射
  createdAt?: string;
  updatedAt?: string;
}

interface SysDictionaryDetail {
  id: number;
  label: string;
  value: string;
  extend?: string;
  status: boolean;
  sort: number;
  sysDictionaryId: number;
}
```

---

## 3. 页面布局（双面板）

### 3.1 左侧字典主表面板

| 元素 | 说明 |
|------|------|
| 顶部搜索框 | 按 `name` 或 `type` 模糊搜索 |
| 顶部按钮 | `添加字典` |
| 列表项 | `name` + `<Tag>{type}</Tag>` + `<Tag color={status ? "" : "red"}>禁用</Tag>` |
| 选中项 | 高亮，触发右侧加载详情 |
| 行操作 | 编辑 / 删除（弹 Confirm）|

### 3.2 右侧详情列表

| 元素 | 说明 |
|------|------|
| 顶部 | 字典名（只读）+ 搜索框（按 label）+ `添加详情` 按钮（如未选字典则 disabled）|
| 列表 | 见下表 |

**详情表格列**：

| key | 标题 | dataIndex | UI 渲染 |
|-----|------|-----------|--------|
| `label` | 标签 | `label` | 文本 |
| `value` | 值 | `value` | monospace |
| `extend` | 扩展 | `extend` | 文本（`ellipsis`）|
| `status` | 状态 | `status` | `<Switch>` 切换即调 `PUT` |
| `sort` | 排序 | `sort` | 数字 |
| `actions` | 操作 | — | 编辑 / 删除 |

---

## 4. 操作清单

### 4.1 字典主表

| 操作 | 触发 | 接口 |
|------|------|------|
| 添加字典 | 顶部 | `POST /admin/dictionaries` |
| 编辑字典 | 行内 | `PUT /admin/dictionaries/{id}` |
| 删除字典 | 行内 Confirm | `DELETE /admin/dictionaries/{id}`（软删除）|

### 4.2 字典详情

| 操作 | 触发 | 接口 |
|------|------|------|
| 添加详情 | 顶部 | `POST /admin/dictionary-details` |
| 编辑详情 | 行内 | `PUT /admin/dictionary-details/{id}` |
| 删除详情 | 行内 Confirm | `DELETE /admin/dictionary-details/{id}` |
| 状态切换 | `<Switch>` | `PUT /admin/dictionary-details/{id}` 仅更新 `status` |

---

## 5. 表单字段定义

### 5.1 字典主表 Form

| name | 标签 | UI 组件 | 必填 | 校验 |
|------|------|--------|------|------|
| `name` | 字典名称 | `<Input maxLength=128>` | ✅ | |
| `type` | 字典类型（英文）| `<Input>` | ✅ | unique；编辑时只读；建议蛇形如 `device_status` |
| `status` | 启用状态 | `<Switch>` | — | 默认 ON |
| `description` | 描述 | `<Input.TextArea>` | — | |

### 5.2 字典详情 Form

| name | 标签 | UI 组件 | 必填 | 校验 |
|------|------|--------|------|------|
| `label` | 选项标签 | `<Input>` | ✅ | |
| `value` | 选项值 | `<Input>` | ✅ | unique within sysDictionaryId |
| `extend` | 扩展数据 | `<Input>` | — | 自由格式 |
| `status` | 启用状态 | `<Switch>` | — | |
| `sort` | 排序号 | `<InputNumber min=0>` | — | 默认 0 |

---

## 6. 接口契约

> Base URL：`/api/v1/admin`
> 实际路由前缀以 [dictionary_handler.go](../../../internal/admin/dictionary_handler.go) `RegisterRoutes` 为准（推断为 `/dictionaries` + `/dictionary-details`）

### 6.1 字典主表

| Method | 路径 | 说明 |
|--------|------|------|
| GET | `/dictionaries` | 列表（不分页，前端过滤）|
| GET | `/dictionaries/{id}` | 单字典详情（含 details）|
| GET | `/dictionaries/by-type/{type}` | 按 type 取（**前端下拉的真正消费入口**，如告警级别）|
| POST | `/dictionaries` | 创建 |
| PUT | `/dictionaries/{id}` | 编辑 |
| DELETE | `/dictionaries/{id}` | 软删除 |

### 6.2 字典详情

| Method | 路径 | 说明 |
|--------|------|------|
| GET | `/dictionary-details?sysDictionaryId={id}&label=` | 按字典 ID 查 + label 模糊匹配 |
| POST | `/dictionary-details` | 创建 |
| PUT | `/dictionary-details/{id}` | 编辑 |
| DELETE | `/dictionary-details/{id}` | 软删除 |

---

## 7. 后端补齐 Backlog

### P0
1. **路由注册确认**：`dictionary_handler.go` 已存在，但需确认 `RegisterRoutes` 在 `cmd/app/provider/router.go` 中已挂载到 admin 分组（grep `dictionaryHandler.RegisterRoutes` 当前未在搜索结果中出现）
2. **字段命名对齐**：前端 `desc` ↔ 后端 `description`，`status` boolean ↔ 数据库 boolean — `mapBackendDictionary` 需 explicitly handle

### P1
3. **批量操作详情**：当前只能一条条编辑详情，加 `POST /dictionary-details/batch` 一次性导入多条
4. **字典使用引用计数**：删除字典前提示「该字典被 N 个表单引用」，需要 grep 前端 `getDictionaryByType('xxx')` 的调用点（编译时静态分析）

### P2
5. **国际化支持**：`label` 加多语言版本（`label_zh`/`label_en`），或单独表 `dictionary_detail_i18n`
6. **导入/导出**：CSV / JSON 一键导入导出

---

## 8. 验收清单（DoD）

后端：
- [ ] `dictionary_handler.go` 路由已挂载到 `/api/v1/admin`
- [ ] 主表删除 → CASCADE 删详情（已通过 FK 约束自动）
- [ ] 软删除生效（设 `deleted_at`，列表查询过滤）

前端：
- [ ] 双面板交互流畅：左选 → 右加载
- [ ] 详情 `<Switch>` 切换立即生效，错误时回滚
- [ ] `npm run typecheck` & `lint` 通过

---

## 10. v0.2 增量 — 添加子项（字典项树形）

> **状态**：草稿，等审核。审核通过后我才会实施代码 + DDL 迁移。

### 10.1 业务背景

参考 UI 截图（用户提供）：

| 场景 | 现状 | v0.2 后 |
|------|------|--------|
| 字典明细列表列 | label / value / extend / status / sort / actions | label / value / **extend (扩展值)** / **level (层级)** / status / sort / **actions=添加子项+变更+删除** |
| 创建/编辑明细 | 无父级关系 | 抽屉首字段「父级字典项」可选；选了 → 当前项作为该父的孩子 |
| 顶部"+添加字典项" | 创建顶层 | 同；父级字典项 = 空 |
| 行内"+添加子项" | 不存在 | 打开同一抽屉，但「父级字典项」预填为该行 |

适用场景示例：
- **运营商 → 站点类型** 两层级联：`运营商=移动 → 站点=室外/室内`
- **设备类型 → 子类型** 两层下拉：`设备=enb / cpe → 子类型=lte/nr`
- **告警等级 → 默认通知策略** 关联

### 10.2 实体模型补充（DDL 改动）

`sys_dictionary_details` 现有列保持不变，新增两列：

| 列 | 类型 | 约束 | 说明 |
|----|------|------|------|
| `parent_id` | BIGINT | NULL, FK → `sys_dictionary_details(id)` ON DELETE CASCADE | 父级明细 ID。NULL = 顶层项；非 NULL = 子项 |
| `level` | INT | NOT NULL DEFAULT 0 | 冗余字段，0 = 顶层；1 = 一级子项；…。**待决议**：是否用 trigger 维护？还是应用层每次 INSERT/UPDATE 显式计算？|

新约束（已审核）：
- FK self-reference + ON DELETE CASCADE → 删父自动级联删所有子（Q3 决议）
- value 唯一性：**同父唯一、跨分支可重名**（Q4 决议）。落地为 PG 部分唯一索引：
  - `UNIQUE(sys_dictionary_id, value) WHERE parent_id IS NULL AND deleted_at IS NULL`
  - `UNIQUE(sys_dictionary_id, parent_id, value) WHERE parent_id IS NOT NULL AND deleted_at IS NULL`
  - 现表无 (sys_dictionary_id, value) 唯一索引，无需先 DROP

DDL 草案（迁移文件 `migrations/000061_dict_detail_parent_id.sql`）：

```sql
-- +goose Up
ALTER TABLE sys_dictionary_details
  ADD COLUMN parent_id BIGINT NULL REFERENCES sys_dictionary_details(id) ON DELETE CASCADE,
  ADD COLUMN level     INT NOT NULL DEFAULT 0;

CREATE INDEX idx_dict_detail_parent
  ON sys_dictionary_details(parent_id)
  WHERE parent_id IS NOT NULL;

-- 同字典内：顶层项 value 唯一（Q4 决议）
CREATE UNIQUE INDEX uniq_dict_detail_top_value
  ON sys_dictionary_details(sys_dictionary_id, value)
  WHERE parent_id IS NULL AND deleted_at IS NULL;

-- 同字典内：同父子项 value 唯一（跨分支可重名）
CREATE UNIQUE INDEX uniq_dict_detail_child_value
  ON sys_dictionary_details(sys_dictionary_id, parent_id, value)
  WHERE parent_id IS NOT NULL AND deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS uniq_dict_detail_child_value;
DROP INDEX IF EXISTS uniq_dict_detail_top_value;
DROP INDEX IF EXISTS idx_dict_detail_parent;
ALTER TABLE sys_dictionary_details
  DROP COLUMN IF EXISTS level,
  DROP COLUMN IF EXISTS parent_id;
```

### 10.3 前端类型补充

```ts
interface SysDictionaryDetail {
  id: number;
  label: string;
  value: string;
  extend?: string;
  status: boolean;
  sort: number;
  sysDictionaryId: number;
  parentId?: number | null;   // v0.2 新增；null/undefined = 顶层
  level: number;              // v0.2 新增；0 = 顶层
  // children?: SysDictionaryDetail[];  // 保留扩展点，§10.6 Q5 决议是否填充
}
```

### 10.4 UI 改动

**§3.2 详情列表 → v0.2 列定义**：

| key | 标题 | dataIndex | UI 渲染 | v0.2 |
|-----|------|-----------|--------|------|
| `label` | 展示值 | `label` | 文本 | — |
| `value` | 字典值 | `value` | monospace | — |
| `extend` | 扩展值 | `extend` | 文本（`ellipsis`）| — |
| `level` | 层级 | `level` | 数字 | **新增** |
| `status` | 启用状态 | `status` | "是/否" 或 `<Switch>` | — |
| `sort` | 排序标记 | `sort` | 数字 | — |
| `actions` | 操作 | — | **+ 添加子项 ｜ 变更 ｜ 删除** | **新增"+ 添加子项"** |

**操作行为**：
- 点击行内 **+ 添加子项** → 打开「添加字典项」抽屉，「父级字典项」字段预填为该行
- 点击 **变更** → 打开抽屉，预填该行所有字段（含父级字典项）
- 点击 **删除** → Confirm 弹窗（**待决议**：是否提示"该项有 N 个子项，将一并删除"，§10.6 Q3）

**显示策略**：列表平铺，不做缩进树状渲染（与图一致）；用「层级」列让用户感知层级。后续 v0.3 可考虑 antd `<Table expandable>` 树视图（脱离本期范围）。

### 10.5 表单字段补充（添加 / 编辑字典项 抽屉）

抽屉标题：「添加字典项」 / 「编辑字典项」

| 字段顺序 | name | 标签 | UI 组件 | 必填 | 校验 / 默认值 |
|---------|------|------|--------|------|--------------|
| 1 | `parentId` | 父级字典项 | `<Select allowClear>` 选项 = 当前字典内所有 detail（除自己 + 自己的子孙） | — | 留空 = 顶层项；编辑时若已是父则不可改成自己的后代 |
| 2 | `label` | 展示值 | `<Input>` | ✅ | placeholder「请输入展示值」|
| 3 | `value` | 字典值 | `<Input>` | ✅ | placeholder「请输入字典值」；唯一性见 §10.6 Q4 |
| 4 | `extend` | 扩展值 | `<Input>` | — | placeholder「请输入扩展值」|
| 5 | `status` | 启用状态 | `<Switch checkedChildren="开启" unCheckedChildren="停用">` | ✅ | 默认 = ON（开启）|
| 6 | `sort` | 排序标记 | `<InputNumber min=0>` 带 - / + 步进按钮 | ✅ | 默认 = 0；同父同 sort 时按 id 兜底 |

抽屉行为：
- 「添加子项」入口预填 `parentId = 当前行.id`
- 「添加字典项」入口（顶部按钮）`parentId = null`
- 提交前前端校验：`parentId` 不在自己（编辑时）或自己的子孙集合中 — 防环

### 10.6 待审核的开放问题

| # | 问题 | 推荐方案 | 备注 |
|---|------|---------|------|
| Q1 | **树深度上限** | 限 **3 层**（level 0/1/2），后端 INSERT 时校验 `parent.level < 2` | 防止无限嵌套；UI 列表里"层级"列阅读性下降。如需多层，请用户指定上限 |
| Q2 | **level 字段维护方式** | **应用层显式计算**：INSERT 时读父 → `level = parent.level+1`；不写 trigger，避免隐式行为 | 也可以纯派生不存储，让 GET 时计算 |
| Q3 | **删除父项策略** | **CASCADE 删所有子孙**（DDL 已给）+ 前端 Confirm 文案"将一并删除 N 个子项" | 替代方案：阻止删除直到清空子项；或软删除整子树 |
| Q4 | **value 唯一性** | **已决议**（2026-05-07）：**同父唯一、跨分支可重名**。落地为两个部分唯一索引（DDL 见 §10.2）：`(sys_dictionary_id, value) WHERE parent_id IS NULL` + `(sys_dictionary_id, parent_id, value) WHERE parent_id IS NOT NULL` | — |
| Q5 | **GET 返回结构** | 平铺 `[{id, parentId, level, ...}, ...]` 由前端组装；不返嵌套 children | 替代方案：后端递归返回 nested `children[]`。平铺更通用，便于过滤/搜索 |
| Q6 | **sort 排序范围** | **同父内排序**（即 sort 仅在同 parent_id 兄弟中有意义） | 替代方案：整字典全局 sort。同父内更符合用户对兄弟项排序的直觉 |
| Q7 | **现有数据迁移** | parent_id = NULL（全部视为顶层），level = 0，**无破坏性** | 现有 6 条 (gender) + N 条已存在记录均不受影响 |
| Q8 | **API 路径** | 沿用现路径 `POST/PUT /admin/sysDictionaryDetail/...`，请求体加 `parent_id` 字段 | 不引入新端点，对其他消费者零影响 |
| Q9 | **前端"父级字典项"下拉数据** | 调 `GET /admin/sysDictionaryDetail/getSysDictionaryDetailList?sysDictionaryId=N` 拿全集，前端剔除自己 + 子孙 | 现有端点不分页时已能直接用 |
| Q10 | **是否启用"层级"折叠/展开** | 本期不做，平铺即可 | 后续 v0.3 用 antd Table expandable + indentSize 升级 |

### 10.7 实施步骤（审核通过后）

| 阶段 | 内容 | 文件 |
|------|------|------|
| 1. DB 迁移 | 新建 `migrations/000061_dict_detail_parent_id.sql` 加 parent_id + level + index | `omgo/migrations/` |
| 2. 后端 model | `Dictionary*.go` 增字段 + 校验（防环 + 深度限制） | `omgo/internal/admin/dictionary_service.go` |
| 3. 后端 repo | `pg_dictionary_repository.go` SELECT/INSERT/UPDATE 加列 | 同上 |
| 4. 后端 handler | `Create/Update DictionaryDetail` 接受 parent_id；`List` 返 parent_id+level | `dictionary_handler.go` |
| 5. 前端 types | `adminApi.ts` BackendDictionaryDetail / DictionaryDetail 加 parentId / level | `omcmb/frontend-core/...` |
| 6. 前端 service | `mapBackendDictionaryDetail` 加映射；create/update payload 加 `parent_id` | 同上 |
| 7. 前端页面 | DataDictionary/index.tsx：详情表加列；操作列加"+添加子项"；抽屉首字段加 Select | `omcmb/webcode/src/pages/system/DataDictionary/` |
| 8. 验收 | DoD：端到端创建顶层 + 子项 + 删父级联，前端 typecheck/lint，curl 验证 | — |

### 10.8 验收清单（DoD — v0.2 增量）

后端：
- [ ] DDL 迁移成功，已存在数据 parent_id=NULL / level=0
- [ ] 创建子项时 level 自动 = parent.level + 1
- [ ] 拒绝深度 ≥ 3（按 Q1 决议；若改其他上限则同步）
- [ ] 拒绝 parent_id 形成环（编辑时自检 + 后端兜底）
- [ ] 删父 → CASCADE 删子（DDL FK 约束）
- [ ] curl POST/PUT/GET 端到端覆盖

前端：
- [ ] 详情列表新增「层级」列；操作列新增「+ 添加子项」按钮
- [ ] 抽屉新增「父级字典项」字段，行内"+添加子项"预填正确
- [ ] 编辑模式下，下拉中过滤掉自己和子孙
- [ ] 删父的 Confirm 文案显示子项数量（依赖 Q3 决议）
- [ ] `npm run typecheck` & `lint` 通过
