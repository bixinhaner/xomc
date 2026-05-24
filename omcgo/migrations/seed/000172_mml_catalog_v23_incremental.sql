-- ============================================================
-- MML catalog 增量补齐种子（T-0169 自动生成）
--
-- 源规范: 规范/移动/南向数据模型/cmcc-tdlte-southbound-data-model-v2.3.md
-- 规范 hash: b39532acfeb9be81
-- 版本: cmcc-td-lte-v2.3
-- 生成时间: 2026-05-24T08:47:22Z
--
-- Diff Summary:
--   🔴 新增 standard_params:        382
--   🔴 新增 mml_commands:           99
--   🟡 修改 mml_commands:           0
--   🔴 新增 mml_command_sub_fields: 625
--   ⚪ DB 多余 (orphan) commands:   0  (仅标记不删)
--   ⚠ CrossCheck warnings:         0
--
-- 工具: omcctl mml import-spec-md (internal/mml/specparser)
-- 设计: docs/project/prd/F06-mml-catalog-spec-parser.md
-- 由 T-0169 落地；不要手工修改，请重跑 omcctl 重新生成。
-- ============================================================

-- +goose Up
BEGIN;

-- Section A: standard_params — 新增 382 条
INSERT INTO standard_params (standard_path, entry_type, access, data_type, change_applies, min_value, max_value) VALUES
    ('Device.DeviceInfo.UserLabel', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.DnPrefix', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.ManufacturerOUI', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.Manufacturer', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.ModelName', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.SerialNumber', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.HardwareVersion', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.SoftwareVersion', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.HardwarePlatform', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.AdditionalHardwareVersion', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.AdditionalSoftwareVersion', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.ProvisioningCode', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.ProductClass', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.UpTime', 'parameter', 'READ_ONLY', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.3GPPSpecVersion', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.FirstUseDate', 'parameter', 'READ_ONLY', 'dateTime', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.DataModelSpecVersion', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.SwUpgrade.Stage', 'parameter', 'READ_ONLY', 'unsignedInt', 'Immediate', 1, 5),
    ('Device.DeviceInfo.SwUpgrade.FailureCause', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.SwUpgrade.Status', 'parameter', 'READ_ONLY', 'unsignedInt', 'Immediate', 1, 3),
    ('Device.SoftwareCtrl.AutoActivateEnable', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.SoftwareCtrl.ActivateTime', 'parameter', 'READ_WRITE', 'dateTime', 'Immediate', NULL, NULL),
    ('Device.SoftwareCtrl.ActivateEnable', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.SoftwareCtrl.SystemCurrentVersion', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.SoftwareCtrl.SystemBackupVersion', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.ManagementServer.URL', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.ManagementServer.Username', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.ManagementServer.Password', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.ManagementServer.PeriodicInformEnable', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.ManagementServer.PeriodicInformTime', 'parameter', 'READ_WRITE', 'dateTime', 'Immediate', NULL, NULL),
    ('Device.ManagementServer.PeriodicInformInterval', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 1, NULL),
    ('Device.ManagementServer.ParameterKey', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.ManagementServer.ConnectionRequestURL', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.ManagementServer.ConnectionRequestUsername', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.ManagementServer.ConnectionRequestPassword', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.ManagementServer.UDPConnectionRequestAddress', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.ManagementServer.STUNEnable', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.ManagementServer.STUNServerAddress', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.ManagementServer.STUNServerPort', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 65535),
    ('Device.ManagementServer.STUNUsername', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.ManagementServer.STUNPassword', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.ManagementServer.STUNMaximumKeepAlivePeriod', 'parameter', 'READ_WRITE', 'int', 'Immediate', -1, 65535),
    ('Device.ManagementServer.STUNMinimumKeepAlivePeriod', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 65535),
    ('Device.ManagementServer.NATDetected', 'parameter', 'READ_ONLY', 'boolean', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.SupportedAlarmNumberOfEntries', 'parameter', 'READ_ONLY', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.MaxCurrentAlarmEntries', 'parameter', 'READ_ONLY', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.CurrentAlarmNumberOfEntries', 'parameter', 'READ_ONLY', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.HistoryEventNumberOfEntries', 'parameter', 'READ_ONLY', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.ExpeditedEventNumberOfEntries', 'parameter', 'READ_ONLY', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.QueuedEventNumberOfEntries', 'parameter', 'READ_ONLY', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.CurrentAlarm.{i}.AlarmIdentifier', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.CurrentAlarm.{i}.AlarmRaisedTime', 'parameter', 'READ_ONLY', 'dateTime', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.CurrentAlarm.{i}.AlarmChangedTime', 'parameter', 'READ_ONLY', 'dateTime', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.CurrentAlarm.{i}.FaultLocation', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.CurrentAlarm.{i}.ManagedObjectInstance', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.CurrentAlarm.{i}.EventType', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.CurrentAlarm.{i}.ProbableCause', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.CurrentAlarm.{i}.SpecificProblem', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.CurrentAlarm.{i}.PerceivedSeverity', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.CurrentAlarm.{i}.AdditionalText', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.CurrentAlarm.{i}.AdditionalInformation', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.ExpeditedEvent.{i}.EventTime', 'parameter', 'READ_ONLY', 'dateTime', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.ExpeditedEvent.{i}.AlarmIdentifier', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.ExpeditedEvent.{i}.NotificationType', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.ExpeditedEvent.{i}.FaultLocation', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.ExpeditedEvent.{i}.ManagedObjectInstance', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.ExpeditedEvent.{i}.EventType', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.ExpeditedEvent.{i}.ProbableCause', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.ExpeditedEvent.{i}.SpecificProblem', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.ExpeditedEvent.{i}.PerceivedSeverity', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.ExpeditedEvent.{i}.AdditionalText', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.ExpeditedEvent.{i}.AdditionalInformation', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.HistoryEvent.{i}.EventTime', 'parameter', 'READ_ONLY', 'dateTime', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.HistoryEvent.{i}.AlarmIdentifier', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.HistoryEvent.{i}.NotificationType', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.HistoryEvent.{i}.FaultLocation', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.HistoryEvent.{i}.ManagedObjectInstance', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.HistoryEvent.{i}.EventType', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.HistoryEvent.{i}.ProbableCause', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.HistoryEvent.{i}.SpecificProblem', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.HistoryEvent.{i}.PerceivedSeverity', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.HistoryEvent.{i}.AdditionalText', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.HistoryEvent.{i}.AdditionalInformation', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.QueuedEvent.{i}.EventTime', 'parameter', 'READ_ONLY', 'dateTime', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.QueuedEvent.{i}.AlarmIdentifier', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.QueuedEvent.{i}.NotificationType', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.QueuedEvent.{i}.FaultLocation', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.QueuedEvent.{i}.ManagedObjectInstance', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.QueuedEvent.{i}.EventType', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.QueuedEvent.{i}.ProbableCause', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.QueuedEvent.{i}.SpecificProblem', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.QueuedEvent.{i}.PerceivedSeverity', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.QueuedEvent.{i}.AdditionalText', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.QueuedEvent.{i}.AdditionalInformation', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.SupportedAlarm.{i}.EventType', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.SupportedAlarm.{i}.ProbableCause', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.SupportedAlarm.{i}.SpecificProblem', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.SupportedAlarm.{i}.PerceivedSeverity', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.FaultMgmt.SupportedAlarm.{i}.ReportingMechanism', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.LogMgmt.PeriodicUploadEnable', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.LogMgmt.URL', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.LogMgmt.Username', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.LogMgmt.Password', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.LogMgmt.PeriodicUploadInterval', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 1, NULL),
    ('Device.Services.FAPControl.LTE.AdminState', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.LTE.OpState', 'parameter', 'READ_ONLY', 'boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.LTE.RFTxStatus', 'parameter', 'READ_ONLY', 'boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.LTE.Gateway.SecGWServer1', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.LTE.Gateway.SecGWServer2', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.LTE.Gateway.SecGWServer3', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.LTE.Gateway.AGServerEnable', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.LTE.Gateway.AGServerIp1', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.LTE.Gateway.AGServerIp2', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.LTE.Gateway.AGServerIp3', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.LTE.Gateway.AGPort1', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.LTE.Gateway.AGPort2', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.LTE.Gateway.AGPort3', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.PLMNID', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.MMEGroupID', 'parameter', 'READ_ONLY', 'unsignedInt', 'Immediate', 0, 65535),
    ('Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.MMECode', 'parameter', 'READ_ONLY', 'unsignedInt', 'Immediate', 0, 255),
    ('Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.MMEIp1', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.MMEIp2', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.LTE.S1U.{i}.LocIpAddrList', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.LTE.S1U.{i}.FarIpSubnetworkList', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.X2IpAddrMapInfo.{i}.PLMNID', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.X2IpAddrMapInfo.{i}.EnbType', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.X2IpAddrMapInfo.{i}.EnbId', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.X2IpAddrMapInfo.{i}.WanIpAddress', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.X2IpAddrMapInfo.{i}.SubnetMask', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CellRestriction.CellBarred', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CellEnable.AdminState', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.OpState', 'parameter', 'READ_ONLY', 'boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.AccessMgmt.LTE.MaxUEsServed', 'parameter', 'READ_ONLY', 'int', 'Immediate', -1, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.SysInfoCtrlParam.MultiBandInfoListSIB1', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 1, 64),
    ('Device.Services.FAPService.{i}.CellConfig.SysInfoCtrlParam.MultiBandInfoListSIB5', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 1, 64),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RouteIndexList', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RuList', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.UserLabel', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.EARFCNDL', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PhyCellID', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.DLBandwidth', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ULBandwidth', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PSCHPowerOffset', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.SSCHPowerOffset', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PBCHPowerOffset', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.EARFCNUL', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.FreqBandIndicator', 'parameter', 'READ_WRITE', 'int', 'Immediate', 1, 40),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ReferenceSignalPower', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.CellIdentity', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 268435455),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.EnbType', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 1),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.VoLTEParam.SPSSwitchQCI1Ul', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.CAParam.CASwitchUl', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.CAParam.CASwitchDl', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T300', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T301', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T302', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 1, 16),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304EUTRA', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304IRAT', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T310', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T311', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T320', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N310', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N311', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.Capabilities.LTE.NNSFSupported', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.Capabilities.LTE.UeInactiveTimer', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 65535),
    ('Device.Services.FAPService.{i}.CellConfig.Capabilities.LTE.SupportActiveRRCNumbers', 'parameter', 'READ_ONLY', 'unsignedInt', 'Immediate', 0, 65535),
    ('Device.Services.FAPService.{i}.CellConfig.Capabilities.MaxTxPower', 'parameter', 'READ_ONLY', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.EPC.EAID', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 16777216),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.EPC.TAC', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 65535),
    ('Device.Services.FAPControl.Transport.SCTP.Enable', 'parameter', 'READ_WRITE', 'Boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.Transport.SCTP.RTOInitial', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.Transport.SCTP.RTOMin', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.Transport.SCTP.RTOMax', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.Transport.SCTP.MaxInitRetransmits', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.Transport.SCTP.HBInterval', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 1, NULL),
    ('Device.Services.FAPControl.Transport.SCTP.MaxPathRetransmits', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.Transport.SCTP.MaxAssociationRetransmits', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.Transport.SCTP.ValCookieLife', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.Transport.SCTP.Assoc.{i}.SCTPAssocLocalAddr', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.Transport.SCTP.Assoc.{i}.LocalPort', 'parameter', 'READ_ONLY', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.Transport.SCTP.Assoc.{i}.PrimaryPeerAddress', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPControl.Transport.SCTP.Assoc.{i}.RemotePort', 'parameter', 'READ_ONLY', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.NumberOfRaPreambles', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.SizeOfRaGroupA', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MessageSizeGroupA', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MessagePowerOffsetGroupB', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.PowerRampingStep', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.PreambleInitialReceivedTargetPower', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.PreambleTransMax', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.ResponseWindowSize', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.ContentionResolutionTimer', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MaxHARQMsg3Tx', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.DRX.DRXEnabled', 'parameter', 'READ_WRITE', 'Boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.MaxHARQTx', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.PeriodicBSRTimer', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.RetxBSRTimer', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.TTIBundling', 'parameter', 'READ_WRITE', 'Boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.MaxUePerUlSf', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 65535),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.ConfigurationIndex', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.FreqOffset', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.HighSpeedFlag', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.RootSequenceIndex', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.ZeroCorrelationZoneConfig', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSEnabled', 'parameter', 'READ_WRITE', 'Boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSBandwidthConfig', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSMaxUpPTS', 'parameter', 'READ_WRITE', 'Boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.AckNackSRSSimultaneousTransmission', 'parameter', 'READ_WRITE', 'Boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.DeltaPUCCHShift', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.NRBCQI', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.NCSAN', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 7),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.N1PUCCHAN', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.CQIPUCCHResourceIndex', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.K', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 1, 4),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PaParam.PUSCHPowerCtrlSwitch', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 1),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PaParam.PUCCHPowerCtrlSwitch', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 1),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.Enable64QAM', 'parameter', 'READ_WRITE', 'Boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.HoppingMode', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.HoppingOffset', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.NSB', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 1, 4),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRS.NumPRSResourceBlocks', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRS.PRSConfigurationIndex', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 4095),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRS.NumConsecutivePRSSubfames', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.TDDFrame.SpecialSubframePatterns', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 8),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.TDDFrame.SubFrameAssignment', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 6),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PDSCH.Pb', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PDSCH.Pa', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.P0NominalPUSCHPersistent', 'parameter', 'READ_WRITE', 'Int', 'Immediate', -126, 24),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.P0NominalPUSCH', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.Alpha', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.P0NominalPUCCH', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.DeltaMCSEnabled', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 1),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.Antenna.NumOfTxAntenna', 'parameter', 'READ_ONLY', 'unsignedInt', 'Immediate', 1, 7),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.Antenna.NumOfRxAntenna', 'parameter', 'READ_ONLY', 'unsignedInt', 'Immediate', 1, 7),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.NeighCellConfig', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 3),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.MeasureCtrl.Smeasure', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 97),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.QoffsetGERAN', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.MeasQuantityUTRAFDD', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.MeasQuantityGERAN', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.QoffsetUTRA', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.Qhyst', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.IntraFreqReselection', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.QHystSFMedium', 'parameter', 'READ_WRITE', 'int', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.QHystSFHigh', 'parameter', 'READ_WRITE', 'int', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.TEvaluation', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.THystNormal', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.NCellChangeMedium', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 1, 16),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.NCellChangeHigh', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 1, 16),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QRxLevMinSIB1', 'parameter', 'READ_WRITE', 'int', 'Immediate', -70, -22),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QRxLevMinSIB3', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QRxLevMinOffset', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 1, 8),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SIntraSearch', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRA', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SNonIntraSearch', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SNonIntraSearchPR9', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 31),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SNonIntraSearchQR9', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 31),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.CellReselectionPriority', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 7),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.PMax', 'parameter', 'READ_WRITE', 'int', 'Immediate', -30, 33),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.ThreshServingLow', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 31),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.ThreshServingLowQR9', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 31),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRASFMedium', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRASFHigh', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SIntraSearchPR9', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 31),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SIntraSearchQR9', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 31),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QQualMinR9Reselection', 'parameter', 'READ_WRITE', 'int', 'Immediate', -34, -3),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QQualMinR9Selection', 'parameter', 'READ_WRITE', 'int', 'Immediate', -34, -3),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QQualMinOffsetR9', 'parameter', 'READ_WRITE', 'int', 'Immediate', 1, 8),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.AllowedMeasBandwidth', 'parameter', 'READ_WRITE', 'unsignedInt(6,15', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.TReselectionUTRA', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.TReselectionGERAN', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SONSysMode', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SONWorkMode', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.PCIOptEnable', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.PCIReconfigWaitTime', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.CandidateARFCNList', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.CandidatePCIList', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANREnable', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANRInterFeqEnable', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANRGERANEnable', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANRUTRANEnable', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ARFCNEnable', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MaxLTENeighbourCellNum', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MaxUTRANNeighbourCellNum', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MaxGRANNeighbourCellNum', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ReSynCellEnable', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.PowerEnable', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.LTESnifferFreqBandList', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.LTESnifferChannelList', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.GERANSnifferEnable', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.GERANSnifferChannelList', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.UTRANSnifferEnable', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.UTRANSnifferChannelList', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MROEnable', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SHEnable', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SyncMode', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.Services.FAPService.{i}.FAPControl.SelfConfig.Startup.Stage', 'parameter', 'READ_ONLY', 'unsignedInt', 'Immediate', 1, 3),
    ('Device.Services.FAPService.{i}.FAPControl.SelfConfig.Startup.Status', 'parameter', 'READ_ONLY', 'unsignedInt', 'Immediate', 1, 2),
    ('Device.Services.FAPService.{i}.FAPControl.SelfConfig.Startup.FailureCause', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.Ethernet.Interface.{i}.Enable', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.Ethernet.Interface.{i}.UserLabel', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Ethernet.Interface.{i}.Name', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.Ethernet.Interface.{i}.Status', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.Ethernet.Interface.{i}.MACAddress', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.Ethernet.Interface.{i}.MaxBitRate', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Ethernet.Interface.{i}.SignTransMedia', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.Ethernet.Interface.{i}.DuplexMode', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Ethernet.Interface.{i}.PortLocation', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.Ethernet.IpRoute.{i}.IpVer', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 1, 2),
    ('Device.Ethernet.IpRoute.{i}.DstIpNetwork', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Ethernet.IpRoute.{i}.PrefixLength', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 0, 128),
    ('Device.Ethernet.IpRoute.{i}.GatewayIpAddress', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Ethernet.IpRoute.{i}.InterfaceName', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.IPsec.Enable', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.IPsec.MyKeyMode', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.IPsec.Status', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.IPsec.AHSupported', 'parameter', 'READ_ONLY', 'boolean', 'Immediate', NULL, NULL),
    ('Device.IPsec.IKEv2SupportedEncryptionAlgorithms', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.IPsec.ESPSupportedEncryptionAlgorithms', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.IPsec.IKEv2SupportedPseudoRandomFunctions', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.IPsec.SupportedIntegrityAlgorithms', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.IPsec.SupportedDiffieHellmanGroupTransforms', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.Time.Enable', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.Time.NTPServer1', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Time.NTPServer2', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Time.NTPServer3', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Time.NTPServer4', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Time.NTPServer5', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.Time.CurrentLocalTime', 'parameter', 'READ_ONLY', 'dateTime', 'Immediate', NULL, NULL),
    ('Device.Time.LocalTimeZone', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.FAP.GPS.LockedLatitude', 'parameter', 'READ_ONLY', 'int', 'Immediate', -90000000, 90000000),
    ('Device.FAP.GPS.LockedLongitude', 'parameter', 'READ_ONLY', 'int', 'Immediate', -180000000, 180000000),
    ('Device.FAP.GPS.NumberOfSatellites', 'parameter', 'READ_ONLY', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.FAP.MRMgmt.Config.{i}.MrEnable', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.FAP.MRMgmt.Config.{i}.MrUrl', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.FAP.MRMgmt.Config.{i}.MrUsername', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.FAP.MRMgmt.Config.{i}.MrPassword', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.FAP.MRMgmt.Config.{i}.MeasureType', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.FAP.MRMgmt.Config.{i}.OmcName', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.FAP.MRMgmt.Config.{i}.SamplePeriod', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.FAP.MRMgmt.Config.{i}.UploadPeriod', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.FAP.MRMgmt.Config.{i}.SampleBeginTime', 'parameter', 'READ_WRITE', 'dateTime', 'Immediate', NULL, NULL),
    ('Device.FAP.MRMgmt.Config.{i}.SampleEndTime', 'parameter', 'READ_WRITE', 'dateTime', 'Immediate', NULL, NULL),
    ('Device.FAP.MRMgmt.Config.{i}.PrbNum', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.FAP.MRMgmt.Config.{i}.SubFrameNum', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.FAP.MRMgmt.Config.{i}.MRECGIList', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.FAP.MRMgmt.Config.{i}.MeasureItems', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.FAP.PerfMgmt.Config.{i}.Enable', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.FAP.PerfMgmt.Config.{i}.Alias', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.FAP.PerfMgmt.Config.{i}.URL', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.FAP.PerfMgmt.Config.{i}.Username', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.FAP.PerfMgmt.Config.{i}.Password', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.FAP.PerfMgmt.Config.{i}.PeriodicUploadInterval', 'parameter', 'READ_WRITE', 'unsignedInt', 'Immediate', 1, 65535),
    ('Device.FAP.PerfMgmt.Config.{i}.PeriodicUploadTime', 'parameter', 'READ_WRITE', 'dateTime', 'Immediate', NULL, NULL),
    ('Device.FAP.PerfMgmt.Config.{i}.ReplenishEnable', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.FAP.PerfMgmt.Config.{i}.ReplenishStartTime', 'parameter', 'READ_WRITE', 'dateTime', 'Immediate', NULL, NULL),
    ('Device.FAP.PerfMgmt.Config.{i}.ReplenishEndTime', 'parameter', 'READ_WRITE', 'dateTime', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.UserLabel', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.DnPrefix', 'parameter', 'READ_WRITE', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.ManufacturerOUI', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.Manufacturer', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.ModelName', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.VendorUnitFamilyType', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.VendorUnitTypeNumber', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.SerialNumber', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.HardwareVersion', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.SoftwareVersion', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.HardwarePlatform', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.AdditionalHardwareVersion', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.AdditionalSoftwareVersion', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.ProvisioningCode', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.ProductClass', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.Status', 'parameter', 'READ_ONLY', 'unsignedInt', 'Immediate', 1, 3),
    ('Device.DeviceInfo.MU.{i}.Reboot', 'parameter', 'READ_WRITE', 'boolean', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.UpTime', 'parameter', 'READ_ONLY', 'unsignedInt', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.FirstUseDate', 'parameter', 'READ_ONLY', 'dateTime', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.ClockSource', 'parameter', 'READ_ONLY', 'unsignedInt[1:11', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.DateOfLastService', 'parameter', 'READ_ONLY', 'dateTime', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.DateOfManufacture', 'parameter', 'READ_ONLY', 'dateTime', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.ManufacturerData', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.SlotsInformation', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.SwUpgrade.Stage', 'parameter', 'READ_ONLY', 'unsignedInt', 'Immediate', 1, 5),
    ('Device.DeviceInfo.MU.{i}.SwUpgrade.FailureCause', 'parameter', 'READ_ONLY', 'string', 'Immediate', NULL, NULL),
    ('Device.DeviceInfo.MU.{i}.SwUpgrade.Status', 'parameter', 'READ_ONLY', 'unsignedInt', 'Immediate', 1, 3)
ON CONFLICT (standard_path) DO UPDATE SET
    entry_type     = EXCLUDED.entry_type,
    access         = EXCLUDED.access,
    data_type      = EXCLUDED.data_type,
    change_applies = EXCLUDED.change_applies,
    min_value      = EXCLUDED.min_value,
    max_value      = EXCLUDED.max_value,
    updated_at     = NOW();

-- Section B: mml_commands — 新增 99 条 + 修改 0 条
WITH chapter_groups AS (
    SELECT group_code, id FROM mml_command_groups
     WHERE param_version = 'cmcc-td-lte-v2.3'
       AND group_code LIKE 'chapter:%'
)
INSERT INTO mml_commands (
    command_name, command_code, category, description,
    rpc_method, operation_type, target_paths, target_object,
    group_id, command_name_i18n,
    logical_code, logical_name_i18n,
    source, catalog_protected, help_doc
) VALUES
    ('列出 设备基本信息', 'LST DEVICE_INFO', '3', '列出 设备基本信息', 'GetParameterValues', 'LST', '["Device.DeviceInfo.UserLabel","Device.DeviceInfo.DnPrefix","Device.DeviceInfo.ManufacturerOUI","Device.DeviceInfo.Manufacturer","Device.DeviceInfo.ModelName","Device.DeviceInfo.SerialNumber","Device.DeviceInfo.HardwareVersion","Device.DeviceInfo.SoftwareVersion","Device.DeviceInfo.HardwarePlatform","Device.DeviceInfo.AdditionalHardwareVersion","Device.DeviceInfo.AdditionalSoftwareVersion","Device.DeviceInfo.ProvisioningCode","Device.DeviceInfo.ProductClass","Device.DeviceInfo.UpTime","Device.DeviceInfo.3GPPSpecVersion","Device.DeviceInfo.FirstUseDate","Device.DeviceInfo.DataModelSpecVersion"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SA'), '{"en":"List 设备基本信息","zh":"列出 设备基本信息"}'::jsonb,
     'DEVICE_INFO', '{"en":"设备基本信息","zh":"设备基本信息"}'::jsonb,
     'standard', true, ''),
    ('修改 设备基本信息', 'MOD DEVICE_INFO', '3', '修改 设备基本信息', 'SetParameterValues', 'MOD', '["Device.DeviceInfo.UserLabel","Device.DeviceInfo.DnPrefix"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SA'), '{"en":"Modify 设备基本信息","zh":"修改 设备基本信息"}'::jsonb,
     'DEVICE_INFO', '{"en":"设备基本信息","zh":"设备基本信息"}'::jsonb,
     'standard', true, ''),
    ('列出 设备软件升级状态', 'LST DEVICE_INFO_SW_UPGRADE', '3', '列出 设备软件升级状态', 'GetParameterValues', 'LST', '["Device.DeviceInfo.SwUpgrade.Stage","Device.DeviceInfo.SwUpgrade.FailureCause","Device.DeviceInfo.SwUpgrade.Status"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SA'), '{"en":"List 设备软件升级状态","zh":"列出 设备软件升级状态"}'::jsonb,
     'DEVICE_INFO_SW_UPGRADE', '{"en":"设备软件升级状态","zh":"设备软件升级状态"}'::jsonb,
     'standard', true, ''),
    ('列出 软件版本控制', 'LST SOFTWARE_CTRL', '7', '列出 软件版本控制', 'GetParameterValues', 'LST', '["Device.SoftwareCtrl.AutoActivateEnable","Device.SoftwareCtrl.ActivateTime","Device.SoftwareCtrl.ActivateEnable","Device.SoftwareCtrl.SystemCurrentVersion","Device.SoftwareCtrl.SystemBackupVersion"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SB'), '{"en":"List 软件版本控制","zh":"列出 软件版本控制"}'::jsonb,
     'SOFTWARE_CTRL', '{"en":"软件版本控制","zh":"软件版本控制"}'::jsonb,
     'standard', true, ''),
    ('修改 软件版本控制', 'MOD SOFTWARE_CTRL', '7', '修改 软件版本控制', 'SetParameterValues', 'MOD', '["Device.SoftwareCtrl.AutoActivateEnable","Device.SoftwareCtrl.ActivateTime","Device.SoftwareCtrl.ActivateEnable"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SB'), '{"en":"Modify 软件版本控制","zh":"修改 软件版本控制"}'::jsonb,
     'SOFTWARE_CTRL', '{"en":"软件版本控制","zh":"软件版本控制"}'::jsonb,
     'standard', true, ''),
    ('列出 基站网管连接', 'LST MANAGEMENT_SERVER', '6', '列出 基站网管连接', 'GetParameterValues', 'LST', '["Device.ManagementServer.URL","Device.ManagementServer.Username","Device.ManagementServer.Password","Device.ManagementServer.PeriodicInformEnable","Device.ManagementServer.PeriodicInformTime","Device.ManagementServer.PeriodicInformInterval","Device.ManagementServer.ParameterKey","Device.ManagementServer.ConnectionRequestURL","Device.ManagementServer.ConnectionRequestUsername","Device.ManagementServer.ConnectionRequestPassword","Device.ManagementServer.UDPConnectionRequestAddress","Device.ManagementServer.STUNEnable","Device.ManagementServer.STUNServerAddress","Device.ManagementServer.STUNServerPort","Device.ManagementServer.STUNUsername","Device.ManagementServer.STUNPassword","Device.ManagementServer.STUNMaximumKeepAlivePeriod","Device.ManagementServer.STUNMinimumKeepAlivePeriod","Device.ManagementServer.NATDetected"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SC'), '{"en":"List 基站网管连接","zh":"列出 基站网管连接"}'::jsonb,
     'MANAGEMENT_SERVER', '{"en":"基站网管连接","zh":"基站网管连接"}'::jsonb,
     'standard', true, ''),
    ('修改 基站网管连接', 'MOD MANAGEMENT_SERVER', '6', '修改 基站网管连接', 'SetParameterValues', 'MOD', '["Device.ManagementServer.URL","Device.ManagementServer.Username","Device.ManagementServer.Password","Device.ManagementServer.PeriodicInformEnable","Device.ManagementServer.PeriodicInformTime","Device.ManagementServer.PeriodicInformInterval","Device.ManagementServer.ConnectionRequestUsername","Device.ManagementServer.ConnectionRequestPassword","Device.ManagementServer.STUNEnable","Device.ManagementServer.STUNServerAddress","Device.ManagementServer.STUNServerPort","Device.ManagementServer.STUNUsername","Device.ManagementServer.STUNPassword","Device.ManagementServer.STUNMaximumKeepAlivePeriod","Device.ManagementServer.STUNMinimumKeepAlivePeriod"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SC'), '{"en":"Modify 基站网管连接","zh":"修改 基站网管连接"}'::jsonb,
     'MANAGEMENT_SERVER', '{"en":"基站网管连接","zh":"基站网管连接"}'::jsonb,
     'standard', true, ''),
    ('列出 故障管理总览', 'LST FAULT_MGMT', '4', '列出 故障管理总览', 'GetParameterValues', 'LST', '["Device.FaultMgmt.SupportedAlarmNumberOfEntries","Device.FaultMgmt.MaxCurrentAlarmEntries","Device.FaultMgmt.CurrentAlarmNumberOfEntries","Device.FaultMgmt.HistoryEventNumberOfEntries","Device.FaultMgmt.ExpeditedEventNumberOfEntries","Device.FaultMgmt.QueuedEventNumberOfEntries"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SD'), '{"en":"List 故障管理总览","zh":"列出 故障管理总览"}'::jsonb,
     'FAULT_MGMT', '{"en":"故障管理总览","zh":"故障管理总览"}'::jsonb,
     'standard', true, ''),
    ('列出 当前告警实例', 'LST FAULT_MGMT_CURRENT_ALARM', '4', '列出 当前告警实例', 'GetParameterValues', 'LST', '["Device.FaultMgmt.CurrentAlarm.{i}.AlarmIdentifier","Device.FaultMgmt.CurrentAlarm.{i}.AlarmRaisedTime","Device.FaultMgmt.CurrentAlarm.{i}.AlarmChangedTime","Device.FaultMgmt.CurrentAlarm.{i}.FaultLocation","Device.FaultMgmt.CurrentAlarm.{i}.ManagedObjectInstance","Device.FaultMgmt.CurrentAlarm.{i}.EventType","Device.FaultMgmt.CurrentAlarm.{i}.ProbableCause","Device.FaultMgmt.CurrentAlarm.{i}.SpecificProblem","Device.FaultMgmt.CurrentAlarm.{i}.PerceivedSeverity","Device.FaultMgmt.CurrentAlarm.{i}.AdditionalText","Device.FaultMgmt.CurrentAlarm.{i}.AdditionalInformation"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SD'), '{"en":"List 当前告警实例","zh":"列出 当前告警实例"}'::jsonb,
     'FAULT_MGMT_CURRENT_ALARM', '{"en":"当前告警实例","zh":"当前告警实例"}'::jsonb,
     'standard', true, ''),
    ('增加 当前告警实例', 'ADD FAULT_MGMT_CURRENT_ALARM', '4', '增加 当前告警实例', 'AddObject', 'ADD', '["Device.FaultMgmt.CurrentAlarm."]'::jsonb, 'Device.FaultMgmt.CurrentAlarm.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SD'), '{"en":"Add 当前告警实例","zh":"增加 当前告警实例"}'::jsonb,
     'FAULT_MGMT_CURRENT_ALARM', '{"en":"当前告警实例","zh":"当前告警实例"}'::jsonb,
     'standard', true, ''),
    ('删除 当前告警实例', 'RMV FAULT_MGMT_CURRENT_ALARM', '4', '删除 当前告警实例', 'DeleteObject', 'RMV', '["Device.FaultMgmt.CurrentAlarm."]'::jsonb, 'Device.FaultMgmt.CurrentAlarm.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SD'), '{"en":"Remove 当前告警实例","zh":"删除 当前告警实例"}'::jsonb,
     'FAULT_MGMT_CURRENT_ALARM', '{"en":"当前告警实例","zh":"当前告警实例"}'::jsonb,
     'standard', true, ''),
    ('列出 实时告警实例', 'LST FAULT_MGMT_EXPEDITED_EVENT', '4', '列出 实时告警实例', 'GetParameterValues', 'LST', '["Device.FaultMgmt.ExpeditedEvent.{i}.EventTime","Device.FaultMgmt.ExpeditedEvent.{i}.AlarmIdentifier","Device.FaultMgmt.ExpeditedEvent.{i}.NotificationType","Device.FaultMgmt.ExpeditedEvent.{i}.FaultLocation","Device.FaultMgmt.ExpeditedEvent.{i}.ManagedObjectInstance","Device.FaultMgmt.ExpeditedEvent.{i}.EventType","Device.FaultMgmt.ExpeditedEvent.{i}.ProbableCause","Device.FaultMgmt.ExpeditedEvent.{i}.SpecificProblem","Device.FaultMgmt.ExpeditedEvent.{i}.PerceivedSeverity","Device.FaultMgmt.ExpeditedEvent.{i}.AdditionalText","Device.FaultMgmt.ExpeditedEvent.{i}.AdditionalInformation"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SD'), '{"en":"List 实时告警实例","zh":"列出 实时告警实例"}'::jsonb,
     'FAULT_MGMT_EXPEDITED_EVENT', '{"en":"实时告警实例","zh":"实时告警实例"}'::jsonb,
     'standard', true, ''),
    ('增加 实时告警实例', 'ADD FAULT_MGMT_EXPEDITED_EVENT', '4', '增加 实时告警实例', 'AddObject', 'ADD', '["Device.FaultMgmt.ExpeditedEvent."]'::jsonb, 'Device.FaultMgmt.ExpeditedEvent.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SD'), '{"en":"Add 实时告警实例","zh":"增加 实时告警实例"}'::jsonb,
     'FAULT_MGMT_EXPEDITED_EVENT', '{"en":"实时告警实例","zh":"实时告警实例"}'::jsonb,
     'standard', true, ''),
    ('删除 实时告警实例', 'RMV FAULT_MGMT_EXPEDITED_EVENT', '4', '删除 实时告警实例', 'DeleteObject', 'RMV', '["Device.FaultMgmt.ExpeditedEvent."]'::jsonb, 'Device.FaultMgmt.ExpeditedEvent.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SD'), '{"en":"Remove 实时告警实例","zh":"删除 实时告警实例"}'::jsonb,
     'FAULT_MGMT_EXPEDITED_EVENT', '{"en":"实时告警实例","zh":"实时告警实例"}'::jsonb,
     'standard', true, ''),
    ('列出 历史告警实例', 'LST FAULT_MGMT_HISTORY_EVENT', '4', '列出 历史告警实例', 'GetParameterValues', 'LST', '["Device.FaultMgmt.HistoryEvent.{i}.EventTime","Device.FaultMgmt.HistoryEvent.{i}.AlarmIdentifier","Device.FaultMgmt.HistoryEvent.{i}.NotificationType","Device.FaultMgmt.HistoryEvent.{i}.FaultLocation","Device.FaultMgmt.HistoryEvent.{i}.ManagedObjectInstance","Device.FaultMgmt.HistoryEvent.{i}.EventType","Device.FaultMgmt.HistoryEvent.{i}.ProbableCause","Device.FaultMgmt.HistoryEvent.{i}.SpecificProblem","Device.FaultMgmt.HistoryEvent.{i}.PerceivedSeverity","Device.FaultMgmt.HistoryEvent.{i}.AdditionalText","Device.FaultMgmt.HistoryEvent.{i}.AdditionalInformation"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SD'), '{"en":"List 历史告警实例","zh":"列出 历史告警实例"}'::jsonb,
     'FAULT_MGMT_HISTORY_EVENT', '{"en":"历史告警实例","zh":"历史告警实例"}'::jsonb,
     'standard', true, ''),
    ('增加 历史告警实例', 'ADD FAULT_MGMT_HISTORY_EVENT', '4', '增加 历史告警实例', 'AddObject', 'ADD', '["Device.FaultMgmt.HistoryEvent."]'::jsonb, 'Device.FaultMgmt.HistoryEvent.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SD'), '{"en":"Add 历史告警实例","zh":"增加 历史告警实例"}'::jsonb,
     'FAULT_MGMT_HISTORY_EVENT', '{"en":"历史告警实例","zh":"历史告警实例"}'::jsonb,
     'standard', true, ''),
    ('删除 历史告警实例', 'RMV FAULT_MGMT_HISTORY_EVENT', '4', '删除 历史告警实例', 'DeleteObject', 'RMV', '["Device.FaultMgmt.HistoryEvent."]'::jsonb, 'Device.FaultMgmt.HistoryEvent.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SD'), '{"en":"Remove 历史告警实例","zh":"删除 历史告警实例"}'::jsonb,
     'FAULT_MGMT_HISTORY_EVENT', '{"en":"历史告警实例","zh":"历史告警实例"}'::jsonb,
     'standard', true, ''),
    ('列出 队列告警实例', 'LST FAULT_MGMT_QUEUED_EVENT', '4', '列出 队列告警实例', 'GetParameterValues', 'LST', '["Device.FaultMgmt.QueuedEvent.{i}.EventTime","Device.FaultMgmt.QueuedEvent.{i}.AlarmIdentifier","Device.FaultMgmt.QueuedEvent.{i}.NotificationType","Device.FaultMgmt.QueuedEvent.{i}.FaultLocation","Device.FaultMgmt.QueuedEvent.{i}.ManagedObjectInstance","Device.FaultMgmt.QueuedEvent.{i}.EventType","Device.FaultMgmt.QueuedEvent.{i}.ProbableCause","Device.FaultMgmt.QueuedEvent.{i}.SpecificProblem","Device.FaultMgmt.QueuedEvent.{i}.PerceivedSeverity","Device.FaultMgmt.QueuedEvent.{i}.AdditionalText","Device.FaultMgmt.QueuedEvent.{i}.AdditionalInformation"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SD'), '{"en":"List 队列告警实例","zh":"列出 队列告警实例"}'::jsonb,
     'FAULT_MGMT_QUEUED_EVENT', '{"en":"队列告警实例","zh":"队列告警实例"}'::jsonb,
     'standard', true, ''),
    ('增加 队列告警实例', 'ADD FAULT_MGMT_QUEUED_EVENT', '4', '增加 队列告警实例', 'AddObject', 'ADD', '["Device.FaultMgmt.QueuedEvent."]'::jsonb, 'Device.FaultMgmt.QueuedEvent.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SD'), '{"en":"Add 队列告警实例","zh":"增加 队列告警实例"}'::jsonb,
     'FAULT_MGMT_QUEUED_EVENT', '{"en":"队列告警实例","zh":"队列告警实例"}'::jsonb,
     'standard', true, ''),
    ('删除 队列告警实例', 'RMV FAULT_MGMT_QUEUED_EVENT', '4', '删除 队列告警实例', 'DeleteObject', 'RMV', '["Device.FaultMgmt.QueuedEvent."]'::jsonb, 'Device.FaultMgmt.QueuedEvent.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SD'), '{"en":"Remove 队列告警实例","zh":"删除 队列告警实例"}'::jsonb,
     'FAULT_MGMT_QUEUED_EVENT', '{"en":"队列告警实例","zh":"队列告警实例"}'::jsonb,
     'standard', true, ''),
    ('列出 支持告警类型', 'LST FAULT_MGMT_SUPPORTED_ALARM', '4', '列出 支持告警类型', 'GetParameterValues', 'LST', '["Device.FaultMgmt.SupportedAlarm.{i}.EventType","Device.FaultMgmt.SupportedAlarm.{i}.ProbableCause","Device.FaultMgmt.SupportedAlarm.{i}.SpecificProblem","Device.FaultMgmt.SupportedAlarm.{i}.PerceivedSeverity","Device.FaultMgmt.SupportedAlarm.{i}.ReportingMechanism"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SD'), '{"en":"List 支持告警类型","zh":"列出 支持告警类型"}'::jsonb,
     'FAULT_MGMT_SUPPORTED_ALARM', '{"en":"支持告警类型","zh":"支持告警类型"}'::jsonb,
     'standard', true, ''),
    ('修改 支持告警类型', 'MOD FAULT_MGMT_SUPPORTED_ALARM', '4', '修改 支持告警类型', 'SetParameterValues', 'MOD', '["Device.FaultMgmt.SupportedAlarm.{i}.ReportingMechanism"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SD'), '{"en":"Modify 支持告警类型","zh":"修改 支持告警类型"}'::jsonb,
     'FAULT_MGMT_SUPPORTED_ALARM', '{"en":"支持告警类型","zh":"支持告警类型"}'::jsonb,
     'standard', true, ''),
    ('增加 支持告警类型', 'ADD FAULT_MGMT_SUPPORTED_ALARM', '4', '增加 支持告警类型', 'AddObject', 'ADD', '["Device.FaultMgmt.SupportedAlarm."]'::jsonb, 'Device.FaultMgmt.SupportedAlarm.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SD'), '{"en":"Add 支持告警类型","zh":"增加 支持告警类型"}'::jsonb,
     'FAULT_MGMT_SUPPORTED_ALARM', '{"en":"支持告警类型","zh":"支持告警类型"}'::jsonb,
     'standard', true, ''),
    ('删除 支持告警类型', 'RMV FAULT_MGMT_SUPPORTED_ALARM', '4', '删除 支持告警类型', 'DeleteObject', 'RMV', '["Device.FaultMgmt.SupportedAlarm."]'::jsonb, 'Device.FaultMgmt.SupportedAlarm.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SD'), '{"en":"Remove 支持告警类型","zh":"删除 支持告警类型"}'::jsonb,
     'FAULT_MGMT_SUPPORTED_ALARM', '{"en":"支持告警类型","zh":"支持告警类型"}'::jsonb,
     'standard', true, ''),
    ('列出 日志管理配置', 'LST LOG_MGMT', '8', '列出 日志管理配置', 'GetParameterValues', 'LST', '["Device.LogMgmt.PeriodicUploadEnable","Device.LogMgmt.URL","Device.LogMgmt.Username","Device.LogMgmt.Password","Device.LogMgmt.PeriodicUploadInterval"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SE'), '{"en":"List 日志管理配置","zh":"列出 日志管理配置"}'::jsonb,
     'LOG_MGMT', '{"en":"日志管理配置","zh":"日志管理配置"}'::jsonb,
     'standard', true, ''),
    ('修改 日志管理配置', 'MOD LOG_MGMT', '8', '修改 日志管理配置', 'SetParameterValues', 'MOD', '["Device.LogMgmt.PeriodicUploadEnable","Device.LogMgmt.URL","Device.LogMgmt.Username","Device.LogMgmt.Password","Device.LogMgmt.PeriodicUploadInterval"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SE'), '{"en":"Modify 日志管理配置","zh":"修改 日志管理配置"}'::jsonb,
     'LOG_MGMT', '{"en":"日志管理配置","zh":"日志管理配置"}'::jsonb,
     'standard', true, ''),
    ('列出 LTE 接入控制', 'LST LTE', '1', '列出 LTE 接入控制', 'GetParameterValues', 'LST', '["Device.Services.FAPControl.LTE.AdminState","Device.Services.FAPControl.LTE.OpState","Device.Services.FAPControl.LTE.RFTxStatus"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SF'), '{"en":"List LTE 接入控制","zh":"列出 LTE 接入控制"}'::jsonb,
     'LTE', '{"en":"LTE 接入控制","zh":"LTE 接入控制"}'::jsonb,
     'standard', true, ''),
    ('修改 LTE 接入控制', 'MOD LTE', '1', '修改 LTE 接入控制', 'SetParameterValues', 'MOD', '["Device.Services.FAPControl.LTE.AdminState"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SF'), '{"en":"Modify LTE 接入控制","zh":"修改 LTE 接入控制"}'::jsonb,
     'LTE', '{"en":"LTE 接入控制","zh":"LTE 接入控制"}'::jsonb,
     'standard', true, ''),
    ('列出 安全接入网关', 'LST LTE_GATEWAY', '1', '列出 安全接入网关', 'GetParameterValues', 'LST', '["Device.Services.FAPControl.LTE.Gateway.SecGWServer1","Device.Services.FAPControl.LTE.Gateway.SecGWServer2","Device.Services.FAPControl.LTE.Gateway.SecGWServer3","Device.Services.FAPControl.LTE.Gateway.AGServerEnable","Device.Services.FAPControl.LTE.Gateway.AGServerIp1","Device.Services.FAPControl.LTE.Gateway.AGServerIp2","Device.Services.FAPControl.LTE.Gateway.AGServerIp3","Device.Services.FAPControl.LTE.Gateway.AGPort1","Device.Services.FAPControl.LTE.Gateway.AGPort2","Device.Services.FAPControl.LTE.Gateway.AGPort3"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SF'), '{"en":"List 安全接入网关","zh":"列出 安全接入网关"}'::jsonb,
     'LTE_GATEWAY', '{"en":"安全接入网关","zh":"安全接入网关"}'::jsonb,
     'standard', true, ''),
    ('修改 安全接入网关', 'MOD LTE_GATEWAY', '1', '修改 安全接入网关', 'SetParameterValues', 'MOD', '["Device.Services.FAPControl.LTE.Gateway.SecGWServer1","Device.Services.FAPControl.LTE.Gateway.SecGWServer2","Device.Services.FAPControl.LTE.Gateway.SecGWServer3","Device.Services.FAPControl.LTE.Gateway.AGServerEnable","Device.Services.FAPControl.LTE.Gateway.AGServerIp1","Device.Services.FAPControl.LTE.Gateway.AGServerIp2","Device.Services.FAPControl.LTE.Gateway.AGServerIp3","Device.Services.FAPControl.LTE.Gateway.AGPort1","Device.Services.FAPControl.LTE.Gateway.AGPort2","Device.Services.FAPControl.LTE.Gateway.AGPort3"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SF'), '{"en":"Modify 安全接入网关","zh":"修改 安全接入网关"}'::jsonb,
     'LTE_GATEWAY', '{"en":"安全接入网关","zh":"安全接入网关"}'::jsonb,
     'standard', true, ''),
    ('列出 MME 池配置', 'LST LTE_MME_POOL_CONFIG_PARAM', '1', '列出 MME 池配置', 'GetParameterValues', 'LST', '["Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.PLMNID","Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.MMEGroupID","Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.MMECode","Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.MMEIp1","Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.MMEIp2"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SF'), '{"en":"List MME 池配置","zh":"列出 MME 池配置"}'::jsonb,
     'LTE_MME_POOL_CONFIG_PARAM', '{"en":"MME 池配置","zh":"MME 池配置"}'::jsonb,
     'standard', true, ''),
    ('修改 MME 池配置', 'MOD LTE_MME_POOL_CONFIG_PARAM', '1', '修改 MME 池配置', 'SetParameterValues', 'MOD', '["Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.MMEIp1","Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.MMEIp2"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SF'), '{"en":"Modify MME 池配置","zh":"修改 MME 池配置"}'::jsonb,
     'LTE_MME_POOL_CONFIG_PARAM', '{"en":"MME 池配置","zh":"MME 池配置"}'::jsonb,
     'standard', true, ''),
    ('增加 MME 池配置', 'ADD LTE_MME_POOL_CONFIG_PARAM', '1', '增加 MME 池配置', 'AddObject', 'ADD', '["Device.Services.FAPControl.LTE.MmePoolConfigParam."]'::jsonb, 'Device.Services.FAPControl.LTE.MmePoolConfigParam.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SF'), '{"en":"Add MME 池配置","zh":"增加 MME 池配置"}'::jsonb,
     'LTE_MME_POOL_CONFIG_PARAM', '{"en":"MME 池配置","zh":"MME 池配置"}'::jsonb,
     'standard', true, ''),
    ('删除 MME 池配置', 'RMV LTE_MME_POOL_CONFIG_PARAM', '1', '删除 MME 池配置', 'DeleteObject', 'RMV', '["Device.Services.FAPControl.LTE.MmePoolConfigParam."]'::jsonb, 'Device.Services.FAPControl.LTE.MmePoolConfigParam.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SF'), '{"en":"Remove MME 池配置","zh":"删除 MME 池配置"}'::jsonb,
     'LTE_MME_POOL_CONFIG_PARAM', '{"en":"MME 池配置","zh":"MME 池配置"}'::jsonb,
     'standard', true, ''),
    ('列出 S1U 用户面接口', 'LST LTE_S1U', '1', '列出 S1U 用户面接口', 'GetParameterValues', 'LST', '["Device.Services.FAPControl.LTE.S1U.{i}.LocIpAddrList","Device.Services.FAPControl.LTE.S1U.{i}.FarIpSubnetworkList"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SF'), '{"en":"List S1U 用户面接口","zh":"列出 S1U 用户面接口"}'::jsonb,
     'LTE_S1U', '{"en":"S1U 用户面接口","zh":"S1U 用户面接口"}'::jsonb,
     'standard', true, ''),
    ('增加 S1U 用户面接口', 'ADD LTE_S1U', '1', '增加 S1U 用户面接口', 'AddObject', 'ADD', '["Device.Services.FAPControl.LTE.S1U."]'::jsonb, 'Device.Services.FAPControl.LTE.S1U.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SF'), '{"en":"Add S1U 用户面接口","zh":"增加 S1U 用户面接口"}'::jsonb,
     'LTE_S1U', '{"en":"S1U 用户面接口","zh":"S1U 用户面接口"}'::jsonb,
     'standard', true, ''),
    ('删除 S1U 用户面接口', 'RMV LTE_S1U', '1', '删除 S1U 用户面接口', 'DeleteObject', 'RMV', '["Device.Services.FAPControl.LTE.S1U."]'::jsonb, 'Device.Services.FAPControl.LTE.S1U.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SF'), '{"en":"Remove S1U 用户面接口","zh":"删除 S1U 用户面接口"}'::jsonb,
     'LTE_S1U', '{"en":"S1U 用户面接口","zh":"S1U 用户面接口"}'::jsonb,
     'standard', true, ''),
    ('列出 X2 接口 IP 映射', 'LST X2_IP_ADDR_MAP_INFO', '1', '列出 X2 接口 IP 映射', 'GetParameterValues', 'LST', '["Device.Services.FAPControl.X2IpAddrMapInfo.{i}.PLMNID","Device.Services.FAPControl.X2IpAddrMapInfo.{i}.EnbType","Device.Services.FAPControl.X2IpAddrMapInfo.{i}.EnbId","Device.Services.FAPControl.X2IpAddrMapInfo.{i}.WanIpAddress","Device.Services.FAPControl.X2IpAddrMapInfo.{i}.SubnetMask"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SF'), '{"en":"List X2 接口 IP 映射","zh":"列出 X2 接口 IP 映射"}'::jsonb,
     'X2_IP_ADDR_MAP_INFO', '{"en":"X2 接口 IP 映射","zh":"X2 接口 IP 映射"}'::jsonb,
     'standard', true, ''),
    ('修改 X2 接口 IP 映射', 'MOD X2_IP_ADDR_MAP_INFO', '1', '修改 X2 接口 IP 映射', 'SetParameterValues', 'MOD', '["Device.Services.FAPControl.X2IpAddrMapInfo.{i}.PLMNID","Device.Services.FAPControl.X2IpAddrMapInfo.{i}.EnbType","Device.Services.FAPControl.X2IpAddrMapInfo.{i}.EnbId","Device.Services.FAPControl.X2IpAddrMapInfo.{i}.WanIpAddress","Device.Services.FAPControl.X2IpAddrMapInfo.{i}.SubnetMask"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SF'), '{"en":"Modify X2 接口 IP 映射","zh":"修改 X2 接口 IP 映射"}'::jsonb,
     'X2_IP_ADDR_MAP_INFO', '{"en":"X2 接口 IP 映射","zh":"X2 接口 IP 映射"}'::jsonb,
     'standard', true, ''),
    ('增加 X2 接口 IP 映射', 'ADD X2_IP_ADDR_MAP_INFO', '1', '增加 X2 接口 IP 映射', 'AddObject', 'ADD', '["Device.Services.FAPControl.X2IpAddrMapInfo."]'::jsonb, 'Device.Services.FAPControl.X2IpAddrMapInfo.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SF'), '{"en":"Add X2 接口 IP 映射","zh":"增加 X2 接口 IP 映射"}'::jsonb,
     'X2_IP_ADDR_MAP_INFO', '{"en":"X2 接口 IP 映射","zh":"X2 接口 IP 映射"}'::jsonb,
     'standard', true, ''),
    ('删除 X2 接口 IP 映射', 'RMV X2_IP_ADDR_MAP_INFO', '1', '删除 X2 接口 IP 映射', 'DeleteObject', 'RMV', '["Device.Services.FAPControl.X2IpAddrMapInfo."]'::jsonb, 'Device.Services.FAPControl.X2IpAddrMapInfo.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SF'), '{"en":"Remove X2 接口 IP 映射","zh":"删除 X2 接口 IP 映射"}'::jsonb,
     'X2_IP_ADDR_MAP_INFO', '{"en":"X2 接口 IP 映射","zh":"X2 接口 IP 映射"}'::jsonb,
     'standard', true, ''),
    ('列出 FAP 载波基本配置', 'LST ', '1', '列出 FAP 载波基本配置', 'GetParameterValues', 'LST', '["Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CellRestriction.CellBarred","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CellEnable.AdminState","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.OpState","Device.Services.FAPService.{i}.CellConfig.AccessMgmt.LTE.MaxUEsServed","Device.Services.FAPService.{i}.CellConfig.SysInfoCtrlParam.MultiBandInfoListSIB1","Device.Services.FAPService.{i}.CellConfig.SysInfoCtrlParam.MultiBandInfoListSIB5","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RouteIndexList","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RuList","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.UserLabel","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.EARFCNDL","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PhyCellID","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.DLBandwidth","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ULBandwidth","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PSCHPowerOffset","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.SSCHPowerOffset","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PBCHPowerOffset","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.EARFCNUL","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.FreqBandIndicator","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ReferenceSignalPower","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.CellIdentity","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.EnbType","Device.Services.FAPService.{i}.CellConfig.LTE.VoLTEParam.SPSSwitchQCI1Ul","Device.Services.FAPService.{i}.CellConfig.LTE.CAParam.CASwitchUl","Device.Services.FAPService.{i}.CellConfig.LTE.CAParam.CASwitchDl","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T300","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T301","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T302","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304EUTRA","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304IRAT","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T310","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T311","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T320","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N310","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N311"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SF'), '{"en":"List FAP 载波基本配置","zh":"列出 FAP 载波基本配置"}'::jsonb,
     '', '{"en":"FAP 载波基本配置","zh":"FAP 载波基本配置"}'::jsonb,
     'standard', true, ''),
    ('修改 FAP 载波基本配置', 'MOD ', '1', '修改 FAP 载波基本配置', 'SetParameterValues', 'MOD', '["Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CellRestriction.CellBarred","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CellEnable.AdminState","Device.Services.FAPService.{i}.CellConfig.SysInfoCtrlParam.MultiBandInfoListSIB1","Device.Services.FAPService.{i}.CellConfig.SysInfoCtrlParam.MultiBandInfoListSIB5","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RouteIndexList","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RuList","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.UserLabel","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.EARFCNDL","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PhyCellID","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.DLBandwidth","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ULBandwidth","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PSCHPowerOffset","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.SSCHPowerOffset","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PBCHPowerOffset","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.EARFCNUL","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.FreqBandIndicator","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ReferenceSignalPower","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.CellIdentity","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.EnbType","Device.Services.FAPService.{i}.CellConfig.LTE.VoLTEParam.SPSSwitchQCI1Ul","Device.Services.FAPService.{i}.CellConfig.LTE.CAParam.CASwitchUl","Device.Services.FAPService.{i}.CellConfig.LTE.CAParam.CASwitchDl","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T300","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T301","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T302","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304EUTRA","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304IRAT","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T310","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T311","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T320","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N310","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N311"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SF'), '{"en":"Modify FAP 载波基本配置","zh":"修改 FAP 载波基本配置"}'::jsonb,
     '', '{"en":"FAP 载波基本配置","zh":"FAP 载波基本配置"}'::jsonb,
     'standard', true, ''),
    ('列出 FAP 载波能力集', 'LST CAPABILITIES', '1', '列出 FAP 载波能力集', 'GetParameterValues', 'LST', '["Device.Services.FAPService.{i}.Capabilities.LTE.NNSFSupported"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SF'), '{"en":"List FAP 载波能力集","zh":"列出 FAP 载波能力集"}'::jsonb,
     'CAPABILITIES', '{"en":"FAP 载波能力集","zh":"FAP 载波能力集"}'::jsonb,
     'standard', true, ''),
    ('修改 FAP 载波能力集', 'MOD CAPABILITIES', '1', '修改 FAP 载波能力集', 'SetParameterValues', 'MOD', '["Device.Services.FAPService.{i}.Capabilities.LTE.NNSFSupported"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SF'), '{"en":"Modify FAP 载波能力集","zh":"修改 FAP 载波能力集"}'::jsonb,
     'CAPABILITIES', '{"en":"FAP 载波能力集","zh":"FAP 载波能力集"}'::jsonb,
     'standard', true, ''),
    ('列出 小区配置能力集', 'LST CELL_CONFIG_CAPABILITIES', '1', '列出 小区配置能力集', 'GetParameterValues', 'LST', '["Device.Services.FAPService.{i}.CellConfig.Capabilities.LTE.UeInactiveTimer","Device.Services.FAPService.{i}.CellConfig.Capabilities.LTE.SupportActiveRRCNumbers","Device.Services.FAPService.{i}.CellConfig.Capabilities.MaxTxPower"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SF'), '{"en":"List 小区配置能力集","zh":"列出 小区配置能力集"}'::jsonb,
     'CELL_CONFIG_CAPABILITIES', '{"en":"小区配置能力集","zh":"小区配置能力集"}'::jsonb,
     'standard', true, ''),
    ('修改 小区配置能力集', 'MOD CELL_CONFIG_CAPABILITIES', '1', '修改 小区配置能力集', 'SetParameterValues', 'MOD', '["Device.Services.FAPService.{i}.CellConfig.Capabilities.LTE.UeInactiveTimer"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SF'), '{"en":"Modify 小区配置能力集","zh":"修改 小区配置能力集"}'::jsonb,
     'CELL_CONFIG_CAPABILITIES', '{"en":"小区配置能力集","zh":"小区配置能力集"}'::jsonb,
     'standard', true, ''),
    ('列出 EPC 核心网参数', 'LST LTE_EPC', '1', '列出 EPC 核心网参数', 'GetParameterValues', 'LST', '["Device.Services.FAPService.{i}.CellConfig.LTE.EPC.EAID","Device.Services.FAPService.{i}.CellConfig.LTE.EPC.TAC"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SF'), '{"en":"List EPC 核心网参数","zh":"列出 EPC 核心网参数"}'::jsonb,
     'LTE_EPC', '{"en":"EPC 核心网参数","zh":"EPC 核心网参数"}'::jsonb,
     'standard', true, ''),
    ('修改 EPC 核心网参数', 'MOD LTE_EPC', '1', '修改 EPC 核心网参数', 'SetParameterValues', 'MOD', '["Device.Services.FAPService.{i}.CellConfig.LTE.EPC.EAID","Device.Services.FAPService.{i}.CellConfig.LTE.EPC.TAC"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SF'), '{"en":"Modify EPC 核心网参数","zh":"修改 EPC 核心网参数"}'::jsonb,
     'LTE_EPC', '{"en":"EPC 核心网参数","zh":"EPC 核心网参数"}'::jsonb,
     'standard', true, ''),
    ('列出 SCTP 协议配置', 'LST TRANSPORT_SCTP', '1', '列出 SCTP 协议配置', 'GetParameterValues', 'LST', '["Device.Services.FAPControl.Transport.SCTP.Enable","Device.Services.FAPControl.Transport.SCTP.RTOInitial","Device.Services.FAPControl.Transport.SCTP.RTOMin","Device.Services.FAPControl.Transport.SCTP.RTOMax","Device.Services.FAPControl.Transport.SCTP.MaxInitRetransmits","Device.Services.FAPControl.Transport.SCTP.HBInterval","Device.Services.FAPControl.Transport.SCTP.MaxPathRetransmits","Device.Services.FAPControl.Transport.SCTP.MaxAssociationRetransmits","Device.Services.FAPControl.Transport.SCTP.ValCookieLife"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SG'), '{"en":"List SCTP 协议配置","zh":"列出 SCTP 协议配置"}'::jsonb,
     'TRANSPORT_SCTP', '{"en":"SCTP 协议配置","zh":"SCTP 协议配置"}'::jsonb,
     'standard', true, ''),
    ('修改 SCTP 协议配置', 'MOD TRANSPORT_SCTP', '1', '修改 SCTP 协议配置', 'SetParameterValues', 'MOD', '["Device.Services.FAPControl.Transport.SCTP.Enable","Device.Services.FAPControl.Transport.SCTP.RTOInitial","Device.Services.FAPControl.Transport.SCTP.RTOMin","Device.Services.FAPControl.Transport.SCTP.RTOMax","Device.Services.FAPControl.Transport.SCTP.MaxInitRetransmits","Device.Services.FAPControl.Transport.SCTP.HBInterval","Device.Services.FAPControl.Transport.SCTP.MaxPathRetransmits","Device.Services.FAPControl.Transport.SCTP.MaxAssociationRetransmits","Device.Services.FAPControl.Transport.SCTP.ValCookieLife"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SG'), '{"en":"Modify SCTP 协议配置","zh":"修改 SCTP 协议配置"}'::jsonb,
     'TRANSPORT_SCTP', '{"en":"SCTP 协议配置","zh":"SCTP 协议配置"}'::jsonb,
     'standard', true, ''),
    ('列出 SCTP 关联状态', 'LST SCTP_ASSOC', '1', '列出 SCTP 关联状态', 'GetParameterValues', 'LST', '["Device.Services.FAPControl.Transport.SCTP.Assoc.{i}.SCTPAssocLocalAddr","Device.Services.FAPControl.Transport.SCTP.Assoc.{i}.LocalPort","Device.Services.FAPControl.Transport.SCTP.Assoc.{i}.PrimaryPeerAddress","Device.Services.FAPControl.Transport.SCTP.Assoc.{i}.RemotePort"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SG'), '{"en":"List SCTP 关联状态","zh":"列出 SCTP 关联状态"}'::jsonb,
     'SCTP_ASSOC', '{"en":"SCTP 关联状态","zh":"SCTP 关联状态"}'::jsonb,
     'standard', true, ''),
    ('增加 SCTP 关联状态', 'ADD SCTP_ASSOC', '1', '增加 SCTP 关联状态', 'AddObject', 'ADD', '["Device.Services.FAPControl.Transport.SCTP.Assoc."]'::jsonb, 'Device.Services.FAPControl.Transport.SCTP.Assoc.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SG'), '{"en":"Add SCTP 关联状态","zh":"增加 SCTP 关联状态"}'::jsonb,
     'SCTP_ASSOC', '{"en":"SCTP 关联状态","zh":"SCTP 关联状态"}'::jsonb,
     'standard', true, ''),
    ('删除 SCTP 关联状态', 'RMV SCTP_ASSOC', '1', '删除 SCTP 关联状态', 'DeleteObject', 'RMV', '["Device.Services.FAPControl.Transport.SCTP.Assoc."]'::jsonb, 'Device.Services.FAPControl.Transport.SCTP.Assoc.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SG'), '{"en":"Remove SCTP 关联状态","zh":"删除 SCTP 关联状态"}'::jsonb,
     'SCTP_ASSOC', '{"en":"SCTP 关联状态","zh":"SCTP 关联状态"}'::jsonb,
     'standard', true, ''),
    ('列出 RAN MAC 协议层', 'LST RAN_MAC', '1', '列出 RAN MAC 协议层', 'GetParameterValues', 'LST', '["Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.NumberOfRaPreambles","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.SizeOfRaGroupA","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MessageSizeGroupA","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MessagePowerOffsetGroupB","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.PowerRampingStep","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.PreambleInitialReceivedTargetPower","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.PreambleTransMax","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.ResponseWindowSize","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.ContentionResolutionTimer","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MaxHARQMsg3Tx","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.DRX.DRXEnabled","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.MaxHARQTx","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.PeriodicBSRTimer","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.RetxBSRTimer","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.TTIBundling","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.MaxUePerUlSf"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SH'), '{"en":"List RAN MAC 协议层","zh":"列出 RAN MAC 协议层"}'::jsonb,
     'RAN_MAC', '{"en":"RAN MAC 协议层","zh":"RAN MAC 协议层"}'::jsonb,
     'standard', true, ''),
    ('修改 RAN MAC 协议层', 'MOD RAN_MAC', '1', '修改 RAN MAC 协议层', 'SetParameterValues', 'MOD', '["Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.NumberOfRaPreambles","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.SizeOfRaGroupA","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MessageSizeGroupA","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MessagePowerOffsetGroupB","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.PowerRampingStep","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.PreambleInitialReceivedTargetPower","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.PreambleTransMax","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.ResponseWindowSize","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.ContentionResolutionTimer","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MaxHARQMsg3Tx","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.DRX.DRXEnabled","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.MaxHARQTx","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.PeriodicBSRTimer","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.RetxBSRTimer","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.TTIBundling","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.MaxUePerUlSf"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SH'), '{"en":"Modify RAN MAC 协议层","zh":"修改 RAN MAC 协议层"}'::jsonb,
     'RAN_MAC', '{"en":"RAN MAC 协议层","zh":"RAN MAC 协议层"}'::jsonb,
     'standard', true, ''),
    ('列出 RAN PHY 物理层', 'LST RAN_PHY', '1', '列出 RAN PHY 物理层', 'GetParameterValues', 'LST', '["Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.ConfigurationIndex","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.FreqOffset","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.HighSpeedFlag","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.RootSequenceIndex","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.ZeroCorrelationZoneConfig","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSEnabled","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSBandwidthConfig","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSMaxUpPTS","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.AckNackSRSSimultaneousTransmission","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.DeltaPUCCHShift","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.NRBCQI","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.NCSAN","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.N1PUCCHAN","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.CQIPUCCHResourceIndex","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.K","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PaParam.PUSCHPowerCtrlSwitch","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PaParam.PUCCHPowerCtrlSwitch","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.Enable64QAM","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.HoppingMode","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.HoppingOffset","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.NSB","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRS.NumPRSResourceBlocks","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRS.PRSConfigurationIndex","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRS.NumConsecutivePRSSubfames","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.TDDFrame.SpecialSubframePatterns","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.TDDFrame.SubFrameAssignment","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PDSCH.Pb","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PDSCH.Pa","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.P0NominalPUSCHPersistent","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.P0NominalPUSCH","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.Alpha","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.P0NominalPUCCH","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.DeltaMCSEnabled","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.Antenna.NumOfTxAntenna","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.Antenna.NumOfRxAntenna"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SH'), '{"en":"List RAN PHY 物理层","zh":"列出 RAN PHY 物理层"}'::jsonb,
     'RAN_PHY', '{"en":"RAN PHY 物理层","zh":"RAN PHY 物理层"}'::jsonb,
     'standard', true, ''),
    ('修改 RAN PHY 物理层', 'MOD RAN_PHY', '1', '修改 RAN PHY 物理层', 'SetParameterValues', 'MOD', '["Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.ConfigurationIndex","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.FreqOffset","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.HighSpeedFlag","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.RootSequenceIndex","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.ZeroCorrelationZoneConfig","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSEnabled","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSBandwidthConfig","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSMaxUpPTS","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.AckNackSRSSimultaneousTransmission","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.DeltaPUCCHShift","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.NRBCQI","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.NCSAN","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.N1PUCCHAN","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.CQIPUCCHResourceIndex","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.K","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PaParam.PUSCHPowerCtrlSwitch","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PaParam.PUCCHPowerCtrlSwitch","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.Enable64QAM","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.HoppingMode","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.HoppingOffset","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.NSB","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRS.NumPRSResourceBlocks","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRS.PRSConfigurationIndex","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRS.NumConsecutivePRSSubfames","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.TDDFrame.SpecialSubframePatterns","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.TDDFrame.SubFrameAssignment","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PDSCH.Pb","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PDSCH.Pa","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.P0NominalPUSCHPersistent","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.P0NominalPUSCH","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.Alpha","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.P0NominalPUCCH","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.DeltaMCSEnabled"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SH'), '{"en":"Modify RAN PHY 物理层","zh":"修改 RAN PHY 物理层"}'::jsonb,
     'RAN_PHY', '{"en":"RAN PHY 物理层","zh":"RAN PHY 物理层"}'::jsonb,
     'standard', true, ''),
    ('列出 PHY MBSFN 子帧', 'LST PHY_MBSFN', '1', '列出 PHY MBSFN 子帧', 'GetParameterValues', 'LST', '["Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.NeighCellConfig"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SH'), '{"en":"List PHY MBSFN 子帧","zh":"列出 PHY MBSFN 子帧"}'::jsonb,
     'PHY_MBSFN', '{"en":"PHY MBSFN 子帧","zh":"PHY MBSFN 子帧"}'::jsonb,
     'standard', true, ''),
    ('修改 PHY MBSFN 子帧', 'MOD PHY_MBSFN', '1', '修改 PHY MBSFN 子帧', 'SetParameterValues', 'MOD', '["Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.NeighCellConfig"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SH'), '{"en":"Modify PHY MBSFN 子帧","zh":"修改 PHY MBSFN 子帧"}'::jsonb,
     'PHY_MBSFN', '{"en":"PHY MBSFN 子帧","zh":"PHY MBSFN 子帧"}'::jsonb,
     'standard', true, ''),
    ('列出 连接态 EUTRA 测量', 'LST CONN_MODE_EUTRA', '1', '列出 连接态 EUTRA 测量', 'GetParameterValues', 'LST', '["Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.MeasureCtrl.Smeasure"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SJ'), '{"en":"List 连接态 EUTRA 测量","zh":"列出 连接态 EUTRA 测量"}'::jsonb,
     'CONN_MODE_EUTRA', '{"en":"连接态 EUTRA 测量","zh":"连接态 EUTRA 测量"}'::jsonb,
     'standard', true, ''),
    ('修改 连接态 EUTRA 测量', 'MOD CONN_MODE_EUTRA', '1', '修改 连接态 EUTRA 测量', 'SetParameterValues', 'MOD', '["Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.MeasureCtrl.Smeasure"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SJ'), '{"en":"Modify 连接态 EUTRA 测量","zh":"修改 连接态 EUTRA 测量"}'::jsonb,
     'CONN_MODE_EUTRA', '{"en":"连接态 EUTRA 测量","zh":"连接态 EUTRA 测量"}'::jsonb,
     'standard', true, ''),
    ('列出 连接态 IRAT 测量', 'LST CONN_MODE_IRAT', '1', '列出 连接态 IRAT 测量', 'GetParameterValues', 'LST', '["Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.QoffsetGERAN","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.MeasQuantityUTRAFDD","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.MeasQuantityGERAN","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.QoffsetUTRA"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SJ'), '{"en":"List 连接态 IRAT 测量","zh":"列出 连接态 IRAT 测量"}'::jsonb,
     'CONN_MODE_IRAT', '{"en":"连接态 IRAT 测量","zh":"连接态 IRAT 测量"}'::jsonb,
     'standard', true, ''),
    ('修改 连接态 IRAT 测量', 'MOD CONN_MODE_IRAT', '1', '修改 连接态 IRAT 测量', 'SetParameterValues', 'MOD', '["Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.QoffsetGERAN","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.MeasQuantityUTRAFDD","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.MeasQuantityGERAN","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.QoffsetUTRA"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SJ'), '{"en":"Modify 连接态 IRAT 测量","zh":"修改 连接态 IRAT 测量"}'::jsonb,
     'CONN_MODE_IRAT', '{"en":"连接态 IRAT 测量","zh":"连接态 IRAT 测量"}'::jsonb,
     'standard', true, ''),
    ('列出 空闲态移动性', 'LST MOBILITY_IDLE_MODE', '1', '列出 空闲态移动性', 'GetParameterValues', 'LST', '["Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.Qhyst","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.IntraFreqReselection","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.QHystSFMedium","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.QHystSFHigh","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.TEvaluation","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.THystNormal","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.NCellChangeMedium","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.NCellChangeHigh","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QRxLevMinSIB1","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QRxLevMinSIB3","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QRxLevMinOffset","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SIntraSearch","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRA","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SNonIntraSearch","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SNonIntraSearchPR9","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SNonIntraSearchQR9","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.CellReselectionPriority","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.PMax","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.ThreshServingLow","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.ThreshServingLowQR9","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRASFMedium","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRASFHigh","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SIntraSearchPR9","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SIntraSearchQR9","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QQualMinR9Reselection","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QQualMinR9Selection","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QQualMinOffsetR9","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.AllowedMeasBandwidth"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SJ'), '{"en":"List 空闲态移动性","zh":"列出 空闲态移动性"}'::jsonb,
     'MOBILITY_IDLE_MODE', '{"en":"空闲态移动性","zh":"空闲态移动性"}'::jsonb,
     'standard', true, ''),
    ('修改 空闲态移动性', 'MOD MOBILITY_IDLE_MODE', '1', '修改 空闲态移动性', 'SetParameterValues', 'MOD', '["Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.Qhyst","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.IntraFreqReselection","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.QHystSFMedium","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.QHystSFHigh","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.TEvaluation","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.THystNormal","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.NCellChangeMedium","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.NCellChangeHigh","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QRxLevMinSIB1","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QRxLevMinSIB3","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QRxLevMinOffset","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SIntraSearch","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRA","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SNonIntraSearch","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SNonIntraSearchPR9","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SNonIntraSearchQR9","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.CellReselectionPriority","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.PMax","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.ThreshServingLow","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.ThreshServingLowQR9","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRASFMedium","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRASFHigh","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SIntraSearchPR9","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SIntraSearchQR9","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QQualMinR9Reselection","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QQualMinR9Selection","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QQualMinOffsetR9","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.AllowedMeasBandwidth"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SJ'), '{"en":"Modify 空闲态移动性","zh":"修改 空闲态移动性"}'::jsonb,
     'MOBILITY_IDLE_MODE', '{"en":"空闲态移动性","zh":"空闲态移动性"}'::jsonb,
     'standard', true, ''),
    ('列出 空闲态 IRAT 移动性', 'LST IDLE_MODE_IRAT', '1', '列出 空闲态 IRAT 移动性', 'GetParameterValues', 'LST', '["Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.TReselectionUTRA","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.TReselectionGERAN"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SJ'), '{"en":"List 空闲态 IRAT 移动性","zh":"列出 空闲态 IRAT 移动性"}'::jsonb,
     'IDLE_MODE_IRAT', '{"en":"空闲态 IRAT 移动性","zh":"空闲态 IRAT 移动性"}'::jsonb,
     'standard', true, ''),
    ('修改 空闲态 IRAT 移动性', 'MOD IDLE_MODE_IRAT', '1', '修改 空闲态 IRAT 移动性', 'SetParameterValues', 'MOD', '["Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.TReselectionUTRA","Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.TReselectionGERAN"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SJ'), '{"en":"Modify 空闲态 IRAT 移动性","zh":"修改 空闲态 IRAT 移动性"}'::jsonb,
     'IDLE_MODE_IRAT', '{"en":"空闲态 IRAT 移动性","zh":"空闲态 IRAT 移动性"}'::jsonb,
     'standard', true, ''),
    ('列出 SON 自配置参数', 'LST SELF_CONFIG_SON_CONFIG_PARAM', '1', '列出 SON 自配置参数', 'GetParameterValues', 'LST', '["Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SONSysMode","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SONWorkMode","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.PCIOptEnable","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.PCIReconfigWaitTime","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.CandidateARFCNList","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.CandidatePCIList","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANREnable","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANRInterFeqEnable","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANRGERANEnable","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANRUTRANEnable","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ARFCNEnable","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MaxLTENeighbourCellNum","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MaxUTRANNeighbourCellNum","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MaxGRANNeighbourCellNum","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ReSynCellEnable","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.PowerEnable","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.LTESnifferFreqBandList","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.LTESnifferChannelList","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.GERANSnifferEnable","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.GERANSnifferChannelList","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.UTRANSnifferEnable","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.UTRANSnifferChannelList","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MROEnable","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SHEnable","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SyncMode"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SK'), '{"en":"List SON 自配置参数","zh":"列出 SON 自配置参数"}'::jsonb,
     'SELF_CONFIG_SON_CONFIG_PARAM', '{"en":"SON 自配置参数","zh":"SON 自配置参数"}'::jsonb,
     'standard', true, ''),
    ('修改 SON 自配置参数', 'MOD SELF_CONFIG_SON_CONFIG_PARAM', '1', '修改 SON 自配置参数', 'SetParameterValues', 'MOD', '["Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SONSysMode","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SONWorkMode","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.PCIOptEnable","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.PCIReconfigWaitTime","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.CandidateARFCNList","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.CandidatePCIList","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANREnable","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANRInterFeqEnable","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANRGERANEnable","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANRUTRANEnable","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ARFCNEnable","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MaxLTENeighbourCellNum","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MaxUTRANNeighbourCellNum","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MaxGRANNeighbourCellNum","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ReSynCellEnable","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.PowerEnable","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.LTESnifferFreqBandList","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.LTESnifferChannelList","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.GERANSnifferEnable","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.GERANSnifferChannelList","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.UTRANSnifferEnable","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.UTRANSnifferChannelList","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MROEnable","Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SHEnable"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SK'), '{"en":"Modify SON 自配置参数","zh":"修改 SON 自配置参数"}'::jsonb,
     'SELF_CONFIG_SON_CONFIG_PARAM', '{"en":"SON 自配置参数","zh":"SON 自配置参数"}'::jsonb,
     'standard', true, ''),
    ('列出 自配置启动状态', 'LST SELF_CONFIG', '1', '列出 自配置启动状态', 'GetParameterValues', 'LST', '["Device.Services.FAPService.{i}.FAPControl.SelfConfig.Startup.Stage","Device.Services.FAPService.{i}.FAPControl.SelfConfig.Startup.Status","Device.Services.FAPService.{i}.FAPControl.SelfConfig.Startup.FailureCause"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SK'), '{"en":"List 自配置启动状态","zh":"列出 自配置启动状态"}'::jsonb,
     'SELF_CONFIG', '{"en":"自配置启动状态","zh":"自配置启动状态"}'::jsonb,
     'standard', true, ''),
    ('增加 自配置启动状态', 'ADD SELF_CONFIG', '1', '增加 自配置启动状态', 'AddObject', 'ADD', '["Device.Services.FAPService.{i}.FAPControl.SelfConfig."]'::jsonb, 'Device.Services.FAPService.{i}.FAPControl.SelfConfig.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SK'), '{"en":"Add 自配置启动状态","zh":"增加 自配置启动状态"}'::jsonb,
     'SELF_CONFIG', '{"en":"自配置启动状态","zh":"自配置启动状态"}'::jsonb,
     'standard', true, ''),
    ('删除 自配置启动状态', 'RMV SELF_CONFIG', '1', '删除 自配置启动状态', 'DeleteObject', 'RMV', '["Device.Services.FAPService.{i}.FAPControl.SelfConfig."]'::jsonb, 'Device.Services.FAPService.{i}.FAPControl.SelfConfig.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SK'), '{"en":"Remove 自配置启动状态","zh":"删除 自配置启动状态"}'::jsonb,
     'SELF_CONFIG', '{"en":"自配置启动状态","zh":"自配置启动状态"}'::jsonb,
     'standard', true, ''),
    ('列出 以太网接口', 'LST ETHERNET_INTERFACE', '6', '列出 以太网接口', 'GetParameterValues', 'LST', '["Device.Ethernet.Interface.{i}.Enable","Device.Ethernet.Interface.{i}.UserLabel","Device.Ethernet.Interface.{i}.Name","Device.Ethernet.Interface.{i}.Status","Device.Ethernet.Interface.{i}.MACAddress","Device.Ethernet.Interface.{i}.MaxBitRate","Device.Ethernet.Interface.{i}.SignTransMedia","Device.Ethernet.Interface.{i}.DuplexMode","Device.Ethernet.Interface.{i}.PortLocation"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SL'), '{"en":"List 以太网接口","zh":"列出 以太网接口"}'::jsonb,
     'ETHERNET_INTERFACE', '{"en":"以太网接口","zh":"以太网接口"}'::jsonb,
     'standard', true, ''),
    ('修改 以太网接口', 'MOD ETHERNET_INTERFACE', '6', '修改 以太网接口', 'SetParameterValues', 'MOD', '["Device.Ethernet.Interface.{i}.Enable","Device.Ethernet.Interface.{i}.UserLabel","Device.Ethernet.Interface.{i}.MaxBitRate","Device.Ethernet.Interface.{i}.DuplexMode"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SL'), '{"en":"Modify 以太网接口","zh":"修改 以太网接口"}'::jsonb,
     'ETHERNET_INTERFACE', '{"en":"以太网接口","zh":"以太网接口"}'::jsonb,
     'standard', true, ''),
    ('增加 以太网接口', 'ADD ETHERNET_INTERFACE', '6', '增加 以太网接口', 'AddObject', 'ADD', '["Device.Ethernet.Interface."]'::jsonb, 'Device.Ethernet.Interface.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SL'), '{"en":"Add 以太网接口","zh":"增加 以太网接口"}'::jsonb,
     'ETHERNET_INTERFACE', '{"en":"以太网接口","zh":"以太网接口"}'::jsonb,
     'standard', true, ''),
    ('删除 以太网接口', 'RMV ETHERNET_INTERFACE', '6', '删除 以太网接口', 'DeleteObject', 'RMV', '["Device.Ethernet.Interface."]'::jsonb, 'Device.Ethernet.Interface.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SL'), '{"en":"Remove 以太网接口","zh":"删除 以太网接口"}'::jsonb,
     'ETHERNET_INTERFACE', '{"en":"以太网接口","zh":"以太网接口"}'::jsonb,
     'standard', true, ''),
    ('列出 静态路由表项', 'LST ETHERNET_IP_ROUTE', '6', '列出 静态路由表项', 'GetParameterValues', 'LST', '["Device.Ethernet.IpRoute.{i}.IpVer","Device.Ethernet.IpRoute.{i}.DstIpNetwork","Device.Ethernet.IpRoute.{i}.PrefixLength","Device.Ethernet.IpRoute.{i}.GatewayIpAddress","Device.Ethernet.IpRoute.{i}.InterfaceName"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SL'), '{"en":"List 静态路由表项","zh":"列出 静态路由表项"}'::jsonb,
     'ETHERNET_IP_ROUTE', '{"en":"静态路由表项","zh":"静态路由表项"}'::jsonb,
     'standard', true, ''),
    ('修改 静态路由表项', 'MOD ETHERNET_IP_ROUTE', '6', '修改 静态路由表项', 'SetParameterValues', 'MOD', '["Device.Ethernet.IpRoute.{i}.IpVer","Device.Ethernet.IpRoute.{i}.DstIpNetwork","Device.Ethernet.IpRoute.{i}.PrefixLength","Device.Ethernet.IpRoute.{i}.GatewayIpAddress","Device.Ethernet.IpRoute.{i}.InterfaceName"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SL'), '{"en":"Modify 静态路由表项","zh":"修改 静态路由表项"}'::jsonb,
     'ETHERNET_IP_ROUTE', '{"en":"静态路由表项","zh":"静态路由表项"}'::jsonb,
     'standard', true, ''),
    ('增加 静态路由表项', 'ADD ETHERNET_IP_ROUTE', '6', '增加 静态路由表项', 'AddObject', 'ADD', '["Device.Ethernet.IpRoute."]'::jsonb, 'Device.Ethernet.IpRoute.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SL'), '{"en":"Add 静态路由表项","zh":"增加 静态路由表项"}'::jsonb,
     'ETHERNET_IP_ROUTE', '{"en":"静态路由表项","zh":"静态路由表项"}'::jsonb,
     'standard', true, ''),
    ('删除 静态路由表项', 'RMV ETHERNET_IP_ROUTE', '6', '删除 静态路由表项', 'DeleteObject', 'RMV', '["Device.Ethernet.IpRoute."]'::jsonb, 'Device.Ethernet.IpRoute.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SL'), '{"en":"Remove 静态路由表项","zh":"删除 静态路由表项"}'::jsonb,
     'ETHERNET_IP_ROUTE', '{"en":"静态路由表项","zh":"静态路由表项"}'::jsonb,
     'standard', true, ''),
    ('列出 IPsec 安全配置', 'LST I_PSEC', '6', '列出 IPsec 安全配置', 'GetParameterValues', 'LST', '["Device.IPsec.Enable","Device.IPsec.MyKeyMode","Device.IPsec.Status","Device.IPsec.AHSupported","Device.IPsec.IKEv2SupportedEncryptionAlgorithms","Device.IPsec.ESPSupportedEncryptionAlgorithms","Device.IPsec.IKEv2SupportedPseudoRandomFunctions","Device.IPsec.SupportedIntegrityAlgorithms","Device.IPsec.SupportedDiffieHellmanGroupTransforms"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SM'), '{"en":"List IPsec 安全配置","zh":"列出 IPsec 安全配置"}'::jsonb,
     'I_PSEC', '{"en":"IPsec 安全配置","zh":"IPsec 安全配置"}'::jsonb,
     'standard', true, ''),
    ('修改 IPsec 安全配置', 'MOD I_PSEC', '6', '修改 IPsec 安全配置', 'SetParameterValues', 'MOD', '["Device.IPsec.Enable","Device.IPsec.MyKeyMode"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SM'), '{"en":"Modify IPsec 安全配置","zh":"修改 IPsec 安全配置"}'::jsonb,
     'I_PSEC', '{"en":"IPsec 安全配置","zh":"IPsec 安全配置"}'::jsonb,
     'standard', true, ''),
    ('列出 时间同步服务器', 'LST TIME', '6', '列出 时间同步服务器', 'GetParameterValues', 'LST', '["Device.Time.Enable","Device.Time.NTPServer1","Device.Time.NTPServer2","Device.Time.NTPServer3","Device.Time.NTPServer4","Device.Time.NTPServer5","Device.Time.CurrentLocalTime","Device.Time.LocalTimeZone"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SN'), '{"en":"List 时间同步服务器","zh":"列出 时间同步服务器"}'::jsonb,
     'TIME', '{"en":"时间同步服务器","zh":"时间同步服务器"}'::jsonb,
     'standard', true, ''),
    ('修改 时间同步服务器', 'MOD TIME', '6', '修改 时间同步服务器', 'SetParameterValues', 'MOD', '["Device.Time.Enable","Device.Time.NTPServer1","Device.Time.NTPServer2","Device.Time.NTPServer3","Device.Time.NTPServer4","Device.Time.NTPServer5","Device.Time.LocalTimeZone"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SN'), '{"en":"Modify 时间同步服务器","zh":"修改 时间同步服务器"}'::jsonb,
     'TIME', '{"en":"时间同步服务器","zh":"时间同步服务器"}'::jsonb,
     'standard', true, ''),
    ('列出 GPS 定位信息', 'LST FAP_GPS', '5', '列出 GPS 定位信息', 'GetParameterValues', 'LST', '["Device.FAP.GPS.LockedLatitude","Device.FAP.GPS.LockedLongitude","Device.FAP.GPS.NumberOfSatellites"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SO'), '{"en":"List GPS 定位信息","zh":"列出 GPS 定位信息"}'::jsonb,
     'FAP_GPS', '{"en":"GPS 定位信息","zh":"GPS 定位信息"}'::jsonb,
     'standard', true, ''),
    ('列出 MR 上报配置', 'LST MR_MGMT_CONFIG', '2', '列出 MR 上报配置', 'GetParameterValues', 'LST', '["Device.FAP.MRMgmt.Config.{i}.MrEnable","Device.FAP.MRMgmt.Config.{i}.MrUrl","Device.FAP.MRMgmt.Config.{i}.MrUsername","Device.FAP.MRMgmt.Config.{i}.MrPassword","Device.FAP.MRMgmt.Config.{i}.MeasureType","Device.FAP.MRMgmt.Config.{i}.OmcName","Device.FAP.MRMgmt.Config.{i}.SamplePeriod","Device.FAP.MRMgmt.Config.{i}.UploadPeriod","Device.FAP.MRMgmt.Config.{i}.SampleBeginTime","Device.FAP.MRMgmt.Config.{i}.SampleEndTime","Device.FAP.MRMgmt.Config.{i}.PrbNum","Device.FAP.MRMgmt.Config.{i}.SubFrameNum","Device.FAP.MRMgmt.Config.{i}.MRECGIList","Device.FAP.MRMgmt.Config.{i}.MeasureItems"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SP'), '{"en":"List MR 上报配置","zh":"列出 MR 上报配置"}'::jsonb,
     'MR_MGMT_CONFIG', '{"en":"MR 上报配置","zh":"MR 上报配置"}'::jsonb,
     'standard', true, ''),
    ('修改 MR 上报配置', 'MOD MR_MGMT_CONFIG', '2', '修改 MR 上报配置', 'SetParameterValues', 'MOD', '["Device.FAP.MRMgmt.Config.{i}.MrEnable","Device.FAP.MRMgmt.Config.{i}.MrUrl","Device.FAP.MRMgmt.Config.{i}.MrUsername","Device.FAP.MRMgmt.Config.{i}.MrPassword","Device.FAP.MRMgmt.Config.{i}.MeasureType","Device.FAP.MRMgmt.Config.{i}.OmcName","Device.FAP.MRMgmt.Config.{i}.SamplePeriod","Device.FAP.MRMgmt.Config.{i}.UploadPeriod","Device.FAP.MRMgmt.Config.{i}.SampleBeginTime","Device.FAP.MRMgmt.Config.{i}.SampleEndTime","Device.FAP.MRMgmt.Config.{i}.PrbNum","Device.FAP.MRMgmt.Config.{i}.SubFrameNum","Device.FAP.MRMgmt.Config.{i}.MRECGIList","Device.FAP.MRMgmt.Config.{i}.MeasureItems"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SP'), '{"en":"Modify MR 上报配置","zh":"修改 MR 上报配置"}'::jsonb,
     'MR_MGMT_CONFIG', '{"en":"MR 上报配置","zh":"MR 上报配置"}'::jsonb,
     'standard', true, ''),
    ('增加 MR 上报配置', 'ADD MR_MGMT_CONFIG', '2', '增加 MR 上报配置', 'AddObject', 'ADD', '["Device.FAP.MRMgmt.Config."]'::jsonb, 'Device.FAP.MRMgmt.Config.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SP'), '{"en":"Add MR 上报配置","zh":"增加 MR 上报配置"}'::jsonb,
     'MR_MGMT_CONFIG', '{"en":"MR 上报配置","zh":"MR 上报配置"}'::jsonb,
     'standard', true, ''),
    ('删除 MR 上报配置', 'RMV MR_MGMT_CONFIG', '2', '删除 MR 上报配置', 'DeleteObject', 'RMV', '["Device.FAP.MRMgmt.Config."]'::jsonb, 'Device.FAP.MRMgmt.Config.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SP'), '{"en":"Remove MR 上报配置","zh":"删除 MR 上报配置"}'::jsonb,
     'MR_MGMT_CONFIG', '{"en":"MR 上报配置","zh":"MR 上报配置"}'::jsonb,
     'standard', true, ''),
    ('列出 PM 性能上报配置', 'LST PERF_MGMT_CONFIG', '2', '列出 PM 性能上报配置', 'GetParameterValues', 'LST', '["Device.FAP.PerfMgmt.Config.{i}.Enable","Device.FAP.PerfMgmt.Config.{i}.Alias","Device.FAP.PerfMgmt.Config.{i}.URL","Device.FAP.PerfMgmt.Config.{i}.Username","Device.FAP.PerfMgmt.Config.{i}.Password","Device.FAP.PerfMgmt.Config.{i}.PeriodicUploadInterval","Device.FAP.PerfMgmt.Config.{i}.PeriodicUploadTime","Device.FAP.PerfMgmt.Config.{i}.ReplenishEnable","Device.FAP.PerfMgmt.Config.{i}.ReplenishStartTime","Device.FAP.PerfMgmt.Config.{i}.ReplenishEndTime"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SQ'), '{"en":"List PM 性能上报配置","zh":"列出 PM 性能上报配置"}'::jsonb,
     'PERF_MGMT_CONFIG', '{"en":"PM 性能上报配置","zh":"PM 性能上报配置"}'::jsonb,
     'standard', true, ''),
    ('修改 PM 性能上报配置', 'MOD PERF_MGMT_CONFIG', '2', '修改 PM 性能上报配置', 'SetParameterValues', 'MOD', '["Device.FAP.PerfMgmt.Config.{i}.Enable","Device.FAP.PerfMgmt.Config.{i}.Alias","Device.FAP.PerfMgmt.Config.{i}.URL","Device.FAP.PerfMgmt.Config.{i}.Username","Device.FAP.PerfMgmt.Config.{i}.Password","Device.FAP.PerfMgmt.Config.{i}.PeriodicUploadInterval","Device.FAP.PerfMgmt.Config.{i}.PeriodicUploadTime","Device.FAP.PerfMgmt.Config.{i}.ReplenishEnable","Device.FAP.PerfMgmt.Config.{i}.ReplenishStartTime","Device.FAP.PerfMgmt.Config.{i}.ReplenishEndTime"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SQ'), '{"en":"Modify PM 性能上报配置","zh":"修改 PM 性能上报配置"}'::jsonb,
     'PERF_MGMT_CONFIG', '{"en":"PM 性能上报配置","zh":"PM 性能上报配置"}'::jsonb,
     'standard', true, ''),
    ('增加 PM 性能上报配置', 'ADD PERF_MGMT_CONFIG', '2', '增加 PM 性能上报配置', 'AddObject', 'ADD', '["Device.FAP.PerfMgmt.Config."]'::jsonb, 'Device.FAP.PerfMgmt.Config.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SQ'), '{"en":"Add PM 性能上报配置","zh":"增加 PM 性能上报配置"}'::jsonb,
     'PERF_MGMT_CONFIG', '{"en":"PM 性能上报配置","zh":"PM 性能上报配置"}'::jsonb,
     'standard', true, ''),
    ('删除 PM 性能上报配置', 'RMV PERF_MGMT_CONFIG', '2', '删除 PM 性能上报配置', 'DeleteObject', 'RMV', '["Device.FAP.PerfMgmt.Config."]'::jsonb, 'Device.FAP.PerfMgmt.Config.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SQ'), '{"en":"Remove PM 性能上报配置","zh":"删除 PM 性能上报配置"}'::jsonb,
     'PERF_MGMT_CONFIG', '{"en":"PM 性能上报配置","zh":"PM 性能上报配置"}'::jsonb,
     'standard', true, ''),
    ('列出 主机单元基本信息', 'LST DEVICE_INFO_MU', '3', '列出 主机单元基本信息', 'GetParameterValues', 'LST', '["Device.DeviceInfo.MU.{i}.UserLabel","Device.DeviceInfo.MU.{i}.DnPrefix","Device.DeviceInfo.MU.{i}.ManufacturerOUI","Device.DeviceInfo.MU.{i}.Manufacturer","Device.DeviceInfo.MU.{i}.ModelName","Device.DeviceInfo.MU.{i}.VendorUnitFamilyType","Device.DeviceInfo.MU.{i}.VendorUnitTypeNumber","Device.DeviceInfo.MU.{i}.SerialNumber","Device.DeviceInfo.MU.{i}.HardwareVersion","Device.DeviceInfo.MU.{i}.SoftwareVersion","Device.DeviceInfo.MU.{i}.HardwarePlatform","Device.DeviceInfo.MU.{i}.AdditionalHardwareVersion","Device.DeviceInfo.MU.{i}.AdditionalSoftwareVersion","Device.DeviceInfo.MU.{i}.ProvisioningCode","Device.DeviceInfo.MU.{i}.ProductClass","Device.DeviceInfo.MU.{i}.Status","Device.DeviceInfo.MU.{i}.Reboot","Device.DeviceInfo.MU.{i}.UpTime","Device.DeviceInfo.MU.{i}.FirstUseDate","Device.DeviceInfo.MU.{i}.ClockSource","Device.DeviceInfo.MU.{i}.DateOfLastService","Device.DeviceInfo.MU.{i}.DateOfManufacture","Device.DeviceInfo.MU.{i}.ManufacturerData","Device.DeviceInfo.MU.{i}.SlotsInformation"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SR'), '{"en":"List 主机单元基本信息","zh":"列出 主机单元基本信息"}'::jsonb,
     'DEVICE_INFO_MU', '{"en":"主机单元基本信息","zh":"主机单元基本信息"}'::jsonb,
     'standard', true, ''),
    ('修改 主机单元基本信息', 'MOD DEVICE_INFO_MU', '3', '修改 主机单元基本信息', 'SetParameterValues', 'MOD', '["Device.DeviceInfo.MU.{i}.UserLabel","Device.DeviceInfo.MU.{i}.DnPrefix","Device.DeviceInfo.MU.{i}.Reboot"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SR'), '{"en":"Modify 主机单元基本信息","zh":"修改 主机单元基本信息"}'::jsonb,
     'DEVICE_INFO_MU', '{"en":"主机单元基本信息","zh":"主机单元基本信息"}'::jsonb,
     'standard', true, ''),
    ('列出 主机单元软件升级', 'LST MU_SW_UPGRADE', '3', '列出 主机单元软件升级', 'GetParameterValues', 'LST', '["Device.DeviceInfo.MU.{i}.SwUpgrade.Stage","Device.DeviceInfo.MU.{i}.SwUpgrade.FailureCause","Device.DeviceInfo.MU.{i}.SwUpgrade.Status"]'::jsonb, NULL,
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SR'), '{"en":"List 主机单元软件升级","zh":"列出 主机单元软件升级"}'::jsonb,
     'MU_SW_UPGRADE', '{"en":"主机单元软件升级","zh":"主机单元软件升级"}'::jsonb,
     'standard', true, ''),
    ('增加 主机单元软件升级', 'ADD MU_SW_UPGRADE', '3', '增加 主机单元软件升级', 'AddObject', 'ADD', '["Device.DeviceInfo.MU.{i}.SwUpgrade."]'::jsonb, 'Device.DeviceInfo.MU.{i}.SwUpgrade.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SR'), '{"en":"Add 主机单元软件升级","zh":"增加 主机单元软件升级"}'::jsonb,
     'MU_SW_UPGRADE', '{"en":"主机单元软件升级","zh":"主机单元软件升级"}'::jsonb,
     'standard', true, ''),
    ('删除 主机单元软件升级', 'RMV MU_SW_UPGRADE', '3', '删除 主机单元软件升级', 'DeleteObject', 'RMV', '["Device.DeviceInfo.MU.{i}.SwUpgrade."]'::jsonb, 'Device.DeviceInfo.MU.{i}.SwUpgrade.',
     (SELECT id FROM chapter_groups WHERE group_code = 'chapter:SR'), '{"en":"Remove 主机单元软件升级","zh":"删除 主机单元软件升级"}'::jsonb,
     'MU_SW_UPGRADE', '{"en":"主机单元软件升级","zh":"主机单元软件升级"}'::jsonb,
     'standard', true, '')
