-- +goose Up
-- KPI-EXPORT T1：KPI 数据导出任务表。
--
-- 一张表喂两个视图（设计 §4）：
--   - 任务管理 Tab：查全部
--   - 文件管理 Tab：查 status='succeeded' AND file_path <> ''（文件就绪）
--
-- source_type：dashboard（仪表盘曲线导出）/ adhoc（adhoc 任务结果导出）
-- status：pending → running → succeeded / failed
-- params(jsonb)：导出范围参数（dashboard / adhoc 各自结构，见设计 §4）
CREATE TABLE IF NOT EXISTS pm_kpi_export_tasks (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    task_name   TEXT         NOT NULL DEFAULT '',
    source_type TEXT         NOT NULL,
    params      JSONB        NOT NULL DEFAULT '{}'::jsonb,
    format      TEXT         NOT NULL DEFAULT 'csv',
    status      TEXT         NOT NULL DEFAULT 'pending',
    row_count   BIGINT       NOT NULL DEFAULT 0,
    bucket      TEXT         NOT NULL DEFAULT '',
    file_path   TEXT         NOT NULL DEFAULT '',
    file_size   BIGINT       NOT NULL DEFAULT 0,
    error       TEXT         NOT NULL DEFAULT '',
    create_user TEXT         NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    started_at  TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    expire_at   TIMESTAMPTZ,
    CONSTRAINT pm_kpi_export_tasks_source_type_chk
        CHECK (source_type IN ('dashboard', 'adhoc')),
    CONSTRAINT pm_kpi_export_tasks_status_chk
        CHECK (status IN ('pending', 'running', 'succeeded', 'failed'))
);

-- 任务管理 Tab：按创建时间倒序列表（默认排序）。
CREATE INDEX IF NOT EXISTS idx_pm_kpi_export_tasks_created_at
    ON pm_kpi_export_tasks (created_at DESC);

-- 文件管理 Tab：只筛已成功 + 文件就绪，按创建时间倒序。
CREATE INDEX IF NOT EXISTS idx_pm_kpi_export_tasks_files
    ON pm_kpi_export_tasks (created_at DESC)
    WHERE status = 'succeeded' AND file_path <> '';

-- +goose Down
DROP INDEX IF EXISTS idx_pm_kpi_export_tasks_files;
DROP INDEX IF EXISTS idx_pm_kpi_export_tasks_created_at;
DROP TABLE IF EXISTS pm_kpi_export_tasks;
