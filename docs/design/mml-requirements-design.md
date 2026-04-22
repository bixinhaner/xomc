# OMC MML 维护命令功能需求设计文档

> **文档版本**: v5.0
> **创建日期**: 2026-04-14
> **更新日期**: 2026-04-21
> **适用项目**: OMC（基站网络运营管理系统）
> **技术栈**: Go + Gin + Squirrel + PostgreSQL + React 19 + Ant Design 5 + TanStack Query
> **变更说明**: v5.0 新增第 2 章"数据库结构设计"；mml_templates → mml_custom_command；mml_sub_commands → mml_params + mml_command_params_rel；mml_scripts 新增 status/start_time/end_time/type/progress/result；后端/前端代码同步更新

---

## 1. 文档概述

### 1.1 文档目的

本文档基于新版 MML 页面 UI 截图及完整前后端源码，对 OMC 系统 MML 维护命令模块进行全面、精确的需求设计描述，覆盖每个 UI 元素的组件规格、数据来源、状态管理方式和 API 接口定义，可直接作为前后端开发实施依据。

### 1.2 适用范围

| 子模块 | 路由 | 说明 |
|--------|------|------|
| MML 命令控制台 | `/mml/console` | 三栏布局：设备选择（左）+ 命令树（中）+ 执行面板（右） |
| MML 脚本任务 | `/mml/script` | 任务列表管理、新建任务 Drawer、筛选搜索 |

### 1.3 项目技术栈

| 层次 | 技术 |
|------|------|
| **后端** | Go 1.22+ · Gin 框架 · Squirrel（SQL Builder）· PostgreSQL 16 |
| **权限引擎** | Casbin v2（PostgreSQL 适配器） |
| **前端** | React 19 · TypeScript · Vite · Ant Design 5 · TanStack Query (React Query) |
| **国际化** | i18next（`useT` hook） |
| **设备通信** | TR-069 / CWMP (SOAP over HTTP)，ACS 服务端口 7547 |
| **本地维护** | LMT（Local Management Tool，17547 端口） |

### 1.4 术语定义

| 术语 | 说明 |
|------|------|
| MML | Man-Machine Language，人机语言，设备配置维护命令语言 |
| LMT | Local Management Tool，本地管理工具 |
| TR-069 | CWMP 协议，OMC 远程管理基站的标准协议 |
| ACS | Auto-Configuration Server，即 OMC 的 TR-069 服务端 |
| CPE | Customer Premises Equipment，用户端设备（基站） |
| LST | List，查询操作 → TR-069 `GetParameterValues` |
| MOD | Modify，修改操作 → TR-069 `SetParameterValues` |
| ADD | Add，添加对象 → TR-069 `AddObject` |
| RMV | Remove，删除对象 → TR-069 `DeleteObject` |
| DSP | Display，显示详情 → TR-069 `GetParameterValues` |
| ACT | Activate，激活 → TR-069 `SetParameterValues` |
| DEA | Deactivate，去激活 → TR-069 `SetParameterValues` |
| RST | Reset，复位/重启 → TR-069 `Reboot` 或 `SetParameterValues` |
| CLR | Clear，清除 → TR-069 `SetParameterValues` |
| UPG | Upgrade，升级 → TR-069 `Download` |
| PrivateTemplate | 私有命令模板，仅创建者可见 |
| PublicTemplate | 公共命令模板，所有用户可见 |
| SN | Serial Number，设备唯一序列号 |
| eNB | Evolved NodeB，4G 基站 |
| gNB | Next Generation NodeB，5G 基站 |

---

## 2. 数据库结构设计

MML 模块共 **10 张表**、2 个辅助函数，分为四个功能域。

### 2.1 命令执行域（Command Execution）

#### `mml_commands` — 预定义命令目录

| 列 | 类型 | 说明 |
|----|------|------|
| id | UUID PK | |
| command_name | VARCHAR(200) | 显示名称，如"查询小区信息" |
| command_code | VARCHAR(100) UNIQUE | 命令码，如"LST CELL"、"MOD CELL" |
| category | VARCHAR(50) | 分类编号，引用 sys_dictionary_details |
| rpc_method | VARCHAR(50) | TR-069 RPC：GetParameterValues / SetParameterValues / Reboot / Download |
| operation_type | TEXT DEFAULT 'LST' | 操作类型：LST/MOD/ADD/RMV/ACT/DEA/RST/UPG/CLR/DSP |
| param_template | JSONB | 参数 Schema：键为参数名，值描述类型/范围/默认值 |
| param_paths | JSONB | TR-069 参数路径数组 |
| supported_operations | JSONB | 支持的操作类型数组 |
| help_doc / notes | TEXT | 帮助文档和备注 |
| product_types | JSONB | 适用产品类型，如 ["eNB","gNB"] |

**功能**：定义 MML 控制台可选的命令集合，每条命令绑定 TR-069 RPC 方法和参数模板。

#### `mml_command_params_rel` — 命令-参数 N:M 关联

| 列 | 类型 | 说明 |
|----|------|------|
| command_id | UUID FK → mml_commands (CASCADE) | 命令 ID |
| param_id | UUID FK → mml_params (CASCADE) | 参数 ID |
| sort_order | INT | 显示排序 |
| PK | (command_id, param_id) | |

**关系**：命令直接关联 mml_params，通过此关联表实现 N:M 关系。如 LST DEVICE_INFO 关联 14 个参数；MOD DEVICE_INFO 关联 7 个可写参数。

---

### 2.2 脚本与任务域（Script & Task）

#### `mml_scripts` — 用户脚本库

| 列 | 类型 | 说明 |
|----|------|------|
| id | UUID PK | |
| script_name | VARCHAR(200) | 脚本名称 |
| content | TEXT | 脚本正文（每行一条命令） |
| device_type | VARCHAR(50) | 适用设备类型 |
| creator | VARCHAR(100) | 创建者 |
| tags | JSONB | 标签数组 |
| status | VARCHAR(20) NOT NULL DEFAULT 'active' | 脚本状态：active / archived |
| start_time | TIMESTAMPTZ | 开始执行时间 |
| end_time | TIMESTAMPTZ | 结束执行时间 |
| type | VARCHAR(20) NOT NULL DEFAULT 'manual' | 脚本类型：manual / batch |
| progress | NUMERIC(5,2) NOT NULL DEFAULT 0 | 执行进度百分比 |
| result | JSONB DEFAULT '{}' | 执行结果 |

**功能**：保存用户编写的 MML 命令脚本，可被任务引用。支持执行状态追踪、进度和结果记录。

#### `mml_tasks` — 批量执行任务

| 列 | 类型 | 说明 |
|----|------|------|
| id | UUID PK | |
| task_name | VARCHAR(200) | 任务名称 |
| script_id | UUID FK → mml_scripts (SET NULL) | 关联脚本 |
| device_sns | JSONB | 目标设备 SN 列表 |
| commands | JSONB | 待执行命令对象数组 |
| status | VARCHAR(20) | pending / running / completed / failed / paused / cancelled |
| results | JSONB | 每设备执行结果 |
| executor | VARCHAR(100) | SSE 推送目标用户 |
| execute_type | VARCHAR(20) | immediate / scheduled / periodic / suspended |
| scheduled_at / period_start / period_end / period_time | | 调度参数 |
| offline_retry / failed_retry | | 重试策略 |
| total_devices / success_count / failed_count / result | | 执行统计 |

**关系**：
- `script_id` → `mml_scripts`：一个脚本可触发多个任务
- `device_tasks.parent_task_id` → `mml_tasks`：Fan-out 到设备级子任务（逻辑引用，分区表无 FK）
- 执行完成后 `mml_tasks.results` 记录结果，`mml_audit_log` 记录审计日志

#### `mml_audit_log` — 命令执行审计

| 列 | 类型 | 说明 |
|----|------|------|
| id | UUID PK | |
| task_id | UUID FK → mml_tasks (SET NULL) | 关联任务 |
| command_code | VARCHAR(100) | 执行的命令码 |
| operation_type | VARCHAR(20) | 操作类型 |
| device_sn | VARCHAR(100) | 目标设备 |
| parameters / param_paths | JSONB | 实际发送的参数 |
| result_status / result_message | | 执行结果 |
| duration_ms | INT | 耗时 |

**功能**：每个设备+命令组合生成一条审计记录，用于合规审计和问题追溯。

---

### 2.3 自定义命令域（Custom Command）

#### `mml_custom_command` — 用户自定义命令

| 列 | 类型 | 说明 |
|----|------|------|
| id | UUID PK | |
| command_name | VARCHAR(200) | 自定义命令名称 |
| command_code | VARCHAR(100) | 关联命令码（逻辑引用，无 FK） |
| operation_type | VARCHAR(20) CHECK | LST / MOD / ADD / RMV |
| command_scope | VARCHAR(20) CHECK | 'private' 或 'public' |
| category_group | VARCHAR(50) | 自定义分类目录 |
| parameters | JSONB | 预填参数值 |
| param_paths | JSONB | 参数路径 |
| description | TEXT | 描述 |
| product_types | JSONB | 适用产品类型 |
| creator | VARCHAR(100) | 创建者 |

**功能**：用户可将常用参数组合保存为自定义命令（公开/私有），方便快速执行。公开命令所有人可见，私有命令仅创建者可见。

---

### 2.4 参数库域（Parameter Library）

#### `mml_param_versions` — 参数版本注册表

| 列 | 类型 | 说明 |
|----|------|------|
| version_code | VARCHAR(50) PK | 版本码，如"QB1.0" |
| version_name | VARCHAR(200) | 版本名 |
| product_models | VARCHAR(200)[] | 适用产品型号 |
| software_versions | VARCHAR(100)[] | 适用软件版本 |
| group_count / param_count | INT | 缓存统计 |

**功能**：管理参数库版本，支持多产品多版本并行。

#### `mml_param_groups` — 层级参数分组（自引用树）

| 列 | 类型 | 说明 |
|----|------|------|
| id | UUID PK | |
| group_code | VARCHAR(100) | 分组编码 |
| group_name_zh / group_name_en | | 中英文名称 |
| parent_id | UUID FK → 自身 (CASCADE) | 父分组，自引用树 |
| path | LTREE | 路径（GiST 索引） |
| level | INT CHECK >= 0 | 层级深度 |
| is_listable / is_modifiable / is_addable / is_removable | BOOLEAN | 操作权限标记 |
| param_version | VARCHAR(50) FK → mml_param_versions | 所属版本 |

**功能**：以树形结构组织参数，如"基本信息 → 设备信息 → 设备类型"。LTREE 支持高效的层级查询。

#### `mml_params` — TR-069 参数定义

| 列 | 类型 | 说明 |
|----|------|------|
| id | UUID PK | |
| param_code | VARCHAR(200) | MIB 标识 |
| param_name_zh / param_name_en | | 中英文名称 |
| tr069_path | VARCHAR(1000) | 完整 TR-069 数据模型路径 |
| tr069_path_parts | TEXT[] GENERATED | 自动拆分路径段（GIN 索引） |
| value_type | VARCHAR(50) CHECK | string / enum / unsignedInt / boolean / int 等 |
| value_constraint | JSONB | 范围约束、枚举值 |
| is_writable / is_listable / is_modifiable / is_addable / is_removable | | 操作权限 |
| param_version | VARCHAR(50) FK → mml_param_versions | 所属版本 |

**功能**：TR-069 参数的完整定义，包含类型、约束、路径、权限等元数据。同时作为命令的参数来源（通过 mml_command_params_rel 关联）。

#### `mml_group_param_rel` — 分组-参数 N:M 关联

| 列 | 类型 | 说明 |
|----|------|------|
| id | UUID PK | |
| group_id | UUID FK → mml_param_groups (CASCADE) | |
| param_id | UUID FK → mml_params (CASCADE) | |
| sort_order | INT | 排序 |
| matched_by | VARCHAR(20) | 关联方式：'manual' 或自动匹配规则 |

**功能**：将参数挂载到分组树的叶节点，一个参数可属于多个分组。

---

### 2.5 实体关系总图

```
mml_param_versions (version_code PK)
    │
    ├─ 1:N ── mml_param_groups (param_version FK)
    │             │
    │             ├─ self-ref (parent_id → 自身, LTREE 层级树)
    │             │
    │             └─ N:M ── mml_group_param_rel ── mml_params (param_version FK)
    │                        (group_id, param_id)    │
    │                                                  └─ tr069_path_parts (GENERATED, GIN)
    │
    └─ 1:N ── mml_params (param_version FK)

mml_commands (id PK, command_code UNIQUE)
    │
    ├─ N:M ── mml_command_params_rel ── mml_params (id PK)
    │          (command_id, param_id)
    │
    └─ logical ref ← mml_custom_command.command_code (无 FK)

mml_custom_command (id PK)
    └─ command_scope: private(仅创建者可见) / public(所有人可见)

mml_scripts (id PK)
    │
    └─ 1:N ── mml_tasks (script_id FK, SET NULL)
                  │
                  ├─ 1:N ── mml_audit_log (task_id FK, SET NULL)
                  │
                  └─ logical ref → device_tasks.parent_task_id (分区表, 无 FK)
```

### 2.6 功能映射

| 表 | 实现的功能 |
|----|-----------|
| mml_commands + mml_params + mml_command_params_rel | **命令树**：控制台左侧命令选择，三级（分类→命令→参数） |
| mml_params + mml_param_groups + mml_param_versions + mml_group_param_rel | **参数库**：TR-069 参数的版本化管理，树形浏览，用于"参数路径指定"标签页 |
| mml_scripts | **脚本库**：用户保存的命令脚本，可被任务引用或独立执行 |
| mml_tasks | **任务管理**：批量执行记录，含调度、重试、进度统计 |
| mml_audit_log | **审计日志**：每条命令+设备的执行记录 |
| mml_custom_command | **自定义命令**：用户保存的参数组合，公开/私有 |

---

## 3. 功能总览

### 3.1 模块架构图

