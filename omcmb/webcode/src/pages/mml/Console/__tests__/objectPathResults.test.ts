import { describe, expect, it } from 'vitest';
import { applyFrameToRow, buildDeviceRows, expandObjectPathColumns } from '../adapters';
import type { ResultColumn, ResultRow } from '../types';
import type { DeviceTaskResultItem } from '@core/types/mml';

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
});
