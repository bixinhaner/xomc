import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { attentionApi } from '../../services/api/attentionApi';
import type { AttentionSectionKey } from '../../types/attention';

export const attentionKeys = {
  all: ['dashboard', 'attention'] as const,
  summary: () => [...attentionKeys.all, 'summary'] as const,
  drawer: (section: AttentionSectionKey, page: number, pageSize: number) =>
    [...attentionKeys.all, 'drawer', section, page, pageSize] as const,
};

export function useAttentionSummary() {
  return useQuery({
    queryKey: attentionKeys.summary(),
    queryFn: () => attentionApi.getSummary(1),
    staleTime: 15_000,
    refetchInterval: 30_000,
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: true,
    retry: 1,
  });
}

export function useAttentionPage(
  section: AttentionSectionKey,
  page: number,
  pageSize: number,
  enabled: boolean,
) {
  return useQuery({
    queryKey: attentionKeys.drawer(section, page, pageSize),
    queryFn: () => attentionApi.getPage(section, page, pageSize),
    enabled,
    placeholderData: keepPreviousData,
    staleTime: 15_000,
    retry: 1,
  });
}
