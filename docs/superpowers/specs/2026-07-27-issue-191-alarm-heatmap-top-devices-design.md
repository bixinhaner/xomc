# Issue #191 告警热力图与 Top10 设备设计

## 目标

仅调整告警统计页的两个图表：

1. Alarm Heatmap 保留 7/30 天切换，增加“全部/紧急/重要/次要/警告”级别选择，以级别色系和可见数值色阶体现当前筛选条件。
2. Top Alarm Devices 使用当前未清除告警的真实 Top10，显示完整 SN，并用四级告警数量的堆叠条形体现级别构成。

## Alarm Heatmap

- “全部”继续请求既有 `GET /dashboard/alarm-heatmap?days=N`。
- 指定级别请求既有 `GET /dashboard/alarm-heatmap-by-severity?days=N&severity=...`，使用响应中的 `data`。
- 查询缓存键同时包含 `days` 与 `severity`，两者任一变化都重新取数。
- 全部使用蓝色色阶；紧急红色、重要橙色、次要黄色、警告蓝色。
- `visualMap` 显示 0 到当前响应 `max_count` 的数值图例。
- Tooltip 显示星期、小时、当前级别和数量。
- 周期、级别、星期、Tooltip 等所有可见文案走 i18n。

不新增或修改热力图后端结构，不在同一格混合展示“主导告警级别”。

## Top Alarm Devices

新增 `GET /api/v1/dashboard/top-alarm-devices`：

- 只查询主库 `alarms_active`，即当前未清除告警。
- 按 `device_sn + technology` 分组。
- 固定返回最多 10 条，排序为 `alarm_count DESC, device_sn ASC`。
- 返回 `device_sn`, `technology`, `alarm_count`, `critical`, `major`, `minor`, `warning`。
- 级别同时兼容 `1..4` 与 `31001..31004`。
- 复用统一设备组可见性过滤，受限用户只能看到其可见设备聚合。

前端直接使用专用端点，不再从 Summary 的 5 条 `recent_alarms` 推导 Top10。Y 轴显示完整 SN；四个级别使用标准颜色并堆叠；Tooltip 显示完整 SN、四级数量和总数；点击仍以完整 SN 跳转。

## 范围边界

不删除或重构 Summary 的 `recent_alarms`，不修改 Dashboard 首页，不增加时间/级别筛选接口参数，不修改数据库结构或索引，不重构共享 BarChart，不修改告警列表与历史告警页面。
