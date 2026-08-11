import type { Device } from '@core/types/device';
import { supportsOwnRF } from './deviceRadioFieldSupport';

type NetworkScope = 'common' | 'eNB' | 'gNB' | 'GSM';

interface DeviceListSyncParam {
  key: string;
  scope: NetworkScope;
  paths: string[];
}

const DEVICE_LIST_SYNC_PARAMS: DeviceListSyncParam[] = [
  { key: 'deviceModel', scope: 'common', paths: ['Device.DeviceInfo.ModelName'] },
  { key: 'softwareVersion', scope: 'common', paths: ['Device.DeviceInfo.SoftwareVersion'] },
  {
    key: 'macAddress',
    scope: 'common',
    paths: [
      'Device.Ethernet.Interface.MACAddress',
      'Device.Ethernet.Interface.{i}.MACAddress',
      'Device.DeviceInfo.X_COM_MACAddress',
    ],
  },
  { key: 'upTime', scope: 'common', paths: ['Device.DeviceInfo.UpTime', 'Device.DeviceInfo.X_COM_STATION_RUN_Time'] },
  { key: 'ueCount', scope: 'common', paths: ['Device.DeviceInfo.UE_Count'] },
  {
    key: 'mmeStatus',
    scope: 'eNB',
    paths: [
      'Device.Services.FAPService.{i}.FAPControl.LTE.Gateway.MmeStatus',
      'Device.Services.FAPService.{i}.CellConfig.LTE.EPC.MmePoolConfigParam.{i}.MME1Status',
      'Device.Services.FAPService.{i}.CellConfig.LTE.MmePoolConfigParam.{i}.MME1Status',
    ],
  },
  { key: 'amfStatus', scope: 'gNB', paths: ['Device.Services.FAPService.1.AmfsStatus'] },
  {
    key: 'rfStatus',
    scope: 'common',
    paths: [
      'Device.DeviceInfo.SAS.RadioEnable',
      'Device.DeviceInfo.SAS.RadioEnable1',
      'Device.DeviceInfo.SAS.RadioEnable2',
      'Device.DeviceInfo.SAS.RadioEnable3',
      'Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus',
    ],
  },
  {
    key: 'rfStatus',
    scope: 'eNB',
    paths: [
      'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.X_COM_RadioEnable',
      'Device.DeviceInfo.RU.{i}.RFTxStatus',
      'Device.DeviceInfo.EU.{i}.RU.{i}.RFTxStatus',
    ],
  },
  {
    key: 'rfStatus',
    scope: 'gNB',
    paths: [
      'Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.rftxEnable',
      'Device.DeviceInfo.CellConfig.{i}.SAS.RadioEnable',
    ],
  },
  {
    key: 'rfStatus',
    scope: 'GSM',
    paths: ['Device.Services.GsmBTSCellDT.{i}.RfState'],
  },
  {
    key: 'syncStatus',
    scope: 'common',
    paths: [
      'Device.Services.FAPService.1.FAPControl.PLLSyncState',
      'Device.FAP.Synchronization.ClockSourceSyncState',
      'Device.ManagementServer.tfcsSyncState',
      'Device.Services.FAPService.1.FAPControl.LTE.Gateway.X_COM_tfcsSyncState',
      'Device.Services.FAPService.1.FAPControl.NR.Gateway.X_COM_tfcsSyncState',
      'Device.DeviceInfo.X_COM_GPS_Status',
      'Device.DeviceInfo.GPS_Status',
      'Device.FAP.GPS.SyncStatus',
      'Device.DeviceInfo.X_COM_BDS_Status',
      'Device.DeviceInfo.BDS_Status',
      'Device.DeviceInfo.X_COM_1588_Status',
      'Device.DeviceInfo.1588_Status',
      'Device.FAP.PTP1588.SyncStatus',
    ],
  },
  {
    key: 'longitude',
    scope: 'common',
    paths: [
      'Device.DeviceInfo.SAS.FAP.GPS.LockedLongitude',
      'Device.DeviceInfo.SAS.FAP.GPS.LockedLongitude2',
      'Device.DeviceInfo.SAS.FAP.GPS.LockedLongitude3',
    ],
  },
  {
    key: 'latitude',
    scope: 'common',
    paths: [
      'Device.DeviceInfo.SAS.FAP.GPS.LockedLatitude',
      'Device.DeviceInfo.SAS.FAP.GPS.LockedLatitude2',
      'Device.DeviceInfo.SAS.FAP.GPS.LockedLatitude3',
    ],
  },
  { key: 'gpsHeight', scope: 'common', paths: ['Device.FAP.GPS.altidute', 'Device.FAP.GPS.Altitude', 'Device.FAP.GPS.LockedAltitude', 'Device.FAP.Synchronization.Altitude', 'Device.FAP.GPS.Height'] },
  { key: 'gpsSatelliteCount', scope: 'common', paths: ['Device.FAP.GPS.NumberOfSatellites'] },
  { key: 'pci', scope: 'common', paths: ['Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.PhyCellID', 'Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.RF.PhyCellID'] },
  { key: 'tac', scope: 'common', paths: ['Device.Services.FAPService.{i}.CellConfig.LTE.EPC.TAC', 'Device.Services.FAPService.{i}.CellConfig.{i}.NR.CN.TA.{i}.TAC'] },
  { key: 'band', scope: 'common', paths: ['Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.FreqBandIndicator', 'Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.PHY.FrequencyInfoDLSIB.MultiFrequencyBandListNRSIB.{i}.FreqBandIndicatorNR', 'Device.Services.GsmBTSCellDT.{i}.GsmBtsBand'] },
  { key: 'dlEarfcn', scope: 'common', paths: ['Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.EARFCNDL', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.EARFCNDL', 'Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.RF.NRARFCNDL'] },
  { key: 'ulEarfcn', scope: 'common', paths: ['Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.EARFCNUL', 'Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.RF.NRARFCNUL'] },
  {
    key: 'txPower',
    scope: 'eNB',
    paths: [
      'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.X_COM_MaxTxPowerExpanded',
      'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ReferenceSignalPower',
    ],
  },
  {
    key: 'txPower',
    scope: 'gNB',
    paths: ['Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.PowerModify'],
  },
  {
    key: 'txPower',
    scope: 'GSM',
    paths: [
      'Device.DeviceInfo.GSM.BtsRfPower',
      'Device.Services.GsmBTSCellDT.{i}.GsmBtsRFPower',
    ],
  },
  { key: 'adminState', scope: 'common', paths: ['Device.DeviceInfo.FAP_adminstate', 'Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.CellEnable.AdminState', 'Device.Services.FAPService.{i}.FAPControl.LTE.AdminState'] },
  { key: 'ipsecAddr', scope: 'common', paths: ['Device.FAP.Ipsec.{i}.TUNNEL_GATEWAY'] },
  { key: 'siteName', scope: 'eNB', paths: ['Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.UserLabel'] },
  {
    key: 'opState',
    scope: 'common',
    paths: [
      'Device.Services.FAPService.{i}.FAPControl.LTE.OpState',
      'Device.Services.FAPService.1.CellConfig.{i}.NR.RAN.OpState',
      'Device.Services.GsmBTSCellDT.{i}.OpState',
    ],
  },
  { key: 'plmnId', scope: 'eNB', paths: ['Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.PLMNID'] },
  { key: 'bandwidth', scope: 'eNB', paths: ['Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.DLBandwidth', 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ULBandwidth'] },
  { key: 'rootIndex', scope: 'eNB', paths: ['Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.PRACH.RootSequenceIndex'] },
  { key: 'subframeAssignment', scope: 'eNB', paths: ['Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.TDDFrame.SubFrameAssignment'] },
  { key: 'specialSubframe', scope: 'eNB', paths: ['Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.TDDFrame.SpecialSubframePatterns'] },
  { key: 'halobFlag', scope: 'gNB', paths: ['Device.Services.FAPService.{i}.FAPControl.NR.HaloBEnable'] },
  { key: 'multiPlmnEnable', scope: 'gNB', paths: ['Device.Services.FAPService.{i}.CellConfig.{i}.NR.CN.TA.{i}.MultiPLMNEnable'] },
  { key: 'lac', scope: 'GSM', paths: ['Device.DeviceInfo.BTS.CurrentLac', 'Device.DeviceInfo.GSM.CurrentLac', 'Device.Services.GsmBTSCellDT.{i}.CurrLocAreaCode'] },
  { key: 'arfcn', scope: 'GSM', paths: ['Device.DeviceInfo.BTS.CurrentArfcn', 'Device.DeviceInfo.GSM.CurrentArfcn', 'Device.Services.GsmBTSCellDT.{i}.CurrentArfcn', 'DeviceGSM.Bts.{i}.Trx.{i}.Arfcn'] },
  { key: 'bscSelect', scope: 'GSM', paths: ['DeviceGSM.BscSelect'] },
  { key: 'omlRemoteIp', scope: 'GSM', paths: ['DeviceGSM.OmlRemoteIp', 'DeviceGSM.OmlRemoteIpBak'] },
  { key: 'ipaUnitId', scope: 'GSM', paths: ['DeviceGSM.IpaUnitId'] },
];

function paramAppliesToDevice(param: DeviceListSyncParam, device: Device): boolean {
  if (param.key === 'rfStatus' && !supportsOwnRF(device)) return false;
  if (param.scope === 'common') return true;
  return device.networkType === param.scope;
}

export function getDeviceListParamSyncPaths(device: Device): string[] {
  const paths = DEVICE_LIST_SYNC_PARAMS
    .filter((param) => paramAppliesToDevice(param, device))
    .flatMap((param) => param.paths);
  return Array.from(new Set(paths));
}
