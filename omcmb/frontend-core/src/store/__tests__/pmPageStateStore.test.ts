import { describe, it, expect, beforeEach } from 'vitest'
import {
  isPerformanceTabPath,
  performancePageKeyFromPath,
  usePmPageStateStore,
} from '../../store/pmPageStateStore'
import { useTabStore } from '../../store/tabStore'
import { useUserStore } from '../../store/userStore'
import type { User } from '../../types/system'

const mockUser: User = {
  id: 'user-001',
  username: 'testuser',
  displayName: 'Test User',
  email: 'test@example.com',
  phone: '13800138000',
  role: 'operator',
  status: 'active',
  lastLoginTime: '2026-01-01T00:00:00Z',
  createTime: '2025-01-01T00:00:00Z',
}

const otherUser: User = {
  ...mockUser,
  id: 'user-002',
  username: 'otheruser',
}

function resetStores() {
  usePmPageStateStore.setState({ pages: {} })
  useTabStore.getState().closeAllTabs()
  useUserStore.setState({
    currentUser: null,
    accessToken: null,
    refreshToken: null,
    tokenExpiresAt: null,
    isAuthenticated: false,
    permissions: [],
    loading: false,
    mustChangePassword: false,
  })
  sessionStorage.clear()
  localStorage.clear()
}

