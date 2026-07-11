-- +goose Up
-- +goose StatementBegin
-- Repair schema drift for databases that applied older versions of baseline or
-- versioned migrations before these columns were folded into them.
ALTER TABLE public.device_groups
    ADD COLUMN IF NOT EXISTS source_group_id uuid;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_constraint
         WHERE conname = 'device_groups_source_group_id_fkey'
           AND conrelid = 'public.device_groups'::regclass
    ) THEN
        ALTER TABLE public.device_groups
            ADD CONSTRAINT device_groups_source_group_id_fkey
            FOREIGN KEY (source_group_id) REFERENCES public.device_groups(id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_constraint
         WHERE conname = 'device_groups_rule_source_not_self'
           AND conrelid = 'public.device_groups'::regclass
    ) THEN
        ALTER TABLE public.device_groups
            ADD CONSTRAINT device_groups_rule_source_not_self
            CHECK (source_group_id IS NULL OR source_group_id <> id);
    END IF;
END
$$;

CREATE INDEX IF NOT EXISTS idx_device_groups_source_group_id
    ON public.device_groups(source_group_id)
    WHERE source_group_id IS NOT NULL;

COMMENT ON COLUMN public.device_groups.source_group_id IS
    'Source L2 group for automatic matching; NULL disables legacy rules until configured.';

ALTER TABLE public.device_info ADD COLUMN IF NOT EXISTS bsc_select varchar(8);
ALTER TABLE public.device_info ADD COLUMN IF NOT EXISTS oml_remote_ip varchar(45);
ALTER TABLE public.device_info ADD COLUMN IF NOT EXISTS oml_remote_ip_bak varchar(45);
ALTER TABLE public.device_info ADD COLUMN IF NOT EXISTS ipa_unit_id varchar(32);

COMMENT ON COLUMN public.device_info.bsc_select IS
    'GSM BSC primary/backup role from DeviceGSM.BscSelect ("0"=Master, "1"=Backup). BTS-only; LTE/NR stay NULL.';
COMMENT ON COLUMN public.device_info.oml_remote_ip IS
    'Abis OML primary BSC IP from DeviceGSM.OmlRemoteIp. BTS-only.';
COMMENT ON COLUMN public.device_info.oml_remote_ip_bak IS
    'Abis OML backup BSC IP from DeviceGSM.OmlRemoteIpBak. BTS-only.';
COMMENT ON COLUMN public.device_info.ipa_unit_id IS
    'IPA unit ID from DeviceGSM.IpaUnitId, for example "9227-2". BTS-only.';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Non-destructive by design: these columns are part of the current canonical
-- schema and may already exist on fresh databases via 000001/000015.
SELECT 1;
-- +goose StatementEnd
