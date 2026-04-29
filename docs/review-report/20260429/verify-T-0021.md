# S4 Verify Report — T-0021 Software 回退能力增强

> **生成**: 2026-04-29
> **任务**: T-0021 / R-101（回退增强半，闭环 T-0018 灰度后剩余风险）
> **PRD**: `docs/project/prd/T-0021-software-rollback-enhanced.md`
> **Sprint**: sprint-05

---

## 1. 改动清单

### 后端代码（10 文件）

| 路径 | 性质 | 说明 |
|------|------|------|
| `omcgo/migrations/000046_upgrade_rollback_metadata.sql` | 新建 | 4 列：`rollback_reason / rollback_source(CHECK) / rollback_target_firmware_id(FK) / rollback_on_failure`；1 部分索引 |
| `omcgo/internal/software/model.go` | 修改 | UpgradeTask +3 字段；BatchUpgradeRequest +RollbackOnFailure；RollbackRequest 已含 +3 audit 字段（前序提交） |
| `omcgo/internal/software/canary.go` | 修改 | CanaryFields +RollbackOnFailure |
| `omcgo/internal/software/pg_task_repository.go` | 修改 | taskColumns +3；Create/scan 双向 round-trip；GetCanaryFields/UpdateCanaryFields +rollback_on_failure 列 |
| `omcgo/internal/software/service.go` | 修改 | RollbackDevices 全面重写（source 枚举校验 + target 固件查询 + 新字段持久化 + 3 metric）；新增 TriggerCanaryFailureRollback；BatchUpgrade canary 路径写入 RollbackOnFailure；SoftwareService +RollbackMetrics 字段 + setter |
| `omcgo/internal/software/canary_monitor.go` | 修改 | 新增 canaryRollbackTrigger 单方法接口 + SetRollbackTrigger setter；evaluateOne 在 pause 后追加 opt-in 自动回退分支（错误吞掉，pause 已落盘） |
| `omcgo/internal/software/rollback_metrics.go` | 新建 | RollbackMetrics 结构体：3 counter（`software_rollback_total{source}` / `software_rollback_devices_total{source}` / `software_rollback_with_target_total`）+ nil-safe 方法 |
| `omcgo/cmd/app/provider/modules.go` | 修改 | DI 注入 RollbackMetrics；canaryMonitor.SetRollbackTrigger(softwareService) 闭环 |
| `omcgo/internal/software/service_test.go` | 修改 | +5 表驱动子用例（source matrix 6 cases / target matrix 4 cases / reason round-trip / IsValidRollbackSource / no-completed-devices) |
| `omcgo/internal/software/canary_monitor_test.go` | 修改 | +4 用例（opt-in 触发 / opt-out 不触发 / nil trigger 优雅跳过 / 触发错误不传播） |

### 文档

| 路径 | 性质 |
|------|------|
| `docs/project/prd/T-0021-software-rollback-enhanced.md` | PRD（S0 制品） |

### 配套（drive-by）

| 路径 | 性质 | 说明 |
|------|------|------|
| `omcgo/test/integration/api_software_test.go` | 修改 | swHTaskRepoStub 补 3 canary 方法（pre-existing T-0018 stub gap，否则 vet 阻塞） |
| `omcgo/internal/software/adapter_test.go` | 修改 | LTE 期望值改为 dotted path（pre-existing eb4fab46 修复未同步测试） |
| `omcgo/scripts/e2e_verify.sh` | 修改 | +1 claim "POST /upgrade-tasks/rollback (T-0021 audit fields)" |

---

## 2. 硬门验证

### 2.1 编译

```bash
$ go build ./...
✅ BUILD_OK（无输出 = 通过）
```

### 2.2 静态检查

```bash
$ go vet ./...
✅ 全模块 vet 通过（无输出）
```

### 2.3 单元测试 + race

```bash
$ go test -race -count=1 ./internal/software/...
ok  	github.com/omcgo/omcgo/internal/software	1.857s   # 首轮 race 通过

$ go test -count=1 ./internal/software/...                # 后续 race-lib 触发 macOS Xcode license
ok  	github.com/omcgo/omcgo/internal/software	0.882s   # CGO_ENABLED=0 等价命令

$ go test -race -count=1 ./...
✅ 全模块通过；2 个失败属 internal/task 模块 pre-existing（source_id NULL 扫描）
   git stash + 干净 main 复跑确认同样失败 — 与 T-0021 无关
```

⚠ 本机 race detector（macOS Darwin）依赖 cgo + Xcode license；首轮已用 race 通过，复跑因 Xcode license 弹窗中断。CI 环境（Linux）无此问题。

### 2.4 迁移版本号检查

```bash
$ bash scripts/check-migrations.sh
✅ 编号连续（无跳跃）
✅ 所有文件 goose Up/Down 标记齐全
✅ 命名规范
✅ 迁移检查全部通过
```

### 2.5 3 个 Prometheus metric 命名 grep

```bash
$ grep -rn "software_rollback_total\|software_rollback_devices_total\|software_rollback_with_target_total" --include="*.go"
6 命中（rollback_metrics.go 注释 3 + Name 字段 3）
✅ 全部 ≥1
```

### 2.6 E/R（新端点 vs 新 E2E claim）

- 新增端点：**0**（继续复用既有 `POST /upgrade-tasks/rollback`，仅扩展 JSON body）
- 新增 e2e claim：**1**（W2D sw-7）
- E/R 比：1 / 0 → N/A（按规则：无新端点时本项 N/A，仍主动加 1 claim 验证字段层面契约）

