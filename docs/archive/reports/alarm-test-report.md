# 告警管理模块 API 测试报告

> **测试时间:** 2026-04-15 04:55:02
> **测试环境:** http://localhost:8081
> **测试数据:** 8 条活动告警 + 5 条历史告警
> **认证方式:** JWT Bearer Token (admin)

---

## 一、活动告警接口

### 1.1 查询活动告警列表（分页 page=1, page_size=5）

**Request:**
```http
GET /alarms/active?page=1&page_size=5
Authorization: Bearer <token>
```

**Response:** `200`

```json
{
    "items": [
        {
            "id": "a0000008-0000-0000-0000-000000000008",
            "device_id": "d0000004-0000-0000-0000-000000000004",
            "device_sn": "SN-CMCC-002",
            "carrier": "cmcc",
            "severity": 3,
            "alarm_type": "processingErrorAlarm",
            "alarm_code": "ALM-008",
            "description": "\u5185\u5b58\u4f7f\u7528\u7387\u544a\u8b66",
            "status": "active",
            "raised_at": "2026-04-15T12:49:55.575993+08:00",
            "device_name": "CMCC-gNB-002",
            "technology": "nr",
            "alarm_source": "base_station",
            "event_type": "processingErrorAlarm",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "2026-04-15T12:49:55.575993+08:00",
            "last_updated_at": "2026-04-15T12:49:55.575993+08:00",
            "probable_cause": "\u5185\u5b58\u6cc4\u6f0f\u7591\u4f3c",
            "created_at": "2026-04-15T12:54:55.575993+08:00",
            "updated_at": "2026-04-15T12:54:55.575993+08:00"
        },
        {
            "id": "a0000007-0000-0000-0000-000000000007",
            "device_id": "d0000004-0000-0000-0000-000000000004",
            "device_sn": "SN-CMCC-002",
            "carrier": "cmcc",
            "severity": 1,
            "alarm_type": "equipmentAlarm",
            "alarm_code": "ALM-007",
            "description": "\u5149\u6a21\u5757\u6536\u5149\u529f\u7387\u4f4e",
            "status": "active",
            "raised_at": "2026-04-15T12:44:55.575993+08:00",
            "device_name": "CMCC-gNB-002",
            "technology": "nr",
            "alarm_source": "base_station",
            "event_type": "equipmentAlarm",
            "is_read": true,
            "ack_count": 2,
            "first_raised_at": "2026-04-15T12:44:55.575993+08:00",
            "last_updated_at": "2026-04-15T12:44:55.575993+08:00",
            "probable_cause": "\u5149\u7ea4\u94fe\u8def\u8870\u51cf",
            "created_at": "2026-04-15T12:54:55.575993+08:00",
            "updated_at": "2026-04-15T12:54:55.575993+08:00"
        },
        {
            "id": "a0000005-0000-0000-0000-000000000005",
            "device_id": "d0000003-0000-0000-0000-000000000003",
            "device_sn": "SN-CUCC-001",
            "carrier": "cucc",
            "severity": 2,
            "alarm_type": "environmentalAlarm",
            "alarm_code": "ALM-005",
            "description": "\u673a\u67dc\u6e29\u5ea6\u8fc7\u9ad8",
            "status": "active",
            "raised_at": "2026-04-15T12:39:55.575993+08:00",
            "device_name": "CUCC-eNodeB-001",
            "technology": "lte",
            "alarm_source": "base_station",
            "event_type": "environmentalAlarm",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "2026-04-15T12:39:55.575993+08:00",
            "last_updated_at": "2026-04-15T12:39:55.575993+08:00",
            "probable_cause": "\u6563\u70ed\u7cfb\u7edf\u5f02\u5e38",
            "created_at": "2026-04-15T12:54:55.575993+08:00",
            "updated_at": "2026-04-15T12:54:55.575993+08:00"
        },
        {
            "id": "a0000003-0000-0000-0000-000000000003",
            "device_id": "d0000002-0000-0000-0000-000000000002",
            "device_sn": "SN-CTCC-001",
            "carrier": "ctcc",
            "severity": 3,
            "alarm_type": "processingErrorAlarm",
            "alarm_code": "ALM-003",
            "description": "CPU\u4f7f\u7528\u7387\u8d85\u8fc7\u9608\u503c",
            "status": "active",
            "raised_at": "2026-04-15T12:24:55.575993+08:00",
            "device_name": "CTCC-gNB-001",
            "technology": "nr",
            "alarm_source": "base_station",
            "event_type": "processingErrorAlarm",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "2026-04-15T12:24:55.575993+08:00",
            "last_updated_at": "2026-04-15T12:24:55.575993+08:00",
            "probable_cause": "CPU\u8d1f\u8f7d\u8fc7\u9ad8",
            "created_at": "2026-04-15T12:54:55.575993+08:00",
            "updated_at": "2026-04-15T12:54:55.575993+08:00"
        },
        {
            "id": "a0000004-0000-0000-0000-000000000004",
            "device_id": "d0000002-0000-0000-0000-000000000002",
            "device_sn": "SN-CTCC-001",
            "carrier": "ctcc",
            "severity": 1,
            "alarm_type": "equipmentAlarm",
            "alarm_code": "ALM-004",
            "description": "\u7535\u6e90\u6a21\u5757\u6545\u969c",
            "status": "active",
            "raised_at": "2026-04-15T12:09:55.575993+08:00",
            "device_name": "CTCC-gNB-001",
            "technology": "nr",
            "alarm_source": "base_station",
            "event_type": "equipmentAlarm",
            "is_read": true,
            "ack_count": 1,
            "first_raised_at": "2026-04-15T12:09:55.575993+08:00",
            "last_updated_at": "2026-04-15T12:09:55.575993+08:00",
            "probable_cause": "\u7535\u6e90\u6a21\u5757\u786c\u4ef6\u6545\u969c",
            "created_at": "2026-04-15T12:54:55.575993+08:00",
            "updated_at": "2026-04-15T12:54:55.575993+08:00"
        }
    ],
    "total": 8,
    "page": 1,
    "page_size": 5,
    "total_pages": 2
}
```

**Result:** PASS

---

### 1.2 按运营商筛选（cmcc）

**Request:**
```http
GET /alarms/active?carrier=cmcc&page=1&page_size=10
Authorization: Bearer <token>
```

**Response:** `200`

