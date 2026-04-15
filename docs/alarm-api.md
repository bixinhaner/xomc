# 告警管理模块 API 文档

**Base URL:** `/api/v1`
**认证:** Bearer Token (JWT)
**权限资源:** `alarms`

---

## 一、告警管理 (`/alarms`)

### 1.1 活动告警列表

```
GET /api/v1/alarms/active
```

**Query 参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| device_id | string | 否 | 设备 UUID |
| device_sn | string | 否 | 设备序列号 |
| carrier | string | 否 | 运营商代码 (cmcc/ctcc/cucc) |
| severity | string | 否 | 告警级别 (1-紧急 2-重要 3-次要 4-警告) |
| status | string | 否 | 告警状态 (active/acknowledged) |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

**响应:**
```json
{
  "items": [...],
  "total": 0,
  "page": 1,
  "page_size": 20,
  "total_pages": 0
}
```

### 1.2 历史告警列表

```
GET /api/v1/alarms/history
```

**Query 参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| device_id | string | 否 | 设备 UUID |
| severity | string | 否 | 告警级别 |
| start_time | string | 否 | 开始时间 (RFC3339) |
| end_time | string | 否 | 结束时间 (RFC3339) |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

**响应:** 同活动告警列表

### 1.3 告警统计

```
GET /api/v1/alarms/statistics
```

**Query 参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| device_id | string | 否 | 设备 UUID |

**响应:**
```json
{
  "total_active": 0,
  "by_severity": {},
  "by_type": {}
}
```

### 1.4 告警详情

```
GET /api/v1/alarms/:id
```

**响应:** 单个告警对象

### 1.5 确认告警

```
POST /api/v1/alarms/:id/acknowledge
```

**请求体:**
```json
{
  "acknowledged_by": "admin@test.com"
}
```

**响应:**
```json
{ "message": "alarm acknowledged" }
```

### 1.6 清除告警

```
POST /api/v1/alarms/:id/clear
```

**响应:**
```json
{ "message": "alarm cleared" }
```

### 1.7 批量确认告警

```
POST /api/v1/alarms/active/batch/acknowledge
```

**请求体:**
```json
{
  "ids": ["uuid-1", "uuid-2"],
  "acknowledged_by": "admin@test.com"
}
```

**响应:**
```json
{ "message": "alarms acknowledged", "count": 2 }
```

### 1.8 批量清除告警

```
POST /api/v1/alarms/active/batch/clear
```

**请求体:**
```json
{
  "ids": ["uuid-1", "uuid-2"]
}
```

**响应:**
```json
{ "message": "alarms cleared", "count": 2 }
```

### 1.9 标记告警已读

```
POST /api/v1/alarms/active/:id/read
```

**响应:**
```json
{ "message": "alarm marked as read" }
```

---

## 二、告警库 (`/alarm-libraries`)

### 2.1 告警库列表

```
GET /api/v1/alarm-libraries
```

**Query 参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| alarm_code | string | 否 | 告警编码 |
| alarm_source | string | 否 | 告警来源 (模糊匹配) |
| severity | int | 否 | 告警级别 |
| enabled | string | 否 | 是否启用 (true/false) |
| carrier | string | 否 | 运营商 (模糊匹配) |
| event_type | string | 否 | 事件类型 (模糊匹配) |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

### 2.2 告警库详情

```
GET /api/v1/alarm-libraries/:id
```

### 2.3 国际化列表

```
GET /api/v1/alarm-libraries/:id/i18n
```

### 2.4 创建告警库条目

```
POST /api/v1/alarm-libraries
```

**请求体:**
```json
{
  "alarm_code": "CUSTOM_ALARM",
  "alarm_source": "Device",
  "event_type": "equipment",
  "severity": 2,
  "probable_cause": "自定义告警原因",
  "explanation": "告警说明",
  "additional_info": { "key": "value" },
  "carrier": "cmcc",
  "technology": "lte",
  "enabled": true
}
```

### 2.5 创建国际化

```
POST /api/v1/alarm-libraries/:id/i18n
```

