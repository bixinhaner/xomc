# Code Review — T-0164 收尾 G8-Gap-3 残尾闭环（HeartbeatInterval 可配置）

| 字段 | 值 |
|------|----|
| Commit | `300ae4a1` |
| Author | kevin |
| Branch | draft/pm-kpi-impl |
| Backlog | T-0164 |
| Scope | core (internal/core/asyncjob + cmd/worker) |
| Reviewer | Claude Opus 4.7 |
| Date | 2026-05-25 |
| Conclusion | **PASS**（无 CRITICAL，1 个低风险 WARNING 已说明） |

---

## 1. 变更概述

### 1.1 文件清单

| 文件 | 变更类型 | 行数 |
|------|---------|------|
| `omcgo/internal/core/asyncjob/model.go` | const → var + 注释 | +3 / -1 |
| `omcgo/cmd/worker/aggregator.go` | 扩展 `loadAsyncJobThresholds` 读 heartbeat_interval_seconds + 启动日志加 heartbeat_interval 字段 + 注释更新 | +13 / -8 |
| `docs/project/plan-T-0164-followup-gaps.md` | §0 / §5 / 终态确认段同步：38 项 100% done 终态、G8-Gap-3 由 ⚠️ partial 改 ✅、§5 末尾"上下文窗口预警"替换为"终态核实"段（含 G4/G7-Gap-9/Cross-Gap/Prometheus hooks 的代码核实证据） | +18 / -14 |

### 1.2 变更目标

收尾 T-0164 G8-Gap-3 唯一残尾（HeartbeatInterval 仍硬编码）：
- 把 `asyncjob.HeartbeatInterval` 由 `const` 改 `var`
- `cmd/worker/aggregator.go:loadAsyncJobThresholds` 启动期从 `sys_configs.asyncjob.heartbeat_interval_seconds` 读值写回包级 var
- 对齐既有 `sweeper_interval_seconds` / `zombie_threshold_seconds` 注入模式
- **不做** EventBus 热重载（按用户简化偏好放弃；改值需重启 worker 生效）

---

## 2. 审查结果

### CRITICAL

无。

### WARNING

#### W-1（低风险，已规避）：跨包写入包级 var

- **位置**：`cmd/worker/aggregator.go:419` `asyncjob.HeartbeatInterval = ...`
- **现象**：外部包写入 `asyncjob` 包的导出 var，破坏封装。
- **风险评估**：
  - 写入时点：`startPMAggregatorPipeline` 启动期单次，发生在所有 worker goroutine 启动 **之前**
  - 读取时点：`asyncjob/runner.go:128 time.NewTicker(HeartbeatInterval)`，发生在 worker goroutine 跑任务时
  - 启动期单线程写 + 运行期只读，**无 data race**
- **缓解措施**：`model.go:75-77` 注释明确"启动期可通过 sys_configs 覆盖；改值需重启 worker 生效（NewTicker 在 Runner.runOnce 创建时锁定当前值）"，使用语义已文档化
- **替代方案考虑**：用 `atomic.Int64` + getter；本次未采用，因增加复杂度且不解决"已跑任务持有的 ticker 周期不变"问题（要解决得加 ticker.Reset 通道，与"v2 留 EventBus 热重载"等价，超出本次范围）

### INFO

#### I-1：缺单元测试覆盖

- `loadAsyncJobThresholds` 是 `cmd/worker` 内的命令式装配函数，与既有 sweeper/zombie 注入路径同样未覆盖单元测试
- 接受现状（保持一致性 > 增加测试），实际验证由 docker-deploy 后看 worker 启动日志 `asyncjob thresholds loaded from sys_configs heartbeat_interval=...` 完成

#### I-2：日志字段补全

- 启动日志 `asyncjob sweeper started` 新增 `heartbeat_interval` 字段（已加）
- 启动日志 `asyncjob thresholds loaded from sys_configs` 新增 `heartbeat_interval` 字段（已加）
- 可观测性闭环 ✅

---

## 3. 检查项逐项

### Go 后端

| 项 | 结论 | 说明 |
|----|------|------|
| 命名规范 | ✅ | 导出 var PascalCase 保持 |
| 错误处理 | ✅ | `readSysConfigInt` 失败 fallback 走默认值（与既有逻辑对称） |
| SQL 安全 | ✅ | 复用既有 `PgSysConfigRepository.GetByKey`（Squirrel 参数化） |
| 运营商硬编码 | N/A | 无运营商分支 |
| 认证 | N/A | 无对外接口变更 |
| 资源泄漏 | ✅ | 无新增连接 / goroutine |
| 测试覆盖 | INFO I-1 | 与既有装配代码一致，未单测 |

### 通用

| 项 | 结论 | 说明 |
|----|------|------|
| 代码重复 | ✅ | 复用 `readSysConfigInt` helper |
| 硬编码 | ✅ | 默认值 30s 仍在 `model.go` 单点定义 |
| 日志质量 | ✅ | 启动日志带 heartbeat_interval 字段 |
| 文档同步 | ✅ | `plan-T-0164-followup-gaps.md` 同 commit 更新 |

---

## 4. 影响范围

| 维度 | 评估 |
|------|------|
| 默认行为 | **不变**（默认 30s 与改前一致） |
| 运维 | 可通过 `UPDATE sys_configs SET value='60' WHERE category='asyncjob' AND key='heartbeat_interval_seconds'` + 重启 worker 调整 |
| migration | **无新增**（seed 000166 早已 seed 该 key，本次只是 Go 代码消费） |
| 多 worker 实例 | 每个实例独立读 sys_configs；同步靠运维操作一致性，与 sweeper/zombie 同模式 |
| 测试 | `go test ./internal/core/asyncjob/... ./cmd/worker/...` PASS（无新增测试，既有测试不受影响） |
| 编译 | `go build ./...` PASS |

---

## 5. 结论

**PASS** — 可合入。

T-0164 收尾 38 项 100% done 状态确认（见 `plan-T-0164-followup-gaps.md §0 + §5 终态核实段`）。下一步：docker 全栈部署 + 真机端到端验证。