```
┌─────────────────────────────────────────────────────────────────────┐
│                         OMC MML 维护模块                             │
│                                                                     │
│  ┌─────────────────────────┐   ┌─────────────────────────────────┐  │
│  │  MML 命令控制台          │   │  MML 脚本任务                   │  │
│  │  /mml/console           │   │  /mml/script                   │  │
│  │                         │   │                                 │  │
│  │  左栏: DeviceTree        │   │  工具栏: + 新增                  │  │
│  │  中栏: CommandTree       │   │  筛选栏: FilterBar               │  │
│  │  右栏: TerminalPanel     │   │  表格: DataTable                 │  │
│  │        + CommandInput    │   │  抽屉: Drawer（新建任务）        │  │
│  └─────────────────────────┘   └─────────────────────────────────┘  │
│                                                                     │
│                      命令执行引擎                                    │
│            POST /api/v1/mml/execute → ACS Worker → 设备             │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.2 页面路由结构

```
/mml
├── /console    ← MML 命令控制台（src/pages/mml/Console/index.tsx）
└── /script     ← MML 脚本任务（src/pages/mml/ScriptTask/index.tsx）
```

菜单路径：**MML 管理 → MML 控制台 / 脚本任务**（参见主页截图左侧导航栏）

![登录后主页 - 左侧导航 MML 管理菜单](/tmp/mml_v2_01_home.png)

### 3.3 与 TR-069/LMT 的关系说明

| 协议 | 端口 | 使用场景 | 特点 |
|------|------|----------|------|
| TR-069/CWMP | 7547 | OMC 远程批量管理基站 | 长会话、异步、支持批量 |
| LMT | 17547 | 本地维护工具 | 临时会话、即时响应 |

MML 命令通过 ACS Worker 封装为 TR-069 SOAP 消息下发到设备：

```
前端 → POST /api/v1/mml/execute
     → 创建 mml_tasks 记录（status=pending）
     → ACS Worker 消费任务
     → 构造 TR-069 SOAP 请求
     → 设备响应
     → 写回 mml_tasks.results
```

---

## 4. MML 命令控制台（/mml/console）

### 4.1 页面布局（三栏布局）

控制台采用三栏弹性布局，列宽比例为 **1fr : 1fr : 2fr**，右栏内部纵向分为终端输出（上 50%）和执行面板（下 50%）。

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  顶部工具栏: [🔲 MML控制台]                                [设备: 0] [命令码]  │
├──────────────────┬──────────────────┬───────────────────────────────────────┤
│     左栏 (1fr)   │     中栏 (1fr)   │                右栏 (2fr)             │
│                  │                  │  ┌─────────────────────────────────┐  │
│   设备名称 (0)   │   命令树          │  │  ● 终端输出         [复制][清][下]│  │
│  [批量输入]      │                  │  │  > 等待命令输出...               │  │
│  ─────────────  │  搜索框           │  │  （深色主题，#0d1117 背景）       │  │
│  搜索框          │  按分类筛选       │  └─────────────────────────────────┘  │
│  按类型筛选      │  ─────────────   │  ┌─────────────────────────────────┐  │
│  ─────────────  │  ▼ 总览 (3)       │  │  当前命令: 未选择  目标设备: 未选  │  │
│  □ 全选 (0/16)  │    基本信息 LST.. │  │  ──── 控制面板 | 参数路径指定 ──  │  │
│                  │    状态信息 LST.. │  │                                 │  │
│  □ ENB00001 eNB │    修改状态 MOD.. │  │  [请先从左侧选择命令]             │  │
│  □ ENB00002 eNB │  ▼ 快速设置 (3)  │  │                                 │  │
│  □ ENB00003 eNB │  ▼ 告警管理 (2)  │  │  > 输入命令，多个命令用分号隔开   │  │
│  ...             │  ▼ 性能统计 (1)  │  │                    [▶ 执行]     │  │
│  ─────────────  │  ▼ 设备控制 (2)  │  │  ─────────────────────────────  │  │
│  < 1 / 2 >      │                  │  │  0 设备 · 未选命令      [C 重置]  │  │
└──────────────────┴──────────────────┴─────────────────────────────────────┘
```

**布局实现**（来源：`Console/index.tsx`）：

```typescript
// 外层容器
display: 'grid'
gridTemplateColumns: '1fr 1fr 2fr'  // 三栏比例

// 右栏内部
height: '50%'   // 终端输出区
flex: 1         // 执行面板区（占剩余空间）
```

![控制台完整页面](/tmp/mml_v2_02_console_full.png)

---

### 4.2 左栏：设备选择面板（DeviceTree）

**组件文件**：`Console/components/DeviceTree.tsx`

**状态管理 Hook**：`Console/hooks/useDeviceSelection.ts`

#### 4.2.1 面板头部

| 元素 | 规格 |
|------|------|
| 标题 | "设备名称" + 已选数量 Badge（蓝色 Tag，数字为 `selectedDevices.length`） |
| 批量输入按钮 | `Button` size="small" type="primary" ghost，图标 `UserAddOutlined`，文本"批量输入"，点击触发 `setBatchSnModalOpen(true)` |

#### 4.2.2 搜索框

| 属性 | 值 |
|------|-----|
| 组件 | `Input` size="small" |
| placeholder | 搜索（来自 i18n `common.search`） |
| 前缀图标 | `SearchOutlined` |
| 功能 | `allowClear`，`onChange` → `onSearchChange`，支持 SN/名称模糊搜索 |

#### 4.2.3 产品类型筛选下拉框

| 属性 | 值 |
|------|-----|
| 组件 | `Select` size="small" style.width="100%" |
| placeholder | "产品类型" |
| allowClear | true |
| 选项来源 | 字典接口 `GET /api/v1/dict/product_type`（数字编码 → 产品名称） |

**产品类型字典值**（`sys_dictionaries` type=`product_type`）：

| label | value（数字编码） |
|-------|-------|
| SmallCell-LTE | 1 |
| gNB-100 | 2 |
| gNB-200 | 3 |
| FAP-LTE-100 | 4 |
| FAP-LTE-200 | 5 |
| FAP-LTE-300 | 6 |

![设备类型筛选下拉框](/tmp/mml_v2_06_device_filter.png)

#### 4.2.4 全选复选框

| 属性 | 值 |
|------|-----|
| 组件 | `Checkbox` |
| checked | `isAllSelected`（`selectedInFiltered === filteredDevices.length`） |
| indeterminate | `isIndeterminate`（部分选中时） |
| onChange | `onToggleSelectAll` |
| 文本 | "全选 (当前页已选/全部过滤数)" |
| disabled | `totalFiltered === 0` 时禁用 |

#### 4.2.5 设备列表

| 属性 | 值 |
|------|-----|
| 组件 | `List` size="small" |
| 数据源 | `paginatedDevices`（当前页，每页 8 条，`DEVICE_PAGE_SIZE = 8`） |
| 选中样式 | 背景 `colorPrimaryBg`，左侧 3px 蓝色边框 |

**每行设备项内容**：

| 元素 | 说明 |
|------|------|
| `Checkbox` | 选中/取消，`onChange` → `onToggleDevice` |
| 状态圆点 | 直径 8px，颜色来自 `STATUS_COLORS`：online=`#52c41a`，offline=`#d9d9d9`，alarm=`#fa8c16` |
| SN（主行） | `fontWeight: 500` fontSize: 12 |
| 设备名称（副行） | `colorTextSecondary`，fontSize: 10，超长省略 |
| 类型 Tag | `device.type`（eNB/gNB/GSM 等），背景 `colorBgLayout` |

#### 4.2.6 分页

| 属性 | 值 |
|------|-----|
| 组件 | `Pagination` size="small" simple |
| 每页条数 | 8（`DEVICE_PAGE_SIZE`） |
| 显示条件 | `totalPages > 1` 时显示 |

#### 4.2.7 已选设备 Tag 展示区

仅当 `selectedDevices.length > 0` 时显示：

| 元素 | 说明 |
|------|------|
| 标题 | "已选设备 (N)" 蓝色文字，右侧删除全部按钮（`DeleteOutlined` + danger） |
| Tag 区域 | `maxHeight: 70px`，溢出滚动，每个 Tag 显示 `{type}-{sn}`，closable，关闭调用 `onRemoveDevice` |

![设备选择左栏](/tmp/mml_v2_03_device_panel.png)

#### 4.2.8 Mock 设备数据（`DEVICE_LIST`）

当前前端使用 Mock 数据（`Console/constants.ts`），共 16 台设备：

| SN | 名称 | 类型 | 产品类型 | 状态 |
|----|------|------|---------|------|
| ENB00001 | 北京朝阳基站01 | eNB | PM-B4860 | online |
| ENB00002 | 北京海淀基站01 | eNB | PM-B4860 | online |
| ENB00003 | 上海浦东基站01 | eNB | QAFA | alarm |
| ENB00004 | 上海徐汇基站01 | eNB | PM-B4860 | online |
| ENB00005 | 广州天河基站01 | eNB | QAFA | online |
| ENB00006 | 深圳南山基站01 | eNB | PM-B4860 | offline |
| ENB00007 | 杭州西湖基站01 | eNB | BaiBNX | online |
| ENB00008 | 南京鼓楼基站01 | eNB | PM-B4860 | alarm |
| GNB00001 | 北京5G基站01 | gNB | BaiBS5163 | online |
| GNB00002 | 北京5G基站02 | gNB | BaiBS5163 | offline |
| GNB00003 | 上海5G基站01 | gNB | BaiBS5263 | online |
| GNB00004 | 广州5G基站01 | gNB | BaiBS5163 | online |
| GNB00005 | 深圳5G基站01 | gNB | BaiBS5263 | alarm |
| GSM00001 | 北京GSM基站01 | GSM | BTS | online |
| GSM00002 | 上海GSM基站01 | GSM | BTS | offline |
| GSM00003 | 广州GSM基站01 | GSM | BSC | online |

---

### 4.3 批量输入弹窗（BatchSnModal）

**组件文件**：`Console/components/BatchSnModal.tsx`

| 属性 | 值 |
|------|-----|
| 弹窗标题（输入阶段） | "批量输入设备SN" |
| 弹窗标题（结果阶段） | "批量添加结果" |
| 宽度 | 480px |
| 输入区说明 | "每行一个SN，或用逗号、分号分隔" |

**输入阶段 UI**：

| 元素 | 规格 |
|------|------|
| 文本框 | `TextArea` rows=8，`fontFamily: monospace`，placeholder: "ENB00001\nENB00002\nENB00003" |
| 底部左侧 | "已选设备: N 台"（`existingSns.size`） |
| 底部右侧 | "待解析: N 个SN"（实时解析 `parsedSns.length`） |
| 确认按钮 | 解析 SN，进入结果阶段 |
| 取消按钮 | 关闭弹窗 |

**SN 解析规则**（`parsedSns` computed）：
```typescript
inputValue.split(/[\n,;]+/).map(s => s.trim().toUpperCase()).filter(Boolean)
```

**结果阶段分类**：

| 分类 | 颜色 | 条件 |
|------|------|------|
| 添加成功 | green | SN 在 `allDeviceSns` 中且不在 `existingSns` 中 |
| 已存在，跳过 | warning | SN 在 `existingSns` 中 |
| 设备不存在 | error | SN 不在 `allDeviceSns` 中 |

**结果阶段按钮**：
- 确认按钮（"完成"）：调用 `onConfirm(result.success)`
- 取消按钮（"继续添加"）：返回输入阶段

![批量输入设备弹窗](/tmp/mml_v2_07_batch_input.png)

---

### 4.4 中栏：命令树面板（CommandTree）

**组件文件**：`Console/components/CommandTree.tsx`

**状态管理 Hook**：`Console/hooks/useCommandSelection.ts`

#### 4.4.1 面板头部

| 元素 | 说明 |
|------|------|
| 标题 | "命令树"（来自 i18n `nav.mml.commands`） |
| 已选命令 Badge | 仅有选中命令时显示，展示 `commandCode`，monospace 字体，蓝色边框圆角 Tag |

#### 4.4.2 搜索框

| 属性 | 值 |
|------|-----|
| placeholder | 搜索（i18n） |
| 功能 | allowClear，过滤 `commandName` 和 `commandCode` |

#### 4.4.3 分类筛选下拉框

| 属性 | 值 |
|------|-----|
| placeholder | "按分类筛选" |
| 选项 | 动态生成自 `categories`（来自 `MOCK_COMMANDS` 的 category 字段集合） |
| allowClear | true |

#### 4.4.4 命令树结构

使用 Ant Design `Tree` 组件，`showLine={{ showLeafIcon: false }}`，默认展开所有分类节点。

**树节点层级**：

```
一级：分类节点（category）
  图标: FolderOutlined（蓝色 colorPrimary）
  标题: {category} ({count}) 粗体
  selectable: false（不可选中）

二级：命令节点（command leaf）
  图标: CodeOutlined（绿色 #52c41a）
  左侧: {commandName} 粗体
  右侧: {commandCode} monospace 小号灰色
  isLeaf: true
```

**当前内置命令分类**（来自 `mml_commands` 种子数据，字典 `mml_command_category`）：

| 分类（字典 value） | 命令数 | 命令列表 |
|------|--------|---------|
| 小区管理 (1) | 5 | 查询小区 `LST CELL`、激活小区 `ACT CELL`、去激活小区 `DEA CELL`、修改小区 `MOD CELL`、重置小区 `RST CELL` |
| 邻区管理 (2) | 3 | 查询邻区 `LST NCELL`、添加邻区 `ADD NCELL`、删除邻区 `DEL NCELL` |
| 基站管理 (3) | 5 | 查询基站状态 `LST BTSSTATE`、查询单板状态 `DSP BOARDSTATUS`、复位基站 `RST BTS`、查询系统资源 `DSP SYSRESOURCE`、查询时钟状态 `DSP CLOCKSTATUS` |
| 告警查询 (4) | 3 | 查询活动告警 `LST ALMAF`、查询历史告警 `LST ALMHIS`、清除告警 `CLR ALM` |
| 性能采集 (5) | 3 | 查询性能计数器 `DSP PERF`、查询性能统计 `LST PM`、查询RRU信息 `DSP RRUINFO` |
| 传输管理 (6) | 3 | 查询传输链路 `DSP LINKSTATUS`、查询SCTP链路 `DSP SCTP`、查询IP地址 `LST IPADDR` |
| 版本管理 (7) | 3 | 查询设备版本 `DSP VERSION`、查询软件包 `LST PKG`、升级软件包 `UPG PKG` |

![命令树中栏](/tmp/mml_v2_04_command_tree.png)

![命令树展开状态（选中高亮）](/tmp/mml_v2_08_cmd_tree_expanded.png)

#### 4.4.5 命令树交互

- 点击命令节点 → 调用 `onSelectCommand(cmd)`，更新 `selectedCommand`
- 已选中命令的节点背景高亮（蓝色）
- 空结果时显示 `Empty` 组件（"暂无匹配的命令"）

---

### 4.5 右栏：执行面板

右栏由两个子区域组成，均封装在独立组件中。

#### 4.5.1 终端输出区（TerminalPanel）

**组件文件**：`Console/components/TerminalPanel.tsx`

**主题**：深色（GitHub Dark 风格）

| 属性 | 值 |
|------|-----|
| 容器背景 | `#0d1117` |
| 字体 | `JetBrains Mono, SFMono-Regular, Consolas, monospace` |
| 字号 | 12px，行高 1.7 |
| 默认文字颜色 | `#7ee787`（绿色） |

