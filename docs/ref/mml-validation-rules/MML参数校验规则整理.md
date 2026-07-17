# MML 参数校验规则整理

## 一、目标

整理 MML 参数的存储位置、字段含义、接口返回字段、前端校验逻辑和后台校验逻辑，便于后续在测试环境读取数据库后，生成每个参数的完整校验规则。

## 二、相关接口

前端请求：

```text
POST /cell/param/getParamGroupTreeNodes.action
```

主要请求参数：

| 参数 | 含义 |
|---|---|
| `operID` | 参数组 ID，后台转换为 `paramGroupId` |
| `actionType` | 操作类型，例如 `v_lst`、`v_add`、`v_mod`、`v_rmv` |
| `hardwareVersion` | 设备硬件或软件版本 |
| `isGnb` | GNB 页面可能携带，当前接口本身没有直接使用 |

主要代码位置：

- Web Controller：`OMCWebServer/src/main/java/com/baicells/omc/busi/cell/paramconfig/action/CellParamConfigAction.java`
- RPC 客户端：`OMCWebServer/src/main/java/com/baicells/omc/rpc/client/MmlHandlerClient.java`
- RPC 服务：`cellHandler/src/main/java/com/baicells/cellHandler/rpc/service/CellMMLParamServiceRpc.java`
- 业务实现：`cellHandler/src/main/java/com/baicells/cellHandler/beans/task/service/impl/CellMMLParamServiceImpl.java`

调用链：

```text
浏览器
  -> OMCWebServer
  -> MmlHandlerClient
  -> cellHandler
  -> MySQL
  -> 返回参数列表
```

## 三、数据库表

### 1. 参数定义表

```text
small_cell.small_cell_param
```

一行通常表示一个参数，保存参数名称、数据类型、取值范围、默认值、正则表达式等规则。

### 2. 参数组关联表

```text
small_cell.rela_groupid_paramid
```

表示参数组包含哪些参数，以及参数在页面上的显示顺序。

关联关系：

```text
rela_groupid_paramid.param_group_id = small_cell_param_group.id
rela_groupid_paramid.param_id       = small_cell_param.param_id
```

### 3. 参数组表

```text
small_cell.small_cell_param_group
```

保存参数组 ID、操作关键字、参数版本、参数组名称、`add_path` 等信息。

## 四、参数表主要字段

| 字段 | 含义 |
|---|---|
| `param_id` | 参数唯一 ID，用于关联参数组和参数定义 |
| `param_name` / `PARAM_NAME` | 参数中文名称 |
| `param_name_en` / `PARAM_NAME_EN` | 参数英文名称 |
| `mib_dn` | 参数实际命令字段名，前端输入框的 `name` 通常使用此值 |
| `name_path` | 参数完整路径，用于多实例、多小区和层级参数 |
| `v_type` | 数据类型、长度、数值范围或枚举规则，是后台校验的核心字段 |
| `js_regex` | 前端 JavaScript 正则表达式校验规则 |
| `dft_value` | 参数默认值 |
| `v_dynamic` | 是否支持动态生效；通常 `1` 表示动态，`0` 表示可能需要重启 |
| `v_lst` | 是否支持查询操作 |
| `v_add` | 是否支持新增操作 |
| `v_mod` | 是否支持修改操作 |
| `v_rmv` | 是否支持删除操作 |
| `title_cn` | 中文提示标题 |
| `title_en` | 英文提示标题 |
| `cn_explanation` / `CN_EXPLANATION` | 中文参数说明 |
| `en_explanation` / `EN_EXPLANATION` | 英文参数说明 |
| `software_version` / `param_version` | 参数适用的设备或参数版本，实际字段以数据库结构为准 |

## 五、操作字段和必填规则

操作字段常见值：

| 值 | 含义 |
|---|---|
| `N` | 不支持该操作 |
| `Y` | 支持该操作，但通常不是必填 |
| `Y*` | 支持该操作，并且是必填参数 |

`must` 不是数据库原始字段，而是后台根据操作字段推导出来的：

```text
v_add = Y* -> add_required = 1
v_mod = Y* -> mod_required = 1
v_rmv = Y* -> rmv_required = 1
v_lst = Y* -> lst_required = 1
```

整理时不要只记录一个 `must`，建议分别记录：

```text
add_required
mod_required
rmv_required
lst_required
```

## 六、`v_type` 规则格式

### 1. 字符串最大长度

```text
string-32
```

表示字符串最大长度为 32。

### 2. 字符串长度范围

```text
string-[1:32]
```

表示最小长度为 1，最大长度为 32。

### 3. 整数只有最小值

```text
int-[1:]
unsignedInt-[1:]
```

表示最小值为 1，没有明确最大值。

### 4. 整数有最小值和最大值

```text
int-[0:65535]
unsignedInt-[0:65535]
```

表示取值范围为 0 到 65535。

### 5. 整数只有最大值

```text
int-[:65535]
unsignedInt-[:65535]
```

表示最大值为 65535。`unsignedInt` 通常还应考虑最小值不能小于 0。

### 6. 普通枚举

```text
enum-{0,1,2}
```

只允许输入 `0`、`1`、`2`。

### 7. 带显示值和实际值的枚举

```text
enum-{TRUE,FALSE}-{1,0}
```

通常表示页面显示 `TRUE/FALSE`，实际提交值为 `1/0`。

### 8. 布尔值

```text
bool-{0,1}
bool-{TRUE,FALSE}-{1,0}
```

表示布尔类型参数。

### 9. List 和 struct

包含 `List` 的类型通常表示列表参数，包含 `struct` 的类型通常表示结构化参数。这类参数不能简单按照普通整数或普通字符串处理，需要结合前端代码和设备协议进一步确认。

后台解析 `v_type` 的核心代码：

```text
OMCWebServer/src/main/java/com/baicells/omc/busi/cell/cpeinfos/param/validate/SetParameterValidate.java
```

主要支持：

- 字符串长度校验
- 整数最小值校验
- 整数最大值校验
- 整数最小值和最大值校验
- 枚举值校验
- 布尔值校验
- 部分特殊整数处理

## 七、`js_regex` 规则

`js_regex` 主要用于前端校验。接口返回后，前端将它放到输入框属性中，并在输入框失去焦点时执行。

例如：

```text
/^\\w+$/
```

表示只允许字母、数字和下划线。

```text
/^(?:\\d+|%forever)$/
```

表示允许数字或特殊值 `%forever`。

特殊值：

```text
no_zh
```

