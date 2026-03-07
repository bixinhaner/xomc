# OMC 前后端整合 — 开发实施计划 (Sprint 6-9)

## Context

Sprint 0-5 全部完成后，整体前后端对齐率 ~68%。基于《前后端整合方案.md》(2026-03-07 更新) 的精确代码分析，识别出四类剩余差距：

- **B 类 (部分对齐)**: 4 个前端 API 服务存在但部分函数仍委托 mock
- **C 类 (纯 Mock)**: 7 个 Hook 无 real API 服务，无后端端点
- **D 类 (前端未消费)**: ~20 个后端端点前端完全未调用

本计划将剩余工作拆分为 **Sprint 6-9**，每个 Sprint 含联合调试验证 (E2E)，逐步将对齐率从 68% 提升至 95%。

---

## 进度追踪

| Sprint | 前端状态 | 后端状态 | 里程碑 | 对齐率目标 | 完成日期 |
|--------|---------|---------|--------|-----------|---------|
| **Sprint 6** | 待开始 (10 tasks) | 无改动 (0 tasks) | M6: 部分→完全对齐 | ~80% | — |
| **Sprint 7** | 待开始 (8 tasks) | 待开始 (6 tasks) | M7: Dashboard+Config 完整 | ~85% | — |
| **Sprint 8** | 无改动 (0 tasks) | 待开始 (8 tasks) | M8: C 类后端就绪 | ~88% | — |
| **Sprint 9** | 待开始 (7 tasks) | 待开始 (2 tasks) | M9: 全面整合 | ~95% | — |

---

## 总体时间线

```
Sprint  │ 6 (1-2天)       │ 7 (1-2天)        │ 8 (1-2天)       │ 9 (1-2天)
────────┼─────────────────┼──────────────────┼─────────────────┼─────────────────
前端    │ ██ 连接已有后端  │ ██ Dashboard完整  │                 │ ██ 新模块集成
        │ B+D类消除       │ 配置/拓扑增强     │ (无前端改动)      │ backup/files/mml
────────┼─────────────────┼──────────────────┼─────────────────┼─────────────────
后端    │                 │ ██ Dashboard      │ ██ C类模块构建   │ ██ OpenAPI+测试
        │ (无后端改动)     │ 4新聚合端点       │ backup/files/mml │ 回归稳定化
────────┼─────────────────┼──────────────────┼─────────────────┼─────────────────
E2E     │ +30 → 240用例   │ +20 → 260用例    │ +25 → 285用例   │ +15 → 300用例
────────┼─────────────────┼──────────────────┼─────────────────┼─────────────────
对齐率  │ 68% → 80%       │ 80% → 85%        │ 85% → 88%       │ 88% → 95%
```

---

## 一、Sprint 6 — 快速收益：前端连接已有后端端点

**目标:** 消除所有 B 类 mock 委托 + D 类未消费端点，对齐率 68% → ~80%
**策略:** 零后端改动，所有后端端点已存在且经过 Sprint 4 E2E 验证
**工作量:** ~1-2 天

### 前端任务

