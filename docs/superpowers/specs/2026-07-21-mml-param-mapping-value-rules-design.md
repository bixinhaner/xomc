# MML 参数模型取值规则与控制台校验设计

日期：2026-07-21

## 1. 背景

`omcgo/data/param-mappings/*.xml` 保存不同参数模型下的参数映射规则。当前 XML 已包含以下信息：

- `min` / `max`
- `defaultValue`
- `validationPattern`
- `enumValues` / `enumLabels`

目前 `min_value` / `max_value` 已进入 `param_mappings`，但 MML 控制台 sub-field 查询主要读取全局 `standard_params`；`defaultValue` 和 `validationPattern` 尚未进入参数模型数据结构和控制台接口。因此 XML 中已更新的模型级规则不能完整驱动 MML 修改参数页面。

本次目标是重新梳理当前 XML 数据，刷新已有数据库数据，并让 MML 控制台在修改参数时使用当前产品参数模型的范围、默认值、正则和枚举规则。

## 2. 数据现状基线

当前 `param-mappings` 目录包含 14 个 XML 文件。只读统计结果：

| 属性 | 非空数量 |
| --- | ---: |
| `defaultValue` | 444 |
| `validationPattern` | 253 |
| `enumValues` | 655 |
| `enumLabels` | 647 |
| `min` | 1576 |
| `max` | 1733 |

同一 XML 内仅 BM.xml 存在 1 个重复 `standardPath`，两行规则一致；当前未发现同一 XML 内规则冲突。

## 3. 设计决策

### 3.1 规则归属

模型差异化规则保存在 `param_mappings`，不写入全局 `standard_params`：

```text
mml_command_sub_fields.standard_path_id
        -> standard_params.id
        -> standard_params.standard_path
        -> param_mappings.standard_path + param_model_id
```

`standard_params` 继续作为标准参数树和公共元数据来源；`param_mappings` 作为当前参数模型的覆盖规则来源。

### 3.2 规则优先级

对于带产品或设备上下文的 MML 控制台请求：

1. 当前 `param_model_id` 对应的 active `param_mappings` 优先；
2. `min_value` / `max_value` 为空时回退 `standard_params` 的同名范围；
3. `default_value`、`validation_pattern`、枚举只使用参数模型数据，没有模型规则时为空；
4. 未传产品模型的 admin 视图仍返回标准参数树公共信息，不伪造产品规则。

查询按模型和标准路径取一条确定性映射，保留 custom 覆盖优先级，并按 private path 稳定排序，避免一个标准路径的多个私有别名造成 sub-field 重复。

### 3.3 min/max 的重新梳理

`min_value` / `max_value` 不新增数据库列，因为两列已经存在。本次明确纳入数据刷新和控制台规则优先级：

- XML Loader 每次重载重新解析 `min` / `max`；
- 删除并重建该模型的 builtin 映射，更新已有 `param_mappings`；
- 对已有 `discovered_param_mappings` 同步默认范围；
- 如果产品配置了 `device_attrs_override.min_value/max_value=true`，继续保留设备发现值；否则使用最新默认映射值；
- MML 控制台校验使用参数模型范围，标准树范围作为缺省回退。

### 3.4 枚举规范化

XML 枚举以 CSV 保存。Loader 在写入前统一去除外围空格，并把全角逗号规范成半角逗号。当前发现 `Device.IPsec.MyKeyMode` 的枚举使用全角逗号，需要规范为多个选项。

后端向控制台返回结构化 `enum_options`，每个选项包含 value 和 label；没有 label 时使用 value。

BOOLEAN 仍由前端使用 `true` / `false` 语义选择框，兼容 XML 中常见的 `1,0` 枚举编码，不把设备内部编码直接展示给用户。

## 4. 后端实现范围

### 4.1 数据库迁移

新增主库 schema migration `000003_mml_param_mapping_value_rules.sql`：

- `param_mappings.default_value text`
- `param_mappings.validation_pattern text`
- `discovered_param_mappings.default_value text`
- `discovered_param_mappings.validation_pattern text`

