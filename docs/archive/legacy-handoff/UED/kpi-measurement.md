# KPI 测量维护页面

## 页面概述

**页面名称**: 测量维护 (KPI Measurement Maintenance)
**菜单路径**: 性能管理 > KPI管理 > 测量维护
**API前缀**: `/api/v1/pm/kpi-measurement`

---

## 搜索功能

### 模糊搜索
| 字段 | 说明 |
|------|------|
| searchText | 支持按**基站编码**或**基站名称**模糊搜索，回车触发搜索 |

---

## 高级筛选

### 1. 状态筛选 (status)
| 值 | 显示文本 |
|----|----------|
| '' | 全部 |
| '0' | 关 |
| '1' | 正常 |
| '2' | 损毁中止 |

### 2. 设置开关筛选 (measEnable)
| 值 | 显示文本 |
|----|----------|
| '' | 全部 |
| '1' | 启用 |
| '0' | 禁用 |

### 操作
- **清空筛选**: 重置所有筛选条件为默认值

---

## 表格字段

| 序号 | 字段名 | 显示名称 | 类型 | 宽度 | 可排序 | 说明 |
|------|--------|----------|------|------|--------|------|
| 1 | - | 选择框 | checkbox | 50px | - | 批量选择，支持跨页保留选择 (reserve-selection) |
| 2 | - | 文件 | icon按钮 | 60px | - | 点击打开测量文件页面 |
| 3 | report_enable | 是否启用 | switch开关 | 110px | 是 | 启用/禁用测量任务，值: "0"/"1" |
| 4 | status | 状态 | 图标+文本 | 120px | 是 | 见状态说明 |
| 5 | serialNumber | 小站编码 | 文本 | min-120px | 是 | 设备序列号 (row-key) |
| 6 | hostName | HostName | 文本 | min-120px | 是 | 主机名 |
| 7 | cellId | 基站ID | 文本 | min-120px | 是 | 基站标识 |
| 8 | reportPeriod | 测量周期(分钟) | 文本 | min-120px | - | PM上报周期 |
| 9 | startTime | 开始时间 | 日期时间 | min-120px | 是 | 测量开始时间 |
| 10 | updateTime | 更新时间 | 日期时间 | min-120px | 是 | 最后更新时间 |

### 状态显示说明
| 值 | 显示文本 | 图标 | 图标颜色 |
|----|----------|------|----------|
| '0' | 关 | el-icon-status-kpi-off | 灰色 |
| '1' | 正常 | el-icon-status-kpi-normal | 绿色 (greenIcon) |
| '2' | 损毁中止 | el-icon-status-kpi-failure | 红色 (redIcon) |

---

## 批量操作

### 工具栏按钮

| 按钮 | 图标 | 功能说明 | 触发条件 |
|------|------|----------|----------|
| 已选设备 | el-icon-selected | 显示已选设备数量 `(n)`，点击展开已选列表弹窗 | 始终显示 |
| 启用 | el-icon-KPI-Meas | 批量开启测量任务 | 至少选中1条 |
| 禁用 | el-icon-operation-CancelMeasure | 批量关闭测量任务 | 至少选中1条 |

### 已选设备弹窗
- 显示已选设备列表（小站编码 serialNumber）
- 支持单个删除已选设备（点击 X 图标）
- 支持清空全部已选（Clear 按钮）
- 分页显示（前端分页，height: 270px）
- 使用 `serialNumber` 作为唯一标识

### 启用/禁用确认弹窗
根据设备是否需要重启（needReboot字段），显示不同确认信息：

**启用时**:
- 需重启设备: "需要重启确认开启测量任务"
- 无需重启: "确定开启测量任务"

**禁用时**:
- 需重启设备: "需要重启确认关闭测量任务"
- 无需重启: "确认关闭测量任务"

---

## API 接口

### 1. 获取列表数据
```
GET /api/v1/pm/kpi-measurement
```

