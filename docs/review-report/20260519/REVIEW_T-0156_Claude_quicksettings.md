# Code Review — T-0156 快速设置切走切回 Tag 反馈消失（修复）

- **日期**：2026-05-19
- **范围**：`omcmb/frontend-core/src/store/{index,quickSettingsFeedbackStore}.ts` / `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/{CellParameterForm,MultiInstanceTable}.tsx`
- **作者**：Claude
- **Reviewer**：Claude（self-review，前端专家 + 用户验证）
- **关联 Task**：T-0156

---

## Bug 描述

用户在快速设置 tab 改参数 → 保存 → Tag 显示"已入队,等待下发"→ 切到设备列表（TabBar 顶层 tab 切换）→ 再切回设备详情快速设置 → Tag 消失（即使 task 已到终态）。

## 根因（两层）

1. **状态生命周期错配**：`CellParameterForm` / `MultiInstanceTable` 的 `lastSubmit` / `lastAction` 用 `useState` 持有；TabBar 顶层 tab 切换时 Layout 用 `<Outlet />` 不带 KeepAlive → DeviceDetail 整树卸载 → useState 状态丢失
2. **Card loading 遮蔽 extra 区**：`CellParameterForm` 的 `<Card loading={isLoading}>` 在 schema refetch 期间会让**整张卡片包括 extra 区的 Tag** 进骨架屏；即使 store 数据还在，Tag 也会瞬间消失

## 修复

1. 新建 `frontend-core/src/store/quickSettingsFeedbackStore.ts` — zustand + sessionStorage persist，按 `(deviceId, groupId, fapInstance)` 索引 CellFeedback/MultiFeedback
2. `CellParameterForm` / `MultiInstanceTable` 把 `useState` 替换为 store 读写；`Date` → `number`（epoch ms，绕开 JSON 持久化丢类型）
3. `notifiedFailedTaskId` 也存入 store，避免切回时重复弹 notification.error
4. `CellParameterForm` 的 Card 移除 `loading` prop，改用 `<Spin spinning={isLoading}>` 包裹内部 Form；extra 区 Tag 始终保留

---

## Checklist

| 类别 | 项 | 结果 |
|---|---|---|
| 类型安全 | union type CellFeedback / MultiFeedback；无 any | ✅ |
| 业务层归属 | store 写 `frontend-core/`，UI 改动写 `webcode/` | ✅ |
| 持久化 | sessionStorage 与 tabStore 一致；Date → epoch ms 避免序列化丢类型 | ✅ |
| 多皮肤影响 | 新 store 仅多出一个文件，三皮肤都可消费；非破坏性 | ✅ |
| 渲染优化 | Card loading 拆为 Spin，extra 区不受刷新干扰 | ✅ |
| 引用清理 | LastSubmitState / LastActionState 旧 interface 全部删除 | ✅ |
| dedup | feedbackKey 设计支持单/多分组互不干扰 | ✅ |

---

## 发现

### CRITICAL
- 无

### WARNING
- W-01：`notifiedFailedTaskId` 存进 store 后，store 条目永远不会被自动 GC（仅在用户主动改参数时覆盖）。store 长期累积可能占满 sessionStorage。
  - **缓解**：sessionStorage 5MB 容量；每个 feedback 条目约 200B，理论可存 25k 条；常规用户单 session 改参数不会超过几十次，可接受
  - **建议**：T-0157 消息中心 V1 上线后，本 store 会被废弃（卡片 Tag 改读 useDeviceTaskStatus），届时清理（见 §10.10 Phase 1 任务 #18）

### INFO
- I-01：MultiInstanceTable 的 Card loading 沿用在 Table 上（非 Card），不受 §修复 #4 影响——已确认不需要拆。
- I-02：旧的 `quickSettingsFeedbackStore` 文件保留——T-0157 实施时（Phase 1 task #18）会迁移卡片 Tag 到 `useDeviceTaskStatus` 后再清理。

---

## 测试

- Playwright E2E（先前会话）：登录 → 改 TAC 3→4 → 保存 → Tag 显示"已入队"→ 切设备列表 → 切回 → Tag 恢复 "基站应答成功 · 1 项 · 13:45:17"（时间戳与首次保存一致）→ 8 次 200ms 采样无 cardLoading 闪烁
- `npm run typecheck` 通过

---

## 结论

**PASS_WITH_WARNINGS**（W-01 容量风险已有缓解路径，不阻塞合入）

可合入。
