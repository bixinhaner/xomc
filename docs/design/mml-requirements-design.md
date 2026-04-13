# OMC MML 维护命令功能需求设计文档

> **文档版本**: v1.0  
> **创建日期**: 2026-04-13  
> **适用项目**: OMC（基站网络运营管理系统）  
> **技术栈**: Go + Gin + PostgreSQL + TimescaleDB + React 18 + Ant Design 5 + TanStack Query

---

## 1. 文档概述

### 1.1 文档目的

本文档基于 OMC 项目当前 MML 维护命令模块的实现现状，整理已有功能的详细规格，明确哪些功能已实现、哪些待完善、哪些需要新开发，并提出分阶段实施建议，作为后续开发和测试的功能需求设计依据。

文档覆盖 MML 模块的四大功能子系统：命令控制台、脚本管理、任务管理、设备重启管理，以及贯穿所有子系统的命令执行引擎。

### 1.2 适用范围

本文档适用于 OMC 系统维护（MML）模块，包括：

| 子模块 | 说明 |
|--------|------|
| MML 命令控制台 | 设备选择、命令树浏览、参数输入、实时执行结果展示 |
| MML 脚本管理 | 用户自定义脚本 CRUD、私有/公共模板、批量执行 |
| MML 任务管理 | 脚本任务列表、执行进度、结果查询、任务控制 |
| 命令执行引擎 | TR-069 RPC 调用、参数验证、批量执行策略、异步任务 |
| 设备重启管理 | 即时重启任务、周期重启配置 |

### 1.3 项目技术栈

- **后端**: Go 1.21+ + Gin 框架 + sqlc（原生 SQL）
- **数据库**: PostgreSQL 15 + TimescaleDB
- **权限引擎**: Casbin v2（PostgreSQL 适配器）
- **前端**: React 18 + TypeScript + Vite + Ant Design 5 (antd) + TanStack Query (React Query)
- **国际化**: i18next
- **设备通信协议**: TR-069 / CWMP (SOAP over HTTP)
- **本地维护协议**: LMT（Local Management Tool，17547 端口）

### 1.4 术语定义

| 术语 | 全称/说明 |
|------|-----------|
| MML | Man-Machine Language，人机语言，用于设备配置和维护的命令语言 |
| LMT | Local Management Tool，本地管理工具，TR-069 的简化本地版本 |
| TR-069 | Technical Report 069，CPE WAN 管理协议（CWMP），用于 OMC 远程管理基站 |
| CWMP | CPE WAN Management Protocol，CPE 广域网管理协议 |
| RPC | Remote Procedure Call，TR-069 中的远程过程调用方法集合 |
| ACS | Auto-Configuration Server，自动配置服务器，即 OMC 的 TR-069 服务端 |
| CPE | Customer Premises Equipment，用户端设备，即基站 |
| BSC | Base Station Controller，基站控制器（4G 产品类型）|
| gNB | Next Generation NodeB，5G 基站 |
| eNB | Evolved NodeB，4G 基站 |
| RTS | Radio Transmission System，射频传输系统 |
| BAIBLQ | Baicells LQ 产品型号 |

---

## 2. 功能总览

### 2.1 功能模块架构图

```
┌──────────────────────────────────────────────────────────────────┐
│                      OMC MML 维护模块                             │
├──────────────┬───────────────┬──────────────┬───────────────────┤
│  MML 控制台  │  MML 脚本管理  │  MML 任务管理 │    设备重启管理   │
│  (Console)   │   (Scripts)   │   (Tasks)    │     (Reboot)      │
├──────────────┴───────────────┴──────────────┴───────────────────┤
│                       命令执行引擎                                │
│          (TR-069 RPC → ACS Worker → 设备)                        │
└──────────────────────────────────────────────────────────────────┘
```

### 2.2 模块间关系图

```
设备 (devices)
  └── 按产品类型筛选 (BSC/RTS/gNB/CPE...)
        └── 选择目标设备 (device_sns[])
              │
              ├──► MML 命令 (mml_commands)
              │      └── 命令模板 (param_template JSONB)
              │
              ├──► MML 脚本 (mml_scripts)
              │      └── 脚本内容 (content TEXT)
              │
              └──► MML 任务 (mml_tasks)
                     ├── status: pending/running/completed/failed
                     └── results: [{device_sn, success, raw_output}]

命令执行路径:
用户操作 → POST /api/v1/mml/execute → ACS Worker → TR-069/LMT → 设备 → 结果写回 mml_tasks.results
```

### 2.3 与 TR-069/LMT 的关系说明

OMC 系统通过两种协议与基站设备通信：

| 协议 | 端口 | 使用场景 | 特点 |
|------|------|----------|------|
| TR-069/CWMP | 7547 | OMC 远程批量管理 | 长会话、异步、支持批量操作 |
| LMT | 17547 | 本地维护工具 | 临时会话、即时响应、单次请求-响应 |

MML 命令通过 OMC ACS Worker 封装为 TR-069 SOAP 消息下发到设备，支持的 RPC 方法包括：

```
GetParameterValues / SetParameterValues / GetParameterNames
GetParameterAttributes / SetParameterAttributes
AddObject / DeleteObject
Download / Upload
Reboot / FactoryReset
GetRPCMethods
```

---

## 3. MML 命令控制台

> 老系统参考截图：
> - 主界面布局（两栏）：[/tmp/mml_03_mml_page.png](/tmp/mml_03_mml_page.png)
> - 帮助面板：[/tmp/mml_04_help.png](/tmp/mml_04_help.png)
> - 命令输入：[/tmp/mml_05_command_input.png](/tmp/mml_05_command_input.png)
> - 执行结果：[/tmp/mml_06_execution_result.png](/tmp/mml_06_execution_result.png)
> - 完整页面（全宽展开）：[/tmp/mml_14_batch_input_expanded.png](/tmp/mml_14_batch_input_expanded.png)
> - 产品类型切换（5G）：[/tmp/mml_16_5g_mml_page.png](/tmp/mml_16_5g_mml_page.png)

### 3.1 功能需求清单

| 需求编号 | 功能描述 | 状态 | 优先级 |
|----------|----------|------|--------|
| MML-CON-01 | 按产品类型（BSC/RTS/BAIBLQ/gNB 等）筛选设备列表 | ✅ 已实现（前端 DeviceTree 组件）| P0 |
| MML-CON-02 | 支持基站编码/名称模糊搜索 | ✅ 已实现 | P0 |
| MML-CON-03 | 多选设备（Checkbox 批量选择） | ✅ 已实现（useDeviceSelection hook）| P0 |
| MML-CON-04 | 批量输入 SN（弹窗形式粘贴 SN 列表）| ✅ 已实现（BatchSnModal 组件）| P1 |
| MML-CON-05 | 命令树展示（按分类折叠树形结构）| ✅ 已实现（CommandTree 组件）| P0 |
| MML-CON-06 | 命令树搜索（编码/名称关键词过滤）| ✅ 已实现 | P1 |
| MML-CON-07 | 选中命令后展示命令详情及参数说明 | ✅ 已实现（CommandInput 组件）| P0 |
| MML-CON-08 | 参数输入表单（支持 string/number/enum/range 等类型）| ✅ 已实现 | P0 |
| MML-CON-09 | 操作面板：自由输入 MML 命令文本 | ✅ 已实现（TerminalPanel 组件）| P0 |
| MML-CON-10 | 参数路径指定（手动输入 TR-069 参数路径）| ✅ 已实现（Tab 切换）| P1 |
| MML-CON-11 | 点击"执行"发起命令执行并显示结果 | ✅ 已实现（POST /mml/execute）| P0 |
| MML-CON-12 | 执行结果实时展示（含时间戳、设备 SN、原始输出）| ⚠️ 待完善（当前任务异步，结果不实时）| P0 |
| MML-CON-13 | 帮助面板（产品类型/参数列表查询）| ⚠️ 待完善（后端无 Help API）| P2 |
| MML-CON-14 | 结果导出（下载为 CSV/TXT 文件）| ❌ 待开发 | P2 |
| MML-CON-15 | 四步引导流程（选设备→选命令→配参数→查结果）| ✅ 已实现（Steps 组件展示）| P1 |
| MML-CON-16 | 支持多 Tab 多设备同时操作 | ✅ 老系统有，当前新系统待实现 | P2 |
| MML-CON-17 | 脚本 Tab（切换到 MML 脚本管理视图）| ✅ 已实现（Tab 切换）| P1 |
| MML-CON-18 | 设备私有模板/公共模板展示（脚本库）| ✅ 老系统截图可见，新系统已有脚本管理 | P1 |

