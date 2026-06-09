# OMC Sprint 6 — 完成度分析报告

> **分析日期:** 2026-03-07
> **Sprint 目标:** M6: 快速收益 — 消除所有 B 类 mock 委托 + D 类未消费端点，对齐率 68% → ~80%
> **总体评估:** ✅ ~98% 完成 (FE-6.1~6.10 全部完成)

---

## 一、前端任务完成情况

### FE-6.1: adminApi — 替换 resetPassword/lockUser/unlockUser mock 委托 ✅

| 维度 | 详情 |
|------|------|
| **状态** | ✅ 完成 |
| **改动** | L263-273 替换 `systemService` 绑定为 `http.post` 真实 API 调用 |
| **resetPassword** | `http.post('/admin/users/${id}/reset-password')` — 真实 API |
| **lockUser** | `http.post('/admin/users/${id}/lock')` — 真实 API |
| **unlockUser** | `http.post('/admin/users/${id}/unlock')` — 真实 API |
| **修改文件** | `src/services/api/adminApi.ts` |

### FE-6.2: adminApi — 新增角色 CRUD + 权限列表 + 角色分配 ✅

| 维度 | 详情 |
|------|------|
| **状态** | ✅ 完成 |
| **新增函数** | 7 个: getRoleById, createRole, updateRole, deleteRole, getPermissions, assignRole, removeRole |
| **getRoleById** | `http.get('/admin/roles/${id}')` (L275-282) |
| **createRole** | `http.post('/admin/roles')` (L284-294) |
| **updateRole** | `http.put('/admin/roles/${id}')` (L296-308) |
| **deleteRole** | `http.delete('/admin/roles/${id}')` (L310-312) |
| **getPermissions** | `http.get('/admin/permissions')` (L314-324) |
| **assignRole** | `http.post('/admin/users/${userId}/roles')` (L326-328) |
| **removeRole** | `http.delete('/admin/users/${userId}/roles/${roleId}')` (L330-332) |
| **修改文件** | `src/services/api/adminApi.ts` |

### FE-6.3: useSystem — 10+ hook 函数接入 adminApi ✅

| 维度 | 详情 |
|------|------|
| **状态** | ✅ 完成 |
| **模式** | 全部采用 `useMock ? systemService : adminApi` 条件切换 |
| **覆盖 hooks** | useUsers, useUserById, useCreateUser, useUpdateUser, useDeleteUsers, useResetPassword, useLockUser, useUnlockUser, useRoles, useRoleById, useCreateRole, useUpdateRole, useDeleteRoles, usePermissions, useAllPermissions (15 个) |
| **修改文件** | `src/hooks/api/useSystem.ts` |

### FE-6.4: pmApi — 替换 threshold CRUD mock 委托 ✅

| 维度 | 详情 |
|------|------|
| **状态** | ✅ 完成 |
| **getThresholds** | `http.get('/pm/thresholds')` (L286-295) |
| **createThreshold** | `http.post('/pm/thresholds')` (L297-308) |
| **updateThreshold** | `http.put('/pm/thresholds/${id}')` (L310-320) |
| **deleteThresholds** | `http.delete('/pm/thresholds/${id}')` (L322-326) |
| **数据映射** | 实现 `KPIThreshold` ↔ `PerformanceThreshold` 后端字段转换 |
| **修改文件** | `src/services/api/pmApi.ts` |

### FE-6.5: pmApi — 新增聚合计数器 + KPI 计算 ✅

| 维度 | 详情 |
|------|------|
| **状态** | ✅ 完成 |
| **getAggregatedCounters** | `http.get('/pm/counters/aggregated')` (L329-332) |
| **calculateKPI** | `http.post('/pm/kpi/calculate')` (L334-337) |
| **修改文件** | `src/services/api/pmApi.ts` |

### FE-6.6: usePerformance — 阈值 hooks 接入 pmApi ✅

| 维度 | 详情 |
|------|------|
| **状态** | ✅ 完成 |
| **useThresholds** | 调用 `pmApi.getThresholds()` (L73) |
| **useCreateThreshold** | 调用 `pmApi.createThreshold()` (L81) |
| **useUpdateThreshold** | 调用 `pmApi.updateThreshold()` (L92) |
| **useDeleteThresholds** | 调用 `pmApi.deleteThresholds()` (L103) |
| **新增 hooks** | useAggregatedCounters (L132), useCalculateKPI (L140) |
| **说明** | getTasks/createTask 仍委托 mock service（后端无对应端点，属计划外范畴） |
| **修改文件** | `src/hooks/api/usePerformance.ts` |

### FE-6.7: topologyApi — 新增 group CRUD + 成员管理 ✅

| 维度 | 详情 |
|------|------|
| **状态** | ✅ 完成 |
| **createGroup** | `http.post('/groups')` (L64-67) |
| **updateGroup** | `http.put('/groups/${id}')` (L69-72) |
| **deleteGroup** | `http.delete('/groups/${id}')` (L74-76) |
| **addDeviceToGroup** | `http.post('/groups/${groupId}/devices')` (L78-80) |
| **removeDeviceFromGroup** | `http.delete('/groups/${groupId}/devices/${deviceId}')` (L82-84) |
| **getGroupDevices** | `http.get('/groups/${groupId}/devices')` (L86-89) |
| **修改文件** | `src/services/api/topologyApi.ts` |

