import { useState } from 'react';
import { Tabs } from 'antd';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';
import AbnormalRebootTab from './AbnormalRebootTab';
import EventLogTab from './EventLogTab';

// 「重启记录」页面：tab 容器
//   - Tab 1 「事件日志」      → event_logs (普通 1 BOOT)
//   - Tab 2 「异常重启记录」  → station_fault_logs (1 BOOT + HaltReason)
//
// 两表语义分开，UI 在同一入口下展示，因为业务上都属于设备重启相关活动审计。

export default function AbnormalReboot() {
  const t = useT();
  const [activeKey, setActiveKey] = useState<'event' | 'abnormal'>('event');

  return (
    <ListPageLayout title={t('page.rebootRecords.title')}>
      <Tabs
        activeKey={activeKey}
        onChange={(k) => setActiveKey(k as 'event' | 'abnormal')}
        destroyInactiveTabPane
        items={[
          {
            key: 'event',
            label: t('page.rebootRecords.tab.boot'),
            children: <EventLogTab />,
          },
          {
            key: 'abnormal',
            label: t('page.rebootRecords.tab.abnormal'),
            children: <AbnormalRebootTab />,
          },
        ]}
      />
    </ListPageLayout>
  );
}
