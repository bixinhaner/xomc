# OMC MML 维护命令功能需求设计文档

> **文档版本**: v3.1
> **创建日期**: 2026-04-14
> **更新日期**: 2026-04-14
> **适用项目**: OMC（基站网络运营管理系统）
> **技术栈**: Go + Gin + PostgreSQL + React 18 + Ant Design 5 + TanStack Query
> **变更说明**: v3.1 新增实时推送架构（SSE + 统一消息通道），更新自定义模板公有/私有分类目录机制；v3.0 基于新版 UI 截图与源码全量重写

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
| **后端** | Go 1.21+ · Gin 框架 · sqlc（原生 SQL）· PostgreSQL 15 |
| **权限引擎** | Casbin v2（PostgreSQL 适配器） |
| **前端** | React 18 · TypeScript · Vite · Ant Design 5 · TanStack Query (React Query) |
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
| PrivateTemplate | 私有命令模板，仅创建者可见 |
| PublicTemplate | 公共命令模板，所有用户可见 |
| SN | Serial Number，设备唯一序列号 |
| eNB | Evolved NodeB，4G 基站 |
| gNB | Next Generation NodeB，5G 基站 |

---

## 2. 功能总览

### 2.1 模块架构图

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

### 2.2 页面路由结构

```
/mml
├── /console    ← MML 命令控制台（src/pages/mml/Console/index.tsx）
└── /script     ← MML 脚本任务（src/pages/mml/ScriptTask/index.tsx）
```

菜单路径：**MML 管理 → MML 控制台 / 脚本任务**（参见主页截图左侧导航栏）

![登录后主页 - 左侧导航 MML 管理菜单](/tmp/mml_v2_01_home.png)

### 2.3 与 TR-069/LMT 的关系说明

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

## 3. MML 命令控制台（/mml/console）

### 3.1 页面布局（三栏布局）

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

### 3.2 左栏：设备选择面板（DeviceTree）

**组件文件**：`Console/components/DeviceTree.tsx`

**状态管理 Hook**：`Console/hooks/useDeviceSelection.ts`

#### 3.2.1 面板头部

| 元素 | 规格 |
|------|------|
| 标题 | "设备名称" + 已选数量 Badge（蓝色 Tag，数字为 `selectedDevices.length`） |
| 批量输入按钮 | `Button` size="small" type="primary" ghost，图标 `UserAddOutlined`，文本"批量输入"，点击触发 `setBatchSnModalOpen(true)` |

#### 3.2.2 搜索框

| 属性 | 值 |
|------|-----|
| 组件 | `Input` size="small" |
| placeholder | 搜索（来自 i18n `common.search`） |
| 前缀图标 | `SearchOutlined` |
| 功能 | `allowClear`，`onChange` → `onSearchChange`，支持 SN/名称模糊搜索 |

#### 3.2.3 产品类型筛选下拉框

| 属性 | 值 |
|------|-----|
| 组件 | `Select` size="small" style.width="100%" |
| placeholder | "按类型筛选" |
| allowClear | true |
| 选项来源 | `PRODUCT_TYPE_OPTIONS`（来自 `Console/types.ts`） |

**产品类型选项完整列表**（`PRODUCT_TYPE_OPTIONS`）：

| label | value |
|-------|-------|
| PM-B4860 | PM-B4860 |
| QAFA | QAFA |
| BaiBNX | BaiBNX |
| BaiBS5163 | BaiBS5163 |
| BaiBS5263 | BaiBS5263 |
| BTS | BTS |
| BSC | BSC |

![设备类型筛选下拉框](/tmp/mml_v2_06_device_filter.png)

#### 3.2.4 全选复选框

| 属性 | 值 |
|------|-----|
| 组件 | `Checkbox` |
| checked | `isAllSelected`（`selectedInFiltered === filteredDevices.length`） |
| indeterminate | `isIndeterminate`（部分选中时） |
| onChange | `onToggleSelectAll` |
| 文本 | "全选 (当前页已选/全部过滤数)" |
| disabled | `totalFiltered === 0` 时禁用 |

#### 3.2.5 设备列表

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

#### 3.2.6 分页

| 属性 | 值 |
|------|-----|
| 组件 | `Pagination` size="small" simple |
| 每页条数 | 8（`DEVICE_PAGE_SIZE`） |
| 显示条件 | `totalPages > 1` 时显示 |

#### 3.2.7 已选设备 Tag 展示区

仅当 `selectedDevices.length > 0` 时显示：

| 元素 | 说明 |
|------|------|
| 标题 | "已选设备 (N)" 蓝色文字，右侧删除全部按钮（`DeleteOutlined` + danger） |
| Tag 区域 | `maxHeight: 70px`，溢出滚动，每个 Tag 显示 `{type}-{sn}`，closable，关闭调用 `onRemoveDevice` |

![设备选择左栏](/tmp/mml_v2_03_device_panel.png)

#### 3.2.8 Mock 设备数据（`DEVICE_LIST`）

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

### 3.3 批量输入弹窗（BatchSnModal）

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

### 3.4 中栏：命令树面板（CommandTree）

**组件文件**：`Console/components/CommandTree.tsx`

**状态管理 Hook**：`Console/hooks/useCommandSelection.ts`

#### 3.4.1 面板头部

| 元素 | 说明 |
|------|------|
| 标题 | "命令树"（来自 i18n `nav.mml.commands`） |
| 已选命令 Badge | 仅有选中命令时显示，展示 `commandCode`，monospace 字体，蓝色边框圆角 Tag |

#### 3.4.2 搜索框

| 属性 | 值 |
|------|-----|
| placeholder | 搜索（i18n） |
| 功能 | allowClear，过滤 `commandName` 和 `commandCode` |

#### 3.4.3 分类筛选下拉框

| 属性 | 值 |
|------|-----|
| placeholder | "按分类筛选" |
| 选项 | 动态生成自 `categories`（来自 `MOCK_COMMANDS` 的 category 字段集合） |
| allowClear | true |

#### 3.4.4 命令树结构

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

**当前内置命令分类**（来自 `MOCK_COMMANDS`）：

| 分类 | 命令数 | 命令列表 |
|------|--------|---------|
| 总览 | 3 | 基本信息 `LST BASIC_INFO`、状态信息 `LST STATUS_INFO`、修改状态 `MOD STATUS_INFO` |
| 快速设置 | 3 | eNB配置查询 `LST eNB_CONFIG`、eNB配置修改 `MOD eNB_CONFIG`、小区配置查询 `LST CELL` |
| 告警管理 | 2 | 告警查询 `LST ALARM`、告警清除 `CLR ALARM` |
| 性能统计 | 1 | 性能统计查询 `LST PM` |
| 设备控制 | 2 | 设备重启 `RST DEVICE`、软件版本查询 `LST VERSION` |

![命令树中栏](/tmp/mml_v2_04_command_tree.png)

![命令树展开状态（选中高亮）](/tmp/mml_v2_08_cmd_tree_expanded.png)

#### 3.4.5 命令树交互

- 点击命令节点 → 调用 `onSelectCommand(cmd)`，更新 `selectedCommand`
- 已选中命令的节点背景高亮（蓝色）
- 空结果时显示 `Empty` 组件（"暂无匹配的命令"）

