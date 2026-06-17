-- 000004: Add admin_state and ipsec_addr columns to device_info
--
-- Both columns are populated by InfoSyncer.SyncFromParameters via
-- universalInformMapping:
--   - admin_state ← Device.Services.FAPService.1.FAPControl.NR.RAN.Common.AdminState
--     (NR-only; values "1"=Locked, "2"=Unlocked, "3"=ShuttingDown — front-end
--      device list "Admin State" column renders via fmtStatus map; matches
--      legacy gNB JSP semantics. LTE devices keep this column NULL and use
--      lock_status instead.)
--   - ipsec_addr ← Device.DeviceInfo.SERVING_UNIT1_IPSEC_Address
--     (carrier-agnostic; serving unit 1 IPSec tunnel address. BaiBNQ/BLQ/MLN
--      param-mappings already declare the standardPath via X_COM_SERVING_UNIT1_IPSEC_Address
--      private path. Empty / "0.0.0.0" means tunnel not yet established.)

-- +goose Up
ALTER TABLE public.device_info
    ADD COLUMN IF NOT EXISTS admin_state character varying(16),
    ADD COLUMN IF NOT EXISTS ipsec_addr  character varying(64);

COMMENT ON COLUMN public.device_info.admin_state IS
    'NR FAPControl AdminState ("1"=Locked, "2"=Unlocked, "3"=ShuttingDown), source Device.Services.FAPService.1.FAPControl.NR.RAN.Common.AdminState. NULL for LTE devices (use lock_status instead).';

COMMENT ON COLUMN public.device_info.ipsec_addr IS
    'IPSec serving unit 1 tunnel address, source Device.DeviceInfo.SERVING_UNIT1_IPSEC_Address. "0.0.0.0" means tunnel not established.';

-- +goose Down
ALTER TABLE public.device_info DROP COLUMN IF EXISTS admin_state;
ALTER TABLE public.device_info DROP COLUMN IF EXISTS ipsec_addr;
