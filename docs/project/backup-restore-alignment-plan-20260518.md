# 配置文件备份与恢复 — 规范对齐计划（2026-05-18）

> 规范来源：[docs/files/Back-end/基站配置文件备份与恢复功能流程.md](../files/Back-end/基站配置文件备份与恢复功能流程.md)
> 涉及模块：`omcgo/internal/backup/` · `omcgo/internal/transfer/` · `omcgo/internal/acs/` · `omcgo/internal/task/`
> 模板模块（`omcgo/internal/config/template/`）经核对**与备份/恢复无耦合**（命名空间隔离，无相互引用），本计划不涉及对模板模块的修改。

---

## 一、整体结论

当前实现已具备备份/恢复的核心骨架（事件驱动执行 + MinIO 文件存储 + 策略清理告警），但**协议外观、表结构、调度机制、命令命名**等多处与规范不对齐。整体成熟度约 60%，需补齐的关键短板集中在：

| 维度 | 现状 | 规范 | 偏差等级 |
|------|------|------|---------|
| 任务表结构 | `backup_tasks` UUID 主键 / 字符串状态 / 无 `task_result` | `backup_restore_task` 整数 task_id / 七态整数 / `task_result` 1\|2 | 高 |
| 文件元数据表 | **缺失** | `backup_restore_file`（SN/file_name/md5/size/operator_code） | 高 |
| 周期任务 | `backup_schedules` 配置已存，**无 Cron 调度器** | `backup_period_task` + `BackupPeriodTaskJob` | 高 |
| 触发机制 | 事件驱动（`SubjectBackupTaskCreated`） | `BackupRestoreTaskJob` 扫表 0→2 | 中 |
| 文件传输通道 | MinIO + presigned HTTP URL | FTP（含 Username/Password） | 中 |
| Upload/Download CommandKey | `backup-<uuid前8>` / 恢复侧未显式构造 | `{cellCode}_BACKUP` / `{cellCode}_RESTORE` | 高 |
| FileType | `"3"` | `3 Vendor Configuration File` | 低（值正确） |
| TransferComplete 状态回写 | 仅发事件，未按 CommandKey 回写备份/恢复任务 | 按 CommandKey 关联回写 | 高 |
| 运营商编码 | 任务表无 `operator_code` | 必须含 `operator_code` | 中 |
| REST API 前缀 | `/api/v1/backup/*` | `/task/enb/config/backupRestore/*` | 中 |
| 过期清理 | `PolicyMonitor`（T-0073） | `DeleteExpireBackupRestoreJob` | ✓ 已满足 |
| FileType=3 | ✓ | ✓ | ✓ 已满足 |

---

## 二、当前实现盘点

### 2.1 模块组成（`omcgo/internal/backup/`）

| 层 | 文件 | 关键类型/方法 |
|----|------|--------------|
| 模型 | [model.go](../../omcgo/internal/backup/model.go) | `BackupTask` `BackupSchedule` `TaskStatus`（字符串） |
| 模型 | [restore_model.go](../../omcgo/internal/backup/restore_model.go) | `RestoreTask` `RestoreStatus` |
| 模型 | [policy_model.go](../../omcgo/internal/backup/policy_model.go) | `BackupPolicy`（T-0071 保留/压缩/加密策略） |
| 模型 | [ftp_model.go](../../omcgo/internal/backup/ftp_model.go) | `FTPConfig` |
| 业务 | [service.go](../../omcgo/internal/backup/service.go) | `BackupService`（创建/查询/取消） |
| 业务 | [restore_service.go](../../omcgo/internal/backup/restore_service.go) | `RestoreService.Create` 扇出 device_tasks |
| 业务 | [executor.go](../../omcgo/internal/backup/executor.go) | `BackupExecutor.handleTaskCreated` 订阅 `SubjectBackupTaskCreated` |
| 业务 | [policy_service.go](../../omcgo/internal/backup/policy_service.go) | 策略 CRUD |
| 业务 | [policy_monitor.go](../../omcgo/internal/backup/policy_monitor.go) | 过期清理 + 失败告警（T-0073/0076/0083） |
| 业务 | [file_path_recorder.go](../../omcgo/internal/backup/file_path_recorder.go) | 备份文件路径记录器（**未持久 MD5/SN 元数据**） |
| 仓储 | [pg_repository.go](../../omcgo/internal/backup/pg_repository.go) | `backup_tasks` / `backup_schedules` CRUD |
| 仓储 | [restore_pg_repository.go](../../omcgo/internal/backup/restore_pg_repository.go) | `restore_tasks` CRUD |
| 仓储 | [ftp_pg_repository.go](../../omcgo/internal/backup/ftp_pg_repository.go) | `FTPConfig` CRUD |
| 仓储 | [policy_pg_repository.go](../../omcgo/internal/backup/policy_pg_repository.go) | 策略持久化 |
| 基础设施 | [ftp_tester.go](../../omcgo/internal/backup/ftp_tester.go) | FTP/SFTP/FTPS 连接测试（T-0032/0093，仅测试，未参与实际传输） |
| 基础设施 | [compression.go](../../omcgo/internal/backup/compression.go) [encryption.go](../../omcgo/internal/backup/encryption.go) [key_provider*.go](../../omcgo/internal/backup/key_provider.go) [reencryptor.go](../../omcgo/internal/backup/reencryptor.go) | T-0074/0075 备份压缩加密、密钥轮换 |
| HTTP | [handler.go](../../omcgo/internal/backup/handler.go) | 备份/计划路由 |
| HTTP | [restore_handler.go](../../omcgo/internal/backup/restore_handler.go) | 恢复路由 |

