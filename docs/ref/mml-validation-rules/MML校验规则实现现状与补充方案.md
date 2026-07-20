# MML 校验规则实现现状与补充方案

更新时间：2026-07-20

## 1. 结论

当前 OMC 的 MML 校验规则实现是不完整的。

`param-mappings/*.xml` 中补充的 `min`、`max`、`enumValues`、`enumLabels` 等机型参数规则，目前没有完整进入 MML 控制台的展示和执行前校验链路。MML 现在主要依赖 `standard_params` 表里的全局标准参数元数据，而不是按当前设备参数模型读取 `param_mappings` 表里的机型规则。

用当前在线设备验证：

- 在线设备：`1202000240194DP0015`
- 产品类型：`FAP/mBS31001/CA`
- 参数模型：`BLQ`
- 规则源文件：`omcgo/data/param-mappings/BLQ.xml`
- 参数：`Device.DeviceInfo.AntennaInfo.Azimuth`

`BLQ.xml` 中该参数已经是：

```xml
<param name="Device.DeviceInfo.AntennaInfo.Azimuth" standardPath="Device.DeviceInfo.AntennaInfo.Azimuth" access="READ_WRITE" type="INT" changeApplies="Immediate" max="359" min="0"/>
```

但 MML 页面接口实际返回的 `AZIMUTH` 规则只有：

```json
{
  "mml_code": "AZIMUTH",
  "tr069_path": "Device.DeviceInfo.AntennaInfo.Azimuth",
  "value_type": "INT",
  "access_type": "READ_WRITE",
  "constraint_text": "≤ 359"
}
```

页面上也只显示：

```text
AZIMUTH Device.DeviceInfo.AntennaInfo.Azimuth
```

没有显示 `≥ 0`，也没有显示完整范围 `[0, 359]`。

原因是 `standard_params` 中该参数仍为：

```text
min_value = NULL
max_value = 359
```

而 MML 接口当前从 `standard_params` 派生 `constraint_text`，不是从当前设备的 `param_mappings` 派生。

## 2. 当前数据模型

### 2.1 `param_mappings`

表结构位置：

```text
omcgo/migrations/000001_init_schema.sql
```

当前可存储字段包括：

```text
standard_path
private_path
entry_type
access
data_type
change_applies
min_value
max_value
is_storable
is_supported
enum_values
enum_labels
mirror_with
source
```

这张表是按 `param_model_id + standard_path/private_path` 维度保存的，所以能表达不同参数模型之间的差异。

### 2.2 `standard_params`

当前可存储字段包括：

```text
standard_path
entry_type
access
data_type
change_applies
min_value
max_value
description
updated_fields
```

这张表是全局标准参数字典，不区分 BLQ、MLQ、ENB_DEFAULT 等参数模型。它适合保存标准路径、默认类型、通用范围和描述，不适合单独承载所有机型差异规则。

### 2.3 XML 加载器现状

`omcgo/internal/config/parammodel/model.go` 的 `xmlParamEntry` 当前解析：

```text
name
standardPath
access
type
changeApplies
min
max
enumValues
enumLabels
mirrorWith
store
supported
```

`omcgo/internal/config/parammodel/loader.go` 会把这些字段写入 `param_mappings`。

限制：

- 当前不解析 `defaultValue`。
- 当前不解析 `validationPattern`。
- 当前表结构也没有 `default_value` / `validation_pattern` 列。

所以即使 XML 中补了 `defaultValue` 或 `validationPattern`，当前 Go 加载器也不会把它们写入数据库，MML 更不会拿到。

## 3. 当前 MML 页面链路

### 3.1 参数列表接口

接口：

```text
GET /api/v1/mml/commands/:id/sub-fields?lang=&device_sn=&product_class=
```

主要实现：

```text
omcgo/internal/mml/console_handler.go
omcgo/internal/mml/console_service.go
omcgo/internal/mml/admin_repository.go
```

当前逻辑：

