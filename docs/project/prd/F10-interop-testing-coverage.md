# PRD: F10 互操作用例库扩充（GA 级质量补强）

> **关联**：Backlog T-0030 / Risk R-202 / Sprint-10 / Milestone 2026Q2-to-RC
> **作者**：Claude（代 Owner = 测试专家 + 电信业务专家）
> **创建**：2026-05-11
> **状态**：草案 → S1 规划

---

## 1. 业务背景

F10 互操作（Interoperability Testing）是 OMC 在出厂联调阶段对设备完成"接入合规性 + 数据模型一致性 + RPC 行为正确性"批量验证的能力。OMC 作为 ACS 必须在交付前回答两个问题：

1. 任一型号 CPE 接入 OMC 后，必备 RPC 是否全可达？
2. CPE 上报的参数树是否符合厂商声明的数据模型（型号 × 运营商 × 制式 三维契约）？

### 1.1 现状（2026-05-11 实测）

| 维度 | 现状 | 缺口 |
|------|------|------|
| 后端代码 | `internal/interop/` ~1850 LOC（handler / service / runner / validator / cases）✅ | 架构齐整，仅缺用例 |
| API 端点 | 4 个：GET `/test-cases`、POST `/run`、POST `/run/:category`、POST `/validate/:deviceId` ✅ | 端点足够 |
| 用例库 | **14 条**：DataModel 2 / Protocol 3 / RPC 9 | **覆盖稀疏，无失败路径，无运营商参数化** |
| 单元测试 | 29 Test 函数（service 13 + runner 10 + validator 6）✅ | 但用例增长后需同步 |
| 前端页面 | `pages/config/InteropTesting/index.tsx`（list + run + validate）✅ | 现状已足够 |
| E2E 断言 | 3 条 claim（iop-1/2/3）✅ | 缺失败路径 claim、缺各 category 独立断言 |
| 持久化 | 无 — 用例内存注册（`cases/*.go` 硬编码） | 本任务**不引入持久化** |
| 风险登记 | R-202 P2 Open（"仅 RPC/Protocol/DataModel 三类"） | 关闭门槛见 §7 度量 |

### 1.2 R-202 风险描述

> F10 互操作用例库稀疏：仅 RPC/Protocol/DataModel 三类，14 用例，无失败路径用例，无运营商差异化矩阵。GA 阶段交付给客户时如出现"用 OMC 内置互操作工具仍漏检产线问题"会损害商用形象。

---

## 2. 用户故事

| 角色 | 故事 |
|------|------|
| QA 工程师 | 我希望对新接入的 CPE 一键跑完互操作矩阵（成功路径 + 失败路径），输出通过率与失败明细 |
| 设备研发 | 我希望产线测试 OMC 互操作工具能识别厂商私有参数与非法值，提前发现协议偏差 |
| 现场运维 | 我希望在客户验收阶段，跑互操作用例库等价于跑了行业基线测试，得到可签字的报告 |
| 测试基线维护者 | 我希望新增用例像加单元测试一样轻量（在 `cases/*.go` 加一个 struct），低门槛持续扩充 |

---

## 3. 验收标准（Given-When-Then）

### V1 — 用例数量与分类完整
- **Given** 互操作用例库已完成 Phase 1 扩充
- **When** 调 `GET /api/v1/interop/test-cases`
- **Then** 返回 ≥ **30** 条用例（当前 14 + 新增 ≥ 16）
- **And** 每个 category（datamodel / protocol / rpc / inform）均 ≥ 4 条
- **And** 每个 category 至少含 1 条 **negative path**（预期失败，验证 OMC 能识别错误输入）

### V2 — Inform 类用例补齐
- **Given** Phase 1 扩充已完成
- **When** 调 `POST /api/v1/interop/run/inform`
- **Then** 返回 ≥ 4 条 inform-related 用例的运行结果（首启 BOOTSTRAP / 周期 PERIODIC / 配置变更 VALUE CHANGE / 参数请求 CONNECTION REQUEST）
- **And** 当前 category 不存在 inform 时，该断言现在会失败 → 验收前必须先在 `cases/` 下加 `inform_cases.go`