表示不允许输入中文。前端会将它转换成禁止中文字符的正则。

注意：

- `js_regex` 主要是前端校验规则；
- `v_type` 是前端和后台都使用的重要类型规则；
- `js_regex` 为空不代表没有校验，仍然可能存在 `v_type` 校验；
- 前端使用 `eval` 将字符串转换成正则，规则内容必须保证格式正确。

## 八、接口返回给前端的字段

对于 `v_add`、`v_mod` 等详细操作，后台通常将数据库字段转换为以下结构：

| 返回字段 | 数据库来源 | 含义 |
|---|---|---|
| `param_id` | `param_id` | 参数 ID |
| `paramName` | `PARAM_NAME` 或 `PARAM_NAME_EN` | 参数名称 |
| `dftValue` | `dft_value` | 默认值；部分 `v_mod` 场景会强制为空 |
| `mib_dn` | `mib_dn` | 实际命令字段名 |
| `dataType` | `v_type` | 类型及范围 |
| `dynamic` | `v_dynamic` | 是否动态生效 |
| `name_path` | `name_path` | 参数完整路径 |
| `js_regex` | `js_regex` | 前端正则校验规则 |
| `title` | `title_cn` 或 `title_en` | 页面提示标题 |
| `must` | 根据对应操作字段是否为 `Y*` 推导 | 是否必填 |
| `explanation` | `CN_EXPLANATION` 或 `EN_EXPLANATION` | 参数说明 |

## 九、前端校验逻辑

前端主要进行以下检查：

### 1. 必填检查

当返回的 `must` 为 `1` 时，绑定必填校验。

### 2. 数值范围和字符串长度检查

解析 `dataType`，也就是数据库的 `v_type`，生成最小值、最大值或长度限制。

例如：

```text
unsignedInt-[0:65535]
```

会生成：

```text
min_value = 0
max_value = 65535
```

### 3. 正则检查

如果 `js_regex` 不为空，则输入框失去焦点时执行正则校验。

主要前端代码：

```text
OMCWebServer/src/main/webapp/WEB-INF/content/gnodeb/maintenance/mml.jsp
```

## 十、后台校验逻辑

后台提交 MML 参数时会重新从数据库查询参数定义，不完全相信前端传来的内容。

主要入口：

```text
OMCWebServer/src/main/java/com/baicells/omc/busi/cell/cpeinfos/param/validate/SetParameterValidate.java
```

后台主要校验：

1. 操作格式是否正确；
2. 参数名是否存在；
3. 参数是否属于当前操作；
4. 字符串长度；
5. 数值最小值；
6. 数值最大值；
7. 枚举值；
8. 布尔值；
9. 部分特殊参数规则。

需要特别注意：

> 当前后台校验类重点解析 `v_type`，不一定重新执行数据库中的 `js_regex`。因此整理时必须把“前端规则”和“后台规则”分开记录。

## 十一、建议在测试环境执行的 SQL

### 1. 先确认参数表结构

```sql
SELECT
    COLUMN_NAME,
    DATA_TYPE,
    CHARACTER_MAXIMUM_LENGTH,
    IS_NULLABLE,
    COLUMN_DEFAULT,
    COLUMN_COMMENT
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = 'small_cell'
  AND TABLE_NAME = 'small_cell_param'
ORDER BY ORDINAL_POSITION;
```

### 2. 查询参数原始规则

实际字段名称以表结构查询结果为准：

```sql
SELECT
    param_id,
    param_name,
    param_name_en,
    mib_dn,
    name_path,
    v_type,
    js_regex,
    dft_value,
    v_dynamic,
    v_add,
    v_mod,
    v_lst,
    v_rmv,
    title_cn,
    title_en,
    cn_explanation,
    en_explanation,
    software_version,
    param_version
FROM small_cell.small_cell_param;
```

如果 `software_version` 或 `param_version` 不存在，应根据第一步的实际字段调整 SQL。

### 3. 查询参数组

```sql
SELECT
    id,
    keyword,
    param_version,
    add_path,
    param_name,
    param_name_en
FROM small_cell.small_cell_param_group;
```

### 4. 查询参数组和参数关系

```sql
SELECT
    param_group_id,
    param_id,
    `order`
FROM small_cell.rela_groupid_paramid
ORDER BY param_group_id, `order`;
```

### 5. 推荐的关联查询

如果实际字段与代码一致，可以执行：

```sql
SELECT
    pg.id AS param_group_id,
    pg.keyword AS operation_keyword,
    pg.param_version,
    rela.`order` AS display_order,
    pa.param_id,
    pa.param_name,
    pa.param_name_en,
    pa.mib_dn,
    pa.name_path,
    pa.v_type,
    pa.js_regex,
    pa.dft_value,
    pa.v_dynamic,
    pa.v_lst,
    pa.v_add,
    pa.v_mod,
    pa.v_rmv,
    CASE WHEN pa.v_lst = 'Y*' THEN 1 ELSE 0 END AS lst_required,
    CASE WHEN pa.v_add = 'Y*' THEN 1 ELSE 0 END AS add_required,
    CASE WHEN pa.v_mod = 'Y*' THEN 1 ELSE 0 END AS mod_required,
    CASE WHEN pa.v_rmv = 'Y*' THEN 1 ELSE 0 END AS rmv_required,
    pa.title_cn,
    pa.title_en,
    pa.cn_explanation,
    pa.en_explanation
FROM small_cell.small_cell_param_group pg
JOIN small_cell.rela_groupid_paramid rela
    ON rela.param_group_id = pg.id
JOIN small_cell.small_cell_param pa
    ON pa.param_id = rela.param_id
ORDER BY
    pg.param_version,
    pg.id,
    rela.`order`;
```

## 十二、最终结果：用 path 代表一个参数

最终整理结果中，一个参数必须使用 `path` 作为唯一代表。这里的 `path` 对应数据库字段 `name_path`：

```text
path = small_cell.small_cell_param.name_path
```

例如：

```text
path = CELL.{i}.LTE_CELL_ID
```

`mib_dn` 只是命令字段名，不能单独代表一个参数。`param_id` 是数据库记录 ID，参数名称只是显示文本，也不能作为最终唯一标识。

同一个 `mib_dn` 可能出现在不同路径下，例如：

```text
CELL.{i}.INDEX
CELL.{i}.NEIGHBOR.{j}.INDEX
```

这两个参数的 `mib_dn` 都可能是 `INDEX`，但实际参数不同。如果只按 `mib_dn` 合并，会导致校验规则错误。

