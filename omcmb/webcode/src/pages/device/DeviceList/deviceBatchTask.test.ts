import { describe, it, expect } from 'vitest';
import type { Device } from '@core/types/device';
import { buildBatchTaskTypeMap, batchActionHasDetail, removeParamSyncOptimisticDeviceId } from './deviceBatchTask';
import { getDeviceListParamSyncPaths } from './deviceListParamSync';

// 用恒等 t 让 map 值即 i18n key，便于断言键集与映射关系。
const t = (k: string) => k;

describe('buildBatchTaskTypeMap (#179 抓包入口移除)', () => {
  it('保留 reboot / log-collect / alarm-sync / param-sync 四个批量操作映射', () => {
    const map = buildBatchTaskTypeMap(t);
    expect(map['batch-reboot']).toBe('common.batchReboot');
    expect(map['batch-log-collect']).toBe('device.action.logCollect');
    expect(map['batch-alarm-sync']).toBe('device.action.alarmSync');
    expect(map['batch-param-sync']).toBe('device.action.paramSync');
  });

  it('不再识别 batch-tr069-collect（抓包按钮已移除）', () => {
    const map = buildBatchTaskTypeMap(t);
    expect('batch-tr069-collect' in map).toBe(false);
    expect(Object.keys(map)).toHaveLength(4);
  });
});

describe('batchActionHasDetail', () => {
  it('仅日志采集带详情', () => {
    expect(batchActionHasDetail('batch-log-collect')).toBe(true);
  });

  it('重启 / 告警同步 / 参数同步 / 已移除的抓包均无详情', () => {
    expect(batchActionHasDetail('batch-reboot')).toBe(false);
    expect(batchActionHasDetail('batch-alarm-sync')).toBe(false);
    expect(batchActionHasDetail('batch-param-sync')).toBe(false);
    expect(batchActionHasDetail('batch-tr069-collect')).toBe(false);
    expect(batchActionHasDetail(undefined)).toBe(false);
  });
});

describe('removeParamSyncOptimisticDeviceId', () => {
  it('参数同步任务进入终态后移除对应设备的转圈状态', () => {
    const prev = new Set(['device-1', 'device-2']);
    const next = removeParamSyncOptimisticDeviceId(prev, 'device-1');

    expect([...next]).toEqual(['device-2']);
    expect([...prev]).toEqual(['device-1', 'device-2']);
  });

  it('设备不在转圈集合里时复用原集合，避免无意义刷新', () => {
    const prev = new Set(['device-2']);
    const next = removeParamSyncOptimisticDeviceId(prev, 'device-1');

    expect(next).toBe(prev);
  });
});

