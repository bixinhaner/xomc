# 系统日志管理功能设计文档

## 1. 页面概述

系统日志管理页面用于查看和管理OMC系统的各类日志信息，包括操作日志、安全日志、系统日志和北向接口日志。通过页签切换查看不同类型的日志记录。

## 2. 页面结构

页面采用 Tab 页签结构，根据用户权限显示不同的页签：

| 页签名称 | 权限码 | 显示条件 |
|----------|--------|----------|
| 操作日志 | CODE_SYSTEM_LOGS_OPERATION | 有权限时显示 |
| 安全日志 | CODE_SYSTEM_LOGS_SECURITY | 有权限且非云核心环境 |
| 系统日志 | CODE_SYSTEM_LOGS_SYSTEM | 有权限且为管理员 |
| 北向接口日志 | CODE_SYSTEM_LOGS_NORTH | 有权限时显示 |

## 3. 全局功能

### 3.1 工具栏按钮

| 按钮名称 | 图标 | 说明 |
|----------|------|------|
| 导出 | el-icon-circle-export | 导出当前页签日志数据为CSV文件 |
| 查看旧版本 | el-icon-circle-oldData | 查看历史版本日志（有旧数据时显示） |

---

## 4. 操作日志页签

### 4.1 搜索条件

| 字段名 | 字段标识 | 类型 | 说明 |
|--------|----------|------|------|
| 用户名称 | userCode | 下拉选择 | 从用户列表选择 |
| IP地址 | operateIp | 输入框 | 支持模糊搜索 |
| 日志名称 | logName | 下拉选择 | 从操作日志名称列表选择 |
| 详细记录 | detail | 输入框 | 支持模糊搜索 |
| 结果 | result | 下拉选择 | 所有/成功(1)/失败(0) |
| 失败原因 | reason | 输入框 | 支持模糊搜索 |
| 时间 | startTime/endTime | 日期时间范围 | 开始时间-结束时间 |

**快捷搜索占位符**：用户名称 / IP地址 / 日志名称

### 4.2 列表字段

| 字段名 | 字段标识 | 宽度 | 说明 |
|--------|----------|------|------|
| 序号 | - | 自动 | 行号 |
| 用户名称 | user_code | 150px | 操作用户名 |
| IP地址 | operate_ip | 200px | 操作来源IP |
| 日志名称 | log_name | 200px | 操作类型名称 |
| 详细记录 | detail | min 300px | 操作详情，溢出显示tooltip |
| 结果 | result | 100px | 成功(1)/失败(0) |
| 失败原因 | reason | min 200px | 失败时的原因，溢出显示tooltip |
| 操作开始时间 | op_start_time | 150px | 操作开始时间 |
| 操作结束时间 | op_end_time | 150px | 操作结束时间 |

### 4.3 API 接口

| 功能 | 接口地址 | 请求方式 |
|------|----------|----------|
| 获取列表 | /system/operatorLog/getOperationLogPageList.action | POST |
| 获取用户下拉 | /system/operatorLog/getUserListForCombobox.action | POST |
| 获取日志名称下拉 | /system/operatorLog/getOperationNameForCombobox.action | POST |
| 导出CSV | /system/operatorLog/exportLogToCsvFile.action | POST |

### 4.4 请求参数

```json
{
  "timeZone": "时区",
  "logName": "日志名称",
  "userCode": "用户名",
  "operateIp": "IP地址",
  "detail": "详细记录",
  "result": "结果",
  "reason": "失败原因",
  "startTime": "开始时间",
  "endTime": "结束时间",
  "searchText": "快捷搜索文本"
}
```

---

## 5. 安全日志页签

### 5.1 搜索条件

| 字段名 | 字段标识 | 类型 | 说明 |
|--------|----------|------|------|
| ID | id | 输入框 | 日志ID |
| 用户名称 | userCode | 下拉选择 | 从用户列表选择 |
| IP地址 | operateIp | 输入框 | 支持模糊搜索 |
| 日志名称 | logName | 下拉选择 | 从安全日志名称列表选择 |
| 详细记录 | detail | 输入框 | 支持模糊搜索 |
| 结果 | result | 下拉选择 | 所有/成功(1)/失败(0) |
| 失败原因 | reason | 输入框 | 支持模糊搜索 |
| 时间 | startTime/endTime | 日期时间范围 | 开始时间-结束时间 |

**快捷搜索占位符**：ID / 用户名称 / IP地址 / 日志名称

