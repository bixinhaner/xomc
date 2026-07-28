# Task 1 收尾报告：固化指标/标签/队列目录契约

## 结论

Task 1 已完成。当前 diff 与 brief 的实现范围一致，未发现需要修正的代码或文档问题。

## 改动

- `deployments/monitoring/README.md`
  - 增加资源与队列指标契约表，记录指标后缀、标签、单位、真实 0 和采集失败语义。
  - 固定 8 个持久化队列目录：`device_tasks`、`async_jobs`、`parameter_sync_outbox`、`northbound_outbox`、`pm_kpi_export`、`trace_export`、`backup_tasks`、`dead_letters`。
  - 固定状态值，并说明 PromQL 查询失败/序列缺失与真实 0 的区别。
  - 明确排除 Go Channel、Worker Channel、参数同步内存 Channel、Trace 本地 Capture Queue 等应用内存队列。
- `docs/design/资源监控与队列治理及磁盘写入保护功能设计-20260728.md`
  - 补充 Task 1 的 queue/status/result 标签边界、高基数禁止规则、单位和可用性语义。
  - 补充 PM 指标的 `subject`/`durable` 枚举和采集失败行为。
- `omcgo/internal/core/event/metrics.go`
  - 增加 PM 指标契约注释，说明低基数标签、单位、真实 0、保留上次值和失败计数语义。
- `omcgo/internal/core/event/metrics_test.go`
  - 增加低基数回归断言，确认 device SN、完整 Redis key、对象路径等不进入指标标签。
- `omcgo/internal/task/metrics.go`
  - 增加固定持久化队列目录、指标后缀、状态值和有限标签名契约常量/注释。
- `omcgo/internal/task/metrics_test.go`
  - 增加低基数标签断言及目录、指标后缀、状态值契约断言。

`.codex-tools/` 是工作区已有的未跟踪浏览器运行时目录，不属于 Task 1，未暂存、未提交。

## 测试与输出

### 格式检查

命令：

```bash
gofmt -d omcgo/internal/core/event/metrics.go omcgo/internal/core/event/metrics_test.go omcgo/internal/task/metrics.go omcgo/internal/task/metrics_test.go
```

输出：无输出，表示无需格式化调整。

### Diff 检查

命令：

```bash
git diff --check
```

输出：无输出，检查通过。

### Task 1 focused tests

命令：

```bash
cd omcgo && go test -count=1 ./internal/core/event ./internal/task
```

第一次在普通 sandbox 中执行时，event/task 的部分既有测试因环境禁止本地监听而失败，错误均为：

```text
listen tcp 127.0.0.1:0: bind: operation not permitted
```

按仓库 AGENTS.md 的环境规则在允许本地监听的提权环境中复跑同一命令，输出为：

```text
ok  github.com/omcgo/omcgo/internal/core/event  3.971s
ok  github.com/omcgo/omcgo/internal/task       1.380s
```

此前缓存执行的 brief 指定命令也通过：

```text
ok  github.com/omcgo/omcgo/internal/core/event  (cached)
ok  github.com/omcgo/omcgo/internal/task       (cached)
```

## 风险与疑问

- Task 1 只固化契约，没有创建新的持久化队列观测器或 Prometheus 指标实现；后续任务必须复用本报告及 README/设计文档中的目录、标签、状态和失败语义。
- `PersistentQueueNames` 与 `PersistentQueueStatuses` 当前是导出的 slice，调用方理论上可以修改其内容；本 Task 仅要求契约登记和回归断言，是否改为不可变访问方式应在后续任务需要时单独评估。
- 普通 sandbox 的本地监听失败属于测试环境限制；提权后的 focused tests 已通过。

## Fix round 1：受控枚举与非法标签拒绝

### 审查问题与根因

审查发现 `PersistentQueueNames`、`PersistentQueueStatuses` 是可由外部包直接修改的导出 slice，且没有受控的 label 构造/校验入口。因此后续实现可以绕过契约，将 device SN、完整 Redis key 或对象路径作为 queue/status/result 标签。