describe('pmPageStateStore', () => {
  beforeEach(() => {
    resetStores()
  })

  it('saves and restores a JSON-serializable performance page state', () => {
    usePmPageStateStore.getState().savePageState('/performance/query?template=daily', {
      filters: {
        granularity: 'hourly',
        metricPaths: ['K001', 'C001'],
        includeWeekends: false,
      },
      view: {
        activePanel: 'chart',
        selectedTemplateId: 'template-001',
      },
      lastAction: {
        submittedQuery: true,
        renderedChart: true,
      },
    })

    const snapshot = usePmPageStateStore.getState().getPageState('/performance/query')

    expect(snapshot).toMatchObject({
      filters: {
        granularity: 'hourly',
        metricPaths: ['K001', 'C001'],
        includeWeekends: false,
      },
      view: {
        activePanel: 'chart',
        selectedTemplateId: 'template-001',
      },
      lastAction: {
        submittedQuery: true,
        renderedChart: true,
        refreshed: false,
      },
    })
    expect(snapshot?.savedAt).toEqual(expect.any(String))
    expect(JSON.parse(JSON.stringify(snapshot))).toEqual(snapshot)
  })

  it('keeps each performance page state isolated', () => {
    usePmPageStateStore.getState().savePageState('/performance/query', {
      filters: { granularity: 'daily' },
      lastAction: { submittedQuery: true },
    })
    usePmPageStateStore.getState().savePageState('/performance/device-view', {
      filters: { granularity: 'hourly' },
      lastAction: { refreshed: true },
    })

    expect(usePmPageStateStore.getState().getPageState('/performance/query')?.filters).toEqual({
      granularity: 'daily',
    })
    expect(usePmPageStateStore.getState().getPageState('/performance/device-view')?.filters).toEqual({
      granularity: 'hourly',
    })
  })

  it('clears a page state when the page resets', () => {
    usePmPageStateStore.getState().savePageState('/performance/query', {
      filters: { granularity: 'daily' },
      lastAction: { submittedQuery: true },
    })

    usePmPageStateStore.getState().clearPageState('/performance/query')

    expect(usePmPageStateStore.getState().getPageState('/performance/query')).toBeNull()
  })

  it('rejects non-serializable values and strips forbidden result fields', () => {
    expect(() =>
      usePmPageStateStore.getState().savePageState('/performance/query', {
        filters: { from: Number.NaN },
      }),
    ).toThrow('JSON serializable')

    usePmPageStateStore.getState().savePageState('/performance/query', {
      view: {
        rows: [{ id: 'row-001' }],
        chartSeries: [{ name: 'K001', data: [1, 2] }],
        results: [{ timestamp: '2026-07-28T00:00:00Z', value: 1 }],
        exportTask: { id: 'export-001' },
        response: { data: [{ value: 1 }] },
        loading: true,
        error: { message: 'failed' },
        activePanel: 'table',
      },
    })

    expect(usePmPageStateStore.getState().getPageState('/performance/query')?.view).toEqual({
      activePanel: 'table',
    })
  })

  it('recognizes performance tab paths without treating unrelated pages as performance state', () => {
    expect(isPerformanceTabPath('/performance')).toBe(true)
    expect(isPerformanceTabPath('/performance/device-view?tab=a')).toBe(true)
    expect(isPerformanceTabPath('/performance/pm-adhoc')).toBe(true)
    expect(isPerformanceTabPath('/performance/pm-adhoc/new')).toBe(true)
    expect(isPerformanceTabPath('/performance/pm-adhoc/adhoc-001/edit')).toBe(true)
    expect(isPerformanceTabPath('/performance/query')).toBe(true)
    expect(isPerformanceTabPath('/performance/task-config')).toBe(false)
    expect(isPerformanceTabPath('/device/performance-profile')).toBe(false)
    expect(performancePageKeyFromPath('/performance/query?template=daily')).toBe('/performance/query')
    expect(performancePageKeyFromPath('/performance/pm-adhoc/new')).toBe('/performance/pm-adhoc')
    expect(performancePageKeyFromPath('/performance/pm-adhoc/adhoc-001/edit')).toBe('/performance/pm-adhoc')
  })

  it('rejects state saves for unrelated performance management pages', () => {
    expect(() =>
      usePmPageStateStore.getState().savePageState('/performance/task-config', {
        filters: { granularity: 'daily' },
      }),
    ).toThrow('unsupported performance page state key')
  })

  it('clears the closed performance tab state only', () => {
    useTabStore.getState().openTab({
      key: 'pm-query',
      label: 'nav.performance.query',
      path: '/performance/query',
      closable: true,
    })
    useTabStore.getState().openTab({
      key: 'pm-device-view',
      label: 'nav.performance.deviceView',
      path: '/performance/device-view',
      closable: true,
    })
    usePmPageStateStore.getState().savePageState('/performance/query', {
      filters: { granularity: 'daily' },
      lastAction: { submittedQuery: true },
    })
    usePmPageStateStore.getState().savePageState('/performance/device-view', {
      filters: { granularity: 'hourly' },
      lastAction: { renderedChart: true },
    })

    useTabStore.getState().closeTab('pm-query')

    expect(usePmPageStateStore.getState().getPageState('/performance/query')).toBeNull()
    expect(usePmPageStateStore.getState().getPageState('/performance/device-view')).not.toBeNull()
  })

  it('clears performance page states covered by closeOtherTabs, closeTabsToRight, and closeAllTabs', () => {
    useTabStore.getState().openTab({
      key: 'pm-query',
      label: 'nav.performance.query',
      path: '/performance/query',
      closable: true,
    })
    useTabStore.getState().openTab({
      key: 'device-list',
      label: 'nav.device.list',
      path: '/device/list',
      closable: true,
    })
    useTabStore.getState().openTab({
      key: 'pm-device-view',
      label: 'nav.performance.deviceView',
      path: '/performance/device-view',
      closable: true,
    })
    usePmPageStateStore.getState().savePageState('/performance/query', {
      filters: { granularity: 'daily' },
    })
    usePmPageStateStore.getState().savePageState('/performance/device-view', {
      filters: { granularity: 'hourly' },
    })

    useTabStore.getState().closeOtherTabs('pm-query')

    expect(usePmPageStateStore.getState().getPageState('/performance/query')).not.toBeNull()
    expect(usePmPageStateStore.getState().getPageState('/performance/device-view')).toBeNull()

    useTabStore.getState().openTab({
      key: 'pm-device-view',
      label: 'nav.performance.deviceView',
      path: '/performance/device-view',
      closable: true,
    })
    usePmPageStateStore.getState().savePageState('/performance/device-view', {
      filters: { granularity: 'hourly' },
    })

    useTabStore.getState().closeTabsToRight('pm-query')

    expect(usePmPageStateStore.getState().getPageState('/performance/query')).not.toBeNull()
    expect(usePmPageStateStore.getState().getPageState('/performance/device-view')).toBeNull()

    useTabStore.getState().closeAllTabs()

    expect(usePmPageStateStore.getState().getPageState('/performance/query')).toBeNull()
  })

  it('clears performance page state when a tab is removed by the max tab limit', () => {
    useTabStore.getState().openTab({
      key: 'pm-query',
      label: 'nav.performance.query',
      path: '/performance/query',
      closable: true,
    })
    usePmPageStateStore.getState().savePageState('/performance/query', {
      filters: { granularity: 'daily' },
    })

    for (let index = 1; index <= 9; index += 1) {
      useTabStore.getState().openTab({
        key: `device-${index}`,
        label: 'nav.device.list',
        path: `/device/list-${index}`,
        closable: true,
      })
    }

    useTabStore.getState().openTab({
      key: 'alarm-current',
      label: 'nav.alarm.current',
      path: '/alarm/current',
      closable: true,
    })

    expect(useTabStore.getState().tabs.some((tab) => tab.key === 'pm-query')).toBe(false)
    expect(usePmPageStateStore.getState().getPageState('/performance/query')).toBeNull()
  })

  it('clears performance page states on logout and user switch', () => {
    useUserStore.getState().login(mockUser)
    usePmPageStateStore.getState().savePageState('/performance/query', {
      filters: { granularity: 'daily' },
    })

    useUserStore.getState().logout()

    expect(usePmPageStateStore.getState().pages).toEqual({})

    useUserStore.getState().login(mockUser)
    usePmPageStateStore.getState().savePageState('/performance/query', {
      filters: { granularity: 'daily' },
    })
    useUserStore.getState().clearAuth()

    expect(usePmPageStateStore.getState().pages).toEqual({})

    useUserStore.getState().login(mockUser)
    usePmPageStateStore.getState().savePageState('/performance/query', {
      filters: { granularity: 'daily' },
    })
    useUserStore.getState().login(otherUser)

    expect(usePmPageStateStore.getState().pages).toEqual({})

    useUserStore.getState().login(mockUser)
    usePmPageStateStore.getState().savePageState('/performance/query', {
      filters: { granularity: 'daily' },
    })
    useUserStore.getState().setUser(otherUser)

    expect(usePmPageStateStore.getState().pages).toEqual({})

    useUserStore.getState().login(mockUser)
    usePmPageStateStore.getState().savePageState('/performance/query', {
      filters: { granularity: 'daily' },
    })
    useUserStore.getState().lock()
    useUserStore.getState().login(otherUser)

    expect(usePmPageStateStore.getState().pages).toEqual({})
  })
})
