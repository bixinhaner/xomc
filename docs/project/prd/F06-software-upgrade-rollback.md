# 设备升级与回退功能开发设计方案

> **文档版本**：v3.1（统一表命名：upgrade_tasks 主任务 + upgrade_sub_tasks 子任务）
> **日期**：2026-04-23
> **范围**：对标《设备升级与回退流程设计文档》的功能需求和设计思想，基于当前 Go 项目实际架构给出生产级开发方案
> **审查基线**：对齐 `provision/Engine`、`task/TaskService`、`acs/handler.go` 等已验证的可靠性模式

---

## 一、当前项目架构基线

### 1.1 已完成的工作

| 层次 | 组件 | 状态 | 说明 |
|------|------|------|------|
| 数据库 | `firmware_versions` 表 | ✅ | UUID 主键、carrier/product_class/version 唯一约束、compatible_oui JSONB |
| 数据库 | `upgrade_tasks` 表 | ✅ | 已有 batch_id 分组、firmware_id 外键、status 状态、部分索引（后续迁移为 upgrade_sub_tasks 子任务表） |
| 后端 | model/repository/handler/service | ✅ | 完整四层：model → repository(接口+pg实现) → service → handler |
| 后端 | state_machine.go | ✅ | pending→downloading→rebooting→verifying→completed，含挂起/终止分支 |
| 后端 | 路由注册 | ✅ | `/firmware` 4 端点 + `/upgrade-tasks` 8 端点，JWT + 权限中间件 |
| 后端 | MinIO 集成 | ✅ | FirmwarePath 路径生成、PutObject 上传 |
| 后端 | EventBus 集成 | ✅ | 订阅 `device.inform.transfer_complete`，HandleTransferComplete 推进状态 |
| 后端 | cmdqueue 集成 | ✅ | Push Download Command + Connection Request 唤醒设备 |
| 前端 | 5 个页面 UI | ✅ | 升级计划/升级文件/版本回退/版本查询/激活计划 |
| 前端 | frontend-core API 层 | 🔶 | softwareApi.ts + useSoftware.ts 已实现，部分函数仍用 mock |
| 前端 | 页面 ↔ API 对接 | ❌ | 3 个核心页面用本地 Mock 数据，未调用真实 API |

### 1.2 项目中已建立的架构模式（本方案必须遵循）

| 模式 | 已有实例 | 说明 |
|------|----------|------|
| **task_id 分组** | `upgrade_tasks`（主任务） + `upgrade_sub_tasks.task_id`（子任务） | 前端"任务列表"直接查主任务表；"设备列表"查子任务表 `WHERE task_id=?`。主任务表存实时统计（success_count/fail_count） |
| **两表分离 + 聚合统计** | `ops_tasks`(SuccessCount/FailCount/TotalCount/Progress) | 主任务表存元数据和实时统计，子任务表存设备级执行状态。统计由子任务状态变更时 SQL 原子递增 |
| **事件驱动步骤编排** | `provision/Engine` | 订阅 `device.inform.*` + `command.*.response` → cmdqueue.Push → 等 Event 回调。**不是同步步骤循环** |
| **Carrier 适配器** | `carrier.Carrier` 接口 | 运营商差异通过 `CarrierRegistry.Get(code)` 获取适配器。4G/5G 差异同理 |
| **Redis 短生命周期状态** | `acs:session:*`(5min), `alarm:active:*`(24h), `acs:heartbeat:*`(2×interval) | 会话/缓存/去重用 Redis TTL 自动过期 |
| **配置注入** | `AppConfig.Provision` | Viper YAML + 环境变量，appconfig 结构体 |
| **Squirrel SQL** | 所有 pg_repository | `storage.Psql.Select/Insert/Update` + `pgxpool` |
| **前端 API 分层查询** | `softwareApi`（任务列表/设备列表分开查询） | 后端任务列表直接查 `upgrade_tasks`，设备列表查 `upgrade_sub_tasks WHERE task_id=?`，无需客户端聚合 |

---

## 二、差距分析（精简版）

### 2.1 需要补的功能（对标设计文档的核心能力）

| # | 功能需求（设计文档思想） | 当前状态 | 方案思路 |
|---|------------------------|----------|----------|
| F1 | 固件文件支持多种类型（IMG/PATCH/FPGA） | ❌ 仅 IMG | firmware_versions 加 file_type 字段，MinIO 按类型分目录 |
| F2 | 上传时计算 MD5、校验唯一性 | ❌ 未实现 | 上传流 tee 一份给 md5 哈希，唯一约束改为 (file_name, file_type) |
| F3 | 任务携带名称/类型/保留配置等元数据 | ❌ 字段缺失 | upgrade_tasks 加 task_name, task_type, is_keep_config 等字段 |
| F4 | 升级流程：冲突检查 → 在线检查 → Download → 等下载 → 等 TC → [5G:等升级完成] | 🔶 只有推 Download 和收 TC 两个点 | 参照 provision 模式，补充完整事件驱动流程 |
| F5 | 回退流程：SetParameterValues → 等 RebootComplete | ❌ 桩代码 | 新建回退逻辑，走 SetValues + 订阅 reboot_complete |
| F6 | 4G/5G 差异处理 | ❌ 无区分 | 扩展 Carrier 接口或新建 UpgradeAdapter |
| F7 | 超时 + 断线恢复 | ❌ 未实现 | Redis Key + TTL + 定时扫描 |
| F8 | 前端页面对接真实 API | ❌ Mock 数据 | 替换 Mock，对齐字段 |

### 2.2 不需要照搬的部分

| 设计文档描述 | 为什么不需要照搬 | 替代方案 |
|-------------|-----------------|---------|
| `upgrade_main_tasks` 主任务表 | 本方案已新增 `upgrade_tasks` 主任务表（3.2 节），前端直接查主任务表，无需客户端聚合 | 任务列表查 `upgrade_tasks`，设备列表查 `upgrade_sub_tasks WHERE task_id=?` |
| `upgrade_sub_tasks` 子任务表 | 本方案已新建 `upgrade_sub_tasks` 子任务表（3.3 节），通过 `task_id` 关联主任务 | 子任务只存设备级数据，主任务存元数据和统计 |
| `upgrading_info` 表 | Redis 更适合短生命周期锁（参考 `alarm:active:*`） | `software:upgrade:active:{deviceSN}` Redis Hash + TTL |
| `rela_upgrade_file_product` 关联表 | 项目已用 JSONB 数组模式（compatible_oui） | firmware_versions.compatible_products JSONB 数组 |
| 同步步骤调度循环（Step 0→5 循环） | 项目用事件驱动（provision 模式） | cmdqueue.Push → 等 EventBus 回调推进状态 |
| `upgradeWaitDeviceMap` Redis Hash | 过于复杂 | `software:upgrade:wait:{deviceSN}` Redis Key + TTL |

---

## 三、数据库 Schema 变更

### 3.1 firmware_versions 表扩展字段

```sql
-- +goose Up

-- 固件版本扩展字段
ALTER TABLE firmware_versions ADD COLUMN IF NOT EXISTS file_type SMALLINT NOT NULL DEFAULT 0;
  -- 0=IMG, 1=PATCH, 6=FPGA
ALTER TABLE firmware_versions ADD COLUMN IF NOT EXISTS md5_val VARCHAR(64);
ALTER TABLE firmware_versions ADD COLUMN IF NOT EXISTS recommend BOOLEAN DEFAULT false;
ALTER TABLE firmware_versions ADD COLUMN IF NOT EXISTS uploader VARCHAR(64);
ALTER TABLE firmware_versions ADD COLUMN IF NOT EXISTS manufacturer VARCHAR(128);
ALTER TABLE firmware_versions ADD COLUMN IF NOT EXISTS description TEXT;

-- ★ 唯一约束保留语义粒度：(carrier, COALESCE(product_class,''), version, file_type)
-- 保留 carrier 维度（不同运营商可上传同名文件），
-- 加 file_type（同一产品可有 IMG + PATCH 同版本号）
-- 不用 (file_name, file_type)，因为不同运营商可能上传同名文件导致冲突
-- ★ 使用 COALESCE(product_class, '') 处理 NULL：SQL NULL 不参与唯一约束判定，
--   不用 COALESCE 会导致多个 product_class=NULL 的记录被允许插入（D8 缺陷修复）
DROP INDEX IF EXISTS idx_firmware_unique_version;
CREATE UNIQUE INDEX IF NOT EXISTS idx_firmware_unique_version
  ON firmware_versions (carrier, COALESCE(product_class, ''), version, file_type);

-- 文件类型 + 状态 复合索引（支持按类型列表查询）
CREATE INDEX IF NOT EXISTS idx_firmware_file_type_status
  ON firmware_versions (file_type, status);

-- +goose Down
DROP INDEX IF EXISTS idx_firmware_file_type_status;
DROP INDEX IF EXISTS idx_firmware_unique_version;
CREATE UNIQUE INDEX IF NOT EXISTS idx_firmware_unique_version
  ON firmware_versions (carrier, product_class, version);
ALTER TABLE firmware_versions DROP COLUMN IF EXISTS description;
ALTER TABLE firmware_versions DROP COLUMN IF EXISTS manufacturer;
ALTER TABLE firmware_versions DROP COLUMN IF EXISTS uploader;
ALTER TABLE firmware_versions DROP COLUMN IF EXISTS recommend;
ALTER TABLE firmware_versions DROP COLUMN IF EXISTS md5_val;
ALTER TABLE firmware_versions DROP COLUMN IF EXISTS file_type;
```

### 3.2 upgrade_tasks 表扩展（ALTER 已有表）

前端页面的"任务列表"Tab 需要主任务级元数据和实时统计。已有 `upgrade_tasks` 表（migration 000006 创建）通过 ALTER 添加新列，无需重建。

> **迁移策略**：现有 `upgrade_tasks` 是设备级表（有 device_id、batch_id），本方案将其**扩展**为主任务表。
> 历史设备级数据（device_id/batch_id 行）迁移到 `upgrade_sub_tasks`（3.3 节），迁移后清空旧行。
> 新增的列均有默认值，ALTER 不影响已有数据。

