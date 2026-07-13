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

  it('parses standard PATH script rows into raw path command payloads', () => {
    const result = parseMmlScriptPlan(
      [
        'LST PATH:Device.IP.Interface.1.Enable;SN001',
        'MOD PATH:Device.IP.Interface.1.Enable=true,Device.IP.Interface.2.Enable=false;SN001',
        'ADD PATH:Device.IP.Interface.1.IPv4Address.:IPAddress=192.168.1.10,SubnetMask=255.255.255.0;SN001',
        'RMV PATH:Device.IP.Interface.1.IPv4Address.3.;SN001',
      ].join('\n'),
    );

    expect(result.executeMode).toBe('device_bound');
    expect(result.planItems).toHaveLength(4);
    expect(result.planItems[0].command).toMatchObject({
      commandCode: 'RAW LST',
      operationType: 'LST',
      paramPaths: ['Device.IP.Interface.1.Enable'],
    });
    expect(result.planItems[1].command).toMatchObject({
      commandCode: 'RAW MOD',
      operationType: 'MOD',
      paramPaths: ['Device.IP.Interface.1.Enable', 'Device.IP.Interface.2.Enable'],
      parameters: {
        'Device.IP.Interface.1.Enable': 'true',
        'Device.IP.Interface.2.Enable': 'false',
      },
    });
    expect(result.planItems[2].command).toMatchObject({
      commandCode: 'RAW ADD',
      operationType: 'ADD',
      paramPaths: ['Device.IP.Interface.1.IPv4Address.'],
      parameters: {
        IPAddress: '192.168.1.10',
        SubnetMask: '255.255.255.0',
      },
    });
    expect(result.planItems[3].command).toMatchObject({
      commandCode: 'RAW RMV',
      operationType: 'RMV',
      paramPaths: ['Device.IP.Interface.1.IPv4Address.3.'],
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
