# MML 临区增删与校验问题修复记录

> 日期：2026-08-21
> 范围：MML 控制台、参数模型加载、MML seed 修复、前端写参数校验
> 入口：`/mml/console`

## 背景

现场在 MML 控制台执行临区相关命令时发现三类问题：

1. 添加 4G 临区时报错，ADD 编译阶段提示 compound path 的 `{i}` 数量不匹配。
2. 2G/GSM 临区参数管理只有查询、修改，没有新增、删除。
3. 添加临区时，`QRxLevMinSIB5 = -22` 被前端校验为“长度不能超过 -22 个字符”。

5G 临区可以正常添加，原因不是代码路径完全不同，而是 5G 的参数模型支持集包含了正确的多实例对象路径；GSM 和部分 4G/空闲态异频载波数据存在模型/seed 缺口。

## 问题 1：4G 临区 ADD 编译报错

### 现象

执行 `ADD LTE_CELL` 时失败：

```text
ADD compound path .{i}. count=1, expected selectors+1=2
path=Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}
```

### 根因

后端在处理 ADD compound 对象路径时，只按 `.{i}.` 统计实例占位符。

4G 临区 ADD 的目标对象路径形态为：

```text
Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}
```

最后一个占位符是终止在路径末尾的 `.{i}`，不是 `.{i}.`，旧逻辑漏数，导致：

```text
实际应识别 2 个实例占位符
旧逻辑只识别 1 个
```

### 修复

文件：

- `omcgo/internal/mml/console_executor.go`
- `omcgo/internal/mml/console_executor_test.go`

修复点：

- 占位符统计改为覆盖 `.{i}`，兼容中间对象和末尾对象。
- 替换逻辑保留路径尾部是否带点的语义。
- 增加 terminal object placeholder 回归测试。

## 问题 2：2G/GSM 临区没有 ADD/RMV

### 现象

MML 选择命令弹窗中，“邻区参数管理”下 GSM 只显示：

```text
MOD 修改 GSM邻区参数管理
LST 查询 GSM邻区参数管理
```

没有：

```text
ADD 添加 GSM异系统邻区
RMV 删除 GSM异系统邻区
```

### 根因

MML 命令树会根据所选设备的参数模型支持集过滤 ADD/RMV。过滤条件要求目标集合对象支持对应实例对象：

```text
targetObject = ...InterRATCell.GSM.
需要支持集包含 ...InterRATCell.GSM.{i}.
```

本地库排查结果：

```text
存在：Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.
存在：Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{i}.BCCHARFCN 等叶子参数
缺失：Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{i}.
```

因此过滤器判断“该对象集合不可增删”，ADD/RMV 被隐藏。

5G/LTE 可正常显示，是因为它们的支持集里已有：

```text
Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.5GCell.{i}.
Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.
```

### 修复

文件：

- `omcgo/internal/config/parammodel/loader.go`
- `omcgo/internal/config/parammodel/model_test.go`
- `omcgo/data/param-mappings/standard-model.xml`
- `omcgo/migrations/seed/000001_init_seed.sql`
- `omcgo/internal/mml/issue_274_seed_test.go`

修复点：

- 标准模型中补齐 GSM 实例对象：

```text
Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{i}.
```

- 将 GSM 集合对象访问性调整为 `READ_WRITE`：

```text
Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.
```

- 参数模型加载器增加规范化逻辑：

```text
如果模型中存在 GSM.{i}.xxx 叶子参数，但缺少 GSM.{i}. 对象，
则自动派生并插入 GSM.{i}. object 映射。
```

- seed 增加幂等修复：

```text
现有库执行 migrate-seed 后，也会补齐 standard_params / param_mappings。
```

这样既修复现有数据库，也避免后续重新加载产品 XML 后回退。

## 问题 3：`-22` 被当成字符串长度校验

### 现象

添加 LTE 邻区/异频载波参数时，字段：

```text
QRxLevMinSIB5
异频邻区最小接收电平
```

输入：

```text
-22
```

前端报错：

```text
长度不能超过 -22 个字符
```

### 根因

部分产品参数模型 XML 中，`QRxLevMinSIB5` 被标为 `STRING`，但 min/max 是数值范围：

```text
type=STRING
min=-70
max=-22
```

前端校验规则是：

```text
STRING  → min/max 表示字符串长度
INT     → min/max 表示数值范围
```

于是 `max=-22` 被错误解释成“最大长度 -22”。

标准模型和业务规则实际应为：

```text
type=INT
range=-70..-22
```

