-- +goose Up
-- ============================================================
-- 000176_mml_extension_sub_field_label_humanize_t0171.sql
-- T-0171 增量：extension sub_field 的 label_i18n 人性化命名
--
-- 用户反馈（2026-05-24）：
--   000174 派生时把 standard_path 完整路径直接塞进 label_i18n，UI 显示
--   "Device.DeviceInfo.AntennaInfo.Azimuth" 这种长串，对用户不友好。
--   应参考 spec md 对 path 的命名约定，转人类可读形态。
--
-- 派生规则：
--   英文名（en）= path 末段 CamelCase 拆词 + snake 转空格
--     例: UserLabel       → "User Label"
--         AntennaAzimuth  → "Antenna Azimuth"
--         GPSLatitude     → "GPS Latitude"      (upper-cluster→lower 边界)
--         3GPPSpecVersion → "3GPP Spec Version" (数字+大写连用保留)
--         1588_Status     → "1588 Status"
--         LAN_CONFIG_IPADDR → "LAN CONFIG IPADDR" (全大写 snake 保留)
--
--   中文名（zh）优先级:
--     1. standard_params.description（若非空 — 适用于 spec md 71 group 派生的 path）
--     2. 兜底 = 同英文名（业务方后续补 zh 字典再覆盖）
--
-- key 对齐:
--   - standard catalog 现状用 {"en":..., "zh":...}（无 -US/-CN 后缀）
--   - extension 此前用 {"en-US":..., "zh-CN":...}（不一致）
--   - 本次统一为 {"en":..., "zh":...}（与 standard 对齐，前端读法统一）
--
-- 影响范围:
--   仅更新 source='extension' 命令的 sub_field（不动 standard / admin）
--   2795 条 sub_field 全部 UPDATE
-- ============================================================

-- ----------------------------------------------------------------------------
-- Step 1: extension sub_field label 人性化（path 末段拆词 + 中文用 description）
-- ----------------------------------------------------------------------------
UPDATE mml_command_sub_fields csf
   SET label_i18n = jsonb_build_object(
           'en', readable.en_name,
           'zh', COALESCE(NULLIF(sp.description, ''), readable.en_name)
       ),
       updated_at = NOW()
  FROM mml_commands c, standard_params sp,
       LATERAL (
           SELECT TRIM(REGEXP_REPLACE(
                       REGEXP_REPLACE(
                           REGEXP_REPLACE(
                               -- 取 path 末段（最后一个 . 后的所有字符）
                               REGEXP_REPLACE(sp.standard_path, '^.*\.', ''),
                               '([a-z])([A-Z])', '\1 \2', 'g'),       -- CamelCase 间插空格
                           '([A-Z])([A-Z][a-z])', '\1 \2', 'g'),       -- upper-cluster 边界
                       '_+', ' ', 'g')                                  -- snake 下划线 → 空格
                   ) AS en_name
       ) readable
 WHERE csf.command_id = c.id
   AND csf.standard_path_id = sp.id
   AND c.source = 'extension';


-- ----------------------------------------------------------------------------
-- Step 2: 顺便统一 standard catalog 的 label_i18n key — 614 条用旧 'en-US/zh-CN'
-- 转为 'en/zh'（与新 1780 条 + extension 2795 条对齐），不动 value 保留原中文名
-- ----------------------------------------------------------------------------
UPDATE mml_command_sub_fields csf
   SET label_i18n = jsonb_build_object(
           'en', csf.label_i18n->>'en-US',
           'zh', csf.label_i18n->>'zh-CN'
       ),
       updated_at = NOW()
  FROM mml_commands c
 WHERE csf.command_id = c.id
   AND c.source = 'standard'
   AND csf.label_i18n ? 'en-US'
   AND csf.label_i18n ? 'zh-CN'
   AND NOT (csf.label_i18n ? 'en' AND csf.label_i18n ? 'zh');


-- 自检
-- +goose StatementBegin
DO $$
DECLARE
    updated_count    INT;
    still_path_count INT;
BEGIN
    SELECT COUNT(*) INTO updated_count
      FROM mml_command_sub_fields csf
      JOIN mml_commands c ON c.id = csf.command_id
     WHERE c.source = 'extension';

    -- 检测：label 还含 path 形态（含多个 "." 的认为是未拆词的完整 path）
    SELECT COUNT(*) INTO still_path_count
      FROM mml_command_sub_fields csf
      JOIN mml_commands c ON c.id = csf.command_id
     WHERE c.source = 'extension'
       AND (
           position('.' IN csf.label_i18n->>'en') > 0
        OR position('.' IN csf.label_i18n->>'zh') > 0
       );

    RAISE NOTICE 'T-0171 label humanize complete:';
    RAISE NOTICE '  Extension sub_fields updated: %', updated_count;
    RAISE NOTICE '  Labels still containing dots (should be 0): %', still_path_count;

    IF still_path_count > 0 THEN
        RAISE EXCEPTION 'Humanize failed: % labels still contain dots (path not transformed)', still_path_count;
    END IF;
END $$;
-- +goose StatementEnd


-- +goose Down
-- ============================================================
-- 回滚：
--   1. extension sub_field label 恢复旧形态 {"en-US": 完整path, "zh-CN": 完整path}
--   2. standard catalog 的 key 转换不做 Down — 原本就是 614+1780 混乱状态，
--      统一为 en/zh 更优；如真需回滚混乱，需 git revert 配合精确数据备份
-- ============================================================
UPDATE mml_command_sub_fields csf
   SET label_i18n = jsonb_build_object(
           'en-US', sp.standard_path,
           'zh-CN', sp.standard_path
       ),
       updated_at = NOW()
  FROM mml_commands c, standard_params sp
 WHERE csf.command_id = c.id
   AND csf.standard_path_id = sp.id
   AND c.source = 'extension';
