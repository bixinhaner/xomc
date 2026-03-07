CREATE TABLE licenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    license_name VARCHAR(200) NOT NULL,
    license_code VARCHAR(100) NOT NULL UNIQUE,
    product_name VARCHAR(200) NOT NULL,
    license_type VARCHAR(20) NOT NULL DEFAULT 'subscription',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    max_devices INTEGER NOT NULL DEFAULT 0,
    used_devices INTEGER NOT NULL DEFAULT 0,
    features JSONB NOT NULL DEFAULT '[]',
    issue_date TIMESTAMPTZ NOT NULL,
    expiry_date TIMESTAMPTZ,
    licensor VARCHAR(200),
    device_type VARCHAR(50),
    region VARCHAR(100),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_licenses_status ON licenses(status);
CREATE INDEX idx_licenses_code ON licenses(license_code);
CREATE INDEX idx_licenses_type ON licenses(license_type);
CREATE INDEX idx_licenses_device_type ON licenses(device_type);
CREATE INDEX idx_licenses_expiry ON licenses(expiry_date);
CREATE TRIGGER trigger_licenses_updated_at
    BEFORE UPDATE ON licenses FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
