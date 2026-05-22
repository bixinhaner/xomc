-- +goose Up
-- ============================================================
-- 000159_add_devices_other_partition.up.sql
-- 新增 devices_other 分区（支持非中国三大运营商设备）
-- 功能域: F06 拓扑管理
--
-- 说明：
--   - 新增 'other' 分区支持非 cmcc/ctcc/cucc 运营商的设备
--   - 不修改现有分区结构，不影响现有查询逻辑
--   - 使用 carrier='other' 隔离数据
-- ============================================================

-- 创建 other 运营商分区
CREATE TABLE devices_other PARTITION OF devices
FOR VALUES IN ('other');

-- 添加注释
COMMENT ON TABLE devices_other IS '设备表 - 其他运营商分区（非 cmcc/ctcc/cucc，如赞比亚 ZED 等国际运营商）';

-- 为 other 分区创建索引（与其他分区保持一致）
CREATE INDEX idx_devices_other_serial_number ON devices_other (serial_number);
CREATE INDEX idx_devices_other_carrier_status ON devices_other (carrier, status);
CREATE INDEX idx_devices_other_oui ON devices_other (oui);
CREATE INDEX idx_devices_other_carrier_tech ON devices_other (carrier, technology);
CREATE INDEX idx_devices_other_status ON devices_other (status);
CREATE INDEX idx_devices_other_last_inform ON devices_other (last_inform_at);
CREATE INDEX idx_devices_other_deleted_at ON devices_other (deleted_at) WHERE deleted_at IS NOT NULL;

-- +goose Down
-- ============================================================
-- 回滚迁移
-- ============================================================

-- 删除索引
DROP INDEX IF EXISTS idx_devices_other_deleted_at;
DROP INDEX IF EXISTS idx_devices_other_last_inform;
DROP INDEX IF EXISTS idx_devices_other_status;
DROP INDEX IF EXISTS idx_devices_other_carrier_tech;
DROP INDEX IF EXISTS idx_devices_other_oui;
DROP INDEX IF EXISTS idx_devices_other_carrier_status;
DROP INDEX IF EXISTS idx_devices_other_serial_number;

-- 删除分区表
DROP TABLE IF EXISTS devices_other;