### 5.2 列表字段

| 字段名 | 字段标识 | 宽度 | 说明 |
|--------|----------|------|------|
| 序号 | - | 自动 | 行号 |
| ID | id | 80px | 日志ID |
| 用户名称 | user_code | 150px | 操作用户名 |
| IP地址 | operate_ip | 200px | 操作来源IP |
| 日志名称 | log_name | 200px | 安全操作类型名称 |
| 详细记录 | detail | min 300px | 操作详情，溢出显示tooltip |
| 结果 | result | 100px | 成功(1)/失败(0) |
| 失败原因 | reason | min 200px | 失败时的原因，溢出显示tooltip |
| 时间 | time | 150px | 操作时间 |

### 5.3 API 接口

| 功能 | 接口地址 | 请求方式 |
|------|----------|----------|
| 获取列表 | /system/securityLog/getSecurityLogPageList.action | POST |
| 获取用户下拉 | /system/securityLog/getUserListForCombobox.action | POST |
| 获取日志名称下拉 | /system/securityLog/getOperationNameForCombobox.action | POST |
| 导出CSV | /system/securityLog/exportLogToCsvFile.action | POST |

### 5.4 请求参数

```json
{
  "timeZone": "时区",
  "id": "日志ID",
  "logName": "日志名称",
  "userCode": "用户名",
  "operateIp": "IP地址",
  "detail": "详细记录",
  "result": "结果",
  "reason": "失败原因",
  "startTime": "开始时间",
  "endTime": "结束时间",
  "searchText": "快捷搜索文本"
}
```

### 5.5 显示条件

- 有 `CODE_SYSTEM_LOGS_SECURITY` 权限
- 非云核心环境（isCloudCore == 'false'）

---

## 6. 系统日志页签

### 6.1 搜索条件

| 字段名 | 字段标识 | 类型 | 说明 |
|--------|----------|------|------|
| ID | id | 输入框 | 日志ID |
| 日志名称 | logName | 下拉选择 | 从系统日志名称列表选择 |
| 详细记录 | detail | 输入框 | 支持模糊搜索 |
| 结果 | result | 下拉选择 | 所有/成功(1)/失败(0) |
| 失败原因 | reason | 输入框 | 支持模糊搜索 |
| 时间 | startTime/endTime | 日期时间范围 | 开始时间-结束时间 |

**快捷搜索占位符**：ID / 日志名称

### 6.2 列表字段

| 字段名 | 字段标识 | 宽度 | 说明 |
|--------|----------|------|------|
| 序号 | - | 自动 | 行号 |
| ID | id | 80px | 日志ID |
| 日志名称 | log_name | 200px | 系统操作类型名称 |
| 详细记录 | detail | min 300px | 操作详情，溢出显示tooltip |
| 结果 | result | 100px | 成功(1)/失败(0) |
| 失败原因 | reason | min 200px | 失败时的原因，溢出显示tooltip |
| 时间 | time | 150px | 操作时间 |

### 6.3 API 接口

| 功能 | 接口地址 | 请求方式 |
|------|----------|----------|
| 获取列表 | /system/systemLog/getSystemLogPageList.action | POST |
| 获取日志名称下拉 | /system/systemLog/getOperationNameForCombobox.action | POST |
| 导出CSV | /system/systemLog/exportLogToCsvFile.action | POST |

### 6.4 请求参数

```json
{
  "timeZone": "时区",
  "id": "日志ID",
  "logName": "日志名称",
  "detail": "详细记录",
  "result": "结果",
  "reason": "失败原因",
  "startTime": "开始时间",
  "endTime": "结束时间",
  "searchText": "快捷搜索文本"
}
```

### 6.5 显示条件

- 有 `CODE_SYSTEM_LOGS_SYSTEM` 权限
- 为管理员用户（isAdmin == 'true'）

---

## 7. 北向接口日志页签

### 7.1 搜索条件

| 字段名 | 字段标识 | 类型 | 说明 |
|--------|----------|------|------|
| IP地址 | ipAddress | 输入框 | 支持模糊搜索 |
| 日志名称 | name | 输入框 | 支持模糊搜索 |
| 北向接口类型 | type | 下拉选择 | 见类型选项 |
| 时间 | startTime/endTime | 日期时间范围 | 开始时间-结束时间 |

**快捷搜索占位符**：IP地址

### 7.2 北向接口类型选项

