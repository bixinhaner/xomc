-- +goose Up
-- ============================================================
-- 000203_disambiguate_ext_container_instance_command_names.sql
-- ============================================================
-- 方案 2：给 EXT 命令族中 `_` (容器) 与 `_I` (实例) 同名簇加后缀区分。
--
-- 现象：catalog v2 重建的 SX_DEVICE_EXT 章节下，单个对象常会同时生成两条命令：
--   · LST EXT_FAULTMGMT_CURRENTALARM_   target=Device.FaultMgmt.CurrentAlarm.
--   · LST EXT_FAULTMGMT_CURRENTALARM_I  target=Device.FaultMgmt.CurrentAlarm.{i}.
--   两者 command_name 都派生为 "查询 故障管理 · 当前告警" → 用户在树上看不出区别。
--
-- 与 000178 的关系：000178_mml_case_b2_rename.sql 处理过类似簇（IPv4 / IRAT /
-- 能力集等手工映射加 ` (物理口)` / ` (VLAN)` / ` (连接态)` 等后缀）；
-- 本次按"_ vs _I" pattern 做泛化清理（之前漏掉的 EXT 全簇）。
--
-- 策略：
--   1. 找 sibling 对：command_code 仅末尾差 `_` 与 `_I`，operation_type 相同
--   2. 仅当二者 command_name 完全相同时才动（已有手工区分的不动）
--   3. 容器版（_）追加 ` (容器)` / ` (Object)`
--   4. 实例版（_I）追加 ` (实例)` / ` (Instance)`
--   5. 同步更新 command_name_i18n.zh-CN / en-US（保持与 command_name 一致）
--      + description（与 command_name 一致，保持搜索匹配）
--
-- 重跑安全：UPDATE 时 EXCLUDE 已包含 "(容器)" / "(实例)" / "(Object)" / "(Instance)"
-- 后缀的 row，避免反复追加。
-- ============================================================

-- +goose StatementBegin
DO $$
DECLARE
    affected INT;
BEGIN
    WITH pairs AS (
        -- 找所有 (container, instance) 命令对：仅末尾 _ vs _I 不同
        SELECT a.id AS container_id, b.id AS instance_id,
               a.command_name AS shared_name
        FROM mml_commands a
        JOIN mml_commands b
          ON b.command_code = a.command_code || 'I'      -- a 末尾 `_`, b 末尾 `_I`
         AND b.operation_type = a.operation_type
         AND b.command_name = a.command_name             -- 仅同名才需要区分
         AND b.source = a.source
        WHERE a.source = 'standard'
          -- 显式 ESCAPE 让所有 `\_` 都是字面下划线（避免 LIKE 通配陷阱）
          AND a.command_code LIKE '% EXT\_%\_' ESCAPE '\'   -- 容器版：以 _ 结尾
          AND b.command_code LIKE '% EXT\_%\_I' ESCAPE '\'  -- 实例版：以 _I 结尾
          -- 防止重复追加：name 里已含区分标记的不再动
          AND a.command_name NOT LIKE '% (容器)%'
          AND a.command_name NOT LIKE '% (Object)%'
          AND b.command_name NOT LIKE '% (实例)%'
          AND b.command_name NOT LIKE '% (Instance)%'
    ),
    -- container 行：追加"(容器)"
    upd_container AS (
        UPDATE mml_commands m
           SET command_name = m.command_name || ' (容器)',
               description  = COALESCE(m.description, m.command_name) || ' (容器)',
               command_name_i18n = jsonb_set(
                   jsonb_set(
                       COALESCE(m.command_name_i18n, '{}'::jsonb),
                       '{zh-CN}',
                       to_jsonb(COALESCE(m.command_name_i18n->>'zh-CN', m.command_name) || ' (容器)')
                   ),
                   '{en-US}',
                   to_jsonb(COALESCE(m.command_name_i18n->>'en-US', m.command_name) || ' (Object)')
               ),
               updated_at = NOW()
         WHERE m.id IN (SELECT container_id FROM pairs)
        RETURNING m.id
    ),
    -- instance 行：追加"(实例)"
    upd_instance AS (
        UPDATE mml_commands m
           SET command_name = m.command_name || ' (实例)',
               description  = COALESCE(m.description, m.command_name) || ' (实例)',
               command_name_i18n = jsonb_set(
                   jsonb_set(
                       COALESCE(m.command_name_i18n, '{}'::jsonb),
                       '{zh-CN}',
                       to_jsonb(COALESCE(m.command_name_i18n->>'zh-CN', m.command_name) || ' (实例)')
                   ),
                   '{en-US}',
                   to_jsonb(COALESCE(m.command_name_i18n->>'en-US', m.command_name) || ' (Instance)')
               ),
               updated_at = NOW()
         WHERE m.id IN (SELECT instance_id FROM pairs)
        RETURNING m.id
    )
    SELECT (SELECT COUNT(*) FROM upd_container) + (SELECT COUNT(*) FROM upd_instance) INTO affected;

    RAISE NOTICE 'disambiguate_ext_container_instance: renamed % commands (含 container + instance 两侧)', affected;
END $$;
-- +goose StatementEnd


-- +goose Down
-- ============================================================
-- 回滚：去掉本次追加的 (容器) / (实例) / (Object) / (Instance) 后缀。
-- 仅匹配末尾出现的此类标签，避免误伤命令名本身就含括号的情况。
-- ============================================================

-- +goose StatementBegin
DO $$
BEGIN
    UPDATE mml_commands
       SET command_name = regexp_replace(command_name, ' \((容器|实例|Object|Instance)\)$', ''),
           description  = regexp_replace(COALESCE(description, ''),
                                         ' \((容器|实例|Object|Instance)\)$', ''),
           command_name_i18n = jsonb_set(
               jsonb_set(
                   command_name_i18n,
                   '{zh-CN}',
                   to_jsonb(regexp_replace(
                       COALESCE(command_name_i18n->>'zh-CN', ''),
                       ' \((容器|实例)\)$', ''
                   ))
               ),
               '{en-US}',
               to_jsonb(regexp_replace(
                   COALESCE(command_name_i18n->>'en-US', ''),
                   ' \((Object|Instance)\)$', ''
               ))
           ),
           updated_at = NOW()
     WHERE source = 'standard'
       AND command_code LIKE '% EXT_%'
       AND command_name ~ ' \((容器|实例|Object|Instance)\)$';
END $$;
-- +goose StatementEnd
