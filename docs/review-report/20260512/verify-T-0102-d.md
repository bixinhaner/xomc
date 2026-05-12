# T-0102-d Verify Report — MML 多帧 SSE 推送

**Date**: 2026-05-12
**Branch**: main
**Author**: Claude
**Type**: feat (F06 / ops + mml)
**Sub-task**: T-0102-d (from T-0102 umbrella — SSE 通道 5 sub-task; 进度 → 4/5)
**PRD**: F06 ops PRD §4.2.3

## Scope

让 MML 任务（多设备 fan-out）在每个 device_task 终态时实时向 executor 用户推送一条 `mml_device_frame` SSE 事件，把"50 设备等全完再看结果"改成"50 帧流式到达 UI"。多帧语义即多设备 fan-out 的逐帧到达。

保持 Q1=B「双通道共存」边界：ops/commands/rpc 走 `internal/ops.SSEHub`，MML 走 `internal/events.MessageHub`（既有 `SSEPublisher` 接口）。两套不混。

## Files Changed

| File | Change | Lines |
|------|--------|-------|
| `omcgo/internal/mml/result_aggregator.go` | +`publishDeviceFrame` 方法（per-device frame 拼装 + 缺 executor / nil hub / GetByID 错全 best-effort 跳过）+ `OnTaskCompleted` 在 IncrementStats 后调一次 publishDeviceFrame | +50 / -0 |
| `omcgo/internal/mml/result_aggregator_test.go` (NEW) | 7 testcase — V1 success frame / V2 failure frame 带 error_message / V3 non-terminal (sent/pending/cancelled) 不发 / V4 nil hub 不 panic / V4b 空 Executor 跳过 / V5 空 SourceID 早返 / V6 GetByID 错仍 increment | +205 |

## Design Notes

- **per-device frame 而非 per-cwmp frame**: 当前 internal/task 是「1 RPC = 1 response」模型，没有真正"半帧推送"概念。SSE 多帧语义落地为「多设备 fan-out 时每设备完成一帧」，UI 看到 device-by-device 进度。
- **publishDeviceFrame 在 IncrementStats 后 / finalizeIfComplete 前**: 顺序保证 frame 到达时统计已增；frame 失败不阻塞 finalize（best-effort）。
- **payload 含 device_task_id**: UI 可以以此 key 去重 / 关联后续重试帧。
- **command_index + device_index**: 多命令 × 多设备的二维索引保留，方便 UI 按矩阵渲染。
- **Q1=B 双通道边界**: MML 用 `events.MessageHub.PublishSimple(userID, ...)` 走 per-user 通道；ops 用 `ops.SSEHub.Publish(channel, ...)` 走 per-channel。两个独立 hub，不共享代码。
- **frame 失败不阻塞主流程**: GetByID / Marshal / Publish 失败全部走 logger.Warn，**绝不返回错** — 这是 SSE fan-out 的语义边界（best-effort，与持久化层正交）。

## Test Plan

| Test | Path | Asserts |
|------|------|---------|
| V1 PublishesDeviceFrame_Success | dt.Status=completed + result | IncrementStats(+1,0) + 1 mml_device_frame 事件到 alice / payload 含 device_task_id+device_sn+method+status+result+command_index+device_index / 无 error_message |
| V2 PublishesDeviceFrame_Failure | dt.Status=failed + error | IncrementStats(0,+1) + frame 含 error_message |
| V3 NonTerminal_NoFrame | dt.Status=sent/pending/cancelled 各 1 | 0 increment 0 frame |
| V4 NilHub_NoPanic | hub=nil | 不 panic |
| V4b EmptyExecutor_SkipsFrame | Executor="" | 跳 publish 不 panic |
| V5 EmptySourceID | SourceID="" | 早返 0 increment 0 event |
| V6 GetByIDError_StillIncrements | GetByID 错 | IncrementStats 仍执行 + 0 frame（best-effort） |

**Race**: `go test -race ./internal/mml/ -count=1` PASS 1.079s

## Commands

```bash
$ go build ./...                                                       # ✓
$ go test -race ./internal/mml/ -run TestAggregator -v -count=1        # 7 PASS / 0 FAIL
$ go test -race ./internal/mml/ -count=1                               # ok 1.079s
```

## Out of Scope

- **真正的「设备返多帧」（同一 cwmp 多次响应）**: TR-069 spec 不支持单 RPC 多响应，CPE 协议层就是「1 请求 1 响应」。如果未来要做「半帧」（如长 GetParameterValues 分块），需要 ACS 层 chunking 协议设计，超出本任务。
- **frontend 订阅 + 渲染**: 本任务只做后端推送；T-0103-f CommandManagement UI（已 proposed）会消费 frame 渲染流式输出。
- **frame 持久化 / 回放**: events.MessageStore 已有持久化基础设施但 MML 没接入；frame 实时到达即可，离线回放未来需求再做。

## DoD Checklist

- [x] `go build ./...` 通过
- [x] `go test -race ./internal/mml/...` 全绿（7 新 testcase + 既有 mml 测试）
- [x] 无新增 TODO/FIXME/panic
- [x] 无新增 `if carrier == "..."` 硬编码
- [x] 无新增 `any`/`interface{}`（`map[string]interface{}` 是既有 payload 类型）
- [x] SSE 推送 best-effort 不影响 stats 一致性
- [x] Q1=B 双通道边界守住（MML hub / ops hub 不混）
- [x] payload 字段 device_task_id / device_sn / method / status / command_index / device_index 6 字段供 UI 矩阵渲染