推荐定义：

```text
参数唯一代表 = path
path = name_path
mib_dn = 命令字段名
param_id = 数据库参数记录 ID
```

建议最终使用以下维度区分规则：

```text
参数主键 = param_version + path
参数所在操作组 = param_group_id + display_order
操作校验规则 = action_type
```

相同 `path` 的不同操作规则应合并到同一参数下，但分别保留 `add`、`mod`、`rmv`、`lst` 的规则；不同版本的规则不能互相覆盖。

### 推荐最终结果结构

```text
path：
param_id：
mib_dn：
参数中文名：
参数英文名：
参数版本：
默认值：
是否动态生效：
中文说明：
英文说明：

操作规则：
  add：支持、是否必填、前端正则、数据类型、取值范围、长度、枚举值
  mod：支持、是否必填、前端正则、数据类型、取值范围、长度、枚举值
  rmv：支持、是否必填
  lst：支持、是否必填
```

如果 `path` 为空，应单独标记为异常数据，不能直接用 `mib_dn` 静默替代。

### SQL 查询时保留 path

建议将 `name_path` 明确查询为 `path`：

```sql
SELECT
    pg.id AS param_group_id,
    pg.keyword AS operation_keyword,
    pg.param_version,
    rela.`order` AS display_order,
    pa.param_id,
    pa.name_path AS path,
    pa.mib_dn,
    pa.param_name,
    pa.param_name_en,
    pa.v_type,
    pa.js_regex,
    pa.dft_value,
    pa.v_dynamic,
    pa.v_lst,
    pa.v_add,
    pa.v_mod,
    pa.v_rmv,
    CASE WHEN pa.v_lst = 'Y*' THEN 1 ELSE 0 END AS lst_required,
    CASE WHEN pa.v_add = 'Y*' THEN 1 ELSE 0 END AS add_required,
    CASE WHEN pa.v_mod = 'Y*' THEN 1 ELSE 0 END AS mod_required,
    CASE WHEN pa.v_rmv = 'Y*' THEN 1 ELSE 0 END AS rmv_required,
    pa.title_cn,
    pa.title_en,
    pa.cn_explanation,
    pa.en_explanation
FROM small_cell.small_cell_param_group pg
JOIN small_cell.rela_groupid_paramid rela
    ON rela.param_group_id = pg.id
JOIN small_cell.small_cell_param pa
    ON pa.param_id = rela.param_id
ORDER BY
    pg.param_version,
    path,
    rela.`order`;
```

## 十三、后续 AI 整理建议

从测试环境导出数据后，建议让 AI 按以下维度整理：

```text
param_version
path
action_type
param_group_id
operation_keyword
```

必须使用 `path` 代表一个参数，不能只按 `mib_dn`、参数名称或 `param_id` 合并。相同 `path` 的记录应合并操作规则，但保留版本差异。

每个参数建议输出：

```text
path：
参数组 ID：
操作关键字：
参数版本：
显示顺序：
参数 ID：
参数中文名：
参数英文名：
MIB 字段名：
参数路径：
支持查询：
支持新增：
支持修改：
支持删除：
新增是否必填：
修改是否必填：
删除是否必填：
数据类型：
最小值：
最大值：
最小长度：
最大长度：
枚举值：
前端正则：
是否禁止中文：
默认值：
是否动态生效：
中文说明：
英文说明：
规则是否被代码支持：
前端规则与后台规则是否一致：
备注：
```

## 十四、重点注意事项

1. `v_lst` 接口通常主要返回查询索引节点，不一定包含完整校验规则。
2. 要整理校验规则，应重点关注 `v_add`、`v_mod`、`v_rmv` 对应的参数详情。
3. `must` 是根据 `Y*` 推导出来的，不是数据库原始字段。
4. `js_regex` 主要用于前端，`v_type` 是后台校验的核心。
5. 最终整理结果必须使用 `path=name_path` 代表一个参数。
6. 同一个 `mib_dn` 可能对应多个不同的 `path`，不能只按 `mib_dn` 合并。
7. `v_type` 如果是代码未识别的格式，应单独标记，不能默认认为没有校验。
8. `hardwareVersion`、`param_version` 和参数组 ID 都可能影响最终规则。
9. 数据库中 `v_type` 和 `js_regex` 可能存在不一致，整理时应分别展示并标记差异。
10. `mib_dn`、`name_path`、参数名称不是同一个概念，不能混用。
11. 前端校验可以被绕过，最终安全性应以后台校验为准。

## 十五、本机数据库初步检查结果

本机 MySQL 已确认可以连接：

```text
地址：127.0.0.1
端口：3307
数据库：small_cell
```

已确认三张核心表存在：

| 表 | 当前记录数（约） |
|---|---:|
| `small_cell_param` | 7231 |
| `small_cell_param_group` | 1919 |
| `rela_groupid_paramid` | 7273 |

### 实际字段差异

本机数据库的实际字段名称与部分代码/示例存在差异，后续 SQL 应以实际表结构为准：

1. `small_cell_param` 中实际存在 `PARAM_VERSION`，不是 `param_version` 小写字段；MySQL 通常不区分字段大小写，但建议保持数据库原始写法。
2. `small_cell_param` 同时存在 `SOFTWARE_VERSION`。
3. `rela_groupid_paramid` 中使用 `software_version`，没有发现 `param_version` 字段。
4. `small_cell_param` 中确认存在 `NAME_PATH`、`MIB_DN`、`V_TYPE`、`JS_REGEX`、`DFT_VALUE`、`V_DYNAMIC`、`V_ADD`、`V_MOD`、`V_LST`、`V_RMV` 等核心字段。
5. `small_cell_param_group` 中存在 `param_version`、`keyword`、`add_path`，并且还有 `parent_id`、`cell_number`、`cell_index_location` 等参数组辅助字段。

### 当前数据概况

已导入参数数据包含 22 个参数版本，包括：

```text
436Q1.0、BAIBLQ1.0、BaiBNX1.0、BLX1.0、BSC1.0、BTS1.0、CA2.0、
CR4.0、DXDF1.0、EA4.0、EA4.0DUAL、ENB_DEFAULT_098、ENB_DEFAULT_181、
MLN1.0、MLQ1.0、NBIOT1.0、Nova430、Nova430i、QB1.0、QC3.1、QC4.2、QC4.2T
```

初步质量检查结果：