| 值 | 显示文本 |
|----|----------|
| 1 | 同步请求 |
| 2 | 异步请求 |
| Real Alarm | 告警 |
| Sync Msg | 同步消息 |
| Login | 登录 |
| Sync File | 同步文件 |
| Disconnection | 非连接 |
| Connection | 连接 |
| Connection Timeout | 连接超时 |
| Idle Timeout | 空闲超时 |
| HEARTBEAT | 心跳 |

### 7.3 列表字段

| 字段名 | 字段标识 | 宽度 | 说明 |
|--------|----------|------|------|
| 序号 | - | 自动 | 行号 |
| 日志名称 | nameCn/nameEn | 300px | 中文名/英文名（根据语言环境） |
| IP地址 | ipAddress | 200px | 请求来源IP |
| 北向接口类型 | type | 150px | 见类型选项 |
| 北向接口请求 | reqParams | 250px | 请求参数 |
| 北向接口返回 | status | 自适应 | 异步请求显示resParams2，其他显示resParams1 |
| 北向接口响应时间 | createTime | 200px | 响应时间 |

### 7.4 API 接口

| 功能 | 接口地址 | 请求方式 |
|------|----------|----------|
| 获取列表 | /northboundApi/v1/log/page | POST |
| 导出CSV | /northboundApi/v1/log/exportLogToCsvFile | POST |

### 7.5 请求参数

```json
{
  "timeZone": "时区",
  "searchText": "快捷搜索文本",
  "type": "接口类型",
  "startTime": "开始时间",
  "endTime": "结束时间",
  "name": "日志名称",
  "ipAddress": "IP地址",
  "operatorCode": "操作员代码",
  "language": "语言"
}
```

---

## 8. 交互说明

### 8.1 查询功能

1. **快捷查询**：在搜索框输入关键字，按回车或点击查询按钮
2. **高级查询**：展开查询面板，填写具体条件后点击查询
3. **重置**：点击重置按钮清空所有查询条件

### 8.2 导出功能

1. 点击页面右上角"导出"按钮
2. 导出当前页签的日志数据为CSV文件
3. 导出数据包含当前查询条件筛选的结果

### 8.3 查看旧版本日志

1. 当存在旧版本日志数据时，显示"查看旧版本"按钮
2. 点击后在新页面查看历史日志数据
3. 点击关闭按钮返回主页面

### 8.4 页签切换

1. 点击页签名称切换不同类型的日志
2. 切换页签时自动关闭旧版本日志查看页面

---

## 9. 数据字典

### 9.1 结果状态

| 值 | 显示文本 | 颜色 |
|----|----------|------|
| 1 | 成功 | 绿色 |
| 0 | 失败 | 红色 |

### 9.2 权限码

| 权限码 | 说明 |
|--------|------|
| CODE_SYSTEM_LOGS_OPERATION | 操作日志查看权限 |
| CODE_SYSTEM_LOGS_SECURITY | 安全日志查看权限 |
| CODE_SYSTEM_LOGS_SYSTEM | 系统日志查看权限 |
| CODE_SYSTEM_LOGS_NORTH | 北向接口日志查看权限 |

---

## 10. 页面布局

```
┌─────────────────────────────────────────────────────────────┐
│  [导出]                            [查看旧版本]    [关闭]   │
├─────────────────────────────────────────────────────────────┤
│ [操作日志] [安全日志] [系统日志] [北向接口日志]              │
├─────────────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ [快捷搜索框] [查询] [重置]                              │ │
│ │ ─────────────────────────────────────────────────────── │ │
│ │ 展开条件：                                              │ │
│ │ 用户名称 [下拉]    IP地址 [输入]    日志名称 [下拉]    │ │
│ │ 详细记录 [输入]    结果 [下拉]      失败原因 [输入]    │ │
│ │ 时间 [开始时间] ~ [结束时间]                           │ │
│ └─────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────┤
│ 序号│用户名称│IP地址│日志名称│详细记录│结果│失败原因│时间   │
├─────┼────────┼──────┼────────┼────────┼────┼────────┼───────┤
│  1  │ admin  │192...│ 登录   │ 用户..│成功│   -    │10:00  │
│  2  │ user1  │192...│ 配置   │ 修改..│失败│ 权限.. │11:30  │
│ ... │        │      │        │        │    │        │       │
├─────────────────────────────────────────────────────────────┤
│                         [分页控件]                          │
└─────────────────────────────────────────────────────────────┘
```
