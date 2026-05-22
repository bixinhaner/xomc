-- +goose Up
-- ============================================================
-- 000116_mml_categories_consolidate.sql
-- 把 mml_command_groups 从 90 一级 + 383 keyword 折叠为 10 老 OMC 一级大类
-- （用户决策 2026-05-18 — Phase C：参考老 OMC 系统命令树结构）
--
-- 影响：
--   · 10 个新顶（fixed UUID）：BTS Info / BTS Setting / Network / System /
--     LTE Setting / Special Configuration / Maintenance / Electric Adjustment
--     Configuration / EU&RU Setting / VSWR
--   · 87 个旧一级大类映射到 10 新顶（命令 reparent）
--   · 3 个旧一级大类 DROP：性能（导入误归，老 OMC 无此 MML 大类）/
--     防疫中台 / 防疫中台(原始)（业务无关临时模块，重复条目）
--   · 老 90 一级 + 383 keyword 子组全删（CASCADE 由 parent_id FK 自动处理）
--   · mml_commands.group_id ON DELETE SET NULL：先 UPDATE 后 DELETE 保 FK 不悬空
--   · DROP 类目下命令 ~10-15 条一并删除（mml_command_sub_fields CASCADE）
--
-- 顺序：
--   Step 1: INSERT 10 个新顶（固定 UUID，ON CONFLICT 幂等）
--   Step 2: DELETE 3 个 DROP 大类下的所有命令（sub_fields 随 CASCADE）
--   Step 3: UPDATE 剩余 mml_commands.group_id 老 keyword → 新顶
--   Step 4: DELETE 所有 source='standard' 但非 10 新顶的 mml_command_groups
--           （parent_id ON DELETE CASCADE 自动清掉 keyword 子组）
--
-- 幂等：所有步骤可重复执行（ON CONFLICT / JOIN 不命中 = no-op）
-- 回滚：90→10 映射有损不可精确恢复；down 仅删 10 新顶，需重跑
--       seed/000111_mml_old_catalog_import.sql 还原 90+383 结构
-- ============================================================

-- ---------- Step 1: 10 个新顶（固定 UUID 便于 Step 3 映射引用） ----------
-- 列集对齐 seed/000111_mml_old_catalog_import.sql 已用的 11 列；migration
-- 000090 把 level / parent_id / 一堆 BOOLEAN 元属性 DROP 掉了，本表当前
-- schema 不含 level（fix 自 SQLSTATE 42703 失败回滚）。
INSERT INTO mml_command_groups
    (id, group_code, group_name_zh, group_name_en, name_i18n, path,
     param_version, display_order, is_active, source, catalog_protected)
VALUES
    ('b0c0d0e0-0001-4000-8000-000000000001'::uuid, 'top_btsinfo',
        '设备信息', 'BTS Info',
        '{"zh-CN":"设备信息","en-US":"BTS Info"}'::jsonb,
        'top_btsinfo'::ltree, 'STANDARD', 0, true, 'standard', true),
    ('b0c0d0e0-0001-4000-8000-000000000002'::uuid, 'top_btssetting',
        '设备设置', 'BTS Setting',
        '{"zh-CN":"设备设置","en-US":"BTS Setting"}'::jsonb,
        'top_btssetting'::ltree, 'STANDARD', 1, true, 'standard', true),
    ('b0c0d0e0-0001-4000-8000-000000000003'::uuid, 'top_network',
        '网络', 'Network',
        '{"zh-CN":"网络","en-US":"Network"}'::jsonb,
        'top_network'::ltree, 'STANDARD', 2, true, 'standard', true),
    ('b0c0d0e0-0001-4000-8000-000000000004'::uuid, 'top_system',
        '系统', 'System',
        '{"zh-CN":"系统","en-US":"System"}'::jsonb,
        'top_system'::ltree, 'STANDARD', 3, true, 'standard', true),
    ('b0c0d0e0-0001-4000-8000-000000000005'::uuid, 'top_lte',
        'LTE 配置', 'LTE Setting',
        '{"zh-CN":"LTE 配置","en-US":"LTE Setting"}'::jsonb,
        'top_lte'::ltree, 'STANDARD', 4, true, 'standard', true),
    ('b0c0d0e0-0001-4000-8000-000000000006'::uuid, 'top_special',
        '高级配置', 'Special Configuration',
        '{"zh-CN":"高级配置","en-US":"Special Configuration"}'::jsonb,
        'top_special'::ltree, 'STANDARD', 5, true, 'standard', true),
    ('b0c0d0e0-0001-4000-8000-000000000007'::uuid, 'top_maintenance',
        '维护', 'Maintenance',
        '{"zh-CN":"维护","en-US":"Maintenance"}'::jsonb,
        'top_maintenance'::ltree, 'STANDARD', 6, true, 'standard', true),
    ('b0c0d0e0-0001-4000-8000-000000000008'::uuid, 'top_elecadj',
        '电调天线', 'Electric Adjustment Configuration',
        '{"zh-CN":"电调天线","en-US":"Electric Adjustment Configuration"}'::jsonb,
        'top_elecadj'::ltree, 'STANDARD', 7, true, 'standard', true),
    ('b0c0d0e0-0001-4000-8000-000000000009'::uuid, 'top_euru',
        'EU&RU', 'EU&RU Setting',
        '{"zh-CN":"EU&RU","en-US":"EU&RU Setting"}'::jsonb,
        'top_euru'::ltree, 'STANDARD', 8, true, 'standard', true),
    ('b0c0d0e0-0001-4000-8000-00000000000a'::uuid, 'top_vswr',
        '驻波比', 'VSWR',
        '{"zh-CN":"驻波比","en-US":"VSWR"}'::jsonb,
        'top_vswr'::ltree, 'STANDARD', 9, true, 'standard', true)
