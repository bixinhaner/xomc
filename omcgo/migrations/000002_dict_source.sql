-- T-0182 数据字典数据源绑定
-- PRD: docs/project/prd/F06-data-dictionary-source.md §13 A.3
--
-- 给 sys_dictionaries 加 8 列保存数据源绑定 + 同步元数据;
-- 给 sys_dictionary_details 加 origin 列区分手工项与自动同步项。
-- 软约束:source 字典只能含 origin='manual' 或 origin='auto';
-- 手工字典只能含 origin='manual'。DB 不加 CHECK,由 service 层守门(兼容历史数据)。

-- +goose Up
-- +goose StatementBegin
ALTER TABLE public.sys_dictionaries
    ADD COLUMN IF NOT EXISTS source_table          VARCHAR(64),
    ADD COLUMN IF NOT EXISTS source_label_field    VARCHAR(64),
    ADD COLUMN IF NOT EXISTS source_value_field    VARCHAR(64),
    ADD COLUMN IF NOT EXISTS source_filter         JSONB,
    ADD COLUMN IF NOT EXISTS last_refresh_at       TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_refresh_status   VARCHAR(16),
    ADD COLUMN IF NOT EXISTS last_refresh_error    TEXT,
    ADD COLUMN IF NOT EXISTS last_refresh_count    INTEGER;

COMMENT ON COLUMN public.sys_dictionaries.source_table       IS '数据源表(白名单业务名);NULL=手工字典';
COMMENT ON COLUMN public.sys_dictionaries.source_label_field IS '源表中作为字典项 label 的字段(白名单内)';
COMMENT ON COLUMN public.sys_dictionaries.source_value_field IS '源表中作为字典项 value 的字段(白名单内,可与 label 同字段)';
COMMENT ON COLUMN public.sys_dictionaries.source_filter      IS 'v1 预留,JSONB 过滤条件;v1 不读不写';
COMMENT ON COLUMN public.sys_dictionaries.last_refresh_at    IS '上次同步成功/失败的时间';
COMMENT ON COLUMN public.sys_dictionaries.last_refresh_status IS '上次同步状态 ok|failed|running|timeout';
COMMENT ON COLUMN public.sys_dictionaries.last_refresh_error IS '上次同步失败摘要(<=500 chars)';
COMMENT ON COLUMN public.sys_dictionaries.last_refresh_count IS '上次同步完成后 origin=auto 项总数';

ALTER TABLE public.sys_dictionary_details
    ADD COLUMN IF NOT EXISTS origin VARCHAR(16) NOT NULL DEFAULT 'manual';

COMMENT ON COLUMN public.sys_dictionary_details.origin
    IS '来源 manual=手工 / auto=数据源同步;同步任务只动 auto 行,manual 行保留';

CREATE INDEX IF NOT EXISTS idx_sys_dict_source_table
    ON public.sys_dictionaries (source_table)
    WHERE source_table IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_sys_dict_detail_dictid_origin
    ON public.sys_dictionary_details (sys_dictionary_id, origin);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS public.idx_sys_dict_detail_dictid_origin;
DROP INDEX IF EXISTS public.idx_sys_dict_source_table;

ALTER TABLE public.sys_dictionary_details DROP COLUMN IF EXISTS origin;

ALTER TABLE public.sys_dictionaries
    DROP COLUMN IF EXISTS last_refresh_count,
    DROP COLUMN IF EXISTS last_refresh_error,
    DROP COLUMN IF EXISTS last_refresh_status,
    DROP COLUMN IF EXISTS last_refresh_at,
    DROP COLUMN IF EXISTS source_filter,
    DROP COLUMN IF EXISTS source_value_field,
    DROP COLUMN IF EXISTS source_label_field,
    DROP COLUMN IF EXISTS source_table;
-- +goose StatementEnd
