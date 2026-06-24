# KPI Mgmt（指标管理）功能说明

> 本文档面向 OMC 性能管理（Performance Management）下的"指标管理（KPI Mgmt）"子功能。
>
> 入口路径：`/cell/perfmgmt/kpimanage/goKpiManagementPage`
> 主页面文件：[OMCWebServer/src/main/webapp/WEB-INF/content/pm/perfmgmt/kpi_management.jsp](../src/main/webapp/WEB-INF/content/pm/perfmgmt/kpi_management.jsp)
>
> 涉及网元：**ENB（4G）**、**GSM（2G）**、**GNB（5G）**、**eGW（WCG）**

---

## 一、模块定位

KPI Mgmt 是 OMC 性能管理的"元数据维护"中心，负责对全网性能测量指标进行**全生命周期管理**。它不直接产生 PM 数据，而是定义 PM 数据采集与计算的"规则"：

| 维度 | 说明 |
|------|------|
| 指标功能集（Indicator Group） | 指标的逻辑分组，类似"文件夹" |
| 指标（Indicator） | KPI 或 Counter 的元数据定义 |
| 启用状态（Enable） | 是否参与 PM 文件解析与入库 |
| 计算公式（Arithmetic） | KPI 由多个 Counter 通过表达式计算得出 |
| 平台公式映射（Formula） | 各平台下展开后的原始 Counter 公式 |

页面操作产生的所有变更，都会通过 **Redis 缓存 + 消息发布** 同步到各服务节点（OMCParseKpiFile、cellHandler 等），影响后续的 PM 解析与 KPI 汇总。

---

## 二、页面元素与交互

主页面采用 Vue + EasyUI 混合，分为左侧"指标功能集树"与右侧"指标列表"两部分。下方与右侧滑出 4 个子面板用于增删改查。

### 2.1 整体布局

```
┌──────────────────────────────────────────────────────────────────┐
│  [+] 新建     [↗] 导出                                            │
├──────────────┬───────────────────────────────────────────────────┤
│              │  [搜索框] [产品类型筛选] [等级筛选]                │
│ 指标功能集树 │ ┌─────────────────────────────────────────────┐   │
│  (左侧)      │ │ 已选: (n)   [测量]  [取消测量]              │   │
│              │ ├─────────────────────────────────────────────┤   │
│  - Group1    │ │ □ 指标ID │ 指标名称 │ 单位 │ 类型 │ 等级 │ │   │
│  - Group2    │ │ □ ...                                       │   │
│  - ...       │ │                                              │   │
│              │ │ (分页)                                       │   │
└──────────────┴───────────────────────────────────────────────────┘

  滑出面板（按需弹出）：
   • kpi_addGroup.jsp        — 新建指标功能集
   • kpi_modifyGroup.jsp     — 修改指标功能集
   • kpi_addKpi.jsp          — 新建指标（KPI / Counter）
   • kpi_modifyKpi.jsp       — 修改指标
   • kpi_viewKpi.jsp         — 查看指标详情
   • kpi_addArithmetic.jsp   — 修改内置 KPI（仅基础信息及门限）
```

### 2.2 左侧：指标功能集树

| 元素 | 说明 |
|------|------|
| **搜索框** `#kpimgt_serach_text` | 输入指标名称/ID，回车后重新加载树；树会自动展开第一个有匹配子节点的根节点 |
| **+ 添加按钮**（仅 gNB/eGW 显示） | 点击调用 `goAddKpiGroup('')` 打开新建功能集面板 |
| **树根节点** | ENB 显示 `4G ENB` 与 `4G GSM` 两棵子树；gNB/eGW 只有一棵 |
| **叶子节点** | 实际功能集，点击触发 `onSelect` 加载右侧指标列表 |
| **右键菜单** | 自定义功能集可"修改 / 删除"，内置功能集不可操作 |

后台接口：
- ENB/GSM：`POST /pm/indicatormg/getIndicatorGroupTree`
- GNB：`POST /gnb/pm/indicatormg/getIndicatorGroupTree`
- eGW：`POST /egw/pm/indicatormg/getIndicatorGroupTree`

### 2.3 右侧：指标列表表格

| 列 | 含义 |
|----|------|
| `kpiId` | 指标 ID |
| `kpiName` | 指标名称（运营商可改自定义名） |
| `catagoryName` | 所属功能集 |
| `unit` | 单位（来自 `indicator_unit` 表） |
| `isEnable` | 是否已启用（1=已测量，0=未测量） |
| `indicatorLevel` | 等级：Device / PLMN / Device/PLMN |
| `product_type` | 适用产品类型 |
| `updateTime` | 最后更新时间（按用户时区显示） |
| `custName` | 自定义显示名称 |
| `calculating_status` | 重新计算进度（仅自定义指标，如 `45%` / `100%`） |

