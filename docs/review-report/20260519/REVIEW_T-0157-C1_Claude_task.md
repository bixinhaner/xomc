# Code Review — T-0157 C1 task 超时配置化 + CreateTask 默认值兜底

- **日期**：2026-05-19
- **范围**：`internal/core/appconfig/config.go` / `internal/task/{service,service_test,service_expires_test}.go` / `cmd/app/bootstrap.go` / `cmd/app/etc/config.{dev,test,prod}.yaml`
- **作者**：Claude
- **Reviewer**：Claude（self-review，Go 工程 + 架构 + TR-069 视角）
- **关联**：T-0157 Phase 1 sub-task **C1**

---

## 变更概要

1. `appconfig.TaskConfig` 新结构体：`DefaultExpiresInSeconds` / `SweepIntervalSeconds`（C2 用，C1 阶段占位）
2. `AppConfig.Task` 字段挂载
3. `TaskService` 新增字段 `defaultExpiresIn` + setter `SetDefaultExpiresIn`
4. `TaskService.CreateTask` 在 NewTask 前兜底：`req.ExpiresIn == 0 && defaultExpiresIn > 0 → req.ExpiresIn = defaultExpiresIn`
5. `cmd/app/bootstrap.go` 注入配置值到 TaskService
6. 三份 yaml 加 `task:` 段（dev/test/prod 默认 120s + 10s sweeper 间隔）
7. 新增专项测试 `service_expires_test.go`：4 例 table-driven + 1 例 negative 规范化

---

## 出口门验证

| 门 | 结果 |
|---|---|
| `go build ./...` | ✅ |
| `go test ./internal/task/...` | ✅ ok 6.4s |
| `go test ./internal/core/appconfig/...` | ✅ ok 1.2s |
| 3 种分支 table-driven 覆盖（未传/显式 0/显式值/无配置默认） | ✅ 4 例全通过 |

---

## Checklist

| 类别 | 项 | 结果 |
|---|---|---|
| 接口设计 | setter 模式与 SetMetrics/SetEventBus 一致（"接受具体，可选注入"） | ✅ |
| 兜底语义 | `defaultExpiresIn <= 0` 等价历史行为（不兜底） | ✅ |
| 负值防御 | SetDefaultExpiresIn(-1) 规范化为 0；防误配置导致 task 立即过期 | ✅ |
| 调用方覆盖 | 业务显式传 ExpiresIn > 0 不被兜底覆盖（ops 60s / alarm 600s 仍生效） | ✅ |
| 业务向后兼容 | 现有 20 处 CreateTask 调用点零改动，全部继承默认 120s 兜底 | ✅ |
| 配置一致性 | dev/test/prod 三份 yaml 同步加 task 段 | ✅ |
| 注释 | TaskConfig 字段 godoc 说明三种分支语义 + 取值约定 | ✅ |

---

## 发现

### CRITICAL
- 无

### WARNING
- 无

### INFO
- I-01：testableTaskService 是 service_test.go 内的 mirror（mock 队列+repo 复制 CreateTask 逻辑），本次同步加了相同兜底分支，保持单测对真实服务的覆盖等价。维护成本：未来 CreateTask 主路径有改动时需同步两边——历史就是这样设计的，无新增技术债。
- I-02：`SweepIntervalSeconds` 字段已在 TaskConfig 中占位但 C1 不使用，C2 (task_sweeper) 会消费。配置文件已写入 10s 默认值，避免 C2 实施时再改三份 yaml。
- I-03：acs / worker 进程的 TaskService 也存在（cmd/acs/main.go:81 + cmd/worker/bootstrap.go:58），但**它们不调 CreateTask**（acs/worker 只读 task / 处理终态），所以无需注入 defaultExpiresIn。验证：`grep .CreateTask omcgo/cmd` 零命中。
- I-04：本次未改业务调用点（device/config/ops/alarm 等的 CreateTask 调用）—— 兜底由 service 层接管，业务代码不感知。未来若想"明确声明永不超时"，业务可显式传一个极大值（如 86400）；当前没有这种诉求。

---

## 与设计文档对齐

| §4.6 项 | 实施 |
|---|---|
| 配置化 default_expires_in_seconds=120 | ✅ 三份 yaml |
| sweep_interval_seconds=10 | ✅ 占位（C2 消费） |
| CreateTask 默认值兜底 | ✅ |
| 调用方可显式覆盖 | ✅ |
| 业务硬编码（ops 60s / alarm 600s）保留 | ✅ |

---

## 测试

- 4 例 table-driven 覆盖 §4.6 兜底语义的 4 种组合
- 1 例负值规范化防御测试
- 无网络 / DB / Redis 依赖，运行时间 < 0.1s
- 通过 mock 队列 / mock repo（既有基础设施）

---

## 结论

**PASS**

可合入，可继续 C2（task_sweeper）。
