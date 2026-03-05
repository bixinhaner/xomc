-- Phase 2: Data Model Import Log table
-- Audit trail for data model lifecycle operations

CREATE TABLE data_model_import_log (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    data_model_id    UUID NOT NULL REFERENCES data_model_definitions(id) ON DELETE CASCADE,
    action           VARCHAR(16) NOT NULL
                     CHECK (action IN ('created', 'updated', 'activated', 'deprecated', 'deleted')),
    performed_by     VARCHAR(128) NOT NULL,
    changes_summary  JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dm_import_log_model ON data_model_import_log (data_model_id, created_at DESC);
CREATE INDEX idx_dm_import_log_action ON data_model_import_log (action);
