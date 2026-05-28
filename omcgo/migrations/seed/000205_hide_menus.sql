-- +goose Up
-- ============================================================
-- 2026-05-27 用户决策:
--   Part 1: 8 个菜单(及其全部子节点)统一设为 show_status='hide'
--   Part 2: 回填菜单 name_i18n.en-US(运维 UI 切英文 locale 时不再为空)
--
-- 目标 8 项:
--   1. 拓扑设置        — topology:settings
--   2. 图例管理        — topology:legend
--   3. 设备规则        — device:rules
--   4. 设备注册        — device:register
--   5. 即插即用        — device:plug-and-play
--   6. License        — system_license(seed/000123 重建后);老环境可能仍是 '许可证管理' / 'license'
--   7. 运维管理 目录    — ops + 全部子菜单
--   8. 脚本任务        — mml:script(MML 管理下二级菜单)
--
-- 跨环境匹配策略:
--   不同环境的 menus.name 可能因历史 seed / UI 手改而不同(尤其 License 在
--   seed/000123 前 name='许可证管理' / permission_key='license',之后改为
--   name='License' / permission_key='system_license')。
--   → name + permission_key + route_path 三重 OR 命中,任一匹配即纳入,确保:
--     · 新环境(完整跑过 seed/000123)命中 name='License'
--     · 老环境(未跑 000123 残留旧 row)命中 name='许可证管理' / permission_key='license'
--     · 部分环境运维改名后仍可通过 permission_key / route_path 兜底
-- ============================================================


-- ============================================================
-- Part 1: 隐藏 7 个菜单(递归含子节点)
-- ============================================================
-- +goose StatementBegin
WITH RECURSIVE target_menus AS (
    -- 根节点:name / permission_key / route_path 三重 OR 命中
    SELECT id FROM menus
     WHERE name IN (
            '拓扑设置', '图例管理', '设备规则', '设备注册',
            '即插即用', 'License', '许可证管理', '运维管理',
            '脚本任务'
        )
        OR permission_key IN (
            'topology:settings', 'topology:legend',
            'device:rules', 'device:register', 'device:plug-and-play',
            'system_license', 'license',
            'ops',
            'mml:script'
        )
        OR route_path IN (
            '/topology/settings', '/topology/legend',
            '/device/rules', '/device/register', '/device/plug-and-play',
            '/license', '/ops',
            '/mml/script'
        )
    UNION ALL
    -- 递归向下:把所有子节点(子页 + button)一并纳入
    SELECT m.id FROM menus m
    JOIN target_menus t ON m.parent_id = t.id
)
UPDATE menus
   SET show_status = 'hide',
       updated_at  = NOW()
 WHERE id IN (SELECT id FROM target_menus)
   AND show_status IS DISTINCT FROM 'hide';
-- +goose StatementEnd


