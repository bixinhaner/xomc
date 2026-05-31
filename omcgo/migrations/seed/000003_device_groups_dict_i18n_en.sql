-- +goose Up
-- +goose StatementBegin
-- 翻译 device_groups + sys_dictionaries + sys_dictionary_details 的 en-US 文案。
-- 终态: 切英文后 /device/group + /system/data-dictionary 页面分组名/字典名/字典项都英文显示。
-- 命名规则:
--   - 城市/省份 → 拼音 + 运营商缩写 (北京移动 → "Beijing CMCC")
--   - 通用术语 → 英文常见说法 (运维 → "OPS",维护中 → "Under Maintenance")
--   - 设备型号/产品类等代码 → 保持原样 (英文已是 ASCII)

-- ===== device_groups =====

UPDATE device_groups SET
  name_i18n = jsonb_build_object('zh-CN', name, 'en-US', CASE name
    WHEN '默认设备组'         THEN 'Default Group'
    WHEN '未分组设备'         THEN 'Ungrouped Devices'
    WHEN '移动设备域'         THEN 'CMCC Device Domain'
    WHEN '电信设备域'         THEN 'CTCC Device Domain'
    WHEN '联通设备域'         THEN 'CUCC Device Domain'
    WHEN '测试设备域'         THEN 'Test Device Domain'
    WHEN '运维设备域'         THEN 'OPS Device Domain'
    WHEN '北京移动'           THEN 'Beijing CMCC'
    WHEN '上海移动'           THEN 'Shanghai CMCC'
    WHEN '广东移动'           THEN 'Guangdong CMCC'
    WHEN '江苏电信'           THEN 'Jiangsu CTCC'
    WHEN '浙江电信'           THEN 'Zhejiang CTCC'
    WHEN '河南联通'           THEN 'Henan CUCC'
    WHEN '山东联通'           THEN 'Shandong CUCC'
    WHEN '联调测试组'         THEN 'Joint Test Group'
    WHEN '巡检设备组'         THEN 'Routine Inspection Group'
    WHEN '一致性测试组'       THEN 'Conformance Test Group'
    ELSE name
  END);

UPDATE device_groups SET
  description_i18n = jsonb_build_object('zh-CN', description, 'en-US', CASE description
    WHEN '中国移动设备管理域' THEN 'China Mobile (CMCC) device management domain'
    WHEN '中国电信设备管理域' THEN 'China Telecom (CTCC) device management domain'
    WHEN '中国联通设备管理域' THEN 'China Unicom (CUCC) device management domain'
    WHEN '测试与验收设备'     THEN 'Test and acceptance devices'
    WHEN '运维专用设备'       THEN 'OPS-dedicated devices'
    WHEN '北京地区移动设备'   THEN 'CMCC devices in Beijing'
    WHEN '上海地区移动设备'   THEN 'CMCC devices in Shanghai'
    WHEN '广东地区移动设备'   THEN 'CMCC devices in Guangdong'
    WHEN '江苏地区电信设备'   THEN 'CTCC devices in Jiangsu'
    WHEN '浙江地区电信设备'   THEN 'CTCC devices in Zhejiang'
    WHEN '河南地区联通设备'   THEN 'CUCC devices in Henan'
    WHEN '山东地区联通设备'   THEN 'CUCC devices in Shandong'
    WHEN '一致性验证设备'     THEN 'Conformance verification devices'
    WHEN '联调测试专用设备'   THEN 'Joint debug & test devices'
    WHEN '日常巡检设备'       THEN 'Daily inspection devices'
    WHEN '通过 SN 列表自动匹配的设备分组（T-DRULE-SN）'
                              THEN 'Devices auto-grouped by serial-number list (T-DRULE-SN)'
    ELSE description
  END)
WHERE description IS NOT NULL AND description <> '';

UPDATE device_groups SET
  remark_i18n = jsonb_build_object('zh-CN', remark, 'en-US', CASE remark
    WHEN '系统默认一级设备组，不可修改删除'
              THEN 'System default L1 group, cannot be modified or deleted'
    WHEN '系统默认二级设备组，删除组后设备自动归入此组'
              THEN 'System default L2 group, devices auto-fallback here when their group is deleted'
    WHEN '心跳路径自动归组：matching_mode=serialNumber + serial_number_list'
              THEN 'Auto-grouped on inform: matching_mode=serialNumber + serial_number_list'
    ELSE remark
  END)
WHERE remark IS NOT NULL AND remark <> '';

-- ===== sys_dictionaries =====

UPDATE sys_dictionaries SET
  name_i18n = jsonb_build_object('zh-CN', name, 'en-US', CASE type
    WHEN 'bool'             THEN 'Boolean'
    WHEN 'device_model'     THEN 'Device Model'
    WHEN 'firmware_version' THEN 'Firmware Version'
    WHEN 'float64'          THEN 'Float Type'
    WHEN 'gender'           THEN 'Gender'
    WHEN 'int'              THEN 'Integer Type'
    WHEN 'is_online'        THEN 'Online Status'
    WHEN 'lifecycle_state'  THEN 'Lifecycle State'
    WHEN 'network_type'     THEN 'Network Type'
    WHEN 'op_state'         THEN 'Activation State'
    WHEN 'product_class'    THEN 'Product Class'
    WHEN 'software_version' THEN 'Software Version'
    WHEN 'string'           THEN 'String Type'
    WHEN 'time.Time'        THEN 'DateTime Type'
    ELSE name
  END)
