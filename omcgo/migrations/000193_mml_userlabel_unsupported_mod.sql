-- +goose Up
-- ============================================================
-- 000193_mml_userlabel_unsupported_mod.sql
--
-- 实测 BAICELLS mBS31001 / FAP/mBS31001/SC 固件不支持 SET 路径
-- Device.DeviceInfo.UserLabel —— SetParameterValues 直接返回
--   <FaultCode>9003</FaultCode> Invalid arguments
--   <SetParameterValuesFault>
--     <ParameterName>Device.DeviceInfo.UserLabel</ParameterName>
--     <FaultCode>9005</FaultCode>
--     <FaultString>AttributeIdNotFound : Device.DeviceInfo.UserLabel</FaultString>
--   </SetParameterValuesFault>
-- TR-069 SPV 是原子操作，单 path 9005 → 整条 RPC 9003 回滚，连带把
-- 同批的 DnPrefix 等本来能 set 的字段也一起失败。
--
-- 当前 BLQ 默认 param_model 把 Device.DeviceInfo.UserLabel 登记为
-- READ_WRITE（spec 期望），UI 会勾上让用户改 → 触发 9003。
--
-- 处置（hotfix）：手工把 standard MOD 命令下指向 standardPath
-- "Device.DeviceInfo.UserLabel" 的 sub_field 标 is_supported=false。
-- 等价于 LST 同 path 在前几次 catalog 维护时已被标的状态（参见 admin
-- 查询：LST DEVICE_INFO USER_LABEL 已 is_supported=false）。
--
-- 注意：本次只处理 standard 命令 MOD 操作下 standardPath 全等的那条
-- （MOD DEVICE_INFO），不动 MOD DEVICE_INFO_MU / MOD ETHERNET_INTERFACE
-- 等其它 UserLabel —— 它们 standardPath 不同（嵌实例后缀），是否支持要
-- 看具体固件，留给 T-0174 auto-learn 链路按设备实测反馈逐条收敛。
--
-- 后续兜底：000192 之后 ACS 已经把 SetParameterValuesFault 详情写进
-- device_tasks.result，MML 模块在 ResultAggregator.OnTaskCompleted 看到
-- code=9005 时自动 UPDATE is_supported=false，无需再为下一个不支持
-- path 单独发 migration。
-- ============================================================

UPDATE mml_command_sub_fields csf
   SET is_supported = false,
       updated_at   = NOW()
  FROM mml_commands c, standard_params sp
 WHERE csf.command_id = c.id
   AND csf.standard_path_id = sp.id
   AND c.source = 'standard'
   AND c.operation_type = 'MOD'
   AND sp.standard_path = 'Device.DeviceInfo.UserLabel'
   AND csf.is_supported = true;


-- +goose Down
-- 还原 hotfix：把 standard MOD 下 Device.DeviceInfo.UserLabel 重新标 supported。
-- 注意：auto-learn 也可能写过 is_supported=false，down 之后下次执行
-- 仍可能被 auto-learn 重新置 false，这是预期行为（auto-learn 才是长期真相源）。
UPDATE mml_command_sub_fields csf
   SET is_supported = true,
       updated_at   = NOW()
  FROM mml_commands c, standard_params sp
 WHERE csf.command_id = c.id
   AND csf.standard_path_id = sp.id
   AND c.source = 'standard'
   AND c.operation_type = 'MOD'
   AND sp.standard_path = 'Device.DeviceInfo.UserLabel';