**顶部工具栏（仅 ENB 显示）：**
- `已选 (n)` — 弹出已勾选指标列表
- `[测量]` — 批量启用：调用 `enableIndicator`
- `[取消测量]` — 批量禁用，先调用 `isIndicatorInTemplate` 检查是否被模板引用；若有引用则弹确认对话框
- `[产品类型]` 单选下拉过滤
- `[等级]` 单选下拉过滤（Device / PLMN / 全部）

**右上角全局按钮：**
- `[+] 新建` — 弹出指标新建面板（KPI / Counter 二选一）
- `[↗] 导出` — 调用 `exportAllIndicator` 下载 CSV（GBK 编码）

### 2.4 滑出面板：新建/修改指标 (kpi_addKpi.jsp / kpi_modifyKpi.jsp)

页面分为三个区块：

#### (1) 基本信息

| 字段 | 控件 | 必填 | 说明 |
|------|------|------|------|
| Type | 单选 | 是 | `kpi` / `counter`（仅 ENB 4G 支持新建 Counter） |
| 等级 (Level) | 单选 | ENB 必填 | `device` / `plmn` |
| 指标名称 | 输入框 | 是 | 最长 50 字符 |
| 自定义名称 (Custom Name) | 输入框 | 否 | 仅修改基础指标可设置 |
| 所属功能集 | combotree / combobox | 是 | ENB 显示 4G/2G 二级树 |
| 指标 ID | 只读 | - | 系统自动生成（创建时） |
| 单位 | 下拉 | 是 | 来自 `indicator_unit` 表 |
| 统计类型 | 下拉 | 是 | sum / avg / max / min / pct |
| 测量 (Enable) | 下拉 | 是（ENB） | 1=启用，0=禁用 |
| 说明 | 文本域 | 否 | 最长 2000 字符 |

#### (2) 计算公式

- **公式显示区** `#calcExpValueShow`：以"标签 + 运算符"的可视化方式展示
- **运算符按钮**：`+` `-` `*` `/` `(` `)` `0-9` `Duration` `清除`
- **指标选择树 + 表格** `#kpiSetTree` + `#kpiListDatagrid`：从左侧功能集树中选择已有指标加入公式
- **产品类型过滤** `#productTypeSelect`：仅 ENB 显示
- **`Duration` 占位符**：代表统计周期秒数，PM 解析时被替换为 `period × 60`

#### (3) 公式合法性校验（保存时）

1. 公式不为空
2. 同名功能集下指标公式可解析（用 `Evaluator` 把每个 ID 替换为 `1` 后求值，失败则返回 `Expression is invalid`）
3. Counter 名称唯一（仅新建 Counter）

### 2.5 查看指标详情 (kpi_viewKpi.jsp)

只读视图。除展示基本信息和公式外，公式区会针对每个子 Counter 显示其在各平台下展开后的实际公式（`indicatorProductRela` 字段），便于排查计算差异。

### 2.6 功能集面板

| 文件 | 用途 | 触发接口 |
|------|------|----------|
| [kpi_addGroup.jsp](../src/main/webapp/WEB-INF/content/pm/perfmgmt/kpi_addGroup.jsp) | 新建功能集 | `POST /pm/indicatormg/addIndicatorGroup` |
| [kpi_modifyGroup.jsp](../src/main/webapp/WEB-INF/content/pm/perfmgmt/kpi_modifyGroup.jsp) | 修改功能集 | `POST /pm/indicatormg/modifyIndicatorGroup` |

字段：`catagoryName`（最长 50）、`description`（最长 500）。

新建时前端通过 `sessionStorage.addGroupEnbOrGnb` 区分 `0=ENB/GSM`、`1=GNB`、`2=eGW`，但 ENB 模式下还需指定 `device_type=ENB|GSM`（在主页面 Vue 组件 `enbOrGsmTreeAddType` 中维护）。

---

## 三、数据模型

### 3.1 核心数据库表