### 修复

文件：

- `omcgo/internal/config/parammodel/loader.go`
- `omcgo/migrations/seed/000001_init_seed.sql`
- `omcmb/webcode/src/pages/mml/Console/modParamValidation.ts`
- `omcmb/webcode/src/pages/mml/Console/components/ConfigParamsModal.tsx`
- `omcmb/webcode/src/pages/mml/Console/modParamValidation.test.ts`
- `omcmb/webcode/src/pages/mml/Console/components/ConfigParamsModal.test.tsx`

后端修复：

- 参数模型加载时，将 LTE InterFreq Carrier 下的 `QRxLevMinSIB5` 规范化为 `INT`。
- seed 幂等修正现有库里所有 `QRxLevMinSIB5` 映射为 `INT`。

前端兜底：

- 新增 `modParamRangeKind()`：

```text
如果 valueType=STRING 但 min/max 出现负数，则按数值范围处理。
```

- 配置弹窗默认值初始化也复用该判断，避免负数数值字段不填默认最小值。

前端兜底的目的不是替代数据修复，而是避免旧数据、缓存数据或未重载模型继续触发离谱的“负数字符串长度”提示。

## 涉及文件汇总

后端：

```text
omcgo/internal/mml/console_executor.go
omcgo/internal/mml/console_executor_test.go
omcgo/internal/config/parammodel/loader.go
omcgo/internal/config/parammodel/model_test.go
omcgo/internal/mml/issue_274_seed_test.go
omcgo/data/param-mappings/standard-model.xml
omcgo/migrations/seed/000001_init_seed.sql
```

前端：

```text
omcmb/webcode/src/pages/mml/Console/modParamValidation.ts
omcmb/webcode/src/pages/mml/Console/modParamValidation.test.ts
omcmb/webcode/src/pages/mml/Console/components/ConfigParamsModal.tsx
omcmb/webcode/src/pages/mml/Console/components/ConfigParamsModal.test.tsx
```

## 验证记录

### 自动化测试

后端：

```bash
go test ./internal/config/parammodel ./internal/mml -count=1
```

结果：

```text
ok github.com/omcgo/omcgo/internal/config/parammodel
ok github.com/omcgo/omcgo/internal/mml
```

前端：

```bash
npm test -- src/pages/mml/Console/modParamValidation.test.ts src/pages/mml/Console/components/ConfigParamsModal.test.tsx
npm run typecheck
npm run build
```

结果：

```text
58 tests passed
typecheck passed
vite build passed
```

### 部署验证

执行：

```bash
OMC_PROJECT=goomc-local bash deployments/docker/dc.sh up -d --build app web
```

结果：

```text
goomc-local-app-1  Up
goomc-local-web-1  Up
goomc-local-acs-1  Up
migrate-schema     Exited (0)
migrate-seed       Exited (0)
migrate-tsdb       Exited (0)
```

HTTP 验证：

```bash
curl -I --max-time 10 http://localhost:8081/
```

结果：

```text
HTTP/1.1 200 OK
Last-Modified: Fri, 21 Aug 2026 09:57:47 GMT
```

### 数据验证

GSM 临区对象映射：

```text
Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.      object READ_WRITE
Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.InterRATCell.GSM.{i}.  object READ_WRITE
```

`QRxLevMinSIB5` 映射：

```text
data_type = INT
min_value = -70
max_value = -22
```

## 横向排查：其他 MML 命令与参数

### 排查口径

本次按最终入库数据横向检查，而不是只看 catalog JSON：

```text
mml_commands
mml_command_sub_fields
standard_params
param_mappings
```

当前本地库有效 MML 命令规模：

```text
ADD 42
LST 164
MOD 157
RMV 42
```

### 结论 1：负数范围类型问题不止 QRxLevMinSIB5

`standard_params` 中已经没有 `STRING + negative min/max`：

```text
0 rows
```

但 `param_mappings` 中仍有 18 行同类脏数据，涉及 8 个标准路径。其中会被 MML MOD 弹窗唤起的路径如下：

