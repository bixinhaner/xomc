-- Report definitions
CREATE TABLE report_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_name VARCHAR(200) NOT NULL,
    report_type VARCHAR(20) NOT NULL,
    description TEXT,
    format JSONB NOT NULL DEFAULT '["pdf"]',
    period VARCHAR(20) NOT NULL DEFAULT 'daily',
    kpi_codes JSONB DEFAULT '[]',
    device_groups JSONB DEFAULT '[]',
    auto_generate BOOLEAN NOT NULL DEFAULT false,
    cron_expression VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    creator VARCHAR(100),
    last_gen_time TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_report_defs_type ON report_definitions(report_type);
CREATE INDEX idx_report_defs_status ON report_definitions(status);
CREATE TRIGGER trigger_report_definitions_updated_at
    BEFORE UPDATE ON report_definitions FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Report records (generated instances)
CREATE TABLE report_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_definition_id UUID NOT NULL REFERENCES report_definitions(id) ON DELETE CASCADE,
    report_name VARCHAR(200) NOT NULL,
    period VARCHAR(100),
    generate_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    file_size BIGINT NOT NULL DEFAULT 0,
    download_url TEXT,
    format VARCHAR(10) NOT NULL DEFAULT 'pdf',
    status VARCHAR(20) NOT NULL DEFAULT 'generating',
    minio_path TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_report_records_def ON report_records(report_definition_id);
CREATE INDEX idx_report_records_status ON report_records(status);
CREATE INDEX idx_report_records_time ON report_records(generate_time DESC);
