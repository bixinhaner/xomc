# KPI 定时汇总流程说明

## 1. 功能概述

系统按小时、天、周、月四种时间粒度，将存储在 PostgreSQL 中的 15min 原始 KPI 数据定时聚合为高粒度汇总数据，写入对应的 PostgreSQL 表，供 KPI 报表和首页看板展示使用。

---

## 2. 核心概念

### 2.1 设备类型

| 设备类型 | 说明 |
|---------|------|
| ENB | 4G 小基站 |
| GSM | 2G/GSM 基站（BSC 平台） |
| GNB | 5G 小基站 |

ENB 和 GSM 共用同一套定时任务（`QuartzStatisticsKPIDataJob_*`），GNB 使用独立的定时任务（`GnbQuartzStatisticsKPIDataJob_*`）。

### 2.2 统计粒度（period）

| 数值 | 粒度 | 说明 |
|-----|------|------|
| 60 | 小时 | 每小时触发一次，无条件执行 |
| 1440 | 天 | 每天触发一次，无条件执行 |
| 10080 | 周 | 每周触发一次，受 `kpiWeekAndMonthSwitch` 开关控制 |
| 43200 | 月 | 每天触发一次，仅在当月第 1 天或最后一天执行，受 `kpiWeekAndMonthSwitch` 开关控制 |

### 2.3 指标级别（仅 ENB/GSM 适用）

| 级别 | 说明 |
|-----|------|
| DEVICE | 设备级指标，存储在按天分表的设备维度表中 |
| PLMN | 运营商级指标，存储在按天分表的 PLMN 维度表中 |

GNB 不区分指标级别，只有一套表。

### 2.4 指标统计类型

| 统计类型 | 说明 |
|--------|------|
| sum | 对多条 15min 数据的指标值求和 |
| avg | 对多条 15min 数据的指标值求平均 |
| max | 取多条 15min 数据的最大值 |
| min | 取多条 15min 数据的最小值 |
| pct | 公式型指标，先对公式中引用的各子 counter 按其自身 statis_type 聚合，再将聚合结果代入公式计算最终值 |

pct 型指标的公式格式说明见第 5.6 节。

### 2.5 指标值特殊值

| 值 | 含义 |
|----|------|
| `-` | 设备不支持该指标（原始数据中无此指标字段，或值为 `-`） |
| `N/A` | 指标计算出错（原始数据为 `N/A`，或公式计算结果为 Infinity / NaN） |

### 2.6 系统开关

`kpiWeekAndMonthSwitch`：存储在 PostgreSQL `sys_settings` 表中，值为 `1` 时开启周和月粒度的汇总统计。该值每 5 分钟从数据库同步到内存缓存中。

---

## 3. 定时任务一览

| 任务名称 | 设备类型 | 粒度 | 开关控制 | 额外触发条件 |
|---------|---------|------|---------|------------|
| QuartzStatisticsKPIDataJob_60 | ENB/GSM | 小时 | 无 | 无 |
| QuartzStatisticsKPIDataJob_1440 | ENB/GSM | 天 | 无 | 无 |
| QuartzStatisticsKPIDataJob_10080 | ENB/GSM | 周 | `kpiWeekAndMonthSwitch=1` | 无 |
| QuartzStatisticsKPIDataJob_43200 | ENB/GSM | 月 | `kpiWeekAndMonthSwitch=1` | 当天是月第 1 天或最后一天 |
| GnbQuartzStatisticsKPIDataJob_60 | GNB | 小时 | 无 | 无 |
| GnbQuartzStatisticsKPIDataJob_1440 | GNB | 天 | 无 | 无 |
| GnbQuartzStatisticsKPIDataJob_10080 | GNB | 周 | `kpiWeekAndMonthSwitch=1` | 无 |
| GnbQuartzStatisticsKPIDataJob_43200 | GNB | 月 | `kpiWeekAndMonthSwitch=1` | 当天是月第 1 天或最后一天 |

所有任务取服务器 UTC 时间，秒、毫秒清零后作为统计时间点。

---

## 4. 各粒度汇总执行步骤

