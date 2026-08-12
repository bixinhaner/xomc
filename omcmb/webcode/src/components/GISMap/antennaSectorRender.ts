import type { AntennaSector } from '@core/types/map';

export const MIN_VISIBLE_SECTOR_WIDTH_PX = 6;
export const DIRECTION_INDICATOR_LENGTH_PX = 32;

export type AntennaSectorRenderMode = 'unavailable' | 'narrow' | 'polygon';

type Pixel = [number, number];

export function pixelDistance(start: Pixel, end: Pixel): number {
  return Math.hypot(end[0] - start[0], end[1] - start[1]);
}

export function resolveAntennaSectorRenderMode(
  sector: AntennaSector,
  outerStartPixel?: Pixel,
  outerEndPixel?: Pixel,
): AntennaSectorRenderMode {
  if (!sector.coverageAvailable) return 'unavailable';
  // 对大于等于半圆的宽波束，弧线两端之间的弦长不再能代表覆盖宽度；
  // 特别是 360° 全向覆盖的两端重合，不能因此误判为窄波束。
  if ((sector.horizontalBeamwidth ?? 0) >= 180) return 'polygon';
  if (!outerStartPixel || !outerEndPixel) return 'polygon';
  return pixelDistance(outerStartPixel, outerEndPixel) < MIN_VISIBLE_SECTOR_WIDTH_PX
    ? 'narrow'
    : 'polygon';
}

export function formatAntennaCoverageRange(sector: AntennaSector): string | undefined {
  if (sector.nearRadiusMeters === undefined || sector.farRadiusMeters === undefined) return undefined;
  return `${Math.round(sector.nearRadiusMeters)}–${Math.round(sector.farRadiusMeters)} m`;
}
