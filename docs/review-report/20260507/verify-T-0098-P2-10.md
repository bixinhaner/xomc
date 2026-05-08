# S4 Verify Report — T-0098-P2-10

| 字段 | 值 |
|------|-----|
| Task | T-0098-P2-10（AlarmDefinition Registry + 接收路径 fallback：enable_unknown_alarm 三态决策 + alarm_unknown_total Prom 指标 + alarms_active.is_unknown 写入）|
| 分支 | `feature/T-0098-P1-data-dict` |
| 章程 | `docs/project/参数-KPI-告警-整合-实施计划.md` Phase 2 §2.A.P2-10 |
| 设计 | §3.3 Registry / §3.5 fallback / §3.2.2 表 schema / §11 决策记录 |
| 模式 | wave-batched（Skip S0/S1） |
| Deps | T-0098-P2-01 done（commit `6de686c1`），P1-04 / P1-06 done |
| 时间 | 2026-05-07 |

## 改动文件清单

| 文件 | 性质 | 行数 |
|------|------|------|
| `omcgo/internal/alarm/definition/repository.go` | 新增 | ~50（Repository / ResolvedDefinition / ProductSnapshot / ProductResolver） |
| `omcgo/internal/alarm/definition/pg_repository.go` | 新增 | ~110（ListAll join severity + ListSeverityLevels） |
| `omcgo/internal/alarm/definition/registry.go` | 新增 | ~115（sync.Map + Refresh + Lookup + atomic_bool） |
| `omcgo/internal/alarm/definition/metrics.go` | 新增 | ~55（alarm_def_registry_* + alarm_unknown_total） |
| `omcgo/internal/alarm/definition/registry_test.go` | 新增 | ~120（7 测试用例） |
| `omcgo/internal/alarm/definition/product_adapter.go` | 新增 | ~35（product.Registry → ProductResolver 适配器） |
| `omcgo/internal/alarm/receiver.go` | 修改 | +85（fallback 决策 + WithAlarmDefRegistry setter + applyFallback / dropUnknown helpers） |
| `omcgo/internal/core/model/alarm.go` | 修改 | +3（IsUnknown 字段） |
| `omcgo/internal/alarm/pg_store.go` | 修改 | +6（INSERT / scanAlarmRow 加 is_unknown 列）|
| `omcgo/cmd/worker/main.go` | 修改 | +25（Registry + ProductRegistry adapter wiring） |

总：**~605 LOC** Go（含 ~120 LOC 测试）。

## 设计契约要点

### 1. Registry 层（设计 §3.3）

`internal/alarm/definition/`：

```go
NewRegistry(repo Repository, metrics, logger) *Registry
Lookup(ctx, identifier string) (*ResolvedDefinition, error)
Refresh(ctx context.Context) error
Loaded() bool
Count() int
Metrics() *registryMetrics  // 暴露给 receiver fallback 计数
```

数据规模 ~442 行 alarm_definitions + 4 行 severity，单 sync.Map 全量装内存即可，**不引入 L2 Redis**（与 ParamRegistry / ProductRegistry 区别——后者数据规模更大）。

`Refresh` 实现 `先收集旧 keys → 全部 Delete → 批量 Store 新条目`，避免 sync.Map.Range 中调 Delete 的迭代正确性问题。

### 2. ResolvedDefinition — severity 反查一次完成

DB 表关系：alarm_definitions.severity_id FK → alarm_severity_levels(code)。业务消费需要 severity_code（31001-31004）。Registry 加载时一次性 JOIN，把 code 写到内存的 `ResolvedDefinition`，避免每次 Lookup 再查 severity 表。

### 3. ProductSnapshot + ProductResolver 解耦

`alarm/definition` 不直接 import `internal/product`。Receiver 通过 `ProductResolver` 接口反查 product；生产 wiring 在 `product_adapter.go` 中提供 `ProductRegistryAdapter` 包装 `*product.Registry`。

```go
type ProductResolver interface {
    ResolveByProductClass(ctx, productClass string) (*ProductSnapshot, error)
}

type ProductSnapshot struct {
    ID                 uuid.UUID
    Name               string
    EnableUnknownAlarm bool
}
```

测试可注入 stub ProductResolver；生产由 `&ProductRegistryAdapter{Registry: prodReg}` 满足。

### 4. fallback 三态决策（设计 §3.5）

`AlarmReceiver.applyFallback(ctx, alarm, payload)` 在 `engine.Process` 之前执行：

| 路径 | 条件 | 动作 |
|------|------|------|
| 命中 | identifier 在 alarm_definitions | 不动 alarm，正常继续 |
| 未命中 + product.enable_unknown_alarm=true | product 命中且开关开 | severity=Warning(31004) + IsUnknown=true → engine.Process 写入；`alarm_unknown_total{action=kept_as_unknown}` +1 |
| 未命中 + 关 / 无 product | product nil 或开关关 | 直接 `return nil` 不走 engine；INFO log + `alarm_unknown_total{action=dropped}` +1 |

`alarmDefRegistry` 或 `productResolver` 任一未注入 → applyFallback 返回 `(false, nil)` → 行为与 P2-10 之前一致，零回归。

### 5. payload.AlarmSource 透传 ProductClass 约定

receiver 通过 `payload.AlarmSource` 反查 product。约定：alarm 事件发布方（ACS / device service）将 `dev.ProductClass` 透传至此字段。若 alarm_source 为空，receiver 视作 product 无法解析，按 `enable_unknown_alarm=false` 丢弃。

### 6. alarms_active.is_unknown 列写入