**状态说明**：✅ 已实现 | ⚠️ 待完善 | ❌ 待开发

### 3.2 页面布局设计（三栏布局详细规格）

新系统采用三栏布局，比老系统（两栏）更清晰：

```
┌─────────────────────────────────────────────────────────────────────────┐
│  顶部工具栏: 模块标题 + 操作按钮（重置 / 帮助）                          │
├──────────────────┬──────────────────┬───────────────────────────────────┤
│     左栏 (1fr)   │     中栏 (1fr)   │           右栏 (2fr)              │
│                  │                  │  ┌─────────────────────────────┐  │
│   基站设备选择   │    命令树         │  │     终端输出区 (40%)        │  │
│   ─────────────  │    ──────────     │  │   时间戳 + 原始输出文本     │  │
│   产品类型下拉   │    搜索框         │  └─────────────────────────────┘  │
│   搜索框         │    树形结构       │  ┌─────────────────────────────┐  │
│   设备列表表格   │    (分类>命令)    │  │  操作面板 / 参数路径 (60%)  │  │
│   (多选)         │                  │  │  MML 文本输入框             │  │
│   批量输入按钮   │                  │  │  已选设备显示               │  │
│                  │                  │  │  [执行] 按钮                │  │
│                  │                  │  └─────────────────────────────┘  │
└──────────────────┴──────────────────┴───────────────────────────────────┘
```

**列宽比例**: `1fr : 1fr : 2fr`

#### 3.2.1 左栏 - 设备选择与产品类型

| 组件 | 规格 |
|------|------|
| 产品类型下拉 | Select 组件，枚举值：BSC / RTS / BAIBLQ / QAFA / QRTB-SC / QRTB-DC / CR-B4860-DC / CR-B4860-SC / QAFB / CR-B4860-CA / BTS（4G）；gNB（5G）|
| 搜索框 | 输入基站编码/名称关键词实时过滤 |
| 设备列表 | Table 组件，列：序号 / 基站编码 / 基站名称，支持分页（默认 20 条/页）|
| 多选 | Checkbox 全选 + 单行选择，选中数量显示"已选(N)" |
| 批量输入 | 按钮触发 BatchSnModal，允许粘贴多行 SN |
| 在线状态 | 行内显示设备在线（绿色）/ 离线（红色）状态标记 |

**老系统截图对应**：[/tmp/mml_03_mml_page.png](/tmp/mml_03_mml_page.png) 左半部分"基站设备"区域；[/tmp/mml_14_batch_input_expanded.png](/tmp/mml_14_batch_input_expanded.png) 展示了完整的产品类型下拉选项（BSC/RTS/BAIBLQ 等）。

#### 3.2.2 中栏 - 命令树

| 组件 | 规格 |
|------|------|
| 搜索框 | 按命令编码/名称关键词过滤树节点 |
| 树形结构 | Ant Design Tree 组件，一级分类为文件夹，二级为具体命令 |
| 命令分类 | 对应 mml_commands.category 字段（query / config / maintenance）|
| 4G 命令分类 | BSC配置 > 基本信息 / BTS / Msc / Cs7 / Mgw / DNS / keepalived |
| RTS 命令分类 | 设置信息 / 网络设置 / 同步 / 性能配置 / 管理设置 / SAS设置 / 高级配置 / 无线配置 > 基本配置 |
| 5G gNB 命令分类 | 无线配置 > 基本配置 / QoS业务设置 / 邻区配置 / ANR / 性能配置 / Xn / 高级设置 / LGW |
| 脚本库（命令树底部）| 私有模板（PrivateTemplate，按用户分组）/ 公共模板（PublicTemplate），支持新建 (+) |

**老系统截图对应**：[/tmp/mml_05_command_input.png](/tmp/mml_05_command_input.png) 展示了脚本库树（PrivateTemplate > admin > 123/test/eq）；[/tmp/mml_15_complete_mml_page.png](/tmp/mml_15_complete_mml_page.png) 展示 RTS 命令树；[/tmp/mml_16_5g_mml_page.png](/tmp/mml_16_5g_mml_page.png) 展示 5G gNB 命令树。

#### 3.2.3 右栏 - 参数输入与执行面板

**上半部分：终端输出区（40%）**

| 组件 | 规格 |
|------|------|
| 结果 Tab | 显示执行结果（时间戳 + 设备SN列表 + 原始输出文本）|
| 帮助 Tab | 显示当前选中命令的参数说明文档 |
| 四步引导 | 未执行时显示步骤指引（选设备→选命令→配参数→查结果）|
| 工具栏图标 | 清空结果（扫把图标）/ 下载结果（下载图标）|

**下半部分：操作面板（60%）**

| 组件 | 规格 |
|------|------|
| 操作面板 Tab | 自由文本输入框，支持多个命令（分号分隔），placeholder: "请输入MML命令，多个命令用分号隔开" |
| 参数路径指定 Tab | 直接输入 TR-069 参数路径（如 `Device.X_BAICELLS_COM.RTS.BasicInfo.`）|
| 已选设备显示 | Select/Tag 组件显示当前选中的设备 SN 列表 |
| 执行按钮 | 主色调橙色按钮，点击触发 POST /api/v1/mml/execute |

**老系统截图对应**：[/tmp/mml_06_execution_result.png](/tmp/mml_06_execution_result.png) 展示了执行结果输出格式（含时间戳、SN列表、错误信息）。

### 3.3 数据模型（mml_commands 表 DDL）

```sql
-- 来源：omcgo/migrations/000007_system_infra.sql
CREATE TABLE mml_commands (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    command_name  VARCHAR(200) NOT NULL,             -- 命令名称（中文）
    command_code  VARCHAR(100) NOT NULL UNIQUE,      -- 命令编码（如 LST_DEVPARAM）
    category      VARCHAR(50),                       -- 命令分类（query/config/maintenance）
    description   TEXT,                              -- 命令描述
    rpc_method    VARCHAR(50) NOT NULL,              -- TR-069 RPC 方法名
    param_template JSONB,                            -- 参数模板（含类型/必填/默认值等元数据）
    product_types JSONB DEFAULT '[]',               -- 适用产品类型列表
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 索引（来源：000011_optimize_indexes.sql）
CREATE INDEX IF NOT EXISTS idx_mml_commands_param_template_gin
    ON mml_commands USING GIN (param_template);
CREATE INDEX IF NOT EXISTS idx_mml_commands_product_types_gin
    ON mml_commands USING GIN (product_types);
```

**param_template JSONB 结构示例**：

