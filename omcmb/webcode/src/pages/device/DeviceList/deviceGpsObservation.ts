export interface GpsObservationSource {
  sourcePath: string;
  slot: 1 | 2 | 3;
}

export function parseGpsObservationSource(sourcePath: string): GpsObservationSource {
  const normalized = sourcePath.trim();
  const slot = normalized.endsWith('.3') ? 3 : normalized.endsWith('.2') ? 2 : 1;
  return { sourcePath: normalized, slot };
}
