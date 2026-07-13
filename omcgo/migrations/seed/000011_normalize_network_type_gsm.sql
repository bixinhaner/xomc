-- +goose Up
-- #50 / #58: network_type 的机器值统一为 lte/nr/gsm；GSM 仅作为展示文本。
WITH network_type AS (
    SELECT id FROM sys_dictionaries WHERE type = 'network_type' AND deleted_at IS NULL
)
UPDATE sys_dictionary_details AS legacy
SET deleted_at = NOW(), updated_at = NOW()
FROM network_type
WHERE legacy.sys_dictionary_id = network_type.id AND legacy.parent_id IS NULL
  AND legacy.value = 'GSM' AND legacy.deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM sys_dictionary_details AS canonical
              WHERE canonical.sys_dictionary_id = network_type.id AND canonical.parent_id IS NULL
                AND canonical.value = 'gsm' AND canonical.deleted_at IS NULL);

WITH network_type AS (
    SELECT id FROM sys_dictionaries WHERE type = 'network_type' AND deleted_at IS NULL
)
UPDATE sys_dictionary_details AS legacy
SET value = 'gsm', updated_at = NOW()
FROM network_type
WHERE legacy.sys_dictionary_id = network_type.id AND legacy.parent_id IS NULL
  AND legacy.value = 'GSM' AND legacy.deleted_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM sys_dictionary_details AS canonical
                  WHERE canonical.sys_dictionary_id = network_type.id AND canonical.parent_id IS NULL
                    AND canonical.value = 'gsm' AND canonical.deleted_at IS NULL);

INSERT INTO sys_dictionary_details (label, value, extend, status, sort, sys_dictionary_id, origin, label_i18n)
SELECT 'GSM', 'gsm', '', true, 3, dictionary.id, 'manual',
       '{"en-US": "GSM", "zh-CN": "GSM"}'::jsonb
FROM sys_dictionaries AS dictionary
WHERE dictionary.type = 'network_type' AND dictionary.deleted_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM sys_dictionary_details AS detail
                  WHERE detail.sys_dictionary_id = dictionary.id AND detail.parent_id IS NULL
                    AND detail.value = 'gsm' AND detail.deleted_at IS NULL);

-- +goose Down
-- 数据规范化不可安全逆转，故 no-op。
