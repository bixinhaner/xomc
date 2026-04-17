-- +goose Up
-- MML 模块增强：扩展 mml_tasks 调度字段 + 增强 mml_commands 元数据
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
-- B) mml_commands 新增元数据列（幂等）
-- =============================================

-- +goose StatementBegin
DO $$ BEGIN
IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'mml_commands' AND column_name = 'operation_type') THEN
    ALTER TABLE mml_commands ADD COLUMN operation_type TEXT DEFAULT 'LST';
END IF;
IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'mml_commands' AND column_name = 'param_paths') THEN
    ALTER TABLE mml_commands ADD COLUMN param_paths JSONB DEFAULT '[]';
END IF;
IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'mml_commands' AND column_name = 'supported_operations') THEN
    ALTER TABLE mml_commands ADD COLUMN supported_operations JSONB DEFAULT '["LST"]';
END IF;
-- Ensure supported_operations is JSONB (may be TEXT[] from older seed)
IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'mml_commands' AND column_name = 'supported_operations' AND udt_name = '_text') THEN
    ALTER TABLE mml_commands ALTER COLUMN supported_operations SET DATA TYPE JSONB USING to_jsonb(supported_operations);
END IF;
IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'mml_commands' AND column_name = 'help_doc') THEN
    ALTER TABLE mml_commands ADD COLUMN help_doc TEXT DEFAULT '';
END IF;
IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'mml_commands' AND column_name = 'notes') THEN
    ALTER TABLE mml_commands ADD COLUMN notes TEXT DEFAULT '';
END IF;
END $$;
-- +goose StatementEnd

-- =============================================
-- C) 更新命令种子数据（25 条，按 7 个标准类别组织）
--    类别：小区管理、邻区管理、基站管理、告警查询、性能采集、传输管理、版本管理
-- =============================================

-- 清理旧版基础种子
DELETE FROM mml_commands WHERE command_code IN ('LST_DEVPARAM', 'SET_DEVPARAM', 'RST_DEV');

