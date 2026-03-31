# KPI 指标管理功能说明

> 本文档覆盖 **ENB（4G）**、**GSM（2G）**、**GNB（5G）** 三种网元的指标管理逻辑。

---

## 1. 功能概述

KPI 指标管理是性能管理（Performance Management）模块的核心子功能，允许运营商对性能测量指标进行全生命周期管理，包括：

- 查看系统内置指标和自定义指标
- 新建、修改、删除自定义指标功能集（指标组）
- 新建、修改、删除自定义 KPI 指标和自定义 Counter
- 启用（测量）/ 禁用（取消测量）指标
- 导出指标列表

入口页面路径：`/cell/perfmgmt/kpimanage/goKpiManagementPage`。

---

## 2. 核心概念说明

### 2.1 指标功能集（Indicator Group）

指标功能集是指标的分组容器，类似"文件夹"的概念。页面左侧以树形结构展示。

- **根节点**：`id = parent_id` 且 `is_build_in = '1'` 的记录，作为树的根，不对用户展示
- **内置功能集**：`is_build_in = '1'`，系统预置，不可删除
- **自定义功能集**：`is_build_in = '0'`，运营商创建，`operator_code` 字段记录归属运营商
- **ENB 和 GSM 使用不同的表**：`indicator_group`（ENB）和 `indicator_group_gsm`（GSM），表结构完全相同

`Customize[DEPRECATED]` 是一个特殊的历史遗留分组，在下拉框和树查询中会被过滤掉。

### 2.2 指标（Indicator / KPI）

所有指标（ENB 和 GSM）均存储在 `perf_indicators` 表中，通过 `group_id` 关联到对应的指标功能集表。

指标分为两个维度：

**按来源分：**

| 类型 | `is_build_in` | 说明 |
|---|---|---|
| 内置指标 | `1` | 系统预置，随 PM 文件解析产生 |
| 自定义指标 | `0` | 运营商创建，`operator_code` 字段非空 |

**按结构分：**

| 类型 | `is_counter` | 说明 |
|---|---|---|
| KPI 指标 | `0` | 有计算公式（`arithmetic` 字段），由一个或多个 Counter 通过算术表达式计算得出 |
| Counter | `1` | 原始计数器，`arithmetic` 字段存放的是 Counter 名称本身（无运算符）；自定义 Counter 的 ID 以 `D` 开头 |

### 2.3 指标级别（indicator_level）

`perf_indicators.indicator_level` 字段标识指标在哪个维度可用：

| 值 | 含义 |
|---|---|
| `device` | 设备（小区）级别，默认值 |
| `plmn` | PLMN 级别 |
| `both` | 设备和 PLMN 两个维度均可用（用于 GSM BSC/BTS 设备的临时兼容方案） |

查询时的过滤规则：
- 查 `device` 维度：返回 `indicator_level = 'device'` 和 `'both'` 的指标
- 查 `plmn` 维度：返回 `indicator_level = 'plmn'` 和 `'both'` 的指标

### 2.4 产品类型（product_type）

`perf_indicators.product_types` 字段记录指标适用的产品类型（可多个，用逗号或空格分隔，如 `QRTB,MLN,ALL`）。`ALL` 表示适用于所有产品类型。

前端传入的产品类型名称（如 `QRTB`、`CR-B4860`）与数据库中的平台名称（如 `436Q`、`Intel_CR`）之间存在映射转换，由服务层负责处理。

GNB 的产品类型较简单，服务层负责映射转换，当前支持的映射：

| 产品类型（前端） | 平台名称（数据库） |
|---|---|
| `BaiBNX` | `BaiBNX` |
| `BaiBNQ` | `BaiBNQ` |
| `CHINA_TELECOM` | `CHINA_TELECOM` |
| `custom` / `gnb_custom` | `ALL`（展示用） |

### 2.5 指标启用/禁用

只有被"启用（测量）"的指标，PM 文件解析时才会存储该指标的数据。

- 启用状态记录在 `enabled_pm_indicators` 表中（`operator_code` + `indicator_id` 联合主键）
- 服务启动后在内存中维护启用指标的缓存，读多写少
- 启用/禁用操作完成后，通过 Redis 消息发布通知各节点刷新内存缓存

### 2.6 设备类型区分（ENB / GSM）

