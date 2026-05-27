# MML Admin Catalog 页面重构设计

> **文档编号**：mml-admin-catalog-redesign-20260527
> **创建日期**：2026-05-27
> **作者**：前端设计
> **关联功能域**：F06 OMC-R 核心 · MML 控制台管理
> **当前实现**：`omcmb/webcode/src/pages/mml/admin/catalog/`

---

## 1. 背景与目标

### 1.1 现状

当前 `mml/admin/catalog` 页面采用 **Tabs 双标签** 形式：

- **"分组" 标签**：树形展示 group → group 子节点 → command 叶子，支持多层嵌套；每个分组有"新增子分组"菜单。
- **"命令" 标签**：扁平 Table 列出所有命令，点击"编辑"打开右侧 Drawer 展示并管理 path（subField）。

两个标签信息分离，运维人员需要在两个标签之间反复切换才能完成"找到分组 → 看命令 → 编辑命令的 path"这条主线。

### 1.2 目标（来自 2026-05-27 用户需求）

| 编号 | 需求 |
|------|------|
| R1 | 分组只有一级，不能添加子分组 |
| R2 | "新增根分组" → "新增分组" |
| R3 | 分组名后的 "…" 提供 **编辑** 和 **删除**（仅当不含命令时可删） |
| R4 | 分组名后的 "…" 提供 **新增命令** 入口 |
| R5 | 命令名后增加 **编辑** 和 **删除** 入口 |
| R6 | 点击命令后，**右侧** 展示其 path 列表并支持 CRUD；将"命令"标签合并到这一个页面 |

### 1.3 设计决策

| 决策 | 选择 | 备注 |
|------|------|------|
| 已有多层结构如何处理 | **保留数据，UI 只显示一级** | 后端 schema 不动；UI 仅渲染顶层 group；新建分组 `parentId=null` |
| 命令元数据编辑形式 | **右侧详情页顶部展示+内联编辑** | "编辑命令"按钮把顶部 Descriptions 切到 Form 表单态 |
| 分组删除校验位置 | **前后端双重校验** | 前端按 `commands.length>0` 禁用按钮+悬停提示；后端 API 兜底拒绝 |

---

## 2. 总体布局

### 2.1 页面结构图

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│  MML 控制台管理 / 目录管理                                              [面包屑/Tab]│
├──────────────────────────────────────────────────────────────────────────────────┤
│  Card (无 Tabs)                                                                   │
│  ┌────────────────────────────────────────────────────────────────────────────┐  │
│  │  顶部工具栏                                                                 │  │
│  │  [+ 新增分组]   [搜索框 ──────────── 🔍]      多级表头切换刷新按钮         │  │
│  ├────────────────────────────────┬─────────────────────────────────────────┐  │
│  │                                │                                          │  │
│  │  左栏 (固定 320px)              │  右栏 (flex:1)                            │  │
│  │  ──────────────                │  ──────────────                          │  │
│  │  分组 + 命令导航树              │  命令详情面板                             │  │
│  │                                │                                          │  │
│  │  📂 基站配置          [⋯ ▼]    │  ┌──────────────────────────────────┐    │  │
│  │   ├─ ⚡ 查询小区状态  [⋯ ▼]    │  │ 命令元数据 (Descriptions / Form) │    │  │
│  │   └─ ⚡ 重启小区      [⋯ ▼]    │  │  commandCode / logicalCode / op / │    │  │
│  │                                │  │  displayName / requireConfirm     │    │  │
│  │  📂 性能管理          [⋯ ▼]    │  │            [编辑命令] [保存][取消]│    │  │
│  │   ├─ ⚡ 启动测量       [⋯ ▼]    │  └──────────────────────────────────┘    │  │
│  │   └─ ⚡ 停止测量       [⋯ ▼]    │                                          │  │
│  │                                │  ┌──────────────────────────────────┐    │  │
│  │  📂 (空分组示例)      [⋯ ▼]    │  │ Path 列表 (Table)        [+ 新增] │    │  │
│  │                                │  │ ┌────────────────────────────────┐│    │  │
│  │                                │  │ │ mmlCode │ 显示名 │ 标准 path │... ││    │  │
│  │                                │  │ │ ────────────────────────────── ││    │  │
│  │                                │  │ │ CELL_ID │ 小区 ID│ Device.X.Y │... ││    │  │
│  │                                │  │ │ ...                              ││    │  │
│  │                                │  │ └────────────────────────────────┘│    │  │
│  │                                │  └──────────────────────────────────┘    │  │
│  │                                │                                          │  │
│  └────────────────────────────────┴─────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 区域拆解