```json
{
  "parameter_path": {
    "type": "string",
    "required": true,
    "description": "TR-069 参数路径",
    "default_value": "Device.",
    "pattern": "^Device\\."
  },
  "next_level": {
    "type": "boolean",
    "required": false,
    "description": "是否获取下级参数",
    "default_value": false
  }
}
```

**预置命令（Seed Data）**：

| command_code | command_name | rpc_method | category |
|--------------|--------------|------------|----------|
| LST_DEVPARAM | 查询设备参数 | GetParameterValues | query |
| SET_DEVPARAM | 设置设备参数 | SetParameterValues | config |
| RST_DEV | 设备重启 | Reboot | maintenance |

> **待完善**：当前预置命令仅 3 条，需根据老系统 BSC/RTS/gNB 命令树扩充到完整的命令目录（约 50~100 条）。

### 3.4 API 接口设计

#### 3.4.1 获取命令列表

```
GET /api/v1/mml/commands
```

**Query 参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页数量，默认 20 |
| category | string | 否 | 命令分类过滤 |
| search | string | 否 | 关键词搜索（命令名/编码）|

**响应体**：

```json
{
  "items": [
    {
      "id": "uuid",
      "command_name": "查询设备参数",
      "command_code": "LST_DEVPARAM",
      "category": "query",
      "description": "查询设备TR069参数树",
      "rpc_method": "GetParameterValues",
      "param_template": { "parameter_path": "Device." },
      "product_types": ["BSC", "RTS", "gNB"],
      "created_at": "2026-01-01T00:00:00Z"
    }
  ],
  "total": 100,
  "page": 1,
  "page_size": 20,
  "total_pages": 5
}
```

#### 3.4.2 获取单个命令详情

```
GET /api/v1/mml/commands/:id
```

**响应体**：同列表中单条命令结构。

#### 3.4.3 执行 MML 命令

```
POST /api/v1/mml/execute
```

**请求体**：

```json
{
  "command_code": "LST_DEVPARAM",
  "device_sns": ["F4F1F7EF0A0D5C4B2E", "0D59FBEC36F218D695"],
  "parameters": {
    "parameter_path": "Device.DeviceInfo."
  },
  "task_name": "查询设备信息_2026-04-13"
}
```

**响应体**（返回创建的任务对象）：

```json
{
  "id": "task-uuid",
  "task_name": "查询设备信息_2026-04-13",
  "device_sns": ["F4F1F7EF0A0D5C4B2E", "0D59FBEC36F218D695"],
  "commands": [
    {
      "command_code": "LST_DEVPARAM",
      "rpc_method": "GetParameterValues",
      "parameters": { "parameter_path": "Device.DeviceInfo." }
    }
  ],
  "status": "pending",
  "results": [],
  "creator": "admin",
  "created_at": "2026-04-13T10:00:00Z",
  "updated_at": "2026-04-13T10:00:00Z"
}
```

> **注意**：当前执行接口为异步创建任务，**不立即返回执行结果**。前端需轮询 `GET /api/v1/mml/tasks/:id` 获取最终结果。这是当前主要的待完善项。

### 3.5 交互流程说明

```
用户操作流程（四步）:

第一步：选择基站设备
  └── 选择产品类型 → 搜索/浏览设备 → 勾选目标设备（可多选）

第二步：选择命令
  └── 浏览命令树 → 点击命令节点（高亮选中）→ 命令详情加载到右栏

第三步：配置具体参数
  └── 操作面板输入 MML 文本 或 填写参数表单 → 点击"执行"按钮

第四步：查看结果
  └── 右栏上部结果区展示:
        - 执行时间戳
        - 设备 SN 列表
        - 每台设备的原始输出文本
        - 错误信息（如 "Error MML Script: GET BSC"）
```

**老系统结果展示格式示例**（来自截图 mml_06）：
```
2026-04-13 16:40:40 GET; GET BSC;
{F4F1F7EF0A0D5C4B2BB252A0661C,0D59FBEC36F218D69583979A3EA6,...}
Error MML Script:
GET BSC
```

---

## 4. MML 脚本管理

> 老系统截图：[/tmp/mml_05_command_input.png](/tmp/mml_05_command_input.png)（脚本树展示私有/公共模板）

### 4.1 功能需求清单

| 需求编号 | 功能描述 | 状态 | 优先级 |
|----------|----------|------|--------|
| MML-SCR-01 | 脚本列表查询（分页、按设备类型/创建者过滤）| ✅ 已实现（GET /mml/scripts）| P0 |
| MML-SCR-02 | 创建脚本（名称、描述、内容、设备类型、标签）| ✅ 已实现（POST /mml/scripts）| P0 |
| MML-SCR-03 | 更新脚本 | ✅ 已实现（PUT /mml/scripts/:id）| P0 |
| MML-SCR-04 | 删除脚本（支持批量删除）| ✅ 已实现（DELETE /mml/scripts/:id）| P0 |
| MML-SCR-05 | 脚本内容编辑器（代码高亮、语法提示）| ⚠️ 待完善（当前纯文本框）| P1 |
| MML-SCR-06 | 私有模板/公共模板分类（creator 维度区分）| ⚠️ 待完善（后端已有 creator 字段，前端未分类展示）| P1 |
| MML-SCR-07 | 脚本直接执行（选设备后一键执行脚本）| ✅ 已实现（executeScript API）| P0 |
| MML-SCR-08 | 脚本导入（上传 .txt 文件）| ❌ 待开发 | P1 |
| MML-SCR-09 | 脚本模板下载（导出模板文件）| ❌ 待开发 | P2 |
| MML-SCR-10 | 脚本语法校验（上传时检测错误行）| ❌ 待开发 | P2 |
| MML-SCR-11 | 标签管理（tags 字段，用于分类筛选）| ⚠️ 待完善（数据模型已有，UI 未实现）| P2 |
| MML-SCR-12 | 脚本搜索（关键词搜索脚本名/内容）| ✅ 已实现（search 参数）| P1 |

### 4.2 脚本数据模型（mml_scripts 表 DDL）

```sql
-- 来源：omcgo/migrations/000007_system_infra.sql
CREATE TABLE mml_scripts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    script_name VARCHAR(200) NOT NULL,    -- 脚本名称
    description TEXT,                     -- 脚本描述
    content     TEXT NOT NULL,            -- 脚本内容（MML 命令文本，分号分隔）
    device_type VARCHAR(50),              -- 适用设备类型（BSC/RTS/gNB 等）
    creator     VARCHAR(100),             -- 创建者用户名
    tags        JSONB DEFAULT '[]',       -- 标签数组
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 自动更新 updated_at 触发器
CREATE TRIGGER trigger_mml_scripts_updated_at
    BEFORE UPDATE ON mml_scripts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 索引（来源：000011_optimize_indexes.sql）
CREATE INDEX IF NOT EXISTS idx_mml_scripts_tags_gin
    ON mml_scripts USING GIN (tags);
```

**脚本内容格式示例**：

```
GET;
GET BSC;
LST BSCBASIC;
SET BSCBASIC: BSCID=1, BSCNAME="TEST_BSC";
```

### 4.3 API 接口设计

#### 4.3.1 获取脚本列表

