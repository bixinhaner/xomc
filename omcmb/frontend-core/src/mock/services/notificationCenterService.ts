// 消息中心 Mock 服务（T-0157 C8）。
// VITE_USE_MOCK=true 时由 hook 切换到该实现，模拟用户消息中心的增删改查。

import type { PageRequest, PageResponse } from '../../types/pagination';
import type {
  NotificationCenterFilter,
  NotificationCenterItem,
} from '../../types/notificationCenter';
import { delay } from '../utils';

// 模拟一组消息（按时间倒序）—— 涵盖五态供 UI 视觉验证
let items: NotificationCenterItem[] = [
  {
    id: 'mock-001',
    userId: 'mock-user',
    type: 'task_complete',
    status: 'queued',
    priority: 'normal',
    title: '参数设置 · 设备 SN001 · 进行中 · 1 项',
    content: 'TAC = 4',
    link: '/device/detail/SN001?tab=quickSettings',
    sender: 'system',
    dedupKey: 'mock-task-001',
    isRead: false,
    createdAt: new Date(Date.now() - 30 * 1000).toISOString(),
  },
  {
    id: 'mock-002',
    userId: 'mock-user',
    type: 'task_complete',
    status: 'completed',
    priority: 'normal',
    title: '参数设置 · 设备 SN002 · 已完成 · 3 项',
    content: 'TAC = 4\nPCI = 5\nDLEarfcn = 55340',
    link: '/device/detail/SN002?tab=quickSettings',
    sender: 'system',
    dedupKey: 'mock-task-002',
    isRead: false,
    createdAt: new Date(Date.now() - 2 * 60 * 1000).toISOString(),
  },
  {
    id: 'mock-003',
    userId: 'mock-user',
    type: 'task_complete',
    status: 'failed',
    priority: 'high',
    title: '参数设置 · 设备 SN003 · 失败 · 2 项',
    content: 'TAC = 4\nPCI = 5\n\n错误：CPE 拒绝 SPV — invalid value for PCI',
    link: '/device/detail/SN003?tab=quickSettings',
    sender: 'system',
    dedupKey: 'mock-task-003',
    isRead: true,
    readAt: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
    createdAt: new Date(Date.now() - 10 * 60 * 1000).toISOString(),
  },
  {
    id: 'mock-004',
    userId: 'mock-user',
    type: 'task_complete',
    status: 'expired',
    priority: 'high',
    title: '设备重启 · 设备 SN004 · 超时',
    content: '错误：等待 CPE 应答超时 120s',
    link: '/device/detail/SN004?tab=quickSettings',
    sender: 'system',
    dedupKey: 'mock-task-004',
    isRead: true,
    createdAt: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
  },
];

export const notificationCenterService = {
  async list(
    params: PageRequest & { filter?: NotificationCenterFilter },
  ): Promise<PageResponse<NotificationCenterItem>> {
    await delay(50, 150);
    let filtered = items.slice();
    if (params.filter?.type) {
      filtered = filtered.filter((n) => n.type === params.filter!.type);
    }
    if (params.filter?.isRead !== undefined) {
      filtered = filtered.filter((n) => n.isRead === params.filter!.isRead);
    }
    filtered.sort((a, b) => (a.createdAt < b.createdAt ? 1 : -1));
    const total = filtered.length;
    const start = (params.page - 1) * params.pageSize;
    const page = filtered.slice(start, start + params.pageSize);
    return { items: page, total, page: params.page, pageSize: params.pageSize };
  },

  async getUnreadCount(): Promise<number> {
    await delay(20, 80);
    return items.filter((n) => !n.isRead).length;
  },

  async markRead(id: string): Promise<void> {
    await delay(30, 100);
    const idx = items.findIndex((n) => n.id === id);
    if (idx >= 0 && !items[idx].isRead) {
      items[idx] = { ...items[idx], isRead: true, readAt: new Date().toISOString() };
    }
  },

  async markAllRead(): Promise<void> {
    await delay(30, 100);
    const now = new Date().toISOString();
    items = items.map((n) => (n.isRead ? n : { ...n, isRead: true, readAt: now }));
  },

  async delete(id: string): Promise<void> {
    await delay(30, 100);
    items = items.filter((n) => n.id !== id);
  },

  async clearAll(): Promise<number> {
    await delay(50, 150);
    const n = items.length;
    items = [];
    return n;
  },

  async syncStale(): Promise<number> {
    await delay(30, 80);
    // mock: 假装把 queued/sent 升级到 completed
    let updated = 0;
    items = items.map((n) => {
      if (n.status === 'queued' || n.status === 'sent') {
        updated++;
        return { ...n, status: 'completed' as const, title: n.title.replace('· 进行中 ·', '· 已完成 ·') };
      }
      return n;
    });
    return updated;
  },
};
