import { describe, expect, it } from 'vitest';
import type { AntennaSector } from '@core/types/map';
import {
  formatAntennaCoverageRange,
  MIN_VISIBLE_SECTOR_WIDTH_PX,
  pixelDistance,
  resolveAntennaSectorRenderMode,
} from './antennaSectorRender';

const sector = (overrides: Partial<AntennaSector> = {}): AntennaSector => ({
  number: 1,
  azimuth: 120,
  antennaHeight: 27,
  mechanicalDowntilt: 1,
  horizontalBeamwidth: 0.01,
  verticalBeamwidth: 1,
  nearRadiusMeters: 1031,
  farRadiusMeters: 3094,
  fieldSources: {},
  directionAvailable: true,
  coverageAvailable: true,
  coverageStatus: 'available',
  missingFields: [],
  ...overrides,
});

describe('antennaSectorRender', () => {
  it('按屏幕像素宽度识别窄波瓣', () => {
    expect(resolveAntennaSectorRenderMode(sector(), [10, 10], [12, 10])).toBe('narrow');
    expect(resolveAntennaSectorRenderMode(
      sector(),
      [10, 10],
      [10 + MIN_VISIBLE_SECTOR_WIDTH_PX, 10],
    )).toBe('polygon');
  });

  it('覆盖不可用时不进入窄波瓣模式', () => {
    expect(resolveAntennaSectorRenderMode(
      sector({ coverageAvailable: false, coverageStatus: 'invalid_geometry' }),
      [10, 10],
      [11, 10],
    )).toBe('unavailable');
  });

  it('全向波束两端重合时仍按覆盖面渲染', () => {
    expect(resolveAntennaSectorRenderMode(
      sector({ horizontalBeamwidth: 360 }),
      [10, 10],
      [10, 10],
    )).toBe('polygon');
  });

  it('格式化真实近远端覆盖距离', () => {
    expect(formatAntennaCoverageRange(sector())).toBe('1031–3094 m');
    expect(pixelDistance([0, 0], [3, 4])).toBe(5);
  });
});
