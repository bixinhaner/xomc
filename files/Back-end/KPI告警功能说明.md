# KPI 告警功能

## 1. 功能概述

KPI 告警功能允许用户在 Web 页面上配置告警规则模板，指定 KPI 指标的值达到某个阈值条件（如大于某值）时，自动生成告警。告警触发后还可以选择执行关联操作（限制带宽或去激活小区）。

## 2. 核心概念

### 2.1 告警规则模板（perf_alarm_temp）

用户创建的告警规则集合，包含模板名称、启停状态、设备关联方式、描述等信息。每个模板归属于一个运营商（operator_code），同一运营商下模板名称不可重复。

### 2.2 阈值规则（perf_alarm_threshold）

模板中的具体告警条件，每条规则包含：
- **指标**（indicator_id）：要监控的 KPI 指标 ID
- **条件 1**（comparison + threshold_value）：如 `>` `90`，表示指标值大于 90
- **条件 2**（comparison2 + threshold_value2）：可选的第二组条件，与条件 1 为 AND 关系
- **关联操作**（operation）：告警触发后对基站执行的动作

两组条件中至少要有一组不为空，若只配置一组则另一组视为"无条件通过"。

### 2.3 设备关联方式

模板通过 `select_device_type` 字段决定如何关联设备：
- **按设备组**（select_device_type=1）：模板关联若干设备组，组内所有设备均受此模板约束
- **按设备**（select_device_type=2）：模板直接关联若干具体设备（通过序列号）

### 2.4 告警状态

告警记录有三种状态：
- **create（活跃）**：status=1，告警正在持续中
- **clear（清除）**：status=0，告警条件不再满足，已自动清除
- **update（更新）**：告警仍在持续，但指标实际值发生变化，更新了告警描述

### 2.5 关联操作（operation）

告警触发后，除了生成告警外，还可以选择对基站执行以下操作：
- **Alarm**（或不填）：仅生成告警，不执行额外操作
- **Bandwidth Limiting XM**：将基站带宽限制为 X MHz（如 `Bandwidth Limiting 5M` 表示限制为 5MHz）
- **Deactive**：去激活基站小区

关联操作受系统开关 `kpi_alarm_operation_enable` 控制，开关为 0 时所有关联操作都不生效。

### 2.6 additionalText（告警唯一标识）

每条 KPI 告警通过 `additionalText` 字段在活动告警表中唯一标识，格式为：

```
300-{kpiId}-{comparison}-{thresholdValue}-{comparison2}-{thresholdValue2}-{operation};BTS_{btsId};PLMN_{plmnId};ECI_{cellId}
```

- `300` 为 KPI 告警固定的 ALARM_IDENTIFIER
- 维度信息（BTS/PLMN/ECI）按实际数据附加，用于区分多 BTS、多 PLMN、多小区场景

### 2.7 多维度支持

KPI 告警支持以下多维度场景的告警区分：
- **BSC 设备**：通过 bts_id 和 plmnId 区分不同 BTS 和 PLMN 的告警
- **4G 设备**：通过 cellId（实际存储为 ECI）区分不同小区的告警，展示时会将 ECI 转换为 eNB ID + Cell ID

## 3. 系统设置项

以下配置存储在 `sys_settings` 表中：

| 配置项 | 键名 | 说明 |
|-------|------|------|
| 模板数量上限 | `KpiAlarmTempMaxnum` | 单个运营商可创建的最大模板数。添加模板时校验，超过则拒绝 |
| 阈值规则数上限 | `KpiAlarmThresholdMaxnum` | 单个模板可配置的最大阈值规则数。前端控制 |
| 操作开关 | `kpi_alarm_operation_enable` | 是否启用告警关联操作功能。值为 `1` 时启用，为 `0` 或不存在时不启用 |

## 4. 数据库表结构

所有表位于 `small_cell` 数据库下。