| 检查项 | 结果 |
|---|---:|
| 参数记录数 | 7226（查询统计结果） |
| `path` 为空的参数数 | 0 |
| 参数版本数 | 22 |
| `v_type` 为空的参数数 | 25 |
| `js_regex` 非空的参数数 | 768 |
| `v_add='Y*'` 的参数数 | 851 |
| `v_mod='Y*'` 的参数数 | 200 |
| `v_rmv='Y*'` 的参数数 | 148 |

目前 `NAME_PATH` 没有发现空值，适合按照：

```text
param_version + path
```

作为参数规则的主要整理维度。后续仍需重点处理 `v_type` 为空、同一 `path` 在不同版本下规则不同、参数组关系以及代码中特殊处理的情况。

## 十六、按参数版本生成 XML

已根据本机 `small_cell` 数据生成 XML 文件，输出目录为：

```text
OMC/plan/MML参数校验规则XML/
```

共生成 22 个文件，每个参数版本一个独立文件：

```text
436Q1.0.xml
BAIBLQ1.0.xml
BaiBNX1.0.xml
BLX1.0.xml
BSC1.0.xml
BTS1.0.xml
CA2.0.xml
CR4.0.xml
DXDF1.0.xml
EA4.0.xml
EA4.0DUAL.xml
ENB_DEFAULT_098.xml
ENB_DEFAULT_181.xml
MLN1.0.xml
MLQ1.0.xml
NBIOT1.0.xml
Nova430.xml
Nova430i.xml
QB1.0.xml
QC3.1.xml
QC4.2.xml
QC4.2T.xml
```

每个 XML 文件的根节点为：

```xml
<mmlParameterRules version="版本号" pathField="NAME_PATH">
```

文件内容包括：

1. `summary`：该版本的参数数量、参数组数量和关联关系数量；
2. `parameters`：该版本全部参数，以 `path` 属性代表参数；
3. 参数原始字段：名称、`MIB_DN`、`V_TYPE`、`JS_REGEX`、默认值、操作标识等；
4. `validation`：按 `add`、`mod`、`rmv`、`lst` 拆分支持状态和必填状态；
5. `parameterGroups`：参数组原始信息；
6. `relations`：参数组和参数之间的关联关系及显示顺序。

参数节点示意：

```xml
<row path="参数路径" paramId="参数ID">
    <field name="NAME_PATH">参数路径</field>
    <field name="V_TYPE">数据类型及范围</field>
    <field name="JS_REGEX">前端正则</field>
    <field name="V_ADD">Y*</field>
    <field name="V_MOD">Y</field>
    <validation path="参数路径" frontEndRule="前端正则" backendRule="数据类型及范围">
        <add supported="true" required="true" rawFlag="Y*" />
        <mod supported="true" required="false" rawFlag="Y" />
    </validation>
</row>
```

注意：XML 中的 `path` 来自 `small_cell_param.NAME_PATH`，这是最终代表一个参数的核心字段；`MIB_DN`、`PARAM_ID` 和参数名称均作为辅助信息保留。

## 十七、参数版本与 CellPlatformType 映射关系

### 1. 映射结论

`PARAM_VERSION` 与 Java 枚举 `CellPlatformType` 存在映射关系，但不是简单的同名一对一映射。系统实际通过以下链路识别平台：

```text
产品类型/product
    -> product_type.param_model
    -> CellPlatformType

软件版本/software_version
    -> rela_param_version.param_version
```

平台识别的主要代码位于：

```text
cellHandler/src/main/java/com/baicells/cellHandler/beans/cache/CellCache.java
```

`getPlatformVersionBySoftwareVersion` 的判断优先级大致为：

1. 根据产品类型匹配；
2. 根据 `product_type.param_model` 转换为 `CellPlatformType`；
3. 根据 `product_re_type_regx` 匹配默认平台；
4. 根据 `rela_param_version` 查询软件版本对应的参数版本；
5. 软件版本未命中数据库时，再根据软件版本前缀进行兜底判断。

因此，参数版本 XML 的版本号不能直接替换为平台枚举名。XML 应继续使用 `PARAM_VERSION` 命名，平台枚举作为附加元数据记录。

### 2. 当前数据中确认的映射

| 参数版本 `PARAM_VERSION` | `CellPlatformType` | 映射级别 | 依据和说明 |
|---|---|---|---|
| `EA4.0` | `Intel` | 明确 | 产品类型 `RTS` 对应 `EA4.0`，其 `param_model` 为 `Intel` |
| `EA4.0DUAL` | `Intel` | 明确 | 产品类型 `RTD` 的 `param_model` 为 `Intel` |
| `QC3.1` | `QA_V3` | 明确 | 产品类型 `QAFB` 对应 `QC3.1`，其 `param_model` 为 `QA_V3` |
| `QC4.2` | `QA_V4` | 明确 | 产品类型 `QAFA` 对应 `QC4.2`，其 `param_model` 为 `QA_V4` |
| `QC4.2T` | `QA_V4` | 明确 | 产品类型 `QATA` 对应 `QC4.2T`，其 `param_model` 为 `QA_V4` |
| `QB1.0` | `QB` | 明确 | `CellCache` 对 `QB1.0` 有明确的参数版本判断 |
| `BAIBLQ1.0` | `BAIBLQ` | 高可信 | 产品类型 `BAIBLQ` 对应 `BAIBLQ1.0`；代码将 `QA_V3.1` 归类为 `BAIBLQ` |
| `MLQ1.0` | `MLQ` | 高可信 | 产品类型 `MLQ` 对应 `MLQ1.0`，`param_model` 为 `MLQ` |
| `BLX1.0` | `BLX` | 高可信 | 产品类型映射中存在 `BLX` 平台 |
| `BSC1.0` | `BSC` | 高可信 | 产品类型 `BSC` 对应 `BSC1.0` |
| `BTS1.0` | `BTS` | 高可信 | 产品类型 `BTS` 对应 `BTS1.0` |
| `NBIOT1.0` | `NB_IOT` | 高可信 | 产品类型 `NBIOT` 对应 `NBIOT1.0`，`param_model` 为 `NB_IOT` |
| `DXDF1.0` | `DXDF` | 高可信 | 产品类型 `DXDF` 对应 `DXDF1.0` |
| `BaiBNX1.0` | `BaiBNX` | 高可信 | 产品类型 `BaiBNX` 对应 `BaiBNX1.0` |
| `MLN1.0` | `MLN_SC` / `MLN_CA` / `MLN_DC` | 明确为多对多 | `MLN-SC`、`MLN-CA`、`MLN-DC` 使用同一参数版本系列，具体枚举由产品名称继续区分 |
| `CR4.0` | `Intel_CR` 系列 | 部分明确 | `CR-B4860` 对应 `CR4.0`；SC、DC、CA、TC 由产品名称继续区分 |
| `436Q1.0` | `QA_436Q_SC` / `QA_436Q_DC` / `QA_436Q_CA` | 不能仅凭版本确定 | 代码支持三种 436Q 子类型，但当前参数版本是统一的 `436Q1.0` |
| `Nova430` | `Nova430_SC` / `Nova430_DC` / `Nova430_CA` | 不能仅凭版本确定 | SC、DC、CA 由产品名称继续区分 |
| `Nova430i` | `Nova430i_SC` / `Nova430i_DC` / `Nova430i_CA` | 不能仅凭版本确定 | SC、DC、CA 由产品名称继续区分 |
| `CA2.0` | `Intel_CA` | 待确认 | 当前代码和本机产品类型数据没有形成直接、完整的证据链 |
| `ENB_DEFAULT_098` | `ENB_DEFAULT_098` | 明确 | 默认参数模型与枚举同名 |
| `ENB_DEFAULT_181` | `ENB_DEFAULT_181` | 明确 | 默认参数模型与枚举同名 |

