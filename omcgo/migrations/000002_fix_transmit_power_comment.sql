-- +goose Up
-- #362：修正 device_info.transmit_power 列 COMMENT。
-- 原 000001 baseline 注释把该列写成「对应 TR-181 FAPService.{i}.Capabilities.MaxTxPower」，
-- 但 MaxTxPower 是硬件最大能力上限(READ_ONLY)，而前端"发射功率"列与 LMT 口径是
-- 参考信号功率 ReferenceSignalPower(RW，小区实际工作功率)。已在 device_info_sync.go
-- 删除 universalInformMapping 对 transmit_power 的 MaxTxPower 覆盖，使 cmcc/ctcc adapter
-- 的 ReferenceSignalPower→transmit_power 成为唯一权威来源。此处同步修正列注释口径。
COMMENT ON COLUMN public.device_info.transmit_power IS '发射功率（dBm），口径=参考信号功率（与 LMT 一致），来源 TR-181 FAPService.{i}.CellConfig.LTE.RAN.RF.ReferenceSignalPower（经 cmcc/ctcc carrier adapter 映射）。注：非硬件最大能力上限 MaxTxPower。';

-- +goose Down
COMMENT ON COLUMN public.device_info.transmit_power IS '发射功率（dBm）,对应 TR-181 FAPService.{i}.Capabilities.MaxTxPower。';
