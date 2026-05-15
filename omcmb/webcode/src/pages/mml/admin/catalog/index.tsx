import { useState } from 'react';
import { Card, Tabs } from 'antd';
import { useT } from '@/hooks/useT';
import GroupsTab from './GroupsTab';
import CommandsTab from './CommandsTab';
import ParamsTab from './ParamsTab';
import XmlImportTab from './XmlImportTab';

type CatalogTab = 'groups' | 'commands' | 'params' | 'xmlImport';

export default function MMLAdminCatalog() {
  const t = useT();
  const [tab, setTab] = useState<CatalogTab>('groups');

  return (
    <Card variant="borderless" title={t('mml.admin.catalog.title')}>
      <Tabs
        activeKey={tab}
        onChange={(k) => setTab(k as CatalogTab)}
        items={[
          {
            key: 'groups',
            label: t('mml.admin.catalog.tab.groups'),
            children: <GroupsTab />,
          },
          {
            key: 'commands',
            label: t('mml.admin.catalog.tab.commands'),
            children: <CommandsTab />,
          },
          {
            key: 'params',
            label: t('mml.admin.catalog.tab.params'),
            children: <ParamsTab />,
          },
          {
            key: 'xmlImport',
            label: t('mml.admin.catalog.tab.xmlImport'),
            children: <XmlImportTab />,
          },
        ]}
      />
    </Card>
  );
}