| 区域 | 宽度/高度 | 内容 |
|------|----------|------|
| 顶部工具栏 | full × 56px | 主操作按钮、搜索、刷新 |
| 左栏（导航树） | 320px × auto | 一级分组 + 命令，支持点击选中、右侧 "…" 菜单 |
| 右栏（详情面板） | flex:1 × auto | 顶部命令元数据 + 下方 path 列表 |
| 分隔线 | 1px | 视觉分隔，左右两栏独立滚动 |

### 2.3 选中态约定

- 左栏点击 **分组** 节点：右栏显示**空态**（提示"请选择命令"或显示分组级统计：命令数、最近修改时间）
- 左栏点击 **命令** 节点：右栏切换到该命令的详情视图
- 选中命令以浅蓝背景高亮（Antd Tree `selectedKeys`）
- 切换命令时，若详情区处于"编辑命令"未保存状态，弹 `Modal.confirm` 询问是否丢弃

---

## 3. 左栏导航树详细设计

### 3.1 层级与渲染规则

- **只渲染两层**：
  - Level 1: Group（图标 `FolderOutlined`，文字 `displayName`，后挂 `…` 操作菜单）
  - Level 2: Command（图标 `CodeOutlined`，文字 `displayName`，后挂行内操作 `编辑` / `删除`）
- 默认 **所有分组展开**（初始化时 `expandedKeys=tree.map(g=>g.id)`）
- 已有多层数据（rootId/parentId）的子分组：
  - 当前阶段**不展示**（只过滤 `parentId==null` 的顶层）
  - 在管理员调试模式（URL `?show=all` 或 feature flag）下可见，用于数据治理
- 命令排序：`displayName` 字典序；分组排序：`displayOrder` 数值

### 3.2 分组节点（Group node）UI

```
┌────────────────────────────────────────────┐
│ ▾ 📂 基站配置                       [⋯ ▼]   │  ← 鼠标悬停时菜单按钮高亮
├────────────────────────────────────────────┤
│   ⚡ 查询小区状态                  [✎][🗑]  │
│   ⚡ 重启小区                      [✎][🗑]  │
└────────────────────────────────────────────┘
```

**右侧 "…" Dropdown 菜单项**：

| 菜单项 | 图标 | 行为 | 启用条件 |
|--------|------|------|----------|
| 编辑 | `EditOutlined` | 打开 `GroupEditorModal` (mode=`rename`) | 始终启用（除 ROOT 系统分组） |
| 新增命令 | `PlusOutlined` | 打开 `CommandEditorModal` (mode=`create`, parent=group.id) | 始终启用 |
| 删除 | `DeleteOutlined` (red) | 确认弹窗 → 调用删除 API | `commands.length === 0` 时启用；否则 disabled + Tooltip "请先删除该分组下所有命令" |

### 3.3 命令节点（Command node）UI

命令节点用 `title` 渲染图标 + 文字，**行内**挂两个 `<a>` 链接（hover 时显示）：

| 链接 | 行为 | 启用条件 |
|------|------|----------|
| 编辑 | 选中该命令，并自动把右栏顶部切到表单编辑态（等价于"选中命令 → 点编辑命令按钮"的快捷路径） | 始终启用 |
| 删除 | 确认弹窗 → 调用 `deleteCommand` | `catalogProtected === false`，否则禁用 + Tooltip"标准命令受锁定保护" |

行内链接默认**透明显示 hint**（`opacity:0.6`），hover 行时浮现完整可读颜色。

### 3.4 顶部工具栏

```
┌────────────────────────────────────────────────────────────────────┐
│ [+ 新增分组]   [搜索 group/cmd/code/path ─────────────]   [🔄 刷新] │
└────────────────────────────────────────────────────────────────────┘
```

- **新增分组按钮**：左侧 primary，点击打开 `GroupEditorModal` (mode=`create-root`)
- **搜索框**：在左栏内部即时过滤：
  - 命中 `displayName` / `commandCode` / `logicalCode` / 子 path 的 `tr069Path`
  - 命中的命令展开其所在分组并高亮
  - 清空搜索恢复全部
