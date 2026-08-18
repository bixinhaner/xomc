export interface PlugAndPlayLicenseListParams {
  page: number;
  pageSize: number;
  serialNumber: string | undefined;
}

/**
 * Plug-and-play licenses are pre-provisioned by device SN. Product metadata
 * may not exist until the device first registers, so it must never constrain
 * this list.
 */
export function buildPlugAndPlayLicenseListParams(
  serialNumber: string,
): PlugAndPlayLicenseListParams {
  return {
    page: 1,
    pageSize: 1000,
    serialNumber: serialNumber || undefined,
  };
}
