-- +goose Up
-- ============================================================
-- 000097_ufte_task_types.sql
-- UFTE 统一文件传输引擎 — 任务类型表（内置模板）
-- ============================================================

CREATE TABLE IF NOT EXISTS ufte_task_types (
    type_code                 TEXT    NOT NULL,
    category                  TEXT    NOT NULL,
    category_label            TEXT    NOT NULL,
    display_name              TEXT    NOT NULL,
    description               TEXT    NOT NULL DEFAULT '',
    rpc_type                  TEXT    NOT NULL CHECK (rpc_type IN ('DOWNLOAD', 'UPLOAD', 'SET_PARAM_VALUES')),
    built_in                  BOOLEAN NOT NULL DEFAULT false,
    enabled                   BOOLEAN NOT NULL DEFAULT true,
    step_chain                JSONB   NOT NULL DEFAULT '[]',
    post_tc_event_code        TEXT    NOT NULL DEFAULT '',
    permission_code           TEXT    NOT NULL,
    platform_scope            JSONB   NOT NULL DEFAULT '[]',
    file_type                 TEXT    NOT NULL DEFAULT '',
    file_type_label           TEXT    NOT NULL DEFAULT '',
    file_type_editable        BOOLEAN NOT NULL DEFAULT true,
    url_template              TEXT    NOT NULL DEFAULT '',
    target_file_name_template TEXT    NOT NULL DEFAULT '',
    file_name_template        TEXT    NOT NULL DEFAULT '',
    file_size_field           TEXT    NOT NULL DEFAULT '',
    checksum_field            TEXT    NOT NULL DEFAULT '',
    raw_mode                  TEXT    NOT NULL DEFAULT '',
    delay_seconds             INT     NOT NULL DEFAULT 0,
    transport_path            TEXT    NOT NULL DEFAULT '',
    last_editor               TEXT    NOT NULL DEFAULT '',
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE ufte_task_types DROP CONSTRAINT IF EXISTS ufte_task_types_pkey;
ALTER TABLE ufte_task_types ADD CONSTRAINT ufte_task_types_pkey PRIMARY KEY (type_code);

CREATE INDEX IF NOT EXISTS idx_ufte_task_types_built_in ON ufte_task_types(built_in);
CREATE INDEX IF NOT EXISTS idx_ufte_task_types_category  ON ufte_task_types(category);
CREATE INDEX IF NOT EXISTS idx_ufte_task_types_enabled   ON ufte_task_types(enabled);

-- +goose Down
DROP TABLE IF EXISTS ufte_task_types CASCADE;
