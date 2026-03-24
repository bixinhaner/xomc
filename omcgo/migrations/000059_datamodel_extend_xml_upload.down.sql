-- 000059 down: Revert XML parameter model upload extensions.

ALTER TABLE data_model_definitions DROP COLUMN IF EXISTS model_metadata;
ALTER TABLE data_model_definitions DROP COLUMN IF EXISTS object_tree;

ALTER TABLE data_model_definitions
    DROP CONSTRAINT IF EXISTS data_model_definitions_source_type_check;
ALTER TABLE data_model_definitions
    ADD CONSTRAINT data_model_definitions_source_type_check
    CHECK (source_type IN ('manual', 'auto_discovered'));