### 3. 产品类型与参数版本的代码证据

在以下代码中存在产品类型到参数版本的明确关系：

```text
OMCWebServer/src/main/java/com/baicells/omc/busi/cell/cpeinfos/service/impl/CellCpeInfosServiceImpl.java
```

核心关系包括：

```text
RTS          -> EA4.0
RTD          -> EA4.0DUAL
QAFB         -> QC3.1
QAFA         -> QC4.2
QATA         -> QC4.2T
BAIBLQ       -> BAIBLQ1.0
MLQ          -> MLQ1.0
CR-B4860     -> CR4.0
NBIOT        -> NBIOT1.0
DXDF         -> DXDF1.0
```

该代码还记录了以下平台子类型的参数版本命名方式：

```text
QRTB-SC      -> 436Q_SC1.0
QRTB-DC      -> 436Q_DC1.0
QRTB-CA      -> 436Q_CA1.0
Nova 430-SC  -> Nova430_SC
Nova 430-DC  -> Nova430_DC
Nova 430-CA  -> Nova430_CA
Nova 430i-SC -> Nova430i_SC
Nova 430i-DC -> Nova430i_DC
Nova 430i-CA -> Nova430i_CA
MLN-SC       -> MLN_SC1.0
MLN-DC       -> MLN_DC1.0
MLN-CA       -> MLN_CA1.0
```

注意：上述带子类型后缀的版本名在当前 `small_cell_param` 查询结果中没有全部出现。当前数据库实际存在的是统一版本 `436Q1.0`、`Nova430`、`Nova430i` 和 `MLN1.0`，因此不能直接把代码中的历史/产品匹配名称当作当前参数表版本。

### 4. `rela_param_version` 的作用和当前覆盖范围

`OMCWebServer` 的 `DMCache` 会从以下表加载软件版本和参数版本的关系，并写入 Redis 的 `param_ver_soft_ver`：

```text
small_cell.rela_param_version
```

关系方向为：

```text
software_version -> param_version
```

本机数据库检查结果：

| 检查项 | 结果 |
|---|---:|
| 关系记录数 | 46 |
| 已覆盖参数版本数 | 5 |
| 已覆盖版本 | `EA4.0`、`GA3.0`、`QB1.0`、`QC3.1`、`QC4.2` |

因此，`rela_param_version` 不能单独解释当前参数表中的全部 22 个参数版本。其余版本主要依赖产品类型、产品型号和代码中的平台匹配逻辑。

### 5. 使用建议

XML 文件仍按参数版本保存：

```text
QC4.2.xml       -> platformType = QA_V4
QC4.2T.xml      -> platformType = QA_V4
MLN1.0.xml      -> platformType = MLN_SC / MLN_CA / MLN_DC
```

建议 XML 或后续映射文件同时保留以下字段：

```text
param_version
platform_type
product_name
software_version
mapping_confidence
mapping_source
```

其中：

- `param_version`：参数规则实际使用的版本；
- `platform_type`：Java 业务层平台枚举；
- `product_name`：用于区分 SC、DC、CA、TC 等子类型；
- `software_version`：设备软件版本；
- `mapping_confidence`：建议填写 `明确`、`高可信`、`部分明确` 或 `待确认`；
- `mapping_source`：记录来自 `product_type`、`rela_param_version` 或 Java 代码。

最终原则：参数规则的唯一标识仍然是：

```text
param_version + path
```

`CellPlatformType` 只作为平台分类和业务处理维度，不能替代 `param_version`，也不能替代参数的 `path`。

## 十八、XML 字段含义说明

前面的 XML 生成说明只概括了 XML 的内容结构，没有逐项解释所有字段。下面是当前 XML 文件的完整字段说明。所有参数版本 XML 使用相同的结构，例如：

```text
OMC/plan/MML参数校验规则XML/436Q1.0.xml
```

### 1. 根节点和统计节点

| 节点或属性 | 含义 |
|---|---|
| `mmlParameterRules` | XML 根节点，表示一个参数版本的 MML 参数规则集合 |
| `version` | 当前 XML 对应的 `PARAM_VERSION`，例如 `436Q1.0` |
| `pathField` | 参数路径的来源字段，当前固定为 `NAME_PATH`；XML 中统一使用 `path` 表示参数 |
| `summary` | 当前版本的统计信息 |
| `parameterCount` | 参数数量，对应 `parameters` 下的参数记录数 |
| `parameterGroupCount` | 参数组数量，对应 `parameterGroups` 下的记录数 |
| `relationCount` | 参数组与参数的关联关系数量，对应 `relations` 下的记录数 |

### 2. 参数节点

`parameters` 表示当前参数版本的全部参数。每个 `row` 表示一个参数，参数唯一识别维度为：

```text
param_version + row.path
```

| 节点或属性 | 含义 |
|---|---|
| `parameters` | 当前参数版本的参数集合 |
| `row` | 一条参数定义记录 |
| `row.path` | 参数完整路径，来源于 `NAME_PATH`；这是 XML 中代表一个参数的核心字段 |
| `row.paramId` | 数据库参数 ID，来源于 `PARAM_ID`，用于与参数组关系关联 |
| `field` | 参数在数据库中的原始字段和值 |
| `field.name` | 数据库字段名 |

### 3. 参数原始字段