```sql
-- +goose Up

-- ★ Step 1: 迁移已有设备级数据到 upgrade_sub_tasks（3.3 节 CREATE 后执行）
-- 已有行的 device_id/batch_id 是设备级数据，迁移到子任务表
-- batch_id 作为 task_id 关联（batch_id 非空的行才迁移）
INSERT INTO upgrade_sub_tasks (id, task_id, device_id, firmware_id, status, error_message,
    retry_count, max_retries, started_at, completed_at, created_at, updated_at)
SELECT id,
    batch_id,     -- batch_id → task_id（已有批次分组复用）
    device_id, firmware_id, status, error_message,
    retry_count, max_retries, started_at, completed_at, created_at, updated_at
FROM upgrade_tasks
WHERE device_id IS NOT NULL
ON CONFLICT DO NOTHING;

-- ★ Step 2: 删除已迁移的设备级行（保留 batch_id 汇总行，或清空后只留新增行）
DELETE FROM upgrade_tasks WHERE device_id IS NOT NULL;

-- ★ Step 3: ALTER 添加主任务级新列
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS task_name       VARCHAR(256);
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS task_type       SMALLINT NOT NULL DEFAULT 1;
  -- 1=IMG升级, 2=回退, 4=PATCH, 6=FPGA, 8=预留
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS file_name       VARCHAR(256);
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS file_md5        VARCHAR(64);
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS result          VARCHAR(20);
  -- success/partial/failed/terminated（ended 时才有值）
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS operator_code   VARCHAR(8) NOT NULL DEFAULT 'cmcc';
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS product_class   VARCHAR(64);
  -- 产品类型（如 SmallCell-LTE、SmallCell-NR），替代 is_gnb，从固件版本继承
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS is_keep_config  BOOLEAN DEFAULT true;
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS create_status   VARCHAR(16) NOT NULL DEFAULT 'active';
  -- active=立即执行, suspend=挂起, timing=定时（timing 暂未实现，预留）
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS create_user     VARCHAR(64) NOT NULL DEFAULT 'system';
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS total_count     INTEGER NOT NULL DEFAULT 0;
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS success_count   INTEGER NOT NULL DEFAULT 0;
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS fail_count      INTEGER NOT NULL DEFAULT 0;
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS max_concurrent  INTEGER DEFAULT 5;
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS ended_at        TIMESTAMPTZ;

-- ★ Step 4: 清理不再需要的旧列（设备级数据已迁移到 upgrade_sub_tasks）
ALTER TABLE upgrade_tasks DROP COLUMN IF EXISTS device_id;
ALTER TABLE upgrade_tasks DROP COLUMN IF EXISTS batch_id;
-- firmware_id 保留：主任务仍需记录目标固件
-- status/error_message/retry_count/max_retries/started_at/completed_at 保留：主任务状态管理

-- ★ Step 5: 新增索引
CREATE INDEX IF NOT EXISTS idx_upgrade_tasks_task_status ON upgrade_tasks (status);
CREATE INDEX IF NOT EXISTS idx_upgrade_tasks_task_operator ON upgrade_tasks (operator_code);
CREATE INDEX IF NOT EXISTS idx_upgrade_tasks_task_product_class ON upgrade_tasks (product_class);
CREATE INDEX IF NOT EXISTS idx_upgrade_tasks_task_create_user ON upgrade_tasks (create_user);
CREATE INDEX IF NOT EXISTS idx_upgrade_tasks_task_created_at ON upgrade_tasks (created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_upgrade_tasks_task_created_at;
DROP INDEX IF EXISTS idx_upgrade_tasks_task_create_user;
DROP INDEX IF EXISTS idx_upgrade_tasks_task_product_class;
DROP INDEX IF EXISTS idx_upgrade_tasks_task_operator;
DROP INDEX IF EXISTS idx_upgrade_tasks_task_status;
ALTER TABLE upgrade_tasks DROP COLUMN IF EXISTS ended_at;
ALTER TABLE upgrade_tasks DROP COLUMN IF EXISTS max_concurrent;
ALTER TABLE upgrade_tasks DROP COLUMN IF EXISTS fail_count;
ALTER TABLE upgrade_tasks DROP COLUMN IF EXISTS success_count;
ALTER TABLE upgrade_tasks DROP COLUMN IF EXISTS total_count;
ALTER TABLE upgrade_tasks DROP COLUMN IF EXISTS create_user;
ALTER TABLE upgrade_tasks DROP COLUMN IF EXISTS create_status;
ALTER TABLE upgrade_tasks DROP COLUMN IF EXISTS is_keep_config;
ALTER TABLE upgrade_tasks DROP COLUMN IF EXISTS product_class;
ALTER TABLE upgrade_tasks DROP COLUMN IF EXISTS operator_code;
ALTER TABLE upgrade_tasks DROP COLUMN IF EXISTS result;
ALTER TABLE upgrade_tasks DROP COLUMN IF EXISTS file_md5;
ALTER TABLE upgrade_tasks DROP COLUMN IF EXISTS file_name;
ALTER TABLE upgrade_tasks DROP COLUMN IF EXISTS task_type;
ALTER TABLE upgrade_tasks DROP COLUMN IF EXISTS task_name;
-- 恢复旧列（此处不恢复数据，仅恢复结构）
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS device_id UUID NOT NULL DEFAULT gen_random_uuid();
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS batch_id UUID;
```

### 3.3 新建 upgrade_sub_tasks 子任务表

`upgrade_tasks` 已在 3.2 节通过 ALTER 扩展为主任务表。本节新建 `upgrade_sub_tasks` 子任务表，通过 `task_id` 关联主任务，存储设备级执行数据。

```sql
-- +goose Up

-- ★ 子任务表：每个设备一行，通过 task_id 关联 upgrade_tasks 主任务
CREATE TABLE IF NOT EXISTS upgrade_sub_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id         UUID NOT NULL REFERENCES upgrade_tasks(id) ON DELETE CASCADE,
    device_id       UUID NOT NULL,
    firmware_id     UUID REFERENCES firmware_versions(id) ON DELETE SET NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    error_message   TEXT,
    retry_count     INTEGER NOT NULL DEFAULT 0,
    max_retries     INTEGER NOT NULL DEFAULT 3,
    device_sn       VARCHAR(64),                               -- 冗余设备 SN
    ori_version     VARCHAR(64),                               -- 升级前版本
    dest_version    VARCHAR(64),                               -- 目标版本
    command_key     VARCHAR(256),                              -- Download 命令 key
    failure_reason  TEXT,
    pre_suspend_status VARCHAR(20),                            -- 挂起前状态
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ★ 关键索引
CREATE INDEX idx_upgrade_sub_tasks_task_id_status
    ON upgrade_sub_tasks (task_id, status);                    -- 按主任务聚合子任务状态
CREATE INDEX idx_upgrade_sub_tasks_device_sn
    ON upgrade_sub_tasks (device_sn);                          -- 断线恢复按 SN 查找
CREATE INDEX idx_upgrade_sub_tasks_device_active
    ON upgrade_sub_tasks (device_id, status);                  -- GetActiveByDeviceID
CREATE INDEX idx_upgrade_sub_tasks_created_at
    ON upgrade_sub_tasks (created_at DESC);                    -- 排序
CREATE INDEX idx_upgrade_sub_tasks_command_key
    ON upgrade_sub_tasks (command_key) WHERE command_key IS NOT NULL;  -- commandKey 反查

-- ★ 同一设备最多一条活跃子任务（数据库级互斥）
CREATE UNIQUE INDEX idx_upgrade_sub_tasks_device_active_uniq
    ON upgrade_sub_tasks (device_id)
    WHERE status NOT IN ('completed', 'failed', 'terminated');

-- 子任务 status CHECK 约束
ALTER TABLE upgrade_sub_tasks ADD CONSTRAINT chk_upgrade_sub_tasks_status
    CHECK (status IN ('pending', 'downloading', 'rebooting', 'verifying',
                       'completed', 'failed', 'suspended', 'terminated'));

-- updated_at 触发器
CREATE TRIGGER trigger_upgrade_sub_tasks_updated_at
    BEFORE UPDATE ON upgrade_sub_tasks FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- +goose Down
DROP TRIGGER IF EXISTS trigger_upgrade_sub_tasks_updated_at ON upgrade_sub_tasks;
ALTER TABLE upgrade_sub_tasks DROP CONSTRAINT IF EXISTS chk_upgrade_sub_tasks_status;
DROP INDEX IF EXISTS idx_upgrade_sub_tasks_device_active_uniq;
DROP INDEX IF EXISTS idx_upgrade_sub_tasks_command_key;
DROP INDEX IF EXISTS idx_upgrade_sub_tasks_created_at;
DROP INDEX IF EXISTS idx_upgrade_sub_tasks_device_active;
DROP INDEX IF EXISTS idx_upgrade_sub_tasks_device_sn;
DROP INDEX IF EXISTS idx_upgrade_sub_tasks_task_id_status;
DROP TABLE IF EXISTS upgrade_sub_tasks;
```

**设计说明**：
- **两表分离**：`upgrade_tasks` 存任务级元数据（名称/类型/统计/操作人），`upgrade_sub_tasks` 存设备级执行状态。
- **ALTER 而非重建**：已有 `upgrade_tasks` 表通过 ALTER 添加新列扩展为主任务表，旧设备级数据迁移到 `upgrade_sub_tasks` 后清空。
- **task_id 即主任务 ID**：`upgrade_sub_tasks.task_id` 外键指向 `upgrade_tasks.id`，级联删除。前端"任务列表"查 `upgrade_tasks`，"设备列表"查 `upgrade_sub_tasks WHERE task_id=?`。
- **子任务只存设备级数据**：device_sn、ori_version、dest_version、command_key、failure_reason。
- **实时统计**：主任务表的 success_count/fail_count 由子任务状态变更时 SQL 原子递增更新。
- **用 product_class 区分制式**：不再使用 `is_gnb` 布尔字段，通过 `product_class`（如 `SmallCell-LTE`/`SmallCell-NR`）区分 4G/5G，与设备表和固件表保持一致。

