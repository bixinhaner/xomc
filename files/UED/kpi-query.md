# KPI 性能查询页面

## 页面概述

**页面名称**: 性能查询 (KPI Query)
**菜单路径**: 性能管理 > 性能查询
**API前缀**: `/api/v1/pm/kpi-query`

---

## 页面布局

### 左侧面板 - 查询模板
- **宽度**: 260px
- **标题**: KPI性能查询模板
- **功能**:
  - 模板名称搜索
  - 模板树形列表（公共模板/私有模板分组）
  - 新增模板按钮

### 右侧面板 - 查询结果
- **标签页**: 支持多个模板同时打开（Tab 页签形式）
- **视图切换**: 表格视图 / 图表视图
- **内容**:
  - 查询条件区域
  - 结果表格/图表区域

---

## 左侧模板树功能

### 模板树结构

模板树采用两级分组结构：

```
├── 公共模板 (isPublic = '1')
│   ├── admin
│   │   └── 模板名称1 (默认图标)
│   │   └── 模板名称2
│   └── 其他分组...
└── 私有模板 (isPublic = '0')
    ├── 用户名1
    │   └── 模板A
    │   └── 模板B
    └── 用户名2 (仅管理员可见)
        └── 模板C
```

### 模板树字段

| 字段 | 说明 |
|------|------|
| group_name | 模板分组名称 |
| tempId | 模板ID |
| is_default | 是否为默认模板（显示默认图标） |
| reportSwitch | 是否启用定时报表（显示时钟图标） |
| isPublic | 是否公共模板：'0'-私有模板，'1'-公共模板 |
| isOneSelf | 是否自己创建：'1'-本人创建 |
| isAdmin | 是否管理员：'1'-管理员 |

### 模板可见性规则

| 模板类型 | 可见用户 |
|----------|----------|
| 公共模板 | 所有用户可见 |
| 私有模板 | 仅创建者可见 |
| 私有模板 | 管理员可见所有用户的私有模板 |

### 模板操作菜单

#### 操作项列表

| 操作 | 图标 | 说明 | 显示条件 |
|------|------|------|----------|
| 已为默认 | star-filled | 当前已是默认模板，点击无操作 | `is_default == 'true'` |
| 设为默认 | star | 将模板设置为默认 | `is_default != 'true'` |
| 详情 | info | 查看模板详情信息 | 始终显示 |
| 修改 | edit | 编辑模板配置 | 始终显示 |
| 导出指标 | export | 导出KPI指标配置 | 始终显示 |
| 定时报表 | clock | 配置定时报表 | 始终显示 |
| 删除 | delete | 删除模板 | 非默认模板 |
| 复制模板 | copy | 复制模板 | 始终显示 |

#### 删除权限说明
- 默认模板不可删除
- 内置模板 (`isCustomize == '0'`) 不可删除

### 模板搜索
- 支持按模板名称模糊搜索
- 回车触发搜索

---

## 新增/编辑模板页面

### 模板基础信息

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| templateName | 输入框 | 是 | 模板名称 |
| isPublic | 单选 | 是 | 公共/私有模板 |
| description | 文本域 | 否 | 模板描述 |

### 模板指标配置

| 字段 | 类型 | 说明 |
|------|------|------|
| networkType | 单选 | 网元类型：eNB/gNB/eGW |
| operatorType | 多选 | 运营商类型（多运营商时显示） |
| indicators | 多选树 | KPI指标选择（支持搜索、分类筛选） |

### 指标选择字段

| 字段 | 说明 |
|------|------|
| kpiId | 指标ID |
| kpiName | 指标名称 |
| unit | 单位 |
| isCounter | 是否计数器：'0'-否，'1'-是 |
| isBuildIn | 是否内置：'0'-否，'1'-是 |
| platformSupported | 支持平台：enb,gnb,egw |
| catagoryId | 分类ID |
| catagoryName | 分类名称 |

---

## 右侧查询条件

### 基础查询条件

| 字段 | 类型 | 说明 |
|------|------|------|
| selDeviceType | 单选 | 查询对象类型：1-设备组 2-设备 |
| searchText | 输入框 | 设备编码/设备名称搜索 |
| operatorParam | 下拉框 | 运营商筛选（当模板为多运营商时显示） |
| deviceGroupParam | 树选择 | 设备组筛选 |
| dateValue | 日期范围 | 查询时间范围 |

### 时间粒度选择

| 值 | 显示文本 | 说明 |
|----|----------|------|
| 15 | 15Min | 15分钟粒度 |
| 60 | 60Min | 60分钟粒度 |
| 1440 | 24Hour | 24小时粒度 |
| 10080 | Week | 周粒度（仅 eNB） |
| 43200 | Month | 月粒度（仅 eNB） |

### 时间范围限制