**工具栏**（顶部渐变 `#21262d → #161b22`，底部 `#30363d` 分隔线）：

| 元素 | 说明 |
|------|------|
| 状态指示灯 | 12×12px 圆形，颜色 `#238636`，绿色辉光 |
| 标题 | "终端输出"，`#c9d1d9` 颜色 |
| 复制按钮 | `CopyOutlined`，点击调用 `navigator.clipboard.writeText` |
| 清空按钮 | `ClearOutlined`，调用 `onClear` |
| 下载按钮 | `DownloadOutlined`，调用 `onDownload`（下载为 `.txt` 文件） |

**输出行颜色编码**（`LINE_COLORS`）：

| type | 颜色 | 含义 |
|------|------|------|
| stdout | `#52C41A` | 标准输出（绿色） |
| stderr | `#F5222D` | 错误输出（红色） |
| info | `#91D5FF` | 信息（浅蓝色） |
| success | `#B7EB8F` | 成功（浅绿色） |

**空状态**：显示 `> 等待命令输出...`（灰色 `#484f58`）

![终端输出区域](/tmp/mml_v2_09_terminal.png)

**终端输出行结构**（`TerminalLine`）：

```typescript
interface TerminalLine {
  text: string;
  type?: 'stdout' | 'stderr' | 'info' | 'success';
  timestamp?: string;  // 格式: HH:mm:ss，可选前缀展示
}
```

**自动滚动**：每次 `lines` 变化时，`containerRef.current.scrollTop = scrollHeight`。

#### 4.5.2 执行面板（CommandInput）

**组件文件**：`Console/components/CommandInput.tsx`

**面板头部信息区**（`Descriptions` 双列）：

| 标签 | 内容 |
|------|------|
| 当前命令 | 选中命令的 `commandCode`（code 样式，蓝色边框），未选时显示"未选择" |
| 目标设备 | "N 台"（蓝色），未选时显示"未选择" |

**Tab 切换**：

| Tab key | 标签 | 说明 |
|---------|------|------|
| `control` | 控制面板 | 参数动态表单 |
| `paramPath` | 参数路径指定 | TR-069 路径手动输入 |

---

#### 4.5.3 控制面板 Tab（参数动态表单）

**组件**：`Form` layout="vertical" size="small"

**渲染规则**（遍历 `selectedCommand.params`）：

| 参数类型（`MMLParamType`） | 渲染组件 | 特殊规格 |
|--------------------------|----------|---------|
| `enum` | `Select` | 选项来自 `param.options`，required 时 allowClear=false |
| `number` | `InputNumber` style.width="100%" | min=`param.minValue`，max=`param.maxValue` |
| `string`（默认） | `Input` | - |
| `boolean` | `Select`（true/false 选项） | - |

**必填标识**：标签文字右侧红色 `*`（`param.required === true`）

**参数 Tooltip**：`Form.Item tooltip={param.description}`

**无参数命令**：显示"该命令无需配置参数"居中文字

**未选命令**：显示"请先从左侧选择命令"居中文字

![参数配置区域](/tmp/mml_v2_10_params.png)

---

#### 4.5.4 参数路径指定 Tab

**操作类型下拉框**：

| 属性 | 值 |
|------|-----|
| 标签 | "操作类型" |
| 组件 | `Select` style.width="100%" |
| 默认值 | `LST` |
| 状态变量 | `operationType`（本地 `useState`） |

**操作类型选项**（`OPERATION_TYPE_OPTIONS`）：

| label | value |
|-------|-------|
| LST - 查询 | LST |
| MOD - 修改 | MOD |
| ADD - 增加 | ADD |
| RMV - 删除 | RMV |

![参数路径指定 Tab - 操作类型下拉](/tmp/mml_v2_params_path.png)

![操作类型选中 LST-查询](/tmp/mml_v2_11_op_type.png)

**参数路径列表**：

| 元素 | 说明 |
|------|------|
| 标题 | "参数路径" + "添加路径"按钮（`Button` type="dashed" icon=`PlusOutlined`） |
| 路径项 | 序号（1.2.3...）+ `Input` size="small" + 删除按钮（`MinusCircleOutlined`） |
| 删除限制 | `paramPaths.length <= 1` 时禁用删除按钮 |
| placeholder | "例如: Device.Services.FAPService.1.CellConfig.1" |

**帮助提示块**：
```
提示
• 参数路径支持 TR-069 参数树路径格式
```

---

#### 4.5.5 命令输入栏（控制面板 Tab 专属）

| 元素 | 说明 |
|------|------|
| 输入框前缀 | `>` 蓝色 monospace 粗体 |
| placeholder | "输入命令，多个命令用分号隔开" |
| 字体 | `'SFMono-Regular', Consolas, monospace` |
| 快捷键 | `Ctrl+Enter` / `Cmd+Enter` 触发执行 |
| 执行按钮 | `Button` type="primary" icon=`PlayCircleOutlined` "执行" |
| 禁用条件 | `selectedDevices.length === 0 || !selectedCommand` |

#### 4.5.6 底部操作栏

| 元素 | 说明 |
|------|------|
| 状态文字（左侧） | "N 设备 · 命令码"（控制面板 Tab），参数路径 Tab 不显示 |
| 执行按钮 | 参数路径 Tab 时在底部显示，复用 `handleExecute` |
| 重置按钮 | `ReloadOutlined`，调用 `onReset`（清空设备/命令/输出/参数） |
| 保存脚本按钮 | `SaveOutlined`，仅控制面板 Tab 显示，`onSaveScript` 回调 |

#### 4.5.7 危险命令确认弹窗

当执行以下命令前，弹出二次确认 `Modal`：

| 命令模式 | 名称 | 描述 |
|---------|------|------|
| `/RST/i` | 重启 | 此操作将重启设备，设备会暂时断开连接 |
| `/FACTORYRESET/i` | 恢复默认配置 | 此操作将恢复设备出厂设置，所有配置将被清除 |
| `/CELLDEACTIVATE/i` | 小区去激活 | 此操作将去激活小区，可能影响网络服务 |
| `/RFCTXOFF/i` | 关闭小区射频 | 此操作将关闭小区射频发射，会影响无线信号 |
| `/COLDREBOOT/i` | 冷重启 | 此操作将执行设备冷重启，设备会完全断电重启 |

确认弹窗规格：宽度 420px，okButtonProps.danger=true，显示命令名称、描述和目标设备数。

---

### 4.6 组件层次与数据流

#### 4.6.1 React 组件树

```
MMLConsole (index.tsx)
├── 顶部工具栏（内联 JSX）
│   ├── AppstoreOutlined 图标 + 标题
│   └── 状态 Tag（已选设备数 + 当前命令码）
├── Card → DeviceTree (components/DeviceTree.tsx)
│   └── props: selectedDevices, filteredDevices, paginatedDevices,
│              searchText, productTypeFilter, currentPage,
│              isAllSelected, isIndeterminate, totalFiltered, totalPages,
│              onSearchChange, onFilterChange, onPageChange,
│              onToggleDevice, onToggleSelectAll, onRemoveDevice,
│              onClearSelection, onBatchInput
├── Card → CommandTree (components/CommandTree.tsx)
│   └── props: selectedCommand, commands, categories,
│              commandsByCategory, searchText, categoryFilter,
│              onSearchChange, onFilterChange, onSelectCommand
│   └── 内嵌 AddTemplateModal (components/AddTemplateModal.tsx)
│       └── 模板创建/编辑弹窗，支持 private/public 作用域
├── 右栏 div
│   ├── TerminalPanel (components/TerminalPanel.tsx)
│   │   └── props: lines, onClear, onDownload
│   └── CommandInput (components/CommandInput.tsx)
│       ├── Tab: Control Panel → ParamFormRenderer (components/ParamFormRenderer.tsx)
│       │   └── 根据操作类型（LST/MOD/ADD/RMV/ACT/DEA/RST/CLR/UPG）动态渲染参数表单
│       ├── Tab: ParameterPath → ParamPathPanel (components/ParamPathPanel.tsx)
│       │   └── TR-069 参数路径编辑面板，支持操作类型切换和路径增删
│       └── props: selectedDevices, selectedCommand, paramValues,
│                  onParamChange, onExecute, onReset, loading
└── BatchSnModal (components/BatchSnModal.tsx)
    └── props: open, onClose, onConfirm, existingSns, allDeviceSns
```

#### 4.6.2 状态管理（Hooks）

| Hook | 文件 | 管理的状态 |
|------|------|---------|
| `useDeviceSelection` | `hooks/useDeviceSelection.ts` | 设备选择、搜索、筛选、分页 |
| `useCommandSelection` | `hooks/useCommandSelection.ts` | 命令选择、搜索、分类过滤 |
| `useCommandExecution` | `hooks/useCommandExecution.ts` | 命令执行、输出行、loading 状态 |
| `useState` | `index.tsx` | `batchSnModalOpen`、`paramValues` |

#### 4.6.3 数据流向图

```
用户选择设备                     用户选择命令
    │                                │
    ▼                                ▼
useDeviceSelection               useCommandSelection
 selectedDevices[]                selectedCommand
    │                                │
    └────────────────┬───────────────┘
                     ▼
              CommandInput
         (handleExecute 触发)
                     │
                     ▼
          useCommandExecution
          executeCommand(devices, command, params)
                     │
                     ▼
          useExecuteMMLCommand (useMutation)
          POST /api/v1/mml/execute
                     │
                     ▼
          outputLines → TerminalPanel
```

---

### 4.7 API 接口设计

> **当前已注册 24 个 API 端点**（MML 主模块 20 个 + 参数库 4 个），MML 主模块在 `omcgo/internal/mml/handler.go` 中实现，参数库在 `omcgo/internal/mml/param_handler.go` 中实现，均通过 `permGroup("devices")` 注册到 `/api/v1/mml/` 路径下。

