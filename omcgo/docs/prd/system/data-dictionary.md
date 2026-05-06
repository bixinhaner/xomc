# 系统管理 — 字典管理（System / Data Dictionary）PRD

> 文档目的：管理系统枚举字典（如「设备状态」「告警级别」「用户性别」），为前端下拉/标签提供集中式数据源，并支持运营商自定义扩展。

| 版本 | 日期 | 作者 | 备注 |
|------|------|------|------|
| 0.1  | 2026-05-06 | Backend/Frontend Team | 基于现网 `DataDictionary/index.tsx` + `internal/admin/dictionary_*.go` 抽取 |

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
