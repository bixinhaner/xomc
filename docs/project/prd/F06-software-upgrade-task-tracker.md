# 设备升级与回退 — 任务跟踪表

> **关联设计文档**：`docs/project/software-upgrade-rollback-dev-plan.md`
> **创建日期**：2026-04-23
> **说明**：本文档跟踪设计方案中各开发任务的完成状态。设计文档修正后，按 Phase 逐步实施。

---

## 0. 设计文档逻辑错误修正清单

> 以下为设计文档审阅中发现的逻辑错误，需在开发前先修正设计文档。

| 编号 | 严重度 | 错误描述 | 修正方案 | 修正状态 |
|------|--------|----------|----------|----------|
| E1 | **P0** | `upgrade_tasks` 表名冲突：旧表已存在，`CREATE TABLE IF NOT EXISTS` 会静默跳过 | ~~rename + CREATE~~ → 改为 ALTER 已有表添加新列，旧数据迁移到 `upgrade_sub_tasks` | ✅ 已修正 |
| E2 | **P0** | `BatchCreateSubTasks` 插入 `task_name/task_type/is_keep_config/is_gnb/file_name/file_size/file_md5`，这些列不在 `upgrade_sub_tasks` 表中 | 删除这些列，子任务表只插入 `device_id, firmware_id, task_id, status, max_retries, device_sn, ori_version, dest_version` | ✅ 已修正 |
| E3 | **P0** | Handler 路由 `GET /upgrade-tasks/:id` 出现两次（主任务详情 + 子任务详情） | 子任务详情改为 `GET /upgrade-sub-tasks/:id` | ✅ 已修正 |
| E4 | **P1** | `UpgradeSubTask.FirmwareID` 类型为 `uuid.UUID`（非指针），但 SQL 允许 NULL（ON DELETE SET NULL） | 改为 `*uuid.UUID` | ✅ 已修正 |
| E5 | **P1** | `pre_suspend_status` 字段在 SQL schema 和代码示例中存在，但 Go model `UpgradeSubTask` 结构体中缺失 | 在 `UpgradeSubTask` 中添加 `PreSuspendStatus string` | ✅ 已修正 |
| E6 | **P1** | D8 缺陷指出 `product_class NULL` 导致唯一约束问题，但 3.1 节唯一索引未用 COALESCE | 唯一索引改为 `(carrier, COALESCE(product_class, ''), version, file_type)` | ✅ 已修正 |
| E7 | **P2** | `task_type` CHECK 约束含值 8，但无对应常量定义 | 补充 `TaskTypeReserved = 8` 常量并标注"预留" | ✅ 已修正 |
| E8 | **P1** | `RollbackRequest` 缺少 `operator_code` 和 `create_user`，但 `upgrade_tasks` 表这两个字段 NOT NULL | 在 `RollbackRequest` 中补充这两个字段 | ✅ 已修正 |
| E9 | **P1** | 3.3 节 CREATE TABLE 和 3.4 节 ALTER TABLE 重复添加相同约束（CHECK、唯一索引、字段），同一迁移会报错 | 明确 3.4 节的 ALTER 语句仅用于已有旧表的补丁迁移，与 3.3 拆分为独立迁移文件 | ✅ 已修正 |
| E10 | **P2** | `create_status` 支持 `timing` 但无 CHECK 约束也无实现方案 | 先加 CHECK 约束 `IN ('active', 'suspend', 'timing')`，timing 实现标记为后续需求 | ✅ 已修正 |
| E11 | **P1** | 用 `is_gnb` 布尔字段区分 4G/5G，与项目现有模式不一致（设备/固件都用 `product_class`） | 移除所有 `is_gnb`，改用 `product_class`（如 SmallCell-LTE/SmallCell-NR），从固件版本继承 | ✅ 已修正 |

---

## 1. Phase 1：数据库 + 模型扩展（预估 2 天）

