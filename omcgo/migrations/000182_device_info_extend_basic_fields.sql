-- +goose Up
-- Phase 2 of docs/design/device-detail-basic-fields-from-parameters-20260525.md
--
-- 设备详情「基础信息」字段补齐：在 device_info 表新增 11 列承接 CPE 已经上报
-- 但当前 schema 未承接的字段。
--
-- 所有列 nullable + 不写默认值 — 旧设备的列保持 NULL，InfoSyncer 在下次
-- SyncFromParameters 调用时按 carrier-specific mapping 表自动回填。
--
-- 字段对应的 TR-069 path 由各 carrier adapter (cmcc/ctcc/cucc) 的
-- GetInfoParamMapping 维护，本迁移仅扩展 schema，不触碰映射代码。

ALTER TABLE device_info
    ADD COLUMN IF NOT EXISTS tac                  VARCHAR(16),
    ADD COLUMN IF NOT EXISTS band                 VARCHAR(16),
    ADD COLUMN IF NOT EXISTS ul_earfcn            VARCHAR(32),
    ADD COLUMN IF NOT EXISTS subframe_assignment  VARCHAR(8),
    ADD COLUMN IF NOT EXISTS special_subframe     VARCHAR(8),
    ADD COLUMN IF NOT EXISTS root_index           VARCHAR(16),
    ADD COLUMN IF NOT EXISTS gps_satellites       INTEGER,
    ADD COLUMN IF NOT EXISTS gps_height           NUMERIC(10,2),
    ADD COLUMN IF NOT EXISTS lock_status          VARCHAR(16),
    ADD COLUMN IF NOT EXISTS enb_id               VARCHAR(32),
    ADD COLUMN IF NOT EXISTS network_model        VARCHAR(16);

COMMENT ON COLUMN device_info.tac IS 'TAC (Tracking Area Code), TR-069 EPC.TAC';
COMMENT ON COLUMN device_info.band IS 'LTE Band, TR-069 RAN.RF.FreqBandIndicator';
COMMENT ON COLUMN device_info.ul_earfcn IS 'Uplink EARFCN, TR-069 RAN.RF.EARFCNUL';
COMMENT ON COLUMN device_info.subframe_assignment IS 'TDD frame, RAN.PHY.TDDFrame.SubFrameAssignment';
COMMENT ON COLUMN device_info.special_subframe IS 'TDD special subframe pattern';
COMMENT ON COLUMN device_info.root_index IS 'PRACH ZeroCorrelationZoneConfig';
COMMENT ON COLUMN device_info.gps_satellites IS 'GPS satellite count, FAP.GPS.NumberOfSatellites';
COMMENT ON COLUMN device_info.gps_height IS 'GPS height (meters), FAP.GPS.altidute';
COMMENT ON COLUMN device_info.lock_status IS 'Cell AdminState (true=unlocked / false=locked)';
COMMENT ON COLUMN device_info.enb_id IS 'eNodeB ID, derived from ECI >> 8 (LTE 28-bit ECI = 20-bit eNB + 8-bit Cell)';
COMMENT ON COLUMN device_info.network_model IS 'TDD or FDD, derived from frame structure';

-- +goose Down
ALTER TABLE device_info
    DROP COLUMN IF EXISTS tac,
    DROP COLUMN IF EXISTS band,
    DROP COLUMN IF EXISTS ul_earfcn,
    DROP COLUMN IF EXISTS subframe_assignment,
    DROP COLUMN IF EXISTS special_subframe,
    DROP COLUMN IF EXISTS root_index,
    DROP COLUMN IF EXISTS gps_satellites,
    DROP COLUMN IF EXISTS gps_height,
    DROP COLUMN IF EXISTS lock_status,
    DROP COLUMN IF EXISTS enb_id,
    DROP COLUMN IF EXISTS network_model;
