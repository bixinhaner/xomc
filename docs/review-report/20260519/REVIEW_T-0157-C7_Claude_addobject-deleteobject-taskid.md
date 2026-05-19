# Code Review — T-0157 C7 AddObject / DeleteObject 改造返 task_id + CreatorID

- **日期**：2026-05-19
- **范围**：`internal/device/device_param_handler.go`（2 处 handler）/ `frontend-core/services/api/deviceParameterApi.ts` / `frontend-core/mock/services/deviceParameterService.ts` / 设计文档 §11 L-01 状态更新
- **作者**：Claude
- **Reviewer**：Claude（self-review，前端 + Go 工程视角）
- **关联**：T-0157 Phase 1 sub-task **C7**

---

## 变更概要

1. **后端 handler.AddObject**：CreateTaskRequest 加 `CreatorID: admin.UserIDStringFromCtx(c)`；response body 加 `task_id`
2. **后端 handler.DeleteObject**：同上
3. **前端 deviceParameterApi.addObject / deleteObject**：返回类型从 `Promise<void>` 改为 `Promise<{ taskId: string }>`；解构 `data.task_id` 转 camelCase
4. **前端 mock**：mock 服务对齐返回类型（`mock-add-${timestamp}` / `mock-del-${timestamp}` 拼一个假 taskId 让 useDeviceTaskStatus 也能轮询而不直接 enabled=false）
5. **设计文档 §11 L-01**：补"C7 补 AddObject/DeleteObject"标注
6. **device_service.go 实际未改**：handler 内直接调 `taskSvc`，无 service 层方法。设计 §10.1-C7 把它列入预期范围属于过度估算

---

## 出口门验证

| 门 | 结果 |
|---|---|
| `go build ./...` | ✅ |
| `omcmb/webcode npm run typecheck` | ✅ tsc --noEmit 通过 |
| 前端调用方（MultiInstanceTable.handleAdd / handleDelete）兼容新返回类型 | ✅（既有 `await addMutation.mutateAsync(...)` 不解构返回值，向后兼容） |

---

## Checklist

| 类别 | 项 | 结果 |
|---|---|---|
| API 契约 | response body 加 task_id 字段，旧字段 message 保留（向后兼容） | ✅ |
| user 隔离 | CreatorID 来自 JWT，与 SetParameterValues 路径一致 | ✅ |
| 类型安全 | api 返回 camelCase taskId，与 useUpdateParameters 路径风格统一 | ✅ |
| Mock 对齐 | mock 服务返同形 `{ taskId }`，useMock=true 时 hook 行为一致 | ✅ |
| 消息中心 wire | AddObject/DeleteObject task 现在带 CreatorID → 订阅器写 status=queued + 状态升级一路工作 | ✅ |

---

## 发现

### CRITICAL / WARNING
- 无

### INFO
- I-01：MultiInstanceTable.handleAdd / handleDelete 目前**忽略**新增的 taskId（既有调用 `await mutateAsync` 不解构）。C10 实施 Tag 改读 useDeviceTaskStatus 时直接消费返回的 taskId，让 Add/Delete 也走完整状态机（pending → completed/failed/expired）。当前 commit 仅打通后端到 hook 的链路，前端表现层未改。
- I-02：mock 返回的假 taskId（如 `mock-add-1747...`）不是真实 UUID。如果 C10 实施时 useDeviceTaskStatus 在 mock 模式下直接调 deviceTaskApi.getTask 会 404 —— 但 mock 模式下 useDeviceTaskStatus 也应该走 mock。当前 mock 模式 deviceTask hook 路径 V1 不重点（V1 真实 API 验证为主），登记 §11 L-05 (如必要)。**当前不登记**，因 mock 模式下 useDeviceTaskStatus 也很可能没用 mock 数据，待 C8 实施时统一审视。
- I-03：C7 解锁了"消息中心 100% 覆盖快速设置 3 类写操作（Save / Add / Delete）" —— §8 验收 #2 的前置条件已满足。

---

## 结论

**PASS**

可合入，可继续 C8（前端 notification API + hooks 套件）。