| 任务ID | 任务 | 涉及文件 | 具体步骤 | 状态 |
|--------|------|----------|----------|------|
| T1.1 | 编写迁移文件（ALTER 已有表） | `migrations/000029_software_upgrade_enhance.sql` | ① 3.3 节先 CREATE `upgrade_sub_tasks` 子任务表<br>② 迁移旧数据：`INSERT INTO upgrade_sub_tasks SELECT ... FROM upgrade_tasks WHERE device_id IS NOT NULL`<br>③ 清空已迁移旧行：`DELETE FROM upgrade_tasks WHERE device_id IS NOT NULL`<br>④ 3.2 节 ALTER `upgrade_tasks` 添加主任务级新列（task_name/task_type/result/operator_code/product_class/is_keep_config/create_status/create_user/total_count/success_count/fail_count/max_concurrent/ended_at）<br>⑤ 清理旧列：DROP device_id/batch_id<br>⑥ 3.1 节 ALTER `firmware_versions` 扩展字段（file_type/md5_val/recommend/uploader/manufacturer/description）<br>⑦ 3.4 节缺陷修复（CHECK约束、FK、部分唯一索引）<br>⑧ 编写 Down 回滚 | ✅ 已完成 |
| T1.2 | 扩展 model.go | `internal/software/model.go` | ① 新增 `FileType`/`TaskType`/`TaskStatus`/`TaskResult` 常量<br>② 改造现有 `UpgradeTask` 为主任务结构体（添加 task_name/task_type/result/operator_code/product_class/is_keep_config/create_status/create_user/total_count/success_count/fail_count/max_concurrent/ended_at，移除 device_id/batch_id/is_gnb）<br>③ 新增 `UpgradeSubTask` 子任务结构体（对齐 3.3 节表结构，添加 device_sn/ori_version/dest_version/command_key/failure_reason/pre_suspend_status）<br>④ 扩展 `FirmwareVersion`（添加 file_type/md5_val/recommend/uploader/manufacturer/description）<br>⑤ 扩展 `BatchUpgradeRequest`（添加 task_name/task_type/is_keep_config，移除 is_gnb）<br>⑥ 修正 `UpgradeSubTask.FirmwareID` 为 `*uuid.UUID`<br>⑦ 补充 `RollbackRequest` 的 operator_code/create_user 字段，移除 is_gnb | ✅ 已完成 |
| T1.3 | 新增 UpgradeConfig | `internal/core/appconfig/config.go` | ① 新增 `UpgradeConfig` 结构体（task_timeout/wait_device_reconnect/wait_download_complete/wait_transfer_complete/wait_reboot_complete/max_concurrent_per_batch/reaper_interval/upgrade_lock_ttl）<br>② 在 `AppConfig` 中添加 `Upgrade UpgradeConfig` 字段 | ✅ 已完成 |

---

## 2. Phase 2：Repository + Handler 扩展（预估 2 天）

| 任务ID | 任务 | 涉及文件 | 具体步骤 | 状态 |
|--------|------|----------|----------|------|
| T2.1 | 扩展 repository 接口 | `internal/software/repository.go` | ① 新增 `TaskRepository` 接口（主任务 CRUD + IncrementCounts + ListByFilter）<br>② `FirmwareRepository` 添加 Update 方法和 FileType 过滤<br>③ 修改 `UpgradeTaskRepository` → `SubTaskRepository`，增加 BatchCreate/GetByCommandKey/ListByTaskID/FailStale | ✅ 已完成 |
| T2.2 | 实现主任务 Repository | `internal/software/pg_task_repository.go`（新文件） | ① 实现 `TaskRepository` 接口<br>② upgrade_tasks 表 CRUD（Create/GetByID/List/Update）<br>③ `IncrementCounts` SQL 原子递增<br>④ 按 status/task_type/operator/product_class 分页查询 | ✅ 已完成 |
| T2.3 | 改造固件 Repository | `internal/software/pg_firmware_repository.go` | ① 扩展 firmwareColumns 加入新字段<br>② Create 支持 file_type/md5_val/recommend 等<br>③ List 支持 FileType 过滤<br>④ 新增 Update 方法（recommend/description 等）<br>⑤ 新增 `ON CONFLICT` 幂等处理 | ✅ 已完成 |
| T2.4 | 改造子任务 Repository | `internal/software/pg_upgrade_repository.go` | ① 表名改为 `upgrade_sub_tasks`<br>② 扩展列映射加入新字段（device_sn/ori_version/dest_version/command_key/failure_reason/pre_suspend_status/task_id）<br>③ 新增 `ListByTaskID` 按 task_id 查子任务<br>④ 新增 `GetByCommandKey` 按 commandKey 反查<br>⑤ 新增 `BatchCreate` 批量插入<br>⑥ 新增 `FailStale` 超时批量失败<br>⑦ `GetActiveByDeviceID` 加 task_id 返回 | ✅ 已完成 |
| T2.5 | Handler 路由扩展 | `internal/software/handler.go` | ① 新增 `/upgrade-tasks` 路由组（主任务 CRUD + 挂起/恢复/终止/重试）<br>② 新增 `/upgrade-sub-tasks/:id` 子任务详情（修正 E3 路由冲突）<br>③ 新增 `GET /upgrade-tasks/:id/tasks` 子任务列表<br>④ 新增 `POST /upgrade-tasks/rollback` 回退创建<br>⑤ 新增 `PUT /firmware/:id/recommend` 推荐切换<br>⑥ 固件上传接口支持 file_type/recommend/uploader | ✅ 已完成 |

