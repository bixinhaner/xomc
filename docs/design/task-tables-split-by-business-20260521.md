# 文件传输任务表按业务拆分设计 - 20260521

> 状态：**设计稿**（待 review 后分阶段落地）
> 估算：核心工作量 2 周（含数据迁移 + 回归），含 buffer 3 周
> 关联：本设计独立于 dev-pipeline，跨多模块大改造

---

## 1. 现状

`upgrade_tasks` + `upgrade_sub_tasks` 两张共用表承载 **6 类业务**：

| TaskType 数字 | 业务 | UFTE TypeCode | 协议 RPC |
|---------------|------|---------------|----------|
| 1 | 固件升级 | `FIRMWARE_UPGRADE` 等 | Download |
| ? | 版本回退 | `VERSION_ROLLBACK` | Download |
| 10 (LogCollect) | 配置文件备份 NV | `CONFIG_BACKUP_NV` | Upload |
| 10 | 配置文件备份 XML | `CONFIG_BACKUP_XML` | Upload |
| 10 | 配置文件下发 | `CONFIG_RESTORE` | Download |
| 10 | 运行日志收集 | `RUNTIME_LOG_COLLECT` | Upload |
| 10 | 异常日志收集 | `FAULT_LOG_COLLECT` | Upload |

**痛点**：
- 6 类业务挤一张表，task_type 区分；查询都得过滤
- 业务专属字段（如固件 firmware_id、备份压缩选项、日志包大小阈值）混在一张宽表
- 状态机一刀切：升级走 downloading→rebooting→verifying→completed；备份只走 uploading→completed；却共用同一组 status 字段
- 跨业务通用功能（reaper / fileLandedLookup / completeSubTask）写死在 software 模块，跟业务边界不一致

---

## 2. 目标表结构（12 张物理表）

按业务命名：

```
firmware_upgrade_tasks     + firmware_upgrade_sub_tasks
device_rollback_tasks      + device_rollback_sub_tasks
config_backup_tasks        + config_backup_sub_tasks       (含 NV / XML，子类型放字段)
config_restore_tasks       + config_restore_sub_tasks
runtime_log_collect_tasks  + runtime_log_collect_sub_tasks
fault_log_collect_tasks    + fault_log_collect_sub_tasks
```

**字段策略**：
- **共有字段下沉到共有结构**（Go 端）：`TaskBase { ID, TaskName, Status, CreateUser, TotalCount, SuccessCount, FailCount, CreatedAt, UpdatedAt }`、`SubTaskBase { ID, TaskID, DeviceID, DeviceSN, Status, CommandKey, FailureReason, ErrorMessage, StartedAt, CompletedAt, CreatedAt, UpdatedAt }`
- **业务专属字段独立**：
  - firmware_upgrade: `firmware_id`, `is_keep_config`, `ori_version`, `dest_version`, `download_file_type`, `max_retries`, `max_concurrent`
  - device_rollback: `target_firmware_id`, `rollback_target_firmware_id`, `source` (manual/canary/auto)
  - config_backup: `backup_format` (NV|XML), `target_file_name`, `transport_path`, `enable_compression`
  - config_restore: `source_file_id`, `pre_validate`, `force` 等
  - runtime_log_collect / fault_log_collect: `target_file_name`, `transport_path`, `time_range_start`, `time_range_end`

---

## 3. 关键决策点（请逐项 review）

### 决策 1：状态机要不要统一？

**选 A — 6 套独立状态机**：每业务自己定义合法转换。优点：贴合业务；缺点：6 处状态机代码 + 6 套 reaper 超时。

**选 B — 抽象到 enum + 表内 check 约束**：保留通用枚举 (pending/in_progress/suspended/completed/failed/terminated)，子任务专属状态（downloading / uploading / rebooting）作为子表 phase 字段。优点：通用逻辑可复用；缺点：抽象成本。

**建议**：B（半统一）—— main task 状态共用，sub_task 状态按业务保留 phase 字段。

