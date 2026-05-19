// 消息中心类型（T-0157 C8）。后端 internal/notification.Notification 模型镜像。
// 与设计文档 §4.4 五态文案分层对齐 + §6 数据模型（含 status / dedup_key 扩展）。
//
// 注：本文件**不复用** types/notification.ts（那是 SMTP 模板 / 历史记录领域），
// 二者后端是同一个 internal/notification 包但服务两条独立链路。

/** 消息类型 — V1 仅 task_complete 落地，其他类型预留 */
export type NotificationCenterType =
  | 'alarm'
  | 'task_complete'
  | 'system'
  | 'approval'
  | 'device_status';

/** 内部状态 — 与后端 NotificationStatus 一一对应 */
export type NotificationCenterStatus =
  | 'queued'
  | 'sent'
  | 'completed'
  | 'failed'
  | 'expired'
  | 'cancelled';

/** 优先级 — failed/expired 升 high，其他 normal */
export type NotificationCenterPriority = 'critical' | 'high' | 'normal' | 'low';

/** UI 五态文案分层（§4.4）— `queued+sent` 聚合为"进行中" */
export type NotificationCenterUiState =
  | 'in_progress'
  | 'completed'
  | 'failed'
  | 'expired'
  | 'cancelled';

/** 把内部 6 态映射到 UI 5 态（§4.4 表格）*/
export function toUiState(s: NotificationCenterStatus): NotificationCenterUiState {
  if (s === 'queued' || s === 'sent') return 'in_progress';
  return s;
}

export interface NotificationCenterItem {
  id: string;
  userId: string;
  type: NotificationCenterType;
  status: NotificationCenterStatus;
  priority: NotificationCenterPriority;
  title: string;
  content?: string;
  link?: string;
  sender: string;
  /** task_complete 类型的去重键 = task uuid；非 task 类为 undefined */
  dedupKey?: string;
  isRead: boolean;
  readAt?: string;
  createdAt: string;
}

/** GET /notifications 过滤参数 */
export interface NotificationCenterFilter {
  type?: NotificationCenterType;
  isRead?: boolean;
}