### TDD RED

先在 `omcgo/internal/task/metrics_test.go` 增加以下行为测试：

- `TestPersistentQueueLabelsRejectUnregisteredValues`：非法 queue、status、result 必须被 `NewPersistentQueueLabels` 拒绝。
- `TestPersistentQueueLabelsExposeOnlyValidatedValues`：合法值可被构造并导出；修改 `PersistentQueueNames()` 返回值不能改变注册目录。

命令：

```bash
cd omcgo && go test -count=1 ./internal/task -run 'TestPersistentQueue(Labels|MetricContract)'
```

在实现接口前，提权执行输出为预期编译失败：

```text
invalid operation: cannot call PersistentQueueNames (variable of type []string)
invalid operation: cannot call PersistentQueueStatuses (variable of type []string)
undefined: NewPersistentQueueLabels
FAIL github.com/omcgo/omcgo/internal/task [build failed]
```

普通 sandbox 运行同一 RED 命令另因 Go 构建缓存权限报错：

```text
open /Users/a1/Library/Caches/go-build/...: operation not permitted
```

### GREEN 修复

- 将队列目录和状态改为不可变字符串常量。
- 将 `PersistentQueueNames()`、`PersistentQueueStatuses()` 改为每次返回新 slice 的函数，避免外部修改内部注册表。
- 增加 `PersistentQueueLabels` 受控值类型及 `NewPersistentQueueLabels(queue, status, result)` 校验入口；非法 queue/status/result 返回错误，空 result 仅用于不使用 result 维度的指标。
- 增加 `Values()` 作为后续指标构造使用的已校验值出口；未新增任何 observer 或具体指标实现。

GREEN 命令：

```bash
cd omcgo && go test -count=1 ./internal/task -run 'TestPersistentQueue(Labels|MetricContract)'
```

输出：

```text
ok  github.com/omcgo/omcgo/internal/task  3.603s
```

Fix round 1 仅涉及 `omcgo/internal/task/metrics.go` 和 `omcgo/internal/task/metrics_test.go`，未扩大到后续 observer 实现。

### Fix round 1 收尾补充

- `omcgo/internal/task/metrics.go`
  - 增加 `PersistentQueueLabels.LabelValues()` 受控出口，只返回已校验的 queue/status/result 标签值；未使用 result 维度时省略该标签。
  - 对零值或不合法的标签对象返回错误，不能通过该出口产生 Prometheus 标签值。
- `omcgo/internal/task/metrics_test.go`
  - 增加 `TestPersistentQueueLabelsOnlyProduceValidatedMetricValues`，覆盖合法标签导出和零值拒绝。

TDD RED 命令：

```bash
cd omcgo && go test -count=1 ./internal/task -run 'TestPersistentQueueLabelsOnlyProduceValidatedMetricValues'
```

输出：预期编译失败，`PersistentQueueLabels has no field or method LabelValues`。

TDD GREEN / focused tests：

```bash
cd omcgo && go test -count=1 ./internal/task -run 'TestPersistentQueue(Labels|MetricContract)'
```

普通 sandbox 首次运行受 Go 构建缓存权限限制失败：

```text
open /Users/a1/Library/Caches/go-build/...: operation not permitted
```

按仓库环境规则提权复跑输出：

```text
ok  github.com/omcgo/omcgo/internal/task  0.515s
```

Concerns：仅验证 Task 1 的枚举/标签契约；未运行全量测试，也未实现后续持久化队列 observer。`.codex-tools/` 保持未跟踪，未提交。

### Fix round 1 最终收尾

本轮最终 focused test：

```text
$ cd omcgo && go test -count=1 ./internal/task -run 'TestPersistentQueue(Labels|MetricContract)'
ok  github.com/omcgo/omcgo/internal/task  2.386s
```

本轮提交：`fix(task): 完善持久化队列指标标签值出口`；基于 `4e2c6730c`，仅包含 `LabelValues` 收尾实现、对应测试及本报告更新。`.codex-tools/` 未提交。