---

### 3.5 右栏：执行面板

右栏由两个子区域组成，均封装在独立组件中。

#### 3.5.1 终端输出区（TerminalPanel）

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

#### 3.5.2 执行面板（CommandInput）

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

#### 3.5.3 控制面板 Tab（参数动态表单）

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

#### 3.5.4 参数路径指定 Tab

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

#### 3.5.5 命令输入栏（控制面板 Tab 专属）

| 元素 | 说明 |
|------|------|
| 输入框前缀 | `>` 蓝色 monospace 粗体 |
| placeholder | "输入命令，多个命令用分号隔开" |
| 字体 | `'SFMono-Regular', Consolas, monospace` |
| 快捷键 | `Ctrl+Enter` / `Cmd+Enter` 触发执行 |
| 执行按钮 | `Button` type="primary" icon=`PlayCircleOutlined` "执行" |
| 禁用条件 | `selectedDevices.length === 0 || !selectedCommand` |

#### 3.5.6 底部操作栏

| 元素 | 说明 |
|------|------|
| 状态文字（左侧） | "N 设备 · 命令码"（控制面板 Tab），参数路径 Tab 不显示 |
| 执行按钮 | 参数路径 Tab 时在底部显示，复用 `handleExecute` |
| 重置按钮 | `ReloadOutlined`，调用 `onReset`（清空设备/命令/输出/参数） |
| 保存脚本按钮 | `SaveOutlined`，仅控制面板 Tab 显示，`onSaveScript` 回调 |

#### 3.5.7 危险命令确认弹窗

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

### 3.6 组件层次与数据流

#### 3.6.1 React 组件树

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
├── 右栏 div
│   ├── TerminalPanel (components/TerminalPanel.tsx)
│   │   └── props: lines, onClear, onDownload
│   └── CommandInput (components/CommandInput.tsx)
│       └── props: selectedDevices, selectedCommand, paramValues,
│                  onParamChange, onExecute, onReset, loading
└── BatchSnModal (components/BatchSnModal.tsx)
    └── props: open, onClose, onConfirm, existingSns, allDeviceSns
```

#### 3.6.2 状态管理（Hooks）

| Hook | 文件 | 管理的状态 |
|------|------|---------|
| `useDeviceSelection` | `hooks/useDeviceSelection.ts` | 设备选择、搜索、筛选、分页 |
| `useCommandSelection` | `hooks/useCommandSelection.ts` | 命令选择、搜索、分类过滤 |
| `useCommandExecution` | `hooks/useCommandExecution.ts` | 命令执行、输出行、loading 状态 |
| `useState` | `index.tsx` | `batchSnModalOpen`、`paramValues` |

#### 3.6.3 数据流向图

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

### 3.7 API 接口设计

#### 3.7.1 获取命令列表

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

#### 3.7.2 获取单个命令详情

```
GET /api/v1/mml/commands/:id
```

响应体：同列表中单条命令结构（`BackendMMLCommand`）。

#### 3.7.3 执行 MML 命令

```
POST /api/v1/mml/execute
```

**请求体**（`ExecuteHTTPRequest`）：

```json
{
  "command_code": "LST BASIC_INFO",
  "device_sns": ["ENB00001", "ENB00002"],
  "parameters": {
    "ALARM_LEVEL": 1
  },
  "task_name": "查询设备基本信息_2026-04-14"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| command_code | string | 否 | 命令编码（与 task 中的 commands 二选一）|
| device_sns | string[] | **是** | 目标设备 SN 列表 |
| parameters | object | 否 | 命令参数键值对 |
| task_name | string | 否 | 自定义任务名称，后端从 context 取 creator |

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

### 3.8 数据模型

#### 3.8.1 mml_commands 表

```sql
CREATE TABLE mml_commands (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    command_name  VARCHAR(200) NOT NULL,       -- 命令名称（中文显示名）
    command_code  VARCHAR(100) NOT NULL UNIQUE,-- 命令编码（如 LST BASIC_INFO）
    category      VARCHAR(50),                 -- 命令分类（总览/快速设置/告警管理等）
    description   TEXT,                        -- 命令描述
    rpc_method    VARCHAR(50) NOT NULL,        -- TR-069 RPC 方法名
    param_template JSONB,                      -- 参数模板（见下方结构定义）
    product_types JSONB DEFAULT '[]',          -- 适用产品类型列表（如 ["eNB","gNB"]）
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
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

#### 3.8.2 前端类型定义（`src/types/mml.ts`）

```typescript
export type MMLParamType = 'string' | 'number' | 'boolean' | 'enum' | 'range' | 'ipAddress' | 'list';

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
}

export interface MMLCommand {
  id: string;
  commandName: string;
  commandCode: string;
  category: string;
  description: string;
  params: MMLParam[];
  productTypes: string[];
}
```

#### 3.8.3 控制台本地类型（`Console/types.ts`）

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

## 4. MML 脚本任务（/mml/script）

### 4.1 页面布局

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

### 4.2 搜索筛选区域

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

### 4.3 工具栏

| 元素 | 说明 |
|------|------|
| "+ 新增" 按钮 | `Button` type="primary" icon=`PlusOutlined`，点击 `openAddModal` |

![工具栏区域](/tmp/mml_v2_13_script_toolbar.png)

---

### 4.4 任务列表表格

使用 `DataTable<ScriptRow>` 组件，`tableId="script-task"`，`scroll={{ x: 1400 }}`。

#### 4.4.1 列定义

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

#### 4.4.2 类型 Tag（CREATE_STATUS_MAP）

| value | text | color |
|-------|------|-------|
| active | 立即执行 | green |
| suspend | 挂起 | orange |
| timing | 定时执行 | blue |
| period | 周期任务 | purple |

#### 4.4.3 状态 Tag（TASK_STATUS_MAP）

| value | text | color |
|-------|------|-------|
| waiting | 等待中 | default |
| running | 执行中 | processing |
| paused | 已暂停 | warning |
| completed | 已完成 | success |
| terminated | 已终止 | error |
| exception | 异常 | error |

#### 4.4.4 结果 Tag（TASK_RESULT_MAP）

| value | text | color |
|-------|------|-------|
| success | 成功 | success |
| partial | 部分成功 | warning |
| failed | 失败 | error |

![任务列表表格](/tmp/mml_v2_14_task_table.png)

#### 4.4.5 操作列按钮

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

### 4.5 新建任务 Drawer

**组件**：`Drawer` 宽度 560px，`destroyOnClose`，标题"新建MML脚本任务"

![新建任务表单](/tmp/mml_v2_15_new_task.png)

#### 4.5.1 基本信息分区

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

#### 4.5.2 执行方式分区

**标识**：蓝色竖条 + "选择执行方式" 文字

使用 `Radio.Group` name="status"，默认值 `active`：

| Radio 值 | 显示文字 | 附加控件 |
|---------|---------|---------|
| active | 立即执行 | 无 |
| suspend | 挂起 | 无 |
| timing | 定时执行 | `DatePicker` showTime，禁选过去时间，宽度 185px，格式 "YYYY-MM-DD HH:mm:ss" |
| period | 周期任务 | `DatePicker.RangePicker`（禁选过去，宽度 240px）+ `:`分隔 + `DatePicker.TimePicker` format="HH:mm:ss" 宽度 110px |

> 周期任务的 Radio 使用独立 `Form.Item`（使用受控模式 `checked={executeType === 'period'}`）

#### 4.5.3 执行策略分区

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

#### 4.5.4 AddTaskForm 接口定义

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

### 4.6 任务详情弹窗

| 属性 | 值 |
|------|-----|
| 标题 | "任务详情" |
| 宽度 | 680px |
| footer | null（只读展示）|

**展示字段**：任务名称、创建者、创建时间、类型（文字）、状态（文字）、进度、结果（文字）、开始时间、结束时间

---

### 4.7 任务状态流转

```
                          ┌───────────────┐
                          │   waiting     │ ← 挂起(suspend)/定时(timing)/周期(period)创建时
                          └───────┬───────┘
                                  │ 用户手动 start / 到达定时时间
                                  ▼
             创建立即执行任务 ──► running ◄── 周期任务再次触发
                                  │
               ┌──────────────────┼──────────────────┐
               ▼                  ▼                  ▼
           completed           terminated          exception
          （全部完成）         （用户终止）         （异常中断）
               │
               └── paused ←─── running （用户暂停）
                       │
                       └──► running （用户恢复）

