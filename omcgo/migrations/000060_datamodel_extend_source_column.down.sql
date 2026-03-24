-- 000060 rollback: Revert source column to VARCHAR(32).
ALTER TABLE data_model_definitions
    ALTER COLUMN source TYPE VARCHAR(32);
