-- +goose Up
-- ============================================================
-- 000029_software_upgrade_enhance.up.sql
-- 固件版本扩展字段 + upgrade_tasks 改造为主任务表 + 新建 upgrade_sub_tasks 子任务表
-- ============================================================

-- ============================================================
-- Step 1: 新建 upgrade_sub_tasks 子任务表
-- ============================================================
CREATE TABLE IF NOT EXISTS upgrade_sub_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id         UUID NOT NULL REFERENCES upgrade_tasks(id) ON DELETE CASCADE,
    device_id       UUID NOT NULL,
    firmware_id     UUID REFERENCES firmware_versions(id) ON DELETE SET NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    error_message   TEXT,
    retry_count     INTEGER NOT NULL DEFAULT 0,
    max_retries     INTEGER NOT NULL DEFAULT 3,
    device_sn       VARCHAR(64),
    ori_version     VARCHAR(64),
    dest_version    VARCHAR(64),
    command_key     VARCHAR(256),
    failure_reason  TEXT,
    pre_suspend_status VARCHAR(20),
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 子任务索引
CREATE INDEX IF NOT EXISTS idx_upgrade_sub_tasks_task_id_status
    ON upgrade_sub_tasks (task_id, status);
CREATE INDEX IF NOT EXISTS idx_upgrade_sub_tasks_device_sn
    ON upgrade_sub_tasks (device_sn);
CREATE INDEX IF NOT EXISTS idx_upgrade_sub_tasks_device_active
    ON upgrade_sub_tasks (device_id, status);