```
GET /api/v1/mml/scripts
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码 |
| page_size | int | 否 | 每页数量 |
| device_type | string | 否 | 设备类型过滤 |
| creator | string | 否 | 创建者过滤（private: 当前用户; public: 空creator）|
| search | string | 否 | 关键词搜索 |

**响应体**：

```json
{
  "items": [
    {
      "id": "uuid",
      "script_name": "查询BSC基本信息",
      "description": "查询BSC基本配置参数",
      "content": "LST BSCBASIC;",
      "device_type": "BSC",
      "creator": "admin",
      "tags": ["query", "basic"],
      "created_at": "2026-04-13T00:00:00Z",
      "updated_at": "2026-04-13T00:00:00Z"
    }
  ],
  "total": 10,
  "page": 1,
  "page_size": 20,
  "total_pages": 1
}
```

#### 4.3.2 创建脚本

```
POST /api/v1/mml/scripts
```

**请求体**：

```json
{
  "script_name": "查询BSC基本信息",
  "description": "查询BSC基本配置参数",
  "content": "LST BSCBASIC;\nLST BTSBASIC;",
  "device_type": "BSC",
  "tags": ["query", "basic"]
}
```

#### 4.3.3 更新脚本

```
PUT /api/v1/mml/scripts/:id
```

请求体与创建接口相同，所有字段均必填（script_name + content）。

#### 4.3.4 删除脚本

```
DELETE /api/v1/mml/scripts/:id
```

返回 204 No Content。

> **待开发接口**（参考老系统）：
> - `POST /api/v1/mml/scripts/import` - 上传 .txt 文件导入脚本
> - `GET /api/v1/mml/scripts/template` - 下载脚本模板文件

### 4.4 脚本编辑器设计

| 组件 | 规格 |
|------|------|
| 编辑器类型 | 当前为 Textarea，建议升级为 Monaco Editor（轻量模式）|
| 语法高亮 | MML 关键字高亮（LST/SET/ADD/DEL/RST 等动词）|
| 行号显示 | 支持行号，便于错误定位 |
| 自动补全 | 根据设备类型提示可用命令（待开发）|
| 字符限制 | 建议不超过 100KB（约 5000 行命令）|

### 4.5 脚本格式规范

```
规范：
1. 每条命令以分号（;）结尾
2. 多条命令换行分隔
3. 注释以 # 开头（部分系统支持）
4. 参数格式：命令名 参数名=参数值, 参数名=参数值;
5. 字符串参数用双引号包裹

示例：
LST BSCBASIC;
SET BSCBASIC: BSCID=1, BSCNAME="SITE-001";
ADD BTS: BTSID=1, BTSNAME="BTS-001", LAC=10001;
RST DEV;
```

---

## 5. MML 任务管理

> 老系统截图：[/tmp/mml_03_mml_page.png](/tmp/mml_03_mml_page.png) 展示执行结果面板；[/tmp/mml_06_execution_result.png](/tmp/mml_06_execution_result.png) 展示结果输出格式。

### 5.1 功能需求清单

| 需求编号 | 功能描述 | 状态 | 优先级 |
|----------|----------|------|--------|
| MML-TASK-01 | 任务列表分页查询 | ✅ 已实现（GET /mml/tasks）| P0 |
| MML-TASK-02 | 按状态过滤任务（pending/running/completed/failed）| ✅ 已实现 | P0 |
| MML-TASK-03 | 查看单个任务详情（含结果）| ✅ 已实现（GET /mml/tasks/:id）| P0 |
| MML-TASK-04 | 任务结果明细（每台设备的执行结果）| ✅ 已实现（results JSONB）| P0 |
| MML-TASK-05 | 任务执行进度显示（成功N/总N）| ⚠️ 待完善（当前无进度字段）| P0 |
| MML-TASK-06 | 任务结果导出（CSV）| ❌ 待开发 | P1 |
| MML-TASK-07 | 手动启动挂起任务 | ❌ 待开发（后端缺少 start 接口）| P1 |
| MML-TASK-08 | 暂停执行中的任务 | ❌ 待开发 | P1 |
| MML-TASK-09 | 终止任务 | ❌ 待开发 | P1 |
| MML-TASK-10 | 删除任务（非执行中状态）| ❌ 待开发（后端缺少 delete 接口）| P1 |
| MML-TASK-11 | 创建定时任务（timing 执行方式）| ❌ 待开发 | P2 |
| MML-TASK-12 | 创建周期任务（period 执行方式）| ❌ 待开发 | P2 |
| MML-TASK-13 | 离线设备等待重试策略 | ❌ 待开发 | P2 |
| MML-TASK-14 | 在线设备失败重试策略 | ❌ 待开发 | P2 |
| MML-TASK-15 | 任务状态轮询（前端定时刷新）| ⚠️ 待完善（React Query 轮询已有框架，具体逻辑待实现）| P0 |

### 5.2 页面布局设计（上下分栏）

```
┌─────────────────────────────────────────────────────────┐
│  工具栏: [新建任务] [搜索框] [时间范围选择器]             │
├─────────────────────────────────────────────────────────┤
│                                                         │
│              MML 任务列表区（上半部分）                  │
│  列：任务名称 | 创建者 | 创建时间 | 类型 | 状态 | 进度 | 结果 │
│                                                         │
├─────────────────────────────────────────────────────────┤
│                                                         │
│         任务结果明细区（下半部分，点击任务后展开）        │
│  列：设备SN | 主机名 | MML命令 | 状态 | 结果 | 失败原因 │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

#### 5.2.1 任务列表区

| 列名 | 字段 | 宽度 | 说明 |
|------|------|------|------|
| 序号 | - | 60px | 自增序号 |
| 操作 | - | 80px | 结果查看图标 + 更多操作菜单 |
| 任务名称 | task_name | 自适应 | 支持模糊搜索 |
| 创建者 | creator | 120px | 创建任务的用户 |
| 创建时间 | created_at | 160px | 格式：YYYY-MM-DD HH:mm:ss |
| 类型 | execute_type | 100px | 立即/挂起/定时/周期（枚举 Tag）|
| 状态 | status | 100px | pending/running/completed/failed（带颜色 Badge）|
| 进度 | progress | 100px | 已完成设备数/总设备数（如 2/3）|
| 结果 | result | 100px | 成功/部分成功/失败 |
| 开始时间 | started_at | 160px | 任务开始执行时间 |
| 结束时间 | finished_at | 160px | 任务结束时间 |

**操作菜单项**：

| 操作 | 权限 | 可用状态 | 说明 |
|------|------|----------|------|
| 详情（信息） | 所有用户 | 所有状态 | 查看任务详情 |
| 开始 | CODE_ENB_MML | 挂起状态 | 启动挂起任务 |
| 暂停 | CODE_ENB_MML | 执行中 | 暂停任务 |
| 终止 | CODE_ENB_MML | 执行中/挂起 | 终止任务 |
| 删除 | CODE_ENB_MML | 非执行中 | 删除任务及结果 |

#### 5.2.2 任务结果明细区

| 列名 | 字段 | 宽度 | 说明 |
|------|------|------|------|
| 设备SN | device_sn | 180px | 设备序列号 |
| 主机名 | host_name | 自适应 | 设备主机名 |
| MML 命令 | mml_command | 自适应 | 执行的命令/脚本 |
| 状态 | progress_status | 100px | 等待/执行中/成功/失败/超时 |
| 结果 | progress_result | 100px | 成功/失败 |
| 失败原因 | failure_reason | 自适应 | 执行失败原因说明 |
| 详情 | detail | 200px | JSON 详情（超25字符截断，点击弹窗展示）|
| 开始时间 | run_time | 160px | 命令开始执行时间 |
| 结束时间 | end_time | 160px | 命令结束时间 |

**工具栏**：任务名称标题 + 导出 CSV 按钮 + 关闭按钮 + 搜索框（按设备内容过滤）。

### 5.3 新建任务对话框

#### 5.3.1 基本信息

| 字段 | 类型 | 必填 | 限制 | 说明 |
|------|------|------|------|------|
| 任务名称 | 文本输入 | ✓ | 最大50字符，不允许纯空格 | 任务唯一标识 |
| 选择脚本 | 脚本选择器或文件上传 | ✓ | .txt 格式 | 选择已有脚本或上传新脚本 |