ON CONFLICT (param_version, group_code) DO NOTHING;

-- ---------- Step 2: 删除 DROP 类目下的所有命令 ----------
-- 通过 keyword 子组（depth=2）的 parent depth-1 名定位
-- mml_command_sub_fields 通过 FK CASCADE 随 mml_commands 删除一并清理
DELETE FROM mml_commands
WHERE group_id IN (
    SELECT kw.id
    FROM mml_command_groups kw
    JOIN mml_command_groups p
         ON p.path = subltree(kw.path, 0, 1)
         AND p.source = 'standard'
    WHERE kw.source = 'standard'
      AND nlevel(kw.path) = 2
      AND p.group_name_zh IN ('性能', '防疫中台', '防疫中台(原始)')
);

-- ---------- Step 3: UPDATE mml_commands.group_id 老 keyword → 新顶 ----------
-- 映射表（CTE）：87 个旧大类名 → 10 个新顶 UUID
-- 通过 keyword 的 depth-1 parent 名 JOIN，无需逐条命令处理
WITH name_to_top (old_name, new_top_id) AS (
    VALUES
        -- 1. BTS Info 设备信息（5 条）
        ('设备信息',           'b0c0d0e0-0001-4000-8000-000000000001'::uuid),
        ('BTS',                'b0c0d0e0-0001-4000-8000-000000000001'::uuid),
        ('快速设置',           'b0c0d0e0-0001-4000-8000-000000000001'::uuid),
        ('总览',               'b0c0d0e0-0001-4000-8000-000000000001'::uuid),
        ('定时器',             'b0c0d0e0-0001-4000-8000-000000000001'::uuid),
        -- 2. BTS Setting 设备设置（18 条）
        ('基本配置',           'b0c0d0e0-0001-4000-8000-000000000002'::uuid),
        ('基站配置',           'b0c0d0e0-0001-4000-8000-000000000002'::uuid),
        ('通用的配置',         'b0c0d0e0-0001-4000-8000-000000000002'::uuid),
        ('参数配置',           'b0c0d0e0-0001-4000-8000-000000000002'::uuid),
        ('管理设置',           'b0c0d0e0-0001-4000-8000-000000000002'::uuid),
        ('载波设置',           'b0c0d0e0-0001-4000-8000-000000000002'::uuid),
        ('特性配置',           'b0c0d0e0-0001-4000-8000-000000000002'::uuid),
        ('License',            'b0c0d0e0-0001-4000-8000-000000000002'::uuid),
        ('License管理',        'b0c0d0e0-0001-4000-8000-000000000002'::uuid),
        ('SON功能设置',        'b0c0d0e0-0001-4000-8000-000000000002'::uuid),
        ('LMT设置',            'b0c0d0e0-0001-4000-8000-000000000002'::uuid),
        ('HTTP设置',           'b0c0d0e0-0001-4000-8000-000000000002'::uuid),
        ('EPC',                'b0c0d0e0-0001-4000-8000-000000000002'::uuid),
        ('HaloD',              'b0c0d0e0-0001-4000-8000-000000000002'::uuid),
        ('MME 与 EPC',         'b0c0d0e0-0001-4000-8000-000000000002'::uuid),
        ('安全',               'b0c0d0e0-0001-4000-8000-000000000002'::uuid),
        ('修改密码',           'b0c0d0e0-0001-4000-8000-000000000002'::uuid),
        ('MOCN配置',           'b0c0d0e0-0001-4000-8000-000000000002'::uuid),
        -- 3. Network 网络（19 条）
        ('网络',               'b0c0d0e0-0001-4000-8000-000000000003'::uuid),
        ('WAN&LAN',            'b0c0d0e0-0001-4000-8000-000000000003'::uuid),
        ('LAN',                'b0c0d0e0-0001-4000-8000-000000000003'::uuid),
        ('LAN口上网配置',      'b0c0d0e0-0001-4000-8000-000000000003'::uuid),
        ('VLAN',               'b0c0d0e0-0001-4000-8000-000000000003'::uuid),
        ('DNS',                'b0c0d0e0-0001-4000-8000-000000000003'::uuid),
        ('Bridge',             'b0c0d0e0-0001-4000-8000-000000000003'::uuid),
        ('Host',               'b0c0d0e0-0001-4000-8000-000000000003'::uuid),
        ('MTU',                'b0c0d0e0-0001-4000-8000-000000000003'::uuid),
        ('路由',               'b0c0d0e0-0001-4000-8000-000000000003'::uuid),
        ('keepalived',         'b0c0d0e0-0001-4000-8000-000000000003'::uuid),
        ('QOS',                'b0c0d0e0-0001-4000-8000-000000000003'::uuid),
        ('QoS业务设置',        'b0c0d0e0-0001-4000-8000-000000000003'::uuid),
        ('LBO设置',            'b0c0d0e0-0001-4000-8000-000000000003'::uuid),
        ('X2设置',             'b0c0d0e0-0001-4000-8000-000000000003'::uuid),
        ('Xn',                 'b0c0d0e0-0001-4000-8000-000000000003'::uuid),
        ('AMF',                'b0c0d0e0-0001-4000-8000-000000000003'::uuid),
        ('Mgw',                'b0c0d0e0-0001-4000-8000-000000000003'::uuid),
        ('Msc',                'b0c0d0e0-0001-4000-8000-000000000003'::uuid),
        -- 4. System 系统（1 条）
        ('同步',               'b0c0d0e0-0001-4000-8000-000000000004'::uuid),
        -- 5. LTE Setting LTE 配置（22 条）
        ('邻区',               'b0c0d0e0-0001-4000-8000-000000000005'::uuid),
        ('切换',               'b0c0d0e0-0001-4000-8000-000000000005'::uuid),
        ('Handover',           'b0c0d0e0-0001-4000-8000-000000000005'::uuid),
        ('BWP',                'b0c0d0e0-0001-4000-8000-000000000005'::uuid),
        ('DRX配置',            'b0c0d0e0-0001-4000-8000-000000000005'::uuid),
        ('CSIRS报告的配置',    'b0c0d0e0-0001-4000-8000-000000000005'::uuid),
        ('SSB',                'b0c0d0e0-0001-4000-8000-000000000005'::uuid),
        ('LTE同频邻区',        'b0c0d0e0-0001-4000-8000-000000000005'::uuid),
        ('UMTS邻频邻区',       'b0c0d0e0-0001-4000-8000-000000000005'::uuid),
        ('RRC信令配置',        'b0c0d0e0-0001-4000-8000-000000000005'::uuid),
        ('RRC参数列表',        'b0c0d0e0-0001-4000-8000-000000000005'::uuid),
        ('信令',               'b0c0d0e0-0001-4000-8000-000000000005'::uuid),
        ('小区选择与重选',     'b0c0d0e0-0001-4000-8000-000000000005'::uuid),
        ('TDD 上下行配比',     'b0c0d0e0-0001-4000-8000-000000000005'::uuid),
        ('信道配置',           'b0c0d0e0-0001-4000-8000-000000000005'::uuid),
        ('上行失步定时器',     'b0c0d0e0-0001-4000-8000-000000000005'::uuid),
        ('移动性参数',         'b0c0d0e0-0001-4000-8000-000000000005'::uuid),
        ('接入容量配置',       'b0c0d0e0-0001-4000-8000-000000000005'::uuid),
        ('空口限速设置',       'b0c0d0e0-0001-4000-8000-000000000005'::uuid),
        ('附加测量配置',       'b0c0d0e0-0001-4000-8000-000000000005'::uuid),
        ('速率相关参数',       'b0c0d0e0-0001-4000-8000-000000000005'::uuid),
        ('RSSI',               'b0c0d0e0-0001-4000-8000-000000000005'::uuid),
        -- 6. Special Configuration 高级配置（9 条）
        ('高级',               'b0c0d0e0-0001-4000-8000-000000000006'::uuid),
        ('AMBR',               'b0c0d0e0-0001-4000-8000-000000000006'::uuid),
        ('LBT',                'b0c0d0e0-0001-4000-8000-000000000006'::uuid),
        ('LGW',                'b0c0d0e0-0001-4000-8000-000000000006'::uuid),
        ('LGW设置',            'b0c0d0e0-0001-4000-8000-000000000006'::uuid),
        ('HaloB',              'b0c0d0e0-0001-4000-8000-000000000006'::uuid),
        ('SAS',                'b0c0d0e0-0001-4000-8000-000000000006'::uuid),
        ('UE',                 'b0c0d0e0-0001-4000-8000-000000000006'::uuid),
        ('终端业务控制',       'b0c0d0e0-0001-4000-8000-000000000006'::uuid),
        -- 7. Maintenance 维护（9 条）
        ('运维',               'b0c0d0e0-0001-4000-8000-000000000007'::uuid),
        ('ACT分组',            'b0c0d0e0-0001-4000-8000-000000000007'::uuid),
        ('REBOOT分组',         'b0c0d0e0-0001-4000-8000-000000000007'::uuid),
        ('RESET分组',          'b0c0d0e0-0001-4000-8000-000000000007'::uuid),
        ('RF分组',             'b0c0d0e0-0001-4000-8000-000000000007'::uuid),
        ('升级',               'b0c0d0e0-0001-4000-8000-000000000007'::uuid),
        ('启停设备',           'b0c0d0e0-0001-4000-8000-000000000007'::uuid),
        ('功率操作',           'b0c0d0e0-0001-4000-8000-000000000007'::uuid),
        ('抓包配置',           'b0c0d0e0-0001-4000-8000-000000000007'::uuid),
        -- 8. Electric Adjustment Configuration 电调天线（1 条）
        ('天线',               'b0c0d0e0-0001-4000-8000-000000000008'::uuid),
        -- 9. EU&RU Setting（2 条）
        ('EU',                 'b0c0d0e0-0001-4000-8000-000000000009'::uuid),
        ('RU',                 'b0c0d0e0-0001-4000-8000-000000000009'::uuid),
        -- 10. VSWR 驻波比（1 条）
        ('驻波检测',           'b0c0d0e0-0001-4000-8000-00000000000a'::uuid)
)
UPDATE mml_commands c
SET group_id = mapping.new_top_id
FROM mml_command_groups kw
JOIN mml_command_groups p
     ON p.path = subltree(kw.path, 0, 1)
     AND p.source = 'standard'