部分接口通过 `device_type` 参数区分设备类型：

| `device_type` 值 | 数据来源 |
|---|---|
| `ENB`（或不传） | `indicator_group` 表（ENB 4G 指标） |
| `GSM` | `indicator_group_gsm` 表（GSM 2G 指标） |

新建指标组时 `device_type` 为必填参数，只接受 `ENB` 或 `GSM`（大小写不敏感）。

修改/删除指标组时，系统根据 `group_id` 自动检测属于哪张表，无需传 `device_type`。

---

## 3. 关键数据库表

### 3.1 indicator_group / indicator_group_gsm（指标功能集）

```sql
CREATE TABLE `indicator_group` (
  `id`            varchar(32)  NOT NULL,  -- UUID，主键
  `en_name`       varchar(50),            -- 英文名
  `cn_name`       varchar(50),            -- 中文名
  `operator_code` varchar(100),           -- 运营商编码，NULL 表示系统内置
  `is_build_in`   char(1),                -- 1=内置，0=自定义
  `description`   varchar(500),           -- 描述
  `parent_id`     varchar(32)             -- 父节点 ID；根节点的 id = parent_id
);
```

`indicator_group_gsm` 表结构与 `indicator_group` 完全相同，存储 GSM 指标功能集。

### 3.2 perf_indicators（指标）

```sql
CREATE TABLE `perf_indicators` (
  `id`                varchar(20)   NOT NULL,  -- 指标 ID，主键
  `en_name`           varchar(200),            -- 英文名
  `cn_name`           varchar(200),            -- 中文名
  `en_description`    text,                    -- 英文描述
  `cn_description`    text,                    -- 中文描述
  `group_id`          varchar(32),             -- 所属指标功能集 ID
  `operator_code`     varchar(100),            -- 运营商编码，NULL 表示系统内置
  `unit_id`           varchar(15),             -- 单位 ID，关联 indicator_unit 表
  `updator`           varchar(200),            -- 最后更新人
  `uptime`            datetime,                -- 最后更新时间（UTC）
  `is_build_in`       char(1),                 -- 1=内置，0=自定义
  `is_counter`        char(1),                 -- 1=Counter，0=KPI 指标
  `arithmetic`        text,                    -- 计算公式（Counter 时存放 Counter 名称）
  `statis_type`       varchar(10),             -- 统计类型（sum/avg 等）
  `calculating_status` varchar(10),            -- 重新计算的进度（如 "0%"、"100%"）
  `product_types`     varchar(200),            -- 适用产品类型
  `indicator_level`   varchar(20)  DEFAULT 'device'  -- 指标级别：device/plmn/both
);
```

ENB、GSM 指标共用此表，通过 `group_id` 关联到不同的指标功能集表。

### 3.3 rela_platform_indicator_formula（平台-指标公式关系）

```sql
CREATE TABLE `rela_platform_indicator_formula` (
  `platform_name`  varchar(50),   -- 平台名称（如 436Q、Intel_CR、custom）
  `indicator_id`   varchar(20),   -- 指标 ID
  `formula`        text           -- 该平台下的展开公式（Counter 已被替换为原始公式）
);
```

当一个 KPI 指标的公式中引用了其他自定义指标时，此表记录各平台下展开后的原始 Counter 公式，供 PM 文件解析时直接使用。新建/修改指标后，该表记录会自动同步更新，Redis 缓存也会随之刷新。

### 3.4 enabled_pm_indicators（已启用指标）

```sql
CREATE TABLE `enabled_pm_indicators` (
  `operator_code`  varchar(100) NOT NULL,
  `indicator_id`   varchar(20)  NOT NULL,
  PRIMARY KEY (`operator_code`, `indicator_id`)
);
```

只有在此表中有记录的指标，PM 文件解析时才会存储其数据。

### 3.5 perf_template_rel_arithmetic（模板-指标关联）

```sql
-- temp_id: 模板 ID；indicator_id: 指标 ID
CREATE TABLE `perf_template_rel_arithmetic` (
  `temp_id`       varchar(32) NOT NULL,
  `indicator_id`  varchar(20) NOT NULL
);
```

删除指标或指标功能集时，会先查此表判断是否有模板正在使用，有则阻止删除并返回 `"template is using"`。

### 3.6 perf_cust_name（指标自定义名称）