使用 `ADD COLUMN IF NOT EXISTS`，Down 删除本迁移新增列。

### 4.2 XML 与领域模型

扩展 `xmlParamEntry` 和 `ParamMapping`：

- 解析并持久化 `defaultValue`、`validationPattern`；
- 保持现有 `min` / `max`、枚举、镜像和 supported 语义；
- 统一枚举分隔符；
- 对枚举值和标签数量进行校验，标签缺失时回退 value；
- 规则内容保持文本，不在 Loader 阶段把正则改写成另一种表达式。

同步更新默认映射 repository、discovered repository、Redis cache payload、交集写入和 Loader 的 discovered 对账逻辑。

### 4.3 数据刷新

沿用现有 param-model Loader 的 builtin/custom 语义：

- builtin 行按 XML 全量重建；
- custom 行保留，冲突时 custom 胜出；
- 重新统计 `param_models.total_*`；
- discovered 行更新最新默认规则，保留 min/max 的设备覆盖；
- 完成后刷新 ParamRegistry 缓存版本。

不把 XML 内容硬编码进 migration；既有环境通过 migration 建列后，由 Loader reload 使用当前仓库 XML 完成数据更新。

## 5. MML 接口与控制台实现

### 5.1 API

扩展 `GET /api/v1/mml/commands/:id/sub-fields` 响应：

- `default_value`
- `validation_pattern`
- `enum_options`
- `min_value` / `max_value` 使用参数模型优先、标准树回退后的结果

产品 class 或 device key 解析出的 `param_model_id` 必须传入 enriched 查询，保证命令树支持集合和参数规则使用同一参数模型。

### 5.2 前端输入控件

- 有枚举选项的参数使用 `Select`；
- BOOLEAN 使用 true/false 选择框；
- 其他参数使用 `Input`；
- 默认值优先使用 `defaultValue`；
- 无默认值的枚举不擅自选第一项，提交时按必填规则提示；
- 不在进入页面时额外展示范围、正则或默认值说明，避免占用一行；
- 规则只在提交校验失败后显示在对应输入框下方。

### 5.3 提交校验

校验顺序：

1. 必填；
2. 枚举值；
3. 数据类型和整数格式；
4. min/max 数值范围或字符串长度；
5. `validationPattern` 正则匹配。

正则兼容 XML 常见的 `/^...$/` 包装格式；无效正则不导致页面崩溃，而是按规则不可用记录并由测试覆盖。

### 5.4 参数控件宽度与箭头布局

普通输入框、枚举下拉框和 BOOLEAN 下拉框共用同一参数控件列，控件及其内部选择区域均占满可用宽度。Select 右侧预留固定尾部空间，并将下拉箭头固定在控件最右侧；枚举文本过长时截断，不允许挤压箭头或改变控件宽度。该调整只影响展示样式，不改变参数值、默认值、接口字段和提交校验。

## 6. 测试与验收

### 后端

- XML 新属性解析和全角逗号枚举规范化；
- Loader 写入 default/pattern/min/max/enum；
- builtin 重载更新旧值且保留 custom；
- discovered 更新规则并保留范围覆盖；
- enriched 查询按 param model 返回模型规则并正确回退标准树范围；
- MML sub-field API 字段映射。

### 前端

- enum option 映射和 Select 渲染；
- BOOLEAN true/false 兼容；
- defaultValue 初始化；
- min/max、枚举、正则提交校验；
- 校验错误只在提交后显示。

### 命令

- `cd omcgo && go build ./...`
- `cd omcgo && go test ./...`
- `cd omcmb && npm run typecheck`

### 数据验收

迁移和 Loader reload 后，按 XML 对账检查至少 BLQ、BSC、MLQ、ENB_DEFAULT_098、ENB_DEFAULT_181：

- XML 与 `param_mappings` 的 min/max/default/pattern/enum 数量一致；
- 典型 path 的规则值一致；
- 旧 builtin 规则不残留；
- custom 行未被删除；
- MML 控制台 API 返回模型级规则，页面实际显示 Select 并在提交时校验。