### 3.4 现有表设计缺陷修复

> **注意**：以下修复已合并到 3.1-3.3 节的迁移文件中（`000029_software_upgrade_enhance.sql`）。
> 如果 3.2/3.3 节是新建表，CHECK 约束、唯一索引、字段已在 CREATE TABLE 中包含，**无需重复 ALTER**。
> 以下 ALTER 语句仅作为"补丁修复已有环境"的备选方案，不与 3.2/3.3 同时执行。
>
> 基于对 `firmware_versions`、`upgrade_tasks`（主任务）、`upgrade_sub_tasks`（子任务）三张表的全面审查，修复以下设计问题。

#### 缺陷 1：firmware_id 外键阻塞固件删除

**问题**：`upgrade_tasks.firmware_id` 和 `upgrade_sub_tasks.firmware_id` 的 `REFERENCES firmware_versions(id)` 意味着只要有任务引用了该固件，就**无法删除**固件记录。实际业务中，管理员可能需要清理过期固件，但升级历史应保留。

**修复**：两表均改为 `ON DELETE SET NULL`，固件删除后 firmware_id 变为 NULL，但设备级冗余字段（upgrade_sub_tasks 中的 dest_version）和主任务表冗余字段（file_name/file_md5）保留历史信息。

```sql
-- +goose Up
-- 修复 upgrade_sub_tasks FK：固件删除时解除引用而非阻塞
-- （注：3.3 节 CREATE TABLE 已含 ON DELETE SET NULL，此为补丁修复已有表）
ALTER TABLE upgrade_sub_tasks DROP CONSTRAINT IF EXISTS upgrade_sub_tasks_firmware_id_fkey;
ALTER TABLE upgrade_sub_tasks ADD CONSTRAINT upgrade_sub_tasks_firmware_id_fkey
    FOREIGN KEY (firmware_id) REFERENCES firmware_versions(id) ON DELETE SET NULL;

-- 修复 upgrade_tasks 主任务表 FK
ALTER TABLE upgrade_tasks DROP CONSTRAINT IF EXISTS upgrade_tasks_firmware_id_fkey;
ALTER TABLE upgrade_tasks ADD CONSTRAINT upgrade_tasks_firmware_id_fkey
    FOREIGN KEY (firmware_id) REFERENCES firmware_versions(id) ON DELETE SET NULL;
-- +goose Down
ALTER TABLE upgrade_tasks DROP CONSTRAINT IF EXISTS upgrade_tasks_firmware_id_fkey;
ALTER TABLE upgrade_tasks ADD CONSTRAINT upgrade_tasks_firmware_id_fkey
    FOREIGN KEY (firmware_id) REFERENCES firmware_versions(id);
ALTER TABLE upgrade_sub_tasks DROP CONSTRAINT IF EXISTS upgrade_sub_tasks_firmware_id_fkey;
ALTER TABLE upgrade_sub_tasks ADD CONSTRAINT upgrade_sub_tasks_firmware_id_fkey
    FOREIGN KEY (firmware_id) REFERENCES firmware_versions(id);
```

#### 缺陷 2：状态/类型字段无 CHECK 约束

**问题**：`upgrade_tasks.status`、`upgrade_sub_tasks.status`、`firmware_versions.status` 都是 VARCHAR 但无 CHECK 约束，应用层可以写入任意值（如 `"sucess"` 拼写错误），数据库层面不做防护。

```sql
-- +goose Up
-- upgrade_sub_tasks.status CHECK 约束（子任务：设备级状态）
-- 注：3.3 节 CREATE TABLE 已含此约束，此为补丁修复已有表
ALTER TABLE upgrade_sub_tasks ADD CONSTRAINT chk_upgrade_sub_tasks_status
    CHECK (status IN ('pending', 'downloading', 'rebooting', 'verifying',
                       'completed', 'failed', 'suspended', 'terminated'));

-- upgrade_tasks.status CHECK 约束（主任务：任务级状态）
ALTER TABLE upgrade_tasks ADD CONSTRAINT chk_upgrade_tasks_status
    CHECK (status IN ('pending', 'in_progress', 'suspended', 'ended'));

-- upgrade_tasks.result CHECK 约束（NULL 或有效值）
ALTER TABLE upgrade_tasks ADD CONSTRAINT chk_upgrade_tasks_result
    CHECK (result IS NULL OR result IN ('success', 'partial', 'failed', 'terminated'));

-- upgrade_tasks.task_type CHECK 约束
ALTER TABLE upgrade_tasks ADD CONSTRAINT chk_upgrade_tasks_task_type
    CHECK (task_type IN (1, 2, 4, 6, 8));

-- upgrade_tasks.create_status CHECK 约束
ALTER TABLE upgrade_tasks ADD CONSTRAINT chk_upgrade_tasks_create_status
    CHECK (create_status IN ('active', 'suspend', 'timing'));

-- firmware_versions.file_type CHECK 约束
ALTER TABLE firmware_versions ADD CONSTRAINT chk_firmware_versions_file_type
    CHECK (file_type IN (0, 1, 6));

-- firmware_versions.status CHECK 约束
ALTER TABLE firmware_versions ADD CONSTRAINT chk_firmware_versions_status
    CHECK (status IN ('active', 'deprecated', 'archived'));

-- +goose Down
ALTER TABLE upgrade_tasks DROP CONSTRAINT IF EXISTS chk_upgrade_tasks_create_status;
ALTER TABLE firmware_versions DROP CONSTRAINT IF EXISTS chk_firmware_versions_status;
ALTER TABLE firmware_versions DROP CONSTRAINT IF EXISTS chk_firmware_versions_file_type;
ALTER TABLE upgrade_tasks DROP CONSTRAINT IF EXISTS chk_upgrade_tasks_task_type;
ALTER TABLE upgrade_tasks DROP CONSTRAINT IF EXISTS chk_upgrade_tasks_result;
ALTER TABLE upgrade_tasks DROP CONSTRAINT IF EXISTS chk_upgrade_tasks_status;
ALTER TABLE upgrade_sub_tasks DROP CONSTRAINT IF EXISTS chk_upgrade_sub_tasks_status;
```

#### 缺陷 3：同一设备可同时创建多条 active task（竞态条件）

**问题**：当前 `GetActiveByDeviceID` 先查再创建，两个并发请求可能同时通过检查，各自创建一条 active task。

**修复**：添加部分唯一索引，数据库层面强制一个设备最多一条非终态任务。

```sql
-- +goose Up
-- ★ 同一设备最多一条活跃子任务（数据库级互斥，防止并发竞态）
-- 注：3.3 节 CREATE TABLE 已定义 idx_upgrade_sub_tasks_device_active_uniq，此为补丁修复已有表
CREATE UNIQUE INDEX IF NOT EXISTS idx_upgrade_sub_tasks_device_active_uniq
    ON upgrade_sub_tasks (device_id)
    WHERE status NOT IN ('completed', 'failed', 'terminated');

-- +goose Down
DROP INDEX IF EXISTS idx_upgrade_sub_tasks_device_active_uniq;
```

**代码配合**：`upgradeRepo.Create()` 改为捕获 unique violation 返回友好错误：

```go
func (r *PgUpgradeTaskRepository) Create(ctx context.Context, task *UpgradeTask) error {
    // ... squirrel Insert ...
    if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
        return commonerrors.NewBusinessError(8001, "设备已有正在进行的升级任务", commonerrors.ErrAlreadyExists)
    }
}
```

#### 缺陷 4：SuspendUpgrade 滥用 error_message 存储恢复状态

**问题**：当前 `SuspendUpgrade` 将挂起前的状态（如 `"downloading"`）写入 `error_message` 字段，`ResumeUpgrade` 再从中读取。但 `error_message` 语义是"错误消息"，用存状态值会与实际错误混淆。

**修复**：新增 `pre_suspend_status` 字段，语义明确。

```sql
-- +goose Up
-- 注：3.3 节 CREATE TABLE 已含此列，此为补丁修复已有表
ALTER TABLE upgrade_sub_tasks ADD COLUMN IF NOT EXISTS pre_suspend_status VARCHAR(20);
COMMENT ON COLUMN upgrade_sub_tasks.pre_suspend_status IS '挂起前的状态，用于恢复时还原';
-- +goose Down
ALTER TABLE upgrade_sub_tasks DROP COLUMN IF EXISTS pre_suspend_status;
```

**代码配合**：
```go
// SuspendUpgrade — 保存挂起前状态到专用字段
func (s *SoftwareService) SuspendUpgrade(ctx context.Context, taskID uuid.UUID) error {
    // ...
    if err := s.upgradeRepo.Suspend(ctx, taskID, task.Status); err != nil { ... }
    // Suspend 方法: UPDATE SET status='suspended', pre_suspend_status=$previous_status, error_message=NULL
}

// ResumeUpgrade — 从专用字段恢复
func (s *SoftwareService) ResumeUpgrade(ctx context.Context, taskID uuid.UUID) error {
    // ...
    resumeState := task.PreSuspendStatus  // ★ 不再从 error_message 读
    if resumeState == "" { resumeState = UpgradeDownloading }
}
```

#### 缺陷 5：upgrade_tasks 主任务统计计数并发安全

**问题**：子任务完成时更新主任务的 `success_count++`，如果两个子任务同时完成，service 层的 `先读→计算→写回` 可能丢失更新。

**修复**：使用 SQL 原子递增，而非应用层计算。

```go
// TaskRepository 接口
IncrementCounts(ctx context.Context, taskID uuid.UUID, successDelta, failDelta int) error

// pg_task_repository.go 实现
func (r *PgTaskRepository) IncrementCounts(ctx context.Context, taskID uuid.UUID, successDelta, failDelta int) error {
    query := `UPDATE upgrade_tasks
              SET success_count = success_count + $1,
                  fail_count = fail_count + $2,
                  updated_at = NOW()
              WHERE id = $3`
    _, err := r.pool.Exec(ctx, query, successDelta, failDelta, taskID)
    return err
}
```