参数 `row` 中的 `field` 保留了数据库原始字段，具体含义如下：

| 字段 | 含义 |
|---|---|
| `PARAM_ID` | 参数记录 ID，用于和参数组关系表关联；不是参数最终唯一标识 |
| `PARAM_NAME` | 参数中文名称或显示名称 |
| `PARAM_NAME_EN` | 参数英文名称 |
| `NAME_PATH` | 参数完整路径；XML 中映射为 `row.path`，是参数的主要标识 |
| `MIB_DN` | MML 命令中的参数字段名；可能与其他路径下的参数重复，不能单独作为唯一标识 |
| `DFT_VALUE` | 参数默认值 |
| `V_TYPE` | 后台和前端使用的数据类型及取值规则，例如 `int-[0:359]`、`string-[0:64]`、`enum-{AGL}` |
| `V_WRITABLE` | 参数是否可写；`-` 表示数据库中的原始标记值，不在 XML 中重新解释 |
| `V_LST` | 参数是否支持查询操作；通常使用 `Y`、`N` 等标记 |
| `V_MOD` | 参数是否支持修改操作 |
| `V_ADD` | 参数是否支持新增操作 |
| `V_RMV` | 参数是否支持删除操作 |
| `IS_LEAF` | 是否为参数树叶子节点；`Y` 表示叶子参数 |
| `DISP_ORD` | 参数在原始数据中的显示顺序 |
| `STOP_SIGN` | 原始流程或处理停止标记 |
| `MEMO` | 参数备注；为空或 `xsi:nil="true"` 表示数据库没有备注 |
| `V_DYNAMIC` | 参数是否支持动态生效；保留数据库原始值 |
| `PARAM_VERSION` | 参数所属参数版本 |
| `MOBILE_SUPPORT` | 是否支持移动网络场景 |
| `BROADBAND_SUPPORT` | 是否支持宽带场景 |
| `JS_REGEX` | 前端输入校验正则表达式；为空表示数据库未配置前端正则，不代表没有 `V_TYPE` 校验 |
| `TITLE_CN` | 中文页面提示标题 |
| `TITLE_EN` | 英文页面提示标题 |
| `PLATFORM_SUPPORT` | 原始平台支持标记；当前 XML 保留数据库值，不将数字直接转换成枚举名称 |
| `CN_EXPLANATION` | 中文参数说明 |
| `EN_EXPLANATION` | 英文参数说明 |
| `SOFTWARE_VERSION` | 参数关联的软件版本；为空表示该参数记录没有直接填写软件版本 |
| `second_confirm` | 是否需要二次确认的原始标记 |
| `confirm_en` | 英文二次确认提示内容 |
| `confirm_cn` | 中文二次确认提示内容 |

### 4. `xsi:nil`、空值和原始值

XML 中存在两种空值表现：

```xml
<field name="DFT_VALUE" />
<field name="MEMO" xsi:nil="true" />
```

含义分别是：

- 空元素表示该字段没有文本内容；
- `xsi:nil="true"` 表示数据库查询结果为 `NULL`；
- XML 没有将 `NULL`、空字符串和特殊标记统一转换，便于保留数据库原始信息；
- 读取 XML 时应同时判断元素文本和 `xsi:nil` 属性。

### 5. `validation` 校验节点

每个参数下有一个 `validation` 节点，是根据原始操作字段生成的便于使用的校验视图：

| 节点或属性 | 含义 |
|---|---|
| `validation` | 当前参数的综合校验规则 |
| `validation.path` | 当前参数路径，与 `row.path` 相同 |
| `frontEndRule` | 前端校验规则，来源于 `JS_REGEX` |
| `backendRule` | 后台校验规则，来源于 `V_TYPE` |
| `lst` | 查询操作规则 |
| `add` | 新增操作规则 |
| `mod` | 修改操作规则 |
| `rmv` | 删除操作规则 |
| `supported` | 是否支持该操作；通常由对应 `V_*` 是否为 `N` 推导 |
| `required` | 该操作是否必填；对应原始标记为 `Y*` 时为 `true` |
| `rawFlag` | 对应的数据库原始操作标记，例如 `Y`、`Y*`、`N` |

`validation` 是整理后的派生信息，`field` 是数据库原始信息。两者同时保留，便于核对派生结果是否正确。

### 6. 参数组节点

`parameterGroups` 表示页面上的参数组或参数菜单，每个 `row` 表示一个参数组。

| 字段 | 含义 |
|---|---|
| `id` | 参数组 ID |
| `param_name` | 参数组中文名称 |
| `param_name_en` | 参数组英文名称 |
| `keyword` | 参数组操作关键字 |
| `parent_id` | 父参数组 ID；根节点通常指向自身或顶层节点 |
| `param_version` | 参数组所属参数版本 |
| `add_path` | 新增操作使用的路径 |
| `v_lst` | 参数组是否支持查询操作 |
| `v_mod` | 参数组是否支持修改操作 |
| `v_add` | 参数组是否支持新增操作 |
| `v_rmv` | 参数组是否支持删除操作 |
| `mobile_support` | 参数组是否支持移动网络场景 |
| `broadband_support` | 参数组是否支持宽带场景 |
| `platform_support` | 参数组的平台支持原始标记 |
| `cell_number` | 小区或小区实例数量相关配置 |
| `cell_index_location` | 小区索引在路径或参数中的位置 |
| `second_confirm` | 参数组是否需要二次确认 |
| `confirm_en` | 英文二次确认提示 |
| `confirm_cn` | 中文二次确认提示 |
| `dis_order` | 参数组在页面上的显示顺序 |

### 7. 关联关系节点

`relations` 表示参数组和参数之间的关系，每个 `row` 表示一个参数属于某个参数组的关联记录。

| 字段 | 含义 |
|---|---|
| `param_group_id` | 参数组 ID，对应 `parameterGroups` 中的 `id` |
| `param_id` | 参数 ID，对应 `parameters` 中的 `row.paramId` 和 `PARAM_ID` |
| `software_version` | 该关联关系对应的软件版本或参数版本；当前生成 XML 时保留原始字段值 |
| `display_order` | 参数在参数组中的显示顺序，来源于关联表的 `order` 字段 |
| `platform_support` | 该关联关系的平台支持原始标记 |

关联关系可以理解为：

```text
parameterGroups.id
    -> relations.param_group_id

parameters.row.paramId
    -> relations.param_id
```

### 8. XML 字段与数据库字段的关系

