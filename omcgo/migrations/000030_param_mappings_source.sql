-- +goose Up
-- T-PMSRC: param_mappings 行级来源标记(builtin / custom)。
-- builtin = dictloader 从 XML 加载;custom = 管理员经 UI 新增,或对 builtin 行编辑后转化的覆盖项。
-- 重载 XML 时只清 source='builtin' 行(见 loader.go),custom 行保留,从而"重新加载不覆盖自定义数据"。
-- 既有行(均来自 XML)经 DEFAULT 回填为 builtin。
-- +goose StatementBegin
ALTER TABLE param_mappings ADD COLUMN IF NOT EXISTS source varchar(16) NOT NULL DEFAULT 'builtin';
-- +goose StatementEnd
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'param_mappings_source_check') THEN
        ALTER TABLE param_mappings
            ADD CONSTRAINT param_mappings_source_check CHECK (source IN ('builtin', 'custom'));
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE param_mappings DROP CONSTRAINT IF EXISTS param_mappings_source_check;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE param_mappings DROP COLUMN IF EXISTS source;
-- +goose StatementEnd