```sql
CREATE TABLE `perf_cust_name` (
  `operator_code`  varchar(100) NOT NULL,
  `perf_id`        varchar(20)  NOT NULL,  -- 指标 ID
  `cust_name`      varchar(200),           -- 运营商自定义的指标显示名称
  PRIMARY KEY (`operator_code`, `perf_id`)
);
```

运营商可以为内置指标设置自定义显示名称，不影响系统内置的 `en_name`/`cn_name`。

### 3.7 indicator_threshold（指标门限）

```sql
CREATE TABLE `indicator_threshold` (
  `indicator_id`      varchar(20),
  `threshold_period`  varchar(10),   -- 门限周期
  `threshold_color`   varchar(10),   -- 颜色
  `threshold_low`     varchar(20),   -- 下限
  `threshold_high`    varchar(20),   -- 上限
  `threshold_level`   varchar(10)    -- 级别：minor（一般）/ major（严重）
);
```

查看指标详情时，同步返回门限信息（分 `general`/`serious` 两档）。

### 3.8 GNB 专用表

GNB 使用一组独立的数据库表，结构与 ENB 对应表类似：

| 表名 | 对应 ENB 表 | 说明 |
|---|---|---|
| `gnb_indicator_group` | `indicator_group` | GNB 指标功能集 |
| `gnb_perf_indicators` | `perf_indicators` | GNB 指标（注意：无 `product_types` 和 `indicator_level` 字段） |
| `gnb_rela_platform_indicator_formula` | `rela_platform_indicator_formula` | GNB 平台-指标公式关系 |

---

## 4. 指标 ID 命名规范

| 类型 | ID 格式 | 示例 | 说明 |
|---|---|---|---|
| 自定义 KPI 指标 | `{cloudKey}K90000{序号}` | `defaultK900000001` | `cloudKey` 来自 `operators_info.cloud_key`；序号 4 位，上限 9999 |
| 自定义 Counter | `D00000{序号}` | `D000000001` | 序号 4 位，上限 9999 |

内置指标的 ID 由系统初始化数据决定，代码层面无强制格式约束。

---

## 5. 接口详细说明

### 5.1 页面跳转接口

以下接口仅用于返回页面视图，无业务逻辑，新实现可忽略此节：

| 接口 URL | 说明 |
|---|---|
| `GET /cell/perfmgmt/kpimanage/goKpiManagementPage` | 指标管理主页面入口，ENB/GNB 共用，需权限校验 |
| `GET /pm/indicatormg/goAddIndicatorGroupPage` | 新建指标功能集页面 |
| `GET /pm/indicatormg/goModifyGroupPage` | 修改指标功能集页面 |
| `GET /pm/indicatormg/goModifyKPIPage` | 新建/修改指标页面 |
| `GET /pm/indicatormg/viewKPIInfo` | 查看指标详情页面；参数：`isBasic`（是否内置）、`isCustomView` |

---

### 5.2 指标功能集接口

#### 查询指标功能集树

```
POST /pm/indicatormg/getIndicatorGroupTree
```

**参数：**

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `product_type` | String | 否 | 产品类型，多个用逗号分隔 |
| `device_type` | String | 否 | 设备类型，`ENB` / `GSM`，多个用逗号分隔；不传则同时返回 ENB 和 GSM 两棵树 |
| `searchText` | String | 否 | 模糊搜索关键字 |
| `noKpiShowGroup` | String | 否 | `1`=显示无指标的分组，`0`=只显示有指标的分组 |
| `language` | String | 否 | `zh` / `en` |

**返回值：** 树节点数组。每个根节点代表一种设备类型（ENB 或 GSM），根节点下包含各指标功能集子节点。每个节点包含 `id`、`text`（显示名）、`device_type`、`children`（子节点列表）等字段。

> 子节点构建时，关联统计各组的指标数量，并支持按产品类型和搜索词过滤。

---

#### 新建指标功能集

```
POST /pm/indicatormg/addIndicatorGroup
```

**参数：**

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `catagoryName` | String | 是 | 功能集名称 |
| `description` | String | 否 | 描述 |
| `device_type` | String | 是 | `ENB` 或 `GSM`（大小写不敏感），决定写入哪张表 |