| 粒度 | 数据保留天数 | 可选范围限制 |
|------|--------------|--------------|
| 15Min | 30天 | 最多选择7天 |
| 60Min | 30天 | 最多选择7天 |
| 24Hour | 365天 | 最多选择365天 |
| Week | 365天 | 最多选择365天 |
| Month | 365天 | 最多选择365天 |

### 快捷时间选项（15/60分钟粒度）

| 选项 | 说明 |
|------|------|
| 今天 | 当天 00:00 - 当前时间 |
| 昨天 | 昨天 00:00 - 23:59 |
| 本周 | 本周一 - 当前时间 |
| 上周 | 上周一 - 上周日 |
| 近7天 | 7天前 - 当前时间 |
| 近30天 | 30天前 - 当前时间 |

---

## 视图切换功能

### 表格视图
- **入口**: 默认视图或点击"列表"图标
- **展示**: 以数据表格形式展示KPI数据
- **功能**: 排序、筛选、导出、KPI钻取

### 图表视图
- **入口**: 点击"图表"图标或"自定义图表"图标
- **展示**: 以折线图形式展示KPI趋势

### 视图切换逻辑

| 当前视图 | 操作 | 目标视图 |
|----------|------|----------|
| 表格 | 点击"图表"图标 | 常规图表 |
| 表格 | 点击"自定义图表"图标 | 自定义图表 |
| 常规图表 | 点击"列表"图标 | 表格 |
| 自定义图表 | 点击"列表"图标 | 表格 |

---

## 结果表格字段

### 冻结列（左侧固定）

| 字段 | 显示名称 | 宽度 | 说明 |
|------|----------|------|------|
| serialNumber | 小站编码 | 200px | 设备序列号 |
| hostName | 名称 | 200px | 设备名称 |

### 基础列（eNB/gNB 设备查询）

| 字段 | 显示名称 | 宽度 | 说明 |
|------|----------|------|------|
| id | - | - | 唯一标识（隐藏） |
| smallCellCode | - | - | 设备代码（隐藏） |
| plmnId | PLMN标识 | 200px | 仅5G显示 |
| subStationName | 站址名称 | 200px | 部分型号显示 |
| enodeId | eNodeBID | 190px | 基站ID |
| cellId | 小区ID | 190px | 小区标识 |
| eci | ECI | 200px | E-UTRAN小区标识 |
| groupName | 设备组名称 | 230px | 所属设备组 |
| timeLevel | 查询粒度(分钟/小时/周/月) | 170px | 带粒度单位 |
| startTime | 开始时间 | 190px | 数据开始时间 |
| endTime | 结束时间 | 190px | 数据结束时间 |

### 基础列（eGW 设备查询）

| 字段 | 显示名称 | 宽度 | 说明 |
|------|----------|------|------|
| serialNumber | eGW编码 | 200px | 设备序列号 |
| hostName | eGW名称 | 200px | 设备名称 |
| groupName | 设备组名称 | 230px | 所属设备组 |
| timeLevel | 查询粒度(分钟/小时) | 170px | 带粒度单位 |
| startTime | 开始时间 | 190px | 数据开始时间 |
| endTime | 结束时间 | 190px | 数据结束时间 |

### 基础列（设备组查询）

| 字段 | 显示名称 | 宽度 | 说明 |
|------|----------|------|------|
| group_name | 设备组名称 | 460px | 设备组名称 |
| timeLevel | 查询粒度(分钟/小时/周/月) | 300px | 带粒度单位 |
| startTime | 开始时间 | 300px | 数据开始时间 |
| endTime | 结束时间 | 300px | 数据结束时间 |

### 动态KPI列

根据模板配置动态生成，格式：`KPI名称(单位)`

| 属性 | 说明 |
|------|------|
| field | kpiId |
| title | kpiName + "(" + unit + ")" |
| width | 260px |
| formatter | 可钻取的KPI显示为蓝色链接 |

### KPI钻取

- **触发条件**: 点击可钻取的KPI值（isCounter != "1" || isBuildIn != "1"）
- **钻取页面**: 显示该KPI的详细数据
- **返回**: 关闭钻取页面返回主表格

---

## 图表功能

### 常规图表
- **入口**: 点击表格右上角"图表"图标（el-icon-circle-chart）
- **展示**: 以折线图展示选中模板的KPI趋势
- **切换回表格**: 点击"列表"图标

### 自定义图表

#### 入口
点击表格右上角"自定义图表"图标（el-chart_template）

#### 图表时间控制

| 控件 | 说明 |
|------|------|
| 时间步进（左箭头） | 查看上一个时间点数据 |
| 时间选择器 | 选择具体日期/周/月 |
| 时间步进（右箭头） | 查看下一个时间点数据（当前时间后置灰） |
| 天粒度 | 按天查看数据 |
| 周粒度 | 按周查看数据 |
| 月粒度 | 按月查看数据 |