#### 5.3.2 执行方式（立即/挂起/定时/周期）

| 方式 | 字段值 | 附加字段 | 说明 |
|------|--------|----------|------|
| 立即执行 | active | - | 创建后立即开始执行 |
| 挂起 | suspend | - | 创建后处于挂起状态，需手动启动 |
| 定时执行 | timing | time（DateTimePicker，禁选过去时间）| 按指定时间执行 |
| 周期任务 | period | periodStartTime + periodEndTime（DateRangePicker）+ periodTime（TimePicker）| 按周期重复执行 |

#### 5.3.3 执行策略（离线重试/在线重试）

**离线设备策略**：

| 字段 | 类型 | 默认值 | 限制 | 说明 |
|------|------|--------|------|------|
| 离线设备重试开关 | Switch | off | - | 离线设备是否等待上线重试 |
| 等待设备上线重试时间 | 数字输入（分钟）| 60 | 最小20，最大10080（7天）| 等待上线的超时时间 |

**在线设备策略**：

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| 配置失败重试开关 | Switch | off | 在线设备配置失败是否自动重试 |
| 重试次数 | 数字输入 | 3 | 最大重试次数 |
| 间隔重试时间 | 数字输入（分钟）| 5 | 每次重试间隔 |

### 5.4 数据模型（mml_tasks 表 DDL）

```sql
-- 来源：omcgo/migrations/000007_system_infra.sql
CREATE TABLE mml_tasks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_name   VARCHAR(200),                          -- 任务名称
    script_id   UUID REFERENCES mml_scripts(id)
                    ON DELETE SET NULL,                -- 关联脚本（可选）
    device_sns  JSONB NOT NULL,                        -- 目标设备SN列表
    commands    JSONB NOT NULL DEFAULT '[]',           -- 命令列表（含code/method/params）
    status      VARCHAR(20) NOT NULL DEFAULT 'pending', -- 任务状态
    results     JSONB DEFAULT '[]',                    -- 执行结果（每设备一条）
    creator     VARCHAR(100),                          -- 创建者
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 基础索引
CREATE INDEX idx_mml_tasks_status  ON mml_tasks(status);
CREATE INDEX idx_mml_tasks_created ON mml_tasks(created_at DESC);

-- GIN 索引（来源：000011_optimize_indexes.sql）
CREATE INDEX IF NOT EXISTS idx_mml_tasks_commands_gin
    ON mml_tasks USING GIN (commands jsonb_path_ops);
CREATE INDEX IF NOT EXISTS idx_mml_tasks_results_gin
    ON mml_tasks USING GIN (results jsonb_path_ops);
CREATE INDEX IF NOT EXISTS idx_mml_tasks_device_sns_gin
    ON mml_tasks USING GIN (device_sns);

-- 自动更新触发器
CREATE TRIGGER trigger_mml_tasks_updated_at
    BEFORE UPDATE ON mml_tasks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

**待完善字段**（建议通过迁移添加）：

```sql
-- 建议添加的字段（用于支持定时/周期/重试策略）
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS execute_type    VARCHAR(20) DEFAULT 'active';
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS scheduled_at    TIMESTAMPTZ;       -- 定时执行时间
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS period_start    TIMESTAMPTZ;       -- 周期开始时间
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS period_end      TIMESTAMPTZ;       -- 周期结束时间
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS period_time     TIME;              -- 周期执行时刻
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS offline_retry   BOOLEAN DEFAULT false;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS offline_wait    INT DEFAULT 60;    -- 分钟
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS failed_retry    BOOLEAN DEFAULT false;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS retry_count     INT DEFAULT 3;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS retry_interval  INT DEFAULT 5;     -- 分钟
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS started_at      TIMESTAMPTZ;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS finished_at     TIMESTAMPTZ;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS total_devices   INT DEFAULT 0;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS success_count   INT DEFAULT 0;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS failed_count    INT DEFAULT 0;
```

### 5.5 API 接口设计

#### 5.5.1 获取任务列表

```
GET /api/v1/mml/tasks
```

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码 |
| page_size | int | 每页数量 |
| status | string | 状态过滤：pending/running/completed/failed |
| search | string | 任务名称关键词 |
| start_time | string | 创建时间范围开始（ISO 8601）|
| end_time | string | 创建时间范围结束（ISO 8601）|

#### 5.5.2 获取任务详情

```
GET /api/v1/mml/tasks/:id
```

#### 5.5.3 任务控制接口（待开发）

```
POST /api/v1/mml/tasks/:id/start     -- 启动挂起任务
POST /api/v1/mml/tasks/:id/pause     -- 暂停任务
POST /api/v1/mml/tasks/:id/cancel    -- 终止任务
DELETE /api/v1/mml/tasks/:id         -- 删除任务
GET /api/v1/mml/tasks/:id/results    -- 获取任务结果明细（支持分页）
POST /api/v1/mml/tasks/:id/export    -- 导出结果 CSV
```

### 5.6 任务状态流转

```
                    ┌─────────────┐
                    │   pending   │ ← 创建任务时默认状态
                    └──────┬──────┘
                           │ ACS Worker 开始执行
                           ▼
                    ┌─────────────┐
                    │   running   │ ← 正在向设备发送命令
                    └──────┬──────┘
              ┌────────────┼────────────┐
              ▼            ▼            ▼
       ┌──────────┐  ┌──────────┐  ┌──────────┐
       │completed │  │  failed  │  │cancelled │
       └──────────┘  └──────────┘  └──────────┘

挂起流程（suspend）:
  pending → [用户手动 start] → running → completed/failed

定时流程（timing）:
  pending → [到达指定时间] → running → completed/failed

周期流程（period）:
  每次执行周期: pending → running → completed/failed → pending（下次周期）
```

---

## 6. 命令执行引擎

### 6.1 执行流程（用户输入 → RPC 调用 → 结果返回）

```
前端用户操作
    │
    ▼
POST /api/v1/mml/execute
    │ (创建 MML Task，status=pending)
    │
    ▼
ACS Worker 任务队列
    │ (轮询 pending 任务)
    │
    ▼
命令解析器
    │ (command_code → RPC Method + 参数构造)
    │
    ▼
TR-069 SOAP 消息构造
    │ (cwmp:GetParameterValues / SetParameterValues / Reboot ...)
    │
    ▼
目标设备（通过 ConnectionRequest 唤醒 or 等待设备 Inform）
    │
    ├── 设备在线: 发送 SOAP 请求 → 等待响应（超时: 60s）
    └── 设备离线: 标记结果 offline，根据重试策略决定等待重试
    │
    ▼
解析响应
    │ (XML 解析 → 结构化数据)
    │
    ▼
更新 mml_tasks.results[]
    │ (写入每台设备的执行结果)
    │
    ▼
更新 mml_tasks.status = completed/failed
    │
    ▼