### 4.1 小时粒度（60）

**ENB/GSM 执行步骤：**

1. 执行基站维度 KPI 汇总（period=60，ENB/GSM 设备）
2. 执行组维度 KPI 汇总（period=60，ENB/GSM 设备）
3. 将数据结束时间写入 `perf_statis_data_end_time`，`device_type=enb`
4. 执行首页看板统计（ENB 设备，小时粒度）
5. 执行首页看板统计（GSM 设备，小时粒度）
6. 执行自定义看板统计（ENB 设备，小时粒度，出错不中断整体流程）
7. 执行自定义看板统计（GSM 设备，小时粒度，出错不中断整体流程）

**GNB 执行步骤：**

1. 执行基站维度 KPI 汇总（period=60，GNB 设备）
2. 执行组维度 KPI 汇总（period=60，GNB 设备）
3. 将数据结束时间写入 `perf_statis_data_end_time`，`device_type=gnb`
4. 执行首页看板统计（GNB 设备，小时粒度）
5. 执行自定义看板统计（GNB 设备，小时粒度，出错不中断整体流程）

### 4.2 天粒度（1440）

**ENB/GSM 执行步骤：**

1. 执行基站维度 KPI 汇总（period=1440，ENB/GSM 设备）
2. 执行组维度 KPI 汇总（period=1440，ENB/GSM 设备）
3. 将数据结束时间写入 `perf_statis_data_end_time`，`device_type=enb`
4. 执行首页看板统计（ENB 设备，天粒度）
5. 执行首页看板统计（GSM 设备，天粒度）
6. 执行首页看板 Top10 统计（ENB 设备）
7. 执行首页看板 Bottom10 统计（ENB 设备）
8. 执行首页看板 Top10 统计（GSM 设备）
9. 执行首页看板 Bottom10 统计（GSM 设备）
10. 执行自定义看板统计（ENB/GSM 设备，天粒度，含 Top10/Bottom10，出错不中断整体流程）

**GNB 执行步骤：**

1. 执行基站维度 KPI 汇总（period=1440，GNB 设备）
2. 执行组维度 KPI 汇总（period=1440，GNB 设备）
3. 将数据结束时间写入 `perf_statis_data_end_time`，`device_type=gnb`
4. 执行首页看板统计（GNB 设备，天粒度）
5. 执行首页看板 Top10 统计（GNB 设备）
6. 执行首页看板 Bottom10 统计（GNB 设备）
7. 执行自定义看板统计（GNB 设备，天粒度，含 Top10/Bottom10，出错不中断整体流程）

### 4.3 周粒度（10080）

须 `kpiWeekAndMonthSwitch=1` 才执行，否则跳过。

**ENB/GSM 执行步骤：**

1. 执行基站维度 KPI 汇总（period=10080，ENB/GSM 设备）
2. 执行组维度 KPI 汇总（period=10080，ENB/GSM 设备）
3. 将数据结束时间写入 `perf_statis_data_end_time`，`device_type=enb`

**GNB 执行步骤：**

1. 执行基站维度 KPI 汇总（period=10080，GNB 设备）
2. 执行组维度 KPI 汇总（period=10080，GNB 设备）
3. 将数据结束时间写入 `perf_statis_data_end_time`，`device_type=gnb`

周粒度不执行 Dashboard（首页看板）统计。

### 4.4 月粒度（43200）

须同时满足以下两个条件才执行：
- `kpiWeekAndMonthSwitch=1`
- 当天（UTC 日期）是本月第 1 天，**或者**是本月最后一天

**ENB/GSM 执行步骤：**

1. 执行基站维度 KPI 汇总（period=43200，ENB/GSM 设备）
2. 执行组维度 KPI 汇总（period=43200，ENB/GSM 设备）
3. 将数据结束时间写入 `perf_statis_data_end_time`，`device_type=enb`

**GNB 执行步骤：**

1. 执行基站维度 KPI 汇总（period=43200，GNB 设备）
2. 执行组维度 KPI 汇总（period=43200，GNB 设备）
3. 将数据结束时间写入 `perf_statis_data_end_time`，`device_type=gnb`

