-- 000059: Extend data_model_definitions for XML parameter model upload.
-- Adds object_tree, model_metadata columns and expands source_type CHECK.

-- 1. Expand source_type CHECK to include 'cpe_uploaded'.
--    Drop both possible constraint names (055 used chk_dm_source_type, earlier migrations used the default name).
ALTER TABLE data_model_definitions
    DROP CONSTRAINT IF EXISTS chk_dm_source_type;
ALTER TABLE data_model_definitions
    DROP CONSTRAINT IF EXISTS data_model_definitions_source_type_check;
ALTER TABLE data_model_definitions
    ADD CONSTRAINT data_model_definitions_source_type_check
    CHECK (source_type IN ('manual', 'auto_discovered', 'cpe_uploaded'));

-- 2. Add object_tree column (stores parsed object node definitions from XML).
ALTER TABLE data_model_definitions
    ADD COLUMN IF NOT EXISTS object_tree JSONB;

-- 3. Add model_metadata column (stores XML metadata: generateTime, modelVersion, totalEntries).
ALTER TABLE data_model_definitions
    ADD COLUMN IF NOT EXISTS model_metadata JSONB;

COMMENT ON COLUMN data_model_definitions.object_tree IS 'JSON array of object node definitions parsed from XML parameter model';
COMMENT ON COLUMN data_model_definitions.model_metadata IS 'XML metadata: generateTime, modelVersion, totalEntries, serialNumber';