| # | 任务 | P | 产出文件 | 说明 |
|---|------|---|---------|------|
| FE-6.1 | adminApi: 替换 resetPassword/lockUser/unlockUser mock 委托 | P0 | `src/services/api/adminApi.ts` | L265-267 替换 `systemService` 绑定为 `http.post('/admin/users/${id}/reset-password')` 等 |
| FE-6.2 | adminApi: 新增角色 CRUD + 权限列表 + 角色分配 | P0 | `src/services/api/adminApi.ts` | 新增 `getRoleById`, `createRole`, `updateRole`, `deleteRole`, `getPermissions`, `assignRole`, `removeRole` 7 个函数 |
| FE-6.3 | useSystem: 10 个 hook 函数接入 adminApi | P0 | `src/hooks/api/useSystem.ts` | L61-66 `useResetPassword`, L68-75 `useLockUser`, L78-85 `useUnlockUser`, L106-111 `useRoleById`, L114-122 `useCreateRole`, L125-133 `useUpdateRole`, L136-143 `useDeleteRoles`, L146-151 `usePermissions`, L154-159 `useAllPermissions` → 全部改为 `useMock ? systemService : adminApi` |
| FE-6.4 | pmApi: 替换 threshold CRUD mock 委托 | P0 | `src/services/api/pmApi.ts` | 替换 `getThresholds`, `createThreshold`, `updateThreshold`, `deleteThresholds` 为 `http.get/post/put/delete('/pm/thresholds')`, 需增加 `KPIThreshold` ↔ `PerformanceThreshold` 映射 |
| FE-6.5 | pmApi: 新增聚合计数器 + KPI 计算 | P1 | `src/services/api/pmApi.ts` | 新增 `getAggregatedCounters()` → `GET /pm/counters/aggregated`, `calculateKPI()` → `POST /pm/kpi/calculate` |
| FE-6.6 | usePerformance: 阈值 hooks 接入 pmApi | P0 | `src/hooks/api/usePerformance.ts` | L68-72 `useThresholds`, L75-83 `useCreateThreshold`, L86-94 `useUpdateThreshold`, L97-104 `useDeleteThresholds` → `useMock ? performanceService : pmApi` |
| FE-6.7 | topologyApi: 新增 group CRUD + 成员管理 | P1 | `src/services/api/topologyApi.ts` | 新增 `createGroup`, `updateGroup`, `deleteGroup`, `addDeviceToGroup`, `removeDeviceFromGroup`, `getGroupDevices` 6 个函数 |
| FE-6.8 | deviceApi: 新增 stats/parameters/reboot | P1 | `src/services/api/deviceApi.ts` | 新增 `getStats()` → `GET /devices/stats`, `getParameters(id)` → `GET /devices/${id}/parameters`, `reboot(id)` → `POST /devices/${id}/reboot` |
| FE-6.9 | useNEs: 切换到 deviceApi | P1 | `src/hooks/api/useNEs.ts` | 所有 `neService` 调用改为 `useMock ? neService : deviceApi` 等效调用 |
| FE-6.10 | 新建 configSyncApi | P2 | `src/services/api/configSyncApi.ts` (新建) | `pushConfig(deviceId, params[])`, `pullConfig(deviceId, paramNames[])`, `getSyncStatus(deviceId)` |

### 关键适配说明

**adminApi L265-267 替换前:**
```typescript
resetPassword: systemService.resetPassword.bind(systemService),
lockUser: systemService.lockUser.bind(systemService),
unlockUser: systemService.unlockUser.bind(systemService),
```

**替换后:**
```typescript
async resetPassword(id: string, newPassword: string): Promise<void> {
  await http.post(`/admin/users/${id}/reset-password`, { new_password: newPassword });
},
async lockUser(id: string): Promise<void> {
  await http.post(`/admin/users/${id}/lock`);
},
async unlockUser(id: string): Promise<void> {
  await http.post(`/admin/users/${id}/unlock`);
},
```

**pmApi KPI Threshold 后端模型:**
```go
// Backend: KPIThreshold (pm/threshold_handler.go)
type KPIThreshold struct {
    ID                 string   `json:"id"`
    KPIName            string   `json:"kpi_name"`
    Carrier            string   `json:"carrier"`
    Technology         string   `json:"technology"`
    WarningThreshold   *float64 `json:"warning_threshold"`
    MinorThreshold     *float64 `json:"minor_threshold"`
    MajorThreshold     *float64 `json:"major_threshold"`
    CriticalThreshold  *float64 `json:"critical_threshold"`
    Comparison         string   `json:"comparison"`  // ">", "<", ">=", "<="
    Enabled            bool     `json:"enabled"`
    Description        string   `json:"description"`
}
```

### E2E 联合调试验证 (Sprint 6)

**新增测试用例: ~30 个**
**Seed 数据: 无需新增** (Sprint 4 已有 alarm_rules, kpi_thresholds, roles, permissions, device_group_members)

**e2e_verify.sh 新增 Section:**