1. 根据 `device_sn` 或 `product_class` 找到当前产品的 `param_model_id`。
2. 用 `param_mappings` 判断某个 path 当前产品是否支持。
3. 真正返回字段元数据时，仍主要 JOIN `standard_params`。
4. `constraint_text_i18n` 由 `standard_params.min_value/max_value` 派生。
5. DTO 只返回了 `min_value`，没有独立返回 `max_value`。
6. 没有返回 `enum_values` / `enum_labels`。
7. 没有返回 `validation_pattern` / `default_value`。

关键现状：

```text
param_mappings 当前只参与“支持/不支持过滤”，没有参与“有效校验规则生成”。
```

### 3.2 前端渲染

主要实现：

```text
omcmb/frontend-core/src/types/mmlConsole.ts
omcmb/frontend-core/src/services/api/mmlApi.ts
omcmb/webcode/src/pages/mml/Console/adapters.ts
omcmb/webcode/src/pages/mml/Console/components/ConfigParamsModal.tsx
```

当前前端类型只稳定消费：

```text
constraintText
minValue
defaultValue
jsRegex
```

但配置参数弹框实际渲染时只展示：

```text
参数 label
standard path
普通 Input
```

没有显示 `constraintText`，也没有根据 `minValue/maxValue/enum` 切换输入控件或做实时校验。

`minValue` 当前只被用作 MOD/ADD 输入框的默认值：

```text
如果 p.minValue != null，则初始值 = minValue
```

这不是校验，只是默认填值。

### 3.3 flat group tree 接口

`omcgo/internal/mml/flat_group_tree.go` 的 `format=flat` 响应会把 `standard_params.min_value/max_value` 放进 MOD 参数的 `object_path`：

```text
Min/Max <- standard_params.min_value / max_value
```

但这仍是 `standard_params` 规则，不是当前设备 `param_mappings` 规则。

## 4. 当前 MML 执行链路

### 4.1 控制台直接执行

主要实现：

```text
omcgo/internal/mml/console_structured.go
omcgo/internal/mml/console_executor.go
omcgo/internal/mml/tr069_payload.go
```

当前做了这些校验：

- `command_id` 是否存在。
- 用户选择的 `standardPath` 是否属于该命令的 sub-field。
- `LST/MOD/ADD/RMV` 操作类型是否合法。
- `MOD` 是否至少有一个值。
- `ADD/RMV` 是否有目标对象。
- 实例号占位符数量是否匹配。
- TR-069 path 是否含非法前缀、未替换 `{i}`、非法字符。

当前没有做这些校验：

- 用户输入值是否低于 `min`。
- 用户输入值是否超过 `max`。
- 用户输入值是否属于枚举。
- 用户输入值是否匹配正则。
- 字符串长度是否按机型规则校验。
- 布尔/unsignedInt 之外的更多规则提示和阻断。

也就是说，MML 控制台“直接执行”当前主要校验命令和路径能不能编译成任务，不完整校验参数值是否合法。

### 4.2 路径支持校验和翻译

主要实现：

```text
omcgo/internal/mml/console_validate.go
omcgo/internal/mml/fanout.go
omcgo/internal/acs/path_translator.go
```

这里会使用 `param_mappings` 做路径翻译和支持性判断：

- 当前产品是否能识别。
- 当前 path 是否在该产品参数模型中存在。
- standardPath 下发前翻译为 privatePath。

但这条链路仍然没有用 `param_mappings.min_value/max_value/enum_values` 校验用户输入值。

### 4.3 脚本导入校验

主要实现：

```text
omcgo/internal/mml/script_import_validator.go
omcgo/internal/mml/pg_repository.go
```

脚本导入校验函数本身支持：

- 必填参数校验。
- 未知参数校验。
- 只读参数校验。
- 类型校验。
- `enum` 校验。
- `regex` 校验。
- `min/max` 范围校验。

但是实际加载规则时：