INSERT INTO mml_commands (
    id,
    command_name,
    command_code,
    category,
    description,
    rpc_method,
    operation_type,
    param_template,
    param_paths,
    supported_operations,
    help_doc,
    notes,
    product_types,
    created_at
) VALUES
-- ---- 小区管理 ----
(
    '00000000-0000-0000-0001-000000000001',
    '查询小区信息',
    'LST CELL',
    '1',
    '列出当前基站所有小区的配置信息',
    'GetParameterValues',
    'LST',
    $json${"CELLID":{"type":"number","required":false,"description":"小区ID（不填则查询所有）","min_value":0,"max_value":255,"default_value":0,"suggested_value":1,"unit":"","restart_required":false,"help_text":"小区标识，取值范围0-255；留空时返回所有小区。","order":1}}$json$::jsonb,
    $json$["Device.Services.FAPService.{i}.CellConfig.{i}"]$json$::jsonb,
    '["LST"]'::jsonb,
    '查询小区基础配置、射频及运行状态参数。',
    '适用于小区清单查询；不传 CELLID 时按设备返回全部小区。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0001-000000000002',
    '激活小区',
    'ACT CELL',
    '1',
    '激活指定小区使其开始提供服务',
    'SetParameterValues',
    'ACT',
    $json${"CELLID":{"type":"number","required":true,"description":"小区ID","min_value":0,"max_value":255,"default_value":0,"suggested_value":1,"unit":"","restart_required":false,"help_text":"指定需要激活的小区标识。","order":1}}$json$::jsonb,
    $json$["Device.Services.FAPService.{i}.CellConfig.{i}.LTE.RAN.CellState"]$json$::jsonb,
    '["ACT"]'::jsonb,
    '将目标小区状态切换为激活态。',
    '执行前建议确认小区参数已配置完成且处于可用状态。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0001-000000000003',
    '去激活小区',
    'DEA CELL',
    '1',
    '去激活指定小区停止服务',
    'SetParameterValues',
    'DEA',
    $json${"CELLID":{"type":"number","required":true,"description":"小区ID","min_value":0,"max_value":255,"default_value":0,"suggested_value":1,"unit":"","restart_required":false,"help_text":"指定需要去激活的小区标识。","order":1}}$json$::jsonb,
    $json$["Device.Services.FAPService.{i}.CellConfig.{i}.LTE.RAN.CellState"]$json$::jsonb,
    '["DEA"]'::jsonb,
    '将目标小区状态切换为去激活态。',
    '去激活会影响业务承载，建议在维护窗口内执行。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0001-000000000004',
    '修改小区参数',
    'MOD CELL',
    '1',
    '修改指定小区的配置参数',
    'SetParameterValues',
    'MOD',
    $json${"CELLID":{"type":"number","required":true,"description":"小区ID","min_value":0,"max_value":255,"default_value":0,"suggested_value":1,"unit":"","restart_required":false,"help_text":"指定要修改的小区标识。","order":1},"PCI":{"type":"number","required":false,"description":"物理小区标识","min_value":0,"max_value":503,"default_value":0,"suggested_value":100,"unit":"","restart_required":false,"help_text":"PCI 取值范围0-503，需避免邻区冲突。","order":2},"TXPOWER":{"type":"number","required":false,"description":"发射功率","min_value":0,"max_value":50,"default_value":43,"suggested_value":43,"unit":"dBm","restart_required":true,"help_text":"修改发射功率后通常需要小区重建或重启生效。","order":3}}$json$::jsonb,
    $json$["Device.Services.FAPService.{i}.CellConfig.{i}.LTE.RAN.RF"]$json$::jsonb,
    '["LST","MOD"]'::jsonb,
    '修改小区射频及基础配置参数。',
    '建议先通过 LST CELL 获取当前参数，再执行变更。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0001-000000000005',
    '重置小区',
    'RST CELL',
    '1',
    '对指定小区执行重置操作',
    'SetParameterValues',
    'RST',
    $json${"CELLID":{"type":"number","required":true,"description":"小区ID","min_value":0,"max_value":255,"default_value":0,"suggested_value":1,"unit":"","restart_required":false,"help_text":"指定执行重置操作的小区标识。","order":1}}$json$::jsonb,
    $json$["Device.Services.FAPService.{i}.CellConfig.{i}.LTE.RAN.CellState"]$json$::jsonb,
    '["RST"]'::jsonb,
    '触发目标小区执行逻辑复位。',
    '重置期间小区业务会短暂中断。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
-- ---- 邻区管理 ----
(
    '00000000-0000-0000-0002-000000000001',
    '查询邻区',
    'LST NCELL',
    '2',
    '查询邻区配置关系',
    'GetParameterValues',
    'LST',
    $json${"LOCALCELLID":{"type":"number","required":false,"description":"本地小区ID","min_value":0,"max_value":255,"default_value":0,"suggested_value":1,"unit":"","restart_required":false,"help_text":"过滤指定本地小区的邻区关系；留空时查询全部。","order":1}}$json$::jsonb,
    $json$["Device.Services.FAPService.{i}.CellConfig.{i}.NeighborList.{i}"]$json$::jsonb,
    '["LST"]'::jsonb,
    '查询小区间邻区关系与切换配置。',
    '通常用于核查邻区规划与自动邻区维护结果。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0002-000000000002',
    '添加邻区',
    'ADD NCELL',
    '2',
    '添加邻区关系',
    'SetParameterValues',
    'ADD',
    $json${"LOCALCELLID":{"type":"number","required":true,"description":"本地小区ID","min_value":0,"max_value":255,"default_value":0,"suggested_value":1,"unit":"","restart_required":false,"help_text":"需要增加邻区关系的源小区。","order":1},"NCELLID":{"type":"number","required":true,"description":"邻小区ID","min_value":0,"max_value":255,"default_value":0,"suggested_value":2,"unit":"","restart_required":false,"help_text":"新增的目标邻区标识。","order":2}}$json$::jsonb,
    $json$["Device.Services.FAPService.{i}.CellConfig.{i}.NeighborList.{i}"]$json$::jsonb,
    '["ADD"]'::jsonb,
    '为指定本地小区新增邻区关系。',
    '新增前建议确认邻区频点与 PCI 规划无冲突。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0002-000000000003',
    '删除邻区',
    'DEL NCELL',
    '2',
    '删除邻区关系',
    'SetParameterValues',
    'RMV',
    $json${"LOCALCELLID":{"type":"number","required":true,"description":"本地小区ID","min_value":0,"max_value":255,"default_value":0,"suggested_value":1,"unit":"","restart_required":false,"help_text":"需要删除邻区关系的源小区。","order":1},"NCELLID":{"type":"number","required":true,"description":"邻小区ID","min_value":0,"max_value":255,"default_value":0,"suggested_value":2,"unit":"","restart_required":false,"help_text":"待删除的目标邻区标识。","order":2}}$json$::jsonb,
    $json$["Device.Services.FAPService.{i}.CellConfig.{i}.NeighborList.{i}"]$json$::jsonb,
    '["RMV"]'::jsonb,
    '删除指定本地小区与邻小区之间的关系。',
    '删除后可能影响切换成功率，建议先评估影响范围。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
-- ---- 基站管理 ----
(
    '00000000-0000-0000-0003-000000000001',
    '查询基站状态',
    'LST BTSSTATE',
    '3',
    '查询基站运行状态信息',
    'GetParameterValues',
    'LST',
    '{}'::jsonb,
    $json$["Device.DeviceInfo"]$json$::jsonb,
    '["LST"]'::jsonb,
    '查询基站整体运行态、设备信息与在线状态。',
    '适合用作远程巡检的基础命令。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0003-000000000002',
    '查询单板状态',
    'DSP BOARDSTATUS',
    '3',
    '显示所有单板的当前运行状态',
    'GetParameterValues',
    'DSP',
    $json${"SRN":{"type":"number","required":false,"description":"子框号","min_value":0,"max_value":15,"default_value":0,"suggested_value":0,"unit":"","restart_required":false,"help_text":"用于过滤指定子框。","order":1},"SN":{"type":"number","required":false,"description":"槽位号","min_value":0,"max_value":31,"default_value":0,"suggested_value":1,"unit":"","restart_required":false,"help_text":"用于过滤指定槽位。","order":2}}$json$::jsonb,
    $json$["Device.Services.FAPService.{i}.Board.{i}"]$json$::jsonb,
    '["DSP"]'::jsonb,
    '展示单板在位、运行与告警状态。',
    'SRN 与 SN 可联合过滤指定板卡。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0003-000000000003',
    '复位基站',
    'RST BTS',
    '3',
    '对基站执行复位操作',
    'Reboot',
    'RST',
    $json${"RSTTYPE":{"type":"enum","required":true,"description":"复位类型","min_value":0,"max_value":1,"default_value":0,"suggested_value":0,"unit":"","restart_required":true,"help_text":"0 表示软复位，1 表示硬复位。","order":1,"options":[{"label":"软复位","value":0},{"label":"硬复位","value":1}]}}$json$::jsonb,
    $json$["Device.DeviceInfo.Reboot"]$json$::jsonb,
    '["RST"]'::jsonb,
    '触发整机级别复位动作。',
    '高风险操作，执行前应确认维护窗口并通知相关人员。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0003-000000000004',
    '查询系统资源',
    'DSP SYSRESOURCE',
    '3',
    '显示系统CPU、内存等资源占用情况',
    'GetParameterValues',
    'DSP',
    '{}'::jsonb,
    $json$["Device.Services.FAPService.{i}.System.Resource"]$json$::jsonb,
    '["DSP"]'::jsonb,
    '展示系统 CPU、内存、磁盘等资源使用情况。',
    '建议与性能类命令结合，用于定位资源瓶颈。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0003-000000000005',
    '查询时钟状态',
    'DSP CLOCKSTATUS',
    '3',
    '显示系统时钟同步状态',
    'GetParameterValues',
    'DSP',
    '{}'::jsonb,
    $json$["Device.Time", "Device.Services.FAPService.{i}.Sync"]$json$::jsonb,
    '["DSP"]'::jsonb,
    '展示系统时间、授时源与同步锁定状态。',
    '适合排查 GPS、1588、SyncE 等时钟问题。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
-- ---- 告警查询 ----
(
    '00000000-0000-0000-0004-000000000001',
    '查询活动告警',
    'LST ALMAF',
    '4',
    '查询当前活动告警列表',
    'GetParameterValues',
    'LST',
    $json${"ALMFAULTID":{"type":"number","required":false,"description":"告警ID","min_value":0,"max_value":999999,"default_value":0,"suggested_value":1001,"unit":"","restart_required":false,"help_text":"可按告警ID过滤活动告警。","order":1},"SEVERITY":{"type":"enum","required":false,"description":"告警级别","min_value":1,"max_value":4,"default_value":1,"suggested_value":1,"unit":"","restart_required":false,"help_text":"1严重、2主要、3次要、4提示。","order":2,"options":[{"label":"严重","value":1},{"label":"主要","value":2},{"label":"次要","value":3},{"label":"提示","value":4}]}}$json$::jsonb,
    $json$["Device.Services.FAPService.{i}.FaultMgmt.CurrentAlarm.{i}"]$json$::jsonb,
    '["LST"]'::jsonb,
    '查询当前仍处于激活态的告警。',
    '可结合 SEVERITY 快速过滤高优先级告警。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0004-000000000002',
    '查询历史告警',
    'LST ALMHIS',
    '4',
    '查询历史告警记录',
    'GetParameterValues',
    'LST',
    $json${"START_TIME":{"type":"string","required":false,"description":"开始时间","min_value":0,"max_value":0,"default_value":"","suggested_value":"2026-01-01 00:00:00","unit":"datetime","restart_required":false,"help_text":"推荐使用 YYYY-MM-DD HH:MM:SS 格式。","order":1},"END_TIME":{"type":"string","required":false,"description":"结束时间","min_value":0,"max_value":0,"default_value":"","suggested_value":"2026-01-01 23:59:59","unit":"datetime","restart_required":false,"help_text":"结束时间应大于开始时间。","order":2}}$json$::jsonb,
    $json$["Device.Services.FAPService.{i}.FaultMgmt.HistoryAlarm.{i}"]$json$::jsonb,
    '["LST"]'::jsonb,
    '按时间范围查询历史告警记录。',
    '若不传时间范围，设备侧可能返回默认窗口内的历史告警。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0004-000000000003',
    '清除告警',
    'CLR ALM',
    '4',
    '手动清除指定告警',
    'SetParameterValues',
    'CLR',
    $json${"ALMID":{"type":"number","required":true,"description":"告警ID","min_value":0,"max_value":999999,"default_value":0,"suggested_value":1001,"unit":"","restart_required":false,"help_text":"指定需要手动清除的告警ID。","order":1}}$json$::jsonb,
    $json$["Device.Services.FAPService.{i}.FaultMgmt.CurrentAlarm.{i}"]$json$::jsonb,
    '["CLR"]'::jsonb,
    '对支持手动确认或清除的告警执行清理动作。',
    '建议先核实根因已消除，否则告警可能再次上报。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
-- ---- 性能采集 ----
(
    '00000000-0000-0000-0005-000000000001',
    '查询性能计数器',
    'DSP PERF',
    '5',
    '实时查询指定性能计数器值',
    'GetParameterValues',
    'DSP',
    $json${"COUNTER":{"type":"string","required":true,"description":"计数器名称","min_value":0,"max_value":0,"default_value":"","suggested_value":"PRB.Usage.DL","unit":"","restart_required":false,"help_text":"请输入设备支持的性能计数器名称。","order":1},"CELLID":{"type":"number","required":false,"description":"小区ID","min_value":0,"max_value":255,"default_value":0,"suggested_value":1,"unit":"","restart_required":false,"help_text":"可按小区粒度过滤计数器数据。","order":2}}$json$::jsonb,
    $json$["Device.Services.FAPService.{i}.Performance.Counter.{i}"]$json$::jsonb,
    '["DSP"]'::jsonb,
    '实时读取指定性能计数器值。',
    '适用于现场快速排障，不替代周期性 PM 采集。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0005-000000000002',
    '查询性能统计',
    'LST PM',
    '5',
    '查询设备性能统计信息',
    'GetParameterValues',
    'LST',
    $json${"START_TIME":{"type":"string","required":true,"description":"开始时间","min_value":0,"max_value":0,"default_value":"","suggested_value":"2026-01-01 00:00:00","unit":"datetime","restart_required":false,"help_text":"性能查询开始时间，推荐使用 YYYY-MM-DD HH:MM:SS。","order":1},"END_TIME":{"type":"string","required":true,"description":"结束时间","min_value":0,"max_value":0,"default_value":"","suggested_value":"2026-01-01 01:00:00","unit":"datetime","restart_required":false,"help_text":"性能查询结束时间，必须大于开始时间。","order":2}}$json$::jsonb,
    $json$["Device.Services.FAPService.{i}.Performance.History.{i}"]$json$::jsonb,
    '["LST"]'::jsonb,
    '按时间窗口查询周期性性能统计结果。',
    '适合对接报表与趋势分析场景。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0005-000000000003',
    '查询RRU信息',
    'DSP RRUINFO',
    '5',
    '显示RRU单元的详细信息',
    'GetParameterValues',
    'DSP',
    $json${"RRUID":{"type":"number","required":false,"description":"RRU编号","min_value":0,"max_value":63,"default_value":0,"suggested_value":1,"unit":"","restart_required":false,"help_text":"过滤指定 RRU 单元；留空时返回全部。","order":1}}$json$::jsonb,
    $json$["Device.Services.FAPService.{i}.RRU.{i}"]$json$::jsonb,
    '["DSP"]'::jsonb,
    '展示 RRU 型号、版本、链路和射频状态。',
    '适合排查 RRU 失联、驻波或版本不一致问题。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
-- ---- 传输管理 ----
(
    '00000000-0000-0000-0006-000000000001',
    '查询传输链路',
    'DSP LINKSTATUS',
    '6',
    '显示所有传输链路的当前状态',
    'GetParameterValues',
    'DSP',
    $json${"LINKTYPE":{"type":"enum","required":false,"description":"链路类型","min_value":0,"max_value":0,"default_value":"S1","suggested_value":"S1","unit":"","restart_required":false,"help_text":"可按 S1、X2、NG 等链路类型过滤。","order":1,"options":[{"label":"S1","value":"S1"},{"label":"X2","value":"X2"},{"label":"NG","value":"NG"}]}}$json$::jsonb,
    $json$["Device.Services.FAPService.{i}.Transport.Link.{i}"]$json$::jsonb,
    '["DSP"]'::jsonb,
    '展示回传与对等接口的链路状态。',
    '常用于排查回传中断、对端不可达等问题。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0006-000000000002',
    '查询SCTP链路',
    'DSP SCTP',
    '6',
    '显示SCTP传输链路状态',
    'GetParameterValues',
    'DSP',
    $json${"LNKID":{"type":"number","required":false,"description":"链路ID","min_value":0,"max_value":255,"default_value":0,"suggested_value":1,"unit":"","restart_required":false,"help_text":"过滤指定 SCTP 链路编号。","order":1}}$json$::jsonb,
    $json$["Device.Services.FAPService.{i}.Transport.SCTP.{i}"]$json$::jsonb,
    '["DSP"]'::jsonb,
    '查看 SCTP 关联状态、心跳与重传信息。',
    '适合排查核心网连接异常与链路抖动。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0006-000000000003',
    '查询IP地址',
    'LST IPADDR',
    '6',
    '列出设备所有接口IP地址配置',
    'GetParameterValues',
    'LST',
    '{}'::jsonb,
    $json$["Device.IP.Interface.{i}"]$json$::jsonb,
    '["LST"]'::jsonb,
    '列出管理口、业务口及其他接口的 IP 配置。',
    '适合核查地址、掩码、网关与 VLAN 规划。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
-- ---- 版本管理 ----
(
    '00000000-0000-0000-0007-000000000001',
    '查询设备版本',
    'DSP VERSION',
    '7',
    '显示设备软件版本信息',
    'GetParameterValues',
    'DSP',
    '{}'::jsonb,
    $json$["Device.DeviceInfo.SoftwareVersion"]$json$::jsonb,
    '["DSP"]'::jsonb,
    '查询设备当前运行的软件版本与构建信息。',
    '常用于升级前后版本核对。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0007-000000000002',
    '查询软件包',
    'LST PKG',
    '7',
    '查询可用软件包列表',
    'GetParameterValues',
    'LST',
    '{}'::jsonb,
    $json$["Device.Services.FAPService.{i}.SoftwareImage.{i}"]$json$::jsonb,
    '["LST"]'::jsonb,
    '列出设备可识别的软件包及其状态。',
    '适合在升级前检查包文件是否已正确下发。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
),
(
    '00000000-0000-0000-0007-000000000003',
    '升级软件包',
    'UPG PKG',
    '7',
    '执行软件包升级',
    'Download',
    'UPG',
    $json${"PKGID":{"type":"string","required":true,"description":"软件包ID","min_value":0,"max_value":0,"default_value":"","suggested_value":"pkg-lte-v1.0.0.bin","unit":"","restart_required":true,"help_text":"输入待升级的软件包标识或文件名。","order":1},"MODE":{"type":"enum","required":false,"description":"升级模式","min_value":0,"max_value":0,"default_value":"IMMEDIATE","suggested_value":"IMMEDIATE","unit":"","restart_required":true,"help_text":"可选择立即升级或延迟升级。","order":2,"options":[{"label":"立即","value":"IMMEDIATE"},{"label":"延迟","value":"DELAYED"}]}}$json$::jsonb,
    $json$["Device.Services.FAPService.{i}.SoftwareImage.{i}.Download"]$json$::jsonb,
    '["UPG"]'::jsonb,
    '触发设备下载并升级指定软件包。',
    '升级通常会导致业务中断，建议结合版本校验与回退预案执行。',
    '["eNB", "gNB"]'::jsonb,
    NOW()
)
ON CONFLICT (command_code) DO UPDATE SET
    command_name          = EXCLUDED.command_name,
    category              = EXCLUDED.category,
    description           = EXCLUDED.description,
    rpc_method            = EXCLUDED.rpc_method,
    operation_type        = EXCLUDED.operation_type,
    param_template        = EXCLUDED.param_template,
    param_paths           = EXCLUDED.param_paths,
    supported_operations  = EXCLUDED.supported_operations,
    help_doc              = EXCLUDED.help_doc,
    notes                 = EXCLUDED.notes,
    product_types         = EXCLUDED.product_types;

-- =============================================
-- D) 字典种子数据：产品类型 & MML 命令分类
-- =============================================

INSERT INTO sys_dictionaries (name, type, status, description) VALUES
('产品类型', 'product_type', TRUE, '设备产品类型')
ON CONFLICT DO NOTHING;

DELETE FROM sys_dictionary_details WHERE sys_dictionary_id = (SELECT id FROM sys_dictionaries WHERE type = 'product_type');

INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id) VALUES
('SmallCell-LTE', '1', 1, (SELECT id FROM sys_dictionaries WHERE type = 'product_type')),
('gNB-100', '2', 2, (SELECT id FROM sys_dictionaries WHERE type = 'product_type')),
('gNB-200', '3', 3, (SELECT id FROM sys_dictionaries WHERE type = 'product_type')),
('FAP-LTE-100', '4', 4, (SELECT id FROM sys_dictionaries WHERE type = 'product_type')),
('FAP-LTE-200', '5', 5, (SELECT id FROM sys_dictionaries WHERE type = 'product_type')),
('FAP-LTE-300', '6', 6, (SELECT id FROM sys_dictionaries WHERE type = 'product_type'));

INSERT INTO sys_dictionaries (name, type, status, description) VALUES
('MML命令类型', 'mml_command_category', TRUE, 'MML命令分类')
ON CONFLICT DO NOTHING;

DELETE FROM sys_dictionary_details WHERE sys_dictionary_id = (SELECT id FROM sys_dictionaries WHERE type = 'mml_command_category');

INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id) VALUES
('小区管理', '1', 1, (SELECT id FROM sys_dictionaries WHERE type = 'mml_command_category')),
('邻区管理', '2', 2, (SELECT id FROM sys_dictionaries WHERE type = 'mml_command_category')),
('基站管理', '3', 3, (SELECT id FROM sys_dictionaries WHERE type = 'mml_command_category')),
('告警查询', '4', 4, (SELECT id FROM sys_dictionaries WHERE type = 'mml_command_category')),
('性能采集', '5', 5, (SELECT id FROM sys_dictionaries WHERE type = 'mml_command_category')),
('传输管理', '6', 6, (SELECT id FROM sys_dictionaries WHERE type = 'mml_command_category')),
('版本管理', '7', 7, (SELECT id FROM sys_dictionaries WHERE type = 'mml_command_category'));

-- +goose Down
DELETE FROM sys_dictionary_details WHERE sys_dictionary_id IN (
    SELECT id FROM sys_dictionaries WHERE type IN ('product_type', 'mml_command_category')
);
DELETE FROM sys_dictionaries WHERE type IN ('product_type', 'mml_command_category');
DELETE FROM mml_commands WHERE id::text LIKE '00000000-0000-0000-%';
