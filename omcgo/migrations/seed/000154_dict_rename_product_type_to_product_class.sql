-- ============================================================
-- 000154_dict_rename_product_type_to_product_class.sql
-- 字典 type 重命名：product_type → product_class，并用 devices.product_class
-- DISTINCT 真实值重置明细。
--
-- 背景：
--   · 历史字典 type='product_type'（见 seed/000004_mml_enhance.sql）使用数字
--     value（'1'..'6'）+ 中文 label，导致前端拿到 value 后无法直接用作
--     /mml/console/command-compatibility?product_class=... 的查询键，
--     需要前端按 label 反查，多了一次转换且易丢同步。
--   · 后端 devices.product_class 列才是设备真实上报的 productClass 字符串，
--     ProductRegistry 与命令兼容性检查 (R-8.5) 都按它路由。
--   · 字典 type 名也叫 product_class 后，前后端、SQL、UI 标签统一来源。
--
-- 改动：
--   1. sys_dictionaries.type='product_type' 改名为 'product_class'，
--      name 改为 '产品类型 (productClass)' 以便管理员一眼看出归属。
--   2. 清空旧明细，按 devices.product_class DISTINCT 重新插入：
--      label = value = product_class 字符串本身（不再用数字编号），
--      sort 按字母序确定，便于 UI 稳定排序。
--   3. 若 devices 表暂时为空（裸数据库），保留原 6 个标签作 fallback，
--      避免 UI 字典空导致 MML 控制台禁用。
-- ============================================================

-- +goose Up

-- 1. 重命名 type + name
UPDATE sys_dictionaries
   SET type        = 'product_class',
       name        = '产品类型 (productClass)',
       description = '设备 productClass — 来自 devices.product_class DISTINCT'
 WHERE type = 'product_type';

-- 若历史库根本没有 product_type 字典（fresh DB），首次创建之
INSERT INTO sys_dictionaries (name, type, status, description)
SELECT '产品类型 (productClass)', 'product_class', TRUE,
       '设备 productClass — 来自 devices.product_class DISTINCT'
 WHERE NOT EXISTS (
        SELECT 1 FROM sys_dictionaries WHERE type = 'product_class'
       );

-- 2. 清空旧明细（旧的 '1'..'6' 数字 value 已不再适用）
DELETE FROM sys_dictionary_details
 WHERE sys_dictionary_id = (SELECT id FROM sys_dictionaries WHERE type = 'product_class');

-- 3. 按 devices.product_class DISTINCT 重新插入。
--    label = value = product_class 字符串本身。
--    若 devices 为空则插入 6 个 fallback（与历史 seed/000004 行为对齐）。
-- +goose StatementBegin
DO $$
DECLARE
    dict_id BIGINT;
    has_devices BOOLEAN;
BEGIN
    SELECT id INTO dict_id FROM sys_dictionaries WHERE type = 'product_class';

    SELECT EXISTS (
        SELECT 1 FROM devices WHERE product_class IS NOT NULL AND product_class <> ''
    ) INTO has_devices;

    IF has_devices THEN
        INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
        SELECT pc, pc, ROW_NUMBER() OVER (ORDER BY pc), dict_id
          FROM (
                SELECT DISTINCT product_class AS pc
                  FROM devices
                 WHERE product_class IS NOT NULL AND product_class <> ''
               ) t;
    ELSE
        -- Fallback for empty DB — keep UI workable
        INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id) VALUES
            ('SmallCell-LTE', 'SmallCell-LTE', 1, dict_id),
            ('FAP-LTE-100',   'FAP-LTE-100',   2, dict_id),
            ('FAP-LTE-200',   'FAP-LTE-200',   3, dict_id),
            ('FAP-LTE-300',   'FAP-LTE-300',   4, dict_id),
            ('gNB-100',       'gNB-100',       5, dict_id),
            ('gNB-200',       'gNB-200',       6, dict_id);
    END IF;
END $$;
-- +goose StatementEnd


-- +goose Down

-- 还原 type/name
UPDATE sys_dictionaries
   SET type        = 'product_type',
       name        = '产品类型',
       description = '设备产品类型'
 WHERE type = 'product_class';

-- 还原明细到 seed/000004 的固定 6 项（数字 value）
DELETE FROM sys_dictionary_details
 WHERE sys_dictionary_id = (SELECT id FROM sys_dictionaries WHERE type = 'product_type');

INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT 'SmallCell-LTE', '1', 1, id FROM sys_dictionaries WHERE type = 'product_type'
ON CONFLICT DO NOTHING;
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT 'gNB-100', '2', 2, id FROM sys_dictionaries WHERE type = 'product_type'
ON CONFLICT DO NOTHING;
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT 'gNB-200', '3', 3, id FROM sys_dictionaries WHERE type = 'product_type'
ON CONFLICT DO NOTHING;
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT 'FAP-LTE-100', '4', 4, id FROM sys_dictionaries WHERE type = 'product_type'
ON CONFLICT DO NOTHING;
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT 'FAP-LTE-200', '5', 5, id FROM sys_dictionaries WHERE type = 'product_type'
ON CONFLICT DO NOTHING;
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT 'FAP-LTE-300', '6', 6, id FROM sys_dictionaries WHERE type = 'product_type'
ON CONFLICT DO NOTHING;