#### 缺陷 6：firmware_versions 缺少 updated_at 触发器兼容

**现状**：migration 000006 已有 `trigger_firmware_versions_updated_at`，无需修复。确认此触发器能覆盖新增字段的更新场景——可以，它对所有 UPDATE 生效。

#### 缺陷 7：upgrade_tasks 缺少触发器

**问题**：migration 000006 已定义了 `trigger_upgrade_tasks_updated_at`，新增字段后此触发器自动生效，无需额外操作。

---

## 四、后端改造方案

### 4.1 目录结构（全部在 `internal/software/` 下）

```
internal/software/
├── model.go                      # 扩展：新增 UpgradeTask 主任务 + UpgradeSubTask 子任务模型、FileType/TaskType 常量
├── repository.go                 # 扩展：新增 TaskRepository 接口
├── pg_task_repository.go        # 新增：upgrade_tasks 表 CRUD + 聚合更新
├── pg_firmware_repository.go     # 改造：支持新字段读写、file_type 过滤、recommend 切换
├── pg_upgrade_repository.go      # 改造：子任务支持新字段、按 task_id 查询
├── handler.go                    # 扩展：新增 batch 端点 + 子任务查询端点
├── service.go                    # 重构：核心升级/回退逻辑
├── state_machine.go              # 不变（当前状态机已覆盖升级全流程）
├── executor.go                   # 新增：升级执行器（事件驱动模式）
├── rollback.go                   # 新增：回退执行器
└── adapter.go                    # 新增：4G/5G 差异适配
```

### 4.2 model.go 扩展

```go
// FileType 升级文件类型
type FileType int

const (
    FileTypeIMG   FileType = 0 // 软件主镜像（默认）
    FileTypePATCH FileType = 1 // 补丁包
    FileTypeFPGA  FileType = 6 // FPGA
)

// TaskType 任务类型
type TaskType int

const (
    TaskTypeUpgrade  TaskType = 1 // IMG 升级
    TaskTypeRollback TaskType = 2 // 版本回退
    TaskTypePatch    TaskType = 4 // PATCH 升级
    TaskTypeFPGA     TaskType = 6 // FPGA 升级
    TaskTypeReserved TaskType = 8 // 预留（配置升级）
)

// TaskStatus 主任务状态
type TaskStatus string

const (
    TaskPending    TaskStatus = "pending"
    TaskInProgress TaskStatus = "in_progress"
    TaskSuspended  TaskStatus = "suspended"
    TaskEnded      TaskStatus = "ended"
)

// TaskResult 主任务结果（ended 时才有值）
type TaskResult string

const (
    TaskResultSuccess    TaskResult = "success"    // 全部成功
    TaskResultPartial    TaskResult = "partial"    // 部分成功
    TaskResultFailed     TaskResult = "failed"     // 全部失败
    TaskResultTerminated TaskResult = "terminated" // 已终止
)

// ★ UpgradeTask 主任务（对应前端"任务列表"Tab）
type UpgradeTask struct {
    ID            uuid.UUID       `json:"id"`
    TaskName      string          `json:"task_name" binding:"required"`
    TaskType      TaskType        `json:"task_type"`
    FirmwareID    *uuid.UUID      `json:"firmware_id,omitempty"`
    FileName      string          `json:"file_name,omitempty"`
    FileMD5       string          `json:"file_md5,omitempty"`
    Status        TaskStatus     `json:"status"`
    Result        TaskResult     `json:"result,omitempty"`
    OperatorCode  model.CarrierCode `json:"operator_code"`
    ProductClass  string          `json:"product_class"` // 产品类型（如 SmallCell-LTE/SmallCell-NR），替代 is_gnb
    IsKeepConfig  bool            `json:"is_keep_config"`
    CreateStatus  string          `json:"create_status"` // active/suspend/timing
    CreateUser    string          `json:"create_user"`
    TotalCount    int             `json:"total_count"`
    SuccessCount  int             `json:"success_count"`
    FailCount     int             `json:"fail_count"`
    MaxConcurrent int             `json:"max_concurrent"`
    StartedAt     *time.Time      `json:"started_at,omitempty"`
    EndedAt       *time.Time      `json:"ended_at,omitempty"`
    CreatedAt     time.Time       `json:"created_at"`
    UpdatedAt     time.Time       `json:"updated_at"`
}

// FirmwareVersion 扩展字段
type FirmwareVersion struct {
    ID            uuid.UUID         `json:"id"`
    Carrier       model.CarrierCode `json:"carrier"`
    ProductClass  string            `json:"product_class"`
    Version       string            `json:"version"`
    FileName      string            `json:"file_name"`
    FileSize      int64             `json:"file_size"`
    FileType      FileType          `json:"file_type"`       // 新增
    MinIOPath     string            `json:"minio_path"`
    CompatibleOUI []string          `json:"compatible_oui"`
    MD5Val        string            `json:"md5_val"`          // 新增
    Recommend     bool              `json:"recommend"`        // 新增
    Uploader      string            `json:"uploader"`         // 新增
    Manufacturer  string            `json:"manufacturer"`     // 新增
    ReleaseNotes  string            `json:"release_notes"`
    Description   string            `json:"description"`      // 新增
    Status        string            `json:"status"`
    CreatedAt     time.Time         `json:"created_at"`
    UpdatedAt     time.Time         `json:"updated_at"`
}

// UpgradeSubTask 子任务（对应前端"设备列表"Tab，映射 upgrade_sub_tasks 表）
type UpgradeSubTask struct {
    // 现有字段保持不变
    ID           uuid.UUID    `json:"id"`
    DeviceID     uuid.UUID    `json:"device_id"`
    FirmwareID   *uuid.UUID   `json:"firmware_id,omitempty"` // ★ 改为指针：ON DELETE SET NULL 后可为空
    TaskID      *uuid.UUID   `json:"task_id,omitempty"` // → upgrade_tasks.id
    Status       UpgradeState `json:"status"`
    ErrorMessage string       `json:"error_message,omitempty"`
    RetryCount   int          `json:"retry_count"`
    MaxRetries   int          `json:"max_retries"`
    StartedAt    *time.Time   `json:"started_at,omitempty"`
    CompletedAt  *time.Time   `json:"completed_at,omitempty"`
    CreatedAt    time.Time    `json:"created_at"`
    UpdatedAt    time.Time    `json:"updated_at"`

    // 新增：设备级信息（子任务独有）
    DeviceSN      string `json:"device_sn,omitempty"`
    OriVersion    string `json:"ori_version,omitempty"`
    DestVersion   string `json:"dest_version,omitempty"`
    CommandKey    string `json:"command_key,omitempty"`
    FailureReason string `json:"failure_reason,omitempty"`
    PreSuspendStatus string `json:"pre_suspend_status,omitempty"` // ★ 新增：挂起前状态，用于恢复时还原
}

// BatchUpgradeRequest 扩展
type BatchUpgradeRequest struct {
    DeviceIDs    []uuid.UUID `json:"device_ids" binding:"required,min=1"`
    FirmwareID   uuid.UUID   `json:"firmware_id" binding:"required"`
    Concurrency  int         `json:"concurrency"`
    TaskName     string      `json:"task_name" binding:"required"`
    TaskType     TaskType    `json:"task_type"`
    IsKeepConfig bool        `json:"is_keep_config"`
}

// BatchActionRequest 批量操作请求
type BatchActionRequest struct {
    TaskID uuid.UUID `json:"task_id" binding:"required"`
}

// RollbackRequest 回退请求
type RollbackRequest struct {
    DeviceIDs    []uuid.UUID     `json:"device_ids" binding:"required,min=1"`
    TaskName     string          `json:"task_name" binding:"required"`
    OperatorCode model.CarrierCode `json:"operator_code" binding:"required"`
    CreateUser   string          `json:"create_user" binding:"required"`
}
```

### 4.3 升级执行器（executor.go）— 事件驱动模式

**核心设计**：参照 `provision/Engine` 的事件驱动模式，不使用同步步骤循环。

```
BatchUpgrade()
  → 逐设备创建 UpgradeTask（状态 pending）
  → goroutine-per-device: executeOne(task)
      → 检查冲突（Redis SETNX software:upgrade:active:{deviceSN}）
      → 查设备在线 → 离线则写 Redis software:upgrade:wait:{deviceSN} + return
      → 记录 ori_version（从 device.FirmwareVersion 获取）
      → 构造 Download Command + commandKey
      → cmdqueue.Push → 状态推进到 downloading
      → 等待 command.download.response 事件（成功/失败）
      → 等待 device.inform.transfer_complete 事件
      → [5G] 等待 device.inform 102 事件
      → 状态推进到 completed / failed

HandleDownloadResponse(event)    ← 订阅 command.download.response
  → 找到对应 task（通过 commandKey 匹配）
  → FaultCode → failed
  → 成功 → 继续等待（不做状态推进，等 TransferComplete）

HandleTransferComplete(event)    ← 订阅 device.inform.transfer_complete（已有）
  → 找到对应 task（通过 deviceSN 匹配 active task）
  → FaultCode → failed
  → 成功 → 4G: completed; 5G: 继续等待 102 事件

HandleUpgradeFinish(event)       ← 订阅 device.inform（过滤 102 事件码）
  → [仅 5G] 找到对应 task
  → upgradeStatus 判定成功/失败
  → 推进到 completed / failed

HandleRebootComplete(event)      ← 订阅 device.inform.reboot_complete
  → [仅回退] 找到对应 task
  → 推进到 completed / failed
```

**Redis Key 设计**（遵循 `module:entity:id` 模式）：

