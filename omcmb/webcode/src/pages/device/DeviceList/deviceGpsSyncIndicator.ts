/** Keep a discoverable GPS action beside every concrete location value. */
export function shouldShowLocationSyncIndicator(value: number | null | undefined): boolean {
  return value != null;
}
