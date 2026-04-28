import { useState } from 'react';
import { Tabs } from 'antd';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';
import TemplateList from './TemplateList';
import HistoryList from './HistoryList';

export default function NotificationsPage() {
  const t = useT();
  const [activeKey, setActiveKey] = useState<'templates' | 'history'>('templates');

  return (
    <ListPageLayout title={t('notification.title')}>
      <Tabs
        activeKey={activeKey}
        onChange={(k) => setActiveKey(k as 'templates' | 'history')}
        items={[
          {
            key: 'templates',
            label: t('notification.tab.template'),
            children: <TemplateList />,
          },
          {
            key: 'history',
            label: t('notification.tab.history'),
            children: <HistoryList />,
          },
        ]}
      />
    </ListPageLayout>
  );
}
