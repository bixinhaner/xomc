-- Sprint 5: Index optimization for foreign keys and common query patterns

-- 1. provisioning_tasks: index on template_id FK for join performance
CREATE INDEX IF NOT EXISTS idx_pt_template ON provisioning_tasks (template_id)
    WHERE template_id IS NOT NULL;

-- 2. alarms_active: index on carrier for carrier-scoped queries
CREATE INDEX IF NOT EXISTS idx_alarms_active_carrier ON alarms_active (carrier);

-- 3. user_roles: index on role_id for CASCADE delete lookups
CREATE INDEX IF NOT EXISTS idx_user_roles_role ON user_roles (role_id);

-- 4. audit_logs: index on action for action-based filtering
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs (action);

-- 5. ne_message_logs: index on message_type for type-based filtering
CREATE INDEX IF NOT EXISTS idx_ne_message_logs_message_type ON ne_message_logs (message_type);

-- 6. kpi_values: composite index for carrier+technology filtering
CREATE INDEX IF NOT EXISTS idx_kpi_values_carrier_tech ON kpi_values (carrier, technology, time DESC);

-- 7. alarms_active: composite index for carrier+severity queries (dashboard)
CREATE INDEX IF NOT EXISTS idx_alarms_active_carrier_severity ON alarms_active (carrier, severity);

-- 8. BRIN indexes for append-only tables (compact range-based access)
-- system_logs: BRIN on created_at
CREATE INDEX IF NOT EXISTS idx_system_logs_created_at_brin ON system_logs USING BRIN (created_at)
    WITH (pages_per_range = 128);

-- ne_message_logs: BRIN on created_at
CREATE INDEX IF NOT EXISTS idx_ne_message_logs_created_at_brin ON ne_message_logs USING BRIN (created_at)
    WITH (pages_per_range = 128);

-- alarms_active: BRIN on raised_at
CREATE INDEX IF NOT EXISTS idx_alarms_active_raised_at_brin ON alarms_active USING BRIN (raised_at)
    WITH (pages_per_range = 128);
