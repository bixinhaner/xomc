# ACS 引擎压测就绪修复方案

> 日期：2026-03-10
> 状态：**全部 5 项修复已完成**

---

## Context

ACS 引擎 32 个 TR069 协议流程已补全，但深度审计发现 **3 个致命 bug + 2 个高危问题**，会导致压测数据完全失真。本修复方案聚焦让压测结果具备参考价值的最小必要改动。

---

## Fix 1: Admission Control + Metrics 生命周期修正 [CRITICAL] ✅

**问题**: `handler.go:108-116` — `defer h.admission.Release()` 和 `defer h.metrics.ActiveSessions.Dec()` 在 handleInform 返回时就释放，而非整个会话结束时。导致并发限制 10000 形同虚设。

**方案**: 将 admission/metrics 与 connSessions 绑定，在 session 结束时（handleEmpty/handleRPCResponse 的 Complete 路径）统一释放。

**改动**:
- 移除 handleInform 中的 `defer h.admission.Release()` 和 `defer h.metrics.ActiveSessions.Dec()`
- 新增 `completeSession(ctx, deviceSN, remoteAddr, session)` 私有方法
- handleEmpty 和 handleRPCResponse 中所有 session complete 路径调用 `completeSession()`

---

## Fix 2: connSessions 内存泄漏 — 后台清理 [CRITICAL] ✅

**问题**: `connSessions sync.Map` 无上限无 TTL，TCP 异常断开时 entry 永远残留。

**改动**:
- 新类型 `connSessionEntry{DeviceSN, CreatedAt}`
- `startSessionReaper(30s, 5min)` 后台 goroutine
- `NewACSServer()` 中自动启动 reaper

---

## Fix 3: 补全关键 Metrics [HIGH] ✅

**改动**:
- `SessionDuration.Observe()` in completeSession + reaper
- `RPCDuration.WithLabelValues().Observe()` in handleRPCResponse
- `RPCErrorsTotal.WithLabelValues().Inc()` on build failure

---

## Fix 4: 命令队列递归 Pop → 迭代 [MEDIUM] ✅

**改动**: `Pop()` 改为 `for i := 0; i < 100; i++` 循环

---

## Fix 5: 压测工具升级 — 完整会话模拟 [CRITICAL] ✅

**改动**: 新增 `-full-session` flag (默认 true)，模拟完整 TR069 三步会话

---

## 文件改动统计

| 文件 | 改造前 | 改造后 | 变化 |
|------|--------|--------|------|
| `internal/acs/handler.go` | 474 | 529 | +55 |
| `internal/acs/server.go` | 124 | 127 | +3 |
| `internal/acs/cmdqueue/queue.go` | 130 | 134 | +4 |
| `scripts/loadtest/main.go` | 328 | 661 | +333 |
| **总计** | **1056** | **1451** | **+395** |
