import { useMemo } from 'react';
import { Dropdown } from 'antd';
import type { MenuProps } from 'antd';
import type { TabItem } from '@/store/tabStore';
import { useTabStore } from '@/store/tabStore';
import { useT } from '@/hooks/useT';

interface Props {
  tab: TabItem;
  children: React.ReactNode;
}

export default function TabContextMenu({ tab, children }: Props) {
  const closeTab = useTabStore((s) => s.closeTab);
  const closeOtherTabs = useTabStore((s) => s.closeOtherTabs);
  const closeAllTabs = useTabStore((s) => s.closeAllTabs);
  const closeTabsToRight = useTabStore((s) => s.closeTabsToRight);
  const tabs = useTabStore((s) => s.tabs);
  const t = useT();

  const tabIndex = tabs.findIndex((tk) => tk.key === tab.key);
  const hasTabsToRight = tabs.slice(tabIndex + 1).some((tk) => tk.closable !== false);
  const hasOtherClosable = tabs.some((tk) => tk.key !== tab.key && tk.closable !== false);

  const menuItems: MenuProps['items'] = useMemo(() => [
    {
      key: 'close',
      label: t('common.close'),
      disabled: !tab.closable,
      onClick: () => closeTab(tab.key),
    },
    {
      key: 'closeOthers',
      label: t('tab.closeOthers'),
      disabled: !hasOtherClosable,
      onClick: () => closeOtherTabs(tab.key),
    },
    {
      key: 'closeAll',
      label: t('tab.closeAll'),
      onClick: () => closeAllTabs(),
    },
    {
      key: 'closeRight',
      label: t('tab.closeRight'),
      disabled: !hasTabsToRight,
      onClick: () => closeTabsToRight(tab.key),
    },
  ], [t, tab.closable, tab.key, hasOtherClosable, hasTabsToRight, closeTab, closeOtherTabs, closeAllTabs, closeTabsToRight]);

  return (
    <Dropdown menu={{ items: menuItems }} trigger={['contextMenu']}>
      <div style={{ height: '100%', display: 'flex', alignItems: 'center' }}>
        {children}
      </div>
    </Dropdown>
  );
}