- **刷新按钮**：调用 `useGroupTree.refetch()`

---

## 4. 右栏详情面板详细设计

右栏分两个区域：**命令元数据**（顶部）和 **Path 列表**（下方）。

### 4.1 空态

未选中命令时：

```
┌──────────────────────────────────────────┐
│                                          │
│          (居中 Empty 图)                  │
│        请从左侧选择一个命令               │
│   或点击分组旁的 [⋯] → [新增命令]         │
│                                          │
└──────────────────────────────────────────┘
```

### 4.2 命令元数据区（显示态）

默认渲染 `<Descriptions column={2} bordered size="small">`：

```
┌────────────────────────────────────────────────────────────┐
│ 命令名称: 查询小区状态                    [编辑命令] [删除] │
├──────────────────┬─────────────────────────────────────────┤
│ commandCode      │ QUERY_CELL_STATUS                       │
│ logicalCode      │ MML_QRY_CELL                            │
│ operationType    │ [Tag: QUERY]                            │
│ displayName      │ 查询小区状态                            │
│ targetObject     │ Device.Services.X_CMCC_LTE.Cell.{i}.    │
│ requireConfirm   │ false                                   │
│ source           │ [Tag: standard]                         │
│ catalogProtected │ -                                       │
└──────────────────┴─────────────────────────────────────────┘
```

- **编辑命令** 按钮：切换为"编辑态"（见 4.3）。`catalogProtected===true` 时禁用并显示锁定 Tooltip
- **删除** 按钮：danger，行为同左栏命令节点行内删除（双入口）

### 4.3 命令元数据区（编辑态）

点 "编辑命令" 后，Descriptions 隐藏，**同位置**渲染 `<Form layout="vertical">`：

```
┌────────────────────────────────────────────────────────────┐
│ 编辑命令                                                    │
├────────────────────────────────────────────────────────────┤
│  commandCode *        [QUERY_CELL_STATUS_________]         │
│  logicalCode *        [MML_QRY_CELL_______________]        │
│  operationType *      [Select: QUERY / EXECUTE / ...  ▾]   │
│  displayName (zh) *   [查询小区状态________________]       │
│  displayName (en) *   [Query Cell Status_____________]     │
│  所属分组 *           [Select: 基站配置 ▾]                 │
│  targetObject         [Device.Services.X_CMCC_LTE._]       │
│  requireConfirm       [Switch: ◯]                          │
│                                                            │
│                              [取消]   [保存]                │
└────────────────────────────────────────────────────────────┘
```

- 校验规则：`commandCode` UPPER_SNAKE_CASE；`logicalCode` 必填；`operationType` 必选
- 保存：调用 `useUpdateCommand`，成功后切回显示态并 `refetch`
- 取消：还原原值，切回显示态
- 标准命令（`catalogProtected===true`）整个表单 readonly，并显示提示"标准命令不可编辑，请克隆为自定义命令"（保留扩展空间，本期不实现克隆）

### 4.4 Path 列表区

下方独立卡片 / Section：

```
┌──────────────────────────────────────────────────────────────┐
│ Path 列表                                          [+ 新增 path]│
├──────────────────────────────────────────────────────────────┤
│ mmlCode │ 显示名     │ 标准 path                │ 默认 │ 支持      │ 操作      │
│ ────────────────────────────────────────────────────────────│
│ CELL_ID │ 小区 ID    │ Device.Services...Cell.X │  ✓   │ 3/3 绿  │ [✎][🗑]│
│ CELL_ST │ 小区状态   │ Device.Services...State  │  ✓   │ 2/3 橙  │ [✎][🗑]│
│ ...                                                          │
└──────────────────────────────────────────────────────────────┘
```

**列定义**（在现有 `CommandDetailDrawer` 列基础上调整）：

| 列 | 字段 | 宽度 | 说明 |
|----|------|------|------|
| mmlCode | `mmlCode` | 160 | path 的 MML 别名 |
| 显示名 | `labelI18n['zh-CN']` | 160 | 多语言显示名 |
| 标准 path | `tr069Path` | flex | TR-069 标准路径（主信息列，code 字体，带 tooltip） |
| 默认勾选 | `defaultSelected` | 80 | ✓/- Tag |
| 支持状态 | 聚合显示 | 130 | `M/N` 三色 Tag，悬停显示来源 |
| 操作 | — | 120 | **编辑** + **删除** 链接 |