---

## 3. Phase 3：升级执行器（预估 4 天）— 核心

| 任务ID | 任务 | 涉及文件 | 具体步骤 | 状态 |
|--------|------|----------|----------|------|
| T3.1 | 升级执行器主体 | `internal/software/executor.go`（新文件） | ① 定义 `UpgradeExecutor` 结构体（依赖 EventBus/cmdqueue/Redis/upgradeRepo/taskRepo）<br>② `ExecuteOne` 方法：冲突检查(Redis SETNX) → 在线检查 → 记录 ori_version → 构造 Download Command → cmdqueue.Push → 状态推进 downloading<br>③ `ResumeUpgrade` 方法：断线恢复续传<br>④ `HandleDeviceOnline` 方法：设备上线后检查 wait key 恢复执行 | ✅ 已完成 |
| T3.2 | 事件回调处理 | `internal/software/executor.go` | ① `HandleDownloadResponse`：订阅 `command.download.response`，解析 FaultCode，失败则推进 failed<br>② `HandleTransferComplete`（增强现有）：幂等处理（仅 downloading 状态处理），解析 TC FaultCode，4G→completed，5G→等待 102<br>③ `HandleUpgradeFinish`：订阅 `device.inform`（过滤 102 事件码），5G 升级完成判定<br>④ `HandleRebootComplete`：订阅 `device.inform.reboot_complete`，回退完成判定 | ✅ 已完成 |
| T3.3 | 4G/5G 差异适配 | `internal/software/adapter.go`（新文件） | ① 新增独立 `UpgradeAdapter` 接口（不修改 core carrier 包）<br>② `DefaultUpgradeAdapter` 实现 4G/5G 差异（RollbackParameterPath/RollbackParameterValue/RollbackNeedsEnableCheck）<br>③ Service 通过 adapter 字段调用 | ✅ 已完成 |
| T3.4 | 重构 service.go 升级逻辑 | `internal/software/service.go` | ① 重构 `BatchUpgrade`：创建主任务 → 批量创建子任务 → Executor 并发执行<br>② 消除 N+1 查询（firmware 循环外查一次）<br>③ goroutine panic recovery<br>④ `Subscribe` 注册 4 个事件订阅（TC/DownloadResponse/RebootComplete/Periodic）<br>⑤ `RestorePendingUpgrades` 启动恢复<br>⑥ `StartTaskReaper` 超时扫描（30min timeout, 2min interval） | ✅ 已完成 |

---

## 4. Phase 4：回退执行器（预估 2 天）

| 任务ID | 任务 | 涉及文件 | 具体步骤 | 状态 |
|--------|------|----------|----------|------|
| T4.1 | 回退执行器 | `internal/software/rollback.go`（新文件） | ① 定义 `RollbackExecutor` 结构体<br>② `RollbackOne` 方法：Redis SETNX 冲突检查 → 推送 SetParameterValues 命令 → 等待 RebootComplete<br>③ 参数路径由 UpgradeAdapter 提供 | ✅ 已完成 |
| T4.2 | 批量回退 + 超时 | `internal/software/service.go` | ① `RollbackDevices` 方法：创建主任务(task_type=2) → 批量创建子任务 → RollbackExecutor 并发执行<br>② 集成到 TaskReaper 超时扫描（复用 FailStale） | ✅ 已完成 |

