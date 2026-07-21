import type { LocationSync } from '@core/types/device';

/** The yellow warning represents a real, actionable coordinate difference. */
export function shouldShowLocationSyncIndicator(
  sync: Pick<LocationSync, 'status' | 'reported'> | null | undefined,
): boolean {
  return sync?.status === 'pending' && sync.reported != null;
}