前端轮询获取最终结果（GET /api/v1/mml/tasks/:id）
```

### 6.2 支持的 RPC 方法

| RPC 方法 | 说明 | 对应命令类型 | 当前支持状态 |
|----------|------|------------|------------|
| GetParameterValues | 查询设备参数值 | 查询类（query）| ✅ 支持 |
| SetParameterValues | 设置设备参数值 | 配置类（config）| ✅ 支持 |
| GetParameterNames | 查询参数树结构 | 查询类 | ⚠️ 部分支持 |
| GetParameterAttributes | 查询参数属性 | 查询类 | ⚠️ 部分支持 |
| SetParameterAttributes | 设置参数属性（通知级别）| 配置类 | ❌ 待开发 |
| AddObject | 添加对象实例 | 配置类 | ❌ 待开发 |
| DeleteObject | 删除对象实例 | 配置类 | ❌ 待开发 |
| Reboot | 设备重启 | 维护类（maintenance）| ✅ 支持 |
| FactoryReset | 恢复出厂设置 | 维护类 | ❌ 待开发（危险操作）|
| Download | 下载文件到设备（固件/配置）| 维护类 | ⚠️ 通过升级模块 |
| Upload | 从设备上传文件 | 维护类 | ❌ 待开发 |
| GetRPCMethods | 查询设备支持的 RPC 方法 | 查询类 | ✅ 支持 |

### 6.3 参数模板与验证规则

**param_template JSONB 完整结构定义**：

```typescript
// 来源：omcmb/webcode/src/types/mml.ts
interface MMLParam {
  name: string;           // 参数名
  type: 'string' | 'number' | 'boolean' | 'enum' | 'range' | 'ipAddress' | 'list';
  required: boolean;      // 是否必填
  defaultValue?: string | number | boolean;
  description: string;    // 参数说明
  options?: Array<{ label: string; value: string | number }>; // enum 类型选项
  minValue?: number;      // range 类型最小值
  maxValue?: number;      // range 类型最大值
  pattern?: string;       // string 类型正则校验
}
```

**前端参数验证规则**：

| 参数类型 | 验证逻辑 |
|----------|----------|
| string | required 时非空；pattern 存在时正则匹配 |
| number | required 时非空；min/max 范围校验 |
| boolean | 无特殊校验 |
| enum | 值必须在 options 列表中 |
| range | 等同 number，强制 min/max 校验 |
| ipAddress | IPv4/IPv6 格式校验 |
| list | 逗号分隔列表，每项校验 |

### 6.4 结果解析与展示

**results JSONB 结构（每台设备一条记录）**：

```json
[
  {
    "device_sn": "F4F1F7EF0A0D5C4B2E",
    "success": true,
    "raw_output": "BSC BSCID=1 BSCNAME=SITE-001 ...",
    "parsed_data": {
      "BSCID": "1",
      "BSCNAME": "SITE-001"
    },
    "execution_time": 1250,
    "timestamp": "2026-04-13T10:00:01Z",
    "failure_reason": ""
  },
  {
    "device_sn": "0D59FBEC36F218D695",
    "success": false,
    "raw_output": "Error MML Script: GET BSC",
    "parsed_data": null,
    "execution_time": 5000,
    "timestamp": "2026-04-13T10:00:06Z",
    "failure_reason": "MML command syntax error: unrecognized command GET BSC"
  }
]
```

**前端展示规则**：

| 字段 | 展示方式 |
|------|----------|
| success=true | 绿色 "成功" Badge |
| success=false | 红色 "失败" Badge + failure_reason 提示 |
| raw_output | 等宽字体文本区展示，超长截断（>25字符显示"..."）|
| parsed_data | 可点击展开 JSON 详情弹窗（600×500px）|
| execution_time | 以毫秒或秒为单位显示 |
| timestamp | 本地时区格式化 |

### 6.5 错误处理与超时机制

| 错误类型 | 处理策略 |
|----------|----------|
| 设备离线（ConnectionRequest 超时）| 根据 offline_retry 策略决定等待重试或标记失败 |
| RPC 超时（SOAP 响应超时）| 默认超时 60s，超时后标记设备结果为 timeout |
| SOAP Fault（设备返回错误）| 解析 Fault Code + FaultString，写入 failure_reason |
| 网络错误（连接被拒）| 标记 success=false，写入网络错误信息 |
| 参数验证失败（客户端）| 前端校验，不发起请求，提示具体字段错误 |
| 命令解析失败（MML 语法错误）| 上传时校验（待开发），执行时写入错误信息 |
| 批量部分失败 | 允许部分成功（支持部分失败处理），统计 success_count/failed_count |

### 6.6 批量执行策略

| 策略维度 | 说明 |
|----------|------|
| 并发数 | 默认并发 10 台设备，可配置上限（建议 ≤50）|
| 执行顺序 | 按 device_sns 数组顺序，或按设备在线状态优先 |
| 失败继续 | 单台设备失败不影响其他设备执行 |
| 进度更新 | 每台设备完成后实时更新 mml_tasks.results 和计数 |
| 超大批量 | 超过 100 台设备时建议分批（每批 50 台）|

---

## 7. 设备重启管理

> 老系统截图：
> - 重启任务列表：[/tmp/mml_07_reboot_page.png](/tmp/mml_07_reboot_page.png)
> - 周期重启配置：[/tmp/mml_08_periodic_reboot.png](/tmp/mml_08_periodic_reboot.png)

### 7.1 功能需求

| 需求编号 | 功能描述 | 状态 | 优先级 |
|----------|----------|------|--------|
| MML-RBT-01 | 重启任务列表（含历史记录）| ⚠️ 部分实现（通过 MML 任务实现，无独立页面）| P0 |
| MML-RBT-02 | 新建重启任务（选择设备 + 立即执行）| ✅ 可通过 POST /mml/execute + Reboot | P0 |
| MML-RBT-03 | 重启任务进度展示（N/N）| ⚠️ 待完善 | P0 |
| MML-RBT-04 | 重启任务结果（成功/失败/进行中）| ⚠️ 待完善 | P0 |
| MML-RBT-05 | 周期重启配置（任务列表视图）| ❌ 待开发 | P1 |
| MML-RBT-06 | 周期重启设备列表视图 | ❌ 待开发 | P1 |
| MML-RBT-07 | 全局禁用/启用周期重启开关 | ❌ 待开发 | P1 |
| MML-RBT-08 | 任务按产品型号分类展示 | ⚠️ 待完善 | P2 |

### 7.2 重启任务列表（表格设计）

**参考老系统**（截图 mml_07）：

| 列名 | 字段 | 宽度 | 说明 |
|------|------|------|------|
| 序号 | - | 60px | 自增序号 |
| 操作 | - | 80px | 更多操作菜单（三点图标）|
| 任务名称 | task_name | 自适应 | 格式示例：Reboot_admin_2025-12-24 02:11:18 |
| 操作人 | creator | 100px | 执行重启的用户 |
| 操作时间 | created_at | 160px | 任务创建时间 |
| 产品型号 | product_type | 100px | RTS/BM/BAIBLQ/QAFA 等 |
| 状态 | status | 100px | 已结束/进行中（带图标）|
| 进度 | progress | 80px | 格式：成功N/总N（如 1/1）|
| 结果 | result | 80px | 成功/失败 |

**老系统中可见的重启任务记录示例**（截图 mml_07）：
- test重启_admin_2025-12-24 02:11:18 | admin | 2025-12-24 10:11:31 | RTS | 已结束 1/1 | 失败
- Reboot_admin_2025-11-15 13:53:48 | admin | 2025-11-15 13:53:49 | BM | 已结束 1/1 | 成功

### 7.3 周期重启配置

**老系统截图**（mml_08）显示周期重启页面有两个视图：
- **任务列表视图**：展示所有周期重启任务
- **设备列表视图**：展示配置了周期重启的设备列表

**页面功能**：
- 搜索框（任务名称过滤）
- 禁用开关（全局禁用所有周期重启）
- 刷新按钮

**列定义**（任务视图）：

| 列名 | 说明 |
|------|------|
| 任务名称 | 周期重启任务名 |
| 操作人 | 创建者 |
| 操作时间 | 创建时间 |
| 状态 | 启用/禁用 |
| 进度 | 当前轮次执行进度 |
| 结果 | 上次执行结果 |
| 开始时间 | 周期开始时间 |

### 7.4 重启 API 接口

**当前实现**（通过 MML execute 接口）：

```
POST /api/v1/mml/execute
{
  "command_code": "RST_DEV",
  "device_sns": ["SN1", "SN2"],
  "task_name": "Reboot_admin_2026-04-13 10:00:00"
}
```

**建议新增独立接口**：

```
GET    /api/v1/mml/reboot/tasks          -- 重启任务列表（带产品型号过滤）
POST   /api/v1/mml/reboot/tasks          -- 创建重启任务
GET    /api/v1/mml/reboot/tasks/:id      -- 获取任务详情
DELETE /api/v1/mml/reboot/tasks/:id      -- 删除任务