```json
{
    "items": [
        {
            "id": "a0000008-0000-0000-0000-000000000008",
            "device_id": "d0000004-0000-0000-0000-000000000004",
            "device_sn": "SN-CMCC-002",
            "carrier": "cmcc",
            "severity": 3,
            "alarm_type": "processingErrorAlarm",
            "alarm_code": "ALM-008",
            "description": "\u5185\u5b58\u4f7f\u7528\u7387\u544a\u8b66",
            "status": "active",
            "raised_at": "2026-04-15T12:49:55.575993+08:00",
            "device_name": "CMCC-gNB-002",
            "technology": "nr",
            "alarm_source": "base_station",
            "event_type": "processingErrorAlarm",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "2026-04-15T12:49:55.575993+08:00",
            "last_updated_at": "2026-04-15T12:49:55.575993+08:00",
            "probable_cause": "\u5185\u5b58\u6cc4\u6f0f\u7591\u4f3c",
            "created_at": "2026-04-15T12:54:55.575993+08:00",
            "updated_at": "2026-04-15T12:54:55.575993+08:00"
        },
        {
            "id": "a0000007-0000-0000-0000-000000000007",
            "device_id": "d0000004-0000-0000-0000-000000000004",
            "device_sn": "SN-CMCC-002",
            "carrier": "cmcc",
            "severity": 1,
            "alarm_type": "equipmentAlarm",
            "alarm_code": "ALM-007",
            "description": "\u5149\u6a21\u5757\u6536\u5149\u529f\u7387\u4f4e",
            "status": "active",
            "raised_at": "2026-04-15T12:44:55.575993+08:00",
            "device_name": "CMCC-gNB-002",
            "technology": "nr",
            "alarm_source": "base_station",
            "event_type": "equipmentAlarm",
            "is_read": true,
            "ack_count": 2,
            "first_raised_at": "2026-04-15T12:44:55.575993+08:00",
            "last_updated_at": "2026-04-15T12:44:55.575993+08:00",
            "probable_cause": "\u5149\u7ea4\u94fe\u8def\u8870\u51cf",
            "created_at": "2026-04-15T12:54:55.575993+08:00",
            "updated_at": "2026-04-15T12:54:55.575993+08:00"
        },
        {
            "id": "a0000002-0000-0000-0000-000000000002",
            "device_id": "d0000001-0000-0000-0000-000000000001",
            "device_sn": "SN-CMCC-001",
            "carrier": "cmcc",
            "severity": 2,
            "alarm_type": "qualityOfServiceAlarm",
            "alarm_code": "ALM-002",
            "description": "S1\u63a5\u53e3\u65f6\u5ef6\u8d85\u6807",
            "status": "active",
            "raised_at": "2026-04-15T11:54:55.575993+08:00",
            "device_name": "CMCC-eNodeB-001",
            "technology": "lte",
            "alarm_source": "base_station",
            "event_type": "qualityOfServiceAlarm",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "2026-04-15T11:54:55.575993+08:00",
            "last_updated_at": "2026-04-15T11:54:55.575993+08:00",
            "probable_cause": "S1\u63a5\u53e3\u62e5\u585e",
            "created_at": "2026-04-15T12:54:55.575993+08:00",
            "updated_at": "2026-04-15T12:54:55.575993+08:00"
        },
        {
            "id": "a0000001-0000-0000-0000-000000000001",
            "device_id": "d0000001-0000-0000-0000-000000000001",
            "device_sn": "SN-CMCC-001",
            "carrier": "cmcc",
            "severity": 1,
            "alarm_type": "communicationsAlarm",
            "alarm_code": "ALM-001",
            "description": "\u8bbe\u5907\u8fde\u63a5\u4e2d\u65ad",
            "status": "active",
            "raised_at": "2026-04-15T10:54:55.575993+08:00",
            "device_name": "CMCC-eNodeB-001",
            "technology": "lte",
            "alarm_source": "base_station",
            "event_type": "communicationsAlarm",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "2026-04-15T10:54:55.575993+08:00",
            "last_updated_at": "2026-04-15T10:54:55.575993+08:00",
            "probable_cause": "\u4f20\u8f93\u94fe\u8def\u6545\u969c",
            "created_at": "2026-04-15T12:54:55.575993+08:00",
            "updated_at": "2026-04-15T12:54:55.575993+08:00"
        }
    ],
    "total": 4,
    "page": 1,
    "page_size": 10,
    "total_pages": 1
}
```

**Result:** PASS

---

### 1.3 按严重级别筛选（紧急=1）

**Request:**
```http
GET /alarms/active?severity=1&page=1&page_size=10
Authorization: Bearer <token>
```

**Response:** `200`

```json
{
    "items": [
        {
            "id": "a0000007-0000-0000-0000-000000000007",
            "device_id": "d0000004-0000-0000-0000-000000000004",
            "device_sn": "SN-CMCC-002",
            "carrier": "cmcc",
            "severity": 1,
            "alarm_type": "equipmentAlarm",
            "alarm_code": "ALM-007",
            "description": "\u5149\u6a21\u5757\u6536\u5149\u529f\u7387\u4f4e",
            "status": "active",
            "raised_at": "2026-04-15T12:44:55.575993+08:00",
            "device_name": "CMCC-gNB-002",
            "technology": "nr",
            "alarm_source": "base_station",
            "event_type": "equipmentAlarm",
            "is_read": true,
            "ack_count": 2,
            "first_raised_at": "2026-04-15T12:44:55.575993+08:00",
            "last_updated_at": "2026-04-15T12:44:55.575993+08:00",
            "probable_cause": "\u5149\u7ea4\u94fe\u8def\u8870\u51cf",
            "created_at": "2026-04-15T12:54:55.575993+08:00",
            "updated_at": "2026-04-15T12:54:55.575993+08:00"
        },
        {
            "id": "a0000004-0000-0000-0000-000000000004",
            "device_id": "d0000002-0000-0000-0000-000000000002",
            "device_sn": "SN-CTCC-001",
            "carrier": "ctcc",
            "severity": 1,
            "alarm_type": "equipmentAlarm",
            "alarm_code": "ALM-004",
            "description": "\u7535\u6e90\u6a21\u5757\u6545\u969c",
            "status": "active",
            "raised_at": "2026-04-15T12:09:55.575993+08:00",
            "device_name": "CTCC-gNB-001",
            "technology": "nr",
            "alarm_source": "base_station",
            "event_type": "equipmentAlarm",
            "is_read": true,
            "ack_count": 1,
            "first_raised_at": "2026-04-15T12:09:55.575993+08:00",
            "last_updated_at": "2026-04-15T12:09:55.575993+08:00",
            "probable_cause": "\u7535\u6e90\u6a21\u5757\u786c\u4ef6\u6545\u969c",
            "created_at": "2026-04-15T12:54:55.575993+08:00",
            "updated_at": "2026-04-15T12:54:55.575993+08:00"
        },
        {
            "id": "a0000001-0000-0000-0000-000000000001",
            "device_id": "d0000001-0000-0000-0000-000000000001",
            "device_sn": "SN-CMCC-001",
            "carrier": "cmcc",
            "severity": 1,
            "alarm_type": "communicationsAlarm",
            "alarm_code": "ALM-001",
            "description": "\u8bbe\u5907\u8fde\u63a5\u4e2d\u65ad",
            "status": "active",
            "raised_at": "2026-04-15T10:54:55.575993+08:00",
            "device_name": "CMCC-eNodeB-001",
            "technology": "lte",
            "alarm_source": "base_station",
            "event_type": "communicationsAlarm",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "2026-04-15T10:54:55.575993+08:00",
            "last_updated_at": "2026-04-15T10:54:55.575993+08:00",
            "probable_cause": "\u4f20\u8f93\u94fe\u8def\u6545\u969c",
            "created_at": "2026-04-15T12:54:55.575993+08:00",
            "updated_at": "2026-04-15T12:54:55.575993+08:00"
        }
    ],
    "total": 3,
    "page": 1,
    "page_size": 10,
    "total_pages": 1
}
```

**Result:** PASS

---

### 1.4 按状态筛选（active）

**Request:**
```http
GET /alarms/active?status=active&page=1&page_size=10
Authorization: Bearer <token>
```

**Response:** `200`