月粒度不执行 Dashboard（首页看板）统计。

---

## 5. 基站维度 KPI 汇总算法

### 5.1 整体流程

```
准备指标元数据（从 perf_indicators / gnb_perf_indicators 加载）
  ↓
计算待统计运营商列表（按时区判断是否到统计时间点）
  ↓
查询待统计基站列表（状态=在线，且属于需统计运营商）
  ↓
逐基站处理：
  ↓
  获取基站所属平台类型
  ↓
  构建查询条件（时间范围 + small_cell_code）
  ↓
  查询 15min 原始数据
  ↓
  按分组键分组（ENB: cellId+plmnId；GNB: nrCGI+plmnId）
  ↓
  逐组按指标统计类型聚合（sum/avg/max/min）
  ↓
  计算 pct 公式型指标
  ↓
  写入目标表（直接插入）
```

### 5.2 时间范围计算规则

汇总查询时，时间范围仅限制 15min 原始数据的 `start_time` 字段：

- **minStartTime（下限）**
  - 小时/天/周粒度：`统计时间 - period 分钟`
  - 月粒度：将统计时间转换为运营商时区后向前推 1 个自然月，再转回 UTC
- **maxStartTime（上限）**：`统计时间 - 15 分钟`（因为原始数据是 15min 粒度）

若运营商时区含半小时偏移（如 UTC+03:30），minStartTime 和 maxStartTime 均额外向前偏移对应分钟数，保证运营商本地视角的时间段完整。

### 5.3 运营商过滤规则

各粒度仅对满足条件的运营商执行汇总。判断步骤：

1. 根据统计时间和运营商时区，计算出该运营商的 `minStartTime`（时间范围下限，月粒度按自然月推算，其余粒度为 `统计时间 - period 分钟`）
2. 将 `minStartTime` 转换为运营商本地时间
3. 判断转换后的本地时间是否满足下表条件：

| 粒度 | 判断条件（运营商本地时间） |
|------|------------------------|
| 小时（60） | 无限制，所有运营商参与 |
| 天（1440） | 小时数为 0 |
| 周（10080） | 星期一且小时数为 0 |
| 月（43200） | 每月 1 号且小时数为 0 |

未满足条件的运营商，本次不执行汇总，其基站也不在查询范围内。

**月粒度补充说明：** 月粒度定时任务在 UTC 当月第 1 天和最后一天均会触发，但真正执行汇总的运营商仍须满足"运营商本地时间为月初 0 时整"的条件。由于不同运营商时区不同，UTC 月末最后一天对部分时区正是该运营商的下个月 1 号 0 时，因此最后一天触发是为了覆盖这类跨月时区的运营商。

### 5.4 ENB 设备：DEVICE/PLMN 两批处理

ENB 设备（非 BSC 平台）的指标分两个级别存储，需分别处理：

1. **第一批：DEVICE 级指标**
   - 查询 `pm_15min_yyyy_MM_dd`（设备维度 15min 分表）
   - 只处理 `indicator_level=device` 的指标
   - 结果写入设备维度目标表

2. **第二批：PLMN 级指标**
   - 查询 `pm_15min_plmn_yyyy_MM_dd`（PLMN 维度 15min 分表）
   - 只处理 `indicator_level=plmn` 的指标
   - 结果写入 PLMN 维度目标表

BSC 平台（GSM）：`indicator_level` 可能为 `both`，不区分 DEVICE/PLMN，只查询设备维度分表，处理所有指标。

GNB：不区分级别，只查询固定表 `gnb_pm_value_15`，处理所有指标。

### 5.5 ENB 设备：QA_V3/QA_V4 平台特殊处理

针对 QA_V3 和 QA_V4 平台，在汇总结果写入数据库前，做如下字段替换：

- `K900010013` 的值 ← `C000080027` 的值
- `K900010014` 的值 ← `C000080026` 的值

### 5.6 pct 型指标的计算规则

#### 5.6.1 arithmetic 字段格式

