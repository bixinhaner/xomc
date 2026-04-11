-- ============================================================
-- 000010_seed_data.up.sql
-- 全部种子数据
-- 所有 INSERT 使用 ON CONFLICT DO NOTHING 确保幂等性
-- ============================================================

-- 1. 系统角色
INSERT INTO roles (id, name, description, is_system) VALUES
    ('10000000-0000-0000-0000-000000000001', 'admin', 'System administrator with full access', TRUE),
    ('10000000-0000-0000-0000-000000000002', 'operator', 'Operator with read/write access to operational resources', TRUE),
    ('10000000-0000-0000-0000-000000000003', 'viewer', 'Read-only viewer', TRUE)
ON CONFLICT (name) DO NOTHING;

-- 2. 管理员用户
INSERT INTO users (id, username, password_hash, display_name, status) VALUES
    ('20000000-0000-0000-0000-000000000001', 'admin',
     '$2a$10$5feKmwxvoxEyqIo5DaYQNuNcPFWZnRdNytomGLrXDnv0e5MgnEJT6',
     'System Admin', 'active')
ON CONFLICT (username) DO NOTHING;

-- 3. 用户角色绑定 + 默认角色
INSERT INTO user_roles (user_id, role_id) VALUES
    ('20000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001')
ON CONFLICT (user_id, role_id) DO NOTHING;

UPDATE user_roles SET is_default = TRUE
WHERE user_id = '20000000-0000-0000-0000-000000000001'
  AND role_id = '10000000-0000-0000-0000-000000000001'
  AND is_default = FALSE;

-- 4. 角色继承
INSERT INTO role_inheritance (parent_role_id, child_role_id, domain)
SELECT p.id, c.id, 'system' FROM roles p, roles c WHERE p.name = 'admin' AND c.name = 'operator'
ON CONFLICT DO NOTHING;

INSERT INTO role_inheritance (parent_role_id, child_role_id, domain)
SELECT p.id, c.id, 'system' FROM roles p, roles c WHERE p.name = 'operator' AND c.name = 'viewer'
ON CONFLICT DO NOTHING;

-- 5. 权限
INSERT INTO permissions (role_id, resource, action)
SELECT '10000000-0000-0000-0000-000000000001', r, a
FROM unnest(ARRAY['devices','alarms','pm','config','datamodels','users','roles','firmware','northbound','interop']) AS r,
     unnest(ARRAY['read','write','delete','admin']) AS a
ON CONFLICT DO NOTHING;

INSERT INTO permissions (role_id, resource, action)
SELECT '10000000-0000-0000-0000-000000000002', r, a
FROM unnest(ARRAY['devices','alarms','pm','config','datamodels','firmware','northbound']) AS r,
     unnest(ARRAY['read','write']) AS a
ON CONFLICT DO NOTHING;

INSERT INTO permissions (role_id, resource, action)
SELECT '10000000-0000-0000-0000-000000000003', r, 'read'
FROM unnest(ARRAY['devices','alarms','pm','config','datamodels','firmware','northbound']) AS r
ON CONFLICT DO NOTHING;

-- 6. OUI 厂商注册
INSERT INTO oui_registry (oui, manufacturer, short_name, country) VALUES
    ('00E0FC', 'Huawei Technologies Co., Ltd.', 'Huawei', 'China'),
    ('001E7E', 'ZTE Corporation', 'ZTE', 'China'),
    ('000DB9', 'Ericsson AB', 'Ericsson', 'Sweden'),
    ('0004F2', 'Nokia Corporation', 'Nokia', 'Finland'),
    ('58FB96', 'Comba Telecom Systems', 'Comba', 'China'),
    ('D4612E', 'Datang Mobile Communications', 'Datang', 'China'),
    ('00259C', 'Cisco-Linksys LLC', 'Cisco', 'USA'),
    ('7C7A53', 'Ruijie Networks Co., Ltd.', 'Ruijie', 'China')
ON CONFLICT (oui) DO NOTHING;

-- 7. 默认设备组
INSERT INTO device_groups (id, name, parent_id, level, is_default, status, remark, created_by) VALUES
    ('00000000-0000-0000-0000-000000000001', '默认设备组', NULL, 1, TRUE, 'active', '系统默认一级设备组，不可修改删除', 'system')
