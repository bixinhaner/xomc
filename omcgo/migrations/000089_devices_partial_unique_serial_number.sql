-- +goose Up
-- T-DIAG-DEV-PARTIAL-UNIQUE
-- 把 devices(serial_number, carrier) 全表唯一索引改为 partial unique
-- (WHERE deleted_at IS NULL)。
--
-- 背景：原索引 idx_devices_serial_number（ON ONLY 父表声明）+ 三个分区表各自
-- 独立的 devices_{cmcc,ctcc,cucc}_serial_number_carrier_idx 形成全表硬唯一约束。
-- 软删 device 后 deleted_at 不为 NULL 但 (serial_number, carrier) 仍占用唯一槽位
-- → 同 SN 设备无法重新注册（auto-register 路径报 SQLSTATE 23505）。
--
-- 改为 partial unique，软删行不再占唯一槽，同 SN 重新注册可走通。

-- DROP 旧索引（4 个：父 + 3 个分区，独立创建不级联）
-- +goose StatementBegin
DO $$
BEGIN
    EXECUTE 'DROP INDEX IF EXISTS devices_cmcc_serial_number_carrier_idx';
    EXECUTE 'DROP INDEX IF EXISTS devices_ctcc_serial_number_carrier_idx';
    EXECUTE 'DROP INDEX IF EXISTS devices_cucc_serial_number_carrier_idx';
    EXECUTE 'DROP INDEX IF EXISTS idx_devices_serial_number';
END $$;
-- +goose StatementEnd

-- 重建为 partitioned partial unique（不带 ONLY，PG 会自动级联到每个分区）
CREATE UNIQUE INDEX idx_devices_serial_number
ON devices(serial_number, carrier)
WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_devices_serial_number;

-- 还原为原状（ONLY 父索引 + 3 个独立分区索引）
CREATE UNIQUE INDEX idx_devices_serial_number ON ONLY devices(serial_number, carrier);
CREATE UNIQUE INDEX devices_cmcc_serial_number_carrier_idx ON devices_cmcc(serial_number, carrier);
CREATE UNIQUE INDEX devices_ctcc_serial_number_carrier_idx ON devices_ctcc(serial_number, carrier);
CREATE UNIQUE INDEX devices_cucc_serial_number_carrier_idx ON devices_cucc(serial_number, carrier);
