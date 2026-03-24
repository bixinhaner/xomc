-- 000060: Extend source column from VARCHAR(32) to VARCHAR(128).
-- CPE-uploaded models have source like "xml_upload:{serialNumber}" which can exceed 32 chars.

ALTER TABLE data_model_definitions
    ALTER COLUMN source TYPE VARCHAR(128);
