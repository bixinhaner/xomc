# MML 管理

## 概述
MML (Man-Machine Language) 管理模块提供对基站设备的命令行操作界面，支持执行 MML 命令和批量 MML 脚本任务。
- 涉及系统：仅 OMC

---

## 一、Cloudcore OMC — MML 命令

### 来源
- 系统：Cloudcore OMC
- 截图文件：OMC/1MML-1 4 (2).png

### 页面布局
- 左右分栏:
  - 左侧 (约40%): 设备选择 + MML 命令列表
  - 右侧 (约60%): 命令详情 + 参数配置 + 执行面板

### 导航元素
- **功能导航栏 Tab**: `MML`（橙色选中）| `Configuration` | `Change Password` | `Reboot` | `Logs` | `Signaling Trace` | `Backup&Restore`
- **子Tab**: `MML`（选中）| `MML Script`

### 左侧面板 — 设备选择区
- **标题**: `eNB`
- **批量输入按钮**: `Batch Input`（橙色边框按钮）
- **搜索框**: `Serial Number/ Cell Name` + 搜索图标
- **筛选器**: `Product Type ∨ ×`（下拉带清除）
- **设备表格**:
  | 列名 | 说明 |
  |------|------|
  | Serial Number | 设备序列号 |
  | Cell Name | 小区名称 |
- 表格显示 "No data" 无数据状态
- **分页器**: `50/page ∨` | `< 1 2 ... 6 >` | `go to [ ] 1 ↻` | `Total 20`

### 左侧面板 — MML List 命令列表
- **标题**: `MML List`
- **搜索框**: `code/Name` + 搜索图标
- **树形分类列表**:
  - `> eNB Configuration`（可展开）
  - `> Radio Configuration`（可展开）
  - `∨ Customized`（已展开）
    - `> admin`（用户自定义分组）
    - `∨ ju_dust`（已展开）
      - `Device.Service(LST NTP_SYNC))`
      - `Device.Service(MOD NTP_SYNC)`（橙色高亮选中，带 × 删除按钮）
  - 每个分组旁有 `⊕` 添加按钮

### 右侧面板 — 命令详情区
- **命令标题**: `NTP SYNC(MOD NTP_SYNC)`
- **操作按钮**: `Result`（默认按钮）| `Help`（默认按钮）| 外链图标
- **命令信息**:
  | 字段 | 值 |
  |------|-----|
  | 所属产品类型 | RTD |
  | 命令功能 | 该命令用于查询异频切换参数组 |
  | 注意事项 | 无 |

- **参数说明区域**:
  - 搜索框: `参数名称` + 搜索图标
  - 参数卡片（如 NTP interval）:
    | 字段 | 值 |
    |------|-----|
    | 数据类型 | unsignedInt |
    | 取值范围 | [60:65535] |
    | 默认值 | 60 |
    | 建议值 | 60 |
    | 单位 | s |
    | 是否重启生效 | 是 |
    | 说明 | 1202000089177LP0008(null) 0235a1e63b494e46ae0ee984c252f4c6:1/1 |

### 右侧面板 — 执行区
- **Tab 切换**: `Control Panel` | `Parameter Command`（橙色选中）
- **Operation Type**: `LST` 下拉选择
- **执行按钮**: `DO`（橙色按钮）| `Save`（橙色按钮）
- **Parameter Path 列表**（多行）:
  - 每行包含:
    - 标签: `Parameter Path`
    - 输入框: 如 `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.PowerModify`
    - 操作按钮: `+`（添加）或 `—`（删除）
  - 支持多个 Parameter Path

---

## 二、Cloudcore OMC — MML Script 批量脚本

### 来源
- 系统：Cloudcore OMC
- 截图文件：OMC/MMLscript1.png, OMC/MMLscript2 (1).png

### 页面布局
- 上下分区:
  - 上方: 任务列表
  - 下方: 执行结果详情

### 导航元素
- **子Tab**: `MML` | `MML Script`（橙色选中）