| Section | 名称 | 用例数 | 验证内容 |
|---------|------|--------|---------|
| S31 | Admin 角色 CRUD 扩展 | 5 | createRole → getRoleById → updateRole → deleteRole → getPermissions |
| S32 | Admin 用户操作扩展 | 3 | resetPassword → lockUser → unlockUser |
| S33 | Admin 角色分配 | 3 | assignRole → verify → removeRole |
| S34 | PM 阈值 via pmApi | 5 | listThresholds → createThreshold → updateThreshold → verify → deleteThreshold |
| S35 | PM 聚合+KPI 计算 | 3 | getAggregatedCounters → calculateKPI → verify |
| S36 | Group CRUD 扩展 | 5 | createGroup → addDevice → listDevices → removeDevice → deleteGroup |
| S37 | Device 扩展操作 | 3 | getStats → getParameters → reboot |
| S38 | Config Sync | 3 | pushConfig → pullConfig → getSyncStatus |

**e2e.md 更新:**
- 测试总数: 210 → ~240
- 新增 Sprint 6 描述段落

### 验收标准

| 检查项 | 预期结果 |
|--------|---------|
| `adminApi.resetPassword(id, pw)` | `POST /admin/users/:id/reset-password` 返回 200 |
| `adminApi.createRole(data)` | `POST /admin/roles` 返回 201 |
| `adminApi.getPermissions()` | `GET /admin/permissions` 返回权限数组 |
| `pmApi.getThresholds(params)` | `GET /pm/thresholds` 返回分页列表 |
| `configSyncApi.pushConfig(id, params)` | `POST /config/sync/push/:id` 返回 command_id |
| `deviceApi.getStats()` | `GET /devices/stats` 返回 counts 对象 |
| Mock 模式回归 | `VITE_USE_MOCK=true` 所有页面正常 |
| 前端构建 | `tsc --noEmit` + `vite build` 通过 |
| E2E | ~240/240 PASS |

---

## 二、Sprint 7 — Dashboard 完整化 + Config 整合

**目标:** Dashboard 5 个 mock 函数全部替换为真实后端，对齐率 ~80% → ~85%
**策略:** 后端新增 4 个 Dashboard 聚合端点，前端全面对接
**工作量:** ~1-2 天

### 后端任务

| # | 任务 | P | 产出文件 | 说明 |
|---|------|---|---------|------|
| BE-7.1 | Dashboard: 告警趋势聚合 | P0 | `internal/omcr/dashboard/handler.go` | `GET /dashboard/alarm-trend?days=7` → 按日期+严重度聚合 alarms 表，返回 `[{date, critical, major, minor, warning}]`。复用 alarmStore 查询 |
| BE-7.2 | Dashboard: 设备状态饼图 | P0 | `internal/omcr/dashboard/handler.go` | `GET /dashboard/device-status` → 调用已有 `DeviceService.CountByStatus`，返回 `{online, offline, maintenance, ...}` |
| BE-7.3 | Dashboard: KPI 趋势 | P1 | `internal/omcr/dashboard/handler.go` | `GET /dashboard/kpi-trend?kpi_name=...&days=7` → 查询 kpi_values 时序，返回 `[{time, value}]` |
| BE-7.4 | Dashboard: 区域统计 | P2 | `internal/omcr/dashboard/handler.go` | `GET /dashboard/region-stats` → 按分组聚合设备/在线/告警数，返回 `[{region, device_count, online_count, alarm_count}]` |
| BE-7.5 | 注册新路由 | P0 | `cmd/app/main.go` | 在已有 dashboard 路由组下注册 4 个新子路由 |
| BE-7.6 | OpenAPI 补充 | P2 | `api/openapi/openapi.yaml` | 新增 4 个 Dashboard path 定义 |

### 前端任务