WHERE deleted_at IS NULL;

UPDATE sys_dictionaries SET
  description_i18n = jsonb_build_object('zh-CN', description, 'en-US', CASE type
    WHEN 'bool'             THEN 'Boolean type mapping'
    WHEN 'device_model'     THEN 'T-0162: device hardware model, aligned with devices.model_name'
    WHEN 'firmware_version' THEN 'T-0162: firmware version, aligned with devices.firmware_version'
    WHEN 'float64'          THEN 'Float type mapping'
    WHEN 'gender'           THEN 'User gender'
    WHEN 'int'              THEN 'Integer type mapping'
    WHEN 'is_online'        THEN 'T-0162: realtime heartbeat presence (true=online, false=offline)'
    WHEN 'lifecycle_state'  THEN 'T-0162: business lifecycle stage (6 states: discovered/registered/provisioning/commissioned/maintenance/decommissioned)'
    WHEN 'network_type'     THEN 'Device list filter — networkType values: eNB=LTE, gNB=NR'
    WHEN 'op_state'         THEN 'Device list filter — opState values: active=activated, inactive=not activated'
    WHEN 'product_class'    THEN 'Device productClass — DISTINCT from devices.product_class'
    WHEN 'software_version' THEN 'T-0162: software version, TR-069 Device.DeviceInfo.SoftwareVersion'
    WHEN 'string'           THEN 'String type mapping'
    WHEN 'time.Time'        THEN 'DateTime type mapping'
    ELSE description
  END)
WHERE deleted_at IS NULL AND description IS NOT NULL AND description <> '';

-- ===== sys_dictionary_details (按 dict type + label 翻译) =====
-- 用 from 子查询拿 dict type 上下文,只更新真正需要翻译的中文 label。

UPDATE sys_dictionary_details sdd
SET label_i18n = jsonb_build_object('zh-CN', sdd.label, 'en-US', CASE
    -- gender
    WHEN sd.type = 'gender' AND sdd.value = '1' THEN 'Male'
    WHEN sd.type = 'gender' AND sdd.value = '2' THEN 'Female'
    -- is_online
    WHEN sd.type = 'is_online' AND sdd.value = 'true'  THEN 'Online'
    WHEN sd.type = 'is_online' AND sdd.value = 'false' THEN 'Offline'
    -- lifecycle_state
    WHEN sd.type = 'lifecycle_state' AND sdd.value = 'discovered'      THEN 'Discovered'
    WHEN sd.type = 'lifecycle_state' AND sdd.value = 'registered'      THEN 'Registered'
    WHEN sd.type = 'lifecycle_state' AND sdd.value = 'provisioning'    THEN 'Provisioning'
    WHEN sd.type = 'lifecycle_state' AND sdd.value = 'commissioned'    THEN 'Commissioned'
    WHEN sd.type = 'lifecycle_state' AND sdd.value = 'maintenance'     THEN 'Under Maintenance'
    WHEN sd.type = 'lifecycle_state' AND sdd.value = 'decommissioned'  THEN 'Decommissioned'
    -- op_state
    WHEN sd.type = 'op_state' AND sdd.value = '1' THEN 'Activated'
    WHEN sd.type = 'op_state' AND sdd.value = '0' THEN 'Not Activated'
    -- network_type: 标签已是 "eNB (LTE)" 等,保持
    -- 其它 (bool/int/float/string/time/device_model/firmware_version/software_version/product_class):
    -- label 已经是英文/代码 — 保持
    ELSE sdd.label
  END)
FROM sys_dictionaries sd
WHERE sdd.sys_dictionary_id = sd.id
  AND sdd.deleted_at IS NULL
  AND sd.deleted_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- 回滚:把 i18n.en 还原为 zh 兜底(无法精准还原翻译前状态)
UPDATE device_groups SET
  name_i18n = jsonb_set(COALESCE(name_i18n,'{}'::jsonb), '{en-US}', to_jsonb(name)),
  description_i18n = jsonb_set(COALESCE(description_i18n,'{}'::jsonb), '{en-US}', to_jsonb(COALESCE(description,''))),
  remark_i18n = jsonb_set(COALESCE(remark_i18n,'{}'::jsonb), '{en-US}', to_jsonb(COALESCE(remark,'')));

UPDATE sys_dictionaries SET
  name_i18n = jsonb_set(COALESCE(name_i18n,'{}'::jsonb), '{en-US}', to_jsonb(name)),
  description_i18n = jsonb_set(COALESCE(description_i18n,'{}'::jsonb), '{en-US}', to_jsonb(COALESCE(description,'')))
WHERE deleted_at IS NULL;

UPDATE sys_dictionary_details SET
  label_i18n = jsonb_set(COALESCE(label_i18n,'{}'::jsonb), '{en-US}', to_jsonb(label))
WHERE deleted_at IS NULL;
-- +goose StatementEnd
