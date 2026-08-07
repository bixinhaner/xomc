import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  alarmEmailSettingsApi,
  notificationChannelApi,
  notificationDeliveryApi,
  statusSummaryConfigApi,
} from '../../services/api/notificationApi';
import type {
  AlarmEmailSettingPayload,
  NotificationDeliveryListParams,
  StatusSummaryConfigPayload,
} from '../../types/notification';

const channelKeys = ['notification-management', 'channels'] as const;
const deliveryKeys = ['notification-management', 'deliveries'] as const;
const alarmEmailSettingKeys = ['alarm-email-settings'] as const;
const alarmEmailDefaultsKey = [...alarmEmailSettingKeys, 'default-recipients'] as const;
const statusSummaryConfigKey = ['notification-management', 'status-summary-config'] as const;

export function useStatusSummaryConfig(enabled = true) {
  return useQuery({ queryKey: statusSummaryConfigKey, queryFn: statusSummaryConfigApi.get, enabled });
}

export function useUpdateStatusSummaryConfig() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ revision, payload }: { revision: number; payload: StatusSummaryConfigPayload }) =>
      statusSummaryConfigApi.update(revision, payload),
    onSuccess: (config) => client.setQueryData(statusSummaryConfigKey, config),
  });
}

export function useAlarmEmailSettings() {
  return useQuery({ queryKey: alarmEmailSettingKeys, queryFn: alarmEmailSettingsApi.list });
}

export function useAlarmEmailDefaults(enabled = true) {
  return useQuery({ queryKey: alarmEmailDefaultsKey, queryFn: alarmEmailSettingsApi.getDefaults, enabled });
}

export function useUpdateAlarmEmailDefaults() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ revision, recipients }: { revision: number; recipients: string[] }) =>
      alarmEmailSettingsApi.updateDefaults(revision, recipients),
    onSuccess: () => client.invalidateQueries({ queryKey: alarmEmailDefaultsKey }),
  });
}

export function useCreateAlarmEmailSetting() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (payload: AlarmEmailSettingPayload) => alarmEmailSettingsApi.create(payload),
    onSuccess: () => client.invalidateQueries({ queryKey: alarmEmailSettingKeys }),
  });
}

export function useUpdateAlarmEmailSetting() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ id, revision, payload }: { id: string; revision: number; payload: AlarmEmailSettingPayload }) =>
      alarmEmailSettingsApi.update(id, revision, payload),
    onSuccess: () => client.invalidateQueries({ queryKey: alarmEmailSettingKeys }),
  });
}

export function useArchiveAlarmEmailSetting() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ id, revision }: { id: string; revision: number }) => alarmEmailSettingsApi.archive(id, revision),
    onSuccess: () => client.invalidateQueries({ queryKey: alarmEmailSettingKeys }),
  });
}

export function useNotificationChannels() {
  return useQuery({ queryKey: channelKeys, queryFn: notificationChannelApi.list });
}

export function useNotificationChannelHealth(id: string) {
  return useQuery({
    queryKey: [...channelKeys, id, 'health'],
    queryFn: () => notificationChannelApi.health(id),
    enabled: Boolean(id),
    refetchInterval: 30000,
  });
}

export function useVerifyNotificationChannel() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: notificationChannelApi.verify,
    onSuccess: (_result, id) => client.invalidateQueries({ queryKey: [...channelKeys, id, 'health'] }),
  });
}

export function useNotificationDeliveries(params: NotificationDeliveryListParams) {
  return useQuery({
    queryKey: [...deliveryKeys, params],
    queryFn: () => notificationDeliveryApi.list(params),
    refetchInterval: 30000,
  });
}

export function useNotificationDelivery(id: string) {
  return useQuery({
    queryKey: [...deliveryKeys, id],
    queryFn: () => notificationDeliveryApi.get(id),
    enabled: Boolean(id),
  });
}

export function useNotificationDeliveryAttempts(id: string) {
  return useQuery({
    queryKey: [...deliveryKeys, id, 'attempts'],
    queryFn: () => notificationDeliveryApi.attempts(id),
    enabled: Boolean(id),
  });
}

export function useRetryNotificationDelivery() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) => notificationDeliveryApi.retry(id, reason),
    onSuccess: () => client.invalidateQueries({ queryKey: deliveryKeys }),
  });
}