ON CONFLICT (id) DO NOTHING;

INSERT INTO device_groups (id, name, parent_id, level, is_default, status, remark, created_by) VALUES
    ('00000000-0000-0000-0000-000000000002', '未分组设备', '00000000-0000-0000-0000-000000000001', 2, TRUE, 'active', '系统默认二级设备组，删除组后设备自动归入此组', 'system')
ON CONFLICT (id) DO NOTHING;

-- 将现有未归组的设备自动加入默认二级组
INSERT INTO device_group_members (group_id, device_id, added_at)
SELECT '00000000-0000-0000-0000-000000000002'::uuid, d.id, NOW()
FROM devices d
WHERE NOT EXISTS (SELECT 1 FROM device_group_members dgm WHERE dgm.device_id = d.id)
ON CONFLICT (device_id) DO NOTHING;

-- 8. KPI 定义 (LTE)
INSERT INTO kpi_definitions (id, name, display_name, formula, unit, category, carrier, technology, counters) VALUES
('30000000-0001-4000-8000-000000000001', 'RRC_CONN_SETUP_SR', 'RRC连接建立成功率', '(rrc_conn_setup_succ / rrc_conn_setup_att) * 100', '%', 'accessibility', NULL, 'lte', '["rrc_conn_setup_succ", "rrc_conn_setup_att"]'::jsonb),
('30000000-0001-4000-8000-000000000002', 'ERAB_SETUP_SR', 'E-RAB建立成功率', '(erab_setup_succ / erab_setup_att) * 100', '%', 'accessibility', NULL, 'lte', '["erab_setup_succ", "erab_setup_att"]'::jsonb),
('30000000-0001-4000-8000-000000000003', 'INTRA_FREQ_HO_SR', '同频切换成功率', '(intra_freq_ho_succ / intra_freq_ho_att) * 100', '%', 'mobility', NULL, 'lte', '["intra_freq_ho_succ", "intra_freq_ho_att"]'::jsonb),
('30000000-0001-4000-8000-000000000004', 'INTER_FREQ_HO_SR', '异频切换成功率', '(inter_freq_ho_succ / inter_freq_ho_att) * 100', '%', 'mobility', NULL, 'lte', '["inter_freq_ho_succ", "inter_freq_ho_att"]'::jsonb),
('30000000-0001-4000-8000-000000000005', 'CALL_DROP_RATE', '掉话率', '(erab_abnormal_release / erab_release_total) * 100', '%', 'retainability', NULL, 'lte', '["erab_abnormal_release", "erab_release_total"]'::jsonb),
('30000000-0001-4000-8000-000000000006', 'DL_PRB_UTIL', '下行PRB利用率', '(dl_prb_used_avg / dl_prb_available) * 100', '%', 'utilization', NULL, 'lte', '["dl_prb_used_avg", "dl_prb_available"]'::jsonb)
ON CONFLICT (name) DO NOTHING;

-- KPI 定义 (NR)
INSERT INTO kpi_definitions (id, name, display_name, formula, unit, category, carrier, technology, counters) VALUES
('30000000-0002-4000-8000-000000000001', 'NR_RRC_SETUP_SR', 'NR RRC建立成功率', '(nr_rrc_setup_succ / nr_rrc_setup_att) * 100', '%', 'accessibility', NULL, 'nr', '["nr_rrc_setup_succ", "nr_rrc_setup_att"]'::jsonb),
('30000000-0002-4000-8000-000000000002', 'NR_PDCP_RATE_DL', 'NR 下行PDCP速率', 'pdcp_vol_dl / report_period', 'Mbps', 'throughput', NULL, 'nr', '["pdcp_vol_dl", "report_period"]'::jsonb),
('30000000-0002-4000-8000-000000000003', 'NR_SA_HO_SR', 'NR SA切换成功率', '(nr_ho_succ / nr_ho_att) * 100', '%', 'mobility', NULL, 'nr', '["nr_ho_succ", "nr_ho_att"]'::jsonb),
('30000000-0002-4000-8000-000000000004', 'NR_PRB_UTIL_DL', 'NR 下行PRB利用率', '(nr_dl_prb_used / nr_dl_prb_total) * 100', '%', 'utilization', NULL, 'nr', '["nr_dl_prb_used", "nr_dl_prb_total"]'::jsonb),
('30000000-0002-4000-8000-000000000005', 'NR_CQI_AVG', 'NR 平均CQI', 'AVG(cqi_value)', '', 'quality', NULL, 'nr', '["cqi_value"]'::jsonb),
('30000000-0002-4000-8000-000000000006', 'NR_RLC_LOSS_RATE', 'NR RLC丢包率', '(rlc_retx_dl / rlc_tx_dl) * 100', '%', 'retainability', NULL, 'nr', '["rlc_retx_dl", "rlc_tx_dl"]'::jsonb)
ON CONFLICT (name) DO NOTHING;