#### 时间粒度规则

| 粒度 | 时间格式 | 示例 |
|------|----------|------|
| 天 | YYYY-MM-DD | 2026-04-02 |
| 周 | YYYY-MM-DD 00:00:00 | 2026-03-31 00:00:00 |
| 月 | YYYY-MM | 2026-04 |

#### 图表模板操作

| 操作 | 图标 | 说明 |
|------|------|------|
| 添加图表 | el-icon-plus | 新增一个图表模板 |
| 修改图表 | el-icon-operation-edit | 修改当前图表配置 |
| 删除图表 | el-icon-operation-delete | 删除当前图表 |

#### 图表配置面板

**右侧配置面板字段**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| 设备/设备组切换 | 单选 | - | 仅eNB网元支持设备组 |
| 设备选择 | 多选表格 | 是 | 最多选择N个设备（配置限制） |
| 设备组选择 | 多选表格 | 是 | 最多选择N个设备组 |
| 指标选择 | 多选表格 | 是 | 最多选择N个指标 |

#### 设备选择表格字段

| 字段 | 说明 |
|------|------|
| serialNumber | 小站编码 |
| hostName | 名称（eNB/gNB） |
| cellId | 小区ID（eNB） |
| plmnId | PLMN标识（gNB） |
| nrCGI | nrCGI（gNB） |

#### 指标选择表格字段

| 字段 | 说明 |
|------|------|
| kpiId | 指标ID |
| kpiName | 指标名称 |
| catagoryId | 分类ID（下拉筛选） |

#### 图表展示

- 多个图表并排显示（每行2个）
- 图表标题：显示选中的指标名称
- 图表内容：ECharts 折线图
- 图表高度：690px

---

## 定时报表配置

### 配置字段

| 字段 | 类型 | 说明 |
|------|------|------|
| reportStatus | 开关 | 启用/禁用定时报表 |
| reportPeriod | 多选框 | 报表周期：15Min/60Min/24Hour/Week/Month |
| reportTime | 下拉框 | 报表发送时间：00:00-23:00 |
| mailStatus | 开关 | 启用/禁用邮件发送 |
| mailAddress | 文本框 | 邮件地址（多个用分号分隔） |
| ftpSwitch | 开关 | 启用/禁用FTP上传 |
| ftpProtocol | 单选 | FTP协议：SFTP/FTP |
| ftpPath | 输入框 | 上传路径 |
| ftpIp | 输入框 | FTP服务器IP |
| ftpPort | 输入框 | FTP端口（1-65535） |
| ftpUser | 输入框 | 用户名 |
| ftpPassword | 密码框 | 密码 |

### 验证规则

| 字段 | 规则 |
|------|------|
| reportPeriod | 启用定时报表时必选 |
| mailAddress | 启用邮件时必填，格式验证 |
| ftpPath | 启用FTP时必填 |
| ftpIp | 启用FTP时必填，IP格式验证 |
| ftpPort | 启用FTP时必填，1-65535范围 |
| ftpUser | 启用FTP时必填 |
| ftpPassword | 启用FTP时必填 |

---

## 导出功能

### 导出格式
- Excel (.xlsx)
- CSV (.csv)

### 导出范围
- 当前页
- 全部数据
- 选中数据

---

## API 接口

### 1. 获取模板列表
```
GET /api/v1/pm/kpi-query/templates
```

**Response**:
```json
{
  "code": 0,
  "data": [
    {
      "tempId": "public1",
      "templateName": "公共模板",
      "isPublic": "1",
      "isDefault": true,
      "reportSwitch": "0",
      "children": [
        {
          "tempId": "template-001",
          "templateName": "基础指标模板",
          "isDefault": true,
          "reportSwitch": "1"
        }
      ]
    },
    {
      "tempId": "private0",
      "templateName": "私有模板",
      "isPublic": "0",
      "children": [
        {
          "tempId": "group-admin",
          "group_name": "admin",
          "children": [
            {
              "tempId": "template-002",
              "templateName": "我的模板",
              "isOneSelf": "1"
            }
          ]
        }
      ]
    }
  ]
}
```

### 2. 获取模板详情
```
GET /api/v1/pm/kpi-query/templates/{tempId}
```

### 3. 获取模板关联指标
```
GET /api/v1/pm/kpi-query/templates/{tempId}/indicators
```

**Response**:
```json
{
  "code": 0,
  "data": [
    {
      "kpiId": "string",
      "kpiName": "string",
      "unit": "string",
      "isCounter": "0",
      "isBuildIn": "1",
      "platformSupported": "enb,gnb"
    }
  ]
}
```