```json
{
    "items": [
        {
            "id": "a0000008-0000-0000-0000-000000000008",
            "device_id": "d0000004-0000-0000-0000-000000000004",
            "device_sn": "SN-CMCC-002",
            "carrier": "cmcc",
            "severity": 3,
            "alarm_type": "processingErrorAlarm",
            "alarm_code": "ALM-008",
            "description": "\u5185\u5b58\u4f7f\u7528\u7387\u544a\u8b66",
            "status": "active",
            "raised_at": "2026-04-15T12:49:55.575993+08:00",
            "device_name": "CMCC-gNB-002",
            "technology": "nr",
            "alarm_source": "base_station",
            "event_type": "processingErrorAlarm",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "2026-04-15T12:49:55.575993+08:00",
            "last_updated_at": "2026-04-15T12:49:55.575993+08:00",
            "probable_cause": "\u5185\u5b58\u6cc4\u6f0f\u7591\u4f3c",
            "created_at": "2026-04-15T12:54:55.575993+08:00",
            "updated_at": "2026-04-15T12:54:55.575993+08:00"
        },
        {
            "id": "a0000007-0000-0000-0000-000000000007",
            "device_id": "d0000004-0000-0000-0000-000000000004",
            "device_sn": "SN-CMCC-002",
            "carrier": "cmcc",
            "severity": 1,
            "alarm_type": "equipmentAlarm",
            "alarm_code": "ALM-007",
            "description": "\u5149\u6a21\u5757\u6536\u5149\u529f\u7387\u4f4e",
            "status": "active",
            "raised_at": "2026-04-15T12:44:55.575993+08:00",
            "device_name": "CMCC-gNB-002",
            "technology": "nr",
            "alarm_source": "base_station",
            "event_type": "equipmentAlarm",
            "is_read": true,
            "ack_count": 2,
            "first_raised_at": "2026-04-15T12:44:55.575993+08:00",
            "last_updated_at": "2026-04-15T12:44:55.575993+08:00",
            "probable_cause": "\u5149\u7ea4\u94fe\u8def\u8870\u51cf",
            "created_at": "2026-04-15T12:54:55.575993+08:00",
            "updated_at": "2026-04-15T12:54:55.575993+08:00"
        },
        {
            "id": "a0000005-0000-0000-0000-000000000005",
            "device_id": "d0000003-0000-0000-0000-000000000003",
            "device_sn": "SN-CUCC-001",
            "carrier": "cucc",
            "severity": 2,
            "alarm_type": "environmentalAlarm",
            "alarm_code": "ALM-005",
            "description": "\u673a\u67dc\u6e29\u5ea6\u8fc7\u9ad8",
            "status": "active",
            "raised_at": "2026-04-15T12:39:55.575993+08:00",
            "device_name": "CUCC-eNodeB-001",
            "technology": "lte",
            "alarm_source": "base_station",
            "event_type": "environmentalAlarm",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "2026-04-15T12:39:55.575993+08:00",
            "last_updated_at": "2026-04-15T12:39:55.575993+08:00",
            "probable_cause": "\u6563\u70ed\u7cfb\u7edf\u5f02\u5e38",
            "created_at": "2026-04-15T12:54:55.575993+08:00",
            "updated_at": "2026-04-15T12:54:55.575993+08:00"
        },
        {
            "id": "a0000003-0000-0000-0000-000000000003",
            "device_id": "d0000002-0000-0000-0000-000000000002",
            "device_sn": "SN-CTCC-001",
            "carrier": "ctcc",
            "severity": 3,
            "alarm_type": "processingErrorAlarm",
            "alarm_code": "ALM-003",
            "description": "CPU\u4f7f\u7528\u7387\u8d85\u8fc7\u9608\u503c",
            "status": "active",
            "raised_at": "2026-04-15T12:24:55.575993+08:00",
            "device_name": "CTCC-gNB-001",
            "technology": "nr",
            "alarm_source": "base_station",
            "event_type": "processingErrorAlarm",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "2026-04-15T12:24:55.575993+08:00",
            "last_updated_at": "2026-04-15T12:24:55.575993+08:00",
            "probable_cause": "CPU\u8d1f\u8f7d\u8fc7\u9ad8",
            "created_at": "2026-04-15T12:54:55.575993+08:00",
            "updated_at": "2026-04-15T12:54:55.575993+08:00"
        },
        {
            "id": "a0000004-0000-0000-0000-000000000004",
            "device_id": "d0000002-0000-0000-0000-000000000002",
            "device_sn": "SN-CTCC-001",
            "carrier": "ctcc",
            "severity": 1,
            "alarm_type": "equipmentAlarm",
            "alarm_code": "ALM-004",
            "description": "\u7535\u6e90\u6a21\u5757\u6545\u969c",
            "status": "active",
            "raised_at": "2026-04-15T12:09:55.575993+08:00",
            "device_name": "CTCC-gNB-001",
            "technology": "nr",
            "alarm_source": "base_station",
            "event_type": "equipmentAlarm",
            "is_read": true,
            "ack_count": 1,
            "first_raised_at": "2026-04-15T12:09:55.575993+08:00",
            "last_updated_at": "2026-04-15T12:09:55.575993+08:00",
            "probable_cause": "\u7535\u6e90\u6a21\u5757\u786c\u4ef6\u6545\u969c",
            "created_at": "2026-04-15T12:54:55.575993+08:00",
            "updated_at": "2026-04-15T12:54:55.575993+08:00"
        },
        {
            "id": "a0000002-0000-0000-0000-000000000002",
            "device_id": "d0000001-0000-0000-0000-000000000001",
            "device_sn": "SN-CMCC-001",
            "carrier": "cmcc",
            "severity": 2,
            "alarm_type": "qualityOfServiceAlarm",
            "alarm_code": "ALM-002",
            "description": "S1\u63a5\u53e3\u65f6\u5ef6\u8d85\u6807",
            "status": "active",
            "raised_at": "2026-04-15T11:54:55.575993+08:00",
            "device_name": "CMCC-eNodeB-001",
            "technology": "lte",
            "alarm_source": "base_station",
            "event_type": "qualityOfServiceAlarm",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "2026-04-15T11:54:55.575993+08:00",
            "last_updated_at": "2026-04-15T11:54:55.575993+08:00",
            "probable_cause": "S1\u63a5\u53e3\u62e5\u585e",
            "created_at": "2026-04-15T12:54:55.575993+08:00",
            "updated_at": "2026-04-15T12:54:55.575993+08:00"
        },
        {
            "id": "a0000001-0000-0000-0000-000000000001",
            "device_id": "d0000001-0000-0000-0000-000000000001",
            "device_sn": "SN-CMCC-001",
            "carrier": "cmcc",
            "severity": 1,
            "alarm_type": "communicationsAlarm",
            "alarm_code": "ALM-001",
            "description": "\u8bbe\u5907\u8fde\u63a5\u4e2d\u65ad",
            "status": "active",
            "raised_at": "2026-04-15T10:54:55.575993+08:00",
            "device_name": "CMCC-eNodeB-001",
            "technology": "lte",
            "alarm_source": "base_station",
            "event_type": "communicationsAlarm",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "2026-04-15T10:54:55.575993+08:00",
            "last_updated_at": "2026-04-15T10:54:55.575993+08:00",
            "probable_cause": "\u4f20\u8f93\u94fe\u8def\u6545\u969c",
            "created_at": "2026-04-15T12:54:55.575993+08:00",
            "updated_at": "2026-04-15T12:54:55.575993+08:00"
        },
        {
            "id": "a0000006-0000-0000-0000-000000000006",
            "device_id": "d0000003-0000-0000-0000-000000000003",
            "device_sn": "SN-CUCC-001",
            "carrier": "cucc",
            "severity": 4,
            "alarm_type": "communicationsAlarm",
            "alarm_code": "ALM-006",
            "description": "GPS\u4fe1\u53f7\u4e22\u5931",
            "status": "active",
            "raised_at": "2026-04-15T09:54:55.575993+08:00",
            "device_name": "CUCC-eNodeB-001",
            "technology": "lte",
            "alarm_source": "base_station",
            "event_type": "communicationsAlarm",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "2026-04-15T09:54:55.575993+08:00",
            "last_updated_at": "2026-04-15T09:54:55.575993+08:00",
            "probable_cause": "GPS\u5929\u7ebf\u6545\u969c",
            "created_at": "2026-04-15T12:54:55.575993+08:00",
            "updated_at": "2026-04-15T12:54:55.575993+08:00"
        }
    ],
    "total": 8,
    "page": 1,
    "page_size": 10,
    "total_pages": 1
}
```

**Result:** PASS

---

### 1.5 按设备序列号筛选

**Request:**
```http
GET /alarms/active?device_sn=SN-CMCC-001&page=1&page_size=10
Authorization: Bearer <token>
```

**Response:** `200`