### 4.1 perf_alarm_temp — 告警模板表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | varchar(32) | 主键，UUID 格式 |
| temp_name | varchar | 模板名称，同一 operator_code 下唯一 |
| status | varchar(1) | 状态：`1`=启用，`0`=禁用 |
| select_device_type | varchar(1) | 设备关联方式：`1`=按设备组，`2`=按设备 |
| operator_code | varchar | 归属的运营商编码 |
| updator | varchar | 最后修改人 |
| update_time | datetime | 最后修改时间（UTC） |
| description | varchar | 模板描述 |
| alarm_update_time | datetime | 该模板最后一次产生告警的时间 |

### 4.2 perf_alarm_threshold — 阈值规则表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int (auto_increment) | 主键 |
| temp_id | varchar(32) | 关联的模板 ID |
| indicator_id | varchar | KPI 指标 ID |
| comparison | varchar(8) | 条件 1 的比较运算符：`>`、`<`、`>=`、`<=`、`==`，可为空 |
| threshold_value | varchar(20) | 条件 1 的阈值，可为空 |
| comparison2 | varchar(8) | 条件 2 的比较运算符，可为空 |
| threshold_value2 | varchar(20) | 条件 2 的阈值，可为空 |
| operation | varchar(50) | 关联操作：`Alarm`/`Bandwidth Limiting XM`/`Deactive`，可为空 |

> 约束：comparison+threshold_value 与 comparison2+threshold_value2 两组条件至少有一组不为空。索引：(temp_id, indicator_id)。

### 4.3 perf_alarm_device_group — 模板-设备组关联表

| 字段 | 类型 | 说明 |
|------|------|------|
| temp_id | varchar(32) | 关联的模板 ID |
| group_id | int | 设备组 ID |

### 4.4 perf_alarm_device_info — 模板-设备关联表

| 字段 | 类型 | 说明 |
|------|------|------|
| temp_id | varchar(32) | 关联的模板 ID |
| serial_number | varchar | 设备序列号 |

### 4.5 perf_alarm_infos — KPI 告警记录表

| 字段 | 类型 | 说明 |
|------|------|------|
| alarm_index | varchar | 告警索引 ID，关联活动告警表中的 ALARM_ID。新增告警时初始为空，告警处理模块创建活动告警后回填 |
| event_type | varchar | 事件类型，固定 `30003`（表示 Major） |
| alarm_identifier | varchar | 告警标识，固定 `300`（KPI 告警） |
| probable_cause | varchar | 告警可能原因，固定 `KPI Alarm` |
| specific_problem | varchar | 告警详细描述，包含实际值与阈值对比信息及维度信息 |
| alarm_time | datetime | 告警时间（UTC） |
| status | varchar(1) | `1`=create（活跃），`0`=clear（已清除） |
| temp_id | varchar(32) | 关联的模板 ID |
| serial_number | varchar | 设备序列号 |

### 4.6 alarm_infos — 活动告警表（公共告警表）

KPI 告警与设备告警共用此表。KPI 告警的 `ALARM_IDENTIFIER` 固定为 `300`。

| 关键字段 | 说明 |
|---------|------|
| ALARM_ID | 告警唯一 ID |
| ALARM_IDENTIFIER | 告警标识，KPI 告警为 `300` |
| ADDITIONAL_TEXT | 告警定位信息，KPI 告警的唯一标识（见 2.6 节格式） |
| SMALL_CELL_CODE | 基站编码 |
| SERIAL_NUMBER | 设备序列号 |
| EVENT_TYPE | 事件类型 |
| SPECIFIC_PROBLEM | 告警标题 |
| EVENT_TIME | 事件时间 |
| DEAL_STATE | 处理状态 |

### 4.7 alarm_infos_his — 历史告警表

结构与 `alarm_infos` 基本一致，额外包含：
- **clear_user**：清除用户（系统自动清除时为 `SYSTEM`）
- **clear_time**：清除时间
- **clear_type**：清除类型（`2`=系统自动清除）
- **deal_state**：处理状态（清除时在原值基础上 +2）

### 4.8 alarm_serverity — 告警级别配置表

KPI 告警要求此表中存在 `ALARM_IDENTIFIER = '300'` 且 `BROAD_VERSION = 'Y'` 的记录，否则不会生成 KPI 告警。

## 5. 接口说明

所有接口基础路径：`/pm/alarm`

### 5.1 查询告警模板列表

