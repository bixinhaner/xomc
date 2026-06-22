import { describe, it, expect } from 'vitest';
import {
  parseCellId,
  parsePlmn,
  parseGnbId,
  parseNrCgi,
  parseCuId,
  parseDuId,
  parsePlmnId,
  parseNssai,
  parseSliceGroup,
  parseUid,
  parseObjectLdn,
  formatObjectLdn,
  deviceSnTail,
  buildDeviceSeriesName,
} from '../pmObject';

describe('pmObject — 4G LTE 解析', () => {
  it('parseCellId 取 Cellid 段（大小写不敏感）', () => {
    expect(parseCellId('Cellid=111172245,PLMN=46068')).toBe('111172245');
    expect(parseCellId('CELLID=999,PLMN=1')).toBe('999');
    expect(parseCellId('PLMN=46068')).toBeUndefined();
    expect(parseCellId('')).toBeUndefined();
    expect(parseCellId(null)).toBeUndefined();
  });

  it('parsePlmn 取 PLMN 段（不与 PLMNID 互窜）', () => {
    expect(parsePlmn('Cellid=111172245,PLMN=46068')).toBe('46068');
    expect(parsePlmn('Cellid=999')).toBeUndefined();
    expect(parsePlmn(undefined)).toBeUndefined();
    // 单独出现 PLMNID（5G 段）时 PLMN 不该误命中
    expect(parsePlmn('Type=Cell,gNBID=1,NrCGI=2,CUID=3,PLMNID=00101')).toBeUndefined();
  });
});

describe('pmObject — 5G NR 解析', () => {
  it('parseGnbId / parseNrCgi / parseCuId / parseDuId', () => {
    const ldn = 'Type=Cell,Mode=SA,gNBID=350251605,NrCGI=15153,CUID=1';
    expect(parseGnbId(ldn)).toBe('350251605');
    expect(parseNrCgi(ldn)).toBe('15153');
    expect(parseCuId(ldn)).toBe('1');
    expect(parseDuId(ldn)).toBeUndefined();

    const duLdn = 'Type=Cell,Mode=SA,gNBID=350251605,NrCGI=15153,DUID=2';
    expect(parseDuId(duLdn)).toBe('2');
  });

  it('parsePlmnId 取 PLMNID（与 PLMN 区分）', () => {
    const ldn = 'Type=Cell,gNBID=1,NrCGI=2,CUID=3,PLMNID=00101';
    expect(parsePlmnId(ldn)).toBe('00101');
    // 4G PLMN 字段不命中 PLMNID
    expect(parsePlmnId('Cellid=66,PLMN=46001')).toBeUndefined();
  });

  it('parseNssai 含斜杠', () => {
    const ldn = 'Type=Cell,gNBID=1,NrCGI=2,CUID=3,NSSAI=0/1/2';
    expect(parseNssai(ldn)).toBe('0/1/2');
  });

  it('parseSliceGroup 含斜杠', () => {
    const ldn = 'Type=Cell,gNBID=1,NrCGI=2,CUID=3,SCLICEGROUP=0/1/2';
    expect(parseSliceGroup(ldn)).toBe('0/1/2');
  });
});

describe('pmObject — GSM 解析', () => {
  it('parseUid 含连字符', () => {
    expect(parseUid('Uid=4002-1')).toBe('4002-1');
    expect(parseUid('Uid=1110-101')).toBe('1110-101');
    expect(parseUid('Cellid=66')).toBeUndefined();
  });
});

describe('pmObject — parseObjectLdn 制式判定', () => {
  it('4G Cellid → tech=lte', () => {
    const f = parseObjectLdn('Cellid=66,PLMN=46001');
    expect(f.tech).toBe('lte');
    expect(f.cellId).toBe('66');
    expect(f.plmn).toBe('46001');
    expect(f.gnbId).toBeUndefined();
    expect(f.uid).toBeUndefined();
  });

  it('5G gNBID → tech=nr，cellId/plmn 留空（避免与 LTE 语义混淆）', () => {
    const f = parseObjectLdn('Type=Cell,gNBID=350251605,NrCGI=15153,CUID=1,PLMNID=00101');
    expect(f.tech).toBe('nr');
    expect(f.cellId).toBeUndefined();
    expect(f.plmn).toBeUndefined();
    expect(f.gnbId).toBe('350251605');
    expect(f.nrCgi).toBe('15153');
    expect(f.cuId).toBe('1');
    expect(f.plmnId).toBe('00101');
  });

  it('GSM Uid → tech=gsm', () => {
    const f = parseObjectLdn('Uid=4002-1');
    expect(f.tech).toBe('gsm');
    expect(f.uid).toBe('4002-1');
    expect(f.cellId).toBeUndefined();
    expect(f.plmn).toBeUndefined();
  });

  it('无法识别 → tech=unknown', () => {
    expect(parseObjectLdn('SomeUnknownThing').tech).toBe('unknown');
    expect(parseObjectLdn('').tech).toBe('unknown');
    expect(parseObjectLdn(null).tech).toBe('unknown');
  });
});