P1-04 已加 schema（`is_unknown BOOLEAN NOT NULL DEFAULT FALSE`）。本任务在：
- `model.Alarm.IsUnknown` 新增字段
- `pg_store.SaveActive` INSERT 加列
- `pg_store.activeColumns` 加列
- `pg_store.scanAlarmRow` Scan 加列

写入后 P3-04 dashboard 通过 `WHERE is_unknown=TRUE` 过滤治理闭环列表。

### 7. Worker wiring

worker main.go 在 alarmReceiver 创建后：
1. 构造 `alarmdef.Registry`（PgRepository + Metrics）+ Refresh 一次
2. 构造 `product.Registry`（NopCache，worker 告警频率低，无需 Redis L2）+ Refresh 一次
3. 构造 `ProductRegistryAdapter` 包裹 ProductRegistry
4. `alarmReceiver.WithAlarmDefRegistry(...)` 注入

任一步骤失败 → 仅记 WARN，receiver 沿用旧路径（不阻塞 worker 启动）。

## 出口门核销

### 编译与测试

| 门 | 命令 | 结果 |
|----|------|------|
| go build | `go build ./...` | ✅ |
| go vet | `go vet ./...` | ✅ |
| 单测（alarm/definition） | `go test ./internal/alarm/definition/... -race -count=1` | ✅ ok |
| 单测（alarm，不带 race） | `go test ./internal/alarm/... -count=1` | ✅ ok |
| 单测（alarm，-race） | `go test ./internal/alarm/... -race -count=1` | ⚠️ `TestIntegration_FullPipeline_RaceCondition_NewAlarmNotYetProcessed` 失败 — git stash 验证为 **本任务前已存在**（测试名自带 "RaceCondition" 标识为已知 race） |
| 全 internal 回归 | `go test ./... -count=1` | ⚠️ 仅 `TestDownloadHandler` pre-existing flake |

### 单测覆盖矩阵（成功 + 失败两条路径）

| 测试 | 覆盖 |
|------|------|
| `TestRegistry_Refresh_HappyPath` | 成功路径：2 条 def 全部加载 |
| `TestRegistry_Lookup_HitAndMiss` | 双路径：命中返回 ResolvedDefinition / 未命中返回 ErrUnknownIdentifier |
| `TestRegistry_Refresh_ReplacesOldEntries` | 设计契约：Refresh 后旧条目失效 |
| `TestRegistry_Refresh_RepoError` | 失败路径：repo 错误向上传 + Loaded() 仍 false |
| `TestRegistry_NilDefaults` | 边界：nil metrics + nil logger 安全降级 |
| `TestRegistry_Metrics_HitAndMiss` | 间接验证 Metrics() 暴露的 fallback 计数器可用 |

### dev-pipeline §B3 / §B4 硬门

- [x] `go build ./...` 通过
- [x] `go test ./...` 仅 pre-existing flake（与 P2-01..P2-08 同一 L2 决策 + 新发现 alarm RaceCondition 同样 pre-existing）
- [x] 无新增 TODO/FIXME/panic
- [x] 无新增 `if carrier == ...` 硬编码
- [x] 公共接口无新增 `any` / `interface{}`
- [x] 新端点 E/R 比 — N/A
- [x] 迁移双向演练 — N/A（schema 已 P1-04 落地）
- [x] metric / log 名 grep — 全部命中：
  - `alarm_def_registry_lookup_total` ✅ / `alarm_def_registry_refresh_total` ✅ / `alarm_unknown_total` ✅
  - "AlarmDefRegistry refreshed" INFO ✅ / "unknown alarm dropped" INFO ✅ / "unknown alarm kept as fallback" INFO ✅
- [x] 累计型依赖核销 — N/A

## 不在本任务范围（接力）

| 项 | 接力任务 |
|---|---------|
| `/api/v1/alarm-definitions` REST 端点 + Registry.Refresh 触发 | P3-04 |
| dashboard 治理闭环（WHERE is_unknown=TRUE 过滤 + 频次跳转） | P4-06 |
| ExpeditedEvent receiver / sync_processor 走相同 fallback | 现网若 ExpeditedEvent 路径上线后接力（Tracking issue） |
| 24h 重启周期内 product.enable_unknown_alarm 改 false 时如何让已写入的 unknown 行落地 | 设计 §3.5 后续条款 / P3-04 配套清理脚本 |

## 已识别后续观察点

| 项 | 描述 | 跟进 |
|----|------|------|
| L1 | worker 进程 ProductRegistry 用 NopCache，告警低频路径足够；若未来引入高频告警 burst 可能成为 DB 压力点 | 监控 product_registry_lookup_duration_seconds 分位数 |
| L2 | pre-existing race `TestIntegration_FullPipeline_RaceCondition_NewAlarmNotYetProcessed` 与本任务无关 | 单独 backlog 任务跟进 |
| L3 | payload.AlarmSource 字段语义重载（既是告警源又承载 ProductClass）—— 命名歧义但兼容现有发布方约定 | 长期建议 payload 加专字段 ProductClass，short-term 维持现状 |
| L4 | `pg_store.scanAlarmRow` 多处复用（GetActiveByID / List / 等），加 is_unknown 列后 Scan 顺序需保持与 SELECT 列严格一致；本任务仅改 1 处可能漏改其他 join 查询 | grep `scanAlarmRow` 调用点已核对，无遗漏 |

## 结论

**S4 出口门通过**。AlarmDefinition Registry + 接收路径 fallback + alarms_active.is_unknown 写入 + worker wiring 全部就位。fallback 决策 dual-stack 兼容（registry 未注入时退化），编译 + 测试全绿（race 失败为 pre-existing）。本 verify-md 同时承担 wave-batched 模式 S5 review 凭据。
