import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  notificationTemplateApi,
  notificationHistoryApi,
  emailRunHistoryApi,
} from '../../services/api/notificationApi';
import type {
  NotificationTemplateListParams,
  NotificationTemplateCreatePayload,
  NotificationTemplateUpdatePayload,
  NotificationHistoryListParams,
  EmailRunHistoryListParams,
  EmailRunBusinessType,
} from '../../types/notification';

// ---------------------------------------------------------------------------
// 通知模板 hooks
// ---------------------------------------------------------------------------

export function useNotificationTemplates(params: NotificationTemplateListParams) {
  return useQuery({
    queryKey: ['notifications', 'templates', 'list', params],
    queryFn: () => notificationTemplateApi.list(params),
  });
}

export function useEmailRunHistory(params: EmailRunHistoryListParams) {
  return useQuery({
    queryKey: ['notifications', 'email-runs', 'list', params],
    queryFn: () => emailRunHistoryApi.list(params),
    refetchInterval: 30000,
  });
}

export function useEmailRunDeliveries(
  businessType: EmailRunBusinessType | undefined,
  runId: string | undefined
) {
  return useQuery({
    queryKey: ['notifications', 'email-runs', businessType, runId, 'deliveries'],
    queryFn: () => emailRunHistoryApi.deliveries(businessType!, runId!),
    enabled: Boolean(businessType && runId),
  });
}

export function useNotificationTemplate(id: string) {
  return useQuery({
    queryKey: ['notifications', 'templates', 'detail', id],
    queryFn: () => notificationTemplateApi.get(id),
    enabled: Boolean(id),
  });
}

export function useCreateNotificationTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: NotificationTemplateCreatePayload) =>
      notificationTemplateApi.create(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ['notifications', 'templates'],
      });
    },
  });
}

export function useUpdateNotificationTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      payload,
    }: {
      id: string;
      payload: NotificationTemplateUpdatePayload;
    }) => notificationTemplateApi.update(id, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ['notifications', 'templates'],
      });
    },
  });
}

export function useDeleteNotificationTemplate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => notificationTemplateApi.delete(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ['notifications', 'templates'],
      });
    },
  });
}

// ---------------------------------------------------------------------------
// 通知历史 hooks
// ---------------------------------------------------------------------------

export function useNotificationHistory(params: NotificationHistoryListParams) {
  return useQuery({
    queryKey: ['notifications', 'history', 'list', params],
    queryFn: () => notificationHistoryApi.list(params),
    refetchInterval: 30000,
  });
}

export function useNotificationHistoryDetail(id: string) {
  return useQuery({
    queryKey: ['notifications', 'history', 'detail', id],
    queryFn: () => notificationHistoryApi.get(id),
    enabled: Boolean(id),
  });
}