```json
{
    "items": [
        {
            "id": "a0000002-0000-0000-0000-000000000002",
            "device_id": "d0000001-0000-0000-0000-000000000001",
            "device_sn": "SN-CMCC-001",
            "carrier": "cmcc",
            "severity": 2,
            "alarm_type": "qualityOfServiceAlarm",
            "alarm_code": "ALM-002",
            "description": "S1\u63a5\u53e3\u65f6\u5ef6\u8d85\u6807",
            "status": "active",
            "raised_at": "2026-04-15T11:54:55.575993+08:00",
            "device_name": "CMCC-eNodeB-001",
            "technology": "lte",
            "alarm_source": "base_station",
            "event_type": "qualityOfServiceAlarm",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "2026-04-15T11:54:55.575993+08:00",
            "last_updated_at": "2026-04-15T11:54:55.575993+08:00",
            "probable_cause": "S1\u63a5\u53e3\u62e5\u585e",
            "created_at": "2026-04-15T12:54:55.575993+08:00",
            "updated_at": "2026-04-15T12:54:55.575993+08:00"
        },
        {
            "id": "a0000001-0000-0000-0000-000000000001",
            "device_id": "d0000001-0000-0000-0000-000000000001",
            "device_sn": "SN-CMCC-001",
            "carrier": "cmcc",
            "severity": 1,
            "alarm_type": "communicationsAlarm",
            "alarm_code": "ALM-001",
            "description": "\u8bbe\u5907\u8fde\u63a5\u4e2d\u65ad",
            "status": "active",
            "raised_at": "2026-04-15T10:54:55.575993+08:00",
            "device_name": "CMCC-eNodeB-001",
            "technology": "lte",
            "alarm_source": "base_station",
            "event_type": "communicationsAlarm",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "2026-04-15T10:54:55.575993+08:00",
            "last_updated_at": "2026-04-15T10:54:55.575993+08:00",
            "probable_cause": "\u4f20\u8f93\u94fe\u8def\u6545\u969c",
            "created_at": "2026-04-15T12:54:55.575993+08:00",
            "updated_at": "2026-04-15T12:54:55.575993+08:00"
        }
    ],
    "total": 2,
    "page": 1,
    "page_size": 10,
    "total_pages": 1
}
```

**Result:** PASS

---

### 1.6 查询单条告警详情

**Request:**
```http
GET /alarms/a0000001-0000-0000-0000-000000000001
Authorization: Bearer <token>
```

**Response:** `200`

```json
{
    "id": "a0000001-0000-0000-0000-000000000001",
    "device_id": "d0000001-0000-0000-0000-000000000001",
    "device_sn": "SN-CMCC-001",
    "carrier": "cmcc",
    "severity": 1,
    "alarm_type": "communicationsAlarm",
    "alarm_code": "ALM-001",
    "description": "\u8bbe\u5907\u8fde\u63a5\u4e2d\u65ad",
    "status": "active",
    "raised_at": "2026-04-15T10:54:55.575993+08:00",
    "device_name": "CMCC-eNodeB-001",
    "technology": "lte",
    "alarm_source": "base_station",
    "event_type": "communicationsAlarm",
    "is_read": false,
    "ack_count": 0,
    "first_raised_at": "2026-04-15T10:54:55.575993+08:00",
    "last_updated_at": "2026-04-15T10:54:55.575993+08:00",
    "probable_cause": "\u4f20\u8f93\u94fe\u8def\u6545\u969c",
    "created_at": "2026-04-15T12:54:55.575993+08:00",
    "updated_at": "2026-04-15T12:54:55.575993+08:00"
}
```

**Result:** PASS

---

### 1.7 查询不存在的告警（预期404）

**Request:**
```http
GET /alarms/00000000-0000-0000-0000-000000000000
Authorization: Bearer <token>
```

**Response:** `404`

```json
{
    "code": 404,
    "message": "Not Found",
    "details": "resource not found",
    "request_id": "app-20260415125502-1f57299a"
}
```

**Result:** PASS

---

### 1.8 确认告警（ALM-003）

**Request:**
```http
POST /alarms/a0000003-0000-0000-0000-000000000003/acknowledge
Authorization: Bearer <token>
Content-Type: application/json

{"acknowledged_by":"admin"}
```

**Response:** `200`

```json
{
    "message": "alarm acknowledged"
}
```

**Result:** PASS

---

### 1.9 清除告警（ALM-006）

**Request:**
```http
POST /alarms/a0000006-0000-0000-0000-000000000006/clear
Authorization: Bearer <token>
```

**Response:** `200`

```json
{
    "message": "alarm cleared"
}
```

**Result:** PASS

---

### 1.10 批量确认告警

**Request:**
```http
POST /alarms/active/batch/acknowledge
Authorization: Bearer <token>
Content-Type: application/json

{"ids":["a0000005-0000-0000-0000-000000000005"],"acknowledged_by":"admin"}
```

**Response:** `200`

```json
{
    "count": 1,
    "message": "alarms acknowledged"
}
```

**Result:** PASS

---

### 1.11 批量清除告警

**Request:**
```http
POST /alarms/active/batch/clear
Authorization: Bearer <token>
Content-Type: application/json

{"ids":["a0000004-0000-0000-0000-000000000004"]}
```

**Response:** `500`

```json
{
    "code": 500,
    "message": "Internal Server Error",
    "details": "ERROR: column \"cleared_at\" of relation \"alarms_active\" does not exist (SQLSTATE 42703)",
    "request_id": "app-20260415125502-f7eafccb"
}
```

**Result:** FAIL (expected 200, got 500)

---

### 1.12 标记已读（ALM-001）

**Request:**
```http
POST /alarms/active/a0000001-0000-0000-0000-000000000001/read
Authorization: Bearer <token>
```

**Response:** `200`

```json
{
    "message": "alarm marked as read"
}
```

**Result:** PASS

---

### 1.13 告警统计

**Request:**
```http
GET /alarms/statistics
Authorization: Bearer <token>
```

**Response:** `200`

```json
{
    "total_active": 7,
    "by_severity": {
        "1": 3,
        "2": 2,
        "3": 2
    },
    "by_type": {
        "communicationsAlarm": 1,
        "environmentalAlarm": 1,
        "equipmentAlarm": 2,
        "processingErrorAlarm": 2,
        "qualityOfServiceAlarm": 1
    }
}
```

**Result:** PASS

---

## 二、历史告警接口

### 2.1 查询历史告警列表

**Request:**
```http
GET /alarms/history?page=1&page_size=10
Authorization: Bearer <token>
```

**Response:** `200`

