// 消息中心 API 客户端（T-0157 C8）。
//
// 后端契约（internal/notification handler.go）：
//   GET    /notifications                   list（含 page/pageSize/type/is_read 过滤）
//   GET    /notifications/unread-count
//   PUT    /notifications/:id/read
//   PUT    /notifications/read-all
//   DELETE /notifications/:id
//   DELETE /notifications                   一键清空（C4 新加）
//
// 注：http 客户端已配 axios 拦截器自动 snake_case ↔ camelCase 转换（services/http.ts），
// 但 list 响应里嵌套数组 items 是否被遍历转换需视拦截器实现 — 这里仍提供显式 mapper
// 保底，避免某些字段（如 user_id）未转换时 UI 拿不到。

import http from '../http';
import type { PageRequest, PageResponse } from '../../types/pagination';
import type {
  NotificationCenterFilter,
  NotificationCenterItem,
  NotificationCenterPriority,
  NotificationCenterStatus,
  NotificationCenterType,
} from '../../types/notificationCenter';

interface BackendNotification {
  id: string;
  user_id: string;
  type: NotificationCenterType;
  status: NotificationCenterStatus;
  priority: NotificationCenterPriority;
  title: string;
  content?: string;
  link?: string;
  sender: string;
  dedup_key?: string | null;
  is_read: boolean;
  read_at?: string | null;
  created_at: string;
}

function mapBackendNotification(n: BackendNotification): NotificationCenterItem {
  return {
    id: n.id,
    userId: n.user_id,
    type: n.type,
    status: n.status,
    priority: n.priority,
    title: n.title,
    content: n.content,
    link: n.link,
    sender: n.sender,
    dedupKey: n.dedup_key ?? undefined,
    isRead: n.is_read,
    readAt: n.read_at ?? undefined,
    createdAt: n.created_at,
  };
}

export interface NotificationCenterListParams extends PageRequest {
  filter?: NotificationCenterFilter;
}

interface ListQuery {
  page?: number;
  page_size?: number;
  sort_by?: string;
  sort_dir?: 'asc' | 'desc';
  type?: NotificationCenterType;
  is_read?: 'true' | 'false';
}

export const notificationCenterApi = {
  async list(params: NotificationCenterListParams): Promise<PageResponse<NotificationCenterItem>> {
    const q: ListQuery = {
      page: params.page,
      page_size: params.pageSize,
      sort_by: params.sortField,
      sort_dir: params.sortOrder === 'ascend' ? 'asc' : params.sortOrder === 'descend' ? 'desc' : undefined,
    };
    if (params.filter?.type) q.type = params.filter.type;
    if (params.filter?.isRead !== undefined) q.is_read = params.filter.isRead ? 'true' : 'false';

    const { data } = await http.get<{
      items: BackendNotification[];
      total: number;
      page: number;
      page_size: number;
    }>('/notifications', { params: q });

    return {
      items: (data.items ?? []).map(mapBackendNotification),
      total: data.total,
      page: data.page,
      pageSize: data.page_size,
    };
  },

  async getUnreadCount(): Promise<number> {
    const { data } = await http.get<{ count: number }>('/notifications/unread-count');
    return data.count;
  },

  async markRead(id: string): Promise<void> {
    await http.put(`/notifications/${id}/read`);
  },

  async markAllRead(): Promise<void> {
    await http.put('/notifications/read-all');
  },

  async delete(id: string): Promise<void> {
    await http.delete(`/notifications/${id}`);
  },

  /** 一键清空当前用户全部消息，返回删除数（T-0157 C4 后端补） */
  async clearAll(): Promise<number> {
    const { data } = await http.delete<{ deleted: number }>('/notifications');
    return data.deleted;
  },

  /**
   * 反查 task 状态修正卡死消息（T-0157 stale sync）。
   * 用于 Popover 打开时兜底 — 后端 subscriber 漏接事件导致的 queued/sent 消息会被同步到当前 task 状态。
   * 返回更新条数。
   */
  async syncStale(): Promise<number> {
    const { data } = await http.post<{ updated: number }>('/notifications/sync-stale');
    return data.updated;
  },
};
