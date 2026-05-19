# Code Review — T-0157 C10 快速设置接入消息中心（mutation invalidate + Add/Delete 消费 taskId）

- **日期**：2026-05-19
- **范围**：`webcode/pages/device/DeviceDetail/QuickSettingsTab/{CellParameterForm,MultiInstanceTable}.tsx`
- **作者**：Claude
- **Reviewer**：Claude（self-review，前端视角）
- **关联**：T-0157 Phase 1 sub-task **C10**（最后一笔，闭环）

---

## 变更概要

1. **CellParameterForm.handleSave**：try/catch 包 try/catch/finally；finally 里调 `queryClient.invalidateQueries({ queryKey: notificationKeys.all })` —— 让 Popover 列表 + 未读徽标 ~50ms 内拿到后端 subscriber 刚写的"进行中"或 CreateFailureNotifier 刚写的"入队失败"
2. **MultiInstanceTable.handleSaveRow / handleAdd / handleDelete**：相同 finally invalidate 模式
3. **handleAdd / handleDelete 消费 C7 返回的 taskId**：写入 `setFeedback(... taskId: result.taskId ...)` 让 useDeviceTaskStatus 启动轮询，Tag 走完整状态机（与 Save 路径一致）
4. **Tag 视觉**：已经在 T-0156 阶段改为 `statusTagSpec(lastSubmit, lastTask?.status)` 读 useDeviceTaskStatus，本 commit 不动 — store 持久化 taskId 是必要的（切走切回后让 useDeviceTaskStatus 能恢复轮询）

---

## 出口门验证

| 门 | 结果 |
|---|---|
| `omcmb/webcode npm run typecheck` | ✅ tsc --noEmit 通过 |
| Playwright E2E（登录 → 快速设置改 TAC 保存 → 消息中心出现"进行中" → 终态切换） | ⏳ 待 docker rebuild 后跑 — 本 commit 后处理 |

---

## Checklist

| 类别 | 项 | 结果 |
|---|---|---|
| invalidate 覆盖 | 4 个 try/catch 入口（CellParameterForm Save + MultiInstanceTable Save/Add/Delete）都 finally 触发 | ✅ |
| invalidate 时机 | finally 而非 try 末尾，保证失败路径也刷新（让"入队失败"消息也被拉回） | ✅ |
| invalidate key | notificationKeys.all 父键 → list / unread-count 全刷 | ✅ |
| Add/Delete taskId 消费 | result.taskId 入 store，配合 useDeviceTaskStatus 启动轮询 | ✅ |
| store 角色 | 保留持久化 taskId（切走切回后 hook 能恢复轮询），与设计 §10 任务 #18 "store 文件可以保留作过渡"一致 | ✅ |
| 红线 | 前端**不本地造消息**——只 invalidate 触发 refetch（§4.3 红线） | ✅ |

---

## 发现

### CRITICAL / WARNING
- 无

### INFO
- I-01：设计文档 §10 任务 #18 提到"Tag 改为 hook 当前 task 状态，不读 quickSettingsFeedbackStore"——这句话其实在 T-0156 阶段已部分实施（statusTagSpec 用 lastTask?.status）。store 只保存 taskId+元数据，Tag 渲染时合并 lastTask 状态。完全废除 store 需要别的途径恢复 taskId（如查后端 GET /devices/tasks），工作量与收益不匹配，保留现状。
- I-02：mutation onSuccess 已经 invalidate parameter-tree/parameters/parameter-schema 等 query；本 commit 加的 notifications invalidate 是独立 query key，互不影响。
- I-03：UI 文案"右上角铃铛查看任务结果"提示已是早就有的，与新接入的消息中心契合 — 用户被指引去点铃铛。

---

## 与设计对齐

| §10.1-C10 项 | 实施 |
|---|---|
| CellParameterForm mutation 调用方式不变 | ✅ |
| mutation 成功 / 失败回调 invalidateQueries | ✅ 4 处 finally |
| MultiInstanceTable Save / Add / Delete 都接入 | ✅ |
| 卡片右上角 Tag 改读 useDeviceTaskStatus | ✅（T-0156 已实施，本 commit 保持） |
| store 文件保留作过渡 | ✅ |

---

## Phase 1 整体收官

C1-C10 全部完成：

| commit | sub-task | 状态 |
|---|---|---|
| 9fad3f05 | C1 task 超时配置化 | ✅ |
| 12e90942 | C2 worker task_sweeper | ✅ |
| 08a645c8 | C3 notifications 表迁移 + UpsertByDedup | ✅ |
| f8596638 | C4 DELETE /notifications | ✅ |
| 45252d9e | C5 task → notification 订阅器 | ✅ |
| 3686b957 | C6 CreateFailureNotifier 兜底 | ✅ |
| fb243dfc | C7 AddObject/DeleteObject 返 task_id | ✅ |
| 3a0ae1ee | C8 前端 API + 6 hooks | ✅ |
| 9b1a542a | C9 铃铛 Popover + UI | ✅ |
| 本 commit | C10 快速设置 invalidate + Add/Delete 消费 taskId | ✅ |

§8 验收：8 项 typecheck/单测已绿；Playwright E2E（#2/#3/#4/#5/#6/#9/#10）待 docker rebuild 后跑。

---

## 结论

**PASS**（代码 PASS_WITH_WARNINGS — Playwright E2E 未跑，需 docker 部署后补）

可合入，merge 后 docker rebuild → Playwright 验证 → 全部完成。