ON CONFLICT (command_code) DO UPDATE SET
    target_paths      = EXCLUDED.target_paths,
    target_object     = EXCLUDED.target_object,
    command_name_i18n = EXCLUDED.command_name_i18n,
    logical_name_i18n = EXCLUDED.logical_name_i18n,
    description       = EXCLUDED.description,
    source            = 'standard',
    catalog_protected = true;

-- Section C: mml_command_sub_fields — 新增 625 条关联
INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CELL_BARRED', '{"en":"CellBarred","zh":"小区闭塞"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CellRestriction.CellBarred'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ADMIN_STATE', '{"en":"AdminState","zh":"小区管理状态"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CellEnable.AdminState'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'OP_STATE', '{"en":"OpState","zh":"小区运行状态"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.OpState'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SUPPORT_RRC_NUMBERS', '{"en":"SupportRRCNumbers","zh":"同时支持RRC连接最大用户数"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.AccessMgmt.LTE.MaxUEsServed'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MULTI_BAND_INFO_LIST_SIB1', '{"en":"MultiBandInfoListSIB1","zh":"SIB1中多频段指示参数"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.SysInfoCtrlParam.MultiBandInfoListSIB1'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MULTI_BAND_INFO_LIST_SIB5', '{"en":"MultiBandInfoListSIB5","zh":"SIB5中多频段指示参数"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.SysInfoCtrlParam.MultiBandInfoListSIB5'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ROUTE_INDEX_LIST', '{"en":"RouteIndexList","zh":"路由指示列表"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RouteIndexList'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'RU_LIST', '{"en":"RuList","zh":"RU列表"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RuList'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'USER_LABEL', '{"en":"UserLabel","zh":"用户友好名"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.UserLabel'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'EARFCNDL', '{"en":"EARFCNDL","zh":"下行EARFCN"}'::jsonb, true, false, 10
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.EARFCNDL'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PHY_CELL_ID', '{"en":"PhyCellID","zh":"PCI"}'::jsonb, true, false, 11
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PhyCellID'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'DL_BANDWIDTH', '{"en":"DLBandwidth","zh":"下行带宽"}'::jsonb, true, false, 12
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.DLBandwidth'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'UL_BANDWIDTH', '{"en":"ULBandwidth","zh":"上行带宽"}'::jsonb, true, false, 13
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ULBandwidth'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PSCH_POWER_OFFSET', '{"en":"PSCHPowerOffset","zh":"PSCH功率偏置"}'::jsonb, true, false, 14
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PSCHPowerOffset'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SSCH_POWER_OFFSET', '{"en":"SSCHPowerOffset","zh":"SSCH功率偏置"}'::jsonb, true, false, 15
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.SSCHPowerOffset'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PBCH_POWER_OFFSET', '{"en":"PBCHPowerOffset","zh":"PBCH功率偏置"}'::jsonb, true, false, 16
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PBCHPowerOffset'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'EARFCNUL', '{"en":"EARFCNUL","zh":"上行EARFCN"}'::jsonb, true, false, 17
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.EARFCNUL'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'FREQ_BAND_INDICATOR', '{"en":"FreqBandIndicator","zh":"频带指示"}'::jsonb, true, false, 18
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.FreqBandIndicator'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'REFERENCE_SIGNAL_POWER', '{"en":"ReferenceSignalPower","zh":"参考信号功率"}'::jsonb, true, false, 19
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ReferenceSignalPower'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CELL_IDENTITY', '{"en":"CellIdentity","zh":"小区标识"}'::jsonb, true, false, 20
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.CellIdentity'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ENB_TYPE', '{"en":"EnbType","zh":"基站类型"}'::jsonb, true, false, 21
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.EnbType'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SPS_SWITCH_QCI1_UL', '{"en":"SPSSwitchQCI1Ul","zh":"QCI1上行SPS功能开关"}'::jsonb, true, false, 22
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.VoLTEParam.SPSSwitchQCI1Ul'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CA_SWITCH_UL', '{"en":"CASwitchUl","zh":"上行载波聚合功能开关"}'::jsonb, true, false, 23
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.CAParam.CASwitchUl'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CA_SWITCH_DL', '{"en":"CASwitchDl","zh":"下行载波聚合功能开关"}'::jsonb, true, false, 24
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.CAParam.CASwitchDl'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T300', '{"en":"T300","zh":"T300定时器"}'::jsonb, true, false, 25
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T300'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T301', '{"en":"T301","zh":"T301定时器"}'::jsonb, true, false, 26
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T301'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T302', '{"en":"T302","zh":"T302定时器"}'::jsonb, true, false, 27
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T302'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T304EUTRA', '{"en":"T304EUTRA","zh":"T304(EUTRA)定时器"}'::jsonb, true, false, 28
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304EUTRA'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T304IRAT', '{"en":"T304IRAT","zh":"T304(异系统)定时器"}'::jsonb, true, false, 29
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304IRAT'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T310', '{"en":"T310","zh":"T310定时器"}'::jsonb, true, false, 30
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T310'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T311', '{"en":"T311","zh":"T311定时器"}'::jsonb, true, false, 31
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T311'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T320', '{"en":"T320","zh":"T320定时器"}'::jsonb, true, false, 32
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T320'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'N310', '{"en":"N310","zh":"T310定时器"}'::jsonb, true, false, 33
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N310'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'N311', '{"en":"N311","zh":"T311定时器"}'::jsonb, true, false, 34
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N311'
 WHERE c.command_code = 'LST '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NNSF_SUPPORTED', '{"en":"NNSFSupported","zh":"负载均衡参数"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.Capabilities.LTE.NNSFSupported'
 WHERE c.command_code = 'LST CAPABILITIES'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'UE_INACTIVE_TIMER', '{"en":"UeInactiveTimer","zh":"连接态UE不活动定时器"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.Capabilities.LTE.UeInactiveTimer'
 WHERE c.command_code = 'LST CELL_CONFIG_CAPABILITIES'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SUPPORT_ACTIVE_RRC_NUMBERS', '{"en":"SupportActiveRRCNumbers","zh":"同时支持RRC激活最大用户数"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.Capabilities.LTE.SupportActiveRRCNumbers'
 WHERE c.command_code = 'LST CELL_CONFIG_CAPABILITIES'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MAX_TX_POWER', '{"en":"MaxTxPower","zh":"最大发射功率"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.Capabilities.MaxTxPower'
 WHERE c.command_code = 'LST CELL_CONFIG_CAPABILITIES'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SMEASURE', '{"en":"Smeasure","zh":"测量启动门限"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.MeasureCtrl.Smeasure'
 WHERE c.command_code = 'LST CONN_MODE_EUTRA'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'QOFFSET_GERAN', '{"en":"QoffsetGERAN","zh":"GERAN频点偏移量"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.QoffsetGERAN'
 WHERE c.command_code = 'LST CONN_MODE_IRAT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MEAS_QUANTITY_UTRAFDD', '{"en":"MeasQuantityUTRAFDD","zh":"UTRA测量量"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.MeasQuantityUTRAFDD'
 WHERE c.command_code = 'LST CONN_MODE_IRAT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MEAS_QUANTITY_GERAN', '{"en":"MeasQuantityGERAN","zh":"GERAN测量量"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.MeasQuantityGERAN'
 WHERE c.command_code = 'LST CONN_MODE_IRAT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'QOFFSET_UTRA', '{"en":"QoffsetUTRA","zh":"UTRA频点偏移量"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.QoffsetUTRA'
 WHERE c.command_code = 'LST CONN_MODE_IRAT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'USER_LABEL', '{"en":"UserLabel","zh":"用户友好名"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.UserLabel'
 WHERE c.command_code = 'LST DEVICE_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'DN_PREFIX', '{"en":"DnPrefix","zh":"DN前缀"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.DnPrefix'
 WHERE c.command_code = 'LST DEVICE_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MANUFACTURER_OUI', '{"en":"ManufacturerOUI","zh":"制造商OUI"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.ManufacturerOUI'
 WHERE c.command_code = 'LST DEVICE_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MANUFACTURER', '{"en":"Manufacturer","zh":"制造商"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.Manufacturer'
 WHERE c.command_code = 'LST DEVICE_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MODEL_NAME', '{"en":"ModelName","zh":"设备型号"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.ModelName'
 WHERE c.command_code = 'LST DEVICE_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SERIAL_NUMBER', '{"en":"SerialNumber","zh":"序列号"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.SerialNumber'
 WHERE c.command_code = 'LST DEVICE_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'HARDWARE_VERSION', '{"en":"HardwareVersion","zh":"硬件版本"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.HardwareVersion'
 WHERE c.command_code = 'LST DEVICE_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SOFTWARE_VERSION', '{"en":"SoftwareVersion","zh":"软件版本"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.SoftwareVersion'
 WHERE c.command_code = 'LST DEVICE_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'HARDWARE_PLATFORM', '{"en":"HardwarePlatform","zh":"硬件平台"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.HardwarePlatform'
 WHERE c.command_code = 'LST DEVICE_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ADDITIONAL_HARDWARE_VERSION', '{"en":"AdditionalHardwareVersion","zh":"附加硬件版本"}'::jsonb, true, false, 10
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.AdditionalHardwareVersion'
 WHERE c.command_code = 'LST DEVICE_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ADDITIONAL_SOFTWARE_VERSION', '{"en":"AdditionalSoftwareVersion","zh":"附加软件版本"}'::jsonb, true, false, 11
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.AdditionalSoftwareVersion'
 WHERE c.command_code = 'LST DEVICE_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PROVISIONING_CODE', '{"en":"ProvisioningCode","zh":"供应商代号"}'::jsonb, true, false, 12
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.ProvisioningCode'
 WHERE c.command_code = 'LST DEVICE_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PRODUCT_CLASS', '{"en":"ProductClass","zh":"产品分类"}'::jsonb, true, false, 13
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.ProductClass'
 WHERE c.command_code = 'LST DEVICE_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'UP_TIME', '{"en":"UpTime","zh":"运行时间"}'::jsonb, true, false, 14
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.UpTime'
 WHERE c.command_code = 'LST DEVICE_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, '3GPP_SPEC_VERSION', '{"en":"3GPPSpecVersion","zh":"3GPP协议版本"}'::jsonb, true, false, 15
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.3GPPSpecVersion'
 WHERE c.command_code = 'LST DEVICE_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'FIRST_USE_DATE', '{"en":"FirstUseDate","zh":"首次使用日期"}'::jsonb, true, false, 16
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.FirstUseDate'
 WHERE c.command_code = 'LST DEVICE_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'DATA_MODEL_SPEC_VERSION', '{"en":"DataModelSpecVersion","zh":"数据模型版本"}'::jsonb, true, false, 17
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.DataModelSpecVersion'
 WHERE c.command_code = 'LST DEVICE_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'USER_LABEL', '{"en":"UserLabel","zh":"用户友好名"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.UserLabel'
 WHERE c.command_code = 'LST DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'DN_PREFIX', '{"en":"DnPrefix","zh":"DN前缀"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.DnPrefix'
 WHERE c.command_code = 'LST DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MANUFACTURER_OUI', '{"en":"ManufacturerOUI","zh":"制造商OUI"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.ManufacturerOUI'
 WHERE c.command_code = 'LST DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MANUFACTURER', '{"en":"Manufacturer","zh":"制造商"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.Manufacturer'
 WHERE c.command_code = 'LST DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MODEL_NAME', '{"en":"ModelName","zh":"设备型号"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.ModelName'
 WHERE c.command_code = 'LST DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'VENDOR_UNIT_FAMILY_TYPE', '{"en":"VendorUnitFamilyType","zh":"归属类型"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.VendorUnitFamilyType'
 WHERE c.command_code = 'LST DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'VENDOR_UNIT_TYPE_NUMBER', '{"en":"VendorUnitTypeNumber","zh":"资产单元类型版本号"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.VendorUnitTypeNumber'
 WHERE c.command_code = 'LST DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SERIAL_NUMBER', '{"en":"SerialNumber","zh":"序列号"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.SerialNumber'
 WHERE c.command_code = 'LST DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'HARDWARE_VERSION', '{"en":"HardwareVersion","zh":"硬件版本"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.HardwareVersion'
 WHERE c.command_code = 'LST DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SOFTWARE_VERSION', '{"en":"SoftwareVersion","zh":"软件版本"}'::jsonb, true, false, 10
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.SoftwareVersion'
 WHERE c.command_code = 'LST DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'HARDWARE_PLATFORM', '{"en":"HardwarePlatform","zh":"硬件平台"}'::jsonb, true, false, 11
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.HardwarePlatform'
 WHERE c.command_code = 'LST DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ADDITIONAL_HARDWARE_VERSION', '{"en":"AdditionalHardwareVersion","zh":"附加硬件版本"}'::jsonb, true, false, 12
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.AdditionalHardwareVersion'
 WHERE c.command_code = 'LST DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ADDITIONAL_SOFTWARE_VERSION', '{"en":"AdditionalSoftwareVersion","zh":"附加软件版本"}'::jsonb, true, false, 13
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.AdditionalSoftwareVersion'
 WHERE c.command_code = 'LST DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PROVISIONING_CODE', '{"en":"ProvisioningCode","zh":"供应商代号"}'::jsonb, true, false, 14
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.ProvisioningCode'
 WHERE c.command_code = 'LST DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PRODUCT_CLASS', '{"en":"ProductClass","zh":"产品分类"}'::jsonb, true, false, 15
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.ProductClass'
 WHERE c.command_code = 'LST DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'STATUS', '{"en":"Status","zh":"机框/服务器状态"}'::jsonb, true, false, 16
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.Status'
 WHERE c.command_code = 'LST DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'REBOOT', '{"en":"Reboot","zh":"重启开关"}'::jsonb, true, false, 17
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.Reboot'
 WHERE c.command_code = 'LST DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'UP_TIME', '{"en":"UpTime","zh":"运行时间"}'::jsonb, true, false, 18
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.UpTime'
 WHERE c.command_code = 'LST DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'FIRST_USE_DATE', '{"en":"FirstUseDate","zh":"首次使用日期"}'::jsonb, true, false, 19
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.FirstUseDate'
 WHERE c.command_code = 'LST DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CLOCK_SOURCE', '{"en":"ClockSource","zh":"时钟源类型"}'::jsonb, true, false, 20
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.ClockSource'
 WHERE c.command_code = 'LST DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'DATE_OF_LAST_SERVICE', '{"en":"DateOfLastService","zh":"最近服务的日期（最近一次恢复工作正常状态的时间）"}'::jsonb, true, false, 21
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.DateOfLastService'
 WHERE c.command_code = 'LST DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'DATE_OF_MANUFACTURE', '{"en":"DateOfManufacture","zh":"生产日期"}'::jsonb, true, false, 22
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.DateOfManufacture'
 WHERE c.command_code = 'LST DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MANUFACTURER_DATA', '{"en":"ManufacturerData","zh":"特殊信息"}'::jsonb, true, false, 23
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.ManufacturerData'
 WHERE c.command_code = 'LST DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SLOTS_INFORMATION', '{"en":"SlotsInformation","zh":"插槽信息"}'::jsonb, true, false, 24
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.SlotsInformation'
 WHERE c.command_code = 'LST DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'STAGE', '{"en":"Stage","zh":"升级阶段"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.SwUpgrade.Stage'
 WHERE c.command_code = 'LST DEVICE_INFO_SW_UPGRADE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'FAILURE_CAUSE', '{"en":"FailureCause","zh":"升级失败原因"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.SwUpgrade.FailureCause'
 WHERE c.command_code = 'LST DEVICE_INFO_SW_UPGRADE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'STATUS', '{"en":"Status","zh":"状态"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.SwUpgrade.Status'
 WHERE c.command_code = 'LST DEVICE_INFO_SW_UPGRADE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ENABLE', '{"en":"Enable","zh":"启用或禁用该接口"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Ethernet.Interface.{i}.Enable'
 WHERE c.command_code = 'LST ETHERNET_INTERFACE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'USER_LABEL', '{"en":"UserLabel","zh":"用户友好名"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Ethernet.Interface.{i}.UserLabel'
 WHERE c.command_code = 'LST ETHERNET_INTERFACE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NAME', '{"en":"Name","zh":"端口名称"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Ethernet.Interface.{i}.Name'
 WHERE c.command_code = 'LST ETHERNET_INTERFACE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'STATUS', '{"en":"Status","zh":"表明接口的状态"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Ethernet.Interface.{i}.Status'
 WHERE c.command_code = 'LST ETHERNET_INTERFACE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MAC_ADDRESS', '{"en":"MACAddress","zh":"接口的物理地址"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Ethernet.Interface.{i}.MACAddress'
 WHERE c.command_code = 'LST ETHERNET_INTERFACE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MAX_BIT_RATE', '{"en":"MaxBitRate","zh":"该连接可用的最大速率模式"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Ethernet.Interface.{i}.MaxBitRate'
 WHERE c.command_code = 'LST ETHERNET_INTERFACE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SIGN_TRANS_MEDIA', '{"en":"SignTransMedia","zh":"信号传送介质类型"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Ethernet.Interface.{i}.SignTransMedia'
 WHERE c.command_code = 'LST ETHERNET_INTERFACE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'DUPLEX_MODE', '{"en":"DuplexMode","zh":"该连接使用的双工模式"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Ethernet.Interface.{i}.DuplexMode'
 WHERE c.command_code = 'LST ETHERNET_INTERFACE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PORT_LOCATION', '{"en":"PortLocation","zh":"端口位置"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Ethernet.Interface.{i}.PortLocation'
 WHERE c.command_code = 'LST ETHERNET_INTERFACE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'IP_VER', '{"en":"IpVer","zh":"IP地址版本"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Ethernet.IpRoute.{i}.IpVer'
 WHERE c.command_code = 'LST ETHERNET_IP_ROUTE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'DST_IP_NETWORK', '{"en":"DstIpNetwork","zh":"目的网段"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Ethernet.IpRoute.{i}.DstIpNetwork'
 WHERE c.command_code = 'LST ETHERNET_IP_ROUTE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PREFIX_LENGTH', '{"en":"PrefixLength","zh":"前缀长度"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Ethernet.IpRoute.{i}.PrefixLength'
 WHERE c.command_code = 'LST ETHERNET_IP_ROUTE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'GATEWAY_IP_ADDRESS', '{"en":"GatewayIpAddress","zh":"网关地址"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Ethernet.IpRoute.{i}.GatewayIpAddress'
 WHERE c.command_code = 'LST ETHERNET_IP_ROUTE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'INTERFACE_NAME', '{"en":"InterfaceName","zh":"端口名称"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Ethernet.IpRoute.{i}.InterfaceName'
 WHERE c.command_code = 'LST ETHERNET_IP_ROUTE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'LOCKED_LATITUDE', '{"en":"LockedLatitude","zh":"纬度"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.GPS.LockedLatitude'
 WHERE c.command_code = 'LST FAP_GPS'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'LOCKED_LONGITUDE', '{"en":"LockedLongitude","zh":"经度"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.GPS.LockedLongitude'
 WHERE c.command_code = 'LST FAP_GPS'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NUMBER_OF_SATELLITES', '{"en":"NumberOfSatellites","zh":"星个数"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.GPS.NumberOfSatellites'
 WHERE c.command_code = 'LST FAP_GPS'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SUPPORTED_ALARM_NUMBER_OF_ENTRIES', '{"en":"SupportedAlarmNumberOfEntries","zh":"最大告警实例数量"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.SupportedAlarmNumberOfEntries'
 WHERE c.command_code = 'LST FAULT_MGMT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MAX_CURRENT_ALARM_ENTRIES', '{"en":"MaxCurrentAlarmEntries","zh":"最大当前告警实例数量"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.MaxCurrentAlarmEntries'
 WHERE c.command_code = 'LST FAULT_MGMT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CURRENT_ALARM_NUMBER_OF_ENTRIES', '{"en":"CurrentAlarmNumberOfEntries","zh":"当前告警实例数量"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.CurrentAlarmNumberOfEntries'
 WHERE c.command_code = 'LST FAULT_MGMT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'HISTORY_EVENT_NUMBER_OF_ENTRIES', '{"en":"HistoryEventNumberOfEntries","zh":"历史事件实例数量"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.HistoryEventNumberOfEntries'
 WHERE c.command_code = 'LST FAULT_MGMT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'EXPEDITED_EVENT_NUMBER_OF_ENTRIES', '{"en":"ExpeditedEventNumberOfEntries","zh":"紧急事件实例数量"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.ExpeditedEventNumberOfEntries'
 WHERE c.command_code = 'LST FAULT_MGMT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'QUEUED_EVENT_NUMBER_OF_ENTRIES', '{"en":"QueuedEventNumberOfEntries","zh":"队列事件实例数量"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.QueuedEventNumberOfEntries'
 WHERE c.command_code = 'LST FAULT_MGMT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ALARM_IDENTIFIER', '{"en":"AlarmIdentifier","zh":"该告警的唯一序列号"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.CurrentAlarm.{i}.AlarmIdentifier'
 WHERE c.command_code = 'LST FAULT_MGMT_CURRENT_ALARM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ALARM_RAISED_TIME', '{"en":"AlarmRaisedTime","zh":"告警发生时间"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.CurrentAlarm.{i}.AlarmRaisedTime'
 WHERE c.command_code = 'LST FAULT_MGMT_CURRENT_ALARM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ALARM_CHANGED_TIME', '{"en":"AlarmChangedTime","zh":"告警改变时间"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.CurrentAlarm.{i}.AlarmChangedTime'
 WHERE c.command_code = 'LST FAULT_MGMT_CURRENT_ALARM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'FAULT_LOCATION', '{"en":"FaultLocation","zh":"告警源定位信息"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.CurrentAlarm.{i}.FaultLocation'
 WHERE c.command_code = 'LST FAULT_MGMT_CURRENT_ALARM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MANAGED_OBJECT_INSTANCE', '{"en":"ManagedObjectInstance","zh":"管理对象实例"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.CurrentAlarm.{i}.ManagedObjectInstance'
 WHERE c.command_code = 'LST FAULT_MGMT_CURRENT_ALARM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'EVENT_TYPE', '{"en":"EventType","zh":"告警类型"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.CurrentAlarm.{i}.EventType'
 WHERE c.command_code = 'LST FAULT_MGMT_CURRENT_ALARM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PROBABLE_CAUSE', '{"en":"ProbableCause","zh":"告警原因"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.CurrentAlarm.{i}.ProbableCause'
 WHERE c.command_code = 'LST FAULT_MGMT_CURRENT_ALARM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SPECIFIC_PROBLEM', '{"en":"SpecificProblem","zh":"告警描述"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.CurrentAlarm.{i}.SpecificProblem'
 WHERE c.command_code = 'LST FAULT_MGMT_CURRENT_ALARM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PERCEIVED_SEVERITY', '{"en":"PerceivedSeverity","zh":"当前告警级别"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.CurrentAlarm.{i}.PerceivedSeverity'
 WHERE c.command_code = 'LST FAULT_MGMT_CURRENT_ALARM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ADDITIONAL_TEXT', '{"en":"AdditionalText","zh":"告警附加文本"}'::jsonb, true, false, 10
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.CurrentAlarm.{i}.AdditionalText'
 WHERE c.command_code = 'LST FAULT_MGMT_CURRENT_ALARM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ADDITIONAL_INFORMATION', '{"en":"AdditionalInformation","zh":"告警附加信息"}'::jsonb, true, false, 11
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.CurrentAlarm.{i}.AdditionalInformation'
 WHERE c.command_code = 'LST FAULT_MGMT_CURRENT_ALARM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'EVENT_TIME', '{"en":"EventTime","zh":"告警发生、改变、清除的时间"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.ExpeditedEvent.{i}.EventTime'
 WHERE c.command_code = 'LST FAULT_MGMT_EXPEDITED_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ALARM_IDENTIFIER', '{"en":"AlarmIdentifier","zh":"该告警的唯一序列号"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.ExpeditedEvent.{i}.AlarmIdentifier'
 WHERE c.command_code = 'LST FAULT_MGMT_EXPEDITED_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NOTIFICATION_TYPE', '{"en":"NotificationType","zh":"通知类型"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.ExpeditedEvent.{i}.NotificationType'
 WHERE c.command_code = 'LST FAULT_MGMT_EXPEDITED_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'FAULT_LOCATION', '{"en":"FaultLocation","zh":"告警源定位信息"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.ExpeditedEvent.{i}.FaultLocation'
 WHERE c.command_code = 'LST FAULT_MGMT_EXPEDITED_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MANAGED_OBJECT_INSTANCE', '{"en":"ManagedObjectInstance","zh":"管理对象实例"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.ExpeditedEvent.{i}.ManagedObjectInstance'
 WHERE c.command_code = 'LST FAULT_MGMT_EXPEDITED_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'EVENT_TYPE', '{"en":"EventType","zh":"告警类型"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.ExpeditedEvent.{i}.EventType'
 WHERE c.command_code = 'LST FAULT_MGMT_EXPEDITED_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PROBABLE_CAUSE', '{"en":"ProbableCause","zh":"告警原因"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.ExpeditedEvent.{i}.ProbableCause'
 WHERE c.command_code = 'LST FAULT_MGMT_EXPEDITED_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SPECIFIC_PROBLEM', '{"en":"SpecificProblem","zh":"告警描述"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.ExpeditedEvent.{i}.SpecificProblem'
 WHERE c.command_code = 'LST FAULT_MGMT_EXPEDITED_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PERCEIVED_SEVERITY', '{"en":"PerceivedSeverity","zh":"告警级别"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.ExpeditedEvent.{i}.PerceivedSeverity'
 WHERE c.command_code = 'LST FAULT_MGMT_EXPEDITED_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ADDITIONAL_TEXT', '{"en":"AdditionalText","zh":"告警附加文本"}'::jsonb, true, false, 10
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.ExpeditedEvent.{i}.AdditionalText'
 WHERE c.command_code = 'LST FAULT_MGMT_EXPEDITED_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ADDITIONAL_INFORMATION', '{"en":"AdditionalInformation","zh":"告警附加信息"}'::jsonb, true, false, 11
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.ExpeditedEvent.{i}.AdditionalInformation'
 WHERE c.command_code = 'LST FAULT_MGMT_EXPEDITED_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'EVENT_TIME', '{"en":"EventTime","zh":"告警发生、改变、清除的时间"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.HistoryEvent.{i}.EventTime'
 WHERE c.command_code = 'LST FAULT_MGMT_HISTORY_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ALARM_IDENTIFIER', '{"en":"AlarmIdentifier","zh":"该告警的唯一序列号"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.HistoryEvent.{i}.AlarmIdentifier'
 WHERE c.command_code = 'LST FAULT_MGMT_HISTORY_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NOTIFICATION_TYPE', '{"en":"NotificationType","zh":"通知类型"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.HistoryEvent.{i}.NotificationType'
 WHERE c.command_code = 'LST FAULT_MGMT_HISTORY_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'FAULT_LOCATION', '{"en":"FaultLocation","zh":"告警源定位信息"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.HistoryEvent.{i}.FaultLocation'
 WHERE c.command_code = 'LST FAULT_MGMT_HISTORY_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MANAGED_OBJECT_INSTANCE', '{"en":"ManagedObjectInstance","zh":"管理对象实例"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.HistoryEvent.{i}.ManagedObjectInstance'
 WHERE c.command_code = 'LST FAULT_MGMT_HISTORY_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'EVENT_TYPE', '{"en":"EventType","zh":"告警类型"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.HistoryEvent.{i}.EventType'
 WHERE c.command_code = 'LST FAULT_MGMT_HISTORY_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PROBABLE_CAUSE', '{"en":"ProbableCause","zh":"告警原因"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.HistoryEvent.{i}.ProbableCause'
 WHERE c.command_code = 'LST FAULT_MGMT_HISTORY_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SPECIFIC_PROBLEM', '{"en":"SpecificProblem","zh":"告警描述"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.HistoryEvent.{i}.SpecificProblem'
 WHERE c.command_code = 'LST FAULT_MGMT_HISTORY_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PERCEIVED_SEVERITY', '{"en":"PerceivedSeverity","zh":"告警级别"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.HistoryEvent.{i}.PerceivedSeverity'
 WHERE c.command_code = 'LST FAULT_MGMT_HISTORY_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ADDITIONAL_TEXT', '{"en":"AdditionalText","zh":"告警附加文本"}'::jsonb, true, false, 10
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.HistoryEvent.{i}.AdditionalText'
 WHERE c.command_code = 'LST FAULT_MGMT_HISTORY_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ADDITIONAL_INFORMATION', '{"en":"AdditionalInformation","zh":"告警附加信息"}'::jsonb, true, false, 11
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.HistoryEvent.{i}.AdditionalInformation'
 WHERE c.command_code = 'LST FAULT_MGMT_HISTORY_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'EVENT_TIME', '{"en":"EventTime","zh":"告警发生、改变、清除的时间"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.QueuedEvent.{i}.EventTime'
 WHERE c.command_code = 'LST FAULT_MGMT_QUEUED_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ALARM_IDENTIFIER', '{"en":"AlarmIdentifier","zh":"该告警的唯一序列号"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.QueuedEvent.{i}.AlarmIdentifier'
 WHERE c.command_code = 'LST FAULT_MGMT_QUEUED_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NOTIFICATION_TYPE', '{"en":"NotificationType","zh":"通知类型"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.QueuedEvent.{i}.NotificationType'
 WHERE c.command_code = 'LST FAULT_MGMT_QUEUED_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'FAULT_LOCATION', '{"en":"FaultLocation","zh":"告警源定位信息"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.QueuedEvent.{i}.FaultLocation'
 WHERE c.command_code = 'LST FAULT_MGMT_QUEUED_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MANAGED_OBJECT_INSTANCE', '{"en":"ManagedObjectInstance","zh":"管理对象实例"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.QueuedEvent.{i}.ManagedObjectInstance'
 WHERE c.command_code = 'LST FAULT_MGMT_QUEUED_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'EVENT_TYPE', '{"en":"EventType","zh":"告警类型"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.QueuedEvent.{i}.EventType'
 WHERE c.command_code = 'LST FAULT_MGMT_QUEUED_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PROBABLE_CAUSE', '{"en":"ProbableCause","zh":"告警原因"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.QueuedEvent.{i}.ProbableCause'
 WHERE c.command_code = 'LST FAULT_MGMT_QUEUED_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SPECIFIC_PROBLEM', '{"en":"SpecificProblem","zh":"告警描述"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.QueuedEvent.{i}.SpecificProblem'
 WHERE c.command_code = 'LST FAULT_MGMT_QUEUED_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PERCEIVED_SEVERITY', '{"en":"PerceivedSeverity","zh":"告警级别"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.QueuedEvent.{i}.PerceivedSeverity'
 WHERE c.command_code = 'LST FAULT_MGMT_QUEUED_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ADDITIONAL_TEXT', '{"en":"AdditionalText","zh":"告警附加文本"}'::jsonb, true, false, 10
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.QueuedEvent.{i}.AdditionalText'
 WHERE c.command_code = 'LST FAULT_MGMT_QUEUED_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ADDITIONAL_INFORMATION', '{"en":"AdditionalInformation","zh":"告警附加信息"}'::jsonb, true, false, 11
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.QueuedEvent.{i}.AdditionalInformation'
 WHERE c.command_code = 'LST FAULT_MGMT_QUEUED_EVENT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'EVENT_TYPE', '{"en":"EventType","zh":"告警类型"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.SupportedAlarm.{i}.EventType'
 WHERE c.command_code = 'LST FAULT_MGMT_SUPPORTED_ALARM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PROBABLE_CAUSE', '{"en":"ProbableCause","zh":"告警原因"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.SupportedAlarm.{i}.ProbableCause'
 WHERE c.command_code = 'LST FAULT_MGMT_SUPPORTED_ALARM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SPECIFIC_PROBLEM', '{"en":"SpecificProblem","zh":"告警描述"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.SupportedAlarm.{i}.SpecificProblem'
 WHERE c.command_code = 'LST FAULT_MGMT_SUPPORTED_ALARM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PERCEIVED_SEVERITY', '{"en":"PerceivedSeverity","zh":"支持告警级别"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.SupportedAlarm.{i}.PerceivedSeverity'
 WHERE c.command_code = 'LST FAULT_MGMT_SUPPORTED_ALARM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'REPORTING_MECHANISM', '{"en":"ReportingMechanism","zh":"告警上报机制"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.SupportedAlarm.{i}.ReportingMechanism'
 WHERE c.command_code = 'LST FAULT_MGMT_SUPPORTED_ALARM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T_RESELECTION_UTRA', '{"en":"TReselectionUTRA","zh":"UTRAN小区重选时间"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.TReselectionUTRA'
 WHERE c.command_code = 'LST IDLE_MODE_IRAT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T_RESELECTION_GERAN', '{"en":"TReselectionGERAN","zh":"GERAN小区重选时间"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.TReselectionGERAN'
 WHERE c.command_code = 'LST IDLE_MODE_IRAT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ENABLE', '{"en":"Enable","zh":"开关"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.IPsec.Enable'
 WHERE c.command_code = 'LST I_PSEC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MY_KEY_MODE', '{"en":"MyKeyMode","zh":"鉴权方式"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.IPsec.MyKeyMode'
 WHERE c.command_code = 'LST I_PSEC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'STATUS', '{"en":"Status","zh":"状态"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.IPsec.Status'
 WHERE c.command_code = 'LST I_PSEC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'AH_SUPPORTED', '{"en":"AHSupported","zh":"AH是否支持"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.IPsec.AHSupported'
 WHERE c.command_code = 'LST I_PSEC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'IK_EV2_SUPPORTED_ENCRYPTION_ALGORITHMS', '{"en":"IKEv2SupportedEncryptionAlgorithms","zh":"IKE2加密算法"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.IPsec.IKEv2SupportedEncryptionAlgorithms'
 WHERE c.command_code = 'LST I_PSEC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ESP_SUPPORTED_ENCRYPTION_ALGORITHMS', '{"en":"ESPSupportedEncryptionAlgorithms","zh":"ESP加密算法"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.IPsec.ESPSupportedEncryptionAlgorithms'
 WHERE c.command_code = 'LST I_PSEC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'IK_EV2_SUPPORTED_PSEUDO_RANDOM_FUNCTIONS', '{"en":"IKEv2SupportedPseudoRandomFunctions","zh":"随机函数"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.IPsec.IKEv2SupportedPseudoRandomFunctions'
 WHERE c.command_code = 'LST I_PSEC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SUPPORTED_INTEGRITY_ALGORITHMS', '{"en":"SupportedIntegrityAlgorithms","zh":"完整性算法"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.IPsec.SupportedIntegrityAlgorithms'
 WHERE c.command_code = 'LST I_PSEC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SUPPORTED_DIFFIE_HELLMAN_GROUP_TRANSFORMS', '{"en":"SupportedDiffieHellmanGroupTransforms","zh":"Diffie-Hellman交换"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.IPsec.SupportedDiffieHellmanGroupTransforms'
 WHERE c.command_code = 'LST I_PSEC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PERIODIC_UPLOAD_ENABLE', '{"en":"PeriodicUploadEnable","zh":"日志周期上传使能开关"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.LogMgmt.PeriodicUploadEnable'
 WHERE c.command_code = 'LST LOG_MGMT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'URL', '{"en":"URL","zh":"日志上传URL"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.LogMgmt.URL'
 WHERE c.command_code = 'LST LOG_MGMT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'USERNAME', '{"en":"Username","zh":"日志管理用户名"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.LogMgmt.Username'
 WHERE c.command_code = 'LST LOG_MGMT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PASSWORD', '{"en":"Password","zh":"日志管理密码"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.LogMgmt.Password'
 WHERE c.command_code = 'LST LOG_MGMT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PERIODIC_UPLOAD_INTERVAL', '{"en":"PeriodicUploadInterval","zh":"日志周期上传时间间隔"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.LogMgmt.PeriodicUploadInterval'
 WHERE c.command_code = 'LST LOG_MGMT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ADMIN_STATE', '{"en":"AdminState","zh":"基站管理状态"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.AdminState'
 WHERE c.command_code = 'LST LTE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'OP_STATE', '{"en":"OpState","zh":"基站运行状态"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.OpState'
 WHERE c.command_code = 'LST LTE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'RF_TX_STATUS', '{"en":"RFTxStatus","zh":"基站射频状态"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.RFTxStatus'
 WHERE c.command_code = 'LST LTE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'EAID', '{"en":"EAID","zh":"EPCEAID"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.EPC.EAID'
 WHERE c.command_code = 'LST LTE_EPC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'TAC', '{"en":"TAC","zh":"TAC"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.EPC.TAC'
 WHERE c.command_code = 'LST LTE_EPC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SEC_GW_SERVER1', '{"en":"SecGWServer1","zh":"安全网关1"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.Gateway.SecGWServer1'
 WHERE c.command_code = 'LST LTE_GATEWAY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SEC_GW_SERVER2', '{"en":"SecGWServer2","zh":"安全网关2"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.Gateway.SecGWServer2'
 WHERE c.command_code = 'LST LTE_GATEWAY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SEC_GW_SERVER3', '{"en":"SecGWServer3","zh":"安全网关3"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.Gateway.SecGWServer3'
 WHERE c.command_code = 'LST LTE_GATEWAY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'AG_SERVER_ENABLE', '{"en":"AGServerEnable","zh":"HeGW方式接入"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.Gateway.AGServerEnable'
 WHERE c.command_code = 'LST LTE_GATEWAY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'AG_SERVER_IP1', '{"en":"AGServerIp1","zh":"接入网关1"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.Gateway.AGServerIp1'
 WHERE c.command_code = 'LST LTE_GATEWAY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'AG_SERVER_IP2', '{"en":"AGServerIp2","zh":"接入网关2"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.Gateway.AGServerIp2'
 WHERE c.command_code = 'LST LTE_GATEWAY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'AG_SERVER_IP3', '{"en":"AGServerIp3","zh":"接入网关3"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.Gateway.AGServerIp3'
 WHERE c.command_code = 'LST LTE_GATEWAY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'AG_PORT1', '{"en":"AGPort1","zh":"接入网关端口1"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.Gateway.AGPort1'
 WHERE c.command_code = 'LST LTE_GATEWAY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'AG_PORT2', '{"en":"AGPort2","zh":"接入网关端口2"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.Gateway.AGPort2'
 WHERE c.command_code = 'LST LTE_GATEWAY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'AG_PORT3', '{"en":"AGPort3","zh":"接入网关端口3"}'::jsonb, true, false, 10
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.Gateway.AGPort3'
 WHERE c.command_code = 'LST LTE_GATEWAY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PLMNID_LIST', '{"en":"PLMNIDList","zh":"MMEPLMN标识"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.PLMNID'
 WHERE c.command_code = 'LST LTE_MME_POOL_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MME_GROUP_ID', '{"en":"MMEGroupID","zh":"MMEGroup标识"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.MMEGroupID'
 WHERE c.command_code = 'LST LTE_MME_POOL_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MME_CODE', '{"en":"MMECode","zh":"MME代码"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.MMECode'
 WHERE c.command_code = 'LST LTE_MME_POOL_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MME_IP1', '{"en":"MMEIp1","zh":"MMEIP1"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.MMEIp1'
 WHERE c.command_code = 'LST LTE_MME_POOL_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MME_IP2', '{"en":"MMEIp2","zh":"MMEIP2"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.MMEIp2'
 WHERE c.command_code = 'LST LTE_MME_POOL_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'LOC_IP_ADDR_LIST', '{"en":"LocIpAddrList","zh":"本地IP地址列表"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.S1U.{i}.LocIpAddrList'
 WHERE c.command_code = 'LST LTE_S1U'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'FAR_IP_SUBNETWORK_LIST', '{"en":"FarIpSubnetworkList","zh":"远端IP地址列表"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.S1U.{i}.FarIpSubnetworkList'
 WHERE c.command_code = 'LST LTE_S1U'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'URL', '{"en":"URL","zh":"网管服务器URL"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.URL'
 WHERE c.command_code = 'LST MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'USERNAME', '{"en":"Username","zh":"连接用户名"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.Username'
 WHERE c.command_code = 'LST MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PASSWORD', '{"en":"Password","zh":"连接密码"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.Password'
 WHERE c.command_code = 'LST MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PERIODIC_INFORM_ENABLE', '{"en":"PeriodicInformEnable","zh":"连接使能开关"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.PeriodicInformEnable'
 WHERE c.command_code = 'LST MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PERIODIC_INFORM_TIME', '{"en":"PeriodicInformTime","zh":"上报周期"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.PeriodicInformTime'
 WHERE c.command_code = 'LST MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PERIODIC_INFORM_INTERVAL', '{"en":"PeriodicInformInterval","zh":"周期上报时间间隔"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.PeriodicInformInterval'
 WHERE c.command_code = 'LST MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PARAMETER_KEY', '{"en":"ParameterKey","zh":"键值"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.ParameterKey'
 WHERE c.command_code = 'LST MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CONNECTION_REQUEST_URL', '{"en":"ConnectionRequestURL","zh":"连接请求URL"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.ConnectionRequestURL'
 WHERE c.command_code = 'LST MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CONNECTION_REQUEST_USERNAME', '{"en":"ConnectionRequestUsername","zh":"连接请求用户名"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.ConnectionRequestUsername'
 WHERE c.command_code = 'LST MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CONNECTION_REQUEST_PASSWORD', '{"en":"ConnectionRequestPassword","zh":"连接请求密码"}'::jsonb, true, false, 10
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.ConnectionRequestPassword'
 WHERE c.command_code = 'LST MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'UDP_CONNECTION_REQUEST_ADDRESS', '{"en":"UDPConnectionRequestAddress","zh":"UDP连接请求地址"}'::jsonb, true, false, 11
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.UDPConnectionRequestAddress'
 WHERE c.command_code = 'LST MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'STUN_ENABLE', '{"en":"STUNEnable","zh":"STUN使能开关"}'::jsonb, true, false, 12
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.STUNEnable'
 WHERE c.command_code = 'LST MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'STUN_SERVER_ADDRESS', '{"en":"STUNServerAddress","zh":"STUN服务器地址"}'::jsonb, true, false, 13
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.STUNServerAddress'
 WHERE c.command_code = 'LST MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'STUN_SERVER_PORT', '{"en":"STUNServerPort","zh":"STUN服务器端口"}'::jsonb, true, false, 14
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.STUNServerPort'
 WHERE c.command_code = 'LST MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'STUN_USERNAME', '{"en":"STUNUsername","zh":"STUN用户名"}'::jsonb, true, false, 15
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.STUNUsername'
 WHERE c.command_code = 'LST MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'STUN_PASSWORD', '{"en":"STUNPassword","zh":"STUN密码"}'::jsonb, true, false, 16
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.STUNPassword'
 WHERE c.command_code = 'LST MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'STUN_MAXIMUM_KEEP_ALIVE_PERIOD', '{"en":"STUNMaximumKeepAlivePeriod","zh":"STUN最大生存周期"}'::jsonb, true, false, 17
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.STUNMaximumKeepAlivePeriod'
 WHERE c.command_code = 'LST MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'STUN_MINIMUM_KEEP_ALIVE_PERIOD', '{"en":"STUNMinimumKeepAlivePeriod","zh":"STUN最小生存周期"}'::jsonb, true, false, 18
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.STUNMinimumKeepAlivePeriod'
 WHERE c.command_code = 'LST MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NAT_DETECTED', '{"en":"NATDetected","zh":"NAT转换地址"}'::jsonb, true, false, 19
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.NATDetected'
 WHERE c.command_code = 'LST MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'QHYST', '{"en":"Qhyst","zh":"服务小区重选迟滞值"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.Qhyst'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'INTRA_FREQ_RESELECTION', '{"en":"IntraFreqReselection","zh":"同频重选指示"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.IntraFreqReselection'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'Q_HYST_SF_MEDIUM', '{"en":"QHystSFMedium","zh":"Qhyst比例因子(中速)"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.QHystSFMedium'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'Q_HYST_SF_HIGH', '{"en":"QHystSFHigh","zh":"Qhyst比例因子(高速)"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.QHystSFHigh'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T_EVALUATION', '{"en":"TEvaluation","zh":"允许小区重选数目的间隔时间"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.TEvaluation'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T_HYST_NORMAL', '{"en":"THystNormal","zh":"正常状态附加判决时长"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.THystNormal'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'N_CELL_CHANGE_MEDIUM', '{"en":"NCellChangeMedium","zh":"进入中速重选次数门限"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.NCellChangeMedium'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'N_CELL_CHANGE_HIGH', '{"en":"NCellChangeHigh","zh":"进入高速重选次数门限"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.NCellChangeHigh'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'Q_RX_LEV_MIN_SIB1', '{"en":"QRxLevMinSIB1","zh":"服务小区最小接收电平"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QRxLevMinSIB1'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'Q_RX_LEV_MIN_SIB3', '{"en":"QRxLevMinSIB3","zh":"同频邻区最小接收电平"}'::jsonb, true, false, 10
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QRxLevMinSIB3'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'Q_RX_LEV_MIN_OFFSET', '{"en":"QRxLevMinOffset","zh":"最小接收电平偏移量"}'::jsonb, true, false, 11
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QRxLevMinOffset'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'S_INTRA_SEARCH', '{"en":"SIntraSearch","zh":"同频测量启动门限"}'::jsonb, true, false, 12
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SIntraSearch'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T_RESELECTION_EUTRA', '{"en":"TReselectionEUTRA","zh":"同频小区重选时间"}'::jsonb, true, false, 13
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRA'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'S_NON_INTRA_SEARCH', '{"en":"SNonIntraSearch","zh":"非同频测量启动门限"}'::jsonb, true, false, 14
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SNonIntraSearch'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'S_NON_INTRA_SEARCH_PR9', '{"en":"SNonIntraSearchPR9","zh":"异频/异系统RSRP测量启动门限"}'::jsonb, true, false, 15
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SNonIntraSearchPR9'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'S_NON_INTRA_SEARCH_QR9', '{"en":"SNonIntraSearchQR9","zh":"异频/异系统RSRQ测量启动门限"}'::jsonb, true, false, 16
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SNonIntraSearchQR9'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CELL_RESELECTION_PRIORITY', '{"en":"CellReselectionPriority","zh":"同频小区重选优先级"}'::jsonb, true, false, 17
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.CellReselectionPriority'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'P_MAX', '{"en":"PMax","zh":"UE允许最大发射功率"}'::jsonb, true, false, 18
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.PMax'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'THRESH_SERVING_LOW', '{"en":"ThreshServingLow","zh":"低优先级重选门限"}'::jsonb, true, false, 19
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.ThreshServingLow'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'THRESH_SERVING_LOW_QR9', '{"en":"ThreshServingLowQR9","zh":"服务频点低优先级RSRQ重选门限"}'::jsonb, true, false, 20
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.ThreshServingLowQR9'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T_RESELECTION_EUTRASF_MEDIUM', '{"en":"TReselectionEUTRASFMedium","zh":"TReselectionUTRA比例因子(中速)"}'::jsonb, true, false, 21
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRASFMedium'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T_RESELECTION_EUTRASF_HIGH', '{"en":"TReselectionEUTRASFHigh","zh":"TReselectionUTRA比例因子(高速)"}'::jsonb, true, false, 22
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRASFHigh'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'S_INTRA_SEARCH_PR9', '{"en":"SIntraSearchPR9","zh":"同频RSRP测量启动门限"}'::jsonb, true, false, 23
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SIntraSearchPR9'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'S_INTRA_SEARCH_QR9', '{"en":"SIntraSearchQR9","zh":"同频RSRQ测量启动门限"}'::jsonb, true, false, 24
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SIntraSearchQR9'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'Q_QUAL_MIN_R9_RESELECTION', '{"en":"QQualMinR9Reselection","zh":"小区重选最低接入信号质量"}'::jsonb, true, false, 25
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QQualMinR9Reselection'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'Q_QUAL_MIN_R9_SELECTION', '{"en":"QQualMinR9Selection","zh":"小区选择最低接入信号质量"}'::jsonb, true, false, 26
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QQualMinR9Selection'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'Q_QUAL_MIN_OFFSET_R9', '{"en":"QQualMinOffsetR9","zh":"小区选择最低接入信号质量偏置"}'::jsonb, true, false, 27
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QQualMinOffsetR9'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ALLOWED_MEAS_BANDWIDTH', '{"en":"AllowedMeasBandwidth","zh":"测量带宽"}'::jsonb, true, false, 28
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.AllowedMeasBandwidth'
 WHERE c.command_code = 'LST MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MR_ENABLE', '{"en":"MrEnable","zh":"MR开关"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.MrEnable'
 WHERE c.command_code = 'LST MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MR_URL', '{"en":"MrUrl","zh":"Url地址"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.MrUrl'
 WHERE c.command_code = 'LST MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MR_USERNAME', '{"en":"MrUsername","zh":"用户名"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.MrUsername'
 WHERE c.command_code = 'LST MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MR_PASSWORD', '{"en":"MrPassword","zh":"密码"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.MrPassword'
 WHERE c.command_code = 'LST MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MEASURE_TYPE', '{"en":"MeasureType","zh":"MR文件类型"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.MeasureType'
 WHERE c.command_code = 'LST MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'OMC_NAME', '{"en":"OmcName","zh":"OMC-R名称"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.OmcName'
 WHERE c.command_code = 'LST MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SAMPLE_PERIOD', '{"en":"SamplePeriod","zh":"MR采样周期"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.SamplePeriod'
 WHERE c.command_code = 'LST MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'UPLOAD_PERIOD', '{"en":"UploadPeriod","zh":"MR采集周期"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.UploadPeriod'
 WHERE c.command_code = 'LST MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SAMPLE_BEGIN_TIME', '{"en":"SampleBeginTime","zh":"绝对时间参考"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.SampleBeginTime'
 WHERE c.command_code = 'LST MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SAMPLE_END_TIME', '{"en":"SampleEndTime","zh":"绝对时间参考"}'::jsonb, true, false, 10
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.SampleEndTime'
 WHERE c.command_code = 'LST MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PRB_NUM', '{"en":"PrbNum","zh":"子帧的PRB"}'::jsonb, true, false, 11
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.PrbNum'
 WHERE c.command_code = 'LST MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SUB_FRAME_NUM', '{"en":"SubFrameNum","zh":"子帧数"}'::jsonb, true, false, 12
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.SubFrameNum'
 WHERE c.command_code = 'LST MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MRECGI_LIST', '{"en":"MRECGIList","zh":"MR小区列表"}'::jsonb, true, false, 13
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.MRECGIList'
 WHERE c.command_code = 'LST MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MEASURE_ITEMS', '{"en":"MeasureItems","zh":"测量项"}'::jsonb, true, false, 14
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.MeasureItems'
 WHERE c.command_code = 'LST MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'STAGE', '{"en":"Stage","zh":"升级阶段"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.SwUpgrade.Stage'
 WHERE c.command_code = 'LST MU_SW_UPGRADE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'FAILURE_CAUSE', '{"en":"FailureCause","zh":"升级失败原因"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.SwUpgrade.FailureCause'
 WHERE c.command_code = 'LST MU_SW_UPGRADE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'STATUS', '{"en":"Status","zh":"状态"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.SwUpgrade.Status'
 WHERE c.command_code = 'LST MU_SW_UPGRADE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ENABLE', '{"en":"Enable","zh":"文件周期上传使能开关"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.PerfMgmt.Config.{i}.Enable'
 WHERE c.command_code = 'LST PERF_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ALIAS', '{"en":"Alias","zh":"别名"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.PerfMgmt.Config.{i}.Alias'
 WHERE c.command_code = 'LST PERF_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'URL', '{"en":"URL","zh":"文件管理URL"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.PerfMgmt.Config.{i}.URL'
 WHERE c.command_code = 'LST PERF_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'USERNAME', '{"en":"Username","zh":"文件管理用户名"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.PerfMgmt.Config.{i}.Username'
 WHERE c.command_code = 'LST PERF_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PASSWORD', '{"en":"Password","zh":"文件管理密码"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.PerfMgmt.Config.{i}.Password'
 WHERE c.command_code = 'LST PERF_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PERIODIC_UPLOAD_INTERVAL', '{"en":"PeriodicUploadInterval","zh":"文件周期上传时间间隔"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.PerfMgmt.Config.{i}.PeriodicUploadInterval'
 WHERE c.command_code = 'LST PERF_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PERIODIC_UPLOAD_TIME', '{"en":"PeriodicUploadTime","zh":"文件上传时间"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.PerfMgmt.Config.{i}.PeriodicUploadTime'
 WHERE c.command_code = 'LST PERF_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'REPLENISH_ENABLE', '{"en":"ReplenishEnable","zh":"补采开关"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.PerfMgmt.Config.{i}.ReplenishEnable'
 WHERE c.command_code = 'LST PERF_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'REPLENISH_START_TIME', '{"en":"ReplenishStartTime","zh":"补采开始时间"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.PerfMgmt.Config.{i}.ReplenishStartTime'
 WHERE c.command_code = 'LST PERF_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'REPLENISH_END_TIME', '{"en":"ReplenishEndTime","zh":"补采结束时间"}'::jsonb, true, false, 10
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.PerfMgmt.Config.{i}.ReplenishEndTime'
 WHERE c.command_code = 'LST PERF_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NEIGH_CELL_CONFIG', '{"en":"NeighCellConfig","zh":"邻区配置"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.NeighCellConfig'
 WHERE c.command_code = 'LST PHY_MBSFN'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NUMBER_OF_RA_PREAMBLES', '{"en":"NumberOfRaPreambles","zh":"竞争随机接入前导码数"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.NumberOfRaPreambles'
 WHERE c.command_code = 'LST RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SIZE_OF_RA_GROUP_A', '{"en":"SizeOfRaGroupA","zh":"随机接入前导码组A大小"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.SizeOfRaGroupA'
 WHERE c.command_code = 'LST RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MESSAGE_SIZE_GROUP_A', '{"en":"MessageSizeGroupA","zh":"组A信息大小"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MessageSizeGroupA'
 WHERE c.command_code = 'LST RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MESSAGE_POWER_OFFSET_GROUP_B', '{"en":"MessagePowerOffsetGroupB","zh":"组B功率补偿信息"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MessagePowerOffsetGroupB'
 WHERE c.command_code = 'LST RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'POWER_RAMPING_STEP', '{"en":"PowerRampingStep","zh":"功率增加补偿"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.PowerRampingStep'
 WHERE c.command_code = 'LST RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PREAMBLE_INITIAL_RECEIVED_TARGET_POWER', '{"en":"PreambleInitialReceivedTargetPower","zh":"前导码初始接收目标功率"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.PreambleInitialReceivedTargetPower'
 WHERE c.command_code = 'LST RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PREAMBLE_TRANS_MAX', '{"en":"PreambleTransMax","zh":"最大随机接入次数"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.PreambleTransMax'
 WHERE c.command_code = 'LST RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'RESPONSE_WINDOW_SIZE', '{"en":"ResponseWindowSize","zh":"随机接入响应窗口大小"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.ResponseWindowSize'
 WHERE c.command_code = 'LST RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CONTENTION_RESLUTION_TIMER', '{"en":"ContentionReslutionTimer","zh":"冲突解决定时器"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.ContentionResolutionTimer'
 WHERE c.command_code = 'LST RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MAX_HARQ_MSG3_TX', '{"en":"MaxHARQMsg3Tx","zh":"Msg3HARQ最大传输次数"}'::jsonb, true, false, 10
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MaxHARQMsg3Tx'
 WHERE c.command_code = 'LST RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'DRX_ENABLED', '{"en":"DRXEnabled","zh":"DRX功能开关"}'::jsonb, true, false, 11
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.DRX.DRXEnabled'
 WHERE c.command_code = 'LST RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MAX_HARQ_TX', '{"en":"MaxHARQTx","zh":"上行HARQ最大传输次数"}'::jsonb, true, false, 12
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.MaxHARQTx'
 WHERE c.command_code = 'LST RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PERIODIC_BSR_TIMER', '{"en":"PeriodicBSRTimer","zh":"周期性BSR定时器"}'::jsonb, true, false, 13
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.PeriodicBSRTimer'
 WHERE c.command_code = 'LST RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'RETX_BSR_TIMER', '{"en":"RetxBSRTimer","zh":"重传BSR定时器"}'::jsonb, true, false, 14
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.RetxBSRTimer'
 WHERE c.command_code = 'LST RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'TTI_BUNDLING', '{"en":"TTIBundling","zh":"TTI绑定功能开关"}'::jsonb, true, false, 15
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.TTIBundling'
 WHERE c.command_code = 'LST RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MAX_UE_PER_UL_SF', '{"en":"MaxUePerUlSf","zh":"上行单帧最大调度用户数"}'::jsonb, true, false, 16
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.MaxUePerUlSf'
 WHERE c.command_code = 'LST RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CONFIGURATION_INDEX', '{"en":"ConfigurationIndex","zh":"PRACH配置索引"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.ConfigurationIndex'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'FREQ_OFFSET', '{"en":"FreqOffset","zh":"PRACH频率偏移"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.FreqOffset'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'HIGH_SPEED_FLAG', '{"en":"HighSpeedFlag","zh":"高速状态标识"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.HighSpeedFlag'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ROOT_SEQUENCE_INDEX', '{"en":"RootSequenceIndex","zh":"逻辑根序列索引"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.RootSequenceIndex'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ZERO_CORRELATION_ZONE_CONFIG', '{"en":"ZeroCorrelationZoneConfig","zh":"零相关配置"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.ZeroCorrelationZoneConfig'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SRS_ENABLED', '{"en":"SRSEnabled","zh":"SRS使能开关"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSEnabled'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SRS_BANDWIDTH_CONFIG', '{"en":"SRSBandwidthConfig","zh":"SRS带宽配置"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSBandwidthConfig'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SRS_MAX_UP_PTS', '{"en":"SRSMaxUpPTS","zh":"SRSUpPTS带宽重配指示"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSMaxUpPTS'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ACK_NACK_SRS_SIMULTANEOUS_TRANSMISSION', '{"en":"AckNackSRSSimultaneousTransmission","zh":"SRS/ACK/NACK同时传输标识"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.AckNackSRSSimultaneousTransmission'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'DELTA_PUCCH_SHIFT', '{"en":"DeltaPUCCHShift","zh":"PUCCH循环移位间隔"}'::jsonb, true, false, 10
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.DeltaPUCCHShift'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NRBCQI', '{"en":"NRBCQI","zh":"PUCCH2/2a/2b占用RB数"}'::jsonb, true, false, 11
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.NRBCQI'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NCSAN', '{"en":"NCSAN","zh":"PUCCH1/1a/1b预留资源数"}'::jsonb, true, false, 12
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.NCSAN'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'N1PUCCHAN', '{"en":"N1PUCCHAN","zh":"SPSACK/NACK及SR预留资源数"}'::jsonb, true, false, 13
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.N1PUCCHAN'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CQIPUCCH_RESOURCE_INDEX', '{"en":"CQIPUCCHResourceIndex","zh":"CQI报告使用PUCCH资源索引"}'::jsonb, true, false, 14
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.CQIPUCCHResourceIndex'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'K', '{"en":"K","zh":"子带CQI报告带宽"}'::jsonb, true, false, 15
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.K'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PUSCH_POWER_CTRL_SWITCH', '{"en":"PUSCHPowerCtrlSwitch","zh":"上行PUSCH闭环功控开关"}'::jsonb, true, false, 16
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PaParam.PUSCHPowerCtrlSwitch'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PUCCH_POWER_CTRL_SWITCH', '{"en":"PUCCHPowerCtrlSwitch","zh":"上行PUCCH闭环功控开关"}'::jsonb, true, false, 17
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PaParam.PUCCHPowerCtrlSwitch'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ENABLE64QAM', '{"en":"Enable64QAM","zh":"PUSCH64QAM使能开关"}'::jsonb, true, false, 18
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.Enable64QAM'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'HOPPING_MODE', '{"en":"HoppingMode","zh":"PUSCH跳频模式"}'::jsonb, true, false, 19
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.HoppingMode'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'HOPPING_OFFSET', '{"en":"HoppingOffset","zh":"PUSCH跳频偏置"}'::jsonb, true, false, 20
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.HoppingOffset'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NSB', '{"en":"NSB","zh":"跳频子带个数"}'::jsonb, true, false, 21
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.NSB'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NUM_PRS_RESOURCE_BLOCKS', '{"en":"NumPRSResourceBlocks","zh":"PRSRB数目"}'::jsonb, true, false, 22
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRS.NumPRSResourceBlocks'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PRS_CONFIGURATION_INDEX', '{"en":"PRSConfigurationIndex","zh":"PRS配置索引"}'::jsonb, true, false, 23
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRS.PRSConfigurationIndex'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NUM_CONSECUTIVE_PRS_SUBFAMES', '{"en":"NumConsecutivePRSSubfames","zh":"PRS连续子帧数"}'::jsonb, true, false, 24
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRS.NumConsecutivePRSSubfames'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SPECIAL_SUBFRAME_PATTERNS', '{"en":"SpecialSubframePatterns","zh":"特殊子帧配置"}'::jsonb, true, false, 25
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.TDDFrame.SpecialSubframePatterns'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SUB_FRAME_ASSIGNMENT', '{"en":"SubFrameAssignment","zh":"上下行子帧配置"}'::jsonb, true, false, 26
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.TDDFrame.SubFrameAssignment'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PB', '{"en":"Pb","zh":"天线端口信号功率比"}'::jsonb, true, false, 27
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PDSCH.Pb'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PA', '{"en":"Pa","zh":"小区PDSCH采用固定功率分配时的PA取值"}'::jsonb, true, false, 28
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PDSCH.Pa'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'P0_NOMINAL_PUSCH_PERSISTENT', '{"en":"P0NominalPUSCHPersistent","zh":"持续调度期望接收功率"}'::jsonb, true, false, 29
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.P0NominalPUSCHPersistent'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'P0_NOMINAL_PUSCH', '{"en":"P0NominalPUSCH","zh":"非持续调度期望功率"}'::jsonb, true, false, 30
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.P0NominalPUSCH'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ALPHA', '{"en":"Alpha","zh":"部分路损补偿系数"}'::jsonb, true, false, 31
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.Alpha'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'P0_NOMINAL_PUCCH', '{"en":"P0NominalPUCCH","zh":"PUCCH期望功率"}'::jsonb, true, false, 32
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.P0NominalPUCCH'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'DELTA_MCS_ENABLED', '{"en":"DeltaMCSEnabled","zh":"MCS补偿值"}'::jsonb, true, false, 33
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.DeltaMCSEnabled'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NUM_OF_TX_ANTENNA', '{"en":"NumOfTxAntenna","zh":"发射通道数"}'::jsonb, true, false, 34
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.Antenna.NumOfTxAntenna'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NUM_OF_RX_ANTENNA', '{"en":"NumOfRxAntenna","zh":"接收通道数"}'::jsonb, true, false, 35
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.Antenna.NumOfRxAntenna'
 WHERE c.command_code = 'LST RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SCTP_ASSOC_LOCAL_ADDR', '{"en":"SCTPAssocLocalAddr","zh":"本端IP地址"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.Transport.SCTP.Assoc.{i}.SCTPAssocLocalAddr'
 WHERE c.command_code = 'LST SCTP_ASSOC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'LOCAL_PORT', '{"en":"LocalPort","zh":"本端端口"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.Transport.SCTP.Assoc.{i}.LocalPort'
 WHERE c.command_code = 'LST SCTP_ASSOC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PRIMARY_PEER_ADDRESS', '{"en":"PrimaryPeerAddress","zh":"远端IP地址"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.Transport.SCTP.Assoc.{i}.PrimaryPeerAddress'
 WHERE c.command_code = 'LST SCTP_ASSOC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'REMOTE_PORT', '{"en":"RemotePort","zh":"远端端口"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.Transport.SCTP.Assoc.{i}.RemotePort'
 WHERE c.command_code = 'LST SCTP_ASSOC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'STAGE', '{"en":"Stage","zh":"开站阶段"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.SelfConfig.Startup.Stage'
 WHERE c.command_code = 'LST SELF_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'STATUS', '{"en":"Status","zh":"开站状态"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.SelfConfig.Startup.Status'
 WHERE c.command_code = 'LST SELF_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'FAILURE_CAUSE', '{"en":"FailureCause","zh":"开站失败原因"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.SelfConfig.Startup.FailureCause'
 WHERE c.command_code = 'LST SELF_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SON_SYS_MODE', '{"en":"SONSysMode","zh":"SON系统模式"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SONSysMode'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SON_WORK_MODE', '{"en":"SONWorkMode","zh":"SON模式设置"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SONWorkMode'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PCI_OPT_ENABLE', '{"en":"PCIOptEnable","zh":"PCI自优化算法开关"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.PCIOptEnable'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PCI_RECONFIG_WAIT_TIME', '{"en":"PCIReconfigWaitTime","zh":"PCI重配等待定时器"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.PCIReconfigWaitTime'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CANDIDATE_ARFCN_LIST', '{"en":"CandidateARFCNList","zh":"候选频点列表"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.CandidateARFCNList'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CANDIDATE_PCI_LIST', '{"en":"CandidatePCIList","zh":"候选PCI列表"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.CandidatePCIList'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ANR_ENABLE', '{"en":"ANREnable","zh":"ANR算法总开关"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANREnable'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ANR_INTER_FEQ_ENABLE', '{"en":"ANRInterFeqEnable","zh":"E-UTRAN异频ANR算法开关"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANRInterFeqEnable'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ANRGERAN_ENABLE', '{"en":"ANRGERANEnable","zh":"GERAN异系统ANR算法开关"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANRGERANEnable'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ANRUTRAN_ENABLE', '{"en":"ANRUTRANEnable","zh":"UTRAN异系统ANR算法开关"}'::jsonb, true, false, 10
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANRUTRANEnable'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ARFCN_ENABLE', '{"en":"ARFCNEnable","zh":"频点自配置算法开关"}'::jsonb, true, false, 11
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ARFCNEnable'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MAX_LTE_NEIGHBOUR_CELL_NUM', '{"en":"MaxLTENeighbourCellNum","zh":"最大LTE邻区数"}'::jsonb, true, false, 12
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MaxLTENeighbourCellNum'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MAX_UTRAN_NEIGHBOUR_CELL_NUM', '{"en":"MaxUTRANNeighbourCellNum","zh":"最大UTRAN邻区数"}'::jsonb, true, false, 13
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MaxUTRANNeighbourCellNum'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MAX_GERAN_NEIGHBOUR_CELL_NUM', '{"en":"MaxGERANNeighbourCellNum","zh":"最大GERAN邻区数"}'::jsonb, true, false, 14
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MaxGRANNeighbourCellNum'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'RE_SYN_CELL_ENABLE', '{"en":"ReSynCellEnable","zh":"重新同步小区使能开关"}'::jsonb, true, false, 15
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ReSynCellEnable'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'POWER_ENABLE', '{"en":"PowerEnable","zh":"功率自配置算法开关"}'::jsonb, true, false, 16
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.PowerEnable'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'LTE_SNIFFER_FREQ_BAND_LIST', '{"en":"LTESnifferFreqBandList","zh":"LTE侦听频段列表"}'::jsonb, true, false, 17
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.LTESnifferFreqBandList'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'LTE_SNIFFER_CHANNEL_LIST', '{"en":"LTESnifferChannelList","zh":"LTE侦听频点列表"}'::jsonb, true, false, 18
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.LTESnifferChannelList'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'GERAN_SNIFFER_ENABLE', '{"en":"GERANSnifferEnable","zh":"GERAN侦听使能"}'::jsonb, true, false, 19
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.GERANSnifferEnable'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'GERAN_SNIFFER_CHANNEL_LIST', '{"en":"GERANSnifferChannelList","zh":"GERAN侦听频道号列表"}'::jsonb, true, false, 20
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.GERANSnifferChannelList'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'UTRAN_SNIFFER_ENABLE', '{"en":"UTRANSnifferEnable","zh":"UTRAN侦听使能"}'::jsonb, true, false, 21
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.UTRANSnifferEnable'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'UTRAN_SNIFFER_CHANNEL_LIST', '{"en":"UTRANSnifferChannelList","zh":"UTRAN侦听频道号列表"}'::jsonb, true, false, 22
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.UTRANSnifferChannelList'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MRO_ENABLE', '{"en":"MROEnable","zh":"邻区鲁棒性优化功能开关"}'::jsonb, true, false, 23
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MROEnable'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SH_ENABLE', '{"en":"SHEnable","zh":"自治愈功能开关"}'::jsonb, true, false, 24
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SHEnable'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SYNC_MODE', '{"en":"SyncMode","zh":"时钟同步模式"}'::jsonb, true, false, 25
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SyncMode'
 WHERE c.command_code = 'LST SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'AUTO_ACTIVATE_ENABLE', '{"en":"AutoActivateEnable","zh":"立即激活目标升级版本使能开关"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.SoftwareCtrl.AutoActivateEnable'
 WHERE c.command_code = 'LST SOFTWARE_CTRL'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ACTIVATE_TIME', '{"en":"ActivateTime","zh":"软件激活时间"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.SoftwareCtrl.ActivateTime'
 WHERE c.command_code = 'LST SOFTWARE_CTRL'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ACTIVATE_ENABLE', '{"en":"ActivateEnable","zh":"激活备份版本使能开关"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.SoftwareCtrl.ActivateEnable'
 WHERE c.command_code = 'LST SOFTWARE_CTRL'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SYSTEM_CURRENT_VERSION', '{"en":"SystemCurrentVersion","zh":"系统当前版本"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.SoftwareCtrl.SystemCurrentVersion'
 WHERE c.command_code = 'LST SOFTWARE_CTRL'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SYSTEM_BACKUP_VERSION', '{"en":"SystemBackupVersion","zh":"系统备份版本"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.SoftwareCtrl.SystemBackupVersion'
 WHERE c.command_code = 'LST SOFTWARE_CTRL'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ENABLE', '{"en":"Enable","zh":"NTP使能开关"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Time.Enable'
 WHERE c.command_code = 'LST TIME'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NTP_SERVER1', '{"en":"NTPServer1","zh":"NTP服务器1"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Time.NTPServer1'
 WHERE c.command_code = 'LST TIME'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NTP_SERVER2', '{"en":"NTPServer2","zh":"NTP服务器2"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Time.NTPServer2'
 WHERE c.command_code = 'LST TIME'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NTP_SERVER3', '{"en":"NTPServer3","zh":"NTP服务器3"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Time.NTPServer3'
 WHERE c.command_code = 'LST TIME'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NTP_SERVER4', '{"en":"NTPServer4","zh":"NTP服务器4"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Time.NTPServer4'
 WHERE c.command_code = 'LST TIME'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NTP_SERVER5', '{"en":"NTPServer5","zh":"NTP服务器5"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Time.NTPServer5'
 WHERE c.command_code = 'LST TIME'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CURRENT_LOCAL_TIME', '{"en":"CurrentLocalTime","zh":"本地时间"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Time.CurrentLocalTime'
 WHERE c.command_code = 'LST TIME'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'LOCAL_TIME_ZONE', '{"en":"LocalTimeZone","zh":"本地时区"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Time.LocalTimeZone'
 WHERE c.command_code = 'LST TIME'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ENABLE', '{"en":"Enable","zh":"使能开关"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.Transport.SCTP.Enable'
 WHERE c.command_code = 'LST TRANSPORT_SCTP'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'RTO_INITIAL', '{"en":"RTOInitial","zh":"重传超时的初始值"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.Transport.SCTP.RTOInitial'
 WHERE c.command_code = 'LST TRANSPORT_SCTP'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'RTO_MIN', '{"en":"RTOMin","zh":"重传超时的最小值"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.Transport.SCTP.RTOMin'
 WHERE c.command_code = 'LST TRANSPORT_SCTP'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'RTO_MAX', '{"en":"RTOMax","zh":"重传超时的最大值"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.Transport.SCTP.RTOMax'
 WHERE c.command_code = 'LST TRANSPORT_SCTP'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MAX_INIT_RETRANSMITS', '{"en":"MaxInitRetransmits","zh":"最大初始重传数"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.Transport.SCTP.MaxInitRetransmits'
 WHERE c.command_code = 'LST TRANSPORT_SCTP'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'HB_INTERVAL', '{"en":"HBInterval","zh":"心跳时间间隔"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.Transport.SCTP.HBInterval'
 WHERE c.command_code = 'LST TRANSPORT_SCTP'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MAX_PATH_RETRANSMITS', '{"en":"MaxPathRetransmits","zh":"最大重传次数"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.Transport.SCTP.MaxPathRetransmits'
 WHERE c.command_code = 'LST TRANSPORT_SCTP'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MAX_ASSOCIATION_RETRANSMITS', '{"en":"MaxAssociationRetransmits","zh":"本端的最大连续重传数目"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.Transport.SCTP.MaxAssociationRetransmits'
 WHERE c.command_code = 'LST TRANSPORT_SCTP'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'VAL_COOKIE_LIFE', '{"en":"ValCookieLife","zh":"有效cookie的生命周期"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.Transport.SCTP.ValCookieLife'
 WHERE c.command_code = 'LST TRANSPORT_SCTP'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PLMNID_LIST', '{"en":"PLMNIDList","zh":"X2PLMN标识"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.X2IpAddrMapInfo.{i}.PLMNID'
 WHERE c.command_code = 'LST X2_IP_ADDR_MAP_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ENB_TYPE', '{"en":"EnbType","zh":"基站类型"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.X2IpAddrMapInfo.{i}.EnbType'
 WHERE c.command_code = 'LST X2_IP_ADDR_MAP_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ENB_ID', '{"en":"EnbId","zh":"eNBID"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.X2IpAddrMapInfo.{i}.EnbId'
 WHERE c.command_code = 'LST X2_IP_ADDR_MAP_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'WAN_IP_ADDRESS', '{"en":"WanIpAddress","zh":"WAN口IP"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.X2IpAddrMapInfo.{i}.WanIpAddress'
 WHERE c.command_code = 'LST X2_IP_ADDR_MAP_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SUBNET_MASK', '{"en":"SubnetMask","zh":"子网掩码"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.X2IpAddrMapInfo.{i}.SubnetMask'
 WHERE c.command_code = 'LST X2_IP_ADDR_MAP_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CELL_BARRED', '{"en":"CellBarred","zh":"小区闭塞"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CellRestriction.CellBarred'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ADMIN_STATE', '{"en":"AdminState","zh":"小区管理状态"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CellEnable.AdminState'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MULTI_BAND_INFO_LIST_SIB1', '{"en":"MultiBandInfoListSIB1","zh":"SIB1中多频段指示参数"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.SysInfoCtrlParam.MultiBandInfoListSIB1'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MULTI_BAND_INFO_LIST_SIB5', '{"en":"MultiBandInfoListSIB5","zh":"SIB5中多频段指示参数"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.SysInfoCtrlParam.MultiBandInfoListSIB5'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ROUTE_INDEX_LIST', '{"en":"RouteIndexList","zh":"路由指示列表"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RouteIndexList'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'RU_LIST', '{"en":"RuList","zh":"RU列表"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RuList'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'USER_LABEL', '{"en":"UserLabel","zh":"用户友好名"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.UserLabel'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'EARFCNDL', '{"en":"EARFCNDL","zh":"下行EARFCN"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.EARFCNDL'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PHY_CELL_ID', '{"en":"PhyCellID","zh":"PCI"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PhyCellID'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'DL_BANDWIDTH', '{"en":"DLBandwidth","zh":"下行带宽"}'::jsonb, true, false, 10
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.DLBandwidth'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'UL_BANDWIDTH', '{"en":"ULBandwidth","zh":"上行带宽"}'::jsonb, true, false, 11
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ULBandwidth'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PSCH_POWER_OFFSET', '{"en":"PSCHPowerOffset","zh":"PSCH功率偏置"}'::jsonb, true, false, 12
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PSCHPowerOffset'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SSCH_POWER_OFFSET', '{"en":"SSCHPowerOffset","zh":"SSCH功率偏置"}'::jsonb, true, false, 13
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.SSCHPowerOffset'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PBCH_POWER_OFFSET', '{"en":"PBCHPowerOffset","zh":"PBCH功率偏置"}'::jsonb, true, false, 14
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PBCHPowerOffset'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'EARFCNUL', '{"en":"EARFCNUL","zh":"上行EARFCN"}'::jsonb, true, false, 15
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.EARFCNUL'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'FREQ_BAND_INDICATOR', '{"en":"FreqBandIndicator","zh":"频带指示"}'::jsonb, true, false, 16
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.FreqBandIndicator'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'REFERENCE_SIGNAL_POWER', '{"en":"ReferenceSignalPower","zh":"参考信号功率"}'::jsonb, true, false, 17
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ReferenceSignalPower'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CELL_IDENTITY', '{"en":"CellIdentity","zh":"小区标识"}'::jsonb, true, false, 18
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.CellIdentity'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ENB_TYPE', '{"en":"EnbType","zh":"基站类型"}'::jsonb, true, false, 19
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.EnbType'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SPS_SWITCH_QCI1_UL', '{"en":"SPSSwitchQCI1Ul","zh":"QCI1上行SPS功能开关"}'::jsonb, true, false, 20
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.VoLTEParam.SPSSwitchQCI1Ul'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CA_SWITCH_UL', '{"en":"CASwitchUl","zh":"上行载波聚合功能开关"}'::jsonb, true, false, 21
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.CAParam.CASwitchUl'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CA_SWITCH_DL', '{"en":"CASwitchDl","zh":"下行载波聚合功能开关"}'::jsonb, true, false, 22
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.CAParam.CASwitchDl'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T300', '{"en":"T300","zh":"T300定时器"}'::jsonb, true, false, 23
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T300'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T301', '{"en":"T301","zh":"T301定时器"}'::jsonb, true, false, 24
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T301'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T302', '{"en":"T302","zh":"T302定时器"}'::jsonb, true, false, 25
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T302'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T304EUTRA', '{"en":"T304EUTRA","zh":"T304(EUTRA)定时器"}'::jsonb, true, false, 26
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304EUTRA'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T304IRAT', '{"en":"T304IRAT","zh":"T304(异系统)定时器"}'::jsonb, true, false, 27
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T304IRAT'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T310', '{"en":"T310","zh":"T310定时器"}'::jsonb, true, false, 28
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T310'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T311', '{"en":"T311","zh":"T311定时器"}'::jsonb, true, false, 29
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T311'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T320', '{"en":"T320","zh":"T320定时器"}'::jsonb, true, false, 30
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.T320'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'N310', '{"en":"N310","zh":"T310定时器"}'::jsonb, true, false, 31
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N310'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'N311', '{"en":"N311","zh":"T311定时器"}'::jsonb, true, false, 32
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RRCTimers.N311'
 WHERE c.command_code = 'MOD '
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NNSF_SUPPORTED', '{"en":"NNSFSupported","zh":"负载均衡参数"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.Capabilities.LTE.NNSFSupported'
 WHERE c.command_code = 'MOD CAPABILITIES'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'UE_INACTIVE_TIMER', '{"en":"UeInactiveTimer","zh":"连接态UE不活动定时器"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.Capabilities.LTE.UeInactiveTimer'
 WHERE c.command_code = 'MOD CELL_CONFIG_CAPABILITIES'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SMEASURE', '{"en":"Smeasure","zh":"测量启动门限"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.EUTRA.MeasureCtrl.Smeasure'
 WHERE c.command_code = 'MOD CONN_MODE_EUTRA'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'QOFFSET_GERAN', '{"en":"QoffsetGERAN","zh":"GERAN频点偏移量"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.QoffsetGERAN'
 WHERE c.command_code = 'MOD CONN_MODE_IRAT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MEAS_QUANTITY_UTRAFDD', '{"en":"MeasQuantityUTRAFDD","zh":"UTRA测量量"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.MeasQuantityUTRAFDD'
 WHERE c.command_code = 'MOD CONN_MODE_IRAT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MEAS_QUANTITY_GERAN', '{"en":"MeasQuantityGERAN","zh":"GERAN测量量"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.MeasQuantityGERAN'
 WHERE c.command_code = 'MOD CONN_MODE_IRAT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'QOFFSET_UTRA', '{"en":"QoffsetUTRA","zh":"UTRA频点偏移量"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.QoffsetUTRA'
 WHERE c.command_code = 'MOD CONN_MODE_IRAT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'USER_LABEL', '{"en":"UserLabel","zh":"用户友好名"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.UserLabel'
 WHERE c.command_code = 'MOD DEVICE_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'DN_PREFIX', '{"en":"DnPrefix","zh":"DN前缀"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.DnPrefix'
 WHERE c.command_code = 'MOD DEVICE_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'USER_LABEL', '{"en":"UserLabel","zh":"用户友好名"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.UserLabel'
 WHERE c.command_code = 'MOD DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'DN_PREFIX', '{"en":"DnPrefix","zh":"DN前缀"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.DnPrefix'
 WHERE c.command_code = 'MOD DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'REBOOT', '{"en":"Reboot","zh":"重启开关"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.DeviceInfo.MU.{i}.Reboot'
 WHERE c.command_code = 'MOD DEVICE_INFO_MU'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ENABLE', '{"en":"Enable","zh":"启用或禁用该接口"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Ethernet.Interface.{i}.Enable'
 WHERE c.command_code = 'MOD ETHERNET_INTERFACE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'USER_LABEL', '{"en":"UserLabel","zh":"用户友好名"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Ethernet.Interface.{i}.UserLabel'
 WHERE c.command_code = 'MOD ETHERNET_INTERFACE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MAX_BIT_RATE', '{"en":"MaxBitRate","zh":"该连接可用的最大速率模式"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Ethernet.Interface.{i}.MaxBitRate'
 WHERE c.command_code = 'MOD ETHERNET_INTERFACE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'DUPLEX_MODE', '{"en":"DuplexMode","zh":"该连接使用的双工模式"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Ethernet.Interface.{i}.DuplexMode'
 WHERE c.command_code = 'MOD ETHERNET_INTERFACE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'IP_VER', '{"en":"IpVer","zh":"IP地址版本"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Ethernet.IpRoute.{i}.IpVer'
 WHERE c.command_code = 'MOD ETHERNET_IP_ROUTE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'DST_IP_NETWORK', '{"en":"DstIpNetwork","zh":"目的网段"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Ethernet.IpRoute.{i}.DstIpNetwork'
 WHERE c.command_code = 'MOD ETHERNET_IP_ROUTE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PREFIX_LENGTH', '{"en":"PrefixLength","zh":"前缀长度"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Ethernet.IpRoute.{i}.PrefixLength'
 WHERE c.command_code = 'MOD ETHERNET_IP_ROUTE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'GATEWAY_IP_ADDRESS', '{"en":"GatewayIpAddress","zh":"网关地址"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Ethernet.IpRoute.{i}.GatewayIpAddress'
 WHERE c.command_code = 'MOD ETHERNET_IP_ROUTE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'INTERFACE_NAME', '{"en":"InterfaceName","zh":"端口名称"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Ethernet.IpRoute.{i}.InterfaceName'
 WHERE c.command_code = 'MOD ETHERNET_IP_ROUTE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'REPORTING_MECHANISM', '{"en":"ReportingMechanism","zh":"告警上报机制"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FaultMgmt.SupportedAlarm.{i}.ReportingMechanism'
 WHERE c.command_code = 'MOD FAULT_MGMT_SUPPORTED_ALARM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T_RESELECTION_UTRA', '{"en":"TReselectionUTRA","zh":"UTRAN小区重选时间"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.TReselectionUTRA'
 WHERE c.command_code = 'MOD IDLE_MODE_IRAT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T_RESELECTION_GERAN', '{"en":"TReselectionGERAN","zh":"GERAN小区重选时间"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.TReselectionGERAN'
 WHERE c.command_code = 'MOD IDLE_MODE_IRAT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ENABLE', '{"en":"Enable","zh":"开关"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.IPsec.Enable'
 WHERE c.command_code = 'MOD I_PSEC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MY_KEY_MODE', '{"en":"MyKeyMode","zh":"鉴权方式"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.IPsec.MyKeyMode'
 WHERE c.command_code = 'MOD I_PSEC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PERIODIC_UPLOAD_ENABLE', '{"en":"PeriodicUploadEnable","zh":"日志周期上传使能开关"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.LogMgmt.PeriodicUploadEnable'
 WHERE c.command_code = 'MOD LOG_MGMT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'URL', '{"en":"URL","zh":"日志上传URL"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.LogMgmt.URL'
 WHERE c.command_code = 'MOD LOG_MGMT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'USERNAME', '{"en":"Username","zh":"日志管理用户名"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.LogMgmt.Username'
 WHERE c.command_code = 'MOD LOG_MGMT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PASSWORD', '{"en":"Password","zh":"日志管理密码"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.LogMgmt.Password'
 WHERE c.command_code = 'MOD LOG_MGMT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PERIODIC_UPLOAD_INTERVAL', '{"en":"PeriodicUploadInterval","zh":"日志周期上传时间间隔"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.LogMgmt.PeriodicUploadInterval'
 WHERE c.command_code = 'MOD LOG_MGMT'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ADMIN_STATE', '{"en":"AdminState","zh":"基站管理状态"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.AdminState'
 WHERE c.command_code = 'MOD LTE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'EAID', '{"en":"EAID","zh":"EPCEAID"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.EPC.EAID'
 WHERE c.command_code = 'MOD LTE_EPC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'TAC', '{"en":"TAC","zh":"TAC"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.EPC.TAC'
 WHERE c.command_code = 'MOD LTE_EPC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SEC_GW_SERVER1', '{"en":"SecGWServer1","zh":"安全网关1"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.Gateway.SecGWServer1'
 WHERE c.command_code = 'MOD LTE_GATEWAY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SEC_GW_SERVER2', '{"en":"SecGWServer2","zh":"安全网关2"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.Gateway.SecGWServer2'
 WHERE c.command_code = 'MOD LTE_GATEWAY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SEC_GW_SERVER3', '{"en":"SecGWServer3","zh":"安全网关3"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.Gateway.SecGWServer3'
 WHERE c.command_code = 'MOD LTE_GATEWAY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'AG_SERVER_ENABLE', '{"en":"AGServerEnable","zh":"HeGW方式接入"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.Gateway.AGServerEnable'
 WHERE c.command_code = 'MOD LTE_GATEWAY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'AG_SERVER_IP1', '{"en":"AGServerIp1","zh":"接入网关1"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.Gateway.AGServerIp1'
 WHERE c.command_code = 'MOD LTE_GATEWAY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'AG_SERVER_IP2', '{"en":"AGServerIp2","zh":"接入网关2"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.Gateway.AGServerIp2'
 WHERE c.command_code = 'MOD LTE_GATEWAY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'AG_SERVER_IP3', '{"en":"AGServerIp3","zh":"接入网关3"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.Gateway.AGServerIp3'
 WHERE c.command_code = 'MOD LTE_GATEWAY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'AG_PORT1', '{"en":"AGPort1","zh":"接入网关端口1"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.Gateway.AGPort1'
 WHERE c.command_code = 'MOD LTE_GATEWAY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'AG_PORT2', '{"en":"AGPort2","zh":"接入网关端口2"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.Gateway.AGPort2'
 WHERE c.command_code = 'MOD LTE_GATEWAY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'AG_PORT3', '{"en":"AGPort3","zh":"接入网关端口3"}'::jsonb, true, false, 10
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.Gateway.AGPort3'
 WHERE c.command_code = 'MOD LTE_GATEWAY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MME_IP1', '{"en":"MMEIp1","zh":"MMEIP1"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.MMEIp1'
 WHERE c.command_code = 'MOD LTE_MME_POOL_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MME_IP2', '{"en":"MMEIp2","zh":"MMEIP2"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.LTE.MmePoolConfigParam.{i}.MMEIp2'
 WHERE c.command_code = 'MOD LTE_MME_POOL_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'URL', '{"en":"URL","zh":"网管服务器URL"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.URL'
 WHERE c.command_code = 'MOD MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'USERNAME', '{"en":"Username","zh":"连接用户名"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.Username'
 WHERE c.command_code = 'MOD MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PASSWORD', '{"en":"Password","zh":"连接密码"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.Password'
 WHERE c.command_code = 'MOD MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PERIODIC_INFORM_ENABLE', '{"en":"PeriodicInformEnable","zh":"连接使能开关"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.PeriodicInformEnable'
 WHERE c.command_code = 'MOD MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PERIODIC_INFORM_TIME', '{"en":"PeriodicInformTime","zh":"上报周期"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.PeriodicInformTime'
 WHERE c.command_code = 'MOD MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PERIODIC_INFORM_INTERVAL', '{"en":"PeriodicInformInterval","zh":"周期上报时间间隔"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.PeriodicInformInterval'
 WHERE c.command_code = 'MOD MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CONNECTION_REQUEST_USERNAME', '{"en":"ConnectionRequestUsername","zh":"连接请求用户名"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.ConnectionRequestUsername'
 WHERE c.command_code = 'MOD MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CONNECTION_REQUEST_PASSWORD', '{"en":"ConnectionRequestPassword","zh":"连接请求密码"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.ConnectionRequestPassword'
 WHERE c.command_code = 'MOD MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'STUN_ENABLE', '{"en":"STUNEnable","zh":"STUN使能开关"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.STUNEnable'
 WHERE c.command_code = 'MOD MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'STUN_SERVER_ADDRESS', '{"en":"STUNServerAddress","zh":"STUN服务器地址"}'::jsonb, true, false, 10
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.STUNServerAddress'
 WHERE c.command_code = 'MOD MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'STUN_SERVER_PORT', '{"en":"STUNServerPort","zh":"STUN服务器端口"}'::jsonb, true, false, 11
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.STUNServerPort'
 WHERE c.command_code = 'MOD MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'STUN_USERNAME', '{"en":"STUNUsername","zh":"STUN用户名"}'::jsonb, true, false, 12
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.STUNUsername'
 WHERE c.command_code = 'MOD MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'STUN_PASSWORD', '{"en":"STUNPassword","zh":"STUN密码"}'::jsonb, true, false, 13
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.STUNPassword'
 WHERE c.command_code = 'MOD MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'STUN_MAXIMUM_KEEP_ALIVE_PERIOD', '{"en":"STUNMaximumKeepAlivePeriod","zh":"STUN最大生存周期"}'::jsonb, true, false, 14
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.STUNMaximumKeepAlivePeriod'
 WHERE c.command_code = 'MOD MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'STUN_MINIMUM_KEEP_ALIVE_PERIOD', '{"en":"STUNMinimumKeepAlivePeriod","zh":"STUN最小生存周期"}'::jsonb, true, false, 15
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.ManagementServer.STUNMinimumKeepAlivePeriod'
 WHERE c.command_code = 'MOD MANAGEMENT_SERVER'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'QHYST', '{"en":"Qhyst","zh":"服务小区重选迟滞值"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.Qhyst'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'INTRA_FREQ_RESELECTION', '{"en":"IntraFreqReselection","zh":"同频重选指示"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.IntraFreqReselection'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'Q_HYST_SF_MEDIUM', '{"en":"QHystSFMedium","zh":"Qhyst比例因子(中速)"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.QHystSFMedium'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'Q_HYST_SF_HIGH', '{"en":"QHystSFHigh","zh":"Qhyst比例因子(高速)"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.QHystSFHigh'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T_EVALUATION', '{"en":"TEvaluation","zh":"允许小区重选数目的间隔时间"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.TEvaluation'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T_HYST_NORMAL', '{"en":"THystNormal","zh":"正常状态附加判决时长"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.THystNormal'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'N_CELL_CHANGE_MEDIUM', '{"en":"NCellChangeMedium","zh":"进入中速重选次数门限"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.NCellChangeMedium'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'N_CELL_CHANGE_HIGH', '{"en":"NCellChangeHigh","zh":"进入高速重选次数门限"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.NCellChangeHigh'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'Q_RX_LEV_MIN_SIB1', '{"en":"QRxLevMinSIB1","zh":"服务小区最小接收电平"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QRxLevMinSIB1'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'Q_RX_LEV_MIN_SIB3', '{"en":"QRxLevMinSIB3","zh":"同频邻区最小接收电平"}'::jsonb, true, false, 10
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QRxLevMinSIB3'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'Q_RX_LEV_MIN_OFFSET', '{"en":"QRxLevMinOffset","zh":"最小接收电平偏移量"}'::jsonb, true, false, 11
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QRxLevMinOffset'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'S_INTRA_SEARCH', '{"en":"SIntraSearch","zh":"同频测量启动门限"}'::jsonb, true, false, 12
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SIntraSearch'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T_RESELECTION_EUTRA', '{"en":"TReselectionEUTRA","zh":"同频小区重选时间"}'::jsonb, true, false, 13
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRA'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'S_NON_INTRA_SEARCH', '{"en":"SNonIntraSearch","zh":"非同频测量启动门限"}'::jsonb, true, false, 14
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SNonIntraSearch'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'S_NON_INTRA_SEARCH_PR9', '{"en":"SNonIntraSearchPR9","zh":"异频/异系统RSRP测量启动门限"}'::jsonb, true, false, 15
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SNonIntraSearchPR9'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'S_NON_INTRA_SEARCH_QR9', '{"en":"SNonIntraSearchQR9","zh":"异频/异系统RSRQ测量启动门限"}'::jsonb, true, false, 16
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SNonIntraSearchQR9'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CELL_RESELECTION_PRIORITY', '{"en":"CellReselectionPriority","zh":"同频小区重选优先级"}'::jsonb, true, false, 17
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.CellReselectionPriority'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'P_MAX', '{"en":"PMax","zh":"UE允许最大发射功率"}'::jsonb, true, false, 18
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.PMax'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'THRESH_SERVING_LOW', '{"en":"ThreshServingLow","zh":"低优先级重选门限"}'::jsonb, true, false, 19
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.ThreshServingLow'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'THRESH_SERVING_LOW_QR9', '{"en":"ThreshServingLowQR9","zh":"服务频点低优先级RSRQ重选门限"}'::jsonb, true, false, 20
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.ThreshServingLowQR9'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T_RESELECTION_EUTRASF_MEDIUM', '{"en":"TReselectionEUTRASFMedium","zh":"TReselectionUTRA比例因子(中速)"}'::jsonb, true, false, 21
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRASFMedium'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'T_RESELECTION_EUTRASF_HIGH', '{"en":"TReselectionEUTRASFHigh","zh":"TReselectionUTRA比例因子(高速)"}'::jsonb, true, false, 22
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRASFHigh'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'S_INTRA_SEARCH_PR9', '{"en":"SIntraSearchPR9","zh":"同频RSRP测量启动门限"}'::jsonb, true, false, 23
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SIntraSearchPR9'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'S_INTRA_SEARCH_QR9', '{"en":"SIntraSearchQR9","zh":"同频RSRQ测量启动门限"}'::jsonb, true, false, 24
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.SIntraSearchQR9'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'Q_QUAL_MIN_R9_RESELECTION', '{"en":"QQualMinR9Reselection","zh":"小区重选最低接入信号质量"}'::jsonb, true, false, 25
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QQualMinR9Reselection'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'Q_QUAL_MIN_R9_SELECTION', '{"en":"QQualMinR9Selection","zh":"小区选择最低接入信号质量"}'::jsonb, true, false, 26
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QQualMinR9Selection'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'Q_QUAL_MIN_OFFSET_R9', '{"en":"QQualMinOffsetR9","zh":"小区选择最低接入信号质量偏置"}'::jsonb, true, false, 27
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.QQualMinOffsetR9'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ALLOWED_MEAS_BANDWIDTH', '{"en":"AllowedMeasBandwidth","zh":"测量带宽"}'::jsonb, true, false, 28
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.AllowedMeasBandwidth'
 WHERE c.command_code = 'MOD MOBILITY_IDLE_MODE'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MR_ENABLE', '{"en":"MrEnable","zh":"MR开关"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.MrEnable'
 WHERE c.command_code = 'MOD MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MR_URL', '{"en":"MrUrl","zh":"Url地址"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.MrUrl'
 WHERE c.command_code = 'MOD MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MR_USERNAME', '{"en":"MrUsername","zh":"用户名"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.MrUsername'
 WHERE c.command_code = 'MOD MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MR_PASSWORD', '{"en":"MrPassword","zh":"密码"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.MrPassword'
 WHERE c.command_code = 'MOD MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MEASURE_TYPE', '{"en":"MeasureType","zh":"MR文件类型"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.MeasureType'
 WHERE c.command_code = 'MOD MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'OMC_NAME', '{"en":"OmcName","zh":"OMC-R名称"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.OmcName'
 WHERE c.command_code = 'MOD MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SAMPLE_PERIOD', '{"en":"SamplePeriod","zh":"MR采样周期"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.SamplePeriod'
 WHERE c.command_code = 'MOD MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'UPLOAD_PERIOD', '{"en":"UploadPeriod","zh":"MR采集周期"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.UploadPeriod'
 WHERE c.command_code = 'MOD MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SAMPLE_BEGIN_TIME', '{"en":"SampleBeginTime","zh":"绝对时间参考"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.SampleBeginTime'
 WHERE c.command_code = 'MOD MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SAMPLE_END_TIME', '{"en":"SampleEndTime","zh":"绝对时间参考"}'::jsonb, true, false, 10
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.SampleEndTime'
 WHERE c.command_code = 'MOD MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PRB_NUM', '{"en":"PrbNum","zh":"子帧的PRB"}'::jsonb, true, false, 11
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.PrbNum'
 WHERE c.command_code = 'MOD MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SUB_FRAME_NUM', '{"en":"SubFrameNum","zh":"子帧数"}'::jsonb, true, false, 12
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.SubFrameNum'
 WHERE c.command_code = 'MOD MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MRECGI_LIST', '{"en":"MRECGIList","zh":"MR小区列表"}'::jsonb, true, false, 13
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.MRECGIList'
 WHERE c.command_code = 'MOD MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MEASURE_ITEMS', '{"en":"MeasureItems","zh":"测量项"}'::jsonb, true, false, 14
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.MRMgmt.Config.{i}.MeasureItems'
 WHERE c.command_code = 'MOD MR_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ENABLE', '{"en":"Enable","zh":"文件周期上传使能开关"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.PerfMgmt.Config.{i}.Enable'
 WHERE c.command_code = 'MOD PERF_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ALIAS', '{"en":"Alias","zh":"别名"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.PerfMgmt.Config.{i}.Alias'
 WHERE c.command_code = 'MOD PERF_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'URL', '{"en":"URL","zh":"文件管理URL"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.PerfMgmt.Config.{i}.URL'
 WHERE c.command_code = 'MOD PERF_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'USERNAME', '{"en":"Username","zh":"文件管理用户名"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.PerfMgmt.Config.{i}.Username'
 WHERE c.command_code = 'MOD PERF_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PASSWORD', '{"en":"Password","zh":"文件管理密码"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.PerfMgmt.Config.{i}.Password'
 WHERE c.command_code = 'MOD PERF_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PERIODIC_UPLOAD_INTERVAL', '{"en":"PeriodicUploadInterval","zh":"文件周期上传时间间隔"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.PerfMgmt.Config.{i}.PeriodicUploadInterval'
 WHERE c.command_code = 'MOD PERF_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PERIODIC_UPLOAD_TIME', '{"en":"PeriodicUploadTime","zh":"文件上传时间"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.PerfMgmt.Config.{i}.PeriodicUploadTime'
 WHERE c.command_code = 'MOD PERF_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'REPLENISH_ENABLE', '{"en":"ReplenishEnable","zh":"补采开关"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.PerfMgmt.Config.{i}.ReplenishEnable'
 WHERE c.command_code = 'MOD PERF_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'REPLENISH_START_TIME', '{"en":"ReplenishStartTime","zh":"补采开始时间"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.PerfMgmt.Config.{i}.ReplenishStartTime'
 WHERE c.command_code = 'MOD PERF_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'REPLENISH_END_TIME', '{"en":"ReplenishEndTime","zh":"补采结束时间"}'::jsonb, true, false, 10
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.FAP.PerfMgmt.Config.{i}.ReplenishEndTime'
 WHERE c.command_code = 'MOD PERF_MGMT_CONFIG'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NEIGH_CELL_CONFIG', '{"en":"NeighCellConfig","zh":"邻区配置"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.MBSFN.NeighCellConfig'
 WHERE c.command_code = 'MOD PHY_MBSFN'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NUMBER_OF_RA_PREAMBLES', '{"en":"NumberOfRaPreambles","zh":"竞争随机接入前导码数"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.NumberOfRaPreambles'
 WHERE c.command_code = 'MOD RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SIZE_OF_RA_GROUP_A', '{"en":"SizeOfRaGroupA","zh":"随机接入前导码组A大小"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.SizeOfRaGroupA'
 WHERE c.command_code = 'MOD RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MESSAGE_SIZE_GROUP_A', '{"en":"MessageSizeGroupA","zh":"组A信息大小"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MessageSizeGroupA'
 WHERE c.command_code = 'MOD RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MESSAGE_POWER_OFFSET_GROUP_B', '{"en":"MessagePowerOffsetGroupB","zh":"组B功率补偿信息"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MessagePowerOffsetGroupB'
 WHERE c.command_code = 'MOD RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'POWER_RAMPING_STEP', '{"en":"PowerRampingStep","zh":"功率增加补偿"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.PowerRampingStep'
 WHERE c.command_code = 'MOD RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PREAMBLE_INITIAL_RECEIVED_TARGET_POWER', '{"en":"PreambleInitialReceivedTargetPower","zh":"前导码初始接收目标功率"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.PreambleInitialReceivedTargetPower'
 WHERE c.command_code = 'MOD RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PREAMBLE_TRANS_MAX', '{"en":"PreambleTransMax","zh":"最大随机接入次数"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.PreambleTransMax'
 WHERE c.command_code = 'MOD RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'RESPONSE_WINDOW_SIZE', '{"en":"ResponseWindowSize","zh":"随机接入响应窗口大小"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.ResponseWindowSize'
 WHERE c.command_code = 'MOD RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CONTENTION_RESLUTION_TIMER', '{"en":"ContentionReslutionTimer","zh":"冲突解决定时器"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.ContentionResolutionTimer'
 WHERE c.command_code = 'MOD RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MAX_HARQ_MSG3_TX', '{"en":"MaxHARQMsg3Tx","zh":"Msg3HARQ最大传输次数"}'::jsonb, true, false, 10
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MaxHARQMsg3Tx'
 WHERE c.command_code = 'MOD RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'DRX_ENABLED', '{"en":"DRXEnabled","zh":"DRX功能开关"}'::jsonb, true, false, 11
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.DRX.DRXEnabled'
 WHERE c.command_code = 'MOD RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MAX_HARQ_TX', '{"en":"MaxHARQTx","zh":"上行HARQ最大传输次数"}'::jsonb, true, false, 12
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.MaxHARQTx'
 WHERE c.command_code = 'MOD RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PERIODIC_BSR_TIMER', '{"en":"PeriodicBSRTimer","zh":"周期性BSR定时器"}'::jsonb, true, false, 13
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.PeriodicBSRTimer'
 WHERE c.command_code = 'MOD RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'RETX_BSR_TIMER', '{"en":"RetxBSRTimer","zh":"重传BSR定时器"}'::jsonb, true, false, 14
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.RetxBSRTimer'
 WHERE c.command_code = 'MOD RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'TTI_BUNDLING', '{"en":"TTIBundling","zh":"TTI绑定功能开关"}'::jsonb, true, false, 15
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.TTIBundling'
 WHERE c.command_code = 'MOD RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MAX_UE_PER_UL_SF', '{"en":"MaxUePerUlSf","zh":"上行单帧最大调度用户数"}'::jsonb, true, false, 16
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.MaxUePerUlSf'
 WHERE c.command_code = 'MOD RAN_MAC'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CONFIGURATION_INDEX', '{"en":"ConfigurationIndex","zh":"PRACH配置索引"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.ConfigurationIndex'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'FREQ_OFFSET', '{"en":"FreqOffset","zh":"PRACH频率偏移"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.FreqOffset'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'HIGH_SPEED_FLAG', '{"en":"HighSpeedFlag","zh":"高速状态标识"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.HighSpeedFlag'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ROOT_SEQUENCE_INDEX', '{"en":"RootSequenceIndex","zh":"逻辑根序列索引"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.RootSequenceIndex'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ZERO_CORRELATION_ZONE_CONFIG', '{"en":"ZeroCorrelationZoneConfig","zh":"零相关配置"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.ZeroCorrelationZoneConfig'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SRS_ENABLED', '{"en":"SRSEnabled","zh":"SRS使能开关"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSEnabled'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SRS_BANDWIDTH_CONFIG', '{"en":"SRSBandwidthConfig","zh":"SRS带宽配置"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSBandwidthConfig'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SRS_MAX_UP_PTS', '{"en":"SRSMaxUpPTS","zh":"SRSUpPTS带宽重配指示"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSMaxUpPTS'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ACK_NACK_SRS_SIMULTANEOUS_TRANSMISSION', '{"en":"AckNackSRSSimultaneousTransmission","zh":"SRS/ACK/NACK同时传输标识"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.AckNackSRSSimultaneousTransmission'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'DELTA_PUCCH_SHIFT', '{"en":"DeltaPUCCHShift","zh":"PUCCH循环移位间隔"}'::jsonb, true, false, 10
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.DeltaPUCCHShift'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NRBCQI', '{"en":"NRBCQI","zh":"PUCCH2/2a/2b占用RB数"}'::jsonb, true, false, 11
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.NRBCQI'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NCSAN', '{"en":"NCSAN","zh":"PUCCH1/1a/1b预留资源数"}'::jsonb, true, false, 12
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.NCSAN'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'N1PUCCHAN', '{"en":"N1PUCCHAN","zh":"SPSACK/NACK及SR预留资源数"}'::jsonb, true, false, 13
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.N1PUCCHAN'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CQIPUCCH_RESOURCE_INDEX', '{"en":"CQIPUCCHResourceIndex","zh":"CQI报告使用PUCCH资源索引"}'::jsonb, true, false, 14
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.CQIPUCCHResourceIndex'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'K', '{"en":"K","zh":"子带CQI报告带宽"}'::jsonb, true, false, 15
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.K'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PUSCH_POWER_CTRL_SWITCH', '{"en":"PUSCHPowerCtrlSwitch","zh":"上行PUSCH闭环功控开关"}'::jsonb, true, false, 16
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PaParam.PUSCHPowerCtrlSwitch'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PUCCH_POWER_CTRL_SWITCH', '{"en":"PUCCHPowerCtrlSwitch","zh":"上行PUCCH闭环功控开关"}'::jsonb, true, false, 17
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PaParam.PUCCHPowerCtrlSwitch'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ENABLE64QAM', '{"en":"Enable64QAM","zh":"PUSCH64QAM使能开关"}'::jsonb, true, false, 18
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.Enable64QAM'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'HOPPING_MODE', '{"en":"HoppingMode","zh":"PUSCH跳频模式"}'::jsonb, true, false, 19
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.HoppingMode'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'HOPPING_OFFSET', '{"en":"HoppingOffset","zh":"PUSCH跳频偏置"}'::jsonb, true, false, 20
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.HoppingOffset'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NSB', '{"en":"NSB","zh":"跳频子带个数"}'::jsonb, true, false, 21
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.NSB'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NUM_PRS_RESOURCE_BLOCKS', '{"en":"NumPRSResourceBlocks","zh":"PRSRB数目"}'::jsonb, true, false, 22
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRS.NumPRSResourceBlocks'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PRS_CONFIGURATION_INDEX', '{"en":"PRSConfigurationIndex","zh":"PRS配置索引"}'::jsonb, true, false, 23
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRS.PRSConfigurationIndex'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NUM_CONSECUTIVE_PRS_SUBFAMES', '{"en":"NumConsecutivePRSSubfames","zh":"PRS连续子帧数"}'::jsonb, true, false, 24
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRS.NumConsecutivePRSSubfames'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SPECIAL_SUBFRAME_PATTERNS', '{"en":"SpecialSubframePatterns","zh":"特殊子帧配置"}'::jsonb, true, false, 25
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.TDDFrame.SpecialSubframePatterns'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SUB_FRAME_ASSIGNMENT', '{"en":"SubFrameAssignment","zh":"上下行子帧配置"}'::jsonb, true, false, 26
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.TDDFrame.SubFrameAssignment'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PB', '{"en":"Pb","zh":"天线端口信号功率比"}'::jsonb, true, false, 27
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PDSCH.Pb'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PA', '{"en":"Pa","zh":"小区PDSCH采用固定功率分配时的PA取值"}'::jsonb, true, false, 28
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PDSCH.Pa'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'P0_NOMINAL_PUSCH_PERSISTENT', '{"en":"P0NominalPUSCHPersistent","zh":"持续调度期望接收功率"}'::jsonb, true, false, 29
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.P0NominalPUSCHPersistent'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'P0_NOMINAL_PUSCH', '{"en":"P0NominalPUSCH","zh":"非持续调度期望功率"}'::jsonb, true, false, 30
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.P0NominalPUSCH'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ALPHA', '{"en":"Alpha","zh":"部分路损补偿系数"}'::jsonb, true, false, 31
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.Alpha'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'P0_NOMINAL_PUCCH', '{"en":"P0NominalPUCCH","zh":"PUCCH期望功率"}'::jsonb, true, false, 32
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.P0NominalPUCCH'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'DELTA_MCS_ENABLED', '{"en":"DeltaMCSEnabled","zh":"MCS补偿值"}'::jsonb, true, false, 33
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.ULPowerControl.DeltaMCSEnabled'
 WHERE c.command_code = 'MOD RAN_PHY'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SON_SYS_MODE', '{"en":"SONSysMode","zh":"SON系统模式"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SONSysMode'
 WHERE c.command_code = 'MOD SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SON_WORK_MODE', '{"en":"SONWorkMode","zh":"SON模式设置"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SONWorkMode'
 WHERE c.command_code = 'MOD SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PCI_OPT_ENABLE', '{"en":"PCIOptEnable","zh":"PCI自优化算法开关"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.PCIOptEnable'
 WHERE c.command_code = 'MOD SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PCI_RECONFIG_WAIT_TIME', '{"en":"PCIReconfigWaitTime","zh":"PCI重配等待定时器"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.PCIReconfigWaitTime'
 WHERE c.command_code = 'MOD SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CANDIDATE_ARFCN_LIST', '{"en":"CandidateARFCNList","zh":"候选频点列表"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.CandidateARFCNList'
 WHERE c.command_code = 'MOD SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'CANDIDATE_PCI_LIST', '{"en":"CandidatePCIList","zh":"候选PCI列表"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.CandidatePCIList'
 WHERE c.command_code = 'MOD SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ANR_ENABLE', '{"en":"ANREnable","zh":"ANR算法总开关"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANREnable'
 WHERE c.command_code = 'MOD SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ANR_INTER_FEQ_ENABLE', '{"en":"ANRInterFeqEnable","zh":"E-UTRAN异频ANR算法开关"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANRInterFeqEnable'
 WHERE c.command_code = 'MOD SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ANRGERAN_ENABLE', '{"en":"ANRGERANEnable","zh":"GERAN异系统ANR算法开关"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANRGERANEnable'
 WHERE c.command_code = 'MOD SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ANRUTRAN_ENABLE', '{"en":"ANRUTRANEnable","zh":"UTRAN异系统ANR算法开关"}'::jsonb, true, false, 10
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ANRUTRANEnable'
 WHERE c.command_code = 'MOD SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ARFCN_ENABLE', '{"en":"ARFCNEnable","zh":"频点自配置算法开关"}'::jsonb, true, false, 11
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ARFCNEnable'
 WHERE c.command_code = 'MOD SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MAX_LTE_NEIGHBOUR_CELL_NUM', '{"en":"MaxLTENeighbourCellNum","zh":"最大LTE邻区数"}'::jsonb, true, false, 12
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MaxLTENeighbourCellNum'
 WHERE c.command_code = 'MOD SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MAX_UTRAN_NEIGHBOUR_CELL_NUM', '{"en":"MaxUTRANNeighbourCellNum","zh":"最大UTRAN邻区数"}'::jsonb, true, false, 13
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MaxUTRANNeighbourCellNum'
 WHERE c.command_code = 'MOD SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MAX_GERAN_NEIGHBOUR_CELL_NUM', '{"en":"MaxGERANNeighbourCellNum","zh":"最大GERAN邻区数"}'::jsonb, true, false, 14
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MaxGRANNeighbourCellNum'
 WHERE c.command_code = 'MOD SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'RE_SYN_CELL_ENABLE', '{"en":"ReSynCellEnable","zh":"重新同步小区使能开关"}'::jsonb, true, false, 15
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.ReSynCellEnable'
 WHERE c.command_code = 'MOD SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'POWER_ENABLE', '{"en":"PowerEnable","zh":"功率自配置算法开关"}'::jsonb, true, false, 16
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.PowerEnable'
 WHERE c.command_code = 'MOD SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'LTE_SNIFFER_FREQ_BAND_LIST', '{"en":"LTESnifferFreqBandList","zh":"LTE侦听频段列表"}'::jsonb, true, false, 17
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.LTESnifferFreqBandList'
 WHERE c.command_code = 'MOD SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'LTE_SNIFFER_CHANNEL_LIST', '{"en":"LTESnifferChannelList","zh":"LTE侦听频点列表"}'::jsonb, true, false, 18
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.LTESnifferChannelList'
 WHERE c.command_code = 'MOD SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'GERAN_SNIFFER_ENABLE', '{"en":"GERANSnifferEnable","zh":"GERAN侦听使能"}'::jsonb, true, false, 19
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.GERANSnifferEnable'
 WHERE c.command_code = 'MOD SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'GERAN_SNIFFER_CHANNEL_LIST', '{"en":"GERANSnifferChannelList","zh":"GERAN侦听频道号列表"}'::jsonb, true, false, 20
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.GERANSnifferChannelList'
 WHERE c.command_code = 'MOD SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'UTRAN_SNIFFER_ENABLE', '{"en":"UTRANSnifferEnable","zh":"UTRAN侦听使能"}'::jsonb, true, false, 21
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.UTRANSnifferEnable'
 WHERE c.command_code = 'MOD SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'UTRAN_SNIFFER_CHANNEL_LIST', '{"en":"UTRANSnifferChannelList","zh":"UTRAN侦听频道号列表"}'::jsonb, true, false, 22
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.UTRANSnifferChannelList'
 WHERE c.command_code = 'MOD SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MRO_ENABLE', '{"en":"MROEnable","zh":"邻区鲁棒性优化功能开关"}'::jsonb, true, false, 23
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.MROEnable'
 WHERE c.command_code = 'MOD SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SH_ENABLE', '{"en":"SHEnable","zh":"自治愈功能开关"}'::jsonb, true, false, 24
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPService.{i}.FAPControl.LTE.SelfConfig.SONConfigParam.SHEnable'
 WHERE c.command_code = 'MOD SELF_CONFIG_SON_CONFIG_PARAM'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'AUTO_ACTIVATE_ENABLE', '{"en":"AutoActivateEnable","zh":"立即激活目标升级版本使能开关"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.SoftwareCtrl.AutoActivateEnable'
 WHERE c.command_code = 'MOD SOFTWARE_CTRL'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ACTIVATE_TIME', '{"en":"ActivateTime","zh":"软件激活时间"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.SoftwareCtrl.ActivateTime'
 WHERE c.command_code = 'MOD SOFTWARE_CTRL'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ACTIVATE_ENABLE', '{"en":"ActivateEnable","zh":"激活备份版本使能开关"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.SoftwareCtrl.ActivateEnable'
 WHERE c.command_code = 'MOD SOFTWARE_CTRL'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ENABLE', '{"en":"Enable","zh":"NTP使能开关"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Time.Enable'
 WHERE c.command_code = 'MOD TIME'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NTP_SERVER1', '{"en":"NTPServer1","zh":"NTP服务器1"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Time.NTPServer1'
 WHERE c.command_code = 'MOD TIME'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NTP_SERVER2', '{"en":"NTPServer2","zh":"NTP服务器2"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Time.NTPServer2'
 WHERE c.command_code = 'MOD TIME'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NTP_SERVER3', '{"en":"NTPServer3","zh":"NTP服务器3"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Time.NTPServer3'
 WHERE c.command_code = 'MOD TIME'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NTP_SERVER4', '{"en":"NTPServer4","zh":"NTP服务器4"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Time.NTPServer4'
 WHERE c.command_code = 'MOD TIME'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'NTP_SERVER5', '{"en":"NTPServer5","zh":"NTP服务器5"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Time.NTPServer5'
 WHERE c.command_code = 'MOD TIME'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'LOCAL_TIME_ZONE', '{"en":"LocalTimeZone","zh":"本地时区"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Time.LocalTimeZone'
 WHERE c.command_code = 'MOD TIME'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ENABLE', '{"en":"Enable","zh":"使能开关"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.Transport.SCTP.Enable'
 WHERE c.command_code = 'MOD TRANSPORT_SCTP'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'RTO_INITIAL', '{"en":"RTOInitial","zh":"重传超时的初始值"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.Transport.SCTP.RTOInitial'
 WHERE c.command_code = 'MOD TRANSPORT_SCTP'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'RTO_MIN', '{"en":"RTOMin","zh":"重传超时的最小值"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.Transport.SCTP.RTOMin'
 WHERE c.command_code = 'MOD TRANSPORT_SCTP'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'RTO_MAX', '{"en":"RTOMax","zh":"重传超时的最大值"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.Transport.SCTP.RTOMax'
 WHERE c.command_code = 'MOD TRANSPORT_SCTP'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MAX_INIT_RETRANSMITS', '{"en":"MaxInitRetransmits","zh":"最大初始重传数"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.Transport.SCTP.MaxInitRetransmits'
 WHERE c.command_code = 'MOD TRANSPORT_SCTP'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'HB_INTERVAL', '{"en":"HBInterval","zh":"心跳时间间隔"}'::jsonb, true, false, 6
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.Transport.SCTP.HBInterval'
 WHERE c.command_code = 'MOD TRANSPORT_SCTP'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MAX_PATH_RETRANSMITS', '{"en":"MaxPathRetransmits","zh":"最大重传次数"}'::jsonb, true, false, 7
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.Transport.SCTP.MaxPathRetransmits'
 WHERE c.command_code = 'MOD TRANSPORT_SCTP'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'MAX_ASSOCIATION_RETRANSMITS', '{"en":"MaxAssociationRetransmits","zh":"本端的最大连续重传数目"}'::jsonb, true, false, 8
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.Transport.SCTP.MaxAssociationRetransmits'
 WHERE c.command_code = 'MOD TRANSPORT_SCTP'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'VAL_COOKIE_LIFE', '{"en":"ValCookieLife","zh":"有效cookie的生命周期"}'::jsonb, true, false, 9
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.Transport.SCTP.ValCookieLife'
 WHERE c.command_code = 'MOD TRANSPORT_SCTP'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'PLMNID_LIST', '{"en":"PLMNIDList","zh":"X2PLMN标识"}'::jsonb, true, false, 1
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.X2IpAddrMapInfo.{i}.PLMNID'
 WHERE c.command_code = 'MOD X2_IP_ADDR_MAP_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ENB_TYPE', '{"en":"EnbType","zh":"基站类型"}'::jsonb, true, false, 2
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.X2IpAddrMapInfo.{i}.EnbType'
 WHERE c.command_code = 'MOD X2_IP_ADDR_MAP_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'ENB_ID', '{"en":"EnbId","zh":"eNBID"}'::jsonb, true, false, 3
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.X2IpAddrMapInfo.{i}.EnbId'
 WHERE c.command_code = 'MOD X2_IP_ADDR_MAP_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'WAN_IP_ADDRESS', '{"en":"WanIpAddress","zh":"WAN口IP"}'::jsonb, true, false, 4
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.X2IpAddrMapInfo.{i}.WanIpAddress'
 WHERE c.command_code = 'MOD X2_IP_ADDR_MAP_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

INSERT INTO mml_command_sub_fields (command_id, standard_path_id, mml_code, label_i18n, default_selected, is_required, sort_order)
SELECT c.id, p.id, 'SUBNET_MASK', '{"en":"SubnetMask","zh":"子网掩码"}'::jsonb, true, false, 5
  FROM mml_commands  c
  JOIN standard_params p ON p.standard_path = 'Device.Services.FAPControl.X2IpAddrMapInfo.{i}.SubnetMask'
 WHERE c.command_code = 'MOD X2_IP_ADDR_MAP_INFO'
ON CONFLICT (command_id, standard_path_id) DO UPDATE SET
    mml_code   = EXCLUDED.mml_code,
    label_i18n = EXCLUDED.label_i18n,
    sort_order = EXCLUDED.sort_order;

COMMIT;

-- +goose Down
BEGIN;

-- 精确 DELETE 本 seed INSERT 的命令 (source='standard') 的 sub_field 关联
DELETE FROM mml_command_sub_fields
 WHERE command_id IN (
   SELECT id FROM mml_commands WHERE command_code IN (
     'LST DEVICE_INFO',
     'MOD DEVICE_INFO',
     'LST DEVICE_INFO_SW_UPGRADE',
     'LST SOFTWARE_CTRL',
     'MOD SOFTWARE_CTRL',
     'LST MANAGEMENT_SERVER',
     'MOD MANAGEMENT_SERVER',
     'LST FAULT_MGMT',
     'LST FAULT_MGMT_CURRENT_ALARM',
     'ADD FAULT_MGMT_CURRENT_ALARM',
     'RMV FAULT_MGMT_CURRENT_ALARM',
     'LST FAULT_MGMT_EXPEDITED_EVENT',
     'ADD FAULT_MGMT_EXPEDITED_EVENT',
     'RMV FAULT_MGMT_EXPEDITED_EVENT',
     'LST FAULT_MGMT_HISTORY_EVENT',
     'ADD FAULT_MGMT_HISTORY_EVENT',
     'RMV FAULT_MGMT_HISTORY_EVENT',
     'LST FAULT_MGMT_QUEUED_EVENT',
     'ADD FAULT_MGMT_QUEUED_EVENT',
     'RMV FAULT_MGMT_QUEUED_EVENT',
     'LST FAULT_MGMT_SUPPORTED_ALARM',
     'MOD FAULT_MGMT_SUPPORTED_ALARM',
     'ADD FAULT_MGMT_SUPPORTED_ALARM',
     'RMV FAULT_MGMT_SUPPORTED_ALARM',
     'LST LOG_MGMT',
     'MOD LOG_MGMT',
     'LST LTE',
     'MOD LTE',
     'LST LTE_GATEWAY',
     'MOD LTE_GATEWAY',
     'LST LTE_MME_POOL_CONFIG_PARAM',
     'MOD LTE_MME_POOL_CONFIG_PARAM',
     'ADD LTE_MME_POOL_CONFIG_PARAM',
     'RMV LTE_MME_POOL_CONFIG_PARAM',
     'LST LTE_S1U',
     'ADD LTE_S1U',
     'RMV LTE_S1U',
     'LST X2_IP_ADDR_MAP_INFO',
     'MOD X2_IP_ADDR_MAP_INFO',
     'ADD X2_IP_ADDR_MAP_INFO',
     'RMV X2_IP_ADDR_MAP_INFO',
     'LST ',
     'MOD ',
     'LST CAPABILITIES',
     'MOD CAPABILITIES',
     'LST CELL_CONFIG_CAPABILITIES',
     'MOD CELL_CONFIG_CAPABILITIES',
     'LST LTE_EPC',
     'MOD LTE_EPC',
     'LST TRANSPORT_SCTP',
     'MOD TRANSPORT_SCTP',
     'LST SCTP_ASSOC',
     'ADD SCTP_ASSOC',
     'RMV SCTP_ASSOC',
     'LST RAN_MAC',
     'MOD RAN_MAC',
     'LST RAN_PHY',
     'MOD RAN_PHY',
     'LST PHY_MBSFN',
     'MOD PHY_MBSFN',
     'LST CONN_MODE_EUTRA',
     'MOD CONN_MODE_EUTRA',
     'LST CONN_MODE_IRAT',
     'MOD CONN_MODE_IRAT',
     'LST MOBILITY_IDLE_MODE',
     'MOD MOBILITY_IDLE_MODE',
     'LST IDLE_MODE_IRAT',
     'MOD IDLE_MODE_IRAT',
     'LST SELF_CONFIG_SON_CONFIG_PARAM',
     'MOD SELF_CONFIG_SON_CONFIG_PARAM',
     'LST SELF_CONFIG',
     'ADD SELF_CONFIG',
     'RMV SELF_CONFIG',
     'LST ETHERNET_INTERFACE',
     'MOD ETHERNET_INTERFACE',
     'ADD ETHERNET_INTERFACE',
     'RMV ETHERNET_INTERFACE',
     'LST ETHERNET_IP_ROUTE',
     'MOD ETHERNET_IP_ROUTE',
     'ADD ETHERNET_IP_ROUTE',
     'RMV ETHERNET_IP_ROUTE',
     'LST I_PSEC',
     'MOD I_PSEC',
     'LST TIME',
     'MOD TIME',
     'LST FAP_GPS',
     'LST MR_MGMT_CONFIG',
     'MOD MR_MGMT_CONFIG',
     'ADD MR_MGMT_CONFIG',
     'RMV MR_MGMT_CONFIG',
     'LST PERF_MGMT_CONFIG',
     'MOD PERF_MGMT_CONFIG',
     'ADD PERF_MGMT_CONFIG',
     'RMV PERF_MGMT_CONFIG',
     'LST DEVICE_INFO_MU',
     'MOD DEVICE_INFO_MU',
     'LST MU_SW_UPGRADE',
     'ADD MU_SW_UPGRADE',
     'RMV MU_SW_UPGRADE'
   )
 );

DELETE FROM mml_commands
 WHERE command_code IN (
     'LST DEVICE_INFO',
     'MOD DEVICE_INFO',
     'LST DEVICE_INFO_SW_UPGRADE',
     'LST SOFTWARE_CTRL',
     'MOD SOFTWARE_CTRL',
     'LST MANAGEMENT_SERVER',
     'MOD MANAGEMENT_SERVER',
     'LST FAULT_MGMT',
     'LST FAULT_MGMT_CURRENT_ALARM',
     'ADD FAULT_MGMT_CURRENT_ALARM',
     'RMV FAULT_MGMT_CURRENT_ALARM',
     'LST FAULT_MGMT_EXPEDITED_EVENT',
     'ADD FAULT_MGMT_EXPEDITED_EVENT',
     'RMV FAULT_MGMT_EXPEDITED_EVENT',
     'LST FAULT_MGMT_HISTORY_EVENT',
     'ADD FAULT_MGMT_HISTORY_EVENT',
     'RMV FAULT_MGMT_HISTORY_EVENT',
     'LST FAULT_MGMT_QUEUED_EVENT',
     'ADD FAULT_MGMT_QUEUED_EVENT',
     'RMV FAULT_MGMT_QUEUED_EVENT',
     'LST FAULT_MGMT_SUPPORTED_ALARM',
     'MOD FAULT_MGMT_SUPPORTED_ALARM',
     'ADD FAULT_MGMT_SUPPORTED_ALARM',
     'RMV FAULT_MGMT_SUPPORTED_ALARM',
     'LST LOG_MGMT',
     'MOD LOG_MGMT',
     'LST LTE',
     'MOD LTE',
     'LST LTE_GATEWAY',
     'MOD LTE_GATEWAY',
     'LST LTE_MME_POOL_CONFIG_PARAM',
     'MOD LTE_MME_POOL_CONFIG_PARAM',
     'ADD LTE_MME_POOL_CONFIG_PARAM',
     'RMV LTE_MME_POOL_CONFIG_PARAM',
     'LST LTE_S1U',
     'ADD LTE_S1U',
     'RMV LTE_S1U',
     'LST X2_IP_ADDR_MAP_INFO',
     'MOD X2_IP_ADDR_MAP_INFO',
     'ADD X2_IP_ADDR_MAP_INFO',
     'RMV X2_IP_ADDR_MAP_INFO',
     'LST ',
     'MOD ',
     'LST CAPABILITIES',
     'MOD CAPABILITIES',
     'LST CELL_CONFIG_CAPABILITIES',
     'MOD CELL_CONFIG_CAPABILITIES',
     'LST LTE_EPC',
     'MOD LTE_EPC',
     'LST TRANSPORT_SCTP',
     'MOD TRANSPORT_SCTP',
     'LST SCTP_ASSOC',
     'ADD SCTP_ASSOC',
     'RMV SCTP_ASSOC',
     'LST RAN_MAC',
     'MOD RAN_MAC',
     'LST RAN_PHY',
     'MOD RAN_PHY',
     'LST PHY_MBSFN',
     'MOD PHY_MBSFN',
     'LST CONN_MODE_EUTRA',
     'MOD CONN_MODE_EUTRA',
     'LST CONN_MODE_IRAT',
     'MOD CONN_MODE_IRAT',
     'LST MOBILITY_IDLE_MODE',
     'MOD MOBILITY_IDLE_MODE',
     'LST IDLE_MODE_IRAT',
     'MOD IDLE_MODE_IRAT',
     'LST SELF_CONFIG_SON_CONFIG_PARAM',
     'MOD SELF_CONFIG_SON_CONFIG_PARAM',
     'LST SELF_CONFIG',
     'ADD SELF_CONFIG',
     'RMV SELF_CONFIG',
     'LST ETHERNET_INTERFACE',
     'MOD ETHERNET_INTERFACE',
     'ADD ETHERNET_INTERFACE',
     'RMV ETHERNET_INTERFACE',
     'LST ETHERNET_IP_ROUTE',
     'MOD ETHERNET_IP_ROUTE',
     'ADD ETHERNET_IP_ROUTE',
     'RMV ETHERNET_IP_ROUTE',
     'LST I_PSEC',
     'MOD I_PSEC',
     'LST TIME',
     'MOD TIME',
     'LST FAP_GPS',
     'LST MR_MGMT_CONFIG',
     'MOD MR_MGMT_CONFIG',
     'ADD MR_MGMT_CONFIG',
     'RMV MR_MGMT_CONFIG',
     'LST PERF_MGMT_CONFIG',
     'MOD PERF_MGMT_CONFIG',
     'ADD PERF_MGMT_CONFIG',
     'RMV PERF_MGMT_CONFIG',
     'LST DEVICE_INFO_MU',
     'MOD DEVICE_INFO_MU',
     'LST MU_SW_UPGRADE',
     'ADD MU_SW_UPGRADE',
     'RMV MU_SW_UPGRADE'
 );

-- standard_params 不删（多消费者共享，可能被其他 catalog 引用）
COMMIT;