| MML 命令 | 标准路径 | 产品模型中的错误范围 |
|---|---|---|
| `MOD MML350_DEVICE_FAP__NL` | `Device.FAP.NL.{i}.cellID` | `STRING -1..503` |
| `MOD MML350_DEVICE_FAP__NL` | `Device.FAP.NL.{i}.freqUncerThreshlod` | `STRING -32768..32767` |
| `MOD MML350_DEVICE_FAP__NL` | `Device.FAP.NL.{i}.phaseOffset` | `STRING -32768..32767` |
| `MOD SJ_SUB_10` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QRxLevMinSIB1` | `STRING -70..-22` |
| `MOD SJ_SUB_10` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QRxLevMinSIB3` | `STRING -70..-22` |
| `MOD SH_SUB_01` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.P0NominalPUCCH` | `STRING -127..-96` |
| `MOD SH_SUB_01` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.P0NominalPUSCH` | `STRING -126..24` |
| `MOD FAP_SERVICE` | `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ReferenceSignalPower` | `STRING -60..50` |

当前前端 MML 弹窗主要从 `standard_params` 取 `data_type/min/max`，这些路径在 `standard_params` 中没有负数范围，因此当前页面不一定复现“长度不能超过负数”的报错。

但这些路径在产品模型 `param_mappings` 层仍是同类问题。如果后续接口按设备产品模型富化约束、脚本导入校验改走 `param_mappings`、或旧缓存返回产品模型元数据，就会重新触发。建议将 loader 的规范化从单点 `QRxLevMinSIB5` 扩展为规则化修正：`STRING + negative min/max` 且路径/业务语义为数值字段时强制转为 `INT`，并用 allowlist 保留真正按长度校验的字符串字段。

### 结论 2：ADD 参数层级混入仍存在

全量检查 ADD 命令的 active sub-field 后，发现两个命令仍混有与 `target_object` `{i}` 层级不一致的字段：

```text
ADD LTE_CELL     target_object={i}，期望字段 {i} 数量=2，实际有 7 个字段为 {i} 数量=3
ADD NR_LTE_CELL  target_object={i}{i}，期望字段 {i} 数量=3，实际有 17 个字段为 {i} 数量=2
```

典型错配：

```text
ADD LTE_CELL
target_object = Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.
混入字段     = Device.Services.FAPService.{i}.CellConfig.{i}.LTE.RAN.NeighborList.LTECell.{i}.CID

ADD NR_LTE_CELL
target_object = Device.Services.FAPService.{i}.CellConfig.{i}.LTE.RAN.NeighborList.LTECell.
混入字段     = Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.CID
```

这不是“末尾 `.{i}` 漏计数”的同一根因，而是 ADD sub-field 绑定混入了另一套对象层级。它会导致 ADD 编译时 selector/new-instance 替换语义不稳定，应单独清理：

- `ADD LTE_CELL` 只保留 `Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.*`
- `ADD NR_LTE_CELL` 只保留 `Device.Services.FAPService.{i}.CellConfig.{i}.LTE.RAN.NeighborList.LTECell.{i}.*`

### 结论 3：对象实例缺失不只 GSM，但影响面不同

按 `ADD target_object + {i}.` 反查 `param_mappings` 的 object 行，GSM 已补齐并可被支持集识别。仍有部分 ADD 命令在当前全部 active 参数模型中找不到实例 object 行，例如：

```text
ADD INTER_RAT_CELL_NR
ADD INTER_RAT_CELL_UMTS
ADD LTE_S1U
ADD PDCP_INIT_PARAM
ADD MML350_DEVICE_KEEPALIVEDMGMT__VRRPMGMT_VIRTUALIPLIST
ADD MML350_DEVICE_SERVICES__GSMBTSCELLDT
ADD SF_NR_SIB_PARAMS
ADD SJ_CONN_EUTRA_CARRIER
ADD X2_IP_ADDR_MAP_INFO
```

这些命令未必都会在当前设备上展示，因为命令树会按产品支持集过滤；但如果业务期望它们可新增/删除，就需要像 GSM 一样补齐对应实例 object 映射，不能只依赖叶子参数存在。

## 注意事项

1. 5G 能正常添加并不代表 ADD/RMV 后端逻辑对所有制式都天然完整；它只是刚好有正确的 `{i}.` 实例对象支持集。
2. GSM ADD/RMV 展示依赖 `param_mappings` 中的对象实例映射，不只依赖 `mml_commands` 行存在。
3. 参数模型 XML 重新加载会重写 builtin `param_mappings`，因此必须在 loader 层修复，不能只靠一次性 SQL。
4. 前端负数范围兜底只处理明显不合理的 `STRING + negative min/max`，不会改变普通字符串长度校验，例如 PLMNID `5..6` 仍按长度处理。
5. app 日志中仍可见既有的 northbound stream、tracing exporter、slow-query 等警告，本次未涉及这些问题。
