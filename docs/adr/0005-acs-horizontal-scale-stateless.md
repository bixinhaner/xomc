# 0005 — ACS 横向扩展去进程态：会话副作用状态全量接共享 Redis（Option B），不靠会话亲和

## Status

Accepted。决策于 2026-06-11（issue #65）。

> 本决策建立在 ADR 0001（模块化单体 + 独立 ACS 引擎，ACS 可水平扩展）之上：ACS 一直被宣称为「无状态、可水平扩展」，本 ADR 把这句话坐实。

## Context

`omcgo-acs` 是独立部署、可水平扩展的 TR-069 ACS 引擎（ADR 0001）。会话状态（`acs:session:*`）、任务队列（`acs:taskq:*`）、STUN 地址（`acs:stun:*`）早已在 Redis，跨实例共享。但有 **4 类会话副作用状态仍是进程内（process-local）**，在多实例「无亲和（no session affinity）」部署下会漂移 / 泄漏 / 失败：

1. **准入计数**（`internal/acs/admission.go`）：仅 `maxSessions int64` + `atomic.Int64`。→ 每实例各算各的，全局并发上限变成 `N×max`；跨实例 Acquire/Release 不配对，Release 落在没 Acquire 过的实例上把计数下溢成负。
2. **设备孤儿会话表**（`handler.go` 的 `deviceSessions sync.Map`）：记录设备 → 当前 sessionID，用于新 Inform 到达时清理上一个未完成的会话。→ 设备在实例 A 上的孤儿会话，当下一个 Inform 落到实例 B 时读不到，无法清理（槽位泄漏 + 队列剩余命令不续唤）。
3. **ConnectionRequestURL 缓存**（`handler.go` 的 `connReqURLCache sync.Map`）：Inform 上报的 HTTP CR URL，会话后续唤（postSessionWake）做 HTTP 回退用。→ 续唤可能落在非 Inform 的实例上 → 缓存 miss → httpURL 为空 → 纯 HTTP-CR 设备（无 STUN）即时续唤失败。
4. **Digest nonce**（`internal/acs/auth/authenticator.go` 的 `nonces map`）：HTTP Digest 一次性 nonce。→ Challenge 落在 A、Authenticate 落在 B → nonce miss → 401 死循环。（`auth.mode` 默认 `none`，此项为潜在缺陷。）

可选方案：
- **Option A（会话亲和）**：在 nginx/k8s 配 sticky session（按设备 SN / 源 IP 路由到固定实例），让进程态「够用」。代价：负载不均（大设备热点压垮单实例）、实例伸缩 / 滚动重启时亲和键重映射导致状态丢失、与「无状态可水平扩展」的宣称相悖。
- **Option B（全量去进程态）**：把这 4 类状态全部接共享 Redis，ACS 真正无状态。代价：每条状态多一次 Redis 往返；准入热路径（handleInform，高频）需控制开销。

## Decision

选 **Option B（全量去进程态 / shared-storage stateless）**。ACS 已硬依赖 Redis（会话 / 任务 / STUN 都在 Redis，无 Redis 起不来），新增这 4 类状态接 Redis 与既有架构一致，不引入新依赖。

四类状态的 Redis 化设计（键集中在 `redisx.Keys`）：

| # | 状态 | 键 | 数据结构 / 原子操作 |
|---|------|----|--------------------|
| 1 | 准入计数 | `acs:admission:slots` | Sorted Set，member=sessionID、score=过期 unix 秒。**Acquire = 单条 Lua**：`ZREMRANGEBYSCORE(-inf, now)` 驱逐过期 → `ZCARD` → 若 `< max` 则 `ZADD sessionID (now+ttl)` 返回 1，否则 0。**Release = `ZREM sessionID`**。slotTTL（10min）≥ 会话 TTL（5min）+ 余量。 |
| 2 | 设备活跃会话指针 | `acs:device:session:{sn}` | STRING + TTL。**Swap = Lua**（GET 旧值 → SET 新值），返回被顶替的旧 sessionID 供跨实例清理；**CompareAndDelete = Lua**（仅当 GET==本 sessionID 才 DEL），避免误删已被新 Inform 覆盖的指针。 |
| 3 | ConnectionRequestURL | `acs:connreq:url:{sn}` | STRING + TTL（30min），镜像 `acs:stun:{sn}`。 |
| 4 | Digest nonce | `acs:auth:nonce:{nonce}` | `SETEX` 写 + `GETDEL` 原子一次性消费（TTL 5min）。 |

