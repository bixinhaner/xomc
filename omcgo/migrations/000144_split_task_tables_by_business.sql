-- +goose Up
-- 文件传输任务表按业务拆分 - S1 建表（不迁移数据）
--
-- 设计稿：docs/design/task-tables-split-by-business-20260521.md
-- 拆分策略：
--   · 升级 / 回退 继续用 upgrade_tasks / upgrade_sub_tasks（保留旧表）
--   · 4 类业务各自一对新表：
--       config_backup_*       — 配置文件备份（NV/XML 通过 download_file_type 字段区分）
--       config_restore_*      — 配置文件下发
--       runtime_log_collect_* — 运行日志收集
--       fault_log_collect_*   — 异常日志收集
--   · 新表 schema 跟旧表完全一致（CREATE TABLE LIKE upgrade_* INCLUDING DEFAULTS
--     INCLUDING CONSTRAINTS）。索引/外键/触发器需要显式重建（LIKE 默认不复制）。
--
-- 跨业务设备锁：新建 device_active_tasks 中间表（PK device_id），保证同设备同时
-- 只能跑一个文件传输任务（含旧表升级/回退）。各业务 Executor 在创建 sub_task 前
-- 先 INSERT 此表，冲突 → 报"设备已有进行中任务"；sub_task 终态 → DELETE 此行。

-- ============================================================
-- 1. config_backup_tasks / sub_tasks
-- ============================================================
CREATE TABLE IF NOT EXISTS config_backup_tasks (LIKE upgrade_tasks INCLUDING DEFAULTS INCLUDING CONSTRAINTS);
ALTER TABLE config_backup_tasks ADD PRIMARY KEY (id);