| # | 任务 | P | 产出文件 | 说明 |
|---|------|---|---------|------|
| FE-7.1 | dashboardApi: 替换 5 个 mock 函数 | P0 | `src/services/api/dashboardApi.ts` | `getChartData` → 组合 alarm-trend+device-status+kpi-trend; `getAlarmTrend` → `/dashboard/alarm-trend`; `getDeviceStatusPie` → `/dashboard/device-status`; `getKPITrend` → `/dashboard/kpi-trend`; `getRegionStats` → `/dashboard/region-stats` |
| FE-7.2 | useDashboard: 确认 API switch 完整 | P0 | `src/hooks/api/useDashboard.ts` | L6 `const api = useMock ? dashboardService : dashboardApi` 确保所有 7 个 hook 均走此 switch |
| FE-7.3 | useConfig: configParams 对接 configSyncApi | P1 | `src/hooks/api/useConfig.ts` | `useConfigParams` → `configSyncApi.pullConfig` 或 `deviceApi.getParameters`; `useUpdateConfigParam` → `configSyncApi.pushConfig` |
| FE-7.4 | useTopology: 新增 group CRUD hooks | P1 | `src/hooks/api/useTopology.ts` | 新增 `useCreateGroup`, `useUpdateGroup`, `useDeleteGroup`, `useGroupDevices`, `useAddDeviceToGroup`, `useRemoveDeviceFromGroup` 调用 Sprint 6 topologyApi |
| FE-7.5 | dashboardApi.getDashboardData 去除 mock 降级 | P1 | `src/services/api/dashboardApi.ts` | 组合真实端点数据替代 `dashboardService.getDashboardData()` fallback |
| FE-7.6 | 设备列表页: 接入 stats 面板 | P2 | 设备列表页组件 | 调用 `deviceApi.getStats()` 展示设备统计卡片 |
| FE-7.7 | 设备详情页: 参数 Tab + 重启按钮 | P2 | 设备详情页组件 | `deviceApi.getParameters(id)` 展示 TR069 参数表; `deviceApi.reboot(id)` 重启按钮 |
| FE-7.8 | datamodelApi: 连接 resolve 端点 | P2 | `src/services/api/datamodelApi.ts` | 新增 `resolveDataModel(carrier, tech, oui, productClass)` → `GET /datamodels/resolve` |

### E2E 联合调试验证 (Sprint 7)

**新增测试用例: ~20 个**

**Seed 数据补充 (`seed_e2e_testdata.sql`):**
- 补充 2-3 行不同日期的告警数据，用于验证告警趋势聚合

**e2e_verify.sh 新增 Section:**

| Section | 名称 | 用例数 | 验证内容 |
|---------|------|--------|---------|
| S39 | Dashboard 告警趋势 | 3 | 返回数据 + 日期分组正确 + 7 天范围 |
| S40 | Dashboard 设备状态 | 2 | 返回状态计数 + 总数匹配 |
| S41 | Dashboard KPI 趋势 | 3 | 返回时序 + kpi_name 正确 + 非空数据 |
| S42 | Dashboard 区域统计 | 2 | 返回区域列表 + 含 device_count |
| S43 | Config Sync 集成 | 3 | push+params + pull+names + status+pending_count |
| S44 | Device 参数查询 | 2 | 返回 items 数组 + 含 name/value |
| S45 | DataModel resolve | 2 | 返回 model + 未知参数优雅降级 |
| S46 | Sprint 7 回归 | 3 | dashboard/summary 仍正常 + healthz + CORS |

**e2e.md 更新:**
- 测试总数: ~240 → ~260
- 新增 Sprint 7 描述段落

### 验收标准

| 检查项 | 预期结果 |
|--------|---------|
| `GET /dashboard/alarm-trend?days=7` | 返回数组，含 date+severity 计数 |
| `GET /dashboard/device-status` | 返回各状态设备数 |
| `GET /dashboard/kpi-trend?kpi_name=...` | 返回时序数据数组 |
| Dashboard 页面 (`VITE_USE_MOCK=false`) | 所有 6 个图表组件显示真实数据 |
| 设备详情页 | 参数 Tab 展示 TR069 参数 + 重启按钮可用 |
| 后端编译 | `go build ./...` 通过 |
| 前端构建 | `tsc --noEmit` + `vite build` 通过 |
| E2E | ~260/260 PASS |

---

## 三、Sprint 8 — C 类后端构建

