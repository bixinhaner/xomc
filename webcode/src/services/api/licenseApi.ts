import http from '../http';
import type { License } from '@/mock/data/license';
import type { PageRequest, PageResponse } from '@/types/pagination';

// ---------------------------------------------------------------------------
// Backend response types
// ---------------------------------------------------------------------------

interface BackendLicense {
  id: string;
  license_name: string;
  license_code: string;
  product_name: string;
  license_type: string;
  status: string;
  max_devices: number;
  used_devices: number;
  features: string[];
  issue_date: string;
  expiry_date: string | null;
  licensor: string | null;
  device_type: string | null;
  region: string | null;
  notes: string | null;
  created_at: string;
  updated_at: string;
}

interface BackendLicenseSummary {
  total: number;
  active: number;
  expired: number;
  pending: number;
  expiring_soon: number;
}

interface BackendListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

// ---------------------------------------------------------------------------
// Mapping helpers: backend -> frontend
// ---------------------------------------------------------------------------

function mapBackendLicense(bl: BackendLicense): License {
  return {
    id: bl.id,
    licenseName: bl.license_name,
    licenseCode: bl.license_code,
    productName: bl.product_name,
    licenseType: bl.license_type as License['licenseType'],
    status: bl.status as License['status'],
    maxDevices: bl.max_devices,
    usedDevices: bl.used_devices,
    features: bl.features || [],
    issueDate: bl.issue_date,
    expiryDate: bl.expiry_date,
    licensor: bl.licensor || '',
    deviceType: bl.device_type || '',
    region: bl.region || '',
    notes: bl.notes || undefined,
  };
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

export const licenseApi = {
  async getLicenses(
    params: { status?: string; licenseType?: string; deviceType?: string } & PageRequest
  ): Promise<PageResponse<License>> {
    const query: Record<string, unknown> = {
      page: params.page,
      pageSize: params.pageSize,
    };
    if (params.status) query.status = params.status;
    if (params.licenseType) query.license_type = params.licenseType;
    if (params.deviceType) query.device_type = params.deviceType;

    const { data } = await http.get<BackendListResponse<BackendLicense>>(
      '/licenses',
      { params: query }
    );

    return {
      items: (data.items || []).map(mapBackendLicense),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getLicenseById(id: string): Promise<License | null> {
    try {
      const { data } = await http.get<BackendLicense>(`/licenses/${id}`);
      return mapBackendLicense(data);
    } catch {
      return null;
    }
  },

  async getLicenseSummary(): Promise<{
    total: number;
    active: number;
    expired: number;
    pending: number;
    expiringSoon: number;
  }> {
    const { data } = await http.get<BackendLicenseSummary>('/licenses/summary');
    return {
      total: data.total,
      active: data.active,
      expired: data.expired,
      pending: data.pending,
      expiringSoon: data.expiring_soon,
    };
  },

  async activateLicense(licenseCode: string): Promise<License> {
    const { data } = await http.post<BackendLicense>('/licenses/activate', {
      license_code: licenseCode,
    });
    return mapBackendLicense(data);
  },

  async revokeLicense(id: string): Promise<void> {
    await http.post(`/licenses/${id}/revoke`);
  },

  async importLicense(licenseData: Omit<License, 'id'>): Promise<License> {
    const payload = {
      license_name: licenseData.licenseName,
      license_code: licenseData.licenseCode,
      product_name: licenseData.productName,
      license_type: licenseData.licenseType,
      status: licenseData.status,
      max_devices: licenseData.maxDevices,
      used_devices: licenseData.usedDevices,
      features: licenseData.features,
      issue_date: licenseData.issueDate,
      expiry_date: licenseData.expiryDate,
      licensor: licenseData.licensor || null,
      device_type: licenseData.deviceType || null,
      region: licenseData.region || null,
      notes: licenseData.notes || null,
    };
    const { data } = await http.post<BackendLicense>('/licenses/import', payload);
    return mapBackendLicense(data);
  },
};