**请求体:**
```json
{
  "locale": "en-US",
  "probable_cause": "Custom alarm cause",
  "explanation": "Alarm explanation"
}
```

### 2.6 更新告警库条目

```
PUT /api/v1/alarm-libraries/:id
```

**请求体 (所有字段可选):**
```json
{
  "severity": 1,
  "enabled": false,
  "probable_cause": "更新后的原因",
  "explanation": "更新后的说明",
  "additional_info": {},
  "carrier": "ctcc",
  "technology": "nr"
}
```

### 2.7 删除告警库条目

```
DELETE /api/v1/alarm-libraries/:id
```

### 2.8 删除国际化

```
DELETE /api/v1/alarm-libraries/:id/i18n/:i18nId
```

---

## 三、告警过滤规则 (`/alarm-filters`)

### 3.1 过滤规则列表

```
GET /api/v1/alarm-filters
```

**Query 参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| filter_type | string | 否 | 过滤类型 (alarm_source/alarm_code/device_group/device) |
| action | string | 否 | 动作 (ignore/auto_acknowledge/auto_clear) |
| enabled | string | 否 | 是否启用 (true/false) |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

### 3.2 规则详情

```
GET /api/v1/alarm-filters/:id
```

### 3.3 创建规则

```
POST /api/v1/alarm-filters
```

**请求体:**
```json
{
  "name": "忽略GPS告警",
  "filter_type": "alarm_source",
  "alarm_sources": ["Device"],
  "alarm_codes": ["GPS_LOSS"],
  "device_ids": [],
  "device_group_ids": [],
  "action": "ignore",
  "acknowledge_desc": "自动忽略",
  "priority": 10,
  "enabled": true
}
```

### 3.4 更新规则

```
PUT /api/v1/alarm-filters/:id
```

**请求体:** 同创建，所有字段可选。

### 3.5 删除规则

```
DELETE /api/v1/alarm-filters/:id
```

### 3.6 启用/禁用规则

```
POST /api/v1/alarm-filters/:id/toggle
```

**响应:**
```json
{ "message": "filter rule toggled", "enabled": false }
```

---

## API 汇总

| # | 方法 | 路径 | 说明 |
|---|------|------|------|
| 1 | GET | `/alarms/active` | 活动告警列表 |
| 2 | GET | `/alarms/history` | 历史告警列表 |
| 3 | GET | `/alarms/statistics` | 告警统计 |
| 4 | GET | `/alarms/:id` | 告警详情 |
| 5 | POST | `/alarms/:id/acknowledge` | 确认告警 |
| 6 | POST | `/alarms/:id/clear` | 清除告警 |
| 7 | POST | `/alarms/active/batch/acknowledge` | 批量确认 |
| 8 | POST | `/alarms/active/batch/clear` | 批量清除 |
| 9 | POST | `/alarms/active/:id/read` | 标记已读 |
| 10 | GET | `/alarm-libraries` | 告警库列表 |
| 11 | GET | `/alarm-libraries/:id` | 告警库详情 |
| 12 | GET | `/alarm-libraries/:id/i18n` | 国际化列表 |
| 13 | POST | `/alarm-libraries` | 创建告警库条目 |
| 14 | POST | `/alarm-libraries/:id/i18n` | 创建国际化 |
| 15 | PUT | `/alarm-libraries/:id` | 更新告警库条目 |
| 16 | DELETE | `/alarm-libraries/:id` | 删除告警库条目 |
| 17 | DELETE | `/alarm-libraries/:id/i18n/:i18nId` | 删除国际化 |
| 18 | GET | `/alarm-filters` | 过滤规则列表 |
| 19 | GET | `/alarm-filters/:id` | 规则详情 |
| 20 | POST | `/alarm-filters` | 创建规则 |
| 21 | PUT | `/alarm-filters/:id` | 更新规则 |
| 22 | DELETE | `/alarm-filters/:id` | 删除规则 |
| 23 | POST | `/alarm-filters/:id/toggle` | 启用/禁用 |