### 决策 2：跨业务通用功能（reaper / fileLandedLookup）怎么拆？

- **reaper**：每业务一个 reaper goroutine + 各自的 StaleTimeouts 配置 ✓
- **fileLandedLookup**：UFTE 反查 backup_restore_file 表的链路是文件落地的共性。建议保留单点查询，但 backup_restore_file 表加 `business_type` 字段区分（升级/备份/日志各占一类）
- **completeSubTask / failSubTask / IncrementCounts**：从 SoftwareExecutor 抽象到 BaseExecutor，6 个业务 Executor 继承
- **CommandKey 解析 / TC matched 反查**：保留 software 模块作为协议层兜底（ACS handler 收 TC 后用 CommandKey 反查到具体哪个业务的 sub_task）—— 需要在 sub_task.command_key 上加 business_type 标识，或者 6 张子表各建一个 by-command-key 索引，TC 处理时按业务类型分发

### 决策 3：UFTE 路由层怎么调整？

当前 `ufte.Service.CreateTask` 按 typeCode 分发到 `BatchUpgrade / BatchCollect / RollbackDevices` 三个底层方法。拆分后：

```
ufte.Service.CreateTask
  ├─ typeCode = FIRMWARE_UPGRADE / PATCH_UPGRADE / ...  → firmwareUpgradeSvc.Create
  ├─ typeCode = VERSION_ROLLBACK                        → rollbackSvc.Create
  ├─ typeCode = CONFIG_BACKUP_NV / CONFIG_BACKUP_XML    → configBackupSvc.Create
  ├─ typeCode = CONFIG_RESTORE                          → configRestoreSvc.Create
  ├─ typeCode = RUNTIME_LOG_COLLECT                     → runtimeLogSvc.Create
  └─ typeCode = FAULT_LOG_COLLECT                       → faultLogSvc.Create
```

每个业务 Service 自己持有 main + sub repository，状态机，executor 钩子。

### 决策 4：UFTE 列表跨业务查询怎么实现？

前端 UFTE 设备列表（`/api/v1/ufte/devices`）跨所有任务类型展示。拆表后：

**选 A — 应用层 6 次查询合并**：UFTE service 各调 6 个 Service.List 拼接。简单但 N+1。

**选 B — 数据库 VIEW**：建 `v_all_sub_tasks` UNION ALL 6 张子表，UFTE 列表查 view。性能跟拆表之前接近。

**建议**：B —— 物理表拆开支持业务独立演进，view 提供跨业务统一视图。

### 决策 5：数据迁移策略？

现有 `upgrade_tasks` 表 16 行（全是 task_type=10/LogCollect）+ `upgrade_sub_tasks` 若干行。

**选 A — 一次性数据迁移**：迁移脚本按 task_type 分发到 6 张新表；删旧表。简单但要求停机/停接口。

**选 B — 双写 + 灰度切换**：保留旧表，新业务写新表 + 旧表镜像；逐个业务切换读路径到新表；最后下线旧表。中等改动周期长。

**选 C — 物理表用 PG inheritance / partitioning**：旧表作为父表，6 张新表作为子表分区，按 task_type 路由。PG 原生支持，但分区表跨表查询性能不一定好。

**建议**：A（一次性） — 当前生产数据少（< 100 行），切换窗口可控；提前备份 upgrade_tasks/upgrade_sub_tasks，迁移失败可回滚。

### 决策 6：旧表保留还是 DROP？

**选 A — 迁移完保留旧表只读 30 天**：作为兜底回滚
**选 B — 迁移完立即 DROP**：彻底

**建议**：A —— 旧表 RENAME 加 _legacy 后缀保留 30 天，30 天后 DROP（写新迁移）。

---

## 4. 阶段计划（如批准 C 方案 + 上述建议）