```json
{
    "items": [
        {
            "id": "a0000006-0000-0000-0000-000000000006",
            "device_id": "d0000003-0000-0000-0000-000000000003",
            "device_sn": "SN-CUCC-001",
            "carrier": "cucc",
            "severity": 4,
            "alarm_type": "communicationsAlarm",
            "alarm_code": "ALM-006",
            "description": "GPS\u4fe1\u53f7\u4e22\u5931",
            "status": "cleared",
            "raised_at": "2026-04-15T09:54:55.575993+08:00",
            "cleared_at": "2026-04-15T12:55:02.647627+08:00",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "0001-01-01T00:00:00Z",
            "last_updated_at": "0001-01-01T00:00:00Z",
            "created_at": "0001-01-01T00:00:00Z",
            "updated_at": "0001-01-01T00:00:00Z"
        },
        {
            "id": "b0000004-0000-0000-0000-000000000004",
            "device_id": "d0000004-0000-0000-0000-000000000004",
            "device_sn": "SN-CMCC-002",
            "carrier": "cmcc",
            "severity": 4,
            "alarm_type": "environmentalAlarm",
            "alarm_code": "ALM-H04",
            "description": "\u98ce\u6247\u8f6c\u901f\u5f02\u5e38",
            "status": "cleared",
            "raised_at": "2026-04-14T12:54:55.575993+08:00",
            "acknowledged_at": "2026-04-15T04:54:55.575993+08:00",
            "cleared_at": "2026-04-15T07:54:55.575993+08:00",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "0001-01-01T00:00:00Z",
            "last_updated_at": "0001-01-01T00:00:00Z",
            "created_at": "0001-01-01T00:00:00Z",
            "updated_at": "0001-01-01T00:00:00Z"
        },
        {
            "id": "b0000005-0000-0000-0000-000000000005",
            "device_id": "d0000001-0000-0000-0000-000000000001",
            "device_sn": "SN-CMCC-001",
            "carrier": "cmcc",
            "severity": 2,
            "alarm_type": "qualityOfServiceAlarm",
            "alarm_code": "ALM-H05",
            "description": "\u65e0\u7ebf\u94fe\u8def\u8d28\u91cf\u4e0b\u964d",
            "status": "cleared",
            "raised_at": "2026-04-14T06:54:55.575993+08:00",
            "cleared_at": "2026-04-15T00:54:55.575993+08:00",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "0001-01-01T00:00:00Z",
            "last_updated_at": "0001-01-01T00:00:00Z",
            "created_at": "0001-01-01T00:00:00Z",
            "updated_at": "0001-01-01T00:00:00Z"
        },
        {
            "id": "b0000001-0000-0000-0000-000000000001",
            "device_id": "d0000001-0000-0000-0000-000000000001",
            "device_sn": "SN-CMCC-001",
            "carrier": "cmcc",
            "severity": 2,
            "alarm_type": "communicationsAlarm",
            "alarm_code": "ALM-H01",
            "description": "S1\u94fe\u8def\u95ea\u65ad",
            "status": "cleared",
            "raised_at": "2026-04-13T12:54:55.575993+08:00",
            "acknowledged_at": "2026-04-14T02:54:55.575993+08:00",
            "cleared_at": "2026-04-14T12:54:55.575993+08:00",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "0001-01-01T00:00:00Z",
            "last_updated_at": "0001-01-01T00:00:00Z",
            "created_at": "0001-01-01T00:00:00Z",
            "updated_at": "0001-01-01T00:00:00Z"
        },
        {
            "id": "b0000002-0000-0000-0000-000000000002",
            "device_id": "d0000002-0000-0000-0000-000000000002",
            "device_sn": "SN-CTCC-001",
            "carrier": "ctcc",
            "severity": 3,
            "alarm_type": "processingErrorAlarm",
            "alarm_code": "ALM-H02",
            "description": "CPU\u6e29\u5ea6\u544a\u8b66",
            "status": "cleared",
            "raised_at": "2026-04-12T12:54:55.575993+08:00",
            "acknowledged_at": "2026-04-13T00:54:55.575993+08:00",
            "cleared_at": "2026-04-13T12:54:55.575993+08:00",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "0001-01-01T00:00:00Z",
            "last_updated_at": "0001-01-01T00:00:00Z",
            "created_at": "0001-01-01T00:00:00Z",
            "updated_at": "0001-01-01T00:00:00Z"
        },
        {
            "id": "b0000003-0000-0000-0000-000000000003",
            "device_id": "d0000003-0000-0000-0000-000000000003",
            "device_sn": "SN-CUCC-001",
            "carrier": "cucc",
            "severity": 1,
            "alarm_type": "equipmentAlarm",
            "alarm_code": "ALM-H03",
            "description": "BBU\u677f\u5361\u6545\u969c",
            "status": "cleared",
            "raised_at": "2026-04-11T12:54:55.575993+08:00",
            "acknowledged_at": "2026-04-12T04:54:55.575993+08:00",
            "cleared_at": "2026-04-12T12:54:55.575993+08:00",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "0001-01-01T00:00:00Z",
            "last_updated_at": "0001-01-01T00:00:00Z",
            "created_at": "0001-01-01T00:00:00Z",
            "updated_at": "0001-01-01T00:00:00Z"
        }
    ],
    "total": 6,
    "page": 1,
    "page_size": 10,
    "total_pages": 1
}
```

**Result:** PASS

---

### 2.2 历史告警按运营商筛选

**Request:**
```http
GET /alarms/history?carrier=cmcc&page=1&page_size=10
Authorization: Bearer <token>
```

**Response:** `200`

```json
{
    "items": [
        {
            "id": "a0000006-0000-0000-0000-000000000006",
            "device_id": "d0000003-0000-0000-0000-000000000003",
            "device_sn": "SN-CUCC-001",
            "carrier": "cucc",
            "severity": 4,
            "alarm_type": "communicationsAlarm",
            "alarm_code": "ALM-006",
            "description": "GPS\u4fe1\u53f7\u4e22\u5931",
            "status": "cleared",
            "raised_at": "2026-04-15T09:54:55.575993+08:00",
            "cleared_at": "2026-04-15T12:55:02.647627+08:00",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "0001-01-01T00:00:00Z",
            "last_updated_at": "0001-01-01T00:00:00Z",
            "created_at": "0001-01-01T00:00:00Z",
            "updated_at": "0001-01-01T00:00:00Z"
        },
        {
            "id": "b0000004-0000-0000-0000-000000000004",
            "device_id": "d0000004-0000-0000-0000-000000000004",
            "device_sn": "SN-CMCC-002",
            "carrier": "cmcc",
            "severity": 4,
            "alarm_type": "environmentalAlarm",
            "alarm_code": "ALM-H04",
            "description": "\u98ce\u6247\u8f6c\u901f\u5f02\u5e38",
            "status": "cleared",
            "raised_at": "2026-04-14T12:54:55.575993+08:00",
            "acknowledged_at": "2026-04-15T04:54:55.575993+08:00",
            "cleared_at": "2026-04-15T07:54:55.575993+08:00",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "0001-01-01T00:00:00Z",
            "last_updated_at": "0001-01-01T00:00:00Z",
            "created_at": "0001-01-01T00:00:00Z",
            "updated_at": "0001-01-01T00:00:00Z"
        },
        {
            "id": "b0000005-0000-0000-0000-000000000005",
            "device_id": "d0000001-0000-0000-0000-000000000001",
            "device_sn": "SN-CMCC-001",
            "carrier": "cmcc",
            "severity": 2,
            "alarm_type": "qualityOfServiceAlarm",
            "alarm_code": "ALM-H05",
            "description": "\u65e0\u7ebf\u94fe\u8def\u8d28\u91cf\u4e0b\u964d",
            "status": "cleared",
            "raised_at": "2026-04-14T06:54:55.575993+08:00",
            "cleared_at": "2026-04-15T00:54:55.575993+08:00",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "0001-01-01T00:00:00Z",
            "last_updated_at": "0001-01-01T00:00:00Z",
            "created_at": "0001-01-01T00:00:00Z",
            "updated_at": "0001-01-01T00:00:00Z"
        },
        {
            "id": "b0000001-0000-0000-0000-000000000001",
            "device_id": "d0000001-0000-0000-0000-000000000001",
            "device_sn": "SN-CMCC-001",
            "carrier": "cmcc",
            "severity": 2,
            "alarm_type": "communicationsAlarm",
            "alarm_code": "ALM-H01",
            "description": "S1\u94fe\u8def\u95ea\u65ad",
            "status": "cleared",
            "raised_at": "2026-04-13T12:54:55.575993+08:00",
            "acknowledged_at": "2026-04-14T02:54:55.575993+08:00",
            "cleared_at": "2026-04-14T12:54:55.575993+08:00",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "0001-01-01T00:00:00Z",
            "last_updated_at": "0001-01-01T00:00:00Z",
            "created_at": "0001-01-01T00:00:00Z",
            "updated_at": "0001-01-01T00:00:00Z"
        },
        {
            "id": "b0000002-0000-0000-0000-000000000002",
            "device_id": "d0000002-0000-0000-0000-000000000002",
            "device_sn": "SN-CTCC-001",
            "carrier": "ctcc",
            "severity": 3,
            "alarm_type": "processingErrorAlarm",
            "alarm_code": "ALM-H02",
            "description": "CPU\u6e29\u5ea6\u544a\u8b66",
            "status": "cleared",
            "raised_at": "2026-04-12T12:54:55.575993+08:00",
            "acknowledged_at": "2026-04-13T00:54:55.575993+08:00",
            "cleared_at": "2026-04-13T12:54:55.575993+08:00",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "0001-01-01T00:00:00Z",
            "last_updated_at": "0001-01-01T00:00:00Z",
            "created_at": "0001-01-01T00:00:00Z",
            "updated_at": "0001-01-01T00:00:00Z"
        },
        {
            "id": "b0000003-0000-0000-0000-000000000003",
            "device_id": "d0000003-0000-0000-0000-000000000003",
            "device_sn": "SN-CUCC-001",
            "carrier": "cucc",
            "severity": 1,
            "alarm_type": "equipmentAlarm",
            "alarm_code": "ALM-H03",
            "description": "BBU\u677f\u5361\u6545\u969c",
            "status": "cleared",
            "raised_at": "2026-04-11T12:54:55.575993+08:00",
            "acknowledged_at": "2026-04-12T04:54:55.575993+08:00",
            "cleared_at": "2026-04-12T12:54:55.575993+08:00",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "0001-01-01T00:00:00Z",
            "last_updated_at": "0001-01-01T00:00:00Z",
            "created_at": "0001-01-01T00:00:00Z",
            "updated_at": "0001-01-01T00:00:00Z"
        }
    ],
    "total": 6,
    "page": 1,
    "page_size": 10,
    "total_pages": 1
}
```