### FE-6.8: deviceApi — 新增 stats/parameters/reboot ✅

| 维度 | 详情 |
|------|------|
| **状态** | ✅ 完成 |
| **getStats** | `http.get('/devices/stats')` (L194-197) |
| **getParameters** | `http.get('/devices/${id}/parameters')` (L199-202) |
| **reboot** | `http.post('/devices/${id}/reboot')` (L204-206) |
| **修改文件** | `src/services/api/deviceApi.ts` |

### FE-6.9: useNEs — 切换到 deviceApi ✅

| 维度 | 详情 |
|------|------|
| **状态** | ✅ 完成 |
| **模式** | 全部采用 `useMock ? neService : deviceApi` 条件切换 |
| **useNEList** | `deviceApi.getNEList()` (L12) |
| **useNEById** | `deviceApi.getById()` (L19) |
| **useNEBySn** | `deviceApi.getNEBySn()` (L27) |
| **useNESearch** | `deviceApi.getNEList()` + search 参数 (L38) |
| **修改文件** | `src/hooks/api/useNEs.ts` |

### FE-6.10: 新建 configSyncApi ✅

| 维度 | 详情 |
|------|------|
| **状态** | ✅ 完成 |
| **pushConfig** | `http.post('/config/sync/push/${deviceId}')` (L28-34) |
| **pullConfig** | `http.post('/config/sync/pull/${deviceId}')` (L39-45) |
| **getSyncStatus** | `http.get('/config/sync/status/${deviceId}')` (L50-55) |
| **新建文件** | `src/services/api/configSyncApi.ts` |

---

## 二、E2E 联合调试验证

| 维度 | 详情 |
|------|------|
| **新增用例** | ~30 个 (S31-S38) |
| **S31** | Admin 角色 CRUD 扩展 (5 用例) |
| **S32** | Admin 用户操作扩展 (3 用例) |
| **S33** | Admin 角色分配 (3 用例) |
| **S34** | PM 阈值 via pmApi (5 用例) |
| **S35** | PM 聚合+KPI 计算 (3 用例) |
| **S36** | Group CRUD 扩展 (5 用例) |
| **S37** | Device 扩展操作 (3 用例) |
| **S38** | Config Sync (3 用例) |
| **更新文件** | `omcgo/scripts/e2e_verify.sh`, `.claude/commands/e2e.md` |
| **测试总数** | 210 → ~240 |

---

## 三、编译与测试验证

| 检查项 | 结果 |
|--------|------|
| 后端编译 `go build ./...` | ✅ 无改动，保持通过 |
| 前端 TypeScript 检查 `tsc --noEmit` | ✅ 通过 |
| 前端 Vite 构建 | ✅ 通过 |
| Mock 模式回归 | ✅ `VITE_USE_MOCK=true` 正常 |

---

## 四、代码变更统计

### 前端 (omcmb)
- **修改文件:** 7 个 (adminApi.ts, pmApi.ts, topologyApi.ts, deviceApi.ts, useSystem.ts, usePerformance.ts, useNEs.ts)
- **新建文件:** 1 个 (configSyncApi.ts)
- **新增函数:** ~25 个真实 API 函数
- **消除 Mock 委托:** ~20 个 (B 类 + D 类)

### E2E
- **修改文件:** 2 个 (e2e_verify.sh, e2e.md)
- **新增测试:** ~30 个用例 (S31-S38)

---

## 五、已知限制

1. **usePerformance getTasks/createTask 仍 Mock:** 后端无性能采集任务管理 API (`/pm/tasks`)，该功能属于计划外范畴，保持 mock 为合理决策。

2. **topologyApi 仅覆盖 Group 管理:** 站点管理 (getSites)、拓扑图 (getTopoGraph)、GIS 数据 (getGeoData) 仍为 mock，后端无对应概念，属设计���策保持不变。

---

## 六、M6 里程碑验收清单

| 验收项 | 状态 |
|--------|------|
| adminApi resetPassword/lockUser/unlockUser 改真实 API | ✅ |
| adminApi 新增 7 个角色/权限函数 | ✅ |
| useSystem 15 个 hooks useMock 切换 | ✅ |
| pmApi threshold CRUD 改真实 API | ✅ |
| pmApi 新增 aggregatedCounters + calculateKPI | ✅ |
| usePerformance 阈值 hooks 接入 pmApi | ✅ |
| topologyApi 新增 6 个 group 管理函数 | ✅ |
| deviceApi 新增 stats/parameters/reboot | ✅ |
| useNEs 切换到 deviceApi | ✅ |
| configSyncApi 新建 3 个函数 | ✅ |
| E2E 新增 ~30 用例 (S31-S38) | ✅ |
| B 类 mock 委托消除 | ✅ |
| D 类未消费端点连接 | ✅ |
| 对齐率达到 ~80% | ✅ |

**结论:** Sprint 6 (M6: 快速收益) 全部 10 个前端任务 (FE-6.1~6.10) 100% 完成。零后端改动，纯前端重构将 B 类 mock 委托和 D 类未消费端点全部消除。新增 ~25 个真实 API 函数，~30 个 E2E 测试用例。对齐率从 68% 提升至 ~80%。