- **URL**: `POST /pm/alarm/getKpiAlarmTempListPageData`
- **权限**: 需要登录
- **请求参数**:

| 参数 | 必填 | 说明 |
|------|------|------|
| searchText | 否 | 模糊搜索，按模板名称搜索 |
| timeZone | 是 | 时区偏移分钟数（如 480 表示 UTC+8） |
| page | 是 | 当前页码 |
| rows | 是 | 每页行数 |
| sort | 否 | 排序字段 |
| order | 否 | 排序方向（asc/desc） |

- **返回值**: 分页对象

```json
{
  "total": 10,
  "rows": [
    {
      "id": "模板ID",
      "temp_name": "模板名称",
      "status": "on 或 off",
      "select_device_type": "1 或 2",
      "updator": "修改人",
      "update_time": "修改时间（已转时区）",
      "alarm_update_time": "最后告警时间（已转时区）",
      "index": "最后一次告警的 ALARM_ID",
      "last_time": "最后一次告警的时间（已转时区）"
    }
  ]
}
```

> 返回中 status 已从 `1`/`0` 转换为 `on`/`off`。index 和 last_time 为该模板最近一条告警的信息，需要额外查询 alarm_infos 表关联获得。

### 5.2 查询告警记录列表

- **URL**: `POST /pm/alarm/getKpiAlarmInfoListPageData`
- **权限**: 需要登录
- **请求参数**:

| 参数 | 必填 | 说明 |
|------|------|------|
| tempId | 是 | 模板 ID |
| timeZone | 是 | 时区偏移分钟数 |
| page | 是 | 当前页码 |
| rows | 是 | 每页行数 |
| sort | 否 | 排序字段 |
| order | 否 | 排序方向 |

- **返回值**: 分页对象

```json
{
  "total": 5,
  "rows": [
    {
      "alarm_index": "告警索引ID，为空时显示 Filtered out",
      "event_type": "Major",
      "alarm_identifier": "300",
      "probable_cause": "KPI Alarm",
      "specific_problem": "KPI_ID: Actual value(95.5) > Threshold value(90).",
      "alarm_time": "告警时间（已转时区）",
      "status": "create 或 clear",
      "serial_number": "设备序列号"
    }
  ]
}
```

> 默认按 alarm_time 降序排列。

### 5.3 添加或修改告警模板

- **URL**: `POST /pm/alarm/addKpiAlarmTempInfo`
- **权限**: 需要登录
- **请求参数**（JSON Body）:

| 参数 | 必填 | 说明 |
|------|------|------|
| tempId | 否 | 模板 ID。为空表示新增，不为空表示修改 |
| tempName | 是 | 模板名称 |
| status | 是 | 启停状态：`on` 或 `off` |
| description | 否 | 模板描述 |
| selectType | 是 | 设备关联方式：`device` 或 `group` |
| groups | 否 | selectType=group 时，设备组 ID，逗号分隔 |
| devices | 否 | selectType=device 时，设备序列号，逗号分隔 |
| rules | 是 | 阈值规则数组 |
| rules[].indicator | 是 | 指标 ID |
| rules[].comparison | 否 | 条件 1 的比较运算符 |
| rules[].threshold | 否 | 条件 1 的阈值 |
| rules[].comparison2 | 否 | 条件 2 的比较运算符 |
| rules[].threshold2 | 否 | 条件 2 的阈值 |
| rules[].operation | 否 | 关联操作 |

- **返回值**:

```json
{ "success": true, "message": "" }
{ "success": false, "message": "名称已存在" }
{ "success": false, "message": "可配置最大数 50" }
```

- **业务规则**:
  1. 新增时生成 UUID 作为模板 ID
  2. 新增时校验当前运营商模板数量是否达到上限（KpiAlarmTempMaxnum）
  3. 校验模板名称在同一运营商下是否重复
  4. status 从 `on`/`off` 转换为 `1`/`0` 存入数据库
  5. selectType 从 `group`/`device` 转换为 `1`/`2` 存入数据库
  6. 保存模板时采用 `INSERT ... ON DUPLICATE KEY UPDATE`，支持幂等
  7. 保存阈值规则时先删除旧规则再批量插入，支持幂等
  8. rules 中 threshold 和 threshold2 不能同时为空
  9. 操作开关 `kpi_alarm_operation_enable` 不为 `1` 时，operation 字段置空
  10. operation 为空或 `Alarm` 时，存入数据库为 null

