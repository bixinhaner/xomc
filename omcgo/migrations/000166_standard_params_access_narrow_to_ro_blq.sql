-- 000166_standard_params_access_narrow_to_ro_blq.sql
--
-- 把 95 条 standard_params.access 由 READ_WRITE 收窄为 READ_ONLY（基于 BLQ CPE
-- 早期 GetParameterNames 实测：device_parameters.writable=false）。
--
-- 背景（QA v2 §3 类 B 修复）：
--   1. M1a/M1b 失败：MOD Device.DeviceInfo.UserLabel / DnPrefix 等字段返
--      9003 [Client] Invalid arguments — CPE 拒写
--   2. 根因：standard_params.access='READ_WRITE' 不反映 BaiBLQ 实际能力
--   3. 数据源：device_parameters 表（早期 ACS 收 GetParameterNamesResponse
--      时由 RPCResponseSubscriber 写入），其中 writable 来自 CPE 自报告
--   4. 取证 SQL：standard_params LEFT JOIN device_parameters，针对 BLQ device
--      4c15e314-5daf-4b5c-a5a6-e626f3796f58 全部实例归一聚合后 all_writable=false
--      的 95 条 path（详见 docs/design/mml-access-probe-narrow-to-ro-20260524.csv）
--
-- 影响：
--   - admin_repository.go::ListEnrichedByCommand SQL COALESCE(sp.access, 'READ_ONLY')
--     → 这 95 条 path 对应的 sub_fields 返回 access_type='READ_ONLY'
--   - 前端 AccessTypeTag 显示"只读"；SubFieldInputList MOD 模式过滤掉只读字段
--   - 用户不再尝试 MOD 这些字段 → 不再撞 9003
--
-- 边界 / 风险：
--   - 单设备实测（仅 BLQ FAP/mBS31001/SC）覆盖全局字典：可接受，因当前只支持
--     一种 product class；未来若引入支持写这些字段的其他 CPE，需要走 X3
--     (discovered_param_mappings.discovered_access) 隔离方案
--   - device_parameters.writable 可能反映运行态锁（如 cell 工作中 PCI 锁），
--     非纯协议级 access；但 95 条均为系统配置类（DnPrefix / MRMgmt Config /
--     SAS RadioEnable 等），日常运维不应频繁改写
--
-- 回滚：down 段对同样 95 条 path 反向 UPDATE 回 READ_WRITE

-- +goose Up