| 阶段 | 工作 | 工期 | 风险 |
|------|------|------|------|
| **S1** 建表 + view | 12 张物理表 + 1 个 v_all_sub_tasks view，先空表 | 1 天 | 低 |
| **S2** Repository 层 | 6 套 main repo + 6 套 sub repo + 共有 BaseRepo（Squirrel 构建） | 3 天 | 中 |
| **S3** Service 层 | 6 个业务 Service（继承 BaseService 提供通用 CRUD）；reaper / executor 钩子 | 4 天 | 高（跨业务通用功能拆分） |
| **S4** UFTE 路由改造 | typeCode → 6 个 Service 的分发；UFTE List 查 view | 2 天 | 中 |
| **S5** 跨域适配 | FilePathRecorder / fileLandedLookup / completeSubTask / failSubTask / LogCollectResumer 全部按业务分发 | 3 天 | 高 |
| **S6** 数据迁移 | 一次性脚本：upgrade_tasks → 6 表；upgrade_sub_tasks → 6 表；旧表 RENAME _legacy | 1 天 | 高（数据一致性） |
| **S7** 回归测试 | 6 业务逐个端到端跑（设备真实触发） | 2-3 天 | 高 |
| **S8** 旧表清理 | 30 天后 DROP _legacy 表 | 0.5 天 | 低 |

**总工期**：保守 **3 周**（含 1 周 buffer 和回归）

---

## 5. 风险与回滚

### 主要风险
- **回归风险**：升级/备份/日志收集刚跑通，全部底层重写有较高概率引入新 bug
- **跨业务通用功能拆分难度**：reaper / fileLandedLookup / completeSubTask 等过去靠 software 单点维护，拆分后边界不清楚
- **CommandKey 反查**：ACS handler 收 TC 后按 CommandKey 反查 sub_task，6 张子表的 by-command-key 索引必须保证全局唯一（不同业务的 CommandKey 不能冲突）

### 回滚预案
- 每阶段独立 commit；可按阶段回滚
- 数据迁移用事务，失败立即回滚
- 旧表保留 30 天，最坏情况一条 ALTER VIEW 切回旧表

---

## 6. 待 review 的关键决策（开工前必须明确）

- [ ] **决策 1**：状态机统一性方案（A 独立 / B 半统一 / 我建议 B）
- [ ] **决策 2**：reaper、fileLandedLookup、completeSubTask 通用功能怎么抽象（建议 BaseExecutor）
- [ ] **决策 3**：UFTE 路由分发改造细节（6 个 Service 是否独立模块还是 software/{module}/ 子目录）
- [ ] **决策 4**：列表跨业务查询方案（建议 view）
- [ ] **决策 5**：数据迁移策略（建议一次性 + 旧表保留 30 天）
- [ ] **决策 6**：旧表保留多久（建议 30 天后 DROP）
- [ ] **决策 7**：是否同期重命名"软件升级模块" → "文件传输任务模块"以反映广义化语义？

---

## 7. 不推荐的替代方案及理由

| 方案 | 不推荐理由 |
|------|----------|
| A. PG VIEW 拆分（0 数据迁移） | 物理上还是一张表，业务边界没真正分开；适合还没有大规模生产数据的场景 |
| B. 按 RPC 类型拆 2 张 | 业务边界跟 RPC 类型不完全对齐（备份 + 日志都是 Upload 但语义不同） |
| D. 加 module 列 + filter | 没解决业务字段冲突问题（宽表越来越宽） |

---

## 8. 决策完毕后我能立即开工的清单

一旦决策 1-7 都拍板：
1. 我先写 S1 迁移文件（12 张表 + view）—— 半天，提 commit 让你 review schema
2. 然后 S2 Repository 层 + 单元测试 —— 3 天
3. 依次 S3 → S8

**每完成一个阶段，独立 commit + 部署 + 验证**，不一口气写完。

---

请回复以下问题再开工：

1. 上述决策 1-7 你的选择？
2. 是否同意按 S1-S8 分阶段，每阶段独立 commit + 验证？
3. 当前线上有没有更紧迫的功能不能等 3 周（这个改造期不能并行做其他大特性）？