### 5.4 查询告警模板详情

- **URL**: `POST /pm/alarm/getKpiAlarmTempInfo`
- **权限**: 需要登录
- **请求参数**:

| 参数 | 必填 | 说明 |
|------|------|------|
| tempId | 是 | 模板 ID |

- **返回值**:

```json
{
  "tempId": "模板ID",
  "tempName": "模板名称",
  "status": "on 或 off",
  "selectType": "group 或 device",
  "updator": "修改人",
  "update_time": "修改时间",
  "description": "描述",
  "groups": [1, 2, 3],
  "devices": [
    {
      "smallCellCode": "基站编码",
      "serialNumber": "序列号",
      "module_type": "模块类型",
      "hostName": "主机名",
      "product": "产品型号",
      "product_name": "产品名称"
    }
  ],
  "rules": [
    {
      "indicator": "指标ID",
      "comparison": ">",
      "threshold": "90",
      "comparison2": "none",
      "threshold2": "",
      "operation": "Alarm"
    }
  ]
}
```

> 返回中 status 已从 `1`/`0` 转换为 `on`/`off`，selectType 已从 `1`/`2` 转换为 `group`/`device`。rules 中空条件会补充为 comparison=`none`、threshold=`""`。

### 5.5 修改模板状态（启用/禁用）

- **URL**: `POST /pm/alarm/updateKpiAlarmTempStatus`
- **权限**: 需要登录
- **请求参数**:

| 参数 | 必填 | 说明 |
|------|------|------|
| tempId | 是 | 模板 ID |
| status | 是 | `on` 或 `off` |

- **返回值**:

```json
{ "success": true, "message": "" }
{ "success": false, "message": "" }
```

- **业务规则**: 禁用模板时，会将该模板产生的所有活动告警转为历史告警（见 8.3 节）。

### 5.6 删除告警模板

- **URL**: `POST /pm/alarm/delKpiAlarmTemp`
- **权限**: 需要登录
- **请求参数**:

| 参数 | 必填 | 说明 |
|------|------|------|
| tempId | 是 | 模板 ID |

- **返回值**:

```json
{ "success": true, "message": "" }
{ "success": false, "message": "" }
```

- **业务规则**: 删除模板时级联删除所有关联数据（见 8.2 节）。

### 5.7 获取可选指标列表

- **URL**: `POST /pm/alarm/getIndicatorsList`
- **权限**: 需要登录
- **请求参数**:

| 参数 | 必填 | 说明 |
|------|------|------|
| product_type | 否 | 产品类型，逗号分隔（如 `ENB,GSM,GNB`） |

- **返回值**: 指标列表

```json
[
  {
    "value": "K000010001",
    "text": "Indicator English Name",
    "unit_id": "Unit Name"
  }
]
```

- **业务规则**:
  1. 查询 `perf_indicators` 表中属于当前运营商或公共（operator_code 为空）的指标
  2. 指标必须属于某个指标分组（indicator_group 或 indicator_group_gsm），排除 `Customize[DEPRECATED]` 分组
  3. 按 product_type 过滤指标的 product_types 字段
  4. 支持自定义指标名称（perf_cust_name），按运营商匹配

### 5.8 检查模板数量是否达到上限

- **URL**: `POST /pm/alarm/checkKpiAlarmTempIsExist`
- **权限**: 需要登录
- **请求参数**: 无（自动获取当前登录用户的运营商）
- **返回值**:

```json
{ "success": true, "message": "" }
{ "success": false, "message": "可配置最大数 50" }
```

### 5.9 查询模板关联的设备列表

- **URL**: `POST /pm/alarm/getKPIAlarmSelDeviceListPageData`
- **权限**: 需要登录
- **请求参数**:

| 参数 | 必填 | 说明 |
|------|------|------|
| tempId | 是 | 模板 ID |
| page | 是 | 当前页码 |
| rows | 是 | 每页行数 |
| sort | 否 | 排序字段 |
| order | 否 | 排序方向 |