#### 4.7.0 完整 API 端点一览

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/mml/commands` | 获取命令列表（分页+搜索+分类过滤） |
| GET | `/mml/commands/:id` | 获取单个命令详情 |
| GET | `/mml/commands/:id/param-paths` | 获取命令的 TR-069 参数路径 |
| POST | `/mml/execute` | 执行 MML 命令（创建任务） |
| GET | `/mml/dangerous-check` | 检查命令码是否为危险命令 |
| GET | `/mml/scripts` | 获取脚本列表 |
| POST | `/mml/scripts` | 创建脚本 |
| GET | `/mml/scripts/:id` | 获取脚本详情 |
| PUT | `/mml/scripts/:id` | 更新脚本 |
| DELETE | `/mml/scripts/:id` | 删除脚本 |
| GET | `/mml/tasks` | 获取任务列表（分页+状态/类型/结果过滤） |
| GET | `/mml/tasks/:id` | 获取任务详情 |
| GET | `/mml/tasks/:id/results` | 获取任务执行结果明细（分页） |
| POST | `/mml/tasks/:id/start` | 启动挂起/暂停的任务 |
| POST | `/mml/tasks/:id/pause` | 暂停运行中的任务 |
| POST | `/mml/tasks/:id/cancel` | 取消任务 |
| DELETE | `/mml/tasks/:id` | 删除任务（仅非运行态） |
| GET | `/mml/templates` | 获取模板列表（分页+命令码/操作类型/作用域过滤） |
| POST | `/mml/templates` | 创建模板 |
| GET | `/mml/templates/:id` | 获取模板详情 |
| PUT | `/mml/templates/:id` | 更新模板 |
| DELETE | `/mml/templates/:id` | 删除模板 |
| POST | `/mml/templates/:id/clone` | 克隆模板为私有副本 |
| GET | `/mml/param-versions` | 获取参数库版本列表 |
| GET | `/mml/param-versions/:version/groups` | 获取版本的参数分组树 |
| GET | `/mml/param-versions/:version/groups/:groupId/params` | 获取分组的参数列表 |
| GET | `/mml/param-versions/:version/params` | 搜索参数（支持 search/tr069_path 过滤） |

#### 4.7.1 获取命令列表

```
GET /api/v1/mml/commands
```

**Query 参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页数量，默认 20；`getAllCommands` 传 1000 |
| category | string | 否 | 分类过滤 |
| search | string | 否 | 关键词搜索（命令名/编码） |

**响应体**：

```json
{
  "items": [
    {
      "id": "uuid",
      "command_name": "基本信息",
      "command_code": "LST BASIC_INFO",
      "category": "总览",
      "description": "查询设备基本信息",
      "rpc_method": "GetParameterValues",
      "param_template": null,
      "product_types": ["eNB", "gNB", "GSM"],
      "created_at": "2026-01-01T00:00:00Z"
    }
  ],
  "total": 11,
  "page": 1,
  "page_size": 20,
  "total_pages": 1
}
```

**前端映射函数**（`mmlApi.ts` → `mapBackendCommand`）：

```typescript
// snake_case → camelCase
command_name → commandName
command_code → commandCode
param_template（JSONB）→ MMLParam[]（通过 mapParamTemplate 函数解析）
product_types（null 时取 []）→ productTypes
```

#### 4.7.2 获取单个命令详情

```
GET /api/v1/mml/commands/:id
```

响应体：同列表中单条命令结构（`BackendMMLCommand`）。

#### 4.7.3 执行 MML 命令

```
POST /api/v1/mml/execute
```

**请求体**（`ExecuteHTTPRequest`）：

```json
{
  "command_code": "LST CELL",
  "device_sns": ["ENB00001", "ENB00002"],
  "parameters": {"CELLID": 1},
  "task_name": "查询小区信息_2026-04-17",
  "script_id": "",
  "commands": [],
  "execute_type": "immediate",
  "scheduled_at": "",
  "period_start": "",
  "period_end": "",
  "period_time": "",
  "offline_retry": false,
  "offline_retry_wait": 60,
  "failed_retry": false,
  "failed_retry_count": 3,
  "failed_retry_interval": 5,
  "param_paths": ["Device.Services.FAPService.{i}.CellConfig.{i}"],
  "operation_type": "LST"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| command_code | string | 否 | 命令编码（与 `script_id`/`commands` 二选一）|
| device_sns | string[] | **是** | 目标设备 SN 列表 |
| parameters | object | 否 | 命令参数键值对 |
| task_name | string | 否 | 自定义任务名称，后端从 context 取 creator |
| script_id | string | 否 | 脚本 ID（传入后自动解析脚本内容为命令列表） |
| commands | []map | 否 | 直接传入命令列表 |
| execute_type | string | 否 | 执行类型：`immediate`/`scheduled`/`periodic`/`suspended`，默认 `immediate` |
| scheduled_at | string | 否 | 定时执行时间（RFC3339） |
| period_start/end/time | string | 否 | 周期任务时间窗口 |
| offline_retry | bool | 否 | 离线设备是否等待重试 |
| offline_retry_wait | int | 否 | 离线重试等待秒数，默认 60 |
| failed_retry | bool | 否 | 失败是否重试 |
| failed_retry_count | int | 否 | 最大重试次数，默认 3 |
| failed_retry_interval | int | 否 | 重试间隔秒数，默认 5 |
| param_paths | string[] | 否 | TR-069 参数路径列表 |
| operation_type | string | 否 | 操作类型：LST/MOD/ADD/RMV/ACT/DEA/RST/CLR/UPG |

**响应体**（返回创建的 `MMLTask`）：

```json
{
  "id": "task-uuid",
  "task_name": "查询设备基本信息_2026-04-14",
  "device_sns": ["ENB00001", "ENB00002"],
  "commands": [{"command_code": "LST BASIC_INFO", "parameters": {}}],
  "status": "pending",
  "results": [],
  "creator": "admin",
  "created_at": "2026-04-14T10:00:00Z",
  "updated_at": "2026-04-14T10:00:00Z"
}
```

> **重要**：当前执行为**异步任务**，不立即返回结果。前端需轮询 `GET /api/v1/mml/tasks/:id` 获取最终结果。

**前端执行流程**（`useCommandExecution.ts`）：

```
1. 遍历 devices[]，逐台设备调用 executeMutation.mutateAsync
2. 每台设备调用前，在 outputLines 中插入 info 行（设备 SN 标头）
3. 调用成功后，插入 stdout 行（设备名、类型、状态）
4. result.success=true → success 行；false → stderr 行
5. 最终插入 "✓ 命令执行完成，共处理 N 台设备" 的 success 行
6. catch 异常 → 插入 stderr 行
```

---

### 4.8 数据模型

#### 4.8.1 mml_commands 表

```sql
CREATE TABLE mml_commands (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    command_name         VARCHAR(200) NOT NULL,       -- 命令名称（中文显示名）
    command_code         VARCHAR(100) NOT NULL UNIQUE,-- 命令编码（如 LST CELL）
    category             VARCHAR(50),                 -- 命令分类（字典编码：1=小区管理 2=邻区管理 ...）
    description          TEXT,                        -- 命令描述
    rpc_method           VARCHAR(50) NOT NULL,        -- TR-069 RPC 方法名
    operation_type       TEXT DEFAULT 'LST',          -- 操作类型（LST/MOD/ADD/RMV/ACT/DEA/RST/CLR/UPG）
    param_template       JSONB,                       -- 参数模板（见下方结构定义）
    param_paths          JSONB DEFAULT '[]',          -- TR-069 参数路径列表
    supported_operations JSONB DEFAULT '["LST"]',     -- 支持的操作类型列表
    help_doc             TEXT DEFAULT '',              -- 命令帮助文档
    notes                TEXT DEFAULT '',              -- 使用注意事项
    product_types        JSONB DEFAULT '[]',          -- 适用产品类型列表
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- GIN 索引（支持 param_template 和 product_types 的 JSON 查询）
CREATE INDEX IF NOT EXISTS idx_mml_commands_param_template_gin
    ON mml_commands USING GIN (param_template);
CREATE INDEX IF NOT EXISTS idx_mml_commands_product_types_gin
    ON mml_commands USING GIN (product_types);
```

**param_template JSONB 结构**（对应前端 `MMLParam[]`）：

```json
{
  "ALARM_LEVEL": {
    "type": "enum",
    "required": false,
    "description": "告警级别",
    "options": [
      {"label": "紧急", "value": 1},
      {"label": "重要", "value": 2},
      {"label": "一般", "value": 3},
      {"label": "提示", "value": 4}
    ]
  },
  "FREQ": {
    "type": "number",
    "required": true,
    "description": "频点",
    "min_value": 0,
    "max_value": 65535
  }
}
```

#### 4.8.2 前端类型定义（`src/types/mml.ts`）

```typescript
export type MMLParamType = 'string' | 'number' | 'boolean' | 'enum' | 'range' | 'ipAddress' | 'list' | 'unsignedInt';

export type MMLOperationType = 'LST' | 'MOD' | 'ADD' | 'RMV' | 'DSP' | 'ACT' | 'DEA' | 'RST' | 'CLR' | 'UPG';

export interface MMLParam {
  name: string;
  type: MMLParamType;
  required: boolean;
  defaultValue?: string | number | boolean;
  description: string;
  options?: Array<{ label: string; value: string | number }>;
  minValue?: number;
  maxValue?: number;
  pattern?: string;
  suggestedValue?: string | number;
  unit?: string;
  restartRequired?: boolean;
  helpText?: string;
  order?: number;
}

export interface MMLCommand {
  id: string;
  commandName: string;
  commandCode: string;
  category: string;
  description: string;
  operationType: MMLOperationType;
  params: MMLParam[];
  paramPaths: string[];
  supportedOperations: string[];
  helpDoc: string;
  notes: string;
  productTypes: string[];
}
```

#### 4.8.3 控制台本地类型（`Console/types.ts`）

```typescript
// 设备状态颜色
export const STATUS_COLORS: Record<string, string> = {
  online: '#52c41a',
  offline: '#d9d9d9',
  alarm: '#fa8c16',
};

// 终端行类型
export interface TerminalLine {
  text: string;
  type?: 'stdout' | 'stderr' | 'info' | 'success';
  timestamp?: string;
}
```

---

## 5. MML 脚本任务（/mml/script）

### 5.1 页面布局

采用 `ListPageLayout` 组件（标准列表页布局）：

```
┌──────────────────────────────────────────────────────────────────────────┐
│  脚本任务                                                  [+ 新增]        │
├──────────────────────────────────────────────────────────────────────────┤
│  [任务名称输入] [开始时间范围] [类型▼] [状态▼] [结果▼]   [重置] [查询]    │
├──────────────────────────────────────────────────────────────────────────┤
│                               表格工具栏 [刷新][列设置][全屏][清空]        │
├──────────────────────────────────────────────────────────────────────────┤
│  操作 │ 任务名称 │ 创建者 │ 创建时间 │ 类型 │ 状态 │ 进度 │ 结果 │ 开始时间 │ 结束时间 │
│  👁 ⋮ │ ...      │ admin  │ ...      │ Tag  │ Tag  │ ...  │ Tag  │ ...      │ ...      │
│  ...  │ ...      │ ...    │ ...      │      │      │      │      │          │          │
├──────────────────────────────────────────────────────────────────────────┤
│                                      共 1 条   < 1 >   20 条/页 ▼         │
└──────────────────────────────────────────────────────────────────────────┘
```

![脚本任务完整页面](/tmp/mml_v2_12_script_full.png)

---

### 5.2 搜索筛选区域

使用 `FilterBar` 组件，`filterId="mml-script-task"`。

**筛选字段定义**（`filterFields`）：

| name | label | type | placeholder/options |
|------|-------|------|---------------------|
| taskName | 任务名称 | input | "请输入任务名称" |
| startTime | 开始时间 | date-range | - |
| createStatus | 类型 | select | 全部/立即执行/挂起/定时执行/周期任务 |
| taskStatus | 状态 | select | 全部/等待中/执行中/已暂停/已完成/已终止/异常 |
| taskResult | 结果 | select | 全部/成功/部分成功/失败 |

![搜索筛选区域](/tmp/mml_v2_search_filter.png)

![任务类型筛选](/tmp/mml_v2_task_type_filter.png)

![任务状态筛选](/tmp/mml_v2_task_status_filter.png)

![任务结果筛选](/tmp/mml_v2_task_result_filter.png)

**筛选逻辑**（前端本地过滤 `filteredData`）：

```typescript
// taskName: 包含匹配（toLowerCase）
// startTime: 时间范围过滤 row.START_TIME
// createStatus: 精确匹配 row.CREATE_STATUS
// taskStatus: 精确匹配 row.TASK_STATUS
// taskResult: 精确匹配 row.TASK_RESULT
```

---

### 5.3 工具栏

| 元素 | 说明 |
|------|------|
| "+ 新增" 按钮 | `Button` type="primary" icon=`PlusOutlined`，点击 `openAddModal` |

![工具栏区域](/tmp/mml_v2_13_script_toolbar.png)

---

### 5.4 任务列表表格

使用 `DataTable<ScriptRow>` 组件，`tableId="script-task"`，`scroll={{ x: 1400 }}`。

#### 5.4.1 列定义

| key | 标题 | dataIndex | 宽度 | 说明 |
|-----|------|-----------|------|------|
| operation | 操作 | TASK_ID | 70px | fixed='left'，渲染操作按钮 |
| TASK_NAME | 任务名称 | TASK_NAME | 自适应（ellipsis）| - |
| CREATE_USER | 创建者 | CREATE_USER | 100px | - |
| CREATE_TIME | 创建时间 | CREATE_TIME | 160px | 格式 YYYY-MM-DD HH:mm:ss |
| CREATE_STATUS | 类型 | CREATE_STATUS | 100px | Tag，颜色见下方 |
| TASK_STATUS | 状态 | TASK_STATUS | 100px | Tag，颜色见下方 |
| TASK_PROGRESS | 进度 | TASK_PROGRESS | 80px | 字符串如 "45%" |
| TASK_RESULT | 结果 | TASK_RESULT | 100px | Tag，颜色见下方 |
| START_TIME | 开始时间 | START_TIME | 140px | - |
| END_TIME | 结束时间 | END_TIME | 140px | - |

#### 5.4.2 类型 Tag（CREATE_STATUS_MAP）

| value | text | color |
|-------|------|-------|
| active | 立即执行 | green |
| suspend | 挂起 | orange |
| timing | 定时执行 | blue |
| period | 周期任务 | purple |

#### 5.4.3 状态 Tag（TASK_STATUS_MAP）

| value | text | color |
|-------|------|-------|
| waiting | 等待中 | default |
| running | 执行中 | processing |
| paused | 已暂停 | warning |
| completed | 已完成 | success |
| terminated | 已终止 | error |
| exception | 异常 | error |

#### 5.4.4 结果 Tag（TASK_RESULT_MAP）

| value | text | color |
|-------|------|-------|
| success | 成功 | success |
| partial | 部分成功 | warning |
| failed | 失败 | error |

![任务列表表格](/tmp/mml_v2_14_task_table.png)

#### 5.4.5 操作列按钮

操作列渲染两个按钮：

| 按钮 | 图标 | 说明 |
|------|------|------|
| 查看结果 | `EyeOutlined` | type="link"，调用 `showResult(record)`，打开结果弹窗 |
| 更多操作 | `MoreOutlined` | `Dropdown` trigger=['click']，展开下拉菜单 |

**下拉菜单项**（`getActionMenu(row)`）：

| key | 图标 | 标签 | 禁用条件 | 操作 |
|-----|------|------|---------|------|
| info | `InfoCircleOutlined` | 信息 | 始终可用 | `viewTaskInfo(row)` |
| start | `PlayCircleOutlined` | 开始 | status !== 'paused' | `startTask(row)` |
| wait | `PauseCircleOutlined` | 暂停 | status !== 'running' | `suspendTask(row)` |
| end | `StopOutlined` | 终止任务 | !['running','waiting'].includes(status) | `terminateTask(row)` |
| del | `DeleteOutlined` | 删除（danger） | status === 'running' | `deleteTask(row)` |

---

### 5.5 新建任务 Drawer

**组件**：`Drawer` 宽度 560px，`destroyOnClose`，标题"新建MML脚本任务"

![新建任务表单](/tmp/mml_v2_15_new_task.png)

#### 5.5.1 基本信息分区

**标识**：蓝色竖条 + "基本信息" 文字

| 字段 | 组件 | 验证 | 说明 |
|------|------|------|------|
| 任务名称（必填）| `Input` maxLength=50 | required | placeholder "请输入新建任务名称" |
| 选择脚本（必填）| `Upload` + 下载模板链接 | required（文件名非空）| 仅支持 `.txt` 格式，maxCount=1 |

**脚本选择详细规格**：
- 上传按钮：`Button` icon=`UploadOutlined` "选择文件"
- 格式说明：灰色文字"( 仅支持 .txt 格式 )"
- 模板导入提示：灰色文字 "使用模板导入提示：支持使用模板导入" + `Button` type="link" icon=`DownloadOutlined` "导出模板"

```typescript
// Upload 配置
accept: ".txt"
beforeUpload: (file) => {
  setFileList([file])
  addForm.setFieldValue('fileName', file.name)
  return false  // 阻止自动上传
}
```

#### 5.5.2 执行方式分区

**标识**：蓝色竖条 + "选择执行方式" 文字

使用 `Radio.Group` name="status"，默认值 `active`：

| Radio 值 | 显示文字 | 附加控件 |
|---------|---------|---------|
| active | 立即执行 | 无 |
| suspend | 挂起 | 无 |
| timing | 定时执行 | `DatePicker` showTime，禁选过去时间，宽度 185px，格式 "YYYY-MM-DD HH:mm:ss" |
| period | 周期任务 | `DatePicker.RangePicker`（禁选过去，宽度 240px）+ `:`分隔 + `DatePicker.TimePicker` format="HH:mm:ss" 宽度 110px |

> 周期任务的 Radio 使用独立 `Form.Item`（使用受控模式 `checked={executeType === 'period'}`）

#### 5.5.3 执行策略分区

**标识**：蓝色竖条 + "执行策略" 文字

**离线设备策略**（同行布局）：

| 元素 | 说明 |
|------|------|
| 标签 | "离线设备" |
| Checkbox | name="offlineRetryEnable"，"等待设备上线重试" |
| InputNumber | name="offlineRetryWaitTime"，min=20，max=10080，宽度 80px |
| 单位 | "分钟" |

**在线设备策略**（同行布局）：

| 元素 | 说明 |
|------|------|
| 标签 | "在线设备" |
| Checkbox | name="failedRetryEnable"，"配置失败重试" |
| InputNumber | name="failedRetryCount"，min=1，宽度 70px |
| 文字 | "间隔次数重试" |
| InputNumber | name="failedRetryWaitTime"，min=1，宽度 70px |
| 单位 | "分钟" |

**表单默认值**（`openAddModal` 时设置）：

```typescript
addForm.setFieldsValue({
  status: 'active',
  offlineRetryEnable: false,
  offlineRetryWaitTime: 60,
  failedRetryEnable: false,
  failedRetryCount: 3,
  failedRetryWaitTime: 5,
})
```

**Drawer 底部按钮**：
- "取消"：关闭 Drawer
- "确定"：`addForm.validateFields()` → `handleAddTask()`

#### 5.5.4 AddTaskForm 接口定义

```typescript
interface AddTaskForm {
  taskName: string;
  fileName: string;
  status: 'active' | 'suspend' | 'timing' | 'period';
  time: Dayjs | null;              // 定时执行时间
  periodStartTime: Dayjs | null;   // 周期开始（含在 periodDateRange 内）
  periodEndTime: Dayjs | null;     // 周期结束
  periodTime: Dayjs | null;        // 周期执行时刻
  offlineRetryEnable: boolean;
  offlineRetryWaitTime: number;    // 分钟，默认 60
  failedRetryEnable: boolean;
  failedRetryCount: number;        // 次数，默认 3
  failedRetryWaitTime: number;     // 分钟，默认 5
}
```

---

### 5.6 任务详情弹窗

| 属性 | 值 |
|------|-----|
| 标题 | "任务详情" |
| 宽度 | 680px |
| footer | null（只读展示）|

**展示字段**：任务名称、创建者、创建时间、类型（文字）、状态（文字）、进度、结果（文字）、开始时间、结束时间

---

### 5.7 任务状态流转

```
                          ┌───────────────┐
                          │   pending     │ ← immediate 创建时
                          └───────┬───────┘
                                  │ 自动启动（immediate）/ 用户手动 start / 到达定时时间
                                  ▼
             创建挂起任务 ──► running ◄── 周期任务再次触发
              (paused)           │
               │    ┌────────────┼────────────────┐
               │    ▼            ▼                ▼
               └─► completed   cancelled         failed
                  （全部完成） （用户取消）        （执行异常）
                        ▲
                        │
               paused ←─┘ （用户暂停 running）
                  │
                  └──► running （用户恢复 start）

后端状态枚举（omcgo/internal/mml/model.go）：
  TaskPending   = "pending"    -- 初始态（immediate 创建时）
  TaskRunning   = "running"    -- 执行中
  TaskCompleted = "completed"  -- 已完成
  TaskFailed    = "failed"     -- 执行失败
  TaskPaused    = "paused"     -- 已暂停（或 suspended 方式创建）
  TaskCancelled = "cancelled"  -- 已取消

前端状态枚举（omcmb/webcode/src/types/mml.ts）：
  MMLTaskStatus = 'pending' | 'running' | 'completed' | 'failed' | 'paused' | 'cancelled'

执行类型枚举：
  ExecuteImmediate = "immediate"   -- 立即执行
  ExecuteScheduled = "scheduled"   -- 定时执行
  ExecutePeriodic  = "periodic"    -- 周期执行
  ExecuteSuspended = "suspended"   -- 挂起（创建时 status=paused）

任务结果枚举：
  ResultSuccess = "success"  -- 全部成功
  ResultPartial = "partial"  -- 部分成功
  ResultFailed  = "failed"   -- 全部失败
```

> **已对齐**：前端和后端状态枚举已统一为 6 种状态（pending/running/completed/failed/paused/cancelled），通过 i18n Tag 映射展示。

---

### 5.8 API 接口设计

#### 5.8.1 获取脚本列表（实际用于脚本任务数据查询）

```
GET /api/v1/mml/scripts
```

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码 |
| page_size | int | 每页数量 |
| device_type | string | 设备类型过滤 |
| creator | string | 创建者过滤 |
| search | string | 关键词搜索 |

**响应体**：

```json
{
  "items": [
    {
      "id": "uuid",
      "script_name": "批量激活小区脚本",
      "description": "",
      "content": "LST CELL;",
      "device_type": "eNB",
      "creator": "admin",
      "tags": [],
      "created_at": "2026-03-01T06:00:00Z",
      "updated_at": "2026-03-01T06:00:00Z"
    }
  ],
  "total": 25,
  "page": 1,
  "page_size": 20,
  "total_pages": 2
}
```

#### 5.8.2 获取任务列表

```
GET /api/v1/mml/tasks
```

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码 |
| page_size | int | 每页数量 |
| status | string | 状态过滤 |

**响应体**（同脚本列表格式，`items` 为 `MMLTask[]`）

#### 5.8.3 创建脚本

```
POST /api/v1/mml/scripts
```

**请求体**（`CreateScriptRequest`）：

```json
{
  "script_name": "批量激活小区脚本",
  "description": "激活全部小区",
  "content": "LST CELL;\nMOD CELL: CellId=1, Status=active;",
  "device_type": "eNB",
  "tags": ["cell", "activate"]
}
```

#### 5.8.4 更新脚本

```
PUT /api/v1/mml/scripts/:id
```

请求体同创建接口（`script_name` + `content` 必填）。

#### 5.8.5 删除脚本

```
DELETE /api/v1/mml/scripts/:id
```

返回 `204 No Content`。

#### 5.8.6 任务控制接口（已实现）

```
POST /api/v1/mml/tasks/:id/start      启动挂起/暂停的任务（pending/paused → running）
POST /api/v1/mml/tasks/:id/pause      暂停运行中的任务（running → paused）
POST /api/v1/mml/tasks/:id/cancel     取消任务（pending/running/paused → cancelled）
DELETE /api/v1/mml/tasks/:id          删除任务（仅非 running 态可删）
GET /api/v1/mml/tasks/:id/results     获取任务结果明细（支持分页）
```

**状态转换规则**（`service.go`）：

| 操作 | 允许的当前状态 | 目标状态 |
|------|--------------|---------|
| StartTask | pending, paused | running |
| PauseTask | running | paused |
| CancelTask | pending, running, paused | cancelled |
| DeleteTask | pending, paused, completed, failed, cancelled | 已删除 |

---

### 5.9 数据模型

#### 5.9.1 mml_scripts 表

```sql
CREATE TABLE mml_scripts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    script_name VARCHAR(200) NOT NULL,    -- 脚本名称
    description TEXT,                     -- 脚本描述
    content     TEXT NOT NULL,            -- 脚本内容（每行一条 MML 命令）
    device_type VARCHAR(50),              -- 适用设备类型（eNB/gNB 等）
    creator     VARCHAR(100),             -- 创建者用户名（从 auth context 取）
    tags        JSONB DEFAULT '[]',       -- 标签数组
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trigger_mml_scripts_updated_at
    BEFORE UPDATE ON mml_scripts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_mml_scripts_tags_gin ON mml_scripts USING GIN (tags);
```

#### 5.9.2 mml_tasks 表

```sql
CREATE TABLE mml_tasks (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_name             VARCHAR(200),
    script_id             UUID REFERENCES mml_scripts(id) ON DELETE SET NULL,
    device_sns            JSONB NOT NULL,                 -- 目标设备 SN 列表
    commands              JSONB NOT NULL DEFAULT '[]',    -- [{command_code, parameters}]
    status                VARCHAR(20) NOT NULL DEFAULT 'pending',
    results               JSONB DEFAULT '[]',             -- 每台设备执行结果
    creator               VARCHAR(100),
    executor              VARCHAR(100),                   -- 实际执行者（ACS Worker）
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- 调度字段
    execute_type          VARCHAR(20) NOT NULL DEFAULT 'immediate',
    scheduled_at          TIMESTAMPTZ,
    period_start          TIMESTAMPTZ,
    period_end            TIMESTAMPTZ,
    period_time           VARCHAR(10),

    -- 重试策略
    offline_retry         BOOLEAN DEFAULT false,
    offline_retry_wait    INT DEFAULT 60,
    failed_retry          BOOLEAN DEFAULT false,
    failed_retry_count    INT DEFAULT 3,
    failed_retry_interval INT DEFAULT 5,

    -- 执行时间戳
    started_at            TIMESTAMPTZ,
    finished_at           TIMESTAMPTZ,

    -- 统计计数
    total_devices         INT DEFAULT 0,
    success_count         INT DEFAULT 0,
    failed_count          INT DEFAULT 0,
    result                VARCHAR(20)                     -- success/partial/failed
);

CREATE INDEX idx_mml_tasks_status  ON mml_tasks(status);
CREATE INDEX idx_mml_tasks_created ON mml_tasks(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_mml_tasks_commands_gin
    ON mml_tasks USING GIN (commands jsonb_path_ops);
CREATE INDEX IF NOT EXISTS idx_mml_tasks_results_gin
    ON mml_tasks USING GIN (results jsonb_path_ops);
CREATE INDEX IF NOT EXISTS idx_mml_tasks_device_sns_gin
    ON mml_tasks USING GIN (device_sns);
```

**后端 Go 模型**（`omcgo/internal/mml/model.go`）：

```go
type TaskStatus string

const (
    TaskPending   TaskStatus = "pending"
    TaskRunning   TaskStatus = "running"
    TaskCompleted TaskStatus = "completed"
    TaskFailed    TaskStatus = "failed"
    TaskPaused    TaskStatus = "paused"
    TaskCancelled TaskStatus = "cancelled"
)

type ExecuteType string

const (
    ExecuteImmediate ExecuteType = "immediate"
    ExecuteScheduled ExecuteType = "scheduled"
    ExecutePeriodic  ExecuteType = "periodic"
    ExecuteSuspended ExecuteType = "suspended"
)

type TaskResult string

const (
    ResultSuccess TaskResult = "success"
    ResultPartial TaskResult = "partial"
    ResultFailed  TaskResult = "failed"
)

type MMLTask struct {
    ID        uuid.UUID                `json:"id"`
    TaskName  string                   `json:"task_name"`
    ScriptID  *uuid.UUID               `json:"script_id,omitempty"`
    DeviceSNs []string                 `json:"device_sns"`
    Commands  []map[string]interface{} `json:"commands"`
    Status    TaskStatus               `json:"status"`
    Results   []map[string]interface{} `json:"results"`
    Creator   string                   `json:"creator"`
    Executor  string                   `json:"executor,omitempty"`
    CreatedAt time.Time                `json:"created_at"`
    UpdatedAt time.Time                `json:"updated_at"`

    // Scheduling
    ExecuteType ExecuteType  `json:"execute_type"`
    ScheduledAt *time.Time   `json:"scheduled_at,omitempty"`
    PeriodStart *time.Time   `json:"period_start,omitempty"`
    PeriodEnd   *time.Time   `json:"period_end,omitempty"`
    PeriodTime  string       `json:"period_time,omitempty"`

    // Retry strategy
    OfflineRetry        bool `json:"offline_retry"`
    OfflineRetryWait    int  `json:"offline_retry_wait"`
    FailedRetry         bool `json:"failed_retry"`
    FailedRetryCount    int  `json:"failed_retry_count"`
    FailedRetryInterval int  `json:"failed_retry_interval"`

    // Execution timestamps
    StartedAt  *time.Time `json:"started_at,omitempty"`
    FinishedAt *time.Time `json:"finished_at,omitempty"`

    // Statistics
    TotalDevices int        `json:"total_devices"`
    SuccessCount int        `json:"success_count"`
    FailedCount  int        `json:"failed_count"`
    Result       TaskResult `json:"result,omitempty"`
}
```

#### 5.9.3 mml_templates 表（已实现）

```sql
CREATE TABLE mml_templates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_name   VARCHAR(200) NOT NULL,
    command_code    VARCHAR(100) NOT NULL,
    operation_type  VARCHAR(20) NOT NULL,
    template_scope  VARCHAR(20) NOT NULL DEFAULT 'private',
    category_group  VARCHAR(50),
    parameters      JSONB NOT NULL DEFAULT '{}'::jsonb,
    param_paths     JSONB NOT NULL DEFAULT '[]'::jsonb,
    description     TEXT,
    product_types   JSONB NOT NULL DEFAULT '[]'::jsonb,
    creator         VARCHAR(100) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_mml_templates_scope CHECK (template_scope IN ('private', 'public')),
    CONSTRAINT chk_mml_templates_op CHECK (operation_type IN ('LST', 'MOD', 'ADD', 'RMV', 'DSP', 'ACT', 'DEA', 'RST', 'CLR', 'UPG'))
);

CREATE INDEX idx_mml_templates_command_code ON mml_templates(command_code);
CREATE INDEX idx_mml_templates_scope_creator ON mml_templates(template_scope, creator);
CREATE INDEX idx_mml_templates_product_types_gin ON mml_templates USING GIN (product_types);
CREATE INDEX idx_mml_templates_parameters_gin ON mml_templates USING GIN (parameters);
```

#### 5.9.4 mml_audit_log 表（已建表，写入逻辑待实现）

```sql
CREATE TABLE mml_audit_log (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id        UUID REFERENCES mml_tasks(id),
    command_code   VARCHAR(100) NOT NULL,
    operation_type VARCHAR(20) NOT NULL,
    device_sn      VARCHAR(100) NOT NULL,
    parameters     JSONB DEFAULT '{}'::jsonb,
    param_paths    JSONB DEFAULT '[]'::jsonb,
    result_status  VARCHAR(20),
    result_message TEXT,
    creator        VARCHAR(100) NOT NULL,
    executed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    duration_ms    INT
);

CREATE INDEX idx_mml_audit_task_id ON mml_audit_log(task_id);
CREATE INDEX idx_mml_audit_device_sn ON mml_audit_log(device_sn);
CREATE INDEX idx_mml_audit_creator ON mml_audit_log(creator);
CREATE INDEX idx_mml_audit_executed_at ON mml_audit_log(executed_at DESC);
```

#### 5.9.5 参数库表结构（已建表 + 后端模块已实现）

> 参数库用于管理 TR-069 设备参数的版本化定义，支持按产品型号和软件版本匹配参数集。迁移文件：`migrations/000022_mml_param_library.sql`，种子数据：`migrations/seed/000005_seed_mml_param_library.sql`（24 个版本、1919 个分组、7226 条参数）。
>
> **后端已实现**：`param_model.go`、`param_service.go`、`param_pg_repository.go`、`param_handler.go`，4 个 API 端点已注册（`GET /mml/param-versions`、`GET /mml/param-versions/:version/groups`、`GET /mml/param-versions/:version/groups/:groupId/params`、`GET /mml/param-versions/:version/params`）。

```sql
-- 参数版本
CREATE TABLE mml_param_versions (
    version_code      VARCHAR(100) PRIMARY KEY,
    version_name      VARCHAR(200),           -- 版本名称（如 Qcells B1.0）
    description       TEXT DEFAULT '',
    release_date      TIMESTAMPTZ,            -- 发布日期
    product_models    TEXT[] DEFAULT '{}',
    software_versions TEXT[] DEFAULT '{}',
    is_active         BOOLEAN DEFAULT true,
    is_deprecated     BOOLEAN DEFAULT false,
    group_count       INT DEFAULT 0,
    param_count       INT DEFAULT 0,
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    updated_at        TIMESTAMPTZ DEFAULT NOW()
);

-- 参数分组（层级树结构）
CREATE TABLE mml_param_groups (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_code              VARCHAR(100) NOT NULL,
    group_name_zh           VARCHAR(200) NOT NULL,
    group_name_en           VARCHAR(200),
    parent_id               UUID REFERENCES mml_param_groups(id),
    level                   INT DEFAULT 0,
    is_listable             BOOLEAN DEFAULT false,
    is_modifiable           BOOLEAN DEFAULT false,
    is_addable              BOOLEAN DEFAULT false,
    is_removable            BOOLEAN DEFAULT false,
    add_object_path         TEXT,
    delete_object_path      TEXT,
    param_version           VARCHAR(100) REFERENCES mml_param_versions(version_code),
    platform_support        TEXT[] DEFAULT '{}',
    mobile_support          BOOLEAN DEFAULT false,
    broadband_support       BOOLEAN DEFAULT false,
    cell_number             INT DEFAULT 0,
    cell_index_location     INT DEFAULT 0,
    require_second_confirm  BOOLEAN DEFAULT false,
    confirm_message_zh      TEXT,
    confirm_message_en      TEXT,
    display_order           INT DEFAULT 0,
    is_active               BOOLEAN DEFAULT true,
    created_at              TIMESTAMPTZ DEFAULT NOW()
);

-- 参数定义
CREATE TABLE mml_params (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    param_code             VARCHAR(200) NOT NULL,
    param_name_zh          VARCHAR(200) NOT NULL,
    param_name_en          VARCHAR(200),
    tr069_path             TEXT NOT NULL,
    value_type             VARCHAR(20) CHECK (value_type IN ('INTEGER','UNSIGNED_INT','STRING','BOOLEAN','ENUM','IP_ADDRESS','LIST','HEX_BINARY')),
    value_constraint       JSONB DEFAULT '{}',
    default_value          TEXT,
    js_regex               TEXT,               -- 前端校验正则
    is_writable            BOOLEAN DEFAULT true,
    is_listable            BOOLEAN DEFAULT false,
    is_modifiable          BOOLEAN DEFAULT false,
    is_addable             BOOLEAN DEFAULT false,
    is_removable           BOOLEAN DEFAULT false,
    is_leaf                BOOLEAN DEFAULT true,
    is_dynamic             BOOLEAN DEFAULT false,
    display_order          INT DEFAULT 0,
    param_version          VARCHAR(100) REFERENCES mml_param_versions(version_code),
    software_version       VARCHAR(100),
    platform_support       TEXT[] DEFAULT '{}',
    mobile_support         BOOLEAN DEFAULT false,
    broadband_support      BOOLEAN DEFAULT false,
    memo                   TEXT,
    explanation_zh         TEXT,
    explanation_en         TEXT,
    title_zh               TEXT,
    title_en               TEXT,
    require_second_confirm BOOLEAN DEFAULT false,
    confirm_message_zh     TEXT,
    confirm_message_en     TEXT,
    is_active              BOOLEAN DEFAULT true,
    created_at             TIMESTAMPTZ DEFAULT NOW()
);

-- 分组-参数关联
CREATE TABLE mml_group_param_rel (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id    UUID REFERENCES mml_param_groups(id),
    param_id    UUID REFERENCES mml_params(id),
    sort_order  INT DEFAULT 0
);
```

---

## 6. 命令执行引擎

### 6.1 完整执行流程图

```
前端用户操作（MML 控制台）
         │
         │ 点击"执行"按钮
         ▼
handleExecute() [CommandInput.tsx]
         │
         ├── 检查: selectedDevices.length === 0 → 输出错误行
         ├── 检查: !selectedCommand → 输出错误行
         └── 检查: 危险命令 → 弹出确认 Modal
                   │
                   ▼
executeCommand(devices, command, params) [useCommandExecution.ts]
         │
         │ 逐台设备循环
         ▼
executeMutation.mutateAsync({commandCode, deviceSns:[sn], params})
         │
         │ POST /api/v1/mml/execute
         ▼
Execute Handler [handler.go]
         │
         │ ShouldBindJSON → ExecuteHTTPRequest
         │ 从 context 取 creator
         ▼
service.ExecuteCommand(ctx, execReq)
         │
         │ 创建 MMLTask（status=pending）
         │ 写入 mml_tasks 表
         ▼
返回 task 对象（status=pending，results=[]）
         │
         ▼
前端根据 task.results 展示（当前为空）
         │
         │（异步，需轮询）
         ▼
ACS Worker 消费任务（TODO：待实现）
         │
         │ 读取 mml_tasks 中 status=pending 的任务
         │ 根据 command_code 找到对应 rpc_method
         ▼
TR-069 SOAP 请求发送到设备
         │
         │ 根据设备在线状态：
         ├── 在线: ConnectionRequest → 等待设备 Inform → 发送 RPC
         └── 离线: 根据 offline_retry 策略等待或标记失败
         │
         ▼
解析 SOAP 响应 → 更新 mml_tasks.results
更新 mml_tasks.status = completed / failed
         │
         ▼
前端 React Query 轮询获取最终结果
GET /api/v1/mml/tasks/:id
```

### 6.2 RPC 方法映射

| 操作类型 | rpc_method | TR-069 SOAP Action | 说明 |
|---------|-----------|-------------------|------|
| LST/DSP | GetParameterValues | `cwmp:GetParameterValues` | 查询参数值 |
| MOD/ACT/DEA/CLR | SetParameterValues | `cwmp:SetParameterValues` | 修改参数值 |
| ADD | AddObject | `cwmp:AddObject` | 添加对象实例 |
| RMV | DeleteObject | `cwmp:DeleteObject` | 删除对象实例 |
| RST | Reboot | `cwmp:Reboot` | 设备重启 |
| UPG | Download | `cwmp:Download` | 软件升级下载 |

**当前支持状态**（参照 handler.go + service.go）：

| RPC 方法 | 支持状态 |
|----------|---------|
| GetParameterValues | ✅ 已有框架 |
| SetParameterValues | ✅ 已有框架 |
| Reboot | ✅ 已有框架 |
| Download | ✅ 已有框架 |
| AddObject | ❌ 待实现 |
| DeleteObject | ❌ 待实现 |
| GetParameterNames | ⚠️ 部分支持 |

### 6.3 参数验证规则

| 参数类型 | 前端验证 |
|---------|---------|
| string | required 时非空；`pattern` 存在时正则匹配 |
| number | required 时非空；`minValue`/`maxValue` 范围校验 |
| boolean | 无特殊校验 |
| enum | 值必须在 `options` 中 |
| range | 等同 number，强制 min/max |
| ipAddress | IPv4/IPv6 格式正则 |
| list | 逗号分隔，每项独立校验 |

### 6.4 结果解析与展示

**results JSONB 中每条记录结构**：

```json
{
  "device_sn": "ENB00001",
  "success": true,
  "raw_output": "BSCID=1 BSCNAME=SITE-001 ...",
  "parsed_data": {"BSCID": "1", "BSCNAME": "SITE-001"},
  "execution_time": 1250,
  "timestamp": "2026-04-14T10:00:01Z"
}
```

**终端展示规则**（`useCommandExecution.ts`）：

```
[HH:mm:ss] 执行命令: {commandCode}          → type: info（#91D5FF）
[HH:mm:ss] 目标设备: {sn1}, {sn2}           → type: info
[HH:mm:ss] ──────────────────────────────   → type: info
[HH:mm:ss] --- 设备: {sn} ---               → type: info
[HH:mm:ss]   设备名称: {name}               → type: stdout（#52C41A）
[HH:mm:ss]   设备类型: {type}               → type: stdout
[HH:mm:ss]   产品型号: {productType}        → type: stdout
[HH:mm:ss]   运行状态: 在线/告警/离线        → type: stdout
[HH:mm:ss]   执行结果: 成功/失败             → type: success/stderr
[HH:mm:ss] ✓ 命令执行完成，共处理 N 台设备  → type: success（#B7EB8F）
```

### 6.5 错误处理

| 错误类型 | 前端处理 |
|---------|---------|
| 未选设备 | 输出 `stderr` 行"错误：请先选择设备"，不发起请求 |
| 未选命令 | 输出 `stderr` 行"错误：请先选择命令"，不发起请求 |
| 危险命令 | 弹出确认 Modal，用户取消则不执行 |
| API 异常 | catch → 输出 `stderr` 行（`error.message`） |
| 设备离线 | 根据 offline_retry 策略（后端），前端展示结果状态 |
| RPC 超时 | 默认 60s，超时后标记 success=false |

### 6.6 批量执行策略

| 维度 | 当前实现 | 建议目标 |
|------|---------|---------|
| 并发数 | 逐台串行（for 循环）| 建议并发 10 台设备 |
| 失败继续 | ✅ 单台失败不影响其他设备 | - |
| 进度更新 | 每台完成后更新输出行 | 实时更新 task.results |
| 超大批量 | 无限制 | 建议超 100 台时分批（每批 50 台）|

---

## 7. 操作类型详解

### 7.1 操作类型总表

| 操作类型 | 中文含义 | TR-069 RPC | 典型命令模式 | 是否改变设备配置 | 是否要求危险确认 |
|---------|----------|------------|--------------|------------------|------------------|
| LST | 查询 | `GetParameterValues` | `LST CELL`、`LST BTSSTATE`、`LST PM` | 否 | 否 |
| MOD | 修改 | `SetParameterValues` | `MOD CELL` | 是 | 视命令而定 |
| ADD | 增加 | `SetParameterValues`/`AddObject` | `ADD NCELL` | 是 | 是 |
| RMV | 删除 | `SetParameterValues`/`DeleteObject` | `DEL NCELL` | 是 | 是 |
| DSP | 显示详情 | `GetParameterValues` | `DSP BOARDSTATUS`、`DSP VERSION` | 否 | 否 |
| ACT | 激活 | `SetParameterValues` | `ACT CELL` | 是 | 是 |
| DEA | 去激活 | `SetParameterValues` | `DEA CELL` | 是 | 是 |
| RST | 复位/重启 | `Reboot`/`SetParameterValues` | `RST BTS`、`RST CELL` | 是 | 是 |
| CLR | 清除 | `SetParameterValues` | `CLR ALM` | 是 | 否 |
| UPG | 升级 | `Download` | `UPG PKG` | 是 | 是 |

### 7.2 LST（查询）

#### 7.2.1 含义与适用场景

LST 用于读取设备当前配置、运行状态、告警、性能等只读信息，不修改设备配置。适合以下场景：

- 日常巡检：查询版本、板卡、状态、性能。
- 故障定位：查询告警、参数、链路状态。
- 批量核查：对大量设备统一读取相同参数。

#### 7.2.2 RPC 映射

```text
MML LST → TR-069 GetParameterValues → 返回参数树当前值
```

#### 7.2.3 参数规则

| 规则 | 说明 |
|------|------|
| 必填参数 | 通常较少，缺省表示查询全部对象 |
| 可选参数 | 用于缩小范围，如 `CELLID`、时间范围 |
| 参数路径 | 支持在“参数路径指定”Tab 中录入 TR-069 完整路径 |
| 返回结果 | 可能为表格型、键值型、纯文本型 |

#### 7.2.4 UI 行为

- 默认操作类型为 `LST`。
- 动态参数表单中通常仅显示查询条件。
- 执行后优先在 [CommandInput](file:///Users/cb/code/baicells/goomc/omcmb/webcode/src/pages/mml/Console/components/CommandInput.tsx) 和 [TerminalPanel](file:///Users/cb/code/baicells/goomc/omcmb/webcode/src/pages/mml/Console/components/TerminalPanel.tsx) 展示结果摘要。
- 结果明细应通过任务详情接口轮询补全。

### 7.3 MOD（修改）

#### 7.3.1 含义与适用场景

MOD 用于修改已存在参数值，如开关状态、频点、功率、阈值等。属于高风险操作。

典型场景：
- 修改 eNB/gNB 配置。
- 调整小区参数。
- 清除告警标志、改变设备状态。

#### 7.3.2 RPC 映射

```text
MML MOD → TR-069 SetParameterValues → 写入参数值
```

#### 7.3.3 参数规则

| 规则 | 说明 |
|------|------|
| 必填参数必须完整 | 必须明确被修改对象和目标值 |
| 范围校验 | `number` / `range` 类型严格校验 `minValue`、`maxValue` |
| enum 校验 | 值必须在 `options` 集合中 |
| 批量执行 | 建议按设备逐台记录结果，避免整体回滚语义不清 |

#### 7.3.4 安全要求

- 命令码命中危险模式时必须二次确认。
- 审计日志需记录操作者、旧值（如可取）、新值、设备 SN、时间戳。
- 默认仅管理员或具备 `mml.console.execute.write` 权限的角色可执行。

### 7.4 ADD（增加）

#### 7.4.1 含义与适用场景

ADD 用于在 TR-069 对象树中增加实例，例如新增邻区、端口、规则项、路由项等。

#### 7.4.2 RPC 映射

```text
MML ADD → TR-069 AddObject → 返回 instance number / status
```

#### 7.4.3 参数规则

| 规则 | 说明 |
|------|------|
| 必须提供父路径 | 例如 `Device.Services.FAPService.1.CellConfig.` |
| 新建后补写参数 | 如返回对象实例号，系统应串接后续 `SetParameterValues` |
| 结果处理 | 响应需保存新实例号，便于后续显示和回滚 |

#### 7.4.4 实施注意

当前后端模型和执行引擎仅完成 `ExecuteCommand` 框架，[service.go](file:///Users/cb/code/baicells/goomc/omcgo/internal/mml/service.go) 与 [handler.go](file:///Users/cb/code/baicells/goomc/omcgo/internal/mml/handler.go) 尚未体现 `AddObject` 的完整执行逻辑，因此本类型属于待实现项。

### 7.5 RMV（删除）

#### 7.5.1 含义与适用场景

RMV 用于删除对象实例，如删除邻区、规则项、临时策略等。

#### 7.5.2 RPC 映射

```text
MML RMV → TR-069 DeleteObject → 删除目标实例
```

#### 7.5.3 参数规则

| 规则 | 说明 |
|------|------|
| 必须指定实例路径 | 不允许对模糊路径执行删除 |
| 删除前确认 | 必须弹出危险操作确认框 |
| 删除后刷新 | 建议自动触发一次 LST 校验对象是否已删除 |

### 7.6 操作类型与参数路径联动规则

| 场景 | UI 行为 | 后端要求 |
|------|---------|---------|
| 切换操作类型 | 重置不兼容参数，保留通用路径项 | 重新解析 `rpc_method` 和参数模板 |
| 切到 ADD/RMV | 突出显示参数路径区，底部显示告警提示 | 校验路径必须为对象级路径 |
| 切到 MOD | 高亮被改动参数项 | 记录审计日志 |
| 切到 LST | 隐藏危险提示 | 支持更高并发读操作 |

---

## 8. 自定义命令模板

### 8.1 模板目标与范围

自定义命令模板用于沉淀常用参数组合，减少重复录入，提高批量维护效率。模板与脚本不同：

- **命令模板**：面向单条命令 + 参数快照。
- **脚本模板**：面向多条 MML 命令的脚本内容。

模板需要支持两种可见性：

| 类型 | 说明 | 可见范围 |
|------|------|----------|
| PrivateTemplate | 私有模板 | 仅创建者本人 |
| PublicTemplate | 公共模板 | 有读取权限的全部用户 |

### 8.2 业务规则

| 规则 | 说明 |
|------|------|
| 归属命令 | 每个模板必须绑定一个 `command_code` |
| 操作类型 | 记录 `operation_type`，避免 LST/MOD 模板混用 |
| 参数快照 | 保存当时填写的 `parameters` JSON |
| 参数路径 | 如使用参数路径指定模式，应同步保存 `param_paths` |
| 可见性 | private/public 二选一 |
| 产品类型限制 | 可选绑定 `product_types`，仅在匹配设备型号时显示 |
| 删除规则 | PublicTemplate 仅创建者或管理员可删除 |

### 8.3 推荐数据模型

#### 8.3.1 mml_templates 表 DDL

```sql
CREATE TABLE mml_templates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_name   VARCHAR(200) NOT NULL,                 -- 模板名称
    command_code    VARCHAR(100) NOT NULL,                 -- 绑定命令编码
    operation_type  VARCHAR(20) NOT NULL,                  -- LST/MOD/ADD/RMV
    template_scope  VARCHAR(20) NOT NULL DEFAULT 'private',-- private/public
    parameters      JSONB NOT NULL DEFAULT '{}'::jsonb,    -- 参数快照
    param_paths     JSONB NOT NULL DEFAULT '[]'::jsonb,    -- 参数路径列表
    description     TEXT,                                  -- 模板说明
    product_types   JSONB NOT NULL DEFAULT '[]'::jsonb,    -- 适用产品类型
    creator         VARCHAR(100) NOT NULL,                 -- 创建者
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_mml_templates_scope CHECK (template_scope IN ('private', 'public')),
    CONSTRAINT chk_mml_templates_op CHECK (operation_type IN ('LST', 'MOD', 'ADD', 'RMV'))
);

CREATE INDEX idx_mml_templates_command_code ON mml_templates(command_code);
CREATE INDEX idx_mml_templates_scope_creator ON mml_templates(template_scope, creator);
CREATE INDEX idx_mml_templates_product_types_gin ON mml_templates USING GIN (product_types);
CREATE INDEX idx_mml_templates_parameters_gin ON mml_templates USING GIN (parameters);
```

#### 8.3.2 前端类型建议

```typescript
interface MMLTemplate {
  id: string;
  templateName: string;
  commandCode: string;
  operationType: 'LST' | 'MOD' | 'ADD' | 'RMV';
  templateScope: 'private' | 'public';
  parameters: Record<string, string | number | boolean>;
  paramPaths: string[];
  description: string;
  productTypes: string[];
  creator: string;
  createdAt: string;
  updatedAt: string;
}
```

### 8.4 API 设计（已实现）

> 模板 CRUD 已在 `omcgo/internal/mml/handler.go` 和 `service.go` 中完整实现，包含 6 个端点。

#### 8.4.1 查询模板列表

```text
GET /api/v1/mml/templates
```

**Query 参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| command_code | string | 按命令过滤 |
| operation_type | string | 按操作类型过滤 |
| template_scope | string | private/public |
| page | int | 页码 |
| page_size | int | 每页数量 |

**响应体示例**：

```json
{
  "items": [
    {
      "id": "tmpl-uuid",
      "template_name": "查询全部小区",
      "command_code": "LST CELL",
      "operation_type": "LST",
      "template_scope": "private",
      "parameters": {"CELLID": 0},
      "param_paths": ["Device.Services.FAPService.1.CellConfig.1"],
      "description": "用于巡检小区配置",
      "product_types": ["eNB", "gNB"],
      "creator": "admin",
      "created_at": "2026-04-14T10:00:00Z",
      "updated_at": "2026-04-14T10:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20,
  "total_pages": 1
}
```

#### 8.4.2 创建模板

```text
POST /api/v1/mml/templates
```

```json
{
  "template_name": "eNB功率调整",
  "command_code": "MOD eNB_CONFIG",
  "operation_type": "MOD",
  "template_scope": "private",
  "parameters": {
    "FREQ": 1850,
    "PCI": 128,
    "PWR": 43
  },
  "param_paths": ["Device.Services.FAPService.1.CellConfig.1"],
  "description": "默认功率模板",
  "product_types": ["eNB"]
}
```

#### 8.4.3 更新模板

```text
PUT /api/v1/mml/templates/:id
```

#### 8.4.4 删除模板

```text
DELETE /api/v1/mml/templates/:id
```

#### 8.4.5 复制公共模板为私有模板

```text
POST /api/v1/mml/templates/:id/clone
```

### 8.5 前端交互实现

| 位置 | 行为 | 状态 |
|------|------|------|
| CommandTree | 命令树底部显示"自定义模板"分组（public/private 分离） | ✅ 已实现 |
| AddTemplateModal | 弹窗创建/编辑模板，支持 private/public 作用域切换 | ✅ 已实现 |
| CommandInput | 控制面板 Tab 渲染参数表单时，从模板加载参数 | ✅ 已实现 |
| ParamPathPanel | 参数路径 Tab 支持操作类型切换和路径增删 | ✅ 已实现 |
| ParamFormRenderer | 根据操作类型动态渲染不同参数表单样式 | ✅ 已实现 |

| 位置 | 行为 |
|------|------|
| [CommandInput](file:///Users/cb/code/baicells/goomc/omcmb/webcode/src/pages/mml/Console/components/CommandInput.tsx) | 点击“保存脚本/模板”时区分命令模板和脚本模板 |
| [CommandTree](file:///Users/cb/code/baicells/goomc/omcmb/webcode/src/pages/mml/Console/components/CommandTree.tsx) | 在分类树末尾增加“自定义模板”分组 |
| 控制面板 Tab | 增加“从模板加载”下拉 |
| 参数路径 Tab | 加载模板时同步恢复 `operationType` 与 `paramPaths` |

### 8.6 权限规则

| 动作 | 普通用户 | 运维管理员 | 系统管理员 |
|------|----------|------------|------------|
| 查看私有模板 | 仅本人 | 仅本人 | 全部 |
| 查看公共模板 | ✅ | ✅ | ✅ |
| 创建私有模板 | ✅ | ✅ | ✅ |
| 创建公共模板 | ❌ | ✅ | ✅ |
| 删除他人公共模板 | ❌ | ❌ | ✅ |

---

## 9. 内置命令种子数据

### 9.1 当前命令分类树（已落库）

以下分类树基于种子数据 `migrations/seed/000018_mml_enhance.sql`，共 **25 条命令** 跨 **7 个分类**，分类使用字典编码（`mml_command_category`）。

```text
小区管理 (1) — 5 条
├── LST CELL         查询小区信息        LST
├── ACT CELL         激活小区            ACT
├── DEA CELL         去激活小区          DEA
├── MOD CELL         修改小区参数        MOD
└── RST CELL         重置小区            RST

邻区管理 (2) — 3 条
├── LST NCELL        查询邻区            LST
├── ADD NCELL        添加邻区            ADD
└── DEL NCELL        删除邻区            RMV

基站管理 (3) — 5 条
├── LST BTSSTATE     查询基站状态        LST
├── DSP BOARDSTATUS  查询单板状态        DSP
├── RST BTS          复位基站            RST
├── DSP SYSRESOURCE  查询系统资源        DSP
└── DSP CLOCKSTATUS  查询时钟状态        DSP

告警查询 (4) — 3 条
├── LST ALMAF        查询活动告警        LST
├── LST ALMHIS       查询历史告警        LST
└── CLR ALM          清除告警            CLR

性能采集 (5) — 3 条
├── DSP PERF         查询性能计数器      DSP
├── LST PM           查询性能统计        LST
└── DSP RRUINFO      查询RRU信息         DSP

传输管理 (6) — 3 条
├── DSP LINKSTATUS   查询传输链路        DSP
├── DSP SCTP         查询SCTP链路        DSP
└── LST IPADDR       查询IP地址          LST

版本管理 (7) — 3 条
├── DSP VERSION      查询设备版本        DSP
├── LST PKG          查询软件包          LST
└── UPG PKG          升级软件包          UPG
```

### 9.2 种子数据设计原则

| 原则 | 说明 |
|------|------|
| command_code 唯一 | 作为命令实体主标识 |
| category 稳定 | 前端树按该字段聚合 |
| rpc_method 明确 | 执行引擎据此选择 TR-069 RPC |
| param_template 为 JSONB | 直接驱动动态表单渲染 |
| product_types 精确 | 前端根据设备型号过滤可用命令 |

### 9.3 内置命令种子 SQL

> 完整种子数据见 `omcgo/migrations/seed/000018_mml_enhance.sql`，共 25 条命令，使用 `ON CONFLICT (command_code) DO UPDATE` 实现幂等写入。下方展示关键结构示例：

```sql
-- 每条命令包含完整元数据
INSERT INTO mml_commands (
    id, command_name, command_code, category, description,
    rpc_method, operation_type, param_template, param_paths,
    supported_operations, help_doc, notes, product_types, created_at
) VALUES
(
    '00000000-0000-0000-0001-000000000001',
    '查询小区信息',
    'LST CELL',
    '1',                              -- 字典编码：小区管理
    '列出当前基站所有小区的配置信息',
    'GetParameterValues',
    'LST',
    '{"CELLID":{"type":"number","required":false,"description":"小区ID","min_value":0,"max_value":255}}'::jsonb,
    '["Device.Services.FAPService.{i}.CellConfig.{i}"]'::jsonb,
    '["LST"]'::jsonb,
    '查询小区基础配置、射频及运行状态参数。',
    '不传 CELLID 时返回所有小区。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
)
-- ... 其余 24 条见完整迁移文件
ON CONFLICT (command_code) DO UPDATE SET
    command_name = EXCLUDED.command_name,
    category = EXCLUDED.category,
    description = EXCLUDED.description,
    rpc_method = EXCLUDED.rpc_method,
    operation_type = EXCLUDED.operation_type,
    param_template = EXCLUDED.param_template,
    param_paths = EXCLUDED.param_paths,
    supported_operations = EXCLUDED.supported_operations,
    help_doc = EXCLUDED.help_doc,
    notes = EXCLUDED.notes,
    product_types = EXCLUDED.product_types;
```

**字典种子**（同步写入 `sys_dictionaries` / `sys_dictionary_details`）：

```sql
-- 产品类型字典
INSERT INTO sys_dictionaries (name, type, status, description) VALUES
('产品类型', 'product_type', TRUE, '设备产品类型');

INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id) VALUES
('SmallCell-LTE', '1', 1, ...),
('gNB-100', '2', 2, ...),
('gNB-200', '3', 3, ...),
('FAP-LTE-100', '4', 4, ...),
('FAP-LTE-200', '5', 5, ...),
('FAP-LTE-300', '6', 6, ...);

-- MML 命令分类字典
INSERT INTO sys_dictionaries (name, type, status, description) VALUES
('MML命令类型', 'mml_command_category', TRUE, 'MML命令分类');

INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id) VALUES
('小区管理', '1', 1, ...),
('邻区管理', '2', 2, ...),
('基站管理', '3', 3, ...),
('告警查询', '4', 4, ...),
('性能采集', '5', 5, ...),
('传输管理', '6', 6, ...),
('版本管理', '7', 7, ...);
```

### 9.4 初始化与刷新策略

| 场景 | 建议 |
|------|------|
| 首次部署 | 迁移脚本初始化 `mml_commands` |
| 新增内置命令 | 通过版本化 SQL patch 增量插入 |
| 命令废弃 | 增加 `enabled` 字段或归档表，避免直接删除 |
| 前端缓存 | React Query `staleTime=10min`，后台变更后主动失效 |

---

## 10. 前端技术方案

### 10.1 组件架构图

```text
页面层
├── /mml/console
│   ├── DeviceTree            — 设备选择（搜索、产品类型字典筛选、分页、批量输入）
│   ├── CommandTree           — 命令树（7 分类 + 自定义模板 public/private）
│   ├── TerminalPanel         — 终端输出（深色主题、复制/清空/下载）
│   ├── CommandInput          — 执行面板（双 Tab、危险确认）
│   │   ├── ParamFormRenderer — 参数动态表单（按操作类型渲染）
│   │   └── ParamPathPanel    — TR-069 参数路径编辑
│   ├── AddTemplateModal      — 模板创建/编辑弹窗
│   └── BatchSnModal          — 批量 SN 输入弹窗
└── /mml/script
    ├── FilterBar             — 搜索筛选栏
    ├── DataTable             — 任务列表表格
    ├── Drawer(新建任务)       — 新建 MML 脚本任务
    ├── Modal(任务详情)        — 任务详情查看
    └── Modal(执行结果)        — 执行结果明细

数据层
├── useMMLCommands / useAllMMLCommands
├── useMMLScripts / useMMLScriptById
├── useMMLTasks / useMMLTaskById / useMMLTaskPolling / useMMLTaskResults
├── useMMLTemplates / useDangerousCheck
├── useExecuteMMLCommand / useCreateMMLTask / useExecuteMMLScript
├── useStartMMLTask / usePauseMMLTask / useCancelMMLTask / useDeleteMMLTask
├── useCreateMMLTemplate / useUpdateMMLTemplate / useDeleteMMLTemplate / useCloneMMLTemplate
└── mmlApi / mmlService(apiSwitch)
```

### 10.2 状态管理

| 层次 | 实现方式 | 说明 |
|------|----------|------|
| 页面局部 UI 状态 | `useState` | Tab、弹窗开关、输入值、分页 |
| 控制台业务状态 | 自定义 hooks | `useDeviceSelection`、`useCommandSelection`、`useCommandExecution` |
| 服务端缓存 | TanStack Query | 命令、脚本、任务列表查询 |
| 主题与国际化 | `useThemeToken`、`useT` | 统一颜色 Token 与文案翻译 |

### 10.3 API 接入层

当前接入通过 [useMML](file:///Users/cb/code/baicells/goomc/omcmb/webcode/src/hooks/api/useMML.ts) 中的 `createApiSwitch(mmlService, mmlApi)` 实现 Mock / Real API 可切换。

| 层级 | 文件 | 职责 |
|------|------|------|
| Hook 层 | [useMML.ts](file:///Users/cb/code/baicells/goomc/omcmb/webcode/src/hooks/api/useMML.ts) | 提供查询与 mutation Hook |
| API 层 | [mmlApi.ts](file:///Users/cb/code/baicells/goomc/omcmb/webcode/src/services/api/mmlApi.ts) | HTTP 调用、snake_case → camelCase 映射 |
| Mock 层 | [mmlService.ts](file:///Users/cb/code/baicells/goomc/omcmb/webcode/src/mock/services/mmlService.ts) | 本地假数据与延时模拟 |

### 10.4 i18n 方案

- 控制台标题、搜索框、命令树标题等已使用 `useT()` 获取国际化文案。
- 命令分类在 [zh-CN 词条](file:///Users/cb/code/baicells/goomc/omcmb/webcode/src/i18n/zh-CN/index.ts) 与 [en-US 词条](file:///Users/cb/code/baicells/goomc/omcmb/webcode/src/i18n/en-US/index.ts) 中已有基础定义。
- 建议将脚本任务页当前硬编码中文文案也迁移至 i18n，以支持国际版本。

### 10.5 Mock 数据策略

| 模块 | 当前策略 | 后续要求 |
|------|---------|---------|
| 控制台设备列表 | 本地常量 `DEVICE_LIST` | 替换为真实设备查询接口 |
| 命令树 | 本地常量 `MOCK_COMMANDS` | 替换为 `GET /mml/commands` |
| 执行结果 | Mock 服务直接返回 `MMLResult[]` | 真实接口返回任务对象 + 轮询 |
| 脚本任务页 | 本地表格数据 + 前端过滤 | 切换为服务端分页与过滤 |

### 10.6 推荐前端改造步骤

1. 先完成命令树与脚本/任务列表的真实接口接入。
2. 再补齐控制台执行结果轮询和任务详情展示。
3. 最后实现模板系统、脚本上传解析、结果导出。

### 10.7 页面级错误与空状态规范

| 场景 | 组件表现 |
|------|---------|
| 命令列表加载失败 | `Result`/`Empty` + 重试按钮 |
| 脚本列表为空 | 表格 `Empty`，提示“暂无脚本” |
| 无权限 | 页面级 `Result 403` |
| 接口超时 | `message.error` + 查询缓存不更新 |

---

## 11. 权限与安全

### 11.1 权限矩阵

| 功能 | 权限编码建议 | 说明 |
|------|--------------|------|
| 查看 MML 控制台 | `mml.console.view` | 允许进入 `/mml/console` |
| 执行只读命令 | `mml.console.execute.read` | LST 类命令 |
| 执行写操作命令 | `mml.console.execute.write` | MOD/ADD/RMV/RST |
| 查看脚本任务 | `mml.script.view` | 查看任务列表和详情 |
| 创建脚本/任务 | `mml.script.create` | 新建脚本、创建任务 |
| 控制任务 | `mml.script.control` | 启动/暂停/终止/删除 |
| 管理公共模板 | `mml.template.public.manage` | 创建/编辑/删除公共模板 |

### 11.2 前端安全控制

- 页面路由需按权限控制菜单显隐。
- 写操作按钮无权限时直接禁用，并显示 Tooltip 提示。
- 危险命令必须弹出二次确认，确认信息中展示命令名、设备数、影响说明。
- 文件上传仅允许 `.txt`，并限制大小与 MIME。

### 11.3 后端安全控制

| 项目 | 设计要求 |
|------|---------|
| 鉴权 | 从 Gin context 中读取 `username`，并结合 Casbin 判权 |
| 参数校验 | `binding:"required"` + service 层业务校验 |
| 审计日志 | 记录命令、参数、设备 SN、执行人、任务 ID |
| 限流 | 对 `/mml/execute` 做用户级频控 |
| SQL 安全 | 使用参数化查询 / Query Builder |
| 数据隔离 | 私有模板与私有脚本需按 creator 过滤 |

### 11.4 危险命令防护

当前前端已在 [CommandInput](file:///Users/cb/code/baicells/goomc/omcmb/webcode/src/pages/mml/Console/components/CommandInput.tsx) 内置危险命令正则。后端还需增加同等校验，避免绕过前端直接调用接口。

### 11.5 数据脱敏与导出控制

- 终端导出结果中如包含 IP、认证信息、密钥参数，应支持脱敏导出。
- 公共模板与公共脚本导出时不得包含用户私有标签与备注。

---

## 12. 非功能需求

### 12.1 性能要求

| 指标 | 目标值 |
|------|--------|
| 控制台页面首次可交互时间 | ≤ 3s |
| 命令树本地筛选响应 | ≤ 200ms |
| 任务列表翻页响应 | ≤ 1s |
| 单批次执行任务创建 | ≤ 500ms 返回 task id |
| 100 台设备批量执行调度启动 | ≤ 5s 完成任务入队 |

### 12.2 可用性要求

| 项目 | 要求 |
|------|------|
| 任务持久化 | 浏览器刷新后可通过任务 ID 继续查看结果 |
| 失败恢复 | ACS Worker 重启后可继续消费 `pending/running` 任务 |
| 幂等性 | 重试提交时通过业务 key 避免重复创建相同任务 |

### 12.3 可观测性要求

- 为 MML 执行链路增加 trace id。
- 记录任务创建、出队、下发、回执、完成五个阶段日志。
- 暴露 Prometheus 指标：任务创建数、成功率、平均耗时、超时数、失败原因分布。

### 12.4 兼容性要求

| 维度 | 要求 |
|------|------|
| 浏览器 | Chrome / Edge 最新两个大版本 |
| 设备类型 | eNB、gNB、GSM（按 `product_types` 过滤能力） |
| 响应格式 | 兼容纯文本结果与结构化 JSON 结果 |

### 12.5 易用性要求

- 三栏布局在 1440px 宽度下不应出现水平滚动。
- 参数过多时执行面板内部滚动，不影响终端输出区。
- 常用操作支持键盘快捷键：执行 `Ctrl/Cmd+Enter`，保存 `Ctrl/Cmd+S`。

### 12.6 可维护性要求

- 命令参数模板必须来源于数据库 JSONB，不得在前端硬编码多份。
- 任务状态枚举必须前后端统一并集中维护。
- 新增命令不应要求改动前端组件，只需新增数据和参数模板。

---

## 13. 待完善事项与实施建议

### 13.1 当前 Mock vs 真实 API 对照

| 项目 | Mock/前端现状 | 真实后端现状 | 对齐状态 |
|------|---------------|--------------|---------|
| 命令列表分页参数 | 前端 `pageSize` | 后端 `page_size` | ✅ Axios 拦截器自动转换 |
| 脚本列表分页参数 | 前端 `pageSize` | 后端 `page_size` | ✅ Axios 拦截器自动转换 |
| 任务列表分页参数 | 前端 `pageSize` | 后端 `page_size` | ✅ Axios 拦截器自动转换 |
| 命令执行请求体 | 前端 `payload.params` | 后端 `parameters` | ✅ 已对齐 |
| 命令执行返回值 | Mock 返回 `Array<{deviceSn,result}>` | 返回 `MMLTask` | ✅ 前端已适配 task 模式 |
| 获取脚本详情 | `getScriptById()` | `GET /mml/scripts/:id` | ✅ 已注册 |
| 创建任务 | `createTask()` POST `/mml/execute` | `ExecuteHTTPRequest` 支持 `script_id/commands/调度/重试` | ✅ 已扩展 |
| 执行脚本 | `executeScript()` 发送 `script_id` | 后端自动解析脚本内容为命令列表 | ✅ 已实现 |
| 任务状态枚举 | `pending/running/completed/failed/paused/cancelled` | 同左 | ✅ 已统一 |
| 模板 CRUD | 前端 `useMMLTemplates` 系列 hooks | 6 个端点已注册 | ✅ 已实现 |
| 危险命令检测 | 前端正则 + `useDangerousCheck` | `GET /mml/dangerous-check` | ✅ 已实现 |
| 任务轮询 | `useMMLTaskPolling` 2s 轮询 | `GET /mml/tasks/:id` | ✅ 已实现 |

### 13.2 后端已实现清单

1. ✅ `GET /mml/scripts/:id` 脚本详情
2. ✅ 任务启动 `start`、暂停 `pause`、取消 `cancel`、删除 `delete`、结果明细 `results` 接口
3. ✅ `ExecuteHTTPRequest` 已扩展支持 `script_id`、`commands[]`、执行策略、重试策略、参数路径
4. ✅ `mml_templates` 表 + 仓储 + 服务 + Handler（6 个端点）
5. ✅ `mml_tasks` 调度字段、统计字段、开始/结束时间已全部落库
6. ✅ 状态机已实现（6 种状态 + 合法转换校验）
7. ✅ 任务结果明细分页接口 `GET /mml/tasks/:id/results`
8. ✅ `mml_audit_log` 审计日志表已创建
9. ✅ `mml_commands` 新增 `operation_type`、`param_paths`、`supported_operations`、`help_doc`、`notes` 字段
10. ✅ 危险命令检测端点 `GET /mml/dangerous-check`
11. ✅ 命令参数路径端点 `GET /mml/commands/:id/param-paths`
12. ✅ 字典种子数据（产品类型、命令分类）
13. ✅ 参数库模块（`param_handler.go`/`param_service.go`/`param_pg_repository.go`/`param_model.go`）— 4 个 API 端点 + 种子数据（24 版本/1919 分组/7226 参数）
14. ✅ 参数库种子数据迁移（`000005_seed_mml_param_library.sql`）

### 13.3 后端待实现清单

1. ACS Worker 消费 `mml_tasks` 中 `pending/running` 任务并下发 TR-069 SOAP 请求
2. `AddObject`、`DeleteObject` 的完整执行逻辑
3. 定时/周期任务的调度器（当前 `scheduled/periodic` 创建后不会自动触发）
4. 审计日志写入逻辑（表已建，service 层需在任务执行链路中写入）
5. 批量执行并发控制（当前逐台串行，建议并发 10 台）

### 13.4 前端已完善清单

1. ✅ 命令树已从 `MOCK_COMMANDS` 切换到真实接口 `useAllMMLCommands`
2. ✅ `mmlApi.ts` 中 `pageSize`/`page_size`、`params`/`parameters` 通过 Axios 拦截器自动转换
3. ✅ `useCommandExecution` 已适配 mutation 返回 task 对象模式
4. ✅ 控制台执行后展示任务信息 + 通过 `useMMLTaskPolling` 轮询
5. ✅ 脚本任务页已对接真实服务端分页和任务控制动作
6. ✅ 模板系统已实现：`AddTemplateModal`、命令树集成 public/private 分组
7. ✅ `ParamFormRenderer` 按操作类型动态渲染参数表单
8. ✅ `ParamPathPanel` 支持 TR-069 参数路径编辑和操作类型切换
9. ✅ 产品类型和命令分类改为字典驱动（数字编码）
10. ✅ Console 页面 i18n 基本完成（~110+ keys）

### 13.5 前端待完善清单

1. 终端输出 `TerminalPanel` 中残留部分硬编码中文（”已复制到剪贴板”、”复制失败”、”等待命令输出...”等）
2. `DeviceTree` 中部分 placeholder 和按钮文本仍为中文硬编码
3. `CommandInput` Tab 标签使用英文硬编码（”Control Panel”/”ParameterPath Command”）
4. 独立 `CommandTree` 页面（`/mml/CommandTree/index.tsx`）仍使用 Mock 数据和硬编码中文
5. 脚本任务页中少量 `t('key') || '中文回退'` 模式需清理
6. 设备列表仍使用本地 Mock 数据（`DEVICE_LIST`），需替换为真实设备查询接口
7. 参数库前端集成：`mmlApi.ts` 中需新增参数库 API 调用（`getParamVersions`/`getParamGroupTree`/`getGroupParams`/`queryParams`），`CommandTree` 或独立面板中需接入参数库数据展示

### 13.6 分阶段实施建议

#### 第一阶段：接口对齐 ✅ 已完成
- ✅ 统一任务状态枚举（6 种：pending/running/completed/failed/paused/cancelled）
- ✅ 前端 API 参数名与返回值映射通过 Axios 拦截器自动处理
- ✅ 命令列表、脚本列表、任务列表真实接口已打通

#### 第二阶段：执行链路闭环 ✅ 基本完成
- ✅ `/mml/execute` 返回 task 对象，前端通过 `useMMLTaskPolling` 轮询
- ✅ 任务控制接口已实现（start/pause/cancel/delete）
- ✅ 模板 CRUD 已实现
- ⬜ ACS Worker 异步执行结果写回 `results`（Worker 待实现）
- ⬜ 脚本任务页查看单任务结果明细（前端已对接，后端数据待 Worker 填充）

#### 第三阶段：高级能力 🔄 进行中
- ✅ 参数库模块后端（表已建 + 仓储/服务/Handler 已实现 + 种子数据已入库）
- ⬜ 参数库前端集成（API 调用 + UI 展示）
- ⬜ 脚本上传解析与语法校验
- ⬜ 大批量设备并发、分批、失败重试策略
- ⬜ 定时/周期任务调度器
- ⬜ 审计日志自动写入

### 13.7 验收要点

| 验收项 | 标准 | 状态 |
|------|------|------|
| Console 页面 | 能完成设备选择、命令选择、参数配置、任务创建、结果轮询 | ✅ 已实现（设备列表仍为 Mock） |
| ScriptTask 页面 | 能新建、查看、筛选、启动、暂停、取消、终止任务 | ✅ 已实现 |
| 数据模型 | `mml_commands`、`mml_scripts`、`mml_tasks`、`mml_templates`、`mml_audit_log`、参数库 4 表结构完整 | ✅ 已落库 |
| 参数库后端 | 4 个 API 端点 + 种子数据（24 版本/1919 分组/7226 参数） | ✅ 已实现 |
| 参数库前端 | API 调用 + UI 展示集成到 CommandTree | ⬜ 待实现 |
| API 端点 | 24 个端点全部注册并可用（MML 主模块 20 + 参数库 4） | ✅ 已实现 |
| 模板系统 | 模板 CRUD + 克隆 + public/private 可见性隔离 | ✅ 已实现 |
| 任务状态机 | 6 种状态 + 合法转换校验 | ✅ 已实现 |
| 权限 | 只读/写操作/模板公共管理权限隔离正确 | ⬜ 待 Casbin 策略配置 |
| 安全 | 危险命令确认、审计日志表、参数校验均生效 | ⚠️ 审计日志待写入逻辑 |
| ACS Worker | 任务出队 → TR-069 SOAP 下发 → 结果写回 | ⬜ 待实现 |
| 文档一致性 | API、类型、DDL 与源码/实现保持一致 | ✅ 本次更新对齐 |