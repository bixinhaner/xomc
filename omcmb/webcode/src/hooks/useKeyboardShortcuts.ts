import { useEffect } from 'react';
import { useTabStore } from '@core/store/tabStore';

export function useKeyboardShortcuts() {
  useEffect(() => {
    function handleKeyDown(event: KeyboardEvent) {
      const isCtrl = event.ctrlKey || event.metaKey;

      if (!isCtrl) return;

      // 在按键时刻读取最新 tab 状态，而非订阅整个 store 反复重渲染 / 反复重挂监听。
      const { tabs, activeTabKey, closeTab, setActiveKey } = useTabStore.getState();

      // Ctrl+W: Close current tab
      if (event.key === 'w' || event.key === 'W') {
        const currentTab = tabs.find((t) => t.key === activeTabKey);
        if (currentTab?.closable) {
          event.preventDefault();
          closeTab(activeTabKey);
        }
        return;
      }

      // Ctrl+Tab: Next tab
      if (event.key === 'Tab' && !event.shiftKey) {
        if (tabs.length === 0) return;
        event.preventDefault();
        const currentIndex = tabs.findIndex((t) => t.key === activeTabKey);
        const nextIndex = (currentIndex + 1) % tabs.length;
        setActiveKey(tabs[nextIndex].key);
        return;
      }

      // Ctrl+Shift+Tab: Previous tab
      if (event.key === 'Tab' && event.shiftKey) {
        if (tabs.length === 0) return;
        event.preventDefault();
        const currentIndex = tabs.findIndex((t) => t.key === activeTabKey);
        const prevIndex = (currentIndex - 1 + tabs.length) % tabs.length;
        setActiveKey(tabs[prevIndex].key);
        return;
      }
    }

    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, []);
}