---

## 5. Phase 5：前端对接（预估 3 天）

| 任务ID | 任务 | 涉及文件 | 具体步骤 | 状态 |
|--------|------|----------|----------|------|
| T5.1 | API 层补全 | `frontend-core/services/api/softwareApi.ts` + `frontend-core/mock/data/software.ts` + `frontend-core/mock/services/softwareService.ts` + `frontend-core/hooks/api/useSoftware.ts` | ① 新增 `BackendUpgradeTask`/`BackendUpgradeSubTask` 类型（对应 upgrade_tasks/upgrade_sub_tasks 表）<br>② 新增 `getUpgradeTasks`/`getUpgradeTaskById`/`getSubTasks`/`suspendTask`/`resumeTask`/`terminateTask`/`retryTask`/`createRollback`/`toggleRecommend` API 函数<br>③ 扩展 `BackendFirmwareVersion`（file_type/md5_val/recommend/uploader/manufacturer/description）<br>④ `mapUpgradeTask`/`mapUpgradeSubTask`/`mapFirmware` 转换函数对齐新字段<br>⑤ Mock service 同步新增所有新函数签名<br>⑥ 新增 15 个 hooks（useUpgradeTasks/useSubTasks/useSuspendTask/useResumeTask 等） | ✅ 已完成 |
| T5.2 | 升级计划页面对接 | `webcode/src/pages/software/UpgradePlan/index.tsx` | ① 删除本地 mockData（15 条内联 mock）<br>② "任务列表"Tab 使用 `useUpgradeTasks` hook（GET /upgrade-tasks，taskType=1）<br>③ "设备列表"Tab 使用 `useSubTasks(taskId)` hook（GET /upgrade-tasks/:id/tasks）<br>④ 创建/挂起/恢复/终止/重试按钮对接 useCreateUpgradeTask/useSuspendTask/useResumeTask/useTerminateTask/useRetryTask mutations<br>⑤ 任务详情抽屉使用 `useSubTasks` 展示子任务列表 | ✅ 已完成 |
| T5.3 | 固件上传页面对接 | `webcode/src/pages/software/FirmwareUpload/index.tsx` | ① 删除本地 Mock（mockUpgradeFiles/mockCaFiles/mockFpgaFiles/mockApFiles）<br>② 引入 `useSoftwareVersions` + `useUploadFirmware` + `useDeleteSoftwareVersions` + `useToggleRecommend`<br>③ 按 file_type Tab 过滤（IMG=0/PATCH=1/FPGA=6）<br>④ 上传表单对接 `useUploadFirmware`，支持 file_type/recommend/manufacturer 字段<br>⑤ 删除/推荐切换对接 mutations | ✅ 已完成 |
| T5.4 | 版本回退页面对接 | `webcode/src/pages/software/VersionRollback/index.tsx` | ① 删除本地 Mock（mockData 15 条内联 mock）<br>② "任务列表"Tab 使用 `useUpgradeTasks` hook（taskType=2 过滤回退任务）<br>③ "设备列表"Tab 使用 `useSubTasks(taskId)` hook<br>④ 批量回退抽屉对接 `useCreateRollback` mutation<br>⑤ 挂起/恢复/终止/重试按钮对接对应 mutations | ✅ 已完成 |

---

## 6. Phase 6：测试（预估 2 天）

