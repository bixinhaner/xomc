-- G28: Extend device_info with 4 new quick-query columns
ALTER TABLE device_info
    ADD COLUMN num_of_cells     INTEGER DEFAULT 1,
    ADD COLUMN gps_status       VARCHAR(20),
    ADD COLUMN alarm_severity   VARCHAR(20),
    ADD COLUMN license_status   VARCHAR(20);

CREATE INDEX idx_device_info_gps_status ON device_info (gps_status);
CREATE INDEX idx_device_info_alarm_severity ON device_info (alarm_severity);
CREATE INDEX idx_device_info_license_status ON device_info (license_status);