describe('pmObject — formatObjectLdn 三制式友好名', () => {
  // ===== 4G =====
  it('4G 两段都有 → 「小区X · PLMNY」', () => {
    expect(formatObjectLdn('Cellid=111172245,PLMN=46068')).toBe('小区111172245 · PLMN46068');
  });

  it('4G 只有小区 → 「小区X」', () => {
    expect(formatObjectLdn('Cellid=111172245')).toBe('小区111172245');
  });

  it('4G 只有 PLMN（无 Cellid）→ 「PLMNY」', () => {
    expect(formatObjectLdn('PLMN=46068')).toBe('PLMN46068');
  });

  // ===== 5G =====
  it('5G 全字段 → 含 gNB / 小区 / PLMN 段', () => {
    const got = formatObjectLdn(
      'Type=Cell,Mode=SA,gNBID=350251605,NrCGI=15153,CUID=1,PLMNID=00101',
    );
    expect(got).toContain('gNB');
    expect(got).toContain('小区');
    expect(got).toContain('PLMN');
    expect(got).toBe('gNB350251605 · 小区15153/1 · PLMN00101');
  });

  it('5G 仅 gNBID（设备级）→ 「gNB{id}」', () => {
    expect(formatObjectLdn('Type=gNB,Mode=SA,gNBID=350251605')).toBe('gNB350251605');
  });

  it('5G CU 小区级 无 PLMN → 「gNB · 小区/CU」', () => {
    expect(formatObjectLdn('Type=Cell,Mode=SA,gNBID=350251605,NrCGI=15153,CUID=1')).toBe(
      'gNB350251605 · 小区15153/1',
    );
  });

  it('5G DU 小区级 → 用 DUID 作小区后缀', () => {
    expect(formatObjectLdn('Type=Cell,Mode=SA,gNBID=350251605,NrCGI=15153,DUID=2')).toBe(
      'gNB350251605 · 小区15153/2',
    );
  });

  it('5G NSSAI 段降级追加切片', () => {
    expect(
      formatObjectLdn('Type=Cell,gNBID=1,NrCGI=2,CUID=3,NSSAI=0/1/2'),
    ).toContain('切片0/1/2');
  });

  it('5G SCLICEGROUP 段追加切片组', () => {
    expect(
      formatObjectLdn('Type=Cell,gNBID=1,NrCGI=2,CUID=3,SCLICEGROUP=0/1/2'),
    ).toContain('切片组0/1/2');
  });

  // ===== GSM =====
  it('GSM Uid → 「小区 Uid{val}」', () => {
    expect(formatObjectLdn('Uid=4002-1')).toBe('小区 Uid4002-1');
    expect(formatObjectLdn('Uid=4002-1')).toContain('小区');
    expect(formatObjectLdn('Uid=4002-1')).toContain('Uid');
  });

  it('GSM 真机大号', () => {
    expect(formatObjectLdn('Uid=1110-101')).toBe('小区 Uid1110-101');
  });

  // ===== 回退 =====
  it('无法拆解 → 回退原串', () => {
    expect(formatObjectLdn('SomeOtherLdn')).toBe('SomeOtherLdn');
  });

  it('空 / null → 空串', () => {
    expect(formatObjectLdn('')).toBe('');
    expect(formatObjectLdn(null)).toBe('');
    expect(formatObjectLdn(undefined)).toBe('');
  });
});

describe('pmObject — deviceSnTail / buildDeviceSeriesName', () => {
  it('deviceSnTail 取末 6 位，短于 6 全量', () => {
    expect(deviceSnTail('1202000240194DP0015')).toBe('DP0015');
    expect(deviceSnTail('SN-A')).toBe('SN-A');
    expect(deviceSnTail('')).toBe('');
    expect(deviceSnTail(null)).toBe('');
  });

  it('buildDeviceSeriesName 有小区 → 尾号 · 友好名', () => {
    expect(buildDeviceSeriesName('SN-AAAAAA', 'Cellid=111,PLMN=222')).toBe(
      'AAAAAA · 小区111 · PLMN222',
    );
  });

  it('buildDeviceSeriesName 5G → 尾号 · 5G 友好名', () => {
    expect(
      buildDeviceSeriesName('SN-AAAAAA', 'Type=Cell,gNBID=1,NrCGI=2,CUID=3'),
    ).toBe('AAAAAA · gNB1 · 小区2/3');
  });

  it('buildDeviceSeriesName GSM → 尾号 · GSM 友好名', () => {
    expect(buildDeviceSeriesName('SN-AAAAAA', 'Uid=4002-1')).toBe('AAAAAA · 小区 Uid4002-1');
  });

  it('buildDeviceSeriesName 无小区 → 仅尾号（兜底单线）', () => {
    expect(buildDeviceSeriesName('SN-AAAAAA', null)).toBe('AAAAAA');
    expect(buildDeviceSeriesName('SN-AAAAAA', '')).toBe('AAAAAA');
  });
});
