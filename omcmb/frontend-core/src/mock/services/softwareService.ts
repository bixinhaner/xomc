import type { PageRequest, PageResponse } from '../../types/pagination';
import type { SoftwareVersion, UpgradePlan } from '../data/software';
import { mockSoftwareVersions, mockUpgradePlans } from '../data/software';
import { delay, paginate, generateId } from '../utils';

let versions = [...mockSoftwareVersions];
const upgradePlans = [...mockUpgradePlans];

export const softwareService = {
  async getVersions(
    params: { deviceType?: string; status?: string; vendor?: string } & PageRequest
  ): Promise<PageResponse<SoftwareVersion>> {
    await delay(100, 200);
    let filtered = [...versions];
    if (params.deviceType) filtered = filtered.filter((v) => v.deviceType === params.deviceType);
    if (params.status) filtered = filtered.filter((v) => v.status === params.status);
    if (params.vendor) filtered = filtered.filter((v) => v.vendor === params.vendor);
    return paginate(filtered, params.page, params.pageSize);
  },

  async getVersionById(id: string): Promise<SoftwareVersion | null> {
    await delay(80, 150);
    return versions.find((v) => v.id === id) ?? null;
  },

  async uploadVersion(data: Omit<SoftwareVersion, 'id' | 'releaseDate'>): Promise<SoftwareVersion> {
    await delay(500, 1500);
    const newItem: SoftwareVersion = {
      ...data,
      id: generateId('ver'),
      releaseDate: new Date().toISOString(),
    };
    versions.push(newItem);
    return newItem;
  },

  async deleteVersions(ids: string[]): Promise<void> {
    await delay(150, 300);
    versions = versions.filter((v) => !ids.includes(v.id));
  },

  async getUpgradePlans(
    params: { status?: string } & PageRequest
  ): Promise<PageResponse<UpgradePlan>> {
    await delay(100, 200);
    let filtered = [...upgradePlans];
    if (params.status) filtered = filtered.filter((p) => p.status === params.status);
    return paginate(filtered, params.page, params.pageSize);
  },

  async getUpgradePlanById(id: string): Promise<UpgradePlan | null> {
    await delay(80, 150);
    return upgradePlans.find((p) => p.id === id) ?? null;
  },

  async createUpgradePlan(data: Omit<UpgradePlan, 'id' | 'status' | 'progress' | 'successCount' | 'failCount' | 'createdAt' | 'updatedAt'>): Promise<UpgradePlan> {
    await delay(200, 400);
    const newItem: UpgradePlan = {
      ...data,
      id: generateId('upg'),
      status: data.scheduledTime ? 'scheduled' : 'pending',
      progress: 0,
      successCount: 0,
      failCount: 0,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
    upgradePlans.push(newItem);
    return newItem;
  },

  async cancelUpgradePlan(id: string): Promise<void> {
    await delay(200, 400);
    const idx = upgradePlans.findIndex((p) => p.id === id);
    if (idx !== -1) {
      upgradePlans[idx] = {
        ...upgradePlans[idx],
        status: 'cancelled',
        updatedAt: new Date().toISOString(),
      };
    }
  },

  async precheck(deviceSns: string[], versionId: string): Promise<Array<{ deviceSn: string; passed: boolean; issues: string[] }>> {
    await delay(1000, 3000);
    void versionId;
    return deviceSns.map((sn) => ({
      deviceSn: sn,
      passed: Math.random() > 0.15,
      issues: Math.random() > 0.8 ? ['磁盘空间不足，需要2GB', '当前版本不支持直升'] : [],
    }));
  },
};