| 表名 | 作用 |
|------|------|
| `indicator_group` | ENB 4G 指标功能集 |
| `indicator_group_gsm` | GSM 2G 指标功能集（与 ENB 表结构完全相同） |
| `gnb_indicator_group` | GNB 5G 指标功能集 |
| `wcg_indicator_group` | eGW 指标功能集 |
| `perf_indicators` | ENB/GSM 指标主表 |
| `gnb_perf_indicators` | GNB 指标主表 |
| `wcg_perf_indicators` | eGW 指标主表 |
| `rela_platform_indicator_formula` | ENB/GSM 平台-指标公式关系表 |
| `gnb_rela_platform_indicator_formula` | GNB 平台-指标公式关系表 |
| `enabled_pm_indicators` | 已启用指标表（`operator_code` + `indicator_id` 联合主键） |
| `perf_cust_name` | 运营商对内置指标的自定义显示名 |
| `indicator_unit` | 指标单位字典 |
| `indicator_threshold` | 指标门限配置（minor/major 两档） |
| `perf_template_rel_arithmetic` | 模板-指标关联表（删除指标前需检查） |
| `perf_alarm_threshold` | KPI 告警阈值（删除指标前需检查） |

### 3.2 关键字段说明（`perf_indicators`）

| 字段 | 类型 | 含义 |
|------|------|------|
| `id` | varchar(20) | 指标 ID（主键） |
| `en_name` / `cn_name` | varchar(200) | 中英文名称 |
| `group_id` | varchar(32) | 所属功能集 ID |
| `operator_code` | varchar(100) | 运营商编码，`NULL` 表示系统内置 |
| `unit_id` | varchar(15) | 单位 ID |
| `is_build_in` | char(1) | 1=内置，0=自定义 |
| `is_counter` | char(1) | 1=Counter，0=KPI |
| `arithmetic` | text | 计算公式（Counter 时存放自身名称） |
| `statis_type` | varchar(10) | 统计类型：sum/avg/max/min/pct |
| `indicator_level` | varchar(20) | device/plmn/both（默认 device） |
| `product_types` | varchar(200) | 适用产品类型（多个逗号分隔） |
| `calculating_status` | varchar(10) | 重新计算进度（如 `45%`） |
| `uptime` | datetime | 最后更新时间（UTC） |

### 3.3 指标 ID 命名规范

| 类型 | 格式 | 示例 |
|------|------|------|
| 自定义 KPI | `{cloudKey}K90000{NNNN}` | `defaultK900000001` |
| 自定义 Counter | `D00000{NNNN}` | `D000000001` |
| 内置 KPI | `K{XXXXXXXX}` | `K900010013` |
| 内置 Counter | `C{XXXXXXXX}` | `C000060011` |

`cloudKey` 来自 `operators_info.cloud_key`；序号 4 位，运营商范围内上限 `9999`。

---

## 四、业务逻辑：创建与维护

### 4.1 系统整体调用链

```
浏览器 (kpi_management.jsp + Vue)
   │
   ├─ ENB/GSM ─► OMCWebServer/IndicatorManageAction
   │                │
   │                └─► RPC → OMCParseKpiFile/IndicatorManagerRpcService
   │                              │
   │                              └─► IndicatorMgmtService (写 MySQL + 写 Redis)
   │
   ├─ GNB    ─► OMCWebServer/GNBIndicatorManageAction
   │                │
   │                └─► RPC → GNB 专用 RPC
   │
   └─ eGW    ─► OMCWebServer/WCGIndicatorManageAction
                    │
                    └─► 本地 Service (直接写库)
```

### 4.2 指标功能集 CRUD

#### 新建（`addIndicatorGroup`）

1. 前端表单校验：名称非空、长度 ≤ 50
2. Action 校验 `device_type` 是必填且只能是 `ENB` / `GSM`（仅 ENB 主网元下）
3. RPC → `IndicatorMgmtService.addIndicatorGroup`：
   - 查询运营商下同名记录（**跨 ENB/GSM 表分别检查**）
   - 名称已存在 → 返回 `name exists`
   - 生成 UUID 作为 `id`，`parent_id` 设为对应表的根节点 ID
   - 插入 `indicator_group` 或 `indicator_group_gsm` 表
4. 记录操作日志（`OMCLogConstants.ADDKPI = 10019`）

#### 修改（`modifyIndicatorGroup`）

根据 `group_id` **自动检测**属于哪张表（ENB 或 GSM），不再传 `device_type`。修改名称时仍需检查同名冲突。

#### 删除（`delIndicatorGroup`）

**前置检查**：
- 该功能集下的指标，若有任意一个被 KPI 报表模板引用（`perf_template_rel_arithmetic`），则阻止删除并返回 `template is using`

**事务删除步骤**：
1. 删除 `rela_platform_indicator_formula` 中该功能集所有指标的公式关系
2. 删除 `perf_indicators` 中该功能集所有指标
3. 删除 `indicator_group` / `indicator_group_gsm` 记录
4. 清理 Redis 缓存

### 4.3 指标 CRUD

#### 新建/修改指标（`addOrModifyIndicator`）

