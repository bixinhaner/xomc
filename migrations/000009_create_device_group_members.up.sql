-- Phase 2: Device Group Members table
-- Associates devices with groups (many-to-many)

CREATE TABLE device_group_members (
    group_id   UUID NOT NULL REFERENCES device_groups(id) ON DELETE CASCADE,
    device_id  UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    added_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, device_id)
);

CREATE INDEX idx_dgm_device ON device_group_members (device_id);
CREATE INDEX idx_dgm_group ON device_group_members (group_id);