-- ============================================================
-- Part 2: 回填菜单 name_i18n.en-US
--
-- 规则:
--   · 按 menus.name(中文)匹配,设置对应英文 — 全集涵盖现有所有 menu / directory
--     节点常见命名(button 类如 查询/添加/修改等已在 seed/000084 完成,本迁移不重复)
--   · 仅当 name_i18n 为 NULL / en-US 为空字符串 / en-US 错填为中文(== name)时
--     才覆盖,保留运维通过 UI 手动配置的合理英文
--   · zh-CN 维度同步保证存在(COALESCE 兜底 name 字段)
--
-- 本迁移最终目的:运维切 en-US locale 时,所有菜单都能正确显示英文标签;
-- 旧 seed/000084 仅覆盖了部分 UUID,后续菜单(seed/000087/000091/...)未顾及。
-- ============================================================
-- +goose StatementBegin
UPDATE menus AS m
   SET name_i18n = jsonb_build_object(
        'zh-CN', COALESCE(m.name_i18n->>'zh-CN', m.name),
        'en-US', t.en_us
    )
  FROM (VALUES
      -- 隐藏 7 项(也补 i18n,避免日后取消隐藏后还得修)
      ('拓扑设置',              'Topology Settings'),
      ('图例管理',              'Legend Management'),
      ('设备规则',              'Device Rules'),
      ('设备注册',              'Device Registration'),
      ('即插即用',              'Plug and Play'),
      ('即插即用策略编辑',      'Plug-and-Play Policy Editor'),
      ('License',              'License'),
      ('许可证管理',            'License'),
      ('许可证列表',            'License List'),
      ('许可证操作',            'License Operations'),
      ('运维管理',              'Operations'),
      ('运维模板',              'Operations Templates'),
      ('运维任务',              'Operations Tasks'),
      ('运维命令',              'Operations Commands'),
      ('运维下载',              'Operations Downloads'),
      ('网络诊断',              'Network Diagnostics'),

      -- 设备管理子树
      ('设备管理',              'Devices'),
      ('设备列表',              'Device List'),
      ('设备分组',              'Device Groups'),
      ('设备详情',              'Device Details'),
      ('回收站',                'Recycle Bin'),
      ('孤儿设备',              'Orphan Devices'),
      ('设备上报日志',          'Device Report Logs'),
      ('设备异常日志',          'Device Exception Logs'),

      -- 拓扑管理
      ('拓扑管理',              'Topology'),
      ('拓扑图',                'Topology Map'),
      ('GIS地图',               'GIS Map'),

      -- 告警管理
      ('告警管理',              'Alarms'),
      ('当前告警',              'Active Alarms'),
      ('历史告警',              'Historical Alarms'),
      ('告警库',                'Alarm Library'),
      ('告警规则',              'Alarm Rules'),
      ('自定义告警',            'Custom Alarms'),
      ('事件日志',              'Event Logs'),

      -- 性能管理
      ('性能管理',              'Performance'),
      ('性能查询',              'Performance Query'),
      ('基站KPI',               'Station KPI'),
      ('标准KPI',               'Standard KPI'),
      ('KPI 指标库',            'KPI Indicator Library'),
      ('KPI指标详情',           'KPI Indicator Details'),
      ('UE详情',                'UE Details'),

      -- MML 管理
      ('MML管理',               'MML Management'),
      ('MML控制台',             'MML Console'),
      ('任务记录',              'Task History'),
      ('脚本任务',              'Script Tasks'),

      -- 系统管理
      ('系统管理',              'System'),
      ('系统配置',              'System Config'),
      ('用户管理',              'Users'),
      ('角色管理',              'Roles'),
      ('菜单管理',              'Menus'),
      ('API管理',               'API Management'),
      ('字典管理',              'Dictionary'),
      ('UI定制化',              'UI Customization'),
      ('操作日志',              'Operation Logs'),
      ('域管理',                'Domain Management'),
      ('审计日志',              'Audit Logs'),

      -- 仪表板 / 站点
      ('仪表板',                'Dashboard'),
      ('站点管理',              'Sites'),

      -- 软件 / 备份
      ('软件管理',              'Software'),
      ('升级文件',              'Upgrade Files'),
      ('版本升级',              'Version Upgrade'),
      ('版本回退',              'Version Rollback'),
      ('配置文件',              'Config Files'),
      ('备份恢复',              'Backup & Restore'),
      ('备份任务',              'Backup Tasks'),
      ('数据恢复',              'Data Restore'),

      -- 产品 / 参数模型
      ('产品管理',              'Products'),
      ('参数模型',              'Param Models'),

      -- 日志
      ('日志管理',              'Logs')
  ) AS t(zh_cn, en_us)
 WHERE m.name = t.zh_cn
   AND (
       m.name_i18n IS NULL
       OR (m.name_i18n->>'en-US') IS NULL
       OR (m.name_i18n->>'en-US') = ''
       OR (m.name_i18n->>'en-US') = m.name  -- 错填为中文同名
   );
-- +goose StatementEnd


-- +goose Down
-- 反向:仅恢复隐藏(Part 1);i18n 不做反向 — 回填本就是修正空白,Down 不该清掉。
-- +goose StatementBegin
WITH RECURSIVE target_menus AS (
    SELECT id FROM menus
     WHERE name IN (
            '拓扑设置', '图例管理', '设备规则', '设备注册',
            '即插即用', 'License', '许可证管理', '运维管理',
            '脚本任务'
        )
        OR permission_key IN (
            'topology:settings', 'topology:legend',
            'device:rules', 'device:register', 'device:plug-and-play',
            'system_license', 'license',
            'ops',
            'mml:script'
        )
        OR route_path IN (
            '/topology/settings', '/topology/legend',
            '/device/rules', '/device/register', '/device/plug-and-play',
            '/license', '/ops',
            '/mml/script'
        )
    UNION ALL
    SELECT m.id FROM menus m
    JOIN target_menus t ON m.parent_id = t.id
)
UPDATE menus
   SET show_status = 'show',
       updated_at  = NOW()
 WHERE id IN (SELECT id FROM target_menus);
-- +goose StatementEnd
