-- ============================================================
-- 000155_mml_catalog_v23_canonicalize.sql
-- v2.3 MML catalog 单源化（方案 A）：删重复 + 规范 group_code 前缀
--
-- 背景：commit 3bc401f8 引入 seed/000152 通过 Goose 注入 v2.3 (cmcc-td-lte-v2.3)
--      catalog（18 章 / 190 命令 / 1054 sub_fields），同时
--      `internal/mml/catalogloader/*` 启动期 Loader 也读同一份 JSON
--      (datamodels/mml-catalog/cmcc-tdlte-v2.3.json) 创建 cmcc-lte-v2.3
--      catalog（18 章 / 190 命令 / 0 sub_fields）。两条注入路径并存导致：
--        · DB 里出现两份 v2.3 catalog（version_code 不同、group_code 前缀习惯不同）
--        · 后端 SQL `g.group_code LIKE 'chapter:%'` 只看到 Loader 注入的旧版
--          （chapter:SA 前缀），但旧版 0 sub_fields → 用户点命令面板空
--        · seed/000152 注入的新版用裸 SA 前缀（不符 spec §R-1），但 1054 sub_fields
--          在它身上 → 后端 SQL 看不到
--      用户决策：删旧、留新、给新版加 chapter: 前缀，单一来源。
--
-- 改动：
--   1. DELETE cmcc-lte-v2.3 catalog（旧 Loader 自动注入；0 sub_fields 无数据损失）
--   2. DELETE 老的 STANDARD top_* 空壳分组（seed/000116 残留；0 commands 无数据损失）
--   3. UPDATE cmcc-td-lte-v2.3 group_code 加 chapter: 前缀，path 加 chapter_ 前缀
--      （与 spec §R-1 "一级分组稳定标识 group_code=chapter:<SA-SR>" 对齐，
--      后端 SQL `LIKE 'chapter:%'` 即可命中）
--   4. UPDATE mml_param_versions: 新版描述标注 "STANDARD canonical"
--
-- 不动：
--   · mml_param_versions 的 STANDARD anchor 行（4304 行 mml_params 仍挂着，
--     与 v2.3 catalog 无业务关联，属老 MML 参数库 legacy 数据；强行重命名
--     cmcc-td-lte-v2.3 → STANDARD 会与现有 STANDARD anchor 冲突，且 FK
--     ON DELETE RESTRICT 阻塞）
--   · 任何 customized 分组（source='admin'）
--   · 任何运行时数据
--
-- 阻止再次撕裂（配套改动，本迁移外）：
--   · cmd/app/provider/dictload.go：mmlCatalogLoader 从启动列表移除
--   · seed/000152 + cmd/tools/gen_seed_sql/main.go：emit chapter: 前缀
--
-- 幂等性：所有改动按 WHERE 过滤（已加前缀 / 不存在）→ 二次执行 no-op。
-- ============================================================

-- +goose Up

-- ---------- Step 1: 删 cmcc-lte-v2.3 重复 catalog ----------
-- mml_commands.group_id FK ON DELETE SET NULL（seed/000116 注释），手工先删 commands
-- 避免遗留孤儿 group_id=NULL 行污染后续命令树。
DELETE FROM mml_commands
 WHERE group_id IN (
        SELECT id FROM mml_command_groups WHERE param_version = 'cmcc-lte-v2.3'
       );

DELETE FROM mml_command_groups
 WHERE param_version = 'cmcc-lte-v2.3';

-- mml_param_versions ON DELETE RESTRICT；上两步已清完 FK 引用
DELETE FROM mml_param_versions
 WHERE version_code = 'cmcc-lte-v2.3';

-- ---------- Step 2: 删老 STANDARD top_* 空壳分组 ----------
-- seed/000116 注入的 10 个 top_btsinfo / top_btssetting / top_lte / ... 行，
-- 0 commands、0 sub_fields，纯空壳。
-- mml_param_versions 的 STANDARD anchor 不动（4304 行 mml_params 还挂着，是另一个 legacy）。
DELETE FROM mml_command_groups
 WHERE param_version = 'STANDARD'
   AND source = 'standard'
   AND group_code LIKE 'top\_%' ESCAPE '\';

-- ---------- Step 3: 把新 catalog (cmcc-td-lte-v2.3) 规范为 chapter:SA 前缀 ----------
-- spec §R-1 要求一级分组 group_code = chapter:<SA-SR>；
-- ltree path 不允许冒号，path 用 chapter_<X> 形式。
-- 幂等：WHERE 排除已加前缀的行，二次执行 0 rows updated。
UPDATE mml_command_groups
   SET group_code = 'chapter:' || group_code,
       path       = ('chapter_' || path::text)::ltree
 WHERE param_version = 'cmcc-td-lte-v2.3'
   AND group_code NOT LIKE 'chapter:%';

-- ---------- Step 4: 标注 cmcc-td-lte-v2.3 为 STANDARD canonical ----------
UPDATE mml_param_versions
   SET version_name = 'CMCC TD-LTE v2.3 (STANDARD canonical)',
       description  = 'Single source of truth for v2.3 MML catalog. '
                   || 'Normalized to chapter: prefix by seed/000155; '
                   || 'duplicate cmcc-lte-v2.3 + empty STANDARD top_* groups removed.'
 WHERE version_code = 'cmcc-td-lte-v2.3';


-- +goose Down

-- Down 仅恢复 description / version_name；group_code 前缀回滚有数据损失风险
-- （前端代码已切换到 chapter: 前缀语义），不做实际回滚。
UPDATE mml_param_versions
   SET version_name = 'CMCC TD-LTE v2.3',
       description  = '由 cmcc_tdlte_v2.3.json 派生（goose seed 000152）'
 WHERE version_code = 'cmcc-td-lte-v2.3';
