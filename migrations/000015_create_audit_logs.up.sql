-- Phase 4 Migration: Audit logs + RBAC seed data

-- Audit logs table
CREATE TABLE audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    username    VARCHAR(64) NOT NULL,
    action      VARCHAR(64) NOT NULL,
    resource    VARCHAR(64),
    resource_id VARCHAR(128),
    details     JSONB,
    ip_address  INET,
    user_agent  VARCHAR(256),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user_time ON audit_logs (user_id, created_at DESC);
CREATE INDEX idx_audit_logs_time ON audit_logs (created_at DESC);
CREATE INDEX idx_audit_logs_resource ON audit_logs (resource, resource_id, created_at DESC);

-- Seed: System roles
INSERT INTO roles (id, name, description, is_system) VALUES
    ('10000000-0000-0000-0000-000000000001', 'admin', 'System administrator with full access', TRUE),
    ('10000000-0000-0000-0000-000000000002', 'operator', 'Operator with read/write access to operational resources', TRUE),
    ('10000000-0000-0000-0000-000000000003', 'viewer', 'Read-only viewer', TRUE);

-- Seed: Admin permissions (all resources, all actions)
INSERT INTO permissions (role_id, resource, action)
SELECT '10000000-0000-0000-0000-000000000001', r, a
FROM unnest(ARRAY['devices','alarms','pm','config','datamodels','users','roles','firmware','northbound','interop']) AS r,
     unnest(ARRAY['read','write','delete','admin']) AS a;

-- Seed: Operator permissions (read + write on operational resources)
INSERT INTO permissions (role_id, resource, action)
SELECT '10000000-0000-0000-0000-000000000002', r, a
FROM unnest(ARRAY['devices','alarms','pm','config','datamodels','firmware','northbound']) AS r,
     unnest(ARRAY['read','write']) AS a;

-- Seed: Viewer permissions (read only)
INSERT INTO permissions (role_id, resource, action)
SELECT '10000000-0000-0000-0000-000000000003', r, 'read'
FROM unnest(ARRAY['devices','alarms','pm','config','datamodels','firmware','northbound']) AS r;

-- Seed: Default admin user (password: admin123)
-- bcrypt hash of 'admin123' with cost 10
INSERT INTO users (id, username, password_hash, display_name, status) VALUES
    ('20000000-0000-0000-0000-000000000001', 'admin',
     '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
     'System Admin', 'active');

INSERT INTO user_roles (user_id, role_id) VALUES
    ('20000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001');
