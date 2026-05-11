-- F06 运维管理 8 个新权限点 + 6 内置 OEM 模板
-- 来源 PRD: docs/project/prd/F06-ops-management.md §8.1 + §4.1.3
-- 推进计划: T-0112-c + T-0112-d

-- +goose Up
-- ============================================================
-- 1. 新增 8 个权限点 button menus（沿用 aaaa000a-1*00-*-X 命名空间）
-- ============================================================
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, status, show_status)
VALUES
    -- 命令风险分级（3 档）— 挂在 /ops/commands menu 下
    ('aaaa000a-1300-0000-0000-000000000003'::uuid, '安全命令', 'button', 'ops:command:safe',      'aaaa000a-1000-0000-0000-000000000003'::uuid, 3, '', 'normal', 'show'),
    ('aaaa000a-1300-0000-0000-000000000004'::uuid, '谨慎命令', 'button', 'ops:command:cautious',  'aaaa000a-1000-0000-0000-000000000003'::uuid, 4, '', 'normal', 'show'),
    ('aaaa000a-1300-0000-0000-000000000005'::uuid, '危险命令', 'button', 'ops:command:dangerous', 'aaaa000a-1000-0000-0000-000000000003'::uuid, 5, '', 'normal', 'show'),
    -- 任务审批
    ('aaaa000a-1200-0000-0000-000000000005'::uuid, '审批',    'button', 'ops:task:approve',       'aaaa000a-1000-0000-0000-000000000002'::uuid, 5, '', 'normal', 'show'),
    -- 诊断 / 下载发起
    ('aaaa000a-1400-0000-0000-000000000002'::uuid, '发起',    'button', 'ops:diagnostic:run',     'aaaa000a-1000-0000-0000-000000000004'::uuid, 2, '', 'normal', 'show'),
    ('aaaa000a-1500-0000-0000-000000000002'::uuid, '触发',    'button', 'ops:download:trigger',   'aaaa000a-1000-0000-0000-000000000005'::uuid, 2, '', 'normal', 'show'),
    -- 审计 / 紧急
    ('aaaa000a-1600-0000-0000-000000000001'::uuid, '审计查看', 'button', 'ops:audit:view',        'aaaa000a-0000-0000-0000-000000000001'::uuid, 6, '', 'normal', 'show'),
    ('aaaa000a-1700-0000-0000-000000000001'::uuid, '紧急权限', 'button', 'ops:break_glass',       'aaaa000a-0000-0000-0000-000000000001'::uuid, 7, '', 'normal', 'show')
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- 2. role_menus：admin 全部；operator 含 run/trigger + approve；viewer 仅 audit:view
-- ============================================================
INSERT INTO role_menus (role_id, menu_id)
SELECT '10000000-0000-0000-0000-000000000001'::uuid, id
FROM menus WHERE id IN (
    'aaaa000a-1300-0000-0000-000000000003'::uuid,
    'aaaa000a-1300-0000-0000-000000000004'::uuid,
    'aaaa000a-1300-0000-0000-000000000005'::uuid,
    'aaaa000a-1200-0000-0000-000000000005'::uuid,
    'aaaa000a-1400-0000-0000-000000000002'::uuid,
    'aaaa000a-1500-0000-0000-000000000002'::uuid,
    'aaaa000a-1600-0000-0000-000000000001'::uuid,
    'aaaa000a-1700-0000-0000-000000000001'::uuid
)
ON CONFLICT (role_id, menu_id) DO NOTHING;

INSERT INTO role_menus (role_id, menu_id)
SELECT '10000000-0000-0000-0000-000000000002'::uuid, id
FROM menus WHERE id IN (
    'aaaa000a-1300-0000-0000-000000000003'::uuid,  -- safe
    'aaaa000a-1300-0000-0000-000000000004'::uuid,  -- cautious
    'aaaa000a-1400-0000-0000-000000000002'::uuid,  -- diag run
    'aaaa000a-1500-0000-0000-000000000002'::uuid,  -- download trigger
    'aaaa000a-1600-0000-0000-000000000001'::uuid   -- audit view
)
ON CONFLICT (role_id, menu_id) DO NOTHING;

