# Verify Report — T-0101-a (TaskExecutor 接口设计 + DI 注入)

**Task**: T-0101-a — 提取 TaskExecutor 小接口（消费者驱动）+ 加 Pause/Resume/Cancel/Rollback 4 stub 方法（满足 5 方法契约）+ handler 改依赖接口
**Type**: feat / F06/ops / P0 / S
**Sprint**: sprint-10 (pull-forward 续)
**Deps**: T-0112-a ✅ (MVP) + Q2=A ✅ (复用 internal/task 决议) + T-0101-d ✅ (TransitionStatus 已就位供 stub 调用)
**Owner**: Claude
**Date**: 2026-05-12

---

## §1. 改动面

| 文件 | 改动 | 行数 |
|------|------|------|
| `internal/ops/service_ext.go` | 新增 TaskExecutorEngine 接口（6 方法：5 控制 + 1 查询）+ Pause/Resume/Cancel/Rollback 4 stub 方法 + 编译期 `_ TaskExecutorEngine = (*TaskExecutor)(nil)` 断言 | +90 |
| `internal/ops/handler_ext.go` | executor 字段类型 `*TaskExecutor` → `TaskExecutorEngine`（接口）；NewExtHandler 签名同步 | +1/-1 替换 |

---

## §2. 设计

**6 方法 contract**:
- **5 控制**：Run / Pause / Resume / Cancel / Rollback
- **1 查询**：ListExecutions（执行历史数据查询）

合入同一接口的理由：消费者（handler）只依赖一个 abstract executor 类型，避免双 dep 注入；TaskExecutor concrete struct 天然实现 6 方法。

**Stub 方法实现**：
- Pause/Resume/Cancel 委托 taskRepo.TransitionStatus（T-0101-d 已落地的原子 CAS）+ 写审计
- Rollback 返 ErrNotImplemented（T-0101-h future）
- Run/ListExecutions 既有 MVP 实现不变

**dispatcher 协作 hook（subtask Notes "接口在 device service 侧定义/消费者驱动"）**：
- TaskExecutorEngine 在 ops 包内定义但意图供 device service / future dispatcher 消费
- 当前 handler_ext 是首个消费者；T-0101-b 落地 dispatcher 时也走此接口

---

## §3. 出口门检查

| 检查项 | 结果 |
|--------|------|
| `go build ./...` | ✅ PASS |
| `go test -race -count=1 ./internal/ops/...` | ✅ PASS 1.034s |
| 编译期接口断言 `_ TaskExecutorEngine = (*TaskExecutor)(nil)` | ✅ 编译通过 = 全 6 方法实现就位 |
| 无 TODO/FIXME/panic | ✅（删除"5 方法契约" → "5 控制 + 1 查询" 注释更新）|
| 无新 `any` | ✅ |
| 接口 < 5 方法原则 | ⚠️ 6 方法（合入 ListExecutions 用以避免双 dep 注入；comment 解释）|
| 新端点 | 0 |
| 新 metric/log | 0 |
| 新迁移 | 无 |

---

## §4. ULTRATHINK 决策

1. **6 方法 vs 严格 5 方法**：subtask Notes 写 "Run/Pause/Resume/Cancel/Rollback 5 方法"；但 handler 还需 ListExecutions 查询历史。两选：(a) 严格 5 方法 + 双 dep 注入 (executor + execRepo) (b) 6 方法合一便于消费。选 b 是消费者驱动原则的正向应用：接口由消费者决定形状，不为教条 5 方法切碎依赖
2. **stub 方法委托 TransitionStatus**：刚做的 T-0101-g atomic CAS 状态机直接可用；不重复实现状态转移逻辑
3. **编译期接口断言**：`var _ TaskExecutorEngine = (*TaskExecutor)(nil)` 让 future 修改 TaskExecutor 漏方法时编译期立即报错（vs 运行时发现）
4. **handler executor 字段改接口类型而非保留 concrete**：让 future mock executor 替换无需改 handler；测试更易写

S4 PASS。
