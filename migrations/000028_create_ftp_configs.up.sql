CREATE TABLE ftp_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_name VARCHAR(200) NOT NULL,
    host VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL DEFAULT 21,
    username VARCHAR(100) NOT NULL,
    password_encrypted TEXT,
    protocol VARCHAR(10) NOT NULL DEFAULT 'FTP',
    remote_path VARCHAR(512) NOT NULL DEFAULT '/',
    passive BOOLEAN NOT NULL DEFAULT true,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ftp_configs_enabled ON ftp_configs(enabled);
CREATE TRIGGER trigger_ftp_configs_updated_at
    BEFORE UPDATE ON ftp_configs FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