| XML 内容 | 数据库来源或生成方式 |
|---|---|
| `row.path` | `small_cell_param.NAME_PATH` |
| `row.paramId` | `small_cell_param.PARAM_ID` |
| `parameters/field` | `small_cell_param` 的原始字段 |
| `validation` | 根据 `V_LST`、`V_ADD`、`V_MOD`、`V_RMV`、`V_TYPE`、`JS_REGEX` 派生 |
| `parameterGroups/field` | `small_cell_param_group` 的原始字段 |
| `relations/field` | `rela_groupid_paramid` 的原始字段及 XML 生成时保留的版本字段 |
| `summary` | XML 生成时对参数、参数组和关系记录计数 |

使用 XML 时建议：

1. 用 `row.path` 作为参数标识；
2. 用 `validation` 读取操作支持和校验规则；
3. 用 `field` 获取完整数据库原始字段；
4. 用 `relations` 判断参数所属参数组及显示顺序；
5. 不要用 `MIB_DN`、参数名称或 `PARAM_ID` 单独替代 `path`。

## 十九、交接目标：仅依赖本文档和 XML 获取参数校验规则

### 1. 是否能够直接使用

可以。另一位 agent 只要：

1. 阅读本文档；
2. 读取 `MML参数校验规则XML` 目录下与目标参数版本对应的 XML 文件；
3. 在 `parameters/row` 中按 `path` 查找参数；
4. 读取该参数的 `validation` 和原始 `field`；

就可以获得该参数的操作权限、必填规则、数据类型、取值范围、枚举值、字符串长度和正则校验规则。

参数查询路径：

```text
指定版本 XML
    -> parameters
    -> row[@path='目标路径']
    -> validation
    -> field[@name='V_TYPE'] / field[@name='JS_REGEX']
```

参数的最终标识必须使用：

```text
row.path
```

如果需要跨版本比较，则使用：

```text
XML 文件对应的 version + row.path
```

### 2. 取值规则的读取优先级

对于一个参数，应按以下顺序读取：

| 顺序 | XML 内容 | 用途 |
|---:|---|---|
| 1 | `row.path` | 确认查询到的是目标参数 |
| 2 | `validation` 下的操作节点 | 判断 `lst`、`add`、`mod`、`rmv` 是否支持及是否必填 |
| 3 | `backendRule` | 获取后台类型、范围、长度或枚举规则；其值来自 `V_TYPE` |
| 4 | `frontEndRule` | 获取前端正则补充限制；其值来自 `JS_REGEX` |
| 5 | `DFT_VALUE` | 获取默认值 |
| 6 | `MIB_DN`、名称和说明字段 | 用于展示和生成命令，不作为参数唯一标识 |

后台有效性判断不能只看 `frontEndRule`。应以 `backendRule` 为基础，再将非空的 `frontEndRule` 作为附加限制：

```text
有效取值 = V_TYPE 规则 AND JS_REGEX 规则（JS_REGEX 非空时）
```

### 3. `V_TYPE` 的取值规则格式

XML 当前没有单独的 `minValue`、`maxValue`、`enumValues` 字段，而是将完整规则保存在 `V_TYPE` 和 `validation.backendRule` 中。读取 agent 必须解析该字符串。

#### 3.1 整数或无符号整数范围

格式：

```text
int-[最小值:最大值]
unsignedInt-[最小值:最大值]
```

示例：

```text
int-[-90:90]
unsignedInt-[0:65535]
```

含义：

```text
数据类型为整数
最小值 <= 参数值 <= 最大值
```

方括号表示包含边界值。例如 `int-[-1:31]` 允许 `-1` 和 `31`。

无范围时：

```text
int
unsignedInt
```

表示类型已知，但 XML 中没有进一步的最小值和最大值，不能自行推断业务范围。

#### 3.2 字符串长度

格式和含义如下：

```text
string-[最小长度:最大长度]
string-最大长度
string
```

示例：

```text
string-[0:64]  -> 长度范围为 0 到 64
string-[5:6]   -> 长度范围为 5 到 6
string-256     -> 最大长度为 256
string         -> 字符串类型，但没有在 XML 中声明长度
```

#### 3.3 枚举值

格式：

```text
enum-{值1,值2,值3}
```

示例：

```text
enum-{AGL}
```

表示实际允许值集合为 `{AGL}`。

如果出现两个集合：

```text
enum-{显示值1,显示值2}-{实际值1,实际值2}
```

例如：

```text
enum-{true,false}-{1,0}
```

表示界面值和设备实际值存在映射：

```text
true  -> 1
false -> 0
```

因此，两个集合按位置一一对应，不能把两个大括号中的内容合并成一个无序集合。

#### 3.4 命名类型

以下类型表示系统预定义的数据格式：

```text
Ipv4Addr
Ipv6Addr
```

它们代表 IPv4 地址或 IPv6 地址格式。XML 中没有继续展开为正则表达式或数值范围，读取 agent 应按类型名称识别，不应把它们当作普通字符串。

#### 3.5 空的 `V_TYPE`

当前 22 个 XML 文件共包含 7226 条参数记录，其中 25 条的 `V_TYPE` 为空。对于这类参数：

- 不能从 XML 推断最小值、最大值、长度或枚举值；
- 应标记为“数据库未配置 `V_TYPE`”；
- 仍需检查 `JS_REGEX` 是否提供前端限制；
- 仍需检查 `V_ADD`、`V_MOD`、`V_RMV`、`V_LST` 获取操作权限；
- 不能因为 `V_TYPE` 为空就认定参数可以接受任意值。

### 4. 操作权限和必填规则

每个参数下的操作节点均有三个关键属性：

```xml
<mod supported="true" required="false" rawFlag="Y" />
```

判断方式：

| 属性 | 判断方式 | 含义 |
|---|---|---|
| `supported` | 对应原始 `V_*` 不是 `N` | 是否支持该操作 |
| `required` | 对应原始 `V_*` 等于 `Y*` | 参数在该操作中是否必填 |
| `rawFlag` | 直接读取数据库原始值 | 用于追溯和核对 |

四种操作与数据库字段的对应关系：

```text
lst -> V_LST
add -> V_ADD
mod -> V_MOD
rmv -> V_RMV
```

`required` 只表示是否必填，不表示取值范围；取值范围必须继续读取 `backendRule` 或 `V_TYPE`。

### 5. 前端正则和后台规则的关系

```text
backendRule = V_TYPE
frontEndRule = JS_REGEX
```

当前 XML 统计结果：

