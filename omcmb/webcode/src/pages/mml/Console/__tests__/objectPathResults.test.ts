import { describe, expect, it } from 'vitest';
import {
  applyFrameToRow,
  buildDeviceRows,
  expandObjectPathColumns,
  mapTaskToRecord,
} from '../adapters';
import type { ResultColumn, ResultRow } from '../types';
import type { DeviceTaskResultItem } from '@core/types/mml';
import type { MMLTask } from '@core/types/mml';

const OBJECT_PATH = 'DeviceGSM.Bts.1.';
const BAND_PATH = `${OBJECT_PATH}Band`;
const BSIC_PATH = `${OBJECT_PATH}Bsic`;

const columns: ResultColumn[] = [
  { key: 'c0', label: '1', path: OBJECT_PATH },
];

const pendingRow: ResultRow = {
  deviceSn: 'E8F2971A3DC921A03D3E4FD4A0C1',
  deviceTaskId: '',
  status: 'running',
  cells: {},
  raw: '',
  elapsedMs: 0,
};

const rawResponse = `
  <cwmp:GetParameterValuesResponse xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
    <ParameterList>
      <ParameterValueStruct><Name>${BAND_PATH}</Name><Value>GSM900</Value></ParameterValueStruct>
      <ParameterValueStruct><Name>${BSIC_PATH}</Name><Value>9</Value></ParameterValueStruct>
    </ParameterList>
  </cwmp:GetParameterValuesResponse>`;

