-- +goose Up
-- T-DIAG-DEV-PARTIAL-UNIQUE
-- 把 devices(serial_number, carrier) 全表唯一索引改为 partial unique
-- (WHERE deleted_at IS NULL)。
--
-- 背景：原索引 idx_devices_serial_number + 三个分区表各自
-- 的 devices_{cmcc,ctcc,cucc}_serial_number_carrier_idx 形成全表硬唯一约束。
-- 软删 device 后 deleted_at 不为 NULL 但 (serial_number, carrier) 仍占用唯一槽位
-- → 同 SN 设备无法重新注册（auto-register 路径报 SQLSTATE 23505）。
--
-- 改为 partial unique，软删行不再占唯一槽，同 SN 重新注册可走通。
--
-- 修订（2026-05-13）：原 DO 块按"子 → 父"顺序 drop，触发
-- SQLSTATE 2BP01 (dependent_objects_still_exist)：
--   "cannot drop index devices_cmcc_serial_number_carrier_idx
--    because index idx_devices_serial_number requires it"
-- 实际数据库中 idx_devices_serial_number 是 partitioned index（不是 ON ONLY），
-- 三个子分区索引 attach 在其下，必须先 drop 父（PG 级联子）。
-- 改为两段 DO：先 drop 父级联子；再单独 drop 子作为兜底（处理父已 DETACH
-- 或独立创建的边界状态）。两段都用 IF EXISTS，幂等。

-- DROP 父索引（partitioned index 会级联清理子分区索引）
-- +goose StatementBegin
DO $$
BEGIN
    EXECUTE 'DROP INDEX IF EXISTS idx_devices_serial_number';
END $$;
-- +goose StatementEnd

-- 兜底：清理可能残留的独立子分区索引（父非 partitioned 或父已 DETACH 的边界状态）
-- +goose StatementBegin
DO $$
BEGIN
    EXECUTE 'DROP INDEX IF EXISTS devices_cmcc_serial_number_carrier_idx';
    EXECUTE 'DROP INDEX IF EXISTS devices_ctcc_serial_number_carrier_idx';
    EXECUTE 'DROP INDEX IF EXISTS devices_cucc_serial_number_carrier_idx';
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