**返回值：** `{ "success": true, "message": "..." }`（`success` 为 bool，`message` 为描述字符串）

**业务规则：**
- `device_type` 为必填，不传或传入非 `ENB`/`GSM` 值时返回错误
- 同一运营商下名称不能重复（跨 ENB/GSM 表分别检查）
- ID 使用 UUID 生成，`parent_id` 设为对应表的根节点 ID
- 会记录操作日志

---

#### 获取指标功能集信息

```
POST /pm/indicatormg/getIndicatorGroupInfo
参数：catagoryId（指标功能集 ID）
```

先查 `indicator_group`，未找到则查 `indicator_group_gsm`，返回 `catagoryName`、`description`。

---

#### 修改指标功能集

```
POST /pm/indicatormg/modifyIndicatorGroup
```

**参数：** `catagoryId`、`catagoryName`、`description`

**业务规则：**
- 系统根据 `catagoryId` 自动识别属于 ENB 还是 GSM 表
- 名称在同一运营商下不能与其他功能集重复

---

#### 删除指标功能集

```
POST /pm/indicatormg/delIndicatorGroup
参数：catagoryId
```

**删除前置检查（任一条件满足则阻止删除）：**
1. 该功能集下有指标被 KPI 报表模板使用 → 返回 `"template is using"`

**删除步骤（事务内）：**
1. 删除 `rela_platform_indicator_formula` 中该功能集下所有指标的平台公式关系
2. 删除 `perf_indicators` 中该功能集下的所有指标
3. 删除 `indicator_group`（或 `indicator_group_gsm`）中的功能集记录
4. 清除相关 Redis 缓存

---

### 5.3 指标列表查询接口

#### 分页查询指标列表（指标管理页面主表格）

```
POST /pm/indicatormg/getIndicatorListByPage
```

**参数：**

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `catagoryId` | String | 否 | 指标功能集 ID，传则只返回该组下的指标 |
| `searchText` | String | 否 | 模糊搜索，匹配指标名称或指标 ID |
| `product_type` | String | 否 | 产品类型，多个用逗号分隔 |
| `isEnable` | String | 否 | `1`=只返回已启用，`0`=只返回已禁用，不传=全部 |
| `indicatorLevel` | String | 否 | `device` / `plmn`，不传=全部 |
| `device_type` | String | 否 | `gsm`=查 GSM 指标；不传或其他值=查 ENB 指标 |
| `timeZone` | String | 是 | 时区偏移分钟数，用于转换更新时间显示 |
| `page` | int | 是 | 当前页码 |
| `rows` | int | 是 | 每页条数 |
| `sort` | String | 否 | 排序字段 |
| `order` | String | 否 | 排序方向 |

**返回值：** 分页数据，含 `total`（总数）和 `rows`（当前页列表）。每条记录包含 `kpiId`、`kpiName`、`catagoryName`、`product_type`、`isEnable`、`indicatorLevel`、`unit`、`custName` 等字段。

**ENB / GSM 分支逻辑：**
- `device_type=gsm`：查询 GSM 指标
- 其他：查询 ENB 指标

**结果后处理：** `product_type` 字段会根据当前运营商接入的设备产品类型列表进行过滤，只保留实际接入的产品类型显示。

---

#### 获取有效指标列表

```
POST /pm/indicatormg/getEffectiveIndicators
```

**参数：**

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `product_type` | String | 是 | 产品类型，逗号分隔 |
| `indicator_ids` | String | 是 | 指标 ID 列表，逗号分隔；数量不能超过配置项 `kpiTemplatePageSelectedIndicatorNum` |
| `indicator_level` | String | 是 | `device` 或 `plmn` |
| `language` | String | 否 | `zh` / `en`，默认 `en` |

**返回值：** 有效指标列表，每条包含 `kpiId`、`kpiName`、`catagoryName`、`product_type`、`isEnable` 字段。

---

### 5.4 GNB 指标功能集接口

GNB 指标功能集接口路径前缀为 `/gnb/pm/indicatormg`，与 ENB/GSM 的 `/pm/indicatormg` 平行。接口语义相同，但数据读写 GNB 专用表。

