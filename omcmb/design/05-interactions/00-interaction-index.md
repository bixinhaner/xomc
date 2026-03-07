# 交互模式索引 (Interaction Patterns Index)

> OMC 统一网管系统 - 交互设计规范

## 概述

本章节定义了 OMC 统一网管系统中所有通用交互模式。这些模式为前端开发提供一致的用户体验参考，确保系统各模块的交互行为统一、可预测。

---

## 文件索引

| 编号 | 文件名 | 交互模式 | 简要说明 |
|------|--------|---------|---------|
| 01 | `01-loading-states.md` | 加载状态 (Loading States) | Spinner、Skeleton、Progress Bar、全页加载、局部加载等模式 |
| 02 | `02-error-states.md` | 错误状态 (Error States) | 网络错误、服务器错误、404、403、表单校验、API 错误、连接断开等处理 |
| 03 | `03-batch-operations.md` | 批量操作 (Batch Operations) | Checkbox 选择、批量操作工具栏、确认对话框、进度跟踪、结果通知 |
| 04 | `04-search-filter-patterns.md` | 搜索与筛选 (Search & Filter) | 即时搜索、下拉筛选、日期范围、可展开筛选栏、自定义筛选、URL 同步 |
| 05 | `05-table-interactions.md` | 表格交互 (Table Interactions) | 排序、筛选、列宽调整、列排序、列显隐、行选择、行展开、虚拟滚动 |
| 06 | `06-tree-interactions.md` | 树形交互 (Tree Interactions) | 展开/折叠、节点选择、多选级联、搜索、懒加载、拖拽排序、右键菜单 |
| 07 | `07-form-interactions.md` | 表单交互 (Form Interactions) | 必填标记、行内校验、动态字段、联动下拉、自动保存、未保存提醒 |
| 08 | `08-modal-flows.md` | 弹窗流程 (Modal Flows) | 打开/关闭动画、嵌套弹窗、全屏模式、表单提交、Drawer 抽屉变体 |
| 09 | `09-real-time-updates.md` | 实时更新 (Real-time Updates) | WebSocket 告警/任务/Dashboard、Polling 降级、实时指示器、连接状态 |
| 10 | `10-export-download.md` | 导出与下载 (Export & Download) | 格式选择、列选择、进度指示、下载通知、任务跟踪、批量下载 |
| 11 | `11-drag-drop.md` | 拖拽交互 (Drag & Drop) | 文件上传拖放区、树节点排序、Dashboard 组件排序、拓扑画布定位 |
| 12 | `12-keyboard-shortcuts.md` | 键盘快捷键 (Keyboard Shortcuts) | 全局快捷键、表格快捷键、表单快捷键、导航快捷键、焦点指示器 |
| 13 | `13-context-menu-patterns.md` | 右键菜单 (Context Menu) | 表格行右键、树节点右键、拓扑节点右键、菜单结构与分组 |
| 14 | `14-task-tracking.md` | 任务跟踪 (Task Tracking) | 任务创建、状态流转、自动刷新、完成/失败通知、任务导航 |

---

## 交互模式分类

### 数据展示类
- 加载状态 (01)
- 错误状态 (02)
- 实时更新 (09)

### 数据操作类
- 批量操作 (03)
- 搜索与筛选 (04)
- 表格交互 (05)
- 树形交互 (06)
- 导出与下载 (10)

### 表单与弹窗类
- 表单交互 (07)
- 弹窗流程 (08)

### 高级交互类
- 拖拽交互 (11)
- 键盘快捷键 (12)
- 右键菜单 (13)

### 后台任务类
- 任务跟踪 (14)

---

## 优先级说明

| 优先级 | 模式 | 说明 |
|--------|------|------|
| P0 - 必须 | 加载状态、错误状态、表格交互、表单交互、弹窗流程 | 基础功能，首期必须实现 |
| P1 - 重要 | 搜索筛选、批量操作、任务跟踪、实时更新、导出下载 | 核心业务功能依赖 |
| P2 - 增强 | 树形交互、右键菜单、拖拽交互、键盘快捷键 | 提升用户效率的增强功能 |

---

## 与其他章节的关系

- **01-design-system**: 交互模式使用设计系统中定义的颜色、字体、间距 Token
- **02-components**: 组件是交互模式的载体，本章节描述组件的行为规范
- **03-layouts**: 布局定义了交互发生的空间结构
- **04-modules**: 各业务模块引用本章节的交互模式
- **06-user-flows**: 用户流程由多个交互模式组合而成