### V3 — 失败路径可识别
- **Given** 用例库含 negative path 用例（e.g. "RPC GetRPCMethods 返回非法 SOAP" / "DataModel 参数路径不在模型中" / "Inform 事件码越界"）
- **When** 调 `POST /api/v1/interop/run` 执行该类用例
- **Then** 单元 result 状态为 `failed`（非 `pass`），且 `failure_reason` 字段非空，含具体不符项

### V4 — E2E 断言增量
- **Given** Phase 1 扩充已完成
- **When** 跑 `bash scripts/e2e_verify.sh`
- **Then** interop 域 claim 数从当前 3 → ≥ **6**（每 category 至少 1 claim：iop-list / iop-run-protocol / iop-run-datamodel / iop-run-rpc / iop-run-inform / iop-validate）
- **And** 所有 claim 通过率 100%（PASS / 0 FAIL）

### V5 — 单元测试同步
- **Given** 新增用例已注册到 `cases/<category>_cases.go`
- **When** 跑 `go test ./internal/interop/...`
- **Then** 用例总数断言与代码计数一致（runner_test `TestAllCasesRegistered` 等）
- **And** 单元测试通过率 100%

---

## 4. 运营商差异矩阵

| 维度 | CMCC | CTCC | CUCC | 本任务处理 |
|------|------|------|------|------------|
| Inform 事件码扩展 | 含 `M Download` / `M Upload` 移动私有 | 标准 + 私有 `M Reboot` | 标准 + 私有标识 | Phase 1 仅覆盖**标准事件码 0/1/2/4/6/7**；运营商私有事件码 placeholder + 留 Phase 2 |
| DataModel 命名前缀 | `Device.X_CMCC_*` 私有节点（计数器/告警 OID） | `Device.X_CT-COM_*` | `Device.X_CU-COM_*` | Phase 1 用例覆盖**通用基线 + cmcc 私有节点 1 条**示例；其余厂商在用例 description 加 `// Carrier: ctcc/cucc` 标签即可（用例本身可参数化为"运营商无关"） |
| RPC 参数差异 | 多数标准；部分私有扩展（如 X_CMCC_Reboot） | 多数标准 | 多数标准 | Phase 1 不专门做运营商差异化 RPC 用例（用 Carrier 抽象处理） |
| 互操作报告格式 | 移动有专有验收模板 | 电信有 | 联通无 | 本任务**不**生成报告（沿用现有 `/run` JSON 输出）；后续 Phase 2 可考虑 markdown/PDF export |

**核心原则**：用例库结构按"标准 + carrier-specific 标签"组织，新增用例时通过 description 加 `Carrier:` 标签声明归属，不在用例代码里 `if cmcc/ctcc/cucc` 硬编码（仍遵守 §16.4 Carrier 抽象原则）。

---

## 5. 非目标（Phase 1 不做）

- ❌ 不重写 `runner.go` / `validator.go` 架构 — 沿用现有注册表 + 执行循环
- ❌ 不引入用例持久化（数据库表 / migration）— 继续走内存注册
- ❌ 不接真实设备硬件 — 用 `scripts/cpe_simulator.py` 模拟（已存在）
- ❌ 不做 GA 级压测 / 并发 — T-0023（5K 设备 24h 压测）的范畴
- ❌ 不做运营商专有验收报告 export（PDF / markdown / xlsx）— 留 Phase 2 选项
- ❌ 不动前端 `InteropTesting/index.tsx`（现有列表 + run + validate 已满足新用例量）
- ❌ 不做 RPC 协议私有扩展用例（X_CMCC_Reboot 等）— 留 Phase 2

---

## 6. 依赖