| 接口 URL | 方法 | 说明 | 与 ENB/GSM 对应接口的差异 |
|---|---|---|---|
| `/gnb/pm/indicatormg/getIndicatorGroupTree` | POST | 查询 GNB 指标功能集树 | 数据源为 `gnb_indicator_group` 表；无 `device_type` 参数 |
| `/gnb/pm/indicatormg/addIndicatorGroup` | POST | 新建 GNB 指标功能集 | 无需传 `device_type`，写入 `gnb_indicator_group` 表 |
| `/gnb/pm/indicatormg/getIndicatorGroupInfo` | POST | 获取 GNB 指标功能集信息 | 查 `gnb_indicator_group` 表 |
| `/gnb/pm/indicatormg/modifyIndicatorGroup` | POST | 修改 GNB 指标功能集 | 操作 `gnb_indicator_group` 表 |
| `/gnb/pm/indicatormg/delIndicatorGroup` | POST | 删除 GNB 指标功能集 | 删除前检查：仅检查模板引用和 KPI 公式引用，**不检查告警模板**（ENB/GSM 检查三层） |

#### 查询 GNB 指标功能集树

```
POST /gnb/pm/indicatormg/getIndicatorGroupTree
```

**参数：**

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `product_type` | String | 否 | 产品类型，多个用逗号分隔（`BaiBNX`、`BaiBNQ`、`CHINA_TELECOM`） |
| `searchText` | String | 否 | 模糊搜索关键字 |
| `noKpiShowGroup` | String | 否 | `1`=显示无指标的分组 |
| `isCustomize` | String | 否 | 是否只看自定义指标 |
| `language` | String | 否 | `zh` / `en` |

**返回值：** 与 ENB/GSM 树结构相同（含 `id`、`text`、`children` 等字段），但无 `device_type` 字段（GNB 只有一棵树）。

---

### 5.5 GNB 指标列表查询接口

#### 分页查询 GNB 指标列表（指标管理页面主表格）

```
POST /gnb/pm/indicatormg/getIndicatorListPageData
```

**参数：**

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `catagoryId` | String | 否 | 指标功能集 ID |
| `searchText` | String | 否 | 模糊搜索，匹配指标名称或指标 ID |
| `product_type` | String | 否 | 产品类型，多个用逗号分隔 |
| `isCustomize` | String | 否 | 是否只看自定义指标 |
| `tempId` | String | 否 | 按模板过滤 |
| `timeZone` | String | 是 | 时区偏移分钟数 |
| `page` | int | 是 | 当前页码 |
| `rows` | int | 是 | 每页条数 |
| `sort` | String | 否 | 排序字段 |

**与 ENB 接口的关键差异：**
- 数据源为 `gnb_perf_indicators`、`gnb_indicator_group`、`gnb_rela_platform_indicator_formula` 表
- 无 `isEnable`、`indicatorLevel`、`device_type` 参数
- 无产品类型结果过滤（ENB 会根据运营商接入设备列表过滤 `product_type`）

---

### 5.6 GNB 指标 CRUD 接口

GNB 指标 CRUD 接口路径前缀为 `/gnb/pm/indicatormg`，与 ENB/GSM 接口的主要差异如下：

| 接口 URL | 说明 | 与 ENB/GSM 的差异 |
|---|---|---|
| `POST /gnb/pm/indicatormg/getIndicatorInfo` | 查看 GNB 指标详情 | 数据来自 `gnb_perf_indicators`；返回值无 `indicatorLevel`、`indicatorProductRela` 字段 |
| `POST /gnb/pm/indicatormg/addOrModifyIndicator` | 新建/修改 GNB 指标 | 无 `product_type`、`isEnable`、`indicatorLevel` 参数；公式引用范围为 GNB 指标；保存后触发 GNB 重新计算 |
| `POST /gnb/pm/indicatormg/delIndicator` | 删除 GNB 指标 | 删除前检查：模板引用 + KPI 公式引用（**无**告警模板检查） |
| `POST /gnb/pm/indicatormg/exportAllIndicator` | 导出 GNB 指标列表 | 导出列无"等级"（`indicatorLevel`）列 |
| `POST /gnb/pm/indicatormg/updateBaseKpiCustName` | 修改 GNB 基础指标自定义名称 | 逻辑与 ENB 相同，写 `perf_cust_name` 表 |
| `POST /gnb/pm/indicatormg/updateGnbIndicatorsName` | 修改 GNB Counter 名称 | ENB 对应接口为 `updateEnbIndicatorsName` |

