-- +goose Up
-- Path B/C states (`discovering`, `syncing`) are used by ProvisioningEngine
-- since T-0098 but the original CHECK constraint only allowed Path A states.
-- Without this fix, `transitionTask(StateSyncing|StateDiscovering)` fails with
-- SQLSTATE 23514 and provisioning never reaches discovery / auto-sync.
ALTER TABLE provisioning_tasks DROP CONSTRAINT IF EXISTS provisioning_tasks_status_check;
ALTER TABLE provisioning_tasks ADD CONSTRAINT provisioning_tasks_status_check
    CHECK (status IN (
        'discovered', 'identifying', 'matching',
        'configuring', 'verifying',
        'discovering', 'syncing',
        'completed', 'failed'
    ));

-- +goose Down
ALTER TABLE provisioning_tasks DROP CONSTRAINT IF EXISTS provisioning_tasks_status_check;
ALTER TABLE provisioning_tasks ADD CONSTRAINT provisioning_tasks_status_check
    CHECK (status IN (
        'discovered', 'identifying', 'matching',
        'configuring', 'verifying',
        'completed', 'failed'
    ));
