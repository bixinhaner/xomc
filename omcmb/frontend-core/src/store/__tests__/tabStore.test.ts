import { beforeEach, describe, expect, it } from 'vitest';
import { useTabStore } from '../../store/tabStore';

const kpiConfigTab = {
  key: '/system/kpi-config',
  label: '首页 KPI 配置',
  path: '/system/kpi-config',
  closable: true,
  labelRaw: true,
};

describe('tabStore direct entry', () => {
  beforeEach(() => {
    useTabStore.getState().closeAllTabs();
  });

  it('replaces the untouched dashboard placeholder on a direct entry', () => {
    useTabStore.getState().openTabForDirectEntry(kpiConfigTab);

    expect(useTabStore.getState().tabs).toEqual([kpiConfigTab]);
    expect(useTabStore.getState().activeTabKey).toBe(kpiConfigTab.key);
  });

  it('keeps existing tabs when the session already has business pages', () => {
    useTabStore.getState().openTab({
      key: 'device-list',
      label: 'nav.device.list',
      path: '/device/list',
      closable: true,
    });

    useTabStore.getState().openTabForDirectEntry(kpiConfigTab);

    expect(useTabStore.getState().tabs.map((tab) => tab.key)).toEqual([
      'dashboard',
      'device-list',
      '/system/kpi-config',
    ]);
  });

  it('restores the dashboard when the only direct-entry tab is closed', () => {
    useTabStore.getState().openTabForDirectEntry(kpiConfigTab);
    useTabStore.getState().closeTab(kpiConfigTab.key);

    expect(useTabStore.getState().tabs.map((tab) => tab.key)).toEqual(['dashboard']);
    expect(useTabStore.getState().activeTabKey).toBe('dashboard');
  });
});