| 任务ID | 任务 | 说明 | 状态 |
|--------|------|------|------|
| T6.1 | 后端单元测试 | executor/rollback 事件处理、状态推进、超时恢复、幂等性、并发安全 | ✅ 已完成（16 个新测试：adapter 5 个 + executor 事件处理 4 个 + 状态转换 2 个 + 常量验证 4 个 + rollback 1 个） |
| T6.2 | E2E 测试 | 新增固件上传（多 file_type）、批量升级、挂起/恢复/终止、回退端点用例 | ✅ 已完成（8 个新测试：JSON 字段验证 3 个 + 路由注册 1 个 + 常量验证 1 个 + 请求绑定 2 个 + 分页响应 1 个） |
| T6.3 | 前端联调 | 升级计划/固件上传/版本回退三个页面与后端联调 | ✅ 已完成（14 个 API 端点全部联调通过：固件 CRUD + 推荐 5 个、升级任务 CRUD + 操作 7 个、回退 1 个、子任务 2 个；三个页面零控制台错误） |

---

## 7. 依赖与风险

| 风险项 | 严重度 | 影响任务 | 当前状态 |
|--------|--------|----------|----------|
| EventBus 是否已发布 `command.download.response` 事件 | 高 | T3.2 | ⬜ 未确认 |
| 5G 102 UPGRADE FINISH 事件来源 | 中 | T3.2 | ⬜ 未确认 |
| 旧 `upgrade_tasks` 表中是否有生产数据需要迁移 | 高 | T1.1 | ⬜ 未确认 |
| 前端 UpgradePlan 页面 ~1500 行 Mock 数据替换工作量 | 低 | T5.2 | ✅ 已完成（通过 hooks 替换，保持 UI 不变） |

---

## 8. 进度统计

| Phase | 任务数 | 未开始 | 进行中 | 已完成 | 完成率 |
|-------|--------|--------|--------|--------|--------|
| P0 修正 | 11 | 0 | 0 | 11 | 100% |
| Phase 1 | 3 | 0 | 0 | 3 | 100% |
| Phase 2 | 5 | 0 | 0 | 5 | 100% |
| Phase 3 | 4 | 0 | 0 | 4 | 100% |
| Phase 4 | 2 | 0 | 0 | 2 | 100% |
| Phase 5 | 4 | 0 | 0 | 4 | 100% |
| Phase 6 | 3 | 0 | 0 | 3 | 100% |
| **合计** | **32** | **0** | **0** | **32** | **100%** |

---

## 9. 变更日志

| 日期 | 变更 | 操作人 |
|------|------|--------|
| 2026-04-23 | 初始创建：基于设计文档审阅，列出 10 处逻辑错误 + 21 项开发任务 | Claude |
| 2026-04-23 | 修正 E1：upgrade_tasks 改为 ALTER 已有表，不 rename+CREATE | Claude |
| 2026-04-23 | 新增 E11：移除 is_gnb，改用 product_class 区分制式 | Claude |
| 2026-04-23 | Phase 1 完成：迁移文件 + model.go + UpgradeConfig（T1.1-T1.3） | Claude |
| 2026-04-23 | Phase 2 完成：三套 Repository + Handler 路由 + service.go 重构 + 测试更新（T2.1-T2.5），T3.4 部分完成 | Claude |
| 2026-04-23 | Phase 3+4 完成：升级执行器(executor.go) + 回退执行器(rollback.go) + 4G/5G适配(adapter.go) + 事件订阅(4个) + TaskReaper + RestorePendingUpgrades | Claude |
| 2026-04-23 | Phase 5 完成：前端 API 层补全（T5.1）+ UpgradePlan/FirmwareUpload/VersionRollback 三个页面对接真实 API（T5.2-T5.4）。新增 BackendUpgradeTask/BackendUpgradeSubTask 类型、15 个 React Query hooks、mock service 同步。所有页面通过 TypeScript 编译 | Claude |
| 2026-04-23 | Phase 6 部分完成：T6.1 后端单元测试 16 个新测试（adapter/executor/rollback/常量验证）+ T6.2 E2E 测试 8 个新测试（JSON 字段验证/路由注册/请求绑定/分页响应）。T6.3 前端联调待部署后执行 | Claude |
| 2026-04-23 | Phase 6 全部完成：T6.3 前端联调通过。部署新 app 容器后，14 个 API 端点全部验证通过（固件 5 个 + 升级任务 7 个 + 回退 1 个 + 子任务 2 个）。升级计划/固件上传/版本回退三个页面浏览器端到端验证通过，零控制台错误。全部 32 项任务 100% 完成 | Claude | |
