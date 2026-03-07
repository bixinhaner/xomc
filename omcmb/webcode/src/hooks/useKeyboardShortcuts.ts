import { useEffect } from 'react';
import { useTabStore } from '@/store/tabStore';

export function useKeyboardShortcuts() {
  const tabStore = useTabStore();

  useEffect(() => {
    function handleKeyDown(event: KeyboardEvent) {
      const isCtrl = event.ctrlKey || event.metaKey;

      if (!isCtrl) return;

      // Ctrl+W: Close current tab
      if (event.key === 'w' || event.key === 'W') {
        const tabs = tabStore.tabs;
        const activeKey = tabStore.activeTabKey;
        const currentTab = tabs.find((t) => t.key === activeKey);
        if (currentTab?.closable) {
          event.preventDefault();
          tabStore.closeTab(activeKey);
        }
        return;
      }

      // Ctrl+Tab: Next tab
      if (event.key === 'Tab' && !event.shiftKey) {
        const tabs = tabStore.tabs;
        if (tabs.length === 0) return;
        event.preventDefault();
        const activeKey = tabStore.activeTabKey;
        const currentIndex = tabs.findIndex((t) => t.key === activeKey);
        const nextIndex = (currentIndex + 1) % tabs.length;
        tabStore.setActiveKey(tabs[nextIndex].key);
        return;
      }

      // Ctrl+Shift+Tab: Previous tab
      if (event.key === 'Tab' && event.shiftKey) {
        const tabs = tabStore.tabs;
        if (tabs.length === 0) return;
        event.preventDefault();
        const activeKey = tabStore.activeTabKey;
        const currentIndex = tabs.findIndex((t) => t.key === activeKey);
        const prevIndex = (currentIndex - 1 + tabs.length) % tabs.length;
        tabStore.setActiveKey(tabs[prevIndex].key);
        return;
      }
    }

    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [tabStore]);
}