**抽象保持**：`AdmissionController` 仍是接口，`Acquire(ctx, sessionID)` / `Release(ctx, sessionID)` 增加 sessionID 形参使跨实例配对；提供 `localAdmissionController`（进程内，单实例 / 测试）与 `redisAdmissionController`（生产）两实现。`DeviceSessionStore`、`ConnReqURLStore`、`NonceStore` 同样接口优先，Redis / 本地双实现。`cmd/acs/main.go` 在 `inf.Redis != nil` 时注入 Redis 实现，否则退化为进程内实现（单实例语义不变）。

**自愈**：准入槽位与设备会话指针都带 TTL —— 丢失的 Release（实例崩溃）由 `ZREMRANGEBYSCORE` 的过期分回收，全局计数最终自洽，不会被泄漏的槽位永久占满。

**CR-URL 选型（低风险）**：复用设备表 `connection_request_url` 列（与 App 侧 `wakeDevice` 同源）本可省一个 Redis 键，但 ACS 进程刻意不强依赖 device repo（`buildACSPathTranslator` 全程 nil-safe，`acsConnReqSender` 注释明确「HTTP URL 在 ACS handler 上下文不易取得」），把 device repo 接入 Inform 热路径会引入新耦合，风险更高。改用 Redis 键 `acs:connreq:url:{sn}` —— 完全镜像同一个 Inform 循环里已有的 `stun.Store`，无新耦合，是更低风险方案。

**Redis 故障行为（fail-closed）**：准入 Acquire 遇 Redis 错误 → 返回 false（503 拒绝），保守拒绝优于放任全局并发击穿。设备指针 Swap / CR-URL / nonce 遇错只 WARN 不阻塞 InformResponse（这些是优化路径，失败退化为「不清理孤儿 / HTTP 回退拿空 URL / nonce 当未命中」，不破坏主流程）。

## Consequences

**正向**：

- 全局并发上限真正全局（不再 N×max），跨实例 Acquire/Release 严格配对，丢失的 Release 由 TTL 自愈，无下溢、无 false 503。
- 设备孤儿会话可在任意实例的下一个 Inform 被检测清理（槽位释放 + postSessionWake 续唤），不再依赖 Inform 落回同一实例。
- 纯 HTTP-CR 设备在任意实例上续唤都能拿到 CR URL。
- Digest Challenge / Authenticate 可分落不同实例，不再 401 死循环。
- **nginx / k8s 不再需要会话亲和（sticky session）** —— ACS 可任意伸缩 / 滚动重启 / 负载均衡，无状态丢失。

**代价 / 约束**：

- 准入热路径（handleInform）每次 Acquire 多一条 Lua 往返（单条 RTT，毫秒级）；其余状态各多 1 次 Redis 往返。控制在「准入 1 条 Lua / 会话」的预算内，不放大热路径。
- 准入计数从「O(1) 原子读」变成「ZCARD + 驱逐」；高频下 sorted set 体量 = 当前活跃会话数（≤ maxSessions），可控。
- `completeSession(nil)`（无会话上下文）不再释放准入槽位（没有 sessionID 无法配对释放），残留槽位靠 TTL 回收 —— 契约变更，已在测试 `TestCompleteSession_NilSession_DoesNotReleaseAdmissionSlot` 固化。
- 单实例 / 无 Redis 部署退化为进程内实现，行为与改造前一致（向后兼容）。

## 实现位置

- 准入：`internal/acs/admission.go`（`AdmissionController` 接口 + local/redis 双实现 + `admitScript` Lua）。
- 设备会话指针：`internal/acs/device_session_store.go`（`swapScript` / `casDeleteScript` Lua）。
- CR-URL：`internal/acs/connreq_url_store.go`。
- nonce：`internal/acs/auth/nonce_store.go`（`NonceStore` 接口 + memory/redis 双实现）。
- 热路径接线：`internal/acs/handler.go`（handleInform 的 Acquire→Swap→reap、completeSession 的配对 Release + CAS 删除、postSessionWake 的共享 CR-URL 读）。
- 装配：`internal/acs/server.go`（`ServerDeps` 新增 store 字段）+ `cmd/acs/main.go`（`inf.Redis != nil` 注入）。
- 键定义：`internal/core/components/redisx/keys.go`（`ACSAdmissionSlots` / `ACSDeviceSession` / `ACSConnReqURL` / `ACSAuthNonce`）。