**目标:** 为 3 个高优先级纯 Mock 模块构建后端 API，对齐率 ~85% → ~88%
**策略:** 纯后端开发，不涉及前端改动
**范围决策:**
- **本 Sprint 构建:** 备份恢复 (运维必需), 文件管理 (MinIO 集成就绪), MML 控制台 (核心运维)
- **保持 Mock:** License (静态数据, 低频), 报表 (可从已有 API 聚合), 运维工具 (部署相关)
**工作量:** ~1-2 天

### 后端任务

| # | 任务 | P | 产出文件 | 说明 |
|---|------|---|---------|------|
| BE-8.1 | 备份恢复模块 | P0 | `internal/omcr/backup/` (model.go, repository.go, pg_repository.go, handler.go, service.go) | `backup_tasks` 表 (id, task_type, target_type, target_ids, status, progress, file_path, started_at, completed_at); `backup_schedules` 表 (id, name, cron_expr, enabled, task_type); MinIO 存储备份文件 |
| BE-8.2 | 备份表 Migration | P0 | `migrations/000022_create_backup_tables.up.sql` + `.down.sql` | backup_tasks, backup_schedules 表 + 索引 |
| BE-8.3 | 文件管理模块 | P1 | `internal/omcr/filemanager/` (model.go, repository.go, pg_repository.go, handler.go) | `managed_files` 表 (id, file_name, file_type, file_size, minio_path, uploader, device_sn, status, description); CRUD + 上传(multipart) + 下载(blob); 复用 MinIO client |
| BE-8.4 | 文件表 Migration | P1 | `migrations/000023_create_managed_files.up.sql` + `.down.sql` | managed_files 表 + file_type/status/device_sn 索引 |
| BE-8.5 | MML 控制台模块 | P1 | `internal/omcr/mml/` (model.go, handler.go, service.go) | `mml_commands` 表 (预定义命令); `mml_scripts` 表 (���户��本); `mml_tasks` 表 (执行历史); 通过 ACS cmdQueue 推送执行 |
| BE-8.6 | MML 表 Migration | P1 | `migrations/000024_create_mml_tables.up.sql` + `.down.sql` | mml_commands, mml_scripts, mml_tasks 表 + 种子 MML 命令数据 |
| BE-8.7 | 注册新路由 | P0 | `cmd/app/main.go` | `/backup/*`, `/files/*`, `/mml/*` 路由注册 |
| BE-8.8 | 集成测试 | P1 | `test/integration/api_sprint8_test.go` | httptest 测试：backup CRUD, 文件上传/下载, MML 命令执行 (~15-20 用例) |

### API 路由设计

**备份恢复 `/api/v1/backup/*`:**
```
GET    /backup/tasks                    — 任务列表 (filter: status, task_type)
POST   /backup/tasks                    — 创建备份任务
GET    /backup/tasks/:id                — 任务详情
DELETE /backup/tasks/:id                — 删除任务
POST   /backup/tasks/:id/cancel         — 取消任务
GET    /backup/schedules                — 计划列表
POST   /backup/schedules                — 创建计划
PUT    /backup/schedules/:id            — 更新计划
DELETE /backup/schedules/:id            — 删除计划
```

**文件管理 `/api/v1/files/*`:**
```
GET    /files                           — 文件列表 (filter: file_type, device_sn, status)
POST   /files                           — 上传文件 (multipart/form-data)
GET    /files/:id                       — 文件详情
DELETE /files/:id                       — 删除文件
GET    /files/:id/download              — 下载文件 (blob)
POST   /files/:id/distribute            — 分发文件到设备
```

**MML 控制台 `/api/v1/mml/*`:**
```
GET    /mml/commands                    — 预定义命令列表
GET    /mml/commands/:id                — 命令详情
POST   /mml/execute                     — 执行 MML 命令 (同步/异步)
GET    /mml/scripts                     — 用户脚本列表
POST   /mml/scripts                     — 创建脚本
PUT    /mml/scripts/:id                 — 更新脚本
DELETE /mml/scripts/:id                 — 删除脚本
GET    /mml/tasks                       — 执行历史
GET    /mml/tasks/:id                   — 任务详情
```

### E2E 联合调试验证 (Sprint 8)

**新增测试用例: ~25 个**