- **返回值**: 分页对象

```json
{
  "total": 3,
  "rows": [
    {
      "smallCellCode": "基站编码",
      "serialNumber": "序列号",
      "module_type": "模块类型",
      "hostName": "主机名",
      "product": "产品型号",
      "product_name": "产品名称"
    }
  ]
}
```

> 默认按 serial_number 降序。产品型号会做合并处理（如 CR-B4860-SC/CA/TC/DC 统一显示为 CR-B4860）。

### 5.10 查询模板关联的设备组列表

- **URL**: `POST /pm/alarm/getKPIAlarmSelDeviceGroupListPageData`
- **权限**: 需要登录
- **请求参数**:

| 参数 | 必填 | 说明 |
|------|------|------|
| tempId | 是 | 模板 ID |
| page | 是 | 当前页码 |
| rows | 是 | 每页行数 |
| sort | 否 | 排序字段 |
| order | 否 | 排序方向 |

- **返回值**: 分页对象

```json
{
  "total": 2,
  "rows": [
    {
      "id": 1,
      "group_name": "设备组名称"
    }
  ]
}
```

> 默认按 group_id 降序。

## 6. 告警生成逻辑

### 6.1 触发时机

KPI 告警的生成入口是唯一的，发生在**每次 KPI 指标值计算完成时**。系统将以下两类指标的值传入告警检查流程：
- **公式计算的 KPI 指标值**：通过公式从 counter 数据计算得出的 KPI 指标
- **非计算型的指标原始值**：原始数据中直接存在的指标值（包括 counter 类指标），不包含值为 `-` 或 `N/A` 的指标

### 6.2 生成流程

1. 从设备缓存中获取基站信息（operator_code、device_group_code、serial_number、host_name）
2. 从原始 PM 数据中提取维度信息：bts_id（BSC 设备）、plmnId、cellId（4G 设备为 ECI）
3. 如果有 KPI 指标值（kpiAlarmMap 不为空），调用告警检查流程
4. 查询当前运营商下所有**启用状态**（status=1）的告警模板
5. 检查 `alarm_serverity` 表中是否存在 `ALARM_IDENTIFIER='300'` 且 `BROAD_VERSION='Y'` 的记录，不存在则跳过
6. 对每个模板，判断当前设备是否匹配：
   - 若模板关联方式为设备组（select_device_type=1），查询该设备所属组是否在模板关联的组中
   - 若模板关联方式为设备（select_device_type=2），查询该设备序列号是否在模板关联的设备中
7. 设备匹配成功后，查询模板的所有阈值规则（perf_alarm_threshold），逐条检查
8. 对每条阈值规则进行门限匹配检查（见 6.4 节）
9. 根据匹配结果和当前告警状态，决定新增/更新/清除告警（见 6.6 节）
10. 如果告警触发且有关联操作，执行操作（见第 7 节）

### 6.3 告警处理优先级

同一台设备在同一次检查中可能产生多种告警操作，系统按以下顺序处理，防止新增的告警被误清除：

1. **第一步**：处理清除告警 — 将不再满足阈值条件的活动告警清除
2. **第二步**：清除不在阈值规则范围内的历史告警 — 处理阈值规则被修改前产生的旧告警
3. **第三步**：处理更新告警 — 更新仍在持续的告警的描述信息
4. **第四步**：处理新增告警 — 生成新的活动告警

#### 去重机制

在同一次检查中，对同一台设备，相同的 additionalText 只会处理一次：
- 用 `isList` 记录已处理的 additionalText，遍历阈值规则时若重复则跳过
- 新增告警列表 `activeAlarmlist` 中也做 additionalText + smallCellCode 去重，防止多个模板对同一设备同一告警重复新增

### 6.4 阈值匹配算法

对单条阈值规则：

1. 从当前 KPI 数据中获取该规则指定指标（indicator_id）的值
2. 如果指标值不存在，跳过该规则
3. 检查条件 1：使用 comparison（如 `>`）和 threshold_value（如 `90`）进行比较
   - 若 threshold_value 为空，条件 1 视为"通过"
