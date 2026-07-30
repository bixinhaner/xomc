-- +goose Up
-- Collapse the original logical-target policies into one physical policy.
-- This deployment has no external storage; MinIO, PostgreSQL, queues and
-- monitoring data all consume the same host root filesystem. Keep the most
-- recently updated/enabled policy and re-point audit rows to it so historical
-- state remains queryable after the target model is normalized.
-- +goose StatementBegin
DO $$
DECLARE
    keeper uuid;
BEGIN
    SELECT id
      INTO keeper
      FROM storage_protection_policies
     ORDER BY enabled DESC, updated_at DESC, id
     LIMIT 1;

    IF keeper IS NOT NULL THEN
        UPDATE storage_protection_events
           SET policy_id = keeper
         WHERE policy_id <> keeper;

        DELETE FROM storage_protection_policies
         WHERE id <> keeper;

        UPDATE storage_protection_policies
           SET target_type = 'filesystem',
               target_id = 'root',
               write_scope = 'all',
               updated_at = now();

        UPDATE storage_protection_events
           SET target_type = 'filesystem',
               target_id = 'root',
               write_scope = 'all';
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE storage_protection_policies
    DROP CONSTRAINT IF EXISTS storage_protection_target_type_chk;

ALTER TABLE storage_protection_policies
    ADD CONSTRAINT storage_protection_target_type_chk CHECK (
        target_type = 'filesystem' AND target_id = 'root' AND write_scope = 'all'
    );

-- +goose Down
ALTER TABLE storage_protection_policies
    DROP CONSTRAINT IF EXISTS storage_protection_target_type_chk;

ALTER TABLE storage_protection_policies
    ADD CONSTRAINT storage_protection_target_type_chk CHECK (
        target_type IN ('filesystem', 'minio', 'database', 'redis', 'nats', 'monitoring', 'application')
    );
