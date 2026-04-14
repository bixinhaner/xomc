-- MML 模块增强：扩展 mml_tasks 调度字段 + 替换命令种子数据
-- 对应设计文档 mml-requirements-design.md 第 8.3 节

-- =============================================
-- A) mml_tasks 新增列（幂等）
-- =============================================

DO $$ BEGIN
-- 执行类型
IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'mml_tasks' AND column_name = 'execute_type') THEN
    ALTER TABLE mml_tasks ADD COLUMN execute_type VARCHAR(20) NOT NULL DEFAULT 'immediate';
END IF;

-- 定时执行时间
IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'mml_tasks' AND column_name = 'scheduled_at') THEN
    ALTER TABLE mml_tasks ADD COLUMN scheduled_at TIMESTAMPTZ;
END IF;

-- 周期任务时间窗口
IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'mml_tasks' AND column_name = 'period_start') THEN
    ALTER TABLE mml_tasks ADD COLUMN period_start TIMESTAMPTZ;
END IF;
IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'mml_tasks' AND column_name = 'period_end') THEN
    ALTER TABLE mml_tasks ADD COLUMN period_end TIMESTAMPTZ;
END IF;
IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'mml_tasks' AND column_name = 'period_time') THEN
    ALTER TABLE mml_tasks ADD COLUMN period_time VARCHAR(10);
END IF;

-- 离线重试
IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'mml_tasks' AND column_name = 'offline_retry') THEN
    ALTER TABLE mml_tasks ADD COLUMN offline_retry BOOLEAN DEFAULT false;
END IF;
IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'mml_tasks' AND column_name = 'offline_retry_wait') THEN
    ALTER TABLE mml_tasks ADD COLUMN offline_retry_wait INT DEFAULT 60;
END IF;

-- 失败重试
IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'mml_tasks' AND column_name = 'failed_retry') THEN
    ALTER TABLE mml_tasks ADD COLUMN failed_retry BOOLEAN DEFAULT false;
END IF;
IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'mml_tasks' AND column_name = 'failed_retry_count') THEN
    ALTER TABLE mml_tasks ADD COLUMN failed_retry_count INT DEFAULT 3;
END IF;
IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'mml_tasks' AND column_name = 'failed_retry_interval') THEN
    ALTER TABLE mml_tasks ADD COLUMN failed_retry_interval INT DEFAULT 5;
END IF;

-- 执行时间戳
IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'mml_tasks' AND column_name = 'started_at') THEN
    ALTER TABLE mml_tasks ADD COLUMN started_at TIMESTAMPTZ;
END IF;
IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'mml_tasks' AND column_name = 'finished_at') THEN
    ALTER TABLE mml_tasks ADD COLUMN finished_at TIMESTAMPTZ;
END IF;

-- 统计计数
IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'mml_tasks' AND column_name = 'total_devices') THEN
    ALTER TABLE mml_tasks ADD COLUMN total_devices INT DEFAULT 0;
END IF;
IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'mml_tasks' AND column_name = 'success_count') THEN
    ALTER TABLE mml_tasks ADD COLUMN success_count INT DEFAULT 0;
END IF;
IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'mml_tasks' AND column_name = 'failed_count') THEN
    ALTER TABLE mml_tasks ADD COLUMN failed_count INT DEFAULT 0;
END IF;

-- 任务结果
IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'mml_tasks' AND column_name = 'result') THEN
    ALTER TABLE mml_tasks ADD COLUMN result VARCHAR(20);
END IF;

END $$;

-- =============================================
-- B) 替换命令种子数据（11 条，来自设计文档 8.3 节）
-- =============================================

INSERT INTO mml_commands (
    id, command_name, command_code, category, description, rpc_method, param_template, product_types, created_at
) VALUES
(
    '00000000-0000-0000-0000-000000000001',
    '基本信息',
    'LST BASIC_INFO',
    '总览',
    '查询设备基本信息',
    'GetParameterValues',
    '{}'::jsonb,
    '["eNB", "gNB", "GSM"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0000-000000000002',
    '状态信息',
    'LST STATUS_INFO',
    '总览',
    '查询设备状态信息',
    'GetParameterValues',
    '{}'::jsonb,
    '["eNB", "gNB", "GSM"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0000-000000000003',
    '修改状态',
    'MOD STATUS_INFO',
    '总览',
    '修改设备状态信息配置',
    'SetParameterValues',
    '{"STATUS":{"type":"enum","required":true,"description":"状态","options":[{"label":"启用","value":1},{"label":"禁用","value":0}]}}'::jsonb,
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0000-000000000004',
    'eNB配置查询',
    'LST eNB_CONFIG',
    '快速设置',
    '查询eNB快速配置信息',
    'GetParameterValues',
    '{}'::jsonb,
    '["eNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0000-000000000005',
    'eNB配置修改',
    'MOD eNB_CONFIG',
    '快速设置',
    '修改eNB快速配置',
    'SetParameterValues',
    '{"FREQ":{"type":"number","required":true,"description":"频点","min_value":0,"max_value":65535},"PCI":{"type":"number","required":true,"description":"物理小区标识","min_value":0,"max_value":503},"PWR":{"type":"number","required":false,"description":"发射功率(dBm)","min_value":-30,"max_value":50}}'::jsonb,
    '["eNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0000-000000000006',
    '小区配置查询',
    'LST CELL',
    '快速设置',
    '查询小区配置信息',
    'GetParameterValues',
    '{"CELLID":{"type":"number","required":false,"description":"小区ID，不填则查询全部","min_value":0,"max_value":65535}}'::jsonb,
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0000-000000000007',
    '告警查询',
    'LST ALARM',
    '告警管理',
    '查询设备当前告警',
    'GetParameterValues',
    '{"ALARM_LEVEL":{"type":"enum","required":false,"description":"告警级别","options":[{"label":"紧急","value":1},{"label":"重要","value":2},{"label":"一般","value":3},{"label":"提示","value":4}]}}'::jsonb,
    '["eNB", "gNB", "GSM"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0000-000000000008',
    '告警清除',
    'CLR ALARM',
    '告警管理',
    '清除指定告警',
    'SetParameterValues',
    '{"ALARM_ID":{"type":"string","required":true,"description":"告警ID"}}'::jsonb,
    '["eNB", "gNB", "GSM"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0000-000000000009',
    '性能统计查询',
    'LST PM',
    '性能统计',
    '查询设备性能统计信息',
    'GetParameterValues',
    '{"START_TIME":{"type":"string","required":true,"description":"开始时间"},"END_TIME":{"type":"string","required":true,"description":"结束时间"}}'::jsonb,
    '["eNB", "gNB", "GSM"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0000-000000000010',
    '设备重启',
    'RST DEVICE',
    '设备控制',
    '重启指定设备',
    'Reboot',
    '{"DELAY":{"type":"number","required":false,"description":"延迟秒数","min_value":0,"max_value":3600}}'::jsonb,
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0000-000000000011',
    '软件版本查询',
    'LST VERSION',
    '设备控制',
    '查询设备软件版本',
    'GetParameterValues',
    '{}'::jsonb,
    '["eNB", "gNB", "GSM"]'::jsonb,
    NOW()
)
ON CONFLICT (command_code) DO UPDATE SET
    command_name  = EXCLUDED.command_name,
    category      = EXCLUDED.category,
    description   = EXCLUDED.description,
    rpc_method    = EXCLUDED.rpc_method,
    param_template = EXCLUDED.param_template,
    product_types = EXCLUDED.product_types;