**Seed 数据补充 (`seed_e2e_testdata.sql`):**
```
Section 17: backup_tasks (2 rows: 1 completed + 1 pending)
Section 18: backup_schedules (2 rows: 1 enabled + 1 disabled)
Section 19: managed_files (3 rows: config/log/firmware)
Section 20: mml_commands (3 predefined commands)
Section 21: mml_scripts (1 user script)
```

**e2e_verify.sh 新增 Section:**

| Section | 名称 | 用例数 | 验证内容 |
|---------|------|--------|---------|
| S47 | Backup 任务 CRUD | 6 | list → create → get → verify → cancel → delete |
| S48 | Backup 计划 CRUD | 4 | list → create → update → delete |
| S49 | 文件管理 | 6 | list → upload → get → download → typeFilter → delete |
| S50 | MML 命令 | 4 | listCommands → getCommand → execute → verifyTask |
| S51 | MML 脚本 | 3 | listScripts → create → delete |
| S52 | MML 任务历史 | 2 | listTasks → taskDetail |

**e2e.md 更新:**
- 测试总数: ~260 → ~285
- 新增 Sprint 8 描述段落

### 验收标准

| 检查项 | 预期结果 |
|--------|---------|
| `POST /backup/tasks` | 创建备份任务返回 201 |
| `GET /backup/tasks?page=1&page_size=10` | 返回分页列表 |
| `POST /files` (multipart) | 文件上传到 MinIO 返回元数据 |
| `GET /files/:id/download` | 返回文件 blob |
| `POST /mml/execute` | 队列命令返回 task_id |
| 后端编译 | `go build ./...` 通过 |
| 集成测试 | 新增用例全部通过 |
| E2E | ~285/285 PASS |

---

## 四、Sprint 9 — C 类前端集成 + 收尾

**目标:** 将 Sprint 8 后端连接到前端，最终对齐率 ~88% → ~95%
**策略:** 新建 3 个 API 服务文件 + 修改 3 个 Hook 文件
**工作量:** ~1-2 天

### 前端任务

| # | 任务 | P | 产出文件 | 说明 |
|---|------|---|---------|------|
| FE-9.1 | 新建 backupApi.ts | P0 | `src/services/api/backupApi.ts` (新建) | 映射 backup_tasks/backup_schedules 后端模型到前端 BackupTask/BackupSchedule 类型; CRUD + cancel |
| FE-9.2 | useBackup: 接入 backupApi | P0 | `src/hooks/api/useBackup.ts` | 所有 `backupService` 直接调用改为 `useMock ? backupService : backupApi` |
| FE-9.3 | 新建 fileApi.ts | P0 | `src/services/api/fileApi.ts` (新建) | 文件 CRUD + multipart 上传 + blob 下载 (复用 mrApi.ts downloadFile 模式) |
| FE-9.4 | useFiles: 接入 fileApi | P0 | `src/hooks/api/useFiles.ts` | 所有 `fileService` 直接调用改为 `useMock ? fileService : fileApi` |
| FE-9.5 | 新建 mmlApi.ts | P1 | `src/services/api/mmlApi.ts` (新建) | MML 命令/脚本/任务 CRUD + 执行; 映射后端模型到前端 MMLScript/MMLTask 类型 |
| FE-9.6 | useMML: 接入 mmlApi | P1 | `src/hooks/api/useMML.ts` | 所有 `mmlService` 直接调用改为 `useMock ? mmlService : mmlApi` |
| FE-9.7 | API 入口更新 | P2 | `src/services/api/index.ts` | 导出 `backupApi`, `fileApi`, `mmlApi`, `configSyncApi` |

### 后端任务

| # | 任务 | P | 产出文件 | 说明 |
|---|------|---|---------|------|
| BE-9.1 | OpenAPI 规范补充 | P1 | `api/openapi/openapi.yaml` | Sprint 8-9 所有新端点的 paths + schemas |
| BE-9.2 | 全量回归测试 | P1 | `test/integration/api_sprint9_test.go` | Sprint 6-9 API 契约回归测试 |

### E2E 联合调试验证 (Sprint 9)