JOIN name_to_top mapping ON mapping.old_name = p.group_name_zh
WHERE c.group_id = kw.id
  AND kw.source = 'standard'
  AND nlevel(kw.path) = 2;

-- ---------- Step 4: 删除老 90 + 383 组（保留 10 新顶 + admin 自建） ----------
-- parent_id ON DELETE CASCADE：删 depth-1 时 depth-2 keyword 子组自动消失
-- 用 group_code NOT LIKE 'top_%' 区分新旧；source='standard' 保护 admin 自建组。
-- param_version='STANDARD' 守卫：只清理本迁移目标的 87-class 老结构，不波及
-- 其它 param_version（例如 seed/000152 注入的 'cmcc-td-lte-v2.3' 18 个
-- SA…SR 分组）。缺这守卫会在多次 migrate up 时把后续 catalog 一并删空
-- （fix 自 2026-05-22 dev DB 实测 18 SA-SR 全部被删事件）。
DELETE FROM mml_command_groups
WHERE source = 'standard'
  AND param_version = 'STANDARD'
  AND group_code NOT LIKE 'top\_%' ESCAPE '\';

-- +goose Down
-- 警告：down 不能精确恢复 90+383 老结构（映射有损）。
-- 仅删 10 新顶；若需还原完整旧 catalog，请重跑 seed/000111_mml_old_catalog_import.sql。
DELETE FROM mml_command_groups
WHERE group_code IN (
    'top_btsinfo', 'top_btssetting', 'top_network', 'top_system', 'top_lte',
    'top_special', 'top_maintenance', 'top_elecadj', 'top_euru', 'top_vswr'
)
  AND param_version = 'STANDARD';
