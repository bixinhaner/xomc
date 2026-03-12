# 08 — Alarm (告警管理)

> **覆盖状态：⚠️ 部分覆盖**

---

## 原系统二级菜单

| # | 二级菜单 | 原系统文件数 | 新系统对应 | 状态 |
|---|---------|------------|----------|------|
| 1 | Library → AlarmLevelConf (告警级别配置) | 1 | Alarm → AlarmRules + AlarmSupportLibrary | ⚠️ |
| 2 | Setting → AlarmFilter (告警过滤规则) | 3 | Alarm → AlarmRules | ⚠️ |
| 3 | Cell/Fault → FaultList (活跃告警) | 1 | Alarm → CurrentAlarms | ✅ |
| 4 | Cell/Fault → AlarmDetail (告警详情) | 1 | CurrentAlarms 行展开 | ⚠️ 字段少 |
| 5 | Cell/Fault → AlarmConfirm (告警确认) | 1 | CurrentAlarms 确认按钮 | ✅ |
| 6 | Cell/Fault → AlarmStatistic (告警统计) | 3 | Alarm → AlarmStatistics | ✅ |
| 7 | Cell/Fault → Notification (邮件通知) | 1 | — | ❌ 缺失 |
| 8 | Cell/Fault → AlarmNoticeSetting (通知设置) | 1 | — | ❌ 缺失 |
| 9 | Cell/Fault → View (可配置仪表板) | 3 | — | ❌ 缺失 |
| 10 | Cell/Fault → ItfnFaultList (接口故障) | 1 | — | ❌ 缺失 |

---

## 8.1 Library → Alarm Level Configuration

### 原系统 `alarm/library/alarm_level_conf.jsp`

**表格列：**
- `DEVICE_TYPE_NAME` — 设备类型名称
- `ALARM_IDENTIFIER` — 告警标识
- `ALARM_NAME` — 可能原因
- `SERVERITY_TYPE` — 严重等级（内联编辑弹出菜单：Critical/Major/Minor/Warning）
- `EVENT_TYPE` — 事件类型
- Explanation — 说明

**操作：** 搜索(标识/名称), 导出, 右键菜单修改等级

### 新系统 Alarm → AlarmSupportLibrary

**表格列：** alarmCode, alarmName, severity, neType, vendor, possibleCauses, handlingSuggestions, updateTime

**详情抽屉：** 告警码, 名称, 等级, NE类型, 厂商, 可能原因(列表), 处理建议(列表)

| 原系统字段 | 新系统 | 状态 |
|-----------|--------|------|
| DEVICE_TYPE_NAME | neType | ✅ |
| ALARM_IDENTIFIER | alarmCode | ✅ |
| ALARM_NAME | alarmName | ✅ |
| SERVERITY_TYPE | severity | ✅ |
| EVENT_TYPE | — | ❌ |
| 内联编辑等级 | — | ❌ 只读 |
| 右键修改等级 | — | ❌ |
| — | vendor | ✅ **新增** |
| — | possibleCauses | ✅ **新增** |
| — | handlingSuggestions | ✅ **新增** |

---

## 8.2 Setting → Alarm Filter Rules

### 原系统 `alarm/setting/setting_main.jsp`

**表格列：**
- `status` — 启用/停用
- `rule_name` — 规则名称
- `device_type` — 设备类型
- `rule_type` — 动作类型（抑制/排除/包含/自动确认）
- `user_code` — 操作人
- `operation_time` — 更新时间

**操作：** 查看/启用/停用/编辑/删除, 新增规则

### 原系统 `filter/template_add.jsp`

**表单：**
- 规则名称
- 设备类型（多选：All/ENB/CPE 等）
- 规则类型（单选：抑制/排除/包含/自动确认）
- 告警条件（多选告警类型）

### 新系统 Alarm → AlarmRules

**表格列：** ruleName, ruleType(threshold/correlation/suppression/escalation), severity, enabled(开关), createTime, updateTime

**表单：** ruleName, ruleType, severity, description

| 原系统功能 | 新系统 | 状态 |
|-----------|--------|------|
| 规则 CRUD | ✅ CRUD | ✅ |
| 规则类型(抑制/排除/包含/自动确认) | 规则类型(threshold/correlation/suppression/escalation) | ⚠️ 不同分类 |
| 设备类型筛选 | — | ❌ |
| 告警条件多选 | — | ❌ |
| 启用/停用开关 | enabled 开关 | ✅ |

---

## 8.3 Fault → 活跃告警列表

### 原系统 `fault_list.jsp`

**表格列：** Alarm ID, Alarm Identifier, Possible Cause, Alarm Name, Severity Type, Event Type, Equipment Info, Alarm Status, Alarm Time, Update Time, Confirm Status, Clear Status

**操作：** 确认, 清除, 查看详情, 导出, 搜索, 刷新

### 新系统 Alarm → CurrentAlarms

**表格列：** severity, alarmCode, alarmName, deviceSn, deviceName, alarmContent, alarmTime, duration, ackStatus, ackUser