**GNB 新建/修改指标流程与 ENB 的主要差异：**
- 公式引用映射从 GNB 指标表（`gnb_perf_indicators`）获取，确保公式只引用 GNB 指标
- KPI 指标写库：写入 GNB 指标表
- Counter 写库：通过独立入口处理，写入 `gnb_perf_indicators` 表
- 重新计算：触发 GNB 历史数据重算（ENB 触发 ENB 重算）
- **无**启用/禁用步骤（ENB 保存后根据 `isEnable` 参数调用启停接口）

---

#### 查看指标详情

```
POST /pm/indicatormg/getIndicatorInfo
参数：kpiId
```

**返回值关键字段：**

| 字段 | 说明 |
|---|---|
| `kpiId` | 指标 ID |
| `kpiName` | 指标名称 |
| `catagoryId` / `catagoryName` | 所属功能集 |
| `unit` | 单位 |
| `statisType` | 统计类型 |
| `indicatorLevel` | 指标级别 |
| `product_type` | 适用产品类型 |
| `definition` | 描述 |
| `arithmetic` | 计算公式（拆分为 `keys`/`values`/`names` 三个数组，方便前端展示） |
| `isEnable` | `1`=已启用，`0`=未启用 |
| `custName` | 自定义显示名称 |
| `indicatorProductRela` | 公式中各子指标的产品类型关系映射，如 `{"C0001": "QRTB/ALL"}` |
| `generalColor`/`seriousColor` 等 | 门限信息 |

**`arithmetic` 字段的处理：**
1. 先获取全量指标 ID → 名称映射
2. 再将公式字符串按运算符拆分为数组（`keys`=ID 列表，`names`=名称列表）

---

#### 新建 / 修改指标

```
POST /pm/indicatormg/addOrModifyIndicator
```

**参数：**

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `kpiId` | String | 否 | 指标 ID；为空=新建，非空=修改 |
| `indicatorType` | String | 否 | `counter`=新建/修改 Counter；其他=KPI 指标 |
| `kpiName` | String | 是 | 指标名称 |
| `catagoryId` | String | 是 | 所属功能集 ID（必填，决定 ENB/GSM 类型） |
| `unit` | String | 是 | 单位 ID |
| `statisType` | String | 是 | 统计类型 |
| `arithmetic` | String | 是 | 计算公式（Counter 类型时为 Counter 名称） |
| `product_type` | String | 是 | 适用产品类型 |
| `isEnable` | String | 是 | `1`=启用，`0`=禁用 |
| `definition` | String | 否 | 描述 |
| `custName` | String | 否 | 自定义显示名称 |
| `indicatorLevel` | String | 否 | `device`/`plmn`，默认 `device` |

**业务流程：**

```
1. 参数提取：获取 catagoryId、arithmetic、indicatorType 等参数

2. 公式合法性校验（仅 KPI 指标，Counter 跳过此步）：
   - 获取同功能集下所有指标的 ID→公式映射
     （catagoryId 决定查询 ENB 还是 GSM 表，确保公式引用的是同类型指标）
   - 逐 token 解析公式，将指标 ID 替换为 "1" 后用数值表达式求值器验证合法性

3. Counter 名称唯一性校验（仅新建 Counter）：
   - 校验名称是否已存在

4. 写入数据库
   - 新建指标：查询当前最大自定义指标序号，生成新 ID
     （自定义 KPI：{cloudKey}K90000{序号}；自定义 Counter：D00000{序号}）
   - 写入 perf_indicators 表（UPSERT 语句）
   - 更新 rela_platform_indicator_formula 表中的平台公式关系
   - 更新 Redis 缓存（indicator_unit_map、indicator_statis_type_map）

5. 处理启用状态：
   - isEnable=1 → 启用指标
   - isEnable=0 → 禁用指标

6. 判断是否需要重新计算：
   - 新建指标：始终需要重算
   - 修改指标：仅当 arithmetic 或 statisType 发生变化时才重算
   - 触发重算：先更新 calculating_status 为 "0%"，再异步触发历史数据重新计算
```

**返回值：** `{ "success": true, "message": "指标ID" }`（`success` 为 bool，成功时 `message` 为指标 ID）

**错误码说明：**

