-- +goose Up
-- MML 模块增强：扩展 mml_tasks 调度字段 + 替换命令种子数据
-- 对应设计文档 mml-requirements-design.md 第 8.3 节

-- =============================================
-- A) mml_tasks 新增列（幂等）
-- =============================================

-- +goose StatementBegin
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
-- +goose StatementEnd

-- =============================================
-- B) 替换命令种子数据（25 条，按 7 个标准类别组织）
--    类别：小区管理、邻区管理、基站管理、告警查询、性能采集、传输管理、版本管理
-- =============================================

-- 先清理旧种子（000007 中的 LST_DEVPARAM/SET_DEVPARAM/RST_DEV 以及本文件之前的版本）
DELETE FROM mml_commands WHERE id::text LIKE '00000000-0000-0000-%';

INSERT INTO mml_commands (
    id, command_name, command_code, category, description, rpc_method, param_template, product_types, created_at
) VALUES
-- ---- 小区管理 ----
(
    '00000000-0000-0000-0001-000000000001', '查询小区信息', 'LST CELL', '小区管理',
    '列出当前基站所有小区的配置信息',
    'GetParameterValues',
    '{"CELLID":{"type":"number","required":false,"description":"小区ID（不填则查询所有）","min_value":0,"max_value":255}}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
),
(
    '00000000-0000-0000-0001-000000000002', '激活小区', 'ACT CELL', '小区管理',
    '激活指定小区使其开始提供服务',
    'SetParameterValues',
    '{"CELLID":{"type":"number","required":true,"description":"小区ID","min_value":0,"max_value":255}}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
),
(
    '00000000-0000-0000-0001-000000000003', '去激活小区', 'DEA CELL', '小区管理',
    '去激活指定小区停止服务',
    'SetParameterValues',
    '{"CELLID":{"type":"number","required":true,"description":"小区ID","min_value":0,"max_value":255}}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
),
(
    '00000000-0000-0000-0001-000000000004', '修改小区参数', 'MOD CELL', '小区管理',
    '修改指定小区的配置参数',
    'SetParameterValues',
    '{"CELLID":{"type":"number","required":true,"description":"小区ID","min_value":0,"max_value":255},"PCI":{"type":"number","required":false,"description":"物理小区标识","min_value":0,"max_value":503},"TXPOWER":{"type":"number","required":false,"description":"发射功率(dBm)","min_value":0,"max_value":50,"default_value":43}}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
),
(
    '00000000-0000-0000-0001-000000000005', '重置小区', 'RST CELL', '小区管理',
    '对指定小区执行重置操作',
    'SetParameterValues',
    '{"CELLID":{"type":"number","required":true,"description":"小区ID","min_value":0,"max_value":255}}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
),
-- ---- 邻区管理 ----
(
    '00000000-0000-0000-0002-000000000001', '查询邻区', 'LST NCELL', '邻区管理',
    '查询邻区配置关系',
    'GetParameterValues',
    '{"LOCALCELLID":{"type":"number","required":false,"description":"本地小区ID","min_value":0,"max_value":255}}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
),
(
    '00000000-0000-0000-0002-000000000002', '添加邻区', 'ADD NCELL', '邻区管理',
    '添加邻区关系',
    'SetParameterValues',
    '{"LOCALCELLID":{"type":"number","required":true,"description":"本地小区ID","min_value":0,"max_value":255},"NCELLID":{"type":"number","required":true,"description":"邻小区ID","min_value":0,"max_value":255}}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
),
(
    '00000000-0000-0000-0002-000000000003', '删除邻区', 'DEL NCELL', '邻区管理',
    '删除邻区关系',
    'SetParameterValues',
    '{"LOCALCELLID":{"type":"number","required":true,"description":"本地小区ID","min_value":0,"max_value":255},"NCELLID":{"type":"number","required":true,"description":"邻小区ID","min_value":0,"max_value":255}}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
),
-- ---- 基站管理 ----
(
    '00000000-0000-0000-0003-000000000001', '查询基站状态', 'LST BTSSTATE', '基站管理',
    '查询基站运行状态信息',
    'GetParameterValues',
    '{}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
),
(
    '00000000-0000-0000-0003-000000000002', '查询单板状态', 'DSP BOARDSTATUS', '基站管理',
    '显示所有单板的当前运行状态',
    'GetParameterValues',
    '{"SRN":{"type":"number","required":false,"description":"子框号","min_value":0,"max_value":15},"SN":{"type":"number","required":false,"description":"槽位号","min_value":0,"max_value":31}}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
),
(
    '00000000-0000-0000-0003-000000000003', '复位基站', 'RST BTS', '基站管理',
    '对基站执行复位操作',
    'Reboot',
    '{"RSTTYPE":{"type":"enum","required":true,"description":"复位类型","options":[{"label":"软复位","value":0},{"label":"硬复位","value":1}]}}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
),
(
    '00000000-0000-0000-0003-000000000004', '查询系统资源', 'DSP SYSRESOURCE', '基站管理',
    '显示系统CPU、内存等资源占用情况',
    'GetParameterValues',
    '{}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
),
(
    '00000000-0000-0000-0003-000000000005', '查询时钟状态', 'DSP CLOCKSTATUS', '基站管理',
    '显示系统时钟同步状态',
    'GetParameterValues',
    '{}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
),
-- ---- 告警查询 ----
(
    '00000000-0000-0000-0004-000000000001', '查询活动告警', 'LST ALMAF', '告警查询',
    '查询当前活动告警列表',
    'GetParameterValues',
    '{"ALMFAULTID":{"type":"number","required":false,"description":"告警ID"},"SEVERITY":{"type":"enum","required":false,"description":"告警级别","options":[{"label":"严重","value":1},{"label":"主要","value":2},{"label":"次要","value":3},{"label":"提示","value":4}]}}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
),
(
    '00000000-0000-0000-0004-000000000002', '查询历史告警', 'LST ALMHIS', '告警查询',
    '查询历史告警记录',
    'GetParameterValues',
    '{"START_TIME":{"type":"string","required":false,"description":"开始时间"},"END_TIME":{"type":"string","required":false,"description":"结束时间"}}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
),
(
    '00000000-0000-0000-0004-000000000003', '清除告警', 'CLR ALM', '告警查询',
    '手动清除指定告警',
    'SetParameterValues',
    '{"ALMID":{"type":"number","required":true,"description":"告警ID"}}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
),
-- ---- 性能采集 ----
(
    '00000000-0000-0000-0005-000000000001', '查询性能计数器', 'DSP PERF', '性能采集',
    '实时查询指定性能计数器值',
    'GetParameterValues',
    '{"COUNTER":{"type":"string","required":true,"description":"计数器名称"},"CELLID":{"type":"number","required":false,"description":"小区ID","min_value":0,"max_value":255}}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
),
(
    '00000000-0000-0000-0005-000000000002', '查询性能统计', 'LST PM', '性能采集',
    '查询设备性能统计信息',
    'GetParameterValues',
    '{"START_TIME":{"type":"string","required":true,"description":"开始时间"},"END_TIME":{"type":"string","required":true,"description":"结束时间"}}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
),
(
    '00000000-0000-0000-0005-000000000003', '查询RRU信息', 'DSP RRUINFO', '性能采集',
    '显示RRU单元的详细信息',
    'GetParameterValues',
    '{"RRUID":{"type":"number","required":false,"description":"RRU编号","min_value":0,"max_value":63}}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
),
-- ---- 传输管理 ----
(
    '00000000-0000-0000-0006-000000000001', '查询传输链路', 'DSP LINKSTATUS', '传输管理',
    '显示所有传输链路的当前状态',
    'GetParameterValues',
    '{"LINKTYPE":{"type":"enum","required":false,"description":"链路类型","options":[{"label":"S1","value":"S1"},{"label":"X2","value":"X2"},{"label":"NG","value":"NG"}]}}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
),
(
    '00000000-0000-0000-0006-000000000002', '查询SCTP链路', 'DSP SCTP', '传输管理',
    '显示SCTP传输链路状态',
    'GetParameterValues',
    '{"LNKID":{"type":"number","required":false,"description":"链路ID","min_value":0,"max_value":255}}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
),
(
    '00000000-0000-0000-0006-000000000003', '查询IP地址', 'LST IPADDR', '传输管理',
    '列出设备所有接口IP地址配置',
    'GetParameterValues',
    '{}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
),
-- ---- 版本管理 ----
(
    '00000000-0000-0000-0007-000000000001', '查询设备版本', 'DSP VERSION', '版本管理',
    '显示设备软件版本信息',
    'GetParameterValues',
    '{}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
),
(
    '00000000-0000-0000-0007-000000000002', '查询软件包', 'LST PKG', '版本管理',
    '查询可用软件包列表',
    'GetParameterValues',
    '{}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
),
(
    '00000000-0000-0000-0007-000000000003', '升级软件包', 'UPG PKG', '版本管理',
    '执行软件包升级',
    'Download',
    '{"PKGID":{"type":"string","required":true,"description":"软件包ID"},"MODE":{"type":"enum","required":false,"description":"升级模式","options":[{"label":"立即","value":"IMMEDIATE"},{"label":"延迟","value":"DELAYED"}]}}'::jsonb,
    '["eNB", "gNB"]'::jsonb, NOW()
)
ON CONFLICT (command_code) DO UPDATE SET
    command_name   = EXCLUDED.command_name,
    category       = EXCLUDED.category,
    description    = EXCLUDED.description,
    rpc_method     = EXCLUDED.rpc_method,
    param_template = EXCLUDED.param_template,
    product_types  = EXCLUDED.product_types;
