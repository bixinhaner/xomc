# 活动告警与历史告警页面分析

> 基于 `/cell/fault/view.jsp` 页面分析

## 一、页面概述

告警视图页面（view.jsp）采用统一的页面结构，通过 Tab 切换展示三种告警类型：
- **全部**：包含活动告警和历史告警
- **活动**：仅显示活动告警（Active）
- **历史**：仅显示历史告警（History）

---

## 二、活动告警（Active Alarm）

### 2.1 列表字段

| 序号 | 字段名称 | 字段ID | 宽度 | 可排序 | 说明 |
|------|----------|--------|------|--------|------|
| 1 | (详情图标) | - | 40px | - | 未读告警显示红点标识 |
| 2 | 序号 | alarm_id | 80px | ✓ | 告警唯一ID |
| 3 | 告警级别 | alarm_serverity_value | 100px | ✓ | Critical/Major/Minor/Warning |
| 4 | 告警唯一标识 | alarm_identifier | 130px | - | 告警标识码 |
| 5 | 可能原因 | alarm_name | 180px | - | 告警名称/原因描述 |
| 6 | 告警源 | ne_type | 120px | - | 网元类型 |
| 7 | 网元定位 | equip_info | 250px | - | 设备信息（SN、小区名等） |
| 8 | 事件类型 | event_type | 160px | - | 通信告警/服务质量告警/处理失败告警/设备告警/环境告警 |
| 9 | 告警状态 | deal_state | 190px | ✓ | 未确认未清除/已确认未清除 |
| 10 | 告警类型 | alarm_type | 100px | - | 显示"活动告警" |
| 11 | 故障时间 | event_time | 150px | ✓ | 告警发生时间 |
| 12 | 更新时间 | upd_time | 150px | ✓ | 最后更新时间 |
| 13 | 具体故障 | specific_problem | 150px | - | 具体问题描述 |
| 14 | 告警次数 | alarm_count | 100px | ✓ | 告警重复次数 |
| 15 | 描述 | deal_memo | 100px | - | 处理备注 |

**告警级别样式**：
- Critical（紧急告警）：红色 #FC5959
- Major（主要告警）：橙色 #FF973E
- Minor（次要告警）：黄色 #FFDA41
- Warning（警告告警）：蓝色 #60BEFC

**告警状态说明**：
- 0：未确认未清除（unconfirmInactive）
- 1：已确认未清除（confirmInactive）

### 2.2 单列操作（右键菜单）

| 操作名称 | 图标 | 说明 | 权限 |
|----------|------|------|------|
| 详细 | el-icon-operation-info | 打开告警详情侧滑页面 | - |
| 过滤告警 | el-icon-operation-alarm-filter | 过滤同类告警 | CODE_ALARM_VIEW |
| 确认告警 | el-icon-operation-confirm | 确认告警，填写描述 | CODE_ALARM_VIEW |
| 反确认告警 | el-icon-operation-unconfirm | 取消告警确认（仅已确认状态可用） | CODE_ALARM_VIEW |
| 清除告警 | el-icon-operation-clear | 清除活动告警，转为历史告警 | CODE_ALARM_VIEW |

### 2.3 批量操作

| 操作名称 | 图标 | 说明 | 权限 |
|----------|------|------|------|
| 过滤告警 | el-icon-operation-alarm-filter | 批量过滤选中的告警 | CODE_ALARM_VIEW |
| 确认告警 | el-icon-operation-confirm | 批量确认选中的告警 | CODE_ALARM_VIEW |
| 反确认告警 | el-icon-operation-unconfirm | 批量取消确认 | CODE_ALARM_VIEW |
| 清除告警 | el-icon-operation-clear | 批量清除活动告警 | CODE_ALARM_VIEW |
| 标记为已读 | el-icon-read | 批量标记已读 | - |

### 2.4 查询条件