| 项目 | 数量 |
|---|---:|
| XML 文件 | 22 |
| 参数记录 | 7226 |
| `V_TYPE` 为空 | 25 |
| `JS_REGEX` 为空 | 6458 |

`JS_REGEX` 为空是正常情况，表示没有配置额外的前端正则；此时仍应执行 `V_TYPE` 规则。`V_TYPE` 和 `JS_REGEX` 都存在时，两者都应满足。

### 6. 参数校验规则读取模板

另一位 agent 可以按以下结构整理任何参数：

```text
版本：从 XML 根节点 version 读取
path：从 parameters/row@path 读取
参数名称：field[@name='PARAM_NAME']
MIB 字段：field[@name='MIB_DN']
数据类型和取值规则：field[@name='V_TYPE'] 或 validation@backendRule
前端正则：field[@name='JS_REGEX'] 或 validation@frontEndRule
默认值：field[@name='DFT_VALUE']

查询：validation/lst
新增：validation/add
修改：validation/mod
删除：validation/rmv

每个操作读取：supported、required、rawFlag
```

### 7. 最终检查结论和限制

当前文档和 XML 已足以让 agent 获取数据库中已经配置的参数校验规则，尤其是：

- 整数范围；
- 无符号整数范围；
- 字符串长度；
- 枚举值及显示值到设备值的映射；
- IPv4/IPv6 类型；
- 前端正则；
- 各操作的支持状态和必填状态。

仍需明确标记的情况只有：

1. `V_TYPE` 为空，数据库没有提供明确类型或范围；
2. `V_TYPE` 只有基础类型，没有提供范围，例如 `string`、`unsignedInt`；
3. `V_TYPE` 与 `JS_REGEX` 同时存在时，需要同时满足两套规则；
4. 代码中的特殊业务校验不一定能由数据库 XML 完全表达，本文档已在“前端校验逻辑”和“后台校验逻辑”章节说明；
5. 如果目标是复现 Java 后台的全部特殊校验，还需要结合代码；如果目标是读取数据库中配置的参数取值规则，本文档和 XML 已经足够。

### 8. 当前 XML 中发现的特殊 `V_TYPE`

对 22 个 XML 文件的全部 7226 条参数记录进行检查后，除了常见的 `int`、`unsignedInt`、`string`、`enum`、`Ipv4Addr` 和 `Ipv6Addr`，还发现以下特殊类型：

| `V_TYPE` | 出现数量 | 说明 | 读取方式 |
|---|---:|---|---|
| `stringList` | 72 | 字符串列表类型 | 不能按单个普通字符串处理；需要结合参数的 `JS_REGEX`、默认值和业务分隔格式判断列表元素规则 |
| `unsignedIntList` | 39 | 无符号整数列表类型 | 每个列表元素应按无符号整数处理，但 XML 当前没有统一声明列表长度、分隔符或每个元素的范围 |
| `uniqueInt` | 2 | 特殊整数类型 | 表示整数值还可能需要满足唯一性约束；当前 XML 未展开唯一性作用域 |
| `strint` | 2 | 字符串形式的整数类型 | 应按项目实际接口格式处理，不能直接等同于 `int` |
| `eunsignedInt` | 1 | 数据库中的特殊/历史类型名称 | 不能仅根据名称推断完整范围，应保留原始类型并标记为待确认 |
| `Ipv4AddrArr` | 1 | IPv4 地址数组类型 | 应按 IPv4 地址列表处理；当前 XML 未声明数组长度和分隔符 |
| `LocalUplinkIpAddr` | 1 | 本地上行 IP 地址类型 | 属于业务预定义类型；当前 XML 未展开为具体正则或网段范围 |
| `RemoteUplinkIpAddr` | 1 | 远端上行 IP 地址类型 | 属于业务预定义类型；当前 XML 未展开为具体正则或网段范围 |
| `enum` | 30 | 枚举类型但未在 `V_TYPE` 中列出枚举集合 | 必须检查 `JS_REGEX`、参数说明或对应业务代码；不能从 `enum` 三个字符本身推断允许值 |

### 9. 需要特别标记的“规则不完整”参数

如果出现以下情况，另一位 agent 不能声称已经知道精确取值范围，而应将规则状态标记为 `incomplete` 或“需要业务代码确认”：

1. `V_TYPE` 为空；
2. `V_TYPE` 为 `enum`，但没有枚举值集合；
3. `V_TYPE` 为 `stringList`、`unsignedIntList` 或 `Ipv4AddrArr`，但没有列表长度和分隔规则；
4. `V_TYPE` 为 `uniqueInt`，但没有唯一性范围说明；
5. `V_TYPE` 为 `LocalUplinkIpAddr`、`RemoteUplinkIpAddr` 或其他预定义类型，但没有对应格式定义；
6. `V_TYPE` 只有 `string`、`int` 或 `unsignedInt`，没有长度或数值范围；
7. 只有 `JS_REGEX` 而没有 `V_TYPE`，此时只能知道前端正则，不能补造后台类型范围。

这些情况不是 XML 丢失字段，而是数据库原始规则本身没有展开成更细的限制。XML 已保留原始值，后续可以根据 `path` 精确定位问题参数。

### 10. 对当前交接资料的最终建议

当前文件和 XML 已能覆盖普通参数的完整规则读取。为了让自动 agent 更稳定，建议其输出结果增加以下字段：

```text
rule_status
value_type
min_value
max_value
min_length
max_length
allowed_values
display_to_device_mapping
frontend_regex
operation_rules
notes
```

其中：

- `rule_status`：`complete`、`incomplete` 或 `code_required`；
- `value_type`：从 `V_TYPE` 解析出的基础类型；
- `min_value`、`max_value`：从 `int-[a:b]` 或 `unsignedInt-[a:b]` 解析；
- `min_length`、`max_length`：从 `string-[a:b]` 解析；
- `allowed_values`：从 `enum-{...}` 的第一个集合解析；
- `display_to_device_mapping`：从 `enum-{显示值}-{设备值}` 按位置建立映射；
- `frontend_regex`：从 `JS_REGEX` 或 `validation.frontEndRule` 读取；
- `operation_rules`：从 `validation` 的四个操作节点读取；
- `notes`：记录空类型、特殊类型和代码依赖。

结论：不需要重新生成当前 22 个 XML。需要补充的主要是对特殊 `V_TYPE` 和“不足以推断精确范围”的情况进行明确标记。这样另一位 agent 既能读取普通参数的完整校验规则，也不会对特殊参数作出错误推断。
