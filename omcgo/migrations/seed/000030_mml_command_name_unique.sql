-- +goose Up
-- T-MMLSRC: 强制 command_name 唯一(同一 command_name 不允许重复)。
--
-- ⚠️ 放在 seed 阶段、且必须在 000029 去重之后:
--   init_seed(seed 000001)灌入的原始数据含重复 command_name,000029 去重后才满足唯一性。
--   若把本索引放 DDL 阶段(migrations/),DDL 阶段先于 seed 阶段执行 → 索引建在空表上,
--   随后 init_seed 灌入重复数据时违反唯一索引 → migrate-seed 失败 → app 起不来
--   (见 omcgo/CLAUDE.md §5.5.11)。每次 fresh 部署的 seed 链顺序:000001 灌(无索引)→
--   000029 去重 → 000030 建索引,始终自洽。
--
-- 用部分索引 WHERE deprecated_at IS NULL:仅约束"未软删"的活跃命令唯一;允许某命令被软删后
-- 再以同名新建(self-heal / 重建场景),也不与历史软删行冲突。
CREATE UNIQUE INDEX IF NOT EXISTS uniq_mml_commands_command_name_active
    ON public.mml_commands (command_name)
    WHERE deprecated_at IS NULL;

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS uniq_mml_commands_command_name_active;
-- +goose StatementEnd
