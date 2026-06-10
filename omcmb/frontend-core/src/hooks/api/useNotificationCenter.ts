// 消息中心 hooks 套件（T-0157 C8）。
//
// 6 个 hook 与设计文档 §5.2 读取侧表格对齐：
//   useNotificationCenter        — list（铃铛打开时调）
//   useNotificationUnreadCount   — 每 10s 轮询 + 任意页面挂载即用
//   useMarkNotificationRead      — PUT /:id/read（点击单条）
//   useMarkAllNotificationsRead  — PUT /read-all（"全部已读"按钮）
//   useDeleteNotification        — DELETE /:id
//   useClearAllNotifications     — DELETE /notifications（"清空"按钮）

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { createApiSwitchWithMock } from '../../services/apiSwitch';
import { notificationCenterApi } from '../../services/api/notificationCenterApi';
import { notificationCenterService } from '../../mock/services/notificationCenterService';
import type {
  NotificationCenterFilter,
  NotificationCenterItem,
} from '../../types/notificationCenter';
import type { PageResponse } from '../../types/pagination';

const api = createApiSwitchWithMock(notificationCenterService, notificationCenterApi);

/** React Query key 前缀。所有消息中心相关 query 共用，便于一次性 invalidate */
export const notificationKeys = {
  all: ['notifications'] as const,
  list: (page: number, pageSize: number, filter?: NotificationCenterFilter) =>
    ['notifications', 'list', page, pageSize, filter] as const,
  unreadCount: () => ['notifications', 'unread-count'] as const,
};

/**
 * 拉取消息列表（按 createdAt 倒序）。默认 page=1, pageSize=20。
 *
 * refetchInterval: 5 秒自动刷新，让 Popover 打开期间用户能看到 task 状态升级
 * (queued → sent → completed/failed/expired) 而不需要手动操作。Popover 关闭时
 * NotificationCenter 组件 unmount → hook 自动停轮询，不占资源。
 */
export function useNotificationCenter(opts?: {
  page?: number;
  pageSize?: number;
  filter?: NotificationCenterFilter;
  enabled?: boolean;
  refetchIntervalMs?: number;
}) {
  const page = opts?.page ?? 1;
  const pageSize = opts?.pageSize ?? 20;
  return useQuery<PageResponse<NotificationCenterItem>>({
    queryKey: notificationKeys.list(page, pageSize, opts?.filter),
    queryFn: () =>
      api.list({
        page,
        pageSize,
        sortField: 'created_at',
        sortOrder: 'descend',
        filter: opts?.filter,
      }),
    enabled: opts?.enabled ?? true,
    refetchInterval: opts?.refetchIntervalMs ?? 5_000,
    refetchIntervalInBackground: false,
  });
}

/**
 * 未读计数。默认 10 秒轮询（§5.2 设计），用于 Header 铃铛 Badge。
 *
 * 设计上轮询 unread-count 单端点（< 100B）比 list 轻量得多，10k 用户 × 0.1 QPS = 1k QPS
 * 可接受；用户点开铃铛时再调 useNotificationCenter 拉列表。
 *
 * 性能（#15）：铃铛在所有页面常驻，标签页隐藏时无需继续轮询——
 * refetchIntervalInBackground: false 让浏览器后台标签暂停轮询定时器，省电省带宽；
 * 切回前台时 React Query 自动恢复轮询并立即刷新一次。
 */
export function useNotificationUnreadCount(opts?: { refetchIntervalMs?: number; enabled?: boolean }) {
  const interval = opts?.refetchIntervalMs ?? 10_000;
  return useQuery<number>({
    queryKey: notificationKeys.unreadCount(),
    queryFn: () => api.getUnreadCount(),
    refetchInterval: interval,
    refetchIntervalInBackground: false,
    enabled: opts?.enabled ?? true,
  });
}

export function useMarkNotificationRead() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.markRead(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: notificationKeys.all });
    },
  });
}

export function useMarkAllNotificationsRead() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api.markAllRead(),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: notificationKeys.all });
    },
  });
}

export function useDeleteNotification() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.delete(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: notificationKeys.all });
    },
  });
}

/** 一键清空当前用户全部消息，返回删除数（T-0157 C4 后端补的 DELETE /notifications） */
export function useClearAllNotifications() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api.clearAll(),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: notificationKeys.all });
    },
  });
}

/**
 * 反查 task 状态修正卡死消息（T-0157 stale sync）。
 * 用法：NotificationCenter Popover onOpenChange(true) 时调用一次。
 * onSuccess 时 invalidate notifications 让 UI 拉到最新状态。
 */
export function useSyncStaleNotifications() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api.syncStale(),
    onSuccess: (updated) => {
      if (updated > 0) {
        void qc.invalidateQueries({ queryKey: notificationKeys.all });
      }
    },
  });
}
