import { useQuery } from '@tanstack/react-query';
import type { PageRequest } from '@/types/pagination';
import { useMock } from '@/services/apiSwitch';
import { deviceApi } from '@/services/api/deviceApi';
import { neService } from '@/mock/services/neService';

export function useNEList(
  params: { keyword?: string; neType?: string; connStatus?: string; region?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['nes', 'list', params],
    queryFn: () => useMock ? neService.getList(params) : deviceApi.getNEList(params),
  });
}

export function useNEById(id: string) {
  return useQuery({
    queryKey: ['nes', 'detail', id],
    queryFn: () => useMock ? neService.getById(id) : deviceApi.getById(id),
    enabled: Boolean(id),
  });
}

export function useNEBySn(sn: string) {
  return useQuery({
    queryKey: ['nes', 'sn', sn],
    queryFn: () => useMock ? neService.getBySn(sn) : deviceApi.getNEBySn(sn),
    enabled: Boolean(sn),
  });
}

export function useNESearch(keyword: string) {
  return useQuery({
    queryKey: ['nes', 'search', keyword],
    queryFn: () =>
      useMock
        ? neService.searchByName(keyword)
        : deviceApi.getNEList({ keyword, page: 1, pageSize: 20 }).then((r) => r.items),
    enabled: keyword.length >= 2,
  });
}