CREATE INDEX IF NOT EXISTS idx_upgrade_sub_tasks_created_at
    ON upgrade_sub_tasks (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_upgrade_sub_tasks_command_key
    ON upgrade_sub_tasks (command_key) WHERE command_key IS NOT NULL;

-- 同一设备最多一条活跃子任务（数据库级互斥）
CREATE UNIQUE INDEX IF NOT EXISTS idx_upgrade_sub_tasks_device_active_uniq
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

-- ============================================================
-- Step 2: 迁移已有设备级数据到 upgrade_sub_tasks
-- ============================================================
INSERT INTO upgrade_sub_tasks (id, task_id, device_id, firmware_id, status, error_message,
    retry_count, max_retries, started_at, completed_at, created_at, updated_at)
SELECT id,
    COALESCE(batch_id, gen_random_uuid()),
    device_id, firmware_id, status, error_message,
    retry_count, max_retries, started_at, completed_at, created_at, updated_at
FROM upgrade_tasks
WHERE device_id IS NOT NULL
ON CONFLICT DO NOTHING;

-- 删除已迁移的设备级行
DELETE FROM upgrade_tasks WHERE device_id IS NOT NULL;

-- ============================================================
-- Step 3: ALTER upgrade_tasks 添加主任务级新列
-- ============================================================
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS task_name       VARCHAR(256);
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS task_type       SMALLINT NOT NULL DEFAULT 1;
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS file_name       VARCHAR(256);
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS file_md5        VARCHAR(64);
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS result          VARCHAR(20);
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS operator_code   VARCHAR(8) NOT NULL DEFAULT 'cmcc';
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS product_class   VARCHAR(64);
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS is_keep_config  BOOLEAN DEFAULT true;
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS create_status   VARCHAR(16) NOT NULL DEFAULT 'active';
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS create_user     VARCHAR(64) NOT NULL DEFAULT 'system';
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS total_count     INTEGER NOT NULL DEFAULT 0;
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS success_count   INTEGER NOT NULL DEFAULT 0;
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS fail_count      INTEGER NOT NULL DEFAULT 0;
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS max_concurrent  INTEGER DEFAULT 5;
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS ended_at        TIMESTAMPTZ;

-- ============================================================
-- Step 4: 清理旧列（设备级数据已迁移到 upgrade_sub_tasks）
-- ============================================================
ALTER TABLE upgrade_tasks DROP COLUMN IF EXISTS device_id;
ALTER TABLE upgrade_tasks DROP COLUMN IF EXISTS batch_id;

-- ============================================================
-- Step 5: upgrade_tasks 索引
-- ============================================================
CREATE INDEX IF NOT EXISTS idx_upgrade_tasks_task_status ON upgrade_tasks (status);
CREATE INDEX IF NOT EXISTS idx_upgrade_tasks_task_operator ON upgrade_tasks (operator_code);
CREATE INDEX IF NOT EXISTS idx_upgrade_tasks_task_product_class ON upgrade_tasks (product_class);
CREATE INDEX IF NOT EXISTS idx_upgrade_tasks_task_create_user ON upgrade_tasks (create_user);
CREATE INDEX IF NOT EXISTS idx_upgrade_tasks_task_created_at ON upgrade_tasks (created_at DESC);

-- ============================================================
-- Step 6: upgrade_tasks CHECK 约束
-- ============================================================
ALTER TABLE upgrade_tasks ADD CONSTRAINT chk_upgrade_tasks_status
    CHECK (status IN ('pending', 'in_progress', 'suspended', 'ended'));
ALTER TABLE upgrade_tasks ADD CONSTRAINT chk_upgrade_tasks_result
    CHECK (result IS NULL OR result IN ('success', 'partial', 'failed', 'terminated'));
ALTER TABLE upgrade_tasks ADD CONSTRAINT chk_upgrade_tasks_task_type
    CHECK (task_type IN (1, 2, 4, 6, 8));
ALTER TABLE upgrade_tasks ADD CONSTRAINT chk_upgrade_tasks_create_status
    CHECK (create_status IN ('active', 'suspend', 'timing'));

-- ============================================================
-- Step 7: 修复 upgrade_tasks FK（固件删除时解除引用）
-- ============================================================
ALTER TABLE upgrade_tasks DROP CONSTRAINT IF EXISTS upgrade_tasks_firmware_id_fkey;
ALTER TABLE upgrade_tasks ADD CONSTRAINT upgrade_tasks_firmware_id_fkey
    FOREIGN KEY (firmware_id) REFERENCES firmware_versions(id) ON DELETE SET NULL;

-- ============================================================
-- Step 8: firmware_versions 扩展字段
-- ============================================================
ALTER TABLE firmware_versions ADD COLUMN IF NOT EXISTS file_type SMALLINT NOT NULL DEFAULT 0;
ALTER TABLE firmware_versions ADD COLUMN IF NOT EXISTS md5_val VARCHAR(64);
ALTER TABLE firmware_versions ADD COLUMN IF NOT EXISTS recommend BOOLEAN DEFAULT false;
ALTER TABLE firmware_versions ADD COLUMN IF NOT EXISTS uploader VARCHAR(64);
ALTER TABLE firmware_versions ADD COLUMN IF NOT EXISTS manufacturer VARCHAR(128);
ALTER TABLE firmware_versions ADD COLUMN IF NOT EXISTS description TEXT;

-- 唯一约束：使用 COALESCE 处理 NULL product_class，加入 file_type
DROP INDEX IF EXISTS idx_firmware_unique_version;
CREATE UNIQUE INDEX IF NOT EXISTS idx_firmware_unique_version
    ON firmware_versions (carrier, COALESCE(product_class, ''), version, file_type);

-- 文件类型 + 状态 复合索引
CREATE INDEX IF NOT EXISTS idx_firmware_file_type_status
    ON firmware_versions (file_type, status);

-- firmware_versions CHECK 约束
ALTER TABLE firmware_versions ADD CONSTRAINT chk_firmware_versions_file_type
    CHECK (file_type IN (0, 1, 6));
ALTER TABLE firmware_versions ADD CONSTRAINT chk_firmware_versions_status
    CHECK (status IN ('active', 'deprecated', 'archived'));

-- +goose Down
-- ============================================================
-- 回滚：按相反顺序删除所有新增对象
-- ============================================================

-- firmware_versions 回滚
ALTER TABLE firmware_versions DROP CONSTRAINT IF EXISTS chk_firmware_versions_status;
ALTER TABLE firmware_versions DROP CONSTRAINT IF EXISTS chk_firmware_versions_file_type;
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

-- upgrade_tasks 回滚
ALTER TABLE upgrade_tasks DROP CONSTRAINT IF EXISTS upgrade_tasks_firmware_id_fkey;
ALTER TABLE upgrade_tasks ADD CONSTRAINT upgrade_tasks_firmware_id_fkey
    FOREIGN KEY (firmware_id) REFERENCES firmware_versions(id);
ALTER TABLE upgrade_tasks DROP CONSTRAINT IF EXISTS chk_upgrade_tasks_create_status;
ALTER TABLE upgrade_tasks DROP CONSTRAINT IF EXISTS chk_upgrade_tasks_task_type;
ALTER TABLE upgrade_tasks DROP CONSTRAINT IF EXISTS chk_upgrade_tasks_result;
ALTER TABLE upgrade_tasks DROP CONSTRAINT IF EXISTS chk_upgrade_tasks_status;
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
-- 恢复旧列结构（仅恢复列定义，不恢复数据）
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS device_id UUID;
ALTER TABLE upgrade_tasks ADD COLUMN IF NOT EXISTS batch_id UUID;

-- upgrade_sub_tasks 回滚
DROP TRIGGER IF EXISTS trigger_upgrade_sub_tasks_updated_at ON upgrade_sub_tasks;
ALTER TABLE upgrade_sub_tasks DROP CONSTRAINT IF EXISTS chk_upgrade_sub_tasks_status;
DROP INDEX IF EXISTS idx_upgrade_sub_tasks_device_active_uniq;
DROP INDEX IF EXISTS idx_upgrade_sub_tasks_command_key;
DROP INDEX IF EXISTS idx_upgrade_sub_tasks_created_at;
DROP INDEX IF EXISTS idx_upgrade_sub_tasks_device_active;
DROP INDEX IF EXISTS idx_upgrade_sub_tasks_device_sn;
DROP INDEX IF EXISTS idx_upgrade_sub_tasks_task_id_status;
DROP TABLE IF EXISTS upgrade_sub_tasks;