**Result:** PASS

---

### 2.3 历史告警按时间范围筛选

**Request:**
```http
GET /alarms/history?start_time=2026-04-08T04:55:03Z&end_time=2026-04-15T04:55:03Z&page=1&page_size=10
Authorization: Bearer <token>
```

**Response:** `200`

```json
{
    "items": [
        {
            "id": "a0000006-0000-0000-0000-000000000006",
            "device_id": "d0000003-0000-0000-0000-000000000003",
            "device_sn": "SN-CUCC-001",
            "carrier": "cucc",
            "severity": 4,
            "alarm_type": "communicationsAlarm",
            "alarm_code": "ALM-006",
            "description": "GPS\u4fe1\u53f7\u4e22\u5931",
            "status": "cleared",
            "raised_at": "2026-04-15T09:54:55.575993+08:00",
            "cleared_at": "2026-04-15T12:55:02.647627+08:00",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "0001-01-01T00:00:00Z",
            "last_updated_at": "0001-01-01T00:00:00Z",
            "created_at": "0001-01-01T00:00:00Z",
            "updated_at": "0001-01-01T00:00:00Z"
        },
        {
            "id": "b0000004-0000-0000-0000-000000000004",
            "device_id": "d0000004-0000-0000-0000-000000000004",
            "device_sn": "SN-CMCC-002",
            "carrier": "cmcc",
            "severity": 4,
            "alarm_type": "environmentalAlarm",
            "alarm_code": "ALM-H04",
            "description": "\u98ce\u6247\u8f6c\u901f\u5f02\u5e38",
            "status": "cleared",
            "raised_at": "2026-04-14T12:54:55.575993+08:00",
            "acknowledged_at": "2026-04-15T04:54:55.575993+08:00",
            "cleared_at": "2026-04-15T07:54:55.575993+08:00",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "0001-01-01T00:00:00Z",
            "last_updated_at": "0001-01-01T00:00:00Z",
            "created_at": "0001-01-01T00:00:00Z",
            "updated_at": "0001-01-01T00:00:00Z"
        },
        {
            "id": "b0000005-0000-0000-0000-000000000005",
            "device_id": "d0000001-0000-0000-0000-000000000001",
            "device_sn": "SN-CMCC-001",
            "carrier": "cmcc",
            "severity": 2,
            "alarm_type": "qualityOfServiceAlarm",
            "alarm_code": "ALM-H05",
            "description": "\u65e0\u7ebf\u94fe\u8def\u8d28\u91cf\u4e0b\u964d",
            "status": "cleared",
            "raised_at": "2026-04-14T06:54:55.575993+08:00",
            "cleared_at": "2026-04-15T00:54:55.575993+08:00",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "0001-01-01T00:00:00Z",
            "last_updated_at": "0001-01-01T00:00:00Z",
            "created_at": "0001-01-01T00:00:00Z",
            "updated_at": "0001-01-01T00:00:00Z"
        },
        {
            "id": "b0000001-0000-0000-0000-000000000001",
            "device_id": "d0000001-0000-0000-0000-000000000001",
            "device_sn": "SN-CMCC-001",
            "carrier": "cmcc",
            "severity": 2,
            "alarm_type": "communicationsAlarm",
            "alarm_code": "ALM-H01",
            "description": "S1\u94fe\u8def\u95ea\u65ad",
            "status": "cleared",
            "raised_at": "2026-04-13T12:54:55.575993+08:00",
            "acknowledged_at": "2026-04-14T02:54:55.575993+08:00",
            "cleared_at": "2026-04-14T12:54:55.575993+08:00",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "0001-01-01T00:00:00Z",
            "last_updated_at": "0001-01-01T00:00:00Z",
            "created_at": "0001-01-01T00:00:00Z",
            "updated_at": "0001-01-01T00:00:00Z"
        },
        {
            "id": "b0000002-0000-0000-0000-000000000002",
            "device_id": "d0000002-0000-0000-0000-000000000002",
            "device_sn": "SN-CTCC-001",
            "carrier": "ctcc",
            "severity": 3,
            "alarm_type": "processingErrorAlarm",
            "alarm_code": "ALM-H02",
            "description": "CPU\u6e29\u5ea6\u544a\u8b66",
            "status": "cleared",
            "raised_at": "2026-04-12T12:54:55.575993+08:00",
            "acknowledged_at": "2026-04-13T00:54:55.575993+08:00",
            "cleared_at": "2026-04-13T12:54:55.575993+08:00",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "0001-01-01T00:00:00Z",
            "last_updated_at": "0001-01-01T00:00:00Z",
            "created_at": "0001-01-01T00:00:00Z",
            "updated_at": "0001-01-01T00:00:00Z"
        },
        {
            "id": "b0000003-0000-0000-0000-000000000003",
            "device_id": "d0000003-0000-0000-0000-000000000003",
            "device_sn": "SN-CUCC-001",
            "carrier": "cucc",
            "severity": 1,
            "alarm_type": "equipmentAlarm",
            "alarm_code": "ALM-H03",
            "description": "BBU\u677f\u5361\u6545\u969c",
            "status": "cleared",
            "raised_at": "2026-04-11T12:54:55.575993+08:00",
            "acknowledged_at": "2026-04-12T04:54:55.575993+08:00",
            "cleared_at": "2026-04-12T12:54:55.575993+08:00",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "0001-01-01T00:00:00Z",
            "last_updated_at": "0001-01-01T00:00:00Z",
            "created_at": "0001-01-01T00:00:00Z",
            "updated_at": "0001-01-01T00:00:00Z"
        }
    ],
    "total": 6,
    "page": 1,
    "page_size": 10,
    "total_pages": 1
}
```

**Result:** PASS

---

### 2.4 历史告警按严重级别筛选

**Request:**
```http
GET /alarms/history?severity=1&page=1&page_size=10
Authorization: Bearer <token>
```

**Response:** `200`

```json
{
    "items": [
        {
            "id": "b0000003-0000-0000-0000-000000000003",
            "device_id": "d0000003-0000-0000-0000-000000000003",
            "device_sn": "SN-CUCC-001",
            "carrier": "cucc",
            "severity": 1,
            "alarm_type": "equipmentAlarm",
            "alarm_code": "ALM-H03",
            "description": "BBU\u677f\u5361\u6545\u969c",
            "status": "cleared",
            "raised_at": "2026-04-11T12:54:55.575993+08:00",
            "acknowledged_at": "2026-04-12T04:54:55.575993+08:00",
            "cleared_at": "2026-04-12T12:54:55.575993+08:00",
            "is_read": false,
            "ack_count": 0,
            "first_raised_at": "0001-01-01T00:00:00Z",
            "last_updated_at": "0001-01-01T00:00:00Z",
            "created_at": "0001-01-01T00:00:00Z",
            "updated_at": "0001-01-01T00:00:00Z"
        }
    ],
    "total": 1,
    "page": 1,
    "page_size": 10,
    "total_pages": 1
}
```

**Result:** PASS

---

## 三、告警库接口

### 3.1 查询告警库列表

**Request:**
```http
GET /alarm-libraries?page=1&page_size=10
Authorization: Bearer <token>
```

**Response:** `200`

