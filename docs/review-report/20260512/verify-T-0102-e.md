# T-0102-e Verify Report — per-device 超时 + 重试 + 限流

**Date**: 2026-05-12
**Branch**: main
**Author**: Claude
**Type**: feat (F06 / ops)
**Sub-task**: T-0102-e (from T-0102 umbrella — SSE 通道 5 sub-task; 进度 → 5/5 收官)
**PRD**: F06 ops PRD §4.2.3

## Scope

把 T-0102-c 已建的 inline RPC fan-out 派发路径加上 PRD §4.2.3 要求的三件套：

1. **单命令默认 60s 超时** — CreateTaskRequest.ExpiresIn = 60s（device queue 层强制；CPE 在该时长内未应答则 task 状态 → expired）。
2. **per-device 限流** — 同一 device_sn 短时间被同任务多次轰炸时排队（5 RPC/sec/device 默认）。
3. **重试-with-backoff** — 入队失败（transient DB/Redis 抖动）自动重试 3 次，指数退避 2s/4s 钳到 5s 上限。

## Files Changed

| File | Change | Lines |
|------|--------|-------|
| `omcgo/internal/ops/concurrency.go` | +`DefaultPerDeviceRPCRateLimit=5`, `ConcurrencyLimiter.devLimiters sync.Map` + `perDeviceRate`, `SetPerDeviceRate`, `WaitForDevice(ctx, sn)`, `DeviceLimiterCount` (监控) | +50 / -5 |
| `omcgo/internal/ops/service_ext.go` | TaskExecutor +`limiter` 字段 + `SetLimiter` setter；`+DefaultPerCmdTimeoutSeconds=60`, `DefaultEnqueueMaxRetries=3`；`dispatchInlineRPC` 每设备前调 `limiter.WaitForDevice` + `CreateTaskRequest.ExpiresIn=60` + 调新增 `enqueueWithRetry` 重试 + `recordEnqueueFailure` helper 写 failed exec row | +75 / -10 |
| `omcgo/internal/ops/dispatcher_retry_test.go` (NEW) | 8 testcase — V1 ExpiresIn=60 / V2 retry 恢复 (failFirst=2 → 3 attempts ≥6s backoff) / V3 retry 用尽含 attempt count 错误 / V4 ctx 取消短路 backoff / V5 per-device limiter 同 SN 串行 / V6 不同 SN 独立 / V7 SetPerDeviceRate clamp / V8 空 SN no-op | +220 |
| `omcgo/internal/ops/rpc_dispatcher_test.go` | T-0102-c PartialFailure 更新断言 — SN-bad 现触发 retry 3 次 → enqueuer.requests=5 (1+3+1) + 错误消息含 "enqueue after" attempt count；execution 行仍 1 行/设备（3 总） | +9 / -4 |
| `omcgo/cmd/app/provider/modules.go` | DI 加 `opsExecutor.SetLimiter(ops.NewConcurrencyLimiter(0, 0))` 启用默认 5/sec/device + 3 enqueue retries | +4 |

## Design Notes

- **per-device limiter 懒创建 + sync.Map 缓存复用** — 不做主动 GC，rate.Limiter 约 48 字节体积小，dispatcher 调用面 device_sn 集合在一段时间内基本稳定；DeviceLimiterCount 监控可观测内存占用，需要时再加 LRU 是 future 议题。
- **SetPerDeviceRate setter 模式** — 不破 NewConcurrencyLimiter 签名；与 SetEnqueuer/SetSSEHub/SetLimiter 三个 setter 一致的注入风格。
- **enqueueWithRetry 内置 backoff** — 2s/4s 指数退避钳到 5s 上限，与 PolicyDecider (T-0101-f) 的 30s 上限不同：dispatcher level retry 是 transient 入队错（DB/Redis），快速失败比慢退避更适合；PolicyDecider 是 task level retry 决策器（步骤/RPC 重试）。两层职责不同不复用。
- **per-device 限流被 ctx 取消** → 写 failed exec row 继续推下个设备（per-device 失败隔离 PRD §4.2.3）；不通过 retry 因为 ctx cancel 是终态语义。
- **ExpiresIn = 60s 是入队侧承诺** — 真正的"批量场景设备级独立超时"由 device queue 层 task.IsExpired() + ACS engine PopTask + Connection Request 链路联合实现（既有基础设施）。本任务只在入队点把超时承诺写进去。
- **重试只针对入队失败，不针对设备执行失败** — 设备 RPC 失败由 ACS engine + task.Task.RetryCount 自行处理（既有路径）；ops 层不重复造轮子。
- **公共构造函数 NewTaskExecutor 签名向后兼容** — limiter 通过 setter 注入；nil 时跳过 per-device 限流，retry 仍生效（不依赖 limiter）；T-0101 umbrella 既有 template-driven 流程不受影响。