| 依赖 | 类型 | 状态 |
|------|------|------|
| `internal/interop/` 现有架构 | 内部代码 | ✅ ready |
| `runner.Register()` API | 内部接口 | ✅ ready（cases 注册示例已有） |
| `scripts/e2e_verify.sh` 框架 | 测试基线 | ✅ ready |
| 无外部 trigger | — | ✅ 立即可开工 |

---

## 7. 度量（含 R-202 关闭门槛）

| 指标 | 原状 | Phase 1 目标 | Phase 2 目标（2026-05-11 修订）|
|------|------|--------------|--------------------------------|
| 用例总数 | 14 | **≥ 30** ✅ (实际 30) | **≥ 35**（原 ≥ 50）|
| Category 数 | 3（DM / Protocol / RPC） | **4**（+ Inform）✅ | 4（保持，fault-inject 可独立子任务）|
| Negative path 用例数 | 0 | **≥ 4**（每 category ≥ 1）✅ | **≥ 8**（每 category ≥ 2）|
| 运营商覆盖 | 通用基线 | 标签声明法 ✅ | **三家覆盖**（CMCC + CTCC + CUCC 至少各 1 私有 RPC 用例）|
| E2E claim 数（interop 域）| 3 | **≥ 6** ✅ | **≥ 8**（原 ≥ 10）|
| 单元测试通过率 | 100% | **100%**（含新增）✅ | **100%** |
| R-202 状态 | Open | **Mitigating** ✅ | **Closed** |

**Phase 2 门槛 2026-05-11 修订理由**：原门槛（用例 ≥ 50 / e2e ≥ 10）在字段池有限的现实下会产生"伪用例"（多个 step 复制粘贴只改 Name）有损用例库质量。修订后门槛聚焦三件事 — (a) 每类 negative ≥ 2 让 negative 机制覆盖度饱满；(b) 三家运营商覆盖是 Phase 2 旗舰需求；(c) 用例 ≥ 35 增量保守不强求伪用例。**真正未做的 Phase 2 选项**（fault-inject category / 验收报告 export / 设备型号矩阵参数化）留独立 sub-task，需架构改动不宜并入 T-0030。

**R-202 闭环路径**：Phase 1 完成 → P2 Mitigating；Phase 2 修订门槛达成 → P2 Closed。

---

## 8. 范围分阶段

### Phase 1（本 Sprint，T-0030 范围）
- 新增 `cases/inform_cases.go` — 4 条 Inform 类用例（标准事件码 0/1/2/4/6/7 至少覆盖到 4 个）
- 既有 3 类各补 ≥ 4 条（含 ≥ 1 个 negative path）：
  - `cases/datamodel_cases.go` 2 → 6
  - `cases/protocol_cases.go` 3 → 7
  - `cases/rpc_cases.go` 9 → 13
- 合计新增 ≥ 16 条，达 ≥ 30 条总量
- 同步 `runner_test.go` + `service_test.go` 用例计数断言
- `scripts/e2e_verify.sh` interop 域加 3 个 claim（按 category 拆 + 1 个 negative）

### Phase 2（sprint-11+ 可选，不在本任务承诺）
- 运营商私有 RPC 用例（X_CMCC_Reboot / X_CT-COM_* 等）
- 运营商专有验收报告 export
- 设备型号矩阵参数化（NR/LTE × 主流型号）
- fault-inject 类 category

如 Sprint-10 余力允许，可提前启动 Phase 2 部分，但不计入 T-0030 的 DoD。

---

## 9. 设计备忘（S2）

### 9.1 用例 ID 命名规范

沿用现有 `<CAT>-NNN` 三段式：
- `DM-001..002`（已有）→ `DM-003..006`（新增 4）
- `PROTO-001..003`（已有）→ `PROTO-004..007`（新增 4）
- `RPC-001..009`（已有）→ `RPC-010..013`（新增 4）
- **新建** `INF-001..004`（Inform 类，对应标准事件码 0 BOOTSTRAP / 1 BOOT / 2 PERIODIC / 4 VALUE CHANGE）