**操作按钮**：

- **新增 path**（顶部 primary）：打开 `AddSubFieldsModal`（沿用现有，多选 standardPath 批量添加）
- **编辑** 单行 path：本期新增功能 → 打开 `EditSubFieldModal`（小型 Modal，编辑 `labelI18n` / `defaultSelected` / `displayOrder`，不改 `tr069Path`）
- **删除**：弹 `Modal.confirm`，调用 `useDeleteSubField`

> **现状差异**：当前实现没有单行 path 的"编辑"入口，本次需补齐。

---

## 5. 弹窗与表单（Modals）

| 名称 | 触发位置 | 模式 | 关键字段 |
|------|----------|------|----------|
| `GroupEditorModal` | 顶部 [+ 新增分组]、分组 [⋯] → 编辑 | `create` / `rename` | groupCode, displayName(zh/en), displayOrder |
| `CommandEditorModal`（**新增**） | 分组 [⋯] → 新增命令 | `create` | commandCode, logicalCode, operationType, displayName(zh/en), targetObject, requireConfirm（parentId 由触发位置预填） |
| `AddSubFieldsModal` | path 区 [+ 新增 path] | 沿用 | 批量多选 standardPath |
| `EditSubFieldModal`（**新增**） | path 行 [✎] | `edit` | labelI18n, defaultSelected, displayOrder（**不可改** tr069Path 与 paramId） |
| `Modal.confirm` | 删除场景 | confirm | 显示对象名称 + 危险色按钮 |

### 5.1 GroupEditorModal 改动

- **mode 简化**：原来三种 `create-root` / `create-child` / `rename` → 现在两种 `create` / `rename`
- `create` 模式下 `parentId` 恒为 `null`（呼应 R1：只能建一级分组）
- 文案：`mml.admin.catalog.groups.addRoot` → `mml.admin.catalog.groups.add`（i18n 改名，呼应 R2）

### 5.2 CommandEditorModal（新增组件）

```
┌──────────────────────────────────────┐
│  新增命令 / 编辑命令          [x]    │
├──────────────────────────────────────┤
│  所属分组 *      [Select: 基站配置 ▾]│
│  commandCode *   [______________]    │
│  logicalCode *   [______________]    │
│  operationType * [Select  ▾]         │
│  displayName(zh)*[______________]    │
│  displayName(en)*[______________]    │
│  targetObject    [______________]    │
│  requireConfirm  [Switch: ◯]         │
│                                      │
│              [取消]      [保存]      │
└──────────────────────────────────────┘
```

- 新增时"所属分组" 默认预填触发它的 group.id 且 disabled
- 编辑时（从右栏顶部进入编辑态）不弹 Modal，直接在右栏内联（4.3）

---

## 6. 状态管理与数据流

### 6.1 主数据源

- **左栏树**：复用 `useGroupTree(undefined, 'zh-CN')`
  - 渲染前用 `groups.filter(g => !g.parentId)` 过滤顶层
  - 每个 group 的 `commands` 数组直接挂在 Level 2
- **右栏命令元数据**：直接用左栏选中的 `GroupTreeCommand` 对象（无需二次请求）
- **右栏 path 列表**：`useAdminSubFieldList(command.id)`

### 6.2 选中态管理

页面级状态：

```typescript
interface CatalogPageState {
  selectedKey: string | null;       // 形如 "cmd:xxx" 或 "group:xxx"
  expandedKeys: string[];
  search: string;
  editingCommand: boolean;          // 右栏顶部是否处于编辑态
  pendingFormDirty: boolean;        // 切换前需提示丢弃
}
```

切换命令前若 `pendingFormDirty===true`：

```
Modal.confirm({
  title: '当前命令有未保存的修改,确定离开?',
  okButtonProps: { danger: true },
  onOk: () => { setSelectedKey(newKey); setEditingCommand(false); }
})
```

### 6.3 Mutation 列表

