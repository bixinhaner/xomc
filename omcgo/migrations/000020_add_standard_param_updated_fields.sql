-- +goose Up
ALTER TABLE public.standard_params
    ADD COLUMN IF NOT EXISTS updated_fields text[] DEFAULT '{}'::text[] NOT NULL;

COMMENT ON COLUMN public.standard_params.updated_fields IS
    '最近一次人工编辑发生变化的字段名；新增及尚未人工编辑的记录为空数组';

-- +goose Down
ALTER TABLE public.standard_params
    DROP COLUMN IF EXISTS updated_fields;