describe('getDeviceListParamSyncPaths', () => {
  const baseDevice = {
    id: 'device-1',
    sn: 'SN001',
    networkType: 'eNB',
    isOnline: true,
  } as Device;

  it('下发固定、实例化和厂商兼容的 MAC 参数路径', () => {
    const paths = getDeviceListParamSyncPaths(baseDevice);

    expect(paths).toContain('Device.Ethernet.Interface.MACAddress');
    expect(paths).toContain('Device.Ethernet.Interface.{i}.MACAddress');
    expect(paths).toContain('Device.DeviceInfo.X_COM_MACAddress');
  });

  it('经纬度局部同步使用 SAS 标准路径并覆盖三个槽位', () => {
    const paths = getDeviceListParamSyncPaths(baseDevice);

    for (const suffix of ['', '2', '3']) {
      expect(paths).toContain(`Device.DeviceInfo.SAS.FAP.GPS.LockedLongitude${suffix}`);
      expect(paths).toContain(`Device.DeviceInfo.SAS.FAP.GPS.LockedLatitude${suffix}`);
    }
    expect(paths).not.toContain('Device.FAP.GPS.LockedLongitude');
    expect(paths).not.toContain('Device.FAP.GPS.LockedLatitude');
  });

  it('UE 数同步使用当前接入数标准路径，不查询 LTE 容量上限', () => {
    for (const networkType of ['eNB', 'gNB', 'GSM'] as const) {
      const paths = getDeviceListParamSyncPaths({ ...baseDevice, networkType });

      expect(paths).toContain('Device.DeviceInfo.UE_Count');
      expect(paths).not.toContain(
        'Device.Services.FAPService.{i}.CellConfig.AccessMgmt.LTE.MaxUEsServed',
      );
    }
  });

  it('RF 状态同步使用产品模型可翻译的标准路径', () => {
    const ltePaths = getDeviceListParamSyncPaths(baseDevice);
    const nrPaths = getDeviceListParamSyncPaths({ ...baseDevice, networkType: 'gNB' });
    const gsmPaths = getDeviceListParamSyncPaths({ ...baseDevice, networkType: 'GSM' });

    expect(ltePaths).toContain('Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus');
    expect(ltePaths).toContain('Device.DeviceInfo.SAS.RadioEnable');
    expect(nrPaths).toContain('Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.rftxEnable');
    expect(nrPaths).toContain('Device.DeviceInfo.SAS.RadioEnable');
    expect(nrPaths).toContain('Device.DeviceInfo.CellConfig.{i}.SAS.RadioEnable');
    expect(gsmPaths).toContain('Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus');
    expect(gsmPaths).toContain('Device.Services.GsmBTSCellDT.{i}.RfState');
  });

  it('BSC 不同步下级 RF 路径，BTS 仍同步自身 RF 路径', () => {
    const bscPaths = getDeviceListParamSyncPaths({
      ...baseDevice,
      networkType: 'GSM',
      productClass: 'FAP/PGSM',
    });
    const btsPaths = getDeviceListParamSyncPaths({
      ...baseDevice,
      networkType: 'GSM',
      productClass: 'FAP/BTS',
    });

    expect(bscPaths).not.toContain('Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus');
    expect(bscPaths).not.toContain('Device.Services.GsmBTSCellDT.{i}.RfState');
    expect(btsPaths).toContain('Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus');
    expect(btsPaths).toContain('Device.Services.GsmBTSCellDT.{i}.RfState');
  });

  it('下发列表 opState 后端派生实际读取的 LTE/NR/GSM 参数', () => {
    const paths = getDeviceListParamSyncPaths(baseDevice);

    expect(paths).toContain('Device.Services.FAPService.{i}.FAPControl.LTE.OpState');
    expect(paths).toContain('Device.Services.FAPService.1.CellConfig.{i}.NR.RAN.OpState');
    expect(paths).toContain('Device.Services.GsmBTSCellDT.{i}.OpState');
    expect(paths).not.toContain('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.OpState');
  });

  it('按制式下发实际发射功率参数且不把能力上限当成当前功率', () => {
    const ltePaths = getDeviceListParamSyncPaths(baseDevice);
    const nrPaths = getDeviceListParamSyncPaths({ ...baseDevice, networkType: 'gNB' });
    const gsmPaths = getDeviceListParamSyncPaths({ ...baseDevice, networkType: 'GSM' });

    expect(ltePaths).toContain('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ReferenceSignalPower');
    expect(ltePaths).toContain('Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.X_COM_MaxTxPowerExpanded');
    expect(nrPaths).toContain('Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.PowerModify');
    expect(gsmPaths).toContain('Device.DeviceInfo.GSM.BtsRfPower');
    expect(gsmPaths).toContain('Device.Services.GsmBTSCellDT.{i}.GsmBtsRFPower');

    for (const paths of [ltePaths, nrPaths, gsmPaths]) {
      expect(paths).not.toContain('Device.Services.FAPService.{i}.CellConfig.Capabilities.MaxTxPower');
      expect(paths).not.toContain('Device.Services.FAPService.{i}.Capabilities.MaxTxPower');
    }
  });

  it('BaiBNQ Band 使用产品模型中的 FreqBandIndicatorNR 路径', () => {
    const paths = getDeviceListParamSyncPaths({ ...baseDevice, networkType: 'gNB' });

    expect(paths).toContain(
      'Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.PHY.FrequencyInfoDLSIB.MultiFrequencyBandListNRSIB.{i}.FreqBandIndicatorNR',
    );
    expect(paths).not.toContain('Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.RF.FreqBandIndicator');
  });

  it('按制式下发 MME/AMF 状态参数', () => {
    const ltePaths = getDeviceListParamSyncPaths(baseDevice);
    const nrPaths = getDeviceListParamSyncPaths({ ...baseDevice, networkType: 'gNB' });

    expect(ltePaths).toContain('Device.Services.FAPService.{i}.FAPControl.LTE.Gateway.MmeStatus');
    expect(ltePaths).toContain('Device.Services.FAPService.{i}.CellConfig.LTE.EPC.MmePoolConfigParam.{i}.MME1Status');
    expect(ltePaths).toContain('Device.Services.FAPService.{i}.CellConfig.LTE.MmePoolConfigParam.{i}.MME1Status');
    expect(ltePaths).not.toContain('Device.Services.FAPService.1.AmfsStatus');
    expect(nrPaths).toContain('Device.Services.FAPService.1.AmfsStatus');
    expect(nrPaths).not.toContain('Device.Services.FAPService.{i}.FAPControl.LTE.Gateway.MmeStatus');
  });
});