| Key | 类型 | TTL | 用途 |
|-----|------|-----|------|
| `software:upgrade:active:{deviceSN}` | String (taskID) | 1h | 冲突检查：设备正在升级中 |
| `software:upgrade:wait:{deviceSN}` | String (JSON) | 1h | 断线等待恢复：`{"task_id":"...", "step":"download"}` |
| `software:upgrade:cmdkey:{commandKey}` | String (taskID) | 30min | commandKey → taskID 反向映射 |

**与 provision 模式的一致性**：

| 维度 | provision/Engine | software/Executor（本方案） |
|------|------------------|---------------------------|
| 创建任务 | HandleBootstrap → 创建 ProvisioningTask | BatchUpgrade → 创建 UpgradeSubTask(s) |
| 推送命令 | cmdqueue.Push(GetParameterValues) | cmdqueue.Push(Download) |
| 接收响应 | Subscribe command.get_parameters.response | Subscribe command.download.response |
| 状态推进 | 回调中 UpdateStatus | 回调中 UpdateStatus |
| 超时清理 | TaskReaper 定时扫描 | 类似 Reaper，扫描超时 task |
| 断线恢复 | Reset-on-BOOT 重试 | wait key + 设备上线事件恢复 |

### 4.4 回退执行器（rollback.go）

```go
// Rollback 回退流程（独立于升级，走 SetParameterValues 路径）
//
// RollbackDevices()
//   → 创建 UpgradeTask(task_type=2, status=pending)
//   → 查设备在线
//   → [4G] 先 GetParameterValues 查 ROLLBACK_ENABLE
//        enable=false → failed
//   → SetParameterValues 设置回退参数
//        参数路径由 Carrier 适配器提供（参见 4.5）
//   → 等待 device.inform.reboot_complete
//   → 推进到 completed / failed
```

### 4.5 4G/5G 差异适配（adapter.go）— 扩展 Carrier 接口

**方式**：在 `Carrier` 接口新增升级/回退相关方法，各适配器实现。遵循项目已有的运营商适配模式。

```go
// internal/core/carrier/carrier.go 新增方法

type Carrier interface {
    // ... 现有方法 ...

    // 升级相关（新增）
    // UpgradeDownloadFileType 返回 TR-069 Download FileType 字符串
    UpgradeDownloadFileType(fileType string) string
    // RollbackEnabled 是否需要先查询回退可用性
    RollbackNeedsEnableCheck(tech model.Technology) bool
    // RollbackParameterPath 回退参数路径
    RollbackParameterPath(tech model.Technology) string
    // RollbackParameterValue 回退参数值
    RollbackParameterValue(tech model.Technology) string
    // UpgradePermissionCode 升级权限码
    UpgradePermissionCode(tech model.Technology) string
    // RollbackPermissionCode 回退权限码
    RollbackPermissionCode(tech model.Technology) string
}
```

各适配器实现差异：

| 方法 | 4G (LTE) | 5G (NR) |
|------|----------|---------|
| `RollbackNeedsEnableCheck` | `true`（先查 ROLLBACK_ENABLE） | `false` |
| `RollbackParameterPath` | `ROLLBACK_CONTROL`（平台映射） | `Device.SoftwareCtrl.ActivateEnable` |
| `UpgradePermissionCode` | `CODE_ENB_UPGRADE_IMAGE` | `CODE_GNB_UPGRADE_IMAGE` |
| `RollbackPermissionCode` | `CODE_ENB_ROLLBACK` | `CODE_GNB_ROLLBACK` |

### 4.6 配置扩展

在 `AppConfig` 中增加 `Upgrade` 配置块，参照 `ProvisionConfig` 模式：

```go
// internal/core/appconfig/config.go

type AppConfig struct {
    // ... 现有字段 ...
    Provision       ProvisionConfig       `mapstructure:"provision"`
    Upgrade         UpgradeConfig         `mapstructure:"upgrade"`   // 新增
    // ...
}

// UpgradeConfig 升级/回退超时和并发配置
type UpgradeConfig struct {
    TaskTimeout            time.Duration `mapstructure:"task_timeout"`              // 单设备升级超时，默认 30min
    WaitDeviceReconnect    time.Duration `mapstructure:"wait_device_reconnect"`     // 等待离线设备重连，默认 1h
    WaitDownloadComplete   time.Duration `mapstructure:"wait_download_complete"`    // 等待文件下载完成，默认 10min
    WaitTransferComplete   time.Duration `mapstructure:"wait_transfer_complete"`    // 等待 TC，默认 30min
    WaitRebootComplete     time.Duration `mapstructure:"wait_reboot_complete"`      // 回退等待重启，默认 5min
    MaxConcurrentPerBatch  int           `mapstructure:"max_concurrent_per_batch"`  // 每批最大并发，默认 5
    ReaperInterval         time.Duration `mapstructure:"reaper_interval"`           // 超时扫描间隔，默认 2min
    UpgradeLockTTL         time.Duration `mapstructure:"upgrade_lock_ttl"`          // Redis 升级锁 TTL，默认 1h
}
```

### 4.7 Handler 端点改造

**保持现有端点结构**，扩展请求体和新增少量端点：

```go
// ===== 主任务（upgrade_tasks）端点 =====
// 前端"任务列表"Tab 查这个
GET    /upgrade-tasks              → 列表查询（分页、按 status/task_type/operator 过滤）
GET    /upgrade-tasks/:id          → 主任务详情（含 success_count/fail_count 实时统计）
POST   /upgrade-tasks              → 创建主任务 + 批量创建子任务
PUT    /upgrade-tasks/:id/suspend   → 挂起主任务（及所有进行中的子任务）
PUT    /upgrade-tasks/:id/resume    → 恢复主任务（重新启动挂起的子任务）
PUT    /upgrade-tasks/:id/terminate → 终止主任务（终止所有进行中的子任务）
POST   /upgrade-tasks/:id/retry    → 重试失败设备
POST   /upgrade-tasks/rollback     → 创建回退任务（task_type=2）

// ===== 子任务（upgrade_sub_tasks）端点 =====
// 前端"设备列表"Tab 查这个
GET    /upgrade-tasks/:id/tasks    → 查询主任务下的子任务列表
GET    /upgrade-sub-tasks/:id      → 单个子任务详情（独立路径，避免与主任务 :id 冲突）

// ===== 固件端点改造 =====
POST   /firmware                     → 支持 file_type, recommend, uploader, manufacturer
PUT    /firmware/:id/recommend       → 切换推荐标识
GET    /firmware/:id/download        → CPE 下载固件文件（MinIO 代理）
```

**设计说明**：
- 前端"任务列表"Tab → `GET /upgrade-tasks`，一行就是一个任务，无需聚合。
- 前端"设备列表"Tab → `GET /upgrade-tasks/:id/tasks`，直接查子任务。
- 批量操作（挂起/恢复/终止）操作主任务表 + 联动更新子任务表。
- 保留原有 `/upgrade-tasks/:id` 单任务查询，兼容单设备升级场景。

### 4.8 service.go 核心改造

```go
// SoftwareService 改造要点

// 1. UploadFirmware 增强
func (s *SoftwareService) UploadFirmware(ctx context.Context, fw *FirmwareVersion, file io.Reader, fileSize int64) error {
    // ① 按 file_type 确定 MinIO 存储目录（img/patch/fpga）
    category := "img"
    switch fw.FileType {
    case FileTypePATCH: category = "patch"
    case FileTypeFPGA:  category = "fpga"
    }
    objectPath := storage.FirmwarePath(category, string(fw.Carrier), fw.ProductClass, fw.Version, fw.FileName)

    // ② tee 文件流：一份写 MinIO，一份算 MD5
    hash := md5.New()
    teeReader := io.TeeReader(file, hash)

    // ③ MinIO PutObject
    s.minioClient.PutObject(ctx, s.firmwareBkt, objectPath, teeReader, fileSize, ...)

    // ④ 计算 MD5
    fw.MD5Val = hex.EncodeToString(hash.Sum(nil))
    // ⑤ 写 DB
    s.firmwareRepo.Create(ctx, fw)
}

// 2. BatchUpgrade 增强（核心改造）
func (s *SoftwareService) BatchUpgrade(ctx context.Context, req BatchUpgradeRequest) ([]UpgradeSubTask, error) {
    taskID := uuid.New()

    for _, deviceID := range req.DeviceIDs {
        task := &UpgradeSubTask{
            DeviceID: deviceID, FirmwareID: req.FirmwareID, TaskID: &taskID,
            Status: UpgradePending,
        }
        s.upgradeRepo.Create(ctx, task)

        // 异步启动执行器（每个设备一个 goroutine，受 semaphore 控制）
        go s.executor.ExecuteOne(context.Background(), task)
    }
}

// 3. Subscribe 事件订阅（参照 provision/Engine 模式）
func (s *SoftwareService) Subscribe(eventBus event.EventBus) error {
    // 已有：TransferComplete
    eventBus.Subscribe(event.SubjectDeviceTransferComplete, s.HandleTransferComplete)
    // 新增：Download Response
    eventBus.Subscribe(event.SubjectCommandDownloadResponse, s.HandleDownloadResponse)
    // 新增：Reboot Complete（回退用）
    eventBus.Subscribe(event.SubjectDeviceRebootComplete, s.HandleRebootComplete)
    // 新增：设备上线（断线恢复）
    eventBus.Subscribe("device.inform.periodic", s.HandleDeviceOnline)
}
```

---

## 五、前端对接方案

### 5.1 当前前端集成架构

```
frontend-core/                        # 共享层
├── services/api/softwareApi.ts       # 真实 API（部分函数仍 delegate 给 mock）
├── hooks/api/useSoftware.ts          # React Query hooks
├── mock/services/softwareService.ts  # Mock 实现
└── mock/data/software.ts             # Mock 数据

webcode/src/pages/software/           # 页面层
├── UpgradePlan/index.tsx             # ❌ 用本地 Mock，未使用 frontend-core hooks
├── FirmwareUpload/index.tsx          # ❌ 用本地 Mock
├── VersionRollback/index.tsx         # ❌ 用本地 Mock
├── VersionQuery/index.tsx            # ✅ 用 useSoftwareVersions hook
└── ActivationPlan/index.tsx          # ✅ 用 useSoftwareVersions hook
```

