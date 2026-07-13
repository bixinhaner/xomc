-- +goose Up
-- +goose StatementBegin
ALTER TABLE public.device_groups
    ADD COLUMN IF NOT EXISTS source_group_id uuid;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'device_groups_source_group_id_fkey') THEN
        ALTER TABLE public.device_groups
            ADD CONSTRAINT device_groups_source_group_id_fkey
            FOREIGN KEY (source_group_id) REFERENCES public.device_groups(id) ON DELETE SET NULL;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'device_groups_rule_source_not_self') THEN
        ALTER TABLE public.device_groups
            ADD CONSTRAINT device_groups_rule_source_not_self
            CHECK (source_group_id IS NULL OR source_group_id <> id);
    END IF;
END
$$;

CREATE INDEX IF NOT EXISTS idx_device_groups_source_group_id
    ON public.device_groups(source_group_id) WHERE source_group_id IS NOT NULL;

COMMENT ON COLUMN public.device_groups.source_group_id IS
    'Source L2 group for automatic matching; NULL disables legacy rules until configured.';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS public.idx_device_groups_source_group_id;
ALTER TABLE public.device_groups
    DROP CONSTRAINT IF EXISTS device_groups_rule_source_not_self,
    DROP CONSTRAINT IF EXISTS device_groups_source_group_id_fkey,
    DROP COLUMN IF EXISTS source_group_id;
-- +goose StatementEnd
