import { describe, expect, it } from 'vitest';
import { parseGpsObservationSource } from './deviceGpsObservation';

describe('parseGpsObservationSource', () => {
  it.each([
    ['Device.DeviceInfo.SAS.FAP.GPS', 1],
    ['Device.DeviceInfo.SAS.FAP.GPS.2', 2],
    ['Device.FAP.GPS.3', 3],
  ] as const)('keeps %s and resolves coordinate slot %s', (sourcePath, slot) => {
    expect(parseGpsObservationSource(sourcePath)).toEqual({ sourcePath, slot });
  });
});