### 4. 查询KPI数据
```
POST /api/v1/pm/kpi-query/data
```

**Request Body**:
```json
{
  "tempId": "string",
  "selDeviceType": "2",
  "searchText": "string",
  "groupId": "string",
  "startTime": "2026-04-01",
  "endTime": "2026-04-02",
  "periodActive": "15",
  "timeZone": "Asia/Shanghai",
  "page": 1,
  "pageSize": 20
}
```

### 5. KPI钻取查询
```
POST /api/v1/pm/kpi-query/drilldown
```

**Request Body**:
```json
{
  "tempId": "string",
  "kpiId": "string",
  "smallCellCode": "string",
  "startTime": "2026-04-01 00:00:00",
  "endTime": "2026-04-02 00:00:00",
  "periodActive": "15",
  "timeZone": "Asia/Shanghai"
}
```

### 6. 保存定时报表配置
```
POST /api/v1/pm/kpi-query/templates/{tempId}/report
```

**Request Body**:
```json
{
  "reportStatus": "1",
  "reportPeriod": ["15", "60"],
  "reportTime": 8,
  "mailStatus": "1",
  "mailAddress": "user@example.com",
  "ftpSwitch": "1",
  "ftpProtocol": "sftp",
  "ftpPath": "/data/reports",
  "ftpIp": "192.168.1.100",
  "ftpPort": "22",
  "ftpUser": "admin",
  "ftpPassword": "password"
}
```

### 7. 导出KPI数据
```
POST /api/v1/pm/kpi-query/export
```

### 8. 获取图表模板列表
```
GET /api/v1/pm/kpi-query/templates/{tempId}/chart-templates
```

**Response**:
```json
{
  "code": 0,
  "data": [
    {
      "id": "chart-001",
      "indicators": [
        { "indicator_id": "KPI001", "indicator_name": "指标1" }
      ],
      "devices": [
        { "serialNumber": "ENB00001", "hostName": "基站1" }
      ],
      "groups": []
    }
  ]
}
```

### 9. 新增/修改图表模板
```
POST /api/v1/pm/kpi-query/templates/{tempId}/chart-templates
```

**Request Body**:
```json
{
  "id": "chart-001",
  "devices": ["device-id-1", "device-id-2"],
  "groups": ["group-id-1"],
  "indicators": ["KPI001", "KPI002"]
}
```

### 10. 删除图表模板
```
DELETE /api/v1/pm/kpi-query/templates/{tempId}/chart-templates/{chartId}
```

### 11. 获取图表数据
```
POST /api/v1/pm/kpi-query/chart-data
```

**Request Body**:
```json
{
  "tempId": "string",
  "chartId": "string",
  "period": "0",
  "startTime": "2026-04-01",
  "endTime": "2026-04-02"
}
```

---

## 权限控制

| 权限码 | 说明 |
|--------|------|
| CODE_PERFORMANCE_VIEW | 性能查询查看权限 |
| CODE_PERFORMANCE_VIEW | 新增/编辑/删除模板 |

---

## 前端组件建议

```
src/pages/performance/KPIQuery/
├── index.tsx                    # 主页面
├── components/
│   ├── TemplateTree.tsx         # 左侧模板树（公共/私有分组）
│   ├── TemplateDialog.tsx       # 模板新增/编辑弹窗
│   ├── QueryForm.tsx            # 查询条件表单
│   ├── ResultTable.tsx          # 结果表格
│   ├── ResultChart.tsx          # 常规图表
│   ├── CustomChartPanel.tsx     # 自定义图表面板
│   ├── ChartTemplateForm.tsx    # 图表模板配置表单
│   ├── DrilldownDrawer.tsx      # KPI钻取抽屉
│   └── ReportConfigDrawer.tsx   # 定时报表配置抽屉
├── hooks/
│   └── useKPIQuery.ts
└── types.ts                     # 类型定义
```

---

## 注意事项

1. **时间粒度切换**: 切换粒度时需要重新选择时间范围（不同粒度的可选范围不同）
2. **KPI钻取**: 仅非内置计数器类型的KPI支持钻取
3. **模板管理**: 公共模板所有用户可见，私有模板仅创建者和管理员可见
4. **定时报表**: 定时报表会按配置的周期和时间自动发送
5. **数据保留**: 不同粒度数据保留时间不同，超过保留时间的数据不可查询
6. **网元类型**: 当前支持 eNB、gNB、eGW 三种网元类型
7. **图表限制**: 自定义图表有设备数量和指标数量限制（配置控制）
8. **默认模板**: 管理员可将其他用户的私有模板设为默认（需确认提示）
9. **图表粒度**: 当模板上报周期为24h时，图表取消"天"粒度选项
10. **时间步进**: 图表时间步进时，当前时间之后的右箭头置灰不可点击
