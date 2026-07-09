import { describe, expect, it } from 'vitest';

import { parseMmlScriptPlan } from '../mmlScriptPlanParser';

describe('parseMmlScriptPlan', () => {
  it('parses semicolon device-bound text lines into plan items', () => {
    const result = parseMmlScriptPlan(
      [
        'LST EUTRANNFREQ;SN001',
        'MOD EUTRANNFREQ:CELL_INDEX={1},LTE_INTER_FREQ_DL_EARFCN={41390};SN001',
        'LST CELL;SN002;',
      ].join('\n'),
    );

    expect(result.executeMode).toBe('device_bound');
    expect(result.deviceSns).toEqual(['SN001', 'SN002']);
    expect(result.planItems).toHaveLength(3);
    expect(result.planItems[1]).toMatchObject({
      lineNo: 2,
      deviceSn: 'SN001',
      order: 2,
      command: {
        commandCode: 'MOD EUTRANNFREQ',
        operationType: 'MOD',
        parameters: {
          CELL_INDEX: '{1}',
          LTE_INTER_FREQ_DL_EARFCN: '{41390}',
        },
      },
    });
  });

  it('parses csv plan rows with headers', () => {
    const result = parseMmlScriptPlan(
      [
        'line_no,device_sn,order,command',
        '7,SN100,3,"ADD EUTRANNFREQ:LTE_INTER_FREQ_DL_EARFCN={41390}"',
      ].join('\n'),
      { format: 'csv' },
    );

    expect(result.executeMode).toBe('device_bound');
    expect(result.planItems).toEqual([
      {
        lineNo: 7,
        deviceSn: 'SN100',
        order: 3,
        rawLine: '7,SN100,3,ADD EUTRANNFREQ:LTE_INTER_FREQ_DL_EARFCN={41390}',
        command: {
          commandCode: 'ADD EUTRANNFREQ',
          operationType: 'ADD',
          parameters: {
            LTE_INTER_FREQ_DL_EARFCN: '{41390}',
          },
        },
      },
    ]);
  });
});
