import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import { useTabStore } from '@core/store/tabStore';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';
import MessageList from './MessageList';

export default function NotificationsPage() {
  const t = useT();
  const location = useLocation();
  const openTab = useTabStore((s) => s.openTab);

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
      <MessageList />
    </ListPageLayout>
  );
}
