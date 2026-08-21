# Review: Issue #274 MML 邻区增删与参数校验修复

固定点：`fix/274-mml-neighbor-validation`

规格来源：`docs/design/mml-neighbor-add-validation-fix-20260821.md`。

规范来源：`AGENTS.md`、`CLAUDE.md`、`.claude/commands/commit.md`、`.claude/commands/ship.md`。

## Scope

- `omcgo/internal/mml/console_executor.go` 修正 ADD compound 对象路径中末尾 `.{i}` 占位符识别。
- `omcgo/internal/config/parammodel/loader.go` 增加内置 MML 参数模型规范化，补齐 GSM instance object 并修正负数范围 STRING 类型。
- `omcgo/migrations/seed/000001_init_seed.sql`、`omcgo/scripts/mml_apply_config_updates_20260721.sql` 补齐现有库幂等修复。
- `omcmb/webcode/src/pages/mml/Console/*` 对旧模型数据增加前端兜底校验。
- 新增和更新 Go/Vitest 回归测试覆盖 MML seed、对象路径和负数范围校验。

## Findings

CRITICAL：0

WARNING：0

INFO：

- seed 修复涉及 `standard_params`、`param_mappings`、`mml_commands`、`mml_command_sub_fields` 的幂等更新，建议上线前在目标环境执行一次 seed/migration 演练并抽查 ADD/RMV 命令可见性。
- 前端兜底逻辑只在 STRING 且出现负数 min/max 时切换为数值范围，范围足够窄，不会改变普通 STRING 长度校验。

## Review Notes

- ADD compound 路径替换现在同时支持中间段 `.{i}.` 和末尾对象段 `.{i}`，并保留尾部点号语义；新增用例覆盖终止占位符。
- GSM Inter-RAT 对象补齐同时在标准模型、加载器和 seed 中存在，覆盖新导入模型和历史数据库两条路径。
- 负数范围 STRING 规范化在后端数据层修正，前端兜底用于旧数据、缓存数据或未重载模型场景。
- 未发现 SQL 字符串拼接注入、裸 `panic`、安全降级、测试删除或跨运营商硬编码问题。

## Validation

- `cd omcgo && go test ./internal/config/parammodel ./internal/mml`：通过。
- `cd omcgo && go build ./...`：通过。
- `cd omcgo && go test ./...`：通过。
- `cd omcmb && npm run typecheck`：通过。
- `cd omcmb && npm run test --workspace webcode -- src/pages/mml/Console/modParamValidation.test.ts src/pages/mml/Console/components/ConfigParamsModal.test.tsx`：通过，2 个文件 58 个测试通过。
- `git diff --check`：通过。

## Conclusion

结论：PASS_WITH_INFO。未发现阻塞提交的 CRITICAL 问题，可以进入提交和 MR。
