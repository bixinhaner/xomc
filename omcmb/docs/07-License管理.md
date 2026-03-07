# License 管理

## 概述
License 管理模块用于管理基站设备的授权许可证，支持 License 的发送、下载、删除和批量输入操作。
- 涉及系统：仅 OMC

---

## 一、Cloudcore OMC — License 管理主页面

### 来源
- 系统：Cloudcore OMC
- 截图文件：OMC/4GLicense.png, OMC/4GLicense-1.png

### 页面布局
- 上下分区:
  - 上方: License 设备列表
  - 下方: License 操作日志（Logs）

### 导航元素
- **功能Tab**: `Device` | `Date Model` | `Certificate` | `License`（橙色选中）
- **License 类型 Tab**: `Basic License`（橙色选中）| `1688License`

### 工具栏操作
| 按钮 | 图标 | 说明 |
|------|------|------|
| `☑(2)` | 复选框 | 已选设备计数 |
| Send License | 发送图标 | 发送 License 到设备 |
| Download | 下载图标 | 下载 License |
| Delete | 删除图标 | 删除 License |

### 右上角图标按钮
- 批量选择图标、导出日志图标、导出列表图标

### 筛选/搜索条件
| 筛选项 | 类型 | 说明 |
|--------|------|------|
| Serial Number | 搜索框 + 搜索图标 | 设备序列号搜索 |
| Online | 下拉标签 `∨ ×` | 在线状态筛选 |
| Status | 下拉标签 `∨ ×` | License 状态筛选 |
| Product Type | 下拉标签 `∨ ×` | 产品类型筛选 |

### 数据表格列（License 列表）
| 列名 | 说明 |
|------|------|
| (复选框) | 多选 |
| (状态图标) | 设备在线状态圆点（绿色/橙色/红色） |
| (更多菜单) | `⋮` 三点操作菜单 |
| Serial Number ↕ | 设备序列号，可排序 |
| Product Type | 产品类型（QRTB/RTD/RTS/QAFA/QAFB） |
| License File | License 文件名（如 120200054822CHB0004.lic） |
| Upload Time ↕ | 上传时间 |
| Status | 状态: `Execute failure`（红色失败图标） |

### 分页器
- `50/page ∨` | `< 1 2 ... 6 >` | `go to [ ] 1 ↻` | `Total 20`

### 下方 — Logs 操作日志
- **标题**: `Logs`
- **搜索**: `Serial Number` + 搜索图标
- **右上角图标**: 导出图标、外链图标

#### Logs 表格列
| 列名 | 说明 |
|------|------|
| Serial Number | 设备序列号 |
| Start Time | 开始时间 |
| Task Progress | 任务进度 |
| Results | 结果: `Success`(绿色) |

### 分页器（Logs）
- `50/page ∨` | `< 1 2 ... 6 >` | `go to [ ] 1 ↻` | `Total 20`

---

## 二、Batch Input 批量输入弹窗

### 来源
- 截图文件：OMC/4GLicense-1.png

### 弹窗: Batch Input
- **标题**: `Batch Input`
- **关闭按钮**: 右上角 ×
- **Serial Number**: 多行文本输入框（大文本区域）
- **提示文字**: `When registering multiple devices, use a semi-colon(;) or space after each one or hit Enter to put each device on a separate line.`
- **操作按钮**: `OK`（橙色按钮）| `Cancel`（灰色按钮）

---

## 三、UI 元素汇总

| 组件 | 说明 |
|------|------|
| License 类型切换 | Basic License / 1688License Tab 切换 |
| 设备表格 | 支持序列号搜索 + 多条件筛选 + 排序 |
| 批量操作 | Send License / Download / Delete + 批量输入弹窗 |
| 操作日志 | 独立的 Logs 区域，记录操作历史和结果 |
| 状态标识 | Execute failure 红色图标 / Success 绿色文本 |
