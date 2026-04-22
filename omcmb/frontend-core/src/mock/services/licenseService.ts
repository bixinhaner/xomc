import type { PageRequest, PageResponse } from '../../types/pagination';
import type { License } from '../data/license';
import { mockLicenses } from '../data/license';
import { delay, paginate, generateId } from '../utils';

const licenses = [...mockLicenses];

export const licenseService = {
  async getList(
    params: { status?: string; licenseType?: string; deviceType?: string } & PageRequest
  ): Promise<PageResponse<License>> {
    await delay(100, 200);
    let filtered = [...licenses];
    if (params.status) filtered = filtered.filter((l) => l.status === params.status);
    if (params.licenseType) filtered = filtered.filter((l) => l.licenseType === params.licenseType);
    if (params.deviceType) filtered = filtered.filter((l) => l.deviceType.includes(params.deviceType!));
    return paginate(filtered, params.page, params.pageSize);
  },

  async getById(id: string): Promise<License | null> {
    await delay(80, 150);
    return licenses.find((l) => l.id === id) ?? null;
  },

  async activate(licenseCode: string): Promise<License> {
    await delay(1000, 2000);
    const idx = licenses.findIndex((l) => l.licenseCode === licenseCode);
    if (idx === -1) throw new Error(`License ${licenseCode} not found`);
    licenses[idx] = { ...licenses[idx], status: 'active' };
    return licenses[idx];
  },

  async revoke(id: string): Promise<void> {
    await delay(500, 1000);
    const idx = licenses.findIndex((l) => l.id === id);
    if (idx !== -1) {
      licenses[idx] = { ...licenses[idx], status: 'revoked' };
    }
  },

  async importLicense(data: Omit<License, 'id'>): Promise<License> {
    await delay(500, 1000);
    const newItem: License = {
      ...data,
      id: generateId('lic'),
    };
    licenses.push(newItem);
    return newItem;
  },

  async getSummary(): Promise<{ total: number; active: number; expired: number; pending: number; expiringSoon: number }> {
    await delay(80, 150);
    const now = new Date().toISOString();
    const thirtyDaysLater = new Date(Date.now() + 30 * 86400000).toISOString();
    return {
      total: licenses.length,
      active: licenses.filter((l) => l.status === 'active').length,
      expired: licenses.filter((l) => l.status === 'expired').length,
      pending: licenses.filter((l) => l.status === 'pending').length,
      expiringSoon: licenses.filter(
        (l) => l.status === 'active' && l.expiryDate && l.expiryDate > now && l.expiryDate <= thirtyDaysLater
      ).length,
    };
  },
};