### 9.2 注册入口

`omcgo/cmd/app/provider/modules.go:308-310` 现有三行注册：

```go
testRunner.RegisterCases(cases.ProtocolCases())
testRunner.RegisterCases(cases.DataModelCases())
testRunner.RegisterCases(cases.RPCCases())
// S3 新增：
testRunner.RegisterCases(cases.InformCases())
```

`cases/` 目录新增 `inform_cases.go`，导出 `InformCases() []interop.TestCase` 函数（与现有三文件同模式）。

### 9.3 Negative path 机制

**当前模型**：`TestCase` struct 含 `ID/Name/Description/Category/Steps`，runner 默认所有 step 都"应当成功"，任一 step 失败则 result.Status=failed。

**S3 改造**（最小变更）：
- `TestCase` 新增字段 `ExpectedOutcome string`（取值 `pass` / `fail`，默认空 = `pass`）
- `runner.runCase` 在汇总时对比：
  - `ExpectedOutcome=""` 或 `"pass"`：与现状一致（任 step 失败 → result=failed）
  - `ExpectedOutcome="fail"`：runner 期望至少一个 step 失败，**全过反而 result=failed**（标注 reason="negative case unexpectedly passed"）
- `TestResult` 新增字段 `ExpectedOutcome string`（透传），前端可选展示
- 现有用例不需任何修改（兼容默认 pass）

> **改动面**：`model.go` +2 字段（TestCase / TestResult）+ `runner.go` runCase 汇总逻辑 ~15 行 + `runner_test.go` 新增 1-2 个 negative path 单测。无 migration、无 API 形态变更（JSON 输出多 2 个可选字段）。

### 9.4 Inform 用例的 mock 策略

Inform 类用例与现有 3 类不同：现有用例本质是"向 ACS 端发 RPC / 比对参数树"，而 Inform 是 CPE 主动上报，OMC 端是"接收 + 解析 + 路由"。

**S3 选择内联 fixture，不接 cpe_simulator.py**：
- 新增 `test/fixtures/interop/inform-bootstrap.xml` / `inform-boot.xml` / `inform-periodic.xml` / `inform-valuechange.xml`（标准 SOAP/CWMP 报文样例，~50 行/个）
- `cases/inform_cases.go` 中每个用例的 Step Params 用 `fixture_path` 引用 fixture 文件
- runner 新增 action `verify_inform_parse`：读 fixture → 调内部 SOAP 解析器（如 `internal/acs/soap`）→ 校验事件码 / 序列号 / Inform 结构完整性
- **不接真实 ACS 会话**，因为本任务范围是"用例库扩充"而非"端到端 Inform 处理验证"（后者已被 ACS 单测和 cpe_simulator 覆盖）

> **可选简化路径**：如果引内部 SOAP 解析器耦合过深（cases 包反过来依赖 acs 包不太干净），可在 cases/inform_cases.go 内联一个 minimal XML 节点检查（用 `encoding/xml` 直接解 Envelope/Body/Inform/EventStruct/EventCode）。S3 起手第一步先 grep `internal/acs/soap` 看接口可用性再定。

### 9.5 E2E claim 增量

`scripts/e2e_verify.sh` 现有 3 条 interop claim：
- `iop-1` GET /interop/test-cases
- `iop-2` POST /interop/validate
- `iop-3` POST /interop/run

S3 新增 3 条（达 V4 ≥ 6 门槛）：
- `iop-4` POST /interop/run/protocol（按 category 跑，断言 result 数 ≥ 7）
- `iop-5` POST /interop/run/datamodel（断言 result 数 ≥ 6）
- `iop-6` POST /interop/run/inform（断言 result 数 ≥ 4 + 新 category 存在）
- 可选：`iop-7` POST /interop/run + 断言 ExpectedOutcome 字段在响应中（验证 negative path 已挂上）