## Test Plan

| Test | 路径 | 断言 |
|------|------|------|
| V1 PerCmdTimeoutInExpiresIn | 1 device | CreateTaskRequest.ExpiresIn == 60 |
| V2 RetryRecoversFromTransientFailure | failFirst=2 | enq.count=3 / exec row status=running / elapsed ≥ 6s (2s+4s backoff) |
| V3 RetryExhaustedRecordsFailure | failFirst=99 | exactly DefaultEnqueueMaxRetries 尝试 + failed row + "enqueue after" |
| V4 RetryCtxCancelExitsEarly | ctx 500ms 超时 | elapsed ≤ 3s / failed row 含 "cancelled" |
| V5 PerDeviceRateLimit | 同 SN 两次 + perDeviceRate=1 | 2 成功 / elapsed ≥ 800ms / DeviceLimiterCount=1 |
| V6 PerDeviceLimiterIndependent | 3 不同 SN + perDeviceRate=1 | 3 成功 / elapsed < 500ms / DeviceLimiterCount=3 |
| V7 SetPerDeviceRate_DefaultsOnZero | 0/-5/20 | clamp 到 default / 接受正值 |
| V8 WaitForDevice_EmptySN_NoOp | sn="" | nil err + DeviceLimiterCount=0 |

**Race**: `go test -race ./internal/ops/ -count=1` PASS 21.880s（含 T-0102-c PartialFailure 更新后通过）。

## Commands

```bash
$ go build ./...                                                              # ✓
$ go test -race ./internal/ops/ -run "DispatchInline|ConcurrencyLimiter" -v   # 8 PASS
$ go test -race ./internal/ops/ -count=1                                      # ok 21.880s (全 ops 含 T-0102-c 更新)
```

## Out of Scope

- **per-device limiter LRU GC** — 当前 sync.Map 长期持有，DeviceLimiterCount 监控就位足够；满 10 万设备时再迁 LRU 是 future。
- **重试只针对入队失败** — 设备 RPC 失败由 ACS engine + task.Task.RetryCount 自处理。
- **dispatcher-level concurrency (max=20)** — ConcurrencyLimiter.AcquireTask/ReleaseTask 还未在 dispatchInlineRPC 调用面挂上；本任务聚焦 per-device 限流；上层 task 并发限流可在 future T-0102 后续 / 整体 dispatcher 大改时加。
- **WaitRPC 全局节流** — 同上，未挂上 inline RPC 路径；T-0101-c 已建好基建，下次大改时启用。

## DoD Checklist

- [x] `go build ./...` 通过
- [x] `go test -race ./internal/ops/...` 全绿（8 新 testcase + 既有 ops 含更新的 PartialFailure）
- [x] 无新增 TODO/FIXME/panic
- [x] 无新增 `if carrier == "..."` 硬编码
- [x] 无新增 `any`/`interface{}`
- [x] 公共构造函数签名向后兼容（setter 注入）
- [x] PRD §4.2.3 三件套覆盖：单命令 60s 超时 / per-device 限流 / batch 设备级独立
- [x] 重试退避钳到 5s 上限防长尾阻塞
- [x] ctx cancel 短路 backoff 防 paused/cancelled task 浪费重试预算
- [x] T-0102 umbrella 5 sub-task 全闭（a 接口 + b 入队 + c RPC 派发 + d MML 多帧 + e 超时/重试/限流）