前端状态枚举（ScriptTask/index.tsx）：
  TaskStatus = 'waiting' | 'running' | 'paused' | 'completed' | 'terminated' | 'exception'

后端状态枚举（mml/model.go）：
  TaskStatus = 'pending' | 'running' | 'completed' | 'failed'
```

> **注意**：前端和后端状态枚举存在差异，前端使用 Mock 数据时有更细粒度的状态（paused/terminated/exception/waiting），后端目前只有4种状态。对齐工作见第12章。

---

### 4.8 API 接口设计

#### 4.8.1 获取脚本列表（实际用于脚本任务数据查询）

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

#### 4.8.2 获取任务列表

```
GET /api/v1/mml/tasks
```

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码 |
| page_size | int | 每页数量 |
| status | string | 状态过滤 |

**响应体**（同脚本列表格式，`items` 为 `MMLTask[]`）

#### 4.8.3 创建脚本

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

#### 4.8.4 更新脚本

```
PUT /api/v1/mml/scripts/:id
```

请求体同创建接口（`script_name` + `content` 必填）。

#### 4.8.5 删除脚本

```
DELETE /api/v1/mml/scripts/:id
```

返回 `204 No Content`。

#### 4.8.6 任务控制接口（待开发）

```
POST /api/v1/mml/tasks/:id/start      启动挂起任务
POST /api/v1/mml/tasks/:id/pause      暂停任务
POST /api/v1/mml/tasks/:id/terminate  终止任务
DELETE /api/v1/mml/tasks/:id          删除任务
GET /api/v1/mml/tasks/:id/results     获取任务结果明细（支持分页）
```

---

### 4.9 数据模型

#### 4.9.1 mml_scripts 表

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

#### 4.9.2 mml_tasks 表

```sql
CREATE TABLE mml_tasks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_name   VARCHAR(200),
    script_id   UUID REFERENCES mml_scripts(id) ON DELETE SET NULL,
    device_sns  JSONB NOT NULL,                 -- 目标设备 SN 列表
    commands    JSONB NOT NULL DEFAULT '[]',    -- [{command_code, parameters}]
    status      VARCHAR(20) NOT NULL DEFAULT 'pending',
    results     JSONB DEFAULT '[]',             -- 每台设备执行结果
    creator     VARCHAR(100),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 建议追加字段（支持定时/周期/重试策略）
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS execute_type   VARCHAR(20) DEFAULT 'active';
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS scheduled_at   TIMESTAMPTZ;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS period_start   TIMESTAMPTZ;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS period_end     TIMESTAMPTZ;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS period_time    TIME;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS offline_retry  BOOLEAN DEFAULT false;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS offline_wait   INT DEFAULT 60;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS failed_retry   BOOLEAN DEFAULT false;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS retry_count    INT DEFAULT 3;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS retry_interval INT DEFAULT 5;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS started_at     TIMESTAMPTZ;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS finished_at    TIMESTAMPTZ;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS total_devices  INT DEFAULT 0;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS success_count  INT DEFAULT 0;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS failed_count   INT DEFAULT 0;

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
    CreatedAt time.Time                `json:"created_at"`
    UpdatedAt time.Time                `json:"updated_at"`
}
```

---

## 5. 命令执行引擎

### 5.1 完整执行流程图

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

### 5.2 RPC 方法映射

| 操作类型 | rpc_method | TR-069 SOAP Action | 说明 |
|---------|-----------|-------------------|------|
| LST | GetParameterValues | `cwmp:GetParameterValues` | 查询参数值 |
| MOD | SetParameterValues | `cwmp:SetParameterValues` | 修改参数值 |
| ADD | AddObject | `cwmp:AddObject` | 添加对象实例 |
| RMV | DeleteObject | `cwmp:DeleteObject` | 删除对象实例 |
| RST（重启）| Reboot | `cwmp:Reboot` | 设备重启 |
| CLR（清除告警）| SetParameterValues | `cwmp:SetParameterValues` | 写入清除指令 |

**当前支持状态**（参照 handler.go）：

| RPC 方法 | 支持状态 |
|----------|---------|
| GetParameterValues | ✅ 已有框架 |
| SetParameterValues | ✅ 已有框架 |
| Reboot | ✅ 已有框架 |
| AddObject | ❌ 待实现 |
| DeleteObject | ❌ 待实现 |
| GetParameterNames | ⚠️ 部分支持 |

### 5.3 参数验证规则

| 参数类型 | 前端验证 |
|---------|---------|
| string | required 时非空；`pattern` 存在时正则匹配 |
| number | required 时非空；`minValue`/`maxValue` 范围校验 |
| boolean | 无特殊校验 |
| enum | 值必须在 `options` 中 |
| range | 等同 number，强制 min/max |
| ipAddress | IPv4/IPv6 格式正则 |
| list | 逗号分隔，每项独立校验 |

### 5.4 结果解析与展示

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

### 5.5 错误处理

| 错误类型 | 前端处理 |
|---------|---------|
| 未选设备 | 输出 `stderr` 行"错误：请先选择设备"，不发起请求 |
| 未选命令 | 输出 `stderr` 行"错误：请先选择命令"，不发起请求 |
| 危险命令 | 弹出确认 Modal，用户取消则不执行 |
| API 异常 | catch → 输出 `stderr` 行（`error.message`） |
| 设备离线 | 根据 offline_retry 策略（后端），前端展示结果状态 |
| RPC 超时 | 默认 60s，超时后标记 success=false |

### 5.6 批量执行策略

| 维度 | 当前实现 | 建议目标 |
|------|---------|---------|
| 并发数 | 逐台串行（for 循环）| 建议并发 10 台设备 |
| 失败继续 | ✅ 单台失败不影响其他设备 | - |
| 进度更新 | 每台完成后更新输出行 | 实时更新 task.results |
| 超大批量 | 无限制 | 建议超 100 台时分批（每批 50 台）|

---

## 6. 实时推送架构（SSE + 统一消息通道）

### 6.1 需求背景

MML 命令执行是异步过程，前端提交命令后需要实时回显执行进度和结果到终端输出区。同时，系统还需要支持**站内消息**（如告警通知、任务完成通知、系统公告等）。为避免重复开发两套实时通信机制，设计一套**统一消息通道**，同时满足命令执行结果回显和站内消息推送的需求。

### 6.2 技术选型：SSE（Server-Sent Events）

| 维度 | SSE | WebSocket | 结论 |
|------|-----|-----------|------|
| 协议 | 基于 HTTP/1.1，单向（服务端→客户端） | 全双工，独立协议 | 命令回显和站内消息均为服务端→客户端推送，不需要双向通信 |
| 断线重连 | 浏览器原生自动重连（`Last-Event-ID`） | 需要自行实现 | SSE 更简单可靠 |
| 代理/防火墙兼容 | 标准 HTTP，兼容性好 | 部分代理不支持 Upgrade | 运维网络环境 SSE 更友好 |
| 浏览器支持 | 除 IE 外全部支持 | 全部支持 | 本项目目标浏览器 Chrome/Edge 均支持 |
| 复杂度 | 低 | 高（需要连接管理、心跳、协议帧） | SSE 实现和维护成本更低 |
| 扩展性 | 可通过多个 SSE 连接分流 | 单连接承载所有消息 | 够用 |

**结论：选用 SSE**。命令回显和站内消息都是服务端主动推送场景，不需要客户端向服务端发送数据（命令执行仍走 REST POST）。SSE 在简单性、可靠性、兼容性上均优于 WebSocket。

### 6.3 统一消息通道架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                         后端消息产生源                                │
│                                                                     │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐  │
│  │  MML 命令执行     │  │  站内消息服务     │  │  其他事件源       │  │
│  │  (ACS Worker)    │  │  (告警/通知/公告)  │  │  (升级/配置变更)  │  │
│  └────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘  │
│           │                      │                      │           │
│           ▼                      ▼                      ▼           │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │              MessageHub（统一消息分发中心）                   │   │
│  │                                                             │   │
│  │  - 按用户 ID 分发（user-specific）                           │   │
│  │  - 按频道分发（channel-based）                               │   │
│  │  - 消息类型路由（mml_result / notification / system）        │   │
│  │  - 背压控制（slow consumer 丢弃策略）                        │   │
│  └───────────────────────────┬─────────────────────────────────┘   │
│                              │                                      │
│                              ▼                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │              SSE Endpoint                                    │   │
│  │              GET /api/v1/events/stream                       │   │
│  │                                                             │   │
│  │  - 认证: Bearer Token (query param or header)               │   │
│  │  - Last-Event-ID: 断线重连续传                               │   │
│  │  - Heartbeat: 每 30s 发送 `:keepalive\n\n`                  │   │
│  └───────────────────────────┬─────────────────────────────────┘   │
│                              │                                      │
└──────────────────────────────┼──────────────────────────────────────┘
                               │ SSE Stream
                               ▼
┌──────────────────────────────────────────────────────────────────────┐
│                          前端消费层                                   │
│                                                                      │
│  ┌──────────────────────┐  ┌──────────────────────────────────────┐ │
│  │  useEventStream      │  │  消息分发器                          │ │
│  │  (全局 SSE Hook)     │  │  根据 event.type 分发到对应处理器    │ │
│  │  - 自动连接/重连     │  │                                      │ │
│  │  - 认证 Token 注入   │  │  mml_result → TerminalPanel          │ │
│  │  - 连接状态管理      │  │  notification → 站内消息组件          │ │
│  │                      │  │  system → 全局通知                   │ │
│  └──────────────────────┘  └──────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────┘
```