**操作：** 确认, 清除, 批量确认, 批量清除

| 原系统字段 | 新系统 | 状态 |
|-----------|--------|------|
| Alarm ID | — | ❌ |
| Alarm Identifier | alarmCode | ✅ |
| Alarm Name | alarmName | ✅ |
| Severity Type | severity | ✅ |
| Event Type | — | ❌ |
| Equipment Info | deviceSn + deviceName | ✅ |
| Alarm Status | — | ❌ |
| Alarm Time | alarmTime | ✅ |
| Update Time | — | ❌ |
| Confirm Status | ackStatus | ✅ |
| Clear Status | — | ❌ |
| — | alarmContent | ✅ **新增** |
| — | duration(自动计算) | ✅ **新增** |
| — | ackUser | ✅ **新增** |
| 导出 | — | ❌ (历史告警页有) |

---

## 8.4 Fault → 告警详情

### 原系统 `alarm_detail.jsp` (19 字段)

| 原系统字段 | 新系统 | 状态 |
|-----------|--------|------|
| ALARM_ID | — | ❌ |
| ALARM_IDENTIFIER | alarmCode | ✅ |
| ALARM_NAME | alarmName | ✅ |
| SPECIFIC_PROBLEM | — | ❌ |
| ADDITIONAL_INFORMATION | alarmContent(合并) | ⚠️ |
| ADDITIONAL_TEXT | — | ❌ |
| ALARM_SERVERITY | severity | ✅ |
| EVENT_TYPE | — | ❌ |
| NE_TYPE | — | ❌ |
| EQUIP_INFO | deviceSn + deviceName | ✅ |
| DEAL_STATE | ackStatus | ✅ |
| EVENT_TIME | alarmTime | ✅ |
| UPD_TIME | — | ❌ |
| DEAL_USER | ackUser | ✅ |
| DEAL_TIME | — | ❌ |
| CLEAR_USER | — | ❌ |
| CLEAR_TIME | clearTime(历史告警页) | ⚠️ |
| SUGGESTION | — | ❌ |
| DEAL_MEMO | — | ❌ |

---

## 8.5 Fault → 告警确认

### 原系统 `alarm_confirm.jsp`

**表单字段：**
- `DEAL_USER` — 确认人（禁用）
- `DEAL_TIME` — 确认时间（禁用）
- 描述/备忘录 — 文本域（最长 500 字符）

### 新系统：点击确认按钮直接确认，无备忘录输入

**缺失：** ❌ 确认备忘录/描述字段

---

## 8.6 Fault → 告警统计

### 原系统 `alarm_statistic.jsp`

**图表类型：** 柱状图(按等级), 饼图(Top 10 设备/告警)
**时间控制：** 日/月 单选, 前后翻页
**筛选：** 设备分组, 设备, 告警ID
**统计维度：** 按设备分组/设备/告警ID

### 新系统 Alarm → AlarmStatistics

**卡片：** Critical/Major/Minor/Warning 数量
**图表：** 等级分布饼图, 7天趋势折线, TOP10设备柱状图, 告警类型分布, 24小时分布

| 原系统功能 | 新系统 | 状态 |
|-----------|--------|------|
| 等级柱状图 | 等级饼图 | ✅ 不同样式 |
| Top 10 | TOP10 设备柱状图 | ✅ |
| 按时段翻页 | — | ❌ |
| 按设备分组统计 | — | ❌ |
| 按告警ID统计 | — | ❌ |
| — | 7天趋势 | ✅ **新增** |
| — | 24小时分布 | ✅ **新增** |
| — | 告警类型分布 | ✅ **新增** |

---

## 8.7~8.10 缺失的二级菜单

| 二级菜单 | 原系统功能 | 状态 |
|---------|-----------|------|
| Notification (邮件通知) | 通知模板列表(名称/状态/创建人/时间), 发送结果(主题/收件人/时间/结果) | ❌ |
| AlarmNoticeSetting (通知设置) | 声音提醒开关+等级选择, 邮件通知开关+默认邮箱, 自定义模板(启用/周期/邮箱/全局), 保留时间 | ❌ |
| View (可配置仪表板) | 可配置告警面板, 设备类型选择(ENB/CPE/UPS/GSM), 时间范围, 模板创建/编辑 | ❌ |
| ItfnFaultList (接口故障) | 接口故障列表(发生时间/清除时间/状态) | ❌ |

---

## 缺失操作流程汇总

| 流程 | 状态 |
|------|------|
| 告警等级内联编辑 | ❌ |
| 告警过滤规则(设备类型+告警条件多选) | ❌ |
| 告警确认备忘录 | ❌ |
| 邮件通知全流程(模板CRUD+发送+结果) | ❌ |
| 声音/邮件通知设置 | ❌ |
| 可配置告警仪表板 | ❌ |
| 接口故障列表 | ❌ |
| 按设备分组/告警ID统计 | ❌ |