### 5.2 改造路径

**策略**：页面层不再内嵌 Mock 数据，改为使用 `frontend-core` 的 hooks。hooks 通过 `createApiSwitch(mockService, realApi)` 自动切换 mock/real。

| 步骤 | 文件 | 改动 |
|------|------|------|
| 1 | `frontend-core/services/api/softwareApi.ts` | 新增 `getUpgradeTasks`/`getUpgradeTaskById`/`getSubTasks` 等 API 函数。`getUpgradePlans` 改为直接查 `/upgrade-tasks`（无需聚合） |
| 2 | `frontend-core/types/software.ts` | 新增 `BackendUpgradeTask` 类型（对应 upgrade_tasks 表）。扩展 `BackendFirmwareVersion`（file_type, md5_val, recommend 等） |
| 3 | `webcode/src/pages/software/UpgradePlan/index.tsx` | 删除本地 mockData。"任务列表"Tab 用 `useUpgradeTasks` hook；"设备列表"Tab 用 `useSubTasks(taskId)` hook |
| 4 | `webcode/src/pages/software/FirmwareUpload/index.tsx` | 删除本地 mock，引入 `useSoftwareVersions` + `useUploadSoftwareVersion`。按 file_type tab 过滤 |
| 5 | `webcode/src/pages/software/VersionRollback/index.tsx` | 对接 `createRollback` API，任务列表复用 `useUpgradeTasks` 并按 task_type=2 过滤 |

**API 调用映射**：

| 前端操作 | API 调用 |
|----------|----------|
| 打开升级计划页面 → 任务列表 | `GET /upgrade-tasks?task_type=1&page=1` |
| 点击任务 → 设备列表 | `GET /upgrade-tasks/:id/tasks?page=1` |
| 创建批量升级 | `POST /upgrade-tasks` |
| 挂起/恢复/终止 | `PUT /upgrade-tasks/:id/suspend\|resume\|terminate` |
| 重试失败设备 | `POST /upgrade-tasks/:id/retry` |
| 版本回退 → 创建回退 | `POST /upgrade-tasks/rollback` |

---

## 六、开发任务分解

### Phase 1：数据库 + 模型扩展（2 天）

| 任务 | 文件 | 说明 |
|------|------|------|
| T1.1 | `migrations/000029_firmware_upgrade_enhance.sql` | firmware_versions ALTER + 新建 upgrade_tasks 主任务表 + 新建 upgrade_sub_tasks 子任务表 + ALTER 约束修复 |
| T1.2 | `internal/software/model.go` | 新增 UpgradeTask 主任务 + UpgradeSubTask 子任务模型、FileType/TaskType/TaskStatus 常量 |
| T1.3 | `internal/core/appconfig/config.go` | 新增 UpgradeConfig |

### Phase 2：Repository + Handler 扩展（2 天）

| 任务 | 文件 | 说明 |
|------|------|------|
| T2.1 | `internal/software/repository.go` | 新增 TaskRepository 接口；FirmwareFilter 增加 FileType |
| T2.2 | `pg_task_repository.go` | 新增：upgrade_tasks 主任务 CRUD + IncrementCounts 原子更新 |
| T2.3 | `pg_firmware_repository.go` | 改造：支持新字段读写、file_type 过滤 |
| T2.4 | `pg_upgrade_repository.go` | 改造：upgrade_sub_tasks 子任务新字段、按 task_id 查询、批量状态更新 |
| T2.5 | `internal/software/handler.go` | 新增 `/upgrade-tasks` 路由组 + `/firmware/:id/download` |

### Phase 3：升级执行器（4 天）— 核心

| 任务 | 文件 | 说明 |
|------|------|------|
| T3.1 | `internal/software/executor.go` | 事件驱动执行器：冲突检查、Download 推送、事件回调处理 |
| T3.2 | `internal/software/executor.go` | HandleDownloadResponse、HandleTransferComplete 增强 |
| T3.3 | `internal/software/adapter.go` | Carrier 接口扩展 + CMCC/CTCC/CUCC 4G/5G 适配实现 |
| T3.4 | `internal/software/service.go` | 重构 BatchUpgrade/StartUpgrade，集成 Executor |

### Phase 4：回退执行器（2 天）

| 任务 | 文件 | 说明 |
|------|------|------|
| T4.1 | `internal/software/rollback.go` | SetParameterValues 触发回退 + 订阅 RebootComplete |
| T4.2 | `internal/software/service.go` | RollbackDevices 批量回退 + 超时扫描 |

### Phase 5：前端对接（3 天）

| 任务 | 文件 | 说明 |
|------|------|------|
| T5.1 | `frontend-core/services/api/softwareApi.ts` | 补全所有 API 函数 + 类型对齐 |
| T5.2 | `webcode/src/pages/software/UpgradePlan/index.tsx` | 替换本地 Mock → 使用 hooks |
| T5.3 | `webcode/src/pages/software/FirmwareUpload/index.tsx` | 替换本地 Mock → 使用 hooks |
| T5.4 | `webcode/src/pages/software/VersionRollback/index.tsx` | 对接回退 API |

### Phase 6：测试（2 天）

| 任务 | 说明 |
|------|------|
| T6.1 | 后端单元测试（executor/rollback 事件处理） |
| T6.2 | E2E 测试（新增固件上传、批量升级、回退端点用例） |
| T6.3 | 前端联调 |

---

## 七、关键设计决策

### 7.1 主任务/子任务分表

**决策**：新增 `upgrade_tasks` 主任务表，`upgrade_sub_tasks` 作为子任务表通过 `task_id` 关联。

**理由**：
- 前端页面明确有"任务列表"和"设备列表"两个独立 Tab，分表后查询路径清晰：任务列表查 `upgrade_tasks`，设备列表查 `upgrade_sub_tasks WHERE task_id=?`
- batch 级元数据（task_name、task_type、is_keep_config 等）只存一份在主任务表，子任务表只存设备级数据，消除冗余
- 主任务表的 `success_count`/`fail_count` 提供实时统计，避免每次打开任务详情都要 `GROUP BY` 聚合
- 批量操作（挂起/恢复/终止）直接操作主任务表，语义更清晰
- 与前端 `aggregateTasksIntoPlan()` 的关系：该函数改为直接返回 `UpgradeTask`（无需聚合），子任务列表单独查 `GET /upgrade-tasks/:id/tasks`

### 7.2 事件驱动而非同步步骤循环

**决策**：使用 EventBus 回调推进状态，不用同步 for 循环。

**理由**：
- 项目已有 `provision/Engine` 成功模式：cmdqueue.Push → Subscribe → 回调推进
- TR-069 通信本质是异步的（CPE 发 Inform → ACS 回复 → CPE 执行 → CPE 再 Inform）
- 同步循环会阻塞 goroutine 等待事件，浪费资源
- EventBus 天然支持超时（NACK + 重试）和崩溃恢复（NATS JetStream 持久化）

### 7.3 Redis 做升级锁而非数据库表

**决策**：用 `software:upgrade:active:{deviceSN}` Redis String + TTL 替代 `upgrading_info` 表。

**理由**：
- 参照 `alarm:active:{deviceSN}` 模式（Redis Hash + TTL）
- 升级锁是短生命周期（分钟~小时），任务结束后立即释放
- Redis SETNX 天然支持互斥，比数据库 INSERT + 唯一约束更轻量
- TTL 自动兜底：即使进程崩溃，锁也不会永久占着

### 7.4 扩展 Carrier 接口而非新建 PlatformAdapter

**决策**：在 Carrier 接口增加升级/回退方法。

**理由**：
- 项目已有 `Carrier` 接口 + `CarrierRegistry` + cmcc/ctcc/cucc 适配器
- 4G/5G 差异本质是运营商+制式差异，Carrier 接口正是为此设计
- 新建 PlatformAdapter 会引入第二套适配器体系，增加理解成本
- 运营商差异方法（如参数路径映射）和升级差异方法属于同一抽象层

---

## 八、风险与依赖

| 风险 | 影响 | 缓解 |
|------|------|------|
| EventBus `command.download.response` 事件是否已由 ACS 发布 | 高 | 需确认 ACS session.go 中 Download RPC 的响应是否发布此事件。如未发布，需在 ACS 侧补充 |
| 5G 的 102 UPGRADE FINISH 事件来源 | 中 | ACS 的 Inform 解析需识别 102 事件码并发布专属事件。provision 已处理多种事件码，可参照 |
| cmdqueue deprecated → task.TaskService 迁移 | 中 | 当前 software 仍用 cmdqueue.CommandQueue。若需迁移到 TaskService，可后续统一处理，不影响功能设计 |
| 前端 UpgradePlan 页面 Mock 数据量较大 | 低 | 页面有 ~1500 行，但核心是替换数据源（Mock → hooks），UI 逻辑不变 |

---

## 九、高可用/性能/稳定性设计（生产级审查）

> 本章基于对 `task/TaskService`、`provision/Engine`、`acs/handler.go` 等已验证模块的代码审计，
> 识别当前 software 模块的可靠性缺陷，给出生产级加固方案。

### 9.1 现有代码缺陷（必须修复）