-- 9. 默认数据模型 (carrier_default scope)
INSERT INTO data_model_definitions (id, carrier, technology, version, oui, product_class, scope, status, is_active, parameter_tree, description) VALUES
('30000047-0001-4000-8000-000000000001', 'cmcc', 'lte', '1.0', NULL, NULL, 'carrier_default', 'active', true, '{"Device.DeviceInfo": {"access": "r"}, "Device.DeviceInfo.SoftwareVersion": {"access": "r", "type": "string"}, "Device.DeviceInfo.HardwareVersion": {"access": "r", "type": "string"}, "Device.ManagementServer": {"access": "rw"}, "Device.Services.FAPService.1": {"access": "rw"}, "Device.Services.FAPService.1.FAPControl.LTE": {"access": "rw"}, "Device.FAP.GPS": {"access": "r"}}'::jsonb, '中国移动 LTE 默认数据模型')
ON CONFLICT DO NOTHING;

INSERT INTO data_model_definitions (id, carrier, technology, version, oui, product_class, scope, status, is_active, parameter_tree, description) VALUES
('30000047-0001-4000-8000-000000000002', 'cmcc', 'nr', '1.0', NULL, NULL, 'carrier_default', 'active', true, '{"Device.DeviceInfo": {"access": "r"}, "Device.DeviceInfo.SoftwareVersion": {"access": "r", "type": "string"}, "Device.DeviceInfo.HardwareVersion": {"access": "r", "type": "string"}, "Device.ManagementServer": {"access": "rw"}}'::jsonb, '中国移动 NR 默认数据模型')
ON CONFLICT DO NOTHING;

INSERT INTO data_model_definitions (id, carrier, technology, version, oui, product_class, scope, status, is_active, parameter_tree, description) VALUES
('30000047-0001-4000-8000-000000000003', 'ctcc', 'lte', '1.0', NULL, NULL, 'carrier_default', 'active', true, '{"Device.DeviceInfo": {"access": "r"}, "Device.DeviceInfo.SoftwareVersion": {"access": "r", "type": "string"}, "Device.ManagementServer": {"access": "rw"}}'::jsonb, '中国电信 LTE 默认数据模型')
ON CONFLICT DO NOTHING;

INSERT INTO data_model_definitions (id, carrier, technology, version, oui, product_class, scope, status, is_active, parameter_tree, description) VALUES
('30000047-0001-4000-8000-000000000004', 'cucc', 'lte', '1.0', NULL, NULL, 'carrier_default', 'active', true, '{"Device.DeviceInfo": {"access": "r"}, "Device.DeviceInfo.SoftwareVersion": {"access": "r", "type": "string"}, "Device.ManagementServer": {"access": "rw"}}'::jsonb, '中国联通 LTE 默认数据模型')
ON CONFLICT DO NOTHING;

-- 默认数据模型 (oui scope - BaiCells)
INSERT INTO data_model_definitions (id, carrier, technology, version, oui, product_class, scope, status, is_active, parameter_tree, description) VALUES
('30000047-0002-4000-8000-000000000001', 'cmcc', 'lte', '1.0', '001A2B', NULL, 'oui', 'active', true, '{"Device.DeviceInfo": {"access": "r"}, "Device.DeviceInfo.SoftwareVersion": {"access": "r", "type": "string"}, "Device.DeviceInfo.HardwareVersion": {"access": "r", "type": "string"}, "Device.ManagementServer": {"access": "rw"}, "Device.Services.FAPService.1": {"access": "rw"}, "Device.Services.FAPService.1.FAPControl.LTE": {"access": "rw"}, "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus": {"access": "rw", "type": "boolean"}, "Device.FAP.GPS": {"access": "r"}, "Device.FAP.GPS.LockedLatitude": {"access": "r", "type": "string"}, "Device.FAP.GPS.LockedLongitude": {"access": "r", "type": "string"}}'::jsonb, 'BaiCells 厂商 LTE 设备默认配置')
ON CONFLICT DO NOTHING;

