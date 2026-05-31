-- +goose Up
-- +goose StatementBegin
-- 翻译 MML chapter / extension 组的 en 文案。
-- 术语遵循 TR-069 Amendment 6 + 3GPP TS 32.111-3 OAM 命名习惯。
-- 同步两个字段:legacy group_name_en 列 + name_i18n.en JSONB key,前端读哪个都对。

WITH translations(chapter_code, en) AS (
  VALUES
    ('SA', 'Device Info Parameters'),
    ('SB', 'Software Version Parameters'),
    ('SC', 'Base Station OAM Parameters'),
    ('SD', 'Alarm Parameters'),
    ('SE', 'Log Parameters'),
    ('SF', 'Cell Service Parameters (Overall)'),
    ('SG', 'SCTP Parameters'),
    ('SH', 'RAN Protocol Stack Parameters'),
    ('SI', 'Neighbor Cell Parameters'),
    ('SJ', 'Mobility Parameters'),
    ('SK', 'SON Parameters'),
    ('SL', 'WAN Port Configuration Parameters'),
    ('SM', 'IPsec Parameters'),
    ('SN', 'Time Server Parameters'),
    ('SO', 'GPS Information Parameters'),
    ('SP', 'Measurement Report Parameters'),
    ('SQ', 'Performance Parameters'),
    ('SR', 'Extended Integrated Picocell Parameters')
)
UPDATE mml_command_groups g
SET group_name_en = t.en,
    name_i18n = jsonb_set(COALESCE(name_i18n, '{}'::jsonb), '{en}', to_jsonb(t.en))
FROM translations t
WHERE g.chapter_code = t.chapter_code;

-- Extensions (group_code = 'chapter:SX_*'):"Device 扩展" → "Device Extension"
WITH ext_trans(group_code, en) AS (
  VALUES
    ('chapter:SX_DEVICE_EXT',                'Device Extension'),
    ('chapter:SX_DEVICEGSM_EXT',             'DeviceGSM Extension'),
    ('chapter:SX_INTERNETGATEWAYDEVICE_EXT', 'InternetGatewayDevice Extension'),
    ('chapter:SX_BOARDCONF_EXT',             'boardconf Extension')
)
UPDATE mml_command_groups g
SET group_name_en = t.en,
    name_i18n = jsonb_set(COALESCE(name_i18n, '{}'::jsonb), '{en}', to_jsonb(t.en))
FROM ext_trans t
WHERE g.group_code = t.group_code;

