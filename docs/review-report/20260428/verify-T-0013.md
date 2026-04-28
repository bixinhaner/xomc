# T-0013 Verify Report — F08 SNMP Trap 骨架

**任务**：T-0013 F08 SNMP Trap 骨架（PRD + 设计 + 骨架代码不接生产）
**Worktree**：`.claude/worktrees/agent-t0013`（branch `worktree-agent-t0013`）
**日期**：2026-04-28
**范围**：S0 PRD + S2 设计 + S3 骨架代码（**不接** alarm.Engine / Carrier / DI / router / migration）

---

## 1. 交付物清单

### 1.1 PRD（S0）+ 设计备忘（S2）

| 路径 | 行数 | 状态 |
|------|------|------|
| `docs/project/prd/F08-oss-protocol.md` | 365 | NEW |

PRD 七要素 grep 验证（所有标题均存在）：

```
## 1. 业务背景（Why）
## 2. 用户故事（Who / What）
## 3. 验收标准（Given/When/Then）          — 4 条 AC（v2c 基本可用 / 多 OSS 故障隔离 / 禁用与热加载 / v3 authPriv）
## 4. 运营商差异矩阵                        — CMCC / CTCC / CUCC 三列均填充（8 行差异点）
## 5. 非目标（Non-Goals）                   — 含骨架阶段不做项 6 条
## 6. 依赖                                  — 阻塞项 / 被阻塞项 / 外部依赖三块齐
## 7. 度量（如何证明上线成功）              — 5 项指标 + 反例监控 4 条
## 8. 实施要点（非规范性，供参考）
## 9. 审批
## 10. 变更记录
## 设计备忘（S2 设计 — T-0013 骨架阶段）   — 6 节（D1 数据模型 / D2 接口 / D3 PDU / D4 Carrier 适配点 / D5 依赖图 / D6 测试覆盖）
```

### 1.2 骨架代码（S3）

| 路径 | 行数 | 用途 |
|------|------|------|
| `omcgo/internal/northbound/snmp/doc.go` | 44 | package 注释 + 隔离契约 |
| `omcgo/internal/northbound/snmp/types.go` | 153 | TrapTarget / SNMPVersion / Variable / SendResult / AlarmEvent / AlarmMapper |
| `omcgo/internal/northbound/snmp/oid.go` | 32 | OMC 私有 OID 占位常量 + TODO(T-0017) |
| `omcgo/internal/northbound/snmp/mapper.go` | 93 | defaultMapper（CMCC 占位）+ severityToInt |
| `omcgo/internal/northbound/snmp/sender.go` | 255 | Sender 接口 + GoSNMPSender + 验证 + buildPDUs + v3 USM |
| `omcgo/internal/northbound/snmp/registry.go` | 176 | TargetRegistry 接口 + InMemoryRegistry |
| `omcgo/internal/northbound/snmp/engine.go` | 182 | Engine + Process（多 target 并发 fan-out） |
| `omcgo/internal/northbound/snmp/sender_test.go` | 284 | 11 个测试函数（含 table-driven） |
| `omcgo/internal/northbound/snmp/registry_test.go` | 148 | 5 个测试函数 |
| `omcgo/internal/northbound/snmp/engine_test.go` | 236 | 11 个测试函数（mockSender / failingMapper） |

**新建文件 = 10 个**（章程要求 ≥ 6 个，超额 4 个用于职责拆分）。
**总行数 = 1603**（snmp 包），单文件最大 284 < 800 章程上限。

### 1.3 依赖变更

`omcgo/go.mod` / `omcgo/go.sum`：
- 新增 `github.com/gosnmp/gosnmp v1.43.2`（直接）
- 新增 `github.com/benbjohnson/clock v1.1.0`（gosnmp 间接传递依赖）

无其他依赖变更。

---

## 2. Pass 自验证据

### 2.1 编译

```
$ go build ./internal/northbound/snmp/...
（无输出 = 通过）

$ go build ./...
（无输出 = 整个 omcgo 模块仍可编译，未破坏既有代码）
```

### 2.2 vet

```
$ go vet ./internal/northbound/snmp/...
（无输出 = 0 warning）
```

### 2.3 race + count=1 测试

```
$ go test -race -count=1 ./internal/northbound/snmp/...
ok  	github.com/omcgo/omcgo/internal/northbound/snmp	1.660s
```

测试统计（`go test -v` 计数）：
- 测试函数（顶层 Test*）：**27 个**
- 子测试（包含 t.Run / table-driven 子用例）：合计 **50 个 RUN** 全部 PASS
- **章程要求 ≥ 5 测例，实际 50（10 倍于阈值）**

测试覆盖率：
```
$ go test -cover -count=1 ./internal/northbound/snmp/...
ok  	github.com/omcgo/omcgo/internal/northbound/snmp	0.654s	coverage: 82.1% of statements
```

**覆盖率 82.1% ≥ 章程 80% 要求**。未覆盖部分主要为 GoSNMPSender.Send 真实 UDP I/O 路径（需要 SNMP 收信端，留 T-0017 联调验证）。

### 2.4 包依赖隔离证据

`go list -f '{{ join .Imports "\n" }}'` 输出（生产代码 imports）：

```
context
errors
fmt
github.com/google/uuid
github.com/gosnmp/gosnmp
go.uber.org/zap
sort
strings
sync
time
```

测试代码 imports 增量：

```
github.com/stretchr/testify/assert
github.com/stretchr/testify/require
go.uber.org/zap/zaptest
sync/atomic
testing
```