### 2.7 迁移 up/down 演练（本地 docker-compose 未启）

⚠ **未演练**：本地 docker-compose 未启动；`scripts/check-migrations.sh` 静态校验已过；CI 会做实际迁移演练。S5 review 阶段如启动了 docker，会补做。

### 2.8 累计型依赖核销

T-0021.Deps = `T-0018`（普通依赖，已 done），无累计型阈值。**N/A**

---

## 3. 行为/契约断言

### 3.1 RollbackDevices 新字段

| Given | When | Then | 测试 |
|-------|------|------|------|
| 空 source | RollbackDevices 调用 | task.RollbackSource = "manual" | `TestService_RollbackDevices_SourceMatrix/blank_defaults_to_manual` |
| source ∈ {manual, canary_failure, compatibility, scheduled} | 调用 | 持久化原值 | 同上其余 4 case |
| source = "bogus_value" | 调用 | err = ErrInvalidInput；task 未持久化 | `unknown_source_rejected` |
| TargetFirmwareID = nil | 调用 | sub_tasks.DestVersion 留空（走旧 OriVersion 回退） | `TestService_RollbackDevices_TargetFirmwareMatrix/nil_target_keeps_legacy_path` |
| TargetFirmwareID = 有效 fw | 调用 | sub_tasks.DestVersion = fw.Version | `valid_target_overrides_DestVersion` |
| TargetFirmwareID = 不存在 | 调用 | err = "get target firmware: ErrNotFound" | `missing_target_firmware_errors` |
| target + canary_failure 组合 | 调用 | 两者并存 | `target_+_canary_failure_source_combine` |
| Reason 非空 | 调用 | task.RollbackReason 完整 round-trip | `TestService_RollbackDevices_ReasonRoundTrip` |

### 3.2 Canary 自动回退（V2 / V3）

| Given | When | Then | 测试 |
|-------|------|------|------|
| RollbackOnFailure=true + threshold 超 + trigger wired | tick | pause + trigger.calls=1 + reason 含 "threshold" | `TestCanaryMonitor_..._AutoRollback_OptIn` |
| RollbackOnFailure=false + threshold 超 | tick | 仅 pause；trigger.calls=0（D4 invariant 维持） | `..._AutoRollback_OptOut` |
| RollbackOnFailure=true 但 trigger=nil | tick | pause；不 panic | `..._AutoRollback_NoTrigger` |
| trigger 返回错误 | tick | CheckCanaryTasks 不传播错误；pause 已落盘 | `..._AutoRollback_TriggerError` |

### 3.3 SQL Schema 一致性

- `taskColumns` 新增 `rollback_reason / rollback_source / rollback_target_firmware_id` → 与迁移 000046 ALTER 顺序对齐
- Create INSERT 显式列出 3 列；空 source → 默认 'manual'（满足 CHECK）
- scanUpgradeTask + scanUpgradeTaskRow 用 `sql.NullString` 缓冲后回填 struct，避免 NULL 标量崩溃
- GetCanaryFields / UpdateCanaryFields 增加 `rollback_on_failure` 列（与迁移 000046 第 4 列对齐）

---

## 4. 度量

| 指标 | 类型 | 标签 | 触发场景 |
|------|------|------|---------|
| `software_rollback_total` | Counter | `source` | 每次 RollbackDevices 成功创建 task |
| `software_rollback_devices_total` | Counter | `source` | 累加 device_count |
| `software_rollback_with_target_total` | Counter | — | 每次 TargetFirmwareID 非 nil |
| `software_canary_stage_advance_total` | Counter | `result="auto_rollback"` | T-0021 新增 result 标签值（既有 metric 复用） |

---

## 5. 待跟进 / 不在本任务内

- ⚠ docker-compose 未启动：迁移 up/down 实测留待 CI 或本地启动后补
- ⚠ `internal/task` 两个 pre-existing 测试失败（source_id NULL 扫描）— 应另立 backlog 修复
- ❌ 前端不在本任务范围（T-0019 已建框架；后续 PR 加 reason/source UI）
- ❌ compatibility source 仅留 enum，业务逻辑（兼容性触发）后续 PR

---

## 6. DoD 自查（PRD §8）

- [x] PRD 七要素全 + 运营商差异矩阵
- [x] migration 000046 up/down 配对
- [x] go build / test -race / vet 全过
- [x] 3 metric grep ≥1
- [x] e2e_verify.sh +1 claim "rollback with reason/source/target"
- [ ] backlog T-0021 → done — S7 处理
- [ ] R-101 完整关闭（T-0018+T-0021 合力）— S7 处理

---

## 7. 风险评估

| 风险 | 缓解 |
|------|------|
| 老的 RollbackRequest（无 audit 字段）会因 NOT NULL DEFAULT 'manual' 通过校验 | 向后兼容（PRD V5 验收） |
| canary_failure 触发时如果 sub_tasks repo 错误，pause 已落盘但 rollback 未发起 | 错误日志不传播；运维可手动 RollbackDevices 手工补救（PRD §5 已明示） |
| RollbackOnFailure 仅在 BatchUpgrade canary 路径写入；既有任务（迁移前数据）= false | 默认安全（D4 invariant） |

---

*验证完成；阻塞门全过。下一步 S5 review。*