| Mutation Hook | 来源 | 用途 |
|---------------|------|------|
| `useCreateGroup` | 现有 | 新增分组（parentId=null） |
| `useUpdateGroup` | 现有 | 重命名分组 |
| `useDeleteGroup` | **需补齐** | 删除分组（后端需要"分组非空拒绝"语义） |
| `useCreateCommand` | **需补齐** | 新增命令 |
| `useUpdateCommand` | 现有 | 编辑命令元数据 |
| `useDeleteCommand` | 现有 | 删除命令 |
| `useCreateSubField` | 经由 `AddSubFieldsModal` | 批量新增 path |
| `useUpdateSubField` | **需补齐** | 编辑 path 显示名/默认勾选 |
| `useDeleteSubField` | 现有 | 删除 path |

### 6.4 后端 API 契约要点

| 端点 | 期望行为变更 |
|------|--------------|
| `DELETE /api/v1/mml/admin/groups/:id` | 当 `commands.length > 0` 返回 400 + 错误码 `GROUP_NOT_EMPTY`；前端解析后弹 `message.error` |
| `POST /api/v1/mml/admin/commands` | 新建命令，必填 `groupId / commandCode / logicalCode / operationType / displayNameI18n` |
| `PATCH /api/v1/mml/admin/sub-fields/:id` | 单行编辑 path，仅允许改 `labelI18n / defaultSelected / displayOrder` |

后端如尚未实现，需配合 backend 同步开发，PRD 单独立项跟踪（详见 §8）。

---

## 7. 交互细节与边界

### 7.1 拖拽

当前 Tree 支持 group 拖拽排序（同级 / 跨 parent）。**重构后**：

- 仅同级排序（命令所属分组在左栏不通过拖拽改）；命令的 `groupId` 通过编辑命令的"所属分组" Select 字段改
- 仅 Level 1（group）可拖；命令不可拖
- 实现：`draggable.nodeDraggable = (n) => String(n.key).startsWith('group:')`，且 `dropToGap===true` 才接受

### 7.2 受保护对象（catalog_protected）

| 对象 | 受保护时 |
|------|----------|
| 系统分组（ROOT 等） | 编辑/删除按钮 disabled + Tooltip "系统分组不可修改" |
| 标准命令（`catalogProtected=true`） | 命令行 [删除] 灰色 + Tooltip；右栏 [编辑命令] 禁用 |
| 标准 path | path 行 [编辑] [删除] 灰色 + Tooltip |

### 7.3 国际化

需新增 / 调整的 i18n key（`omcmb/frontend-core/src/i18n/`）：

| Key | zh-CN | en-US |
|-----|-------|-------|
| `mml.admin.catalog.groups.add` | 新增分组 | Add Group |
| `mml.admin.catalog.groups.addCommand` | 新增命令 | Add Command |
| `mml.admin.catalog.groups.editGroup` | 编辑分组 | Edit Group |
| `mml.admin.catalog.groups.deleteDisabledTip` | 请先删除该分组下所有命令 | Remove all commands first |
| `mml.admin.catalog.commands.editCommand` | 编辑命令 | Edit Command |
| `mml.admin.catalog.commands.deleteConfirm` | 确认删除命令 {name}? | Delete command {name}? |
| `mml.admin.catalog.commands.discardDirty` | 当前命令有未保存的修改,确定离开? | Discard unsaved changes? |
| `mml.admin.catalog.subField.editTitle` | 编辑 Path | Edit Path |
| `mml.admin.catalog.subField.addCommand` | 添加 Path | Add Paths |
| `mml.admin.catalog.empty.selectCommand` | 请从左侧选择一个命令 | Select a command from the left |
| `mml.admin.catalog.tab.commands` | （废弃，移除） | — |

### 7.4 响应式

- 最小窗口宽度 **1280px** 时左栏 320 + 右栏 ≥ 800（核心场景）
- 窗口宽度 < 1280px 时：左栏可折叠为 60px 图标列（仅显示文件夹图标 + tooltip 显示分组名），点击展开抽屉式叠层

### 7.5 加载与错误态

| 场景 | 表现 |
|------|------|
| 初次加载 group 树 | 左栏 Spin 居中 |
| 右栏切换命令 | path 表格上方显 Spin overlay（保留旧数据避免闪烁） |
| API 失败 | `message.error(err.message)` + 列表保持原状 |
| 删除冲突（GROUP_NOT_EMPTY） | `message.warning('请先删除该分组下所有命令')` |

---

