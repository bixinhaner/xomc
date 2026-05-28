-- +goose Up
-- ============================================================
-- 2026-05-27 修补 seed/000205:
--
-- 问题:
--   seed/000205 第一次跑时用的是 UUID 列表(包含 'aaaa0009-0000-0000-0000-000000000001'
--   即"旧 License 目录"UUID,但 seed/000123 已经把这行 DELETE 改成新 row
--   'aaaa0009-0000-0000-0000-000000000010' / name='License' / permission_key='system_license'),
--   所以 UPDATE 0 行 — License 始终没隐藏。
--
--   后续 000205 文件被改成 name 三重 OR 匹配,但 goose 按版本号跳过已 applied 的版本,
--   修订永远没真正在 DB 上跑。
--
-- 解法:
--   新增版本 000206,以 name + permission_key + route_path 三重 OR 兜底
--   重跑 Part 1(hide)+ Part 2(i18n backfill)。两者都幂等(IS DISTINCT FROM /
--   条件守护),已正确状态二次执行 0 行。
--
-- 目标 8 项:
--   拓扑设置 / 图例管理 / 设备规则 / 设备注册 / 即插即用 / License / 运维管理 / 脚本任务
-- ============================================================


-- ============================================================
-- Part 1: 兜底隐藏 8 个菜单(递归含子节点)
-- ============================================================
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
   SET show_status = 'hide',
       updated_at  = NOW()
 WHERE id IN (SELECT id FROM target_menus)
   AND show_status IS DISTINCT FROM 'hide';
-- +goose StatementEnd


-- ============================================================
-- Part 2: 兜底回填 name_i18n.en-US — 保留 zh-CN(若空兜底 name 字段)
-- 守护:仅当 en-US 缺失 / 空串 / 错填中文同名时才覆盖
-- ============================================================
-- +goose StatementBegin
UPDATE menus AS m
   SET name_i18n = jsonb_build_object(
        'zh-CN', COALESCE(m.name_i18n->>'zh-CN', m.name),
        'en-US', t.en_us
    )
  FROM (VALUES
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
      ('脚本任务',              'Script Tasks'),

      ('设备管理',              'Devices'),
      ('设备列表',              'Device List'),
      ('设备分组',              'Device Groups'),
      ('设备详情',              'Device Details'),
      ('回收站',                'Recycle Bin'),
      ('孤儿设备',              'Orphan Devices'),
      ('设备上报日志',          'Device Report Logs'),
      ('设备异常日志',          'Device Exception Logs'),

      ('拓扑管理',              'Topology'),
      ('拓扑图',                'Topology Map'),
      ('GIS地图',               'GIS Map'),

      ('告警管理',              'Alarms'),
      ('当前告警',              'Active Alarms'),
      ('历史告警',              'Historical Alarms'),
      ('告警库',                'Alarm Library'),
      ('告警规则',              'Alarm Rules'),
      ('自定义告警',            'Custom Alarms'),
      ('事件日志',              'Event Logs'),

      ('性能管理',              'Performance'),
      ('性能查询',              'Performance Query'),
      ('基站KPI',               'Station KPI'),
      ('标准KPI',               'Standard KPI'),
      ('KPI 指标库',            'KPI Indicator Library'),
      ('KPI指标详情',           'KPI Indicator Details'),
      ('UE详情',                'UE Details'),

      ('MML管理',               'MML Management'),
      ('MML控制台',             'MML Console'),
      ('任务记录',              'Task History'),

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

      ('仪表板',                'Dashboard'),
      ('站点管理',              'Sites'),

      ('软件管理',              'Software'),
      ('升级文件',              'Upgrade Files'),
      ('版本升级',              'Version Upgrade'),
      ('版本回退',              'Version Rollback'),
      ('配置文件',              'Config Files'),
      ('备份恢复',              'Backup & Restore'),
      ('备份任务',              'Backup Tasks'),
      ('数据恢复',              'Data Restore'),

      ('产品管理',              'Products'),
      ('参数模型',              'Param Models'),

      ('日志管理',              'Logs')
  ) AS t(zh_cn, en_us)
 WHERE m.name = t.zh_cn
   AND (
       m.name_i18n IS NULL
       OR (m.name_i18n->>'en-US') IS NULL
       OR (m.name_i18n->>'en-US') = ''
       OR (m.name_i18n->>'en-US') = m.name
   );
-- +goose StatementEnd


-- +goose Down
-- 反向仅恢复 hide;i18n 不动。
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