-- 默认数据模型 (product scope - BaiCells SmallCell-LTE)
INSERT INTO data_model_definitions (id, carrier, technology, version, oui, product_class, scope, status, is_active, parameter_tree, description) VALUES
('30000047-0003-4000-8000-000000000001', 'cmcc', 'lte', '1.0', '001A2B', 'SmallCell-LTE', 'product', 'active', true, '{"Device.DeviceInfo": {"access": "r"}, "Device.DeviceInfo.SoftwareVersion": {"access": "r", "type": "string"}, "Device.DeviceInfo.HardwareVersion": {"access": "r", "type": "string"}, "Device.ManagementServer": {"access": "rw"}, "Device.Services.FAPService.1": {"access": "rw"}, "Device.Services.FAPService.1.FAPControl.LTE": {"access": "rw"}, "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus": {"access": "rw", "type": "boolean"}, "Device.FAP.GPS": {"access": "r"}}'::jsonb, 'BaiCells SmallCell-LTE 产品数据模型')
ON CONFLICT DO NOTHING;

-- 10. 字典数据
INSERT INTO sys_dictionaries (name, type, status, description) VALUES
('性别', 'gender', TRUE, '用户性别'),
('数据库int类型', 'int', TRUE, '整型映射'),
('时间日期类型', 'time.Time', TRUE, '时间类型映射'),
('浮点型', 'float64', TRUE, '浮点类型映射'),
('字符串', 'string', TRUE, '字符串类型映射'),
('布尔类型', 'bool', TRUE, '布尔类型映射')
ON CONFLICT DO NOTHING;

INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id) VALUES
('男', '1', 1, (SELECT id FROM sys_dictionaries WHERE type = 'gender')),
('女', '2', 2, (SELECT id FROM sys_dictionaries WHERE type = 'gender'));

INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id) VALUES
('int', '1', 1, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('int8', '2', 2, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('int16', '3', 3, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('int32', '4', 4, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('int64', '5', 5, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('rune', '6', 6, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('uint', '7', 7, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('uint8', '8', 8, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('uint16', '9', 9, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('uint32', '10', 10, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('uint64', '11', 11, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('uintptr', '12', 12, (SELECT id FROM sys_dictionaries WHERE type = 'int')),
('byte', '13', 13, (SELECT id FROM sys_dictionaries WHERE type = 'int'));

INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id) VALUES
('time.Time', '1', 1, (SELECT id FROM sys_dictionaries WHERE type = 'time.Time'));

INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id) VALUES
('float32', '1', 1, (SELECT id FROM sys_dictionaries WHERE type = 'float64')),
('float64', '2', 2, (SELECT id FROM sys_dictionaries WHERE type = 'float64'));

INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id) VALUES
('string', '1', 1, (SELECT id FROM sys_dictionaries WHERE type = 'string'));

INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id) VALUES
('bool', '1', 1, (SELECT id FROM sys_dictionaries WHERE type = 'bool'));

-- 11. 系统配置
INSERT INTO sys_configs (category, key, value, value_type, description, is_public) VALUES
('system', 'system_name', 'OMC 网管系统', 'string', '系统名称', TRUE),
('system', 'system_version', '1.0.0', 'string', '系统版本', TRUE),
('system', 'session_timeout', '30', 'int', '会话超时时间（分钟）', FALSE),
('system', 'max_login_attempts', '5', 'int', '最大登录尝试次数', FALSE),
('system', 'lockout_duration', '30', 'int', '锁定时长（分钟）', FALSE),
('system', 'password_min_length', '6', 'int', '密码最小长度', FALSE)
ON CONFLICT DO NOTHING;