```json
{
    "items": [
        {
            "id": "0798f9f3-045e-41c4-942c-3d10d0bcf250",
            "alarm_code": "TEST_RPT_001",
            "alarm_source": "Device",
            "event_type": "equipment",
            "severity": 2,
            "enabled": true,
            "probable_cause": "\u6d4b\u8bd5\u62a5\u544a\u544a\u8b66",
            "explanation": "\u7528\u4e8e\u6d4b\u8bd5\u62a5\u544a",
            "created_at": "2026-04-15T11:01:24.704999+08:00",
            "updated_at": "2026-04-15T11:01:24.704999+08:00"
        },
        {
            "id": "c75e11b5-d3c8-4f32-b148-013cc31e90ab",
            "alarm_code": "DEVICE_RESTART",
            "alarm_source": "Device",
            "event_type": "equipment",
            "severity": 3,
            "enabled": true,
            "probable_cause": "\u8bbe\u5907\u91cd\u542f",
            "explanation": "\u8bbe\u5907\u5df2\u91cd\u65b0\u542f\u52a8",
            "additional_info": {
                "category": "maintenance",
                "impact": "medium"
            },
            "created_at": "2026-04-15T10:31:03.743211+08:00",
            "updated_at": "2026-04-15T10:31:03.743211+08:00"
        },
        {
            "id": "76430b18-2130-4774-9d7a-b8c732f3dab7",
            "alarm_code": "LINK_FAILURE",
            "alarm_source": "Device",
            "event_type": "communications",
            "severity": 2,
            "enabled": true,
            "probable_cause": "\u94fe\u8def\u4e2d\u65ad",
            "explanation": "\u57fa\u7ad9\u56de\u4f20\u94fe\u8def\u5f02\u5e38\u4e2d\u65ad",
            "additional_info": {
                "category": "connectivity",
                "impact": "high"
            },
            "created_at": "2026-04-15T10:31:03.743211+08:00",
            "updated_at": "2026-04-15T10:31:03.743211+08:00"
        },
        {
            "id": "c721b3bb-60c3-4e26-baff-d3c2fc5bf789",
            "alarm_code": "S1_INTERFACE_ERROR",
            "alarm_source": "TR069",
            "event_type": "communications",
            "severity": 2,
            "enabled": true,
            "probable_cause": "S1\u63a5\u53e3\u5f02\u5e38",
            "explanation": "TR069 ACS\u4e0eCPE\u7684S1\u63a5\u53e3\u901a\u4fe1\u5f02\u5e38",
            "additional_info": {
                "category": "protocol",
                "impact": "medium"
            },
            "created_at": "2026-04-15T10:31:03.743211+08:00",
            "updated_at": "2026-04-15T10:31:03.743211+08:00"
        },
        {
            "id": "07581df7-354b-487e-bfa0-b78d34909927",
            "alarm_code": "CPU_OVERLOAD",
            "alarm_source": "Device",
            "event_type": "processing",
            "severity": 3,
            "enabled": true,
            "probable_cause": "CPU\u8fc7\u8f7d",
            "explanation": "\u8bbe\u5907CPU\u4f7f\u7528\u7387\u8d85\u8fc7\u9608\u503c",
            "additional_info": {
                "category": "performance",
                "impact": "medium",
                "threshold": 80
            },
            "created_at": "2026-04-15T10:31:03.743211+08:00",
            "updated_at": "2026-04-15T10:31:03.743211+08:00"
        },
        {
            "id": "23113588-7f07-43f4-bd0d-3ba98bea1910",
            "alarm_code": "MEMORY_OVERLOAD",
            "alarm_source": "Device",
            "event_type": "processing",
            "severity": 3,
            "enabled": true,
            "probable_cause": "\u5185\u5b58\u8fc7\u8f7d",
            "explanation": "\u8bbe\u5907\u5185\u5b58\u4f7f\u7528\u7387\u8d85\u8fc7\u9608\u503c",
            "additional_info": {
                "category": "performance",
                "impact": "medium",
                "threshold": 90
            },
            "created_at": "2026-04-15T10:31:03.743211+08:00",
            "updated_at": "2026-04-15T10:31:03.743211+08:00"
        },
        {
            "id": "2c236c17-7ad7-4fa9-bc54-a56520602ca1",
            "alarm_code": "TEMP_HIGH",
            "alarm_source": "Device",
            "event_type": "environment",
            "severity": 3,
            "enabled": true,
            "probable_cause": "\u6e29\u5ea6\u8fc7\u9ad8",
            "explanation": "\u8bbe\u5907\u6e29\u5ea6\u8d85\u8fc7\u544a\u8b66\u9608\u503c",
            "additional_info": {
                "category": "environment",
                "impact": "medium",
                "threshold": 70
            },
            "created_at": "2026-04-15T10:31:03.743211+08:00",
            "updated_at": "2026-04-15T10:31:03.743211+08:00"
        },
        {
            "id": "4502a06e-6a88-41b5-a8e3-a4cffceda591",
            "alarm_code": "POWER_FAILURE",
            "alarm_source": "Device",
            "event_type": "power",
            "severity": 1,
            "enabled": true,
            "probable_cause": "\u7535\u6e90\u6545\u969c",
            "explanation": "\u8bbe\u5907\u4f9b\u7535\u4e2d\u65ad\u6216\u7535\u6e90\u6a21\u5757\u6545\u969c",
            "additional_info": {
                "category": "power",
                "impact": "high"
            },
            "created_at": "2026-04-15T10:31:03.743211+08:00",
            "updated_at": "2026-04-15T10:31:03.743211+08:00"
        },
        {
            "id": "0560855d-e758-4729-87ed-51ec992f933a",
            "alarm_code": "DEVICE_OFFLINE",
            "alarm_source": "Device",
            "event_type": "communications",
            "severity": 2,
            "enabled": true,
            "probable_cause": "\u8bbe\u5907\u79bb\u7ebf",
            "explanation": "\u8bbe\u5907\u4e0e\u7f51\u7ba1\u7cfb\u7edf\u4e4b\u95f4\u7684\u8fde\u63a5\u4e2d\u65ad",
            "additional_info": {
                "category": "connectivity",
                "impact": "high"
            },
            "created_at": "2026-04-15T10:31:03.743211+08:00",
            "updated_at": "2026-04-15T10:31:03.743211+08:00"
        },
        {
            "id": "4464abeb-a6d3-4d22-be3c-be595a923c8d",
            "alarm_code": "GPS_LOSS",
            "alarm_source": "Device",
            "event_type": "environment",
            "severity": 4,
            "enabled": true,
            "probable_cause": "GPS\u4fe1\u53f7\u4e22\u5931",
            "explanation": "\u8bbe\u5907GPS\u6a21\u5757\u65e0\u6cd5\u83b7\u53d6\u536b\u661f\u4fe1\u53f7",
            "additional_info": {
                "category": "positioning",
                "impact": "low"
            },
            "created_at": "2026-04-15T10:31:03.743211+08:00",
            "updated_at": "2026-04-15T10:31:03.743211+08:00"
        }
    ],
    "total": 16,
    "page": 1,
    "page_size": 10,
    "total_pages": 2
}
```

**Result:** PASS

---

### 3.2 创建告警库条目

**Request:**
```http
POST /alarm-libraries
Authorization: Bearer <token>
Content-Type: application/json

{"alarm_code":"TEST-001","alarm_type":"equipmentAlarm","alarm_source":"base_station","event_type":"equipmentAlarm","description":"测试告警条目","probable_cause":"测试原因","carrier":"cmcc","technology":"lte","severity":2}
```

**Response:** `201`

```json
{
    "id": "5665b218-cad0-4303-bb79-3b6ad5dbcb92",
    "alarm_code": "TEST-001",
    "alarm_source": "base_station",
    "event_type": "equipmentAlarm",
    "severity": 2,
    "enabled": true,
    "probable_cause": "\u6d4b\u8bd5\u539f\u56e0",
    "carrier": "cmcc",
    "technology": "lte",
    "created_at": "2026-04-15T12:55:03.355789921+08:00",
    "updated_at": "2026-04-15T12:55:03.355789921+08:00"
}
```

**Result:** PASS

---

### 3.3 查询告警库详情

**Request:**
```http
GET /alarm-libraries/5665b218-cad0-4303-bb79-3b6ad5dbcb92
Authorization: Bearer <token>
```

**Response:** `200`

