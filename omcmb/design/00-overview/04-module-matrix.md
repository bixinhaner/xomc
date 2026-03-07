# 模块功能矩阵 (Module Matrix)

## 17 模块总览

| 序号 | 模块 | OMC | jx | 广研院 | 子页面数 | 主布局模板 | 核心组件 |
|------|------|:---:|:---:|:---:|---------|-----------|---------|
| 01 | 全局框架与导航 | ✓ | ✓ | ✓ | 6 | app-shell | Header, Sidebar, Tabs |
| 02 | Dashboard 首页 | ✓ | ✓ | ✓ | 7 | dashboard-page | KPI Card, Charts |
| 03 | 设备管理 | ✓ | ✓ | ✓ | 12 | tree-list / list | DataTable, DeviceSelector |
| 04 | 告警管理 | ✓ | ✓ | ✓ | 9 | list / dashboard | DataTable, AlarmFilter |
| 05 | MML 管理 | ✓ | — | — | 5 | command-console | Tree, CodeEditor |
| 06 | 备份恢复 | ✓ | — | — | 5 | split-panel / form | SchedulePicker, FTP Config |
| 07 | License 管理 | ✓ | — | — | 3 | split-panel | DataTable, BatchInput |
| 08 | 基站监控与拓扑 | ✓ | — | ✓ | 6 | map-page | GIS Map, TopologyCanvas |
| 09 | 配置管理 | — | ✓ | ✓ | 10 | tree-list / form | Tree, TransferBox, Form |
| 10 | 性能管理 | — | ✓ | ✓ | 7 | tree-list / wizard | Charts, StepWizard |
| 11 | MR 管理 | — | ✓ | ✓ | 6 | tree-list / list | DataTable, Charts |
| 12 | 报表管理 | — | ✓ | — | 4 | tree-list | DataTable, CombinedChart |
| 13 | 系统管理 | — | ✓ | — | 8 | list / dashboard | DataTable, Toggle |
| 14 | 软件版本管理 | — | ✓ | ✓ | 4 | list / form | FileUpload, Plan Form |
| 15 | 文件管理 | — | ✓ | ✓ | 7 | tree-list / split | FileUpload, DataTable |
| 16 | 日志管理 | — | — | ✓ | 6 | list | DataTable, Charts, JSON Viewer |
| 17 | 运维工具 | — | — | ✓ | 5 | list | FileUpload, DataTable |

## 布局模板使用统计

| 布局模板 | 使用次数 | 使用模块 |
|---------|---------|---------|
| list-page（列表页） | ~40 | 几乎所有模块 |
| tree-list-page（左树右列表） | ~20 | 设备、配置、性能、MR、报表、文件 |
| split-panel-page（上下分屏） | ~8 | MML、备份、License |
| form-page（表单页） | ~10 | 备份创建、升级计划、任务创建 |
| dashboard-page（仪表盘） | 3 | Dashboard、告警统计、系统首页 |
| map-page（地图页） | 2 | GIS 地图、拓扑画布 |
| wizard-page（向导页） | 1 | 性能指标提取 |
| command-console-page（控制台） | 1 | MML 命令 |

## 核心组件使用频率

| 组件 | 使用频率 | 关键模块 |
|------|---------|---------|
| DataTable（数据表格） | 极高 | 所有模块 |
| FilterBar（筛选栏） | 极高 | 所有列表页 |
| Modal（弹窗） | 高 | 所有有 CRUD 的模块 |
| Tree（树） | 高 | 设备、配置、性能、文件、报表 |
| Pagination（分页） | 极高 | 所有列表页 |
| StatusIndicator（状态指示） | 高 | 设备、告警、任务 |
| DeviceSelector（设备选择器） | 高 | 配置、性能、软件、文件、MR |
| Charts（图表） | 中 | Dashboard、性能、告警统计、日志统计 |
| FileUpload（文件上传） | 中 | 设备注册、软件版本、运维模板 |
| TaskPanel（任务面板） | 全局 | 所有模块共享 |

## 模块间导航关系

```
Dashboard ──→ 告警管理（点击告警卡片）
Dashboard ──→ 设备管理（点击设备统计）
Dashboard ──→ 基站监控（点击地图小部件）

设备管理 ←→ 告警管理（设备详情中查看告警）
设备管理 ←→ 性能管理（设备详情中查看性能）
设备管理 ──→ 配置管理（设备配置操作）
设备管理 ──→ 软件版本（设备升级操作）

告警管理 ──→ 设备管理（从告警定位设备）

配置管理 ←→ 性能管理（参数调整与性能对比）
配置管理 ──→ 运维工具（使用配置模板）

软件版本 ──→ 文件管理（固件文件）
```