| # | 严重度 | 缺陷 | 位置 | 修复方案 |
|---|--------|------|------|----------|
| D1 | **P0** | **N+1 查询**：`BatchUpgrade()` 循环内对每个设备调用 `firmwareRepo.GetByID()`，但 firmwareID 全部相同 | service.go:215 | 循环外查一次 firmware，缓存后传入 |
| D2 | **P0** | **无事务保护**：`UploadFirmware()` 先 MinIO PutObject 再 DB Insert，若 DB 失败则 MinIO 残留孤立文件 | service.go:60-79 | DB 失败时 `defer` 清理 MinIO 对象 |
| D3 | **P0** | **无事务保护**：`StartUpgrade()` 先 Create 再 UpdateStatus，若后者失败则 task 卡在 pending 永不推进 | service.go:114-129 | 合并为单次 INSERT（直接写入 downloading 状态） |
| D4 | **P0** | **Goroutine 泄漏**：`BatchUpgrade()` 为每个设备 `go func()` 无退出机制，context 不可取消 | service.go:184-251 | 使用 `errgroup` + context + panic recovery |
| D5 | **P1** | **缺少索引**：`CountByTaskStatus()` 用 `GROUP BY status` 但无 `(task_id, status)` 复合索引 | pg_upgrade_repository.go:272 | 3.3 节已新增 `idx_upgrade_sub_tasks_task_id_status` |
| D6 | **P1** | **缺少索引**：`GetActiveByDeviceID()` 用 `WHERE device_id=? AND status NOT IN(...)` 但无复合索引 | pg_upgrade_repository.go:248-270 | 3.3 节已新增 `idx_upgrade_sub_tasks_device_active` |
| D7 | **P1** | **缺少索引**：`List()` 用 `ORDER BY created_at DESC` 但无排序索引 | pg_upgrade_repository.go:187 | 3.2 节已新增 `idx_upgrade_tasks_task_created_at` |
| D8 | **P1** | **唯一约束错误风险**：`firmware_versions` 的 `product_class` 允许 NULL，SQL NULL 不参与唯一约束判定，可能导致重复 | migration 000006 | 唯一约束中 product_class 使用 `COALESCE(product_class, '')` |
| D9 | **P1** | **外键阻塞删除**：`upgrade_tasks.firmware_id REFERENCES firmware_versions(id)` 无 ON DELETE 策略，引用固件无法删除 | migration 000006 | 改为 `ON DELETE SET NULL`（见 3.4 缺陷 1） |
| D10 | **P1** | **无 CHECK 约束**：status/task_type/file_type 等字段允许任意 VARCHAR 值，数据库层面不防护拼写错误 | migration 000006 | 新增 CHECK 约束（见 3.4 缺陷 2） |
| D11 | **P1** | **并发竞态**：`GetActiveByDeviceID` 先查再创建，两个并发请求可同时通过检查创建重复 active task | service.go + pg_repo | 部分唯一索引（见 3.4 缺陷 3） |
| D12 | **P1** | **滥用 error_message**：`SuspendUpgrade` 将挂起前状态存入 error_message，与实际错误消息混淆 | service.go:333 | 新增 `pre_suspend_status` 专用字段（见 3.4 缺陷 4） |
| D13 | **P1** | **统计计数竞态**：多个子任务同时完成时 `success_count++` 的读-改-写无原子保护 | upgrade_tasks 主任务表 | SQL 原子递增（见 3.4 缺陷 5） |
| D14 | **P2** | **FirmwareRepository 无 Update 方法**：无法修改固件元数据（recommend、description 等） | repository.go | 新增 Update 方法 |
| D15 | **P2** | **RollbackUpgrade 是空实现**：只改状态为 failed 并在 error_message 写 "rolled back"，无实际回退逻辑 | service.go:409-440 | Phase 4 重写为真正回退流程 |
| D16 | **P2** | **Repository 接受 `*pgxpool.Pool`** 而非 `storage.DB` 接口：无法参与共享事务、无法 Mock 测试 | pg_*.go | 后续统一重构，本次不改（范围控制） |

### 9.2 可靠性设计（对齐 provision/Engine 和 task/TaskService）

#### 9.2.1 事务保护

```go
// UploadFirmware — 先上传再入库，失败则清理
func (s *SoftwareService) UploadFirmware(ctx context.Context, fw *FirmwareVersion, file io.Reader, fileSize int64) error {
    // ... MD5 计算 + MinIO PutObject ...

    if err := s.firmwareRepo.Create(ctx, fw); err != nil {
        // ★ 清理 MinIO 孤立文件
        if delErr := s.minioClient.RemoveObject(ctx, s.firmwareBkt, fw.MinIOPath, minio.RemoveObjectOptions{}); delErr != nil {
            s.logger.Error("cleanup orphaned firmware file", zap.String("path", fw.MinIOPath), zap.Error(delErr))
        }
        return fmt.Errorf("create firmware record: %w", err)
    }
    return nil
}

// StartUpgrade — 单条 SQL 完成创建+状态设置，避免两步非原子操作
func (s *SoftwareService) startUpgradeTask(ctx context.Context, task *UpgradeTask) error {
    task.Status = UpgradeDownloading  // 直接以 downloading 状态创建
    task.MaxRetries = 3
    return s.upgradeRepo.Create(ctx, task)  // 一次 INSERT
}
```

#### 9.2.2 Goroutine 生命周期管理

```go
// BatchUpgrade — 使用 errgroup 管理并发，支持取消
func (s *SoftwareService) BatchUpgrade(ctx context.Context, req BatchUpgradeRequest) ([]UpgradeSubTask, error) {
    fw, err := s.firmwareRepo.GetByID(ctx, req.FirmwareID)  // ★ 循环外查一次
    if err != nil {
        return nil, fmt.Errorf("get firmware: %w", err)
    }

    concurrency := req.Concurrency
    if concurrency < 1 { concurrency = 5 }

    taskID := uuid.New()
    tasks := make([]UpgradeSubTask, 0, len(req.DeviceIDs))

    // ★ errgroup + semaphore 控制并发和生命周期
    g, gctx := errgroup.WithContext(ctx)
    sem := make(chan struct{}, concurrency)
    var mu sync.Mutex

    for _, deviceID := range req.DeviceIDs {
        if gctx.Err() != nil { break }  // 上下文取消则停止

        task := &UpgradeSubTask{...}
        if err := s.upgradeRepo.Create(ctx, task); err != nil { continue }

        sem <- struct{}{}
        g.Go(func() error {
            defer func() { <-sem }()
            defer func() {
                if r := recover(); r != nil {  // ★ goroutine panic 恢复
                    s.logger.Error("upgrade executor panic", zap.String("task_id", task.ID.String()), zap.Any("recover", r))
                }
            }()
            return s.executor.ExecuteOne(gctx, task, fw)
        })

        mu.Lock()
        tasks = append(tasks, *task)
        mu.Unlock()
    }

    // 等待所有 goroutine 完成，但不传播单个设备的失败
    _ = g.Wait()
    return tasks, nil
}
```

#### 9.2.3 超时扫描（TaskReaper 模式）

参照 `provision/Engine.StartTaskReaper()`：

```go
// StartTaskReaper 定时扫描超时的升级任务
func (s *SoftwareService) StartTaskReaper(ctx context.Context) {
    interval := s.config.ReaperInterval
    if interval == 0 { interval = 2 * time.Minute }

    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            s.reapStaleTasks(ctx)
        }
    }
}

func (s *SoftwareService) reapStaleTasks(ctx context.Context) {
    cutoff := time.Now().Add(-s.config.TaskTimeout)
    // 单条 SQL 原子更新所有超时子任务
    // UPDATE upgrade_sub_tasks SET status='failed', error_message='task timeout'
    //   WHERE status NOT IN ('completed','failed','terminated')
    //     AND updated_at < $1
    affected, err := s.upgradeRepo.FailStale(ctx, cutoff)
    if err != nil {
        s.logger.Error("reap stale upgrade tasks", zap.Error(err))
        return
    }
    if affected > 0 {
        s.logger.Warn("reaped stale upgrade tasks", zap.Int64("count", affected))
        metrics.UpgradeTasksReaped.Add(float64(affected))
    }
}
```

#### 9.2.4 崩溃恢复（对齐 TaskService.RestorePendingQueues）

```go
// RestorePendingUpgrades 启动时恢复未完成的升级任务
// 参照 task/service.go RestorePendingQueues
func (s *SoftwareService) RestorePendingUpgrades(ctx context.Context) error {
    // 查所有 downloading/rebooting 状态的 task（进程崩溃前正在执行）
    tasks, err := s.upgradeRepo.ListStaleByStatus(ctx, []UpgradeState{
        UpgradeDownloading, UpgradeRebooting, UpgradeVerifying,
    })
    if err != nil {
        return fmt.Errorf("list stale upgrade tasks: %w", err)
    }

    for _, task := range tasks {
        // 检查设备是否仍然活跃（心跳检查）
        dev, err := s.deviceRepo.GetByID(ctx, task.DeviceID)
        if err != nil || dev == nil { continue }

        // ★ 使用 Redis heartbeat 判断设备是否在线
        heartbeatKey := fmt.Sprintf("acs:heartbeat:%s", dev.SerialNumber)
        exists, _ := s.redis.Exists(ctx, heartbeatKey).Result()
        if exists == 1 {
            // 设备在线 → 重新发送 Connection Request 触发续传
            s.logger.Info("restoring pending upgrade", zap.String("task_id", task.ID.String()))
            go s.executor.ResumeUpgrade(context.Background(), &task)
        } else {
            // 设备离线 → 写入等待恢复队列
            waitKey := fmt.Sprintf("software:upgrade:wait:%s", dev.SerialNumber)
            s.redis.Set(ctx, waitKey, task.ID.String(), s.config.WaitDeviceReconnect)
        }
    }
    return nil
}
```

#### 9.2.5 幂等性保证

| 操作 | 幂等策略 |
|------|----------|
| UploadFirmware | DB 唯一约束 `(carrier, product_class, version, file_type)` + `ON CONFLICT` 返回已有记录 |
| CreateUpgradeTask | `GetActiveByDeviceID` 先查再创建；Redis `SETNX` 做设备级互斥锁 |
| HandleTransferComplete | 先检查 task.Status 是否仍为 `downloading`（非则忽略），再推进。防止重复回调 |
| HandleRebootComplete | 同上：检查 task.Status 是否为 `rebooting` |
| BatchSuspend/Resume/Terminate | `WHERE task_id=? AND status NOT IN (completed, failed, terminated)` 条件更新 |

