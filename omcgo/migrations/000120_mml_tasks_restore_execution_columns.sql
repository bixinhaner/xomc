-- +goose Up
-- 补回 mml_tasks 的执行调度 / 重试策略 / 统计 / 聚合结果列。
--
-- 背景（修复代码↔schema 不一致）：
--   · internal/mml 的 taskColumns（pg_repository.go）共 29 列，对应 Go 模型 MMLTask；
--   · 但迁移只创建了 14 列：000007 建表 10 列 + 000020 executor + 000032
--     scheduled_at/next_trigger_at/parent_task_id；
--   · 000090_mml_schema_rebuild 注释称单次执行字段「归 mml_tasks」，但实际从未把
--     对应的 ADD COLUMN 写进迁移 —— 以下 15 列在任何迁移中都不存在。
--   · 结果：SELECT taskColumns FROM mml_tasks 报 column does not exist，
--     /mml/task-records 列表、新建、删除（DeleteTask 先 GetByID）全部失败。
--
-- 列定义按 Go 模型 MMLTask 字段类型确定：指针 *time.Time → 可空 TIMESTAMPTZ；
-- 值类型 string/int/bool → NOT NULL + 合理默认值；*TaskResult → 可空 JSONB。
-- ADD COLUMN IF NOT EXISTS 保证幂等、可重复执行。
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS execute_type          VARCHAR(20)  NOT NULL DEFAULT 'immediate';
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS period_start          TIMESTAMPTZ;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS period_end            TIMESTAMPTZ;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS period_time           VARCHAR(20)  NOT NULL DEFAULT '';
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS offline_retry         BOOLEAN      NOT NULL DEFAULT false;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS offline_retry_wait    INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS failed_retry          BOOLEAN      NOT NULL DEFAULT false;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS failed_retry_count    INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS failed_retry_interval INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS started_at            TIMESTAMPTZ;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS finished_at           TIMESTAMPTZ;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS total_devices         INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS success_count         INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS failed_count          INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE mml_tasks ADD COLUMN IF NOT EXISTS result                JSONB;

-- +goose Down
ALTER TABLE mml_tasks DROP COLUMN IF EXISTS result;
ALTER TABLE mml_tasks DROP COLUMN IF EXISTS failed_count;
ALTER TABLE mml_tasks DROP COLUMN IF EXISTS success_count;
ALTER TABLE mml_tasks DROP COLUMN IF EXISTS total_devices;
ALTER TABLE mml_tasks DROP COLUMN IF EXISTS finished_at;
ALTER TABLE mml_tasks DROP COLUMN IF EXISTS started_at;
ALTER TABLE mml_tasks DROP COLUMN IF EXISTS failed_retry_interval;
ALTER TABLE mml_tasks DROP COLUMN IF EXISTS failed_retry_count;
ALTER TABLE mml_tasks DROP COLUMN IF EXISTS failed_retry;
ALTER TABLE mml_tasks DROP COLUMN IF EXISTS offline_retry_wait;
ALTER TABLE mml_tasks DROP COLUMN IF EXISTS offline_retry;
ALTER TABLE mml_tasks DROP COLUMN IF EXISTS period_time;
ALTER TABLE mml_tasks DROP COLUMN IF EXISTS period_end;
ALTER TABLE mml_tasks DROP COLUMN IF EXISTS period_start;
ALTER TABLE mml_tasks DROP COLUMN IF EXISTS execute_type;