| 条件名称 | 类型 | 选项 | 默认值 |
|----------|------|------|--------|
| 搜索方式 | 下拉选择 | 全部/告警唯一标识 | 全部（模糊查询） |
| 关键字搜索 | 输入框 | 告警唯一标识/可能原因/网元定位 | - |
| 故障时间 | 日期时间范围 | 开始时间—结束时间 | - |
| 严重程度 | 下拉选择 | 全部/紧急/主要/次要/警告 | 全部 |
| 事件类型 | 下拉选择 | 全部/通信告警/服务质量告警/处理失败告警/设备告警/环境告警 | 全部 |
| 告警源 | 下拉选择 | 动态加载（设备类型） | 全部 |
| 阅读状态 | 下拉选择 | 全部/已读/未读 | 全部 |
| 告警状态 | 下拉选择 | 全部/未确认未清除/已确认未清除 | 全部 |

**高级筛选**：支持添加筛选条件，可动态添加/移除筛选项

---

## 三、历史告警（History Alarm）

### 3.1 列表字段

| 序号 | 字段名称 | 字段ID | 宽度 | 可排序 | 说明 |
|------|----------|--------|------|--------|------|
| 1 | (详情图标) | - | 40px | - | 未读告警显示红点标识 |
| 2 | 序号 | alarm_id | 80px | ✓ | 告警唯一ID |
| 3 | 告警级别 | alarm_serverity_value | 100px | ✓ | Critical/Major/Minor/Warning |
| 4 | 告警唯一标识 | alarm_identifier | 130px | - | 告警标识码 |
| 5 | 可能原因 | alarm_name | 180px | - | 告警名称/原因描述 |
| 6 | 告警源 | ne_type | 120px | - | 网元类型 |
| 7 | 网元定位 | equip_info | 250px | - | 设备信息 |
| 8 | 事件类型 | event_type | 160px | - | 通信告警/服务质量告警等 |
| 9 | 告警状态 | deal_state | 190px | ✓ | 未确认已清除/已确认已清除 |
| 10 | 告警类型 | alarm_type | 100px | - | 显示"历史告警" |
| 11 | 故障时间 | event_time | 150px | ✓ | 告警发生时间 |
| 12 | 更新时间 | upd_time | 150px | ✓ | 最后更新时间 |
| 13 | 告警清除时间 | clear_time | 150px | ✓ | **历史告警独有** |
| 14 | 具体故障 | specific_problem | 150px | - | 具体问题描述 |
| 15 | 告警次数 | alarm_count | 100px | ✓ | 告警重复次数 |
| 16 | 描述 | deal_memo | 100px | - | 处理备注 |

**告警状态说明**：
- 2：未确认已清除（unconfirmActive）
- 3：已确认已清除（confirmActive）

### 3.2 单列操作（右键菜单）

| 操作名称 | 图标 | 说明 | 权限 |
|----------|------|------|------|
| 详细 | el-icon-operation-info | 打开告警详情侧滑页面 | - |
| 过滤告警 | el-icon-operation-alarm-filter | 过滤同类告警 | CODE_ALARM_VIEW |
| 确认告警 | el-icon-operation-confirm | 确认告警 | CODE_ALARM_VIEW |
| 反确认告警 | el-icon-operation-unconfirm | 取消告警确认（仅已确认状态可用） | CODE_ALARM_VIEW |
| 删除告警 | el-icon-operation-delete | 删除历史告警 | CODE_ALARM_VIEW |

### 3.3 批量操作

| 操作名称 | 图标 | 说明 | 权限 |
|----------|------|------|------|
| 过滤告警 | el-icon-operation-alarm-filter | 批量过滤选中的告警 | CODE_ALARM_VIEW |
| 确认告警 | el-icon-operation-confirm | 批量确认选中的告警 | CODE_ALARM_VIEW |
| 反确认告警 | el-icon-operation-unconfirm | 批量取消确认 | CODE_ALARM_VIEW |
| 删除告警 | el-icon-operation-delete | 批量删除历史告警 | CODE_ALARM_VIEW |

### 3.4 查询条件

| 条件名称 | 类型 | 选项 | 默认值 |
|----------|------|------|--------|
| 搜索方式 | 下拉选择 | 全部/告警唯一标识 | 全部（模糊查询） |
| 关键字搜索 | 输入框 | 告警唯一标识/可能原因/网元定位 | - |
| 故障时间 | 日期时间范围 | 开始时间—结束时间 | - |
| 严重程度 | 下拉选择 | 全部/紧急/主要/次要/警告 | 全部 |
| 事件类型 | 下拉选择 | 全部/通信告警/服务质量告警/处理失败告警/设备告警/环境告警 | 全部 |
| 告警源 | 下拉选择 | 动态加载（设备类型） | 全部 |
| 告警状态 | 下拉选择 | 全部/未确认已清除/已确认已清除 | 全部 |