`perf_indicators.arithmetic` 和 `gnb_perf_indicators.arithmetic` 存储指标的计算公式，格式为**由运算符分隔的 counter ID 序列**，例如：

```
C000060011/C000060012*100
C000060001+C000060002
```

公式中可以出现：
- **counter ID**：引用其他指标的聚合值
- **数字**：整数或浮点数常量
- **运算符**：`+` `-` `*` `/` `(` `)`
- **特殊占位符 `Duration`**：代表统计周期秒数，计算时替换为 `period × 60`（如 period=60 时 Duration=3600）

**非 pct 型指标**（counter）：`arithmetic` 字段值与 `id` 字段值相同（即 `arithmetic == id`）。

#### 5.6.2 pct 型指标的判断条件

满足以下条件的指标视为 pct 型，需走公式计算流程：
- `statis_type = 'pct'`
- `arithmetic != id`（公式中引用了其他 counter，而非自身）

#### 5.6.3 计算步骤

1. 对公式中引用的所有子 counter，先按各自的 `statis_type`（sum/avg/max/min）对本周期内的 15min 原始数据进行聚合
2. 将公式字符串中的每个 counter ID 替换为其聚合结果数值；将 `Duration` 替换为 `period × 60`（秒）
3. 对替换后的数值表达式求值，得到最终结果
4. 特殊值处理：
   - 如果任一子 counter 的聚合值为空或为 `-`，则该 pct 指标结果记为 `-`
   - 如果表达式求值结果为 `Infinity`、`-Infinity` 或 `NaN`，结果记为 `N/A`

---

## 6. 组维度 KPI 汇总

组维度汇总是在基站维度汇总完成后执行的，以基站维度汇总数据作为源数据，按设备组（device_group）聚合。

### 6.1 运营商过滤规则

与基站维度汇总的规则相同（见 5.3 节），仅对已到统计时间点的运营商执行组维度汇总。

### 6.2 ENB 组维度汇总

- 分两次执行：第一次汇总 DEVICE 级指标，第二次汇总 PLMN 级指标
- 源数据表：对应粒度的基站维度汇总数据表（如小时粒度读 `pm_hour_yyyy_MM_dd`，天粒度读 `pm_value_1440`，见第 8 节）
- 按设备组分两层聚合：
  - **二级设备组**：按 `group_id` 聚合，结果 `group_id` 保持不变
  - **一级设备组**：将其下属所有二级组的数据合并聚合，结果 `group_id` 为 `top_{一级组id}`
- pct 型指标：先聚合子指标，汇总后代入公式计算
- ENB 需检查指标的测量定制开关（`isEnable`），未启用的指标跳过

### 6.3 GNB 组维度汇总

- 只执行一次（不区分 DEVICE/PLMN）
- 源数据表：GNB 专用固定表
- 聚合逻辑与 ENB 相同（二级组 + 一级组两层）
- GNB 不检查指标测量定制开关

### 6.4 目标表

| 粒度 | ENB/GSM 目标表 | GNB 目标表 |
|------|----------------|-------------|
| 小时（60） | `pm_group_hour_yyyy_MM_dd` | `gnb_pm_group_hour_data` |
| 天（1440） | `pm_group_day_data` | `gnb_pm_group_day_data` |
| 周（10080） | `pm_group_week_data` | `gnb_pm_group_week_data` |
| 月（43200） | `pm_group_month_data` | `gnb_pm_group_month_data` |

写入方式为 upsert（按 `group_id + start_time` 匹配，存在则更新，不存在则插入）。

---

## 7. 首页看板（Dashboard）统计

首页看板统计是在基站维度汇总完成后执行的。仅在**小时和天粒度**执行，周/月粒度不执行。

### 7.1 固定指标列表（配置项）

各设备类型的指标列表通过配置文件指定，含默认值：

**ENB 指标（`dashboard.statistics.enb.kpi.list`，可配置）：**

```
K900010006, K900010002, K900010005, K900010027, K900010029,
K900010017, K900010022, K900010021, K900010026, K900010014,
K900010013,
(C000060011+C000060022)*8/#period/1000,
(C000060001+C000060021)*8/#period/1000,
(C000060011+C000060022)/1000/1000,
(C000060001+C000060021)/1000/1000,
C000060216/#period*100
```