**Query 参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| searchText | string | 否 | 模糊搜索（基站编码/名称） |
| status | string | 否 | 状态筛选 (0/1/2) |
| measEnable | string | 否 | 启用状态筛选 (0/1) |
| timeZone | string | 是 | 时区 |
| page | int | 是 | 页码 |
| pageSize | int | 是 | 每页条数 |

**Response**:
```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "serialNumber": "string",      // 小站编码
        "hostName": "string",          // 主机名
        "cellId": "string",            // 基站ID
        "smallCellCode": "string",     // 小站代码（用于API调用）
        "reportEnable": "0" | "1",     // 是否启用测量
        "reportPeriod": 15,            // 测量周期(分钟)
        "status": "0" | "1" | "2",     // 状态
        "needReboot": "0" | "1",       // 是否需要重启
        "startTime": "2026-01-01 00:00:00",
        "updateTime": "2026-01-01 00:00:00"
      }
    ],
    "total": 100
  }
}
```

### 2. 设置测量开关（单个/批量）
```
POST /api/v1/pm/kpi-measurement/switch
```

**Request Body**:
```json
{
  "activeReport": "0" | "1",       // 0=禁用, 1=启用
  "smallCellCodes": "code1,code2"  // 逗号分隔的设备编码列表
}
```

**Response**:
```json
{
  "code": 0,
  "message": "操作成功"
}
```

---

## 权限控制

| 权限码 | 说明 |
|--------|------|
| CODE_PERFORMANCE_MEASUREMENT | 控制是否显示操作按钮（启用/禁用、选择框、开关） |

无写权限时：
- 隐藏工具栏按钮（已选设备、启用、禁用）
- 隐藏表格选择列
- 隐藏"是否启用"开关列

---

## 二级页面：测量文件页面

### 页面概述
- **触发方式**: 点击主表格行的"文件"图标（el-icon-operation-result）
- **展示方式**: 全屏滑出面板 (Slide Panel)，从顶部滑入
- **页面标题**: "文件 (SN: 设备序列号)"
- **关闭按钮**: 右上角 X 按钮

### 筛选功能

#### 日期范围筛选
| 字段 | 类型 | 说明 |
|------|------|------|
| startTime | 日期 | 开始时间 |
| endTime | 日期 | 结束时间 |
| 类型 | daterange | 日期范围选择器，格式: yyyy-MM-dd |

### 表格字段

| 序号 | 字段名 | 显示名称 | 类型 | 宽度 | 可排序 | 说明 |
|------|--------|----------|------|------|--------|------|
| 1 | - | 选择框 | checkbox | 50px | - | 批量选择，支持跨页保留 (row-key: fileName) |
| 2 | fileName | 文件名 | 图标+文本 | min-200px | - | 显示压缩图标 + 文件名，超长省略 |
| 3 | uploadTime | 上传时间 | 日期时间 | 160px | 是 | 文件上传时间 |

### 批量操作

| 按钮 | 图标 | 功能说明 | 触发条件 |
|------|------|----------|----------|
| 已选文件 | el-icon-selected | 显示已选文件数量 `(n)`，点击展开已选列表弹窗 | 有权限时显示 |
| 下载 | el-icon-operation-download | 批量下载文件（打包为ZIP） | 至少选中1条 |
| 删除 | el-icon-operation-delete | 批量删除文件 | 至少选中1条 |

### 已选文件弹窗
- 显示已选文件列表（文件名 fileName）
- 支持单个删除已选文件
- 支持清空全部已选
- 分页显示（前端分页，height: 270px）
- 使用 `fileName` 作为唯一标识

### 删除确认弹窗
- 标题: "确认"
- 内容: "确认删除"
- 类型: warning

### API 接口

#### 1. 获取文件列表
```
GET /api/v1/pm/kpi-measurement/files
```

**Query 参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| smallCellCode | string | 是 | 设备编码 |
| startTime | string | 否 | 开始时间 (yyyy-MM-dd) |
| endTime | string | 否 | 结束时间 (yyyy-MM-dd) |
| timeZone | string | 是 | 时区 |
| page | int | 是 | 页码 |
| pageSize | int | 是 | 每页条数 |