## 8. 文件改动清单

| 文件 | 类型 | 改动 |
|------|------|------|
| `pages/mml/admin/catalog/index.tsx` | 改 | 移除 Tabs，改为左右两栏布局组件 |
| `pages/mml/admin/catalog/GroupsTab.tsx` | **删除** | 内容拆到 `LeftNavTree.tsx` |
| `pages/mml/admin/catalog/CommandsTab.tsx` | **删除** | 不再需要扁平命令列表 |
| `pages/mml/admin/catalog/CommandDetailDrawer.tsx` | 改名 + 改造 | → `RightDetailPanel.tsx`（去掉 Drawer 壳，嵌入页面） |
| `pages/mml/admin/catalog/LeftNavTree.tsx` | **新增** | 左栏导航树组件 |
| `pages/mml/admin/catalog/RightDetailPanel.tsx` | **新增** | 右栏详情面板 |
| `pages/mml/admin/catalog/CommandMetaSection.tsx` | **新增** | 顶部命令元数据展示+编辑 |
| `pages/mml/admin/catalog/PathListSection.tsx` | **新增** | 下方 path 表格 |
| `pages/mml/admin/catalog/GroupEditorModal.tsx` | 改 | mode 简化为 `create`/`rename` |
| `pages/mml/admin/catalog/CommandEditorModal.tsx` | **新增** | 新增命令时使用 |
| `pages/mml/admin/catalog/EditSubFieldModal.tsx` | **新增** | 编辑单行 path |
| `pages/mml/admin/catalog/AddSubFieldsModal.tsx` | 保留 | 批量加 path |
| `frontend-core/src/hooks/api/useMmlAdmin.ts` | 改 | 补 `useDeleteGroup` / `useCreateCommand` / `useUpdateSubField` |
| `frontend-core/src/services/api/mmlAdminApi.ts` | 改 | 补对应端点 |
| `frontend-core/src/i18n/zh-CN/mml.ts` / `en-US/mml.ts` | 改 | 新增 §7.3 列出的 key |
| `__tests__/` | 改+增 | 删除旧 Tab 测试；新增左右栏交互测试 |

### 8.1 后端协同改动（PRD 单列）

- `groupRepo.Delete` 在 `commands.length > 0` 时返回业务错误码 `GROUP_NOT_EMPTY`
- 新增 `POST /api/v1/mml/admin/commands` 接口（若已有覆盖跳过）
- 新增 `PATCH /api/v1/mml/admin/sub-fields/:id` 接口

---

## 9. 操作流程示例（验收用例）

### 9.1 新增分组 → 新增命令 → 加 path

```
1. 顶部 [+ 新增分组]
2. 弹出 GroupEditorModal: 填 groupCode="DEMO_GROUP" / 显示名 "演示分组"
3. 保存,左栏出现新分组 (空,无命令)
4. 鼠标移到 "演示分组",点 [⋯] → "新增命令"
5. 弹出 CommandEditorModal: 所属分组预填 "演示分组" 且 disabled
6. 填 commandCode="DEMO_CMD" / displayName / operationType=QUERY
7. 保存,左栏展开 "演示分组",自动选中新命令
8. 右栏顶部展示命令元数据,下方 path 列表为空 + [+ 新增 path]
9. 点 [+ 新增 path],批量选 3 个 standardPath 加入
10. 列表刷新,显示 3 行 path
```

### 9.2 编辑命令并切换前提示

```
1. 选中命令 A,点右栏顶部 [编辑命令]
2. 改 displayName,不点保存
3. 点左栏命令 B
4. 弹出 Modal.confirm "当前命令有未保存的修改,确定离开?"
5. 点取消 → 仍在命令 A 编辑态
6. 点确定 → 切到命令 B,A 的改动丢弃
```

### 9.3 删除空分组与非空分组

```
场景 A (空分组):
1. 鼠标移到 "演示分组" (假设已无命令)
2. 点 [⋯] → "删除" (启用)
3. 弹 Modal.confirm,点确认 → 分组消失

场景 B (非空分组):
1. 鼠标移到 "基站配置" (含 5 个命令)
2. [⋯] 下拉中的 "删除" 灰色禁用
3. 悬停 → Tooltip "请先删除该分组下所有命令"
4. (兜底) 假设前端校验绕过 → 后端返回 GROUP_NOT_EMPTY
5. 前端 message.warning "请先删除该分组下所有命令"
```

