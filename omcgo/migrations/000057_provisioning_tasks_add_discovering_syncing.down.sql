-- Revert to original status check constraint (without discovering/syncing).

ALTER TABLE provisioning_tasks
    DROP CONSTRAINT provisioning_tasks_status_check;

ALTER TABLE provisioning_tasks
    ADD CONSTRAINT provisioning_tasks_status_check
    CHECK (status IN ('discovered', 'identifying', 'matching', 'configuring', 'verifying', 'completed', 'failed'));