### 2.2 当前数据库表（迁移）

| 表 | 迁移 | 主要字段 |
|----|------|----------|
| `backup_tasks` | [000006](../../omcgo/migrations/000006_alarms_mr_firmware.sql) | `id UUID` / `task_type VARCHAR` / `status VARCHAR` / `progress` / `file_path` / `started_at` / `completed_at` |
| `backup_schedules` | [000006](../../omcgo/migrations/000006_alarms_mr_firmware.sql) | `id UUID` / `name` / `cron_expr` / `enabled` / `task_type` / `target_type` / `target_ids` |
| `restore_tasks` | [000048](../../omcgo/migrations/000048_restore_tasks.sql) | `id UUID` / `source_bucket` / `source_object_path` / `target_device_sns` / `status` |
| `backup_policies` | [000047](../../omcgo/migrations/000047_backup_policies.sql) | T-0071 策略表 |

### 2.3 当前传输与 RPC 链路

```
[用户] → POST /api/v1/backup/tasks
       → BackupService.Create
       → 写 backup_tasks(status="pending")
       → EventBus.Publish(SubjectBackupTaskCreated)
[Worker] BackupExecutor.handleTaskCreated
       → 对每台设备调用 task.Enqueuer
       → 生成 device_tasks(Upload RPC, FileType="3",
                            CommandKey="backup-<uuid8>")
[ACS]  下发 Upload SOAP → CPE
[CPE]  上传至 MinIO presigned PUT URL (HTTP, 非 FTP)
[CPE]  Inform: TransferComplete → ACS handler
       → 发布 SubjectDeviceTransferComplete 事件
       → ❌ 当前**未**按 CommandKey 回写 backup_tasks.status / restore_tasks.status
```

---

## 三、差距详表

### 3.1 数据库字段差距

#### `backup_restore_task`（规范）vs `backup_tasks` + `restore_tasks`（现状）

| 规范字段 | 现状 | 缺失/差异 |
|---------|------|----------|
| `task_id` int PK 自增 | `id UUID` | 主键类型变更 |
| `task_name` | 无 | **缺失** |
| `task_type` (1备份/2恢复) | `task_type VARCHAR("full"\|"incremental"\|"config_only")` | 语义不符；且备份/恢复目前**分两张表**，规范是**一张表用 task_type 区分** |
| `task_status` (0-6) | `status VARCHAR("pending"\|"running"\|"completed"\|"failed"\|"cancelled")` | 仅 5 态字符串；缺 "等待中(1)" "已暂停(3)" "终止中(5)" "暂停中(6)" |
| `task_result` (1成功/2失败) | 无 | **缺失** |
| `start_time` / `end_time` | `started_at` / `completed_at` | 命名差异 |
| `create_time` / `create_user` | `created_at` / 无 | `create_user` **缺失** |
| `operator_code` | 无 | **缺失** |

#### `backup_restore_file`（规范）— **完全缺失**