**GSM 指标（`dashboard.statistics.gsm.kpi.list`，可配置）：**

```
KGSM0102, KGSM0103, KGSM0101
```

**GNB 指标（`dashboard.statistics.gnb.kpi.list`，可配置）：**

```
KGNB0506, KGNB0505,
C010030006/1000/1000, C010030005/1000/1000,
C010030006*8/#period/1000, C010030005*8/#period/1000
```

其中 `#period` 在计算时替换为实际秒数（`period（分钟数）× 60`，如小时粒度 period=60，则 `#period` 替换为 3600）。

**GNB 复合指标说明：**

`KGNB0505` 和 `KGNB0506` 是复合指标，依赖原子指标计算：

| 复合指标 | 依赖原子指标 |
|---------|------------|
| KGNB0505 | C010000003, C010000006 |
| KGNB0506 | C010000004, C010000007 |

原子指标参与查询但不单独出现在结果中。

### 7.2 统计逻辑（普通汇总）

1. 对每个指标，按 `group_id` 分组聚合，各原子指标按自身统计类型（sum/avg/max）聚合；pct 型指标及特殊指标 `C000060216` 使用 avg 聚合
2. 区分设备类型过滤数据：
   - ENB：过滤条件 `bts_id` 字段不存在
   - GSM：过滤条件 `bts_id` 字段存在（不为 null）
   - GNB：查询专用表，无额外过滤
3. 对公式型指标（含 `/`、`*` 等运算符），代入各原子指标聚合值计算最终结果
4. 结果写入 PostgreSQL 对应表（见 7.4 节）

### 7.3 Top10 / Bottom10 统计（仅天粒度）

1. 查询昨天（`start_time = 统计时间 - 1 天`）的天粒度汇总数据
2. 对每个指标，按指标值降序（Top10）或升序（Bottom10）排序后，取各设备组内最多 10 条
3. 结果写入 PostgreSQL `pm` 库对应表（见 7.4 节）

### 7.4 看板统计目标表（PostgreSQL `pm` 库）

**首页看板（内置）：**

| 类型 | ENB 表名 | GSM 表名 | GNB 表名 |
|-----|---------|---------|---------|
| 小时普通 | `statistics_enb_kpi_hour` | `statistics_gsm_kpi_hour` | `statistics_gnb_kpi_hour` |
| 天普通 | `statistics_enb_kpi_day` | `statistics_gsm_kpi_day` | `statistics_gnb_kpi_day` |
| 天 Top10 | `statistics_kpi_day_top` | `statistics_kpi_day_top` | `statistics_kpi_day_top` |
| 天 Bottom10 | `statistics_kpi_day_bottom` | `statistics_kpi_day_bottom` | `statistics_kpi_day_bottom` |

**自定义看板（Custom Dashboard）：**

| 类型 | 表名 |
|-----|-----|
| 小时普通 | `statistics_dashboard_custom_kpi_hour` |
| 天普通 | `statistics_dashboard_custom_kpi_day` |
| 天 Top10 | `statistics_dashboard_custom_kpi_day_top` |
| 天 Bottom10 | `statistics_dashboard_custom_kpi_day_bottom` |

---

## 8. PostgreSQL 数据表说明

### 8.1 基站维度原始数据表（15min，汇总的数据来源）

| 类型 | 表名规则 | 说明 |
|-----|----------|------|
| ENB 设备维度 | `pm_15min_yyyy_MM_dd` | 按天分表 |
| ENB PLMN 维度 | `pm_15min_plmn_yyyy_MM_dd` | 按天分表 |
| GNB | `gnb_pm_value_15` | 固定表 |

### 8.2 基站维度汇总目标表