**新增测试用例: ~15 个**

**e2e_verify.sh 新增 Section:**

| Section | 名称 | 用例数 | 验证内容 |
|---------|------|--------|---------|
| S53 | Backup 前端集成 | 3 | 前端格式 CRUD 验证 |
| S54 | 文件管理前端集成 | 3 | upload/download/list 前端格式验证 |
| S55 | MML 前端集成 | 3 | execute/list/verify 前端格式验证 |
| S56 | 全量回归 | 6 | Sprint 0-9 冒烟测试 + healthz + CORS + mock 模式 + 构建验证 |

**e2e.md 最终更新:**
- 测试总数: ~285 → ~300
- 新增 Sprint 9 描述段落
- 更新总覆盖范围描述

### 验收标准

| 检查项 | 预期结果 |
|--------|---------|
| 备份页面 (`VITE_USE_MOCK=false`) | 任务列表、计划列表显示真实数据 |
| 文件管理 real 模式 | 上传→列表出现→下载正确 |
| MML 控制台 real 模式 | 命令列表加载 + 执行返回 task_id |
| Mock 全量回归 | `VITE_USE_MOCK=true` 全部 94+ 页面正常 |
| 前端构建 | `tsc --noEmit` (0 errors) + `vite build` 通过 |
| 后端编译 | `go build ./...` 通过 |
| E2E | ~300/300 PASS |
| 对齐率 | ~95% |

---

## 五、对齐率进展

```
Sprint 5 (当前):   ~68%  (64 已对齐端点 / ~110 总端点)
Sprint 6:          ~80%  (+20 端点连接, 0 新后端)
Sprint 7:          ~85%  (+4 新 Dashboard 端点, config sync 连接)
Sprint 8:          ~88%  (+15 新后端端点: backup/files/MML)
Sprint 9:          ~95%  (前端连接 Sprint 8 后端)
```

**剩余 ~5% 保持 Mock (设计决策, 低优先级):**
- License 管理 (3 页面) — 静态数据，访问频率低
- 报表生成 (4 页面) — 可从已有聚合 API 前端组装
- 运维工具 (5 页面) — 工具脚本，因部署而异
- 拓扑 GIS/图 (3 函数) — 需要地图基础设施
- 配置基线/邻区参数 (2 函数) — 专用 TR069 特性

---

## 六、文件变更汇总

### Sprint 6 (纯前端, ~10 文件)

| 类型 | 文件路径 |
|------|---------|
| 修改 | `src/services/api/adminApi.ts` |
| 修改 | `src/services/api/pmApi.ts` |
| 修改 | `src/services/api/topologyApi.ts` |
| 修改 | `src/services/api/deviceApi.ts` |
| 修改 | `src/hooks/api/useSystem.ts` |
| 修改 | `src/hooks/api/usePerformance.ts` |
| 修改 | `src/hooks/api/useNEs.ts` |
| 新建 | `src/services/api/configSyncApi.ts` |
| 修改 | `omcgo/scripts/e2e_verify.sh` (+~200 行) |
| 修改 | `.claude/commands/e2e.md` |

### Sprint 7 (前端 + 后端, ~15 文件)

| 类型 | 文件路径 |
|------|---------|
| 修改 | `omcgo/internal/omcr/dashboard/handler.go` |
| 修改 | `omcgo/internal/omcr/dashboard/service.go` |
| 修改 | `omcgo/cmd/app/main.go` |
| 修改 | `src/services/api/dashboardApi.ts` |
| 修改 | `src/services/api/datamodelApi.ts` |
| 修改 | `src/hooks/api/useDashboard.ts` |
| 修改 | `src/hooks/api/useConfig.ts` |
| 修改 | `src/hooks/api/useTopology.ts` |
| 修改 | 设备列表/详情页组件 |
| 修改 | `omcgo/scripts/e2e_verify.sh` (+~150 行) |
| 修改 | `omcgo/scripts/seed_e2e_testdata.sql` (+~10 行) |

### Sprint 8 (纯后端, ~20 文件)

