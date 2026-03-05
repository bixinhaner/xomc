-- Device parameters table: stores TR069 parameter values per device.
CREATE TABLE device_parameters (
    device_id        UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    parameter_path   VARCHAR(512) NOT NULL,
    parameter_value  TEXT,
    parameter_type   VARCHAR(20),
    writable         BOOLEAN DEFAULT false,
    last_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (device_id, parameter_path)
);

CREATE INDEX idx_device_params_device ON device_parameters (device_id);