```go
// HandleTransferComplete — 幂等处理
func (s *SoftwareService) HandleTransferComplete(ctx context.Context, evt event.Event) error {
    // ... 解析 deviceSN ...
    task, err := s.upgradeRepo.GetActiveByDeviceID(ctx, dev.ID)
    if err != nil { return nil } // 无活跃任务，忽略

    // ★ 幂等：仅 downloading 状态才处理
    if task.Status != UpgradeDownloading {
        s.logger.Debug("ignore TC for non-downloading task",
            zap.String("task_id", task.ID.String()),
            zap.String("status", string(task.Status)))
        return nil
    }

    // 解析 TC FaultCode...
    // 推进状态...
}
```

### 9.3 并发安全

| 场景 | 保护机制 | 参照 |
|------|----------|------|
| 同一设备同时触发两次升级 | Redis `SETNX software:upgrade:active:{deviceSN}` + TTL 1h | `acs:connreq:pending:*` 去重模式 |
| 批量操作中的 task 列表竞争 | `sync.Mutex` 保护 tasks slice | `errgroup` + `sync.Mutex` 模式 |
| 事件回调并发修改同一 task | DB `WHERE id=? AND status=?` 条件更新（乐观锁） | `pgx` 行级锁 |
| Redis 操作原子性 | Pipeline 批量执行（SETNX + EXPIRE） | `task/redis_queue.go` Pipeline 模式 |

```go
// 设备级互斥锁
func (s *SoftwareService) acquireDeviceLock(ctx context.Context, deviceSN string, taskID uuid.UUID) (bool, error) {
    key := fmt.Sprintf("software:upgrade:active:%s", deviceSN)
    // ★ SETNX + EXPIRE 原子操作（Pipeline）
    ok, err := s.redis.SetNX(ctx, key, taskID.String(), s.config.UpgradeLockTTL).Result()
    if err != nil {
        return false, fmt.Errorf("acquire device lock: %w", err)
    }
    return ok, nil
}

func (s *SoftwareService) releaseDeviceLock(ctx context.Context, deviceSN string) {
    key := fmt.Sprintf("software:upgrade:active:%s", deviceSN)
    s.redis.Del(ctx, key)
}
```

### 9.4 可观测性（对齐现有 Prometheus 指标体系）

```go
// internal/software/metrics.go — 新增
var (
    // 计数器
    UpgradeTasksCreated   = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "software_upgrade_tasks_created_total",
        Help: "Total number of upgrade tasks created",
    }, []string{"task_type", "carrier"})
    UpgradeTasksCompleted = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "software_upgrade_tasks_completed_total",
        Help: "Total number of upgrade tasks completed",
    }, []string{"task_type", "result"}) // result: success/failed/terminated
    UpgradeTasksReaped   = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "software_upgrade_tasks_reaped_total",
        Help: "Number of upgrade tasks reaped by timeout",
    })
    FirmwareUploads      = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "software_firmware_uploads_total",
        Help: "Total firmware upload operations",
    }, []string{"file_type", "result"}) // result: success/failed

    // 直方图
    UpgradeDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "software_upgrade_duration_seconds",
        Help:    "Time from task creation to completion",
        Buckets: []float64{30, 60, 120, 300, 600, 1800, 3600}, // 30s ~ 1h
    }, []string{"task_type"})

    // 仪表盘
    ActiveUpgrades = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "software_active_upgrades",
        Help: "Number of currently active upgrade tasks",
    })
)
```

OpenTelemetry 链路追踪（对齐 `task/tracing.go` 和 `provision/tracing.go`）：

```go
// executor.go 中关键操作创建 span
func (e *UpgradeExecutor) ExecuteOne(ctx context.Context, task *UpgradeSubTask, fw *FirmwareVersion) error {
    ctx, span := otel.Tracer("software").Start(ctx, "upgrade.execute",
        trace.WithAttributes(
            attribute.String("task.id", task.ID.String()),
            attribute.String("device.sn", task.DeviceSN),
            attribute.String("firmware.version", fw.Version),
        ))
    defer span.End()
    // ...
}
```

### 9.5 降级与容错

| 故障场景 | 影响 | 容错策略 |
|----------|------|----------|
| Redis 不可用 | 无法获取设备锁、无法断线恢复 | 降级为 DB 查询：`GetActiveByDeviceID` 做冲突检查（性能下降但功能不中断） |
| MinIO 上传失败 | 固件无法存储 | 返回错误，不写 DB。用户可重试上传 |
| EventBus (NATS) 断连 | 升级事件回调延迟 | NATS 自动重连 + JetStream 持久化，事件不丢失。TaskReaper 兜底超时处理 |
| DB 不可用 | 无法创建/更新任务 | 依赖 `components/health` 健康检查，API 返回 503。TaskService 的最终一致性模式可恢复 |
| CPE 下载中断 | 设备可能处于不稳定状态 | TransferComplete 超时后版本比对容错：查设备当前版本 vs 目标版本，一致则视为成功 |
| App 进程崩溃 | 正在执行的升级中断 | 启动时 `RestorePendingUpgrades` 恢复；Redis 锁 TTL 自动过期 |
| 设备升级后重启无响应 | 5G 设备升级可能成功但不发 102 事件 | TaskReaper 超时后查设备版本比对（设计文档 Step 4 容错机制） |

### 9.6 性能基线与优化

| 操作 | 10 万基站基线 | 优化手段 |
|------|-------------|----------|
| 固件上传 | ~1 次/天，非热路径 | MD5 流式计算（TeeReader），不额外读磁盘 |
| 批量升级创建 | 1 次创建 1000 个 task | 循环外查 firmware（消除 N+1）；批量 INSERT（单事务 `tx.Exec` 多行 VALUES） |
| 升级任务列表查询 | 管理员查看，~10 QPS | `(created_at DESC)` 索引支持排序；考虑 keyset 分页（替代 OFFSET） |
| CountByTaskStatus 聚合 | 每个批次每次刷新 | `idx_upgrade_sub_tasks_task_id_status` 复合索引覆盖 |
| 事件回调处理 | ~5000 并发设备升级中 | 按 deviceSN 查 active sub_task 走 `idx_upgrade_sub_tasks_device_active` 索引 |
| Redis 升级锁 | ~5000 SETNX/s | 单 key 操作，Redis 单线程保证原子性 |

**批量 INSERT 优化**（当前逐行 INSERT，1000 设备 = 1000 次 DB 往返）：

```go
// BatchCreateSubTasks — 批量插入子任务（单事务多行 VALUES）
func (r *PgUpgradeSubTaskRepository) BatchCreate(ctx context.Context, tasks []*UpgradeSubTask) error {
    tx, err := r.pool.Begin(ctx)
    if err != nil { return fmt.Errorf("begin tx: %w", err) }
    defer tx.Rollback(ctx)

    // 构造多行 VALUES: ($1, $2, ...), ($N+1, $N+2, ...), ...
    builder := storage.Psql.Insert("upgrade_sub_tasks").
        Columns("device_id", "firmware_id", "task_id", "status", "max_retries",
                "device_sn", "ori_version", "dest_version")
        // ★ 注意：task_name/task_type/is_keep_config/product_class/file_name/file_size/file_md5
        //   属于主任务表 upgrade_tasks，不在子任务表中

    for _, t := range tasks {
        builder = builder.Values(
            t.DeviceID, t.FirmwareID, t.TaskID, t.Status, t.MaxRetries,
            t.DeviceSN, t.OriVersion, t.DestVersion,
        )
    }

    query, args, err := builder.Suffix("RETURNING id").ToSql()
    // ...
    return tx.Commit(ctx)
}
```

### 9.7 启动与关闭生命周期

```
App 启动流程（cmd/app/main.go）：
  ① 基础设施初始化（DB/Redis/NATS/MinIO）
  ② 模块初始化（modules.go → initSoftwareModule）
     → 创建 Service + Executor
     → Subscribe 事件订阅
     → RestorePendingUpgrades（恢复未完成任务）
     → StartTaskReaper（启动超时扫描）
  ③ 注册路由
  ④ 启动 HTTP 服务

App 关闭流程（graceful shutdown）：
  ① 停止接收新请求
  ② 等待进行中的升级回调完成（context cancel 传播）
  ③ TaskReaper ticker.Stop()
  ④ EventBus.Close()
  ⑤ DB pool.Close()
```

### 9.8 生产级自检清单

| 检查项 | 状态 | 说明 |
|--------|------|------|
| 所有 DB 写操作有错误处理和回滚 | ✅ | UploadFirmware 清理 MinIO；TaskCreate 用事务 |
| 无 goroutine 泄漏 | ✅ | errgroup + context 取消 + panic recovery |
| Redis Key 全部有 TTL | ✅ | active 锁 1h、wait 锁 1h、cmdkey 映射 30min |
| 幂等：重复回调不产生副作用 | ✅ | 状态前置检查 + 条件更新 |
| 超时兜底：TaskReaper 定期扫描 | ✅ | 对齐 provision TaskReaper |
| 崩溃恢复：启动时恢复 pending 任务 | ✅ | RestorePendingUpgrades 对齐 TaskService |
| Prometheus 指标覆盖关键操作 | ✅ | 创建/完成/超时/时长/活跃数 |
| OpenTelemetry 链路追踪 | ✅ | 关键操作创建 span |
| 索引覆盖热查询路径 | ✅ | 6 个新增索引 + 1 个部分唯一索引 |
| N+1 查询消除 | ✅ | firmware 循环外查询；TaskCreate 批量 INSERT |
| 固件唯一约束保留语义正确性 | ✅ | `(carrier, product_class, version, file_type)` |
| 外键不阻塞业务操作 | ✅ | firmware_id ON DELETE SET NULL |
| 状态/类型字段有 CHECK 约束 | ✅ | 6 个 CHECK 约束覆盖所有枚举字段 |
| 并发安全：设备级升级互斥 | ✅ | 部分唯一索引 + Redis SETNX 双重保护 |
| 统计计数原子更新 | ✅ | `success_count = success_count + $1` SQL 原子递增 |
| 挂起恢复状态存储在专用字段 | ✅ | pre_suspend_status 替代 error_message 滥用 |