### 6.4 SSE 消息格式

每条 SSE 消息使用标准 `event` + `data` 格式：

```
event: {消息类型}
id: {消息唯一ID（用于断线重连）}
data: {JSON payload}
```

#### 6.4.1 消息类型定义

| event 类型 | 用途 | payload 结构 |
|-----------|------|-------------|
| `mml_output` | MML 命令执行进度/结果回显 | `MMLExecutionEvent` |
| `mml_task_status` | MML 任务状态变更通知 | `MMLTaskStatusEvent` |
| `notification` | 站内消息（告警通知、系统公告等） | `NotificationEvent` |
| `system` | 系统级消息（版本升级、强制登出等） | `SystemEvent` |
| `:keepalive` | 心跳（SSE 注释行） | 无 |

#### 6.4.2 MML 命令执行事件（`mml_output`）

```json
{
  "event": "mml_output",
  "id": "evt-uuid-001",
  "data": {
    "task_id": "task-uuid",
    "device_sn": "ENB00001",
    "command_code": "LST BASIC_INFO",
    "executor": "admin",
    "output_type": "stdout | stderr | info | success",
    "text": "设备名称: 北京朝阳基站01",
    "timestamp": "2026-04-14T10:00:01Z",
    "progress": {
      "total": 3,
      "completed": 1,
      "current_device": "ENB00001"
    }
  }
}
```

#### 6.4.3 MML 任务状态变更事件（`mml_task_status`）

```json
{
  "event": "mml_task_status",
  "id": "evt-uuid-002",
  "data": {
    "task_id": "task-uuid",
    "old_status": "pending",
    "new_status": "running",
    "executor": "admin",
    "timestamp": "2026-04-14T10:00:00Z",
    "summary": {
      "total_devices": 3,
      "success_count": 0,
      "failed_count": 0
    }
  }
}
```

#### 6.4.4 站内通知事件（`notification`）

```json
{
  "event": "notification",
  "id": "evt-uuid-003",
  "data": {
    "notification_id": "notif-uuid",
    "type": "alarm | task_complete | system | approval",
    "priority": "critical | high | normal | low",
    "title": "告警通知",
    "content": "设备 ENB00001 产生紧急告警",
    "link": "/mml/console?task=task-uuid",
    "sender": "system",
    "created_at": "2026-04-14T10:00:05Z"
  }
}
```

### 6.5 后端实现设计

#### 6.5.1 SSE Endpoint

```
GET /api/v1/events/stream
```

**认证**：通过 Query 参数 `?token={jwt}` 或 Header `Authorization: Bearer {jwt}`。

**请求头**：

| Header | 说明 |
|--------|------|
| `Authorization` / `?token=` | JWT Token，认证当前用户身份 |
| `Last-Event-ID` | 断线重连时传递上次收到的最后一条消息 ID |
| `Accept` | `text/event-stream` |

**响应头**：

| Header | 值 |
|--------|-----|
| `Content-Type` | `text/event-stream` |
| `Cache-Control` | `no-cache` |
| `Connection` | `keep-alive` |
| `X-Accel-Buffering` | `no`（禁用 Nginx 缓冲） |

**连接生命周期**：

```
客户端连接
    │
    ├── 认证 JWT → 提取 user_id
    ├── 注册到 MessageHub（user_id → SSE channel）
    ├── 发送历史未读消息（可选，基于 Last-Event-ID）
    │
    ├── 循环:
    │   ├── MessageHub.Push(user_id, event) → 写入 SSE
    │   ├── 每 30s 发送 `:keepalive\n\n`
    │   └── 监听 ctx.Done() → 清理注册
    │
    └── 客户端断开 → 从 MessageHub 注销
```