CREATE TABLE IF NOT EXISTS config_backup_sub_tasks (LIKE upgrade_sub_tasks INCLUDING DEFAULTS INCLUDING CONSTRAINTS);
ALTER TABLE config_backup_sub_tasks ADD PRIMARY KEY (id);
ALTER TABLE config_backup_sub_tasks
    ADD CONSTRAINT config_backup_sub_tasks_task_id_fkey
        FOREIGN KEY (task_id) REFERENCES config_backup_tasks(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_config_backup_sub_tasks_command_key ON config_backup_sub_tasks (command_key) WHERE command_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_config_backup_sub_tasks_created_at  ON config_backup_sub_tasks (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_config_backup_sub_tasks_device_sn   ON config_backup_sub_tasks (device_sn);
CREATE INDEX IF NOT EXISTS idx_config_backup_sub_tasks_task_status ON config_backup_sub_tasks (task_id, status);

CREATE TRIGGER trigger_config_backup_tasks_updated_at     BEFORE UPDATE ON config_backup_tasks     FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trigger_config_backup_sub_tasks_updated_at BEFORE UPDATE ON config_backup_sub_tasks FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();


-- ============================================================
-- 2. config_restore_tasks / sub_tasks
-- ============================================================
CREATE TABLE IF NOT EXISTS config_restore_tasks (LIKE upgrade_tasks INCLUDING DEFAULTS INCLUDING CONSTRAINTS);
ALTER TABLE config_restore_tasks ADD PRIMARY KEY (id);

CREATE TABLE IF NOT EXISTS config_restore_sub_tasks (LIKE upgrade_sub_tasks INCLUDING DEFAULTS INCLUDING CONSTRAINTS);
ALTER TABLE config_restore_sub_tasks ADD PRIMARY KEY (id);
ALTER TABLE config_restore_sub_tasks
    ADD CONSTRAINT config_restore_sub_tasks_task_id_fkey
        FOREIGN KEY (task_id) REFERENCES config_restore_tasks(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_config_restore_sub_tasks_command_key ON config_restore_sub_tasks (command_key) WHERE command_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_config_restore_sub_tasks_created_at  ON config_restore_sub_tasks (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_config_restore_sub_tasks_device_sn   ON config_restore_sub_tasks (device_sn);
CREATE INDEX IF NOT EXISTS idx_config_restore_sub_tasks_task_status ON config_restore_sub_tasks (task_id, status);

CREATE TRIGGER trigger_config_restore_tasks_updated_at     BEFORE UPDATE ON config_restore_tasks     FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trigger_config_restore_sub_tasks_updated_at BEFORE UPDATE ON config_restore_sub_tasks FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();


-- ============================================================
-- 3. runtime_log_collect_tasks / sub_tasks
-- ============================================================
CREATE TABLE IF NOT EXISTS runtime_log_collect_tasks (LIKE upgrade_tasks INCLUDING DEFAULTS INCLUDING CONSTRAINTS);
ALTER TABLE runtime_log_collect_tasks ADD PRIMARY KEY (id);

CREATE TABLE IF NOT EXISTS runtime_log_collect_sub_tasks (LIKE upgrade_sub_tasks INCLUDING DEFAULTS INCLUDING CONSTRAINTS);
ALTER TABLE runtime_log_collect_sub_tasks ADD PRIMARY KEY (id);
ALTER TABLE runtime_log_collect_sub_tasks
    ADD CONSTRAINT runtime_log_collect_sub_tasks_task_id_fkey
        FOREIGN KEY (task_id) REFERENCES runtime_log_collect_tasks(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_runtime_log_collect_sub_tasks_command_key ON runtime_log_collect_sub_tasks (command_key) WHERE command_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_runtime_log_collect_sub_tasks_created_at  ON runtime_log_collect_sub_tasks (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_runtime_log_collect_sub_tasks_device_sn   ON runtime_log_collect_sub_tasks (device_sn);
CREATE INDEX IF NOT EXISTS idx_runtime_log_collect_sub_tasks_task_status ON runtime_log_collect_sub_tasks (task_id, status);

CREATE TRIGGER trigger_runtime_log_collect_tasks_updated_at     BEFORE UPDATE ON runtime_log_collect_tasks     FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trigger_runtime_log_collect_sub_tasks_updated_at BEFORE UPDATE ON runtime_log_collect_sub_tasks FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();


-- ============================================================
-- 4. fault_log_collect_tasks / sub_tasks
-- ============================================================
CREATE TABLE IF NOT EXISTS fault_log_collect_tasks (LIKE upgrade_tasks INCLUDING DEFAULTS INCLUDING CONSTRAINTS);
ALTER TABLE fault_log_collect_tasks ADD PRIMARY KEY (id);

CREATE TABLE IF NOT EXISTS fault_log_collect_sub_tasks (LIKE upgrade_sub_tasks INCLUDING DEFAULTS INCLUDING CONSTRAINTS);
ALTER TABLE fault_log_collect_sub_tasks ADD PRIMARY KEY (id);
ALTER TABLE fault_log_collect_sub_tasks
    ADD CONSTRAINT fault_log_collect_sub_tasks_task_id_fkey
        FOREIGN KEY (task_id) REFERENCES fault_log_collect_tasks(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_fault_log_collect_sub_tasks_command_key ON fault_log_collect_sub_tasks (command_key) WHERE command_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_fault_log_collect_sub_tasks_created_at  ON fault_log_collect_sub_tasks (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_fault_log_collect_sub_tasks_device_sn   ON fault_log_collect_sub_tasks (device_sn);
CREATE INDEX IF NOT EXISTS idx_fault_log_collect_sub_tasks_task_status ON fault_log_collect_sub_tasks (task_id, status);

CREATE TRIGGER trigger_fault_log_collect_tasks_updated_at     BEFORE UPDATE ON fault_log_collect_tasks     FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trigger_fault_log_collect_sub_tasks_updated_at BEFORE UPDATE ON fault_log_collect_sub_tasks FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();


-- ============================================================
-- 5. device_active_tasks - 跨业务设备唯一锁
-- ============================================================
-- 替代旧 idx_upgrade_sub_tasks_device_active_uniq 的"同设备唯一活跃任务"约束。
-- 拆表后 5 张 sub_task 表（旧的 upgrade_sub_tasks + 4 张新表）都参与此约束。
-- 各业务 Executor 在创建 sub_task 前先 INSERT 此表，PK 冲突即"设备已有任务"；
-- sub_task 进 terminal 状态（completed/failed/terminated）时 DELETE 此行。
-- business_type 枚举：upgrade / rollback / config_backup / config_restore /
-- runtime_log_collect / fault_log_collect。
CREATE TABLE IF NOT EXISTS device_active_tasks (
    device_id     UUID         PRIMARY KEY,
    sub_task_id   UUID         NOT NULL,
    business_type VARCHAR(32)  NOT NULL CHECK (business_type IN (
        'upgrade', 'rollback',
        'config_backup', 'config_restore',
        'runtime_log_collect', 'fault_log_collect'
    )),
    sub_task_table VARCHAR(64) NOT NULL,
    acquired_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_device_active_tasks_business ON device_active_tasks (business_type);
CREATE INDEX IF NOT EXISTS idx_device_active_tasks_sub_task ON device_active_tasks (sub_task_id);


-- +goose Down
DROP TABLE IF EXISTS device_active_tasks;
DROP TABLE IF EXISTS fault_log_collect_sub_tasks;
DROP TABLE IF EXISTS fault_log_collect_tasks;
DROP TABLE IF EXISTS runtime_log_collect_sub_tasks;
DROP TABLE IF EXISTS runtime_log_collect_tasks;
DROP TABLE IF EXISTS config_restore_sub_tasks;
DROP TABLE IF EXISTS config_restore_tasks;
DROP TABLE IF EXISTS config_backup_sub_tasks;
DROP TABLE IF EXISTS config_backup_tasks;
