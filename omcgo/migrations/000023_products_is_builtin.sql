-- +goose Up
-- 产品装配件加「内置」标记：由 products.xml 装配加载的产品 is_builtin=true（不可删除），
-- UI 新建的自定义产品默认 false（可删除）。
-- 新增列带 DEFAULT false 且 NOT NULL，既有行/既有 seed 自动满足（§5.5.11 铁律 1）。
-- 存量内置产品在下次 app 启动 loader UPSERT 时自动回填 is_builtin=true（self-healing）。
ALTER TABLE products ADD COLUMN IF NOT EXISTS is_builtin boolean NOT NULL DEFAULT false;
COMMENT ON COLUMN products.is_builtin IS 'true=products.xml 装配加载的内置产品（禁止删除）；false=UI 新建的自定义产品。loader UPSERT 置 true，Create 默认 false。';

-- +goose Down
ALTER TABLE products DROP COLUMN IF EXISTS is_builtin;