#### 6.5.2 MessageHub 核心设计

```go
// MessageHub 统一消息分发中心
type MessageHub struct {
    mu       sync.RWMutex
    channels map[string]*UserChannel  // key: user_id
    store    MessageStore             // 消息持久化（支持断线重连）
}

// UserChannel 单用户的 SSE 通道
type UserChannel struct {
    userID    string
    ch        chan *SSEMessage
    lastAckID string
    created   time.Time
}

// SSEMessage SSE 消息结构
type SSEMessage struct {
    ID      string          `json:"id"`
    Event   string          `json:"event"`     // mml_output / mml_task_status / notification / system
    Data    json.RawMessage `json:"data"`
    UserID  string          `json:"-"`         // 目标用户（空表示广播）
}

// Publish 向指定用户推送消息
func (h *MessageHub) Publish(userID string, msg *SSEMessage) error

// PublishGlobal 全局广播（所有在线用户）
func (h *MessageHub) PublishGlobal(msg *SSEMessage) error

// Subscribe 订阅用户的 SSE 通道
func (h *MessageHub) Subscribe(userID string) (<-chan *SSEMessage, error)

// Unsubscribe 取消订阅
func (h *MessageHub) Unsubscribe(userID string)
```

#### 6.5.3 命令执行关联当前管理员

每次 MML 命令执行都会关联当前操作的管理员，消息精确推送给执行者：

```go
// ExecuteCommand 执行命令时记录 executor
func (s *MMLService) ExecuteCommand(ctx context.Context, req *ExecuteRequest) (*MMLTask, error) {
    // 从 context 获取当前管理员
    executor := auth.GetUsernameFromContext(ctx)

    task := &MMLTask{
        ID:        uuid.New(),
        Executor:  executor,       // 记录执行者
        DeviceSNs: req.DeviceSNs,
        Commands:  req.Commands,
        Status:    TaskPending,
    }

    // 持久化任务
    if err := s.repo.CreateTask(ctx, task); err != nil {
        return nil, fmt.Errorf("create task: %w", err)
    }

    // 通过 MessageHub 实时推送执行进度给 executor
    go s.executeAsync(task, executor)

    return task, nil
}

// executeAsync 异步执行，逐台推送结果
func (s *MMLService) executeAsync(task *MMLTask, executor string) {
    for _, sn := range task.DeviceSNs {
        result := s.executeForDevice(task, sn)

        // 推送单台设备执行结果到执行者的 SSE 通道
        s.hub.Publish(executor, &SSEMessage{
            Event: "mml_output",
            Data:  marshalMMLResult(task.ID, sn, result),
        })
    }

    // 推送任务完成状态
    s.hub.Publish(executor, &SSEMessage{
        Event: "mml_task_status",
        Data:  marshalTaskStatus(task.ID, TaskCompleted),
    })
}
```

#### 6.5.4 消息持久化与断线重连

| 维度 | 设计 |
|------|------|
| 持久化存储 | Redis Sorted Set（key: `sse:pending:{user_id}`，score: timestamp），TTL 1 小时 |
| 断线重连 | 客户端发送 `Last-Event-ID`，服务端从 Redis 读取该 ID 之后的消息重放 |
| 消息窗口 | 保留最近 1 小时的消息，超时自动清理 |
| 背压策略 | 用户通道缓冲区 256 条，慢消费者丢弃最老消息并推送 `buffer_overflow` 警告 |

#### 6.5.5 与站内消息服务的复用

站内消息服务和 MML 执行结果共用同一个 MessageHub 和 SSE 连接：

```
站内消息产生源:
  - 告警服务 → notification (alarm)
  - 任务完成 → notification (task_complete) + mml_task_status
  - 审批流程 → notification (approval)
  - 系统公告 → notification (system)
  - 设备上下线 → notification (device_status)

所有消息统一通过 MessageHub.Publish() 推送到用户的 SSE 通道，
前端通过 event type 区分处理逻辑。
```

### 6.6 前端实现设计

#### 6.6.1 全局 SSE Hook（useEventStream）

```typescript
// hooks/useEventStream.ts
// 全局单例，App 级别初始化，自动管理连接生命周期

interface SSEEvent {
  id: string;
  event: string;
  data: unknown;
}

function useEventStream() {
  // 连接状态: connecting | connected | disconnected
  const [status, setStatus] = useState<'connecting' | 'connected' | 'disconnected'>('disconnected');

  // 事件分发器
  const dispatcher = useRef<EventTarget>(new EventTarget());

  // 自动连接、认证、重连
  useEffect(() => {
    const token = getToken();
    const es = new EventSource(`/api/v1/events/stream?token=${token}`);

    es.onopen = () => setStatus('connected');
    es.onerror = () => setStatus('disconnected'); // 浏览器自动重连

    // 心跳处理
    es.addEventListener('keepalive', () => { /* 更新最后心跳时间 */ });

    // 统一消息分发
    const eventTypes = ['mml_output', 'mml_task_status', 'notification', 'system'];
    eventTypes.forEach(type => {
      es.addEventListener(type, (e) => {
        dispatcher.current.dispatchEvent(
          new CustomEvent(type, { detail: JSON.parse(e.data) })
        );
      });
    });

    return () => es.close();
  }, []);

  return { status, dispatcher: dispatcher.current };
}
```

#### 6.6.2 MML 终端输出消费

```typescript
// Console/hooks/useTerminalSSE.ts
// 在 Console 页面内监听 mml_output 事件，实时追加到终端输出

function useTerminalSSE(dispatcher: EventTarget) {
  const { appendLine } = useCommandExecution();

  useEffect(() => {
    const handler = (e: CustomEvent) => {
      const data = e.detail as MMLExecutionEvent;

      appendLine({
        text: `[${formatTime(data.timestamp)}] ${data.text}`,
        type: data.output_type,  // stdout | stderr | info | success
        timestamp: data.timestamp,
      });
    };

    dispatcher.addEventListener('mml_output', handler);
    return () => dispatcher.removeEventListener('mml_output', handler);
  }, [dispatcher, appendLine]);
}
```

#### 6.6.3 站内消息消费

```typescript
// hooks/useNotifications.ts
// 全局监听 notification 事件，更新站内消息状态

function useNotifications(dispatcher: EventTarget) {
  const queryClient = useQueryClient();

  useEffect(() => {
    const handler = (e: CustomEvent) => {
      const data = e.detail as NotificationEvent;

      // 更新未读消息计数（触发 Header 铃铛 Badge 刷新）
      queryClient.setQueryData(['notifications', 'unread_count'], (old: number) => old + 1);

      // 根据 priority 弹出全局提示
      if (data.priority === 'critical' || data.priority === 'high') {
        notification.open({
          message: data.title,
          description: data.content,
          duration: 0,  // 不自动关闭
        });
      }
    };

    dispatcher.addEventListener('notification', handler);
    return () => dispatcher.removeEventListener('notification', handler);
  }, [dispatcher, queryClient]);
}
```

### 6.7 数据模型

#### 6.7.1 站内消息表

