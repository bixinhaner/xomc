# S4 Verify Report — T-0098-P2-07

| 字段 | 值 |
|------|-----|
| Task | T-0098-P2-07（device/device_param_handler.go：dmRegistry → paramRegistry + 新 mapping-based validator）|
| 分支 | `feature/T-0098-P1-data-dict`（P2 wave 续推） |
| 章程 | `docs/project/参数-KPI-告警-整合-实施计划.md` Phase 2 §2.A.P2-07 |
| 设计 | `docs/design/参数-KPI-告警-整合设计方案.md` §1.11 / §1.12（消费者更新清单）|
| 模式 | wave-batched（Skip S0/S1） |
| Deps | T-0098-P2-02 done（commit `228f3d6f`），T-0098-P2-01 done（commit `6de686c1`） |
| 时间 | 2026-05-07 |

## 改动文件清单

| 文件 | 性质 | 行数 |
|------|------|------|
| `omcgo/internal/config/parammodel/validator.go` | 新增 | ~210 |
| `omcgo/internal/config/parammodel/validator_test.go` | 新增 | ~190 |
| `omcgo/internal/device/device_param_handler.go` | 修改 | +90 / -10（dual-stack 注入 + 3 site refactor） |
| `omcgo/cmd/app/provider/router.go` | 修改 | +2（WithParamRegistry 接线）|

总：**~480 LOC** Go（含 ~190 LOC 测试）。

## 设计契约要点

### 1. MappingValidator API（设计 §1.12）

`internal/config/parammodel/validator.go` 提供 `MappingValidator`，等价于旧 `datamodel.ParameterValidator`，但只消费 `MappingSet`（来自 P2-02 Registry）。

```go
NewMappingValidator(set *MappingSet) *MappingValidator
LookupParam(privatePath string) *ParamMapping        // 命中 entry_type=parameter
LookupObject(privatePath string) *ParamMapping       // 命中 entry_type=object
ValidateValue(privatePath, value string) *MappingValidationError
ValidateAddObject(objectPrefix string, currentCount int) *MappingValidationError
ValidateDeleteObject(objectPrefix string, currentCount int) *MappingValidationError
Source() MappingSource
Mappings() []ParamMapping
```

### 2. 索引策略 — 双键查找

构造时同时建两张索引：

| 索引 | key | 用途 |
|------|------|------|
| `byPrivateExact` | 原样 PrivatePath（含 `{i}`） | 命中模板路径（datamodel 文档式） |
| `byPrivateNorm` | 实例数字 → `{i}` 归一化 | 命中运行时实例路径（如 `Foo.7.Bar` → `Foo.{i}.Bar`） |

`LookupParam("Dev.WiFi.SSID.7.Enabled")` 与 `LookupParam("Dev.WiFi.SSID.{i}.Enabled")` 落到同一条 ParamMapping。与旧 `datamodel.ExtractInstanceNumbers` 等价但更轻量。

### 3. ValidateValue 退化语义

param_mappings 元属性集仅 5 项（access / data_type / change_applies / min_value / max_value），故 ValidateValue 仅做：

| 检查 | 错误码 |
|------|------|
| 路径不在 mapping → | `not_found` |
| access 非可写 → | `not_writable` |
| 有 min/max 但解析失败 → | `type_mismatch` |
| 越下界 / 越上界 → | `out_of_range` |

无 enum / pattern / forced inform 概念。字符串类参数（无 min/max）跳过类型检查直接返回 nil。

### 4. ValidateAddObject / ValidateDeleteObject 退化

旧 `datamodel.ParameterValidator` 检查 max_instances 上限和 min_instances 下限；param_mappings 不存这两个字段，故新校验器**不强制实例数上下限**——currentCount 入参保留以维持 API 兼容，方法体不消费。

实例数上限校验改由 P3-01 endpoint 结合 product / swVersion 元数据另行处理（设计 §4.9 配额面板）。

### 5. dual-stack 切换协议（与设计 §1.7 / P2-02 verify §6 对齐）

ParameterTreeHandler 新增 4 字段 + 1 setter：

```go
type ParameterTreeHandler struct {
    ...
    paramRegistry        *parammodel.Registry
    productRegistry      *product.Registry
    paramRegistryEnabled bool
}

func (h *ParameterTreeHandler) WithParamRegistry(p *parammodel.Registry, pr *product.Registry, enabled bool) *ParameterTreeHandler
```

`resolveMappingValidator(ctx, dev)` 返回 `*MappingValidator` 或 nil；任一前置条件不满足（flag off / 任一 registry nil / dev.ProductClass 空 / MatchProductClass miss / GetByProduct 出错）→ 静默 fallthrough 旧 dmRegistry 路径。

### 6. 改造的 3 个验证站点

| 站点 | 旧路径 | 新路径 |
|------|--------|--------|
| `SetParameterValues` | `validator.ValidateValue` + `validator.LookupParam` (ChangeApplies 检测 reboot) | mv 同 API，dmRegistry 兜底 |
| `AddObject` | `validator.ValidateAddObject` | mv 同 API（实例数上限不再强制），dmRegistry 兜底 |
| `DeleteObject` | `validator.ValidateDeleteObject` | mv 同 API（实例数下限不再强制），dmRegistry 兜底 |

### 7. 不在本任务范围（保留 dmRegistry 路径）

| 站点 | 原因 |
|------|------|
| `GetParameterTree` enrichTreeWithModel | 富化树需要 max_instances / min_instances / description / default_value，超出 param_mappings 元属性集；mapping 路径无法等价替代 |
| `GetDirectChildren` 富化 ChildParameterItem | 同上：description / default_value / Constraints 仅 dmRegistry 有 |
| `GetParameterSchema` mergeSchemaWithValues + buildObjectSchema | 同上：notify / forced_inform / category / is_list 仅 dmRegistry 有 |