| 类型 | 文件路径 |
|------|---------|
| 新建 | `omcgo/internal/omcr/backup/` (5 文件) |
| 新建 | `omcgo/internal/omcr/filemanager/` (4 文件) |
| 新建 | `omcgo/internal/omcr/mml/` (3 文件) |
| 新建 | `omcgo/migrations/000022_*` (2 文件) |
| 新建 | `omcgo/migrations/000023_*` (2 文件) |
| 新建 | `omcgo/migrations/000024_*` (2 文件) |
| 修改 | `omcgo/cmd/app/main.go` |
| 新建 | `omcgo/test/integration/api_sprint8_test.go` |
| 修改 | `omcgo/scripts/e2e_verify.sh` (+~180 行) |
| 修改 | `omcgo/scripts/seed_e2e_testdata.sql` (+~40 行) |

### Sprint 9 (前端 + 收尾, ~12 文件)

| 类型 | 文件路径 |
|------|---------|
| 新建 | `src/services/api/backupApi.ts` |
| 新建 | `src/services/api/fileApi.ts` |
| 新建 | `src/services/api/mmlApi.ts` |
| 修改 | `src/hooks/api/useBackup.ts` |
| 修改 | `src/hooks/api/useFiles.ts` |
| 修改 | `src/hooks/api/useMML.ts` |
| 修改 | `src/services/api/index.ts` |
| 修改 | `omcgo/api/openapi/openapi.yaml` |
| 新建 | `omcgo/test/integration/api_sprint9_test.go` |
| 修改 | `omcgo/scripts/e2e_verify.sh` (+~100 行) |
| 修改 | `.claude/commands/e2e.md` (最终版) |

---

## 七、E2E Skill 更新计划

每个 Sprint 完成后需更新 `.claude/commands/e2e.md`:

| Sprint | 更新内容 |
|--------|---------|
| Sprint 6 | 测试总数 210→~240; 新增 Sprint 6 段落 (8 section, ~30 用例); 更新 Seed 数据描述 |
| Sprint 7 | 测试总数 ~240→~260; 新增 Sprint 7 段落 (8 section, ~20 用例); 新增 Dashboard 端点覆盖 |
| Sprint 8 | 测试总数 ~260→~285; 新增 Sprint 8 段落 (6 section, ~25 用例); 新增 backup/files/mml 覆盖; 更新 Seed 数据描述 (新增 5 类数据) |
| Sprint 9 | 测试总数 ~285→~300; 新增 Sprint 9 段落 (4 section, ~15 用例); 最终版 — 完整覆盖描述 |

---

## 八、关键依赖链

```
Sprint 6 (无阻塞):
  所有后端端点已存在 → FE-6.1~6.10 可全部并行开发

Sprint 7:
  BE-7.1~7.4 (Dashboard 端点) → FE-7.1 (dashboardApi 替换)
  FE-6.10 (configSyncApi) → FE-7.3 (useConfig 对接)
  FE-6.7 (topologyApi CRUD) → FE-7.4 (useTopology hooks)

Sprint 8 (无前端依赖):
  BE-8.2 (migration) → BE-8.1 (backup module)
  BE-8.4 (migration) → BE-8.3 (file module)
  BE-8.6 (migration) → BE-8.5 (MML module)
  BE-8.1+8.3+8.5 → BE-8.7 (路由注册)

Sprint 9:
  BE-8.* (全部后端) → FE-9.1~9.7 (前端集成)
```

**并行优化:** Sprint 8 的 3 个模块 (backup/files/MML) 彼此独立，可并行开发。

---

## 九、验证方案汇总

| Sprint | 验证步骤 |
|--------|---------|
| Sprint 6 | `tsc --noEmit` + `vite build` → E2E (~240/240) → mock 回归 |
| Sprint 7 | `go build ./...` + `tsc --noEmit` + `vite build` → E2E (~260/260) → Dashboard 页面真实数据验证 → mock 回归 |
| Sprint 8 | `go build ./...` + 集成测试 → E2E (~285/285) |
| Sprint 9 | `go build ./...` + `tsc --noEmit` + `vite build` → E2E (~300/300) → 全量 mock 回归 → 生产构建 |