```sql
CREATE TABLE notifications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         VARCHAR(100) NOT NULL,              -- 目标用户
    type            VARCHAR(20) NOT NULL,                -- alarm / task_complete / system / approval / device_status
    priority        VARCHAR(10) NOT NULL DEFAULT 'normal', -- critical / high / normal / low
    title           VARCHAR(200) NOT NULL,               -- 通知标题
    content         TEXT,                                -- 通知内容
    link            VARCHAR(500),                        -- 跳转链接
    sender          VARCHAR(100) DEFAULT 'system',       -- 发送者
    is_read         BOOLEAN NOT NULL DEFAULT false,      -- 已读标记
    read_at         TIMESTAMPTZ,                         -- 已读时间
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_user_unread ON notifications(user_id, is_read, created_at DESC);
CREATE INDEX idx_notifications_type ON notifications(type, created_at DESC);
```

#### 6.7.2 mml_tasks 表增强

在现有 `mml_tasks` 表基础上增加 `executor` 字段，关联执行命令的管理员：

```sql
-- mml_tasks 增加执行者字段
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS executor VARCHAR(100);
COMMENT ON COLUMN mml_tasks.executor IS '命令执行者（管理员用户名），用于 SSE 精确推送';
```

### 6.8 API 接口设计

#### 6.8.1 SSE 连接

```
GET /api/v1/events/stream?token={jwt}
```

响应：`text/event-stream`，长连接，持续推送。

#### 6.8.2 站内消息管理

```
GET    /api/v1/notifications              获取通知列表（分页）
GET    /api/v1/notifications/unread-count 获取未读数量
PUT    /api/v1/notifications/:id/read     标记已读
PUT    /api/v1/notifications/read-all     全部标记已读
DELETE /api/v1/notifications/:id          删除通知
```

### 6.9 性能与可靠性

| 维度 | 设计 |
|------|------|
| 连接数 | 单用户最多 1 个 SSE 连接（新连接踢掉旧连接） |
| 心跳间隔 | 30 秒 |
| 消息延迟 | < 500ms（内存直推，不经消息队列） |
| 断线重连 | 浏览器原生重连 + Last-Event-ID 补发 |
| 消息窗口 | Redis 保留最近 1 小时，断线超 1 小时后消息可从 DB 补查 |
| 并发 SSE | 预估 100 并发管理员，MessageHub 使用 ring buffer |
| Nginx 配置 | `proxy_buffering off; proxy_read_timeout 3600s;` |

---

## 7. 操作类型详解

### 7.1 操作类型总表

| 操作类型 | 中文含义 | TR-069 RPC | 典型命令模式 | 是否改变设备配置 | 是否要求危险确认 |
|---------|----------|------------|--------------|------------------|------------------|
| LST | 查询 | `GetParameterValues` | `LST BASIC_INFO`、`LST CELL` | 否 | 否 |
| MOD | 修改 | `SetParameterValues` | `MOD STATUS_INFO`、`MOD eNB_CONFIG` | 是 | 视命令而定 |
| ADD | 增加 | `AddObject` | `ADD XXX` | 是 | 是 |
| RMV | 删除 | `DeleteObject` | `RMV XXX` | 是 | 是 |

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
| PrivateTemplate | 私有命令 | 仅创建者本人 |
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

#### 8.2.1 分类目录机制

自定义命令模板在命令树中作为**一级分类**（与"总览""快速设置"等内置分类平级），在"自定义模板"一级分类下再分为"公有命令"和"私有命令"两个二级分类目录。每个二级分类节点右侧带有 `[+]` 添加图标按钮，点击可直接新增对应类型的自定义命令：

**命令树层级结构**：

```
命令树
├── ▼ 总览 (3)                    ← 一级分类（内置）
│       LST BASIC_INFO
│       LST STATUS_INFO
│       MOD STATUS_INFO
├── ▼ 快速设置 (3)                ← 一级分类（内置）
│       LST eNB_CONFIG
│       MOD eNB_CONFIG
│       LST CELL
├── ▼ 告警管理 (2)                ← 一级分类（内置）
│       LST ALARM
│       CLR ALARM
├── ▼ 性能统计 (1)                ← 一级分类（内置）
│       LST PM
├── ▼ 设备控制 (2)                ← 一级分类（内置）
│       RST DEVICE
│       LST VERSION
└── ▼ 自定义模板                  ← 一级分类（新增）
    ├── ▼ 公有命令 (5)  [+]       ← 二级分类 + 添加按钮（点击新增公有命令）
    │       命令A - LST BASIC_INFO
    │       命令B - MOD eNB_CONFIG
    │       命令C - LST CELL
    │       ...
    └── ▼ 私有命令      [+]       ← 二级分类 + 添加按钮（点击新增私有命令）
        ├── ▼ admin (3)           ← 三级：当前管理员的私有目录（自动创建）
        │       命令D - LST ALARM
        │       命令E - MOD STATUS_INFO
        └── ▼ operator_zhang (2)  ← 三级：其他管理员的私有目录（仅超管可见）
                命令F - LST PM
                命令G - RST DEVICE
```

**添加按钮 `[+]` 行为**：

| 按钮 | 位置 | 点击行为 |
|------|------|---------|
| 公有命令 `[+]` | "公有命令"节点右侧，图标 `PlusOutlined` | 弹出"新增公有命令"表单，填写命令名称、命令编码、操作类型、参数模板、描述等，提交后保存到公有命令列表 |
| 私有命令 `[+]` | "私有命令"节点右侧，图标 `PlusOutlined` | 弹出"新增私有命令"表单，同上，提交后自动保存到当前用户的私有目录下 |

**公有命令**：
- 位于"自定义模板 → 公有命令"下，**扁平展示**，不按管理员创建子目录
- 所有人共享同一个公有命令列表
- 任何有创建公有命令权限的用户都可以将命令添加到该分类
- 通过"公有命令"右侧的 `[+]` 按钮新增

**私有命令**：
- 位于"自定义模板 → 私有命令"下，按管理员用户名自动创建三级子目录
- 创建私有命令时，系统自动在"私有命令"下为当前管理员创建以用户名命名的专属目录
- **仅创建者本人和超级管理员**可见该目录及其下的命令
- 普通管理员只能看到自己的私有命令目录，看不到其他管理员的目录
- 超级管理员可以看到所有管理员的私有命令目录
- 通过"私有命令"右侧的 `[+]` 按钮新增

### 8.3 推荐数据模型

#### 8.3.1 mml_templates 表 DDL

```sql
CREATE TABLE mml_templates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_name   VARCHAR(200) NOT NULL,                 -- 模板名称
    command_code    VARCHAR(100) NOT NULL,                 -- 绑定命令编码
    operation_type  VARCHAR(20) NOT NULL,                  -- LST/MOD/ADD/RMV
    template_scope  VARCHAR(20) NOT NULL DEFAULT 'private',-- private/public
    category_group  VARCHAR(50),                           -- 分类组（关联 mml_commands.category，如"总览""快速设置"）
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
CREATE INDEX idx_mml_templates_scope_group ON mml_templates(template_scope, category_group);
CREATE INDEX idx_mml_templates_product_types_gin ON mml_templates USING GIN (product_types);
CREATE INDEX idx_mml_templates_parameters_gin ON mml_templates USING GIN (parameters);
```