**前端接口路径**：`POST /pm/indicatormg/addOrModifyIndicator`
**Action 文件**：[IndicatorManageAction.java#L1012](../src/main/java/com/baicells/omc/busi/cell/perfmgmt/action/IndicatorManageAction.java#L1012)

**完整流程**：

```
Step 1：参数提取
   kpiId / kpiName / catagoryId / unit / arithmetic /
   statisType / product_type / isEnable / indicatorLevel / custName
   （kpiId 为空 → 新建；否则修改；以 'D' 开头则强制 indicatorType=counter）

Step 2：参数校验
   ✗ catagoryId 为空 → "Category ID is required"

Step 3：公式合法性校验（仅 KPI 类型）
   • 取同功能集下所有指标的 ID→公式映射 (getIndicatorIdRelArithMap)
   • 用 Jeval Evaluator 把公式中每个 ID 替换为 1 再求值
   • 失败 → "Expression is invalid"

Step 4：Counter 名称唯一性（仅新建 Counter）
   • 调用 getEnbCounterNameExists → 已存在则返回 "counter name already exists"

Step 5：判断是否需要重新计算（whetherNeedToReStatistics）
   • 新建指标 → true
   • 修改指标 → 仅当 arithmetic 或 statisType 变化时才 true

Step 6：调用 RPC addOrModifyIndicatorInfo (in OMCParseKpiFile)
   • 新建：
       - 校验自定义指标数 < 9999；超出 → "customize kpi number is over 9999"
       - 查 operators_info.cloud_key
       - 生成新 ID：{cloudKey}K90000{NNNN} 或 D00000{NNNN}
   • 修改：
       - 先 deleteIndicatorCache 删除旧 Redis 缓存
       - 检查 is_build_in=0 → 标记为自定义指标

   • UPSERT perf_indicators （插入或更新基础信息）
   • addOrupdatePlatformArith：
       - 自定义指标修改时，先 DELETE 该指标旧的平台关系
       - 拆解公式中的 counter ID，按平台支持性分别 INSERT/UPDATE
         rela_platform_indicator_formula
   • Redis 缓存写入：
       - hset "indicator_unit_map"        ID→unit
       - hset "indicator_statis_type_map" ID→statisType
       - hset "enb_custom_counter_map"    name→ID  （仅自定义 Counter）

Step 7：处理启用状态（仅 ENB，由 isEnable 参数决定）
   • isEnable=1 → kpiClient.enableIndicator(operatorCode, newKpiId)
   • isEnable=0 → kpiClient.disableIndicator(operatorCode, newKpiId)

Step 8：触发重新计算（仅 needToReStatistics=true 且 isEnable=1）
   • updateKpiCalculatingStatus(newKpiId, "0%")
   • kpiClient.reStatisticsKPI(operatorCode, newKpiId)
     → OMCParseKpiFile.StatisticsRpcService.reStatisticsKPI
     → 加入 ReStatisticsQueueService 队列
     → ReStatisticsKPIThread 异步消费

Step 9：写操作日志（OMCLogConstants.ADDKPI = 10019）
```

#### 删除指标（`delIndicator`）

**前置检查**（任一满足则阻止）：

| 返回值 | 含义 |
|--------|------|
| `template is using` | 被 KPI 报表模板引用 |
| `alarm template is using` | 被 KPI 告警模板引用 |
| `kpi formuals is using` | 被其他 KPI 的计算公式引用 |

**通过后执行**：
1. `DELETE FROM perf_indicators WHERE id=?`
2. `DELETE FROM rela_platform_indicator_formula WHERE indicator_id=?`
3. `PmIndicatorEnableService.disableIndicator(operatorCode, [id])`（移除启用记录 + 发布禁用消息）
4. 清理 Redis 缓存（`indicator_unit_map`、`indicator_statis_type_map`、`enb_custom_counter_map` 等）

### 4.4 指标启用 / 禁用

| 接口 | 路径 |
|------|------|
| 启用 | `POST /cell/perfmgmt/kpimanage/enableIndicator` |
| 禁用 | `POST /cell/perfmgmt/kpimanage/disableIndicator` |
| 模板关联检查 | `GET /cell/perfmgmt/kpimanage/isIndicatorInTemplate` |

**后端流程**（以启用为例）：

```
OMCWebServer.KPIManageAction.enableIndicator
  └─ KPIClient (Feign) → OMCParseKpiFile /rpc/enableIndicator
        └─ PmIndicatorEnableService.enableIndicator
             ├─ INSERT enabled_pm_indicators (operator_code, indicator_id) — UPSERT
             └─ KpiEnableUpdateMsgPublisher.publish
                  └─ Redis 发布 "TOPIC_KPI_ENABLE_UPDATE" 频道消息
                       └─ 各节点 KpiEnableUpdateMsgListener.onMessage
                            └─ enableIndicatorCache / disableIndicatorCache 刷新内存缓存
```

**缓存结构**：`ConcurrentHashMap<operatorCode, ConcurrentHashMap<indicatorId, Object>>`
**淘汰策略**：LRU，最多缓存 `prop.getEnabledIndicatorOperatorCacheCount()` 个运营商。

### 4.5 指标自定义名称

接口：`POST /pm/indicatormg/updateBaseKpiCustName`
参数：`kpiId` / `custName` / `isEnable`
逻辑：`perf_cust_name` 表 UPSERT（`operator_code`+`perf_id` 复合主键），同步处理启用状态。

### 4.6 修改 Counter 名称

接口：`POST /pm/indicatormg/updateEnbIndicatorsName`
逻辑：校验唯一性后更新 `perf_indicators.en_name` 与 `cn_name`。

### 4.7 导出指标列表

接口：`POST /pm/indicatormg/exportAllIndicator`
输出：CSV 文件，GBK 编码（兼容 Excel 打开中文）
导出列：`指标ID`、`指标名称`、`产品类型`、`自定义名称`、`等级`、`单位`、`类型`、`更新人`、`更新时间`、`计算公式`

---

## 五、KPI 记录生效后：后台如何执行 KPI 统计

KPI Mgmt 页面的所有变更最终都会影响 OMCParseKpiFile 子系统的三个流程：

### 5.1 PM 文件解析（采集时生效）

**触发**：基站通过 TR-069 上报 PM 文件 → cellHandler → 推送到 Redis 队列 → `ParseOneCellCounterThread`/`ParseOneCellKPIThread` 消费

**关键过滤**：[ParsePerfFileService.filterOutDisabledIndicator](../../OMCParseKpiFile/src/main/java/com/baicells/omc/OMCParseKpiFileToDB/com/service/ParsePerfFileService.java)

```java
for (entry : indicatorMap) {
    if (!indicatorEnableService.isEnable(operatorCode, indicatorId)) {
        it.remove();   // 未启用 → 跳过，不入库
    }
}
```

即：**只有在 `enabled_pm_indicators` 表中存在记录的指标，PM 文件解析时才会真正写入 MongoDB 15min 表**。

**判断顺序**：
1. 先查内存缓存 `enabledIndicatorsCache`
2. 缓存未命中 → 从 `enabled_pm_indicators` 表加载并写入缓存
3. 缓存满 → LRU 淘汰最久未访问运营商

**入库的目标表**：

| 类型 | 表名 |
|------|------|
| ENB Device | `pm_15min_yyyy_MM_dd`（按天分表） |
| ENB PLMN | `pm_15min_plmn_yyyy_MM_dd`（按天分表） |
| GSM (BSC) | `pm_15min_yyyy_MM_dd` |
| GNB | `gnb_pm_value_15`（固定表） |

> **公式型 KPI（statis_type=pct）的特殊性**：解析时只入库其引用的 Counter，KPI 本身不直接入库；查询/汇总时再按公式实时计算。

### 5.2 定时汇总（小时 / 天 / 周 / 月）

OMCParseKpiFile 启动时通过 Quartz 注册 4 个 Job（ENB/GSM 与 GNB 各一套，共 8 个）：

| Job | 类 | Cron | 开关 |
|------|------|------|------|
| 小时（60） | `QuartzStatisticsKPIDataJob_60` | `0 {KPIStatis60Delay} * * * ?` | 无 |
| 天（1440） | `QuartzStatisticsKPIDataJob_1440` | `0 {KPIStatis1440Delay} * * * ?` | 无 |
| 周（10080） | `QuartzStatisticsKPIDataJob_10080` | `0 30 * ? * SUN,MON` | `kpiWeekAndMonthSwitch=1` |
| 月（43200） | `QuartzStatisticsKPIDataJob_43200` | `0 30 * 28,29,30,31,1 * ?` | `kpiWeekAndMonthSwitch=1` |

GNB 对应 `GnbQuartzStatisticsKPIDataJob_*`，eGW 对应 `WCGQuartzStatisticsKPIDataJob_*`。

**小时粒度汇总执行步骤**（以 `QuartzStatisticsKPIDataJob_60` 为例）：

```
1. 取服务器 UTC 当前时间，秒/分/毫秒置零，作为 statisTime
2. TimingStatistics.statistics(statisTime, 60)
   │
   ├─ prepareIndicatorInfo: 从缓存装配 statisType / unit / arithmetic
   │
   ├─ prepareCellInfoAndTimeRange:
   │   • 计算 minStartTime / maxStartTime（运营商时区敏感）
   │   • 过滤出"已到统计时间点"的运营商（仅小时粒度不限时区）
   │   • 收集这些运营商下的所有在线基站
   │
   └─ 逐基站 processCellStatistics：
       • 非 BSC 设备：分 DEVICE / PLMN 两次汇总
       • BSC 设备：按 bts_id 分批查询
       • 按 (cellId, plmnId) 或 (nrCGI, plmnId) 分组
       • MongoDB Aggregate：sum/avg/max/min Counter 值
       • pct 型 KPI：将子 Counter 聚合值代入公式 Evaluator 求值
       • UPSERT 到 pm_hour_yyyy_MM_dd / pm_hour_plmn_yyyy_MM_dd

3. KpiGroupDataStatisService.statisGroupData(statisTime, 60)
   • 以基站维度数据为源，按设备组聚合（二级 + 一级两层）
   • UPSERT 到 pm_group_hour_yyyy_MM_dd

4. updateDataEndTime(ENB, statisTime, "60")
   • 写 perf_statis_data_end_time，供前端"最新数据时间"展示

5. DashboardStatisticsService.statistics(ENB/GSM, statisTime, 60)
   • 按 dashboard.statistics.{enb|gsm|gnb}.kpi.list 配置项中的固定指标列表统计
   • 写 statistics_enb_kpi_hour / statistics_gsm_kpi_hour

6. DashboardStatisticsService.statisticsCustom(...)
   • 自定义看板统计，出错不中断
   • 写 statistics_dashboard_custom_kpi_hour
```

**天粒度（1440）额外执行**：

- `statisticsTop(ENB/GSM, statisTime, STATISTICS_KPI_DAY_TOP_KEY)` 写 `statistics_kpi_day_top`
- `statisticsTop(ENB/GSM, statisTime, STATISTICS_KPI_DAY_BOTTOM_KEY)` 写 `statistics_kpi_day_bottom`

**月粒度（43200）额外条件**：当天必须是 UTC 月的第 1 天或最后一天（覆盖跨时区运营商）。

### 5.3 公式型（pct）KPI 的汇总细节

公式存储格式（`arithmetic` 字段）：

```
C000060011/C000060012*100
(C000060001+C000060021)*8/Duration/1000
```

**汇总步骤**：

```
1. 拆分公式：StringTokenizer st = new StringTokenizer(arithmetic, "+-*/()", true)

2. 对公式中引用的每个 counter，按其自身 statis_type 聚合
   （sum/avg/max/min 由 MongoDB Aggregate $sum/$avg/... 完成）

3. 将公式 token 逐一替换：
   • "Duration"     → period × 60（秒数）
   • 数字 / 运算符 → 保持不变
   • Counter ID    → 已聚合的 Decimal128 值
                     - 任一子 Counter 为 "-" 或缺失 → 整条公式作废，结果记为 "-"

4. 用 net.sourceforge.jeval.Evaluator 求值
   • 结果为 Infinity / -Infinity / NaN → 记为 "N/A"
   • 否则按指标单位调用 ParseKPIUtil.processValue 调整精度

5. 写入目标表
```

### 5.4 新建/修改指标后的"重新计算"

如果在 KPI Mgmt 页面新建或修改了 KPI 且勾选"启用"，且公式或统计类型有变化，会触发历史数据重算：

```
Web 端 IndicatorManageAction
   │
   └─ kpiIndicatorManageClient.updateKpiCalculatingStatus(kpiId, "0%")
        ↓ (同步)
        UPDATE perf_indicators SET calculating_status='0%' WHERE id=?
   │
   └─ kpiClient.reStatisticsKPI(operatorCode, kpiId)
        ↓ (异步)
        OMCParseKpiFile.StatisticsRpcService
        ↓
        ReStatisticsQueueService.add(new ReStatisticsKPIRequest(...))
        ↓
        ReStatisticsKPIThread.run() 后台线程消费
           │
           ├─ 若运营商无基站 → 直接 calculating_status='100%'
           │
           └─ 否则三段式重算（15min → 1h → 1day）：
               • reStatistics(15min,  base= 0%, weight=0.5)   →  0%-50%
               • reStatistics(60min,  base=50%, weight=0.3)   → 50%-80%
               • reStatistics(1440,   base=80%, weight=0.2)   → 80%-100%

              每个粒度内：
               • 按 totalEndTime 向前逆推，每个周期一次
               • 单周期：StatisticsService.reStatisticsForSpecificIndicator(...)
                 → 内部按 pct / 非 pct 走不同分支
               • 每完成一个周期更新 calculating_status（仅在百分比变化时写库）
```

> **保留数据时间窗**：取 15min/小时/天 三类数据保留时长的**最小值**，超出此范围的旧数据不再重算。

### 5.5 启用状态的多节点同步

```
任一节点改启用状态 (enableIndicator / disableIndicator)
   │
   ├─ 写 enabled_pm_indicators 表
   │
   └─ KpiEnableUpdateMsgPublisher.publish (Redis Pub/Sub)
        │
        └─ TOPIC_KPI_ENABLE_UPDATE 频道
             │
             └─ 所有订阅节点 KpiEnableUpdateMsgListener.onMessage
                  • enable=1 → enableIndicatorCache(operator, [ids])
                  • enable=0 → disableIndicatorCache(operator, [ids])
```

确保 PM 解析瞬时拿到最新的启用状态。

---

## 六、ENB / GSM / GNB / eGW 差异速查

| 维度 | ENB (4G) | GSM (2G) | GNB (5G) | eGW (WCG) |
|------|----------|----------|----------|-----------|
| 功能集表 | `indicator_group` | `indicator_group_gsm` | `gnb_indicator_group` | `wcg_indicator_group` |
| 指标表 | `perf_indicators` | `perf_indicators`<sub>（共表，靠 group_id 区分）</sub> | `gnb_perf_indicators` | `wcg_perf_indicators` |
| URL 前缀 | `/pm/indicatormg` | `/pm/indicatormg`<br/>（参 `device_type=gsm`） | `/gnb/pm/indicatormg` | `/egw/pm/indicatormg` |
| 等级（level） | device/plmn/both | both（兼容方案） | 无 | 无 |
| 启用/禁用 | 有 | 有 | **无**（自动测量） | **无** |
| 产品类型 | 有（多个） | 通常仅 BSC/BTS | 有（BaiBNX/BaiBNQ/CHINA_TELECOM） | 无 |
| Counter 新建 | 支持 | 支持 | 支持（统一入口） | 支持 |
| 告警模板检查 | 删除时检查 | 删除时检查 | **不检查** | 删除时检查 |
| 重算队列 | `ReStatisticsQueueService` | 同 ENB | `ReStatisticsGNBQueueService` | `ReStatisticsWCGQueueService` |
| 定时 Job 后缀 | `QuartzStatisticsKPIDataJob_*` | 同 ENB | `GnbQuartzStatisticsKPIDataJob_*` | `WCGQuartzStatisticsKPIDataJob_*` |

---

## 七、关键代码索引

| 文件 | 作用 |
|------|------|
| [KPIManageAction.java](../src/main/java/com/baicells/omc/busi/cell/perfmgmt/action/KPIManageAction.java) | 页面入口 + 启用/禁用 + 模板关联检查 |
| [IndicatorManageAction.java](../src/main/java/com/baicells/omc/busi/cell/perfmgmt/action/IndicatorManageAction.java) | ENB/GSM 指标管理 Action |
| [GNBIndicatorManageAction.java](../src/main/java/com/baicells/omc/busi/cell/gnb/perfmgmt/action/GNBIndicatorManageAction.java) | GNB 指标管理 Action |
| [WCGIndicatorManageAction.java](../src/main/java/com/baicells/omc/busi/cell/eGW/perfmgmt/action/WCGIndicatorManageAction.java) | eGW 指标管理 Action |
| [IndicatorMgmtService.java](../../OMCParseKpiFile/src/main/java/com/baicells/omc/OMCParseKpiFileToDB/com/service/indicator/IndicatorMgmtService.java) | 真正的写库逻辑（OMCParseKpiFile 子系统） |
| [PmIndicatorEnableService.java](../../OMCParseKpiFile/src/main/java/com/baicells/omc/OMCParseKpiFileToDB/com/service/PmIndicatorEnableService.java) | 启用/禁用 + 缓存 + Redis 发布 |
| [KpiEnableUpdateMsgListener.java](../../OMCParseKpiFile/src/main/java/com/baicells/omc/OMCParseKpiFileToDB/rpc/subscribe/listener/KpiEnableUpdateMsgListener.java) | 启用变更消息订阅 |
| [TimingStatistics.java](../../OMCParseKpiFile/src/main/java/com/baicells/omc/OMCParseKpiFileToDB/com/service/statistics/TimingStatistics.java) | ENB 定时汇总核心 |
| [QuartzStatisticsKPIDataJob_60.java](../../OMCParseKpiFile/src/main/java/com/baicells/omc/OMCParseKpiFileToDB/com/job/QuartzStatisticsKPIDataJob_60.java) | 小时粒度 Quartz Job |
| [ReStatisticsKPIThread.java](../../OMCParseKpiFile/src/main/java/com/baicells/omc/OMCParseKpiFileToDB/com/thread/ReStatisticsKPIThread.java) | 重新计算后台线程 |
| [StatisticsService.java](../../OMCParseKpiFile/src/main/java/com/baicells/omc/OMCParseKpiFileToDB/com/service/statistics/StatisticsService.java) | 单指标重算 + pct 公式计算 |
| [ApplicationStartup.java](../../OMCParseKpiFile/src/main/java/com/baicells/omc/OMCParseKpiFileToDB/ApplicationStartup.java) | Quartz Job 注册（搜索 `goStatisticKPIData_`） |

相关文档：
- [kpi_indicator_management.md](kpi_indicator_management.md) — 接口与字段级别详细规范
- [kpi_template_page_analysis.md](kpi_template_page_analysis.md) — KPI 模板页面分析（与本模块上下游）
- [KPI数据链路与Mongo依赖-记忆沉淀.md](KPI数据链路与Mongo依赖-记忆沉淀.md) — 数据链路与 Mongo 依赖说明
- [../../OMCParseKpiFile/docs/KPI定时汇总流程说明.md](../../OMCParseKpiFile/docs/KPI定时汇总流程说明.md) — 汇总流程完整规范
- [../../OMCParseKpiFile/docs/pm文件解析流程.md](../../OMCParseKpiFile/docs/pm文件解析流程.md) — PM 文件解析流程

---

## 八、流程图：一次"新建自定义 KPI 并启用"完整链路

```
[ kpi_management.jsp ]
       │ 用户点 [+] 新建
       ▼
[ kpi_addKpi.jsp ]
   填表 → 选公式 → 勾选启用 → 确定
       │ POST /pm/indicatormg/addOrModifyIndicator
       ▼
[ OMCWebServer.IndicatorManageAction#addOrModifyIndicator ]
   1. 公式合法性校验
   2. needToReStatistics = true  （新建）
   3. RPC: kpiIndicatorManageClient.addOrModifyIndicatorInfo
       ▼
[ OMCParseKpiFile.IndicatorMgmtService#addOrModifyIndicatorInfo ]
   • 校验自定义指标数 < 9999
   • 生成 ID {cloudKey}K90000NNNN
   • UPSERT perf_indicators
   • addOrupdatePlatformArith → 更新 rela_platform_indicator_formula
   • Redis hset indicator_unit_map / indicator_statis_type_map
       │ 返回 SaveMessage{success=true, message=newKpiId}
       ▼
   4. kpiClient.enableIndicator(operatorCode, newKpiId)
       ▼
[ OMCParseKpiFile.PmIndicatorEnableService#enableIndicator ]
   • INSERT enabled_pm_indicators
   • Redis publish "TOPIC_KPI_ENABLE_UPDATE"
       ▼
[ 所有节点的 KpiEnableUpdateMsgListener ]
   • enableIndicatorCache(operator, [newKpiId])
       │
       ▼
   5. updateKpiCalculatingStatus(newKpiId, "0%")
       ▼
   6. kpiClient.reStatisticsKPI(operatorCode, newKpiId)
       ▼
[ OMCParseKpiFile.StatisticsRpcService#reStatisticsKPI ]
   • queue.add(new ReStatisticsKPIRequest(...))
       ▼ (异步)
[ ReStatisticsKPIThread.run() ]
   • 15min  阶段 ( 0%-50%) → reCalculatePctKPI/reCalculateNonPctKPI
   • 60min  阶段 (50%-80%) → ...
   • 1440   阶段 (80%-100%) → ...
   • 每段完成更新 calculating_status
       │
       ▼
[ 用户在页面看到 calculating_status 实时变化，到达 100% 即完成 ]

—— 此后正常运转 ——

[ 基站持续上报 PM 文件 ]
       ▼
[ ParsePerfFileService.filterOutDisabledIndicator ]
   • 查 isEnable(operator, newKpiId) = true → 保留
       ▼
[ 入库 pm_15min_yyyy_MM_dd ]
       ▼ (每小时 Quartz 触发)
[ QuartzStatisticsKPIDataJob_60 → TimingStatistics → KpiGroupDataStatisService ]
   • 聚合 → pm_hour_yyyy_MM_dd / pm_group_hour_yyyy_MM_dd
       ▼
[ 前端 KPI 查询页面 / 报表 / 看板 ] 可看到新指标数据
```

---

*文档版本：2026-06-22*
