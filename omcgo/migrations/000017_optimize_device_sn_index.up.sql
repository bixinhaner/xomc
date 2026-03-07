-- Optimize device serial number lookup with unique index
-- Include carrier in unique index since devices table is partitioned by carrier
CREATE UNIQUE INDEX IF NOT EXISTS idx_devices_serial_number ON devices(serial_number, carrier);

-- Composite index for carrier + status filtering
CREATE INDEX IF NOT EXISTS idx_devices_carrier_status ON devices(carrier, status);
