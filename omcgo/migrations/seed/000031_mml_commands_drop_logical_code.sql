-- +goose Up
-- mml_commands.logical_code 列下线：改为读时从 command_code 派生（去 "OP " 前缀）。
-- 放 seed 目录而非 migrations/ 根目录的原因（见 omcgo/CLAUDE.md §5.5.11）：
-- DDL 阶段先于 seed 阶段整体执行，init_seed / seed/000002_mml_i18n_en.sql 在写入/读取
-- logical_code 列；若把 DROP COLUMN 放 migrations/ 根，DROP 会先于 seed 执行，导致后续
-- seed 引用不存在的列而失败。放 seed 目录并取最大版本号 +1，使 DROP 在所有依赖该列的
-- seed 之后执行——列先被建立/写入/读取，最后被删除，fresh 库与存量库均自洽。
DROP INDEX IF EXISTS idx_mml_commands_logical_code;
ALTER TABLE mml_commands DROP COLUMN IF EXISTS logical_code;

-- +goose Down
ALTER TABLE mml_commands ADD COLUMN IF NOT EXISTS logical_code varchar(100);
CREATE INDEX IF NOT EXISTS idx_mml_commands_logical_code ON mml_commands(logical_code);