INSERT INTO role_menus (role_id, menu_id)
SELECT '10000000-0000-0000-0000-000000000003'::uuid,
       'aaaa000a-1600-0000-0000-000000000001'::uuid
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- ============================================================
-- 3. 6 个内置 OEM 模板（PRD §4.1.3）
-- ============================================================
INSERT INTO ops_templates (
    id, template_name, description, category, target_device_types, steps,
    estimated_duration, creator, use_count, tags, risk_level, version, target_carriers
) VALUES
    ('aaaa000a-2000-0000-0000-000000000001'::uuid,
     'GPS 失锁恢复', '检查 GPS 卫星数 → 重启 GPS 模块 → 等待 60s → 再次检查',
     '故障处置', '["eNB","gNB"]'::jsonb,
     '[{"step_no":1,"step_name":"检查 GPS 卫星数","step_type":"check","description":"GPV Device.GPS.SatelliteCount"},
       {"step_no":2,"step_name":"重启 GPS 模块","step_type":"mml","command":"RST GPS","description":"软重启 GPS"},
       {"step_no":3,"step_name":"等待恢复","step_type":"wait","wait_seconds":60,"description":"等待 GPS 重新锁定"},
       {"step_no":4,"step_name":"复检","step_type":"check","description":"确认 SatelliteCount ≥ 4"}]'::jsonb,
     120, 'system', 0, '["GPS","故障","常用"]'::jsonb, 'cautious', 1, '[]'::jsonb),

    ('aaaa000a-2000-0000-0000-000000000002'::uuid,
     '时间同步对齐', '取当前时间 → 强制同步 → 验证',
     '巡检运维', '["eNB","gNB","CPE"]'::jsonb,
     '[{"step_no":1,"step_name":"取当前时间","step_type":"check","description":"GPV Device.Time.CurrentLocalTime"},
       {"step_no":2,"step_name":"强制同步","step_type":"mml","command":"SET NTP","description":"触发 NTP 同步"},
       {"step_no":3,"step_name":"验证偏差","step_type":"check","description":"对比 OMC 时钟，偏差 < 1s"}]'::jsonb,
     60, 'system', 0, '["NTP","时钟","巡检"]'::jsonb, 'safe', 1, '[]'::jsonb),

    ('aaaa000a-2000-0000-0000-000000000003'::uuid,
     '计划重启', '备份配置 → 重启 → 等待 inform → 校验状态',
     '维护操作', '["eNB","gNB"]'::jsonb,
     '[{"step_no":1,"step_name":"备份配置","step_type":"script","description":"Upload Vendor Config"},
       {"step_no":2,"step_name":"软重启","step_type":"mml","command":"REBOOT","description":"发起 reboot RPC"},
       {"step_no":3,"step_name":"等待 inform","step_type":"wait","wait_seconds":180,"description":"等设备上线 inform"},
       {"step_no":4,"step_name":"状态校验","step_type":"check","description":"确认状态 = online"}]'::jsonb,
     300, 'system', 0, '["重启","维护"]'::jsonb, 'cautious', 1, '[]'::jsonb),

    ('aaaa000a-2000-0000-0000-000000000004'::uuid,
     '设备隔离', '关闭小区 → 通知 EMS 摘流量 → 告警抑制',
     '故障处置', '["eNB","gNB"]'::jsonb,
     '[{"step_no":1,"step_name":"关闭小区","step_type":"mml","command":"DEACT CELL","description":"AdminState=Locked"},
       {"step_no":2,"step_name":"通知 EMS","step_type":"notify","notify_target":"ems","description":"摘流量"},
       {"step_no":3,"step_name":"告警抑制","step_type":"script","description":"30 分钟维护窗口"}]'::jsonb,
     180, 'system', 0, '["紧急","隔离"]'::jsonb, 'dangerous', 1, '[]'::jsonb),

    ('aaaa000a-2000-0000-0000-000000000005'::uuid,
     '现场协助包', '取配置 + 取最近 1h 日志 + 取 GPS + 取邻区 + IPPing 测试',
     '故障处置', '["eNB","gNB","CPE"]'::jsonb,
     '[{"step_no":1,"step_name":"取当前配置","step_type":"script","description":"Upload Config File"},
       {"step_no":2,"step_name":"取近 1h 日志","step_type":"script","description":"Upload Log File"},
       {"step_no":3,"step_name":"取 GPS 状态","step_type":"check","description":"GPV GPS.*"},
       {"step_no":4,"step_name":"取邻区","step_type":"check","description":"GPV NeighborList.*"},
       {"step_no":5,"step_name":"IPPing 测试","step_type":"script","description":"IPPingDiagnostics"}]'::jsonb,
     240, 'system', 0, '["调试","支持","常用"]'::jsonb, 'safe', 1, '[]'::jsonb),

    ('aaaa000a-2000-0000-0000-000000000006'::uuid,
     '软重启 SDR', '取告警 → 取 RF 参数 → 软重启 SDR → 验证',
     '故障处置', '["gNB"]'::jsonb,
     '[{"step_no":1,"step_name":"取告警","step_type":"check","description":"列出活跃告警"},
       {"step_no":2,"step_name":"取 RF 参数","step_type":"check","description":"GPV RF.*"},
       {"step_no":3,"step_name":"软重启 SDR","step_type":"mml","command":"RST SDR","description":"SDR 模块软重启"},
       {"step_no":4,"step_name":"验证","step_type":"check","wait_seconds":30,"description":"小区状态恢复"}]'::jsonb,
     180, 'system', 0, '["RF","SDR","5G"]'::jsonb, 'cautious', 1, '[]'::jsonb)
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM ops_templates WHERE id::text LIKE 'aaaa000a-2000-%';
DELETE FROM role_menus WHERE menu_id::text LIKE 'aaaa000a-13%' OR menu_id::text LIKE 'aaaa000a-14%'
    OR menu_id::text LIKE 'aaaa000a-15%' OR menu_id::text LIKE 'aaaa000a-16%' OR menu_id::text LIKE 'aaaa000a-17%';
DELETE FROM menus WHERE id::text LIKE 'aaaa000a-13%' OR id::text LIKE 'aaaa000a-14%'
    OR id::text LIKE 'aaaa000a-15%' OR id::text LIKE 'aaaa000a-16%' OR id::text LIKE 'aaaa000a-17%';
