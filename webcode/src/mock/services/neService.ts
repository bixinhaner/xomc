import type { NE } from '@/types/device';
import type { PageRequest, PageResponse } from '@/types/pagination';
import { mockNEs } from '../data/nes';
import { delay, paginate, sortBy, filterByText } from '../utils';

export const neService = {
  async getList(
    params: { keyword?: string; neType?: string; connStatus?: string; region?: string } & PageRequest
  ): Promise<PageResponse<NE>> {
    await delay(100, 250);
    let filtered = [...mockNEs];

    if (params.keyword) {
      filtered = filtered.filter(
        (n) =>
          n.neName.includes(params.keyword!) ||
          n.sn.includes(params.keyword!)
      );
    }
    if (params.neType) filtered = filtered.filter((n) => n.neType === params.neType);
    if (params.connStatus) filtered = filtered.filter((n) => n.connStatus === params.connStatus);
    if (params.region) filtered = filtered.filter((n) => n.region === params.region);

    if (params.sortField) {
      filtered = sortBy(filtered, params.sortField as keyof NE, params.sortOrder ?? 'ascend');
    }

    return paginate(filtered, params.page, params.pageSize);
  },

  async getById(id: string): Promise<NE | null> {
    await delay(80, 150);
    return mockNEs.find((n) => n.id === id) ?? null;
  },

  async getBySn(sn: string): Promise<NE | null> {
    await delay(80, 150);
    return mockNEs.find((n) => n.sn === sn) ?? null;
  },

  async searchByName(keyword: string): Promise<NE[]> {
    await delay(80, 150);
    return filterByText(mockNEs, 'neName', keyword).slice(0, 20);
  },
};
