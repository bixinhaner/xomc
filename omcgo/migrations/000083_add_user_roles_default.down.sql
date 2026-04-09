DROP INDEX IF EXISTS uniq_user_default_role;
ALTER TABLE user_roles DROP COLUMN IF EXISTS is_default;