GET    /api/v1/mml/reboot/periodic       -- 周期重启配置列表
POST   /api/v1/mml/reboot/periodic       -- 创建周期重启配置
PUT    /api/v1/mml/reboot/periodic/:id   -- 更新周期重启配置
DELETE /api/v1/mml/reboot/periodic/:id   -- 删除周期重启配置
POST   /api/v1/mml/reboot/periodic/:id/enable   -- 启用
POST   /api/v1/mml/reboot/periodic/:id/disable  -- 禁用
```

---

## 8. 前端技术方案

### 8.1 页面组件设计

```
src/pages/mml/
├── Console/
│   ├── index.tsx              -- MML 控制台主页（三栏布局）
│   ├── components/
│   │   ├── DeviceTree.tsx     -- 左栏：设备选择
│   │   ├── CommandTree.tsx    -- 中栏：命令树
│   │   ├── TerminalPanel.tsx  -- 右栏上：终端输出
│   │   ├── CommandInput.tsx   -- 右栏下：命令输入/参数表单
│   │   └── BatchSnModal.tsx   -- 批量 SN 输入弹窗
│   └── hooks/
│       ├── useDeviceSelection.ts  -- 设备选择状态
│       ├── useCommandSelection.ts -- 命令选择状态
│       └── useCommandExecution.ts -- 命令执行与结果
├── Scripts/
│   ├── index.tsx              -- 脚本管理列表页
│   ├── components/
│   │   ├── ScriptList.tsx     -- 脚本列表表格
│   │   ├── ScriptEditor.tsx   -- 脚本编辑器（Modal/Drawer）
│   │   └── ScriptTree.tsx     -- 私有/公共模板树形展示
│   └── hooks/
│       └── useScriptManager.ts
├── Tasks/
│   ├── index.tsx              -- 任务列表页（上下分栏）
│   ├── components/
│   │   ├── TaskList.tsx       -- 任务列表表格
│   │   ├── TaskResults.tsx    -- 任务结果明细
│   │   ├── CreateTaskModal.tsx -- 新建任务弹窗
│   │   └── TaskStatusBadge.tsx -- 状态 Badge 组件
│   └── hooks/
│       └── useTaskPolling.ts  -- 任务轮询逻辑
└── Reboot/
    ├── index.tsx              -- 重启管理页（重启 + 周期重启 Tab）
    └── components/
        ├── RebootList.tsx
        └── PeriodicReboot.tsx
```

### 8.2 状态管理方案（React Query + 轮询）

**核心 Query Keys**：

```typescript
export const mmlQueryKeys = {
  commands: (params?: CommandQueryParams) => ['mml', 'commands', params],
  command: (id: string) => ['mml', 'commands', id],
  scripts: (params?: ScriptQueryParams) => ['mml', 'scripts', params],
  script: (id: string) => ['mml', 'scripts', id],
  tasks: (params?: TaskQueryParams) => ['mml', 'tasks', params],
  task: (id: string) => ['mml', 'tasks', id],
};
```

**任务轮询策略**（running 状态时自动刷新）：

```typescript
const { data: task } = useQuery({
  queryKey: mmlQueryKeys.task(taskId),
  queryFn: () => mmlApi.getTaskById(taskId),
  refetchInterval: (data) => {
    // 任务 running 时每 3 秒轮询一次
    if (data?.status === 'running') return 3000;
    // 任务 pending 时每 5 秒轮询
    if (data?.status === 'pending') return 5000;
    // 终态时停止轮询
    return false;
  },
});
```

**Mutation 钩子**：

```typescript
// 执行命令
const executeMutation = useMutation({
  mutationFn: mmlApi.executeCommand,
  onSuccess: (task) => {
    // 将任务 ID 加入轮询队列
    startPolling(task.id);
    message.success('命令已提交执行');
  },
});
```

### 8.3 i18n 国际化

**命名空间**: `mml`

```typescript
// 建议添加到 src/i18n/zh-CN/index.ts
mml: {
  console: {
    title: 'MML 控制台',
    devicePanel: '基站设备',
    commandPanel: '命令列表',
    resultPanel: '结果',
    helpPanel: '帮助',
    batchInput: '批量输入',
    executeBtn: '执行',
    resetBtn: '重置',
    step1: '第一步 选择基站设备',
    step2: '第二步 选择命令',
    step3: '第三步 配置具体参数',
    step4: '第四步 查看结果',
    operationPanel: '操作面板',
    paramPathPanel: '参数路径指定',
  },
  scripts: {
    title: 'MML 脚本管理',
    privateTemplate: '私有模板',
    publicTemplate: '公共模板',
    createScript: '新建脚本',
    editScript: '编辑脚本',
    deleteConfirm: '确定删除所选脚本？',
  },
  tasks: {
    title: 'MML 任务管理',
    createTask: '新建任务',
    status: {
      pending: '等待中',
      running: '执行中',
      completed: '已完成',
      failed: '失败',
      cancelled: '已取消',
    },
    result: {
      success: '成功',
      partial: '部分成功',
      failure: '失败',
    },
    executeType: {
      active: '立即执行',
      suspend: '挂起',
      timing: '定时执行',
      period: '周期任务',
    },
  },
  reboot: {
    title: '重启管理',
    periodicReboot: '周期重启',
    disableAll: '禁用',
  },
},
```

### 8.4 权限控制（CODE_ENB_MML）

| 权限代码 | 控制范围 |
|----------|----------|
| CODE_ENB_MML | MML 模块写权限（执行命令、创建/修改/删除脚本、控制任务）|
| CODE_ENB_MML_READ | MML 模块只读权限（查看命令/脚本/任务，不可执行）|

**前端权限判断示例**：

```typescript
const { hasPermission } = usePermission();
const canExecute = hasPermission('CODE_ENB_MML');

// 执行按钮
<Button
  type="primary"
  disabled={!canExecute || selectedDevices.length === 0}
  onClick={handleExecute}
>
  执行
