import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import { Tabs } from 'antd';
import { useTabStore } from '@core/store/tabStore';
import { useUserStore } from '@core/store/userStore';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';
import MessageList from './MessageList';
import ChannelHealth from './ChannelHealth';
import HistoryList from './HistoryList';
import StatusSummarySettings from './StatusSummarySettings';

export default function NotificationsPage() {
  const t = useT();
  const location = useLocation();
  const openTab = useTabStore((s) => s.openTab);
  const isSuperAdmin = useUserStore((state) => state.currentUser?.isSuperAdmin === true);

  useEffect(() => {
    openTab({
      key: 'notifications',
      label: 'notification.title',
      labelRaw: false,
      path: `${location.pathname}${location.search}`,
      closable: true,
    });
  }, [location.pathname, location.search, openTab]);

  return (
    <ListPageLayout title={t('notification.title')}>
      <Tabs
        defaultActiveKey="messages"
        items={[
          { key: 'messages', label: t('notification.tabs.messages'), children: <MessageList /> },
          { key: 'channels', label: t('notification.tabs.channels'), children: <ChannelHealth /> },
          { key: 'history', label: t('notification.tabs.history'), children: <HistoryList /> },
          ...(isSuperAdmin ? [{ key: 'status-summary', label: t('notification.tabs.statusSummary'), children: <StatusSummarySettings /> }] : []),
        ]}
      />
    </ListPageLayout>
  );
}
