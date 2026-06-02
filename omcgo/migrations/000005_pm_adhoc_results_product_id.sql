-- +goose Up
-- T-0182-fix: product 维度聚合结果落库丢分组键。
-- pm_adhoc_aggregation_results 原仅有 device_oui/device_sn 作为分组标签，
-- product 维度聚合按 devices.product_id 分组，结果行需要带回 product_id 才能区分产品。
-- 加可空 product_id 列（device/aggregate_group 维度落 NULL，product 维度落聚合分组键）。
ALTER TABLE pm_adhoc_aggregation_results
    ADD COLUMN IF NOT EXISTS product_id UUID;

-- +goose Down
ALTER TABLE pm_adhoc_aggregation_results
    DROP COLUMN IF EXISTS product_id;