UPDATE standard_params
   SET access = 'READ_ONLY',
       updated_at = NOW()
 WHERE access = 'READ_WRITE'
   AND standard_path IN (
  'Device.DeviceInfo.AntennaInfo.HeightType',
  'Device.DeviceInfo.DnPrefix',
  'Device.DeviceInfo.SAS.RadioEnable',
  'Device.DeviceInfo.SAS.RadioEnable3',
  'Device.FAP.MRMgmt.Config.{i}.MRECGIList',
  'Device.FAP.MRMgmt.Config.{i}.MeasureItems',
  'Device.FAP.MRMgmt.Config.{i}.MeasureType',
  'Device.FAP.MRMgmt.Config.{i}.MrPassword',
  'Device.FAP.MRMgmt.Config.{i}.MrUsername',
  'Device.FAP.MRMgmt.Config.{i}.PrbNum',
  'Device.FAP.MRMgmt.Config.{i}.SampleBeginTime',
  'Device.FAP.MRMgmt.Config.{i}.SampleEndTime',
  'Device.FAP.MRMgmt.Config.{i}.SamplePeriod',
  'Device.FAP.MRMgmt.Config.{i}.SubFrameNum',
  'Device.FAP.PerfMgmt.Config.{i}.Password',
  'Device.FAP.PerfMgmt.Config.{i}.PeriodicUploadTime',
  'Device.FAP.PerfMgmt.Config.{i}.Username',
  'Device.IP.Interface.{i}.IPv4Address.{i}.AddressingType',
  'Device.IP.Interface.{i}.IPv4Address.{i}.IPAddress',
  'Device.IP.Interface.{i}.IPv4Address.{i}.SubnetMask',
  'Device.ManagementServer.ParameterKey',
  'Device.ManagementServer.URL',
  'Device.Services.FAPService.Ipsec.IPSEC_PORT',
  'Device.Services.FAPService.Ipsec.IPSEC_PORT_NAT_T',
  'Device.Services.FAPService.Ipsec.IPSEC_RIGHTIKEPORT',
  'Device.Services.FAPService.Ipsec.IpsecUsimAuthenticationEnable',
  'Device.Services.FAPService.Ipsec.LEFT_INTERFACE',
  'Device.Services.FAPService.{i}.Capabilities.LTE.NNSFSupported',
  'Device.Services.FAPService.{i}.CellConfig.LTE.EPC.EAID',
  'Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.CellReservedForOperatorUse',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CA.PARAMS.CARRIER_MODE',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CellRestriction.CellBarred',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.DRX.DRXEnabled',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.ContentionResolutionTimer',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MaxHARQMsg3Tx',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MessagePowerOffsetGroupB',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MessageSizeGroupA',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.NumberOfRaPreambles',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.PreambleTransMax',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.ResponseWindowSize',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.SizeOfRaGroupA',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.MaxHARQTx',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.PeriodicBSRTimer',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.RetxBSRTimer',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.TTIBundling',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.MeasQuantityUTRAFDD',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.QoffsetGERAN',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.TReselectionGERAN',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.TReselectionUTRA',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.TReselectionEUTRASFHigh',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.TReselectionEUTRASFMedium',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRASFHigh',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRASFMedium',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.NeighCellTypeContainer',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.RSTxPower',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.X_COM_TAC',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.AntennaInfo.AntennaPortsCount',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.AntennaInfo.Multiplex',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.HighSpeedFlag',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.CQIPUCCHResourceIndex',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.DeltaPUCCHShift',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.K',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.N1PUCCHAN',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.NCSAN',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.NRBCQI',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.Enable64QAM',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.HoppingMode',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.HoppingOffset',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.NSB',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.AckNackSRSSimultaneousTransmission',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSBandwidthConfig',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSEnabled',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSMaxUpPTS',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.AdminCellState',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.LteCellWithRuList',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PBCHPowerOffset',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PSCHPowerOffset',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.SSCHPowerOffset',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.X_COM_RadioEnable',
  'Device.Services.FAPService.{i}.FAPControl.EciAutoEnable',
  'Device.Services.FAPService.{i}.FAPControl.LTE.AdminState',
  'Device.Services.FAPService.{i}.FAPControl.LTE.Gateway.ExistPlmnidList',
  'Device.Services.FAPService.{i}.FAPControl.LTE.Gateway.SecGWServer1',
  'Device.Services.FAPService.{i}.FAPControl.LTE.Gateway.SecGWServer2',
  'Device.Services.FAPService.{i}.FAPControl.LTE.Gateway.SecGWServer3',
  'Device.Services.FAPService.{i}.FAPControl.LTE.OpState',
  'Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus',
  'Device.Services.FAPService.{i}.Transport.SCTP.HBInterval',
  'Device.Services.FAPService.{i}.Transport.SCTP.MaxAssociationRetransmits',
  'Device.Services.FAPService.{i}.Transport.SCTP.MaxInitRetransmits',
  'Device.Services.FAPService.{i}.Transport.SCTP.MaxPathRetransmits',
  'Device.Services.FAPService.{i}.Transport.SCTP.RTOInitial',
  'Device.Services.FAPService.{i}.Transport.SCTP.RTOMax',
  'Device.Services.FAPService.{i}.Transport.SCTP.RTOMin',
  'Device.Services.FAPService.{i}.Transport.SCTP.ValCookieLife'
);

-- +goose Down