4. 检查条件 2：使用 comparison2 和 threshold_value2 进行比较
   - 若 threshold_value2 为空，条件 2 视为"通过"
5. 两组条件取 **AND** 逻辑：仅当两组条件**同时通过**时，判定为触发告警

比较运算符支持：`>`、`<`、`>=`、`<=`、`==`

### 6.5 specific_problem 构造规则

specific_problem 是告警的详细描述，格式如下：

```
{kpiId}: Actual value({实际值}) {comparison} Threshold value({阈值1}) [and {comparison2} Threshold value({阈值2})] [, {operation}.] [BTS: {btsId}({bts序列号}), PLMN: {plmnId}, Cell ID: {cellId} (eNB ID: {enodeId}, ECI: {eci})]
```

构造规则：
- 前缀：`{kpiId}: Actual value({indicatorVal})`
- 若条件 1 不为空，追加 `{comparison} Threshold value({thresholdValue})`
- 若条件 2 不为空，追加 `and {comparison2} Threshold value({thresholdValue2})`（and 前有空格）
- 若 operation 不为空且不为 `Alarm`，追加 `, {operation}.`
- 否则以 `.` 结尾
- 维度信息（至少一个不为空时追加 ` [...]`）：
  - btsId 不为空时，追加 `BTS: {btsId}`（若有对应 BTS 序列号则显示为 `BTS: {btsId}({bts序列号})`）
  - plmnId 不为空时，追加 `PLMN: {plmnId}`
  - cellId 不为空时，若为非 BSC 设备且 cellId 为数字，追加 `Cell ID: {eci % 256} (eNB ID: {eci / 256}, ECI: {eci})`

示例：
```
K000010001: Actual value(95.5) > Threshold value(90).
K000010001: Actual value(95.5) > Threshold value(90) and < Threshold value(100), Bandwidth Limiting 5M.
K000010001: Actual value(95.5) > Threshold value(90). [BTS: 1(SN001), PLMN: 46000]
```

### 6.6 告警生命周期管理

根据阈值匹配结果和活动告警表（alarm_infos）中是否已存在同 additionalText + SMALL_CELL_CODE 的告警，分为三种情况：

| 阈值匹配 | 活动告警已存在 | 操作 | 说明 |
|---------|-------------|------|------|
| 匹配 | 已存在 | **更新** | 更新 perf_alarm_infos 的 specific_problem 和 alarm_time；将告警信息推入告警队列，告警系统更新 ALARM_ID 对应的活动告警 |
| 匹配 | 不存在 | **新增** | 在 perf_alarm_infos 插入新记录（status=1）；推入告警队列，告警系统创建新的活动告警 |
| 不匹配 | 已存在 | **清除** | 更新 perf_alarm_infos 的 status=0；推入清除告警队列，告警系统清除对应活动告警 |

告警队列通过消息队列传递给告警处理模块，告警处理模块负责实际的 alarm_infos 表写入和北向通知。

### 6.7 多维度场景支持

#### BSC 设备（多 BTS、多 PLMN）

- 原始 PM 数据中包含 bts_id 和 plmnId 字段
- 告警的 additionalText 中附加 `;BTS_{btsId};PLMN_{plmnId}` 以区分不同 BTS/PLMN
- specific_problem 中会查询 BTS 的序列号显示为 `BTS: {btsId}({btsSerialNumber})`

#### 4G 设备（多小区）

- 原始 PM 数据中 cellId 实际存储为 ECI（E-UTRAN Cell Identifier）
- 告警的 additionalText 中附加 `;ECI_{cellId}`
- specific_problem 展示时将 ECI 转换：`Cell ID: {eci % 256} (eNB ID: {eci / 256}, ECI: {eci})`

#### DC 基站辅小区

- 当平台类型为 `QA_436Q_DC`、`NEU430_DC`、`Intel_CR_DC`、`MLN_DC` 且 smallCellCode 以 `-2` 结尾时，表示 DC 基站的辅小区
- 此场景下使用主小区编码（去掉 `-2` 后缀）从设备缓存获取 operator_code 等信息，但 serial_number 追加 `-2` 后缀
- 关联操作时使用辅小区的 cellIdx=2