```sql
SELECT
  csf.command_id,
  csf.mml_code,
  sp.standard_path,
  lower(COALESCE(sp.data_type, 'string')),
  (sp.access = 'READ_WRITE'),
  csf.is_required,
  sp.min_value,
  sp.max_value
FROM mml_command_sub_fields csf
JOIN standard_params sp ON sp.id = csf.standard_path_id
```

随后调用：

```text
standardValidationRules(valueType, minValue, maxValue)
```

因此脚本导入校验当前也是基于 `standard_params`，不是基于目标设备的 `param_mappings`。

额外限制：

- 脚本行有设备 SN，但 `validateLineParameters` 当前没有把设备对应的 `param_model_id` 纳入规则选择。
- `standardValidationRules` 只从类型和 `min/max` 派生布尔、unsignedInt、范围规则。
- 没有从 `param_mappings.enum_values` 读取业务枚举。
- 没有正则来源，除 boolean/unsignedInt 的内置规则外不支持 XML 正则。

## 5. 设备参数树链路对比

设备参数树相关实现：

```text
omcgo/internal/device/device_param_handler.go
omcgo/internal/config/parammodel/validator.go
```

`MappingValidator` 使用 `param_mappings` 里的规则校验设备参数树写值：

- `access`
- `data_type`
- `min_value`
- `max_value`

但它也明确写了限制：

```text
没有 enum / pattern / forced inform 概念。
```

所以即使不看 MML，当前参数模型校验本身也只覆盖一部分字段。

## 6. 为什么不能只改 `standard_params`

短期把 `standard_params.min_value` 补成 `0`，可以让 `AZIMUTH` 在 MML 接口里显示 `[0, 359]` 或 `≥ 0`。

但这不是推荐方案，原因：

1. `standard_params` 是全局字典，不区分机型。
2. 历史规则里已经存在机型差异和冲突项。
3. 用户已确认冲突以历史规则为准，这个“历史规则”是按规则源/机型上下文判断的，不一定适合变成全局规则。
4. `param_mappings` 才是当前产品参数模型的实际支持集合，MML 已经用它判断 path 是否支持，继续用它派生校验规则更一致。

推荐原则：

```text
MML 展示和执行校验应使用“当前设备/产品参数模型下的有效规则”。
standard_params 作为兜底。
param_mappings 作为当前 paramModel 的优先来源。
```

## 7. 推荐补充方案

### 7.1 后端：统一有效规则模型

新增一个内部结构，表示 MML 当前设备上下文下的最终校验规则：

```text
EffectiveMMLParamRule
```

建议字段：

```text
standard_path
access
data_type
change_applies
min_value
max_value
enum_values
enum_labels
default_value
validation_pattern
constraint_text_i18n
source
```

规则来源优先级：

```text
param_mappings 当前 paramModel 非空字段
  > standard_params 全局字段
  > 类型派生规则，例如 boolean / unsignedInt
```

注意：

- `is_supported=false` 的 path 继续过滤掉，不应返回给控制台。
- 如果选择多台设备且参数模型不同，不能只按第一台设备显示规则。需要按所有目标设备逐一校验，页面可以提示“多机型规则可能不同”。

### 7.2 后端：补 MML 参数列表接口

修改位置：

```text
omcgo/internal/mml/admin_repository.go
omcgo/internal/mml/console_service.go
omcgo/internal/mml/sub_field_model.go
```

建议改动：

1. `ListEnrichedByCommand(ctx, commandID, paramModelID)` 在 `paramModelID != nil` 时 LEFT JOIN 当前模型的 `param_mappings pm`。
2. 返回字段使用有效值：

```text
effective_access         = COALESCE(pm.access, sp.access)
effective_data_type      = COALESCE(pm.data_type, sp.data_type)
effective_change_applies = COALESCE(pm.change_applies, sp.change_applies)
effective_min_value      = COALESCE(pm.min_value, sp.min_value)
effective_max_value      = COALESCE(pm.max_value, sp.max_value)
effective_enum_values    = pm.enum_values
effective_enum_labels    = pm.enum_labels
```

