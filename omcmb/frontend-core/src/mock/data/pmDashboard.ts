/**
 * Mock data for T-0164-P6 / G6 PM dashboards / panels / 用户偏好。
 *
 * VITE_USE_MOCK=true 时由 pmDashboardApi.ts 的 mockService 消费。
 */

import type {
  Dashboard,
  Panel,
  UserDashboardPreferences,
} from '../../types/pmDashboard';

const now = new Date().toISOString();

export const mockOwnerId = 'mock-user-owner-0001';
export const mockOtherUserId = 'mock-user-other-0002';

export const mockDashboards: Dashboard[] = [
  {
    id: 'mock-dash-001',
    name: '基础 LTE 性能概览',
    description: '默认初始仪表盘 — RRC / E-RAB / 切换 KPI 三大件',
    ownerId: mockOwnerId,
    sharedWith: [mockOtherUserId],
    technology: 'lte',
    layout: {
      panels: [
        { i: 'mock-panel-001', x: 0, y: 0, w: 6, h: 4 },
        { i: 'mock-panel-002', x: 6, y: 0, w: 6, h: 4 },
        { i: 'mock-panel-003', x: 0, y: 4, w: 12, h: 4 },
      ],
    },
    createdAt: now,
    updatedAt: now,
  },
  {
    id: 'mock-dash-002',
    name: '5G NR 接通率监控',
    ownerId: mockOwnerId,
    sharedWith: [],
    technology: 'nr',
    layout: { panels: [] },
    createdAt: now,
    updatedAt: now,
  },
];

export const mockPanels: Panel[] = [
  {
    id: 'mock-panel-001',
    dashboardId: 'mock-dash-001',
    panelType: 'kpi_card',
    title: 'RRC 建立成功率',
    metricPaths: ['L.RRC.SuccRate'],
    granularity: 'hourly',
    dimension: 'device',
    deviceSns: ['BLQ-001'],
    timeRange: { start_offset: '-24h' },
    config: { unit: '%', threshold: 95 },
  },
  {
    id: 'mock-panel-002',
    dashboardId: 'mock-dash-001',
    panelType: 'line_chart',
    title: 'E-RAB 建立成功率（24h）',
    metricPaths: ['L.ERAB.SuccRate'],
    granularity: 'hourly',
    dimension: 'device',
    deviceSns: ['BLQ-001'],
    timeRange: { start_offset: '-24h' },
    compareMode: 'previous_window',
    config: { yAxisMin: 90, yAxisMax: 100 },
  },
  {
    id: 'mock-panel-003',
    dashboardId: 'mock-dash-001',
    panelType: 'table',
    title: '设备组吞吐量排行',
    metricPaths: ['L.DL.Throughput', 'L.UL.Throughput'],
    granularity: 'daily',
    dimension: 'device_group',
    deviceGroupIds: ['mock-group-001'],
    timeRange: { start_offset: '-7d' },
    config: { pageSize: 10 },
  },
];

export const mockUserPreferences: UserDashboardPreferences = {
  userId: mockOwnerId,
  kpiCardLayout: {
    order: ['rrcSuccRate', 'erabSuccRate', 'hoSuccRate', 'dlThroughput'],
    hidden: ['voLteSuccRate'],
  },
};