### MML Script 任务列表（上方）

#### 筛选条件
| 筛选项 | 类型 | 说明 |
|--------|------|------|
| Task Name | 搜索框 + 搜索图标 | 任务名称搜索 |
| Start Time — End Time | 时间范围选择器 | 执行时间范围 |

#### 新建按钮
- 右上角 `+` 橙色圆形按钮

#### 数据表格列
| 列名 | 说明 |
|------|------|
| (无标题) | 文件图标 / 终止图标 |
| Task Name | 任务名称（如 MML Task_admin_2022-11-15 19:29:51） |
| Creator | 创建者（如 admin） |
| Create Time | 创建时间 |
| Type | 执行类型（Immediately） |
| Status | 状态: `Waiting`(蓝色) / `In progress`(蓝色) / `End` / `Terminate`(橙色) |
| Progress | 进度（如 0/7, 1/15, 5/5） |
| Results | 结果: `Success`(绿色) / `Partial success` / `Fail`(红色) |
| Start Time | 开始时间 |
| End time | 结束时间 |

#### 行内操作（右键菜单）
- `Start`（绿色，启动）
- `Terminate`（终止）
- `Delete`（删除）

### 执行结果详情（下方）
- **标题**: `Results (MML Task_admin_2022-11-15 19:29:51)` — 任务名链接
- **搜索**: `Serial Number` + 搜索图标
- **导出/关闭按钮**: 导出图标 + × 关闭

#### 结果表格列
| 列名 | 说明 |
|------|------|
| Serial Number | 设备序列号 |
| Cell Name | 小区名称 |
| MML Script | MML 脚本内容（如 MOD MME:LTE_SIGLINK_SERVER_LIST=...） |
| Status | 状态（End） |
| Results | 结果（Success） |
| Failure Reason | 失败原因 |
| Detail | 详情 |
| Start Time | 开始时间 |
| End Time | 结束时间 |

### 分页器
- 上下两个表格各有独立分页器

---

## 三、Cloudcore OMC — 新建 MML Script 任务

### 来源
- 系统：Cloudcore OMC
- 截图文件：OMC/MMLscript2 (1).png

### 弹窗/表单: New Task
- **关闭按钮**: 右上角 ×（橙色）

#### Basic info 基本信息区
| 字段 | 类型 | 示例值 | 说明 |
|------|------|--------|------|
| Task Name | 输入框 | Upgrade_admin_2022-11-07 09:10:53 | 任务名称，自动生成 |
| Select MML Script (左) | 文件选择 + 查看按钮 | MML result 20251111140702.txt | 选择 MML 脚本文件 |
| Select MML Script (右) | 文件选择 + 查看按钮 | MML result 20251111140702.txt | 第二个脚本文件 |
| | 链接 | `Reading File...` / `✓ Finish` | 文件读取状态 |
| | 链接 | `Export Template` | 导出模板下载 |
| | 链接 | `The file contains no errors.` / `The file contains 10 errors.` | 文件校验结果（红色） |
- 提示: `Only .txt is supported.`

#### Execute Mode 执行模式区
| 选项 | 类型 | 说明 |
|------|------|------|
| Immediately | 单选按钮（橙色选中） | 立即执行 |
| Awaiting Start | 单选按钮 | 等待启动 |
| Schedule Time | 单选按钮 | 定时执行，显示时间选择器 |
| Recurring | 单选按钮 | 周期执行，显示时间+间隔选择器 |

#### Execution Policy 执行策略区
| 字段 | 类型 | 说明 |
|------|------|------|
| Offline Devices (策略1) | 复选框 + 输入框 | Wait for the device to go online and try again. The maximum wait time is [60] minutes. |
| Offline Devices (策略2) | 复选框 + 输入框 | If the configuration fails, retry for a maximum of [3] times with an interval of [5] minutes. |

#### 操作按钮
- `OK`（橙色按钮）| `Cancel`（灰色按钮）
