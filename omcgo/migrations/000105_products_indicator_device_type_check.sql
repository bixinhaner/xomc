-- +goose Up
-- 约束 products.indicator_device_type 只能为小写 enb/gsm/gnb。
-- XML loader 与 Create/Update handler 均按小写写入；本 CHECK 防未来旁路写入大写值
-- 导致 formulaTableByDeviceType / indicatorTableByDeviceType 的 switch case 查表失败。
-- 当前存量数据全部为小写，无需 UPDATE 修存量。

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_products_indicator_device_type'
    ) THEN
        ALTER TABLE products
            ADD CONSTRAINT chk_products_indicator_device_type
            CHECK (indicator_device_type IN ('enb', 'gsm', 'gnb'));
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
ALTER TABLE products DROP CONSTRAINT IF EXISTS chk_products_indicator_device_type;