**分类目录查询逻辑**：

| 模板类型 | 目录结构 | 查询条件 |
|---------|---------|---------|
| 公有命令 | 按命令分类（category_group）分组展示 | `WHERE template_scope = 'public'`，按 `category_group` 聚合 |
| 私有命令（本人） | 按管理员用户名自动创建一级目录，其下按 `category_group` 分组 | `WHERE template_scope = 'private' AND creator = '{current_user}'` |
| 私有命令（超管视角） | 按管理员用户名分目录，每个目录下按 `category_group` 分组 | `WHERE template_scope = 'private'`，按 `creator` + `category_group` 聚合 |

#### 8.3.2 前端类型建议

```typescript
interface MMLTemplate {
  id: string;
  templateName: string;
  commandCode: string;
  operationType: 'LST' | 'MOD' | 'ADD' | 'RMV';
  templateScope: 'private' | 'public';
  categoryGroup: string;  // 分类组，对应 mml_commands.category（如"总览""快速设置"）
  parameters: Record<string, string | number | boolean>;
  paramPaths: string[];
  description: string;
  productTypes: string[];
  creator: string;
  createdAt: string;
  updatedAt: string;
}
```

### 8.4 API 设计（待新增）

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

#### 8.4.5 复制公共模板为私有命令

```text
POST /api/v1/mml/templates/:id/clone
```

### 8.5 前端交互建议

