# 色彩体系 (Color System)

## 主色 (Primary - Blue)

| Token | Hex | 用途 |
|-------|-----|------|
| `primary-50` | `#EBF5FF` | 选中行背景、信息提示背景 |
| `primary-100` | `#D6EBFF` | 浅色强调背景 |
| `primary-200` | `#ADD6FF` | 浅色悬浮态 |
| `primary-300` | `#85C1FF` | 禁用态主色元素 |
| `primary-400` | `#5CACFF` | 链接悬浮色 |
| `primary-500` | `#3396FF` | 活动链接、次要强调 |
| `primary-600` | `#1677FF` | **主操作色** — 按钮、活跃导航、交互元素 |
| `primary-700` | `#0958D9` | 主按钮悬浮态 |
| `primary-800` | `#003EB3` | 主按钮按下态 |
| `primary-900` | `#002C8C` | 深色强调（极少使用） |

### 使用规则
- 主操作按钮使用 `primary-600`
- 悬浮态使用 `primary-700`，按下态使用 `primary-800`
- 链接文字使用 `primary-600`，悬浮态 `primary-400`
- 选中行/项目背景使用 `primary-50`

## 中性色 (Neutral - Gray)

| Token | Hex | 用途 |
|-------|-----|------|
| `neutral-50` | `#FAFAFA` | 页面背景色 |
| `neutral-100` | `#F5F5F5` | 卡片背景、表格斑马纹 |
| `neutral-200` | `#F0F0F0` | 分割线、细边框 |
| `neutral-300` | `#D9D9D9` | 禁用态边框、输入框占位符 |
| `neutral-400` | `#BFBFBF` | 禁用态文字、非活跃图标 |
| `neutral-500` | `#8C8C8C` | 次要文字、时间戳、说明文字 |
| `neutral-600` | `#595959` | **默认正文色** |
| `neutral-700` | `#434343` | 强调正文、副标题 |
| `neutral-800` | `#262626` | 主标题、表头 |
| `neutral-900` | `#1F1F1F` | 侧边栏深色背景 |
| `neutral-950` | `#141414` | 侧边栏最深区域 |

## 告警等级色 (Alarm Severity)

遵循中国电信行业 YD/T 标准。显示顺序：从高到低（紧急→主要→次要→警告）。

### 紧急 (Critical)

| 用途 | Token | Hex |
|------|-------|-----|
| 主色 | `severity-critical` | `#F5222D` |
| 背景 | `severity-critical-bg` | `#FFF1F0` |
| 边框 | `severity-critical-border` | `#FFA39E` |
| 深色文字 | `severity-critical-text` | `#A8071A` |

### 主要 (Major)

| 用途 | Token | Hex |
|------|-------|-----|
| 主色 | `severity-major` | `#FA8C16` |
| 背景 | `severity-major-bg` | `#FFF7E6` |
| 边框 | `severity-major-border` | `#FFD591` |
| 深色文字 | `severity-major-text` | `#AD4E00` |

### 次要 (Minor)

| 用途 | Token | Hex |
|------|-------|-----|
| 主色 | `severity-minor` | `#FAAD14` |
| 背景 | `severity-minor-bg` | `#FFFBE6` |
| 边框 | `severity-minor-border` | `#FFE58F` |
| 深色文字 | `severity-minor-text` | `#AD6800` |

### 警告 (Warning)

| 用途 | Token | Hex |
|------|-------|-----|
| 主色 | `severity-warning` | `#1890FF` |
| 背景 | `severity-warning-bg` | `#E6F7FF` |
| 边框 | `severity-warning-border` | `#91D5FF` |
| 深色文字 | `severity-warning-text` | `#0050B3` |

## 状态色 (Status)

| 状态 | Token | 主色 | 背景色 | OMC 使用场景 |
|------|-------|------|--------|-------------|
| 在线/成功 | `status-success` | `#52C41A` | `#F6FFED` | 设备在线、命令成功、任务完成 |
| 离线/失败 | `status-error` | `#F5222D` | `#FFF1F0` | 设备离线、命令失败、监控异常 |
| 进行中 | `status-processing` | `#1677FF` | `#E6F7FF` | 任务执行中、同步中、等待中 |
| 异常/暂停 | `status-warning` | `#FA8C16` | `#FFF7E6` | 任务暂停、设备异常 |
| 未知/禁用 | `status-inactive` | `#8C8C8C` | `#FAFAFA` | 未注册、未知状态、禁用 |
| 锁定 | `status-locked` | `#722ED1` | `#F9F0FF` | 设备锁定、HaloB 锁定 |

## 侧边栏深色主题

| Token | 值 | 用途 |
|-------|-----|------|
| `sidebar-bg` | `#001529` | 侧边栏背景 |
| `sidebar-bg-hover` | `rgba(255,255,255,0.08)` | 菜单项悬浮 |
| `sidebar-bg-active` | `#1677FF` | 选中菜单项背景 |
| `sidebar-text` | `rgba(255,255,255,0.65)` | 默认菜单文字 |
| `sidebar-text-active` | `#FFFFFF` | 选中菜单文字 |
| `sidebar-text-hover` | `rgba(255,255,255,0.85)` | 悬浮菜单文字 |
| `sidebar-divider` | `rgba(255,255,255,0.06)` | 菜单组分隔线 |
| `sidebar-submenu-bg` | `#000C17` | 展开子菜单背景 |

## 图表调色板

用于数据可视化，8 色序列：

| 序号 | Hex | 用途 |
|------|-----|------|
| Series 1 | `#1677FF` | 主数据系列 |
| Series 2 | `#52C41A` | 在线/成功数据 |
| Series 3 | `#FA8C16` | 警告/注意数据 |
| Series 4 | `#F5222D` | 紧急/错误数据 |
| Series 5 | `#722ED1` | 紫色补充 |
| Series 6 | `#13C2C2` | 青色补充 |
| Series 7 | `#EB2F96` | 品红补充 |
| Series 8 | `#FAAD14` | 黄色补充 |

## 使用规范

### ✅ 正确做法
- 告警等级始终使用对应的 4 级色彩
- 状态指示同时使用颜色和文字/图标（不仅依赖颜色）
- 主操作按钮使用 `primary-600`
- 危险操作按钮使用 `status-error`

### ❌ 错误做法
- 不要用告警红色作为普通强调色
- 不要混淆告警色和状态色的语义
- 不要在亮色背景上使用浅色文字
- 不要在同一图表中使用超过 8 种颜色