| 粒度 | ENB 设备维度 | ENB PLMN 维度 | GNB |
|------|------------|-------------|-----|
| 小时（60） | `pm_hour_yyyy_MM_dd` | `pm_hour_plmn_yyyy_MM_dd` | `gnb_pm_value_60` |
| 天（1440） | `pm_value_1440` | `pm_value_1440_plmn` | `gnb_pm_value_1440` |
| 周（10080） | `pm_value_10080` | `pm_value_10080_plmn` | `gnb_pm_value_week` |
| 月（43200） | `pm_value_43200` | `pm_value_43200_plmn` | `gnb_pm_value_month` |

### 8.3 基站维度汇总表字段说明

| 字段名 | 类型 | 说明 |
|-------|------|------|
| `small_cell_code` | VARCHAR | 基站编码 |
| `serial_number` | VARCHAR | 基站序列号 |
| `group_id` | VARCHAR | 所属设备组 ID |
| `start_time` | TIMESTAMP | 统计周期开始时间（UTC） |
| `end_time` | TIMESTAMP | 统计周期结束时间（UTC） |
| `hour` | VARCHAR | `start_time` 的 UTC 小时数，供前端时维度展示 |
| `week` | VARCHAR | `start_time` 的 UTC 星期几（1=周一，7=周日），供前端周维度展示 |
| `storage_time` | VARCHAR | 数据写入时间（UTC） |
| `bts_id` | VARCHAR | BSC/BTS 基站 ID，仅 GSM 设备有值 |
| `nrCGI` | VARCHAR | 小区全局标识，仅 GNB 设备有值 |
| `plmnId` | VARCHAR | 运营商标识，ENB/GNB 均有值 |
| `cellId` | VARCHAR | 小区 ID，仅 ENB 设备有值 |
| `{indicatorId}` | NUMERIC / VARCHAR | 各指标值；`-` 表示不支持，`N/A` 表示计算出错 |

---

## 9. PostgreSQL 配置与系统表

### 9.1 `sys_settings`（系统参数表）

| 字段 | 类型 | 说明 |
|-----|------|------|
| `S_WORD` | VARCHAR | 参数键名 |
| `S_VALUE` | VARCHAR | 参数值 |

**相关参数：**

| S_WORD | S_VALUE 示例 | 说明 |
|--------|------------|------|
| `kpiWeekAndMonthSwitch` | `1` / `0` | 周/月粒度汇总开关，`1` 开启，`0` 关闭 |

### 9.2 `perf_statis_data_end_time`（数据就绪时间表）

每次汇总完成后，更新本表记录，供报表模块判断某粒度的汇总数据是否已准备好。

写入方式：`INSERT ... ON CONFLICT (device_type, period) DO UPDATE SET ...`。

| 字段 | 类型 | 说明 |
|-----|------|------|
| `device_type` | VARCHAR | 设备类型，值为 `enb` 或 `gnb` |
| `period` | VARCHAR | 粒度，值为 `60` / `1440` / `10080` / `43200` |
| `pm_data_end_time` | VARCHAR | 本次汇总的统计时间点（UTC） |
| `update_time` | VARCHAR | 本记录更新时间（UTC） |

### 9.3 `operators_info`（运营商信息表）

| 字段 | 类型 | 说明 |
|-----|------|------|
| `operator_code` | VARCHAR | 运营商编码（主键） |
| `time_zone` | VARCHAR | 运营商时区，单位分钟（如 `480` 表示 UTC+8，`210` 表示 UTC+3:30） |

### 9.4 `small_cell_infos`（基站信息表）

汇总时用于过滤有效基站，关键字段说明：

| 字段 | 类型 | 说明 |
|-----|------|------|
| `small_cell_code` | VARCHAR | 基站编码（主键） |
| `device_status` | INT | 设备状态，汇总时只取 `device_status = 2`（在线）的基站 |
| `is_gnb` | CHAR(1) | 是否为 5G 基站，`'1'` 表示 GNB，否则为 ENB/GSM |
| `platform_type` | VARCHAR | 基站平台类型，如 `Intel`、`QA_V3`、`QA_V4` 等 |

### 9.5 `rela_device_group_cell`（基站-设备组关联表）

| 字段 | 类型 | 说明 |
|-----|------|------|
| `small_cell_code` | VARCHAR | 基站编码 |
| `group_id` | VARCHAR | 设备组 ID（关联 `device_group.id`） |

