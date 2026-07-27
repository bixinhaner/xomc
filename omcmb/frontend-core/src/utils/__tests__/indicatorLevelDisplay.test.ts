import { describe, expect, it } from 'vitest';
import { formatIndicatorLevel, shouldShowIndicatorLevel } from '../indicatorLevelDisplay';

describe('indicatorLevelDisplay', () => {
  it('按指标级别码显示中文文案，空值和未知值显示 -', () => {
    const t = (id: string) => ({
      'perf.query.indicatorLevelDevice': '设备级',
      'perf.query.indicatorLevelPlmn': 'PLMN级',
      'perf.query.indicatorLevelBoth': '设备级 / PLMN级',
    })[id] ?? id;

    expect(formatIndicatorLevel('device', t)).toBe('设备级');
    expect(formatIndicatorLevel('plmn', t)).toBe('PLMN级');
    expect(formatIndicatorLevel('both', t)).toBe('设备级 / PLMN级');
    expect(formatIndicatorLevel('', t)).toBe('-');
    expect(formatIndicatorLevel(undefined, t)).toBe('-');
    expect(formatIndicatorLevel('cell', t)).toBe('-');
  });

  it('只在 ENB/GSM 指标列表显示指标级别，GNB 暂不显示', () => {
    expect(shouldShowIndicatorLevel('ENB')).toBe(true);
    expect(shouldShowIndicatorLevel('GSM')).toBe(true);
    expect(shouldShowIndicatorLevel('GNB')).toBe(false);
  });
});