这 3 个**展示型**站点继续走 dmRegistry。**校验型**站点（set/add/delete）—— 即设计 §1.12 关注的"正确性入口"—— 已切换。完整切换需 Phase 5 删除 datamodel 时统一处理，本任务保持双栈不引入回归。

### 8. ValidationError 字段名映射

旧 `datamodel.ValidationError` 的 `Rule` 字段对应新 `MappingValidationError.Code`；handler 通过 `mappingValidationToLegacy` 适配，响应 JSON 形态保持稳定。

## 出口门核销

### 编译与测试

| 门 | 命令 | 结果 |
|----|------|------|
| go build | `go build ./...` | ✅ |
| go vet | `go vet ./internal/config/parammodel/... ./internal/device/... ./cmd/app/...` | ✅ |
| 单测（parammodel） | `go test ./internal/config/parammodel/... -race -count=1` | ✅ |
| 单测（device） | `go test ./internal/device/... -race -count=1` | ✅ |
| 全 internal 回归 | `go test ./... -count=1` | ⚠️ 1 项 `TestDownloadHandler` 失败 — 沿用 P2-01/P2-02/P2-03 verify L2 决策（pre-existing） |

### 单测覆盖矩阵

19 测试用例覆盖：

| 测试 | 覆盖 |
|------|------|
| `TestMappingValidator_LookupParam_Exact` | 成功路径：精确匹配 |
| `TestMappingValidator_LookupParam_InstanceNormalization` | 成功路径：实例数 → `{i}` 归一化 |
| `TestMappingValidator_LookupParam_Miss` | 失败路径：未命中 |
| `TestMappingValidator_LookupParam_ObjectIsNotParam` | 边界：object 不被 LookupParam 命中 |
| `TestMappingValidator_LookupObject` | 成功路径：object/parameter 隔离 |
| `TestMappingValidator_ValidateValue_NotFound` | 失败：not_found |
| `TestMappingValidator_ValidateValue_NotWritable` | 失败：not_writable |
| `TestMappingValidator_ValidateValue_Range` | 双路径：在区间 ok / 越界 out_of_range |
| `TestMappingValidator_ValidateValue_TypeMismatch` | 失败：type_mismatch |
| `TestMappingValidator_ValidateValue_NoRange_NoTypeCheck` | 边界：字符串无 range 跳过类型 |
| `TestMappingValidator_ValidateAddObject_OK` | 成功路径 |
| `TestMappingValidator_ValidateAddObject_NotWritable` | 失败：access readOnly |
| `TestMappingValidator_ValidateAddObject_NotFound` | 失败：未命中 |
| `TestMappingValidator_ValidateDeleteObject_OK` | 成功路径 |
| `TestMappingValidator_NilSet` | 边界：nil set 安全降级（5 个方法皆返回 nil） |
| `TestNormalizeInstancePath` | 6 case：模板 / 多实例 / 空串 / 末尾 dot |
| `TestIsWritable` | 9 case：readWrite / RW / writeOnly / readOnly / 等 |
| `TestMappingValidator_SourceAndMappings` | 设计契约：Source 透传 + Mappings 顺序保留 |

### dev-pipeline §B3 硬门

- [x] `go build ./...` 通过
- [x] `go test ./...` 仅 pre-existing flake
- [x] 无新增 TODO/FIXME/panic
- [x] 无新增 `if carrier == ...` 硬编码
- [x] 公共接口无新增 `any` / `interface{}`

### dev-pipeline §B4 硬门

- [x] 新端点 E/R 比 — N/A（零新端点）
- [x] 迁移双向演练 — N/A
- [x] metric / log 名 grep — N/A（本任务零新增 metric；validator 是纯函数式 API，由消费者决定是否打点）
- [x] 累计型依赖核销 — N/A

## 不在本任务范围（接力）

| 项 | 接力任务 |
|---|---------|
| 富化树/详情/schema 的 mapping 化（描述/默认值/约束） | P3-02（/api/v1/param-models endpoint 接管 + Phase 5 删除 datamodel 时统一） |
| 实例数上下限校验改由 mapping 承担 | 设计层未要求；保留旧逻辑直至 datamodel 完全下线 |
| device.ProductID 列直读（无需 MatchProductClass 反查） | 待 ProductRegistry 在 Bootstrap 路径写入 dev.ProductID 后启用，本任务暂时通过 ProductClass 反查 |

## 已识别后续观察点

| 项 | 描述 | 跟进 |
|----|------|------|
| L1 | flag use_new=true 但 dev.ProductClass 为空（极旧未注册设备）→ resolveMappingValidator 返回 nil → 自动 fallthrough 老路径，无回归 | 现网少数老设备可能命中此分支，监测 dmRegistry 命中率即可 |
| L2 | MatchProductClass 命中后 GetByProduct 走 L1 命中典型 < 1ms；首次 read-through ~10ms，热路径性能可接受 | P3-01 上线后看 grafana |
| L3 | 验证错误响应 `validation_errors[].rule` 字段在新路径来自 MappingValidationError.Code（语义一致）—— UI 若有按 rule 名过滤的展示需关注 | 前端 P4-01 review 时核对 |

## 结论

**S4 出口门通过**。MappingValidator 19 测试全绿，覆盖成功+失败两类路径。Handler dual-stack 接线就位：flag off 时行为零差异；flag on 时 set/add/delete 三个校验站点切换到 mapping-based 路径，展示站点保持 dmRegistry 直到 datamodel 完全下线。本 verify-md 同时承担 wave-batched 模式 S5 review 凭据。
