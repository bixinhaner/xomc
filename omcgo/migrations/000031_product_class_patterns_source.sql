-- +goose Up
-- product_class_patterns 行级来源标记(builtin / custom)。
-- builtin = dictloader 从 products.xml 加载;custom = 管理员经 UI 新增的自定义路由正则。
-- 重载 XML 时只清 source='builtin' 行(见 internal/product/loader.go),custom 行保留,
-- 从而「重新加载不覆盖 UI 自定义正则」。既有行(均来自 XML)经 DEFAULT 回填为 builtin。
-- 对标 migrations/000030_param_mappings_source.sql 同款行级 source 范式。
-- +goose StatementBegin
ALTER TABLE product_class_patterns ADD COLUMN IF NOT EXISTS source varchar(16) NOT NULL DEFAULT 'builtin';
-- +goose StatementEnd
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'product_class_patterns_source_check') THEN
        ALTER TABLE product_class_patterns
            ADD CONSTRAINT product_class_patterns_source_check CHECK (source IN ('builtin', 'custom'));
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE product_class_patterns DROP CONSTRAINT IF EXISTS product_class_patterns_source_check;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE product_class_patterns DROP COLUMN IF EXISTS source;
-- +goose StatementEnd
