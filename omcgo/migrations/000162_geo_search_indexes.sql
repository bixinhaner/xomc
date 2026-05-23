-- +goose Up
-- 启用 pg_trgm 扩展（支持模糊搜索索引）
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_trgm') THEN
        CREATE EXTENSION pg_trgm;
    END IF;
END $$;
-- +goose StatementEnd

-- devices 表：B-tree 索引（精确匹配）
-- 注意：inet 类型不支持空字符串比较，只检查 IS NOT NULL
CREATE INDEX IF NOT EXISTS idx_devices_ip_address ON devices(ip_address)
    WHERE ip_address IS NOT NULL;

-- device_info 表：B-tree 索引（精确匹配）
CREATE INDEX IF NOT EXISTS idx_device_info_mac ON device_info(mac)
    WHERE mac IS NOT NULL AND mac != '';
CREATE INDEX IF NOT EXISTS idx_device_info_pci ON device_info(pci)
    WHERE pci IS NOT NULL AND pci != '';
CREATE INDEX IF NOT EXISTS idx_device_info_device_name ON device_info(device_name)
    WHERE device_name IS NOT NULL AND device_name != '';

-- device_info 表：GIN 索引（模糊搜索，ILIKE 优化）
-- 注意：pg_trgm 索引对 ILIKE 有效，但需要文本非空
CREATE INDEX IF NOT EXISTS idx_device_info_mac_trgm ON device_info USING gin(mac gin_trgm_ops)
    WHERE mac IS NOT NULL AND mac != '';
CREATE INDEX IF NOT EXISTS idx_device_info_pci_trgm ON device_info USING gin(pci gin_trgm_ops)
    WHERE pci IS NOT NULL AND pci != '';
CREATE INDEX IF NOT EXISTS idx_device_info_device_name_trgm ON device_info USING gin(device_name gin_trgm_ops)
    WHERE device_name IS NOT NULL AND device_name != '';

-- +goose Down
DROP INDEX IF EXISTS idx_devices_ip_address;
DROP INDEX IF EXISTS idx_device_info_mac;
DROP INDEX IF EXISTS idx_device_info_pci;
DROP INDEX IF EXISTS idx_device_info_device_name;
DROP INDEX IF EXISTS idx_device_info_mac_trgm;
DROP INDEX IF EXISTS idx_device_info_pci_trgm;
DROP INDEX IF EXISTS idx_device_info_device_name_trgm;