## 7. 关联操作

### 7.1 操作开关

系统设置 `kpi_alarm_operation_enable` 控制是否启用关联操作：
- 值为 `1` 时启用
- 值为 `0` 或不存在时，所有 operation 都被忽略，仅生成告警

### 7.2 限制带宽（Bandwidth Limiting）

当 operation 包含 `Bandwidth` 关键字时执行带宽限制操作：

1. 从 operation 字段中提取带宽数值（如 `Bandwidth Limiting 5M` → 5）
2. 根据基站平台类型，将带宽数值转换为基站参数值：

| 平台类型 | 参数值范围 | 转换规则 |
|---------|----------|---------|
| QA_V3 / QA_V4 | 5-20 | 直接使用原始值 |
| BAIBLQ / MLQ / 436Q / Intel_CR / BLX / MLN | 25-100 | 原始值 × 5 |
| Intel / DXDF / 其他 | n25-n100 | `n` + 原始值 × 5 |

3. 读取基站当前下行带宽值，若与目标值相同则跳过（避免重复下发）
4. 组装参数修改任务：
   - QA_V3/QA_V4 平台：仅设置下行带宽
   - 其他平台：同时设置上行和下行带宽
5. 设置回调 URL：`/rpc/kpiAlarm/callback/bandwidthLimitCallback?sn={sn}`
6. 下发参数修改任务给基站

> 辅小区（smallCellCode 以 `-2` 结尾）场景下，带宽参数中的小区索引使用 `{i}` 替换为 `2`。

### 7.3 去激活小区（Deactive）

当 operation 包含 `Deactive` 关键字时执行去激活操作：

1. 根据平台确定下发的参数值：

| 平台类型 | 下发值 |
|---------|-------|
| QA_V3 / QA_V4 | `DOWN` |
| 其他 | `0` |

2. 根据平台确定参数路径：

| 平台类型 | 参数 |
|---------|------|
| Intel_CR / MLN | RF 状态参数（小区激活与 RF 开关合一） |
| 其他 | 管理状态参数（AdminState） |

3. 下发参数修改任务给基站

### 7.4 带宽限制回调

- **URL**: `GET /rpc/kpiAlarm/callback/bandwidthLimitCallback`
- **参数**: sn（基站序列号）、success（是否成功）、errMsg（失败原因）
- **逻辑**:
  - 成功时（success=true）：自动创建设备重启任务
  - 失败时：仅记录日志，不做额外处理

## 8. 模板管理业务规则

### 8.1 新增模板校验

新增模板时需要通过以下校验：

1. **数量限制**：查询当前运营商已有的模板数量，与 `KpiAlarmTempMaxnum` 比较，达到上限则拒绝
2. **名称唯一**：同一 operator_code 下，id 不同但 temp_name 相同的记录不能存在

> 修改模板时不检查数量限制，仅检查名称唯一性。

### 8.2 删除模板的影响

删除模板时级联执行以下操作：

1. 删除 `perf_alarm_temp` 中的模板记录
2. 删除 `perf_alarm_device_group` 中该模板的设备组关联
3. 删除 `perf_alarm_device_info` 中该模板的设备关联
4. 删除 `perf_alarm_infos` 中该模板的告警记录
5. 删除 `perf_alarm_threshold` 中该模板的阈值规则
6. 将该模板产生的所有**活动告警**从 `alarm_infos` 转移到 `alarm_infos_his`：
   - 查询该模板在 perf_alarm_infos 中的所有告警记录
   - 根据 alarm_index 和 serial_number 从 alarm_infos 中找到对应的活动告警
   - 将活动告警插入 alarm_infos_his（clear_user=`SYSTEM`，clear_type=2，deal_state 在原值基础上 +2）
   - 删除 alarm_infos 中的原活动告警
   - 如支持北向接口，推送清除告警通知

### 8.3 禁用模板的影响

禁用模板（status 从 `1` 改为 `0`）时：

1. 更新 perf_alarm_temp 的 status 字段为 `0`
2. 将该模板产生的所有**活动告警**从 `alarm_infos` 转移到 `alarm_infos_his`（流程同 8.2 节第 6 步）