| 位置 | 行为 |
|------|------|
| [CommandTree](file:///Users/cb/code/baicells/goomc/omcmb/webcode/src/pages/mml/Console/components/CommandTree.tsx) | 在命令树末尾增加一级分类”自定义模板”，其下分”公有命令”和”私有命令”两个二级分类，每个分类右侧带 `[+]` 添加按钮（`PlusOutlined` 图标） |
| 公有命令 `[+]` | 点击弹出”新增公有命令”表单，填写命令名称、命令编码、操作类型、参数模板、描述等，提交后保存到公有命令列表 |
| 私有命令 `[+]` | 点击弹出”新增私有命令”表单，同上，提交后自动保存到当前用户的私有目录下 |
| 控制面板 Tab | 增加”从自定义命令加载”下拉，可选中已有的公有/私有命令快速填充参数 |
| 参数路径 Tab | 加载自定义命令时同步恢复 `operationType` 与 `paramPaths` |
| 命令树自定义节点 | 右键或操作按钮支持：加载命令、编辑命令、删除命令、复制为我的私有命令 |

### 8.6 权限规则

| 动作 | 普通用户 | 运维管理员 | 系统管理员（超级管理员） |
|------|----------|------------|------------|
| 查看私有命令 | 仅本人 | 仅本人 | **全部**（可见所有管理员的私有命令目录） |
| 查看公有命令 | ✅ | ✅ | ✅ |
| 创建私有命令 | ✅ | ✅ | ✅ |
| 创建公有命令 | ❌ | ✅ | ✅ |
| 删除他人公有命令 | ❌ | ❌ | ✅ |
| 查看他人私有命令目录 | ❌ | ❌ | ✅ |
| 删除他人私有命令 | ❌ | ❌ | ✅ |

**分类目录可见性规则**：

| 用户角色 | 可见的命令树模板节点 |
|---------|-------------------|
| 普通用户 | "自定义模板 → 公有命令"全部 + "自定义模板 → 私有命令"下仅自己的目录 |
| 运维管理员 | 同上 |
| 超级管理员 | "自定义模板 → 公有命令"全部 + "自定义模板 → 私有命令"下**所有管理员**的目录 |

---

## 9. 内置命令种子数据

### 9.1 当前应落库的命令分类树

以下分类树以当前前端 [MOCK_COMMANDS](file:///Users/cb/code/baicells/goomc/omcmb/webcode/src/pages/mml/Console/constants.ts) 与后端 [MMLCommand](file:///Users/cb/code/baicells/goomc/omcgo/internal/mml/model.go) 数据模型为准，用于初始化 `mml_commands`。

```text
eNB / 4G
├── 总览
│   ├── LST BASIC_INFO
│   ├── LST STATUS_INFO
│   └── MOD STATUS_INFO
├── 快速设置
│   ├── LST eNB_CONFIG
│   ├── MOD eNB_CONFIG
│   └── LST CELL
├── 告警管理
│   ├── LST ALARM
│   └── CLR ALARM
├── 性能统计
│   └── LST PM
└── 设备控制
    ├── RST DEVICE
    └── LST VERSION

gNB / 5G
├── 总览
│   ├── LST BASIC_INFO
│   ├── LST STATUS_INFO
│   └── MOD STATUS_INFO
├── 快速设置
│   └── LST CELL
├── 告警管理
│   ├── LST ALARM
│   └── CLR ALARM
├── 性能统计
│   └── LST PM
└── 设备控制
    ├── RST DEVICE
    └── LST VERSION

GSM
├── 总览
│   ├── LST BASIC_INFO
│   └── LST STATUS_INFO
├── 告警管理
│   ├── LST ALARM
│   └── CLR ALARM
├── 性能统计
│   └── LST PM
└── 设备控制
    └── LST VERSION
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

```sql
INSERT INTO mml_commands (
    id, command_name, command_code, category, description, rpc_method, param_template, product_types, created_at
) VALUES
(
    '00000000-0000-0000-0000-000000000001',
    '基本信息',
    'LST BASIC_INFO',
    '总览',
    '查询设备基本信息',
    'GetParameterValues',
    '{}'::jsonb,
    '["eNB", "gNB", "GSM"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0000-000000000002',
    '状态信息',
    'LST STATUS_INFO',
    '总览',
    '查询设备状态信息',
    'GetParameterValues',
    '{}'::jsonb,
    '["eNB", "gNB", "GSM"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0000-000000000003',
    '修改状态',
    'MOD STATUS_INFO',
    '总览',
    '修改设备状态信息配置',
    'SetParameterValues',
    '{"STATUS":{"type":"enum","required":true,"description":"状态","options":[{"label":"启用","value":1},{"label":"禁用","value":0}]}}'::jsonb,
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0000-000000000004',
    'eNB配置查询',
    'LST eNB_CONFIG',
    '快速设置',
    '查询eNB快速配置信息',
    'GetParameterValues',
    '{}'::jsonb,
    '["eNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0000-000000000005',
    'eNB配置修改',
    'MOD eNB_CONFIG',
    '快速设置',
    '修改eNB快速配置',
    'SetParameterValues',
    '{"FREQ":{"type":"number","required":true,"description":"频点","min_value":0,"max_value":65535},"PCI":{"type":"number","required":true,"description":"物理小区标识","min_value":0,"max_value":503},"PWR":{"type":"number","required":false,"description":"发射功率(dBm)","min_value":-30,"max_value":50}}'::jsonb,
    '["eNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0000-000000000006',
    '小区配置查询',
    'LST CELL',
    '快速设置',
    '查询小区配置信息',
    'GetParameterValues',
    '{"CELLID":{"type":"number","required":false,"description":"小区ID，不填则查询全部","min_value":0,"max_value":65535}}'::jsonb,
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0000-000000000007',
    '告警查询',
    'LST ALARM',
    '告警管理',
    '查询设备当前告警',
    'GetParameterValues',
    '{"ALARM_LEVEL":{"type":"enum","required":false,"description":"告警级别","options":[{"label":"紧急","value":1},{"label":"重要","value":2},{"label":"一般","value":3},{"label":"提示","value":4}]}}'::jsonb,
    '["eNB", "gNB", "GSM"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0000-000000000008',
    '告警清除',
    'CLR ALARM',
    '告警管理',
    '清除指定告警',
    'SetParameterValues',
    '{"ALARM_ID":{"type":"string","required":true,"description":"告警ID"}}'::jsonb,
    '["eNB", "gNB", "GSM"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0000-000000000009',
    '性能统计查询',
    'LST PM',
    '性能统计',
    '查询设备性能统计信息',
    'GetParameterValues',
    '{"START_TIME":{"type":"string","required":true,"description":"开始时间"},"END_TIME":{"type":"string","required":true,"description":"结束时间"}}'::jsonb,
    '["eNB", "gNB", "GSM"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0000-000000000010',
    '设备重启',
    'RST DEVICE',
    '设备控制',
    '重启指定设备',
    'Reboot',
    '{"DELAY":{"type":"number","required":false,"description":"延迟秒数","min_value":0,"max_value":3600}}'::jsonb,
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0000-000000000011',
    '软件版本查询',
    'LST VERSION',
    '设备控制',
    '查询设备软件版本',
    'GetParameterValues',
    '{}'::jsonb,
    '["eNB", "gNB", "GSM"]'::jsonb,
    NOW()
);
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
│   ├── DeviceTree
│   ├── CommandTree
│   ├── TerminalPanel
│   ├── CommandInput
│   └── BatchSnModal
└── /mml/script
    ├── FilterBar
    ├── DataTable
    ├── Drawer(新建任务)
    ├── Modal(任务详情)
    └── Modal(执行结果)

数据层
├── useMMLCommands / useAllMMLCommands
├── useMMLScripts / useMMLTasks
├── useExecuteMMLCommand / useCreateMMLTask
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
| 数据隔离 | 私有命令与私有脚本需按 creator 过滤 |

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

| 项目 | Mock/前端现状 | 真实后端现状 | 问题 |
|------|---------------|--------------|------|
| 命令列表分页参数 | 前端 `pageSize` | 后端 `page_size` | 查询参数命名不一致，真实分页可能失效 |
| 脚本列表分页参数 | 前端 `pageSize` | 后端 `page_size` | 同上 |
| 任务列表分页参数 | 前端 `pageSize` | 后端 `page_size` | 同上 |
| 命令执行请求体 | 前端 `payload.params` | 后端字段名 `parameters` | 字段名不一致，参数可能丢失 |
| 命令执行返回值 | [mmlService](file:///Users/cb/code/baicells/goomc/omcmb/webcode/src/mock/services/mmlService.ts) 返回 `Array<{deviceSn,result}>` | [handler.go](file:///Users/cb/code/baicells/goomc/omcgo/internal/mml/handler.go) 返回 `MMLTask` | 前端执行逻辑与真实接口不兼容 |
| `useCommandExecution` 结果使用 | 直接访问 `result.success` | 实际 mutation 返回数组 | 类型/运行时逻辑错误 |
| 获取脚本详情 | 前端存在 `getScriptById()` | 后端未注册 `GET /mml/scripts/:id` | 路由缺失 |
| 创建任务 | 前端 `createTask()` POST `/mml/execute`，发送 `script_id/commands` | 后端 `ExecuteHTTPRequest` 仅接收 `command_code/device_sns/parameters/task_name` | 创建任务接口模型不匹配 |
| 执行脚本 | 前端 `executeScript()` POST `/mml/execute`，发送 `script_id` | 后端无 `script_id` 字段 | 脚本执行未真正打通 |
| 任务状态枚举 | [types/mml.ts](file:///Users/cb/code/baicells/goomc/omcmb/webcode/src/types/mml.ts) 为 `pending/running/success/failed/cancelled`；脚本页 Mock 又使用 `waiting/paused/terminated/exception` | 后端 [model.go](file:///Users/cb/code/baicells/goomc/omcgo/internal/mml/model.go) 为 `pending/running/completed/failed` | 三套枚举并存，需统一 |

### 13.2 后端待实现清单

1. 新增 `GET /mml/scripts/:id`。
2. 为任务新增启动、暂停、终止、删除、结果明细接口。
3. 将 `ExecuteRequest` 扩展为支持 `script_id`、`commands[]`、执行策略字段。
4. 补齐 `AddObject`、`DeleteObject` 的 ACS Worker 实现。
5. 增加 `mml_templates` 仓储、服务、Handler。
6. 为 `mml_tasks` 增加调度字段、统计字段、开始/结束时间。
7. 增加统一状态机和状态转换校验。
8. 增加任务轮询接口的结果分页能力。
9. 增加命令执行审计日志表。

### 13.3 前端待完善清单

1. 将控制台命令树从 [MOCK_COMMANDS](file:///Users/cb/code/baicells/goomc/omcmb/webcode/src/pages/mml/Console/constants.ts) 切换到真实接口。
2. 修复 [mmlApi](file:///Users/cb/code/baicells/goomc/omcmb/webcode/src/services/api/mmlApi.ts) 中 `pageSize` / `page_size`、`params` / `parameters` 字段不一致问题。
3. 修复 [useCommandExecution](file:///Users/cb/code/baicells/goomc/omcmb/webcode/src/pages/mml/Console/hooks/useCommandExecution.ts) 对 mutation 返回值的错误假设。
4. 控制台执行后改为展示“任务已创建”，并轮询任务详情，不再假定同步成功。
5. 脚本任务页改造为真实服务端分页、筛选、控制动作。
6. 新建任务 Drawer 补充设备选择、脚本校验、模板加载、批量 SN 解析等真实能力。
7. 补充模板管理入口和从模板加载交互。
8. 将脚本任务页所有中文硬编码迁移到 i18n。

### 13.4 分阶段实施建议

#### 第一阶段：接口对齐
- 统一任务状态枚举。
- 修复前端 API 参数名与返回值映射。
- 打通命令列表、脚本列表、任务列表的真实接口。

#### 第二阶段：执行链路闭环
- `/mml/execute` 返回 task id 后，前端轮询 `GET /mml/tasks/:id`。
- Worker 异步执行结果写回 `results`。
- 脚本任务页支持查看单任务结果明细。

#### 第三阶段：高级能力
- 自定义命令模板。
- 脚本校验与模板导入。
- 大批量设备并发、分批、失败重试。

### 13.5 验收要点

| 验收项 | 标准 |
|------|------|
| Console 页面 | 能完成设备选择、命令选择、参数配置、任务创建、结果轮询 |
| ScriptTask 页面 | 能新建、查看、筛选、启动、终止任务 |
| 数据模型 | `mml_commands`、`mml_scripts`、`mml_tasks`、`mml_templates` 结构完整 |
| 权限 | 只读/写操作/模板公共管理权限隔离正确 |
| 安全 | 危险命令确认、审计日志、参数校验均生效 |
| 文档一致性 | API、类型、DDL 与源码/实现保持一致 |