describe('GPV 对象路径结果展示', () => {
  it('保留对象路径下设备返回的所有叶子值，并展开为结果列', () => {
    const row = applyFrameToRow(
      pendingRow,
      {
        task_id: 'task-1',
        device_sn: pendingRow.deviceSn,
        status: 'completed',
        result: { method: 'GetParameterValuesResponse', raw_response: rawResponse },
      },
      columns,
      true,
    );

    expect(row.cells[BAND_PATH]).toBe('GSM900');
    expect(row.cells[BSIC_PATH]).toBe('9');
    expect(expandObjectPathColumns(columns, [row]).map((column) => column.path)).toEqual([
      BAND_PATH,
      BSIC_PATH,
    ]);
  });

  it('任务完成后从持久化结果重建时也保留对象路径的所有叶子值', () => {
    const item: DeviceTaskResultItem = {
      deviceSn: pendingRow.deviceSn,
      result: {
        success: true,
        rawOutput: rawResponse,
        parsedData: { method: 'GetParameterValuesResponse', raw_response: rawResponse },
        executionTime: 600,
        timestamp: '',
      },
    };

    const [row] = buildDeviceRows([item], columns, true);
    expect(row.cells[BAND_PATH]).toBe('GSM900');
    expect(row.cells[BSIC_PATH]).toBe('9');
  });

  it('设备未返回对象后代时保留原对象列作为空结果占位', () => {
    expect(expandObjectPathColumns(columns, [pendingRow])).toEqual(columns);
  });

  it('叶子选择折叠为对象查询时只展开匹配原始 Path 模板的返回值', () => {
    const dlTemplate = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.DLBandwidth';
    const ulTemplate = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ULBandwidth';
    const objectColumn = {
      key: 'bandwidth',
      label: '下行带宽',
      path: 'Device.Services.FAPService.',
      selectedPathTemplates: [dlTemplate, ulTemplate],
    } as ResultColumn;
    const dlPath = 'Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.DLBandwidth';
    const ulPath = 'Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.ULBandwidth';
    const unrelatedPath = 'Device.Services.FAPService.1.CellConfig.NR.CN.EmergencyAreaID';
    const row = {
      ...pendingRow,
      cells: {
        [unrelatedPath]: '0',
        [dlPath]: '100',
        [ulPath]: '100',
      },
    };

    expect(expandObjectPathColumns([objectColumn], [row]).map((column) => column.path)).toEqual([
      dlPath,
      ulPath,
    ]);
  });

  it('同名叶子来自不同对象实例时表头必须带实例上下文', () => {
    const dlTemplate = 'Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.RF.DLBandwidth';
    const ulTemplate = 'Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.RF.ULBandwidth';
    const objectColumn: ResultColumn = {
      key: 'bandwidth',
      label: '下行带宽',
      path: 'Device.Services.FAPService.1.CellConfig.',
      selectedPathTemplates: [dlTemplate, ulTemplate],
    };
    const paths = [
      'Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.DLBandwidth',
      'Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.ULBandwidth',
      'Device.Services.FAPService.1.CellConfig.2.NR.RAN.RF.DLBandwidth',
      'Device.Services.FAPService.1.CellConfig.2.NR.RAN.RF.ULBandwidth',
    ];
    const row = {
      ...pendingRow,
      cells: Object.fromEntries(paths.map((path) => [path, '40'])),
    };

    expect(expandObjectPathColumns([objectColumn], [row]).map((column) => column.label)).toEqual([
      'DLBandwidth [CellConfig.1]',
      'ULBandwidth [CellConfig.1]',
      'DLBandwidth [CellConfig.2]',
      'ULBandwidth [CellConfig.2]',
    ]);
  });

  it('多小区结果按小区实例号数值排序，小区内保持所选参数顺序', () => {
    const templates = [
      'Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.RF.PhyCellID',
      'Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.RF.DLBandwidth',
    ];
    const objectColumn: ResultColumn = {
      key: 'cell-rf',
      label: '小区射频参数',
      path: 'Device.Services.FAPService.1.CellConfig.',
      selectedPathTemplates: templates,
    };
    const path = (cell: number, leaf: string) => (
      `Device.Services.FAPService.1.CellConfig.${cell}.NR.RAN.RF.${leaf}`
    );
    const interleavedPaths = [
      path(2, 'DLBandwidth'),
      path(1, 'DLBandwidth'),
      path(10, 'PhyCellID'),
      path(2, 'PhyCellID'),
      path(10, 'DLBandwidth'),
      path(1, 'PhyCellID'),
    ];
    const row = {
      ...pendingRow,
      cells: Object.fromEntries(interleavedPaths.map((item) => [item, '40'])),
    };

    expect(expandObjectPathColumns([objectColumn], [row]).map((column) => column.path)).toEqual([
      path(1, 'PhyCellID'),
      path(1, 'DLBandwidth'),
      path(2, 'PhyCellID'),
      path(2, 'DLBandwidth'),
      path(10, 'PhyCellID'),
      path(10, 'DLBandwidth'),
    ]);
  });

  it('非 CellConfig 动态对象也按实例号排序，并在实例内保持所选参数顺序', () => {
    const templates = [
      'Device.DeviceInfo.MU.{i}.Slot.{i}.SwUpgrade.Status',
      'Device.DeviceInfo.MU.{i}.Slot.{i}.SwUpgrade.FailureCause',
    ];
    const objectColumn: ResultColumn = {
      key: 'slot-upgrade',
      label: '槽位升级状态',
      path: 'Device.DeviceInfo.MU.1.Slot.',
      selectedPathTemplates: templates,
    };
    const path = (slot: number, leaf: string) => (
      `Device.DeviceInfo.MU.1.Slot.${slot}.SwUpgrade.${leaf}`
    );
    const interleavedPaths = [
      path(10, 'FailureCause'),
      path(2, 'Status'),
      path(1, 'FailureCause'),
      path(10, 'Status'),
      path(1, 'Status'),
      path(2, 'FailureCause'),
    ];
    const row = {
      ...pendingRow,
      cells: Object.fromEntries(interleavedPaths.map((item) => [item, 'ok'])),
    };

    expect(expandObjectPathColumns([objectColumn], [row]).map((column) => column.path)).toEqual([
      path(1, 'Status'),
      path(1, 'FailureCause'),
      path(2, 'Status'),
      path(2, 'FailureCause'),
      path(10, 'Status'),
      path(10, 'FailureCause'),
    ]);
  });

  it('QOS 多实例按数值实例号排序，不能按 1、10、2 的字符串顺序', () => {
    const templates = [
      'Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.QOS.{i}.5QI',
      'Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.QOS.{i}.AllowedIntegrityAlgo',
    ];
    const objectColumn: ResultColumn = {
      key: 'qos',
      label: 'QOS',
      path: 'Device.Services.FAPService.1.CellConfig.1.NR.RAN.QOS.',
      selectedPathTemplates: templates,
    };
    const path = (qos: number, leaf: string) => `${objectColumn.path}${qos}.${leaf}`;
    const lexicographicDevicePaths = [
      path(1, '5QI'),
      path(10, '5QI'),
      path(2, '5QI'),
      path(1, 'AllowedIntegrityAlgo'),
      path(10, 'AllowedIntegrityAlgo'),
      path(2, 'AllowedIntegrityAlgo'),
    ];
    const row = {
      ...pendingRow,
      cells: Object.fromEntries(lexicographicDevicePaths.map((item) => [item, '1'])),
    };

    expect(expandObjectPathColumns([objectColumn], [row]).map((column) => column.path)).toEqual([
      path(1, '5QI'),
      path(1, 'AllowedIntegrityAlgo'),
      path(2, '5QI'),
      path(2, 'AllowedIntegrityAlgo'),
      path(10, '5QI'),
      path(10, 'AllowedIntegrityAlgo'),
    ]);
  });

  it('QOS 10 个实例的所有参数都应解析并展开，不得在 80 列截断', () => {
    const root = 'Device.Services.FAPService.1.CellConfig.1.NR.RAN.QOS.';
    const leafNames = [
      '5QI',
      ...Array.from({ length: 34 }, (_value, index) => `Param${index + 2}`),
    ];
    const templates = leafNames.map(
      (leaf) => `Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.QOS.{i}.${leaf}`,
    );
    const objectColumn: ResultColumn = {
      key: 'qos-all',
      label: 'QOS',
      path: root,
      selectedPathTemplates: templates,
    };
    const paths = Array.from({ length: 10 }, (_value, instanceIndex) => (
      leafNames.map((leaf) => `${root}${instanceIndex + 1}.${leaf}`)
    )).flat();
    const response = `<cwmp:GetParameterValuesResponse xmlns:cwmp="urn:dslforum-org:cwmp-1-0"><ParameterList>${paths.map((path) => (
      `<ParameterValueStruct><Name>${path}</Name><Value>1</Value></ParameterValueStruct>`
    )).join('')}</ParameterList></cwmp:GetParameterValuesResponse>`;
    const item: DeviceTaskResultItem = {
      deviceSn: pendingRow.deviceSn,
      result: {
        success: true,
        rawOutput: response,
        parsedData: { method: 'GetParameterValuesResponse', raw_response: response },
        executionTime: 600,
        timestamp: '',
      },
    };

    const rows = buildDeviceRows([item], [objectColumn], true);
    expect(Object.keys(rows[0].cells)).toHaveLength(350);

    const expanded = expandObjectPathColumns([objectColumn], rows);
    expect(expanded).toHaveLength(350);
    expect(expanded.at(-1)?.path).toBe(`${root}10.Param35`);
    expect(expanded.some((column) => column.path === `${root}10.5QI`)).toBe(true);
  });

  it('多层动态对象按外层到内层实例号排序', () => {
    const templates = [
      'Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.NeighborList.NRCell.{i}.PhyCellID',
      'Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.NeighborList.NRCell.{i}.PLMNID',
    ];
    const objectColumn: ResultColumn = {
      key: 'nr-neighbor',
      label: 'NR 邻区',
      path: 'Device.Services.FAPService.1.CellConfig.',
      selectedPathTemplates: templates,
    };
    const path = (cell: number, neighbor: number, leaf: string) => (
      `Device.Services.FAPService.1.CellConfig.${cell}.NR.RAN.NeighborList.NRCell.${neighbor}.${leaf}`
    );
    const interleavedPaths = [
      path(2, 1, 'PLMNID'),
      path(1, 10, 'PhyCellID'),
      path(1, 2, 'PLMNID'),
      path(1, 2, 'PhyCellID'),
      path(2, 1, 'PhyCellID'),
      path(1, 10, 'PLMNID'),
    ];
    const row = {
      ...pendingRow,
      cells: Object.fromEntries(interleavedPaths.map((item) => [item, '1'])),
    };

    expect(expandObjectPathColumns([objectColumn], [row]).map((column) => column.path)).toEqual([
      path(1, 2, 'PhyCellID'),
      path(1, 2, 'PLMNID'),
      path(1, 10, 'PhyCellID'),
      path(1, 10, 'PLMNID'),
      path(2, 1, 'PhyCellID'),
      path(2, 1, 'PLMNID'),
    ]);
  });

  it('显式对象查询没有叶子模板时仍按返回对象实例号排序', () => {
    const objectColumn: ResultColumn = {
      key: 'bts',
      label: 'BTS',
      path: 'DeviceGSM.Bts.',
    };
    const paths = [
      'DeviceGSM.Bts.10.Band',
      'DeviceGSM.Bts.2.Band',
      'DeviceGSM.Bts.1.Band',
    ];
    const row = {
      ...pendingRow,
      cells: Object.fromEntries(paths.map((item) => [item, 'GSM900'])),
    };

    expect(expandObjectPathColumns([objectColumn], [row]).map((column) => column.path)).toEqual([
      'DeviceGSM.Bts.1.Band',
      'DeviceGSM.Bts.2.Band',
      'DeviceGSM.Bts.10.Band',
    ]);
  });

  it('大对象保留全部返回参数，并按实例号排序', () => {
    const objectColumn: ResultColumn = {
      key: 'bts-large',
      label: 'BTS',
      path: 'DeviceGSM.Bts.',
    };
    const descendingPaths = Array.from(
      { length: 100 },
      (_value, index) => `DeviceGSM.Bts.${100 - index}.Band`,
    );
    const row = {
      ...pendingRow,
      cells: Object.fromEntries(descendingPaths.map((item) => [item, 'GSM900'])),
    };

    const result = expandObjectPathColumns([objectColumn], [row]);
    expect(result).toHaveLength(100);
    expect(result[0].path).toBe('DeviceGSM.Bts.1.Band');
    expect(result[99].path).toBe('DeviceGSM.Bts.100.Band');
  });

  it('刷新后从任务快照重建时继续保留对象查询的原始叶子模板', () => {
    const templates = [
      'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.DLBandwidth',
      'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ULBandwidth',
    ];
    const task = {
      id: 'task-object-filter',
      taskName: '带宽查询',
      taskOrigin: 'console',
      deviceSns: [pendingRow.deviceSn],
      commands: ['LST FAP_SERVICE'],
      commandsDetail: [{
        commandCode: 'LST FAP_SERVICE',
        commandName: '查询 FAP 载波基本配置',
        operationType: 'LST',
        paramPaths: ['Device.Services.FAPService.', 'Device.Services.FAPService.'],
        selectedStandardPaths: templates,
      }],
      results: [],
      status: 'completed',
      createdAt: '',
      updatedAt: '',
      creator: '',
      executeType: 'immediate',
      offlineRetry: false,
      offlineRetryWait: 0,
      failedRetry: false,
      failedRetryCount: 0,
      failedRetryInterval: 0,
      totalDevices: 1,
      successCount: 1,
      failedCount: 0,
    } as MMLTask;

    expect(mapTaskToRecord(task).columns).toEqual([
      expect.objectContaining({
        path: 'Device.Services.FAPService.',
        selectedPathTemplates: templates,
      }),
    ]);
  });

  it('对象前缀重叠时同一返回后代 Path 只展开一次', () => {
    const overlappingColumns: ResultColumn[] = [
      { key: 'c0', label: 'A', path: 'Device.A.' },
      { key: 'c1', label: '1', path: 'Device.A.1.' },
    ];
    const descendantPath = 'Device.A.1.Name';
    const row = {
      ...pendingRow,
      cells: { [descendantPath]: 'first' },
    };

    expect(
      expandObjectPathColumns(overlappingColumns, [row])
        .map(({ key, path }) => ({ key, path })),
    ).toEqual([
      { key: 'c0:child:0', path: descendantPath },
    ]);
  });

  it('显式叶子与对象动态后代命中同一 Path 时保留首次出现列', () => {
    const objectColumn: ResultColumn = { key: 'object', label: 'A', path: 'Device.A.' };
    const leafColumn: ResultColumn = {
      key: 'leaf',
      label: 'Name',
      path: 'Device.A.1.Name',
    };
    const row = {
      ...pendingRow,
      cells: { [leafColumn.path]: 'first' },
    };
    const keyAndPath = (resultColumns: ResultColumn[]) =>
      resultColumns.map(({ key, path }) => ({ key, path }));

    expect(keyAndPath(expandObjectPathColumns([leafColumn, objectColumn], [row]))).toEqual([
      { key: 'leaf', path: leafColumn.path },
    ]);
    expect(keyAndPath(expandObjectPathColumns([objectColumn, leafColumn], [row]))).toEqual([
      { key: 'object:child:0', path: leafColumn.path },
    ]);
  });

  it('对象展开已有同结构实值时移除模型别名产生的同名空列', () => {
    const aliasColumn: ResultColumn = {
      key: 'alias-cid',
      label: 'CID',
      path: 'Device.Services.FAPService.{i}.CellConfig.{i}.LTE.RAN.NeighborList.LTECell.{i}.CID',
    };
    const objectColumn: ResultColumn = {
      key: 'nr-lte-cell',
      label: 'LTECell',
      path: 'Device.Services.FAPService.1.CellConfig.1.NR.RAN.NeighborList.LTECell.',
    };
    const actualPath = `${objectColumn.path}1.CID`;
    const row = { ...pendingRow, cells: { [actualPath]: '1' } };

    expect(expandObjectPathColumns([aliasColumn, objectColumn], [row])).toEqual([
      { key: 'nr-lte-cell:child:0', label: 'CID', path: actualPath },
    ]);
  });

  it('不同对象的同名叶子不能因为一列为空而被误删', () => {
    const servingCell: ResultColumn = {
      key: 'serving-cid',
      label: 'CID',
      path: 'Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.Common.CID',
    };
    const neighborObject: ResultColumn = {
      key: 'neighbor',
      label: 'LTECell',
      path: 'Device.Services.FAPService.1.CellConfig.1.NR.RAN.NeighborList.LTECell.',
    };
    const neighborCID = `${neighborObject.path}1.CID`;
    const row = { ...pendingRow, cells: { [neighborCID]: '1' } };

    expect(expandObjectPathColumns([servingCell, neighborObject], [row]).map((column) => column.path)).toEqual([
      servingCell.path,
      neighborCID,
    ]);
  });

  it('大对象路径不按固定列数截断', () => {
    const row = {
      ...pendingRow,
      cells: Object.fromEntries(
        Array.from({ length: 100 }, (_v, i) => [
          `${OBJECT_PATH}Param${i}`,
          String(i),
        ]),
      ),
    };

    const result = expandObjectPathColumns(columns, [row]);
    expect(result).toHaveLength(100);
    expect(result[0].path).toBe(`${OBJECT_PATH}Param0`);
    expect(result[99].path).toBe(`${OBJECT_PATH}Param99`);
  });
});
