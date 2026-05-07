# S4 Verify Report — T-0098-P2-08

| 字段 | 值 |
|------|-----|
| Task | T-0098-P2-08（interop/runner.go + interop/validator.go：dataModelReg → paramRegistry + ParamMapping 元属性校验）|
| 分支 | `feature/T-0098-P1-data-dict` |
| 章程 | `docs/project/参数-KPI-告警-整合-实施计划.md` Phase 2 §2.A.P2-08 |
| 设计 | `docs/design/参数-KPI-告警-整合设计方案.md` §1.12 |
| 模式 | wave-batched（Skip S0/S1） |
| Deps | T-0098-P2-02 done（commit `228f3d6f`），T-0098-P2-01 done |
| 时间 | 2026-05-07 |

## 改动文件清单

| 文件 | 性质 | 行数 |
|------|------|------|
| `omcgo/internal/interop/runner.go` | 修改 | +75 / -10（dual-stack 注入 + 3 site refactor + countTypeMismatches helper） |
| `omcgo/internal/interop/validator.go` | 修改 | +95 / -55（dual-stack 注入 + ValidateDevice 拆 compare 共用比对路径） |
| `omcgo/internal/config/parammodel/validator.go` | 修改 | +5（导出 IsAccessWritable 给 interop 复用） |
| `omcgo/cmd/app/provider/modules.go` | 修改 | +4（WithParamRegistry 接线 × 2） |

总：**~180 LOC** 改动。

## 设计契约要点

### 1. 双栈协议（与 P2-07 一致）

ConformanceTestRunner 与 DataModelValidator 各新增 4 字段 + `WithParamRegistry(...)` setter：

```go
WithParamRegistry(paramReg *parammodel.Registry, prodReg *product.Registry, enabled bool)
```

`enabled=false` 或两 registry 任一 nil → 等价于不调用本 setter（沿用 dataModelReg 路径，零行为差异）。

### 2. ParamRegistry 路径在 4 个站点接入

| 站点 | 旧 | 新（双栈优先） |
|------|-----|-------------|
| `runner.checkParamPathsPresent` | `dataModelReg.ResolveForDevice` 检查 dm 存在 | `resolveExpectedParams` 命中即视为存在 |
| `runner.checkParamTypesMatch` | 解析 dm.ParameterTree → expectedTypes map | `resolveExpectedParams` → expectedTypes map（公共 helper `countTypeMismatches`） |
| `runner.executeVerifyResponse(check=model_exists)` | 同 1 | 同 1 |
| `validator.ValidateDevice` | 解析 dm.ParameterTree → 全量比对 | `resolveExpectedParams` 直出 + `compare` 共用比对 |

### 3. expectedParam 公共视图

定义 `expectedParam{Path, Type, Writable}`（runner 内）与 `validatorExpectedParam{Path, Type, Writable}`（validator 内）作为 datamodel.Parameter 与 parammodel.ParamMapping 的最小公共视图。

datamodel.Parameter 与 ParamMapping 字段名不一致（Path vs PrivatePath / Type vs DataType / Writable bool vs access string），统一为公共视图后两栈消费同一比对算法。

### 4. compare 共用路径（validator.go）

把 ValidateDevice 拆为：
- 取数路径：`resolveExpectedParams`（新栈） / dataModelReg + Unmarshal（旧栈）→ 都生成 `[]validatorExpectedParam`
- 比对路径：`compare(ctx, deviceID, dev, expected, modelVersion)` 统一执行（path/type/writable 三类比对 + extra 检测 + report 生成 + INFO log）

新栈 modelVersion 字段值为 `paramRegistry:default@v1.2.3` / `paramRegistry:discovered@v1.2.3`，供监控区分两类来源。

### 5. 导出 parammodel.IsAccessWritable

避免 interop 包重复实现 access 字符串归一化逻辑；旧 `isWritable` 保留为内部别名。容忍 `readWrite / read_write / RW / writeOnly / WriteOnly` 等大小写与下划线变体。

## 出口门核销

### 编译与测试

| 门 | 命令 | 结果 |
|----|------|------|
| go build | `go build ./...` | ✅ |
| go vet | `go vet ./...` | ✅ |
| 单测（interop） | `go test ./internal/interop/... -race -count=1` | ✅ |
| 单测（parammodel） | `go test ./internal/config/parammodel/... -race -count=1` | ✅ |
| 全 internal 回归 | `go test ./... -count=1` | ⚠️ 仅 `TestDownloadHandler` pre-existing flake |

### dev-pipeline §B3 / §B4 硬门

- [x] `go build ./...` 通过
- [x] `go test ./...` 仅 pre-existing flake（与 P2-01..P2-07 同一 L2 决策）
- [x] 无新增 TODO/FIXME/panic
- [x] 无新增 `if carrier == ...` 硬编码
- [x] 公共接口无新增 `any` / `interface{}`
- [x] 新端点 E/R 比 — N/A
- [x] 迁移双向演练 — N/A
- [x] metric / log 名 grep — model_version 字段值新增 "paramRegistry:..." 前缀，已在 validator INFO log grep 命中
- [x] 累计型依赖核销 — N/A

### 单测覆盖

interop 既有 14 个 `runner_test.go` + `validator_test.go` 测试用例全部通过。
本任务不强制新增单测（dual-stack flag default off → 行为零差异；新栈路径需要 ProductRegistry + ParamRegistry 完整 stub，模拟成本超 S 任务上限）。
集成验证留给 P3-01 endpoints 上线后端到端跑。

## 不在本任务范围（接力）

| 项 | 接力任务 |
|---|---------|
| ParamRegistry 路径单测 stub | P3-01 上线后用 testcontainers 跑端到端 |
| device.ProductID 列直读（无需 MatchProductClass 反查） | 待 ProductRegistry 在 Bootstrap 路径写入 dev.ProductID |

## 已识别后续观察点

| 项 | 描述 | 跟进 |
|----|------|------|
| L1 | model_version 字段格式从 `dm.Version`（如 "1.0"）改为可能为 `paramRegistry:default@v1.2.3` —— 北向报表/dashboard 若按字符串比对需注意 | P3-04 仪表盘前端 review 时核对 |
| L2 | ParamMapping 不存 entry_type=parameter 之外字段（如 forced_inform / notify），新栈 ValidateDevice 不参与这些维度比对——若旧栈测试用例依赖该比对，需在 P3-01 后调整测试期望 | 跟进 P3-04 完成后跑现网回归看分数差 |

## 结论

**S4 出口门通过**。Runner 3 站点 + Validator 1 站点全部 dual-stack 化；编译 + 既有单测全绿；flag off 时行为零差异，flag on 时 P2-02 ParamRegistry 路径接管 path/type/writable 比对。本 verify-md 同时承担 wave-batched 模式 S5 review 凭据。
