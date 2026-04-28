# T-0051 验证报告 — interop 模块 service facade

- **章程位**: Wave 2 / Block B.4
- **任务**: 新建 `internal/interop/service.go` 作为 thin service facade，统一封装既有 `runner` / `validator` / `report` 层
- **Worktree**: `agent-a978e30e`，分支 `worktree-agent-a978e30e`
- **执行时间**: 2026-04-28
- **结果**: PASS

---

## 1. 与既有代码的关系

interop 模块在本任务前已有较完整的多文件结构：

| 既有文件 | 角色 | 是否被 facade 调用 |
|---|---|---|
| `runner.go` | `ConformanceTestRunner` — 注册测试用例 / 执行 RunAll / RunByCategory | 是（通过 `CaseRunner` 接口） |
| `validator.go` | `DataModelValidator` — 比对设备实际参数与数据模型 | 是（通过 `ModelValidator` 接口） |
| `report.go` | `ValidationReport` 结构 + `ParamMismatch` | 是（作为 ValidateDevice 返回类型） |
| `model.go` | `TestCategory` / `TestCase` / `TestStep` / `TestResult` / `RunTestsRequest` | 是（输入输出复用） |
| `handler.go` | Gin REST handler — 现仍直接持有 runner+validator | 不改动；后续可改造为依赖 `InteropService`（章程整合阶段） |
| `cases/` | 测试用例数据集 | 不动 |

**facade 设计要点**（不重写、不复制既有逻辑）：

1. **新增 `InteropService` 接口**（5 个方法）：`ListCases` / `RunCases` / `RunByCategory` / `ValidateDevice` / 隐含的 summary 聚合。
2. **新增两个消费者侧小接口**：`CaseRunner`（runner 满足）、`ModelValidator`（validator 满足）。
   - 既不修改 `ConformanceTestRunner` / `DataModelValidator` 的源码，也使 facade 测试可以注入 mock。
3. **新增 `RunSummary` 聚合类型** + 私有 `summarize()` 纯函数：把 `[]TestResult` 折叠成 total/passed/failed，handler 现有的同款汇总以后可统一调用 facade，不再各自重算。
4. **错误处理**：facade 层对所有外部输入（device_sn 空、category 非法、carrier/tech 非法）做 fail-fast，并用 `fmt.Errorf("%w")` 包装下游错误。
5. **nil 防御**：`NewService(nil, nil, nil)` 不 panic，调用各方法返回明确错误。

---

## 2. 自跑验证结果

```
$ go build ./...
(no output, exit 0)

$ go test -race -count=1 ./internal/interop/...
ok  	github.com/omcgo/omcgo/internal/interop	1.658s
?   	github.com/omcgo/omcgo/internal/interop/cases	[no test files]
```

**新增测试用例**（`service_test.go`）：12 个，全部 PASS — 包含成功路径 + 失败路径。

| # | 用例 | 覆盖点 |
|---|---|---|
| 1 | `TestService_ListCases` | facade 透传 ListTestCases |
| 2 | `TestService_RunCases_AllCategories` | categories 为空 → RunAll，summary 计数 |
| 3 | `TestService_RunCases_SpecificCategories` | 多 category 串行调用 + 聚合 |
| 4 | `TestService_RunCases_InvalidCategoryShortCircuits` | 非法 category 立即返错，不调 runner |
| 5 | `TestService_RunCases_EmptyDeviceSN` | 空 SN 校验 |
| 6 | `TestService_RunCases_RunnerError` | runner 错误用 `%w` 包装上抛 |
| 7 | `TestService_RunByCategory` | 单分类成功路径 |
| 8 | `TestService_RunByCategory_InvalidCategory` | 单分类非法 |
| 9 | `TestService_ValidateDevice_Success` | validator facade 透传 |
| 10 | `TestService_ValidateDevice_InvalidCarrier` | 非法 carrier 不调 validator |
| 11 | `TestService_ValidateDevice_ValidatorError` | validator 错误透传 |
| 12 | `TestService_NilDependenciesGuarded` | nil 依赖不 panic + 明确错误 |
| (helper) | `TestSummarize_PureFunction` | 聚合纯函数独立可测 |

```
$ ls internal/interop/*service*.go
internal/interop/service.go
internal/interop/service_test.go
```

✅ 章程 W2.B.4 Pass 标准（`*service*.go ≥ 1`）满足，实际产出 2 个文件。

---

## 3. 不动的边界

按 sub-agent 路径互斥约束：

- ❌ 未碰 `cmd/app/provider/modules.go` 或 `cmd/app/provider/interop.go`（主会话整合时改）
- ❌ 未改其他模块（task / events / core / mr / syslog / provision）
- ❌ 未改既有 `runner.go` / `validator.go` / `handler.go` / `model.go` / `report.go` 的任何一行
- ❌ 未新增 go.mod 依赖
- ❌ 未 commit / push / pull

---

## 4. 后续整合建议（留给主会话）

1. `Handler` 当前仍持有 `*ConformanceTestRunner` + `*DataModelValidator`。后续可在 `cmd/app/provider/interop.go` 把 `Handler` 的依赖换成 `InteropService`，把汇总 / 入参校验下沉到 facade，handler 仅做 HTTP 翻译。
2. 如需在其他位置（如定时任务、CLI、gRPC）复用 interop 能力，统一通过 `InteropService` 注入，避免再次绑定到具体 runner/validator。
3. 当 `report` 子能力扩展（例如持久化历史报告）时，可在 facade 上加 `GetReport` / `ListReports`，handler 与新增入口都不需要再改 runner/validator。
