-- +goose Up
-- ============================================================
-- 000116_mml_categories_consolidate.sql —— 已下线 noop（2026-05-22）
--
-- 原内容：把 mml_command_groups 从「90 一级 + 383 keyword」折叠为「10 老 OMC
-- 一级大类」（用户决策 2026-05-18 — Phase C）。Step 1 INSERT 10 个 top_*
-- 新顶（param_version='STANDARD'）、Step 2/3 reparent 命令、Step 4 删老
-- keyword 子组。
--
-- 下线原因：
--   1. 源数据：老 90 一级 + 383 keyword catalog 由 seed/000111
--      (mml_old_catalog_import) 写入，该 seed 已下线 noop（被 v2.3 catalog
--      替代）。fresh DB 上没有 90 一级数据可折叠。
--   2. FK 阻塞：Step 1 的 INSERT 引用 `param_version='STANDARD'`，但
--      main/000154_drop_mml_params 主动删除了 mml_param_versions 的 STANDARD
--      anchor 行 → fresh DB 上必报 FK 23503 "violates foreign key constraint
--      mml_param_groups_param_version_fkey"。
--   3. 目标数据：10 top_* 新顶在 seed/000155_mml_catalog_v23_canonicalize.sql
--      Step 2 又被显式 DELETE（注释明示 "0 commands 无数据损失"），整段
--      迁移净效果为 0。
--   4. v2.3 单源 catalog 由 seed/000152 + 000155 重组，不再使用 STANDARD
--      top_* 分组体系。
--
-- 与 seed/000005/006/007/008/096/111 同期 noop 处理一致（这一批 mml 旧
-- catalog seed 由 v2.3 catalog 单源化整体替代）。Down 段保持 noop 对称。
-- ============================================================
SELECT 1;

-- +goose Down
SELECT 1;
