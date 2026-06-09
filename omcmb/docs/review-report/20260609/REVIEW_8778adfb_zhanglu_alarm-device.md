# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-06-09 03:30 |
| 提交 | 8778adfb |
| 作者 | zhanglu |
| 范围 | alarm-device |
| 变更文件数 | 5 |
| 新增行数 | +149 |
| 删除行数 | -48 |

## 变更概要

本次变更统一了当前告警、历史告警以及设备详情内当前告警页签的自动刷新交互：从“先开开关、再选频率”调整为“选择频率即开启”，并在页面级入口与 DataTable 工具栏之间消除了重复的自动刷新按钮。

同时补齐了 historical alarms 查询 Hook 的可选轮询参数，并进一步把自动刷新下拉抽成共享组件，消除了三个页面间的重复菜单和按钮状态机逻辑。

## 审查发现

### 🔴 CRITICAL (严重)

无

### 🟡 WARNING (警告)

无

### 🔵 INFO (建议)

1. 这次变更改的是用户可见交互，但目前没有看到针对“选择频率即开启”“关闭项仅在启用后出现”“隐藏 DataTable 默认实时刷新按钮”的自动化覆盖。建议后续至少补一条当前告警页或设备详情告警页签的前端交互测试，避免回归。

## 详细分析

### [omcmb/frontend-core/src/hooks/api/useAlarms.ts](omcmb/frontend-core/src/hooks/api/useAlarms.ts#L42)

将 useHistoricalAlarms 扩展为可选接收轮询参数，接口改动保持向后兼容，现有调用方不传 options 时默认仍不会开启后台轮询，没有发现签名破坏问题。

### [omcmb/webcode/src/pages/alarm/components/AutoRefreshDropdown.tsx](omcmb/webcode/src/pages/alarm/components/AutoRefreshDropdown.tsx#L15)

新增共享自动刷新下拉组件，把频率选项、当前选中态和“关闭自动刷新”项集中到一处。组件只暴露 enabled、interval 和两个 setter，边界清晰，没有把页面查询逻辑反向耦合进来。

### [omcmb/webcode/src/pages/alarm/CurrentAlarms/index.tsx](omcmb/webcode/src/pages/alarm/CurrentAlarms/index.tsx#L780)

当前告警页已切换到共享 AutoRefreshDropdown，并继续通过 [omcmb/webcode/src/pages/alarm/CurrentAlarms/index.tsx](omcmb/webcode/src/pages/alarm/CurrentAlarms/index.tsx#L856) 隐藏 DataTable 默认实时刷新按钮，行为与先前修复保持一致。

### [omcmb/webcode/src/pages/alarm/HistoricalAlarms/index.tsx](omcmb/webcode/src/pages/alarm/HistoricalAlarms/index.tsx#L183)

历史告警页接入同样的轮询参数，并改用共享 AutoRefreshDropdown；开启或切换频率时立即 refetch 的行为仍然保留，用户无需等待下一轮计时。

### [omcmb/webcode/src/pages/device/DeviceDetail/index.tsx](omcmb/webcode/src/pages/device/DeviceDetail/index.tsx#L1729)

设备详情的当前告警页签已改成通过共享 AutoRefreshDropdown 渲染右侧工具栏入口，并在 [omcmb/webcode/src/pages/device/DeviceDetail/index.tsx](omcmb/webcode/src/pages/device/DeviceDetail/index.tsx#L1736) 继续隐藏默认实时刷新图标。查询仍以 deviceSn + 分页状态为 key，没有引入额外的查询污染风险。

## 业务完整性检查

- 页面级入口与查询轮询链路完整闭环：UI 状态已连接到 React Query 的 refetchInterval。
- 自动刷新公共交互已抽到共享组件，三个页面不再各自维护重复状态机。
- 没有发现新增 API 契约、路由或后端数据结构变更。
- 历史告警 Hook 的 options 为可选参数，未破坏现有调用方。

## 业务影响范围检查

- 影响范围限定在前端告警页面和设备详情告警页签。
- useHistoricalAlarms 的签名扩展会影响所有调用点的类型推导，但当前调用方式保持兼容。
- Docker web 已重建，8081 入口可直接验证本次前端改动。

## 前后端一致性检查

- 本次未改 REST 路径、请求参数或响应字段。
- 所有变更均停留在前端轮询和交互层，前后端契约保持不变。

## 配套更新提醒

- 建议同步补一个前端交互测试，覆盖自动刷新单入口行为。

## 审查结论

PASS_WITH_WARNINGS