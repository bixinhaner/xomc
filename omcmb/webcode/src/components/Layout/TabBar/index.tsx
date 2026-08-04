import { useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { Dropdown } from 'antd';
import type { MenuProps } from 'antd';
import { DownOutlined } from '@ant-design/icons';
import {
  DndContext,
  closestCenter,
  PointerSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core';
import type { DragEndEvent } from '@dnd-kit/core';
import {
  SortableContext,
  horizontalListSortingStrategy,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { useAppStore } from '@core/store/appStore';
import { useTabStore } from '@core/store/tabStore';
import type { TabItem as TabItemType } from '@core/store/tabStore';
import { useT } from '@/hooks/useT';
import TabItem from './TabItem';
import { useTabLabelResolver } from './useTabLabel';
import styles from './TabBar.module.css';

const MAX_VISIBLE_TABS = 12;

export default function TabBar() {
  const navigate = useNavigate();
  const tabBarPosition = useAppStore((s) => s.tabBarPosition);
  const tabs = useTabStore((s) => s.tabs);
  const activeTabKey = useTabStore((s) => s.activeTabKey);
  const closeTab = useTabStore((s) => s.closeTab);
  const setActiveTab = useTabStore((s) => s.setActiveTab);
  const moveTab = useTabStore((s) => s.moveTab);
  const tabListRef = useRef<HTMLDivElement>(null);
  const t = useT();
  const tabLabel = useTabLabelResolver();

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: { distance: 5 },
    }),
  );

  const visibleTabs = tabs.slice(0, MAX_VISIBLE_TABS);
  const overflowTabs = tabs.slice(MAX_VISIBLE_TABS);

  const handleTabClick = (tab: TabItemType) => {
    setActiveTab(tab.key);
    void navigate(tab.path);
  };

  const handleClose = (key: string, e: React.MouseEvent) => {
    e.stopPropagation();
    const tab = tabs.find((tk) => tk.key === key);
    if (!tab) return;
    // If closing active tab, navigate to the tab that will become active
    if (key === activeTabKey) {
      const index = tabs.findIndex((tk) => tk.key === key);
      const newTabs = tabs.filter((tk) => tk.key !== key);
      const next = newTabs[index] ?? newTabs[index - 1];
      void navigate(next?.path ?? '/dashboard');
    }
    closeTab(key);
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;
    const fromIndex = tabs.findIndex((tk) => tk.key === active.id);
    const toIndex = tabs.findIndex((tk) => tk.key === over.id);
    if (fromIndex !== -1 && toIndex !== -1) {
      moveTab(fromIndex, toIndex);
    }
  };

  const overflowMenuItems: MenuProps['items'] = overflowTabs.map((tab) => ({
    key: tab.key,
    label: tabLabel(tab),
    onClick: () => handleTabClick(tab),
  }));

  return (
    <div className={styles.tabBar} role="tablist">
      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragEnd={handleDragEnd}
      >
        <SortableContext
          items={visibleTabs.map((tk) => tk.key)}
          strategy={tabBarPosition === 'left' ? verticalListSortingStrategy : horizontalListSortingStrategy}
        >
          <div className={styles.tabList} ref={tabListRef}>
            {visibleTabs.map((tab) => (
              <TabItem
                key={tab.key}
                tab={tab}
                label={tabLabel(tab)}
                isActive={tab.key === activeTabKey}
                onClose={handleClose}
                onClick={handleTabClick}
              />
            ))}
          </div>
        </SortableContext>
      </DndContext>

      {overflowTabs.length > 0 && (
        <Dropdown menu={{ items: overflowMenuItems }} placement="bottomRight">
          <button className={styles.moreBtn} type="button" aria-label={t('common.more')}>
            {t('common.more')} <DownOutlined style={{ fontSize: 10 }} />
          </button>
        </Dropdown>
      )}
    </div>
  );
}