用途：通过此表将基站关联到所属设备组，进而关联到运营商。

### 9.6 `device_group`（设备组表）

| 字段 | 类型 | 说明 |
|-----|------|------|
| `id` | VARCHAR | 设备组 ID（主键） |
| `operator_code` | VARCHAR | 所属运营商编码（关联 `operators_info.operator_code`） |

查询某运营商的基站列表 SQL 逻辑（三表关联）：

```sql
SELECT sci.small_cell_code
FROM small_cell.small_cell_infos sci
INNER JOIN small_cell.rela_device_group_cell rdgc ON sci.small_cell_code = rdgc.small_cell_code
INNER JOIN small_cell.device_group dg ON dg.id = rdgc.group_id
WHERE sci.device_status = 2
  AND dg.operator_code = ?
```

### 9.7 `enabled_pm_indicators`（指标测量定制开关表）

记录各运营商已启用的测量定制指标，ENB 组维度汇总时需检查此表（GNB 不检查）。

| 字段 | 类型 | 说明 |
|-----|------|------|
| `operator_code` | VARCHAR(100) | 运营商编码（联合主键） |
| `indicator_id` | VARCHAR(20) | 指标 ID（联合主键） |

**isEnable 判断逻辑：**

查询 `enabled_pm_indicators` 表，检查 `(operator_code, indicator_id)` 记录是否存在。存在即为已启用，不存在即为未启用，跳过该指标的汇总。结果在内存中按运营商缓存，避免频繁查询数据库。

### 9.8 看板统计表（PostgreSQL `pm` 库）

**首页看板普通统计表（`statistics_enb_kpi_hour` 等，结构相同）：**

| 字段 | 类型 | 说明 |
|-----|------|------|
| `device_group_id` | VARCHAR | 设备组 ID |
| `start_time` | VARCHAR | 统计周期开始时间 |
| `end_time` | VARCHAR | 统计周期结束时间 |
| `indicator` | VARCHAR | 指标 ID 或指标公式 |
| `value` | NUMERIC | 指标值 |
| `sn_count` | INT | 该统计周期内的基站数量，用于加权平均 |

写入方式：`INSERT ... ON CONFLICT (device_group_id, start_time, end_time, indicator) DO UPDATE SET ...`。

**Top10 / Bottom10 统计表（`statistics_kpi_day_top` / `statistics_kpi_day_bottom`，结构相同）：**

| 字段 | 类型 | 说明 |
|-----|------|------|
| `device_type` | VARCHAR | 设备类型，值为 `eNB` / `GSM` / `gNB` |
| `device_group_id` | VARCHAR | 设备组 ID |
| `serial_number` | VARCHAR | 基站序列号 |
| `indicator` | VARCHAR | 指标 ID 或指标公式 |
| `value` | NUMERIC | 指标值 |
| `statistics_time` | VARCHAR | 统计时间点 |

数据保留策略：每次统计前删除 24 小时以前的记录。

**自定义看板统计表（`statistics_dashboard_custom_kpi_hour` / `statistics_dashboard_custom_kpi_day`）：**

| 字段 | 类型 | 说明 |
|-----|------|------|
| `device_group_id` | VARCHAR | 设备组 ID |
| `start_time` | VARCHAR | 统计周期开始时间 |
| `end_time` | VARCHAR | 统计周期结束时间 |
| `source_device_type` | VARCHAR | 数据来源设备类型，值为 `enb` / `gsm` / `gnb` |
| `indicator` | VARCHAR | 指标 ID 或指标公式 |
| `value` | NUMERIC | 指标值 |
| `sn_count` | INT | 该统计周期内的基站数量 |

---

## 10. 指标元数据表

### 10.1 `perf_indicators`（ENB/GSM 指标定义表）

存储所有 ENB 和 GSM 指标的元数据，是汇总计算的核心参考表。