### 9.4 删除命令

```
1. 左栏命令 "查询小区状态" 行内点 [🗑]
2. 弹 Modal.confirm "删除命令 查询小区状态?"
3. 确认 → API 调用 → 命令从左栏移除
4. 若该命令是当前选中,右栏回到空态
```

### 9.5 编辑 path

```
1. 右栏 path 列表中的 "CELL_ID" 行点 [✎]
2. 弹出 EditSubFieldModal: 显示 mmlCode (readonly) / tr069Path (readonly) /
   labelI18n / defaultSelected / displayOrder
3. 改 displayOrder=20,保存
4. path 表格刷新,行序调整
```

---

## 10. 风险与遗留

| 风险 | 缓解 |
|------|------|
| 后端 API 未就绪（`useDeleteGroup` / `useCreateCommand` / `useUpdateSubField`） | 拆 PR：UI 先用 mock，后端跟进；E2E 上线 gate 待 API 联通 |
| 已有多层分组数据被"隐藏" | 增加 admin 调试模式 `?show=all` 暴露原始树，便于数据治理；后续视情况推数据迁移 |
| 标准命令受保护让用户疑惑"为什么不能编辑" | 编辑/删除按钮禁用 + 显式 Tooltip + 文档化"克隆为自定义命令"路径（本期不实现） |
| 切换命令的脏检查可能误判 | 监听 form `onValuesChange`，提交/取消后清 dirty |
| 拖拽排序在过滤后的搜索态下行为奇怪 | 搜索激活时禁用拖拽（`draggable=false`） |

---

## 11. 测试要点

| 测试类型 | 重点 |
|----------|------|
| 单元测试（Vitest） | LeftNavTree 渲染过滤、CommandMetaSection 表单切换、dirty 检测 |
| 组件测试（Testing Library） | 删除分组的启用/禁用矩阵、编辑命令 + 取消 的还原、path 编辑保存 |
| E2E（Playwright） | §9 五个验收用例端到端跑一遍 |
| 视觉回归 | 左右两栏布局在 1280 / 1440 / 1920 三档宽度的稳定性 |

---

## 12. 工时与排期建议

| 阶段 | 工作量 | 备注 |
|------|--------|------|
| 后端 API 补齐 | 1-1.5 d | `useDeleteGroup` 改语义、`useCreateCommand`、`useUpdateSubField` |
| 前端组件拆分 | 2 d | LeftNavTree / RightDetailPanel / CommandMetaSection / PathListSection |
| Modal 重构 | 0.5 d | GroupEditorModal 简化、CommandEditorModal / EditSubFieldModal 新增 |
| i18n + 样式打磨 | 0.5 d | 新 key 翻译、空态、响应式 |
| 测试编写 | 1 d | 单元 + 组件 + E2E |
| **总计** | **5-5.5 d** | 后端可与前端并行 |

---

## 13. 验收清单（DoD）

- [ ] R1：左栏不再渲染子分组层级（仅渲染 `parentId==null` 顶层）
- [ ] R2：i18n key `mml.admin.catalog.groups.add` 已落地，按钮文案显示"新增分组"
- [ ] R3：分组 [⋯] 菜单含编辑+删除，删除按钮的启用条件与 §7.2 一致
- [ ] R4：分组 [⋯] 菜单含"新增命令",触发 CommandEditorModal
- [ ] R5：命令名后/右栏顶部均有"编辑"和"删除"入口
- [ ] R6：原 "命令" 标签下线，path 列表与分组导航在同一页面内通过左右栏完成
- [ ] §9 五个验收用例 E2E 全部通过
- [ ] 标准命令/系统分组的锁定保护 Tooltip 文案就位
- [ ] 后端 `GROUP_NOT_EMPTY` 错误码联调通过
- [ ] frontend-core 多皮肤兼容性确认（webcode-v2 / webcode-v3 可编译）

---

**关联文档**：

- 现有架构：`docs/design/mml-console-architecture-overview-20260521.md`
- 重建计划：`docs/design/mml-rebuild-plan-20260513.md`
- CMCC TDLTE 调整：`docs/design/mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md`
- QA 测试报告：`docs/design/mml-console-qa-test-report-20260523.md`
