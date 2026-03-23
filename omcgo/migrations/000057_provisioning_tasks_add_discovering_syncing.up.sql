-- Add 'discovering' and 'syncing' states to provisioning_tasks status check constraint.
-- These states support auto-discovery (Path C) and auto-sync (Path B) workflows.

ALTER TABLE provisioning_tasks
    DROP CONSTRAINT provisioning_tasks_status_check;

ALTER TABLE provisioning_tasks
    ADD CONSTRAINT provisioning_tasks_status_check
    CHECK (status IN ('discovered', 'identifying', 'matching', 'configuring', 'verifying',
                      'discovering', 'syncing', 'completed', 'failed'));