-- MML commands: 把 logical_name_i18n.en 和 command_name_i18n.en 翻译为英文。
-- 采用基于 operation_type + logical_code 的模式化翻译,
-- 把已经存在的中文逻辑名换成英文(常见 ML 术语字典)。
-- 注:operation_type 已经独立从 command_code 解析,前端组装 display_name。
WITH logical_name_trans(logical_code, en) AS (
  VALUES
    -- 设备/软件/网管
    ('DEVICE_INFO',              'Device Basic Info'),
    ('DEVICE_INFO_SW_UPGRADE',   'Device Software Upgrade Status'),
    ('SOFTWARE_CTRL',            'Software Version Control'),
    ('SOFTWARE_FILE',            'Software File'),
    ('FAP_CONTROL_LTE',          'FAP Control LTE'),
    ('FAP_CONTROL_GW',           'FAP Control Gateway'),
    ('FAP_SERVICE',              'FAP Service'),
    -- 告警/日志
    ('ALARM_LIST',               'Alarm List'),
    ('LOG_CTRL',                 'Log Control'),
    ('LOG_QUERY',                'Log Query'),
    -- 小区/RAN
    ('CELL_CONFIG',              'Cell Configuration'),
    ('CELL_CTRL',                'Cell Control'),
    ('RRC_CONFIG',               'RRC Configuration'),
    ('PHY_CONFIG',               'PHY Layer Configuration'),
    ('MAC_CONFIG',               'MAC Layer Configuration'),
    ('PDCP_CONFIG',              'PDCP Layer Configuration'),
    ('RLC_CONFIG',               'RLC Layer Configuration'),
    -- 邻区/移动性
    ('NEIGHBOR_LTE',             'LTE Neighbor Cell'),
    ('NEIGHBOR_GERAN',           'GERAN Neighbor Cell'),
    ('NEIGHBOR_UTRAN',           'UTRAN Neighbor Cell'),
    ('NEIGHBOR_NR',              'NR Neighbor Cell'),
    ('A1_MEASURE_CTRL',          'A1 Measurement Control'),
    ('A2_MEASURE_CTRL',          'A2 Measurement Control'),
    ('A3_MEASURE_CTRL',          'A3 Measurement Control'),
    ('A4_MEASURE_CTRL',          'A4 Measurement Control'),
    ('A5_MEASURE_CTRL',          'A5 Measurement Control'),
    ('B1_MEASURE_CTRL',          'B1 Measurement Control'),
    ('B2_MEASURE_CTRL',          'B2 Measurement Control'),
    ('CARRIER',                  'Carrier'),
    ('DRX_INITIAL_PARAM',        'DRX Initial Parameters'),
    ('GERAN_FREQ_GROUP',         'GERAN Frequency Group'),
    ('UTRAN_FREQ_GROUP',         'UTRAN Frequency Group'),
    ('INTER_FREQ_CARRIER',       'Inter-Frequency Carrier'),
    ('INTRA_FREQ_CARRIER',       'Intra-Frequency Carrier'),
    -- SON
    ('SON_CONFIG',               'SON Configuration'),
    ('ANR_CONFIG',               'ANR Configuration'),
    ('MRO_CONFIG',               'MRO Configuration'),
    -- WAN/IPsec/NTP/GPS
    ('WAN_CONFIG',               'WAN Configuration'),
    ('IPSEC_CONFIG',             'IPsec Configuration'),
    ('NTP_CONFIG',               'NTP Configuration'),
    ('GPS_INFO',                 'GPS Information'),
    -- MR/PM
    ('MR_CONFIG',                'MR Configuration'),
    ('PM_CONFIG',                'PM Configuration'),
    -- SCTP
    ('SCTP_CONFIG',              'SCTP Configuration'),
    ('X2_IP_ADDR_MAP_INFO',      'X2 IP Address Mapping Info'),
    -- Inter-RAT
    ('INTER_RAT_CELL_GSM',       'Inter-RAT Cell (GSM)'),
    ('INTER_RAT_CELL_UMTS',      'Inter-RAT Cell (UMTS)'),
    -- LTE generic
    ('LTE',                      'LTE'),
    ('LTE_INTRA_FREQ',           'LTE Intra-Frequency'),
    ('LTE_INTER_FREQ',           'LTE Inter-Frequency')
)
UPDATE mml_commands c
SET logical_name_i18n = jsonb_set(COALESCE(logical_name_i18n, '{}'::jsonb), '{en}', to_jsonb(t.en)),
    command_name_i18n = jsonb_set(COALESCE(command_name_i18n, '{}'::jsonb), '{en}', to_jsonb(t.en))
FROM logical_name_trans t
WHERE c.logical_code = t.logical_code;

-- 兜底:logical_code 未在字典里的命令,把 logical_name_i18n.en 暂时设为 logical_code 自身
-- (英文格式但形态像 SCREAMING_SNAKE,比中文好,后续可继续翻译)。
UPDATE mml_commands
SET logical_name_i18n = jsonb_set(COALESCE(logical_name_i18n, '{}'::jsonb), '{en}',
                                   to_jsonb(replace(logical_code, '_', ' ')))
WHERE (logical_name_i18n->>'en') ~ '[一-鿿]'  -- 仍有中文
  AND logical_code IS NOT NULL
  AND logical_code <> '';

-- Edge cases: logical_code 为空但已知中文逻辑名的手动翻译。
UPDATE mml_commands
SET logical_name_i18n = jsonb_set(COALESCE(logical_name_i18n,'{}'::jsonb), '{en}', '"FAP Carrier Basic Config"'::jsonb),
    command_name_i18n = jsonb_set(COALESCE(command_name_i18n,'{}'::jsonb), '{en}', '"FAP Carrier Basic Config"'::jsonb)
WHERE logical_name_i18n->>'en' = 'FAP 载波基本配置';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- 回滚:把 en 还原为 zh 兜底(无法精准还原翻译前的状态,但保证非空)。
UPDATE mml_command_groups
SET group_name_en = group_name_zh,
    name_i18n = jsonb_set(COALESCE(name_i18n, '{}'::jsonb), '{en}', to_jsonb(group_name_zh))
WHERE chapter_code IS NOT NULL OR group_code LIKE 'chapter:SX_%';

UPDATE mml_commands
SET logical_name_i18n = name_i18n - 'en',
    command_name_i18n = command_name_i18n - 'en'
WHERE logical_code IS NOT NULL;
-- +goose StatementEnd