| 字段 | 类型 | 说明 |
|-----|------|------|
| `id` | VARCHAR(20) | 指标 ID（主键），如 `K900010006`、`C000060011` |
| `arithmetic` | TEXT | 计算公式（详见第 5.6 节）；counter 型指标此字段值与 `id` 相同 |
| `statis_type` | VARCHAR(10) | 统计类型：`sum` / `avg` / `max` / `min` / `pct` |
| `indicator_level` | VARCHAR(20) | 指标级别：`device`（默认）/ `plmn` / `both`（BSC/GSM 用） |
| `group_id` | VARCHAR(32) | 所属指标分组 ID，关联 `indicator_group` 或 `indicator_group_gsm` |
| `unit_id` | VARCHAR(15) | 单位 ID，关联 `indicator_unit` 表 |
| `en_name` | VARCHAR(200) | 指标英文名 |
| `cn_name` | VARCHAR(200) | 指标中文名 |
| `product_types` | VARCHAR(200) | 适用产品类型 |
| `calculating_status` | VARCHAR(10) | 计算状态 |

**查询指标公式的 SQL 逻辑（兼容 ENB 和 GSM）：**

```sql
SELECT pi.id AS kpiId, pi.arithmetic
FROM small_cell.perf_indicators pi
INNER JOIN small_cell.indicator_unit iu ON iu.id = pi.unit_id
LEFT JOIN small_cell.indicator_group ig ON ig.id = pi.group_id
LEFT JOIN small_cell.indicator_group_gsm gsm_ig ON gsm_ig.id = pi.group_id
WHERE (ig.id IS NOT NULL OR gsm_ig.id IS NOT NULL)
```

注意：使用双 LEFT JOIN 是为了同时兼容 ENB 指标（关联 `indicator_group`）和 GSM 指标（关联 `indicator_group_gsm`），若改用 INNER JOIN 会导致 GSM 指标公式加载失败。

### 10.2 `gnb_perf_indicators`（GNB 指标定义表）

与 `perf_indicators` 结构基本相同，但无 `indicator_level` 字段（GNB 不区分级别），且关联 `gnb_indicator_group` 表。

| 字段 | 类型 | 说明 |
|-----|------|------|
| `id` | VARCHAR(20) | 指标 ID（主键），如 `KGNB0505`、`C010000003` |
| `arithmetic` | TEXT | 计算公式；counter 型指标此字段值与 `id` 相同 |
| `statis_type` | VARCHAR(10) | 统计类型：`sum` / `avg` / `max` / `min` / `pct` |
| `group_id` | VARCHAR(32) | 所属指标分组 ID，关联 `gnb_indicator_group` |
| `unit_id` | VARCHAR(10) | 单位 ID，关联 `indicator_unit` 表 |
| `en_name` | VARCHAR(200) | 指标英文名 |
| `cn_name` | VARCHAR(200) | 指标中文名 |
| `calculating_status` | VARCHAR(10) | 计算状态 |

**查询 GNB 指标公式的 SQL 逻辑：**

```sql
SELECT pi.id AS kpiId, pi.arithmetic
FROM small_cell.gnb_perf_indicators pi
INNER JOIN small_cell.indicator_unit iu ON iu.id = pi.unit_id
INNER JOIN small_cell.gnb_indicator_group ig ON ig.id = pi.group_id
```

### 10.3 平台-指标关联表

基站平台（如 `Intel`、`QA_V3`、`QA_V4` 等）与其支持的指标之间的关联，用于按基站平台过滤可统计的指标列表。

| 表名 | 用途 |
|-----|-----|
| `rela_platform_indicator_formula` | ENB/GSM 平台-指标关联 |
| `gnb_rela_platform_indicator_formula` | GNB 平台-指标关联 |

字段：`platform_name`（平台标识）、`indicator_id`（指标 ID）。

已知平台标识值：ENB/GSM：`Intel` / `QA_V3` / `QA_V4` / `436Q` / `Intel_CR` / `NB_IOT` / `NEU430` / `YD` / `ALL`；GNB：`Intel` / `QA_V3` / `QA_V4` / `436Q` / `Intel_CR` / `NB_IOT` / `NEU430` / `YD` / `BBU_XSS`。