3. `constraint_text_i18n` 从有效规则派生，不再只看 `sp.min_value/max_value`。
4. DTO 增加：

```text
max_value
enum_values
enum_labels
value_constraint
rule_source
```

5. 保留 `standard_params.description` 作为说明文案来源。

### 7.3 后端：补控制台直接执行前校验

修改位置：

```text
omcgo/internal/mml/console_structured.go
omcgo/internal/mml/console_executor.go
omcgo/internal/mml/console_validate.go
```

建议新增校验点：

```text
StructuredToStatement 之后、CreateAndFanoutTask 之前
```

需要校验：

- MOD/ADD 的值类型。
- min/max。
- enum。
- regex。
- access 是否可写。

校验规则必须按目标设备解析：

```text
device_sn -> product_class -> product -> param_model_id -> param_mappings
```

如果一次选择多台设备：

- 对每个不同 `param_model_id` 都校验一次。
- 任一设备不通过，整次任务不创建，返回 422。
- 错误体要包含 `device_sn`、`mml_code`、`standard_path`、`value`、`rule`、`reason`。

这样可以避免页面漏拦截时，后端仍把非法值下发给真实基站。

### 7.4 后端：补脚本导入校验

修改位置：

```text
omcgo/internal/mml/script_import_validator.go
omcgo/internal/mml/pg_repository.go
```

当前脚本导入先批量加载命令，再按行校验参数。需要把设备上下文加入参数规则选择：

1. 已有 `LoadDevicesBySNs`，可拿到每行设备的 `product_class`。
2. 需要新增按产品/参数模型加载有效规则的方法，例如：

```text
LoadEffectiveCommandRules(ctx, commandIDs, productClasses)
```

3. `validateScriptLine` 中，先确定当前行设备，再用该设备对应的有效规则校验参数值。
4. 原 `standardValidationRules` 保留为兜底，不再作为唯一来源。

脚本导入是批量入口，必须和控制台直接执行保持同一套规则，否则会出现“页面不能下发但脚本能导入”或反过来的不一致。

### 7.5 后端：补 XML / DB 对 `defaultValue` 和 `validationPattern` 的支持

如果要完整承接整理出来的规则，需要扩展参数模型链路。

建议新增迁移：

```text
ALTER TABLE param_mappings ADD COLUMN default_value text;
ALTER TABLE param_mappings ADD COLUMN validation_pattern text;
ALTER TABLE discovered_param_mappings ADD COLUMN default_value text;
ALTER TABLE discovered_param_mappings ADD COLUMN validation_pattern text;
```

同步修改：

```text
omcgo/internal/config/parammodel/model.go
omcgo/internal/config/parammodel/loader.go
omcgo/internal/config/parammodel/pg_repository.go
omcgo/internal/config/parammodel/intersect.go
omcgo/internal/config/parammodel/handler.go
```

否则 XML 里即使写了这两个属性，也只是“文件里有”，不会进入运行时。

### 7.6 前端：展示和即时校验

修改位置：

```text
omcmb/frontend-core/src/types/mmlConsole.ts
omcmb/frontend-core/src/services/api/mmlApi.ts
omcmb/webcode/src/pages/mml/Console/types.ts
omcmb/webcode/src/pages/mml/Console/adapters.ts
omcmb/webcode/src/pages/mml/Console/components/ConfigParamsModal.tsx
```

建议补充：

1. `SubFieldDef` 增加：

```text
maxValue
enumValues
enumLabels
validationPattern
valueConstraint
ruleSource
```

2. `CommandParamPath` 增加同样字段。
3. `enumValues` 存在时，用下拉框，而不是普通 Input。
4. 数值型且有 min/max 时，用 `InputNumber`，设置 `min/max`。
5. 字符串型 min/max 按长度校验。
6. 正则存在时，输入后校验并给出错误提示。
7. 参数名下方或输入框旁边展示规则提示，例如：