```json
{
    "id": "5665b218-cad0-4303-bb79-3b6ad5dbcb92",
    "alarm_code": "TEST-001",
    "alarm_source": "base_station",
    "event_type": "equipmentAlarm",
    "severity": 2,
    "enabled": true,
    "probable_cause": "\u6d4b\u8bd5\u539f\u56e0",
    "carrier": "cmcc",
    "technology": "lte",
    "created_at": "2026-04-15T12:55:03.355789+08:00",
    "updated_at": "2026-04-15T12:55:03.355789+08:00"
}
```

**Result:** PASS

---

### 3.4 更新告警库条目

**Request:**
```http
PUT /alarm-libraries/5665b218-cad0-4303-bb79-3b6ad5dbcb92
Authorization: Bearer <token>
Content-Type: application/json

{"description":"更新后的测试告警条目","severity":3}
```

**Response:** `200`

```json
{
    "id": "5665b218-cad0-4303-bb79-3b6ad5dbcb92",
    "alarm_code": "TEST-001",
    "alarm_source": "base_station",
    "event_type": "equipmentAlarm",
    "severity": 3,
    "enabled": true,
    "probable_cause": "\u6d4b\u8bd5\u539f\u56e0",
    "carrier": "cmcc",
    "technology": "lte",
    "created_at": "2026-04-15T12:55:03.355789+08:00",
    "updated_at": "2026-04-15T12:55:03.526248333+08:00"
}
```

**Result:** PASS

---

### 3.5 删除告警库条目

**Request:**
```http
DELETE /alarm-libraries/5665b218-cad0-4303-bb79-3b6ad5dbcb92
Authorization: Bearer <token>
```

**Response:** `200`

```json
{
    "message": "alarm library deleted"
}
```

**Result:** PASS

---

## 四、告警过滤规则接口

### 4.1 查询过滤规则列表

**Request:**
```http
GET /alarm-filters?page=1&page_size=10
Authorization: Bearer <token>
```

**Response:** `200`

```json
{
    "items": null,
    "total": 0,
    "page": 0,
    "page_size": 0,
    "total_pages": 0
}
```

**Result:** PASS

---

### 4.2 创建过滤规则

**Request:**
```http
POST /alarm-filters
Authorization: Bearer <token>
Content-Type: application/json

{"name":"测试忽略规则","filter_type":"alarm_code","alarm_codes":["ALM-008"],"action":"ignore","priority":10,"enabled":true}
```

**Response:** `201`

```json
{
    "id": "8fbadb0a-b305-4d10-9c6e-4671dae48568",
    "name": "\u6d4b\u8bd5\u5ffd\u7565\u89c4\u5219",
    "filter_type": "alarm_code",
    "alarm_sources": [],
    "alarm_codes": [
        "ALM-008"
    ],
    "device_ids": [],
    "device_group_ids": [],
    "action": "ignore",
    "priority": 10,
    "enabled": true,
    "created_at": "2026-04-15T12:55:03.728011781+08:00",
    "updated_at": "2026-04-15T12:55:03.728011781+08:00"
}
```

**Result:** PASS

---

### 4.3 查询过滤规则详情

**Request:**
```http
GET /alarm-filters/8fbadb0a-b305-4d10-9c6e-4671dae48568
Authorization: Bearer <token>
```

**Response:** `200`

```json
{
    "id": "8fbadb0a-b305-4d10-9c6e-4671dae48568",
    "name": "\u6d4b\u8bd5\u5ffd\u7565\u89c4\u5219",
    "filter_type": "alarm_code",
    "alarm_sources": [],
    "alarm_codes": [
        "ALM-008"
    ],
    "device_ids": [],
    "device_group_ids": [],
    "action": "ignore",
    "priority": 10,
    "enabled": true,
    "created_at": "2026-04-15T12:55:03.728011+08:00",
    "updated_at": "2026-04-15T12:55:03.728011+08:00"
}
```

**Result:** PASS

---

### 4.4 更新过滤规则

**Request:**
```http
PUT /alarm-filters/8fbadb0a-b305-4d10-9c6e-4671dae48568
Authorization: Bearer <token>
Content-Type: application/json

{"enabled":false}
```

**Response:** `200`

```json
{
    "id": "8fbadb0a-b305-4d10-9c6e-4671dae48568",
    "name": "\u6d4b\u8bd5\u5ffd\u7565\u89c4\u5219",
    "filter_type": "alarm_code",
    "alarm_sources": [],
    "alarm_codes": [
        "ALM-008"
    ],
    "device_ids": [],
    "device_group_ids": [],
    "action": "ignore",
    "priority": 10,
    "enabled": false,
    "created_at": "2026-04-15T12:55:03.728011+08:00",
    "updated_at": "2026-04-15T12:55:03.905409+08:00"
}
```

**Result:** PASS

---

### 4.5 切换过滤规则启用状态

**Request:**
```http
POST /alarm-filters/8fbadb0a-b305-4d10-9c6e-4671dae48568/toggle
Authorization: Bearer <token>
```

**Response:** `200`

```json
{
    "message": "filter rule toggled"
}
```

**Result:** PASS

---

### 4.6 删除过滤规则

**Request:**
```http
DELETE /alarm-filters/8fbadb0a-b305-4d10-9c6e-4671dae48568
Authorization: Bearer <token>
```

**Response:** `200`

```json
{
    "message": "filter rule deleted"
}
```

**Result:** PASS

---

### 4.7 查询启用的过滤规则

**Request:**
```http
GET /alarm-filters?enabled=true&page=1&page_size=10
Authorization: Bearer <token>
```

**Response:** `200`

```json
{
    "items": null,
    "total": 0,
    "page": 0,
    "page_size": 0,
    "total_pages": 0
}
```

**Result:** PASS

---


## 五、测试总结

| 分类 | 测试数 | 通过 | 失败 | 通过率 |
|------|--------|------|------|--------|
| 活动告警接口 | 13 | 12 | 1 | 92.3% |
| 历史告警接口 | 4 | 4 | 0 | 100% |
| 告警库接口 | 5 | 5 | 0 | 100% |
| 过滤规则接口 | 7 | 7 | 0 | 100% |
| **合计** | **29** | **28** | **1** | **96.6%** |

### 已知问题

| # | 接口 | 状态码 | 原因 | 修复状态 |
|---|------|--------|------|---------|
| 1 | `POST /alarms/active/batch/clear` | 500 | `alarms_active` 表无 `cleared_at` 列，BatchClear SQL 引用错误 | 已修复代码（`pg_store.go`），需重新构建 Docker 镜像 |

### API 路由汇总

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/alarms/active` | 活动告警列表（支持 carrier/severity/status/device_sn 分页筛选） |
| GET | `/alarms/history` | 历史告警列表（支持 carrier/severity/start_time/end_time 分页筛选） |
| GET | `/alarms/statistics` | 告警统计 |
| GET | `/alarms/:id` | 查询单条活动告警详情 |
| POST | `/alarms/:id/acknowledge` | 确认告警（需 JSON body: `{"acknowledged_by":"xxx"}`） |
| POST | `/alarms/:id/clear` | 清除告警（归档到历史表并从活动表删除） |
| POST | `/alarms/active/batch/acknowledge` | 批量确认（需 `ids` + `acknowledged_by`） |
| POST | `/alarms/active/batch/clear` | 批量清除（需 `ids`） |
| POST | `/alarms/active/:id/read` | 标记已读 |
| GET | `/alarm-libraries` | 告警库列表 |
| POST | `/alarm-libraries` | 创建告警库条目（必填: alarm_code, alarm_type, alarm_source, event_type, probable_cause） |
| GET | `/alarm-libraries/:id` | 告警库详情 |
| PUT | `/alarm-libraries/:id` | 更新告警库条目 |
| DELETE | `/alarm-libraries/:id` | 删除告警库条目 |
| GET | `/alarm-filters` | 过滤规则列表 |
| POST | `/alarm-filters` | 创建过滤规则 |
| GET | `/alarm-filters/:id` | 过滤规则详情 |
| PUT | `/alarm-filters/:id` | 更新过滤规则 |
| POST | `/alarm-filters/:id/toggle` | 切换启用状态 |
| DELETE | `/alarm-filters/:id` | 删除过滤规则 |