`go list -deps ./internal/northbound/snmp` 中 grep 禁用包：

```
$ go list -deps ./internal/northbound/snmp | grep -E "internal/(alarm|carrier|core/event|core/appconfig|northbound/push)"
（无输出 = 0 个传递依赖触碰禁区）
```

**依赖隔离契约 100% 满足**：
- 仅依赖 stdlib + gosnmp + zap + uuid（章程允许）
- 不引用 internal/alarm、internal/carrier、internal/core/event、internal/core/appconfig、internal/northbound/push、cmd/

PRD / 注释中的"internal/alarm"、"internal/carrier"等字样仅出现在**注释**里（指向 T-0017 阶段的接入点说明），生产 import 表中**零依赖**。

---

## 3. 严禁项审计（git status 校验）

```
$ git status
On branch worktree-agent-t0013
Changes not staged for commit:
	modified:   omcgo/go.mod                               ← 仅 gosnmp 增项（允许）
	modified:   omcgo/go.sum                               ← 同上（允许）

Untracked files:
	.wave-progress.log                                     ← 心跳协议（允许）
	docs/project/prd/F08-oss-protocol.md                   ← 允许
	omcgo/internal/northbound/snmp/                        ← 允许（新包）
```

**严禁项逐条核对**：

| 严禁项 | 状态 |
|--------|------|
| ⛔ 不动 `omcgo/cmd/app/provider/*` | ✅ 未触碰 |
| ⛔ 不动 `omcgo/cmd/app/router/*` | ✅ 未触碰 |
| ⛔ 不动 `omcgo/cmd/acs/*` `omcgo/cmd/worker/*` | ✅ 未触碰 |
| ⛔ 不动 `omcgo/internal/alarm/*` | ✅ 未触碰 |
| ⛔ 不动 `omcgo/internal/carrier/*` | ✅ 未触碰 |
| ⛔ 不动 `migrations/` | ✅ 未触碰 |
| ⛔ 不引入 gosnmp 之外的新依赖 | ✅ 仅 gosnmp（+ 其传递依赖 benbjohnson/clock，gosnmp 内部要求） |
| ⛔ 不动 `docs/project/backlog.md` | ✅ 未触碰 |
| ⛔ 不动 `docs/methodology/` | ✅ 未触碰 |
| ⛔ 不动 `docs/project/release-gate.md` | ✅ 未触碰 |

**关于 gosnmp 间接依赖 `benbjohnson/clock`**：这是 gosnmp 内部用于测试时间抽象的 indirect dep，由 `go mod tidy` 自动写入 go.sum。它**不出现在 snmp 包的 import 表**，仅在 gosnmp 自身代码路径里。属于"使用 gosnmp 的合理代价"，未违反"不引入除 gosnmp 之外的新依赖"的精神（即"我们这边只主动引入 gosnmp"）。

---

## 4. 关键设计决策摘要

1. **Sender 接口 + GoSNMPSender 实现**：方便 mock，T-0017 接入时无须改 Engine
2. **TargetRegistry 接口 + InMemoryRegistry**：T-0017 直接替换 PG-backed 实现，Engine 零改动
3. **AlarmMapper 接口 + defaultMapper**：T-0017 用 Carrier-aware mapper 通过 `WithMapper` 替换，避免动 Carrier 接口
4. **AlarmEvent 私有 stub 类型**：保证骨架不依赖 internal/alarm；T-0017 接入时由消费者 adapter 做 alarm.Event → snmp.AlarmEvent 转换
5. **多 target 并发 + 失败隔离**：每个 target 独立 goroutine + bounded ctx，故障 OSS 不阻塞健康 OSS
6. **敏感字段红线**：`TrapTarget.String()` 不渲染 community / authPassword / privPassword，并在 sender_test.go 写测试守护

---

## 5. 后续阶段（不在本 sub-agent 范围）

- **T-0017 接入**（独立 PR）：
  - 扩展 `internal/carrier/Carrier` 增加 `MapAlarmToTrapPDU` 方法
  - cmcc/ctcc/cucc 各自实现 carrier-aware mapper 并通过 `snmp.WithMapper` 注入 Engine
  - 新建 migration `000NNN_add_snmp_trap_targets.sql`
  - 新建 PG-backed `*PgRegistry` 实现 `TargetRegistry` 接口
  - 在 `cmd/app/provider/snmp.go` 完成 DI
  - 在 `cmd/app/router/router.go` 挂 admin REST API（CRUD + 测试 trap）
  - 订阅 `alarm.Engine` 输出（同进程）+ NATS subject `oss.alarm.forward`（多实例）
- **T-0020 可靠性**（更后 PR）：
  - 与 northbound/push 共享 outbox 抽象（持久化）
  - 重试策略 + 熔断 + Prometheus 告警规则

---

## 6. 结论

**PASS** — 全部 Pass 标准达成：
- ✅ PRD 七要素 + 设计备忘齐全
- ✅ 运营商差异矩阵 CMCC/CTCC/CUCC 三列充实（8 行差异点）
- ✅ 骨架代码 ≥ 6 文件（实际 10 文件 + 4 拆分）
- ✅ `go build` / `go vet` / `go test -race -count=1` 全过
- ✅ 50 测例（≥ 5 阈值），覆盖率 82.1%（≥ 80% 阈值）
- ✅ 包依赖隔离契约 100% 满足（imports 仅 stdlib + gosnmp + zap + uuid）
- ✅ 严禁项 10/10 全部未触碰
- ✅ 最小依赖增量（仅 gosnmp + 其传递 clock）
