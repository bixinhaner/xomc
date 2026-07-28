# Issue #190 告警趋势当前库存口径设计

## 背景

告警统计页的趋势图当前按 `raised_at` 统计每天新产生的告警事件，而页面上方告警数量统计的是 `alarms_active` 中尚未清除的当前库存。两者口径不同，导致当天趋势点与当前告警数量不一致。

## 目标

仅修正告警统计页的 Alarm Trend：

- 页面请求 `GET /api/v1/dashboard/alarm-trend?days=N&metric=active`。
- `metric=active` 返回最近 N 个自然日每日时点的未清除告警库存，按 critical/major/minor/warning 分组。
- 历史日期使用该日本地时区日末快照；今天使用请求时刻快照，因此最新点与当前告警库存一致。
- 旧调用不传 `metric` 时继续使用原有 raised-event 口径，避免影响 Dashboard 首页等既有消费者。

## 数据口径

历史日快照时刻 `t` 的库存由两部分相加：

1. 主库 `alarms_active`：`raised_at <= t`。
2. 时序库 `alarms_history`：`raised_at <= t AND cleared_at > t`。

`alarms_active.id` 与 `alarms_history.alarm_id` 分别去重计数，history 查询同时排除 active 查询已命中的告警 ID，避免“先归档、后删除”窗口跨库重复计数。今天的最新点仅查询 `alarms_active`，与当前告警卡片保持同一数据源。告警级别同时兼容旧编号 `1..4` 与字典码 `31001..31004`。

服务端向主库批量提交全部快照时刻，向时序库批量提交历史日快照时刻，通过 `unnest(... WITH ORDINALITY)` 分桶，避免按天逐次查询。缺失日期补零，结果始终按日期升序返回完整 N 点。

active 和 history 两侧都复用统一的设备组可见性过滤，确保受限用户的趋势与同页告警库存卡口径一致。前端直接使用服务端返回的日期桶，不再按浏览器本地时区重新生成日期键。

## 接口兼容

- `metric` 允许 `raised`、`active`，缺省为 `raised`。
- 非法 `metric` 返回 400。
- 返回结构不变：`date`, `critical`, `major`, `minor`, `warning`。
- `days` 的现有校验与最多 365 天限制保持不变。

## 范围边界

不修改 Summary、Dashboard 首页、告警列表、历史告警、清除逻辑、数据库表结构、索引、全局时区配置或共享图表组件。
