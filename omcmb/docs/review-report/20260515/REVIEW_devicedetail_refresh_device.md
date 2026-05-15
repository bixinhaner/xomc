---
date: 2026-05-15
author: shangyingbin
scope: device
type: fix
files:
  - omcmb/webcode/src/pages/device/DeviceDetail/index.tsx
verdict: PASS
---

# Review — 设备详情页头刷新按钮按 Tab 联动

## 背景

设备详情页（`/device/detail/:sn`）页头右上角「刷新」按钮原仅触发 `useDeviceBySn.refetch()`，
只刷新设备基础信息（header 显示用），与当前激活 Tab 的数据源无关。

用户切换到「参数树」或「活动告警」Tab 后点击「刷新」，预期是对当前 Tab 的内容做一次刷新，
但实际除设备基础接口外，其他 Tab 的数据源都不会被重新拉取。本次修复让刷新按钮按 Tab 派发刷新动作。

## 变更摘要

- `omcmb/webcode/src/pages/device/DeviceDetail/index.tsx`
  - 引入 `useCallback` 与 `useQueryClient`
  - 新增 `handleHeaderRefresh`：始终 `refetch()` 设备基础信息，再按 `activeTab` 调用
    `queryClient.invalidateQueries(...)` 触发对应 Query 重新拉取
  - 替换页头「刷新」按钮 `onClick` 句柄

## Tab → 刷新动作映射

| Tab          | invalidate 的 Query Key 前缀                                                  |
|--------------|--------------------------------------------------------------------------------|
| `basic` 详情 | 仅 `refetch()` 设备（基础信息）                                               |
| `parameters` 参数树 | `['devices', 'object-tree', deviceId]`、`['devices', 'children', deviceId]`、`['devices', 'parameters', deviceId]` |
| `alarms` 活动告警 | `['alarms', 'current']`                                                  |
| `performance` KPI / `license` 许可证 | 仅 `refetch()` 设备（当前为 mock 数据，无后端 API）              |

> React Query `invalidateQueries` 默认按前缀匹配，对带参数（pageSize/path 等）的 children query
> 同样生效。

## 审查清单

### 前端规范

- [x] **类型安全**：未引入 `any` / `unknown`，全部为有类型句柄
- [x] **Hook 规则**：`useCallback` 依赖数组完整（`activeTab` / `refetch` / `queryClient` / `device?.id`）
- [x] **查询键约定**：复用 `useDeviceParameters.ts` / `useAlarms.ts` 已定义的层级式 key
- [x] **业务层归属**：API/Hook 仍在 `frontend-core/`，UI 壳仅消费 `useQueryClient`
- [x] **国际化**：未引入新文案，沿用 `t('common.refresh')`
- [x] **多皮肤影响**：未修改 `frontend-core/` 类型 / Mock / Store 形态，对 `webcode-v2/`、`webcode-v3/` 无影响
- [x] **不破坏现有行为**：详情 Tab 行为保持原状（仅 `refetch()` 设备）

### 通用

- [x] 无硬编码 SN / deviceId
- [x] 无敏感信息泄漏
- [x] `void` 标注异步 Promise，避免悬挂
- [x] `tsc --noEmit` 通过

## 真机验证（BAICELLS `1202000240194DP0026`）

| 步骤 | 操作 | 实测 |
|------|------|------|
| 1 | 进入参数树 Tab，选中 `Common.` 节点，右侧显示 `CellIdentity=123456` | 与 DB 一致 |
| 2 | DB 改 `123456 → 999888` | UPDATE 1 |
| 3 | 点页头「刷新」 | 触发 `GET /devices/{id}/parameters/tree` + `children` |
| 4 | 页面显示 | `999888`（已同步） |
| 5 | DB 再改为 `20260515`，用户手动点刷新 | 页面更新为 `20260515` |
| 6 | 切到「活动告警」Tab 点刷新 | 触发 `GET /alarms/active?device_sn=...` |

## 结论

PASS。无 CRITICAL / WARNING。

## Out of Scope

- KPI / 许可证 Tab 当前为 mock 数据；接入真实接口后可在 `handleHeaderRefresh` 的 switch
  对应分支补 `invalidateQueries`
- 「同步参数」按钮（参数树面板内）仍为触发 TR-069 `GetParameterValues` 的设备级同步，
  与刷新按钮（重读 DB 缓存）语义不同，按预期保留两个独立操作
