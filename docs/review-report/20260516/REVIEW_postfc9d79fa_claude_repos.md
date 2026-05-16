# 代码审查报告 — V-1 软件/拓扑 mock 接口签名 + Download SOAP 模板 cwmp 前缀

| 项 | 值 |
|---|---|
| 审查时间 | 2026-05-16 |
| 审查对象 | 5 文件 / +26 / -26 |
| 基线 commit | fc9d79fa（段 1） |
| 审查者 | Claude（自审） |
| 关联 Backlog | T-0137（hygiene 副产品，T-0137 活体验证时 `go vet ./...` 全量跑发现的预存问题） |
| 性质 | hygiene / 协议合规 hotfix |
| **审查结论** | **PASS** |

---

## 1. 变更范围

### 1.1 测试 mock 接口签名补齐（4 文件）

`SubTaskRepository.List` / `ListByTaskID` 接口的返回类型在 commit `a6487474` 之后改成 `ListResponse[UpgradeSubTaskWithTaskName]`（之前是 `UpgradeSubTask`），但下列 mock 漏改：

| 文件 | 修改 |
|---|---|
| `omcgo/internal/software/handler_test.go` | `swHSubTaskRepo.List/ListByTaskID` 返回类型对齐 |
| `omcgo/internal/software/service_test.go` | `svcMockSubTaskRepo.List/ListByTaskID` 返回类型对齐 |
| `omcgo/test/integration/api_software_test.go` | `swHSubTaskRepoStub.List/ListByTaskID` 返回类型对齐 |
| `omcgo/internal/topology/handler_test.go` | `mockTopoNodeRepo.ListAll` 加 2 个 `*string` 参数 + `NewHandler` 调用补 2 个参数（对齐 `6f5e4156` 拓扑 CRUD 接口扩展） |

### 1.2 Download SOAP 模板 cwmp 前缀补齐

| 文件 | 修改 |
|---|---|
| `omcgo/pkg/soap/templates.go` | `downloadXML` 模板内 13 个子元素（CommandKey / FileType / URL / Username / Password / FileSize / TargetFileName / DelaySeconds / Md5 / RawMode / SuccessURL / FailureURL）补 `cwmp:` namespace 前缀，与 `uploadXML` 风格保持一致 |

---

## 2. 根因分析

### 2.1 测试 mock 漏改

接口签名变更的 commit `a6487474 fix(acs)` 和 `6f5e4156 feat(topology)` 只更新了主实现，编译器允许 _test.go 不参与生产构建但 `go vet ./...` / `go test ./...` 会报错。这俩问题积累了一段时间，直到 T-0137 活体验证时跑全量 vet 才浮出。**影响范围**：

- `go vet ./...` 失败 — 违反 DoD §10 "无编译器/linter 警告"
- `go test ./...` 失败 — software / topology 两个包的所有测试都跑不了，**回归保护盲区**

### 2.2 Download SOAP 模板漏修

commit `a6487474 fix(acs): 修复 CWMP SOAP 模板子元素缺少 cwmp: 命名空间前缀` 修了 Upload 但漏了 Download。**TR-069 协议合规问题**：

- TR-069 Amendment 6 §3.2 SOAP 命名空间约定，cwmp namespace 下所有元素应携带前缀
- 不同厂商 CPE 对无前缀子元素的容忍度不同，有些会忽略，有些会 SOAP Fault
- 本修复前 Download RPC 在严格 CPE 上可能不工作（联调期未必能复现，活体测试才能发现）

---

## 3. 检查项

| 检查项 | 状态 | 备注 |
|---|---|---|
| `go build ./...` | ✅ | 通过 |
| `go test ./...` | ✅ | 全绿（含 software / topology / acs/rpc 三个之前 fail 的包）|
| `go vet ./...` | ✅ | 全绿 |
| `TestDownloadHandler` | ✅ | 修复前 fail（5 个 assertion），修复后 pass |
| 接口签名一致性 | ✅ | mock 现在精确匹配 `SubTaskRepository` / `TopoNodeRepository` 接口 |
| SOAP 风格一致 | ✅ | downloadXML 现与 uploadXML 风格相同（全 cwmp 前缀） |
| 破坏性变更 | ✅ N/A | 测试 mock 改动只影响测试编译；SOAP 模板改变了 wire format，但旧 wire format **本来就违反规范**，CPE 端的行为应是"更宽容地接收"而非"破坏" |

---

## 4. 发现

### 4.1 CRITICAL

**无**。

### 4.2 WARNING

**无**。

### 4.3 INFO

- 这两个问题积累在 main 分支已 commit，按 git blame 看不是本 sprint 引入。本次顺手修是因为 T-0137 验证时跑全量 vet 才发现。建议在 CI 加强：每个 PR 跑 `go test ./...` + `go vet ./...` 避免再积累。

---

## 5. 结论

**PASS** — 可合入。

- 0 CRITICAL / 0 WARNING
- 全套自动化检查通过
- 修复了 TR-069 Download RPC 的真实协议合规问题（不止测试 hygiene）