每个 claim 用现有 `check_status` + `check_field` 框架（grep `scripts/e2e_verify.sh` 已有的 claim 风格沿用）。

### 9.6 Carrier 差异处理

新增用例代码内**不**写 `if carrier == "cmcc/ctcc/cucc"`，遵循 §16.4。运营商相关用例通过 description 加标签声明归属：

```go
{
    ID:          "DM-005",
    Name:        "CMCC Private Counter Node Path Existence",
    Description: "Carrier: cmcc — Verify Device.X_CMCC_Counters.* nodes are reported when product matches cmcc datamodel",
    Category:    interop.CategoryDataModel,
    ...
}
```

前端 / 报告未来若要做"按运营商筛选"，从 description 解析 `Carrier:` 前缀即可（Phase 2 选项）。

### 9.7 待定点（< 3，符合 S2 出口门）

| W | 描述 | 解法位置 |
|---|------|---------|
| W1 | Negative path 用例的 step 该写哪个 action？现有 `verify_response` / `check_param` / `send_rpc` 三个都是 success path 语义 | S3 起手第一动作：在 cases/ 实写一条 negative 后看 runner 是否自然 fail；不行就加 action `expect_failure` |
| W2 | Inform fixture 是放 `test/fixtures/interop/` 还是 `internal/interop/cases/fixtures/`？前者更通用、后者更内聚 | S3 起手第二动作：看 cases 包是否能 embed 静态文件；优先 `//go:embed` 路线放 `cases/fixtures/`（编译期嵌入，避免运行时路径依赖） |

### 9.8 S3 实施清单（预计 ~12 步 / ~1.5-2 工作日）

1. `internal/interop/model.go` 加 `TestCase.ExpectedOutcome` + `TestResult.ExpectedOutcome` 字段
2. `internal/interop/runner.go` runCase 末尾加 expected outcome 判定逻辑
3. `internal/interop/runner_test.go` 加 negative path 单测（覆盖 W1 决策）
4. `internal/interop/cases/inform_cases.go` 新增 4 个用例 INF-001..004
5. 视 W2 决议放置 inform fixture XML（4 个文件）
6. `cmd/app/provider/modules.go:311` 加 `testRunner.RegisterCases(cases.InformCases())`
7. `internal/interop/cases/datamodel_cases.go` 加 DM-003..006（其中 ≥ 1 negative）
8. `internal/interop/cases/protocol_cases.go` 加 PROTO-004..007（其中 ≥ 1 negative）
9. `internal/interop/cases/rpc_cases.go` 加 RPC-010..013（其中 ≥ 1 negative）
10. `internal/interop/runner_test.go` / `service_test.go` 同步更新用例数断言（如 `len(cases) == N`）
11. `scripts/e2e_verify.sh` interop 域加 3 个 claim（iop-4/5/6）
12. 全量 `go build ./...` / `go test ./internal/interop/...` / `bash scripts/e2e_verify.sh` 三套验证

### 9.9 观测埋点

本任务不新增 Prometheus metric / Zap log key（用例量增长是隐式行为，由 `/test-cases` 返回数量+测试通过率反映；新指标会 over-engineer P2 任务）。

---

---

## 10. 风险

| 风险 | 概率 | 影响 | 缓解 |
|------|------|------|------|
| Inform 用例需 mock CPE 报文 | 中 | 用例只能跑半（数据来自 fixture） | 用 `test/fixtures/interop/inform-*.xml` 准备静态报文；不依赖在线 CPE |
| 用例量增多后 `/run` 超时 | 低 | 30 条全跑 < 5s（每条平均 ~100ms），无需分页 | 监测 `/run` 耗时；超 30s 再考虑 streaming |
| Negative path 用例可能误标"失败" | 中 | 报告显示一片红，QA 误读 | 在 runner result 里加 `expected_status` 字段；UI 区分"按预期失败" vs "意外失败" — 留 Phase 2 |

---

**完整 rationale + sub-task 拆分（如有）**：S2 设计阶段补充。
