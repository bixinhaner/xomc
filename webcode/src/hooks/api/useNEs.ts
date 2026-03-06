import { useQuery } from '@tanstack/react-query';
import type { PageRequest } from '@/types/pagination';
import { neService } from '@/mock/services/neService';

export function useNEList(
  params: { keyword?: string; neType?: string; connStatus?: string; region?: string } & PageRequest
) {
  return useQuery({
    queryKey: ['nes', 'list', params],
    queryFn: () => neService.getList(params),
  });
}

export function useNEById(id: string) {
  return useQuery({
    queryKey: ['nes', 'detail', id],
    queryFn: () => neService.getById(id),
    enabled: Boolean(id),
  });
}

export function useNEBySn(sn: string) {
  return useQuery({
    queryKey: ['nes', 'sn', sn],
    queryFn: () => neService.getBySn(sn),
    enabled: Boolean(sn),
  });
}

export function useNESearch(keyword: string) {
  return useQuery({
    queryKey: ['nes', 'search', keyword],
    queryFn: () => neService.searchByName(keyword),
    enabled: keyword.length >= 2,
  });
}
