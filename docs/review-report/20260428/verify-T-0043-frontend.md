# T-0043.F 前端验证报告 — 通知模板 + 历史记录

- 任务: T-0043.F (W2.A.4 通知模板 + 历史记录 — 前端)
- worktree 分支: worktree-agent-aa618f42
- 日期: 2026-04-28
- 范围: 仅 `omcmb/frontend-core/`、`omcmb/webcode/`，未触碰 `omcgo/`、`webcode-v2/`、`webcode-v3/`

---

## 1. 改动清单（7 类文件 + 路由）

| # | 文件 | 类型 | 说明 |
|---|------|------|------|
| 1 | `omcmb/frontend-core/src/types/notification.ts` | 新增 | 类型定义：`NotificationTemplate` / `NotificationHistory` / 列表分页响应 / Create-Update payload |
| 2 | `omcmb/frontend-core/src/services/api/notificationApi.ts` | 新增 | `notificationTemplateApi` + `notificationHistoryApi` 服务对象，含 `BackendXxx` snake_case 模型 + `mapBackend*` 转换 |
| 3 | `omcmb/frontend-core/src/hooks/api/useNotifications.ts` | 新增 | React Query Hooks：`useNotificationTemplates` / `useNotificationTemplate` / `useCreateNotificationTemplate` / `useUpdateNotificationTemplate` / `useDeleteNotificationTemplate` / `useNotificationHistory` / `useNotificationHistoryDetail` |
| 4 | `omcmb/frontend-core/src/i18n/zh-CN/index.ts` | 修改 | 末尾追加 30+ `notification.*` 中文键 |
| 5 | `omcmb/frontend-core/src/i18n/en-US/index.ts` | 修改 | 末尾追加 30+ `notification.*` 英文键（与 zh-CN 一一对应） |
| 6 | `omcmb/webcode/src/pages/notifications/` | 新增目录 | `index.tsx`（Tabs 入口）/ `TemplateList.tsx` / `TemplateForm.tsx`（Drawer 表单）/ `HistoryList.tsx` / `HistoryDetail.tsx`（Drawer） |
| 7 | `omcmb/webcode/src/router/routes.tsx` | 修改 | 注册 `/notifications` 路由（lazy 加载 + Suspense） |

---

## 2. 自跑验证

### 2.1 文件存在性
```
$ ls omcmb/frontend-core/src/services/api/notificationApi.ts \
     omcmb/frontend-core/src/hooks/api/useNotifications.ts \
     omcmb/frontend-core/src/types/notification.ts
✓ all present

$ ls -d omcmb/webcode/src/pages/notifications/
✓ HistoryDetail.tsx  HistoryList.tsx  TemplateForm.tsx  TemplateList.tsx  index.tsx
```

### 2.2 typecheck
```
$ cd omcmb/webcode && npm run typecheck
> webcode@0.0.0 typecheck
> tsc --noEmit
(0 errors, 0 warnings)
```

---

## 3. 与后端契约一致性

| 字段 | 类型 | 备注 |
|------|------|------|
| `id`, `name`, `subject`, `body`, `enabled` | string / boolean | 直通 |
| `channel` | `'email' \| 'sms' \| 'webhook'` | `asChannel()` 兜底转换；未知值降级为 webhook |
| `language` | `'zh-CN' \| 'en-US'` | `asLanguage()` 兜底转换；非 en-US 一律视为 zh-CN |
| `variables` | `string[]` | `null` → `[]` |
| `status` | `'pending' \| 'sent' \| 'failed' \| 'dead_letter'` | `asHistoryStatus()` 兜底；未知值降级为 pending |
| `createdAt` / `updatedAt` / `sentAt` | string (ISO8601) | 后端 `created_at` 等通过 `mapBackendXxx` 转 camelCase |
| `templateId` / `alarmId` / `errorMessage` / `retryCount` | optional / number | 历史专属，可空 |

**API 路径**（与任务定义一致）:
- `GET    /api/v1/notifications/templates` ?channel=&language=&enabled=&page=&page_size=
- `GET    /api/v1/notifications/templates/:id`
- `POST   /api/v1/notifications/templates`
- `PUT    /api/v1/notifications/templates/:id`
- `DELETE /api/v1/notifications/templates/:id`
- `GET    /api/v1/notifications/history`   ?channel=&status=&template_id=&alarm_id=&page=&page_size=
- `GET    /api/v1/notifications/history/:id`

> Axios baseURL `VITE_API_BASE_URL || /api/v1`，service 中相对路径 `/notifications/...` 即可。
> Query 参数 `pageSize` 通过 http.ts 拦截器映射为 `page_size`；`templateId/alarmId` 在 service 中显式以 `template_id/alarm_id` 传递。

---

## 4. UI 风格遵循

- 入口页 `notifications/index.tsx` 使用 `ListPageLayout` + Antd `Tabs`，与 `/license` 等模块一致
- 表单走 `Drawer`，列表走 `DataTable` + `FilterBar`，与 `/alarm`、`/device` 一致
- i18n 走 `useT()` Hook，所有用户可见文本均通过 `notification.*` 命名空间
- 类型严格：禁止 `any`，全部接口/字段显式 typed
- 错误反馈走 `antd.message.error()` + `err.message`（http.ts 拦截器统一处理 details/message）

---

## 5. 章程 W2.A.4 前端 Pass 标准

- [x] `omcmb/frontend-core/src/services/api/notificationApi.ts` 存在
- [x] `omcmb/webcode/src/pages/notifications/` 目录存在（5 文件）
- [x] `npm run typecheck` 0 errors
- [x] 路由 `/notifications` 已注册
- [x] 未触碰 `omcgo/`、`webcode-v2/`、`webcode-v3/`
- [x] 未新增 npm 依赖

---

## 6. 后续依赖

- 等待后端 sub-agent 完成 `internal/notification/` handler/service/router 的实际接口实现
- 联调验收：`bash run/scripts/start-all.sh` 起前后端，访问 `/notifications` 验证模板 CRUD + 历史浏览全链路
- 必要时补充导航菜单 entry（菜单在 `omcmb/webcode/src/components/Layout/Sidebar/` — 不在本 sub-agent 范围）