```text
范围：0 ~ 359
可选值：0=OFF, 1=ON
格式：IPv4 地址
```

8. “执行”按钮在本地校验失败时禁用。

但前端校验只能改善体验，后端执行前校验仍必须补。

## 8. 分阶段落地建议

### 阶段 A：让页面看得到规则

目标：

- 当前设备 `BLQ` 的 `AZIMUTH` 在 MML 配置参数弹框显示 `范围：0 ~ 359`。

改动：

- MML sub-fields 接口合并 `param_mappings` 有效规则。
- DTO 返回 `max_value`、`enum_values`、`enum_labels`。
- 前端展示 `constraintText`。

验收：

- 选择 `1202000240194DP0015`。
- 选择 `MOD DEVICE_INFO`。
- 配置参数里 `AZIMUTH` 显示 `0 ~ 359`。

### 阶段 B：控制台执行前阻断非法值

目标：

- 控制台 MOD/ADD 下发前，后端阻断非法参数值。

验收：

- `AZIMUTH=-1`：后端返回 422，不创建任务。
- `AZIMUTH=0`：允许创建任务。
- `AZIMUTH=360`：后端返回 422。

### 阶段 C：脚本导入同规则

目标：

- TXT 脚本导入和控制台直接执行使用同一套有效规则。

验收：

- 脚本中 `MOD DEVICE_INFO:AZIMUTH=-1;1202000240194DP0015` 校验失败。
- 错误能指出设备、命令、参数和值。

### 阶段 D：补 `defaultValue` / `validationPattern`

目标：

- XML 中的默认值和正则规则进入 DB、接口、页面和后端校验。

验收：

- 选一个有 `validationPattern` 的参数，页面显示格式提示。
- 输入不匹配时前端阻断。
- 绕过前端直接调接口时后端仍阻断。

## 9. 回归测试建议

后端测试：

- `PgSubFieldRepository.ListEnrichedByCommand`：paramModel 规则覆盖 standard 规则。
- `ConsoleService.ExecuteStructured`：非法 min/max/enum/pattern 值返回 422。
- `ScriptImportValidator`：同一命令在不同设备参数模型下按各自规则校验。
- `parammodel.Loader`：XML 的 `defaultValue` / `validationPattern` 能入库。

前端测试：

- `ConfigParamsModal`：显示 constraintText。
- enum 参数渲染为 Select。
- 数值参数 min/max 阻断。
- 字符串长度和正则错误提示。
- 执行按钮在校验失败时禁用。

真实验证：

- 当前在线 BLQ 设备：

```text
SN: 1202000240194DP0015
Command: MOD DEVICE_INFO
Param: AZIMUTH
Rule: 0 <= value <= 359
```

测试值：

```text
-1   应阻断
0    应允许
359  应允许
360  应阻断
```

## 10. 风险点

1. 不能把所有机型规则简单写进 `standard_params`，否则会丢掉机型差异。
2. 多设备执行时，如果设备参数模型不同，页面展示一个规则可能误导用户；后端必须逐设备校验。
3. `param_mappings` 当前没有 `default_value` / `validation_pattern`，完整支持需要数据库迁移。
4. `discovered_param_mappings` 也要同步扩列，否则设备上传模型交集后的规则会比默认模型少。
5. 前端校验不能替代后端校验；MML 最终会下发到真实基站，后端必须作为最后一道拦截。

## 11. 建议的最终原则

MML 参数校验应统一成一条规则：

```text
按当前目标设备解析参数模型
  -> 从 param_mappings / discovered_param_mappings 取机型有效规则
  -> fallback 到 standard_params
  -> 前端展示和预校验
  -> 后端执行前强校验
  -> 再创建 MML 任务并下发
```

这样才能保证设备参数树、MML 控制台、MML 脚本导入三条入口对同一个参数给出一致结果。