**Response**:
```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "fileId": "string",         // 文件ID
        "fileName": "string",       // 文件名
        "uploadTime": "2026-01-01 00:00:00"  // 上传时间
      }
    ],
    "total": 50
  }
}
```

#### 2. 批量下载文件
```
POST /api/v1/pm/kpi-measurement/files/download
```

**Request Body**:
```json
{
  "serialNumber": "string",     // 设备序列号
  "fileList": [                 // 文件列表
    {
      "fileId": "string",
      "fileName": "string"
    }
  ]
}
```

**Response**: 文件流（ZIP压缩包）

#### 3. 批量删除文件
```
POST /api/v1/pm/kpi-measurement/files/delete
```

**Request Body**:
```json
{
  "serialNumber": "string",     // 设备序列号
  "fileList": [                 // 文件列表
    {
      "fileId": "string",
      "fileName": "string"
    }
  ]
}
```

**Response**:
```json
{
  "code": 0,
  "message": "操作成功"
}
```

### 权限控制

| 权限码 | 说明 |
|--------|------|
| CODE_PERFORMANCE_MEASUREMENT | 控制是否显示操作按钮 |

---

## 前端组件建议

```
src/pages/performance/KpiMeasurement/
├── index.tsx                    # 主页面
├── components/
│   ├── SearchBar.tsx            # 搜索和筛选栏
│   ├── KpiTable.tsx             # 数据表格
│   ├── SelectedDevicesDrawer.tsx # 已选设备抽屉
│   ├── MeasurementFileDrawer.tsx # 测量文件抽屉（全屏）
│   └── SelectedFilesPopover.tsx  # 已选文件弹窗
├── hooks/
│   └── useKpiMeasurement.ts
└── types.ts                     # 类型定义
```

---

## 状态流转图

```
         ┌─────────────────────────────────────┐
         │                                     │
         ▼                                     │
    ┌────────┐    启用成功    ┌────────┐       │
    │   关   │ ─────────────▶ │  正常  │ ──────┤
    │  (0)   │                │  (1)   │       │
    └────────┘ ◀───────────── └────────┘       │
         ▲        禁用成功         │            │
         │                         │ 设备异常   │
         │         重启/恢复       ▼            │
         │                   ┌────────┐        │
         └───────────────────│损毁中止│────────┘
                             │  (2)   │
                             └────────┘
```

---

## 注意事项

1. **开关操作**: 启用/禁用测量任务时，如果设备 `needReboot=1`，需要提示用户设备将重启
2. **批量操作**: 支持跨页批量选择
   - 主表格使用 `serialNumber` 作为 row-key 保留选择
   - 文件表格使用 `fileName` 作为 row-key 保留选择
3. **实时刷新**: 开关操作成功后自动刷新表格数据
4. **权限控制**: 无写权限时隐藏操作按钮，表格不显示选择列和开关列
5. **时区参数**: 所有查询接口都需要传递 timeZone 参数
6. **文件下载**: 下载完成后自动清空已选文件列表
7. **文件删除**: 删除成功后自动刷新文件列表并清空已选

---

## 数据字段对照表

### 主表格字段映射 (Backend → Frontend)

| Backend (snake_case) | Frontend (camelCase) | 说明 |
|----------------------|----------------------|------|
| serial_number | serialNumber | 小站编码 |
| host_name | hostName | 主机名 |
| cell_id | cellId | 基站ID |
| small_cell_code | smallCellCode | 小站代码 |
| report_enable | reportEnable | 是否启用 |
| report_period | reportPeriod | 测量周期 |
| status | status | 状态 |
| need_reboot | needReboot | 是否需要重启 |
| start_time | startTime | 开始时间 |
| update_time | updateTime | 更新时间 |

### 文件表格字段映射 (Backend → Frontend)

| Backend (snake_case) | Frontend (camelCase) | 说明 |
|----------------------|----------------------|------|
| file_id | fileId | 文件ID |
| file_name | fileName | 文件名 |
| upload_time | uploadTime | 上传时间 |