| 错误信息 | 含义 |
|---|---|
| `Expression is invalid` | 计算公式非法（数值表达式求值失败） |
| `counter name already exists` | Counter 名称已存在 |
| `Category ID is required` | 未传 `catagoryId` |
| `customize kpi number is over 9999` | 自定义指标数量已达上限 9999 |

---

#### 删除指标

```
POST /pm/indicatormg/delIndicator
参数：kpiId
```

**删除前置检查（任一条件满足则阻止删除）：**

| 返回值 | 含义 |
|---|---|
| `"template is using"` | 指标被 KPI 报表模板引用 |
| `"alarm template is using"` | 指标被 KPI 告警模板引用（`perf_alarm_threshold` 表） |
| `"kpi formuals is using"` | 指标被其他 KPI 指标的计算公式引用（注意：`formuals` 为原始代码中的拼写，实现时须与此保持一致） |

**删除步骤：**
1. 删除 `perf_indicators` 记录
2. 删除 `rela_platform_indicator_formula` 中的平台公式关系
3. 从 `enabled_pm_indicators` 表删除该指标的启用记录
4. 清除相关 Redis 缓存

---

#### 修改基础指标自定义名称

```
POST /pm/indicatormg/updateBaseKpiCustName
参数：kpiId、custName、isEnable
```

将自定义名称写入 `perf_cust_name` 表（UPSERT）。同时根据 `isEnable` 参数同步处理启用/禁用状态。

---

#### 修改 Counter 名称

```
POST /pm/indicatormg/updateEnbIndicatorsName
参数：kpiId、indicatorName
```

修改 ENB Counter 的名称。修改前校验新名称的唯一性。

---

### 5.7 指标启停接口

#### 启用指标（测量）

```
POST /cell/perfmgmt/kpimanage/enableIndicator
参数：indicatorIds（逗号分隔的指标 ID 列表）
```

效果：在 `enabled_pm_indicators` 表插入记录，并通过 Redis 消息通知各节点刷新内存缓存。

---

#### 禁用指标（取消测量）

```
POST /cell/perfmgmt/kpimanage/disableIndicator
参数：indicatorIds（逗号分隔的指标 ID 列表）
```

效果：从 `enabled_pm_indicators` 表删除记录，并通过 Redis 消息通知各节点刷新内存缓存。

---

#### 查询指标是否被模板关联

```
GET /cell/perfmgmt/kpimanage/isIndicatorInTemplate
参数：indicatorIds（逗号分隔的指标 ID 列表）
```

**返回值示例：** `{"K900000001": 1, "K900000002": 0}`，`1` 表示已被模板关联，`0` 表示未关联。

用途：前端在执行"取消测量"前调用此接口，若有指标已被模板关联，则弹出确认对话框提示用户。

---

### 5.8 导出接口

#### 导出指标列表（CSV）

```
POST /pm/indicatormg/exportAllIndicator
```

**参数：**

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `timeZone` | String | 是 | 时区偏移分钟数，用于生成文件名和转换时间 |
| `searchText` | String | 否 | 模糊搜索关键字 |
| `product_type` | String | 否 | 产品类型，逗号分隔 |
| `indicatorLevel` | String | 否 | `device` 或 `plmn`，其他值忽略 |

**导出字段（CSV 列顺序）：** 指标ID、指标名称、产品类型、自定义名称、等级、单位、类型（内置/自定义）、更新人、更新时间、计算公式

**编码：** GBK（兼容 Excel 直接打开中文内容）

---

### 5.9 辅助查询接口

#### 获取指标单位列表

```
GET /pm/indicatormg/getIndicatorUnitList
```

返回单位列表，用于新建/修改指标页面的下拉框。

---

#### 获取指标功能集下拉框数据

```
POST /pm/indicatormg/getIndicatorGroupList
参数：isShowAll（1=在列表头部加入"全部"选项）、tempId（按模板过滤）、noKpiShowGroup
```

返回适用于下拉框的指标功能集列表。只查 ENB 的 `indicator_group` 表（不含 GSM）。过滤掉 `Customize[DEPRECATED]` 分组。

---

#### 获取指标类型信息

```
POST /pm/indicatormg/getIndicatorTypes
参数：catagoryId、searchText
```

返回当前功能集下存在的指标类型（内置指标 / 自定义指标），用于前端类型过滤下拉框。

