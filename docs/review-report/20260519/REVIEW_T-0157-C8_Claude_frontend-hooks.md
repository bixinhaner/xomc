# Code Review — T-0157 C8 frontend-core 消息中心 API + 6 hooks 套件

- **日期**：2026-05-19
- **范围**：`frontend-core/types/notificationCenter.ts` / `services/api/notificationCenterApi.ts` / `mock/services/notificationCenterService.ts` / `mock/services/index.ts` / `hooks/api/useNotificationCenter.ts`
- **作者**：Claude
- **Reviewer**：Claude（self-review，前端视角）
- **关联**：T-0157 Phase 1 sub-task **C8**

---

## 变更概要

1. **types/notificationCenter.ts**：6 个类型（Type/Status/Priority/UiState/Item/Filter）+ `toUiState` 辅助函数（§4.4 五态映射）
2. **services/api/notificationCenterApi.ts**：6 个方法对接后端 6 个端点；BackendNotification → camelCase mapper
3. **mock/services/notificationCenterService.ts**：4 条 mock 消息（覆盖 queued/completed/failed/expired 四态，cancelled 暂不示例）+ in-memory CRUD
4. **mock/services/index.ts**：导出 notificationCenterService
5. **hooks/api/useNotificationCenter.ts**：6 个 hook + `notificationKeys` 共用 query key 前缀

---

## 命名决策

| 设计 §10.1 列的名 | 实施改用 | 理由 |
|---|---|---|
| `useNotifications` | `useNotificationCenter` | 与现有 `types/notification.ts`（SMTP 模板 / 通知历史）领域冲突 |
| `useUnreadCount` | `useNotificationUnreadCount` | 同上，全局唯一前缀 |
| `useMarkRead` | `useMarkNotificationRead` | 同上 |
| `useMarkAllRead` | `useMarkAllNotificationsRead` | 同上 |
| `useDeleteNotification` | `useDeleteNotification` | 已唯一 |
| `useClearAllNotifications` | `useClearAllNotifications` | 已唯一 |
| `notificationApi.ts` | `notificationCenterApi.ts` | 文件名同样避冲突 |
| `notification.ts` types | `notificationCenter.ts` types | 同上 |

→ 整体新增物件加 `Center` 后缀，与既有"通知模板/历史"领域明确解耦。

---

## 出口门验证

| 门 | 结果 |
|---|---|
| `omcmb/webcode npm run typecheck` | ✅ tsc --noEmit 通过 |
| `useMock` 切换路径双源行为一致（mock + api 同形签名） | ✅ |
| 6 个 mutation hook 都 invalidate `notificationKeys.all` 父键，list/unread 同时刷新 | ✅ |

---

## Checklist

| 类别 | 项 | 结果 |
|---|---|---|
| 业务层归属 | API/Hook/Types/Mock 全部在 frontend-core，UI 留给 C9 在 webcode | ✅ |
| 类型安全 | 无 any；BackendNotification 显式 interface；mapper 转 camelCase | ✅ |
| Mock/Real 切换 | useMock 决定 api 指向，两端签名同形（TS 类型校验保证） | ✅ |
| React Query 模式 | 6 hook 用统一 queryKey 前缀；mutation onSuccess 全 invalidate | ✅ |
| 默认值 | useNotificationCenter 默认 page=1 pageSize=20；useNotificationUnreadCount 默认 10s 轮询（§5.2） | ✅ |
| 后台轮询 | `refetchIntervalInBackground: true`，切到其他 tab 也保持铃铛准确 | ✅ |
| 字段映射保底 | mapBackendNotification 显式列字段（即使 http 拦截器自动转换，本 mapper 兜底） | ✅ |
| Sort 协议 | sortOrder='descend' → API 'desc'，与 PageRequest 类型一致 | ✅ |

---

## 发现

### CRITICAL / WARNING
- 无

### INFO
- I-01：mock 数据 4 条覆盖 queued/completed/failed/expired，**cancelled 未给样例** — V1 用户不会在 mock 模式碰到 cancelled（task 取消是用户主动行为，需端到端走 ACS 才会触发），缺也无大碍。
- I-02：`notificationCenterApi.list` 接受 `params.filter.isRead` 转 'true'/'false' 字符串 — 后端 handler 也是字符串比较（`c.Query("is_read") == "true"`），契约对齐。
- I-03：`useNotificationUnreadCount` 默认轮询 10s，但**没有暂停机制**（如登出后仍在轮询）。生产中由 React Query `enabled` 配合 userStore 控制：登出后 userId 清空，UI 层把 `enabled=false` 传入即可（C9 实施时由 NotificationCenter 组件接管）。
- I-04：hook 模式与 useDeviceParameters 等既有 hook 完全一致，无新模式引入。

---

## 结论

**PASS**

可合入，可继续 C9（Header 铃铛 Popover + 消息列表 UI 组件）。
