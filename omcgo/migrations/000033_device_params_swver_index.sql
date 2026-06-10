-- +goose Up
-- #10：device_parameters 按 SoftwareVersion 过滤（ListDevices filter.SoftwareVersion）走子查询：
--   SELECT device_id FROM device_parameters
--   WHERE parameter_path = 'Device.DeviceInfo.SoftwareVersion'
--     AND parameter_value IN (...)
-- 既有索引均以 device_id 打头（idx_device_params_device / idx_device_params_path_prefix），
-- 而该子查询无 device_id 谓词 → 32 个 HASH 分区全表扫描。
--
-- 这里加“仅覆盖 SoftwareVersion 行”的部分覆盖索引：
--   - 部分索引（WHERE parameter_path = 'Device.DeviceInfo.SoftwareVersion'）：每设备仅 1 行，
--     索引体积极小；同时**避开**对 parameter_value(text) 全量建索引时，长值（证书 / 大配置块）
--     超过 btree 单行 2704B 上限导致写入失败的风险。
--   - 覆盖列 (parameter_value, device_id)：支持 parameter_value IN (...) 等值谓词，
--     并可 index-only scan 直接返回 device_id，无需回堆。
-- 分区父表上 CREATE INDEX 会自动下发到全部 32 个 HASH 分区；用普通（非 CONCURRENTLY）建索引
-- 以便在 goose 事务内执行，大表首建建议放维护窗口。
CREATE INDEX IF NOT EXISTS idx_device_params_swver
    ON public.device_parameters (parameter_value, device_id)
    WHERE parameter_path = 'Device.DeviceInfo.SoftwareVersion';

-- +goose Down
DROP INDEX IF EXISTS idx_device_params_swver;