---

## 6. 关键业务逻辑

### 6.1 指标功能集树的构建

指标功能集树查询根据 `device_type` 参数决定构建哪棵树（ENB、GSM 或同时构建两棵）：

1. 查询对应表的根节点（`id = parent_id AND is_build_in = '1'`）
2. 递归构建子节点列表：
   - 关联统计各组的指标数量
   - 支持按产品类型、指标 ID 或名称模糊搜索过滤
   - `noKpiShowGroup` 控制是否显示指标数为 0 的分组
3. 每个根节点附带 `device_type` 字段（`"ENB"` 或 `"GSM"`），前端通过此字段区分行为

### 6.2 计算公式验证流程

新建/修改 KPI 指标（非 Counter）时，后端对公式进行合法性验证：

```
输入：arithmetic 公式字符串（如 "C000000001/(C000000002+C000000003)*100"）

1. 获取同类型指标的 ID→公式映射
   （catagoryId 决定查 ENB 还是 GSM 表，防止跨类型引用）

2. 按运算符（+-*/()等）将公式拆分为 token 列表

3. 逐 token 处理：
   - 是数字或运算符 → 原样保留
   - "Duration" → 替换为 "1"（特殊关键字，表示采集周期）
   - 是指标 ID 且存在于映射表中 → 替换为 "1"（用于计算验证）；
     若该指标是自定义指标（含 K90000），还需递归展开其公式
   - 不在映射表中 → 标记公式非法，返回 "--"

4. 对替换后的纯数字表达式进行求值验证：
   - 求值成功 → 公式合法
   - 求值失败 → 公式非法，返回错误 "Expression is invalid"

5. 统计公式中指标数量：若指标数为 0（全为数字常量），也视为非法

6. 所有 token 处理完成后，若公式只包含一个指标 ID（无任何运算符），写库时标记 is_counter=1
```

### 6.3 重新计算触发条件

新建/修改指标后，满足以下任一条件且指标处于启用状态时，触发重新计算历史数据：

- **新建指标**：始终触发
- **修改指标**：`arithmetic`（计算公式）或 `statis_type`（统计类型）与修改前相比发生了变化

触发流程：
1. 将 `perf_indicators.calculating_status` 更新为 `"0%"`
2. 异步触发历史数据重新计算

### 6.4 删除指标的前置检查

删除指标时，按顺序执行三层检查：

```
第一层：查询 perf_template_rel_arithmetic 表
  → 若指标被任意报表模板引用 → 返回 "template is using"，阻止删除

第二层：查询 perf_alarm_threshold 表
  → 若指标被任意 KPI 告警模板引用 → 返回 "alarm template is using"，阻止删除

第三层：扫描 perf_indicators 表中所有 is_counter=0 的指标的 arithmetic 字段
  → 若当前指标 ID 出现在其他指标的计算公式中 → 返回 "kpi formuals is using"，阻止删除
  （注意：formuals 为原始代码中的拼写；同时检查 arithmetic = 指标ID 本身的情况，覆盖 isCounter=1 的场景）
```

### 6.5 指标产品类型与平台公式关系

保存指标时，同步维护 `rela_platform_indicator_formula` 表：

1. 解析公式中的所有 Counter ID
2. 查询每个 Counter 在各平台（`platform_name`）下的原始公式
3. 将 KPI 公式中的 Counter 替换为对应平台的原始公式，生成各平台的展开公式
4. 批量写入 `rela_platform_indicator_formula` 表
5. 更新 Redis 缓存（供 PM 文件解析时快速查询）

对于自定义指标的修改，还会先删除旧的平台关系记录，再重新插入，避免历史残留数据干扰。

### 6.6 Redis 缓存更新时机

| 缓存 Key | 内容 | 更新时机 |
|---|---|---|
| `indicator_unit_map` | 指标 ID → 单位 | 新建/修改指标后 |
| `indicator_statis_type_map` | 指标 ID → 统计类型 | 新建/修改指标后 |
| `enb_custom_counter_map` | Counter 名称 → Counter ID | 新建/修改 Counter 后 |
| 平台公式缓存 | 平台名 → 指标展开公式 | 新建/修改/删除指标后 |

缓存删除（旧数据清理）在写库前执行，写库成功后再写入新缓存。