</Button>
```

---

## 9. 权限与安全

### 9.1 功能权限控制

| API 路径 | 方法 | 所需权限 | 说明 |
|----------|------|----------|------|
| /api/v1/mml/commands | GET | 已登录 | 查看命令列表 |
| /api/v1/mml/commands/:id | GET | 已登录 | 查看命令详情 |
| /api/v1/mml/execute | POST | CODE_ENB_MML | **执行命令（高权限）** |
| /api/v1/mml/scripts | GET | 已登录 | 查看脚本列表 |
| /api/v1/mml/scripts | POST | CODE_ENB_MML | 创建脚本 |
| /api/v1/mml/scripts/:id | PUT | CODE_ENB_MML | 更新脚本（仅本人或管理员）|
| /api/v1/mml/scripts/:id | DELETE | CODE_ENB_MML | 删除脚本 |
| /api/v1/mml/tasks | GET | 已登录 | 查看任务列表 |
| /api/v1/mml/tasks/:id | GET | 已登录 | 查看任务详情 |
| /api/v1/mml/tasks/:id/cancel | POST | CODE_ENB_MML | 终止任务 |

### 9.2 操作日志审计

所有写操作（命令执行、脚本创建/修改/删除、任务控制）应记录审计日志：

```sql
-- 写入 audit_logs 表（已有字段）
INSERT INTO audit_logs (
    user_id, action, resource_type, resource_id,
    details, ip_address, created_at
) VALUES (
    $1,      -- 操作用户
    'execute_mml_command',  -- 操作类型
    'mml_task',             -- 资源类型
    $task_id,               -- 资源 ID
    $details_json,          -- 操作详情（含命令内容、目标设备）
    $ip,
    NOW()
);
```

### 9.3 危险操作确认（修改/删除/重启）

| 危险操作 | 确认方式 | 说明 |
|----------|----------|------|
| 执行 RST_DEV（重启）| Modal 二次确认，显示将重启的设备列表 | 高危操作 |
| 执行 FactoryReset | Modal 二次确认 + 要求输入确认文本 | 极高危操作 |
| SetParameterValues 批量 | 提示影响范围（N 台设备将被修改）| 中危操作 |
| 删除脚本 | Modal 二次确认 | 低危操作 |
| 终止/删除任务 | Popconfirm 组件确认 | 低危操作 |

---

## 10. 非功能需求

### 10.1 性能指标（响应时间、并发支持）

| 指标 | 目标值 | 说明 |
|------|--------|------|
| 命令列表查询 | < 200ms | P99 响应时间 |
| 脚本列表查询 | < 200ms | P99 响应时间 |
| 任务创建 | < 500ms | 创建任务并入队 |
| 单台设备命令执行 | < 5s | 在线设备 GetParameterValues |
| 单台设备重启 | < 30s | Reboot RPC 完成 |
| 批量执行（100台）| < 10min | 受限于设备响应速度 |
| 并发执行任务数 | 10个任务并发 | ACS Worker 队列调度 |
| 前端轮询间隔 | 3~5s | running 状态任务 |

### 10.2 可靠性（超时重试、断线重连）

| 场景 | 处理策略 |
|------|----------|
| 设备 RPC 超时 | 默认 60s 超时，可配置（10~300s）|
| 设备离线 | 根据 offline_retry 配置等待重试（默认不重试）|
| 配置失败 | 根据 failed_retry 配置自动重试（默认不重试）|
| ACS Worker 崩溃 | 重启后恢复 running 状态的任务 |
| 数据库写入失败 | 重试 3 次，失败后记录错误日志 |
| 网络抖动 | 任务结果缓冲写入，批量提交（每 10 条或 5s 一次）|

### 10.3 可扩展性（新设备类型、新命令）

| 扩展维度 | 当前设计 | 扩展方式 |
|----------|----------|----------|
| 新设备产品类型 | product_types JSONB 数组 | 在 mml_commands 中添加产品类型，前端下拉枚举更新 |
| 新 MML 命令 | INSERT INTO mml_commands | 无需代码变更，纯数据驱动 |
| 新 RPC 方法 | ACS Worker 扩展 RPC 处理器 | 新增 RPC Handler，注册到路由 |
| 新参数类型 | MMLParamType 枚举扩展 | 更新 types/mml.ts + 对应 UI 组件 |
| 多租户支持 | 当前无租户隔离 | 建议在 mml_scripts/mml_tasks 中添加 carrier_id 字段 |

---

## 11. 待完善事项与实施建议

### 11.1 当前缺口与优先级

**高优先级（P0 - 阻塞核心流程）**：

| 缺口描述 | 影响范围 | 解决方案 |
|----------|----------|----------|
| 执行结果不实时（任务异步，结果轮询未实现）| 控制台用户体验 | 前端 React Query refetchInterval 轮询 + 任务状态 Badge |
| ACS Worker 与 mml_tasks 集成缺失（任务永远是 pending）| 全部功能均无法真实执行 | 实现 ACS Worker 任务消费循环 |
| mml_commands 命令目录不完整（仅3条预置命令）| 命令树为空 | 根据老系统截图扩充完整命令目录 |
| 任务进度字段缺失（无 total_devices/success_count）| 任务列表展示 | 数据库迁移添加进度字段 |

**中优先级（P1 - 影响主要用例）**：

| 缺口描述 | 影响范围 | 解决方案 |
|----------|----------|----------|
| 缺少任务控制接口（start/pause/cancel/delete）| 任务管理页 | 新增 4 个 API 接口 |
| 脚本管理无独立页面（前端 Scripts/ 目录结构存在但组件待完善）| 脚本管理用例 | 实现 ScriptList + ScriptEditor 组件 |
| 重启管理无独立页面（复用 MML 任务展示不直观）| 运维操作效率 | 新增 Reboot 页面，按产品型号分类 |
| 脚本文件导入功能缺失 | 老系统迁移 | 后端添加 multipart 文件上传接口 |
| 私有/公共模板树形分类展示未实现 | 命令控制台体验 | 前端按 creator 字段区分展示 |

**低优先级（P2 - 锦上添花）**：

| 缺口描述 | 影响范围 | 解决方案 |
|----------|----------|----------|
| 结果导出 CSV 功能 | 运维数据分析 | 后端流式导出 + 前端触发下载 |
| 脚本代码高亮编辑器 | 脚本编辑体验 | 引入 Monaco Editor（轻量模式）|
| 周期重启配置 | 定时维护场景 | 新增 mml_periodic_reboot 表 + CRON 调度 |
| 帮助文档 API | 命令使用引导 | 扩展 mml_commands 表加入详细参数说明 |
| 命令自动补全 | 控制台输入体验 | 基于 mml_commands 数据实现 CodeMirror/Monaco 补全 |

### 11.2 分阶段实施计划

**Phase 1（1~2 周）：核心功能闭环**

1. 实现 ACS Worker 消费 mml_tasks 队列，完成真实命令执行
2. 前端控制台添加任务状态轮询（useTaskPolling hook）
3. 扩充 mml_commands 种子数据（至少 20 条常用命令，覆盖 BSC/RTS/gNB）
4. 数据库迁移：添加任务进度字段（execute_type/started_at/finished_at/total_devices 等）

**Phase 2（2~3 周）：任务管理完善**

1. 实现任务控制接口（start/pause/cancel/delete）
2. 实现独立任务管理页面（上下分栏，结果明细展示）
3. 实现脚本管理独立页面（列表 + 编辑器 Modal）
4. 实现私有/公共模板分类展示

**Phase 3（3~4 周）：重启管理与辅助功能**

1. 实现独立重启管理页面
2. 实现脚本文件导入功能
3. 实现结果导出 CSV
4. 实现危险操作二次确认弹窗

**Phase 4（按需）：高级功能**

1. 周期重启配置
2. 脚本代码高亮编辑器（Monaco Editor）
3. 命令帮助文档
4. 多租户隔离（carrier_id）

### 11.3 测试验证建议

| 测试类型 | 测试重点 |
|----------|----------|
| 单元测试 | mml/service.go 各方法；命令参数模板解析；任务状态机流转 |
| 集成测试 | POST /mml/execute 全链路（创建任务→Worker执行→结果写回）|
| 前端组件测试 | DeviceTree 设备选择；CommandInput 参数表单验证 |
| E2E 测试 | 控制台执行 GetParameterValues 命令全流程 |
| 性能测试 | 100台设备批量重启场景；任务列表分页查询（10万条数据）|
| 错误处理测试 | 设备离线场景；RPC 超时场景；MML 语法错误场景 |
| 权限测试 | CODE_ENB_MML 权限控制；只读用户无法执行命令 |

---

*文档结束*
