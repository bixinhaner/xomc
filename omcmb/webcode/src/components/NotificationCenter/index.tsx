// 消息中心 Popover 主体（T-0157 C9）。
//
// 视觉与交互（§4.2 + §4.4 + §4.5 + §5.5）：
//   - 360px 宽、最大 480px 高，超出滚动
//   - 头部：标题"消息中心" + 右侧"全部已读"/"清空" 两个文字按钮
//   - 列表：按 createdAt 倒序的 NotificationItem
//   - 空态："暂无消息"
//   - 状态图标 5 色（蓝旋转/绿√/红×/黄⏰/灰⊘）
//   - 未读条目左侧蓝色竖条 + 标题加粗
//   - 已读条目灰色显示
//   - 点击条目 markRead + navigate(link)
//
// 数据：useNotificationCenter list + 三个 mutation hook（mark-all-read / clearAll / delete 单条隐式）

import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button, Empty, Flex, Spin, theme, Tooltip, Typography, message } from 'antd';
import {
  CheckCircleFilled,
  ClockCircleFilled,
  CloseCircleFilled,
  StopFilled,
  SyncOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import 'dayjs/locale/zh-cn';

import { useT } from '@/hooks/useT';
import {
  useNotificationCenter,
  useMarkNotificationRead,
  useMarkAllNotificationsRead,
  useClearAllNotifications,
} from '@core/hooks/api/useNotificationCenter';
import {
  toUiState,
  type NotificationCenterItem,
  type NotificationCenterUiState,
} from '@core/types/notificationCenter';

dayjs.extend(relativeTime);

const { Text, Paragraph } = Typography;

interface Props {
  /** Popover 关闭回调（点击条目跳转后由父组件关闭） */
  onClose?: () => void;
}

export default function NotificationCenter({ onClose }: Props) {
  const navigate = useNavigate();
  const t = useT();
  const { token } = theme.useToken();
  const listQuery = useNotificationCenter({ page: 1, pageSize: 20 });
  const markRead = useMarkNotificationRead();
  const markAllRead = useMarkAllNotificationsRead();
  const clearAll = useClearAllNotifications();
  const [busy, setBusy] = useState(false);

  const items = listQuery.data?.items ?? [];
  const isLoading = listQuery.isLoading;

  const handleItemClick = async (item: NotificationCenterItem) => {
    if (!item.isRead) {
      try {
        await markRead.mutateAsync(item.id);
      } catch {
        // 静默 — 用户跳转更重要
      }
    }
    if (item.link) {
      navigate(item.link);
      onClose?.();
    }
  };

  const handleMarkAllRead = async () => {
    setBusy(true);
    try {
      await markAllRead.mutateAsync();
      message.success({ content: t('notificationCenter.markAllReadSuccess'), duration: 2 });
    } catch (e) {
      message.error({ content: t('notificationCenter.markAllReadFailed', { error: String(e) }), duration: 4 });
    } finally {
      setBusy(false);
    }
  };

  const handleClearAll = async () => {
    setBusy(true);
    try {
      const n = await clearAll.mutateAsync();
      message.success({ content: t('notificationCenter.clearSuccess', { count: n }), duration: 2 });
    } catch (e) {
      message.error({ content: t('notificationCenter.clearFailed', { error: String(e) }), duration: 4 });
    } finally {
      setBusy(false);
    }
  };

  return (
    <div style={{ width: 360, background: token.colorBgElevated, color: token.colorText }}>
      {/* 头部 */}
      <div
        style={{
          padding: '8px 12px',
          borderBottom: `1px solid ${token.colorBorderSecondary}`,
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}
      >
        <Text strong>{t('notificationCenter.title')}</Text>
        <div>
          <Button type="link" size="small" onClick={handleMarkAllRead} disabled={busy || items.length === 0}>
            {t('notificationCenter.markAllRead')}
          </Button>
          <Button type="link" size="small" danger onClick={handleClearAll} disabled={busy || items.length === 0}>
            {t('notificationCenter.clear')}
          </Button>
        </div>
      </div>

      {/* 列表 / 空态 */}
      <div style={{ maxHeight: 440, overflowY: 'auto' }}>
        {isLoading ? (
          <div style={{ padding: 24, textAlign: 'center' }}>
            <Spin />
          </div>
        ) : items.length === 0 ? (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('notificationCenter.empty')} style={{ padding: 32 }} />
        ) : (
          // antd6 List 已废弃：改用 Flex 纵向容器 + map，逐条沿用原 NotificationItem
          // 标记（item 间分隔线下沉到 NotificationItem 内的 borderBottom，与原 List
          // 默认 split 视觉一致）。
          <Flex vertical>
            {items.map((item, idx) => (
              <NotificationItem
                key={item.id}
                item={item}
                last={idx === items.length - 1}
                onClick={() => void handleItemClick(item)}
                unreadLabel={t('notificationCenter.unread')}
              />
            ))}
          </Flex>
        )}
      </div>

      {/* 底部：查看全部 → 通知中心整页的消息 tab */}
      <div
        style={{
          padding: '6px 12px',
          borderTop: `1px solid ${token.colorBorderSecondary}`,
          textAlign: 'center',
        }}
      >
        <Button
          type="link"
          size="small"
          onClick={() => {
            navigate('/notifications?tab=messages');
            onClose?.();
          }}
        >
          {t('notificationCenter.viewAll')}
        </Button>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// 单条消息条目
// ---------------------------------------------------------------------------

interface ItemProps {
  item: NotificationCenterItem;
  /** 是否末条 —— 末条不画底部分隔线（复刻原 List split 默认行为） */
  last: boolean;
  onClick: () => void;
  /** 未读小圆点的 aria-label（由父组件经 i18n 传入，#226） */
  unreadLabel: string;
}

function NotificationItem({ item, last, onClick, unreadLabel }: ItemProps) {
  const { token } = theme.useToken();
  const ui = toUiState(item.status);
  const { icon, color } = uiVisuals(ui);
  // 标题已含"设备 SN"（§4.5 文案规范），副标题只显示相对时间避免重复
  const subtitleDate = dayjs(item.createdAt);
  const subtitle = subtitleDate.locale('zh-cn').fromNow();

  // 视觉规则（强化已读/未读对比，跟随明/暗主题）：
  //   未读：colorBgElevated 底 + 左侧 4px 蓝竖条 + 标题加粗 colorText + 右侧蓝色小圆点
  //   已读：colorFillTertiary 底（比 quaternary 沉一档，对比更强）+ 标题 type=secondary
  //         + 整行 opacity 0.55 让用户一眼区分
  // 不再硬编码颜色，避免暗模式背景仍为白、字体被覆盖看不见。
  // antd6 List.Item 随 List 一并废弃：用 div 复刻，原内联样式照搬，并补一条底部
  // 分隔线还原 List split 默认（末条不画）。
  return (
    <div
      onClick={onClick}
      style={{
        cursor: 'pointer',
        padding: '12px 12px 12px 14px',
        position: 'relative',
        borderLeft: item.isRead ? 'none' : `4px solid ${token.colorPrimary}`,
        borderBottom: last ? 'none' : `1px solid ${token.colorSplit}`,
        background: item.isRead ? token.colorFillTertiary : token.colorBgElevated,
        opacity: item.isRead ? 0.55 : 1,
      }}
    >
      <div style={{ display: 'flex', gap: 10, width: '100%' }}>
        <div style={{ paddingTop: 2, fontSize: 18, color, flexShrink: 0 }}>{icon}</div>
        <div style={{ flex: 1, minWidth: 0 }}>
          {/* 标题完整显示 —— 长内容自动换行（含 break-all 处理 device SN 等连续字符） */}
          <Text
            strong={!item.isRead}
            type={item.isRead ? 'secondary' : undefined}
            style={{ display: 'block', whiteSpace: 'normal', wordBreak: 'break-all' }}
          >
            {item.title}
            {!item.isRead && (
              <span
                style={{
                  display: 'inline-block',
                  width: 8,
                  height: 8,
                  borderRadius: '50%',
                  background: token.colorPrimary,
                  marginLeft: 6,
                  verticalAlign: 'middle',
                }}
                aria-label={unreadLabel}
              />
            )}
          </Text>
          {item.content ? (
            <Paragraph
              type="secondary"
              style={{
                fontSize: 12,
                margin: '4px 0',
                whiteSpace: 'pre-line',
                wordBreak: 'break-all',
              }}
            >
              {item.content}
            </Paragraph>
          ) : null}
          <Tooltip title={subtitleDate.format('YYYY-MM-DD HH:mm:ss')}>
            <Text type="secondary" style={{ fontSize: 11 }}>
              {subtitle}
            </Text>
          </Tooltip>
        </div>
      </div>
    </div>
  );
}

// 5 态视觉（§4.4）
function uiVisuals(s: NotificationCenterUiState): { icon: React.ReactNode; color: string } {
  switch (s) {
    case 'in_progress':
      return { icon: <SyncOutlined spin />, color: '#1677ff' };
    case 'completed':
      return { icon: <CheckCircleFilled />, color: '#52c41a' };
    case 'failed':
      return { icon: <CloseCircleFilled />, color: '#ff4d4f' };
    case 'expired':
      return { icon: <ClockCircleFilled />, color: '#faad14' };
    case 'cancelled':
      return { icon: <StopFilled />, color: '#bfbfbf' };
  }
}
