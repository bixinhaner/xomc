import { describe, expect, it } from 'vitest';
import { mergeMmlTaskCsvTexts } from '../mmlCsvMerge';

describe('mergeMmlTaskCsvTexts', () => {
  it('renumbers command/device blocks continuously across task CSV files', () => {
    const header = '序号,设备SN,命令,操作类型,故障信息\r\n';
    const first = `\uFEFF${header}1,SN-1,MOD A,MOD,"line 1\nline 2"\r\n,,,,\r\n`;
    const second = `\uFEFF${header}1,SN-1,LST B,LST,\r\n,,,,\r\n1,SN-2,MOD C,MOD,\r\n`;
    const result = mergeMmlTaskCsvTexts([first, second]);
    expect(result).toContain('\r\n1,SN-1,MOD A,MOD,"line 1\nline 2"\r\n');
    expect(result).toContain('\r\n2,SN-1,LST B,LST,\r\n');
    expect(result).toContain('\r\n3,SN-2,MOD C,MOD,\r\n');
    expect(result.match(/序号,设备SN/g)).toHaveLength(1);
  });
});