**注意**：历史告警没有"阅读状态"筛选条件

---

## 四、差异对比

### 4.1 列表字段差异

| 差异项 | 活动告警 | 历史告警 |
|--------|----------|----------|
| 告警清除时间 | 无 | 有 |
| 阅读状态筛选 | 有 | 无 |
| 告警状态选项 | 未确认未清除/已确认未清除 | 未确认已清除/已确认已清除 |

### 4.2 操作差异

| 操作 | 活动告警 | 历史告警 |
|------|----------|----------|
| 清除告警 | ✓ | ✗ |
| 删除告警 | ✗ | ✓ |
| 标记为已读 | ✓ | ✗ |

### 4.3 状态流转

```
活动告警                          历史告警
┌─────────────────┐              ┌─────────────────┐
│ 未确认未清除(0)  │──确认──>│ 已确认未清除(1)  │
│        │        │              │        │        │
│        │        │              │        │        │
│   清除 │        │──清除──>│        │        │
│        ↓        │              │        ↓        │
│ 未确认已清除(2)  │──确认──>│ 已确认已清除(3)  │
└─────────────────┘              └─────────────────┘
```

---

## 五、告警详情页字段

| 字段名称 | 字段ID | 说明 |
|----------|--------|------|
| 序号 | ALARM_ID | 告警唯一ID |
| 告警唯一标识 | ALARM_IDENTIFIER | 告警标识码 |
| 可能原因 | ALARM_NAME | 告警名称 |
| 具体故障 | SPECIFIC_PROBLEM | 具体问题描述 |
| 附件信息 | ADDITIONAL_INFORMATION | 附加信息 |
| 附件文本 | ADDITIONAL_TEXT | 附加文本 |
| 严重程度 | ALARM_SERVERITY | 告警级别 |
| 事件类型 | EVENT_TYPE | 告警事件类型 |
| 告警源 | NE_TYPE | 网元类型 |
| 网元定位 | EQUIP_INFO | 设备信息 |
| 告警状态 | DEAL_STATE | 当前状态 |
| 故障时间 | EVENT_TIME | 告警发生时间 |
| 更新时间 | UPD_TIME | 最后更新时间 |
| 确认人 | DEAL_USER | 确认操作人（已确认时显示） |
| 确认时间 | DEAL_TIME | 确认时间（已确认时显示） |
| 告警清除人 | CLEAR_USER | 清除操作人（已清除时显示） |
| 告警清除时间 | ClEAR_TIME | 清除时间（已清除时显示） |
| 处理建议 | SUGGESTION | 告警处理建议 |
| 描述 | DEAL_MEMO | 处理备注 |

---

## 六、API 接口

| 接口名称 | URL | 说明 |
|----------|-----|------|
| 查询告警列表 | /cell/fault/*.action | 根据 alarmType 参数区分 |
| 查询告警详情 | /cell/fault/queryAlarmDetail.action | 获取单条告警详情 |
| 确认告警 | /cell/fault/confirmAlarm.action | 单条确认 |
| 批量确认告警 | /cell/fault/batchConfirmAlarm.action | 批量确认 |
| 反确认告警 | /cell/fault/cancelConfirmAlarm.action | 取消确认 |
| 清除告警 | /cell/fault/clearAlarm.action | 单条清除 |
| 批量清除告警 | /cell/fault/batchClearAlarm.action | 批量清除 |
| 删除告警 | /cell/fault/clearHistoryAlarm.action | 删除历史告警 |
| 批量删除告警 | /cell/fault/batchClearHistoryAlarm.action | 批量删除 |
| 过滤告警 | /cell/fault/batchFilterAlarm.action | 过滤同类告警 |
| 标记已读 | /fault/viewConfig/updateUnreadAlarmAlert.action | 更新阅读状态 |

---

## 七、权限码

| 权限码 | 说明 |
|--------|------|
| CODE_ALARM_VIEW | 告警查看及操作权限 |
| CODE_ALARM_LIBRARY | 告警库配置权限 |
