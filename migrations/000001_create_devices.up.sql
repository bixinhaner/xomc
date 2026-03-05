-- Devices table: partitioned by carrier for data isolation and query performance.
CREATE TABLE devices (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    serial_number          VARCHAR(64) NOT NULL,
    oui                    VARCHAR(6) NOT NULL,
    product_class          VARCHAR(64),
    manufacturer           VARCHAR(128),
    model_name             VARCHAR(128),
    carrier                VARCHAR(4) NOT NULL,
    technology             VARCHAR(3) NOT NULL,
    data_model_id          UUID,
    status                 VARCHAR(20) NOT NULL DEFAULT 'discovered',
    firmware_version       VARCHAR(64),
    ip_address             INET,
    connection_request_url VARCHAR(256),
    last_inform_at         TIMESTAMPTZ,
    last_inform_events     JSONB,
    inform_interval        INTEGER DEFAULT 300,
    site_name              VARCHAR(128),
    site_id                VARCHAR(64),
    latitude               DOUBLE PRECISION,
    longitude              DOUBLE PRECISION,
    extension_data         JSONB,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY LIST (carrier);

-- Carrier partitions
CREATE TABLE devices_cmcc PARTITION OF devices FOR VALUES IN ('cmcc');
CREATE TABLE devices_ctcc PARTITION OF devices FOR VALUES IN ('ctcc');
CREATE TABLE devices_cucc PARTITION OF devices FOR VALUES IN ('cucc');

-- Indexes
CREATE UNIQUE INDEX idx_devices_serial_number ON devices (serial_number);
CREATE INDEX idx_devices_carrier_status ON devices (carrier, status);
CREATE INDEX idx_devices_oui ON devices (oui);
CREATE INDEX idx_devices_carrier_tech ON devices (carrier, technology);
CREATE INDEX idx_devices_status ON devices (status);
CREATE INDEX idx_devices_last_inform ON devices (last_inform_at);

-- Updated_at trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_devices_updated_at
    BEFORE UPDATE ON devices
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
