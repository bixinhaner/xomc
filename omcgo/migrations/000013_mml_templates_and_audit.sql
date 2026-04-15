-- +goose Up

-- MML Templates table: stores user-defined command parameter templates
CREATE TABLE IF NOT EXISTS mml_templates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_name   VARCHAR(200) NOT NULL,
    command_code    VARCHAR(100) NOT NULL,
    operation_type  VARCHAR(20) NOT NULL,
    template_scope  VARCHAR(20) NOT NULL DEFAULT 'private',
    parameters      JSONB NOT NULL DEFAULT '{}'::jsonb,
    param_paths     JSONB NOT NULL DEFAULT '[]'::jsonb,
    description     TEXT,
    product_types   JSONB NOT NULL DEFAULT '[]'::jsonb,
    creator         VARCHAR(100) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_mml_templates_scope CHECK (template_scope IN ('private', 'public')),
    CONSTRAINT chk_mml_templates_op CHECK (operation_type IN ('LST', 'MOD', 'ADD', 'RMV'))
);

CREATE INDEX IF NOT EXISTS idx_mml_templates_command_code ON mml_templates(command_code);
CREATE INDEX IF NOT EXISTS idx_mml_templates_scope_creator ON mml_templates(template_scope, creator);
CREATE INDEX IF NOT EXISTS idx_mml_templates_product_types_gin ON mml_templates USING GIN (product_types);
CREATE INDEX IF NOT EXISTS idx_mml_templates_parameters_gin ON mml_templates USING GIN (parameters);

-- Trigger for auto-updating updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_mml_templates_updated_at ON mml_templates;
CREATE TRIGGER trigger_mml_templates_updated_at
    BEFORE UPDATE ON mml_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- MML Audit Log table: records command execution for compliance
CREATE TABLE IF NOT EXISTS mml_audit_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id         UUID REFERENCES mml_tasks(id) ON DELETE SET NULL,
    command_code    VARCHAR(100) NOT NULL,
    operation_type  VARCHAR(20) NOT NULL,
    device_sn       VARCHAR(100) NOT NULL,
    parameters      JSONB DEFAULT '{}'::jsonb,
    param_paths     JSONB DEFAULT '[]'::jsonb,
    result_status   VARCHAR(20),
    result_message  TEXT,
    creator         VARCHAR(100) NOT NULL,
    executed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    duration_ms     INT
);

CREATE INDEX IF NOT EXISTS idx_mml_audit_task_id ON mml_audit_log(task_id);
CREATE INDEX IF NOT EXISTS idx_mml_audit_device_sn ON mml_audit_log(device_sn);
CREATE INDEX IF NOT EXISTS idx_mml_audit_creator ON mml_audit_log(creator);
CREATE INDEX IF NOT EXISTS idx_mml_audit_executed_at ON mml_audit_log(executed_at DESC);

-- +goose Down
DROP TABLE IF EXISTS mml_audit_log;
DROP TABLE IF EXISTS mml_templates;