UPDATE standard_params
   SET access = 'READ_WRITE',
       updated_at = NOW()
 WHERE access = 'READ_ONLY'
   AND standard_path IN (
  'Device.DeviceInfo.AntennaInfo.HeightType',
  'Device.DeviceInfo.DnPrefix',
  'Device.DeviceInfo.SAS.RadioEnable',
  'Device.DeviceInfo.SAS.RadioEnable3',
  'Device.FAP.MRMgmt.Config.{i}.MRECGIList',
  'Device.FAP.MRMgmt.Config.{i}.MeasureItems',
  'Device.FAP.MRMgmt.Config.{i}.MeasureType',
  'Device.FAP.MRMgmt.Config.{i}.MrPassword',
  'Device.FAP.MRMgmt.Config.{i}.MrUsername',
  'Device.FAP.MRMgmt.Config.{i}.PrbNum',
  'Device.FAP.MRMgmt.Config.{i}.SampleBeginTime',
  'Device.FAP.MRMgmt.Config.{i}.SampleEndTime',
  'Device.FAP.MRMgmt.Config.{i}.SamplePeriod',
  'Device.FAP.MRMgmt.Config.{i}.SubFrameNum',
  'Device.FAP.PerfMgmt.Config.{i}.Password',
  'Device.FAP.PerfMgmt.Config.{i}.PeriodicUploadTime',
  'Device.FAP.PerfMgmt.Config.{i}.Username',
  'Device.IP.Interface.{i}.IPv4Address.{i}.AddressingType',
  'Device.IP.Interface.{i}.IPv4Address.{i}.IPAddress',
  'Device.IP.Interface.{i}.IPv4Address.{i}.SubnetMask',
  'Device.ManagementServer.ParameterKey',
  'Device.ManagementServer.URL',
  'Device.Services.FAPService.Ipsec.IPSEC_PORT',
  'Device.Services.FAPService.Ipsec.IPSEC_PORT_NAT_T',
  'Device.Services.FAPService.Ipsec.IPSEC_RIGHTIKEPORT',
  'Device.Services.FAPService.Ipsec.IpsecUsimAuthenticationEnable',
  'Device.Services.FAPService.Ipsec.LEFT_INTERFACE',
  'Device.Services.FAPService.{i}.Capabilities.LTE.NNSFSupported',
  'Device.Services.FAPService.{i}.CellConfig.LTE.EPC.EAID',
  'Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.CellReservedForOperatorUse',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CA.PARAMS.CARRIER_MODE',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.CellRestriction.CellBarred',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.DRX.DRXEnabled',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.ContentionResolutionTimer',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MaxHARQMsg3Tx',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MessagePowerOffsetGroupB',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.MessageSizeGroupA',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.NumberOfRaPreambles',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.PreambleTransMax',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.ResponseWindowSize',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.RACH.SizeOfRaGroupA',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.MaxHARQTx',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.PeriodicBSRTimer',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.RetxBSRTimer',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.ULSCH.TTIBundling',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.MeasQuantityUTRAFDD',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.ConnMode.IRAT.QoffsetGERAN',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.GERAN.TReselectionGERAN',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IRAT.UTRA.TReselectionUTRA',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.TReselectionEUTRASFHigh',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}.TReselectionEUTRASFMedium',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRASFHigh',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.IntraFreq.TReselectionEUTRASFMedium',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.NeighCellTypeContainer',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.RSTxPower',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.X_COM_TAC',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.AntennaInfo.AntennaPortsCount',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.AntennaInfo.Multiplex',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.HighSpeedFlag',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.CQIPUCCHResourceIndex',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.DeltaPUCCHShift',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.K',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.N1PUCCHAN',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.NCSAN',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUCCH.NRBCQI',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.Enable64QAM',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.HoppingMode',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.HoppingOffset',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PUSCH.NSB',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.AckNackSRSSimultaneousTransmission',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSBandwidthConfig',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSEnabled',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.SRS.SRSMaxUpPTS',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.AdminCellState',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.LteCellWithRuList',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PBCHPowerOffset',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PSCHPowerOffset',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.SSCHPowerOffset',
  'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.X_COM_RadioEnable',
  'Device.Services.FAPService.{i}.FAPControl.EciAutoEnable',
  'Device.Services.FAPService.{i}.FAPControl.LTE.AdminState',
  'Device.Services.FAPService.{i}.FAPControl.LTE.Gateway.ExistPlmnidList',
  'Device.Services.FAPService.{i}.FAPControl.LTE.Gateway.SecGWServer1',
  'Device.Services.FAPService.{i}.FAPControl.LTE.Gateway.SecGWServer2',
  'Device.Services.FAPService.{i}.FAPControl.LTE.Gateway.SecGWServer3',
  'Device.Services.FAPService.{i}.FAPControl.LTE.OpState',
  'Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus',
  'Device.Services.FAPService.{i}.Transport.SCTP.HBInterval',
  'Device.Services.FAPService.{i}.Transport.SCTP.MaxAssociationRetransmits',
  'Device.Services.FAPService.{i}.Transport.SCTP.MaxInitRetransmits',
  'Device.Services.FAPService.{i}.Transport.SCTP.MaxPathRetransmits',
  'Device.Services.FAPService.{i}.Transport.SCTP.RTOInitial',
  'Device.Services.FAPService.{i}.Transport.SCTP.RTOMax',
  'Device.Services.FAPService.{i}.Transport.SCTP.RTOMin',
  'Device.Services.FAPService.{i}.Transport.SCTP.ValCookieLife'
);
