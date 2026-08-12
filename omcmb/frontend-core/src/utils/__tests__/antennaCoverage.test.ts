import { describe, expect, it } from 'vitest';
import type { AntennaSector } from '../../types/map';
import { calculateAntennaCoverage } from '../antennaCoverage';

const sector = (overrides: Partial<AntennaSector> = {}): AntennaSector => ({
  number: 1,
  azimuth: 120,
  antennaHeight: 18,
  mechanicalDowntilt: 6,
  horizontalBeamwidth: 65,
  verticalBeamwidth: 8,
  fieldSources: {},
  directionAvailable: false,
  coverageAvailable: false,
  coverageStatus: 'incomplete',
  missingFields: [],
  ...overrides,
});

describe('calculateAntennaCoverage', () => {
  it('参数补齐后重新计算覆盖半径', () => {
    const result = calculateAntennaCoverage(sector());
    expect(result.directionAvailable).toBe(true);
    expect(result.coverageAvailable).toBe(true);
    expect(result.nearRadiusMeters).toBeCloseTo(102.08, 2);
    expect(result.farRadiusMeters).toBeCloseTo(515.45, 2);
  });

  it('保留合法的零度方位角', () => {
    const result = calculateAntennaCoverage(sector({ azimuth: 0 }));
    expect(result.directionAvailable).toBe(true);
    expect(result.azimuth).toBe(0);
  });

  it('几何条件不成立时不生成伪覆盖', () => {
    const result = calculateAntennaCoverage(sector({ mechanicalDowntilt: 2 }));
    expect(result.coverageAvailable).toBe(false);
    expect(result.coverageStatus).toBe('invalid_geometry');
    expect(result.coverageIssue).toBe('far_angle_not_positive');
    expect(result.missingFields).toEqual([]);
  });

  it('极窄水平波宽仍保留真实覆盖计算', () => {
    const result = calculateAntennaCoverage(sector({
      antennaHeight: 27,
      mechanicalDowntilt: 1,
      horizontalBeamwidth: 0.01,
      verticalBeamwidth: 1,
    }));

    expect(result.coverageAvailable).toBe(true);
    expect(result.coverageStatus).toBe('available');
    expect(result.horizontalBeamwidth).toBe(0.01);
  });
});