需新建表，含 `serial_number` `file_name` `update_time` `md5` `file_size` `operator_code`。当前 [file_path_recorder.go](../../omcgo/internal/backup/file_path_recorder.go) 仅记录文件路径，**未计算/持久化 MD5 和文件大小**。

#### `backup_period_task`（规范）vs `backup_schedules`（现状）

| 规范字段 | 现状 | 差异 |
|---------|------|------|
| `task_id` int PK | `id UUID` | 类型 |
| `task_name` | `name` | 命名 |
| `cron_expression` | `cron_expr` | 命名 |
| `is_enable` | `enabled` | 命名 + 类型（0/1 vs bool） |
| `create_time` | `created_at` | 命名 |

### 3.2 协议/命令命名差距

| 项 | 规范 | 现状 | 文件:行 |
|----|------|------|--------|
| Upload CommandKey | `{cellCode}_BACKUP` | `"backup-" + uuid前8位` | [executor.go:171](../../omcgo/internal/backup/executor.go#L171) |
| Download CommandKey | `{cellCode}_RESTORE` | 未显式构造 | [restore_service.go](../../omcgo/internal/backup/restore_service.go) |
| FileType 值 | `3 Vendor Configuration File`（完整字符串）| `"3"` | [executor.go:147](../../omcgo/internal/backup/executor.go#L147) |
| 传输 URL Scheme | `ftp://...` + Username/Password | MinIO presigned `https://...` | [transfer/bridge.go](../../omcgo/internal/transfer/bridge.go) |
| Download 必含参数 | URL/Username/Password/**FileSize/TargetFileName/DelaySeconds** | FileSize/TargetFileName 来源不清晰 | [restore_service.go](../../omcgo/internal/backup/restore_service.go) |

### 3.3 调度与回写差距

| 规范 Job | 现状 | 结论 |
|---------|------|------|
| `BackupRestoreTaskJob`（扫 status=0→2） | 事件驱动 `SubjectBackupTaskCreated`，无扫表兜底 | **缺失**（事件丢失即丢任务） |
| `BackupPeriodTaskJob`（按 cron 生任务） | 仅有 schedule CRUD，无后台 cron 调度器 | **完全缺失**（周期备份不工作） |
| `DeleteExpireBackupRestoreJob` | `PolicyMonitor` 已实现 | ✓ 满足 |
| TransferComplete → 任务回写 | 仅发事件，无按 CommandKey 匹配回写 backup/restore_tasks 的处理器 | **缺失** |

### 3.4 REST API 差距

| 规范路径 | 现状 | 状态 |
|---------|------|------|
| `POST /task/enb/config/backupRestore/addBackupRestoreTask` | `POST /api/v1/backup/tasks` | 路径不同 |
| `POST .../queryCellInfos` | 无 | **缺失** |
| `POST .../queryTaskList` | `GET /api/v1/backup/tasks` | 方法+路径不同 |
| `POST .../queryTaskDeviceList` | 无（需 join device_tasks） | **缺失** |
| `GET .../getProductType` | 无 | **缺失** |
| `POST .../single/importFile` | `POST /api/v1/backup/restore` | 路径不同 |
| `GET .../single/exportFile` | 无 | **缺失** |
| `POST .../periodTask` | `POST /api/v1/backup/schedules` | 路径不同 |
| `POST .../terminateTask` | `DELETE /api/v1/backup/tasks/:id` | 方法+路径不同 |

---

## 四、对齐方案

### 4.1 设计决策（需评审）

下面 4 项是**架构级**决策，建议在动工前与 F02/F06 模块负责人达成一致。

| # | 决策点 | 选项 A（保留现状） | 选项 B（向规范靠拢） | 建议 |
|---|--------|------------------|--------------------|------|
| D-1 | 任务主键 | UUID（已上线，外部引用多） | 整数自增 `task_id` | **A + 暴露字段双轨**：DB 主键保留 UUID，新增 `task_seq BIGSERIAL UNIQUE` 作为对外 `task_id`，避免破坏现有引用 |
| D-2 | 备份/恢复表是否合并 | 拆两张（现状） | 合并为 `backup_restore_task` + `task_type` 区分 | **B**：合并便于复用 Job/状态机，迁移期通过 view 兼容旧表查询 |
| D-3 | 文件传输通道 | MinIO + 内置 FTP 网关 | 直连客户 FTP 服务器 | **A + B 双轨**：保留 MinIO 路径不变；额外暴露 FTP 适配（通过 `FTPConfig`），由运营商配置决定 URL Scheme |
| D-4 | API 前缀 | `/api/v1/backup/*`（现状） | `/task/enb/config/backupRestore/*` | **A + alias 路由**：保留 `/api/v1/backup/*`，同时在 router 注册 `/task/enb/config/backupRestore/*` 别名指向同 handler，前端可平滑切换 |

> 若评审决定全部采用"选项 B 严格对齐"，则需追加迁移与前端联调工时，请重新评估排期。

### 4.2 实施阶段（建议拆为 4 个里程碑）

#### 里程碑 M1 — 数据层对齐与文件元数据落库

| 任务 | 产物 |
|------|------|
| 新建迁移 `000NNN_backup_restore_alignment.sql` | ① `backup_tasks` 增列：`task_seq BIGSERIAL UNIQUE` `task_name` `task_result SMALLINT` `operator_code VARCHAR(8)` `create_user VARCHAR(64)`；② `restore_tasks` 同样补列；③ 新建 `backup_restore_file` 表（SN/file_name/md5/size/operator_code/update_time）；④ `backup_schedules` 增列 `is_enable SMALLINT GENERATED ALWAYS AS (CASE WHEN enabled THEN 1 ELSE 0 END) STORED`（兼容查询） |
| 状态码映射层 | 新增 `internal/backup/status_codec.go`：`statusString ↔ statusCode(0-6)` 双向映射；`taskResult` 派生 |
| `file_path_recorder.go` 改造 | 上传完成后计算 MD5 + size，写入 `backup_restore_file`（按 SN upsert） |
| 单元测试 | 状态映射全覆盖、文件记录 upsert 幂等 |

#### 里程碑 M2 — RPC/CommandKey/TransferComplete 闭环

| 任务 | 产物 |
|------|------|
| `executor.go:171` 改 CommandKey 为 `{cellCode}_BACKUP`（`cellCode` 取自 `device.cell_code` 或 SN 兜底） | [executor.go](../../omcgo/internal/backup/executor.go) |
| `restore_service.go` 增 Download 任务 CommandKey=`{cellCode}_RESTORE` 与 FileSize/TargetFileName/DelaySeconds | [restore_service.go](../../omcgo/internal/backup/restore_service.go) |
| 新增 `internal/backup/transfer_complete_router.go`：订阅 `SubjectDeviceTransferComplete`，按 CommandKey 后缀路由到备份/恢复回写逻辑，更新 `task_status=4` `task_result=1\|2` `end_time` | 新文件 + 在 `cmd/worker/main.go` 注册订阅 |
| FileType 字符串可选增强：保持值 `"3"`，在日志/SOAP 描述位置补 `"3 Vendor Configuration File"` | 仅描述层修改 |
| E2E 用例补充：模拟 CPE 上报 TransferComplete 验证回写 | `omcgo/scripts/e2e_verify.sh` + cpe_simulator 用例 |

#### 里程碑 M3 — 周期备份调度器（P0）

| 任务 | 产物 |
|------|------|
| 新建 `internal/backup/period_scheduler.go`：基于 `robfig/cron/v3`，启动时加载启用的 schedule，热更新订阅 schedule CRUD 事件 | 新文件 |
| 周期触发 → 调用 `BackupService.Create` 生成 backup_task（携带 `source="schedule:<id>"`） | 同上 |
| 新建 `internal/backup/task_reaper.go`：每 30s 扫 `status=0` 且 `created_at < now-2min` 的任务，重新发布 `SubjectBackupTaskCreated`（兜底事件丢失） | 对应规范 `BackupRestoreTaskJob` 的可靠性意图 |
| Worker 启动注册：`cmd/worker/main.go` 中初始化两个后台任务 | [cmd/worker/main.go](../../omcgo/cmd/worker/main.go) |
| 监控指标 | `omc_backup_schedule_fired_total` `omc_backup_task_reaped_total` |
| 单元测试 | cron 解析、热更新、reaper 幂等 |

#### 里程碑 M4 — REST API 别名与缺失端点

| 任务 | 产物 |
|------|------|
| 在 `handler.go` / `restore_handler.go` 增加路由别名 `/task/enb/config/backupRestore/*`（保留 `/api/v1/backup/*` 不动） | 兼容前端切换 |
| 新增 `queryCellInfos`：复用 `device.DeviceRepository.List`，按运营商/产品类型过滤 | handler 新增方法 |
| 新增 `queryTaskDeviceList`：join `device_tasks` 返回每设备执行细节 | 新增 repository 方法 |
| 新增 `getProductType`：枚举 `device.product_type` distinct 列表 | handler 新增方法 |
| 新增 `single/exportFile`：从 `backup_restore_file` 取文件，返回 MinIO presigned GET URL（或 FTP URL） | handler 新增方法 |
| 重命名 `terminateTask`：保留 `DELETE /api/v1/backup/tasks/:id`，新增 `POST /task/enb/config/backupRestore/terminateTask` 别名 | handler 新增 |

### 4.3 工作量预估（粗略）

| 里程碑 | 后端 | 测试 | 前端联调 | 备注 |
|--------|------|------|---------|------|
| M1 | 中 | 中 | 无 | 迁移 + 元数据落库 |
| M2 | 中 | 中 | 无 | 闭环修复，影响 ACS handler |
| M3 | 中 | 中 | 小 | 周期备份首次端到端 |
| M4 | 小 | 小 | 中 | API alias 与缺失端点 |

> 不给绝对工时，建议按 Sprint 节奏拆分到 2 个 Sprint 内完成（M1+M2 一个 Sprint，M3+M4 一个 Sprint）。

---

## 五、风险与缓解

| 风险 | 影响 | 缓解 |
|------|------|------|
| 任务表合并/拆分迁移 | 现有数据 + 外部引用受影响 | 采用增列 + view，不删旧字段，灰度切换 |
| CommandKey 改名 | 在飞任务 CPE 端 CommandKey 不匹配 | 部署前清空 in-flight 任务；TransferComplete router 同时识别新旧两种前缀，过渡 1 个 sprint |
| MinIO → FTP 切换 | 客户 FTP 不稳定会破坏现有备份 | 默认仍 MinIO；FTP 通过 `FTPConfig` 显式启用，按基站维度灰度 |
| Cron 调度器引入 | 集群多副本 worker 时可能重复触发 | 使用 PG advisory lock 或 Redis SETNX 互斥；worker 单实例时直接启用 |
| API 别名共存 | 重复 handler 易漂移 | alias 路由共享同一 handler 函数，不复制实现 |

---

## 六、验收清单（DoD）

- [ ] `backup_restore_task` 字段全部对齐（task_id 双轨 / task_status 七态映射 / task_result / operator_code / create_user 可查）
- [ ] `backup_restore_file` 表存在并随每次成功备份 upsert，含 MD5 与 size
- [ ] `backup_period_task` 字段对齐；启用的 schedule 按 cron 自动生成任务（验证 1 分钟级触发）
- [ ] Upload CommandKey == `{cellCode}_BACKUP`，Download CommandKey == `{cellCode}_RESTORE`
- [ ] FileType 值 `"3"`（描述位置含 `Vendor Configuration File`）
- [ ] TransferComplete 按 CommandKey 回写 `task_status=4` 与 `task_result`，含失败 FaultCode
- [ ] `BackupRestoreTaskJob` 兜底 reaper 上线，事件丢失场景任务不僵死
- [ ] `/task/enb/config/backupRestore/*` 9 个端点全部可用（alias 或新实现）
- [ ] E2E 用例覆盖：手动备份成功 / 周期备份触发 / 恢复成功 / TransferComplete 失败回写
- [ ] 迁移版本号连续；`scripts/check-migrations.sh` 通过
- [ ] `go build ./...` `go test ./...` `golangci-lint run` 全绿

---

## 七、未在本计划范围

- FTP 实际作为主传输通道：保留为可选项，等运营商提出明确需求再做（M4 之后）
- 模板下发与备份/恢复联动：经核查 `config/template/` 与 `backup/` 无引用关系，不在本次范围
- 多副本 worker 的分布式调度（advisory lock）：单实例 worker 阶段先用进程内互斥，集群化时再升级

---

*文档版本：v1.0，2026-05-18*
